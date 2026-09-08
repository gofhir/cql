package cql

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gofhir/cql/eval"
)

// choiceComparisonProvider serves observations whose value[x] takes two different
// branches, which is the ordinary shape of FHIR data rather than an edge case:
// Observation.value is a choice of eleven types.
type choiceComparisonProvider struct{}

func (choiceComparisonProvider) Retrieve(_ context.Context, r eval.RetrieveRequest) ([]json.RawMessage, error) {
	if r.ResourceType != "Observation" {
		return nil, nil
	}
	return []json.RawMessage{
		json.RawMessage(`{"resourceType":"Observation","id":"high","status":"final",` +
			`"valueQuantity":{"value":190,"unit":"mg/dL","system":"http://unitsofmeasure.org","code":"mg/dL"}}`),
		json.RawMessage(`{"resourceType":"Observation","id":"low","status":"final",` +
			`"valueQuantity":{"value":95,"unit":"mg/dL","system":"http://unitsofmeasure.org","code":"mg/dL"}}`),
		// The same element on another branch, which is what makes the comparison
		// impossible for this row rather than merely false.
		json.RawMessage(`{"resourceType":"Observation","id":"coded","status":"final",` +
			`"valueCodeableConcept":{"coding":[{"system":"http://loinc.org","code":"LA6576-8"}]}}`),
	}, nil
}

func evalChoiceCompare(t *testing.T, expr string) string {
	t.Helper()
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\n" +
		"include FHIRHelpers version '4.0.1' called FHIRHelpers\ncontext Patient\ndefine A: " + expr + "\n"
	got, err := NewEngine(WithDataProvider(choiceComparisonProvider{}), WithEvaluationTimestamp(
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

// TestComparingAChoiceElementConvertsIt covers a FHIR choice element reaching a
// comparison, which no static type can convert.
//
// The semantic phase inserts the model's conversion where it knows the type:
// `O.valueQuantity` is a plain FHIR.Quantity, so FHIRHelpers.ToQuantity is
// planned against that node and the comparison gets a System.Quantity. `O.value`
// is typed Choice<FHIR.Quantity, FHIR.CodeableConcept, FHIR.string, …> and there
// is no way to say at compile time which branch the data will carry, so the phase
// planned nothing and the comparison was handed the raw FHIR object:
//
//	O.value >= 190 'mg/dL'   error: cannot compare Quantity
//	O.value  = 190 'mg/dL'   false
//
// The second is the one that matters. An error gets looked at; a measure quietly
// reporting false does not, and `LDL.value >= 190 'mg/dL'` is the line
// FHIR347 decides an exclusion on.
//
// Which branch a value is on is decidable at evaluation, where the value is in
// hand, and that is where the engine already converts for the timing operators,
// the membership operators and the interval accessors. The comparison operators
// were the ones nobody had taught.
func TestComparingAChoiceElementConvertsIt(t *testing.T) {
	// The premise: the concrete spelling was already right, and stays right.
	if got := evalChoiceCompare(t, "Count([Observation] O where O.valueQuantity >= 190 'mg/dL')"); got != "1" {
		t.Fatalf("the branch named concretely = %s, want 1 — this test's premise", got)
	}

	for _, tt := range []struct{ expr, want, why string }{
		{"Count([Observation] O where O.value >= 190 'mg/dL')", "1", "one of the two quantities reaches 190"},
		{"Count([Observation] O where O.value > 50 'mg/dL')", "2", "both quantities do"},
		{"Count([Observation] O where O.value < 50 'mg/dL')", "0", "neither does, and neither errors"},
		{"Count([Observation] O where O.value = 190 'mg/dL')", "1", "equality converts too, and answered false before"},
		{"Count([Observation] O where O.value ~ 95 'mg/dL')", "1", "so does equivalence"},
	} {
		if got := evalChoiceCompare(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — %s", tt.expr, got, tt.want, tt.why)
		}
	}
}

// TestTheWrongBranchOfAChoiceIsNull covers the row the comparison cannot be made
// about at all: an Observation whose value is a CodeableConcept, asked whether it
// is at least 190 mg/dL.
//
// The reference translator answers this by construction. It types the element as
// a choice, sees the comparison wants a Quantity, and emits `as Quantity` — which
// is null for a row on another branch. So the query answers about the quantities
// and passes over the rest, rather than failing the whole measure on the first
// row of the wrong kind. FHIR347's test data has both kinds.
func TestTheWrongBranchOfAChoiceIsNull(t *testing.T) {
	// Present, and on the other branch: the premise of everything below.
	if got := evalChoiceCompare(t, "Count([Observation] O where O.value is not null)"); got != "3" {
		t.Fatalf("all three rows carry a value = %s, want 3 — this test's premise", got)
	}
	if got := evalChoiceCompare(t, "Exists([Observation] O where O.value ~ ToConcept(Code { code: 'LA6576-8', system: 'http://loinc.org' }))"); got != "true" {
		t.Errorf("the coded row's value as a Concept = %s, want true", got)
	}

	// It answers about the rows it can, and does not fail on the row it cannot.
	for _, expr := range []string{
		"Count([Observation] O where O.value >= 190 'mg/dL')",
		"Count([Observation] O where O.value <= 190 'mg/dL')",
	} {
		if got := evalChoiceCompare(t, expr); strings.HasPrefix(got, "ERROR") {
			t.Errorf("%s = %s, want an answer about the quantities", expr, got)
		}
	}
}

// TestAMismatchThatIsNotANarrowingStillFails is the limit on the rule above.
//
// Answering null when a comparison cannot be made is right for a choice element,
// where "the wrong branch" is a fact about the data. It is not right for two
// values an author wrote that could never be compared: that is an authoring
// mistake, a static phase would refuse it, and turning it into null would replace
// a loud failure with a quiet wrong answer — which is the trade this repository
// has repeatedly decided the other way.
//
// So the rule is conditioned on a side having come from FHIR, and these are
// expected to keep failing.
func TestAMismatchThatIsNotANarrowingStillFails(t *testing.T) {
	for _, expr := range []string{
		"1 'mg' >= 'abc'",
		"@2020-01-01 >= 1 'mg'",
		"(Code { code: 'a' }) >= 1 'mg'",
	} {
		if got := evalChoiceCompare(t, expr); !strings.HasPrefix(got, "ERROR") {
			t.Errorf("%s = %s, want an error — nothing here came from a choice element", expr, got)
		}
	}
}
