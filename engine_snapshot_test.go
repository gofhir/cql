package cql

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/eval"
	cqltypes "github.com/gofhir/cql/types"
)

// updateSnapshots rewrites the answers: go test -run TestAnswersMatchTheSnapshots -update-snapshots.
var updateSnapshots = flag.Bool("update-snapshots", false, "rewrite testdata/snapshots/*.answers.tsv from this engine")

// snapshotAreas is each corpus in testdata/snapshots with the number of
// generated definitions it holds, so that a corpus that lost its definitions
// fails rather than compares nothing. The corpora are written by
// internal/referenceharness/differential/gen with these seeds, and kept: a
// change to the generator does not change them.
var snapshotAreas = []struct {
	area        string
	definitions int
}{
	{"arithmetic", 800}, // gen -area arithmetic -seed 7 -n 800
	{"integer", 1000},   // gen -area integer -seed 1 -n 1000
	{"scope", 600},      // gen -area scope -seed 1 -n 600
	{"types", 900},      // gen -area types -seed 1 -n 900
	{"fhir", 900},       // gen -area fhir -seed 1 -n 900, on testdata/snapshots/fhir-data.json
}

// TestAnswersMatchTheSnapshots holds this engine to the answers it gave over
// generated corpora, one per area, stored beside them. It is no oracle: it
// says what changed, not whether the change is right. A change to how the
// engine evaluates shows up here as the answers it moves, every one of them,
// for the pull request to account for — by regenerating the answers, so that
// the diff lists them — before it is reviewed. The differential against the
// reference engines says which are right; this says which moved, including
// where no reference can say, as on FHIR data.
//
// Review rounds that read a change case by case did not converge: the fixes
// to one round broke what the next found, in places no check covered. Every
// one of those breaks was an answer that moved.
func TestAnswersMatchTheSnapshots(t *testing.T) {
	for _, a := range snapshotAreas {
		t.Run(a.area, func(t *testing.T) {
			got := snapshotAnswers(t, a.area)
			if len(got) != a.definitions {
				t.Fatalf("%s has %d definitions; the manifest says %d", a.area, len(got), a.definitions)
			}
			path := filepath.Join("testdata", "snapshots", a.area+".answers.tsv")
			if *updateSnapshots {
				writeSnapshot(t, path, got)
				return
			}
			want := readSnapshot(t, path)
			var moved []string
			for _, name := range sortedNames(got) {
				if want[name] != got[name] {
					moved = append(moved, fmt.Sprintf("%s: %s, was %s", name, got[name], want[name]))
				}
			}
			for name := range want {
				if _, ok := got[name]; !ok {
					moved = append(moved, name+": gone")
				}
			}
			if len(moved) > 0 {
				t.Errorf("%d of %d answers moved in %s; account for each, then regenerate with "+
					"go test -run TestAnswersMatchTheSnapshots -update-snapshots:\n%s",
					len(moved), len(got), a.area, strings.Join(moved, "\n"))
			}
		})
	}
}

// snapshotDefine is a generated definition, one line each; what comes before
// the first is the area's header — the library, its preamble.
var snapshotDefine = regexp.MustCompile(`(?m)^define (D\d+):.*$`)

// snapshotAnswers evaluates every generated definition of an area's corpus,
// each in a library of its own with the area's header, so that one invalid
// expression — the corpus holds them on purpose — fails alone rather than
// fail the semantic check of the whole library.
func snapshotAnswers(t *testing.T, area string) map[string]string {
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
	answers := map[string]string{}
	for _, m := range snapshotDefine.FindAllStringSubmatch(src, -1) {
		v, err := engine.EvaluateExpression(context.Background(), header+m[0]+"\n", m[1], patient, nil)
		if err != nil {
			answers[m[1]] = "error: " + snapshotErrorPosition.ReplaceAllString(err.Error(), "")
			continue
		}
		answers[m[1]] = snapshotText(v)
	}
	return answers
}

// snapshotErrorPosition is where in the corpus an error happened, which moves
// whenever a definition is added before it and says nothing about the error.
var snapshotErrorPosition = regexp.MustCompile(`\b\d+:\d+: `)

// snapshotText is a value as the snapshot records it: its type and its value,
// each part of it typed, so that 1 and 1.0, an open and a closed bound, a
// String and a Code of the same text read apart. It is this test's own
// rendering, not String(), which drops what a snapshot has to keep.
func snapshotText(v fptypes.Value) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case fptypes.String:
		return fmt.Sprintf("String %q", x.Value())
	case cqltypes.List:
		parts := make([]string, len(x.Values))
		for i, e := range x.Values {
			parts[i] = snapshotText(e)
		}
		return "List[" + strings.Join(parts, ", ") + "]"
	case cqltypes.Interval:
		open, closeBracket := "(", ")"
		if x.LowClosed {
			open = "["
		}
		if x.HighClosed {
			closeBracket = "]"
		}
		return "Interval" + open + snapshotText(x.Low) + ", " + snapshotText(x.High) + closeBracket
	case cqltypes.Uncertainty:
		return "Uncertainty[" + snapshotText(x.Low) + ", " + snapshotText(x.High) + "]"
	case cqltypes.Tuple:
		names := make([]string, 0, len(x.Elements))
		for name := range x.Elements {
			names = append(names, name)
		}
		sort.Strings(names)
		parts := make([]string, len(names))
		for i, name := range names {
			parts[i] = name + ": " + snapshotText(x.Elements[name])
		}
		return x.Type() + "{" + strings.Join(parts, ", ") + "}"
	case cqltypes.Code:
		return fmt.Sprintf("Code{system: %q, code: %q, display: %q, version: %q}", x.System, x.Code, x.Display, x.Version)
	case cqltypes.Concept:
		parts := make([]string, len(x.Codes))
		for i, c := range x.Codes {
			parts[i] = snapshotText(c)
		}
		return fmt.Sprintf("Concept{codes: [%s], display: %q}", strings.Join(parts, ", "), x.Display)
	case *fptypes.ObjectValue:
		return x.Type() + " " + string(x.Data())
	}
	return v.Type() + " " + v.String()
}

// snapshotProvider serves the fhir area's resources, and its patient.
func snapshotProvider(t *testing.T) (provider eval.DataProvider, patient json.RawMessage) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", "fhir-data.json"))
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	resources := map[string][]json.RawMessage{}
	for kind, v := range data {
		if kind == "Patient" {
			continue
		}
		var list []json.RawMessage
		if err := json.Unmarshal(v, &list); err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		resources[kind] = list
	}
	return snapshotData(resources), data["Patient"]
}

type snapshotData map[string][]json.RawMessage

func (d snapshotData) Retrieve(_ context.Context, r eval.RetrieveRequest) ([]json.RawMessage, error) {
	return d[r.ResourceType], nil
}

func sortedNames(m map[string]string) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func writeSnapshot(t *testing.T, path string, answers map[string]string) {
	t.Helper()
	var b strings.Builder
	for _, name := range sortedNames(answers) {
		fmt.Fprintf(&b, "%s\t%s\n", name, strings.ReplaceAll(answers[name], "\n", `\n`))
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readSnapshot(t *testing.T, path string) map[string]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("%v; write it with go test -run TestAnswersMatchTheSnapshots -update-snapshots", err)
	}
	defer f.Close()
	answers := map[string]string{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		name, answer, ok := strings.Cut(scanner.Text(), "\t")
		if !ok {
			t.Fatalf("%s: a line with no tab: %q", path, scanner.Text())
		}
		answers[name] = answer
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return answers
}
