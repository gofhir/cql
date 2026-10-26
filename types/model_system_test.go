package types

import (
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"
)

// TestListsAndTuplesKeepModelAndSystemValuesApart holds the equality of the
// elements of a list and of a tuple to the rule of the operations that hold
// values: a FHIR value is not equal to a System value, which is of another
// type (docs/plans/008-fhirpath-1-16.md). Where the translator wants them
// compared as quantities it inserts the conversion, and the list arrives with
// System values on both sides.
func TestListsAndTuplesKeepModelAndSystemValuesApart(t *testing.T) {
	fhir := fptypes.NewObjectValue([]byte(`{"value":-5,"unit":"mg","system":"http://unitsofmeasure.org","code":"mg"}`))
	sys, err := fptypes.NewQuantity("-0.005 'g'")
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]fptypes.Value{
		{NewList(fptypes.Collection{fhir}), NewList(fptypes.Collection{sys})},
		{NewList(fptypes.Collection{sys}), NewList(fptypes.Collection{fhir})},
		{NewTuple(map[string]fptypes.Value{"x": fhir}), NewTuple(map[string]fptypes.Value{"x": sys})},
		{NewTuple(map[string]fptypes.Value{"x": sys}), NewTuple(map[string]fptypes.Value{"x": fhir})},
	} {
		if pair[0].Equal(pair[1]) {
			t.Errorf("%s = %s is true", pair[0], pair[1])
		}
	}
	if !ModelAgainstSystem(fhir, sys) || !ModelAgainstSystem(sys, fhir) {
		t.Error("ModelAgainstSystem does not see a FHIR value against a System one")
	}
	if ModelAgainstSystem(fhir, fhir) || ModelAgainstSystem(sys, sys) {
		t.Error("ModelAgainstSystem sees two values of one kind as of two")
	}
}
