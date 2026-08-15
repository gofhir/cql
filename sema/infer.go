package sema

import (
	"github.com/gofhir/cql/ast"
)

// infer returns the type of an expression, recording it and everything it
// found on the way.
//
// Every branch ends in a type. Where one cannot be worked out the answer is
// Unknown, never a panic and never an abort: the pass has to reach the end of
// the library to report everything in it.
func (c *checker) infer(expr ast.Expression) Type {
	if expr == nil {
		return Unknown
	}
	switch e := expr.(type) {
	case *ast.Literal:
		return c.record(e, literalType(e.ValueType))
	case *ast.IdentifierRef:
		return c.record(e, c.inferIdentifier(e))
	case *ast.Retrieve:
		return c.record(e, c.inferRetrieve(e))
	case *ast.Query:
		return c.record(e, c.inferQuery(e))
	case *ast.BinaryExpression:
		return c.record(e, c.inferBinary(e))
	case *ast.UnaryExpression:
		return c.record(e, c.inferUnary(e))
	case *ast.MemberAccess:
		return c.record(e, c.inferMemberAccess(e))
	case *ast.FunctionCall:
		return c.record(e, c.inferFunctionCall(e))
	case *ast.IndexAccess:
		return c.record(e, c.inferIndexAccess(e))
	case *ast.IfThenElse:
		return c.record(e, c.inferIf(e))
	case *ast.CaseExpression:
		return c.record(e, c.inferCase(e))
	case *ast.IntervalExpression:
		return c.record(e, c.inferIntervalConstructor(e))
	case *ast.ListExpression:
		return c.record(e, c.inferListConstructor(e))
	case *ast.TupleExpression:
		return c.record(e, c.inferTupleConstructor(e))
	case *ast.InstanceExpression:
		return c.record(e, c.inferInstance(e))
	case *ast.IsExpression:
		c.infer(e.Operand)
		return c.record(e, Boolean)
	case *ast.BooleanTestExpression:
		c.infer(e.Operand)
		return c.record(e, Boolean)
	case *ast.AsExpression:
		c.infer(e.Operand)
		return c.record(e, c.resolveTypeSpecifier(e.Type))
	case *ast.CastExpression:
		c.infer(e.Operand)
		return c.record(e, c.resolveTypeSpecifier(e.Type))
	case *ast.ConvertExpression:
		return c.record(e, c.inferConvert(e))
	case *ast.BetweenExpression:
		return c.record(e, c.inferBetween(e))
	case *ast.TimingExpression:
		return c.record(e, c.inferTiming(e))
	case *ast.MembershipExpression:
		return c.record(e, c.inferMembership(e))
	case *ast.DurationBetween:
		c.expectTemporalOrNumeric(e.Low)
		c.expectTemporalOrNumeric(e.High)
		return c.record(e, Integer)
	case *ast.DifferenceBetween:
		c.expectTemporalOrNumeric(e.Low)
		c.expectTemporalOrNumeric(e.High)
		return c.record(e, Integer)
	case *ast.DurationOf:
		c.inferIntervalOperand(e.Operand, e)
		return c.record(e, Integer)
	case *ast.DifferenceOf:
		c.inferIntervalOperand(e.Operand, e)
		return c.record(e, Integer)
	case *ast.DateTimeComponentFrom:
		return c.record(e, c.inferComponentFrom(e))
	case *ast.TypeExtent:
		return c.record(e, c.resolveNamedType(e.Type))
	case *ast.CodeExpression:
		return c.record(e, Code)
	case *ast.ConceptExpression:
		return c.record(e, Concept)
	case *ast.ExternalConstant:
		// %-constants come from outside the library, with nothing declaring
		// what they are.
		return c.record(e, Unknown)
	case *ast.ThisExpression:
		return c.record(e, c.implicitOr(Unknown))
	case *ast.IndexExpression:
		return c.record(e, Integer)
	case *ast.TotalExpression:
		if t, ok := c.lookup("$total"); ok {
			return c.record(e, t)
		}
		return c.record(e, Unknown)
	case *ast.SetAggregateExpression:
		return c.record(e, c.inferSetAggregate(e))
	}
	return c.record(expr, Unknown)
}

func literalType(kind ast.LiteralType) Type {
	switch kind {
	case ast.LiteralBoolean:
		return Boolean
	case ast.LiteralString:
		return String
	case ast.LiteralInteger:
		return Integer
	case ast.LiteralLong:
		return Long
	case ast.LiteralDecimal:
		return Decimal
	case ast.LiteralDate:
		return Date
	case ast.LiteralDateTime:
		return DateTime
	case ast.LiteralTime:
		return Time
	case ast.LiteralQuantity:
		return Quantity
	case ast.LiteralRatio:
		return Ratio
	case ast.LiteralNull:
		// `null` is of no particular type; it takes the one the context wants,
		// which is what Any means here.
		return Any
	}
	return Unknown
}

// inferIdentifier resolves a name the way CQL scoping says to: innermost
// binding first, then the library's own declarations, and only then as a
// property of whatever is implicitly in scope.
func (c *checker) inferIdentifier(e *ast.IdentifierRef) Type {
	if e.Library != "" {
		// A name qualified by an included library. Resolving it needs that
		// library's AST, which this pass is not given; saying nothing is
		// better than reporting a name that is very likely fine.
		return Unknown
	}
	if t, ok := c.lookup(e.Name); ok {
		return t
	}
	if t, ok := c.params[e.Name]; ok {
		return t
	}
	if _, ok := c.defNodes[e.Name]; ok {
		return c.typeOfDefine(e.Name)
	}
	if t, ok := c.terminology[e.Name]; ok {
		return t
	}
	// The context statement introduces a definition of its own name: in
	// `context Patient`, `Patient` is the patient being evaluated.
	if named, ok := c.contextType.(*Named); ok && named.Name == e.Name {
		return c.contextType
	}
	// Last, an unqualified property of what is implicitly in scope — the query
	// alias inside a query, the context outside one.
	//
	// When there is nothing in scope to be a property *of*, the name is
	// undefined and saying so is the whole point. When what is in scope could
	// not be typed, it is not: the property may well exist on whatever that
	// turns out to be, and blaming the author for a type this phase failed to
	// work out is how a checker earns its reputation for crying wolf.
	if !IsUnknown(c.implicit) {
		if t, ok := c.propertyOf(c.implicit, e.Name, e); ok {
			return t
		}
	} else if len(c.scopes) > 0 {
		return Unknown
	}
	c.reportf(e, SeverityError, "%s is not defined", e.Name)
	return Unknown
}

func (c *checker) implicitOr(fallback Type) Type {
	if c.implicit != nil && !IsUnknown(c.implicit) {
		return c.implicit
	}
	return fallback
}

// inferRetrieve types a data access. A retrieve always yields a list, even when
// the data has one row or none.
func (c *checker) inferRetrieve(e *ast.Retrieve) Type {
	if e.Codes != nil {
		c.infer(e.Codes)
	}
	if e.DateRange != nil {
		c.infer(e.DateRange)
	}
	if e.Context != nil {
		c.infer(e.Context)
	}
	if e.ResourceType == nil {
		return &List{Element: Unknown}
	}
	return &List{Element: c.resolveNamedType(e.ResourceType)}
}

// inferMemberAccess resolves a dotted property against the model.
func (c *checker) inferMemberAccess(e *ast.MemberAccess) Type {
	source := c.infer(e.Source)
	t, ok := c.propertyOf(source, e.Member, e)
	if ok {
		return t
	}
	if IsUnknown(source) {
		return Unknown // already reported wherever the source went wrong
	}
	c.reportf(e, SeverityError, "%s has no element %s", source, e.Member)
	return Unknown
}

// propertyOf resolves one property of one type, reporting nothing: callers
// decide whether a miss is an error, because an unqualified identifier that is
// not a property is a different mistake from a dotted one that is not.
//
// A property of a list is the property of its elements, gathered back into a
// list. CQL proper requires a query for that and the reference translator
// refuses it; this engine's FHIRPath layer allows it, and the two disagreeing
// about what is legal is not this phase's argument to have — it types what the
// evaluator will do.
func (c *checker) propertyOf(source Type, name string, at ast.Expression) (Type, bool) {
	switch s := source.(type) {
	case nil:
		return nil, false
	case unknownType:
		return Unknown, true
	case *List:
		inner, ok := c.propertyOf(s.Element, name, at)
		if !ok {
			return nil, false
		}
		if _, alreadyList := inner.(*List); alreadyList {
			return inner, true // flattened, as FHIRPath collections are
		}
		return &List{Element: inner}, true
	case *Tuple:
		return s.Element(name)
	case *Choice:
		// A property of a choice is whatever it is on the branches that have
		// it. Branches without it are not an error: narrowing by `as` is how
		// CQL says which branch was meant, and the property exists on the one
		// the author intends.
		var found []Type
		for _, branch := range s.Types {
			if t, ok := c.propertyOf(branch, name, at); ok {
				found = append(found, t)
			}
		}
		if len(found) == 0 {
			return nil, false
		}
		return NewChoice(found), true
	case *Named:
		return c.namedProperty(s, name, at)
	}
	return nil, false
}

// namedProperty resolves a property of a named type: an element the model
// declares, or the identity accessors CQL puts on its own types.
func (c *checker) namedProperty(s *Named, name string, at ast.Expression) (Type, bool) {
	if s.Model == "System" {
		return systemProperty(s, name)
	}
	if c.model == nil {
		return Unknown, true // no model to ask; not the author's mistake
	}
	if t, ok := c.model.ElementType(s.String(), name); ok {
		return t, true
	}
	// A property the model does not declare on the type itself may still be
	// reachable through the conversion the model declares for it — `.value` on
	// a FHIR primitive, which is what makes `Enc.status.value` a String.
	for _, mc := range c.model.ConversionsFrom(s.String()) {
		target := ParseTypeName(mc.To)
		if t, ok := c.propertyOf(target, name, at); ok {
			return t, true
		}
	}
	return nil, false
}

// systemProperty covers the properties CQL defines on its own types.
func systemProperty(s *Named, name string) (Type, bool) {
	switch s.Name {
	case "Quantity":
		switch name {
		case "value":
			return Decimal, true
		case "unit":
			return String, true
		}
	case "Ratio":
		switch name {
		case "numerator", "denominator":
			return Quantity, true
		}
	case "Code":
		switch name {
		case "code", "system", "version", "display":
			return String, true
		}
	case "Concept":
		switch name {
		case "codes":
			return &List{Element: Code}, true
		case "display":
			return String, true
		}
	case "ValueSet":
		switch name {
		case "id", "version":
			return String, true
		case "codesystems":
			return &List{Element: CodeSystem}, true
		}
	case "CodeSystem":
		switch name {
		case "id", "version":
			return String, true
		}
	case "Any":
		return Unknown, true
	}
	return nil, false
}

// inferIndexAccess types expr[i].
func (c *checker) inferIndexAccess(e *ast.IndexAccess) Type {
	source := c.infer(e.Source)
	c.expect(e.Index, Integer)
	if l, ok := source.(*List); ok {
		return l.Element
	}
	if IsUnknown(source) {
		return Unknown
	}
	if s, ok := source.(*Named); ok && Equal(s, String) {
		return String // indexing a string yields a one-character string
	}
	c.reportf(e, SeverityError, "%s cannot be indexed", source)
	return Unknown
}

// inferIf types if-then-else: the condition must be a Boolean, and the result
// is what both branches together are.
func (c *checker) inferIf(e *ast.IfThenElse) Type {
	c.expect(e.Condition, Boolean)
	then := c.infer(e.Then)
	if e.Else == nil {
		return then
	}
	return Common(then, c.infer(e.Else), c.model)
}

func (c *checker) inferCase(e *ast.CaseExpression) Type {
	var comparand Type
	if e.Comparand != nil {
		comparand = c.infer(e.Comparand)
	}
	result := Unknown
	for _, item := range e.Items {
		if comparand == nil {
			c.expect(item.When, Boolean)
		} else {
			c.expect(item.When, comparand)
		}
		result = Common(result, c.infer(item.Then), c.model)
	}
	if e.Else != nil {
		result = Common(result, c.infer(e.Else), c.model)
	}
	return result
}

// inferIntervalConstructor types Interval[low, high].
func (c *checker) inferIntervalConstructor(e *ast.IntervalExpression) Type {
	low, high := Unknown, Unknown
	if e.Low != nil {
		low = c.infer(e.Low)
	}
	if e.High != nil {
		high = c.infer(e.High)
	}
	point := Common(low, high, c.model)
	if IsUnknown(point) || Equal(point, Any) {
		// Interval[null, null] says nothing about its points; neither does an
		// interval over something this phase could not type.
		return &Interval{Point: Unknown}
	}
	return &Interval{Point: point}
}

func (c *checker) inferListConstructor(e *ast.ListExpression) Type {
	if e.TypeSpec != nil {
		declared := c.resolveTypeSpecifier(e.TypeSpec)
		for _, el := range e.Elements {
			c.expect(el, declared)
		}
		return &List{Element: declared}
	}
	element := Unknown
	for _, el := range e.Elements {
		element = Common(element, c.infer(el), c.model)
	}
	return &List{Element: element}
}

func (c *checker) inferTupleConstructor(e *ast.TupleExpression) Type {
	elements := make([]TupleElement, 0, len(e.Elements))
	for _, el := range e.Elements {
		elements = append(elements, TupleElement{Name: el.Name, Type: c.infer(el.Expression)})
	}
	return NewTuple(elements)
}

// inferInstance types `TypeName { field: value }`, checking each field against
// what the model says that element is.
func (c *checker) inferInstance(e *ast.InstanceExpression) Type {
	t := c.resolveNamedType(e.Type)
	for _, el := range e.Elements {
		want, ok := c.propertyOf(t, el.Name, e)
		if !ok {
			c.infer(el.Expression)
			if !IsUnknown(t) && c.model != nil {
				c.reportf(e, SeverityError, "%s has no element %s", t, el.Name)
			}
			continue
		}
		c.expect(el.Expression, want)
	}
	return t
}

func (c *checker) inferConvert(e *ast.ConvertExpression) Type {
	operand := c.infer(e.Operand)
	if e.ToType != nil {
		return c.resolveTypeSpecifier(e.ToType)
	}
	if e.ToUnit != "" {
		return Quantity
	}
	return operand
}

// inferBetween types `x between low and high`, which is a pair of comparisons
// and so a Boolean.
func (c *checker) inferBetween(e *ast.BetweenExpression) Type {
	operand := c.infer(e.Operand)
	c.expect(e.Low, operand)
	c.expect(e.High, operand)
	return Boolean
}

// inferTiming types the interval timing operators. Every one of them answers a
// Boolean; what they need is for their operands to be intervals or points, and
// a FHIR.Period is an interval only after the model's conversion is applied —
// which is exactly the conversion this records.
func (c *checker) inferTiming(e *ast.TimingExpression) Type {
	c.inferTimingOperand(e.Left)
	c.inferTimingOperand(e.Right)
	return Boolean
}

// inferTimingOperand types one side of a timing expression, converting a model
// type into the interval it stands for where the model says how.
func (c *checker) inferTimingOperand(expr ast.Expression) Type {
	t := c.infer(expr)
	if _, ok := t.(*Interval); ok {
		return t
	}
	if converted, ok := c.convertToInterval(expr, t); ok {
		return converted
	}
	return t // a point operand, which the timing operators also accept
}

// convertToInterval applies a model-declared conversion whose target is an
// interval, recording it against the node so a caller can see where the
// reference translator would have inserted a FHIRHelpers call.
func (c *checker) convertToInterval(expr ast.Expression, t Type) (Type, bool) {
	named, ok := t.(*Named)
	if !ok || c.model == nil || named.Model == "System" {
		return nil, false
	}
	for _, mc := range c.model.ConversionsFrom(named.String()) {
		target := ParseTypeName(mc.To)
		if _, isInterval := target.(*Interval); !isInterval {
			continue
		}
		c.convert(expr, Conversion{Cost: costModel, Function: mc.Function, Target: target})
		return target, true
	}
	return nil, false
}

// inferIntervalOperand types an operand that has to be an interval, and returns
// its point type.
func (c *checker) inferIntervalOperand(expr, at ast.Expression) Type {
	t := c.inferTimingOperand(expr)
	if iv, ok := t.(*Interval); ok {
		return iv.Point
	}
	if IsUnknown(t) {
		return Unknown
	}
	c.reportf(at, SeverityError, "expected an interval, got %s", t)
	return Unknown
}

func (c *checker) inferMembership(e *ast.MembershipExpression) Type {
	left := c.infer(e.Left)
	right := c.infer(e.Right)
	// `in` accepts a list, an interval or a valueset on the right; `contains`
	// is the same operator with its operands the other way round.
	if e.Operator == "contains" {
		left, right = right, left
	}
	_ = left
	switch right.(type) {
	case *List, *Interval, unknownType, nil:
		return Boolean
	}
	if n, ok := right.(*Named); ok {
		switch n.Name {
		case "ValueSet", "CodeSystem", "Vocabulary", "Concept", "Any":
			return Boolean
		}
		// A FHIR type that converts to an interval — a Period — is a legal
		// right operand once converted.
		if e.Operator == "contains" {
			if _, ok := c.convertToInterval(e.Left, right); ok {
				return Boolean
			}
		} else if _, ok := c.convertToInterval(e.Right, right); ok {
			return Boolean
		}
	}
	c.reportf(e, SeverityError, "%s is not something a value can be in", right)
	return Boolean
}

// inferComponentFrom types `year from x` and its siblings.
func (c *checker) inferComponentFrom(e *ast.DateTimeComponentFrom) Type {
	c.infer(e.Operand)
	switch e.Component {
	case "date":
		return Date
	case "time":
		return Time
	case "timezoneoffset":
		return Decimal
	}
	return Integer
}

// inferSetAggregate types `expand` and `collapse`, both of which take a list of
// intervals and answer one.
func (c *checker) inferSetAggregate(e *ast.SetAggregateExpression) Type {
	operand := c.infer(e.Operand)
	if e.Per != nil {
		c.infer(e.Per)
	}
	if l, ok := operand.(*List); ok {
		if _, isInterval := l.Element.(*Interval); isInterval {
			return operand
		}
		if e.Kind == "expand" {
			// Expanding a list of intervals per a quantity yields the points
			// themselves when the interval type is not preserved.
			return operand
		}
	}
	return operand
}

// expectTemporalOrNumeric types an operand of a duration or difference, which
// both need something that can be subtracted.
func (c *checker) expectTemporalOrNumeric(expr ast.Expression) {
	if expr == nil {
		return
	}
	t := c.infer(expr)
	if named, ok := t.(*Named); ok && named.Model != "System" && c.model != nil {
		// A FHIR dateTime is a System.DateTime once converted, and every
		// duration between two of them depends on it.
		c.convertToSystem(expr, named)
	}
}

// convertToSystem applies the model's conversion from a model type to a system
// one, recording it. Where the model declares several, the one this takes is
// the only one that is not itself a model type — conversions do not chain.
func (c *checker) convertToSystem(expr ast.Expression, from *Named) (Type, bool) {
	for _, mc := range c.model.ConversionsFrom(from.String()) {
		target := ParseTypeName(mc.To)
		if !IsSystem(target) {
			continue
		}
		c.convert(expr, Conversion{Cost: costModel, Function: mc.Function, Target: target})
		return target, true
	}
	return nil, false
}

// expect types an expression and checks it against the type the context needs,
// recording the conversion where one is required and reporting where none
// exists.
func (c *checker) expect(expr ast.Expression, want Type) {
	if expr == nil {
		return
	}
	got := c.infer(expr)
	if IsUnknown(got) || IsUnknown(want) {
		return
	}
	conv, ok := Convertible(got, want, c.model)
	if !ok {
		c.reportf(expr, SeverityError, "expected %s, got %s", want, got)
		return
	}
	c.convert(expr, conv)
}
