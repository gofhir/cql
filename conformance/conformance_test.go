package conformance

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cql "github.com/gofhir/cql"
	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
)

// wrapExpression wraps a raw CQL expression in a minimal library so the engine
// can evaluate it as a named definition called "result".
func wrapExpression(expr string) string {
	return fmt.Sprintf("library ConformanceTest version '1.0'\ndefine \"result\": %s", expr)
}

// valuesEqual compares two fptypes.Value instances for equality.
func valuesEqual(got, want fptypes.Value) bool {
	if got == nil && want == nil {
		return true
	}
	if got == nil || want == nil {
		return false
	}
	return got.Equal(want)
}

func TestConformance(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("testdata/*.xml")
	if err != nil {
		t.Fatalf("globbing test files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no XML test files found in testdata/")
	}

	engine := cql.NewEngine()

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading %s: %v", file, err)
		}

		var suite TestSuite
		if err := xml.Unmarshal(data, &suite); err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}

		suiteName := suite.Name
		if suiteName == "" {
			suiteName = strings.TrimSuffix(filepath.Base(file), ".xml")
		}

		t.Run(suiteName, func(t *testing.T) {
			t.Parallel()
			for _, group := range suite.Groups {
				t.Run(group.Name, func(t *testing.T) {
					t.Parallel()
					for _, tc := range group.Tests {
						testName := tc.Name
						if testName == "" {
							testName = strings.TrimSpace(tc.Expression.Value)
							if len(testName) > 60 {
								testName = testName[:60]
							}
						}
						t.Run(testName, func(t *testing.T) {
							t.Parallel()
							runTestCase(t, engine, tc)
						})
					}
				})
			}
		})
	}
}

func runTestCase(t *testing.T, engine *cql.Engine, tc TestCase) {
	t.Helper()

	expr := strings.TrimSpace(tc.Expression.Value)
	if expr == "" {
		t.Skip("empty expression")
	}

	invalid := tc.Expression.Invalid

	// If invalid is set (and not "false"), we expect an error.
	if invalid != "" && invalid != "false" {
		cqlSource := wrapExpression(expr)
		_, err := engine.EvaluateExpression(context.Background(), cqlSource, "result", nil, nil)
		if err == nil {
			t.Errorf("expected error (invalid=%q) but got success for: %s", invalid, expr)
		}
		return
	}

	// Normal test: evaluate and compare.
	cqlSource := wrapExpression(expr)
	got, err := engine.EvaluateExpression(context.Background(), cqlSource, "result", nil, nil)
	if err != nil {
		t.Fatalf("evaluation error: %v\nexpression: %s", err, expr)
	}

	if len(tc.Outputs) == 0 {
		return // no output to check
	}

	if len(tc.Outputs) == 1 {
		outputStr := strings.TrimSpace(tc.Outputs[0].Value)
		want, err := parseExpectedOutput(outputStr)
		if err != nil {
			t.Fatalf("parse expected output %q: %v", outputStr, err)
		}
		if !valuesEqual(got, want) {
			t.Errorf("expression: %s\ngot:  %v (%T)\nwant: %v (%T)", expr, got, got, want, want)
		}
		return
	}

	// Multiple outputs — treat as expected list.
	// Build a slice of expected values and compare as a list.
	wantValues := make(fptypes.Collection, 0, len(tc.Outputs))
	for _, out := range tc.Outputs {
		outputStr := strings.TrimSpace(out.Value)
		w, err := parseExpectedOutput(outputStr)
		if err != nil {
			t.Fatalf("parse expected output %q: %v", outputStr, err)
		}
		wantValues = append(wantValues, w)
	}

	want := cqltypes.NewList(wantValues)
	if !valuesEqual(got, want) {
		t.Errorf("expression: %s\ngot:  %v (%T)\nwant: %v (%T)", expr, got, got, want, want)
	}
}
