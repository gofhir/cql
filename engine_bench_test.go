package cql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofhir/cql/eval"
)

// benchProvider serves a handful of resources with the shapes a measure reads:
// periods, choice elements, codes, and repeated elements.
type benchProvider struct{}

func (benchProvider) Retrieve(_ context.Context, req eval.RetrieveRequest) ([]json.RawMessage, error) {
	switch req.ResourceType {
	case "Patient":
		return []json.RawMessage{json.RawMessage(
			`{"resourceType":"Patient","id":"p1","birthDate":"1980-05-15"}`)}, nil
	case "Encounter":
		out := make([]json.RawMessage, 0, 20)
		for i := range 20 {
			out = append(out, json.RawMessage(`{"resourceType":"Encounter","id":"e`+
				string(rune('a'+i))+`","status":"finished",`+
				`"class":{"code":"IMP"},`+
				`"period":{"start":"2020-03-01T08:00:00Z","end":"2020-03-05T10:00:00Z"},`+
				`"location":[{"location":{"reference":"Location/1"}},`+
				`{"location":{"reference":"Location/2"}}]}`))
		}
		return out, nil
	case "Observation":
		out := make([]json.RawMessage, 0, 20)
		for i := range 20 {
			out = append(out, json.RawMessage(`{"resourceType":"Observation","id":"o`+
				string(rune('a'+i))+`","status":"final",`+
				`"effectiveDateTime":"2020-03-02T09:00:00Z",`+
				`"valueQuantity":{"value":12.5,"unit":"mg"}}`))
		}
		return out, nil
	}
	return nil, nil
}

const benchLibrary = `library Bench version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers

context Patient

define MeasurementPeriod: Interval[@2020-01-01T00:00:00.0, @2021-01-01T00:00:00.0)

define Encounters:
  [Encounter] E
    where E.status.value = 'finished'
      and E.period during MeasurementPeriod

define Observations:
  [Observation] O
    where O.status.value = 'final'
      and O.effective during MeasurementPeriod

define LongStays:
  Encounters E where duration in days of E.period >= 3

define TotalDays:
  Sum(Encounters E return all duration in days of E.period)

define Values:
  Observations O return O.value as FHIR.Quantity

define Result:
  Count(LongStays) + Count(Values)
`

// BenchmarkEvaluateMeasureShapedLibrary exists so a claim about performance can
// be checked rather than read off a changelog. It walks the shapes a measure
// walks: a retrieve per resource type, a period compared against a measurement
// period, a choice element, a duration, and an aggregate over both.
func BenchmarkEvaluateMeasureShapedLibrary(b *testing.B) {
	engine := NewEngine(WithDataProvider(benchProvider{}))
	patient := []byte(`{"resourceType":"Patient","id":"p1","birthDate":"1980-05-15"}`)
	ctx := context.Background()

	// Warm the compiled-library cache, so the loop measures evaluation rather
	// than parsing once and evaluating many times.
	if _, err := engine.EvaluateExpression(ctx, benchLibrary, "Result", patient, nil); err != nil {
		b.Fatalf("warmup: %v", err)
	}

	for b.Loop() {
		if _, err := engine.EvaluateExpression(ctx, benchLibrary, "Result", patient, nil); err != nil {
			b.Fatalf("%v", err)
		}
	}
}

// BenchmarkEvaluateFromCold measures the parse and check as well, which is what
// a caller evaluating one library once actually pays.
func BenchmarkEvaluateFromCold(b *testing.B) {
	patient := []byte(`{"resourceType":"Patient","id":"p1","birthDate":"1980-05-15"}`)
	ctx := context.Background()

	for b.Loop() {
		engine := NewEngine(WithDataProvider(benchProvider{}))
		if _, err := engine.EvaluateExpression(ctx, benchLibrary, "Result", patient, nil); err != nil {
			b.Fatalf("%v", err)
		}
	}
}
