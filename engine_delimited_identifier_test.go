package cql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofhir/cql/eval"
)

// resourceTypeRecorder remembers what each retrieve asked for.
type resourceTypeRecorder struct {
	types []string
	paths []string
}

func (r *resourceTypeRecorder) Retrieve(_ context.Context, req eval.RetrieveRequest) ([]json.RawMessage, error) {
	r.types = append(r.types, req.ResourceType)
	r.paths = append(r.paths, req.CodePath)
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

// TestDelimitedIdentifiersAcrossTheDeclarations covers the rest of the class.
// The first fix undelimited the retrieve's type and left the same leak in its
// siblings — one of them a line below it — so these are the places a delimited
// name still had its delimiters.
func TestDelimitedIdentifiersAcrossTheDeclarations(t *testing.T) {
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)

	// A function nothing could call: the definition was named `"F"` and the
	// call looked for F.
	t.Run("function name", func(t *testing.T) {
		src := "library T version '1.0'\n" +
			"define function \"F\"(x Integer) returns Integer: x + 1\n" +
			"define A: \"F\"(1)\n"
		got, err := NewEngine().EvaluateExpression(context.Background(), src, "A", nil, nil)
		if err != nil {
			t.Fatalf("calling a quoted function: %v", err)
		}
		if s := valueString(got); s != "2" {
			t.Errorf("= %s, want 2", s)
		}
	})

	// The code path travels to the provider beside the resource type, and was
	// still carrying quotes when the type had stopped. No published measure
	// writes it this way — they write `[Coverage: type in "Payer"]`, where the
	// quoted name is the value set — so this one is for consistency rather than
	// for a fault anyone has hit.
	t.Run("code path", func(t *testing.T) {
		src := "library T version '1.0'\nusing FHIR version '4.0.1'\n" +
			"valueset \"VS\": 'http://example.org/vs'\ncontext Patient\n" +
			"define A: [\"Encounter\": \"status\" in \"VS\"]\n"
		rec := &resourceTypeRecorder{}
		if _, err := NewEngine(WithDataProvider(rec)).
			EvaluateExpression(context.Background(), src, "A", patient, nil); err != nil {
			t.Fatalf("evaluating: %v", err)
		}
		if rec.paths[0] != "status" {
			t.Errorf("code path = %q, want status", rec.paths[0])
		}
	})

	// Backticks are the other delimiter CQL allows, and the first fix only
	// handled quotes.
	t.Run("backticks", func(t *testing.T) {
		src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\n" +
			"define A: Count([`Encounter`])\n"
		rec := &resourceTypeRecorder{}
		if _, err := NewEngine(WithDataProvider(rec)).
			EvaluateExpression(context.Background(), src, "A", patient, nil); err != nil {
			t.Fatalf("evaluating: %v", err)
		}
		if rec.types[0] != "Encounter" {
			t.Errorf("provider was asked for %q, want Encounter", rec.types[0])
		}
	})

	// A quoted include alias has to match the name the calls use, or the
	// semantic phase reports every call into that library again.
	t.Run("include alias", func(t *testing.T) {
		src := "library T version '1.0'\n" +
			"include Common version '1.0' called \"C\"\n" +
			"define A: \"C\".Helper(1)\n"
		diags, err := NewEngine().Check(src)
		if err != nil {
			t.Fatalf("checking: %v", err)
		}
		if errs := diags.Errors(); len(errs) != 0 {
			t.Errorf("want no findings, got %v", errs)
		}
	})
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
