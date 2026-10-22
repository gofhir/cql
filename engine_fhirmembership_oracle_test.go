package cql

import (
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/gofhir/cql/internal/referenceharness/answers"
)

// The fhirmembership oracle computes the answer of each definition of the
// fhirmembership corpus (plan 006): in, contains, includes, included in and
// during with a FHIR operand, and a null on one side at least, over
// testdata/snapshots/fhir-data.json. It uses neither this engine nor its model:
//
//   - the overload is the node the Java translator wrote for the definition
//     alone in its library (fhirmembership.nodes.tsv, from its ELM): In and
//     Contains for the element one, IncludedIn and Includes for two lists, or
//     none where it refused the expression;
//   - whether an operand is null, and what a list holds, is read from the text
//     for a literal and from the JSON of the data for a FHIR path;
//   - the answer is the specification's rule for that overload, as in the
//     membership oracle of plan 005.

const fhirMembershipCorpusSize = 200

const fhirMembershipOracleRule = "fhirmembership oracle (engine_fhirmembership_oracle_test.go), the Java translator's node " +
	"for each definition alone, CQL 1.5.3 (cql.hl7.org/09-b-cqlreference.html): List Operators (In, Contains, Includes, " +
	"Included In), Interval Operators (In, Contains, Includes, Included In); " + undecidedADR + ", decision 6"

var classifyFHIRMembership = flag.Bool("classify-fhirmembership", false,
	"give the unclassified divergences of the fhirmembership area the verdict of its oracle")

// fhirMemberOperand is one side of a definition as the oracle reads it.
type fhirMemberOperand struct {
	null     bool
	interval bool
	items    []string // a list's elements; a null one is "null"
}

// fhirMembershipExpected computes the answer of a definition from the node the
// translator wrote for it and the data.
func fhirMembershipExpected(data fhirMemberData, expr, node string) (membershipVerdict, error) {
	body, ok := unwrapped(expr)
	if !ok {
		return membershipVerdict{}, fmt.Errorf("not one parenthesized membership: %q", expr)
	}
	op, left, right, ok := splitMembership(body)
	if !ok {
		return membershipVerdict{}, fmt.Errorf("no membership operator at the top level: %q", expr)
	}
	element, container := left, right
	if op == "contains" || op == "includes" {
		element, container = right, left
	}
	e, err := data.read(element)
	if err != nil {
		return membershipVerdict{}, err
	}
	c, err := data.read(container)
	if err != nil {
		return membershipVerdict{}, err
	}
	if !e.null && !c.null {
		return membershipVerdict{}, fmt.Errorf("no null on either side, which is not this corpus: %q", expr)
	}
	switch node {
	case "rejected":
		return membershipVerdict{"refused", "the translator resolves no overload: it chains a model conversion with a System one (backlog 1c15)"}, nil
	case "IncludedIn", "Includes":
		return membershipVerdict{"null", "a null in the list-list overload of includes or included in is null"}, nil
	case "In", "Contains":
	default:
		return membershipVerdict{}, fmt.Errorf("a node no membership is: %s", node)
	}
	// A null container answers alike as a list and as an interval, and the
	// data does not say which an absent one would have been.
	switch {
	case c.null && e.null:
		return membershipVerdict{"false", "a null container and a null element: the null container decides, false (ADR 003, decision 6)"}, nil
	case c.null:
		return membershipVerdict{"false", "a null list or interval holds nothing: List and Interval In, second argument null, and Contains, first argument null, are false"}, nil
	case c.interval:
		return membershipVerdict{"null", "a null point in an interval: Interval Contains, second argument null, and Interval In, first argument null, are null"}, nil
	}
	if slices.Contains(c.items, "null") {
		return membershipVerdict{"true", "a null element is in a list that holds a null element: List In, first argument null, and List Contains, second argument null"}, nil
	}
	return membershipVerdict{"false", "a null element is not in a list that holds no null element: List In, first argument null, and List Contains, second argument null"}, nil
}

// fhirMemberData is the resources of fhir-data.json, read as JSON.
type fhirMemberData map[string]json.RawMessage

var (
	fhirMemberTypedNull = regexp.MustCompile(`^\(null as ([A-Za-z.]+(<[A-Za-z.<>]+>)?)\)$`)
	// First([Observation] O where O.id = 'q') and First([Encounter]), and Patient.
	fhirMemberSource = regexp.MustCompile(`^\(?(?:First\(\[(\w+)\](?: \w+ where \w+\.id = '(\w+)')?\)\)?|(Patient))\.([\w.]+)$`)
)

// read reads one operand of the corpus.
func (d fhirMemberData) read(text string) (fhirMemberOperand, error) {
	if fhirMemberTypedNull.MatchString(text) {
		return fhirMemberOperand{null: true}, nil
	}
	if strings.HasPrefix(text, "Interval[") {
		return fhirMemberOperand{interval: true}, nil
	}
	if i := strings.Index(text, "{"); i >= 0 && strings.HasSuffix(text, "}") && (i == 0 || strings.HasPrefix(text, "List<")) {
		inner := strings.TrimSpace(text[i+1 : len(text)-1])
		var items []string
		if inner != "" {
			for _, item := range splitTopLevel(inner, ',') {
				items = append(items, strings.TrimSpace(item))
			}
		}
		return fhirMemberOperand{items: items}, nil
	}
	m := fhirMemberSource.FindStringSubmatch(text)
	if m == nil {
		return fhirMemberOperand{}, fmt.Errorf("an operand the oracle does not read: %q", text)
	}
	kind, id, path := m[1], m[2], strings.Split(m[4], ".")
	if m[3] != "" {
		kind = "Patient"
	}
	resource, err := d.resource(kind, id)
	if err != nil {
		return fhirMemberOperand{}, err
	}
	values := []any{resource}
	interval := false
	for _, segment := range path {
		var next []any
		for _, v := range values {
			field, isPeriod := fhirMemberField(v, segment)
			interval = isPeriod
			switch f := field.(type) {
			case nil:
			case []any:
				next = append(next, f...)
			default:
				next = append(next, f)
			}
		}
		values = next
	}
	// A path with no value reads as null. For a list-valued one, as
	// Patient.name.given, that is an empty list, and the answer is the same in
	// every shape this corpus writes: it holds no element, and no null. A path
	// with values is read as one value, and the corpus's data has no repeated
	// one: a list of values, which this reader does not read, fails here.
	if len(values) > 1 {
		return fhirMemberOperand{}, fmt.Errorf("%q holds %d values, and is not list-valued", text, len(values))
	}
	return fhirMemberOperand{null: len(values) == 0, interval: interval}, nil
}

// resource is the Patient, or the first resource of a kind, or the one with an
// id; nil where there is none.
func (d fhirMemberData) resource(kind, id string) (any, error) {
	raw, ok := d[kind]
	if !ok {
		return nil, fmt.Errorf("no %s in the data", kind)
	}
	if kind == "Patient" {
		var patient any
		err := json.Unmarshal(raw, &patient)
		return patient, err
	}
	var list []map[string]any
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	for _, r := range list {
		if id == "" || r["id"] == id {
			return r, nil
		}
	}
	return nil, nil
}

// fhirMemberField is an element of a JSON object, a choice element read by its
// name and the type it is suffixed with (effective is effectiveDateTime or
// effectivePeriod), and whether it is a Period, which converts to an interval.
func fhirMemberField(v any, name string) (any, bool) {
	// An absent element is null, whether it would be an interval or not: a null
	// container answers alike as either.
	object, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	if field, ok := object[name]; ok {
		return field, name == "period"
	}
	for key, field := range object {
		if suffix, ok := strings.CutPrefix(key, name); ok && suffix != "" && suffix[0] >= 'A' && suffix[0] <= 'Z' {
			return field, suffix == "Period"
		}
	}
	return nil, false
}

// fhirMembershipCorpus reads the definitions, the translator's nodes and the
// data of the corpus.
func fhirMembershipCorpus(t *testing.T) (defs, nodes map[string]string, data fhirMemberData) {
	t.Helper()
	dir := filepath.Join("testdata", "snapshots")
	raw, err := os.ReadFile(filepath.Join(dir, "fhirmembership.cql"))
	if err != nil {
		t.Fatal(err)
	}
	defs = map[string]string{}
	for _, m := range regexp.MustCompile(`(?m)^define (D\d+): (.*)$`).FindAllStringSubmatch(string(raw), -1) {
		defs[m[1]] = m[2]
	}
	raw, err = os.ReadFile(filepath.Join(dir, "fhirmembership.nodes.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	nodes = map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		name, node, ok := strings.Cut(line, "\t")
		if !ok {
			t.Fatalf("fhirmembership.nodes.tsv: %q", line)
		}
		nodes[name] = node
	}
	raw, err = os.ReadFile(filepath.Join(dir, "fhir-data.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	return defs, nodes, data
}

// classifiedByFHIRMembershipOracle is the verdict the oracle gives a
// divergence: a defect of whichever answer is not the oracle's.
func classifiedByFHIRMembershipOracle(t *testing.T, defs, nodes map[string]string, data fhirMemberData, key string, d divergence) divergence {
	t.Helper()
	name, _, _ := strings.Cut(key, "\t")
	v, err := fhirMembershipExpected(data, defs[name], nodes[name])
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	d.Expected = v.expected
	d.Class = ownDefect
	if expectedMatches(d.Expected, d.Engine) {
		d.Class = referenceDefect
	}
	d.Evidence = "fhirmembership oracle: " + v.reason + "; " + strings.TrimPrefix(fhirMembershipOracleRule, "fhirmembership oracle ")
	return d
}

// TestTheFHIRMembershipVerdictsFollowTheOracle holds every verdict of the
// fhirmembership area to what the oracle computes for its definition.
func TestTheFHIRMembershipVerdictsFollowTheOracle(t *testing.T) {
	defs, nodes, data := fhirMembershipCorpus(t)
	path := filepath.Join("testdata", "snapshots", "fhirmembership.divergences.tsv")
	recorded := readDivergences(t, path)
	if *classifyFHIRMembership {
		for key, d := range recorded {
			recorded[key] = classifiedByFHIRMembershipOracle(t, defs, nodes, data, key, d)
		}
		writeVerdicts(t, path, recorded)
		return
	}
	for _, key := range slices.Sorted(maps.Keys(recorded)) {
		d := recorded[key]
		want := classifiedByFHIRMembershipOracle(t, defs, nodes, data, key, d)
		if want.Class != d.Class || want.Expected != d.Expected || want.Evidence != d.Evidence {
			t.Errorf("%s: recorded %s expecting %q (%s); the oracle gives %s expecting %q (%s)",
				oneLine(key), d.Class, d.Expected, oneLine(d.Evidence), want.Class, want.Expected, oneLine(want.Evidence))
		}
	}
}

// TestTheFHIRMembershipOracleAgreesWhereAllAgree holds the oracle to the
// definitions where this engine and both references give one answer.
func TestTheFHIRMembershipOracleAgreesWhereAllAgree(t *testing.T) {
	defs, nodes, data := fhirMembershipCorpus(t)
	recorded := readDivergences(t, filepath.Join("testdata", "snapshots", "fhirmembership.divergences.tsv"))
	java := readReferenceResults(t, "fhirmembership", "java")
	agreed := 0
	for _, name := range slices.Sorted(maps.Keys(defs)) {
		_, javaDiffers := recorded[name+"\tjava"]
		_, jsDiffers := recorded[name+"\tjs"]
		raw, answered := java[name]
		if javaDiffers || jsDiffers || !answered {
			continue
		}
		agreed++
		v, err := fhirMembershipExpected(data, defs[name], nodes[name])
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if answer := pinned(answers.Reference(raw).Text); !expectedMatches(v.expected, answer) {
			t.Errorf("%s: all three answer %q; the oracle expects %q (%s)", name, oneLine(answer), v.expected, v.reason)
		}
	}
	if agreed == 0 {
		t.Fatal("no definition where all three agree: the oracle is checked on nothing")
	}
}

// TestTheFHIRMembershipOracleReadsTheWholeCorpus holds the oracle to every
// definition of the corpus, divergent or not, and to a node for each.
func TestTheFHIRMembershipOracleReadsTheWholeCorpus(t *testing.T) {
	defs, nodes, data := fhirMembershipCorpus(t)
	if len(defs) != fhirMembershipCorpusSize || len(nodes) != len(defs) {
		t.Fatalf("the oracle reads %d definitions and %d nodes; the corpus has %d", len(defs), len(nodes), fhirMembershipCorpusSize)
	}
	for name, expr := range defs {
		if _, err := fhirMembershipExpected(data, expr, nodes[name]); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}
