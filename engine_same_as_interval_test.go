package cql

import (
	"context"
	"fmt"
	"testing"
)

func evalSameAs(t *testing.T, expr string) string {
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

// TestSameAsOnIntervalsIsEquality covers the operator CQL gives the same
// definition as `=` and which answered differently.
//
//	Interval[@2020-01, @2020-05] same as Interval[@2020-01-01T10:00:00, @2020-05]   false
//	Interval[@2020-01, @2020-05] =       Interval[@2020-01-01T10:00:00, @2020-05]   null
//
// It also disagreed with its own scalar spelling, where `@2020-01 same as
// @2020-01-01T10:00:00` is null. The cause is the shape v1.15.2 and v1.20.0 each
// removed one level up: the operator read `leftIv.Equal(rightIv)`, a bool with
// nowhere to say it could not decide, and so never reached containerEquality —
// which was sitting right there and already answered this correctly for `=`.
//
// The two are one question, and the specification says so twice. Interval
// equality is decided by the boundaries "as determined by the Start and End
// operators"; `same as` for intervals "returns true if the two intervals start
// and end at the same value, using the semantics described in the Start and End
// operators". With no precision named they are the same operator, so they now
// ask the same function.
func TestSameAsOnIntervalsIsEquality(t *testing.T) {
	// Every pair is asked both ways, because "they agree" is the claim and one
	// spelling answering correctly on its own is not it.
	for _, pair := range []struct{ left, right, want, why string }{
		{"Interval[@2020-01, @2020-05]", "Interval[@2020-01-01T10:00:00, @2020-05]", "null",
			"a month against a second: the bound cannot be compared"},
		{"Interval[1 'cm', 2 'cm']", "Interval[1 's', 2 's']", "null",
			"no unit both bounds can be stated in"},
		{"Interval[@2020-01, @2020-05]", "Interval[@2020-01, @2020-05]", "true", "identical"},
		{"Interval[@2020-01, @2020-05]", "Interval[@2020-02, @2020-05]", "false", "the low bound differs"},
		{"Interval[1, 5]", "Interval[1, 5]", "true", "identical integers"},
		{"Interval[1, 5]", "Interval[2, 5]", "false", "the low bound differs"},
		// Written differently and the same interval, which is what "as determined
		// by the Start and End operators" means for a discrete point type.
		{"Interval(0, 6)", "Interval[1, 5]", "true", "the same integers, open bounds"},
		{"Interval[1 'cm', 2 'cm']", "Interval[1 'cm', 2 'cm']", "true", "identical quantities"},
		// A Date bound against a DateTime one: implicitly converted, and then the
		// comparison runs to the finest precision either side states, which the
		// Date-derived bound does not reach.
		{"Interval[@2020-01-01, @2020-05-01]", "Interval[@2020-01-01T00:00:00, @2020-05-01]", "null",
			"a day against a second"},
		// Bounds that state different offsets, which is what FHIR data looks like
		// next to a measure's period.
		{"Interval[@2020-03-05T23:00:00Z, @2020-03-08T10:00:00Z]",
			"Interval[@2020-03-05T23:00:00+05:00, @2020-03-08T10:00:00Z]", "false",
			"23:00Z is not 18:00Z"},
	} {
		eq := evalSameAs(t, fmt.Sprintf("%s = %s", pair.left, pair.right))
		same := evalSameAs(t, fmt.Sprintf("%s same as %s", pair.left, pair.right))
		if eq != pair.want {
			t.Errorf("`=` over %s and %s = %s, want %s — %s", pair.left, pair.right, eq, pair.want, pair.why)
		}
		if same != eq {
			t.Errorf("one question, two answers:\n  %s =       %s = %s\n  %s same as %s = %s",
				pair.left, pair.right, eq, pair.left, pair.right, same)
		}
	}
}

// TestSameAsOnIntervalsHonoursItsPrecision covers the half of this that was a
// wrong answer rather than a fold.
//
//	Interval[@2020-01, @2020-05] same year as Interval[@2020-01-01T10:00:00, @2020-05]
//
// Both intervals begin in 2020 and end at the same value, so at year precision
// they are the same interval. It answered false, because the stated precision
// was not consulted at all — the operator compared the intervals whole and the
// word "year" changed nothing.
//
// That is the shape v1.20.2 removed from `in day of`: a precision is the
// precision of the operation, not a second attempt for when the first fails. The
// point spelling had it right all along, which is what made the two disagree.
func TestSameAsOnIntervalsHonoursItsPrecision(t *testing.T) {
	if got := evalSameAs(t, "@2020-01 same year as @2020-01-01T10:00:00"); got != "true" {
		t.Fatalf("the point spelling = %s, want true — this test's premise", got)
	}

	for _, tt := range []struct{ expr, want, why string }{
		{"Interval[@2020-01, @2020-05] same year as Interval[@2020-01-01T10:00:00, @2020-05]", "true",
			"both begin in 2020 and end at the same value"},
		{"Interval[@2020-01, @2020-05] same year as Interval[@2021-01-01T10:00:00, @2021-05]", "false",
			"2020 against 2021"},
		{"Interval[@2020-01, @2020-05] same month as Interval[@2020-01-01T10:00:00, @2020-05]", "true",
			"both begin in January"},
		{"Interval[@2020-01, @2020-05] same day as Interval[@2020-01-01T10:00:00, @2020-05]", "null",
			"the low bound has no day, and the specification says the result is null"},
		// A precision the boundaries cannot answer at is null, not false, at
		// whichever end runs out first.
		{"Interval[@2020-01-01, @2020-05-01] same second as Interval[@2020-01-01T10:00:00, @2020-05-01]", "null",
			"neither low bound states a second on both sides"},
	} {
		if got := evalSameAs(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — %s", tt.expr, got, tt.want, tt.why)
		}
	}

	// A precision means nothing for a boundary that is not temporal, and inventing
	// a reading of "year" for a quantity would put the two operators back into
	// disagreement.
	for _, tt := range []struct{ expr, want string }{
		{"Interval[1 'cm', 2 'cm'] same year as Interval[1 'cm', 2 'cm']", "true"},
		{"Interval[1 'cm', 2 'cm'] same year as Interval[1 's', 2 's']", "null"},
		{"Interval[1, 5] same year as Interval[1, 5]", "true"},
	} {
		if got := evalSameAs(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestEquivalenceOnIntervalsStillDecides is the limit, and the same one `~` has
// everywhere else: equivalence never returns null in CQL, so a pair `same as`
// declines is simply not equivalent.
func TestEquivalenceOnIntervalsStillDecides(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"Interval[@2020-01, @2020-05] ~ Interval[@2020-01-01T10:00:00, @2020-05]", "false"},
		{"Interval[1 'cm', 2 'cm'] ~ Interval[1 's', 2 's']", "false"},
		{"Interval[@2020-01, @2020-05] ~ Interval[@2020-01, @2020-05]", "true"},
	} {
		if got := evalSameAs(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — equivalence has no null to return", tt.expr, got, tt.want)
		}
	}
}

// TestSameAsOnIntervalsNormalisesOffsetsWhereTheSpecificationSaysTo covers what
// this change made zone-dependent, which the operator was not before.
//
// Reading `leftIv.Equal(rightIv)` made the answer the same everywhere because it
// never consulted a frame. Delegating to the point operator brings in CQL's rule,
// and the rule is conditional:
//
//	"When comparing DateTime values with different timezone offsets, the values
//	 are normalized to the timezone offset of the evaluation request timestamp,
//	 but only when the comparison precision is hours, minutes, seconds, or
//	 milliseconds."
//
// So `same hour as` over 23:00Z and 23:00+05:00 is false — they are 23:00 and
// 18:00 once placed — while `same day as` over the same pair is true, since both
// land on the 5th and the day precision does not normalize.
//
// The answer must not depend on where the process runs. Both values state an
// offset, so there is nothing for the request's zone to supply, and this repository
// has twice shipped a change that was right at UTC and wrong at an extreme. The
// Zones job runs this at +14, -11 and +05:45; these are the assertions that make
// running it there mean something.
func TestSameAsOnIntervalsNormalisesOffsetsWhereTheSpecificationSaysTo(t *testing.T) {
	const (
		a = "Interval[@2020-03-05T23:00:00Z, @2020-03-08T10:00:00Z]"
		b = "Interval[@2020-03-05T23:00:00+05:00, @2020-03-08T10:00:00Z]"
	)
	for _, tt := range []struct{ precision, want, why string }{
		{"hour", "false", "23:00Z against 18:00Z once both are placed"},
		{"minute", "false", "the same, one precision finer"},
		{"day", "true", "both land on the 5th, and a day does not normalize"},
		{"year", "true", "both are 2020"},
	} {
		expr := fmt.Sprintf("%s same %s as %s", a, tt.precision, b)
		if got := evalSameAs(t, expr); got != tt.want {
			t.Errorf("same %s as = %s, want %s — %s", tt.precision, got, tt.want, tt.why)
		}
	}

	// And with no precision named it agrees with `=` on the same pair, which is
	// the whole claim of this change.
	eq := evalSameAs(t, a+" = "+b)
	same := evalSameAs(t, a+" same as "+b)
	if eq != "false" || same != eq {
		t.Errorf("`=` = %s and `same as` = %s, want both false", eq, same)
	}
}
