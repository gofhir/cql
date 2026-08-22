package cql

import "testing"

// TestTimingOperatorsAgreeWithTheOrderedComparisons closes the divergence the
// review of the timezone work found and that work deliberately left open.
//
// `before`, `after`, `same as` and `same day as` reach the temporal comparison by
// their own route — temporalCompareAtPrecision, over temporalComponents — and so
// never saw the offset rule. Inside the 26-hour window `<` declined while
// `before` answered true.
//
// The cause is one line in temporalComponents: a value that writes an offset is
// normalized to UTC, and one that does not keeps its local components. That
// compares two different frames, which is the mistake CompareTemporal's own
// comment warns about and avoids:
//
//	"Reading one value at UTC while the other stays in local components compares
//	two different frames"
//
// The corpus says normalizing is right. SameAsTimezoneFalse compares
// @2012-03-10T10:20:00.999+07:00 with @2012-03-10T10:20:00.999+06:00 at the hour
// and expects false — 03:20 and 04:20 at UTC — so these operators read instants,
// not the digits each value happens to be written with. An unwritten offset
// therefore leaves the instant unknown across the same window, and the same rule
// applies: knowable where the whole window falls on one side.
//
// Measured against the published measures before changing anything, because that
// is what the earlier round said had to happen first. Neither use is affected:
//
//   - ControllingHighBloodPressureFHIR writes `DBPReading.effective same day as
//     "Most Recent Blood Pressure Day"`, and that second value is `date from
//     ...`, a Date. A Date has no time of day and no offset to be missing, so
//     the pair is the mixed Date/DateTime question, which this does not touch.
//   - `start of Payer.period same as start of InpatientEncounter.period` reads
//     two FHIR periods, so either both carry an offset or neither does.
func TestTimingOperatorsAgreeWithTheOrderedComparisons(t *testing.T) {
	const (
		a = "@2020-03-05T10:00:00Z"  // offset written
		b = "@2020-03-05T11:00:00.0" // none written, an hour later as written
	)

	// Inside the window every operator has to decline, not just the ordered ones.
	// An offset of +14:00 puts b on the 4th of March at UTC and one of -12:00
	// puts it late on the 5th, so neither the instant nor even the day is settled.
	for _, expr := range []string{
		a + " < " + b,
		a + " > " + b,
		a + " = " + b,
		a + " before " + b,
		a + " after " + b,
		a + " same as " + b,
		a + " same day as " + b,
		a + " same hour as " + b,
		a + " on or before " + b,
		a + " on or after " + b,
	} {
		got, err := evalCQL(t, expr)
		if err != nil {
			t.Errorf("%s: %v", expr, err)
			continue
		}
		if got != nil {
			t.Errorf("%s = %v, want null — an unwritten offset moves this either way", expr, got)
		}
	}

	// Outside the window they answer, and they answer the same thing.
	const far = "@2020-04-05T11:00:00.0" // a month later, no offset written
	for _, tt := range []struct{ expr, want string }{
		{a + " < " + far, "true"},
		{a + " before " + far, "true"},
		{a + " after " + far, "false"},
		{a + " same as " + far, "false"},
		{a + " same day as " + far, "false"},
		{a + " same month as " + far, "false"},
		{a + " on or before " + far, "true"},
		// And a year is coarse enough to settle even at that precision.
		{a + " same year as " + far, "true"},
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

// TestTimingOperatorsUnchangedWhereTheCorpusSpeaks pins what must not move. Every
// `same as` case the conformance corpus carries writes an offset on both sides,
// and the published measures compare a dateTime against a Date — so both of those
// have to answer exactly as before.
func TestTimingOperatorsUnchangedWhereTheCorpusSpeaks(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		// Straight from the corpus.
		{"@2022-02-22T00:00:00.000-05:00 same day as @2022-02-22T04:59:00.000Z", "true"},
		{"@2012-03-10T10:20:00.999+07:00 same hour as @2012-03-10T09:20:00.999+06:00", "true"},
		{"@2012-03-10T10:20:00.999+07:00 same hour as @2012-03-10T10:20:00.999+06:00", "false"},
		{"@2012-03-10T10:20:00.999+07:00 same hour or after @2012-03-10T09:20:00.999+06:00", "true"},
		{"@2012-03-10T09:20:00.999+07:00 same hour or before @2012-03-10T10:20:00.999+06:00", "true"},

		// The shape the measures use: a dateTime against a Date. A Date has no
		// offset to be missing, so this is untouched.
		{"@2020-03-05T10:00:00Z same day as @2020-03-05", "true"},
		{"@2020-03-05T10:00:00Z same day as @2020-03-06", "false"},

		// Neither side writing an offset is untouched too.
		{"@2020-03-05T10:00:00 same day as @2020-03-05T11:00:00.0", "true"},
		{"@2020-03-05T10:00:00 before @2020-03-05T11:00:00.0", "true"},

		// And both writing the same one.
		{"@2020-03-05T10:00:00Z same day as @2020-03-05T11:00:00.0Z", "true"},
		{"@2020-03-05T10:00:00Z before @2020-03-05T11:00:00.0Z", "true"},
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
