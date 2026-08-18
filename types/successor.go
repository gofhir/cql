package types

import (
	"fmt"

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
			if err != nil {
				return nil, err
			}
			if result.Year() > 9999 {
				return nil, fmt.Errorf("successor overflow: DateTime exceeds maximum")
			}
			return result, nil
		}
		result, err := t.SubtractDuration(1, unit)
		if err != nil {
			return nil, err
		}
		if result.Year() < 1 {
			return nil, fmt.Errorf("predecessor underflow: DateTime below minimum")
		}
		return result, nil

	case fptypes.Date:
		// Guarded the way the DateTime branch above is: a Date has the same
		// representable range, and stepping past it produced year 0 and year
		// 10000 rather than saying there is no such date.
		unit := DateUnit(t.Precision())
		if delta > 0 {
			result, err := t.AddDuration(1, unit)
			if err != nil {
				return nil, err
			}
			if result.Year() > 9999 {
				return nil, fmt.Errorf("successor overflow: Date exceeds maximum")
			}
			return result, nil
		}
		result, err := t.SubtractDuration(1, unit)
		if err != nil {
			return nil, err
		}
		if result.Year() < 1 {
			return nil, fmt.Errorf("predecessor underflow: Date below minimum")
		}
		return result, nil

	case fptypes.Time:
		result := AdjustTime(t, delta)
		if result == nil {
			if delta > 0 {
				return nil, fmt.Errorf("successor overflow: Time exceeds maximum")
			}
			return nil, fmt.Errorf("predecessor underflow: Time below minimum")
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
