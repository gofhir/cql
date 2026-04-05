package funcs

import (
	"sort"

	"github.com/shopspring/decimal"

	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
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
		if isAmbiguousComparisonErr(err) {
			return nil, nil
		}
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
	return fptypes.NewBoolean(intervalEndMeetsStart(a.High, a.HighClosed, b.Low, b.LowClosed)), nil
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
		cmp, cmpErr := compareVals(sorted[i].Low, sorted[j].Low)
		_ = cmpErr
		return cmp < 0
	})

	result := []cqltypes.Interval{sorted[0]}
	for _, iv := range sorted[1:] {
		last := &result[len(result)-1]
		overlaps, _ := last.Overlaps(iv) //nolint:errcheck // best-effort merge
		meets := intervalEndMeetsStart(last.High, last.HighClosed, iv.Low, iv.LowClosed)
		if overlaps || meets {
			// Merge
			if iv.High != nil && last.High != nil {
				cmp, _ := compareVals(iv.High, last.High) //nolint:errcheck // best-effort merge
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
				end := v + step - 1
				if end > hi {
					end = hi
				}
				result = append(result, cqltypes.NewInterval(fptypes.NewInteger(v), fptypes.NewInteger(end), true, true))
			}
			return result, nil
		}
	}

	// Decimal intervals
	if ld, ok := interval.Low.(fptypes.Decimal); ok {
		if hd, ok := interval.High.(fptypes.Decimal); ok {
			step := decimal.NewFromFloat(1.0)
			if !per.IsZero() {
				step = per
			}
			lo := ld.Value()
			hi := hd.Value()
			if !interval.LowClosed {
				lo = lo.Add(step)
			}
			if !interval.HighClosed {
				hi = hi.Sub(step)
			}
			var result []fptypes.Value
			for v := lo; v.LessThanOrEqual(hi); v = v.Add(step) {
				end := v.Add(step).Sub(decimal.NewFromFloat(0.00000001))
				if end.GreaterThan(hi) {
					end = hi
				}
				lowVal := decimalToValue(v)
				highVal := decimalToValue(end)
				if lowVal == nil || highVal == nil {
					break
				}
				result = append(result, cqltypes.NewInterval(lowVal, highVal, true, true))
				// Safety: limit to reasonable count
				if len(result) > 10000 {
					break
				}
			}
			return result, nil
		}
	}

	return nil, nil
}
