package cql

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"

	"github.com/gofhir/cql/eval"
	"github.com/gofhir/cql/model"
)

// Evaluate a whole published measure against the clinical data the corpus ships
// with it, and hold the result to the MeasureReport it also ships.
//
// That report is the point. Everything else in this repository is measured against
// something written here — expectations in tests, a conformance corpus of loose
// expressions — and this is the one oracle nobody here produced: it says which
// populations the reference implementation puts a patient in, for the same data.
//
// It is also the only level that can see a whole class of defect. `Check` types
// each library in isolation, without the graph of includes, so an alias always
// exists for it; the conformance corpus is expressions with no data and no
// included libraries. Four engine defects have been found here that neither saw,
// and a code review reported "no findings" on three of them:
//
//	a delimited function name in a library-qualified call (14 of the 19 measures)
//	`X years or less` limiting nothing, so a twenty-year-old screening counted
//	`sort by effective` refusing a name the same element answers to elsewhere
//	a choice element reaching a comparison unconverted
//
// Run it with ECQM_CONTENT_DIR pointing at a checkout of
// https://github.com/cqframework/ecqm-content-r4 (`git clone --depth 1`). Without
// it the test skips, which is how it behaves in CI.

type ecqmValueSet struct {
	// system|code
	members map[string]bool
}

type ecqmTerminology struct {
	sets map[string]*ecqmValueSet // url → members
}

func (t *ecqmTerminology) InValueSet(_ context.Context, code, system, url string) (bool, error) {
	vs, ok := t.sets[url]
	if !ok {
		return false, nil
	}
	if system != "" {
		return vs.members[system+"|"+code], nil
	}
	// No system given: any system with that code counts.
	for k := range vs.members {
		if strings.HasSuffix(k, "|"+code) {
			return true, nil
		}
	}
	return false, nil
}

func loadValueSets(t *testing.T, root string) *ecqmTerminology {
	t.Helper()
	out := &ecqmTerminology{sets: map[string]*ecqmValueSet{}}
	dirs := []string{
		filepath.Join(root, "input", "vocabulary", "valueset", "external"),
		filepath.Join(root, "input", "vocabulary", "valueset"),
	}
	for _, dir := range dirs {
		files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
		for _, f := range files {
			raw, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			var vs struct {
				ResourceType string `json:"resourceType"`
				URL          string `json:"url"`
				Expansion    struct {
					Contains []struct {
						System string `json:"system"`
						Code   string `json:"code"`
					} `json:"contains"`
				} `json:"expansion"`
			}
			if json.Unmarshal(raw, &vs) != nil || vs.ResourceType != "ValueSet" || vs.URL == "" {
				continue
			}
			set := out.sets[vs.URL]
			if set == nil {
				set = &ecqmValueSet{members: map[string]bool{}}
				out.sets[vs.URL] = set
			}
			for _, c := range vs.Expansion.Contains {
				set.members[c.System+"|"+c.Code] = true
			}
		}
	}
	return out
}

// ecqmProvider serves one test case's resources.
//
// It filters by code itself. The engine passes Codes through to the provider and
// does not verify what comes back — unlike DateRange and IDs, which it applies
// again — so a provider that ignores the filter makes every retrieve unfiltered,
// and a measure then counts everybody.
type ecqmProvider struct {
	byType map[string][]json.RawMessage
	term   *ecqmTerminology
	model  interface {
		PrimaryCodePath(string) string
	}
}

func (p *ecqmProvider) Retrieve(ctx context.Context, r eval.RetrieveRequest) ([]json.RawMessage, error) {
	rows := p.byType[r.ResourceType]
	if r.Codes == nil {
		return rows, nil
	}
	path := r.CodePath
	if path == "" {
		path = p.model.PrimaryCodePath(r.ResourceType)
	}
	if path == "" {
		return rows, nil
	}
	out := make([]json.RawMessage, 0, len(rows))
	for _, row := range rows {
		ok, err := p.matches(ctx, row, path, r.Codes)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, row)
		}
	}
	return out, nil
}

// matches reports whether one resource's code element satisfies the filter.
func (p *ecqmProvider) matches(ctx context.Context, row json.RawMessage, path string, codes interface{}) (bool, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(row, &obj); err != nil {
		return false, nil
	}
	for _, cc := range codingsAt(obj, path) {
		switch f := codes.(type) {
		case string: // a value set URL
			in, err := p.term.InValueSet(ctx, cc.code, cc.system, f)
			if err != nil {
				return false, err
			}
			if in {
				return true, nil
			}
		default:
			for _, want := range directCodes(codes) {
				if want.code == cc.code && (want.system == "" || want.system == cc.system) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

type codeVal struct{ system, code string }

// directCodes reads the code values a retrieve filtered by when it named them
// rather than a value set.
func directCodes(v interface{}) []codeVal {
	var out []codeVal
	switch c := v.(type) {
	case cqltypes.Code:
		out = append(out, codeVal{system: c.System, code: c.Code})
	case cqltypes.Concept:
		for _, cc := range c.Codes {
			out = append(out, codeVal{system: cc.System, code: cc.Code})
		}
	case cqltypes.List:
		for _, item := range c.Values {
			out = append(out, directCodes(item)...)
		}
	case fptypes.Collection:
		for _, item := range c {
			out = append(out, directCodes(item)...)
		}
	case []interface{}:
		for _, item := range c {
			out = append(out, directCodes(item)...)
		}
	}
	return out
}

// codingsAt reads the codings under an element, accepting the shapes FHIR uses:
// a CodeableConcept, a Coding, a bare code string, and a list of any of those.
func codingsAt(obj map[string]interface{}, path string) []codeVal {
	cur, ok := obj[path]
	if !ok {
		// A choice element: the JSON names it after its type.
		for k, v := range obj {
			if strings.HasPrefix(k, path) && len(k) > len(path) && k[len(path)] >= 'A' && k[len(path)] <= 'Z' {
				cur = v
				ok = true
				break
			}
		}
		if !ok {
			return nil
		}
	}
	return codingsOf(cur)
}

func codingsOf(v interface{}) []codeVal {
	switch x := v.(type) {
	case string:
		return []codeVal{{code: x}}
	case []interface{}:
		var out []codeVal
		for _, item := range x {
			out = append(out, codingsOf(item)...)
		}
		return out
	case map[string]interface{}:
		if codings, ok := x["coding"].([]interface{}); ok {
			var out []codeVal
			for _, c := range codings {
				out = append(out, codingsOf(c)...)
			}
			return out
		}
		sys, _ := x["system"].(string)
		code, _ := x["code"].(string)
		if code != "" {
			return []codeVal{{system: sys, code: code}}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------

type ecqmPopulation struct {
	code       string
	expression string
}

type ecqmGroup struct{ populations []ecqmPopulation }

type ecqmMeasure struct {
	name    string
	library string
	scoring string
	groups  []ecqmGroup
}

func loadMeasure(path string) (*ecqmMeasure, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m struct {
		Library []string `json:"library"`
		Scoring struct {
			Coding []struct {
				Code string `json:"code"`
			} `json:"coding"`
		} `json:"scoring"`
		Group []struct {
			Population []struct {
				Code struct {
					Coding []struct {
						Code string `json:"code"`
					} `json:"coding"`
				} `json:"code"`
				Criteria struct {
					Expression string `json:"expression"`
				} `json:"criteria"`
			} `json:"population"`
		} `json:"group"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	out := &ecqmMeasure{name: strings.TrimSuffix(filepath.Base(path), ".json")}
	if len(m.Scoring.Coding) > 0 {
		out.scoring = m.Scoring.Coding[0].Code
	}
	if len(m.Library) > 0 {
		parts := strings.Split(m.Library[0], "/")
		out.library = parts[len(parts)-1]
	}
	for _, g := range m.Group {
		var grp ecqmGroup
		for _, p := range g.Population {
			if len(p.Code.Coding) == 0 {
				continue
			}
			grp.populations = append(grp.populations, ecqmPopulation{
				code: p.Code.Coding[0].Code, expression: p.Criteria.Expression,
			})
		}
		out.groups = append(out.groups, grp)
	}
	return out, nil
}

// expected reads the oracle: the counts the corpus ships for this case.
func loadReport(path string) (period string, groups []map[string]int, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	var r struct {
		Period struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"period"`
		Group []struct {
			Population []struct {
				Code struct {
					Coding []struct {
						Code string `json:"code"`
					} `json:"coding"`
				} `json:"code"`
				Count *int `json:"count"`
			} `json:"population"`
		} `json:"group"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", nil, err
	}
	for _, g := range r.Group {
		m := map[string]int{}
		for _, p := range g.Population {
			if len(p.Code.Coding) == 0 || p.Count == nil {
				continue
			}
			m[p.Code.Coding[0].Code] = *p.Count
		}
		groups = append(groups, m)
	}
	return r.Period.Start, groups, nil
}

// measurementPeriod builds the interval the measure is evaluated over.
//
// The MeasureReport abbreviates the end as 12-31 while the measures declare the
// full year with an exclusive high bound. Taking the report's end literally
// shortens the year by a day, which is a day of data a measure can turn on.
func measurementPeriod(start string) (fptypes.Value, error) {
	ts, err := time.Parse(time.RFC3339, start)
	if err != nil {
		if ts, err = time.Parse("2006-01-02", start); err != nil {
			return nil, err
		}
	}
	lo, err := fptypes.NewDateTime(ts.Format("2006-01-02T15:04:05.000-07:00"))
	if err != nil {
		return nil, err
	}
	hi, err := fptypes.NewDateTime(ts.AddDate(1, 0, 0).Format("2006-01-02T15:04:05.000-07:00"))
	if err != nil {
		return nil, err
	}
	return cqltypes.NewInterval(lo, hi, true, false), nil
}

// ---------------------------------------------------------------------------

// ecqmCheckoutRoot reads ECQM_CONTENT_DIR whichever way it was set.
//
// Two tests here want the same checkout at different depths — this one reads
// tests, resources and vocabulary and so needs the root, while
// TestSemanticPhaseAcceptsPublishedMeasures only ever wanted input/cql, and its
// doc comment has said so since before this file existed. One variable that both
// accept beats two variables that mean almost the same thing, and beats a second
// spelling of the same rule in each test.
func ecqmCheckoutRoot(dir string) string {
	if filepath.Base(dir) == "cql" && filepath.Base(filepath.Dir(dir)) == "input" {
		return filepath.Dir(filepath.Dir(dir))
	}
	return dir
}

// knownMismatches are the cases whose oracle this engine does not reproduce, with
// the reason each one is not a defect to fix. A case named here that *does* match
// fails the test and asks to be removed: an exception list nobody prunes stops
// being a record of what is wrong and becomes a place answers go to hide.
var knownMismatches = map[string]string{
	"FHIR347/denomexcl2-EXM347": "the fixture contradicts its own oracle. It asks for the " +
		"exclusion in the group whose denominator requires NOT having an ASCVD diagnosis, " +
		"while its data carries I25.110 (atherosclerotic heart disease) — which the corpus's " +
		"own expansion of `Atherosclerosis and Peripheral Arterial Disease` contains under " +
		"exactly the system the resource uses. Its two siblings each carry the condition " +
		"matching their group (denomexcl1 a myocardial infarction, denomexcl3 a diabetes " +
		"plus an LDL of 95) and both match. No reading of this data produces this report.",
}

// knownNonEvaluable are cases without an oracle that this harness cannot run.
// Both reasons are the harness's own limits, not the engine's.
var knownNonEvaluable = map[string]string{
	"CMS111/measure-strat1-EXM111":      "measure-observation is a function with a parameter",
	"CMS111/measure-strat1-excl-EXM111": "measure-observation is a function with a parameter",
	"CMS111/measure-strat2-EXM111":      "measure-observation is a function with a parameter",
	"CMS111/measure-strat2-excl-EXM111": "measure-observation is a function with a parameter",
	"CMS111/neg-measure-EXM111":         "measure-observation is a function with a parameter",

	"PrimaryCariesPreventionasOfferedbyPCPsincludingDentistsFHIR/denom-EXM74":        "the case directory carries no Patient",
	"PrimaryCariesPreventionasOfferedbyPCPsincludingDentistsFHIR/denomexcl-EXM74":    "the case directory carries no Patient",
	"PrimaryCariesPreventionasOfferedbyPCPsincludingDentistsFHIR/numer-strat1-EXM74": "the case directory carries no Patient",
	"PrimaryCariesPreventionasOfferedbyPCPsincludingDentistsFHIR/numer-strat2-EXM74": "the case directory carries no Patient",
	"PrimaryCariesPreventionasOfferedbyPCPsincludingDentistsFHIR/numer-strat3-EXM74": "the case directory carries no Patient",
}

// minimumCasesWithAnOracle guards against measuring nothing and reporting success.
//
// The checkout is somebody else's repository and it moves, so the count is a floor
// rather than an exact figure — but a floor is what catches the failure that
// matters here: a layout change, a bad clone or a rename leaves the globs matching
// nothing, every loop body unreached, and a test that passes having compared zero
// cases. This repository has shipped a check that could not check twice already —
// a conformance harness comparing a value against itself, and a test comparing two
// syntax errors and calling them equal — and a third time is not a surprise.
//
// At the time of writing the corpus yields 24 cases with an oracle out of 72.
const minimumCasesWithAnOracle = 20

func TestECQMCases(t *testing.T) {
	root := os.Getenv("ECQM_CONTENT_DIR")
	if root == "" {
		t.Skip("set ECQM_CONTENT_DIR to a checkout of cqframework/ecqm-content-r4")
	}
	root = ecqmCheckoutRoot(root)
	cqlDir := filepath.Join(root, "input", "cql")
	term := loadValueSets(t, root)

	resolver := func(_ context.Context, name, _ string) (string, error) {
		b, err := os.ReadFile(filepath.Join(cqlDir, name+".cql"))
		if err != nil {
			// The corpus versions some file names.
			if matches, _ := filepath.Glob(filepath.Join(cqlDir, name+"-*.cql")); len(matches) > 0 {
				b, err = os.ReadFile(matches[0])
			}
		}
		if err != nil {
			return "", fmt.Errorf("library %q: %w", name, err)
		}
		return string(b), nil
	}

	measureDirs, _ := filepath.Glob(filepath.Join(root, "input", "tests", "measure", "*"))
	sort.Strings(measureDirs)

	var total, withOracle, matched, withoutOracle int
	// Every exception that was not needed, so a list nobody prunes cannot hide a
	// case that started working.
	unusedMismatch := map[string]bool{}
	for k := range knownMismatches {
		unusedMismatch[k] = true
	}
	unusedNonEvaluable := map[string]bool{}
	for k := range knownNonEvaluable {
		unusedNonEvaluable[k] = true
	}

	for _, md := range measureDirs {
		measureName := filepath.Base(md)
		measure, err := loadMeasure(filepath.Join(root, "input", "resources", "measure", measureName+".json"))
		if err != nil {
			// A directory of cases for a measure the corpus does not publish —
			// "Testing" is one — has nothing to be held to.
			t.Logf("skipping %s: no Measure resource", measureName)
			continue
		}
		source, err := resolver(context.Background(), measure.library, "")
		if err != nil {
			t.Errorf("%s names library %q, which is not in %s: %v",
				measureName, measure.library, cqlDir, err)
			continue
		}

		caseDirs, _ := filepath.Glob(filepath.Join(md, "*"))
		sort.Strings(caseDirs)
		for _, cd := range caseDirs {
			if info, err := os.Stat(cd); err != nil || !info.IsDir() {
				continue
			}
			caseName := filepath.Base(cd)
			label := measureName + "/" + caseName
			total++

			reportPath := filepath.Join(cd, "measurereport-"+caseName+".json")
			if _, statErr := os.Stat(reportPath); statErr != nil {
				// No oracle, but the data is real, and three of the four defects
				// this harness found were failures rather than wrong answers. That
				// it evaluates at all is worth holding.
				withoutOracle++
				_, runErr := runCase(cd, source, resolver, term, measure, "2019-01-01")
				why, expected := knownNonEvaluable[label]
				delete(unusedNonEvaluable, label)
				switch {
				case runErr != nil && !expected:
					t.Errorf("%s does not evaluate: %v", label, runErr)
				case runErr == nil && expected:
					t.Errorf("%s evaluates now (it was listed as %q) — remove it from "+
						"knownNonEvaluable", label, why)
				}
				continue
			}

			periodStart, wantGroups, err := loadReport(reportPath)
			if err != nil || len(wantGroups) == 0 {
				t.Errorf("%s has a MeasureReport that yields no populations: %v", label, err)
				continue
			}
			withOracle++

			if dbg := os.Getenv("ECQM_DEBUG"); dbg != "" && strings.Contains(label, dbg) {
				debugCase(t, cd, source, resolver, term, periodStart,
					strings.Split(os.Getenv("ECQM_DEBUG_DEFINES"), ","))
			}

			diff := ""
			got, err := runCase(cd, source, resolver, term, measure, periodStart)
			if err != nil {
				diff = err.Error()
			} else {
				diff = compareGroups(measure, wantGroups, got)
			}

			why, excused := knownMismatches[label]
			delete(unusedMismatch, label)
			switch {
			case diff == "" && excused:
				t.Errorf("%s matches its oracle now — remove it from knownMismatches, "+
					"where it is recorded as: %s", label, why)
			case diff == "":
				matched++
			case excused:
				t.Logf("%s does not match, as expected: %s\n    (%s)", label, why, diff)
			default:
				t.Errorf("%s does not match its oracle: %s", label, diff)
			}
		}
	}

	for label := range unusedMismatch {
		t.Errorf("knownMismatches names %q, which the corpus no longer has", label)
	}
	for label := range unusedNonEvaluable {
		t.Errorf("knownNonEvaluable names %q, which the corpus no longer has", label)
	}

	if withOracle < minimumCasesWithAnOracle {
		t.Fatalf("only %d cases carry a MeasureReport, want at least %d — a harness that "+
			"compares nothing must fail rather than report success. Check that %s is a "+
			"complete checkout of cqframework/ecqm-content-r4",
			withOracle, minimumCasesWithAnOracle, root)
	}
	t.Logf("%d cases: %d with an oracle (%d matched, %d excused), %d without",
		total, withOracle, matched, withOracle-matched, withoutOracle)
}

// scoreGroup turns the raw truth of each population expression into the counts a
// MeasureReport carries, which are not the same thing.
//
// A measure's populations are nested, and the corpus's own oracle says so
// plainly. FHIR347 has three groups sharing one Numerator expression, and for a
// patient the expression holds for, the report counts the numerator in exactly
// the group whose denominator they are in:
//
//	numer1   g0 {ip 1, denominator 1, numerator 1}
//	         g1 {ip 1, denominator 0, numerator 0}   ← same Numerator, still 0
//
// and the reported denominator has both the exclusions and the exceptions taken
// out of it:
//
//	denomexcl1   g0 {denominator 0, denominator-exclusion 1}
//	denomexcpt1  g0 {denominator 0, denominator-exception 1}
//
// Evaluating each expression on its own and reporting that was the harness
// measuring something other than what the oracle reports — the fourth defect of
// that kind here, after the flattened groups, the shortened measurement period
// and the missing value set filtering.
func scoreGroup(scoring string, raw map[string]bool) map[string]int {
	out := map[string]int{}
	num := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	ip := raw["initial-population"]
	out["initial-population"] = num(ip)

	switch scoring {
	case "cohort":
		return out
	case "continuous-variable":
		pop := ip && raw["measure-population"]
		excl := pop && raw["measure-population-exclusion"]
		out["measure-population"] = num(pop && !excl)
		out["measure-population-exclusion"] = num(excl)
		return out
	}

	// proportion, and ratio, which nests the same way.
	inDenom := ip && raw["denominator"]
	exclusion := inDenom && raw["denominator-exclusion"]
	eligible := inDenom && !exclusion
	numerator := eligible && raw["numerator"]
	numExclusion := numerator && raw["numerator-exclusion"]
	numerator = numerator && !numExclusion
	// An exception only applies to someone the numerator did not already claim.
	exception := eligible && !numerator && raw["denominator-exception"]

	out["denominator"] = num(eligible && !exception)
	out["denominator-exclusion"] = num(exclusion)
	out["denominator-exception"] = num(exception)
	out["numerator"] = num(numerator)
	out["numerator-exclusion"] = num(numExclusion)
	return out
}

func compareGroups(m *ecqmMeasure, want []map[string]int, raw []map[string]bool) string {
	var diffs []string
	for gi := range m.groups {
		if gi >= len(want) || gi >= len(raw) {
			break
		}
		got := scoreGroup(m.scoring, raw[gi])
		codes := make([]string, 0, len(want[gi]))
		for code := range want[gi] {
			codes = append(codes, code)
		}
		sort.Strings(codes)
		for _, code := range codes {
			g, reported := got[code]
			if !reported {
				continue
			}
			if g != want[gi][code] {
				diffs = append(diffs, fmt.Sprintf("g%d.%s want %d got %d", gi, code, want[gi][code], g))
			}
		}
	}
	return strings.Join(diffs, ", ")
}

// loadCase reads one case directory: the resources it serves, grouped by type,
// and the Patient the evaluation is scoped to.
func loadCase(caseDir string) (byType map[string][]json.RawMessage, patient json.RawMessage) {
	byType = map[string][]json.RawMessage{}
	typeDirs, _ := filepath.Glob(filepath.Join(caseDir, "*"))
	for _, td := range typeDirs {
		if info, err := os.Stat(td); err != nil || !info.IsDir() {
			continue
		}
		files, _ := filepath.Glob(filepath.Join(td, "*.json"))
		for _, f := range files {
			raw, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			var envelope struct {
				ResourceType string `json:"resourceType"`
			}
			if json.Unmarshal(raw, &envelope) != nil {
				continue
			}
			byType[envelope.ResourceType] = append(byType[envelope.ResourceType], raw)
			if envelope.ResourceType == "Patient" && patient == nil {
				patient = raw
			}
		}
	}
	return byType, patient
}

// evaluateCase runs the measure's library over one case's data.
func evaluateCase(caseDir, source string, resolver LibraryResolver,
	term *ecqmTerminology, periodStart string) (map[string]fptypes.Value, error) {
	byType, patient := loadCase(caseDir)
	if patient == nil {
		return nil, fmt.Errorf("no Patient resource")
	}
	mi, err := model.LoadR4ModelInfo()
	if err != nil {
		return nil, err
	}
	provider := &ecqmProvider{byType: byType, term: term, model: mi}
	mp, err := measurementPeriod(periodStart)
	if err != nil {
		return nil, err
	}
	engine := NewEngine(
		WithDataProvider(provider),
		WithTerminologyProvider(term),
		WithLibraryResolver(resolver),
		WithTimeout(60*time.Second),
	)
	return engine.EvaluateLibrary(context.Background(), source, patient,
		map[string]fptypes.Value{"Measurement Period": mp})
}

func runCase(
	caseDir, source string,
	resolver LibraryResolver, term *ecqmTerminology,
	measure *ecqmMeasure, periodStart string,
) ([]map[string]bool, error) {
	results, err := evaluateCase(caseDir, source, resolver, term, periodStart)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]bool, len(measure.groups))
	for gi, grp := range measure.groups {
		out[gi] = map[string]bool{}
		for _, pop := range grp.populations {
			v, ok := results[pop.expression]
			if !ok {
				return nil, fmt.Errorf("no result for %q", pop.expression)
			}
			out[gi][pop.code] = isTruthy(v)
		}
	}
	return out, nil
}

func isTruthy(v fptypes.Value) bool {
	switch x := v.(type) {
	case nil:
		return false
	case fptypes.Boolean:
		return x.Bool()
	case cqltypes.List:
		return x.Values.Count() > 0
	}
	return true
}

// debugCase prints named definitions for one case, which is how each of the
// engine defects above was narrowed from "this case does not match" to a line of
// CQL. Driven by ECQM_DEBUG=<substring of the case label> and
// ECQM_DEBUG_DEFINES=<comma-separated define names>.
func debugCase(t *testing.T, caseDir, source string, resolver LibraryResolver,
	term *ecqmTerminology, periodStart string, names []string) {
	t.Helper()
	results, err := evaluateCase(caseDir, source, resolver, term, periodStart)
	if err != nil {
		t.Logf("  DEBUG error: %v", err)
		return
	}
	for _, n := range names {
		if n = strings.TrimSpace(n); n == "" {
			continue
		}
		v, ok := results[n]
		switch {
		case !ok:
			t.Logf("  DEBUG %-58s <no such define>", n)
		case v == nil:
			t.Logf("  DEBUG %-58s null", n)
		default:
			out := v.String()
			if len(out) > 160 {
				out = out[:160] + "…"
			}
			t.Logf("  DEBUG %-58s %s", n, out)
		}
	}
}
