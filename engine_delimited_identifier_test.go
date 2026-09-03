package cql

import (
	"context"
	"encoding/json"
	"fmt"
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

// TestDelimitedFunctionNameInAQualifiedCall covers the last place a name kept
// its delimiters, and the one that took a real measure to find.
//
// `define function "Normalize Interval"` registers the name without its quotes —
// the declaration side has read names through undelimitIdentifier since the
// quotes were first fixed. A call read its name the same way, except when the
// call was qualified by a library alias, where the builder took the text as
// written. So the two halves of one name disagreed:
//
//	H.Plain(1)      4
//	H."Plain"(1)    error: function '"Plain"' not found in library 'H'
//
// Same function, same library, spelled two of the three ways CQL allows.
//
// It reaches published content: `Global."Normalize Interval"` appears in 14 of
// the 19 libraries in cqframework/ecqm-content-r4, and evaluating any of their
// numerators failed on it. Nothing here caught it because Engine.Check validates
// each library on its own, without the include graph — an alias it cannot see
// into is honestly Unknown to it, so all 19 check clean — and the conformance
// corpus is loose expressions with no included libraries at all. It took
// evaluating a measure against its own test case to surface.
func TestDelimitedFunctionNameInAQualifiedCall(t *testing.T) {
	const helper = `library Helper version '1.0'
define function "Normalize Interval"(x Integer): x + 1
define function Plain(x Integer): x + 3
`
	resolve := func(_ context.Context, name, _ string) (string, error) {
		if name == "Helper" {
			return helper, nil
		}
		return "", fmt.Errorf("no library %q", name)
	}

	for _, tt := range []struct{ call, want string }{
		// The three spellings of one name have to reach one function.
		{`H.Plain(1)`, "4"},
		{`H."Plain"(1)`, "4"},
		{"H.`Plain`(1)", "4"},
		// And a name that can only be written delimited, because it has a space —
		// which is the form published measures actually use.
		{`H."Normalize Interval"(1)`, "2"},
	} {
		src := "library T version '1.0'\ninclude Helper version '1.0' called H\ndefine A: " + tt.call + "\n"
		got, err := NewEngine(WithLibraryResolver(resolve)).
			EvaluateExpression(context.Background(), src, "A", nil, nil)
		if err != nil {
			t.Errorf("%s: %v — a delimited name is the same name", tt.call, err)
			continue
		}
		if got == nil || got.String() != tt.want {
			t.Errorf("%s = %v, want %s", tt.call, got, tt.want)
		}
	}

	// A function that genuinely is not there still says so, so this does not make
	// every qualified call resolve to something.
	src := "library T version '1.0'\ninclude Helper version '1.0' called H\ndefine A: H.\"Absent\"(1)\n"
	if _, err := NewEngine(WithLibraryResolver(resolve)).
		EvaluateExpression(context.Background(), src, "A", nil, nil); err == nil {
		t.Error("a call to a function the library does not define should fail")
	}
}
