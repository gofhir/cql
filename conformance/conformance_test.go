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

// valuesEqual compares two fptypes.Value instances for equality, and their
// numbers for their type too: CQL's equality says the Integer 3 equals the
// Decimal 3.0, and through it `10.1 div 3.1` answering an Integer passed a
// suite that expects 3.0. A Long of the expected output is read as an Integer
// (see outputParser), so the two are not told apart.
func valuesEqual(got, want fptypes.Value) bool {
	if got == nil && want == nil {
		return true
	}
	if got == nil || want == nil {
		return false
	}
	return got.Equal(want) && sameNumberTypes(got, want)
}

// sameNumberTypes reports whether every number in want is answered by a number
// of its type in got, inside lists, intervals and tuples. Where the two differ
// in shape it says yes, and leaves the difference to Equal.
func sameNumberTypes(got, want fptypes.Value) bool {
	switch w := want.(type) {
	case fptypes.Integer:
		_, ok := got.(fptypes.Integer)
		return ok
	case fptypes.Decimal:
		_, ok := got.(fptypes.Decimal)
		return ok
	case cqltypes.List:
		g, ok := got.(cqltypes.List)
		if !ok || len(g.Values) != len(w.Values) {
			return true
		}
		for i := range w.Values {
			if w.Values[i] != nil && g.Values[i] != nil && !sameNumberTypes(g.Values[i], w.Values[i]) {
				return false
			}
		}
	case cqltypes.Interval:
		g, ok := got.(cqltypes.Interval)
		if !ok {
			return true
		}
		for _, pair := range [][2]fptypes.Value{{g.Low, w.Low}, {g.High, w.High}} {
			if pair[0] != nil && pair[1] != nil && !sameNumberTypes(pair[0], pair[1]) {
				return false
			}
		}
	case cqltypes.Tuple:
		g, ok := got.(cqltypes.Tuple)
		if !ok {
			return true
		}
		for name, wv := range w.Elements {
			if gv := g.Elements[name]; wv != nil && gv != nil && !sameNumberTypes(gv, wv) {
				return false
			}
		}
	}
	return true
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
}

// untypedOutputs are the cases whose expected output is written without the
// type of its numbers. The Java engine, 3.26.0 and 5.4.0, answers each of them
// with Decimal points, as this engine does, and as the specification has it:
// the points of a Decimal interval are Decimals. The output text writes them
// as whole numbers, which reads as Integers, so their numbers are read as the
// Decimals Java answers, and compared with their type all the same: compared
// by value alone, { 10.0, 11, 12 } would pass.
var untypedOutputs = map[string]string{
	"ExpandPer1":                     "expand { Interval[10.0, 12.5] } per 1: Java answers Interval[10.0, 10.0] and on, Decimal points",
	"ExpandPer1Open":                 "expand { Interval[10.0, 12.5) } per 1: Java answers Decimal points",
	"ExpandPer1IntervalOverload":     "expand Interval[10.0, 12.5] per 1: Java answers { 10.0, 11.0, 12.0 }, Decimals",
	"ExpandPer1OpenIntervalOverload": "expand Interval[10.0, 12.5) per 1: Java answers Decimals",
}

// equalFor is the comparison a case is held to: by value and by the type of
// its numbers, with the numbers of one in untypedOutputs read as Decimals.
func equalFor(tc TestCase) func(got, want fptypes.Value) bool {
	if _, untyped := untypedOutputs[tc.Name]; untyped {
		return func(got, want fptypes.Value) bool { return valuesEqual(got, asDecimals(want)) }
	}
	return valuesEqual
}

// asDecimals is v with every Integer in it, inside lists and intervals, made the
// Decimal of its value.
func asDecimals(v fptypes.Value) fptypes.Value {
	switch w := v.(type) {
	case fptypes.Integer:
		d, err := fptypes.NewDecimal(w.String())
		if err != nil {
			return v
		}
		return d
	case cqltypes.List:
		values := make(fptypes.Collection, len(w.Values))
		for i, e := range w.Values {
			values[i] = asDecimals(e)
		}
		return cqltypes.NewList(values)
	case cqltypes.Interval:
		return cqltypes.Interval{Low: asDecimals(w.Low), High: asDecimals(w.High), LowClosed: w.LowClosed, HighClosed: w.HighClosed}
	}
	return v
}

func runTestCase(t *testing.T, engine *cql.Engine, parser outputParser, tc TestCase) {
	t.Helper()

	expr := strings.TrimSpace(tc.Expression.Value)
	if expr == "" {
		t.Skip("empty expression")
	}

	if reason, known := upstreamContradictions[tc.Name]; known {
		if clean, _ := runsClean(engine, parser, tc); clean {
			t.Fatalf("%s passes now, so upstream has settled it — remove this entry from "+
				"upstreamContradictions.\n\nRecorded reason: %s", tc.Name, reason)
		}
		t.Logf("known upstream contradiction: %s", reason)
		return
	}

	if kf, known := knownFailures[tc.Name]; known {
		clean, err := runsClean(engine, parser, tc)
		if clean {
			t.Fatalf("%s passes now; remove it from knownFailures.\n\nIts group: %s", tc.Name, failureGroups[kf.group])
		}
		if (err != nil) != kf.errs {
			t.Fatalf("%s fails another way now (error: %v), and knownFailures records errs %v; classify it again",
				tc.Name, err, kf.errs)
		}
		t.Logf("known failure: %s", failureGroups[kf.group])
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
	equal := equalFor(tc)
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
		if !equal(got, want) {
			t.Errorf("expression: %s\ngot:  %v (%T)\nwant: %v (%T)", expr, got, got, want, want)
		}
		if _, untyped := untypedOutputs[tc.Name]; untyped && valuesEqual(got, want) {
			t.Errorf("%s passes with the types of its numbers compared; remove it from untypedOutputs", tc.Name)
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
	if !equal(got, want) {
		t.Errorf("expression: %s\ngot:  %v (%T)\nwant: %v (%T)", expr, got, got, want, want)
	}
}

// runsClean answers whether a test case would pass, and the error the engine
// gave if any, without reporting anything. It
// mirrors the verdicts of runTestCase and exists only so an entry in
// upstreamContradictions or knownFailures can be checked for still failing.
func runsClean(engine *cql.Engine, parser outputParser, tc TestCase) (bool, error) {
	expr := strings.TrimSpace(tc.Expression.Value)
	cqlSource := wrapExpression(expr)
	got, err := engine.EvaluateExpression(context.Background(), cqlSource, "result", nil, nil)

	if invalid := tc.Expression.Invalid; invalid != "" && invalid != "false" {
		return err != nil, err
	}
	if err != nil {
		return false, err
	}
	if len(tc.Outputs) == 0 {
		return true, nil
	}
	if len(tc.Outputs) == 1 {
		want, perr := parser.parse(strings.TrimSpace(tc.Outputs[0].Value))
		return perr == nil && equalFor(tc)(got, want), nil
	}

	wantValues := make(fptypes.Collection, 0, len(tc.Outputs))
	for _, out := range tc.Outputs {
		w, perr := parser.parse(strings.TrimSpace(out.Value))
		if perr != nil {
			return false, nil
		}
		wantValues = append(wantValues, w)
	}
	return equalFor(tc)(got, cqltypes.NewList(wantValues)), nil
}
