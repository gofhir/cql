package cql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofhir/cql/eval"
)

// TestEveryValueIsOfTypeAny covers Any, the type every other one descends
// from. No value's type is named Any, so comparing names made `x as Any` null
// and `x is Any` false for every x, and `cast x as Any` failed converting to
// it. A list is a List<T> and an interval an Interval<T> when what they hold is
// of T, which only a named type had ever been compared for.
func TestEveryValueIsOfTypeAny(t *testing.T) {
	for expr, want := range map[string]string{
		"5 as Any":                                          "5",
		"'a' as System.Any":                                 "a",
		"(5 as Any) + 1":                                    "6",
		"@2020-01-01 as Any":                                "2020-01-01",
		"{ 1, 2 } as Any":                                   "{1, 2}",
		"Interval[1, 2] as Any":                             "Interval[1, 2]",
		"5 'mg' as Any":                                     "5 'mg'",
		"5 is Any":                                          "true",
		"5 is System.Any":                                   "true",
		"(null as Integer) is Any":                          "false",
		"(null as Any) is null":                             "true",
		"cast 5 as Any":                                     "5",
		"(5 as Any) as Integer":                             "5",
		"((5 as Any) as String) is null":                    "true",
		"(5 as Any) is Integer":                             "true",
		"(List<Any>{ { 1, 2 } })[0] is List<Any>":           "true",
		"(List<Any>{ { 1, 2 } })[0] is List<Integer>":       "true",
		"(List<Any>{ { 1, 'a' } })[0] is List<Integer>":     "false",
		"(List<Any>{ Interval[1, 2] })[0] is Interval<Any>": "true",
		"(List<Any>{ 5 })[0] is List<Any>":                  "false",
	} {
		v, err := NewEngine().EvaluateExpression(context.Background(), "library T version '1.0'\ndefine R: "+expr+"\n", "R", nil, nil)
		got := "null"
		if v != nil {
			got = valueString(v)
		}
		if err != nil || got != want {
			t.Errorf("%s = %s, %v; want %s", expr, got, err, want)
		}
	}
}

// TestTheFHIRModelsDescentIsOnlyAFHIRValues covers a supertype of the FHIR
// model, qualified or not, which a System value of a FHIR type's name is not
// of: a System Quantity is not a FHIR.Element, or an Element where the library
// uses FHIR, though FHIR's Quantity descends from it. And arithmetic on a
// value that is no number fails rather than read it as zero, as negation, Abs,
// Sum and Avg did; `as Any` hands them such values.
func TestTheFHIRModelsDescentIsOnlyAFHIRValues(t *testing.T) {
	const header = "library T version '1.0'\nusing FHIR version '4.0.1'\n"
	for expr, want := range map[string]string{
		"5 'mg' as FHIR.Element":                                              "null",
		"5 'mg' as Element":                                                   "null",
		"5 'mg' is Element":                                                   "false",
		"-(years between @2000 and @2010-05-01)":                              "Interval[-10, -9]",
		"5 'mg' is FHIR.Element":                                              "false",
		"5 'mg' is System.Quantity":                                           "true",
		"(List<Any>{ 5 'mg' })[0] as Quantity":                                "5 'mg'",
		"FHIR.Coding { code: FHIR.code { value: 'x' } } is FHIR.Element":      "true",
		"(FHIR.Coding { code: FHIR.code { value: 'x' } } as Element) is null": "false",
		"System.ValueSet { id: 'x' } is FHIR.Resource":                        "false",
		"{ 1, 2 } is FHIR.Resource":                                           "false",
		"FHIR.Coding { code: FHIR.code { value: 'x' } } is FHIR.Coding":       "true",
	} {
		v, err := NewEngine().EvaluateExpression(context.Background(), header+"define R: "+expr+"\n", "R", nil, nil)
		got := "null"
		if v != nil {
			got = valueString(v)
		}
		if err != nil || got != want {
			t.Errorf("%s = %s, %v; want %s", expr, got, err, want)
		}
	}
	for _, expr := range []string{"Abs('a' as Any)", "-('a' as Any)", "Sum({ 'a' as Any })", "Avg({ 'a' as Any })"} {
		if _, err := NewEngine().EvaluateExpression(context.Background(), header+"define R: "+expr+"\n", "R", nil, nil); err == nil {
			t.Errorf("%s was evaluated; a string is no number", expr)
		}
	}
}

// fhirTypesProvider serves an encounter that starts on a date with no time,
// and an observation with a quantity, a code and a date-only effective time.
type fhirTypesProvider struct{}

func (fhirTypesProvider) Retrieve(_ context.Context, r eval.RetrieveRequest) ([]json.RawMessage, error) {
	switch r.ResourceType {
	case "Encounter":
		return []json.RawMessage{json.RawMessage(`{"resourceType":"Encounter","id":"e","status":"finished",` +
			`"period":{"start":"2020-01-01","end":"2020-01-03T10:00:00Z"}}`)}, nil
	case "Observation":
		return []json.RawMessage{json.RawMessage(`{"resourceType":"Observation","id":"o","status":"final",` +
			`"effectiveDateTime":"2020-01-01","code":{"coding":[{"system":"s","code":"c"}]},` +
			`"valueQuantity":{"value":-5,"unit":"mg","system":"http://unitsofmeasure.org","code":"mg"}}`)}, nil
	}
	return nil, nil
}

// TestAFHIRValueThroughIsAsAndCast covers FHIR data. A dateTime with no time
// keeps its type and its arithmetic — the timing of a measure reads it — and
// is a FHIR.dateTime. A FHIR element cast to a FHIRPath System type is the
// System value it holds when that is the type, and a value cast to its own
// type is itself: cast Patient as Patient, and cast O.value as Quantity,
// failed converting. Abs of a FHIR Quantity is the System one's.
func TestAFHIRValueThroughIsAsAndCast(t *testing.T) {
	const header = "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1'\n" +
		"context Patient\n"
	patient := []byte(`{"resourceType":"Patient","id":"p","birthDate":"1980-01-01","deceasedDateTime":"2020-02-02"}`)
	engine := NewEngine(WithDataProvider(fhirTypesProvider{}))
	for expr, want := range map[string]string{
		"First([Encounter]).period.start + 1 day":                               "2020-01-02",
		"Count([Encounter] E where E.period.start same day as @2020-01-01T)":    "1",
		"Count([Encounter] E where start of E.period same day as @2020-01-01T)": "1",
		"Patient.deceased same year as @2020T":                                  "true",
		"Patient.deceased is FHIR.dateTime":                                     "true",
		"First([Observation]).effective is FHIR.dateTime":                       "true",
		"Patient.birthDate is FHIR.date":                                        "true",
		"Abs(First([Observation]).value as FHIR.Quantity)":                      "5 'mg'",
		"(cast First([Observation]).value as Quantity) + 1 'mg'":                "-4 'mg'",
		"(cast Patient as Patient).id":                                          "p",
		"(cast Patient as FHIR.Patient).id":                                     "p",
		"(cast First([Observation]).code as CodeableConcept) is null":           "false",
	} {
		v, err := engine.EvaluateExpression(context.Background(), header+"define R: "+expr+"\n", "R", patient, nil)
		got := "null"
		if v != nil {
			got = valueString(v)
		}
		if err != nil || got != want {
			t.Errorf("%s = %s, %v; want %s", expr, got, err, want)
		}
	}
}
