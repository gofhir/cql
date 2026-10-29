package types

import (
	"errors"
	"fmt"
	"math"

	"github.com/shopspring/decimal"

	fptypes "github.com/gofhir/fhirpath/types"
)

// Successor and Predecessor are the neighboring values of an ordered type, and
// they live here because an interval needs them to say where it starts.
//
// `start of Interval(1, 5)` is 2: an open boundary excludes its own value, so
// the first included point is the successor of it. That made three copies of the
// same arithmetic — one in the evaluator for the operators, one nowhere at all
// for funcs.IntervalStartOf, which returned the raw boundary — and Interval.Equal
// could reach none of them, so two intervals with the same points compared
// unequal whenever they were written with different boundaries.
//
// decimalStep is the smallest difference CQL defines between two Decimals.
var decimalStep = decimal.RequireFromString("0.00000001")

// HasSuccessor reports whether a value's type defines a successor, which is what
// an open interval boundary needs in order to name its first included point.
// String and the like have none, so an open boundary over them stands for itself.
func HasSuccessor(v fptypes.Value) bool {
	switch v.(type) {
	case fptypes.Integer, fptypes.Decimal, fptypes.Date, fptypes.DateTime,
		fptypes.Time, fptypes.Quantity:
		return true
	}
	return false
}

// Successor returns the next value of an ordered type.
func Successor(v fptypes.Value) (fptypes.Value, error) { return step(v, 1) }

// ErrPastRange is a successor or predecessor past the range of its type: of
// the largest Long, of the last Date, DateTime or Time, or — the evaluator's
// to tell, from the static type — of the largest Integer. The operators
// answer it with null, as the specification does (CQL 1.5.3, Appendix B,
// Successor: "If the argument is already the maximum value for the type, a
// null is returned"; plan 013); an interval boundary reads it as no step to
// take. A Decimal has no such edge here: its range is the operator's to hold.
var ErrPastRange = errors.New("the result is past the range of its type")

// Predecessor returns the previous value of an ordered type.
func Predecessor(v fptypes.Value) (fptypes.Value, error) { return step(v, -1) }

// step moves one value along, forwards or back.
//
// Temporal types step at their own precision rather than by a fixed unit: the
// successor of @2020-01 is @2020-02, not @2020-01-02, or the result claims a
// precision the operand never had.
func step(v fptypes.Value, delta int) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	switch t := v.(type) {
	case fptypes.Integer:
		// Past the largest or the least value is an error, as it is for the
		// other types; the int64 wrapped there. The 32-bit edge of an Integer
		// is the static type's, and the evaluator's to catch.
		if delta > 0 && t.Value() == math.MaxInt64 {
			return nil, fmt.Errorf("successor of %d: %w", t.Value(), ErrPastRange)
		}
		if delta < 0 && t.Value() == math.MinInt64 {
			return nil, fmt.Errorf("predecessor of %d: %w", t.Value(), ErrPastRange)
		}
		return fptypes.NewInteger(t.Value() + int64(delta)), nil

	case fptypes.Decimal:
		if delta > 0 {
			return newDecimal(t.Value().Add(decimalStep))
		}
		return newDecimal(t.Value().Sub(decimalStep))

	case fptypes.DateTime:
		unit := DateTimeUnit(t.Precision())
		if delta > 0 {
			result, err := t.AddDuration(1, unit)
			return calendarStep(t, delta, result, err)
		}
		result, err := t.SubtractDuration(1, unit)
		return calendarStep(t, delta, result, err)

	case fptypes.Date:
		// The same range as a DateTime: stepping past it produced year 0 and
		// year 10000 rather than saying there is no such date.
		unit := DateUnit(t.Precision())
		if delta > 0 {
			result, err := t.AddDuration(1, unit)
			return calendarStep(t, delta, result, err)
		}
		result, err := t.SubtractDuration(1, unit)
		return calendarStep(t, delta, result, err)

	case fptypes.Time:
		result := AdjustTime(t, delta)
		if result == nil {
			return nil, stepError(t, delta)
		}
		return result, nil

	case fptypes.Quantity:
		if delta > 0 {
			return fptypes.NewQuantityFromDecimal(t.Value().Add(decimalStep), t.Unit()), nil
		}
		return fptypes.NewQuantityFromDecimal(t.Value().Sub(decimalStep), t.Unit()), nil
	}
	return nil, fmt.Errorf("successor/predecessor not supported for %s", v.Type())
}

// calendarStep reads the step of a Date or a DateTime: fhirpath refuses a
// date outside the years 1 to 9999 with ErrInvalidTemporal, and that is the
// only way a step of one unit of the value's own precision can fail — the
// unit is never finer than the value, so no day is off the calendar but past
// the last and the first. That refusal is the step past the range.
func calendarStep(from fptypes.Value, delta int, result fptypes.Value, err error) (fptypes.Value, error) {
	if errors.Is(err, fptypes.ErrInvalidTemporal) {
		return nil, fmt.Errorf("%w: %w", stepError(from, delta), err)
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// stepError is ErrPastRange, naming the value a step left from.
func stepError(v fptypes.Value, delta int) error {
	if delta > 0 {
		return fmt.Errorf("successor of %v: %w", v, ErrPastRange)
	}
	return fmt.Errorf("predecessor of %v: %w", v, ErrPastRange)
}

func newDecimal(d decimal.Decimal) (fptypes.Value, error) {
	v, err := fptypes.NewDecimal(d.String())
	if err != nil {
		return nil, err
	}
	return v, nil
}

// DateTimeUnit is the duration unit matching a DateTime's precision.
func DateTimeUnit(prec fptypes.DateTimePrecision) string {
	switch prec {
	case fptypes.DTYearPrecision:
		return "year"
	case fptypes.DTMonthPrecision:
		return "month"
	case fptypes.DTDayPrecision:
		return "day"
	case fptypes.DTHourPrecision:
		return "hour"
	case fptypes.DTMinutePrecision:
		return "minute"
	case fptypes.DTSecondPrecision:
		return "second"
	case fptypes.DTMillisPrecision:
		return "millisecond"
	default:
		return "day"
	}
}

// DateUnit is the duration unit matching a Date's precision.
func DateUnit(prec fptypes.DatePrecision) string {
	switch prec {
	case fptypes.YearPrecision:
		return "year"
	case fptypes.MonthPrecision:
		return "month"
	default:
		return "day"
	}
}

// AdjustTime moves a Time by whole units of its own precision, carrying and
// borrowing across the fields, and answers nil where that would leave the day.
func AdjustTime(t fptypes.Time, delta int) fptypes.Value {
	h, m, s, ms := t.Hour(), t.Minute(), t.Second(), t.Millisecond()
	switch t.Precision() {
	case fptypes.MillisPrecision:
		ms += delta
	case fptypes.SecondPrecision:
		s += delta
	case fptypes.MinutePrecision:
		m += delta
	case fptypes.HourPrecision:
		h += delta
	default:
		ms += delta
	}
	if ms < 0 {
		ms += 1000
		s--
	} else if ms >= 1000 {
		ms -= 1000
		s++
	}
	if s < 0 {
		s += 60
		m--
	} else if s >= 60 {
		s -= 60
		m++
	}
	if m < 0 {
		m += 60
		h--
	} else if m >= 60 {
		m -= 60
		h++
	}
	if h < 0 || h > 23 {
		return nil
	}
	// Rendered at the operand's precision and parsed back, so the result carries
	// the precision it was stepped at rather than a wider one.
	var str string
	switch t.Precision() {
	case fptypes.HourPrecision:
		str = fmt.Sprintf("T%02d", h)
	case fptypes.MinutePrecision:
		str = fmt.Sprintf("T%02d:%02d", h, m)
	case fptypes.SecondPrecision:
		str = fmt.Sprintf("T%02d:%02d:%02d", h, m, s)
	default:
		str = fmt.Sprintf("T%02d:%02d:%02d.%03d", h, m, s, ms)
	}
	result, err := fptypes.NewTime(str)
	if err != nil {
		return t // unreachable for values built from a valid Time
	}
	return result
}
