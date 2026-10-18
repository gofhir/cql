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
	"sort"
	"strings"
)

// An area is the part of the language a run exercises: a generator of one
// expression at a given depth.
type area func(g *generator, depth int) string

var areas = map[string]area{
	"arithmetic": arithmetic,
	"integer":    integer,
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
