library Differential version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
context Patient
define D0: Patient.gender included in { 'male', 'female' }
define D1: { 'male', 'female' } includes Patient.gender
define D2: Patient.gender during { 'male', 'female' }
define D3: Patient.gender included in { 'female', null }
define D4: Patient.gender included in (null as List<String>)
define D5: Patient.name.given includes (null as String)
define D6: (null as String) included in Patient.name.given
define D7: Patient.name.given includes (null as List<String>)
define D8: Patient.name.given includes (null as List<Any>)
define D9: (null as List<String>) included in Patient.name.given
define D10: Patient.deceased included in { true }
define D11: { true } includes Patient.deceased
define D12: FHIR.dateTime { value: @2020-01-02T00:00:00Z } during First([Encounter]).period
define D13: First([Encounter]).period includes FHIR.dateTime { value: @2020-01-02T00:00:00Z }
define D14: First([Encounter]).period includes (null as DateTime)
define D15: (null as DateTime) during First([Encounter]).period
define D16: (null as DateTime) included in First([Encounter]).period
define D17: First([Observation]).effective during First([Encounter]).period
define D18: First([Observation]).effective included in First([Encounter]).period
define D19: First([Encounter]).period includes First([Observation]).effective
define D20: First([Observation]).effective includes (null as DateTime)
define D21: First([Observation]).value included in { 5 'mg' }
define D22: { 5 'mg' } includes First([Observation]).value
define D23: First([Encounter]).period included in (null as List<Interval<DateTime>>)
define D24: Patient.birthDate during (null as FHIR.Period)
define D25: { 1.0, 2.0 } includes Patient.multipleBirth
define D26: List<FHIR.string>{ FHIR.string { value: null as String } } includes (null as String)
define D27: List<FHIR.string>{ FHIR.string { value: null as String } } contains (null as String)
define D28: List<Any>{ null } includes (null as Any)
define D29: List<Any>{ 1 } includes (null as List<Any>)
define D30: { 'a', null } includes (null as Choice<List<FHIR.string>, String>)
define D31: (null as System.Quantity) included in (null as List<Choice<FHIR.string, FHIR.Quantity>>)
define D32: First([Observation]).component.value includes (null as System.Quantity)
define D33: (null as FHIR.Quantity) included in (null as List<Choice<System.Quantity, String>>)
define D34: (null as Interval<Date>) included in (null as List<Interval<DateTime>>)
define D35: First([Observation]).value during (null as FHIR.Range)
define D36: First([Observation]).value included in (null as FHIR.Range)
define D37: (null as FHIR.Range) includes First([Observation]).value
