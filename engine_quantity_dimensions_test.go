package cql

import (
	"context"
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
		// A quantity against a bare number is not the case this rule is about, and
		// it keeps the answer it had: nothing here decided it, and deciding it is a
		// separate argument about what unit a bare number carries.
		{"1 'mg' = 1", "false"},
	} {
		if got := evalQuantityCompare(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestBetweenAgreesWithTheComparisonsItIsMadeOf covers the operator CQL defines
// in terms of the two this change fixed.
//
//	"If the first argument is greater than or equal to the low argument, and less
//	 than or equal to the high argument, the result is true, otherwise false."
//
// So `between` cannot answer where `>=` and `<=` decline, and it was raising an
// error where both of them now return null. It reaches the same place `in` and
// `contains` do — Interval.Contains — so the three are one question and now give
// one answer.
func TestBetweenAgreesWithTheComparisonsItIsMadeOf(t *testing.T) {
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

	// And an interval of one dimension still answers about points in it.
	for _, tt := range []struct{ expr, want string }{
		{"1 'cm' between 1 'cm' and 2 'cm'", "true"},
		{"1 'cm' in Interval[1 'cm', 2 'cm']", "true"},
		{"3 'cm' in Interval[1 'cm', 2 'cm']", "false"},
		{"Interval[1 'cm', 2 'cm'] overlaps Interval[2 'cm', 3 'cm']", "true"},
	} {
		if got := evalQuantityCompare(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestOperationsThatStillFailOnMixedDimensions records where the same rule is not
// applied yet, asserted so that applying it trips here.
//
// All of these raise "incompatible units" and take the whole define with them,
// where the comparison operators now answer null. They are left for one change
// rather than taken piecemeal, because what stands between them and the rule is
// structural rather than four separate oversights:
//
//	`isAmbiguousComparisonErr` — the predicate that says "this comparison could
//	not be made, so the answer is null" — exists TWICE, once in eval and once in
//	funcs, and about thirty call sites read one or the other. The two copies do
//	not even agree today: eval's matches the precision sentinel and the text of an
//	offset mismatch, funcs' matches the precision sentinel and the text "ambiguous
//	comparison".
//
// Adding a third reason to a rule that has two divergent implementations is the
// defect this engine has spent several changes removing, so the reason is added
// where each site can be pointed at it deliberately — which is what the sites in
// the change above are — and the rest wait for the predicate to become one.
//
// `types` is where it belongs: eval and funcs both import it, and it imports
// neither.
//
// Two of these are not comparisons at all and carry their own citation. CQL says
// of addition: "units of 'cm2' and 'cm' cannot be added… Attempting to operate on
// quantities with invalid or special units will result in a null." So `+` and `-`
// are the same rule in a different operator family, and Sum and Avg are built on
// them.
func TestOperationsThatStillFailOnMixedDimensions(t *testing.T) {
	for _, tt := range []struct{ expr, why string }{
		{"Interval[1 'cm', 2 'cm'] overlaps Interval[1 's', 2 's']",
			"reaches Interval.Overlaps through funcs, which has the other copy of the predicate"},
		{"Count(({1 'cm2', 1 'cm'}) X sort by X)",
			"sorting asks for an order that does not exist; see TestSortingByAnUnorderableKeyStillFails"},
		{"Min({1 'cm2', 1 'cm'})", "there is no minimum of two things that cannot be compared"},
		{"1 'cm2' - 1 'cm'", "arithmetic, which the specification also says is null"},
		{"Sum({1 'cm2', 1 'cm'})", "built on the addition above"},
	} {
		if got := evalQuantityCompare(t, tt.expr); !strings.HasPrefix(got, "ERROR") {
			t.Errorf("%s = %s, an answer rather than a failure now (%s) — remove it from "+
				"this test and cover it above", tt.expr, got, tt.why)
		}
	}
}
