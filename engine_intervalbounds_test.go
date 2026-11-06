package cql

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// The tests of plan 023 (docs/plans/023-interval-start-end.md): `duration in X
// of I` and `difference in X of I` measure between `start of I` and `end of I`,
// and those read a closed null boundary as the minimum or maximum of the point
// type.
// CQL 1.5.3 (https://cql.hl7.org/N1A/), Author's Guide, Interval Operators:
// "duration in days of X" is "duration in days between start of X and end of
// X", and the same for difference. Appendix B, Start: "If the low boundary of
// the interval is open, this operator returns the successor of the low value of
// the interval. Note that if the low value of the interval is null, the result
// is null. If the low boundary of the interval is closed and the low value of
// the interval is not null, this operator returns the low value of the
// interval. Otherwise, the result is the minimum value of the point type of the
// interval"; End is symmetric.

// TestIntervalBoundsByHand holds each row to the value the specification and
// Java 5.4.0 give, at -03:00, compiled and through the ELM.
func TestIntervalBoundsByHand(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"duration in days of Interval(@2012-01-01, @2012-01-05)", "2"},
		{"duration in days of Interval[@2012-01-01, @2012-01-05)", "3"},
		{"duration in days of Interval(@2012-01-01, @2012-01-05]", "3"},
		{"difference in days of Interval(@2012-01-01, @2012-01-05)", "2"},
		{"difference in days of Interval[@2012-01-01, @2012-01-05)", "3"},
		{"duration in hours of Interval(@2012-01-01T10:00, @2012-01-01T12:00)", "1"},
		{"duration in milliseconds of Interval(@2012-01-01T10:00:00.000, @2012-01-01T10:00:01.000)", "998"},
		{"duration in days of Interval(@2012-01-01T10:00:00.000, @2012-01-05T10:00:00.000)", "3"},
		{"duration in hours of Interval(@T10:00, @T12:00)", "1"},
		// A closed null boundary is the minimum or maximum of the point type.
		{"duration in days of Interval[@2012-01-01, null]", "2917556"},
		{"duration in days of Interval[null, @2012-01-01]", "734502"},
		// An open null boundary stays null in 1.5.3.
		{"duration in days of Interval[@2012-01-01, null)", "null"},
		{"duration in days of Interval(null, @2012-01-01]", "null"},
		{"difference in days of Interval[@2012-01-01, null]", "2917556"},
		{"difference in days of Interval[null, @2012-01-01]", "734502"},
		{"duration in days of Interval[@2012-01-01T00:00:00.000, null]", "2917556"},
		{"duration in years of Interval[@2012-01-01T00:00:00.000, null]", "7987"},
		{"difference in years of Interval[@2012-01-01T00:00:00.000, null]", "7987"},
		{"duration in hours of Interval[@T10:00, null]", "13"},
		{"duration in hours of Interval(@T10:00, null]", "13"},
		// The maximum DateTime is at UTC, as in Java: 20 hours from 03:00Z.
		// cql-execution says 23, with the maximum at -03:00, a defect there.
		{"duration in hours of Interval[@9999-12-31T00:00:00.000-03:00, null]", "20"},
		// No points: from January 2 to January 1, by composition. Java raises
		// Invalid Interval, cql-execution answers -1 (plan 023, divergence).
		{"duration in days of Interval(@2012-01-01, @2012-01-02)", "-1"},
		{"duration in days of Interval[null as Date, null]", "null"},
		{"start of Interval[null, @2012-01-01]", "0001-01-01"},
		{"end of Interval[@2012-01-01, null]", "9999-12-31"},
		{"end of Interval[@T10:00, null]", "23:59:59.999"},
		{"end of Interval[@2012-01-01T00:00:00.000, null]", "9999-12-31T23:59:59.999Z"},
		{"start of Interval[null, @2012-01-01T00:00:00.000]", "0001-01-01T00:00:00.000Z"},
		{"end of Interval[@2012-01-01T00:00:00.000, null] = maximum DateTime", "true"},
		{"start of Interval[null, @2012-01-01T00:00:00.000] = minimum DateTime", "true"},
		{"end of Interval[@2012-01-01, null)", "null"},
		// No successor, no predecessor.
		{"start of Interval(@9999-12-31, null]", "null"},
		{"duration in days of Interval(@9999-12-31, null]", "null"},
		{"duration in days of Interval[null, @0001-01-01)", "null"},
		{"end of Interval[null, @0001-01-01)", "null"},
		// Integer, Long, Decimal and Quantity are plan 024's.
		{"end of Interval[1, null]", "null"},
	} {
		for _, imported := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/imported=%v", tt.expr, imported), func(t *testing.T) {
				if got := differenceAt(t, tt.expr, -3*60, imported); got != tt.want {
					t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
				}
			})
		}
	}
}

// TestDurationOfAnIntervalIsBetweenItsStartAndEnd holds `duration in U of I`
// and `difference in U of I` to the same phrases over `start of I` and `end of
// I`, over a grid of intervals, in every unit valid for the type, compiled and
// through the ELM (Author's Guide, Interval Operators).
func TestDurationOfAnIntervalIsBetweenItsStartAndEnd(t *testing.T) {
	dateUnits := []string{"years", "months", "weeks", "days"}
	timeUnits := []string{"hours", "minutes", "seconds", "milliseconds"}
	type kind struct {
		name   string
		units  []string
		values []string // the boundary values, of different precisions
	}
	kinds := []kind{
		{"Date", dateUnits, []string{"@2012", "@2012-06", "@2012-01-01", "@2012-01-05", "@0001-01-01", "@9999-12-31"}},
		{"DateTime", append(append([]string{}, dateUnits...), timeUnits...), []string{
			"@2012T", "@2012-06T", "@2012-01-01T", "@2012-01-01T10", "@2012-01-01T10:00", "@2012-01-05T10:30:15",
			"@2012-01-01T00:00:00.000", "@2012-01-05T10:00:00.000", "@2012-01-01T00:00:00.000Z",
			"@2012-01-05T10:00:00.000+03:00", "@0001-01-01T00:00:00.000", "@9999-12-31T23:59:59.999Z"}},
		{"Time", timeUnits, []string{"@T10", "@T10:00", "@T10:30:15", "@T12:00:00.000", "@T00:00:00.000", "@T23:59:59.999"}},
	}
	forms := []func(v string) (string, bool){ // the bound written, and whether it is null
		func(v string) (string, bool) { return v, false },
		func(v string) (string, bool) { return v, false },
		func(string) (string, bool) { return "null", true },
		func(string) (string, bool) { return "null", true },
	}
	closedLow := []string{"[", "(", "[", "("}
	closedHigh := []string{"]", ")", "]", ")"}
	checked := 0
	for _, k := range kinds {
		for _, lo := range k.values {
			for _, hi := range k.values {
				for i := range forms {
					for j := range forms {
						l, lNull := forms[i](lo)
						h, hNull := forms[j](hi)
						if lNull && hNull {
							continue
						}
						if lNull && lo != k.values[0] && lo != k.values[3] || hNull && hi != k.values[0] && hi != k.values[3] {
							continue // a null bound does not depend on the value dropped
						}
						iv := closedLow[i] + l + ", " + h + closedHigh[j]
						ivs := "Interval" + iv
						for _, u := range k.units {
							for _, form := range []struct{ of, between string }{
								{"duration in " + u + " of ", u + " between "},
								{"difference in " + u + " of ", "difference in " + u + " between "},
							} {
								of := form.of + ivs
								between := form.between + "start of " + ivs + " and end of " + ivs
								for _, imported := range []bool{false, true} {
									a := snapshotErrorPosition.ReplaceAllString(differenceAt(t, of, -3*60, imported), "")
									b := snapshotErrorPosition.ReplaceAllString(differenceAt(t, between, -3*60, imported), "")
									if a != b {
										t.Errorf("imported=%v: %s = %s, but %s = %s", imported, of, a, between, b)
									}
									checked++
								}
							}
						}
					}
				}
			}
		}
	}
	if checked < 20000 {
		t.Fatalf("only %d comparisons were made", checked)
	}
}

// TestAnIntervalPromotedToDateTimeHasDateTimeBounds holds what plan 019 settled:
// the compiler types the `if` as Interval<DateTime> and the Date interval is
// promoted to it (CQL 1.5.3, https://cql.hl7.org/N1A/, Developer's Guide,
// Implicit Conversions, table 3-J: Date to DateTime is implicit), so its closed
// null end is the maximum DateTime, and the hours to it are counted from the
// midnight of 9999-12-30 at the evaluation's offset to 23:59:59.999Z.
// Java 5.4.0 gives Interval[21, 44] (snapshot area intervalbounds, D590);
// cql-execution Uncertainty[24, 47]. The engine answered 24, holding Dates.
func TestAnIntervalPromotedToDateTimeHasDateTimeBounds(t *testing.T) {
	expr := "duration in hours of (if true then Interval[@9999-12-30, null] else Interval[@9999-12-30T00:00:00.000, null])"
	if got := differenceAt(t, expr, -3*60, false); got != "Interval[21, 44]" {
		t.Errorf("%s = %s, want Interval[21, 44]", expr, got)
	}
}

// TestTheIntervalBoundsAreTheirOracle holds the snapshot area `intervalbounds`
// to testdata/intervalbounds/oracle.tsv, which scripts/intervalbounds-area.py
// writes with `--oracle` and which shares nothing with the engine: Start and End
// by CQL 1.5.3, days and months by Julian day number, hours to milliseconds by
// instants in UTC. The oracle answers the rows where both ends are known and
// at the unit's precision or finer; the others are left to the references. The
// rows are counted by what they ask, and each kind has to be well represented.
func TestTheIntervalBoundsAreTheirOracle(t *testing.T) {
	raw, err := os.ReadFile("testdata/intervalbounds/oracle.tsv")
	if err != nil {
		t.Fatal(err)
	}
	cql, err := os.ReadFile("testdata/snapshots/intervalbounds.cql")
	if err != nil {
		t.Fatal(err)
	}
	asked := map[string]string{}
	for _, m := range snapshotDefine.FindAllStringSubmatch(string(cql), -1) {
		asked[m[1]] = m[0]
	}
	answers := readAnswers(t, "intervalbounds")
	classes := map[string]int{}
	checked := 0
	for _, line := range strings.Split(string(raw), "\n") {
		name, want, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		def, found := asked[name]
		if !found {
			t.Errorf("%s is in the oracle and not in the area", name)
			continue
		}
		// readAnswers drops the "Integer " that the oracle writes.
		want = strings.TrimPrefix(want, "Integer ")
		if got := answers[name]; got != want {
			t.Errorf("%s: %s = %s, the oracle gives %s", name, def, got, want)
		}
		class := "start"
		switch body := strings.TrimPrefix(def, "define "+name+": "); {
		case want == "null":
			class = "null"
		case strings.HasPrefix(body, "duration in "):
			class = "duration"
		case strings.HasPrefix(body, "difference in "):
			class = "difference"
		case strings.HasPrefix(body, "end of "):
			class = "end"
		}
		classes[class]++
		checked++
	}
	if checked < 450 {
		t.Fatalf("only %d definitions of the area were checked against the oracle", checked)
	}
	for class, least := range map[string]int{"duration": 120, "difference": 120, "start": 35, "end": 35, "null": 60} {
		if classes[class] < least {
			t.Errorf("only %d %s rows were checked against the oracle, want at least %d", classes[class], class, least)
		}
	}
	t.Logf("%d rows against the oracle: %v", checked, classes)
}

// TestAPeriodOpenAtTheEndLastsToTheLastInstant reads a FHIR Period with no end
// through FHIRHelpers.ToInterval, which gives a closed null high bound; its
// end is the maximum DateTime (CQL 1.5.3, https://cql.hl7.org/N1A/, Appendix B,
// End: "Otherwise, the result is the maximum value of the point type of the
// interval"). The snapshot area cannot hold these rows: the reference harness
// gives the reference engines no FHIR model there.
func TestAPeriodOpenAtTheEndLastsToTheLastInstant(t *testing.T) {
	const header = "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1'\n"
	const period = "FHIRHelpers.ToInterval(FHIR.Period { start: FHIR.dateTime { value: @2012-01-01T00:00:00.000Z } })"
	engine := NewEngine(WithEvaluationTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", -3*3600))))
	for expr, want := range map[string]string{
		"duration in days of " + period:            "2917556",
		"end of " + period:                         "9999-12-31T23:59:59.999Z",
		"end of " + period + " = maximum DateTime": "true",
	} {
		v, err := engine.EvaluateExpression(context.Background(), header+"define R: "+expr+"\n", "R", nil, nil)
		got := "null"
		if v != nil {
			got = valueString(v)
		}
		if err != nil || got != want {
			t.Errorf("%s = %s, %v; want %s", expr, got, err, want)
		}
	}
}
