package sema

import (
	"strings"

	"github.com/gofhir/cql/ast"
)

// inferQuery types a CQL query.
//
// The shape of the answer follows the sources: a query over a list yields a
// list, a query over a single value yields a single value, and `aggregate`
// yields whatever it accumulated, list or not. What it yields *of* is the
// return clause, or the source element when there is none — a tuple of the
// aliases when there is more than one source, which is what a multi-source
// query without a return means.
func (c *checker) inferQuery(e *ast.Query) Type {
	scope := map[string]Type{}
	elements := make([]TupleElement, 0, len(e.Sources))
	overList := false

	for _, src := range e.Sources {
		t := c.infer(src.Source)
		element := t
		if l, ok := t.(*List); ok {
			element = l.Element
			overList = true
		}
		scope[src.Alias] = element
		elements = append(elements, TupleElement{Name: src.Alias, Type: element})
	}

	c.pushScope(scope)
	outerImplicit := c.implicit
	if len(elements) == 1 {
		// A single-source query is what makes an unqualified property mean a
		// property of the thing being iterated.
		c.implicit = elements[0].Type
	}
	defer func() {
		c.implicit = outerImplicit
		c.popScope()
	}()

	// Let bindings see the aliases and each other, in order.
	for _, let := range e.Let {
		scope[let.Identifier] = c.infer(let.Expression)
	}

	for _, with := range e.With {
		c.inferRelationship(with.Source, with.Condition)
	}
	for _, without := range e.Without {
		c.inferRelationship(without.Source, without.Condition)
	}

	if e.Where != nil {
		c.expect(e.Where, Boolean)
	}

	if e.Aggregate != nil {
		return c.inferAggregate(e.Aggregate)
	}

	result := Unknown
	switch {
	case e.Return != nil:
		result = c.infer(e.Return.Expression)
	case len(elements) == 1:
		result = elements[0].Type
	case len(elements) > 1:
		result = NewTuple(elements)
	}

	// Sort keys are expressions over the result, so they are typed with the
	// result element in scope rather than the sources.
	if e.Sort != nil {
		c.inferSortKeys(e.Sort, result)
	}

	if overList {
		return &List{Element: result}
	}
	return result
}

// inferRelationship types a with/without clause: its source binds an alias for
// the duration of the such-that condition, and nothing beyond it.
func (c *checker) inferRelationship(source *ast.AliasedSource, condition ast.Expression) {
	if source == nil {
		return
	}
	t := c.infer(source.Source)
	element := t
	if l, ok := t.(*List); ok {
		element = l.Element
	}
	c.pushScope(map[string]Type{source.Alias: element})
	if condition != nil {
		c.expect(condition, Boolean)
	}
	c.popScope()
}

// inferAggregate types an aggregate clause, whose expression accumulates into
// $total.
//
// $total starts as the starting expression's type, or as the accumulating
// expression's own type where there is no starting clause — which needs one
// pass with $total unknown before the type is available. Unknown propagates
// harmlessly, so the first pass costs a walk and reports nothing new.
func (c *checker) inferAggregate(agg *ast.AggregateClause) Type {
	total := Unknown
	if agg.Starting != nil {
		total = c.infer(agg.Starting)
	}
	c.pushScope(map[string]Type{"$total": total, agg.Identifier: total})
	t := c.infer(agg.Expression)
	c.popScope()

	if IsUnknown(total) && !IsUnknown(t) {
		// Now that the accumulator's type is known, type the body again with
		// $total bound, so that expressions using it are recorded correctly.
		c.pushScope(map[string]Type{"$total": t, agg.Identifier: t})
		t = c.infer(agg.Expression)
		c.popScope()
	}
	return t
}

// inferSortKeys types the expressions a sort orders by, with the element being
// sorted in scope as $this.
func (c *checker) inferSortKeys(s *ast.SortClause, element Type) {
	if len(s.ByItems) == 0 {
		return
	}
	c.pushScope(map[string]Type{"$this": element})
	outer := c.implicit
	c.implicit = element
	for _, item := range s.ByItems {
		c.infer(item.Expression)
	}
	c.implicit = outer
	c.popScope()
}

// inferFunctionCall types an invocation: a function this library defines, or
// one of the operators the specification defines as a function.
func (c *checker) inferFunctionCall(e *ast.FunctionCall) Type {
	args := make([]Type, 0, len(e.Operands)+1)
	if e.Source != nil {
		// A method-style call passes its source as the first argument, which is
		// how both fluent functions and the FHIRPath-style operators read.
		args = append(args, c.infer(e.Source))
	}
	for _, operand := range e.Operands {
		args = append(args, c.infer(operand))
	}

	if e.Library != "" {
		// Defined in an included library, whose AST this pass is not given.
		return Unknown
	}
	if overloads, ok := c.funcNodes[e.Name]; ok {
		if fn := c.resolveOverload(overloads, args); fn != nil {
			return c.typeOfFunction(fn, args)
		}
		c.reportf(e, SeverityError, "no overload of %s takes %s", e.Name, describeArgs(args))
		return Unknown
	}
	if t, ok := systemFunctionType(e.Name, args, c.model); ok {
		return t
	}
	// Reported as a warning, not an error: this table covers what the engine
	// implements, and a library may legitimately call something it does not yet
	// know — a fluent function reached through an include, most of all. Being
	// wrong about that should not make a library refuse to compile.
	c.reportf(e, SeverityWarning, "unknown function %s", e.Name)
	return Unknown
}

// resolveOverload picks the overload whose operands the arguments reach most
// cheaply, which is what makes an exact match beat one that needs a conversion.
//
// Ties keep the one declared first. Reporting the ambiguity would be more
// correct and less useful: overloads that tie are ones whose operand types
// convert to each other, and a library that declares them has already decided
// it does not care which runs.
func (c *checker) resolveOverload(overloads []*ast.FunctionDef, args []Type) *ast.FunctionDef {
	best, bestCost := (*ast.FunctionDef)(nil), 0
	for _, fn := range overloads {
		if len(fn.Operands) != len(args) {
			continue
		}
		cost, ok := 0, true
		for i, op := range fn.Operands {
			want := c.resolveTypeSpecifier(op.Type)
			if IsUnknown(want) {
				continue // an untyped operand accepts anything, at no cost
			}
			conv, fits := Convertible(args[i], want, c.model)
			if !fits {
				ok = false
				break
			}
			cost += conv.Cost
		}
		if !ok {
			continue
		}
		if best == nil || cost < bestCost {
			best, bestCost = fn, cost
		}
	}
	return best
}

func describeArgs(args []Type) string {
	if len(args) == 0 {
		return "no arguments"
	}
	names := make([]string, len(args))
	for i, a := range args {
		names[i] = a.String()
	}
	return "(" + strings.Join(names, ", ") + ")"
}
