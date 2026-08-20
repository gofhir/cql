package cql

import "testing"

// TestComparisonAcrossWrittenTimezone covers the comparison that decides which
// population a patient belongs to, and answered null.
//
// A FHIR dateTime that carries a time must carry an offset, so every Period a
// server serves looks like "2020-03-05T10:00:00Z". A measure's "Measurement
// Period" is written without one: every published eCQM library declares it as
// Interval[@2019-01-01T00:00:00.0, @2020-01-01T00:00:00.0). Comparing the two
// reported a precision mismatch — including where both are specified to the same
// precision, which is what gives the diagnosis away. It is not a precision that
// is missing, it is an offset.
//
// This is a defect in fhirpath's Compare, present in v1.6.0 and v1.8.0 alike and
// reported upstream. Both specifications say an absent offset resolves rather
// than invalidates:
//
//	CQL, DateTime Literals: "If no timezone offset is specified, the timezone
//	offset of the evaluation request timestamp is used" — and on extracting it,
//	"the result ... will be the timezone offset of the evaluation request, not
//	null".
//
//	FHIRPath, Comparison: "either both values have no timezone offset specified,
//	or both values are converted to a common timezone offset".
//
// CompareTemporal already exists to absorb where fptypes and CQL part company, so
// it absorbs this too. It delegates first, so when fhirpath stops reporting the
// mismatch this compensation simply stops being reached.
//
// What it does NOT do is invent the evaluation request's offset. It answers only
// where the answer holds for every offset a value could have been written in:
// legal offsets span -12:00 to +14:00, so an unwritten offset leaves the instant
// uncertain across a 26-hour window, and a comparison is knowable exactly when
// that whole window falls on one side. Ten months apart is knowable. One hour
// apart is not, and stays null — the same shape as an uncertainty, and the same
// reason: knowable outside the range, unknown inside.
func TestComparisonAcrossWrittenTimezone(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		// Months apart: no offset can reorder these.
		{"@2020-03-05T10:00:00Z < @2021-01-01T00:00:00.0", "true"},
		{"@2020-03-05T10:00:00Z > @2019-01-01T00:00:00.0", "true"},
		{"@2020-03-05T10:00:00Z > @2021-01-01T00:00:00.0", "false"},
		// Same precision on both sides, which the mismatch could never have been
		// about.
		{"@2020-03-05T10:00:00.0Z < @2021-01-01T00:00:00.0", "true"},
		// The offset written on the other side instead.
		{"@2020-03-05T10:00:00.0 < @2021-01-01T00:00:00Z", "true"},
		// Two days apart clears the window; one day does not, and is asserted
		// below.
		{"@2020-03-05T10:00:00Z < @2020-03-07T10:00:00.0", "true"},

		// Inside the window, where the answer really does depend on the offset
		// the value was written in. Null, not a guess.
		{"@2020-03-05T10:00:00Z < @2020-03-05T11:00:00.0", "null"},
		{"@2020-03-05T10:00:00Z = @2020-03-05T10:00:00.0", "null"},
		{"@2020-03-05T10:00:00Z < @2020-03-06T10:00:00.0", "null"},

		// Controls: both offsets written, and neither written, are untouched.
		{"@2020-03-05T10:00:00Z < @2021-01-01T00:00:00.0Z", "true"},
		{"@2020-03-05T10:00:00 < @2021-01-01T00:00:00.0", "true"},
		{"@2020-03-05T10:00:00Z = @2020-03-05T10:00:00Z", "true"},
		// And a genuine precision mismatch is still one. CQL keeps millisecond a
		// precision of its own, so a value written to the second and one written
		// to the millisecond are specified differently and comparing them is
		// unknown — offsets or no offsets.
		{"@2020-03-05T10:00:00Z = @2020-03-05T10:00:00.500Z", "null"},
		{"@2020-01-01 = @2020-01-01T00:00:00.0", "null"},
	} {
		got, err := evalCQL(t, tt.expr)
		if err != nil {
			t.Errorf("%s: %v", tt.expr, err)
			continue
		}
		if tt.want == "null" {
			if got != nil {
				t.Errorf("%s = %v, want null", tt.expr, got)
			}
			continue
		}
		if got == nil || got.String() != tt.want {
			t.Errorf("%s = %v, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestMeasurementPeriodMembershipAcrossTimezone is the shape the published
// libraries are built on: 11 of the 19 write `during "Measurement Period"`, 38
// times between them, against data whose periods carry an offset.
func TestMeasurementPeriodMembershipAcrossTimezone(t *testing.T) {
	const mp = "Interval[@2020-01-01T00:00:00.0, @2021-01-01T00:00:00.0)"

	for _, tt := range []struct{ expr, want string }{
		// A period served by a FHIR server, inside the measurement period.
		{"Interval[@2020-03-01T00:00:00Z, @2020-03-05T10:00:00Z] during " + mp, "true"},
		{"Interval[@2020-03-01T00:00:00Z, @2020-03-05T10:00:00Z] overlaps " + mp, "true"},
		{"Interval[@2020-03-01T00:00:00Z, @2020-03-05T10:00:00Z] included in " + mp, "true"},
		{"@2020-03-05T10:00:00Z in " + mp, "true"},

		// Wholly outside it.
		{"Interval[@2022-03-01T00:00:00Z, @2022-03-05T10:00:00Z] during " + mp, "false"},
		{"@2022-03-05T10:00:00Z in " + mp, "false"},
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

// TestOneOrderAcrossEveryOperator is the property this repository has had to
// re-establish before: a comparison rule added in one place is a rule seven
// operators do not follow. v1.15.2 found interval equality answering one thing
// while `start of` answered another, across seven paths through the same helper.
//
// So: one pair, ten months apart, asked by every route that decides an order.
// They have to agree, and Min and Max have to disagree with each other.
func TestOneOrderAcrossEveryOperator(t *testing.T) {
	const (
		earlier = "@2020-03-05T10:00:00Z"  // offset written
		later   = "@2021-01-01T00:00:00.0" // none written
	)

	for _, tt := range []struct{ expr, want string }{
		{earlier + " < " + later, "true"},
		{earlier + " <= " + later, "true"},
		{earlier + " > " + later, "false"},
		{earlier + " >= " + later, "false"},
		{earlier + " = " + later, "false"},
		{earlier + " != " + later, "true"},
		{earlier + " ~ " + later, "false"},
		{earlier + " before " + later, "true"},
		{earlier + " after " + later, "false"},
		{earlier + " same as " + later, "false"},
		{earlier + " same day as " + later, "false"},
		// Read the other way round, the same order has to come out.
		{later + " > " + earlier, "true"},
		{later + " after " + earlier, "true"},

		// Min and Max used to answer the same value, because the comparison
		// error was skipped and the first element stood.
		{"Min({" + earlier + ", " + later + "}) = " + earlier, "true"},
		{"Max({" + earlier + ", " + later + "}) = " + later, "true"},
		// And in the other input order, since that is what made it wrong.
		{"Min({" + later + ", " + earlier + "}) = " + earlier, "true"},
		{"Max({" + later + ", " + earlier + "}) = " + later, "true"},
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

	// The duration operators read the same pair and have to place it too.
	for _, tt := range []struct{ expr, want string }{
		{"months between " + earlier + " and " + later, "9"},
		{"duration in days of Interval[" + earlier + ", " + later + "]", "301"},
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

// TestMinMaxKeepATotalOrderInsideTheWindow covers the one operation that does
// NOT decline inside the offset window, and must not.
//
// The ordered comparisons and the timing operators all answer null there — the
// unwritten offset moves the pair either way, and the operators say so.
// TestTimingOperatorsAgreeWithTheOrderedComparisons pins that agreement.
//
// Min and Max are held to a different rule, written down where it is enforced:
// "Min, Max and sort need a total order; answering unknown there would drop
// values from a result rather than describe them, so they keep ordering at the
// shared precision." So they still place this pair. What they must not do is what
// they used to: return the same value as each other, or a different one when the
// list is reordered.
func TestMinMaxKeepATotalOrderInsideTheWindow(t *testing.T) {
	const (
		a = "@2020-03-05T10:00:00Z"  // offset written
		b = "@2020-03-05T11:00:00.0" // none written, an hour later as written
	)

	// The premise: no operator can order this pair.
	for _, expr := range []string{a + " < " + b, a + " before " + b} {
		got, err := evalCQL(t, expr)
		if err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
		if got != nil {
			t.Fatalf("%s = %v, want null — this test's premise is that the pair does not order", expr, got)
		}
	}

	// And Min and Max place it anyway, consistently, in either input order.
	for _, tt := range []struct{ expr, want string }{
		{"Min({" + a + ", " + b + "}) = " + a, "true"},
		{"Max({" + a + ", " + b + "}) = " + b, "true"},
		{"Min({" + b + ", " + a + "}) = " + a, "true"},
		{"Max({" + b + ", " + a + "}) = " + b, "true"},
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

// TestTimeIsNotADateTimeMissingAnOffset guards the regression the review caught.
//
// The compensation first excluded Date and forgot Time. A Time reports no offset
// and has no day to place, so it landed at year zero on the clock, the gap
// cleared the margin, and an order came back for a pair fhirpath refuses
// outright: `@2020-03-05T10:00:00Z > @T10:00:00` answered true, while the same
// pair written without an offset kept erroring. The compensation had invented
// both an order and a disagreement.
//
// It requires a DateTime on both sides now — naming what qualifies rather than
// listing what does not, which is what let a type slip through the first time.
func TestTimeIsNotADateTimeMissingAnOffset(t *testing.T) {
	for _, expr := range []string{
		"@2020-03-05T10:00:00Z > @T10:00:00",
		"@T10:00:00 < @2020-03-05T10:00:00Z",
		"@T10:00:00 in Interval[@2020-01-01T00:00:00Z, @2021-01-01T00:00:00Z]",
		"Min({@2020-03-05T10:00:00Z, @T10:00:00})",
		// A Date writes no offset either, and comparing it against a DateTime is
		// the mixed Date/DateTime question, which this does not answer.
		"@2020-03-05T10:00:00Z = @2020-03-05",
	} {
		got, err := evalCQL(t, expr)
		if err != nil {
			continue // refusing is what main does, and what this must keep doing
		}
		if got != nil {
			t.Errorf("%s = %v, want an error or null: a Time has no day and a Date "+
				"has no time of day, so neither is a DateTime that omitted an offset", expr, got)
		}
	}
}
