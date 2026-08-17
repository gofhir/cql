package cql

import (
	"strings"
	"testing"
)

// TestCheckResolvesBackboneElements covers what published measures spend their
// time reading, and what the semantic phase reported as missing.
//
// FHIR names a backbone element after the type that owns it, so
// FHIR.Encounter.Hospitalization is one type whose name contains a dot. Two
// functions disagreed about which dot mattered: the qualifier was dropped by
// cutting at the last one, leaving "Hospitalization", and the element path was
// split at the first, looking for "Hospitalization.dischargeDisposition" on
// Encounter. Between them, every element of every backbone element was unknown.
func TestCheckResolvesBackboneElements(t *testing.T) {
	for _, expr := range []string{
		// The forms the eCQM libraries actually use.
		`First([Encounter]).hospitalization.dischargeDisposition`,
		`First([Encounter]).location`,
		`First([Observation]).component`,
		`First([MedicationRequest]).dispenseRequest.validityPeriod`,

		// Through a list, which is how a repeating backbone element reads.
		`First([Encounter]).diagnosis.condition`,
		`First([Observation]).component.code`,
		`First([Observation]).component.value`,

		// Two levels down, where the second dot is inside the type name and the
		// third is the element.
		`First([Encounter]).location.period.start`,
	} {
		src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\ndefine A: " + expr + "\n"
		diags, err := NewEngine().Check(src)
		if err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
		if errs := diags.Errors(); len(errs) != 0 {
			t.Errorf("%s: want no findings, got %v", expr, errs)
		}
	}
}

// A backbone element resolving must not make everything resolve: an element it
// does not have is still reported, and so is a backbone element that does not
// exist.
func TestCheckStillReportsMissingBackboneElements(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{
			`First([Encounter]).hospitalization.notAnElement`,
			"has no element notAnElement",
		},
		{
			`First([Encounter]).notABackboneElement.field`,
			"has no element notABackboneElement",
		},
		{
			`First([Observation]).component.notAnElement`,
			"has no element notAnElement",
		},
	} {
		src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\ndefine A: " + tt.expr + "\n"
		diags, err := NewEngine().Check(src)
		if err != nil {
			t.Fatalf("%s: %v", tt.expr, err)
		}
		errs := diags.Errors()
		if len(errs) == 0 {
			t.Errorf("%s: want a finding mentioning %q, got none", tt.expr, tt.want)
			continue
		}
		if !strings.Contains(errs[0].Message, tt.want) {
			t.Errorf("%s: got %q, want it to mention %q", tt.expr, errs[0].Message, tt.want)
		}
	}
}

// A measure-shaped library reading backbone elements comes back clean, which is
// the shape the eCQM corpus is made of: 16 of its 19 libraries report nothing
// now, against 11 before.
func TestCheckIsQuietOnBackboneElementUse(t *testing.T) {
	src := `library M version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
parameter "Measurement Period" Interval<DateTime>
context Patient

define Discharges:
  [Encounter] E
    where E.period during "Measurement Period"
    return E.hospitalization.dischargeDisposition

define Components: [Observation] O return O.component
define Diagnoses: [Encounter] E return E.diagnosis
`
	diags, err := NewEngine().Check(src)
	if err != nil {
		t.Fatalf("checking: %v", err)
	}
	if errs := diags.Errors(); len(errs) != 0 {
		t.Errorf("want no findings, got %v", errs)
	}
}
