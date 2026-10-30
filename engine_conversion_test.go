package cql

import (
	"context"
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

// evalConversion evaluates at a fixed timestamp, at -12:00, so that a string
// written at another offset shows whether a conversion moved it there.
func evalConversion(t *testing.T, expr string) string {
	t.Helper()
	v, err := NewEngine(WithEvaluationTimestamp(
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", -12*3600)))).
		EvaluateExpression(context.Background(),
			"library T version '1.0'\ndefine R: "+expr+"\n", "R", nil, nil)
	if err != nil {
		return "error: " + err.Error()
	}
	return valueString(v)
}

// TestToDateReadsAString covers the String overload. CQL 1.5.3, Appendix B,
// ToDate: "the operator expects the string to be formatted using the ISO-8601
// date representation: YYYY-MM-DD", "the string must be interpretable as a
// valid date value", with '2014-01', '2014-01-01' and '2014-01-01T12:30:00'
// ("Still January 1st, 2014, time portion will be ignored") as valid, and "If
// the input string is not formatted correctly, or does not represent a valid
// date value, the result is null". A string with a time is read as ToDateTime
// reads one, and the date is the one written (plan 014, decisions A and B);
// both reference engines answer null to every such string.
func TestToDateReadsAString(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		// cql-tests ToDate1, ToDateYearMonth, ToDateYear, ToDateIgnoresTime and
		// ToDateMalformed.
		{"ToDate('2014-01-01')", "2014-01-01"},
		{"ToDate('2014-01')", "2014-01"},
		{"ToDate('2014')", "2014"},
		{"ToDate('2014-01-01T12:05:05.955')", "2014-01-01"},
		{"ToDate('2014/01/01')", "null"},
		{"ToDate('2014-01-01') is Date", "true"},
		{"ToDate('2014-01-01T12:30:00')", "2014-01-01"},
		{"ToDate('2014-01-01T12:30:00') = @2014-01-01", "true"},
		{"ToDate('2014-01-01T12:30')", "2014-01-01"},
		{"ToDate('2014-01-01T12')", "2014-01-01"},
		{"ToDate('2014-01-01T12:30:00Z')", "2014-01-01"},
		// The date as written, not moved to the evaluation's offset of -12:00.
		{"ToDate('2014-01-01T00:30:00+14:00')", "2014-01-01"},
		{"ToDate('2014-01-01T23:30:00-05:00')", "2014-01-01"},
		{"ToDate('2014-01-01T23:30:00')", "2014-01-01"},
		{"ToDate('0001-01-01')", "0001-01-01"},
		{"ToDate('9999-12-31')", "9999-12-31"},
		{"ToDate('2016-02-29')", "2016-02-29"},
		// Not a valid date, or not a valid DateTime (decision B).
		{"ToDate('2015-02-29')", "null"},
		{"ToDate('2014-02-30')", "null"},
		{"ToDate('2014-13')", "null"},
		{"ToDate('2014-1-1')", "null"},
		{"ToDate('20140101')", "null"},
		{"ToDate(' 2014-01-01')", "null"},
		{"ToDate('2014-01-01 ')", "null"},
		{"ToDate('0000-01-01')", "null"},
		{"ToDate('')", "null"},
		{"ToDate('abc')", "null"},
		{"ToDate('2014-01-01Z')", "null"},
		{"ToDate('2014-01-01+01:00')", "null"},
		{"ToDate('2014-01-01T25:00:00')", "null"},
		{"ToDate('2014-01-01T24:00:00')", "null"},
		{"ToDate('2014-01-01T12:60')", "null"},
		{"ToDate('2014-02-30T10:00')", "null"},
		{"ToDate('2014-01-01Tabc')", "null"},
		{"ToDate('2014-01-01T12:30:00+25:00')", "null"},
		{"ToDate(null as String)", "null"},
	} {
		if got := evalConversion(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestToDateOfADateTime covers the DateTime overload. CQL 1.5.3, Appendix B,
// ToDate: "For the DateTime overload, the operator is equivalent to invoking
// date from on the DateTime value", which keeps the DateTime's own date. There
// is no Date overload; the reference translator reaches one through DateTime,
// and both reference engines answer the same Date.
func TestToDateOfADateTime(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		// cql-tests ToDateFromDateTime.
		{"ToDate(DateTime(2014, 01, 01, 12, 30, 0, 0))", "2014-01-01"},
		{"ToDate(DateTime(2014, 1, 1, 23, 0, 0, 0, -5.0))", "2014-01-01"},
		{"ToDate(@2014-01-01T23:59:59.999+14:00)", "2014-01-01"},
		{"ToDate(@2014-01-01T00:00:00.000-12:00)", "2014-01-01"},
		{"ToDate(DateTime(2014))", "2014"},
		{"ToDate(DateTime(2014, 1))", "2014-01"},
		{"ToDate(DateTime(2014, 1, 1, 12)) = @2014-01-01", "true"},
		{"ToDate(@2014-01-01)", "2014-01-01"},
		{"ToDate(null as DateTime)", "null"},
	} {
		if got := evalConversion(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestToDateOfAFHIRDateIsASystemDate covers a FHIR.date, which reaches the
// operator as a Date that carries its FHIR type: the result is a System.Date.
// `is FHIR.date` and `is System.Date` hold of any Date under the engine's
// model of primitives, so the value is asked in Go (review 1 of plan 014).
func TestToDateOfAFHIRDateIsASystemDate(t *testing.T) {
	const header = "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\n"
	patient := []byte(`{"resourceType":"Patient","id":"p","birthDate":"1980-01-01"}`)
	engine := NewEngine(WithDataProvider(snapshotData{}))
	birthDate, err := engine.EvaluateExpression(context.Background(), header+"define R: Patient.birthDate\n", "R", patient, nil)
	if err != nil || !fptypes.IsFHIRPrimitive(birthDate) {
		t.Fatalf("Patient.birthDate = %v, %v; want a FHIR.date, or the test proves nothing", birthDate, err)
	}
	v, err := engine.EvaluateExpression(context.Background(), header+"define R: ToDate(Patient.birthDate)\n", "R", patient, nil)
	if got := valueString(v); err != nil || got != "1980-01-01" || fptypes.IsFHIRPrimitive(v) {
		t.Errorf("ToDate(Patient.birthDate) = %s, %v, FHIR primitive %v; want the System.Date 1980-01-01",
			got, err, v != nil && fptypes.IsFHIRPrimitive(v))
	}
}

// TestAConversionWithoutAnOverloadIsAnError covers an argument of a type no
// overload takes. The reference translator refuses that CQL, and ToQuantity
// fails on it as well (plan 010); ConvertsTo takes Any, and answers false.
func TestAConversionWithoutAnOverloadIsAnError(t *testing.T) {
	for _, expr := range []string{"ToDate(5)", "ToDate(1 'mg')", "ToDate(@T12:00)", "ToRatio(5)", "ToRatio(1 'mg')"} {
		if got := evalConversion(t, expr); len(got) < 6 || got[:6] != "error:" {
			t.Errorf("%s = %s, want an error", expr, got)
		}
	}
	for _, tt := range []struct{ expr, want string }{
		{"ConvertsToDate(5)", "false"},
		{"ConvertsToDate(1 'mg')", "false"},
		{"ConvertsToDate(@T12:00)", "false"},
		{"ConvertsToRatio(5)", "false"},
		{"ConvertsToRatio(1 'mg')", "false"},
	} {
		if got := evalConversion(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestToRatioReadsTwoQuantities covers ToRatio. CQL 1.5.3, Appendix B:
// "The operator accepts strings using the following format:
// <quantity>:<quantity> where <quantity> is the format used to by the
// ToQuantity operator", and "If the input string is not formatted correctly,
// or cannot be interpreted as a valid Ratio value, the result is null". Each
// side is what ToQuantity reads, so a unit UCUM does not know is null (plan
// 010, decision A); the separator is the one ':' outside quotes (plan 014,
// decision C).
func TestToRatioReadsTwoQuantities(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		// cql-tests StringToRatio and StringMalformedToRatio.
		{"ToRatio('1.0 \\'mg\\':2.0 \\'mg\\'') is Ratio", "true"},
		{"ToRatio('1.0 \\'mg\\';2.0 \\'mg\\'')", "null"},
		{"ToRatio('1.0 \\'mg\\':2.0 \\'mg\\'').numerator", "1.0 'mg'"},
		{"ToRatio('1.0 \\'mg\\':2.0 \\'mg\\'').denominator", "2.0 'mg'"},
		{"ToRatio('1:2').numerator", "1 '1'"},
		{"ToRatio('1:2').denominator", "2 '1'"},
		{"ToRatio('1:2').numerator is Quantity", "true"},
		{"ToRatio('1 : 2').denominator", "2 '1'"},
		{"ToRatio('1 \\'mg\\' :2 \\'mg\\'').numerator", "1 'mg'"},
		{"ToRatio('1 \\'mg\\': 2 \\'mg\\'').denominator", "2 'mg'"},
		{"ToRatio('1 \\'mg\\':2 \\'mL\\'').denominator", "2 'mL'"},
		{"ToRatio('1.5 \\'mg/dL\\':1 \\'1\\'').numerator", "1.5 'mg/dL'"},
		{"ToRatio('1 \\'g\\':1000 \\'mg\\'').denominator", "1000 'mg'"},
		{"ToRatio('1\\'mg\\':2\\'mg\\'').numerator", "1 'mg'"},
		// A zero is a Quantity like any other: both reference engines read 1:0.
		{"ToRatio('1:0').denominator", "0 '1'"},
		{"ToRatio('0:0').numerator", "0 '1'"},
		// What ToQuantity reads on each side.
		{"ToRatio(' 1:2').numerator", "1 '1'"},
		{"ToRatio('1:2 ').denominator", "2 '1'"},
		{"ToRatio('-1:2').numerator", "-1 '1'"},
		{"ToRatio('1:-2').denominator", "-2 '1'"},
		{"ToRatio('1e2:3').numerator", "100 '1'"},
		{"ToRatio('99999999999999999999.99999999:1').numerator", "99999999999999999999.99999999 '1'"},
		{"ToRatio('1 \\'foo\\':2')", "null"},
		{"ToRatio('1 mg:2 mg')", "null"},
		{"ToRatio('1 day:2 days')", "null"},
		{"ToRatio('1L:2')", "null"},
		{"ToRatio('Inf:NaN')", "null"},
		// The separator.
		{"ToRatio('1 \\'{a:b}\\':2').numerator", "1 '{a:b}'"},
		{"ToRatio('1:2 \\'{a:b}\\'').denominator", "2 '{a:b}'"},
		{"ToRatio('1 \\'mg:2')", "null"},
		{"ToRatio('1 \\'m\\'\\'g\\':2')", "null"},
		{"ToRatio('1')", "null"},
		{"ToRatio('1:2:3')", "null"},
		{"ToRatio('1 \\'mg\\':2 \\'mg\\':3')", "null"},
		{"ToRatio('1::2')", "null"},
		{"ToRatio(':2')", "null"},
		{"ToRatio('1:')", "null"},
		{"ToRatio(':')", "null"},
		{"ToRatio('')", "null"},
		{"ToRatio(null as String)", "null"},
	} {
		if got := evalConversion(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestARatioLiteralIsARatio covers the literal, which was a String at run time.
// CQL 1.5.3, Appendix B, Ratio: "a relationship between two quantities"; its
// equality is by numerator and denominator, as the Java engine answers
// ((1:2) = (2:4) is false there). The literal is read as ToRatio reads one
// (plan 014, decision D).
func TestARatioLiteralIsARatio(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"(1 'mg':2 'mg') is Ratio", "true"},
		{"(1 'mg':2 'mg') is String", "false"},
		{"(1 'mg':2 'mg').numerator", "1 'mg'"},
		{"(1 'mg':2 'mg').denominator", "2 'mg'"},
		{"(1.0 'mg':2.0 'mg').numerator", "1.0 'mg'"},
		{"(1:2).denominator", "2 '1'"},
		{"(1 'mg':2 'mg') = (1 'mg':2 'mg')", "true"},
		{"(1 'mg':2 'mg') = (1 'mg':3 'mg')", "false"},
		{"(1:2) = (2:4)", "false"},
		{"(1 'g':2 'mg') = (1000 'mg':2 'mg')", "true"},
		{"ToRatio('1 \\'mg\\':2 \\'mg\\'') = (1 'mg':2 'mg')", "true"},
		{"ToRatio('1:2') = (1:2)", "true"},
		{"ConvertsToRatio(1 'mg':2 'mg')", "true"},
		// A ToString read back by ToRatio is the same Ratio: by value, as
		// ToString of a Quantity drops its scale (plan 014, out of scope).
		{"ToRatio(ToString(1.0 'mg':2.0 'mg')) = (1.0 'mg':2.0 'mg')", "true"},
		{"ToRatio(ToString(ToRatio('1 \\'g\\':3 \\'mL\\''))) = (1 'g':3 'mL')", "true"},
		// A literal of two quantities is what each Quantity literal is, which
		// admits a calendar keyword unquoted; ToRatio's string does not (review
		// 1 of plan 014). The Quantity literal also reads a unit UCUM does not
		// know, which the reference translator refuses: a defect of that
		// literal, in the backlog, which the Ratio literal shares.
		{"(1 day:2 days).numerator", "1 day"},
		{"(1 week:2 weeks).denominator", "2 weeks"},
		{"(1 'foo':2 'foo').numerator", "1 'foo'"},
		// ToString writes each Quantity as ToString does, its unit quoted, so a
		// calendar unit reads back: "The result of any ToString must be
		// round-trippable back to the source value" (CQL 1.5.3, Appendix B).
		{"ToString(1 'mg':2 'mg')", "1 'mg':2 'mg'"},
		{"ToRatio(ToString(1 'day':2 'days')) = (1 'day':2 'days')", "true"},
		{"ToRatio(ToString(ToRatio('1 \\'day\\':2 \\'days\\''))) = ToRatio('1 \\'day\\':2 \\'days\\'')", "true"},
		// convert to String is ToString (review 2 of plan 014).
		{"convert (1 'day':2 'days') to String", "1 'day':2 'days'"},
		{"ToRatio(convert (1 'day':2 'days') to String) = (1 'day':2 'days')", "true"},
	} {
		if got := evalConversion(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestConvertsToFollowsItsConversion holds ConvertsToDate and ConvertsToRatio
// to their conversion. CQL 1.5.3, Appendix B: "returns true if its argument is
// or can be converted to a Date [Ratio] value. See the ToDate [ToRatio]
// operator for a description of the supported conversions", false when the
// string is not formatted correctly, and null for a null argument.
func TestConvertsToFollowsItsConversion(t *testing.T) {
	strings := []string{
		"2014-01-01", "2014-01", "2014", "2014-01-01T12:05:05.955", "2014-01-01T12:30:00+14:00", "2014-01-01T",
		"2014-01-01T25:00:00", "2014-02-30", "2014/01/01", "0000-01-01", "", "abc",
		"1.0 'mg':2.0 'mg'", "1.0 'mg';2.0 'mg'", "1:2", " 1:2", "-1:2", "1e2:3", "1 'foo':2", "1 '{a:b}':2",
		"1 'mg:2", "1:2:3", ":2", "Inf:NaN", "1:128", "hello:world",
	}
	for _, s := range strings {
		lit := "'" + escapeForCQL(s) + "'"
		for _, pair := range [][2]string{{"ConvertsToDate", "ToDate"}, {"ConvertsToRatio", "ToRatio"}} {
			converts := evalConversion(t, pair[0]+"("+lit+")")
			isNot := evalConversion(t, pair[1]+"("+lit+") is not null")
			if converts != isNot {
				t.Errorf("%s(%s) = %s, but %s(%s) is not null = %s", pair[0], lit, converts, pair[1], lit, isNot)
			}
		}
	}
	for _, tt := range []struct{ expr, want string }{
		{"ConvertsToDate('2014-01-01T12:30:00')", "true"},
		{"ConvertsToDate('2014-01-01T25:00:00')", "false"},
		{"ConvertsToDate(@2014-01-01T10:00)", "true"},
		{"ConvertsToDate(@2014-01-01)", "true"},
		{"ConvertsToDate(null as String)", "null"},
		{"ConvertsToRatio('1.0 \\'mg\\':2.0 \\'mg\\'')", "true"},
		{"ConvertsToRatio(' 1:2')", "true"},
		{"ConvertsToRatio('1:128')", "true"},
		{"ConvertsToRatio('Inf:NaN')", "false"},
		{"ConvertsToRatio('hello:world')", "false"},
		{"ConvertsToRatio(null as String)", "null"},
	} {
		if got := evalConversion(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// escapeForCQL writes s inside a CQL string literal.
func escapeForCQL(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' || s[i] == '\\' {
			out = append(out, '\\')
		}
		out = append(out, s[i])
	}
	return string(out)
}

// TestConvertToDateIsNullWhereItCannot covers convert to Date. CQL 1.5.3,
// Appendix B, Convert: "If the conversion cannot be performed, the result is
// null"; the conversion is ToDate's. Both reference engines answer null to a
// string that is no date, and the date of a DateTime. A cast is no conversion,
// and stays as it was.
func TestConvertToDateIsNullWhereItCannot(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"convert '2014-01' to Date", "2014-01"},
		{"convert '2014-01-01' to System.Date", "2014-01-01"},
		{"convert '2014-01-01T12:00:00' to Date", "2014-01-01"},
		{"convert 'abc' to Date", "null"},
		{"convert '2014/01/01' to Date", "null"},
		{"convert @2014-01-01T10:00 to Date", "2014-01-01"},
		{"convert (null as String) to Date", "null"},
		{"cast '2014-01-01' as Date", "2014-01-01"},
		// convert to Ratio is ToRatio: the conversion from String to Ratio is
		// explicit (CQL 1.5.3, Appendix B, Convert, Table 9-E).
		{"(convert '1:2' to Ratio).denominator", "2 '1'"},
		{"(convert '1 \\'mg\\':2 \\'mg\\'' to System.Ratio).numerator", "1 'mg'"},
		{"convert 'abc' to Ratio", "null"},
		{"convert (null as String) to Ratio", "null"},
	} {
		if got := evalConversion(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
	// A cast does not convert: a string that is no Date stays an error.
	for _, expr := range []string{"cast 'abc' as Date", "cast '2014-01-01T10:00' as Date"} {
		if got := evalConversion(t, expr); len(got) < 6 || got[:6] != "error:" {
			t.Errorf("%s = %s, want an error", expr, got)
		}
	}
}
