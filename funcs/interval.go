package funcs

import (
	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

// IntervalContains checks if an interval contains a point.
func IntervalContains(interval cqltypes.Interval, point fptypes.Value) (fptypes.Value, error) {
	result, err := interval.Contains(point)
	if err != nil {
		if cqltypes.UndecidableComparison(err) {
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
		if cqltypes.UndecidableComparison(err) {
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
		if cqltypes.UndecidableComparison(err) {
			return nil, nil
		}
		return nil, err
	}
	return fptypes.NewBoolean(result), nil
}

// IntervalStartOf returns the first point an interval includes, which for an
// open boundary is the successor of it rather than the boundary itself.
//
// The step fails at the limit of the point type, where there is no successor to
// name, and that error travels rather than answering with a boundary the
// interval excludes.
func IntervalStartOf(interval cqltypes.Interval) (fptypes.Value, error) {
	return interval.Start()
}

// IntervalEndOf returns the last point an interval includes.
func IntervalEndOf(interval cqltypes.Interval) (fptypes.Value, error) {
	return interval.End()
}

// IntervalUnion returns the union of two intervals.
// Returns null if the intervals do not overlap or meet.
func IntervalUnion(a, b cqltypes.Interval) (fptypes.Value, error) {
	// Check if intervals overlap or are adjacent (meet)
	overlaps, err := a.Overlaps(b)
	if err != nil {
		if cqltypes.UndecidableComparison(err) {
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
	switch {
	case a.Low != nil && b.Low != nil:
		cmp, err := compareVals(a.Low, b.Low)
		if err != nil {
			if cqltypes.UndecidableComparison(err) {
				return nil, nil
			}
			return nil, err
		}
		if cmp < 0 {
			low = b.Low
			lowClosed = b.LowClosed
		}
	case a.Low == nil:
		low = nil
		lowClosed = a.LowClosed
	default:
		// b.Low is nil
		low = nil
		lowClosed = b.LowClosed
	}

	// Take min high — if either is null (unknown), result high is null (unknown)
	high := a.High
	highClosed := a.HighClosed
	switch {
	case a.High != nil && b.High != nil:
		cmp, err := compareVals(a.High, b.High)
		if err != nil {
			if cqltypes.UndecidableComparison(err) {
				return nil, nil
			}
			return nil, err
		}
		if cmp > 0 {
			high = b.High
			highClosed = b.HighClosed
		}
	case a.High == nil:
		high = nil
		highClosed = a.HighClosed
	default:
		// b.High is nil
		high = nil
		highClosed = b.HighClosed
	}

	// Check if result is valid (low <= high)
	if low != nil && high != nil {
		cmp, err := compareVals(low, high)
		if err != nil {
			if cqltypes.UndecidableComparison(err) {
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

// intervalPredecessor and intervalSuccessor are the value one step before and
// after a boundary, and whether there was a step to take.
//
// They were a second implementation of cqltypes.Predecessor and Successor, and a
// poorer one: those know Integer, Decimal, DateTime, Date, Time and Quantity and
// guard the representable range, while the copies here knew everything except
// Date. Through them, two consecutive days did not touch while two consecutive
// integers did:
//
//	Interval[1, 3] meets Interval[4, 7]                                true
//	Interval[@2020-01-01, @2020-01-03] meets Interval[@2020-01-04, …]   false
//	successor of @2020-01-03                                           2020-01-04
//
// The engine knew the successor of that date perfectly well through the other
// implementation, which is what made it a contradiction rather than a limit. The
// three callers here — `meets`, `except`, and the collapse that rests on `meets` —
// now get the answer the operator gives.
//
// The false return means "no step to take", which is what the callers read to
// decide whether a new boundary is closed or open. An error from the range guard
// reads the same way: at the end of the calendar there is no next day to move to.
func intervalPredecessor(v fptypes.Value) (fptypes.Value, bool) {
	stepped, err := cqltypes.Predecessor(v)
	return stepOrStay(stepped, err, v)
}

func intervalSuccessor(v fptypes.Value) (fptypes.Value, bool) {
	stepped, err := cqltypes.Successor(v)
	return stepOrStay(stepped, err, v)
}

func stepOrStay(stepped fptypes.Value, err error, original fptypes.Value) (fptypes.Value, bool) {
	if err != nil || stepped == nil {
		return original, false
	}
	return stepped, true
}

// TemporalUnit maps DateTime precision to a duration unit string.
func TemporalUnit(prec fptypes.DateTimePrecision) string {
	return cqltypes.DateTimeUnit(prec)
}

// AdjustTime adds delta units at the Time's precision (e.g., +1 ms, -1 second).
func AdjustTime(t fptypes.Time, delta int) fptypes.Value {
	return cqltypes.AdjustTime(t, delta)
}

// IntervalExcept returns a minus b for intervals.
func IntervalExcept(a, b cqltypes.Interval) (fptypes.Value, error) {
	overlap, err := a.Overlaps(b)
	if err != nil {
		if cqltypes.UndecidableComparison(err) {
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
		// These two had no guard at all, where every operator around them has
		// one: `Interval[1 'cm', 2 'cm'] before Interval[1 's', 2 's']` raised an
		// error while `starts before` and `overlaps before` on the same pair
		// answered null, and so did the scalar `1 'cm' before 1 's'`.
		if cqltypes.UndecidableComparison(err) {
			return nil, nil
		}
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
		if cqltypes.UndecidableComparison(err) {
			return nil, nil
		}
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
		if cqltypes.UndecidableComparison(err) {
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

// compareVals is the one place this package orders two values, so routing it
// through CompareTemporal is what gives the interval and timing operators CQL's
// precision rule rather than the one fptypes applies.
func compareVals(a, b fptypes.Value) (int, error) {
	if a == nil || b == nil {
		return 0, nil // treat nil comparisons as equal (callers handle nil separately)
	}
	if _, ok := a.(fptypes.Comparable); ok {
		return cqltypes.CompareTemporal(a, b)
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
