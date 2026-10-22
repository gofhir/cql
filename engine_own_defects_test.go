package cql

import (
	"bufio"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

var updateOwnDefects = flag.Bool("update-own-defects", false,
	"rewrite testdata/snapshots/*.own-defects.tsv from the oracles")

// engineDefect is a definition where this engine does not give the answer its
// area's oracle computes, with the engine's answer pinned, as the file holds it.
type engineDefect struct {
	Expected, Engine, Reason string
}

const ownDefectsHeader = "# name\texpected\tengine answer\treason — see TestTheEngineFollowsTheOracles\n"

// ownDefectAreas are the areas whose oracle decides definitions, each with what
// it finds wrong in this engine over the whole corpus, and how many definitions
// it decided. The scope area holds its engine to its oracle on its own
// (TestTheEngineFollowsTheScopeOracle).
var ownDefectAreas = []struct {
	area string
	find func(t *testing.T) (defects map[string]engineDefect, decided int)
}{
	{"types", typesOwnDefects},
	{"fhir", fhirOwnDefects},
	{"arithmetic", arithmeticOwnDefects},
	{"codes", func(t *testing.T) (map[string]engineDefect, int) { return codesOwnDefects(t, "codes") }},
	{"codesets", func(t *testing.T) (map[string]engineDefect, int) { return codesOwnDefects(t, "codesets") }},
	{"membership", membershipOwnDefects},
	{"fhirmembership", fhirMembershipOwnDefects},
}

// TestTheEngineFollowsTheOracles holds this engine to each area's oracle on
// every definition the oracle decides, without going through the references:
// the definitions where the engine is wrong are exactly the ones its
// *.own-defects.tsv names, with the answer recorded there. A divergence row
// is about a pair of answers, and when a reference changes its answer to this
// engine's, the row goes and the defect with it (plan 007, fhir D240); this
// file keeps it. -update-own-defects rewrites the files.
func TestTheEngineFollowsTheOracles(t *testing.T) {
	for _, a := range ownDefectAreas {
		t.Run(a.area, func(t *testing.T) {
			found, decided := a.find(t)
			if decided == 0 {
				t.Fatal("the oracle decides no definition: the engine is held to nothing")
			}
			path := filepath.Join("testdata", "snapshots", a.area+".own-defects.tsv")
			if *updateOwnDefects {
				writeOwnDefects(t, path, found)
				return
			}
			recorded := readOwnDefects(t, path)
			for _, name := range slices.Sorted(maps.Keys(found)) {
				r, ok := recorded[name]
				switch {
				case !ok:
					t.Errorf("%s: this engine answers %s, the oracle expects %s (%s), and %s does not record it",
						name, oneLine(found[name].Engine), oneLine(found[name].Expected), found[name].Reason, path)
				case !sameDefect(r, found[name], exactArea(a.area)):
					t.Errorf("%s: recorded %s expecting %s; now %s expecting %s (%s)", name,
						oneLine(r.Engine), oneLine(r.Expected), oneLine(found[name].Engine), oneLine(found[name].Expected), found[name].Reason)
				}
			}
			for _, name := range slices.Sorted(maps.Keys(recorded)) {
				if _, ok := found[name]; !ok {
					t.Errorf("%s: %s records a defect the oracle no longer finds; remove it", name, path)
				}
			}
		})
	}
}

// sameDefect reports whether a defect found is the one recorded: the engine's
// answer within the tolerance its snapshot has for a float64's last digits,
// in an area that is not exact, since Exp writes another last digit on amd64
// than on arm64.
func sameDefect(recorded, found engineDefect, exact bool) bool {
	return recorded.Expected == found.Expected && recorded.Reason == found.Reason &&
		(recorded.Engine == found.Engine || !exact && snapshotSame(recorded.Engine, found.Engine))
}

// exactArea reports whether the snapshot of an area compares its answers as
// text.
func exactArea(area string) bool {
	for _, a := range snapshotAreas {
		if a.area == area {
			return a.exact
		}
	}
	panic("no snapshot area " + area)
}

func typesOwnDefects(t *testing.T) (defects map[string]engineDefect, decided int) {
	defs := typesDefinitions(t)
	engine := snapshotEngineAnswers(t, "types")
	out, decided := map[string]engineDefect{}, 0
	for name, expr := range defs {
		v, err := typesExpected(expr)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if v.undecided {
			continue
		}
		decided++
		if got := pinned(engine[name].Text); !expectedMatches(v.expected, got) {
			out[name] = engineDefect{v.expected, got, v.reason}
		}
	}
	return out, decided
}

func fhirOwnDefects(t *testing.T) (defects map[string]engineDefect, decided int) {
	defs := fhirDefinitions(t)
	c := readFHIRCorpus(t)
	engine := fhirEngineAnswers(t, defs)
	out := map[string]engineDefect{}
	for name, expr := range defs {
		v, err := c.fhirExpected(expr)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if engine[name] != v.expected {
			out[name] = engineDefect{v.expected, pinned(engine[name]), v.reason}
		}
	}
	return out, len(defs)
}

func arithmeticOwnDefects(t *testing.T) (defects map[string]engineDefect, decided int) {
	defs := arithmeticDefinitions(t)
	engine := snapshotAnswers(t, "arithmetic")
	out := map[string]engineDefect{}
	for name, expr := range defs {
		v, err := arithmeticOracle(expr)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !follows(readEngineArithmetic(engine[name]), v) {
			out[name] = engineDefect{v.policy.String(), pinned(engine[name]), "the scale policy gives " + v.policy.String()}
		}
	}
	return out, len(defs)
}

func codesOwnDefects(t *testing.T, area string) (defects map[string]engineDefect, decided int) {
	defs, declared := codesCorpus(t, area)
	engine := snapshotEngineAnswers(t, area)
	out := map[string]engineDefect{}
	for name, expr := range defs {
		v, err := codesExpected(declared, expr)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := pinned(engine[name].Text); !expectedMatches(v.expected, got) {
			out[name] = engineDefect{v.expected, got, v.reason}
		}
	}
	return out, len(defs)
}

func membershipOwnDefects(t *testing.T) (defects map[string]engineDefect, decided int) {
	defs := membershipCorpus(t)
	engine := snapshotEngineAnswers(t, "membership")
	out := map[string]engineDefect{}
	for name, expr := range defs {
		v, err := membershipExpected(expr)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := pinned(engine[name].Text); !expectedMatches(v.expected, got) {
			out[name] = engineDefect{v.expected, got, v.reason}
		}
	}
	return out, len(defs)
}

func fhirMembershipOwnDefects(t *testing.T) (defects map[string]engineDefect, decided int) {
	defs, nodes, data := fhirMembershipCorpus(t)
	engine := snapshotEngineAnswers(t, "fhirmembership")
	out := map[string]engineDefect{}
	for name, expr := range defs {
		v, err := fhirMembershipExpected(data, expr, nodes[name])
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := pinned(engine[name].Text); !expectedMatches(v.expected, got) {
			out[name] = engineDefect{v.expected, got, v.reason}
		}
	}
	return out, len(defs)
}

func readOwnDefects(t *testing.T, path string) map[string]engineDefect {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	out := map[string]engineDefect{}
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 0, 1<<16), 1<<22)
	for s.Scan() {
		if strings.HasPrefix(s.Text(), "#") {
			continue
		}
		fields := strings.Split(s.Text(), "\t")
		if len(fields) != 4 {
			t.Fatalf("%s: %q has %d fields, want 4", path, s.Text(), len(fields))
		}
		if _, dup := out[fields[0]]; dup {
			t.Fatalf("%s: %s is recorded twice", path, fields[0])
		}
		out[fields[0]] = engineDefect{fields[1], fields[2], fields[3]}
	}
	if err := s.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func writeOwnDefects(t *testing.T, path string, defects map[string]engineDefect) {
	t.Helper()
	var b strings.Builder
	b.WriteString(ownDefectsHeader)
	for _, name := range slices.Sorted(maps.Keys(defects)) {
		d := defects[name]
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\n", name, d.Expected, d.Engine, d.Reason)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}
