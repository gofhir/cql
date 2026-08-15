package sema_test

import (
	"strings"
	"testing"

	"github.com/gofhir/cql/compiler"
	"github.com/gofhir/cql/fhirhelpers"
	"github.com/gofhir/cql/model"
	"github.com/gofhir/cql/sema"
)

// check compiles a library and runs the semantic phase over it against the
// official R4 model.
func check(t *testing.T, source string) *sema.Result {
	t.Helper()
	lib, err := compiler.Compile(source)
	if err != nil {
		t.Fatalf("compiling: %v", err)
	}
	mi, err := model.LoadR4ModelInfo()
	if err != nil {
		t.Fatalf("loading model info: %v", err)
	}
	return sema.Check(lib, sema.FromModelInfo(mi))
}

// preamble is the header a library needs before it can retrieve anything.
const preamble = `library Test version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
context Patient
`

func TestInfersTypesWithoutEvaluating(t *testing.T) {
	result := check(t, preamble+`
define Sum: 1 + 2
define Mixed: 1 + 2.5
define Divided: 3 / 2
define Text: 'a' + 'b'
define Truth: 1 < 2
define Encounters: [Encounter]
define Enc: First([Encounter])
define Period: Enc.period
define Numbers: { 1, 2, 3 }
define Range: Interval[1, 5]
define Pair: Tuple { a: 1, b: 'two' }
`)

	want := map[string]string{
		"Sum":        "System.Integer",
		"Mixed":      "System.Decimal",
		"Divided":    "System.Decimal",
		"Text":       "System.String",
		"Truth":      "System.Boolean",
		"Encounters": "List<FHIR.Encounter>",
		"Enc":        "FHIR.Encounter",
		"Period":     "FHIR.Period",
		"Numbers":    "List<System.Integer>",
		"Range":      "Interval<System.Integer>",
		"Pair":       "Tuple{a System.Integer, b System.String}",
	}
	for name, expected := range want {
		got, ok := result.Defines[name]
		if !ok {
			t.Errorf("%s: not typed at all", name)
			continue
		}
		if got.String() != expected {
			t.Errorf("%s is %s, want %s", name, got, expected)
		}
	}
	if errs := result.Diagnostics.Errors(); len(errs) > 0 {
		t.Errorf("nothing should have been reported, got:\n%s", errs.Error())
	}
}

// TestStartOfPeriodIsADateTime covers the divergence that motivated the
// semantic phase.
//
// `start of Enc.period` answers a Date at evaluation, because the value in hand
// carries no time and nothing says it should. The reference translator answers
// DateTime, because it knows FHIR.Period converts to Interval<System.DateTime>
// — a static fact about the model, not about the data. That difference is not
// cosmetic: comparing a Date against a DateTime brings in the uncertainty rules
// and a measure's answer changes with it.
func TestStartOfPeriodIsADateTime(t *testing.T) {
	result := check(t, preamble+`
define Enc: First([Encounter])
define StartOf: start of Enc.period
`)

	got := result.Defines["StartOf"]
	if got == nil || got.String() != "System.DateTime" {
		t.Errorf("start of a Period is %v, want System.DateTime", got)
	}
	if errs := result.Diagnostics.Errors(); len(errs) > 0 {
		t.Errorf("unexpected diagnostics:\n%s", errs.Error())
	}
}

// TestRecordsWhereAConversionBelongs covers the other half of that: knowing the
// type is what says a FHIRHelpers call has to happen, and where.
func TestRecordsWhereAConversionBelongs(t *testing.T) {
	result := check(t, preamble+`
define Enc: First([Encounter])
define StartOf: start of Enc.period
`)

	functions := make([]string, 0, len(result.Conversions))
	for _, conv := range result.Conversions {
		functions = append(functions, conv.Function)
	}
	if len(functions) != 1 || functions[0] != "FHIRHelpers.ToInterval" {
		t.Errorf("conversions inserted: %v, want one FHIRHelpers.ToInterval", functions)
	}
}

// TestReportsEverythingInOnePass is the recovery the stage is about: three
// mistakes have to come back as three diagnostics, not as the first one three
// times over.
func TestReportsEverythingInOnePass(t *testing.T) {
	result := check(t, preamble+`
define A: Missing + 1
define B: AlsoMissing
define C: First([Encounter]).notAnElement
`)

	errs := result.Diagnostics.Errors()
	if len(errs) != 3 {
		t.Fatalf("want three diagnostics, got %d:\n%s", len(errs), errs.Error())
	}
	for _, diag := range errs {
		if !diag.Position.Known() {
			t.Errorf("no position on %q", diag.Message)
		}
	}
	if errs[0].Define != "A" {
		t.Errorf("first diagnostic is in %q, want A", errs[0].Define)
	}
}

// TestOneUnknownReportsOnce covers the other side of recovery: an unresolved
// name that is used five times is still one mistake.
func TestOneUnknownReportsOnce(t *testing.T) {
	result := check(t, preamble+`
define A: Missing + Missing * 2 - 1
`)

	if errs := result.Diagnostics.Errors(); len(errs) != 2 {
		// Two mentions of the name, one diagnostic each; what must not happen
		// is a third from the arithmetic that could not be typed.
		t.Errorf("want one diagnostic per mention, got %d:\n%s", len(errs), errs.Error())
	}
}

func TestQueryTypes(t *testing.T) {
	result := check(t, preamble+`
define Finished: [Encounter] E where E.status = 'finished' return E.period
define Aliases: from [Encounter] E, [Condition] C return { e: E, c: C }
define Counted: Count([Encounter])
define Sorted: [Encounter] E sort by start of period
`)

	if got := result.Defines["Finished"]; got == nil || got.String() != "List<FHIR.Period>" {
		t.Errorf("query returning a period is %v, want List<FHIR.Period>", got)
	}
	if got := result.Defines["Counted"]; got == nil || got.String() != "System.Integer" {
		t.Errorf("Count is %v, want System.Integer", got)
	}
	if got := result.Defines["Aliases"]; got == nil ||
		got.String() != "List<Tuple{c FHIR.Condition, e FHIR.Encounter}>" {
		t.Errorf("multi-source query is %v", got)
	}
	if errs := result.Diagnostics.Errors(); len(errs) > 0 {
		t.Errorf("unexpected diagnostics:\n%s", errs.Error())
	}
}

// TestChoiceElementsStayChoices covers FHIR's [x] elements: Observation.value
// is eleven types until someone narrows it.
func TestChoiceElementsStayChoices(t *testing.T) {
	result := check(t, preamble+`
define Obs: First([Observation])
define Value: Obs.value
define Narrowed: Obs.value as FHIR.Quantity
`)

	value := result.Defines["Value"]
	if value == nil || !strings.HasPrefix(value.String(), "Choice<") {
		t.Fatalf("Observation.value is %v, want a choice", value)
	}
	if !strings.Contains(value.String(), "FHIR.Quantity") {
		t.Errorf("the choice does not include FHIR.Quantity: %s", value)
	}
	if got := result.Defines["Narrowed"]; got == nil || got.String() != "FHIR.Quantity" {
		t.Errorf("narrowing with `as` gives %v, want FHIR.Quantity", got)
	}
}

// TestSelfReferenceIsReportedNotFollowed covers what a naive resolver does with
// a definition that mentions itself: recurse until the stack runs out.
func TestSelfReferenceIsReportedNotFollowed(t *testing.T) {
	result := check(t, preamble+`
define Loop: Loop + 1
`)

	errs := result.Diagnostics.Errors()
	if len(errs) == 0 || !strings.Contains(errs[0].Message, "itself") {
		t.Errorf("want a self-reference diagnostic, got:\n%s", errs.Error())
	}
}

func TestDeclaredReturnTypeIsChecked(t *testing.T) {
	result := check(t, preamble+`
define function Wrong(x Integer) returns String: x + 1
define function Right(x Integer) returns Integer: x + 1
`)

	errs := result.Diagnostics.Errors()
	if len(errs) != 1 {
		t.Fatalf("want one diagnostic, got %d:\n%s", len(errs), errs.Error())
	}
	if !strings.Contains(errs[0].Message, "Wrong") {
		t.Errorf("the diagnostic blames the wrong function: %s", errs[0].Message)
	}
}

// TestOverloadsResolveByCost covers what the plan asks of overload resolution:
// an exact match must win over one reached by conversion.
func TestOverloadsResolveByCost(t *testing.T) {
	result := check(t, preamble+`
define function F(x Integer) returns String: 'integer'
define function F(x Decimal) returns Boolean: true
define Exact: F(1)
define Converted: F(1.5)
`)

	if got := result.Defines["Exact"]; got == nil || got.String() != "System.String" {
		t.Errorf("F(1) resolved to the %v overload, want the Integer one", got)
	}
	if got := result.Defines["Converted"]; got == nil || got.String() != "System.Boolean" {
		t.Errorf("F(1.5) resolved to the %v overload, want the Decimal one", got)
	}
}

func TestUnknownFunctionIsAWarningNotAnError(t *testing.T) {
	result := check(t, preamble+`
define A: SomethingNobodyDefined(1)
`)

	if result.Diagnostics.HasErrors() {
		t.Errorf("an unknown function should not make a library wrong:\n%s", result.Diagnostics.Error())
	}
	if len(result.Diagnostics) == 0 {
		t.Error("an unknown function should still be mentioned")
	}
}

// TestWithoutAModel covers a library that uses no data model at all: the phase
// has to work on arithmetic and intervals with nothing to ask.
func TestWithoutAModel(t *testing.T) {
	lib, err := compiler.Compile(`library Bare version '1.0'
define Sum: 1 + 2
define Window: Interval[@2020-01-01, @2020-12-31]
define Width: end of Window
`)
	if err != nil {
		t.Fatalf("compiling: %v", err)
	}
	result := sema.Check(lib, nil)

	if got := result.Defines["Sum"]; got == nil || got.String() != "System.Integer" {
		t.Errorf("Sum is %v", got)
	}
	if got := result.Defines["Width"]; got == nil || got.String() != "System.Date" {
		t.Errorf("end of a date interval is %v, want System.Date", got)
	}
	if result.Diagnostics.HasErrors() {
		t.Errorf("unexpected diagnostics:\n%s", result.Diagnostics.Error())
	}
}

func TestParametersAndTerminology(t *testing.T) {
	result := check(t, `library Test version '1.0'
using FHIR version '4.0.1'
codesystem SNOMED: 'http://snomed.info/sct'
valueset Diabetes: 'urn:oid:1.2.3'
code Fever: '386661006' from SNOMED
parameter MP Interval<DateTime> default Interval[@2020-01-01, @2020-12-31]
parameter Threshold default 5
context Patient
define Window: MP
define Limit: Threshold + 1
define VS: Diabetes
define InRange: start of MP
`)

	want := map[string]string{
		"Window":  "Interval<System.DateTime>",
		"Limit":   "System.Integer",
		"VS":      "System.ValueSet",
		"InRange": "System.DateTime",
	}
	for name, expected := range want {
		got := result.Defines[name]
		if got == nil || got.String() != expected {
			t.Errorf("%s is %v, want %s", name, got, expected)
		}
	}
	if result.Diagnostics.HasErrors() {
		t.Errorf("unexpected diagnostics:\n%s", result.Diagnostics.Error())
	}
}

// TestTheOfficialFHIRHelpersChecksCleanly runs the phase over the library the
// engine embeds: 297 functions of real CQL, written by the people who define
// the language.
//
// It is the hardest thing in the repository to type. It narrows FHIR choice
// elements with `as`, builds System.Concept and System.ValueSet instances by
// hand, and returns null from one branch of nearly every function. Each of
// those found a rule that was wrong here — that `null` in one branch made the
// whole expression untyped, that System.ValueSet has an `id`, and that an
// unqualified `Quantity` in a library using FHIR means FHIR's, not CQL's.
func TestTheOfficialFHIRHelpersChecksCleanly(t *testing.T) {
	lib, err := compiler.Compile(fhirhelpers.Source)
	if err != nil {
		t.Fatalf("compiling FHIRHelpers: %v", err)
	}
	mi, err := model.LoadR4ModelInfo()
	if err != nil {
		t.Fatalf("loading model info: %v", err)
	}

	result := sema.Check(lib, sema.FromModelInfo(mi))
	if len(result.Diagnostics) > 0 {
		t.Errorf("the official library should check cleanly, and reported:\n%s",
			result.Diagnostics.Error())
	}
	if len(lib.Functions) < 250 {
		t.Errorf("only %d functions parsed; this is meant to cover all 297", len(lib.Functions))
	}
}
