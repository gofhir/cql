package types

import (
	"fmt"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

// CQL precision levels, from coarsest to finest. Millisecond is a level of its own:
// this is where CQL and FHIRPath part company. FHIRPath folds seconds and
// milliseconds into a single precision compared as a decimal, so @T10:30:00 and
// @T10:30:00.0 are equal there. CQL keeps millisecond separate, so the same pair is
// specified to different precisions and comparing them is unknown.
const (
	precYear = iota
	precMonth
	precDay
	precHour
	precMinute
	precSecond
	precMillisecond
)

// temporalValue is a temporal broken into components on the CQL precision scale,
// held in the frame it was written in. Normalizing to UTC is a decision about a
// pair of values, not about one of them, so the offset travels alongside the
// components rather than being folded into them here.
type temporalValue struct {
	parts     [7]int
	precision int
	hasOffset bool
	offset    int // minutes east of UTC
}

// temporalPartsOf breaks a temporal value into its components. Absent month and day
// take their lowest value, as fptypes does, so that a low-precision value can still
// be placed on a clock when a pair needs normalizing.
// Reports ok=false for anything that is not a Date, DateTime or Time.
func temporalPartsOf(v fptypes.Value) (value temporalValue, ok bool) {
	lowest := func(component int) int {
		if component == 0 {
			return 1
		}
		return component
	}

	switch t := v.(type) {
	case fptypes.DateTime:
		precision := int(t.Precision())
		if precision > precMillisecond {
			precision = precMillisecond
		}
		return temporalValue{
			parts: [7]int{t.Year(), lowest(t.Month()), lowest(t.Day()),
				t.Hour(), t.Minute(), t.Second(), t.Millisecond()},
			precision: precision,
			hasOffset: t.HasTZ(),
			offset:    t.TZOffset(),
		}, true
	case fptypes.Date:
		// A date carries no offset, and its precision is already on the shared scale
		return temporalValue{
			parts:     [7]int{t.Year(), lowest(t.Month()), lowest(t.Day()), 0, 0, 0, 0},
			precision: int(t.Precision()),
		}, true
	case fptypes.Time:
		// Time precision counts from the hour, so it shifts onto the shared scale
		precision := int(t.Precision()) + precHour
		if precision > precMillisecond {
			precision = precMillisecond
		}
		return temporalValue{
			parts:     [7]int{0, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Millisecond()},
			precision: precision,
		}, true
	}
	return value, false
}

// inUTC re-expresses the components at UTC so two values written against different
// offsets can be read on the same footing.
func (v temporalValue) inUTC() temporalValue {
	instant := time.Date(
		v.parts[precYear], time.Month(v.parts[precMonth]), v.parts[precDay],
		v.parts[precHour], v.parts[precMinute], v.parts[precSecond],
		v.parts[precMillisecond]*1e6,
		time.FixedZone("", v.offset*60),
	).UTC()

	shifted := v
	shifted.parts = [7]int{
		instant.Year(), int(instant.Month()), instant.Day(),
		instant.Hour(), instant.Minute(), instant.Second(), instant.Nanosecond() / 1e6,
	}
	shifted.offset = 0
	return shifted
}

// CompareTemporal orders two values under CQL's precision rule, deferring to
// fptypes for everything except the one place the two specifications disagree.
//
// fptypes decides the comparison first, which keeps its handling of timezones, of
// Date against DateTime, and of the coarser precision mismatches it already reports.
// The extra step here is the millisecond: when both values agree on every component
// they share but one is specified to a finer precision than the other, the answer is
// unknown and ErrPrecisionMismatch is returned, the same sentinel fptypes uses.
//
// The normalization rule below mirrors fptypes exactly, and has to. Reading one
// value at UTC while the other stays in local components compares two different
// frames: @2020-06-01T02:00:00+05:00 falls on the 31st of May at UTC, so against the
// date @2020-05-31 it would look equal as far as both are specified, and a
// comparison fptypes settles would be reported as unknown.
//
//	@T15:59:59 vs @T15:59:59.999   // unknown in CQL, -1 to FHIRPath
//	@T15:59:59 vs @T20:59:59.999   // -1, decided at the hour
func CompareTemporal(a, b fptypes.Value) (int, error) {
	ordered, ok := a.(fptypes.Comparable)
	if !ok {
		return 0, fmt.Errorf("cannot compare %s", a.Type())
	}
	cmp, err := ordered.Compare(b)
	if err != nil {
		if decided, ok := compareAcrossAbsentOffset(a, b); ok {
			return decided, nil
		}
		return cmp, err
	}

	left, leftOK := temporalPartsOf(a)
	right, rightOK := temporalPartsOf(b)
	if !leftOK || !rightOK || left.precision == right.precision {
		return cmp, nil
	}

	// Offsets only come into play when both values carry one and they differ,
	// which is the only case fptypes converts for
	if left.hasOffset && right.hasOffset && left.offset != right.offset {
		left = left.inUTC()
		right = right.inUTC()
	}

	shared := left.precision
	if right.precision < shared {
		shared = right.precision
	}
	for level := precYear; level <= shared; level++ {
		if left.parts[level] != right.parts[level] {
			// Decided by a component both values specify
			return cmp, nil
		}
	}
	return 0, fptypes.ErrPrecisionMismatch
}

// Legal timezone offsets run from -12:00 to +14:00, so a value that does not
// write one names an instant somewhere in a 26-hour window.
const (
	offsetWindowEarliest = 14 * 60 // minutes: the offset that puts the instant earliest at UTC
	offsetWindowLatest   = 12 * 60 // and the one that puts it latest
)

// compareAcrossAbsentOffset orders a pair where one value writes a timezone
// offset and the other does not.
//
// fptypes declines this pair, reporting the precision mismatch it reports for a
// value specified to a coarser precision — and reporting it even where both are
// specified to the same precision, which is what gives the diagnosis away. It is
// not a precision that is missing, it is an offset. (Present in fhirpath v1.6.0
// and v1.8.0 alike, and reported upstream. This delegates first, so when Compare
// stops declining, none of this is reached.)
//
// Declining is wrong because an absent offset resolves rather than invalidates:
//
//	CQL, DateTime Literals: "If no timezone offset is specified, the timezone
//	offset of the evaluation request timestamp is used" — and extracting it gives
//	"the timezone offset of the evaluation request, not null".
//
//	FHIRPath, Comparison: "either both values have no timezone offset specified,
//	or both values are converted to a common timezone offset".
//
// This does not reach for the evaluation request's offset, which types has no way
// to ask for. It answers only where the answer does not depend on it: the
// unwritten offset leaves an instant uncertain across a 26-hour window, so the
// order is knowable exactly when that whole window — widened by whatever the
// coarser precision leaves open — falls on one side. Ten months apart is
// knowable; an hour apart is not, and stays unknown.
//
// That is the same rule the uncertainty operators follow, for the same reason:
// knowable outside the range, unknown inside. Widening rather than narrowing is
// deliberate. Every value this declines was already being declined, so the only
// thing an over-wide margin costs is an answer that was not being given anyway,
// while an over-narrow one would state an order that the offset could reverse.
func compareAcrossAbsentOffset(a, b fptypes.Value) (int, bool) {
	// Both sides have to be a DateTime. Only a DateTime can write an offset, so
	// only a DateTime can have omitted one — a Date has no time of day to place
	// and a Time has no day to place it on, and neither absence is the one this
	// resolves. Naming the type that qualifies rather than listing the ones that
	// do not is what keeps this from widening on its own: a first version excluded
	// Date and forgot Time, and since a Time reports no offset and lands at year
	// zero on the clock below, the gap cleared the margin and an order was
	// returned for a pair fhirpath had refused outright. `@2020-03-05T10:00:00Z >
	// @T10:00:00` answered true, while the same pair with no offset written kept
	// erroring — so the compensation invented both an order and a disagreement.
	if _, ok := a.(fptypes.DateTime); !ok {
		return 0, false
	}
	if _, ok := b.(fptypes.DateTime); !ok {
		return 0, false
	}

	left, leftOK := temporalPartsOf(a)
	right, rightOK := temporalPartsOf(b)
	if !leftOK || !rightOK || left.hasOffset == right.hasOffset {
		return 0, false
	}

	// Both sides on one clock: the written offset resolved, the unwritten one
	// read as if it were UTC so the window below can be measured from it.
	leftAt := left.inUTC().instant()
	rightAt := right.inUTC().instant()

	// How far apart they have to be for no offset to reorder them. The window is
	// one-sided per operand, and the coarser precision widens it because a value
	// specified to the day names any instant within that day.
	margin := time.Duration(offsetWindowEarliest+offsetWindowLatest) * time.Minute
	margin += coarserSpan(left.precision, right.precision)

	gap := leftAt.Sub(rightAt)
	switch {
	case gap <= -margin:
		return -1, true
	case gap >= margin:
		return 1, true
	}
	return 0, false
}

// instant places the components on the clock. The components are already at UTC
// by the time this is called, so the zone is fixed rather than assumed.
func (v temporalValue) instant() time.Time {
	return time.Date(
		v.parts[precYear], time.Month(v.parts[precMonth]), v.parts[precDay],
		v.parts[precHour], v.parts[precMinute], v.parts[precSecond],
		v.parts[precMillisecond]*1e6, time.UTC)
}

// coarserSpan reports how much the coarser of two precisions leaves open. A value
// written to the day names any instant in that day, so the span is what has to be
// added to the offset window before an order can be called.
//
// The year and month spans take their longest reading — 366 and 31 days — since
// widening only ever declines an answer, never states a wrong one.
func coarserSpan(a, b int) time.Duration {
	coarser := a
	if b < coarser {
		coarser = b
	}
	switch coarser {
	case precYear:
		return 366 * 24 * time.Hour
	case precMonth:
		return 31 * 24 * time.Hour
	case precDay:
		return 24 * time.Hour
	case precHour:
		return time.Hour
	case precMinute:
		return time.Minute
	case precSecond:
		return time.Second
	}
	return time.Millisecond
}
