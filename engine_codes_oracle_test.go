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
	kind   string     // "code", "concept", "list", "string", "boolean", "integer", "null"
	fields [4]*string // a Code's code, system, display and version
	codes  []codesValue
	text   *string // a Concept's display or a String
	truth  tri
	number int
}

var codesFieldIndex = map[string]int{"code": 0, "system": 1, "display": 2, "version": 3}

var codesNull = codesValue{kind: "null"}

// answer is a value as answers writes it.
func (v codesValue) answer() string {
	switch v.kind {
	case "string":
		return strconv.Quote(*v.text)
	case "integer":
		return "Integer " + strconv.Itoa(v.number)
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

// sameCodesValue is the equality the operations that hold values use —
// distinct, union, intersect, except, in, contains, IndexOf, Mode and a
// query's return: "equality semantics, with the exception that nulls are
// considered equal", where an Equal that is unknown, a null element on one
// side, is not equal (both references answer so; plan 004).
func sameCodesValue(a, b codesValue) bool {
	if a.kind == "null" || b.kind == "null" {
		return a.kind == b.kind
	}
	return codesEqual(a, b) == triTrue
}

// indexOfCodes is the place of v in list, or -1.
func indexOfCodes(list []codesValue, v codesValue) int {
	for i, x := range list {
		if sameCodesValue(x, v) {
			return i
		}
	}
	return -1
}

// distinctCodes is list without the values an earlier one equals.
func distinctCodes(list []codesValue) []codesValue {
	var out []codesValue
	for _, v := range list {
		if indexOfCodes(out, v) < 0 {
			out = append(out, v)
		}
	}
	return out
}

// codesReader reads one definition of the corpus.
type codesReader struct {
	toks     []string
	at       int
	declared map[string]codesValue
	reason   string
	// mode is whether the expression asks Mode, and equivalentMode the values
	// of its mode by equivalence, the reading ADR 003's decision 5 leaves
	// aside; alternative is that reading's answer, "" when it has none.
	mode           bool
	equivalentMode []codesValue
	alternative    string
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
	switch op {
	case "=", "!=", "~", "!~", "union", "intersect", "except", "in", "contains":
	default:
		return a
	}
	r.next()
	b := r.postfix()
	switch op {
	case "union":
		r.reason = "union is the distinct values of both lists, by equality"
		return codesValue{kind: "list", codes: distinctCodes(append(slices.Clone(a.codes), b.codes...))}
	case "intersect", "except":
		var kept []codesValue
		for _, v := range a.codes {
			if (indexOfCodes(b.codes, v) >= 0) == (op == "intersect") {
				kept = append(kept, v)
			}
		}
		r.reason = op + " keeps the distinct values of the first list " + map[string]string{"intersect": "the second holds", "except": "the second does not hold"}[op] + ", by equality"
		return codesValue{kind: "list", codes: distinctCodes(kept)}
	case "in", "contains":
		list, v := b, a
		if op == "contains" {
			list, v = a, b
		}
		r.reason = op + " looks for a value equal to it, an unknown equality being no match"
		return codesValue{kind: "boolean", truth: triOf(indexOfCodes(list.codes, v) >= 0)}
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
			if r.mode && r.alternative == "" {
				r.alternative = alternativeField(r.equivalentMode, i)
			}
			v = codesText(v.fields[i])
		case v.kind == "concept" && field == "display":
			v = codesText(v.text)
		default:
			panic(fmt.Sprintf("no %s element %s", v.kind, field))
		}
		if r.reason == "" {
			r.reason = "an element reads as what the selector or the declaration sets, null when neither does"
		}
	}
	if r.peek() == "is" {
		r.next()
		r.expect("null")
		r.reason = "the empty String is a String, not null"
		return codesValue{kind: "boolean", truth: triOf(v.kind == "null")}
	}
	return v
}

// modeClass is the values of list in the class that occurs most often by
// same, or nil when two classes tie.
func modeClass(list []codesValue, same func(a, b codesValue) bool) []codesValue {
	var best []codesValue
	tie := false
	var seen []codesValue
	for _, v := range list {
		if slices.ContainsFunc(seen, func(x codesValue) bool { return same(x, v) }) {
			continue
		}
		seen = append(seen, v)
		var class []codesValue
		for _, x := range list {
			if same(x, v) {
				class = append(class, x)
			}
		}
		switch {
		case len(class) > len(best):
			best, tie = class, false
		case len(class) == len(best):
			tie = true
		}
	}
	if tie {
		return nil
	}
	return best
}

// modeOfCodes is the value that occurs most often, by equality; the corpus
// writes no list with a tie.
func modeOfCodes(list []codesValue) codesValue {
	best, count, tie := codesNull, 0, false
	for _, v := range distinctCodes(list) {
		n := 0
		for _, x := range list {
			if sameCodesValue(x, v) {
				n++
			}
		}
		switch {
		case n > count:
			best, count, tie = v, n, false
		case n == count:
			tie = true
		}
	}
	if tie {
		panic("a Mode with a tie, which the specification does not decide")
	}
	return best
}

// alternativeField is the answer of element i of the values of a mode by
// equivalence, "" when they give more than one or there is no mode. A class
// whose values differ in that element has no answer by equivalence: which
// value a reference keeps is not a reading of the specification, and
// cql-execution, which counts every Code as one value ("[object Object]"),
// would be excused by it.
func alternativeField(class []codesValue, i int) string {
	answer := ""
	for _, c := range class {
		a := codesText(c.fields[i]).answer()
		if answer != "" && a != answer {
			return ""
		}
		answer = a
	}
	return answer
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
	case tok == "Count":
		r.expect("(")
		v := r.expr()
		if r.peek() == "C" {
			// Count((list) C return C): a query's return is distinct by
			// default.
			r.next()
			r.expect("return")
			r.expect("C")
			v = codesValue{kind: "list", codes: distinctCodes(v.codes)}
			r.reason = "a query's return is distinct by default, by equality"
		}
		r.expect(")")
		// "If the list contains no non-null elements, the result is 0."
		n := 0
		for _, x := range v.codes {
			if x.kind != "null" {
				n++
			}
		}
		return codesValue{kind: "integer", number: n}
	case tok == "distinct":
		v := r.postfix()
		r.reason = "distinct keeps one of the values that are equal"
		return codesValue{kind: "list", codes: distinctCodes(v.codes)}
	case tok == "IndexOf":
		r.expect("(")
		list := r.expr()
		r.expect(",")
		v := r.expr()
		r.expect(")")
		r.reason = "IndexOf is the first place of a value equal to it, by equality"
		if v.kind == "null" {
			r.reason = "IndexOf of a null is null"
			return codesNull
		}
		return codesValue{kind: "integer", number: indexOfCodes(list.codes, v)}
	case tok == "Mode":
		r.expect("(")
		list := r.expr()
		r.expect(")")
		r.reason = "Mode is the value that occurs most often, by equality (" + undecidedADR + ", decision 5)"
		var present []codesValue
		for _, x := range list.codes {
			if x.kind != "null" {
				present = append(present, x)
			}
		}
		// The reading the decision leaves aside, by equivalence: the values
		// equivalent to the mode, nil when two classes tie.
		r.mode, r.equivalentMode = true, modeClass(present, codesEquivalent)
		return modeOfCodes(present)
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

// codesVerdict is what the oracle expects of a definition, and why. A Mode
// whose answer by equivalence is another than by equality depends on ADR
// 003's decision 5, and alternative is that other answer ("" for none).
type codesVerdict struct {
	expected, reason, alternative string
	decided                       bool
}

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
	v = codesVerdict{expected: value.answer(), reason: r.reason}
	if r.mode && r.alternative != v.expected {
		v.decided, v.alternative = true, r.alternative
	}
	return v, nil
}

var (
	codesSystemDecl  = regexp.MustCompile(`(?m)^codesystem "(\w+)": '([^']*)'(?: version ('[^']*'))?$`)
	codesCodeDecl    = regexp.MustCompile(`(?m)^code "(\w+)": '([^']*)' from "(\w+)"(?: display ('[^']*'))?$`)
	codesConceptDecl = regexp.MustCompile(`(?m)^concept "(\w+)": \{ ([^}]*) \}(?: display ('[^']*'))?$`)
)

// codesCorpus is the definitions of an area's corpus, codes or codesets, and
// the codes and concepts its preamble declares.
func codesCorpus(t *testing.T, area string) (defs map[string]string, declared map[string]codesValue) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", area+".cql"))
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

// codesAreas is the corpora the codes oracle reads: codes, the fields and
// the comparisons, and codesets, the operations that hold values (plan 004,
// PR 2).
var codesAreas = []struct {
	area     string
	size     int
	classify *bool
	rule     string
}{
	{"codes", codesCorpusSize, classifyCodes, codesOracleRule},
	{"codesets", 400, classifyCodeSets, codeSetsOracleRule},
}

const codeSetsOracleRule = "codes oracle (engine_codes_oracle_test.go), CQL 1.5.3 (cql.hl7.org/09-b-cqlreference.html): " +
	"Clinical Values (Code), Comparison Operators (Equal), List Operators (Distinct, Union, Intersect, Except, In, Contains, IndexOf), " +
	"Aggregate Functions (Mode), Queries (return)"

var classifyCodeSets = flag.Bool("classify-codesets", false,
	"give the unclassified divergences of the codesets area the verdict of its oracle")

// TestTheCodesVerdictsFollowTheOracle holds every verdict of the codes and
// codesets areas to what the oracle computes for its definition: its
// expected answer, the class that follows from it, and the evidence that says
// so.
func TestTheCodesVerdictsFollowTheOracle(t *testing.T) {
	for _, a := range codesAreas {
		defs, declared := codesCorpus(t, a.area)
		path := filepath.Join("testdata", "snapshots", a.area+".divergences.tsv")
		recorded := readDivergences(t, path)
		if *a.classify {
			for key, d := range recorded {
				recorded[key] = classifiedByCodesOracle(t, defs, declared, a.rule, key, d)
			}
			writeVerdicts(t, path, recorded)
			continue
		}
		for _, key := range slices.Sorted(maps.Keys(recorded)) {
			d := recorded[key]
			want := classifiedByCodesOracle(t, defs, declared, a.rule, key, d)
			if want.Class != d.Class || want.Expected != d.Expected || want.Evidence != d.Evidence {
				t.Errorf("%s %s: recorded %s expecting %q (%s); the oracle gives %s expecting %q (%s)", a.area,
					oneLine(key), d.Class, d.Expected, oneLine(d.Evidence), want.Class, want.Expected, oneLine(want.Evidence))
			}
		}
	}
}

// classifiedByCodesOracle is the verdict the oracle gives a divergence: the
// specification decides every definition of the area, so a divergence is a
// defect of whichever answer is not the oracle's.
func classifiedByCodesOracle(t *testing.T, defs map[string]string, declared map[string]codesValue, rule, key string, d divergence) divergence {
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
	d.Evidence = "codes oracle: " + v.reason + "; " + strings.TrimPrefix(rule, "codes oracle ")
	if d.Class == referenceDefect && v.decided && v.alternative != "" && d.Reference == v.alternative {
		// The reference counts by equivalence what the specification leaves
		// open, and ADR 003 decided by equality.
		d.Class = deliberate
		d.Evidence = "codes oracle: Mode counts by equality (" + undecidedADR + ", decision 5), which the reference reads " +
			"by equivalence, " + v.alternative + "; " + strings.TrimPrefix(rule, "codes oracle ")
	}
	return d
}

// TestTheCodesOracleAgreesWhereAllAgree holds the oracle to the definitions
// where this engine and both references give one answer: there, the oracle
// gives that answer. It is what checks the oracle where no divergence reaches.
func TestTheCodesOracleAgreesWhereAllAgree(t *testing.T) {
	for _, a := range codesAreas {
		defs, declared := codesCorpus(t, a.area)
		recorded := readDivergences(t, filepath.Join("testdata", "snapshots", a.area+".divergences.tsv"))
		java := readReferenceResults(t, a.area, "java")
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
				t.Fatalf("%s %s: %v", a.area, name, err)
			}
			if answer := pinned(answers.Reference(raw).Text); !expectedMatches(v.expected, answer) {
				t.Errorf("%s %s: all three answer %q; the oracle expects %q (%s)", a.area, name, oneLine(answer), v.expected, v.reason)
			}
		}
		if agreed == 0 {
			t.Fatalf("%s: no definition where all three agree: the oracle is checked on nothing", a.area)
		}
	}
}

// TestTheCodesOracleReadsTheWholeCorpus holds the oracle to every definition
// of the corpora, divergent or not: one it cannot read is a gap in it.
func TestTheCodesOracleReadsTheWholeCorpus(t *testing.T) {
	for _, a := range codesAreas {
		defs, declared := codesCorpus(t, a.area)
		if len(defs) != a.size {
			t.Fatalf("the oracle reads %d definitions of %s.cql; the corpus has %d", len(defs), a.area, a.size)
		}
		if len(declared) != 7 {
			t.Fatalf("the oracle reads %d declarations of %s.cql's preamble; it has 7", len(declared), a.area)
		}
		for name, expr := range defs {
			if _, err := codesExpected(declared, expr); err != nil {
				t.Errorf("%s %s: %v", a.area, name, err)
			}
		}
	}
}

// TestTheModeDecisionIsTheOnlyReadingLeftAside holds the classifier to ADR
// 003's decision 5 where no row of the corpus reaches it: a Mode whose answer
// by equivalence is another than by equality. A reference that gives the
// equivalence's answer reads otherwise what the specification leaves open, a
// deliberate difference; one that gives any other answer is a defect.
func TestTheModeDecisionIsTheOnlyReadingLeftAside(t *testing.T) {
	for _, tt := range []struct{ expr, equivalent string }{
		// By equality the mode is the two Codes of display d; by equivalence
		// it is the three Codes of code y, of display e.
		{"Mode({ Code { code: 'x', display: 'd' }, Code { code: 'x', display: 'd' }, Code { code: 'y', display: 'e' }, " +
			"Code { code: 'Y', display: 'e' }, Code { code: 'y', display: 'e', version: '1' } }).display", deliberate},
		// By equivalence the mode is all three Codes, which differ in display:
		// that reading has no answer, and e, the last Code's, is what
		// cql-execution gives by counting every Code as one value.
		{"Mode({ Code { code: 'x', display: 'd' }, Code { code: 'x', display: 'd' }, Code { code: 'x', display: 'e' } }).display", referenceDefect},
	} {
		defs := map[string]string{"D0": tt.expr}
		for _, ref := range []struct{ reference, class string }{
			{`"e"`, tt.equivalent},
			{`"z"`, referenceDefect},
		} {
			d := classifiedByCodesOracle(t, defs, nil, codeSetsOracleRule, "D0\tjs",
				divergence{answerPair: answerPair{Reference: ref.reference, Engine: `"d"`}})
			if d.Class != ref.class || d.Expected != `"d"` {
				t.Errorf("%s, a reference answering %s: %s expecting %s (%s); want %s expecting \"d\"",
					tt.expr, ref.reference, d.Class, d.Expected, d.Evidence, ref.class)
			}
			if ref.class == deliberate && !cites(d, undecidedADR) {
				t.Errorf("a deliberate Mode cites %q first, not ADR 003", d.Evidence)
			}
		}
	}
}
