package cql

import (
	"context"
	"testing"
	"time"
)

// evalTiming evaluates at a fixed offset, so a phrase measured in years does not
// depend on where the test runs.
func evalTiming(t *testing.T, expr string) string {
	t.Helper()
	got, err := NewEngine(WithEvaluationTimestamp(
		time.Date(2019, 6, 1, 12, 0, 0, 0, time.UTC))).
		EvaluateExpression(context.Background(),
			"library T version '1.0'\ndefine A: "+expr+"\n", "A", nil, nil)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	if got == nil {
		return "null"
	}
	return got.String()
}

// TestATimingPhraseHonoursItsQuantityOffset covers a phrase that was read as its
// direction alone.
//
// CQL writes `A ends 10 years or less on or before B`, and the builder kept only
// the "on or before". The quantity and the `starts|ends|occurs` it applies to were
// dropped on the floor, so the phrase asked nothing the bare relationship did not:
//
//	Interval[@1999-06-01, …] ends 10 years or less on or before end of MP
//	  was true — twenty years, under a bound of ten
//
// That is how ColorectalCancerScreeningsFHIR decides its numerator: a colonoscopy
// counts as screening if it ended within ten years of the measurement period.
// Every date in the past qualified, so the measure reported anyone who had ever
// had one.
//
// The bound is a point, not a length of time. `10 years or less before X` means
// the value sits in [X - 10 years, X], and measuring the gap instead reads it as a
// truncated count — 2009-12-30 against a period ending 2019-12-31 is ten years and
// a day, which `years between` reports as 10 and would let through. The measure's
// own published test case turns on exactly that day, which is what caught the
// first attempt at this.
//
// 22 phrases across 9 of the 19 published libraries state an offset, including
// MATGlobalCommonFunctions, which every measure includes.
func TestATimingPhraseHonoursItsQuantityOffset(t *testing.T) {
	const mp = "Interval[@2019-01-01T00:00:00.0, @2020-01-01T00:00:00.0)"
	iv := func(date string) string {
		return "Interval[@" + date + "T12:00:00, @" + date + "T13:00:00]"
	}

	for _, tt := range []struct{ what, expr, want string }{
		// The boundary the published test case is built on: ten years and one day
		// before the end of the period is outside a bound of ten years.
		{
			"a day past the bound",
			iv("2009-12-30") + " ends 10 years or less on or before end of " + mp,
			"false",
		},
		{
			"inside the bound",
			iv("2010-06-01") + " ends 10 years or less on or before end of " + mp,
			"true",
		},
		{
			"twice the bound",
			iv("1999-06-01") + " ends 10 years or less on or before end of " + mp,
			"false",
		},
		// The direction still has to hold: a bound of ten years does not admit a
		// value from the other side of the point.
		{
			"after the point entirely",
			iv("2030-06-01") + " ends 10 years or less on or before end of " + mp,
			"false",
		},

		// `or more` is the same bound read from the far side.
		{
			"or more, twice the bound",
			iv("1999-06-01") + " ends 10 years or more on or before end of " + mp,
			"true",
		},
		{
			"or more, inside the bound",
			iv("2018-06-01") + " ends 10 years or more on or before end of " + mp,
			"false",
		},

		// `starts` reads the other end of the same interval, which is the other
		// half of what the phrase was dropping.
		{
			"starts, inside",
			"Interval[@2018-06-01T12:00:00, @2019-06-01T12:00:00] starts 2 years or less on or before end of " + mp,
			"true",
		},
		{
			"starts, outside",
			"Interval[@2000-06-01T12:00:00, @2019-06-01T12:00:00] starts 2 years or less on or before end of " + mp,
			"false",
		},

		// A stated precision is the precision the order is read at, so two instants
		// half an hour apart are on the same day and the direction holds. This is
		// the conformance corpus's Issue32Interval, which the first attempt broke.
		{
			"precision decides the order",
			"Interval[@2017-12-20T10:30:00, @2017-12-20T12:00:00] starts 1 day or less on or after day of start of " +
				"Interval[@2017-12-20T11:00:00, @2017-12-21T21:00:00]",
			"true",
		},

		// And a phrase with no offset is untouched: it still asks only its
		// direction, which is what it says.
		{
			"no offset, before",
			iv("1999-06-01") + " ends on or before end of " + mp,
			"true",
		},
	} {
		if got := evalTiming(t, tt.expr); got != tt.want {
			t.Errorf("%s: %s = %s, want %s", tt.what, tt.expr, got, tt.want)
		}
	}
}
