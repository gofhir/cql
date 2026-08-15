package cql

import (
	"strings"
	"testing"
)

// TestCheckFindsMistakesWithoutData covers what the semantic phase is for: a
// library with three mistakes in it comes back with three findings, from a call
// that touched no provider and evaluated nothing.
func TestCheckFindsMistakesWithoutData(t *testing.T) {
	engine := NewEngine()

	diags, err := engine.Check(`library Broken version '1.0'
using FHIR version '4.0.1'
context Patient
define Enc: First([Encounter])
define A: Enc.notAnElement
define B: Undefined + 1
define C: 'text' + 1
`)
	if err != nil {
		t.Fatalf("the library parses; Check should not have failed: %v", err)
	}

	errs := diags.Errors()
	if len(errs) != 3 {
		t.Fatalf("want three findings, got %d:\n%s", len(errs), diags.Error())
	}
	for _, diag := range errs {
		if !diag.Position.Known() {
			t.Errorf("no position on %q", diag.Message)
		}
		if diag.Define == "" {
			t.Errorf("no definition named on %q", diag.Message)
		}
	}
}

// TestCheckAcceptsAWorkingLibrary is the other half: a library the engine
// evaluates has to pass the phase, or the phase is a source of false alarms.
func TestCheckAcceptsAWorkingLibrary(t *testing.T) {
	engine := NewEngine()

	diags, err := engine.Check(`library Good version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
parameter MP Interval<DateTime> default Interval[@2020-01-01, @2020-12-31]
context Patient
define Encounters: [Encounter]
define Finished: Encounters E where E.status = 'finished'
define InPeriod: Encounters E where E.period during MP
define Ages: AgeInYearsAt(start of MP)
define Count: Count(Finished)
`)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if diags.HasErrors() {
		t.Errorf("a working library should check cleanly:\n%s", diags.Errors().Error())
	}
}

// TestCheckWithoutParsingTwice covers CheckParsed, which is the same pass over
// a library the caller already paid to parse.
func TestCheckWithoutParsingTwice(t *testing.T) {
	engine := NewEngine()

	lib, err := engine.Parse(`library Reused version '1.0'
define A: Missing
`)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	diags := engine.CheckParsed(lib)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "Missing") {
		t.Errorf("want one finding about Missing, got:\n%s", diags.Error())
	}
	if engine.CheckParsed(nil) != nil {
		t.Error("checking nothing should find nothing")
	}
}

// TestCheckDoesNotRefuseOnSyntax covers the boundary between the two failures a
// caller has to tell apart: a library that does not parse is an error, and one
// that parses but does not make sense is a list of findings.
func TestCheckDoesNotRefuseOnSyntax(t *testing.T) {
	engine := NewEngine()

	if _, err := engine.Check("library Broken version '1.0'\ndefine A: ((("); err == nil {
		t.Error("a library that does not parse should come back as an error")
	}
}
