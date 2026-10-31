package funcs

import (
	"fmt"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

// NowAt returns the given instant as a DateTime. CQL asks for the timestamp of
// the evaluation request rather than the clock's reading at the moment of the
// call, so that every Now in one evaluation agrees with every other.
func NowAt(t time.Time) (fptypes.Value, error) {
	// Formatted in the request's own zone, not at UTC. CQL defines this as "the
	// date and time of the start timestamp associated with the evaluation
	// request", and converting to UTC first kept the instant while discarding the
	// frame it was asked in — so a request at 23:30 on the 1st at UTC-5 reported
	// 04:30 on the 2nd. The instant is the same either way; what was lost is the
	// offset, which CQL expects to be the request's.
	return fptypes.NewDateTime(t.Format("2006-01-02T15:04:05.000Z07:00"))
}

// TodayAt returns the given instant's date. See NowAt.
func TodayAt(t time.Time) (fptypes.Value, error) {
	// The date of the request in its own zone. At UTC this dated a late-evening
	// request in a western zone to the following day, which is the answer a
	// measure reads when it asks what "today" is.
	return fptypes.NewDate(t.Format("2006-01-02"))
}

// TimeOfDayAt returns the given instant's time of day. See NowAt.
func TimeOfDayAt(t time.Time) (fptypes.Value, error) {
	// The time of day of the request in its own zone. See NowAt.
	return fptypes.NewTime(t.Format("15:04:05.000"))
}

// temporalParts holds a Date or DateTime as its written components, year to
// millisecond, and the precision that says how many of them are known.
type temporalParts struct {
	field [7]int // year, month, day, hour, minute, second, millisecond
	prec  int    // index of the last component written
}

// partsOf reads a Date or DateTime, or the string of one, as its components.
// A missing month or day reads as 1, as the engine's own arithmetic reads it;
// the comparison never looks past the precision, so they decide nothing.
func partsOf(v fptypes.Value) (temporalParts, bool) {
	switch t := v.(type) {
	case fptypes.DateTime:
		return temporalParts{
			field: [7]int{t.Year(), max(t.Month(), 1), max(t.Day(), 1), t.Hour(), t.Minute(), t.Second(), t.Millisecond()},
			prec:  int(t.Precision()),
		}, true
	case fptypes.Date:
		return temporalParts{
			field: [7]int{t.Year(), max(t.Month(), 1), max(t.Day(), 1)},
			prec:  int(t.Precision()),
		}, true
	case nil:
		return temporalParts{}, false
	}
	if dt, err := fptypes.NewDateTime(v.String()); err == nil {
		return partsOf(dt)
	}
	if d, err := fptypes.NewDate(v.String()); err == nil {
		return partsOf(d)
	}
	return temporalParts{}, false
}

// compareParts orders two values at the given precision: -1, 0 or 1. A
// component finer than the precision is not compared (CQL 1.5.3, Comparison
// Operators: "the comparison is performed at the highest precision both
// operands share"), which is why a Date beside a DateTime assumes no midnight.
func compareParts(a, b temporalParts, prec int) int {
	for i := 0; i <= prec; i++ {
		if a.field[i] != b.field[i] {
			if a.field[i] < b.field[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

// calendarPeriodsBetween counts the whole calendar periods, years or months, from
// low to high. CQL 1.5.3, Appendix B, Duration: "the number of whole calendar
// periods for the specified precision between the first and second arguments. If
// the first argument is after the second argument, the result is negative".
//
// A whole period is the largest n for which low + n periods is not after high
// (plan 015, decision A), the sum being the one the `+` operator performs: a day
// the target month does not have falls on its last day, so 29 February 2012 is a
// year old on 28 February 2013. The comparison runs at the precision both
// operands share, down to the millisecond (decision B).
func calendarPeriodsBetween(low, high temporalParts, unit string) int64 {
	prec := min(low.prec, high.prec)
	if compareParts(low, high, prec) > 0 {
		return -calendarPeriodsBetween(high, low, unit)
	}
	periods := int64(high.field[0] - low.field[0])
	if unit == "month" {
		periods = periods*12 + int64(high.field[1]-low.field[1])
	}
	if periods > 0 && !periodsFit(low, high, prec, periods, unit) {
		// The year or month field of low + periods is high's own, so one period
		// fewer is always short of high.
		periods--
	}
	return periods
}

// periodsFit reports whether low + periods is not after high at the precision.
func periodsFit(low, high temporalParts, prec int, periods int64, unit string) bool {
	day, err := fptypes.NewDate(fmt.Sprintf("%04d-%02d-%02d", low.field[0], low.field[1], low.field[2]))
	if err != nil {
		return false
	}
	moved, err := day.AddDuration(int(periods), unit)
	if err != nil {
		// Past year 9999: no such date, which is later than any that exists.
		return false
	}
	candidate := low
	candidate.field[0], candidate.field[1], candidate.field[2] = moved.Year(), moved.Month(), moved.Day()
	return compareParts(candidate, high, prec) <= 0
}

func calendarBetween(low, high fptypes.Value, unit string) (fptypes.Value, error) {
	l, lok := partsOf(low)
	h, hok := partsOf(high)
	if !lok || !hok {
		return nil, nil
	}
	return fptypes.NewInteger(calendarPeriodsBetween(l, h, unit)), nil
}

// YearsBetween calculates the number of whole years between two dates.
func YearsBetween(low, high fptypes.Value) (fptypes.Value, error) {
	return calendarBetween(low, high, "year")
}

// MonthsBetween calculates the number of whole months between two dates.
func MonthsBetween(low, high fptypes.Value) (fptypes.Value, error) {
	return calendarBetween(low, high, "month")
}

// WeeksBetween calculates the number of whole weeks between two dates.
func WeeksBetween(low, high fptypes.Value) (fptypes.Value, error) {
	days, err := DaysBetween(low, high)
	if err != nil || days == nil {
		return nil, err
	}
	d := days.(fptypes.Integer).Value()
	return fptypes.NewInteger(d / 7), nil
}

// DaysBetween calculates the number of whole days between two dates.
func DaysBetween(low, high fptypes.Value) (fptypes.Value, error) {
	tl := toGoTime(low)
	th := toGoTime(high)
	if tl.IsZero() || th.IsZero() {
		return nil, nil
	}
	days := int(th.Sub(tl).Hours() / 24)
	return fptypes.NewInteger(int64(days)), nil
}

// HoursBetween calculates the number of whole hours between two datetimes.
func HoursBetween(low, high fptypes.Value) (fptypes.Value, error) {
	tl := toGoTime(low)
	th := toGoTime(high)
	if tl.IsZero() || th.IsZero() {
		return nil, nil
	}
	hours := int(th.Sub(tl).Hours())
	return fptypes.NewInteger(int64(hours)), nil
}

// MinutesBetween calculates the number of whole minutes between two datetimes.
func MinutesBetween(low, high fptypes.Value) (fptypes.Value, error) {
	tl := toGoTime(low)
	th := toGoTime(high)
	if tl.IsZero() || th.IsZero() {
		return nil, nil
	}
	minutes := int(th.Sub(tl).Minutes())
	return fptypes.NewInteger(int64(minutes)), nil
}

// SecondsBetween calculates the number of whole seconds between two datetimes.
func SecondsBetween(low, high fptypes.Value) (fptypes.Value, error) {
	tl := toGoTime(low)
	th := toGoTime(high)
	if tl.IsZero() || th.IsZero() {
		return nil, nil
	}
	seconds := int(th.Sub(tl).Seconds())
	return fptypes.NewInteger(int64(seconds)), nil
}
