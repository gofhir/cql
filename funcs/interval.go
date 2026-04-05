package funcs

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

// isAmbiguousComparisonErr returns true if the error is an ambiguous temporal comparison.
func isAmbiguousComparisonErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "ambiguous comparison")
}

// IntervalContains checks if an interval contains a point.
func IntervalContains(interval cqltypes.Interval, point fptypes.Value) (fptypes.Value, error) {
	result, err := interval.Contains(point)
	if err != nil {
		if isAmbiguousComparisonErr(err) {
			return nil, nil
		}
		return nil, err
	}
	return fptypes.NewBoolean(result), nil
}

// IntervalIncludes checks if interval a includes interval b.
func IntervalIncludes(a, b cqltypes.Interval) (fptypes.Value, error) {
	result, err := a.Includes(b)
	if err != nil {
		if isAmbiguousComparisonErr(err) {
			return nil, nil
		}
		return nil, err
	}
	return fptypes.NewBoolean(result), nil
}

// IntervalIncludedIn checks if interval a is included in interval b.
func IntervalIncludedIn(a, b cqltypes.Interval) (fptypes.Value, error) {
	return IntervalIncludes(b, a)
}

// IntervalOverlaps checks if two intervals overlap.
func IntervalOverlaps(a, b cqltypes.Interval) (fptypes.Value, error) {
	result, err := a.Overlaps(b)
	if err != nil {
		if isAmbiguousComparisonErr(err) {
			return nil, nil
		}
		return nil, err
	}
	return fptypes.NewBoolean(result), nil
}

// IntervalStartOf returns the low boundary of an interval.
func IntervalStartOf(interval cqltypes.Interval) fptypes.Value {
	return interval.Low
}

// IntervalEndOf returns the high boundary of an interval.
func IntervalEndOf(interval cqltypes.Interval) fptypes.Value {
	return interval.High
}

// IntervalUnion returns the union of two intervals.
// Returns null if the intervals do not overlap or meet.
func IntervalUnion(a, b cqltypes.Interval) (fptypes.Value, error) {
	// Check if intervals overlap or are adjacent (meet)
	overlaps, err := a.Overlaps(b)
	if err != nil {
		if isAmbiguousComparisonErr(err) {
			return nil, nil
		}
		return nil, err
	}
	meets := false
	if !overlaps {
		meets = intervalEndMeetsStart(a.High, a.HighClosed, b.Low, b.LowClosed) ||
			intervalEndMeetsStart(b.High, b.HighClosed, a.Low, a.LowClosed)
	}
	if !overlaps && !meets {
		return nil, nil // Non-overlapping, non-adjacent intervals → null
	}

	// Take min low, max high
	low := a.Low
	lowClosed := a.LowClosed
	if a.Low != nil && b.Low != nil {
		cmp, err := compareVals(a.Low, b.Low)
		if err != nil {
			return nil, err
		}
		if cmp > 0 {
			low = b.Low
			lowClosed = b.LowClosed
		}
	} else if a.Low == nil {
		low = b.Low
		lowClosed = b.LowClosed
	}
	high := a.High
	highClosed := a.HighClosed
	if a.High != nil && b.High != nil {
		cmp, err := compareVals(a.High, b.High)
		if err != nil {
			return nil, err
		}
		if cmp < 0 {
			high = b.High
			highClosed = b.HighClosed
		}
	} else if a.High == nil {
		high = b.High
		highClosed = b.HighClosed
	}
	return cqltypes.NewInterval(low, high, lowClosed, highClosed), nil
}

// IntervalIntersect returns the intersection of two intervals.
func IntervalIntersect(a, b cqltypes.Interval) (fptypes.Value, error) {
	// Take max low — if either is null (unknown), result low is null (unknown)
	low := a.Low
	lowClosed := a.LowClosed
	if a.Low != nil && b.Low != nil {
		cmp, err := compareVals(a.Low, b.Low)
		if err != nil {
			if isAmbiguousComparisonErr(err) {
				return nil, nil
			}
			return nil, err
		}
		if cmp < 0 {
			low = b.Low
			lowClosed = b.LowClosed
		}
	} else if a.Low == nil {
		low = nil
		lowClosed = a.LowClosed
	} else {
		// b.Low is nil
		low = nil
		lowClosed = b.LowClosed
	}

	// Take min high — if either is null (unknown), result high is null (unknown)
	high := a.High
	highClosed := a.HighClosed
	if a.High != nil && b.High != nil {
		cmp, err := compareVals(a.High, b.High)
		if err != nil {
			if isAmbiguousComparisonErr(err) {
				return nil, nil
			}
			return nil, err
		}
		if cmp > 0 {
			high = b.High
			highClosed = b.HighClosed
		}
	} else if a.High == nil {
		high = nil
		highClosed = a.HighClosed
	} else {
		// b.High is nil
		high = nil
		highClosed = b.HighClosed
	}

	// Check if result is valid (low <= high)
	if low != nil && high != nil {
		cmp, err := compareVals(low, high)
		if err != nil {
			if isAmbiguousComparisonErr(err) {
				return nil, nil
			}
			return nil, err
		}
		if cmp > 0 {
			return nil, nil // empty intersection
		}
	}
	// If both bounds are null, return null
	if low == nil && high == nil {
		return nil, nil
	}
	return cqltypes.NewInterval(low, high, lowClosed, highClosed), nil
}

// intervalPredecessor returns the predecessor for discrete types, adjusting the boundary
// to be closed. For types without a known step, returns (val, false) with open boundary.
func intervalPredecessor(v fptypes.Value) (fptypes.Value, bool) {
	if iv, ok := v.(fptypes.Integer); ok {
		return fptypes.NewInteger(iv.Value() - 1), true
	}
	if dv, ok := v.(fptypes.Decimal); ok {
		pred := dv.Value().Sub(smallDecimalStep)
		result := decimalToValue(pred)
		if result != nil {
			return result, true
		}
	}
	if dt, ok := v.(fptypes.DateTime); ok {
		unit := TemporalUnit(dt.Precision())
		return dt.SubtractDuration(1, unit), true
	}
	if t, ok := v.(fptypes.Time); ok {
		return AdjustTime(t, -1), true
	}
	if q, ok := v.(fptypes.Quantity); ok {
		pred := q.Value().Sub(smallDecimalStep)
		newQ := fptypes.NewQuantityFromDecimal(pred, q.Unit())
		return newQ, true
	}
	return v, false
}

// intervalSuccessor returns the successor for discrete types, adjusting the boundary
// to be closed. For types without a known step, returns (val, false) with open boundary.
func intervalSuccessor(v fptypes.Value) (fptypes.Value, bool) {
	if iv, ok := v.(fptypes.Integer); ok {
		return fptypes.NewInteger(iv.Value() + 1), true
	}
	if dv, ok := v.(fptypes.Decimal); ok {
		succ := dv.Value().Add(smallDecimalStep)
		result := decimalToValue(succ)
		if result != nil {
			return result, true
		}
	}
	if dt, ok := v.(fptypes.DateTime); ok {
		unit := TemporalUnit(dt.Precision())
		return dt.AddDuration(1, unit), true
	}
	if t, ok := v.(fptypes.Time); ok {
		return AdjustTime(t, 1), true
	}
	if q, ok := v.(fptypes.Quantity); ok {
		succ := q.Value().Add(smallDecimalStep)
		newQ := fptypes.NewQuantityFromDecimal(succ, q.Unit())
		return newQ, true
	}
	return v, false
}

// TemporalUnit maps DateTime precision to a duration unit string.
func TemporalUnit(prec fptypes.DateTimePrecision) string {
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

// AdjustTime adds delta units at the Time's precision (e.g., +1 ms, -1 second).
func AdjustTime(t fptypes.Time, delta int) fptypes.Value {
	h, m, s, ms := t.Hour(), t.Minute(), t.Second(), t.Millisecond()
	prec := t.Precision()
	switch prec {
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
	// Carry/borrow
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
	// Detect overflow/underflow
	if h >= 24 || h < 0 {
		return nil // signal overflow/underflow
	}
	// Construct time string based on precision
	var str string
	switch prec {
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
		return t // fallback to original
	}
	return result
}

// IntervalExcept returns a minus b for intervals.
func IntervalExcept(a, b cqltypes.Interval) (fptypes.Value, error) {
	overlap, err := a.Overlaps(b)
	if err != nil {
		if isAmbiguousComparisonErr(err) {
			return nil, nil
		}
		return nil, err
	}
	if !overlap {
		return a, nil
	}
	// If b fully includes a, return null
	included, err := b.Includes(a)
	if err != nil {
		return nil, err
	}
	if included {
		return nil, nil
	}
	// If b overlaps the low end of a, return the upper portion
	if a.Low != nil && b.Low != nil {
		cmpLow, err := compareVals(b.Low, a.Low)
		if err != nil {
			return nil, err
		}
		if cmpLow <= 0 && b.High != nil {
			// b covers the low end → result starts after b.High
			newLow, isClosed := intervalSuccessor(b.High)
			if !isClosed {
				return cqltypes.NewInterval(newLow, a.High, !b.HighClosed, a.HighClosed), nil
			}
			return cqltypes.NewInterval(newLow, a.High, true, a.HighClosed), nil
		}
	}
	// If b overlaps the high end of a, return the lower portion
	if a.High != nil && b.High != nil {
		cmpHigh, err := compareVals(b.High, a.High)
		if err != nil {
			return nil, err
		}
		if cmpHigh >= 0 && b.Low != nil {
			// b covers the high end → result ends before b.Low
			newHigh, isClosed := intervalPredecessor(b.Low)
			if !isClosed {
				return cqltypes.NewInterval(a.Low, newHigh, a.LowClosed, !b.LowClosed), nil
			}
			return cqltypes.NewInterval(a.Low, newHigh, a.LowClosed, true), nil
		}
	}
	// b is entirely inside a — would split a into two disjoint intervals, which is null in CQL
	return nil, nil
}

// IntervalBefore checks if interval a ends before interval b starts.
func IntervalBefore(a, b cqltypes.Interval) (fptypes.Value, error) {
	if a.High == nil || b.Low == nil {
		return nil, nil
	}
	cmp, err := compareVals(a.High, b.Low)
	if err != nil {
		return nil, err
	}
	return fptypes.NewBoolean(cmp < 0), nil
}

// IntervalAfter checks if interval a starts after interval b ends.
func IntervalAfter(a, b cqltypes.Interval) (fptypes.Value, error) {
	if a.Low == nil || b.High == nil {
		return nil, nil
	}
	cmp, err := compareVals(a.Low, b.High)
	if err != nil {
		return nil, err
	}
	return fptypes.NewBoolean(cmp > 0), nil
}

// intervalEndMeetsStart checks if the end of interval a meets the start of interval b.
// For closed boundaries, the successor of a.High must equal b.Low (e.g., [1,10] meets [11,20] for integers).
// For open/closed boundaries, equality is checked directly.
func intervalEndMeetsStart(aHigh fptypes.Value, aHighClosed bool, bLow fptypes.Value, bLowClosed bool) bool {
	if aHigh == nil || bLow == nil {
		return false
	}
	if aHighClosed && bLowClosed {
		// Both closed: successor of a.High should equal b.Low
		succ, ok := intervalSuccessor(aHigh)
		if ok {
			return succ.Equal(bLow)
		}
		return aHigh.Equal(bLow)
	}
	if aHighClosed && !bLowClosed {
		// a.High closed, b.Low open: they meet if a.High == b.Low
		return aHigh.Equal(bLow)
	}
	if !aHighClosed && bLowClosed {
		// a.High open, b.Low closed: they meet if a.High == b.Low
		return aHigh.Equal(bLow)
	}
	return false
}

var smallDecimalStep, _ = decimal.NewFromString("0.00000001")

// IntervalMeets checks if interval a meets interval b (a.high = b.low or a.low = b.high).
func IntervalMeets(a, b cqltypes.Interval) (fptypes.Value, error) {
	// If any bound involved in the meets check is null, result is null
	// (we need both endpoints that could touch to be known)
	if a.High == nil && a.Low == nil {
		return nil, nil
	}
	if b.High == nil && b.Low == nil {
		return nil, nil
	}
	// For a meets b: a.High meets b.Low (need both non-null)
	// For b meets a: b.High meets a.Low (need both non-null)
	aCanMeetB := a.High != nil && b.Low != nil
	bCanMeetA := b.High != nil && a.Low != nil
	if !aCanMeetB && !bCanMeetA {
		// Cannot determine meets relationship with null bounds
		return nil, nil
	}

	// Check if they overlap first - if they overlap, they don't meet
	overlaps, err := a.Overlaps(b)
	if err != nil {
		if isAmbiguousComparisonErr(err) {
			return nil, nil
		}
		return nil, err
	}
	if overlaps {
		return fptypes.NewBoolean(false), nil
	}
	if aCanMeetB && intervalEndMeetsStart(a.High, a.HighClosed, b.Low, b.LowClosed) {
		return fptypes.NewBoolean(true), nil
	}
	if bCanMeetA && intervalEndMeetsStart(b.High, b.HighClosed, a.Low, a.LowClosed) {
		return fptypes.NewBoolean(true), nil
	}
	return fptypes.NewBoolean(false), nil
}

func compareVals(a, b fptypes.Value) (int, error) {
	if a == nil || b == nil {
		return 0, nil // treat nil comparisons as equal (callers handle nil separately)
	}
	if ac, ok := a.(fptypes.Comparable); ok {
		return ac.Compare(b)
	}
	// Fall back to string comparison
	if a.String() < b.String() {
		return -1, nil
	}
	if a.String() > b.String() {
		return 1, nil
	}
	return 0, nil
}
