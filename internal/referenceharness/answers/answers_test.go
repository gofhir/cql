package answers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	cql "github.com/gofhir/cql"
)

// TestReferenceAnswersAreReadAsEvaluateWritesThem covers the encoding
// Evaluate.java writes: numbers as text with their type, errors with a message,
// null as JSON null.
func TestReferenceAnswersAreReadAsEvaluateWritesThem(t *testing.T) {
	for raw, want := range map[string]string{
		`{"type":"Decimal","value":"3.33333333"}`:  "number",
		`{"type":"Integer","value":"1"}`:           "number",
		`{"type":"Boolean","value":true}`:          "boolean",
		`{"type":"String","value":"a"}`:            "string",
		`null`:                                     "null",
		`{"error":"CqlException","message":"NaN"}`: "error",
		`{"type":"List","elements":[]}`:            "list",
		`{"type":"Tuple","fields":{}}`:             "other",
	} {
		if got := Reference(json.RawMessage(raw)).Kind; got != want {
			t.Errorf("%s read as %s, want %s", raw, got, want)
		}
	}
}

// TestNumbersAreComparedByValue covers 1.0 = 1: the two engines write a number
// at different scales, and that is not a difference.
func TestNumbersAreComparedByValue(t *testing.T) {
	ref := Reference(json.RawMessage(`{"type":"Decimal","value":"1.0"}`))
	engine := evaluate(cql.NewEngine(), "library T version '1.0'\ndefine R: 1\n", "R")
	if !Same(ref, engine) {
		t.Errorf("1.0 and 1 were reported different: %+v %+v", ref, engine)
	}
	other := Reference(json.RawMessage(`{"type":"Decimal","value":"1.00000001"}`))
	if Same(other, engine) {
		t.Error("1.00000001 and 1 were reported the same")
	}
}

// TestAnEnginePanicIsCaughtAndMarked covers what a panic is turned into: an
// error marked as a panic, so it is reported even where the reference failed
// too and a plain comparison would call the two the same.
func TestAnEnginePanicIsCaughtAndMarked(t *testing.T) {
	a := evaluate(nil, "", "R") // a nil engine panics on use
	if !a.Panicked || a.Kind != "error" {
		t.Errorf("a panic became %+v, want an error marked as a panic", a)
	}
}

// TestListsAreComparedByTheirElements covers lists, which both engines write
// their own way: by element, numbers by value, nested lists alike. Every list
// was reported different before, equal or not.
func TestListsAreComparedByTheirElements(t *testing.T) {
	src := "library T version '1.0'\ndefine R: { 1, 2.0, null }\ndefine N: { { 1 }, {} }\n"
	ref := Reference(json.RawMessage(`{"type":"List","elements":[{"type":"Integer","value":"1"},` +
		`{"type":"Decimal","value":"2"},null]}`))
	if engine := evaluate(cql.NewEngine(), src, "R"); !Same(ref, engine) {
		t.Errorf("equal lists were reported different: %+v %+v", ref, engine)
	}
	nested := Reference(json.RawMessage(`{"type":"List","elements":[{"type":"List","elements":` +
		`[{"type":"Integer","value":"1"}]},{"type":"List","elements":[]}]}`))
	if engine := evaluate(cql.NewEngine(), src, "N"); !Same(nested, engine) {
		t.Errorf("equal nested lists were reported different: %+v %+v", nested, engine)
	}
	strs := Reference(json.RawMessage(`{"type":"List","elements":[{"type":"String","value":"a"}]}`))
	if engine := evaluate(cql.NewEngine(), "library T version '1.0'\ndefine S: { 'a' }\n", "S"); !Same(strs, engine) {
		t.Errorf("equal string lists were reported different: %+v %+v", strs, engine)
	}
	other := Reference(json.RawMessage(`{"type":"List","elements":[{"type":"Integer","value":"1"}]}`))
	if engine := evaluate(cql.NewEngine(), src, "R"); Same(other, engine) {
		t.Error("{1} and {1, 2.0, null} were reported the same")
	}
}

// TestQuantitiesTemporalsIntervalsAndFHIRValuesAreRead covers the answers both
// engines write that are neither a number nor a list: a quantity by its value
// and unit, a temporal by its text, an interval with its bounds' closure, and a
// FHIR resource by its JSON read back, so that the key order and the spacing of
// two encoders do not make one resource two.
func TestQuantitiesTemporalsIntervalsAndFHIRValuesAreRead(t *testing.T) {
	src := "library T version '1.0'\ndefine Q: 6.0 'mg'\ndefine D: @1980-01-02\ndefine I: Interval[1, 2)\n"
	for name, raw := range map[string]string{
		"Q": `{"type":"Quantity","value":"6","unit":"mg"}`,
		"D": `{"type":"Date","value":"1980-01-02","precision":"day"}`,
		"I": `{"type":"Interval","low":{"type":"Integer","value":"1"},"lowClosed":true,` +
			`"high":{"type":"Integer","value":"2"},"highClosed":false}`,
	} {
		ref := Reference(json.RawMessage(raw))
		if engine := evaluate(cql.NewEngine(), src, name); !Same(ref, engine) {
			t.Errorf("%s: equal answers were reported different: %+v %+v", name, ref, engine)
		}
	}
	closed := Reference(json.RawMessage(`{"type":"Interval","low":{"type":"Integer","value":"1"},` +
		`"lowClosed":true,"high":{"type":"Integer","value":"2"},"highClosed":true}`))
	if engine := evaluate(cql.NewEngine(), src, "I"); Same(closed, engine) {
		t.Error("Interval[1, 2] and Interval[1, 2) were reported the same")
	}
	a := Reference(json.RawMessage(`{"type":"FHIR.Quantity","value":"{\"unit\":\"mg\",\"value\":-5}"}`))
	b := Reference(json.RawMessage(`{"type":"FHIR.Quantity","value":"{ \"value\": -5, \"unit\": \"mg\" }"}`))
	if !Same(a, b) {
		t.Errorf("one FHIR resource in two encodings was reported different: %+v %+v", a, b)
	}
}

// evaluate is this engine's answer to a definition, as compare reads it.
func evaluate(engine *cql.Engine, src, name string) (a Answer) {
	defer func() {
		if p := recover(); p != nil {
			a = Failed(fmt.Errorf("%v", p), true)
		}
	}()
	v, err := engine.EvaluateExpression(context.Background(), src, name, nil, nil)
	if err != nil {
		return Failed(err, false)
	}
	return Value(v)
}
