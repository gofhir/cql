package eval

import (
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

// comparisons is what a comparison of two temporals at a precision can come to:
// the set of results, -1, 0 and 1, that the values it stands for can give, and
// whether some part of it is unknown (a component the precision asks for that a
// value does not have).
//
// A pair read one way has one result. Two things make it a set: a DateTime that
// writes no offset, which is somewhere in a window of offsets
// (eval/timing_offset.go), and a DateTime to the hour read in an offset that
// cuts its hour in two (hourSpread). A predicate answers when it has the same
// answer for every member of the set, and nothing is unknown; otherwise the
// answer is unknown.
type comparisons struct {
	seen      [3]bool // seen[c+1] is whether the pair can compare as c
	uncertain bool
}

func comparisonOf(cmp int, certain bool) comparisons {
	if !certain {
		return comparisons{uncertain: true}
	}
	var s comparisons
	s.seen[cmp+1] = true
	return s
}

func (s comparisons) union(o comparisons) comparisons {
	for i := range s.seen {
		s.seen[i] = s.seen[i] || o.seen[i]
	}
	s.uncertain = s.uncertain || o.uncertain
	return s
}

// filled adds every result between the least and the greatest one in the set.
// Truncating to a precision is monotonic in the instant and the instants of a
// window or of a span are continuous, so a pair that can compare as -1 and as 1
// can compare as 0 as well.
func (s comparisons) filled() comparisons {
	first, last := -1, -1
	for i, in := range s.seen {
		if in {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	for i := first; i >= 0 && i <= last; i++ {
		s.seen[i] = true
	}
	return s
}

// holds is the answer of a predicate over the set: true or false where it gives
// that for every member and nothing is unknown, null otherwise.
func (s comparisons) holds(predicate func(cmp int) bool) fptypes.Value {
	if s.uncertain {
		return nil
	}
	yes, no := false, false
	for i, in := range s.seen {
		if !in {
			continue
		}
		if predicate(i - 1) {
			yes = true
		} else {
			no = true
		}
	}
	switch {
	case yes && !no:
		return fptypes.NewBoolean(true)
	case no && !yes:
		return fptypes.NewBoolean(false)
	}
	return nil
}

// temporalCompareSet compares two temporal values up to the given precision, and
// gives every result the pair can come to.
//
// One side writing an offset and the other not is not a pair of components to
// line up: reading components moves the written one to a frame and leaves the
// other in its local digits, which compares two different frames — the mistake
// CompareTemporal's own comment warns about. It is resolved over the window of
// offsets before any component is read: both ends of it, and the offset of the
// other side as written (plan 024, plan 025).
//
// Skipped at a precision coarser than the hour. Both ends of the window would
// be read as written and agree anyway — atOffset keeps the digits — so this
// only saves the two readings. See readAsWritten.
func temporalCompareSet(frame *time.Location, left, right fptypes.Value, precision string) comparisons {
	if !coarserThanHour(precision) {
		if lo, hi, asymmetric := offsetWindowOf(left, right); asymmetric {
			set := temporalCompareRead(frame, lo.left, lo.right, precision, true).
				union(temporalCompareRead(frame, hi.left, hi.right, precision, true))
			if same, ok := offsetOfTheOtherOf(left, right); ok {
				set = set.union(temporalCompareRead(frame, same.left, same.right, precision, false))
			}
			// Across the window both values are read in the evaluation offset,
			// where truncating to a precision is monotonic in the instant, so
			// the results between the two ends are results too.
			return set.filled()
		}
	}
	return temporalCompareRead(frame, left, right, precision, false)
}

// temporalCompareRead compares two temporals read one way: in the frame when
// inFrame is set and the pair is not read as written, otherwise as
// readingFrameOf decides.
func temporalCompareRead(frame *time.Location, left, right fptypes.Value, precision string, inFrame bool) comparisons {
	in := readingFrameOf(left, right, precision, frame)
	if inFrame && precision != "" && !readAsWritten(left, right, precision) {
		in = frame
	}
	if set, spread := spreadComparisons(left, right, precision, in); spread {
		return set
	}
	return comparisonOf(compareComponentsIn(left, right, precision, in))
}

// hourSpread reports the first and last instant of a DateTime to the hour that
// an offset cuts in two: read in in, it does not start on the hour.
//
// A DateTime to the hour is a stretch of an hour on its own clock, and
// `@2020-01-01T00Z` is 05:30 to 06:29 at +05:30, in neither the hour 5 nor the
// 6. Each such value is its span, as the pair of values is for `<` and `=`
// (types.compareAcrossPartialOffsets, plan 008); a value that is cut on the hour
// or that is more specific, or in the digits it was written in, is not.
func hourSpread(v fptypes.Value, in *time.Location) (first, last time.Time, ok bool) {
	dt, isDateTime := v.(fptypes.DateTime)
	if !isDateTime || in == nil || dt.Precision() != fptypes.DTHourPrecision {
		return first, last, false
	}
	first, known := instantOf(dt)
	if !known {
		return first, last, false
	}
	if read := first.In(in); read.Minute() == 0 && read.Second() == 0 && read.Nanosecond() == 0 {
		return first, last, false
	}
	return first, first.Add(time.Hour - time.Millisecond), true
}

// readComponents is a temporal read as a pair of components and the index of the
// finest it has.
type readComponents struct {
	comps []int
	max   int
}

// cornersOf is the components to compare of a value: its first and last instant
// where it is spread, which have every component, and otherwise what it was
// written with.
func cornersOf(v fptypes.Value, in *time.Location) (corners []readComponents, spread bool) {
	if first, last, ok := hourSpread(v, in); ok {
		return []readComponents{{componentsOf(first.In(in)), 6}, {componentsOf(last.In(in)), 6}}, true
	}
	comps, maxPrec := temporalComponentsAgainst(v, in)
	return []readComponents{{comps, maxPrec}}, false
}

// spreadComparisons compares a pair where a side is an hour spread, and reports
// whether it is one. Each corner of one is compared with each of the other, by
// the rule of components truncated at the precision, and the results between the
// least and the greatest are filled in: the span is continuous.
func spreadComparisons(left, right fptypes.Value, precision string, in *time.Location) (comparisons, bool) {
	p := precisionIndex(precision)
	if p < 0 || in == nil {
		return comparisons{}, false
	}
	lCorners, lSpread := cornersOf(left, in)
	rCorners, rSpread := cornersOf(right, in)
	if !lSpread && !rSpread {
		return comparisons{}, false
	}
	var set comparisons
	for _, l := range lCorners {
		for _, r := range rCorners {
			set = set.union(comparisonOf(compareThrough(l, r, 0, p)))
		}
	}
	return set.filled(), true
}

// compareThrough lines up the components from index from to index to, and is
// unknown where either side lacks one it reaches.
func compareThrough(l, r readComponents, from, to int) (int, bool) {
	if l.comps == nil || r.comps == nil {
		return 0, false
	}
	for i := from; i <= to; i++ {
		if i > l.max || i > r.max {
			return 0, false
		}
		if l.comps[i] < r.comps[i] {
			return -1, true
		}
		if l.comps[i] > r.comps[i] {
			return 1, true
		}
	}
	return 0, true
}

// instantOf is the instant a DateTime names, where it has an offset, stated or
// assumed.
func instantOf(t fptypes.DateTime) (time.Time, bool) {
	minutes, ok := t.EffectiveOffset()
	if !ok {
		return time.Time{}, false
	}
	at := t.ToTime()
	if !t.HasTZ() {
		at = time.Date(at.Year(), at.Month(), at.Day(), at.Hour(), at.Minute(), at.Second(),
			at.Nanosecond(), time.FixedZone("", minutes*60))
	}
	return at, true
}

// componentsOf breaks an instant into year, month, day, hour, minute, second and
// millisecond.
func componentsOf(t time.Time) []int {
	return []int{t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond() / 1e6}
}
