package sema

import (
	"github.com/gofhir/cql/ast"
)

// inferBinary types an infix operator.
func (c *checker) inferBinary(e *ast.BinaryExpression) Type {
	switch e.Operator {
	case ast.OpAnd, ast.OpOr, ast.OpXor, ast.OpImplies:
		c.expect(e.Left, Boolean)
		c.expect(e.Right, Boolean)
		return Boolean

	case ast.OpEqual, ast.OpNotEqual, ast.OpEquivalent, ast.OpNotEquivalent:
		c.inferComparableOperands(e)
		return Boolean

	case ast.OpLess, ast.OpLessOrEqual, ast.OpGreater, ast.OpGreaterOrEqual:
		c.inferComparableOperands(e)
		return Boolean

	case ast.OpConcatenate:
		c.expectStringish(e.Left)
		c.expectStringish(e.Right)
		return String

	case ast.OpIn, ast.OpContains:
		return c.inferBinaryMembership(e)

	case ast.OpUnion, ast.OpIntersect, ast.OpExcept:
		return c.inferSetOperation(e)

	case ast.OpDivide:
		// `/` is always decimal division: 3 / 2 is 1.5, and `div` is what
		// integer division is spelled.
		left, right := c.arithmeticOperand(e.Left), c.arithmeticOperand(e.Right)
		result := Common(left, right, c.model)
		if Equal(result, Integer) || Equal(result, Long) {
			return Decimal
		}
		return c.arithmeticResult(e, result)

	case ast.OpDiv, ast.OpMod:
		left, right := c.arithmeticOperand(e.Left), c.arithmeticOperand(e.Right)
		return c.arithmeticResult(e, Common(left, right, c.model))

	case ast.OpAdd, ast.OpSubtract:
		return c.inferAdditive(e)

	case ast.OpMultiply, ast.OpPower:
		left, right := c.arithmeticOperand(e.Left), c.arithmeticOperand(e.Right)
		return c.arithmeticResult(e, Common(left, right, c.model))
	}
	c.infer(e.Left)
	c.infer(e.Right)
	return Unknown
}

// inferAdditive types `+` and `-`, which mean three different things depending
// on their operands: arithmetic, string concatenation, and moving a point in
// time by a quantity.
func (c *checker) inferAdditive(e *ast.BinaryExpression) Type {
	left := c.arithmeticOperand(e.Left)
	right := c.arithmeticOperand(e.Right)

	if isTemporal(left) {
		// DateTime + 1 year is a DateTime; DateTime - DateTime is not
		// subtraction in CQL but `difference between`, so only the quantity
		// form is accepted here.
		if Equal(right, Quantity) || IsUnknown(right) {
			return left
		}
		if e.Operator == ast.OpSubtract && isTemporal(right) {
			c.reportf(e, SeverityError,
				"two %ss cannot be subtracted; use `difference in <precision> between`", left)
			return Unknown
		}
	}
	if e.Operator == ast.OpAdd && (Equal(left, String) || Equal(right, String)) {
		c.expectStringish(e.Left)
		c.expectStringish(e.Right)
		return String
	}
	if _, ok := left.(*List); ok && e.Operator == ast.OpAdd {
		// List + List is not CQL, but List + element is how some libraries
		// spell append; leave the type as the list rather than inventing one.
		return left
	}
	return c.arithmeticResult(e, Common(left, right, c.model))
}

// arithmeticOperand types one side of an arithmetic operator, converting a
// model type into the system type it stands for.
func (c *checker) arithmeticOperand(expr ast.Expression) Type {
	t := c.infer(expr)
	named, ok := t.(*Named)
	if !ok || named.Model == "System" || c.model == nil {
		return t
	}
	if converted, ok := c.convertToSystem(expr, named); ok {
		return converted
	}
	return t
}

// arithmeticResult checks that what the operands came to is something
// arithmetic can be done on.
func (c *checker) arithmeticResult(e *ast.BinaryExpression, result Type) Type {
	if IsUnknown(result) || isNumeric(result) || Equal(result, Any) {
		return result
	}
	c.reportf(e, SeverityError, "%s is not a number", result)
	return Unknown
}

// inferComparableOperands types both sides of a comparison and says so when
// they are types no value could ever satisfy both of.
//
// Equality between unrelated types is reported as a warning rather than an
// error: it is legal CQL, it evaluates to false, and it is nearly always a
// mistake — comparing a FHIR.CodeableConcept to a String is how a missing
// FHIRHelpers conversion shows up.
func (c *checker) inferComparableOperands(e *ast.BinaryExpression) {
	left := c.arithmeticOperand(e.Left)
	right := c.arithmeticOperand(e.Right)
	if IsUnknown(left) || IsUnknown(right) || Equal(left, Any) || Equal(right, Any) {
		return
	}
	if _, ok := Convertible(left, right, c.model); ok {
		return
	}
	if _, ok := Convertible(right, left, c.model); ok {
		return
	}
	c.reportf(e, SeverityWarning, "%s and %s can never be equal", left, right)
}

// expectStringish accepts anything a concatenation can render, which is a
// String or a model type that converts to one.
func (c *checker) expectStringish(expr ast.Expression) {
	t := c.arithmeticOperand(expr)
	if IsUnknown(t) || Equal(t, String) || Equal(t, Any) {
		return
	}
	if _, ok := Convertible(t, String, c.model); ok {
		return
	}
	c.reportf(expr, SeverityError, "expected a String, got %s", t)
}

// inferBinaryMembership types `x in y` and `y contains x` in their plain form,
// where the grammar produced a binary node rather than the precision-carrying
// membership node.
func (c *checker) inferBinaryMembership(e *ast.BinaryExpression) Type {
	container := e.Right
	if e.Operator == ast.OpContains {
		container = e.Left
	}
	c.infer(e.Left)
	c.infer(e.Right)
	t := c.types[container]
	switch t.(type) {
	case *List, *Interval, unknownType, nil:
		return Boolean
	}
	if n, ok := t.(*Named); ok {
		switch n.Name {
		case "ValueSet", "CodeSystem", "Vocabulary", "Concept", "Any":
			return Boolean
		}
		if _, ok := c.convertToInterval(container, t); ok {
			return Boolean
		}
	}
	c.reportf(e, SeverityError, "%s is not something a value can be in", t)
	return Boolean
}

// inferSetOperation types union, intersect and except, which work on lists and
// on intervals, and answer whichever they were given.
func (c *checker) inferSetOperation(e *ast.BinaryExpression) Type {
	left := c.infer(e.Left)
	right := c.infer(e.Right)

	// A `null` operand is neither a list nor an interval and is legal in both:
	// `{ 1, 4 } except null` is a list. Any says as little here as Unknown
	// does, so it takes the other side's shape.
	if IsUnknown(left) || isAny(left) {
		return right
	}
	if IsUnknown(right) || isAny(right) {
		return left
	}
	if l, ok := left.(*List); ok {
		if r, ok := right.(*List); ok {
			return &List{Element: Common(l.Element, r.Element, c.model)}
		}
	}
	if l, ok := left.(*Interval); ok {
		if r, ok := right.(*Interval); ok {
			return &Interval{Point: Common(l.Point, r.Point, c.model)}
		}
	}
	c.reportf(e, SeverityError, "%s and %s are not both lists or both intervals", left, right)
	return Unknown
}

// inferUnary types a prefix operator.
func (c *checker) inferUnary(e *ast.UnaryExpression) Type {
	switch e.Operator {
	case ast.OpNot:
		c.expect(e.Operand, Boolean)
		return Boolean

	case ast.OpExists:
		c.infer(e.Operand)
		return Boolean

	case ast.OpPositive, ast.OpNegate:
		t := c.arithmeticOperand(e.Operand)
		if IsUnknown(t) || isNumeric(t) || Equal(t, Any) {
			return t
		}
		c.reportf(e, SeverityError, "%s is not a number", t)
		return Unknown

	case ast.OpDistinct:
		return c.listOperand(e.Operand, e)

	case ast.OpFlatten:
		t := c.listOperand(e.Operand, e)
		if outer, ok := t.(*List); ok {
			if inner, ok := outer.Element.(*List); ok {
				return inner
			}
		}
		return t

	case ast.OpSingletonFrom:
		t := c.listOperand(e.Operand, e)
		if l, ok := t.(*List); ok {
			return l.Element
		}
		return Unknown

	case ast.OpPointFrom, ast.OpStartOf, ast.OpEndOf, ast.OpWidthOf:
		return c.inferIntervalOperand(e.Operand, e)

	case ast.OpSuccessorOf, ast.OpPredecessorOf:
		return c.arithmeticOperand(e.Operand)
	}
	c.infer(e.Operand)
	return Unknown
}

// listOperand types an operand that has to be a list, and answers the list
// type it is.
func (c *checker) listOperand(expr, at ast.Expression) Type {
	t := c.infer(expr)
	if _, ok := t.(*List); ok {
		return t
	}
	if IsUnknown(t) {
		return Unknown
	}
	c.reportf(at, SeverityError, "expected a list, got %s", t)
	return Unknown
}

func isNumeric(t Type) bool {
	n, ok := t.(*Named)
	if !ok || n.Model != "System" {
		return false
	}
	switch n.Name {
	case "Integer", "Long", "Decimal", "Quantity":
		return true
	}
	return false
}

func isTemporal(t Type) bool {
	n, ok := t.(*Named)
	if !ok || n.Model != "System" {
		return false
	}
	switch n.Name {
	case "Date", "DateTime", "Time":
		return true
	}
	return false
}
