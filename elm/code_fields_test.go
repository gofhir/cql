package elm

import (
	"testing"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/compiler"
)

// TestCodeFieldsSurviveTheRoundTrip takes the display and the version of
// codes, concepts and code systems from CQL through ELM JSON and back: a
// clause that says the empty String keeps it, and one left out is null (plan
// 004).
func TestCodeFieldsSurviveTheRoundTrip(t *testing.T) {
	lib, err := compiler.Compile(`library T version '1.0'
codesystem "E": 'http://e' version ''
codesystem "N": 'http://n'
code "CE": '1' from "E" display ''
code "CN": '1' from "N"
concept "KE": { "CE" } display ''
concept "KN": { "CN" }
define SE: Code '1' from "E" display ''
define SN: Code '1' from "N"
define QE: Concept { Code '1' from "E" } display ''
define QN: Concept { Code '1' from "N" }
`)
	if err != nil {
		t.Fatal(err)
	}
	data, err := MarshalLibrary(Translate(lib))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := UnmarshalLibrary(data)
	if err != nil {
		t.Fatal(err)
	}
	back, err := Import(decoded)
	if err != nil {
		t.Fatal(err)
	}
	check := func(what string, got *string, empty bool) {
		t.Helper()
		switch {
		case empty && (got == nil || *got != ""):
			t.Errorf("%s: got %v, want the empty String", what, got)
		case !empty && got != nil:
			t.Errorf("%s: got %q, want null", what, *got)
		}
	}
	systems := map[string]*ast.CodeSystemDef{}
	for _, cs := range back.CodeSystems {
		systems[cs.Name] = cs
	}
	check("codesystem E version", systems["E"].Version, true)
	check("codesystem N version", systems["N"].Version, false)
	codes := map[string]*ast.CodeDef{}
	for _, c := range back.Codes {
		codes[c.Name] = c
	}
	check("code CE display", codes["CE"].Display, true)
	check("code CN display", codes["CN"].Display, false)
	concepts := map[string]*ast.ConceptDef{}
	for _, c := range back.Concepts {
		concepts[c.Name] = c
	}
	check("concept KE display", concepts["KE"].Display, true)
	check("concept KN display", concepts["KN"].Display, false)
	defs := map[string]ast.Expression{}
	for _, d := range back.Statements {
		defs[d.Name] = d.Expression
	}
	check("selector SE display", defs["SE"].(*ast.CodeExpression).Display, true)
	check("selector SN display", defs["SN"].(*ast.CodeExpression).Display, false)
	check("selector QE display", defs["QE"].(*ast.ConceptExpression).Display, true)
	check("selector QN display", defs["QN"].(*ast.ConceptExpression).Display, false)
}
