package cql

import (
	"context"
	"testing"
	"time"
)

// The tests of plan 019 (docs/plans/019-date-promotion.md): a Date next to a
// DateTime is converted implicitly.
//
// CQL 1.5.3 (https://cql.hl7.org/N1A/), Developer's Guide, Implicit Conversions,
// table 3-J: from Date to DateTime the conversion is "Implicit", and
// "ToDateTime(Date) Converts a Date value to a DateTime with all time
// components unspecified and the timezone offset of the request". The
// expected values are the ones both reference engines give (Java 5.4.0 and
// cql-execution, scratchpad r019: probe019.java.json, probe019.js.json,
// probe019c.java.json, probe019p.java.json), read as the type of the result:
// this engine writes a DateTime of a day as the Date it looks like, so the
// tests ask `is DateTime`.

// promotionAnswer evaluates `define R: <expr>` after the declarations, at the
// offset -03:00, on a Patient born 1980-01-01. It is compiled, not imported:
// the ELM this engine exports does not write the implicit ToDateTime (plan 019,
// out of scope, backlog 1c37).
func promotionAnswer(t *testing.T, decls, expr string) string {
	t.Helper()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", -3*3600))
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1' called FHIRHelpers\n" +
		decls + "context Patient\ndefine R: " + expr + "\n"
	patient := []byte(`{"resourceType":"Patient","id":"p","birthDate":"1980-01-01"}`)
	v, err := NewEngine(WithEvaluationTimestamp(at)).EvaluateExpression(context.Background(), src, "R", patient, nil)
	if err != nil {
		return "error: " + err.Error()
	}
	return valueString(v)
}

// TestADateNextToADateTimeIsADateTime holds the rows of the probe where the
// translator inserts ToDateTime and both references answer a DateTime where
// this engine answered a Date: a list, an if, a case, Coalesce, an argument
// (D38, D42, D44, D46, D88, D89).
func TestADateNextToADateTimeIsADateTime(t *testing.T) {
	const (
		d  = "@2020-03-01"
		d2 = "@2020-03-05"
		dt = "@2020-03-01T10:00:00.000"
	)
	const fns = "define function F(x DateTime): x\ndefine function G(x Interval<DateTime>): x\n"
	for _, tt := range []struct{ decls, expr, want string }{
		// D38, D39: both elements of a list are DateTimes.
		{"", "({ " + d + ", " + dt + " })[0] is DateTime", "true"},
		{"", "({ " + d + ", " + dt + " })[1] is DateTime", "true"},
		{"", "({ " + dt + ", " + d + " })[1] is DateTime", "true"},
		{"", "First({ " + d + ", " + dt + " }) is DateTime", "true"},
		// D42, D43: an if.
		{"", "(if true then " + d + " else " + dt + ") is DateTime", "true"},
		{"", "(if false then " + dt + " else " + d + ") is DateTime", "true"},
		// D44, D45: a case.
		{"", "(case when true then " + d + " else " + dt + " end) is DateTime", "true"},
		// D46, D47: Coalesce.
		{"", "Coalesce(" + d + ", " + dt + ") is DateTime", "true"},
		{"", "Coalesce(null, " + dt + ", " + d + ") is DateTime", "true"},
		// elmconv-c D2: Coalesce of lists.
		{"", "Coalesce({ " + d + " }, { " + dt + " })[0] is DateTime", "true"},
		// elmconv-c D3, D4: nested lists, and intervals in a list.
		{"", "({ { " + d + " }, { " + dt + " } })[0][0] is DateTime", "true"},
		{"", "start of ({ Interval[" + d + ", " + d + "], Interval[" + dt + ", " + dt + "] })[0] is DateTime", "true"},
		// elmconv-c D7 to D11: queries convert the Date they read.
		{"", "({ " + d + " }) X return X < " + dt, "{null}"},
		{"", "({ " + d + ", " + dt + " }) X return X is DateTime", "{true}"},
		// D88: a function with a DateTime parameter converts its argument.
		{fns, "F(" + d + ") is DateTime", "true"},
		// D89: and one with an Interval<DateTime> parameter.
		{fns, "start of G(Interval[" + d + ", " + d2 + "]) is DateTime", "true"},
		{fns, "end of G(Interval[" + d + ", " + d2 + "]) is DateTime", "true"},
		// Controls: a DateTime argument stays one, and a Date next to a Date stays one.
		{fns, "F(" + dt + ") is DateTime", "true"},
		{"", "First({ " + d + ", " + d2 + " }) is Date", "true"},
	} {
		t.Run(tt.expr, func(t *testing.T) {
			if got := promotionAnswer(t, tt.decls, tt.expr); got != tt.want {
				t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
			}
		})
	}
}

// TestACoalesceWithANullFirstAndAUnionOfListsAreNotConverted holds the forms the
// translator does not convert, which it resolves to a Choice with As: Coalesce
// with a null first answers the first value as it is (elmconv-c D0, D1: a Date,
// then a DateTime), and union, intersect and except of lists keep a Date a Date
// (elmconv.tsv D70 to D73, elmconv-c D17: `{Date 2020-03-01, DateTime ...}`).
func TestACoalesceWithANullFirstAndAUnionOfListsAreNotConverted(t *testing.T) {
	const (
		d  = "@2020-03-01"
		dt = "@2020-03-01T10:00:00.000"
	)
	for _, tt := range []struct{ expr, want string }{
		{"Coalesce(null, " + d + ", " + dt + ") is Date", "true"},
		{"Coalesce(null, " + d + ", " + dt + ") is DateTime", "false"},
		{"Coalesce(null, " + dt + ", " + d + ") is DateTime", "true"},
		{"({ " + d + " } union { " + dt + " })[0] is Date", "true"},
		{"({ " + d + " } union { " + dt + " })[1] is DateTime", "true"},
		{"({ " + dt + " } union { " + d + " })[0] is DateTime", "true"},
		{"({ " + dt + " } union { " + d + " })[1] is Date", "true"},
		{"({ " + d + " } except { " + dt + " })[0] is Date", "true"},
		// The tuples of a list are not converted (elmconv-c D6).
		{"(({ Tuple { a: " + d + " }, Tuple { a: " + dt + " } })[0]).a is Date", "true"},
	} {
		t.Run(tt.expr, func(t *testing.T) {
			if got := promotionAnswer(t, "", tt.expr); got != tt.want {
				t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
			}
		})
	}
}

// TestTheDefaultOfATypedParameterIsConverted covers probe019p: the default of a
// parameter with a declared type is converted to it, as the translator writes
// ToDateTime into the default (`parameter MP Interval<DateTime> default
// Interval[@2020-03-01, @2020-03-05]`): both references answer DateTimes.
func TestTheDefaultOfATypedParameterIsConverted(t *testing.T) {
	// N and PF name a parameter and a function declared after them: a default
	// is read once the whole library is known (code review 1 of plan 019).
	const decls = "parameter MP Interval<DateTime> default Interval[@2020-03-01, @2020-03-05]\n" +
		"parameter P DateTime default @2020-03-01\n" +
		"parameter Q default @2020-03-01\n" +
		"parameter N Integer default M\n" +
		"parameter M Integer default 1\n" +
		"parameter PF DateTime default G(@2020-03-01)\n" +
		"define function G(x DateTime): x\n"
	for _, tt := range []struct{ expr, want string }{
		{"start of MP is DateTime", "true"},
		{"end of MP is DateTime", "true"},
		{"P is DateTime", "true"},
		// A parameter with no declared type is the type of its default: a Date.
		{"Q is Date", "true"},
		{"N", "1"},
		// The argument of G is a DateTime (Developer's Guide, Implicit Conversions).
		{"PF is DateTime", "true"},
	} {
		t.Run(tt.expr, func(t *testing.T) {
			if got := promotionAnswer(t, decls, tt.expr); got != tt.want {
				t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
			}
		})
	}
}

// TestAFHIRDateNextToADateTimeIsConverted covers plan 019, M2: a FHIR.date reads
// as a Date through FHIRHelpers.ToDate, and then as a DateTime next to one
// (table 3-J of the Developer's Guide). The reference translator refuses the
// signature (`Could not resolve call to operator Equal with signature
// (FHIR.date, System.DateTime)`: no reference answers), so the expected value
// is the specification's: ToDateTime of the Date, a DateTime of a day.
func TestAFHIRDateNextToADateTimeIsConverted(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"(if true then Patient.birthDate else @2020-01-01T10:00:00.000) is DateTime", "true"},
		{"({ Patient.birthDate, @2020-01-01T10:00:00.000 })[0] is DateTime", "true"},
		// 1980-01-01 as a DateTime of a day is before 1980-01-02T10:00 whatever its hour.
		{"Patient.birthDate before @1980-01-02T10:00:00.000", "true"},
		{"Patient.birthDate after @1979-12-31T10:00:00.000", "true"},
	} {
		t.Run(tt.expr, func(t *testing.T) {
			if got := promotionAnswer(t, "", tt.expr); got != tt.want {
				t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
			}
		})
	}
}

// TestAValueWithNoStaticTypeIsNotConverted fixes what a value with no static type
// gives, so that it does not change by accident: the call into an included
// library is typed Unknown by sema (plan 019, out of scope, backlog), so the
// Date it answers is not promoted and compares to a DateTime as a Date does.
func TestAValueWithNoStaticTypeIsNotConverted(t *testing.T) {
	const date = "FHIRHelpers.ToDate(FHIR.date { value: @2020-03-01 })"
	for _, tt := range []struct{ expr, want string }{
		{date + " is Date", "true"},
		{date + " is DateTime", "false"},
		{date + " = @2020-03-01T10:00:00.000", "null"},
		{"(if true then " + date + " else @2020-03-01T10:00:00.000) is Date", "true"},
	} {
		t.Run(tt.expr, func(t *testing.T) {
			if got := promotionAnswer(t, "", tt.expr); got != tt.want {
				t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
			}
		})
	}
}
