package funcs

import (
	"errors"
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"
)

// TestDateAdd_UCUMMatchesCalendarKeyword pins the equivalence that was silently
// broken: every UCUM duration of a week or less must shift a DateTime exactly as
// the calendar keyword it converts to.
func TestDateAdd_UCUMMatchesCalendarKeyword(t *testing.T) {
	cases := []struct {
		ucum    string
		keyword string
	}{
		{"wk", "week"},
		{"d", "day"},
		{"h", "hour"},
		{"min", "minute"},
		{"s", "second"},
		{"ms", "millisecond"},
	}

	base, err := fptypes.NewDateTime("2020-01-01T00:00:00.000")
	if err != nil {
		t.Fatalf("NewDateTime: %v", err)
	}

	for _, tc := range cases {
		t.Run(tc.ucum, func(t *testing.T) {
			viaUCUM, err := DateAdd(base, 5, tc.ucum)
			if err != nil {
				t.Fatalf("DateAdd with %q: %v", tc.ucum, err)
			}
			viaKeyword, err := DateAdd(base, 5, tc.keyword)
			if err != nil {
				t.Fatalf("DateAdd with %q: %v", tc.keyword, err)
			}
			if !viaUCUM.Equal(viaKeyword) {
				t.Errorf("5 %q gave %v, want %v (as 5 %q)", tc.ucum, viaUCUM, viaKeyword, tc.keyword)
			}
			// The operand must actually have moved; returning it unchanged was the bug
			if viaUCUM.Equal(base) {
				t.Errorf("5 %q left the operand unchanged at %v", tc.ucum, viaUCUM)
			}
		})
	}
}

// TestDateAdd_UCUMCalendarUnitsRefused covers the two UCUM codes that have no exact
// calendar meaning. Silently treating them as a no-op is what made a wrong measure
// look like a working one, so they must report rather than return a value.
func TestDateAdd_UCUMCalendarUnitsRefused(t *testing.T) {
	base, err := fptypes.NewDateTime("2020-01-01T00:00:00")
	if err != nil {
		t.Fatalf("NewDateTime: %v", err)
	}

	for _, unit := range []string{"a", "mo"} {
		t.Run(unit, func(t *testing.T) {
			result, err := DateAdd(base, 5, unit)
			if err == nil {
				t.Fatalf("5 %q returned %v, want an error", unit, result)
			}
			if !errors.Is(err, fptypes.ErrCalendarConversionRequired) {
				t.Errorf("5 %q gave %v, want ErrCalendarConversionRequired", unit, err)
			}
			if result != nil {
				t.Errorf("5 %q returned %v alongside its error, want nil", unit, result)
			}
		})
	}
}

// TestDateAdd_UnknownUnitReported guards the other half of the silent no-op: a unit
// that is neither a keyword nor a UCUM code must not pass for a successful shift.
func TestDateAdd_UnknownUnitReported(t *testing.T) {
	base, err := fptypes.NewDateTime("2020-01-01T00:00:00")
	if err != nil {
		t.Fatalf("NewDateTime: %v", err)
	}
	if result, err := DateAdd(base, 5, "furlong"); err == nil {
		t.Errorf("an unknown unit returned %v, want an error", result)
	}
}

// TestDateAdd_ClampsToEndOfMonth covers the day clamp that moved into fptypes when
// the two-value AddDuration landed: adding a year to a leap day gives the 28th.
func TestDateAdd_ClampsToEndOfMonth(t *testing.T) {
	cases := []struct {
		name   string
		start  string
		amount int
		unit   string
		want   string
	}{
		{"leap day plus one year", "2012-02-29", 1, "year", "2013-02-28"},
		{"end of january plus one month", "2020-01-31", 1, "month", "2020-02-29"},
		{"leap day minus one year", "2012-02-29", -1, "year", "2011-02-28"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start, err := fptypes.NewDate(tc.start)
			if err != nil {
				t.Fatalf("NewDate: %v", err)
			}
			got, err := DateAdd(start, tc.amount, tc.unit)
			if err != nil {
				t.Fatalf("DateAdd: %v", err)
			}
			want, err := fptypes.NewDate(tc.want)
			if err != nil {
				t.Fatalf("NewDate: %v", err)
			}
			if !got.Equal(want) {
				t.Errorf("%s %+d %s = %v, want %v", tc.start, tc.amount, tc.unit, got, tc.want)
			}
		})
	}
}
