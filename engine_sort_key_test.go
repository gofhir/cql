package cql

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gofhir/cql/eval"
)

// sortKeyProvider serves two observations in the wrong order, so a test can tell
// a sort that ran from one that merely did not fail.
type sortKeyProvider struct {
	empty bool
	// withoutEffective serves a resource that omits the element being sorted by,
	// which FHIR permits and a measure meets constantly.
	withoutEffective bool
	// withUndated adds one of those alongside dated ones, which is what decides
	// where a missing key lands in the order.
	withUndated bool
	// periods serves the other branch of the choice: an effective[x] that is a
	// Period, which has no ordering of its own.
	periods bool
}

func (p sortKeyProvider) Retrieve(_ context.Context, r eval.RetrieveRequest) ([]json.RawMessage, error) {
	if p.empty || r.ResourceType != "Observation" {
		return nil, nil
	}
	if p.periods {
		return []json.RawMessage{
			json.RawMessage(`{"resourceType":"Observation","id":"earlier","status":"final",` +
				`"effectivePeriod":{"start":"2019-03-01T10:00:00Z","end":"2019-03-01T11:00:00Z"}}`),
			json.RawMessage(`{"resourceType":"Observation","id":"later","status":"final",` +
				`"effectivePeriod":{"start":"2019-06-01T10:00:00Z","end":"2019-06-01T11:00:00Z"}}`),
		}, nil
	}
	if p.withUndated {
		return []json.RawMessage{
			json.RawMessage(`{"resourceType":"Observation","id":"undated","status":"final"}`),
			json.RawMessage(`{"resourceType":"Observation","id":"june","status":"final",` +
				`"effectiveDateTime":"2019-06-01T10:00:00Z"}`),
			json.RawMessage(`{"resourceType":"Observation","id":"march","status":"final",` +
				`"effectiveDateTime":"2019-03-01T10:00:00Z"}`),
		}, nil
	}
	if p.withoutEffective {
		return []json.RawMessage{
			json.RawMessage(`{"resourceType":"Observation","id":"undated","status":"final",` +
				`"valueQuantity":{"value":60,"unit":"mg/dL"}}`),
		}, nil
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
		// As must a plain element that is not a choice at all — asserted by
		// position, since Count would pass just as well if the sort quietly
		// stopped happening.
		{"First([Observation] O sort by id).id", "june"},
		{"Last([Observation] O sort by id).id", "march"},
	} {
		if got := evalWithObservations(t, p, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}

	// An element the type declares is a column whether or not these particular
	// resources carry it. FHIR makes almost all of them optional, so a sort key
	// naming one has to survive the rows that lack it — which is the case a measure
	// meets constantly.
	if got := evalWithObservations(t, sortKeyProvider{withoutEffective: true},
		"Count(([Observation] O sort by effective))"); got != "1" {
		t.Errorf("sorting by an element this resource does not carry = %s, want 1", got)
	}

	// A key that names nothing is still a mistake worth reporting.
	if got := evalWithObservations(t, p, "Count(([Observation] O sort by noSuchElement))"); !strings.HasPrefix(got, "ERROR") {
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

	// The name is still checked here — by the semantic phase, which does not need
	// rows to know that Observation declares no such element.
	//
	// That is not a guarantee for every query: where the semantic phase cannot type
	// the rows, an invented key over no rows goes unreported. Nothing is misordered
	// by it — there is nothing to order — so the cost is a missing diagnostic rather
	// than a wrong answer, and it is not worth failing a query that has no rows to
	// judge the key against.
	if got := evalWithObservations(t, empty, "Count(([Observation] O sort by noSuchElement))"); !strings.HasPrefix(got, "ERROR") {
		t.Errorf("a key naming nothing = %s, want an error even with no rows", got)
	}
}

// TestASortPutsMissingKeysLow covers where a row without the sort element lands,
// which decides what "the most recent one" means.
//
// CQL: "When the data being sorted includes nulls, they are considered lower than
// any non-null value, meaning they will appear at the beginning of the list when
// the data is sorted ascending, and at the end of the list when the data is sorted
// descending."
//
// The engine ranked them high instead, so a row missing the element became
// whatever the caller reads for the latest:
//
//	Last([Observation] O sort by effective)        an observation with no date
//	First([Observation] O sort by effective desc)  the same one
//
// That is exactly the idiom the published measures use — FHIR347 takes the most
// recent LDL result this way — and it is the kind of defect that reads as an
// answer. Until this branch the same query failed outright with "unknown sort
// key", so the risk arrived with the fix: a loud error became a quiet wrong
// number, which is worse.
func TestASortPutsMissingKeysLow(t *testing.T) {
	p := sortKeyProvider{withUndated: true}

	for _, tt := range []struct{ what, expr, want string }{
		{"the latest, read from the end of ascending", "Last([Observation] O sort by effective).id", "june"},
		{"the latest, read from the front of descending", "First([Observation] O sort by effective desc).id", "june"},
		// And the missing one is at the low end of each, which is where the
		// specification puts it rather than an accident of this fix.
		{"ascending starts with the missing one", "First([Observation] O sort by effective).id", "undated"},
		{"descending ends with the missing one", "Last([Observation] O sort by effective desc).id", "undated"},
	} {
		if got := evalWithObservations(t, p, tt.expr); got != tt.want {
			t.Errorf("%s: %s = %s, want %s", tt.what, tt.expr, got, tt.want)
		}
	}
}

// TestASortKeyWithNoOrderingOfItsOwnIsNotAFailure covers the other branch of a
// choice element.
//
// Observation.effective is a Period as readily as a dateTime, and a Period has no
// ordering — so resolving the choice correctly handed the sort something it could
// not compare, and the whole define aborted with "cannot compare type Period".
//
// A key that cannot be ordered sorts as null, which is what this clause already
// did for a repeating element that yields more than one value. The alternative is
// failing a measure on data FHIR fully permits, which this clause has now had to
// stop doing three times: an absent element, no rows at all, and this.
func TestASortKeyWithNoOrderingOfItsOwnIsNotAFailure(t *testing.T) {
	p := sortKeyProvider{periods: true}

	if got := evalWithObservations(t, p, "Count(([Observation] O sort by effective))"); got != "2" {
		t.Errorf("sorting by a Period-valued choice = %s, want 2 rows back", got)
	}
	// Unorderable keys are all equal, so the sort is stable and leaves them as
	// they came. That is the honest answer for values with no order, and it is
	// asserted so that giving them one later is a deliberate change rather than a
	// surprise.
	if got := evalWithObservations(t, p, "First([Observation] O sort by effective).id"); got != "earlier" {
		t.Errorf("unorderable keys should leave the order untouched, got %s", got)
	}
}
