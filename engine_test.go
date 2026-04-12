package cql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

func TestEngine_NewEngineDefaults(t *testing.T) {
	e := NewEngine()
	if e.maxExpressionLen != 100*1024 {
		t.Errorf("maxExpressionLen = %d, want %d", e.maxExpressionLen, 100*1024)
	}
	if e.evalTimeout != 30*time.Second {
		t.Errorf("evalTimeout = %v, want %v", e.evalTimeout, 30*time.Second)
	}
	if e.maxRetrieveSize != 10000 {
		t.Errorf("maxRetrieveSize = %d, want %d", e.maxRetrieveSize, 10000)
	}
	if e.modelInfo == nil {
		t.Error("modelInfo should not be nil (default R4)")
	}
}

func TestEngine_WithOptions(t *testing.T) {
	e := NewEngine(
		WithMaxExpressionLen(1024),
		WithTimeout(5*time.Second),
		WithMaxRetrieveSize(100),
		WithMaxDepth(50),
	)
	if e.maxExpressionLen != 1024 {
		t.Errorf("maxExpressionLen = %d, want 1024", e.maxExpressionLen)
	}
	if e.evalTimeout != 5*time.Second {
		t.Errorf("evalTimeout = %v, want 5s", e.evalTimeout)
	}
	if e.maxRetrieveSize != 100 {
		t.Errorf("maxRetrieveSize = %d, want 100", e.maxRetrieveSize)
	}
	if e.maxDepth != 50 {
		t.Errorf("maxDepth = %d, want 50", e.maxDepth)
	}
}

func TestEngine_Compile_Valid(t *testing.T) {
	e := NewEngine()
	err := e.Compile(`library Test version '1.0'
using FHIR version '4.0.1'
define X: 42`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEngine_Compile_SyntaxError(t *testing.T) {
	e := NewEngine()
	err := e.Compile("this is not valid CQL @@@ !!! {{{")
	if err == nil {
		t.Fatal("expected syntax error")
	}
	var syntaxErr *ErrSyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Errorf("expected ErrSyntaxError, got %T", err)
	}
}

func TestEngine_Compile_TooCostly(t *testing.T) {
	e := NewEngine(WithMaxExpressionLen(10))
	err := e.Compile("library Test version '1.0'\nusing FHIR version '4.0.1'")
	if err == nil {
		t.Fatal("expected too-costly error")
	}
	var tcErr *ErrTooCostly
	if !errors.As(err, &tcErr) {
		t.Errorf("expected ErrTooCostly, got %T", err)
	}
}

func TestEngine_EvaluateLibrary(t *testing.T) {
	e := NewEngine()
	cqlSource := `library Test version '1.0'
using FHIR version '4.0.1'
define X: 42
define Y: 'hello'`

	results, err := e.EvaluateLibrary(context.Background(), cqlSource, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	x := results["X"]
	if x == nil {
		t.Fatal("X should not be nil")
	}
	xi, ok := x.(fptypes.Integer)
	if !ok {
		t.Fatalf("X: expected Integer, got %T", x)
	}
	if xi.Value() != 42 {
		t.Errorf("X = %d, want 42", xi.Value())
	}

	y := results["Y"]
	if y == nil {
		t.Fatal("Y should not be nil")
	}
	ys, ok := y.(fptypes.String)
	if !ok {
		t.Fatalf("Y: expected String, got %T", y)
	}
	if ys.Value() != "hello" {
		t.Errorf("Y = %q, want %q", ys.Value(), "hello")
	}
}

func TestEngine_EvaluateExpression(t *testing.T) {
	e := NewEngine()
	cqlSource := `library Test version '1.0'
using FHIR version '4.0.1'
define Answer: 42
define Greeting: 'world'`

	val, err := e.EvaluateExpression(context.Background(), cqlSource, "Answer", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	i, ok := val.(fptypes.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", val)
	}
	if i.Value() != 42 {
		t.Errorf("Answer = %d, want 42", i.Value())
	}
}

func TestEngine_EvaluateExpression_NotFound(t *testing.T) {
	e := NewEngine()
	cqlSource := `library Test version '1.0'
using FHIR version '4.0.1'
define X: 1`

	_, err := e.EvaluateExpression(context.Background(), cqlSource, "Missing", nil, nil)
	if err == nil {
		t.Fatal("expected error for missing expression")
	}
	var evalErr *ErrEvaluation
	if !errors.As(err, &evalErr) {
		t.Errorf("expected ErrEvaluation, got %T: %v", err, err)
	}
}

func TestEngine_EvaluateLibrary_WithParameters(t *testing.T) {
	e := NewEngine()
	cqlSource := `library Test version '1.0'
using FHIR version '4.0.1'
parameter Threshold Integer
define AboveThreshold: Threshold`

	params := map[string]fptypes.Value{
		"Threshold": fptypes.NewInteger(100),
	}

	results, err := e.EvaluateLibrary(context.Background(), cqlSource, nil, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val := results["AboveThreshold"]
	if val == nil {
		t.Fatal("AboveThreshold should not be nil")
	}
	i, ok := val.(fptypes.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", val)
	}
	if i.Value() != 100 {
		t.Errorf("AboveThreshold = %d, want 100", i.Value())
	}
}

func TestEngine_EvaluateLibrary_TooCostly(t *testing.T) {
	e := NewEngine(WithMaxExpressionLen(10))
	_, err := e.EvaluateLibrary(context.Background(), "library Test version '1.0'\ndefine X: 42", nil, nil)
	if err == nil {
		t.Fatal("expected too-costly error")
	}
	var tcErr *ErrTooCostly
	if !errors.As(err, &tcErr) {
		t.Errorf("expected ErrTooCostly, got %T", err)
	}
}

func TestEngine_EvaluateLibrary_WithContext(t *testing.T) {
	e := NewEngine()
	cqlSource := `library Test version '1.0'
using FHIR version '4.0.1'
define Result: 'evaluated'`

	patient := json.RawMessage(`{"resourceType":"Patient","id":"123","birthDate":"1990-01-15"}`)
	results, err := e.EvaluateLibrary(context.Background(), cqlSource, patient, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results["Result"] == nil {
		t.Error("Result should not be nil")
	}
}

func TestEngine_EvaluateLibrary_WithIncludedLibrary(t *testing.T) {
	mathLib := `library MathHelpers version '1.0'
using FHIR version '4.0.1'
define function Double(x Integer) returns Integer: x * 2`

	resolver := func(ctx context.Context, name, version string) (string, error) {
		if name == "MathHelpers" {
			return mathLib, nil
		}
		return "", fmt.Errorf("library '%s' not found", name)
	}

	e := NewEngine(WithLibraryResolver(resolver))

	cqlSource := `library Test version '1.0'
using FHIR version '4.0.1'
include MathHelpers version '1.0'
define Result: MathHelpers.Double(21)`

	results, err := e.EvaluateLibrary(context.Background(), cqlSource, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := results["Result"].(fptypes.Integer)
	if !ok {
		t.Fatalf("Result: expected Integer, got %T (%v)", results["Result"], results["Result"])
	}
	if val.Value() != 42 {
		t.Errorf("Result = %d, want 42", val.Value())
	}
}

func TestEngine_EvaluateLibrary_IncludeWithoutResolver(t *testing.T) {
	e := NewEngine()
	cqlSource := `library Test version '1.0'
using FHIR version '4.0.1'
include SomeLib version '1.0'
define X: 1`

	_, err := e.EvaluateLibrary(context.Background(), cqlSource, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing resolver")
	}
}

func TestEngine_FHIRHelpers_BuiltIn(t *testing.T) {
	// No LibraryResolver provided — FHIRHelpers should be auto-resolved
	e := NewEngine()

	cqlSource := `library Test version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
define Val: FHIRHelpers.ToString('hello')`

	results, err := e.EvaluateLibrary(context.Background(), cqlSource, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := results["Val"].(fptypes.String)
	if !ok {
		t.Fatalf("Val: expected String, got %T (%v)", results["Val"], results["Val"])
	}
	if val.Value() != "hello" {
		t.Errorf("Val = %q, want %q", val.Value(), "hello")
	}
}

func TestErrorTypes(t *testing.T) {
	t.Run("ErrSyntaxError", func(t *testing.T) {
		err := &ErrSyntaxError{Cause: errors.New("unexpected token")}
		if err.Error() != "CQL syntax error: unexpected token" {
			t.Errorf("error = %q", err.Error())
		}
		if err.Unwrap() == nil {
			t.Error("Unwrap should return cause")
		}
	})

	t.Run("ErrEvaluation", func(t *testing.T) {
		err := &ErrEvaluation{Cause: errors.New("division by zero")}
		if err.Error() != "CQL evaluation error: division by zero" {
			t.Errorf("error = %q", err.Error())
		}
		if err.Unwrap() == nil {
			t.Error("Unwrap should return cause")
		}
	})

	t.Run("ErrTimeout", func(t *testing.T) {
		err := &ErrTimeout{Duration: 5 * time.Second}
		if err.Error() != "CQL evaluation timed out after 5s" {
			t.Errorf("error = %q", err.Error())
		}
	})

	t.Run("ErrTooCostly", func(t *testing.T) {
		err := &ErrTooCostly{Msg: "too big"}
		if err.Error() != "CQL evaluation too costly: too big" {
			t.Errorf("error = %q", err.Error())
		}
	})
}
