package eval

import (
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"
)

// TestAModelValueIsNotEqualToASystemValue holds the equality between elements
// to CQL's: a FHIR value and a System value are values of different types, and
// are not equal. The translator compares them as quantities only where it
// inserts FHIRHelpers.ToQuantity; where it casts both lists to a choice, as in
// `{ Q } except { -0.005 'g' }`, it inserts nothing, and the Java engine and
// cql-execution keep the two apart (testdata/snapshots/crossmodel.cql, D4 to
// D9). fhirpath v1.16.6 answers FHIRPath's equality instead, which maps a FHIR
// Quantity onto a System.Quantity (docs/plans/008-fhirpath-1-16.md).
func TestAModelValueIsNotEqualToASystemValue(t *testing.T) {
	quantity := func(s string) fptypes.Value {
		q, err := fptypes.NewQuantity(s)
		if err != nil {
			t.Fatal(err)
		}
		return q
	}
	object := func(s string) fptypes.Value { return fptypes.NewObjectValue([]byte(s)) }

	for _, pair := range []struct {
		name       string
		model, sys fptypes.Value
	}{
		{"a FHIR Quantity and the quantity it maps to", object(`{"value":-5,"unit":"mg","system":"http://unitsofmeasure.org","code":"mg"}`), quantity("-0.005 'g'")},
		{"in its own unit", object(`{"value":-5,"unit":"mg","system":"http://unitsofmeasure.org","code":"mg"}`), quantity("-5 'mg'")},
		{"with a unit alone", object(`{"value":1,"unit":"g"}`), quantity("1000 'mg'")},
		{"dimensionless, against a number", object(`{"value":1,"unit":"1"}`), fptypes.NewInteger(1)},
		{"with a value alone, against a number", object(`{"value":1}`), fptypes.NewInteger(1)},
	} {
		for _, order := range [][2]fptypes.Value{{pair.model, pair.sys}, {pair.sys, pair.model}} {
			if sameValue(order[0], order[1]) {
				t.Errorf("%s: sameValue(%s, %s) is true", pair.name, order[0], order[1])
			}
			if got := elementEquality(order[0], order[1]); got != elementsUnequal {
				t.Errorf("%s: elementEquality(%s, %s) = %v, want unequal", pair.name, order[0], order[1], got)
			}
		}
	}

	// Two FHIR values still compare as FHIRPath compares them, child by child.
	a, b := object(`{"value":1,"unit":"g"}`), object(`{"unit":"g","value":1}`)
	if !sameValue(a, b) {
		t.Errorf("sameValue(%s, %s) is false", a, b)
	}
}
