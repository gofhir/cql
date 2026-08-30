package cql

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gofhir/cql/eval"
)

// doorProvider serves a period whose start carries a time but no offset. FHIR
// requires one, so this is non-conforming JSON — and a server can send it, which
// is why the engine has to decide what such a value means.
type doorProvider struct{}

func (doorProvider) Retrieve(_ context.Context, r eval.RetrieveRequest) ([]json.RawMessage, error) {
	if r.ResourceType != "Encounter" {
		return nil, nil
	}
	return []json.RawMessage{json.RawMessage(
		`{"resourceType":"Encounter","id":"e1","status":"finished",` +
			`"period":{"start":"2020-06-15T23:00:00","end":"2020-06-16T23:00:00"}}`)}, nil
}

func askAtOffset(t *testing.T, hours int, expr string) string {
	t.Helper()
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\n" +
		"include FHIRHelpers version '4.0.1' called FH\ncontext Patient\ndefine A: " + expr + "\n"
	got, err := NewEngine(WithDataProvider(doorProvider{}), WithEvaluationTimestamp(
		time.Date(2020, 6, 1, 12, 0, 0, 0, time.FixedZone("", hours*3600)))).
		EvaluateExpression(context.Background(), src, "A",
			[]byte(`{"resourceType":"Patient","id":"p1"}`), nil)
	if err != nil {
		return "ERROR"
	}
	if got == nil {
		return "null"
	}
	return valueString(got)
}

// TestEveryDoorPlacesADateTime enumerates the ways a DateTime enters the engine
// and asserts that each one carries the evaluation request's offset, because CQL
// says a value written without one assumes it.
//
// The test is by door rather than by symptom deliberately. Applying the default
// to literals alone left two populations of value — placed and unplaced — and
// they compare correctly only while the comparison ignores the default. fhirpath
// did until v1.9.1 and no longer does, so the split is a real defect rather than
// a tidiness question, and the way to find the next open door is to have written
// the list down.
//
// Not all of them are closed. The engine builds a DateTime in fifteen places; the
// doors below are the three that data actually arrives through. What remains is
// recorded in the test that follows, not left to be rediscovered.
func TestEveryDoorPlacesADateTime(t *testing.T) {
	for _, tt := range []struct{ door, expr string }{
		{"a CQL literal", "timezoneoffset from @2020-06-15T23:00:00"},
		{"FHIR JSON", "First([Encounter] E return timezoneoffset from E.period.start)"},
		{"a string conversion", "timezoneoffset from ToDateTime('2020-06-15T23:00:00')"},
	} {
		for _, tz := range []struct {
			hours int
			want  string
		}{
			{0, "0"}, {-5, "-5"}, {9, "9"},
		} {
			if got := askAtOffset(t, tz.hours, tt.expr); got != tz.want {
				t.Errorf("%s at UTC%+d = %s, want %s — every door places a DateTime",
					tt.door, tz.hours, got, tz.want)
			}
		}
	}

	// And a value that states its own offset keeps it, whichever door it came
	// through.
	if got := askAtOffset(t, -5, "timezoneoffset from @2020-06-15T23:00:00+07:00"); got != "7" {
		t.Errorf("a stated offset = %s, want 7", got)
	}
}

// TestDoorsStillOpen records what is not done, measured rather than guessed, so
// the next person starts from the list instead of from a failing conformance run.
//
// Closing these is what unblocks fhirpath v1.9.1. On v1.9.0 the engine holds at
// 2084 conformance cases because that version's equality ignores the default; on
// v1.9.1 it drops to 2069, and closing doors one at a time moves the failures
// around rather than reducing them — 15 with none closed, 19 with three, 43 with
// the corpus parser as well. They only cancel once every door is shut.
func TestDoorsStillOpen(t *testing.T) {
	for _, tt := range []struct{ door, expr string }{
		{"the DateTime constructor", "timezoneoffset from DateTime(2020, 6, 15, 23, 0, 0, 0)"},
		{"temporal arithmetic", "timezoneoffset from (@2020-06-15T23:00:00 + 1 day)"},
	} {
		got := askAtOffset(t, -5, tt.expr)
		if got != "null" {
			t.Errorf("%s now answers %s at UTC-5 — if this door is closed, move it "+
				"into TestEveryDoorPlacesADateTime and re-measure v1.9.1", tt.door, got)
		}
	}
}
