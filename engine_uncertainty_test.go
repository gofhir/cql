package cql

import (
	"testing"
)

// TestUncertaintyEquality is the bug that motivated giving uncertainties a type
// of their own.
//
// `months between DateTime(2005) and DateTime(2006, 7)` is somewhere in [6, 18]:
// 2005 names a year and 2006-07 names a month, so CQL declines to pick a number
// and hands back an uncertainty.
//
// Asking whether that equals 24 has an answer — no, 24 is not in the range — and
// the conformance corpus states it: DateTimeDurationBetweenMonthUncertain5
// expects false. Asking whether it equals 10 does not have one, because it might.
// The engine used to answer false to both, since an uncertainty was an interval
// and an interval is not a number. That is how `where duration in days of
// E.period = 1` came to silently match nothing when the period was imprecise.
func TestUncertaintyEquality(t *testing.T) {
	const uncertain = "months between DateTime(2005) and DateTime(2006, 7)"

	cases := []struct {
		expression string
		want       string // "null" for unknown, else the boolean
	}{
		// Outside the range: knowable.
		{uncertain + " = 24", "false"},
		{uncertain + " = 3", "false"},
		{uncertain + " != 24", "true"},
		// Inside the range: it might be, so there is no answer.
		{uncertain + " = 10", "null"},
		{uncertain + " != 10", "null"},
		// The bounds are possibilities too, so they are inside.
		{uncertain + " = 6", "null"},
		{uncertain + " = 18", "null"},
		// With the scalar on the left, same question.
		{"10 = " + uncertain, "null"},
		{"24 = " + uncertain, "false"},
		// An interval an author wrote is a different thing: it is not a number,
		// and saying so is not a guess.
		{"Interval[6, 18] = 10", "false"},
		{"Interval[6, 18] != 10", "true"},
	}

	for _, c := range cases {
		t.Run(c.expression, func(t *testing.T) {
			got, err := evalCQL(t, c.expression)
			if err != nil {
				t.Fatalf("%s: %v", c.expression, err)
			}
			if c.want == "null" {
				if got != nil {
					t.Errorf("%s = %v, want null", c.expression, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("%s = null, want %s", c.expression, c.want)
			}
			if got.String() != c.want {
				t.Errorf("%s = %s, want %s", c.expression, got.String(), c.want)
			}
		})
	}
}

// TestUncertaintyEqualsUncertainty covers the two-uncertainty case, which the
// corpus states directly: DateTimeDurationBetweenUncertainInterval2 expects
// `Interval[4, 16]` for the neighboring month, so the range itself is a value.
func TestUncertaintyEqualsUncertainty(t *testing.T) {
	const uncertain = "months between DateTime(2005) and DateTime(2006, 7)"

	// The same uncertainty twice: the ranges coincide, but neither is pinned
	// down, so whether the two values are equal is still unknown.
	got, err := evalCQL(t, uncertain+" = "+uncertain)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if got != nil {
		t.Errorf("U = U is %v, want null", got)
	}

	// Two uncertainties that cannot overlap are knowably different.
	other := "months between DateTime(2005) and DateTime(2010, 7)"
	got, err = evalCQL(t, uncertain+" = "+other)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if got == nil || got.String() != "false" {
		t.Errorf("disjoint uncertainties compare %v, want false", got)
	}
}

// TestUncertaintyAggregates is the other half of what giving uncertainties a type
// bought, and the reason PR #52 was not merged as it stood.
//
// Sum over a collection of uncertainties answered 0 — not null, a number, in an
// operation a measure runs on every patient. It came from reading each element
// through a private toDecimal that reports zero for anything that is not an
// Integer or a Decimal. The aggregates over Quantity had the same defect and were
// fixed the same way: stop guessing, and ask the operator that already knows.
//
// The specification defines `+` and `/` on uncertainties — the conformance corpus
// states `U + U` as `Interval[32, 88]` — so Sum and Avg have a rule to follow. It
// defines no aggregate over uncertainties at all, so the rest have none, and an
// error naming that is worth more than a plausible number.
func TestUncertaintyAggregates(t *testing.T) {
	// [6, 18], per DateTimeDurationBetweenUncertainInterval2's neighboring case.
	const U = "months between DateTime(2005) and DateTime(2006, 7)"

	t.Run("Sum delegates to +", func(t *testing.T) {
		got, err := evalCQL(t, "Sum({"+U+", "+U+"})")
		if err != nil {
			t.Fatalf("%v", err)
		}
		if got == nil || got.String() != "Interval[12, 36]" {
			t.Errorf("Sum({U, U}) = %v, want Interval[12, 36]", got)
		}
	})

	t.Run("Avg divides the sum", func(t *testing.T) {
		got, err := evalCQL(t, "Avg({"+U+", "+U+"})")
		if err != nil {
			t.Fatalf("%v", err)
		}
		if got == nil || got.String() != "Interval[6, 18]" {
			t.Errorf("Avg({U, U}) = %v, want Interval[6, 18]", got)
		}
	})

	t.Run("a single uncertainty sums to itself", func(t *testing.T) {
		got, err := evalCQL(t, "Sum({"+U+"})")
		if err != nil {
			t.Fatalf("%v", err)
		}
		if got == nil || got.String() != "Interval[6, 18]" {
			t.Errorf("Sum({U}) = %v, want Interval[6, 18]", got)
		}
	})

	// Mixing an uncertainty with a number is refused for the same reason mixing a
	// Quantity with one is: the collection cannot say what to do with either.
	t.Run("mixing is refused", func(t *testing.T) {
		if _, err := evalCQL(t, "Sum({"+U+", 3})"); err == nil {
			t.Error("Sum({U, 3}) was answered, want an error")
		}
	})

	// The aggregates the specification says nothing about. Each of these answered
	// 0 or a number derived from 0.
	for _, name := range []string{
		"Product", "Median", "Variance", "StdDev",
		"PopulationVariance", "PopulationStdDev", "GeometricMean",
	} {
		t.Run(name+" declines", func(t *testing.T) {
			expr := name + "({" + U + ", " + U + "})"
			got, err := evalCQL(t, expr)
			if err != nil {
				return // an error naming the gap is the answer wanted
			}
			if got != nil {
				t.Errorf("%s = %s, want an error or null, not a number", expr, got.String())
			}
		})
	}
}

// TestUncertaintyIsStillARange guards what giving uncertainties their own type
// nearly cost. `days between @2014-01-15 and @2014-02` used to be a
// cqltypes.Interval, so every operator that read one by type assertion read it —
// and after the change they silently stopped.
//
// Three did: `overlaps` and `included in` began reporting "cannot compare
// Interval", because an unrecognized uncertainty reads as a single point and the
// interval was asked to compare against it; `expand` returned the empty list; and
// the cast to Interval returned null. None of these was covered, which is why
// nothing failed. They are covered here.
//
// The point of the type was to make `=` answerable, not to stop an uncertainty
// being a range. It is still a range wherever an operator asks about the range.
func TestUncertaintyIsStillARange(t *testing.T) {
	const U = "(days between @2014-01-15 and @2014-02)" // Interval[17, 44]

	for _, tt := range []struct{ expr, want string }{
		{U + " overlaps Interval[10, 30]", "true"},
		{U + " overlaps Interval[100, 200]", "false"},
		{U + " included in Interval[0, 100]", "true"},
		{"Interval[20, 25] included in " + U, "true"},

		// The cast is the way to ask for the bounds as a plain interval, which
		// the conformance corpus notes CQL otherwise gives no way to select. It
		// compares as an interval afterwards, uncertainty discarded on request.
		{"(" + U + " as Interval<Integer>) = Interval[17, 44]", "true"},
	} {
		got, err := evalCQL(t, tt.expr)
		if err != nil {
			t.Errorf("%s: %v", tt.expr, err)
			continue
		}
		if got == nil || got.String() != tt.want {
			t.Errorf("%s = %v, want %s", tt.expr, got, tt.want)
		}
	}

	// expand walks the possibilities, all 28 of them.
	got, err := evalCQL(t, "Count(expand "+U+")")
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if got == nil || got.String() != "28" {
		t.Errorf("Count(expand U) = %v, want 28", got)
	}

	// The ordered comparisons, which the conformance corpus already covers, and
	// which have to keep agreeing with everything above.
	for _, tt := range []struct{ expr, want string }{
		{U + " > 10", "true"},
		{U + " < 100", "true"},
		{U + " > 20", "null"},
	} {
		got, err := evalCQL(t, tt.expr)
		if err != nil {
			t.Errorf("%s: %v", tt.expr, err)
			continue
		}
		if tt.want == "null" {
			if got != nil {
				t.Errorf("%s = %v, want null", tt.expr, got)
			}
			continue
		}
		if got == nil || got.String() != tt.want {
			t.Errorf("%s = %v, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestMinMaxDeclineOverUncertainties covers the defect that justified Median
// declining, found by asking whether Median could have picked a middle at all.
//
// It could not, and Min and Max were already answering as though it could:
//
//	A = [17, 44]   B = [22, 49]   C = [12, 39]
//	Min({A, B, C}) was Interval[17, 44]   — A, the first element
//	Max({A, B, C}) was Interval[17, 44]   — A again, the same answer
//	Min({C, B, A}) was Interval[12, 39]   — C, because C came first
//
// Min and Max walk the collection asking each value to Compare, and an
// uncertainty is not Comparable, so every comparison was skipped and the first
// element survived as the answer. Not a wrong ordering: no ordering, plus a
// plausible value that changes when the input is reordered.
//
// There is no ordering to be had. `A < B` is null, because the ranges overlap and
// which is smaller is precisely what CQL could not determine. An aggregate that
// needs an order over values that have none has no answer to give, so these say
// so — as Median, Variance and the rest already do.
func TestMinMaxDeclineOverUncertainties(t *testing.T) {
	const (
		a = "(days between @2014-01-15 and @2014-02)" // [17, 44]
		b = "(days between @2014-01-10 and @2014-02)" // [22, 49]
		c = "(days between @2014-01-20 and @2014-02)" // [12, 39]
	)

	// The premise: the ranges overlap, so neither is knowably the smaller.
	for _, expr := range []string{a + " < " + b, a + " < " + c, b + " < " + c} {
		got, err := evalCQL(t, expr)
		if err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
		if got != nil {
			t.Fatalf("%s = %v, want null — the test's premise is that these do not order", expr, got)
		}
	}

	for _, expr := range []string{
		"Min({" + a + ", " + b + ", " + c + "})",
		"Max({" + a + ", " + b + ", " + c + "})",
		"Min({" + c + ", " + b + ", " + a + "})",
		"Max({" + c + ", " + b + ", " + a + "})",
	} {
		got, err := evalCQL(t, expr)
		if err != nil {
			continue // declining is the answer wanted
		}
		t.Errorf("%s = %v, want an error: there is no order over these", expr, got)
	}

	// An interval an author wrote had the same defect, and the fix is the same
	// rule rather than a special case for uncertainties: the specification lists
	// the types Min and Max are defined over — Integer, Long, Decimal, Quantity,
	// Date, DateTime, Time, String — and an interval is not among them.
	// `Min({Interval[5, 9], Interval[1, 3]})` used to answer Interval[5, 9], the
	// first element.
	if got, err := evalCQL(t, "Min({Interval[5, 9], Interval[1, 3]})"); err == nil {
		t.Errorf("Min over intervals = %v, want an error", got)
	}

	// The types the specification does define them over are untouched.
	for _, tt := range []struct{ expr, want string }{
		{"Min({3, 1, 2})", "1"},
		{"Max({3, 1, 2})", "3"},
		{"Min({1 'mg', 3 'mg'})", "1 'mg'"},
		{"Max({@2014-01-01, @2014-06-01})", "2014-06-01"},
		{"Min({'b', 'a'})", "a"},
	} {
		got, err := evalCQL(t, tt.expr)
		if err != nil {
			t.Errorf("%s: %v", tt.expr, err)
			continue
		}
		if got == nil || got.String() != tt.want {
			t.Errorf("%s = %v, want %s", tt.expr, got, tt.want)
		}
	}
}
