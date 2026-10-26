package cql

import (
	"context"
	"testing"
)

// TestALeapSecondReadFromFHIRDataIsNoCQLValue holds a FHIR dateTime and time
// written with a second of 60 to what this engine answered before fhirpath
// v1.16.6: the text it was written as, which no CQL conversion turns into a
// DateTime or a Time.
//
// A FHIR time cannot be read as a Time at all, with a 60 or without (backlog
// 1c22 of docs/plans/README.md), so only the dateTime is held here.
//
// FHIR allows the leap second ("([0-5][0-9]|60)", R4 Data Types, dateTime and
// time), and CQL has no place for one: "ss - A full two digit second value
// (00..59)" (CQL Reference, formatting strings). fhirpath reads it from
// v1.16.6, as FHIRPath 3.0.0 has it. cql-execution 3.3.2 answers null for every
// System operation below, as this engine did; the Java engine moves the value
// to 2017-01-01T00:00:00Z, the second after (docs/plans/008-fhirpath-1-16.md,
// alcance 4).
func TestALeapSecondReadFromFHIRDataIsNoCQLValue(t *testing.T) {
	src := `library Leap version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
context Patient
define Instant: (Patient.deceased as FHIR.dateTime) > @2016-12-31T23:59:59Z
define Year: year from (Patient.deceased as FHIR.dateTime)
define Second: second from (Patient.deceased as FHIR.dateTime)
define Later: (Patient.deceased as FHIR.dateTime) + 1 second
define Text: ToString(Patient.deceased as FHIR.dateTime)
`
	e := NewEngine()
	lib, err := e.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		deceased string
		want     map[string]string
	}{
		{"2016-12-31T23:59:60Z", map[string]string{"Instant": "<nil>", "Year": "<nil>", "Second": "<nil>", "Later": "<nil>", "Text": "<nil>"}},
		// The second before is a CQL value, so the test is of the 60 alone.
		{"2016-12-31T23:59:59Z", map[string]string{"Instant": "false", "Year": "2016", "Second": "59", "Later": "2017-01-01T00:00:00Z"}},
	} {
		patient := []byte(`{"resourceType":"Patient","id":"p","deceasedDateTime":"` + tt.deceased + `"}`)
		for name, want := range tt.want {
			v, err := e.EvaluateParsedExpression(context.Background(), lib, name, patient, nil)
			if err != nil {
				t.Errorf("%s, %s: %v", tt.deceased, name, err)
				continue
			}
			got := "<nil>"
			if v != nil {
				got = v.String()
			}
			if got != want {
				t.Errorf("%s, %s = %s; want %s", tt.deceased, name, got, want)
			}
		}
	}
}
