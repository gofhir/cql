// Package model provides version-agnostic FHIR model information for the CQL engine.
//
// ModelInfo maps FHIR types and paths to CQL type system concepts, enabling
// the CQL engine to work with any FHIR version (R4, R4B, R5) by deriving
// type metadata from StructureDefinitions at runtime.
package model

import (
	"sort"
	"strings"
)

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

	// Populated from an official ModelInfo document; empty for hand-built ones.
	retrievable          map[string]bool
	contextKeys          map[string]string        // context name → key element
	conversions          map[conversionKey]string // from/to → converting function
	singleConversion     map[string]string        // from → the only conversion, when there is exactly one
	contextRels          map[contextRelKey]string // type+context → search parameter
	patientClassName     string
	patientBirthDatePath string
}

// conversionKey names a declared conversion between two model types.
type conversionKey struct{ From, To string }

// contextRelKey names how one type relates to one context.
type contextRelKey struct{ Type, Context string }

// NewStaticModelInfo creates a new static model info.
func NewStaticModelInfo(version string) *StaticModelInfo {
	return &StaticModelInfo{
		version:          version,
		types:            make(map[string]*TypeInfo),
		elementTypes:     make(map[string]string),
		choiceTypes:      make(map[string]bool),
		contextTypes:     make(map[string]string),
		codePaths:        make(map[string]string),
		retrievable:      make(map[string]bool),
		contextKeys:      make(map[string]string),
		conversions:      make(map[conversionKey]string),
		singleConversion: make(map[string]string),
		contextRels:      make(map[contextRelKey]string),
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

// PrimaryCodePath returns the element a retrieve filters its codes against, or
// "" when the model declares none.
//
// 84 of the 147 retrievable types declare no primary code path — a retrieve
// naming one has to say which element it means. Answering "code" for those
// would leave a provider unable to tell what the model said from what it did
// not, and Media, ImagingStudy and DocumentReference have no code element at
// all.
func (m *StaticModelInfo) PrimaryCodePath(resourceType string) string {
	return m.codePaths[resourceType]
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

// IsRetrievable reports whether a retrieve may name this type. Only a model
// parsed from an official document knows; a hand-built one says nothing.
func (m *StaticModelInfo) IsRetrievable(typeName string) bool {
	return m.retrievable[typeName]
}

// ContextKeyElement returns the element that identifies the subject of a
// context, which is what narrows a retrieve to one patient.
func (m *StaticModelInfo) ContextKeyElement(contextName string) (string, bool) {
	k, ok := m.contextKeys[contextName]
	return k, ok
}

// ConversionFunction returns the function the model declares for converting
// between two types, if it declares one.
func (m *StaticModelInfo) ConversionFunction(from, to string) (string, bool) {
	fn, ok := m.conversions[conversionKey{From: from, To: to}]
	return fn, ok
}

// Conversion is one declared conversion: the type arrived at, and the function
// that gets there.
type Conversion struct {
	To       string
	Function string
}

// ConversionsFrom lists every conversion the model declares from a type, in a
// stable order.
//
// ConversionFrom answers only when there is exactly one, because the evaluator
// has no way to choose between two. A semantic phase does — it knows the type
// the surrounding expression wants — so it needs all of them, with their
// targets. FHIR.Period declares one conversion, to Interval<System.DateTime>;
// were it to declare two, only the type in hand could say which was meant.
func (m *StaticModelInfo) ConversionsFrom(from string) []Conversion {
	var out []Conversion
	for k, fn := range m.conversions {
		if k.From == from {
			out = append(out, Conversion{To: k.To, Function: fn})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].To < out[j].To })
	return out
}

// ContextSearchParam returns the search parameter relating a resource type to a
// context — "subject" for Observation in a Patient context, "patient" for
// Condition. It is what a data provider needs to scope a retrieve; it is not an
// element path, so the engine cannot navigate it itself.
//
// Two kinds of declaration are refused rather than passed on. A type that *is*
// the context is identified by its own key, not by a relationship to itself:
// the model says Patient relates to Patient through "other", which is
// Patient.link.other and would fetch the linked patients instead. And some
// types declare a FHIRPath fragment — AuditEvent, Provenance, Basic, Person and
// MeasureReport all say "where(resolve() is Patient)" — which is not a search
// parameter and would make a query no server can answer.
func (m *StaticModelInfo) ContextSearchParam(resourceType, contextName string) (string, bool) {
	if strings.EqualFold(unqualify(resourceType), unqualify(contextName)) {
		return "", false
	}
	p, ok := m.contextRels[contextRelKey{Type: resourceType, Context: contextName}]
	if !ok || !isSearchParamName(p) {
		return "", false
	}
	return p, true
}

// isSearchParamName reports whether a declaration looks like a search parameter
// rather than a path expression.
func isSearchParamName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

// indexSingleConversions records, per source type, the one conversion declared
// from it — and records nothing when several are, since choosing between them
// needs a return type this has no way to know.
func (m *StaticModelInfo) indexSingleConversions() {
	counts := make(map[string]int, len(m.conversions))
	for k := range m.conversions {
		counts[k.From]++
	}
	clear(m.singleConversion)
	for k, fn := range m.conversions {
		if counts[k.From] == 1 {
			m.singleConversion[k.From] = fn
		}
	}
}

// ConversionFrom names the one conversion the model declares from a type.
//
// It reports false when the model declares none, and equally when it declares
// more than one: choosing between them needs the return type the context wants,
// which is what a semantic phase would know and this does not.
//
// The index is built once at load. Scanning all 264 conversions per call put a
// linear search and a map allocation on the hot path — both operands of every
// timing expression, and every interval unary.
func (m *StaticModelInfo) ConversionFrom(from string) (string, bool) {
	fn, ok := m.singleConversion[from]
	return fn, ok
}

// PatientClassName returns the model's patient type, e.g. "FHIR.Patient".
func (m *StaticModelInfo) PatientClassName() string { return m.patientClassName }

// IsSubtypeOf reports whether concrete is target or descends from it, walking
// the declared base chain. Comparing names alone made `x is DomainResource`
// false for every resource.
func (m *StaticModelInfo) IsSubtypeOf(concrete, target string) bool {
	if concrete == "" || target == "" {
		return false
	}
	if strings.EqualFold(unqualify(concrete), unqualify(target)) {
		return true
	}
	seen := 0
	for name := concrete; name != ""; seen++ {
		// The chain is a chain, but a malformed document could loop it.
		if seen > 64 {
			return false
		}
		ti, ok := m.types[unqualify(name)]
		if !ok {
			return false
		}
		if strings.EqualFold(unqualify(ti.BaseName), unqualify(target)) {
			return true
		}
		name = ti.BaseName
	}
	return false
}

// unqualify drops a namespace prefix: "FHIR.Patient" becomes "Patient".
func unqualify(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}
	return name
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
