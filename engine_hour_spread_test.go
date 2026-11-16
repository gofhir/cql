package cql

import (
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

// Plan 025 (docs/plans/025-hour-spread-by-offset.md): a DateTime to the hour that
// the offset of the comparison cuts in two is its span of instants, and a
// comparison at a precision answers where it is the same for every result its
// pair can come to.
//
// CQL 1.5.3, Appendix B (https://cql.hl7.org/N1A/09-b-cqlreference.html), Same
// As, Same Or Before, Same Or After, Before and After: "When comparing DateTime
// values with different timezone offsets, implementations should normalize to
// the timezone offset of the evaluation request timestamp, but only when the
// comparison precision is hours, minutes, seconds, or milliseconds"; "if either
// input has no value for the precision, the comparison stops and the result is
// null". In, Includes and Included In: "If precision is specified and the point
// type is a Date, DateTime, or Time type, comparisons used in the operation are
// performed at the specified precision". The specification does not say what a
// DateTime to the hour is once an offset cuts it: `@2020-01-01T00Z` at +05:30 is
// 05:30 to 06:29, and neither the hour 5 nor the 6. The engine reads it as the
// span, as plan 008 reads a pair for `<` and `=`.
//
// Each expected value was worked out from that rule, by the independent Python
// oracle of scripts/fractionaloffset-area.py or by hand, and not by running the
// engine.

const plus1400 = 14 * 3600

// theFourOffsets are the evaluation offsets of every case: -03:00, +05:30,
// +05:45 and +14:00, in the order of the wants.
var theFourOffsets = [4]int{minus0300, plus0530, plus0545, plus1400}

func expectAtTheFourOffsets(t *testing.T, expr string, wants [4]string) {
	t.Helper()
	for i, offset := range theFourOffsets {
		if got := answerText(evalValueAt(t, offset, expr)); got != wants[i] {
			t.Errorf("at %+ds: %s = %s; want %s", offset, expr, got, wants[i])
		}
	}
}

// TestAnHourCutByAnOffsetIsItsSpan: the probes of the plan, at -03:00, +05:30,
// +05:45 and +14:00. Java 5.4.0 and cql-execution 3.3.2 read a pair in the
// offset of the left operand, and answer the hour its first instant is in, so
// they differ from these on the spread cases (a deliberate difference of plan 025).
func TestAnHourCutByAnOffsetIsItsSpan(t *testing.T) {
	for _, tt := range []struct {
		expr  string
		wants [4]string
	}{
		{"@2020-01-01T00Z before minute of @2020-01-01T01:40+01:00",
			[4]string{"null", "null", "null", "null"}},
		{"@2020-01-01T00Z same minute or before @2020-01-01T01:40+01:00",
			[4]string{"null", "null", "null", "null"}},
		{"@2020-01-01T00Z after minute of @2020-01-01T00:10+01:00",
			[4]string{"true", "true", "true", "true"}},
		{"@2020-01-01T00Z after minute of @2020-01-01T00:20Z",
			[4]string{"null", "null", "null", "null"}},
		{"@2020-01-01T00Z after minute of @2020-01-01T05:10+05:30",
			[4]string{"true", "true", "true", "true"}},
		{"@2020-01-01T10+05:30 before minute of @2020-01-01T02:00-03:00",
			[4]string{"null", "null", "null", "null"}},
		{"@2020-01-01T10+05:30 same hour as @2020-01-01T02:00-03:00",
			[4]string{"null", "true", "null", "null"}},
		{"@2020-01-01T10+05:30 same hour as @2020-01-01T01:00-03:00",
			[4]string{"null", "false", "false", "null"}},
		{"@2020-01-01T10+05:30 same hour or after @2020-01-01T01:00-03:00",
			[4]string{"true", "true", "true", "true"}},
		{"@2020-01-01T10+05:30 same hour or before @2020-01-01T02:00-03:00",
			[4]string{"true", "true", "null", "true"}},
		{"@2020-01-01T10+05:30 before hour of @2020-01-01T03:00-03:00",
			[4]string{"true", "true", "null", "true"}},
		{"@2020-01-01T10+05:30 after hour of @2020-01-01T00:59-03:00",
			[4]string{"true", "true", "true", "true"}},
		{"@2020-01-01T10+05:30 in hour of Interval[@2020-01-01T01:00-03:00, @2020-01-01T03:00-03:00]",
			[4]string{"true", "true", "true", "true"}},
		{"@2020-01-01T10+05:30 in hour of Interval[@2020-01-01T02:00-03:00, @2020-01-01T03:00-03:00]",
			[4]string{"null", "true", "true", "null"}},
		{"@2020-01-01T10+05:30 in minute of Interval[@2020-01-01T01:00-03:00, @2020-01-01T03:00-03:00]",
			[4]string{"true", "true", "true", "true"}},
		{"@2020-01-01T10+05:30 in minute of Interval[@2020-01-01T01:40-03:00, @2020-01-01T03:00-03:00]",
			[4]string{"null", "null", "null", "null"}},
		{"Interval[@2020-01-01T10+05:30, @2020-01-01T12+05:30] includes hour of @2020-01-01T02:00-03:00",
			[4]string{"true", "true", "null", "true"}},
		{"Interval[@2020-01-01T10+05:30, @2020-01-01T12+05:30] includes minute of @2020-01-01T02:00-03:00",
			[4]string{"null", "null", "null", "null"}},
		{"Interval[@2020-01-01T10+05:30, @2020-01-01T12+05:30] includes minute of @2020-01-01T02:40-03:00",
			[4]string{"true", "true", "true", "true"}},
		{"@2020-01-01T00Z in minute of Interval[@2020-01-01T01:40+01:00, @2020-01-01T02:50+01:00]",
			[4]string{"null", "null", "null", "null"}},
		{"@2020-01-01T00Z in minute of Interval[@2020-01-01T00:10+01:00, @2020-01-01T02:50+01:00]",
			[4]string{"true", "true", "true", "true"}},
		{"@2020-01-01T00Z in hour of Interval[@2020-01-01T01:40+01:00, @2020-01-01T02:50+01:00]",
			[4]string{"true", "null", "null", "true"}},
		{"@2020-01-01T00Z same hour as @2020-01-01T01+01:00",
			[4]string{"true", "null", "null", "true"}},
		{"@2020-01-01T00Z same hour or before @2020-01-01T01+01:00",
			[4]string{"true", "null", "null", "true"}},
		{"@2020-01-01T00Z same minute as @2020-01-01T01:30+01:00",
			[4]string{"null", "null", "null", "null"}},
		{"Interval[@2020-01-01T00Z, @2020-01-01T01Z] meets hour of Interval[@2020-01-01T02:30+00:30, @2020-01-01T05+00:30]",
			[4]string{"true", "null", "null", "true"}},
		{"Interval[@2019-12-31T23Z, @2020-01-01T00Z] meets before minute of Interval[@2020-01-01T06:50+05:30, @2020-01-01T08:00+05:30]",
			[4]string{"false", "false", "false", "false"}},
		{"@2020-01-01T00Z same day as @2019-12-31T23:00-03:00",
			[4]string{"false", "false", "false", "false"}},
		{"@2020-01-01T00Z after minute of @2020-01-01T00:59Z",
			[4]string{"null", "null", "null", "null"}},
		{"@2020-01-01T10+05:30 same hour or after @2020-01-01T01:00",
			[4]string{"true", "true", "true", "true"}},
		{"@2020-01-01T10+05:30 in hour of Interval[@2020-01-01T01:00, @2020-01-01T03:00]",
			[4]string{"true", "false", "false", "false"}},
		{"@2020-01-01T10+05:30 same hour as @2020-01-01T04Z",
			[4]string{"null", "null", "null", "null"}},
	} {
		expectAtTheFourOffsets(t, tt.expr, tt.wants)
	}
}

// TestAnOpenBoundOfAnIntervalIsItsSuccessorBetweenIntervals: between two
// intervals, Includes compares Start and End, and the start of an open bound is
// its successor, which is in the hour 00 here (plan 025, round 1, finding 2).
// Java answers true. All the values are at Z, so they are read as written.
func TestAnOpenBoundOfAnIntervalIsItsSuccessorBetweenIntervals(t *testing.T) {
	expectAtTheFourOffsets(t,
		"Interval(@2020-01-01T00:10Z, @2020-01-01T02:00Z] includes hour of Interval[@2020-01-01T00:45Z, @2020-01-01T01:00Z]",
		[4]string{"true", "true", "true", "true"})
	// Against a point the comparison stays exclusive: 00:10 is not after itself.
	expectAtTheFourOffsets(t,
		"@2020-01-01T00:10Z in minute of Interval(@2020-01-01T00:10Z, @2020-01-01T02:00Z]",
		[4]string{"false", "false", "false", "false"})
}

// TestAQuantityPhraseAppliesItsPredicateToEveryReading: `@…T10+05:30` is
// 04:30Z to 05:29Z. Its hours are 1 and 2 at -03:00 against the near end, 3, and
// the far end, 1 ({0, 1}: not past it), so the phrase is true at -03:00 and at
// +14:00 (hours 18 and 19 against 20 and 18). At +05:45 it is the hours 10 and
// 11 against the near end, 11: before for one and the same hour for the other,
// which `before` does not allow, so the answer is unknown (plan 025, round 1,
// finding 1).
func TestAQuantityPhraseAppliesItsPredicateToEveryReading(t *testing.T) {
	expectAtTheFourOffsets(t,
		"@2020-01-01T10+05:30 2 hours or less before hour of @2020-01-01T03:00-03:00",
		[4]string{"true", "true", "null", "true"})
}

// TestTheWindowOfAnUnplacedValueGivesASetOfResults: an unplaced DateTime read at
// each end of its window can be in the hour of the other value at one end and
// after it at the other, which for `same hour or after` is true at both. At each
// offset the earlier end, +14:00, has the hour of 20:00Z, the other value's, and
// the later end, -12:00, an hour after it (plan 025, which reopens the round 2 of
// plan 024).
func TestTheWindowOfAnUnplacedValueGivesASetOfResults(t *testing.T) {
	unplacedValue, err := fptypes.NewDateTime("2020-01-01T10:30")
	if err != nil {
		t.Fatal(err)
	}
	for _, offset := range theFourOffsets {
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", offset))
		for _, tt := range []struct{ expr, want string }{
			{"P same hour or after @2019-12-31T20Z", "true"},
			// The set is {0, 1}: not the same hour for all of it, not before at all.
			{"P same hour as @2019-12-31T20Z", "null"},
			{"P before hour of @2019-12-31T20Z", "false"},
			// The ends of the window are before and after 11:00+01:00 (10:00Z), and
			// the digits written, 10 against 11, before: {-1, 1} at the ends, which
			// is {-1, 0, 1} once the instants between them are counted. Without
			// them `same hour as` would be false for both.
			{"P same hour as @2020-01-01T11:00+01:00", "null"},
		} {
			v, err := NewEngine(WithEvaluationTimestamp(at)).EvaluateExpression(t.Context(),
				"library P\nparameter P DateTime\ndefine X: "+tt.expr, "X", nil, map[string]fptypes.Value{"P": unplacedValue})
			if err != nil {
				t.Fatal(err)
			}
			if got := answerText(v); got != tt.want {
				t.Errorf("at %+ds: P = 2020-01-01T10:30: %s = %s; want %s", offset, tt.expr, got, tt.want)
			}
		}
	}
}

// TestAnUnplacedValueInAnIntervalIsUnknownWhereTheWindowDecidesNothing (M3 of
// plan 024, backlog 1c50): 00:50 at an offset from -12:00 to +14:00 may or may
// not fall in the hour of 00Z, so `in hour of` has no answer. `in`, `includes`
// and `during` with a precision resolve the window, as the comparisons do.
func TestAnUnplacedValueInAnIntervalIsUnknownWhereTheWindowDecidesNothing(t *testing.T) {
	unplacedValue, err := fptypes.NewDateTime("2020-01-01T00:50")
	if err != nil {
		t.Fatal(err)
	}
	for _, offset := range theFourOffsets {
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", offset))
		v, err := NewEngine(WithEvaluationTimestamp(at)).EvaluateExpression(t.Context(),
			"library P\nparameter P DateTime\ndefine X: P in hour of Interval[@2020-01-01T00:00Z, @2020-01-01T01:00Z]",
			"X", nil, map[string]fptypes.Value{"P": unplacedValue})
		if err != nil {
			t.Fatal(err)
		}
		if got := answerText(v); got != "null" {
			t.Errorf("at %+ds: P in hour of Interval[00Z, 01Z] = %s; want null", offset, got)
		}
	}
}

// The same window in `includes` between two intervals, which resolves it as `in`
// does. Intervals of one hour at Z against P = 00:50: unknown at each bound.
func TestAnUnplacedValueInAnIntervalIsUnknownBetweenIntervals(t *testing.T) {
	unplacedValue, err := fptypes.NewDateTime("2020-01-01T00:50")
	if err != nil {
		t.Fatal(err)
	}
	for _, offset := range theFourOffsets {
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", offset))
		v, err := NewEngine(WithEvaluationTimestamp(at)).EvaluateExpression(t.Context(),
			"library P\nparameter P DateTime\ndefine X: Interval[@2020-01-01T00:00Z, @2020-01-01T01:00Z] includes hour of Interval[P, P]",
			"X", nil, map[string]fptypes.Value{"P": unplacedValue})
		if err != nil {
			t.Fatal(err)
		}
		if got := answerText(v); got != "null" {
			t.Errorf("at %+ds: Interval[00Z, 01Z] includes hour of Interval[P, P] = %s; want null", offset, got)
		}
	}
}

// TestAPartThatIsOutDecidesWhereTheOtherIsUnknown (plan 025, D2): `includes`
// is the start of the outer interval the same or before the start of the inner
// and the end of the inner the same or before the end of the outer, and the
// first is unknown here — the outer starts at the hour 00 and the inner at
// 00:30, which is a minute it has no value for — where the second is false:
// 02:30 is after 01. A part that is unknown does not make the answer unknown
// where another is false, as the `and` of the phrases does not.
func TestAPartThatIsOutDecidesWhereTheOtherIsUnknown(t *testing.T) {
	expectAtTheFourOffsets(t,
		"Interval[@2020-01-01T00Z, @2020-01-01T01Z] includes minute of Interval[@2020-01-01T00:30Z, @2020-01-01T02:30Z]",
		[4]string{"false", "false", "false", "false"})
	expectAtTheFourOffsets(t,
		"Interval[@2020-01-01T00Z, @2020-01-01T01Z] includes minute of Interval[@2020-01-01T00:30Z, @2020-01-01T00:40Z]",
		[4]string{"null", "null", "null", "null"})
	// The same for a point: 11:00 is before the start, 11:30, so it is outside
	// whatever the end, written to the hour, has for the minute.
	expectAtTheFourOffsets(t,
		"@2020-01-02T11:00:00Z in minute of Interval[@2020-01-02T11:30:00Z, @2020-01-02T11Z]",
		[4]string{"false", "false", "false", "false"})
}

// TestTheRegressionsOfTheThirdRoundOfPlan024StayAsTheyWere: a value that writes
// no offset has the evaluation's, so at -03:00 `@…T10+05:30` is 01:30 to 02:29
// and both phrases are true there (M1 and M2 of the round).
func TestTheRegressionsOfTheThirdRoundOfPlan024StayAsTheyWere(t *testing.T) {
	expectAtTheFourOffsets(t, "@2020-01-01T10+05:30 same hour or after @2020-01-01T01:00",
		[4]string{"true", "true", "true", "true"})
	expectAtTheFourOffsets(t, "@2020-01-01T10+05:30 in hour of Interval[@2020-01-01T01:00, @2020-01-01T03:00]",
		[4]string{"true", "false", "false", "false"})
}

// TestThePointSpellingsOfMembershipAgree: `in`, `includes` and `during` with a
// point ask one question, and against an open bound the point compares
// exclusively at the precision (CQL 1.5.3, Appendix B, In: "greater than the
// starting point and less than the ending point"): 00:45 is not after 00:10 at
// the hour, nor 01:15 before 02:00 at it. cql-execution 3.3.2 and main answer
// false; Java 5.4.0 answers true for the open start (plan 025, code review
// round 1, finding 1).
func TestThePointSpellingsOfMembershipAgree(t *testing.T) {
	for _, expr := range []string{
		"@2020-01-01T00:45Z in hour of Interval(@2020-01-01T00:10Z, @2020-01-01T02:00Z]",
		"Interval(@2020-01-01T00:10Z, @2020-01-01T02:00Z] includes hour of @2020-01-01T00:45Z",
		"@2020-01-01T00:45Z during hour of Interval(@2020-01-01T00:10Z, @2020-01-01T02:00Z]",
		"@2020-01-01T00:45Z included in hour of Interval(@2020-01-01T00:10Z, @2020-01-01T02:00Z]",
		"@2020-01-01T01:15Z in hour of Interval[@2020-01-01T00:10Z, @2020-01-01T01:50Z)",
		"Interval[@2020-01-01T00:10Z, @2020-01-01T01:50Z) includes hour of @2020-01-01T01:15Z",
		"@2020-01-01T01:15Z during hour of Interval[@2020-01-01T00:10Z, @2020-01-01T01:50Z)",
	} {
		expectAtTheFourOffsets(t, expr, [4]string{"false", "false", "false", "false"})
	}
	for _, expr := range []string{
		"@2020-01-01T01:15Z in hour of Interval(@2020-01-01T00:10Z, @2020-01-01T02:00Z]",
		"Interval(@2020-01-01T00:10Z, @2020-01-01T02:00Z] includes hour of @2020-01-01T01:15Z",
		"@2020-01-01T01:15Z during hour of Interval(@2020-01-01T00:10Z, @2020-01-01T02:00Z]",
	} {
		expectAtTheFourOffsets(t, expr, [4]string{"true", "true", "true", "true"})
	}
}

// TestProperlyIncludesIsStrictOnTheEffectiveBounds: an interval properly
// includes another at a precision when it includes it and is larger there, on
// the start and end each has (CQL 1.5.3, Appendix B, Properly Includes, with
// the comparisons at the precision): (2019-12-31, 2020-01-03] starts on the 1st
// at the day, as [2020-01-01, 2020-01-03] does, so neither is larger; the hours
// alone settle that 00Z to 03Z is larger than 01Z to 02Z at the minute (plan
// 025, code review round 2, finding 1).
func TestProperlyIncludesIsStrictOnTheEffectiveBounds(t *testing.T) {
	for expr, want := range map[string]string{
		"Interval[@2020-01-01, @2020-01-03] properly includes day of Interval(@2019-12-31, @2020-01-03]":                              "false",
		"Interval(@2019-12-31, @2020-01-03] properly included in day of Interval[@2020-01-01, @2020-01-03]":                           "false",
		"Interval(@2019-12-31, @2020-01-03] properly includes day of Interval[@2020-01-01, @2020-01-03]":                              "false",
		"Interval[@2020-01-01T00Z, @2020-01-01T03Z] properly includes minute of Interval[@2020-01-01T01Z, @2020-01-01T02Z]":           "true",
		"Interval[@2020-01-01T01Z, @2020-01-01T02Z] properly included in minute of Interval[@2020-01-01T00Z, @2020-01-01T03Z]":        "true",
		"Interval[@2020-01-01T00:00Z, @2020-01-01T03:00Z] properly includes hour of Interval[@2020-01-01T00:30Z, @2020-01-01T03:30Z]": "false",
		"Interval[@2020-01-01T00:00Z, @2020-01-01T03:00Z] properly includes hour of Interval[@2020-01-01T00:30Z, @2020-01-01T02:30Z]": "true",
	} {
		expectAtTheFourOffsets(t, expr, [4]string{want, want, want, want})
	}
}

// TestANullBoundAtAPrecision: a closed null bound is the minimum or the
// maximum of the point type, and an open one is unknown (CQL 1.5.3, Appendix
// B, Start and End; Interval Operators), as Java 5.4.0 and cql-execution 3.3.2
// both answer at -03:00 and +05:30. And `properly` combines the inclusion and
// the size as `and` does: an unknown inclusion next to a size that is not
// larger is false, as both answer (plan 025, code review round 3).
func TestANullBoundAtAPrecision(t *testing.T) {
	for expr, want := range map[string]string{
		"@2020-01-01T10:00Z in hour of Interval(null, @2020-01-03T00:00Z]":                                                              "null",
		"@2020-01-01T10:00Z in hour of Interval[null, @2020-01-03T00:00Z]":                                                              "true",
		"Interval[@2019-12-29T00Z, @2020-01-01T03:00Z] includes minute of Interval[@2020-01-01T00:00Z, null]":                           "false",
		"Interval[@2019-12-29T00Z, @2020-01-01T03:00Z] includes minute of Interval[@2020-01-01T00:00Z, null)":                           "null",
		"Interval(null, @2020-01-03T00:00Z] includes hour of Interval[@2020-01-01T00:00Z, @2020-01-02T00:00Z]":                          "null",
		"Interval[@2019-12-29T00Z, @2020-01-01T03:00Z] properly includes minute of Interval[@2020-01-01T00:00Z, null]":                  "false",
		"@2020-01-02 included in hour of Interval(null, @2020-01-03T00:00Z]":                                                            "null",
		"Interval[@2020-01-01T00Z, @2020-01-01T03:00Z] properly includes hour of Interval[@2020-01-01T05:30+05:30, @2020-01-01T03:00Z]": "false",
	} {
		expectAtTheFourOffsets(t, expr, [4]string{want, want, want, want})
	}
}
