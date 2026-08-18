package cql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofhir/cql/eval"
)

const primitivePreamble = `library T version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH

context Patient

`

// primitiveProvider serves a Period written to the day and one written to the
// second, plus a Patient whose birthDate is a FHIR date.
type primitiveProvider struct{}

func (primitiveProvider) Retrieve(_ context.Context, req eval.RetrieveRequest) ([]json.RawMessage, error) {
	switch req.ResourceType {
	case "Patient":
		return []json.RawMessage{json.RawMessage(
			`{"resourceType":"Patient","id":"p1","birthDate":"1980-05-15"}`)}, nil
	case "Encounter":
		return []json.RawMessage{json.RawMessage(
			`{"resourceType":"Encounter","id":"e1",` +
				`"period":{"start":"2020-03-01","end":"2020-03-05T10:00:00Z"}}`)}, nil
	}
	return nil, nil
}

func evalPrimitive(t *testing.T, expr string) string {
	t.Helper()
	src := primitivePreamble + "define A: " + expr + "\n"
	got, err := NewEngine(WithDataProvider(primitiveProvider{})).EvaluateExpression(
		context.Background(), src, "A", []byte(`{"resourceType":"Patient","id":"p1"}`), nil)
	if err != nil {
		t.Fatalf("%s: %v", expr, err)
	}
	return valueString(got)
}

// TestFHIRDateTimeValuesAreDateTimes covers a value whose type the engine read
// off the JSON text rather than off the model.
//
// The model says what a FHIR primitive holds: FHIR.dateTime.value is
// System.DateTime and FHIR.date.value is System.Date. That is the whole of the
// difference between FHIRHelpers' ToDate and ToDateTime, which have the same body
// — `value.value` — and differ only in the type of the parameter.
//
// In JSON both are a string, and `"2020-03-01"` was read as a Date because it
// carried no time. So Enc.period.start came back a Date although Period.start is
// declared FHIR.dateTime, and one Period could have endpoints of two different
// types.
//
// It went unnoticed because the operators convert as they go. Where it shows is
// where the type *is* the question.
func TestFHIRDateTimeValuesAreDateTimes(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`First([Encounter]).period.start is DateTime`, "true"},
		{`First([Encounter]).period.start is Date`, "false"},

		// `start of` reaches the value through FHIRHelpers.ToInterval, which is
		// CQL from another library and carries its own plan, so the element's
		// declared type is what answers here.
		{`start of First([Encounter]).period is DateTime`, "true"},
		{`end of First([Encounter]).period is DateTime`, "true"},

		// The cast used to answer null on a value that is one.
		{`(start of First([Encounter]).period as DateTime) is DateTime`, "true"},

		// Which branch a library takes turned on this.
		{`if start of First([Encounter]).period is DateTime then 'dateTime' else 'date'`, "dateTime"},
	} {
		if got := evalPrimitive(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// A FHIR date is a Date, and stays one. Patient.birthDate is the case that would
// notice if the promotion were applied to every temporal primitive.
func TestFHIRDateValuesStayDates(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`First([Patient]).birthDate is Date`, "true"},
		{`First([Patient]).birthDate is DateTime`, "false"},
		{`First([Patient]).birthDate`, "1980-05-15"},
	} {
		if got := evalPrimitive(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// The value does not change, only its type: a dateTime written to the day is a
// DateTime written to the day, and nothing invents a time it did not carry.
func TestPromotionKeepsThePrecision(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`start of First([Encounter]).period`, "2020-03-01"},
		{`end of First([Encounter]).period`, "2020-03-05T10:00:00Z"},

		// And the operators that always worked keep working.
		{`First([Encounter]).period overlaps Interval[@2020-01-01T00:00:00Z, @2020-12-31T23:59:59Z]`, "true"},
		{`First([Encounter]).period during Interval[@2020-01-01T00:00:00Z, @2020-12-31T23:59:59Z]`, "true"},
		{`start of First([Encounter]).period < end of First([Encounter]).period`, "true"},
	} {
		if got := evalPrimitive(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestDurationBecomesUncertain covers the consequence, which is the reason to
// look twice before calling this an improvement.
//
// A dateTime written to the day means the time is unknown, not midnight. Two
// indeterminate instants in different days are an uncertain number of days apart,
// and CQL says so with an interval. The conformance corpus holds the same rule
// for the same reason: `years between DateTime(2005) and DateTime(2010)` is
// Interval[4, 5], while `years between DateTime(2005, 5) and DateTime(2010, 4)`
// is exactly 4.
//
// It answered a plain number while the endpoints came back as Dates, because a
// Date *is* the day and counting days off it is exact. Reading them as the
// dateTimes the model declares moves the answer onto what the data supports.
func TestDurationBecomesUncertain(t *testing.T) {
	// One endpoint to the day, so the count of days is uncertain by one.
	if got := evalPrimitive(t, `duration in days of First([Encounter]).period`); got != "Interval[3, 4]" {
		t.Errorf("= %s, want Interval[3, 4]", got)
	}

	// A Date is still the day itself, so counting days off two of them is exact.
	src := "library T version '1.0'\ndefine A: days between @2020-03-01 and @2020-03-05\n"
	got, err := NewEngine().EvaluateExpression(context.Background(), src, "A", nil, nil)
	if err != nil {
		t.Fatalf("between two dates: %v", err)
	}
	if s := valueString(got); s != "4" {
		t.Errorf("days between two Dates = %s, want 4", s)
	}
}
