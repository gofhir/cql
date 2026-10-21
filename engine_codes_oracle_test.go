package cql

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/gofhir/cql/internal/referenceharness/answers"
)

// The codes oracle computes, from the specification and not from this
// engine, what each definition of testdata/snapshots/codes.cql answers. It
// reads the corpus's own text: the preamble's declarations and the
// selectors, lists and operators gen writes. It reads, from CQL 1.5.3
// (cql.hl7.org/09-b-cqlreference.html):
//   - Clinical Values: `structured type Code { code String, display String,
//     system String, version String }` and `structured type Concept { codes
//     List<Code>, display String }`. An element a selector leaves out is null;
//     the empty String is a String. A declared code takes its code system's
//     version.
//   - Comparison Operators, Equal: a structured type compares as "a
//     conjunction of equality comparisons" of its elements; an element null on
//     both sides is equal, null on one side unknown; a list compares element by
//     element, in order, "with the exception that null elements are considered
//     equal"; a null operand gives null. Not Equal is its negation. A list with
//     one pair unknown and another unequal the specification does not decide,
//     and the corpus holds none (plan 004, review round 1).
//   - Comparison Operators, Equivalent: a Code by "the code and system
//     elements only", as Strings, ignoring case; a Concept by "a non-empty
//     intersection of the codes"; never null. Not Equivalent is its negation.
//
// The order of the conjunction is the user's decision of plan 004: an element
// that differs gives false whatever the others are, where the Java engine
// answers null when a null comes first.
const codesOracleRule = "codes oracle (engine_codes_oracle_test.go), CQL 1.5.3 (cql.hl7.org/09-b-cqlreference.html): " +
	"Clinical Values (Code, Concept), Comparison Operators (Equal, Equivalent)"

// codesCorpusSize is the number of definitions of testdata/snapshots/codes.cql.
const codesCorpusSize = 600

var classifyCodes = flag.Bool("classify-codes", false,
	"give the unclassified divergences of the codes area the verdict of its oracle")

// tri is a three-valued Boolean.
type tri int

const (
	triNull tri = iota
	triFalse
	triTrue
)

func triOf(b bool) tri {
	if b {
		return triTrue
	}
	return triFalse
}

func (v tri) not() tri {
	switch v {
	case triTrue:
		return triFalse
	case triFalse:
		return triTrue
	}
	return triNull
}

// conjunctionOf is the CQL and of its operands: false if any is false,
// unknown if any is, true otherwise.
func conjunctionOf(vs ...tri) tri {
	out := triTrue
	for _, v := range vs {
		switch v {
		case triFalse:
			return triFalse
		case triNull:
			out = triNull
		}
	}
	return out
}

// codesValue is a value of the oracle: a Code, a Concept, a list of them, a
// String, a Boolean, or null.
type codesValue struct {
	kind   string     // "code", "concept", "list", "string", "boolean", "null"
	fields [4]*string // a Code's code, system, display and version
	codes  []codesValue
	text   *string // a Concept's display or a String
	truth  tri
}

var codesFieldIndex = map[string]int{"code": 0, "system": 1, "display": 2, "version": 3}

var codesNull = codesValue{kind: "null"}

// answer is a value as answers writes it.
func (v codesValue) answer() string {
	switch v.kind {
	case "string":
		return strconv.Quote(*v.text)
	case "boolean":
		switch v.truth {
		case triTrue:
			return "true"
		case triFalse:
			return "false"
		}
	}
	return "null"
}

// textEqual is Equal of two elements of a structured value.
func textEqual(a, b *string) tri {
	switch {
	case a == nil && b == nil:
		return triTrue
	case a == nil || b == nil:
		return triNull
	}
	return triOf(*a == *b)
}

// textEquivalent is Equivalent of two Strings: ignoring case, and true for
// two nulls.
func textEquivalent(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return strings.EqualFold(*a, *b)
}

func codesEqual(a, b codesValue) tri {
	if a.kind == "null" || b.kind == "null" {
		return triNull
	}
	switch a.kind {
	case "code":
		return conjunctionOf(textEqual(a.fields[0], b.fields[0]), textEqual(a.fields[1], b.fields[1]),
			textEqual(a.fields[2], b.fields[2]), textEqual(a.fields[3], b.fields[3]))
	case "concept":
		return conjunctionOf(listEqual(a.codes, b.codes), textEqual(a.text, b.text))
	}
	return listEqual(a.codes, b.codes)
}

func listEqual(a, b []codesValue) tri {
	if len(a) != len(b) {
		return triFalse
	}
	each := make([]tri, len(a))
	for i := range a {
		// "with the exception that null elements are considered equal".
		if a[i].kind == "null" && b[i].kind == "null" {
			each[i] = triTrue
			continue
		}
		each[i] = codesEqual(a[i], b[i])
	}
	if slices.Contains(each, triNull) && slices.Contains(each, triFalse) {
		panic("a list with one pair unknown and another unequal, which the specification does not decide (plan 004, backlog 1c2)")
	}
	return conjunctionOf(each...)
}

func codesEquivalent(a, b codesValue) bool {
	if a.kind == "null" || b.kind == "null" {
		return a.kind == b.kind
	}
	if a.kind == "code" {
		return textEquivalent(a.fields[0], b.fields[0]) && textEquivalent(a.fields[1], b.fields[1])
	}
	for _, x := range a.codes {
		for _, y := range b.codes {
			if codesEquivalent(x, y) {
				return true
			}
		}
	}
	return false
}

// codesReader reads one definition of the corpus.
type codesReader struct {
	toks     []string
	at       int
	declared map[string]codesValue
	reason   string
}

var codesToken = regexp.MustCompile(`'[^']*'|"[^"]*"|!=|!~|[A-Za-z]+|[{}(),:.=~]`)

func (r *codesReader) peek() string {
	if r.at < len(r.toks) {
		return r.toks[r.at]
	}
	return ""
}

func (r *codesReader) next() string {
	tok := r.peek()
	r.at++
	return tok
}

func (r *codesReader) expect(tok string) {
	if got := r.next(); got != tok {
		panic(fmt.Sprintf("expected %q, read %q", tok, got))
	}
}

// expr is an operand, read alone or compared with another.
func (r *codesReader) expr() codesValue {
	a := r.postfix()
	op := r.peek()
	if op != "=" && op != "!=" && op != "~" && op != "!~" {
		return a
	}
	r.next()
	b := r.postfix()
	switch op {
	case "=", "!=":
		v := codesEqual(a, b)
		r.reason = fmt.Sprintf("Equal of two %s values is the conjunction of their elements' equality, %s",
			a.kind, codesValue{kind: "boolean", truth: v}.answer())
		if op == "!=" {
			v = v.not()
			r.reason += ", which Not Equal negates"
		}
		return codesValue{kind: "boolean", truth: v}
	}
	v := codesEquivalent(a, b)
	r.reason = fmt.Sprintf("Equivalent of two %s values compares the codes by code and system only, ignoring case, %t", a.kind, v)
	if op == "!~" {
		v = !v
		r.reason += ", which Not Equivalent negates"
	}
	return codesValue{kind: "boolean", truth: triOf(v)}
}

func (r *codesReader) postfix() codesValue {
	v := r.primary()
	for r.peek() == "." {
		r.next()
		field := r.next()
		switch {
		case v.kind == "null":
		case v.kind == "code":
			i, ok := codesFieldIndex[field]
			if !ok {
				panic("no Code element " + field)
			}
			v = codesText(v.fields[i])
		case v.kind == "concept" && field == "display":
			v = codesText(v.text)
		default:
			panic(fmt.Sprintf("no %s element %s", v.kind, field))
		}
		r.reason = "an element reads as what the selector or the declaration sets, null when neither does"
	}
	if r.peek() == "is" {
		r.next()
		r.expect("null")
		r.reason = "the empty String is a String, not null"
		return codesValue{kind: "boolean", truth: triOf(v.kind == "null")}
	}
	return v
}

func codesText(s *string) codesValue {
	if s == nil {
		return codesNull
	}
	return codesValue{kind: "string", text: s}
}

func (r *codesReader) primary() codesValue {
	switch tok := r.next(); {
	case tok == "(":
		if r.peek() == "null" {
			r.next()
			r.expect("as")
			r.next()
			r.expect(")")
			return codesNull
		}
		v := r.expr()
		r.expect(")")
		return v
	case tok == "{":
		return codesValue{kind: "list", codes: r.list()}
	case strings.HasPrefix(tok, `"`):
		v, ok := r.declared[strings.Trim(tok, `"`)]
		if !ok {
			panic("no declaration " + tok)
		}
		return v
	case tok == "Code" && strings.HasPrefix(r.peek(), "'"):
		// Code '1' from "CS" display 'd': the code system gives the system
		// and the version.
		code := strings.Trim(r.next(), "'")
		r.expect("from")
		cs, ok := r.declared["codesystem "+strings.Trim(r.next(), `"`)]
		if !ok {
			panic("no code system declared")
		}
		v := codesValue{kind: "code", fields: [4]*string{&code, cs.fields[1], nil, cs.fields[3]}}
		if r.peek() == "display" {
			r.next()
			d := strings.Trim(r.next(), "'")
			v.fields[2] = &d
		}
		return v
	case tok == "Code":
		v := codesValue{kind: "code"}
		for _, f := range r.selector() {
			i, ok := codesFieldIndex[f.name]
			if !ok {
				panic("no Code element " + f.name)
			}
			v.fields[i] = f.value
		}
		return v
	case tok == "Concept":
		v := codesValue{kind: "concept"}
		for _, f := range r.selector() {
			switch f.name {
			case "codes":
				v.codes = f.list
			case "display":
				v.text = f.value
			default:
				panic("no Concept element " + f.name)
			}
		}
		return v
	default:
		panic("cannot read " + tok)
	}
}

// list reads the elements of a list whose "{" is read.
func (r *codesReader) list() []codesValue {
	var items []codesValue
	for r.peek() != "}" {
		items = append(items, r.postfix())
		if r.peek() == "," {
			r.next()
		}
	}
	r.next()
	return items
}

type codesElement struct {
	name  string
	value *string
	list  []codesValue
}

// selector reads the elements of a selector, from its "{".
func (r *codesReader) selector() []codesElement {
	r.expect("{")
	var out []codesElement
	for r.peek() != "}" {
		e := codesElement{name: r.next()}
		r.expect(":")
		switch tok := r.next(); {
		case tok == "null":
		case tok == "{":
			e.list = r.list()
		case strings.HasPrefix(tok, "'"):
			s := strings.Trim(tok, "'")
			e.value = &s
		default:
			panic("cannot read the value " + tok)
		}
		out = append(out, e)
		if r.peek() == "," {
			r.next()
		}
	}
	r.next()
	return out
}

// codesVerdict is what the oracle expects of a definition, and why.
type codesVerdict struct{ expected, reason string }

func codesExpected(declared map[string]codesValue, expr string) (v codesVerdict, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("the oracle cannot read %q: %v", expr, p)
		}
	}()
	r := &codesReader{toks: codesToken.FindAllString(expr, -1), declared: declared}
	value := r.expr()
	if r.at != len(r.toks) {
		panic("text left after the expression")
	}
	return codesVerdict{expected: value.answer(), reason: r.reason}, nil
}

var (
	codesSystemDecl  = regexp.MustCompile(`(?m)^codesystem "(\w+)": '([^']*)'(?: version ('[^']*'))?$`)
	codesCodeDecl    = regexp.MustCompile(`(?m)^code "(\w+)": '([^']*)' from "(\w+)"(?: display ('[^']*'))?$`)
	codesConceptDecl = regexp.MustCompile(`(?m)^concept "(\w+)": \{ ([^}]*) \}(?: display ('[^']*'))?$`)
)

// codesCorpus is the definitions of testdata/snapshots/codes.cql and the codes
// and concepts its preamble declares.
func codesCorpus(t *testing.T) (defs map[string]string, declared map[string]codesValue) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", "codes.cql"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(raw)
	// An optional clause is captured with its quotes, so that one left out
	// reads apart from one of the empty String.
	optional := func(quoted string) *string {
		if quoted == "" {
			return nil
		}
		s := strings.Trim(quoted, "'")
		return &s
	}
	type system struct{ id, version *string }
	systems := map[string]system{}
	for _, m := range codesSystemDecl.FindAllStringSubmatch(src, -1) {
		id := m[2]
		systems[m[1]] = system{&id, optional(m[3])}
	}
	declared = map[string]codesValue{}
	for name, cs := range systems {
		declared["codesystem "+name] = codesValue{kind: "code", fields: [4]*string{nil, cs.id, nil, cs.version}}
	}
	for _, m := range codesCodeDecl.FindAllStringSubmatch(src, -1) {
		cs, ok := systems[m[3]]
		if !ok {
			t.Fatalf("code %s is from %s, which the preamble does not declare", m[1], m[3])
		}
		code := m[2]
		declared[m[1]] = codesValue{kind: "code", fields: [4]*string{&code, cs.id, optional(m[4]), cs.version}}
	}
	for _, m := range codesConceptDecl.FindAllStringSubmatch(src, -1) {
		k := codesValue{kind: "concept", text: optional(m[3])}
		for _, name := range strings.Split(m[2], ", ") {
			c, ok := declared[strings.Trim(name, `"`)]
			if !ok {
				t.Fatalf("concept %s holds %s, which the preamble does not declare", m[1], name)
			}
			k.codes = append(k.codes, c)
		}
		declared[m[1]] = k
	}
	defs = map[string]string{}
	for _, m := range regexp.MustCompile(`(?m)^define (D\d+): (.*)$`).FindAllStringSubmatch(src, -1) {
		defs[m[1]] = m[2]
	}
	return defs, declared
}

// TestTheCodesVerdictsFollowTheOracle holds every verdict of the codes area
// to what the oracle computes for its definition: its expected answer, the
// class that follows from it, and the evidence that says so.
func TestTheCodesVerdictsFollowTheOracle(t *testing.T) {
	defs, declared := codesCorpus(t)
	path := filepath.Join("testdata", "snapshots", "codes.divergences.tsv")
	recorded := readDivergences(t, path)
	if *classifyCodes {
		for key, d := range recorded {
			recorded[key] = classifiedByCodesOracle(t, defs, declared, key, d)
		}
		writeVerdicts(t, path, recorded)
		return
	}
	for _, key := range slices.Sorted(maps.Keys(recorded)) {
		d := recorded[key]
		want := classifiedByCodesOracle(t, defs, declared, key, d)
		if want.Class != d.Class || want.Expected != d.Expected || want.Evidence != d.Evidence {
			t.Errorf("%s: recorded %s expecting %q (%s); the oracle gives %s expecting %q (%s)",
				oneLine(key), d.Class, d.Expected, oneLine(d.Evidence), want.Class, want.Expected, oneLine(want.Evidence))
		}
	}
}

// classifiedByCodesOracle is the verdict the oracle gives a divergence: the
// specification decides every definition of the area, so a divergence is a
// defect of whichever answer is not the oracle's.
func classifiedByCodesOracle(t *testing.T, defs map[string]string, declared map[string]codesValue, key string, d divergence) divergence {
	t.Helper()
	name, _, _ := strings.Cut(key, "\t")
	v, err := codesExpected(declared, defs[name])
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	d.Expected = v.expected
	d.Class = ownDefect
	if expectedMatches(d.Expected, d.Engine) {
		d.Class = referenceDefect
	}
	d.Evidence = "codes oracle: " + v.reason + "; " + strings.TrimPrefix(codesOracleRule, "codes oracle ")
	return d
}

// TestTheCodesOracleAgreesWhereAllAgree holds the oracle to the definitions
// where this engine and both references give one answer: there, the oracle
// gives that answer. It is what checks the oracle where no divergence reaches.
func TestTheCodesOracleAgreesWhereAllAgree(t *testing.T) {
	defs, declared := codesCorpus(t)
	recorded := readDivergences(t, filepath.Join("testdata", "snapshots", "codes.divergences.tsv"))
	java := readReferenceResults(t, "codes", "java")
	agreed := 0
	for _, name := range slices.Sorted(maps.Keys(defs)) {
		_, javaDiffers := recorded[name+"\tjava"]
		_, jsDiffers := recorded[name+"\tjs"]
		raw, answered := java[name]
		if javaDiffers || jsDiffers || !answered {
			continue
		}
		agreed++
		v, err := codesExpected(declared, defs[name])
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

// TestTheCodesOracleReadsTheWholeCorpus holds the oracle to every definition
// of the corpus, divergent or not: one it cannot read is a gap in it.
func TestTheCodesOracleReadsTheWholeCorpus(t *testing.T) {
	defs, declared := codesCorpus(t)
	if len(defs) != codesCorpusSize {
		t.Fatalf("the oracle reads %d definitions of codes.cql; the corpus has %d", len(defs), codesCorpusSize)
	}
	if len(declared) != 7 {
		t.Fatalf("the oracle reads %d declarations of codes.cql's preamble; it has 7", len(declared))
	}
	for name, expr := range defs {
		if _, err := codesExpected(declared, expr); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}
