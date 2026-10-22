library Encoder version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
context Patient
define D0: true
define D1: 2147483647
define D2: 9007199254740993L
define D3: 1.50
define D4: 'a "b"'
define D5: null
define D6: 5.5 'mg'
define D7: @2020-01-02T03:04:05.006+02:00
define D8: @2020-01T
define D9: @2020-01-02
define D10: @T10:11
define D11: Interval[1, 5)
define D12: Interval(null, @2020-01-02]
define D13: { 1, null, 3 }
define D14: List<Integer>{}
define D15: Code { code: 'c', system: 's', version: 'v', display: 'd' }
define D16: Code { code: 'c' }
define D17: Concept { codes: { Code { code: 'c', system: 's' } }, display: 'x' }
define D18: Tuple { b: 'x', a: 1, c: null }
define D19: Tuple { l: { Tuple { x: 1 } } }
define D20: FHIR.string { value: 'a' }
define D21: Patient.birthDate
define D22: FHIR.Coding { system: FHIR.uri { value: 's' }, code: FHIR.code { value: 'c' }, extension: { FHIR.Extension { url: FHIR.uri { value: 'http://example.org/e' }, value: FHIR.string { value: 'x' } } } }
define D23: Patient
define D24: { Patient.birthDate }
