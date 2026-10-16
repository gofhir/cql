package cql

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gofhir/cql/eval"
)

// TestProperlyIncludesAtADayReadsTheDaysAsWritten covers the one path to the
// as-written rule the runtime probe does not reach: `properly includes day of`
// a point, which asks at the stated precision only when the comparison at full
// precision cannot be made.
//
// Here it cannot: the point is written to the hour, and 01:00+02:00 is the
// same hour as the low bound's 23:30Z. Read as written, the bound is on
// 2019-12-31 and the point on 2020-01-01, so the point is properly inside —
// the Java engine answers true. Moved to UTC, both were on the 31st and this
// engine answered false. cql-execution answers null, so the row is pinned here
// rather than in runtime.cql, where the two references would dispute it.
func TestProperlyIncludesAtADayReadsTheDaysAsWritten(t *testing.T) {
	src := "library T version '1.0'\n" +
		"define R: Interval[@2019-12-31T23:30:00Z, @2020-01-05T00:00:00Z] properly includes day of @2020-01-01T01+02:00\n"
	v, err := NewEngine().EvaluateExpression(context.Background(), src, "R", nil, nil)
	if err != nil || valueString(v) != "true" {
		t.Errorf("properly includes day of, across offsets = %v, %v; want true", v, err)
	}
}

type unwrittenOffsetProvider struct{}

func (unwrittenOffsetProvider) Retrieve(_ context.Context, r eval.RetrieveRequest) ([]json.RawMessage, error) {
	if r.ResourceType != "Observation" {
		return nil, nil
	}
	return []json.RawMessage{json.RawMessage(`{"resourceType":"Observation","id":"o","status":"final",` +
		`"code":{"text":"x"},"effectiveDateTime":"2020-03-05T23:00:00"}`)}, nil
}

// TestADayFromDataIsTheDayItWasWrittenOn covers a FHIR dateTime written
// without an offset, compared at the day with one that has one.
//
// It is placed at the request's offset, and at the hour and finer that is the
// instant it names. At the day it is the day written: 23:00 on the 5th is the
// 5th. Moved to UTC with the request at -03:00 it was the 6th, and `same day
// as` the 5th was false — whatever the process's own zone, since the day as
// written does not depend on it.
func TestADayFromDataIsTheDayItWasWrittenOn(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1' called FH\n" +
		"context Patient\n" +
		"define R: First([Observation] O return (O.effective as dateTime) same day as @2020-03-05T00:00:00Z)\n"
	e := NewEngine(WithDataProvider(unwrittenOffsetProvider{}),
		WithEvaluationTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", -3*3600))))
	v, err := e.EvaluateExpression(context.Background(), src, "R", []byte(`{"resourceType":"Patient","id":"p"}`), nil)
	if err != nil || valueString(v) != "true" {
		t.Errorf("a dateTime from data without an offset, same day as = %v, %v; want true", v, err)
	}
}
