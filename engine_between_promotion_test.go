package cql

import (
	"context"
	"strings"
	"testing"
)

// evalOrDiagnostic answers what an expression evaluates to, or the diagnostic
// that stopped it. Both are answers here: half of what this file pins is that a
// mismatch is still reported, and reported against the operand that causes it.
func evalOrDiagnostic(t *testing.T, expr string) string {
	t.Helper()
	src := "library T version '1.0'\ndefine X: " + expr + "\n"
	got, err := NewEngine().EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	if got == nil {
		return "null"
	}
	return got.String()
}

// numericSpellings writes one number in each of the three numeric types that
// promote to one another, so a table can vary the type of a value independently
// of the value itself.
func numericSpellings(n string) []string {
	return []string{n, n + "L", n + ".0"}
}

// TestBetweenPromotesLikeTheTwoSpellingsItIsDefinedAs is the whole of the rule.
//
// `between` is `>= low and <= high`, and the specification defines it by that
// expansion. The engine typed it by a different rule: the bounds had to convert
// to the *operand's* type, so an integer measured against decimal bounds was
// refused outright while both spellings of the same question answered it.
//
//	150 between 100.0 and 200.0      expected System.Integer, got System.Decimal
//	150 >= 100.0 and 150 <= 200.0    true
//	150 in Interval[100.0, 200.0]    true
//
// Interval[] is what shows the correct rule: it takes the Common of its two
// bounds, so the interval `in` builds is over decimals and the integer is
// compared against that. `between` now puts the operand in the same pot.
//
// The table is every combination of the three promoting numeric types across the
// operand and both bounds, over a range whose two ends are genuinely different
// values and with the operand once inside it and once outside — so a rule that
// only held at a boundary, or that answered a constant, would not survive.
//
// Each row is checked against both expansions rather than against a hand-written
// expectation: the two spellings are the authority here, and a written-out table
// would be a fourth opinion to keep in sync with them.
func TestBetweenPromotesLikeTheTwoSpellingsItIsDefinedAs(t *testing.T) {
	var checked, answeredTrue, answeredFalse int
	for _, operand := range append(numericSpellings("150"), numericSpellings("250")...) {
		for _, low := range numericSpellings("100") {
			for _, high := range numericSpellings("200") {
				between := evalOrDiagnostic(t, operand+" between "+low+" and "+high)
				conjunction := evalOrDiagnostic(t,
					operand+" >= "+low+" and "+operand+" <= "+high)
				inInterval := evalOrDiagnostic(t, operand+" in Interval["+low+", "+high+"]")
				checked++
				switch between {
				case "true":
					answeredTrue++
				case "false":
					answeredFalse++
				default:
					t.Errorf("%s between %s and %s did not answer: %s",
						operand, low, high, between)
				}
				if between != conjunction {
					t.Errorf("%s between %s and %s = %s, but the conjunction it is defined as = %s",
						operand, low, high, between, conjunction)
				}
				if between != inInterval {
					t.Errorf("%s between %s and %s = %s, but `in Interval` = %s",
						operand, low, high, between, inInterval)
				}
			}
		}
	}
	// The table is worth nothing if it did not enumerate what it claims to, and
	// worth little if every row landed on the same answer. Both counts are derived
	// from the spellings rather than written down, so adding a numeric type to
	// numericSpellings widens the table instead of breaking these two lines.
	perHalf := len(numericSpellings("")) * len(numericSpellings("")) * len(numericSpellings(""))
	if checked != 2*perHalf {
		t.Fatalf("the table checked %d combinations, want %d", checked, 2*perHalf)
	}
	if answeredTrue != perHalf || answeredFalse != perHalf {
		t.Fatalf("the table answered true %d times and false %d, want %d of each — "+
			"one half has the operand inside the range and the other outside",
			answeredTrue, answeredFalse, perHalf)
	}
}

// TestBetweenStillReportsWhatCannotBeCompared is the limit on the rule above.
//
// Widening what `between` accepts must not widen it to everything: a bound of a
// type the operand can never be compared against is still an error, and it is
// still reported by this phase rather than left for the evaluator.
//
// The second row is why the types are gathered one at a time instead of nested.
// Folding a pair that meets nowhere back into Common answers a Choice, and
// Common of that Choice with the remaining bound lands on the bound's own type —
// which both bounds then satisfy, so nothing was reported and the String reached
// the evaluator. A diagnostic had turned into a runtime error.
func TestBetweenStillReportsWhatCannotBeCompared(t *testing.T) {
	for _, tt := range []struct{ expr, wants string }{
		// The bound is what does not belong, and is what the message names.
		{"150 between 100.0 and 'abc'", "expected System.Decimal, got System.String"},
		// The operand is what does not belong. Neither bound meets it, so the
		// operand stands and the bounds are measured against it.
		{"'abc' between 100 and 200", "expected System.String, got System.Integer"},
		{"true between 1 and 2", "expected System.Boolean, got System.Integer"},
	} {
		got := evalOrDiagnostic(t, tt.expr)
		if !strings.Contains(got, tt.wants) {
			t.Errorf("%s = %s, want a diagnostic containing %q", tt.expr, got, tt.wants)
		}
		if !strings.Contains(got, "semantic error") {
			t.Errorf("%s failed at evaluation, not at check: %s", tt.expr, got)
		}
	}

	// Bounds that do not agree with each other are reported even when the operand
	// is not something this phase could type at all. That is a second diagnostic
	// where main gives one, and it is deliberate: `100` and `'abc'` cannot be the
	// two ends of anything, whoever is being tested against them. It is the same
	// call `between` over a choice already makes, where the operand names no
	// single type either and the bounds are still checked against each other.
	src := "library T version '1.0'\ndefine X: Undefined between 100 and 'abc'\n"
	_, err := NewEngine().EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err == nil {
		t.Fatal("an undefined operand between mismatched bounds reported nothing")
	}
	if n := strings.Count(err.Error(), "error: in X:"); n != 2 {
		t.Errorf("`Undefined between 100 and 'abc'` gave %d diagnostics, want 2 — "+
			"the undefined name and the two bounds that cannot bracket anything:\n%s",
			n, err.Error())
	}
}

// TestAReversedRangeFollowsTheConjunctionNotTheIntervalConstructor draws the
// limit on which of the two expansions is the authority when they disagree.
//
// They disagree in exactly one place, and a sweep of the nine numeric spellings
// across operand and both bounds found no other: where the bounds are the wrong
// way round, `between` and the conjunction both answer false, while
// `Interval[2, 1]` is refused by the constructor. All 729 rows agreed with the
// conjunction; the 243 that differed from `in Interval` were all this.
//
// The conjunction is what the specification defines `between` as, so it wins,
// and a promotion table that compared against `in Interval` over reversed bounds
// would be measuring the interval constructor instead. That constructor refusing
// a reversed pair, where CQL calls such an interval empty, is a separate matter
// and older than this.
func TestAReversedRangeFollowsTheConjunctionNotTheIntervalConstructor(t *testing.T) {
	for _, tt := range []struct{ low, high string }{
		{"200", "100"},
		{"200.0", "100"},
		{"200", "100.0"},
		{"200L", "100"},
	} {
		between := evalOrDiagnostic(t, "150 between "+tt.low+" and "+tt.high)
		conjunction := evalOrDiagnostic(t, "150 >= "+tt.low+" and 150 <= "+tt.high)
		if between != conjunction {
			t.Errorf("150 between %s and %s = %s, but the conjunction = %s",
				tt.low, tt.high, between, conjunction)
		}
		if between != "false" {
			t.Errorf("150 between %s and %s = %s, want false — a reversed range holds nothing",
				tt.low, tt.high, between)
		}
	}
}

// TestABareNumberAgainstQuantityBoundsFailsTheSameWayEverywhere covers where
// widening what `between` accepts moves a diagnostic from this phase to the
// evaluator, and why that is the point rather than a cost.
//
// Sema holds that an Integer converts to a Quantity, so Common of the two is
// Quantity and the bounds satisfy it — nothing is reported, and the bare number
// reaches the evaluator, which refuses to compare it. On main `between` reported
// it here instead. But the two spellings `between` is defined as never did:
//
//	150 >= 100 'mg' and 150 <= 200 'mg'    evaluation error, on main and here
//	150 in Interval[100 'mg', 200 'mg']    evaluation error, on main and here
//	150 between 100 'mg' and 200 'mg'      semantic error on main, evaluation error here
//
// So `between` was the odd one out and now is not.
//
// None of the three is right, though, and this test says so rather than blessing
// the agreement. Arithmetic in this same engine already applies the rule the
// specification states — "when a quantity has no units specified, it is treated
// as a quantity with the default unit ('1')" — so `1 '1' + 2` is `3 '1'` and
// `1 + 1 'cm'` is null, both pinned in engine_quantity_arithmetic_test.go.
// Comparison never got it, so one operator apart the same pair raises an error:
//
//	150 >= 100 '1'    should be true, is an error
//	150 >= 100 'mg'   should be null (different dimensions), is an error
//
// That is a defect of its own, older than any of this and squarely between the
// phases: sema declares the Integer-to-Quantity conversion, and the evaluator
// does not perform it. It is asserted here rather than described, so closing it
// breaks this test — and the three spellings are compared to each other so that
// whoever closes it finds out here if they move only one.
func TestABareNumberAgainstQuantityBoundsFailsTheSameWayEverywhere(t *testing.T) {
	for _, unit := range []string{"'mg'", "'1'"} {
		between := evalOrDiagnostic(t, "150 between 100 "+unit+" and 200 "+unit)
		conjunction := evalOrDiagnostic(t, "150 >= 100 "+unit+" and 150 <= 200 "+unit)
		inInterval := evalOrDiagnostic(t, "150 in Interval[100 "+unit+", 200 "+unit+"]")
		if between != conjunction || between != inInterval {
			t.Errorf("with bounds in %s the three spellings disagree:\n  between     %s\n  conjunction %s\n  in Interval %s",
				unit, between, conjunction, inInterval)
		}
		if !strings.Contains(between, "ERROR") {
			t.Errorf("150 between 100 %s and 200 %s now answers %s. If comparison learned "+
				"the default unit, this assertion has done its job: the answers to want are "+
				"true for '1' and null for 'mg', and all three spellings have to give them.",
				unit, unit, between)
		}
	}
}

// TestBetweenPromotionLeavesTheRestOfTheOperatorAlone pins the parts a change to
// how the operand and its bounds are typed could have moved without any of the
// above noticing: the boundaries, the ordering of the bounds, and the types that
// do not promote at all but were always accepted.
func TestBetweenPromotionLeavesTheRestOfTheOperatorAlone(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		// `properly` still excludes the bounds, at a promoted type as at a plain one.
		{"100 between 100.0 and 200.0", "true"},
		{"100 properly between 100.0 and 200.0", "false"},
		{"100 properly between 99.9 and 200.0", "true"},
		// A reversed pair is empty, not an error.
		{"150 between 200.0 and 100", "false"},
		{"150 between 200 and 100.0", "false"},
		// Quantities promote by their own rule and keep it.
		{"150 'mg' between 100.0 'mg' and 200 'mg'", "true"},
		{"150 'mg' between 100 'mg' and 200 'mg'", "true"},
		// A dimension the bounds are not stated in is unanswerable, not false.
		{"150 'mg' between 100 's' and 200 's'", "null"},
		// Temporals never promoted through this path and still do not need to.
		{"@2020-06-15 between @2020-01-01 and @2020-12-31", "true"},
		{"@2020-06-15 between @2021-01-01 and @2021-12-31", "false"},
		// Strings order among themselves.
		{"'m' between 'a' and 'z'", "true"},
		// A null bound leaves the question unanswerable.
		{"150 between null and 200.0", "null"},
	} {
		if got := evalOrDiagnostic(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}
