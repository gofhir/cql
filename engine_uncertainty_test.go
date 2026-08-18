package cql

import (
	"context"
	"strings"
	"testing"
)

// uncertainPreamble defines two uncertainties. `days between DateTime(2014, 1,
// 15) and DateTime(2014, 2)` cannot be pinned down — February is a month, not a
// day — so CQL answers with the interval of what it could be.
const uncertainPreamble = `library T version '1.0'

define U: days between DateTime(2014, 1, 15) and DateTime(2014, 2)
define V: days between DateTime(2014, 3, 1) and DateTime(2014, 4)
define M: months between DateTime(2005) and DateTime(2006, 7)

`

func evalUncertain(t *testing.T, expr string) string {
	t.Helper()
	src := uncertainPreamble + "define A: " + expr + "\n"
	got, err := NewEngine().EvaluateExpression(context.Background(), src, "A", nil, nil)
	if err != nil {
		t.Fatalf("%s: %v", expr, err)
	}
	return valueString(got)
}

// TestEqualityAgainstAnUncertaintyIsKnownWhereItCanBe covers a comparison that
// answered false for anything.
//
// An uncertainty is an interval, and equality read it as an ordinary value: an
// Interval is not an Integer, so `U = 20` came back false. It is not false — it
// might be 20 — and `where duration in days of E.period = 1` silently stopped
// matching on that basis.
//
// Null is not the answer either, which the conformance corpus is what says: it
// fixes `months between DateTime(2005) and DateTime(2006, 7) = 24` at false,
// because 24 lies outside what the duration could be. Knowable where the value
// falls outside, unknown where it falls inside.
func TestEqualityAgainstAnUncertaintyIsKnownWhereItCanBe(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		// Inside: it might be, so there is no answer.
		{`U = 20`, "null"},
		{`U != 20`, "null"},
		{`U = 16`, "null"},

		// Outside: it cannot be, and that is knowable.
		{`U = 100`, "false"},
		{`U != 100`, "true"},

		// The corpus's own case, which a first attempt at this broke.
		{`M = 24`, "false"},

		// The ordered comparisons already worked this way and still do.
		{`M > 5`, "true"},
		{`M > 25`, "false"},
		{`U > 10`, "true"},
	} {
		if got := evalUncertain(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// Comparing intervals with each other, and comparing one against something of
// another kind, are untouched: the first is a question about intervals and the
// second is not a question about a value at all.
func TestOrdinaryIntervalComparisonsAreUnchanged(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`Interval[1, 5] = Interval[1, 5]`, "true"},
		{`Interval[1, 5] = Interval[2, 4]`, "false"},
		{`Interval[1, 5] = 'text'`, "false"},
		{`{1, 2} = Interval[1, 2]`, "false"},
		{`1 = 1`, "true"},
		{`@2020-01-01 = @2020-01-01`, "true"},
	} {
		if got := evalUncertain(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestAggregatesPropagateAnUncertainty covers aggregates that answered 0.
//
// toDecimal reads an interval as zero, so summing durations the engine could not
// pin down produced 0 — the arithmetic of a continuous-variable measure, and a 0
// no caller can tell from a real answer.
//
// Arithmetic already propagated: the corpus fixes `U + U` at Interval[32, 88],
// and the aggregates now agree with the operator rather than with toDecimal.
func TestAggregatesPropagateAnUncertainty(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`U`, "Interval[16, 44]"},

		// The operator, which is what the corpus pins down.
		{`U + U`, "Interval[32, 88]"},

		// And the aggregates, which have to agree with it.
		{`Sum({ U })`, "Interval[16, 44]"},
		{`Sum({ U, U })`, "Interval[32, 88]"},
		{`Avg({ U })`, "Interval[16, 44]"},
		{`Avg({ U, U })`, "Interval[16, 44]"},

		// The middle uncertainty whole: averaging the middle pair on an even
		// count would widen the answer past what the data supports.
		{`Median({ U, V })`, "Interval[30, 60]"},
	} {
		if got := evalUncertain(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// Mixing an uncertainty with a certain value is refused rather than averaged,
// for the same reason a Quantity beside a bare number is: nothing says which of
// the two the caller meant.
func TestAggregatesRefuseMixedCertainty(t *testing.T) {
	for _, expr := range []string{
		`Sum({ U, 5 })`,
		`Avg({ 5, U })`,
	} {
		src := uncertainPreamble + "define A: " + expr + "\n"
		_, err := NewEngine().EvaluateExpression(context.Background(), src, "A", nil, nil)
		if err == nil {
			t.Errorf("%s evaluated, want it refused", expr)
			continue
		}
		if !strings.Contains(err.Error(), "uncertain") {
			t.Errorf("%s: error = %v, want it to mention the uncertainty", expr, err)
		}
	}
}

// The numeric paths are untouched: an uncertainty takes a branch of its own and
// nothing that already worked goes through it.
func TestNumericAggregatesUnaffectedByUncertainty(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`Sum({ 1, 2, 3 })`, "6"},
		{`Avg({ 1.0, 2.0 })`, "1.5"},
		{`Median({ 1, 2, 3 })`, "2"},
		{`Sum({ 1 'mg', 2 'mg' })`, "3 'mg'"},
		{`Avg({ 1 'mg', 2 'mg' })`, "1.5 'mg'"},
	} {
		if got := evalUncertain(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}
