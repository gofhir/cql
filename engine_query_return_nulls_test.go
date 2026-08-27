package cql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofhir/cql/eval"
)

// sparseObs serves three observations where the middle one has no value, which is
// what a FHIR server returns when a field was not recorded.
type sparseObs struct{}

func (sparseObs) Retrieve(_ context.Context, req eval.RetrieveRequest) ([]json.RawMessage, error) {
	if req.ResourceType != "Observation" {
		return nil, nil
	}
	return []json.RawMessage{
		json.RawMessage(`{"resourceType":"Observation","id":"o1","status":"final",` +
			`"valueQuantity":{"value":10,"unit":"mg"}}`),
		json.RawMessage(`{"resourceType":"Observation","id":"o2","status":"final"}`),
		json.RawMessage(`{"resourceType":"Observation","id":"o3","status":"final",` +
			`"valueQuantity":{"value":30,"unit":"mg"}}`),
	}, nil
}

func evalSparse(t *testing.T, expr string) string {
	t.Helper()
	src := "library T version '1.0'\n" +
		"using FHIR version '4.0.1'\n" +
		"include FHIRHelpers version '4.0.1' called FH\n" +
		"context Patient\n" +
		"define A: " + expr + "\n"
	got, err := NewEngine(WithDataProvider(sparseObs{})).EvaluateExpression(
		context.Background(), src, "A", []byte(`{"resourceType":"Patient","id":"p1"}`), nil)
	if err != nil {
		t.Fatalf("%s: %v", expr, err)
	}
	if got == nil {
		return "null"
	}
	return valueString(got)
}

// TestQueryReturnKeepsNulls covers a silent drop, and one the engine contradicts
// itself about rather than one a specification has to settle.
//
// A query's return clause discarded any element whose expression evaluated to
// null, so a projection came out shorter than the rows it projected. Three
// observations, one of them without a value:
//
//	[Observation] O return all (O.value as Quantity).value
//	  was {10, 30}       is {10, null, 30}
//
// Count does not show it, and that is worth recording because it is how I
// measured this wrong the first time: CQL's Count counts non-nulls, so 2 is its
// right answer either way. What shows it is position. Two projections of the same
// source must have the same length, and they did not — `ids[2]` was 'o3' while
// `values[2]` was past the end, so nothing derived from one query could be lined
// up against anything derived from the same query.
//
// The engine contradicts itself here rather than the specification being unclear.
// A list written by hand keeps its nulls: `{null, true}` has two elements. And the
// same function decided three ways — the multi-source branch and the no-return
// branch appended unconditionally, and only the return branch filtered, with
// nothing saying why.
//
// `return all` settles it on its own terms. Its purpose is to remove nothing,
// since `return` alone already means `return distinct`, so a `return all` that
// removes elements does the one thing the keyword rules out.
func TestQueryReturnKeepsNulls(t *testing.T) {
	// The premise: there are three observations, and the middle one has no value.
	if got := evalSparse(t, "Count([Observation])"); got != "3" {
		t.Fatalf("Count([Observation]) = %s, want 3", got)
	}

	const values = "[Observation] O return all (O.value as FHIR.Quantity).value"
	const ids = "[Observation] O return all O.id"

	for _, tt := range []struct{ expr, want string }{
		// The null holds its place.
		{values, "{10, null, 30}"},
		{"(" + values + ")[1]", "null"},
		{"(" + values + ")[2]", "30"},
		{"Last(" + values + ")", "30"},

		// So the two projections of one source line up, which is the property
		// that was actually broken.
		{"(" + ids + ")[2] = 'o3'", "true"},
		{"(" + values + ")[2] = 30", "true"},

		// Count still counts non-nulls, which is CQL's rule and unchanged.
		{"Count(" + values + ")", "2"},
		{"Count({null, true})", "1"},
	} {
		if got := evalSparse(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}

	// Sum and Avg skip nulls of their own accord, so keeping them changes no
	// arithmetic — only what the list says about how many rows it describes.
	for _, tt := range []struct{ expr, want string }{
		{"Sum(" + values + ")", "40"},
		{"Avg(" + values + ")", "20"},
	} {
		if got := evalSparse(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}
