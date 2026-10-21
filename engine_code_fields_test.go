package cql

import (
	"context"
	"testing"

	"github.com/gofhir/cql/internal/referenceharness/answers"
)

// codeFieldsLibrary declares the code system, codes and concept the cases of
// TestCodeFieldsKeepNullApartFromEmpty refer to.
const codeFieldsLibrary = `library CodeFields version '1.0'
codesystem "CS": 'http://example.org/cs' version '2'
codesystem "S": 'http://s'
code "A": '1' from "CS"
code "B": '1' from "CS" display 'bee'
code "C1": 'x' from "S"
code "C2": 'x' from "S" display ''
concept "K": { "A" } display 'kay'
`

// TestCodeFieldsKeepNullApartFromEmpty holds Code and Concept to CQL 1.5
// (cql.hl7.org/09-b-cqlreference.html): Code is `structured type Code { code
// String, display String, system String, version String }`, so the empty String is a String
// and not null; Equal compares a structured type as "a conjunction of equality
// comparisons" of its elements; Equivalent compares a Code by "the code and
// system elements only" and never answers null, and a Concept by "a non-empty
// intersection of the codes".
//
// Each answer is the Java engine's 3.26.0 (probes of 21-10-2026, plan 004),
// but for the two marked: there Java answers null where the conjunction gives
// false, and the specification decides (the user's decision, plan 004).
func TestCodeFieldsKeepNullApartFromEmpty(t *testing.T) {
	const (
		null      = "null"
		yes, no   = "true", "false"
		nullCode  = `Code{code: "x", display: null, system: null, version: null}`
		emptyCode = `Code{code: "x", display: null, system: "", version: null}`
		javaWrong = "the conjunction: an element that differs settles it (Java answers null)"
	)
	engine := NewEngine()
	for _, tt := range []struct {
		name, expr, want, note string
	}{
		// '' is a String: reading a field gives what it holds.
		{"an empty system is not null", "Code { code: 'x', system: '' }.system is null", no, ""},
		{"an unset system is null", "Code { code: 'x' }.system is null", yes, ""},
		{"an empty system reads as ''", "Code { code: 'x', system: '' }.system", `""`, ""},
		{"an empty system equals ''", "Code { code: 'x', system: '' }.system = ''", yes, ""},
		{"an empty code is not null", "Code { code: '' }.code is null", no, ""},
		{"an empty Concept display is not null", "Concept { codes: { Code { code: 'x' } }, display: '' }.display is null", no, ""},
		{"a declared empty display is not null", `"C2".display is null`, no, ""},
		{"a declared unset display is null", `"C1".display is null`, yes, ""},
		{"a declared code takes its code system's version", `"A".version`, `"2"`, ""},
		{"a declared code without display has none", `"A".display`, null, ""},
		{"a declared concept keeps its display", `"K".display`, `"kay"`, ""},
		{"a Code selector takes its code system's version", `(Code '1' from "CS").version`, `"2"`, ""},
		{"a Code selector takes its code system's id", `(Code '1' from "CS").system`, `"http://example.org/cs"`, ""},
		{"a Code selector's empty display is not null", `(Code '1' from "CS" display '').display is null`, no, ""},
		{"a Code selector without display has none", `(Code '1' from "CS").display is null`, yes, ""},
		{"a Code selector equals the declared code", `Code '1' from "CS" = "A"`, yes, ""},
		{"a Concept selector's empty display is ''", `(Concept { Code '1' from "CS" } display '').display`, `""`, ""},
		{"ToConcept leaves the Concept's display null", "ToConcept(Code { code: 'x', display: 'd' }).display", null, ""},

		// Equal: a conjunction of the elements.
		{"'' against null is unknown", "Code { code: 'x', system: '' } = Code { code: 'x' }", null, ""},
		{"'' against null is unknown, negated", "Code { code: 'x', system: '' } != Code { code: 'x' }", null, ""},
		{"displays that differ", "Code { code: 'x', system: 's', display: 'a' } = Code { code: 'x', system: 's', display: 'b' }", no, ""},
		{"versions that differ", "Code { code: 'x', system: 's', version: '1' } = Code { code: 'x', system: 's', version: '2' }", no, ""},
		{"null on both sides", "Code { code: 'x' } = Code { code: 'x' }", yes, ""},
		{"a null code on both sides", "Code { code: null } = Code { code: null }", yes, ""},
		{"null against a value", "Code { code: 'x' } = Code { code: 'x', system: 's' }", null, ""},
		{"a null Code", "(null as Code) = Code { code: 'x' }", null, ""},
		{"case counts", "Code { code: 'X', system: 's' } = Code { code: 'x', system: 's' }", no, ""},
		{"a null before an element that differs", "Code { code: 'x', display: 'a', system: 's' } = Code { code: 'x', system: 't' }", no, javaWrong},
		{"an element that differs before a null", "Code { code: 'x', system: 's', display: 'a' } = Code { code: 'y', system: 's' }", no, javaWrong},
		{"a declared code against a selector without its version", `"A" = Code { code: '1', system: 'http://example.org/cs' }`, null, ""},
		{"a declared code against a selector with its version", `"A" = Code { code: '1', system: 'http://example.org/cs', version: '2' }`, yes, ""},
		{"two declared codes, one with a display", `"A" = "B"`, null, ""},
		{"lists of Codes compare by the conjunction", "{ Code { code: 'x', system: 'a' } } = { Code { code: 'x' } }", null, ""},
		{"Concept displays that differ", "Concept { codes: { Code { code: 'x' } }, display: 'a' } = Concept { codes: { Code { code: 'x' } }, display: 'b' }", no, ""},
		{"a Concept display against null", "Concept { codes: { Code { code: 'x' } } } = Concept { codes: { Code { code: 'x' } }, display: 'a' }", null, ""},
		{"a Concept display '' against null", "Concept { codes: { Code { code: 'x' } }, display: '' } = Concept { codes: { Code { code: 'x' } } }", null, ""},
		{"a Concept's codes are a list, in order", "Concept { codes: { Code { code: 'a' }, Code { code: 'b' } } } = Concept { codes: { Code { code: 'b' }, Code { code: 'a' } } }", no, ""},
		{"the same Concept", "Concept { codes: { Code { code: 'a' }, Code { code: 'b' } } } = Concept { codes: { Code { code: 'a' }, Code { code: 'b' } } }", yes, ""},

		// Equivalent: code and system only, never null.
		{"'' is not equivalent to null", "Code { code: 'x', system: '' } ~ Code { code: 'x' }", no, ""},
		{"null systems are equivalent", "Code { code: 'x' } ~ Code { code: 'x' }", yes, ""},
		{"equivalence ignores case", "Code { code: 'X', system: 's' } ~ Code { code: 'x', system: 's' }", yes, ""},
		{"equivalence does not trim", "Code { code: 'x ', system: 's' } ~ Code { code: 'x', system: 's' }", no, ""},
		{"equivalence ignores display", `"A" ~ "B"`, yes, ""},
		{"Concepts with no codes share none", "Concept { codes: {} } ~ Concept { codes: {} }", no, ""},
		{"Concepts with null codes share none", "Concept { display: 'd' } ~ Concept { display: 'd' }", no, ""},
		{"a Concept's '' system is not a null one", "Concept { codes: { Code { code: 'x', system: '' } } } ~ Concept { codes: { Code { code: 'x' } } }", no, ""},
		{"a Code reads its fields as the reference writes them", "Code { code: 'x' }", nullCode, ""},
		{"a Code writes an empty field as a String", "Code { code: 'x', system: '' }", emptyCode, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			src := codeFieldsLibrary + "define R: " + tt.expr + "\n"
			v, err := engine.EvaluateExpression(context.Background(), src, "R", nil, nil)
			if err != nil {
				t.Fatalf("%s: %v", tt.expr, err)
			}
			if got := answers.Value(v).Text; got != tt.want {
				t.Errorf("%s = %s, want %s %s", tt.expr, got, tt.want, tt.note)
			}
		})
	}
}
