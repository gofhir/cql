package cql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofhir/cql/eval"
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
