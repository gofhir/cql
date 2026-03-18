package eval

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/funcs"
	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
)

// Evaluator interprets CQL AST nodes.
type Evaluator struct {
	ctx   *Context
	funcs map[string]*ast.FunctionDef // registered library functions
}

// NewEvaluator creates a new evaluator for the given context.
func NewEvaluator(ctx *Context) *Evaluator {
	e := &Evaluator{
		ctx:   ctx,
		funcs: make(map[string]*ast.FunctionDef),
	}
	// Register library functions
	if ctx.Library != nil {
		for _, f := range ctx.Library.Functions {
			e.funcs[f.Name] = f
		}
	}
	return e
}

// withContext returns a lightweight evaluator sharing the same function registry
// but using a different context. Avoids re-building the funcs map on each iteration.
func (e *Evaluator) withContext(ctx *Context) *Evaluator {
	return &Evaluator{ctx: ctx, funcs: e.funcs}
}

// EvaluateLibrary evaluates all expression definitions in the library.
func (e *Evaluator) EvaluateLibrary() (map[string]fptypes.Value, error) {
	if e.ctx.Library == nil {
		return nil, fmt.Errorf("no library to evaluate")
	}
	results := make(map[string]fptypes.Value)
	for _, stmt := range e.ctx.Library.Statements {
		val, err := e.Eval(stmt.Expression)
		if err != nil {
			return nil, fmt.Errorf("error evaluating '%s': %w", stmt.Name, err)
		}
		e.ctx.Definitions[stmt.Name] = val
		results[stmt.Name] = val
	}
	return results, nil
}

// EvaluateExpression evaluates a named expression by name.
func (e *Evaluator) EvaluateExpression(name string) (fptypes.Value, error) {
	// Check if already evaluated
	if val, ok := e.ctx.Definitions[name]; ok {
		return val, nil
	}
	// Find the expression definition
	if e.ctx.Library != nil {
		for _, stmt := range e.ctx.Library.Statements {
			if stmt.Name == name {
				val, err := e.Eval(stmt.Expression)
				if err != nil {
					return nil, err
				}
				e.ctx.Definitions[name] = val
				return val, nil
			}
		}
	}
	return nil, fmt.Errorf("expression '%s' not found", name)
}

// Eval evaluates a single AST expression node and returns a Value.
func (e *Evaluator) Eval(expr ast.Expression) (result fptypes.Value, err error) {
	if expr == nil {
		return nil, nil
	}
	if tl := e.ctx.TraceListener; tl != nil {
		tl.OnEnter(expr)
		defer func() { tl.OnExit(expr, result, err) }()
	}
	switch n := expr.(type) {
	case *ast.Literal:
		return e.evalLiteral(n)
	case *ast.IdentifierRef:
		return e.evalIdentifierRef(n)
	case *ast.BinaryExpression:
		return e.evalBinary(n)
	case *ast.UnaryExpression:
		return e.evalUnary(n)
	case *ast.IfThenElse:
		return e.evalIfThenElse(n)
	case *ast.CaseExpression:
		return e.evalCase(n)
	case *ast.IsExpression:
		return e.evalIs(n)
	case *ast.AsExpression:
		return e.evalAs(n)
	case *ast.BooleanTestExpression:
		return e.evalBooleanTest(n)
	case *ast.FunctionCall:
		return e.evalFunctionCall(n)
	case *ast.MemberAccess:
		return e.evalMemberAccess(n)
	case *ast.IndexAccess:
		return e.evalIndexAccess(n)
	case *ast.Retrieve:
		return e.evalRetrieve(n)
	case *ast.Query:
		return e.evalQuery(n)
	case *ast.IntervalExpression:
		return e.evalIntervalExpr(n)
	case *ast.TupleExpression:
		return e.evalTupleExpr(n)
	case *ast.ListExpression:
		return e.evalListExpr(n)
	case *ast.CodeExpression:
		return e.evalCodeExpr(n)
	case *ast.ConceptExpression:
		return e.evalConceptExpr(n)
	case *ast.ExternalConstant:
		return e.evalExternalConstant(n)
	case *ast.ThisExpression:
		return e.ctx.This, nil
	case *ast.IndexExpression:
		return fptypes.NewInteger(int64(e.ctx.Index)), nil
	case *ast.TotalExpression:
		return e.ctx.Total, nil
	case *ast.MembershipExpression:
		return e.evalMembership(n)
	case *ast.BetweenExpression:
		return e.evalBetween(n)
	case *ast.DurationBetween:
		return e.evalDurationBetween(n)
	case *ast.DifferenceBetween:
		return e.evalDifferenceBetween(n)
	case *ast.DateTimeComponentFrom:
		return e.evalDateTimeComponentFrom(n)
	case *ast.ConvertExpression:
		return e.evalConvert(n)
	case *ast.CastExpression:
		return e.evalCast(n)
	case *ast.TypeExtent:
		return e.evalTypeExtent(n)
	case *ast.InstanceExpression:
		return e.evalInstanceExpr(n)
	case *ast.TimingExpression:
		return e.evalTimingExpr(n)
	case *ast.SetAggregateExpression:
		return e.evalSetAggregate(n)
	case *ast.DurationOf:
		operand, err := e.Eval(n.Operand)
		if err != nil {
			return nil, err
		}
		if iv, ok := operand.(cqltypes.Interval); ok {
			return funcs.DurationBetween(iv.Low, iv.High, n.Precision)
		}
		return nil, nil
	case *ast.DifferenceOf:
		operand, err := e.Eval(n.Operand)
		if err != nil {
			return nil, err
		}
		if iv, ok := operand.(cqltypes.Interval); ok {
			return funcs.DifferenceBetween(iv.Low, iv.High, n.Precision)
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// ---------------------------------------------------------------------------
// Literal evaluation
// ---------------------------------------------------------------------------

func (e *Evaluator) evalLiteral(n *ast.Literal) (fptypes.Value, error) {
	switch n.ValueType {
	case ast.LiteralNull:
		return nil, nil
	case ast.LiteralBoolean:
		return fptypes.NewBoolean(n.Value == "true"), nil
	case ast.LiteralString:
		return fptypes.NewString(n.Value), nil
	case ast.LiteralInteger:
		v, err := strconv.ParseInt(n.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid integer: %s", n.Value)
		}
		return fptypes.NewInteger(v), nil
	case ast.LiteralLong:
		v, err := strconv.ParseInt(n.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid long: %s", n.Value)
		}
		return fptypes.NewInteger(v), nil
	case ast.LiteralDecimal:
		return fptypes.NewDecimal(n.Value)
	case ast.LiteralDate:
		return fptypes.NewDate(n.Value)
	case ast.LiteralDateTime:
		return fptypes.NewDateTime(n.Value)
	case ast.LiteralTime:
		return fptypes.NewTime(n.Value)
	case ast.LiteralQuantity:
		return fptypes.NewQuantity(n.Value)
	default:
		return fptypes.NewString(n.Value), nil
	}
}

// ---------------------------------------------------------------------------
// Identifier resolution
// ---------------------------------------------------------------------------

func (e *Evaluator) evalIdentifierRef(n *ast.IdentifierRef) (fptypes.Value, error) { //nolint:unparam // error is part of the eval interface
	val, ok := e.ctx.ResolveIdentifier(n.Name)
	if ok {
		return val, nil
	}
	// Could be a resource type name used in query context
	return fptypes.NewString(n.Name), nil
}

// ---------------------------------------------------------------------------
// Binary operators
// ---------------------------------------------------------------------------

func (e *Evaluator) evalBinary(n *ast.BinaryExpression) (fptypes.Value, error) {
	// Short-circuit for logical operators
	switch n.Operator {
	case ast.OpAnd:
		left, err := e.Eval(n.Left)
		if err != nil {
			return nil, err
		}
		if isFalse(left) {
			return fptypes.NewBoolean(false), nil
		}
		right, err := e.Eval(n.Right)
		if err != nil {
			return nil, err
		}
		if left == nil || right == nil {
			if isFalse(right) {
				return fptypes.NewBoolean(false), nil
			}
			return nil, nil
		}
		return fptypes.NewBoolean(isTrue(left) && isTrue(right)), nil

	case ast.OpOr:
		left, err := e.Eval(n.Left)
		if err != nil {
			return nil, err
		}
		if isTrue(left) {
			return fptypes.NewBoolean(true), nil
		}
		right, err := e.Eval(n.Right)
		if err != nil {
			return nil, err
		}
		if left == nil || right == nil {
			if isTrue(right) {
				return fptypes.NewBoolean(true), nil
			}
			return nil, nil
		}
		return fptypes.NewBoolean(isTrue(left) || isTrue(right)), nil

	case ast.OpImplies:
		left, err := e.Eval(n.Left)
		if err != nil {
			return nil, err
		}
		if isFalse(left) {
			return fptypes.NewBoolean(true), nil
		}
		right, err := e.Eval(n.Right)
		if err != nil {
			return nil, err
		}
		if isTrue(left) {
			if right == nil {
				return nil, nil
			}
			return fptypes.NewBoolean(isTrue(right)), nil
		}
		// left is null
		if isTrue(right) {
			return fptypes.NewBoolean(true), nil
		}
		return nil, nil
	}

	left, err := e.Eval(n.Left)
	if err != nil {
		return nil, err
	}
	right, err := e.Eval(n.Right)
	if err != nil {
		return nil, err
	}

	// Null propagation for most operators
	if left == nil || right == nil {
		switch n.Operator {
		case ast.OpEqual, ast.OpNotEqual, ast.OpLess, ast.OpLessOrEqual, ast.OpGreater, ast.OpGreaterOrEqual:
			return nil, nil
		case ast.OpEquivalent:
			return fptypes.NewBoolean(left == nil && right == nil), nil
		case ast.OpNotEquivalent:
			return fptypes.NewBoolean(left != nil || right != nil), nil
		case ast.OpUnion:
			if left == nil {
				return right, nil
			}
			return left, nil
		case ast.OpConcatenate:
			ls, rs := "", ""
			if left != nil {
				ls = left.String()
			}
			if right != nil {
				rs = right.String()
			}
			return fptypes.NewString(ls + rs), nil
		default:
			return nil, nil
		}
	}

	switch n.Operator {
	case ast.OpEqual:
		return fptypes.NewBoolean(left.Equal(right)), nil
	case ast.OpNotEqual:
		return fptypes.NewBoolean(!left.Equal(right)), nil
	case ast.OpEquivalent:
		return fptypes.NewBoolean(left.Equivalent(right)), nil
	case ast.OpNotEquivalent:
		return fptypes.NewBoolean(!left.Equivalent(right)), nil

	case ast.OpLess, ast.OpLessOrEqual, ast.OpGreater, ast.OpGreaterOrEqual:
		lc, ok := left.(fptypes.Comparable)
		if !ok {
			return nil, fmt.Errorf("cannot compare %s", left.Type())
		}
		cmp, err := lc.Compare(right)
		if err != nil {
			return nil, err
		}
		switch n.Operator {
		case ast.OpLess:
			return fptypes.NewBoolean(cmp < 0), nil
		case ast.OpLessOrEqual:
			return fptypes.NewBoolean(cmp <= 0), nil
		case ast.OpGreater:
			return fptypes.NewBoolean(cmp > 0), nil
		case ast.OpGreaterOrEqual:
			return fptypes.NewBoolean(cmp >= 0), nil
		}

	case ast.OpAdd, ast.OpSubtract, ast.OpMultiply, ast.OpDivide, ast.OpDiv, ast.OpMod, ast.OpPower:
		return e.evalArithmetic(n.Operator, left, right)

	case ast.OpConcatenate:
		return fptypes.NewString(left.String() + right.String()), nil

	case ast.OpXor:
		return fptypes.NewBoolean(isTrue(left) != isTrue(right)), nil

	case ast.OpUnion:
		return e.evalSetOp(n.Operator, left, right)
	case ast.OpIntersect:
		return e.evalSetOp(n.Operator, left, right)
	case ast.OpExcept:
		return e.evalSetOp(n.Operator, left, right)

	case ast.OpIn, ast.OpContains:
		return e.evalInContains(n.Operator, left, right)
	}

	return nil, fmt.Errorf("unsupported binary operator: %d", n.Operator)
}

// ---------------------------------------------------------------------------
// Arithmetic
// ---------------------------------------------------------------------------

func (e *Evaluator) evalArithmetic(op ast.BinaryOp, left, right fptypes.Value) (fptypes.Value, error) {
	// Try integer arithmetic first
	li, liOk := left.(fptypes.Integer)
	ri, riOk := right.(fptypes.Integer)
	if liOk && riOk {
		lv, rv := li.Value(), ri.Value()
		switch op {
		case ast.OpAdd:
			return fptypes.NewInteger(lv + rv), nil
		case ast.OpSubtract:
			return fptypes.NewInteger(lv - rv), nil
		case ast.OpMultiply:
			return fptypes.NewInteger(lv * rv), nil
		case ast.OpDivide:
			if rv == 0 {
				return nil, nil // CQL: divide by zero returns null
			}
			return newDecimalFromD(decimal.NewFromInt(lv).Div(decimal.NewFromInt(rv))), nil
		case ast.OpDiv:
			if rv == 0 {
				return nil, nil
			}
			return fptypes.NewInteger(lv / rv), nil
		case ast.OpMod:
			if rv == 0 {
				return nil, nil
			}
			return fptypes.NewInteger(lv % rv), nil
		case ast.OpPower:
			return fptypes.NewInteger(int64(math.Pow(float64(lv), float64(rv)))), nil
		}
	}

	// Fall back to decimal arithmetic
	ld := toDecimal(left)
	rd := toDecimal(right)
	if ld.IsZero() && !liOk && !isDecimal(left) {
		return nil, fmt.Errorf("cannot perform arithmetic on %s", left.Type())
	}

	switch op {
	case ast.OpAdd:
		return newDecimalFromD(ld.Add(rd)), nil
	case ast.OpSubtract:
		return newDecimalFromD(ld.Sub(rd)), nil
	case ast.OpMultiply:
		return newDecimalFromD(ld.Mul(rd)), nil
	case ast.OpDivide:
		if rd.IsZero() {
			return nil, nil
		}
		return newDecimalFromD(ld.Div(rd)), nil
	case ast.OpDiv:
		if rd.IsZero() {
			return nil, nil
		}
		return fptypes.NewInteger(ld.Div(rd).IntPart()), nil
	case ast.OpMod:
		if rd.IsZero() {
			return nil, nil
		}
		return newDecimalFromD(ld.Mod(rd)), nil
	case ast.OpPower:
		f, _ := ld.Float64()
		p, _ := rd.Float64()
		return fptypes.NewDecimalFromFloat(math.Pow(f, p)), nil
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Set operations
// ---------------------------------------------------------------------------

func (e *Evaluator) evalSetOp(op ast.BinaryOp, left, right fptypes.Value) (fptypes.Value, error) {
	lc := toCollection(left)
	rc := toCollection(right)
	switch op {
	case ast.OpUnion:
		return cqltypes.NewList(lc.Union(rc)), nil
	case ast.OpIntersect:
		return cqltypes.NewList(lc.Intersect(rc)), nil
	case ast.OpExcept:
		return cqltypes.NewList(lc.Exclude(rc)), nil
	}
	return nil, nil
}

func (e *Evaluator) evalInContains(op ast.BinaryOp, left, right fptypes.Value) (fptypes.Value, error) {
	if op == ast.OpIn {
		// left in right: check if left is in the right collection/interval
		if interval, ok := right.(cqltypes.Interval); ok {
			result, err := interval.Contains(left)
			if err != nil {
				return nil, err
			}
			return fptypes.NewBoolean(result), nil
		}
		rc := toCollection(right)
		return fptypes.NewBoolean(rc.Contains(left)), nil
	}
	// contains: right in left
	if interval, ok := left.(cqltypes.Interval); ok {
		result, err := interval.Contains(right)
		if err != nil {
			return nil, err
		}
		return fptypes.NewBoolean(result), nil
	}
	lc := toCollection(left)
	return fptypes.NewBoolean(lc.Contains(right)), nil
}

// ---------------------------------------------------------------------------
// Unary operators
// ---------------------------------------------------------------------------

func (e *Evaluator) evalUnary(n *ast.UnaryExpression) (fptypes.Value, error) {
	operand, err := e.Eval(n.Operand)
	if err != nil {
		return nil, err
	}

	switch n.Operator {
	case ast.OpNot:
		if operand == nil {
			return nil, nil
		}
		return fptypes.NewBoolean(!isTrue(operand)), nil
	case ast.OpExists:
		if operand == nil {
			return fptypes.NewBoolean(false), nil
		}
		if list, ok := operand.(cqltypes.List); ok {
			return fptypes.NewBoolean(!list.IsEmpty()), nil
		}
		return fptypes.NewBoolean(true), nil
	case ast.OpNegate:
		if operand == nil {
			return nil, nil
		}
		if i, ok := operand.(fptypes.Integer); ok {
			return fptypes.NewInteger(-i.Value()), nil
		}
		d := toDecimal(operand)
		return newDecimalFromD(d.Neg()), nil
	case ast.OpPositive:
		return operand, nil
	case ast.OpDistinct:
		c := toCollection(operand)
		return cqltypes.NewList(c.Distinct()), nil
	case ast.OpFlatten:
		return e.evalFlatten(operand), nil
	case ast.OpSingletonFrom:
		c := toCollection(operand)
		if c.Count() == 0 {
			return nil, nil
		}
		if c.Count() == 1 {
			return c[0], nil
		}
		return nil, fmt.Errorf("singleton from requires 0 or 1 elements, got %d", c.Count())
	case ast.OpStartOf:
		if iv, ok := operand.(cqltypes.Interval); ok {
			return iv.Low, nil
		}
		return nil, nil
	case ast.OpEndOf:
		if iv, ok := operand.(cqltypes.Interval); ok {
			return iv.High, nil
		}
		return nil, nil
	case ast.OpWidthOf:
		if iv, ok := operand.(cqltypes.Interval); ok {
			return funcs.IntervalWidth(iv)
		}
		return nil, nil
	case ast.OpSuccessorOf, ast.OpPredecessorOf:
		return e.evalSuccessorPredecessor(n.Operator, operand)
	case ast.OpPointFrom:
		if iv, ok := operand.(cqltypes.Interval); ok {
			if iv.Low != nil && iv.High != nil && iv.Low.Equal(iv.High) {
				return iv.Low, nil
			}
		}
		return nil, fmt.Errorf("point from requires a unit interval")
	}
	return nil, fmt.Errorf("unsupported unary operator: %d", n.Operator)
}

func (e *Evaluator) evalFlatten(val fptypes.Value) fptypes.Value {
	c := toCollection(val)
	result := make(fptypes.Collection, 0, len(c))
	for _, item := range c {
		if list, ok := item.(cqltypes.List); ok {
			result = append(result, list.Values...)
		} else {
			result = append(result, item)
		}
	}
	return cqltypes.NewList(result)
}

func (e *Evaluator) evalSuccessorPredecessor(op ast.UnaryOp, operand fptypes.Value) (fptypes.Value, error) {
	if operand == nil {
		return nil, nil
	}
	if i, ok := operand.(fptypes.Integer); ok {
		if op == ast.OpSuccessorOf {
			return fptypes.NewInteger(i.Value() + 1), nil
		}
		return fptypes.NewInteger(i.Value() - 1), nil
	}
	d := toDecimal(operand)
	if isDecimal(operand) {
		epsilon := decimal.NewFromFloat(1e-8)
		if op == ast.OpSuccessorOf {
			return newDecimalFromD(d.Add(epsilon)), nil
		}
		return newDecimalFromD(d.Sub(epsilon)), nil
	}
	return nil, fmt.Errorf("successor/predecessor not supported for %s", operand.Type())
}

// ---------------------------------------------------------------------------
// Conditional
// ---------------------------------------------------------------------------

func (e *Evaluator) evalIfThenElse(n *ast.IfThenElse) (fptypes.Value, error) {
	cond, err := e.Eval(n.Condition)
	if err != nil {
		return nil, err
	}
	if isTrue(cond) {
		return e.Eval(n.Then)
	}
	return e.Eval(n.Else)
}

func (e *Evaluator) evalCase(n *ast.CaseExpression) (fptypes.Value, error) {
	if n.Comparand != nil {
		comp, err := e.Eval(n.Comparand)
		if err != nil {
			return nil, err
		}
		for _, item := range n.Items {
			when, err := e.Eval(item.When)
			if err != nil {
				return nil, err
			}
			if comp != nil && when != nil && comp.Equal(when) {
				return e.Eval(item.Then)
			}
		}
	} else {
		for _, item := range n.Items {
			when, err := e.Eval(item.When)
			if err != nil {
				return nil, err
			}
			if isTrue(when) {
				return e.Eval(item.Then)
			}
		}
	}
	return e.Eval(n.Else)
}

// ---------------------------------------------------------------------------
// Type operations
// ---------------------------------------------------------------------------

func (e *Evaluator) evalIs(n *ast.IsExpression) (fptypes.Value, error) {
	operand, err := e.Eval(n.Operand)
	if err != nil {
		return nil, err
	}
	if operand == nil {
		return fptypes.NewBoolean(false), nil
	}
	nt, ok := n.Type.(*ast.NamedType)
	if !ok {
		return fptypes.NewBoolean(false), nil
	}
	return fptypes.NewBoolean(strings.EqualFold(operand.Type(), nt.Name)), nil
}

func (e *Evaluator) evalAs(n *ast.AsExpression) (fptypes.Value, error) {
	operand, err := e.Eval(n.Operand)
	if err != nil {
		return nil, err
	}
	if operand == nil {
		return nil, nil
	}
	nt, ok := n.Type.(*ast.NamedType)
	if !ok {
		return nil, nil
	}
	if strings.EqualFold(operand.Type(), nt.Name) {
		return operand, nil
	}
	return nil, nil // safe cast returns null
}

func (e *Evaluator) evalBooleanTest(n *ast.BooleanTestExpression) (fptypes.Value, error) {
	operand, err := e.Eval(n.Operand)
	if err != nil {
		return nil, err
	}
	var result bool
	switch n.TestValue {
	case "null":
		result = operand == nil
	case "true":
		result = isTrue(operand)
	case "false":
		result = isFalse(operand)
	}
	if n.Not {
		result = !result
	}
	return fptypes.NewBoolean(result), nil
}

func (e *Evaluator) evalConvert(n *ast.ConvertExpression) (fptypes.Value, error) {
	operand, err := e.Eval(n.Operand)
	if err != nil {
		return nil, err
	}
	if operand == nil {
		return nil, nil
	}
	if n.ToType != nil {
		if nt, ok := n.ToType.(*ast.NamedType); ok {
			return convertToType(operand, nt.Name)
		}
	}
	return operand, nil
}

func (e *Evaluator) evalCast(n *ast.CastExpression) (fptypes.Value, error) {
	operand, err := e.Eval(n.Operand)
	if err != nil {
		return nil, err
	}
	if operand == nil {
		return nil, nil
	}
	if nt, ok := n.Type.(*ast.NamedType); ok {
		val, err := convertToType(operand, nt.Name)
		if err != nil {
			return nil, fmt.Errorf("cast failed: %w", err)
		}
		return val, nil
	}
	return operand, nil
}

func (e *Evaluator) evalTypeExtent(n *ast.TypeExtent) (fptypes.Value, error) { //nolint:unparam // error is part of the eval interface
	if n.Type == nil {
		return nil, nil
	}
	typeName := strings.ToLower(n.Type.Name)
	if n.Extent == "minimum" {
		switch typeName {
		case "integer":
			return fptypes.NewInteger(int64(math.MinInt32)), nil
		case "decimal":
			return fptypes.NewDecimalFromFloat(-1e28), nil
		default:
			return nil, nil
		}
	}
	switch typeName {
	case "integer":
		return fptypes.NewInteger(int64(math.MaxInt32)), nil
	case "decimal":
		return fptypes.NewDecimalFromFloat(1e28), nil
	default:
		return nil, nil
	}
}

// ---------------------------------------------------------------------------
// Function calls
// ---------------------------------------------------------------------------

func (e *Evaluator) evalFunctionCall(n *ast.FunctionCall) (fptypes.Value, error) {
	// Check if it's a library-defined function
	if fd, ok := e.funcs[n.Name]; ok {
		return e.evalUserFunction(fd, n.Operands)
	}
	// Built-in functions handled here
	return e.evalBuiltinFunction(n)
}

func (e *Evaluator) evalUserFunction(fd *ast.FunctionDef, args []ast.Expression) (fptypes.Value, error) {
	if fd.External {
		return nil, fmt.Errorf("external function '%s' not implemented", fd.Name)
	}
	// Create child scope with operand bindings
	child := e.ctx.ChildScope()
	for i, op := range fd.Operands {
		if i < len(args) {
			val, err := e.Eval(args[i])
			if err != nil {
				return nil, err
			}
			child.Aliases[op.Name] = val
		}
	}
	childEval := NewEvaluator(child)
	return childEval.Eval(fd.Body)
}

func (e *Evaluator) evalBuiltinFunction(n *ast.FunctionCall) (fptypes.Value, error) {
	name := strings.ToLower(n.Name)

	// If source is present, evaluate it first
	var source fptypes.Value
	if n.Source != nil {
		var err error
		source, err = e.Eval(n.Source)
		if err != nil {
			return nil, err
		}
	}

	switch name {
	case "count":
		c := toCollection(source)
		return fptypes.NewInteger(int64(c.Count())), nil
	case "exists":
		if len(n.Operands) > 0 {
			val, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			c := toCollection(val)
			return fptypes.NewBoolean(!c.Empty()), nil
		}
		c := toCollection(source)
		return fptypes.NewBoolean(!c.Empty()), nil
	case "first":
		c := toCollection(source)
		v, _ := c.First()
		return v, nil
	case "last":
		c := toCollection(source)
		v, _ := c.Last()
		return v, nil
	case "where":
		return e.evalWhere(source, n.Operands)
	case "select":
		return e.evalSelect(source, n.Operands)
	case "tostring":
		if source != nil {
			return fptypes.NewString(source.String()), nil
		}
		return nil, nil
	case "tointeger":
		if source != nil {
			return convertToType(source, "Integer")
		}
		return nil, nil
	case "todecimal":
		if source != nil {
			return convertToType(source, "Decimal")
		}
		return nil, nil
	case "not":
		if source == nil {
			return nil, nil
		}
		return fptypes.NewBoolean(!isTrue(source)), nil
	case "length":
		if source != nil {
			if s, ok := source.(fptypes.String); ok {
				return fptypes.NewInteger(int64(len(s.Value()))), nil
			}
		}
		return fptypes.NewInteger(0), nil
	case "coalesce":
		for _, arg := range n.Operands {
			val, err := e.Eval(arg)
			if err != nil {
				return nil, err
			}
			if val != nil {
				return val, nil
			}
		}
		return nil, nil
	case "now":
		now := time.Now().UTC()
		return fptypes.NewDateTime(now.Format("2006-01-02T15:04:05.000Z07:00"))
	case "today":
		today := time.Now().UTC()
		return fptypes.NewDate(today.Format("2006-01-02"))
	case "sum":
		return e.evalAggregateSum(source)
	case "avg":
		return e.evalAggregateAvg(source)
	case "min":
		return e.evalAggregateMinMax(source, true)
	case "max":
		return e.evalAggregateMinMax(source, false)
	case "abs":
		return e.evalAbs(source)
	case "flatten":
		return e.evalFlatten(source), nil
	case "distinct":
		c := toCollection(source)
		return cqltypes.NewList(c.Distinct()), nil

	// Clinical functions
	case "ageinyears":
		bd := e.getPatientBirthDate()
		return funcs.AgeInYears(bd)
	case "ageinmonths":
		bd := e.getPatientBirthDate()
		return funcs.AgeInMonths(bd)
	case "ageinweeks":
		bd := e.getPatientBirthDate()
		return funcs.AgeInWeeks(bd)
	case "ageindays":
		bd := e.getPatientBirthDate()
		return funcs.AgeInDays(bd)
	case "ageinyearsat":
		bd := e.getPatientBirthDate()
		if len(n.Operands) > 0 {
			asOf, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.AgeInYearsAt(bd, asOf)
		}
		return funcs.AgeInYears(bd)
	case "ageinmonthsat":
		bd := e.getPatientBirthDate()
		if len(n.Operands) > 0 {
			asOf, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.AgeInMonthsAt(bd, asOf)
		}
		return funcs.AgeInMonths(bd)
	case "calculateageinyears":
		if len(n.Operands) > 0 {
			bd, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			var asOf fptypes.Value
			if len(n.Operands) > 1 {
				asOf, err = e.Eval(n.Operands[1])
				if err != nil {
					return nil, err
				}
			}
			return funcs.CalculateAgeInYears(bd, asOf)
		}
		return nil, nil
	case "calculateageinmonths":
		if len(n.Operands) > 0 {
			bd, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			var asOf fptypes.Value
			if len(n.Operands) > 1 {
				asOf, err = e.Eval(n.Operands[1])
				if err != nil {
					return nil, err
				}
			}
			return funcs.CalculateAgeInMonths(bd, asOf)
		}
		return nil, nil
	case "calculateageinweeks":
		if len(n.Operands) > 0 {
			bd, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.CalculateAgeInWeeks(bd, nil)
		}
		return nil, nil
	case "calculateageindays":
		if len(n.Operands) > 0 {
			bd, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.CalculateAgeInDays(bd, nil)
		}
		return nil, nil

	// String functions
	case "upper":
		return funcs.Upper(source), nil
	case "lower":
		return funcs.Lower(source), nil
	case "startswith":
		if len(n.Operands) > 0 {
			arg, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.StartsWith(source, arg), nil
		}
		return fptypes.NewBoolean(false), nil
	case "endswith":
		if len(n.Operands) > 0 {
			arg, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.EndsWith(source, arg), nil
		}
		return fptypes.NewBoolean(false), nil
	case "indexof":
		if len(n.Operands) > 0 {
			arg, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.IndexOf(source, arg), nil
		}
		return fptypes.NewInteger(-1), nil
	case "matches":
		if len(n.Operands) > 0 {
			arg, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.Matches(source, arg), nil
		}
		return fptypes.NewBoolean(false), nil
	case "replacematches":
		if len(n.Operands) >= 2 {
			pat, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			rep, err := e.Eval(n.Operands[1])
			if err != nil {
				return nil, err
			}
			return funcs.ReplaceMatches(source, pat, rep), nil
		}
		return source, nil
	case "combine":
		c := toCollection(source)
		sep := ""
		if len(n.Operands) > 0 {
			s, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			if s != nil {
				sep = s.String()
			}
		}
		return funcs.Combine(c, sep), nil
	case "split":
		if len(n.Operands) > 0 {
			sep, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			if sep != nil {
				return funcs.Split(source, sep.String()), nil
			}
		}
		return source, nil

	// Statistical aggregate functions
	case "alltrue":
		c := toCollection(source)
		return funcs.AllTrue(c), nil
	case "anytrue":
		c := toCollection(source)
		return funcs.AnyTrue(c), nil
	case "populationstddev":
		c := toCollection(source)
		return funcs.PopulationStdDev(c), nil
	case "populationvariance":
		c := toCollection(source)
		return funcs.PopulationVariance(c), nil
	case "stddev":
		c := toCollection(source)
		return funcs.StdDev(c), nil
	case "variance":
		c := toCollection(source)
		return funcs.Variance(c), nil

	// Temporal functions
	case "timeofday":
		return funcs.TimeOfDay()

	// Advanced string functions
	case "positionof":
		if len(n.Operands) > 0 {
			pattern, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.PositionOf(pattern, source), nil
		}
		return fptypes.NewInteger(-1), nil
	case "lastpositionof":
		if len(n.Operands) > 0 {
			pattern, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.LastPositionOf(pattern, source), nil
		}
		return fptypes.NewInteger(-1), nil
	case "substring":
		if len(n.Operands) > 0 {
			start, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			length := -1
			if len(n.Operands) > 1 {
				l, err := e.Eval(n.Operands[1])
				if err != nil {
					return nil, err
				}
				if li, ok := l.(fptypes.Integer); ok {
					length = int(li.Value())
				}
			}
			startIdx := 0
			if si, ok := start.(fptypes.Integer); ok {
				startIdx = int(si.Value())
			}
			return funcs.Substring(source, startIdx, length), nil
		}
		return source, nil

	// Advanced list functions
	case "mode":
		c := toCollection(source)
		return funcs.Mode(c), nil
	case "median":
		c := toCollection(source)
		return funcs.Median(c), nil
	case "geometricmean":
		c := toCollection(source)
		return funcs.GeometricMean(c), nil
	case "tail":
		c := toCollection(source)
		return cqltypes.NewList(funcs.Tail(c)), nil
	case "take":
		if len(n.Operands) > 0 {
			arg, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			if ai, ok := arg.(fptypes.Integer); ok {
				c := toCollection(source)
				return cqltypes.NewList(funcs.Take(c, int(ai.Value()))), nil
			}
		}
		return source, nil
	case "skip":
		if len(n.Operands) > 0 {
			arg, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			if ai, ok := arg.(fptypes.Integer); ok {
				c := toCollection(source)
				return cqltypes.NewList(funcs.Skip(c, int(ai.Value()))), nil
			}
		}
		return source, nil

	// Null operators
	case "isnull":
		return IsNull(source), nil
	case "istrue":
		return IsTrue(source), nil
	case "isfalse":
		return IsFalse(source), nil

	// DateTime construction
	case "date":
		if len(n.Operands) >= 1 {
			year, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			var month, day fptypes.Value
			if len(n.Operands) >= 2 {
				month, err = e.Eval(n.Operands[1])
				if err != nil {
					return nil, err
				}
			}
			if len(n.Operands) >= 3 {
				day, err = e.Eval(n.Operands[2])
				if err != nil {
					return nil, err
				}
			}
			return funcs.DateConstructor(year, month, day)
		}
		return nil, nil
	case "datetime":
		if len(n.Operands) >= 1 {
			year, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			args := make([]fptypes.Value, 8)
			args[0] = year
			for i := 1; i < len(n.Operands) && i < 8; i++ {
				args[i], err = e.Eval(n.Operands[i])
				if err != nil {
					return nil, err
				}
			}
			return funcs.DateTimeConstructor(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7])
		}
		return nil, nil

	// Interval functions
	case "width":
		if iv, ok := source.(cqltypes.Interval); ok {
			return funcs.IntervalWidth(iv)
		}
		return nil, nil
	case "size":
		if iv, ok := source.(cqltypes.Interval); ok {
			return funcs.IntervalSize(iv)
		}
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown function: %s", n.Name)
	}
}

// ---------------------------------------------------------------------------
// Member access
// ---------------------------------------------------------------------------

func (e *Evaluator) evalMemberAccess(n *ast.MemberAccess) (fptypes.Value, error) {
	source, err := e.Eval(n.Source)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, nil
	}
	// Tuple member access
	if t, ok := source.(cqltypes.Tuple); ok {
		v, _ := t.Get(n.Member)
		return v, nil
	}
	// JSON object member access
	if obj, ok := source.(*fptypes.ObjectValue); ok {
		result := obj.GetCollection(n.Member)
		if result.Count() == 0 {
			return nil, nil
		}
		if result.Count() == 1 {
			return result[0], nil
		}
		return cqltypes.NewList(result), nil
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Index access
// ---------------------------------------------------------------------------

func (e *Evaluator) evalIndexAccess(n *ast.IndexAccess) (fptypes.Value, error) {
	source, err := e.Eval(n.Source)
	if err != nil {
		return nil, err
	}
	idx, err := e.Eval(n.Index)
	if err != nil {
		return nil, err
	}
	if source == nil || idx == nil {
		return nil, nil
	}
	c := toCollection(source)
	i, ok := idx.(fptypes.Integer)
	if !ok {
		return nil, fmt.Errorf("index must be integer, got %s", idx.Type())
	}
	iv := int(i.Value())
	if iv < 0 || iv >= c.Count() {
		return nil, nil
	}
	return c[iv], nil
}

// ---------------------------------------------------------------------------
// Retrieve
// ---------------------------------------------------------------------------

func (e *Evaluator) evalRetrieve(n *ast.Retrieve) (fptypes.Value, error) {
	if e.ctx.DataProvider == nil {
		return cqltypes.NewList(nil), nil
	}
	resourceType := ""
	if n.ResourceType != nil {
		resourceType = n.ResourceType.Name
	}
	// Resolve codes/valueset for filtering
	var codes interface{}
	if n.Codes != nil {
		if ref, ok := n.Codes.(*ast.IdentifierRef); ok {
			// Could be a valueset reference
			if url, ok := e.ctx.ResolveValueSetURL(ref.Name); ok {
				codes = url
			} else {
				val, err := e.Eval(n.Codes)
				if err != nil {
					return nil, err
				}
				codes = val
			}
		} else {
			val, err := e.Eval(n.Codes)
			if err != nil {
				return nil, err
			}
			codes = val
		}
	}
	// Evaluate date range if present
	var dateRange interface{}
	if n.DateRange != nil {
		val, err := e.Eval(n.DateRange)
		if err != nil {
			return nil, fmt.Errorf("retrieve [%s] date range eval: %w", resourceType, err)
		}
		dateRange = val
	}
	results, err := e.ctx.DataProvider.Retrieve(e.ctx.GoCtx, resourceType, n.CodePath, n.CodeComparator, codes, dateRange)
	if err != nil {
		return nil, fmt.Errorf("retrieve [%s] failed: %w", resourceType, err)
	}
	// Convert JSON results to fhirpath Objects
	values := make(fptypes.Collection, 0, len(results))
	for _, raw := range results {
		obj := fptypes.NewObjectValue([]byte(raw))
		values = append(values, obj)
	}
	return cqltypes.NewList(values), nil
}

// ---------------------------------------------------------------------------
// Query
// ---------------------------------------------------------------------------

func (e *Evaluator) evalQuery(n *ast.Query) (fptypes.Value, error) {
	if len(n.Sources) == 0 {
		return cqltypes.NewList(nil), nil
	}
	// Evaluate the first source
	source, err := e.Eval(n.Sources[0].Source)
	if err != nil {
		return nil, err
	}
	items := toCollection(source)

	// Process each item
	var results fptypes.Collection
	for i, item := range items {
		child := e.ctx.ChildScope()
		child.Aliases[n.Sources[0].Alias] = item
		child.This = item
		child.Index = i

		// Process let bindings (reuse funcs map from parent evaluator)
		childEval := e.withContext(child)
		for _, let := range n.Let {
			val, err := childEval.Eval(let.Expression)
			if err != nil {
				return nil, err
			}
			child.LetBindings[let.Identifier] = val
		}

		// Check with clauses
		withOk := true
		for _, w := range n.With {
			ok, err := childEval.evalWithClause(w)
			if err != nil {
				return nil, err
			}
			if !ok {
				withOk = false
				break
			}
		}
		if !withOk {
			continue
		}

		// Check without clauses
		withoutOk := true
		for _, w := range n.Without {
			ok, err := childEval.evalWithoutClause(w)
			if err != nil {
				return nil, err
			}
			if !ok {
				withoutOk = false
				break
			}
		}
		if !withoutOk {
			continue
		}

		// Apply where filter
		if n.Where != nil {
			cond, err := childEval.Eval(n.Where)
			if err != nil {
				return nil, err
			}
			if !isTrue(cond) {
				continue
			}
		}

		// Apply return clause or use the item directly
		if n.Return != nil {
			val, err := childEval.Eval(n.Return.Expression)
			if err != nil {
				return nil, err
			}
			if val != nil {
				results = append(results, val)
			}
		} else {
			results = append(results, item)
		}
	}

	// Apply distinct if specified
	if n.Return != nil && n.Return.Distinct {
		results = results.Distinct()
	}

	// TODO: Apply sort clause

	return cqltypes.NewList(results), nil
}

func (e *Evaluator) evalWithClause(w *ast.WithClause) (bool, error) {
	source, err := e.Eval(w.Source.Source)
	if err != nil {
		return false, err
	}
	items := toCollection(source)
	for _, item := range items {
		e.ctx.Aliases[w.Source.Alias] = item
		cond, err := e.Eval(w.Condition)
		if err != nil {
			return false, err
		}
		if isTrue(cond) {
			return true, nil
		}
	}
	return false, nil
}

func (e *Evaluator) evalWithoutClause(w *ast.WithoutClause) (bool, error) {
	source, err := e.Eval(w.Source.Source)
	if err != nil {
		return false, err
	}
	items := toCollection(source)
	for _, item := range items {
		e.ctx.Aliases[w.Source.Alias] = item
		cond, err := e.Eval(w.Condition)
		if err != nil {
			return false, err
		}
		if isTrue(cond) {
			return false, nil // without: exclude if any match
		}
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

func (e *Evaluator) evalIntervalExpr(n *ast.IntervalExpression) (fptypes.Value, error) {
	low, err := e.Eval(n.Low)
	if err != nil {
		return nil, err
	}
	high, err := e.Eval(n.High)
	if err != nil {
		return nil, err
	}
	return cqltypes.NewInterval(low, high, n.LowClosed, n.HighClosed), nil
}

func (e *Evaluator) evalTupleExpr(n *ast.TupleExpression) (fptypes.Value, error) {
	elements := make(map[string]fptypes.Value)
	for _, elem := range n.Elements {
		val, err := e.Eval(elem.Expression)
		if err != nil {
			return nil, err
		}
		elements[elem.Name] = val
	}
	return cqltypes.NewTuple(elements), nil
}

func (e *Evaluator) evalInstanceExpr(n *ast.InstanceExpression) (fptypes.Value, error) {
	elements := make(map[string]fptypes.Value)
	for _, elem := range n.Elements {
		val, err := e.Eval(elem.Expression)
		if err != nil {
			return nil, err
		}
		elements[elem.Name] = val
	}
	return cqltypes.NewTuple(elements), nil
}

func (e *Evaluator) evalListExpr(n *ast.ListExpression) (fptypes.Value, error) {
	values := make(fptypes.Collection, 0, len(n.Elements))
	for _, elem := range n.Elements {
		val, err := e.Eval(elem)
		if err != nil {
			return nil, err
		}
		if val != nil {
			values = append(values, val)
		}
	}
	return cqltypes.NewList(values), nil
}

func (e *Evaluator) evalCodeExpr(n *ast.CodeExpression) (fptypes.Value, error) { //nolint:unparam // error is part of the eval interface
	system := n.System
	// Resolve system name to URL if it's a codesystem reference
	if cs, ok := e.ctx.CodeSystems[system]; ok {
		system = cs.System
	}
	return cqltypes.NewCode(system, n.Code, n.Display), nil
}

func (e *Evaluator) evalConceptExpr(n *ast.ConceptExpression) (fptypes.Value, error) {
	codes := make([]cqltypes.Code, 0, len(n.Codes))
	for _, c := range n.Codes {
		val, err := e.evalCodeExpr(c)
		if err != nil {
			return nil, err
		}
		if code, ok := val.(cqltypes.Code); ok {
			codes = append(codes, code)
		}
	}
	return cqltypes.NewConcept(codes, n.Display), nil
}

func (e *Evaluator) evalExternalConstant(n *ast.ExternalConstant) (fptypes.Value, error) { //nolint:unparam // error is part of the eval interface
	if val, ok := e.ctx.Parameters[n.Name]; ok {
		return val, nil
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Membership / Between
// ---------------------------------------------------------------------------

func (e *Evaluator) evalMembership(n *ast.MembershipExpression) (fptypes.Value, error) {
	left, err := e.Eval(n.Left)
	if err != nil {
		return nil, err
	}
	right, err := e.Eval(n.Right)
	if err != nil {
		return nil, err
	}
	if n.Operator == "in" {
		return e.evalInContains(ast.OpIn, left, right)
	}
	return e.evalInContains(ast.OpContains, left, right)
}

func (e *Evaluator) evalBetween(n *ast.BetweenExpression) (fptypes.Value, error) {
	operand, err := e.Eval(n.Operand)
	if err != nil {
		return nil, err
	}
	low, err := e.Eval(n.Low)
	if err != nil {
		return nil, err
	}
	high, err := e.Eval(n.High)
	if err != nil {
		return nil, err
	}
	if operand == nil || low == nil || high == nil {
		return nil, nil
	}
	interval := cqltypes.NewInterval(low, high, !n.Properly, !n.Properly)
	result, err := interval.Contains(operand)
	if err != nil {
		return nil, err
	}
	return fptypes.NewBoolean(result), nil
}

func (e *Evaluator) evalDurationBetween(n *ast.DurationBetween) (fptypes.Value, error) {
	low, err := e.Eval(n.Low)
	if err != nil {
		return nil, err
	}
	high, err := e.Eval(n.High)
	if err != nil {
		return nil, err
	}
	return funcs.DurationBetween(low, high, n.Precision)
}

func (e *Evaluator) evalDifferenceBetween(n *ast.DifferenceBetween) (fptypes.Value, error) {
	low, err := e.Eval(n.Low)
	if err != nil {
		return nil, err
	}
	high, err := e.Eval(n.High)
	if err != nil {
		return nil, err
	}
	return funcs.DifferenceBetween(low, high, n.Precision)
}

func (e *Evaluator) evalDateTimeComponentFrom(n *ast.DateTimeComponentFrom) (fptypes.Value, error) {
	operand, err := e.Eval(n.Operand)
	if err != nil {
		return nil, err
	}
	return funcs.DateTimeComponentFrom(operand, n.Component)
}

func (e *Evaluator) evalTimingExpr(n *ast.TimingExpression) (fptypes.Value, error) {
	left, err := e.Eval(n.Left)
	if err != nil {
		return nil, err
	}
	right, err := e.Eval(n.Right)
	if err != nil {
		return nil, err
	}
	leftIv, leftOk := left.(cqltypes.Interval)
	rightIv, rightOk := right.(cqltypes.Interval)
	if !leftOk || !rightOk {
		return nil, nil
	}
	switch n.Operator.Kind {
	case ast.TimingSameAs:
		return fptypes.NewBoolean(leftIv.Equal(rightIv)), nil
	case ast.TimingIncludes:
		result, err := leftIv.Includes(rightIv)
		if err != nil {
			return nil, err
		}
		return fptypes.NewBoolean(result), nil
	case ast.TimingIncludedIn, ast.TimingDuring:
		result, err := rightIv.Includes(leftIv)
		if err != nil {
			return nil, err
		}
		return fptypes.NewBoolean(result), nil
	case ast.TimingBeforeOrAfter:
		if n.Operator.Before {
			return funcs.IntervalBefore(leftIv, rightIv)
		}
		return funcs.IntervalAfter(leftIv, rightIv)
	case ast.TimingMeets:
		return funcs.IntervalMeets(leftIv, rightIv)
	case ast.TimingOverlaps:
		return funcs.IntervalOverlaps(leftIv, rightIv)
	case ast.TimingStarts:
		return funcs.Starts(leftIv, rightIv)
	case ast.TimingEnds:
		return funcs.Ends(leftIv, rightIv)
	case ast.TimingWithin:
		return funcs.During(leftIv, rightIv)
	default:
		return nil, nil
	}
}

func (e *Evaluator) evalSetAggregate(n *ast.SetAggregateExpression) (fptypes.Value, error) {
	operand, err := e.Eval(n.Operand)
	if err != nil {
		return nil, err
	}
	if operand == nil {
		return nil, nil
	}
	c := toCollection(operand)
	switch n.Kind {
	case "expand":
		// Expand intervals into point lists
		var result fptypes.Collection
		for _, item := range c {
			if iv, ok := item.(cqltypes.Interval); ok {
				points, err := funcs.IntervalExpand(iv, decimal.Zero)
				if err != nil {
					return nil, err
				}
				result = append(result, points...)
			}
		}
		return cqltypes.NewList(result), nil
	case "collapse":
		// Collapse overlapping intervals
		var intervals []cqltypes.Interval
		for _, item := range c {
			if iv, ok := item.(cqltypes.Interval); ok {
				intervals = append(intervals, iv)
			}
		}
		collapsed, err := funcs.IntervalCollapse(intervals)
		if err != nil {
			return nil, err
		}
		var result fptypes.Collection
		for _, iv := range collapsed {
			result = append(result, iv)
		}
		return cqltypes.NewList(result), nil
	default:
		return nil, nil
	}
}

// ---------------------------------------------------------------------------
// Collection-level functions
// ---------------------------------------------------------------------------

func (e *Evaluator) evalWhere(source fptypes.Value, operands []ast.Expression) (fptypes.Value, error) {
	if len(operands) == 0 {
		return source, nil
	}
	c := toCollection(source)
	var results fptypes.Collection
	for i, item := range c {
		child := e.ctx.ChildScope()
		child.This = item
		child.Index = i
		childEval := NewEvaluator(child)
		cond, err := childEval.Eval(operands[0])
		if err != nil {
			return nil, err
		}
		if isTrue(cond) {
			results = append(results, item)
		}
	}
	return cqltypes.NewList(results), nil
}

func (e *Evaluator) evalSelect(source fptypes.Value, operands []ast.Expression) (fptypes.Value, error) {
	if len(operands) == 0 {
		return source, nil
	}
	c := toCollection(source)
	var results fptypes.Collection
	for i, item := range c {
		child := e.ctx.ChildScope()
		child.This = item
		child.Index = i
		childEval := NewEvaluator(child)
		val, err := childEval.Eval(operands[0])
		if err != nil {
			return nil, err
		}
		if val != nil {
			results = append(results, val)
		}
	}
	return cqltypes.NewList(results), nil
}

// ---------------------------------------------------------------------------
// Aggregate functions
// ---------------------------------------------------------------------------

func (e *Evaluator) evalAggregateSum(source fptypes.Value) (fptypes.Value, error) {
	c := toCollection(source)
	if c.Empty() {
		return nil, nil
	}
	sum := decimal.Zero
	for _, item := range c {
		if i, ok := item.(fptypes.Integer); ok {
			sum = sum.Add(decimal.NewFromInt(i.Value()))
		} else {
			d := toDecimal(item)
			sum = sum.Add(d)
		}
	}
	return newDecimalFromD(sum), nil
}

func (e *Evaluator) evalAggregateAvg(source fptypes.Value) (fptypes.Value, error) {
	c := toCollection(source)
	if c.Empty() {
		return nil, nil
	}
	sum := decimal.Zero
	for _, item := range c {
		d := toDecimal(item)
		sum = sum.Add(d)
	}
	return newDecimalFromD(sum.Div(decimal.NewFromInt(int64(c.Count())))), nil
}

func (e *Evaluator) evalAggregateMinMax(source fptypes.Value, isMin bool) (fptypes.Value, error) {
	c := toCollection(source)
	if c.Empty() {
		return nil, nil
	}
	result := c[0]
	for _, item := range c[1:] {
		comp, ok := result.(fptypes.Comparable)
		if !ok {
			continue
		}
		cmp, err := comp.Compare(item)
		if err != nil {
			continue
		}
		if (isMin && cmp > 0) || (!isMin && cmp < 0) {
			result = item
		}
	}
	return result, nil
}

func (e *Evaluator) evalAbs(source fptypes.Value) (fptypes.Value, error) {
	if source == nil {
		return nil, nil
	}
	if i, ok := source.(fptypes.Integer); ok {
		v := i.Value()
		if v < 0 {
			v = -v
		}
		return fptypes.NewInteger(v), nil
	}
	d := toDecimal(source)
	return newDecimalFromD(d.Abs()), nil
}

// getPatientBirthDate extracts the birthDate from the context Patient resource.
func (e *Evaluator) getPatientBirthDate() fptypes.Value {
	if len(e.ctx.ContextValue) == 0 {
		return nil
	}
	// Use cached ObjectValue to avoid repeated JSON parsing
	obj := e.ctx.GetContextObject()
	c := obj.GetCollection("birthDate")
	if c.Count() > 0 {
		return c[0]
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func isTrue(v fptypes.Value) bool {
	if v == nil {
		return false
	}
	b, ok := v.(fptypes.Boolean)
	return ok && b.Bool()
}

func isFalse(v fptypes.Value) bool {
	if v == nil {
		return false
	}
	b, ok := v.(fptypes.Boolean)
	return ok && !b.Bool()
}

func toCollection(v fptypes.Value) fptypes.Collection {
	if v == nil {
		return fptypes.Collection{}
	}
	if list, ok := v.(cqltypes.List); ok {
		return list.Values
	}
	return fptypes.Collection{v}
}

func toDecimal(v fptypes.Value) decimal.Decimal {
	if v == nil {
		return decimal.Zero
	}
	if i, ok := v.(fptypes.Integer); ok {
		return decimal.NewFromInt(i.Value())
	}
	if d, ok := v.(fptypes.Decimal); ok {
		return d.Value()
	}
	if n, ok := v.(fptypes.Numeric); ok {
		d := n.ToDecimal()
		return d.Value()
	}
	return decimal.Zero
}

func isDecimal(v fptypes.Value) bool {
	_, ok := v.(fptypes.Decimal)
	return ok
}

// newDecimalFromD creates a fptypes.Value from a decimal.Decimal.
func newDecimalFromD(d decimal.Decimal) fptypes.Value {
	v, err := fptypes.NewDecimal(d.String())
	if err != nil {
		return nil
	}
	return v
}

func convertToType(v fptypes.Value, typeName string) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	switch strings.ToLower(typeName) {
	case "string":
		return fptypes.NewString(v.String()), nil
	case "integer":
		switch val := v.(type) {
		case fptypes.Integer:
			return val, nil
		case fptypes.String:
			i, err := strconv.ParseInt(val.Value(), 10, 64)
			if err != nil {
				return nil, err
			}
			return fptypes.NewInteger(i), nil
		case fptypes.Boolean:
			if val.Bool() {
				return fptypes.NewInteger(1), nil
			}
			return fptypes.NewInteger(0), nil
		}
	case "decimal":
		switch val := v.(type) {
		case fptypes.Decimal:
			return val, nil
		case fptypes.Integer:
			return fptypes.NewDecimalFromInt(val.Value()), nil
		case fptypes.String:
			return fptypes.NewDecimal(val.Value())
		}
	case "boolean":
		switch val := v.(type) {
		case fptypes.Boolean:
			return val, nil
		case fptypes.String:
			s := strings.ToLower(val.Value())
			if s == "true" || s == "1" {
				return fptypes.NewBoolean(true), nil
			}
			if s == "false" || s == "0" {
				return fptypes.NewBoolean(false), nil
			}
		case fptypes.Integer:
			return fptypes.NewBoolean(val.Value() != 0), nil
		}
	}
	return nil, fmt.Errorf("cannot convert %s to %s", v.Type(), typeName)
}
