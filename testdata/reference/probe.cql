library Probe version '1.0'

using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers

parameter MP Interval<DateTime> default Interval[@2020-01-01, @2020-12-31]

context Patient

define Enc: First([Encounter])
define During: Enc.period during MP
define StartOf: start of Enc.period
define Sum: 1 + 2
define Mixed: 1 + 2.5
