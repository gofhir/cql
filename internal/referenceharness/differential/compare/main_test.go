package main

import "testing"

// TestTheEngineIsAtTheOffsetOfTheArea holds the comparison to the offset the
// area is evaluated at by the references (internal/referenceharness/evaloffset):
// the engine of a corpus at +05:30 answers 5.5 to the offset of a DateTime with
// none, and the others -3 (plan 024).
func TestTheEngineIsAtTheOffsetOfTheArea(t *testing.T) {
	const library = "library T version '1.0'\ndefine D1: timezoneoffset from @2020-01-01T00:10:00\n"
	for area, want := range map[string]string{"fractionaloffset": "5.5", "mixeddates": "-3"} {
		engine, patient := newEngine("", area)
		if got := engineAnswer(engine, library, "D1", patient).Text; got != "Decimal "+want {
			t.Errorf("%s: timezoneoffset = %q, want Decimal %s", area, got, want)
		}
	}
}
