package cql

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The scope area's oracle (docs/plans/003-clasificar-divergencias.md, PR 5):
// what each definition of testdata/snapshots/scope.cql is when every name
// resolves where it is written. Its corpus is a closed grammar
// (internal/referenceharness/differential/gen, scope): Integer literals, + -
// * and >, the preamble's definitions and functions, Sum, Count and Max, list
// literals, and queries whose alias, let and where bind the preamble's names.
// The oracle reads the preamble from the file as well, and shares no code
// with this engine's parser or evaluator.
//
// The specification, CQL 1.5.3 (cql.hl7.org/N1A):
//   - Logical Specification, Reusing Logic: "The result of evaluating an
//     ExpressionReference is the result of evaluating the referenced
//     NamedExpression", and an OperandRef is "the value of an operand ...
//     within the body of a function definition"; Queries: an AliasRef and a
//     QueryLetRef reference a source or a let "within the scope of a query".
//     Inferred from them, not written: a definition's value does not depend
//     on where it is referenced, so it is evaluated with no alias, let or
//     operand in scope, and a function's body sees only its operands. The
//     reference translator resolves the names that way: DQuery's body is
//     Add(AliasRef A, ExpressionRef B) in its ELM.
//   - Author's Guide, Queries: "By default, a query returns a list of
//     distinct results, suppressing duplicates", and "only elements where
//     the condition evaluates to true are included in the result".
//   - Appendix B, Distinct: "nulls are considered equal for the purposes of
//     distinct determination".
//   - Appendix B, Add: "If either argument is null, the result is null", and
//     "If the result of the addition cannot be represented (i.e. arithmetic
//     overflow), the result is null"; Arithmetic Operators: operations that
//     overflow result in null. An Integer is of 32 bits (ADR 002).
//   - Appendix B, Aggregate Functions: Sum "returns the sum of non-null
//     elements" and Count "the number of non-null elements"; Sum and Max are
//     null when "the source contains no non-null elements", and Count is 0.

const scopeOracleRule = "CQL 1.5.3 (cql.hl7.org/N1A): Logical Specification, ExpressionRef (\"The result of " +
	"evaluating an ExpressionReference is the result of evaluating the referenced NamedExpression\"), AliasRef, " +
	"QueryLetRef and OperandRef; Author's Guide, Queries; Appendix B, Add, Distinct and Aggregate Functions"

var classifyScope = flag.Bool("classify-scope", false,
	"give the divergences of the scope area the verdict of its oracle")

// sValue is a value of the scope grammar: null, an Integer, a Boolean of a
// where, or a list.
type sValue struct {
	kind byte // 'n' null, 'i' Integer, 'b' Boolean, 'l' list
	i    int64
	list []sValue
}

var sNull = sValue{kind: 'n'}

func (v sValue) String() string {
	switch v.kind {
	case 'n':
		return "null"
	case 'i':
		return "Integer " + strconv.FormatInt(v.i, 10)
	case 'l':
		// As this engine writes a list: List {1, null}.
		elems := make([]string, len(v.list))
		for i, e := range v.list {
			elems[i] = strings.TrimPrefix(e.String(), "Integer ")
		}
		return "List {" + strings.Join(elems, ", ") + "}"
	}
	return fmt.Sprintf("unexpected %c", v.kind)
}

// sNode is an expression of the scope grammar.
type sNode struct {
	kind string // int, null, name, op, call, list, query
	text string
	i    int64
	args []*sNode
	q    *sQuery
}

// sQuery is a query of one source: its alias, an optional let and where, and
// its return, distinct.
type sQuery struct {
	source, let, where, ret *sNode
	alias, letName          string
}

type sFunction struct {
	operands []string
	body     *sNode
}

// sLibrary is the definitions and functions of scope.cql.
type sLibrary struct {
	defs  map[string]*sNode
	funcs map[string]sFunction
	memo  map[string]sValue
	// dynamic, set only by a mutation, is cql-execution's reading.
	dynamic bool
}

var (
	scopeToken     = regexp.MustCompile(`^\s*(\d+|[A-Za-z][A-Za-z0-9]*|[(){},.+\-*>:])`)
	scopeDefine    = regexp.MustCompile(`^define ([A-Za-z][A-Za-z0-9]*): (.*)$`)
	scopeFunction  = regexp.MustCompile(`^define (fluent )?function ([A-Za-z][A-Za-z0-9]*)\(([^)]*)\): (.*)$`)
	scopeOperand   = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9]*) Integer$`)
	scopeKeywords  = []string{"let", "where", "return", "null"}
	scopeAggregate = []string{"Sum", "Count", "Max"}
	scopeCorpusDef = regexp.MustCompile(`^define D\d+:`)
	scopeCorpus    = regexp.MustCompile(`^D\d+$`)
)

func scopeTokens(s string) ([]string, error) {
	var out []string
	for strings.TrimSpace(s) != "" {
		m := scopeToken.FindStringSubmatch(s)
		if m == nil {
			return nil, fmt.Errorf("cannot read %q", s)
		}
		out = append(out, m[1])
		s = s[len(m[0]):]
	}
	return out, nil
}

type sParser struct {
	toks []string
	pos  int
}

func (p *sParser) peek() string {
	if p.pos < len(p.toks) {
		return p.toks[p.pos]
	}
	return ""
}

func (p *sParser) next() string {
	t := p.peek()
	p.pos++
	return t
}

func (p *sParser) expect(t string) error {
	if got := p.next(); got != t {
		return fmt.Errorf("expected %q, found %q", t, got)
	}
	return nil
}

func isScopeName(t string) bool {
	return t != "" && (t[0] >= 'A' && t[0] <= 'Z' || t[0] >= 'a' && t[0] <= 'z') && !slices.Contains(scopeKeywords, t)
}

// parseScope reads one whole expression; trailing input is an error.
func parseScope(s string) (*sNode, error) {
	toks, err := scopeTokens(s)
	if err != nil {
		return nil, err
	}
	p := &sParser{toks: toks}
	n, err := p.expr()
	if err != nil {
		return nil, err
	}
	if p.pos != len(p.toks) {
		return nil, fmt.Errorf("trailing %q", strings.Join(p.toks[p.pos:], " "))
	}
	return n, nil
}

// expr is a comparison, the lowest precedence, then + and -, then *.
func (p *sParser) expr() (*sNode, error) {
	a, err := p.additive()
	if err != nil || p.peek() != ">" {
		return a, err
	}
	p.next()
	b, err := p.additive()
	if err != nil {
		return nil, err
	}
	return &sNode{kind: "op", text: ">", args: []*sNode{a, b}}, nil
}

func (p *sParser) additive() (*sNode, error) {
	a, err := p.multiplicative()
	for err == nil && (p.peek() == "+" || p.peek() == "-") {
		op := p.next()
		var b *sNode
		if b, err = p.multiplicative(); err == nil {
			a = &sNode{kind: "op", text: op, args: []*sNode{a, b}}
		}
	}
	return a, err
}

func (p *sParser) multiplicative() (*sNode, error) {
	a, err := p.postfix()
	for err == nil && p.peek() == "*" {
		p.next()
		var b *sNode
		if b, err = p.postfix(); err == nil {
			a = &sNode{kind: "op", text: "*", args: []*sNode{a, b}}
		}
	}
	return a, err
}

// postfix is a primary, the query it is the source of when a parenthesized
// one is followed by an alias, and the fluent calls on it.
func (p *sParser) postfix() (*sNode, error) {
	n, paren, err := p.primary()
	if err != nil {
		return nil, err
	}
	if paren && isScopeName(p.peek()) {
		if n, err = p.query(n); err != nil {
			return nil, err
		}
	}
	for p.peek() == "." {
		p.next()
		name := p.next()
		args, err := p.arguments()
		if err != nil {
			return nil, err
		}
		n = &sNode{kind: "call", text: name, args: append([]*sNode{n}, args...)}
	}
	return n, nil
}

func (p *sParser) arguments() ([]*sNode, error) {
	if err := p.expect("("); err != nil {
		return nil, err
	}
	var args []*sNode
	for p.peek() != ")" {
		if len(args) > 0 {
			if err := p.expect(","); err != nil {
				return nil, err
			}
		}
		a, err := p.expr()
		if err != nil {
			return nil, err
		}
		args = append(args, a)
	}
	p.next()
	return args, nil
}

func (p *sParser) primary() (n *sNode, paren bool, err error) {
	switch t := p.next(); {
	case t == "(":
		if n, err = p.expr(); err == nil {
			err = p.expect(")")
		}
		return n, true, err
	case t == "{":
		list := &sNode{kind: "list"}
		for p.peek() != "}" {
			if len(list.args) > 0 {
				if err := p.expect(","); err != nil {
					return nil, false, err
				}
			}
			e, err := p.expr()
			if err != nil {
				return nil, false, err
			}
			list.args = append(list.args, e)
		}
		p.next()
		return list, false, nil
	case t == "null":
		return &sNode{kind: "null"}, false, nil
	case t != "" && t[0] >= '0' && t[0] <= '9':
		i, err := strconv.ParseInt(t, 10, 64)
		return &sNode{kind: "int", i: i}, false, err
	case isScopeName(t):
		if p.peek() != "(" {
			return &sNode{kind: "name", text: t}, false, nil
		}
		args, err := p.arguments()
		return &sNode{kind: "call", text: t, args: args}, false, err
	default:
		return nil, false, fmt.Errorf("unexpected %q", t)
	}
}

func (p *sParser) query(source *sNode) (*sNode, error) {
	q := &sQuery{source: source, alias: p.next()}
	var err error
	if p.peek() == "let" {
		p.next()
		q.letName = p.next()
		if !isScopeName(q.letName) {
			return nil, fmt.Errorf("a let named %q", q.letName)
		}
		if t := p.next(); t != ":" {
			return nil, fmt.Errorf("expected \":\", found %q", t)
		}
		if q.let, err = p.expr(); err != nil {
			return nil, err
		}
	}
	if p.peek() == "where" {
		p.next()
		if q.where, err = p.expr(); err != nil {
			return nil, err
		}
	}
	if t := p.next(); t != "return" {
		return nil, fmt.Errorf("expected \"return\", found %q", t)
	}
	if q.ret, err = p.expr(); err != nil {
		return nil, err
	}
	return &sNode{kind: "query", q: q}, nil
}

// parseScopeLibrary reads a library of the scope grammar: its definitions
// and functions, in order of their names.
func parseScopeLibrary(src string) (*sLibrary, []string, error) {
	l := &sLibrary{defs: map[string]*sNode{}, funcs: map[string]sFunction{}, memo: map[string]sValue{}}
	var names []string
	for line := range strings.SplitSeq(src, "\n") {
		if line == "" || strings.HasPrefix(line, "library ") {
			continue
		}
		if m := scopeFunction.FindStringSubmatch(line); m != nil {
			var f sFunction
			for op := range strings.SplitSeq(m[3], ", ") {
				o := scopeOperand.FindStringSubmatch(op)
				if o == nil {
					return nil, nil, fmt.Errorf("%s: operand %q", m[2], op)
				}
				f.operands = append(f.operands, o[1])
			}
			body, err := parseScope(m[4])
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %w", m[2], err)
			}
			f.body = body
			l.funcs[m[2]] = f
			continue
		}
		m := scopeDefine.FindStringSubmatch(line)
		if m == nil {
			return nil, nil, fmt.Errorf("cannot read the line %q", line)
		}
		n, err := parseScope(m[2])
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", m[1], err)
		}
		l.defs[m[1]] = n
		names = append(names, m[1])
	}
	return l, names, nil
}

// sEnv is the aliases, lets and operands in scope, innermost first.
type sEnv struct {
	name   string
	v      sValue
	parent *sEnv
}

func (e *sEnv) with(name string, v sValue) *sEnv { return &sEnv{name: name, v: v, parent: e} }

func (e *sEnv) lookup(name string) (sValue, bool) {
	for ; e != nil; e = e.parent {
		if e.name == name {
			return e.v, true
		}
	}
	return sValue{}, false
}

// definition is a definition's value: evaluated where it is defined, with
// nothing in scope, once.
func (l *sLibrary) definition(name string, caller *sEnv) (sValue, error) {
	if l.dynamic {
		return l.eval(l.defs[name], caller)
	}
	if v, ok := l.memo[name]; ok {
		return v, nil
	}
	v, err := l.eval(l.defs[name], nil)
	if err == nil {
		l.memo[name] = v
	}
	return v, err
}

func scopeInteger(i int64) sValue {
	if i < math.MinInt32 || i > math.MaxInt32 {
		return sNull // "cannot be represented": null (Appendix B, Add)
	}
	return sValue{kind: 'i', i: i}
}

func (l *sLibrary) eval(n *sNode, env *sEnv) (sValue, error) {
	switch n.kind {
	case "int":
		if v := scopeInteger(n.i); v.kind == 'i' {
			return v, nil
		}
		return sValue{}, fmt.Errorf("the literal %d is past an Integer", n.i)
	case "null":
		return sNull, nil
	case "name":
		if v, ok := env.lookup(n.text); ok {
			return v, nil
		}
		if _, ok := l.defs[n.text]; ok {
			return l.definition(n.text, env)
		}
		return sValue{}, fmt.Errorf("no %s in scope", n.text)
	case "list":
		out := sValue{kind: 'l'}
		for _, a := range n.args {
			v, err := l.eval(a, env)
			if err != nil {
				return sValue{}, err
			}
			out.list = append(out.list, v)
		}
		return out, nil
	case "op":
		a, err := l.eval(n.args[0], env)
		if err != nil {
			return sValue{}, err
		}
		b, err := l.eval(n.args[1], env)
		if err != nil {
			return sValue{}, err
		}
		if a.kind == 'n' || b.kind == 'n' {
			return sNull, nil // "If either argument is null, the result is null"
		}
		if a.kind != 'i' || b.kind != 'i' {
			return sValue{}, fmt.Errorf("%s of %c and %c", n.text, a.kind, b.kind)
		}
		switch n.text {
		case "+":
			return scopeInteger(a.i + b.i), nil
		case "-":
			return scopeInteger(a.i - b.i), nil
		case "*":
			return scopeInteger(a.i * b.i), nil
		}
		v := sValue{kind: 'b'}
		if a.i > b.i {
			v.i = 1
		}
		return v, nil
	case "call":
		return l.call(n, env)
	case "query":
		return l.query(n.q, env)
	}
	return sValue{}, fmt.Errorf("unknown node %s", n.kind)
}

func (l *sLibrary) call(n *sNode, env *sEnv) (sValue, error) {
	args := make([]sValue, len(n.args))
	for i, a := range n.args {
		v, err := l.eval(a, env)
		if err != nil {
			return sValue{}, err
		}
		args[i] = v
	}
	if slices.Contains(scopeAggregate, n.text) {
		if len(args) != 1 || args[0].kind != 'l' && args[0].kind != 'n' {
			return sValue{}, fmt.Errorf("%s of %d arguments", n.text, len(args))
		}
		for _, e := range args[0].list {
			if e.kind != 'i' && e.kind != 'n' {
				return sValue{}, fmt.Errorf("%s of a list of %c", n.text, e.kind)
			}
		}
		return scopeAggregateOf(n.text, args[0]), nil
	}
	f, ok := l.funcs[n.text]
	if !ok || len(f.operands) != len(args) {
		return sValue{}, fmt.Errorf("no function %s of %d operands", n.text, len(args))
	}
	// A function's body sees its operands, and nothing of its caller.
	var body *sEnv
	for i, op := range f.operands {
		body = body.with(op, args[i])
	}
	return l.eval(f.body, body)
}

func scopeAggregateOf(name string, list sValue) sValue {
	if list.kind == 'n' {
		if name == "Count" {
			return scopeInteger(0) // "If the list is null, the result is 0"
		}
		return sNull
	}
	var values []int64
	for _, e := range list.list {
		if e.kind == 'i' {
			values = append(values, e.i) // the non-null elements
		}
	}
	switch {
	case name == "Count":
		return scopeInteger(int64(len(values)))
	case len(values) == 0:
		return sNull // "If the source contains no non-null elements, null is returned"
	case name == "Max":
		return scopeInteger(slices.Max(values))
	}
	sum := int64(0)
	for _, v := range values {
		// Each step is an addition, null past the range.
		if s := scopeInteger(sum + v); s.kind == 'i' {
			sum = s.i
		} else {
			return sNull
		}
	}
	return scopeInteger(sum)
}

func (l *sLibrary) query(q *sQuery, env *sEnv) (sValue, error) {
	// The source is evaluated before its alias is bound.
	source, err := l.eval(q.source, env)
	if err != nil || source.kind == 'n' {
		return source, err
	}
	if source.kind != 'l' {
		return sValue{}, fmt.Errorf("a query over %c", source.kind)
	}
	out := sValue{kind: 'l'}
	for _, e := range source.list {
		inner := env.with(q.alias, e)
		if q.let != nil {
			// A let's value is evaluated before its name is bound.
			v, err := l.eval(q.let, inner)
			if err != nil {
				return sValue{}, err
			}
			inner = inner.with(q.letName, v)
		}
		if q.where != nil {
			c, err := l.eval(q.where, inner)
			if err != nil {
				return sValue{}, err
			}
			if c.kind != 'b' && c.kind != 'n' {
				return sValue{}, fmt.Errorf("a where of %c", c.kind)
			}
			if c.kind != 'b' || c.i != 1 {
				continue // only a true condition keeps the element
			}
		}
		r, err := l.eval(q.ret, inner)
		if err != nil {
			return sValue{}, err
		}
		if r.kind != 'i' && r.kind != 'n' {
			return sValue{}, fmt.Errorf("a return of %c", r.kind)
		}
		// Distinct, with nulls equal to each other.
		if !slices.ContainsFunc(out.list, func(o sValue) bool { return o.kind == r.kind && o.i == r.i }) {
			out.list = append(out.list, r)
		}
	}
	return out, nil
}

// scopeOracle is the oracle over scope.cql: the library and the answer of
// each of its D definitions.
func scopeOracle(t *testing.T) (lib *sLibrary, preamble string, answers map[string]string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", "scope.cql"))
	if err != nil {
		t.Fatal(err)
	}
	lib, names, err := parseScopeLibrary(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for line := range strings.SplitSeq(string(raw), "\n") {
		if line != "" && !strings.HasPrefix(line, "library ") && !scopeCorpusDef.MatchString(line) {
			b.WriteString(line + "\n")
		}
	}
	answers = map[string]string{}
	for _, name := range names {
		if !scopeCorpus.MatchString(name) {
			continue
		}
		v, err := lib.definition(name, nil)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		answers[name] = v.String()
	}
	return lib, b.String(), answers
}

// readReferenceScope is a reference's answer in the snapshot's text, or
// "unreadable" and why.
func readReferenceScope(raw json.RawMessage) string {
	if string(raw) == "null" {
		return "null"
	}
	var v struct{ Type, Value string }
	if err := json.Unmarshal(raw, &v); err != nil || v.Type != "Integer" {
		return "unreadable " + string(raw)
	}
	if _, err := strconv.ParseInt(v.Value, 10, 32); err != nil {
		return "unreadable " + string(raw)
	}
	return "Integer " + v.Value
}

// TestTheScopeOracleReadsTheWholeCorpus holds the oracle to the whole file:
// the preamble's eight definitions and four functions, and the 600
// definitions, each read and evaluated.
func TestTheScopeOracleReadsTheWholeCorpus(t *testing.T) {
	lib, _, answers := scopeOracle(t)
	if len(answers) != 600 {
		t.Errorf("the oracle reads %d definitions; scope.cql has 600", len(answers))
	}
	want := []string{"A", "B", "C", "DA", "DB", "DAB", "DList", "DQuery"}
	for _, name := range want {
		if _, ok := lib.defs[name]; !ok {
			t.Errorf("the oracle does not read the preamble's %s", name)
		}
	}
	if len(lib.defs) != len(want)+600 {
		t.Errorf("the oracle reads %d definitions; the preamble has %d and the corpus 600", len(lib.defs), len(want))
	}
	if got := slices.Sorted(maps.Keys(lib.funcs)); !slices.Equal(got, []string{"F", "G", "H", "Up"}) {
		t.Errorf("the oracle reads the functions %v", got)
	}
	for _, bad := range []string{"Sum({ 1 }) 2", "Avg({ 1 })", "1 / 2", "(({ 1 }) A return A", "Sum(({ 1 }) A let return A)",
		"2147483648", "(({ 1 }) A where 1 return A)", "Count({ 1 > 0 })", "Sum({ { 1 } })"} {
		if n, err := parseScope(bad); err == nil {
			if _, err := lib.eval(n, nil); err == nil {
				t.Errorf("the oracle reads %q", bad)
			}
		}
	}
}

// TestTheScopeOracleAgreesWithJava holds the oracle to the Java engine, a
// second opinion that shares nothing with it, on every definition.
func TestTheScopeOracleAgreesWithJava(t *testing.T) {
	_, _, answers := scopeOracle(t)
	java := readReferenceResults(t, "scope", "java")
	for _, name := range slices.Sorted(maps.Keys(answers)) {
		if got := readReferenceScope(java[name]); got != answers[name] {
			t.Errorf("%s: the oracle gives %s and the Java engine %s", name, answers[name], got)
		}
	}
}

// TestTheEngineFollowsTheScopeOracle holds this engine, as it answers now and
// as the snapshot the verdicts read records it, to the oracle on every
// definition.
func TestTheEngineFollowsTheScopeOracle(t *testing.T) {
	_, _, answers := scopeOracle(t)
	live := snapshotAnswers(t, "scope")
	recorded := readSnapshot(t, filepath.Join("testdata", "snapshots", "scope.answers.tsv"))
	for _, name := range slices.Sorted(maps.Keys(answers)) {
		if live[name] != answers[name] || recorded[name] != answers[name] {
			t.Errorf("%s: this engine answers %s, and the snapshot %s; the oracle gives %s", name, live[name], recorded[name], answers[name])
		}
	}
}

// scopeControls are definitions the corpus does not reach, each with what
// the specification makes it, worked by hand from the rule in its comment.
var scopeControls = map[string]string{
	// A definition is the same wherever it is referenced: B in DB is the
	// definition, not the alias; DList's C is the definition's 100.
	"Sum(({ 5 }) B return DB)":         "Integer 20",
	"Sum(({ 3 }) C return Max(DList))": "Integer 100",
	// G(1) is F(1) + DB: DA + 1 + B * 2, with B the definition's 10.
	"Sum(({ 3 }) C return G(1))": "Integer 23",
	// An inner alias hides an outer one; an outer one is seen inside.
	"Sum(({ 1 }) A return Sum(({ 2 }) A return A))":           "Integer 2",
	"Sum(({ 1, 2 }) Q return Sum(({ 10 }) X return (X + Q)))": "Integer 23",
	// A let hides an outer alias of its name: B is the let's Q, not the 9.
	"Sum(({ 9 }) B return Sum(({ 1 }) Q let B: Q return B))": "Integer 1",
	// The source is evaluated before its alias is bound, and a let's value
	// before its name: { A } is { 1 }, and B in let B: B is 10.
	"Sum(({ A }) A return A)":          "Integer 1",
	"Sum(({ 7 }) Q let B: B return B)": "Integer 10",
	// A fluent receiver is evaluated where it is written: Up(5) is 5 + DA.
	"Sum(({ 5 }) A return (A).Up())": "Integer 7",
	// A return is distinct, with nulls equal to each other; a where keeps
	// only a true condition.
	"Sum(({ 3, 4 }) Q return 5)": "Integer 5",
	"({ 1, 2 }) Q return null":   "List {null}",
	"({ 3, 4, 3 }) Q return Q":   "List {3, 4}",
	"Count(({ 1, 2 }) Q where Sum(({ Q }) X where X > 1 return X) > 0 return Q)": "Integer 1",
	// * binds tighter than + and -.
	"(1 + 2 * 3)": "Integer 7",
	// Aggregates: the non-null elements; none is null, except Count's 0,
	// and so is a null list.
	"Count(null)":                           "Integer 0",
	"Sum(null)":                             "null",
	"Max(null)":                             "null",
	"Count({ 1, null, 2 })":                 "Integer 2",
	"Sum({ 1, null, 2 })":                   "Integer 3",
	"Max({ (0 - 3), (0 - 5) })":             "Integer -3",
	"Max({ null, 2 })":                      "Integer 2",
	"Sum(({ 1 }) Q where Q > 5 return Q)":   "null",
	"Count(({ 1 }) Q where Q > 5 return Q)": "Integer 0",
	"Sum(({ 1, 2 }) Q return Sum(({ Q }) X where X > 1 return X))": "Integer 2",
	// An Integer past 32 bits is null, and so is what is computed from it.
	"((2147483647 + 1) - 1)":          "null",
	"Sum({ 2147483647, 1 })":          "null",
	"(null + 1)":                      "null",
	"Count(({ 1, 2 }) Q return null)": "Integer 0",
}

// TestTheScopeOracleHoldsItsControls holds the oracle, and this engine, to
// the controls.
func TestTheScopeOracleHoldsItsControls(t *testing.T) {
	lib, preamble, _ := scopeOracle(t)
	engine := NewEngine()
	for _, expr := range slices.Sorted(maps.Keys(scopeControls)) {
		want := scopeControls[expr]
		n, err := parseScope(expr)
		if err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
		if got, oerr := lib.eval(n, nil); oerr != nil || got.String() != want {
			t.Errorf("%s: the oracle gives %s (%v); the specification gives %s", expr, got, oerr, want)
		}
		got, err := engine.EvaluateExpression(t.Context(), "library Control version '1.0'\n"+preamble+"define X: "+expr+"\n", "X", nil, nil)
		if err != nil {
			t.Errorf("%s: this engine fails: %v", expr, err)
			continue
		}
		text := "null"
		if got != nil {
			text = got.Type() + " " + got.String()
		}
		if text != want {
			t.Errorf("%s: this engine answers %s; the specification gives %s", expr, text, want)
		}
	}
}

// classifiedByScopeOracle is the verdict the oracle gives a divergence.
func classifiedByScopeOracle(key string, d divergence, want, engine, reference string) (divergence, error) {
	_, ref, _ := strings.Cut(key, "\t")
	oracle := "scope oracle (engine_scope_oracle_test.go): with each name resolved where it is written, as the " +
		"reference translator's ELM has it (DQuery is Add(AliasRef A, ExpressionRef B)), the definition is " + want +
		"; " + scopeOracleRule
	switch {
	case engine != want:
		d.Class, d.Expected, d.Evidence = ownDefect, want, oracle+"; this engine answers "+engine
	case strings.HasPrefix(reference, "unreadable"):
		d.Class, d.Expected = unclassified, ""
		d.Evidence = "the reference harness cannot read the reference's answer; the scope oracle gives " + want
	case reference == want:
		return d, errors.New("the reference gives the oracle's answer: it is no divergence")
	default:
		d.Class, d.Expected, d.Evidence = referenceDefect, d.Engine, oracle
		if ref == "js" {
			d.Evidence += "; cql-execution looks a definition's names up through the context it is referenced " +
				"from, and keeps the first value the definition takes there: the harness evaluates each " +
				"definition in a context of its own (evaluate.js), so that first value is the caller's (plan 003, PR 5)"
		}
		d.Evidence += "; the reference answers " + d.Reference
	}
	return d, nil
}

// TestTheScopeVerdictsFollowTheOracle holds every divergence of the area to
// the verdict the oracle computes; -classify-scope writes them.
func TestTheScopeVerdictsFollowTheOracle(t *testing.T) {
	_, _, answers := scopeOracle(t)
	engine := readSnapshot(t, filepath.Join("testdata", "snapshots", "scope.answers.tsv"))
	refs := map[string]map[string]json.RawMessage{
		"java": readReferenceResults(t, "scope", "java"),
		"js":   readReferenceResults(t, "scope", "js"),
	}
	path := filepath.Join("testdata", "snapshots", "scope.divergences.tsv")
	recorded := readDivergences(t, path)
	classify := func(key string, d divergence) divergence {
		name, ref, _ := strings.Cut(key, "\t")
		raw, answered := refs[ref][name]
		if !answered {
			t.Fatalf("%s: %s gives no answer", name, ref)
		}
		got, err := classifiedByScopeOracle(key, d, answers[name], engine[name], readReferenceScope(raw))
		if err != nil {
			t.Fatalf("%s: %v", oneLine(key), err)
		}
		return got
	}
	if *classifyScope {
		for key, d := range recorded {
			recorded[key] = classify(key, d)
		}
		writeVerdicts(t, path, recorded)
		return
	}
	for _, key := range slices.Sorted(maps.Keys(recorded)) {
		d := recorded[key]
		want := classify(key, d)
		if want.Class != d.Class || want.Expected != d.Expected || want.Evidence != d.Evidence {
			t.Errorf("%s: recorded %s expecting %q (%s); the oracle gives %s expecting %q (%s)",
				oneLine(key), d.Class, d.Expected, oneLine(d.Evidence), want.Class, want.Expected, oneLine(want.Evidence))
		}
	}
}
