package cql

import (
	"strings"
	"testing"
)

const qualifiedPreamble = `library T version '1.0'
using FHIR version '4.0.1'

context Patient

`

// TestQualifiedTypeNamesHaveOneSpelling covers a written type name and the same
// name arriving from the model document, which have to compare as one type.
//
// There is one canonical split, the model qualifier from the rest:
// FHIR.Encounter.Hospitalization is model FHIR and name Encounter.Hospitalization,
// because FHIR names a backbone element after the type that owns it. The parser
// reports the last segment as the name and everything before it as the
// namespace, which for a nested type put the dot in the wrong place — model
// "FHIR.Encounter", name "Hospitalization". That prints identically to what the
// document produces and compared as a different type, so a parameter declared as
// exactly the argument's type was rejected.
func TestQualifiedTypeNamesHaveOneSpelling(t *testing.T) {
	for _, tt := range []struct{ name, src string }{
		{
			"a nested type, model qualifier written",
			`define function "H"(h FHIR.Encounter.Hospitalization) returns Boolean: h is not null
define A: H(First([Encounter]).hospitalization)
`,
		},
		{
			// Without the qualifier the whole name belongs to the model in
			// force: Encounter.Hospitalization is not a Hospitalization in a
			// model called Encounter.
			"a nested type, model qualifier omitted",
			`define function "H"(h Encounter.Hospitalization) returns Boolean: h is not null
define A: H(First([Encounter]).hospitalization)
`,
		},
		{
			"another nested type, through a list",
			`define function "C"(c Observation.Component) returns Boolean: c is not null
define A: C(First(First([Observation]).component))
`,
		},
		{
			// One dot, which always worked, and has to keep working.
			"a plain model type",
			`define function "P"(p FHIR.Period) returns Boolean: p is not null
define A: P(First([Encounter]).period)
`,
		},
		{
			"is and as on a nested type, both spellings",
			`define A: First([Encounter]).hospitalization is FHIR.Encounter.Hospitalization
define B: First([Encounter]).hospitalization as Encounter.Hospitalization
`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			diags, err := NewEngine().Check(qualifiedPreamble + tt.src)
			if err != nil {
				t.Fatalf("the library should parse: %v", err)
			}
			if errs := diags.Errors(); len(errs) != 0 {
				t.Errorf("want no findings, got %v", errs)
			}
		})
	}
}

// The System model keeps its own spelling, qualified or not.
func TestSystemTypeNamesStillResolve(t *testing.T) {
	for _, src := range []string{
		"library T version '1.0'\ndefine function \"Q\"(q System.Quantity) returns Boolean: q is not null\ndefine A: Q(1 'mg')\n",
		"library T version '1.0'\ndefine function \"I\"(i Integer) returns Integer: i + 1\ndefine A: I(1)\n",
		"library T version '1.0'\ndefine function \"S\"(s System.String) returns Boolean: s is not null\ndefine A: S('x')\n",
	} {
		diags, err := NewEngine().Check(src)
		if err != nil {
			t.Fatalf("checking: %v", err)
		}
		if errs := diags.Errors(); len(errs) != 0 {
			t.Errorf("want no findings, got %v", errs)
		}
	}
}

// One spelling must not mean everything matches: a type the model does not
// declare, and an argument of the wrong type, are both still reported.
func TestQualifiedTypeNamesStillRejectMismatches(t *testing.T) {
	for _, tt := range []struct{ name, src, want string }{
		{
			"a nested type the model does not declare",
			`define function "X"(x FHIR.Encounter.NotAThing) returns Boolean: x is not null
define A: X(First([Encounter]).hospitalization)
`,
			"no overload of X",
		},
		{
			"the wrong argument for a nested parameter type",
			`define function "H"(h FHIR.Encounter.Hospitalization) returns Boolean: h is not null
define A: H(First([Encounter]).period)
`,
			"no overload of H",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			diags, err := NewEngine().Check(qualifiedPreamble + tt.src)
			if err != nil {
				t.Fatalf("checking: %v", err)
			}
			errs := diags.Errors()
			if len(errs) == 0 {
				t.Fatalf("want a finding mentioning %q, got none", tt.want)
			}
			if !strings.Contains(errs[0].Message, tt.want) {
				t.Errorf("got %q, want it to mention %q", errs[0].Message, tt.want)
			}
		})
	}
}
