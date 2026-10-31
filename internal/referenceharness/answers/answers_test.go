package answers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
		`{"type":"Tuple","fields":{}}`:             "structure",
		`{"type":"Number","value":"1"}`:            "untyped",
		`{"type":"Unencoded:Foo","value":"x"}`:     "other",
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

// TestStructuresAndUntypedValuesAreComparedByTheirParts covers what the
// reference harness writes for a Code, a Concept and a tuple, and what
// cql-execution writes where it does not know the type: a Number (Integer or
// Decimal), a Text (String or Long) and an Object (a tuple or an instance). A
// Code selector leaves the elements it does not set null (CQL 1.5, Code), and
// this engine answers them null, though it holds them as empty strings.
func TestStructuresAndUntypedValuesAreComparedByTheirParts(t *testing.T) {
	ref := func(raw string) Answer { return Reference(json.RawMessage(raw)) }
	eng := func(expr string) Answer {
		return evaluate(cql.NewEngine(), "library T version '1.0'\ndefine R: "+expr+"\n", "R")
	}
	fullCode := `{"type":"Code","code":"x","system":"s","version":"1","display":"d"}`
	for _, tt := range []struct {
		name string
		a, b Answer
		same bool
	}{
		{"a Code with every field", ref(fullCode), eng("Code { code: 'x', system: 's', version: '1', display: 'd' }"), true},
		{"a Code of another system", ref(fullCode), eng("Code { code: 'x', system: 't', version: '1', display: 'd' }"), false},
		{"a Code's null fields against those this engine leaves unset", ref(`{"type":"Code","code":"x","system":null,"version":null,"display":null}`),
			eng("Code { code: 'x' }"), true},
		{"a Code's empty field against one this engine leaves unset", ref(`{"type":"Code","code":"x","system":"","version":null,"display":null}`),
			eng("Code { code: 'x' }"), false},
		{"a Concept", ref(`{"type":"Concept","codes":[` + fullCode + `],"display":"c"}`),
			eng("Concept { codes: { Code { code: 'x', system: 's', version: '1', display: 'd' } }, display: 'c' }"), true},
		{"a Concept short of a code", ref(`{"type":"Concept","codes":[` + fullCode + `],"display":"c"}`),
			eng("Concept { codes: { Code { code: 'x', system: 's', version: '1', display: 'd' }, " +
				"Code { code: 'y', system: 's', version: '1', display: 'd' } }, display: 'c' }"), false},
		{"a Concept of null codes is not one of none", ref(`{"type":"Concept","codes":null,"display":"d"}`),
			eng("Concept { codes: List<Code>{}, display: 'd' }"), false},
		{"a Concept of no codes", ref(`{"type":"Concept","codes":[],"display":"d"}`),
			eng("Concept { codes: List<Code>{}, display: 'd' }"), true},
		{"a typed tuple", ref(`{"type":"Tuple","fields":{"a":{"type":"Integer","value":"1"}}}`), eng("Tuple { a: 1 }"), true},
		{"a typed tuple with a Decimal for an Integer", ref(`{"type":"Tuple","fields":{"a":{"type":"Decimal","value":"1"}}}`),
			eng("Tuple { a: 1 }"), false},
		{"an Object as a tuple", ref(`{"type":"Object","fields":{"a":{"type":"Number","value":"1"}}}`), eng("Tuple { a: 1 }"), true},
		{"an Object with a field more", ref(`{"type":"Object","fields":{"a":{"type":"Number","value":"1"},"b":null}}`),
			eng("Tuple { a: 1 }"), false},
		{"an Object with a field renamed", ref(`{"type":"Object","fields":{"b":{"type":"Number","value":"1"}}}`),
			eng("Tuple { a: 1 }"), false},
		{"an Object as an instance", ref(`{"type":"Object","fields":{"id":{"type":"Text","value":"x"}}}`),
			eng("System.ValueSet { id: 'x' }"), true},
		{"a typed tuple is no instance", ref(`{"type":"Tuple","fields":{"id":{"type":"String","value":"x"}}}`),
			eng("System.ValueSet { id: 'x' }"), false},
		// CQL 1.5.3, Clinical Values: a ValueSet has id, version, name and
		// codesystems, and a CodeSystem id, version and name. The Java engine
		// writes each, a text plain as in a Code.
		{"a ValueSet's null fields against those this engine leaves unset",
			ref(`{"type":"ValueSet","fields":{"codesystems":null,"id":"x","name":null,"version":null}}`),
			eng("System.ValueSet { id: 'x' }"), true},
		{"a ValueSet of another id", ref(`{"type":"ValueSet","fields":{"codesystems":null,"id":"y","name":null,"version":null}}`),
			eng("System.ValueSet { id: 'x' }"), false},
		{"a ValueSet with a name this engine has not", ref(`{"type":"ValueSet","fields":{"codesystems":null,"id":"x","name":"n","version":null}}`),
			eng("System.ValueSet { id: 'x' }"), false},
		{"a ValueSet of no code systems is not one of null",
			ref(`{"type":"ValueSet","fields":{"codesystems":{"type":"List","elements":[]},"id":"x","name":null,"version":null}}`),
			eng("System.ValueSet { id: 'x' }"), false},
		{"a ValueSet of a code system",
			ref(`{"type":"ValueSet","fields":{"codesystems":{"type":"List","elements":[{"type":"CodeSystem","fields":{"id":"s","name":null,"version":"1"}}]},"id":"x","name":null,"version":null}}`),
			eng("System.ValueSet { id: 'x', codesystems: { System.CodeSystem { id: 's', version: '1' } } }"), true},
		{"a ValueSet of a code system of another version",
			ref(`{"type":"ValueSet","fields":{"codesystems":{"type":"List","elements":[{"type":"CodeSystem","fields":{"id":"s","name":null,"version":"2"}}]},"id":"x","name":null,"version":null}}`),
			eng("System.ValueSet { id: 'x', codesystems: { System.CodeSystem { id: 's', version: '1' } } }"), false},
		{"a ValueSet is no tuple", ref(`{"type":"ValueSet","fields":{"codesystems":null,"id":"x","name":null,"version":null}}`),
			eng("Tuple { id: 'x' }"), false},
		{"a CodeSystem", ref(`{"type":"CodeSystem","fields":{"id":"s","name":null,"version":null}}`),
			eng("System.CodeSystem { id: 's' }"), true},
		{"a CodeSystem's null given explicitly", ref(`{"type":"CodeSystem","fields":{"id":"s","name":null,"version":null}}`),
			eng("System.CodeSystem { id: 's', version: null }"), true},
		{"cql-execution's CodeSystem with a null given explicitly", ref(`{"type":"Object","fields":{"id":{"type":"Text","value":"s"},"version":null}}`),
			eng("System.CodeSystem { id: 's', version: null }"), true},
		{"a CodeSystem is no ValueSet", ref(`{"type":"CodeSystem","fields":{"id":"s","name":null,"version":null}}`),
			eng("System.ValueSet { id: 's' }"), false},
		{"an empty Object is no empty list", ref(`{"type":"Object","fields":{}}`), eng("List<Integer>{}"), false},
		{"a Code of another display", ref(fullCode), eng("Code { code: 'x', system: 's', version: '1', display: 'e' }"), false},
		{"a Code of another version", ref(fullCode), eng("Code { code: 'x', system: 's', version: '2', display: 'd' }"), false},
		{"a Number as an Integer", ref(`{"type":"Number","value":"1"}`), eng("1"), true},
		{"a Number as a Decimal", ref(`{"type":"Number","value":"2.5"}`), eng("2.5"), true},
		{"a Number of another value", ref(`{"type":"Number","value":"2"}`), eng("1"), false},
		{"a Number is no String", ref(`{"type":"Number","value":"1"}`), eng("'1'"), false},
		{"a Text as a String", ref(`{"type":"Text","value":"a"}`), eng("'a'"), true},
		{"a Text is no number", ref(`{"type":"Text","value":"5"}`), eng("5"), false},
		{"Numbers in a list", ref(`{"type":"List","elements":[{"type":"Number","value":"1"},{"type":"Number","value":"2"}]}`),
			eng("{ 1, 2 }"), true},
		{"Numbers in a list, one changed", ref(`{"type":"List","elements":[{"type":"Number","value":"1"},{"type":"Number","value":"3"}]}`),
			eng("{ 1, 2 }"), false},
		{"Numbers in a list, one short", ref(`{"type":"List","elements":[{"type":"Number","value":"1"}]}`), eng("{ 1, 2 }"), false},
		{"a Number in an interval", ref(`{"type":"Interval","low":{"type":"Number","value":"1"},"high":null,"lowClosed":true,"highClosed":true}`),
			eng("Interval[1, null]"), true},
		{"a Number in an interval closed otherwise", ref(`{"type":"Interval","low":{"type":"Number","value":"1"},"high":null,"lowClosed":true,"highClosed":false}`),
			eng("Interval[1, null]"), false},
	} {
		if got := Same(tt.a, tt.b); got != tt.same {
			t.Errorf("%s: Same(%s, %s) = %v, want %v", tt.name, tt.a.Text, tt.b.Text, got, tt.same)
		}
		if got := Same(tt.b, tt.a); got != tt.same {
			t.Errorf("%s, reversed: Same = %v, want %v", tt.name, got, tt.same)
		}
		if got := SameExactly(tt.a, tt.b); got != tt.same {
			t.Errorf("%s: SameExactly = %v, want %v", tt.name, got, tt.same)
		}
	}
	if !HoldsUntyped(ref(`{"type":"List","elements":[{"type":"Number","value":"1"}]}`)) || HoldsUntyped(eng("{ 1 }")) {
		t.Error("HoldsUntyped misreads a list")
	}
}

// TestAnIntervalOfTextsIsComparedByItsBounds covers an interval whose bounds
// cql-execution writes as Text: an interval of Strings is no interval the
// text compares, and its bounds still are.
func TestAnIntervalOfTextsIsComparedByItsBounds(t *testing.T) {
	ref := Reference(json.RawMessage(`{"type":"Interval","low":{"type":"Text","value":"a"},"high":{"type":"Text","value":"b"},` +
		`"lowClosed":true,"highClosed":true}`))
	if !Same(ref, evaluate(cql.NewEngine(), "library T version '1.0'\ndefine R: Interval['a', 'b']\n", "R")) {
		t.Error("an interval of Texts is not the same as one of the same Strings")
	}
	if Same(ref, evaluate(cql.NewEngine(), "library T version '1.0'\ndefine R: Interval['a', 'c']\n", "R")) {
		t.Error("an interval of Texts is the same as one of other Strings")
	}
}

// TestTheRuntimeReferencesTuplesAreReadAsTuples covers the tuples both
// reference engines answered in the runtime corpus, read as the harness wrote
// them, against this engine's: a tuple of the same fields is the same, and
// one with a field of another value is not.
func TestTheRuntimeReferencesTuplesAreReadAsTuples(t *testing.T) {
	for _, file := range []string{"runtime.expected.json", "runtime.expected.js.json"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "reference", file))
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Results map[string]json.RawMessage `json:"results"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		for name, expr := range map[string]string{
			"ReturnTuple":         "({1, 1}) X return Tuple { v: X }",
			"MultiSourceNoReturn": "from ({1, 1}) A, ({2}) B",
		} {
			ref := Reference(doc.Results[name])
			if !Same(ref, evaluate(cql.NewEngine(), "library T version '1.0'\ndefine R: "+expr+"\n", "R")) {
				t.Errorf("%s %s: %s is not this engine's", file, name, ref.Text)
			}
		}
		wrong := evaluate(cql.NewEngine(), "library T version '1.0'\ndefine R: { Tuple { v: 2 } }\n", "R")
		if Same(Reference(doc.Results["ReturnTuple"]), wrong) {
			t.Errorf("%s ReturnTuple is the same as a tuple of another value", file)
		}
	}
}

// TestAnUncertaintyIsReadWithBothBounds covers cql-execution's Uncertainty,
// which evaluate.js writes with its two bounds: two uncertainties are the same
// answer only when both bounds agree, and neither is a number or an interval
// (plan 015). Written as Unencoded, every uncertainty read the same.
func TestAnUncertaintyIsReadWithBothBounds(t *testing.T) {
	src := "library T version '1.0'\ndefine U: years between @2012 and @2014\ndefine V: years between @2014 and @2012\n"
	raw := func(low, high string) json.RawMessage {
		return json.RawMessage(`{"type":"Uncertainty","low":{"type":"Integer","value":"` + low +
			`"},"high":{"type":"Integer","value":"` + high + `"}}`)
	}
	engine := cql.NewEngine()
	if ref := Reference(raw("1", "2")); !Same(ref, evaluate(engine, src, "U")) {
		t.Errorf("an uncertainty and the same one were reported different: %+v %+v", ref, evaluate(engine, src, "U"))
	}
	if ref := Reference(raw("1", "3")); Same(ref, evaluate(engine, src, "U")) {
		t.Error("[1, 3] and [1, 2] were reported the same")
	}
	if ref := Reference(raw("-2", "-1")); !Same(ref, evaluate(engine, src, "V")) {
		t.Error("the reverse uncertainty was reported different from [-2, -1]")
	}
	if Same(Reference(raw("1", "2")), Reference(json.RawMessage(`{"type":"Integer","value":"2"}`))) {
		t.Error("an uncertainty was reported the same as its upper bound")
	}
}
