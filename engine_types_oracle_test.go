package cql

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/gofhir/cql/internal/referenceharness/answers"
)

// The types area's oracle (docs/plans/003-clasificar-divergencias.md, PR 2):
// what the specification makes of each definition of testdata/snapshots/types.cql,
// computed from the run-time type of its literal and the rule that decides
// it, not from any engine's answer. Its corpus is a closed grammar — is, as,
// `as … is null` and cast of an element of a List<Any> against a type name —
// so the oracle reads every definition of it, and refuses one it cannot.
//
// The rules, CQL 1.5.3 (cql.hl7.org/N1A):
//   - Appendix B, Type Operators. Is: true "if the run-time type of the
//     argument is the same as or derived from the type being tested". As: "if
//     the argument is not of the specified type at run-time the result is
//     null". cast: "if the argument is not of the specified type at run-time
//     then an exception is thrown".
//   - Developer's Guide, Types. "All the system-defined types derive directly
//     from System.Any" — Code and Concept are not Vocabulary, Long is not
//     Integer, Integer is not Decimal; "list type List<E'> is a subtype of list
//     type List<E>" when E' is of E, and an interval type likewise; choice
//     types are compatible when the choices of one are a subset of the other's.
//   - Neither says what null's run-time type is: `null is T` and
//     `cast null as T` are undecided, with the two readings the references give.
const typesOracleRule = "types oracle (engine_types_oracle_test.go), CQL 1.5.3 (cql.hl7.org/N1A): " +
	"Appendix B, Type Operators (Is, As, cast) and Developer's Guide, Types, Type Testing"

// typesCorpusSize is the number of definitions of testdata/snapshots/types.cql.
const typesCorpusSize = 900

var classifyTypes = flag.Bool("classify-types", false,
	"give the unclassified divergences of the types area the verdict of its oracle")

// cqlType is a type of the oracle: a named type, or List or Interval of one,
// or a choice of several.
type cqlType struct {
	name    string // a System type's name, or "List", "Interval", "Choice"
	element []cqlType
}

func named(name string) cqlType { return cqlType{name: name} }

func (t cqlType) String() string {
	if len(t.element) == 0 {
		return t.name
	}
	parts := make([]string, len(t.element))
	for i, e := range t.element {
		parts[i] = e.String()
	}
	return t.name + "<" + strings.Join(parts, ", ") + ">"
}

// typedLiteral is a literal of the corpus with its run-time type, and its
// answer as the comparison writes it when an `as` or a cast hands it back.
type typedLiteral struct {
	runtime *cqlType // nil for null
	answer  string   // the comparison's text; a prefix where the value is a structure it cannot read
	prefix  bool
}

// typesLiterals is every literal the generator draws (internal/referenceharness/differential/gen,
// typedValues), with the run-time type its selector gives it (Developer's
// Guide, Type Inference: an interval's type is its low bound's; a list's, its
// elements' or its specifier's).
var typesLiterals = map[string]typedLiteral{
	"5":                         {runtime: ptr(named("Integer")), answer: "Integer 5"},
	"5L":                        {runtime: ptr(named("Long")), answer: "Long 5"},
	"2.5":                       {runtime: ptr(named("Decimal")), answer: "Decimal 2.5"},
	"'a'":                       {runtime: ptr(named("String")), answer: `"a"`},
	"true":                      {runtime: ptr(named("Boolean")), answer: "true"},
	"@2020-01-01":               {runtime: ptr(named("Date")), answer: "Date 2020-01-01"},
	"@2020-01-01T10:00:00.000Z": {runtime: ptr(named("DateTime")), answer: "DateTime 2020-01-01T10:00:00.000Z"},
	"@T10:00:00":                {runtime: ptr(named("Time")), answer: "Time 10:00:00"},
	"5 'mg'":                    {runtime: ptr(named("Quantity")), answer: "5 'mg'"},
	"{ 1, 2 }":                  {runtime: ptr(cqlType{"List", []cqlType{named("Integer")}}), answer: "{1, 2}"},
	"Interval[1, 2]":            {runtime: ptr(cqlType{"Interval", []cqlType{named("Integer")}}), answer: "Interval[1, 2]"},
	"Tuple{ a: 1 }":             {runtime: ptr(named("Tuple")), answer: "Tuple", prefix: true},
	"Code{ code: 'x' }":         {runtime: ptr(named("Code")), answer: "Code ", prefix: true},
	"Concept{ codes: { Code{ code: 'x' } } }": {runtime: ptr(named("Concept")), answer: "Concept ", prefix: true},
	"(null as Integer)":                       {},
	"{ 1, 'a' }": {runtime: ptr(cqlType{"List", []cqlType{{"Choice", []cqlType{named("Integer"), named("String")}}}}),
		answer: `{1, "a"}`},
	"{ 'a' }":            {runtime: ptr(cqlType{"List", []cqlType{named("String")}}), answer: `{"a"}`},
	"List<Integer>{}":    {runtime: ptr(cqlType{"List", []cqlType{named("Integer")}}), answer: "{}"},
	"Interval[1, null]":  {runtime: ptr(cqlType{"Interval", []cqlType{named("Integer")}}), answer: "Interval[1, null]"},
	"Interval['a', 'b']": {runtime: ptr(cqlType{"Interval", []cqlType{named("String")}}), answer: `Interval["a", "b"]`},
}

func ptr(t cqlType) *cqlType { return &t }

var typeName = regexp.MustCompile(`^(?:System\.)?(\w+)(?:<(?:System\.)?(\w+)>)?$`)

// parseTypeName reads a type name of the corpus.
func parseTypeName(s string) (cqlType, bool) {
	m := typeName.FindStringSubmatch(s)
	if m == nil {
		return cqlType{}, false
	}
	if m[2] != "" {
		return cqlType{m[1], []cqlType{named(m[2])}}, true
	}
	return named(m[1]), true
}

// derivesFrom reports whether a value of run-time type a is of type b.
func derivesFrom(a, b cqlType) bool {
	switch {
	case b.name == "Any" && len(b.element) == 0:
		return true
	case a.name == "Choice":
		for _, c := range a.element {
			if !derivesFrom(c, b) {
				return false
			}
		}
		return true
	case len(a.element) > 0 || len(b.element) > 0:
		return a.name == b.name && len(a.element) == 1 && len(b.element) == 1 && derivesFrom(a.element[0], b.element[0])
	}
	return a.name == b.name
}

var (
	typesIs     = regexp.MustCompile(`^\(\(List<Any>\{ (.*) \}\)\[0\] (is|as) (\S+)\)$`)
	typesAsNull = regexp.MustCompile(`^\({3}List<Any>\{ (.*) \}\)\[0\] as (\S+)\) is null\)$`)
	typesCast   = regexp.MustCompile(`^\(cast \(List<Any>\{ (.*) \}\)\[0\] as (\S+)\)$`)
)

// typesVerdict is the oracle's answer to a definition: the answer the
// specification expects, how it was reached, or why it cannot say.
type typesVerdict struct {
	expected, reason string
	undecided        bool
	prefix           bool // expected is a prefix of the answer, for a structure the comparison cannot read
}

func typesExpected(expr string) (typesVerdict, error) {
	var op, lit, target string
	if m := typesAsNull.FindStringSubmatch(expr); m != nil {
		op, lit, target = "as-null", m[1], m[2]
	} else if m := typesCast.FindStringSubmatch(expr); m != nil {
		op, lit, target = "cast", m[1], m[2]
	} else if m := typesIs.FindStringSubmatch(expr); m != nil {
		op, lit, target = m[2], m[1], m[3]
	} else {
		return typesVerdict{}, fmt.Errorf("not of the corpus's grammar: %s", expr)
	}
	value, ok := typesLiterals[lit]
	if !ok {
		return typesVerdict{}, fmt.Errorf("a literal the oracle does not know: %s", lit)
	}
	t, ok := parseTypeName(target)
	if !ok {
		return typesVerdict{}, fmt.Errorf("a type the oracle does not know: %s", target)
	}
	if value.runtime == nil {
		switch op {
		case "as":
			return typesVerdict{expected: "null", reason: "null as " + t.String() + " is null, whatever null's run-time type is"}, nil
		case "as-null":
			return typesVerdict{expected: "true", reason: "null as " + t.String() + " is null"}, nil
		}
		return typesVerdict{undecided: true, reason: "the specification does not say what null's run-time type is"}, nil
	}
	// An empty list has no element to give it a type at run time. Its
	// specifier gives it List<Integer> statically, and the specification does
	// not say whether that is its run-time type: all three engines take an
	// empty list for a list of any element (D104, D823 agree on it).
	if lit == "List<Integer>{}" && t.name == "List" && !derivesFrom(named("Integer"), t.element[0]) {
		return typesVerdict{undecided: true, reason: "the specification does not say whether an empty list's specifier is its run-time type"}, nil
	}
	of := derivesFrom(*value.runtime, t)
	relation := " is not of "
	if of {
		relation = " is of "
	}
	reason := "a " + value.runtime.String() + relation + t.String()
	switch op {
	case "is":
		return typesVerdict{expected: fmt.Sprint(of), reason: reason}, nil
	case "as-null":
		return typesVerdict{expected: fmt.Sprint(!of), reason: reason}, nil
	case "as":
		if !of {
			return typesVerdict{expected: "null", reason: reason}, nil
		}
		return typesVerdict{expected: value.answer, prefix: value.prefix, reason: reason}, nil
	}
	if !of {
		return typesVerdict{expected: "error", reason: reason}, nil
	}
	return typesVerdict{expected: value.answer, prefix: value.prefix, reason: reason}, nil
}

// expectedMatches reports whether a pinned answer is the expected one, read as
// the comparison reads answers: every error is one answer, and a structure the
// comparison cannot read is matched by its kind.
func expectedMatches(expected, pin string) bool {
	switch {
	case expected == "error":
		return strings.HasPrefix(pin, "error")
	case strings.HasSuffix(expected, "…"):
		return strings.HasPrefix(pin, strings.TrimSuffix(expected, "…"))
	}
	return expected == pin
}

// storedExpected is a verdict's expected answer as a manifest writes it: a
// prefix ends in "…".
func (v typesVerdict) storedExpected() string {
	if v.prefix {
		return v.expected + "…"
	}
	return v.expected
}

// TestTheTypesVerdictsFollowTheOracle holds every verdict of the types area
// to what the oracle computes for its definition: its
// expected answer, the class that follows from it — this engine's answer is
// the expected one, or it is not — and the evidence that says so. Which rows
// it holds does not depend on their class or on what their evidence cites:
// the area has no deliberate difference, and one would change this test.
func TestTheTypesVerdictsFollowTheOracle(t *testing.T) {
	defs := typesDefinitions(t)
	path := filepath.Join("testdata", "snapshots", "types.divergences.tsv")
	recorded := readDivergences(t, path)
	if *classifyTypes {
		for key, d := range recorded {
			recorded[key] = classifiedByTypesOracle(t, defs, key, d)
		}
		writeVerdicts(t, path, recorded)
		return
	}
	for _, key := range slices.Sorted(maps.Keys(recorded)) {
		d := recorded[key]
		want := classifiedByTypesOracle(t, defs, key, d)
		if want.Class != d.Class || want.Expected != d.Expected || want.Evidence != d.Evidence {
			t.Errorf("%s: recorded %s expecting %q (%s); the oracle gives %s expecting %q (%s)",
				oneLine(key), d.Class, d.Expected, oneLine(d.Evidence), want.Class, want.Expected, oneLine(want.Evidence))
		}
	}
}

// TestTheTypesOracleAgreesWhereAllAgree holds the oracle to the definitions
// where this engine and every reference that answers give one answer (the JS
// one runs only what the translator accepted): there, the oracle
// gives that answer or leaves it undecided. It is what checks the oracle on
// the type relations no divergence reaches.
func TestTheTypesOracleAgreesWhereAllAgree(t *testing.T) {
	defs := typesDefinitions(t)
	recorded := readDivergences(t, filepath.Join("testdata", "snapshots", "types.divergences.tsv"))
	java, js := readReferenceResults(t, "types", "java"), readReferenceResults(t, "types", "js")
	agreed := 0
	for _, name := range slices.Sorted(maps.Keys(defs)) {
		_, javaDiffers := recorded[name+"\tjava"]
		_, jsDiffers := recorded[name+"\tjs"]
		raw, answered := java[name]
		if !answered {
			raw, answered = js[name]
		}
		if javaDiffers || jsDiffers || !answered {
			continue
		}
		agreed++
		v, err := typesExpected(defs[name])
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if answer := pinned(answers.Reference(raw).Text); !v.undecided && !expectedMatches(v.storedExpected(), answer) {
			t.Errorf("%s: all three answer %q; the oracle expects %q (%s)", name, oneLine(answer), v.storedExpected(), v.reason)
		}
	}
	if agreed == 0 {
		t.Fatal("no definition where all three agree: the oracle is checked on nothing")
	}
}

// unencoded finds a value the reference harness could not encode, which it
// writes as an object of type "Unencoded:<the engine's own type>".
var unencoded = regexp.MustCompile(`"type"\s*:\s*"Unencoded:`)

// classifiedByTypesOracle is the verdict the oracle gives a divergence.
func classifiedByTypesOracle(t *testing.T, defs map[string]string, key string, d divergence) divergence {
	t.Helper()
	name, reference, _ := strings.Cut(key, "\t")
	v, err := typesExpected(defs[name])
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if v.undecided {
		d.Class, d.Expected = undecided, ""
		d.Evidence = fmt.Sprintf("reading 1: %s, as the reference answers [source: testdata/snapshots/types.%s.json] | "+
			"reading 2: %s, as this engine answers [source: testdata/snapshots/types.answers.tsv]; %s; %s",
			d.Reference, reference, d.Engine, v.reason, typesOracleRule)
		return d
	}
	d.Expected = v.storedExpected()
	d.Class = ownDefect
	if expectedMatches(d.Expected, d.Engine) {
		d.Class = referenceDefect
		if v.expected != "error" && v.expected != "null" && unencoded.MatchString(d.Reference) {
			// The harness could not write the reference's value down, so
			// nothing says the reference is wrong: a value is expected and
			// it gave one. No verdict until the harness reads it.
			d.Class, d.Expected = unclassified, ""
			d.Evidence = "the reference harness cannot encode the reference's answer; types oracle expects " + v.storedExpected()
			return d
		}
	}
	d.Evidence = "types oracle: " + v.reason + "; " + strings.TrimPrefix(typesOracleRule, "types oracle ")
	return d
}

func typesDefinitions(t *testing.T) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", "types.cql"))
	if err != nil {
		t.Fatal(err)
	}
	defs := map[string]string{}
	for _, m := range regexp.MustCompile(`(?m)^define (D\d+): (.*)$`).FindAllStringSubmatch(string(raw), -1) {
		defs[m[1]] = m[2]
	}
	return defs
}

// writeVerdicts writes a manifest's recorded verdicts back as they are.
func writeVerdicts(t *testing.T, path string, recorded map[string]divergence) {
	t.Helper()
	var b strings.Builder
	b.WriteString(divergencesHeader)
	for _, key := range slices.Sorted(maps.Keys(recorded)) {
		d := recorded[key]
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\t%s\n", key, d.Class, d.Expected, d.Evidence, d.Reference, d.Engine)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestTheTypesOracleReadsTheWholeCorpus holds the oracle to every definition of
// the corpus, divergent or not: one it cannot read is a gap in it.
func TestTheTypesOracleReadsTheWholeCorpus(t *testing.T) {
	defs := typesDefinitions(t)
	if len(defs) != typesCorpusSize {
		t.Fatalf("the oracle reads %d definitions of types.cql; the corpus has %d", len(defs), typesCorpusSize)
	}
	for name, expr := range defs {
		if _, err := typesExpected(expr); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}
