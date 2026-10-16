package cql

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/eval"
	cqltypes "github.com/gofhir/cql/types"
)

// TestPhrasesAtAPrecisionWhereTheReferencesDisagree pins interval phrases at a
// precision measured against the Java engine, whose answers are the ones the
// specification gives. Most cannot go in runtime.cql because cql-execution
// answers otherwise — null for the `properly` and `starts` rows, true for the
// `overlaps before` and `after` ones — and the two references would dispute
// them; the open-ended row they agree on is kept with its neighbors.
//
// `properly includes day of` a point asks for the point strictly inside at the
// day, so a point on the start day is not. It was a retry the full comparison
// never needed for two fully specified DateTimes, and answered with the
// instants. `starts day of` with a start written to the month is unknown at the
// day, but the end is already past, and unknown and false is false. A closed
// null bound is the minimum or maximum of the point type, so a point inside an
// open-ended period is properly inside it, and two intervals starting at the
// minimum start together.
func TestPhrasesAtAPrecisionWhereTheReferencesDisagree(t *testing.T) {
	for expr, want := range map[string]string{
		"Interval[@2020-01-01T10:00:00Z, @2020-01-03T00:00:00Z] properly includes day of @2020-01-01T12:00:00Z":              "false",
		"@2020-01-02T12:00:00Z properly included in day of Interval[@2020-01-01T10:00:00Z, @2020-01-03T00:00:00Z]":           "true",
		"@2020-01-01T23:00:00Z properly included in day of Interval[@2020-01-02T00:00:00+02:00, @2020-01-02T05:00:00+02:00]": "false",
		"Interval[@2020-01, @2020-03] starts day of Interval[@2020-01-01T00:00:00Z, @2020-02-01T00:00:00Z]":                  "false",
		// A closed null bound is the minimum or maximum of the point type, and an
		// open one is unknown.
		"Interval[null, @2020-12-31T00:00:00Z] properly includes day of @2020-06-01T00:00:00Z":      "true",
		"Interval[@2020-01-01T00:00:00Z, null] properly includes day of @2020-06-01T00:00:00Z":      "true",
		"Interval[@2020-01-01T00:00:00Z, null) properly includes day of @2020-06-01T00:00:00Z":      "null",
		"Interval[null, @2020-01-05T00:00:00Z] starts day of Interval[null, @2020-02-01T00:00:00Z]": "true",
		// The extreme is the minimum value itself, so it is the same day as a
		// written minimum DateTime.
		"Interval[null, @2020-01-05T00:00:00Z] starts day of Interval[@0001-01-01T00:00:00.000Z, @2020-01-05T00:00:00Z]": "true",
		// overlaps before is overlaps and starting first, so an open end of B
		// the overlap depends on leaves it unknown; cql-execution answers true.
		"Interval[@2020-01-01T00:00:00Z, @2020-01-10T00:00:00Z] overlaps before day of Interval[@2020-01-05T00:00:00Z, null)": "null",
		"Interval[@2020-01-05T00:00:00Z, @2020-01-20T00:00:00Z] overlaps after day of Interval(null, @2020-01-10T00:00:00Z]":  "null",
	} {
		src := "library T version '1.0'\ndefine R: " + expr + "\n"
		v, err := NewEngine().EvaluateExpression(context.Background(), src, "R", nil, nil)
		got := "null"
		if v != nil {
			got = valueString(v)
		}
		if err != nil || got != want {
			t.Errorf("%s = %s, %v; want %s", expr, got, err, want)
		}
	}
}

// TestPhrasesAtAPrecisionKeepTheirTypeErrors covers operands a precision does
// not apply to. Read component by component they compared a Date's year with
// the zero in a Time's year slot and answered; the reference translator rejects
// both, and without a precision this engine raises a type error, which it still
// does.
func TestPhrasesAtAPrecisionKeepTheirTypeErrors(t *testing.T) {
	for _, expr := range []string{
		"Interval[@2020-01-01, @2020-01-05] before day of Interval[@T10:00, @T11:00]",
		"Interval[1, 5] properly includes day of @2020-01-01T00:00:00Z",
	} {
		src := "library T version '1.0'\ndefine R: " + expr + "\n"
		if v, err := NewEngine().EvaluateExpression(context.Background(), src, "R", nil, nil); err == nil {
			t.Errorf("%s = %v, want a type error", expr, v)
		}
	}
}

// instantRangeProvider honors a pushed-down date range the way a FHIR server's
// date search does: by instant, keeping the encounters whose period overlaps it.
type instantRangeProvider struct {
	seen      *cqltypes.Interval
	precision *string
}

func (p instantRangeProvider) Retrieve(_ context.Context, r eval.RetrieveRequest) ([]json.RawMessage, error) {
	if r.ResourceType != "Encounter" {
		return nil, nil
	}
	start, end := time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC), time.Date(2020, 1, 1, 2, 0, 0, 0, time.UTC)
	*p.precision = r.DatePrecision
	if iv, ok := r.DateRange.(cqltypes.Interval); ok {
		*p.seen = iv
		if low, ok := iv.Low.(fptypes.DateTime); ok && end.Before(low.ToTime()) {
			return nil, nil
		}
		if high, ok := iv.High.(fptypes.DateTime); ok && start.After(high.ToTime()) {
			return nil, nil
		}
	}
	return []json.RawMessage{json.RawMessage(`{"resourceType":"Encounter","id":"e","status":"finished",` +
		`"class":{"code":"AMB"},"period":{"start":"2020-01-01T01:00:00Z","end":"2020-01-01T02:00:00Z"}}`)}, nil
}

// TestAFilterAtADayKeepsTheRowsAProviderSearchesByInstant covers the filter a
// query pushes down to its retrieve when it compares at a precision.
//
// The encounter is on the 1st as written, the day the measurement period
// starts as written, so it is during day of the period. As instants it ends at
// 02:00Z and the period starts at 03:00Z: a provider searching by the period
// itself would not return it, and the measure would never see it. The range is
// widened before the provider gets it.
func TestAFilterAtADayKeepsTheRowsAProviderSearchesByInstant(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1' called FH\n" +
		"parameter \"Measurement Period\" Interval<DateTime> default Interval[@2020-01-01T00:00:00.000-03:00, @2020-12-31T23:59:59.999-03:00]\n" +
		"context Patient\n" +
		"define Encounters: [Encounter] E where E.period during day of \"Measurement Period\"\n" +
		"define R: Count(Encounters)\n"
	var seen cqltypes.Interval
	var precision string
	p := instantRangeProvider{seen: &seen, precision: &precision}
	e := NewEngine(WithDataProvider(p))
	v, err := e.EvaluateExpression(context.Background(), src, "R", []byte(`{"resourceType":"Patient","id":"p"}`), nil)
	if seen.Low == nil {
		t.Fatal("no date range reached the provider: the filter was not pushed down")
	}
	if *p.precision != "day" {
		t.Errorf("the provider was told the range was compared at %q, want day", *p.precision)
	}
	if err != nil || valueString(v) != "1" {
		t.Errorf("encounters during day of the period = %v, %v; want 1 (the range a provider saw: %v)", v, err, seen)
	}
}
