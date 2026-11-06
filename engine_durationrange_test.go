package cql

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The tests of plan 021 (docs/plans/021-duration-integer-range.md): a duration,
// a difference and an age are measured over the whole calendar, from the year 1
// to 9999. The first instant of the calendar, 0001-01-01T00:00:00Z, is a value
// like any other (it was read as "no value"), and a span of more than 292 years
// is counted whole (it saturated at the range of a Go time.Duration).
// CQL 1.5.3, Appendix B, Duration and Difference: the number of whole periods
// (boundaries for Difference) between two values, "any fractional" part dropped,
// with no bound on the span but the range of the calendar.

// TestDurationsOverTheCalendarByHand holds each duration to the value both
// reference engines give (Java translator 5.4.0 and cql-execution 3.3.2, which
// agree on every row below), at -03:00, compiled and through the ELM.
func TestDurationsOverTheCalendarByHand(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		// A negative span with a part of a second left over is truncated toward
		// zero, as both reference engines answer: "any fractional periods are
		// dropped" (Appendix B, Duration).
		{"seconds between @2012-01-01T00:00:01.500Z and @2012-01-01T00:00:00.000Z", "-1"},
		{"minutes between @2012-01-01T00:00:59.999Z and @2012-01-01T00:00:00.000Z", "0"},
		{"days between @2012-01-02T12:00:00.000Z and @2012-01-01T00:00:00.000Z", "-1"},
		{"seconds between @2012-01-01T00:00:01.000Z and @2012-01-01T00:00:00.500Z", "0"},
		{"days between @2012-01-02T00:00:00.000Z and @2012-01-01T00:00:00.500Z", "0"},
		// The year 1: its first instant is a value, in every form it is written.
		{"days between @0001-01-01 and @0001-01-05", "4"},
		{"days between @0001-01-05 and @0001-01-01", "-4"},
		{"hours between @0001-01-01T00:00:00.000Z and @0001-01-02T00:00:00.000Z", "24"},
		{"hours between @0001-01-02T00:00:00.000Z and @0001-01-01T00:00:00.000Z", "-24"},
		{"hours between @0001-01-01T03:00:00.000+03:00 and @0001-01-02T00:00:00.000Z", "24"},
		{"days between @0001-01-01T03:00:00.000+03:00 and @0001-01-05T00:00:00.000Z", "4"},
		{"minutes between @0001-01-01T00:00:00.000Z and @0001-01-01T00:00:00.000Z", "0"},
		{"seconds between @0001-01-01T00:00:00.000Z and @0001-01-01T00:01:00.000Z", "60"},
		{"milliseconds between @0001-01-01T00:00:00.000Z and @0001-01-01T00:00:01.000Z", "1000"},
		{"CalculateAgeInDaysAt(@0001-01-01, @2020-01-01)", "737424"},
		{"CalculateAgeInDaysAt(@0001-01-01T00:00:00.000Z, @2020-01-01T00:00:00.000Z)", "737424"},
		{"duration in days of Interval[@0001-01-01, @0001-01-05]", "4"},
		// An uncertainty whose range starts at the first instant. The second row
		// is derived from the definition (Language Semantics, Determining
		// Difference and Duration): 00:00:59.999 to 05:00:00 and 00:00:00 to it.
		{"minutes between @0001-01-01T00:00Z and @0001-01-01T05:00:00.000Z", "Interval[299, 300]"},
		{"days between @0001-01-01T and @0001-01-05T00:00:00.000", "Interval[3, 4]"},
		// Spans of more than 292 years, in each unit that is not a saturated
		// count, both ways. 106751 days are the 2^63 nanoseconds of a Duration.
		{"days between @0001-01-02 and @9999-12-31", "3652057"},
		{"days between @0001-01-01T00:00:00.000 and @9999-12-31T00:00:00.000", "3652058"},
		{"days between @9999-12-31T00:00:00.000 and @0001-01-01T00:00:00.000", "-3652058"},
		{"weeks between @0001-01-01 and @9999-12-31", "521722"},
		{"weeks between @9999-12-31 and @0001-01-01", "-521722"},
		{"hours between @1700-01-01T00:00:00.000 and @2020-01-01T00:00:00.000", "2805048"},
		{"hours between @2020-01-01T00:00:00.000 and @1700-01-01T00:00:00.000", "-2805048"},
		{"hours between @0001-01-01T00:00:00.000 and @9999-12-31T00:00:00.000", "87649392"},
		{"hours between @9999-12-31T00:00:00.000 and @0001-01-01T00:00:00.000", "-87649392"},
		{"minutes between @1727-01-01T00:00:00.000 and @2020-01-01T00:00:00.000", "154103040"},
		{"minutes between @2020-01-01T00:00:00.000 and @1727-01-01T00:00:00.000", "-154103040"},
		{"CalculateAgeInDaysAt(@1700-01-01, @2020-01-01)", "116877"},
		{"days between @1700 and @2020-01-01", "Interval[116513, 116877]"},
		// Seconds and milliseconds are Integers in CQL, and 2208988800, the
		// seconds of these 70 years, is past 2^31-1: null (plan 022). cql-execution
		// gives the JavaScript number and Java wraps it, to -2085978496.
		{"seconds between @1950-01-01T00:00:00.000 and @2020-01-01T00:00:00.000", "null"},
		// Below 292 years, and the difference, which never used a Duration.
		{"days between @1900-01-01 and @2020-01-01", "43829"},
		{"hours between @1900-01-01T00:00:00.000 and @2020-01-01T00:00:00.000", "1051896"},
		{"difference in days between @0001-01-01 and @9999-12-31", "3652058"},
		{"difference in hours between @0001-01-01T00:00:00.000 and @9999-12-31T00:00:00.000", "87649392"},
	} {
		for _, imported := range []bool{false, true} {
			if got := differenceAt(t, tt.expr, -3*60, imported); got != tt.want {
				t.Errorf("imported %v, %s = %s, want %s", imported, tt.expr, got, tt.want)
			}
		}
	}
	// Compiled only: the ELM this engine exports does not write the implicit
	// ToDateTime (plan 019, backlog 1c37), so the imported library holds a Date.
	// A Date beside a DateTime is the DateTime with its time unspecified (plan
	// 019; CQL 1.5.3, https://cql.hl7.org/N1A/, Developer's Guide, Implicit
	// Conversions: "ToDateTime(Date) Converts a Date value to a DateTime with
	// all time components unspecified"), so the days to a time of day are an
	// uncertainty, as in Appendix B, Duration: `days between @2017-08-07T17:00
	// and @2017-08-14T` is [6, 7]. cql-execution gives Uncertainty[3, 4] here,
	// and so does this engine for ToDateTime(@2012-01-01) written out; Java
	// 5.4.0 gives 4 (snapshot area mixeddates), where the same
	// uncertainty is [6, 29] in hours.
	const expr = "days between @2012-01-01 and @2012-01-05T00:00:00.000"
	if got := differenceAt(t, expr, -3*60, false); got != "Interval[3, 4]" {
		t.Errorf("%s = %s, want Interval[3, 4]", expr, got)
	}
}

// TestDurationsAtTheIntegerEdge holds a duration, a difference and an age to the
// 32 bits of a CQL Integer (plan 022, docs/plans/022-duration-out-of-range.md).
// CQL 1.5.3, Appendix B, Duration and Difference: "The result of this operation
// is always an integer", and Types: an Integer is from -2^31 to 2^31-1. So a
// value past it "cannot be represented" and, as Arithmetic Operators states for
// any such operation, "will result in null". -2^31 is representable.
//
// The expected values are counted by hand from the instants (2^31 ms is 24 days
// 20:31:23.648, 2^31 s is 24855 days 03:14:08 and 2^31 min is 1491308 days
// 02:08), at -03:00, compiled and through the ELM.
func TestDurationsAtTheIntegerEdge(t *testing.T) {
	const (
		ms   = "@2012-01-01T00:00:00.000"
		sec  = "@2000-01-01T00:00:00.000Z"
		mins = "@0001-01-01T00:00:00.000Z"
		msz  = "@2012-01-01T00:00:00.000Z"
	)
	rows := []struct{ expr, want string }{
		// Milliseconds: 24 days 20:31:23.647 is 2147483647.
		{"milliseconds between " + ms + " and @2012-01-25T20:31:23.646", "2147483646"},
		{"milliseconds between " + ms + " and @2012-01-25T20:31:23.647", "2147483647"},
		{"milliseconds between " + ms + " and @2012-01-25T20:31:23.648", "null"},
		{"milliseconds between " + ms + " and @2012-01-25T20:31:23.649", "null"},
		{"milliseconds between @2012-01-25T20:31:23.647 and " + ms, "-2147483647"},
		{"milliseconds between @2012-01-25T20:31:23.648 and " + ms, "-2147483648"},
		{"milliseconds between @2012-01-25T20:31:23.649 and " + ms, "null"},
		{"difference in milliseconds between " + ms + " and @2012-01-25T20:31:23.647", "2147483647"},
		{"difference in milliseconds between " + ms + " and @2012-01-25T20:31:23.648", "null"},
		{"difference in milliseconds between @2012-01-25T20:31:23.648 and " + ms, "-2147483648"},
		{"difference in milliseconds between @2012-01-25T20:31:23.649 and " + ms, "null"},
		{"duration in milliseconds of Interval[" + ms + ", @2012-01-25T20:31:23.647]", "2147483647"},
		{"duration in milliseconds of Interval[" + ms + ", @2012-01-25T20:31:23.648]", "null"},
		{"difference in milliseconds of Interval[" + ms + ", @2012-01-25T20:31:23.647]", "2147483647"},
		{"difference in milliseconds of Interval[" + ms + ", @2012-01-25T20:31:23.648]", "null"},
		// With an offset the span is the same.
		{"milliseconds between @2012-01-01T00:00:00.000+03:00 and @2012-01-25T20:31:23.647+03:00", "2147483647"},
		{"milliseconds between @2012-01-01T00:00:00.000+03:00 and @2012-01-25T20:31:23.648+03:00", "null"},
		// Seconds: 24855 days 03:14:07 is 2147483647, and a part of a second left
		// over is dropped toward zero.
		{"seconds between " + sec + " and @2068-01-19T03:14:06.000Z", "2147483646"},
		{"seconds between " + sec + " and @2068-01-19T03:14:07.000Z", "2147483647"},
		{"seconds between " + sec + " and @2068-01-19T03:14:07.999Z", "2147483647"},
		{"seconds between " + sec + " and @2068-01-19T03:14:08.000Z", "null"},
		{"seconds between " + sec + " and @2068-01-19T03:14:09.000Z", "null"},
		{"seconds between @2068-01-19T03:14:07.999Z and " + sec, "-2147483647"},
		{"seconds between @2068-01-19T03:14:08.000Z and " + sec, "-2147483648"},
		{"seconds between @2068-01-19T03:14:08.999Z and " + sec, "-2147483648"},
		{"seconds between @2068-01-19T03:14:09.000Z and " + sec, "null"},
		{"difference in seconds between " + sec + " and @2068-01-19T03:14:07.000Z", "2147483647"},
		{"difference in seconds between " + sec + " and @2068-01-19T03:14:08.000Z", "null"},
		{"difference in seconds between @2068-01-19T03:14:08.000Z and " + sec, "-2147483648"},
		{"difference in seconds between @2068-01-19T03:14:09.000Z and " + sec, "null"},
		{"duration in seconds of Interval[" + sec + ", @2068-01-19T03:14:07.000Z]", "2147483647"},
		{"duration in seconds of Interval[" + sec + ", @2068-01-19T03:14:08.000Z]", "null"},
		{"difference in seconds of Interval[" + sec + ", @2068-01-19T03:14:07.000Z]", "2147483647"},
		{"difference in seconds of Interval[" + sec + ", @2068-01-19T03:14:08.000Z]", "null"},
		// Minutes: 1491308 days 02:07 is 2147483647.
		{"minutes between " + mins + " and @4084-01-24T02:06:00.000Z", "2147483646"},
		{"minutes between " + mins + " and @4084-01-24T02:07:00.000Z", "2147483647"},
		{"minutes between " + mins + " and @4084-01-24T02:07:59.999Z", "2147483647"},
		{"minutes between " + mins + " and @4084-01-24T02:08:00.000Z", "null"},
		{"minutes between " + mins + " and @4084-01-24T02:09:00.000Z", "null"},
		{"minutes between @4084-01-24T02:07:59.999Z and " + mins, "-2147483647"},
		{"minutes between @4084-01-24T02:08:00.000Z and " + mins, "-2147483648"},
		{"minutes between @4084-01-24T02:08:59.999Z and " + mins, "-2147483648"},
		{"minutes between @4084-01-24T02:09:00.000Z and " + mins, "null"},
		{"difference in minutes between " + mins + " and @4084-01-24T02:07:00.000Z", "2147483647"},
		{"difference in minutes between " + mins + " and @4084-01-24T02:08:00.000Z", "null"},
		{"difference in minutes between @4084-01-24T02:08:00.000Z and " + mins, "-2147483648"},
		{"difference in minutes between @4084-01-24T02:09:00.000Z and " + mins, "null"},
		{"duration in minutes of Interval[" + mins + ", @4084-01-24T02:07:00.000Z]", "2147483647"},
		{"duration in minutes of Interval[" + mins + ", @4084-01-24T02:08:00.000Z]", "null"},
		{"difference in minutes of Interval[" + mins + ", @4084-01-24T02:07:00.000Z]", "2147483647"},
		{"difference in minutes of Interval[" + mins + ", @4084-01-24T02:08:00.000Z]", "null"},
		// An age is a duration from the birth: CQL has it in seconds and minutes
		// (Appendix B, CalculateAge), not in milliseconds. The evaluation is at
		// 2026-01-01T00:00:00-03:00, that is 03:00:00Z.
		{"CalculateAgeInSecondsAt(" + sec + ", @2068-01-19T03:14:07.000Z)", "2147483647"},
		{"CalculateAgeInSecondsAt(" + sec + ", @2068-01-19T03:14:08.000Z)", "null"},
		{"CalculateAgeInMinutesAt(" + mins + ", @4084-01-24T02:07:00.000Z)", "2147483647"},
		{"CalculateAgeInMinutesAt(" + mins + ", @4084-01-24T02:08:00.000Z)", "null"},
		// Born 2147483647 seconds before the evaluation: 1957-12-13T23:45:53Z.
		{"CalculateAgeInSeconds(@1957-12-13T23:45:53.000Z)", "2147483647"},
		{"CalculateAgeInSeconds(@1957-12-13T23:45:52.000Z)", "null"},
		// An uncertainty is the set of values the result could be, and one of them
		// that is no Integer makes the operation impossible: null as a whole
		// (decision D1 of the plan). Inside the range it stays an uncertainty.
		{"milliseconds between " + msz + " and @2012-01-25T20:29Z", "Interval[2147340000, 2147399999]"},
		{"milliseconds between " + msz + " and @2012-01-25T20:31Z", "null"},
		{"milliseconds between " + msz + " and @2012-01-26T00:00Z", "null"},
		{"milliseconds between @2012-01-25T20:29Z and " + msz, "Interval[-2147399999, -2147340000]"},
		{"milliseconds between @2012-01-25T20:31Z and " + msz, "null"},
		{"milliseconds between @2012-01-26T00:00Z and " + msz, "null"},
		{"difference in milliseconds between " + msz + " and @2012-01-25T20:29Z", "Interval[2147340000, 2147399999]"},
		{"difference in milliseconds between " + msz + " and @2012-01-25T20:31Z", "null"},
		{"difference in milliseconds between @2012-01-25T20:31Z and " + msz, "null"},
		{"seconds between " + sec + " and @2068-01-19T03:12Z", "Interval[2147483520, 2147483579]"},
		{"seconds between " + sec + " and @2068-01-19T03:14Z", "null"},
		{"seconds between " + sec + " and @2068-01-20T00:00Z", "null"},
		{"duration in seconds of Interval[" + sec + ", @2068-01-19T03:14Z]", "null"},
		{"CalculateAgeInSecondsAt(" + sec + ", @2068-01-19T03:14Z)", "null"},
		// A month left unknown spans [-2678399999, 0] ms, and 2678399999 is past 2^31.
		{"difference in milliseconds between @2012-01T and @2012-01-01T00:00:00.000", "null"},
		// Controls that do not change: spans that fit, whatever the unit.
		{"hours between @T10:00:00.000 and @T12:00:00.000", "2"},
		{"milliseconds between @T00:00:00.000 and @T23:59:59.999", "86399999"},
		{"days between @0001-01-01T00:00:00.000 and @9999-12-31T00:00:00.000", "3652058"},
		{"hours between @9999-12-31T00:00:00.000 and @0001-01-01T00:00:00.000", "-87649392"},
		{"minutes between @0001-01-01T00:00:00.000Z and @4084-01-24T02:07:00.000Z", "2147483647"},
	}
	// The spans do not depend on the offset of the evaluation, but for the age
	// from a birth date to the evaluation's own instant.
	for _, frame := range []int{-3 * 60, 5*60 + 45} {
		for _, tt := range rows {
			if frame != -3*60 && strings.HasPrefix(tt.expr, "CalculateAgeInSeconds(") {
				continue
			}
			for _, imported := range []bool{false, true} {
				if got := differenceAt(t, tt.expr, frame, imported); got != tt.want {
					t.Errorf("imported %v, frame %d, %s = %s, want %s", imported, frame, tt.expr, got, tt.want)
				}
			}
		}
	}
}

// TestADurationIsAnIntegerForCertain holds the arithmetic of plan 022: the result
// of a duration, a difference or an age is an Integer whatever the operands are,
// so the Integer rule of "cannot be represented" applies to what is done with
// it. -2^31 and 2^31-1 are durations; Abs(-2^31), its negation and the sum past
// 2^31-1 are null, and with a Long they are the Long's number. Only the compiled
// path has the plan of types that tells an Integer from a Long (plan 013): the
// imported ELM keeps the 64-bit edge, the known limit of that plan.
func TestADurationIsAnIntegerForCertain(t *testing.T) {
	const (
		least    = "(milliseconds between @2012-01-25T20:31:23.648 and @2012-01-01T00:00:00.000)" // -2147483648
		greatest = "(milliseconds between @2012-01-01T00:00:00.000 and @2012-01-25T20:31:23.647)" // 2147483647
		spanDiff = "(difference in milliseconds between @2012-01-25T20:31:23.648 and @2012-01-01T00:00:00.000)"
		age      = "(CalculateAgeInSecondsAt(@2000-01-01T00:00:00.000Z, @2068-01-19T03:14:07.000Z))" // 2147483647
		ageNow   = "(CalculateAgeInSeconds(@1957-12-13T23:45:53.000Z))"                              // 2147483647
		interDif = "(difference in milliseconds of Interval[@2012-01-01T00:00:00.000, @2012-01-25T20:31:23.647])"
		inter    = "(duration in milliseconds of Interval[@2012-01-01T00:00:00.000, @2012-01-25T20:31:23.647])"
		// Uncertainties, for the time zone of the evaluation: the bounds are
		// [-2147483648, -2147423649] and [2147340000, 2147399999].
		uncertainLeast = "(milliseconds between @2012-01-25T20:31:23.648Z and @2012-01-01T00:00Z)"
		uncertainGreat = "(milliseconds between @2012-01-01T00:00:00.000Z and @2012-01-25T20:29Z)"
	)
	for _, tt := range []struct{ expr, want string }{
		{"Abs(" + least + ")", "null"},
		{"-" + least, "null"},
		{least + " - 1", "null"},
		{"Abs(" + spanDiff + ")", "null"},
		{"-" + spanDiff, "null"},
		{greatest + " + 1", "null"},
		{age + " + 1", "null"},
		{inter + " + 1", "null"},
		{interDif + " + 1", "null"},
		{ageNow + " + 1", "null"},
		{"2 * " + greatest, "null"},
		// The control: a Long holds the number.
		{greatest + " + 1L", "2147483648"},
		{least + " - 1L", "-2147483649"},
		{"Abs(" + least + " + 0L)", "2147483648"},
		// Within the range nothing changes.
		{least + " + 1", "-2147483647"},
		{"Abs(" + least + " + 1)", "2147483647"},
		{"-" + greatest, "-2147483647"},
		{"Abs(" + greatest + ")", "2147483647"},
		{greatest + " - 1", "2147483646"},
		{age + " - 1", "2147483646"},
		{ageNow + " - 1", "2147483646"},
		{interDif + " - 1", "2147483646"},
		// An uncertainty of Integers is one whole: a bound past the 32 bits
		// makes it null (plan 022, D1), for a negation, Abs and arithmetic.
		{"-" + uncertainLeast, "null"},
		{"Abs(" + uncertainLeast + ")", "null"},
		{uncertainGreat + " + 100000", "null"},
		{uncertainGreat + " + 100000 > 2147400000", "null"},
		// The low bound too: [-2147483648, -2147423649] - 1 reaches -2^31-1.
		{uncertainLeast + " - 1", "null"},
		{"(" + uncertainLeast + " - 1) < 0", "null"},
		// The control: within the range it stays an uncertainty.
		{uncertainGreat + " + 100000L", "Interval[2147440000, 2147499999]"},
		{uncertainGreat + " - 100000", "Interval[2147240000, 2147299999]"},
		{"-" + uncertainGreat, "Interval[-2147399999, -2147340000]"},
	} {
		if got := differenceAt(t, tt.expr, -3*60, false); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// calendarOperand is a literal of the area: a Date, or a DateTime to the
// millisecond with the offset it states, if any.
type calendarOperand struct {
	date                                bool
	year, month, day, hour, minute, sec int64
	milli                               int64
	offset                              int64 // minutes, when stated
	stated                              bool
}

var calendarLiteral = regexp.MustCompile(`^@(\d{4})-(\d{2})-(\d{2})(?:T(\d{2}):(\d{2}):(\d{2})\.(\d{3})(Z|[+-]\d{2}:\d{2})?)?$`)

// parseCalendarOperand reads a literal complete to the day (a Date) or to the
// millisecond. It declines the imprecise ones and the Times.
func parseCalendarOperand(text string) (calendarOperand, bool) {
	m := calendarLiteral.FindStringSubmatch(text)
	if m == nil {
		return calendarOperand{}, false
	}
	n := func(i int) int64 { v, _ := strconv.ParseInt(m[i], 10, 64); return v }
	op := calendarOperand{year: n(1), month: n(2), day: n(3), date: m[4] == ""}
	if op.date {
		return op, true
	}
	op.hour, op.minute, op.sec, op.milli = n(4), n(5), n(6), n(7)
	switch z := m[8]; {
	case z == "Z":
		op.stated = true
	case z != "":
		hh, _ := strconv.ParseInt(z[1:3], 10, 64)
		mm, _ := strconv.ParseInt(z[4:6], 10, 64)
		op.offset = hh*60 + mm
		if z[0] == '-' {
			op.offset = -op.offset
		}
		op.stated = true
	}
	return op, true
}

// julianDay is the Julian day number of a day of the proleptic Gregorian
// calendar (Fliegel and Van Flandern), by integer arithmetic alone.
func julianDay(year, month, day int64) int64 {
	a := (14 - month) / 12
	y := year + 4800 - a
	m := month + 12*a - 3
	return day + (153*m+2)/5 + 365*y + y/4 - y/100 + y/400 - 32045
}

// unixJulianDay is the Julian day number of 1970-01-01.
const unixJulianDay = 2440588

// instantMillis is the instant in milliseconds from 1970-01-01T00:00:00Z. A
// DateTime that states no offset is at the evaluation's, a Date at none.
func (o calendarOperand) instantMillis(frame int64) int64 {
	ms := (julianDay(o.year, o.month, o.day) - unixJulianDay) * 86400000
	if o.date {
		return ms
	}
	ms += ((o.hour*60+o.minute)*60+o.sec)*1000 + o.milli
	if o.stated {
		return ms - o.offset*60000
	}
	return ms - frame*60000
}

// fields are the components as written, most significant first.
func (o calendarOperand) fields() [7]int64 {
	return [7]int64{o.year, o.month, o.day, o.hour, o.minute, o.sec, o.milli}
}

// durationOracle is `<unit> between a and b`, by the Julian day number and
// integer milliseconds: the whole periods, fractions dropped toward zero. Years
// and months are counted on the fields as written. The evaluation is at frame
// minutes from UTC.
func durationOracle(unit string, a, b calendarOperand, frame int64) string {
	switch unit {
	case "years", "months":
		months := (b.year-a.year)*12 + b.month - a.month
		fa, fb := a.fields(), b.fields()
		later := func(x, y [7]int64) bool { // x is after y from the day on
			for i := 2; i < 7; i++ {
				if x[i] != y[i] {
					return x[i] > y[i]
				}
			}
			return false
		}
		switch {
		case months > 0 && later(fa, fb):
			months--
		case months < 0 && later(fb, fa):
			months++
		}
		if unit == "years" {
			months /= 12
		}
		return strconv.FormatInt(months, 10)
	}
	span := b.instantMillis(frame) - a.instantMillis(frame)
	length := map[string]int64{
		"weeks": 7 * 86400000, "days": 86400000, "hours": 3600000, "minutes": 60000, "seconds": 1000, "milliseconds": 1,
	}[unit]
	return strconv.FormatInt(span/length, 10) // Go truncates toward zero
}

// boundaryOracle is `difference in <unit> between a and b`: the boundaries
// crossed, on the fields as written from years to days (a week starts on
// Sunday) and on the wall clock of the evaluation's frame below the day.
func boundaryOracle(unit string, a, b calendarOperand, frame int64) string {
	sunday := func(o calendarOperand) int64 {
		jd := julianDay(o.year, o.month, o.day)
		return jd - (jd+1)%7 // 1970-01-01, Julian day 2440588, was a Thursday
	}
	wall := func(o calendarOperand) int64 { return o.instantMillis(frame) + frame*60000 }
	var n int64
	switch unit {
	case "years":
		n = b.year - a.year
	case "months":
		n = (b.year-a.year)*12 + b.month - a.month
	case "weeks":
		n = (sunday(b) - sunday(a)) / 7
	case "days":
		n = julianDay(b.year, b.month, b.day) - julianDay(a.year, a.month, a.day)
	default:
		length := map[string]int64{"hours": 3600000, "minutes": 60000, "seconds": 1000, "milliseconds": 1}[unit]
		n = floorDiv(wall(b), length) - floorDiv(wall(a), length)
	}
	return strconv.FormatInt(n, 10)
}

// integerOrNull is the answer for a count: the count if it is an Integer, which
// is from -2^31 to 2^31-1, and null if not. The class says where the count is:
// "edge" within two of either limit (-2^31 is one), "out of range" past it, and
// "inside" otherwise.
func integerOrNull(count string) (answer, class string) {
	n, err := strconv.ParseInt(count, 10, 64)
	if err != nil {
		return count, "unparsed"
	}
	const limit = 1 << 31
	switch {
	case n >= limit || n < -limit:
		class = "out of range"
	case n >= limit-2 || n <= -limit+2:
		class = "edge"
	default:
		class = "inside"
	}
	if class == "out of range" {
		return "null", class
	}
	return count, class
}

// ageUnits maps the unit of CalculateAgeIn<Unit>At to the unit of a duration.
var ageUnits = map[string]string{
	"Years": "years", "Months": "months", "Weeks": "weeks", "Days": "days",
	"Hours": "hours", "Minutes": "minutes", "Seconds": "seconds",
}

// readAnswers reads the answers of an area: one define and its answer per line.
func readAnswers(t *testing.T, area string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile("testdata/snapshots/" + area + ".answers.tsv")
	if err != nil {
		t.Fatal(err)
	}
	answers := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		if name, answer, ok := strings.Cut(line, "\t"); ok && !strings.HasPrefix(name, "#") {
			answers[name] = strings.TrimPrefix(answer, "Integer ")
		}
	}
	return answers
}

// TestTheDurationRangeAreaIsItsOracle holds every definition of the snapshot
// area `durationrange` whose operands are literals complete to the day or the
// millisecond to an oracle that counts with the Julian day number and integer
// milliseconds, and uses neither the time package nor the engine's funcs: a
// duration, a difference, the same of an Interval and an age, at -03:00 (the
// evaluation offset of the snapshots, which also fixes the age's reference).
// Operands that are imprecise are left to the references. A count past an
// Integer's 32 bits, below -2^31 or above 2^31-1, is null (plan 022); the rows
// are counted by class (inside the range, at its edge and out of it), and each
// class has to be well represented, so that a change of the edge is seen.
func TestTheDurationRangeAreaIsItsOracle(t *testing.T) {
	const frame = -3 * 60
	pair := regexp.MustCompile(`^define (D\d+): (?:(difference in |duration in )?(\w+) (?:between|of) (?:Interval\[)?(@[^\s,\]]+)(?: and |, )(@[^\s\]]+)\]?|CalculateAgeIn(\w+)At\((@[^,]+), (@[^)]+)\))$`)
	cql, err := os.ReadFile("testdata/snapshots/durationrange.cql")
	if err != nil {
		t.Fatal(err)
	}
	answers := readAnswers(t, "durationrange")
	checked := 0
	classes := map[string]int{}
	for _, line := range strings.Split(string(cql), "\n") {
		m := pair.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name, kind, unit, from, to := m[1], m[2], m[3], m[4], m[5]
		if m[6] != "" {
			kind, unit, from, to = "", ageUnits[m[6]], m[7], m[8]
		}
		a, okA := parseCalendarOperand(from)
		b, okB := parseCalendarOperand(to)
		if !okA || !okB || a.date != b.date || unit == "" {
			continue
		}
		var want string
		if kind == "difference in " {
			want = boundaryOracle(unit, a, b, frame)
		} else {
			want = durationOracle(unit, a, b, frame)
		}
		want, class := integerOrNull(want)
		if got := answers[name]; got != want {
			t.Errorf("%s: %s = %s, the oracle gives %s", name, line, got, want)
		}
		classes[class]++
		checked++
	}
	if checked < 1800 {
		t.Fatalf("only %d definitions of the area were checked against the oracle", checked)
	}
	for class, least := range map[string]int{"inside": 1200, "edge": 120, "out of range": 300} {
		if classes[class] < least {
			t.Errorf("only %d %s rows were checked against the oracle, want at least %d", classes[class], class, least)
		}
	}
}
