package cql

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/gofhir/cql/internal/referenceharness/answers"
)

// unconfirmedUntypedMatches is every match of this engine's answer with a
// reference's that holds a value of no type — a Number, a Text or an Object
// of cql-execution — which the area's oracle does not confirm: the match says
// the value, and only the oracle, which does not read this engine, says the
// type. Where Java gives an error, as on Any (D223, D393, D345), nothing else
// would say it. An area with no oracle for these values has no such match.
func unconfirmedUntypedMatches(t *testing.T, area string, untyped map[string]answers.Answer) []string {
	t.Helper()
	var problems []string
	if (area == "types" || area == "fhir") && len(untyped) == 0 {
		// Both corpora hold values of type Any that cql-execution answers.
		problems = append(problems, "no match with a value of no type, so this check checks nothing")
	}
	switch area {
	case "types":
		defs := typesDefinitions(t)
		for _, key := range slices.Sorted(maps.Keys(untyped)) {
			name, _, _ := strings.Cut(key, "\t")
			v, err := typesExpected(defs[name])
			engine := pinned(untyped[key].Text)
			if err != nil || v.undecided || !expectedMatches(v.expected, engine) {
				problems = append(problems, oneLine(key)+": this engine answers "+engine+"; the oracle expects "+v.expected)
			}
		}
	case "fhir":
		defs, matched := fhirDefinitions(t), map[string]string{}
		for key := range untyped {
			name, _, _ := strings.Cut(key, "\t")
			matched[name] = defs[name]
		}
		c, engine := readFHIRCorpus(t), fhirEngineAnswers(t, matched)
		for _, key := range slices.Sorted(maps.Keys(untyped)) {
			name, _, _ := strings.Cut(key, "\t")
			v, err := c.fhirExpected(defs[name])
			if err != nil || engine[name] != v.expected {
				problems = append(problems, oneLine(key)+": this engine answers "+engine[name]+"; the oracle expects "+v.expected)
			}
		}
	default:
		for _, key := range slices.Sorted(maps.Keys(untyped)) {
			problems = append(problems, oneLine(key)+": no oracle of "+area+" confirms "+untyped[key].Text)
		}
	}
	return problems
}

// TestAnUntypedMatchWithNoOracleIsAProblem covers an area with no oracle for
// a value of no type: none of its corpus has one today, so only a match made
// up here reaches that branch.
func TestAnUntypedMatchWithNoOracleIsAProblem(t *testing.T) {
	number := answers.Reference(json.RawMessage(`{"type":"Number","value":"1"}`))
	if problems := unconfirmedUntypedMatches(t, "arithmetic", map[string]answers.Answer{"D1\tjs": number}); len(problems) != 1 {
		t.Errorf("an untyped match in arithmetic gives %v, want one problem", problems)
	}
	if problems := unconfirmedUntypedMatches(t, "arithmetic", nil); len(problems) != 0 {
		t.Errorf("no untyped match in arithmetic gives %v, want none", problems)
	}
}

// TestTheReferencesHoldNothingUnencoded holds every reference answer the
// tests read to an encoding they can read: a value the harness writes as
// "Unencoded:<its class>" is one no comparison can settle, and 34 of them
// left 16 divergences without a verdict until the harness wrote Code,
// Concept and cql-execution's values of no type.
func TestTheReferencesHoldNothingUnencoded(t *testing.T) {
	var files []string
	for _, pattern := range []string{"testdata/snapshots/*.json", "testdata/reference/*.json"} {
		m, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, m...)
	}
	unencoded := regexp.MustCompile(`"type"\s*:\s*"Unencoded:`)
	read := 0
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if !json.Valid(raw) {
			t.Errorf("%s is not JSON", f)
		}
		read++
		if n := len(unencoded.FindAllIndex(raw, -1)); n > 0 {
			t.Errorf("%s holds %d values the reference harness could not encode", f, n)
		}
	}
	if read < 10 {
		t.Fatalf("read %d reference files, fewer than there are", read)
	}
}
