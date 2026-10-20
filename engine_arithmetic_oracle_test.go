package cql

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

// The arithmetic area's oracle (docs/plans/003-clasificar-divergencias.md,
// PR 4): what the specification allows each definition of
// testdata/snapshots/arithmetic.cql to answer, and what the scale policy of
// docs/adr/002-escala-decimal.md makes this engine answer. Its corpus is a
// closed grammar (internal/referenceharness/differential/gen, arithmetic):
// Integer and Decimal literals, typed nulls and divisions by zero, nested
// under the arithmetic operators and functions and ten aggregates.
//
// The specification, CQL 1.5.3 (cql.hl7.org/N1A):
//   - Types: a Decimal has "a precision of 28 and a scale of 8", within
//     (10^28 − 1)/10^8 either way, and "implementations must support at least
//     the 28 digits of precision and 8 digits of scale, but may support more
//     precise values". An answer is conformant at any scale from 8 up, which
//     is why the oracle computes an interval of them, not a value; past the
//     range a value "cannot be represented" and is null (PR 4, decision 2).
//   - Appendix B, Arithmetic Operators: "operations that cause arithmetic
//     overflow or underflow, or otherwise cannot be performed (such as
//     division by 0) will result in null, rather than a run-time error"; `/`,
//     `div` and `mod` are null when "the right argument is 0"; div truncates
//     and mod is the remainder of it; Round is "a traditional round", -0.5 to
//     -1. Log in base 0 is null (PR 4, decision 3).
//   - Author's Guide, Aggregate Queries: "In general, nulls encountered during
//     aggregation are ignored"; Appendix B, Aggregate Functions: every
//     aggregate is null when "the source contains no non-null elements".
//     Appendix B writes Product, Median and the variances "of the elements"
//     and Sum and Avg "of non-null elements"; the guide decides the first.
//   - Inferred, not written: Log in base 1 divides by ln 1 = 0, and the
//     sample variance of one element by n - 1 = 0, which "cannot be
//     performed", so both are null.
//   - The conformance suite: Power(10, -7) is the Decimal 0.0000001, so a
//     whole-number power with a negative exponent is a Decimal.
const arithmeticOracleRule = "CQL 1.5.3 (cql.hl7.org/N1A): Types, Decimal; Appendix B, Arithmetic Operators " +
	"and Aggregate Functions; Author's Guide, nulls ignored in aggregation; docs/adr/002-escala-decimal.md"

var classifyArithmetic = flag.Bool("classify-arithmetic", false,
	"give the divergences of the arithmetic area the verdict of its oracle")

// aNode is a node of an arithmetic definition.
type aNode struct {
	kind string // "Integer", "Decimal", "null", "op", "neg", "call", "list"
	text string // a literal's text, a null's type, an operator, a function's name
	args []*aNode
}

// parseArithmetic reads a definition of the corpus.
func parseArithmetic(s string) (*aNode, error) {
	p := &aParser{src: s}
	n, err := p.expr(0)
	if err != nil {
		return nil, err
	}
	p.skip()
	if p.pos != len(p.src) {
		return nil, fmt.Errorf("unread from %d: %q", p.pos, p.src[p.pos:])
	}
	return n, nil
}

type aParser struct {
	src string
	pos int
}

var (
	aNumber = regexp.MustCompile(`^\d+(\.\d+)?`)
	aName   = regexp.MustCompile(`^[A-Za-z]+`)
	aNull   = regexp.MustCompile(`^\(null as (Integer|Decimal)\)`)
)

// aPrecedence is the binding of each binary operator (CQL 1.5.3, Appendix
// L): ^, then * / div mod, then + -.
var aPrecedence = map[string]int{"^": 3, "*": 2, "/": 2, "div": 2, "mod": 2, "+": 1, "-": 1}

func (p *aParser) skip() {
	for p.pos < len(p.src) && p.src[p.pos] == ' ' {
		p.pos++
	}
}

func (p *aParser) rest() string { p.skip(); return p.src[p.pos:] }

func (p *aParser) operator() string {
	r := p.rest()
	for _, op := range []string{"div", "mod", "+", "-", "*", "/", "^"} {
		if strings.HasPrefix(r, op) && (len(op) == 1 || len(r) > len(op) && r[len(op)] == ' ') {
			return op
		}
	}
	return ""
}

func (p *aParser) expr(floor int) (*aNode, error) {
	left, err := p.unary()
	if err != nil {
		return nil, err
	}
	for {
		op := p.operator()
		if op == "" || aPrecedence[op] < floor {
			return left, nil
		}
		p.pos += len(op)
		right, err := p.expr(aPrecedence[op] + 1)
		if err != nil {
			return nil, err
		}
		left = &aNode{kind: "op", text: op, args: []*aNode{left, right}}
	}
}

func (p *aParser) unary() (*aNode, error) {
	r := p.rest()
	switch {
	case aNull.MatchString(r):
		p.pos += len(aNull.FindString(r))
		return &aNode{kind: "null", text: aNull.FindStringSubmatch(r)[1]}, nil
	case strings.HasPrefix(r, "-"):
		p.pos++
		a, err := p.unary()
		return &aNode{kind: "neg", args: []*aNode{a}}, err
	case strings.HasPrefix(r, "("):
		p.pos++
		n, err := p.expr(0)
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(p.rest(), ")") {
			return nil, fmt.Errorf("no ) at %d", p.pos)
		}
		p.pos++
		return n, nil
	case strings.HasPrefix(r, "{"):
		p.pos++
		list := &aNode{kind: "list"}
		for {
			item, err := p.expr(0)
			if err != nil {
				return nil, err
			}
			list.args = append(list.args, item)
			switch next := p.rest(); {
			case strings.HasPrefix(next, ","):
				p.pos++
			case strings.HasPrefix(next, "}"):
				p.pos++
				return list, nil
			default:
				return nil, fmt.Errorf("no , or } at %d", p.pos)
			}
		}
	case aNumber.MatchString(r):
		lit := aNumber.FindString(r)
		p.pos += len(lit)
		kind := "Integer"
		if strings.Contains(lit, ".") {
			kind = "Decimal"
		}
		return &aNode{kind: kind, text: lit}, nil
	case aName.MatchString(r):
		name := aName.FindString(r)
		if !slices.Contains(aggregates, name) && !slices.Contains(arithmeticFunctions, name) {
			return nil, fmt.Errorf("a function the oracle does not know: %s", name)
		}
		p.pos += len(name)
		if !strings.HasPrefix(p.rest(), "(") {
			return nil, fmt.Errorf("a name the oracle does not know: %s", name)
		}
		p.pos++
		call := &aNode{kind: "call", text: name}
		for {
			arg, err := p.expr(0)
			if err != nil {
				return nil, err
			}
			call.args = append(call.args, arg)
			switch next := p.rest(); {
			case strings.HasPrefix(next, ","):
				p.pos++
			case strings.HasPrefix(next, ")"):
				p.pos++
				return call, nil
			default:
				return nil, fmt.Errorf("no , or ) at %d", p.pos)
			}
		}
	}
	return nil, fmt.Errorf("unread at %d: %q", p.pos, r)
}

// The bounds of the types (Types): a Decimal within (10^28 − 1)/10^8, an
// Integer within 32 bits.
var (
	decimalMax = decimal.RequireFromString("99999999999999999999.99999999")
	integerMin = decimal.NewFromInt(math.MinInt32)
	integerMax = decimal.NewFromInt(math.MaxInt32)
	scaleStep  = decimal.New(1, -8)
)

// powerCap is where a power stops being multiplied out: past it the value is
// out of the Decimal range, and its inverse within 10^-40 of 0.
var powerCap = decimal.New(1, 40)

// cappedPower is d to the |n|, multiplied out up to powerCap and with the
// sign the whole power has: past the cap, its magnitude is the cap's.
func cappedPower(d decimal.Decimal, n int64) decimal.Decimal {
	p := decimal.NewFromInt(1)
	for range max(n, -n) {
		if p = p.Mul(d); p.Abs().GreaterThan(powerCap) {
			p = powerCap
			if d.Sign() < 0 && n%2 != 0 {
				p = p.Neg()
			}
			break
		}
	}
	return p
}

// precision is the digits the oracle computes with: far past the 28 of the
// type, so its own rounding is no part of what it tells apart.
const precision = 40

// doubleError is the relative error an implementation that computes in
// binary64 may add at one operation: a few units in the last place.
var doubleError = decimal.RequireFromString("0.0000000000000004")

// aConform is what the specification allows a definition to answer: its
// type and an interval of values, or null; or that it does not decide.
type aConform struct {
	typ       string
	null      bool
	ambiguous string
	lo, hi    decimal.Decimal
}

func (c aConform) exact() bool { return c.lo.Equal(c.hi) }

func cNull(typ string) aConform { return aConform{typ: typ, null: true} }

func cAmbiguous(typ, why string) aConform { return aConform{typ: typ, ambiguous: why} }

// atScale reports whether a value has no digit past the Decimal scale: a
// value an implementation in decimal holds exactly.
func atScale(d decimal.Decimal) bool { return d.Equal(d.Truncate(8)) }

// isBinary reports whether a value is a binary64 exactly: a value an
// implementation in binary64 holds exactly.
func isBinary(d decimal.Decimal) bool {
	f, _ := d.Float64()
	if math.IsInf(f, 0) {
		return false
	}
	r, ok := new(big.Rat).SetString(d.String())
	return ok && new(big.Rat).SetFloat64(f).Cmp(r) == 0
}

// widen is the interval of what an implementation may give for the exact
// interval of an operation's result. In decimal, a value past the scale is
// rounded at a scale of 8 or more, of any mode: within 10^-8. In binary64, a
// value that is no binary64 is within a few units in its last place. The two
// are added: the interval holds both kinds of implementation.
func widen(c aConform) aConform {
	if c.null || c.ambiguous != "" || c.typ != "Decimal" {
		return c
	}
	slack := func(d decimal.Decimal) decimal.Decimal {
		s := decimal.Zero
		if !c.exact() || !atScale(d) {
			s = scaleStep
		}
		if !c.exact() || !isBinary(d) {
			s = s.Add(d.Abs().Mul(doubleError))
		}
		return s
	}
	lo, hi := c.lo.Sub(slack(c.lo)), c.hi.Add(slack(c.hi))
	// Rounding a value, in any mode, never crosses 0: a positive value
	// rounds to 0 at the least, a negative one to 0 at the most.
	if c.lo.Sign() >= 0 && lo.Sign() < 0 {
		lo = decimal.Zero
	}
	if c.hi.Sign() <= 0 && hi.Sign() > 0 {
		hi = decimal.Zero
	}
	c.lo, c.hi = lo, hi
	return c
}

// inRange is null past the type's range, and undecided across its edge.
func inRange(c aConform) aConform {
	if c.null || c.ambiguous != "" {
		return c
	}
	lo, hi := decimalMax.Neg(), decimalMax
	if c.typ == "Integer" {
		lo, hi = integerMin, integerMax
	}
	switch {
	case c.lo.GreaterThan(hi) || c.hi.LessThan(lo):
		return cNull(c.typ)
	case c.lo.LessThan(lo) || c.hi.GreaterThan(hi):
		return cAmbiguous(c.typ, "a result at the edge of the "+c.typ+" range")
	}
	return c
}

func span(typ string, values ...decimal.Decimal) aConform {
	return aConform{typ: typ, lo: decimal.Min(values[0], values[1:]...), hi: decimal.Max(values[0], values[1:]...)}
}

func numericType(a, b string) string {
	if a == "Integer" && b == "Integer" {
		return "Integer"
	}
	return "Decimal"
}

func quotient(a, b decimal.Decimal) decimal.Decimal { return a.DivRound(b, precision) }

// conform is the interval the specification allows a node to answer.
func conform(n *aNode) aConform {
	switch n.kind {
	case "Integer", "Decimal":
		d := decimal.RequireFromString(n.text)
		return widen(aConform{typ: n.kind, lo: d, hi: d})
	case "null":
		return cNull(n.text)
	case "neg":
		a := conform(n.args[0])
		if a.null || a.ambiguous != "" {
			return a
		}
		return inRange(aConform{typ: a.typ, lo: a.hi.Neg(), hi: a.lo.Neg()})
	case "op":
		return conformOperator(n.text, conform(n.args[0]), conform(n.args[1]))
	case "call":
		return conformCall(n)
	}
	return cAmbiguous("", "a node the oracle does not know: "+n.kind)
}

// undecidedOf is the first argument that is undecided, or null, which makes
// the operation so.
func undecidedOf(typ string, args ...aConform) (aConform, bool) {
	for _, a := range args {
		if a.ambiguous != "" {
			return cAmbiguous(typ, a.ambiguous), true
		}
	}
	for _, a := range args {
		if a.null {
			return cNull(typ), true
		}
	}
	return aConform{}, false
}

func conformOperator(op string, a, b aConform) aConform {
	typ := numericType(a.typ, b.typ)
	if op == "/" {
		typ = "Decimal"
	}
	if op == "^" {
		return conformPower(a, b)
	}
	if c, done := undecidedOf(typ, a, b); done {
		return c
	}
	switch op {
	case "+":
		return inRange(widen(aConform{typ: typ, lo: a.lo.Add(b.lo), hi: a.hi.Add(b.hi)}))
	case "-":
		return inRange(widen(aConform{typ: typ, lo: a.lo.Sub(b.hi), hi: a.hi.Sub(b.lo)}))
	case "*":
		return inRange(widen(span(typ, a.lo.Mul(b.lo), a.lo.Mul(b.hi), a.hi.Mul(b.lo), a.hi.Mul(b.hi))))
	}
	// `/`, `div` and `mod` are null on a divisor of 0.
	switch {
	case b.exact() && b.lo.IsZero():
		return cNull(typ)
	case b.lo.Sign() <= 0 && b.hi.Sign() >= 0:
		return cAmbiguous(typ, "a divisor that may be 0")
	}
	q := span("Decimal", quotient(a.lo, b.lo), quotient(a.lo, b.hi), quotient(a.hi, b.lo), quotient(a.hi, b.hi))
	if op == "/" {
		return inRange(widen(q))
	}
	// div truncates toward zero; mod is what it leaves, of the dividend's
	// sign.
	whole := q.lo.Truncate(0)
	if !whole.Equal(q.hi.Truncate(0)) {
		return cAmbiguous(typ, "a truncated quotient at a whole number")
	}
	if op == "div" {
		return inRange(aConform{typ: typ, lo: whole, hi: whole})
	}
	return inRange(widen(aConform{typ: typ,
		lo: decimal.Min(a.lo.Sub(b.lo.Mul(whole)), a.lo.Sub(b.hi.Mul(whole))),
		hi: decimal.Max(a.hi.Sub(b.lo.Mul(whole)), a.hi.Sub(b.hi.Mul(whole)))}))
}

// conformPower is a ^ b and Power(a, b): a whole-number exponent multiplies,
// on a negative base too; any other exponent is exp(b ln a), of a positive
// base.
func conformPower(a, b aConform) aConform {
	typ := numericType(a.typ, b.typ)
	if c, done := undecidedOf(typ, a, b); done {
		return c
	}
	if b.exact() && b.lo.Equal(b.lo.Truncate(0)) {
		n := b.lo.IntPart()
		if n < 0 {
			typ = "Decimal" // Power(10, -7) is 0.0000001 in the conformance suite
		}
		if a.lo.Sign() <= 0 && a.hi.Sign() >= 0 && n < 0 {
			if a.exact() {
				return cNull(typ)
			}
			return cAmbiguous(typ, "a negative power of a base that may be 0")
		}
		power := func(d decimal.Decimal) decimal.Decimal {
			p := cappedPower(d, n)
			if n < 0 {
				return quotient(decimal.NewFromInt(1), p)
			}
			return p
		}
		c := span(typ, power(a.lo), power(a.hi))
		if n%2 == 0 && n > 0 && a.lo.Sign() < 0 && a.hi.Sign() > 0 {
			c.lo = decimal.Zero
		}
		return inRange(widen(c))
	}
	typ = "Decimal"
	switch {
	case a.hi.Sign() < 0:
		return cNull(typ) // a negative base to a fraction is no real number
	case a.lo.Sign() <= 0:
		return cAmbiguous(typ, "a fractional power of a base that may be 0 or negative")
	}
	lnLo, lnHi := mustLn(a.lo), mustLn(a.hi)
	e := span(typ, lnLo.Mul(b.lo), lnLo.Mul(b.hi), lnHi.Mul(b.lo), lnHi.Mul(b.hi))
	return conformExp(e)
}

func mustLn(d decimal.Decimal) decimal.Decimal {
	v, err := d.Ln(precision)
	if err != nil {
		panic(fmt.Sprintf("ln %s: %v", d, err))
	}
	return v
}

// exp is e to a power, or past the Decimal range a value beyond it.
func exp(d decimal.Decimal) decimal.Decimal {
	if d.GreaterThan(decimal.NewFromInt(60)) {
		return decimalMax.Mul(decimal.NewFromInt(10))
	}
	if d.LessThan(decimal.NewFromInt(-60)) {
		return decimal.New(1, -60) // below every step of the scale, and above 0
	}
	v, err := d.ExpTaylor(precision)
	if err != nil {
		panic(fmt.Sprintf("exp %s: %v", d, err))
	}
	return v
}

func conformExp(e aConform) aConform {
	return inRange(widen(aConform{typ: "Decimal", lo: exp(e.lo), hi: exp(e.hi)}))
}

// arithmeticFunctions is the other functions of the corpus.
var arithmeticFunctions = []string{"Abs", "Ln", "Exp", "Round", "Truncate", "Ceiling", "Floor", "Log", "Power"}

// aggregates is the aggregate functions of the corpus.
var aggregates = []string{"Avg", "Sum", "Product", "Median", "Variance", "StdDev",
	"PopulationVariance", "PopulationStdDev", "Max", "Min"}

func conformCall(n *aNode) aConform {
	if slices.Contains(aggregates, n.text) {
		return conformAggregate(n.text, n.args[0])
	}
	args := make([]aConform, len(n.args))
	for i, arg := range n.args {
		args[i] = conform(arg)
	}
	if n.text == "Round" && len(args) > 1 && args[1].null {
		args = args[:1] // "If precision is not specified or null, 0 is assumed"
	}
	a := args[0]
	switch n.text {
	case "Power":
		return conformPower(a, args[1])
	case "Log":
		if c, done := undecidedOf("Decimal", args...); done {
			return c
		}
		base := args[1]
		switch {
		case a.hi.Sign() <= 0 || base.hi.Sign() <= 0:
			return cNull("Decimal") // no logarithm of, nor in a base of, 0 or less
		case a.lo.Sign() <= 0 || base.lo.Sign() <= 0:
			return cAmbiguous("Decimal", "a logarithm of, or in a base of, a value that may be 0 or less")
		}
		one := decimal.NewFromInt(1)
		switch {
		case base.exact() && base.lo.Equal(one):
			return cNull("Decimal") // in base 1, a division by ln 1 = 0
		case base.lo.LessThanOrEqual(one) && base.hi.GreaterThanOrEqual(one):
			return cAmbiguous("Decimal", "a logarithm in a base that may be 1")
		}
		num, den := span("Decimal", mustLn(a.lo), mustLn(a.hi)), span("Decimal", mustLn(base.lo), mustLn(base.hi))
		return inRange(widen(span("Decimal", quotient(num.lo, den.lo), quotient(num.lo, den.hi),
			quotient(num.hi, den.lo), quotient(num.hi, den.hi))))
	}
	typ := map[string]string{"Abs": a.typ, "Ln": "Decimal", "Exp": "Decimal", "Round": "Decimal",
		"Truncate": "Integer", "Ceiling": "Integer", "Floor": "Integer"}[n.text]
	if c, done := undecidedOf(typ, args...); done {
		return c
	}
	switch n.text {
	case "Abs":
		switch {
		case a.lo.Sign() >= 0:
			return a
		case a.hi.Sign() <= 0:
			return inRange(aConform{typ: typ, lo: a.hi.Neg(), hi: a.lo.Neg()})
		}
		return inRange(aConform{typ: typ, lo: decimal.Zero, hi: decimal.Max(a.lo.Abs(), a.hi)})
	case "Ln":
		switch {
		case a.hi.Sign() <= 0:
			return cNull(typ)
		case a.lo.Sign() <= 0:
			return cAmbiguous(typ, "a logarithm of a value that may be 0 or less")
		}
		return inRange(widen(aConform{typ: typ, lo: mustLn(a.lo), hi: mustLn(a.hi)}))
	case "Exp":
		return conformExp(a)
	}
	// The functions to a whole number, or to a number of places: a step
	// within the interval is undecided.
	step := func(d decimal.Decimal) decimal.Decimal {
		switch n.text {
		case "Truncate":
			return d.Truncate(0)
		case "Floor":
			return d.Floor()
		case "Ceiling":
			return d.Ceil()
		}
		places := int32(0)
		if len(args) > 1 {
			places = int32(args[1].lo.IntPart())
		}
		return roundTraditional(d, places)
	}
	lo, hi := step(a.lo), step(a.hi)
	if !lo.Equal(hi) {
		return cAmbiguous(typ, n.text+" of a value at a step")
	}
	return inRange(aConform{typ: typ, lo: lo, hi: lo})
}

// roundTraditional is Round (Appendix B): to the nearest, a half away from
// zero, 0.5 to 1 and -0.5 to -1.
func roundTraditional(d decimal.Decimal, places int32) decimal.Decimal {
	shift := decimal.New(1, places)
	half := decimal.RequireFromString("0.5")
	m := d.Abs().Mul(shift).Add(half).Floor()
	if d.Sign() < 0 {
		m = m.Neg()
	}
	return m.Div(shift)
}

func conformAggregate(name string, list *aNode) aConform {
	var items []aConform
	typ := "Integer"
	for _, e := range list.args {
		c := conform(e)
		typ = numericType(typ, c.typ)
		if c.ambiguous != "" {
			return cAmbiguous(typ, c.ambiguous)
		}
		if c.null {
			continue
		}
		items = append(items, c)
	}
	switch name {
	case "Avg", "Median", "Variance", "StdDev", "PopulationVariance", "PopulationStdDev":
		typ = "Decimal"
	}
	if len(items) == 0 {
		return cNull(typ)
	}
	// The list is of its common type, and an aggregate over Decimals takes
	// its Integers converted: Avg({2000000000, 2000000000}) adds Decimals.
	for i := range items {
		items[i].typ = typ
	}
	// Each is computed by the operations an implementation computes it with,
	// each rounded as they may be: the interval holds their errors, step by
	// step.
	chain := func(op string, cs []aConform) aConform {
		acc := cs[0]
		for _, c := range cs[1:] {
			if acc = conformOperator(op, acc, c); acc.null || acc.ambiguous != "" {
				return acc
			}
		}
		return acc
	}
	count := func(k int) aConform {
		d := decimal.NewFromInt(int64(k))
		return aConform{typ: "Integer", lo: d, hi: d}
	}
	sum := chain("+", items)
	sum.typ = numericType(typ, sum.typ)
	mean := conformOperator("/", sum, count(len(items)))
	switch name {
	case "Sum":
		return sum
	case "Avg":
		return mean
	case "Product":
		p := chain("*", items)
		p.typ = numericType(typ, p.typ)
		return p
	case "Max", "Min":
		pick := decimal.Max
		if name == "Min" {
			pick = decimal.Min
		}
		lo, hi := items[0].lo, items[0].hi
		for _, c := range items[1:] {
			lo, hi = pick(lo, c.lo), pick(hi, c.hi)
		}
		return aConform{typ: typ, lo: lo, hi: hi}
	case "Median":
		// The median is monotone in each element: of the lower ends, and of
		// the upper ones.
		lows, highs := make([]aConform, len(items)), make([]aConform, len(items))
		for i, c := range items {
			lows[i] = aConform{typ: c.typ, lo: c.lo, hi: c.lo}
			highs[i] = aConform{typ: c.typ, lo: c.hi, hi: c.hi}
		}
		median := func(cs []aConform) aConform {
			slices.SortFunc(cs, func(a, b aConform) int { return a.lo.Cmp(b.lo) })
			if len(cs)%2 == 1 {
				m := cs[len(cs)/2]
				m.typ = typ
				return m
			}
			return conformOperator("/", conformOperator("+", cs[len(cs)/2-1], cs[len(cs)/2]), count(2))
		}
		lo, hi := median(lows), median(highs)
		switch {
		case lo.ambiguous != "" || hi.ambiguous != "":
			return cAmbiguous(typ, strings.TrimPrefix(lo.ambiguous+"; "+hi.ambiguous, "; "))
		case lo.null && hi.null:
			return cNull(typ)
		case lo.null || hi.null:
			return cAmbiguous(typ, "a median at the edge of the Decimal range")
		}
		return aConform{typ: typ, lo: lo.lo, hi: hi.hi}
	}
	// The variances, of the sample (n - 1) or of the population (n): the
	// examples of Appendix B, {1, 2, 3, 4, 5} with a Variance of 2.5 and a
	// PopulationVariance of 2.0, give the denominators. A sample of one
	// divides by 0, which cannot be performed.
	den := count(len(items))
	if name == "Variance" || name == "StdDev" {
		if len(items) == 1 {
			return cNull(typ)
		}
		den = count(len(items) - 1)
	}
	squares := make([]aConform, len(items))
	for i, c := range items {
		squares[i] = conformPower(conformOperator("-", c, mean), count(2))
	}
	v := conformOperator("/", chain("+", squares), den)
	if v.null || v.ambiguous != "" || name == "Variance" || name == "PopulationVariance" {
		return v
	}
	return inRange(widen(aConform{typ: typ, lo: sqrt(v.lo), hi: sqrt(v.hi)}))
}

func sqrt(d decimal.Decimal) decimal.Decimal {
	if d.Sign() <= 0 {
		return decimal.Zero
	}
	f := new(big.Float).SetPrec(200)
	f.SetString(d.String())
	f.Sqrt(f)
	return decimal.RequireFromString(f.Text('f', precision))
}

// aPolicy is what the scale policy of docs/adr/002-escala-decimal.md makes
// this engine answer: a typed value or null. unscaled marks a float of Ln or
// Exp the policy does not bring to the scale, compared within a relative
// 1e-13 as the snapshot compares it. Every other answer is compared exactly:
// a float floored at the scale whose last digit another platform's libm gives
// otherwise fails there, in view, rather than being accepted.
type aPolicy struct {
	typ      string
	null     bool
	v        decimal.Decimal
	unscaled bool
}

func (p aPolicy) String() string {
	if p.null {
		return "null"
	}
	if p.unscaled {
		// A float's last digits differ by platform: the text a verdict
		// records keeps the 13 digits every platform gives.
		f, _ := p.v.Float64()
		return p.typ + " about " + strconv.FormatFloat(f, 'g', 13, 64)
	}
	return p.typ + " " + p.v.String()
}

func pNull(typ string) aPolicy { return aPolicy{typ: typ, null: true} }

// floor8 brings a Decimal to the scale, toward negative infinity (ADR 002):
// a value within it is kept as it is.
func floor8(d decimal.Decimal) decimal.Decimal {
	if d.Exponent() >= -8 {
		return d
	}
	return d.RoundFloor(8)
}

// fromFloat is a binary64 result read as the shortest decimal that prints as
// it (ADR 002), or null when it is no number.
func fromFloat(typ string, f float64) aPolicy {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return pNull(typ)
	}
	return aPolicy{typ: typ, v: decimal.RequireFromString(strconv.FormatFloat(f, 'g', -1, 64))}
}

func float(d decimal.Decimal) float64 { f, _ := d.Float64(); return f }

// pRange is null past the type's range.
func pRange(p aPolicy) aPolicy {
	if p.null {
		return p
	}
	if p.typ == "Integer" && (p.v.LessThan(integerMin) || p.v.GreaterThan(integerMax)) ||
		p.typ == "Decimal" && p.v.Abs().GreaterThan(decimalMax) {
		return pNull(p.typ)
	}
	return p
}

// exactQuotient is a/b floored at the scale, exactly.
func exactQuotient(a, b decimal.Decimal) decimal.Decimal {
	r := new(big.Rat).Quo(a.Rat(), b.Rat())
	r.Mul(r, new(big.Rat).SetInt64(100000000))
	q := new(big.Int).Quo(r.Num(), r.Denom())
	if r.Sign() < 0 && new(big.Int).Mul(q, r.Denom()).Cmp(r.Num()) != 0 {
		q.Sub(q, big.NewInt(1))
	}
	return decimal.NewFromBigInt(q, -8)
}

// truncatedQuotient is a/b toward zero, exactly.
func truncatedQuotient(a, b decimal.Decimal) decimal.Decimal {
	r := new(big.Rat).Quo(a.Rat(), b.Rat())
	return decimal.NewFromBigInt(new(big.Int).Quo(r.Num(), r.Denom()), 0)
}

// policy is what the scale policy makes this engine answer for a node.
func policy(n *aNode) aPolicy {
	switch n.kind {
	case "Integer", "Decimal":
		return aPolicy{typ: n.kind, v: decimal.RequireFromString(n.text)}
	case "null":
		return pNull(n.text)
	case "neg":
		a := policy(n.args[0])
		if a.null {
			return a
		}
		a.v = a.v.Neg() // keeps a being unscaled
		return pRange(a)
	case "op":
		return policyOperator(n.text, policy(n.args[0]), policy(n.args[1]))
	case "call":
		return policyCall(n)
	}
	panic("a node the oracle does not know: " + n.kind)
}

func policyOperator(op string, a, b aPolicy) aPolicy {
	typ := numericType(a.typ, b.typ)
	if op == "/" {
		typ = "Decimal"
	}
	if op == "^" {
		return policyPower(a, b)
	}
	if a.null || b.null {
		return pNull(typ)
	}
	scale := func(d decimal.Decimal) aPolicy {
		if typ == "Decimal" {
			d = floor8(d)
		}
		return pRange(aPolicy{typ: typ, v: d})
	}
	switch op {
	case "+":
		return scale(a.v.Add(b.v))
	case "-":
		// A difference is not brought to the scale, and keeps an unscaled
		// operand's float.
		return pRange(aPolicy{typ: typ, v: a.v.Sub(b.v), unscaled: a.unscaled || b.unscaled})
	case "*":
		return scale(a.v.Mul(b.v))
	}
	if b.v.IsZero() {
		return pNull(typ)
	}
	switch op {
	case "/":
		return pRange(aPolicy{typ: typ, v: exactQuotient(a.v, b.v)})
	case "div":
		return pRange(aPolicy{typ: typ, v: truncatedQuotient(a.v, b.v)})
	}
	return scale(a.v.Sub(b.v.Mul(truncatedQuotient(a.v, b.v))))
}

// policyPower is a power: of a whole-number exponent, computed in decimal
// and brought to the scale; of any other, through a binary64.
func policyPower(a, b aPolicy) aPolicy {
	typ := numericType(a.typ, b.typ)
	if a.null || b.null {
		return pNull(typ)
	}
	if b.v.Equal(b.v.Truncate(0)) {
		n := b.v.IntPart()
		if n < 0 {
			typ = "Decimal"
			if a.v.IsZero() {
				return pNull(typ)
			}
		}
		p := cappedPower(a.v, n)
		if n < 0 {
			return pRange(aPolicy{typ: typ, v: exactQuotient(decimal.NewFromInt(1), p)})
		}
		if typ == "Decimal" {
			p = floor8(p)
		}
		return pRange(aPolicy{typ: typ, v: p})
	}
	p := fromFloat("Decimal", math.Pow(float(a.v), float(b.v)))
	if !p.null {
		p.v = floor8(p.v)
	}
	return pRange(p)
}

func policyCall(n *aNode) aPolicy {
	if slices.Contains(aggregates, n.text) {
		return policyAggregate(n.text, n.args[0])
	}
	args := make([]aPolicy, len(n.args))
	for i, arg := range n.args {
		args[i] = policy(arg)
	}
	if n.text == "Round" && len(args) > 1 && args[1].null {
		args = args[:1]
	}
	a := args[0]
	if n.text == "Power" {
		return policyPower(a, args[1])
	}
	typ := map[string]string{"Abs": a.typ, "Ln": "Decimal", "Exp": "Decimal", "Round": "Decimal", "Log": "Decimal",
		"Truncate": "Integer", "Ceiling": "Integer", "Floor": "Integer"}[n.text]
	for _, arg := range args {
		if arg.null {
			return pNull(typ)
		}
	}
	switch n.text {
	case "Abs":
		return pRange(aPolicy{typ: typ, v: a.v.Abs(), unscaled: a.unscaled})
	case "Ln":
		if a.v.Sign() <= 0 {
			return pNull(typ)
		}
		p := fromFloat(typ, math.Log(float(a.v)))
		p.unscaled = true // Ln is not brought to the scale (ADR 002)
		return pRange(p)
	case "Exp":
		p := fromFloat(typ, math.Exp(float(a.v)))
		p.unscaled = true // nor is Exp
		return pRange(p)
	case "Log":
		base := args[1]
		if a.v.Sign() <= 0 || base.v.Sign() <= 0 || base.v.Equal(decimal.NewFromInt(1)) {
			return pNull(typ)
		}
		// An exact power has a whole-number logarithm, checked in decimal.
		k := math.Round(math.Log(float(a.v)) / math.Log(float(base.v)))
		if math.Abs(k) < 1000 {
			p := decimal.NewFromInt(1)
			for range int(math.Abs(k)) {
				p = p.Mul(base.v)
			}
			if k < 0 && !p.IsZero() {
				p = decimal.NewFromInt(1).DivRound(p, precision)
			}
			if p.Equal(a.v) {
				return aPolicy{typ: typ, v: decimal.NewFromFloat(k)}
			}
		}
		p := fromFloat(typ, math.Log(float(a.v))/math.Log(float(base.v)))
		if !p.null {
			p.v = floor8(p.v)
		}
		return pRange(p)
	case "Truncate":
		return pRange(aPolicy{typ: typ, v: a.v.Truncate(0)})
	case "Floor":
		return pRange(aPolicy{typ: typ, v: a.v.Floor()})
	case "Ceiling":
		return pRange(aPolicy{typ: typ, v: a.v.Ceil()})
	}
	places := int32(0)
	if len(args) > 1 {
		places = int32(args[1].v.IntPart())
	}
	return pRange(aPolicy{typ: typ, v: roundTraditional(a.v, places)})
}

func policyAggregate(name string, list *aNode) aPolicy {
	var items []aPolicy
	typ := "Integer"
	for _, e := range list.args {
		p := policy(e)
		typ = numericType(typ, p.typ)
		if !p.null {
			items = append(items, p)
		}
	}
	switch name {
	case "Avg", "Median", "Variance", "StdDev", "PopulationVariance", "PopulationStdDev":
		typ = "Decimal"
	}
	if len(items) == 0 {
		return pNull(typ)
	}
	add := func(op string, ps []aPolicy) aPolicy {
		acc := ps[0]
		acc.typ = numericType(typ, acc.typ)
		for _, p := range ps[1:] {
			acc = policyOperator(op, acc, p)
			if acc.null {
				return acc
			}
		}
		return acc
	}
	sum := add("+", items)
	count := aPolicy{typ: "Integer", v: decimal.NewFromInt(int64(len(items)))}
	mean := policyOperator("/", sum, count)
	switch name {
	case "Sum":
		return sum
	case "Product":
		return add("*", items)
	case "Avg":
		return mean
	case "Max", "Min":
		best := items[0]
		for _, p := range items[1:] {
			if name == "Max" && p.v.GreaterThan(best.v) || name == "Min" && p.v.LessThan(best.v) {
				best = p
			}
		}
		best.typ = typ
		return best
	case "Median":
		s := slices.Clone(items)
		slices.SortStableFunc(s, func(a, b aPolicy) int { return a.v.Cmp(b.v) })
		if len(s)%2 == 1 {
			m := s[len(s)/2]
			m.typ = typ
			return m
		}
		// The median is of Decimals: the central Integers are converted
		// before they are added.
		a, b := s[len(s)/2-1], s[len(s)/2]
		a.typ, b.typ = "Decimal", "Decimal"
		return policyOperator("/", policyOperator("+", a, b), aPolicy{typ: "Integer", v: decimal.NewFromInt(2)})
	}
	if mean.null {
		return mean
	}
	squares := make([]aPolicy, len(items))
	for i, p := range items {
		dev := policyOperator("-", p, mean)
		squares[i] = policyOperator("*", dev, dev)
	}
	den := count
	if name == "Variance" || name == "StdDev" {
		den.v = den.v.Sub(decimal.NewFromInt(1))
	}
	v := policyOperator("/", add("+", squares), den)
	if v.null || name == "Variance" || name == "PopulationVariance" {
		return v
	}
	p := fromFloat("Decimal", math.Sqrt(float(v.v)))
	if !p.null {
		p.v = floor8(p.v)
	}
	return p
}

// aAnswer is an answer read as the oracle compares them: a typed value,
// null, an error, or one it cannot read.
type aAnswer struct {
	kind string // "value", "null", "error", "nan", "unreadable"
	typ  string
	v    decimal.Decimal
	text string
}

func readReferenceArithmetic(raw json.RawMessage) aAnswer {
	if len(raw) == 0 || string(raw) == "null" {
		return aAnswer{kind: "null", text: "null"}
	}
	var r struct {
		Type, Value, Error, Message string
	}
	if json.Unmarshal(raw, &r) != nil {
		return aAnswer{kind: "unreadable", text: string(raw)}
	}
	switch {
	case r.Error != "":
		return aAnswer{kind: "error", text: "error: " + r.Message}
	case r.Value == "NaN" || strings.Contains(r.Value, "Infinity"):
		return aAnswer{kind: "nan", typ: r.Type, text: r.Type + " " + r.Value}
	case r.Type != "Integer" && r.Type != "Decimal":
		return aAnswer{kind: "unreadable", text: string(raw)}
	}
	v, err := decimal.NewFromString(r.Value)
	if err != nil {
		return aAnswer{kind: "unreadable", text: string(raw)}
	}
	return aAnswer{kind: "value", typ: r.Type, v: v, text: r.Type + " " + r.Value}
}

// readEngineArithmetic reads this engine's answer as the snapshot pins it.
func readEngineArithmetic(s string) aAnswer {
	switch {
	case s == "null":
		return aAnswer{kind: "null", text: s}
	case strings.HasPrefix(s, "error"):
		return aAnswer{kind: "error", text: s}
	}
	typ, value, _ := strings.Cut(s, " ")
	v, err := decimal.NewFromString(value)
	if err != nil || typ != "Integer" && typ != "Decimal" {
		return aAnswer{kind: "unreadable", text: s}
	}
	return aAnswer{kind: "value", typ: typ, v: v, text: s}
}

// followsPolicy reports whether an answer is the policy's.
func followsPolicy(a aAnswer, p aPolicy) bool {
	if p.null {
		return a.kind == "null"
	}
	if a.kind != "value" || a.typ != p.typ {
		return false
	}
	if p.unscaled {
		return a.v.Sub(p.v).Abs().LessThanOrEqual(p.v.Abs().Mul(decimal.RequireFromString("1e-13")))
	}
	return a.v.Equal(p.v)
}

// follows reports whether an answer is the policy's. An unscaled float is
// within a tolerance of it, and that tolerance does not reach past what the
// specification allows: Exp(0) is 1, not a float a digit off.
func follows(a aAnswer, v arithmeticVerdict) bool {
	if !followsPolicy(a, v.policy) {
		return false
	}
	return !v.policy.unscaled || v.conform.ambiguous != "" || isConformant(a, v.conform)
}

// isConformant reports whether an answer is one the specification allows.
func isConformant(a aAnswer, c aConform) bool {
	if c.null {
		return a.kind == "null"
	}
	return a.kind == "value" && a.typ == c.typ && a.v.GreaterThanOrEqual(c.lo) && a.v.LessThanOrEqual(c.hi)
}

func (c aConform) String() string {
	switch {
	case c.null:
		return "null"
	case c.exact():
		return c.typ + " " + c.lo.String()
	}
	return fmt.Sprintf("%s within [%s, %s]", c.typ, c.lo.Round(12), c.hi.Round(12))
}

// arithmeticDefinitions is the definitions of testdata/snapshots/arithmetic.cql.
func arithmeticDefinitions(t *testing.T) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", "arithmetic.cql"))
	if err != nil {
		t.Fatal(err)
	}
	defs := map[string]string{}
	for _, m := range regexp.MustCompile(`(?m)^define (D\d+): (.*)$`).FindAllStringSubmatch(string(raw), -1) {
		defs[m[1]] = m[2]
	}
	return defs
}

// arithmeticEngineAnswers is this engine's answers as the snapshot pins them;
// TestAnswersMatchTheSnapshots holds the snapshot to the engine.
func arithmeticEngineAnswers(t *testing.T) map[string]string {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "snapshots", "arithmetic.answers.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	out := map[string]string{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		if name, answer, ok := strings.Cut(s.Text(), "\t"); ok && !strings.HasPrefix(name, "#") {
			out[name] = answer
		}
	}
	if err := s.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// arithmeticVerdict is the oracle's reading of a definition.
type arithmeticVerdict struct {
	conform aConform
	policy  aPolicy
}

func arithmeticOracle(expr string) (arithmeticVerdict, error) {
	n, err := parseArithmetic(expr)
	if err != nil {
		return arithmeticVerdict{}, err
	}
	return arithmeticVerdict{conform: conform(n), policy: policy(n)}, nil
}

// arithmeticCorpusSize is the number of definitions of arithmetic.cql.
const arithmeticCorpusSize = 800

// TestTheArithmeticOracleReadsTheWholeCorpus holds the oracle to every
// definition of the corpus.
func TestTheArithmeticOracleReadsTheWholeCorpus(t *testing.T) {
	defs := arithmeticDefinitions(t)
	if len(defs) != arithmeticCorpusSize {
		t.Fatalf("the oracle reads %d definitions of arithmetic.cql; the corpus has %d", len(defs), arithmeticCorpusSize)
	}
	for _, name := range slices.Sorted(maps.Keys(defs)) {
		if _, err := arithmeticOracle(defs[name]); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestTheScalePolicyIsConformant holds the policy of ADR 002 to the
// specification: on every definition it decides, the policy's answer is one
// the specification allows.
func TestTheScalePolicyIsConformant(t *testing.T) {
	defs := arithmeticDefinitions(t)
	for _, name := range slices.Sorted(maps.Keys(defs)) {
		v, err := arithmeticOracle(defs[name])
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if v.conform.ambiguous != "" {
			continue
		}
		p := aAnswer{kind: "value", typ: v.policy.typ, v: v.policy.v}
		if v.policy.null {
			p = aAnswer{kind: "null"}
		}
		if !isConformant(p, v.conform) {
			t.Errorf("%s: the policy answers %s; the specification allows %s (%s)", name, v.policy, v.conform, defs[name])
		}
	}
}

// TestTheEngineFollowsTheScalePolicy holds this engine, as it answers now, to
// the policy on every definition no divergence reaches, where the comparison
// with the references cannot: an answer within its tolerance of theirs and
// off the policy is pinned here, with the policy's answer. The verdicts read
// the snapshot, which tolerates a float's last digits: on every definition,
// this engine follows the policy now exactly where the snapshot does.
func TestTheEngineFollowsTheScalePolicy(t *testing.T) {
	defs := arithmeticDefinitions(t)
	recorded := arithmeticEngineAnswers(t)
	engine := snapshotAnswers(t, "arithmetic")
	divergences := readDivergences(t, filepath.Join("testdata", "snapshots", "arithmetic.divergences.tsv"))
	off := map[string]string{}
	for _, name := range slices.Sorted(maps.Keys(defs)) {
		v, err := arithmeticOracle(defs[name])
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		now := follows(readEngineArithmetic(engine[name]), v)
		if now != follows(readEngineArithmetic(recorded[name]), v) {
			t.Errorf("%s: this engine answers %s and the snapshot %s; the policy gives %s", name, engine[name], recorded[name], v.policy)
		}
		_, javaDiffers := divergences[name+"\tjava"]
		_, jsDiffers := divergences[name+"\tjs"]
		if !now && !javaDiffers && !jsDiffers {
			off[name] = engine[name] + " | policy " + v.policy.String()
		}
	}
	if !maps.Equal(off, arithmeticOffThePolicy) {
		for _, name := range slices.Sorted(maps.Keys(off)) {
			if arithmeticOffThePolicy[name] != off[name] {
				t.Errorf("%s: this engine answers %s (%s)", name, off[name], defs[name])
			}
		}
		for _, name := range slices.Sorted(maps.Keys(arithmeticOffThePolicy)) {
			if _, still := off[name]; !still {
				t.Errorf("%s: pinned off the policy, and it no longer is", name)
			}
		}
	}
}

// arithmeticOffThePolicy is the definitions no divergence reaches where this
// engine is off the policy, each with its answer and the policy's: own
// defects the comparison does not see.
var arithmeticOffThePolicy = map[string]string{}

// classifiedByArithmeticOracle is the verdict the oracle gives a divergence.
func classifiedByArithmeticOracle(key string, d divergence, v arithmeticVerdict, engine, reference aAnswer) divergence {
	name, ref, _ := strings.Cut(key, "\t")
	if v.conform.ambiguous != "" {
		// Undecided between branches, an answer no branch gives is still
		// wrong: this engine off its policy, a run-time error or a NaN, which
		// arithmetic never gives ("null, rather than a run-time error"), or a
		// value of another type. Which value a branch gives is not decided
		// here.
		why := "the specification does not decide between its branches (" + v.conform.ambiguous + "), and none gives "
		switch {
		case !follows(engine, v):
			d.Class, d.Expected = ownDefect, v.policy.String()
			d.Evidence = "arithmetic oracle (engine_arithmetic_oracle_test.go): the scale policy gives " + v.policy.String() +
				"; " + arithmeticOracleRule + "; this engine answers " + engine.text
			return d
		case reference.kind == "error" || reference.kind == "nan" ||
			reference.kind == "value" && reference.typ != v.policy.typ:
			d.Class, d.Expected = referenceDefect, d.Engine
			d.Evidence = "arithmetic oracle (engine_arithmetic_oracle_test.go): " + why + reference.text +
				"; " + arithmeticOracleRule + "; the reference answers " + d.Reference
			return d
		}
		d.Class, d.Expected = undecided, ""
		d.Evidence = fmt.Sprintf("reading 1: %s, as the reference answers [source: testdata/snapshots/arithmetic.%s.json] | "+
			"reading 2: %s, as this engine answers [source: testdata/snapshots/arithmetic.answers.tsv]; "+
			"the arithmetic oracle (engine_arithmetic_oracle_test.go) does not decide: %s; %s",
			d.Reference, ref, d.Engine, v.conform.ambiguous, arithmeticOracleRule)
		return d
	}
	allowed := "the specification allows " + v.conform.String() + " and the scale policy gives " + v.policy.String()
	evidence := "arithmetic oracle (engine_arithmetic_oracle_test.go): " + allowed + "; " + arithmeticOracleRule
	if !follows(engine, v) {
		d.Class, d.Expected, d.Evidence = ownDefect, v.policy.String(), evidence+"; this engine answers "+engine.text
		return d
	}
	switch {
	case reference.kind == "unreadable":
		d.Class, d.Expected = unclassified, ""
		d.Evidence = "the reference harness cannot read the reference's answer; " + allowed
	case isConformant(reference, v.conform):
		d.Class, d.Expected = deliberate, d.Engine
		d.Evidence = "both answers are conformant, and this engine keeps the Decimal scale it chose, decided in " +
			arithmeticDecisionPlan + " (PR 4, decision 1): " + allowed + " (" + name + "); " + arithmeticOracleRule
	default:
		d.Class, d.Expected, d.Evidence = referenceDefect, d.Engine, evidence+"; the reference answers "+d.Reference
	}
	return d
}

// arithmeticDecisionPlan is where the user's decisions about the area are
// recorded, with the definitions they cover.
const arithmeticDecisionPlan = "docs/plans/003-clasificar-divergencias.md"

// TestTheArithmeticVerdictsFollowTheOracle holds every verdict of the
// arithmetic area to what the oracle computes. -classify-arithmetic rewrites
// them.
func TestTheArithmeticVerdictsFollowTheOracle(t *testing.T) {
	defs := arithmeticDefinitions(t)
	engine := arithmeticEngineAnswers(t)
	refs := map[string]map[string]json.RawMessage{
		"java": readReferenceResults(t, "arithmetic", "java"),
		"js":   readReferenceResults(t, "arithmetic", "js"),
	}
	path := filepath.Join("testdata", "snapshots", "arithmetic.divergences.tsv")
	recorded := readDivergences(t, path)
	classify := func(key string, d divergence) divergence {
		name, ref, _ := strings.Cut(key, "\t")
		v, err := arithmeticOracle(defs[name])
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		raw, answered := refs[ref][name]
		if !answered {
			t.Fatalf("%s: %s gives no answer", name, ref)
		}
		return classifiedByArithmeticOracle(key, d, v, readEngineArithmetic(engine[name]), readReferenceArithmetic(raw))
	}
	if *classifyArithmetic {
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

// TestTheArithmeticOracleAllowsWhatAllAgreeOn holds the oracle to the
// definitions where this engine and the references give one answer: each
// answer the specification is held to is one it allows.
func TestTheArithmeticOracleAllowsWhatAllAgreeOn(t *testing.T) {
	defs := arithmeticDefinitions(t)
	recorded := readDivergences(t, filepath.Join("testdata", "snapshots", "arithmetic.divergences.tsv"))
	refs := map[string]map[string]json.RawMessage{
		"java": readReferenceResults(t, "arithmetic", "java"),
		"js":   readReferenceResults(t, "arithmetic", "js"),
	}
	checked := 0
	for _, name := range slices.Sorted(maps.Keys(defs)) {
		v, err := arithmeticOracle(defs[name])
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if v.conform.ambiguous != "" {
			continue
		}
		for _, ref := range []string{"java", "js"} {
			raw, answered := refs[ref][name]
			if _, differs := recorded[name+"\t"+ref]; differs || !answered {
				continue
			}
			a := readReferenceArithmetic(raw)
			if a.kind == "unreadable" {
				continue
			}
			checked++
			if why := arithmeticAgreedOutsideTheSpecification[name+" "+ref]; why != "" {
				if _, told := recorded[name+"\tjs"]; !told || isConformant(a, v.conform) {
					t.Errorf("%s: %s is no exception any longer (%s)", name, ref, why)
				}
				continue
			}
			if !isConformant(a, v.conform) {
				t.Errorf("%s: %s answers %s, as this engine does; the specification allows %s (%s)",
					name, ref, a.text, v.conform, defs[name])
			}
		}
	}
	if checked == 0 {
		t.Fatal("no definition where all agree: the oracle is checked on nothing")
	}
}

// arithmeticAgreedOutsideTheSpecification is the answers a reference gives
// as this engine does that the specification does not allow, with why. Each
// holds only while the other reference tells this engine apart.
var arithmeticAgreedOutsideTheSpecification = map[string]string{
	"D114 java": failsOnInfinity,
	"D115 java": failsOnInfinity,
	"D17 java":  failsOnInfinity,
	"D212 java": failsOnInfinity,
	"D7 java":   failsOnInfinity,
	"D379 java": pastTheRange,
	"D427 java": pastTheRange,
	"D780 java": pastTheRange,
	"D573 java": failsOnInfinity,
	"D699 java": pastTheRange,
}

const (
	failsOnInfinity = "the Java engine fails on an infinity, as this engine does; the specification makes it null, and this engine's own defect is the verdict of the js row"
	pastTheRange    = "the Java engine answers past the Decimal range, as this engine does; the specification makes it null (PR 4, decision 2), and this engine's own defect is the verdict of the js row"
)

// TestTheArithmeticDecisionNamesItsRows holds the plan to the rows decision 1
// covers: it names each of them. -classify-arithmetic writes the list.
func TestTheArithmeticDecisionNamesItsRows(t *testing.T) {
	recorded := readDivergences(t, filepath.Join("testdata", "snapshots", "arithmetic.divergences.tsv"))
	var names []string
	for key, d := range recorded {
		if name, _, _ := strings.Cut(key, "\t"); d.Class == deliberate && !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	want := strings.Join(names, " ")
	raw, err := os.ReadFile(arithmeticDecisionPlan)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	const begin, end = "<!-- arithmetic decision 1, written by -classify-arithmetic: begin -->", "<!-- arithmetic decision 1: end -->"
	b, e := strings.Index(doc, begin), strings.Index(doc, end)
	if b < 0 || e < b {
		t.Fatalf("%s has no block for decision 1: %s … %s", arithmeticDecisionPlan, begin, end)
	}
	if *classifyArithmetic {
		doc = doc[:b] + begin + "\n" + want + "\n" + doc[e:]
		if err := os.WriteFile(arithmeticDecisionPlan, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if got := strings.TrimSpace(doc[b+len(begin) : e]); got != want {
		t.Errorf("%s names other rows than decision 1 covers; rewrite them with -classify-arithmetic", arithmeticDecisionPlan)
	}
}

// scalePolicyControls are the rows of ADR 002's table the corpus does not
// reach, each with the answer the table and the specification give, written
// by hand: a difference past the scale, Round of a half either way, div and
// mod of a negative dividend, and a negative quotient floored.
var scalePolicyControls = map[string]string{
	"Exp(1.0) - 0.0":                   "Decimal 2.718281828459045", // a difference is not brought to the scale
	"Ln(2.0) - 0.0":                    "Decimal 0.6931471805599453",
	"Round(-0.5)":                      "Decimal -1", // "less than or equal to -0.5 and greater than -1.0 will round to -1"
	"Round(2.5)":                       "Decimal 3",
	"Round(-2.5)":                      "Decimal -3",
	"Round(0.125, 2)":                  "Decimal 0.13",
	"(-7) div 3":                       "Integer -2", // truncated
	"(-7) mod 3":                       "Integer -1", // the remainder of the truncated division
	"(-7.5) mod 2":                     "Decimal -1.5",
	"(-2.0) / 3.0":                     "Decimal -0.66666667", // toward negative infinity
	"10.0 / 3.0":                       "Decimal 3.33333333",
	"Power(10, -7)":                    "Decimal 0.0000001", // the conformance suite
	"0.00000001 * 0.5":                 "Decimal 0",
	"Avg({1.0, 2.0, 2.0})":             "Decimal 1.66666666",
	"Round(1.5, (null as Integer))":    "Decimal 2",           // "If precision is not specified or null, 0 is assumed"
	"Avg({2000000000, 2000000000})":    "Decimal 2000000000",  // the Integers are Decimals before they are added
	"Power((-1000.0), -15)":            "Decimal -0.00000001", // -1e-45, floored
	"Power((-2.0), 2)":                 "Decimal 4",           // a whole-number power of a negative base
	"Log(1000, 10)":                    "Decimal 3",           // an exact power, checked in decimal
	"Log(8.0, 1.0)":                    "null",                // a division by ln 1 = 0, which "cannot be performed"
	"Variance({1.0})":                  "null",                // the sample variance of one divides by 0 (an inference, see conformAggregate)
	"PopulationVariance({1.0})":        "Decimal 0",
	"2147483647 + 1":                   "null", // past the 32 bits of Integer
	"Median({2000000000, 2000000000})": "Decimal 2000000000",
	"Exp(-70)":                         "Decimal 0.0000000000000000000000000000003975449735908647",
}

// TestTheScalePolicyHoldsItsControls holds the policy, and this engine, to
// the controls: the oracle's policy gives each, and so does this engine.
func TestTheScalePolicyHoldsItsControls(t *testing.T) {
	engine := NewEngine()
	for _, expr := range slices.Sorted(maps.Keys(scalePolicyControls)) {
		want := scalePolicyControls[expr]
		v, err := arithmeticOracle(expr)
		if err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
		if !followsPolicy(readEngineArithmetic(want), v.policy) || !isConformant(readEngineArithmetic(want), v.conform) {
			t.Errorf("%s: the policy gives %s and the specification allows %s; ADR 002 gives %s", expr, v.policy, v.conform, want)
		}
		got, err := engine.EvaluateExpression(t.Context(), "library Control version '1.0'\ndefine X: "+expr+"\n", "X", nil, nil)
		if err != nil {
			t.Errorf("%s: this engine fails: %v", expr, err)
			continue
		}
		text := "null"
		if got != nil {
			text = got.Type() + " " + got.String()
		}
		if answer := readEngineArithmetic(text); !follows(answer, v) {
			t.Errorf("%s: this engine answers %s; ADR 002 gives %s", expr, answer.text, want)
		}
	}
}
