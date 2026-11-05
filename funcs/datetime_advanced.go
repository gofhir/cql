package funcs

import (
	"fmt"
	"math"
	"time"

	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
)

// DateTimeComponentFrom extracts a component from a DateTime, Date, or Time value.
// Component names: "year", "month", "day", "hour", "minute", "second", "millisecond",
// "timezone", "date", "time".
func DateTimeComponentFrom(operand fptypes.Value, component string) (fptypes.Value, error) {
	if operand == nil {
		return nil, nil
	}

	switch component {
	case "year":
		switch t := operand.(type) {
		case fptypes.DateTime:
			return fptypes.NewInteger(int64(t.Year())), nil
		case fptypes.Date:
			return fptypes.NewInteger(int64(t.Year())), nil
		}
	case "month":
		switch t := operand.(type) {
		case fptypes.DateTime:
			return fptypes.NewInteger(int64(t.Month())), nil
		case fptypes.Date:
			return fptypes.NewInteger(int64(t.Month())), nil
		}
	case "day":
		switch t := operand.(type) {
		case fptypes.DateTime:
			return fptypes.NewInteger(int64(t.Day())), nil
		case fptypes.Date:
			return fptypes.NewInteger(int64(t.Day())), nil
		}
	case "hour":
		if t, ok := operand.(fptypes.DateTime); ok {
			if t.Precision() < fptypes.DTHourPrecision {
				return nil, nil // hour not specified
			}
			return fptypes.NewInteger(int64(t.Hour())), nil
		}
		if t, ok := operand.(fptypes.Time); ok {
			return fptypes.NewInteger(int64(t.Hour())), nil
		}
	case precMinute:
		if t, ok := operand.(fptypes.DateTime); ok {
			if t.Precision() < fptypes.DTMinutePrecision {
				return nil, nil // minute not specified
			}
			return fptypes.NewInteger(int64(t.Minute())), nil
		}
		if t, ok := operand.(fptypes.Time); ok {
			if t.Precision() < fptypes.MinutePrecision {
				return nil, nil
			}
			return fptypes.NewInteger(int64(t.Minute())), nil
		}
	case precSecond:
		if t, ok := operand.(fptypes.DateTime); ok {
			if t.Precision() < fptypes.DTSecondPrecision {
				return nil, nil // second not specified
			}
			return fptypes.NewInteger(int64(t.Second())), nil
		}
		if t, ok := operand.(fptypes.Time); ok {
			if t.Precision() < fptypes.SecondPrecision {
				return nil, nil
			}
			return fptypes.NewInteger(int64(t.Second())), nil
		}
	case precMillisecond:
		if t, ok := operand.(fptypes.DateTime); ok {
			if t.Precision() < fptypes.DTMillisPrecision {
				return nil, nil // millisecond not specified
			}
			return fptypes.NewInteger(int64(t.Millisecond())), nil
		}
		if t, ok := operand.(fptypes.Time); ok {
			if t.Precision() < fptypes.MillisPrecision {
				return nil, nil
			}
			return fptypes.NewInteger(int64(t.Millisecond())), nil
		}
	// "timezone" is the older spelling of "timezoneoffset" and names the same
	// component, so both answer with the offset in hours.
	case "timezone", "timezoneoffset":
		if dt, ok := operand.(fptypes.DateTime); ok {
			// EffectiveOffset rather than HasTZ plus TZOffset, which reported
			// null for a value that states no offset. CQL says otherwise: where
			// the offset was defaulted from the evaluation request, "the result
			// of extracting the timezone offset component will be the timezone
			// offset of the evaluation request, not null". EffectiveOffset
			// answers for a stated offset and a defaulted one alike, and reports
			// ok=false only when there is neither.
			if minutes, ok := dt.EffectiveOffset(); ok {
				return fptypes.NewDecimalFromFloat(float64(minutes) / 60.0), nil
			}
			return nil, nil
		}
		return nil, nil
	case "date":
		if dt, ok := operand.(fptypes.DateTime); ok {
			s := fmt.Sprintf("%04d", dt.Year())
			if dt.Month() > 0 {
				s += fmt.Sprintf("-%02d", dt.Month())
			}
			if dt.Day() > 0 {
				s += fmt.Sprintf("-%02d", dt.Day())
			}
			return fptypes.NewDate(s)
		}
		if d, ok := operand.(fptypes.Date); ok {
			return d, nil
		}
	case "time":
		if dt, ok := operand.(fptypes.DateTime); ok {
			return fptypes.NewTime(fmt.Sprintf("%02d:%02d:%02d.%03d", dt.Hour(), dt.Minute(), dt.Second(), dt.Millisecond()))
		}
		if t, ok := operand.(fptypes.Time); ok {
			return t, nil
		}
	}

	return nil, fmt.Errorf("unknown date/time component: %s", component)
}

// DayOfWeek returns the day of the week (0=Sunday through 6=Saturday).
func DayOfWeek(operand fptypes.Value) (fptypes.Value, error) {
	if operand == nil {
		return nil, nil
	}
	t, err := toTime(operand)
	if err != nil || t.IsZero() {
		return nil, nil
	}
	return fptypes.NewInteger(int64(t.Weekday())), nil
}

// DateConstructor creates a Date value from year, month, day components.
func DateConstructor(year, month, day fptypes.Value) (fptypes.Value, error) {
	y := intVal(year)
	m := intVal(month)
	d := intVal(day)
	if y == 0 {
		return nil, nil
	}
	if m == 0 {
		return fptypes.NewDate(fmt.Sprintf("%04d", y))
	}
	if d == 0 {
		return fptypes.NewDate(fmt.Sprintf("%04d-%02d", y, m))
	}
	return fptypes.NewDate(fmt.Sprintf("%04d-%02d-%02d", y, m, d))
}

// DateTimeConstructor creates a DateTime from components.
// Produces a precision-aware DateTime based on which components are provided.
func DateTimeConstructor(year, month, day, hour, minute, second, millisecond, tzOffset fptypes.Value) (fptypes.Value, error) {
	if year == nil {
		return nil, nil
	}
	y := intVal(year)
	if y < 1 || y > 9999 {
		return nil, fmt.Errorf("invalid DateTime year: %d", y)
	}

	// Build the string representation with only the components that are specified
	s := fmt.Sprintf("%04d", y)
	if month != nil {
		s += fmt.Sprintf("-%02d", intVal(month))
		if day != nil {
			s += fmt.Sprintf("-%02d", intVal(day))
			if hour != nil {
				s += fmt.Sprintf("T%02d", intVal(hour))
				if minute != nil {
					s += fmt.Sprintf(":%02d", intVal(minute))
					if second != nil {
						s += fmt.Sprintf(":%02d", intVal(second))
						if millisecond != nil {
							s += fmt.Sprintf(".%03d", intVal(millisecond))
						}
					}
				}
			}
		}
	}

	if tzOffset != nil {
		switch tz := tzOffset.(type) {
		case fptypes.String:
			s += tz.Value()
		case fptypes.Decimal:
			// Handle decimal timezone offset (e.g., -6.0 -> "-06:00")
			f, _ := tz.Value().Float64()
			hours := int(f)
			mins := int((f - float64(hours)) * 60)
			if mins < 0 {
				mins = -mins
			}
			s += fmt.Sprintf("%+03d:%02d", hours, mins)
		case fptypes.Integer:
			// Handle integer timezone offset (e.g., 1 -> "+01:00")
			hours := int(tz.Value())
			s += fmt.Sprintf("%+03d:00", hours)
		}
	}
	return cqltypes.NewDateTime(s)
}

// TimeConstructor creates a Time from hour, minute, second, millisecond.
func TimeConstructor(hour, minute, second, millisecond fptypes.Value) (fptypes.Value, error) {
	if hour == nil {
		return nil, nil
	}
	s := fmt.Sprintf("%02d", intVal(hour))
	if minute != nil {
		s += fmt.Sprintf(":%02d", intVal(minute))
		if second != nil {
			s += fmt.Sprintf(":%02d", intVal(second))
			if millisecond != nil {
				s += fmt.Sprintf(".%03d", intVal(millisecond))
			}
		}
	}
	return cqltypes.NewTime(s)
}

// DurationBetween calculates the duration between two date/time values at a given precision.
// When operands have different precisions, returns an Interval representing the uncertainty range.
func DurationBetween(low, high fptypes.Value, precision string) (fptypes.Value, error) {
	return DurationBetweenAt(low, high, precision, nil)
}

// DurationBetweenAt is DurationBetween for an evaluation at the offset of
// evaluation, nil for UTC. CQL 1.5.3 puts a DateTime that states no offset at the
// evaluation request's, and a DateTime to the day or coarser cannot carry one (it
// has no time of day to place), so the range of its day is read in this frame.
//
// The result is an Integer, and a count that cannot be represented as one is
// null: see inIntegerRange.
func DurationBetweenAt(low, high fptypes.Value, precision string, evaluation *time.Location) (fptypes.Value, error) {
	v, err := durationBetweenAt(low, high, precision, evaluation)
	if err != nil {
		return nil, err
	}
	return inIntegerRange(v), nil
}

// inIntegerRange is v if it is an Integer, or an uncertainty between Integers,
// that fits the 32 bits of one, and null if not. CQL 1.5.3, Appendix B, Duration
// and Difference: "The result of this operation is always an integer"; and,
// Arithmetic Operators, an operation whose result cannot be represented "will
// result in null". An uncertainty is the set of values the result could be: one
// of them out of range makes the operation impossible as an Integer, and the
// whole is null (decision D1 of docs/plans/022-duration-out-of-range.md). The
// bounds are 64-bit counts until here, so that none wraps on the way.
func inIntegerRange(v fptypes.Value) fptypes.Value {
	switch x := v.(type) {
	case fptypes.Integer:
		if !fitsInteger(x) {
			return nil
		}
	case cqltypes.Uncertainty:
		low, okLow := x.Low.(fptypes.Integer)
		high, okHigh := x.High.(fptypes.Integer)
		if (okLow && !fitsInteger(low)) || (okHigh && !fitsInteger(high)) {
			return nil
		}
	}
	return v
}

func fitsInteger(i fptypes.Integer) bool {
	return i.Value() >= math.MinInt32 && i.Value() <= math.MaxInt32
}

func durationBetweenAt(low, high fptypes.Value, precision string, evaluation *time.Location) (fptypes.Value, error) {
	if low == nil || high == nil {
		return nil, nil
	}

	// Check if either operand has uncertainty (precision <= requested precision
	// and there are sub-components that could vary).
	// For duration, even if precision matches the requested unit, there's uncertainty
	// because sub-unit components are unknown.
	precIdx := durationPrecisionIndex(precision)
	lowPrec := valuePrecisionIndex(low)
	highPrec := valuePrecisionIndex(high)
	maxPossiblePrec := 6 // millisecond

	if precIdx >= 0 && (lowPrec < maxPossiblePrec || highPrec < maxPossiblePrec) && (lowPrec <= precIdx || highPrec <= precIdx) {
		// One or both operands have precision at or below the requested level
		return durationBetweenUncertain(low, high, precision, evaluation)
	}

	return durationBetweenExact(low, high, precision)
}

func durationBetweenExact(low, high fptypes.Value, precision string) (fptypes.Value, error) {
	switch precision {
	case "year", "years":
		return YearsBetween(low, high)
	case "month", "months":
		return MonthsBetween(low, high)
	case "week", "weeks":
		return WeeksBetween(low, high)
	case "day", "days":
		return DaysBetween(low, high)
	case "hour", "hours":
		return HoursBetween(low, high)
	case precMinute, "minutes":
		return MinutesBetween(low, high)
	case precSecond, "seconds":
		return SecondsBetween(low, high)
	case precMillisecond, "milliseconds":
		return MillisecondsBetween(low, high)
	default:
		return DaysBetween(low, high) // default to days
	}
}

// durationPrecisionIndex maps duration precision strings to component indices.
// 0=year, 1=month, 2=day, 3=hour, 4=minute, 5=second, 6=millisecond
func durationPrecisionIndex(precision string) int {
	switch precision {
	case "year", "years":
		return 0
	case "month", "months":
		return 1
	case "week", "weeks":
		return 2 // same as day
	case "day", "days":
		return 2
	case "hour", "hours":
		return 3
	case precMinute, "minutes":
		return 4
	case precSecond, "seconds":
		return 5
	case precMillisecond, "milliseconds":
		return 6
	default:
		return -1
	}
}

// valuePrecisionIndex returns the precision level of a temporal value.
//
// Seconds and milliseconds are one precision: CQL 1.5.3, Language Semantics,
// Determining Difference and Duration, "consider seconds and milliseconds as a
// single combined precision". A value to the second is therefore as precise as
// one to the millisecond, and both report the millisecond's index, so that
// `seconds between @T01:00:00 and @T01:00:01.0` is 1 and not an uncertainty.
func valuePrecisionIndex(v fptypes.Value) int {
	switch t := v.(type) {
	case fptypes.DateTime:
		return combinedSecondsAndMilliseconds(int(t.Precision()))
	case fptypes.Date:
		return int(t.Precision())
	case fptypes.Time:
		return combinedSecondsAndMilliseconds(int(t.Precision()) + 3)
	default:
		return 6 // assume full precision for unknown
	}
}

// combinedSecondsAndMilliseconds maps the second's index to the millisecond's.
func combinedSecondsAndMilliseconds(index int) int {
	if index == 5 {
		return 6
	}
	return index
}

// durationBetweenUncertain computes the min/max range of the duration
// when one or both operands have insufficient precision.
func durationBetweenUncertain(low, high fptypes.Value, precision string, evaluation *time.Location) (fptypes.Value, error) {
	// Compute the earliest and latest possible time.Time for each operand
	lowEarliest, lowLatest, lowOK := temporalRange(low, evaluation)
	highEarliest, highLatest, highOK := temporalRange(high, evaluation)
	if !lowOK || !highOK {
		return nil, nil
	}

	// Min duration: latest low to earliest high
	// Max duration: earliest low to latest high
	minLow := lowLatest
	minHigh := highEarliest
	maxLow := lowEarliest
	maxHigh := highLatest

	// Create temporary fptypes values for calculation
	minLowDT := fptypes.NewDateTimeFromTime(minLow)
	minHighDT := fptypes.NewDateTimeFromTime(minHigh)
	maxLowDT := fptypes.NewDateTimeFromTime(maxLow)
	maxHighDT := fptypes.NewDateTimeFromTime(maxHigh)

	minVal, err := durationBetweenExact(minLowDT, minHighDT, precision)
	if err != nil {
		return nil, err
	}
	maxVal, err := durationBetweenExact(maxLowDT, maxHighDT, precision)
	if err != nil {
		return nil, err
	}

	if minVal == nil || maxVal == nil {
		return nil, nil
	}

	minInt := minVal.(fptypes.Integer).Value()
	maxInt := maxVal.(fptypes.Integer).Value()
	if minInt == maxInt {
		return minVal, nil // no uncertainty
	}

	// An uncertainty rather than an interval: the duration is one of these
	// values and the engine cannot say which. See cqltypes.Uncertainty for why
	// the distinction is a type.
	return cqltypes.NewUncertainty(
		fptypes.NewInteger(minInt),
		fptypes.NewInteger(maxInt),
	), nil
}

// dateTimeInstant places a DateTime on the clock, at the offset it states or the
// one it was told to assume.
//
// ToTime alone puts a value that states no offset at UTC, which is right for
// ToTime and wrong for anything asking where the value sits: the caller may have
// supplied a default, and CQL says a bare value carries the evaluation request's
// offset. Every path here that turns a DateTime into an instant goes through
// this, so they cannot disagree about it — they already did once, with the
// duration operators measuring from UTC while comparison measured from the
// default.
func dateTimeInstant(t fptypes.DateTime) time.Time {
	at := t.ToTime()
	if t.HasTZ() {
		return at
	}
	minutes, ok := t.EffectiveOffset()
	if !ok {
		return at
	}
	return time.Date(at.Year(), at.Month(), at.Day(), at.Hour(), at.Minute(),
		at.Second(), at.Nanosecond(), time.FixedZone("", minutes*60))
}

// temporalRange returns the earliest and latest possible time.Time for a temporal value.
//
// The range of a DateTime is built in the frame of the value: the end of its day
// is the end of the day it sits in. A value with no offset of its own, one to the
// day or coarser (fhirpath places no offset on a value without a time of day),
// is read at the evaluation's: measuring it from UTC against a value at -12:00
// moved the end of its day twelve hours, and `days between
// @2017-08-07T17:00 and @2017-08-14T` was [5, 6] where the specification says
// [6, 7]. With no frame on either side both sit at UTC.
//
// ok is false for a value with no instant, which has no range to measure.
func temporalRange(v fptypes.Value, frame *time.Location) (earliest, latest time.Time, ok bool) {
	switch t := v.(type) {
	case fptypes.DateTime:
		earliest := dateTimeInstant(t)
		if _, placed := t.EffectiveOffset(); !placed && frame != nil {
			earliest = time.Date(earliest.Year(), earliest.Month(), earliest.Day(), earliest.Hour(),
				earliest.Minute(), earliest.Second(), earliest.Nanosecond(), frame)
		}
		latest := earliest
		switch t.Precision() {
		case 0: // year
			latest = time.Date(t.Year(), 12, 31, 23, 59, 59, 999999999, earliest.Location())
		case 1: // month
			m := t.Month()
			if m == 0 {
				m = 1
			}
			// Last day of month
			lastDay := time.Date(t.Year(), time.Month(m+1), 0, 23, 59, 59, 999999999, earliest.Location())
			latest = lastDay
		case 2: // day
			latest = time.Date(earliest.Year(), earliest.Month(), earliest.Day(), 23, 59, 59, 999999999, earliest.Location())
		case 3: // hour
			latest = time.Date(earliest.Year(), earliest.Month(), earliest.Day(), earliest.Hour(), 59, 59, 999999999, earliest.Location())
		case 4: // minute
			latest = time.Date(earliest.Year(), earliest.Month(), earliest.Day(), earliest.Hour(), earliest.Minute(), 59, 999999999, earliest.Location())
		case 5: // second
			latest = time.Date(earliest.Year(), earliest.Month(), earliest.Day(), earliest.Hour(), earliest.Minute(), earliest.Second(), 999999999, earliest.Location())
		}
		return earliest, latest, true
	case fptypes.Date:
		earliest := t.ToTime()
		latest := earliest
		switch t.Precision() {
		case 0: // year
			latest = time.Date(t.Year(), 12, 31, 0, 0, 0, 0, time.UTC)
		case 1: // month
			m := t.Month()
			if m == 0 {
				m = 1
			}
			lastDay := time.Date(t.Year(), time.Month(m+1), 0, 0, 0, 0, 0, time.UTC)
			latest = lastDay
		}
		return earliest, latest, true
	default:
		tt, ok := toGoTime(v)
		return tt, tt, ok
	}
}

// DifferenceBetween is DifferenceBetweenAt with no evaluation offset, UTC.
func DifferenceBetween(low, high fptypes.Value, precision string) (fptypes.Value, error) {
	return DifferenceBetweenAt(low, high, precision, nil)
}

// DifferenceBetweenAt is the number of boundaries of the precision crossed from
// low to high, in an evaluation at the offset of evaluation (nil for UTC).
//
// CQL 1.5.3, Appendix B, Difference: "the number of boundaries crossed for the
// specified precision", "any fractional boundaries are dropped", Sunday the
// first day of a week, and DateTime values with different offsets "normalized to
// the timezone offset of the evaluation request timestamp, but only when the
// comparison precision is hours, minutes, seconds, or milliseconds". So years to
// days are counted on the fields as written, weeks on the Sundays that start
// them, and hours and finer on the instants in the evaluation's frame, each cut
// down to the unit before they are subtracted: 23:59:59.999 to 00:00:00.000 is
// one hour, one minute and one second.
//
// An operand coarser than the precision (a second and a millisecond count as two
// here, as both reference engines read it) makes the answer an uncertainty, from
// the low bound's latest moment to the high bound's earliest, to the low
// bound's earliest to the high bound's latest (Language Semantics, Determining
// Difference and Duration). Two operands at the precision or finer are exact,
// where `duration` is uncertain for any operand short of a millisecond.
//
// The result is an Integer, and a count that cannot be represented as one is
// null: see inIntegerRange.
func DifferenceBetweenAt(low, high fptypes.Value, precision string, evaluation *time.Location) (fptypes.Value, error) {
	v, err := differenceBetweenAt(low, high, precision, evaluation)
	if err != nil {
		return nil, err
	}
	return inIntegerRange(v), nil
}

func differenceBetweenAt(low, high fptypes.Value, precision string, evaluation *time.Location) (fptypes.Value, error) {
	if low == nil || high == nil {
		return nil, nil
	}
	unit := durationPrecisionIndex(precision)
	if unit < 0 {
		return durationBetweenAt(low, high, precision, evaluation)
	}
	lowEarliest, lowLatest, okLow := differenceRange(low, evaluation)
	highEarliest, highLatest, okHigh := differenceRange(high, evaluation)
	if !okLow || !okHigh {
		return nil, nil
	}
	_, lowIsTime := low.(fptypes.Time)
	_, highIsTime := high.(fptypes.Time)
	if differencePrecision(low) < unit || differencePrecision(high) < unit {
		smallest := boundariesCrossed(lowLatest, highEarliest, precision, evaluation, lowIsTime || highIsTime)
		largest := boundariesCrossed(lowEarliest, highLatest, precision, evaluation, lowIsTime || highIsTime)
		if smallest == largest {
			return fptypes.NewInteger(smallest), nil
		}
		return cqltypes.NewUncertainty(fptypes.NewInteger(smallest), fptypes.NewInteger(largest)), nil
	}
	return fptypes.NewInteger(boundariesCrossed(lowEarliest, highEarliest, precision, evaluation, lowIsTime || highIsTime)), nil
}

// differencePrecision is the index of the finest component a value states,
// 0 for the year to 6 for the millisecond, the second and the millisecond apart.
func differencePrecision(v fptypes.Value) int {
	switch t := v.(type) {
	case fptypes.DateTime:
		return int(t.Precision())
	case fptypes.Time:
		return int(t.Precision()) + 3
	case fptypes.Date:
		return int(t.Precision())
	default:
		return 6
	}
}

// differenceRange is the earliest and latest moment a value could be, each as a
// wall clock in the offset it is written at or read at. A Date is read as a
// DateTime to the day; a Time has no date and no offset.
func differenceRange(v fptypes.Value, frame *time.Location) (earliest, latest time.Time, ok bool) {
	switch t := v.(type) {
	case fptypes.DateTime:
		return temporalRange(t, frame)
	case fptypes.Date:
		earliest = t.ToTime()
		latest = earliest
		switch t.Precision() {
		case 0:
			latest = time.Date(t.Year(), 12, 31, 0, 0, 0, 0, time.UTC)
		case 1:
			latest = time.Date(t.Year(), time.Month(max(t.Month(), 1))+1, 0, 0, 0, 0, 0, time.UTC)
		}
		latest = time.Date(latest.Year(), latest.Month(), latest.Day(), 23, 59, 59, 999999999, time.UTC)
		return earliest, latest, true
	case fptypes.Time:
		earliest = time.Date(0, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Millisecond()*1e6, time.UTC)
		latest = earliest
		switch t.Precision() {
		case 0: // hour
			latest = time.Date(0, 1, 1, t.Hour(), 59, 59, 999999999, time.UTC)
		case 1: // minute
			latest = time.Date(0, 1, 1, t.Hour(), t.Minute(), 59, 999999999, time.UTC)
		case 2: // second
			latest = time.Date(0, 1, 1, t.Hour(), t.Minute(), t.Second(), 999999999, time.UTC)
		}
		return earliest, latest, true
	default:
		return time.Time{}, time.Time{}, false
	}
}

// boundariesCrossed counts the boundaries of the unit from a to b, as Appendix B
// defines Difference. Years to days are read on the fields a and b are written
// with; hours and finer are read in the frame, unless both are a Time.
func boundariesCrossed(a, b time.Time, precision string, frame *time.Location, isTime bool) int64 {
	switch precision {
	case "year", "years":
		return int64(b.Year() - a.Year())
	case "month", "months":
		return int64((b.Year()-a.Year())*12 + int(b.Month()) - int(a.Month()))
	case "week", "weeks":
		return (sundayOf(b) - sundayOf(a)) / 7
	case "day", "days":
		return dayNumber(b) - dayNumber(a)
	}
	if !isTime {
		if frame == nil {
			frame = time.UTC
		}
		a, b = a.In(frame), b.In(frame)
	}
	wall := func(t time.Time, nanos int) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/nanos*nanos, time.UTC)
	}
	switch precision {
	case "hour", "hours":
		return (dayNumber(b)*24 + int64(b.Hour())) - (dayNumber(a)*24 + int64(a.Hour()))
	case precMinute, "minutes":
		return (dayNumber(b)*1440 + int64(b.Hour()*60+b.Minute())) - (dayNumber(a)*1440 + int64(a.Hour()*60+a.Minute()))
	case precSecond, "seconds":
		return wall(b, 1e9).Unix() - wall(a, 1e9).Unix()
	default:
		return wall(b, 1e6).UnixMilli() - wall(a, 1e6).UnixMilli()
	}
}

// dayNumber is the number of days from 1970-01-01 to the day t is written on.
func dayNumber(t time.Time) int64 {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix() / 86400
}

// sundayOf is the day number of the Sunday that starts the week of t.
func sundayOf(t time.Time) int64 {
	return dayNumber(t) - int64(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Weekday())
}

// toGoTime converts a fptypes.Value to a time.Time using the type's built-in ToTime method.
// ok is false for a value with no instant. The zero time.Time is not that
// signal: it is 0001-01-01T00:00:00Z, the first instant of the calendar.
func toGoTime(v fptypes.Value) (time.Time, bool) {
	switch t := v.(type) {
	case fptypes.DateTime:
		// EffectiveOffset rather than ToTime alone. ToTime places a value that
		// states no offset at UTC, which ignores any default the caller supplied —
		// so at an evaluation offset of UTC-5 the comparison operators answered
		// from the default while every duration here still measured from UTC. The
		// engine said a bare value was both later than an instant and two hours
		// earlier than it.
		//
		// This is the single conversion the duration and difference operators go
		// through, which is why the fix belongs here rather than in each of them.
		return dateTimeInstant(t), true
	case fptypes.Date:
		return t.ToTime(), true
	case fptypes.Time:
		// Time has no ToTime() method; construct from components
		return time.Date(0, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Millisecond()*1e6, time.UTC), true
	default:
		// Fall back to string parsing
		return parseTemporal(v)
	}
}

// MillisecondsBetween calculates the number of milliseconds between two datetimes.
// Like the other XBetween functions it gives the count as it is, without the
// guarantee of an Integer: past 2^31-1 it is a number no Integer holds. That
// guarantee is DurationBetweenAt's and DifferenceBetweenAt's (plan 022).
func MillisecondsBetween(low, high fptypes.Value) (fptypes.Value, error) {
	seconds, nanos, ok := elapsed(low, high)
	if !ok {
		return nil, nil
	}
	return fptypes.NewInteger(seconds*1000 + nanos/1e6), nil
}

// DateAdd adds a duration to a date/time value.
// Uses the fptypes built-in AddDuration which preserves precision and clamps the
// day to the end of the month a year or month shift lands in.
// Handles CQL semantics: when the unit is finer than the operand's precision,
// convert to the operand's precision first (e.g., 25 months on a year-only Date = 2 years).
// UCUM codes are normalized to calendar keywords first, so `5 'd'` and `5 days` agree;
// a UCUM year or month has no exact calendar meaning and is refused.
func DateAdd(operand fptypes.Value, amount int, precision string) (fptypes.Value, error) {
	if operand == nil {
		return nil, nil
	}
	unit, err := normalizeDurationUnit(precision)
	if err != nil {
		return nil, err
	}
	switch t := operand.(type) {
	case fptypes.DateTime:
		convertedAmount, convertedUnit := convertToMatchPrecision(amount, unit, int(t.Precision()), true)
		result, err := t.AddDuration(convertedAmount, convertedUnit)
		if err != nil {
			return nil, err
		}
		// Validate result year is in valid range
		if result.Year() < 1 || result.Year() > 9999 {
			return nil, fmt.Errorf("datetime addition results in year %d which is out of range [1, 9999]", result.Year())
		}
		return result, nil
	case fptypes.Date:
		convertedAmount, convertedUnit := convertToMatchPrecision(amount, unit, int(t.Precision()), false)
		result, err := t.AddDuration(convertedAmount, convertedUnit)
		if err != nil {
			return nil, err
		}
		// Validate result year is in valid range
		if result.Year() < 1 || result.Year() > 9999 {
			return nil, fmt.Errorf("date addition results in year %d which is out of range [1, 9999]", result.Year())
		}
		return result, nil
	case fptypes.Time:
		// Handle Time type natively to preserve precision
		h, m, s, ms := t.Hour(), t.Minute(), t.Second(), t.Millisecond()
		prec := t.Precision()
		switch unit {
		case "hour", "hours":
			h += amount
		case precMinute, "minutes":
			m += amount
		case precSecond, "seconds":
			s += amount
		case precMillisecond, "milliseconds":
			ms += amount
		default:
			h += amount
		}
		// Normalize milliseconds -> seconds -> minutes -> hours
		if ms < 0 || ms >= 1000 {
			s += ms / 1000
			ms %= 1000
			if ms < 0 {
				ms += 1000
				s--
			}
		}
		if s < 0 || s >= 60 {
			m += s / 60
			s %= 60
			if s < 0 {
				s += 60
				m--
			}
		}
		if m < 0 || m >= 60 {
			h += m / 60
			m %= 60
			if m < 0 {
				m += 60
				h--
			}
		}
		// Wrap hours into 0-23
		h %= 24
		if h < 0 {
			h += 24
		}
		// Build time string at the original precision
		var timeStr string
		switch prec {
		case fptypes.HourPrecision:
			timeStr = fmt.Sprintf("%02d", h)
		case fptypes.MinutePrecision:
			timeStr = fmt.Sprintf("%02d:%02d", h, m)
		case fptypes.SecondPrecision:
			timeStr = fmt.Sprintf("%02d:%02d:%02d", h, m, s)
		default:
			timeStr = fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
		}
		return fptypes.NewTime(timeStr)
	default:
		// For unknown types, fall back to time.Time approach
		tt, err := toTime(operand)
		if err != nil || tt.IsZero() {
			return nil, nil
		}
		var result time.Time
		switch unit {
		case "year", "years":
			result = tt.AddDate(amount, 0, 0)
		case "month", "months":
			result = tt.AddDate(0, amount, 0)
		case "week", "weeks":
			result = tt.AddDate(0, 0, amount*7)
		case "day", "days":
			result = tt.AddDate(0, 0, amount)
		case "hour", "hours":
			result = tt.Add(time.Duration(amount) * time.Hour)
		case precMinute, "minutes":
			result = tt.Add(time.Duration(amount) * time.Minute)
		case precSecond, "seconds":
			result = tt.Add(time.Duration(amount) * time.Second)
		case precMillisecond, "milliseconds":
			result = tt.Add(time.Duration(amount) * time.Millisecond)
		default:
			result = tt.AddDate(0, 0, amount)
		}
		return fptypes.NewTime(result.Format("15:04:05.000"))
	}
}

// convertToMatchPrecision converts a duration amount+unit to a coarser unit
// when the unit is finer than the operand's precision.
// For DateTime: precIdx 0=year,1=month,2=day,3=hour,4=minute,5=second,6=millis
// For Date: precIdx 0=year,1=month,2=day
func convertToMatchPrecision(amount int, unit string, precIdx int, isDateTime bool) (converted int, convertedUnit string) {
	unitIdx := unitPrecisionIndex(unit, isDateTime)
	if unitIdx < 0 || unitIdx <= precIdx {
		return amount, unit // unit is at or above operand precision — no conversion needed
	}

	// Unit is finer than operand precision. Convert by integer division.
	switch {
	case precIdx == 0 && (unit == "month" || unit == "months"):
		// months -> years: truncate toward zero
		return truncDiv(amount, 12), "years"
	case precIdx == 0 && (unit == "day" || unit == "days"):
		return truncDiv(amount, 365), "years"
	case precIdx == 0 && (unit == "week" || unit == "weeks"):
		return truncDiv(amount, 52), "years"
	case precIdx <= 1 && (unit == "day" || unit == "days"):
		return truncDiv(amount, 30), "months"
	case precIdx <= 1 && (unit == "week" || unit == "weeks"):
		return truncDiv(amount*7, 30), "months"
	case precIdx <= 2 && (unit == "hour" || unit == "hours"):
		return truncDiv(amount, 24), "days"
	case precIdx <= 3 && (unit == precMinute || unit == "minutes"):
		return truncDiv(amount, 60), "hours"
	case precIdx <= 4 && (unit == precSecond || unit == "seconds"):
		return truncDiv(amount, 60), "minutes"
	case precIdx <= 5 && (unit == precMillisecond || unit == "milliseconds"):
		return truncDiv(amount, 1000), "seconds"
	default:
		return amount, unit
	}
}

func unitPrecisionIndex(unit string, isDateTime bool) int {
	switch unit {
	case "year", "years":
		return 0
	case "month", "months":
		return 1
	case "week", "weeks":
		return 2 // same level as day
	case "day", "days":
		return 2
	case "hour", "hours":
		if isDateTime {
			return 3
		}
		return -1
	case precMinute, "minutes":
		if isDateTime {
			return 4
		}
		return -1
	case precSecond, "seconds":
		if isDateTime {
			return 5
		}
		return -1
	case precMillisecond, "milliseconds":
		if isDateTime {
			return 6
		}
		return -1
	default:
		return -1
	}
}

// truncDiv performs integer division truncating toward zero.
func truncDiv(a, b int) int {
	return a / b // Go integer division truncates toward zero
}

func intVal(v fptypes.Value) int {
	if v == nil {
		return 0
	}
	if iv, ok := v.(fptypes.Integer); ok {
		return int(iv.Value())
	}
	return 0
}

func toTime(v fptypes.Value) (time.Time, error) { //nolint:unparam // error kept for future format additions
	t, _ := parseTemporal(v)
	return t, nil
}

// parseTemporal reads the string of a value in the common FHIR date and time
// layouts. ok is false for a nil value or one in none of them.
func parseTemporal(v fptypes.Value) (t time.Time, ok bool) {
	if v == nil {
		return time.Time{}, false
	}
	s := v.String()
	for _, layout := range []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"2006-01",
		"2006",
		"15:04:05.000",
		"15:04:05",
		"15:04",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
