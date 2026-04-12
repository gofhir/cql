// Package model provides version-agnostic FHIR model information for the CQL engine.
//
// ModelInfo maps FHIR types and paths to CQL type system concepts, enabling
// the CQL engine to work with any FHIR version (R4, R4B, R5) by deriving
// type metadata from StructureDefinitions at runtime.
package model

import "strings"

// ModelInfo provides type metadata about the FHIR model for CQL evaluation.
type ModelInfo interface { //nolint:revive // stuttering name kept for API clarity
	// TypeInfo returns the type information for a FHIR type name.
	TypeInfo(typeName string) (*TypeInfo, bool)

	// ElementType returns the CQL type of a specific element path.
	ElementType(path string) (string, bool)

	// IsChoiceType returns true if the element at path is a choice type ([x]).
	IsChoiceType(path string) bool

	// ContextType returns the element path for a context name.
	// E.g., "Patient" → "Patient" (the resource type to retrieve).
	ContextType(contextName string) string

	// PrimaryCodePath returns the default code-filter path for a resource type.
	// E.g., "Condition" → "code", "Procedure" → "code".
	PrimaryCodePath(resourceType string) string

	// ElementInfoByPath returns the ElementInfo for a dot-path like "Observation.value".
	ElementInfoByPath(path string) (*ElementInfo, bool)

	// Version returns the FHIR version this model represents.
	Version() string
}

// TypeInfo describes a FHIR/CQL type.
type TypeInfo struct {
	Name       string        // Fully qualified name (e.g., "FHIR.Patient")
	Namespace  string        // "FHIR" or "System"
	BaseName   string        // Base type (e.g., "FHIR.DomainResource")
	Elements   []ElementInfo // Type elements/properties
	PrimaryKey string        // Primary code path for retrieves
}

// ElementInfo describes a single element within a type.
type ElementInfo struct {
	Name        string   // Element name (e.g., "birthDate")
	Type        string   // CQL type name (e.g., "System.Date", "FHIR.HumanName")
	IsList      bool     // True if max cardinality > 1
	IsChoice    bool     // True if element is [x] choice type
	ChoiceTypes []string // Possible types for choice elements
}

// StaticModelInfo is a simple in-memory implementation of ModelInfo
// populated from StructureDefinitions or hardcoded data.
type StaticModelInfo struct {
	version      string
	types        map[string]*TypeInfo
	elementTypes map[string]string // "Patient.birthDate" → "System.Date"
	choiceTypes  map[string]bool
	contextTypes map[string]string
	codePaths    map[string]string // resource type → primary code path
}

// NewStaticModelInfo creates a new static model info.
func NewStaticModelInfo(version string) *StaticModelInfo {
	return &StaticModelInfo{
		version:      version,
		types:        make(map[string]*TypeInfo),
		elementTypes: make(map[string]string),
		choiceTypes:  make(map[string]bool),
		contextTypes: make(map[string]string),
		codePaths:    make(map[string]string),
	}
}

func (m *StaticModelInfo) TypeInfo(typeName string) (*TypeInfo, bool) {
	ti, ok := m.types[typeName]
	return ti, ok
}

func (m *StaticModelInfo) ElementType(path string) (string, bool) {
	t, ok := m.elementTypes[path]
	return t, ok
}

func (m *StaticModelInfo) IsChoiceType(path string) bool {
	return m.choiceTypes[path]
}

func (m *StaticModelInfo) ContextType(contextName string) string {
	if ct, ok := m.contextTypes[contextName]; ok {
		return ct
	}
	return contextName // default: context name is the resource type
}

func (m *StaticModelInfo) PrimaryCodePath(resourceType string) string {
	if cp, ok := m.codePaths[resourceType]; ok {
		return cp
	}
	return "code" // default code path
}

func (m *StaticModelInfo) ElementInfoByPath(path string) (*ElementInfo, bool) {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) != 2 {
		return nil, false
	}
	ti, ok := m.types[parts[0]]
	if !ok {
		return nil, false
	}
	for i := range ti.Elements {
		if ti.Elements[i].Name == parts[1] {
			return &ti.Elements[i], true
		}
	}
	return nil, false
}

func (m *StaticModelInfo) Version() string {
	return m.version
}

// AddType registers a type.
func (m *StaticModelInfo) AddType(ti *TypeInfo) {
	m.types[ti.Name] = ti
	for _, elem := range ti.Elements {
		path := ti.Name + "." + elem.Name
		m.elementTypes[path] = elem.Type
		if elem.IsChoice {
			m.choiceTypes[path] = true
		}
	}
	if ti.PrimaryKey != "" {
		m.codePaths[ti.Name] = ti.PrimaryKey
	}
}

// AddContext registers a context mapping.
func (m *StaticModelInfo) AddContext(name, resourceType string) {
	m.contextTypes[name] = resourceType
}

// DefaultR4ModelInfo returns a minimal R4 model info with common types.
func DefaultR4ModelInfo() *StaticModelInfo {
	mi := NewStaticModelInfo("4.0.1")

	// Register standard contexts
	mi.AddContext("Patient", "Patient")
	mi.AddContext("Practitioner", "Practitioner")
	mi.AddContext("Encounter", "Encounter")

	// Primary code paths for common clinical resources
	mi.codePaths["Condition"] = "code"
	mi.codePaths["Procedure"] = "code"
	mi.codePaths["Observation"] = "code"
	mi.codePaths["MedicationRequest"] = "medication"
	mi.codePaths["Medication"] = "code"
	mi.codePaths["DiagnosticReport"] = "code"
	mi.codePaths["Encounter"] = "type"
	mi.codePaths["AllergyIntolerance"] = "code"
	mi.codePaths["Immunization"] = "vaccineCode"
	mi.codePaths["ServiceRequest"] = "code"

	// Register Patient type with common elements
	mi.AddType(&TypeInfo{
		Name:      "Patient",
		Namespace: "FHIR",
		BaseName:  "FHIR.DomainResource",
		Elements: []ElementInfo{
			{Name: "id", Type: "System.String"},
			{Name: "birthDate", Type: "System.Date"},
			{Name: "gender", Type: "System.String"},
			{Name: "name", Type: "FHIR.HumanName", IsList: true},
			{Name: "identifier", Type: "FHIR.Identifier", IsList: true},
			{Name: "active", Type: "System.Boolean"},
			{Name: "deceased", IsChoice: true, ChoiceTypes: []string{"System.Boolean", "System.DateTime"}},
			{Name: "address", Type: "FHIR.Address", IsList: true},
			{Name: "telecom", Type: "FHIR.ContactPoint", IsList: true},
		},
	})

	// Register Condition
	mi.AddType(&TypeInfo{
		Name:       "Condition",
		Namespace:  "FHIR",
		BaseName:   "FHIR.DomainResource",
		PrimaryKey: "code",
		Elements: []ElementInfo{
			{Name: "id", Type: "System.String"},
			{Name: "code", Type: "FHIR.CodeableConcept"},
			{Name: "subject", Type: "FHIR.Reference"},
			{Name: "onset", IsChoice: true, ChoiceTypes: []string{"System.DateTime", "FHIR.Age", "FHIR.Period", "FHIR.Range", "System.String"}},
			{Name: "clinicalStatus", Type: "FHIR.CodeableConcept"},
			{Name: "verificationStatus", Type: "FHIR.CodeableConcept"},
			{Name: "category", Type: "FHIR.CodeableConcept", IsList: true},
		},
	})

	// Register Observation
	mi.AddType(&TypeInfo{
		Name:       "Observation",
		Namespace:  "FHIR",
		BaseName:   "FHIR.DomainResource",
		PrimaryKey: "code",
		Elements: []ElementInfo{
			{Name: "id", Type: "System.String"},
			{Name: "code", Type: "FHIR.CodeableConcept"},
			{Name: "subject", Type: "FHIR.Reference"},
			{Name: "value", IsChoice: true, ChoiceTypes: []string{"FHIR.Quantity", "FHIR.CodeableConcept", "System.String", "System.Boolean", "System.Integer", "FHIR.Range", "FHIR.Ratio", "FHIR.SampledData", "System.DateTime", "FHIR.Period"}},
			{Name: "effective", IsChoice: true, ChoiceTypes: []string{"System.DateTime", "FHIR.Period", "FHIR.Timing", "System.DateTime"}},
			{Name: "status", Type: "System.String"},
		},
	})

	// Register Encounter
	mi.AddType(&TypeInfo{
		Name:       "Encounter",
		Namespace:  "FHIR",
		BaseName:   "FHIR.DomainResource",
		PrimaryKey: "type",
		Elements: []ElementInfo{
			{Name: "id", Type: "System.String"},
			{Name: "type", Type: "FHIR.CodeableConcept", IsList: true},
			{Name: "class", Type: "FHIR.Coding"},
			{Name: "status", Type: "System.String"},
			{Name: "period", Type: "FHIR.Period"},
			{Name: "subject", Type: "FHIR.Reference"},
		},
	})

	// Register Procedure
	mi.AddType(&TypeInfo{
		Name:       "Procedure",
		Namespace:  "FHIR",
		BaseName:   "FHIR.DomainResource",
		PrimaryKey: "code",
		Elements: []ElementInfo{
			{Name: "id", Type: "System.String"},
			{Name: "code", Type: "FHIR.CodeableConcept"},
			{Name: "subject", Type: "FHIR.Reference"},
			{Name: "performed", IsChoice: true, ChoiceTypes: []string{"System.DateTime", "FHIR.Period", "System.String", "FHIR.Age", "FHIR.Range"}},
			{Name: "status", Type: "System.String"},
		},
	})

	// Register MedicationRequest
	mi.AddType(&TypeInfo{
		Name:       "MedicationRequest",
		Namespace:  "FHIR",
		BaseName:   "FHIR.DomainResource",
		PrimaryKey: "medication",
		Elements: []ElementInfo{
			{Name: "id", Type: "System.String"},
			{Name: "medication", IsChoice: true, ChoiceTypes: []string{"FHIR.CodeableConcept", "FHIR.Reference"}},
			{Name: "subject", Type: "FHIR.Reference"},
			{Name: "status", Type: "System.String"},
			{Name: "authoredOn", Type: "System.DateTime"},
		},
	})

	return mi
}
