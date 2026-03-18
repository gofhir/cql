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
