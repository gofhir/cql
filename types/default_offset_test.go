package types

import (
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

// A DateTime is placed at the offset from the hour on and left unplaced above
// it. The assertion is on EffectiveOffset, which is what a comparison reads:
// String() prints the same either way.
func TestWithDefaultOffsetPlacesFromTheHour(t *testing.T) {
	for _, tt := range []struct {
		text   string
		placed bool
	}{
		{"2020", false},
		{"2020-03", false},
		{"2020-03-01", false},
		{"2020-03-01T10", true},
		{"2020-03-01T10:20", true},
		{"2020-03-01T10:20:30", true},
		{"2020-03-01T10:20:30.123", true},
	} {
		dt, err := fptypes.NewDateTime(tt.text)
		if err != nil {
			t.Fatalf("%s: %v", tt.text, err)
		}
		for _, offset := range []time.Duration{0, -5 * time.Hour, 5*time.Hour + 30*time.Minute, 14 * time.Hour} {
			got := WithDefaultOffset(dt, offset)
			minutes, ok := got.EffectiveOffset()
			if ok != tt.placed || (ok && time.Duration(minutes)*time.Minute != offset) {
				t.Errorf("WithDefaultOffset(%s, %s): offset %d minutes, placed %v; want placed %v",
					tt.text, offset, minutes, ok, tt.placed)
			}
			if got.HasTZ() {
				t.Errorf("WithDefaultOffset(%s, %s) wrote an offset into the value", tt.text, offset)
			}
		}
	}
}

func TestWithDefaultOffsetLeavesAWrittenOffsetAlone(t *testing.T) {
	for _, text := range []string{"2020-03-01T10:20:30+07:00", "2020-03-01T10:20:30.123-03:30", "2020-03-01T10Z"} {
		dt, err := fptypes.NewDateTime(text)
		if err != nil {
			t.Fatalf("%s: %v", text, err)
		}
		want, _ := dt.EffectiveOffset()
		got, ok := WithDefaultOffset(dt, 9*time.Hour).EffectiveOffset()
		if !ok || got != want {
			t.Errorf("WithDefaultOffset(%s, 9h) is at %d minutes, want its own %d", text, got, want)
		}
	}
}
