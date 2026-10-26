package types

import fptypes "github.com/gofhir/fhirpath/types"

// ModelAgainstSystem reports whether one of a and b is a value of the model —
// a FHIR element, held as the object it was read from — and the other a System
// value.
//
// Such a pair is two values of different types, and CQL does not equate them:
// the translator compares a FHIR Quantity with a System Quantity only where it
// inserts FHIRHelpers.ToQuantity, and where it casts both lists to a choice, as
// in `{ Q } except { -0.005 'g' }`, it inserts nothing and both reference
// engines keep the two apart. fhirpath answers FHIRPath's equality, which maps
// a FHIR Quantity onto a System.Quantity: right for FHIRPath, and not CQL's.
// Until v1.16.6 it did not, and the operations that hold values relied on it
// without saying so (docs/plans/008-fhirpath-1-16.md).
//
// The rule is about the representation, so it names no FHIR type: a FHIR
// primitive arrives as a System value and is not affected.
func ModelAgainstSystem(a, b fptypes.Value) bool {
	_, aModel := a.(*fptypes.ObjectValue)
	_, bModel := b.(*fptypes.ObjectValue)
	return aModel != bModel
}
