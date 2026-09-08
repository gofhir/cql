package cql

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func evalWithin(t *testing.T, expr string) string {
	t.Helper()
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ndefine A: " + expr + "\n"
	got, err := NewEngine(WithEvaluationTimestamp(
		time.Date(2019, 6, 1, 12, 0, 0, 0, time.UTC))).
		EvaluateExpression(context.Background(), src, "A", nil, nil)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	if got == nil {
		return "null"
	}
	return got.String()
}

// TestWithinIsASymmetricRange covers the last timing phrase that did not read the
// quantity it states.
//
// CQL defines it as a range around the other operand, and says which range:
//
//	X starts within 3 days of start Y
//
//	"This expression returns true if the start of X is in the interval beginning
//	 three days before the start of Y and ending 3 days after the start of Y."
//
// So `A within Q of B` is membership in [B - Q, B + Q], both ends included. The
// builder marked the phrase and dropped the Quantity, and evaluation answered
// plain `during`, so the phrase meant nothing it stated:
//
//	@2019-06-01 within 2 months of @2019-07-01                    null, is true
//	@2019-01-01 within 2 months of @2019-07-01                    null, is false
//	Interval[@2019-06-01, …] starts within 2 months of @2019-07-01 false, is true
//
// The last is a wrong answer rather than a decline, and it is what the equivalence
// table found: the two spellings of one phrase disagreed. That table now carries
// `within` as a row of its own, which is what it asked for when this was left
// asserted instead of fixed.
func TestWithinIsASymmetricRange(t *testing.T) {
	const b = "@2019-07-01T00:00:00"
	for _, tt := range []struct{ expr, want, why string }{
		{"@2019-06-01T00:00:00 within 2 months of " + b, "true", "one month before, inside"},
		{"@2019-08-01T00:00:00 within 2 months of " + b, "true", "one month after: the range is symmetric"},
		{"@2019-01-01T00:00:00 within 2 months of " + b, "false", "six months before, outside"},
		{"@2019-12-01T00:00:00 within 2 months of " + b, "false", "five months after, outside"},
		// Both ends are included, which is what "beginning three days before and
		// ending three days after" says.
		{"@2019-05-01T00:00:00 within 2 months of " + b, "true", "exactly the low bound"},
		{"@2019-09-01T00:00:00 within 2 months of " + b, "true", "exactly the high bound"},
		{"@2019-04-30T00:00:00 within 2 months of " + b, "false", "a day past the low bound"},
		{"@2019-09-02T00:00:00 within 2 months of " + b, "false", "a day past the high bound"},
	} {
		if got := evalWithin(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — %s", tt.expr, got, tt.want, tt.why)
		}
	}
}

// TestWithinReadsTheEndItsBoundaryWordNames covers the spelling the published
// libraries would write, and the one whose disagreement with it surfaced this.
func TestWithinReadsTheEndItsBoundaryWordNames(t *testing.T) {
	const iv = "Interval[@2019-06-01T00:00:00, @2019-08-01T00:00:00]"
	const b = "@2019-07-01T00:00:00"

	// The boundary word and taking that end first are one thing said two ways.
	for _, pair := range [][2]string{
		{iv + " starts within 2 months of " + b, "start of " + iv + " within 2 months of " + b},
		{iv + " ends within 2 months of " + b, "end of " + iv + " within 2 months of " + b},
	} {
		a, c := evalWithin(t, pair[0]), evalWithin(t, pair[1])
		if a != c {
			t.Errorf("one name for one end:\n  %s = %s\n  %s = %s", pair[0], a, pair[1], c)
		}
		if a != "true" {
			t.Errorf("%s = %s, want true — both ends are within two months of July", pair[0], a)
		}
	}

	// An interval with no boundary word is asked whole: it has to fit in the range.
	for _, tt := range []struct{ expr, want string }{
		{iv + " within 2 months of " + b, "true"},
		{"Interval[@2019-01-01T00:00:00, @2019-08-01T00:00:00] within 2 months of " + b, "false"},
	} {
		if got := evalWithin(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}

	// And an interval on the right widens on both sides, measured from the near
	// end in each direction.
	if got := evalWithin(t, "@2019-04-15T00:00:00 within 1 month of Interval[@2019-05-01T00:00:00, @2019-09-01T00:00:00]"); got != "true" {
		t.Errorf("a point two weeks before the interval starts = %s, want true", got)
	}
	if got := evalWithin(t, "@2019-03-15T00:00:00 within 1 month of Interval[@2019-05-01T00:00:00, @2019-09-01T00:00:00]"); got != "false" {
		t.Errorf("a point six weeks before the interval starts = %s, want false", got)
	}
}

// TestWithinDeclinesARangeItCannotPlace is the limit, and it is the one the
// quantity-offset phrases already take: where the bound cannot be put on the
// value, the phrase declines rather than falling back to meaning nothing.
//
// Answering as though no range were stated is exactly what it used to do.
func TestWithinDeclinesARangeItCannotPlace(t *testing.T) {
	for _, tt := range []struct{ expr, want, why string }{
		// A unit finer than the value states: a Date has no second to move by, and
		// shifting it would move a whole day instead.
		{"@2019-06-01 within 30 seconds of @2019-07-01", "null", "a date has no second"},
		// A fractional offset cannot be applied to a point by whole units.
		{"@2019-06-01T00:00:00 within 1.5 months of @2019-07-01T00:00:00", "null", "not a whole number of units"},
	} {
		if got := evalWithin(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — %s", tt.expr, got, tt.want, tt.why)
		}
	}

	// A range over values that are not temporal is not this phrase's business, and
	// it says so by leaving the ordinary comparison to answer.
	if got := evalWithin(t, "3 within 2 of 4"); got == "ERROR" {
		t.Errorf("a non-temporal `within` failed: %s", got)
	}
}

// TestWithinReadsTheWholeRuleTheGrammarDeclares covers the parts of the phrase a
// first pass at this left unread. The rule is
//
//	('starts' | 'ends' | 'occurs')? 'properly'? 'within' quantity 'of' ('start' | 'end')?
//
// and only the quantity and the leading word were being read.
//
// `properly` is strict here as everywhere else in CQL, so the range excludes its
// own ends. The trailing word names which end of the *right* operand the range is
// measured from, which is the specification's own example of the phrase —
// `X starts within 3 days of start Y` — and ignoring it measured from whichever
// end each direction happened to reach.
func TestWithinReadsTheWholeRuleTheGrammarDeclares(t *testing.T) {
	const b = "@2019-07-01T00:00:00"
	for _, tt := range []struct{ expr, want, why string }{
		{"@2019-05-01T00:00:00 within 2 months of " + b, "true", "exactly the low bound, included"},
		{"@2019-05-01T00:00:00 properly within 2 months of " + b, "false", "and excluded when strict"},
		{"@2019-09-01T00:00:00 properly within 2 months of " + b, "false", "the high bound likewise"},
		{"@2019-06-01T00:00:00 properly within 2 months of " + b, "true", "strictly inside is still inside"},
	} {
		if got := evalWithin(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — %s", tt.expr, got, tt.want, tt.why)
		}
	}

	// The end of the right operand the phrase names.
	const y = "Interval[@2019-05-01T00:00:00, @2019-12-01T00:00:00]"
	for _, tt := range []struct{ expr, want, why string }{
		{"@2019-06-15T00:00:00 within 2 months of start " + y, "true", "six weeks after Y starts"},
		{"@2019-06-15T00:00:00 within 2 months of end " + y, "false", "five months before Y ends"},
		{"@2019-11-15T00:00:00 within 2 months of start " + y, "false", "six months after Y starts"},
		{"@2019-11-15T00:00:00 within 2 months of end " + y, "true", "two weeks before Y ends"},
		// Naming neither end measures from both, which is the widening.
		{"@2019-06-15T00:00:00 within 2 months of " + y, "true", "inside the interval itself"},
		{"@2019-04-01T00:00:00 within 2 months of " + y, "true", "a month before it starts"},
		{"@2019-02-01T00:00:00 within 2 months of " + y, "false", "three months before it starts"},
	} {
		if got := evalWithin(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — %s", tt.expr, got, tt.want, tt.why)
		}
	}
}

// TestOccursWithinIsTheBareSpelling covers the disagreement the equivalence table
// could not see until it was given data that tells the two apart.
//
// `occurs` is the default said aloud: it names no end, so it belongs with writing
// nothing rather than with `starts` and `ends`. Read as a boundary word, it took
// the interval's start where the bare spelling asked about the interval:
//
//	Interval[@2019-06-01, @2019-12-01] within 2 months of @2019-07-01         false
//	Interval[@2019-06-01, @2019-12-01] occurs within 2 months of @2019-07-01  true
//
// The table asserts this equivalence across 1248 pairs and passed anyway, because
// every interval on its left answered the same whether read whole or by its start.
// It has one that does not now.
func TestOccursWithinIsTheBareSpelling(t *testing.T) {
	const iv = "Interval[@2019-06-01T00:00:00, @2019-12-01T00:00:00]"
	const b = "@2019-07-01T00:00:00"

	bare := evalWithin(t, iv+" within 2 months of "+b)
	occurs := evalWithin(t, iv+" occurs within 2 months of "+b)
	if bare != occurs {
		t.Errorf("occurs is the default said aloud:\n  bare   = %s\n  occurs = %s", bare, occurs)
	}
	if bare != "false" {
		t.Errorf("the interval as a whole = %s, want false — it runs five months past the range", bare)
	}
	// And the start of it, asked for by name, is inside.
	if got := evalWithin(t, iv+" starts within 2 months of "+b); got != "true" {
		t.Errorf("its start = %s, want true", got)
	}
}

// TestWithinReadsBothSpellingsOfItsQuantity covers the two ways CQL writes a
// duration, which parseTimingOffset maps to one and which nothing else here
// checks: `2 months` and `2 'mo'` are the same range, and so are `60 days` and
// `60 'd'`.
//
// It matters because the UCUM spelling of a calendar duration is the one the
// engine refuses elsewhere — `@2019-01-01 + 1 'a'` reports that a UCUM year is
// not a calendar year — so a phrase reading it correctly is not automatic.
func TestWithinReadsBothSpellingsOfItsQuantity(t *testing.T) {
	const y = "Interval[@2019-05-01T00:00:00, @2019-12-01T00:00:00]"
	for _, pair := range [][2]string{
		{"2 months", "2 'mo'"},
		{"60 days", "60 'd'"},
		{"3 hours", "3 'h'"},
	} {
		for _, tail := range []string{"", " start", " end"} {
			a := evalWithin(t, fmt.Sprintf("@2019-06-15T00:00:00 within %s of%s %s", pair[0], tail, y))
			b := evalWithin(t, fmt.Sprintf("@2019-06-15T00:00:00 within %s of%s %s", pair[1], tail, y))
			if a != b {
				t.Errorf("`within %s of%s` = %s but `within %s of%s` = %s — one duration, two spellings",
					pair[0], tail, a, pair[1], tail, b)
			}
			if a == "ERROR" {
				t.Errorf("`within %s of%s` failed: %s", pair[0], tail, a)
			}
		}
	}
}
