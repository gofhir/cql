package cql

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/compiler"
	"github.com/gofhir/cql/elm"
	"github.com/gofhir/cql/eval"
)

// The durations area of testdata/snapshots holds these cases against both
// reference engines. These are the tests that area cannot hold: the grid read
// from the other side (the metamorphic relations), the other entry points, the
// ages, and the edges of the calendar. Plan 015.
//
// CQL 1.5.3, Appendix B, Duration: "returns the number of whole calendar periods
// for the specified precision between the first and second arguments. If the
// first argument is after the second argument, the result is negative. […] any
// fractional periods are dropped". Author's Guide, Date and Time Arithmetic:
// "For variable length precisions (months and years), the operation uses the
// calendar length of the precision to determine the number of periods". A whole
// period is the largest n for which low + n periods is not after high (decision
// A); the hour counts (decision B).

// TestYearsAndMonthsBetweenCountCalendarPeriods holds the answers the two
// reference engines give, or, where they differ, the one decisions A and B pick.
func TestYearsAndMonthsBetweenCountCalendarPeriods(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		// The leap day: a person born on 29 February is a year old on the 28th.
		{"years between @2012-02-29 and @2014-02-28", "2"},
		{"years between @2012-02-29 and @2013-02-28", "1"},
		{"years between @2012-02-29 and @2013-02-27", "0"},
		{"years between @2012-02-29 and @2016-02-28", "3"},
		{"years between @2012-02-29 and @2016-02-29", "4"},
		{"years between @2011-03-01 and @2012-02-29", "0"},
		{"years between @2011-02-28 and @2012-02-29", "1"},
		{"years between @2012-03-01 and @2013-02-28", "0"},
		{"years between @2012-01-15 and @2014-01-14", "1"},
		{"years between @2012-01-15 and @2014-01-15", "2"},
		// The other direction is the first one negated, not one period more.
		{"years between @2014-02-28 and @2012-02-29", "-2"},
		{"years between @2013-02-28 and @2012-02-29", "-1"},
		{"years between @2014-01-15 and @2012-01-16", "-1"},
		{"years between @2012-01-16 and @2014-01-15", "1"},
		{"months between @2014-03-31 and @2014-02-28", "-1"},
		{"months between @2014-02-28 and @2014-01-28", "-1"},
		{"months between @2014-02-28 and @2014-01-31", "-1"},
		{"months between @2014-01-28 and @2014-01-28", "0"},
		// The time of the day, to the millisecond (decision B).
		{"years between @2012-02-29T12:34:56 and @2014-02-28T12:34:56", "2"},
		{"years between @2012-02-29T12:34:56 and @2014-02-28T12:34:55", "1"},
		{"years between @2012-02-29T12:34:56 and @2014-02-28T12:34:57", "2"},
		{"years between @2014-02-28T12:34:56 and @2012-02-29T12:34:56", "-2"},
		{"years between @2012-02-29T23:00:00 and @2014-02-28T22:00:00", "1"},
		{"years between @2014-01-15T10:00:00 and @2015-01-15T09:59:59", "0"},
		{"years between @2014-01-15T10:00:00 and @2015-01-15T10:00:00", "1"},
		{"months between @2014-01-15T10:00:00 and @2014-02-15T09:59:59", "0"},
		{"months between @2014-02-15T09:59:59 and @2014-01-15T10:00:00", "0"},
		{"months between @2014-02-15T10:00:00 and @2014-01-15T10:00:00", "-1"},
		{"years between @2012-02-29T12:34:56.000 and @2014-02-28T12:34:56.000", "2"},
		{"years between @2012-02-29T12:34:56.001 and @2014-02-28T12:34:56.000", "1"},
		{"years between @2012-02-29T12:34:56.000 and @2014-02-28T12:34:56.001", "2"},
		{"years between @2014-02-28T12:34:56.000 and @2012-02-29T12:34:56.001", "-1"},
		{"months between @2014-01-31T10:00:00.000 and @2014-02-28T10:00:00.000", "1"},
		{"months between @2014-01-31T10:00:00.000 and @2014-02-28T09:59:59.999", "0"},
		{"months between @2014-01-31T10:00:00.001 and @2014-02-28T10:00:00.000", "0"},
		// The end of the month: the sum of the engine's `+` is the measure
		// (decision A).
		{"months between @2014-01-31 and @2014-02-28", "1"},
		{"months between @2014-01-31 and @2014-02-27", "0"},
		{"months between @2012-01-31 and @2012-02-29", "1"},
		{"months between @2012-01-31 and @2012-02-28", "0"},
		{"months between @2014-03-31 and @2014-04-30", "1"},
		{"months between @2014-01-30 and @2014-02-28", "1"},
		{"months between @2014-01-31T10:00:00 and @2014-02-28T10:00:00", "1"},
		{"months between @2014-01-31T10:00:00 and @2014-02-28T09:59:59", "0"},
		{"months between @2014-01-01 and @2014-03-15", "2"},
		{"months between @2014-01-31 and @2014-02-01", "0"},
		{"@2012-02-29 + 2 years", "2014-02-28"},
		{"@2014-01-31 + 1 month", "2014-02-28"},
		// Mixed precisions, compared at the one both share: a Date beside a
		// DateTime assumes no midnight, and a DateTime to the minute is not asked
		// for its seconds.
		{"years between @2012-02-29 and @2014-02-28T00:00:00.000", "2"},
		{"years between @2012-02-29 and @2014-02-28T23:59:59.999", "2"},
		{"years between @2012-02-29T23:59:59.999 and @2014-02-28", "2"},
		{"years between @2012-02-29T12:00 and @2014-02-28T12:00:00.000", "2"},
		{"years between @2012-02-29T12:00 and @2014-02-28T11:59:59.999", "1"},
		{"years between @2012-02-29T12:00:00 and @2014-02-28T12:00", "2"},
		{"years between @2012-02-29T12:00:00 and @2014-02-28T11:00", "1"},
		{"years between @2012-02-29T12 and @2014-02-28T11:59:59", "1"},
		{"years between @2012-02 and @2014-02-28", "2"},
		{"years between @2012-02 and @2014-01-31", "1"},
		{"months between @2014-01 and @2014-02-28", "1"},
		{"months between @2014-01-31 and @2014-02", "Interval[0, 1]"},
		// The edges of the calendar. 0001-01-01 is a date like any other.
		{"years between @0001-01-01 and @0002-01-01", "1"},
		{"years between @0001-01-01 and @0001-12-31", "0"},
		{"years between @0002-01-01 and @0001-01-01", "-1"},
		{"months between @0001-01-01 and @0001-02-01", "1"},
		{"months between @0001-01-01T00:00:00.000 and @0001-02-01T00:00:00.000", "1"},
		{"years between @0001-03-01 and @9999-02-28", "9997"},
		{"years between @0001-02-28 and @9999-12-31", "9998"},
		{"years between @9999-12-31 and @0001-02-28", "-9998"},
		{"months between @9999-01-31 and @9999-12-31", "11"},
		{"months between @9999-12-31 and @9999-01-31", "-11"},
		{"months between @9999-01-31 and @9999-02-28", "1"},
		{"years between @9999-01-01 and @9999-12-31", "0"},
		{"months between @0001-01-01 and @9999-12-31", "119987"},
		{"years between @9999-12-31T23:59:59.999 and @9999-12-31T23:59:59.999", "0"},
		{"years between @0001-01-01T00:00:00.000 and @9999-12-31T23:59:59.999", "9998"},
		// Nothing to count.
		{"years between (null as Date) and @2014-02-28", "null"},
		{"months between @2014-02-28 and (null as DateTime)", "null"},
	} {
		if got := evalConversion(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestDurationBetweenAtLowPrecisionIsAnUncertainty covers the bounds of the
// uncertainty. CQL 1.5.3, Language Semantics: "Results in an uncertainty from
// the shortest possible duration, to the longest possible duration"; the suite's
// own answer is `Interval[4, 5]` for `years between DateTime(2005) and
// DateTime(2010)`. Each bound is the exact count between the extremes, so the
// bounds move with it. The two forms of the reverse were [-3, -2] and [-1, 0]
// before plan 015.
func TestDurationBetweenAtLowPrecisionIsAnUncertainty(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"years between DateTime(2005) and DateTime(2010)", "Interval[4, 5]"},
		{"years between @2012 and @2014", "Interval[1, 2]"},
		{"years between @2014 and @2012", "Interval[-2, -1]"},
		{"months between @2014-02 and @2014-03", "Interval[0, 1]"},
		{"months between @2014-03 and @2014-02", "Interval[-1, 0]"},
		{"months between DateTime(2005) and DateTime(2006, 7)", "Interval[6, 18]"},
		{"months between DateTime(2006, 7) and DateTime(2005)", "Interval[-18, -6]"},
	} {
		if got := evalConversion(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestADurationIsTheSameFromEveryEntryPoint asks the same case through `years
// between`, through `duration in years of` an interval, and through the ELM
// of the first, imported again.
func TestADurationIsTheSameFromEveryEntryPoint(t *testing.T) {
	for _, tt := range []struct {
		low, high, unit, want string
	}{
		{"@2012-02-29", "@2014-02-28", "year", "2"},
		{"@2012-02-29", "@2013-02-28", "year", "1"},
		{"@2012-02-29T12:34:56", "@2014-02-28T12:34:55", "year", "1"},
		{"@2012-02-29T12:34:56", "@2014-02-28T12:34:56", "year", "2"},
		{"@2014-01-31", "@2014-02-28", "month", "1"},
		{"@2014-01-15T10:00:00", "@2014-02-15T09:59:59", "month", "0"},
		{"@2014-01-31T10:00:00.000", "@2014-02-28T10:00:00.000", "month", "1"},
		{"@0001-01-01", "@0002-01-01", "year", "1"},
		{"@9999-01-31", "@9999-02-28", "month", "1"},
	} {
		between := fmt.Sprintf("%ss between %s and %s", tt.unit, tt.low, tt.high)
		inInterval := fmt.Sprintf("duration in %ss of Interval[%s, %s]", tt.unit, tt.low, tt.high)
		if got := evalConversion(t, between); got != tt.want {
			t.Errorf("%s = %s, want %s", between, got, tt.want)
		}
		if got := evalConversion(t, inInterval); got != tt.want {
			t.Errorf("%s = %s, want %s", inInterval, got, tt.want)
		}
		if got := evalImported(t, between); got != tt.want {
			t.Errorf("%s, through ELM = %s, want %s", between, got, tt.want)
		}
	}
}

// evalImported compiles a library, writes it as ELM, reads it back and imports
// it, so that the operator evaluated is the one an ELM library has.
func evalImported(t *testing.T, expr string) string {
	t.Helper()
	parsed, err := compiler.Compile("library T version '1.0'\ndefine R: " + expr + "\n")
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
	got, err := eval.NewEvaluator(eval.NewContext(context.Background(), imported)).EvaluateExpression("R")
	if err != nil {
		return "error: " + err.Error()
	}
	return valueString(got)
}

// TestTheAgeIsADurationInTheUnit covers the CalculateAgeIn…At operators.
// CQL 1.5.3, Appendix B, CalculateAgeAt: "The CalculateAgeAt operators are
// defined in terms of a datetime duration calculation", for Date and DateTime
// alike, in years, months, weeks and days.
func TestTheAgeIsADurationInTheUnit(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"CalculateAgeInYearsAt(@2012-02-29, @2014-02-28)", "2"},
		{"CalculateAgeInYearsAt(@2012-02-29, @2013-02-28)", "1"},
		{"CalculateAgeInYearsAt(@2012-02-29, @2013-02-27)", "0"},
		{"CalculateAgeInYearsAt(@2000-06-01, @2019-06-01)", "19"},
		{"CalculateAgeInYearsAt(@2000-06-01, @2019-05-31)", "18"},
		{"CalculateAgeInYearsAt(@2014-02-28, @2012-02-29)", "-2"},
		{"CalculateAgeInYearsAt(@2012-02-29T12:00:00, @2014-02-28T11:00:00)", "1"},
		{"CalculateAgeInYearsAt(@2012-02-29T12:00:00, @2014-02-28T12:00:00)", "2"},
		{"CalculateAgeInYearsAt(@2012-02-29, @2014-02-28T00:00:00.000)", "2"},
		{"CalculateAgeInYearsAt(@2012-02-29T12:00:00.000, @2014-02-28)", "2"},
		{"CalculateAgeInMonthsAt(@2014-01-31, @2014-02-28)", "1"},
		{"CalculateAgeInMonthsAt(@2014-01-31, @2014-02-27)", "0"},
		{"CalculateAgeInMonthsAt(@2014-01-31T10:00:00, @2014-02-28T09:00:00)", "0"},
		{"CalculateAgeInMonthsAt(@2014-02-28, @2014-01-31)", "-1"},
		{"CalculateAgeInWeeksAt(@2012-02-29, @2012-03-07)", "1"},
		{"CalculateAgeInWeeksAt(@2012-02-29, @2012-03-06)", "0"},
		{"CalculateAgeInDaysAt(@2012-02-29, @2013-02-28)", "365"},
		{"CalculateAgeInDaysAt(@2012-02-29T00:00:00, @2013-02-28T00:00:00)", "365"},
		// The operators without At measure against the evaluation's own clock,
		// 2026-01-01 at -12:00.
		{"CalculateAgeInYears(@2012-02-29, @2014-02-28)", "2"},
		{"CalculateAgeInMonths(@2014-01-31, @2014-02-28)", "1"},
		{"CalculateAgeInYears(@2000-01-01)", "26"},
		{"CalculateAgeInYears(@2000-01-02)", "25"},
		{"CalculateAgeInYearsAt(@2000-01-01, null as Date)", "null"},
		{"CalculateAgeInYearsAt(null as Date, @2014-02-28)", "null"},
		{"CalculateAgeInMonthsAt(null as Date, @2014-02-28)", "null"},
	} {
		if got := evalConversion(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestAPatientBornOnTheLeapDayIsAYearOlderOnThe28th covers AgeInYears and its
// kin against an R4 Patient born on 29 February 2004, with an explicit asOf (the
// snapshots freeze the clock at the 1st of January), and the clinical threshold
// `>= 18` the day before the birthday, the day, and the day after.
func TestAPatientBornOnTheLeapDayIsAYearOlderOnThe28th(t *testing.T) {
	const header = "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\n"
	patient := []byte(`{"resourceType":"Patient","id":"p","birthDate":"2004-02-29"}`)
	at := func(day string) *Engine {
		d, err := time.Parse("2006-01-02", day)
		if err != nil {
			t.Fatal(err)
		}
		return NewEngine(WithDataProvider(snapshotData{}), WithEvaluationTimestamp(d.In(time.UTC)))
	}
	engine := at("2022-02-28")
	for _, tt := range []struct{ expr, want string }{
		// Born 2004-02-29: 18 on 2022-02-28, the day before, 17.
		{"AgeInYearsAt(@2022-02-27)", "17"},
		{"AgeInYearsAt(@2022-02-27) >= 18", "false"},
		{"AgeInYearsAt(@2022-02-28)", "18"},
		{"AgeInYearsAt(@2022-02-28) >= 18", "true"},
		{"AgeInYearsAt(@2022-03-01)", "18"},
		{"AgeInYearsAt(@2022-03-01) >= 18", "true"},
		{"AgeInYearsAt(@2023-02-27)", "18"},
		{"AgeInYearsAt(@2023-02-28)", "19"},
		{"AgeInYearsAt(@2024-02-28)", "19"},
		{"AgeInYearsAt(@2024-02-29)", "20"},
		{"AgeInYearsAt(@2022-02-28T00:00:00.000)", "18"},
		{"AgeInYearsAt(@2022-02-27T23:59:59.999)", "17"},
		{"AgeInMonthsAt(@2022-02-28)", "216"},
		{"AgeInMonthsAt(@2022-02-27)", "215"},
		{"AgeInMonthsAt(@2004-03-28)", "0"},
		{"AgeInMonthsAt(@2004-03-29)", "1"},
		{"AgeInDaysAt(@2004-03-01)", "1"},
		{"AgeInWeeksAt(@2004-03-07)", "1"},
		{"CalculateAgeInYearsAt(Patient.birthDate, @2022-02-28)", "18"},
		{"CalculateAgeInYearsAt(Patient.birthDate, @2022-02-27)", "17"},
		// Against the evaluation's own timestamp, 2022-02-28.
		{"AgeInYears()", "18"},
		{"AgeInMonths()", "216"},
	} {
		v, err := engine.EvaluateExpression(context.Background(), header+"define R: "+tt.expr+"\n", "R", patient, nil)
		if got := valueString(v); err != nil || got != tt.want {
			t.Errorf("%s = %s, %v; want %s", tt.expr, got, err, tt.want)
		}
	}
	v, err := at("2022-02-27").EvaluateExpression(context.Background(), header+"define R: AgeInYears()\n", "R", patient, nil)
	if got := valueString(v); err != nil || got != "17" {
		t.Errorf("AgeInYears() the day before the birthday = %s, %v; want 17", got, err)
	}
}

// dateGrid is every date of the grid of plan 015, as written in CQL: the days
// from the 28th to the 31st of January to April of 2011, 2012 and 2013 where
// the month has them, and the 27th too for the end of a span.
func dateGrid(firstDay int) []string {
	var out []string
	for year := 2011; year <= 2013; year++ {
		for month := 1; month <= 4; month++ {
			for day := firstDay; day <= 31; day++ {
				if d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC); int(d.Month()) == month {
					out = append(out, d.Format("2006-01-02"))
				}
			}
		}
	}
	return out
}

// TestTheCountsHoldTheirDefinitionOverTheGrid is the metamorphic test of plan 015.
// With n = years between low and high, low + n years <= high < low + (n + 1)
// years; the same for months; and the count from high to low is the count from
// low to high, negated. It is the rule the engine chose held against the
// engine's own `+`, which makes it a consistency check, not a proof the rule is
// the specification's: the answers of the references are in the durations area.
func TestTheCountsHoldTheirDefinitionOverTheGrid(t *testing.T) {
	type span struct{ low, high string }
	var spans []span
	lows, highs := dateGrid(28), dateGrid(27)
	for _, low := range lows {
		for _, high := range highs {
			spans = append(spans, span{"@" + low, "@" + high})
			// With a time: the same, an equal hour, a second before and after.
			for _, hour := range [][2]string{{"T12:34:56", "T12:34:56"}, {"T12:34:56", "T12:34:55"}, {"T12:34:56", "T12:34:57"}} {
				// The days that the end of a month or the leap day turn on; the Date
				// spans hold the rest, and the library stays small.
				if low[8:] < "29" || high[8:] < "28" || high[8:] > "30" {
					continue
				}
				spans = append(spans, span{"@" + low + hour[0], "@" + high + hour[1]})
			}
		}
	}
	// The engine caps the length of a library's source, so the spans go in
	// chunks, each with its own defines; names carry the span's index.
	evaluate := func(define func(b *strings.Builder, i int, s span)) map[string]fptypes.Value {
		all := map[string]fptypes.Value{}
		const chunk = 120
		for from := 0; from < len(spans); from += chunk {
			var src strings.Builder
			src.WriteString("library T version '1.0'\n")
			for i := from; i < min(from+chunk, len(spans)); i++ {
				define(&src, i, spans[i])
			}
			got, err := NewEngine().EvaluateLibrary(context.Background(), src.String(), nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			for k, v := range got {
				all[k] = v
			}
		}
		return all
	}
	got := evaluate(func(src *strings.Builder, i int, s span) {
		fmt.Fprintf(src, "define Y%d: years between %s and %s\n", i, s.low, s.high)
		fmt.Fprintf(src, "define M%d: months between %s and %s\n", i, s.low, s.high)
		fmt.Fprintf(src, "define YR%d: years between %s and %s\n", i, s.high, s.low)
		fmt.Fprintf(src, "define MR%d: months between %s and %s\n", i, s.high, s.low)
		fmt.Fprintf(src, "define AY%d: CalculateAgeInYearsAt(%s, %s)\n", i, s.low, s.high)
		fmt.Fprintf(src, "define AM%d: CalculateAgeInMonthsAt(%s, %s)\n", i, s.low, s.high)
	})
	forward := 0
	for i, s := range spans {
		y, m := valueString(got[fmt.Sprintf("Y%d", i)]), valueString(got[fmt.Sprintf("M%d", i)])
		yr, mr := valueString(got[fmt.Sprintf("YR%d", i)]), valueString(got[fmt.Sprintf("MR%d", i)])
		if y == "null" || m == "null" || yr == "null" || mr == "null" {
			t.Fatalf("span %s .. %s counts null: %s %s %s %s", s.low, s.high, y, m, yr, mr)
		}
		if atoi(y) != -atoi(yr) {
			t.Errorf("years between %s and %s = %s, but the reverse is %s", s.low, s.high, y, yr)
		}
		if atoi(m) != -atoi(mr) {
			t.Errorf("months between %s and %s = %s, but the reverse is %s", s.low, s.high, m, mr)
		}
		if y != valueString(got[fmt.Sprintf("AY%d", i)]) || m != valueString(got[fmt.Sprintf("AM%d", i)]) {
			t.Errorf("age from %s to %s = %s years, %s months; the durations are %s, %s", s.low, s.high,
				valueString(got[fmt.Sprintf("AY%d", i)]), valueString(got[fmt.Sprintf("AM%d", i)]), y, m)
		}
		if s.low <= s.high {
			forward++
		}
	}
	if forward < 1000 {
		t.Fatalf("only %d spans in order; the grid is too small to prove anything", forward)
	}
	// low + n <= high < low + (n + 1), with the engine's own `+` and `<=`, over
	// the spans in order (a reverse span is negative: its count is the other's).
	held := evaluate(func(src *strings.Builder, i int, s span) {
		if s.low > s.high {
			return
		}
		y, m := valueString(got[fmt.Sprintf("Y%d", i)]), valueString(got[fmt.Sprintf("M%d", i)])
		fmt.Fprintf(src, "define Y%d: %s + %s years <= %s and %s < %s + %d years\n", i, s.low, y, s.high, s.high, s.low, atoi(y)+1)
		fmt.Fprintf(src, "define M%d: %s + %s months <= %s and %s < %s + %d months\n", i, s.low, m, s.high, s.high, s.low, atoi(m)+1)
	})
	for name, v := range held {
		if valueString(v) != "true" {
			i := atoi(name[1:])
			t.Errorf("%s of %s .. %s = %s: low + n periods <= high < low + (n+1) periods does not hold; count %s",
				name[:1], spans[i].low, spans[i].high, valueString(v), valueString(got[name]))
		}
	}
}

func atoi(s string) int {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		panic(err)
	}
	return n
}
