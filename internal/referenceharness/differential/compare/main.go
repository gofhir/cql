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
// With -each, every generated definition is evaluated in a library of its own,
// with the corpus's header, as Evaluate.java -each does: a corpus that holds
// invalid expressions on purpose then fails one at a time. With -data, it is
// evaluated over the FHIR resources of that file, the same the snapshot test
// serves, and its Patient as the context.
//
// Usage:
//
//	compare -cql fuzz.cql -reference java.json [-baseline before.json] [-report after.json]
//	compare -each -data fhir-data.json -cql fhir.cql -reference java.json
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"time"

	cql "github.com/gofhir/cql"
	"github.com/gofhir/cql/eval"
	"github.com/gofhir/cql/internal/referenceharness/answers"
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

var (
	definitionLine = regexp.MustCompile(`(?m)^define (\w+): (.*)$`)
	// generatedLine is a definition gen wrote, not one of an area's preamble.
	generatedLine = regexp.MustCompile(`(?m)^define (D\d+): (.*)$`)
)

// readData is the resources of a data file — its Patient, and lists of
// resources by type — served by type, as the snapshot test serves them.
func readData(path string) (provider eval.DataProvider, patient json.RawMessage) {
	raw, err := os.ReadFile(path)
	check(err)
	var data map[string]json.RawMessage
	check(json.Unmarshal(raw, &data))
	served := dataByType{}
	for kind, v := range data {
		if kind == "Patient" {
			continue
		}
		var list []json.RawMessage
		check(json.Unmarshal(v, &list))
		served[kind] = list
	}
	return served, data["Patient"]
}

type dataByType map[string][]json.RawMessage

func (d dataByType) Retrieve(_ context.Context, r eval.RetrieveRequest) ([]json.RawMessage, error) {
	return d[r.ResourceType], nil
}

func main() {
	cqlPath := flag.String("cql", "", "the generated library")
	refPath := flag.String("reference", "", "the reference engine's answers, as Evaluate.java writes them")
	baselinePath := flag.String("baseline", "", "a previous run's report, to say what is new and what is gone")
	reportPath := flag.String("report", "", "where to write this run's report")
	verbose := flag.Bool("v", false, "list every difference, not only the new ones")
	each := flag.Bool("each", false, "evaluate every generated definition in a library of its own")
	dataPath := flag.String("data", "", "FHIR resources to evaluate over, as testdata/snapshots/fhir-data.json")
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

	engine, patient := newEngine(*dataPath)
	header, lines := corpusLines(string(src), *cqlPath, *each)
	var r report
	unanswered := 0
	for _, m := range lines {
		name, expr := m[1], m[2]
		// A definition the reference has no answer for — cql-execution runs only
		// what the translator accepted — is not compared, and not read as null.
		raw, answered := ref.Results[name]
		if !answered {
			unanswered++
			continue
		}
		r.Expressions++
		reference := answers.Reference(raw)
		library := string(src)
		if *each {
			library = header + m[0] + "\n"
		}

		answer := engineAnswer(engine, library, name, patient)
		category := "different"
		switch {
		case answer.Panicked:
			category = "engine-panic"
		case answers.Same(reference, answer) && answers.HoldsUntyped(reference):
			// cql-execution wrote no type for it, and here no oracle says
			// which: the value matches, the type is not confirmed.
			category = "type-unconfirmed"
		case answers.Same(reference, answer):
			continue
		case reference.Kind == "error", reference.Kind == "refused":
			category = "reference-error"
		}
		r.Differences = append(r.Differences, difference{name, expr, category, reference.Text, answer.Text})
	}

	finish(r, unanswered, *baselinePath, *reportPath, *verbose)
}

// finish prints a run's differences — by category, against the baseline when
// one is given — and writes its report.
func finish(r report, unanswered int, baselinePath, reportPath string, verbose bool) {
	byCategory := map[string]int{}
	for _, d := range r.Differences {
		byCategory[d.Category]++
	}
	fmt.Printf("%d expressions, %d differ: %v\n", r.Expressions, len(r.Differences), byCategory)
	if unanswered > 0 {
		fmt.Printf("%d definitions the reference did not answer, not compared\n", unanswered)
	}

	if baselinePath != "" {
		reportAgainst(r, baselinePath)
	}
	if verbose || baselinePath == "" {
		for _, d := range r.Differences {
			printDifference("", d)
		}
	}
	if reportPath != "" {
		out, err := json.MarshalIndent(r, "", "  ")
		check(err)
		check(os.WriteFile(reportPath, append(out, '\n'), 0o600))
	}
}

// newEngine is this engine at the evaluation time Evaluate.java uses, and
// deliberately not UTC (see the README), over the resources of a data file
// when one is given, with its Patient.
func newEngine(dataPath string) (engine *cql.Engine, patient json.RawMessage) {
	opts := []cql.Option{cql.WithEvaluationTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", -3*3600)))}
	if dataPath != "" {
		provider, subject := readData(dataPath)
		opts = append(opts, cql.WithDataProvider(provider))
		patient = subject
	}
	return cql.NewEngine(opts...), patient
}

// corpusLines is the definitions of a corpus to compare, and with -each the
// header each is evaluated after: what comes before the first one gen wrote.
func corpusLines(src, path string, each bool) (header string, lines [][]string) {
	if !each {
		return src, definitionLine.FindAllStringSubmatch(src, -1)
	}
	first := generatedLine.FindStringIndex(src)
	if first == nil {
		check(fmt.Errorf("%s holds no generated definition", path))
	}
	return src[:first[0]], generatedLine.FindAllStringSubmatch(src, -1)
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

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "compare:", err)
		os.Exit(1)
	}
}

// engineAnswer is this engine's answer to a definition, a panic caught and
// marked: always a defect, and reported even where the reference failed too.
func engineAnswer(engine *cql.Engine, src, name string, patient json.RawMessage) (a answers.Answer) {
	defer func() {
		if p := recover(); p != nil {
			a = answers.Failed(fmt.Errorf("%v", p), true)
		}
	}()
	v, err := engine.EvaluateExpression(context.Background(), src, name, patient, nil)
	// A library this engine refuses before running it is a translator's
	// refusal, not a run-time error: the two are different answers.
	var semantic *cql.ErrSemantic
	var syntax *cql.ErrSyntaxError
	switch {
	case errors.As(err, &semantic), errors.As(err, &syntax):
		return answers.Refused(err)
	case err != nil:
		return answers.Failed(err, false)
	}
	return answers.Value(v)
}
