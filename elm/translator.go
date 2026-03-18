package elm

import (
	"fmt"

	"github.com/gofhir/cql/ast"
)

// Translate converts a go-cql AST Library into an ELM Library for JSON serialization.
func Translate(lib *ast.Library) *Library {
	if lib == nil {
		return nil
	}

	out := &Library{
		SchemaIdentifier: &VersionedIdentifier{
			ID:      "urn:hl7-org:elm",
			Version: "r1",
		},
	}

	if lib.Identifier != nil {
		out.Identifier = &VersionedIdentifier{
			ID:      lib.Identifier.Name,
			Version: lib.Identifier.Version,
		}
	}

	out.Usings = translateUsings(lib.Usings)
	out.Includes = translateIncludes(lib.Includes)
	out.Parameters = translateParameters(lib.Parameters)
	out.CodeSystems = translateCodeSystems(lib.CodeSystems)
	out.ValueSets = translateValueSets(lib.ValueSets)
	out.Codes = translateCodes(lib.Codes)
	out.Concepts = translateConcepts(lib.Concepts)
	out.Contexts = translateContexts(lib.Contexts)
	out.Statements = translateStatements(lib.Statements, lib.Functions)

	return out
}

// ---------------------------------------------------------------------------
// Definitions
// ---------------------------------------------------------------------------

func translateUsings(defs []*ast.UsingDef) *UsingDefs {
	if len(defs) == 0 {
		return nil
	}
	out := &UsingDefs{}
	for _, d := range defs {
		u := &UsingDef{
			LocalIdentifier: d.Name,
			Version:         d.Version,
		}
		if d.Alias != "" {
			u.LocalIdentifier = d.Alias
		}
		out.Def = append(out.Def, u)
	}
	return out
}

func translateIncludes(defs []*ast.IncludeDef) *IncludeDefs {
	if len(defs) == 0 {
		return nil
	}
	out := &IncludeDefs{}
	for _, d := range defs {
		out.Def = append(out.Def, &IncludeDef{
			LocalIdentifier: d.Alias,
			Path:            d.Name,
			Version:         d.Version,
		})
	}
	return out
}

func translateParameters(defs []*ast.ParameterDef) *ParameterDefs {
	if len(defs) == 0 {
		return nil
	}
	out := &ParameterDefs{}
	for _, d := range defs {
		p := &ParameterDef{
			Name:        d.Name,
			AccessLevel: translateAccessLevel(d.AccessLevel),
		}
		if d.Type != nil {
			p.ParameterType = translateTypeSpecifier(d.Type)
		}
		if d.Default != nil {
			p.Default = TranslateExpression(d.Default)
		}
		out.Def = append(out.Def, p)
	}
	return out
}

func translateCodeSystems(defs []*ast.CodeSystemDef) *CodeSystemDefs {
	if len(defs) == 0 {
		return nil
	}
	out := &CodeSystemDefs{}
	for _, d := range defs {
		out.Def = append(out.Def, &CodeSystemDef{
			Name:        d.Name,
			ID:          d.ID,
			Version:     d.Version,
			AccessLevel: translateAccessLevel(d.AccessLevel),
		})
	}
	return out
}

func translateValueSets(defs []*ast.ValueSetDef) *ValueSetDefs {
	if len(defs) == 0 {
		return nil
	}
	out := &ValueSetDefs{}
	for _, d := range defs {
		out.Def = append(out.Def, &ValueSetDef{
			Name:        d.Name,
			ID:          d.ID,
			Version:     d.Version,
			AccessLevel: translateAccessLevel(d.AccessLevel),
			CodeSystem:  d.CodeSystems,
		})
	}
	return out
}

func translateCodes(defs []*ast.CodeDef) *CodeDefs {
	if len(defs) == 0 {
		return nil
	}
	out := &CodeDefs{}
	for _, d := range defs {
		cd := &CodeDef{
			Name:        d.Name,
			ID:          d.ID,
			Display:     d.Display,
			AccessLevel: translateAccessLevel(d.AccessLevel),
		}
		if d.System != "" {
			cd.CodeSystem = &CodeSystemRef{Name: d.System}
		}
		out.Def = append(out.Def, cd)
	}
	return out
}

func translateConcepts(defs []*ast.ConceptDef) *ConceptDefs {
	if len(defs) == 0 {
		return nil
	}
	out := &ConceptDefs{}
	for _, d := range defs {
		cd := &ConceptDef{
			Name:        d.Name,
			Display:     d.Display,
			AccessLevel: translateAccessLevel(d.AccessLevel),
		}
		for _, c := range d.Codes {
			cd.Code = append(cd.Code, &CodeRef{Name: c})
		}
		out.Def = append(out.Def, cd)
	}
	return out
}

func translateContexts(defs []*ast.ContextDef) *ContextDefs {
	if len(defs) == 0 {
		return nil
	}
	out := &ContextDefs{}
	for _, d := range defs {
		out.Def = append(out.Def, &ContextDef{Name: d.Name})
	}
	return out
}

func translateStatements(exprs []*ast.ExpressionDef, funcs []*ast.FunctionDef) *Statements {
	if len(exprs) == 0 && len(funcs) == 0 {
		return nil
	}
	out := &Statements{}
	for _, e := range exprs {
		out.Def = append(out.Def, &ExpressionDef{
			Name:        e.Name,
			Context:     e.Context,
			AccessLevel: translateAccessLevel(e.AccessLevel),
			Expression:  TranslateExpression(e.Expression),
		})
	}
	// Functions are serialized as FunctionDef in ELM but stored alongside ExpressionDefs
	for _, f := range funcs {
		fd := &ExpressionDef{
			Name:        f.Name,
			AccessLevel: translateAccessLevel(f.AccessLevel),
			Expression:  TranslateExpression(f.Body),
		}
		out.Def = append(out.Def, fd)
	}
	return out
}

// ---------------------------------------------------------------------------
// Type specifiers
// ---------------------------------------------------------------------------

func translateTypeSpecifier(ts ast.TypeSpecifier) *TypeSpecifier {
	if ts == nil {
		return nil
	}
	switch t := ts.(type) {
	case *ast.NamedType:
		return &TypeSpecifier{
			Type:      "NamedTypeSpecifier",
			Namespace: t.Namespace,
			Name:      t.Name,
		}
	case *ast.ListType:
		return &TypeSpecifier{
			Type:        "ListTypeSpecifier",
			ElementType: translateTypeSpecifier(t.ElementType),
		}
	case *ast.IntervalType:
		return &TypeSpecifier{
			Type:      "IntervalTypeSpecifier",
			PointType: translateTypeSpecifier(t.PointType),
		}
	case *ast.TupleType:
		spec := &TypeSpecifier{Type: "TupleTypeSpecifier"}
		for _, e := range t.Elements {
			spec.Element = append(spec.Element, &TupleElement{
				Name:        e.Name,
				ElementType: translateTypeSpecifier(e.Type),
			})
		}
		return spec
	case *ast.ChoiceType:
		spec := &TypeSpecifier{Type: "ChoiceTypeSpecifier"}
		for _, c := range t.Types {
			spec.Choice = append(spec.Choice, translateTypeSpecifier(c))
		}
		return spec
	default:
		return nil
	}
}

func translateAccessLevel(al ast.AccessLevel) string {
	switch al {
	case ast.AccessPrivate:
		return "Private"
	default:
		return "Public"
	}
}

// ---------------------------------------------------------------------------
// Expression translation (AST → ELM ExpressionNode)
// ---------------------------------------------------------------------------

// TranslateExpression converts a single AST expression to an ELM ExpressionNode.
func TranslateExpression(expr ast.Expression) *ExpressionNode {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *ast.Literal:
		return translateLiteral(e)
	case *ast.IdentifierRef:
		return translateIdentifierRef(e)
	case *ast.Retrieve:
		return translateRetrieve(e)
	case *ast.Query:
		return translateQuery(e)
	case *ast.BinaryExpression:
		return translateBinary(e)
	case *ast.UnaryExpression:
		return translateUnary(e)
	case *ast.IsExpression:
		return &ExpressionNode{
			Type:            "Is",
			Operand:         TranslateExpression(e.Operand),
			IsTypeSpecifier: translateTypeSpecifier(e.Type),
		}
	case *ast.AsExpression:
		return &ExpressionNode{
			Type:            "As",
			Operand:         TranslateExpression(e.Operand),
			AsTypeSpecifier: translateTypeSpecifier(e.Type),
		}
	case *ast.CastExpression:
		return &ExpressionNode{
			Type:            "As",
			Operand:         TranslateExpression(e.Operand),
			AsTypeSpecifier: translateTypeSpecifier(e.Type),
			Strict:          true,
		}
	case *ast.ConvertExpression:
		node := &ExpressionNode{
			Type:    "Convert",
			Operand: TranslateExpression(e.Operand),
		}
		if e.ToType != nil {
			node.ToType = translateTypeSpecifier(e.ToType)
		}
		return node
	case *ast.BooleanTestExpression:
		return translateBooleanTest(e)
	case *ast.IfThenElse:
		return &ExpressionNode{
			Type:      "If",
			Condition: TranslateExpression(e.Condition),
			Then:      TranslateExpression(e.Then),
			Else:      TranslateExpression(e.Else),
		}
	case *ast.CaseExpression:
		return translateCase(e)
	case *ast.BetweenExpression:
		return translateBetween(e)
	case *ast.DurationBetween:
		return &ExpressionNode{
			Type:      "DurationBetween",
			Precision: e.Precision,
			Operand:   []*ExpressionNode{TranslateExpression(e.Low), TranslateExpression(e.High)},
		}
	case *ast.DifferenceBetween:
		return &ExpressionNode{
			Type:      "DifferenceBetween",
			Precision: e.Precision,
			Operand:   []*ExpressionNode{TranslateExpression(e.Low), TranslateExpression(e.High)},
		}
	case *ast.DateTimeComponentFrom:
		return &ExpressionNode{
			Type:      "DateTimeComponentFrom",
			Precision: e.Component,
			Operand:   TranslateExpression(e.Operand),
		}
	case *ast.DurationOf:
		return &ExpressionNode{
			Type:      "DurationOf",
			Precision: e.Precision,
			Operand:   TranslateExpression(e.Operand),
		}
	case *ast.DifferenceOf:
		return &ExpressionNode{
			Type:      "DifferenceOf",
			Precision: e.Precision,
			Operand:   TranslateExpression(e.Operand),
		}
	case *ast.TypeExtent:
		return &ExpressionNode{
			Type:    capitalizeFirst(e.Extent),
			Operand: &ExpressionNode{Type: "NamedTypeSpecifier", Name: e.Type.Name, LibraryName: e.Type.Namespace},
		}
	case *ast.TimingExpression:
		return translateTiming(e)
	case *ast.MembershipExpression:
		return translateMembership(e)
	case *ast.MemberAccess:
		return &ExpressionNode{
			Type:   "Property",
			Path:   e.Member,
			Source: TranslateExpression(e.Source),
		}
	case *ast.FunctionCall:
		return translateFunctionCall(e)
	case *ast.IndexAccess:
		return &ExpressionNode{
			Type:    "Indexer",
			Operand: []*ExpressionNode{TranslateExpression(e.Source), TranslateExpression(e.Index)},
		}
	case *ast.IntervalExpression:
		lc := true
		hc := true
		if !e.LowClosed {
			lc = false
		}
		if !e.HighClosed {
			hc = false
		}
		return &ExpressionNode{
			Type:       "Interval",
			Low:        TranslateExpression(e.Low),
			High:       TranslateExpression(e.High),
			LowClosed:  &lc,
			HighClosed: &hc,
		}
	case *ast.TupleExpression:
		return translateTuple(e)
	case *ast.InstanceExpression:
		return translateInstance(e)
	case *ast.ListExpression:
		return translateList(e)
	case *ast.CodeExpression:
		return &ExpressionNode{
			Type:      "Code",
			CodeValue: e.Code,
			System:    &ExpressionNode{Type: "CodeSystemRef", Name: e.System},
			Display:   e.Display,
		}
	case *ast.ConceptExpression:
		return translateConcept(e)
	case *ast.ExternalConstant:
		return &ExpressionNode{
			Type: "ParameterRef",
			Name: e.Name,
		}
	case *ast.ThisExpression:
		return &ExpressionNode{Type: "This"}
	case *ast.IndexExpression:
		return &ExpressionNode{Type: "IterationIndex"}
	case *ast.TotalExpression:
		return &ExpressionNode{Type: "Total"}
	case *ast.SetAggregateExpression:
		return &ExpressionNode{
			Type:    capitalizeFirst(e.Kind),
			Operand: TranslateExpression(e.Operand),
			Per:     TranslateExpression(e.Per),
		}
	default:
		return &ExpressionNode{Type: "Unknown"}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func translateLiteral(l *ast.Literal) *ExpressionNode {
	node := &ExpressionNode{
		Type:  "Literal",
		Value: l.Value,
	}
	switch l.ValueType {
	case ast.LiteralNull:
		node.Type = "Null"
		return node
	case ast.LiteralBoolean:
		node.ValueType = "{urn:hl7-org:elm-types:r1}Boolean"
	case ast.LiteralString:
		node.ValueType = "{urn:hl7-org:elm-types:r1}String"
	case ast.LiteralInteger:
		node.ValueType = "{urn:hl7-org:elm-types:r1}Integer"
	case ast.LiteralLong:
		node.ValueType = "{urn:hl7-org:elm-types:r1}Long"
	case ast.LiteralDecimal:
		node.ValueType = "{urn:hl7-org:elm-types:r1}Decimal"
	case ast.LiteralDate:
		node.ValueType = "{urn:hl7-org:elm-types:r1}Date"
	case ast.LiteralDateTime:
		node.ValueType = "{urn:hl7-org:elm-types:r1}DateTime"
	case ast.LiteralTime:
		node.ValueType = "{urn:hl7-org:elm-types:r1}Time"
	case ast.LiteralQuantity:
		node.ValueType = "{urn:hl7-org:elm-types:r1}Quantity"
	case ast.LiteralRatio:
		node.ValueType = "{urn:hl7-org:elm-types:r1}Ratio"
	}
	return node
}

func translateIdentifierRef(ref *ast.IdentifierRef) *ExpressionNode {
	return &ExpressionNode{
		Type:        "ExpressionRef",
		Name:        ref.Name,
		LibraryName: ref.Library,
	}
}

func translateRetrieve(r *ast.Retrieve) *ExpressionNode {
	node := &ExpressionNode{
		Type:           "Retrieve",
		CodeProperty:   r.CodePath,
		CodeComparator: r.CodeComparator,
		DateProperty:   r.DatePath,
	}
	if r.ResourceType != nil {
		if r.ResourceType.Namespace != "" {
			node.DataType = fmt.Sprintf("{%s}%s", fhirModelURI(r.ResourceType.Namespace), r.ResourceType.Name)
		} else {
			node.DataType = r.ResourceType.Name
		}
	}
	if r.Codes != nil {
		node.Codes = TranslateExpression(r.Codes)
	}
	if r.DateRange != nil {
		node.DateRange = TranslateExpression(r.DateRange)
	}
	return node
}

func fhirModelURI(ns string) string {
	if ns == "FHIR" {
		return "http://hl7.org/fhir"
	}
	return ns
}

var binaryOpNames = map[ast.BinaryOp]string{
	ast.OpAdd:            "Add",
	ast.OpSubtract:       "Subtract",
	ast.OpMultiply:       "Multiply",
	ast.OpDivide:         "Divide",
	ast.OpDiv:            "TruncatedDivide",
	ast.OpMod:            "Modulo",
	ast.OpPower:          "Power",
	ast.OpConcatenate:    "Concatenate",
	ast.OpEqual:          "Equal",
	ast.OpNotEqual:       "NotEqual",
	ast.OpEquivalent:     "Equivalent",
	ast.OpNotEquivalent:  "NotEquivalent",
	ast.OpLess:           "Less",
	ast.OpLessOrEqual:    "LessOrEqual",
	ast.OpGreater:        "Greater",
	ast.OpGreaterOrEqual: "GreaterOrEqual",
	ast.OpAnd:            "And",
	ast.OpOr:             "Or",
	ast.OpXor:            "Xor",
	ast.OpImplies:        "Implies",
	ast.OpUnion:          "Union",
	ast.OpIntersect:      "Intersect",
	ast.OpExcept:         "Except",
	ast.OpIn:             "In",
	ast.OpContains:       "Contains",
}

func translateBinary(b *ast.BinaryExpression) *ExpressionNode {
	name := binaryOpNames[b.Operator]
	if name == "" {
		name = "Unknown"
	}
	return &ExpressionNode{
		Type:    name,
		Operand: []*ExpressionNode{TranslateExpression(b.Left), TranslateExpression(b.Right)},
	}
}

var unaryOpNames = map[ast.UnaryOp]string{
	ast.OpNot:           "Not",
	ast.OpExists:        "Exists",
	ast.OpPositive:      "Abs",
	ast.OpNegate:        "Negate",
	ast.OpDistinct:      "Distinct",
	ast.OpFlatten:       "Flatten",
	ast.OpSingletonFrom: "SingletonFrom",
	ast.OpPointFrom:     "PointFrom",
	ast.OpStartOf:       "Start",
	ast.OpEndOf:         "End",
	ast.OpWidthOf:       "Width",
	ast.OpSuccessorOf:   "Successor",
	ast.OpPredecessorOf: "Predecessor",
}

func translateUnary(u *ast.UnaryExpression) *ExpressionNode {
	name := unaryOpNames[u.Operator]
	if name == "" {
		name = "Unknown"
	}
	return &ExpressionNode{
		Type:    name,
		Operand: TranslateExpression(u.Operand),
	}
}

func translateBooleanTest(bt *ast.BooleanTestExpression) *ExpressionNode {
	switch bt.TestValue {
	case "null":
		if bt.Not {
			return &ExpressionNode{Type: "Not", Operand: &ExpressionNode{
				Type: "IsNull", Operand: TranslateExpression(bt.Operand),
			}}
		}
		return &ExpressionNode{Type: "IsNull", Operand: TranslateExpression(bt.Operand)}
	case "true":
		if bt.Not {
			return &ExpressionNode{Type: "IsFalse", Operand: TranslateExpression(bt.Operand)}
		}
		return &ExpressionNode{Type: "IsTrue", Operand: TranslateExpression(bt.Operand)}
	case "false":
		if bt.Not {
			return &ExpressionNode{Type: "IsTrue", Operand: TranslateExpression(bt.Operand)}
		}
		return &ExpressionNode{Type: "IsFalse", Operand: TranslateExpression(bt.Operand)}
	default:
		return &ExpressionNode{Type: "Unknown"}
	}
}

func translateCase(c *ast.CaseExpression) *ExpressionNode {
	node := &ExpressionNode{
		Type:      "Case",
		Comparand: TranslateExpression(c.Comparand),
		Else:      TranslateExpression(c.Else),
	}
	for _, item := range c.Items {
		node.CaseItem = append(node.CaseItem, &CaseItem{
			When: TranslateExpression(item.When),
			Then: TranslateExpression(item.Then),
		})
	}
	return node
}

func translateBetween(b *ast.BetweenExpression) *ExpressionNode {
	typeName := "IncludedIn"
	if b.Properly {
		typeName = "ProperIncludedIn"
	}
	// Between is desugared to: operand >= low and operand <= high
	// In ELM, this maps to IncludedIn with an Interval
	return &ExpressionNode{
		Type: typeName,
		Operand: []*ExpressionNode{
			TranslateExpression(b.Operand),
			{
				Type: "Interval",
				Low:  TranslateExpression(b.Low),
				High: TranslateExpression(b.High),
				LowClosed:  boolPtr(true),
				HighClosed: boolPtr(true),
			},
		},
	}
}

var timingKindNames = map[ast.TimingKind]string{
	ast.TimingSameAs:        "SameAs",
	ast.TimingIncludes:      "Includes",
	ast.TimingIncludedIn:    "IncludedIn",
	ast.TimingDuring:        "IncludedIn",
	ast.TimingBeforeOrAfter: "Before",
	ast.TimingWithin:        "In",
	ast.TimingMeets:         "Meets",
	ast.TimingOverlaps:      "Overlaps",
	ast.TimingStarts:        "Starts",
	ast.TimingEnds:          "Ends",
}

func translateTiming(t *ast.TimingExpression) *ExpressionNode {
	name := timingKindNames[t.Operator.Kind]
	if name == "" {
		name = "Unknown"
	}

	if t.Operator.Properly {
		name = "Proper" + name
	}

	// Handle Before/After direction
	if t.Operator.Kind == ast.TimingBeforeOrAfter {
		if t.Operator.After {
			name = "After"
		} else {
			name = "Before"
		}
		if t.Operator.Properly {
			name = "SameOrBefore"
			if t.Operator.After {
				name = "SameOrAfter"
			}
		}
	}

	node := &ExpressionNode{
		Type:    name,
		Operand: []*ExpressionNode{TranslateExpression(t.Left), TranslateExpression(t.Right)},
	}
	if t.Operator.Precision != "" {
		node.Precision = t.Operator.Precision
	}
	return node
}

func translateMembership(m *ast.MembershipExpression) *ExpressionNode {
	typeName := "In"
	if m.Operator == "contains" {
		typeName = "Contains"
	}
	node := &ExpressionNode{
		Type:    typeName,
		Operand: []*ExpressionNode{TranslateExpression(m.Left), TranslateExpression(m.Right)},
	}
	if m.Precision != "" {
		node.Precision = m.Precision
	}
	return node
}

func translateFunctionCall(fc *ast.FunctionCall) *ExpressionNode {
	node := &ExpressionNode{
		Type:        "FunctionRef",
		Name:        fc.Name,
		LibraryName: fc.Library,
	}
	if len(fc.Operands) > 0 {
		ops := make([]*ExpressionNode, len(fc.Operands))
		for i, op := range fc.Operands {
			ops[i] = TranslateExpression(op)
		}
		node.Operand = ops
	}
	return node
}

func translateTuple(t *ast.TupleExpression) *ExpressionNode {
	node := &ExpressionNode{Type: "Tuple"}
	elems := make([]*ExpressionNode, len(t.Elements))
	for i, e := range t.Elements {
		elems[i] = &ExpressionNode{
			Type:   "TupleElement",
			Name:   e.Name,
			Source: TranslateExpression(e.Expression),
		}
	}
	node.Element = elems
	return node
}

func translateInstance(inst *ast.InstanceExpression) *ExpressionNode {
	node := &ExpressionNode{
		Type: "Instance",
	}
	if inst.Type != nil {
		if inst.Type.Namespace != "" {
			node.ClassType = fmt.Sprintf("{%s}%s", fhirModelURI(inst.Type.Namespace), inst.Type.Name)
		} else {
			node.ClassType = inst.Type.Name
		}
	}
	elems := make([]*ExpressionNode, len(inst.Elements))
	for i, e := range inst.Elements {
		elems[i] = &ExpressionNode{
			Type:   "InstanceElement",
			Name:   e.Name,
			Source: TranslateExpression(e.Expression),
		}
	}
	node.Element = elems
	return node
}

func translateList(l *ast.ListExpression) *ExpressionNode {
	node := &ExpressionNode{Type: "List"}
	if len(l.Elements) > 0 {
		elems := make([]*ExpressionNode, len(l.Elements))
		for i, e := range l.Elements {
			elems[i] = TranslateExpression(e)
		}
		node.Element = elems
	}
	return node
}

func translateConcept(c *ast.ConceptExpression) *ExpressionNode {
	node := &ExpressionNode{
		Type:    "Concept",
		Display: c.Display,
	}
	for _, code := range c.Codes {
		node.Element = append(node.Element, &ExpressionNode{
			Type:      "Code",
			CodeValue: code.Code,
			System:    &ExpressionNode{Type: "CodeSystemRef", Name: code.System},
			Display:   code.Display,
		})
	}
	return node
}

func translateQuery(q *ast.Query) *ExpressionNode {
	node := &ExpressionNode{Type: "Query"}

	for _, s := range q.Sources {
		node.SourceClause = append(node.SourceClause, &AliasedQuerySource{
			Expression: TranslateExpression(s.Source),
			Alias:      s.Alias,
		})
	}

	for _, l := range q.Let {
		node.Let = append(node.Let, &LetClause{
			Identifier: l.Identifier,
			Expression: TranslateExpression(l.Expression),
		})
	}

	for _, w := range q.With {
		node.Relationship = append(node.Relationship, &RelationshipClause{
			Type:       "With",
			Expression: TranslateExpression(w.Source.Source),
			Alias:      w.Source.Alias,
			SuchThat:   TranslateExpression(w.Condition),
		})
	}
	for _, w := range q.Without {
		node.Relationship = append(node.Relationship, &RelationshipClause{
			Type:       "Without",
			Expression: TranslateExpression(w.Source.Source),
			Alias:      w.Source.Alias,
			SuchThat:   TranslateExpression(w.Condition),
		})
	}

	if q.Where != nil {
		node.Where = TranslateExpression(q.Where)
	}

	if q.Return != nil {
		node.Return = &ReturnClause{
			Expression: TranslateExpression(q.Return.Expression),
			Distinct:   q.Return.Distinct,
		}
	}

	if q.Sort != nil {
		sc := &SortClause{}
		for _, item := range q.Sort.ByItems {
			dir := "asc"
			if item.Direction == ast.SortDesc {
				dir = "desc"
			}
			sc.By = append(sc.By, &SortByItem{
				Direction:  dir,
				Type:       "ByExpression",
				Expression: TranslateExpression(item.Expression),
			})
		}
		node.Sort = sc
	}

	if q.Aggregate != nil {
		node.Aggregate = &AggregateClause{
			Identifier: q.Aggregate.Identifier,
			Distinct:   q.Aggregate.Distinct,
			Starting:   TranslateExpression(q.Aggregate.Starting),
			Expression: TranslateExpression(q.Aggregate.Expression),
		}
	}

	return node
}

func boolPtr(b bool) *bool { return &b }

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	b := []byte(s)
	if b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 32
	}
	return string(b)
}
