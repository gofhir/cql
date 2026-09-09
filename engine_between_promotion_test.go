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
