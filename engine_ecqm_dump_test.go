package cql

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/model"
	cqltypes "github.com/gofhir/cql/types"
)

// TestECQMDump writes, for every case of the eCQM corpus, the typed answer of
// every definition of the measure's library, at a fixed evaluation timestamp in
// two offsets, to the file ECQM_DUMP names. Two revisions of the engine are
// compared by diffing their dumps: TestECQMCases only holds the populations as
// true or false, so a null that became false, or a change in a definition no
// population reads, passes it (plan 018, round 2, finding 4).
func TestECQMDump(t *testing.T) {
	out := os.Getenv("ECQM_DUMP")
	root := os.Getenv("ECQM_CONTENT_DIR")
	if out == "" || root == "" {
		t.Skip("set ECQM_CONTENT_DIR and ECQM_DUMP to dump every answer of the eCQM corpus")
	}
	root = ecqmCheckoutRoot(root)
	cqlDir := filepath.Join(root, "input", "cql")
	term := loadValueSets(t, root)
	resolver := func(_ context.Context, name, _ string) (string, error) {
		b, err := os.ReadFile(filepath.Join(cqlDir, name+".cql"))
		if err != nil {
			if matches, _ := filepath.Glob(filepath.Join(cqlDir, name+"-*.cql")); len(matches) > 0 {
				b, err = os.ReadFile(matches[0])
			}
		}
		if err != nil {
			return "", fmt.Errorf("library %q: %w", name, err)
		}
		return string(b), nil
	}
	mi, err := model.LoadR4ModelInfo()
	if err != nil {
		t.Fatal(err)
	}
	mp, err := measurementPeriod("2019-01-01")
	if err != nil {
		t.Fatal(err)
	}

	var lines []string
	measureDirs, err := filepath.Glob(filepath.Join(root, "input", "tests", "measure", "*"))
	if err != nil || len(measureDirs) == 0 {
		t.Fatalf("no measure tests under %s: %v", root, err)
	}
	sort.Strings(measureDirs)
	cases := 0
	for _, md := range measureDirs {
		name := filepath.Base(md)
		measure, err := loadMeasure(filepath.Join(root, "input", "resources", "measure", name+".json"))
		if err != nil {
			// The corpus has test directories with no Measure ("Testing"): the
			// line records the omission, so a dump that lost a measure differs.
			lines = append(lines, name+"		(measure)	skipped: "+oneLineFull(err.Error()))
			continue
		}
		source, err := resolver(context.Background(), measure.library, "")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		caseDirs, _ := filepath.Glob(filepath.Join(md, "*"))
		sort.Strings(caseDirs)
		for _, cd := range caseDirs {
			if info, err := os.Stat(cd); err != nil || !info.IsDir() {
				continue
			}
			byType, patient := loadCase(cd)
			if patient == nil {
				// TestECQMCases lists these as not evaluable; the line keeps them in the dump.
				lines = append(lines, name+"/"+filepath.Base(cd)+"\t\t(case)\tskipped: no Patient resource")
				continue
			}
			cases++
			for _, offset := range []int{-3 * 3600, 14 * 3600} {
				at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", offset))
				engine := NewEngine(
					WithDataProvider(&ecqmProvider{byType: byType, term: term, model: mi}),
					WithTerminologyProvider(term),
					WithLibraryResolver(resolver),
					WithEvaluationTimestamp(at),
					WithTimeout(60*time.Second),
				)
				label := fmt.Sprintf("%s/%s\t%+d", name, filepath.Base(cd), offset/3600)
				results, err := engine.EvaluateLibrary(context.Background(), source, patient,
					map[string]fptypes.Value{"Measurement Period": mp})
				if err != nil {
					lines = append(lines, label+"\t(library)\terror: "+oneLineFull(err.Error()))
					continue
				}
				for _, def := range sortedKeys(results) {
					lines = append(lines, label+"\t"+def+"\t"+dumpValue(results[def]))
				}
			}
		}
	}
	// The corpus holds 67 cases with a Measure; fewer means the dump lost some.
	if cases < 67 || len(lines) < 2000 {
		t.Fatalf("only %d cases and %d answers dumped", cases, len(lines))
	}
	if err := os.WriteFile(out, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d answers written to %s", len(lines), out)
}

func sortedKeys(m map[string]fptypes.Value) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// dumpValue writes a value whole, with its type at every level and, for a
// DateTime, its text and the offset it is placed at, so that two answers that
// print alike but differ (a String "false" and a Boolean false, a placed and
// an unplaced DateTime) dump apart. Nothing is truncated.
func dumpValue(v fptypes.Value) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case fptypes.DateTime:
		if minutes, ok := x.EffectiveOffset(); ok {
			return fmt.Sprintf("DateTime(%s, offset %d)", strconv.Quote(x.String()), minutes)
		}
		return fmt.Sprintf("DateTime(%s, no offset)", strconv.Quote(x.String()))
	case cqltypes.List:
		parts := make([]string, len(x.Values))
		for i, e := range x.Values {
			parts[i] = dumpValue(e)
		}
		return "List[" + strings.Join(parts, ", ") + "]"
	case cqltypes.Tuple:
		keys := make([]string, 0, len(x.Elements))
		for k := range x.Elements {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = k + ": " + dumpValue(x.Elements[k])
		}
		return "Tuple{" + strings.Join(parts, ", ") + "}"
	case cqltypes.Interval:
		return fmt.Sprintf("Interval(%t %s, %s %t)", x.LowClosed, dumpValue(x.Low), dumpValue(x.High), x.HighClosed)
	case cqltypes.Ratio:
		return "Ratio(" + dumpValue(x.Numerator) + " : " + dumpValue(x.Denominator) + ")"
	case cqltypes.Code:
		return dumpCode(x)
	case cqltypes.Concept:
		parts := make([]string, len(x.Codes))
		for i, c := range x.Codes {
			parts[i] = dumpCode(c)
		}
		return "Concept([" + strings.Join(parts, ", ") + "], display " + dumpField(x.Display) + ")"
	case fptypes.Decimal:
		// The scale is part of the answer: 1.0 and 1.00 are different Decimals.
		// Exponent exists in every fhirpath version the dump compares.
		return fmt.Sprintf("Decimal(%s, exponent %d)", strconv.Quote(x.String()), x.Value().Exponent())
	case *fptypes.ObjectValue:
		return fmt.Sprintf("Object(%s, %s)", x.Type(), strconv.Quote(string(x.Data())))
	}
	return fmt.Sprintf("%T(%s)", v, strconv.Quote(v.String()))
}

// oneLineFull keeps a message on one line of the dump without cutting it.
func oneLineFull(s string) string {
	return strconv.Quote(s)
}

// dumpCode writes every field of a Code, telling a null field from an empty one.
func dumpCode(c cqltypes.Code) string {
	return "Code(system " + dumpField(c.System) + ", code " + dumpField(c.Code) +
		", version " + dumpField(c.Version) + ", display " + dumpField(c.Display) + ")"
}

func dumpField(s *string) string {
	if s == nil {
		return "null"
	}
	return strconv.Quote(*s)
}
