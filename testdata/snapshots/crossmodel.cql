library Differential version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
context Patient
// Written by hand: the cases of docs/plans/008-fhirpath-1-16.md, section 2. The
// Observation q holds -5 mg; -0.005 'g' is the same quantity as a System value.
// Where the translator inserts FHIRHelpers.ToQuantity the two are compared as
// quantities; where it casts both lists to a choice it inserts nothing, and the
// two are values of different types.
define D0: (First([Observation] O where O.id = 'q').value as FHIR.Quantity) = -0.005 'g'
define D1: -0.005 'g' = (First([Observation] O where O.id = 'q').value as FHIR.Quantity)
define D2: (First([Observation] O where O.id = 'q').value as FHIR.Quantity) ~ -0.005 'g'
define D3: -0.005 'g' ~ (First([Observation] O where O.id = 'q').value as FHIR.Quantity)
define D4: { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) } except { -0.005 'g' }
define D5: { -0.005 'g' } except { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) }
define D6: { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) } intersect { -0.005 'g' }
define D7: { -0.005 'g' } intersect { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) }
define D8: { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) } union { -0.005 'g' }
define D9: { -0.005 'g' } union { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) }
define D10: Count({ (First([Observation] O where O.id = 'q').value as FHIR.Quantity) } union { -0.005 'g' })
define D11: distinct { -0.005 'g', (First([Observation] O where O.id = 'q').value as FHIR.Quantity) }
define D12: distinct { (First([Observation] O where O.id = 'q').value as FHIR.Quantity), -0.005 'g' }
define D13: (First([Observation] O where O.id = 'q').value as FHIR.Quantity) in { -0.005 'g' }
define D14: -0.005 'g' in { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) }
define D15: { -0.005 'g' } contains (First([Observation] O where O.id = 'q').value as FHIR.Quantity)
define D16: { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) } contains -0.005 'g'
define D17: { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) } = { -0.005 'g' }
define D18: { -0.005 'g' } = { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) }
define D19: { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) } ~ { -0.005 'g' }
define D20: { -0.005 'g' } ~ { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) }
define D21: IndexOf({ -0.005 'g' }, (First([Observation] O where O.id = 'q').value as FHIR.Quantity))
define D22: IndexOf({ (First([Observation] O where O.id = 'q').value as FHIR.Quantity) }, -0.005 'g')
define D23: { -0.005 'g' } in { { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) } }
define D24: { { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) } } contains { -0.005 'g' }
define D25: (First([Observation] O where O.id = 'q').value as FHIR.Quantity) in { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) }
define D26: (First([Observation] O where O.id = 'q').value as FHIR.Quantity) in ([Observation] O return O.value as FHIR.Quantity)
define D27: Interval[(First([Observation] O where O.id = 'q').value as FHIR.Quantity), (First([Observation] O where O.id = 'q').value as FHIR.Quantity)] = Interval[-5 'mg', -5 'mg']
define D28: { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) } except { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) }
define D29: distinct { (First([Observation] O where O.id = 'q').value as FHIR.Quantity), (First([Observation] O where O.id = 'q').value as FHIR.Quantity) }
define D30: { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) } includes { -0.005 'g' }
define D31: { -0.005 'g' } included in { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) }
define D32: { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) } except { -5 'mg' }
define D33: { (First([Observation] O where O.id = 'q').value as FHIR.Quantity) } union { -5 'mg' }
define D34: exists ([Observation] O where (O.value as FHIR.Quantity) in { -0.005 'g' })
define D35: { Patient.gender } except { 'male' }
define D36: { First([Observation] O where O.id = 'i').value } except { 7 }
define D37: { (First([Observation] O where O.id = 'q').value as FHIR.Quantity).value } except { -5 }
define D38: { Patient.birthDate } except { @1980-01-01 }
define D39: { Patient.gender } union { 'male' }
