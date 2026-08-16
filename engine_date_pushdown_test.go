package cql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofhir/cql/eval"
)

const datePushdownLibrary = `library T version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
parameter MP Interval<DateTime> default Interval[@2020-01-01, @2020-12-31]

context Patient

define InPeriod: [Encounter] E where E.period during MP return E.id
`

// encounters spans the interval boundary: two inside 2020, two outside.
var encounters = []json.RawMessage{
	json.RawMessage(`{"resourceType":"Encounter","id":"e1","period":{"start":"2020-03-01","end":"2020-03-05"}}`),
	json.RawMessage(`{"resourceType":"Encounter","id":"e2","period":{"start":"2019-06-01","end":"2019-06-02"}}`),
	json.RawMessage(`{"resourceType":"Encounter","id":"e3","period":{"start":"2020-11-01","end":"2020-11-02"}}`),
	json.RawMessage(`{"resourceType":"Encounter","id":"e4","period":{"start":"2021-02-01","end":"2021-02-03"}}`),
}

// datePushdownProvider answers a retrieve either honoring the pushed-down range
// or ignoring it, which is the difference the equivalence test turns on.
type datePushdownProvider struct {
	honor bool
	last  eval.RetrieveRequest
	rows  int
}

func (p *datePushdownProvider) Retrieve(_ context.Context, req eval.RetrieveRequest) ([]json.RawMessage, error) {
	p.last = req
	if !p.honor || req.DateRange == nil {
		p.rows = len(encounters)
		return encounters, nil
	}
	// Standing in for a provider that translated the range into a date search:
	// return the rows inside 2020 and leave the rest in the database.
	inside := []json.RawMessage{encounters[0], encounters[2]}
	p.rows = len(inside)
	return inside, nil
}

// TestDateRangeReachesTheProvider covers the filter a provider never used to
// get from CQL text. Only the ELM importer filled DateRange in, so every
// temporal filter in a library evaluated from text was resolved in memory after
// fetching up to the row limit.
func TestDateRangeReachesTheProvider(t *testing.T) {
	provider := &datePushdownProvider{honor: true}
	if _, err := NewEngine(WithDataProvider(provider)).EvaluateExpression(
		context.Background(), datePushdownLibrary, "InPeriod",
		[]byte(`{"resourceType":"Patient","id":"p1"}`), nil); err != nil {
		t.Fatalf("evaluating: %v", err)
	}
	if provider.last.DateRange == nil {
		t.Error("the retrieve carried no date range")
	}
	// An interval without the element it applies to is not a filter anyone can
	// run. The AST carried the path and the request had no field for it.
	if provider.last.DatePath != "period" {
		t.Errorf("date path = %q, want period", provider.last.DatePath)
	}
}

// TestPushdownIsInvisibleInTheResult is what makes pushing a filter down safe.
// A provider may ignore DateRange, and the FHIR date search it most likely maps
// to answers with an overlap, which returns more than `during` means. The query
// keeps its where clause, so both providers must produce the same answer.
func TestPushdownIsInvisibleInTheResult(t *testing.T) {
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)

	honoring := &datePushdownProvider{honor: true}
	fromHonoring, err := NewEngine(WithDataProvider(honoring)).EvaluateExpression(
		context.Background(), datePushdownLibrary, "InPeriod", patient, nil)
	if err != nil {
		t.Fatalf("honoring provider: %v", err)
	}

	ignoring := &datePushdownProvider{honor: false}
	fromIgnoring, err := NewEngine(WithDataProvider(ignoring)).EvaluateExpression(
		context.Background(), datePushdownLibrary, "InPeriod", patient, nil)
	if err != nil {
		t.Fatalf("ignoring provider: %v", err)
	}

	if valueString(fromHonoring) != valueString(fromIgnoring) {
		t.Errorf("honoring gave %s, ignoring gave %s — the pushdown changed the answer",
			valueString(fromHonoring), valueString(fromIgnoring))
	}
	// And the point of the exercise: the honoring provider did less work.
	if honoring.rows >= ignoring.rows {
		t.Errorf("honoring provider returned %d rows, ignoring returned %d — no work was saved",
			honoring.rows, ignoring.rows)
	}
	// Both must have found the two encounters inside 2020, not all four.
	if got := valueString(fromHonoring); got != "{e1, e3}" {
		t.Errorf("result = %s, want {e1, e3}", got)
	}
}

// TestPushdownRescuesAPopulationThatWouldNotFit covers what this is worth
// beyond speed. Without a pushed-down range the retrieve returns every row and
// exceeds MaxRetrieveSize, so the evaluation fails before any filtering
// happens: the same library either answers or does not, depending on whether
// the provider can narrow the query.
func TestPushdownRescuesAPopulationThatWouldNotFit(t *testing.T) {
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)
	// Three rows allowed; the unfiltered retrieve has four.
	opts := []Option{WithMaxRetrieveSize(3)}

	honoring := &datePushdownProvider{honor: true}
	got, err := NewEngine(append(opts, WithDataProvider(honoring))...).EvaluateExpression(
		context.Background(), datePushdownLibrary, "InPeriod", patient, nil)
	if err != nil {
		t.Fatalf("honoring provider: %v", err)
	}
	if s := valueString(got); s != "{e1, e3}" {
		t.Errorf("result = %s, want {e1, e3}", s)
	}

	ignoring := &datePushdownProvider{honor: false}
	if _, err := NewEngine(append(opts, WithDataProvider(ignoring))...).EvaluateExpression(
		context.Background(), datePushdownLibrary, "InPeriod", patient, nil); err == nil {
		t.Error("the unfiltered retrieve fit within the limit, want it refused")
	}
}
