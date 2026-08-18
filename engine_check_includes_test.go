package cql

import (
	"context"
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

// TestCheckDoesNotResolveIncludedCallsLocally covers the call around the alias,
// which the alias fix alone does not reach. Only the ELM importer sets
// FunctionCall.Library; the text parser builds `C.Helper(x)` as a fluent call
// with the alias as its source, so the alias was counted as a first argument
// and the call resolved against *this* library's overloads.
//
// A local Helper(Integer, Integer) therefore made `C.Helper('x')` report an
// overload failure on valid CQL, and `C.Helper(3)` take the local function's
// return type — a wrong type that reaches the evaluator through the recorded
// conversions.
func TestCheckDoesNotResolveIncludedCallsLocally(t *testing.T) {
	const collides = `library M version '1.0'
include Common version '1.0' called C
define function Helper(a Integer, b Integer) returns Integer: a + b
define A: C.Helper(3)
define B: C.Helper('x')
`
	if msgs := checkDiags(t, collides); len(msgs) != 0 {
		t.Errorf("want no findings for calls into an included library, got %v", msgs)
	}

	// With nothing local to collide with, the bare name used to reach the
	// system function table and answer with its return type instead.
	const systemName = `library M version '1.0'
include Common version '1.0' called C
define A: C.ToString(3)
`
	if msgs := checkDiags(t, systemName); len(msgs) != 0 {
		t.Errorf("want no findings, got %v", msgs)
	}
}

// The call being unknown must not hide a mistake inside its arguments, and a
// call to a local function must still resolve its overloads.
func TestCheckStillChecksAroundIncludedCalls(t *testing.T) {
	const badArgument = `library M version '1.0'
include Common version '1.0' called C
define A: C.Helper(Bogus)
`
	if msgs := checkDiags(t, badArgument); len(msgs) != 1 ||
		!strings.Contains(msgs[0], "Bogus is not defined") {
		t.Errorf("want the undefined argument reported, got %v", msgs)
	}

	const localCall = `library M version '1.0'
define function Helper(a Integer, b Integer) returns Integer: a + b
define A: Helper(1, 2)
define B: Helper('x', 2)
`
	msgs := checkDiags(t, localCall)
	if len(msgs) != 1 || !strings.Contains(msgs[0], "no overload of Helper") {
		t.Errorf("want the local overload failure reported, got %v", msgs)
	}
}

// TestCheckResolvesChoiceElementsByTheirConcreteName covers a form this engine
// resolves and no other CQL tool does. The model declares Observation.value[x]
// as `value`, and the JSON the evaluator reads calls the field valueQuantity, so
// the name resolves here — but it is not an element the model declares, and the
// reference translator refuses it outright:
//
//	ERROR: CqlSemanticException: Member valueQuantity not found for type Observation.
//
// So it resolves and warns, which is checked in
// TestConcreteChoiceNamesWarnThatTheyAreNotPortable. This test holds the half
// that matters for evaluation: it is not an error, and a library using it runs.
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
			t.Errorf("%s: want no errors, got %v", expr, msgs)
		}
	}
}

// TestConcreteChoiceNamesWarnThatTheyAreNotPortable covers what this engine says
// about accepting a form nothing else accepts.
//
// It used to say nothing at all. The reference probe is what turned that up, and
// with it how the support got here: the semantic phase learned these names to
// stop it reporting findings that looked false, and they were not false — the
// eight it reported came from this repository's own tests. Neither the published
// eCQM libraries nor FHIRHelpers writes a choice element this way.
//
// A warning is the whole of the fix. Errors stop evaluation and this must not:
// the evaluator reads FHIR JSON, where the field really is called valueQuantity.
// What was missing was telling the author that their library will not translate
// anywhere else.
func TestConcreteChoiceNamesWarnThatTheyAreNotPortable(t *testing.T) {
	const src = `library M version '1.0'
using FHIR version '4.0.1'
context Patient
define A: First([Observation]).valueQuantity
`
	diags, err := NewEngine().Check(src)
	if err != nil {
		t.Fatalf("checking: %v", err)
	}
	if errs := diags.Errors(); len(errs) != 0 {
		t.Fatalf("want no errors, got %v", errs)
	}
	if len(diags) != 1 {
		t.Fatalf("want one warning, got %d: %v", len(diags), diags)
	}
	// The message has to carry the form that does translate, or it reports a
	// problem and leaves the reader to guess the fix.
	if !strings.Contains(diags[0].Message, "value as FHIR.Quantity") {
		t.Errorf("the warning does not say what to write instead: %s", diags[0].Message)
	}
	if !diags[0].Position.Known() {
		t.Error("the warning carries no position")
	}

	// And the form the reference accepts warns about nothing.
	clean, err := NewEngine().Check(`library M version '1.0'
using FHIR version '4.0.1'
context Patient
define A: First([Observation]).value as FHIR.Quantity
`)
	if err != nil {
		t.Fatalf("checking: %v", err)
	}
	if len(clean) != 0 {
		t.Errorf("the portable form should be silent, got %v", clean)
	}
}

// A warning must not stop evaluation, which is the difference between saying
// something is unportable and refusing to run it.
func TestConcreteChoiceNamesStillEvaluate(t *testing.T) {
	const src = `library M version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
context Patient
define A: First([Observation]).valueQuantity
`
	got, err := NewEngine(WithDataProvider(&requestRecorder{})).EvaluateExpression(
		context.Background(), src, "A", []byte(`{"resourceType":"Patient","id":"p1"}`), nil)
	if err != nil {
		t.Fatalf("a warning stopped the evaluation: %v", err)
	}
	if got == nil {
		t.Error("evaluated to null, want the quantity")
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
