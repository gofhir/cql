package cql

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gofhir/cql/eval"
)

// hugeAndTinyProvider returns two Observations whose values are past what a
// float64 holds, one way and the other. FHIR data can carry them; a CQL literal
// cannot, being limited to 28 integer and 8 fractional digits.
type hugeAndTinyProvider struct{}

func (hugeAndTinyProvider) Retrieve(_ context.Context, r eval.RetrieveRequest) ([]json.RawMessage, error) {
	if r.ResourceType != "Observation" {
		return nil, nil
	}
	huge := "1" + strings.Repeat("0", 320)
	tiny := "0." + strings.Repeat("0", 400) + "1"
	return []json.RawMessage{
		json.RawMessage(`{"resourceType":"Observation","id":"huge","status":"final","code":{"text":"x"},"valueQuantity":{"value":` + huge + `,"unit":"g"}}`),
		json.RawMessage(`{"resourceType":"Observation","id":"tiny","status":"final","code":{"text":"x"},"valueQuantity":{"value":` + tiny + `,"unit":"g"}}`),
	}, nil
}

// TestDecimalFunctionsOutsideAFloatAreNull covers results a float64 or an
// Integer cannot hold. Each of these built a Decimal from the infinity or NaN the float became,
// and decimal.NewFromFloat panics on that, crashing the evaluation: a negative
// base to a fractional power, a power past the range, and Log, Ln and StdDev of
// values out of it, which FHIR data can carry. There is no number to answer, as
// Power already said, so each is null.
func TestDecimalFunctionsOutsideAFloatAreNull(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1' called FH\n" +
		"context Patient\n" +
		"define Values: [Observation] O return (O.value as Quantity).value\n" +
		"define NegativeRoot: (-8.0) ^ 0.5\n" +
		"define PowerOverflow: 2.0 ^ 2000.0\n" +
		"define LogHuge: Log(First(Values), 10.0)\n" +
		"define LnHuge: Ln(First(Values))\n" +
		"define LnTiny: Ln(Last(Values))\n" +
		"define StdDevHuge: StdDev({First(Values), 0.0})\n" +
		"define Quantities: [Observation] O return O.value as Quantity\n" +
		"define StdDevQuantities: StdDev(Quantities)\n" +
		"define PopulationStdDevQuantities: PopulationStdDev(Quantities)\n" +
		"define DivHuge: First(Values) div 3.0\n"
	e := NewEngine(WithDataProvider(hugeAndTinyProvider{}))
	for _, def := range []string{"NegativeRoot", "PowerOverflow", "StdDevHuge",
		"StdDevQuantities", "PopulationStdDevQuantities", "DivHuge"} {
		v, err := e.EvaluateExpression(context.Background(), src, def, []byte(`{"resourceType":"Patient","id":"p"}`), nil)
		if err != nil || v != nil {
			t.Errorf("%s = %v, %v; want null", def, v, err)
		}
	}
}

// TestDivTruncatesExactly covers a quotient just below a whole number. Div
// rounds to 16 places before the integer part is taken, so the quotient became
// the whole number; the reference engine answers 0.
func TestDivTruncatesExactly(t *testing.T) {
	for expr, want := range map[string]string{
		"99999999999999999999.99999999 div 100000000000000000000.0":         "0",
		"99999999999999999999.99999999 'g' div 100000000000000000000.0 'g'": "0 'g'",
	} {
		v, err := NewEngine().EvaluateExpression(context.Background(), "library T version '1.0'\ndefine R: "+expr+"\n", "R", nil, nil)
		if err != nil || valueString(v) != want {
			t.Errorf("%s = %v, %v; want %s", expr, v, err, want)
		}
	}
}

// TestMedianKeepsAZeroWrittenWithAPoint covers a zero written 0.0. Median
// kept a value only if it read as non-zero or printed as "0", which a Decimal
// zero does not, so a lab result of 0.0 dropped out of it.
func TestMedianKeepsAZeroWrittenWithAPoint(t *testing.T) {
	v, err := NewEngine().EvaluateExpression(context.Background(),
		"library T version '1.0'\ndefine R: Median({0.0, 1.0, 2.0, 3.0})\n", "R", nil, nil)
	if err != nil || valueString(v) != "1.5" {
		t.Errorf("Median({0.0, 1.0, 2.0, 3.0}) = %v, %v; want 1.5", v, err)
	}
}

// TestTheLogarithmOfADecimalPastAFloatIsStillOne covers Ln and Log of values a
// float64 does not hold. They have logarithms — ln(1e320) is 736.8…, and
// log10 of it is 320, and ln(1e-401) is -923.3… — and converting the decimal to a float first made it
// infinity or zero, and the answer null.
func TestTheLogarithmOfADecimalPastAFloatIsStillOne(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1' called FH\n" +
		"context Patient\n" +
		"define Values: [Observation] O return (O.value as Quantity).value\n" +
		"define LogHuge: Log(First(Values), 10.0)\n" +
		"define LnHuge: Round(Ln(First(Values)), 3)\n" +
		"define LnTiny: Round(Ln(Last(Values)), 3)\n"
	e := NewEngine(WithDataProvider(hugeAndTinyProvider{}))
	for def, want := range map[string]string{"LogHuge": "320", "LnHuge": "736.827", "LnTiny": "-923.337"} {
		v, err := e.EvaluateExpression(context.Background(), src, def, []byte(`{"resourceType":"Patient","id":"p"}`), nil)
		if err != nil || valueString(v) != want {
			t.Errorf("%s = %v, %v; want %s", def, v, err, want)
		}
	}
}

// TestPowerAndTheOperatorAgree covers powers, through `^` and through Power,
// which are one implementation. A whole-number exponent is computed exactly —
// 0.21 ^ 2 is 0.0441 as 0.21 * 0.21 is, and 3L ^ 35L is exact past what a
// float holds — zero to the zero is one whatever zero it is, and a power past
// its type — 32 bits for an Integer, 64 for a Long, the largest Decimal — is
// null. Each was answered
// differently by the operator and the function, or wrapped a value in silence.
func TestPowerAndTheOperatorAgree(t *testing.T) {
	for _, tt := range []struct{ base, exponent, want string }{
		{"0.21", "2", "0.0441"},
		{"0.0", "0", "1"},
		{"0", "0", "1"},
		{"2", "-1", "0.5"},
		{"3L", "35L", "50031545098999707"},
		{"3", "35", "null"},
		{"2", "30", "1073741824"},
		{"2", "31", "null"},
		{"(-2)", "3", "-8"},
		{"1", "-5000", "1"},
		{"2L", "63L", "null"},
		{"2L", "62L", "4611686018427387904"},
		{"10.0", "21", "null"},
		{"1.1", "2000", "null"},
		{"2.0", "18446744073709551616.0", "null"},
		{"(-8.0)", "0.5", "null"},
		{"2.0", "0.5", "1.41421356"},
		// The float of the largest Decimal rounds to 1e20, which is past it.
		{"10.0", "20", "null"},
		// The sign of minus one from the exponent's parity, which a float past
		// 2^53 no longer keeps.
		{"(-1)", "9007199254740993L", "-1"},
		{"(-1.0)", "18446744073709551617.0", "-1"},
		// Past the largest Decimal exactly, though the float lands under it.
		{"0.1", "-20", "null"},
		{"0.1", "-19", "10000000000000000000"},
	} {
		for _, expr := range []string{tt.base + " ^ " + tt.exponent, "Power(" + tt.base + ", " + tt.exponent + ")"} {
			v, err := NewEngine().EvaluateExpression(context.Background(), "library T version '1.0'\ndefine R: "+expr+"\n", "R", nil, nil)
			got := "null"
			if v != nil {
				got = valueString(v)
			}
			if err != nil || got != tt.want {
				t.Errorf("%s = %s, %v; want %s", expr, got, err, tt.want)
			}
		}
	}
}

// TestANegativeIntegerPowerIsTypedAsTheDecimalItIs covers 2 ^ -1, which is the
// Decimal 0.5 and was typed as an Integer: what the result was used for then
// expected a whole number, and an index or a Take of it misread it.
func TestANegativeIntegerPowerIsTypedAsTheDecimalItIs(t *testing.T) {
	for expr, want := range map[string]string{
		"2 ^ -1 is Decimal": "true",
		"2 ^ 3 is Integer":  "true",
		"2 ^ -0 is Integer": "true",
		// The function as the operator.
		"Power(2, -1) is Decimal": "true",
		"Power(2, 3) is Integer":  "true",
		"(2 ^ -1) + 1":            "1.5",
	} {
		v, err := NewEngine().EvaluateExpression(context.Background(), "library T version '1.0'\ndefine R: "+expr+"\n", "R", nil, nil)
		if err != nil || valueString(v) != want {
			t.Errorf("%s = %v, %v; want %s", expr, v, err, want)
		}
	}
}

// TestThePopulationVarianceOfOneValueIsZero covers a list of one value. The
// population variance divides by n and is 0 for one value, as the reference
// engine answers; only the sample variance, which divides by n - 1, has nothing
// to say about one. Both population aggregates answered null.
func TestThePopulationVarianceOfOneValueIsZero(t *testing.T) {
	for expr, want := range map[string]string{
		"PopulationVariance({3.0})":           "0",
		"PopulationStdDev({3.0})":             "0",
		"PopulationVariance({3.0 'mg'})":      "0 'mg2'",
		"Variance({3.0})":                     "null",
		"StdDev({3.0})":                       "null",
		"PopulationVariance(List<Decimal>{})": "null",
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

// TestAnAggregateOfOnlyNullsIsNull covers a list whose elements are all null.
// "If the source contains no non-null elements, null is returned", and Sum
// answered 0 and Product 1 — the empty sum and product, standing in for an
// answer the list does not have.
func TestAnAggregateOfOnlyNullsIsNull(t *testing.T) {
	for _, expr := range []string{
		"Sum({null as Integer})", "Sum({null as Decimal, null as Decimal})",
		"Product({null as Integer})", "Product({null as Decimal})",
		"Avg({null as Decimal})", "Median({null as Decimal})",
	} {
		v, err := NewEngine().EvaluateExpression(context.Background(), "library T version '1.0'\ndefine R: "+expr+"\n", "R", nil, nil)
		if err != nil || v != nil {
			t.Errorf("%s = %v, %v; want null", expr, v, err)
		}
	}
}

// TestAnUncertainResultWithAnUnansweredBoundIsNull covers arithmetic on an
// uncertainty — months between two years is anything from 0 to 23 — where one
// combination of bounds has no answer: zero to a negative power, a power past
// the largest Decimal. Dropping that combination made the others a definite
// answer, and (months between @2020 and @2021) ^ -1 was 0.04347826.
func TestAnUncertainResultWithAnUnansweredBoundIsNull(t *testing.T) {
	for _, expr := range []string{
		"(months between @2020 and @2021) ^ -1",
		"(months between @2020 and @2021) ^ 63",
		"3 / (months between @2020 and @2021)",
	} {
		v, err := NewEngine().EvaluateExpression(context.Background(), "library T version '1.0'\ndefine R: "+expr+"\n", "R", nil, nil)
		if err != nil || v != nil {
			t.Errorf("%s = %v, %v; want null", expr, v, err)
		}
	}
}

// TestAStringIsReadAsADecimalAtTheScale covers ToDecimal of a string carrying
// more places than the type keeps. A literal cannot, and the string's extra
// digits reached the arithmetic as no Decimal written in CQL can: the reference
// engine floors it to eight places, as its arithmetic floors.
func TestAStringIsReadAsADecimalAtTheScale(t *testing.T) {
	for expr, want := range map[string]string{
		"ToDecimal('1.000000005')":                      "1.00000000",
		"ToDecimal('-1.000000005')":                     "-1.00000001",
		"ToDecimal('1.12345678')":                       "1.12345678",
		"Log(10, ToDecimal('0.999999999999999999999'))": "-230258506.99111846",
		"Log(10, ToDecimal('1.000000001'))":             "null",
		"ToDecimal('99999999999999999999.999999999')":   "null",
		"ToDecimal('-99999999999999999999.999999999')":  "null",
		"ToString(ToDecimal('2.50000000000'))":          "2.50000000",
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

// TestAStringWithAHugeExponentIsAnsweredAtOnce covers exponents no Decimal
// reaches. Comparing or flooring one rescaled it to a number of a billion
// digits, and the evaluation never returned; the reference engine answers each
// at once. The floor below the scale is its answer to '-1e-10001',
// -0.00000001 (D132 of testdata/snapshots/decimalstrings.cql).
func TestAStringWithAHugeExponentIsAnsweredAtOnce(t *testing.T) {
	for expr, want := range map[string]string{
		"ToDecimal('1e999999999')":         "null",
		"ToDecimal('0e999999999')":         "0",
		"ToDecimal('1e-999999999')":        "0.00000000",
		"ToDecimal('-1e-999999999')":       "-0.00000001",
		"ConvertsToDecimal('1e999999999')": "false",
		"ToDecimal('1.5E2')":               "150",
	} {
		start := time.Now()
		v, err := NewEngine().EvaluateExpression(context.Background(), "library T version '1.0'\ndefine R: "+expr+"\n", "R", nil, nil)
		got := "null"
		if v != nil {
			got = valueString(v)
		}
		if err != nil || got != want {
			t.Errorf("%s = %s, %v; want %s", expr, got, err, want)
		}
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Errorf("%s took %s", expr, elapsed)
		}
	}
}
