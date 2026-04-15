package eval

import (
	"context"
	"fmt"
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
)

// mockQuantityConverter supports mg→g conversion only.
type mockQuantityConverter struct{}

func (m *mockQuantityConverter) ConvertQuantity(value float64, from, to string) (float64, error) {
	if from == "mg" && to == "g" {
		return value / 1000, nil
	}
	if from == "g" && to == "mg" {
		return value * 1000, nil
	}
	return 0, fmt.Errorf("unsupported conversion %s -> %s", from, to)
}

func TestConvertQuantity_NilConverter(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	// QuantityConverter is nil by default
	ev := NewEvaluator(ctx)

	q, _ := fptypes.NewQuantity("100 'mg'")
	result, err := ev.evalConvertQuantity(q, "g")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil when converter is nil, got %v", result)
	}
}

func TestConvertQuantity_NilSource(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.QuantityConverter = &mockQuantityConverter{}
	ev := NewEvaluator(ctx)

	result, err := ev.evalConvertQuantity(nil, "g")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for nil source, got %v", result)
	}
}

func TestConvertQuantity_NonQuantitySource(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.QuantityConverter = &mockQuantityConverter{}
	ev := NewEvaluator(ctx)

	result, err := ev.evalConvertQuantity(fptypes.NewString("not a quantity"), "g")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for non-quantity source, got %v", result)
	}
}

func TestConvertQuantity_SuccessfulConversion(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.QuantityConverter = &mockQuantityConverter{}
	ev := NewEvaluator(ctx)

	q, _ := fptypes.NewQuantity("1000 'mg'")
	result, err := ev.evalConvertQuantity(q, "g")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for successful conversion")
	}
	rq, ok := result.(fptypes.Quantity)
	if !ok {
		t.Fatalf("expected Quantity result, got %T", result)
	}
	val, _ := rq.Value().Float64()
	if val != 1.0 {
		t.Errorf("expected 1.0 g, got %v", val)
	}
	if rq.Unit() != "g" {
		t.Errorf("expected unit 'g', got %q", rq.Unit())
	}
}

func TestConvertQuantity_FailedConversion(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.QuantityConverter = &mockQuantityConverter{}
	ev := NewEvaluator(ctx)

	q, _ := fptypes.NewQuantity("100 'mg'")
	result, err := ev.evalConvertQuantity(q, "km")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for failed conversion, got %v", result)
	}
}

func TestCanConvertQuantity_Supported(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.QuantityConverter = &mockQuantityConverter{}
	ev := NewEvaluator(ctx)

	q, _ := fptypes.NewQuantity("100 'mg'")
	result, err := ev.evalCanConvertQuantity(q, "g")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := result.(fptypes.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if !b.Bool() {
		t.Error("expected true for supported conversion mg→g")
	}
}

func TestCanConvertQuantity_Unsupported(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.QuantityConverter = &mockQuantityConverter{}
	ev := NewEvaluator(ctx)

	q, _ := fptypes.NewQuantity("100 'mg'")
	result, err := ev.evalCanConvertQuantity(q, "km")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := result.(fptypes.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if b.Bool() {
		t.Error("expected false for unsupported conversion mg→km")
	}
}

func TestCanConvertQuantity_NilConverter(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	q, _ := fptypes.NewQuantity("100 'mg'")
	result, err := ev.evalCanConvertQuantity(q, "g")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil when converter is nil, got %v", result)
	}
}

func TestCanConvertQuantity_NilSource(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.QuantityConverter = &mockQuantityConverter{}
	ev := NewEvaluator(ctx)

	result, err := ev.evalCanConvertQuantity(nil, "g")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for nil source, got %v", result)
	}
}

func TestCanConvertQuantity_NonQuantitySource(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.QuantityConverter = &mockQuantityConverter{}
	ev := NewEvaluator(ctx)

	result, err := ev.evalCanConvertQuantity(fptypes.NewString("not a quantity"), "g")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := result.(fptypes.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if b.Bool() {
		t.Error("expected false for non-quantity source")
	}
}

// TestConvertQuantity_Dispatch tests the full dispatch path through evalFunctionCall.
func TestConvertQuantity_Dispatch(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.QuantityConverter = &mockQuantityConverter{}

	q, _ := fptypes.NewQuantity("500 'mg'")
	ctx.Definitions["myQty"] = q
	ev := NewEvaluator(ctx)

	result, err := ev.Eval(&ast.FunctionCall{
		Name: "ConvertQuantity",
		Operands: []ast.Expression{
			&ast.IdentifierRef{Name: "myQty"},
			&ast.Literal{ValueType: ast.LiteralString, Value: "g"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	rq, ok := result.(fptypes.Quantity)
	if !ok {
		t.Fatalf("expected Quantity, got %T", result)
	}
	val, _ := rq.Value().Float64()
	if val != 0.5 {
		t.Errorf("expected 0.5 g, got %v", val)
	}
}

func TestCanConvertQuantity_Dispatch(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.QuantityConverter = &mockQuantityConverter{}

	q, _ := fptypes.NewQuantity("500 'mg'")
	ctx.Definitions["myQty"] = q
	ev := NewEvaluator(ctx)

	result, err := ev.Eval(&ast.FunctionCall{
		Name: "CanConvertQuantity",
		Operands: []ast.Expression{
			&ast.IdentifierRef{Name: "myQty"},
			&ast.Literal{ValueType: ast.LiteralString, Value: "g"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := result.(fptypes.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if !b.Bool() {
		t.Error("expected true for CanConvertQuantity mg→g")
	}
}

// TestQuantityConverter_ChildScopePropagation verifies the converter propagates to child scopes.
func TestQuantityConverter_ChildScopePropagation(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.QuantityConverter = &mockQuantityConverter{}

	child := ctx.ChildScope()
	if child.QuantityConverter == nil {
		t.Error("expected QuantityConverter to propagate to child scope")
	}

	ev := NewEvaluator(child)
	q, _ := fptypes.NewQuantity("2000 'mg'")
	result, err := ev.evalConvertQuantity(q, "g")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result in child scope")
	}
	rq := result.(fptypes.Quantity)
	val, _ := rq.Value().Float64()
	if val != 2.0 {
		t.Errorf("expected 2.0 g, got %v", val)
	}
}
