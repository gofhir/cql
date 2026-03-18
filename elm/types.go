// Package elm implements the HL7 CQL Expression Logic Model (ELM) serialization.
//
// ELM is the canonical, machine-processable representation of CQL defined by HL7.
// It provides a JSON-based interchange format that enables CQL expressions to be
// shared across implementations (HAPI FHIR, CQL.js, etc.) without re-parsing.
//
// This package supports:
//   - ELM data model types matching the HL7 specification
//   - AST → ELM translation (export)
//   - ELM → AST translation (import)
//   - Polymorphic JSON marshaling/unmarshaling via "type" discriminator
//
// Reference: https://cql.hl7.org/elm.html
package elm

// ---------------------------------------------------------------------------
// Library (root container)
// ---------------------------------------------------------------------------

// Library is the root ELM element representing a complete CQL library.
type Library struct {
	Identifier       *VersionedIdentifier `json:"identifier,omitempty"`
	SchemaIdentifier *VersionedIdentifier `json:"schemaIdentifier,omitempty"`
	Usings           *UsingDefs           `json:"usings,omitempty"`
	Includes         *IncludeDefs         `json:"includes,omitempty"`
	Parameters       *ParameterDefs       `json:"parameters,omitempty"`
	CodeSystems      *CodeSystemDefs      `json:"codeSystems,omitempty"`
	ValueSets        *ValueSetDefs        `json:"valueSets,omitempty"`
	Codes            *CodeDefs            `json:"codes,omitempty"`
	Concepts         *ConceptDefs         `json:"concepts,omitempty"`
	Contexts         *ContextDefs         `json:"contexts,omitempty"`
	Statements       *Statements          `json:"statements,omitempty"`
}

// VersionedIdentifier identifies a library by name and version.
type VersionedIdentifier struct {
	ID      string `json:"id,omitempty"`
	System  string `json:"system,omitempty"`
	Version string `json:"version,omitempty"`
}

// ---------------------------------------------------------------------------
// Definition containers
// ---------------------------------------------------------------------------

// UsingDefs wraps the list of using definitions.
type UsingDefs struct {
	Def []*UsingDef `json:"def,omitempty"`
}

// UsingDef represents a model usage declaration.
type UsingDef struct {
	LocalIdentifier string `json:"localIdentifier,omitempty"`
	URI             string `json:"uri,omitempty"`
	Version         string `json:"version,omitempty"`
}

// IncludeDefs wraps the list of include definitions.
type IncludeDefs struct {
	Def []*IncludeDef `json:"def,omitempty"`
}

// IncludeDef represents a library include declaration.
type IncludeDef struct {
	LocalIdentifier string `json:"localIdentifier,omitempty"`
	Path            string `json:"path,omitempty"`
	Version         string `json:"version,omitempty"`
}

// ParameterDefs wraps the list of parameter definitions.
type ParameterDefs struct {
	Def []*ParameterDef `json:"def,omitempty"`
}

// ParameterDef represents a library parameter.
type ParameterDef struct {
	Name          string          `json:"name,omitempty"`
	AccessLevel   string          `json:"accessLevel,omitempty"`
	ParameterType *TypeSpecifier  `json:"parameterTypeSpecifier,omitempty"`
	Default       *ExpressionNode `json:"default,omitempty"`
}

// CodeSystemDefs wraps the list of code system definitions.
type CodeSystemDefs struct {
	Def []*CodeSystemDef `json:"def,omitempty"`
}

// CodeSystemDef represents a code system definition.
type CodeSystemDef struct {
	Name        string `json:"name,omitempty"`
	ID          string `json:"id,omitempty"`
	Version     string `json:"version,omitempty"`
	AccessLevel string `json:"accessLevel,omitempty"`
}

// ValueSetDefs wraps the list of value set definitions.
type ValueSetDefs struct {
	Def []*ValueSetDef `json:"def,omitempty"`
}

// ValueSetDef represents a value set definition.
type ValueSetDef struct {
	Name        string   `json:"name,omitempty"`
	ID          string   `json:"id,omitempty"`
	Version     string   `json:"version,omitempty"`
	AccessLevel string   `json:"accessLevel,omitempty"`
	CodeSystem  []string `json:"codeSystem,omitempty"`
}

// CodeDefs wraps the list of code definitions.
type CodeDefs struct {
	Def []*CodeDef `json:"def,omitempty"`
}

// CodeDef represents a code definition.
type CodeDef struct {
	Name        string         `json:"name,omitempty"`
	ID          string         `json:"id,omitempty"`
	Display     string         `json:"display,omitempty"`
	AccessLevel string         `json:"accessLevel,omitempty"`
	CodeSystem  *CodeSystemRef `json:"codeSystem,omitempty"`
}

// ConceptDefs wraps the list of concept definitions.
type ConceptDefs struct {
	Def []*ConceptDef `json:"def,omitempty"`
}

// ConceptDef represents a concept definition.
type ConceptDef struct {
	Name        string     `json:"name,omitempty"`
	Display     string     `json:"display,omitempty"`
	AccessLevel string     `json:"accessLevel,omitempty"`
	Code        []*CodeRef `json:"code,omitempty"`
}

// ContextDefs wraps the list of context definitions.
type ContextDefs struct {
	Def []*ContextDef `json:"def,omitempty"`
}

// ContextDef represents a context definition.
type ContextDef struct {
	Name string `json:"name,omitempty"`
}

// Statements wraps the list of expression definitions.
type Statements struct {
	Def []*ExpressionDef `json:"def,omitempty"`
}

// ExpressionDef represents a named expression definition.
type ExpressionDef struct {
	Name        string          `json:"name,omitempty"`
	Context     string          `json:"context,omitempty"`
	AccessLevel string          `json:"accessLevel,omitempty"`
	Expression  *ExpressionNode `json:"expression,omitempty"`
}

// FunctionDef represents a named function definition (extends ExpressionDef).
type FunctionDef struct {
	Name        string          `json:"name,omitempty"`
	Context     string          `json:"context,omitempty"`
	AccessLevel string          `json:"accessLevel,omitempty"`
	Expression  *ExpressionNode `json:"expression,omitempty"`
	Operand     []*OperandDef   `json:"operand,omitempty"`
	External    bool            `json:"external,omitempty"`
	Fluent      bool            `json:"fluent,omitempty"`
}

// OperandDef represents a function operand definition.
type OperandDef struct {
	Name        string         `json:"name,omitempty"`
	OperandType *TypeSpecifier `json:"operandTypeSpecifier,omitempty"`
}

// ---------------------------------------------------------------------------
// Type specifiers
// ---------------------------------------------------------------------------

// TypeSpecifier represents an ELM type specifier with a discriminator.
type TypeSpecifier struct {
	Type        string           `json:"type"` // discriminator: NamedTypeSpecifier, ListTypeSpecifier, etc.
	Namespace   string           `json:"namespace,omitempty"`
	Name        string           `json:"name,omitempty"`
	ElementType *TypeSpecifier   `json:"elementType,omitempty"` // for ListTypeSpecifier
	PointType   *TypeSpecifier   `json:"pointType,omitempty"`   // for IntervalTypeSpecifier
	Element     []*TupleElement  `json:"element,omitempty"`     // for TupleTypeSpecifier
	Choice      []*TypeSpecifier `json:"choice,omitempty"`      // for ChoiceTypeSpecifier
}

// TupleElement is a single element in a tuple type specifier.
type TupleElement struct {
	Name        string         `json:"name,omitempty"`
	ElementType *TypeSpecifier `json:"elementType,omitempty"`
}

// ---------------------------------------------------------------------------
// References
// ---------------------------------------------------------------------------

// CodeSystemRef references a code system by name.
type CodeSystemRef struct {
	Name        string `json:"name,omitempty"`
	LibraryName string `json:"libraryName,omitempty"`
}

// CodeRef references a code definition by name.
type CodeRef struct {
	Name        string `json:"name,omitempty"`
	LibraryName string `json:"libraryName,omitempty"`
}

// ValueSetRef references a value set definition.
type ValueSetRef struct {
	Name        string `json:"name,omitempty"`
	LibraryName string `json:"libraryName,omitempty"`
}

// ---------------------------------------------------------------------------
// Expression node (polymorphic via "type" discriminator)
// ---------------------------------------------------------------------------

// ExpressionNode is the universal expression container used in ELM JSON.
// The Type field acts as a discriminator to determine which fields are active.
// This mirrors the HL7 ELM JSON representation where each expression node
// carries a "type" field (e.g., "Literal", "Retrieve", "Equal", "And").
type ExpressionNode struct {
	// Discriminator — determines which fields below are populated.
	Type string `json:"type"`

	// -- Shared metadata --
	ResultTypeName      string         `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier *TypeSpecifier `json:"resultTypeSpecifier,omitempty"`

	// -- Literal --
	ValueType string `json:"valueType,omitempty"` // e.g. "{urn:hl7-org:elm-types:r1}Integer"
	Value     string `json:"value,omitempty"`

	// -- IdentifierRef / ExpressionRef / ParameterRef --
	Name        string `json:"name,omitempty"`
	LibraryName string `json:"libraryName,omitempty"`

	// -- Property (member access) --
	Path   string          `json:"path,omitempty"`
	Source *ExpressionNode `json:"source,omitempty"`
	Scope  string          `json:"scope,omitempty"`

	// -- Retrieve --
	DataType       string          `json:"dataType,omitempty"`
	TemplateID     string          `json:"templateId,omitempty"`
	CodeProperty   string          `json:"codeProperty,omitempty"`
	CodeComparator string          `json:"codeComparator,omitempty"`
	Codes          *ExpressionNode `json:"codes,omitempty"`
	DateProperty   string          `json:"dateProperty,omitempty"`
	DateRange      *ExpressionNode `json:"dateRange,omitempty"`

	// -- Unary / Binary operands --
	Operand interface{} `json:"operand,omitempty"` // *ExpressionNode or []*ExpressionNode

	// -- If/Then/Else --
	Condition *ExpressionNode `json:"condition,omitempty"`
	Then      *ExpressionNode `json:"then,omitempty"`
	Else      *ExpressionNode `json:"else,omitempty"`

	// -- Case --
	Comparand *ExpressionNode `json:"comparand,omitempty"`
	CaseItem  []*CaseItem     `json:"caseItem,omitempty"`

	// -- Query --
	SourceClause []*AliasedQuerySource `json:"sourceClause,omitempty"`
	Let          []*LetClause          `json:"let,omitempty"`
	Relationship []*RelationshipClause `json:"relationship,omitempty"`
	Where        *ExpressionNode       `json:"where,omitempty"`
	Return       *ReturnClause         `json:"return,omitempty"`
	Sort         *SortClause           `json:"sort,omitempty"`
	Aggregate    *AggregateClause      `json:"aggregate,omitempty"`

	// -- Interval --
	Low        *ExpressionNode `json:"low,omitempty"`
	High       *ExpressionNode `json:"high,omitempty"`
	LowClosed  *bool           `json:"lowClosed,omitempty"`
	HighClosed *bool           `json:"highClosed,omitempty"`

	// -- List --
	Element []*ExpressionNode `json:"element,omitempty"`

	// -- Tuple / Instance --
	ClassType string `json:"classType,omitempty"` // for Instance

	// -- FunctionRef --
	FunctionName string `json:"functionName,omitempty"`

	// -- Code/Concept constructors --
	CodeValue string          `json:"code,omitempty"`
	System    *ExpressionNode `json:"system,omitempty"`
	Display   string          `json:"display,omitempty"`

	// -- Type operations --
	IsTypeSpecifier *TypeSpecifier `json:"isTypeSpecifier,omitempty"`
	AsTypeSpecifier *TypeSpecifier `json:"asTypeSpecifier,omitempty"`
	Strict          bool           `json:"strict,omitempty"`
	ToType          *TypeSpecifier `json:"toTypeSpecifier,omitempty"`

	// -- DateTime component --
	Precision string `json:"precision,omitempty"`

	// -- Temporal / Between / Duration --
	LowExpr  *ExpressionNode `json:"lowExpr,omitempty"`
	HighExpr *ExpressionNode `json:"highExpr,omitempty"`

	// -- External constant --
	ExternalName string `json:"externalName,omitempty"`

	// -- SetAggregate --
	Per *ExpressionNode `json:"per,omitempty"`

	// -- BooleanTest --
	TestValue string `json:"testValue,omitempty"`
	IsNot     bool   `json:"isNot,omitempty"`

	// -- TypeExtent --
	Extent string `json:"extent,omitempty"`
}

// CaseItem represents a when-then pair in a Case expression.
type CaseItem struct {
	When *ExpressionNode `json:"when,omitempty"`
	Then *ExpressionNode `json:"then,omitempty"`
}

// AliasedQuerySource represents a query source with an alias.
type AliasedQuerySource struct {
	Expression *ExpressionNode `json:"expression,omitempty"`
	Alias      string          `json:"alias,omitempty"`
}

// LetClause represents a let binding in a query.
type LetClause struct {
	Identifier string          `json:"identifier,omitempty"`
	Expression *ExpressionNode `json:"expression,omitempty"`
}

// RelationshipClause represents a with/without clause in a query.
type RelationshipClause struct {
	Type       string          `json:"type"` // "With" or "Without"
	Expression *ExpressionNode `json:"expression,omitempty"`
	Alias      string          `json:"alias,omitempty"`
	SuchThat   *ExpressionNode `json:"suchThat,omitempty"`
}

// ReturnClause specifies what a query returns.
type ReturnClause struct {
	Expression *ExpressionNode `json:"expression,omitempty"`
	Distinct   bool            `json:"distinct,omitempty"`
}

// SortClause specifies query result ordering.
type SortClause struct {
	By []*SortByItem `json:"by,omitempty"`
}

// SortByItem is an individual sort key.
type SortByItem struct {
	Direction  string          `json:"direction,omitempty"` // "asc" or "desc"
	Type       string          `json:"type,omitempty"`      // "ByExpression" or "ByDirection"
	Expression *ExpressionNode `json:"expression,omitempty"`
}

// AggregateClause represents an aggregate clause in a query.
type AggregateClause struct {
	Identifier string          `json:"identifier,omitempty"`
	Distinct   bool            `json:"distinct,omitempty"`
	Starting   *ExpressionNode `json:"starting,omitempty"`
	Expression *ExpressionNode `json:"expression,omitempty"`
}
