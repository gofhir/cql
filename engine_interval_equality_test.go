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

// TestIntervalEqualityOverATypeWithNoSuccessor covers what normalizing cannot
// reach. String has no successor, so an open boundary over it names no point of
// its own — and dropping the closure from the comparison made two intervals that
// differ at both ends compare equal.
func TestIntervalEqualityOverATypeWithNoSuccessor(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`Interval('a', 'z') = Interval['a', 'z']`, "false"},
		{`Interval('a', 'z') ~ Interval['a', 'z']`, "false"},
		{`{ Interval['a', 'z'] } contains Interval('a', 'z')`, "false"},
		{`distinct { Interval('a', 'z'), Interval['a', 'z'] }`,
			"{Interval(a, z), Interval[a, z]}"},

		// Written the same way, they are the same interval.
		{`Interval('a', 'z') = Interval('a', 'z')`, "true"},
		{`Interval['a', 'z'] = Interval['a', 'z']`, "true"},
	} {
		if got := evalInterval(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestSteppingPastTheLimitIsAnError covers a boundary at the edge of its type.
// There is no successor to the last representable DateTime, and answering with
// the boundary the interval excludes would name a point it does not contain.
func TestSteppingPastTheLimitIsAnError(t *testing.T) {
	for _, expr := range []string{
		`end of Interval[null, @0001-01-01T00:00:00.000)`,
		`start of Interval(@9999-12-31T23:59:59.999, null]`,

		// Date has the same range as DateTime, and used to answer with year 0
		// and year 10000 instead of saying so.
		`predecessor of @0001-01-01`,
		`successor of @9999-12-31`,
	} {
		src := "library T version '1.0'\ndefine X: " + expr + "\n"
		if _, err := NewEngine().EvaluateExpression(
			context.Background(), src, "X", nil, nil); err == nil {
			t.Errorf("%s evaluated, want an error about the limit", expr)
		}
	}
}

// TestPointFromReadsTheIncludedPoints covers the same contradiction one operator
// over: Interval(0, 2) contains only 1, so it is a unit interval, and reading
// the boundaries as written said otherwise while equality said it equals
// Interval[1, 1].
func TestPointFromReadsTheIncludedPoints(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`point from Interval(0, 2)`, "1"},
		{`point from Interval[1, 1]`, "1"},
		{`Interval(0, 2) = Interval[1, 1]`, "true"},
	} {
		if got := evalInterval(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
	// And an interval that is not a unit interval still refuses.
	src := "library T version '1.0'\ndefine X: point from Interval[1, 5]\n"
	if _, err := NewEngine().EvaluateExpression(
		context.Background(), src, "X", nil, nil); err == nil {
		t.Error("point from Interval[1, 5] evaluated, want it refused")
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
