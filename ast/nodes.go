// Package ast defines the Abstract Syntax Tree (AST) node types for CQL.
//
// The AST represents a parsed CQL library as a tree of typed nodes. It serves
// as the intermediate representation between the ANTLR parse tree (concrete
// syntax) and the evaluator. Each node captures the semantic meaning of a CQL
// construct without depending on any particular grammar or parser implementation.
package ast

// ---------------------------------------------------------------------------
// Top-level: Library
// ---------------------------------------------------------------------------

// Library is the root AST node representing a complete CQL library.
type Library struct {
	Identifier *LibraryIdentifier // nil if anonymous
	Usings     []*UsingDef
	Includes   []*IncludeDef
	CodeSystems []*CodeSystemDef
	ValueSets  []*ValueSetDef
	Codes      []*CodeDef
	Concepts   []*ConceptDef
	Parameters []*ParameterDef
	Contexts   []*ContextDef
	Statements []*ExpressionDef
	Functions  []*FunctionDef
}

// LibraryIdentifier holds the library name and optional version.
type LibraryIdentifier struct {
	Name    string // qualified name (e.g. "CMS.Common")
	Version string // e.g. "1.0.0"
}

// ---------------------------------------------------------------------------
// Definitions (library header)
// ---------------------------------------------------------------------------

// UsingDef represents a 'using' declaration (e.g. using FHIR version '4.0.1').
type UsingDef struct {
	Name    string // model name (e.g. "FHIR")
	Version string // model version
	Alias   string // optional 'called' alias
}

// IncludeDef represents an 'include' declaration for referencing another library.
type IncludeDef struct {
	Name    string // qualified library name
	Version string
	Alias   string // local alias ('called' clause)
}

// CodeSystemDef represents a 'codesystem' definition.
type CodeSystemDef struct {
	Name       string // identifier
	ID         string // OID or URL
	Version    string
	AccessLevel AccessLevel
}

// ValueSetDef represents a 'valueset' definition.
type ValueSetDef struct {
	Name        string
	ID          string
	Version     string
	CodeSystems []string // optional codesystem references
	AccessLevel AccessLevel
}

// CodeDef represents a 'code' definition.
type CodeDef struct {
	Name       string
	ID         string
	System     string // codesystem identifier reference
	Display    string
	AccessLevel AccessLevel
}

// ConceptDef represents a 'concept' definition.
type ConceptDef struct {
	Name    string
	Codes   []string // code identifier references
	Display string
	AccessLevel AccessLevel
}

// ParameterDef represents a 'parameter' definition.
type ParameterDef struct {
	Name        string
	Type        TypeSpecifier // optional type constraint
	Default     Expression    // optional default value
	AccessLevel AccessLevel
}

// ContextDef represents a 'context' definition (e.g. context Patient).
type ContextDef struct {
	Model string // optional model qualifier
	Name  string // context name (e.g. "Patient")
}

// ExpressionDef represents a 'define' statement (named expression).
type ExpressionDef struct {
	Name        string
	Expression  Expression
	AccessLevel AccessLevel
	Context     string // resolved from preceding context definition
}

// FunctionDef represents a 'define function' statement.
type FunctionDef struct {
	Name        string
	Operands    []*OperandDef
	ReturnType  TypeSpecifier // optional 'returns' clause
	Body        Expression    // nil if 'external'
	External    bool
	Fluent      bool
	AccessLevel AccessLevel
}

// OperandDef represents a function operand (parameter name + type).
type OperandDef struct {
	Name string
	Type TypeSpecifier
}

// AccessLevel represents CQL access modifiers.
type AccessLevel int

const (
	AccessPublic  AccessLevel = iota // default
	AccessPrivate
)

// ---------------------------------------------------------------------------
// Type Specifiers
// ---------------------------------------------------------------------------

// TypeSpecifier is the interface for all CQL type specifiers.
type TypeSpecifier interface {
	typeSpecifier()
}

// NamedType represents a named type (e.g. FHIR.Patient, System.Integer).
type NamedType struct {
	Namespace string // optional qualifier (e.g. "FHIR", "System")
	Name      string
}

func (*NamedType) typeSpecifier() {}

// ListType represents List<T>.
type ListType struct {
	ElementType TypeSpecifier
}

func (*ListType) typeSpecifier() {}

// IntervalType represents Interval<T>.
type IntervalType struct {
	PointType TypeSpecifier
}

func (*IntervalType) typeSpecifier() {}

// TupleType represents Tuple { field1 Type1, field2 Type2, ... }.
type TupleType struct {
	Elements []*TupleElementDef
}

func (*TupleType) typeSpecifier() {}

// TupleElementDef is a single element in a TupleType.
type TupleElementDef struct {
	Name string
	Type TypeSpecifier
}

// ChoiceType represents Choice<T1, T2, ...>.
type ChoiceType struct {
	Types []TypeSpecifier
}

func (*ChoiceType) typeSpecifier() {}

// ---------------------------------------------------------------------------
// Expressions
// ---------------------------------------------------------------------------

// Expression is the interface implemented by all CQL expression nodes.
type Expression interface {
	expression()
}

// --- Literals ---

// Literal represents a constant value (string, number, boolean, null, date, etc.).
type Literal struct {
	ValueType LiteralType
	Value     string // raw text representation
}

func (*Literal) expression() {}

// LiteralType enumerates the kinds of literal values.
type LiteralType int

const (
	LiteralNull LiteralType = iota
	LiteralBoolean
	LiteralString
	LiteralInteger
	LiteralLong
	LiteralDecimal
	LiteralDate
	LiteralDateTime
	LiteralTime
	LiteralQuantity
	LiteralRatio
)

// --- Identifiers / References ---

// IdentifierRef references a named expression, alias, or identifier.
type IdentifierRef struct {
	Library string // optional library qualifier
	Name    string
}

func (*IdentifierRef) expression() {}

// --- Retrieve ---

// Retrieve represents a data access expression: [Condition: code in "ValSet"].
type Retrieve struct {
	ResourceType   *NamedType
	CodePath       string     // e.g. "code"
	CodeComparator string     // "in", "=", "~"
	Codes          Expression // valueset/code reference or expression
	Context        Expression // optional context identifier (e.g. Patient ->)
	DatePath       string     // e.g. "onset" — date property for date-based filtering
	DateRange      Expression // Interval expression constraining the date property
}

func (*Retrieve) expression() {}

// --- Query ---

// Query represents a CQL query: from ... let ... with/without ... where ... return ... sort.
type Query struct {
	Sources    []*AliasedSource
	Let        []*LetClause
	With       []*WithClause
	Without    []*WithoutClause
	Where      Expression
	Return     *ReturnClause
	Aggregate  *AggregateClause
	Sort       *SortClause
}

func (*Query) expression() {}

// AliasedSource is a query source with an alias.
type AliasedSource struct {
	Source Expression
	Alias  string
}

// LetClause is a 'let' binding in a query.
type LetClause struct {
	Identifier string
	Expression Expression
}

// WithClause is a 'with ... such that' inclusion filter.
type WithClause struct {
	Source    *AliasedSource
	Condition Expression
}

// WithoutClause is a 'without ... such that' exclusion filter.
type WithoutClause struct {
	Source    *AliasedSource
	Condition Expression
}

// ReturnClause specifies what a query returns.
type ReturnClause struct {
	Distinct   bool
	All        bool
	Expression Expression
}

// AggregateClause is an 'aggregate' clause in a query.
type AggregateClause struct {
	Identifier string
	Distinct   bool
	All        bool
	Starting   Expression // optional starting value
	Expression Expression
}

// SortClause specifies query result ordering.
type SortClause struct {
	ByItems   []*SortByItem // non-empty if 'sort by ...'
	Direction SortDirection  // used when 'sort asc/desc' without 'by'
}

// SortByItem is an individual sort key.
type SortByItem struct {
	Expression Expression
	Direction  SortDirection
}

// SortDirection indicates ascending or descending order.
type SortDirection int

const (
	SortAsc  SortDirection = iota
	SortDesc
)

// --- Binary / Unary / Logical ---

// BinaryExpression represents an infix binary operator.
type BinaryExpression struct {
	Operator BinaryOp
	Left     Expression
	Right    Expression
}

func (*BinaryExpression) expression() {}

// BinaryOp enumerates binary operators.
type BinaryOp int

const (
	// Arithmetic
	OpAdd BinaryOp = iota
	OpSubtract
	OpMultiply
	OpDivide
	OpDiv
	OpMod
	OpPower
	OpConcatenate // &

	// Comparison
	OpEqual
	OpNotEqual
	OpEquivalent
	OpNotEquivalent
	OpLess
	OpLessOrEqual
	OpGreater
	OpGreaterOrEqual

	// Logical
	OpAnd
	OpOr
	OpXor
	OpImplies

	// Set operations
	OpUnion
	OpIntersect
	OpExcept

	// Membership
	OpIn
	OpContains
)

// UnaryExpression represents a prefix unary operator.
type UnaryExpression struct {
	Operator UnaryOp
	Operand  Expression
}

func (*UnaryExpression) expression() {}

// UnaryOp enumerates unary operators.
type UnaryOp int

const (
	OpNot UnaryOp = iota
	OpExists
	OpPositive
	OpNegate
	OpDistinct
	OpFlatten
	OpSingletonFrom
	OpPointFrom
	OpStartOf
	OpEndOf
	OpWidthOf
	OpSuccessorOf
	OpPredecessorOf
)

// --- Type Expressions ---

// IsExpression represents 'expression is TypeSpec'.
type IsExpression struct {
	Operand Expression
	Type    TypeSpecifier
}

func (*IsExpression) expression() {}

// AsExpression represents 'expression as TypeSpec'.
type AsExpression struct {
	Operand Expression
	Type    TypeSpecifier
}

func (*AsExpression) expression() {}

// CastExpression represents 'cast expression as TypeSpec'.
type CastExpression struct {
	Operand Expression
	Type    TypeSpecifier
}

func (*CastExpression) expression() {}

// ConvertExpression represents 'convert expression to TypeSpec/unit'.
type ConvertExpression struct {
	Operand  Expression
	ToType   TypeSpecifier // non-nil if converting to type
	ToUnit   string        // non-empty if converting to unit
}

func (*ConvertExpression) expression() {}

// --- Boolean test ---

// BooleanTestExpression represents 'expression is [not] (null|true|false)'.
type BooleanTestExpression struct {
	Operand  Expression
	TestValue string // "null", "true", or "false"
	Not      bool    // true if 'is not'
}

func (*BooleanTestExpression) expression() {}

// --- Conditional ---

// IfThenElse represents an 'if ... then ... else ...' expression.
type IfThenElse struct {
	Condition Expression
	Then      Expression
	Else      Expression
}

func (*IfThenElse) expression() {}

// CaseExpression represents 'case ... when ... then ... else ... end'.
type CaseExpression struct {
	Comparand Expression // optional comparison target
	Items     []*CaseItem
	Else      Expression
}

func (*CaseExpression) expression() {}

// CaseItem is a single when-then pair.
type CaseItem struct {
	When Expression
	Then Expression
}

// --- Between ---

// BetweenExpression represents 'expression [properly] between low and high'.
type BetweenExpression struct {
	Operand  Expression
	Low      Expression
	High     Expression
	Properly bool
}

func (*BetweenExpression) expression() {}

// --- Duration / Difference ---

// DurationBetween represents 'duration in <precision> between x and y'.
type DurationBetween struct {
	Precision string // e.g. "years", "months", "days"
	Low       Expression
	High      Expression
}

func (*DurationBetween) expression() {}

// DifferenceBetween represents 'difference in <precision> between x and y'.
type DifferenceBetween struct {
	Precision string
	Low       Expression
	High      Expression
}

func (*DifferenceBetween) expression() {}

// --- Temporal Extraction ---

// DateTimeComponentFrom represents '<component> from expression'.
type DateTimeComponentFrom struct {
	Component string // year, month, day, hour, minute, second, date, time, timezoneoffset
	Operand   Expression
}

func (*DateTimeComponentFrom) expression() {}

// DurationOf represents 'duration in <precision> of expression'.
type DurationOf struct {
	Precision string
	Operand   Expression
}

func (*DurationOf) expression() {}

// DifferenceOf represents 'difference in <precision> of expression'.
type DifferenceOf struct {
	Precision string
	Operand   Expression
}

func (*DifferenceOf) expression() {}

// --- Type Extent ---

// TypeExtent represents 'minimum TypeSpec' or 'maximum TypeSpec'.
type TypeExtent struct {
	Extent string // "minimum" or "maximum"
	Type   *NamedType
}

func (*TypeExtent) expression() {}

// --- Interval Timing ---

// TimingExpression represents CQL interval timing operators (starts, ends, meets, overlaps, etc.).
type TimingExpression struct {
	Left     Expression
	Right    Expression
	Operator TimingOp
}

func (*TimingExpression) expression() {}

// TimingOp captures the rich CQL interval timing operator phrases.
type TimingOp struct {
	Kind      TimingKind
	Precision string // optional dateTimePrecision (e.g. "day")
	Properly  bool   // 'properly' modifier
	Before    bool   // directional modifier
	After     bool   // directional modifier
}

// TimingKind enumerates the kinds of timing operators.
type TimingKind int

const (
	TimingSameAs TimingKind = iota
	TimingIncludes
	TimingIncludedIn
	TimingDuring
	TimingBeforeOrAfter
	TimingWithin
	TimingMeets
	TimingOverlaps
	TimingStarts
	TimingEnds
)

// --- Membership with precision ---

// MembershipExpression represents 'expression (in|contains) [precision of] expression'.
type MembershipExpression struct {
	Left      Expression
	Right     Expression
	Operator  string // "in" or "contains"
	Precision string // optional dateTimePrecision
}

func (*MembershipExpression) expression() {}

// --- Invocation / Member Access ---

// MemberAccess represents dot-notation: expression.member.
type MemberAccess struct {
	Source Expression
	Member string
}

func (*MemberAccess) expression() {}

// FunctionCall represents a function invocation: expr.func(args...) or func(args...).
type FunctionCall struct {
	Source   Expression   // nil for standalone calls
	Name     string
	Library  string       // optional library qualifier
	Operands []Expression
}

func (*FunctionCall) expression() {}

// IndexAccess represents expression[index].
type IndexAccess struct {
	Source Expression
	Index  Expression
}

func (*IndexAccess) expression() {}

// --- Constructors ---

// IntervalExpression represents 'Interval[low, high]' or 'Interval(low, high)'.
type IntervalExpression struct {
	LowClosed  bool
	HighClosed bool
	Low        Expression
	High       Expression
}

func (*IntervalExpression) expression() {}

// TupleExpression represents 'Tuple { field: expr, ... }' or '{ field: expr, ... }'.
type TupleExpression struct {
	Elements []*TupleElement
}

func (*TupleExpression) expression() {}

// TupleElement is a field in a tuple constructor.
type TupleElement struct {
	Name       string
	Expression Expression
}

// InstanceExpression represents 'TypeName { field: expr, ... }'.
type InstanceExpression struct {
	Type     *NamedType
	Elements []*TupleElement
}

func (*InstanceExpression) expression() {}

// ListExpression represents '{ expr, expr, ... }' or 'List<Type> { ... }'.
type ListExpression struct {
	TypeSpec TypeSpecifier  // optional type annotation
	Elements []Expression
}

func (*ListExpression) expression() {}

// CodeExpression represents 'Code "code" from CodeSystem display "text"'.
type CodeExpression struct {
	Code    string
	System  string
	Display string
}

func (*CodeExpression) expression() {}

// ConceptExpression represents 'Concept { Code ... , Code ... } display "text"'.
type ConceptExpression struct {
	Codes   []*CodeExpression
	Display string
}

func (*ConceptExpression) expression() {}

// --- External Constants ---

// ExternalConstant represents '%name' or '%"name"' (external parameter references).
type ExternalConstant struct {
	Name string
}

func (*ExternalConstant) expression() {}

// --- Special Tokens ---

// ThisExpression represents '$this' in iteration context.
type ThisExpression struct{}

func (*ThisExpression) expression() {}

// IndexExpression represents '$index' in iteration context.
type IndexExpression struct{}

func (*IndexExpression) expression() {}

// TotalExpression represents '$total' in aggregate context.
type TotalExpression struct{}

func (*TotalExpression) expression() {}

// --- Set Aggregate ---

// SetAggregateExpression represents 'expand expr per ...' or 'collapse expr per ...'.
type SetAggregateExpression struct {
	Kind    string     // "expand" or "collapse"
	Operand Expression
	Per     Expression // optional 'per' clause
}

func (*SetAggregateExpression) expression() {}
