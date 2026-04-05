package funcs

import (
	"fmt"
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
			return fptypes.NewInteger(int64(t.Hour())), nil
		}
		if t, ok := operand.(fptypes.Time); ok {
			return fptypes.NewInteger(int64(t.Hour())), nil
		}
	case "minute":
		if t, ok := operand.(fptypes.DateTime); ok {
			return fptypes.NewInteger(int64(t.Minute())), nil
		}
		if t, ok := operand.(fptypes.Time); ok {
			return fptypes.NewInteger(int64(t.Minute())), nil
		}
	case "second":
		if t, ok := operand.(fptypes.DateTime); ok {
			return fptypes.NewInteger(int64(t.Second())), nil
		}
		if t, ok := operand.(fptypes.Time); ok {
			return fptypes.NewInteger(int64(t.Second())), nil
		}
	case "millisecond":
		if t, ok := operand.(fptypes.DateTime); ok {
			return fptypes.NewInteger(int64(t.Millisecond())), nil
		}
		if t, ok := operand.(fptypes.Time); ok {
			return fptypes.NewInteger(int64(t.Millisecond())), nil
		}
	case "timezone":
		// "timezone" is not a valid CQL component; use "timezoneoffset" instead
		return nil, fmt.Errorf("'timezone' is not a valid date/time component; use 'timezoneoffset' instead")
	case "timezoneoffset":
		if dt, ok := operand.(fptypes.DateTime); ok {
			if dt.HasTZ() {
				offset := dt.TZOffset()
				return fptypes.NewDecimalFromFloat(float64(offset) / 60.0), nil
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
		if sv, ok := tzOffset.(fptypes.String); ok {
			s += sv.Value()
		} else if di, ok := tzOffset.(fptypes.Decimal); ok {
			// Handle decimal timezone offset (e.g., -6.0 -> "-06:00")
			f, _ := di.Value().Float64()
			hours := int(f)
			mins := int((f - float64(hours)) * 60)
			if mins < 0 {
				mins = -mins
			}
			s += fmt.Sprintf("%+03d:%02d", hours, mins)
		} else if ii, ok := tzOffset.(fptypes.Integer); ok {
			// Handle integer timezone offset (e.g., 1 -> "+01:00")
			hours := int(ii.Value())
			s += fmt.Sprintf("%+03d:00", hours)
		}
	}
	return fptypes.NewDateTime(s)
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
	return fptypes.NewTime(s)
}

// DurationBetween calculates the duration between two date/time values at a given precision.
// When operands have different precisions, returns an Interval representing the uncertainty range.
func DurationBetween(low, high fptypes.Value, precision string) (fptypes.Value, error) {
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
		return durationBetweenUncertain(low, high, precision, precIdx)
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
	case "minute", "minutes":
		return MinutesBetween(low, high)
	case "second", "seconds":
		return SecondsBetween(low, high)
	case "millisecond", "milliseconds":
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
	case "minute", "minutes":
		return 4
	case "second", "seconds":
		return 5
	case "millisecond", "milliseconds":
		return 6
	default:
		return -1
	}
}

// valuePrecisionIndex returns the precision level of a temporal value.
func valuePrecisionIndex(v fptypes.Value) int {
	switch t := v.(type) {
	case fptypes.DateTime:
		return int(t.Precision())
	case fptypes.Date:
		return int(t.Precision())
	case fptypes.Time:
		return int(t.Precision()) + 3
	default:
		return 6 // assume full precision for unknown
	}
}

// durationBetweenUncertain computes the min/max range of the duration
// when one or both operands have insufficient precision.
func durationBetweenUncertain(low, high fptypes.Value, precision string, precIdx int) (fptypes.Value, error) {
	// Compute the earliest and latest possible time.Time for each operand
	lowEarliest, lowLatest := temporalRange(low)
	highEarliest, highLatest := temporalRange(high)

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

	return cqltypes.NewInterval(
		fptypes.NewInteger(minInt),
		fptypes.NewInteger(maxInt),
		true, true,
	), nil
}

// temporalRange returns the earliest and latest possible time.Time for a temporal value.
func temporalRange(v fptypes.Value) (time.Time, time.Time) {
	switch t := v.(type) {
	case fptypes.DateTime:
		earliest := t.ToTime()
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
		return earliest, latest
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
		return earliest, latest
	default:
		tt := toGoTime(v)
		return tt, tt
	}
}

// DifferenceBetween calculates the number of boundaries crossed between two values at a precision.
// Unlike DurationBetween, this counts the number of calendar boundaries crossed.
// For day and coarser precision, uses nominal (stated) values rather than UTC-normalized values.
func DifferenceBetween(low, high fptypes.Value, precision string) (fptypes.Value, error) {
	if low == nil || high == nil {
		return nil, nil
	}

	switch precision {
	case "year", "years":
		lComps, _ := nominalComponents(low)
		hComps, _ := nominalComponents(high)
		if lComps == nil || hComps == nil {
			return nil, nil
		}
		return fptypes.NewInteger(int64(hComps[0] - lComps[0])), nil
	case "month", "months":
		lComps, _ := nominalComponents(low)
		hComps, _ := nominalComponents(high)
		if lComps == nil || hComps == nil {
			return nil, nil
		}
		months := (hComps[0]-lComps[0])*12 + hComps[1] - lComps[1]
		return fptypes.NewInteger(int64(months)), nil
	case "week", "weeks":
		lComps, _ := nominalComponents(low)
		hComps, _ := nominalComponents(high)
		if lComps == nil || hComps == nil {
			return nil, nil
		}
		lDay := time.Date(lComps[0], time.Month(lComps[1]), lComps[2], 0, 0, 0, 0, time.UTC)
		hDay := time.Date(hComps[0], time.Month(hComps[1]), hComps[2], 0, 0, 0, 0, time.UTC)
		days := int(hDay.Sub(lDay).Hours() / 24)
		weeks := days / 7
		return fptypes.NewInteger(int64(weeks)), nil
	case "day", "days":
		lComps, _ := nominalComponents(low)
		hComps, _ := nominalComponents(high)
		if lComps == nil || hComps == nil {
			return nil, nil
		}
		lDay := time.Date(lComps[0], time.Month(lComps[1]), lComps[2], 0, 0, 0, 0, time.UTC)
		hDay := time.Date(hComps[0], time.Month(hComps[1]), hComps[2], 0, 0, 0, 0, time.UTC)
		days := int(hDay.Sub(lDay).Hours() / 24)
		return fptypes.NewInteger(int64(days)), nil
	case "hour", "hours":
		tl := toGoTime(low)
		th := toGoTime(high)
		if tl.IsZero() || th.IsZero() {
			return nil, nil
		}
		hours := int(th.Sub(tl).Hours())
		return fptypes.NewInteger(int64(hours)), nil
	case "minute", "minutes":
		tl := toGoTime(low)
		th := toGoTime(high)
		if tl.IsZero() || th.IsZero() {
			return nil, nil
		}
		minutes := int(th.Sub(tl).Minutes())
		return fptypes.NewInteger(int64(minutes)), nil
	case "second", "seconds":
		tl := toGoTime(low)
		th := toGoTime(high)
		if tl.IsZero() || th.IsZero() {
			return nil, nil
		}
		seconds := int(th.Sub(tl).Seconds())
		return fptypes.NewInteger(int64(seconds)), nil
	case "millisecond", "milliseconds":
		tl := toGoTime(low)
		th := toGoTime(high)
		if tl.IsZero() || th.IsZero() {
			return nil, nil
		}
		ms := th.Sub(tl).Milliseconds()
		return fptypes.NewInteger(ms), nil
	default:
		return DurationBetween(low, high, precision)
	}
}

// nominalComponents extracts the nominal (stated) year, month, day, hour, minute, second, millis
// from a temporal value without timezone normalization.
func nominalComponents(v fptypes.Value) ([]int, int) {
	switch t := v.(type) {
	case fptypes.DateTime:
		m := t.Month()
		if m == 0 {
			m = 1
		}
		d := t.Day()
		if d == 0 {
			d = 1
		}
		return []int{t.Year(), m, d, t.Hour(), t.Minute(), t.Second(), t.Millisecond()}, int(t.Precision())
	case fptypes.Date:
		m := t.Month()
		if m == 0 {
			m = 1
		}
		d := t.Day()
		if d == 0 {
			d = 1
		}
		return []int{t.Year(), m, d, 0, 0, 0, 0}, int(t.Precision())
	case fptypes.Time:
		return []int{0, 0, 0, t.Hour(), t.Minute(), t.Second(), t.Millisecond()}, int(t.Precision()) + 3
	default:
		return nil, -1
	}
}

// toGoTime converts a fptypes.Value to a time.Time using the type's built-in ToTime method.
func toGoTime(v fptypes.Value) time.Time {
	switch t := v.(type) {
	case fptypes.DateTime:
		return t.ToTime()
	case fptypes.Date:
		return t.ToTime()
	case fptypes.Time:
		// Time has no ToTime() method; construct from components
		return time.Date(0, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Millisecond()*1e6, time.UTC)
	default:
		// Fall back to string parsing
		tt, _ := toTime(v)
		return tt
	}
}

// MillisecondsBetween calculates the number of milliseconds between two datetimes.
func MillisecondsBetween(low, high fptypes.Value) (fptypes.Value, error) {
	tl := toGoTime(low)
	th := toGoTime(high)
	if tl.IsZero() || th.IsZero() {
		return nil, nil
	}
	ms := th.Sub(tl).Milliseconds()
	return fptypes.NewInteger(ms), nil
}

// DateAdd adds a duration to a date/time value.
// Uses the fptypes built-in AddDuration which preserves precision.
// Handles CQL semantics: when the unit is finer than the operand's precision,
// convert to the operand's precision first (e.g., 25 months on a year-only Date = 2 years).
func DateAdd(operand fptypes.Value, amount int, precision string) (fptypes.Value, error) {
	if operand == nil {
		return nil, nil
	}
	unit := precision
	switch t := operand.(type) {
	case fptypes.DateTime:
		convertedAmount, convertedUnit := convertToMatchPrecision(amount, unit, int(t.Precision()), true)
		result := t.AddDuration(convertedAmount, convertedUnit)
		// Clamp day for month/year additions that overflow (e.g., Feb 29 + 1 year)
		if isMonthOrYearUnit(convertedUnit) && t.Day() > 0 {
			result = clampDateTimeDay(result, t.Day())
		}
		// Validate result year is in valid range
		if result.Year() < 1 || result.Year() > 9999 {
			return nil, fmt.Errorf("datetime addition results in year %d which is out of range [1, 9999]", result.Year())
		}
		return result, nil
	case fptypes.Date:
		convertedAmount, convertedUnit := convertToMatchPrecision(amount, unit, int(t.Precision()), false)
		result := t.AddDuration(convertedAmount, convertedUnit)
		// Clamp day for month/year additions that overflow
		if isMonthOrYearUnit(convertedUnit) && t.Day() > 0 {
			result = clampDateDay(result, t.Day())
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
		switch precision {
		case "hour", "hours":
			h += amount
		case "minute", "minutes":
			m += amount
		case "second", "seconds":
			s += amount
		case "millisecond", "milliseconds":
			ms += amount
		default:
			h += amount
		}
		// Normalize milliseconds -> seconds -> minutes -> hours
		if ms < 0 || ms >= 1000 {
			s += ms / 1000
			ms = ms % 1000
			if ms < 0 {
				ms += 1000
				s--
			}
		}
		if s < 0 || s >= 60 {
			m += s / 60
			s = s % 60
			if s < 0 {
				s += 60
				m--
			}
		}
		if m < 0 || m >= 60 {
			h += m / 60
			m = m % 60
			if m < 0 {
				m += 60
				h--
			}
		}
		// Wrap hours into 0-23
		h = h % 24
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
		switch precision {
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
		case "minute", "minutes":
			result = tt.Add(time.Duration(amount) * time.Minute)
		case "second", "seconds":
			result = tt.Add(time.Duration(amount) * time.Second)
		case "millisecond", "milliseconds":
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
func convertToMatchPrecision(amount int, unit string, precIdx int, isDateTime bool) (int, string) {
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
	case precIdx <= 3 && (unit == "minute" || unit == "minutes"):
		return truncDiv(amount, 60), "hours"
	case precIdx <= 4 && (unit == "second" || unit == "seconds"):
		return truncDiv(amount, 60), "minutes"
	case precIdx <= 5 && (unit == "millisecond" || unit == "milliseconds"):
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
	case "minute", "minutes":
		if isDateTime {
			return 4
		}
		return -1
	case "second", "seconds":
		if isDateTime {
			return 5
		}
		return -1
	case "millisecond", "milliseconds":
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

func isMonthOrYearUnit(unit string) bool {
	switch unit {
	case "year", "years", "month", "months":
		return true
	}
	return false
}

// clampDateTimeDay corrects day overflow after month/year addition.
// For example, adding 1 year to 2012-02-29 gives 2013-03-01 via Go's time.AddDate,
// but CQL expects 2013-02-28 (clamp to last day of target month).
func clampDateTimeDay(dt fptypes.DateTime, originalDay int) fptypes.DateTime {
	if dt.Day() < originalDay {
		// Day wrapped to next month. Go back to last day of previous month.
		// Reconstruct by going to day 0 of the current month (= last day of prev month).
		t := dt.ToTime()
		lastDay := time.Date(t.Year(), t.Month(), 0, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
		s := fmt.Sprintf("%04d-%02d-%02d", lastDay.Year(), int(lastDay.Month()), lastDay.Day())
		if dt.Precision() >= 3 { // DTHourPrecision
			s += fmt.Sprintf("T%02d", lastDay.Hour())
		}
		if dt.Precision() >= 4 { // DTMinutePrecision
			s += fmt.Sprintf(":%02d", lastDay.Minute())
		}
		if dt.Precision() >= 5 { // DTSecondPrecision
			s += fmt.Sprintf(":%02d", lastDay.Second())
		}
		if dt.Precision() >= 6 { // DTMillisPrecision
			s += fmt.Sprintf(".%03d", lastDay.Nanosecond()/1e6)
		}
		if dt.HasTZ() {
			offset := dt.TZOffset()
			if offset == 0 {
				s += "Z"
			} else {
				sign := "+"
				o := offset
				if o < 0 {
					sign = "-"
					o = -o
				}
				s += fmt.Sprintf("%s%02d:%02d", sign, o/60, o%60)
			}
		}
		result, err := fptypes.NewDateTime(s)
		if err == nil {
			return result
		}
	}
	return dt
}

// clampDateDay corrects day overflow after month/year addition for Date values.
func clampDateDay(d fptypes.Date, originalDay int) fptypes.Date {
	if d.Day() < originalDay && d.Day() > 0 {
		// Day wrapped to next month
		t := d.ToTime()
		lastDay := time.Date(t.Year(), t.Month(), 0, 0, 0, 0, 0, time.UTC)
		s := fmt.Sprintf("%04d-%02d-%02d", lastDay.Year(), int(lastDay.Month()), lastDay.Day())
		result, err := fptypes.NewDate(s)
		if err == nil {
			return result
		}
	}
	return d
}
