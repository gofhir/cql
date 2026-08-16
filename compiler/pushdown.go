package compiler

import (
	"strings"

	"github.com/gofhir/cql/ast"
)

// pushDownDateRanges moves a query's date filter onto the retrieve it reads
// from, so that a data provider can narrow the query instead of the engine
// discarding rows it already paid to fetch.
//
// `[Encounter] E where E.period during "Measurement Period"` reaches a provider
// as a request for every Encounter, and the period is applied afterwards in
// memory. On a real population that is not merely slow: the retrieve exceeds
// MaxRetrieveSize and the evaluation fails before any filtering happens. Only
// the ELM importer ever filled DateRange in, so a library evaluated from CQL
// text — which is what a server receives — could never push one down.
//
// The where clause is left in place. Pushing a filter down is a request, not a
// guarantee: a provider may ignore DateRange, and the FHIR date search it will
// most likely map to is an overlap test, which returns a superset of what
// `during` means. Keeping the clause makes both of those harmless, where the
// reference implementation's habit of removing it makes correctness depend on
// every provider being exact.
func pushDownDateRanges(lib *ast.Library) {
	for _, stmt := range lib.Statements {
		stmt.Expression = pushDownInExpression(stmt.Expression)
	}
}

// pushDownInExpression rewrites the queries it can reach. Only a query's own
// clauses are followed, which is enough for the shape this exists for and
// leaves every other node untouched.
func pushDownInExpression(expr ast.Expression) ast.Expression {
	query, ok := expr.(*ast.Query)
	if !ok {
		return expr
	}
	pushDownIntoQuery(query)
	return query
}

// pushDownIntoQuery attaches a date filter to a query's retrieve when doing so
// cannot change what the query returns.
func pushDownIntoQuery(query *ast.Query) {
	// One source only. With several, a predicate mentioning one alias says
	// nothing about the rows of another, and which retrieve to narrow stops
	// being a question with one answer.
	if len(query.Sources) != 1 || query.Where == nil {
		return
	}
	source := query.Sources[0]
	retrieve, ok := source.Source.(*ast.Retrieve)
	if !ok || retrieve.DateRange != nil {
		return
	}
	for _, term := range conjuncts(query.Where) {
		path, rng, ok := dateFilter(term, source.Alias)
		if !ok {
			continue
		}
		retrieve.DatePath = path
		retrieve.DateRange = rng
		return
	}
}

// conjuncts splits a where clause into the terms that are and-ed together at
// the top level.
//
// Only these can be pushed down. Each conjunct is implied by the whole clause,
// so narrowing the retrieve by one of them cannot drop a row the clause would
// have kept. A disjunct is not: `E.period during X or E.status = 'finished'`
// keeps rows outside X, and a retrieve narrowed to X would never offer them.
func conjuncts(expr ast.Expression) []ast.Expression {
	bin, ok := expr.(*ast.BinaryExpression)
	if !ok || bin.Operator != ast.OpAnd {
		return []ast.Expression{expr}
	}
	return append(conjuncts(bin.Left), conjuncts(bin.Right)...)
}

// dateFilter reports the element path and interval a term constrains, when the
// term is one this can safely hand to a provider.
func dateFilter(term ast.Expression, alias string) (path string, rng ast.Expression, ok bool) {
	var left, right ast.Expression
	switch t := term.(type) {
	case *ast.TimingExpression:
		// `during` and `included in` say the element falls inside the interval;
		// `overlaps` says it meets it. All three are answered by asking a
		// provider for the interval and filtering afterwards, since the rows
		// each admits are a subset of the rows an overlap query returns.
		// `includes` is the one that reads the other way — it constrains the
		// interval by the row — and narrowing the retrieve by it is wrong.
		//
		// TimingDuring is accepted although the builder does not currently
		// produce it: `during` parses as `included in`, the operator CQL
		// defines it as. Refusing a kind that means the same thing would be a
		// silent loss if that ever changes.
		switch t.Operator.Kind {
		case ast.TimingDuring, ast.TimingIncludedIn, ast.TimingOverlaps:
		default:
			return "", nil, false
		}
		// `properly` narrows what qualifies, so the query stays a subset and
		// pushing down is still safe.
		//
		// A direction modifier is refused for two different reasons. On
		// `starts before` the rows are outside the interval, and asking for the
		// interval would exclude exactly the ones meant to be kept. On
		// `overlaps after` the rows do meet the interval, so pushing down
		// would in fact be safe — it is refused because getting the boundary
		// right for each combination is work this has not done, and the cost of
		// refusing is a slower query rather than a wrong one.
		if t.Operator.Before || t.Operator.After {
			return "", nil, false
		}
		left, right = t.Left, t.Right
	case *ast.MembershipExpression:
		// `in` is membership of any kind, and most of what it tests is not a
		// date: `C.code in "Diabetes"` asks a terminology question about a
		// value set. Pushing that down as a date range makes the retrieve fail
		// outright, because the value set name is not an expression the
		// evaluator can resolve where the interval is evaluated.
		//
		// A temporal `in` says so in the syntax: it names an interval, or it
		// carries a date-time precision. Nothing else qualifies.
		if t.Operator != "in" || !isTemporalMembership(t) {
			return "", nil, false
		}
		left, right = t.Left, t.Right
	default:
		return "", nil, false
	}

	path, ok = aliasPath(left, alias)
	if !ok {
		return "", nil, false
	}
	// The interval is evaluated before the retrieve runs, so it cannot be
	// built from the rows the retrieve is about to return.
	if !isAliasFree(right, alias) {
		return "", nil, false
	}
	return path, right, true
}

// isTemporalMembership reports whether an `in` is asking about an interval
// rather than about membership in a list or a value set.
//
// The test is syntactic on purpose: this runs before anything has worked out
// types, and a name on the right could be a parameter holding an interval or a
// value set holding codes. Only a stated interval or a date-time precision
// settles it from the source alone.
func isTemporalMembership(e *ast.MembershipExpression) bool {
	if e.Precision != "" {
		return true
	}
	_, ok := e.Right.(*ast.IntervalExpression)
	return ok
}

// aliasPath reads `E.period` or `E.period.start` as the path below the alias,
// and reports nothing for anything else.
func aliasPath(expr ast.Expression, alias string) (string, bool) {
	access, ok := expr.(*ast.MemberAccess)
	if !ok {
		return "", false
	}
	var segments []string
	for {
		segments = append([]string{access.Member}, segments...)
		switch source := access.Source.(type) {
		case *ast.MemberAccess:
			access = source
		case *ast.IdentifierRef:
			if source.Library != "" || source.Name != alias {
				return "", false
			}
			return strings.Join(segments, "."), true
		default:
			return "", false
		}
	}
}

// isAliasFree reports whether an expression can be evaluated without the row
// the alias stands for.
//
// The list is what this is sure about, not what is possible. An expression
// built from a node it has not been taught is treated as unsafe, so a node
// added later fails to push down rather than pushing down something wrong.
func isAliasFree(expr ast.Expression, alias string) bool {
	switch e := expr.(type) {
	case *ast.Literal:
		return true
	case *ast.IdentifierRef:
		// A parameter, a define, or a value set — unless it is the alias, which
		// is exactly the row this must not depend on.
		return e.Library != "" || e.Name != alias
	case *ast.IntervalExpression:
		return isAliasFree(e.Low, alias) && isAliasFree(e.High, alias)
	case *ast.ListExpression:
		return allAliasFree(e.Elements, alias)
	case *ast.BinaryExpression:
		return isAliasFree(e.Left, alias) && isAliasFree(e.Right, alias)
	case *ast.UnaryExpression:
		return isAliasFree(e.Operand, alias)
	case *ast.MemberAccess:
		return isAliasFree(e.Source, alias)
	case *ast.FunctionCall:
		if e.Source != nil && !isAliasFree(e.Source, alias) {
			return false
		}
		return allAliasFree(e.Operands, alias)
	case *ast.ExternalConstant:
		return true
	default:
		return false
	}
}

func allAliasFree(exprs []ast.Expression, alias string) bool {
	for _, expr := range exprs {
		if !isAliasFree(expr, alias) {
			return false
		}
	}
	return true
}
