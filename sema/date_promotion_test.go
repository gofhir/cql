package sema_test

import (
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/gofhir/cql/compiler"
	"github.com/gofhir/cql/model"
	"github.com/gofhir/cql/sema"
)

// The tests of plan 019 (docs/plans/019-date-promotion.md): a Date next to a
// DateTime is converted implicitly, and sema records the conversion against the
// Date side, with no function (like a numeric promotion).
//
// CQL 1.5.3 (https://cql.hl7.org/N1A/), Developer's Guide, Implicit Conversions,
// table 3-J: Date to DateTime is "Implicit", and "ToDateTime(Date) Converts a
// Date value to a DateTime with all time components unspecified and the
// timezone offset of the request". Where the reference translator (cql-to-elm
// 5.4.0) inserts ToDateTime is in scratchpad r019 elmconv.tsv and elmconv-c.tsv,
// and the table of every operator below follows it.

var (
	modelOnce sync.Once
	modelInfo sema.Model
)

// datePromotions compiles a library that holds one definition, R, after the
// declarations given, and answers the type of every node recorded as converted
// from a Date to a DateTime with no function, sorted.
func datePromotions(t *testing.T, declarations, expr string) []string {
	t.Helper()
	modelOnce.Do(func() {
		mi, err := model.LoadR4ModelInfo()
		if err != nil {
			t.Fatalf("loading model info: %v", err)
		}
		modelInfo = sema.FromModelInfo(mi)
	})
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1' called FHIRHelpers\n" +
		declarations + "define R: " + expr + "\n"
	lib, err := compiler.Compile(src)
	if err != nil {
		t.Fatalf("compiling %s: %v", expr, err)
	}
	result := sema.Check(lib, modelInfo)
	var out []string
	for node, conv := range result.Conversions {
		from := result.Types[node]
		if conv.Function != "" || from == nil || conv.Target == nil {
			continue
		}
		f := from.String()
		if strings.Contains(f, "System.Date") && !strings.Contains(f, "DateTime") && strings.Contains(conv.Target.String(), "System.DateTime") {
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out
}

const (
	dateLit  = "@2020-03-01"
	dateLit2 = "@2020-03-05"
	dtLit    = "@2020-03-01T10:00:00.000"
	dtLit2   = "@2020-03-05T10:00:00.000"
	// the types the conversion is recorded against
	tDate     = "System.Date"
	tDateList = "List<System.Date>"
	tDateIv   = "Interval<System.Date>"
)

// TestADateNextToADateTimeIsRecordedAsPromoted holds, per operator in the scope
// of plan 019, that the conversion is recorded on the Date side, and that it is
// recorded nowhere else. The rows are the ones the reference translator
// converts (elmconv.tsv D0 to D89, elmconv-c.tsv).
func TestADateNextToADateTimeIsRecordedAsPromoted(t *testing.T) {
	d, d2, dt, dt2 := dateLit, dateLit2, dtLit, dtLit2
	ivD := "Interval[" + d + ", " + d2 + "]"
	ivDT := "Interval[" + dt + ", " + dt2 + "]"
	type row struct {
		expr string
		want []string
	}
	rows := make([]row, 0, 128)
	// Comparison (D0 to D11): both orders.
	for _, op := range []string{"=", "!=", "~", "!~", "<", "<=", ">", ">="} {
		rows = append(rows,
			row{d + " " + op + " " + dt, []string{tDate}},
			row{dt + " " + op + " " + d, []string{tDate}})
	}
	rows = append(rows,
		// between (D12, D13): the operand, or the two bounds.
		row{d + " between " + dt + " and " + dt2, []string{tDate}},
		row{dt + " between " + d + " and " + d2, []string{tDate, tDate}},
		// Time phrases (D14 to D25).
		row{d + " same day as " + dt, []string{tDate}},
		row{dt + " same day as " + d, []string{tDate}},
		row{d + " before " + dt, []string{tDate}},
		row{dt + " before " + d, []string{tDate}},
		row{d + " after day of " + dt, []string{tDate}},
		row{d + " same or before " + dt, []string{tDate}},
		row{d + " within 3 days of " + dt, []string{tDate}},
		row{dt + " within 3 days of " + d, []string{tDate}},
		row{d + " 1 day or less before " + dt, []string{tDate}},
		row{dt + " 1 day or less before " + d, []string{tDate}},
		// A point and an interval (D26, D28, D30, D32, D84).
		row{d + " in " + ivDT, []string{tDate}},
		row{d + " during " + ivDT, []string{tDate}},
		row{ivDT + " includes " + d, []string{tDate}},
		row{ivDT + " contains " + d, []string{tDate}},
		row{d + " in day of " + ivDT, []string{tDate}},
		// A point and a list (D34 to D37).
		row{d + " in { " + dt + " }", []string{tDate}},
		row{dt + " in { " + d + " }", []string{tDateList}},
		row{"{ " + dt + " } contains " + d, []string{tDate}},
		row{"{ " + d + " } contains " + dt, []string{tDateList}},
		// A literal list, interval, if, case and Coalesce (D38 to D47).
		row{"{ " + d + ", " + dt + " }", []string{tDate}},
		row{"{ " + dt + ", " + d + " }", []string{tDate}},
		row{"Interval[" + d + ", " + dt + "]", []string{tDate}},
		row{"Interval[" + dt + ", " + d + "]", []string{tDate}},
		row{"if true then " + d + " else " + dt, []string{tDate}},
		row{"if true then " + dt + " else " + d, []string{tDate}},
		row{"case when true then " + d + " else " + dt + " end", []string{tDate}},
		row{"case when true then " + dt + " else " + d + " end", []string{tDate}},
		row{"Coalesce(" + d + ", " + dt + ")", []string{tDate}},
		row{"Coalesce(" + dt + ", " + d + ")", []string{tDate}},
		// Lists, intervals inside a list, and a null among them (elmconv-c).
		row{"{ { " + d + " }, { " + dt + " } }", []string{tDateList}},
		row{"{ Interval[" + d + ", " + d + "], Interval[" + dt + ", " + dt + "] }", []string{tDateIv}},
		row{"Coalesce({ " + d + " }, { " + dt + " })", []string{tDateList}},
		row{"{ " + d + ", null, " + dt + " }", []string{tDate}},
		row{"Interval[" + d + ", null] contains " + dt, []string{tDateIv}},
		row{"{ " + d + ", " + dt + " } includes " + d, []string{tDate, tDate}},
		row{d + " in { " + dt + ", null }", []string{tDate}},
		row{"collapse { Interval[" + d + ", " + d + "], Interval[" + dt + ", " + dt + "] }", []string{tDateIv}},
		row{"expand { Interval[" + d + ", " + d + "] } per day = { Interval[" + dt + ", " + dt + "] }", []string{"List<Interval<System.Date>>"}},
		// Aggregates over a mixed list (D82, D83, elmconv-c D13 to D18).
		row{"Max({ " + d + ", " + dt + " })", []string{tDate}},
		row{"Min({ " + d + ", " + dt + " })", []string{tDate}},
		row{"Count({ " + d + ", " + dt + " })", []string{tDate}},
		row{"Mode({ " + d + ", " + dt + " })", []string{tDate}},
		row{"First({ " + d + ", " + dt + " })", []string{tDate}},
		row{"distinct { " + d + ", " + dt + " }", []string{tDate}},
		// Queries (elmconv-c D7 to D11).
		row{"({ " + d + " }) X where X < " + dt, []string{tDate}},
		row{"({ " + d + " }) X return X = " + dt, []string{tDate}},
		row{"({ " + d + " }) X let Y: " + dt + " return X < Y", []string{tDate}},
		row{"({ " + dt + " }) X where X > " + d, []string{tDate}},
		// Lists compared (D74, D75).
		row{"{ " + d + " } = { " + dt + " }", []string{tDateList}},
		row{"{ " + dt + " } = { " + d + " }", []string{tDateList}},
		// Differences and durations (D76 to D81).
		row{"days between " + d + " and " + dt, []string{tDate}},
		row{"days between " + dt + " and " + d, []string{tDate}},
		row{"difference in days between " + d + " and " + dt, []string{tDate}},
		row{"difference in days between " + dt + " and " + d, []string{tDate}},
		row{"duration in days of Interval[" + d + ", " + dt + "]", []string{tDate}},
		row{"duration in days of Interval[" + dt + ", " + d + "]", []string{tDate}},
	)
	// Two intervals (D48 to D69, D86, D87).
	for _, op := range []string{"overlaps", "meets", "starts", "ends", "during", "includes", "properly includes",
		"=", "union", "intersect", "except", "overlaps day of", "meets before", "included in"} {
		rows = append(rows,
			row{ivD + " " + op + " " + ivDT, []string{tDateIv}},
			row{ivDT + " " + op + " " + ivD, []string{tDateIv}})
	}
	for _, r := range rows {
		t.Run(r.expr, func(t *testing.T) {
			got := datePromotions(t, "", r.expr)
			if strings.Join(got, ",") != strings.Join(r.want, ",") {
				t.Errorf("%s: promotions recorded on %v, want %v", r.expr, got, r.want)
			}
		})
	}
}

// TestADateIsNotPromotedWhereTheTranslatorDoesNotConvert holds the places where
// no conversion is recorded: the reference translator resolves a union,
// intersect and except of lists with As to a Choice (elmconv.tsv D70 to D73,
// elmconv-c D17), Coalesce with a null first to a Choice (elmconv-c D0, D1),
// writes no conversion into a tuple (elmconv-c D6), and nothing is converted
// where there is no static type.
func TestADateIsNotPromotedWhereTheTranslatorDoesNotConvert(t *testing.T) {
	d, dt := dateLit, dtLit
	for _, expr := range []string{
		"{ " + d + " } union { " + dt + " }",
		"{ " + dt + " } union { " + d + " }",
		"{ " + d + " } intersect { " + dt + " }",
		"{ " + dt + " } intersect { " + d + " }",
		"{ " + d + " } except { " + dt + " }",
		"Coalesce(null, " + d + ", " + dt + ")",
		"Coalesce(null, " + dt + ", " + d + ")",
		"{ Tuple { a: " + d + " }, Tuple { a: " + dt + " } }",
		// The signatures the translator refuses (elmconv.tsv D27, D29, D31, D33,
		// D85): a point as the container, an interval as the element. They are a
		// front of their own (plan 019, out of scope) and stay as they were.
		"Interval[" + dt + ", " + dtLit2 + "] in " + d,
		"Interval[" + dt + ", " + dtLit2 + "] during " + d,
		d + " includes Interval[" + dt + ", " + dtLit2 + "]",
		d + " contains Interval[" + dt + ", " + dtLit2 + "]",
		// Controls: the same type on both sides has nothing to convert.
		d + " = " + d,
		dt + " = " + dt,
		"{ " + d + ", " + d + " }",
		"Interval[" + d + ", " + d + "] overlaps Interval[" + d + ", " + d + "]",
		// No static type: a call into an included library is Unknown (plan 019, out
		// of scope), and so is a value with a type this phase cannot work out.
		"FHIRHelpers.ToDate(FHIR.date { value: " + d + " }) = " + dt,
		"FHIRHelpers.ToDate(FHIR.date { value: " + d + " }) in " + "Interval[" + dt + ", " + dt + "]",
	} {
		t.Run(expr, func(t *testing.T) {
			if got := datePromotions(t, "", expr); len(got) != 0 {
				t.Errorf("%s: promotions recorded on %v, want none", expr, got)
			}
		})
	}
}

// TestAFunctionArgumentAndAParameterDefaultAreConverted covers elmconv.tsv D88,
// D89 (a function of the same library with a DateTime or Interval<DateTime>
// parameter converts its Date argument) and probe019p (the default of a
// parameter with a declared type converts to that type).
func TestAFunctionArgumentAndAParameterDefaultAreConverted(t *testing.T) {
	const fns = "define function F(x DateTime): x\ndefine function G(x Interval<DateTime>): x\n"
	for _, tt := range []struct {
		decl, expr string
		want       []string
	}{
		{fns, "F(" + dateLit + ")", []string{tDate}},
		{fns, "G(Interval[" + dateLit + ", " + dateLit2 + "])", []string{tDateIv}},
		{fns, "F(" + dtLit + ")", nil},
		{"parameter MP Interval<DateTime> default Interval[" + dateLit + ", " + dateLit2 + "]\n", "MP", []string{tDateIv}},
		{"parameter P DateTime default " + dateLit + "\n", "P", []string{tDate}},
		// A parameter of no declared type takes the type of its default.
		{"parameter P default " + dateLit + "\n", "P", nil},
		{"parameter P DateTime default " + dtLit + "\n", "P", nil},
	} {
		t.Run(tt.decl+tt.expr, func(t *testing.T) {
			got := datePromotions(t, tt.decl, tt.expr)
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("promotions recorded on %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAModelFunctionIsFollowedByTheDatePromotion covers the FHIR path (plan
// 019, M2): a FHIR.date reaches a Date through FHIRHelpers.ToDate, and a Date
// next to a DateTime reaches a DateTime. The two are one conversion, the
// function and then the promotion, as FHIR.integer reaching a Decimal is.
// Table 3-J of the Developer's Guide gives the second step; the first is the
// model's conversionInfo.
func TestAModelFunctionIsFollowedByTheDatePromotion(t *testing.T) {
	modelOnce.Do(func() {
		mi, err := model.LoadR4ModelInfo()
		if err != nil {
			t.Fatalf("loading model info: %v", err)
		}
		modelInfo = sema.FromModelInfo(mi)
	})
	for _, expr := range []string{
		"Patient.birthDate < " + dtLit,
		dtLit + " > Patient.birthDate",
		"if true then Patient.birthDate else " + dtLit,
	} {
		t.Run(expr, func(t *testing.T) {
			lib, err := compiler.Compile("library T version '1.0'\nusing FHIR version '4.0.1'\n" +
				"include FHIRHelpers version '4.0.1' called FHIRHelpers\ncontext Patient\ndefine R: " + expr + "\n")
			if err != nil {
				t.Fatal(err)
			}
			result := sema.Check(lib, modelInfo)
			found := false
			for node, conv := range result.Conversions {
				from := result.Types[node]
				if from != nil && from.String() == "FHIR.date" {
					found = true
					if !strings.HasSuffix(conv.Function, "ToDate") || conv.Target == nil || conv.Target.String() != "System.DateTime" {
						t.Errorf("birthDate converts by %q to %v, want FHIRHelpers.ToDate to System.DateTime", conv.Function, conv.Target)
					}
				}
			}
			if !found {
				t.Error("no conversion recorded for the FHIR.date")
			}
		})
	}
}
