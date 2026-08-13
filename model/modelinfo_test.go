package model

import "testing"

// TestEmbeddedR4ModelInfo covers the official document the package carries. The
// model it replaced described six resources out of the 147 a retrieve can name,
// so most of what a measure asks about was simply absent.
func TestEmbeddedR4ModelInfo(t *testing.T) {
	mi, err := LoadR4ModelInfo()
	if err != nil {
		t.Fatalf("loading the embedded model info: %v", err)
	}
	if mi.Version() != "4.0.1" {
		t.Errorf("version = %q, want 4.0.1", mi.Version())
	}
	if got := mi.PatientClassName(); got != "FHIR.Patient" {
		t.Errorf("patient class = %q, want FHIR.Patient", got)
	}

	// The code path a retrieve filters on, for types the hand-built model never
	// mentioned. Getting these wrong sends the filter at the wrong element.
	for _, tt := range []struct{ resource, want string }{
		{"Condition", "code"},
		{"Encounter", "type"},
		{"CarePlan", "category"},
		{"Coverage", "type"},
		{"Immunization", "vaccineCode"},
		{"MedicationRequest", "medication"},
	} {
		if got := mi.PrimaryCodePath(tt.resource); got != tt.want {
			t.Errorf("PrimaryCodePath(%s) = %q, want %q", tt.resource, got, tt.want)
		}
		if !mi.IsRetrievable(tt.resource) {
			t.Errorf("%s should be retrievable", tt.resource)
		}
	}

	// The base chain, which is what `is` and `as` ask about.
	for _, tt := range []struct {
		concrete, target string
		want             bool
	}{
		{"Condition", "Condition", true},
		{"Condition", "DomainResource", true},
		{"Condition", "Resource", true},
		{"Patient", "DomainResource", true},
		{"Condition", "Patient", false},
		{"Condition", "Nonexistent", false},
	} {
		if got := mi.IsSubtypeOf(tt.concrete, tt.target); got != tt.want {
			t.Errorf("IsSubtypeOf(%s, %s) = %v, want %v", tt.concrete, tt.target, got, tt.want)
		}
	}

	// Contexts carry the element that identifies their subject, which is what a
	// patient-scoped retrieve needs.
	if k, ok := mi.ContextKeyElement("Patient"); !ok || k != "id" {
		t.Errorf("Patient context key = %q (%v), want id", k, ok)
	}

	// Declared conversions, which Etapa 3 applies and overload dispatch reads.
	for _, tt := range []struct{ from, to, want string }{
		{"FHIR.Coding", "System.Code", "FHIRHelpers.ToCode"},
		{"FHIR.CodeableConcept", "System.Concept", "FHIRHelpers.ToConcept"},
		{"FHIR.Quantity", "System.Quantity", "FHIRHelpers.ToQuantity"},
	} {
		if got, ok := mi.ConversionFunction(tt.from, tt.to); !ok || got != tt.want {
			t.Errorf("ConversionFunction(%s, %s) = %q (%v), want %q", tt.from, tt.to, got, ok, tt.want)
		}
	}

	// Choice elements, which the evaluator expands into concrete field names.
	for _, path := range []string{"Observation.value", "Condition.onset", "Patient.deceased"} {
		if !mi.IsChoiceType(path) {
			t.Errorf("%s should be a choice type", path)
		}
	}
}

// TestFHIRModelInfoRejectsUnknownVersion covers a version the build does not
// carry. `using FHIR version '5.0.0'` used to be accepted in silence and
// evaluated against R4 anyway.
func TestFHIRModelInfoRejectsUnknownVersion(t *testing.T) {
	if _, err := FHIRModelInfo("4.0.1"); err != nil {
		t.Errorf("4.0.1 should be available: %v", err)
	}
	if _, err := FHIRModelInfo("5.0.0"); err == nil {
		t.Error("a version the build does not carry should be an error, not a silent fallback")
	}
}
