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
define ListIntDec: {1, 1.0} // control
// The parts of a list, an if or a case reach the type they have in common.
define ListDecInt: {1.0, 1} // control
define ListIntDecInt: {1, 2.5, 3} // control
define ListQuantityInt: {1 'mg', 2} // control
define IfIntDec: if true then 1 else 2.5 // control
define CaseIntDec: case when true then 1 else 2.5 end // control
define CaseQuantityInt: case when false then 1 'mg' else 2 end // control
define MaxQuantityInt: Max({100 '1', 150}) // control
define ListOfListsIntDec: {{1}, {2.5}} // control
define ListOfIntervalsIntDec: start of First({Interval[1, 2], Interval[1.5, 2.5]}) // control
define IntervalIntDec: start of Interval[1, 2.5] // control
define CoalesceIntDec: Coalesce(1, 2.5) // control
define IntervalIntQuantity: start of Interval[100, 200 '1'] // control
define IfListIntDec: if true then {1} else {2.5} // control
define ToStringPromoted: ToString(if true then 1 else 2.5) // control
// A Date among DateTimes is not promoted: see runtime.divergences.md.
define ListDateDateTime: First({@2020-01-01, @2020-01-01T10:00:00}) is DateTime
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

// Comparison at a precision between values at different offsets. At a
// precision coarser than the hour they are read as written, the day each one
// names; at the hour and finer they are compared as instants. These answered
// as instants at every precision, which put 21:00-03:00 on the 2nd.
define SameDayAcrossOffsets: @2020-01-01T23:00:00Z same day as @2020-01-01T21:00:00-03:00 // control
define SameDayAcrossOffsetsSameInstant: @2020-01-01T23:00:00Z same day as @2020-01-02T01:00:00+02:00 // control
define SameDayUnwrittenOffset: DateTime(2020, 3, 5, 23) same day as @2020-03-05T00:00:00Z // control
define SameMonthAcrossOffsets: @2020-01-31T23:00:00Z same month as @2020-02-01T01:00:00+01:00 // control
define BeforeDayAcrossOffsets: @2020-01-01T23:00:00Z before day of @2020-01-02T01:00:00+02:00 // control
define AfterDayAcrossOffsets: @2020-01-02T01:00:00+02:00 after day of @2020-01-01T23:00:00Z // control
define SameDayOrBeforeAcrossOffsets: @2020-01-02T01:00:00+02:00 same day or before @2020-01-01T23:00:00Z // control
define InDayAcrossOffsets: @2020-01-01T23:00:00Z in day of Interval[@2020-01-02T00:00:00+02:00, @2020-01-02T05:00:00+02:00] // control
define SameHourAcrossOffsets: @2020-01-01T23:00:00Z same hour as @2020-01-01T20:00:00-03:00 // control
define IncludesDayAcrossOffsets: Interval[@2020-01-02T00:00:00+02:00, @2020-01-03T05:00:00+02:00] includes day of Interval[@2020-01-01T23:00:00Z, @2020-01-02T10:00:00Z] // control
define IntervalSameDayAcrossOffsets: Interval[@2020-01-01T23:00:00Z, @2020-01-03T00:00:00Z] same day as Interval[@2020-01-02T01:00:00+02:00, @2020-01-03T00:00:00Z] // control
define InHourAcrossOffsets: @2020-01-01T23:30:00Z in hour of Interval[@2020-01-01T22:00:00-03:00, @2020-01-02T05:00:00Z] // control
define DuringDayAcrossOffsets: Interval[@2020-01-01T23:00:00Z, @2020-01-01T23:30:00Z] during day of Interval[@2020-01-02T00:00:00+02:00, @2020-01-02T05:00:00+02:00] // control

// Interval phrases at a precision compare the intervals' start and end points
// at that precision. meets, overlaps, starts and ends did not read the `day of`
// at all, and before and after compared the raw bounds at full precision.
define PhraseBeforeDay: Interval[@2020-01-01T10:00:00Z, @2020-01-01T12:00:00Z] before day of Interval[@2020-01-01T13:00:00Z, @2020-01-02T00:00:00Z] // control
define PhraseBeforeDayTrue: Interval[@2020-01-01T10:00:00Z, @2020-01-01T12:00:00Z] before day of Interval[@2020-01-02T00:30:00Z, @2020-01-03T00:00:00Z] // control
define PhraseAfterDay: Interval[@2020-01-01T13:00:00Z, @2020-01-02T00:00:00Z] after day of Interval[@2020-01-01T10:00:00Z, @2020-01-01T12:00:00Z] // control
define PhraseMeetsDay: Interval[@2020-01-01T00:00:00Z, @2020-01-01T12:00:00Z] meets day of Interval[@2020-01-02T13:00:00Z, @2020-01-05T00:00:00Z] // control
define PhraseMeetsBeforeDay: Interval[@2020-01-01T00:00:00Z, @2020-01-01T12:00:00Z] meets before day of Interval[@2020-01-02T13:00:00Z, @2020-01-05T00:00:00Z] // control
define PhraseMeetsAfterDay: Interval[@2020-01-02T13:00:00Z, @2020-01-05T00:00:00Z] meets after day of Interval[@2020-01-01T00:00:00Z, @2020-01-01T12:00:00Z] // control
define PhraseMeetsDaySameDay: Interval[@2020-01-01T00:00:00Z, @2020-01-01T12:00:00Z] meets day of Interval[@2020-01-01T13:00:00Z, @2020-01-05T00:00:00Z] // control
define PhraseOverlapsDay: Interval[@2020-01-01T00:00:00Z, @2020-01-01T10:00:00Z] overlaps day of Interval[@2020-01-01T12:00:00Z, @2020-01-05T00:00:00Z] // control
define PhraseOverlapsBeforeDay: Interval[@2020-01-01T00:00:00Z, @2020-01-01T10:00:00Z] overlaps before day of Interval[@2020-01-01T12:00:00Z, @2020-01-05T00:00:00Z] // control
define PhraseOverlapsAfterDay: Interval[@2020-01-01T12:00:00Z, @2020-01-05T00:00:00Z] overlaps after day of Interval[@2020-01-01T00:00:00Z, @2020-01-01T10:00:00Z] // control
define PhraseStartsDay: Interval[@2020-01-01T10:00:00Z, @2020-01-03T00:00:00Z] starts day of Interval[@2020-01-01T08:00:00Z, @2020-01-05T00:00:00Z] // control
define PhraseEndsDay: Interval[@2020-01-03T00:00:00Z, @2020-01-05T10:00:00Z] ends day of Interval[@2020-01-01T08:00:00Z, @2020-01-05T08:00:00Z] // control
define PhraseProperlyIncludesIvDay: Interval[@2020-01-01T10:00:00Z, @2020-01-05T00:00:00Z] properly includes day of Interval[@2020-01-01T12:00:00Z, @2020-01-03T00:00:00Z] // control
define PhraseIncludesDayCtl: Interval[@2020-01-01T10:00:00Z, @2020-01-05T00:00:00Z] includes day of Interval[@2020-01-01T08:00:00Z, @2020-01-03T00:00:00Z] // control
define PhraseIvBeforeDayOfs: Interval[@2020-01-01T10:00:00Z, @2020-01-01T23:00:00Z] before day of Interval[@2020-01-02T01:00:00+02:00, @2020-01-03T00:00:00Z] // control
define PhraseIvStartsDayOfs: Interval[@2020-01-01T23:00:00Z, @2020-01-05T00:00:00Z] starts day of Interval[@2020-01-02T01:00:00+02:00, @2020-01-06T00:00:00Z] // control
define PhraseIvEndsDayOfs: Interval[@2020-01-01T00:00:00Z, @2020-01-01T23:00:00Z] ends day of Interval[@2019-12-01T00:00:00Z, @2020-01-02T01:00:00+02:00] // control
define PhraseIvMeetsDayOfs: Interval[@2019-12-01T00:00:00Z, @2020-01-01T23:00:00Z] meets day of Interval[@2020-01-02T01:00:00+02:00, @2020-02-01T00:00:00Z] // control
define PhraseIvOverlapsDayOfs: Interval[@2019-12-01T00:00:00Z, @2020-01-01T23:00:00Z] overlaps day of Interval[@2020-01-02T01:00:00+02:00, @2020-02-01T00:00:00Z] // control
define PhraseBeforeDayYearOnly: Interval[@2019, @2019] before day of Interval[@2020-01-01T00:00:00Z, @2020-02-01T00:00:00Z] // control
define PhraseBeforeHour: Interval[@2020-01-01T10:00:00Z, @2020-01-01T12:00:00Z] before hour of Interval[@2020-01-01T12:30:00Z, @2020-01-02T00:00:00Z] // control

// A closed null bound is the minimum or maximum value of the point type.
define PhraseBeforeNullLow: Interval[@2020-01-01T00:00:00Z, @2020-01-05T00:00:00Z] before day of Interval[null, @2020-02-01T00:00:00Z] // control
define PhraseBeforeNullHigh: Interval[@2020-01-01T00:00:00Z, null] before day of Interval[@2020-03-01T00:00:00Z, @2020-04-01T00:00:00Z] // control
define PhraseOverlapsNullHigh: Interval[@2020-01-01T00:00:00Z, null] overlaps day of Interval[@2020-03-01T00:00:00Z, @2020-04-01T00:00:00Z] // control
define PhraseMeetsNullHigh: Interval[@2020-01-01T00:00:00Z, null] meets day of Interval[@2020-03-01T00:00:00Z, @2020-04-01T00:00:00Z] // control
// A point without the precision can still be too far to be the next one.
define PhraseMeetsYearFar: Interval[DateTime(2020), DateTime(2020)] meets day of Interval[@2022-01-01T00:00:00Z, @2022-02-01T00:00:00Z] // control
define PhraseMeetsYearBefore: Interval[DateTime(2020), DateTime(2020)] meets day of Interval[@2019-06-01T00:00:00Z, @2019-07-01T00:00:00Z] // control
define PhraseMeetsMonthToYear: Interval[DateTime(2020, 6), DateTime(2020, 6)] meets day of Interval[DateTime(2021), DateTime(2021)] // control
define PhraseMeetsYearToMay: Interval[DateTime(2020), DateTime(2020)] meets day of Interval[@2021-05-01T00:00:00Z, @2021-06-01T00:00:00Z] // control
define PhraseMeetsTimePastMidnight: Interval[@T10:00, @T23:59] meets minute of Interval[@T00, @T01] // control
define PhraseMeetsTimeHourPastMidnight: Interval[@T10, @T23] meets minute of Interval[@T00:00, @T01:00] // control
define PhraseStartsBeforeNullLow: Interval[@2020-01-01T00:00:00Z, @2020-01-05T00:00:00Z] starts before day of Interval[null, @2020-02-01T00:00:00Z] // control
define PhraseEndsAfterNullHigh: Interval[@2020-01-01T00:00:00Z, @2020-01-05T00:00:00Z] ends after day of Interval[@2019-01-01T00:00:00Z, null] // control
define PhraseStartsBeforeNullLowNoPrecision: Interval[@2020-01-01T00:00:00Z, @2020-01-05T00:00:00Z] starts before Interval[null, @2020-02-01T00:00:00Z] // control
define PhraseStartsDuringNullLow: Interval[null, @2020-01-05T00:00:00Z] starts during day of Interval[null, @2020-02-01T00:00:00Z] // control
define PhraseEndsDuringNullHigh: Interval[@2020-01-01T00:00:00Z, null] ends during day of Interval[@2019-01-01T00:00:00Z, null] // control
define PhraseStartsBeforePointNullLow: Interval[null, @2020-02-01T00:00:00Z] starts before @2020-01-01T00:00:00Z // control
define PhraseSameDayOrBefore: Interval[@2020-01-01T00:00:00Z, @2020-01-01T10:00:00Z] same day or before Interval[@2020-01-01T12:00:00Z, @2020-01-05T00:00:00Z] // control

// Controls for the encodings: a Decimal past what binary floating point holds
// exactly, and a Time, which one runtime represents as a DateTime with no date.
define DecimalDivision: 10.0 / 3.0 // control
define DecimalSum: 0.1 + 0.2 // control
// A Decimal result is brought to the scale of 8 toward negative infinity, as
// the Java engine does it, where it does it.
define DecimalDivisionNegative: -2.0 / 3.0 // control
define DecimalDivisionEquals: 10.0 / 3.0 = 3.33333333 // control
define IntegerDivisionScale: 10 / 3 // control
define DecimalProductScale: 0.12345678 * 0.5 // control
define DecimalAvgScale: Avg({1.0, 2.0, 2.0}) // control
define DecimalAvgNegativeScale: Avg({-1.0, -2.0, -2.0}) // control
define DecimalProductAggregateScale: Product({0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5}) // control
define DecimalLogScale: Log(10.0, 3.0) // control
define DecimalPowerScale: 2.0 ^ 0.5 // control
define DecimalVarianceScale: Variance({1.0, 2.0, 4.0}) // control
define DecimalStdDevScale: StdDev({1.0, 2.0, 4.0}) // control
define DecimalPopulationVarianceScale: PopulationVariance({1.0, 2.0, 4.0}) // control
define DecimalAbsOfFloored: Abs(-1.0 / 3.0) // control
define DecimalMedianEvenScale: Median({1.00000001, 1.00000002}) // control
define DecimalDivisionNoDoubleRounding: 0.00000002 / 2.00000001 // control
define DecimalExpUnscaled: Exp(1.0) > 2.71828182 // control
define DecimalSubtractUnscaled: (Exp(1.0) - 0.0) > 2.71828182 // control
define DecimalMultiplyScaled: (Exp(1.0) * 1.0) > 2.71828182 // control
define DecimalModScaled: Exp(1.0) mod 1.0 // control
define IntegerNegativePower: 2 ^ -1 // control
// Sum and Product start from the first value; each further step is scaled.
define DecimalSumOfOne: Sum({Exp(1.0)}) > 2.71828182 // control
define DecimalProductOfOne: Product({Exp(1.0)}) > 2.71828182 // control
define DecimalAvgOfOne: Avg({Exp(1.0)}) > 2.71828182 // control
define DecimalSumOfTwo: Sum({Exp(1.0), 0.0}) > 2.71828182 // control
define DecimalLogNearOne: Log(10, 1.00001) // control
// A string read as a Decimal is brought to the scale, toward negative infinity.
define ToDecimalScaled: ToDecimal('1.000000005') // control
define ToDecimalNegativeScaled: ToDecimal('-1.000000005') // control
define LogOfAScaledString: Log(10, ToDecimal('0.999999999999999999999')) // control
define ToDecimalScaleKept: ToString(ToDecimal('1.000000005')) // control
define ToDecimalPrecision: Precision(ToDecimal('1.000000005')) // control
define ToDecimalTrailingZeros: ToString(ToDecimal('2.50000000000')) // control
define ToDecimalPastTheLargest: ToDecimal('99999999999999999999.999999999') // control
define ToDecimalPastTheSmallest: ToDecimal('-99999999999999999999.999999999') // control
define ToDecimalLeadingZero: ToString(ToDecimal('01.5')) // control
define ToDecimalKeepsItsZeros: ToString(ToDecimal('2.50')) // control
define ToDecimalOutOfRange: ToDecimal('100000000000000000000') // control
define ConvertsToDecimalOutOfRange: ConvertsToDecimal('100000000000000000000')
// Where this engine does not scale, on purpose: see runtime.divergences.md.
define DecimalLnUnscaled: Ln(2.0)
define DecimalGeometricMeanUnscaled: GeometricMean({2.0, 8.0, 3.0})
define QuantityDivisionUnscaled: 10.0 'mg' / 3.0
define DecimalPowerShortestFloat: 0.09 ^ 0.5
define DecimalLogNearInteger: Log(1000.0, 10.0)
define DecimalWholePowerExact: 0.21 ^ 2
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
define TypedListNullsUnion: (null as List<Integer>) union (null as List<Integer>) // control
define TypedIntervalNullsUnion: (null as Interval<Integer>) union (null as Interval<Integer>) // control

// A name applied to a list is read off each entry: nulls dropped, duplicates
// kept, one level flatter. A list stays a list with one entry or none.
define PropertyOfList: ({Tuple{a: 1}, Tuple{a: null as Integer}, Tuple{a: 1}}).a
define PropertyOfListOfLists: ({Tuple{a: {1, 2}}, Tuple{a: {3}}}).a
define PropertyOfOneEntryList: ({Tuple{a: 1}}).a
define PropertyOfEmptyList: (List<Tuple{a Integer}>{}).a
define PropertyOfListAllNull: ({Tuple{a: null as Integer}, Tuple{a: null as Integer}}).a
define PropertyOfListInnerNull: ({Tuple{a: {1, null as Integer}}, Tuple{a: {3}}}).a
define PropertyOfNullList: (null as List<Tuple{a Integer}>).a
// An Integer is 32 bits and a Long 64, and a result past its type is null.
define IntegerAddOverflow: 2147483647 + 1
define IntegerMultiplyOverflow: 65536 * 65536
define IntegerNegateMinimum: -(-2147483648)
define IntegerAbsMinimum: Abs(-2147483648)
define IntegerDivOverflow: -2147483648 div -1
define IntegerSumOverflow: Sum({2147483647, 1})
define IntegerProductOverflow: Product({65536, 65536})
define IntegerPowerOverflow: 2 ^ 31
define IntegerTruncatePastTheType: Truncate(3000000000.5)
define IntegerSumWithinTheType: Sum({2147483647, 1, -1}) // control
define IntegerSuccessorOfMaximum: successor of 2147483647 // control
define LongAddOverflow: 9223372036854775807L + 1L
define LongMultiplyOverflow: 4611686018427387904L * 2L
define LongSuccessorOfMaximum: successor of 9223372036854775807L // control
