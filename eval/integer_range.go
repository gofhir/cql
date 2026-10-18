package eval

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/shopspring/decimal"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/sema"
	cqltypes "github.com/gofhir/cql/types"
)

// An Integer is 32 bits and a Long 64, and a result past its type is null:
// "if the result of the operation cannot be represented, the result is null".
// The engine holds both in an int64, so the 64-bit edge is caught where the
// operation is computed, and the 32-bit one here, from the static type — a
// value of 2147483648 is a Long's and not an Integer's, and only the plan
// tells the two apart. Without a plan, as for imported ELM, only the 64-bit
// edge is caught.
//
// Both wrapped before: 2147483647 + 1 was 2147483648, still an Integer, and
// a Long product past its edge came back as the low 64 bits of the product.
// The reference engine wraps both; the specification says null.

// integerArithmetic is the system functions whose Integer result is an
// arithmetic one, the result the specification's overflow rule is about.
// Durations, widths and counts are left as they are.
var integerArithmetic = []string{"Abs", "Sum", "Product", "Power", "ToInteger"}

// fixedInteger is the system functions whose result is an Integer whatever
// their operands are, so it is one for certain.
var fixedInteger = []string{
	"Count", "Length", "ToInteger", "Truncate", "Floor", "Ceiling",
	"IndexOf", "PositionOf", "LastPositionOf", "Precision",
}

// namedIn reports whether name is one of names, in any case: a fluent call is
// matched so, `x.abs()` as `Abs(x)`. EqualFold allocates nothing, as lowering
// the name on every call did.
func namedIn(name string, names []string) bool {
	for _, n := range names {
		if strings.EqualFold(name, n) {
			return true
		}
	}
	return false
}

// pastInteger reports whether v is an Integer value past the 32 bits of one,
// computed by node from operands that are Integers for certain.
//
// The static type alone does not say so. Sema gives an expression it cannot
// type no type, and an operation between it and an Integer the Integer's: a
// Long from an included library plus 1 is typed an Integer, and was nulled.
// So the 32-bit edge holds only where every operand is an Integer for
// certain — see certainInteger — and elsewhere the 64-bit one, caught in the
// operation, is all there is. The walk runs only for a value already past 32
// bits, so it costs nothing on the way an answer usually takes.
func (e *Evaluator) pastInteger(node ast.Expression, v fptypes.Value) bool {
	i, ok := v.(fptypes.Integer)
	if !ok || (i.Value() >= math.MinInt32 && i.Value() <= math.MaxInt32) || e.ctx.Plan == nil {
		return false
	}
	return e.certainInteger(node, certainty{})
}

// certainty is what a walk has settled about each definition it met: a
// definition used twice is walked once, and one that refers to itself is not
// certain.
type certainty map[string]certain

type certain uint8

const (
	walking certain = iota + 1
	isCertain
	notCertain
)

// certainInteger reports whether expr is an Integer for certain: typed one, and
// built only from what is — literals, typed nulls and the extents of the type,
// the arithmetic and the aggregates of certain Integers, the system functions
// whose result is an Integer whatever they are given, and the definitions of
// this library made of them. A reference sema cannot follow — an included
// library, an alias, a let, an operand, a sort key's column, an Any — is not,
// since a Long can stand behind it.
func (e *Evaluator) certainInteger(expr ast.Expression, seen certainty) bool {
	if !isIntegerType(e.ctx.Plan.TypeOf(expr)) {
		return false
	}
	switch x := expr.(type) {
	case *ast.Literal, *ast.TypeExtent:
		return true
	case *ast.AsExpression:
		// (null as Integer), or a certain Integer stated as one. Anything else
		// cast to Integer may be a Long, which the cast does not tell apart.
		if lit, ok := x.Operand.(*ast.Literal); ok && lit.ValueType == ast.LiteralNull {
			return true
		}
		return e.certainInteger(x.Operand, seen)
	case *ast.BinaryExpression:
		switch x.Operator {
		case ast.OpAdd, ast.OpSubtract, ast.OpMultiply, ast.OpDiv, ast.OpMod, ast.OpPower:
			return e.certainInteger(x.Left, seen) && e.certainInteger(x.Right, seen)
		}
	case *ast.UnaryExpression:
		switch x.Operator {
		case ast.OpNegate, ast.OpPositive, ast.OpSuccessorOf, ast.OpPredecessorOf:
			return e.certainInteger(x.Operand, seen)
		}
	case *ast.FunctionCall:
		return e.certainCall(x, seen)
	case *ast.IdentifierRef:
		// The name is read where it is written; the definition's own names are
		// read in its library's scope, as it evaluates — not under the aliases
		// and lets of the query it was referenced from.
		stmt := e.definitionOf(x)
		if stmt == nil {
			return false
		}
		root := e.inLibraryScope()
		return root.certainDefinition(x.Name, stmt.Expression, seen, root.certainInteger)
	}
	return false
}

func isIntegerType(t sema.Type) bool {
	n, ok := t.(*sema.Named)
	return ok && n.Model == sema.Integer.Model && n.Name == sema.Integer.Name
}

// definitionOf is the definition of this library a name refers to, or nil:
// for a qualified name, a name an alias or a let stands in front of, and an
// unqualified name in a sort key, which is the column of the same name first.
func (e *Evaluator) definitionOf(n *ast.IdentifierRef) *ast.ExpressionDef {
	if n.Library != "" || e.ctx.Library == nil || e.ctx.InSortKey || e.ctx.boundLocally(n.Name) {
		return nil
	}
	for _, stmt := range e.ctx.Library.Statements {
		if stmt.Name == n.Name {
			return stmt
		}
	}
	return nil
}

// certainDefinition is check of a definition's expression, settled once per
// walk. A definition met again while it is being walked refers to itself.
func (e *Evaluator) certainDefinition(name string, expr ast.Expression, seen certainty,
	check func(ast.Expression, certainty) bool) bool {
	switch seen[name] {
	case walking, notCertain:
		return false
	case isCertain:
		return true
	}
	seen[name] = walking
	if check(expr, seen) {
		seen[name] = isCertain
		return true
	}
	seen[name] = notCertain
	return false
}

// certainCall is certainInteger for a call: a system function whose result is
// an Integer whatever it is given, or one of certain Integers.
func (e *Evaluator) certainCall(n *ast.FunctionCall, seen certainty) bool {
	if n.Library != "" || e.funcs[n.Name] != nil {
		return false
	}
	operands := n.Operands
	if n.Source != nil {
		operands = append([]ast.Expression{n.Source}, operands...)
	}
	switch {
	case namedIn(n.Name, []string{"Abs", "Power"}):
		for _, operand := range operands {
			if !e.certainInteger(operand, seen) {
				return false
			}
		}
		return true
	case namedIn(n.Name, []string{"Sum", "Product", "Max", "Min"}):
		return len(operands) == 1 && e.certainList(operands[0], seen)
	}
	return namedIn(n.Name, fixedInteger)
}

// certainList reports whether expr is a list of certain Integers: a list
// literal of them, or a definition of this library that is one. The key of a
// list definition is apart from an Integer one's, since a walk can ask both.
func (e *Evaluator) certainList(expr ast.Expression, seen certainty) bool {
	switch x := expr.(type) {
	case *ast.ListExpression:
		for _, element := range x.Elements {
			if !e.certainInteger(element, seen) {
				return false
			}
		}
		return true
	case *ast.IdentifierRef:
		stmt := e.definitionOf(x)
		if stmt == nil {
			return false
		}
		root := e.inLibraryScope()
		return root.certainDefinition("list "+x.Name, stmt.Expression, seen, root.certainList)
	}
	return false
}

// withinInteger is v, or null when it is an Integer past the 32 bits of one.
func (e *Evaluator) withinInteger(node ast.Expression, v fptypes.Value, err error) (fptypes.Value, error) {
	if err != nil || !e.pastInteger(node, v) {
		return v, err
	}
	return nil, nil
}

// stepWithinInteger is the successor or predecessor v, or an error when it is
// an Integer past the 32 bits of one.
func (e *Evaluator) stepWithinInteger(node *ast.UnaryExpression, v fptypes.Value, err error) (fptypes.Value, error) {
	if err != nil || !e.pastInteger(node, v) {
		return v, err
	}
	// The operand is the edge the step left from, as the 64-bit error names it.
	if node.Operator == ast.OpPredecessorOf {
		return nil, fmt.Errorf("predecessor of %d: %w", math.MinInt32, cqltypes.ErrPastRange)
	}
	return nil, fmt.Errorf("successor of %d: %w", math.MaxInt32, cqltypes.ErrPastRange)
}

// exactInteger is the result of an exact int64 operation, null where it lies
// past the 64 bits.
func exactInteger(v fptypes.Integer, err error) (fptypes.Value, error) {
	if errors.Is(err, fptypes.ErrIntegerRange) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}

// integerOfExact is a whole number computed exactly, as a Sum or a Product of
// Integers is, or null where it lies past the 64 bits. The total is what has
// to fit, not each step: Sum({maximum Long, 1L, -1L}) is the largest Long.
func integerOfExact(d decimal.Decimal) fptypes.Value {
	if !d.IsInteger() || d.LessThan(minInt64) || d.GreaterThan(maxInt64) {
		return nil
	}
	return fptypes.NewInteger(d.IntPart())
}

var (
	minInt64 = decimal.NewFromInt(math.MinInt64)
	maxInt64 = decimal.NewFromInt(math.MaxInt64)
)
