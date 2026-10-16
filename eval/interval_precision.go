package eval

import (
	"time"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/funcs"
	cqltypes "github.com/gofhir/cql/types"
)

// intervalPhraseAtPrecision answers a timing phrase between two temporal
// intervals at the precision it states, and reports whether it applies.
//
// The specification defines each phrase on the intervals' start and end points
// compared at that precision: `A before day of B` is `end of A before day of
// start of B`, `A starts day of B` is `start of A same day as start of B and
// end of A same day or before end of B`, and so on. These phrases compared the
// raw bounds at full precision instead, ignoring the precision named — so
// `Interval[10:00, 12:00] before day of Interval[13:00, …]` on one day was true
// where the Java engine and cql-execution answer false, and `meets day of`,
// `starts day of`, `ends day of` and `overlaps day of` across a day were false
// where both answer true.
//
// Each point comparison is the one the point phrases make (temporalBefore and
// the rest), so the phrases read offsets as those do (see readAsWritten), and a
// point without the precision asked leaves its part unknown. The parts combine
// as `and` and `or` do: `starts` with an uncertain start and an end already
// past is false.
func (e *Evaluator) intervalPhraseAtPrecision(a, b cqltypes.Interval, op ast.TimingOp) (fptypes.Value, bool, error) {
	if !phraseAtPrecision(op) || !samePointKind(a, b, op.Precision) {
		return nil, false, nil
	}
	pts, err := endpointsOf(a, b, op.Precision)
	if err != nil {
		return nil, true, err
	}
	return pts.phrase(op), true, nil
}

// phraseAtPrecision reports whether a phrase is one answered here: it states a
// precision this engine knows, and is not `same <precision> as`, which
// intervalSameAs already compares at the precision.
func phraseAtPrecision(op ast.TimingOp) bool {
	if precisionIndex(op.Precision) < 0 {
		return false
	}
	switch op.Kind {
	case ast.TimingBeforeOrAfter, ast.TimingMeets, ast.TimingOverlaps, ast.TimingStarts, ast.TimingEnds:
		return true
	case ast.TimingSameAs:
		return op.Before || op.After
	}
	return false
}

// samePointKind reports whether two intervals hold the same kind of temporal
// point, at a precision that kind has. A Date and a DateTime are one kind here,
// since CQL compares them by converting the Date. Anything else — a Date
// against a Time, an Integer bound, a Time at the day — is not answered here
// and goes on to the phrase without its precision, which is what it did before
// these phrases read one. The reference translator rejects all of them.
func samePointKind(a, b cqltypes.Interval, precision string) bool {
	ka, kb := pointKind(a), pointKind(b)
	if ka == "" || ka != kb {
		return false
	}
	return ka != "Time" || !coarserThanHour(precision)
}

// properlyContainsPointAtPrecision answers `properly includes <precision> of`
// a point, and reports whether it applies: the point strictly after the
// interval's start and strictly before its end, both at that precision, as the
// Java engine answers. A point on the start day is not properly inside at the
// day, though the instants are apart.
func properlyContainsPointAtPrecision(iv cqltypes.Interval, v fptypes.Value, precision string) (fptypes.Value, bool, error) {
	at := cqltypes.NewInterval(v, v, true, true)
	if precisionIndex(precision) < 0 || !samePointKind(iv, at, precision) {
		return nil, false, nil
	}
	pts, err := endpointsOf(iv, at, precision)
	if err != nil {
		return nil, true, err
	}
	return ThreeValuedAnd(pts.cmp(temporalAfter, pts.startB, pts.startA), pts.cmp(temporalBefore, pts.startB, pts.endA)), true, nil
}

// startOf and endOf are an interval's first and last points, with a closed
// null bound read as the minimum or maximum value of the point type, and an
// open one as an unknown start or end.
//
// That is the specification's reading of a null boundary, and the Java
// engine's: `Interval[@2020-01-01T00:00Z, null] overlaps day of
// Interval[@2020-03-01T00:00Z, …]` is true. Reading every null as unknown made
// that null, and `properly includes day of` a point inside an open-ended
// period null where it had been true. The extreme is the actual value, so a
// closed null start and an explicit minimum DateTime are the same day.
func startOf(iv cqltypes.Interval) (fptypes.Value, error) {
	if iv.Low == nil && iv.LowClosed {
		return extremeLike(iv.High, false), nil
	}
	return iv.Start()
}

func endOf(iv cqltypes.Interval) (fptypes.Value, error) {
	if iv.High == nil && iv.HighClosed {
		return extremeLike(iv.Low, true), nil
	}
	return iv.End()
}

// extremeLike is the minimum or maximum value of the type of sample, or nil
// when there is no sample to take the type from.
func extremeLike(sample fptypes.Value, maximum bool) fptypes.Value {
	var kind string
	switch sample.(type) {
	case fptypes.Date:
		kind = "date"
	case fptypes.DateTime:
		kind = "datetime"
	case fptypes.Time:
		kind = "time"
	default:
		return nil
	}
	v, err := temporalExtreme(kind, maximum)
	if err != nil {
		return nil
	}
	return v
}

// temporalExtreme is the minimum or maximum value of a temporal type, by its
// lower-case name. It is what `minimum DateTime` and the rest evaluate to, and
// what a closed null bound stands for, so the two are always the same value.
//
// A DateTime's is stated at UTC, as the specification and the reference engine
// give it. Not the evaluation offset: at an eastern one the minimum would sit
// before the start of the range it defines, and at a western one the maximum
// past its end. A stated Z moves neither.
func temporalExtreme(kind string, maximum bool) (fptypes.Value, error) {
	switch kind {
	case "datetime":
		if maximum {
			return fptypes.NewDateTime("9999-12-31T23:59:59.999Z")
		}
		return fptypes.NewDateTime("0001-01-01T00:00:00.000Z")
	case "date":
		if maximum {
			return fptypes.NewDate("9999-12-31")
		}
		return fptypes.NewDate("0001-01-01")
	case "time":
		if maximum {
			return fptypes.NewTime("23:59:59.999")
		}
		return fptypes.NewTime("00:00:00.000")
	}
	return nil, nil
}

// endpoints are the first and last points two intervals include, and the
// precision a phrase compares them at.
type endpoints struct {
	startA, endA, startB, endB fptypes.Value
	precision                  string
}

func endpointsOf(a, b cqltypes.Interval, precision string) (endpoints, error) {
	pts := endpoints{precision: precision}
	var err error
	if pts.startA, err = startOf(a); err != nil {
		return pts, err
	}
	if pts.endA, err = endOf(a); err != nil {
		return pts, err
	}
	if pts.startB, err = startOf(b); err != nil {
		return pts, err
	}
	pts.endB, err = endOf(b)
	return pts, err
}

// cmp asks one of the point comparisons at the phrase's precision, unknown
// where either point is. Those comparisons decline rather than fail — an
// uncertain one is null — so an error from one is read the same way.
func (p endpoints) cmp(compare func(x, y fptypes.Value, precision string) (fptypes.Value, error), x, y fptypes.Value) fptypes.Value {
	if x == nil || y == nil {
		return nil
	}
	v, err := compare(x, y, p.precision)
	if err != nil {
		return nil
	}
	return v
}

// phrase answers a timing phrase on the endpoints. Which phrases reach it is
// phraseAtPrecision's to say.
func (p endpoints) phrase(op ast.TimingOp) fptypes.Value {
	switch op.Kind {
	case ast.TimingBeforeOrAfter:
		if op.Before {
			return p.cmp(temporalBefore, p.endA, p.startB)
		}
		return p.cmp(temporalAfter, p.startA, p.endB)
	case ast.TimingSameAs:
		if op.Before {
			return p.cmp(temporalSameOrBefore, p.endA, p.startB)
		}
		return p.cmp(temporalSameOrAfter, p.startA, p.endB)
	case ast.TimingMeets:
		return p.meets(op)
	case ast.TimingOverlaps:
		return p.overlaps(op)
	case ast.TimingStarts:
		return ThreeValuedAnd(p.cmp(temporalSameAs, p.startA, p.startB), p.cmp(temporalSameOrBefore, p.endA, p.endB))
	case ast.TimingEnds:
		return ThreeValuedAnd(p.cmp(temporalSameAs, p.endA, p.endB), p.cmp(temporalSameOrAfter, p.startA, p.startB))
	}
	return nil
}

func (p endpoints) meets(op ast.TimingOp) fptypes.Value {
	before := successorAt(p.endA, p.startB, p.precision)
	after := successorAt(p.endB, p.startA, p.precision)
	switch {
	case op.Before:
		return before
	case op.After:
		return after
	}
	return ThreeValuedOr(before, after)
}

// overlaps is the two intervals sharing a point at the precision; `overlaps
// before` is that and A starting first, and `overlaps after` that and A ending
// last. The direction is added to the whole overlap, as the specification
// words it and the Java engine computes it, so an unknown end the overlap
// depends on leaves the phrase unknown too.
func (p endpoints) overlaps(op ast.TimingOp) fptypes.Value {
	overlap := ThreeValuedAnd(p.cmp(temporalSameOrBefore, p.startA, p.endB), p.cmp(temporalSameOrAfter, p.endA, p.startB))
	switch {
	case op.Before:
		return ThreeValuedAnd(p.cmp(temporalBefore, p.startA, p.startB), overlap)
	case op.After:
		return ThreeValuedAnd(p.cmp(temporalAfter, p.endA, p.endB), overlap)
	}
	return overlap
}

// pointKind names the kind of temporal point every bound of an interval is —
// a calendar point (Date or DateTime) or a Time — or "" when a bound is not
// temporal, the bounds disagree, or there is none to ask.
func pointKind(iv cqltypes.Interval) string {
	kind := ""
	for _, v := range []fptypes.Value{iv.Low, iv.High} {
		if v == nil {
			continue
		}
		k := temporalKind(v)
		if k == "" || (kind != "" && k != kind) {
			return ""
		}
		kind = k
	}
	return kind
}

func temporalKind(v fptypes.Value) string {
	switch v.(type) {
	case fptypes.Date, fptypes.DateTime:
		return "calendar"
	case fptypes.Time:
		return "Time"
	}
	return ""
}

// successorAt reports whether next is the point right after prev at a
// precision — whether the intervals they end and start meet there. At the day,
// 2020-01-01T12:00 and 2020-01-02T13:00 meet: the 2nd is the day after the 1st.
//
// At the hour and finer, a pair where only one side states an offset is
// unknown. An ordering can be settled at the two ends of the window the other
// could be in, as temporalCompareAtPrecision settles it, because truncating is
// monotonic in the instant; being the next hour is not, and both ends can say
// no while an offset between them says yes.
func successorAt(prev, next fptypes.Value, precision string) fptypes.Value {
	if prev == nil || next == nil {
		return nil
	}
	if !coarserThanHour(precision) {
		if _, _, asymmetric := offsetWindowOf(prev, next); asymmetric {
			return nil
		}
	}
	return successorOf(prev, next, precision)
}

// successorOf decides adjacency on the components. A point without the
// precision stands for a range of units — 2020 at the day is every day of the
// year — and so does the set of days after it, which is contiguous: from the
// 2nd of January to the 1st of January 2021. The next point is the successor
// where both are single units and equal; it cannot be where the two ranges do
// not touch; and it might be otherwise.
func successorOf(prev, next fptypes.Value, precision string) fptypes.Value {
	p := precisionIndex(precision)
	pc, pMax, nc, nMax := componentsAtPrecision(prev, next, precision)
	if pc == nil || nc == nil {
		return nil
	}
	_, isTime := prev.(fptypes.Time)
	from := 0
	if isTime {
		// A Time reaches here only at the hour or finer; see samePointKind.
		from = precisionIndex("hour")
		pc = append([]int{2000, 1, 1}, pc[3:]...)
		nc = append([]int{2000, 1, 1}, nc[3:]...)
	}
	succLow := addOneAt(completion(pc, pMax, false), p)
	succHigh := addOneAt(completion(pc, pMax, true), p)
	if isTime {
		// Past the end of the day a time has no successor.
		if pastTheDay(succLow) {
			return fptypes.NewBoolean(false)
		}
		if pastTheDay(succHigh) {
			succHigh = completion(pc[:3:3], 2, true)
		}
	}
	nextLow, nextHigh := completion(nc, nMax, false), completion(nc, nMax, true)
	if lessFrom(succHigh, nextLow, from, p) || lessFrom(nextHigh, succLow, from, p) {
		return fptypes.NewBoolean(false)
	}
	if pMax >= p && nMax >= p {
		return fptypes.NewBoolean(true)
	}
	return nil
}

// The first and last value each component can take: month, day, hour,
// minute, second and millisecond. The last day depends on the month and is
// worked out where it is used.
var (
	firstComponent = [7]int{0, 1, 1, 0, 0, 0, 0}
	lastComponent  = [7]int{0, 12, 0, 23, 59, 59, 999}
)

// completion fills the components after the finest one a point has with the
// first (high=false) or last (high=true) value each can take, so a point
// without a precision becomes the first or last unit it could be.
func completion(c []int, have int, high bool) []int {
	out := make([]int, 7)
	copy(out, c)
	for i := have + 1; i < 7; i++ {
		switch {
		case !high:
			out[i] = firstComponent[i]
		case i == 2:
			out[i] = time.Date(out[0], time.Month(out[1])+1, 0, 0, 0, 0, 0, time.UTC).Day()
		default:
			out[i] = lastComponent[i]
		}
	}
	return out
}

func pastTheDay(c []int) bool { return c[0] != 2000 || c[1] != 1 || c[2] != 1 }

// lessFrom reports whether a comes before b on the components from index from
// to to.
func lessFrom(a, b []int, from, to int) bool {
	for i := from; i <= to; i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// addOneAt adds one unit of the precision at index p to components truncated
// there, carrying through the calendar.
func addOneAt(c []int, p int) []int {
	t := firstComponent
	copy(t[:p+1], c[:p+1])
	at := time.Date(t[0], time.Month(t[1]), t[2], t[3], t[4], t[5], t[6]*int(time.Millisecond), time.UTC)
	switch p {
	case 0:
		at = at.AddDate(1, 0, 0)
	case 1:
		at = at.AddDate(0, 1, 0)
	case 2:
		at = at.AddDate(0, 0, 1)
	default:
		at = at.Add([4]time.Duration{time.Hour, time.Minute, time.Second, time.Millisecond}[p-3])
	}
	return []int{at.Year(), int(at.Month()), at.Day(), at.Hour(), at.Minute(), at.Second(), at.Nanosecond() / int(time.Millisecond)}
}

// widenForPrecision widens a date range pushed down to a provider from a phrase
// at a precision, so the rows it returns still include every one the phrase
// can keep.
//
// A provider compares instants. At the day a row is in the interval when its
// day is, read as written, so it can sit a whole unit before the interval's
// first instant — and a further 26 hours, the widest two written offsets can
// differ, before that. Each bound is moved out by one unit of the precision
// and two days. A bound written only to the year or the month cannot be moved
// by days — DateAdd keeps a value's precision and rounds the step to it — so it
// is moved two of whichever unit is coarser, its own or the phrase's: more
// than a unit of the phrase and two days either way. The rows that brings in that the phrase does not keep, it drops;
// a bound that would leave the calendar becomes the calendar's first or last
// value instead.
func widenForPrecision(rng fptypes.Value, precision string) fptypes.Value {
	iv, ok := rng.(cqltypes.Interval)
	if !ok || precisionIndex(precision) < 0 {
		return rng
	}
	widen := func(v fptypes.Value, sign int) fptypes.Value {
		if v == nil {
			return nil
		}
		if temporalKind(v) != "calendar" {
			return v
		}
		var moved fptypes.Value
		var err error
		if _, written := temporalComponentsAgainst(v, true); written < precisionIndex("day") {
			coarser := min(written, precisionIndex(precision))
			moved, err = funcs.DateAdd(v, 2*sign, []string{"year", "month"}[coarser])
		} else if moved, err = funcs.DateAdd(v, sign, precision); err == nil {
			moved, err = funcs.DateAdd(moved, 2*sign, "day")
		}
		if err != nil {
			return extremeLike(v, sign > 0)
		}
		return moved
	}
	return cqltypes.NewInterval(widen(iv.Low, -1), widen(iv.High, 1), iv.LowClosed, iv.HighClosed)
}
