package cql

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

// countingProvider returns a fixed number of identical resources.
type countingProvider struct{ n int }

func (p *countingProvider) Retrieve(_ context.Context, _, _, _ string, _, _ interface{}) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, p.n)
	for i := range out {
		out[i] = json.RawMessage(`{"resourceType":"Condition","id":"c"}`)
	}
	return out, nil
}

func chainOf(op string, n int) string {
	return "library T version '1.0'\n\ndefine X: 1" + strings.Repeat(" "+op+" 1", n) + "\n"
}

// TestMaxDepth covers WithMaxDepth, which was assigned on the Engine and never
// read, so a self-referencing definition recursed until the process died of an
// unrecoverable stack overflow.
func TestMaxDepth(t *testing.T) {
	deep := chainOf("+", 400)

	if _, err := NewEngine(WithMaxDepth(20)).EvaluateExpression(context.Background(), deep, "X", nil, nil); err == nil {
		t.Error("expected a deep expression to be rejected at maxDepth=20")
	} else {
		var costly *ErrTooCostly
		if !errors.As(err, &costly) {
			t.Errorf("expected ErrTooCostly, got %T: %v", err, err)
		}
	}

	// Zero disables the limit.
	if _, err := NewEngine(WithMaxDepth(0)).EvaluateExpression(context.Background(), deep, "X", nil, nil); err != nil {
		t.Errorf("maxDepth=0 should not bound anything, got: %v", err)
	}

	// The default has to clear ordinary clinical CQL. A chain of 200 terms and a
	// chain of 100 definitions are both well within what a measure library
	// writes, and both were rejected when the default sat at 100.
	if _, err := NewEngine().EvaluateExpression(context.Background(), chainOf("+", 200), "X", nil, nil); err != nil {
		t.Errorf("a 200-term chain should evaluate under the default depth, got: %v", err)
	}
	var b strings.Builder
	b.WriteString("library T version '1.0'\n\ndefine D0: 1\n")
	for i := 1; i <= 100; i++ {
		b.WriteString("define D" + strconv.Itoa(i) + ": D" + strconv.Itoa(i-1) + " + 1\n")
	}
	if _, err := NewEngine().EvaluateExpression(context.Background(), b.String(), "D100", nil, nil); err != nil {
		t.Errorf("a chain of 100 definitions should evaluate under the default depth, got: %v", err)
	}
}

// TestMaxDepthStopsRunawayRecursion is the reason the limit exists: each of
// these recursed without bound while WithMaxDepth was unwired, and a stack
// overflow in Go is a fatal runtime error, not a panic a server can recover.
func TestMaxDepthStopsRunawayRecursion(t *testing.T) {
	for _, src := range []string{
		"library T version '1.0'\n\ndefine A: A\n",
		"library T version '1.0'\n\ndefine A: A + 1\n",
		"library T version '1.0'\n\ndefine A: B\ndefine B: A\n",
		"library T version '1.0'\n\ndefine function F(x Integer): F(x)\ndefine A: F(1)\n",
	} {
		if _, err := NewEngine().EvaluateExpression(context.Background(), src, "A", nil, nil); err == nil {
			t.Errorf("unbounded recursion should be refused: %s", src)
		}
	}
}

// TestMaxRetrieveSize covers WithMaxRetrieveSize, also never read before.
func TestMaxRetrieveSize(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\n\ndefine X: Count([Condition])\n"

	_, err := NewEngine(WithDataProvider(&countingProvider{n: 50}), WithMaxRetrieveSize(10)).
		EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err == nil {
		t.Fatal("a retrieve over the limit should fail rather than be truncated")
	}
	var costly *ErrTooCostly
	if !errors.As(err, &costly) {
		t.Errorf("expected ErrTooCostly, got %T: %v", err, err)
	}

	got, err := NewEngine(WithDataProvider(&countingProvider{n: 50}), WithMaxRetrieveSize(100)).
		EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err != nil {
		t.Fatalf("a retrieve under the limit should succeed: %v", err)
	}
	if valueString(got) != "50" {
		t.Errorf("Count = %s, want 50", valueString(got))
	}
}

// TestTimeoutInterrupts covers cancellation, which nothing in eval used to
// check: a timeout was only noticed once the evaluation had already finished.
func TestTimeoutInterrupts(t *testing.T) {
	// A cartesian product large enough to take far longer than the timeout, and
	// which the query builds before evaluating anything.
	src := `library T version '1.0'

define Src: ({1,2,3,4,5,6,7,8,9,10})
define X: Count(from Src A, Src B, Src C, Src D, Src E, Src F, Src G return A + B)
`
	const timeout = 300 * time.Millisecond
	start := time.Now()
	_, err := NewEngine(WithTimeout(timeout)).EvaluateExpression(context.Background(), src, "X", nil, nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected the evaluation to be cut short")
	}
	var timedOut *ErrTimeout
	if !errors.As(err, &timedOut) {
		t.Errorf("expected ErrTimeout, got %T: %v", err, err)
	}
	// The check is periodic, so allow slack — but not the seconds this takes
	// when it runs to completion.
	if elapsed > 3*timeout {
		t.Errorf("timeout took %v to take effect, limit was %v", elapsed, timeout)
	}
}

// TestUnknownIdentifierIsError covers the fallback that answered an unresolved
// identifier with its own name as a String, turning a typo into a plausible
// value rather than a diagnostic.
func TestUnknownIdentifierIsError(t *testing.T) {
	for _, src := range []string{
		"library T version '1.0'\n\ndefine A: Bogus\n",
		"library T version '1.0'\n\ndefine A: Bogus + 1\n",
		"library T version '1.0'\n\ndefine A: ({1,2}) B where B > Bogus\n",
	} {
		_, err := NewEngine().EvaluateExpression(context.Background(), src, "A", nil, nil)
		if err == nil {
			t.Errorf("an unresolved identifier should be an error: %s", src)
			continue
		}
		if !strings.Contains(err.Error(), "Bogus") {
			t.Errorf("the error should name the identifier, got: %v", err)
		}
	}
}

// TestParameterDefaults covers `parameter X ... default <expr>`, which was
// parsed and never evaluated: referencing the parameter used to yield its own
// name as a String.
func TestParameterDefaults(t *testing.T) {
	src := `library T version '1.0'

parameter MP Interval<DateTime> default Interval[@2020-01-01, @2020-12-31]
parameter N Integer default 42
parameter Optional Integer

define UsesMP: MP
define UsesN: N
define UsesOptional: Optional
`
	tests := []struct{ name, want string }{
		{"UsesMP", "Interval[2020-01-01, 2020-12-31]"},
		{"UsesN", "42"},
		// Declared with neither a default nor a supplied value: null, not an error.
		{"UsesOptional", "null"},
	}
	engine := NewEngine()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.EvaluateExpression(context.Background(), src, tt.name, nil, nil)
			if err != nil {
				t.Fatalf("evaluating %s: %v", tt.name, err)
			}
			if s := valueString(got); s != tt.want {
				t.Errorf("%s = %s, want %s", tt.name, s, tt.want)
			}
		})
	}

	// A supplied value still wins over the declared default.
	supplied, err := engine.EvaluateExpression(context.Background(), src, "UsesN", nil,
		map[string]fptypes.Value{"N": fptypes.NewInteger(7)})
	if err != nil {
		t.Fatalf("evaluating UsesN with a supplied value: %v", err)
	}
	if s := valueString(supplied); s != "7" {
		t.Errorf("a supplied parameter should override its default: got %s, want 7", s)
	}
}

// capturingProvider records what the retrieve asked for.
type capturingProvider struct {
	codes    interface{}
	codePath string
}

func (p *capturingProvider) Retrieve(_ context.Context, _, codePath, _ string, codes, _ interface{}) ([]json.RawMessage, error) {
	p.codes = codes
	p.codePath = codePath
	return nil, nil
}

const terminologyLib = `library T version '1.0'
using FHIR version '4.0.1'
valueset "Diabetes": 'http://example.org/vs/dm'
codesystem "LOINC": 'http://loinc.org'
code "SBP": '8480-6' from "LOINC" display 'Systolic'
concept "BP": { "SBP" } display 'Blood pressure'

define ByValueSet: Count([Condition: "Diabetes"])
define TheCode: "SBP"
define TheConcept: "BP"
`

// TestValueSetNameReachesProvider covers a quoted terminology reference in a
// retrieve. The name was taken from the whole node's text, so it kept its
// quotes, missed the value set table keyed without them, and was handed to the
// data provider as the literal string `"Diabetes"` instead of the URL.
func TestValueSetNameReachesProvider(t *testing.T) {
	p := &capturingProvider{}
	_, err := NewEngine(WithDataProvider(p)).
		EvaluateExpression(context.Background(), terminologyLib, "ByValueSet", nil, nil)
	if err != nil {
		t.Fatalf("evaluating: %v", err)
	}
	if got, want := p.codes, "http://example.org/vs/dm"; got != want {
		t.Errorf("provider received codes %#v, want %q", got, want)
	}
}

// TestDeclaredCodesResolve covers `code` and `concept` declarations, which were
// never loaded into the evaluation context: referencing one yielded its own
// name as a String, so `O.code ~ "SBP"` compared a Code against the text "SBP"
// and was quietly always false.
func TestDeclaredCodesResolve(t *testing.T) {
	engine := NewEngine()

	code, err := engine.EvaluateExpression(context.Background(), terminologyLib, "TheCode", nil, nil)
	if err != nil {
		t.Fatalf("evaluating TheCode: %v", err)
	}
	if code.Type() != "Code" {
		t.Errorf("TheCode is a %s, want Code", code.Type())
	}
	// The system is the code system's URL, not the name it was declared under.
	if s := valueString(code); !strings.Contains(s, "http://loinc.org") || !strings.Contains(s, "8480-6") {
		t.Errorf("TheCode = %s, want the LOINC URL and the code", s)
	}

	concept, err := engine.EvaluateExpression(context.Background(), terminologyLib, "TheConcept", nil, nil)
	if err != nil {
		t.Fatalf("evaluating TheConcept: %v", err)
	}
	if concept.Type() != "Concept" {
		t.Errorf("TheConcept is a %s, want Concept", concept.Type())
	}
	if s := valueString(concept); !strings.Contains(s, "8480-6") {
		t.Errorf("TheConcept = %s, want it to carry its declared code", s)
	}
}
