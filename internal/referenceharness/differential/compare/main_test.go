package main

import (
	"encoding/json"
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
		if got := referenceAnswer(json.RawMessage(raw)).kind; got != want {
			t.Errorf("%s read as %s, want %s", raw, got, want)
		}
	}
}

// TestNumbersAreComparedByValue covers 1.0 = 1: the two engines write a number
// at different scales, and that is not a difference.
func TestNumbersAreComparedByValue(t *testing.T) {
	ref := referenceAnswer(json.RawMessage(`{"type":"Decimal","value":"1.0"}`))
	engine := engineAnswer(cql.NewEngine(), "library T version '1.0'\ndefine R: 1\n", "R")
	if !same(ref, engine) {
		t.Errorf("1.0 and 1 were reported different: %+v %+v", ref, engine)
	}
	other := referenceAnswer(json.RawMessage(`{"type":"Decimal","value":"1.00000001"}`))
	if same(other, engine) {
		t.Error("1.00000001 and 1 were reported the same")
	}
}

// TestAnEnginePanicIsCaughtAndMarked covers what a panic is turned into: an
// error marked as a panic, so it is reported even where the reference failed
// too and a plain comparison would call the two the same.
func TestAnEnginePanicIsCaughtAndMarked(t *testing.T) {
	a := engineAnswer(nil, "", "R") // a nil engine panics on use
	if !a.panicked || a.kind != "error" {
		t.Errorf("a panic became %+v, want an error marked as a panic", a)
	}
}

// TestListsAreComparedByTheirElements covers lists, which both engines write
// their own way: by element, numbers by value, nested lists alike. Every list
// was reported different before, equal or not.
func TestListsAreComparedByTheirElements(t *testing.T) {
	src := "library T version '1.0'\ndefine R: { 1, 2.0, null }\ndefine N: { { 1 }, {} }\n"
	ref := referenceAnswer(json.RawMessage(`{"type":"List","elements":[{"type":"Integer","value":"1"},` +
		`{"type":"Decimal","value":"2"},null]}`))
	if engine := engineAnswer(cql.NewEngine(), src, "R"); !same(ref, engine) {
		t.Errorf("equal lists were reported different: %+v %+v", ref, engine)
	}
	nested := referenceAnswer(json.RawMessage(`{"type":"List","elements":[{"type":"List","elements":` +
		`[{"type":"Integer","value":"1"}]},{"type":"List","elements":[]}]}`))
	if engine := engineAnswer(cql.NewEngine(), src, "N"); !same(nested, engine) {
		t.Errorf("equal nested lists were reported different: %+v %+v", nested, engine)
	}
	strs := referenceAnswer(json.RawMessage(`{"type":"List","elements":[{"type":"String","value":"a"}]}`))
	if engine := engineAnswer(cql.NewEngine(), "library T version '1.0'\ndefine S: { 'a' }\n", "S"); !same(strs, engine) {
		t.Errorf("equal string lists were reported different: %+v %+v", strs, engine)
	}
	other := referenceAnswer(json.RawMessage(`{"type":"List","elements":[{"type":"Integer","value":"1"}]}`))
	if engine := engineAnswer(cql.NewEngine(), src, "R"); same(other, engine) {
		t.Error("{1} and {1, 2.0, null} were reported the same")
	}
}
