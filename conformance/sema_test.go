package conformance

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofhir/cql/compiler"
	"github.com/gofhir/cql/sema"
)

// TestSemanticPhaseAcceptsEverythingTheEngineEvaluates holds the semantic phase
// to the standard that decides whether it is usable: it must not refuse a
// library that works.
//
// A checker that reports errors on correct code is worse than no checker,
// because the first false alarm teaches everyone to ignore the rest. So this
// runs the phase over every expression in the conformance corpus that upstream
// considers valid — a few over 1,700 of them — and requires silence.
//
// It is not a measure of how much the phase finds. The corpus is a test of
// evaluation semantics, and the 39 expressions it marks invalid are invalid at
// evaluation, not statically: Exp(1000) overflows, Ln(0) is undefined,
// `successor of` the last representable DateTime has nowhere to go. None of
// them is a type error, and a phase that reported them would be guessing.
func TestSemanticPhaseAcceptsEverythingTheEngineEvaluates(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("testdata/*.xml")
	if err != nil {
		t.Fatalf("globbing test files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no XML test files found in testdata/")
	}

	checked := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading %s: %v", file, err)
		}
		var suite TestSuite
		if err := xml.Unmarshal(data, &suite); err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}

		for _, group := range suite.Groups {
			for _, tc := range group.Tests {
				expr := strings.TrimSpace(tc.Expression.Value)
				if expr == "" {
					continue
				}
				if invalid := tc.Expression.Invalid; invalid != "" && invalid != "false" {
					continue
				}
				lib, err := compiler.Compile(wrapExpression(expr))
				if err != nil {
					// A parse failure is the parser's business, and some of
					// these expressions are meant to have one.
					continue
				}
				checked++
				// No model: these expressions use none, and passing one would
				// test the adapter rather than the rules.
				if diags := sema.Check(lib, nil).Diagnostics.Errors(); len(diags) > 0 {
					t.Errorf("%s\n  %s\n  %s", tc.Name, expr, diags.Error())
				}
			}
		}
	}

	if checked < 1500 {
		t.Fatalf("only %d expressions checked; the corpus has over 1,700, so "+
			"something stopped this from reading it", checked)
	}
}
