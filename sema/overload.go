package sema

import "github.com/gofhir/cql/ast"

// Overload is which signature of includes, included in or during the static
// types of its operands select.
//
// The evaluator needs it where a value cannot answer: with a null on one side,
// `{ 's' } includes (null as String)` is List Contains, false, and
// `{ 's' } includes (null as List<String>)` is List Includes, null. Only this
// phase knows the types, and the model that says a FHIR.string is a String.
type Overload int

const (
	// ElementOverload is a list and one of its elements, or an interval and a
	// point: a synonym of contains and in.
	ElementOverload Overload = iota + 1
	// ListOverload is two lists.
	ListOverload
)

// OverloadOf returns the overload recorded for a timing expression, if this
// phase decided one.
func (r *Result) OverloadOf(expr ast.Expression) (Overload, bool) {
	if r == nil || expr == nil {
		return 0, false
	}
	o, ok := r.Overloads[expr]
	return o, ok
}

// recordOverload decides the overload of includes, included in and during from
// the types inferTiming gave their operands, after a model conversion to an
// interval, and records it.
//
// It is the translator's resolution, narrowed to what it does: the element
// overload when the element and the container's element or point meet, else
// the list one when the two lists do. A choice is read by its branches, and a
// branch that selects the element overload wins over one that selects the list
// one, as the translator reads `{ 'a', null } includes (null as
// Choice<List<FHIR.string>, String>)`: Contains. `properly` and an offset are
// other operators, and decide nothing here.
func (c *checker) recordOverload(e *ast.TimingExpression, left, right Type) {
	op := e.Operator
	if c.speculative > 0 || op.Properly || op.Offset != "" {
		return
	}
	container, element := left, right
	switch op.Kind {
	case ast.TimingIncludes:
	case ast.TimingIncludedIn, ast.TimingDuring:
		container, element = right, left
	default:
		return
	}
	var decided Overload
	for _, ct := range branches(container) {
		for _, el := range branches(element) {
			if o := c.overloadOf(c.containerForm(ct), el); o != 0 && (decided == 0 || o == ElementOverload) {
				decided = o
			}
		}
	}
	if decided != 0 {
		c.overloads[e] = decided
	}
}

// overloadOf is the overload a container of one type and an element of another
// select, or 0 for none. Any and Unknown meet only themselves: the translator
// also reads `{ 1 } includes (null as Any)` as Contains, by subtype, and that
// is left undecided here (backlog 1c8).
func (c *checker) overloadOf(container, element Type) Overload {
	switch t := container.(type) {
	case *List:
		if c.meets(element, t.Element) {
			return ElementOverload
		}
		if c.meets(element, t) {
			return ListOverload
		}
	case *Interval:
		if c.meets(element, t.Point) {
			return ElementOverload
		}
	}
	return 0
}

// containerForm is a container as the operator reads it: a list or an
// interval as it is, a model type through the conversion the model declares to
// an interval, and nothing for anything else. inferTimingOperand has already
// converted a container of a model type; what reaches the model case here is a
// branch of a choice, as the Period of Observation.effective.
func (c *checker) containerForm(t Type) Type {
	switch t.(type) {
	case *List, *Interval:
		return t
	}
	named, ok := t.(*Named)
	if !ok || c.model == nil || named.Model == systemModel {
		return nil
	}
	for _, mc := range c.model.ConversionsFrom(named.String()) {
		if target, isInterval := ParseTypeName(mc.To).(*Interval); isInterval {
			return target
		}
	}
	return nil
}

// meets reports whether one of a and b reaches the other: overload resolution
// may convert either operand.
func (c *checker) meets(a, b Type) bool {
	return c.reaches(a, b) || c.reaches(b, a)
}

// reaches reports whether from is to, or becomes it by the conversions the
// translator applies to resolve these operators: a System one, one model
// conversion whose target is to itself, or a model subtype. Lists and intervals
// convert element by element, and a choice reaches, or is reached, by any of
// its branches: an element of a list can be a choice, as Observation.component
// .value is.
//
// It is narrower than Convertible on purpose. Convertible also turns a value
// into a list of one and back, and chains a model conversion with a System one,
// and the translator does neither here: with them, `{ 's' } includes (null as
// List<String>)` read as the element overload, and `Patient.birthDate during
// (null as FHIR.Period)`, which the translator refuses, as a membership.
func (c *checker) reaches(from, to Type) bool {
	if ImplicitlyConverts(from, to) {
		return true
	}
	if choice, ok := to.(*Choice); ok {
		for _, branch := range choice.Types {
			if c.reaches(from, branch) {
				return true
			}
		}
	}
	switch f := from.(type) {
	case *Choice:
		for _, branch := range f.Types {
			if c.reaches(branch, to) {
				return true
			}
		}
	case *List:
		t, ok := to.(*List)
		return ok && c.reaches(f.Element, t.Element)
	case *Interval:
		t, ok := to.(*Interval)
		return ok && c.reaches(f.Point, t.Point)
	case *Named:
		if c.model == nil || f.Model == systemModel {
			return false
		}
		if t, ok := to.(*Named); ok && t.Model != systemModel && c.model.IsSubtypeOf(f.String(), t.String()) {
			return true
		}
		for _, mc := range c.model.ConversionsFrom(f.String()) {
			if Equal(ParseTypeName(mc.To), to) {
				return true
			}
		}
	}
	return false
}

// branches is a choice's types, or the type alone.
func branches(t Type) []Type {
	if choice, ok := t.(*Choice); ok {
		return choice.Types
	}
	return []Type{t}
}
