package compiler

import (
	"testing"

	"github.com/gofhir/cql/ast"
)

func TestCompile_EmptyLibrary(t *testing.T) {
	lib, err := Compile(`library Test version '1.0.0'
using FHIR version '4.0.1'
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lib == nil {
		t.Fatal("expected library, got nil")
	}
	if lib.Identifier == nil || lib.Identifier.Name != "Test" {
		t.Errorf("expected library name 'Test', got %v", lib.Identifier)
	}
	if lib.Identifier.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", lib.Identifier.Version)
	}
}

func TestCompile_UsingDef(t *testing.T) {
	lib, err := Compile(`library Test
using FHIR version '4.0.1'
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lib.Usings) != 1 {
		t.Fatalf("expected 1 using, got %d", len(lib.Usings))
	}
	if lib.Usings[0].Name != "FHIR" {
		t.Errorf("expected using FHIR, got %s", lib.Usings[0].Name)
	}
	if lib.Usings[0].Version != "4.0.1" {
		t.Errorf("expected version '4.0.1', got %s", lib.Usings[0].Version)
	}
}

func TestCompile_ValueSetDef(t *testing.T) {
	lib, err := Compile(`library Test
using FHIR version '4.0.1'
valueset "Diabetes": 'http://example.org/fhir/ValueSet/diabetes'
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lib.ValueSets) != 1 {
		t.Fatalf("expected 1 valueset, got %d", len(lib.ValueSets))
	}
	if lib.ValueSets[0].Name != "Diabetes" {
		t.Errorf("expected valueset name 'Diabetes', got %s", lib.ValueSets[0].Name)
	}
	if lib.ValueSets[0].ID != "http://example.org/fhir/ValueSet/diabetes" {
		t.Errorf("unexpected valueset ID: %s", lib.ValueSets[0].ID)
	}
}

func TestCompile_DefineStatement(t *testing.T) {
	lib, err := Compile(`library Test
using FHIR version '4.0.1'
context Patient

define "Is Adult":
  AgeInYears() >= 18
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lib.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(lib.Statements))
	}
	if lib.Statements[0].Name != "Is Adult" {
		t.Errorf("expected statement name 'Is Adult', got %s", lib.Statements[0].Name)
	}
}

func TestCompile_LiteralExpressions(t *testing.T) {
	lib, err := Compile(`library Test
using FHIR version '4.0.1'

define "IntVal": 42
define "BoolVal": true
define "StringVal": 'hello'
define "NullVal": null
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lib.Statements) != 4 {
		t.Fatalf("expected 4 statements, got %d", len(lib.Statements))
	}
	// Check integer literal
	if lit, ok := lib.Statements[0].Expression.(*ast.Literal); ok {
		if lit.ValueType != ast.LiteralInteger || lit.Value != "42" {
			t.Errorf("expected integer 42, got type=%d value=%s", lit.ValueType, lit.Value)
		}
	} else {
		t.Errorf("expected Literal, got %T", lib.Statements[0].Expression)
	}
}

func TestCompile_IfThenElse(t *testing.T) {
	lib, err := Compile(`library Test
using FHIR version '4.0.1'

define "Result":
  if true then 1 else 0
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lib.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(lib.Statements))
	}
	if _, ok := lib.Statements[0].Expression.(*ast.IfThenElse); !ok {
		t.Errorf("expected IfThenElse, got %T", lib.Statements[0].Expression)
	}
}

func TestCompile_SyntaxError(t *testing.T) {
	_, err := Compile(`this is not valid CQL`)
	if err == nil {
		t.Fatal("expected error for invalid CQL")
	}
}

func TestCompile_RetrieveExpression(t *testing.T) {
	lib, err := Compile(`library Test
using FHIR version '4.0.1'
context Patient

define "Conditions":
  [Condition]
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lib.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(lib.Statements))
	}
	ret, ok := lib.Statements[0].Expression.(*ast.Retrieve)
	if !ok {
		t.Fatalf("expected Retrieve, got %T", lib.Statements[0].Expression)
	}
	if ret.ResourceType == nil || ret.ResourceType.Name != "Condition" {
		t.Errorf("expected Condition retrieve, got %v", ret.ResourceType)
	}
}

// TestTerminologyIdentifiersAreUndelimited covers the names that index the
// terminology tables. They used to be read with the whole node's GetText, which
// keeps the quotes, so `[Condition: "VS"]` never matched a value set declared as
// "VS" and the raw string reached the data provider instead of the URL.
func TestTerminologyIdentifiersAreUndelimited(t *testing.T) {
	src := `library T version '1.0'
using FHIR version '4.0.1'
codesystem "LOINC": 'http://loinc.org'
codesystem "cs.dotted": 'http://dotted'
valueset "Diabetes": 'http://example.org/vs/dm'
code "SBP": '8480-6' from "LOINC"
code "DOT": 'x' from "cs.dotted"
concept "BP": { "SBP", "DOT" }

define A: [Condition: "Diabetes"]
`
	lib, err := Compile(src)
	if err != nil {
		t.Fatalf("compiling: %v", err)
	}

	wantSystems := map[string]string{"SBP": "LOINC", "DOT": "cs.dotted"}
	for _, cd := range lib.Codes {
		want, ok := wantSystems[cd.Name]
		if !ok {
			t.Errorf("unexpected code %q", cd.Name)
			continue
		}
		if cd.System != want {
			t.Errorf("code %q system = %q, want %q", cd.Name, cd.System, want)
		}
	}

	if len(lib.Concepts) != 1 {
		t.Fatalf("expected 1 concept, got %d", len(lib.Concepts))
	}
	if got, want := lib.Concepts[0].Codes, []string{"SBP", "DOT"}; len(got) != len(want) {
		t.Fatalf("concept codes = %v, want %v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("concept code %d = %q, want %q", i, got[i], want[i])
			}
		}
	}

	// The retrieve's terminology reference.
	retrieve, ok := lib.Statements[0].Expression.(*ast.Retrieve)
	if !ok {
		t.Fatalf("expected the statement to be a Retrieve, got %T", lib.Statements[0].Expression)
	}
	ref, ok := retrieve.Codes.(*ast.IdentifierRef)
	if !ok {
		t.Fatalf("expected the terminology to be an IdentifierRef, got %T", retrieve.Codes)
	}
	if ref.Name != "Diabetes" {
		t.Errorf("retrieve terminology name = %q, want %q", ref.Name, "Diabetes")
	}
}
