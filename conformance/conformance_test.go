package conformance

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	// One reading of the clock, handed to both sides. The engine places a bare
	// DateTime at the evaluation request's offset and the parser reads the
	// expected text under the same rule, so the two are always comparing values
	// placed the same way. Two readings would agree except across a DST boundary,
	// where they would produce a failure nobody could reproduce.
	//
	// The offset still comes from the process zone, so TZ decides what is
	// measured — and it has to: with a default offset in play the answers depend
	// on it, and one zone proves nothing.
	now := time.Now()
	_, offsetSeconds := now.Zone()
	parser := outputParser{assumedOffset: time.Duration(offsetSeconds) * time.Second}
	engine := cql.NewEngine(cql.WithEvaluationTimestamp(now))

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
							runTestCase(t, engine, parser, tc)
						})
					}
				})
			}
		})
	}
}

// upstreamContradictions are the tests whose expectations cannot all hold at once,
// with the demonstration alongside each. They are not skipped: the engine still runs
// them and this file asserts that they still fail. When upstream corrects one, the
// assertion trips and the entry has to go — a list that cleans itself rather than one
// that quietly outlives its reason.
var upstreamContradictions = map[string]string{
	"DateTimeDurationBetweenUncertainInterval": "expects Interval[17,44] for `days between DateTime(2014, 1, 15) " +
		"and DateTime(2014, 2)`, while DateTimeDurationBetweenUncertainMultiply squares the same expression and " +
		"expects Interval[256,1936] and DateTimeDurationBetweenUncertainAdd doubles it and expects Interval[32,88]. " +
		"Both of those put the base at [16,44] (√256=16, 32/2=16), which is what the engine computes. Three tests " +
		"require [16,44] and this one requires [17,44].",

	"FloorIntegerGreaterThanMaxInteger": "expects null for Floor(2147483648), while CeilingIntegerGreaterThanMaxInteger " +
		"marks the structurally identical Ceiling(2147483648) invalid=\"syntax\" and expects an error. The engine keeps " +
		"the compile error, which is what the spec asks of an out-of-range Integer literal.",

	"FloorIntegerLessThanMinInteger": "expects null for Floor(-2147483649); see FloorIntegerGreaterThanMaxInteger.",
}

func runTestCase(t *testing.T, engine *cql.Engine, parser outputParser, tc TestCase) {
	t.Helper()

	expr := strings.TrimSpace(tc.Expression.Value)
	if expr == "" {
		t.Skip("empty expression")
	}

	if reason, known := upstreamContradictions[tc.Name]; known {
		if runsClean(engine, parser, tc) {
			t.Fatalf("%s passes now, so upstream has settled it — remove this entry from "+
				"upstreamContradictions.\n\nRecorded reason: %s", tc.Name, reason)
		}
		t.Logf("known upstream contradiction: %s", reason)
		return
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
		want, err := parser.parse(outputStr)
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
		w, err := parser.parse(outputStr)
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

// runsClean answers whether a test case would pass, without reporting anything. It
// mirrors the verdicts of runTestCase and exists only so an entry in
// upstreamContradictions can be checked for still being contradictory.
func runsClean(engine *cql.Engine, parser outputParser, tc TestCase) bool {
	expr := strings.TrimSpace(tc.Expression.Value)
	cqlSource := wrapExpression(expr)
	got, err := engine.EvaluateExpression(context.Background(), cqlSource, "result", nil, nil)

	if invalid := tc.Expression.Invalid; invalid != "" && invalid != "false" {
		return err != nil
	}
	if err != nil {
		return false
	}
	if len(tc.Outputs) == 0 {
		return true
	}
	if len(tc.Outputs) == 1 {
		want, perr := parser.parse(strings.TrimSpace(tc.Outputs[0].Value))
		return perr == nil && valuesEqual(got, want)
	}

	wantValues := make(fptypes.Collection, 0, len(tc.Outputs))
	for _, out := range tc.Outputs {
		w, perr := parser.parse(strings.TrimSpace(out.Value))
		if perr != nil {
			return false
		}
		wantValues = append(wantValues, w)
	}
	return valuesEqual(got, cqltypes.NewList(wantValues))
}
