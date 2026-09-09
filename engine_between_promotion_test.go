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

// betweenOperands are values of the numeric types that promote to one another,
// written so every pair of them appears below in both orders.
var betweenOperands = []string{"100", "100L", "100.0"}

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
// The table is every combination of the three promoting numeric types across
// operand and both bounds, in both orders, checked against both expansions
// rather than against a hand-written expectation — the two spellings are the
// authority, and a hand-written table would be a fourth opinion to keep in sync.
func TestBetweenPromotesLikeTheTwoSpellingsItIsDefinedAs(t *testing.T) {
	var checked int
	for _, operand := range betweenOperands {
		for _, low := range betweenOperands {
			for _, high := range betweenOperands {
				// A pair that brackets the operand and one that does not, so a
				// rule that answered a constant would not survive the table.
				for _, shift := range []string{"", " + 50"} {
					lowExpr, highExpr := low+shift, high+shift
					between := evalOrDiagnostic(t,
						operand+" between "+lowExpr+" and "+highExpr)
					conjunction := evalOrDiagnostic(t,
						operand+" >= "+lowExpr+" and "+operand+" <= "+highExpr)
					inInterval := evalOrDiagnostic(t,
						operand+" in Interval["+lowExpr+", "+highExpr+"]")
					checked++
					if between != conjunction {
						t.Errorf("%s between %s and %s = %s, but the conjunction it is defined as = %s",
							operand, lowExpr, highExpr, between, conjunction)
					}
					if between != inInterval {
						t.Errorf("%s between %s and %s = %s, but `in Interval` = %s",
							operand, lowExpr, highExpr, between, inInterval)
					}
					if strings.HasPrefix(between, "ERROR") {
						t.Errorf("%s between %s and %s did not answer: %s",
							operand, lowExpr, highExpr, between)
					}
				}
			}
		}
	}
	// The table is worth nothing if it did not enumerate what it claims to.
	if want := len(betweenOperands) * len(betweenOperands) * len(betweenOperands) * 2; checked != want {
		t.Fatalf("the table checked %d combinations, want %d", checked, want)
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
