package cql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofhir/cql/eval"
)

// partialDatePeriods serves three encounters the way a FHIR server does, with the
// first period's start written to the month. A FHIR dateTime is allowed to carry
// any precision, and real data uses that.
type partialDatePeriods struct{}

func (partialDatePeriods) Retrieve(_ context.Context, req eval.RetrieveRequest) ([]json.RawMessage, error) {
	if req.ResourceType != "Encounter" {
		return nil, nil
	}
	return []json.RawMessage{
		json.RawMessage(`{"resourceType":"Encounter","id":"e1","status":"finished",` +
			`"period":{"start":"2020-03","end":"2020-04-30"}}`),
		json.RawMessage(`{"resourceType":"Encounter","id":"e2","status":"finished",` +
			`"period":{"start":"2020-06-01","end":"2020-07-15"}}`),
		json.RawMessage(`{"resourceType":"Encounter","id":"e3","status":"finished",` +
			`"period":{"start":"2020-09-01","end":"2020-09-20"}}`),
	}, nil
}

// TestPublishedMeasureArithmeticSurvivesPartialDates evaluates a function copied
// from a published measure, on data shaped the way a FHIR server serves it.
//
// AdvancedIllnessandFrailtyExclusionECQMFHIR4 defines this, and the exclusion it
// feeds turns on whether the total reaches 90 days:
//
//	define function "CumulativeDays"(Intervals List<Interval<DateTime>> ):
//	  Sum((collapse Intervals) CollapsedInterval
//	        return all duration in days of CollapsedInterval)
//
// A `Sum` of `duration in days`. Where one period's start is written to the month
// — which FHIR permits and real data uses — that duration is an uncertainty, and
// summing it used to read it as the number zero:
//
//	                  before          after
//	CumulativeDays    63              Interval[90, 123]
//	… >= 90           false           true
//
// 63 is not null and not an error. It is one of the three periods silently
// contributing nothing, landing under the threshold, and excluding a patient who
// belongs in the exclusion. That is the whole reason uncertainties became a type
// of their own, measured on the library that ships it rather than on a probe.
func TestPublishedMeasureArithmeticSurvivesPartialDates(t *testing.T) {
	src := primitivePreamble + `
define function "CumulativeDays"(Intervals List<Interval<DateTime>> ):
  Sum((collapse Intervals) CollapsedInterval
        return all duration in days of CollapsedInterval
  )

define Periods: [Encounter] E return E.period
define Days: "CumulativeDays"(Periods)
`
	ask := func(expr string) string {
		t.Helper()
		got, err := NewEngine(WithDataProvider(partialDatePeriods{})).EvaluateExpression(
			context.Background(), src+"define A: "+expr+"\n", "A",
			[]byte(`{"resourceType":"Patient","id":"p1","birthDate":"1980-05-15"}`), nil)
		if err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
		return valueString(got)
	}

	// The total is a range because one of the three periods is, and the engine
	// says so rather than picking a number out of it.
	if got := ask("Days"); got != "Interval[90, 123]" {
		t.Errorf("CumulativeDays = %s, want Interval[90, 123]", got)
	}

	// The threshold the measure actually tests. Answerable because 90 is the
	// least the total can be, so every possibility clears it.
	if got := ask("Days >= 90"); got != "true" {
		t.Errorf("CumulativeDays >= 90 = %s, want true — this answered false, on a "+
			"total of 63 that counted a partial date as zero", got)
	}

	// Knowable at both ends, so the uncertainty does not make the measure
	// useless — only honest about the one question it cannot settle.
	if got := ask("Days > 200"); got != "false" {
		t.Errorf("CumulativeDays > 200 = %s, want false", got)
	}
	if got := ask("Days < 10"); got != "false" {
		t.Errorf("CumulativeDays < 10 = %s, want false", got)
	}
	// And it declines where the answer really does depend on which value it is.
	if got := ask("Days >= 100"); got != "null" {
		t.Errorf("CumulativeDays >= 100 = %s, want null: 100 is inside [90, 123]", got)
	}
}
