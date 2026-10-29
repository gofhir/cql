library Differential version '1.0'
// The cql-tests cases this engine fails, at cqframework/cql-tests f724ee4
// (conformance/testdata/SOURCE.tsv): each expression as the suite writes it,
// with the name of its test after it. Their classes are those of
// knownFailures (conformance/known_failures_test.go) and of
// docs/plans/012-suite-cql-tests.md.
define D5: predecessor of 1.0 // PredecessorOf1DPrecision
define D6: predecessor of 1.01 // PredecessorOf101DPrecision
define D7: predecessor of 1.0 'cm' // PredecessorOf1QCMPrecision
define D10: successor of 1.0 // SuccessorOf1DPrecision
define D11: successor of 1.01 // SuccessorOf101DPrecision
define D14: 10.1 'cm' div -3.1 'cm' // TruncatedDivide10d1ByNeg3D1Quantity
define D15: 10.0 'g' div 5.0 'g' // TruncatedDivide10By5DQuantity
define D16: 4.14 'm' div 2.06 'm' // TruncatedDivide414By206DQuantity
define D17: Tuple { Id : null, Name : 'John' } = Tuple { Id : 1, Name : 'James' } // TupleEqDifferentNamesWithOneNullId
define D18: Tuple{ Id : null, Name : 'John' } != Tuple{ Id : 1, Name : 'Joe' } // TupleNotEqDifferingNamesWithOneNullId
define D19: null !~ null // NotEquivalentNullNull
define D20: month from DateTime(2012) // DateTimeComponentFromMonthIsNull
define D21: years between @2012-02-29 and @2014-02-28 // YearsBetweenLeapYearDatesEquals2
define D22: years between @2012-02-29T12:34:56 and @2014-02-28T12:34:56 // YearsBetweenLeapYearDateTimesEquals2
define D23: Interval[1, 10] ends Interval[null, null] // TestMaxIntervalEndsFalse
define D24: 5 in Interval[null, null] // TestIntegerInMaxIntervalTrue
define D25: 5 in Interval(null, null) // TestIntegerInUnknownIntervalNull
define D26: Interval(null, 5] meets after Interval[11, null) // TestMeetsAfterNull
define D27: Interval[null, null] overlaps Interval[1, 10] // TestMaxIntervalOverlapsTrue
define D28: Interval[null, null] overlaps before Interval[1, 10] // TestMaxIntervalOverlapsBeforeTrue
define D29: Interval[null, null] overlaps after Interval[1, 10] // TestMaxIntervalOverlapsAfterTrue
define D30: point from Interval[null, null] // TestPointFromMaxIntervalError
define D31: Interval[null, null] starts Interval[1, 10] // TestMaxIntervalStartsFalse
define D32: Interval[null, null] union Interval[1, 10] // TestUnionMaxInterval
define D33: { 'A', 'B', 'C' } ~contains 'a' // EquivalentContainsIsTrue
define D34: { 'B', 'C' } ~contains 'a' // EquivalentContainsIsFalse
define D35: (null as List<String>) ~contains 'a' // EquivalentContainsNullLeftIsFalse
define D36: { 'A', 'B', 'C' } ~contains (null as String) // EquivalentContainsNullIsFalse
define D37: { 'A', 'B', 'C', null } ~contains (null as String) // EquivalentContainsNullIsTrue
define D38: distinct (null as List<Integer>) // DistinctNull
define D39: Flatten(null as List<List<Integer>>) // FlattenNull
define D40: 'a' ~in { 'A', 'B', 'C' } // EquivalentInIsTrue
define D41: 'a' ~in { 'B', 'C' } // EquivalentInIsFalse
define D42: 'a' ~in (null as List<String>) // EquivalentInNullRightIsFalse
define D43: null ~in { 1, 3, 5, null } // NullEquivalentInIsTrue
define D44: Skip({ 1, 2, 3 }, -1) // SkipNegative
define D45: Coalesce(null, 15, null, null, null, null) // CoalesceMoreThanFiveArgs
define D46: MatchesFull('N8000123123', 'N[0-9]{10}') // MatchesFullTrue
define D47: MatchesFull('N8000123123', 'N[0-9]{8}') // MatchesFullFalse
define D48: MatchesFull('ABC', null) // MatchesFullNull
define D49: ToDate('2014-01-01') // ToDate1
define D50: ToDate('2014-01') // ToDateYearMonth
define D51: ToDate('2014') // ToDateYear
define D52: ToDate('2014-01-01T12:05:05.955') // ToDateIgnoresTime
define D53: ToDate(DateTime(2014, 01, 01, 12, 30, 0, 0)) // ToDateFromDateTime
define D54: ToDate('2014/01/01') // ToDateMalformed
define D55: ToRatio('1.0 \'mg\':2.0 \'mg\'') is Ratio // StringToRatio
define D56: ToRatio('1.0 \'mg\';2.0 \'mg\'') // StringMalformedToRatio
