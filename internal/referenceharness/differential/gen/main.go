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
