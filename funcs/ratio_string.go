package funcs

import (
	"fmt"

	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
)

// RatioFromString reads s as ToRatio does. CQL 1.5.3, Appendix B, ToRatio:
// "The operator accepts strings using the following format:
// <quantity>:<quantity> where <quantity> is the format used to by the
// ToQuantity operator", and "If the input string is not formatted correctly,
// or cannot be interpreted as a valid Ratio value, the result is null". Each
// side is read by QuantityFromString, so it is whatever ToQuantity accepts.
//
// The error is only QuantityFromString's: that UCUM's definitions did not load.
func RatioFromString(s string) (fptypes.Value, error) {
	left, right, ok := RatioSides(s)
	if !ok {
		return nil, nil
	}
	numerator, err := QuantityFromString(left)
	if err != nil || numerator == nil {
		return nil, err
	}
	denominator, err := QuantityFromString(right)
	if err != nil || denominator == nil {
		return nil, err
	}
	return cqltypes.NewRatio(numerator, denominator), nil
}

// RatioSides splits the text of a Ratio at its separator: the one ':' outside
// single quotes, as a quoted unit may hold one (the UCUM annotation '{a:b}'),
// plan 014, decision C. It is false when there is no such ':' or more than
// one. An unclosed quote needs no check of its own: it leaves an apostrophe on
// a side, which no reader of a Quantity takes outside its unit.
func RatioSides(s string) (left, right string, ok bool) {
	separator, quoted := -1, false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			quoted = !quoted
		case ':':
			if quoted {
				continue
			}
			if separator >= 0 {
				return "", "", false
			}
			separator = i
		}
	}
	if separator < 0 {
		return "", "", false
	}
	return s[:separator], s[separator+1:], true
}

// ToRatio converts a value to a Ratio: a string as RatioFromString reads it,
// and a Ratio is itself. Null is null. Any other type has no overload, which
// the reference translator refuses, and it fails here as ToQuantity does.
func ToRatio(v fptypes.Value) (fptypes.Value, error) {
	switch val := v.(type) {
	case nil:
		return nil, nil
	case cqltypes.Ratio:
		return val, nil
	case fptypes.String:
		return RatioFromString(val.Value())
	}
	return nil, fmt.Errorf("cannot convert %s to Ratio", v.Type())
}
