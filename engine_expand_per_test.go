package cql

import (
	"context"
	"strings"
	"testing"
	"time"

	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
)

// Plan 026 (docs/plans/026-expand-truncates-to-per.md): expand … per, by
// CQL 1.5.3, Expand (https://cql.hl7.org/N1A/09-b-cqlreference.html#expand).
// The expected values come from the specification's examples, from the answers
// of cql-execution 3.3.2 and the Java engine 5.4.0 where they follow its text
// (scratchpad probes r026, r026b and r026c of the plan), and from the user's
// decisions D1 (a negative decimal is truncated toward zero) and D2 (several
// units of per are counted from the start of the first interval).

// expandOffsets: the answers do not depend on the evaluation offset here, and
// two offsets, one of them fractional, keep it so.
var expandOffsets = []int{minus0300, plus0530}

func evalExpandAt(t *testing.T, seconds int, expr string) fptypes.Value {
	t.Helper()
	got, err := NewEngine(WithEvaluationTimestamp(time.Date(2020, 6, 1, 12, 0, 0, 0, time.FixedZone("", seconds)))).
		EvaluateExpression(context.Background(), "library P\ndefine X: "+expr+"\n", "X", nil, nil)
	if err != nil {
		t.Fatalf("%s at %+ds: %v", expr, seconds, err)
	}
	return got
}

// expectExpandEquals holds `(expr) = (want)` to true. List and interval
// equality compare a temporal's precision as well, so a result at another
// precision than per's is not equal (it is null).
func expectExpandEquals(t *testing.T, expr, want string) {
	t.Helper()
	for _, offset := range expandOffsets {
		eq := "(" + expr + ") = (" + want + ")"
		if got := answerText(evalExpandAt(t, offset, eq)); got != "true" {
			t.Errorf("at %+ds: %s gives %s; want %s (got = %s)", offset, expr,
				answerText(evalExpandAt(t, offset, expr)), want, got)
		}
	}
}

// expectExpandEmpty holds expr to an empty list, not null.
func expectExpandEmpty(t *testing.T, expr string) {
	t.Helper()
	for _, offset := range expandOffsets {
		got := evalExpandAt(t, offset, expr)
		if l, ok := got.(cqltypes.List); !ok || len(l.Values) != 0 {
			t.Errorf("at %+ds: %s = %s; want {}", offset, expr, answerText(got))
		}
	}
}

// TestTheExamplesOfExpand are the examples of the specification.
func TestTheExamplesOfExpand(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"expand { Interval[@2018-01-01, @2018-01-04] } per day",
			"{ Interval[@2018-01-01, @2018-01-01], Interval[@2018-01-02, @2018-01-02], Interval[@2018-01-03, @2018-01-03], Interval[@2018-01-04, @2018-01-04] }"},
		{"expand { Interval[@2018-01-01, @2018-01-04] } per 2 days",
			"{ Interval[@2018-01-01, @2018-01-02], Interval[@2018-01-03, @2018-01-04] }"},
		{"expand { Interval[@T10:00, @T12:30] } per hour",
			"{ Interval[@T10, @T10], Interval[@T11, @T11], Interval[@T12, @T12] }"},
		{"expand { Interval[10.0, 12.5] } per 1",
			"{ Interval[10.0, 10.0], Interval[11.0, 11.0], Interval[12.0, 12.0] }"},
		{"expand { Interval[10, 10] } per 0.1",
			"{ Interval[10.0, 10.0], Interval[10.1, 10.1], Interval[10.2, 10.2], Interval[10.3, 10.3], Interval[10.4, 10.4], " +
				"Interval[10.5, 10.5], Interval[10.6, 10.6], Interval[10.7, 10.7], Interval[10.8, 10.8], Interval[10.9, 10.9] }"},
		{"expand Interval[1, 10]", "{ 1, 2, 3, 4, 5, 6, 7, 8, 9, 10 }"},
		{"expand Interval[1, 10] per 2", "{ 1, 3, 5, 7, 9 }"},
	} {
		expectExpandEquals(t, tt.expr, tt.want)
	}
	expectExpandEmpty(t, "expand { Interval[@T10, @T10] } per minute")
}

// TestABoundMorePreciseThanPerIsTruncated: «the more precise values will be
// truncated to the precision specified by the per quantity». cql-execution
// answers these; Java rounds the low bound up (a reference defect).
func TestABoundMorePreciseThanPerIsTruncated(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"expand { Interval[@2020-01-01T00:30Z, @2020-01-01T02:10Z] } per hour",
			"{ Interval[@2020-01-01T00Z, @2020-01-01T00Z], Interval[@2020-01-01T01Z, @2020-01-01T01Z], Interval[@2020-01-01T02Z, @2020-01-01T02Z] }"},
		{"expand { Interval[@2020-01-01T00:30, @2020-01-01T02:10] } per hour",
			"{ Interval[@2020-01-01T00, @2020-01-01T00], Interval[@2020-01-01T01, @2020-01-01T01], Interval[@2020-01-01T02, @2020-01-01T02] }"},
		{"expand { Interval[@2020-01-01T23:30+05:30, @2020-01-02T01:10+05:30] } per hour",
			"{ Interval[@2020-01-01T23+05:30, @2020-01-01T23+05:30], Interval[@2020-01-02T00+05:30, @2020-01-02T00+05:30], Interval[@2020-01-02T01+05:30, @2020-01-02T01+05:30] }"},
		{"expand { Interval[@2020-01-01T10:00Z, @2020-01-03T09:00Z] } per day",
			"{ Interval[@2020-01-01T, @2020-01-01T], Interval[@2020-01-02T, @2020-01-02T], Interval[@2020-01-03T, @2020-01-03T] }"},
		{"expand { Interval[@2020-02-28T10:00Z, @2020-03-01T05:00Z] } per day",
			"{ Interval[@2020-02-28T, @2020-02-28T], Interval[@2020-02-29T, @2020-02-29T], Interval[@2020-03-01T, @2020-03-01T] }"},
		{"expand { Interval[@2020-01-15, @2020-03-10] } per month",
			"{ Interval[@2020-01, @2020-01], Interval[@2020-02, @2020-02], Interval[@2020-03, @2020-03] }"},
		{"expand { Interval[@2020-06-15, @2021-02-01] } per year",
			"{ Interval[@2020, @2020], Interval[@2021, @2021] }"},
		{"expand { Interval[@2020-01-01T00:30:15.500Z, @2020-01-01T00:30:17.200Z] } per second",
			"{ Interval[@2020-01-01T00:30:15Z, @2020-01-01T00:30:15Z], Interval[@2020-01-01T00:30:16Z, @2020-01-01T00:30:16Z], Interval[@2020-01-01T00:30:17Z, @2020-01-01T00:30:17Z] }"},
		{"expand { Interval[@T10:30, @T12:30] } per hour",
			"{ Interval[@T10, @T10], Interval[@T11, @T11], Interval[@T12, @T12] }"},
		{"expand { Interval[1.5, 4.2] } per 1",
			"{ Interval[1.0, 1.0], Interval[2.0, 2.0], Interval[3.0, 3.0], Interval[4.0, 4.0] }"},
		{"expand { Interval[1.5 'mg', 4.2 'mg'] } per 1 'mg'",
			"{ Interval[1 'mg', 1 'mg'], Interval[2 'mg', 2 'mg'], Interval[3 'mg', 3 'mg'], Interval[4 'mg', 4 'mg'] }"},
		// D1 (a): toward zero, as Truncate («the integer component of the
		// argument») and cql-execution.
		{"expand { Interval[-1.5, 1.2] } per 1",
			"{ Interval[-1.0, -1.0], Interval[0.0, 0.0], Interval[1.0, 1.0] }"},
		{"expand { Interval[-2.9, -1.1] } per 1",
			"{ Interval[-2.0, -2.0], Interval[-1.0, -1.0] }"},
		// The point overload: the start of each interval of the list overload.
		{"expand Interval[1.5, 4.2] per 1", "{ 1.0, 2.0, 3.0, 4.0 }"},
		{"expand Interval[@2020-01-01T00:30Z, @2020-01-01T02:10Z] per hour",
			"{ @2020-01-01T00Z, @2020-01-01T01Z, @2020-01-01T02Z }"},
		{"expand Interval[@T10:30, @T12:30] per hour", "{ @T10, @T11, @T12 }"},
	} {
		expectExpandEquals(t, tt.expr, tt.want)
	}
}

// TestATruncatedDateTimeKeepsItsOwnDigitsAndOffset: truncating does not move a
// value to another frame; one that writes no offset keeps writing none.
func TestATruncatedDateTimeKeepsItsOwnDigitsAndOffset(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"expand Interval[@2020-01-01T23:30+05:30, @2020-01-02T00:10+05:30] per hour",
			"{2020-01-01T23+05:30, 2020-01-02T00+05:30}"},
		{"expand Interval[@2020-01-01T00:30, @2020-01-01T01:10] per hour",
			"{2020-01-01T00, 2020-01-01T01}"},
	} {
		for _, offset := range expandOffsets {
			if got := answerText(evalExpandAt(t, offset, tt.expr)); got != tt.want {
				t.Errorf("at %+ds: %s = %s; want %s", offset, tt.expr, got, tt.want)
			}
		}
	}
}

// TestOnlyWholeIntervalsOfPerAreContributed: «contribute all the intervals of
// size per that start on or after the lower boundary and end on or before the
// upper boundary». Java and cql-execution agree on the list overload.
func TestOnlyWholeIntervalsOfPerAreContributed(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"expand { Interval[1, 3] } per 2", "{ Interval[1, 2] }"},
		{"expand Interval[1, 3] per 2", "{ 1 }"},
		{"expand { Interval[1.0, 3.0] } per 2", "{ Interval[1.0, 2.0] }"},
		{"expand Interval[1.0, 3.0] per 2", "{ 1.0 }"},
		{"expand { Interval[1.5, 4.2] } per 2", "{ Interval[1.0, 2.0], Interval[3.0, 4.0] }"},
		{"expand { Interval[@2020-01-01, @2020-01-03] } per 2 days", "{ Interval[@2020-01-01, @2020-01-02] }"},
		{"expand Interval[@2020-01-01, @2020-01-03] per 2 days", "{ @2020-01-01 }"},
		{"expand { Interval[@T10:30, @T12:30] } per 2 hours", "{ Interval[@T10, @T11] }"},
		{"expand { Interval[@2020-01-15, @2020-03-10] } per 2 months", "{ Interval[@2020-01, @2020-02] }"},
		{"expand { Interval[1, 9] } per 3", "{ Interval[1, 3], Interval[4, 6], Interval[7, 9] }"},
		// A week is seven days, not a precision.
		{"expand { Interval[@2020-01-01, @2020-01-07] } per 1 week", "{ Interval[@2020-01-01, @2020-01-07] }"},
		{"expand { Interval[@2020-01-01, @2020-01-21] } per week",
			"{ Interval[@2020-01-01, @2020-01-07], Interval[@2020-01-08, @2020-01-14], Interval[@2020-01-15, @2020-01-21] }"},
	} {
		expectExpandEquals(t, tt.expr, tt.want)
	}
	expectExpandEmpty(t, "expand { Interval[@2020-01-01, @2020-01-01] } per week")
}

// TestPerIsInferredFromTheWholeList: «a per value will be constructed based on
// the coarsest precision of the boundaries of the intervals in the input set».
// Java answers these; a Date is not always read as a day.
func TestPerIsInferredFromTheWholeList(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"expand { Interval[@2020-01, @2020-03] }",
			"{ Interval[@2020-01, @2020-01], Interval[@2020-02, @2020-02], Interval[@2020-03, @2020-03] }"},
		{"expand { Interval[@2020-01, @2020-03] } per null",
			"{ Interval[@2020-01, @2020-01], Interval[@2020-02, @2020-02], Interval[@2020-03, @2020-03] }"},
		{"expand { Interval[@2020-01-01T00:00, @2020-01-01T00:02], Interval[@2020-01-02T00, @2020-01-02T02] }",
			"{ Interval[@2020-01-01T00, @2020-01-01T00], Interval[@2020-01-02T00, @2020-01-02T00], " +
				"Interval[@2020-01-02T01, @2020-01-02T01], Interval[@2020-01-02T02, @2020-01-02T02] }"},
		{"expand { Interval[@2020-01-01T00, @2020-01-01T02:30] }",
			"{ Interval[@2020-01-01T00, @2020-01-01T00], Interval[@2020-01-01T01, @2020-01-01T01], Interval[@2020-01-01T02, @2020-01-01T02] }"},
	} {
		expectExpandEquals(t, tt.expr, tt.want)
	}
}

// TestABoundLessPreciseThanPerContributesNothing: «the interval will not
// contribute any results to the output». Java and cql-execution give {}.
func TestABoundLessPreciseThanPerContributesNothing(t *testing.T) {
	for _, expr := range []string{
		"expand { Interval[@2020-01, @2020-03] } per day",
		"expand { Interval[@2020-01-01T00Z, @2020-01-01T02Z] } per minute",
		"expand { Interval[@2020, @2021] } per month",
	} {
		expectExpandEmpty(t, expr)
	}
}

// TestAnOpenBoundIsItsSuccessorThenTruncated: start of and end of first, at the
// bound's own precision, then truncated. Java and cql-execution give 11 and 12
// for the first; cql-execution 10 and 11 for the second.
func TestAnOpenBoundIsItsSuccessorThenTruncated(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"expand { Interval(@T10:59, @T12:00] } per hour", "{ Interval[@T11, @T11], Interval[@T12, @T12] }"},
		{"expand { Interval[@T10:30, @T12:00) } per hour", "{ Interval[@T10, @T10], Interval[@T11, @T11] }"},
		{"expand { Interval(@T10, @T12] } per hour", "{ Interval[@T11, @T11], Interval[@T12, @T12] }"},
		{"expand { Interval(@2020-01-01T00:30Z, @2020-01-01T02:00Z) } per hour",
			"{ Interval[@2020-01-01T00Z, @2020-01-01T00Z], Interval[@2020-01-01T01Z, @2020-01-01T01Z] }"},
		{"expand { Interval(1.5, 3.0) } per 1", "{ Interval[1.0, 1.0], Interval[2.0, 2.0] }"},
		{"expand { Interval(1.5, 4.2) } per 1", "{ Interval[1.0, 1.0], Interval[2.0, 2.0], Interval[3.0, 3.0], Interval[4.0, 4.0] }"},
		{"expand { Interval(1, 3) } per 1", "{ Interval[2, 2] }"},
	} {
		expectExpandEquals(t, tt.expr, tt.want)
	}
	expectExpandEmpty(t, "expand { Interval[1, null) } per 1")
}

// TestTheCalendarIsWalkedFromTheStart: each start is the first start plus k
// times per, so a month end does not shrink the months after it, and the walk
// ends at the end of the calendar or of the day.
func TestTheCalendarIsWalkedFromTheStart(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"expand { Interval[@2020-01-31, @2020-05-31] } per month",
			"{ Interval[@2020-01, @2020-01], Interval[@2020-02, @2020-02], Interval[@2020-03, @2020-03], Interval[@2020-04, @2020-04], Interval[@2020-05, @2020-05] }"},
		{"expand { Interval[@2020-01-30, @2020-03-02] } per month",
			"{ Interval[@2020-01, @2020-01], Interval[@2020-02, @2020-02], Interval[@2020-03, @2020-03] }"},
		{"Count(expand { Interval[@2020-01-31, @2020-03-31] } per day)", "61"},
		{"Count(expand { Interval[@2019-01-31, @2019-03-31] } per day)", "60"},
		{"expand { Interval[@T23, @T23] } per hour", "{ Interval[@T23, @T23] }"},
		{"expand { Interval[@T22:30, @T23:59] } per hour", "{ Interval[@T22, @T22], Interval[@T23, @T23] }"},
		{"expand { Interval[@T23:58, @T23:59:59.999] } per minute", "{ Interval[@T23:58, @T23:58], Interval[@T23:59, @T23:59] }"},
		{"expand { Interval[@9999-12-31, @9999-12-31] } per day", "{ Interval[@9999-12-31, @9999-12-31] }"},
		{"expand Interval[@9999-12-31, @9999-12-31] per day", "{ @9999-12-31 }"},
		{"expand { Interval[@9999-12-30, @9999-12-31] } per 2 days", "{ Interval[@9999-12-30, @9999-12-31] }"},
		{"expand { Interval[@9999-11, @9999-12] } per month", "{ Interval[@9999-11, @9999-11], Interval[@9999-12, @9999-12] }"},
	} {
		expectExpandEquals(t, tt.expr, tt.want)
	}
}

// TestTheInputIsCollapsedOrderedAndAnchoredAtTheFirst: Java and cql-execution
// collapse and order the input (r026c D3, D35); D2 (b) counts several units of
// per from the start of the first interval, «the 'per' partitions start from the
// starting boundary of the first input interval, ordered».
func TestTheInputIsCollapsedOrderedAndAnchoredAtTheFirst(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"expand { Interval[6, 9], Interval[1, 4] } per 1",
			"{ Interval[1, 1], Interval[2, 2], Interval[3, 3], Interval[4, 4], Interval[6, 6], Interval[7, 7], Interval[8, 8], Interval[9, 9] }"},
		{"expand { Interval[1, 4], Interval[2, 5] } per 2", "{ Interval[1, 2], Interval[3, 4] }"},
		{"expand { Interval[2, 9], Interval[1, 1] } per 3", "{ Interval[1, 3], Interval[4, 6], Interval[7, 9] }"},
		{"expand { Interval[1, 2], null, Interval[3, 3] } per 1", "{ Interval[1, 1], Interval[2, 2], Interval[3, 3] }"},
		{"expand { Interval[1, 4], Interval[6, 9] } per 2", "{ Interval[1, 2], Interval[3, 4], Interval[7, 8] }"},
		{"expand { Interval[6, 9], Interval[1, 4] } per 2", "{ Interval[1, 2], Interval[3, 4], Interval[7, 8] }"},
		{"expand { Interval[@2020-01-01, @2020-01-04], Interval[@2020-01-06, @2020-01-09] } per 2 days",
			"{ Interval[@2020-01-01, @2020-01-02], Interval[@2020-01-03, @2020-01-04], Interval[@2020-01-07, @2020-01-08] }"},
		{"expand Interval[1, 4] per 3", "{ 1 }"},
		// Merged once truncated, at the precision of per: to the hour the two do
		// not meet, to the day they do. Java and cql-execution answer this
		// (expansions D652).
		{"expand { Interval[@2020-08-18T00Z, @2020-08-20T00Z], Interval[@2020-08-15T00Z, @2020-08-17T00Z] } per 2 days",
			"{ Interval[@2020-08-15T, @2020-08-16T], Interval[@2020-08-17T, @2020-08-18T], Interval[@2020-08-19T, @2020-08-20T] }"},
		{"expand { Interval[@2025-01T, @2027-01T], Interval[@2028-01T, @2029-01T], Interval[@2023-01T, @2024-01T] } per 3 years",
			"{ Interval[@2023T, @2025T], Interval[@2026T, @2028T] }"},
	} {
		expectExpandEquals(t, tt.expr, tt.want)
	}
	expectExpandEmpty(t, "expand List<Interval<Integer>>{} per 1")
	for _, offset := range expandOffsets {
		if got := evalExpandAt(t, offset, "expand (null as List<Interval<Integer>>) per 1"); got != nil {
			t.Errorf("at %+ds: expand of a null list = %s; want null", offset, answerText(got))
		}
	}
}

// TestExpandStaysWithinItsLimit: a truncated expansion still counts against the
// limit.
func TestExpandStaysWithinItsLimit(t *testing.T) {
	_, err := NewEngine().EvaluateExpression(context.Background(),
		"library P\ndefine X: expand { Interval[@2000-01-01, @2400-01-01] } per day\n", "X", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "limit") {
		t.Errorf("expand of 146 098 days: err = %v; want the expand limit", err)
	}
}

// TestCollapseIsUnchangedByPlan026: collapse shares helpers with expand, and its
// own reading of per is out of the plan's scope.
func TestCollapseIsUnchangedByPlan026(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"collapse { Interval[@2020-01-01T00:30Z, @2020-01-01T01:10Z], Interval[@2020-01-01T01:50Z, @2020-01-01T03:00Z] } per hour",
			"{ Interval[@2020-01-01T00:30Z, @2020-01-01T03:00Z] }"},
		{"collapse { Interval[1.5, 2.2], Interval[2.9, 4] } per 1", "{ Interval[1.5, 4.0] }"},
		{"collapse { Interval[@2020-01-01, @2020-01-05], Interval[@2020-01-06, @2020-01-09] } per day",
			"{ Interval[@2020-01-01, @2020-01-09] }"},
	} {
		expectExpandEquals(t, tt.expr, tt.want)
	}
}

// Plan 026, rule 1, and CQL 1.5.3, Expand: «a per value will be constructed
// based on the coarsest precision of the boundaries of the intervals in the
// input set». A defined boundary counts even when its interval, with a null
// one, contributes nothing (code review 026, round 1).
func TestPerIsInferredFromEveryDefinedBoundary(t *testing.T) {
	expectExpandEquals(t, "expand { Interval[@2020-01, null), Interval[@2020-02-01, @2020-02-02] }",
		"{ Interval[@2020-02, @2020-02] }")
	expectExpandEquals(t, "expand { Interval[null, @2020-01], Interval[@2020-02-01, @2020-02-02] }",
		"{ Interval[@2020-02, @2020-02] }")
}

// CQL 1.5.3, Expand: only «intervals of size per» are contributed. A per too
// large for any whole partition gives none, rather than overflowing to a
// smaller one or failing (code review 026, round 1).
func TestAPerTooLargeForAnyPartitionGivesNone(t *testing.T) {
	for _, expr := range []string{
		"expand { Interval[@T10, @T12] } per 144115188075855872 hours",
		"expand { Interval[@T10:00, @T12:00] } per 153722867280912931 minutes",
		"expand { Interval[@2020-01-01, @2020-01-10] } per 2635249153387078803 weeks",
		"expand { Interval[@2020-01-01, @2020-01-10] } per 9223372036854775807 days",
		"expand { Interval[@2020-01-01T00Z, @2020-01-01T10Z] } per 9223372036854775807 hours",
	} {
		expectExpandEmpty(t, expr)
	}
	for _, expr := range []string{
		"expand Interval[@T10, @T12] per 144115188075855872 hours",
		"expand Interval[@2020-01-01, @2020-01-10] per 2635249153387078803 weeks",
	} {
		expectExpandEmpty(t, expr)
	}
}

// A numeric per too large for the Integer or Decimal it is read in gives no
// whole partition, rather than wrapping to a small one (code review 026, round 2).
func TestANumericPerTooLargeGivesNone(t *testing.T) {
	for _, expr := range []string{
		"expand { Interval[1, 3] } per 18446744073709551617.0",
		"expand { Interval[1, 3] } per 9223372036854775808.0",
		"expand { Interval[1.0, 3.0] } per 18446744073709551617.0",
		"expand { Interval[1 'mg', 3 'mg'] } per 18446744073709551617.0 'mg'",
		"expand Interval[1, 3] per 18446744073709551617.0",
	} {
		expectExpandEmpty(t, expr)
	}
}

// CQL 1.5.3, Expand: the per «may be any valid positive decimal». A per of 2^63
// does not fit int64, yet its partitions fit the Long domain: the whole range
// holds two of them (code review 026, round 3).
func TestALongPerCanFillTheDomain(t *testing.T) {
	expectExpandEquals(t, "expand { Interval[-9223372036854775807L - 1L, 9223372036854775807L] } per 9223372036854775808.0",
		"{ Interval[-9223372036854775807L - 1L, -1L], Interval[0L, 9223372036854775807L] }")
	expectExpandEmpty(t, "expand { Interval[0L, 9223372036854775807L] } per 9223372036854775809.0")
}
