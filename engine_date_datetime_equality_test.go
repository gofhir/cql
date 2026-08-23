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
