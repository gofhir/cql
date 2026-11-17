package funcs

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
)

// Expand, by CQL 1.5.3 (https://cql.hl7.org/N1A/09-b-cqlreference.html#expand):
// the input is ordered and its overlapping or meeting intervals merged; each
// boundary is read with start of / end of at its own precision and then
// truncated to the precision of per; the 'per' partitions are counted from the
// starting boundary of the first interval, and an interval contributes the
// whole partitions that start on or after its lower boundary and end on or
// before its upper one.

// decimalEpsilon is the smallest difference CQL defines between two Decimals,
// the step an open Decimal or Quantity boundary takes to become a closed one.
var decimalEpsilon = decimal.RequireFromString("0.00000001")

// IntervalExpandPoints is IntervalExpandPointsUpTo at DefaultMaxExpandSize.
func IntervalExpandPoints(interval cqltypes.Interval, perVal fptypes.Value) (fptypes.Collection, error) {
	return IntervalExpandPointsUpTo(interval, perVal, DefaultMaxExpandSize)
}

// IntervalExpandIntervals is IntervalExpandIntervalsUpTo at DefaultMaxExpandSize.
func IntervalExpandIntervals(interval cqltypes.Interval, perVal fptypes.Value) (fptypes.Collection, error) {
	return IntervalExpandIntervalsUpTo(interval, perVal, DefaultMaxExpandSize)
}

// IntervalExpandIntervalsUpTo expands one interval into the intervals of size
// per it holds, failing with ErrExpandTooLarge when there are more than limit
// of them. A limit of zero or less is none.
func IntervalExpandIntervalsUpTo(interval cqltypes.Interval, perVal fptypes.Value, limit int) (fptypes.Collection, error) {
	return ExpandIntervalsUpTo([]cqltypes.Interval{interval}, perVal, limit)
}

// IntervalExpandPointsUpTo expands an interval into a list of point values
// (the single-interval overload): the starts of what the list overload gives
// for the interval. It fails with ErrExpandTooLarge when there are more than
// limit of them. A limit of zero or less is none.
//
//	expand Interval[1, 10]        → {1, 2, 3, ..., 10}
//	expand Interval[1, 10] per 2  → {1, 3, 5, 7, 9}
func IntervalExpandPointsUpTo(interval cqltypes.Interval, perVal fptypes.Value, limit int) (fptypes.Collection, error) {
	parts, err := ExpandIntervalsUpTo([]cqltypes.Interval{interval}, perVal, limit)
	if err != nil {
		return nil, err
	}
	var points fptypes.Collection
	for _, p := range parts {
		points = append(points, p.(cqltypes.Interval).Low)
	}
	return points, nil
}

// ExpandIntervalsUpTo expands a list of intervals into the list of intervals of
// size per they hold (the list overload), failing with ErrExpandTooLarge when
// there are more than limit of them in all. A limit of zero or less is none.
//
//	expand { Interval[1, 10] }        → {Interval[1,1], Interval[2,2], ..., Interval[10,10]}
//	expand { Interval[1, 10] } per 2  → {Interval[1,2], Interval[3,4], ..., Interval[9,10]}
func ExpandIntervalsUpTo(intervals []cqltypes.Interval, perVal fptypes.Value, limit int) (fptypes.Collection, error) {
	perAmount, perUnit, usable := expandGetStep(perVal)
	if !usable {
		return nil, nil
	}
	// An interval with a null bound contributes nothing, though its defined
	// bound still counts to read a per left out from.
	var live []cqltypes.Interval
	for _, iv := range intervals {
		if iv.Low != nil && iv.High != nil {
			live = append(live, iv)
		}
	}
	if len(live) == 0 {
		return nil, nil
	}
	// The intervals are ordered and merged by each family, after their bounds
	// are cut to per's precision: two of them merge when they overlap or meet
	// at that precision, which is what both references do.
	switch live[0].Low.(type) {
	case fptypes.Integer, fptypes.Decimal, fptypes.Quantity:
		return expandNumbers(live, perAmount, perUnit, limit)
	case fptypes.DateTime, fptypes.Date, fptypes.Time:
		// A per left out is read from the boundaries of the whole input.
		return expandTemporal(intervals, live, perAmount, perUnit, limit)
	}
	return nil, nil
}

// numRange is an interval read as closed decimals, in the precision of its own
// type, with what walks it: the step, its unit of scale, and the unit of the
// quantity if it is one. The partitions of the ranges of one unit are counted
// from the first of them.
type numRange struct {
	lo, hi    decimal.Decimal
	step, ulp decimal.Decimal
	unit      string
	lenient   bool
	cut       bool
	cutScale  int32
}

// perScale is the number of decimal places of a step, 0 for a whole one: the
// precision per truncates a Decimal to, and whose unit ends a partition.
func perScale(step decimal.Decimal) int32 {
	s := int32(0)
	for ; s < 28 && !step.Truncate(s).Equal(step); s++ {
	}
	return s
}

// cutTo truncates d toward zero to scale decimal places, leaving as it is a
// value that has no more of them: Truncate would write zeros to fill the scale.
func cutTo(d decimal.Decimal, scale int32) decimal.Decimal {
	if d.Exponent() >= -scale {
		return d
	}
	return d.Truncate(scale)
}

// newRange fills what walks a range: a per left out steps by one and cuts
// nothing, the bounds stay as written.
func newRange(lo, hi, step decimal.Decimal, perGiven bool, unit string) numRange {
	r := numRange{lo: lo, hi: hi, step: step, ulp: decimal.NewFromInt(1), unit: unit}
	if perGiven {
		r.cut, r.cutScale = true, perScale(step)
		if r.cutScale > 0 {
			r.ulp = decimal.New(1, -r.cutScale)
		}
	}
	// A per that is neither a whole number nor one unit of its own scale, as
	// 0.5, is out of the plan of the whole partitions: the last one, cut by the
	// high bound, is kept as it was.
	r.lenient = !step.Equal(step.Truncate(0)) && !step.Equal(r.ulp)
	return r
}

func expandNumbers(ordered []cqltypes.Interval, perAmount decimal.Decimal, perUnit string, limit int) (fptypes.Collection, error) {
	wholeStep := perAmount.IsZero() || perAmount.Equal(perAmount.Truncate(0))
	if _, ok := ordered[0].Low.(fptypes.Integer); ok && wholeStep {
		return expandIntegers(ordered, perAmount, limit)
	}

	perGiven := !perAmount.IsZero()
	step := decimal.NewFromInt(1)
	if perGiven {
		step = perAmount
	}
	one := decimal.NewFromInt(1)
	var ranges []numRange
	for _, iv := range ordered {
		var r numRange
		switch lo := iv.Low.(type) {
		case fptypes.Integer:
			hi, ok := iv.High.(fptypes.Integer)
			if _, first := ordered[0].Low.(fptypes.Integer); !ok || !first {
				continue
			}
			l, h := lo.Value(), hi.Value()
			if (!iv.LowClosed && l == math.MaxInt64) || (!iv.HighClosed && h == math.MinInt64) {
				continue
			}
			if !iv.LowClosed {
				l++
			}
			if !iv.HighClosed {
				h--
			}
			r = newRange(decimal.NewFromInt(l), decimal.NewFromInt(h), step, perGiven, "")
			// An integer holds every value of its own unit: 10 reaches 10.9 at
			// one decimal place.
			r.hi = r.hi.Add(one).Sub(r.ulp)
		case fptypes.Decimal:
			hi, ok := iv.High.(fptypes.Decimal)
			if _, first := ordered[0].Low.(fptypes.Decimal); !ok || !first {
				continue
			}
			r = newRange(lo.Value(), hi.Value(), step, perGiven, "")
			if !iv.LowClosed {
				r.lo = r.lo.Add(decimalEpsilon)
			}
			if !iv.HighClosed {
				r.hi = r.hi.Sub(decimalEpsilon)
			}
		case fptypes.Quantity:
			if _, first := ordered[0].Low.(fptypes.Quantity); !first {
				continue
			}
			lo2, hi2, stepQ, u, ok := quantityExpansion(iv, perAmount, perUnit)
			if !ok {
				continue
			}
			r = newRange(lo2, hi2, stepQ, perGiven, u)
			if !iv.LowClosed {
				r.lo = r.lo.Add(decimalEpsilon)
			}
			if !iv.HighClosed {
				r.hi = r.hi.Sub(decimalEpsilon)
			}
		default:
			continue
		}
		if r.cut {
			// Toward zero, as Truncate does.
			r.lo, r.hi = cutTo(r.lo, r.cutScale), cutTo(r.hi, r.cutScale)
		}
		ranges = append(ranges, r)
	}

	// Ranges of one unit together, in the order the units appear.
	unitOrder := map[string]int{}
	for _, r := range ranges {
		if _, ok := unitOrder[r.unit]; !ok {
			unitOrder[r.unit] = len(unitOrder)
		}
	}
	sort.SliceStable(ranges, func(i, j int) bool {
		if ui, uj := unitOrder[ranges[i].unit], unitOrder[ranges[j].unit]; ui != uj {
			return ui < uj
		}
		return ranges[i].lo.LessThan(ranges[j].lo)
	})
	var merged []numRange
	for _, r := range ranges {
		if n := len(merged); n > 0 && merged[n-1].unit == r.unit && !r.lo.GreaterThan(merged[n-1].hi.Add(r.ulp)) {
			if r.hi.GreaterThan(merged[n-1].hi) {
				merged[n-1].hi = r.hi
			}
			continue
		}
		merged = append(merged, r)
	}
	ranges = merged

	anchors := map[string]decimal.Decimal{}
	var result fptypes.Collection
	for _, r := range ranges {
		if r.step.IsZero() {
			continue
		}
		anchor, seen := anchors[r.unit]
		if !seen {
			anchor = r.lo
			anchors[r.unit] = anchor
		}
		wrap := decimalToValue
		if r.unit != "" {
			wrap = func(d decimal.Decimal) fptypes.Value {
				if v, ok := decimalToValue(d).(fptypes.Decimal); ok {
					return fptypes.NewQuantityFromDecimal(v.Value(), r.unit)
				}
				return nil
			}
		}
		// The first partition that starts on or after the low bound.
		q, rem := r.lo.Sub(anchor).QuoRem(r.step, 0)
		if rem.IsPositive() {
			q = q.Add(one)
		}
		start := anchor
		if !q.IsZero() {
			start = anchor.Add(q.Mul(r.step))
		}
		for ; ; start = start.Add(r.step) {
			end := start.Add(r.step).Sub(r.ulp)
			if end.GreaterThan(r.hi) {
				if !r.lenient || start.GreaterThan(r.hi) {
					break
				}
				end = r.hi
			}
			low, high := wrap(start), wrap(end)
			if low == nil || high == nil {
				break
			}
			result = append(result, cqltypes.NewInterval(low, high, true, true))
			if overLimit(len(result), limit) {
				return nil, ExpandLimitError(limit)
			}
		}
	}
	return result, nil
}

// expandIntegers is expandNumbers for Integer bounds and a whole step, in
// integers. A distance is read unsigned, exact where the signed one overflows.
func expandIntegers(ordered []cqltypes.Interval, perAmount decimal.Decimal, limit int) (fptypes.Collection, error) {
	step := uint64(1)
	wholeDomain := false
	if !perAmount.IsZero() {
		whole := perAmount.BigInt()
		switch {
		case whole.IsUint64():
			step = max(whole.Uint64(), 1)
		case whole.Cmp(new(big.Int).Lsh(big.NewInt(1), 64)) == 0:
			wholeDomain = true // 2^64: the span of the Integer domain
		default:
			return nil, nil // larger than any whole partition
		}
	}
	type ints struct{ lo, hi int64 }
	var ranges []ints
	for _, iv := range ordered {
		lo, lok := iv.Low.(fptypes.Integer)
		hi, hok := iv.High.(fptypes.Integer)
		if !lok || !hok {
			continue
		}
		l, h := lo.Value(), hi.Value()
		if (!iv.LowClosed && l == math.MaxInt64) || (!iv.HighClosed && h == math.MinInt64) {
			continue
		}
		if !iv.LowClosed {
			l++
		}
		if !iv.HighClosed {
			h--
		}
		if l <= h {
			ranges = append(ranges, ints{l, h})
		}
	}
	if len(ranges) == 0 {
		return nil, nil
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].lo < ranges[j].lo })
	merged := ranges[:1]
	for _, r := range ranges[1:] {
		last := &merged[len(merged)-1]
		if last.hi == math.MaxInt64 || r.lo <= last.hi+1 {
			last.hi = max(last.hi, r.hi)
			continue
		}
		merged = append(merged, r)
	}
	ranges = merged

	if wholeDomain {
		// One partition of 2^64 values holds the whole domain, and nothing less.
		if ranges[0].lo != math.MinInt64 || ranges[0].hi != math.MaxInt64 {
			return nil, nil
		}
		if overLimit(1, limit) {
			return nil, ExpandLimitError(limit)
		}
		return fptypes.Collection{cqltypes.NewInterval(fptypes.NewInteger(math.MinInt64), fptypes.NewInteger(math.MaxInt64), true, true)}, nil
	}

	anchor := uint64(ranges[0].lo) //nolint:gosec // two's-complement, read back the same way
	var result fptypes.Collection
	for _, r := range ranges {
		off := uint64(r.lo) - anchor //nolint:gosec // distance, exact: anchor <= lo
		if rem := off % step; rem != 0 {
			if step-rem > math.MaxUint64-off {
				continue
			}
			off += step - rem
		}
		span := uint64(r.hi) - anchor //nolint:gosec // distance, exact: anchor <= hi
		// Whole partitions only: the last value of one is off + step - 1.
		for off <= span && span-off >= step-1 {
			start := int64(anchor + off)          //nolint:gosec // inside the Integer range
			end := int64(anchor + off + step - 1) //nolint:gosec // at most hi
			result = append(result, cqltypes.NewInterval(fptypes.NewInteger(start), fptypes.NewInteger(end), true, true))
			if overLimit(len(result), limit) {
				return nil, ExpandLimitError(limit)
			}
			if span-off < step {
				break
			}
			off += step
		}
	}
	return result, nil
}

// temporalRank orders the precisions of a temporal, year the coarsest, in the
// scale temporalUnitPrecision uses: -1 for a value that is not temporal.
// A Time has no coarser precision than the hour.
func temporalRank(v fptypes.Value) int {
	switch t := v.(type) {
	case fptypes.DateTime:
		return datePartsRank(int(t.Precision()))
	case fptypes.Date:
		return datePartsRank(int(t.Precision()))
	case fptypes.Time:
		return int(t.Precision()) + temporalUnitPrecision("hour")
	}
	return -1
}

// datePartsRank maps the index of a date's last part (year, month, day, hour…)
// to its rank; the rank scale has a week between month and day.
func datePartsRank(parts int) int {
	if parts >= 2 {
		return parts + 1
	}
	return parts
}

// rankUnits are the units a rank names. The week has none: it is seven days.
var rankUnits = [...]string{"year", "month", "", "day", "hour", precMinute, precSecond, precMillisecond}

// expandTemporal expands Date, DateTime and Time intervals. all is the input
// as it came: a per left out is read from it, and it is ordered and merged here.
func expandTemporal(all, live []cqltypes.Interval, perAmount decimal.Decimal, perUnit string, limit int) (fptypes.Collection, error) {
	amount, walks := temporalStep(perAmount)
	if !walks {
		return nil, nil
	}
	unit := perUnit
	if unit == "" || unit == "1" {
		// «a per value will be constructed based on the coarsest precision of
		// the boundaries of the intervals in the input set».
		rank := len(rankUnits)
		for _, iv := range all {
			for _, b := range []fptypes.Value{iv.Low, iv.High} {
				if r := temporalRank(b); r >= 0 {
					rank = min(rank, r)
				}
			}
		}
		if rank == len(rankUnits) {
			return nil, nil
		}
		// A week names no precision: a Date of its width is a day.
		unit = rankUnits[max(rank, 0)]
		if unit == "" {
			unit = "day"
		}
	}
	unit, err := normalizeTemporalUnit(unit)
	if err != nil {
		return nil, err
	}
	if unit == "week" {
		// A step too large to count is too large for any whole partition.
		if amount > math.MaxInt/7 {
			return nil, nil
		}
		amount, unit = amount*7, "day"
	}
	perRank := temporalUnitPrecision(unit)

	type bounds struct{ lo, hi fptypes.Value }
	var ranges []bounds
	for _, iv := range live {
		low, high := iv.Low, iv.High
		var err error
		// start of and end of, at the boundary's own precision, before the cut.
		if !iv.LowClosed {
			if low, err = cqltypes.Successor(low); err != nil || low == nil {
				continue
			}
		}
		if !iv.HighClosed {
			if high, err = cqltypes.Predecessor(high); err != nil || high == nil {
				continue
			}
		}
		lr, hr := temporalRank(low), temporalRank(high)
		// «If the boundaries are less precise than per, the interval will not
		// contribute any results».
		if lr < perRank || hr < perRank {
			continue
		}
		lo, ok1 := truncateTemporal(low, unit)
		hi, ok2 := truncateTemporal(high, unit)
		if !ok1 || !ok2 {
			continue
		}
		if len(ranges) > 0 {
			_, firstTime := ranges[0].lo.(fptypes.Time)
			if _, thisTime := lo.(fptypes.Time); firstTime != thisTime {
				continue
			}
		}
		ranges = append(ranges, bounds{lo, hi})
	}
	if len(ranges) == 0 {
		return nil, nil
	}

	// Ordered, and merged where one starts within a unit of per's precision of
	// where the one before ends.
	oneUnit := func(v fptypes.Value) fptypes.Value {
		if next, err := DateAdd(v, 1, unit); err == nil && next != nil {
			return next
		}
		return v // the end of the calendar or of the day: nothing comes after
	}
	if _, isTime := ranges[0].lo.(fptypes.Time); isTime {
		oneUnit = func(v fptypes.Value) fptypes.Value {
			// A Time wraps when added to; its next unit stops at the day.
			if next, err := DateAdd(v, 1, unit); err == nil && next != nil {
				if c, err := compareVals(next, v); err == nil && c > 0 {
					return next
				}
			}
			return v
		}
	}
	sort.SliceStable(ranges, func(i, j int) bool {
		c, err := compareVals(ranges[i].lo, ranges[j].lo)
		return err == nil && c < 0
	})
	merged := ranges[:1]
	for _, r := range ranges[1:] {
		last := &merged[len(merged)-1]
		if c, err := compareVals(r.lo, oneUnit(last.hi)); err == nil && c <= 0 {
			if c, err := compareVals(r.hi, last.hi); err == nil && c > 0 {
				last.hi = r.hi
			}
			continue
		}
		merged = append(merged, r)
	}
	ranges = merged

	var result fptypes.Collection
	emit := func(lo, hi fptypes.Value) error {
		result = append(result, cqltypes.NewInterval(lo, hi, true, true))
		if overLimit(len(result), limit) {
			return ExpandLimitError(limit)
		}
		return nil
	}

	if _, isTime := ranges[0].lo.(fptypes.Time); isTime {
		unitMs := map[string]int{"hour": 3600000, precMinute: 60000, precSecond: 1000, precMillisecond: 1}[unit]
		if amount > math.MaxInt/unitMs {
			return nil, nil
		}
		stepMs := amount * unitMs
		if stepMs > 24*3600000 {
			return nil, nil // no whole partition fits in a day
		}
		dayMs := func(v fptypes.Value) int {
			t := v.(fptypes.Time)
			return t.Hour()*3600000 + t.Minute()*60000 + t.Second()*1000 + t.Millisecond()
		}
		anchor := dayMs(ranges[0].lo)
		for _, r := range ranges {
			lo, hi := dayMs(r.lo), dayMs(r.hi)
			start := anchor + (lo-anchor+stepMs-1)/stepMs*stepMs
			// Times do not wrap here: the last partition ends by hi, inside the day.
			for ; start+stepMs-unitMs <= hi; start += stepMs {
				a, aok := timeAt(start, unit)
				b, bok := timeAt(start+stepMs-unitMs, unit)
				if !aok || !bok {
					break
				}
				if err := emit(a, b); err != nil {
					return nil, err
				}
			}
		}
		return result, nil
	}

	anchor := ranges[0].lo
	startAt := func(k int) (fptypes.Value, bool) {
		if k > math.MaxInt/amount {
			return nil, false
		}
		v, err := DateAdd(anchor, k*amount, unit)
		return v, err == nil && v != nil
	}
	for _, r := range ranges {
		// The first partition that starts on or after the low bound: partition
		// k starts at the anchor plus k times per, added to the anchor each time
		// rather than to the one before, so a month end does not shrink the
		// months after it.
		reaches := func(k int) bool {
			v, ok := startAt(k)
			if !ok {
				return true
			}
			c, err := compareVals(v, r.lo)
			return err != nil || c >= 0
		}
		k := 0
		if !reaches(0) {
			top := 1
			for !reaches(top) {
				top *= 2
			}
			bot := top / 2 // does not reach
			for top-bot > 1 {
				if mid := (bot + top) / 2; reaches(mid) {
					top = mid
				} else {
					bot = mid
				}
			}
			k = top
		}
		for ; ; k++ {
			start, ok := startAt(k)
			if !ok {
				break
			}
			// The partition ends one unit before the next one starts, asked
			// without the next: at the end of the calendar there is none.
			end, err := DateAdd(start, amount-1, unit)
			if err != nil || end == nil {
				break
			}
			if c, err := compareVals(end, r.hi); err != nil || c > 0 {
				break
			}
			if err := emit(start, end); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

// timeAt is the Time of a millisecond of the day, written to the precision of
// unit.
func timeAt(ms int, unit string) (fptypes.Time, bool) {
	h, m, s, milli := ms/3600000, ms/60000%60, ms/1000%60, ms%1000
	var text string
	switch unit {
	case "hour":
		text = fmt.Sprintf("%02d", h)
	case precMinute:
		text = fmt.Sprintf("%02d:%02d", h, m)
	case precSecond:
		text = fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	default:
		text = fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, milli)
	}
	t, err := fptypes.NewTime(text)
	return t, err == nil
}

// truncateTemporal cuts a Date, DateTime or Time to the precision of unit, a
// boundary no less precise than it. It cuts the digits it holds: a DateTime keeps
// the offset it writes, or the one it is read at when it writes none, and is
// never moved to another.
func truncateTemporal(v fptypes.Value, unit string) (fptypes.Value, bool) {
	rank := temporalUnitPrecision(unit)
	switch t := v.(type) {
	case fptypes.Time:
		if rank < temporalUnitPrecision("hour") {
			return nil, false
		}
		n, ok := timeAt(t.Hour()*3600000+t.Minute()*60000+t.Second()*1000+t.Millisecond(), unit)
		if !ok {
			return nil, false
		}
		return n, true
	case fptypes.Date:
		var text string
		switch rank {
		case 0:
			text = fmt.Sprintf("%04d", t.Year())
		case 1:
			text = fmt.Sprintf("%04d-%02d", t.Year(), t.Month())
		default:
			text = fmt.Sprintf("%04d-%02d-%02d", t.Year(), t.Month(), t.Day())
		}
		d, err := fptypes.NewDate(text)
		return d, err == nil
	case fptypes.DateTime:
		text := fmt.Sprintf("%04d", t.Year())
		if rank >= 1 {
			text += fmt.Sprintf("-%02d", t.Month())
		}
		if rank >= 3 {
			text += fmt.Sprintf("-%02d", t.Day())
		}
		if rank >= 4 {
			text += fmt.Sprintf("T%02d", t.Hour())
		} else {
			text += "T"
		}
		if rank >= 5 {
			text += fmt.Sprintf(":%02d", t.Minute())
		}
		if rank >= 6 {
			text += fmt.Sprintf(":%02d", t.Second())
		}
		if rank >= 7 {
			text += fmt.Sprintf(".%03d", t.Millisecond())
		}
		writes := t.HasTZ() && rank >= 4 // an offset follows a time and nothing else
		if writes {
			switch off := t.TZOffset(); {
			case off == 0:
				text += "Z"
			case off < 0:
				text += fmt.Sprintf("-%02d:%02d", -off/60, -off%60)
			default:
				text += fmt.Sprintf("+%02d:%02d", off/60, off%60)
			}
		}
		n, err := fptypes.NewDateTime(text)
		if err != nil {
			return nil, false
		}
		if !writes {
			// No offset to write: it stays as it was read, at the offset of the
			// evaluation when the value has none. A cut coarser than the hour
			// takes the one of the evaluation, as a literal that coarse does.
			if minutes, ok := t.EffectiveOffset(); ok && rank >= 4 {
				n = cqltypes.WithDefaultOffset(n, time.Duration(minutes)*time.Minute)
			}
		}
		return n, true
	}
	return nil, false
}
