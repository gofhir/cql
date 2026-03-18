package eval

import (
	"context"
	"encoding/json"
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
