package answers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/shopspring/decimal"

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
		`{"error":"Translation","message":"x"}`:    "refused",
		`{"type":"List","elements":[]}`:            "list",
		`{"type":"Tuple","fields":{}}`:             "other",
	} {
		if got := Reference(json.RawMessage(raw)).Kind; got != want {
			t.Errorf("%s read as %s, want %s", raw, got, want)
		}
	}
}

// TestNumbersAreComparedByValueWithinTheirType covers 1.0 = 1 between two
// Decimals, which the two engines write at different scales, and Integer 3
// against Decimal 3.0, which are not the same answer: compared by value alone,
// `7.0 div 2` answering an Integer went unseen by every oracle.
//
// An Integer and a Long are the same here, a known loss: this engine has no Long
// at run time (D4c in docs/arquitectura-verificacion-2026-10-18.md), so a Long
// it computes reads as an Integer, and telling them apart would report every
// Long answer of the integer area.
func TestNumbersAreComparedByValueWithinTheirType(t *testing.T) {
	number := func(raw string) Answer { return Reference(json.RawMessage(raw)) }
	engine := evaluate(cql.NewEngine(), "library T version '1.0'\ndefine R: 1.0\n", "R")
	for _, tt := range []struct {
		name string
		a, b Answer
		same bool
	}{
		{"Decimals at two scales", number(`{"type":"Decimal","value":"1"}`), engine, true},
		{"Decimals that differ", number(`{"type":"Decimal","value":"1.00000001"}`), engine, false},
		{"an Integer and a Decimal", number(`{"type":"Integer","value":"1"}`), engine, false},
		// The defect's own shape: the reference answers a Decimal, this engine an
		// Integer, in the order the oracle calls Same.
		{"a Decimal and an Integer", number(`{"type":"Decimal","value":"1"}`),
			evaluate(cql.NewEngine(), "library T version '1.0'\ndefine R: 1\n", "R"), false},
		{"an Integer and a Long, the known loss", number(`{"type":"Long","value":"3"}`),
			number(`{"type":"Integer","value":"3"}`), true},
	} {
		if got := Same(tt.a, tt.b); got != tt.same {
			t.Errorf("%s: Same(%+v, %+v) = %t; want %t", tt.name, tt.a, tt.b, got, tt.same)
		}
	}
}

// TestAContainersNumbersKeepTheirType covers a number inside a list or an
// interval, compared by its text there: an Integer element stood for a Decimal
// one, so {7.0 div 2} answering {3} was the same as {3.0}.
func TestAContainersNumbersKeepTheirType(t *testing.T) {
	ref := func(raw string) Answer { return Reference(json.RawMessage(raw)) }
	for _, tt := range []struct {
		name string
		a, b Answer
	}{
		{"a list", ref(`{"type":"List","elements":[{"type":"Integer","value":"3"}]}`),
			ref(`{"type":"List","elements":[{"type":"Decimal","value":"3.0"}]}`)},
		{"a nested list", ref(`{"type":"List","elements":[{"type":"List","elements":[{"type":"Integer","value":"3"}]}]}`),
			ref(`{"type":"List","elements":[{"type":"List","elements":[{"type":"Decimal","value":"3"}]}]}`)},
		{"an interval", ref(`{"type":"Interval","low":{"type":"Integer","value":"3"},"high":{"type":"Integer","value":"4"},"lowClosed":true,"highClosed":true}`),
			ref(`{"type":"Interval","low":{"type":"Decimal","value":"3"},"high":{"type":"Decimal","value":"4"},"lowClosed":true,"highClosed":true}`)},
	} {
		if Same(tt.a, tt.b) {
			t.Errorf("%s: an Integer and a Decimal were reported the same: %+v %+v", tt.name, tt.a, tt.b)
		}
	}
	same := ref(`{"type":"List","elements":[{"type":"Decimal","value":"3"}]}`)
	if !Same(same, ref(`{"type":"List","elements":[{"type":"Decimal","value":"3.00"}]}`)) {
		t.Error("two Decimal elements at two scales were reported different")
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
	// Promoted to a list of Decimals, as the Java engine answers a list literal
	// that mixes the two (D22 of testdata/snapshots/division.java.json).
	ref := Reference(json.RawMessage(`{"type":"List","elements":[{"type":"Decimal","value":"1"},` +
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
	var semantic *cql.ErrSemantic
	var syntax *cql.ErrSyntaxError
	switch {
	case errors.As(err, &semantic), errors.As(err, &syntax):
		return Refused(err)
	case err != nil:
		return Failed(err, false)
	}
	return Value(v)
}

// TestARefusalIsNotARunTimeError covers the two phases an error can come
// from: a library the translator refuses is not one that fails as it runs.
// Read as one answer, the Java translator refusing a cast hid this engine
// failing at run time on the same cast, which has to succeed (D40 of fhir).
func TestARefusalIsNotARunTimeError(t *testing.T) {
	translation := Reference(json.RawMessage(`{"error":"Translation","message":"cannot be cast"}`))
	runTime := Reference(json.RawMessage(`{"error":"CqlException","message":"cannot cast"}`))
	refused := evaluate(cql.NewEngine(), "library R version '1'\ndefine R: Undefined + 1\n", "R")
	failed := evaluate(cql.NewEngine(), "library R version '1'\ndefine R: Message(1, true, 'E', 'Error', 'boom')\n", "R")
	if refused.Kind != "refused" || failed.Kind != "error" {
		t.Fatalf("a semantic error read as %s and a run-time one as %s; want refused and error", refused.Kind, failed.Kind)
	}
	for _, c := range []struct {
		name string
		a, b Answer
		same bool
	}{
		{"two refusals", translation, refused, true},
		{"two run-time errors", runTime, failed, true},
		{"a refusal and a run-time error", translation, failed, false},
		{"a run-time error and a refusal", runTime, refused, false},
	} {
		if got := Same(c.a, c.b); got != c.same {
			t.Errorf("%s: Same = %v, want %v", c.name, got, c.same)
		}
	}
}

// TestACorpusWithoutFloatsIsComparedExactly covers what the tolerance cannot
// see: two whole numbers padded with zeros, a quotient off by a hundred
// thousand, are within it, and SameExactly tells them apart.
func TestACorpusWithoutFloatsIsComparedExactly(t *testing.T) {
	number := func(s string) Answer {
		return Answer{Kind: "number", Text: "Decimal " + s, Num: decimal.RequireFromString(s)}
	}
	a, b := number("10000000000000000000"), number("10000000000000100000")
	if !Same(a, b) {
		t.Error("the tolerance no longer covers this pair; the comment on SameExactly is out of date")
	}
	if SameExactly(a, b) {
		t.Errorf("SameExactly(%s, %s) = true; want false", a.Text, b.Text)
	}
	if !SameExactly(number("3.0"), number("3")) {
		t.Error("SameExactly told two scales of one Decimal apart")
	}
	integer := Answer{Kind: "number", Text: "Integer 3", Num: decimal.NewFromInt(3)}
	if SameExactly(integer, number("3")) {
		t.Error("SameExactly took an Integer for a Decimal")
	}
}

// TestAFloat64sNumbersAreTheSameAcrossPlatforms covers the numbers a float64
// computed, which differ in their last digit between amd64 and arm64: the four
// CI and a developer's machine wrote apart are the same; a number of fewer
// digits, one that moved by more than that noise, or a Long, is not.
func TestAFloat64sNumbersAreTheSameAcrossPlatforms(t *testing.T) {
	number := func(s string) Answer {
		d := decimal.RequireFromString(s)
		return Answer{Kind: "number", Text: "Decimal " + s, Num: d}
	}
	for _, tt := range []struct {
		a, b string
		same bool
	}{
		{"183.09405819371818", "183.09405819371815", true},
		{"4.4816890703380645", "4.481689070338065", true},
		{"1907346572495099500000", "1907346572495099800000", true},
		{"0.15311848890866597", "0.153118488908666", true},
		{"3.33333333", "3.33333334", false},
		{"4.4816890703380645", "4.5816890703380645", false},
		{"10.0", "10", true},
		// A whole number of more digits than a float64 writes is no float's,
		// but an exact quotient's: one a unit away is a wrong answer. Whole
		// numbers of 16 and 17 digits stay within the tolerance, a known gap.
		{"99999999999999999999", "99999999999999999998", false},
		{"10000000000000000000", "10000000000000000001", false},
	} {
		if got := Same(number(tt.a), number(tt.b)); got != tt.same {
			t.Errorf("Same(%s, %s) = %t; want %t", tt.a, tt.b, got, tt.same)
		}
	}
	// A Long is exact: one past its range, wrapped, is a hair of its size away.
	long := func(s string) Answer {
		return Answer{Kind: "number", Text: "Long " + s, Num: decimal.RequireFromString(s)}
	}
	if Same(long("9223372036854775807"), long("9223372036854775806")) {
		t.Error("two Longs a unit apart were reported the same")
	}
}
