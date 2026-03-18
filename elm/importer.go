package elm

import (
	"fmt"
	"strings"

	"github.com/gofhir/cql/ast"
)

// Import converts an ELM Library into a go-cql AST Library for evaluation.
func Import(lib *Library) (*ast.Library, error) {
	if lib == nil {
		return nil, fmt.Errorf("elm: nil library")
	}

	out := &ast.Library{}

	if lib.Identifier != nil {
		out.Identifier = &ast.LibraryIdentifier{
			Name:    lib.Identifier.ID,
			Version: lib.Identifier.Version,
		}
	}

	out.Usings = importUsings(lib.Usings)
	out.Includes = importIncludes(lib.Includes)
	out.Parameters = importParameters(lib.Parameters)
	out.CodeSystems = importCodeSystems(lib.CodeSystems)
	out.ValueSets = importValueSets(lib.ValueSets)
	out.Codes = importCodes(lib.Codes)
	out.Concepts = importConcepts(lib.Concepts)
	out.Contexts = importContexts(lib.Contexts)
	out.Statements = importStatements(lib.Statements)

	return out, nil
}

// ---------------------------------------------------------------------------
// Definitions
// ---------------------------------------------------------------------------

func importUsings(defs *UsingDefs) []*ast.UsingDef {
	if defs == nil {
		return nil
	}
	var out []*ast.UsingDef
	for _, d := range defs.Def {
		out = append(out, &ast.UsingDef{
			Name:    d.LocalIdentifier,
			Version: d.Version,
		})
	}
	return out
}

func importIncludes(defs *IncludeDefs) []*ast.IncludeDef {
	if defs == nil {
		return nil
	}
	var out []*ast.IncludeDef
	for _, d := range defs.Def {
		out = append(out, &ast.IncludeDef{
			Name:    d.Path,
			Version: d.Version,
			Alias:   d.LocalIdentifier,
		})
	}
	return out
}

func importParameters(defs *ParameterDefs) []*ast.ParameterDef {
	if defs == nil {
		return nil
	}
	var out []*ast.ParameterDef
	for _, d := range defs.Def {
		p := &ast.ParameterDef{
			Name:        d.Name,
			AccessLevel: importAccessLevel(d.AccessLevel),
		}
		if d.ParameterType != nil {
			p.Type = importTypeSpecifier(d.ParameterType)
		}
		if d.Default != nil {
			p.Default = ImportExpression(d.Default)
		}
		out = append(out, p)
	}
	return out
}

func importCodeSystems(defs *CodeSystemDefs) []*ast.CodeSystemDef {
	if defs == nil {
		return nil
	}
	var out []*ast.CodeSystemDef
	for _, d := range defs.Def {
		out = append(out, &ast.CodeSystemDef{
			Name:        d.Name,
			ID:          d.ID,
			Version:     d.Version,
			AccessLevel: importAccessLevel(d.AccessLevel),
		})
	}
	return out
}

func importValueSets(defs *ValueSetDefs) []*ast.ValueSetDef {
	if defs == nil {
		return nil
	}
	var out []*ast.ValueSetDef
	for _, d := range defs.Def {
		out = append(out, &ast.ValueSetDef{
			Name:        d.Name,
			ID:          d.ID,
			Version:     d.Version,
			AccessLevel: importAccessLevel(d.AccessLevel),
			CodeSystems: d.CodeSystem,
		})
	}
	return out
}

func importCodes(defs *CodeDefs) []*ast.CodeDef {
	if defs == nil {
		return nil
	}
	var out []*ast.CodeDef
	for _, d := range defs.Def {
		cd := &ast.CodeDef{
			Name:        d.Name,
			ID:          d.ID,
			Display:     d.Display,
			AccessLevel: importAccessLevel(d.AccessLevel),
		}
		if d.CodeSystem != nil {
			cd.System = d.CodeSystem.Name
		}
		out = append(out, cd)
	}
	return out
}

func importConcepts(defs *ConceptDefs) []*ast.ConceptDef {
	if defs == nil {
		return nil
	}
	var out []*ast.ConceptDef
	for _, d := range defs.Def {
		cd := &ast.ConceptDef{
			Name:        d.Name,
			Display:     d.Display,
			AccessLevel: importAccessLevel(d.AccessLevel),
		}
		for _, c := range d.Code {
			cd.Codes = append(cd.Codes, c.Name)
		}
		out = append(out, cd)
	}
	return out
}

func importContexts(defs *ContextDefs) []*ast.ContextDef {
	if defs == nil {
		return nil
	}
	var out []*ast.ContextDef
	for _, d := range defs.Def {
		out = append(out, &ast.ContextDef{Name: d.Name})
	}
	return out
}

func importStatements(stmts *Statements) []*ast.ExpressionDef {
	if stmts == nil {
		return nil
	}
	var out []*ast.ExpressionDef
	for _, d := range stmts.Def {
		out = append(out, &ast.ExpressionDef{
			Name:        d.Name,
			Context:     d.Context,
			AccessLevel: importAccessLevel(d.AccessLevel),
			Expression:  ImportExpression(d.Expression),
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Type specifiers
// ---------------------------------------------------------------------------

func importTypeSpecifier(ts *TypeSpecifier) ast.TypeSpecifier {
	if ts == nil {
		return nil
	}
	switch ts.Type {
	case "NamedTypeSpecifier":
		return &ast.NamedType{Namespace: ts.Namespace, Name: ts.Name}
	case "ListTypeSpecifier":
		return &ast.ListType{ElementType: importTypeSpecifier(ts.ElementType)}
	case "IntervalTypeSpecifier":
		return &ast.IntervalType{PointType: importTypeSpecifier(ts.PointType)}
	case "TupleTypeSpecifier":
		tt := &ast.TupleType{}
		for _, e := range ts.Element {
			tt.Elements = append(tt.Elements, &ast.TupleElementDef{
				Name: e.Name,
				Type: importTypeSpecifier(e.ElementType),
			})
		}
		return tt
	case "ChoiceTypeSpecifier":
		ct := &ast.ChoiceType{}
		for _, c := range ts.Choice {
			ct.Types = append(ct.Types, importTypeSpecifier(c))
		}
		return ct
	default:
		return &ast.NamedType{Name: ts.Type}
	}
}

func importAccessLevel(s string) ast.AccessLevel {
	if strings.EqualFold(s, "Private") {
		return ast.AccessPrivate
	}
	return ast.AccessPublic
}

// ---------------------------------------------------------------------------
// Expression import (ELM ExpressionNode → AST Expression)
// ---------------------------------------------------------------------------

// ImportExpression converts an ELM ExpressionNode to a go-cql AST Expression.
func ImportExpression(node *ExpressionNode) ast.Expression {
	if node == nil {
		return nil
	}

	switch node.Type {
	// Literals
	case "Null":
		return &ast.Literal{ValueType: ast.LiteralNull}
	case "Literal":
		return importLiteral(node)

	// References
	case "ExpressionRef":
		return &ast.IdentifierRef{Name: node.Name, Library: node.LibraryName}
	case "ParameterRef":
		return &ast.ExternalConstant{Name: node.Name}

	// Property (member access)
	case "Property":
		return &ast.MemberAccess{
			Source: ImportExpression(node.Source),
			Member: node.Path,
		}

	// Retrieve
	case "Retrieve":
		return importRetrieve(node)

	// Query
	case "Query":
		return importQuery(node)

	// Binary operators
	case "Add", "Subtract", "Multiply", "Divide", "TruncatedDivide", "Modulo", "Power",
		"Concatenate", "Equal", "NotEqual", "Equivalent", "NotEquivalent",
		"Less", "LessOrEqual", "Greater", "GreaterOrEqual",
		"And", "Or", "Xor", "Implies",
		"Union", "Intersect", "Except", "In", "Contains":
		return importBinary(node)

	// Unary operators
	case "Not", "Exists", "Abs", "Negate", "Distinct", "Flatten",
		"SingletonFrom", "PointFrom", "Start", "End", "Width",
		"Successor", "Predecessor":
		return importUnary(node)

	// Boolean tests
	case "IsNull":
		return &ast.BooleanTestExpression{
			Operand:   importSingleOperand(node),
			TestValue: "null",
		}
	case "IsTrue":
		return &ast.BooleanTestExpression{
			Operand:   importSingleOperand(node),
			TestValue: "true",
		}
	case "IsFalse":
		return &ast.BooleanTestExpression{
			Operand:   importSingleOperand(node),
			TestValue: "false",
		}

	// Type operations
	case "Is":
		return &ast.IsExpression{
			Operand: importSingleOperand(node),
			Type:    importTypeSpecifier(node.IsTypeSpecifier),
		}
	case "As":
		if node.Strict {
			return &ast.CastExpression{
				Operand: importSingleOperand(node),
				Type:    importTypeSpecifier(node.AsTypeSpecifier),
			}
		}
		return &ast.AsExpression{
			Operand: importSingleOperand(node),
			Type:    importTypeSpecifier(node.AsTypeSpecifier),
		}
	case "Convert":
		return &ast.ConvertExpression{
			Operand: importSingleOperand(node),
			ToType:  importTypeSpecifier(node.ToType),
		}

	// Conditional
	case "If":
		return &ast.IfThenElse{
			Condition: ImportExpression(node.Condition),
			Then:      ImportExpression(node.Then),
			Else:      ImportExpression(node.Else),
		}
	case "Case":
		return importCase(node)

	// Duration/Difference
	case "DurationBetween":
		ops := importOperandSlice(node)
		if len(ops) == 2 {
			return &ast.DurationBetween{
				Precision: node.Precision,
				Low:       ops[0],
				High:      ops[1],
			}
		}
		return &ast.DurationBetween{Precision: node.Precision}
	case "DifferenceBetween":
		ops := importOperandSlice(node)
		if len(ops) == 2 {
			return &ast.DifferenceBetween{
				Precision: node.Precision,
				Low:       ops[0],
				High:      ops[1],
			}
		}
		return &ast.DifferenceBetween{Precision: node.Precision}

	// DateTime extraction
	case "DateTimeComponentFrom":
		return &ast.DateTimeComponentFrom{
			Component: node.Precision,
			Operand:   importSingleOperand(node),
		}
	case "DurationOf":
		return &ast.DurationOf{
			Precision: node.Precision,
			Operand:   importSingleOperand(node),
		}
	case "DifferenceOf":
		return &ast.DifferenceOf{
			Precision: node.Precision,
			Operand:   importSingleOperand(node),
		}

	// Timing operators
	case "SameAs", "Includes", "IncludedIn", "Before", "After",
		"SameOrBefore", "SameOrAfter", "Meets", "MeetsBefore", "MeetsAfter",
		"Overlaps", "OverlapsBefore", "OverlapsAfter", "Starts", "Ends",
		"ProperIncludes", "ProperIncludedIn":
		return importTiming(node)

	// Interval
	case "Interval":
		return importInterval(node)

	// List
	case "List":
		return importList(node)

	// Tuple
	case "Tuple":
		return importTupleExpr(node)

	// Instance
	case "Instance":
		return importInstanceExpr(node)

	// Code/Concept
	case "Code":
		return importCodeExpr(node)
	case "Concept":
		return importConceptExpr(node)

	// Function call
	case "FunctionRef":
		return importFunctionCall(node)

	// Indexer
	case "Indexer":
		ops := importOperandSlice(node)
		if len(ops) == 2 {
			return &ast.IndexAccess{Source: ops[0], Index: ops[1]}
		}
		return &ast.IndexAccess{}

	// Special tokens
	case "This":
		return &ast.ThisExpression{}
	case "IterationIndex":
		return &ast.IndexExpression{}
	case "Total":
		return &ast.TotalExpression{}

	// Set aggregate
	case "Expand", "Collapse":
		return &ast.SetAggregateExpression{
			Kind:    strings.ToLower(node.Type),
			Operand: importSingleOperand(node),
			Per:     ImportExpression(node.Per),
		}

	// Minimum/Maximum
	case "Minimum", "Maximum":
		return &ast.TypeExtent{
			Extent: strings.ToLower(node.Type),
			Type:   importTypeExtentTarget(node),
		}

	default:
		// Unknown types → literal placeholder
		return &ast.Literal{ValueType: ast.LiteralNull}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func importLiteral(node *ExpressionNode) *ast.Literal {
	lt := ast.LiteralString
	switch {
	case strings.HasSuffix(node.ValueType, "}Boolean"):
		lt = ast.LiteralBoolean
	case strings.HasSuffix(node.ValueType, "}Integer"):
		lt = ast.LiteralInteger
	case strings.HasSuffix(node.ValueType, "}Long"):
		lt = ast.LiteralLong
	case strings.HasSuffix(node.ValueType, "}Decimal"):
		lt = ast.LiteralDecimal
	case strings.HasSuffix(node.ValueType, "}String"):
		lt = ast.LiteralString
	case strings.HasSuffix(node.ValueType, "}Date"):
		lt = ast.LiteralDate
	case strings.HasSuffix(node.ValueType, "}DateTime"):
		lt = ast.LiteralDateTime
	case strings.HasSuffix(node.ValueType, "}Time"):
		lt = ast.LiteralTime
	case strings.HasSuffix(node.ValueType, "}Quantity"):
		lt = ast.LiteralQuantity
	case strings.HasSuffix(node.ValueType, "}Ratio"):
		lt = ast.LiteralRatio
	}
	return &ast.Literal{ValueType: lt, Value: node.Value}
}

func importRetrieve(node *ExpressionNode) *ast.Retrieve {
	r := &ast.Retrieve{
		CodePath:       node.CodeProperty,
		CodeComparator: node.CodeComparator,
		DatePath:       node.DateProperty,
	}
	if node.DataType != "" {
		r.ResourceType = parseDataType(node.DataType)
	}
	if node.Codes != nil {
		r.Codes = ImportExpression(node.Codes)
	}
	if node.DateRange != nil {
		r.DateRange = ImportExpression(node.DateRange)
	}
	return r
}

func parseDataType(dt string) *ast.NamedType {
	// Format: "{namespace}Name" or just "Name"
	if idx := strings.LastIndex(dt, "}"); idx >= 0 && strings.HasPrefix(dt, "{") {
		ns := dt[1:idx]
		name := dt[idx+1:]
		// Convert well-known URIs back to namespace aliases
		if ns == "http://hl7.org/fhir" {
			ns = "FHIR"
		}
		return &ast.NamedType{Namespace: ns, Name: name}
	}
	return &ast.NamedType{Name: dt}
}

var elmBinaryOps = map[string]ast.BinaryOp{
	"Add":             ast.OpAdd,
	"Subtract":        ast.OpSubtract,
	"Multiply":        ast.OpMultiply,
	"Divide":          ast.OpDivide,
	"TruncatedDivide": ast.OpDiv,
	"Modulo":          ast.OpMod,
	"Power":           ast.OpPower,
	"Concatenate":     ast.OpConcatenate,
	"Equal":           ast.OpEqual,
	"NotEqual":        ast.OpNotEqual,
	"Equivalent":      ast.OpEquivalent,
	"NotEquivalent":   ast.OpNotEquivalent,
	"Less":            ast.OpLess,
	"LessOrEqual":     ast.OpLessOrEqual,
	"Greater":         ast.OpGreater,
	"GreaterOrEqual":  ast.OpGreaterOrEqual,
	"And":             ast.OpAnd,
	"Or":              ast.OpOr,
	"Xor":             ast.OpXor,
	"Implies":         ast.OpImplies,
	"Union":           ast.OpUnion,
	"Intersect":       ast.OpIntersect,
	"Except":          ast.OpExcept,
	"In":              ast.OpIn,
	"Contains":        ast.OpContains,
}

func importBinary(node *ExpressionNode) ast.Expression {
	op, ok := elmBinaryOps[node.Type]
	if !ok {
		return &ast.Literal{ValueType: ast.LiteralNull}
	}
	ops := importOperandSlice(node)
	if len(ops) < 2 {
		return &ast.BinaryExpression{Operator: op}
	}
	return &ast.BinaryExpression{
		Operator: op,
		Left:     ops[0],
		Right:    ops[1],
	}
}

var elmUnaryOps = map[string]ast.UnaryOp{
	"Not":           ast.OpNot,
	"Exists":        ast.OpExists,
	"Abs":           ast.OpPositive,
	"Negate":        ast.OpNegate,
	"Distinct":      ast.OpDistinct,
	"Flatten":       ast.OpFlatten,
	"SingletonFrom": ast.OpSingletonFrom,
	"PointFrom":     ast.OpPointFrom,
	"Start":         ast.OpStartOf,
	"End":           ast.OpEndOf,
	"Width":         ast.OpWidthOf,
	"Successor":     ast.OpSuccessorOf,
	"Predecessor":   ast.OpPredecessorOf,
}

func importUnary(node *ExpressionNode) ast.Expression {
	op, ok := elmUnaryOps[node.Type]
	if !ok {
		return &ast.Literal{ValueType: ast.LiteralNull}
	}
	return &ast.UnaryExpression{
		Operator: op,
		Operand:  importSingleOperand(node),
	}
}

func importCase(node *ExpressionNode) *ast.CaseExpression {
	c := &ast.CaseExpression{
		Comparand: ImportExpression(node.Comparand),
		Else:      ImportExpression(node.Else),
	}
	for _, item := range node.CaseItem {
		c.Items = append(c.Items, &ast.CaseItem{
			When: ImportExpression(item.When),
			Then: ImportExpression(item.Then),
		})
	}
	return c
}

var elmTimingKinds = map[string]struct {
	kind     ast.TimingKind
	properly bool
	before   bool
	after    bool
}{
	"SameAs":           {kind: ast.TimingSameAs},
	"Includes":         {kind: ast.TimingIncludes},
	"IncludedIn":       {kind: ast.TimingIncludedIn},
	"Before":           {kind: ast.TimingBeforeOrAfter, before: true},
	"After":            {kind: ast.TimingBeforeOrAfter, after: true},
	"SameOrBefore":     {kind: ast.TimingBeforeOrAfter, before: true, properly: true},
	"SameOrAfter":      {kind: ast.TimingBeforeOrAfter, after: true, properly: true},
	"Meets":            {kind: ast.TimingMeets},
	"MeetsBefore":      {kind: ast.TimingMeets, before: true},
	"MeetsAfter":       {kind: ast.TimingMeets, after: true},
	"Overlaps":         {kind: ast.TimingOverlaps},
	"OverlapsBefore":   {kind: ast.TimingOverlaps, before: true},
	"OverlapsAfter":    {kind: ast.TimingOverlaps, after: true},
	"Starts":           {kind: ast.TimingStarts},
	"Ends":             {kind: ast.TimingEnds},
	"ProperIncludes":   {kind: ast.TimingIncludes, properly: true},
	"ProperIncludedIn": {kind: ast.TimingIncludedIn, properly: true},
}

func importTiming(node *ExpressionNode) ast.Expression {
	info, ok := elmTimingKinds[node.Type]
	if !ok {
		return &ast.Literal{ValueType: ast.LiteralNull}
	}
	ops := importOperandSlice(node)
	var left, right ast.Expression
	if len(ops) >= 2 {
		left = ops[0]
		right = ops[1]
	}
	return &ast.TimingExpression{
		Left:  left,
		Right: right,
		Operator: ast.TimingOp{
			Kind:      info.kind,
			Precision: node.Precision,
			Properly:  info.properly,
			Before:    info.before,
			After:     info.after,
		},
	}
}

func importInterval(node *ExpressionNode) *ast.IntervalExpression {
	ie := &ast.IntervalExpression{
		Low:  ImportExpression(node.Low),
		High: ImportExpression(node.High),
	}
	if node.LowClosed != nil {
		ie.LowClosed = *node.LowClosed
	} else {
		ie.LowClosed = true
	}
	if node.HighClosed != nil {
		ie.HighClosed = *node.HighClosed
	} else {
		ie.HighClosed = true
	}
	return ie
}

func importList(node *ExpressionNode) *ast.ListExpression {
	le := &ast.ListExpression{}
	for _, e := range node.Element {
		le.Elements = append(le.Elements, ImportExpression(e))
	}
	return le
}

func importTupleExpr(node *ExpressionNode) *ast.TupleExpression {
	te := &ast.TupleExpression{}
	for _, e := range node.Element {
		te.Elements = append(te.Elements, &ast.TupleElement{
			Name:       e.Name,
			Expression: ImportExpression(e.Source),
		})
	}
	return te
}

func importInstanceExpr(node *ExpressionNode) *ast.InstanceExpression {
	ie := &ast.InstanceExpression{
		Type: parseDataType(node.ClassType),
	}
	for _, e := range node.Element {
		ie.Elements = append(ie.Elements, &ast.TupleElement{
			Name:       e.Name,
			Expression: ImportExpression(e.Source),
		})
	}
	return ie
}

func importCodeExpr(node *ExpressionNode) *ast.CodeExpression {
	ce := &ast.CodeExpression{
		Code:    node.CodeValue,
		Display: node.Display,
	}
	if node.System != nil {
		ce.System = node.System.Name
	}
	return ce
}

func importConceptExpr(node *ExpressionNode) *ast.ConceptExpression {
	ce := &ast.ConceptExpression{Display: node.Display}
	for _, e := range node.Element {
		if e.Type == "Code" {
			code := &ast.CodeExpression{
				Code:    e.CodeValue,
				Display: e.Display,
			}
			if e.System != nil {
				code.System = e.System.Name
			}
			ce.Codes = append(ce.Codes, code)
		}
	}
	return ce
}

func importFunctionCall(node *ExpressionNode) *ast.FunctionCall {
	fc := &ast.FunctionCall{
		Name:    node.Name,
		Library: node.LibraryName,
	}
	ops := importOperandSlice(node)
	fc.Operands = ops
	return fc
}

func importQuery(node *ExpressionNode) *ast.Query {
	q := &ast.Query{}

	for _, s := range node.SourceClause {
		q.Sources = append(q.Sources, &ast.AliasedSource{
			Source: ImportExpression(s.Expression),
			Alias:  s.Alias,
		})
	}

	for _, l := range node.Let {
		q.Let = append(q.Let, &ast.LetClause{
			Identifier: l.Identifier,
			Expression: ImportExpression(l.Expression),
		})
	}

	for _, r := range node.Relationship {
		source := &ast.AliasedSource{
			Source: ImportExpression(r.Expression),
			Alias:  r.Alias,
		}
		if r.Type == "Without" {
			q.Without = append(q.Without, &ast.WithoutClause{
				Source:    source,
				Condition: ImportExpression(r.SuchThat),
			})
		} else {
			q.With = append(q.With, &ast.WithClause{
				Source:    source,
				Condition: ImportExpression(r.SuchThat),
			})
		}
	}

	if node.Where != nil {
		q.Where = ImportExpression(node.Where)
	}

	if node.Return != nil {
		q.Return = &ast.ReturnClause{
			Expression: ImportExpression(node.Return.Expression),
			Distinct:   node.Return.Distinct,
		}
	}

	if node.Sort != nil {
		sc := &ast.SortClause{}
		for _, item := range node.Sort.By {
			dir := ast.SortAsc
			if item.Direction == "desc" {
				dir = ast.SortDesc
			}
			sc.ByItems = append(sc.ByItems, &ast.SortByItem{
				Direction:  dir,
				Expression: ImportExpression(item.Expression),
			})
		}
		q.Sort = sc
	}

	if node.Aggregate != nil {
		q.Aggregate = &ast.AggregateClause{
			Identifier: node.Aggregate.Identifier,
			Distinct:   node.Aggregate.Distinct,
			Starting:   ImportExpression(node.Aggregate.Starting),
			Expression: ImportExpression(node.Aggregate.Expression),
		}
	}

	return q
}

// importSingleOperand extracts a single operand from the polymorphic Operand field.
func importSingleOperand(node *ExpressionNode) ast.Expression {
	if node.Operand == nil {
		return nil
	}
	switch v := node.Operand.(type) {
	case *ExpressionNode:
		return ImportExpression(v)
	case []*ExpressionNode:
		if len(v) > 0 {
			return ImportExpression(v[0])
		}
	}
	return nil
}

// importOperandSlice extracts the operand list from the polymorphic Operand field.
func importOperandSlice(node *ExpressionNode) []ast.Expression {
	if node.Operand == nil {
		return nil
	}
	switch v := node.Operand.(type) {
	case *ExpressionNode:
		return []ast.Expression{ImportExpression(v)}
	case []*ExpressionNode:
		out := make([]ast.Expression, len(v))
		for i, op := range v {
			out[i] = ImportExpression(op)
		}
		return out
	}
	return nil
}

func importTypeExtentTarget(node *ExpressionNode) *ast.NamedType {
	// The operand of Minimum/Maximum might carry the type info
	if node.Operand != nil {
		if en, ok := node.Operand.(*ExpressionNode); ok && en.Name != "" {
			return &ast.NamedType{Namespace: en.LibraryName, Name: en.Name}
		}
	}
	return &ast.NamedType{Name: "Unknown"}
}
