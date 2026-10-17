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
		`null`:                                     "null",
		`{"error":"CqlException","message":"NaN"}`: "error",
		`{"type":"List","elements":[]}`:            "other",
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
