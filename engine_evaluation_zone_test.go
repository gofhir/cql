package cql

import (
	"context"
	"testing"
	"time"
)

// evalAtZone evaluates an expression with the evaluation request timestamp fixed
// at 23:30 on the 1st of June in the given zone — late enough in the day that a
// zone west of UTC falls on the following date at UTC, which is what makes the
// difference visible.
func evalAtZone(t *testing.T, offsetHours int, expr string) string {
	t.Helper()
	zone := time.FixedZone("", offsetHours*3600)
	src := "library T version '1.0'\ndefine A: " + expr + "\n"
	got, err := NewEngine(WithEvaluationTimestamp(time.Date(2020, 6, 1, 23, 30, 0, 0, zone))).
		EvaluateExpression(context.Background(), src, "A", nil, nil)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	if got == nil {
		return "null"
	}
	return got.String()
}

// TestNowTodayAndTimeOfDayUseTheRequestZone covers three functions that answered
// in UTC rather than in the zone of the request they describe.
//
// CQL defines them against the evaluation request: Now() is "the date and time of
// the start timestamp associated with the evaluation request", and Today() and
// TimeOfDay() are its date and its time of day. All three called t.UTC() before
// formatting, so a request made at 23:30 on the 1st of June at UTC-5 reported:
//
//	Today()       2020-06-02      — the next day
//	TimeOfDay()   04:30:00.000    — a time nobody asked at
//	Now()         ...T04:30:00Z   — the right instant, the wrong frame
//
// The date is the one that matters most: `Today()` decides what "as of today"
// means in a measure, and a measure evaluated in the evening at UTC-5 was dating
// itself tomorrow.
//
// The conformance corpus does not settle this — its fourteen cases are all
// self-referential (`Now() = Now()`, `Today() same day or before Today()`), so
// they hold whichever frame the answers come back in.
func TestNowTodayAndTimeOfDayUseTheRequestZone(t *testing.T) {
	for _, tt := range []struct {
		offset           int
		today, timeOfDay string
	}{
		{0, "2020-06-01", "23:30:00.000"},
		{-5, "2020-06-01", "23:30:00.000"},
		{9, "2020-06-01", "23:30:00.000"},
		{14, "2020-06-01", "23:30:00.000"},
		{-12, "2020-06-01", "23:30:00.000"},
	} {
		if got := evalAtZone(t, tt.offset, "Today()"); got != tt.today {
			t.Errorf("Today() at UTC%+d = %s, want %s — the request was made on the 1st",
				tt.offset, got, tt.today)
		}
		if got := evalAtZone(t, tt.offset, "TimeOfDay()"); got != tt.timeOfDay {
			t.Errorf("TimeOfDay() at UTC%+d = %s, want %s — the request was made at 23:30",
				tt.offset, got, tt.timeOfDay)
		}
	}

	// Now() keeps the instant and reports it in the request's frame, so the
	// offset it carries is the request's.
	for _, tt := range []struct {
		offset int
		want   string
	}{
		{0, "2020-06-01T23:30:00.000Z"},
		{-5, "2020-06-01T23:30:00.000-05:00"},
		{9, "2020-06-01T23:30:00.000+09:00"},
	} {
		if got := evalAtZone(t, tt.offset, "Now()"); got != tt.want {
			t.Errorf("Now() at UTC%+d = %s, want %s", tt.offset, got, tt.want)
		}
	}

	// And the identities the corpus does pin still hold.
	for _, expr := range []string{
		"Now() = Now()",
		"TimeOfDay() = TimeOfDay()",
		"Today() same day or before Today()",
		"Today() same day or before Today() + 1 days",
	} {
		for _, offset := range []int{0, -5, 9} {
			if got := evalAtZone(t, offset, expr); got != "true" {
				t.Errorf("%s at UTC%+d = %s, want true", expr, offset, got)
			}
		}
	}
}

// TestLiteralsAssumeTheRequestOffset covers what CQL says about a DateTime
// written without one: "If no timezone offset is supplied, the timezone offset of
// the evaluation request timestamp is assumed."
//
// Until now the engine assumed nothing, so a comparison it could not place had no
// answer. That is right for FHIRPath, which makes the default a policy decision
// and provides none, and wrong for CQL, which names it.
//
// What it costs is that results depend on the request's zone. That is not a new
// dependency — Now(), Today() and TimeOfDay() already answer from it — but it now
// reaches comparisons, and a measure evaluated in two zones can select different
// populations. That is what CQL intends: a measurement period written without an
// offset is a period in the local frame of wherever it is evaluated.
func TestLiteralsAssumeTheRequestOffset(t *testing.T) {
	const (
		stated = "@2020-01-01T02:00:00Z"
		bare   = "@2020-01-01T00:00:00.0"
	)

	// Two hours apart at UTC, and the order turns on the assumed offset.
	for _, tt := range []struct {
		offset         int
		greater, hours string
	}{
		{0, "true", "2"},    // bare is 00:00Z
		{-5, "false", "-3"}, // bare is 05:00Z, after the stated instant
		{9, "true", "11"},   // bare is 15:00Z the previous day
	} {
		if got := evalAtZone(t, tt.offset, stated+" > "+bare); got != tt.greater {
			t.Errorf("%s > %s at UTC%+d = %s, want %s", stated, bare, tt.offset, got, tt.greater)
		}
		// The duration operators have to agree with the comparison, which is what
		// the toGoTime fix in the previous change made possible.
		expr := "hours between " + bare + " and " + stated
		if got := evalAtZone(t, tt.offset, expr); got != tt.hours {
			t.Errorf("%s at UTC%+d = %s, want %s", expr, tt.offset, got, tt.hours)
		}
	}

	// Equality against a bare literal follows the assumed offset too, and shows
	// why these belong in a test that fixes the zone. This pair is two hours
	// apart at UTC, so it is decidable wherever the offset lands it:
	for _, offset := range []int{0, -5, 9} {
		if got := evalAtZone(t, offset, stated+" = "+bare); got != "false" {
			t.Errorf("%s = %s at UTC%+d = %s, want false", stated, bare, offset, got)
		}
	}

	// And a pair that lands on one instant at UTC is unknown there rather than
	// equal, because the two are specified to different precisions — while away
	// from UTC they differ in the hour and the precision never comes up.
	for _, tt := range []struct {
		offset int
		want   string
	}{
		{0, "null"},
		{-5, "false"},
		{9, "false"},
	} {
		expr := "@2020-01-01T10:00:00Z = @2020-01-01T10:00:00.0"
		if got := evalAtZone(t, tt.offset, expr); got != tt.want {
			t.Errorf("%s at UTC%+d = %s, want %s", expr, tt.offset, got, tt.want)
		}
	}

	// A stated offset is never overridden.
	for _, offset := range []int{0, -5, 9} {
		if got := evalAtZone(t, offset, stated+" > @2020-01-01T00:00:00.0Z"); got != "true" {
			t.Errorf("both offsets stated at UTC%+d = %s, want true", offset, got)
		}
	}

	// Extracting the offset reports what was assumed, which CQL requires to be
	// "the timezone offset of the evaluation request, not null".
	for _, tt := range []struct {
		offset int
		want   string
	}{
		{0, "0"}, {-5, "-5"}, {9, "9"},
	} {
		if got := evalAtZone(t, tt.offset, "timezoneoffset from "+bare); got != tt.want {
			t.Errorf("timezoneoffset from a bare literal at UTC%+d = %s, want %s",
				tt.offset, got, tt.want)
		}
	}

	// And the value still prints as it was written: assuming an offset is not
	// rewriting one. This is what 22 conformance cases require.
	for _, offset := range []int{0, -5, 9} {
		if got := evalAtZone(t, offset, "@2012-03-10T10:20:00"); got != "2012-03-10T10:20:00" {
			t.Errorf("a bare literal at UTC%+d prints as %s, want 2012-03-10T10:20:00",
				offset, got)
		}
	}
}
