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
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	cql "github.com/gofhir/cql"
	"github.com/gofhir/cql/eval"
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
		reference := referenceAnswer(raw)
		library := string(src)
		if *each {
			library = header + m[0] + "\n"
		}

		answer := engineAnswer(engine, library, name, patient)
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

// answer is one engine's answer to a definition: a number, a boolean, a
// string, null, an error, a FHIR value, a list of those, or anything else as
// its text.
type answer struct {
	kind     string // "number", "boolean", "string", "null", "error", "fhir", "list", "other"
	text     string
	num      decimal.Decimal
	panicked bool
}

// referenceValue is one value as the reference engines encode it.
type referenceValue struct {
	Type       string            `json:"type"`
	Value      json.RawMessage   `json:"value"`
	Unit       string            `json:"unit"`
	Elements   []json.RawMessage `json:"elements"`
	Low        json.RawMessage   `json:"low"`
	High       json.RawMessage   `json:"high"`
	LowClosed  bool              `json:"lowClosed"`
	HighClosed bool              `json:"highClosed"`
	Error      string            `json:"error"`
	Message    string            `json:"message"`
}

func referenceAnswer(raw json.RawMessage) answer {
	if len(raw) == 0 || string(raw) == "null" {
		return answer{kind: "null", text: "null"}
	}
	var v referenceValue
	if err := json.Unmarshal(raw, &v); err != nil {
		return answer{kind: "other", text: string(raw)}
	}
	if v.Error != "" {
		return answer{kind: "error", text: "error: " + v.Message}
	}
	// A FHIR value: a primitive as its value, a resource or an element as its
	// JSON, read back so that key order and spacing do not matter.
	if fhirType, ok := strings.CutPrefix(v.Type, "FHIR."); ok {
		var s string
		if json.Unmarshal(v.Value, &s) == nil {
			if strings.HasPrefix(strings.TrimSpace(s), "{") {
				return answer{kind: "fhir", text: canonicalJSON([]byte(s))}
			}
			return answer{kind: "fhir", text: s}
		}
		return answer{kind: "other", text: fhirType + " " + string(v.Value)}
	}
	if v.Type == "List" {
		items := make([]answer, len(v.Elements))
		for i, e := range v.Elements {
			items[i] = referenceAnswer(e)
		}
		return listAnswer(items)
	}
	if a, ok := referenceScalar(v); ok {
		return a
	}
	return answer{kind: "other", text: string(raw)}
}

// referenceScalar is a reference value that is one of CQL's System scalars, or
// an interval of them.
func referenceScalar(v referenceValue) (answer, bool) {
	var s string
	switch v.Type {
	case "Quantity":
		if json.Unmarshal(v.Value, &s) == nil {
			if d, err := decimal.NewFromString(s); err == nil {
				return quantityAnswer(d, v.Unit), true
			}
		}
	case "Date", "DateTime", "Time":
		if json.Unmarshal(v.Value, &s) == nil {
			return answer{kind: "temporal", text: v.Type + " " + s}, true
		}
	case "Interval":
		return intervalAnswer(referenceAnswer(v.Low), referenceAnswer(v.High), v.LowClosed, v.HighClosed), true
	case "Integer", "Long", "Decimal":
		if json.Unmarshal(v.Value, &s) == nil {
			if d, err := decimal.NewFromString(s); err == nil {
				return answer{kind: "number", text: v.Type + " " + s, num: d}, true
			}
		}
	case "Boolean":
		var b bool
		if json.Unmarshal(v.Value, &b) == nil {
			return answer{kind: "boolean", text: fmt.Sprint(b)}, true
		}
	case "String":
		if json.Unmarshal(v.Value, &s) == nil {
			return answer{kind: "string", text: strconv.Quote(s)}, true
		}
	}
	return answer{}, false
}

func engineAnswer(engine *cql.Engine, src, name string, patient json.RawMessage) (a answer) {
	defer func() {
		if p := recover(); p != nil {
			a = answer{kind: "error", text: fmt.Sprintf("PANIC: %v", p), panicked: true}
		}
	}()
	v, err := engine.EvaluateExpression(context.Background(), src, name, patient, nil)
	if err != nil {
		return answer{kind: "error", text: "error: " + err.Error()}
	}
	return valueAnswer(v)
}

// valueAnswer is engineAnswer for a value already evaluated: an element of a
// list.
func valueAnswer(v fptypes.Value) answer {
	if o, ok := v.(*fptypes.ObjectValue); ok {
		return answer{kind: "fhir", text: canonicalJSON(o.Data())}
	}
	if fptypes.IsFHIRPrimitive(v) {
		return answer{kind: "fhir", text: v.String()}
	}
	switch n := v.(type) {
	case fptypes.Quantity:
		return quantityAnswer(n.Value(), n.Unit())
	case fptypes.Date:
		return answer{kind: "temporal", text: "Date " + n.String()}
	case fptypes.DateTime:
		return answer{kind: "temporal", text: "DateTime " + n.String()}
	case fptypes.Time:
		return answer{kind: "temporal", text: "Time " + n.String()}
	case cqltypes.Interval:
		return intervalAnswer(valueAnswer(n.Low), valueAnswer(n.High), n.LowClosed, n.HighClosed)
	case nil:
		return answer{kind: "null", text: "null"}
	case fptypes.Integer:
		return answer{kind: "number", text: "Integer " + n.String(), num: decimal.NewFromInt(n.Value())}
	case fptypes.Decimal:
		return answer{kind: "number", text: "Decimal " + n.String(), num: n.Value()}
	case fptypes.Boolean:
		return answer{kind: "boolean", text: fmt.Sprint(n.Bool())}
	case fptypes.String:
		return answer{kind: "string", text: strconv.Quote(n.Value())}
	case cqltypes.List:
		items := make([]answer, len(n.Values))
		for i, e := range n.Values {
			items[i] = valueAnswer(e)
		}
		return listAnswer(items)
	}
	return answer{kind: "other", text: v.Type() + " " + v.String()}
}

// listAnswer is a list of answers, as one text both engines' lists come to:
// numbers by value, so {1.0} is {1}, strings quoted, and nested lists alike.
// An element of another type — a tuple, an interval, a date — has a text of
// each engine's own, so it keeps the list "other": reported, never matched.
func listAnswer(items []answer) answer {
	parts := make([]string, len(items))
	kind := "list"
	for i, it := range items {
		switch it.kind {
		case "number":
			parts[i] = it.num.String()
		case "null", "boolean", "string", "fhir", "list", "quantity", "temporal", "interval":
			parts[i] = it.text
		default:
			kind = "other"
			parts[i] = it.text
		}
	}
	return answer{kind: kind, text: "{" + strings.Join(parts, ", ") + "}"}
}

// same compares numbers by value, since 1.0 = 1 in CQL, and everything else by
// kind and text, exactly: 'abc' and 'ABC' are two strings. Two errors are the
// same whatever their messages say.
func same(a, b answer) bool {
	switch {
	case a.kind != b.kind:
		return false
	case a.kind == "number":
		return a.num.Equal(b.num)
	case a.kind == "error", a.kind == "null":
		return true
	}
	return a.text == b.text
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "compare:", err)
		os.Exit(1)
	}
}

// canonicalJSON is a JSON document read and written back, with its keys
// sorted, so that two engines' encodings of one resource compare equal.
func canonicalJSON(raw []byte) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	out, err := json.Marshal(v)
	if err != nil {
		return string(raw)
	}
	return string(out)
}

// quantityAnswer is a quantity by its value as a number and its unit, as
// both engines write it.
func quantityAnswer(value decimal.Decimal, unit string) answer {
	return answer{kind: "quantity", text: value.String() + " '" + unit + "'"}
}

// intervalAnswer is an interval of two answers, with its closed and open
// bounds, which a reader of either engine would be misled to drop. An
// interval of points neither engine can say more about stays "other".
func intervalAnswer(low, high answer, lowClosed, highClosed bool) answer {
	open, closeBracket := "(", ")"
	if lowClosed {
		open = "["
	}
	if highClosed {
		closeBracket = "]"
	}
	kind := "interval"
	for _, b := range []answer{low, high} {
		switch b.kind {
		case "number", "null", "temporal", "quantity":
		default:
			kind = "other"
		}
	}
	text := func(b answer) string {
		if b.kind == "number" {
			return b.num.String()
		}
		return b.text
	}
	return answer{kind: kind, text: "Interval" + open + text(low) + ", " + text(high) + closeBracket}
}
