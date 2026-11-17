package cql

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// expansionNumber and expansionDateTime find the parts of an answer whose text
// says more than its value: a Decimal's trailing zeros, and the T and offset
// of a DateTime coarser than the hour, which no comparison of the area reads.
var (
	expansionNumber   = regexp.MustCompile(`(Decimal|Quantity) (-?)(\d+)(?:\.(\d+))?`)
	expansionDateTime = regexp.MustCompile(`DateTime (\d{4}(?:-\d\d){0,2})(T(?:\d\d(?::\d\d(?::\d\d(?:\.\d+)?)?)?)?)?(Z|[+-]\d\d:\d\d)?`)
)

// canonicalExpansion writes an answer of the area so that two texts of the same
// value are the same text.
func canonicalExpansion(s string) string {
	s = expansionNumber.ReplaceAllStringFunc(s, func(m string) string {
		p := expansionNumber.FindStringSubmatch(m)
		frac := strings.TrimRight(p[4], "0")
		sign := p[2]
		if strings.Trim(p[3], "0") == "" && frac == "" {
			sign = ""
		}
		if frac != "" {
			frac = "." + frac
		}
		return p[1] + " " + sign + p[3] + frac
	})
	return expansionDateTime.ReplaceAllStringFunc(s, func(m string) string {
		p := expansionDateTime.FindStringSubmatch(m)
		if p[2] == "" || p[2] == "T" {
			return "DateTime " + p[1]
		}
		return "DateTime " + p[1] + p[2] + p[3]
	})
}

// TestTheExpansionsAreTheirOracle holds the snapshot area `expansions` to
// testdata/expansions/oracle.tsv, which scripts/expansions-area.py writes with
// `--oracle` and which shares nothing with the engine: expand … per by CQL
// 1.5.3, Expand, as plan 026 reads it (docs/plans/026-expand-truncates-to-per.md),
// over ordinals of the calendar and integers at per's scale. The rows are
// counted by kind and overload, and each has to be well represented.
func TestTheExpansionsAreTheirOracle(t *testing.T) {
	raw, err := os.ReadFile("testdata/expansions/oracle.tsv")
	if err != nil {
		t.Fatal(err)
	}
	cql, err := os.ReadFile("testdata/snapshots/expansions.cql")
	if err != nil {
		t.Fatal(err)
	}
	asked := map[string]string{}
	for _, m := range snapshotDefine.FindAllStringSubmatch(string(cql), -1) {
		asked[m[1]] = strings.TrimPrefix(m[0], "define "+m[1]+": ")
	}
	answers := readSnapshot(t, "testdata/snapshots/expansions.answers.tsv")
	classes := map[string]int{}
	checked := 0
	for _, line := range strings.Split(string(raw), "\n") {
		name, want, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		expr, found := asked[name]
		if !found {
			t.Errorf("%s is in the oracle and not in the area", name)
			continue
		}
		if got := canonicalExpansion(answers[name]); got != canonicalExpansion(want) {
			t.Errorf("%s: %s = %s, the oracle gives %s", name, expr, answers[name], want)
		}
		class := "points"
		if strings.HasPrefix(expr, "expand {") {
			class = "list"
		}
		switch {
		case want == "List[]":
			class += " empty"
		case strings.Contains(expr, "@T"):
			class += " Time"
		case regexp.MustCompile(`@\d{4}(?:-\d\d){0,2}T`).MatchString(expr):
			class += " DateTime"
		case strings.Contains(expr, "@"):
			class += " Date"
		default:
			class += " number"
		}
		classes[class]++
		checked++
	}
	if checked < 800 {
		t.Fatalf("only %d definitions of the area were checked against the oracle", checked)
	}
	for class, least := range map[string]int{
		"list DateTime": 300, "list Date": 50, "list Time": 40, "list number": 60,
		"points DateTime": 80, "points Time": 15, "points number": 20, "list empty": 30,
	} {
		if classes[class] < least {
			t.Errorf("only %d %s rows were checked against the oracle, want at least %d", classes[class], class, least)
		}
	}
	t.Logf("%d rows against the oracle: %v", checked, classes)
}
