package funcs

import (
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

// TestDurationFollowsTheDefaultOffset covers a split that fhirpath reported and
// that turned out to be on this side.
//
// fhirpath v1.9.0 lets a caller say what offset to assume for a value that states
// none: WithDefaultOffset sets it, EffectiveOffset reads it back, and comparison
// honors it. What it does not do is change the value's own offset, because
// materializing it broke 22 conformance cases — a literal written without an
// offset has to evaluate to itself.
//
// So ToTime() still places such a value at UTC. Every duration and difference
// operator here goes through toGoTime, which called ToTime and therefore measured
// from UTC while the comparison operators measured from the default. At an
// evaluation offset of UTC-5 the engine would say a bare value was later than an
// instant and also two hours earlier than it.
//
// toGoTime consults EffectiveOffset now. It is the single conversion those
// operators share, which is why the fix belongs there and not in each of them.
func TestDurationFollowsTheDefaultOffset(t *testing.T) {
	mk := func(s string, def time.Duration) fptypes.Value {
		dt, err := fptypes.NewDateTime(s)
		if err != nil {
			t.Fatalf("%s: %v", s, err)
		}
		return dt.WithDefaultOffset(def)
	}

	for _, tt := range []struct {
		offset time.Duration
		want   string
	}{
		{0, "2"},               // bare read as 00:00Z, two hours before 02:00Z
		{-5 * time.Hour, "-3"}, // bare read as 05:00Z, three hours after
		{9 * time.Hour, "11"},  // bare read as 15:00Z the day before
	} {
		bare := mk("2020-01-01T00:00:00.0", tt.offset)
		stated := mk("2020-01-01T02:00:00Z", tt.offset) // a stated offset wins

		got, err := HoursBetween(bare, stated)
		if err != nil {
			t.Errorf("offset %s: %v", tt.offset, err)
			continue
		}
		if got == nil || got.String() != tt.want {
			t.Errorf("HoursBetween at default %s = %v, want %s", tt.offset, got, tt.want)
		}
	}

	// A stated offset is untouched by the default, so a pair that states both
	// measures the same however the default is set.
	for _, offset := range []time.Duration{0, -5 * time.Hour, 9 * time.Hour} {
		got, err := HoursBetween(
			mk("2020-01-01T00:00:00.0Z", offset),
			mk("2020-01-01T02:00:00Z", offset))
		if err != nil {
			t.Errorf("stated pair at %s: %v", offset, err)
			continue
		}
		if got == nil || got.String() != "2" {
			t.Errorf("stated pair at default %s = %v, want 2", offset, got)
		}
	}

	// And with no default supplied at all, nothing moves: this is what the engine
	// does today, and applying a default is a separate decision.
	plain, _ := fptypes.NewDateTime("2020-01-01T00:00:00.0")
	stated, _ := fptypes.NewDateTime("2020-01-01T02:00:00Z")
	got, err := HoursBetween(plain, stated)
	if err != nil {
		t.Fatalf("no default: %v", err)
	}
	if got == nil || got.String() != "2" {
		t.Errorf("with no default = %v, want 2 — unchanged from before", got)
	}
}

// TestOffsetExtractionReportsTheDefault covers the other half. CQL says that where
// the offset was defaulted from the evaluation request, extracting it gives "the
// timezone offset of the evaluation request, not null" — and this answered null,
// because it asked HasTZ() and a default deliberately does not set that.
func TestOffsetExtractionReportsTheDefault(t *testing.T) {
	mk := func(s string, def time.Duration) fptypes.Value {
		dt, _ := fptypes.NewDateTime(s)
		return dt.WithDefaultOffset(def)
	}

	for _, tt := range []struct {
		value  string
		offset time.Duration
		want   string
	}{
		// Defaulted: reported, in hours, as CQL's component is.
		{"2012-03-10T10:20:00", 0, "0"},
		{"2012-03-10T10:20:00", -5 * time.Hour, "-5"},
		{"2012-03-10T10:20:00", 9 * time.Hour, "9"},
		// Stated: the stated one, whatever the default.
		{"2012-03-10T10:20:00+07:00", -5 * time.Hour, "7"},
		{"2012-03-10T10:20:00Z", 9 * time.Hour, "0"},
	} {
		got, err := DateTimeComponentFrom(mk(tt.value, tt.offset), "timezoneoffset")
		if err != nil {
			t.Errorf("%s at %s: %v", tt.value, tt.offset, err)
			continue
		}
		if got == nil || got.String() != tt.want {
			t.Errorf("timezoneoffset from %s at default %s = %v, want %s",
				tt.value, tt.offset, got, tt.want)
		}
	}

	// With no default and no stated offset there is nothing to report.
	plain, _ := fptypes.NewDateTime("2012-03-10T10:20:00")
	got, err := DateTimeComponentFrom(plain, "timezoneoffset")
	if err != nil {
		t.Fatalf("%v", err)
	}
	if got != nil {
		t.Errorf("with neither stated nor defaulted = %v, want null", got)
	}
}
