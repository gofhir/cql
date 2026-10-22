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

// The membership oracle computes, from the specification, the answer of each
// definition of the membership corpus (plan 005): in, contains, includes,
// included in and during over a list or an interval, with a null on one side
// or none. It reads the text, never this engine: the generator writes every
// list with its type, List<T>{ … }, every null with the type it stands for,
// and every value in one spelling and without a null inside, so that the
// overload is what the text says and two values are equal exactly when they
// are written alike.

const membershipCorpusSize = 400

const membershipOracleRule = "membership oracle (engine_membership_oracle_test.go), CQL 1.5.3 (cql.hl7.org/09-b-cqlreference.html): " +
	"List Operators (In, Contains, Includes, Included In), Interval Operators (In, Contains, Includes, Included In); " +
	undecidedADR + ", decision 6"

var classifyMembership = flag.Bool("classify-membership", false,
	"give the unclassified divergences of the membership area the verdict of its oracle")

// membershipOperators are the operators the corpus writes, longest first so
// that `included in` is not read as `in`.
var membershipOperators = []string{"included in", "contains", "includes", "during", "in"}

// membershipVerdict is the oracle's answer and the rule that gives it.
type membershipVerdict struct {
	expected, reason string
}

// membershipExpected computes the answer of a definition of the corpus.
func membershipExpected(expr string) (membershipVerdict, error) {
	body, ok := unwrapped(expr)
	if !ok {
		return membershipVerdict{}, fmt.Errorf("not one parenthesized membership: %q", expr)
	}
	op, left, right, ok := splitMembership(body)
	if !ok {
		return membershipVerdict{}, fmt.Errorf("no membership operator at the top level: %q", expr)
	}
	element, container := left, right
	if op == "contains" || op == "includes" {
		element, container = right, left
	}
	e := readMemberElement(element)
	c, err := readMemberContainer(container)
	if err != nil {
		return membershipVerdict{}, err
	}
	answer := func(b bool) string {
		if b {
			return "true"
		}
		return "false"
	}
	if c.interval {
		switch {
		case c.null && e.null:
			return membershipVerdict{"false", "a null interval and a null point: the null container decides, false (ADR 003, decision 6)"}, nil
		case c.null:
			return membershipVerdict{"false", "a null interval holds no point: Interval Contains, first argument null, and Interval In, second argument null, are false"}, nil
		case e.null:
			return membershipVerdict{"null", "a null point in an interval: Interval Contains, second argument null, and Interval In, first argument null, are null"}, nil
		}
		return membershipVerdict{}, fmt.Errorf("a point and an interval both given is a comparison, not this corpus: %q", expr)
	}
	if e.null && e.typed && e.typ != c.typ {
		// The other operand is no element of the list: the list-list overload.
		if e.typ != "List<"+c.typ+">" || op == "in" || op == "contains" {
			return membershipVerdict{}, fmt.Errorf("a null of type %s against a List<%s>: %q", e.typ, c.typ, expr)
		}
		return membershipVerdict{"null", "a null list in the list-list overload of includes or included in is null"}, nil
	}
	switch {
	case c.null:
		return membershipVerdict{"false", "a null list holds nothing: List In, second argument null, and List Contains, first argument null, are false"}, nil
	case e.null:
		return membershipVerdict{answer(slices.ContainsFunc(c.items, isNullItem)),
			"a null element is in a list if the list holds a null element, and is not otherwise: List In, first argument null, and List Contains, second argument null"}, nil
	}
	return membershipVerdict{answer(slices.Contains(c.items, e.text)),
		"a value is in a list if an element equals it, and values written in one spelling are equal when they are written alike: List In, List Contains"}, nil
}

// memberElement is the side of a membership that is looked for.
type memberElement struct {
	text  string
	null  bool
	typed bool
	typ   string
}

var (
	memberTypedNull = regexp.MustCompile(`^\(null as (.+)\)$`)
	memberFirstNull = regexp.MustCompile(`^First\(List<(.+)>\{\}\)$`)
)

func readMemberElement(s string) memberElement {
	if s == "null" {
		return memberElement{text: s, null: true}
	}
	if m := memberTypedNull.FindStringSubmatch(s); m != nil {
		return memberElement{text: s, null: true, typed: true, typ: m[1]}
	}
	if m := memberFirstNull.FindStringSubmatch(s); m != nil {
		return memberElement{text: s, null: true, typed: true, typ: m[1]}
	}
	return memberElement{text: s}
}

// memberContainer is the list or interval of a membership; items are a list's
// elements at its top level, each as written.
type memberContainer struct {
	interval, null bool
	typ            string
	items          []string
}

var (
	memberNullList     = regexp.MustCompile(`^\(null as List<(.+)>\)$`)
	memberNullInterval = regexp.MustCompile(`^\(null as Interval<(.+)>\)$`)
	memberList         = regexp.MustCompile(`^List<(.+?)>\{(.*)\}$`)
)

func readMemberContainer(s string) (memberContainer, error) {
	if m := memberNullList.FindStringSubmatch(s); m != nil {
		return memberContainer{null: true, typ: m[1]}, nil
	}
	if memberNullInterval.MatchString(s) {
		return memberContainer{interval: true, null: true}, nil
	}
	if strings.HasPrefix(s, "Interval[") {
		return memberContainer{interval: true}, nil
	}
	m := memberList.FindStringSubmatch(s)
	if len(m) < 3 || !balanced(m[1]) {
		return memberContainer{}, fmt.Errorf("no list or interval: %q", s)
	}
	c := memberContainer{typ: m[1]}
	if inner := strings.TrimSpace(m[2]); inner != "" {
		c.items = splitTopLevel(inner, ',')
	}
	return c, nil
}

// isNullItem reports whether an element of a list is null itself — not a value
// that holds a null, as Interval[1, null] or Tuple { a: null } do.
func isNullItem(item string) bool {
	return item == "null" || memberTypedNull.MatchString(item)
}

// unwrapped is s without the parentheses that enclose all of it.
func unwrapped(s string) (string, bool) {
	if !strings.HasPrefix(s, "(") || !strings.HasSuffix(s, ")") {
		return "", false
	}
	inner := s[1 : len(s)-1]
	if !balanced(inner) {
		return "", false
	}
	return inner, true
}

// splitMembership finds the membership operator at the top level of s.
func splitMembership(s string) (op, left, right string, ok bool) {
	depth, quoted := 0, false
	for i := 0; i < len(s); i++ {
		switch ch := s[i]; {
		case ch == '\'':
			quoted = !quoted
		case quoted:
		case ch == '(' || ch == '{' || ch == '[':
			depth++
		case ch == ')' || ch == '}' || ch == ']':
			depth--
		case ch == ' ' && depth == 0:
			for _, candidate := range membershipOperators {
				if strings.HasPrefix(s[i:], " "+candidate+" ") {
					return candidate, s[:i], s[i+len(candidate)+2:], true
				}
			}
		}
	}
	return "", "", "", false
}

// splitTopLevel splits s at sep outside quotes and brackets, trimming each part.
func splitTopLevel(s string, sep byte) []string {
	var parts []string
	depth, quoted, start := 0, false, 0
	for i := 0; i < len(s); i++ {
		switch ch := s[i]; {
		case ch == '\'':
			quoted = !quoted
		case quoted:
		case ch == '(' || ch == '{' || ch == '[':
			depth++
		case ch == ')' || ch == '}' || ch == ']':
			depth--
		case ch == sep && depth == 0:
			parts = append(parts, strings.TrimSpace(s[start:i]))
			start = i + 1
		}
	}
	return append(parts, strings.TrimSpace(s[start:]))
}

// balanced reports whether every bracket of s outside quotes closes, in order.
// An interval written Interval[1, 5) closes a bracket with a parenthesis, and
// counts as balanced.
func balanced(s string) bool {
	depth, quoted := 0, false
	for i := 0; i < len(s); i++ {
		switch ch := s[i]; {
		case ch == '\'':
			quoted = !quoted
		case quoted:
		case ch == '(' || ch == '{' || ch == '[':
			depth++
		case ch == ')' || ch == '}' || ch == ']':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0 && !quoted
}

// membershipCorpus reads the definitions of the membership corpus.
func membershipCorpus(t *testing.T) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", "membership.cql"))
	if err != nil {
		t.Fatal(err)
	}
	defs := map[string]string{}
	for _, m := range regexp.MustCompile(`(?m)^define (D\d+): (.*)$`).FindAllStringSubmatch(string(raw), -1) {
		defs[m[1]] = m[2]
	}
	return defs
}

// classifiedByMembershipOracle is the verdict the oracle gives a divergence:
// the specification, and ADR 003's decision 6, decide every definition of the
// area, so a divergence is a defect of whichever answer is not the oracle's.
func classifiedByMembershipOracle(t *testing.T, defs map[string]string, key string, d divergence) divergence {
	t.Helper()
	name, _, _ := strings.Cut(key, "\t")
	v, err := membershipExpected(defs[name])
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	d.Expected = v.expected
	d.Class = ownDefect
	if expectedMatches(d.Expected, d.Engine) {
		d.Class = referenceDefect
	}
	d.Evidence = "membership oracle: " + v.reason + "; " + strings.TrimPrefix(membershipOracleRule, "membership oracle ")
	return d
}

// TestTheMembershipVerdictsFollowTheOracle holds every verdict of the
// membership area to what the oracle computes for its definition.
func TestTheMembershipVerdictsFollowTheOracle(t *testing.T) {
	defs := membershipCorpus(t)
	path := filepath.Join("testdata", "snapshots", "membership.divergences.tsv")
	recorded := readDivergences(t, path)
	if *classifyMembership {
		for key, d := range recorded {
			recorded[key] = classifiedByMembershipOracle(t, defs, key, d)
		}
		writeVerdicts(t, path, recorded)
		return
	}
	for _, key := range slices.Sorted(maps.Keys(recorded)) {
		d := recorded[key]
		want := classifiedByMembershipOracle(t, defs, key, d)
		if want.Class != d.Class || want.Expected != d.Expected || want.Evidence != d.Evidence {
			t.Errorf("%s: recorded %s expecting %q (%s); the oracle gives %s expecting %q (%s)",
				oneLine(key), d.Class, d.Expected, oneLine(d.Evidence), want.Class, want.Expected, oneLine(want.Evidence))
		}
	}
}

// TestTheMembershipOracleAgreesWhereAllAgree holds the oracle to the
// definitions where this engine and both references give one answer.
func TestTheMembershipOracleAgreesWhereAllAgree(t *testing.T) {
	defs := membershipCorpus(t)
	recorded := readDivergences(t, filepath.Join("testdata", "snapshots", "membership.divergences.tsv"))
	java := readReferenceResults(t, "membership", "java")
	agreed := 0
	for _, name := range slices.Sorted(maps.Keys(defs)) {
		_, javaDiffers := recorded[name+"\tjava"]
		_, jsDiffers := recorded[name+"\tjs"]
		raw, answered := java[name]
		if javaDiffers || jsDiffers || !answered {
			continue
		}
		agreed++
		v, err := membershipExpected(defs[name])
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if answer := pinned(answers.Reference(raw).Text); !expectedMatches(v.expected, answer) {
			t.Errorf("%s: all three answer %q; the oracle expects %q (%s)", name, oneLine(answer), v.expected, v.reason)
		}
	}
	if agreed == 0 {
		t.Fatal("no definition where all three agree: the oracle is checked on nothing")
	}
}

// TestTheMembershipOracleReadsTheWholeCorpus holds the oracle to every
// definition of the corpus, divergent or not: one it cannot read is a gap.
func TestTheMembershipOracleReadsTheWholeCorpus(t *testing.T) {
	defs := membershipCorpus(t)
	if len(defs) != membershipCorpusSize {
		t.Fatalf("the oracle reads %d definitions of membership.cql; the corpus has %d", len(defs), membershipCorpusSize)
	}
	for name, expr := range defs {
		if _, err := membershipExpected(expr); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestTheMembershipOracleReadsANestedNullAsAValue holds the oracle to the
// difference between a null element and a value that holds a null, which a
// reading of the text for "null" would confuse: { Interval[1, null] } holds no
// null element (List Contains; both references answer false).
func TestTheMembershipOracleReadsANestedNullAsAValue(t *testing.T) {
	for expr, want := range map[string]string{
		"(List<Interval<Integer>>{ Interval[1, null] } contains (null as Interval<Integer>))":     "false",
		"(List<Tuple { a Integer }>{ Tuple { a: null } } contains (null as Tuple { a Integer }))": "false",
		"(List<Integer>{ 1, (null as Integer) } contains (null as Integer))":                      "true",
		"(List<String>{ 'a, null' } contains null)":                                               "false",
		"((null as List<String>) included in List<String>{ 's' })":                                "null",
		"((null as List<Integer>) included in List<List<Integer>>{ { 1 } })":                      "false",
	} {
		v, err := membershipExpected(expr)
		if err != nil {
			t.Fatal(err)
		}
		if v.expected != want {
			t.Errorf("%s: the oracle expects %s (%s); want %s", expr, v.expected, v.reason, want)
		}
	}
}
