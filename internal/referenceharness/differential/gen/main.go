// Command gen writes a CQL library of generated expressions, for comparing this
// engine with the reference engine over far more cases than anyone would write
// by hand.
//
// Reviewing a change to how expressions evaluate case by case samples it, and a
// sample does not converge: a change to the Decimal scale went through ten
// rounds of review, each finding defects the last had not looked at. Running a
// few thousand generated expressions through both engines and reading every
// difference found the rest in one pass, and showed after each fix that nothing
// new had moved. See the README's "Differential testing" section.
//
// The output is reproducible from the seed, so a run can be repeated after a
// change and its differences compared with the run before.
//
// Usage:
//
//	gen -area arithmetic -seed 7 -n 600 > fuzz.cql
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"slices"
	"sort"
	"strings"
)

// An area is the part of the language a run exercises: a generator of one
// expression at a given depth.
type area func(g *generator, depth int) string

var areas = map[string]area{
	"arithmetic": arithmetic,
	"integer":    integer,
	"scope":      scope,
	"types":      types,
	"fhir":       fhirArea,
	"codes":      codes,
	"codesets":   codeSets,
	"membership": membership,
	// fhirmembership is membership with FHIR operands (plan 006).
	"fhirmembership": fhirMembership,
}

// preambles are the definitions an area's expressions refer to, written once
// ahead of them.
var preambles = map[string]string{
	"scope":          scopePreamble,
	"fhir":           fhirPreamble,
	"fhirmembership": fhirPreamble,
	"codes":          codesPreamble,
	"codesets":       codesPreamble,
}

type generator struct {
	rnd *rand.Rand
}

func (g *generator) pick(options ...string) string { return options[g.rnd.Intn(len(options))] }

func (g *generator) chance(p float64) bool { return g.rnd.Float64() < p }

// The literals arithmetic draws from: zero, signs, the scale's edge
// (0.00000001, 0.12345678, 99.99999999), values whose float is not their decimal
// (0.1, 0.21, 0.09), and Integers beside Decimals so promotion is exercised.
var (
	integers = []string{"0", "1", "2", "3", "7", "10", "12", "-1", "-2", "-3", "100", "1000"}
	decimals = []string{
		"0.0", "1.0", "0.5", "2.5", "3.0", "-0.5", "-2.0", "0.1", "0.21", "0.33333333",
		"1.5", "10.0", "0.00000001", "0.12345678", "99.99999999", "-7.25", "1000.0", "0.09", "1.1", "8.0",
	}
)

// atom is a literal, a typed null, or a division by zero — the operands the
// null rules are about.
func (g *generator) atom() string {
	switch {
	case g.chance(0.12):
		return g.pick("(null as Decimal)", "(null as Integer)")
	case g.chance(0.07):
		return g.pick("(1.0 / 0.0)", "(2 div 0)", "(5.0 mod 0.0)")
	}
	v := g.pick(decimals...)
	if g.chance(0.4) {
		v = g.pick(integers...)
	}
	if strings.HasPrefix(v, "-") {
		return "(" + v + ")"
	}
	return v
}

// arithmetic is the operators, the arithmetic functions and the aggregates,
// nested.
func arithmetic(g *generator, depth int) string {
	if depth == 0 {
		return g.atom()
	}
	a, b := arithmetic(g, depth-1), arithmetic(g, depth-1)
	switch r := g.rnd.Float64(); {
	case r < 0.45:
		op := g.pick("+", "-", "*", "/", "^", "div", "mod")
		if op == "^" {
			b = g.pick("2", "3", "-1", "-2", "0", "0.5", "2.0", "-3")
		}
		return "(" + a + " " + op + " " + b + ")"
	case r < 0.75:
		switch f := g.pick("Abs", "Negate", "Ln", "Exp", "Round", "Truncate", "Ceiling", "Floor"); f {
		case "Negate":
			return "(-" + a + ")"
		case "Round":
			if g.chance(0.5) {
				return "Round(" + a + ", " + g.pick("0", "2", "8") + ")"
			}
			return "Round(" + a + ")"
		default:
			return f + "(" + a + ")"
		}
	case r < 0.85:
		return g.pick("Log", "Power") + "(" + a + ", " + b + ")"
	default:
		items := make([]string, 1+g.rnd.Intn(4))
		for i := range items {
			items[i] = arithmetic(g, max(depth-2, 0))
		}
		return g.pick("Avg", "Sum", "Product", "Median", "Variance", "StdDev",
			"PopulationVariance", "PopulationStdDev", "Max", "Min") + "({" + strings.Join(items, ", ") + "})"
	}
}

// The literals integer draws from: the edges of Integer and Long, the roots
// and halves whose products and sums cross them (46341² and 65536² pass the
// largest Integer, 3037000500² the largest Long), and small values beside them
// so most results stay in range and the ones that cross are told apart.
var (
	integerEdges = []string{
		"2147483647", "(-2147483648)", "2147483646", "(-2147483647)", "1073741824", "46340", "46341",
		"65536", "0", "1", "(-1)", "2", "3", "10", "1000000",
	}
	longEdges = []string{
		"9223372036854775807L", "(-9223372036854775807L)", "4611686018427387904L", "3037000499L",
		"3037000500L", "4294967296L", "2147483648L", "0L", "1L", "(-1L)", "2L",
	}
	// Decimals at and past the Integer edges, for the functions that return an
	// Integer from one.
	integerDecimals = []string{
		"2147483647.0", "2147483647.5", "2147483648.0", "-2147483648.0", "-2147483648.5",
		"-2147483649.0", "0.5", "-0.5", "3000000000.5",
	}
)

// integerAtom is an Integer, or a Long when long is set, at or near an edge.
func (g *generator) integerAtom(long bool) string {
	switch {
	case g.chance(0.06):
		if long {
			return "(null as Long)"
		}
		return "(null as Integer)"
	case long:
		return g.pick(longEdges...)
	}
	return g.pick(integerEdges...)
}

// integer is the operators and functions whose Integer or Long result can fall
// outside its type, nested; a Long expression mixes in Integer operands.
func integer(g *generator, depth int) string {
	return integerOf(g, depth, g.chance(0.35))
}

func integerOf(g *generator, depth int, long bool) string {
	if depth == 0 {
		return g.integerAtom(long)
	}
	// An operand of a Long can be an Integer, promoted to one, so the edge of
	// each type is reached in the other's company.
	a, b := integerOf(g, depth-1, long && g.chance(0.7)), integerOf(g, depth-1, long && g.chance(0.7))
	switch r := g.rnd.Float64(); {
	case r < 0.5:
		op := g.pick("+", "-", "*", "*", "div", "mod", "^")
		if op == "^" {
			b = g.pick("0", "1", "2", "3", "31", "32", "63")
			if long {
				b += "L"
			}
		}
		return "(" + a + " " + op + " " + b + ")"
	case r < 0.65:
		return g.integerUnary(a)
	case r < 0.75 && !long:
		d := g.pick(integerDecimals...)
		if strings.HasPrefix(d, "-") {
			d = "(" + d + ")"
		}
		return g.pick("Truncate", "Floor", "Ceiling") + "(" + d + ")"
	case r < 0.8 && !long:
		return "ToInteger('" + strings.Trim(g.pick(integerEdges...), "()") + g.pick("", "0", "1") + "')"
	default:
		items := make([]string, 1+g.rnd.Intn(4))
		for i := range items {
			items[i] = integerOf(g, max(depth-2, 0), long)
		}
		return g.pick("Sum", "Product", "Max", "Min", "Sum", "Product") + "({" + strings.Join(items, ", ") + "})"
	}
}

// scopePreamble is definitions and functions whose names the scope area's
// queries reuse as aliases, lets and operands: a definition evaluates in its
// library's scope, and a name bound where it is referenced must not reach it.
const scopePreamble = `define A: 1
define B: 10
define C: 100
define DA: A + 1
define DB: B * 2
define DAB: A + B
define DList: { A, B, C }
define DQuery: ({ 2, 3 }) A return A + B
define function F(A Integer): DA + A
define function G(B Integer): F(B) + DB
define function H(A Integer, B Integer): DAB + A - B
define fluent function Up(A Integer): A + DA
`

// The names a scope expression binds and refers to: the preamble's, so each
// binding shadows a definition, and Q, which shadows none.
var (
	scopeNames = []string{"A", "B", "C", "Q"}
	scopeRefs  = []string{"A", "B", "C", "DA", "DB", "DAB"}
)

// scope is queries, lets and function calls that bind the preamble's names,
// with bodies that refer to both the bindings and the definitions.
func scope(g *generator, depth int) string {
	return g.scopeExpr(depth, nil)
}

// scopeExpr is an Integer expression over the names bound so far.
func (g *generator) scopeExpr(depth int, bound []string) string {
	if depth == 0 {
		switch {
		case len(bound) > 0 && g.chance(0.4):
			return g.pick(bound...)
		case g.chance(0.6):
			return g.pick(scopeRefs...)
		}
		return g.pick("0", "1", "2", "5")
	}
	switch r := g.rnd.Float64(); {
	case r < 0.3:
		return "(" + g.scopeExpr(depth-1, bound) + " " + g.pick("+", "-", "*") + " " + g.scopeExpr(depth-1, bound) + ")"
	case r < 0.45:
		switch f := g.pick("F", "G", "H", "Up"); f {
		case "H":
			return "H(" + g.scopeExpr(depth-1, bound) + ", " + g.scopeExpr(depth-1, bound) + ")"
		case "Up":
			return "(" + g.scopeExpr(depth-1, bound) + ").Up()"
		default:
			return f + "(" + g.scopeExpr(depth-1, bound) + ")"
		}
	case r < 0.55:
		return g.pick("Sum", "Count", "Max") + "(" + g.pick("DList", "DQuery") + ")"
	default:
		return "Sum(" + g.scopeQuery(depth-1, bound) + ")"
	}
}

// scopeQuery is a query whose alias, let and clauses bind the preamble's names.
func (g *generator) scopeQuery(depth int, bound []string) string {
	alias := g.pick(scopeNames...)
	source := g.pick("{ 3, 4 }", "{ 7 }", "DList", "DQuery", "({ 5, 6 }) X return X")
	inner := append(append([]string{}, bound...), alias)
	q := "((" + source + ") " + alias
	// A let takes a name not bound already: one an alias or a let of the
	// same query holds is not a name the query can bind again.
	if free := unbound(scopeNames, inner); len(free) > 0 && g.chance(0.4) {
		let := g.pick(free...)
		q += " let " + let + ": " + g.scopeExpr(max(depth-1, 0), inner)
		inner = append(inner, let)
	}
	if g.chance(0.3) {
		q += " where " + g.scopeExpr(max(depth-1, 0), inner) + " > " + g.pick("0", "5", "50")
	}
	return q + " return " + g.scopeExpr(depth, inner) + ")"
}

// The values types draws from, one of each kind, and the types it asks about:
// the value's own, its supertypes and unrelated ones, by name and qualified.
var (
	typedValues = []string{
		"5", "5L", "2.5", "'a'", "true", "@2020-01-01", "@2020-01-01T10:00:00.000Z", "@T10:00:00",
		"5 'mg'", "{ 1, 2 }", "Interval[1, 2]", "Tuple{ a: 1 }", "Code{ code: 'x' }",
		"Concept{ codes: { Code{ code: 'x' } } }", "(null as Integer)", "{ 1, 'a' }", "{ 'a' }",
		"List<Integer>{}", "Interval[1, null]", "Interval['a', 'b']",
	}
	typeNames = []string{
		"Any", "System.Any", "Integer", "System.Integer", "Long", "Decimal", "String", "Boolean", "Date",
		"DateTime", "Time", "Quantity", "Code", "Concept", "Vocabulary", "List<Integer>", "List<Any>",
		"Interval<Integer>", "Interval<Any>", "List<String>", "Interval<String>",
	}
)

// types is is, as and cast of a value whose static type is Any — an element
// of a List<Any> — so that every type can be asked about it, as a downcast:
// the reference translator refuses a cast between unrelated static types.
func types(g *generator, _ int) string {
	v := "(List<Any>{ " + g.pick(typedValues...) + " })[0]"
	t := g.pick(typeNames...)
	switch g.pick("is", "as", "as-null", "cast") {
	case "is":
		return "(" + v + " is " + t + ")"
	case "as":
		return "(" + v + " as " + t + ")"
	case "as-null":
		return "((" + v + " as " + t + ") is null)"
	}
	return "(cast " + v + " as " + t + ")"
}

// fhirPreamble puts the fhir area's expressions in a FHIR library with the
// model's conversions, about one patient.
const fhirPreamble = `using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
context Patient
`

// The values fhir draws from: what FHIR data holds — a resource, its
// primitives, a choice on each branch, a period, a code — read from resources
// a test serves by these ids (Observation q with valueQuantity, s with
// valueString, i with valueInteger and an effectivePeriod; an Encounter whose
// period starts on a date with no time; a Patient with a gender, a birth date
// and a date-only deceasedDateTime), beside FHIR instances built in CQL and
// System values of the same names.
var (
	fhirValues = []string{
		"Patient", "Patient.gender", "Patient.birthDate", "Patient.deceased", "Patient.id",
		"First([Encounter]).period", "First([Encounter]).period.start",
		"(First([Observation] O where O.id = 'q')).value", "(First([Observation] O where O.id = 's')).value",
		"(First([Observation] O where O.id = 'i')).value", "(First([Observation] O where O.id = 'q')).effective",
		"(First([Observation] O where O.id = 'i')).effective", "(First([Observation] O where O.id = 'q')).code",
		"FHIR.Coding { code: FHIR.code { value: 'x' } }", "FHIR.code { value: 'x' }",
		"5 'mg'", "{ 1, 2 }", "System.ValueSet { id: 'x' }", "(null as Integer)",
	}
	fhirTypes = []string{
		"Any", "Integer", "Decimal", "String", "Date", "DateTime", "Quantity", "System.Quantity", "Code",
		"Concept", "List<Any>", "Interval<DateTime>", "FHIR.Element", "Element", "FHIR.Resource",
		"FHIR.DomainResource", "Patient", "FHIR.Quantity", "FHIR.dateTime", "FHIR.date", "FHIR.string",
		"FHIR.code", "FHIR.integer", "FHIR.Period", "CodeableConcept", "FHIR.Coding",
	}
	fhirOperations = []string{
		"-(%s)", "Abs(%s)", "Sum({ %s })", "Avg({ %s })", "(%s) + 1 day", "(%s) + 1 'mg'",
		"(%s) same day as @2020-01-01T", "ToString(%s)", "(%s) is null",
	}
)

// fhirArea is is, as and cast of FHIR data against System and FHIR types, and
// the operators that read it, where a value's model decides the answer.
func fhirArea(g *generator, _ int) string {
	v := "(" + g.pick(fhirValues...) + ")"
	if g.chance(0.3) {
		return fmt.Sprintf(g.pick(fhirOperations...), v)
	}
	t := g.pick(fhirTypes...)
	switch g.pick("is", "as", "as-null", "cast") {
	case "is":
		return "(" + v + " is " + t + ")"
	case "as":
		return "(" + v + " as " + t + ")"
	case "as-null":
		return "((" + v + " as " + t + ") is null)"
	}
	return "(cast " + v + " as " + t + ")"
}

// codesPreamble declares codes and concepts whose fields come from their
// declarations: a code system's version, an empty display and none.
const codesPreamble = `codesystem "CS": 'a' version 'b'
codesystem "S": 'a'
code "X": 'a' from "CS"
code "Y": 'a' from "S" display ''
code "Z": 'A' from "S" display 'b'
concept "K": { "X", "Z" } display ''
concept "L": { "Y" }
`

// The values a field of a Code or Concept is drawn from: left out, null, the
// empty String and Strings that differ by case, so that null stands beside the empty one
// and equality beside equivalence.
var (
	fieldTexts   = []string{"", "null", "''", "'a'", "'b'", "'A'"}
	codeFields   = []string{"code", "system", "display", "version"}
	codeNames    = []string{`"X"`, `"Y"`, `"Z"`}
	conceptNames = []string{`"K"`, `"L"`}
)

// codes is reading the fields of Codes and Concepts and comparing them with
// =, !=, ~ and !~, alone and in lists: where null and the empty String are told apart.
func codes(g *generator, _ int) string {
	switch g.pick("field", "is-null", "code", "code", "list", "concept", "concept-field") {
	case "field":
		return "(" + g.fieldOperand() + ")." + g.pick(codeFields...)
	case "is-null":
		return "((" + g.fieldOperand() + ")." + g.pick(codeFields...) + " is null)"
	case "code":
		a := g.codeFields()
		return "(" + g.code(a) + " " + g.pick("=", "!=", "~", "!~") + " " + g.code(g.nearCode(a)) + ")"
	case "list":
		// One Code each, or a null in the same place before it: what the
		// specification decides. Two Codes, one unknown and one unequal, it
		// does not (plan 004, backlog 1c2).
		a := g.codeFields()
		left, right := "{ "+g.code(a)+" }", "{ "+g.code(g.nearCode(a))+" }"
		if g.chance(0.3) {
			left, right = "{ (null as Code), "+g.code(a)+" }", "{ (null as Code), "+g.code(g.nearCode(a))+" }"
		}
		return "(" + left + " " + g.pick("=", "!=") + " " + right + ")"
	case "concept-field":
		k := g.concept(2)
		if g.chance(0.5) {
			return "((" + k + ").display is null)"
		}
		return "(" + k + ").display"
	}
	op := g.pick("=", "!=", "~", "!~")
	// Equal of Concepts of two codes each could hold one pair unknown and
	// another unequal, which the specification does not decide (plan 004,
	// backlog 1c2): equality compares Concepts of one code, or declared ones.
	pair := 2
	if op == "=" || op == "!=" {
		pair = 1
	}
	left, right := g.concept(pair), g.concept(pair)
	if g.chance(0.5) {
		a, b, display := g.codeFields(), g.codeFields(), g.pick(fieldTexts...)
		codes, near := [][]string{a, b}[:pair], [][]string{g.nearCode(a), g.nearCode(b)}[:pair]
		left = g.conceptOf(display, codes...)
		right = g.conceptOf(display, near...)
		if g.chance(0.5) {
			right = g.conceptOf(g.pick(fieldTexts...), near...)
		}
	}
	return "(" + left + " " + op + " " + right + ")"
}

// fieldOperand is a Code whose fields are read: as often a declared code or a
// selector from a declared code system, whose system and version come from
// the declaration.
func (g *generator) fieldOperand() string {
	if g.chance(0.3) {
		return g.selectorCode()
	}
	if g.chance(0.3) {
		return g.pick(codeNames...)
	}
	return g.code(nil)
}

// selectorCode is a Code selector from a declared code system.
func (g *generator) selectorCode() string {
	s := "Code " + g.pick("'a'", "'A'", "''") + " from " + g.pick(`"CS"`, `"S"`)
	if d := g.pick(fieldTexts...); d != "" && d != "null" {
		s += " display " + d
	}
	return s
}

// codeFields draws the text of each field of a Code selector, "" for one left
// out; nil is a declared code or a null Code.
func (g *generator) codeFields() []string {
	if g.chance(0.25) {
		return nil
	}
	f := make([]string, len(codeFields))
	for i := range f {
		f[i] = g.pick(fieldTexts...)
	}
	return f
}

// nearCode is a Code like fields, one field drawn again, so that two Codes
// compared differ in little, or as often a Code drawn afresh.
func (g *generator) nearCode(fields []string) []string {
	if fields == nil || g.chance(0.3) {
		return g.codeFields()
	}
	near := slices.Clone(fields)
	near[g.rnd.Intn(len(near))] = g.pick(fieldTexts...)
	return near
}

// code writes a Code: a selector of fields, or, for nil fields, a declared
// code, a Code selector from a declared code system, or a null Code.
func (g *generator) code(fields []string) string {
	return g.codeOr(fields, "(null as Code)")
}

// codeOr is code with other in place of the null Code: a Concept holds no
// null among its codes (plan 004, out of scope).
func (g *generator) codeOr(fields []string, other ...string) string {
	if fields == nil {
		fields = g.codeFields()
	}
	if fields == nil {
		if g.chance(0.6) {
			return g.selectorCode()
		}
		return g.pick(append(slices.Clone(codeNames), other...)...)
	}
	var set []string
	for i, text := range fields {
		if text != "" {
			set = append(set, codeFields[i]+": "+text)
		}
	}
	if len(set) == 0 {
		set = []string{"code: " + g.pick(fieldTexts[1:]...)}
	}
	return "Code { " + strings.Join(set, ", ") + " }"
}

// concept writes a Concept: a selector of up to most Codes and a display, a
// declared concept, or a null one.
func (g *generator) concept(most int) string {
	if g.chance(0.2) {
		return g.pick(append(slices.Clone(conceptNames), "(null as Concept)")...)
	}
	list := [][]string{g.codeFields()}
	if most > 1 && g.chance(0.4) {
		list = append(list, g.codeFields())
	}
	return g.conceptOf(g.pick(fieldTexts...), list...)
}

// conceptOf writes a Concept selector of codes and a display, "" for none.
func (g *generator) conceptOf(display string, codes ...[]string) string {
	list := make([]string, len(codes))
	for i, c := range codes {
		list[i] = g.codeOr(c)
	}
	s := "Concept { codes: { " + strings.Join(list, ", ") + " }"
	if display != "" {
		s += ", display: " + display
	}
	return s + " }"
}

// codeSets is the operations that compare Codes without `=` — distinct,
// union, intersect, except, in, contains, IndexOf, Mode and a query's return —
// over lists of Codes that differ in little, so that two Codes of another
// display, version or case, or of a null against an empty String, meet, and
// null elements, which these operations take as equal. Mode is asked of a
// list written { a, a, b }, which has no tie.
func codeSets(g *generator, _ int) string {
	base := g.codeFields()
	list := func() string {
		items := make([]string, 1+g.rnd.Intn(3))
		for i := range items {
			items[i] = g.code(g.nearCode(base))
			if g.chance(0.15) {
				items[i] = "(null as Code)"
			}
		}
		if len(items) > 1 && g.chance(0.4) {
			// The same Code twice, which distinct and return keep once.
			items[len(items)-1] = items[0]
		}
		return "{ " + strings.Join(items, ", ") + " }"
	}
	// one is a Code an operation looks for, null as often as an element.
	one := func() string {
		if g.chance(0.15) {
			return "(null as Code)"
		}
		return g.code(g.nearCode(base))
	}
	switch g.pick("distinct", "union", "intersect", "except", "in", "contains", "IndexOf", "Mode", "return") {
	case "distinct":
		return "Count(distinct " + list() + ")"
	case "union", "intersect", "except":
		return "Count(" + list() + " " + g.pick("union", "intersect", "except") + " " + list() + ")"
	case "in":
		return "(" + one() + " in " + list() + ")"
	case "contains":
		return "(" + list() + " contains " + one() + ")"
	case "IndexOf":
		return "IndexOf(" + list() + ", " + one() + ")"
	case "Mode":
		a := one()
		return "Mode({ " + a + ", " + a + ", " + one() + " })." + g.pick(codeFields...)
	}
	return "Count((" + list() + ") C return C)"
}

// unbound is the names that are not among bound.
func unbound(names, bound []string) []string {
	var free []string
	for _, n := range names {
		if !slices.Contains(bound, n) {
			free = append(free, n)
		}
	}
	return free
}

// integerUnary is a one-operand operator or function of a.
func (g *generator) integerUnary(a string) string {
	switch f := g.pick("Negate", "Abs", "successor", "predecessor"); f {
	case "Negate":
		return "(-" + a + ")"
	case "successor", "predecessor":
		return "(" + f + " of " + a + ")"
	default:
		return f + "(" + a + ")"
	}
}

// memberType is a type membership draws its elements from, with values that
// each have one spelling, so that two of them are equal exactly when they are
// written alike, and none holds a null. A point type can also be an interval's.
type memberType struct {
	name   string
	values []string
	point  bool
}

var memberTypes = []memberType{
	{"Integer", []string{"1", "2", "3", "7"}, true},
	{"Decimal", []string{"0.5", "1.5", "2.25"}, true},
	{"String", []string{"'a'", "'b'", "'c'"}, false},
	{"Boolean", []string{"true", "false"}, false},
	{"Date", []string{"@2019-12-31", "@2020-01-01", "@2021-06-15"}, true},
	{"DateTime", []string{"@2020-01-01T10:00:00.000Z", "@2020-01-01T11:30:00.000Z", "@2021-06-15T00:00:00.000Z"}, true},
	{"Time", []string{"@T10:00:00.000", "@T11:30:00.000", "@T23:59:59.999"}, true},
	{"Quantity", []string{"1 'mg'", "2 'mg'", "5 'mg'"}, true},
	{"Code", []string{"Code { code: 'a', system: 's' }", "Code { code: 'b', system: 's' }", "Code { code: 'a', system: 't' }"}, false},
	{"Interval<Integer>", []string{"Interval[1, 2]", "Interval[2, 5]", "Interval[1, 5)"}, false},
	{"Tuple { a Integer }", []string{"Tuple { a: 1 }", "Tuple { a: 2 }"}, false},
	{"List<Integer>", []string{"{ 1 }", "{ 1, 2 }", "{ 3 }"}, false},
}

// membership writes in, contains, includes, included in and during, over a
// list or an interval, with a null on one side or none (plan 005). A list is
// written with its type, List<T>{ … }, and a null with the type it stands
// for, so that the overload — the membership one, or the list-list one — is
// what the text says; a bare null appears only where no overload hangs on it.
// Over an interval, a null is on one side at least: a point and an interval
// both given are a comparison, not this area.
func membership(g *generator, _ int) string {
	t := memberTypes[g.rnd.Intn(len(memberTypes))]
	value := func() string { return g.pick(t.values...) }
	typedNull := func() string {
		if g.chance(0.3) {
			return "First(List<" + t.name + ">{})"
		}
		return "(null as " + t.name + ")"
	}
	op := g.pick("in", "contains", "includes", "included in", "during")
	element, null := value(), g.chance(0.4)
	if null {
		element = typedNull()
		if (op == "in" || op == "contains") && g.chance(0.25) {
			element = "null"
		}
	}
	container := g.memberContainer(t, null)
	if op != "in" && op != "contains" && !strings.HasPrefix(container, "Interval") &&
		!strings.HasPrefix(container, "(null as Interval") && g.chance(0.12) {
		// The list-list overload with a null, which answers null.
		element = "(null as List<" + t.name + ">)"
	}
	switch op {
	case "contains", "includes":
		return "(" + container + " " + op + " " + element + ")"
	}
	return "(" + element + " " + op + " " + container + ")"
}

// memberContainer is the list or interval membership looks in: over a point
// type, sometimes an interval, given only when the element is null; otherwise
// a null list, an empty one, or one of one to three elements, any of them null.
func (g *generator) memberContainer(t memberType, null bool) string {
	if t.point && g.chance(0.25) {
		if null && g.chance(0.5) {
			return "Interval[" + t.values[0] + ", " + t.values[len(t.values)-1] + "]"
		}
		return "(null as Interval<" + t.name + ">)"
	}
	if g.chance(0.12) {
		return "(null as List<" + t.name + ">)"
	}
	if g.chance(0.12) {
		return "List<" + t.name + ">{}"
	}
	items := make([]string, 1+g.rnd.Intn(3))
	for i := range items {
		items[i] = g.pick(t.values...)
		if g.chance(0.2) {
			items[i] = g.pick("null", "(null as "+t.name+")")
		}
	}
	return "List<" + t.name + ">{ " + strings.Join(items, ", ") + " }"
}

// fhirOperand is one side of a membership over testdata/snapshots/fhir-data.json:
// its text and whether its value there is null. A FHIR list with no element in
// the data is empty, not null.
type fhirOperand struct {
	text string
	null bool
}

// fhirFamily is the operands of one kind of value: what is looked for, the
// lists it is looked for in, and, for a point kind, the intervals.
type fhirFamily struct {
	elements, lists, intervals []fhirOperand
}

var fhirFamilies = []fhirFamily{
	{ // strings: FHIR.code, FHIR.string and a String branch of a choice
		elements: []fhirOperand{{"Patient.gender", false}, {"Patient.language", true},
			{"(First([Observation] O where O.id = 's')).value", false}, {"(null as String)", true}},
		lists: []fhirOperand{{"{ 'male', 'female' }", false}, {"{ 'female', null }", false}, {"List<String>{}", false},
			{"(null as List<String>)", true}, {"Patient.name.given", false}},
	},
	{ // booleans
		elements: []fhirOperand{{"Patient.active", true}, {"(null as Boolean)", true}},
		lists:    []fhirOperand{{"{ true, false }", false}, {"{ true, null }", false}, {"(null as List<Boolean>)", true}},
	},
	{ // dates
		elements: []fhirOperand{{"Patient.birthDate", false}, {"(null as Date)", true}},
		lists:    []fhirOperand{{"{ @1980-01-01 }", false}, {"{ @1980-01-02, null }", false}, {"(null as List<Date>)", true}},
	},
	{ // date-times, as points and in Periods
		elements: []fhirOperand{{"First([Encounter]).period.start", false},
			{"(First([Observation] O where O.id = 'q')).effective", false},
			{"(First([Observation] O where O.id = 'x')).effective", true}, {"(null as DateTime)", true},
			{"Patient.birthDate", false}},
		lists: []fhirOperand{{"{ @2020-01-01T00:00:00Z }", false}, {"{ null, @2020-01-01T00:00:00Z }", false},
			{"(null as List<DateTime>)", true}},
		intervals: []fhirOperand{{"First([Encounter]).period", false}, {"(First([Encounter] E where E.id = 'x')).period", true},
			{"(null as FHIR.Period)", true}, {"(null as Interval<DateTime>)", true},
			{"Interval[@2020-01-01T00:00:00Z, @2020-12-31T00:00:00Z]", false},
			{"(First([Observation] O where O.id = 'i')).effective", false},
			{"(First([Observation] O where O.id = 'x')).effective", true}},
	},
	{ // quantities
		elements: []fhirOperand{{"(First([Observation] O where O.id = 'q')).value", false},
			{"(First([Observation] O where O.id = 'x')).value", true}, {"(null as Quantity)", true}},
		lists: []fhirOperand{{"{ -5 'mg' }", false}, {"{ 1 'mg', null }", false}, {"(null as List<Quantity>)", true}},
	},
	{ // a FHIR.integer against Decimals, which takes two conversions
		elements: []fhirOperand{{"(First([Observation] O where O.id = 'i')).value", false},
			{"(First([Observation] O where O.id = 'x')).value", true}},
		lists: []fhirOperand{{"{ 7.0 }", false}, {"(null as List<Decimal>)", true}},
	},
}

// fhirMembership writes in, contains, includes, included in and during with a
// FHIR operand on one side or both, and a null on one side at least, where
// the overload hangs on the model's conversions (plan 006). A point and an
// interval both given are a comparison, not this area. A list-list question
// takes a typed null list as the element.
func fhirMembership(g *generator, _ int) string {
	for {
		f := fhirFamilies[g.rnd.Intn(len(fhirFamilies))]
		op := g.pick("in", "contains", "includes", "included in", "during")
		element := f.elements[g.rnd.Intn(len(f.elements))]
		containers := f.lists
		if len(f.intervals) > 0 && g.chance(0.5) {
			containers = f.intervals
		}
		container := containers[g.rnd.Intn(len(containers))]
		if op != "in" && op != "contains" && len(containers) == len(f.lists) && g.chance(0.15) {
			element = fhirOperand{"(null as List<" + listElementType(container.text) + ">)", true}
			if strings.HasSuffix(element.text, "<>)") {
				continue
			}
		}
		if !element.null && !container.null {
			continue
		}
		switch op {
		case "contains", "includes":
			return "(" + container.text + " " + op + " " + element.text + ")"
		}
		return "(" + element.text + " " + op + " " + container.text + ")"
	}
}

// listElementType is the element type a list operand is written with, for the
// typed null list of the list-list overload, or "" where its text does not say.
func listElementType(list string) string {
	if t, ok := strings.CutPrefix(list, "(null as List<"); ok {
		return strings.TrimSuffix(t, ">)")
	}
	if t, ok := strings.CutPrefix(list, "List<"); ok {
		element, _, _ := strings.Cut(t, ">")
		return element
	}
	return ""
}

func main() {
	name := flag.String("area", "arithmetic", "the part of the language to generate: "+strings.Join(areaNames(), ", "))
	seed := flag.Int64("seed", 1, "the seed; the same seed writes the same library")
	n := flag.Int("n", 600, "how many expressions")
	flag.Parse()

	gen, ok := areas[*name]
	if !ok {
		fmt.Fprintf(os.Stderr, "gen: no area %q (have %s)\n", *name, strings.Join(areaNames(), ", "))
		os.Exit(2)
	}
	g := &generator{rnd: rand.New(rand.NewSource(*seed))} //nolint:gosec // reproducible test input, not a secret
	fmt.Println("library Differential version '1.0'")
	fmt.Print(preambles[*name])
	seen := map[string]bool{}
	for i := 0; i < *n; {
		expr := gen(g, 1+g.rnd.Intn(3))
		if seen[expr] {
			continue
		}
		seen[expr] = true
		fmt.Printf("define D%d: %s\n", i, expr)
		i++
	}
}

func areaNames() []string {
	names := make([]string, 0, len(areas))
	for name := range areas {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
