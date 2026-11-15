package cql

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gofhir/cql/internal/referenceharness/evaloffset"
)

// Plan 024 (docs/plans/024-same-precision-offset.md): the area `fractionaloffset`
// is evaluated at +05:30, and the references with it.

// TestEachAreaIsEvaluatedAtItsOffset holds every committed reference to the
// offset its area is evaluated at: the table of internal/referenceharness/evaloffset,
// which the three consumers of the area read, and the line of
// scripts/references.sh that tells the harness. A reference regenerated without
// the variable would carry -03:00, and the area would have compared this engine
// at +05:30 with answers given at -03:00.
func TestEachAreaIsEvaluatedAtItsOffset(t *testing.T) {
	for _, a := range snapshotAreas {
		for _, engine := range []string{"java", "js"} {
			raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", a.area+"."+engine+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var ref struct {
				EvaluationDateTime string `json:"evaluationDateTime"`
			}
			if err := json.Unmarshal(raw, &ref); err != nil {
				t.Fatalf("%s %s: %v", a.area, engine, err)
			}
			want := evaloffset.Timestamp(a.area)
			if engine == "js" {
				want = strings.Replace(want, "00:00:00", "00:00:00.000", 1)
			}
			if ref.EvaluationDateTime != want {
				t.Errorf("%s.%s.json was evaluated at %s; the area is at %s", a.area, engine, ref.EvaluationDateTime, want)
			}
		}
	}
}

// TestTheReferencesScriptTellsTheHarnessTheOffsetOfEachArea holds the offsets of
// scripts/references.sh to the table the tests read.
func TestTheReferencesScriptTellsTheHarnessTheOffsetOfEachArea(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("scripts", "references.sh"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^offsets="([^"]*)"$`).FindSubmatch(raw)
	if m == nil {
		t.Fatal(`scripts/references.sh has no offsets="..." line`)
	}
	told := map[string]string{}
	for _, entry := range strings.Fields(string(m[1])) {
		area, offset, _ := strings.Cut(entry, "=")
		told[area] = offset
	}
	for _, a := range snapshotAreas {
		want := evaloffset.Text(a.area)
		got, listed := told[a.area]
		switch {
		case want == "-03:00" && listed:
			t.Errorf("scripts/references.sh tells the harness %s for %s, which is the default", got, a.area)
		case want != "-03:00" && got != want:
			t.Errorf("scripts/references.sh tells the harness %q for %s; the area is at %s", got, a.area, want)
		}
		delete(told, a.area)
	}
	for area := range told {
		t.Errorf("scripts/references.sh tells the harness an offset for %s, which is not an area", area)
	}
	// Each loop that exports the offset of an area clears it after its last
	// area, or the evaluations that follow it inherit that area's offset (code
	// review of plan 024, round 1, finding 1).
	loops := regexp.MustCompile(`(?s)export CQL_EVALUATION_OFFSET.*?\n\t\tdone\n([^\n]*\n[^\n]*)`).FindAllSubmatch(raw, -1)
	if len(loops) != 2 {
		t.Errorf("scripts/references.sh has %d loops that export the offset; want 2, one per reference", len(loops))
	}
	for _, after := range loops {
		if !strings.Contains(string(after[1]), "unset CQL_EVALUATION_OFFSET") {
			t.Errorf("scripts/references.sh does not clear the offset after a loop of areas:\n%s", after[1])
		}
	}
	if evaloffset.Seconds("fractionaloffset") != 5*3600+30*60 {
		t.Errorf("fractionaloffset is at %d seconds, not +05:30", evaloffset.Seconds("fractionaloffset"))
	}
	if evaloffset.Seconds("mixeddates") != -3*3600 {
		t.Errorf("an area not in the table is at %d seconds, not -03:00", evaloffset.Seconds("mixeddates"))
	}
}

// TestTheFractionalOffsetAreaIsItsOracle holds every row of the enumeration of
// the area to testdata/fractionaloffset/oracle.tsv, which
// scripts/fractionaloffset-area.py writes with `--oracle` and shares nothing with
// the engine: CQL 1.5.3, Appendix B, Same As ("should normalize to the timezone
// offset of the evaluation request timestamp, but only when the comparison
// precision is hours, minutes, seconds, or milliseconds"), applied where the
// offsets differ, and the digits written where they are the same.
func TestTheFractionalOffsetAreaIsItsOracle(t *testing.T) {
	raw, err := os.ReadFile("testdata/fractionaloffset/oracle.tsv")
	if err != nil {
		t.Fatal(err)
	}
	corpus, err := os.ReadFile("testdata/snapshots/fractionaloffset.cql")
	if err != nil {
		t.Fatal(err)
	}
	asked := map[string]string{}
	for _, m := range snapshotDefine.FindAllStringSubmatch(string(corpus), -1) {
		asked[m[1]] = m[0]
	}
	got := readAnswers(t, "fractionaloffset")
	rows, trues := 0, 0
	for _, line := range strings.Split(string(raw), "\n") {
		name, want, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		def, found := asked[name]
		if !found {
			t.Errorf("%s is in the oracle and not in the area", name)
			continue
		}
		if got[name] != want {
			t.Errorf("%s: %s = %s, the oracle gives %s", name, def, got[name], want)
		}
		rows++
		if want == "Boolean true" {
			trues++
		}
	}
	if rows < 1400 {
		t.Fatalf("only %d definitions of the area were checked against the oracle", rows)
	}
	// An answer that is always one of the two checks nothing.
	if trues < rows/5 || trues > rows*4/5 {
		t.Errorf("the oracle answers true to %d of %d rows", trues, rows)
	}
}

// TestCqlExecutionTruncatesAFractionalOffset is the sentinel of a defect of the
// reference (plan 024, round 1, A3): cql-execution 3.3.2 places a DateTime with
// no offset at the whole hours of a fractional evaluation offset — +05:00 for
// +05:30 — through parseInt (lib/elm/literal.js, lib/elm/datetime.js), so
// `timezoneoffset from @2020-01-01T00:10:00` is 5 where `Now()` is 5.5. The
// area's divergences with cql-execution where a value writes no offset are
// recorded as the reference's defect on that account. If a regeneration of the
// references fixes it, this fails, and those verdicts are to be read again.
// The Java engine answers 5.5 and true.
func TestCqlExecutionTruncatesAFractionalOffset(t *testing.T) {
	corpus, err := os.ReadFile("testdata/snapshots/fractionaloffset.cql")
	if err != nil {
		t.Fatal(err)
	}
	named := map[string]string{}
	for _, m := range snapshotDefine.FindAllStringSubmatch(string(corpus), -1) {
		_, expr, _ := strings.Cut(m[0], ": ")
		named[expr] = m[1]
	}
	for _, tt := range []struct{ expr, java, js string }{
		{"timezoneoffset from Now()", "5.5", "5.5"},
		{"timezoneoffset from @2020-01-01T00:10:00", "5.5", "5"},
		{"@2020-01-01T00:10:00 same minute as @2020-01-01T00:10:00+05:30", "true", "false"},
	} {
		name, ok := named[tt.expr]
		if !ok {
			t.Fatalf("the area has no definition %q", tt.expr)
		}
		for engine, want := range map[string]string{"java": tt.java, "js": tt.js} {
			var r struct {
				Value any `json:"value"`
			}
			if err := json.Unmarshal(readReferenceResults(t, "fractionaloffset", engine)[name], &r); err != nil {
				t.Fatal(err)
			}
			got := ""
			switch v := r.Value.(type) {
			case string:
				got = v
			case bool:
				got = map[bool]string{true: "true", false: "false"}[v]
			}
			if got != want {
				t.Errorf("%s: %s answers %s for %s, the sentinel expects %s (the defect of cql-execution is gone or the harness moved)",
					name, engine, got, tt.expr, want)
			}
		}
	}
}

// TestTheHarnessReadsTheEvaluationOffset holds the two harness programs to the
// variable scripts/references.sh sets for an area at another offset, with -03:00
// where it is not set: one that ignored it would write every reference of the
// area at -03:00, and only a regeneration would show it.
func TestTheHarnessReadsTheEvaluationOffset(t *testing.T) {
	for path, want := range map[string]string{
		"internal/referenceharness/src/Evaluate.java": `System.getenv("CQL_EVALUATION_OFFSET") != null`,
		"internal/referenceharness/js/evaluate.js":    "process.env.CQL_EVALUATION_OFFSET || '-03:00'",
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), want) {
			t.Errorf("%s does not read the evaluation offset from CQL_EVALUATION_OFFSET (%s)", path, want)
		}
	}
}
