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

// TestContextSearchParamRefusesWhatItCannotUse covers the declarations the
// model makes that a provider cannot query with.
func TestContextSearchParamRefusesWhatItCannotUse(t *testing.T) {
	mi, err := LoadR4ModelInfo()
	if err != nil {
		t.Fatalf("loading: %v", err)
	}

	// Usable search parameters.
	for _, tt := range []struct{ resource, want string }{
		{"Condition", "patient"},
		{"Observation", "subject"},
		{"MedicationRequest", "subject"},
	} {
		got, ok := mi.ContextSearchParam(tt.resource, "Patient")
		if !ok || got != tt.want {
			t.Errorf("ContextSearchParam(%s, Patient) = %q (%v), want %q", tt.resource, got, ok, tt.want)
		}
	}

	// A type that *is* the context is identified by its own key. The model says
	// Patient relates to Patient through "other", which is Patient.link.other
	// and would fetch the linked patients instead.
	if got, ok := mi.ContextSearchParam("Patient", "Patient"); ok {
		t.Errorf("ContextSearchParam(Patient, Patient) = %q, want nothing", got)
	}

	// A FHIRPath fragment is not a search parameter. These five declare
	// "where(resolve() is Patient)", which no server can answer.
	for _, resource := range []string{"AuditEvent", "Provenance", "Basic", "Person", "MeasureReport"} {
		if got, ok := mi.ContextSearchParam(resource, "Patient"); ok {
			t.Errorf("ContextSearchParam(%s, Patient) = %q, want nothing — it is a path expression", resource, got)
		}
	}
}

// TestContextRelationSeparatesTheThreeSilences covers what having no search
// parameter meant. Three different situations reported the same empty string,
// and two of them still need the retrieve scoped: a provider reading only that
// field returned every patient's data for a Patient retrieve and for the five
// types whose relation is a FHIRPath fragment.
func TestContextRelationSeparatesTheThreeSilences(t *testing.T) {
	mi, err := LoadR4ModelInfo()
	if err != nil {
		t.Fatalf("loading: %v", err)
	}

	for _, tt := range []struct {
		resource string
		want     ContextRelation
	}{
		// The type that is the context: scope by resource id.
		{"Patient", ContextRelation{Kind: ContextSelf}},

		// A search parameter the provider can query with.
		{"Condition", ContextRelation{Kind: ContextBySearchParam, SearchParam: "patient"}},
		{"Observation", ContextRelation{Kind: ContextBySearchParam, SearchParam: "subject"}},

		// A FHIRPath fragment: still needs scoping, but not by a query.
		{"AuditEvent", ContextRelation{Kind: ContextByExpression, Expression: "where(resolve() is Patient)"}},
		{"Provenance", ContextRelation{Kind: ContextByExpression, Expression: "where(resolve() is Patient)"}},

		// No relation at all: these belong to no patient, and scoping them
		// would return nothing rather than everything.
		{"Medication", ContextRelation{Kind: ContextUnrelated}},
		{"Location", ContextRelation{Kind: ContextUnrelated}},
		{"Organization", ContextRelation{Kind: ContextUnrelated}},
	} {
		got := mi.ContextRelation(tt.resource, "Patient")
		if got != tt.want {
			t.Errorf("ContextRelation(%s, Patient) = %+v, want %+v", tt.resource, got, tt.want)
		}
	}
}

// TestPrimaryCodePathDistinguishesSilence covers the 84 retrievable types that
// declare no primary code path. Answering "code" for those would leave a
// provider unable to tell what the model said from what it did not, and some
// of them have no code element at all.
func TestPrimaryCodePathDistinguishesSilence(t *testing.T) {
	mi, err := LoadR4ModelInfo()
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	if got := mi.PrimaryCodePath("Condition"); got != "code" {
		t.Errorf("Condition = %q, want code", got)
	}
	for _, resource := range []string{"FamilyMemberHistory", "DocumentReference", "Media"} {
		if got := mi.PrimaryCodePath(resource); got != "" {
			t.Errorf("PrimaryCodePath(%s) = %q, want nothing declared", resource, got)
		}
	}
}
