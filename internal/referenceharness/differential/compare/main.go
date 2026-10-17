// Command compare evaluates a generated library with this engine and compares
// every definition with what the reference engine answered for it, and with a
// previous run.
//
// It reads the reference engine's answers as Evaluate.java writes them, at the
// same fixed evaluation time this engine is given here, and sorts each
// difference into one of:
//
//	engine-panic     this engine panicked — always a defect, and reported
//	                 even where the reference engine failed too
//	reference-error  the reference engine threw, and this engine answered
//	                 (often null, as CQL asks of an undefined result)
//	different        the two engines answered differently
//
// A difference is not a defect by itself: some are decisions this engine makes
// against the reference, recorded in runtime.divergences.md, and some are the
// reference's own failures. Every one has to be read. What the tool settles is
// that none goes unread, and — given a previous run's report with -baseline —
// which differences are new since then and which are gone.
//
// Usage:
//
//	compare -cql fuzz.cql -reference java.json [-baseline before.json] [-report after.json]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	cql "github.com/gofhir/cql"
	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
)

// difference is one definition the engines answered differently.
type difference struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
	Category   string `json:"category"`
	Reference  string `json:"reference"`
	Engine     string `json:"engine"`
}

type report struct {
	Expressions int          `json:"expressions"`
	Differences []difference `json:"differences"`
}

var definitionLine = regexp.MustCompile(`(?m)^define (\w+): (.*)$`)

func main() {
	cqlPath := flag.String("cql", "", "the generated library")
	refPath := flag.String("reference", "", "the reference engine's answers, as Evaluate.java writes them")
	baselinePath := flag.String("baseline", "", "a previous run's report, to say what is new and what is gone")
	reportPath := flag.String("report", "", "where to write this run's report")
	verbose := flag.Bool("v", false, "list every difference, not only the new ones")
	flag.Parse()
	if *cqlPath == "" || *refPath == "" {
		flag.Usage()
		os.Exit(2)
	}

	src, err := os.ReadFile(*cqlPath)
	check(err)
	var ref struct {
		Results map[string]json.RawMessage `json:"results"`
	}
	raw, err := os.ReadFile(*refPath)
	check(err)
	check(json.Unmarshal(raw, &ref))

	// The evaluation time Evaluate.java uses, and deliberately not UTC: see
	// the README.
	engine := cql.NewEngine(cql.WithEvaluationTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", -3*3600))))
	var r report
	for _, m := range definitionLine.FindAllStringSubmatch(string(src), -1) {
		name, expr := m[1], m[2]
		r.Expressions++
		reference := referenceAnswer(ref.Results[name])
		answer := engineAnswer(engine, string(src), name)
		category := "different"
		switch {
		case answer.panicked:
			category = "engine-panic"
		case same(reference, answer):
			continue
		case reference.kind == "error":
			category = "reference-error"
		}
		r.Differences = append(r.Differences, difference{name, expr, category, reference.text, answer.text})
	}

	byCategory := map[string]int{}
	for _, d := range r.Differences {
		byCategory[d.Category]++
	}
	fmt.Printf("%d expressions, %d differ: %v\n", r.Expressions, len(r.Differences), byCategory)

	if *baselinePath != "" {
		reportAgainst(r, *baselinePath)
	}
	if *verbose || *baselinePath == "" {
		for _, d := range r.Differences {
			printDifference("", d)
		}
	}
	if *reportPath != "" {
		out, err := json.MarshalIndent(r, "", "  ")
		check(err)
		check(os.WriteFile(*reportPath, append(out, '\n'), 0o600))
	}
}

// reportAgainst prints the differences that are new since the baseline and the
// ones that are gone, by definition name. The generator is reproducible from
// its seed, so the same name is the same expression in both runs.
func reportAgainst(r report, path string) {
	raw, err := os.ReadFile(path)
	check(err)
	var before report
	check(json.Unmarshal(raw, &before))
	was := map[string]bool{}
	for _, d := range before.Differences {
		was[d.Name] = true
	}
	now := map[string]bool{}
	var added []difference
	for _, d := range r.Differences {
		now[d.Name] = true
		if !was[d.Name] {
			added = append(added, d)
		}
	}
	var gone []string
	for _, d := range before.Differences {
		if !now[d.Name] {
			gone = append(gone, d.Name)
		}
	}
	sort.Strings(gone)
	fmt.Printf("against the baseline: %d new, %d gone %v\n", len(added), len(gone), gone)
	for _, d := range added {
		printDifference("NEW ", d)
	}
}

func printDifference(prefix string, d difference) {
	fmt.Printf("%s%s [%s]: %s\n    reference: %s\n    engine:    %s\n", prefix, d.Name, d.Category, d.Expression, d.Reference, d.Engine)
}

// answer is one engine's answer to a definition: a number, a boolean, null, an
// error, or anything else as its text.
type answer struct {
	kind     string // "number", "boolean", "null", "error", "other"
	text     string
	num      decimal.Decimal
	panicked bool
}

func referenceAnswer(raw json.RawMessage) answer {
	if len(raw) == 0 || string(raw) == "null" {
		return answer{kind: "null", text: "null"}
	}
	var v struct {
		Type    string          `json:"type"`
		Value   json.RawMessage `json:"value"`
		Error   string          `json:"error"`
		Message string          `json:"message"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return answer{kind: "other", text: string(raw)}
	}
	if v.Error != "" {
		return answer{kind: "error", text: "error: " + v.Message}
	}
	var s string
	switch v.Type {
	case "Integer", "Long", "Decimal":
		if json.Unmarshal(v.Value, &s) == nil {
			if d, err := decimal.NewFromString(s); err == nil {
				return answer{kind: "number", text: v.Type + " " + s, num: d}
			}
		}
	case "Boolean":
		var b bool
		if json.Unmarshal(v.Value, &b) == nil {
			return answer{kind: "boolean", text: fmt.Sprint(b)}
		}
	}
	return answer{kind: "other", text: string(raw)}
}

func engineAnswer(engine *cql.Engine, src, name string) (a answer) {
	defer func() {
		if p := recover(); p != nil {
			a = answer{kind: "error", text: fmt.Sprintf("PANIC: %v", p), panicked: true}
		}
	}()
	v, err := engine.EvaluateExpression(context.Background(), src, name, nil, nil)
	switch {
	case err != nil:
		return answer{kind: "error", text: "error: " + err.Error()}
	case v == nil:
		return answer{kind: "null", text: "null"}
	}
	switch n := v.(type) {
	case fptypes.Integer:
		return answer{kind: "number", text: "Integer " + n.String(), num: decimal.NewFromInt(n.Value())}
	case fptypes.Decimal:
		return answer{kind: "number", text: "Decimal " + n.String(), num: n.Value()}
	case fptypes.Boolean:
		return answer{kind: "boolean", text: fmt.Sprint(n.Bool())}
	case cqltypes.List:
		return answer{kind: "other", text: "list " + n.String()}
	}
	return answer{kind: "other", text: v.Type() + " " + v.String()}
}

// same compares numbers by value, since 1.0 = 1 in CQL, and everything else by
// kind and text. Two errors are the same whatever their messages say.
func same(a, b answer) bool {
	switch {
	case a.kind != b.kind:
		return false
	case a.kind == "number":
		return a.num.Equal(b.num)
	case a.kind == "error", a.kind == "null":
		return true
	}
	return strings.EqualFold(a.text, b.text)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "compare:", err)
		os.Exit(1)
	}
}
