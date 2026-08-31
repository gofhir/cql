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

		// Cases that pair a stated offset with a bare literal are not here any
		// more: a bare literal assumes the evaluation request's offset now, so
		// their answers follow the zone the tests run in.
		// TestLiteralsAssumeTheRequestOffset pins them with the zone fixed, which
		// is the only way such an assertion means anything.

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

	// The duration operators read the same pair and have to place it too. Their
	// answers are asserted as non-null rather than as figures, because a bare
	// literal assumes the evaluation request's offset and a day count moves by one
	// at the extremes — 301 at UTC and 302 at UTC-11. The figures are pinned in
	// TestLiteralsAssumeTheRequestOffset, where the zone is fixed.
	for _, expr := range []string{
		"months between " + earlier + " and " + later,
		"duration in days of Interval[" + earlier + ", " + later + "]",
	} {
		got, err := evalCQL(t, expr)
		if err != nil {
			t.Errorf("%s: %v", expr, err)
			continue
		}
		if got == nil {
			t.Errorf("%s = null, want a figure", expr)
		}
	}
}

// TestMinMaxKeepATotalOrderPlacesEveryPair covers the rule Min and Max are held
// to, which is not CQL's comparison: "Min, Max and sort need a total order;
// answering unknown there would drop values from a result rather than describe
// them, so they keep ordering at the shared precision."
//
// This test used to build its pair from an offset the engine could not place.
// Since a literal now assumes the evaluation request's offset, that pair orders
// like any other, so the property is pinned on a pair that still cannot be
// settled by comparison — two DateTimes specified to different precisions.
func TestMinMaxKeepATotalOrderPlacesEveryPair(t *testing.T) {
	const (
		a = "@2017-09-01T00:00:00"     // second precision
		b = "@2017-09-01T00:00:00.000" // millisecond precision
	)

	// The premise: comparison declines this pair, and that has not changed.
	for _, expr := range []string{a + " < " + b, a + " = " + b} {
		got, err := evalCQL(t, expr)
		if err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
		if got != nil {
			t.Fatalf("%s = %v, want null — this test's premise is that the pair does "+
				"not order by comparison", expr, got)
		}
	}

	// And Min and Max place it anyway, without failing. Which of the two comes
	// back is not the property: they are equal as far as both are specified, so
	// either is a correct minimum. What matters is that an answer comes back at
	// all, where a bare `continue` used to leave whichever element came first.
	for _, expr := range []string{
		"Min({" + a + ", " + b + "})",
		"Min({" + b + ", " + a + "})",
		"Max({" + a + ", " + b + "})",
		"Max({" + b + ", " + a + "})",
	} {
		got, err := evalCQL(t, expr)
		if err != nil {
			t.Errorf("%s: %v — a total order has to place this pair", expr, err)
			continue
		}
		if got == nil {
			t.Errorf("%s = null, want a value", expr)
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

// TestOffsetMismatchIsRecognizedWhicheverWayUpstreamReportsIt guards a break that
// would have arrived with a dependency bump and passed the whole conformance
// corpus on the way in.
//
// fhirpath used to report an absent timezone offset with its precision-mismatch
// sentinel, which is a false diagnosis — the precisions usually agree — and that
// is fixed upstream: an offset mismatch has its own sentinel now, and
// IsUnknownTemporalComparison asks about both.
//
// This engine recognized only the precision sentinel. Measured against the merge
// commit before any release existed, that turned:
//
//	Z < bare    from null into a raised error
//	Min / Max   from a value into a raised error
//	Z = bare    from null into TRUE
//
// The last is a wrong answer rather than a loud failure. And the conformance
// corpus stays at 2084 passing through all of it, because no case in it pairs a
// written offset with a missing one — so nothing would have caught it.
//
// The assertions below are about behavior rather than about which sentinel came
// back, so they hold on both versions and will keep holding when the text match
// they currently rely on is replaced by IsUnknownTemporalComparison.
func TestOffsetMismatchIsRecognizedWhicheverWayUpstreamReportsIt(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		// An answer rather than a raised error, which is the property here. A
		// literal assumes the evaluation request's offset now, so these are
		// decided rather than unknown; what must not happen either way is the
		// sentinel escaping as an error.
		{"@2020-03-05T10:00:00Z < @2021-01-01T00:00:00.0", "true"},
		{"@2020-03-05T10:00:00Z > @2019-01-01T00:00:00.0", "true"},

		// And the aggregates still place the pair instead of failing.
		{"Min({@2020-03-05T10:00:00Z, @2021-01-01T00:00:00.0}) = @2020-03-05T10:00:00Z", "true"},
		{"Max({@2020-03-05T10:00:00Z, @2021-01-01T00:00:00.0}) = @2021-01-01T00:00:00.0", "true"},
	} {
		got, err := evalCQL(t, tt.expr)
		if err != nil {
			t.Errorf("%s raised %v — an unplaceable comparison has no answer, "+
				"which is not the same as failing", tt.expr, err)
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

// TestOneFrameAcrossEveryComponentReader pins a value that has a timezone frame
// against one that has none, through every path that lines temporal values up
// component by component.
//
// A DateTime specified no finer than the day names a day, not an instant. The
// engine normalizes a value into UTC once it knows its offset — stated, or assumed
// from the evaluation request — and normalizing only the side that has an hour
// compares two different frames. At UTC+14 that rolls the placed side back a day,
// which does not merely leave a comparison unknown but answers it wrongly:
//
//	DateTime(2020,1,1,12) during day of Interval[DateTime(2020,1,1), DateTime(2020,1,3)]
//	  true at UTC, UTC-5 and UTC+9; false at UTC+14
//
// There were four readers, not one — sorting, interval-against-interval,
// point-against-interval, and the timing operators — and each had its own copy of
// the mistake. That is the same shape of defect this engine has had to correct
// three times now (v1.15.2's seven paths to interval equality, v1.18.x's eleven
// operators deciding element equality), so the test enumerates the readers rather
// than covering one and trusting the rest.
//
// Every case is asserted across the zones because the answers only diverge at the
// extremes: a one-zone run of any of this is green while three of the four readers
// are wrong.
func TestOneFrameAcrossEveryComponentReader(t *testing.T) {
	const (
		day  = "DateTime(2020,1,1)"    // no hour: names a day, has no frame
		hour = "DateTime(2020,1,1,12)" // an instant, placed at the request's offset
	)

	for _, tt := range []struct{ reader, expr, want string }{
		{
			"sorting", "First(({" + hour + ", " + day + "}) S sort asc) = " + day, "true",
		},
		{
			"point against an interval",
			hour + " during day of Interval[" + day + ", DateTime(2020,1,3)]", "true",
		},
		{
			"point against an interval, properly",
			hour + " properly during day of Interval[DateTime(2019,12,30), DateTime(2020,1,3)]", "true",
		},
		{
			"interval against an interval",
			"Interval[" + hour + ", DateTime(2020,1,3,14)] included in day of Interval[" +
				day + ", DateTime(2020,1,3)]", "true",
		},
		{
			"the timing operators",
			hour + " same day as " + day, "true",
		},
		// And the pair whose comparison must stay unknown: agreeing on every
		// component they share while one is specified more finely is not equality,
		// and no offset makes it one.
		{
			"an ordering that cannot be settled", day + " < " + hour, "null",
		},
	} {
		for _, hours := range []int{0, -5, 9, 14, -11} {
			got := askAtOffset(t, hours, tt.expr)
			if got != tt.want {
				t.Errorf("%s: %s = %s at UTC%+d, want %s — a value with no hour has no "+
					"frame to be moved into, so neither side is normalized against it",
					tt.reader, tt.expr, got, hours, tt.want)
			}
		}
	}
}
