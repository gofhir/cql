package types

import (
	"errors"

	fptypes "github.com/gofhir/fhirpath/types"
)

// UndecidableComparison reports that a comparison could not be made, which in CQL
// is an answer — null — rather than a failure.
//
// It lives here because the two packages that need it, eval and funcs, both
// import this one and it imports neither. Before, each had its own copy, they had
// drifted apart, and about thirty call sites read one or the other:
//
//	                          eval    funcs
//	ErrPrecisionMismatch       yes     yes
//	"ambiguous comparison"     yes     yes
//	"timezone offset"          yes     NO
//	ErrIncompatibleUnits       yes     NO
//
// The two missing rows are what an interval or timing operation went to funcs
// for. The units row was measurably wrong: `Interval[1 'cm', 2 'cm'] overlaps
// Interval[1 's', 2 's']` raised an error and took the define with it, while
// `1 'cm' < 1 's'` — the same question, the other package — answered null. The
// offset row could not be reached through any spelling probed for it, so closing
// it is by construction rather than a repair; it is closed anyway, because the
// reason it was unreachable is that every door a DateTime comes through now
// places it, and a gap that depends on that staying true is a gap.
//
// Three reasons, one predicate. What each caller does with the answer is still
// the caller's: list membership has no null to hold and keeps two values, and a
// sort key retries at a shared precision, so those ask the narrower question
// instead. See the comment on eval.isAmbiguousTemporalComparison.
func UndecidableComparison(err error) bool {
	return AmbiguousTemporalComparison(err) || IncompatibleUnits(err)
}

// AmbiguousTemporalComparison reports the temporal half: two values that cannot
// be ordered because one states a precision or an offset the other does not.
//
// It is asked separately by the callers whose response to it is a *retry* rather
// than null — comparing again at the precision both sides do state. That retry
// means nothing for units, so those callers must not be handed the wider
// question.
//
// It is fhirpath's own IsUnknownTemporalComparison, which reports either of the
// two sentinels it raises: ErrPrecisionMismatch and ErrOffsetMismatch.
//
// Both copies this replaces matched the offset case by text instead, with a note
// to switch once the minimum version carried the helper. It does: go.mod names
// v1.9.1 and the helper has been there since v1.8.something. They also matched
// the text "ambiguous comparison", which nothing produces — not fhirpath v1.9.1
// and not this repository — so it had been matching nothing for as long as it had
// been copied.
//
// Leaving a string match in the function whose whole purpose is to be the one
// implementation of this rule would be the same fragility one layer down.
func AmbiguousTemporalComparison(err error) bool {
	return err != nil && fptypes.IsUnknownTemporalComparison(err)
}

// IncompatibleUnits reports two quantities whose dimensions differ, so there is
// no unit both can be stated in.
//
// fhirpath raises a sentinel for it and says what a caller should do with it —
// "callers translate this sentinel into an empty collection instead of failing
// the whole expression" — which is the same thing CQL calls null.
func IncompatibleUnits(err error) bool {
	return err != nil && errors.Is(err, fptypes.ErrIncompatibleUnits)
}
