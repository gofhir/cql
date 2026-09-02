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

// TestARefinementTakesTheRequestsFrame covers the one derived value that had no
// offset to inherit.
//
// A derived value carries its source's offset rather than the request's, which is
// what keeps a value read from data from silently acquiring the offset of whoever
// asked. A boundary that refines a value with no hour is the case that rule does
// not reach: `HighBoundary(@2020-06-15T, 17)` turns a day into a millisecond, and
// a day has no offset to carry. The result named an instant while sitting in no
// frame at all, so comparing it to anything placed was unknown:
//
//	HighBoundary(@2020-06-15T, 17) < @2020-06-16T00:00:00     was null
//
// The request's offset is the only frame there is, and "the last instant of the
// 15th" is a question about the asking party's day. The rule still holds where it
// applies, which is what the second half of this test is for: a source that states
// its own offset keeps it, and does not pick up the request's.
func TestARefinementTakesTheRequestsFrame(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"timezoneoffset from HighBoundary(@2020-06-15T, 17)", "OFFSET"},
		{"timezoneoffset from LowBoundary(@2020-06-15T, 17)", "OFFSET"},
		{"HighBoundary(@2020-06-15T, 17) < @2020-06-16T00:00:00", "true"},
	} {
		for _, tz := range []struct {
			hours int
			want  string
		}{{0, "0"}, {-5, "-5"}, {9, "9"}} {
			want := tt.want
			if want == "OFFSET" {
				want = tz.want
			}
			if got := askAtOffset(t, tz.hours, tt.expr); got != want {
				t.Errorf("%s at UTC%+d = %s, want %s — a refinement with no frame to "+
					"inherit takes the request's", tt.expr, tz.hours, got, want)
			}
		}
	}

	// And where there IS a frame to inherit, it is inherited rather than replaced.
	for _, expr := range []string{
		"timezoneoffset from HighBoundary(@2020-06-15T23:00:00Z, 17)",
		"timezoneoffset from LowBoundary(@2020-06-15T23:00:00Z, 17)",
	} {
		for _, hours := range []int{0, -5, 9} {
			if got := askAtOffset(t, hours, expr); got != "0" {
				t.Errorf("%s at UTC%+d = %s, want 0 — a stated offset is the source's "+
					"frame and a derived value keeps it, whoever is asking", expr, hours, got)
			}
		}
	}
}

// TestBoundaryFillsARealLastDay covers the day a high boundary supplies for a
// value written only to the month.
//
// The fill takes missing components from the constant "9999-12-31T23:59:59.999",
// so the day it supplied was always the 31st:
//
//	HighBoundary(@2021-02T, 17)   was 2021-02-31T23:59:59.999
//
// February has no 31st, and neither do April, June, September or November. While
// such a value stayed unplaced the comparisons it reached declined, which hid it;
// once a boundary takes the request's offset the engine answers with it, and
// `days between @2021-02-01 and that` said 30 where the month has 27 left.
//
// The year and month are known from the value, and they determine the last day —
// so the calendar is asked rather than a table, which is what gets a leap year and
// the century rule right.
func TestBoundaryFillsARealLastDay(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"HighBoundary(@2021-02T, 17)", "2021-02-28T23:59:59.999"},
		{"HighBoundary(@2020-02T, 17)", "2020-02-29T23:59:59.999"}, // leap year
		{"HighBoundary(@2000-02T, 17)", "2000-02-29T23:59:59.999"}, // divisible by 400
		{"HighBoundary(@1900-02T, 17)", "1900-02-28T23:59:59.999"}, // divisible by 100
		{"HighBoundary(@2021-04T, 17)", "2021-04-30T23:59:59.999"},
		{"HighBoundary(@2021-12T, 17)", "2021-12-31T23:59:59.999"},
		// Only the year is written, so the month is filled too and the constant's
		// own December is the right answer.
		{"HighBoundary(@2021T, 17)", "2021-12-31T23:59:59.999"},
		// A day already written is not touched, and the low boundary is always the
		// 1st so it never had this problem.
		{"HighBoundary(@2021-06-15T, 17)", "2021-06-15T23:59:59.999"},
		{"LowBoundary(@2021-02T, 17)", "2021-02-01T00:00:00.000"},
		// A Date, which takes the same fill through a different kind.
		{"HighBoundary(@2021-02, 8)", "2021-02-28"},
		// What the malformed day cost: an arithmetic answer nobody could see was
		// wrong, since 2021-02-31 parses and prints.
		{"days between @2021-02-01T00:00:00.000 and HighBoundary(@2021-02T, 17)", "27"},
	} {
		for _, hours := range []int{0, -5, 14} {
			if got := askAtOffset(t, hours, tt.expr); got != tt.want {
				t.Errorf("%s = %s at UTC%+d, want %s", tt.expr, got, hours, tt.want)
			}
		}
	}
}
