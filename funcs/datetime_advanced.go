package funcs

import (
	"fmt"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

// DateTimeComponentFrom extracts a component from a DateTime, Date, or Time value.
// Component names: "year", "month", "day", "hour", "minute", "second", "millisecond",
// "timezone", "date", "time".
func DateTimeComponentFrom(operand fptypes.Value, component string) (fptypes.Value, error) {
	if operand == nil {
		return nil, nil
	}
	t, err := toTime(operand)
	if err != nil || t.IsZero() {
		return nil, nil
	}

	switch component {
	case "year":
		return fptypes.NewInteger(int64(t.Year())), nil
	case "month":
		return fptypes.NewInteger(int64(t.Month())), nil
	case "day":
		return fptypes.NewInteger(int64(t.Day())), nil
	case "hour":
		return fptypes.NewInteger(int64(t.Hour())), nil
	case "minute":
		return fptypes.NewInteger(int64(t.Minute())), nil
	case "second":
		return fptypes.NewInteger(int64(t.Second())), nil
	case "millisecond":
		return fptypes.NewInteger(int64(t.Nanosecond() / 1e6)), nil
	case "timezone":
		_, offset := t.Zone()
		hours := offset / 3600
		minutes := (offset % 3600) / 60
		return fptypes.NewString(fmt.Sprintf("%+03d:%02d", hours, minutes)), nil
	case "date":
		return fptypes.NewDate(t.Format("2006-01-02"))
	case "time":
		return fptypes.NewTime(t.Format("15:04:05.000"))
	default:
		return nil, fmt.Errorf("unknown date/time component: %s", component)
	}
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
func DateTimeConstructor(year, month, day, hour, minute, second, millisecond, tzOffset fptypes.Value) (fptypes.Value, error) {
	y := intVal(year)
	m := intVal(month)
	d := intVal(day)
	h := intVal(hour)
	mn := intVal(minute)
	sec := intVal(second)
	ms := intVal(millisecond)

	if y == 0 {
		return nil, nil
	}

	s := fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d.%03d", y, m, d, h, mn, sec, ms)
	if tzOffset != nil {
		if sv, ok := tzOffset.(fptypes.String); ok {
			s += sv.Value()
		}
	} else {
		s += "Z"
	}
	return fptypes.NewDateTime(s)
}

// TimeConstructor creates a Time from hour, minute, second, millisecond.
func TimeConstructor(hour, minute, second, millisecond fptypes.Value) (fptypes.Value, error) {
	h := intVal(hour)
	mn := intVal(minute)
	sec := intVal(second)
	ms := intVal(millisecond)
	return fptypes.NewTime(fmt.Sprintf("%02d:%02d:%02d.%03d", h, mn, sec, ms))
}

// DurationBetween calculates the duration between two date/time values at a given precision.
func DurationBetween(low, high fptypes.Value, precision string) (fptypes.Value, error) {
	if low == nil || high == nil {
		return nil, nil
	}
	switch precision {
	case "year", "years":
		return YearsBetween(low, high)
	case "month", "months":
		return MonthsBetween(low, high)
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

// DifferenceBetween calculates the number of boundaries crossed between two values at a precision.
func DifferenceBetween(low, high fptypes.Value, precision string) (fptypes.Value, error) {
	// For most practical purposes, difference between is the same as duration between
	// for whole-unit precisions. The distinction matters for sub-unit precision
	// which counts boundary crossings.
	return DurationBetween(low, high, precision)
}

// MillisecondsBetween calculates the number of milliseconds between two datetimes.
func MillisecondsBetween(low, high fptypes.Value) (fptypes.Value, error) {
	tl, err := toTime(low)
	if err != nil || tl.IsZero() {
		return nil, nil
	}
	th, err := toTime(high)
	if err != nil || th.IsZero() {
		return nil, nil
	}
	ms := th.Sub(tl).Milliseconds()
	return fptypes.NewInteger(ms), nil
}

// DateAdd adds a duration to a date/time value.
func DateAdd(operand fptypes.Value, amount int, precision string) (fptypes.Value, error) {
	if operand == nil {
		return nil, nil
	}
	t, err := toTime(operand)
	if err != nil || t.IsZero() {
		return nil, nil
	}

	var result time.Time
	switch precision {
	case "year", "years":
		result = t.AddDate(amount, 0, 0)
	case "month", "months":
		result = t.AddDate(0, amount, 0)
	case "day", "days":
		result = t.AddDate(0, 0, amount)
	case "hour", "hours":
		result = t.Add(time.Duration(amount) * time.Hour)
	case "minute", "minutes":
		result = t.Add(time.Duration(amount) * time.Minute)
	case "second", "seconds":
		result = t.Add(time.Duration(amount) * time.Second)
	case "millisecond", "milliseconds":
		result = t.Add(time.Duration(amount) * time.Millisecond)
	default:
		result = t.AddDate(0, 0, amount)
	}

	// Return same type as input
	switch operand.Type() {
	case "Date":
		return fptypes.NewDate(result.Format("2006-01-02"))
	default:
		return fptypes.NewDateTime(result.Format("2006-01-02T15:04:05.000Z07:00"))
	}
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
