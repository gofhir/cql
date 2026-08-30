package cql

import (
	"context"
	"testing"
	"time"
)

// evalMinus5 evaluates with the request offset at UTC-5, where a bare literal and
// a stated one below name the same instant while being specified to different
// precisions — the pair that separates equality from membership.
func evalMinus5(t *testing.T, expr string) string {
	t.Helper()
	got, err := NewEngine(WithEvaluationTimestamp(
		time.Date(2020, 6, 1, 12, 0, 0, 0, time.FixedZone("", -5*3600)))).
		EvaluateExpression(context.Background(),
			"library T version '1.0'\ndefine A: "+expr+"\n", "A", nil, nil)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	if got == nil {
		return "null"
	}
	return got.String()
}

// TestMembershipDoesNotClaimWhatEqualityWontSay covers a contradiction the
// fhirpath team's report led me to check on this side, where it turned out to
// live.
//
// They found equality consulting the default while equivalence, `in` and `|` did
// not. Ours all consult it. What ours disagree about is precision:
//
//	@2020-06-15T23:00:00.0 = @2020-06-16T04:00:00Z      null
//	@2020-06-15T23:00:00.0 in { @2020-06-16T04:00:00Z } true      ← claims more
//
// At UTC-5 the two name one instant, and they are specified to different
// precisions, so equality is unknown — CQL's rule, and the same answer the
// ordering operators give. The collection operators said yes anyway: `distinct`
// folded them into one, `union` and `intersect` treated them as one item,
// IndexOf found one at the other's position, and list equality returned true
// where the elements' own equality returns null.
//
// The cause is where they fall through. types.valuesEqual and eval.sameValue ask
// the temporal comparison first and drop to Value.Equal when it does not decide —
// and Value.Equal compares instants without CQL's precision rule, so it answers
// the question the comparison had just declined.
//
// A collection has no null to return; it either holds an item or it does not. So
// where equality is unknown, membership does not get to claim it: two values that
// are not known to be equal stay two values.
func TestMembershipDoesNotClaimWhatEqualityWontSay(t *testing.T) {
	const (
		bare   = "@2020-06-15T23:00:00.0" // millisecond precision
		stated = "@2020-06-16T04:00:00Z"  // second precision, same instant at UTC-5
	)

	// The premise: equality declines this pair, on the precision rule.
	for _, expr := range []string{bare + " = " + stated, bare + " != " + stated} {
		if got := evalMinus5(t, expr); got != "null" {
			t.Fatalf("%s = %s, want null — this test's premise", expr, got)
		}
	}

	for _, tt := range []struct{ expr, want string }{
		{bare + " in {" + stated + "}", "false"},
		{"{" + stated + "} contains " + bare, "false"},
		{"Count({" + bare + "} union {" + stated + "})", "2"},
		{"Count(distinct {" + bare + ", " + stated + "})", "2"},
		{"Count({" + bare + "} intersect {" + stated + "})", "0"},
		{"IndexOf({" + stated + "}, " + bare + ")", "-1"},
		{"{" + bare + "} = {" + stated + "}", "false"},
	} {
		if got := evalMinus5(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — equality says null for these two", tt.expr, got, tt.want)
		}
	}

	// Equivalence is deliberately not asserted here. It is the one operator
	// entitled to answer where equality will not — it never yields null — but for
	// this pair its answer depends on whether fhirpath's Equivalent consults a
	// supplied default, which changed in v1.9.1. That is a dependency question,
	// not the membership question this test is about.

	// And at the same precision, where equality does decide, everything agrees.
	const same = "@2020-06-15T23:00:00"
	for _, tt := range []struct{ expr, want string }{
		{same + " = " + stated, "true"},
		{same + " in {" + stated + "}", "true"},
		{"Count(distinct {" + same + ", " + stated + "})", "1"},
		{"IndexOf({" + stated + "}, " + same + ")", "0"},
	} {
		if got := evalMinus5(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}
