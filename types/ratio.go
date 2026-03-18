package types

import (
	"fmt"

	fptypes "github.com/gofhir/fhirpath/types"
)

// Ratio represents a CQL Ratio — a numerator and denominator, both Quantities.
type Ratio struct {
	Numerator   fptypes.Value
	Denominator fptypes.Value
}

// NewRatio creates a new Ratio.
func NewRatio(numerator, denominator fptypes.Value) Ratio {
	return Ratio{Numerator: numerator, Denominator: denominator}
}

// Type returns "Ratio".
func (r Ratio) Type() string {
	return "Ratio"
}

// Equal checks exact equality: both numerator and denominator must match.
func (r Ratio) Equal(other fptypes.Value) bool {
	o, ok := other.(Ratio)
	if !ok {
		return false
	}
	return valuesEqual(r.Numerator, o.Numerator) && valuesEqual(r.Denominator, o.Denominator)
}

// Equivalent checks equivalence.
func (r Ratio) Equivalent(other fptypes.Value) bool {
	o, ok := other.(Ratio)
	if !ok {
		return false
	}
	return valuesEquivalent(r.Numerator, o.Numerator) && valuesEquivalent(r.Denominator, o.Denominator)
}

// String returns a text representation.
func (r Ratio) String() string {
	n, d := "null", "null"
	if r.Numerator != nil {
		n = r.Numerator.String()
	}
	if r.Denominator != nil {
		d = r.Denominator.String()
	}
	return fmt.Sprintf("%s : %s", n, d)
}

// IsEmpty returns false for Ratio.
func (r Ratio) IsEmpty() bool {
	return false
}
