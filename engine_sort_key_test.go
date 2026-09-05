package cql

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gofhir/cql/eval"
)

// sortKeyProvider serves two observations in the wrong order, so a test can tell
// a sort that ran from one that merely did not fail.
type sortKeyProvider struct{ empty bool }

func (p sortKeyProvider) Retrieve(_ context.Context, r eval.RetrieveRequest) ([]json.RawMessage, error) {
	if p.empty || r.ResourceType != "Observation" {
		return nil, nil
	}
	return []json.RawMessage{
		json.RawMessage(`{"resourceType":"Observation","id":"june","status":"final",` +
			`"effectiveDateTime":"2019-06-01T10:00:00Z","valueQuantity":{"value":80,"unit":"mg/dL"}}`),
		json.RawMessage(`{"resourceType":"Observation","id":"march","status":"final",` +
			`"effectiveDateTime":"2019-03-01T10:00:00Z","valueQuantity":{"value":60,"unit":"mg/dL"}}`),
	}, nil
}

func evalWithObservations(t *testing.T, p sortKeyProvider, expr string) string {
	t.Helper()
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\n" +
		"include FHIRHelpers version '4.0.1' called FH\ncontext Patient\ndefine A: " + expr + "\n"
	got, err := NewEngine(WithDataProvider(p), WithEvaluationTimestamp(
		time.Date(2019, 6, 1, 12, 0, 0, 0, time.UTC))).
		EvaluateExpression(context.Background(), src, "A",
			[]byte(`{"resourceType":"Patient","id":"p1"}`), nil)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	if got == nil {
		return "null"
	}
	return got.String()
}

// TestASortKeyNamesTheSameElementsAMemberAccessDoes covers a sort key that named
// a real element and was rejected as a mistake.
//
// FHIR stores a choice element under its type: Observation.effective is written
// effectiveDateTime or effectivePeriod, never `effective`. The engine resolves that
// for a member access — and did not for a sort key, which read the name as written
// and concluded it named nothing:
//
//	where O.effective is not null    2
//	sort by effective                error: unknown sort key "effective"
//
// One name, two answers, because only one of the two paths asked the model. The
// resolution now lives in one place that both call.
//
// `sort by effective` appears in the published measures — FHIR347 sorts LDL results
// by it to take the most recent — and 29 sort clauses appear across five of the 19
// libraries.
func TestASortKeyNamesTheSameElementsAMemberAccessDoes(t *testing.T) {
	p := sortKeyProvider{}

	// The member access and the sort key have to agree that the name exists.
	if got := evalWithObservations(t, p, "Count([Observation] O where O.effective is not null)"); got != "2" {
		t.Fatalf("member access on the choice element = %s, want 2 — this test's premise", got)
	}

	// And the sort has to have run, not merely not failed: the provider hands them
	// back June first.
	for _, tt := range []struct{ expr, want string }{
		{"First([Observation] O sort by effective).id", "march"},
		{"Last([Observation] O sort by effective).id", "june"},
		{"First([Observation] O sort by effective desc).id", "june"},
		// The concrete spelling was already working and must stay that way.
		{"Last([Observation] O sort by effectiveDateTime).id", "june"},
		// As must a plain element that is not a choice at all.
		{"Count(([Observation] O sort by status))", "2"},
	} {
		if got := evalWithObservations(t, p, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}

	// A key that names nothing is still a mistake worth reporting.
	if got := evalWithObservations(t, p, "Count(([Observation] O sort by noSuchElement))"); got[:5] != "ERROR" {
		t.Errorf("a sort key naming nothing = %s, want an error", got)
	}
}

// TestSortingNothingIsNotAMistake covers what happens when the query returns no
// rows at all, which for a measure is the ordinary case rather than an unusual one.
//
// Whether a sort key names a column is decided by looking at the rows. With none of
// them every key looks invented, so the query failed outright:
//
//	[Observation] O sort by status     error, for a patient with no observations
//
// A patient without that kind of data is exactly who a measure asks about, and the
// whole evaluation aborted rather than answering that they have none. Sorting an
// empty result is trivially done, and there is nothing there to judge a key by.
func TestSortingNothingIsNotAMistake(t *testing.T) {
	empty := sortKeyProvider{empty: true}

	for _, expr := range []string{
		"Count(([Observation] O sort by effective))",
		"Count(([Observation] O sort by status))",
		"Count(([Observation] O sort by effectiveDateTime))",
		"Count(([Observation] O sort by effective desc))",
	} {
		if got := evalWithObservations(t, empty, expr); got != "0" {
			t.Errorf("%s over no rows = %s, want 0", expr, got)
		}
	}

	// The name is still checked — by the semantic phase, which does not need rows
	// to know that nothing declares it.
	if got := evalWithObservations(t, empty, "Count(([Observation] O sort by noSuchElement))"); got[:5] != "ERROR" {
		t.Errorf("a key naming nothing = %s, want an error even with no rows", got)
	}
}
