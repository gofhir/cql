package funcs

import (
	"github.com/shopspring/decimal"

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
// Returns null if the intervals do not overlap or meet.
func IntervalUnion(a, b cqltypes.Interval) (fptypes.Value, error) {
	// Check if intervals overlap or are adjacent (meet)
	overlaps, err := a.Overlaps(b)
	if err != nil {
		return nil, err
	}
	meets := false
	if !overlaps {
		// Check meets: a.High == b.Low or b.High == a.Low (considering successors for closed boundaries)
		if a.High != nil && b.Low != nil {
			if a.High.Equal(b.Low) {
				meets = true
			}
			// Check adjacency for integers: e.g., Interval[1,10] meets Interval[11,20]
			if !meets {
				if ai, ok := a.High.(fptypes.Integer); ok {
					if bi, ok := b.Low.(fptypes.Integer); ok {
						if a.HighClosed && b.LowClosed && ai.Value()+1 == bi.Value() {
							meets = true
						}
					}
				}
			}
		}
		if !meets && b.High != nil && a.Low != nil {
			if b.High.Equal(a.Low) {
				meets = true
			}
			if !meets {
				if bi, ok := b.High.(fptypes.Integer); ok {
					if ai, ok := a.Low.(fptypes.Integer); ok {
						if b.HighClosed && a.LowClosed && bi.Value()+1 == ai.Value() {
							meets = true
						}
					}
				}
			}
		}
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
	return v, false
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
		if ai, ok := aHigh.(fptypes.Integer); ok {
			if bi, ok := bLow.(fptypes.Integer); ok {
				return ai.Value()+1 == bi.Value()
			}
		}
		if ad, ok := aHigh.(fptypes.Decimal); ok {
			if bd, ok := bLow.(fptypes.Decimal); ok {
				// For decimals, check if they are adjacent within precision
				diff := bd.Value().Sub(ad.Value())
				return diff.IsPositive() && diff.LessThanOrEqual(smallDecimalStep)
			}
		}
		// For DateTime/Date/Time, check if b.Low is successor of a.High
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
	// Check if they overlap first - if they overlap, they don't meet
	overlaps, err := a.Overlaps(b)
	if err != nil {
		return nil, err
	}
	if overlaps {
		return fptypes.NewBoolean(false), nil
	}
	if intervalEndMeetsStart(a.High, a.HighClosed, b.Low, b.LowClosed) {
		return fptypes.NewBoolean(true), nil
	}
	if intervalEndMeetsStart(b.High, b.HighClosed, a.Low, a.LowClosed) {
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
