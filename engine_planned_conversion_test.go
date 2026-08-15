package cql

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gofhir/cql/eval"
	"github.com/gofhir/cql/fhirhelpers"
)

// plannedProvider serves one encounter and one observation whose value is a
// FHIR.Quantity — the type that showed the gap.
type plannedProvider struct{}

func (plannedProvider) Retrieve(_ context.Context, req eval.RetrieveRequest) ([]json.RawMessage, error) {
	switch req.ResourceType {
	case "Encounter":
		return []json.RawMessage{json.RawMessage(`{"resourceType":"Encounter","id":"e1",` +
			`"status":"finished","period":{"start":"2020-03-01T10:00:00Z","end":"2020-03-05T10:00:00Z"}}`)}, nil
	case "Observation":
		return []json.RawMessage{json.RawMessage(`{"resourceType":"Observation","id":"o1",` +
			`"valueQuantity":{"value":9.1,"unit":"mg"},"effectiveDateTime":"2020-03-02T08:00:00Z"}`)}, nil
	}
	return nil, nil
}

const plannedPreamble = `library Planned version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
context Patient
define Enc: First([Encounter])
define Obs: First([Observation])
define Q: Obs.value as FHIR.Quantity
`

// TestPlannedConversionsCoverWhatOperatorsMissed covers the conversions the
// engine now applies because the semantic phase decided they were needed,
// rather than because an operator thought to ask.
//
// Every one of these failed or answered wrongly before. The two that answered
// wrongly are the reason this matters more than the ones that failed: a measure
// that errors gets looked at, and one that quietly says false does not.
func TestPlannedConversionsCoverWhatOperatorsMissed(t *testing.T) {
	engine := NewEngine(WithDataProvider(plannedProvider{}))
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)

	for _, tc := range []struct {
		name string
		expr string
		want string
		note string
	}{
		{"Add", `Q + 1 'mg'`, "10.1 'mg'", "arithmetic on a FHIR.Quantity failed outright"},
		{"Subtract", `Q - 1 'mg'`, "8.1 'mg'", ""},
		{"Multiply", `Q * 2`, "18.2 'mg'", ""},
		{"Less", `Q < 10 'mg'`, "true", "comparison failed outright"},
		{"Greater", `Q > 1 'mg'`, "true", ""},
		{"Between", `Q between 1 'mg' and 20 'mg'`, "true", ""},
		{"Equal", `Q = 9.1 'mg'`, "true", "answered false, on a value that is exactly 9.1 mg"},
		{"Sum", `Sum({ Q, 2 'mg' })`, "11.1 'mg'", "answered 2 'mg', ignoring the other addend"},
		{"IfBranch", `if true then Q else 0 'mg'`, "9.1 'mg'", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := plannedPreamble + "define R: " + tc.expr + "\n"
			got, err := engine.EvaluateExpression(context.Background(), src, "R", patient, nil)
			if err != nil {
				t.Fatalf("%s: %v\n%s", tc.expr, err, tc.note)
			}
			if got == nil {
				t.Fatalf("%s answered null; want %s\n%s", tc.expr, tc.want, tc.note)
			}
			if got.String() != tc.want {
				t.Errorf("%s is %s, want %s\n%s", tc.expr, got, tc.want, tc.note)
			}
		})
	}
}

// TestAPlanBelongsToItsOwnLibrary covers what happens to a decision made about
// one library when another is evaluated.
//
// Decisions are recorded against AST nodes, so a plan can only ever match the
// library it was made for; anything else falls back to the engine deciding for
// itself, which is what an included library relies on. This checks the fallback
// is really there rather than assumed.
func TestAPlanBelongsToItsOwnLibrary(t *testing.T) {
	engine := NewEngine(WithDataProvider(plannedProvider{}))
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)

	// Two libraries with identical text but separate parses, so their nodes are
	// different objects entirely.
	src := plannedPreamble + "define R: start of Enc.period\n"
	first, err := engine.Parse(src)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	second, err := NewEngine(WithDataProvider(plannedProvider{})).Parse(src)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}

	for _, lib := range []*Library{first, second} {
		got, err := engine.EvaluateParsedExpression(context.Background(), lib, "R", patient, nil)
		if err != nil {
			t.Fatalf("evaluating: %v", err)
		}
		if got == nil || got.Type() != "DateTime" {
			t.Errorf("start of a period is %v, want a DateTime", got)
		}
	}
}

// --- what an adversarial review of the planned conversions found ---

// reviewResolver serves the real FHIRHelpers, plus a library that aliases a
// different library under that same name — which is the shape that broke the
// conversion memo.
func reviewResolver(_ context.Context, name, _ string) (string, error) {
	switch name {
	case "FHIRHelpers":
		return fhirhelpers.Source, nil
	case "Fake":
		return `library Fake version '1.0'
using FHIR version '4.0.1'
define function ToInterval(p FHIR.Period): Interval[@2000-01-01T00:00:00Z, @2000-12-31T00:00:00Z]
`, nil
	case "Helper":
		return `library Helper version '1.0'
using FHIR version '4.0.1'
include Fake version '1.0' called FHIRHelpers
define function StartOf(p FHIR.Period): start of p
`, nil
	}
	return "", fmt.Errorf("no library named %s", name)
}

// TestTheConversionMemoIsPerLibrary covers remembering which overload of a
// conversion function takes which type.
//
// FHIRHelpers is the library everybody aliases, so two libraries in one
// evaluation can each mean something different by the name. Keying the memo on
// the name let whichever converted first decide the function body the other one
// ran: the two definitions below came back identical, and which value they both
// took depended on the order they were written in.
func TestTheConversionMemoIsPerLibrary(t *testing.T) {
	src := `library M version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
include Helper version '1.0' called H
context Patient
define Enc: First([Encounter])
define Mine: start of Enc.period
define Theirs: H.StartOf(Enc.period)
define MineFirst: { Mine, Theirs }
define TheirsFirst: { Theirs, Mine }
`
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)
	for _, tc := range []struct{ name, want string }{
		{"MineFirst", "{2020-03-01T10:00:00Z, 2000-01-01T00:00:00Z}"},
		{"TheirsFirst", "{2000-01-01T00:00:00Z, 2020-03-01T10:00:00Z}"},
	} {
		// A fresh engine per case: the point is that order within one
		// evaluation cannot change the answer.
		engine := NewEngine(WithDataProvider(plannedProvider{}), WithLibraryResolver(reviewResolver))
		got, err := engine.EvaluateExpression(context.Background(), src, tc.name, patient, nil)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got == nil || got.String() != tc.want {
			t.Errorf("%s is %v, want %s", tc.name, got, tc.want)
		}
	}
}

// TestCaseBranchesConvergeLikeIfBranches covers the branches of a case, which
// were left out when the branches of an if were made to converge — so the same
// expression behaved one way written with `if` and another with `case`.
func TestCaseBranchesConvergeLikeIfBranches(t *testing.T) {
	engine := NewEngine(WithDataProvider(plannedProvider{}))
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)

	for _, tc := range []struct{ name, expr, want string }{
		{"if", `(if true then Q else 2 'mg') = 9.1 'mg'`, "true"},
		{"case", `(case when true then Q else 2 'mg' end) = 9.1 'mg'`, "true"},
		{"case arithmetic", `(case when true then Q else 2 'mg' end) + 1 'mg'`, "10.1 'mg'"},
	} {
		src := plannedPreamble + "define R: " + tc.expr + "\n"
		got, err := engine.EvaluateExpression(context.Background(), src, "R", patient, nil)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if got == nil || got.String() != tc.want {
			t.Errorf("%s: %s is %v, want %s", tc.name, tc.expr, got, tc.want)
		}
	}
}

// TestAggregatesConvertWhatTheyAreGiven covers Sum and its siblings, which work
// on CQL's own types and walked straight past the FHIR quantities they did not
// recognize — answering with a number quietly too small.
//
// The null case is the one that hid it: with a second quantity beside it the
// list converged on System.Quantity by itself, and with a null beside it there
// was nothing to converge on.
//
// Avg and Median are not covered because they do not work on quantities at all,
// FHIR or otherwise — `Avg({ 1 'mg', 2 'mg' })` is 0 and `Median` of the same is
// null. That is a gap in funcs/, not in what is converted, and it predates this.
func TestAggregatesConvertWhatTheyAreGiven(t *testing.T) {
	engine := NewEngine(WithDataProvider(plannedProvider{}))
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)

	for _, tc := range []struct{ expr, want string }{
		{`Sum({ Q, 2 'mg' })`, "11.1 'mg'"},
		{`Sum({ Q, null })`, "9.1 'mg'"},
		{`Max({ Q, null })`, "9.1 'mg'"},
		{`Min({ Q, 2 'mg' })`, "2 'mg'"},
	} {
		src := plannedPreamble + "define R: " + tc.expr + "\n"
		got, err := engine.EvaluateExpression(context.Background(), src, "R", patient, nil)
		if err != nil {
			t.Errorf("%s: %v", tc.expr, err)
			continue
		}
		if got == nil || got.String() != tc.want {
			t.Errorf("%s is %v, want %s", tc.expr, got, tc.want)
		}
	}
}

// TestAnIntervalConvertsAtItsBoundaries covers the elementwise case that is not
// a list. The conversion applies to the two points; calling it on the interval
// itself yields nothing, and left the boundaries as raw FHIR objects.
func TestAnIntervalConvertsAtItsBoundaries(t *testing.T) {
	engine := NewEngine(WithDataProvider(plannedProvider{}))
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)

	src := plannedPreamble + "define R: { Interval[Q, Q], Interval[1 'mg', 2 'mg'] }\n"
	got, err := engine.EvaluateExpression(context.Background(), src, "R", patient, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	want := "{Interval[9.1 'mg', 9.1 'mg'], Interval[1 'mg', 2 'mg']}"
	if got == nil || got.String() != want {
		t.Errorf("intervals are %v, want %s", got, want)
	}
}
