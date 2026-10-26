package cql

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

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
	// exact is a corpus that computes nothing in floating point, compared with
	// the references without the tolerance for a float's last digits.
	exact bool
}{
	{"arithmetic", 800, false},    // gen -area arithmetic -seed 7 -n 800
	{"integer", 1000, false},      // gen -area integer -seed 1 -n 1000
	{"scope", 600, false},         // gen -area scope -seed 1 -n 600
	{"types", 900, false},         // gen -area types -seed 1 -n 900
	{"fhir", 900, false},          // gen -area fhir -seed 1 -n 900, on testdata/snapshots/fhir-data.json
	{"codes", 600, true},          // gen -area codes -seed 1 -n 600
	{"codesets", 400, true},       // gen -area codesets -seed 1 -n 400
	{"membership", 400, true},     // gen -area membership -seed 1 -n 400
	{"fhirmembership", 200, true}, // gen -area fhirmembership -seed 1 -n 200, on testdata/snapshots/fhir-data.json
	// Written by hand, not generated: the cases of docs/plans/001-decimal-division.md.
	{"division", 29, true},
	// Written by hand: the cases of docs/plans/008-fhirpath-1-16.md, where
	// fhirpath v1.16.6 moved an answer and no generator reaches it.
	{"units", 73, true},
	{"temporal", 49, true},
	{"crossmodel", 40, true}, // on testdata/snapshots/fhir-data.json
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
				// An exact area's answers are its text: the tolerance for a
				// float's last digits would let 10^19 + 1 pass for 10^19.
				if a.exact && want[name] != got[name] || !a.exact && !snapshotSame(want[name], got[name]) {
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
	if readsFHIRData(area) {
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
		return fmt.Sprintf("Code{system: %s, code: %s, display: %s, version: %s}",
			snapshotElement(x.System), snapshotElement(x.Code), snapshotElement(x.Display), snapshotElement(x.Version))
	case cqltypes.Concept:
		parts := make([]string, len(x.Codes))
		for i, c := range x.Codes {
			parts[i] = snapshotText(c)
		}
		return fmt.Sprintf("Concept{codes: [%s], display: %s}", strings.Join(parts, ", "), snapshotElement(x.Display))
	case *fptypes.ObjectValue:
		return x.Type() + " " + string(x.Data())
	}
	return v.Type() + " " + v.String()
}

// snapshotTolerance is how far apart two Decimals may be, relative to the
// larger, and be the same answer. Exp, Ln and a power to a fraction are
// computed in float64, and Go computes those differently by platform — in
// assembly on amd64, in Go on arm64, where a multiply and an add may fuse — so
// their last digit is not the same on the CI's machines and a developer's: Exp
// of one value was 183.09405819371815 on one and 183.09405819371818 on the
// other. Rounding both to fewer digits does not settle it, as a value on a
// rounding boundary goes up on one and down on the other. That the engine's
// answer depends on where it runs is a defect of its own, recorded apart; the
// snapshot keeps the digits the engine wrote and compares past that noise.
var snapshotTolerance = decimal.New(1, -13)

// snapshotSame reports whether an answer is the one recorded: the same text,
// or two Decimals a float64 computed — more significant digits than one holds
// for certain, on either side, since the shortest text of the same float can
// be 15 digits on one platform and 17 on another — within snapshotTolerance of
// each other. Any other Decimal is compared as written, so that 10.0 is still
// not 10.
func snapshotSame(want, got string) bool {
	if want == got {
		return true
	}
	a, aFloat, aok := snapshotDecimalOf(want)
	b, bFloat, bok := snapshotDecimalOf(got)
	if !aok || !bok || (!aFloat && !bFloat) {
		return false
	}
	scale := decimal.Max(a.Abs(), b.Abs())
	return a.Sub(b).Abs().LessThanOrEqual(scale.Mul(snapshotTolerance))
}

// snapshotFloatDigits is the significant digits past which a Decimal is
// taken for a float64's, which holds 15 for certain and writes up to 17.
const snapshotFloatDigits = 15

// snapshotDecimalOf is the Decimal an answer holds, and whether it has more
// significant digits than snapshotFloatDigits.
func snapshotDecimalOf(answer string) (d decimal.Decimal, float, ok bool) {
	text, ok := strings.CutPrefix(answer, "Decimal ")
	if !ok {
		return decimal.Decimal{}, false, false
	}
	d, err := decimal.NewFromString(text)
	if err != nil {
		return decimal.Decimal{}, false, false
	}
	digits := len(strings.TrimLeft(d.Abs().Coefficient().String(), "0"))
	return d, digits > snapshotFloatDigits, true
}

// snapshotProvider serves the fhir area's resources, and its patient.
// readsFHIRData reports whether an area's definitions read the resources of
// testdata/snapshots/fhir-data.json, as scripts/references.sh serves them to
// both reference engines.
func readsFHIRData(area string) bool {
	return area == "fhir" || area == "fhirmembership" || area == "crossmodel"
}

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

// TestASnapshotComparesPastFloatNoiseOnly covers snapshotSame: the four answers
// CI and a developer's machine wrote differently, a float64's last digit apart,
// are the same; a written precision, a digit an answer moved by, or a Decimal
// against an Integer are not.
func TestASnapshotComparesPastFloatNoiseOnly(t *testing.T) {
	for _, tt := range []struct {
		want, got string
		same      bool
	}{
		{"Decimal 183.09405819371818", "Decimal 183.09405819371815", true},
		{"Decimal 4.4816890703380645", "Decimal 4.481689070338065", true},
		{"Decimal 1907346572495099500000", "Decimal 1907346572495099800000", true},
		{"Decimal 0.15311848890866597", "Decimal 0.153118488908666", true},
		{"Decimal 10.0", "Decimal 10", false},
		{"Decimal 3.33333333", "Decimal 3.33333334", false},
		{"Decimal 4.4816890703380645", "Decimal 4.5816890703380645", false},
		{"Decimal 0.15311848890866597", "Decimal 0.15311848890", false},
		{"Integer 5", "Decimal 5", false},
	} {
		if got := snapshotSame(tt.want, tt.got); got != tt.same {
			t.Errorf("snapshotSame(%q, %q) = %t; want %t", tt.want, tt.got, got, tt.same)
		}
	}
}

// TestEveryCorpusIsInTheManifest holds snapshotAreas to the corpora of
// testdata/snapshots: an area left out of it is no longer evaluated nor
// compared, and every test that reads the manifest would pass without it
// (docs/plans/008-fhirpath-1-16.md, criterio de salida).
func TestEveryCorpusIsInTheManifest(t *testing.T) {
	listed := map[string]bool{}
	for _, a := range snapshotAreas {
		listed[a.area] = true
	}
	corpora, err := filepath.Glob(filepath.Join("testdata", "snapshots", "*.cql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range corpora {
		if area := strings.TrimSuffix(filepath.Base(path), ".cql"); !listed[area] {
			t.Errorf("%s is not in snapshotAreas", path)
		}
	}
}

// TestTheReferencesAreOfTheirCorpus holds each area's two references to the
// corpus they were run on: both carry its SHA-256, the Java engine answers
// every definition — a translation error is an answer — and cql-execution
// every one the translator accepted. A missing file fails rather than skips,
// so that the comparison cannot quietly lose its oracle.
func TestTheReferencesAreOfTheirCorpus(t *testing.T) {
	for _, a := range snapshotAreas {
		t.Run(a.area, func(t *testing.T) {
			corpus, err := os.ReadFile(filepath.Join("testdata", "snapshots", a.area+".cql"))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(corpus)
			want := hex.EncodeToString(sum[:])
			java, js := readReference(t, a.area, "java", want), readReference(t, a.area, "js", want)
			definitions := len(snapshotDefine.FindAllString(string(corpus), -1))
			if len(java) != definitions {
				t.Errorf("the Java engine answered %d definitions of %d", len(java), definitions)
			}
			accepted := 0
			for name, answer := range java {
				if translationRefused(answer) {
					if _, ok := js[name]; ok {
						t.Errorf("%s: cql-execution answered a definition the translator refused", name)
					}
					continue
				}
				accepted++
				if _, ok := js[name]; !ok {
					t.Errorf("%s: cql-execution did not answer a definition the translator accepted", name)
				}
			}
			if len(js) != accepted {
				t.Errorf("cql-execution answered %d definitions; the translator accepted %d", len(js), accepted)
			}
		})
	}
}

// readReference is an area's answers from one reference engine, held to the
// corpus by its SHA-256.
func readReference(t *testing.T, area, engine, sha string) map[string]json.RawMessage {
	t.Helper()
	name := area + "." + engine + ".json"
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", name))
	if err != nil {
		t.Fatalf("%v: the references are part of the corpus; see internal/referenceharness/README.md", err)
	}
	var ref struct {
		SourceSha256 string                     `json:"sourceSha256"`
		Engine       string                     `json:"engine"`
		Results      map[string]json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(raw, &ref); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if ref.SourceSha256 != sha {
		t.Errorf("%s was run on another %s.cql (%s, now %s); regenerate it", name, area, ref.SourceSha256, sha)
	}
	// The engine it was run with is the one the harness pins: an answer of any
	// other is not the reference the manifests classify against, and a
	// reference file naming another engine passed (docs/plans/002, round 1).
	if want := pinnedReferenceEngine(t, engine); ref.Engine != want {
		t.Errorf("%s was run with %q; the harness pins %q", name, ref.Engine, want)
	}
	return ref.Results
}

// pinnedReferenceEngine is the engine a reference file has to name, read from
// where the harness pins it: the Java engine in pom.xml, cql-execution and
// cql-exec-fhir in the JavaScript lock file.
func pinnedReferenceEngine(t *testing.T, engine string) string {
	t.Helper()
	dir := filepath.Join("internal", "referenceharness")
	switch engine {
	case "java":
		return pinnedJavaArtifact(t, "engine-jvm")
	case "js":
		raw, err := os.ReadFile(filepath.Join(dir, "js", "package-lock.json"))
		if err != nil {
			t.Fatal(err)
		}
		var lock struct {
			Packages map[string]struct {
				Version string `json:"version"`
			} `json:"packages"`
		}
		if err := json.Unmarshal(raw, &lock); err != nil {
			t.Fatal(err)
		}
		return "cql-execution " + lock.Packages["node_modules/cql-execution"].Version +
			", cql-exec-fhir " + lock.Packages["node_modules/cql-exec-fhir"].Version
	}
	t.Fatalf("no reference engine %q", engine)
	return ""
}

// translationRefused reports whether a Java answer is the translator's refusal.
func translationRefused(answer json.RawMessage) bool {
	var e struct {
		Error string `json:"error"`
	}
	return json.Unmarshal(answer, &e) == nil && e.Error == "Translation"
}

// snapshotElement is an element of a Code or a Concept: null, or its text
// quoted, so that the empty String and null read apart (plan 004).
func snapshotElement(s *string) string {
	if s == nil {
		return "null"
	}
	return fmt.Sprintf("%q", *s)
}

// pinnedJavaArtifact is the version of an org.cqframework artifact the harness
// pins in pom.xml, read as XML: a version in a comment is not the dependency's.
func pinnedJavaArtifact(t *testing.T, artifact string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("internal", "referenceharness", "pom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var pom struct {
		Dependencies []struct {
			GroupID    string `xml:"groupId"`
			ArtifactID string `xml:"artifactId"`
			Version    string `xml:"version"`
		} `xml:"dependencies>dependency"`
	}
	if err := xml.Unmarshal(raw, &pom); err != nil {
		t.Fatal(err)
	}
	for _, d := range pom.Dependencies {
		if d.GroupID == "org.cqframework" && d.ArtifactID == artifact {
			return d.Version
		}
	}
	t.Fatalf("pom.xml pins no org.cqframework:%s", artifact)
	return ""
}
