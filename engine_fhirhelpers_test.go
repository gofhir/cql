package cql

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// fhirProvider serves one Observation carrying every shape the conversions read.
type fhirProvider struct{}

func (fhirProvider) Retrieve(_ context.Context, _, _, _ string, _, _ interface{}) ([]json.RawMessage, error) {
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
