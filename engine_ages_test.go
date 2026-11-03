package cql

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gofhir/cql/elm"
	"github.com/gofhir/cql/eval"
	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
)

// The tests of plan 017 (docs/plans/017-age-duration.md): an age is a duration.
// CQL 1.5.3, Appendix B, CalculateAgeAt: "The CalculateAgeAt operators are
// defined in terms of a datetime duration calculation", so the age in a unit is
// `<unit> between birthDate and asOf`, whole calendar periods (Appendix B,
// Duration), an uncertainty where a date is too imprecise for the unit (plan
// 016), and null where either is null.

// TestAgesByHand holds each age to the value the specification or the plan's
// table gives, through the engine's own ELM (evalImported) and the compiled
// library (evalConversion), both at 2026-01-01T00:00:00 at -12:00.
func TestAgesByHand(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		// Appendix B, CalculateAgeAt, the normative example.
		{"CalculateAgeInYearsAt(@2000-01-01, @2015-01-01)", "15"},
		{"CalculateAgeInYearsAt(@2000-01-01, null)", "null"},
		{"CalculateAgeInYearsAt(null, @2015-01-01)", "null"},
		{"CalculateAgeInDaysAt(@2000-01-01T00:00:00.000, null)", "null"},
		// The plan's table: D3, D4, D18, D6, D8, D15, D16 and the Date in hours.
		{"CalculateAgeInWeeksAt(@2019-01, @2019-03-01)", "Interval[4, 8]"},
		{"CalculateAgeInDaysAt(@2019-01, @2019-03-01)", "Interval[29, 59]"},
		{"CalculateAgeInMonthsAt(@1965, @2019-01-01)", "Interval[636, 648]"},
		{"CalculateAgeInDaysAt(@2019-01-01T10:00, @2019-01-02T09:00)", "0"},
		{"CalculateAgeInHoursAt(@2019-01-01T10:00, @2019-01-02T09:00)", "23"},
		// D15: the plan's table says 1380, as Java does, but both dates are no
		// finer than a minute, so the rule of plan 016 (the one that makes
		// `years between @2012 and @2014` an uncertainty) gives [1379, 1380], as
		// cql-execution does. Hours, with the dates a unit finer, are 23.
		{"CalculateAgeInMinutesAt(@2019-01-01T10:00, @2019-01-02T09:00)", "Interval[1379, 1380]"},
		{"CalculateAgeInMinutesAt(@2019-01-01T10:00:00, @2019-01-02T09:00:00)", "1380"},
		{"CalculateAgeInSecondsAt(@2019-01-01T10:00, @2019-01-01T10:01)", "Interval[0, 119]"},
		{"CalculateAgeInHoursAt(@2019-01-01, @2019-01-02)", "Interval[0, 47]"},
		{"CalculateAgeInMinutesAt(@2019-01-01, @2019-01-02)", "Interval[0, 2879]"},
		{"CalculateAgeInSecondsAt(@2019-01-01, @2019-01-02)", "Interval[0, 172799]"},
		// A negative age is the whole periods from the later to the earlier, negated.
		{"CalculateAgeInYearsAt(@2030-01-15, @2019-06-01)", "-10"},
		{"CalculateAgeInYearsAt(@2014-02-28, @2012-02-29)", "-2"},
		{"CalculateAgeInDaysAt(@2019-01-10, @2019-01-01)", "-9"},
		{"CalculateAgeInHoursAt(@2019-01-02T09:00, @2019-01-01T10:00)", "-23"},
		// The leap day: a year older on 28 February of a common year (plan 015).
		{"CalculateAgeInYearsAt(@2000-02-29, @2021-02-28)", "21"},
		{"CalculateAgeInYearsAt(@2000-02-29, @2021-02-27)", "20"},
		{"CalculateAgeInMonthsAt(@2000-01-31, @2000-02-29)", "1"},
		// A declared offset: the instant, not the field, decides hours, and years
		// count the fields (Appendix B, Duration).
		{"CalculateAgeInYearsAt(@2000-01-01, @2026-01-01T00:00:00.000Z)", "26"},
		{"CalculateAgeInHoursAt(@2019-01-01T10:00:00.000Z, @2019-01-01T10:00:00.000+05:00)", "-5"},
		// A DateTime to the day is read at the evaluation's offset, -12:00 here, so
		// its last moment is 11:59:59.999Z the next day, after 09:00Z (at UTC it
		// would be [0, 1]).
		{"CalculateAgeInDaysAt(@2019-01-01T, @2019-01-02T09:00:00.000Z)", "0"},
		// No asOf: a Date against Today() where the Date overload exists, a
		// DateTime against Now(). Today() is 2026-01-01 here.
		{"CalculateAgeInYears(@1965-06-01)", "60"},
		{"CalculateAgeInMonths(@1965-06-01)", "727"},
		{"CalculateAgeInDays(@2025-12-31)", "1"},
		{"CalculateAgeInWeeks(@2025-12-18)", "2"},
		{"CalculateAgeInWeeksAt(@2012-02-29, @2012-03-07)", "1"},
		{"CalculateAgeInWeeksAt(@2012-02-29, @2012-03-06)", "0"},
		{"CalculateAgeInDays(@2025-12-01T00:00:00.000)", "31"},
		{"CalculateAgeInHours(@2025-12-31T00:00:00.000)", "24"},
		{"CalculateAgeInMinutes(@2025-12-31T23:00:00.000)", "60"},
		{"CalculateAgeInSeconds(@2025-12-31T23:59:00.000)", "60"},
		{"CalculateAgeInHours(@2025-12-31)", "Interval[0, 24]"},
		{"CalculateAgeInYears(null)", "null"},
	} {
		for name, eval := range map[string]func(*testing.T, string) string{"compiled": evalConversion, "imported": evalImported} {
			if got := eval(t, tt.expr); got != tt.want {
				t.Errorf("%s, %s = %s, want %s", name, tt.expr, got, tt.want)
			}
		}
	}
}

// ageUnitNames are the seven units of an age, as the operator and the duration
// spell them.
var ageUnitNames = []struct{ operator, between string }{
	{"Years", "years"}, {"Months", "months"}, {"Weeks", "weeks"}, {"Days", "days"},
	{"Hours", "hours"}, {"Minutes", "minutes"}, {"Seconds", "seconds"},
}

// An independent oracle for an age, which does not call the engine: it
// enumerates the extremes of each date's range, takes whole calendar periods
// between them, and applies the rule of plan 016, which funcs.DurationBetweenAt
// documents and engine_uncertainty_test.go models for `between`. The periods
// are counted by their definition, `a + k periods <= b`, and not by the
// engine's fields.

// ageForms are the written forms of a date: a Date to the year, month and day,
// and a DateTime to the year, month, day, minute and millisecond.
var ageForms = []string{"dateY", "dateM", "dateD", "dtY", "dtM", "dtD", "dtMin", "dtMs"}

// ageDate is one operand of an age: how it is written, and the range of moments
// it could be, in UTC wall clock (every operand of the grid is in the one
// frame, so no offset is involved).
type ageDate struct {
	text       string
	date       bool
	precision  int // 0 year, 1 month, 2 day, 3 hour, 4 minute, 6 second or millisecond
	y, m, d    int
	h, mi, s   int
	ms         int
	start, end time.Time
}

func newAgeDate(form string, y, m, d, h, mi, s, ms int) ageDate {
	a := ageDate{y: y, m: m, d: d, h: h, mi: mi, s: s, ms: ms}
	switch form {
	case "dateY":
		a.text, a.date, a.precision = fmt.Sprintf("@%04d", y), true, 0
	case "dateM":
		a.text, a.date, a.precision = fmt.Sprintf("@%04d-%02d", y, m), true, 1
	case "dateD":
		a.text, a.date, a.precision = fmt.Sprintf("@%04d-%02d-%02d", y, m, d), true, 2
	case "dtY":
		a.text, a.precision = fmt.Sprintf("@%04dT", y), 0
	case "dtM":
		a.text, a.precision = fmt.Sprintf("@%04d-%02dT", y, m), 1
	case "dtD":
		a.text, a.precision = fmt.Sprintf("@%04d-%02d-%02dT", y, m, d), 2
	case "dtMin":
		a.text, a.precision = fmt.Sprintf("@%04d-%02d-%02dT%02d:%02d", y, m, d, h, mi), 4
	default:
		a.text, a.precision = fmt.Sprintf("@%04d-%02d-%02dT%02d:%02d:%02d.%03d", y, m, d, h, mi, s, ms), 6
	}
	a.span()
	return a
}

// span sets the earliest and the latest moment the date could be. A Date to the
// day is the day itself; a Date to the month or year ends on its last day; a
// DateTime ends at the last millisecond of what it states.
func (a *ageDate) span() {
	h, mi, s, ms := a.h, a.mi, a.s, a.ms
	m, d := a.m, a.d
	switch a.precision {
	case 0:
		m, d, h, mi, s, ms = 1, 1, 0, 0, 0, 0
	case 1:
		d, h, mi, s, ms = 1, 0, 0, 0, 0
	case 2:
		h, mi, s, ms = 0, 0, 0, 0
	case 4:
		s, ms = 0, 0
	}
	a.start = time.Date(a.y, time.Month(m), d, h, mi, s, ms*1e6, time.UTC)
	var next time.Time // the first moment after what it states
	switch a.precision {
	case 0:
		next = a.start.AddDate(1, 0, 0)
	case 1:
		next = a.start.AddDate(0, 1, 0)
	case 2:
		next = a.start.AddDate(0, 0, 1)
	case 4:
		next = a.start.Add(time.Minute)
	default:
		next = a.start.Add(time.Millisecond)
	}
	a.end = next.Add(-time.Millisecond)
	if a.date {
		// A Date is a day: its last moment is the start of its last day.
		a.end = time.Date(a.end.Year(), a.end.Month(), a.end.Day(), 0, 0, 0, 0, time.UTC)
	}
}

// asDateTime is the date as ToDateTime makes it: the same components with the
// time unspecified, so every moment of its last day.
func (a ageDate) asDateTime() ageDate {
	a.date = false
	a.span()
	return a
}

// addMonths is t plus k months, the day kept or moved back to the last of the
// month: 29 February plus a year is 28 February.
func addMonths(t time.Time, k int) time.Time {
	total := t.Year()*12 + int(t.Month()) - 1 + k
	y, m := total/12, total%12+1
	if m < 1 {
		y, m = y-1, m+12
	}
	last := time.Date(y, time.Month(m)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return time.Date(y, time.Month(m), min(t.Day(), last), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
}

// wholePeriods is the number of whole periods of the unit from a to b, negated
// when b is the earlier: the periods from b to a, as plan 015 reads a negative
// duration.
func wholePeriods(unit string, a, b time.Time) int64 {
	if b.Before(a) {
		return -wholePeriods(unit, b, a)
	}
	switch unit {
	case "years", "months":
		step := 1
		if unit == "years" {
			step = 12
		}
		k := (b.Year()-a.Year())*12 + int(b.Month()) - int(a.Month())
		k -= k % step
		for addMonths(a, k).After(b) {
			k -= step
		}
		return int64(k / step)
	}
	per := map[string]time.Duration{
		"weeks": 7 * 24 * time.Hour, "days": 24 * time.Hour, "hours": time.Hour,
		"minutes": time.Minute, "seconds": time.Second,
	}[unit]
	return int64(b.Sub(a) / per)
}

// ageOracle is the age in the unit, as the specification and the rule of plan
// 016 give it: the answer text, an integer or the uncertainty of two.
func ageOracle(unit string, birth, asOf ageDate) string {
	needsDateTime := unit == "hours" || unit == "minutes" || unit == "seconds"
	if birth.date != asOf.date || birth.date && needsDateTime {
		// "The Date values will be implicitly converted to DateTime."
		if birth.date {
			birth = birth.asDateTime()
		}
		if asOf.date {
			asOf = asOf.asDateTime()
		}
	}
	unitIndex := map[string]int{"years": 0, "months": 1, "weeks": 2, "days": 2, "hours": 3, "minutes": 4, "seconds": 5}[unit]
	// Not every moment is known (a precision short of the second), and one date
	// is no finer than the unit: an uncertainty. Two dates both finer than the
	// unit are read at the precision they share.
	if (birth.precision < 6 || asOf.precision < 6) && (birth.precision <= unitIndex || asOf.precision <= unitIndex) {
		low := wholePeriods(unit, birth.end, asOf.start)
		high := wholePeriods(unit, birth.start, asOf.end)
		if low == high {
			return fmt.Sprint(low)
		}
		return fmt.Sprintf("Interval[%d, %d]", low, high)
	}
	// Both dates finer than the unit: no uncertainty (plan 016, decision of
	// 31-10-2026, which reads it as the Java engine does). Years and months count
	// the calendar fields the two dates share, so the periods are counted between
	// them at the coarser precision of the two (a Date's 28 February is a year
	// after a DateTime's 29 February at 10:30). Weeks and the finer units are an
	// elapsed time, counted between the moments as written. Whether a unit finer
	// than a month should truncate to the shared precision too (a millisecond
	// birth against a reference to the minute) is the rule of
	// funcs.DurationBetweenAt, which this plan does not change.
	if unit != "years" && unit != "months" {
		return fmt.Sprint(wholePeriods(unit, birth.start, asOf.start))
	}
	shared := min(birth.precision, asOf.precision)
	return fmt.Sprint(wholePeriods(unit, truncate(birth.start, shared), truncate(asOf.start, shared)))
}

// truncate drops what a date of that precision does not state.
func truncate(t time.Time, precision int) time.Time {
	switch precision {
	case 0:
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	case 1:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	case 2:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	case 4:
		return t.Truncate(time.Minute)
	}
	return t
}

// ageGrid is the dates of the grid: births and references on awkward days (a
// leap day, the end of a month and of a year, a day short of an anniversary).
func ageGrid() (births, asOfs []ageDate) {
	for _, b := range [][7]int{
		{2000, 2, 29, 10, 30, 15, 250}, {1999, 12, 31, 23, 59, 59, 999},
		{1980, 1, 1, 0, 0, 0, 0}, {1965, 6, 15, 12, 0, 30, 500},
	} {
		for _, form := range ageForms {
			births = append(births, newAgeDate(form, b[0], b[1], b[2], b[3], b[4], b[5], b[6]))
		}
	}
	for _, a := range [][7]int{
		{2026, 1, 1, 0, 0, 0, 0}, {2019, 6, 14, 23, 59, 59, 999}, {2000, 2, 28, 10, 29, 0, 0},
		{1990, 3, 31, 9, 0, 0, 0}, {2021, 2, 28, 11, 0, 0, 0},
	} {
		for _, form := range []string{"dateY", "dateM", "dateD", "dtD", "dtMin", "dtMs"} {
			asOfs = append(asOfs, newAgeDate(form, a[0], a[1], a[2], a[3], a[4], a[5], a[6]))
		}
	}
	return births, asOfs
}

// TestAnAgeIsAnsweredByItsOracle walks births against references at every
// precision, Date and DateTime, in the seven units, and holds each age to the
// oracle above; and, wherever the two dates are both Dates or both DateTimes,
// to `<unit> between birth and asOf`, which the specification defines it as.
// A Date beside a DateTime is converted by the age with ToDateTime and left to
// plan 018 in `between`, so the second check leaves it out.
func TestAnAgeIsAnsweredByItsOracle(t *testing.T) {
	engine := NewEngine(WithEvaluationTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", -3*3600))))
	ask := func(expr string) string {
		v, err := engine.EvaluateExpression(context.Background(), "library T version '1.0'\ndefine R: "+expr+"\n", "R", nil, nil)
		if err != nil {
			return "error: " + err.Error()
		}
		return valueString(v)
	}
	births, asOfs := ageGrid()
	var checked, between int
	for _, u := range ageUnitNames {
		for _, b := range births {
			for _, a := range asOfs {
				expr := fmt.Sprintf("CalculateAgeIn%sAt(%s, %s)", u.operator, b.text, a.text)
				got := ask(expr)
				checked++
				if want := ageOracle(u.between, b, a); got != want {
					t.Errorf("%s = %s, want %s", expr, got, want)
				}
				needsDateTime := u.between == "hours" || u.between == "minutes" || u.between == "seconds"
				if b.date != a.date || b.date && needsDateTime {
					continue
				}
				between++
				if dur := ask(fmt.Sprintf("%s between %s and %s", u.between, b.text, a.text)); dur != got {
					t.Errorf("%s = %s but %s between = %s", expr, got, u.between, dur)
				}
			}
		}
	}
	if want := 7 * 32 * 30; checked != want || between == 0 {
		t.Fatalf("the grid covered %d ages and %d durations, want %d ages", checked, between, want)
	}
}

// ageOfAPatient evaluates the expressions at 2026-01-01T00:00:00 at -03:00, for
// a synthetic Patient with the birthDate, never real data.
func ageOfAPatient(t *testing.T, birthDate, expr string) string {
	t.Helper()
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\ndefine R: " + expr + "\n"
	got, err := NewEngine(WithEvaluationTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", -3*3600)))).
		EvaluateExpression(context.Background(), src, "R",
			[]byte(`{"resourceType":"Patient","id":"p1","birthDate":"`+birthDate+`"}`), nil)
	if err != nil {
		return "error: " + err.Error()
	}
	return valueString(got)
}

// TestTheAgeOfAPatient covers AgeIn…() and AgeIn…At() in the seven units for a
// birthDate to the day, the month and the year. The overloads are the Java
// translator's (plan 017): years and months read the birthDate as a Date against
// Today(); every finer unit reads ToDateTime(birthDate) against Now(), a day with
// no time of day, so its age is an uncertainty (CQL 1.5.3, Appendix B, Duration:
// `days between @2017-08-07T17:00 and @2017-08-14T` is [6, 7]). The user decided
// on 02-11-2026 to follow the specification where the Java engine gives one
// number. Born 1980-01-01, the 46 years to 2026-01-01 are 16802 days, 12 of
// them leap; the lower bound is counted from the end of 1980-01-01, a day short.
func TestTheAgeOfAPatient(t *testing.T) {
	for _, tt := range []struct{ birth, expr, want string }{
		{"1980-01-01", "AgeInYears()", "46"},
		{"1980-01-01", "AgeInMonths()", "552"},
		{"1980-01-01", "AgeInWeeks()", "2400"},
		{"1980-01-01", "AgeInDays()", "Interval[16801, 16802]"},
		{"1980-01-01", "AgeInHours()", "Interval[403224, 403248]"},
		{"1980-01-01", "AgeInMinutes()", "Interval[24193440, 24194880]"},
		{"1980-01-01", "AgeInSeconds()", "Interval[1451606400, 1451692800]"},
		{"1980-01-01", "AgeInYearsAt(@2019-01-01)", "39"},
		{"1980-01-01", "AgeInDaysAt(@2020-01-01T10:00:00.000Z)", "Interval[14609, 14610]"},
		{"1980-01-01", "AgeInYearsAt(@2020-01-01T10:00:00.000Z)", "40"},
		{"1980-01-01", "AgeInYearsAt(null)", "null"},
		// A birthDate to the month is every day of it.
		{"1980-01", "AgeInYears()", "46"},
		{"1980-01", "AgeInMonths()", "Interval[551, 552]"},
		{"1980-01", "AgeInWeeks()", "Interval[2395, 2400]"},
		{"1980-01", "AgeInDays()", "Interval[16771, 16802]"},
		// A birthDate to the year is every day of it.
		{"1980", "AgeInYears()", "Interval[45, 46]"},
		{"1980", "AgeInMonths()", "Interval[540, 552]"},
	} {
		if got := ageOfAPatient(t, tt.birth, tt.expr); got != tt.want {
			t.Errorf("born %s, %s = %s, want %s", tt.birth, tt.expr, got, tt.want)
		}
	}
	// The same ages through the oracle, which does not call the engine, for the
	// seven units of a birthDate to the day: ToDateTime(birthDate) against Now()
	// for the finer units, and the Date against Today() for years and months.
	now := newAgeDate("dtMs", 2026, 1, 1, 0, 0, 0, 0)
	today := newAgeDate("dateD", 2026, 1, 1, 0, 0, 0, 0)
	for _, form := range []string{"dateY", "dateM", "dateD"} {
		birth := newAgeDate(form, 1980, 1, 1, 0, 0, 0, 0)
		text := strings.TrimPrefix(birth.text, "@")
		for _, u := range ageUnitNames {
			want := ageOracle(u.between, birth, today)
			if u.between != "years" && u.between != "months" {
				want = ageOracle(u.between, birth.asDateTime(), now)
			}
			if got := ageOfAPatient(t, text, "AgeIn"+u.operator+"()"); got != want {
				t.Errorf("born %s, AgeIn%s() = %s, want %s", text, u.operator, got, want)
			}
		}
	}
}

// TestAgesFromTheReferenceELM evaluates the ELM the Java translator 5.4.0 writes
// for testdata/ages/cases.cql: CalculateAge and CalculateAgeAt, with the unit in
// the precision, and the ToDateTime around a birthDate that the finer units
// take. The import reads them as the AST's own call, so the answers are those of
// TestTheAgeOfAPatient. The ELM reader does not take the Date and DateTime
// constructors the translator writes for a literal (backlog 1c9), so a case with
// one reads as null: each is named, so that its fix shows here.
//
// Regenerate the ELM with sh scripts/references.sh write, or, from
// internal/referenceharness with the library beside FHIRHelpers-4.0.1.cql:
//
//	java -cp "out:lib/*" Translate /o/cases.cql /o > /o/cases.elm.json
func TestAgesFromTheReferenceELM(t *testing.T) {
	raw, err := os.ReadFile("testdata/ages/cases.elm.json")
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Library map[string]json.RawMessage `json:"library"`
	}
	if err = json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	// The Patient definition is a retrieve, which has no provider here: `Patient`
	// is the context resource, as in the compiled library.
	var statements struct {
		Def []map[string]json.RawMessage `json:"def"`
	}
	if err = json.Unmarshal(envelope.Library["statements"], &statements); err != nil {
		t.Fatal(err)
	}
	var kept []map[string]json.RawMessage
	for _, d := range statements.Def {
		if string(d["name"]) != `"Patient"` {
			kept = append(kept, d)
		}
	}
	envelope.Library["statements"], _ = json.Marshal(map[string]any{"def": kept})
	one, err := json.Marshal(envelope.Library)
	if err != nil {
		t.Fatal(err)
	}
	read, err := elm.UnmarshalLibrary(one)
	if err != nil {
		t.Fatal(err)
	}
	imported, err := elm.Import(read)
	if err != nil {
		t.Fatal(err)
	}
	evaluate := func(birthDate, name string) string {
		ctx := eval.NewContext(context.Background(), imported)
		ctx.EvaluationTimestamp = time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", -3*3600))
		ctx.ContextValue = []byte(`{"resourceType":"Patient","id":"p1","birthDate":"` + birthDate + `"}`)
		low, _ := fptypes.NewDateTime("2019-01-01T00:00:00.000")
		high, _ := fptypes.NewDateTime("2019-12-31T23:59:59.999")
		ctx.Parameters["Measurement Period"] = cqltypes.NewInterval(low, high, true, true)
		got, err := eval.NewEvaluator(ctx).EvaluateExpression(name)
		if err != nil {
			return "error: " + err.Error()
		}
		return valueString(got)
	}
	// Born 1980-01-01; the period starts at 2019-01-01T00:00:00.000, 39 years and
	// 14245 days (39*365 + 10 leap days) later, 14244 from the end of the day.
	for name, want := range map[string]string{
		"AgeInYears": "46", "AgeInMonths": "552", "AgeInWeeks": "2400",
		"AgeInDays": "Interval[16801, 16802]", "AgeInHours": "Interval[403224, 403248]",
		"AgeInMinutes": "Interval[24193440, 24194880]", "AgeInSeconds": "Interval[1451606400, 1451692800]",
		"AgeInYearsAtStart": "39", "AgeInMonthsAtStart": "468", "AgeInWeeksAtStart": "Interval[2034, 2035]",
		"AgeInDaysAtStart": "Interval[14244, 14245]", "AgeInHoursAtStart": "Interval[341856, 341880]",
		"AgeInMinutesAtStart": "Interval[20511360, 20512800]", "AgeInSecondsAtStart": "Interval[1230681600, 1230768000]",
	} {
		if got := evaluate("1980-01-01", name); got != want {
			t.Errorf("%s from ELM = %s, want %s", name, got, want)
		}
	}
	// A birthDate to the month, through the same ELM.
	for name, want := range map[string]string{
		"AgeInMonths": "Interval[551, 552]", "AgeInDays": "Interval[16771, 16802]",
		"AgeInMonthsAtStart": "Interval[467, 468]",
	} {
		if got := evaluate("1980-01", name); got != want {
			t.Errorf("%s from ELM, born 1980-01 = %s, want %s", name, got, want)
		}
	}
	unread := []string{"AgeInYearsAtDate", "AgeInMonthsAtDate", "CD3", "CD4", "CD18", "CD6", "CD8", "CD15", "CD16",
		"CH5", "CNegative", "CZulu", "CNormative", "CLeapDay", "CNullAsOf", "CYearsToday", "CMonthsToday",
		"CDaysToday", "CDaysNow", "CHoursNow", "CMinutesNow", "CSecondsNow", "CWeeksAtDate"}
	if len(statements.Def) != 1+14+len(unread) {
		t.Fatalf("the ELM has %d definitions; testdata/ages/cases.cql changed", len(statements.Def))
	}
	for _, name := range unread {
		if got := evaluate("1980-01-01", name); got != "null" {
			t.Errorf("%s from ELM = %s: the reader takes the constructor now (backlog 1c9), so check it against TestAgesByHand", name, got)
		}
	}
}

// TestTheAgesAreaIsItsOracle holds every CalculateAgeIn<Unit>At of the snapshot
// area `ages` whose two dates are literals without an offset to the independent
// oracle above, so that a divergence from a reference there is classified
// against a value the engine did not compute (plan 017).
func TestTheAgesAreaIsItsOracle(t *testing.T) {
	define := regexp.MustCompile(`(?m)^define (D\d+): CalculateAgeIn(\w+?)sAt\((@[\d:T.-]+), (@[\d:T.-]+)\)$`)
	cql, err := os.ReadFile("testdata/snapshots/ages.cql")
	if err != nil {
		t.Fatal(err)
	}
	answers := map[string]string{}
	raw, err := os.ReadFile("testdata/snapshots/ages.answers.tsv")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if name, answer, ok := strings.Cut(line, "\t"); ok && !strings.HasPrefix(name, "#") {
			answers[name] = answer
		}
	}
	checked := 0
	for _, m := range define.FindAllStringSubmatch(string(cql), -1) {
		birth, okB := parseAgeDate(m[3])
		asOf, okA := parseAgeDate(m[4])
		if !okB || !okA {
			continue
		}
		want := ageOracle(strings.ToLower(m[2])+"s", birth, asOf)
		got := strings.NewReplacer("Uncertainty[Integer ", "Interval[", ", Integer ", ", ", "Integer ", "").Replace(answers[m[1]])
		if got != want {
			t.Errorf("%s: %s = %s, the oracle gives %s", m[1], m[0], got, want)
		}
		checked++
	}
	if checked < 300 {
		t.Fatalf("only %d ages of the area were checked against the oracle", checked)
	}
}

// parseAgeDate reads a date literal of one of the forms of ageForms.
func parseAgeDate(text string) (ageDate, bool) {
	var y, m, d, h, mi, s, ms int
	switch {
	case regexp.MustCompile(`^@\d{4}$`).MatchString(text):
		fmt.Sscanf(text, "@%4d", &y)
		return newAgeDate("dateY", y, 1, 1, 0, 0, 0, 0), true
	case regexp.MustCompile(`^@\d{4}-\d\d$`).MatchString(text):
		fmt.Sscanf(text, "@%4d-%2d", &y, &m)
		return newAgeDate("dateM", y, m, 1, 0, 0, 0, 0), true
	case regexp.MustCompile(`^@\d{4}-\d\d-\d\d$`).MatchString(text):
		fmt.Sscanf(text, "@%4d-%2d-%2d", &y, &m, &d)
		return newAgeDate("dateD", y, m, d, 0, 0, 0, 0), true
	case regexp.MustCompile(`^@\d{4}T$`).MatchString(text):
		fmt.Sscanf(text, "@%4dT", &y)
		return newAgeDate("dtY", y, 1, 1, 0, 0, 0, 0), true
	case regexp.MustCompile(`^@\d{4}-\d\dT$`).MatchString(text):
		fmt.Sscanf(text, "@%4d-%2dT", &y, &m)
		return newAgeDate("dtM", y, m, 1, 0, 0, 0, 0), true
	case regexp.MustCompile(`^@\d{4}-\d\d-\d\dT$`).MatchString(text):
		fmt.Sscanf(text, "@%4d-%2d-%2dT", &y, &m, &d)
		return newAgeDate("dtD", y, m, d, 0, 0, 0, 0), true
	case regexp.MustCompile(`^@\d{4}-\d\d-\d\dT\d\d:\d\d$`).MatchString(text):
		fmt.Sscanf(text, "@%4d-%2d-%2dT%2d:%2d", &y, &m, &d, &h, &mi)
		return newAgeDate("dtMin", y, m, d, h, mi, 0, 0), true
	case regexp.MustCompile(`^@\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d{3}$`).MatchString(text):
		fmt.Sscanf(text, "@%4d-%2d-%2dT%2d:%2d:%2d.%3d", &y, &m, &d, &h, &mi, &s, &ms)
		return newAgeDate("dtMs", y, m, d, h, mi, s, ms), true
	}
	return ageDate{}, false
}

// TestTodayIsTheEvaluationsDate holds the no-asOf ages of a Date to Today() at
// the evaluation's own offset (CQL 1.5.3, Appendix B, Today: "the date of the
// evaluation request timestamp"): at 2026-01-01T05:00 at +14:00 it is already
// 2026, where in UTC it is still 31 December (review of plan 017, round 1).
func TestTodayIsTheEvaluationsDate(t *testing.T) {
	engine := NewEngine(WithEvaluationTimestamp(time.Date(2026, 1, 1, 5, 0, 0, 0, time.FixedZone("", 14*3600))))
	evaluateCases(t, engine, []uncertainCase{
		{"CalculateAgeInDays(@2025-12-31)", "1"},
		{"CalculateAgeInYears(@2025-01-01)", "1"},
	})
}
