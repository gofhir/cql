package cql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofhir/cql/eval"
)

// This file is the in-repo half of the corpus test. Whether refusing to evaluate
// an unsound library is safe was measured against published measures, but that
// test skips unless the content is on disk, so nothing there runs in CI. These
// pin down the shapes that measure CQL is made of — the ones a conformance
// corpus of loose expressions cannot see, and the ones review found the gate
// wrongly refusing.

// codesRecorder captures what a retrieve filtered by.
type codesRecorder struct{ codes []any }

func (r *codesRecorder) Retrieve(_ context.Context, req eval.RetrieveRequest) ([]json.RawMessage, error) {
	r.codes = append(r.codes, req.Codes)
	return nil, nil
}

// TestMeasureShapesCheckOut covers valid CQL that the semantic phase must not
// refuse, now that refusing stops evaluation.
func TestMeasureShapesCheckOut(t *testing.T) {
	for _, tt := range []struct{ name, src string }{
		{
			// A delimited name may contain a dot of its own. Deciding the shape
			// of a qualified name from the joined text read this as a qualifier
			// and a member, and found neither.
			"a value set whose name contains a dot",
			`library T version '1.0'
using FHIR version '4.0.1'
valueset "Diabetes.All": 'http://example.org/vs/dm'
context Patient
define A: [Condition: "Diabetes.All"]
`,
		},
		{
			// A property of a function's operand, in the same position a value
			// set name goes.
			"a property read in the terminology position",
			`library T version '1.0'
using FHIR version '4.0.1'
context Patient
define function "G"(r Resource): singleton from ([Provenance: target in r.id])
define A: G(First([Encounter]))
`,
		},
		{
			// An element a backbone element inherits rather than declares:
			// extension comes from Element, several levels up.
			"an inherited element on a backbone element",
			`library T version '1.0'
using FHIR version '4.0.1'
context Patient
define A: First([Encounter]).hospitalization.extension
define B: First([Observation]).component.extension
define C: First([Encounter]).id
`,
		},
		{
			// A primitive whose conversion is declared on its base type.
			"a primitive that inherits its conversion",
			`library T version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
context Patient
define A: 'id: ' & First([Encounter]).id
`,
		},
		{
			// Set operators on a FHIR.Period, which is an interval only once
			// the model's conversion is applied.
			"a period in a set operation",
			`library T version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
parameter MP Interval<DateTime> default Interval[@2020-01-01, @2020-12-31]
context Patient
define A: First([Encounter]).period intersect MP
`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			diags, err := NewEngine().Check(tt.src)
			if err != nil {
				t.Fatalf("the library should parse: %v", err)
			}
			if errs := diags.Errors(); len(errs) != 0 {
				t.Errorf("want no findings, got %v", errs)
			}
		})
	}
}

// TestQualifiedValueSetResolvesInItsOwnLibrary covers which value set a
// qualified name means. Resolving by bare name used the local set of that name,
// so a measure and a shared library both defining "Diabetes" — which is the sort
// of name two libraries arrive at independently — filtered by the wrong URL,
// silently.
func TestQualifiedValueSetResolvesInItsOwnLibrary(t *testing.T) {
	const common = `library Common version '1.0'
using FHIR version '4.0.1'
valueset "Diabetes": 'http://example.org/common/dm'
define Unused: 1
`
	resolver := func(_ context.Context, name, _ string) (string, error) {
		if name == "Common" {
			return common, nil
		}
		return "", nil
	}
	const src = `library T version '1.0'
using FHIR version '4.0.1'
include Common version '1.0' called Common
valueset "Diabetes": 'http://example.org/local/dm'
context Patient
define Qualified: [Condition: Common."Diabetes"]
define Bare: [Condition: "Diabetes"]
`
	for _, tt := range []struct{ define, want string }{
		{"Qualified", "http://example.org/common/dm"},
		{"Bare", "http://example.org/local/dm"},
	} {
		rec := &codesRecorder{}
		if _, err := NewEngine(WithDataProvider(rec), WithLibraryResolver(resolver)).
			EvaluateExpression(context.Background(), src, tt.define,
				[]byte(`{"resourceType":"Patient","id":"p1"}`), nil); err != nil {
			t.Fatalf("%s: %v", tt.define, err)
		}
		if len(rec.codes) != 1 {
			t.Fatalf("%s: %d retrieves, want 1", tt.define, len(rec.codes))
		}
		if got, _ := rec.codes[0].(string); got != tt.want {
			t.Errorf("%s filtered by %q, want %q", tt.define, got, tt.want)
		}
	}
}

// An alias that names no loaded library resolves to nothing rather than to a
// local set of the same name, so the mistake is reported instead of answered.
func TestUnknownLibraryAliasDoesNotFallBackToTheLocalSet(t *testing.T) {
	const src = `library T version '1.0'
using FHIR version '4.0.1'
valueset "Diabetes": 'http://example.org/local/dm'
context Patient
define A: [Condition: NotIncluded."Diabetes"]
`
	rec := &codesRecorder{}
	_, err := NewEngine(WithDataProvider(rec)).EvaluateExpression(
		context.Background(), src, "A", []byte(`{"resourceType":"Patient","id":"p1"}`), nil)
	if err == nil {
		t.Fatalf("evaluated with codes %v, want the unknown alias reported", rec.codes)
	}
}
