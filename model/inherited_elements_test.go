package model_test

import (
	"testing"

	"github.com/gofhir/cql/model"
)

// TestATypeHasTheElementsItInherits covers the question every caller of this
// package asks and none of it answered: does this type have this element?
//
// The document declares each element once, where it is introduced — Observation.id
// on Resource, Coding.extension on Element — and the three lookups here were flat
// map reads keyed by the exact path. 126 elements of the ordinary resource types
// were invisible on that basis, and each caller that needed them walked the chain
// itself: two did, in two copies, and two did not.
func TestATypeHasTheElementsItInherits(t *testing.T) {
	mi, err := model.LoadR4ModelInfo()
	if err != nil {
		t.Fatalf("loading the embedded model info: %v", err)
	}

	for _, tt := range []struct{ path, declaredOn, wantType string }{
		{"Observation.id", "Resource", "FHIR.id"},
		{"Observation.meta", "Resource", "FHIR.Meta"},
		{"Observation.implicitRules", "Resource", "FHIR.uri"},
		{"Observation.language", "Resource", "FHIR.code"},
		{"Observation.text", "DomainResource", "FHIR.Narrative"},
		{"Observation.extension", "DomainResource", "FHIR.Extension"},
		{"Observation.modifierExtension", "DomainResource", "FHIR.Extension"},
		{"Observation.contained", "DomainResource", "FHIR.Resource"},
		// Not only resources: a datatype inherits from Element, and a backbone
		// element from BackboneElement.
		{"Coding.extension", "Element", "FHIR.Extension"},
		{"Period.id", "Element", "System.String"},
		{"Observation.Component.modifierExtension", "BackboneElement", "FHIR.Extension"},
	} {
		info, ok := mi.ElementInfoByPath(tt.path)
		if !ok {
			t.Errorf("ElementInfoByPath(%q): not found — it is declared on %s", tt.path, tt.declaredOn)
			continue
		}
		if got, ok := mi.ElementType(tt.path); !ok || got != tt.wantType {
			t.Errorf("ElementType(%q) = %q, %v; want %q", tt.path, got, ok, tt.wantType)
		}
		if info.IsChoice != mi.IsChoiceType(tt.path) {
			t.Errorf("%q: ElementInfo says choice=%v and IsChoiceType says %v",
				tt.path, info.IsChoice, mi.IsChoiceType(tt.path))
		}
	}

	// A name nothing declares is still a name nothing declares, at any depth.
	for _, path := range []string{"Observation.noSuchElement", "NoSuchType.id", "id"} {
		if _, ok := mi.ElementInfoByPath(path); ok {
			t.Errorf("ElementInfoByPath(%q) found something", path)
		}
		if _, ok := mi.ElementType(path); ok {
			t.Errorf("ElementType(%q) found something", path)
		}
		if mi.IsChoiceType(path) {
			t.Errorf("IsChoiceType(%q) is true", path)
		}
	}

	// The element a type declares itself is still the one that answers.
	if got, ok := mi.ElementType("Observation.status"); !ok || got != "FHIR.ObservationStatus" {
		t.Errorf("Observation.status = %q, %v; want FHIR.ObservationStatus", got, ok)
	}
	if !mi.IsChoiceType("Observation.value") {
		t.Error("Observation.value is no longer a choice element")
	}
}

// TestTheNearestDeclarationWins covers a subtype that redeclares an element of
// its base. FHIR 4.0.1 has no such pair, which is exactly why the rule is
// asserted against a model built here rather than left to the document to
// demonstrate: a walk that returned the first hit going *up* would answer with
// the base's declaration, and the difference between FHIR.id and System.String
// is the difference between a value that converts and one that does not.
func TestTheNearestDeclarationWins(t *testing.T) {
	mi := model.NewStaticModelInfo("test")
	mi.AddType(&model.TypeInfo{
		Name:     "Base",
		Elements: []model.ElementInfo{{Name: "shared", Type: "System.String"}, {Name: "onlyOnBase", Type: "System.Integer"}},
	})
	mi.AddType(&model.TypeInfo{
		Name:     "Derived",
		BaseName: "Base",
		Elements: []model.ElementInfo{{Name: "shared", Type: "System.Integer"}},
	})

	if got, _ := mi.ElementType("Derived.shared"); got != "System.Integer" {
		t.Errorf("Derived.shared = %q, want the subtype's own System.Integer", got)
	}
	if got, _ := mi.ElementType("Derived.onlyOnBase"); got != "System.Integer" {
		t.Errorf("Derived.onlyOnBase = %q, want System.Integer from Base", got)
	}

	// A document that names a type as its own base must report nothing rather
	// than spin, and so must a cycle between two.
	mi.AddType(&model.TypeInfo{Name: "SelfBased", BaseName: "SelfBased"})
	mi.AddType(&model.TypeInfo{Name: "Ping", BaseName: "Pong"})
	mi.AddType(&model.TypeInfo{Name: "Pong", BaseName: "Ping"})
	for _, path := range []string{"SelfBased.anything", "Ping.anything"} {
		if _, ok := mi.ElementInfoByPath(path); ok {
			t.Errorf("ElementInfoByPath(%q) found something in a cyclic model", path)
		}
	}
}
