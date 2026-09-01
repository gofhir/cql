package eval

import (
	"fmt"

	fptypes "github.com/gofhir/fhirpath/types"
)

// A DateTime that writes no offset names an instant somewhere inside the window
// the legal offsets allow. These are the two offsets that bound it: +14:00 puts
// the instant earliest at UTC, -12:00 puts it latest.
const (
	earliestOffsetMinutes = 14 * 60
	latestOffsetMinutes   = -12 * 60
)

// offsetPair is one reading of a comparison, with the unwritten offset resolved
// to a specific one.
type offsetPair struct{ left, right fptypes.Value }

// offsetWindowOf reports the two readings that bound a comparison where exactly
// one side writes a timezone offset, and asymmetric=false when that is not the
// situation.
//
// Both sides have to be a DateTime. Only a DateTime can write an offset, so only
// a DateTime can have omitted one: a Date has no time of day to place and a Time
// has no day to place it on, and neither absence is the one this resolves.
// Comparing those against a DateTime is a different question, and the measures
// depend on the answer it already gives — ControllingHighBloodPressureFHIR writes
// `DBPReading.effective same day as "Most Recent Blood Pressure Day"`, whose
// second operand is `date from ...`.
func offsetWindowOf(left, right fptypes.Value) (low, high offsetPair, asymmetric bool) {
	l, lIsDateTime := left.(fptypes.DateTime)
	r, rIsDateTime := right.(fptypes.DateTime)
	if !lIsDateTime || !rIsDateTime {
		return low, high, false
	}
	// EffectiveOffset rather than HasTZ: a value told what offset to assume has a
	// known one, so the pair is not asymmetric and there is no window to bound.
	// Asking HasTZ alone meant the window still fired for a defaulted value, and
	// `before` declined where `<` answered.
	// And a value with no timezone frame is not a side that left its offset out.
	// A DateTime specified no finer than the day names a day, so there is no
	// offset that would turn it into an instant and nothing for the window to
	// bound: widening it across 26 hours turns a question the day settles —
	// `DateTime(2020,1,1,12) same day as DateTime(2020,1,1)` — into one nobody can
	// answer. Both sides are read as written instead, which is what
	// temporalComponentsAgainst does with them.
	if framelessTemporal(left) || framelessTemporal(right) {
		return low, high, false
	}

	_, lKnown := l.EffectiveOffset()
	_, rKnown := r.EffectiveOffset()
	if lKnown == rKnown {
		return low, high, false
	}

	// Resolve whichever side left it out, at each end of the window.
	if !lKnown {
		earliest, ok1 := atOffset(l, earliestOffsetMinutes)
		latest, ok2 := atOffset(l, latestOffsetMinutes)
		if !ok1 || !ok2 {
			return low, high, false
		}
		return offsetPair{earliest, right}, offsetPair{latest, right}, true
	}
	earliest, ok1 := atOffset(r, earliestOffsetMinutes)
	latest, ok2 := atOffset(r, latestOffsetMinutes)
	if !ok1 || !ok2 {
		return low, high, false
	}
	return offsetPair{left, earliest}, offsetPair{left, latest}, true
}

// atOffset re-reads a DateTime's written digits as being in a given offset,
// keeping its precision. The digits are what the author wrote; this says which
// clock they were written against.
func atOffset(v fptypes.DateTime, minutes int) (fptypes.DateTime, bool) {
	sign := "+"
	if minutes < 0 {
		sign, minutes = "-", -minutes
	}
	shifted, err := fptypes.NewDateTime(
		fmt.Sprintf("%s%s%02d:%02d", v.String(), sign, minutes/60, minutes%60))
	if err != nil {
		return v, false
	}
	return shifted, true
}
