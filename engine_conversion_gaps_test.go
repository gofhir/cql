package cql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofhir/cql/eval"
)

const conversionPreamble = `library T version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
parameter MP Interval<DateTime> default Interval[@2020-01-01, @2020-12-31]

context Patient

`

// periodProvider serves one Encounter whose period sits inside 2020.
type periodProvider struct{}

func (periodProvider) Retrieve(_ context.Context, _ eval.RetrieveRequest) ([]json.RawMessage, error) {
	return []json.RawMessage{json.RawMessage(
		`{"resourceType":"Encounter","id":"e1","period":{"start":"2020-03-01","end":"2020-03-05"}}`)}, nil
}

func evalWithPeriod(t *testing.T, expr string) (string, error) {
	t.Helper()
	src := conversionPreamble + "define A: " + expr + "\n"
	got, err := NewEngine(WithDataProvider(periodProvider{})).EvaluateExpression(
		context.Background(), src, "A", []byte(`{"resourceType":"Patient","id":"p1"}`), nil)
	if err != nil {
		return "", err
	}
	return valueString(got), nil
}

// TestSetOperatorsConvertPeriodsToIntervals covers three operators that read a
// FHIR.Period as a list.
//
// union, intersect and except work on lists and on intervals, and decided which
// by asking whether each operand already was an Interval. A Period is not one,
// so `Enc.period intersect MP` took the list branch, found nothing in common
// between a Period and an interval, and answered with the empty list — a
// plausible answer to a question nobody asked, in a published measure that uses
// it to work out how long a stay overlapped the measurement period.
//
// overlaps and during had been converting all along, which is why this went
// unnoticed.
func TestSetOperatorsConvertPeriodsToIntervals(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`First([Encounter]).period intersect MP`, "Interval[2020-03-01, 2020-03-05]"},
		{`First([Encounter]).period union MP`, "Interval[2020-01-01, 2020-12-31]"},

		// The same with the operands the other way round.
		{`MP intersect First([Encounter]).period`, "Interval[2020-03-01, 2020-03-05]"},
	} {
		got, err := evalWithPeriod(t, tt.expr)
		if err != nil {
			t.Errorf("%s: %v", tt.expr, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// The list operations these operators exist for are untouched: a converted
// Period takes a branch of its own and nothing that already worked goes through
// it.
func TestSetOperatorsStillWorkOnLists(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`{1, 2, 3} intersect {2, 3, 4}`, "{2, 3}"},
		{`{1, 2} union {3}`, "{1, 2, 3}"},
		{`{1, 2, 3} except {2}`, "{1, 3}"},
		{`Interval[@2020-01-01, @2020-06-01] intersect MP`, "Interval[2020-01-01, 2020-06-01]"},
	} {
		got, err := evalWithPeriod(t, tt.expr)
		if err != nil {
			t.Errorf("%s: %v", tt.expr, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestCheckAgreesWithTheEvaluatorOnConversions covers the other half of each
// gap. Where the evaluator converts, the semantic phase has to agree, or it
// reports a mistake in a library that evaluates correctly — which is what kept
// published measures from checking out.
func TestCheckAgreesWithTheEvaluatorOnConversions(t *testing.T) {
	for _, tt := range []struct{ name, expr string }{
		{
			// FHIR.id declares no conversion of its own; it extends FHIR.string,
			// which converts to System.String. Asking only about the type itself
			// found nothing, so a String operator applied to Encounter.id was
			// reported as applied to something that is not a String.
			"a primitive that inherits its conversion",
			`'discharged: ' & First([Encounter]).id`,
		},
		{
			"a period in a set operation",
			`First([Encounter]).period intersect MP`,
		},
		{
			"a period in an interval operation, which always worked",
			`First([Encounter]).period overlaps MP`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			src := conversionPreamble + "define A: " + tt.expr + "\n"
			diags, err := NewEngine().Check(src)
			if err != nil {
				t.Fatalf("checking: %v", err)
			}
			if errs := diags.Errors(); len(errs) != 0 {
				t.Errorf("want no findings, got %v", errs)
			}
			// And it evaluates, which is the point of agreeing.
			if _, err := evalWithPeriod(t, tt.expr); err != nil {
				t.Errorf("evaluating: %v", err)
			}
		})
	}
}

// TestCheckAcceptsAChoiceOfIntervals covers an operator that always needs an
// interval, handed a choice every branch of which is one.
//
// FHIR reaches this constantly: Condition.onset is a choice, so a helper that
// normalizes it returns Choice<Interval<Quantity>, Interval<DateTime>>, and
// `start of` that was reported as applied to something that is not an interval.
func TestCheckAcceptsAChoiceOfIntervals(t *testing.T) {
	src := `library T version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
context Patient

define function "Normalize"(onset Choice<FHIR.dateTime, FHIR.Period, FHIR.Quantity, FHIR.Range, FHIR.string>):
  if onset is FHIR.Period then FH.ToInterval(onset as FHIR.Period)
    else Interval[FH.ToDateTime(onset as FHIR.dateTime), FH.ToDateTime(onset as FHIR.dateTime)]

define Onsets:
  [Condition] C
    return Interval[start of "Normalize"(C.onset), end of "Normalize"(C.onset)]
`
	diags, err := NewEngine().Check(src)
	if err != nil {
		t.Fatalf("checking: %v", err)
	}
	if errs := diags.Errors(); len(errs) != 0 {
		t.Errorf("want no findings, got %v", errs)
	}
}

// A choice that is only sometimes an interval is still a mistake at an operator
// that always needs one.
func TestCheckStillRejectsANonIntervalOperand(t *testing.T) {
	for _, expr := range []string{
		`start of 5`,
		`start of 'text'`,
		`start of First([Encounter]).status`,
	} {
		src := conversionPreamble + "define A: " + expr + "\n"
		diags, err := NewEngine().Check(src)
		if err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
		if len(diags.Errors()) == 0 {
			t.Errorf("%s: want a finding, got none", expr)
		}
	}
}
