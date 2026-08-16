package cql

import (
	"os"
	"strings"
	"testing"
)

// checkDiags runs the semantic phase and fails the test if the source does not
// even parse, since that is a different problem from the one under test.
func checkDiags(t *testing.T, src string) []string {
	t.Helper()
	diags, err := NewEngine().Check(src)
	if err != nil {
		t.Fatalf("the library should parse: %v", err)
	}
	errs := diags.Errors()
	msgs := make([]string, 0, len(errs))
	for _, d := range errs {
		msgs = append(msgs, d.Message)
	}
	return msgs
}

// TestCheckAcceptsIncludedLibraryAliases covers what made the semantic phase
// unusable on a real measure. It is given one library, not the graph it sits
// in, so it cannot know what FH.ToString returns — but reporting the alias as
// undefined condemned every call into every included library, which is most of
// the calls a measure makes.
func TestCheckAcceptsIncludedLibraryAliases(t *testing.T) {
	for _, tt := range []struct{ name, src string }{
		{
			"aliased with called",
			`library M version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
context Patient
define S: FH.ToString(First([Encounter]).status)
`,
		},
		{
			// Without a `called` clause the library is referred to by its own
			// name.
			"referred to by its name",
			`library M version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
context Patient
define S: FHIRHelpers.ToString(First([Encounter]).status)
`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if msgs := checkDiags(t, tt.src); len(msgs) != 0 {
				t.Errorf("want no findings, got %v", msgs)
			}
		})
	}
}

// TestCheckResolvesChoiceElementsByTheirConcreteName covers the other half of
// the same problem. The model declares Observation.value[x] as `value`, while
// both the JSON and the CQL that reads it say `valueQuantity`, so every access
// to a choice element was reported as an element the type does not have.
func TestCheckResolvesChoiceElementsByTheirConcreteName(t *testing.T) {
	for _, expr := range []string{
		`First([Observation]).valueQuantity`,
		`First([Observation]).valueString`,
		`First([Observation]).effectivePeriod`,

		// FHIR writes its primitives lower case — FHIR.dateTime behind the
		// effectiveDateTime that reads it — so the match ignores case.
		`First([Observation]).effectiveDateTime`,
		`First([Condition]).onsetDateTime`,
	} {
		src := "library M version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\ndefine A: " + expr + "\n"
		if msgs := checkDiags(t, src); len(msgs) != 0 {
			t.Errorf("%s: want no findings, got %v", expr, msgs)
		}
	}
}

// The concessions above must not swallow the mistakes the phase exists to
// find: a name that is not an alias is still undefined, and a suffix that no
// branch of the choice carries is still wrong.
func TestCheckStillReportsRealMistakes(t *testing.T) {
	for _, tt := range []struct{ src, want string }{
		{
			`library M version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
context Patient
define S: NotAnAlias.ToString('x')
`,
			"NotAnAlias is not defined",
		},
		{
			`library M version '1.0'
using FHIR version '4.0.1'
context Patient
define A: First([Observation]).valueNotAThing
`,
			"has no element valueNotAThing",
		},
		{
			`library M version '1.0'
using FHIR version '4.0.1'
context Patient
define A: First([Encounter]).notAnElement
`,
			"has no element notAnElement",
		},
	} {
		msgs := checkDiags(t, tt.src)
		if len(msgs) == 0 {
			t.Errorf("want a finding mentioning %q, got none", tt.want)
			continue
		}
		if !strings.Contains(strings.Join(msgs, "\n"), tt.want) {
			t.Errorf("want a finding mentioning %q, got %v", tt.want, msgs)
		}
	}
}

// A whole measure-shaped library — includes, choice elements and a query —
// comes back clean, which is the bar the phase has to clear before anything
// can be built on what it reports.
func TestCheckIsQuietOnAMeasureShapedLibrary(t *testing.T) {
	src := `library M version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
parameter "Measurement Period" Interval<DateTime>
context Patient

define InPeriod: [Encounter] E where E.period during "Measurement Period"
define Statuses: [Encounter] E return FH.ToString(E.status)
define Values: [Observation] O return O.valueQuantity
define Denominator: exists InPeriod
`
	if msgs := checkDiags(t, src); len(msgs) != 0 {
		t.Errorf("want no findings, got %v", msgs)
	}
}

// The official FHIRHelpers is the largest real library on hand, and it has to
// stay silent — it is included by nearly every measure, so a finding here would
// reach every caller.
func TestCheckIsQuietOnOfficialFHIRHelpers(t *testing.T) {
	raw, err := os.ReadFile("fhirhelpers/FHIRHelpers-4.0.1.cql")
	if err != nil {
		t.Fatalf("reading FHIRHelpers: %v", err)
	}
	if msgs := checkDiags(t, string(raw)); len(msgs) != 0 {
		t.Errorf("want no findings, got %d: %v", len(msgs), msgs)
	}
}
