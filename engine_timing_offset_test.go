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

		// And a phrase with no offset still asks only its direction.
		{
			"no offset, before",
			iv("1999-06-01") + " ends on or before end of " + mp,
			"true",
		},

		// `less than` and `more than` are the strict forms, and the grammar keeps
		// them apart from `or less` / `or more` for that reason: at exactly the
		// bound, both are false where the inclusive pair would be true.
		{
			"less than, at the bound",
			"@2019-01-01T00:00:00 less than 10 days before @2019-01-11T00:00:00", "false",
		},
		{
			"more than, at the bound",
			"@2019-01-01T00:00:00 more than 10 days before @2019-01-11T00:00:00", "false",
		},
		{
			"less than, inside", "@2019-01-06T00:00:00 less than 10 days before @2019-01-11T00:00:00", "true",
		},
		{
			"more than, outside", "@2018-12-22T00:00:00 more than 10 days before @2019-01-11T00:00:00", "true",
		},

		// Every unit the point arithmetic takes, in both the bare and the UCUM
		// spelling. Reading one of these as "no offset" would put the phrase back
		// to answering from its direction alone.
		{
			"UCUM days", "@2019-01-01T00:00:00 10 'd' or less before @2019-06-01T00:00:00", "false",
		},
		{
			"minutes", "@2019-01-01T00:00:00 30 minutes or less before @2019-01-01T10:00:00", "false",
		},
		{
			"minutes, inside", "@2019-01-01T09:45:00 30 minutes or less before @2019-01-01T10:00:00", "true",
		},

		// A bound this cannot apply declines rather than answering from the
		// direction alone: a fractional offset cannot be applied to a point by
		// whole units, and saying "true" there would claim what the phrase does not.
		{
			"fractional offset", "@2019-01-01T00:00:00 1.5 days or less before @2019-06-01T00:00:00", "null",
		},

		// Time operands, which have the same arithmetic and were answering null.
		{
			"time, inside", "@T09:00:00 2 hours or less before @T10:00:00", "true",
		},
		{
			"time, outside", "@T01:00:00 2 hours or less before @T10:00:00", "false",
		},
		// A Time has no date to carry into, so shifting the bound off either end of
		// the day used to wrap around and read as the opposite direction: three
		// hours before 02:00 became 23:00, and 00:30 read as later than the bound.
		// The day's edge is the bound there — nothing precedes midnight — and the
		// same question spelled in minutes has to give the same answer.
		{
			"time, bound runs off the start", "@T00:30:00 3 hours or less before @T02:00:00", "true",
		},
		{
			"time, the same question in minutes", "@T00:30:00 90 minutes or less before @T02:00:00", "true",
		},

		// A unit finer than the value's own precision cannot be applied to it.
		// fhirpath promotes it — a second off a Date takes a whole day — which put
		// the bound on the value itself, and then `or less` and `or more` both held
		// for one pair. Declining is the answer; agreeing with both is not.
		{
			"unit finer than a Date, or less", "@2019-01-01 1 second or less before @2019-01-02", "null",
		},
		{
			"unit finer than a Date, or more", "@2019-01-01 1 second or more before @2019-01-02", "null",
		},
		{
			"unit finer than the stated precision", "@2019-01-01T 30 minutes or less before @2019-01-02T", "null",
		},

		// `10.0 days` is the same bound as `10 days` and is read as one.
		{
			"decimal that is a whole number", "@2019-01-06T00:00:00 10.0 days or less before @2019-01-11T00:00:00", "true",
		},
	} {
		if got := evalTiming(t, tt.expr); got != tt.want {
			t.Errorf("%s: %s = %s, want %s", tt.what, tt.expr, got, tt.want)
		}
	}
}

// TestABoundaryWordNamesWhichEndIsCompared covers the other half of what the
// phrase was dropping, which shows without any offset at all.
//
// `A before X` is decided by the end of A that faces X. `A starts before X` names
// the other one outright — and the word was being dropped, so the comparison used
// the facing end regardless:
//
//	Interval[@2019-01-05, @2019-01-20] starts before @2019-01-10   was false
//
// January 5th is before January 10th. The interval's *end* is not, and that is
// what was being compared. 19 phrases across the published libraries write
// `starts before`, `starts on or before` or `ends on or before`.
func TestABoundaryWordNamesWhichEndIsCompared(t *testing.T) {
	const iv = "Interval[@2019-01-05T00:00:00, @2019-01-20T00:00:00]"

	for _, tt := range []struct{ what, expr, want string }{
		{"starts before, and it does", iv + " starts before @2019-01-10T00:00:00", "true"},
		{"starts before, and it does not", iv + " starts before @2019-01-01T00:00:00", "false"},
		{"ends after, and it does", iv + " ends after @2019-01-10T00:00:00", "true"},
		{"ends after, and it does not", iv + " ends after @2019-02-01T00:00:00", "false"},
		{"starts on or before", iv + " starts on or before @2019-01-10T00:00:00", "true"},

		// Without a boundary word the facing end still decides, which is what the
		// bare relationship means and what this must not change.
		{"no boundary word, before", iv + " before @2019-02-01T00:00:00", "true"},
		{"no boundary word, not before", iv + " before @2019-01-10T00:00:00", "false"},

		// The word applies whichever shape the other operand has, and whichever
		// relationship the phrase uses. Reading it in one place and not the others
		// left the same phrase answering differently depending on how it was
		// written.
		{
			"starts before, against an interval",
			iv + " starts before Interval[@2019-01-10T00:00:00, @2019-01-30T00:00:00]", "true",
		},
		{
			"starts same or before, against a point",
			iv + " starts same or before @2019-01-10T00:00:00", "true",
		},
		{
			"starts same or before, and it does not",
			iv + " starts same or before @2019-01-01T00:00:00", "false",
		},
	} {
		if got := evalTiming(t, tt.expr); got != tt.want {
			t.Errorf("%s: %s = %s, want %s", tt.what, tt.expr, got, tt.want)
		}
	}
}
