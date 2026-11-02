package cql

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// The uncertainty area of testdata/snapshots holds these cases against both
// reference engines. These are the tests that area cannot hold: the oracle of
// extremes over a grid, the normative examples at several evaluation offsets,
// and what stays an error. Plan 016.
//
// CQL 1.5.3, Language Semantics, Uncertainty: a comparison with an uncertainty
// "results in true if the comparison is true for all values in the uncertainty,
// false if it is false for all of them, and null otherwise".

// uncertainBase is an uncertainty written in CQL with the bounds the
// specification, or the suite, gives it. The bounds are stated here, in Go: they
// are the inputs of the oracle, and no expected value is read from the engine.
type uncertainBase struct {
	expr      string
	low, high int
}

var uncertainBases = []uncertainBase{
	// Language Semantics, Determining Difference and Duration.
	{"years between @2012 and @2014", 1, 2},
	{"years between @2014 and @2012", -2, -1},
	{"years between DateTime(2005) and DateTime(2010)", 4, 5}, // cql-tests: Interval[4, 5]
	{"days between @2012-01 and @2012-02", 1, 59},
	{"days between @2014-01-15 and @2014-02", 17, 44},
	{"days between @2017-08-07T17:00 and @2017-08-14T", 6, 7},
	{"months between DateTime(2005) and DateTime(2006, 7)", 6, 18},
}

// uncertainOperand is one operand of the grid: the CQL that builds it, and the
// integers it could be.
type uncertainOperand struct {
	expr      string
	low, high int
}

// uncertainOperands shifts and negates the bases, so that the grid has the same
// uncertainty at many places on the number line. A shift is an addition of the
// engine's, which another test holds; the bounds here are the arithmetic of Go.
func uncertainOperands(shifts ...int) []uncertainOperand {
	out := make([]uncertainOperand, 0, 2*len(shifts)*len(uncertainBases))
	for _, b := range uncertainBases {
		for _, k := range shifts {
			sign := "+"
			if k < 0 {
				sign = "-"
			}
			abs := k
			if abs < 0 {
				abs = -abs
			}
			out = append(out,
				uncertainOperand{fmt.Sprintf("((%s) %s %d)", b.expr, sign, abs), b.low + k, b.high + k},
				uncertainOperand{fmt.Sprintf("(-((%s) %s %d))", b.expr, sign, abs), -(b.high + k), -(b.low + k)})
		}
	}
	return out
}

// verdict is the answer by the definition: true if every value the uncertainty
// could be satisfies the relation, false if none does, null otherwise.
func verdict(low, high int, holds func(x int) bool) string {
	all, none := true, true
	for x := low; x <= high; x++ {
		if holds(x) {
			none = false
		} else {
			all = false
		}
	}
	return threeValued(all, none)
}

// verdict2 is the same for two uncertainties, over every pair of their values.
func verdict2(l1, h1, l2, h2 int, holds func(x, y int) bool) string {
	all, none := true, true
	for x := l1; x <= h1; x++ {
		for y := l2; y <= h2; y++ {
			if holds(x, y) {
				none = false
			} else {
				all = false
			}
		}
	}
	return threeValued(all, none)
}

func threeValued(all, none bool) string {
	switch {
	case all:
		return "true"
	case none:
		return "false"
	}
	return "null"
}

type uncertainCase struct{ expr, want string }

// evaluateCases evaluates each expression of the cases as a definition of a
// library, in chunks (the engine caps the length of a library's source), and
// reports each that differs from the expected answer. A chunk that fails as a
// whole is asked again one expression at a time, so the report names the
// expression that failed and not the chunk.
func evaluateCases(t *testing.T, engine *Engine, cases []uncertainCase) {
	t.Helper()
	const chunk = 120
	bad := 0
	for from := 0; from < len(cases); from += chunk {
		part := cases[from:min(from+chunk, len(cases))]
		var src strings.Builder
		src.WriteString("library T version '1.0'\n")
		for i, c := range part {
			fmt.Fprintf(&src, "define D%d: %s\n", i, c.expr)
		}
		got, err := engine.EvaluateLibrary(context.Background(), src.String(), nil, nil)
		for i, c := range part {
			var answer string
			if err != nil {
				v, verr := engine.EvaluateExpression(context.Background(),
					"library T version '1.0'\ndefine R: "+c.expr+"\n", "R", nil, nil)
				if verr != nil {
					answer = "error: " + verr.Error()
				} else {
					answer = valueString(v)
				}
			} else {
				answer = valueString(got[fmt.Sprintf("D%d", i)])
			}
			if answer != c.want {
				bad++
				if bad <= 40 {
					t.Errorf("%s = %s, want %s", c.expr, answer, c.want)
				}
			}
		}
	}
	if bad > 40 {
		t.Errorf("%d more expressions differ", bad-40)
	}
}

// TestAnUncertaintyIsAnsweredByItsEndpoints is the oracle of extremes. Over a
// grid of uncertainties, points, closed and open intervals and pairs of
// uncertainties, the expected answer is read off the integers each uncertainty
// could be, which this test enumerates itself: it does not evaluate CQL to find
// what to expect. An uncertainty is a point of unknown value, not an interval
// of points: `U between a and b` asks about each value it could be.
func TestAnUncertaintyIsAnsweredByItsEndpoints(t *testing.T) {
	var cases []uncertainCase
	add := func(expr, want string) { cases = append(cases, uncertainCase{expr, want}) }

	for _, u := range uncertainOperands(-2, 1) {
		low, high := u.low, u.high
		// A point against each side of the range, on each bound and inside.
		for _, a := range uniq(low-1, low, (low+high)/2, high, high+1) {
			p := fmt.Sprint(a)
			if a < 0 {
				p = fmt.Sprintf("(%d)", a)
			}
			for _, rel := range []struct {
				op    string
				holds func(x int) bool
			}{
				{"<", func(x int) bool { return x < a }},
				{"<=", func(x int) bool { return x <= a }},
				{">", func(x int) bool { return x > a }},
				{">=", func(x int) bool { return x >= a }},
				{"=", func(x int) bool { return x == a }},
				{"before", func(x int) bool { return x < a }},
				{"after", func(x int) bool { return x > a }},
				{"same or before", func(x int) bool { return x <= a }},
				{"same or after", func(x int) bool { return x >= a }},
				// No `same as`: the reference translator has no overload of it for
				// two Integers (plan 016).
			} {
				want := verdict(low, high, rel.holds)
				add(fmt.Sprintf("%s %s %s", u.expr, rel.op, p), want)
				// The point on the left reads the other way round.
				switch rel.op {
				case "<", "<=", ">", ">=":
					flipped := map[string]string{"<": ">", "<=": ">=", ">": "<", ">=": "<="}[rel.op]
					add(fmt.Sprintf("%s %s %s", p, flipped, u.expr), want)
				}
			}
		}
		// An interval, open and closed at each end. A bound is chosen so that the
		// interval holds an integer when it is open at both ends: Interval(a, a+1)
		// holds none, and `x > a and x < a + 1` is then an empty set the
		// three-valued reading does not see.
		for _, a := range uniq(low-1, low, low+1) {
			for _, b := range uniq(low+1, high-1, high, high+1) {
				if b < a+2 {
					continue
				}
				for _, lc := range []bool{true, false} {
					for _, hc := range []bool{true, false} {
						open, closeB := "[", "]"
						if !lc {
							open = "("
						}
						if !hc {
							closeB = ")"
						}
						interval := fmt.Sprintf("Interval%s%d, %d%s", open, a, b, closeB)
						if a < 0 || b < 0 {
							interval = fmt.Sprintf("Interval%s(%d), (%d)%s", open, a, b, closeB)
						}
						inside := func(x int) bool {
							return (x > a || lc && x == a) && (x < b || hc && x == b)
						}
						want := verdict(low, high, inside)
						add(fmt.Sprintf("%s in %s", u.expr, interval), want)
						add(fmt.Sprintf("%s during %s", u.expr, interval), want)
						add(fmt.Sprintf("%s included in %s", u.expr, interval), want)
						add(fmt.Sprintf("%s includes %s", interval, u.expr), want)
						add(fmt.Sprintf("%s contains %s", interval, u.expr), want)
						if lc && hc {
							add(fmt.Sprintf("%s between %s and %s", u.expr, paren(a), paren(b)), want)
						}
					}
				}
				// `properly between` is defined as `> a and < b` (CQL 1.5.3, Appendix B,
				// Between), so its oracle composes the two three-valued answers, and
				// is not the set of values strictly between: for [1, 2] in (1, 2) no
				// integer qualifies, and `> 1 and < 2` is still null.
				interior := and3(verdict(low, high, func(x int) bool { return x > a }),
					verdict(low, high, func(x int) bool { return x < b }))
				add(fmt.Sprintf("%s properly between %s and %s", u.expr, paren(a), paren(b)), interior)
			}
		}
	}

	// Two uncertainties: every pair of what each could be.
	for _, u := range uncertainOperands(-2, 1) {
		for _, v := range uncertainOperands(0) {
			for _, rel := range []struct {
				op    string
				holds func(x, y int) bool
			}{
				{"<", func(x, y int) bool { return x < y }},
				{"<=", func(x, y int) bool { return x <= y }},
				{">", func(x, y int) bool { return x > y }},
				{">=", func(x, y int) bool { return x >= y }},
				{"=", func(x, y int) bool { return x == y }},
			} {
				add(fmt.Sprintf("%s %s %s", u.expr, rel.op, v.expr),
					verdict2(u.low, u.high, v.low, v.high, rel.holds))
			}
		}
	}
	if len(cases) < 2000 {
		t.Fatalf("only %d cases; the grid is too small to prove anything", len(cases))
	}
	t.Logf("%d cases", len(cases))
	evaluateCases(t, NewEngine(WithEvaluationTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))), cases)
}

// and3 is `and` over the answers "true", "false" and "null".
func and3(a, b string) string {
	switch {
	case a == "false" || b == "false":
		return "false"
	case a == "null" || b == "null":
		return "null"
	}
	return "true"
}

func paren(n int) string {
	if n < 0 {
		return fmt.Sprintf("(%d)", n)
	}
	return fmt.Sprint(n)
}

func uniq(values ...int) []int {
	var out []int
	seen := map[int]bool{}
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// TestTheOperatorsOfAnUncertaintyByHand holds the cases the plan names, each
// with the reason it has its answer.
func TestTheOperatorsOfAnUncertaintyByHand(t *testing.T) {
	const (
		u12  = "(years between @2012 and @2014)"                   // [1, 2]
		u01  = "(years between @2011 and @2012)"                   // [0, 1]
		u17  = "(days between @2014-01-15 and @2014-02)"           // [17, 44]
		jan  = "(days between @2012-01 and @2012-02)"              // [1, 59]
		days = "(days between @2017-08-07T17:00 and @2017-08-14T)" // [6, 7]
	)
	cases := []uncertainCase{
		// Language Semantics, Uncertainty: "[17, 44] > 2 is true, > 50 is false,
		// > 20 is null".
		{u17 + " > 2", "true"},
		{u17 + " > 50", "false"},
		{u17 + " > 20", "null"},
		{u17 + " < 50", "true"},
		{u17 + " < 2", "false"},
		{u17 + " < 20", "null"},
		// between is `>= a and <= b` (the translator writes it so).
		{u12 + " between 0 and 5", "true"},
		{u12 + " between 2 and 5", "null"},
		{u12 + " between 3 and 5", "false"},
		{u12 + " between 0 and 1", "null"},
		{u12 + " between 1 and 2", "true"},
		{u12 + " properly between 0 and 3", "true"},
		{u12 + " properly between 1 and 3", "null"},
		{u12 + " properly between 0 and 2", "null"},
		// No integer lies strictly between 1 and 2, and still the answer is null:
		// `properly between` is `> and <`, each of them unknown for [1, 2].
		{u12 + " properly between 1 and 2", "null"},
		{"(1 + " + u12 + ") between 2 and 3", "true"},
		// Each operator of the membership family, with the uncertainty inside,
		// outside and astride each bound, closed and open.
		{u12 + " in Interval[0, 5]", "true"},
		{u12 + " in Interval[1, 2]", "true"},
		{u12 + " in Interval(1, 5]", "null"},
		{u12 + " in Interval[0, 2)", "null"},
		{u12 + " in Interval(0, 2)", "null"},
		{u12 + " in Interval[3, 5]", "false"},
		{u12 + " in Interval[-5, 0]", "false"},
		{u12 + " during Interval(0, 2)", "null"},
		{u12 + " during Interval(0, 3)", "true"},
		{u12 + " included in Interval[0, 2)", "null"},
		{u12 + " included in Interval[0, 3)", "true"},
		{"Interval[0, 2) includes " + u12, "null"},
		{"Interval[0, 3) includes " + u12, "true"},
		{"Interval[0, 2] contains " + u12, "true"},
		{"Interval[0, 1] contains " + u12, "null"},
		{"Interval[5, 9] contains " + u12, "false"},
		{u12 + " properly included in Interval[0, 5]", "true"},
		{"Interval[0, 5] properly includes " + u12, "true"},
		{u12 + " in Interval[1, 5]", "true"},
		// A point on each side of it.
		{u12 + " before 2", "null"},
		{u12 + " before 3", "true"},
		{u12 + " before 1", "false"},
		{u12 + " after 0", "true"},
		{u12 + " after 1", "null"},
		{u12 + " after 2", "false"},
		{u12 + " same or before 2", "true"},
		{u12 + " same or before 1", "null"},
		{u12 + " same or after 1", "true"},
		{u12 + " same or after 2", "null"},
		{"2 before " + u12, "false"},
		{"0 before " + u12, "true"},
		// Membership of a list is defined through `=` with each element (CQL 1.5.3,
		// Appendix B, In (list)), so the answer composes the per-element three-valued
		// results: [1, 2] in { 1, 2 } is null, not the true that "every value is a
		// member" would give.
		{u12 + " in { 1, 2 }", "null"},
		{u12 + " in { 5, 6 }", "false"},
		{u12 + " in { 2, 5 }", "null"},
		{"{ 1, 2 } contains " + u12, "null"},
		{"{ 5, 6 } contains " + u12, "false"},
		// A null element is no possibility for a value that is not null, as it is
		// not for a point (review of plan 016, round 1).
		{u12 + " in { 5, null }", "false"},
		// An uncertainty as a bound of between: `5 between U and 10` is `5 >= U
		// and 5 <= 10`, and `2 properly between 0 and U` is `2 > 0 and 2 < U`,
		// with U in {1, 2} (review of plan 016, round 1).
		{"5 between " + u12 + " and 10", "true"},
		{"2 between " + u12 + " and 10", "true"},
		{"1 between " + u12 + " and 10", "null"},
		{"2 properly between 0 and " + u12, "false"},
		{"1 properly between 0 and " + u12, "null"},
		// properly includes asks for the interior, and a list of one element has
		// no other element besides (review of plan 016, round 2).
		{"Interval[1, 3] properly includes " + u12, "null"},
		{"{ 1 } properly includes " + u12, "false"},
		// duration in X of an interval takes the evaluation's offset too, as the
		// normative example does with between (review of plan 016, round 2).
		{"duration in days of Interval[@2017-08-07T17:00, @2017-08-14T]", "Interval[6, 7]"},
		// Two uncertainties: true where every pair satisfies the relation.
		{u12 + " >= " + u01, "true"},
		{u01 + " <= " + u12, "true"},
		{u12 + " > " + u01, "null"},
		{u01 + " < " + u12, "null"},
		{u12 + " > (-(years between @2002 and @2003))", "true"},
		{u12 + " < (-(years between @2002 and @2003))", "false"},
		{u12 + " = " + u01, "null"},
		{u12 + " = (-(years between @2002 and @2003))", "false"},
		{u12 + " != " + u01, "null"},
		{jan + " > " + u12, "null"},
		{u17 + " > " + u12, "true"},
		// The normative examples (Language Semantics, Determining Difference
		// and Duration).
		{"days between @2014-01-15 and @2014-02", "Interval[17, 44]"},
		{days, "Interval[6, 7]"},
		{"days between @2012-01 and @2012-02", "Interval[1, 59]"},
		{"years between DateTime(2005) and DateTime(2010)", "Interval[4, 5]"},
		// Seconds and milliseconds are a single combined precision, so a value
		// to the second is exact to the millisecond (the suite's own example).
		{"seconds between @2012-01-01T01:00:00 and @2012-01-01T01:00:01.0", "1"},
		{"seconds between @2012-01-01T01:00:00.0 and @2012-01-01T01:00:01", "1"},
		{"seconds between @2012-01-01T01:00:00 and @2012-01-01T01:00:01", "1"},
		{"milliseconds between @2012-01-01T01:00:00 and @2012-01-01T01:00:01", "1000"},
		{"milliseconds between @2012-01-01T01:00:00.000 and @2012-01-01T01:00:01", "1000"},
		{"hours between @2012-01-01T01:00:00 and @2012-01-01T02:00:00.0", "1"},
		{"seconds between @T01:00:00 and @T01:00:01.0", "1"},
		// Below the second it is still uncertain.
		{"seconds between @2012-01-01T01:00 and @2012-01-01T01:01", "Interval[0, 119]"},
	}
	engine := NewEngine(WithEvaluationTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", -12*3600))))
	evaluateCases(t, engine, cases)
	// The same cases through the ELM an import reads back, the way a library of
	// ELM reaches the operators.
	for _, c := range cases {
		if got := evalImported(t, c.expr); got != c.want {
			t.Errorf("%s, through ELM = %s, want %s", c.expr, got, c.want)
		}
	}
}

// TestBetweenIsTheTwoComparisons is the metamorphic reading of the same, an
// internal coherence check: it reads the engine against itself, and the oracle
// above is the proof the answers are right.
func TestBetweenIsTheTwoComparisons(t *testing.T) {
	var cases []uncertainCase
	for _, u := range uncertainOperands(0, 1) {
		for _, a := range uniq(u.low-1, u.low, u.high) {
			for _, b := range uniq(u.low, u.high, u.high+1) {
				if b <= a {
					continue
				}
				pa, pb := paren(a), paren(b)
				cases = append(cases,
					uncertainCase{fmt.Sprintf("(%s between %s and %s) ~ (%s >= %s and %s <= %s)", u.expr, pa, pb, u.expr, pa, u.expr, pb), "true"},
					uncertainCase{fmt.Sprintf("(%s properly between %s and %s) ~ (%s > %s and %s < %s)", u.expr, pa, pb, u.expr, pa, u.expr, pb), "true"},
					uncertainCase{fmt.Sprintf("(%s in Interval[%s, %s)) ~ (%s >= %s and %s < %s)", u.expr, pa, pb, u.expr, pa, u.expr, pb), "true"})
			}
		}
	}
	evaluateCases(t, NewEngine(), cases)
}

// TestTheNormativeExamplesDoNotDependOnTheOffset evaluates the examples at the
// offsets a request can carry. A DateTime to the day has no offset of its own
// and the one to the minute sits at the evaluation's, so the two were measured
// in different frames: at -12:00, `days between @2017-08-07T17:00 and
// @2017-08-14T` was [5, 6]. Language Semantics gives [6, 7] whatever the offset.
func TestTheNormativeExamplesDoNotDependOnTheOffset(t *testing.T) {
	cases := []uncertainCase{
		{"days between @2017-08-07T17:00 and @2017-08-14T", "Interval[6, 7]"},
		{"days between @2017-08-07T and @2017-08-14T17:00", "Interval[6, 7]"},
		{"days between @2017-08-14T and @2017-08-07T17:00", "Interval[-7, -6]"},
		{"days between @2017-08-07T17:00 and @2017-08-14T23:59", "7"},
		{"days between @2017-08-07T17:00:00 and @2017-08-14T", "Interval[6, 7]"},
		{"days between @2012-01 and @2012-02", "Interval[1, 59]"},
		{"days between @2014-01-15 and @2014-02", "Interval[17, 44]"},
		{"years between DateTime(2005) and DateTime(2010)", "Interval[4, 5]"},
		{"months between DateTime(2005) and DateTime(2006, 7)", "Interval[6, 18]"},
		{"days between DateTime(2012, 1, 1) and DateTime(2012, 1, 3)", "Interval[1, 2]"},
		{"hours between @2012-01-01T01:00:00 and @2012-01-01T02:00:00.0", "1"},
		{"seconds between @2012-01-01T01:00:00 and @2012-01-01T01:00:01.0", "1"},
		{"(days between @2017-08-07T17:00 and @2017-08-14T) between 6 and 7", "true"},
		{"(days between @2017-08-07T17:00 and @2017-08-14T) > 5", "true"},
		{"(days between @2017-08-07T17:00 and @2017-08-14T) > 6", "null"},
		{"(days between @2017-08-07T17:00 and @2017-08-14T) > 7", "false"},
	}
	for _, offset := range []struct {
		name    string
		seconds int
	}{
		{"+14:00", 14 * 3600},
		{"+05:45", 5*3600 + 45*60},
		{"00:00", 0},
		{"-03:00", -3 * 3600},
		{"-12:00", -12 * 3600},
	} {
		t.Run(offset.name, func(t *testing.T) {
			engine := NewEngine(WithEvaluationTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", offset.seconds))))
			evaluateCases(t, engine, cases)
		})
	}
}

// TestACoarseDateTimeIsReadAtTheEvaluationOffset holds the frame of a DateTime to
// the day against a value that states an offset of its own. CQL 1.5.3: a
// DateTime with no offset is at the offset of the evaluation request, so the day
// of @2017-08-14T runs from 00:00 to 23:59:59.999 at the evaluation's offset, and
// not at the other operand's. With the evaluation at -03:00 that day starts at
// 03:00 UTC; @2017-08-07T07:59+05:00 is 02:59 UTC, whose latest instant,
// 02:59:59.999, is just short of seven days before, so the least is 7, and the
// greatest, to 02:59:59.999 UTC of the 15th, is 8. Read at UTC it would be
// [6, 7] and at +05:00, [6, 7] too.
func TestACoarseDateTimeIsReadAtTheEvaluationOffset(t *testing.T) {
	engine := NewEngine(WithEvaluationTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", -3*3600))))
	evaluateCases(t, engine, []uncertainCase{
		{"days between @2017-08-07T07:59+05:00 and @2017-08-14T", "Interval[7, 8]"},
		{"days between @2017-08-14T and @2017-08-07T07:59+05:00", "Interval[-8, -7]"},
	})
	engine = NewEngine(WithEvaluationTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", 5*3600))))
	evaluateCases(t, engine, []uncertainCase{
		// At +05:00 the day starts at 19:00 UTC of the 13th: [6, 7].
		{"days between @2017-08-07T07:59+05:00 and @2017-08-14T", "Interval[6, 7]"},
	})
}

// TestMinMaxAndSortOfAnUncertaintyStillRefuse fixes what the plan leaves out
// (plan 016, fuera del alcance): the specification defines no order over
// uncertainties for Min, Max or sort, and they answer with an error, not with a
// guess. A change here is a decision, not a side effect.
func TestMinMaxAndSortOfAnUncertaintyStillRefuse(t *testing.T) {
	const u = "(years between @2012 and @2014)"
	for _, expr := range []string{
		"Min({ " + u + ", 3 })",
		"Max({ " + u + ", 3 })",
		"Min({ 3, " + u + " })",
		"({ 3, " + u + " }) L return L sort asc",
		"({ 3, " + u + " }) L return L sort desc",
	} {
		if got := evalConversion(t, expr); !strings.HasPrefix(got, "error: ") {
			t.Errorf("%s = %s, want an error", expr, got)
		}
	}
}

// TestAnIntervalInAnIntervalWithAnUncertainBoundIsNotAPoint holds that an interval
// asked about an interval one of whose bounds is an uncertainty is not read as a
// point: that question is outside plan 016, and it must not answer null in
// silence (review of plan 016, round 2), on either path.
func TestAnIntervalInAnIntervalWithAnUncertainBoundIsNotAPoint(t *testing.T) {
	bound := "Interval[(years between @2012 and @2014), 5]"
	for _, expr := range []string{
		bound + " includes Interval[2, 3]",
		"Interval[2, 3] during " + bound,
		"Interval[2, 3] included in " + bound,
		"Interval[2, 3] in " + bound,
	} {
		if got := evalConversion(t, expr); got == "null" {
			t.Errorf("%s = null, read as a point", expr)
		}
		if got := evalImported(t, expr); got == "null" {
			t.Errorf("%s, through ELM = null, read as a point", expr)
		}
	}
}
