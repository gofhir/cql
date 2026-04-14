package eval

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/funcs"
	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
)

// isAmbiguousComparisonErr returns true if the error is an ambiguous temporal comparison.
// In CQL, ambiguous comparisons should return null, not error.
func isAmbiguousComparisonErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "ambiguous comparison")
}

// queryCombo holds one combination of alias bindings from a multi-source query.
type queryCombo struct {
	aliases map[string]fptypes.Value
}

// Evaluator interprets CQL AST nodes.
type Evaluator struct {
	ctx           *Context
	funcs         map[string][]*ast.FunctionDef            // local overloads
	includedFuncs map[string]map[string][]*ast.FunctionDef // alias → name → overloads
}

// NewEvaluator creates a new evaluator for the given context.
func NewEvaluator(ctx *Context) *Evaluator {
	e := &Evaluator{
		ctx:           ctx,
		funcs:         make(map[string][]*ast.FunctionDef),
		includedFuncs: make(map[string]map[string][]*ast.FunctionDef),
	}
	// Register library functions
	if ctx.Library != nil {
		for _, f := range ctx.Library.Functions {
			e.funcs[f.Name] = append(e.funcs[f.Name], f)
		}
	}
	// Register included library functions
	for alias, lib := range ctx.IncludedLibraries {
		libFuncs := make(map[string][]*ast.FunctionDef)
		for _, f := range lib.Functions {
			libFuncs[f.Name] = append(libFuncs[f.Name], f)
		}
		e.includedFuncs[alias] = libFuncs
	}
	return e
}

// withContext returns a lightweight evaluator sharing the same function registry
// but using a different context. Avoids re-building the funcs map on each iteration.
func (e *Evaluator) withContext(ctx *Context) *Evaluator {
	return &Evaluator{ctx: ctx, funcs: e.funcs, includedFuncs: e.includedFuncs}
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
		// CQL integers are 32-bit: valid range is -2^31 to 2^31-1
		if v < math.MinInt32 || v > math.MaxInt32 {
			return nil, fmt.Errorf("integer overflow: %s", n.Value)
		}
		return fptypes.NewInteger(v), nil
	case ast.LiteralLong:
		v, err := strconv.ParseInt(n.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid long: %s", n.Value)
		}
		return fptypes.NewInteger(v), nil
	case ast.LiteralDecimal:
		// CQL decimal validation: max 28 digits before decimal, max 8 digits after
		if err := validateCQLDecimal(n.Value); err != nil {
			return nil, err
		}
		return fptypes.NewDecimal(n.Value)
	case ast.LiteralDate:
		return fptypes.NewDate(n.Value)
	case ast.LiteralDateTime:
		// Strip trailing 'T' when no time component follows (e.g., "2015-02-10T" -> "2015-02-10")
		dtVal := strings.TrimSuffix(n.Value, "T")
		return fptypes.NewDateTime(dtVal)
	case ast.LiteralTime:
		// Validate millisecond digits before parsing: CQL allows at most 3 fractional digits.
		if dotIdx := strings.LastIndex(n.Value, "."); dotIdx >= 0 {
			frac := n.Value[dotIdx+1:]
			if len(frac) > 3 {
				return nil, fmt.Errorf("invalid time literal (milliseconds exceed 3 digits): %s", n.Value)
			}
		}
		t, err := fptypes.NewTime(n.Value)
		if err != nil {
			return nil, err
		}
		// Validate time component ranges
		if t.Hour() > 23 || t.Minute() > 59 || t.Second() > 59 || t.Millisecond() > 999 {
			return nil, fmt.Errorf("invalid time: %s", n.Value)
		}
		return t, nil
	case ast.LiteralQuantity:
		return fptypes.NewQuantity(n.Value)
	default:
		return fptypes.NewString(n.Value), nil
	}
}

// ---------------------------------------------------------------------------
// Identifier resolution
// ---------------------------------------------------------------------------

func (e *Evaluator) evalIdentifierRef(n *ast.IdentifierRef) (fptypes.Value, error) {
	val, ok := e.ctx.ResolveIdentifier(n.Name)
	if ok {
		return val, nil
	}
	// Check if the identifier refers to the context resource type (e.g. "Patient").
	// In CQL, `context Patient` makes `Patient` resolve to the current context resource.
	if e.ctx.Library != nil && len(e.ctx.ContextValue) > 0 {
		for _, ctxDef := range e.ctx.Library.Contexts {
			if ctxDef.Name == n.Name {
				obj := e.ctx.GetContextObject()
				if obj != nil {
					return obj, nil
				}
			}
		}
	}

	// Lazily evaluate library expression definitions referenced by name.
	// This handles CQL like: define "A": true  define "B": "A" and false
	// where "B" references "A" via IdentifierRef.
	if e.ctx.Library != nil {
		for _, stmt := range e.ctx.Library.Statements {
			if stmt.Name == n.Name {
				result, err := e.Eval(stmt.Expression)
				if err != nil {
					return nil, fmt.Errorf("evaluating referenced expression %q: %w", n.Name, err)
				}
				e.ctx.Definitions[n.Name] = result
				return result, nil
			}
		}
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
			// For list union: null union list = list, list union null = list
			// For interval union: null union interval = null
			if left == nil {
				if _, rok := right.(cqltypes.Interval); rok {
					return nil, nil
				}
				return right, nil
			}
			if right == nil {
				if _, lok := left.(cqltypes.Interval); lok {
					return nil, nil
				}
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
		case ast.OpExcept:
			// list except null = list
			if left != nil && right == nil {
				return left, nil
			}
			return nil, nil
		case ast.OpIntersect:
			// list intersect null = null
			return nil, nil
		case ast.OpIn, ast.OpContains:
			// CQL: null in/contains needs special handling — pass through to evalInContains
			return e.evalInContains(n.Operator, left, right)
		case ast.OpAdd:
			// CQL: string + null = null (null propagation for string concat too)
			return nil, nil
		default:
			return nil, nil
		}
	}

	switch n.Operator {
	case ast.OpEqual:
		// CQL: Tuple equality returns null if any element comparison involves null
		if lt, lok := left.(cqltypes.Tuple); lok {
			if rt, rok := right.(cqltypes.Tuple); rok {
				return tupleEqual(lt, rt)
			}
		}
		return fptypes.NewBoolean(left.Equal(right)), nil
	case ast.OpNotEqual:
		if lt, lok := left.(cqltypes.Tuple); lok {
			if rt, rok := right.(cqltypes.Tuple); rok {
				eq, err := tupleEqual(lt, rt)
				if err != nil {
					return nil, err
				}
				if eq == nil {
					return nil, nil
				}
				return fptypes.NewBoolean(!isTrue(eq)), nil
			}
		}
		return fptypes.NewBoolean(!left.Equal(right)), nil
	case ast.OpEquivalent:
		// CQL: Tuple equivalence with different shapes is an error
		if lt, lok := left.(cqltypes.Tuple); lok {
			if rt, rok := right.(cqltypes.Tuple); rok {
				if len(lt.Elements) != len(rt.Elements) {
					return nil, fmt.Errorf("tuple equivalence requires tuples with the same elements")
				}
			}
		}
		return fptypes.NewBoolean(cqlEquivalent(left, right)), nil
	case ast.OpNotEquivalent:
		// CQL: Tuple equivalence with different shapes is an error
		if lt, lok := left.(cqltypes.Tuple); lok {
			if rt, rok := right.(cqltypes.Tuple); rok {
				if len(lt.Elements) != len(rt.Elements) {
					return nil, fmt.Errorf("tuple equivalence requires tuples with the same elements")
				}
			}
		}
		return fptypes.NewBoolean(!cqlEquivalent(left, right)), nil

	case ast.OpLess, ast.OpLessOrEqual, ast.OpGreater, ast.OpGreaterOrEqual:
		// Handle uncertainty intervals: Interval compared to scalar
		if iv, ok := left.(cqltypes.Interval); ok {
			return compareIntervalWithScalar(iv, right, n.Operator)
		}
		if iv, ok := right.(cqltypes.Interval); ok {
			// Flip the comparison direction
			flipped := n.Operator
			switch flipped {
			case ast.OpLess:
				flipped = ast.OpGreater
			case ast.OpLessOrEqual:
				flipped = ast.OpGreaterOrEqual
			case ast.OpGreater:
				flipped = ast.OpLess
			case ast.OpGreaterOrEqual:
				flipped = ast.OpLessOrEqual
			}
			return compareIntervalWithScalar(iv, left, flipped)
		}

		// Promote Decimal to Quantity (unit "1") when comparing with Quantity
		if _, lIsQ := left.(fptypes.Quantity); lIsQ {
			if rd, rIsD := right.(fptypes.Decimal); rIsD {
				right = fptypes.NewQuantityFromDecimal(rd.Value(), "1")
			}
		}
		if _, rIsQ := right.(fptypes.Quantity); rIsQ {
			if ld, lIsD := left.(fptypes.Decimal); lIsD {
				left = fptypes.NewQuantityFromDecimal(ld.Value(), "1")
			}
		}
		lc, ok := left.(fptypes.Comparable)
		if !ok {
			return nil, fmt.Errorf("cannot compare %s", left.Type())
		}
		cmp, err := lc.Compare(right)
		if err != nil {
			if isAmbiguousComparisonErr(err) {
				return nil, nil // CQL: ambiguous temporal comparison → null
			}
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
		// Interval union: if both are intervals, compute interval union
		if lIv, lok := left.(cqltypes.Interval); lok {
			if rIv, rok := right.(cqltypes.Interval); rok {
				return funcs.IntervalUnion(lIv, rIv)
			}
		}
		return e.evalSetOp(n.Operator, left, right)
	case ast.OpIntersect:
		// Interval intersect
		if lIv, lok := left.(cqltypes.Interval); lok {
			if rIv, rok := right.(cqltypes.Interval); rok {
				return funcs.IntervalIntersect(lIv, rIv)
			}
		}
		return e.evalSetOp(n.Operator, left, right)
	case ast.OpExcept:
		// Interval except
		if lIv, lok := left.(cqltypes.Interval); lok {
			if rIv, rok := right.(cqltypes.Interval); rok {
				return funcs.IntervalExcept(lIv, rIv)
			}
		}
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
	// Handle uncertainty intervals: Interval op scalar → apply op to both bounds
	leftIsIv, _ := left.(cqltypes.Interval)
	rightIsIv, _ := right.(cqltypes.Interval)
	_, leftIsInterval := left.(cqltypes.Interval)
	_, rightIsInterval := right.(cqltypes.Interval)
	if leftIsInterval || rightIsInterval {
		// div and mod are not supported on uncertainty intervals
		if op == ast.OpDiv || op == ast.OpMod {
			return nil, fmt.Errorf("integer division (div/mod) is not supported on uncertainty intervals")
		}
		if leftIsInterval {
			return intervalArithmetic(e, leftIsIv, right, op, false)
		}
		return intervalArithmetic(e, rightIsIv, left, op, true)
	}

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

	// String + String → concatenation
	if op == ast.OpAdd {
		if ls, lok := left.(fptypes.String); lok {
			if rs, rok := right.(fptypes.String); rok {
				return fptypes.NewString(ls.Value() + rs.Value()), nil
			}
		}
	}

	// DateTime/Date/Time ± Quantity (temporal arithmetic)
	if isTemporalType(left) {
		if rq, ok := right.(fptypes.Quantity); ok {
			amount := int(rq.Value().IntPart())
			unit := rq.Unit()
			switch op {
			case ast.OpAdd:
				return funcs.DateAdd(left, amount, unit)
			case ast.OpSubtract:
				return funcs.DateAdd(left, -amount, unit)
			default:
				return nil, fmt.Errorf("unsupported operator for temporal arithmetic")
			}
		}
	}

	// Quantity ± Quantity
	lq, lqOk := left.(fptypes.Quantity)
	rq, rqOk := right.(fptypes.Quantity)
	if lqOk && rqOk {
		switch op {
		case ast.OpAdd:
			return lq.Add(rq)
		case ast.OpSubtract:
			return lq.Subtract(rq)
		case ast.OpMultiply:
			resultVal := lq.Value().Mul(rq.Value())
			resultUnit := multiplyUnits(lq.Unit(), rq.Unit())
			return fptypes.NewQuantityFromDecimal(resultVal, resultUnit), nil
		case ast.OpDivide:
			if rq.Value().IsZero() {
				return nil, nil
			}
			resultVal := lq.Value().Div(rq.Value())
			resultUnit := divideUnits(lq.Unit(), rq.Unit())
			return fptypes.NewQuantityFromDecimal(resultVal, resultUnit), nil
		case ast.OpDiv:
			if rq.Value().IsZero() {
				return nil, nil
			}
			result := lq.Value().Div(rq.Value()).IntPart()
			return fptypes.NewQuantityFromDecimal(decimal.NewFromInt(result), lq.Unit()), nil
		case ast.OpMod:
			if rq.Value().IsZero() {
				return nil, nil
			}
			return fptypes.NewQuantityFromDecimal(lq.Value().Mod(rq.Value()), lq.Unit()), nil
		default:
			return nil, fmt.Errorf("unsupported operator for quantity arithmetic")
		}
	}
	// Quantity * or / numeric
	if lqOk {
		rd := toDecimal(right)
		switch op {
		case ast.OpMultiply:
			return lq.Multiply(rd), nil
		case ast.OpDivide:
			return lq.Divide(rd)
		}
	}
	// numeric * Quantity
	if rqOk {
		ld := toDecimal(left)
		if op == ast.OpMultiply {
			return rq.Multiply(ld), nil
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
		return cqltypes.NewList(nullSafeUnion(lc, rc)), nil
	case ast.OpIntersect:
		return cqltypes.NewList(nullSafeIntersect(lc, rc)), nil
	case ast.OpExcept:
		return cqltypes.NewList(nullSafeExclude(lc, rc)), nil
	}
	return nil, nil
}

// nullSafeUnion performs union that handles nil elements properly.
func nullSafeUnion(lc, rc fptypes.Collection) fptypes.Collection {
	result := make(fptypes.Collection, 0, len(lc)+len(rc))
	result = append(result, lc...)
	for _, item := range rc {
		if !nullSafeContains(result, item) {
			result = append(result, item)
		}
	}
	return result
}

// nullSafeIntersect performs intersect that handles nil elements properly.
func nullSafeIntersect(lc, rc fptypes.Collection) fptypes.Collection {
	var result fptypes.Collection
	for _, item := range lc {
		if nullSafeContains(rc, item) && !nullSafeContains(result, item) {
			result = append(result, item)
		}
	}
	return result
}

// nullSafeExclude performs except that handles nil elements properly.
func nullSafeExclude(lc, rc fptypes.Collection) fptypes.Collection {
	var result fptypes.Collection
	for _, item := range lc {
		if !nullSafeContains(rc, item) {
			result = append(result, item)
		}
	}
	return result
}

// nullSafeContains checks if collection contains value, handling nil properly.
func nullSafeContains(c fptypes.Collection, v fptypes.Value) bool {
	if v == nil {
		for _, item := range c {
			if item == nil {
				return true
			}
		}
		return false
	}
	for _, item := range c {
		if item != nil && item.Equal(v) {
			return true
		}
	}
	return false
}

// cqlEquivalent implements CQL equivalence with precision-aware decimal comparison.
func cqlEquivalent(left, right fptypes.Value) bool {
	// Decimal equivalence: compare at the least precision (after stripping trailing zeros)
	if ld, ok := left.(fptypes.Decimal); ok {
		if rd, ok := right.(fptypes.Decimal); ok {
			return decimalEquivalent(ld.Value(), rd.Value())
		}
		if ri, ok := right.(fptypes.Integer); ok {
			return decimalEquivalent(ld.Value(), decimal.NewFromInt(ri.Value()))
		}
	}
	if li, ok := left.(fptypes.Integer); ok {
		if rd, ok := right.(fptypes.Decimal); ok {
			return decimalEquivalent(decimal.NewFromInt(li.Value()), rd.Value())
		}
	}
	return left.Equivalent(right)
}

// decimalEquivalent compares two decimals at the least precision after stripping trailing zeros.
func decimalEquivalent(a, b decimal.Decimal) bool {
	// Strip trailing zeros by converting to string and back
	as := a.String()
	bs := b.String()
	// Count significant decimal places (after stripping trailing zeros)
	aDec := decimalPlaces(as)
	bDec := decimalPlaces(bs)
	// Use the lesser precision
	minDec := aDec
	if bDec < minDec {
		minDec = bDec
	}
	// Truncate both to the minimum precision and compare.
	// minDec is bounded by string decimal place count so it fits in int32.
	aRound := a.Truncate(int32(minDec)) //nolint:gosec // minDec is derived from decimal place count, always small
	bRound := b.Truncate(int32(minDec)) //nolint:gosec // minDec is derived from decimal place count, always small
	return aRound.Equal(bRound)
}

// decimalPlaces returns the number of significant decimal places (after removing trailing zeros).
func decimalPlaces(s string) int {
	dotIdx := strings.IndexByte(s, '.')
	if dotIdx < 0 {
		return 0
	}
	frac := s[dotIdx+1:]
	// Strip trailing zeros
	frac = strings.TrimRight(frac, "0")
	return len(frac)
}

// tupleEqual compares two tuples with CQL null semantics:
// If a field is null on both sides, it's treated as matching.
// If a field is null on one side but not the other, the whole comparison returns null.
// If any non-null fields differ, returns false.
func tupleEqual(a, b cqltypes.Tuple) (fptypes.Value, error) {
	if len(a.Elements) != len(b.Elements) {
		return fptypes.NewBoolean(false), nil
	}
	hasAsymmetricNull := false
	for k, av := range a.Elements {
		bv, exists := b.Elements[k]
		if !exists {
			return fptypes.NewBoolean(false), nil
		}
		if av == nil && bv == nil {
			continue // both null → matching
		}
		if av == nil || bv == nil {
			hasAsymmetricNull = true // one null, one not → indeterminate
			continue
		}
		if !av.Equal(bv) {
			return fptypes.NewBoolean(false), nil
		}
	}
	if hasAsymmetricNull {
		return nil, nil
	}
	return fptypes.NewBoolean(true), nil
}

// nullSafeDistinct removes duplicates from a collection, handling nil properly.
func nullSafeDistinct(c fptypes.Collection) fptypes.Collection {
	if len(c) <= 1 {
		return c
	}
	result := make(fptypes.Collection, 0, len(c))
	for _, item := range c {
		if !nullSafeContains(result, item) {
			result = append(result, item)
		}
	}
	return result
}

func (e *Evaluator) evalInContains(op ast.BinaryOp, left, right fptypes.Value) (fptypes.Value, error) {
	if op == ast.OpIn {
		// left in right: check if left is in the right collection/interval
		if interval, ok := right.(cqltypes.Interval); ok {
			if left == nil {
				return nil, nil // CQL: null in Interval → null
			}
			result, err := interval.Contains(left)
			if err != nil {
				if isAmbiguousComparisonErr(err) {
					return nil, nil
				}
				return nil, err
			}
			return fptypes.NewBoolean(result), nil
		}
		rc := toCollection(right)
		// CQL: null in {1, null} = true; null in {} = false; null in {1,2} = null
		if left == nil {
			hasNull := false
			for _, item := range rc {
				if item == nil {
					hasNull = true
					break
				}
			}
			if len(rc) == 0 {
				return fptypes.NewBoolean(false), nil
			}
			if hasNull {
				return fptypes.NewBoolean(true), nil
			}
			return nil, nil
		}
		return fptypes.NewBoolean(listContainsValue(rc, left)), nil
	}
	// contains: right in left
	if interval, ok := left.(cqltypes.Interval); ok {
		if right == nil {
			return nil, nil // CQL: Interval contains null → null
		}
		result, err := interval.Contains(right)
		if err != nil {
			if isAmbiguousComparisonErr(err) {
				return nil, nil
			}
			return nil, err
		}
		return fptypes.NewBoolean(result), nil
	}
	lc := toCollection(left)
	// CQL: {1, null} contains null = true; {} contains null = null
	if right == nil {
		hasNull := false
		for _, item := range lc {
			if item == nil {
				hasNull = true
				break
			}
		}
		if hasNull {
			return fptypes.NewBoolean(true), nil
		}
		return nil, nil
	}
	return fptypes.NewBoolean(listContainsValue(lc, right)), nil
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
			// CQL: Exists returns true if collection has any non-null elements
			for _, v := range list.Values {
				if v != nil {
					return fptypes.NewBoolean(true), nil
				}
			}
			return fptypes.NewBoolean(false), nil
		}
		return fptypes.NewBoolean(true), nil
	case ast.OpNegate:
		if operand == nil {
			return nil, nil
		}
		if i, ok := operand.(fptypes.Integer); ok {
			return fptypes.NewInteger(-i.Value()), nil
		}
		if q, ok := operand.(fptypes.Quantity); ok {
			return fptypes.NewQuantityFromDecimal(q.Value().Neg(), q.Unit()), nil
		}
		d := toDecimal(operand)
		return newDecimalFromD(d.Neg()), nil
	case ast.OpPositive:
		return operand, nil
	case ast.OpDistinct:
		c := toCollection(operand)
		return cqltypes.NewList(nullSafeDistinct(c)), nil
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
		if operand == nil {
			return nil, nil
		}
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
		if item == nil {
			result = append(result, nil)
			continue
		}
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
	// DateTime successor/predecessor: add/subtract 1 unit at the datetime's precision
	if dt, ok := operand.(fptypes.DateTime); ok {
		unit := funcs.TemporalUnit(dt.Precision())
		if op == ast.OpSuccessorOf {
			result := dt.AddDuration(1, unit)
			// Check for overflow (year > 9999)
			if result.Year() > 9999 {
				return nil, fmt.Errorf("successor overflow: DateTime exceeds maximum")
			}
			return result, nil
		}
		result := dt.SubtractDuration(1, unit)
		// Check for underflow
		if result.Year() < 1 {
			return nil, fmt.Errorf("predecessor underflow: DateTime below minimum")
		}
		return result, nil
	}
	// Date successor/predecessor: add/subtract 1 unit at the date's precision
	if dt, ok := operand.(fptypes.Date); ok {
		t := dt.ToTime()
		if op == ast.OpSuccessorOf {
			t = t.AddDate(0, 0, 1)
		} else {
			t = t.AddDate(0, 0, -1)
		}
		return fptypes.NewDate(t.Format("2006-01-02"))
	}
	// Time successor/predecessor: add/subtract 1 unit at time's precision
	if tv, ok := operand.(fptypes.Time); ok {
		delta := 1
		if op != ast.OpSuccessorOf {
			delta = -1
		}
		result := funcs.AdjustTime(tv, delta)
		if result == nil {
			if op == ast.OpSuccessorOf {
				return nil, fmt.Errorf("successor overflow: Time exceeds maximum")
			}
			return nil, fmt.Errorf("predecessor underflow: Time below minimum")
		}
		return result, nil
	}
	// Quantity successor/predecessor
	if q, ok := operand.(fptypes.Quantity); ok {
		epsilon := decimal.RequireFromString("0.00000001")
		if op == ast.OpSuccessorOf {
			return fptypes.NewQuantityFromDecimal(q.Value().Add(epsilon), q.Unit()), nil
		}
		return fptypes.NewQuantityFromDecimal(q.Value().Sub(epsilon), q.Unit()), nil
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
	typeName := nt.Name
	operandType := operand.Type()
	if strings.EqualFold(operandType, typeName) {
		return fptypes.NewBoolean(true), nil
	}
	// CQL: Vocabulary is a supertype of ValueSet and CodeSystem
	if strings.EqualFold(typeName, "Vocabulary") {
		if strings.EqualFold(operandType, "ValueSet") || strings.EqualFold(operandType, "CodeSystem") {
			return fptypes.NewBoolean(true), nil
		}
	}
	return fptypes.NewBoolean(false), nil
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

func (e *Evaluator) evalTypeExtent(n *ast.TypeExtent) (fptypes.Value, error) {
	if n.Type == nil {
		return nil, nil
	}
	typeName := strings.ToLower(n.Type.Name)
	if n.Extent == "minimum" {
		switch typeName {
		case "integer":
			return fptypes.NewInteger(int64(math.MinInt32)), nil
		case "long":
			return fptypes.NewInteger(int64(math.MinInt64)), nil
		case "decimal":
			d := decimal.RequireFromString("-99999999999999999999.99999999")
			return fptypes.NewDecimal(d.String())
		case "datetime":
			return fptypes.NewDateTime("0001-01-01T00:00:00.000")
		case "date":
			return fptypes.NewDate("0001-01-01")
		case "time":
			return fptypes.NewTime("00:00:00.000")
		case "boolean":
			return nil, fmt.Errorf("minimum is not defined for Boolean")
		default:
			return nil, nil
		}
	}
	switch typeName {
	case "integer":
		return fptypes.NewInteger(int64(math.MaxInt32)), nil
	case "long":
		return fptypes.NewInteger(int64(math.MaxInt64)), nil
	case "decimal":
		d := decimal.RequireFromString("99999999999999999999.99999999")
		return fptypes.NewDecimal(d.String())
	case "datetime":
		return fptypes.NewDateTime("9999-12-31T23:59:59.999")
	case "date":
		return fptypes.NewDate("9999-12-31")
	case "time":
		return fptypes.NewTime("23:59:59.999")
	case "boolean":
		return nil, fmt.Errorf("maximum is not defined for Boolean")
	default:
		return nil, nil
	}
}

// ---------------------------------------------------------------------------
// Function calls
// ---------------------------------------------------------------------------

// resolveOverload picks the best FunctionDef matching the given arguments.
// Matches by operand count. Returns first match or first overload as fallback.
func resolveOverload(overloads []*ast.FunctionDef, args []ast.Expression) *ast.FunctionDef {
	if len(overloads) == 1 {
		return overloads[0]
	}

	// First filter by arity
	var candidates []*ast.FunctionDef
	for _, fd := range overloads {
		if len(fd.Operands) == len(args) {
			candidates = append(candidates, fd)
		}
	}
	if len(candidates) == 0 {
		return overloads[0]
	}
	if len(candidates) == 1 {
		return candidates[0]
	}

	// Score by argument type match
	bestScore := -1
	var best *ast.FunctionDef
	for _, fd := range candidates {
		score := 0
		for i, op := range fd.Operands {
			if i < len(args) && op.Type != nil {
				if nt, ok := op.Type.(*ast.NamedType); ok {
					if matchesArgType(args[i], nt.Name) {
						score++
					}
				}
			}
		}
		if score > bestScore {
			bestScore = score
			best = fd
		}
	}
	if best != nil {
		return best
	}
	return candidates[0]
}

// matchesArgType checks if an AST expression matches the expected type name.
func matchesArgType(expr ast.Expression, typeName string) bool {
	lit, ok := expr.(*ast.Literal)
	if !ok {
		return false
	}
	switch lit.ValueType {
	case ast.LiteralInteger:
		return typeName == "Integer" || typeName == "System.Integer"
	case ast.LiteralDecimal:
		return typeName == "Decimal" || typeName == "System.Decimal"
	case ast.LiteralString:
		return typeName == "String" || typeName == "System.String"
	case ast.LiteralBoolean:
		return typeName == "Boolean" || typeName == "System.Boolean"
	case ast.LiteralLong:
		return typeName == "Long" || typeName == "System.Long"
	case ast.LiteralQuantity:
		return typeName == "Quantity" || typeName == "System.Quantity"
	default:
		return false
	}
}

func (e *Evaluator) evalFunctionCall(n *ast.FunctionCall) (fptypes.Value, error) {
	// Check for library-qualified call via Source (e.g. FHIRHelpers.ToQuantity(...))
	// Parser produces: FunctionCall{Source: IdentifierRef{Name: "FHIRHelpers"}, Name: "ToQuantity"}
	if n.Source != nil {
		if idRef, ok := n.Source.(*ast.IdentifierRef); ok {
			if libFuncs, ok := e.includedFuncs[idRef.Name]; ok {
				overloads, ok := libFuncs[n.Name]
				if !ok {
					return nil, fmt.Errorf("function '%s' not found in library '%s'", n.Name, idRef.Name)
				}
				fd := resolveOverload(overloads, n.Operands)
				return e.evalUserFunction(fd, n.Operands)
			}
		}
	}

	// Check for library-qualified call via Library field
	if n.Library != "" {
		if libFuncs, ok := e.includedFuncs[n.Library]; ok {
			overloads, ok := libFuncs[n.Name]
			if !ok {
				return nil, fmt.Errorf("function '%s' not found in library '%s'", n.Name, n.Library)
			}
			fd := resolveOverload(overloads, n.Operands)
			return e.evalUserFunction(fd, n.Operands)
		}
	}

	// Check if it's a library-defined function
	if overloads, ok := e.funcs[n.Name]; ok {
		fd := resolveOverload(overloads, n.Operands)
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

	// resolveSource returns the effective first argument and the remaining operands.
	// For fluent calls (x.func()), source is x and operands are n.Operands.
	// For standalone calls (func(x, ...)), source is nil, so we use Operands[0] as
	// the effective source and Operands[1:] as the remaining operands.
	operands := n.Operands
	resolveSource := func() (fptypes.Value, error) {
		if source != nil {
			return source, nil
		}
		if len(operands) > 0 {
			val, err := e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
			operands = operands[1:]
			return val, nil
		}
		return nil, nil
	}

	switch name {
	case "count":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		c := toCollection(src)
		return funcs.Count(c), nil
	case "exists":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return fptypes.NewBoolean(false), nil
		}
		// CQL: Exists returns true if collection has any non-null elements
		if list, ok := src.(cqltypes.List); ok {
			for _, v := range list.Values {
				if v != nil {
					return fptypes.NewBoolean(true), nil
				}
			}
			return fptypes.NewBoolean(false), nil
		}
		return fptypes.NewBoolean(true), nil
	case "first":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		c := toCollection(src)
		v, _ := c.First()
		return v, nil
	case "last":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		c := toCollection(src)
		v, _ := c.Last()
		return v, nil
	case "where":
		return e.evalWhere(source, n.Operands)
	case "select":
		return e.evalSelect(source, n.Operands)
	case "tostring":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return funcs.ToString(src), nil
	case "tointeger":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src != nil {
			return convertToType(src, "Integer")
		}
		return nil, nil
	case "todecimal":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src != nil {
			return convertToType(src, "Decimal")
		}
		return nil, nil
	case "not":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		return fptypes.NewBoolean(!isTrue(src)), nil
	case "length":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			// CQL: Length(null string) = null, Length(null list) = 0
			// Check if the argument is typed as a list (via "as List<...>" cast)
			if len(n.Operands) > 0 {
				if asExpr, ok := n.Operands[0].(*ast.AsExpression); ok {
					if _, ok := asExpr.Type.(*ast.ListType); ok {
						return fptypes.NewInteger(0), nil
					}
				}
			}
			return nil, nil
		}
		// String length
		if s, ok := src.(fptypes.String); ok {
			return fptypes.NewInteger(int64(len(s.Value()))), nil
		}
		// List length — count all elements including nulls
		if list, ok := src.(cqltypes.List); ok {
			return fptypes.NewInteger(int64(len(list.Values))), nil
		}
		c := toCollection(src)
		return fptypes.NewInteger(int64(c.Count())), nil
	case "coalesce":
		// Coalesce checks source first (for fluent), then all operands.
		// If given a single list argument, iterate its items.
		if source != nil {
			return source, nil
		}
		for _, arg := range n.Operands {
			val, err := e.Eval(arg)
			if err != nil {
				return nil, err
			}
			if val != nil {
				// If the single argument is a list, iterate its items
				if len(n.Operands) == 1 {
					c := toCollection(val)
					if c != nil {
						for _, item := range c {
							if item != nil {
								return item, nil
							}
						}
						return nil, nil
					}
				}
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
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return e.evalAggregateSum(src)
	case "avg":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return e.evalAggregateAvg(src)
	case "min":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return e.evalAggregateMinMax(src, true)
	case "max":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return e.evalAggregateMinMax(src, false)
	case "abs":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return e.evalAbs(src)
	case "flatten":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return e.evalFlatten(src), nil
	case "distinct":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		c := toCollection(src)
		return cqltypes.NewList(nullSafeDistinct(c)), nil

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
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return funcs.Upper(src), nil
	case "lower":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return funcs.Lower(src), nil
	case "startswith":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) > 0 {
			arg, err := e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.StartsWith(src, arg), nil
		}
		return fptypes.NewBoolean(false), nil
	case "endswith":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) > 0 {
			arg, err := e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.EndsWith(src, arg), nil
		}
		return fptypes.NewBoolean(false), nil
	case "indexof":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) > 0 {
			arg, err := e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
			// CQL spec: IndexOf with a null argument returns null
			if arg == nil {
				return nil, nil
			}
			// If source is a list/collection, do list IndexOf
			if _, isList := src.(cqltypes.List); isList {
				c := toCollection(src)
				for i, item := range c {
					if item != nil && item.Equal(arg) {
						return fptypes.NewInteger(int64(i)), nil
					}
				}
				return fptypes.NewInteger(-1), nil
			}
			return funcs.IndexOf(src, arg), nil
		}
		return fptypes.NewInteger(-1), nil
	case "matches":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) > 0 {
			arg, err := e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.Matches(src, arg), nil
		}
		return fptypes.NewBoolean(false), nil
	case "replacematches":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) >= 2 {
			pat, err := e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
			rep, err := e.Eval(operands[1])
			if err != nil {
				return nil, err
			}
			return funcs.ReplaceMatches(src, pat, rep), nil
		}
		return src, nil
	case "combine":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		c := toCollection(src)
		sep := ""
		if len(operands) > 0 {
			s, err := e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
			if s != nil {
				sep = s.String()
			}
		}
		return funcs.Combine(c, sep), nil
	case "split":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) > 0 {
			sep, err := e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
			if sep != nil {
				return funcs.Split(src, sep.String()), nil
			}
			// null separator → return list with source as single element
			return funcs.SplitNull(src), nil
		}
		return src, nil

	// Statistical aggregate functions
	case "alltrue":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		c := toCollection(src)
		return funcs.AllTrue(c), nil
	case "anytrue":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		c := toCollection(src)
		return funcs.AnyTrue(c), nil
	case "populationstddev":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		c := toCollection(src)
		return funcs.PopulationStdDev(c), nil
	case "populationvariance":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		c := toCollection(src)
		return funcs.PopulationVariance(c), nil
	case "stddev":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		c := toCollection(src)
		return funcs.StdDev(c), nil
	case "variance":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		c := toCollection(src)
		return funcs.Variance(c), nil

	// Temporal functions
	case "timeofday":
		return funcs.TimeOfDay()

	// Advanced string functions
	case "positionof":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) > 0 {
			pattern, err := e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.PositionOf(src, pattern), nil
		}
		return fptypes.NewInteger(-1), nil
	case "lastpositionof":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) > 0 {
			pattern, err := e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.LastPositionOf(src, pattern), nil
		}
		return fptypes.NewInteger(-1), nil
	case "substring":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) > 0 {
			start, err := e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
			if start == nil {
				return nil, nil // null start index → null
			}
			length := -1
			if len(operands) > 1 {
				l, err := e.Eval(operands[1])
				if err != nil {
					return nil, err
				}
				if l == nil {
					return nil, nil // null length → null
				}
				if li, ok := l.(fptypes.Integer); ok {
					length = int(li.Value())
				}
			}
			startIdx := 0
			if si, ok := start.(fptypes.Integer); ok {
				startIdx = int(si.Value())
			}
			return funcs.Substring(src, startIdx, length), nil
		}
		return src, nil

	// Advanced list functions
	case "mode":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		c := toCollection(src)
		return funcs.Mode(c), nil
	case "median":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		c := toCollection(src)
		return funcs.Median(c), nil
	case "geometricmean":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		c := toCollection(src)
		return funcs.GeometricMean(c), nil
	case "tail":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		c := toCollection(src)
		return cqltypes.NewList(funcs.Tail(c)), nil
	case "take":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		if len(operands) > 0 {
			arg, err := e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
			if arg == nil {
				return cqltypes.NewList(fptypes.Collection{}), nil
			}
			if ai, ok := arg.(fptypes.Integer); ok {
				c := toCollection(src)
				return cqltypes.NewList(funcs.Take(c, int(ai.Value()))), nil
			}
		}
		return src, nil
	case "skip":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		if len(operands) > 0 {
			arg, err := e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
			if ai, ok := arg.(fptypes.Integer); ok {
				c := toCollection(src)
				return cqltypes.NewList(funcs.Skip(c, int(ai.Value()))), nil
			}
		}
		return src, nil

	// Null operators
	case "isnull":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return IsNull(src), nil
	case "istrue":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return IsTrue(src), nil
	case "isfalse":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return IsFalse(src), nil

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
	case "time":
		if len(n.Operands) >= 1 {
			args := make([]fptypes.Value, 4)
			for i := 0; i < len(n.Operands) && i < 4; i++ {
				var err error
				args[i], err = e.Eval(n.Operands[i])
				if err != nil {
					return nil, err
				}
			}
			return funcs.TimeConstructor(args[0], args[1], args[2], args[3])
		}
		return nil, nil

	// Interval functions
	case "width":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if iv, ok := src.(cqltypes.Interval); ok {
			return funcs.IntervalWidth(iv)
		}
		return nil, nil
	case "size":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if iv, ok := src.(cqltypes.Interval); ok {
			return funcs.IntervalSize(iv)
		}
		return nil, nil

	// Math functions
	case "round":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		precision := 0
		if len(operands) > 0 {
			pv, err := e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
			if pi, ok := pv.(fptypes.Integer); ok {
				precision = int(pi.Value())
			}
		}
		return funcs.Round(src, precision)

	case "floor":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return funcs.Floor(src)

	case "ceiling":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return funcs.Ceiling(src)

	case "truncate":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return funcs.Truncate(src)

	case "ln":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return funcs.Ln(src)

	case "log":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) < 1 {
			return nil, fmt.Errorf("log requires a base argument")
		}
		base, err := e.Eval(operands[0])
		if err != nil {
			return nil, err
		}
		return funcs.Log(src, base)

	case "exp":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return funcs.Exp(src)

	case "power":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) < 1 {
			return nil, fmt.Errorf("power requires an exponent argument")
		}
		exp, err := e.Eval(operands[0])
		if err != nil {
			return nil, err
		}
		return funcs.Power(src, exp)

	case "precision":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return funcs.Precision(src)

	case "highboundary":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		var prec fptypes.Value
		if len(operands) > 0 {
			prec, err = e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
		}
		return funcs.HighBoundary(src, prec)

	case "lowboundary":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		var prec fptypes.Value
		if len(operands) > 0 {
			prec, err = e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
		}
		return funcs.LowBoundary(src, prec)

	// Indexer
	case "indexer":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) > 0 {
			arg, err := e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
			if ai, ok := arg.(fptypes.Integer); ok {
				idx := int(ai.Value())
				// String indexer: return character at index
				if sv, ok := src.(fptypes.String); ok {
					str := sv.Value()
					if idx < 0 || idx >= len(str) {
						return nil, nil
					}
					return fptypes.NewString(string(str[idx])), nil
				}
				// Collection indexer
				c := toCollection(src)
				return funcs.Indexer(c, idx), nil
			}
		}
		return nil, nil

	// Concatenate
	case "concatenate":
		var result strings.Builder
		allArgs := make([]fptypes.Value, 0, 1+len(n.Operands))
		if source != nil {
			allArgs = append(allArgs, source)
		}
		for _, op := range n.Operands {
			v, err := e.Eval(op)
			if err != nil {
				return nil, err
			}
			if v == nil {
				return nil, nil // null propagation
			}
			allArgs = append(allArgs, v)
		}
		for _, arg := range allArgs {
			result.WriteString(arg.String())
		}
		return fptypes.NewString(result.String()), nil

	// Conversion functions
	case "todatetime":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return funcs.ToDateTime(src)
	case "totime":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return funcs.ToTime(src)
	case "toboolean":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return funcs.ToBoolean(src)
	case "toquantity":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return funcs.ToQuantity(src)
	case "toconcept":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return funcs.ToConcept(src)

	// Message(source, condition, code, severity, message) — returns the first argument
	case "message":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		// Evaluate remaining operands
		var condition fptypes.Value
		var severity, msg string
		for i, op := range operands {
			val, err := e.Eval(op)
			if err != nil {
				return nil, err
			}
			switch i {
			case 0:
				condition = val
			case 2:
				if s, ok := val.(fptypes.String); ok {
					severity = s.Value()
				}
			case 3:
				if s, ok := val.(fptypes.String); ok {
					msg = s.Value()
				}
			}
		}
		// If condition is true and severity is Error, raise an error
		if isTrue(condition) && strings.EqualFold(severity, "Error") {
			return nil, fmt.Errorf("CQL Message error: %s", msg)
		}
		return src, nil

	// Product aggregate
	case "product":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		return e.evalAggregateProduct(src)

	// ConvertsTo* predicates (CQL 1.5.3 §22)
	// Null in → null out. Never errors. Returns true if conversion would succeed.
	case "convertstostring":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		return fptypes.NewBoolean(true), nil

	case "convertstoboolean":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		// Per CQL spec, only 0 and 1 convert to Boolean for Integer/Decimal.
		// funcs.ToBoolean is too permissive (any non-zero → true), so we check
		// integer range explicitly before delegating.
		switch val := src.(type) {
		case fptypes.Integer:
			v := val.Value()
			return fptypes.NewBoolean(v == 0 || v == 1), nil
		case fptypes.Decimal:
			v := val.Value()
			return fptypes.NewBoolean(v.IsZero() || v.Equal(decimal.NewFromInt(1))), nil
		default:
			result, convErr := funcs.ToBoolean(src)
			return fptypes.NewBoolean(convErr == nil && result != nil), nil
		}

	case "convertstointeger":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		result, convErr := convertToType(src, "integer")
		return fptypes.NewBoolean(convErr == nil && result != nil), nil

	case "convertstodecimal":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		result, convErr := convertToType(src, "decimal")
		return fptypes.NewBoolean(convErr == nil && result != nil), nil

	case "convertstolong":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		switch v := src.(type) {
		case fptypes.Integer:
			return fptypes.NewBoolean(true), nil
		case fptypes.String:
			_, parseErr := strconv.ParseInt(v.Value(), 10, 64)
			return fptypes.NewBoolean(parseErr == nil), nil
		default:
			return fptypes.NewBoolean(false), nil
		}

	case "convertstoquantity":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		result, convErr := funcs.ToQuantity(src)
		return fptypes.NewBoolean(convErr == nil && result != nil), nil

	case "convertstodate":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		switch v := src.(type) {
		case fptypes.Date:
			return fptypes.NewBoolean(true), nil
		case fptypes.DateTime:
			return fptypes.NewBoolean(true), nil
		case fptypes.String:
			_, parseErr := fptypes.NewDate(v.Value())
			return fptypes.NewBoolean(parseErr == nil), nil
		default:
			return fptypes.NewBoolean(false), nil
		}

	case "convertstodatetime":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		switch v := src.(type) {
		case fptypes.DateTime:
			return fptypes.NewBoolean(true), nil
		case fptypes.Date:
			return fptypes.NewBoolean(true), nil
		case fptypes.String:
			_, parseErr := fptypes.NewDateTime(v.Value())
			return fptypes.NewBoolean(parseErr == nil), nil
		default:
			return fptypes.NewBoolean(false), nil
		}

	case "convertstotime":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		switch v := src.(type) {
		case fptypes.Time:
			return fptypes.NewBoolean(true), nil
		case fptypes.String:
			result, convErr := funcs.ToTime(v)
			return fptypes.NewBoolean(convErr == nil && result != nil), nil
		default:
			return fptypes.NewBoolean(false), nil
		}

	case "convertstoratio":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		if _, ok := src.(cqltypes.Ratio); ok {
			return fptypes.NewBoolean(true), nil
		}
		s, ok := src.(fptypes.String)
		if !ok {
			return fptypes.NewBoolean(false), nil
		}
		parts := strings.SplitN(s.Value(), ":", 2)
		if len(parts) != 2 {
			return fptypes.NewBoolean(false), nil
		}
		n, errN := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		d, errD := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		valid := errN == nil && errD == nil && !math.IsInf(n, 0) && !math.IsNaN(n) && !math.IsInf(d, 0) && !math.IsNaN(d)
		return fptypes.NewBoolean(valid), nil

	case "anyinvalueset":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) < 1 {
			return nil, fmt.Errorf("AnyInValueSet requires a valueset argument")
		}
		vsName := ""
		if idRef, ok := operands[0].(*ast.IdentifierRef); ok {
			vsName = idRef.Name
		} else {
			val, evalErr := e.Eval(operands[0])
			if evalErr != nil {
				return nil, evalErr
			}
			if s, ok := val.(fptypes.String); ok {
				vsName = s.Value()
			}
		}
		if vsName == "" {
			return nil, fmt.Errorf("AnyInValueSet: could not resolve valueset reference")
		}
		return e.evalAnyInValueSet(src, vsName)

	case "anyincodesystem":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) < 1 {
			return nil, fmt.Errorf("AnyInCodeSystem requires a codesystem argument")
		}
		csName := ""
		if idRef, ok := operands[0].(*ast.IdentifierRef); ok {
			csName = idRef.Name
		} else {
			val, evalErr := e.Eval(operands[0])
			if evalErr != nil {
				return nil, evalErr
			}
			if s, ok := val.(fptypes.String); ok {
				csName = s.Value()
			}
		}
		if csName == "" {
			return nil, fmt.Errorf("AnyInCodeSystem: could not resolve codesystem reference")
		}
		return e.evalAnyInCodeSystem(src, csName)

	// descendents/descendants — returns all descendant elements (CQL spec).
	// On null, returns null. On non-null, returns empty list (simplified).
	case "descendents", "descendants":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		return cqltypes.NewList(fptypes.Collection{}), nil

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
		if result.Count() > 0 {
			if result.Count() == 1 {
				return result[0], nil
			}
			return cqltypes.NewList(result), nil
		}

		// Choice type resolution: check ModelInfo for value[x] patterns
		if e.ctx.ModelInfo != nil {
			typeName := obj.Type() // e.g. "Observation"
			path := typeName + "." + n.Member
			if e.ctx.ModelInfo.IsChoiceType(path) {
				if ei, ok := e.ctx.ModelInfo.ElementInfoByPath(path); ok {
					for _, choiceType := range ei.ChoiceTypes {
						// Extract suffix: "FHIR.Quantity" → "Quantity"
						suffix := choiceType
						if idx := strings.LastIndex(choiceType, "."); idx >= 0 {
							suffix = choiceType[idx+1:]
						}
						concreteKey := n.Member + suffix
						result = obj.GetCollection(concreteKey)
						if result.Count() > 0 {
							if result.Count() == 1 {
								return result[0], nil
							}
							return cqltypes.NewList(result), nil
						}
					}
				}
			}
		}

		return nil, nil
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

	// Evaluate all sources and build their collections.
	allSources := make([]fptypes.Collection, len(n.Sources))
	var firstSource fptypes.Value
	for idx, src := range n.Sources {
		val, err := e.Eval(src.Source)
		if err != nil {
			return nil, err
		}
		if idx == 0 {
			firstSource = val
		}
		allSources[idx] = toCollection(val)
	}
	_, sourceIsList := firstSource.(cqltypes.List)

	// Build the cartesian product of all sources as a list of alias maps.
	combos := []queryCombo{{aliases: make(map[string]fptypes.Value)}}
	for idx, src := range n.Sources {
		var next []queryCombo
		for _, c := range combos {
			for _, item := range allSources[idx] {
				newAliases := make(map[string]fptypes.Value, len(c.aliases)+1)
				for k, v := range c.aliases {
					newAliases[k] = v
				}
				newAliases[src.Alias] = item
				next = append(next, queryCombo{aliases: newAliases})
			}
		}
		combos = next
	}

	// Process each combination through filters and return/aggregate.
	var results fptypes.Collection
	for i, c := range combos {
		child := e.ctx.ChildScope()
		for alias, val := range c.aliases {
			child.Aliases[alias] = val
		}
		// Set This and Index to the first source's item
		child.This = c.aliases[n.Sources[0].Alias]
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
		if n.Aggregate == nil {
			switch {
			case n.Return != nil:
				val, err := childEval.Eval(n.Return.Expression)
				if err != nil {
					return nil, err
				}
				if val != nil {
					results = append(results, val)
				}
			case len(n.Sources) > 1:
				// Multi-source query without return: produce a Tuple with all aliases
				tupleElems := make(map[string]fptypes.Value, len(n.Sources))
				for _, src := range n.Sources {
					tupleElems[src.Alias] = c.aliases[src.Alias]
				}
				results = append(results, cqltypes.NewTuple(tupleElems))
			default:
				results = append(results, child.This)
			}
		}
	}

	// Handle aggregate clause - reduction over filtered combos
	if n.Aggregate != nil {
		// Evaluate the starting value
		var accumulator fptypes.Value
		if n.Aggregate.Starting != nil {
			var err error
			accumulator, err = e.Eval(n.Aggregate.Starting)
			if err != nil {
				return nil, err
			}
		}

		// Apply distinct to the cartesian product combos if requested
		aggCombos := combos
		if n.Aggregate.Distinct {
			aggCombos = distinctCombos(aggCombos, n.Sources)
		}

		for i, c := range aggCombos {
			child := e.ctx.ChildScope()
			for alias, val := range c.aliases {
				child.Aliases[alias] = val
			}
			child.This = c.aliases[n.Sources[0].Alias]
			child.Index = i
			child.Aliases[n.Aggregate.Identifier] = accumulator

			// Process let bindings
			childEval := e.withContext(child)
			for _, let := range n.Let {
				val, err := childEval.Eval(let.Expression)
				if err != nil {
					return nil, err
				}
				child.LetBindings[let.Identifier] = val
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

			val, err := childEval.Eval(n.Aggregate.Expression)
			if err != nil {
				return nil, err
			}
			accumulator = val
		}
		return accumulator, nil
	}

	// Apply distinct if specified
	if n.Return != nil && n.Return.Distinct {
		results = nullSafeDistinct(results)
	}

	// Apply sort clause
	if n.Sort != nil {
		var sortErr error
		sort.SliceStable(results, func(i, j int) bool {
			if sortErr != nil {
				return false
			}
			if len(n.Sort.ByItems) > 0 {
				// Sort by explicit expressions
				for _, byItem := range n.Sort.ByItems {
					cmpResult, err := e.compareSortKeys(n.Sources[0].Alias, results[i], results[j], byItem.Expression)
					if err != nil {
						sortErr = err
						return false
					}
					if cmpResult == 0 {
						continue
					}
					if byItem.Direction == ast.SortDesc {
						return cmpResult > 0
					}
					return cmpResult < 0
				}
				return false
			}
			// Sort without 'by' — compare items directly
			cmpResult, err := compareValues(results[i], results[j])
			if err != nil {
				sortErr = err
				return false
			}
			if n.Sort.Direction == ast.SortDesc {
				return cmpResult > 0
			}
			return cmpResult < 0
		})
		if sortErr != nil {
			return nil, sortErr
		}
	}

	// CQL: if the source was a single scalar value (not a list), return a scalar
	if !sourceIsList && len(n.Sources) == 1 && len(results) == 1 {
		return results[0], nil
	}
	return cqltypes.NewList(results), nil
}

// distinctCombos removes duplicate alias combinations based on value equality.
func distinctCombos(combos []queryCombo, sources []*ast.AliasedSource) []queryCombo {
	var result []queryCombo
	seen := make(map[string]bool)
	for _, c := range combos {
		var key string
		for _, src := range sources {
			v := c.aliases[src.Alias]
			if v == nil {
				key += "nil,"
			} else {
				key += v.String() + ","
			}
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, c)
		}
	}
	return result
}

// compareSortKeys evaluates a sort expression against two items and returns their comparison.
func (e *Evaluator) compareSortKeys(alias string, a, b fptypes.Value, expr ast.Expression) (int, error) {
	scopeA := e.ctx.ChildScope()
	scopeA.Aliases[alias] = a
	scopeA.This = a
	keyA, err := e.withContext(scopeA).Eval(expr)
	if err != nil {
		return 0, err
	}

	scopeB := e.ctx.ChildScope()
	scopeB.Aliases[alias] = b
	scopeB.This = b
	keyB, err := e.withContext(scopeB).Eval(expr)
	if err != nil {
		return 0, err
	}

	return compareValues(keyA, keyB)
}

// compareValues returns -1, 0, or 1 for two values. Nulls sort last (after all non-null values).
// For temporal values with different precisions (ambiguous comparison), falls back to
// comparing at the shared precision; when equal, lower precision sorts first.
func compareValues(a, b fptypes.Value) (int, error) {
	if a == nil && b == nil {
		return 0, nil
	}
	if a == nil {
		return 1, nil // nulls sort last
	}
	if b == nil {
		return -1, nil
	}
	ac, ok := a.(fptypes.Comparable)
	if !ok {
		return 0, fmt.Errorf("cannot compare type %s for sorting", a.Type())
	}
	result, err := ac.Compare(b)
	if err != nil && isAmbiguousComparisonErr(err) {
		// Fall back to component-wise comparison at shared precision
		aComps, aMaxPrec := temporalComponents(a)
		bComps, bMaxPrec := temporalComponents(b)
		if aComps != nil && bComps != nil {
			minPrec := aMaxPrec
			if bMaxPrec < minPrec {
				minPrec = bMaxPrec
			}
			for i := 0; i <= minPrec; i++ {
				if aComps[i] < bComps[i] {
					return -1, nil
				}
				if aComps[i] > bComps[i] {
					return 1, nil
				}
			}
			// Equal at shared precision: lower precision sorts first (less specific before more specific)
			if aMaxPrec < bMaxPrec {
				return -1, nil
			}
			if aMaxPrec > bMaxPrec {
				return 1, nil
			}
			return 0, nil
		}
		return 0, nil // can't extract components, treat as equal
	}
	return result, err
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
	// CQL: Interval[null, null] evaluates to null
	if low == nil && high == nil {
		return nil, nil
	}
	// Validate: if both bounds are non-null, low must not exceed high
	if low != nil && high != nil {
		if comp, ok := low.(fptypes.Comparable); ok {
			cmp, cmpErr := comp.Compare(high)
			if cmpErr == nil {
				if cmp > 0 {
					return nil, fmt.Errorf("invalid interval: low bound (%v) is greater than high bound (%v)", low, high)
				}
				// Check for empty interval: Interval[5, 5) or Interval(5, 5] where low==high but one side is open
				if cmp == 0 && (!n.LowClosed || !n.HighClosed) {
					return nil, fmt.Errorf("invalid interval: interval is empty (low equals high with open boundary)")
				}
			}
		}
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

	// Special case: Quantity { value: ..., unit: ... } → produce a real Quantity value
	if n.Type != nil && strings.EqualFold(n.Type.Name, "Quantity") {
		valElem := elements["value"]
		unitElem := elements["unit"]
		if valElem != nil && unitElem != nil {
			numVal := toDecimal(valElem)
			unitStr := ""
			if us, ok := unitElem.(fptypes.String); ok {
				unitStr = us.Value()
			}
			return fptypes.NewQuantityFromDecimal(numVal, unitStr), nil
		}
	}

	t := cqltypes.NewTuple(elements)
	// Preserve the instance type name (e.g., "ValueSet", "CodeSystem")
	if n.Type != nil && n.Type.Name != "" {
		t.TypeOverride = n.Type.Name
	}
	return t, nil
}

func (e *Evaluator) evalListExpr(n *ast.ListExpression) (fptypes.Value, error) {
	values := make(fptypes.Collection, 0, len(n.Elements))
	for _, elem := range n.Elements {
		val, err := e.Eval(elem)
		if err != nil {
			return nil, err
		}
		// CQL lists preserve null elements
		values = append(values, val)
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
	// Pass through to evalInContains which handles null properly
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
		if isAmbiguousComparisonErr(err) {
			return nil, nil
		}
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
	// For includes/includedIn/contains/in with lists, don't short-circuit on nil
	// Handle list-based operations first (before null propagation)
	leftList, leftIsList := left.(cqltypes.List)
	rightList, rightIsList := right.(cqltypes.List)
	if leftIsList || rightIsList {
		// For includes/includedIn: null scalar with list needs special handling
		switch n.Operator.Kind {
		case ast.TimingIncludes:
			if leftIsList && !rightIsList {
				// list includes null-scalar: check if null is in the list
				return e.evalListTimingOp(leftList, rightList, leftIsList, rightIsList, left, right, n.Operator)
			}
			if !leftIsList && left == nil {
				// null includes list → null
				return nil, nil
			}
		case ast.TimingIncludedIn, ast.TimingDuring:
			if rightIsList && !leftIsList {
				// null-scalar included in list: check if null is in the list
				return e.evalListTimingOp(leftList, rightList, leftIsList, rightIsList, left, right, n.Operator)
			}
			if !rightIsList && right == nil {
				// list included in null → null
				return nil, nil
			}
		}
		return e.evalListTimingOp(leftList, rightList, leftIsList, rightIsList, left, right, n.Operator)
	}

	// Special handling for properly includes/included in with null intervals
	// CQL: null interval in properly includes/included in is treated as unbounded (universal)
	if left == nil || right == nil {
		switch n.Operator.Kind {
		case ast.TimingIncludes:
			if n.Operator.Properly && left == nil && right != nil {
				// null properly includes X → X is a proper subset of the universal interval → true
				if _, rightIsIv := right.(cqltypes.Interval); rightIsIv {
					return fptypes.NewBoolean(true), nil
				}
			}
		case ast.TimingIncludedIn, ast.TimingDuring:
			if n.Operator.Properly && right == nil && left != nil {
				// X properly included in null → X is a proper subset of the universal interval → true
				if _, leftIsIv := left.(cqltypes.Interval); leftIsIv {
					return fptypes.NewBoolean(true), nil
				}
			}
		}
		// Default null propagation for non-list operations
		return nil, nil
	}

	// Handle scalar temporal types (DateTime, Date, Time) with precision-aware comparison
	if isTemporalType(left) && isTemporalType(right) {
		return e.evalTemporalComparison(left, right, n.Operator)
	}

	leftIv, leftOk := left.(cqltypes.Interval)
	rightIv, rightOk := right.(cqltypes.Interval)

	// Handle Interval vs scalar DateTime for timing operations
	if leftOk && !rightOk && isTemporalType(right) {
		switch n.Operator.Kind {
		case ast.TimingBeforeOrAfter:
			if n.Operator.Before {
				// Interval before scalar: interval.High before scalar
				return e.evalTemporalComparison(leftIv.High, right, n.Operator)
			}
			// Interval after scalar: interval.Low after scalar
			return e.evalTemporalComparison(leftIv.Low, right, n.Operator)
		case ast.TimingSameAs:
			if n.Operator.Before {
				// Interval same or before scalar: interval.High same or before scalar
				return e.evalTemporalComparison(leftIv.High, right, n.Operator)
			}
			if n.Operator.After {
				// Interval same or after scalar: interval.Low same or after scalar
				return e.evalTemporalComparison(leftIv.Low, right, n.Operator)
			}
		}
	}
	if !leftOk && rightOk && isTemporalType(left) {
		switch n.Operator.Kind {
		case ast.TimingBeforeOrAfter:
			if n.Operator.Before {
				// scalar before Interval: scalar before interval.Low
				return e.evalTemporalComparison(left, rightIv.Low, n.Operator)
			}
			// scalar after Interval: scalar after interval.High
			return e.evalTemporalComparison(left, rightIv.High, n.Operator)
		case ast.TimingSameAs:
			if n.Operator.Before {
				return e.evalTemporalComparison(left, rightIv.Low, n.Operator)
			}
			if n.Operator.After {
				return e.evalTemporalComparison(left, rightIv.High, n.Operator)
			}
		}
	}

	// Handle "interval properly includes point" and "point properly included in interval"
	// CQL: properly contains/includes a point means the point is strictly interior
	// (not equal to either boundary).
	if leftOk && !rightOk && n.Operator.Kind == ast.TimingIncludes && n.Operator.Properly {
		return evalIntervalProperlyContainsPoint(leftIv, right, n.Operator.Precision)
	}
	if !leftOk && rightOk && (n.Operator.Kind == ast.TimingIncludedIn || n.Operator.Kind == ast.TimingDuring) && n.Operator.Properly {
		return evalIntervalProperlyContainsPoint(rightIv, left, n.Operator.Precision)
	}

	// Handle scalar vs interval for non-temporal types (e.g., 9 before Interval[11, 20])
	if !leftOk && rightOk {
		// Promote scalar to point interval [x, x]
		leftIv = cqltypes.NewInterval(left, left, true, true)
		leftOk = true
	}
	if leftOk && !rightOk {
		// Promote scalar to point interval [x, x]
		rightIv = cqltypes.NewInterval(right, right, true, true)
		rightOk = true
	}
	if !leftOk || !rightOk {
		return nil, nil
	}
	switch n.Operator.Kind {
	case ast.TimingSameAs:
		if n.Operator.Before {
			return funcs.SameOrBefore(leftIv, rightIv)
		}
		if n.Operator.After {
			return funcs.SameOrAfter(leftIv, rightIv)
		}
		return fptypes.NewBoolean(leftIv.Equal(rightIv)), nil
	case ast.TimingIncludes:
		if n.Operator.Properly {
			res, err := funcs.IntervalProperlyIncludes(leftIv, rightIv)
			if err != nil {
				return nil, err
			}
			if res == nil && n.Operator.Precision != "" {
				return intervalIncludesAtPrecision(leftIv, rightIv, n.Operator.Precision, true)
			}
			return res, nil
		}
		result, err := leftIv.Includes(rightIv)
		if err != nil {
			if isAmbiguousComparisonErr(err) {
				if n.Operator.Precision != "" {
					return intervalIncludesAtPrecision(leftIv, rightIv, n.Operator.Precision, false)
				}
				return nil, nil
			}
			return nil, err
		}
		return fptypes.NewBoolean(result), nil
	case ast.TimingIncludedIn, ast.TimingDuring:
		if n.Operator.Properly {
			res, err := funcs.IntervalProperlyIncludedIn(leftIv, rightIv)
			if err != nil {
				return nil, err
			}
			if res == nil && n.Operator.Precision != "" {
				return intervalIncludesAtPrecision(rightIv, leftIv, n.Operator.Precision, true)
			}
			return res, nil
		}
		result, err := rightIv.Includes(leftIv)
		if err != nil {
			if isAmbiguousComparisonErr(err) {
				if n.Operator.Precision != "" {
					return intervalIncludesAtPrecision(rightIv, leftIv, n.Operator.Precision, false)
				}
				return nil, nil
			}
			return nil, err
		}
		return fptypes.NewBoolean(result), nil
	case ast.TimingBeforeOrAfter:
		if n.Operator.Before {
			return funcs.IntervalBefore(leftIv, rightIv)
		}
		return funcs.IntervalAfter(leftIv, rightIv)
	case ast.TimingMeets:
		if n.Operator.Before {
			return funcs.IntervalMeetsBefore(leftIv, rightIv)
		}
		if n.Operator.After {
			return funcs.IntervalMeetsAfter(leftIv, rightIv)
		}
		return funcs.IntervalMeets(leftIv, rightIv)
	case ast.TimingOverlaps:
		if n.Operator.Before {
			return funcs.OverlapsBefore(leftIv, rightIv)
		}
		if n.Operator.After {
			return funcs.OverlapsAfter(leftIv, rightIv)
		}
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

// listContainsValue checks if a collection contains a value (including nil/null).
func listContainsValue(c fptypes.Collection, val fptypes.Value) bool {
	return nullSafeContains(c, val)
}

// intervalIncludesAtPrecision checks if outer includes inner at the given precision.
// Compares interval bounds by truncating to the specified precision.
func intervalIncludesAtPrecision(outer, inner cqltypes.Interval, precision string, properly bool) (fptypes.Value, error) {
	pIdx := precisionIndex(precision)
	if pIdx < 0 {
		return nil, nil
	}

	cmpAtPrec := func(a, b fptypes.Value) (int, bool) {
		aComps, aMax := temporalComponents(a)
		bComps, bMax := temporalComponents(b)
		if aComps == nil || bComps == nil {
			return 0, false
		}
		// If the requested precision exceeds either operand's precision, result is ambiguous
		if pIdx > aMax || pIdx > bMax {
			return 0, false
		}
		for i := 0; i <= pIdx; i++ {
			if aComps[i] < bComps[i] {
				return -1, true
			}
			if aComps[i] > bComps[i] {
				return 1, true
			}
		}
		return 0, true
	}

	// Check outer.Low <= inner.Low (at precision)
	if outer.Low != nil && inner.Low != nil {
		cmp, ok := cmpAtPrec(inner.Low, outer.Low)
		if !ok {
			return nil, nil
		}
		if cmp < 0 {
			return fptypes.NewBoolean(false), nil
		}
		if cmp == 0 && inner.LowClosed && !outer.LowClosed {
			return fptypes.NewBoolean(false), nil
		}
	}
	// Check inner.High <= outer.High (at precision)
	if outer.High != nil && inner.High != nil {
		cmp, ok := cmpAtPrec(inner.High, outer.High)
		if !ok {
			return nil, nil
		}
		if cmp > 0 {
			return fptypes.NewBoolean(false), nil
		}
		if cmp == 0 && inner.HighClosed && !outer.HighClosed {
			return fptypes.NewBoolean(false), nil
		}
	}

	if properly {
		// For properly includes, outer must be strictly larger.
		// Check that at least one bound is strictly different.
		outerEqualsInner := true
		if outer.Low != nil && inner.Low != nil {
			cmp, ok := cmpAtPrec(outer.Low, inner.Low)
			if ok && cmp != 0 {
				outerEqualsInner = false
			}
		}
		if outerEqualsInner && outer.High != nil && inner.High != nil {
			cmp, ok := cmpAtPrec(outer.High, inner.High)
			if ok && cmp != 0 {
				outerEqualsInner = false
			}
		}
		if outerEqualsInner {
			return fptypes.NewBoolean(false), nil
		}
	}

	return fptypes.NewBoolean(true), nil
}

// evalIntervalProperlyContainsPoint checks if an interval properly contains a point.
// CQL: point must be contained AND not equal to either boundary.
// When a precision is specified, comparisons are truncated to that precision.
func evalIntervalProperlyContainsPoint(iv cqltypes.Interval, point fptypes.Value, precision string) (fptypes.Value, error) {
	if point == nil {
		return nil, nil
	}
	// First check if the interval contains the point
	contained, err := iv.Contains(point)
	if err != nil {
		if isAmbiguousComparisonErr(err) {
			// With precision specified, try comparing at that precision
			if precision != "" {
				return evalIntervalProperlyContainsPointAtPrecision(iv, point, precision)
			}
			return nil, nil
		}
		return nil, err
	}
	if !contained {
		return fptypes.NewBoolean(false), nil
	}
	// Check that point is NOT equal to the low or high boundary
	if iv.Low != nil && iv.LowClosed && point.Equal(iv.Low) {
		return fptypes.NewBoolean(false), nil
	}
	if iv.High != nil && iv.HighClosed && point.Equal(iv.High) {
		return fptypes.NewBoolean(false), nil
	}
	return fptypes.NewBoolean(true), nil
}

// evalIntervalProperlyContainsPointAtPrecision checks proper containment at a given precision.
func evalIntervalProperlyContainsPointAtPrecision(iv cqltypes.Interval, point fptypes.Value, precision string) (fptypes.Value, error) {
	pIdx := precisionIndex(precision)
	if pIdx < 0 {
		return nil, nil
	}
	pointComps, pointMaxPrec := temporalComponents(point)
	if pointComps == nil {
		return nil, nil
	}
	// If the requested precision exceeds the point's precision, result is null (ambiguous)
	if pIdx > pointMaxPrec {
		return nil, nil
	}
	cmpPrec := pIdx

	// Check low bound
	if iv.Low != nil {
		lowComps, _ := temporalComponents(iv.Low)
		if lowComps != nil {
			cmpResult := 0
			for i := 0; i <= cmpPrec; i++ {
				if pointComps[i] < lowComps[i] {
					cmpResult = -1
					break
				}
				if pointComps[i] > lowComps[i] {
					cmpResult = 1
					break
				}
			}
			if iv.LowClosed && cmpResult < 0 {
				return fptypes.NewBoolean(false), nil
			}
			if !iv.LowClosed && cmpResult <= 0 {
				return fptypes.NewBoolean(false), nil
			}
			// For properly contains: point must not equal the low bound at this precision
			if iv.LowClosed && cmpResult == 0 {
				return fptypes.NewBoolean(false), nil
			}
		}
	}
	// Check high bound
	if iv.High != nil {
		highComps, _ := temporalComponents(iv.High)
		if highComps != nil {
			cmpResult := 0
			for i := 0; i <= cmpPrec; i++ {
				if pointComps[i] < highComps[i] {
					cmpResult = -1
					break
				}
				if pointComps[i] > highComps[i] {
					cmpResult = 1
					break
				}
			}
			if iv.HighClosed && cmpResult > 0 {
				return fptypes.NewBoolean(false), nil
			}
			if !iv.HighClosed && cmpResult >= 0 {
				return fptypes.NewBoolean(false), nil
			}
			if iv.HighClosed && cmpResult == 0 {
				return fptypes.NewBoolean(false), nil
			}
		}
	}
	return fptypes.NewBoolean(true), nil
}

// listContainsValueTriState checks membership with tri-state logic:
// returns (true, false) if found, (false, false) if not found, (false, true) if ambiguous.
func listContainsValueTriState(c fptypes.Collection, val fptypes.Value) (found, ambiguous bool) {
	if val == nil {
		for _, item := range c {
			if item == nil {
				return true, false
			}
		}
		return false, false
	}
	for _, item := range c {
		if item == nil {
			continue
		}
		if item.Equal(val) {
			return true, false
		}
		// Check for ambiguous comparison (different precisions in temporal types)
		if comp, ok := item.(fptypes.Comparable); ok {
			_, err := comp.Compare(val)
			if err != nil && isAmbiguousComparisonErr(err) {
				ambiguous = true
			}
		}
	}
	return false, ambiguous
}

// evalListTimingOp handles timing operations when one or both operands are lists.
func (e *Evaluator) evalListTimingOp(_, _ cqltypes.List, leftIsList, rightIsList bool, left, right fptypes.Value, op ast.TimingOp) (fptypes.Value, error) {
	lc := toCollection(left)
	rc := toCollection(right)

	switch op.Kind {
	case ast.TimingIncludes:
		if leftIsList && !rightIsList {
			// list includes scalar (properly contains / contains)
			if right == nil {
				hasNull := false
				for _, item := range lc {
					if item == nil {
						hasNull = true
						break
					}
				}
				if hasNull {
					if op.Properly {
						return fptypes.NewBoolean(lc.Count() > 1), nil
					}
					return fptypes.NewBoolean(true), nil
				}
				// CQL: for "properly includes null" when null not in list → false
				// For regular "includes null" when null not in list → null
				if op.Properly {
					return fptypes.NewBoolean(false), nil
				}
				return nil, nil
			}
			found, ambig := listContainsValueTriState(lc, right)
			if ambig && !found {
				return nil, nil // ambiguous membership → null
			}
			if op.Properly {
				return fptypes.NewBoolean(found && lc.Count() > 1), nil
			}
			return fptypes.NewBoolean(found), nil
		}
		if op.Properly {
			if rc.Count() >= lc.Count() {
				return fptypes.NewBoolean(false), nil
			}
			for _, item := range rc {
				if !listContainsValue(lc, item) {
					return fptypes.NewBoolean(false), nil
				}
			}
			return fptypes.NewBoolean(true), nil
		}
		for _, item := range rc {
			if !listContainsValue(lc, item) {
				return fptypes.NewBoolean(false), nil
			}
		}
		return fptypes.NewBoolean(true), nil

	case ast.TimingIncludedIn, ast.TimingDuring:
		if rightIsList && !leftIsList {
			// scalar included in list (properly in / in)
			if left == nil {
				hasNull := false
				for _, item := range rc {
					if item == nil {
						hasNull = true
						break
					}
				}
				if hasNull {
					if op.Properly {
						return fptypes.NewBoolean(rc.Count() > 1), nil
					}
					return fptypes.NewBoolean(true), nil
				}
				// CQL: for "null properly included in list" when null not in list → false
				// For regular "null included in list" when null not in list → null
				if op.Properly {
					return fptypes.NewBoolean(false), nil
				}
				return nil, nil
			}
			found, ambig := listContainsValueTriState(rc, left)
			if ambig && !found {
				return nil, nil // ambiguous membership → null
			}
			if op.Properly {
				return fptypes.NewBoolean(found && rc.Count() > 1), nil
			}
			return fptypes.NewBoolean(found), nil
		}
		if op.Properly {
			if lc.Count() >= rc.Count() {
				return fptypes.NewBoolean(false), nil
			}
			for _, item := range lc {
				if !listContainsValue(rc, item) {
					return fptypes.NewBoolean(false), nil
				}
			}
			return fptypes.NewBoolean(true), nil
		}
		for _, item := range lc {
			if !listContainsValue(rc, item) {
				return fptypes.NewBoolean(false), nil
			}
		}
		return fptypes.NewBoolean(true), nil

	case ast.TimingBeforeOrAfter:
		return nil, nil

	case ast.TimingSameAs:
		if lc.Count() != rc.Count() {
			return fptypes.NewBoolean(false), nil
		}
		for _, item := range lc {
			if !listContainsValue(rc, item) {
				return fptypes.NewBoolean(false), nil
			}
		}
		return fptypes.NewBoolean(true), nil

	case ast.TimingOverlaps:
		inter := nullSafeIntersect(lc, rc)
		return fptypes.NewBoolean(len(inter) > 0), nil

	default:
		return nil, nil
	}
}

// evalTemporalComparison handles precision-aware comparison of scalar temporal values.
func (e *Evaluator) evalTemporalComparison(left, right fptypes.Value, op ast.TimingOp) (fptypes.Value, error) {
	precision := op.Precision

	switch op.Kind {
	case ast.TimingSameAs:
		if op.Before {
			// same [precision] or before
			return temporalSameOrBefore(left, right, precision)
		}
		if op.After {
			// same [precision] or after
			return temporalSameOrAfter(left, right, precision)
		}
		// same [precision] as
		return temporalSameAs(left, right, precision)
	case ast.TimingBeforeOrAfter:
		if op.Before {
			return temporalBefore(left, right, precision)
		}
		return temporalAfter(left, right, precision)
	default:
		return nil, nil
	}
}

// temporalComponents extracts year, month, day, hour, minute, second, millisecond
// from a temporal value. Returns the components and the maximum valid precision index.
// Precision indices: 0=year, 1=month, 2=day, 3=hour, 4=minute, 5=second, 6=millisecond
// For DateTime values with timezone info, normalizes to UTC first.
func temporalComponents(v fptypes.Value) (components []int, maxPrec int) {
	switch t := v.(type) {
	case fptypes.DateTime:
		maxPrec := int(t.Precision())
		if maxPrec > 6 {
			maxPrec = 6
		}
		// If the DateTime has a timezone, normalize to UTC for comparison
		if t.HasTZ() {
			utc := t.ToTime().UTC()
			comps := []int{utc.Year(), int(utc.Month()), utc.Day(), utc.Hour(), utc.Minute(), utc.Second(), utc.Nanosecond() / 1e6}
			return comps, maxPrec
		}
		comps := []int{t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Millisecond()}
		return comps, maxPrec
	case fptypes.Date:
		comps := []int{t.Year(), t.Month(), t.Day(), 0, 0, 0, 0}
		maxPrec := int(t.Precision()) // YearPrecision=0, MonthPrecision=1, DayPrecision=2
		return comps, maxPrec
	case fptypes.Time:
		comps := []int{0, 0, 0, t.Hour(), t.Minute(), t.Second(), t.Millisecond()}
		maxPrec := int(t.Precision()) + 3 // HourPrecision=0->3, MinutePrecision=1->4, etc.
		if maxPrec > 6 {
			maxPrec = 6
		}
		return comps, maxPrec
	default:
		return nil, -1
	}
}

// precisionIndex maps a precision string to a component index.
func precisionIndex(precision string) int {
	switch strings.ToLower(precision) {
	case "year":
		return 0
	case "month":
		return 1
	case "day":
		return 2
	case "hour":
		return 3
	case "minute":
		return 4
	case "second":
		return 5
	case "millisecond":
		return 6
	default:
		return -1
	}
}

// temporalCompareAtPrecision compares two temporal values up to the given precision.
// Returns -1, 0, or 1. If the comparison is uncertain (one operand doesn't have
// enough precision), returns (0, false).
func temporalCompareAtPrecision(left, right fptypes.Value, precision string) (int, bool) {
	lComps, lMaxPrec := temporalComponents(left)
	rComps, rMaxPrec := temporalComponents(right)
	if lComps == nil || rComps == nil {
		return 0, false
	}

	targetPrec := precisionIndex(precision)
	if targetPrec < 0 {
		// No precision specified: use minimum of both precisions
		targetPrec = lMaxPrec
		if rMaxPrec < targetPrec {
			targetPrec = rMaxPrec
		}
	}

	// For Date types, start comparison at the appropriate index
	startIdx := 0
	if _, ok := left.(fptypes.Time); ok {
		startIdx = 3
	}

	for i := startIdx; i <= targetPrec; i++ {
		// If either operand doesn't have this component, the result is uncertain
		if i > lMaxPrec || i > rMaxPrec {
			return 0, false
		}
		if lComps[i] < rComps[i] {
			return -1, true
		}
		if lComps[i] > rComps[i] {
			return 1, true
		}
	}
	return 0, true // equal at this precision
}

func temporalSameAs(left, right fptypes.Value, precision string) (fptypes.Value, error) {
	cmp, certain := temporalCompareAtPrecision(left, right, precision)
	if !certain {
		return nil, nil
	}
	return fptypes.NewBoolean(cmp == 0), nil
}

func temporalBefore(left, right fptypes.Value, precision string) (fptypes.Value, error) {
	cmp, certain := temporalCompareAtPrecision(left, right, precision)
	if !certain {
		return nil, nil
	}
	return fptypes.NewBoolean(cmp < 0), nil
}

func temporalAfter(left, right fptypes.Value, precision string) (fptypes.Value, error) {
	cmp, certain := temporalCompareAtPrecision(left, right, precision)
	if !certain {
		return nil, nil
	}
	return fptypes.NewBoolean(cmp > 0), nil
}

func temporalSameOrBefore(left, right fptypes.Value, precision string) (fptypes.Value, error) {
	cmp, certain := temporalCompareAtPrecision(left, right, precision)
	if !certain {
		return nil, nil
	}
	return fptypes.NewBoolean(cmp <= 0), nil
}

func temporalSameOrAfter(left, right fptypes.Value, precision string) (fptypes.Value, error) {
	cmp, certain := temporalCompareAtPrecision(left, right, precision)
	if !certain {
		return nil, nil
	}
	return fptypes.NewBoolean(cmp >= 0), nil
}

func (e *Evaluator) evalSetAggregate(n *ast.SetAggregateExpression) (fptypes.Value, error) {
	operand, err := e.Eval(n.Operand)
	if err != nil {
		return nil, err
	}
	if operand == nil {
		return nil, nil
	}

	// Evaluate per quantity if present
	var perVal fptypes.Value
	if n.Per != nil {
		perVal, err = e.Eval(n.Per)
		if err != nil {
			return nil, err
		}
	}

	switch n.Kind {
	case "expand":
		// Two overloads:
		// 1. expand Interval[a, b] → returns list of point values
		// 2. expand { Interval[a, b] } → returns list of unit intervals
		if iv, ok := operand.(cqltypes.Interval); ok {
			// Single interval overload — returns point values
			points, err := funcs.IntervalExpandPoints(iv, perVal)
			if err != nil {
				return nil, err
			}
			return cqltypes.NewList(points), nil
		}
		// List-of-intervals overload — returns unit intervals
		c := toCollection(operand)
		var result fptypes.Collection
		for _, item := range c {
			if iv, ok := item.(cqltypes.Interval); ok {
				intervals, err := funcs.IntervalExpandIntervals(iv, perVal)
				if err != nil {
					return nil, err
				}
				result = append(result, intervals...)
			}
		}
		return cqltypes.NewList(result), nil
	case "collapse":
		// Collapse overlapping intervals
		c := toCollection(operand)
		var intervals []cqltypes.Interval
		for _, item := range c {
			if iv, ok := item.(cqltypes.Interval); ok {
				// CQL: Interval(null, null) is excluded from collapse
				if iv.Low == nil && iv.High == nil {
					continue
				}
				intervals = append(intervals, iv)
			}
		}
		if len(intervals) == 0 {
			return cqltypes.NewList(nil), nil
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
	// Check if we have Quantity values
	var firstQ fptypes.Quantity
	hasQ := false
	for _, item := range c {
		if item == nil {
			continue
		}
		if q, ok := item.(fptypes.Quantity); ok {
			firstQ = q
			hasQ = true
			break
		}
	}
	if hasQ {
		sum := firstQ.Value()
		unit := firstQ.Unit()
		first := true
		for _, item := range c {
			if item == nil {
				continue
			}
			if q, ok := item.(fptypes.Quantity); ok {
				if first {
					first = false
					continue
				}
				sum = sum.Add(q.Value())
			}
		}
		return fptypes.NewQuantityFromDecimal(sum, unit), nil
	}
	sum := decimal.Zero
	allInt := true
	for _, item := range c {
		if item == nil {
			continue
		}
		if i, ok := item.(fptypes.Integer); ok {
			sum = sum.Add(decimal.NewFromInt(i.Value()))
		} else {
			allInt = false
			d := toDecimal(item)
			sum = sum.Add(d)
		}
	}
	if allInt {
		return fptypes.NewInteger(sum.IntPart()), nil
	}
	return newDecimalFromD(sum), nil
}

func (e *Evaluator) evalAggregateAvg(source fptypes.Value) (fptypes.Value, error) {
	c := toCollection(source)
	if c.Empty() {
		return nil, nil
	}
	sum := decimal.Zero
	count := int64(0)
	for _, item := range c {
		if item == nil {
			continue
		}
		d := toDecimal(item)
		sum = sum.Add(d)
		count++
	}
	if count == 0 {
		return nil, nil
	}
	return newDecimalFromD(sum.Div(decimal.NewFromInt(count))), nil
}

func (e *Evaluator) evalAggregateMinMax(source fptypes.Value, isMin bool) (fptypes.Value, error) {
	c := toCollection(source)
	if c.Empty() {
		return nil, nil
	}
	var result fptypes.Value
	for _, item := range c {
		if item == nil {
			continue
		}
		if result == nil {
			result = item
			continue
		}
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

func (e *Evaluator) evalAggregateProduct(source fptypes.Value) (fptypes.Value, error) {
	c := toCollection(source)
	if c.Empty() {
		return nil, nil
	}
	allInt := true
	product := decimal.NewFromInt(1)
	for _, item := range c {
		if item == nil {
			continue
		}
		if i, ok := item.(fptypes.Integer); ok {
			product = product.Mul(decimal.NewFromInt(i.Value()))
		} else {
			allInt = false
			d := toDecimal(item)
			product = product.Mul(d)
		}
	}
	if allInt {
		return fptypes.NewInteger(product.IntPart()), nil
	}
	return newDecimalFromD(product), nil
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
	if q, ok := source.(fptypes.Quantity); ok {
		v := q.Value()
		if v.IsNegative() {
			v = v.Neg()
		}
		return fptypes.NewQuantityFromDecimal(v, q.Unit()), nil
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

// validateCQLDecimal checks that a decimal literal is within CQL limits:
// max 28 integer digits and max 8 fractional digits.
func validateCQLDecimal(s string) error {
	clean := strings.TrimLeft(s, "+-")
	parts := strings.Split(clean, ".")
	intPart := parts[0]
	if len(intPart) > 28 {
		return fmt.Errorf("decimal overflow: too many integer digits in %s", s)
	}
	if len(parts) == 2 && len(parts[1]) > 8 {
		return fmt.Errorf("decimal overflow: too many fractional digits in %s", s)
	}
	return nil
}

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

// multiplyUnits computes the UCUM product of two units (simplified).
// e.g., "cm" * "cm" → "cm2", "m" * "s" → "m.s"
func multiplyUnits(a, b string) string {
	if a == b {
		return a + "2"
	}
	if a == "1" || a == "" {
		return b
	}
	if b == "1" || b == "" {
		return a
	}
	return a + "." + b
}

// divideUnits computes the UCUM quotient of two units (simplified).
// e.g., "g/cm3" / "g/cm3" → "1"
func divideUnits(a, b string) string {
	if a == b {
		return "1"
	}
	if b == "1" || b == "" {
		return a
	}
	if a == "1" || a == "" {
		return "/" + b
	}
	return a + "/" + b
}

func isTemporalType(v fptypes.Value) bool {
	if v == nil {
		return false
	}
	switch v.Type() {
	case "DateTime", "Date", "Time":
		return true
	}
	return false
}

func isDecimal(v fptypes.Value) bool {
	_, ok := v.(fptypes.Decimal)
	return ok
}

// intervalArithmetic applies a binary arithmetic op to an uncertainty interval and a value.
// If scalarIsLeft is true, the scalar is the left operand (e.g., scalar + Interval).
// When the other operand is also an interval, computes all combinations and returns min/max.
func intervalArithmetic(e *Evaluator, iv cqltypes.Interval, other fptypes.Value, op ast.BinaryOp, scalarIsLeft bool) (fptypes.Value, error) {
	// Collect the bounds of both operands
	leftBounds := []fptypes.Value{iv.Low, iv.High}
	var rightBounds []fptypes.Value
	if iv2, ok := other.(cqltypes.Interval); ok {
		rightBounds = []fptypes.Value{iv2.Low, iv2.High}
	} else {
		rightBounds = []fptypes.Value{other}
	}

	// Compute all combinations
	var results []fptypes.Value
	for _, lb := range leftBounds {
		for _, rb := range rightBounds {
			var r fptypes.Value
			var err error
			if scalarIsLeft {
				r, err = e.evalArithmetic(op, rb, lb)
			} else {
				r, err = e.evalArithmetic(op, lb, rb)
			}
			if err != nil {
				return nil, err
			}
			if r != nil {
				results = append(results, r)
			}
		}
	}
	if len(results) == 0 {
		return nil, nil
	}

	// Find min and max
	minVal := results[0]
	maxVal := results[0]
	for _, r := range results[1:] {
		if rc, ok := r.(fptypes.Comparable); ok {
			if cmp, err := rc.Compare(minVal); err == nil && cmp < 0 {
				minVal = r
			}
			if cmp, err := rc.Compare(maxVal); err == nil && cmp > 0 {
				maxVal = r
			}
		}
	}

	if minVal.Equal(maxVal) {
		return minVal, nil
	}
	return cqltypes.NewInterval(minVal, maxVal, true, true), nil
}

// compareIntervalWithScalar compares an uncertainty interval with a scalar value.
// Returns true if the entire range satisfies the comparison, false if no value
// in the range satisfies it, and null (nil) if uncertain.
func compareIntervalWithScalar(iv cqltypes.Interval, scalar fptypes.Value, op ast.BinaryOp) (fptypes.Value, error) {
	lowC, lowOk := iv.Low.(fptypes.Comparable)
	highC, highOk := iv.High.(fptypes.Comparable)
	if !lowOk || !highOk {
		return nil, nil
	}

	lowCmp, lowErr := lowC.Compare(scalar)
	highCmp, highErr := highC.Compare(scalar)
	if lowErr != nil || highErr != nil {
		return nil, nil
	}

	switch op {
	case ast.OpGreater:
		// true if low > scalar, false if high <= scalar, null otherwise
		if lowCmp > 0 {
			return fptypes.NewBoolean(true), nil
		}
		if highCmp <= 0 {
			return fptypes.NewBoolean(false), nil
		}
		return nil, nil
	case ast.OpGreaterOrEqual:
		if lowCmp >= 0 {
			return fptypes.NewBoolean(true), nil
		}
		if highCmp < 0 {
			return fptypes.NewBoolean(false), nil
		}
		return nil, nil
	case ast.OpLess:
		if highCmp < 0 {
			return fptypes.NewBoolean(true), nil
		}
		if lowCmp >= 0 {
			return fptypes.NewBoolean(false), nil
		}
		return nil, nil
	case ast.OpLessOrEqual:
		if highCmp <= 0 {
			return fptypes.NewBoolean(true), nil
		}
		if lowCmp > 0 {
			return fptypes.NewBoolean(false), nil
		}
		return nil, nil
	}
	return nil, nil
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
			i, err := strconv.ParseInt(val.Value(), 10, 32)
			if err != nil {
				return nil, nil // CQL: invalid string to integer conversion returns null
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
	case "quantity":
		if q, ok := v.(fptypes.Quantity); ok {
			return q, nil
		}
	case "datetime":
		if dt, ok := v.(fptypes.DateTime); ok {
			return dt, nil
		}
		if s, ok := v.(fptypes.String); ok {
			dt, err := fptypes.NewDateTime(s.Value())
			if err != nil {
				return nil, nil // CQL: failed string-to-datetime conversion returns null
			}
			return dt, nil
		}
	case "date":
		if d, ok := v.(fptypes.Date); ok {
			return d, nil
		}
		if s, ok := v.(fptypes.String); ok {
			return fptypes.NewDate(s.Value())
		}
	case "time":
		if t, ok := v.(fptypes.Time); ok {
			return t, nil
		}
		if s, ok := v.(fptypes.String); ok {
			str := strings.TrimPrefix(s.Value(), "T")
			// Strip timezone offset for parsing
			if idx := strings.LastIndexAny(str, "+-"); idx > 0 && strings.Contains(str[idx:], ":") {
				str = str[:idx]
			}
			return fptypes.NewTime(str)
		}
	}
	return nil, fmt.Errorf("cannot convert %s to %s", v.Type(), typeName)
}
