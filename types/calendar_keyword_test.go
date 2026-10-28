package types

import (
	"os"
	"regexp"
	"testing"
)

// TestTheCalendarKeywordsAreTheGrammars holds IsCalendarKeyword to the rules
// dateTimePrecision and pluralDateTimePrecision of grammar/cql.g4, the
// keywords "singular or plural" a calendar duration is written with (CQL
// Reference, ToQuantity; plan 010): every keyword of the two rules is one, and
// nothing else in the set is.
func TestTheCalendarKeywordsAreTheGrammars(t *testing.T) {
	grammar, err := os.ReadFile("../grammar/cql.g4")
	if err != nil {
		t.Fatal(err)
	}
	keywords := map[string]bool{}
	for _, rule := range []string{"dateTimePrecision", "pluralDateTimePrecision"} {
		m := regexp.MustCompile(`(?m)^` + rule + `\s*:([^;]*);`).FindSubmatch(grammar)
		if m == nil {
			t.Fatalf("no rule %s in grammar/cql.g4", rule)
		}
		for _, k := range regexp.MustCompile(`'([a-z]+)'`).FindAllSubmatch(m[1], -1) {
			keywords[string(k[1])] = true
		}
	}
	if len(keywords) != 16 {
		t.Fatalf("the grammar gives %d keywords, want 16", len(keywords))
	}
	for k := range keywords {
		if !IsCalendarKeyword(k) {
			t.Errorf("IsCalendarKeyword(%q) = false, and the grammar has it", k)
		}
	}
	for k := range yearsAndMonths {
		if !keywords[k] {
			t.Errorf("%q is a calendar keyword here and not in the grammar", k)
		}
	}
	for k := range belowTheMonth {
		if !keywords[k] {
			t.Errorf("%q is a calendar keyword here and not in the grammar", k)
		}
	}
	for _, k := range []string{"Day", "DAYS", "d", "wk", "mo", "a", ""} {
		if IsCalendarKeyword(k) {
			t.Errorf("IsCalendarKeyword(%q) = true", k)
		}
	}
}
