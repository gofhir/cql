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
}

// preambles are the definitions an area's expressions refer to, written once
// ahead of them.
var preambles = map[string]string{
	"scope": scopePreamble,
	"fhir":  fhirPreamble,
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
