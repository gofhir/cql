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
// The list is now closed, which is what let the engine move to fhirpath v1.9.1.
// Temporal arithmetic was the last one open and upstream closed it there; `convert
// to DateTime` was found by enumerating the fifteen places the engine builds a
// DateTime rather than by a failing run, and was answering differently from
// ToDateTime about the same string.
func TestEveryDoorPlacesADateTime(t *testing.T) {
	for _, tt := range []struct{ door, expr string }{
		{"a CQL literal", "timezoneoffset from @2020-06-15T23:00:00"},
		{"FHIR JSON", "First([Encounter] E return timezoneoffset from E.period.start)"},
		{"a string conversion", "timezoneoffset from ToDateTime('2020-06-15T23:00:00')"},
		{"the DateTime constructor", "timezoneoffset from DateTime(2020, 6, 15, 23, 0, 0, 0)"},
		{"a convert expression", "timezoneoffset from (convert '2020-06-15T23:00:00' to DateTime)"},
		{"temporal arithmetic", "timezoneoffset from (@2020-06-15T23:00:00 + 1 day)"},
		{"a derived boundary", "timezoneoffset from HighBoundary(@2020-06-15T23, 17)"},
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

// TestWhatIsNotADoor holds the other end of the same rule, because "place every
// DateTime" taken literally gives wrong answers in three places, and each was
// checked rather than assumed.
//
// It is the counterpart of TestEveryDoorPlacesADateTime and exists for the same
// reason: the two populations of value cost this engine several releases, and the
// way not to repeat that is to write down where the line falls instead of
// rediscovering it from a failing run.
func TestWhatIsNotADoor(t *testing.T) {
	for _, tt := range []struct{ what, expr, why string }{
		{
			"the maximum of the type", "timezoneoffset from maximum DateTime",
			"9999-12-31T23:59:59.999 is the largest value the type holds, not a value someone " +
				"wrote without an offset; placing it at a western offset would put it past the " +
				"end of the range it defines",
		},
		{
			"the minimum of the type", "timezoneoffset from minimum DateTime",
			"same, at the other end",
		},
		{
			"a DateTime no finer than the day", "timezoneoffset from ToDateTime(@2020-06-15)",
			"a value with no hour names a day rather than an instant, so there is nothing for " +
				"an offset to move — the rule framelessTemporal applies, and fhirpath reports no " +
				"effective offset for one either",
		},
	} {
		for _, hours := range []int{0, -5, 9} {
			if got := askAtOffset(t, hours, tt.expr); got != "null" {
				t.Errorf("%s answers %s at UTC%+d, want null — %s", tt.what, got, hours, tt.why)
			}
		}
	}
}
