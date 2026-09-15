package cql

import (
	"context"
	"strings"
	"testing"
)

// This file pins three refusals that are decisions rather than defects.
//
// All three sat on a list of "asserted, unfixed defects" in my own notes, and
// none of them belonged there. Checking each showed the engine is right to refuse
// — but only one of the four things on that list had a test saying so, which is
// exactly why the other three could be mistaken for defects. A policy nobody
// asserts is indistinguishable from a bug nobody caught, and it can be undone by
// accident with nothing to notice.
//
// Each test below states the reason and the alternative that was rejected, so
// that changing the decision means arguing with the reason rather than deleting a
// line that merely looked like an oversight.

func evalRefusal(t *testing.T, expr string) string {
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

// TestUCUMYearAndMonthAreRefusedInCalendarArithmetic covers the one refusal whose
// message teaches the rule, which is the standard a diagnostic here is held to.
//
// 'a' and 'mo' are UCUM codes for durations of definite quantity — an 'a' is
// 365.25 days, always. A calendar year is not: it is 365 or 366 depending on where
// you start. So `@2020-01-01 + 1 'a'` has no answer that is both correct and
// useful, and the engine refuses it by name rather than silently picking one of
// the two readings.
//
// The alternative, treating 'a' as a calendar year, would make
// `@2020-01-01 + 1 'a'` and `+ 1 year` agree and quietly lose 6 hours a year. The
// calendar keywords exist precisely so an author can say which they mean.
func TestUCUMYearAndMonthAreRefusedInCalendarArithmetic(t *testing.T) {
	for _, tt := range []struct{ expr, names string }{
		{"@2020-01-01 + 1 'a'", "year"},
		{"@2020-01-01 - 1 'mo'", "month"},
		{"@2020-01-01T00:00:00 + 1 'a'", "year"},
	} {
		got := evalRefusal(t, tt.expr)
		if !strings.Contains(got, "ERROR") {
			t.Errorf("%s = %s — it used to be refused; if that is now a decision, say "+
				"which reading was chosen and why 'a' and `year` may disagree by six hours",
				tt.expr, got)
		}
		// The message has to name the spelling that works, or it is half a
		// diagnostic: it tells the author what is wrong without what to write.
		if !strings.Contains(got, tt.names) {
			t.Errorf("the refusal for %s does not name %q as the alternative: %s",
				tt.expr, tt.names, got)
		}
	}

	// The calendar keywords do work, which is what makes the refusal a redirection
	// rather than a dead end.
	for _, tt := range []struct{ expr, want string }{
		{"@2020-01-01 + 1 year", "2021-01-01"},
		{"@2020-01-01 + 1 month", "2020-02-01"},
		{"@2020-01-01 + 1 'd'", "2020-01-02"},
	} {
		if got := evalRefusal(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}

	// And 'a' is a perfectly good unit away from the calendar: the refusal is
	// about adding one to a date, not about the unit existing.
	if got := evalRefusal(t, "1 'a' + 1 'a'"); got != "2 'a'" {
		t.Errorf("1 'a' + 1 'a' = %s, want 2 'a' — the refusal has spread beyond dates", got)
	}
}

// TestSortingQuantitiesOfDifferentDimensionsFails covers the one place this engine
// answers an undecidable comparison with a failure instead of null, and the reason
// it is the exception.
//
// Everywhere else, two quantities whose dimensions differ answer null: `1 'cm' <
// 1 's'` is null, and Min over the pair is null. A sort cannot do that. Its job is
// to produce an ordering, and there is no null ordering — the elements have to
// come out in some sequence, and any sequence it invented would be a claim about
// which of a centimeter and a second is larger.
//
// Ordering them as equal was the alternative, and it is the same mistake this
// repository already reverted for unorderable sort keys: they would land together
// and `Last(… sort by …)` would answer whichever the sort happened to compare
// last.
func TestSortingQuantitiesOfDifferentDimensionsFails(t *testing.T) {
	for _, expr := range []string{
		"First(({1 'cm', 1 's'}) X sort asc)",
		"({1 'cm', 1 's'}) X sort desc",
	} {
		got := evalRefusal(t, expr)
		if !strings.Contains(got, "ERROR") {
			t.Errorf("%s = %s — sorting across dimensions now answers. If that is "+
				"deliberate, it decides which of a centimeter and a second is larger",
				expr, got)
		}
		if !strings.Contains(got, "incompatible units") {
			t.Errorf("the refusal for %s does not say why: %s", expr, got)
		}
	}

	// The same pair, asked in the ways that have a null to give, does give it.
	// That contrast is the whole argument for the exception.
	for _, expr := range []string{"1 'cm' < 1 's'", "Min({1 'cm', 1 's'})", "1 'cm' = 1 's'"} {
		if got := evalRefusal(t, expr); got != "null" {
			t.Errorf("%s = %s, want null — only the sort is meant to fail", expr, got)
		}
	}

	// And sorting quantities that share a dimension works, so the refusal is about
	// the dimensions and not about quantities.
	if got := evalRefusal(t, "First(({1 'm', 1 'cm'}) X sort asc)"); got != "1 'cm'" {
		t.Errorf("sorting within one dimension = %s, want 1 'cm'", got)
	}
}

// TestASystemStringHasNoElements covers the last of them, which is not a policy at
// all but plain correctness, recorded because it spent a long time on a list of
// suspected defects.
//
// `.value` on a System.String is refused because a System.String has no elements.
// It reads like a defect from a distance because FHIR's own string primitive does
// have a `value` — but that is FHIR.string, a different type, and the diagnostic
// says which one it is looking at.
func TestASystemStringHasNoElements(t *testing.T) {
	for _, expr := range []string{"'abc'.value", "('abc').value", "('abc' as String).value"} {
		got := evalRefusal(t, expr)
		if !strings.Contains(got, "semantic error") {
			t.Errorf("%s = %s, want a diagnostic", expr, got)
		}
		if !strings.Contains(got, "System.String") {
			t.Errorf("the diagnostic for %s does not name the type it looked at: %s", expr, got)
		}
	}
}
