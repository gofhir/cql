library RuntimeProbe version '1.0'

// Each definition is run by the reference engine (Evaluate.java) and its value
// recorded in runtime.expected.json. Those marked "control" are ones this engine
// already answers correctly: if they move, the harness stopped measuring what it
// is believed to measure.

// Relationships inside an aggregate.
define WithAggregate: ({1,2,3}) X with ({2}) Y such that X = Y aggregate A starting 0: A + X
define WithoutAggregate: ({1,2,3}) X without ({2}) Y such that X = Y aggregate A starting 0: A + X
define WhereAggregate: ({1,2,3}) X where X = 2 aggregate A starting 0: A + X // control

// return without all.
define ReturnDefault: ({1,1,2}) X return X
define ReturnAll: ({1,1,2}) X return all X // control
define ReturnDistinct: ({1,1,2}) X return distinct X // control

// List operators with duplicates inside one operand.
define UnionDuplicates: {1,1,2} union {2}
define ExceptDuplicates: {1,1,2} except {3}
define IntersectDuplicates: {1,1,2} intersect {1,2} // control

// aggregate distinct: a textual key merges distinct rows and splits equal ones.
define AggDistinctNilNull: ({'nil', null}) X aggregate distinct A starting 0: A + 1
define AggDistinctPairs: from ({'a,b','a'}) X, ({'c','b,c'}) Y aggregate distinct A starting 0: A + 1
define AggDistinctIntDec: ({1, 1.0}) X aggregate distinct A starting 0: A + 1
define AggDistinctQuantity: ({1 'm', 100 'cm'}) X aggregate distinct A starting 0: A + 1
define AggDistinctOffsets: ({@2020-01-01T10:00:00Z, @2020-01-01T07:00:00-03:00}) X aggregate distinct A starting 0: A + 1
define ListIntDec: {1, 1.0}
define DistinctQuantity: distinct {1 'm', 100 'cm'} // not a control: see runtime.divergences.md
define DistinctOffsets: distinct {@2020-01-01T10:00:00Z, @2020-01-01T07:00:00-03:00} // control
// distinct is defined by equality: these two show which equality the reference uses.
define QuantityEqual: 1 'm' = 100 'cm'
define QuantityEquivalent: 1 'm' ~ 100 'cm'

// expand past ten thousand elements.
define ExpandWeekPerMinute: Count(expand { Interval[@2024-01-01T00:00:00Z, @2024-01-08T00:00:00Z) } per minute)
define ExpandThirtyYearsPerDay: Count(expand { Interval[@2020-01-01, @2049-12-31] } per day)
define ExpandIntegers: Count(expand { Interval[1, 100000] })
define ExpandSmall: Count(expand { Interval[1, 10] }) // control

// Controls for the comparison itself: precisions below a second, an offset that
// is not UTC, and an order the result must keep.
define DateTimeYear: DateTime(2020) // control
define DateTimeMinute: DateTime(2020, 1, 1, 10, 30) // control
define DateTimeMillis: @2020-01-01T10:30:00.123Z // control
define DateTimeOffset: @2020-01-01T07:00:00-03:00 // control
define SortDescending: ({3, 1, 2}) X sort desc // control

// Comparison at day precision between values at different offsets.
define SameDayAcrossOffsets: @2020-01-01T23:00:00Z same day as @2020-01-01T21:00:00-03:00

// Controls for the encodings: a Decimal past what binary floating point holds
// exactly, and a Time, which one runtime represents as a DateTime with no date.
define DecimalDivision: 10.0 / 3.0 // control
define DecimalSum: 0.1 + 0.2 // control
define TimeLiteral: @T10:30 // control
