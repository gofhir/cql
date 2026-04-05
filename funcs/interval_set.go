package funcs

import (
	"fmt"
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

	// Quantity intervals
	if lq, ok := low.(fptypes.Quantity); ok {
		if hq, ok := high.(fptypes.Quantity); ok {
			diff := hq.Value().Sub(lq.Value())
			return fptypes.NewQuantityFromDecimal(diff, lq.Unit()), nil
		}
	}

	// DateTime/Date/Time intervals: width is not defined
	if _, ok := low.(fptypes.DateTime); ok {
		return nil, fmt.Errorf("width is not defined for DateTime intervals")
	}
	if _, ok := low.(fptypes.Date); ok {
		return nil, fmt.Errorf("width is not defined for Date intervals")
	}
	if _, ok := low.(fptypes.Time); ok {
		return nil, fmt.Errorf("width is not defined for Time intervals")
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
// "a meets after b" means a starts immediately after b ends.
func IntervalMeetsAfter(a, b cqltypes.Interval) (fptypes.Value, error) {
	// If a.High and b.Low are both known and a.High < b.Low, intervals are clearly separated
	// in the wrong direction for "a meets after b" → false
	if a.High != nil && b.Low != nil {
		cmp, err := compareVals(a.High, b.Low)
		if err == nil && cmp < 0 {
			return fptypes.NewBoolean(false), nil
		}
	}
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

// expandGetStep extracts the step amount and unit from a per value (Quantity or Integer/Decimal).
func expandGetStep(perVal fptypes.Value) (decimal.Decimal, string) {
	if perVal == nil {
		return decimal.Zero, ""
	}
	if q, ok := perVal.(fptypes.Quantity); ok {
		return q.Value(), q.Unit()
	}
	if i, ok := perVal.(fptypes.Integer); ok {
		return decimal.NewFromInt(i.Value()), ""
	}
	if d, ok := perVal.(fptypes.Decimal); ok {
		return d.Value(), ""
	}
	return decimal.Zero, ""
}

// IntervalExpandPoints expands an interval into a list of point values (single-interval overload).
// expand Interval[1, 10] → {1, 2, 3, ..., 10}
// expand Interval[1, 10] per 2 → {1, 3, 5, 7, 9}
func IntervalExpandPoints(interval cqltypes.Interval, perVal fptypes.Value) (fptypes.Collection, error) {
	if interval.Low == nil || interval.High == nil {
		return nil, nil
	}
	perAmount, perUnit := expandGetStep(perVal)

	// Integer intervals
	if li, ok := interval.Low.(fptypes.Integer); ok {
		if hi, ok := interval.High.(fptypes.Integer); ok {
			// If per is a decimal fraction (e.g., per 0.1), promote to decimal expand
			if !perAmount.IsZero() && !perAmount.Equal(perAmount.Truncate(0)) {
				lo := decimal.NewFromInt(li.Value())
				// For integer bounds with decimal per, the effective high is hi + 1 - per
				// because integer 10 occupies [10.0, 10.9] at 0.1 precision
				effHi := decimal.NewFromInt(hi.Value()).Add(decimal.NewFromInt(1)).Sub(perAmount)
				return expandDecimalPoints(lo, effHi, interval.LowClosed, interval.HighClosed, perAmount)
			}
			step := int64(1)
			if !perAmount.IsZero() {
				step = perAmount.IntPart()
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
			var result fptypes.Collection
			for v := lo; v <= hi; v += step {
				// For step > 1, the point represents a range [v, v+step-1]
				// Only include if the end of that range is within bounds
				if step > 1 {
					end := v + step - 1
					if end > hi {
						break
					}
				}
				result = append(result, fptypes.NewInteger(v))
				if len(result) > 10000 {
					break
				}
			}
			return result, nil
		}
	}

	// Decimal intervals
	if ld, ok := interval.Low.(fptypes.Decimal); ok {
		if hd, ok := interval.High.(fptypes.Decimal); ok {
			step := decimal.NewFromInt(1)
			if !perAmount.IsZero() {
				step = perAmount
			}
			return expandDecimalPoints(ld.Value(), hd.Value(), interval.LowClosed, interval.HighClosed, step)
		}
	}

	// DateTime/Date intervals
	if _, ok := interval.Low.(fptypes.DateTime); ok {
		return expandTemporalPoints(interval, perAmount, perUnit)
	}
	if _, ok := interval.Low.(fptypes.Date); ok {
		return expandTemporalPoints(interval, perAmount, perUnit)
	}
	// Time intervals
	if _, ok := interval.Low.(fptypes.Time); ok {
		return expandTemporalPoints(interval, perAmount, perUnit)
	}

	return nil, nil
}

func expandDecimalPoints(lo, hi decimal.Decimal, lowClosed, highClosed bool, step decimal.Decimal) (fptypes.Collection, error) {
	// For decimals, the effective bounds are based on the original interval bounds
	// Open bounds exclude the boundary value, but we step from the low
	effLo := lo
	if !lowClosed {
		effLo = lo.Add(step)
	}
	// Effective high: for step=1 integer-like, each point represents itself
	// We iterate while v + step - 1 <= hi (closed) or v + step - 1 < hi (open)
	var result fptypes.Collection
	for v := effLo; ; v = v.Add(step) {
		// Check if v itself exceeds the boundary
		if highClosed {
			if v.GreaterThan(hi) {
				break
			}
		} else {
			if v.GreaterThanOrEqual(hi) {
				break
			}
		}
		dv := decimalToValue(v)
		if dv == nil {
			break
		}
		result = append(result, dv)
		if len(result) > 10000 {
			break
		}
	}
	return result, nil
}

// IntervalExpandIntervals expands an interval into a list of unit intervals (list-of-intervals overload).
// expand { Interval[1, 10] } → {Interval[1,1], Interval[2,2], ..., Interval[10,10]}
// expand { Interval[1, 10] } per 2 → {Interval[1,2], Interval[3,4], ..., Interval[9,10]}
func IntervalExpandIntervals(interval cqltypes.Interval, perVal fptypes.Value) (fptypes.Collection, error) {
	if interval.Low == nil || interval.High == nil {
		return nil, nil
	}
	perAmount, perUnit := expandGetStep(perVal)

	// Integer intervals
	if li, ok := interval.Low.(fptypes.Integer); ok {
		if hi, ok := interval.High.(fptypes.Integer); ok {
			// If per is a decimal fraction, promote to decimal expand
			if !perAmount.IsZero() && !perAmount.Equal(perAmount.Truncate(0)) {
				lo := decimal.NewFromInt(li.Value())
				effHi := decimal.NewFromInt(hi.Value()).Add(decimal.NewFromInt(1)).Sub(perAmount)
				return expandDecimalIntervals(lo, effHi, interval.LowClosed, interval.HighClosed, perAmount)
			}
			step := int64(1)
			if !perAmount.IsZero() {
				step = perAmount.IntPart()
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
			var result fptypes.Collection
			for v := lo; v <= hi; v += step {
				end := v + step - 1
				if end > hi {
					// For step > 1, exclude the last incomplete interval
					if step > 1 {
						break
					}
					end = hi
				}
				result = append(result, cqltypes.NewInterval(fptypes.NewInteger(v), fptypes.NewInteger(end), true, true))
				if len(result) > 10000 {
					break
				}
			}
			return result, nil
		}
	}

	// Decimal intervals
	if ld, ok := interval.Low.(fptypes.Decimal); ok {
		if hd, ok := interval.High.(fptypes.Decimal); ok {
			step := decimal.NewFromInt(1)
			if !perAmount.IsZero() {
				step = perAmount
			}
			return expandDecimalIntervals(ld.Value(), hd.Value(), interval.LowClosed, interval.HighClosed, step)
		}
	}

	// DateTime/Date intervals
	if _, ok := interval.Low.(fptypes.DateTime); ok {
		return expandTemporalIntervals(interval, perAmount, perUnit)
	}
	if _, ok := interval.Low.(fptypes.Date); ok {
		return expandTemporalIntervals(interval, perAmount, perUnit)
	}
	// Time intervals
	if _, ok := interval.Low.(fptypes.Time); ok {
		return expandTemporalIntervals(interval, perAmount, perUnit)
	}

	return nil, nil
}

func expandDecimalIntervals(lo, hi decimal.Decimal, lowClosed, highClosed bool, step decimal.Decimal) (fptypes.Collection, error) {
	effLo := lo
	if !lowClosed {
		effLo = lo.Add(step)
	}
	var result fptypes.Collection
	for v := effLo; ; v = v.Add(step) {
		// Check if v is beyond the high bound
		if highClosed {
			if v.GreaterThan(hi) {
				break
			}
		} else {
			if v.GreaterThanOrEqual(hi) {
				break
			}
		}
		end := v.Add(step).Sub(decimal.NewFromFloat(1).Div(decimal.NewFromInt(10).Pow(decimal.NewFromInt(-int64(step.Exponent())))))
		// For integer step, unit intervals: low == high
		if step.Equal(step.Truncate(0)) {
			end = v
		}
		if end.GreaterThan(hi) {
			if highClosed {
				end = hi
			} else {
				// For open high bound with step > smallest unit, skip incomplete last interval
				if !step.Equal(step.Truncate(0)) {
					break
				}
				end = hi
			}
		}
		lowVal := decimalToValue(v)
		highVal := decimalToValue(end)
		if lowVal == nil || highVal == nil {
			break
		}
		result = append(result, cqltypes.NewInterval(lowVal, highVal, true, true))
		if len(result) > 10000 {
			break
		}
	}
	return result, nil
}

// temporalUnitPrecision returns the precision level for a temporal unit (lower = coarser).
func temporalUnitPrecision(unit string) int {
	switch unit {
	case "year":
		return 0
	case "month":
		return 1
	case "week":
		return 2
	case "day":
		return 3
	case "hour":
		return 4
	case "minute":
		return 5
	case "second":
		return 6
	case "millisecond":
		return 7
	}
	return 3
}

// expandTemporalPoints expands a DateTime/Date/Time interval into point values.
func expandTemporalPoints(interval cqltypes.Interval, perAmount decimal.Decimal, perUnit string) (fptypes.Collection, error) {
	amount := int(1)
	if !perAmount.IsZero() {
		amount = int(perAmount.IntPart())
	}
	unit := perUnit
	if unit == "" || unit == "1" {
		unit = defaultTemporalUnit(interval.Low)
	}
	unit = normalizeTemporalUnit(unit)

	// If per unit is finer than the interval's precision, return empty
	intervalUnit := defaultTemporalUnit(interval.Low)
	if temporalUnitPrecision(unit) > temporalUnitPrecision(intervalUnit) {
		return nil, nil
	}

	// Truncate interval bounds to per-unit precision for Time values
	low := truncateTemporalToPrecision(interval.Low, unit)
	high := truncateTemporalToPrecision(interval.High, unit)
	// Adjust high bound closure: if truncation changed the value, open becomes closed
	highClosed := interval.HighClosed
	if !highClosed && high != nil && interval.High != nil && !high.Equal(interval.High) {
		highClosed = true
	}

	var result fptypes.Collection
	cur := low
	for i := 0; i < 10001; i++ {
		if cur == nil {
			break
		}
		inBounds, err := isInBounds(cur, high, interval.LowClosed, highClosed, i == 0)
		if err != nil || !inBounds {
			break
		}
		result = append(result, cur)
		next, err := DateAdd(cur, amount, unit)
		if err != nil {
			break
		}
		cur = next
	}
	return result, nil
}

// expandTemporalIntervals expands a DateTime/Date/Time interval into unit intervals.
func expandTemporalIntervals(interval cqltypes.Interval, perAmount decimal.Decimal, perUnit string) (fptypes.Collection, error) {
	amount := int(1)
	if !perAmount.IsZero() {
		amount = int(perAmount.IntPart())
	}
	unit := perUnit
	if unit == "" || unit == "1" {
		unit = defaultTemporalUnit(interval.Low)
	}
	unit = normalizeTemporalUnit(unit)

	// If per unit is finer than the interval's precision, return empty
	intervalUnit := defaultTemporalUnit(interval.Low)
	if temporalUnitPrecision(unit) > temporalUnitPrecision(intervalUnit) {
		return nil, nil
	}

	// Truncate interval bounds to per-unit precision for Time values
	low := truncateTemporalToPrecision(interval.Low, unit)
	high := truncateTemporalToPrecision(interval.High, unit)
	// Adjust high bound closure: if truncation changed the value, open becomes closed
	highClosed := interval.HighClosed
	if !highClosed && high != nil && interval.High != nil && !high.Equal(interval.High) {
		highClosed = true
	}

	var result fptypes.Collection
	cur := low
	for i := 0; i < 10001; i++ {
		if cur == nil {
			break
		}
		inBounds, err := isInBounds(cur, high, interval.LowClosed, highClosed, i == 0)
		if err != nil || !inBounds {
			break
		}
		next, err := DateAdd(cur, amount, unit)
		if err != nil {
			break
		}
		// End of unit interval is next - 1 unit at the per unit precision
		end, err := DateAdd(next, -1, unit)
		if err != nil {
			break
		}
		// Clamp end to the interval high
		if end != nil {
			cmp, cmpErr := compareVals(end, high)
			if cmpErr == nil && cmp > 0 {
				end = high
			}
		}
		result = append(result, cqltypes.NewInterval(cur, end, true, true))
		cur = next
	}
	return result, nil
}

// isInBounds checks if a value is within the interval bounds.
func isInBounds(val, high fptypes.Value, lowClosed, highClosed, isFirst bool) (bool, error) {
	cmp, err := compareVals(val, high)
	if err != nil {
		return false, err
	}
	if highClosed {
		return cmp <= 0, nil
	}
	return cmp < 0, nil
}

// truncateTemporalToPrecision truncates a Time value to the given unit precision.
// For example, truncating @T10:30 to "hour" gives @T10.
// Non-Time values or cases where no truncation is needed are returned as-is.
func truncateTemporalToPrecision(v fptypes.Value, unit string) fptypes.Value {
	if v == nil {
		return nil
	}
	t, ok := v.(fptypes.Time)
	if !ok {
		return v // only truncate Time values for now
	}
	targetPrec := unitToTimePrecision(unit)
	if targetPrec < 0 || fptypes.TimePrecision(targetPrec) >= t.Precision() {
		return v // already at or below target precision
	}
	// Build truncated time string
	h := t.Hour()
	switch fptypes.TimePrecision(targetPrec) {
	case fptypes.HourPrecision:
		newT, err := fptypes.NewTime(fmt.Sprintf("%02d", h))
		if err != nil {
			return v
		}
		return newT
	case fptypes.MinutePrecision:
		newT, err := fptypes.NewTime(fmt.Sprintf("%02d:%02d", h, t.Minute()))
		if err != nil {
			return v
		}
		return newT
	case fptypes.SecondPrecision:
		newT, err := fptypes.NewTime(fmt.Sprintf("%02d:%02d:%02d", h, t.Minute(), t.Second()))
		if err != nil {
			return v
		}
		return newT
	}
	return v
}

// unitToTimePrecision maps a temporal unit name to a TimePrecision value.
// Returns -1 for units that don't map to Time precision (year, month, etc.).
func unitToTimePrecision(unit string) int {
	switch unit {
	case "hour":
		return int(fptypes.HourPrecision)
	case "minute":
		return int(fptypes.MinutePrecision)
	case "second":
		return int(fptypes.SecondPrecision)
	case "millisecond":
		return int(fptypes.MillisPrecision)
	}
	return -1
}

// defaultTemporalUnit returns the default unit for expand based on the value type and precision.
func defaultTemporalUnit(v fptypes.Value) string {
	switch t := v.(type) {
	case fptypes.DateTime:
		// DateTime precision: 0=year, 1=month, 2=day, 3=hour, 4=minute, 5=second, 6=millisecond
		switch t.Precision() {
		case 0, 1, 2:
			return "day"
		case 3:
			return "hour"
		case 4:
			return "minute"
		case 5:
			return "second"
		default:
			return "millisecond"
		}
	case fptypes.Date:
		return "day"
	case fptypes.Time:
		// Time precision: 0=hour (HourPrecision), 1=minute, 2=second, 3=millisecond
		switch t.Precision() {
		case 0:
			return "hour"
		case 1:
			return "minute"
		case 2:
			return "second"
		default:
			return "millisecond"
		}
	}
	return "day"
}

// normalizeTemporalUnit normalizes unit strings to singular form.
func normalizeTemporalUnit(unit string) string {
	switch unit {
	case "years", "year":
		return "year"
	case "months", "month":
		return "month"
	case "weeks", "week":
		return "week"
	case "days", "day":
		return "day"
	case "hours", "hour":
		return "hour"
	case "minutes", "minute":
		return "minute"
	case "seconds", "second":
		return "second"
	case "milliseconds", "millisecond":
		return "millisecond"
	}
	return unit
}

// IntervalExpand expands an interval into a list of unit intervals per the given per quantity.
// Kept for backward compatibility.
func IntervalExpand(interval cqltypes.Interval, per decimal.Decimal) ([]fptypes.Value, error) {
	var perVal fptypes.Value
	if !per.IsZero() {
		perVal = fptypes.NewQuantityFromDecimal(per, "1")
	}
	result, err := IntervalExpandIntervals(interval, perVal)
	if err != nil {
		return nil, err
	}
	vals := make([]fptypes.Value, len(result))
	for i, v := range result {
		vals[i] = v
	}
	return vals, nil
}
