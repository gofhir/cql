package cql

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/gofhir/cql/compiler"
	"github.com/gofhir/cql/model"
	"github.com/gofhir/cql/sema"
)

// The tests of the snapshot area `mixeddates` of plan 019
// (docs/plans/019-date-promotion.md), written by scripts/mixeddates-area.py: a
// Date next to a DateTime is converted to a DateTime (CQL 1.5.3,
// https://cql.hl7.org/N1A/, Developer's Guide, Implicit Conversions, table 3-J).

// mixedDatesFamily is what a definition of the area asks, by its text.
func mixedDatesFamily(body string) string {
	switch {
	case strings.Contains(body, " is DateTime"):
		return "type"
	case strings.HasPrefix(body, "difference in ") || regexp.MustCompile(`^(years|months|days|hours) between `).MatchString(body):
		return "measure"
	case strings.Contains(body, " between "):
		return "between"
	case regexp.MustCompile(`Interval\[[^\]]*\] (overlaps|overlaps day of|during|includes|starts|ends|=|properly includes|meets|union|intersect|except) Interval`).MatchString(body):
		return "intervals"
	case strings.Contains(body, " in Interval[") || strings.Contains(body, " during Interval[") ||
		strings.Contains(body, " in day of Interval[") || regexp.MustCompile(`^Interval\[[^\]]*\] (includes|contains) @`).MatchString(body):
		return "point-in-interval"
	case strings.Contains(body, "{ @"):
		return "list"
	case regexp.MustCompile(` (same day as|before|after day of|same or before|within 3 days of) `).MatchString(body):
		return "phrase"
	}
	return "compare"
}

// TestTheMixedDatesAreTheirOracle holds the snapshot area `mixeddates` to
// testdata/mixeddates/oracle.tsv, which scripts/mixeddates-area.py writes with
// `--oracle` and which shares nothing with the engine: the comparison of two
// DateTimes by the specification (the fields in order up to the lesser
// precision, null where the values agree to there and one is the more precise;
// offsets normalized only from the hour), a Date converted at the evaluation's
// offset, the boundaries crossed or the whole periods between two values, as a
// range where one is coarser than the unit. It answers the rows whose every
// step it can decide, and the others are left to the references. The rows are
// counted by what they ask, and each family has to be well represented.
func TestTheMixedDatesAreTheirOracle(t *testing.T) {
	raw, err := os.ReadFile("testdata/mixeddates/oracle.tsv")
	if err != nil {
		t.Fatal(err)
	}
	cql, err := os.ReadFile("testdata/snapshots/mixeddates.cql")
	if err != nil {
		t.Fatal(err)
	}
	asked := map[string]string{}
	for _, m := range snapshotDefine.FindAllStringSubmatch(string(cql), -1) {
		asked[m[1]] = m[0]
	}
	answers := readAnswers(t, "mixeddates")
	families := map[string]int{}
	kinds := map[string]int{}
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
		families[mixedDatesFamily(strings.TrimPrefix(def, "define "+name+": "))]++
		switch {
		case want == "null":
			kinds["null"]++
		case strings.HasPrefix(want, "Boolean true"):
			kinds["true"]++
		case strings.HasPrefix(want, "Boolean false"):
			kinds["false"]++
		case strings.HasPrefix(want, "Uncertainty"):
			kinds["uncertainty"]++
		default:
			kinds["integer"]++
		}
		checked++
	}
	if checked < 500 {
		t.Fatalf("only %d definitions of the area were checked against the oracle", checked)
	}
	for family, least := range map[string]int{"compare": 100, "between": 25, "phrase": 40, "point-in-interval": 50,
		"list": 30, "intervals": 60, "measure": 40, "type": 30} {
		if families[family] < least {
			t.Errorf("only %d %s rows were checked against the oracle, want at least %d", families[family], family, least)
		}
	}
	for kind, least := range map[string]int{"null": 20, "true": 60, "false": 60, "uncertainty": 10, "integer": 10} {
		if kinds[kind] < least {
			t.Errorf("only %d %s answers were checked against the oracle, want at least %d", kinds[kind], kind, least)
		}
	}
	t.Logf("%d rows against the oracle: %v %v", checked, families, kinds)
}

// mixedDatesKnownDifferences are the definitions of the area where the
// translator and this engine do not convert alike, each with why. The check
// below fails on a difference not listed, and on a listed one that is gone, so
// that what a change settles is written down.
var mixedDatesKnownDifferences = map[string]string{}

// TestTheMixedDatesPromotionsAreTheTranslators compares, definition by
// definition, where the reference translator (cql-to-elm 5.4.0) inserts
// ToDateTime and where sema records a DatePromotion (plan 019, item 8). The
// number of nodes differs by construction (the translator writes the conversion
// of an Interval<Date> as two, one per bound, and of `between` as one per
// comparison), so the check is that a definition has some on one side if and
// only if it has some on the other. The translator's side is
// testdata/snapshots/mixeddates.promotions.tsv, which scripts/references.sh
// writes from the ELM of every definition; one the translator refused has none.
func TestTheMixedDatesPromotionsAreTheTranslators(t *testing.T) {
	raw, err := os.ReadFile("testdata/snapshots/mixeddates.promotions.tsv")
	if err != nil {
		t.Fatal(err)
	}
	cql, err := os.ReadFile("testdata/snapshots/mixeddates.cql")
	if err != nil {
		t.Fatal(err)
	}
	src := string(cql)
	header := src[:snapshotDefine.FindStringIndex(src)[0]]
	body := map[string]string{}
	for _, m := range snapshotDefine.FindAllStringSubmatch(src, -1) {
		body[m[1]] = m[0]
	}
	info, err := model.LoadR4ModelInfo()
	if err != nil {
		t.Fatal(err)
	}
	m := sema.FromModelInfo(info)

	var differ, seen []string
	rejected, agreed := 0, 0
	for _, line := range strings.Split(string(raw), "\n") {
		name, java, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		def, found := body[name]
		if !found {
			t.Errorf("%s is in the promotions and not in the area", name)
			continue
		}
		if contains(seen, name) {
			t.Errorf("%s is twice in the promotions", name)
			continue
		}
		seen = append(seen, name)
		if java == "rejected" {
			rejected++
			continue // the translator refuses it: there is no ELM to compare
		}
		if strings.Contains(def, "ToDateTime(") {
			continue // the node is the one written out, not a conversion
		}
		nodes, err := strconv.Atoi(java)
		if err != nil {
			t.Fatalf("%s: %q is not a count", name, java)
		}
		lib, err := compiler.Compile(header + def + "\n")
		if err != nil {
			t.Errorf("%s: %v", def, err)
			continue
		}
		recorded := sema.Check(lib, m).DatePromotionsByDefine[name]
		if (nodes > 0) != (recorded > 0) {
			differ = append(differ, name)
			if _, known := mixedDatesKnownDifferences[name]; !known {
				t.Errorf("%s: %s\n\tthe translator wrote %d ToDateTime nodes, sema records %d Date promotions", name, def, nodes, recorded)
			}
			continue
		}
		agreed++
	}
	if len(seen) != len(body) {
		t.Errorf("the promotions name %d definitions and the area holds %d", len(seen), len(body))
	}
	for name, why := range mixedDatesKnownDifferences {
		if !contains(differ, name) {
			t.Errorf("%s is listed as a known difference (%s) and now agrees: remove it from the list", name, why)
		}
	}
	if agreed < 500 {
		t.Errorf("only %d definitions agreed with the translator", agreed)
	}
	t.Logf("%d definitions agree, %d differ, %d refused by the translator", agreed, len(differ), rejected)
}

func contains(in []string, s string) bool {
	for _, v := range in {
		if v == s {
			return true
		}
	}
	return false
}

// mixedDatesKnownPairDifferences are the pairs of the area where the implicit
// form and the ToDateTime written out answer differently, each with why.
var mixedDatesKnownPairDifferences = map[string]string{
	"D27":   "a refused signature: an interval as the element of a Date",
	"D33":   "a refused signature: a Date as the container of an interval",
	"D102":  "a refused signature: an interval as the element of a Date, day of",
	"D48":   "Coalesce with null first: the translator converts nothing",
	"D126":  "as DateTime of a Date: the translator refuses it, the explicit twin converts",
	"D113":  "a tuple: the translator converts nothing inside it",
	"D76":   "a union or except of lists: the translator casts to a choice and converts nothing",
	"D77":   "a union or except of lists: the translator casts to a choice and converts nothing",
	"D80":   "a union or except of lists: the translator casts to a choice and converts nothing",
	"D1059": "a union or except of lists: the translator casts to a choice and converts nothing",
	"D1065": "a union or except of lists: the translator casts to a choice and converts nothing",
	"D1083": "a union or except of lists: the translator casts to a choice and converts nothing",
	"D1089": "a union or except of lists: the translator casts to a choice and converts nothing",
}

// TestAnImplicitConversionAnswersAsTheExplicitOne holds each definition of the
// area that has a twin with the ToDateTime written out (testdata/mixeddates/
// pairs.tsv) to the same answer: the translator writes the same ELM for the
// two, so the two references answer them alike, and this engine has to as well
// (plan 019, hallazgo 7: the invariant between the two paths).
func TestAnImplicitConversionAnswersAsTheExplicitOne(t *testing.T) {
	raw, err := os.ReadFile("testdata/mixeddates/pairs.tsv")
	if err != nil {
		t.Fatal(err)
	}
	answers := readAnswers(t, "mixeddates")
	var differ []string
	n := 0
	for _, line := range strings.Split(string(raw), "\n") {
		implicit, explicit, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		n++
		a, b := answers[implicit], answers[explicit]
		if a == b {
			continue
		}
		differ = append(differ, implicit)
		if _, known := mixedDatesKnownPairDifferences[implicit]; !known {
			t.Errorf("%s answers %s and its twin %s answers %s", implicit, a, explicit, b)
		}
	}
	for name, why := range mixedDatesKnownPairDifferences {
		if !contains(differ, name) {
			t.Errorf("%s is listed as a known difference (%s) and now agrees: remove it from the list", name, why)
		}
	}
	if n < 250 {
		t.Errorf("only %d pairs were compared", n)
	}
}
