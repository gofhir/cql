package types_test

import (
	"testing"

	"github.com/shopspring/decimal"

	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

// TestPairWithQuantityOnlyActsOnTheOnePairItIsFor pins the contract, because the
// function is applied at five places and a wrong reading of it would be five
// wrong readings.
//
// It promotes a bare number *only* when the other side is a quantity. A pair with
// no quantity in it is left alone — promoting there would give `1 = 1` a unit it
// never had, and `Sum({1, 2})` the unit '1', which is the mistake the aggregate
// path names in its own comment.
func TestPairWithQuantityOnlyActsOnTheOnePairItIsFor(t *testing.T) {
	q := fptypes.NewQuantityFromDecimal(decimal.NewFromInt(1), "mg")
	i := fptypes.NewInteger(1)
	s := fptypes.NewString("x")

	isQuantity := func(v fptypes.Value) bool {
		_, ok := v.(fptypes.Quantity)
		return ok
	}

	for _, tt := range []struct {
		what         string
		a, b         fptypes.Value
		wantA, wantB bool // whether each side comes back as a Quantity
	}{
		{"a quantity on the right promotes the left", i, q, true, true},
		{"a quantity on the left promotes the right", q, i, true, true},
		{"two quantities are already what they are", q, q, true, true},
		{"two bare numbers are left alone", i, i, false, false},
		{"a non-numeric is left alone", s, q, false, true},
		{"a nil side is left alone", nil, q, false, true},
	} {
		t.Run(tt.what, func(t *testing.T) {
			gotA, gotB := cqltypes.PairWithQuantity(tt.a, tt.b)
			if isQuantity(gotA) != tt.wantA || isQuantity(gotB) != tt.wantB {
				t.Errorf("PairWithQuantity gave (quantity=%v, quantity=%v), want (%v, %v)",
					isQuantity(gotA), isQuantity(gotB), tt.wantA, tt.wantB)
			}
		})
	}

	// The unit it gives is the dimensionless one, and the value survives it.
	gotA, _ := cqltypes.PairWithQuantity(fptypes.NewInteger(150), q)
	promoted, ok := gotA.(fptypes.Quantity)
	if !ok {
		t.Fatalf("150 against a quantity came back as %T", gotA)
	}
	if promoted.Unit() != "1" {
		t.Errorf("the default unit is %q, want \"1\"", promoted.Unit())
	}
	if !promoted.Value().Equal(decimal.NewFromInt(150)) {
		t.Errorf("promoting 150 gave %s", promoted.String())
	}
}
