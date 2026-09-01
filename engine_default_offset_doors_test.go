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
		// Naming the type a value already has is not a door, and these two say so
		// where it is easiest to get wrong: a conversion that converts nothing.
		// Placing the result of one placed whatever it was handed.
		{
			"a cast that converts nothing", "timezoneoffset from (cast (maximum DateTime) as DateTime)",
			"`cast X as DateTime` where X is already a DateTime produces X — the door is a " +
				"conversion that makes a DateTime out of something else, not the naming of a type",
		},
		{
			"a convert that converts nothing", "timezoneoffset from (convert (maximum DateTime) to DateTime)",
			"same, through the other spelling",
		},
	} {
		for _, hours := range []int{0, -5, 9, 14, -11} {
			if got := askAtOffset(t, hours, tt.expr); got != "null" {
				t.Errorf("%s answers %s at UTC%+d, want null — %s", tt.what, got, hours, tt.why)
			}
		}
	}

	// What the boundary case costs if it goes wrong is not the offset but the
	// value: a placed maximum is no longer equal to the maximum.
	for _, hours := range []int{0, 14, -11} {
		if got := askAtOffset(t, hours, "maximum DateTime = (cast (maximum DateTime) as DateTime)"); got != "true" {
			t.Errorf("maximum DateTime is not equal to itself cast to its own type at UTC%+d: %s", hours, got)
		}
	}
}

// TestBoundaryKeepsWhatTheValueStates covers a boundary of a value that writes its
// own offset, which aborted the expression rather than answering:
//
//	HighBoundary(@2020-06-15T23:00:00Z, 17)
//	  invalid datetime format: 2020-06-15T23:00:00Z999
//
// The fill counts digits and appends more of them, and the offset sat at the end
// being counted as precision. It is not precision — it says where the value sits,
// not how finely it is stated — so it comes off before the fill and back on after.
//
// A FHIR dateTime that carries a time must carry an offset, so this is the shape
// every boundary over served data has.
func TestBoundaryKeepsWhatTheValueStates(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"HighBoundary(@2020-06-15T23:00:00Z, 17)", "2020-06-15T23:00:00.999Z"},
		{"LowBoundary(@2020-06-15T23:00:00Z, 17)", "2020-06-15T23:00:00.000Z"},
		// An offset that is not whole hours, since the fill is digit-counting and
		// this one has the most digits to miscount.
		{"HighBoundary(@2020-06-15T23:00:00+05:30, 17)", "2020-06-15T23:00:00.999+05:30"},
		{"timezoneoffset from HighBoundary(@2020-06-15T23:00:00Z, 17)", "0"},
		// Unchanged by any of this: no stated offset, and the non-temporal path.
		{"HighBoundary(@2020-06-15T23, 17)", "2020-06-15T23:59:59.999"},
		{"HighBoundary(1.5, 3)", "1.599"},
	} {
		for _, hours := range []int{0, -5, 14} {
			if got := askAtOffset(t, hours, tt.expr); got != tt.want {
				t.Errorf("%s = %s at UTC%+d, want %s", tt.expr, got, hours, tt.want)
			}
		}
	}
}
