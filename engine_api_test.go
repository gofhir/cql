package cql

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestParseIsReusable covers separating parsing from evaluation. Parsing
// dominates the first use of a source — a 300-term expression costs about 1.4ms
// to parse and 26µs to evaluate — and a caller that builds an engine per
// request paid it every time, since the cache lives on the engine.
func TestParseIsReusable(t *testing.T) {
	src := `library Reusable version '2.1'

define private Helper: 21
define Answer: Helper * 2
define Other: 'x'
`
	engine := NewEngine()
	lib, err := engine.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if lib.Name() != "Reusable" || lib.Version() != "2.1" {
		t.Errorf("library is %q/%q, want Reusable/2.1", lib.Name(), lib.Version())
	}
	// Private definitions are the library's own business and are not offered.
	names := strings.Join(lib.ExpressionNames(), ",")
	if names != "Answer,Other" {
		t.Errorf("ExpressionNames = %q, want Answer,Other", names)
	}

	// Evaluating twice from the same parse gives the same answer.
	for range 2 {
		got, evalErr := engine.EvaluateParsedExpression(context.Background(), lib, "Answer", nil, nil)
		if evalErr != nil {
			t.Fatalf("EvaluateParsedExpression: %v", evalErr)
		}
		if valueString(got) != "42" {
			t.Errorf("Answer = %s, want 42", valueString(got))
		}
	}

	results, err := engine.EvaluateParsedLibrary(context.Background(), lib, nil, nil)
	if err != nil {
		t.Fatalf("EvaluateParsedLibrary: %v", err)
	}
	if _, listed := results["Helper"]; listed {
		t.Error("a private definition was returned to the caller")
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}

// TestPrivateDefinitionsAreNotOffered covers `define private` in the library
// being evaluated. It was parsed and never checked, so a caller could ask for
// one by name and receive it.
func TestPrivateDefinitionsAreNotOffered(t *testing.T) {
	src := "library T version '1.0'\n\ndefine private Hidden: 42\ndefine Pub: Hidden + 1\n"
	engine := NewEngine()

	// The public definition is built on it, so it still evaluates.
	got, err := engine.EvaluateExpression(context.Background(), src, "Pub", nil, nil)
	if err != nil {
		t.Fatalf("Pub: %v", err)
	}
	if valueString(got) != "43" {
		t.Errorf("Pub = %s, want 43", valueString(got))
	}

	if _, err := engine.EvaluateExpression(context.Background(), src, "Hidden", nil, nil); err == nil {
		t.Error("asking for a private definition by name should be refused")
	}
}

// TestCompileCacheVerifiesItsSource covers the cache that indexed compiled
// libraries by a 64-bit hash of the source and nothing else. A collision handed
// back a different library's AST — silently, with no way for the caller to tell.
func TestCompileCacheVerifiesItsSource(t *testing.T) {
	engine := NewEngine()
	first := "library A version '1.0'\n\ndefine X: 1\n"
	second := "library B version '1.0'\n\ndefine X: 2\n"

	// Distinct sources must keep their own ASTs however the cache is keyed.
	for range 3 {
		a, err := engine.EvaluateExpression(context.Background(), first, "X", nil, nil)
		if err != nil {
			t.Fatalf("first: %v", err)
		}
		b, err := engine.EvaluateExpression(context.Background(), second, "X", nil, nil)
		if err != nil {
			t.Fatalf("second: %v", err)
		}
		if valueString(a) != "1" || valueString(b) != "2" {
			t.Fatalf("got %s and %s, want 1 and 2", valueString(a), valueString(b))
		}
	}
}

// TestEvaluationTimestampIsInjectable covers supplying the instant Now, Today
// and TimeOfDay answer with. Freezing it made them consistent within one
// evaluation; supplying it is what makes a measure re-runnable — the same data
// and the same timestamp give the same answer months later.
func TestEvaluationTimestampIsInjectable(t *testing.T) {
	src := "library T version '1.0'\n\ndefine N: Now()\ndefine D: Today()\ndefine T: TimeOfDay()\n"
	fixed := time.Date(2021, 7, 4, 15, 30, 0, 0, time.UTC)

	engine := NewEngine(WithEvaluationTimestamp(fixed))
	for _, tt := range []struct{ name, want string }{
		{"N", "2021-07-04T15:30:00.000Z"},
		{"D", "2021-07-04"},
		{"T", "15:30:00.000"},
	} {
		got, err := engine.EvaluateExpression(context.Background(), src, tt.name, nil, nil)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if s := valueString(got); s != tt.want {
			t.Errorf("%s = %s, want %s", tt.name, s, tt.want)
		}
	}

	// One call may re-run a measure as of another date without a second engine.
	other := time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC)
	got, err := engine.EvaluateExpression(context.Background(), src, "D", nil, nil,
		WithCallEvaluationTimestamp(other))
	if err != nil {
		t.Fatalf("per-call: %v", err)
	}
	if s := valueString(got); s != "1999-12-31" {
		t.Errorf("per-call D = %s, want 1999-12-31", s)
	}

	// Unset still means the clock, so an engine that says nothing behaves as before.
	got, err = NewEngine().EvaluateExpression(context.Background(), src, "D", nil, nil)
	if err != nil {
		t.Fatalf("unset: %v", err)
	}
	if valueString(got) == "2021-07-04" {
		t.Error("an engine with no timestamp set should read the clock")
	}
}
