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
		// A day apart is still outside the window.
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
