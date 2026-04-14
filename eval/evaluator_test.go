package eval

import (
	"context"
	"encoding/json"
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/model"
	cqltypes "github.com/gofhir/cql/types"
)

func TestEval_IntegerLiteral(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.Literal{ValueType: ast.LiteralInteger, Value: "42"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, val, 42)
}

func TestEval_StringLiteral(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.Literal{ValueType: ast.LiteralString, Value: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := val.(fptypes.String)
	if !ok {
		t.Fatalf("expected String, got %T", val)
	}
	if s.Value() != "hello" {
		t.Errorf("got %q, want %q", s.Value(), "hello")
	}
}

func TestEval_BooleanLiteral(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.Literal{ValueType: ast.LiteralBoolean, Value: "true"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := val.(fptypes.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", val)
	}
	if !b.Bool() {
		t.Error("expected true")
	}
}

func TestEval_NullLiteral(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.Literal{ValueType: ast.LiteralNull})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}

func TestEval_DecimalLiteral(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.Literal{ValueType: ast.LiteralDecimal, Value: "3.14"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val == nil {
		t.Fatal("expected non-nil")
	}
	if val.Type() != "Decimal" {
		t.Errorf("type = %s, want Decimal", val.Type())
	}
}

func TestEval_BinaryAdd(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.BinaryExpression{
		Operator: ast.OpAdd,
		Left:     &ast.Literal{ValueType: ast.LiteralInteger, Value: "10"},
		Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "20"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, val, 30)
}

func TestEval_BinaryMultiply(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.BinaryExpression{
		Operator: ast.OpMultiply,
		Left:     &ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
		Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "7"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, val, 35)
}

func TestEval_BinarySubtract(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.BinaryExpression{
		Operator: ast.OpSubtract,
		Left:     &ast.Literal{ValueType: ast.LiteralInteger, Value: "50"},
		Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "8"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, val, 42)
}

func TestEval_BinaryEqual(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.BinaryExpression{
		Operator: ast.OpEqual,
		Left:     &ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
		Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
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
		t.Error("5 = 5 should be true")
	}
}

func TestEval_BinaryNotEqual(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.BinaryExpression{
		Operator: ast.OpNotEqual,
		Left:     &ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
		Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
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
		t.Error("5 != 3 should be true")
	}
}

func TestEval_BinaryComparison(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	tests := []struct {
		name     string
		op       ast.BinaryOp
		left     string
		right    string
		expected bool
	}{
		{"3 < 5", ast.OpLess, "3", "5", true},
		{"5 < 3", ast.OpLess, "5", "3", false},
		{"3 <= 3", ast.OpLessOrEqual, "3", "3", true},
		{"5 > 3", ast.OpGreater, "5", "3", true},
		{"3 > 5", ast.OpGreater, "3", "5", false},
		{"5 >= 5", ast.OpGreaterOrEqual, "5", "5", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := &ast.BinaryExpression{
				Operator: tt.op,
				Left:     &ast.Literal{ValueType: ast.LiteralInteger, Value: tt.left},
				Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: tt.right},
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
				t.Errorf("%s = %v, want %v", tt.name, b.Bool(), tt.expected)
			}
		})
	}
}

func TestEval_LogicalAnd(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// true and false = false
	expr := &ast.BinaryExpression{
		Operator: ast.OpAnd,
		Left:     &ast.Literal{ValueType: ast.LiteralBoolean, Value: "true"},
		Right:    &ast.Literal{ValueType: ast.LiteralBoolean, Value: "false"},
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
		t.Error("true and false should be false")
	}
}

func TestEval_LogicalOr(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// false or true = true
	expr := &ast.BinaryExpression{
		Operator: ast.OpOr,
		Left:     &ast.Literal{ValueType: ast.LiteralBoolean, Value: "false"},
		Right:    &ast.Literal{ValueType: ast.LiteralBoolean, Value: "true"},
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
		t.Error("false or true should be true")
	}
}

func TestEval_UnaryNot(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.UnaryExpression{
		Operator: ast.OpNot,
		Operand:  &ast.Literal{ValueType: ast.LiteralBoolean, Value: "true"},
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
		t.Error("not true should be false")
	}
}

func TestEval_UnaryNegate(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.UnaryExpression{
		Operator: ast.OpNegate,
		Operand:  &ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, val, -5)
}

func TestEval_IfThenElse(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// if true then 1 else 2 → 1
	expr := &ast.IfThenElse{
		Condition: &ast.Literal{ValueType: ast.LiteralBoolean, Value: "true"},
		Then:      &ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
		Else:      &ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, val, 1)

	// if false then 1 else 2 → 2
	expr.Condition = &ast.Literal{ValueType: ast.LiteralBoolean, Value: "false"}
	val, err = ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, val, 2)
}

func TestEval_IdentifierRef(t *testing.T) {
	lib := &ast.Library{
		Statements: []*ast.ExpressionDef{
			{
				Name:       "MyValue",
				Expression: &ast.Literal{ValueType: ast.LiteralInteger, Value: "99"},
			},
		},
	}
	ctx := NewContext(context.Background(), lib)
	ev := NewEvaluator(ctx)

	// Evaluate all definitions first
	_, err := ev.EvaluateLibrary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Now reference should resolve
	val, err := ev.Eval(&ast.IdentifierRef{Name: "MyValue"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, val, 99)
}

func TestEvaluateLibrary(t *testing.T) {
	lib := &ast.Library{
		Statements: []*ast.ExpressionDef{
			{
				Name:       "X",
				Expression: &ast.Literal{ValueType: ast.LiteralInteger, Value: "10"},
			},
			{
				Name:       "Y",
				Expression: &ast.Literal{ValueType: ast.LiteralString, Value: "hello"},
			},
		},
	}
	ctx := NewContext(context.Background(), lib)
	ev := NewEvaluator(ctx)

	results, err := ev.EvaluateLibrary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	assertInteger(t, results["X"], 10)
	s, ok := results["Y"].(fptypes.String)
	if !ok {
		t.Fatalf("expected String for Y, got %T", results["Y"])
	}
	if s.Value() != "hello" {
		t.Errorf("Y = %q, want %q", s.Value(), "hello")
	}
}

func TestEvaluateExpression_ByName(t *testing.T) {
	lib := &ast.Library{
		Statements: []*ast.ExpressionDef{
			{
				Name:       "Result",
				Expression: &ast.Literal{ValueType: ast.LiteralInteger, Value: "42"},
			},
		},
	}
	ctx := NewContext(context.Background(), lib)
	ev := NewEvaluator(ctx)

	val, err := ev.EvaluateExpression("Result")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, val, 42)
}

func TestEvaluateExpression_NotFound(t *testing.T) {
	lib := &ast.Library{}
	ctx := NewContext(context.Background(), lib)
	ev := NewEvaluator(ctx)

	_, err := ev.EvaluateExpression("Missing")
	if err == nil {
		t.Fatal("expected error for missing expression")
	}
}

func TestEval_StringConcatenation(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.BinaryExpression{
		Operator: ast.OpConcatenate,
		Left:     &ast.Literal{ValueType: ast.LiteralString, Value: "hello"},
		Right:    &ast.Literal{ValueType: ast.LiteralString, Value: " world"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := val.(fptypes.String)
	if !ok {
		t.Fatalf("expected String, got %T", val)
	}
	if s.Value() != "hello world" {
		t.Errorf("got %q, want %q", s.Value(), "hello world")
	}
}

func TestEval_ListExpression(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.ListExpression{
		Elements: []ast.Expression{
			&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
			&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
			&ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
		},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val == nil {
		t.Fatal("expected non-nil list")
	}
}

func TestEval_IntervalExpression(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	expr := &ast.IntervalExpression{
		LowClosed:  true,
		HighClosed: true,
		Low:        &ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
		High:       &ast.Literal{ValueType: ast.LiteralInteger, Value: "10"},
	}
	val, err := ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val == nil {
		t.Fatal("expected non-nil interval")
	}
	if val.Type() != "Interval" {
		t.Errorf("type = %s, want Interval", val.Type())
	}
}

func TestEval_BetweenExpression(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	// 5 between 1 and 10 → true
	expr := &ast.BetweenExpression{
		Operand: &ast.Literal{ValueType: ast.LiteralInteger, Value: "5"},
		Low:     &ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
		High:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "10"},
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
		t.Error("5 between 1 and 10 should be true")
	}

	// 15 between 1 and 10 → false
	expr.Operand = &ast.Literal{ValueType: ast.LiteralInteger, Value: "15"}
	val, err = ev.Eval(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok = val.(fptypes.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", val)
	}
	if b.Bool() {
		t.Error("15 between 1 and 10 should be false")
	}
}

func TestEval_ExternalConstant(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.Parameters["MeasurementPeriod"] = fptypes.NewString("2024")
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.ExternalConstant{Name: "MeasurementPeriod"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := val.(fptypes.String)
	if !ok {
		t.Fatalf("expected String, got %T", val)
	}
	if s.Value() != "2024" {
		t.Errorf("got %q, want %q", s.Value(), "2024")
	}
}

func TestEval_ThisExpression(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.This = fptypes.NewInteger(42)
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.ThisExpression{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, val, 42)
}

func TestEval_NilExpression(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}

func TestContext_ChildScope(t *testing.T) {
	parent := NewContext(context.Background(), nil)
	parent.Parameters["x"] = fptypes.NewInteger(10)

	child := parent.ChildScope()
	child.Aliases["y"] = fptypes.NewString("hello")

	// Child should see parent params
	val, ok := child.ResolveIdentifier("x")
	if !ok {
		t.Fatal("child should resolve parent parameter")
	}
	assertInteger(t, val, 10)

	// Parent should NOT see child aliases
	_, ok = parent.ResolveIdentifier("y")
	if ok {
		t.Error("parent should not see child alias")
	}
}

func TestContext_ResolveValueSetURL(t *testing.T) {
	lib := &ast.Library{
		ValueSets: []*ast.ValueSetDef{
			{Name: "Diabetes", ID: "http://example.org/vs/diabetes"},
		},
	}
	ctx := NewContext(context.Background(), lib)

	url, ok := ctx.ResolveValueSetURL("Diabetes")
	if !ok {
		t.Fatal("should resolve Diabetes value set")
	}
	if url != "http://example.org/vs/diabetes" {
		t.Errorf("url = %q, want %q", url, "http://example.org/vs/diabetes")
	}

	_, ok = ctx.ResolveValueSetURL("Missing")
	if ok {
		t.Error("should not resolve missing value set")
	}
}

// ---------------------------------------------------------------------------
// Phase 2 — Nullological Operators
// ---------------------------------------------------------------------------

func TestIsNull(t *testing.T) {
	result := IsNull(nil)
	assertBoolean(t, result, true, "IsNull(nil)")

	result = IsNull(fptypes.NewInteger(1))
	assertBoolean(t, result, false, "IsNull(1)")
}

func TestIsTrue(t *testing.T) {
	result := IsTrue(fptypes.NewBoolean(true))
	assertBoolean(t, result, true, "IsTrue(true)")

	result = IsTrue(fptypes.NewBoolean(false))
	assertBoolean(t, result, false, "IsTrue(false)")

	result = IsTrue(nil)
	assertBoolean(t, result, false, "IsTrue(nil)")

	// Non-boolean → false
	result = IsTrue(fptypes.NewInteger(1))
	assertBoolean(t, result, false, "IsTrue(Integer)")
}

func TestIsFalse(t *testing.T) {
	result := IsFalse(fptypes.NewBoolean(false))
	assertBoolean(t, result, true, "IsFalse(false)")

	result = IsFalse(fptypes.NewBoolean(true))
	assertBoolean(t, result, false, "IsFalse(true)")

	result = IsFalse(nil)
	assertBoolean(t, result, false, "IsFalse(nil)")
}

func TestCoalesce(t *testing.T) {
	result := Coalesce(nil, nil, fptypes.NewInteger(42), fptypes.NewInteger(99))
	assertInteger(t, result, 42)

	result = Coalesce(nil, nil, nil)
	if result != nil {
		t.Error("Coalesce of all nils should be nil")
	}
}

func TestIfNull(t *testing.T) {
	result := IfNull(nil, fptypes.NewInteger(42))
	assertInteger(t, result, 42)

	result = IfNull(fptypes.NewInteger(10), fptypes.NewInteger(42))
	assertInteger(t, result, 10)
}

func TestThreeValuedAnd(t *testing.T) {
	tests := []struct {
		name     string
		left     fptypes.Value
		right    fptypes.Value
		expected *bool
	}{
		{"true AND true", fptypes.NewBoolean(true), fptypes.NewBoolean(true), boolPtr(true)},
		{"true AND false", fptypes.NewBoolean(true), fptypes.NewBoolean(false), boolPtr(false)},
		{"false AND true", fptypes.NewBoolean(false), fptypes.NewBoolean(true), boolPtr(false)},
		{"false AND false", fptypes.NewBoolean(false), fptypes.NewBoolean(false), boolPtr(false)},
		{"false AND null", fptypes.NewBoolean(false), nil, boolPtr(false)},
		{"null AND false", nil, fptypes.NewBoolean(false), boolPtr(false)},
		{"true AND null", fptypes.NewBoolean(true), nil, nil},
		{"null AND true", nil, fptypes.NewBoolean(true), nil},
		{"null AND null", nil, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ThreeValuedAnd(tt.left, tt.right)
			assertNullableBool(t, result, tt.expected, tt.name)
		})
	}
}

func TestThreeValuedOr(t *testing.T) {
	tests := []struct {
		name     string
		left     fptypes.Value
		right    fptypes.Value
		expected *bool
	}{
		{"true OR true", fptypes.NewBoolean(true), fptypes.NewBoolean(true), boolPtr(true)},
		{"true OR false", fptypes.NewBoolean(true), fptypes.NewBoolean(false), boolPtr(true)},
		{"false OR true", fptypes.NewBoolean(false), fptypes.NewBoolean(true), boolPtr(true)},
		{"false OR false", fptypes.NewBoolean(false), fptypes.NewBoolean(false), boolPtr(false)},
		{"true OR null", fptypes.NewBoolean(true), nil, boolPtr(true)},
		{"null OR true", nil, fptypes.NewBoolean(true), boolPtr(true)},
		{"false OR null", fptypes.NewBoolean(false), nil, nil},
		{"null OR false", nil, fptypes.NewBoolean(false), nil},
		{"null OR null", nil, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ThreeValuedOr(tt.left, tt.right)
			assertNullableBool(t, result, tt.expected, tt.name)
		})
	}
}

func TestThreeValuedNot(t *testing.T) {
	result := ThreeValuedNot(fptypes.NewBoolean(true))
	assertBoolean(t, result, false, "NOT true")

	result = ThreeValuedNot(fptypes.NewBoolean(false))
	assertBoolean(t, result, true, "NOT false")

	result = ThreeValuedNot(nil)
	if result != nil {
		t.Error("NOT null should be null")
	}
}

func TestThreeValuedImplies(t *testing.T) {
	// false implies anything → true
	result := ThreeValuedImplies(fptypes.NewBoolean(false), fptypes.NewBoolean(false))
	assertBoolean(t, result, true, "false implies false")

	result = ThreeValuedImplies(fptypes.NewBoolean(false), nil)
	assertBoolean(t, result, true, "false implies null")

	// true implies true → true
	result = ThreeValuedImplies(fptypes.NewBoolean(true), fptypes.NewBoolean(true))
	assertBoolean(t, result, true, "true implies true")

	// true implies false → false
	result = ThreeValuedImplies(fptypes.NewBoolean(true), fptypes.NewBoolean(false))
	assertBoolean(t, result, false, "true implies false")

	// true implies null → null
	result = ThreeValuedImplies(fptypes.NewBoolean(true), nil)
	if result != nil {
		t.Error("true implies null should be null")
	}
}

// ---------------------------------------------------------------------------
// Retrieve with DateRange
// ---------------------------------------------------------------------------

type mockDataProvider struct {
	lastResourceType   string
	lastCodePath       string
	lastCodeComparator string
	lastCodes          interface{}
	lastDateRange      interface{}
	results            []json.RawMessage
	err                error
}

func (m *mockDataProvider) Retrieve(_ context.Context, resourceType, codePath, codeComparator string, codes, dateRange interface{}) ([]json.RawMessage, error) {
	m.lastResourceType = resourceType
	m.lastCodePath = codePath
	m.lastCodeComparator = codeComparator
	m.lastCodes = codes
	m.lastDateRange = dateRange
	return m.results, m.err
}

func TestEval_Retrieve_NoDateRange(t *testing.T) {
	dp := &mockDataProvider{
		results: []json.RawMessage{
			json.RawMessage(`{"resourceType":"Condition","id":"c1"}`),
		},
	}
	ctx := NewContext(context.Background(), nil)
	ctx.DataProvider = dp
	ev := NewEvaluator(ctx)

	retrieve := &ast.Retrieve{
		ResourceType: &ast.NamedType{Name: "Condition"},
	}
	val, err := ev.Eval(retrieve)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val == nil {
		t.Fatal("expected non-nil result")
	}
	if dp.lastDateRange != nil {
		t.Errorf("expected nil dateRange, got %v", dp.lastDateRange)
	}
	if dp.lastResourceType != "Condition" {
		t.Errorf("resourceType = %q, want Condition", dp.lastResourceType)
	}
}

func TestEval_Retrieve_WithDateRange(t *testing.T) {
	dp := &mockDataProvider{
		results: []json.RawMessage{
			json.RawMessage(`{"resourceType":"Condition","id":"c1"}`),
		},
	}
	ctx := NewContext(context.Background(), nil)
	ctx.DataProvider = dp
	ev := NewEvaluator(ctx)

	// Retrieve with an Interval date range expression
	retrieve := &ast.Retrieve{
		ResourceType: &ast.NamedType{Name: "Condition"},
		DatePath:     "onset",
		DateRange: &ast.IntervalExpression{
			LowClosed:  true,
			HighClosed: true,
			Low:        &ast.Literal{ValueType: ast.LiteralDate, Value: "2024-01-01"},
			High:       &ast.Literal{ValueType: ast.LiteralDate, Value: "2024-12-31"},
		},
	}
	val, err := ev.Eval(retrieve)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val == nil {
		t.Fatal("expected non-nil result")
	}
	// dateRange should be a cqltypes.Interval
	interval, ok := dp.lastDateRange.(cqltypes.Interval)
	if !ok {
		t.Fatalf("expected dateRange to be Interval, got %T", dp.lastDateRange)
	}
	if !interval.LowClosed {
		t.Error("expected LowClosed=true")
	}
	if !interval.HighClosed {
		t.Error("expected HighClosed=true")
	}
}

func TestEval_Retrieve_NilDataProvider(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	retrieve := &ast.Retrieve{
		ResourceType: &ast.NamedType{Name: "Condition"},
		DatePath:     "onset",
		DateRange: &ast.IntervalExpression{
			LowClosed: true, HighClosed: true,
			Low:  &ast.Literal{ValueType: ast.LiteralDate, Value: "2024-01-01"},
			High: &ast.Literal{ValueType: ast.LiteralDate, Value: "2024-12-31"},
		},
	}
	val, err := ev.Eval(retrieve)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return empty list when no DataProvider
	if val == nil {
		t.Fatal("expected non-nil empty list")
	}
}

// ---------------------------------------------------------------------------
// Phase 2 — Terminology Integration
// ---------------------------------------------------------------------------

type mockTerminologyProvider struct {
	inValueSet bool
}

func (m *mockTerminologyProvider) InValueSet(_ context.Context, code, system, valueSetURL string) (bool, error) {
	return m.inValueSet, nil
}

func TestEvalInValueSet(t *testing.T) {
	lib := &ast.Library{
		ValueSets: []*ast.ValueSetDef{
			{Name: "DiabetesCodes", ID: "http://example.org/vs/diabetes"},
		},
	}
	ctx := NewContext(context.Background(), lib)
	ctx.TerminologyProvider = &mockTerminologyProvider{inValueSet: true}
	ev := NewEvaluator(ctx)

	result, err := ev.evalInValueSet(
		fptypes.NewString("E11"),
		&ast.IdentifierRef{Name: "DiabetesCodes"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBoolean(t, result, true, "InValueSet(found)")
}

func TestEvalInValueSet_NotFound(t *testing.T) {
	lib := &ast.Library{
		ValueSets: []*ast.ValueSetDef{
			{Name: "DiabetesCodes", ID: "http://example.org/vs/diabetes"},
		},
	}
	ctx := NewContext(context.Background(), lib)
	ctx.TerminologyProvider = &mockTerminologyProvider{inValueSet: false}
	ev := NewEvaluator(ctx)

	result, err := ev.evalInValueSet(
		fptypes.NewString("ZZZZZ"),
		&ast.IdentifierRef{Name: "DiabetesCodes"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBoolean(t, result, false, "InValueSet(not found)")
}

func TestEvalInValueSet_MissingValueSet(t *testing.T) {
	ctx := NewContext(context.Background(), &ast.Library{})
	ev := NewEvaluator(ctx)

	_, err := ev.evalInValueSet(
		fptypes.NewString("E11"),
		&ast.IdentifierRef{Name: "NonExistent"},
	)
	if err == nil {
		t.Error("expected error for missing value set")
	}
}

func TestEvalInCodeSystem(t *testing.T) {
	lib := &ast.Library{
		CodeSystems: []*ast.CodeSystemDef{
			{Name: "LOINC", ID: "http://loinc.org"},
		},
	}
	ctx := NewContext(context.Background(), lib)
	ev := NewEvaluator(ctx)

	code := cqltypes.NewCode("http://loinc.org", "12345-6", "Test")
	result, err := ev.evalInCodeSystem(code, &ast.IdentifierRef{Name: "LOINC"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBoolean(t, result, true, "InCodeSystem(matching)")

	// Different system → false
	code2 := cqltypes.NewCode("http://snomed.info/sct", "123", "Test")
	result, err = ev.evalInCodeSystem(code2, &ast.IdentifierRef{Name: "LOINC"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBoolean(t, result, false, "InCodeSystem(different)")
}

func TestResolveCodeRef(t *testing.T) {
	lib := &ast.Library{
		CodeSystems: []*ast.CodeSystemDef{
			{Name: "LOINC", ID: "http://loinc.org"},
		},
		Codes: []*ast.CodeDef{
			{Name: "HeartRate", ID: "8867-4", Display: "Heart rate", System: "LOINC"},
		},
	}
	ctx := NewContext(context.Background(), lib)
	ev := NewEvaluator(ctx)

	result := ev.resolveCodeRef(cqltypes.NewCodeRef("HeartRate", ""))
	if result == nil {
		t.Fatal("expected non-nil resolved code")
	}
	code, ok := result.(cqltypes.Code)
	if !ok {
		t.Fatalf("expected Code, got %T", result)
	}
	if code.Code != "8867-4" {
		t.Errorf("code = %q, want %q", code.Code, "8867-4")
	}
	if code.System != "http://loinc.org" {
		t.Errorf("system = %q, want %q", code.System, "http://loinc.org")
	}
}

func TestResolveCodeRef_NotFound(t *testing.T) {
	ctx := NewContext(context.Background(), &ast.Library{})
	ev := NewEvaluator(ctx)

	result := ev.resolveCodeRef(cqltypes.NewCodeRef("Missing", ""))
	if result != nil {
		t.Error("resolveCodeRef of missing code should return nil")
	}
}

func TestExtractCodeComponents(t *testing.T) {
	// From Code type
	code := cqltypes.NewCode("http://loinc.org", "12345-6", "Test")
	system, codeVal := extractCodeComponents(code)
	if system != "http://loinc.org" {
		t.Errorf("system = %q, want http://loinc.org", system)
	}
	if codeVal != "12345-6" {
		t.Errorf("code = %q, want 12345-6", codeVal)
	}

	// From String type
	system, codeVal = extractCodeComponents(fptypes.NewString("E11"))
	if system != "" {
		t.Errorf("system from string should be empty, got %q", system)
	}
	if codeVal != "E11" {
		t.Errorf("code = %q, want E11", codeVal)
	}
}

// ---------------------------------------------------------------------------
// Phase 2 — Multi-Context
// ---------------------------------------------------------------------------

func TestContext_SetGetContextType(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.SetContextType(ContextPatient)

	if ctx.GetContextType() != ContextPatient {
		t.Errorf("context type = %s, want Patient", ctx.GetContextType())
	}
}

func TestContext_SwitchContext(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.SetContextType(ContextPatient)

	child := ctx.SwitchContext(ContextPractitioner, []byte(`{"resourceType":"Practitioner","id":"123"}`))
	if child.GetContextType() != ContextPractitioner {
		t.Errorf("child context type = %s, want Practitioner", child.GetContextType())
	}
	// Parent should not change
	if ctx.GetContextType() != ContextPatient {
		t.Errorf("parent context should remain Patient, got %s", ctx.GetContextType())
	}
}

func TestContext_GetContextSubjectID(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.SetContextResource("Patient", []byte(`{"resourceType":"Patient","id":"pat-123"}`))

	id := ctx.GetContextSubjectID()
	if id != "pat-123" {
		t.Errorf("subject ID = %q, want pat-123", id)
	}
}

func TestContext_GetContextSubjectID_Nil(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	id := ctx.GetContextSubjectID()
	if id != "" {
		t.Errorf("expected empty subject ID for nil context, got %q", id)
	}
}

func TestContext_IsUnfilteredContext(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	if ctx.IsUnfilteredContext() {
		t.Error("default context should not be unfiltered")
	}

	ctx.SetContextType(ContextUnfiltered)
	if !ctx.IsUnfilteredContext() {
		t.Error("Unfiltered context should be unfiltered")
	}
}

func TestContext_ChildScope_PropagatesContextType(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.SetContextType(ContextEncounter)
	ctx.SetContextResource("Encounter", []byte(`{"resourceType":"Encounter","id":"enc-1"}`))

	child := ctx.ChildScope()
	if child.GetContextType() != ContextEncounter {
		t.Errorf("child should inherit Encounter context, got %s", child.GetContextType())
	}
	if child.GetContextResourceType() != "Encounter" {
		t.Errorf("child should inherit resource type, got %s", child.GetContextResourceType())
	}
}

func TestEvalFunctionOverloads(t *testing.T) {
	lib := &ast.Library{
		Functions: []*ast.FunctionDef{
			{
				Name:     "Greet",
				Operands: []*ast.OperandDef{{Name: "name"}},
				Body: &ast.BinaryExpression{
					Operator: ast.OpConcatenate,
					Left:     &ast.Literal{ValueType: ast.LiteralString, Value: "Hello, "},
					Right:    &ast.IdentifierRef{Name: "name"},
				},
			},
			{
				Name:     "Greet",
				Operands: []*ast.OperandDef{},
				Body:     &ast.Literal{ValueType: ast.LiteralString, Value: "Hello, World"},
			},
		},
		Statements: []*ast.ExpressionDef{
			{
				Name: "WithArg",
				Expression: &ast.FunctionCall{
					Name:     "Greet",
					Operands: []ast.Expression{&ast.Literal{ValueType: ast.LiteralString, Value: "CQL"}},
				},
			},
			{
				Name: "NoArg",
				Expression: &ast.FunctionCall{
					Name:     "Greet",
					Operands: []ast.Expression{},
				},
			},
		},
	}

	ctx := NewContext(context.Background(), lib)
	evaluator := NewEvaluator(ctx)
	results, err := evaluator.EvaluateLibrary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wa, ok := results["WithArg"].(fptypes.String)
	if !ok || wa.Value() != "Hello, CQL" {
		t.Errorf("WithArg = %v, want 'Hello, CQL'", results["WithArg"])
	}

	na, ok := results["NoArg"].(fptypes.String)
	if !ok || na.Value() != "Hello, World" {
		t.Errorf("NoArg = %v, want 'Hello, World'", results["NoArg"])
	}
}

func TestEvalLibraryQualifiedFunctionCall(t *testing.T) {
	includedLib := &ast.Library{
		Functions: []*ast.FunctionDef{
			{
				Name:     "Double",
				Operands: []*ast.OperandDef{{Name: "x"}},
				Body: &ast.BinaryExpression{
					Operator: ast.OpMultiply,
					Left:     &ast.IdentifierRef{Name: "x"},
					Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
				},
			},
		},
	}

	mainLib := &ast.Library{
		Includes: []*ast.IncludeDef{
			{Name: "MyLib", Alias: "MyLib"},
		},
		Statements: []*ast.ExpressionDef{
			{
				Name: "Result",
				Expression: &ast.FunctionCall{
					Source:   &ast.IdentifierRef{Name: "MyLib"},
					Name:     "Double",
					Operands: []ast.Expression{&ast.Literal{ValueType: ast.LiteralInteger, Value: "21"}},
				},
			},
		},
	}

	ctx := NewContext(context.Background(), mainLib)
	ctx.IncludedLibraries["MyLib"] = includedLib
	evaluator := NewEvaluator(ctx)
	results, err := evaluator.EvaluateLibrary()
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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func assertInteger(t *testing.T, v fptypes.Value, expected int64) {
	t.Helper()
	if v == nil {
		t.Fatalf("got nil, want %d", expected)
	}
	i, ok := v.(fptypes.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T (%v)", v, v)
	}
	if i.Value() != expected {
		t.Errorf("got %d, want %d", i.Value(), expected)
	}
}

func assertBoolean(t *testing.T, v fptypes.Value, expected bool, label string) {
	t.Helper()
	if v == nil {
		t.Fatalf("%s: got nil", label)
	}
	b, ok := v.(fptypes.Boolean)
	if !ok {
		t.Fatalf("%s: expected Boolean, got %T (%v)", label, v, v)
	}
	if b.Bool() != expected {
		t.Errorf("%s = %v, want %v", label, b.Bool(), expected)
	}
}

func assertNullableBool(t *testing.T, v fptypes.Value, expected *bool, label string) {
	t.Helper()
	if expected == nil {
		if v != nil {
			t.Errorf("%s: expected null, got %v", label, v)
		}
		return
	}
	assertBoolean(t, v, *expected, label)
}

func boolPtr(b bool) *bool {
	return &b
}

func TestEval_ConvertsToBoolean(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	tests := []struct {
		name     string
		operand  ast.Expression
		wantNil  bool
		wantBool bool
	}{
		{"null", &ast.Literal{ValueType: ast.LiteralNull}, true, false},
		{"string_true", &ast.Literal{ValueType: ast.LiteralString, Value: "true"}, false, true},
		{"string_false", &ast.Literal{ValueType: ast.LiteralString, Value: "false"}, false, true},
		{"string_maybe", &ast.Literal{ValueType: ast.LiteralString, Value: "maybe"}, false, false},
		{"bool_true", &ast.Literal{ValueType: ast.LiteralBoolean, Value: "true"}, false, true},
		{"int_0", &ast.Literal{ValueType: ast.LiteralInteger, Value: "0"}, false, true},
		{"int_1", &ast.Literal{ValueType: ast.LiteralInteger, Value: "1"}, false, true},
		{"int_2", &ast.Literal{ValueType: ast.LiteralInteger, Value: "2"}, false, false},
		{"decimal_0.0", &ast.Literal{ValueType: ast.LiteralDecimal, Value: "0.0"}, false, true},
		{"decimal_1.0", &ast.Literal{ValueType: ast.LiteralDecimal, Value: "1.0"}, false, true},
		{"decimal_2.5", &ast.Literal{ValueType: ast.LiteralDecimal, Value: "2.5"}, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := ev.Eval(&ast.FunctionCall{
				Name:     "ConvertsToBoolean",
				Operands: []ast.Expression{tt.operand},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if val != nil {
					t.Fatalf("expected nil, got %v", val)
				}
				return
			}
			assertBoolean(t, val, tt.wantBool, tt.name)
		})
	}
}

func TestEval_ConvertsToString(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	tests := []struct {
		name     string
		operand  ast.Expression
		wantNil  bool
		wantBool bool
	}{
		{"null", &ast.Literal{ValueType: ast.LiteralNull}, true, false},
		{"bool_true", &ast.Literal{ValueType: ast.LiteralBoolean, Value: "true"}, false, true},
		{"int_42", &ast.Literal{ValueType: ast.LiteralInteger, Value: "42"}, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := ev.Eval(&ast.FunctionCall{
				Name:     "ConvertsToString",
				Operands: []ast.Expression{tt.operand},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if val != nil {
					t.Fatalf("expected nil, got %v", val)
				}
				return
			}
			assertBoolean(t, val, tt.wantBool, tt.name)
		})
	}
}

func TestEval_ConvertsToInteger(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	tests := []struct {
		name     string
		operand  ast.Expression
		wantNil  bool
		wantBool bool
	}{
		{"null", &ast.Literal{ValueType: ast.LiteralNull}, true, false},
		{"int_42", &ast.Literal{ValueType: ast.LiteralInteger, Value: "42"}, false, true},
		{"string_42", &ast.Literal{ValueType: ast.LiteralString, Value: "42"}, false, true},
		{"string_abc", &ast.Literal{ValueType: ast.LiteralString, Value: "abc"}, false, false},
		{"bool_true", &ast.Literal{ValueType: ast.LiteralBoolean, Value: "true"}, false, true},
		{"decimal_3.14", &ast.Literal{ValueType: ast.LiteralDecimal, Value: "3.14"}, false, false},
		{"overflow_32bit", &ast.Literal{ValueType: ast.LiteralString, Value: "2147483648"}, false, false},
		{"max_32bit", &ast.Literal{ValueType: ast.LiteralString, Value: "2147483647"}, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := ev.Eval(&ast.FunctionCall{
				Name:     "ConvertsToInteger",
				Operands: []ast.Expression{tt.operand},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if val != nil {
					t.Fatalf("expected nil, got %v", val)
				}
				return
			}
			assertBoolean(t, val, tt.wantBool, tt.name)
		})
	}
}

func TestEval_ConvertsToDecimal(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	tests := []struct {
		name     string
		operand  ast.Expression
		wantNil  bool
		wantBool bool
	}{
		{"null", &ast.Literal{ValueType: ast.LiteralNull}, true, false},
		{"decimal_3.14", &ast.Literal{ValueType: ast.LiteralDecimal, Value: "3.14"}, false, true},
		{"int_42", &ast.Literal{ValueType: ast.LiteralInteger, Value: "42"}, false, true},
		{"string_1.5", &ast.Literal{ValueType: ast.LiteralString, Value: "1.5"}, false, true},
		{"string_abc", &ast.Literal{ValueType: ast.LiteralString, Value: "abc"}, false, false},
		{"bool_true", &ast.Literal{ValueType: ast.LiteralBoolean, Value: "true"}, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := ev.Eval(&ast.FunctionCall{
				Name:     "ConvertsToDecimal",
				Operands: []ast.Expression{tt.operand},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if val != nil {
					t.Fatalf("expected nil, got %v", val)
				}
				return
			}
			assertBoolean(t, val, tt.wantBool, tt.name)
		})
	}
}

func TestEval_ConvertsToLong(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	tests := []struct {
		name     string
		operand  ast.Expression
		wantNil  bool
		wantBool bool
	}{
		{"null", &ast.Literal{ValueType: ast.LiteralNull}, true, false},
		{"int_42", &ast.Literal{ValueType: ast.LiteralInteger, Value: "42"}, false, true},
		{"string_max_int64", &ast.Literal{ValueType: ast.LiteralString, Value: "9223372036854775807"}, false, true},
		{"string_abc", &ast.Literal{ValueType: ast.LiteralString, Value: "abc"}, false, false},
		{"decimal_3.14", &ast.Literal{ValueType: ast.LiteralDecimal, Value: "3.14"}, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := ev.Eval(&ast.FunctionCall{
				Name:     "ConvertsToLong",
				Operands: []ast.Expression{tt.operand},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if val != nil {
					t.Fatalf("expected nil, got %v", val)
				}
				return
			}
			assertBoolean(t, val, tt.wantBool, tt.name)
		})
	}
}

func TestEval_ConvertsToQuantity(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	tests := []struct {
		name     string
		operand  ast.Expression
		wantNil  bool
		wantBool bool
	}{
		{"null", &ast.Literal{ValueType: ast.LiteralNull}, true, false},
		{"int_42", &ast.Literal{ValueType: ast.LiteralInteger, Value: "42"}, false, true},
		{"decimal_3.14", &ast.Literal{ValueType: ast.LiteralDecimal, Value: "3.14"}, false, true},
		{"string_invalid", &ast.Literal{ValueType: ast.LiteralString, Value: "not a quantity"}, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := ev.Eval(&ast.FunctionCall{
				Name:     "ConvertsToQuantity",
				Operands: []ast.Expression{tt.operand},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if val != nil {
					t.Fatalf("expected nil, got %v", val)
				}
				return
			}
			assertBoolean(t, val, tt.wantBool, tt.name)
		})
	}
}

func TestEval_ConvertsToDate(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	tests := []struct {
		name     string
		operand  ast.Expression
		wantNil  bool
		wantBool bool
	}{
		{"null", &ast.Literal{ValueType: ast.LiteralNull}, true, false},
		{"date_valid", &ast.Literal{ValueType: ast.LiteralDate, Value: "2024-01-15"}, false, true},
		{"string_valid", &ast.Literal{ValueType: ast.LiteralString, Value: "2024-01-15"}, false, true},
		{"string_invalid", &ast.Literal{ValueType: ast.LiteralString, Value: "not-a-date"}, false, false},
		{"datetime_to_date", &ast.Literal{ValueType: ast.LiteralDateTime, Value: "2024-01-15T10:00:00"}, false, true},
		{"int_42", &ast.Literal{ValueType: ast.LiteralInteger, Value: "42"}, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := ev.Eval(&ast.FunctionCall{
				Name:     "ConvertsToDate",
				Operands: []ast.Expression{tt.operand},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if val != nil {
					t.Fatalf("expected nil, got %v", val)
				}
				return
			}
			assertBoolean(t, val, tt.wantBool, tt.name)
		})
	}
}

func TestEval_ConvertsToDateTime(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	tests := []struct {
		name     string
		operand  ast.Expression
		wantNil  bool
		wantBool bool
	}{
		{"null", &ast.Literal{ValueType: ast.LiteralNull}, true, false},
		{"datetime_valid", &ast.Literal{ValueType: ast.LiteralDateTime, Value: "2024-01-15T10:30:00"}, false, true},
		{"date_valid", &ast.Literal{ValueType: ast.LiteralDate, Value: "2024-01-15"}, false, true},
		{"string_valid", &ast.Literal{ValueType: ast.LiteralString, Value: "2024-01-15T10:30:00"}, false, true},
		{"string_invalid", &ast.Literal{ValueType: ast.LiteralString, Value: "not-a-datetime"}, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := ev.Eval(&ast.FunctionCall{
				Name:     "ConvertsToDateTime",
				Operands: []ast.Expression{tt.operand},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if val != nil {
					t.Fatalf("expected nil, got %v", val)
				}
				return
			}
			assertBoolean(t, val, tt.wantBool, tt.name)
		})
	}
}

func TestEval_ConvertsToTime(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	tests := []struct {
		name     string
		operand  ast.Expression
		wantNil  bool
		wantBool bool
	}{
		{"null", &ast.Literal{ValueType: ast.LiteralNull}, true, false},
		{"time_valid", &ast.Literal{ValueType: ast.LiteralTime, Value: "10:30:00"}, false, true},
		{"string_valid", &ast.Literal{ValueType: ast.LiteralString, Value: "10:30:00"}, false, true},
		{"string_invalid", &ast.Literal{ValueType: ast.LiteralString, Value: "not-a-time"}, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := ev.Eval(&ast.FunctionCall{
				Name:     "ConvertsToTime",
				Operands: []ast.Expression{tt.operand},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if val != nil {
					t.Fatalf("expected nil, got %v", val)
				}
				return
			}
			assertBoolean(t, val, tt.wantBool, tt.name)
		})
	}
}

func TestEval_ConvertsToRatio(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ev := NewEvaluator(ctx)

	tests := []struct {
		name     string
		operand  ast.Expression
		wantNil  bool
		wantBool bool
	}{
		{"null", &ast.Literal{ValueType: ast.LiteralNull}, true, false},
		{"valid_ratio", &ast.Literal{ValueType: ast.LiteralString, Value: "1:128"}, false, true},
		{"invalid_words", &ast.Literal{ValueType: ast.LiteralString, Value: "hello:world"}, false, false},
		{"no_colon", &ast.Literal{ValueType: ast.LiteralString, Value: "abc"}, false, false},
		{"int_not_string", &ast.Literal{ValueType: ast.LiteralInteger, Value: "42"}, false, false},
		{"multiple_colons", &ast.Literal{ValueType: ast.LiteralString, Value: "1:2:3"}, false, false},
		{"inf_nan", &ast.Literal{ValueType: ast.LiteralString, Value: "Inf:NaN"}, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := ev.Eval(&ast.FunctionCall{
				Name:     "ConvertsToRatio",
				Operands: []ast.Expression{tt.operand},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if val != nil {
					t.Fatalf("expected nil, got %v", val)
				}
				return
			}
			assertBoolean(t, val, tt.wantBool, tt.name)
		})
	}
}

// TestEval_IdentifierRef_LazyEvaluation verifies that IdentifierRef lazily evaluates
// library expression definitions without requiring EvaluateLibrary() to be called first.
// This is the core fix for the $evaluate-measure numerator=0 bug where expressions like
// "Needs Follow Up" = "In Demographic" and "Has Hypertension" failed to resolve references.
func TestEval_IdentifierRef_LazyEvaluation(t *testing.T) {
	lib := &ast.Library{
		Statements: []*ast.ExpressionDef{
			{
				Name:       "In Demographic",
				Expression: &ast.Literal{ValueType: ast.LiteralBoolean, Value: "true"},
			},
			{
				Name:       "Has Hypertension",
				Expression: &ast.Literal{ValueType: ast.LiteralBoolean, Value: "true"},
			},
			{
				Name: "Needs Follow Up",
				Expression: &ast.BinaryExpression{
					Operator: ast.OpAnd,
					Left:     &ast.IdentifierRef{Name: "In Demographic"},
					Right:    &ast.IdentifierRef{Name: "Has Hypertension"},
				},
			},
		},
	}
	ctx := NewContext(context.Background(), lib)
	ev := NewEvaluator(ctx)

	// Evaluate "Needs Follow Up" directly — without calling EvaluateLibrary() first.
	// The IdentifierRef nodes for "In Demographic" and "Has Hypertension" must be
	// lazily evaluated from the library's statement definitions.
	val, err := ev.EvaluateExpression("Needs Follow Up")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBoolean(t, val, true, "Needs Follow Up")

	// Verify both referenced definitions were cached
	if _, ok := ctx.Definitions["In Demographic"]; !ok {
		t.Error("expected 'In Demographic' to be cached in Definitions")
	}
	if _, ok := ctx.Definitions["Has Hypertension"]; !ok {
		t.Error("expected 'Has Hypertension' to be cached in Definitions")
	}
}

// TestEval_IdentifierRef_LazyEvaluation_FalseCase verifies lazy evaluation with mixed results.
func TestEval_IdentifierRef_LazyEvaluation_FalseCase(t *testing.T) {
	lib := &ast.Library{
		Statements: []*ast.ExpressionDef{
			{
				Name:       "A",
				Expression: &ast.Literal{ValueType: ast.LiteralBoolean, Value: "true"},
			},
			{
				Name:       "B",
				Expression: &ast.Literal{ValueType: ast.LiteralBoolean, Value: "false"},
			},
			{
				Name: "C",
				Expression: &ast.BinaryExpression{
					Operator: ast.OpAnd,
					Left:     &ast.IdentifierRef{Name: "A"},
					Right:    &ast.IdentifierRef{Name: "B"},
				},
			},
		},
	}
	ctx := NewContext(context.Background(), lib)
	ev := NewEvaluator(ctx)

	val, err := ev.EvaluateExpression("C")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBoolean(t, val, false, "C = A and B")
}

func TestEvalMemberAccess_ChoiceType(t *testing.T) {
	obsJSON := []byte(`{
		"resourceType": "Observation",
		"status": "final",
		"valueQuantity": {
			"value": 128,
			"unit": "cm"
		}
	}`)

	obj := fptypes.NewObjectValue(obsJSON)

	lib := &ast.Library{
		Statements: []*ast.ExpressionDef{
			{
				Name: "ObsValue",
				Expression: &ast.MemberAccess{
					Source: &ast.IdentifierRef{Name: "obs"},
					Member: "value",
				},
			},
		},
	}

	ctx := NewContext(context.Background(), lib)
	ctx.ModelInfo = model.DefaultR4ModelInfo()
	ctx.Aliases["obs"] = obj
	evaluator := NewEvaluator(ctx)
	results, err := evaluator.EvaluateLibrary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val := results["ObsValue"]
	if val == nil {
		t.Fatal("ObsValue should not be nil — value[x] resolution failed")
	}
	if _, ok := val.(*fptypes.ObjectValue); !ok {
		t.Errorf("ObsValue: expected *ObjectValue, got %T", val)
	}
}

func TestEvalMemberAccess_DirectField(t *testing.T) {
	obsJSON := []byte(`{"resourceType": "Observation", "status": "final"}`)
	obj := fptypes.NewObjectValue(obsJSON)

	lib := &ast.Library{
		Statements: []*ast.ExpressionDef{
			{
				Name: "Status",
				Expression: &ast.MemberAccess{
					Source: &ast.IdentifierRef{Name: "obs"},
					Member: "status",
				},
			},
		},
	}

	ctx := NewContext(context.Background(), lib)
	ctx.ModelInfo = model.DefaultR4ModelInfo()
	ctx.Aliases["obs"] = obj
	evaluator := NewEvaluator(ctx)
	results, err := evaluator.EvaluateLibrary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s, ok := results["Status"].(fptypes.String)
	if !ok || s.Value() != "final" {
		t.Errorf("Status = %v, want 'final'", results["Status"])
	}
}

func TestResolveOverload_ByArgumentType(t *testing.T) {
	// Two overloads with same arity but different operand types
	lib := &ast.Library{
		Functions: []*ast.FunctionDef{
			{
				Name:     "Convert",
				Operands: []*ast.OperandDef{{Name: "val", Type: &ast.NamedType{Name: "Integer"}}},
				Body:     &ast.Literal{ValueType: ast.LiteralString, Value: "integer-path"},
			},
			{
				Name:     "Convert",
				Operands: []*ast.OperandDef{{Name: "val", Type: &ast.NamedType{Name: "String"}}},
				Body:     &ast.Literal{ValueType: ast.LiteralString, Value: "string-path"},
			},
		},
		Statements: []*ast.ExpressionDef{
			{
				Name: "FromString",
				Expression: &ast.FunctionCall{
					Name:     "Convert",
					Operands: []ast.Expression{&ast.Literal{ValueType: ast.LiteralString, Value: "hello"}},
				},
			},
			{
				Name: "FromInt",
				Expression: &ast.FunctionCall{
					Name:     "Convert",
					Operands: []ast.Expression{&ast.Literal{ValueType: ast.LiteralInteger, Value: "42"}},
				},
			},
		},
	}

	ctx := NewContext(context.Background(), lib)
	evaluator := NewEvaluator(ctx)
	results, err := evaluator.EvaluateLibrary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fs, ok := results["FromString"].(fptypes.String)
	if !ok || fs.Value() != "string-path" {
		t.Errorf("FromString = %v, want 'string-path'", results["FromString"])
	}

	fi, ok := results["FromInt"].(fptypes.String)
	if !ok || fi.Value() != "integer-path" {
		t.Errorf("FromInt = %v, want 'integer-path'", results["FromInt"])
	}
}
