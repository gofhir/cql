package cql

import (
	"context"
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
	"time"

	"github.com/shopspring/decimal"

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
				writeDivergences(t, path, got, recorded, a.exact)
				return
			}
			var problems []string
			for _, key := range slices.Sorted(maps.Keys(got)) {
				r, ok := recorded[key]
				switch {
				case !ok:
					problems = append(problems, "new: "+key+": "+got[key].String())
				case !samePins(r.answerPair, got[key], a.exact):
					// The verdict was given about two answers; another answer
					// is not the divergence it decided.
					problems = append(problems, fmt.Sprintf("changed: %s: was %s, now %s", key, r.answers().String(), got[key].String()))
				}
			}
			for _, key := range slices.Sorted(maps.Keys(recorded)) {
				if _, ok := got[key]; !ok {
					problems = append(problems, "gone: "+key+" ("+recorded[key].Class+")")
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
func referenceDivergences(t *testing.T, area string, exact bool) map[string]answerPair {
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
	divergences := map[string]answerPair{}
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
			divergences[name+"\t"+engineName] = answerPair{Reference: pinned(reference.Text), Engine: pinned(answer.Text)}
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

// answerPair is the two answers a divergence is between: the reference's and
// this engine's, each on one line and whole, so that a verdict about them is
// held to exactly them.
type answerPair struct{ Reference, Engine string }

func (p answerPair) String() string { return p.Reference + " vs " + p.Engine }

// The classes a divergence is given (NUCLEO.md §10: a defect of this engine, a
// defect of the reference, or a deliberate difference), and the two states of
// one not settled yet: undecided, which the specification does not decide and
// the user has to, with its two readings; and unclassified, the debt.
const (
	ownDefect       = "own defect"
	referenceDefect = "reference defect"
	deliberate      = "deliberate"
	undecided       = "undecided"
	unclassified    = "unclassified"
)

// divergence is one line of a manifest: the verdict about a divergence, the
// answer the specification expects, the evidence for both, and the two
// answers the verdict was given about.
type divergence struct {
	Class, Expected, Evidence string
	answerPair
}

func (d divergence) answers() answerPair { return d.answerPair }

const divergencesHeader = "# name\treference\tclass\texpected\tevidence\treference answer\tengine answer" +
	" — see TestAnswersMatchTheReferences and TestTheDivergencesAreConsistent\n"

func readDivergences(t *testing.T, path string) map[string]divergence {
	t.Helper()
	recorded := map[string]divergence{}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) && *updateDivergences {
		return recorded
	}
	if err != nil {
		t.Fatalf("%v; write it with go test -run TestAnswersMatchTheReferences -update-divergences", err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 7 {
			t.Fatalf("%s: a line that is not name, reference, class, expected, evidence and the two answers: %q", path, oneLine(line))
		}
		key := fields[0] + "\t" + fields[1]
		if _, dup := recorded[key]; dup {
			t.Fatalf("%s: %s is recorded twice", path, oneLine(key))
		}
		recorded[key] = divergence{Class: fields[2], Expected: fields[3], Evidence: fields[4],
			answerPair: answerPair{Reference: fields[5], Engine: fields[6]}}
	}
	return recorded
}

// writeDivergences writes a manifest of the divergences found. A divergence
// whose two answers are the ones its verdict was given about keeps it; one
// whose answers changed, and a new one, are unclassified, to be settled again.
func writeDivergences(t *testing.T, path string, found map[string]answerPair, recorded map[string]divergence, exact bool) {
	t.Helper()
	var b strings.Builder
	b.WriteString(divergencesHeader)
	for _, key := range slices.Sorted(maps.Keys(found)) {
		d, ok := recorded[key]
		if !ok || !samePins(d.answerPair, found[key], exact) {
			d = divergence{Class: unclassified, answerPair: found[key]}
			if ok {
				d.Evidence = "answers changed from " + recorded[key].String()
			}
		}
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\t%s\n", key, d.Class, d.Expected, d.Evidence, d.Reference, d.Engine)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// namesWord reports whether doc names name as a word of its own: D1 is not
// named by a document that names D11.
func namesWord(doc []byte, name string) bool {
	return regexp.MustCompile(`(?:^|[^A-Za-z0-9])` + regexp.QuoteMeta(name) + `(?:$|[^A-Za-z0-9])`).Match(doc)
}

// pinned is an answer as a manifest pins it: whole and on one line, its
// backslashes, tabs and line breaks escaped. oneLine cuts it to read in a
// message, and collapsing its spaces made "a b" and "a  b" one answer; a
// verdict pinned to either would not see the other.
func pinned(s string) string {
	return strings.NewReplacer(`\`, `\\`, "\t", `\t`, "\n", `\n`, "\r", `\r`).Replace(s)
}

// samePins reports whether two pinned answers are the one the verdict was
// given about. They are the same text, or, in an area that is not exact, the
// same Decimal within the tolerance its comparison has for a float64's last
// digits: Exp and Power write another last digit on amd64 than on arm64, and
// a pin of one machine would fail on the other where the comparison itself
// does not.
func samePins(recorded, found answerPair, exact bool) bool {
	return samePin(recorded.Reference, found.Reference, exact) && samePin(recorded.Engine, found.Engine, exact)
}

func samePin(a, b string, exact bool) bool {
	if a == b {
		return true
	}
	if exact {
		return false
	}
	da, okA := pinnedDecimal(a)
	db, okB := pinnedDecimal(b)
	return okA && okB && answers.Same(da, db)
}

// pinnedDecimal reads a pinned answer that is one Decimal back as the answer it
// was written from.
func pinnedDecimal(pin string) (answers.Answer, bool) {
	text, ok := strings.CutPrefix(pin, "Decimal ")
	if !ok {
		return answers.Answer{}, false
	}
	d, err := decimal.NewFromString(text)
	if err != nil {
		return answers.Answer{}, false
	}
	return answers.Answer{Kind: "number", Text: pin, Num: d}, true
}

// TestTheDivergencesAreConsistent holds each verdict to what its class means.
// An own defect expects an answer other than this engine's; a reference defect
// expects this engine's, which the reference's is not; a deliberate difference
// cites the decision that made it; an undecided one records its two readings;
// and anything but the debt says what its evidence is.
func TestTheDivergencesAreConsistent(t *testing.T) {
	// A decision is a file of the repository, named whole: an ADR, or the plan
	// whose approval made it. A prefix like "docs/plans/003" named a plan that
	// decides nothing, and a name that does not exist passed.
	// It is cited as a path of the repository, not inside a URL, and the file
	// names the divergence it decided: a document that exists and decides
	// something else is not the decision.
	decision := regexp.MustCompile(`(?:^|[\s(])(docs/(?:adr|plans)/\d{3}-[a-z0-9-]+\.md)`)
	// An undecided divergence records the two readings the specification
	// leaves, each with the file of the repository it comes from. The fields
	// are delimited by brackets and a bar, which answers do not hold: a
	// parenthesis or a semicolon inside an answer is no delimiter.
	readings := regexp.MustCompile(`^reading 1: (.+?) \[source: ([^\]\s]+)\] \| reading 2: (.+?) \[source: ([^\]\s]+)\]`)
	for _, a := range snapshotAreas {
		for key, d := range readDivergences(t, filepath.Join("testdata", "snapshots", a.area+".divergences.tsv")) {
			where := a.area + " " + oneLine(key)
			if d.Class != unclassified && strings.TrimSpace(d.Evidence) == "" {
				t.Errorf("%s: %s without evidence", where, d.Class)
			}
			switch d.Class {
			case unclassified:
			case ownDefect:
				if d.Expected == "" || expectedMatches(d.Expected, d.Engine) {
					t.Errorf("%s: an own defect expects an answer other than this engine's (%s)", where, d.Engine)
				}
			case referenceDefect:
				if !expectedMatches(d.Expected, d.Engine) || expectedMatches(d.Expected, d.Reference) {
					t.Errorf("%s: a reference defect expects this engine's answer, %s, and not the reference's, %s; it expects %s",
						where, d.Engine, d.Reference, d.Expected)
				}
			case deliberate:
				m := decision.FindStringSubmatch(d.Evidence)
				if m == nil {
					t.Errorf("%s: a deliberate difference cites the ADR or plan that decided it, by its whole name", where)
				} else if doc, err := os.ReadFile(m[1]); err != nil {
					t.Errorf("%s: a deliberate difference cites %s, which is not in the repository", where, m[1])
				} else if name, _, _ := strings.Cut(key, "\t"); !namesWord(doc, name) {
					t.Errorf("%s: a deliberate difference cites %s, which does not name %s", where, m[1], name)
				}
				if !expectedMatches(d.Expected, d.Engine) {
					t.Errorf("%s: a deliberate difference expects this engine's answer, %s, which is the one decided", where, d.Engine)
				}
			case undecided:
				if d.Expected != "" {
					t.Errorf("%s: an undecided divergence expects nothing yet", where)
				}
				m := readings.FindStringSubmatch(d.Evidence)
				if len(m) != 5 || strings.TrimSpace(m[1]) == "" || strings.TrimSpace(m[3]) == "" {
					t.Errorf("%s: an undecided divergence records its two readings, each with its source: "+
						"\"reading 1: … [source: path] | reading 2: … [source: path]\"", where)
					break
				}
				name, _, _ := strings.Cut(key, "\t")
				for _, source := range []string{m[2], m[4]} {
					// A file of the corpora, named relative to the repository,
					// that answers the divergence: the snapshot or a reference.
					doc, err := os.ReadFile(source)
					switch {
					case filepath.Dir(source) != filepath.Join("testdata", "snapshots") || filepath.Clean(source) != source:
						t.Errorf("%s: a reading cites %s, which is not a file of testdata/snapshots", where, source)
					case err != nil:
						t.Errorf("%s: a reading cites %s, which is not in the repository", where, source)
					case !namesWord(doc, name):
						t.Errorf("%s: a reading cites %s, which does not answer %s", where, source, name)
					}
				}
			default:
				t.Errorf("%s: %q is no class (%s, %s, %s, %s or %s)", where, d.Class,
					ownDefect, referenceDefect, deliberate, undecided, unclassified)
			}
		}
	}
}

// TestThePendingVerdictsDoNotGrow holds the divergences without a verdict —
// unclassified, or undecided until the user decides them — to the debt
// recorded in testdata/snapshots/pending-verdicts.tsv: every one of them has
// to be an entry there, and every entry has to be still pending, so an entry
// can only be removed, when its divergence gets a verdict.
// -update-divergences writes a new divergence as unclassified on its own,
// which would join the debt unseen; a count let one row be settled while
// another, settled before, was unsettled in its place; and moving a row to
// undecided was a way out of the debt that needed no verdict.
func TestThePendingVerdictsDoNotGrow(t *testing.T) {
	debt := readUnclassifiedDebt(t)
	pending := map[string]bool{}
	for _, a := range snapshotAreas {
		for key, d := range readDivergences(t, filepath.Join("testdata", "snapshots", a.area+".divergences.tsv")) {
			if d.Class == unclassified || d.Class == undecided {
				pending[a.area+"\t"+key] = true
			}
		}
	}
	for _, key := range slices.Sorted(maps.Keys(pending)) {
		if !debt[key] {
			t.Errorf("%s has no verdict and is not in pending-verdicts.tsv: settle it in its manifest", oneLine(key))
		}
	}
	for _, key := range slices.Sorted(maps.Keys(debt)) {
		if !pending[key] {
			t.Errorf("%s is in pending-verdicts.tsv and has a verdict now: remove it from the debt", oneLine(key))
		}
	}
}

func readUnclassifiedDebt(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", "pending-verdicts.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "\r") {
		t.Fatal("pending-verdicts.tsv has CRLF line endings: write it with LF")
	}
	debt := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Count(line, "\t") != 2 {
			t.Fatalf("pending-verdicts.tsv: %q is not area, name and reference", line)
		}
		if debt[line] {
			t.Errorf("pending-verdicts.tsv lists %s twice", oneLine(line))
		}
		debt[line] = true
	}
	return debt
}

// TestAChangedAnswerLosesItsVerdict covers -update-divergences over a
// divergence whose answers moved: the verdict was given about the old two, so
// it is not carried to the new ones, which are unclassified until settled
// again; a divergence whose answers did not move keeps its verdict.
func TestAChangedAnswerLosesItsVerdict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "area.divergences.tsv")
	kept := answerPair{Reference: "null", Engine: "Integer 1"}
	recorded := map[string]divergence{
		"D1\tjava": {Class: referenceDefect, Expected: "Integer 1", Evidence: "kept", answerPair: kept},
		"D2\tjava": {Class: referenceDefect, Expected: "Integer 2", Evidence: "moved", answerPair: answerPair{Reference: "null", Engine: "Integer 2"}},
	}
	writeDivergences(t, path, map[string]answerPair{"D1\tjava": kept, "D2\tjava": {Reference: "null", Engine: "Integer 3"}}, recorded, true)
	got := readDivergences(t, path)
	if got["D1\tjava"].Class != referenceDefect || got["D1\tjava"].Evidence != "kept" {
		t.Errorf("D1 = %+v; its answers did not move, so its verdict stays", got["D1\tjava"])
	}
	if d := got["D2\tjava"]; d.Class != unclassified || d.Expected != "" || d.Engine != "Integer 3" {
		t.Errorf("D2 = %+v; its answers moved, so it is unclassified, about the new ones", d)
	}
}

// TestAPinIsTheSameAcrossPlatforms covers a verdict pinned on one machine and
// checked on another: a Decimal a float64 computed writes another last digit
// on amd64 than on arm64, and is the same pin in an area that is not exact,
// as the comparison takes it; in an exact one, and for anything but a
// Decimal, only the same text is.
func TestAPinIsTheSameAcrossPlatforms(t *testing.T) {
	for _, tt := range []struct {
		a, b  string
		exact bool
		same  bool
	}{
		{"Decimal 1907346572495099500000", "Decimal 1907346572495099800000", false, true},
		{"Decimal 1907346572495099500000", "Decimal 1907346572495099800000", true, false},
		{"Decimal 3.33333333", "Decimal 3.33333334", false, false},
		{"Integer 1907346572495099500000", "Integer 1907346572495099800000", false, false},
		{`"a b"`, `"a  b"`, false, false},
		{`"a\tb"`, `"a b"`, false, false},
	} {
		if got := samePin(tt.a, tt.b, tt.exact); got != tt.same {
			t.Errorf("samePin(%s, %s, exact %t) = %t; want %t", tt.a, tt.b, tt.exact, got, tt.same)
		}
	}
	if pinned("a\tb\nc\\") != `a\tb\nc\\` {
		t.Errorf("pinned escapes tabs, line breaks and backslashes; got %q", pinned("a\tb\nc\\"))
	}
}
