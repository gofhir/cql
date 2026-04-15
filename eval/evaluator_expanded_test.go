package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
	cqltypes "github.com/gofhir/cql/types"
)

// ---------------------------------------------------------------------------
// Case Expression
// ---------------------------------------------------------------------------

func TestEval_CaseExpression_WhenThen(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// case when true then 42 when false then 0 else -1 end
	expr := &ast.CaseExpression{
		Items: []*ast.CaseItem{
			{
				When: &ast.Literal{ValueType: ast.LiteralBoolean, Value: "true"},
				Then: &ast.Literal{ValueType: ast.LiteralInteger, Value: "42"},
			},
			{
				When: &ast.Literal{ValueType: ast.LiteralBoolean, Value: "false"},
				Then: &ast.Literal{ValueType: ast.LiteralInteger, Value: "0"},
			},
		},
		Else: &ast.Literal{ValueType: ast.LiteralInteger, Value: "-1"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, val, 42)
}

func TestEval_CaseExpression_FallsToElse(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.CaseExpression{
		Items: []*ast.CaseItem{
			{
				When: &ast.Literal{ValueType: ast.LiteralBoolean, Value: "false"},
				Then: &ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
			},
		},
		Else: &ast.Literal{ValueType: ast.LiteralInteger, Value: "99"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, val, 99)
}

func TestEval_CaseExpression_WithComparand(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// case 5 when 3 then 'three' when 5 then 'five' else 'other' end
	expr := &ast.CaseExpression{
		Comparand: &ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
		Items: []*ast.CaseItem{
			{
				When: &ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
				Then: &ast.Literal{ValueType: ast.LiteralString, Value: "three"},
			},
			{
				When: &ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
				Then: &ast.Literal{ValueType: ast.LiteralString, Value: "five"},
			},
		},
		Else: &ast.Literal{ValueType: ast.LiteralString, Value: "other"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := val.(fptypes.String)
	if !ok {
		t.Fatalf("expected String, got %T", val)
	}
	if s.Value() != "five" {
		t.Errorf("got %q, want %q", s.Value(), "five")
	}
}

// ---------------------------------------------------------------------------
// Is Expression
// ---------------------------------------------------------------------------

func TestEval_IsExpression(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// 42 is Integer → true
	expr := &ast.IsExpression{
		Operand: &ast.Literal{ValueType: ast.LiteralInteger, Value: "42"},
		Type:    &ast.NamedType{Name: "Integer"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := val.(fptypes.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", val)
	}
	if !b.Bool() {
		t.Error("42 is Integer should be true")
	}
}

func TestEval_IsExpression_Mismatch(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// 'hello' is Integer → false
	expr := &ast.IsExpression{
		Operand: &ast.Literal{ValueType: ast.LiteralString, Value: "hello"},
		Type:    &ast.NamedType{Name: "Integer"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := val.(fptypes.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", val)
	}
	if b.Bool() {
		t.Error("'hello' is Integer should be false")
	}
}

// ---------------------------------------------------------------------------
// BooleanTest Expression
// ---------------------------------------------------------------------------

func TestEval_BooleanTest_IsTrue(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.BooleanTestExpression{
		Operand:   &ast.Literal{ValueType: ast.LiteralBoolean, Value: "true"},
		TestValue: "true",
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := val.(fptypes.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", val)
	}
	if !b.Bool() {
		t.Error("true is true should be true")
	}
}

func TestEval_BooleanTest_IsFalse(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.BooleanTestExpression{
		Operand:   &ast.Literal{ValueType: ast.LiteralBoolean, Value: "false"},
		TestValue: "false",
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := val.(fptypes.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", val)
	}
	if !b.Bool() {
		t.Error("false is false should be true")
	}
}

func TestEval_BooleanTest_IsNull(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.BooleanTestExpression{
		Operand:   &ast.Literal{ValueType: ast.LiteralNull},
		TestValue: "null",
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := val.(fptypes.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", val)
	}
	if !b.Bool() {
		t.Error("null is null should be true")
	}
}

func TestEval_BooleanTest_IsNotNull(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.BooleanTestExpression{
		Operand:   &ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
		TestValue: "null",
		Not:       true,
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := val.(fptypes.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", val)
	}
	if !b.Bool() {
		t.Error("1 is not null should be true")
	}
}

// ---------------------------------------------------------------------------
// MemberAccess
// ---------------------------------------------------------------------------

func TestEval_MemberAccess_Tuple(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// Build a tuple first, then access a member
	tupleExpr := &ast.TupleExpression{
		Elements: []*ast.TupleElement{
			{Name: "name", Expression: &ast.Literal{ValueType: ast.LiteralString, Value: "John"}},
			{Name: "age", Expression: &ast.Literal{ValueType: ast.LiteralInteger, Value: "30"}},
		},
	}
	expr := &ast.MemberAccess{
		Source: tupleExpr,
		Member: "age",
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, val, 30)
}

// ---------------------------------------------------------------------------
// IndexAccess
// ---------------------------------------------------------------------------

func TestEval_IndexAccess(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// {10, 20, 30}[1] = 20
	list := &ast.ListExpression{
		Elements: []ast.Expression{
			&ast.Literal{ValueType: ast.LiteralInteger, Value: "10"},
			&ast.Literal{ValueType: ast.LiteralInteger, Value: "20"},
			&ast.Literal{ValueType: ast.LiteralInteger, Value: "30"},
		},
	}
	expr := &ast.IndexAccess{
		Source: list,
		Index:  &ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, val, 20)
}

func TestEval_IndexAccess_OutOfBounds(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	list := &ast.ListExpression{
		Elements: []ast.Expression{
			&ast.Literal{ValueType: ast.LiteralInteger, Value: "10"},
		},
	}
	expr := &ast.IndexAccess{
		Source: list,
		Index:  &ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil for out-of-bounds, got %v", val)
	}
}

// ---------------------------------------------------------------------------
// Query (with where, let, return)
// ---------------------------------------------------------------------------

func TestEval_Query_BasicFilter(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// from {1, 2, 3, 4, 5} X where X > 3 return X
	expr := &ast.Query{
		Sources: []*ast.AliasedSource{
			{
				Source: &ast.ListExpression{
					Elements: []ast.Expression{
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "4"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
					},
				},
				Alias: "X",
			},
		},
		Where: &ast.BinaryExpression{
			Operator: ast.OpGreater,
			Left:     &ast.IdentifierRef{Name: "X"},
			Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	if list.Values.Count() != 2 {
		t.Errorf("expected 2 items (4, 5), got %d", list.Values.Count())
	}
}

func TestEval_Query_WithLetBinding(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// from {1, 2, 3} X let doubled: X * 2 return doubled
	expr := &ast.Query{
		Sources: []*ast.AliasedSource{
			{
				Source: &ast.ListExpression{
					Elements: []ast.Expression{
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
					},
				},
				Alias: "X",
			},
		},
		Let: []*ast.LetClause{
			{
				Identifier: "doubled",
				Expression: &ast.BinaryExpression{
					Operator: ast.OpMultiply,
					Left:     &ast.IdentifierRef{Name: "X"},
					Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
				},
			},
		},
		Return: &ast.ReturnClause{
			Expression: &ast.IdentifierRef{Name: "doubled"},
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	if list.Values.Count() != 3 {
		t.Errorf("expected 3 items, got %d", list.Values.Count())
	}
	// First item should be 2
	assertInteger(t, list.Values[0], 2)
}

func TestEval_Query_ReturnDistinct(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// from {1, 1, 2, 2, 3} X return distinct X
	expr := &ast.Query{
		Sources: []*ast.AliasedSource{
			{
				Source: &ast.ListExpression{
					Elements: []ast.Expression{
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
					},
				},
				Alias: "X",
			},
		},
		Return: &ast.ReturnClause{
			Distinct:   true,
			Expression: &ast.IdentifierRef{Name: "X"},
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	if list.Values.Count() != 3 {
		t.Errorf("expected 3 distinct items, got %d", list.Values.Count())
	}
}

func TestEval_Query_EmptySource(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.Query{
		Sources: []*ast.AliasedSource{
			{
				Source: &ast.ListExpression{Elements: nil},
				Alias:  "X",
			},
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	if list.Values.Count() != 0 {
		t.Errorf("expected empty list, got %d items", list.Values.Count())
	}
}

// ---------------------------------------------------------------------------
// Sort clause
// ---------------------------------------------------------------------------

func TestEval_Query_SortAsc(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// from {3, 1, 4, 1, 5} X sort asc
	expr := &ast.Query{
		Sources: []*ast.AliasedSource{
			{
				Source: &ast.ListExpression{
					Elements: []ast.Expression{
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "4"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
					},
				},
				Alias: "X",
			},
		},
		Sort: &ast.SortClause{
			Direction: ast.SortAsc,
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	expected := []int64{1, 1, 3, 4, 5}
	if list.Values.Count() != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), list.Values.Count())
	}
	for i, exp := range expected {
		assertInteger(t, list.Values[i], exp)
	}
}

func TestEval_Query_SortDesc(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// from {3, 1, 4, 1, 5} X sort desc
	expr := &ast.Query{
		Sources: []*ast.AliasedSource{
			{
				Source: &ast.ListExpression{
					Elements: []ast.Expression{
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "4"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
					},
				},
				Alias: "X",
			},
		},
		Sort: &ast.SortClause{
			Direction: ast.SortDesc,
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	expected := []int64{5, 4, 3, 1, 1}
	if list.Values.Count() != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), list.Values.Count())
	}
	for i, exp := range expected {
		assertInteger(t, list.Values[i], exp)
	}
}

func TestEval_Query_SortByExpression(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// from {5, 2, 8, 1, 9} X sort by X asc
	expr := &ast.Query{
		Sources: []*ast.AliasedSource{
			{
				Source: &ast.ListExpression{
					Elements: []ast.Expression{
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "8"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "9"},
					},
				},
				Alias: "X",
			},
		},
		Sort: &ast.SortClause{
			ByItems: []*ast.SortByItem{
				{
					Expression: &ast.IdentifierRef{Name: "X"},
					Direction:  ast.SortAsc,
				},
			},
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	expected := []int64{1, 2, 5, 8, 9}
	if list.Values.Count() != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), list.Values.Count())
	}
	for i, exp := range expected {
		assertInteger(t, list.Values[i], exp)
	}
}

func TestEval_Query_SortByExpressionDesc(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// from {5, 2, 8, 1, 9} X sort by X desc
	expr := &ast.Query{
		Sources: []*ast.AliasedSource{
			{
				Source: &ast.ListExpression{
					Elements: []ast.Expression{
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "8"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "9"},
					},
				},
				Alias: "X",
			},
		},
		Sort: &ast.SortClause{
			ByItems: []*ast.SortByItem{
				{
					Expression: &ast.IdentifierRef{Name: "X"},
					Direction:  ast.SortDesc,
				},
			},
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	expected := []int64{9, 8, 5, 2, 1}
	if list.Values.Count() != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), list.Values.Count())
	}
	for i, exp := range expected {
		assertInteger(t, list.Values[i], exp)
	}
}

func TestEval_Query_SortByStringAsc(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// from {'banana', 'apple', 'cherry'} X sort by X asc
	expr := &ast.Query{
		Sources: []*ast.AliasedSource{
			{
				Source: &ast.ListExpression{
					Elements: []ast.Expression{
						&ast.Literal{ValueType: ast.LiteralString, Value: "banana"},
						&ast.Literal{ValueType: ast.LiteralString, Value: "apple"},
						&ast.Literal{ValueType: ast.LiteralString, Value: "cherry"},
					},
				},
				Alias: "X",
			},
		},
		Sort: &ast.SortClause{
			ByItems: []*ast.SortByItem{
				{
					Expression: &ast.IdentifierRef{Name: "X"},
					Direction:  ast.SortAsc,
				},
			},
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	expectedStrs := []string{"apple", "banana", "cherry"}
	if list.Values.Count() != len(expectedStrs) {
		t.Fatalf("expected %d items, got %d", len(expectedStrs), list.Values.Count())
	}
	for i, exp := range expectedStrs {
		s, ok := list.Values[i].(fptypes.String)
		if !ok {
			t.Fatalf("item %d: expected String, got %T", i, list.Values[i])
		}
		if s.Value() != exp {
			t.Errorf("item %d: expected %q, got %q", i, exp, s.Value())
		}
	}
}

func TestEval_Query_SortEmptyList(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.Query{
		Sources: []*ast.AliasedSource{
			{
				Source: &ast.ListExpression{Elements: nil},
				Alias:  "X",
			},
		},
		Sort: &ast.SortClause{
			Direction: ast.SortAsc,
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	if list.Values.Count() != 0 {
		t.Errorf("expected empty list, got %d items", list.Values.Count())
	}
}

func TestEval_Query_SortStableOrder(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// from {3, 1, 4, 1, 5, 9, 2, 6, 5, 3} X sort asc
	// Verifies stable sort: equal elements maintain original relative order
	expr := &ast.Query{
		Sources: []*ast.AliasedSource{
			{
				Source: &ast.ListExpression{
					Elements: []ast.Expression{
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "4"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "9"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "6"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
					},
				},
				Alias: "X",
			},
		},
		Sort: &ast.SortClause{
			Direction: ast.SortAsc,
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	expected := []int64{1, 1, 2, 3, 3, 4, 5, 5, 6, 9}
	if list.Values.Count() != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), list.Values.Count())
	}
	for i, exp := range expected {
		assertInteger(t, list.Values[i], exp)
	}
}

// ---------------------------------------------------------------------------
// Division by zero (CQL spec: returns null)
// ---------------------------------------------------------------------------

func TestEval_DivisionByZero(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.BinaryExpression{
		Operator: ast.OpDivide,
		Left:     &ast.Literal{ValueType: ast.LiteralInteger, Value: "10"},
		Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "0"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("division by zero should return null, got %v", val)
	}
}

func TestEval_ModuloByZero(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.BinaryExpression{
		Operator: ast.OpMod,
		Left:     &ast.Literal{ValueType: ast.LiteralInteger, Value: "10"},
		Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "0"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("modulo by zero should return null, got %v", val)
	}
}

// ---------------------------------------------------------------------------
// Set operations: union, intersect, except
// ---------------------------------------------------------------------------

func TestEval_Union(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.BinaryExpression{
		Operator: ast.OpUnion,
		Left: &ast.ListExpression{
			Elements: []ast.Expression{
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
			},
		},
		Right: &ast.ListExpression{
			Elements: []ast.Expression{
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
			},
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	// Union should produce {1, 2, 3} (distinct)
	if list.Values.Count() != 3 {
		t.Errorf("expected 3 items in union, got %d", list.Values.Count())
	}
}

func TestEval_Intersect(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.BinaryExpression{
		Operator: ast.OpIntersect,
		Left: &ast.ListExpression{
			Elements: []ast.Expression{
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
			},
		},
		Right: &ast.ListExpression{
			Elements: []ast.Expression{
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "4"},
			},
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	// Intersect should produce {2, 3}
	if list.Values.Count() != 2 {
		t.Errorf("expected 2 items in intersect, got %d", list.Values.Count())
	}
}

func TestEval_Except(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.BinaryExpression{
		Operator: ast.OpExcept,
		Left: &ast.ListExpression{
			Elements: []ast.Expression{
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
			},
		},
		Right: &ast.ListExpression{
			Elements: []ast.Expression{
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
			},
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := val.(cqltypes.List)
	if !ok {
		t.Fatalf("expected List, got %T", val)
	}
	// Except should produce {1, 3}
	if list.Values.Count() != 2 {
		t.Errorf("expected 2 items in except, got %d", list.Values.Count())
	}
}

// ---------------------------------------------------------------------------
// Membership expression: in / contains
// ---------------------------------------------------------------------------

func TestEval_Membership_In(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// 2 in {1, 2, 3}
	expr := &ast.MembershipExpression{
		Left: &ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
		Right: &ast.ListExpression{
			Elements: []ast.Expression{
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
			},
		},
		Operator: "in",
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := val.(fptypes.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", val)
	}
	if !b.Bool() {
		t.Error("2 in {1, 2, 3} should be true")
	}
}

func TestEval_Membership_NotIn(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// 5 in {1, 2}
	expr := &ast.MembershipExpression{
		Left: &ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
		Right: &ast.ListExpression{
			Elements: []ast.Expression{
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
			},
		},
		Operator: "in",
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := val.(fptypes.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", val)
	}
	if b.Bool() {
		t.Error("5 in {1, 2} should be false")
	}
}

// ---------------------------------------------------------------------------
// Tuple expression
// ---------------------------------------------------------------------------

func TestEval_TupleExpression(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.TupleExpression{
		Elements: []*ast.TupleElement{
			{Name: "name", Expression: &ast.Literal{ValueType: ast.LiteralString, Value: "John"}},
			{Name: "age", Expression: &ast.Literal{ValueType: ast.LiteralInteger, Value: "30"}},
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val == nil {
		t.Fatal("expected non-nil tuple")
	}
}

// ---------------------------------------------------------------------------
// Convert expression
// ---------------------------------------------------------------------------

func TestEval_ConvertExpression_IntToString(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.ConvertExpression{
		Operand: &ast.Literal{ValueType: ast.LiteralInteger, Value: "42"},
		ToType:  &ast.NamedType{Name: "String"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := val.(fptypes.String)
	if !ok {
		t.Fatalf("expected String, got %T", val)
	}
	if s.Value() != "42" {
		t.Errorf("got %q, want %q", s.Value(), "42")
	}
}

// ---------------------------------------------------------------------------
// Null propagation in arithmetic
// ---------------------------------------------------------------------------

func TestEval_NullPropagation_Add(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.BinaryExpression{
		Operator: ast.OpAdd,
		Left:     &ast.Literal{ValueType: ast.LiteralNull},
		Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("null + 5 should be null, got %v", val)
	}
}

// ---------------------------------------------------------------------------
// User-defined functions
// ---------------------------------------------------------------------------

func TestEval_UserDefinedFunction(t *testing.T) {
	lib := &ast.Library{
		Functions: []*ast.FunctionDef{
			{
				Name: "Double",
				Operands: []*ast.OperandDef{
					{Name: "x"},
				},
				Body: &ast.BinaryExpression{
					Operator: ast.OpMultiply,
					Left:     &ast.IdentifierRef{Name: "x"},
					Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
				},
			},
		},
	}
	ctx := NewContext(context.Background(), lib)
	ev := NewEvaluator(ctx)

	expr := &ast.FunctionCall{
		Name: "Double",
		Operands: []ast.Expression{
			&ast.Literal{ValueType: ast.LiteralInteger, Value: "21"},
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, val, 42)
}

// ---------------------------------------------------------------------------
// FunctionCall — built-in Count
// ---------------------------------------------------------------------------

func TestEval_FunctionCall_Count(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.FunctionCall{
		Name: "Count",
		Source: &ast.ListExpression{
			Elements: []ast.Expression{
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
			},
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, val, 3)
}

// ---------------------------------------------------------------------------
// FunctionCall — built-in Exists
// ---------------------------------------------------------------------------

func TestEval_FunctionCall_Exists(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	tests := []struct {
		name     string
		elements []ast.Expression
		expected bool
	}{
		{"non-empty list", []ast.Expression{&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"}}, true},
		{"empty list", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := &ast.FunctionCall{
				Name:   "Exists",
				Source: &ast.ListExpression{Elements: tt.elements},
			}
			val, err := ev.Eval(expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			b, ok := val.(fptypes.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", val)
			}
			if b.Bool() != tt.expected {
				t.Errorf("got %v, want %v", b.Bool(), tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FunctionCall — built-in Sum
// ---------------------------------------------------------------------------

func TestEval_FunctionCall_Sum(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.FunctionCall{
		Name: "Sum",
		Source: &ast.ListExpression{
			Elements: []ast.Expression{
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "10"},
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "20"},
				&ast.Literal{ValueType: ast.LiteralInteger, Value: "30"},
			},
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val == nil {
		t.Fatal("got nil, want 60")
	}
	// Sum returns a Decimal
	if val.String() != "60" {
		t.Errorf("got %s, want 60", val.String())
	}
}

// ---------------------------------------------------------------------------
// Context caching: GetContextSubjectID caches, SetContextResource invalidates
// ---------------------------------------------------------------------------

func TestContext_GetContextSubjectID_Caching(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.ContextValue = json.RawMessage(`{"resourceType":"Patient","id":"patient-1"}`)

	id1 := ctx.GetContextSubjectID()
	if id1 != "patient-1" {
		t.Errorf("expected patient-1, got %s", id1)
	}
	// Should be cached
	id2 := ctx.GetContextSubjectID()
	if id2 != "patient-1" {
		t.Errorf("cached value should return patient-1, got %s", id2)
	}

	// SetContextResource should invalidate cache
	ctx.SetContextResource("Patient", json.RawMessage(`{"resourceType":"Patient","id":"patient-2"}`))
	id3 := ctx.GetContextSubjectID()
	if id3 != "patient-2" {
		t.Errorf("after SetContextResource, expected patient-2, got %s", id3)
	}
}

func TestContext_GetContextObject_Caching(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.ContextValue = json.RawMessage(`{"resourceType":"Patient","id":"p1","birthDate":"1990-01-01"}`)

	obj1 := ctx.GetContextObject()
	if obj1 == nil {
		t.Fatal("expected non-nil object")
	}
	obj2 := ctx.GetContextObject()
	if obj1 != obj2 {
		t.Error("expected same cached object on second call")
	}

	// SetContextResource should invalidate
	ctx.SetContextResource("Patient", json.RawMessage(`{"resourceType":"Patient","id":"p2"}`))
	obj3 := ctx.GetContextObject()
	if obj3 == obj1 {
		t.Error("expected new object after SetContextResource")
	}
}

// ---------------------------------------------------------------------------
// withContext reuses funcs map
// ---------------------------------------------------------------------------

func TestEvaluator_WithContext(t *testing.T) {
	lib := &ast.Library{
		Functions: []*ast.FunctionDef{
			{Name: "TestFunc", Body: &ast.Literal{ValueType: ast.LiteralInteger, Value: "1"}},
		},
	}
	ctx1 := NewContext(context.Background(), lib)
	ev1 := NewEvaluator(ctx1)

	ctx2 := ctx1.ChildScope()
	ev2 := ev1.withContext(ctx2)

	// ev2 should be able to call TestFunc even though it wasn't built from NewEvaluator
	if _, ok := ev2.funcs["TestFunc"]; !ok {
		t.Error("withContext should share the funcs map")
	}
}

// ---------------------------------------------------------------------------
// SubList
// ---------------------------------------------------------------------------

func TestEval_SubList(t *testing.T) {
	mkList := func(vals ...int64) *ast.ListExpression {
		elems := make([]ast.Expression, len(vals))
		for i, v := range vals {
			elems[i] = &ast.Literal{ValueType: ast.LiteralInteger, Value: fmt.Sprintf("%d", v)}
		}
		return &ast.ListExpression{Elements: elems}
	}
	mkInt := func(v int64) ast.Expression {
		return &ast.Literal{ValueType: ast.LiteralInteger, Value: fmt.Sprintf("%d", v)}
	}

	tests := []struct {
		name     string
		src      ast.Expression
		operands []ast.Expression
		wantNil  bool
		wantVals []int64
	}{
		{
			name:     "null list returns null",
			src:      nil,
			operands: []ast.Expression{mkInt(0)},
			wantNil:  true,
		},
		{
			name:     "from middle without length",
			src:      mkList(10, 20, 30, 40, 50),
			operands: []ast.Expression{mkInt(2)},
			wantVals: []int64{30, 40, 50},
		},
		{
			name:     "from middle with length",
			src:      mkList(10, 20, 30, 40, 50),
			operands: []ast.Expression{mkInt(1), mkInt(2)},
			wantVals: []int64{20, 30},
		},
		{
			name:     "start past end returns empty",
			src:      mkList(10, 20),
			operands: []ast.Expression{mkInt(5)},
			wantVals: []int64{},
		},
		{
			name:     "negative start clamps to 0",
			src:      mkList(10, 20, 30),
			operands: []ast.Expression{mkInt(-2)},
			wantVals: []int64{10, 20, 30},
		},
		{
			name:     "length exceeds remainder returns all remaining",
			src:      mkList(10, 20, 30),
			operands: []ast.Expression{mkInt(1), mkInt(100)},
			wantVals: []int64{20, 30},
		},
		{
			name:     "start at 0 with length 0 returns empty",
			src:      mkList(10, 20, 30),
			operands: []ast.Expression{mkInt(0), mkInt(0)},
			wantVals: []int64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext(context.Background(), nil)
			ev := NewEvaluator(ctx)

			var expr *ast.FunctionCall
			if tt.src == nil {
				// Standalone call: SubList(null, startIndex)
				ops := append([]ast.Expression{&ast.Literal{ValueType: ast.LiteralNull}}, tt.operands...)
				expr = &ast.FunctionCall{
					Name:     "SubList",
					Operands: ops,
				}
			} else {
				expr = &ast.FunctionCall{
					Name:     "SubList",
					Source:   tt.src,
					Operands: tt.operands,
				}
			}
			val, err := ev.Eval(expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if val != nil {
					t.Fatalf("want nil, got %v", val)
				}
				return
			}
			list, ok := val.(cqltypes.List)
			if !ok {
				t.Fatalf("want List, got %T", val)
			}
			if len(list.Values) != len(tt.wantVals) {
				t.Fatalf("want %d elements, got %d: %v", len(tt.wantVals), len(list.Values), list)
			}
			for i, want := range tt.wantVals {
				iv, ok := list.Values[i].(fptypes.Integer)
				if !ok {
					t.Fatalf("element %d: want Integer, got %T", i, list.Values[i])
				}
				if iv.Value() != want {
					t.Errorf("element %d: got %d, want %d", i, iv.Value(), want)
				}
			}
		})
	}
}

// TestEval_SubList_NoAlias verifies SubList copies the slice and does not alias the original.
func TestEval_SubList_NoAlias(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	listExpr := &ast.ListExpression{
		Elements: []ast.Expression{
			&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
			&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
			&ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
		},
	}
	// Evaluate the original list
	origVal, err := ev.Eval(listExpr)
	if err != nil {
		t.Fatal(err)
	}
	origList := origVal.(cqltypes.List)

	// SubList(list, 0)
	expr := &ast.FunctionCall{
		Name:     "SubList",
		Operands: []ast.Expression{
			listExpr,
			&ast.Literal{ValueType: ast.LiteralInteger, Value: "0"},
		},
	}
	subVal, err := ev.Eval(expr)
	if err != nil {
		t.Fatal(err)
	}
	subList := subVal.(cqltypes.List)

	// Mutating subList should not affect origList
	if len(subList.Values) > 0 {
		subList.Values[0] = fptypes.NewInteger(999)
	}
	first := origList.Values[0].(fptypes.Integer)
	if first.Value() == 999 {
		t.Error("SubList result aliases the original list")
	}
}

// ---------------------------------------------------------------------------
// SplitOnMatches
// ---------------------------------------------------------------------------

func TestEval_SplitOnMatches(t *testing.T) {
	mkStr := func(s string) ast.Expression {
		return &ast.Literal{ValueType: ast.LiteralString, Value: s}
	}

	tests := []struct {
		name     string
		src      ast.Expression
		pattern  ast.Expression
		wantNil  bool
		wantVals []string
	}{
		{
			name:    "null string returns null",
			src:     nil,
			pattern: mkStr(`\s+`),
			wantNil: true,
		},
		{
			name:     "split on whitespace",
			src:      mkStr("hello world  foo"),
			pattern:  mkStr(`\s+`),
			wantVals: []string{"hello", "world", "foo"},
		},
		{
			name:     "no match returns single element",
			src:      mkStr("hello"),
			pattern:  mkStr(`,`),
			wantVals: []string{"hello"},
		},
		{
			name:     "multiple parts with comma",
			src:      mkStr("a,b,,c"),
			pattern:  mkStr(`,`),
			wantVals: []string{"a", "b", "", "c"},
		},
		{
			name:     "empty string",
			src:      mkStr(""),
			pattern:  mkStr(`,`),
			wantVals: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext(context.Background(), nil)
			ev := NewEvaluator(ctx)

			var expr *ast.FunctionCall
			if tt.src == nil {
				// Standalone call: SplitOnMatches(null, pattern)
				expr = &ast.FunctionCall{
					Name:     "SplitOnMatches",
					Operands: []ast.Expression{&ast.Literal{ValueType: ast.LiteralNull}, tt.pattern},
				}
			} else {
				expr = &ast.FunctionCall{
					Name:     "SplitOnMatches",
					Source:   tt.src,
					Operands: []ast.Expression{tt.pattern},
				}
			}
			val, err := ev.Eval(expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if val != nil {
					t.Fatalf("want nil, got %v", val)
				}
				return
			}
			list, ok := val.(cqltypes.List)
			if !ok {
				t.Fatalf("want List, got %T", val)
			}
			if len(list.Values) != len(tt.wantVals) {
				t.Fatalf("want %d elements, got %d: %v", len(tt.wantVals), len(list.Values), list)
			}
			for i, want := range tt.wantVals {
				sv, ok := list.Values[i].(fptypes.String)
				if !ok {
					t.Fatalf("element %d: want String, got %T", i, list.Values[i])
				}
				if sv.Value() != want {
					t.Errorf("element %d: got %q, want %q", i, sv.Value(), want)
				}
			}
		})
	}
}

func TestEval_SplitOnMatches_InvalidRegex(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.FunctionCall{
		Name:   "SplitOnMatches",
		Source: &ast.Literal{ValueType: ast.LiteralString, Value: "test"},
		Operands: []ast.Expression{
			&ast.Literal{ValueType: ast.LiteralString, Value: "[invalid"},
		},
	}
	_, err := ev.Eval(expr)
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}
