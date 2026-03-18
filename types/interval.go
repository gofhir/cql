// Package types defines CQL-specific types that extend the FHIRPath type system.
package types

import (
	"fmt"

	fptypes "github.com/gofhir/fhirpath/types"
)

// Interval represents a CQL Interval<T> with closed/open boundaries.
// T must be an ordered type (Integer, Decimal, DateTime, Date, Time, Quantity).
type Interval struct {
	Low       fptypes.Value
	High      fptypes.Value
	LowClosed bool
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

// Equal checks exact equality: same boundaries and same closure.
func (i Interval) Equal(other fptypes.Value) bool {
	o, ok := other.(Interval)
	if !ok {
		return false
	}
	if i.LowClosed != o.LowClosed || i.HighClosed != o.HighClosed {
		return false
	}
	lowEq := valuesEqual(i.Low, o.Low)
	highEq := valuesEqual(i.High, o.High)
	return lowEq && highEq
}

// Equivalent checks equivalence.
func (i Interval) Equivalent(other fptypes.Value) bool {
	o, ok := other.(Interval)
	if !ok {
		return false
	}
	if i.LowClosed != o.LowClosed || i.HighClosed != o.HighClosed {
		return false
	}
	lowEq := valuesEquivalent(i.Low, o.Low)
	highEq := valuesEquivalent(i.High, o.High)
	return lowEq && highEq
}

// String returns a text representation.
func (i Interval) String() string {
	open := "["
	if !i.LowClosed {
		open = "("
	}
	close := "]"
	if !i.HighClosed {
		close = ")"
	}
	low := "null"
	if i.Low != nil {
		low = i.Low.String()
	}
	high := "null"
	if i.High != nil {
		high = i.High.String()
	}
	return fmt.Sprintf("Interval%s%s, %s%s", open, low, high, close)
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
		cmp, err := comp.Compare(i.Low)
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
		cmp, err := comp.Compare(i.High)
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
	}
	if i.High != nil && other.Low != nil {
		cmp, err := compareValues(i.High, other.Low)
		if err != nil {
			return false, err
		}
		if cmp < 0 || (cmp == 0 && (!i.HighClosed || !other.LowClosed)) {
			return false, nil
		}
	}
	return true, nil
}

// containsBound checks if a boundary point is within this interval.
func (i Interval) containsBound(val fptypes.Value, closed bool, isLow bool) (bool, error) {
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
		cmp, err := comp.Compare(i.Low)
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
		cmp, err := comp.Compare(i.High)
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

func valuesEqual(a, b fptypes.Value) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
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
	return a.Equivalent(b)
}

func compareValues(a, b fptypes.Value) (int, error) {
	ca, ok := a.(fptypes.Comparable)
	if !ok {
		return 0, fmt.Errorf("cannot compare %s", a.Type())
	}
	return ca.Compare(b)
}
