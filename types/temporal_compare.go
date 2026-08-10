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
