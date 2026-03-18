package funcs

import (
	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

// IntervalContains checks if an interval contains a point.
func IntervalContains(interval cqltypes.Interval, point fptypes.Value) (fptypes.Value, error) {
	result, err := interval.Contains(point)
	if err != nil {
		return nil, err
	}
	return fptypes.NewBoolean(result), nil
}

// IntervalIncludes checks if interval a includes interval b.
func IntervalIncludes(a, b cqltypes.Interval) (fptypes.Value, error) {
	result, err := a.Includes(b)
	if err != nil {
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
func IntervalUnion(a, b cqltypes.Interval) (fptypes.Value, error) {
	if a.Low == nil && b.Low == nil {
		return cqltypes.NewInterval(nil, nil, true, true), nil
	}
	// Simple union: take min low, max high
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
	// Take max low, min high
	low := a.Low
	lowClosed := a.LowClosed
	if a.Low != nil && b.Low != nil {
		cmp, err := compareVals(a.Low, b.Low)
		if err != nil {
			return nil, err
		}
		if cmp < 0 {
			low = b.Low
			lowClosed = b.LowClosed
		}
	}
	high := a.High
	highClosed := a.HighClosed
	if a.High != nil && b.High != nil {
		cmp, err := compareVals(a.High, b.High)
		if err != nil {
			return nil, err
		}
		if cmp > 0 {
			high = b.High
			highClosed = b.HighClosed
		}
	}
	// Check if result is valid (low <= high)
	if low != nil && high != nil {
		cmp, err := compareVals(low, high)
		if err != nil {
			return nil, err
		}
		if cmp > 0 {
			return nil, nil // empty intersection
		}
	}
	return cqltypes.NewInterval(low, high, lowClosed, highClosed), nil
}

// IntervalExcept returns a minus b for intervals.
func IntervalExcept(a, b cqltypes.Interval) (fptypes.Value, error) {
	overlap, err := a.Overlaps(b)
	if err != nil {
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
			// b covers the low end → result is (b.High, a.High]
			return cqltypes.NewInterval(b.High, a.High, !b.HighClosed, a.HighClosed), nil
		}
	}
	// If b overlaps the high end of a, return the lower portion
	if a.High != nil && b.High != nil {
		cmpHigh, err := compareVals(b.High, a.High)
		if err != nil {
			return nil, err
		}
		if cmpHigh >= 0 && b.Low != nil {
			// b covers the high end → result is [a.Low, b.Low)
			return cqltypes.NewInterval(a.Low, b.Low, a.LowClosed, !b.LowClosed), nil
		}
	}
	return a, nil
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

// IntervalMeets checks if interval a meets interval b (a.high = b.low or a.low = b.high).
func IntervalMeets(a, b cqltypes.Interval) (fptypes.Value, error) {
	if a.High != nil && b.Low != nil && a.High.Equal(b.Low) {
		return fptypes.NewBoolean(true), nil
	}
	if a.Low != nil && b.High != nil && a.Low.Equal(b.High) {
		return fptypes.NewBoolean(true), nil
	}
	return fptypes.NewBoolean(false), nil
}

func compareVals(a, b fptypes.Value) (int, error) {
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
