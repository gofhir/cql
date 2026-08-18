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

// The definitions above came with the harness. Everything below covers ground
// this engine has since changed, so that what it decided is diffed against the
// reference rather than argued about.

// Backbone elements. The model names them after the type that owns them, and
// both halves of reading one used to cut the name at the wrong dot.
define Hospitalization: Enc.hospitalization
define DischargeDisposition: Enc.hospitalization.dischargeDisposition
define Locations: Enc.location
define InheritedElement: Enc.hospitalization.extension

// Choice elements. The reference translator accepts only the declared element
// and a cast — `Obs.value as FHIR.Quantity` — and refuses the concrete name the
// FHIR JSON uses. This engine accepts `Obs.valueQuantity` as well, which is a
// divergence and is recorded as one in probe.divergences.md rather than probed
// here: the translator will not produce ELM for it, so there is nothing to diff.
define Obs: First([Observation])
define ChoiceElement: Obs.value
define ValueAsQuantity: Obs.value as FHIR.Quantity
define EffectiveAsPeriod: Obs.effective as FHIR.Period

// A primitive whose conversion is declared on its base type: FHIR.id extends
// FHIR.string, which is what converts to System.String.
define IdAsString: 'encounter: ' & Enc.id
define StatusAsString: Enc.status

// Two FHIR operands in one definition, which needs the conversion twice. The
// fold deduplicated them and the engine did not, so this is where an invented
// divergence would show up.
define TwoConversions: Enc.id & Enc.status

// A FHIR.Period reaching operators that work on intervals. overlaps converted
// all along; the set operators did not.
define PeriodIntersect: Enc.period intersect MP
define PeriodUnion: Enc.period union MP
define PeriodOverlaps: Enc.period overlaps MP
define EndOf: end of Enc.period

// Interval boundaries. Equality is decided by Start and End, which normalize an
// open boundary to the point it includes.
define OpenClosedEqual: Interval(1, 5) = Interval[2, 4]
define OpenStart: start of Interval(1, 5)
define OpenEnd: end of Interval(1, 5)
define UnitInterval: point from Interval(0, 2)
define SuccessorOf: successor of 1

// Quantities through the aggregates, which read them as zero until they were
// taught otherwise.
define QuantitySum: Sum({ 1 'mg', 2 'mg' })
define QuantityAvg: Avg({ 1 'mg', 2 'mg' })
define QuantityMedian: Median({ 1 'mg', 2 'mg', 3 'mg' })

// A nested type written as a parameter, which has one canonical spelling.
define function "TakesHospitalization"(h FHIR.Encounter.Hospitalization) returns Boolean:
  h is not null

define NestedParameter: TakesHospitalization(Enc.hospitalization)

// A choice every branch of which is an interval, which is what a helper that
// normalizes Condition.onset returns.
define Cond: First([Condition])
define OnsetChoice: Cond.onset
