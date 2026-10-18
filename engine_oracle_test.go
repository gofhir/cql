package cql

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gofhir/cql/internal/referenceharness/answers"
)

// updateDivergences rewrites the divergence manifests, keeping the note of
// every entry that stays: go test -run TestAnswersMatchTheReferences -update-divergences.
var updateDivergences = flag.Bool("update-divergences", false,
	"rewrite testdata/snapshots/*.divergences.tsv from this engine against the references")

// TestAnswersMatchTheReferences compares this engine with the Java engine and
// cql-execution over every area's corpus, and holds it to the divergences its
// manifest records: <area>.divergences.tsv, one line per definition and
// reference that differ, with a note saying why. It fails on a divergence the
// manifest does not record — a regression, or a change to account for — and on
// one it records that is gone, so that what a change settles is written down
// rather than left to rot in the list.
//
// The snapshot says which answers moved; this says whether they moved toward
// or away from the references, which a run of compare by hand used to say
// only when someone remembered to run it. A divergence is not a defect by
// itself: the note says whether it is a decision, a reference's own failure
// (the Java engine wraps an Integer past its range, and fails on Any), or one
// still to settle.
func TestAnswersMatchTheReferences(t *testing.T) {
	for _, a := range snapshotAreas {
		t.Run(a.area, func(t *testing.T) {
			got := referenceDivergences(t, a.area, a.exact)
			path := filepath.Join("testdata", "snapshots", a.area+".divergences.tsv")
			recorded := readDivergences(t, path)
			if *updateDivergences {
				writeDivergences(t, path, got, recorded)
				return
			}
			var problems []string
			for _, key := range sortedKeys(got) {
				if _, ok := recorded[key]; !ok {
					problems = append(problems, "new: "+key+": "+got[key])
				}
			}
			for _, key := range sortedKeys(recorded) {
				if _, ok := got[key]; !ok {
					problems = append(problems, "gone: "+key+" ("+recorded[key]+")")
				}
			}
			if len(problems) > 0 {
				t.Errorf("%d divergences of %s from the references are not the ones recorded in %s; account for "+
					"each, then rewrite it with go test -run TestAnswersMatchTheReferences -update-divergences:\n%s",
					len(problems), a.area, path, strings.Join(problems, "\n"))
			}
		})
	}
}

// referenceDivergences is every definition of an area and reference engine
// whose answers differ, keyed "D17\tjava", with both answers as its value.
func referenceDivergences(t *testing.T, area string, exact bool) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", area+".cql"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(raw)
	first := snapshotDefine.FindStringIndex(src)
	if first == nil {
		t.Fatalf("%s holds no generated definition", area)
	}
	header := src[:first[0]]
	opts := []Option{WithEvaluationTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", -3*3600)))}
	var patient json.RawMessage
	if area == "fhir" {
		provider, subject := snapshotProvider(t)
		opts = append(opts, WithDataProvider(provider))
		patient = subject
	}
	engine := NewEngine(opts...)
	refs := map[string]map[string]json.RawMessage{
		"java": readReferenceResults(t, area, "java"),
		"js":   readReferenceResults(t, area, "js"),
	}
	same := answers.Same
	if exact {
		same = answers.SameExactly
	}
	divergences := map[string]string{}
	for _, m := range snapshotDefine.FindAllStringSubmatch(src, -1) {
		name := m[1]
		answer := snapshotEngineAnswer(engine, header+m[0]+"\n", name, patient)
		for _, engineName := range []string{"java", "js"} {
			raw, answered := refs[engineName][name]
			if !answered {
				continue // cql-execution runs only what the translator accepted
			}
			reference := answers.Reference(raw)
			if same(reference, answer) && !answer.Panicked {
				continue
			}
			divergences[name+"\t"+engineName] = fmt.Sprintf("%s vs %s", oneLine(reference.Text), oneLine(answer.Text))
		}
	}
	return divergences
}

// snapshotEngineAnswer is this engine's answer to a definition, read as the
// references' are.
func snapshotEngineAnswer(engine *Engine, src, name string, patient json.RawMessage) (a answers.Answer) {
	defer func() {
		if p := recover(); p != nil {
			a = answers.Failed(fmt.Errorf("%v", p), true)
		}
	}()
	v, err := engine.EvaluateExpression(context.Background(), src, name, patient, nil)
	if err != nil {
		return answers.Failed(err, false)
	}
	return answers.Value(v)
}

// readReferenceResults is an area's answers from one reference engine.
func readReferenceResults(t *testing.T, area, engine string) map[string]json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", area+"."+engine+".json"))
	if err != nil {
		t.Fatalf("%v: the references are part of the corpus", err)
	}
	var ref struct {
		Results map[string]json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	return ref.Results
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 120 {
		s = s[:120] + "…"
	}
	return s
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// readDivergences is a manifest: "name\treference\tnote" per line, keyed by
// its first two fields, with its note.
func readDivergences(t *testing.T, path string) map[string]string {
	t.Helper()
	recorded := map[string]string{}
	f, err := os.Open(path)
	if os.IsNotExist(err) && *updateDivergences {
		return recorded
	}
	if err != nil {
		t.Fatalf("%v; write it with go test -run TestAnswersMatchTheReferences -update-divergences", err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) != 3 {
			t.Fatalf("%s: a line that is not name, reference and note: %q", path, line)
		}
		recorded[fields[0]+"\t"+fields[1]] = fields[2]
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return recorded
}

// writeDivergences writes a manifest of the divergences found, keeping the
// note each had; a new one is noted as unclassified, to be settled later.
func writeDivergences(t *testing.T, path string, found, recorded map[string]string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("# name\treference\tnote — see TestAnswersMatchTheReferences\n")
	for _, key := range sortedKeys(found) {
		note, ok := recorded[key]
		if !ok {
			note = "unclassified: " + found[key]
		}
		fmt.Fprintf(&b, "%s\t%s\n", key, note)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestTheUnclassifiedDivergencesDoNotGrow holds the divergences the manifests
// mark "unclassified" to the debt recorded in
// testdata/snapshots/unclassified-debt.tsv when ADR-001 was written: every one
// of them has to be an entry there, and every entry has to be still
// unclassified, so an entry can only be removed, when its divergence is
// classified. -update-divergences writes a new divergence as unclassified on
// its own, which would join the debt unseen; and a count let one row be
// classified while another, classified before, was unclassified in its place.
func TestTheUnclassifiedDivergencesDoNotGrow(t *testing.T) {
	debt := readUnclassifiedDebt(t)
	unclassified := map[string]bool{}
	for _, a := range snapshotAreas {
		for key, note := range readDivergences(t, filepath.Join("testdata", "snapshots", a.area+".divergences.tsv")) {
			if strings.HasPrefix(note, "unclassified") {
				unclassified[a.area+"\t"+key] = true
			}
		}
	}
	for _, key := range slices.Sorted(maps.Keys(unclassified)) {
		if !debt[key] {
			t.Errorf("%s is unclassified and not in unclassified-debt.tsv: classify it in its manifest", oneLine(key))
		}
	}
	for _, key := range slices.Sorted(maps.Keys(debt)) {
		if !unclassified[key] {
			t.Errorf("%s is in unclassified-debt.tsv and classified now: remove it from the debt", oneLine(key))
		}
	}
}

func readUnclassifiedDebt(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", "unclassified-debt.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "\r") {
		t.Fatal("unclassified-debt.tsv has CRLF line endings: write it with LF")
	}
	debt := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Count(line, "\t") != 2 {
			t.Fatalf("unclassified-debt.tsv: %q is not area, name and reference", line)
		}
		if debt[line] {
			t.Errorf("unclassified-debt.tsv lists %s twice", oneLine(line))
		}
		debt[line] = true
	}
	return debt
}
