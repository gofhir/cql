package cql

import "testing"

// TestCalendarAndRangeAgreeWithTheReference pins what fhirpath v1.14 changed in
// how a DateTime is built and moved, each against what the reference engine
// (info.cqframework:engine 3.26.0, and org.cqframework:engine-jvm 5.4.0
// re-measured 22-10-2026, plan 007) answers for the same expression evaluated at
// UTC-3. Before v1.14 every one of these answered differently here: a date off
// the calendar was accepted, a large clock amount wrapped around, and a value
// at the end of the range could not be compared with the maximum at all.
//
// They are pinned because the change came from a dependency. A later bump that
// undid one would otherwise go through with every other test green.
func TestCalendarAndRangeAgreeWithTheReference(t *testing.T) {
	for _, tt := range []struct{ expr, want, why string }{
		{
			"DateTime(2020, 2, 30)", "ERROR",
			"there is no 30 February; the reference refuses it as InvalidDateTime",
		},
		{
			"ToDateTime('2020-02-30T10:00:00')", "null",
			"a conversion from text answers null for text that names no date, as the reference does",
		},
		{
			"@2020-01-01T00:00:00Z + 3000000 hours", "2362-03-29T00:00:00Z",
			"the amount is added, not wrapped: this used to answer 1777-09-07T00:25:26",
		},
		{
			"@9999-12-31T23:59:59.999 > maximum DateTime", "true",
			"written at UTC-3, the last millisecond of the range is three hours past the maximum, " +
				"which is stated at UTC",
		},
		{
			"Interval[@2020-01-01T00:00:00, maximum DateTime] contains @9999-12-31T23:59:59.999", "false",
			"same instant, so outside the interval",
		},
		{
			"@0001-01-01T00:00:00.000Z < minimum DateTime", "false",
			"the minimum stated at UTC is exactly that instant",
		},
	} {
		if got := askAtOffset(t, -3, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — %s", tt.expr, got, tt.want, tt.why)
		}
	}
}

// TestOffsetsPastFourteenHoursAreRefused pins a place where this engine and the
// reference part ways, and says so rather than leaving it to be rediscovered.
//
// fhirpath v1.14 refuses an offset beyond ±14:00, which is the range a FHIR
// dateTime allows, and the refusal ends the evaluation. The reference engine
// accepts both and answers a DateTime at +14:30 and +15:00. No measure writes
// such an offset, so the difference is recorded, not resolved; a change here
// means fhirpath moved, and should be read against this note.
func TestOffsetsPastFourteenHoursAreRefused(t *testing.T) {
	for _, expr := range []string{
		"DateTime(2020, 1, 1, 0, 0, 0, 0, 14.5)",
		"DateTime(2020, 1, 1, 0, 0, 0, 0, 15.0)",
	} {
		if got := askAtOffset(t, -3, expr); got != "ERROR" {
			t.Errorf("%s = %s, want ERROR — fhirpath v1.14 refuses an offset past ±14:00 "+
				"(the reference engine accepts it)", expr, got)
		}
	}
}
