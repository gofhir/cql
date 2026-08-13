package eval

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/shopspring/decimal"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/funcs"
	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
)

// isAmbiguousComparisonErr returns true if the error is an ambiguous temporal comparison.
// In CQL, ambiguous comparisons should return null, not error.
//
// fptypes reports this as ErrPrecisionMismatch; the string check is kept for the
// wording used before that sentinel existed.
func isAmbiguousComparisonErr(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, fptypes.ErrPrecisionMismatch) ||
		strings.Contains(err.Error(), "ambiguous comparison")
}

// temporalEqualityUnknown reports whether two temporal values agree on everything
// they share while one is specified more precisely, which leaves equality unknown.
//
// Value.Equal answers with a bool and has nowhere to say "unknown", so the question
// is asked here instead: @2017-09-01T00:00:00 = @2017-09-01T00:00:00.000 is null in
// CQL, the same answer the ordering operators give for that pair. Non-temporal
// values are none of this function's business and always compare as they did.
func temporalEqualityUnknown(left, right fptypes.Value) bool {
	if !fptypes.IsTemporal(left) || !fptypes.IsTemporal(right) {
		return false
	}
	_, err := cqltypes.CompareTemporal(left, right)
	return isAmbiguousComparisonErr(err)
}

// calendarUCUMQuantities reports whether two values are quantities measured in a
// calendar year or month on one side and the matching UCUM code on the other,
// which is the pairing CQL declines to decide. See funcs.IsCalendarUCUMDurationPair.
func calendarUCUMQuantities(left, right fptypes.Value) bool {
	lq, lok := left.(fptypes.Quantity)
	rq, rok := right.(fptypes.Quantity)
	return lok && rok && funcs.IsCalendarUCUMDurationPair(lq.Unit(), rq.Unit())
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
//
// The function registries are memoized per library rather than rebuilt here.
// This runs on every user function call, and the official FHIRHelpers declares
// 297 functions: rebuilding those maps per call accounted for 64% of everything
// a measure evaluation allocated.
func NewEvaluator(ctx *Context) *Evaluator {
	return &Evaluator{
		ctx:           ctx,
		funcs:         ctx.functionRegistry(ctx.Library),
		includedFuncs: ctx.includedFunctionRegistry(),
	}
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
		e.ctx.StatementContext = stmt.Context
		val, err := e.Eval(stmt.Expression)
		if err != nil {
			return nil, fmt.Errorf("error evaluating '%s': %w", stmt.Name, err)
		}
		// A private definition is still evaluated — the public ones may be built
		// on it — but it is not part of what the library offers, so it is not
		// part of what the caller is handed.
		e.ctx.Definitions[stmt.Name] = val
		if stmt.AccessLevel != ast.AccessPrivate {
			results[stmt.Name] = val
		}
	}
	return results, nil
}

// EvaluateExpression evaluates a named expression by name.
func (e *Evaluator) EvaluateExpression(name string) (fptypes.Value, error) {
	// Access is decided from the declaration, before the memoized results are
	// consulted. That cache fills as the library evaluates its own definitions
	// and is pre-seeded with declared codes and concepts, so checking it first
	// let a private definition escape as soon as anything had read it — the
	// same hazard evalIncludedDefinition guards against, on the wrong side of
	// the memo.
	if e.ctx.Library != nil {
		for _, stmt := range e.ctx.Library.Statements {
			if stmt.Name == name && stmt.AccessLevel == ast.AccessPrivate {
				return nil, fmt.Errorf("expression %q is private to the library", name)
			}
		}
	}
	// Check if already evaluated
	if val, ok := e.ctx.Definitions[name]; ok {
		return val, nil
	}
	// Find the expression definition
	if e.ctx.Library != nil {
		for _, stmt := range e.ctx.Library.Statements {
			if stmt.Name != name {
				continue
			}
			e.ctx.StatementContext = stmt.Context
			val, err := e.Eval(stmt.Expression)
			if err != nil {
				return nil, err
			}
			e.ctx.Definitions[name] = val
			return val, nil
		}
	}
	return nil, fmt.Errorf("expression '%s' not found", name)
}

// cancelCheckInterval is how many Eval calls pass between two checks of the Go
// context. Checking on every node would put a channel receive on the hottest
// path in the evaluator; checking never is what let a timeout go unnoticed until
// the whole evaluation had finished.
const cancelCheckInterval = 256

// Eval evaluates a single AST expression node and returns a Value.
//
// It is also where the two bounds on an evaluation are enforced — nesting depth
// and cancellation — because it is the one path every node goes through.
func (e *Evaluator) Eval(expr ast.Expression) (fptypes.Value, error) {
	if expr == nil {
		return nil, nil
	}
	if e.ctx.evalTicks != nil {
		*e.ctx.evalTicks++
		if *e.ctx.evalTicks >= cancelCheckInterval {
			*e.ctx.evalTicks = 0
			if err := e.ctx.checkCanceled(); err != nil {
				return nil, err
			}
		}
	}
	if e.ctx.MaxDepth <= 0 || e.ctx.depth == nil {
		return e.eval(expr)
	}
	*e.ctx.depth++
	if *e.ctx.depth > e.ctx.MaxDepth {
		*e.ctx.depth--
		return nil, fmt.Errorf("%w: expression nests deeper than %d", ErrMaxDepthExceeded, e.ctx.MaxDepth)
	}
	result, err := e.eval(expr)
	*e.ctx.depth--
	return result, err
}

func (e *Evaluator) eval(expr ast.Expression) (result fptypes.Value, err error) {
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
		operand, err = e.coerceToSystem(operand)
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
		operand, err = e.coerceToSystem(operand)
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
		// Time carries no finer precision than the millisecond, so anything written past
		// the third fractional digit is dropped rather than rejected: @T23:59:59.10000 is
		// @T23:59:59.100.
		timeVal := n.Value
		if dotIdx := strings.LastIndex(timeVal, "."); dotIdx >= 0 {
			if frac := timeVal[dotIdx+1:]; len(frac) > 3 {
				timeVal = timeVal[:dotIdx+1] + frac[:3]
			}
		}
		t, err := fptypes.NewTime(timeVal)
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
	// Inside a sort key an unqualified identifier names a column of the result,
	// and the column wins over a query alias of the same name: in
	// `({Tuple{A: 1}}) A sort by A` the key is the column, not the whole tuple.
	if e.ctx.InSortKey {
		if v, ok := propertyOf(e.ctx.This, n.Name); ok {
			return v, nil
		}
	}
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
					return nil, wrapUnlessLimit(err, "evaluating referenced expression %q", n.Name)
				}
				e.ctx.Definitions[n.Name] = result
				return result, nil
			}
		}
	}
	// A parameter the caller did not supply falls back to its declared default.
	// The default is an expression — CQL allows `default Interval[@2020-01-01,
	// @2020-12-31]` — so it is evaluated here rather than at parse time, and
	// cached, since ChildScope shares the parameter map with its parent.
	//
	// A parameter declared with neither a default nor a supplied value is null,
	// not an error: declaring an optional parameter is legitimate, and null is
	// what the rest of CQL does with a missing value.
	if e.ctx.Library != nil {
		for _, p := range e.ctx.Library.Parameters {
			if p.Name != n.Name {
				continue
			}
			if p.Default == nil {
				return nil, nil
			}
			result, err := e.Eval(p.Default)
			if err != nil {
				return nil, wrapUnlessLimit(err, "evaluating default for parameter %q", n.Name)
			}
			e.ctx.Parameters[n.Name] = result
			return result, nil
		}
	}
	// A sort key naming no column of *this* element is null, not an error: an
	// optional FHIR element is absent from some resources and present on others,
	// and `[Patient] P sort by birthDate` must not fail on the ones without it.
	// Whether the key names nothing anywhere is decided once by sortKeyIsTypo,
	// before any comparison happens.
	if e.ctx.InSortKey {
		return nil, nil
	}
	// Anywhere else, a name that resolves to nothing is a mistake: answering
	// with the name itself hides it behind a plausible-looking String.
	return nil, fmt.Errorf("unknown identifier %q", n.Name)
}

// evaluationNow renders the evaluation's frozen timestamp as a value, for the
// operators that take an "as of" argument and would otherwise read the clock.
// An age computed against a different instant than Today() answers in the same
// expression is the same inconsistency, one function further out.
func (e *Evaluator) evaluationNow() fptypes.Value {
	v, err := funcs.NowAt(e.ctx.EvaluationTimestamp)
	if err != nil {
		return nil
	}
	return v
}

// isUnconvertedFHIR reports whether a value is still a FHIR object after
// coercion had its chance, which means no conversion was declared for it or its
// type could not be told — an empty object has nothing to infer a type from.
func isUnconvertedFHIR(v fptypes.Value) bool {
	_, ok := v.(*fptypes.ObjectValue)
	return ok
}

// retrieveLimit is what a provider is asked for: one more than the caller will
// accept.
//
// Asking for exactly the maximum makes the refusal below unreachable — a
// provider that honors the limit returns exactly that many rows, len(results)
// never exceeds it, and the population is silently truncated, which is the one
// outcome the limit exists to prevent. One extra row is how the engine can tell
// "there were more" from "that was all".
func retrieveLimit(maxSize int) int {
	if maxSize <= 0 {
		return 0
	}
	return maxSize + 1
}

// wrapUnlessLimit adds context to an evaluation error, except when the error is
// one of the resource limits.
//
// Those two travel up through as many frames as the expression was deep, and
// each frame naming itself is what turned `define A: A` into a 380 KB error
// message once the depth limit made ten thousand levels reachable. The limit
// errors already say everything useful, so they are passed through untouched.
func wrapUnlessLimit(err error, format string, args ...interface{}) error {
	if errors.Is(err, ErrMaxDepthExceeded) || errors.Is(err, ErrMaxRetrieveSizeExceeded) {
		return err
	}
	return fmt.Errorf(fmt.Sprintf(format, args...)+": %w", err)
}

// propertyOf reads a named element from a query result item, reporting whether
// the item has one. The distinction between absent and null is what lets a sort
// key tell a typo from a column that happens to be null.
func propertyOf(v fptypes.Value, name string) (fptypes.Value, bool) {
	switch src := v.(type) {
	case cqltypes.Tuple:
		return src.Get(name)
	case *fptypes.ObjectValue:
		c := src.GetCollection(name)
		switch c.Count() {
		case 0:
			return nil, false
		case 1:
			return c[0], true
		default:
			return cqltypes.NewList(c), true
		}
	}
	return nil, false
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
		if temporalEqualityUnknown(left, right) || calendarUCUMQuantities(left, right) {
			return nil, nil
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
		if temporalEqualityUnknown(left, right) || calendarUCUMQuantities(left, right) {
			return nil, nil
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
		cmp, err := cqltypes.CompareTemporal(left, right)
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
	// A calendar year or month and its UCUM code name the same span even though CQL
	// will not equate them, so equivalence answers on the magnitude alone.
	if calendarUCUMQuantities(left, right) {
		lq := left.(fptypes.Quantity)
		rq := right.(fptypes.Quantity)
		return lq.Value().Equal(rq.Value())
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
	// Round both to the least precision and compare. Rounding, not truncation: at one
	// decimal place 1.55 is 1.6, so 1.5 ~ 1.55 is false. Truncating would discard the
	// digit that decides it and call the two equivalent.
	// minDec is bounded by string decimal place count so it fits in int32.
	aRound := a.Round(int32(minDec)) //nolint:gosec // minDec is derived from decimal place count, always small
	bRound := b.Round(int32(minDec)) //nolint:gosec // minDec is derived from decimal place count, always small
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
	// The first element that does not match settles the answer, so an element that
	// differs and an element that is unknown give different results depending on which
	// comes first: {Id:1,Name:'John'} = {Id:2,Name:null} is false because Id already
	// differs, while {Id:null,Name:'John'} = {Id:1,Name:'James'} is null because Id is
	// unknown before Name is ever reached.
	//
	// Elements are held in a map, so the walk is over sorted keys to make the order
	// deterministic rather than whatever the runtime hands back.
	keys := make([]string, 0, len(a.Elements))
	for k := range a.Elements {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		av := a.Elements[k]
		bv, exists := b.Elements[k]
		if !exists {
			return fptypes.NewBoolean(false), nil
		}
		if av == nil && bv == nil {
			continue // both null → matching
		}
		if av == nil || bv == nil {
			return nil, nil // one null, one not → indeterminate
		}
		if !av.Equal(bv) {
			return fptypes.NewBoolean(false), nil
		}
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
	// An interval on both sides is an inclusion question, not a membership one:
	// CQL reads `X in Y` as IncludedIn when X is an interval and as In when it
	// is a point, a distinction the translator makes from static types. Deciding
	// it from the runtime values is the same call this stage makes elsewhere.
	if leftIv, ok := left.(cqltypes.Interval); ok {
		if rightIv, ok := right.(cqltypes.Interval); ok {
			if op == ast.OpIn {
				return funcs.IntervalIncludedIn(leftIv, rightIv)
			}
			return funcs.IntervalIncludedIn(rightIv, leftIv)
		}
	}
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
	// The most negative Integer has no positive counterpart, so -2147483648 only
	// exists as a whole: evaluating the 2147483648 on its own would overflow before
	// the minus ever applied. Fold the sign into the literal first.
	if n.Operator == ast.OpNegate {
		if lit, ok := n.Operand.(*ast.Literal); ok && lit.ValueType == ast.LiteralInteger {
			if v, err := strconv.ParseInt("-"+lit.Value, 10, 64); err == nil {
				if v < math.MinInt32 || v > math.MaxInt32 {
					return nil, fmt.Errorf("integer overflow: -%s", lit.Value)
				}
				return fptypes.NewInteger(v), nil
			}
		}
	}

	operand, err := e.Eval(n.Operand)
	if err != nil {
		return nil, err
	}
	// The interval operators want an interval, so a FHIR type reaching one is
	// converted the way the model says: `start of encounter.period` is asking
	// about an Interval<DateTime>, not about a FHIR.Period.
	switch n.Operator {
	case ast.OpStartOf, ast.OpEndOf, ast.OpWidthOf, ast.OpPointFrom, ast.OpSuccessorOf, ast.OpPredecessorOf:
		operand, err = e.coerceToSystem(operand)
		if err != nil {
			return nil, err
		}
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
			// An open boundary excludes its own value, so the starting point is
			// the successor of it: `start of Interval(1, 5)` is 2, not 1. A type
			// with no successor cannot name that point, so the boundary stands
			// for itself rather than failing the expression.
			if iv.Low != nil && !iv.LowClosed && hasSuccessor(iv.Low) {
				return e.evalSuccessorPredecessor(ast.OpSuccessorOf, iv.Low)
			}
			return iv.Low, nil
		}
		return nil, nil
	case ast.OpEndOf:
		if iv, ok := operand.(cqltypes.Interval); ok {
			if iv.High != nil && !iv.HighClosed && hasSuccessor(iv.High) {
				return e.evalSuccessorPredecessor(ast.OpPredecessorOf, iv.High)
			}
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

// dateUnit names the unit a Date steps by at its own precision, the Date-side
// counterpart of funcs.TemporalUnit.
func dateUnit(prec fptypes.DatePrecision) string {
	switch prec {
	case fptypes.YearPrecision:
		return "year"
	case fptypes.MonthPrecision:
		return "month"
	default:
		return "day"
	}
}

// hasSuccessor reports whether a value's type defines a successor, which is what
// an open interval boundary needs in order to name its first included point.
// String and the like have none, so an open boundary over them can only stand
// for itself.
func hasSuccessor(v fptypes.Value) bool {
	switch v.(type) {
	case fptypes.Integer, fptypes.Decimal, fptypes.Date, fptypes.DateTime, fptypes.Time, fptypes.Quantity:
		return true
	}
	return false
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
		// The unit comes from the value's own precision, so it is always one fptypes
		// recognizes; the error is threaded through rather than assumed away.
		unit := funcs.TemporalUnit(dt.Precision())
		if op == ast.OpSuccessorOf {
			result, err := dt.AddDuration(1, unit)
			if err != nil {
				return nil, err
			}
			// Check for overflow (year > 9999)
			if result.Year() > 9999 {
				return nil, fmt.Errorf("successor overflow: DateTime exceeds maximum")
			}
			return result, nil
		}
		result, err := dt.SubtractDuration(1, unit)
		if err != nil {
			return nil, err
		}
		// Check for underflow
		if result.Year() < 1 {
			return nil, fmt.Errorf("predecessor underflow: DateTime below minimum")
		}
		return result, nil
	}
	// Date successor/predecessor: add/subtract 1 unit at the date's precision.
	// The step has to follow the precision the way the DateTime branch above
	// does — the successor of @2020-01 is @2020-02, not @2020-01-02 — or the
	// result claims a precision the operand never had.
	if dt, ok := operand.(fptypes.Date); ok {
		unit := dateUnit(dt.Precision())
		if op == ast.OpSuccessorOf {
			return dt.AddDuration(1, unit)
		}
		return dt.SubtractDuration(1, unit)
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

// subtypeChecker is the part of a model that knows its own type hierarchy. It
// is asked for optionally rather than added to the ModelInfo interface, so a
// model built by hand still works without one.
type subtypeChecker interface {
	IsSubtypeOf(concrete, target string) bool
}

// modelSaysSubtype reports whether the model places concrete under target.
func (e *Evaluator) modelSaysSubtype(concrete, target string) bool {
	checker, ok := e.ctx.ModelInfo.(subtypeChecker)
	return ok && checker.IsSubtypeOf(concrete, target)
}

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
	// A resource is also every type it descends from. Comparing names alone
	// made `x is DomainResource` false for a Patient, which is the whole point
	// of asking.
	if e.modelSaysSubtype(operandType, typeName) {
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
	switch t := n.Type.(type) {
	case *ast.NamedType:
		if strings.EqualFold(operand.Type(), t.Name) {
			return operand, nil
		}
	case *ast.ListType:
		// A list keeps its elements whatever the cast names them: the element type is
		// a promise about what the list may hold, not a filter applied to it. Casting
		// to List<Any> is how the conformance suite compares lists of unlike elements
		// at all, so dropping the list here would leave those comparisons null.
		if list, ok := operand.(cqltypes.List); ok {
			return list, nil
		}
	case *ast.IntervalType:
		if iv, ok := operand.(cqltypes.Interval); ok {
			return iv, nil
		}
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
// resolveOverloadByValues picks the overload whose declared operand types best
// fit the arguments actually passed.
//
// Scoring, highest first: the declared type names the value's own type; the
// model places the value's type under the declared one. A candidate that fits
// nothing still beats no candidate at all, which keeps a single same-arity
// overload working whatever it declares.
func (e *Evaluator) resolveOverloadByValues(overloads []*ast.FunctionDef, values []fptypes.Value) *ast.FunctionDef {
	if len(overloads) == 1 {
		return overloads[0]
	}
	var candidates []*ast.FunctionDef
	for _, fd := range overloads {
		if len(fd.Operands) == len(values) {
			candidates = append(candidates, fd)
		}
	}
	switch len(candidates) {
	case 0:
		return overloads[0]
	case 1:
		return candidates[0]
	}

	best, bestScore := candidates[0], -1
	for _, fd := range candidates {
		// Candidates are filtered to exactly this arity above, so the two
		// slices line up; zipping them keeps that visible.
		score := e.scoreOperands(fd.Operands, values)
		if score > bestScore {
			best, bestScore = fd, score
		}
	}
	return best
}

// scoreOperands rates how well a call's arguments fit a candidate's operands.
func (e *Evaluator) scoreOperands(operands []*ast.OperandDef, values []fptypes.Value) int {
	n := len(operands)
	if len(values) < n {
		n = len(values)
	}
	score := 0
	for i := range n {
		score += e.scoreOperand(operands[i], values[i])
	}
	return score
}

// scoreOperand rates how well one argument fits one declared operand type.
func (e *Evaluator) scoreOperand(op *ast.OperandDef, value fptypes.Value) int {
	if op.Type == nil || value == nil {
		return 0
	}
	named, ok := op.Type.(*ast.NamedType)
	if !ok || named.Name == "" {
		return 0
	}
	declared := named.Name
	if i := strings.LastIndex(declared, "."); i >= 0 {
		declared = declared[i+1:]
	}
	actual := value.Type()
	if strings.EqualFold(declared, actual) {
		return 2
	}
	if e.modelSaysSubtype(actual, declared) {
		return 1
	}
	return 0
}

// ensureLibraryLoaded lazily loads an included library by alias using the LibraryLoader.
// It is a no-op if no loader is configured or the library is already loaded.
func (e *Evaluator) ensureLibraryLoaded(alias string) error {
	if e.ctx.Library == nil || e.ctx.LibraryLoader == nil {
		return nil
	}
	for _, inc := range e.ctx.Library.Includes {
		incAlias := inc.Alias
		if incAlias == "" {
			incAlias = inc.Name
		}
		if incAlias != alias {
			continue
		}
		// Recursion guard: detect circular dependencies
		key := inc.Name + "/" + inc.Version
		if e.ctx.loadingLibs == nil {
			e.ctx.loadingLibs = make(map[string]bool)
		}
		if e.ctx.loadingLibs[key] {
			return fmt.Errorf("circular library dependency detected: %s", key)
		}
		e.ctx.loadingLibs[key] = true

		lib, err := e.ctx.LibraryLoader.LoadLibrary(e.ctx.GoCtx, inc.Name, inc.Version)
		if err != nil {
			delete(e.ctx.loadingLibs, key)
			return fmt.Errorf("failed to load library %q v%q: %w", inc.Name, inc.Version, err)
		}
		if lib == nil {
			delete(e.ctx.loadingLibs, key)
			return nil
		}
		e.ctx.IncludedLibraries[alias] = lib
		libFuncs := make(map[string][]*ast.FunctionDef)
		for _, f := range lib.Functions {
			libFuncs[f.Name] = append(libFuncs[f.Name], f)
		}
		e.includedFuncs[alias] = libFuncs
		delete(e.ctx.loadingLibs, key)
		return nil
	}
	return nil
}

func (e *Evaluator) evalFunctionCall(n *ast.FunctionCall) (fptypes.Value, error) {
	// Check for library-qualified call via Source (e.g. FHIRHelpers.ToQuantity(...))
	// Parser produces: FunctionCall{Source: IdentifierRef{Name: "FHIRHelpers"}, Name: "ToQuantity"}
	if n.Source != nil {
		if idRef, ok := n.Source.(*ast.IdentifierRef); ok {
			// Lazy-load the library if not yet resolved
			if _, loaded := e.includedFuncs[idRef.Name]; !loaded {
				if err := e.ensureLibraryLoaded(idRef.Name); err != nil {
					return nil, err
				}
			}
			if libFuncs, ok := e.includedFuncs[idRef.Name]; ok {
				overloads, ok := libFuncs[n.Name]
				if !ok {
					return nil, fmt.Errorf("function '%s' not found in library '%s'", n.Name, idRef.Name)
				}
				return e.callOverload(overloads, n.Operands, e.ctx.IncludedLibraries[idRef.Name])
			}
		}
	}

	// Check for library-qualified call via Library field
	if n.Library != "" {
		// Lazy-load the library if not yet resolved
		if _, loaded := e.includedFuncs[n.Library]; !loaded {
			if err := e.ensureLibraryLoaded(n.Library); err != nil {
				return nil, err
			}
		}
		if libFuncs, ok := e.includedFuncs[n.Library]; ok {
			overloads, ok := libFuncs[n.Name]
			if !ok {
				return nil, fmt.Errorf("function '%s' not found in library '%s'", n.Name, n.Library)
			}
			return e.callOverload(overloads, n.Operands, e.ctx.IncludedLibraries[n.Library])
		}
	}

	// Check if it's a library-defined function
	if overloads, ok := e.funcs[n.Name]; ok {
		return e.callOverload(overloads, n.Operands, nil)
	}
	// Built-in functions handled here
	return e.evalBuiltinFunction(n)
}

// callOverload evaluates the arguments once, picks the overload their types fit,
// and runs it.
//
// The arguments have to be evaluated before the choice, not after: which
// FHIRHelpers.ToInterval is meant depends on whether it was handed a Period, a
// Range or a Quantity, and that is not visible in the syntax. Scoring only
// literals is why the Period overload used to win every time.
func (e *Evaluator) callOverload(overloads []*ast.FunctionDef, args []ast.Expression, owner *ast.Library) (fptypes.Value, error) {
	values := make([]fptypes.Value, 0, len(args))
	for _, arg := range args {
		v, err := e.Eval(arg)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	fd := e.resolveOverloadByValues(overloads, values)
	if fd == nil {
		return nil, fmt.Errorf("no overload of %q accepts %d arguments", overloads[0].Name, len(values))
	}
	return e.runFunction(fd, values, owner)
}

// runFunction binds already-evaluated arguments and runs the body.
func (e *Evaluator) runFunction(fd *ast.FunctionDef, values []fptypes.Value, owner *ast.Library) (fptypes.Value, error) {
	if fd.External {
		return nil, fmt.Errorf("external function '%s' not implemented", fd.Name)
	}
	child := e.ctx.ChildScope()
	if owner != nil {
		// A child of the memoized library scope, not a fresh one: the operand
		// bindings are per call, everything else the scope carries is per library.
		child = e.libraryScope(owner).ChildScope()
	}
	for i, val := range values {
		if i < len(fd.Operands) {
			child.Aliases[fd.Operands[i].Name] = val
		}
	}
	return NewEvaluator(child).Eval(fd.Body)
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
		return funcs.NowAt(e.ctx.EvaluationTimestamp)
	case "today":
		return funcs.TodayAt(e.ctx.EvaluationTimestamp)
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
		return funcs.AgeInYearsAt(bd, e.evaluationNow())
	case "ageinmonths":
		bd := e.getPatientBirthDate()
		return funcs.AgeInMonthsAt(bd, e.evaluationNow())
	case "ageinweeks":
		bd := e.getPatientBirthDate()
		return funcs.AgeInWeeksAt(bd, e.evaluationNow())
	case "ageindays":
		bd := e.getPatientBirthDate()
		return funcs.AgeInDaysAt(bd, e.evaluationNow())
	case "ageinyearsat":
		bd := e.getPatientBirthDate()
		if len(n.Operands) > 0 {
			asOf, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.AgeInYearsAt(bd, asOf)
		}
		return funcs.AgeInYearsAt(bd, e.evaluationNow())
	case "ageinmonthsat":
		bd := e.getPatientBirthDate()
		if len(n.Operands) > 0 {
			asOf, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.AgeInMonthsAt(bd, asOf)
		}
		return funcs.AgeInMonthsAt(bd, e.evaluationNow())
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
			return funcs.CalculateAgeInWeeks(bd, e.evaluationNow())
		}
		return nil, nil
	case "calculateageindays":
		if len(n.Operands) > 0 {
			bd, err := e.Eval(n.Operands[0])
			if err != nil {
				return nil, err
			}
			return funcs.CalculateAgeInDays(bd, e.evaluationNow())
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
		return funcs.TimeOfDayAt(e.ctx.EvaluationTimestamp)

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

	case "slice":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		list, ok := src.(cqltypes.List)
		if !ok {
			return nil, nil
		}
		// An omitted or null bound falls back to the edge of the list, which is what
		// makes Slice(list) the whole list and Slice(list, 1, null) everything after
		// the first element.
		start, end := 0, len(list.Values)
		if len(operands) >= 1 {
			startVal, err := e.Eval(operands[0])
			if err != nil {
				return nil, err
			}
			if startInt, ok := startVal.(fptypes.Integer); ok {
				start = int(startInt.Value())
			}
		}
		if len(operands) >= 2 {
			endVal, err := e.Eval(operands[1])
			if err != nil {
				return nil, err
			}
			if endInt, ok := endVal.(fptypes.Integer); ok {
				end = int(endInt.Value())
			}
		}
		return cqltypes.NewList(funcs.Slice(list.Values, start, end)), nil

	case "sublist":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		list, ok := src.(cqltypes.List)
		if !ok {
			return nil, nil
		}
		if len(operands) < 1 {
			return nil, fmt.Errorf("SubList requires a start index")
		}
		startVal, err := e.Eval(operands[0])
		if err != nil {
			return nil, err
		}
		startInt, ok := startVal.(fptypes.Integer)
		if !ok {
			return nil, nil
		}
		start := int(startInt.Value())
		if start < 0 {
			start = 0
		}
		items := list.Values
		if start >= len(items) {
			return cqltypes.NewList(fptypes.Collection{}), nil
		}
		result := items[start:]
		if len(operands) >= 2 {
			lenVal, err := e.Eval(operands[1])
			if err != nil {
				return nil, err
			}
			if lenInt, ok := lenVal.(fptypes.Integer); ok {
				length := int(lenInt.Value())
				if length >= 0 && length < len(result) {
					result = result[:length]
				}
			}
		}
		cp := make(fptypes.Collection, len(result))
		copy(cp, result)
		return cqltypes.NewList(cp), nil

	case "splitonmatches":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		s, ok := src.(fptypes.String)
		if !ok {
			return nil, nil
		}
		if len(operands) < 1 {
			return nil, fmt.Errorf("SplitOnMatches requires a regex pattern")
		}
		patternVal, err := e.Eval(operands[0])
		if err != nil {
			return nil, err
		}
		pattern, ok := patternVal.(fptypes.String)
		if !ok {
			return nil, nil
		}
		re, err := regexp.Compile(pattern.Value())
		if err != nil {
			return nil, fmt.Errorf("SplitOnMatches: invalid regex %q: %w", pattern.Value(), err)
		}
		parts := re.Split(s.Value(), -1)
		vals := make(fptypes.Collection, len(parts))
		for i, p := range parts {
			vals[i] = fptypes.NewString(p)
		}
		return cqltypes.NewList(vals), nil

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

	case "subsumes":
		left, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) < 1 {
			return nil, fmt.Errorf("Subsumes requires two code arguments")
		}
		right, err := e.Eval(operands[0])
		if err != nil {
			return nil, err
		}
		return e.evalSubsumes(left, right)

	case "subsumedby":
		left, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) < 1 {
			return nil, fmt.Errorf("SubsumedBy requires two code arguments")
		}
		right, err := e.Eval(operands[0])
		if err != nil {
			return nil, err
		}
		return e.evalSubsumedBy(left, right)

	case "expandvalueset":
		vsURL := ""
		// Fluent: source is a string URL
		if source != nil {
			if s, ok := source.(fptypes.String); ok {
				vsURL = s.Value()
			}
		}
		// Standalone: first operand is a ValueSet reference or URL string
		if vsURL == "" && len(operands) > 0 {
			if idRef, ok := operands[0].(*ast.IdentifierRef); ok {
				vsURL, _ = e.ctx.ResolveValueSetURL(idRef.Name)
			}
			if vsURL == "" {
				val, evalErr := e.Eval(operands[0])
				if evalErr != nil {
					return nil, evalErr
				}
				if s, ok := val.(fptypes.String); ok {
					vsURL = s.Value()
				}
			}
		}
		return e.evalExpandValueSet(vsURL)

	case "convertquantity":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) < 1 {
			return nil, fmt.Errorf("ConvertQuantity requires a target unit")
		}
		unitVal, err := e.Eval(operands[0])
		if err != nil {
			return nil, err
		}
		unitStr, ok := unitVal.(fptypes.String)
		if !ok {
			return nil, nil
		}
		return e.evalConvertQuantity(src, unitStr.Value())

	case "canconvertquantity":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if len(operands) < 1 {
			return nil, fmt.Errorf("CanConvertQuantity requires a target unit")
		}
		unitVal, err := e.Eval(operands[0])
		if err != nil {
			return nil, err
		}
		unitStr, ok := unitVal.(fptypes.String)
		if !ok {
			return nil, nil
		}
		return e.evalCanConvertQuantity(src, unitStr.Value())

	// tolong — converts Integer or String to Long (int64).
	// On null, returns null. On invalid input, returns null.
	case "tolong":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		switch v := src.(type) {
		case fptypes.Integer:
			return v, nil // Already int64 internally
		case fptypes.String:
			n, parseErr := strconv.ParseInt(v.Value(), 10, 64)
			if parseErr != nil {
				return nil, nil
			}
			return fptypes.NewInteger(n), nil
		default:
			return nil, nil
		}

	// children — returns all child values of the input object.
	// On null, returns null. On non-object, returns empty list.
	case "children":
		src, err := resolveSource()
		if err != nil {
			return nil, err
		}
		if src == nil {
			return nil, nil
		}
		if obj, ok := src.(*fptypes.ObjectValue); ok {
			return cqltypes.NewList(obj.Children()), nil
		}
		return cqltypes.NewList(fptypes.Collection{}), nil

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

// evalIncludedDefinition resolves `alias.name` against an included library's
// definitions, codes, concepts and value sets. The third return reports whether
// alias named an include at all, so an ordinary member access on a value keeps
// its existing path.
func (e *Evaluator) evalIncludedDefinition(alias, name string) (fptypes.Value, bool, error) {
	if _, loaded := e.ctx.IncludedLibraries[alias]; !loaded {
		// The alias may name an include that is only resolved on demand.
		if !e.ctx.hasInclude(alias) {
			return nil, false, nil
		}
		if err := e.ensureLibraryLoaded(alias); err != nil {
			return nil, true, err
		}
	}
	lib, ok := e.ctx.IncludedLibraries[alias]
	if !ok || lib == nil {
		return nil, false, nil
	}
	scope := e.libraryScope(lib)
	// Codes, concepts and anything already evaluated.
	// Access is decided from the declaration, before the memoized results are
	// consulted: the cache fills as the library evaluates its own definitions, so
	// checking it first let a private define escape as soon as anything inside
	// the library had read it.
	for _, stmt := range lib.Statements {
		if stmt.Name == name && stmt.AccessLevel == ast.AccessPrivate {
			return nil, true, fmt.Errorf("%q is private to library %q", name, alias)
		}
	}
	if v, ok := scope.Definitions[name]; ok {
		return v, true, nil
	}
	if _, ok := scope.ValueSets[name]; ok {
		return cqltypes.NewValueSetRef(name, alias), true, nil
	}
	for _, stmt := range lib.Statements {
		if stmt.Name != name {
			continue
		}
		v, err := NewEvaluator(scope).Eval(stmt.Expression)
		if err != nil {
			return nil, true, fmt.Errorf("evaluating %q of library %q: %w", name, alias, err)
		}
		// Memoize: a CQL definition is evaluated once, and referencing one three
		// times must not issue three retrieves.
		scope.Definitions[name] = v
		return v, true, nil
	}
	return nil, true, fmt.Errorf("library %q has no definition named %q", alias, name)
}

// libraryScope returns the memoized scope for a library, building it on first
// use. Sharing it makes a definition of an included library evaluate once per
// evaluation, and keeps the six maps a library scope carries from being rebuilt
// on every call into that library.
func (e *Evaluator) libraryScope(lib *ast.Library) *Context {
	if lib == nil {
		return e.ctx
	}
	if e.ctx.libraryScopes == nil {
		return e.ctx.LibraryScope(lib)
	}
	if scope, ok := e.ctx.libraryScopes[lib]; ok {
		return scope
	}
	scope := e.ctx.LibraryScope(lib)
	e.ctx.libraryScopes[lib] = scope
	return scope
}

func (e *Evaluator) evalMemberAccess(n *ast.MemberAccess) (fptypes.Value, error) {
	// `Alias.Name` where Alias is an include names a definition of that library,
	// not a property of a value. Only functions were reachable across an include
	// before, so a library's definitions were invisible to the libraries that
	// included it.
	//
	// Anything bound in scope wins, and is checked first: a query alias or a
	// function operand may share a name with an include, and `({…}) H return H.x`
	// is about the row, not about the library that happens to be called H.
	if idRef, ok := n.Source.(*ast.IdentifierRef); ok {
		if _, bound := e.ctx.ResolveIdentifier(idRef.Name); !bound {
			if v, ok, err := e.evalIncludedDefinition(idRef.Name, n.Member); ok {
				return v, err
			}
		}
	}
	source, err := e.Eval(n.Source)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, nil
	}
	// `.value` on a system primitive is the primitive itself. The official
	// ModelInfo models FHIR.string as an object with a value element, so the
	// official FHIRHelpers is written as coding.code.value — while the evaluator
	// navigates raw JSON, where coding.code already is the scalar. This bridges
	// the two without wrapping every primitive.
	//
	// The rule is deliberately narrow. If `.value` on any value returned that
	// value, a mistyped someString.value would stop failing and quietly answer
	// the string, turning a typo into a silence.
	if n.Member == "value" && isSystemPrimitive(source) {
		return source, nil
	}
	// The clinical types carry named elements too. Materializing Code and
	// Concept as real values rather than labeled Tuples took their member
	// access away with them, and `ToConcept(x).codes` is exactly what the
	// official FHIRHelpers is written against.
	switch src := source.(type) {
	case cqltypes.Code:
		switch n.Member {
		case "code":
			return optionalString(src.Code), nil
		case "system":
			return optionalString(src.System), nil
		case "display":
			return optionalString(src.Display), nil
		case "version":
			return optionalString(src.Version), nil
		}
	case cqltypes.Concept:
		switch n.Member {
		case "codes":
			codes := make(fptypes.Collection, 0, len(src.Codes))
			for _, c := range src.Codes {
				codes = append(codes, c)
			}
			return cqltypes.NewList(codes), nil
		case "display":
			return optionalString(src.Display), nil
		}
	}
	// A System.Quantity has value and unit elements. FHIRHelpers.ToQuantity
	// returns one, and `FHIRHelpers.ToQuantity(o.valueQuantity).value` is a
	// common enough shape that it is worth naming: without this the accessor
	// answered null on a perfectly good Quantity.
	if q, ok := source.(fptypes.Quantity); ok {
		switch n.Member {
		case "value":
			return newDecimalFromD(q.Value()), nil
		case "unit":
			return optionalString(q.Unit()), nil
		}
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
						// FHIR names a choice element for its type with the
						// first letter capitalized, and the model spells the
						// primitive types in lower case: FHIR.dateTime is the
						// type, effectiveDateTime is the field.
						concreteKey := n.Member + capitalizeFirst(suffix)
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
	// `[Condition: "Diabetes"]` says which codes to filter by but not which
	// element carries them. That is the model's primary code path, and it was
	// never consulted: every retrieve reached the provider with an empty path,
	// leaving it to guess which element the codes referred to.
	codePath := n.CodePath
	if codePath == "" && codes != nil && e.ctx.ModelInfo != nil {
		codePath = e.ctx.ModelInfo.PrimaryCodePath(resourceType)
	}
	req := RetrieveRequest{
		ResourceType:   resourceType,
		CodePath:       codePath,
		CodeComparator: n.CodeComparator,
		Codes:          codes,
		DateRange:      dateRange,
		Limit:          retrieveLimit(e.ctx.MaxRetrieveSize),
	}
	e.ctx.applyRetrieveContext(&req)
	results, err := e.ctx.DataProvider.Retrieve(e.ctx.GoCtx, req)
	if observer, ok := e.ctx.TraceListener.(RetrieveObserver); ok {
		observer.OnRetrieve(req, len(results), err)
	}
	if err != nil {
		return nil, fmt.Errorf("retrieve [%s] failed: %w", resourceType, err)
	}
	// Refuse rather than truncate. A quality measure computed over a silently
	// shortened population is wrong without saying so, and a wrong denominator
	// is worse than no answer.
	if e.ctx.MaxRetrieveSize > 0 && len(results) > e.ctx.MaxRetrieveSize {
		return nil, fmt.Errorf("%w: retrieve [%s] returned %d resources, limit is %d",
			ErrMaxRetrieveSizeExceeded, resourceType, len(results), e.ctx.MaxRetrieveSize)
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

	// Build the cartesian product of all sources as a list of alias maps. This
	// loop evaluates nothing, so the periodic check in Eval never runs here and
	// a product large enough to matter has to be interrupted on its own.
	combos := []queryCombo{{aliases: make(map[string]fptypes.Value)}}
	for idx, src := range n.Sources {
		var next []queryCombo
		for i, c := range combos {
			if i%cancelCheckInterval == 0 {
				if err := e.ctx.checkCanceled(); err != nil {
					return nil, err
				}
			}
			for _, item := range allSources[idx] {
				newAliases := make(map[string]fptypes.Value, len(c.aliases)+1)
				maps.Copy(newAliases, c.aliases)
				newAliases[src.Alias] = item
				next = append(next, queryCombo{aliases: newAliases})
			}
		}
		combos = next
	}

	// Process each combination through filters and return/aggregate.
	//
	// One scope serves the whole loop, rebound per row rather than allocated per
	// row: a Context carries thirty-odd fields and two maps, and over a thousand
	// resources that was the single largest source of garbage in a measure. The
	// scope does not outlive the query — nothing retains it, and values produced
	// from it are plain data — so rebinding is equivalent to the enter/exit scope
	// a stack-based resolver would do.
	var results fptypes.Collection
	child := e.ctx.ChildScope()
	childEval := e.withContext(child)
	for i, c := range combos {
		if err := e.ctx.checkCanceled(); err != nil {
			return nil, err
		}
		clear(child.Aliases)
		clear(child.LetBindings)
		maps.Copy(child.Aliases, c.aliases)
		// Set This and Index to the first source's item
		child.This = c.aliases[n.Sources[0].Alias]
		child.Index = i
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
		for _, byItem := range n.Sort.ByItems {
			if e.sortKeyIsTypo(byItem.Expression, n.Sources, results) {
				return nil, fmt.Errorf("unknown sort key %q: not a column of the query result",
					byItem.Expression.(*ast.IdentifierRef).Name)
			}
		}
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

// sortKeyIsTypo reports whether a bare identifier sort key names nothing at all:
// not a column of any element of the result, not a query alias, and nothing in
// scope. It is deliberately asked once per query rather than once per element,
// because a column missing from *some* elements is an optional element and must
// sort as null, while one missing from every element and resolving nowhere is a
// mistake — and answering it with its own name as a String would give every
// element the same key, leaving the query silently unsorted.
func (e *Evaluator) sortKeyIsTypo(expr ast.Expression, sources []*ast.AliasedSource, results fptypes.Collection) bool {
	id, ok := expr.(*ast.IdentifierRef)
	if !ok {
		return false
	}
	for _, src := range sources {
		if src.Alias == id.Name {
			return false
		}
	}
	for _, item := range results {
		if _, ok := propertyOf(item, id.Name); ok {
			return false
		}
	}
	if _, ok := e.ctx.ResolveIdentifier(id.Name); ok {
		return false
	}
	if e.ctx.Library != nil {
		for _, stmt := range e.ctx.Library.Statements {
			if stmt.Name == id.Name {
				return false
			}
		}
		for _, p := range e.ctx.Library.Parameters {
			if p.Name == id.Name {
				return false
			}
		}
	}
	return true
}

// sortKeyValue reduces a sort key to something orderable. A repeating element
// yields a list, which has no ordering: a single entry stands for itself, and
// anything longer sorts as null rather than failing the whole query on whichever
// pairs the sort happened to compare.
func sortKeyValue(v fptypes.Value) fptypes.Value {
	list, ok := v.(cqltypes.List)
	if !ok {
		return v
	}
	if list.Values.Count() == 1 {
		return list.Values[0]
	}
	return nil
}

// compareSortKeys evaluates a sort expression against two items and returns their comparison.
func (e *Evaluator) compareSortKeys(alias string, a, b fptypes.Value, expr ast.Expression) (int, error) {
	scopeA := e.ctx.ChildScope()
	scopeA.Aliases[alias] = a
	scopeA.This = a
	scopeA.InSortKey = true
	keyA, err := e.withContext(scopeA).Eval(expr)
	if err != nil {
		return 0, err
	}

	scopeB := e.ctx.ChildScope()
	scopeB.Aliases[alias] = b
	scopeB.This = b
	scopeB.InSortKey = true
	keyB, err := e.withContext(scopeB).Eval(expr)
	if err != nil {
		return 0, err
	}

	return compareValues(sortKeyValue(keyA), sortKeyValue(keyB))
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

// optionalString answers null for an element the value does not carry.
//
// These structs hold plain strings, so absent and empty are the same thing to
// them; answering with the empty string would make `code.display is null` false
// for a Code that never had a display, which is not what CQL says about a
// missing element.
func optionalString(s string) fptypes.Value {
	if s == "" {
		return nil
	}
	return fptypes.NewString(s)
}

// capitalizeFirst upper-cases the first rune, leaving the rest alone. It builds
// the concrete name of a choice element from its type: FHIR spells the type
// dateTime and the element effectiveDateTime.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// isSystemPrimitive reports whether a value is one of the CQL system primitives,
// the only types for which `.value` is the identity. Objects, tuples and the
// clinical types are excluded: on those, `.value` names a real element and has
// to keep resolving as one.
func isSystemPrimitive(v fptypes.Value) bool {
	switch v.(type) {
	case fptypes.Boolean, fptypes.String, fptypes.Integer, fptypes.Decimal,
		fptypes.Date, fptypes.DateTime, fptypes.Time:
		return true
	}
	return false
}

// stringElem reads a Tuple element as a plain string.
//
// A collection is not a string and must not be rendered as one: a repeated FHIR
// element feeding a Code selector would otherwise produce a code literally
// spelled "{a, b}", which no terminology server will ever match — a silently
// false membership check rather than a visible mistake. A one-element
// collection is its element, since that is what a singleton repeat means.
func stringElem(elements map[string]fptypes.Value, name string) string {
	v, ok := elements[name]
	if !ok || v == nil {
		return ""
	}
	if list, ok := v.(cqltypes.List); ok {
		if list.Values.Count() != 1 {
			return ""
		}
		v = list.Values[0]
		if v == nil {
			return ""
		}
	}
	if s, ok := v.(fptypes.String); ok {
		return s.Value()
	}
	if _, isList := v.(cqltypes.List); isList {
		return ""
	}
	return v.String()
}

// buildCode materializes a System.Code instance selector.
func buildCode(elements map[string]fptypes.Value) cqltypes.Code {
	return cqltypes.Code{
		Code:    stringElem(elements, "code"),
		System:  stringElem(elements, "system"),
		Display: stringElem(elements, "display"),
		Version: stringElem(elements, "version"),
	}
}

// buildConcept materializes a System.Concept instance selector, taking the codes
// it was given however they arrived: a list, a single code, or nothing.
func buildConcept(elements map[string]fptypes.Value) cqltypes.Concept {
	var codes []cqltypes.Code
	switch v := elements["codes"].(type) {
	case cqltypes.List:
		for _, item := range v.Values {
			if c, ok := item.(cqltypes.Code); ok {
				codes = append(codes, c)
			}
		}
	case cqltypes.Code:
		codes = append(codes, v)
	}
	return cqltypes.NewConcept(codes, stringElem(elements, "display"))
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

	// The clinical types have to be materialized, not labeled. A Tuple carrying
	// a TypeOverride answers "Code" to Type() but is still a Tuple underneath, so
	// extractCodeComponents does not recognize it and `x in "ValueSet"` answers
	// false for a Code that FHIRHelpers.ToCode just built.
	if n.Type != nil {
		switch {
		case strings.EqualFold(n.Type.Name, "Quantity"):
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
		case strings.EqualFold(n.Type.Name, "Code"):
			return buildCode(elements), nil
		case strings.EqualFold(n.Type.Name, "Concept"):
			return buildConcept(elements), nil
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
	// `code in "Diabetes"` names a value set, not a list to search. The
	// terminology path existed but nothing routed to it from here, so the
	// membership was evaluated against the name as a plain string and answered
	// false for every code.
	// The same conversion the timing operators need: `encounter.period in MP`
	// is the more common spelling of `during`, and was failing where `during`
	// worked.
	if left, err = e.coerceToSystem(left); err != nil {
		return nil, err
	}
	if n.Operator == "in" {
		if ref, ok := n.Right.(*ast.IdentifierRef); ok {
			if _, found := e.ctx.ResolveValueSetURL(ref.Name); found {
				return e.evalInValueSet(left, ref)
			}
			if e.ctx.Library != nil {
				for _, cs := range e.ctx.Library.CodeSystems {
					if cs.Name == ref.Name {
						return e.evalInCodeSystem(left, ref)
					}
				}
			}
		}
		// `x in H."Diabetes"`, where the value set belongs to an included library.
		if ma, ok := n.Right.(*ast.MemberAccess); ok {
			if idRef, ok := ma.Source.(*ast.IdentifierRef); ok {
				if url, ok := e.includedValueSetURL(idRef.Name, ma.Member); ok {
					return e.inValueSetURL(left, url)
				}
			}
		}
	}
	right, err := e.Eval(n.Right)
	if err != nil {
		return nil, err
	}
	// A list is a membership question, not a conversion one — only convert the
	// operand that is not the collection being searched.
	if _, isList := right.(cqltypes.List); !isList {
		if right, err = e.coerceToSystem(right); err != nil {
			return nil, err
		}
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
	// A timing operator works on points and intervals, so a FHIR type reaching
	// one has to be converted first: `encounter.period during "MP"` hands it a
	// FHIR.Period where it needs an Interval<DateTime>, and the model says which
	// function makes that.
	if left, err = e.coerceToSystem(left); err != nil {
		return nil, err
	}
	if right, err = e.coerceToSystem(right); err != nil {
		return nil, err
	}
	// For includes/includedIn/contains/in with lists, don't short-circuit on nil
	// Handle list-based operations first (before null propagation)
	leftList, leftIsList := left.(cqltypes.List)
	rightList, rightIsList := right.(cqltypes.List)
	// A FHIR value the model cannot convert is not something the point and
	// interval operators can answer about, and answering null is what CQL does
	// with a value it cannot interpret — `Interval[null, null] during MP` is
	// null too. Letting it through produced "cannot compare Object", which is
	// how an Encounter carrying an empty `period: {}` failed a whole measure.
	//
	// Membership in a list is a different question, and one a FHIR resource is
	// a perfectly good subject for: `encounter included in [Encounter]` asks
	// whether the list contains it, not how it converts.
	if !leftIsList && !rightIsList && (isUnconvertedFHIR(left) || isUnconvertedFHIR(right)) {
		return nil, nil
	}
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
				// A null collection holds nothing, so it properly includes nothing. That
				// is a different question from the unbounded interval above, and it has a
				// definite answer rather than an unknown one.
				return fptypes.NewBoolean(false), nil
			}
		case ast.TimingIncludedIn, ast.TimingDuring:
			if n.Operator.Properly && right == nil && left != nil {
				// X properly included in null → X is a proper subset of the universal interval → true
				if _, leftIsIv := left.(cqltypes.Interval); leftIsIv {
					return fptypes.NewBoolean(true), nil
				}
				return fptypes.NewBoolean(false), nil
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
		if _, err := cqltypes.CompareTemporal(item, val); isAmbiguousComparisonErr(err) {
			ambiguous = true
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
			if op.Properly {
				// Proper containment asks for a member, and a value that only might match
				// is not one. @T15:59:59 is not a member of a list of millisecond-precision
				// times, so the answer is false rather than unknown.
				return fptypes.NewBoolean(found && lc.Count() > 1), nil
			}
			if ambig && !found {
				return nil, nil // ambiguous membership → null
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
			if op.Properly {
				// See the mirror of this in the includes branch above.
				return fptypes.NewBoolean(found && rc.Count() > 1), nil
			}
			if ambig && !found {
				return nil, nil // ambiguous membership → null
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
//
// An explicit precision states how far to look, so agreeing that far settles the
// question: `@T15:59:59 same second as @T15:59:59.999` is true. Without one the
// comparison runs to the precision the operands carry, and agreeing on everything
// they share while one is specified more finely leaves the result unknown, which
// is the rule types.CompareTemporal applies to the ordering operators.
func temporalCompareAtPrecision(left, right fptypes.Value, precision string) (int, bool) {
	lComps, lMaxPrec := temporalComponents(left)
	rComps, rMaxPrec := temporalComponents(right)
	if lComps == nil || rComps == nil {
		return 0, false
	}

	targetPrec := precisionIndex(precision)
	explicitPrecision := targetPrec >= 0
	if !explicitPrecision {
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
	if !explicitPrecision && lMaxPrec != rMaxPrec {
		return 0, false // equal as far as both are specified, but not equally specified
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
