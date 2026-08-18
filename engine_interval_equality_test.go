package cql

import (
	"context"
	"testing"
)

func evalInterval(t *testing.T, expr string) string {
	t.Helper()
	src := "library T version '1.0'\ndefine X: " + expr + "\n"
	got, err := NewEngine().EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err != nil {
		t.Fatalf("%s: %v", expr, err)
	}
	return valueString(got)
}

// TestIntervalEqualityUsesStartAndEnd covers intervals that cover the same
// points and were written differently.
//
// The specification decides this by the boundaries "as determined by the Start
// and End operators", and those operators already normalized: the start of
// Interval(1, 5) is 2 and its end is 4, exactly Interval[2, 4]. Equality
// compared the literal boundaries and their closures instead, so the engine
// answered that the two intervals have the same start, the same end, and are not
// equal.
func TestIntervalEqualityUsesStartAndEnd(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`Interval(1, 5) = Interval[2, 4]`, "true"},
		{`Interval(1, 5) ~ Interval[2, 4]`, "true"},

		// Half-open, which is how a measurement period is often written.
		{`Interval[1, 5) = Interval[1, 4]`, "true"},
		{`Interval(1, 5] = Interval[2, 5]`, "true"},

		// Dates step at their own precision, so the same holds for them.
		{`Interval(@2020-01-01, @2020-01-05) = Interval[@2020-01-02, @2020-01-04]`, "true"},
		{`Interval[@2020-01-01, @2020-01-05) = Interval[@2020-01-01, @2020-01-04]`, "true"},
	} {
		if got := evalInterval(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// Normalizing must not make different intervals equal.
func TestIntervalEqualityStillDistinguishes(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`Interval[1, 4] = Interval[1, 5]`, "false"},
		{`Interval(1, 5) = Interval[2, 5]`, "false"},
		{`Interval(1, 5) = Interval[1, 4]`, "false"},
		{`Interval[@2020-01-01, @2020-01-04] = Interval[@2020-01-01, @2020-01-05]`, "false"},

		// Same points, same answer as before.
		{`Interval[1, 10] = Interval[1, 10]`, "true"},
		{`Interval(1, 10) = Interval(1, 10)`, "true"},
	} {
		if got := evalInterval(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestIntervalEqualityReachesEveryOperationThatUsesIt covers why this was worth
// fixing on the type rather than at the equality operator: seven operations come
// through Interval.Equal, and all seven answered wrongly.
func TestIntervalEqualityReachesEveryOperationThatUsesIt(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`distinct { Interval(1, 5), Interval[2, 4] }`, "{Interval(1, 5)}"},
		{`{ Interval(1, 5) } contains Interval[2, 4]`, "true"},
		{`Interval[2, 4] in { Interval(1, 5) }`, "true"},
		{`{ Interval(1, 5) } intersect { Interval[2, 4] }`, "{Interval(1, 5)}"},
		{`{ Interval(1, 5) } except { Interval[2, 4] }`, "{}"},

		// And a list of genuinely different intervals keeps both.
		{`distinct { Interval[1, 4], Interval[1, 5] }`, "{Interval[1, 4], Interval[1, 5]}"},
	} {
		if got := evalInterval(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// The operators the normalization came from still answer what they did, since
// they now read it off the type instead of computing it themselves.
func TestStartAndEndOfStillNormalize(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`start of Interval(1, 5)`, "2"},
		{`end of Interval(1, 5)`, "4"},
		{`start of Interval[2, 4]`, "2"},
		{`end of Interval[2, 4]`, "4"},
		{`start of Interval(@2020-01-01, @2020-01-05)`, "2020-01-02"},
		{`end of Interval(@2020-01-01, @2020-01-05)`, "2020-01-04"},

		// A type with no successor: the boundary stands for itself.
		{`start of Interval('a', 'z')`, "a"},

		// And successor/predecessor themselves, which now live in one place.
		{`successor of 1`, "2"},
		{`predecessor of 1`, "0"},
		{`successor of @2020-01`, "2020-02"},
		{`predecessor of @2020-01-01`, "2019-12-31"},
	} {
		if got := evalInterval(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}
