package cql

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/gofhir/cql/eval"
	"github.com/gofhir/cql/model"
)

// pathRecorder captures the code path a retrieve asks the provider for.
type pathRecorder struct{ codePath string }

func (p *pathRecorder) Retrieve(_ context.Context, req eval.RetrieveRequest) ([]json.RawMessage, error) {
	p.codePath = req.CodePath
	switch req.ResourceType {
	case "Observation":
		return []json.RawMessage{json.RawMessage(`{"resourceType":"Observation","id":"o1","valueQuantity":{"value":5,"unit":"mg"},"effectiveDateTime":"2020-06-01"}`)}, nil
	case "Condition":
		return []json.RawMessage{json.RawMessage(`{"resourceType":"Condition","id":"c1","onsetDateTime":"2019-01-01"}`)}, nil
	case "Patient":
		return []json.RawMessage{json.RawMessage(`{"resourceType":"Patient","id":"p1","birthDate":"1980-01-01","deceasedBoolean":false}`)}, nil
	}
	return nil, nil
}

// TestPrimaryCodePathReachesProvider covers the element a retrieve filters its
// codes against. The model knew it for 147 types and the evaluator asked for
// none of them, so every retrieve reached the provider with an empty path and
// left it to guess.
func TestPrimaryCodePathReachesProvider(t *testing.T) {
	for _, tt := range []struct{ resource, want string }{
		{"Condition", "code"},
		{"Encounter", "type"},
		{"CarePlan", "category"},
		{"Immunization", "vaccineCode"},
	} {
		src := "library T version '1.0'\nusing FHIR version '4.0.1'\nvalueset \"VS\": 'http://x'\n\ndefine X: [" +
			tt.resource + ": \"VS\"]\n"
		p := &pathRecorder{}
		if _, err := NewEngine(WithDataProvider(p)).
			EvaluateExpression(context.Background(), src, "X", nil, nil); err != nil {
			t.Fatalf("[%s: \"VS\"]: %v", tt.resource, err)
		}
		if p.codePath != tt.want {
			t.Errorf("[%s: \"VS\"] asked for codePath %q, want %q", tt.resource, p.codePath, tt.want)
		}
	}

	// An unfiltered retrieve has no codes, so it names no path.
	p := &pathRecorder{}
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\n\ndefine X: [Condition]\n"
	if _, err := NewEngine(WithDataProvider(p)).
		EvaluateExpression(context.Background(), src, "X", nil, nil); err != nil {
		t.Fatalf("unfiltered retrieve: %v", err)
	}
	if p.codePath != "" {
		t.Errorf("an unfiltered retrieve asked for codePath %q, want empty", p.codePath)
	}
}

// TestTypeHierarchy covers `is` and `as` against the declared base chain.
// Comparing type names alone made `x is DomainResource` false for every
// resource.
func TestTypeHierarchy(t *testing.T) {
	src := `library T version '1.0'
using FHIR version '4.0.1'

define C: First([Condition])
define IsCondition: C is Condition
define IsDomainResource: C is DomainResource
define IsResource: C is Resource
define IsPatient: C is Patient
define AsDomainResource: (C as DomainResource) is null
`
	engine := NewEngine(WithDataProvider(&pathRecorder{}))
	for _, tt := range []struct{ name, want string }{
		{"IsCondition", "true"},
		{"IsDomainResource", "true"},
		{"IsResource", "true"},
		{"IsPatient", "false"},
	} {
		got, err := engine.EvaluateExpression(context.Background(), src, tt.name, nil, nil)
		if err != nil {
			t.Fatalf("evaluating %s: %v", tt.name, err)
		}
		if s := valueString(got); s != tt.want {
			t.Errorf("%s = %s, want %s", tt.name, s, tt.want)
		}
	}
}

// TestChoiceElementsResolve covers value[x], onset[x] and deceased[x]. The
// official model spells the FHIR primitives in lower case — FHIR.dateTime —
// while the JSON field capitalizes them, so the concrete name has to be built
// rather than concatenated.
func TestChoiceElementsResolve(t *testing.T) {
	src := `library T version '1.0'
using FHIR version '4.0.1'

define ObsValue: First([Observation]).value
define ObsEffective: First([Observation]).effective
define CondOnset: First([Condition]).onset
define PatDeceased: First([Patient]).deceased
`
	engine := NewEngine(WithDataProvider(&pathRecorder{}))
	for _, tt := range []struct{ name, want string }{
		{"ObsEffective", "2020-06-01"},
		{"CondOnset", "2019-01-01"},
		{"PatDeceased", "false"},
	} {
		got, err := engine.EvaluateExpression(context.Background(), src, tt.name, nil, nil)
		if err != nil {
			t.Fatalf("evaluating %s: %v", tt.name, err)
		}
		if s := valueString(got); s != tt.want {
			t.Errorf("%s = %s, want %s", tt.name, s, tt.want)
		}
	}
	// The one the previous model happened to get right must keep working.
	got, err := engine.EvaluateExpression(context.Background(), src, "ObsValue", nil, nil)
	if err != nil {
		t.Fatalf("evaluating ObsValue: %v", err)
	}
	if got == nil {
		t.Error("ObsValue = null, want the valueQuantity")
	}
}

// requestRecorder captures the whole retrieve request.
type requestRecorder struct{ last eval.RetrieveRequest }

func (r *requestRecorder) Retrieve(_ context.Context, req eval.RetrieveRequest) ([]json.RawMessage, error) {
	r.last = req
	return []json.RawMessage{json.RawMessage(
		`{"resourceType":"Observation","id":"o1","effectivePeriod":{"start":"2020-01-01","end":"2020-12-31"},` +
			`"valueQuantity":{"value":5,"unit":"mg","system":"http://unitsofmeasure.org","code":"mg"},` +
			`"referenceRange":[{"low":{"value":1,"unit":"mg","system":"http://unitsofmeasure.org","code":"mg"},` +
			`"high":{"value":9,"unit":"mg","system":"http://unitsofmeasure.org","code":"mg"}}]}`)}, nil
}

// TestRetrieveCarriesItsContext covers the context a retrieve is scoped to.
// CQL evaluates `[Condition]` under `context Patient` as that patient's
// conditions, and the engine had no way to say which patient it meant. The
// model relates a type to a context by search parameter — "patient" for
// Condition, "subject" for Observation — so honoring it is the provider's job
// and the request has to carry enough for it to.
func TestRetrieveCarriesItsContext(t *testing.T) {
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)
	for _, tt := range []struct{ resource, wantParam string }{
		{"Condition", "patient"},
		{"Observation", "subject"},
		{"MedicationRequest", "subject"},
	} {
		src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\n\ndefine X: [" + tt.resource + "]\n"
		rec := &requestRecorder{}
		if _, err := NewEngine(WithDataProvider(rec), WithMaxRetrieveSize(500)).
			EvaluateExpression(context.Background(), src, "X", patient, nil); err != nil {
			t.Fatalf("[%s]: %v", tt.resource, err)
		}
		if rec.last.Context != "Patient" || rec.last.ContextID != "p1" {
			t.Errorf("[%s] context = %q/%q, want Patient/p1", tt.resource, rec.last.Context, rec.last.ContextID)
		}
		if rec.last.ContextSearchParam != tt.wantParam {
			t.Errorf("[%s] search param = %q, want %q", tt.resource, rec.last.ContextSearchParam, tt.wantParam)
		}
		// One more than the caller accepts: asking for exactly the maximum makes
		// the engine unable to tell "there were more" from "that was all", and
		// a provider that honors the limit would truncate the population in
		// silence.
		if rec.last.Limit != 501 {
			t.Errorf("[%s] limit = %d, want 501 (max + 1)", tt.resource, rec.last.Limit)
		}
	}

	// Without a declared context there is nothing to scope to.
	rec := &requestRecorder{}
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\n\ndefine X: [Condition]\n"
	if _, err := NewEngine(WithDataProvider(rec)).
		EvaluateExpression(context.Background(), src, "X", nil, nil); err != nil {
		t.Fatalf("contextless retrieve: %v", err)
	}
	if rec.last.Context != "" || rec.last.ContextID != "" {
		t.Errorf("a contextless retrieve carried %q/%q", rec.last.Context, rec.last.ContextID)
	}
}

// TestOverloadDispatchByArgumentType covers picking between overloads that
// differ only in their operand type. Scoring literals alone meant the first
// candidate always won, so FHIRHelpers.ToInterval always ran its Period body
// and answered null for a Range.
func TestOverloadDispatchByArgumentType(t *testing.T) {
	src := `library T version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH

define Obs: First([Observation])
define FromPeriod: FH.ToInterval(Obs.effectivePeriod)
define FromRange: FH.ToInterval(Obs.referenceRange[0])
define FromQuantity: FH.ToQuantity(Obs.valueQuantity)
`
	engine := NewEngine(WithDataProvider(&requestRecorder{}))
	for _, tt := range []struct{ name, want string }{
		{"FromPeriod", "Interval[2020-01-01, 2020-12-31]"},
		{"FromRange", "Interval[1 'mg', 9 'mg']"},
		{"FromQuantity", "5 'mg'"},
	} {
		got, err := engine.EvaluateExpression(context.Background(), src, tt.name, nil, nil)
		if err != nil {
			t.Fatalf("evaluating %s: %v", tt.name, err)
		}
		if s := valueString(got); s != tt.want {
			t.Errorf("%s = %s, want %s", tt.name, s, tt.want)
		}
	}
}

// TestUsingVersionMustBeAvailable covers a library asking for a FHIR version
// this build does not carry. It used to be accepted in silence and evaluated
// against R4 anyway, so every path resolved against the wrong spec.
func TestUsingVersionMustBeAvailable(t *testing.T) {
	ok := "library T version '1.0'\nusing FHIR version '4.0.1'\n\ndefine X: 1\n"
	if _, err := NewEngine().EvaluateExpression(context.Background(), ok, "X", nil, nil); err != nil {
		t.Errorf("4.0.1 should evaluate: %v", err)
	}
	for _, v := range []string{"5.0.0", "3.0.2"} {
		src := "library T version '1.0'\nusing FHIR version '" + v + "'\n\ndefine X: 1\n"
		if _, err := NewEngine().EvaluateExpression(context.Background(), src, "X", nil, nil); err == nil {
			t.Errorf("using FHIR %s should be refused, not evaluated against R4", v)
		}
	}
	// A caller who supplied their own model is taken at their word.
	src := "library T version '1.0'\nusing FHIR version '5.0.0'\n\ndefine X: 1\n"
	if _, err := NewEngine(WithModelInfo(model.DefaultR4ModelInfo())).
		EvaluateExpression(context.Background(), src, "X", nil, nil); err != nil {
		t.Errorf("an explicit model should override the version check: %v", err)
	}
}

// TestEvaluationTimestampIsFrozen covers Now, TimeOfDay and Today. CQL says
// they answer with the evaluation request's timestamp, so every call in one
// evaluation agrees with every other; reading the clock per call made
// `Now() = Now()` false about once in every 1,700 evaluations, which is a
// conformance failure and a flaky build, not just an irreproducible measure.
func TestEvaluationTimestampIsFrozen(t *testing.T) {
	src := `library T version '1.0'

define SameTimeOfDay: TimeOfDay() = TimeOfDay()
define SameNow: Now() = Now()
define SameToday: Today() = Today()
define AcrossDefinitions: Now() = Anchor
define Anchor: Now()
`
	engine := NewEngine()
	// The window is roughly a millisecond wide, so one evaluation proves little.
	const runs = 2000
	for _, name := range []string{"SameTimeOfDay", "SameNow", "SameToday", "AcrossDefinitions"} {
		for i := 0; i < runs; i++ {
			got, err := engine.EvaluateExpression(context.Background(), src, name, nil, nil)
			if err != nil {
				t.Fatalf("evaluating %s: %v", name, err)
			}
			if valueString(got) != "true" {
				t.Fatalf("%s was false on run %d: the evaluation timestamp is not frozen", name, i)
			}
		}
	}
}

// countingLimitProvider honors the Limit it is given, the way a provider that
// can bound its query would.
type countingLimitProvider struct{ available int }

func (p *countingLimitProvider) Retrieve(_ context.Context, req eval.RetrieveRequest) ([]json.RawMessage, error) {
	n := p.available
	if req.Limit > 0 && n > req.Limit {
		n = req.Limit
	}
	out := make([]json.RawMessage, n)
	for i := range out {
		out[i] = json.RawMessage(`{"resourceType":"Condition","id":"c"}`)
	}
	return out, nil
}

// TestRetrieveLimitStaysReachable covers the interaction between the limit sent
// to the provider and the refusal that backs it. Asking for exactly the maximum
// makes the refusal unreachable: a provider that honors the limit returns
// exactly that many rows, the count never exceeds it, and the population is
// truncated in silence — the one outcome the limit exists to prevent.
func TestRetrieveLimitStaysReachable(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\n\ndefine X: Count([Condition])\n"

	_, err := NewEngine(WithDataProvider(&countingLimitProvider{available: 100}), WithMaxRetrieveSize(2)).
		EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err == nil {
		t.Fatal("100 resources under a limit of 2 should be refused, not truncated to 2")
	}
	var costly *ErrTooCostly
	if !errors.As(err, &costly) {
		t.Errorf("expected ErrTooCostly, got %T: %v", err, err)
	}

	// Exactly at the limit is fine, and is not mistaken for an overflow.
	got, err := NewEngine(WithDataProvider(&countingLimitProvider{available: 2}), WithMaxRetrieveSize(2)).
		EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err != nil {
		t.Fatalf("2 resources under a limit of 2 should succeed: %v", err)
	}
	if valueString(got) != "2" {
		t.Errorf("Count = %s, want 2", valueString(got))
	}
}

// TestContextScopingIsHonest covers the cases where the engine has nothing
// trustworthy to tell a provider, and must say nothing rather than something
// wrong.
func TestContextScopingIsHonest(t *testing.T) {
	// A context with no subject: a population-level run. Naming the context with
	// an empty id would ask a conforming provider to filter on nothing.
	rec := &requestRecorder{}
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\n\ndefine X: [Condition]\n"
	if _, err := NewEngine(WithDataProvider(rec)).
		EvaluateExpression(context.Background(), src, "X", nil, nil); err != nil {
		t.Fatalf("population-level retrieve: %v", err)
	}
	if rec.last.Context != "" || rec.last.ContextID != "" {
		t.Errorf("a subjectless run claimed context %q/%q", rec.last.Context, rec.last.ContextID)
	}

	// The context in force is the statement's, not the library's first.
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)
	rec = &requestRecorder{}
	switched := `library T version '1.0'
using FHIR version '4.0.1'
context Patient
define A: [Condition]
context Unfiltered
define B: [Condition]
`
	if _, err := NewEngine(WithDataProvider(rec)).
		EvaluateExpression(context.Background(), switched, "B", patient, nil); err != nil {
		t.Fatalf("switched context: %v", err)
	}
	if rec.last.Context == "Patient" {
		t.Error("a definition under `context Unfiltered` was scoped to Patient")
	}
}
