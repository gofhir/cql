package types

import (
	fptypes "github.com/gofhir/fhirpath/types"
)

// PairWithQuantity gives a bare number the default unit so that it can be
// compared against a quantity, and returns the pair unchanged when that is not
// what it is looking at.
//
// CQL states the rule in the paragraph that defines addition: "when a quantity
// has no units specified, it is treated as a quantity with the default unit
// ('1')". It is a rule about what a bare number *is*, not about addition, so
// every operator that puts one next to a quantity owes it the same reading.
//
// Arithmetic already had it — `1 '1' + 2` is `3 '1'` and `1 + 1 'cm'` is null —
// while comparison had half of it and the rest had none:
//
//	150.0 >= 100 '1'    true      the Decimal spelling, promoted
//	150   >= 100 '1'    error     the same question, an Integer
//	150.0 =  150 '1'    false     equality, promoted by nobody
//
// The half that existed asked whether the value was a Decimal. A bare number in
// CQL is an Integer, a Long or a Decimal, and the literal `150` is the first of
// those — so the spelling authors write was the one spelling the rule missed.
// The question is asked of fptypes.Numeric here, which is the interface every
// numeric value satisfies and the one ToDecimal reads.
//
// It lives in this package because both eval and funcs need it and both import
// this one, which is where UndecidableComparison ended up for the same reason.
func PairWithQuantity(a, b fptypes.Value) (left, right fptypes.Value) {
	if _, ok := a.(fptypes.Quantity); ok {
		return a, asDimensionless(b)
	}
	if _, ok := b.(fptypes.Quantity); ok {
		return asDimensionless(a), b
	}
	return a, b
}

// asDimensionless is a bare number written as the quantity it stands for, or the
// value untouched when it is not a bare number.
//
// A Quantity is returned as it is: the pair may still be two quantities of
// different dimensions, and deciding that is the comparison's business, not this
// function's.
func asDimensionless(v fptypes.Value) fptypes.Value {
	if v == nil {
		return nil
	}
	if _, isQuantity := v.(fptypes.Quantity); isQuantity {
		return v
	}
	n, ok := v.(fptypes.Numeric)
	if !ok {
		return v
	}
	return fptypes.NewQuantityFromDecimal(n.ToDecimal().Value(), "1")
}
