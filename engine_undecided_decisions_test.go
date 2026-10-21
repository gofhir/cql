package cql

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// undecidedADR is where the user decided what the specification leaves
// open: null's run-time type, a cast of null, a quotient at a whole number,
// the conversions a call takes (plan 003, PR 6) and how Mode compares (plan
// 004, PR 2).
const undecidedADR = "docs/adr/003-lo-que-la-especificacion-no-decide.md"

// cites reports whether a verdict's evidence cites a document first, the one
// TestTheDivergencesAreConsistent holds it to.
func cites(d divergence, doc string) bool {
	m := decisionCited.FindStringSubmatch(d.Evidence)
	return len(m) > 1 && m[1] == doc
}

// TestTheUndecidedADRNamesItsRows holds ADR 003 to the deliberate
// differences that cite it: it names each of them, area by area, and no
// other. The area's -classify flag writes its list.
func TestTheUndecidedADRNamesItsRows(t *testing.T) {
	raw, err := os.ReadFile(undecidedADR)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	for _, area := range []struct {
		name    string
		rewrite bool
	}{{"types", *classifyTypes}, {"fhir", *classifyFHIR}, {"arithmetic", *classifyArithmetic}, {"codesets", *classifyCodeSets}} {
		var names []string
		for key, d := range readDivergences(t, filepath.Join("testdata", "snapshots", area.name+".divergences.tsv")) {
			if name, _, _ := strings.Cut(key, "\t"); d.Class == deliberate && cites(d, undecidedADR) && !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
		slices.Sort(names)
		want := strings.Join(names, " ")
		begin := "<!-- ADR 003, " + area.name + ", written by -classify-" + area.name + ": begin -->"
		end := "<!-- ADR 003, " + area.name + ": end -->"
		b, e := strings.Index(doc, begin), strings.Index(doc, end)
		if b < 0 || e < b {
			t.Fatalf("%s has no block for the %s area: %s … %s", undecidedADR, area.name, begin, end)
		}
		if area.rewrite {
			doc = doc[:b] + begin + "\n" + want + "\n" + doc[e:]
			continue
		}
		if got := strings.TrimSpace(doc[b+len(begin) : e]); got != want {
			t.Errorf("%s names other %s rows than cite it: %q, want %q; rewrite them with -classify-%s",
				undecidedADR, area.name, got, want, area.name)
		}
	}
	if *classifyTypes || *classifyFHIR || *classifyArithmetic || *classifyCodeSets {
		if err := os.WriteFile(undecidedADR, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
