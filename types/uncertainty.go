package types

import (
	"fmt"

	fptypes "github.com/gofhir/fhirpath/types"
)

// Uncertainty is a value CQL could not pin down, bounded by what it could be.
//
// `days between DateTime(2014, 1, 15) and DateTime(2014, 2)` is one: February is
// a month rather than a day, so the answer is somewhere between 16 and 44 and CQL
// declines to choose. The specification calls these uncertainties and writes them
// as intervals, which is why the conformance corpus states the expected value as
// `Interval[16, 44]`.
//
// Writing them as intervals is not the same as being intervals, and the engine
// used to make no distinction. That left two questions unanswerable:
//
//   - `U = 20` has no answer, because it might be 20, while
//     `Interval[1, 5] = 3` has one: an interval is not a number. With both
//     represented the same way, equality could only pick one answer for both, and
//     picked false — which is how `where duration in days of E.period = 1` came
//     to silently stop matching.
//   - `Sum({U, U})` should add what each could least and most be, the way the
//     `+` operator does. An aggregate over intervals should not. Reading the
//     bounds through toDecimal answered 0 for both.
//
// So it is a type of its own. It prints as an interval, because that is how the
// specification and the corpus write it, and it compares and aggregates as what
// it is.
type Uncertainty struct {
	Low  fptypes.Value
	High fptypes.Value
}

// NewUncertainty bounds a value that could not be pinned down. A low and high
// that agree are not uncertain, and the caller is expected to have returned the
// value itself instead.
func NewUncertainty(low, high fptypes.Value) Uncertainty {
	return Uncertainty{Low: low, High: high}
}

// Type reports "Interval", which is what CQL calls this shape and what a library
// asking `is Interval` about a duration is entitled to be told.
func (u Uncertainty) Type() string { return "Interval" }

// String writes it the way the specification and the conformance corpus do.
func (u Uncertainty) String() string {
	low, high := "null", "null"
	if u.Low != nil {
		low = u.Low.String()
	}
	if u.High != nil {
		high = u.High.String()
	}
	return fmt.Sprintf("Interval[%s, %s]", low, high)
}

// Equal reports whether two uncertainties cover the same range.
//
// An interval with the same bounds counts as the same thing, because that is how
// the specification writes an uncertainty and how the conformance corpus states
// the expected value: `Interval[16, 44]`. The two are one value spelled two ways.
//
// Comparing an uncertainty against a single value is a different question, and
// the operators answer it: knowable where the value falls outside the range,
// unknown where it falls inside.
func (u Uncertainty) Equal(other fptypes.Value) bool {
	low, high, ok := rangeBounds(other)
	if !ok {
		return false
	}
	return valuesEqual(u.Low, low) && valuesEqual(u.High, high)
}

// Equivalent is Equal, since an uncertainty has no null-vs-missing distinction of
// its own: its bounds carry that.
func (u Uncertainty) Equivalent(other fptypes.Value) bool {
	low, high, ok := rangeBounds(other)
	if !ok {
		return false
	}
	return valuesEquivalent(u.Low, low) && valuesEquivalent(u.High, high)
}

// rangeBounds reads the bounds of whichever of the two spellings a value uses. A
// closed interval only: an uncertainty is what a value could be, and every
// bound it names is one of the possibilities.
func rangeBounds(v fptypes.Value) (low, high fptypes.Value, ok bool) {
	switch r := v.(type) {
	case Uncertainty:
		return r.Low, r.High, true
	case Interval:
		if !r.LowClosed || !r.HighClosed {
			return nil, nil, false
		}
		return r.Low, r.High, true
	}
	return nil, nil, false
}

// IsEmpty reports whether the uncertainty bounds nothing at all.
func (u Uncertainty) IsEmpty() bool { return u.Low == nil && u.High == nil }

// AsInterval renders the uncertainty as the interval the specification writes it
// as, for the operators that treat it as a range of values.
func (u Uncertainty) AsInterval() Interval {
	return NewInterval(u.Low, u.High, true, true)
}
