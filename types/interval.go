// Package types defines CQL-specific types that extend the FHIRPath type system.
package types

import (
	"errors"
	"fmt"

	fptypes "github.com/gofhir/fhirpath/types"
)

// Interval represents a CQL Interval<T> with closed/open boundaries.
// T must be an ordered type (Integer, Decimal, DateTime, Date, Time, Quantity).
type Interval struct {
	Low        fptypes.Value
	High       fptypes.Value
	LowClosed  bool
	HighClosed bool
}

// NewInterval creates a new Interval.
func NewInterval(low, high fptypes.Value, lowClosed, highClosed bool) Interval {
	return Interval{
		Low:        low,
		High:       high,
		LowClosed:  lowClosed,
		HighClosed: highClosed,
	}
}

// Type returns "Interval".
func (i Interval) Type() string {
	return "Interval"
}

// Start is the first point the interval includes.
//
// An open boundary excludes its own value, so the starting point is the
// successor of it: the start of Interval(1, 5) is 2, not 1. A type with no
// successor cannot name that point — String has none — so there the boundary
// stands for itself rather than failing the expression.
//
// A boundary at the limit of its type is different: the successor of the last
// representable DateTime does not exist, and answering with the excluded
// boundary would name a point the interval does not contain. That is an error
// and it travels.
func (i Interval) Start() (fptypes.Value, error) {
	if i.Low == nil || i.LowClosed || !HasSuccessor(i.Low) {
		return i.Low, nil
	}
	return Successor(i.Low)
}

// End is the last point the interval includes, by the same rule.
func (i Interval) End() (fptypes.Value, error) {
	if i.High == nil || i.HighClosed || !HasSuccessor(i.High) {
		return i.High, nil
	}
	return Predecessor(i.High)
}

// Equal reports whether two intervals cover the same points.
//
// The specification is explicit that this is decided by the boundaries "as
// determined by the Start and End operators", so it compares those and not the
// values as written: Interval(1, 5) and Interval[2, 4] are the same interval
// over integers, and comparing the literal boundaries and their closures made
// them unequal. That answer reached seven operations, since `=`, `~`, distinct,
// contains, in, and list intersect and except all come through here.
func (i Interval) Equal(other fptypes.Value) bool {
	// An uncertainty is written as a closed interval, so one with these bounds is
	// this interval. See Uncertainty.
	if u, isUncertainty := other.(Uncertainty); isUncertainty {
		return u.Equal(i)
	}
	o, ok := other.(Interval)
	if !ok {
		return false
	}
	return samePoint(i.Low, i.LowClosed, o.Low, o.LowClosed, Successor, valuesEqual) &&
		samePoint(i.High, i.HighClosed, o.High, o.HighClosed, Predecessor, valuesEqual)
}

// Equivalent is Equal with equivalence for the points, which differs from
// equality only in how it treats null.
func (i Interval) Equivalent(other fptypes.Value) bool {
	if u, isUncertainty := other.(Uncertainty); isUncertainty {
		return u.Equivalent(i)
	}
	o, ok := other.(Interval)
	if !ok {
		return false
	}
	return samePoint(i.Low, i.LowClosed, o.Low, o.LowClosed, Successor, valuesEquivalent) &&
		samePoint(i.High, i.HighClosed, o.High, o.HighClosed, Predecessor, valuesEquivalent)
}

// samePoint reports whether two boundaries name the same included point.
//
// Where the two are written the same way there is nothing to normalize and the
// values are compared directly — which is also the common case, and keeps the
// arithmetic off it. Where they differ, the open one is stepped inwards to the
// point it actually includes.
//
// Two boundaries that differ in closure over a type with no successor are not
// the same point and cannot be made so: Interval('a','z') and Interval['a','z']
// differ at both ends, and dropping the closure from the comparison made them
// equal. The same holds when the step fails at the limit of the type, where
// there is no successor to name.
func samePoint(
	a fptypes.Value, aClosed bool,
	b fptypes.Value, bClosed bool,
	step func(fptypes.Value) (fptypes.Value, error),
	eq func(x, y fptypes.Value) bool,
) bool {
	if aClosed == bClosed {
		return eq(a, b)
	}
	pointOf := func(v fptypes.Value, closed bool) (fptypes.Value, bool) {
		if v == nil || closed {
			return v, true
		}
		if !HasSuccessor(v) {
			return nil, false
		}
		stepped, err := step(v)
		if err != nil || stepped == nil {
			return nil, false
		}
		return stepped, true
	}
	aPoint, ok := pointOf(a, aClosed)
	if !ok {
		return false
	}
	bPoint, ok := pointOf(b, bClosed)
	if !ok {
		return false
	}
	return eq(aPoint, bPoint)
}

// String returns a text representation.
func (i Interval) String() string {
	open := "["
	if !i.LowClosed {
		open = "("
	}
	cl := "]"
	if !i.HighClosed {
		cl = ")"
	}
	low := "null"
	if i.Low != nil {
		low = i.Low.String()
	}
	high := "null"
	if i.High != nil {
		high = i.High.String()
	}
	return fmt.Sprintf("Interval%s%s, %s%s", open, low, high, cl)
}

// IsEmpty returns false for Interval.
func (i Interval) IsEmpty() bool {
	return false
}

// Contains checks if a point value is within the interval.
func (i Interval) Contains(point fptypes.Value) (bool, error) {
	if point == nil {
		return false, nil
	}
	comp, ok := point.(fptypes.Comparable)
	if !ok {
		return false, fmt.Errorf("cannot compare %s", point.Type())
	}
	if i.Low != nil {
		cmp, err := CompareTemporal(comp, i.Low)
		if err != nil {
			return false, err
		}
		if i.LowClosed && cmp < 0 {
			return false, nil
		}
		if !i.LowClosed && cmp <= 0 {
			return false, nil
		}
	}
	if i.High != nil {
		cmp, err := CompareTemporal(comp, i.High)
		if err != nil {
			return false, err
		}
		if i.HighClosed && cmp > 0 {
			return false, nil
		}
		if !i.HighClosed && cmp >= 0 {
			return false, nil
		}
	}
	return true, nil
}

// Includes checks if this interval includes another interval entirely.
func (i Interval) Includes(other Interval) (bool, error) {
	lowOk, err := i.containsBound(other.Low, other.LowClosed, true)
	if err != nil {
		return false, err
	}
	highOk, err := i.containsBound(other.High, other.HighClosed, false)
	if err != nil {
		return false, err
	}
	return lowOk && highOk, nil
}

// Overlaps checks if two intervals share any points.
func (i Interval) Overlaps(other Interval) (bool, error) {
	if i.Low != nil && other.High != nil {
		cmp, err := compareValues(i.Low, other.High)
		if err != nil {
			return false, err
		}
		if cmp > 0 || (cmp == 0 && (!i.LowClosed || !other.HighClosed)) {
			return false, nil
		}
		// For integer types, check effective boundaries:
		// i.Low (open) means effective = i.Low + 1, other.High (open) means effective = other.High - 1
		if cmp == -1 {
			if _, isInt := i.Low.(fptypes.Integer); isInt {
				iLowEff := effectiveIntLow(i.Low, i.LowClosed)
				oHighEff := effectiveIntHigh(other.High, other.HighClosed)
				if iLowEff > oHighEff {
					return false, nil
				}
			}
		}
	}
	if i.High != nil && other.Low != nil {
		cmp, err := compareValues(i.High, other.Low)
		if err != nil {
			return false, err
		}
		if cmp < 0 || (cmp == 0 && (!i.HighClosed || !other.LowClosed)) {
			return false, nil
		}
		// For integer types, check effective boundaries
		if cmp == 1 {
			if _, isInt := i.High.(fptypes.Integer); isInt {
				iHighEff := effectiveIntHigh(i.High, i.HighClosed)
				oLowEff := effectiveIntLow(other.Low, other.LowClosed)
				if iHighEff < oLowEff {
					return false, nil
				}
			}
		}
	}
	return true, nil
}

func effectiveIntLow(v fptypes.Value, closed bool) int64 {
	iv, ok := v.(fptypes.Integer)
	if !ok {
		return 0
	}
	if closed {
		return iv.Value()
	}
	return iv.Value() + 1
}

func effectiveIntHigh(v fptypes.Value, closed bool) int64 {
	iv, ok := v.(fptypes.Integer)
	if !ok {
		return 0
	}
	if closed {
		return iv.Value()
	}
	return iv.Value() - 1
}

// containsBound checks if a boundary point is within this interval.
func (i Interval) containsBound(val fptypes.Value, closed, isLow bool) (bool, error) {
	if val == nil {
		if isLow {
			return i.Low == nil, nil
		}
		return i.High == nil, nil
	}
	comp, ok := val.(fptypes.Comparable)
	if !ok {
		return false, fmt.Errorf("cannot compare %s", val.Type())
	}
	if isLow && i.Low != nil {
		cmp, err := CompareTemporal(comp, i.Low)
		if err != nil {
			return false, err
		}
		if cmp < 0 {
			return false, nil
		}
		if cmp == 0 && closed && !i.LowClosed {
			return false, nil
		}
	}
	if !isLow && i.High != nil {
		cmp, err := CompareTemporal(comp, i.High)
		if err != nil {
			return false, err
		}
		if cmp > 0 {
			return false, nil
		}
		if cmp == 0 && closed && !i.HighClosed {
			return false, nil
		}
	}
	return true, nil
}

// helpers

// valuesEqual is where a list, a tuple, a ratio and an interval boundary all
// decide whether two of their elements are the same, which is why the temporal
// rule belongs here rather than at each of them.
//
// Value.Equal compares types, so a Date and a DateTime naming the same day were
// unequal — and fixing that for the `=` operator alone left every composite
// holding one contradicting the operator, in the same shape the fix was argued
// from:
//
//	start of A = start of B   true
//	end of A   = end of B     true
//	A = B                     false
//
// Also list equality, `in`, `distinct`, `IndexOf` and tuples, all of which read
// their elements through here.
func valuesEqual(a, b fptypes.Value) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if equal, decided := temporalValuesEqual(a, b); decided {
		return equal
	}
	return a.Equal(b)
}

func valuesEquivalent(a, b fptypes.Value) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	// Only the settled case is taken from the comparison. Equivalence answers
	// where equality will not — "equivalence never yields null in CQL, whatever
	// the precisions" — so a pair the comparison declines is handed to Equivalent
	// rather than called unequal. Routing it through the same rule as valuesEqual
	// made `@2012-03-10T10:20:00` stop being equivalent to itself once one side
	// carried a default, and cost 15 conformance cases.
	if equal, decided := temporalValuesEqual(a, b); decided && equal {
		return true
	}
	return a.Equivalent(b)
}

// TemporalValuesEqual is temporalValuesEqual, exported so that the equality
// paths in eval — list membership, distinct, IndexOf, tuple equality — reach the
// same decision. They hold the same kinds of value and must not disagree with a
// list about whether two of them are the same.
//
// It reports a pair that cannot be settled as not equal, which is the answer for
// somewhere that has no null to hold. Callers that do have one — the `=` operator
// — ask TemporalEquality instead and keep the third case.
func TemporalValuesEqual(a, b fptypes.Value) (equal, decided bool) {
	switch TemporalEquality(a, b) {
	case TemporallyEqual:
		return true, true
	case TemporallyUnequal, TemporallyUnknown:
		return false, true
	default:
		return false, false
	}
}

// TemporalVerdict is what comparing two temporal values settles, kept apart from
// what any one operator does about it.
type TemporalVerdict int

const (
	// NotTemporal means the pair is not two temporal values and this rule has
	// nothing to say about it.
	NotTemporal TemporalVerdict = iota
	// TemporallyEqual and TemporallyUnequal are settled answers.
	TemporallyEqual
	TemporallyUnequal
	// TemporallyUnknown is the pair CQL cannot settle: specified to different
	// precisions, agreeing on everything they share. `=` answers null here and
	// membership answers "not the same value" — the same verdict, two policies,
	// which is why the verdict is reported rather than decided here.
	TemporallyUnknown
)

// TemporalEquality answers equality for two temporal values without choosing what
// an operator should do with an unsettled pair.
//
// Collapsing "unknown" into "not equal" for everyone made a container claim what
// its own elements decline: `@2020-01-01T10:00:00Z = @2020-01-01T10:00:00.0Z` is
// null, and an interval with those bounds answered false while `start of` and
// `end of` both answered null — the very contradiction interval equality was
// fixed to remove in v1.15.2, one level up.
func TemporalEquality(a, b fptypes.Value) TemporalVerdict {
	if a == nil || b == nil {
		return NotTemporal
	}
	if !fptypes.IsTemporal(a) || !fptypes.IsTemporal(b) {
		return NotTemporal
	}
	cmp, err := CompareTemporal(a, b)
	if errors.Is(err, fptypes.ErrPrecisionMismatch) {
		return TemporallyUnknown
	}
	if err != nil {
		// Any other reason the comparison could not be made — an offset one side
		// was told to assume and the other was not, which is what a literal
		// compared against a value from a data provider looks like. Answering
		// anything there would make a literal stop matching the data it
		// describes, and cost 15 conformance cases when a first attempt did.
		return NotTemporal
	}
	if cmp == 0 {
		return TemporallyEqual
	}
	return TemporallyUnequal
}

// temporalValuesEqual answers equality for two temporals for the places inside
// types that hold values — a list, a tuple, a ratio, an interval boundary. None
// of them has a null to return: a list either holds a value or it does not. So a
// pair that cannot be settled is two values rather than one.
//
//	@2020-06-15T23:00:00.0 = @2020-06-16T04:00:00Z       null
//	@2020-06-15T23:00:00.0 in { @2020-06-16T04:00:00Z }  true      ← claimed more
//
// Before this, `distinct` folded two such values into one, `union` and `intersect`
// treated them as one item, and IndexOf matched. Two values not known to be equal
// stay two values.
//
// The `=` operator is not one of these places. It can return null and has to, so
// it asks TemporalEquality and keeps the unknown.
func temporalValuesEqual(a, b fptypes.Value) (equal, decided bool) {
	return TemporalValuesEqual(a, b)
}

func compareValues(a, b fptypes.Value) (int, error) {
	return CompareTemporal(a, b)
}
