package cql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/eval"
)

// provenanceRecorder is the shape a caller building an audit trail would write:
// a trace listener that also takes the two optional observers, so the tree of
// expressions is joined by the decisions that came from outside it.
type provenanceRecorder struct {
	depth       int
	nodes       []string
	retrieves   []eval.RetrieveRequest
	rowCounts   []int
	terminology []string
}

func (r *provenanceRecorder) OnEnter(expr ast.Expression) {
	r.nodes = append(r.nodes, strings.Repeat(" ", r.depth)+fmt.Sprintf("%T", expr))
	r.depth++
}

func (r *provenanceRecorder) OnExit(ast.Expression, fptypes.Value, error) {
	if r.depth > 0 {
		r.depth--
	}
}

func (r *provenanceRecorder) OnRetrieve(req eval.RetrieveRequest, resultCount int, _ error) {
	r.retrieves = append(r.retrieves, req)
	r.rowCounts = append(r.rowCounts, resultCount)
}

func (r *provenanceRecorder) OnTerminologyCheck(code, system, valueSet string, in bool, _ error) {
	r.terminology = append(r.terminology,
		fmt.Sprintf("%s|%s in %s = %v", system, code, valueSet, in))
}

type provenanceProvider struct{}

func (provenanceProvider) Retrieve(_ context.Context, _ eval.RetrieveRequest) ([]json.RawMessage, error) {
	return []json.RawMessage{json.RawMessage(
		`{"resourceType":"Condition","id":"c1","code":{"coding":[{"system":"http://snomed.info/sct","code":"44054006"}]}}`)}, nil
}

type provenanceTerminology struct{}

func (provenanceTerminology) InValueSet(_ context.Context, code, _, _ string) (bool, error) {
	return code == "44054006", nil
}

// TestProvenanceOfAPopulationDecision covers what it takes to justify a
// population decision after the fact.
//
// The expression tree the trace listener already builds shows the derivation,
// but not the two facts that came from outside it: which query produced the
// resources, and what the terminology server was asked. Neither is recoverable
// from the syntax — `[Condition]` carries no code path, no subject and no
// context, and `code in "Diabetes"` records the answer without the question.
func TestProvenanceOfAPopulationDecision(t *testing.T) {
	src := `library M version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
valueset "T2DM": 'http://example.org/vs/t2dm'
context Patient

define InDenominator: exists ([Condition] C where FH.ToCode(C.code.coding[0]) in "T2DM")
`
	rec := &provenanceRecorder{}
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)
	got, err := NewEngine(
		WithDataProvider(provenanceProvider{}),
		WithTerminologyProvider(provenanceTerminology{}),
		WithTraceListener(rec),
	).EvaluateExpression(context.Background(), src, "InDenominator", patient, nil)
	if err != nil {
		t.Fatalf("evaluating: %v", err)
	}
	if valueString(got) != "true" {
		t.Fatalf("InDenominator = %s, want true", valueString(got))
	}

	// The derivation itself.
	if len(rec.nodes) == 0 {
		t.Error("no expression tree was recorded")
	}

	// Which query the resources came from, including what the syntax does not
	// say: the code path from the model, and the patient it was scoped to.
	if len(rec.retrieves) != 1 {
		t.Fatalf("recorded %d retrieves, want 1", len(rec.retrieves))
	}
	req := rec.retrieves[0]
	if req.ResourceType != "Condition" {
		t.Errorf("resource type = %q, want Condition", req.ResourceType)
	}
	if req.Context != "Patient" || req.ContextID != "p1" {
		t.Errorf("context = %q/%q, want Patient/p1", req.Context, req.ContextID)
	}
	if req.ContextSearchParam != "patient" {
		t.Errorf("search param = %q, want patient", req.ContextSearchParam)
	}
	if rec.rowCounts[0] != 1 {
		t.Errorf("row count = %d, want 1", rec.rowCounts[0])
	}

	// What the terminology server was asked, which is the fact the decision
	// turned on.
	if len(rec.terminology) != 1 {
		t.Fatalf("recorded %d terminology checks, want 1", len(rec.terminology))
	}
	want := "http://snomed.info/sct|44054006 in http://example.org/vs/t2dm = true"
	if rec.terminology[0] != want {
		t.Errorf("terminology check = %q, want %q", rec.terminology[0], want)
	}
}

// plainListener implements only TraceListener, without the observers, which is
// what an existing implementation looks like.
type plainListener struct{ entered int }

func (p *plainListener) OnEnter(ast.Expression)                      { p.entered++ }
func (p *plainListener) OnExit(ast.Expression, fptypes.Value, error) {}

// TestObserversAreOptional covers a listener that predates them. The observers
// are separate interfaces rather than methods on TraceListener so that adding
// them does not oblige every implementation to grow.
func TestObserversAreOptional(t *testing.T) {
	src := `library M version '1.0'
using FHIR version '4.0.1'
define X: Count([Condition])
`
	listener := &plainListener{}
	got, err := NewEngine(WithDataProvider(provenanceProvider{}), WithTraceListener(listener)).
		EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err != nil {
		t.Fatalf("evaluating: %v", err)
	}
	if valueString(got) != "1" {
		t.Errorf("X = %s, want 1", valueString(got))
	}
	if listener.entered == 0 {
		t.Error("a listener without the observers should still receive the tree")
	}
}

// TestPrivateSurvivesTheMemo covers the definition cache. Access has to be
// decided from the declaration and not from what has already been evaluated:
// the cache fills as a library evaluates its own definitions, and is pre-seeded
// with declared codes and concepts, so consulting it first let a private
// definition escape as soon as anything had read it.
func TestPrivateSurvivesTheMemo(t *testing.T) {
	src := "library T version '1.0'\n\ndefine private Hidden: 42\ndefine Pub: Hidden + 1\n"
	engine := NewEngine()
	lib, err := engine.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Asking outright.
	if _, askErr := engine.EvaluateParsedExpression(context.Background(), lib, "Hidden", nil, nil); askErr == nil {
		t.Error("a private definition should be refused")
	}

	// The engine builds a fresh context per call, so the leak needs the same
	// evaluator to be reused — which is what a caller holding an eval.Evaluator
	// does, and what EvaluateLibrary does internally.
	results, err := engine.EvaluateParsedLibrary(context.Background(), lib, nil, nil)
	if err != nil {
		t.Fatalf("EvaluateParsedLibrary: %v", err)
	}
	if _, listed := results["Hidden"]; listed {
		t.Error("a private definition was returned among the results")
	}
	if valueString(results["Pub"]) != "43" {
		t.Errorf("Pub = %s, want 43 — a private definition is still evaluated", valueString(results["Pub"]))
	}
}
