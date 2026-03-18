package funcs

import (
	"sort"

	fptypes "github.com/gofhir/fhirpath/types"
	"github.com/shopspring/decimal"

	cqltypes "github.com/gofhir/cql/types"
)

// IntervalSize returns the number of points in the interval (for integer intervals).
func IntervalSize(interval cqltypes.Interval) (fptypes.Value, error) {
	if interval.Low == nil || interval.High == nil {
		return nil, nil
	}
	w, err := IntervalWidth(interval)
	if err != nil || w == nil {
		return nil, err
	}
	// Size = width + 1 for closed integer intervals
	if iv, ok := w.(fptypes.Integer); ok {
		return fptypes.NewInteger(iv.Value() + 1), nil
	}
	return w, nil
}

// IntervalWidth returns the width (distance) of the interval.
func IntervalWidth(interval cqltypes.Interval) (fptypes.Value, error) {
	if interval.Low == nil || interval.High == nil {
		return nil, nil
	}

	low := interval.Low
	high := interval.High

	// Integer intervals
	if li, ok := low.(fptypes.Integer); ok {
		if hi, ok := high.(fptypes.Integer); ok {
			lo := li.Value()
			hi := hi.Value()
			if !interval.LowClosed {
				lo++
			}
			if !interval.HighClosed {
				hi--
			}
			return fptypes.NewInteger(hi - lo), nil
		}
	}

	// Decimal intervals
	if ld, ok := low.(fptypes.Decimal); ok {
		if hd, ok := high.(fptypes.Decimal); ok {
			diff := hd.Value().Sub(ld.Value())
			return decimalToValue(diff), nil
		}
	}

	return nil, nil
}

// IntervalPointFrom extracts a point from a unit interval (where low = high and both closed).
func IntervalPointFrom(interval cqltypes.Interval) (fptypes.Value, error) {
	if interval.Low == nil || interval.High == nil {
		return nil, nil
	}
	if !interval.LowClosed || !interval.HighClosed {
		return nil, nil
	}
	if interval.Low.Equal(interval.High) {
		return interval.Low, nil
	}
	return nil, nil
}

// IntervalProperlyIncludes checks if a properly includes b (includes but not equal).
func IntervalProperlyIncludes(a, b cqltypes.Interval) (fptypes.Value, error) {
	includes, err := a.Includes(b)
	if err != nil {
		return nil, err
	}
	if !includes {
		return fptypes.NewBoolean(false), nil
	}
	return fptypes.NewBoolean(!a.Equal(b)), nil
}

// IntervalProperlyIncludedIn checks if a is properly included in b.
func IntervalProperlyIncludedIn(a, b cqltypes.Interval) (fptypes.Value, error) {
	return IntervalProperlyIncludes(b, a)
}

// IntervalMeetsBefore checks if interval a ends exactly at the start of b.
func IntervalMeetsBefore(a, b cqltypes.Interval) (fptypes.Value, error) {
	if a.High == nil || b.Low == nil {
		return nil, nil
	}
	if a.High.Equal(b.Low) && a.HighClosed != b.LowClosed {
		return fptypes.NewBoolean(true), nil
	}
	// Also check successor relationship for integer intervals
	if ai, ok := a.High.(fptypes.Integer); ok {
		if bi, ok := b.Low.(fptypes.Integer); ok {
			if ai.Value()+1 == bi.Value() {
				return fptypes.NewBoolean(true), nil
			}
		}
	}
	return fptypes.NewBoolean(false), nil
}

// IntervalMeetsAfter checks if interval a starts exactly at the end of b.
func IntervalMeetsAfter(a, b cqltypes.Interval) (fptypes.Value, error) {
	return IntervalMeetsBefore(b, a)
}

// IntervalCollapse collapses a list of intervals into non-overlapping intervals.
func IntervalCollapse(intervals []cqltypes.Interval) ([]cqltypes.Interval, error) {
	if len(intervals) == 0 {
		return nil, nil
	}
	// Sort by low boundary using O(n log n) sort
	sorted := make([]cqltypes.Interval, len(intervals))
	copy(sorted, intervals)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Low == nil {
			return true
		}
		if sorted[j].Low == nil {
			return false
		}
		cmp, _ := compareVals(sorted[i].Low, sorted[j].Low)
		return cmp < 0
	})

	result := []cqltypes.Interval{sorted[0]}
	for _, iv := range sorted[1:] {
		last := &result[len(result)-1]
		overlaps, _ := last.Overlaps(iv)
		meets := false
		if last.High != nil && iv.Low != nil {
			meets = last.High.Equal(iv.Low)
			// Also check integer successor
			if li, ok := last.High.(fptypes.Integer); ok {
				if ri, ok := iv.Low.(fptypes.Integer); ok {
					if li.Value()+1 == ri.Value() {
						meets = true
					}
				}
			}
		}
		if overlaps || meets {
			// Merge
			if iv.High != nil && last.High != nil {
				cmp, _ := compareVals(iv.High, last.High)
				if cmp > 0 {
					last.High = iv.High
					last.HighClosed = iv.HighClosed
				}
			}
		} else {
			result = append(result, iv)
		}
	}
	return result, nil
}

// IntervalExpand expands an interval into a list of unit intervals per the given per quantity.
func IntervalExpand(interval cqltypes.Interval, per decimal.Decimal) ([]fptypes.Value, error) {
	if interval.Low == nil || interval.High == nil {
		return nil, nil
	}

	// Integer intervals
	if li, ok := interval.Low.(fptypes.Integer); ok {
		if hi, ok := interval.High.(fptypes.Integer); ok {
			step := int64(1)
			if !per.IsZero() {
				step = per.IntPart()
				if step < 1 {
					step = 1
				}
			}
			lo := li.Value()
			hi := hi.Value()
			if !interval.LowClosed {
				lo++
			}
			if !interval.HighClosed {
				hi--
			}
			var result []fptypes.Value
			for v := lo; v <= hi; v += step {
				result = append(result, fptypes.NewInteger(v))
			}
			return result, nil
		}
	}

	return nil, nil
}
