package cql

import "testing"

// TestDateAndDateTimeEqualityAgreesWithOrdering is a contradiction rather than a
// specification question, which is what makes it answerable without one.
//
// The conformance corpus has no case comparing a Date against a DateTime, and the
// specification's implicit conversion table is not clear enough to settle it from
// the outside. It does not have to be. The engine already answers four questions
// about this pair, and its own answers determine the fifth:
//
//	@2020-03-01 <  @2020-03-01T   false     — not less
//	@2020-03-01 >  @2020-03-01T   false     — not greater
//	@2020-03-01 <= @2020-03-01T   true      — less or equal
//	@2020-03-01 >= @2020-03-01T   true      — greater or equal
//	@2020-03-01 =  @2020-03-01T   false     ← cannot be
//
// Not less, not greater, and both non-strict comparisons true, is equal. Four
// operators say so and the fifth disagreed.
//
// The cause is that the ordered comparisons go through Compare, which reads the
// values, and `=` goes through Equal, which reads the types. fptypes.Compare
// returns 0 for this pair — both are specified to the day and name the same day —
// while Date.Equal(DateTime) is false because the types differ.
//
// This is how v1.15.2 settled interval equality too: `start of A = start of B`
// and `end of A = end of B` were true while `A = B` was false, and measuring the
// contradiction was worth more than reading the specification about it.
func TestDateAndDateTimeEqualityAgreesWithOrdering(t *testing.T) {
	pairs := [][2]string{
		{"@2020-03-01", "@2020-03-01T"},
		{"@2020-03", "@2020-03T"},
		{"@2020", "@2020T"},
		{"Date(2020, 3, 1)", "DateTime(2020, 3, 1)"},
	}

	for _, p := range pairs {
		// The premise: the ordered comparisons place this pair as equal.
		for _, tt := range []struct{ op, want string }{
			{"<", "false"}, {">", "false"}, {"<=", "true"}, {">=", "true"},
		} {
			expr := p[0] + " " + tt.op + " " + p[1]
			got, err := evalCQL(t, expr)
			if err != nil {
				t.Fatalf("%s: %v", expr, err)
			}
			if got == nil || got.String() != tt.want {
				t.Fatalf("%s = %v, want %s — this test's premise is that ordering "+
					"already places the pair as equal", expr, got, tt.want)
			}
		}

		// So equality has to agree, in both directions and both spellings.
		for _, tt := range []struct{ expr, want string }{
			{p[0] + " = " + p[1], "true"},
			{p[1] + " = " + p[0], "true"},
			{p[0] + " != " + p[1], "false"},
			{p[0] + " ~ " + p[1], "true"},
		} {
			got, err := evalCQL(t, tt.expr)
			if err != nil {
				t.Errorf("%s: %v", tt.expr, err)
				continue
			}
			if got == nil || got.String() != tt.want {
				t.Errorf("%s = %v, want %s", tt.expr, got, tt.want)
			}
		}
	}
}

// TestMixedPrecisionStillUnknown pins what must not move. Equality across
// different precisions is unknown, and that is the specification's rule rather
// than a gap: "If one input has a value for the precision and the other does not,
// the comparison stops and the result is null."
//
// The fix above is only about a pair specified to the SAME precision that the
// engine was ordering as equal and calling unequal.
func TestMixedPrecisionStillUnknown(t *testing.T) {
	for _, expr := range []string{
		// Day against second: the Date has no hour to compare.
		"@2020-03-01 = @2020-03-01T00:00:00",
		"@2020-03-01 != @2020-03-01T00:00:00",
		// Year against day.
		"@2020 = @2020-03-01",
		// And the same rule between two DateTimes, which never changed.
		"@2020-03-01T = @2020-03-01T00:00:00",
	} {
		got, err := evalCQL(t, expr)
		if err != nil {
			t.Errorf("%s: %v", expr, err)
			continue
		}
		if got != nil {
			t.Errorf("%s = %v, want null: one side has no value at the precision "+
				"the other is specified to", expr, got)
		}
	}

	// Decidable inequality stays decidable.
	for _, tt := range []struct{ expr, want string }{
		{"@2020-03-01 = @2020-03-02T", "false"},
		{"@2020-03-01 = @2020-04-01T", "false"},
		{"@2020-03-01 != @2020-03-02T", "true"},
		// Ordinary same-type equality is untouched.
		{"@2020-03-01 = @2020-03-01", "true"},
		{"@2020-03-01T = @2020-03-01T", "true"},
		{"@2020-03-01 = @2020-03-02", "false"},
	} {
		got, err := evalCQL(t, tt.expr)
		if err != nil {
			t.Errorf("%s: %v", tt.expr, err)
			continue
		}
		if got == nil || got.String() != tt.want {
			t.Errorf("%s = %v, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestEquivalenceDoesNotSwallowPrecisionGaps covers two defects review found in
// the equivalence half of this change, both from generalizing a policy further
// than the evidence for it went.
//
// `~` was made to answer true wherever equality was unknown, citing a test that
// expects `@2017-09-01T00:00:00 ~ @2017-09-01T00:00:00.000` to be true. That test
// was true on main for a different reason — fhirpath folds seconds and
// milliseconds into one precision, so Equal already matched the instant — so it
// licensed nothing about a year against a day.
//
// Two consequences, both measured against main:
//
//   - Every precision gap became equivalent: `@2020-06-15 ~ @2020` and
//     `@T10:30 ~ @T10:30:15` flipped to true. That makes `~` intransitive and
//     puts it at odds with list equivalence.
//   - An unknown timezone offset became equivalent: `@2020-03-01T10:00:00 ~
//     @2020-03-01T10:00:00+05:00` flipped to true, for a pair the engine refuses
//     to order or equate and whose instants may be 26 hours apart. The comment
//     justifying it claimed CompareTemporal reports its mismatch only when every
//     shared component agrees — true of the precision path, false of the offset
//     one.
//
// A third case was reported alongside them and is not one: against Z, `~` answers
// true on main as well. Measuring both sides rather than trusting the report is
// what separated the two regressions from the thing that was already there.
//
// So `~` takes the shared decision only where it decides. Where equality is
// unknown, equivalence answers what it answered before.
func TestEquivalenceDoesNotSwallowPrecisionGaps(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		// Precision gaps are not equivalence.
		{"@2020-06-15 ~ @2020", "false"},
		{"@2020-03-01 ~ @2020-03-01T23:59:59", "false"},
		{"@T10:30 ~ @T10:30:15", "false"},

		// Nor is an offset nobody wrote down, where the written one is not UTC.
		{"@2020-03-01T10:00:00 ~ @2020-03-01T10:00:00+05:00", "false"},

		// Against Z it answers true, and that is pre-existing rather than
		// introduced here: main answers the same, because fptypes.Equivalent
		// reads the unwritten offset as UTC and the two then name one instant.
		// Review reported this as a regression; measuring both sides is what
		// showed it is not one. Left alone deliberately — `=` gives null for the
		// same pair, and reconciling that is its own change with its own
		// measurement, not a correction to this one.
		{"@2020-03-01T00:00:00 ~ @2020-03-01T00:00:00Z", "true"},

		// The case that does hold, and the one the policy was quoted from.
		{"@2017-09-01T00:00:00 ~ @2017-09-01T00:00:00.000", "true"},

		// And what this change was actually for: same precision, same day.
		{"@2020-03-01 ~ @2020-03-01T", "true"},
		{"@2020 ~ @2020T", "true"},
		{"@2020-03-01 ~ @2020-03-02T", "false"},
	} {
		got, err := evalCQL(t, tt.expr)
		if err != nil {
			t.Errorf("%s: %v", tt.expr, err)
			continue
		}
		if got == nil || got.String() != tt.want {
			t.Errorf("%s = %v, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestEqualityPropagatesToCompositeValues is the finding that mattered most from
// review, because it recreated the exact contradiction this change argues from,
// one level up:
//
//	start of A = start of B   true
//	end of A   = end of B     true
//	A = B                     false     ← the same thing that was fixed for scalars
//
// with A = Interval[@2020-03-01, @2020-03-05] and B its DateTime spelling. And in
// six more shapes: list equality, `in`, `distinct`, `IndexOf`, tuples, ratios.
//
// They all read their elements through Value.Equal, which compares types, so
// fixing the scalar operator left every composite that contains a scalar
// disagreeing with it. Fixed where they meet rather than one at a time:
// types.valuesEqual is the single place lists, tuples, ratios and interval
// boundaries decide whether two elements are the same.
func TestEqualityPropagatesToCompositeValues(t *testing.T) {
	const (
		a = "Interval[@2020-03-01, @2020-03-05]"
		b = "Interval[@2020-03-01T, @2020-03-05T]"
	)

	// The premise, and the contradiction it used to sit beside.
	for _, expr := range []string{
		"start of " + a + " = start of " + b,
		"end of " + a + " = end of " + b,
	} {
		got, err := evalCQL(t, expr)
		if err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
		if got == nil || got.String() != "true" {
			t.Fatalf("%s = %v, want true — the premise of this test", expr, got)
		}
	}

	for _, tt := range []struct{ expr, want string }{
		// Both boundaries equal, so the intervals are equal.
		{a + " = " + b, "true"},
		{a + " ~ " + b, "true"},

		// Lists, membership, and the operations built on them.
		{"{@2020-03-01} = {@2020-03-01T}", "true"},
		{"@2020-03-01 in {@2020-03-01T}", "true"},
		{"Count(distinct {@2020-03-01, @2020-03-01T})", "1"},
		{"IndexOf({@2020-03-01T}, @2020-03-01)", "0"},

		// Tuples and ratios hold values too.
		{"Tuple{d: @2020-03-01} = Tuple{d: @2020-03-01T}", "true"},

		// And a genuinely different day stays different, everywhere.
		{a + " = Interval[@2020-03-01T, @2020-03-06T]", "false"},
		{"{@2020-03-01} = {@2020-03-02T}", "false"},
		{"@2020-03-01 in {@2020-03-02T}", "false"},
		{"Count(distinct {@2020-03-01, @2020-03-02T})", "2"},
	} {
		got, err := evalCQL(t, tt.expr)
		if err != nil {
			t.Errorf("%s: %v", tt.expr, err)
			continue
		}
		if got == nil || got.String() != tt.want {
			t.Errorf("%s = %v, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestTimingPhraseMembershipAgreesWithIn is the third round of the same mistake,
// and worth naming as such: fixing equality in one place and leaving the operator
// beside it disagreeing.
//
// `in` and `contains` were routed through the shared decision. The timing phrases
// over a list — `includes`, `included in`, `during`, `properly includes` — go
// through listContainsValueTriState, which was not, so they answered false where
// `in` answered true. On main all of them agreed, so this was introduced by the
// fix rather than found by it.
func TestTimingPhraseMembershipAgreesWithIn(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"@2020-03-01 in {@2020-03-01T}", "true"},
		{"{@2020-03-01T} contains @2020-03-01", "true"},
		{"{@2020-03-01T} includes @2020-03-01", "true"},
		{"@2020-03-01 included in {@2020-03-01T}", "true"},
		{"@2020-03-01 during {@2020-03-01T}", "true"},
		{"{@2020-03-01T, @2020-03-05T} properly includes @2020-03-01", "true"},

		// And a day that is genuinely absent stays absent for all of them.
		{"@2020-03-02 in {@2020-03-01T}", "false"},
		{"{@2020-03-01T} includes @2020-03-02", "false"},
		{"@2020-03-02 during {@2020-03-01T}", "false"},
	} {
		got, err := evalCQL(t, tt.expr)
		if err != nil {
			t.Errorf("%s: %v", tt.expr, err)
			continue
		}
		if got == nil || got.String() != tt.want {
			t.Errorf("%s = %v, want %s", tt.expr, got, tt.want)
		}
	}
}
