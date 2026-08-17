package cql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofhir/cql/eval"
)

// resourceTypeRecorder remembers the resource type each retrieve asked for.
type resourceTypeRecorder struct{ types []string }

func (r *resourceTypeRecorder) Retrieve(_ context.Context, req eval.RetrieveRequest) ([]json.RawMessage, error) {
	r.types = append(r.types, req.ResourceType)
	return []json.RawMessage{json.RawMessage(
		`{"resourceType":"Encounter","id":"e1","status":"finished"}`)}, nil
}

// TestDelimitedTypeNamesReachTheProviderUnquoted covers a retrieve that asked
// for a resource type spelled with quotes in it.
//
// CQL lets any identifier be written in double quotes, and published measures
// write `["Encounter"]` — 13 of the 19 libraries in cqframework/ecqm-content-r4
// do. The quotes were kept, so the provider received the resource type
// `"Encounter"`, quotes and all, which matches nothing in a real database.
func TestDelimitedTypeNamesReachTheProviderUnquoted(t *testing.T) {
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)
	for _, tt := range []struct{ name, retrieve string }{
		{"bare", "[Encounter]"},
		{"delimited", `["Encounter"]`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\ndefine A: Count(" +
				tt.retrieve + ")\n"
			rec := &resourceTypeRecorder{}
			if _, err := NewEngine(WithDataProvider(rec)).
				EvaluateExpression(context.Background(), src, "A", patient, nil); err != nil {
				t.Fatalf("evaluating: %v", err)
			}
			if len(rec.types) != 1 || rec.types[0] != "Encounter" {
				t.Errorf("provider was asked for %q, want Encounter", rec.types)
			}
		})
	}
}

// The same name reaches the model, so the semantic phase resolves elements on a
// delimited type instead of reporting every one of them as missing. This is
// where it showed up: 131 findings across the published measures, almost all of
// them "FHIR.\"Encounter\" has no element period" and its like.
func TestDelimitedTypeNamesResolveTheirElements(t *testing.T) {
	for _, retrieve := range []string{"[Encounter]", `["Encounter"]`} {
		src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\ndefine A: First(" +
			retrieve + ").period\n"
		diags, err := NewEngine().Check(src)
		if err != nil {
			t.Fatalf("%s: %v", retrieve, err)
		}
		if errs := diags.Errors(); len(errs) != 0 {
			t.Errorf("%s: want no findings, got %v", retrieve, errs)
		}
	}
}

// A qualified type name written with quotes resolves the same way.
func TestDelimitedQualifiedTypeNames(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\n" +
		"define A: First([\"Observation\"]) is \"FHIR\".\"Observation\"\n"
	diags, err := NewEngine().Check(src)
	if err != nil {
		t.Fatalf("checking: %v", err)
	}
	if errs := diags.Errors(); len(errs) != 0 {
		t.Errorf("want no findings, got %v", errs)
	}
}
