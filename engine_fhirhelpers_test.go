package cql

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofhir/cql/eval"
)

// fhirProvider serves one Observation carrying every shape the conversions read.
type fhirProvider struct{}

func (fhirProvider) Retrieve(_ context.Context, req eval.RetrieveRequest) ([]json.RawMessage, error) {
	return []json.RawMessage{json.RawMessage(
		`{"resourceType":"Observation","id":"o1","status":"final",` +
			`"valueQuantity":{"value":5,"unit":"mg","system":"http://unitsofmeasure.org","code":"mg"},` +
			`"code":{"coding":[{"system":"http://loinc.org","code":"8480-6","display":"SBP"}],"text":"BP"},` +
			`"effectivePeriod":{"start":"2020-01-01","end":"2020-12-31"},` +
			`"referenceRange":[{"low":{"value":1,"unit":"mg","system":"http://unitsofmeasure.org","code":"mg"},` +
			`"high":{"value":9,"unit":"mg","system":"http://unitsofmeasure.org","code":"mg"}}]}`)}, nil
}

type bpTerminology struct{}

func (bpTerminology) InValueSet(_ context.Context, code, _, _ string) (bool, error) {
	return code == "8480-6", nil
}

// TestBuiltInFHIRHelpersConversions covers the library the engine falls back to
// when the caller supplies no resolver. It used to be eight identity functions
// with no ToCode, ToConcept, ToInterval or ToRatio at all, so the conversions
// that turn FHIR data into CQL system types were missing outright.
func TestBuiltInFHIRHelpersConversions(t *testing.T) {
	src := `library T version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
valueset "BP": 'http://example.org/bp'

define Obs: First([Observation])
define AsCode:      FH.ToCode(Obs.code.coding[0])
define AsConcept:   FH.ToConcept(Obs.code)
define AsQuantity:  FH.ToQuantity(Obs.valueQuantity)
define AsInterval:  FH.ToInterval(Obs.effectivePeriod)
define CodeInVS:    FH.ToCode(Obs.code.coding[0]) in "BP"
define ConceptInVS: FH.ToConcept(Obs.code) in "BP"
define QuantityVal: FH.ToQuantity(Obs.valueQuantity).value
define CodeField:   FH.ToCode(Obs.code.coding[0]).code
`
	tests := []struct{ name, want string }{
		{"AsCode", "Code '8480-6' from http://loinc.org display 'SBP'"},
		{"AsQuantity", "5 'mg'"},
		{"AsInterval", "Interval[2020-01-01, 2020-12-31]"},
		{"CodeInVS", "true"},
		{"ConceptInVS", "true"},
		{"QuantityVal", "5"},
		{"CodeField", "8480-6"},
	}

	engine := NewEngine(WithDataProvider(fhirProvider{}), WithTerminologyProvider(bpTerminology{}))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.EvaluateExpression(context.Background(), src, tt.name, nil, nil)
			if err != nil {
				t.Fatalf("evaluating %s: %v", tt.name, err)
			}
			if s := valueString(got); s != tt.want {
				t.Errorf("%s = %s, want %s", tt.name, s, tt.want)
			}
		})
	}

	// The Concept carries its codes, so a membership check has something to ask about.
	concept, err := engine.EvaluateExpression(context.Background(), src, "AsConcept", nil, nil)
	if err != nil {
		t.Fatalf("evaluating AsConcept: %v", err)
	}
	if s := valueString(concept); !strings.Contains(s, "8480-6") {
		t.Errorf("AsConcept = %s, want it to carry its coding", s)
	}
}

// TestBuiltInFHIRHelpersIdentities covers the surface the previous built-in
// library provided, so that replacing it does not silently drop a conversion
// existing callers depend on.
func TestBuiltInFHIRHelpersIdentities(t *testing.T) {
	src := `library T version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH

define Obs: First([Observation])
define S:  FH.ToString(Obs.status)
define B:  FH.ToBoolean(true)
define I:  FH.ToInteger(42)
define D:  FH.ToDecimal(1.5)
define DT: FH.ToDateTime(@2020-01-01T00:00:00)
define Da: FH.ToDate(@2020-01-01)
define Ti: FH.ToTime(@T12:00:00)
`
	tests := []struct{ name, want string }{
		{"S", "final"}, {"B", "true"}, {"I", "42"}, {"D", "1.5"},
		{"DT", "2020-01-01T00:00:00"}, {"Da", "2020-01-01"}, {"Ti", "12:00:00"},
	}
	engine := NewEngine(WithDataProvider(fhirProvider{}))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.EvaluateExpression(context.Background(), src, tt.name, nil, nil)
			if err != nil {
				t.Fatalf("evaluating %s: %v", tt.name, err)
			}
			if s := valueString(got); s != tt.want {
				t.Errorf("%s = %s, want %s", tt.name, s, tt.want)
			}
		})
	}
}

// TestTransitiveIncludeAliases covers a library whose include chose the same
// alias as the top-level library did for something else. Registering the whole
// graph under one alias map made the top-level `C.Answer()` resolve against the
// nested library's C.
func TestTransitiveIncludeAliases(t *testing.T) {
	libs := map[string]string{
		"Good": "library Good version '1.0'\n\ndefine function Answer(): 'GOOD'\n",
		"Bad":  "library Bad version '1.0'\n\ndefine function Answer(): 'BAD'\n",
		"Mid":  "library Mid version '1.0'\ninclude Bad version '1.0' called C\n\ndefine function Ask(): C.Answer()\n",
	}
	src := `library T version '1.0'
include Good version '1.0' called C
include Mid version '1.0' called M

define Mine: C.Answer()
define Theirs: M.Ask()
`
	engine := NewEngine(WithLibraryResolver(func(_ context.Context, name, _ string) (string, error) {
		if s, ok := libs[name]; ok {
			return s, nil
		}
		return "", context.Canceled
	}))
	for _, tt := range []struct{ name, want string }{{"Mine", "GOOD"}, {"Theirs", "BAD"}} {
		got, err := engine.EvaluateExpression(context.Background(), src, tt.name, nil, nil)
		if err != nil {
			t.Fatalf("evaluating %s: %v", tt.name, err)
		}
		if s := valueString(got); s != tt.want {
			t.Errorf("%s = %s, want %s — an alias means what the library that wrote it says", tt.name, s, tt.want)
		}
	}
}

// TestIncludeAliasDoesNotShadowScope covers a query alias or operand sharing a
// name with an include. Anything bound in scope wins.
func TestIncludeAliasDoesNotShadowScope(t *testing.T) {
	libs := map[string]string{"Helper": "library Helper version '1.0'\n\ndefine Anchor: 1\n"}
	src := "library T version '1.0'\ninclude Helper version '1.0' called H\n\ndefine X: ({Tuple{x: 1}}) H return H.x\n"
	engine := NewEngine(WithLibraryResolver(func(_ context.Context, name, _ string) (string, error) {
		if s, ok := libs[name]; ok {
			return s, nil
		}
		return "", context.Canceled
	}))
	got, err := engine.EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err != nil {
		t.Fatalf("a query alias named like an include should win: %v", err)
	}
	if s := valueString(got); s != "{1}" {
		t.Errorf("X = %s, want {1}", s)
	}
}

// TestPrivateDefineStaysPrivate covers access being decided from the
// declaration rather than from the memoized results, which fill in as the
// library evaluates its own definitions.
func TestPrivateDefineStaysPrivate(t *testing.T) {
	libs := map[string]string{"H": "library H version '1.0'\n\ndefine private Secret: 42\ndefine Pub: Secret + 0\n"}
	engine := NewEngine(WithLibraryResolver(func(_ context.Context, name, _ string) (string, error) {
		if s, ok := libs[name]; ok {
			return s, nil
		}
		return "", context.Canceled
	}))
	// Reading the public define first fills the cache with Secret's value.
	src := "library T version '1.0'\ninclude H version '1.0' called H\n\ndefine X: ToString(H.Pub) + ToString(H.Secret)\n"
	if _, err := engine.EvaluateExpression(context.Background(), src, "X", nil, nil); err == nil {
		t.Error("a private define should stay private even after the library has read it")
	}
}

// TestAbsentClinicalElementsAreNull covers elements the value does not carry.
// These structs hold plain strings, so answering with the empty string made
// `code.display is null` false for a Code that never had a display.
func TestAbsentClinicalElementsAreNull(t *testing.T) {
	src := `library T version '1.0'
define C: System.Code { code: '1', system: 'http://x' }
define NoDisplay: C.display is null
define NoVersion: C.version is null
define Cpt: System.Concept { codes: { System.Code { code: '1', system: 'http://x' } } }
define NoConceptDisplay: Cpt.display is null
`
	engine := NewEngine()
	for _, n := range []string{"NoDisplay", "NoVersion", "NoConceptDisplay"} {
		got, err := engine.EvaluateExpression(context.Background(), src, n, nil, nil)
		if err != nil {
			t.Fatalf("evaluating %s: %v", n, err)
		}
		if s := valueString(got); s != "true" {
			t.Errorf("%s = %s, want true", n, s)
		}
	}
}

// recordingTerminology captures what the engine actually asks the server.
type recordingTerminology struct{ code, system string }

func (r *recordingTerminology) InValueSet(_ context.Context, code, system, _ string) (bool, error) {
	r.code, r.system = code, system
	return code == "8480-6", nil
}

// TestTerminologyQueryIsWellFormed covers what reaches the terminology server.
// A value the extractor did not recognize used to be rendered with String(), so
// a raw FHIR Coding was sent as its JSON text in the code field — a query no
// server can answer, and resource content sent outward with it.
func TestTerminologyQueryIsWellFormed(t *testing.T) {
	src := `library T version '1.0'
using FHIR version '4.0.1'
valueset "BP": 'http://example.org/bp'

define Obs: First([Observation])
define RawCodingInVS: Obs.code.coding[0] in "BP"
define RawConceptInVS: Obs.code in "BP"
`
	for _, name := range []string{"RawCodingInVS", "RawConceptInVS"} {
		spy := &recordingTerminology{}
		engine := NewEngine(WithDataProvider(fhirProvider{}), WithTerminologyProvider(spy))
		got, err := engine.EvaluateExpression(context.Background(), src, name, nil, nil)
		if err != nil {
			t.Fatalf("evaluating %s: %v", name, err)
		}
		if spy.code != "8480-6" || spy.system != "http://loinc.org" {
			t.Errorf("%s asked the server for code=%q system=%q, want the coding's own fields",
				name, spy.code, spy.system)
		}
		if s := valueString(got); s != "true" {
			t.Errorf("%s = %s, want true", name, s)
		}
	}
}
