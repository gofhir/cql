package cql

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gofhir/cql/eval"
)

// TestASetOperationWithAnAbsentOperandDoesNotDependOnCardinality covers union and
// except with an operand that evaluates to null, over FHIR data.
//
// A repeating element with one entry arrives unwrapped, as a single value, and
// with two as a list. Union and except with null now remove a list's
// duplicates, and a single value has to come out of it as it went in: a draft
// that read anything but a list as an interval made the union of one name
// with an absent element null, and counted 0 where two names counted 2.
func TestASetOperationWithAnAbsentOperandDoesNotDependOnCardinality(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1' called FH\n" +
		"context Patient\n" +
		"define UnionCount: Count(Patient.name union Patient.contact)\n" +
		"define ExceptCount: Count(Patient.name except Patient.contact)\n" +
		"define PeriodsUnionNull: Count(Patient.name.period union null)\n"
	e := NewEngine()
	for _, tt := range []struct {
		patient, names string
	}{
		{`{"resourceType":"Patient","id":"p1","name":[{"family":"A","period":{"start":"2020-01-01"}}]}`, "1"},
		{`{"resourceType":"Patient","id":"p2","name":[{"family":"A","period":{"start":"2020-01-01"}},{"family":"B","period":{"start":"2021-01-01"}}]}`, "2"},
	} {
		for _, def := range []string{"UnionCount", "ExceptCount"} {
			v, err := e.EvaluateExpression(context.Background(), src, def, []byte(tt.patient), nil)
			if err != nil || valueString(v) != tt.names {
				t.Errorf("%s with %s names = %v, %v; want %s", def, tt.names, v, err, tt.names)
			}
		}
		// A one-entry list of Periods arrives as a single Period. Asked of one
		// name only: path navigation through two names is null today whatever
		// follows it — a separate defect, Patient.name.family included.
		if tt.names == "1" {
			v, err := e.EvaluateExpression(context.Background(), src, "PeriodsUnionNull", []byte(tt.patient), nil)
			if err != nil || valueString(v) != "1" {
				t.Errorf("Count(Patient.name.period union null) with one name = %v, %v; want 1", v, err)
			}
		}
	}
}

// TestASingleValueWithNullIsItself covers union and except of a single value
// that is neither a list nor an interval with null. `5 union null` is not CQL —
// the reference translator has no Union for (Integer, Any) — and the semantic
// phase types it as the Integer, so it answers the Integer, which the
// expressions around it rely on.
func TestASingleValueWithNullIsItself(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1' called FH\n" +
		"context Patient\n" +
		"define PlusOne: (5 union null) + 1\n" +
		"define ExceptEqual: (5 except null) = 5\n" +
		"define FamilyOf: (First(Patient.name) union null).family\n"
	e := NewEngine()
	patient := []byte(`{"resourceType":"Patient","id":"p","name":[{"family":"A"}]}`)
	for def, want := range map[string]string{"PlusOne": "6", "ExceptEqual": "true", "FamilyOf": "A"} {
		v, err := e.EvaluateExpression(context.Background(), src, def, patient, nil)
		if err != nil || valueString(v) != want {
			t.Errorf("%s = %v, %v; want %s", def, v, err, want)
		}
	}
}

// TestASetOperationOverAnIncludedDefinitionIsUnchanged covers operands that are
// definitions of an included library, which is how measures reach most of what
// they share. When a draft converted a value to ask whether it was an interval,
// the one Period of a one-entry list of Periods read as one, and
// Count(L.Periods union null) went from 1 to 0.
func TestASetOperationOverAnIncludedDefinitionIsUnchanged(t *testing.T) {
	lib := "library L version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1' called FH\n" +
		"context Patient\ndefine Periods: Patient.name.period\n"
	e := NewEngine(WithLibraryResolver(func(_ context.Context, name, _ string) (string, error) {
		if name == "L" {
			return lib, nil
		}
		return "", fmt.Errorf("no library %s", name)
	}))
	src := "library M version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1' called FH\n" +
		"include L version '1.0' called L\ncontext Patient\n" +
		"define UnionNull: Count(L.Periods union null)\n" +
		"define ExceptNull: Count(L.Periods except null)\n"
	patient := `{"resourceType":"Patient","id":"p","name":[{"family":"A","period":{"start":"2020-01-01","end":"2020-02-01"}}]}`
	for _, def := range []string{"UnionNull", "ExceptNull"} {
		v, err := e.EvaluateExpression(context.Background(), src, def, []byte(patient), nil)
		if err != nil || valueString(v) != "1" {
			t.Errorf("%s over an included one-entry list of Periods = %v, %v; want 1", def, v, err)
		}
	}
}

type comparatorQuantityProvider struct{}

func (comparatorQuantityProvider) Retrieve(_ context.Context, r eval.RetrieveRequest) ([]json.RawMessage, error) {
	if r.ResourceType == "Observation" {
		return []json.RawMessage{json.RawMessage(`{"resourceType":"Observation","id":"o","status":"final",` +
			`"code":{"text":"x"},"valueQuantity":{"value":5,"comparator":"<","unit":"mg"}}`)}, nil
	}
	return nil, nil
}

// TestASetOperationDoesNotConvertToDecideItsShape covers a value whose
// conversion fails. Deciding whether an operand is a list or an interval
// converts nothing, and converting a Quantity with a comparator fails, which
// would fail the whole evaluation over a question the operation does not use
// the answer to.
func TestASetOperationDoesNotConvertToDecideItsShape(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1' called FH\n" +
		"context Patient\n" +
		"define ValueUnionNull: First([Observation] O return O.value union null)\n" +
		"define ValueExceptNull: First([Observation] O return O.value except null)\n" +
		"define TypedValueUnionNull: First([Observation] O return (O.value as FHIR.Quantity) union null)\n"
	e := NewEngine(WithDataProvider(comparatorQuantityProvider{}))
	for _, def := range []string{"ValueUnionNull", "ValueExceptNull", "TypedValueUnionNull"} {
		v, err := e.EvaluateExpression(context.Background(), src, def, []byte(`{"resourceType":"Patient","id":"p"}`), nil)
		if err != nil || v == nil {
			t.Errorf("%s = %v, %v; want the Quantity", def, v, err)
		}
	}
}
