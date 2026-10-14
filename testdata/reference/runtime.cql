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
define ReturnDefault: ({1,1,2}) X return X // control
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

// return is distinct unless it says all, wherever the return clause is.
define ReturnNulls: ({null, null, 1}) X return X // control
define ReturnSorted: ({1, 1, 2}) X return X sort desc // control
define ReturnLet: ({1, 1, 2}) X let Y: X * 10 return Y // control
define CountReturn: Count(({1, 1, 2}) X return X) // control
define ReturnTuple: ({1, 1}) X return Tuple { v: X } // control
define MultiSourceReturn: from ({1, 1}) A, ({2}) B return A + B // control
// Several sources and no return clause: the translator supplies a distinct one.
define MultiSourceNoReturn: from ({1, 1}) A, ({2}) B // control
// One source and no return clause keeps its duplicates. The references
// disagree, and the conformance suite settles it; the two sorted rows are its
// own cases. See runtime.divergences.md.
define NoReturn: ({1, 1, 2}) X
define NoReturnSortedConformance: ({4, 5, 1, 6, 2, 1}) sL sort asc
define NoReturnSortedDescConformance: ({4, 5, 1, 6, 2, 1}) sL sort desc

// aggregate over the rows every filter keeps, with the lets they were kept with.
define AggregateLetWithout: ({1, 2, 3}) X let L: X * 10 without ({2}) Y such that X = Y aggregate A starting 0: A + L // control
define AggregateDistinctLetWith: ({1, 1, 2, 3}) X let L: X * 10 with ({1, 2}) Y such that X = Y aggregate distinct A starting 0: A + L // control
define AggregateDistinctSources: from ({1, 1}) P, ({2, 2}) Q aggregate distinct S starting 0: S + P + Q // control
define AggregateAllSources: from ({1, 1}) P, ({2, 2}) Q aggregate S starting 0: S + P + Q // control

// List set operations: set semantics with null as an empty list, and
// equality, not scale, deciding which elements are the same.
// A null operand: the references disagree, and the specification settles it.
define UnionNullFirst: null union {1, 1, 2}
define UnionNullSecond: {1, 1, 2} union null
define ExceptNullSecond: {1, 1, 2} except null
define ExceptNullFirst: null except {1} // control
define UnionNullElements: {1, null, null} union {null, 2} // control
define ExceptNullElement: {1, null, 2, null} except {null} // control
define IntersectNullElements: {1, 1, 2, null, null} intersect {1, null} // control
define UnionEmptySecond: {1, 1} union {} // control
define UnionDecimalScale: {1.0} union {1.00} // control
define ExceptDecimalScale: {1.0, 2.0} except {1.00} // control
define IntervalExceptNull: Interval[1, 5] except null // control
define TypedListNullsUnion: (null as List<Integer>) union (null as List<Integer>)
define TypedIntervalNullsUnion: (null as Interval<Integer>) union (null as Interval<Integer>) // control
