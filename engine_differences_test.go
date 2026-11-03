package cql

import (
	"context"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofhir/cql/compiler"
	"github.com/gofhir/cql/elm"
	"github.com/gofhir/cql/eval"
)

// The tests of plan 020 (docs/plans/020-difference-uncertainty.md):
// `difference in X between` counts the boundaries of X crossed, Sundays for
// weeks, in the evaluation's offset below the day, and answers with an
// uncertainty where an operand is coarser than X.
// CQL 1.5.3, Appendix B, Difference: "the number of boundaries crossed for the
// specified precision", "any fractional boundaries are dropped", "Sunday is
// considered to be the first day of the week", and DateTime values "normalized
// to the timezone offset of the evaluation request timestamp, but only when the
// comparison precision is hours, minutes, seconds, or milliseconds".

// differenceAt evaluates the expression at 2026-01-01T00:00:00 at the offset
// (in minutes), compiled (imported false) or through the ELM the compiler
// writes and the importer reads back (imported true).
func differenceAt(t *testing.T, expr string, offset int, imported bool) string {
	t.Helper()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", offset*60))
	src := "library T version '1.0'\ndefine R: " + expr + "\n"
	if !imported {
		v, err := NewEngine(WithEvaluationTimestamp(at)).EvaluateExpression(context.Background(), src, "R", nil, nil)
		if err != nil {
			return "error: " + err.Error()
		}
		return valueString(v)
	}
	parsed, err := compiler.Compile(src)
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
	lib, err := elm.Import(read)
	if err != nil {
		t.Fatal(err)
	}
	ctx := eval.NewContext(context.Background(), lib)
	ctx.EvaluationTimestamp = at
	got, err := eval.NewEvaluator(ctx).EvaluateExpression("R")
	if err != nil {
		return "error: " + err.Error()
	}
	return valueString(got)
}

// TestDifferencesByHand holds each difference to the value the specification or
// both reference engines give, at -03:00, compiled and through the ELM.
func TestDifferencesByHand(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		// Appendix B, Difference, the normative example.
		{"difference in months between @2012-01-02 and @2012", "Interval[0, 11]"},
		{"difference in months between @2012-01-01 and @2012-02-01", "1"},
		// A millisecond before midnight crosses an hour, a minute and a second
		// boundary, though no unit has elapsed (the engine measured 0).
		{"difference in hours between @2012-01-31T23:59:59.999 and @2012-02-01T00:00:00.000", "1"},
		{"difference in minutes between @2012-01-31T23:59:59.999 and @2012-02-01T00:00:00.000", "1"},
		{"difference in seconds between @2012-01-31T23:59:59.999 and @2012-02-01T00:00:00.000", "1"},
		{"difference in milliseconds between @2012-01-31T23:59:59.999 and @2012-02-01T00:00:00.000", "1"},
		{"difference in days between @2012-01-31T23:59:59.999 and @2012-02-01T00:00:00.000", "1"},
		{"difference in months between @2012-01-31T23:59:59.999 and @2012-02-01T00:00:00.000", "1"},
		{"difference in years between @2011-12-31T23:59:59.999 and @2012-01-01T00:00:00.000", "1"},
		// 10:59:30 to 11:01:10 crosses one hour and two minute boundaries.
		{"difference in hours between @2012-01-01T10:59:30.000 and @2012-01-01T11:01:10.000", "1"},
		{"difference in minutes between @2012-01-01T10:59:30.000 and @2012-01-01T11:01:10.000", "2"},
		{"difference in seconds between @2012-01-01T10:59:30.000 and @2012-01-01T11:01:10.000", "100"},
		{"difference in hours between @T10:59:30.000 and @T11:01:10.000", "1"},
		{"difference in minutes between @T10:59:30.000 and @T11:01:10.000", "2"},
		{"difference in milliseconds between @T10:59:30.000 and @T11:01:10.000", "100000"},
		{"difference in minutes between @T23:59:59.999 and @T00:00:00.000", "-1439"},
		// Weeks are the Sundays crossed: Saturday to Sunday is one, a whole week
		// less a millisecond from a Sunday is none.
		{"difference in weeks between @2012-01-07 and @2012-01-08", "1"},
		{"difference in weeks between @2012-01-08 and @2012-01-07", "-1"},
		{"difference in weeks between @2012-01-08T00:00:00.000 and @2012-01-14T23:59:59.999", "0"},
		{"difference in weeks between @2011-12-31T12:00:00.000 and @2013-01-01T12:00:00.000", "53"},
		{"difference in days between @2011-12-31T12:00:00.000 and @2013-01-01T12:00:00.000", "367"},
		// Offsets: hours and finer in the evaluation's -03:00, days and coarser
		// on the fields as written. The first pair is one instant.
		{"difference in hours between @2012-01-01T23:30:00.000-03:00 and @2012-01-02T02:30:00.000Z", "0"},
		{"difference in days between @2012-01-01T23:30:00.000-03:00 and @2012-01-02T02:30:00.000Z", "1"},
		{"difference in hours between @2012-01-01T23:00:00.000+14:00 and @2012-01-01T23:00:00.000-11:00", "25"},
		{"difference in days between @2012-01-01T23:00:00.000+14:00 and @2012-01-01T23:00:00.000-11:00", "0"},
		// A value that states no offset is at the evaluation's.
		{"difference in hours between @2012-01-01T23:30:00.000-03:00 and @2012-01-02T00:30:00.000", "1"},
		// 04:25Z and 10:00Z are 01:25 and 07:00 at -03:00, six hour boundaries.
		{"difference in hours between @2012-01-01T10:10:00.000+05:45 and @2012-01-01T10:00:00.000Z", "6"},
		{"difference in minutes between @2012-01-01T10:10:00.000+05:45 and @2012-01-01T10:00:00.000Z", "335"},
		// An uncertainty, in both orders.
		{"difference in hours between @2012-01-31T and @2012-02-01T00:00:00.000", "Interval[1, 24]"},
		{"difference in hours between @2012-02-01T00:00:00.000 and @2012-01-31T", "Interval[-24, -1]"},
		{"difference in days between @2012-02 and @2012-03-01", "Interval[1, 29]"},
		{"difference in days between @2012-03-01 and @2012-02", "Interval[-29, -1]"},
		{"difference in years between @2012-06 and @2014", "2"},
		{"difference in months between @2012 and @2014-06", "Interval[18, 29]"},
		{"difference in weeks between @2012-01-01 and @2011-12", "Interval[-5, -1]"},
		{"difference in hours between @2012-01-01T10:00:00.000-11:00 and @2012-01-02T", "Interval[6, 29]"},
		// Second and millisecond are two precisions here: a value to the second is
		// uncertain in milliseconds.
		{"difference in milliseconds between @2012-01-31T23:59:59 and @2012-02-01T00:00:00.000", "Interval[1, 1000]"},
		{"difference in seconds between @2012-01-31T23:59:59 and @2012-02-01T00:00:00.000", "1"},
		{"difference in milliseconds between @T10:15 and @T12:05:00.000", "Interval[6540001, 6600000]"},
		{"difference in minutes between @T10:15 and @T12:05:00.000", "110"},
		// At the precision asked, or finer, it is exact.
		{"difference in days between @2012-01-31T and @2012-02-01T", "1"},
		{"difference in months between @2012-01T and @2012-02T", "1"},
		{"difference in hours between @2012-01-01T10 and @2012-01-01T11:30", "1"},
		// The interval form reads the same rule.
		{"difference in hours of Interval[@2012-01-31T, @2012-02-01T00:00:00.000]", "Interval[1, 24]"},
		{"difference in days of Interval[@2012-01-01T10:00:00.000, @2012-01-03T09:00:00.000]", "2"},
		{"difference in months of Interval[@2012-01, @2012-03-15]", "2"},
		// Null.
		{"difference in days between @2012-01-01 and null", "null"},
		{"difference in hours between null and @2012-01-01T10:00:00.000", "null"},
		{"difference in hours between @T10:00 and null", "null"},
	} {
		for name, imported := range map[string]bool{"compiled": false, "imported": true} {
			if got := differenceAt(t, tt.expr, -3*60, imported); got != tt.want {
				t.Errorf("%s, %s = %s, want %s", name, tt.expr, got, tt.want)
			}
		}
	}
}

// diffOperand is one operand of a difference, as the oracle below reads it: the
// moments it could be as wall clocks (milliseconds since 1970-01-01 as if the
// wall clock were UTC), the offset it is read at, and its precision.
type diffOperand struct {
	text       string
	kind       string // "date", "datetime" or "time"
	precision  int    // 0 year, 1 month, 2 day, 3 hour, 4 minute, 5 second, 6 millisecond
	offset     int    // minutes, stated or the evaluation's
	stated     bool
	start, end int64
}

var (
	diffDateTimeText = regexp.MustCompile(`^@(\d{4})(?:-(\d\d))?(?:-(\d\d))?(T)?(?:(\d\d)(?::(\d\d))?(?::(\d\d))?(?:\.(\d{3}))?)?(Z|[+-]\d\d:\d\d)?$`)
	diffTimeText     = regexp.MustCompile(`^@T(\d\d)(?::(\d\d))?(?::(\d\d))?(?:\.(\d{3}))?$`)
)

// parseDiffOperand reads a Date, DateTime or Time literal.
func parseDiffOperand(text string) (diffOperand, bool) {
	num := func(s string, def int) int {
		if s == "" {
			return def
		}
		n, _ := strconv.Atoi(s)
		return n
	}
	op := diffOperand{text: text}
	var f [7]int
	if m := diffTimeText.FindStringSubmatch(text); m != nil {
		op.kind, op.precision = "time", 3
		f = [7]int{1970, 1, 1, num(m[1], 0), num(m[2], 0), num(m[3], 0), num(m[4], 0)}
		for i := 2; i >= 0; i-- {
			if m[i+2] != "" {
				op.precision = 4 + i
				break
			}
		}
		if m[2] == "" {
			op.precision = 3
		}
	} else if m := diffDateTimeText.FindStringSubmatch(text); m != nil {
		op.kind = "date"
		if m[4] == "T" {
			op.kind = "datetime"
		}
		f = [7]int{num(m[1], 0), num(m[2], 1), num(m[3], 1), num(m[5], 0), num(m[6], 0), num(m[7], 0), num(m[8], 0)}
		op.precision = 0
		for i, g := range []string{m[1], m[2], m[3], m[5], m[6], m[7], m[8]} {
			if g != "" {
				op.precision = i
			}
		}
		if m[9] != "" {
			op.stated = true
			if m[9] != "Z" {
				h, _ := strconv.Atoi(m[9][1:3])
				mi, _ := strconv.Atoi(m[9][4:6])
				op.offset = (h*60 + mi) * map[byte]int{'+': 1, '-': -1}[m[9][0]]
			}
		}
	} else {
		return diffOperand{}, false
	}
	// The range is the period the precision states: from its start to the start
	// of the next, less a millisecond.
	start := time.Date(f[0], time.Month(f[1]), f[2], f[3], f[4], f[5], f[6]*1e6, time.UTC)
	next := map[int]time.Time{
		0: start.AddDate(1, 0, 0), 1: start.AddDate(0, 1, 0), 2: start.AddDate(0, 0, 1),
		3: start.Add(time.Hour), 4: start.Add(time.Minute), 5: start.Add(time.Second), 6: start.Add(time.Millisecond),
	}[op.precision]
	op.start, op.end = start.UnixMilli(), next.UnixMilli()-1
	return op, true
}

// floorDiv rounds towards minus infinity.
func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// boundaries is the count of boundaries of the unit from the wall clock a to b:
// years to days on the wall clocks as written, hours and finer on the instants
// in the frame, each floored to the unit. A boundary is a change of
// floor(moment / unit), whatever the distance between the moments.
func boundaries(unit string, a, b int64, aOff, bOff, frame int, isTime bool) int64 {
	const day = 86400000
	switch unit {
	case "years":
		return int64(time.UnixMilli(b).UTC().Year() - time.UnixMilli(a).UTC().Year())
	case "months":
		month := func(ms int64) int64 {
			t := time.UnixMilli(ms).UTC()
			return int64(t.Year())*12 + int64(t.Month())
		}
		return month(b) - month(a)
	case "weeks":
		// 1970-01-01 was a Thursday, so the Sunday-first week index is (day+4)/7.
		return floorDiv(floorDiv(b, day)+4, 7) - floorDiv(floorDiv(a, day)+4, 7)
	case "days":
		return floorDiv(b, day) - floorDiv(a, day)
	}
	local := func(wall int64, off int) int64 {
		if isTime {
			return wall
		}
		return wall - int64(off)*60000 + int64(frame)*60000
	}
	size := map[string]int64{"hours": 3600000, "minutes": 60000, "seconds": 1000, "milliseconds": 1}[unit]
	return floorDiv(local(b, bOff), size) - floorDiv(local(a, aOff), size)
}

// differenceOracle is the answer text for the unit, written from the
// definition and not from funcs.DifferenceBetweenAt: the boundaries crossed
// between the operands' extremes, as Language Semantics (Determining Difference
// and Duration) gives them, an integer or the uncertainty of two, in the frame
// (minutes) of the evaluation.
func differenceOracle(unit string, a, b diffOperand, frame int) string {
	if !a.stated {
		a.offset = frame
	}
	if !b.stated {
		b.offset = frame
	}
	index := map[string]int{"years": 0, "months": 1, "weeks": 2, "days": 2, "hours": 3, "minutes": 4, "seconds": 5, "milliseconds": 6}[unit]
	isTime := a.kind == "time" || b.kind == "time"
	count := func(x, y int64) int64 { return boundaries(unit, x, y, a.offset, b.offset, frame, isTime) }
	if a.precision >= index && b.precision >= index {
		return strconv.FormatInt(count(a.start, b.start), 10)
	}
	low, high := count(a.end, b.start), count(a.start, b.end)
	if low == high {
		return strconv.FormatInt(low, 10)
	}
	return fmt.Sprintf("Interval[%d, %d]", low, high)
}

var differenceUnits = []string{"years", "months", "weeks", "days", "hours", "minutes", "seconds", "milliseconds"}

// differenceGrid is the operands of the grid: ten moments around the edges (the
// end of a month and of a year, a Sunday, a leap day, midnight), each written
// from a year to a millisecond as a Date and a DateTime, and as a DateTime with
// an offset; and as a Time to the hour, minute, second and millisecond.
func differenceGrid() (dated, timed []diffOperand) {
	for _, m := range [][7]int{
		{2012, 1, 31, 23, 59, 59, 999}, {2012, 2, 1, 0, 0, 0, 0}, {2012, 1, 1, 10, 59, 30, 0},
		{2012, 1, 1, 11, 1, 10, 0}, {2011, 12, 31, 12, 0, 0, 0}, {2013, 1, 1, 12, 0, 0, 0},
		{2012, 3, 3, 8, 0, 0, 0}, {2012, 2, 29, 23, 30, 0, 250}, {2011, 6, 30, 23, 0, 0, 0},
		{2012, 1, 1, 0, 0, 0, 0},
	} {
		day := fmt.Sprintf("%04d-%02d-%02d", m[0], m[1], m[2])
		clock := fmt.Sprintf("%02d:%02d:%02d.%03d", m[3], m[4], m[5], m[6])
		texts := make([]string, 0, 16)
		texts = append(texts,
			fmt.Sprintf("@%04d", m[0]), fmt.Sprintf("@%04d-%02d", m[0], m[1]), "@"+day,
			fmt.Sprintf("@%04dT", m[0]), fmt.Sprintf("@%04d-%02dT", m[0], m[1]), "@"+day+"T",
			fmt.Sprintf("@%sT%02d", day, m[3]), fmt.Sprintf("@%sT%02d:%02d", day, m[3], m[4]),
			fmt.Sprintf("@%sT%02d:%02d:%02d", day, m[3], m[4], m[5]), "@"+day+"T"+clock,
		)
		for _, off := range []string{"Z", "+05:45", "-11:00"} {
			texts = append(texts, fmt.Sprintf("@%sT%02d:%02d%s", day, m[3], m[4], off), "@"+day+"T"+clock+off)
		}
		for _, text := range texts {
			if op, ok := parseDiffOperand(text); ok {
				dated = append(dated, op)
			}
		}
		for _, text := range []string{
			fmt.Sprintf("@T%02d", m[3]), fmt.Sprintf("@T%02d:%02d", m[3], m[4]),
			fmt.Sprintf("@T%02d:%02d:%02d", m[3], m[4], m[5]), "@T" + clock,
		} {
			if op, ok := parseDiffOperand(text); ok {
				timed = append(timed, op)
			}
		}
	}
	return dated, timed
}

// differencePairs pairs the operands of the grid: a Date with a Date and a
// DateTime with a DateTime (a Date beside a DateTime is plan 019), a Time with a
// Time. Dates are asked only in the units up to days.
func differencePairs() []struct {
	a, b diffOperand
	unit string
} {
	dated, timed := differenceGrid()
	var pairs []struct {
		a, b diffOperand
		unit string
	}
	add := func(a, b diffOperand, units []string) {
		for _, u := range units {
			pairs = append(pairs, struct {
				a, b diffOperand
				unit string
			}{a, b, u})
		}
	}
	for i, a := range dated {
		for k := 0; k < 6; k++ {
			b := dated[(i*37+k*53+11)%len(dated)]
			switch {
			case a.kind != b.kind:
			case a.kind == "date":
				add(a, b, differenceUnits[:4])
			default:
				add(a, b, differenceUnits)
			}
		}
	}
	for i, a := range timed {
		for k := 0; k < 12; k++ {
			add(a, timed[(i*7+k*3+1)%len(timed)], differenceUnits[4:])
		}
	}
	return pairs
}

// TestDifferencesAtOtherOffsets holds the engine, at the evaluation
// offsets -03:00, +05:45 and +14:00 (a quarter of an hour is where truncating
// the instant and truncating the wall clock part), to differenceOracle.
func TestDifferencesAtOtherOffsets(t *testing.T) {
	pairs := differencePairs()
	for _, frame := range []int{-3 * 60, 5*60 + 45, 14 * 60} {
		engine := NewEngine(WithEvaluationTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", frame*60))))
		cases := make([]uncertainCase, 0, len(pairs))
		for _, p := range pairs {
			cases = append(cases, uncertainCase{
				fmt.Sprintf("difference in %s between %s and %s", p.unit, p.a.text, p.b.text),
				differenceOracle(p.unit, p.a, p.b, frame),
			})
		}
		evaluateCases(t, engine, cases)
	}
	if len(pairs) < 4000 {
		t.Fatalf("the grid covered %d differences", len(pairs))
	}
}

// answersOf asks for the answer of each expression in one library.
func answersOf(t *testing.T, frame int, exprs []string) []string {
	t.Helper()
	engine := NewEngine(WithEvaluationTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", frame*60))))
	answers := make([]string, 0, len(exprs))
	const chunk = 120
	for from := 0; from < len(exprs); from += chunk {
		part := exprs[from:min(from+chunk, len(exprs))]
		var src strings.Builder
		src.WriteString("library T version '1.0'\n")
		for i, e := range part {
			fmt.Fprintf(&src, "define D%d: %s\n", i, e)
		}
		got, err := engine.EvaluateLibrary(context.Background(), src.String(), nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		for i := range part {
			answers = append(answers, valueString(got[fmt.Sprintf("D%d", i)]))
		}
	}
	return answers
}

// negated is the answer for the operands the other way round: an integer
// negated, an uncertainty negated and its bounds swapped.
func negated(answer string) string {
	if n, err := strconv.ParseInt(answer, 10, 64); err == nil {
		return strconv.FormatInt(-n, 10)
	}
	var lo, hi int64
	if _, err := fmt.Sscanf(answer, "Interval[%d, %d]", &lo, &hi); err == nil {
		return fmt.Sprintf("Interval[%d, %d]", -hi, -lo)
	}
	return answer
}

// TestReversingTheOperandsNegatesTheDifference is a metamorphic check of the
// grid: the boundaries from b to a are those from a to b, negated, and an
// uncertainty is negated and its bounds swapped.
func TestReversingTheOperandsNegatesTheDifference(t *testing.T) {
	pairs := differencePairs()
	forward, backward := make([]string, 0, len(pairs)), make([]string, 0, len(pairs))
	for _, p := range pairs {
		forward = append(forward, fmt.Sprintf("difference in %s between %s and %s", p.unit, p.a.text, p.b.text))
		backward = append(backward, fmt.Sprintf("difference in %s between %s and %s", p.unit, p.b.text, p.a.text))
	}
	for _, frame := range []int{-3 * 60, 5*60 + 45} {
		f, b := answersOf(t, frame, forward), answersOf(t, frame, backward)
		bad := 0
		for i := range pairs {
			if f[i] != negated(b[i]) {
				bad++
				if bad <= 20 {
					t.Errorf("at %d: %s = %s but the reverse = %s", frame, forward[i], f[i], b[i])
				}
			}
		}
	}
}

// TestTheSameInstantInAnotherOffsetIsTheSameBelowTheDay writes one instant at
// four offsets: from an hour to a millisecond the difference does not depend on
// the offset it is written at, because both are normalized to the evaluation's.
func TestTheSameInstantInAnotherOffsetIsTheSameBelowTheDay(t *testing.T) {
	instants := []time.Time{
		time.Date(2012, 1, 31, 23, 59, 59, 999e6, time.UTC), time.Date(2012, 2, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2012, 1, 1, 10, 59, 30, 0, time.UTC), time.Date(2011, 12, 31, 12, 0, 0, 0, time.UTC),
		time.Date(2012, 2, 29, 3, 30, 0, 250e6, time.UTC),
	}
	zones := []struct {
		suffix string
		minute int
	}{{"Z", 0}, {"+05:45", 345}, {"-11:00", -660}, {"+14:00", 840}}
	write := func(at time.Time, z int) string {
		return "@" + at.In(time.FixedZone("", z*60)).Format("2006-01-02T15:04:05.000")
	}
	for _, frame := range []int{-3 * 60, 5*60 + 45} {
		exprs := make([]string, 0, len(zones)*len(zones)*(len(differenceUnits)-4)*len(instants)*len(instants))
		for _, a := range instants {
			for _, b := range instants {
				for _, u := range differenceUnits[4:] {
					for _, za := range zones {
						for _, zb := range zones {
							exprs = append(exprs, fmt.Sprintf("difference in %s between %s%s and %s%s",
								u, write(a, za.minute), za.suffix, write(b, zb.minute), zb.suffix))
						}
					}
				}
			}
		}
		answers := answersOf(t, frame, exprs)
		bad := 0
		for i := 0; i < len(exprs); i += len(zones) * len(zones) {
			for j := 1; j < len(zones)*len(zones); j++ {
				if answers[i+j] != answers[i] {
					bad++
					if bad <= 20 {
						t.Errorf("at %d: %s = %s but %s = %s", frame, exprs[i], answers[i], exprs[i+j], answers[i+j])
					}
				}
			}
		}
	}
}

// TestTheDifferencesAreaIsItsOracle holds every `difference in X between` of the
// snapshot area `differences` whose operands are literals to the oracle above,
// so that a divergence from a reference there is classified against a value the
// engine did not compute (plan 020). The area is evaluated at -03:00. A Date in
// a unit below the day, an operand that is not a literal, and an answer
// outside the range of an Integer (backlog 1c31) are left out.
func TestTheDifferencesAreaIsItsOracle(t *testing.T) {
	define := regexp.MustCompile(`(?m)^define (D\d+): difference in (\w+) between (@\S+) and (@\S+)$`)
	cql, err := os.ReadFile("testdata/snapshots/differences.cql")
	if err != nil {
		t.Fatal(err)
	}
	answers := map[string]string{}
	raw, err := os.ReadFile("testdata/snapshots/differences.answers.tsv")
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
		a, okA := parseDiffOperand(m[3])
		b, okB := parseDiffOperand(m[4])
		if !okA || !okB || a.kind != b.kind {
			continue
		}
		if a.kind == "date" && (m[2] == "hours" || m[2] == "minutes" || m[2] == "seconds" || m[2] == "milliseconds") {
			continue
		}
		want := differenceOracle(m[2], a, b, -3*60)
		got := strings.NewReplacer("Uncertainty[Integer ", "Interval[", ", Integer ", ", ", "Integer ", "").Replace(answers[m[1]])
		if outsideInteger(want) {
			continue
		}
		if got != want {
			t.Errorf("%s: %s = %s, the oracle gives %s", m[1], m[0], got, want)
		}
		checked++
	}
	if checked < 600 {
		t.Fatalf("only %d differences of the area were checked against the oracle", checked)
	}
}

// outsideInteger says whether an answer has a number beyond an Integer's 32 bits.
func outsideInteger(answer string) bool {
	for _, n := range regexp.MustCompile(`-?\d+`).FindAllString(answer, -1) {
		if v, err := strconv.ParseInt(n, 10, 64); err != nil || v > math.MaxInt32 || v < math.MinInt32 {
			return true
		}
	}
	return false
}
