package eval

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofhir/cql/ast"
	cqltypes "github.com/gofhir/cql/types"
)

// mockValueSetExpander implements both TerminologyProvider and ValueSetExpander.
type mockValueSetExpander struct{}

func (m *mockValueSetExpander) InValueSet(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}

func (m *mockValueSetExpander) ExpandValueSet(_ context.Context, url string) ([]ExpandedCode, error) {
	if url == "http://example.org/vs/colors" {
		return []ExpandedCode{
			{System: "http://example.org", Code: "red", Display: "Red"},
			{System: "http://example.org", Code: "blue", Display: "Blue"},
		}, nil
	}
	return nil, nil
}

// mockExpanderError returns an error on expand.
type mockExpanderError struct{}

func (m *mockExpanderError) InValueSet(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}

func (m *mockExpanderError) ExpandValueSet(_ context.Context, _ string) ([]ExpandedCode, error) {
	return nil, fmt.Errorf("terminology service unavailable")
}

func TestExpandValueSet_NilProvider(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.FunctionCall{
		Name: "expandValueSet",
		Operands: []ast.Expression{
			&ast.Literal{ValueType: ast.LiteralString, Value: "http://example.org/vs/colors"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil (null) when no provider, got %v", val)
	}
}

func TestExpandValueSet_ProviderWithoutExpander(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.TerminologyProvider = &mockTermOnlyProvider{} // does not implement ValueSetExpander
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.FunctionCall{
		Name: "expandValueSet",
		Operands: []ast.Expression{
			&ast.Literal{ValueType: ast.LiteralString, Value: "http://example.org/vs/colors"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil (null) when provider lacks ValueSetExpander, got %v", val)
	}
}

func TestExpandValueSet_WithExpander(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.TerminologyProvider = &mockValueSetExpander{}
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.FunctionCall{
		Name: "expandValueSet",
		Operands: []ast.Expression{
			&ast.Literal{ValueType: ast.LiteralString, Value: "http://example.org/vs/colors"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	if len(list.Values) != 2 {
		t.Fatalf("expected 2 codes, got %d", len(list.Values))
	}
	// Verify first code
	code0, ok := list.Values[0].(cqltypes.Code)
	if !ok {
		t.Fatalf("expected Code, got %T", list.Values[0])
	}
	if code0.System != "http://example.org" || code0.Code != "red" || code0.Display != "Red" {
		t.Errorf("unexpected first code: %+v", code0)
	}
	// Verify second code
	code1, ok := list.Values[1].(cqltypes.Code)
	if !ok {
		t.Fatalf("expected Code, got %T", list.Values[1])
	}
	if code1.System != "http://example.org" || code1.Code != "blue" || code1.Display != "Blue" {
		t.Errorf("unexpected second code: %+v", code1)
	}
}

func TestExpandValueSet_UnknownValueSet(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.TerminologyProvider = &mockValueSetExpander{}
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.FunctionCall{
		Name: "expandValueSet",
		Operands: []ast.Expression{
			&ast.Literal{ValueType: ast.LiteralString, Value: "http://example.org/vs/unknown"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Provider returns nil for unknown VS, so we get a list with 0 elements
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	if len(list.Values) != 0 {
		t.Errorf("expected empty list for unknown VS, got %d codes", len(list.Values))
	}
}

func TestExpandValueSet_EmptyURL(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.TerminologyProvider = &mockValueSetExpander{}
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.FunctionCall{
		Name: "expandValueSet",
		Operands: []ast.Expression{
			&ast.Literal{ValueType: ast.LiteralString, Value: ""},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil for empty URL, got %v", val)
	}
}

func TestExpandValueSet_WithValueSetRef(t *testing.T) {
	lib := &ast.Library{
		ValueSets: []*ast.ValueSetDef{
			{Name: "Colors", ID: "http://example.org/vs/colors"},
		},
	}
	ctx := NewContext(context.Background(), lib)
	ctx.TerminologyProvider = &mockValueSetExpander{}
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.FunctionCall{
		Name: "expandValueSet",
		Operands: []ast.Expression{
			&ast.IdentifierRef{Name: "Colors"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	if len(list.Values) != 2 {
		t.Fatalf("expected 2 codes via ValueSet ref, got %d", len(list.Values))
	}
}

func TestExpandValueSet_Error(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.TerminologyProvider = &mockExpanderError{}
	ev := NewEvaluator(ctx)

	_, err := ev.Eval(&ast.FunctionCall{
		Name: "expandValueSet",
		Operands: []ast.Expression{
			&ast.Literal{ValueType: ast.LiteralString, Value: "http://example.org/vs/colors"},
		},
	})
	if err == nil {
		t.Fatal("expected error from failing expander, got nil")
	}
}
