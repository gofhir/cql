package cql

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func evalQuantityCompare(t *testing.T, expr string) string {
	t.Helper()
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ndefine A: " + expr + "\n"
	got, err := NewEngine().EvaluateExpression(context.Background(), src, "A", nil, nil)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	if got == nil {
		return "null"
	}
	return got.String()
}

// TestQuantitiesOfDifferentDimensionsAreNull covers a pair of quantities that
// cannot be compared, which the engine answered three different ways.
//
//	3.5 'cm2' =  3.5 'cm'    false
//	3.5 'cm2' != 3.5 'cm'    true
//	3.6 'cm2' <  3.5 'cm'    error: incompatible units — and it took the define
//
// CQL says what it is, twice, once under equality and once under ordering, and
// names an example of each: "QuantityNotEqualIsNull" and "QuantityLessIsNull".
//
//	"the dimensions of each quantity must be the same, but not necessarily the
//	 unit. For example, units of 'cm' and 'm' are comparable, but units of 'cm2'
//	 and 'cm' are not… Attempting to operate on quantities with invalid units
//	 will result in a null."
//
// The conformance corpus carries neither example, which is how 1823 passing cases
// coexisted with three readings of one rule.
//
// Whether two units are commensurable is not decided here. fhirpath owns unit
// algebra and exposes Quantity.Comparable — the same predicate its own `=` uses to
// return empty for `1 'kg' = 1 'm'` — so this asks rather than reimplementing it.
func TestQuantitiesOfDifferentDimensionsAreNull(t *testing.T) {
	for _, expr := range []string{
		// The specification's two named examples.
		"3.5 'cm2' != 3.5 'cm'",
		"3.6 'cm2' < 3.5 'cm'",
		// And the rest of the eight, because a rule that holds for two of them and
		// not the other six is the state this replaces.
		"3.5 'cm2' = 3.5 'cm'",
		"3.5 'cm2' <= 3.5 'cm'",
		"3.5 'cm2' > 3.5 'cm'",
		"3.5 'cm2' >= 3.5 'cm'",
		// Not only areas against lengths: any two dimensions.
		"1 'mg' = 1 's'",
		"1 'mg' < 1 's'",
	} {
		if got := evalQuantityCompare(t, expr); got != "null" {
			t.Errorf("%s = %s, want null", expr, got)
		}
	}
}

// TestEquivalenceStillDecidesIncomparableQuantities is the limit on the rule
// above, and it is a limit rather than an oversight.
//
// CQL's equivalence operator answers whether two values are the same, and it
// never returns null — that is the whole difference between `~` and `=`. So two
// quantities that cannot be compared are simply not equivalent, and fhirpath says
// the same: `1 'kg' ~ 1 'm'` is false there too.
func TestEquivalenceStillDecidesIncomparableQuantities(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"3.5 'cm2' ~ 3.5 'cm'", "false"},
		{"3.5 'cm2' !~ 3.5 'cm'", "true"},
	} {
		if got := evalQuantityCompare(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — equivalence has no null to return", tt.expr, got, tt.want)
		}
	}
}

// TestAContainerFollowsItsQuantities covers the level the last change of this
// shape had to reach and this one does too.
//
// A container may not be more certain than the values it holds. v1.20.0 removed
// that contradiction for temporals, where a list of two DateTimes at mixed
// precisions answered false while the DateTimes themselves answered null; leaving
// quantities out would put it straight back one type over.
//
// Membership is the other case and keeps its own answer, for the reason it always
// has: it has no null to hold, so two values not known to be equal stay two
// values. `1 'cm2' in { 1 'cm' }` is false, and the list has two elements.
func TestAContainerFollowsItsQuantities(t *testing.T) {
	for _, expr := range []string{
		"{ 1 'cm2' } = { 1 'cm' }",
		"Interval[1 'cm2', 2 'cm2'] = Interval[1 'cm', 2 'cm']",
		"Tuple { a: 1 'cm2' } = Tuple { a: 1 'cm' }",
		"{ 1 'cm2' } != { 1 'cm' }",
	} {
		if got := evalQuantityCompare(t, expr); got != "null" {
			t.Errorf("%s = %s, want null — the container cannot be surer than its elements", expr, got)
		}
	}

	for _, tt := range []struct{ expr, want, why string }{
		{"1 'cm2' in { 1 'cm' }", "false", "membership has no null to hold"},
		{"{ 1 'cm' } contains 1 'cm2'", "false", "the same question from the other side"},
		{"Count(distinct { 1 'cm2', 1 'cm' })", "2", "two values not known to be equal stay two"},
		{"IndexOf({ 1 'cm' }, 1 'cm2')", "-1", "not found is an answer"},
		{"Count({ 1 'cm2' } union { 1 'cm' })", "2", "neither is a duplicate of the other"},
		{"Count({ 1 'cm2' } intersect { 1 'cm' })", "0", "nothing is common to both"},
	} {
		if got := evalQuantityCompare(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — %s", tt.expr, got, tt.want, tt.why)
		}
	}
}

// TestComparableQuantitiesStillCompare is the other half of the measurement: the
// rule must fire on dimensions that differ and nowhere else.
//
// Two units of one dimension convert and compare, which is the ordinary case and
// the one a measure depends on — `Obs.value > 9 '%'` against a reading in the same
// dimension has to keep answering.
func TestComparableQuantitiesStillCompare(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"100 'cm' = 1 'm'", "true"},
		{"100 'cm' < 2 'm'", "true"},
		{"100 'cm' != 1 'm'", "false"},
		{"3.5 'cm' = 3.5 'cm'", "true"},
		{"3.5 'cm2' = 3.5 'cm2'", "true"},
		{"500 'mg' < 1 'g'", "true"},
		{"{ 1 'cm' } = { 1 'cm' }", "true"},
		{"1 'cm' in { 1 'cm' }", "true"},
		// A quantity against a bare number was left at false here, on the grounds
		// that what unit a bare number carries was a separate argument. That
		// argument has since been settled the way the specification states it —
		// the default unit is '1' — so this pair is two dimensions after all, and
		// it answers null like every other pair of dimensions that do not meet.
		{"1 'mg' = 1", "null"},
		{"1 '1' = 1", "true"},
	} {
		if got := evalQuantityCompare(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestAPointAgainstAnIntervalAgreesWithTheComparisonsItRestsOn covers `between`
// and the other ways a point is asked about an interval, which CQL defines in
// terms of the two operators this change fixed.
//
//	"If the first argument is greater than or equal to the low argument, and less
//	 than or equal to the high argument, the result is true, otherwise false."
//
// So `between` cannot answer where `>=` and `<=` decline, and it was raising an
// error where both of them now return null. It reaches the same place `in` and
// `contains` do — Interval.Contains — so the three are one question and now give
// one answer.
//
// `properly includes` over a point is here rather than with the interval
// operators because it is the same question, and because it has its own code
// path: it went through a guard that asked the narrow "was this temporal" rather
// than "could this be decided", on the reasoning that the guard protected a retry
// at a shared precision. It only does when a precision is named, and with none
// named the narrow branch returned null anyway — so the narrow question was
// guarding nothing and turning a units error into a failure.
func TestAPointAgainstAnIntervalAgreesWithTheComparisonsItRestsOn(t *testing.T) {
	for _, expr := range []string{
		"1 'cm2' between 1 'cm' and 2 'cm'",
		"1 'cm2' properly between 1 'cm' and 2 'cm'",
		// The conjunction it is defined as, which is the reason the line above
		// cannot be an error.
		"1 'cm2' >= 1 'cm' and 1 'cm2' <= 2 'cm'",
		// The same comparison reached through an interval.
		"1 's' in Interval[1 'cm', 2 'cm']",
		"Interval[1 'cm', 2 'cm'] contains 1 's'",
		"Interval[1 'cm', 2 'cm'] includes Interval[1 's', 2 's']",
		"Interval[1 's', 2 's'] included in Interval[1 'cm', 2 'cm']",
	} {
		if got := evalQuantityCompare(t, expr); got != "null" {
			t.Errorf("%s = %s, want null", expr, got)
		}
	}

	// A point rather than an interval on the right, which reaches its own code
	// path — and did so past a guard that asked the narrow question where there
	// was no retry to guard.
	for _, expr := range []string{
		"Interval[1 'cm', 2 'cm'] properly includes 1 's'",
		"1 's' properly included in Interval[1 'cm', 2 'cm']",
		"1 's' properly during Interval[1 'cm', 2 'cm']",
	} {
		if got := evalQuantityCompare(t, expr); got != "null" {
			t.Errorf("%s = %s, want null", expr, got)
		}
	}

	// And an interval of one dimension still answers about points in it.
	for _, tt := range []struct{ expr, want string }{
		{"1 'cm' between 1 'cm' and 2 'cm'", "true"},
		{"1 'cm' in Interval[1 'cm', 2 'cm']", "true"},
		{"3 'cm' in Interval[1 'cm', 2 'cm']", "false"},
		{"Interval[1 'cm', 2 'cm'] overlaps Interval[2 'cm', 3 'cm']", "true"},
		{"Interval[1 'cm', 3 'cm'] properly includes 2 'cm'", "true"},
		{"Interval[1 'cm', 3 'cm'] properly includes 1 'cm'", "false"},
		{"Interval[@2020-01, @2020-05] properly includes @2020-03", "true"},
		// The retry the narrow question does guard: a precision names one, and it
		// still happens.
		{"Interval[@2020-01-01T10:00:00, @2020-05] properly includes day of @2020-03", "true"},
	} {
		if got := evalQuantityCompare(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestSortingStillFailsOnMixedDimensions records the one place the rule stops,
// and the reason is not that it was overlooked.
//
// Every operation that returns a *value* over quantities of different dimensions
// now answers null: the comparisons, the interval and timing operators, the
// arithmetic, and every aggregate that folds them — Min and Max included, since
// a list with no minimum has none to return.
//
// A sort is the exception because it does not return a value. It has to produce
// an ordering, and there is no null ordering to produce. That is the position
// `sort by` already takes on a Period, which has no ordering either, and the
// reasoning is the same: a loud failure is worse than a right answer and better
// than a quiet wrong one, and a query whose rows cannot be ordered has no right
// answer to give.
//
// See TestSortingByAnUnorderableKeyStillFails for the Period case.
func TestSortingStillFailsOnMixedDimensions(t *testing.T) {
	if got := evalQuantityCompare(t, "Count(({1 'cm2', 1 'cm'}) X sort by X)"); !strings.HasPrefix(got, "ERROR") {
		t.Errorf("sorting quantities of different dimensions = %s — if a sort has an answer "+
			"for rows it cannot order now, decide what order they are in and remove this test", got)
	}
	// And the same list is orderable once the dimensions agree.
	if got := evalQuantityCompare(t, "First(({2 'cm', 1 'cm'}) X sort by X)"); got != "1 'cm'" {
		t.Errorf("sorting quantities of one dimension = %s, want 1 'cm'", got)
	}
}

// TestIntervalOperatorsAgreeWithTheComparisonsTheyRestOn covers the operations
// that live in funcs, which read their own copy of "this comparison could not be
// made" and had drifted from the one in eval.
//
//	Interval[1 'cm', 2 'cm'] overlaps Interval[1 's', 2 's']   error, define lost
//	1 'cm' < 1 's'                                             null
//
// One question, two packages, two answers. The copies differed in two rows — funcs
// knew neither incompatible units nor an offset mismatch — and there is now one
// predicate, in `types`, which both import and which imports neither.
func TestIntervalOperatorsAgreeWithTheComparisonsTheyRestOn(t *testing.T) {
	const mixed = "Interval[1 'cm', 2 'cm'] %s Interval[1 's', 2 's']"
	for _, op := range []string{
		"overlaps", "meets", "union", "intersect", "except",
		"starts", "ends", "same or before", "same or after",
		"includes", "included in", "properly includes",
		// These two had no guard at all where every operator around them has one.
		// They were found by enumerating what calls compareVals — the choke point
		// this package documents as "the one place it orders two values" — rather
		// than by the pattern that caught the rest, which searched for the four
		// methods the others call and so could not see them.
		"before", "after",
		"during", "properly during", "properly included in",
		"overlaps before", "overlaps after",
		// `starts before` and `ends after` are deliberately NOT here. They answer
		// null for every non-temporal interval — integers as readily as
		// quantities, and with matching dimensions as readily as mixed — so a null
		// from them is not evidence about this rule, and asserting one would be an
		// assertion that holds whatever the predicate does. Every entry above was
		// checked by breaking the predicate and confirming the line fails; those
		// two did not, which is how they were found.
	} {
		expr := fmt.Sprintf(mixed, op)
		if got := evalQuantityCompare(t, expr); got != "null" {
			t.Errorf("%s = %s, want null", expr, got)
		}
	}

	// And two intervals of one dimension are untouched, operator by operator.
	for _, tt := range []struct{ expr, want string }{
		{"Interval[1 'cm', 2 'cm'] overlaps Interval[2 'cm', 3 'cm']", "true"},
		{"Interval[1 'cm', 3 'cm'] properly includes 2 'cm'", "true"},
		{"Interval[1 'cm', 3 'cm'] properly includes 1 'cm'", "false"},
		{"Interval[@2020-01, @2020-05] properly includes @2020-03", "true"},
		// The retry the narrow question does guard: a precision names one, and it
		// still happens.
		{"Interval[@2020-01-01T10:00:00, @2020-05] properly includes day of @2020-03", "true"},
		{"Interval[1 'cm', 2 'cm'] includes Interval[1 'cm', 2 'cm']", "true"},
		{"Interval[1 'cm', 3 'cm'] union Interval[2 'cm', 4 'cm']", "Interval[1 'cm', 4 'cm']"},
		{"Interval[1 'cm', 3 'cm'] intersect Interval[2 'cm', 4 'cm']", "Interval[2 'cm', 3 'cm']"},
		// Nothing temporal moves: the reason funcs did not know is a reason it
		// still has to act on, and these are the shapes that exercise it.
		{"Interval[@2020-01-01, @2020-01-05] overlaps Interval[@2020-01-03, @2020-01-08]", "true"},
		{"Interval[@2020-01-01, @2020-01-05] union Interval[@2020-01-03, @2020-01-08]", "Interval[2020-01-01, 2020-01-08]"},
		{"Interval[1, 5] union Interval[3, 8]", "Interval[1, 8]"},
	} {
		if got := evalQuantityCompare(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}
