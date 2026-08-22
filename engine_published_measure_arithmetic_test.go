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

// mixedPrecisionPeriods serves what a real patient looks like: some encounters
// with a full timestamp, some with a partial date. Both are valid FHIR, and a
// server returns whatever was recorded.
type mixedPrecisionPeriods struct{}

func (mixedPrecisionPeriods) Retrieve(_ context.Context, req eval.RetrieveRequest) ([]json.RawMessage, error) {
	if req.ResourceType != "Encounter" {
		return nil, nil
	}
	return []json.RawMessage{
		// Exact: 60 days.
		json.RawMessage(`{"resourceType":"Encounter","id":"e1","status":"finished",` +
			`"period":{"start":"2020-03-01T08:00:00Z","end":"2020-04-30T08:00:00Z"}}`),
		// Partial start, so this duration is a range.
		json.RawMessage(`{"resourceType":"Encounter","id":"e2","status":"finished",` +
			`"period":{"start":"2020-06","end":"2020-07-15T10:00:00Z"}}`),
	}, nil
}

// TestSumMixesCertainWithUncertain covers the case that appears only once the
// FHIR dateTime promotion and the uncertainty type are both in place, and that
// broke the very measure the uncertainty type was written for.
//
// Refusing to aggregate an uncertainty beside a plain number was modeled on
// refusing to aggregate a Quantity beside one, and the two are not alike. A bare
// number next to a Quantity has no unit and the collection cannot supply one. A
// certain number next to an uncertainty has an obvious sum: add it to both
// bounds, which is exactly what `+` already does.
//
// It went unnoticed because it could not happen before. A FHIR Period whose
// endpoints were typed off their JSON text always yielded a certain duration, so
// Sum never saw a mixture. Promote the endpoints and a patient with one complete
// encounter and one partially dated one — the ordinary case — produces one.
//
// So CumulativeDays, the function this was all measured against, went from
// answering 0 to answering an error.
func TestSumMixesCertainWithUncertain(t *testing.T) {
	src := primitivePreamble + `
define function "CumulativeDays"(Intervals List<Interval<DateTime>> ):
  Sum((collapse Intervals) CollapsedInterval
        return all duration in days of CollapsedInterval
  )

define Periods: [Encounter] E return E.period
define Days: "CumulativeDays"(Periods)
`
	ask := func(expr string) (string, error) {
		t.Helper()
		got, err := NewEngine(WithDataProvider(mixedPrecisionPeriods{})).EvaluateExpression(
			context.Background(), src+"define A: "+expr+"\n", "A",
			[]byte(`{"resourceType":"Patient","id":"p1","birthDate":"1980-05-15"}`), nil)
		if err != nil {
			return "", err
		}
		return valueString(got), nil
	}

	// 60 exact days, plus a range for the encounter whose start is written to the
	// month. June is a month, so that second duration is Interval[14, 44] and the
	// total is Interval[74, 104].
	//
	// Not [75, 104], which is what the same expression written with a Date literal
	// gives. `@2020-06` is a Date and a Date *is* the month; a promoted FHIR
	// dateTime written to the month is an indeterminate instant inside it, one day
	// wider at the near end. Checked by asking both:
	//
	//	duration in days of Interval[@2020-06, @2020-07-15T10:00:00Z]          → [15, 44]
	//	duration in days of Interval[DateTime(2020, 6), @2020-07-15T10:00:00Z] → [14, 44]
	got, err := ask("Days")
	if err != nil {
		t.Fatalf("CumulativeDays over a mixture: %v", err)
	}
	if got != "Interval[74, 104]" {
		t.Errorf("CumulativeDays = %s, want Interval[74, 104]", got)
	}

	// The thresholds still resolve, which is the whole point of carrying the range
	// rather than refusing it — and they decline where the range straddles.
	for _, tt := range []struct{ expr, want string }{
		{"Days >= 70", "true"},
		{"Days < 200", "true"},
		{"Days > 200", "false"},
		{"Days >= 105", "false"},
		{"Days >= 90", "null"},
	} {
		got, askErr := ask(tt.expr)
		if askErr != nil {
			t.Errorf("%s: %v", tt.expr, askErr)
			continue
		}
		if tt.want == "null" {
			if got != "null" {
				t.Errorf("%s = %s, want null: 90 is inside [74, 104]", tt.expr, got)
			}
			continue
		}
		if got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}

	// Sum answers exactly what the + operator answers, which is the property that
	// makes delegating to it worth doing rather than reimplementing it.
	viaSum, err := ask("Sum({60, duration in days of Interval[DateTime(2020, 6), @2020-07-15T10:00:00Z]})")
	if err != nil {
		t.Fatalf("Sum: %v", err)
	}
	viaOperator, err := ask("60 + (duration in days of Interval[DateTime(2020, 6), @2020-07-15T10:00:00Z])")
	if err != nil {
		t.Fatalf("operator: %v", err)
	}
	if viaSum != viaOperator {
		t.Errorf("Sum gave %s, + gave %s", viaSum, viaOperator)
	}
}
