package cql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The governance of the ADRs, as NUCLEO.md §3 asks it ("ADRs que se
// cumplen"): each ADR in docs/adr declares its state and a «Verificación»
// section naming the tests or make targets that verify it, every one of which
// has to exist in what go test ./... compiles and not be skipped in its body.
// One that is «Aceptado» has to be run by CI as well. That is not read off the
// workflow — two review rounds found a reader of YAML and shell evaded by
// valid syntax each time (an `if:` after the `run:`, continue-on-error,
// `make -n`) — but required of one job, adrVerificationJob, word for word: it
// runs make evidence, which fails when a named test does not run, is skipped
// or fails, as go test -json reports it. An accepted ADR names only what that
// job runs.
//
// Taken from gofhir/searchindex's adr_test.go, which only checked that a named
// test existed somewhere in the tree: it accepted a test under testdata/ that
// go test ./... never runs, one that calls t.Skip, and a make target CI never
// runs (docs/plans/002-adr-semantica-cql.md, round 1).

var (
	// The ADRs are written in Spanish (NUCLEO.md §3): these literals match
	// their headings and states, not text of the code.
	adrStatus      = regexp.MustCompile(`(?m)^\*\*Estado:\*\*\s*(Propuesto|Aceptado|Reemplazado|Rechazado)\b`)
	adrNamedTest   = regexp.MustCompile("`?((?:Test|Fuzz|Benchmark)[A-Z][A-Za-z0-9_]*)(\\*?)`?")
	adrNamedTarget = regexp.MustCompile("`make ([a-z-]+)`")
	makefileRule   = regexp.MustCompile(`(?m)^([a-z-]+):`)
)

const adrVerificationHeading = "## Verificación"

func TestADRsHaveVerification(t *testing.T) {
	for _, problem := range adrProblems(t) {
		t.Error(problem)
	}
}

func adrProblems(t *testing.T) []string {
	t.Helper()
	adrs, err := filepath.Glob(filepath.Join("docs", "adr", "*.md"))
	if err != nil || len(adrs) == 0 {
		return []string{"no ADR in docs/adr"}
	}
	tests := testsGoTestRuns(t)
	targets := makefileTargets(t)
	ciRunsTheJob := ciHasTheVerificationJob(t)

	var problems []string
	for _, path := range adrs {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text, name := string(raw), filepath.Base(path)
		status := adrStatus.FindStringSubmatch(text)
		if status == nil {
			problems = append(problems, name+": no **Estado:** (Propuesto, Aceptado, Reemplazado or Rechazado)")
			continue
		}
		accepted := status[1] == "Aceptado"
		// One section: a second would be read by nobody, and what it names
		// would pass unchecked.
		if n := strings.Count(text, "\n"+adrVerificationHeading+"\n"); n != 1 {
			problems = append(problems, fmt.Sprintf("%s: %d «%s» sections, want one", name, n, adrVerificationHeading))
			continue
		}
		_, verification, _ := strings.Cut(text, "\n"+adrVerificationHeading+"\n")
		if next := strings.Index(verification, "\n## "); next >= 0 {
			verification = verification[:next]
		}
		if accepted && !ciRunsTheJob {
			problems = append(problems, name+": accepted, and CI does not have the ADR verification job word for word (see adrVerificationJob)")
		}
		named := 0
		for _, m := range adrNamedTest.FindAllStringSubmatch(verification, -1) {
			named++
			problems = append(problems, namedTestProblems(name, m[1], m[2] == "*", accepted, tests)...)
		}
		for _, m := range adrNamedTarget.FindAllStringSubmatch(verification, -1) {
			named++
			if !targets[m[1]] {
				problems = append(problems, name+": names make "+m[1]+", which is not in the Makefile")
			} else if accepted && !adrVerificationTargets[m[1]] {
				problems = append(problems, name+": accepted, and names make "+m[1]+", which the ADR verification job does not run")
			}
		}
		if named == 0 {
			problems = append(problems, name+": its verification names no test and no target")
		}
	}
	return problems
}

// namedTestProblems checks one test an ADR names: it exists in a file go test
// ./... compiles, and is not skipped in its own body — a skip in a helper it
// calls is make evidence's to find — and, for an accepted ADR, it is a test,
// which make evidence runs, not a benchmark or a fuzz target, which it does not.
func namedTestProblems(adr, test string, family, accepted bool, tests map[string]string) []string {
	var bodies []string
	for name, body := range tests {
		if name == test || family && strings.HasPrefix(name, test) {
			bodies = append(bodies, body)
		}
	}
	if len(bodies) == 0 {
		return []string{adr + ": names " + test + ", which no package of go test ./... has"}
	}
	var problems []string
	for _, body := range bodies {
		for _, skip := range []string{".Skip(", ".SkipNow(", ".Skipf(", "testing.Short()"} {
			if strings.Contains(body, skip) {
				problems = append(problems, adr+": names "+test+", which can be skipped ("+skip+")")
			}
		}
	}
	if accepted && !strings.HasPrefix(test, "Test") {
		problems = append(problems, adr+": accepted, and names "+test+", which make evidence does not run")
	}
	return problems
}

// testsGoTestRuns is every test, benchmark and fuzz target, with its body, in
// the test files go test ./... compiles: a test under testdata/ is in no
// package it runs, and one behind a build tag it does not set is in a file it
// leaves out, which go list reports by leaving it out of TestGoFiles.
func testsGoTestRuns(t *testing.T) map[string]string {
	t.Helper()
	out, err := exec.CommandContext(t.Context(), "go", "list", "-json", "./...").Output()
	if err != nil {
		t.Fatalf("go list ./...: %v", err)
	}
	var files []string
	decoder := json.NewDecoder(bytes.NewReader(out))
	for decoder.More() {
		var pkg struct {
			Dir                       string
			TestGoFiles, XTestGoFiles []string
		}
		if err := decoder.Decode(&pkg); err != nil {
			t.Fatalf("go list -json: %v", err)
		}
		for _, f := range append(pkg.TestGoFiles, pkg.XTestGoFiles...) {
			files = append(files, filepath.Join(pkg.Dir, f))
		}
	}
	tests := map[string]string{}
	fset := token.NewFileSet()
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := parser.ParseFile(fset, file, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil {
				continue
			}
			if n := fn.Name.Name; strings.HasPrefix(n, "Test") || strings.HasPrefix(n, "Benchmark") || strings.HasPrefix(n, "Fuzz") {
				tests[n] = string(src[fset.Position(fn.Body.Pos()).Offset:fset.Position(fn.Body.End()).Offset])
			}
		}
	}
	return tests
}

func makefileTargets(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	targets := map[string]bool{}
	for _, m := range makefileRule.FindAllStringSubmatch(string(raw), -1) {
		targets[m[1]] = true
	}
	return targets
}

// adrVerificationJob is the CI job an accepted ADR needs, word for word. It is
// compared as text, whole, from its key to the next job or the end of the
// workflow: anything added to it — an `if:`, continue-on-error, a changed
// command — makes it another job, and the ADR is not accepted.
const adrVerificationJob = `  adr-verification:
    name: ADR verification
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: The unclassified divergences only shrink
        run: sh scripts/debt-only-shrinks.sh

      - name: ADR verification
        run: make evidence conformance-provenance
`

// adrVerificationTargets are the make targets adrVerificationJob runs.
var adrVerificationTargets = map[string]bool{"evidence": true, "conformance-provenance": true}

// ciWorkflowHeader is the CI workflow from its first line to its jobs, word
// for word: the job runs only as this header triggers it, and an `on:` that
// does not trigger it on a pull request, or a `paths-ignore`, a top-level
// `env:` or `defaults:`, would leave the job intact and not run.
const ciWorkflowHeader = `name: CI

on:
  push:
    branches: [main]
  # Every pull request, not only the ones targeting main. A PR stacked on another
  # branch got no checks at all — ` + "`gh pr checks`" + ` reported "no checks reported"
  # while the PR itself showed mergeable and clean, which reads exactly like a
  # PR that passed. Two of four open PRs were in that state.
  pull_request:

env:
  GO_VERSION: "1.27"

jobs:
`

// ciHasTheVerificationJob reports whether the CI workflow has ciWorkflowHeader
// and, among its jobs, adrVerificationJob exactly, as the last job or followed
// by another.
func ciHasTheVerificationJob(t *testing.T) bool {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	workflow := strings.ReplaceAll(string(raw), "\r\n", "\n")
	jobs, ok := strings.CutPrefix(workflow, ciWorkflowHeader)
	if !ok {
		return false
	}
	// Jobs is the last key: a line at the top level after it would be another
	// key, which a job's text could be hidden in or overridden by.
	for _, line := range strings.Split(jobs, "\n") {
		if line != "" && line[0] != ' ' && line[0] != '#' {
			return false
		}
	}
	_, rest, ok := strings.Cut("\n"+jobs, "\n"+adrVerificationJob)
	if !ok {
		return false
	}
	// What follows is the next job, a key at the indent of jobs, or nothing:
	// a line more indented would belong to this one.
	for _, line := range strings.Split(rest, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		return !strings.HasPrefix(line, "   ") && strings.HasPrefix(line, "  ")
	}
	return true
}
