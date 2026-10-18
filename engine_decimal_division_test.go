package cql

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/compiler"
	"github.com/gofhir/cql/elm"
	"github.com/gofhir/cql/eval"
)

// The division area of testdata/snapshots holds the System cases against both
// reference engines. These are the ones it cannot hold: operands read from FHIR
// data, a library imported from ELM, and the quotient the specification and
// the Java engine disagree on.
//
// CQL Appendix B, Arithmetic Operators, Truncated Divide: div(Decimal, Decimal)
// is a Decimal, and an Integer argument with a Decimal one is converted to a
// Decimal; div(Integer, Integer) is an Integer.

// TestATruncatedDivisionOfFHIRDataKeepsItsType covers operands read from FHIR
// data. A FHIR.decimal divided answered an Integer; a FHIR.integer divided by
// another stays one, which guards the fix from taking whole numbers to the
// Decimal branch.
func TestATruncatedDivisionOfFHIRDataKeepsItsType(t *testing.T) {
	const header = "library T version '1.0'\nusing FHIR version '4.0.1'\n" +
		"include FHIRHelpers version '4.0.1' called FHIRHelpers\ncontext Patient\n"
	for _, tt := range []struct {
		name, resource, expr string
		decimal              bool
		want                 string
	}{
		{"a FHIR decimal", `{"resourceType":"ChargeItem","id":"c","factorOverride":7.5}`,
			"First([ChargeItem]).factorOverride div 2", true, "3"},
		{"a FHIR decimal of a choice", `{"resourceType":"Observation","id":"o","valueQuantity":` +
			`{"value":7.5,"unit":"mg","system":"http://unitsofmeasure.org","code":"mg"}}`,
			"(First([Observation]).value as FHIR.Quantity).value div 2", true, "3"},
		{"a FHIR integer", `{"resourceType":"Patient","id":"p","multipleBirthInteger":7}`,
			"(First([Patient]).multipleBirth as FHIR.integer) div 2", false, "3"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine(WithDataProvider(resourceProvider{rows: []string{tt.resource}}))
			got, err := engine.EvaluateExpression(context.Background(), header+"define X: "+tt.expr+"\n", "X", nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			assertNumber(t, got, tt.decimal, tt.want)
		})
	}
}

// TestATruncatedDivisionImportedFromELMIsADecimal covers TruncatedDivide read
// from ELM, which the importer maps to the operator a parsed library has.
func TestATruncatedDivisionImportedFromELMIsADecimal(t *testing.T) {
	parsed, err := compiler.Compile("library T version '1.0'\ndefine X: 7.0 div 2\n")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := elm.MarshalLibrary(elm.Translate(parsed))
	if err != nil {
		t.Fatal(err)
	}
	read, err := elm.UnmarshalLibrary(raw)
	if err != nil {
		t.Fatal(err)
	}
	imported, err := elm.Import(read)
	if err != nil {
		t.Fatal(err)
	}
	got, err := eval.NewEvaluator(eval.NewContext(context.Background(), imported)).EvaluateExpression("X")
	if err != nil {
		t.Fatal(err)
	}
	assertNumber(t, got, true, "3")
}

// TestATruncatedQuotientPastTheDecimalRangeIsNull covers the quotient of two
// valid Decimals that no Decimal holds: 99999999999999999999.99999999 div
// 0.00000001 is 9999999999999999999999999999, past the largest Decimal,
// (10^28 - 1) / 10^8. The specification makes it null — "if the result of the
// operation cannot be represented, the result is null" — and cql-execution
// agrees; the Java engine, with no bound on its BigDecimal, answers the number
// (D11 of the division area records it).
func TestATruncatedQuotientPastTheDecimalRangeIsNull(t *testing.T) {
	src := "library T version '1.0'\ndefine X: 99999999999999999999.99999999 div 0.00000001\n" +
		"define Largest: 99999999999999999999.99999999 div 1\n"
	got, err := NewEngine().EvaluateLibrary(context.Background(), src, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["X"] != nil {
		t.Errorf("X = %v; want null", got["X"])
	}
	// Within the range, as both references answer it.
	assertNumber(t, got["Largest"], true, "99999999999999999999")
}

func assertNumber(t *testing.T, got fptypes.Value, wantDecimal bool, want string) {
	t.Helper()
	d, isDecimal := got.(fptypes.Decimal)
	i, isInteger := got.(fptypes.Integer)
	switch {
	case wantDecimal && isDecimal:
		if !d.Value().Equal(decimal.RequireFromString(want)) {
			t.Errorf("got Decimal %s; want %s", d.Value(), want)
		}
	case !wantDecimal && isInteger:
		if i.String() != want {
			t.Errorf("got Integer %s; want %s", i, want)
		}
	default:
		kind := "Integer"
		if wantDecimal {
			kind = "Decimal"
		}
		t.Errorf("got %T %v; want %s %s", got, got, kind, want)
	}
}
