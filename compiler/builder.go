package compiler

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/parser/grammar"
)

// builder converts an ANTLR parse tree into a CQL AST.
type builder struct {
	grammar.BasecqlVisitor
	err            error
	currentContext string // tracks the current 'context' (e.g. "Patient")
}

func newBuilder() *builder {
	return &builder{}
}

// Visit dispatches to the appropriate visitor method.
func (b *builder) Visit(tree antlr.ParseTree) interface{} {
	if b.err != nil {
		return nil
	}
	if tree == nil {
		return nil
	}
	return tree.Accept(b)
}

func (b *builder) setError(msg string, args ...interface{}) {
	if b.err == nil {
		b.err = fmt.Errorf(msg, args...)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (b *builder) visitExpression(ctx grammar.IExpressionContext) ast.Expression {
	if ctx == nil {
		return nil
	}
	result := b.Visit(ctx)
	if result == nil {
		return nil
	}
	expr, ok := result.(ast.Expression)
	if !ok {
		b.setError("expected Expression, got %T", result)
		return nil
	}
	return expr
}

func (b *builder) visitExpressionTerm(ctx grammar.IExpressionTermContext) ast.Expression {
	if ctx == nil {
		return nil
	}
	result := b.Visit(ctx)
	if result == nil {
		return nil
	}
	expr, ok := result.(ast.Expression)
	if !ok {
		b.setError("expected Expression from expressionTerm, got %T", result)
		return nil
	}
	return expr
}

func (b *builder) visitTypeSpecifier(ctx grammar.ITypeSpecifierContext) ast.TypeSpecifier {
	if ctx == nil {
		return nil
	}
	result := b.Visit(ctx)
	if result == nil {
		return nil
	}
	ts, ok := result.(ast.TypeSpecifier)
	if !ok {
		b.setError("expected TypeSpecifier, got %T", result)
		return nil
	}
	return ts
}

func unquoteString(s string) string {
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			s = s[1 : len(s)-1]
		}
	}
	// handle basic escape sequences
	s = strings.ReplaceAll(s, "\\'", "'")
	s = strings.ReplaceAll(s, "\\\"", "\"")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\r", "\r")
	s = strings.ReplaceAll(s, "\\t", "\t")
	return s
}

func identifierText(ctx grammar.IIdentifierContext) string {
	if ctx == nil {
		return ""
	}
	text := ctx.GetText()
	if strings.HasPrefix(text, "`") && strings.HasSuffix(text, "`") {
		return text[1 : len(text)-1]
	}
	if strings.HasPrefix(text, "\"") && strings.HasSuffix(text, "\"") {
		return text[1 : len(text)-1]
	}
	return text
}

func referentialIdentifierText(ctx grammar.IReferentialIdentifierContext) string {
	if ctx == nil {
		return ""
	}
	if id := ctx.Identifier(); id != nil {
		return identifierText(id)
	}
	if kw := ctx.KeywordIdentifier(); kw != nil {
		return kw.GetText()
	}
	return ctx.GetText()
}

func (b *builder) visitAccessModifier(ctx grammar.IAccessModifierContext) ast.AccessLevel {
	if ctx == nil {
		return ast.AccessPublic
	}
	if ctx.GetText() == "private" {
		return ast.AccessPrivate
	}
	return ast.AccessPublic
}

// ---------------------------------------------------------------------------
// Library (top-level)
// ---------------------------------------------------------------------------

func (b *builder) VisitLibrary(ctx *grammar.LibraryContext) interface{} {
	lib := &ast.Library{}

	if ld := ctx.LibraryDefinition(); ld != nil {
		result := b.Visit(ld)
		if id, ok := result.(*ast.LibraryIdentifier); ok {
			lib.Identifier = id
		}
	}

	for _, def := range ctx.AllDefinition() {
		b.visitDefinition(def, lib)
	}

	for _, stmt := range ctx.AllStatement() {
		b.visitStatement(stmt, lib)
	}

	return lib
}

func (b *builder) VisitLibraryDefinition(ctx *grammar.LibraryDefinitionContext) interface{} {
	id := &ast.LibraryIdentifier{}
	if qi := ctx.QualifiedIdentifier(); qi != nil {
		id.Name = qi.GetText()
	}
	if vs := ctx.VersionSpecifier(); vs != nil {
		id.Version = unquoteString(vs.GetText())
	}
	return id
}

// ---------------------------------------------------------------------------
// Definitions
// ---------------------------------------------------------------------------

func (b *builder) visitDefinition(ctx grammar.IDefinitionContext, lib *ast.Library) {
	if ctx == nil {
		return
	}
	if ud := ctx.UsingDefinition(); ud != nil {
		result := b.Visit(ud)
		if u, ok := result.(*ast.UsingDef); ok {
			lib.Usings = append(lib.Usings, u)
		}
	}
	if id := ctx.IncludeDefinition(); id != nil {
		result := b.Visit(id)
		if i, ok := result.(*ast.IncludeDef); ok {
			lib.Includes = append(lib.Includes, i)
		}
	}
	if csd := ctx.CodesystemDefinition(); csd != nil {
		result := b.Visit(csd)
		if cs, ok := result.(*ast.CodeSystemDef); ok {
			lib.CodeSystems = append(lib.CodeSystems, cs)
		}
	}
	if vsd := ctx.ValuesetDefinition(); vsd != nil {
		result := b.Visit(vsd)
		if vs, ok := result.(*ast.ValueSetDef); ok {
			lib.ValueSets = append(lib.ValueSets, vs)
		}
	}
	if cd := ctx.CodeDefinition(); cd != nil {
		result := b.Visit(cd)
		if c, ok := result.(*ast.CodeDef); ok {
			lib.Codes = append(lib.Codes, c)
		}
	}
	if cpd := ctx.ConceptDefinition(); cpd != nil {
		result := b.Visit(cpd)
		if cp, ok := result.(*ast.ConceptDef); ok {
			lib.Concepts = append(lib.Concepts, cp)
		}
	}
	if pd := ctx.ParameterDefinition(); pd != nil {
		result := b.Visit(pd)
		if p, ok := result.(*ast.ParameterDef); ok {
			lib.Parameters = append(lib.Parameters, p)
		}
	}
}

func (b *builder) VisitUsingDefinition(ctx *grammar.UsingDefinitionContext) interface{} {
	u := &ast.UsingDef{}
	if qi := ctx.QualifiedIdentifier(); qi != nil {
		u.Name = qi.GetText()
	}
	if vs := ctx.VersionSpecifier(); vs != nil {
		u.Version = unquoteString(vs.GetText())
	}
	if li := ctx.LocalIdentifier(); li != nil {
		u.Alias = li.GetText()
	}
	return u
}

func (b *builder) VisitIncludeDefinition(ctx *grammar.IncludeDefinitionContext) interface{} {
	i := &ast.IncludeDef{}
	if qi := ctx.QualifiedIdentifier(); qi != nil {
		i.Name = qi.GetText()
	}
	if vs := ctx.VersionSpecifier(); vs != nil {
		i.Version = unquoteString(vs.GetText())
	}
	if li := ctx.LocalIdentifier(); li != nil {
		i.Alias = li.GetText()
	}
	return i
}

func (b *builder) VisitCodesystemDefinition(ctx *grammar.CodesystemDefinitionContext) interface{} {
	cs := &ast.CodeSystemDef{}
	if id := ctx.Identifier(); id != nil {
		cs.Name = identifierText(id)
	}
	if csid := ctx.CodesystemId(); csid != nil {
		cs.ID = unquoteString(csid.GetText())
	}
	if vs := ctx.VersionSpecifier(); vs != nil {
		cs.Version = unquoteString(vs.GetText())
	}
	if am := ctx.AccessModifier(); am != nil {
		cs.AccessLevel = b.visitAccessModifier(am)
	}
	return cs
}

func (b *builder) VisitValuesetDefinition(ctx *grammar.ValuesetDefinitionContext) interface{} {
	vs := &ast.ValueSetDef{}
	if id := ctx.Identifier(); id != nil {
		vs.Name = identifierText(id)
	}
	if vid := ctx.ValuesetId(); vid != nil {
		vs.ID = unquoteString(vid.GetText())
	}
	if ver := ctx.VersionSpecifier(); ver != nil {
		vs.Version = unquoteString(ver.GetText())
	}
	if am := ctx.AccessModifier(); am != nil {
		vs.AccessLevel = b.visitAccessModifier(am)
	}
	if css := ctx.Codesystems(); css != nil {
		for _, csid := range css.AllCodesystemIdentifier() {
			vs.CodeSystems = append(vs.CodeSystems, csid.GetText())
		}
	}
	return vs
}

func (b *builder) VisitCodeDefinition(ctx *grammar.CodeDefinitionContext) interface{} {
	cd := &ast.CodeDef{}
	if id := ctx.Identifier(); id != nil {
		cd.Name = identifierText(id)
	}
	if cid := ctx.CodeId(); cid != nil {
		cd.ID = unquoteString(cid.GetText())
	}
	if csid := ctx.CodesystemIdentifier(); csid != nil {
		cd.System = csid.GetText()
	}
	if dc := ctx.DisplayClause(); dc != nil {
		cd.Display = unquoteString(dc.STRING().GetText())
	}
	if am := ctx.AccessModifier(); am != nil {
		cd.AccessLevel = b.visitAccessModifier(am)
	}
	return cd
}

func (b *builder) VisitConceptDefinition(ctx *grammar.ConceptDefinitionContext) interface{} {
	cp := &ast.ConceptDef{}
	if id := ctx.Identifier(); id != nil {
		cp.Name = identifierText(id)
	}
	for _, cid := range ctx.AllCodeIdentifier() {
		cp.Codes = append(cp.Codes, cid.GetText())
	}
	if dc := ctx.DisplayClause(); dc != nil {
		cp.Display = unquoteString(dc.STRING().GetText())
	}
	if am := ctx.AccessModifier(); am != nil {
		cp.AccessLevel = b.visitAccessModifier(am)
	}
	return cp
}

func (b *builder) VisitParameterDefinition(ctx *grammar.ParameterDefinitionContext) interface{} {
	p := &ast.ParameterDef{}
	if id := ctx.Identifier(); id != nil {
		p.Name = identifierText(id)
	}
	if ts := ctx.TypeSpecifier(); ts != nil {
		p.Type = b.visitTypeSpecifier(ts)
	}
	if expr := ctx.Expression(); expr != nil {
		p.Default = b.visitExpression(expr)
	}
	if am := ctx.AccessModifier(); am != nil {
		p.AccessLevel = b.visitAccessModifier(am)
	}
	return p
}

// ---------------------------------------------------------------------------
// Statements
// ---------------------------------------------------------------------------

func (b *builder) visitStatement(ctx grammar.IStatementContext, lib *ast.Library) {
	if ctx == nil {
		return
	}
	if ed := ctx.ExpressionDefinition(); ed != nil {
		result := b.Visit(ed)
		if e, ok := result.(*ast.ExpressionDef); ok {
			e.Context = b.currentContext
			lib.Statements = append(lib.Statements, e)
		}
	}
	if cd := ctx.ContextDefinition(); cd != nil {
		result := b.Visit(cd)
		if c, ok := result.(*ast.ContextDef); ok {
			b.currentContext = c.Name
			lib.Contexts = append(lib.Contexts, c)
		}
	}
	if fd := ctx.FunctionDefinition(); fd != nil {
		result := b.Visit(fd)
		if f, ok := result.(*ast.FunctionDef); ok {
			lib.Functions = append(lib.Functions, f)
		}
	}
}

func (b *builder) VisitExpressionDefinition(ctx *grammar.ExpressionDefinitionContext) interface{} {
	ed := &ast.ExpressionDef{}
	if id := ctx.Identifier(); id != nil {
		ed.Name = identifierText(id)
	}
	if expr := ctx.Expression(); expr != nil {
		ed.Expression = b.visitExpression(expr)
	}
	if am := ctx.AccessModifier(); am != nil {
		ed.AccessLevel = b.visitAccessModifier(am)
	}
	return ed
}

func (b *builder) VisitContextDefinition(ctx *grammar.ContextDefinitionContext) interface{} {
	cd := &ast.ContextDef{}
	if mi := ctx.ModelIdentifier(); mi != nil {
		cd.Model = mi.GetText()
	}
	if id := ctx.Identifier(); id != nil {
		cd.Name = identifierText(id)
	}
	return cd
}

func (b *builder) VisitFunctionDefinition(ctx *grammar.FunctionDefinitionContext) interface{} {
	fd := &ast.FunctionDef{}
	if id := ctx.IdentifierOrFunctionIdentifier(); id != nil {
		fd.Name = id.GetText()
	}
	for _, od := range ctx.AllOperandDefinition() {
		result := b.Visit(od)
		if o, ok := result.(*ast.OperandDef); ok {
			fd.Operands = append(fd.Operands, o)
		}
	}
	if ts := ctx.TypeSpecifier(); ts != nil {
		fd.ReturnType = b.visitTypeSpecifier(ts)
	}
	if fb := ctx.FunctionBody(); fb != nil {
		fd.Body = b.visitExpression(fb.Expression())
	}
	if ctx.GetText() != "" && strings.Contains(ctx.GetText(), "external") {
		fd.External = true
	}
	if fm := ctx.FluentModifier(); fm != nil {
		fd.Fluent = true
	}
	if am := ctx.AccessModifier(); am != nil {
		fd.AccessLevel = b.visitAccessModifier(am)
	}
	return fd
}

func (b *builder) VisitOperandDefinition(ctx *grammar.OperandDefinitionContext) interface{} {
	od := &ast.OperandDef{}
	if ri := ctx.ReferentialIdentifier(); ri != nil {
		od.Name = referentialIdentifierText(ri)
	}
	if ts := ctx.TypeSpecifier(); ts != nil {
		od.Type = b.visitTypeSpecifier(ts)
	}
	return od
}

// ---------------------------------------------------------------------------
// Type Specifiers
// ---------------------------------------------------------------------------

func (b *builder) VisitTypeSpecifier(ctx *grammar.TypeSpecifierContext) interface{} {
	if n := ctx.NamedTypeSpecifier(); n != nil {
		return b.Visit(n)
	}
	if l := ctx.ListTypeSpecifier(); l != nil {
		return b.Visit(l)
	}
	if i := ctx.IntervalTypeSpecifier(); i != nil {
		return b.Visit(i)
	}
	if t := ctx.TupleTypeSpecifier(); t != nil {
		return b.Visit(t)
	}
	if c := ctx.ChoiceTypeSpecifier(); c != nil {
		return b.Visit(c)
	}
	return nil
}

func (b *builder) VisitNamedTypeSpecifier(ctx *grammar.NamedTypeSpecifierContext) interface{} {
	nt := &ast.NamedType{}
	if rot := ctx.ReferentialOrTypeNameIdentifier(); rot != nil {
		nt.Name = rot.GetText()
	}
	qualifiers := ctx.AllQualifier()
	if len(qualifiers) > 0 {
		parts := make([]string, len(qualifiers))
		for i, q := range qualifiers {
			parts[i] = q.GetText()
		}
		nt.Namespace = strings.Join(parts, ".")
	}
	return nt
}

func (b *builder) VisitListTypeSpecifier(ctx *grammar.ListTypeSpecifierContext) interface{} {
	lt := &ast.ListType{}
	if ts := ctx.TypeSpecifier(); ts != nil {
		lt.ElementType = b.visitTypeSpecifier(ts)
	}
	return lt
}

func (b *builder) VisitIntervalTypeSpecifier(ctx *grammar.IntervalTypeSpecifierContext) interface{} {
	it := &ast.IntervalType{}
	if ts := ctx.TypeSpecifier(); ts != nil {
		it.PointType = b.visitTypeSpecifier(ts)
	}
	return it
}

func (b *builder) VisitTupleTypeSpecifier(ctx *grammar.TupleTypeSpecifierContext) interface{} {
	tt := &ast.TupleType{}
	for _, ted := range ctx.AllTupleElementDefinition() {
		result := b.Visit(ted)
		if e, ok := result.(*ast.TupleElementDef); ok {
			tt.Elements = append(tt.Elements, e)
		}
	}
	return tt
}

func (b *builder) VisitTupleElementDefinition(ctx *grammar.TupleElementDefinitionContext) interface{} {
	te := &ast.TupleElementDef{}
	if ri := ctx.ReferentialIdentifier(); ri != nil {
		te.Name = referentialIdentifierText(ri)
	}
	if ts := ctx.TypeSpecifier(); ts != nil {
		te.Type = b.visitTypeSpecifier(ts)
	}
	return te
}

func (b *builder) VisitChoiceTypeSpecifier(ctx *grammar.ChoiceTypeSpecifierContext) interface{} {
	ct := &ast.ChoiceType{}
	for _, ts := range ctx.AllTypeSpecifier() {
		ct.Types = append(ct.Types, b.visitTypeSpecifier(ts))
	}
	return ct
}

// ---------------------------------------------------------------------------
// Expressions
// ---------------------------------------------------------------------------

func (b *builder) VisitTermExpression(ctx *grammar.TermExpressionContext) interface{} {
	return b.visitExpressionTerm(ctx.ExpressionTerm())
}

func (b *builder) VisitRetrieveExpression(ctx *grammar.RetrieveExpressionContext) interface{} {
	return b.Visit(ctx.Retrieve())
}

func (b *builder) VisitQueryExpression(ctx *grammar.QueryExpressionContext) interface{} {
	return b.Visit(ctx.Query())
}

func (b *builder) VisitBooleanExpression(ctx *grammar.BooleanExpressionContext) interface{} {
	expr := b.visitExpression(ctx.Expression())
	text := ctx.GetText()
	not := strings.Contains(text, "not")
	testVal := "null"
	if strings.HasSuffix(text, "true") {
		testVal = "true"
	} else if strings.HasSuffix(text, "false") {
		testVal = "false"
	}
	return &ast.BooleanTestExpression{
		Operand:   expr,
		TestValue: testVal,
		Not:       not,
	}
}

func (b *builder) VisitTypeExpression(ctx *grammar.TypeExpressionContext) interface{} {
	expr := b.visitExpression(ctx.Expression())
	ts := b.visitTypeSpecifier(ctx.TypeSpecifier())
	text := ctx.GetText()
	if strings.Contains(text, " is ") || strings.HasPrefix(text, "is ") {
		return &ast.IsExpression{Operand: expr, Type: ts}
	}
	return &ast.AsExpression{Operand: expr, Type: ts}
}

func (b *builder) VisitCastExpression(ctx *grammar.CastExpressionContext) interface{} {
	expr := b.visitExpression(ctx.Expression())
	ts := b.visitTypeSpecifier(ctx.TypeSpecifier())
	return &ast.CastExpression{Operand: expr, Type: ts}
}

func (b *builder) VisitNotExpression(ctx *grammar.NotExpressionContext) interface{} {
	expr := b.visitExpression(ctx.Expression())
	return &ast.UnaryExpression{Operator: ast.OpNot, Operand: expr}
}

func (b *builder) VisitExistenceExpression(ctx *grammar.ExistenceExpressionContext) interface{} {
	expr := b.visitExpression(ctx.Expression())
	return &ast.UnaryExpression{Operator: ast.OpExists, Operand: expr}
}

func (b *builder) VisitBetweenExpression(ctx *grammar.BetweenExpressionContext) interface{} {
	exprs := ctx.AllExpressionTerm()
	expr := b.visitExpression(ctx.Expression())
	properly := strings.Contains(ctx.GetText(), "properly")
	var low, high ast.Expression
	if len(exprs) >= 2 {
		low = b.visitExpressionTerm(exprs[0])
		high = b.visitExpressionTerm(exprs[1])
	}
	return &ast.BetweenExpression{
		Operand:  expr,
		Low:      low,
		High:     high,
		Properly: properly,
	}
}

func (b *builder) VisitDurationBetweenExpression(ctx *grammar.DurationBetweenExpressionContext) interface{} {
	exprs := ctx.AllExpressionTerm()
	precision := ""
	if pdtp := ctx.PluralDateTimePrecision(); pdtp != nil {
		precision = pdtp.GetText()
	}
	var low, high ast.Expression
	if len(exprs) >= 2 {
		low = b.visitExpressionTerm(exprs[0])
		high = b.visitExpressionTerm(exprs[1])
	}
	return &ast.DurationBetween{
		Precision: precision,
		Low:       low,
		High:      high,
	}
}

func (b *builder) VisitDifferenceBetweenExpression(ctx *grammar.DifferenceBetweenExpressionContext) interface{} {
	exprs := ctx.AllExpressionTerm()
	precision := ""
	if pdtp := ctx.PluralDateTimePrecision(); pdtp != nil {
		precision = pdtp.GetText()
	}
	var low, high ast.Expression
	if len(exprs) >= 2 {
		low = b.visitExpressionTerm(exprs[0])
		high = b.visitExpressionTerm(exprs[1])
	}
	return &ast.DifferenceBetween{
		Precision: precision,
		Low:       low,
		High:      high,
	}
}

func (b *builder) VisitInFixSetExpression(ctx *grammar.InFixSetExpressionContext) interface{} {
	exprs := ctx.AllExpression()
	if len(exprs) < 2 {
		return nil
	}
	left := b.visitExpression(exprs[0])
	right := b.visitExpression(exprs[1])
	text := ctx.GetText()
	var op ast.BinaryOp
	switch {
	case strings.Contains(text, "union") || strings.Contains(text, "|"):
		op = ast.OpUnion
	case strings.Contains(text, "intersect"):
		op = ast.OpIntersect
	case strings.Contains(text, "except"):
		op = ast.OpExcept
	}
	return &ast.BinaryExpression{Operator: op, Left: left, Right: right}
}

func (b *builder) VisitInequalityExpression(ctx *grammar.InequalityExpressionContext) interface{} {
	exprs := ctx.AllExpression()
	if len(exprs) < 2 {
		return nil
	}
	left := b.visitExpression(exprs[0])
	right := b.visitExpression(exprs[1])
	// Find the operator token between the two expressions
	var op ast.BinaryOp
	for i := 0; i < ctx.GetChildCount(); i++ {
		child := ctx.GetChild(i)
		if tn, ok := child.(antlr.TerminalNode); ok {
			switch tn.GetText() {
			case "<=":
				op = ast.OpLessOrEqual
			case "<":
				op = ast.OpLess
			case ">":
				op = ast.OpGreater
			case ">=":
				op = ast.OpGreaterOrEqual
			}
		}
	}
	return &ast.BinaryExpression{Operator: op, Left: left, Right: right}
}

func (b *builder) VisitEqualityExpression(ctx *grammar.EqualityExpressionContext) interface{} {
	exprs := ctx.AllExpression()
	if len(exprs) < 2 {
		return nil
	}
	left := b.visitExpression(exprs[0])
	right := b.visitExpression(exprs[1])
	var op ast.BinaryOp
	for i := 0; i < ctx.GetChildCount(); i++ {
		child := ctx.GetChild(i)
		if tn, ok := child.(antlr.TerminalNode); ok {
			switch tn.GetText() {
			case "=":
				op = ast.OpEqual
			case "!=":
				op = ast.OpNotEqual
			case "~":
				op = ast.OpEquivalent
			case "!~":
				op = ast.OpNotEquivalent
			}
		}
	}
	return &ast.BinaryExpression{Operator: op, Left: left, Right: right}
}

func (b *builder) VisitMembershipExpression(ctx *grammar.MembershipExpressionContext) interface{} {
	exprs := ctx.AllExpression()
	if len(exprs) < 2 {
		return nil
	}
	left := b.visitExpression(exprs[0])
	right := b.visitExpression(exprs[1])
	operator := "in"
	for i := 0; i < ctx.GetChildCount(); i++ {
		child := ctx.GetChild(i)
		if tn, ok := child.(antlr.TerminalNode); ok {
			if tn.GetText() == "contains" {
				operator = "contains"
			}
		}
	}
	precision := ""
	if dtps := ctx.DateTimePrecisionSpecifier(); dtps != nil {
		if dtp := dtps.DateTimePrecision(); dtp != nil {
			precision = dtp.GetText()
		}
	}
	return &ast.MembershipExpression{
		Left:      left,
		Right:     right,
		Operator:  operator,
		Precision: precision,
	}
}

func (b *builder) VisitAndExpression(ctx *grammar.AndExpressionContext) interface{} {
	exprs := ctx.AllExpression()
	if len(exprs) < 2 {
		return nil
	}
	return &ast.BinaryExpression{
		Operator: ast.OpAnd,
		Left:     b.visitExpression(exprs[0]),
		Right:    b.visitExpression(exprs[1]),
	}
}

func (b *builder) VisitOrExpression(ctx *grammar.OrExpressionContext) interface{} {
	exprs := ctx.AllExpression()
	if len(exprs) < 2 {
		return nil
	}
	text := ctx.GetText()
	op := ast.OpOr
	if strings.Contains(text, "xor") {
		op = ast.OpXor
	}
	return &ast.BinaryExpression{
		Operator: op,
		Left:     b.visitExpression(exprs[0]),
		Right:    b.visitExpression(exprs[1]),
	}
}

func (b *builder) VisitImpliesExpression(ctx *grammar.ImpliesExpressionContext) interface{} {
	exprs := ctx.AllExpression()
	if len(exprs) < 2 {
		return nil
	}
	return &ast.BinaryExpression{
		Operator: ast.OpImplies,
		Left:     b.visitExpression(exprs[0]),
		Right:    b.visitExpression(exprs[1]),
	}
}

func (b *builder) VisitTimingExpression(ctx *grammar.TimingExpressionContext) interface{} {
	exprs := ctx.AllExpression()
	if len(exprs) < 2 {
		return nil
	}
	left := b.visitExpression(exprs[0])
	right := b.visitExpression(exprs[1])

	op := ast.TimingOp{Kind: ast.TimingSameAs}

	// Parse the IntervalOperatorPhrase to determine the actual operator
	iop := ctx.IntervalOperatorPhrase()
	if iop != nil {
		switch phrase := iop.(type) {
		case *grammar.ConcurrentWithIntervalOperatorPhraseContext:
			// 'same [precision] (as | or before | or after)'
			op.Kind = ast.TimingSameAs
			if dtp := phrase.DateTimePrecision(); dtp != nil {
				op.Precision = strings.ToLower(dtp.GetText())
			}
			if rq := phrase.RelativeQualifier(); rq != nil {
				text := strings.ToLower(rq.GetText())
				if strings.Contains(text, "before") {
					op.Before = true // same or before
				} else if strings.Contains(text, "after") {
					op.After = true // same or after
				}
			}
		case *grammar.BeforeOrAfterIntervalOperatorPhraseContext:
			// '[starts|ends|occurs] [quantityOffset] temporalRelationship [dateTimePrecisionSpecifier]'
			// temporalRelationship: ['on or'] ('before'|'after') | ('before'|'after') ['or on']
			if tr := phrase.TemporalRelationship(); tr != nil {
				text := strings.ToLower(tr.GetText())
				hasOnOr := strings.Contains(text, "on or") || strings.Contains(text, "or on")
				if hasOnOr {
					// "on or before" / "on or after" / "before or on" / "after or on"
					// These are same-or-before / same-or-after semantics
					op.Kind = ast.TimingSameAs
					if strings.Contains(text, "before") {
						op.Before = true
					} else if strings.Contains(text, "after") {
						op.After = true
					}
				} else {
					// plain "before" / "after"
					op.Kind = ast.TimingBeforeOrAfter
					if strings.Contains(text, "before") {
						op.Before = true
					} else if strings.Contains(text, "after") {
						op.After = true
					}
				}
			}
			if dtps := phrase.DateTimePrecisionSpecifier(); dtps != nil {
				if dtp := dtps.DateTimePrecision(); dtp != nil {
					op.Precision = strings.ToLower(dtp.GetText())
				}
			}
		case *grammar.IncludesIntervalOperatorPhraseContext:
			op.Kind = ast.TimingIncludes
			text := strings.ToLower(phrase.GetText())
			if strings.Contains(text, "properly") {
				op.Properly = true
			}
		case *grammar.IncludedInIntervalOperatorPhraseContext:
			op.Kind = ast.TimingIncludedIn
			text := strings.ToLower(phrase.GetText())
			if strings.Contains(text, "properly") {
				op.Properly = true
			}
		case *grammar.MeetsIntervalOperatorPhraseContext:
			op.Kind = ast.TimingMeets
			text := strings.ToLower(phrase.GetText())
			if strings.Contains(text, "before") {
				op.Before = true
			} else if strings.Contains(text, "after") {
				op.After = true
			}
		case *grammar.OverlapsIntervalOperatorPhraseContext:
			op.Kind = ast.TimingOverlaps
			text := strings.ToLower(phrase.GetText())
			if strings.Contains(text, "before") {
				op.Before = true
			} else if strings.Contains(text, "after") {
				op.After = true
			}
		case *grammar.StartsIntervalOperatorPhraseContext:
			op.Kind = ast.TimingStarts
		case *grammar.EndsIntervalOperatorPhraseContext:
			op.Kind = ast.TimingEnds
		case *grammar.WithinIntervalOperatorPhraseContext:
			op.Kind = ast.TimingWithin
		}
	}

	return &ast.TimingExpression{
		Left:     left,
		Right:    right,
		Operator: op,
	}
}

// ---------------------------------------------------------------------------
// Expression Terms
// ---------------------------------------------------------------------------

func (b *builder) VisitTermExpressionTerm(ctx *grammar.TermExpressionTermContext) interface{} {
	return b.Visit(ctx.Term())
}

func (b *builder) VisitInvocationExpressionTerm(ctx *grammar.InvocationExpressionTermContext) interface{} {
	source := b.visitExpressionTerm(ctx.ExpressionTerm())
	qi := ctx.QualifiedInvocation()
	if qi == nil {
		return source
	}
	result := b.Visit(qi)
	switch v := result.(type) {
	case *ast.MemberAccess:
		v.Source = source
		return v
	case *ast.FunctionCall:
		v.Source = source
		return v
	default:
		return source
	}
}

func (b *builder) VisitIndexedExpressionTerm(ctx *grammar.IndexedExpressionTermContext) interface{} {
	source := b.visitExpressionTerm(ctx.ExpressionTerm())
	index := b.visitExpression(ctx.Expression())
	return &ast.IndexAccess{Source: source, Index: index}
}

func (b *builder) VisitPolarityExpressionTerm(ctx *grammar.PolarityExpressionTermContext) interface{} {
	operand := b.visitExpressionTerm(ctx.ExpressionTerm())
	text := strings.TrimSpace(ctx.GetText())
	if strings.HasPrefix(text, "-") {
		return &ast.UnaryExpression{Operator: ast.OpNegate, Operand: operand}
	}
	return &ast.UnaryExpression{Operator: ast.OpPositive, Operand: operand}
}

func (b *builder) VisitTimeBoundaryExpressionTerm(ctx *grammar.TimeBoundaryExpressionTermContext) interface{} {
	operand := b.visitExpressionTerm(ctx.ExpressionTerm())
	text := ctx.GetText()
	if strings.HasPrefix(text, "start") {
		return &ast.UnaryExpression{Operator: ast.OpStartOf, Operand: operand}
	}
	return &ast.UnaryExpression{Operator: ast.OpEndOf, Operand: operand}
}

func (b *builder) VisitTimeUnitExpressionTerm(ctx *grammar.TimeUnitExpressionTermContext) interface{} {
	operand := b.visitExpressionTerm(ctx.ExpressionTerm())
	component := ""
	if dtc := ctx.DateTimeComponent(); dtc != nil {
		component = dtc.GetText()
	}
	return &ast.DateTimeComponentFrom{Component: component, Operand: operand}
}

func (b *builder) VisitDurationExpressionTerm(ctx *grammar.DurationExpressionTermContext) interface{} {
	operand := b.visitExpressionTerm(ctx.ExpressionTerm())
	precision := ""
	if pdtp := ctx.PluralDateTimePrecision(); pdtp != nil {
		precision = pdtp.GetText()
	}
	return &ast.DurationOf{Precision: precision, Operand: operand}
}

func (b *builder) VisitDifferenceExpressionTerm(ctx *grammar.DifferenceExpressionTermContext) interface{} {
	operand := b.visitExpressionTerm(ctx.ExpressionTerm())
	precision := ""
	if pdtp := ctx.PluralDateTimePrecision(); pdtp != nil {
		precision = pdtp.GetText()
	}
	return &ast.DifferenceOf{Precision: precision, Operand: operand}
}

func (b *builder) VisitWidthExpressionTerm(ctx *grammar.WidthExpressionTermContext) interface{} {
	operand := b.visitExpressionTerm(ctx.ExpressionTerm())
	return &ast.UnaryExpression{Operator: ast.OpWidthOf, Operand: operand}
}

func (b *builder) VisitSuccessorExpressionTerm(ctx *grammar.SuccessorExpressionTermContext) interface{} {
	operand := b.visitExpressionTerm(ctx.ExpressionTerm())
	return &ast.UnaryExpression{Operator: ast.OpSuccessorOf, Operand: operand}
}

func (b *builder) VisitPredecessorExpressionTerm(ctx *grammar.PredecessorExpressionTermContext) interface{} {
	operand := b.visitExpressionTerm(ctx.ExpressionTerm())
	return &ast.UnaryExpression{Operator: ast.OpPredecessorOf, Operand: operand}
}

func (b *builder) VisitElementExtractorExpressionTerm(ctx *grammar.ElementExtractorExpressionTermContext) interface{} {
	operand := b.visitExpressionTerm(ctx.ExpressionTerm())
	return &ast.UnaryExpression{Operator: ast.OpSingletonFrom, Operand: operand}
}

func (b *builder) VisitPointExtractorExpressionTerm(ctx *grammar.PointExtractorExpressionTermContext) interface{} {
	operand := b.visitExpressionTerm(ctx.ExpressionTerm())
	return &ast.UnaryExpression{Operator: ast.OpPointFrom, Operand: operand}
}

func (b *builder) VisitTypeExtentExpressionTerm(ctx *grammar.TypeExtentExpressionTermContext) interface{} {
	text := ctx.GetText()
	extent := "minimum"
	if strings.HasPrefix(text, "maximum") {
		extent = "maximum"
	}
	var nt *ast.NamedType
	if nts := ctx.NamedTypeSpecifier(); nts != nil {
		result := b.Visit(nts)
		if n, ok := result.(*ast.NamedType); ok {
			nt = n
		}
	}
	return &ast.TypeExtent{Extent: extent, Type: nt}
}

func (b *builder) VisitConversionExpressionTerm(ctx *grammar.ConversionExpressionTermContext) interface{} {
	expr := b.visitExpression(ctx.Expression())
	ce := &ast.ConvertExpression{Operand: expr}
	if ts := ctx.TypeSpecifier(); ts != nil {
		ce.ToType = b.visitTypeSpecifier(ts)
	}
	if u := ctx.Unit(); u != nil {
		ce.ToUnit = u.GetText()
	}
	return ce
}

func (b *builder) VisitPowerExpressionTerm(ctx *grammar.PowerExpressionTermContext) interface{} {
	terms := ctx.AllExpressionTerm()
	if len(terms) < 2 {
		return nil
	}
	return &ast.BinaryExpression{
		Operator: ast.OpPower,
		Left:     b.visitExpressionTerm(terms[0]),
		Right:    b.visitExpressionTerm(terms[1]),
	}
}

func (b *builder) VisitMultiplicationExpressionTerm(ctx *grammar.MultiplicationExpressionTermContext) interface{} {
	terms := ctx.AllExpressionTerm()
	if len(terms) < 2 {
		return nil
	}
	var op ast.BinaryOp
	for i := 0; i < ctx.GetChildCount(); i++ {
		child := ctx.GetChild(i)
		if tn, ok := child.(antlr.TerminalNode); ok {
			switch tn.GetText() {
			case "*":
				op = ast.OpMultiply
			case "/":
				op = ast.OpDivide
			case "div":
				op = ast.OpDiv
			case "mod":
				op = ast.OpMod
			}
		}
	}
	return &ast.BinaryExpression{
		Operator: op,
		Left:     b.visitExpressionTerm(terms[0]),
		Right:    b.visitExpressionTerm(terms[1]),
	}
}

func (b *builder) VisitAdditionExpressionTerm(ctx *grammar.AdditionExpressionTermContext) interface{} {
	terms := ctx.AllExpressionTerm()
	if len(terms) < 2 {
		return nil
	}
	var op ast.BinaryOp
	for i := 0; i < ctx.GetChildCount(); i++ {
		child := ctx.GetChild(i)
		if tn, ok := child.(antlr.TerminalNode); ok {
			switch tn.GetText() {
			case "+":
				op = ast.OpAdd
			case "-":
				op = ast.OpSubtract
			case "&":
				op = ast.OpConcatenate
			}
		}
	}
	return &ast.BinaryExpression{
		Operator: op,
		Left:     b.visitExpressionTerm(terms[0]),
		Right:    b.visitExpressionTerm(terms[1]),
	}
}

func (b *builder) VisitIfThenElseExpressionTerm(ctx *grammar.IfThenElseExpressionTermContext) interface{} {
	exprs := ctx.AllExpression()
	if len(exprs) < 3 {
		return nil
	}
	return &ast.IfThenElse{
		Condition: b.visitExpression(exprs[0]),
		Then:      b.visitExpression(exprs[1]),
		Else:      b.visitExpression(exprs[2]),
	}
}

func (b *builder) VisitCaseExpressionTerm(ctx *grammar.CaseExpressionTermContext) interface{} {
	ce := &ast.CaseExpression{}
	exprs := ctx.AllExpression()
	// If there's an odd number of expressions, the first is the comparand and the last is 'else'
	// If there's only one expression, it's the 'else' (no comparand)
	items := ctx.AllCaseExpressionItem()
	// Check if there's a comparand (expression before first 'when')
	exprIdx := 0
	if len(exprs) > len(items)*2+1 {
		// comparand present
		ce.Comparand = b.visitExpression(exprs[0])
		exprIdx = 1
	}
	_ = exprIdx
	for _, item := range items {
		result := b.Visit(item)
		if ci, ok := result.(*ast.CaseItem); ok {
			ce.Items = append(ce.Items, ci)
		}
	}
	// Last expression is the 'else'
	if len(exprs) > 0 {
		ce.Else = b.visitExpression(exprs[len(exprs)-1])
	}
	return ce
}

func (b *builder) VisitCaseExpressionItem(ctx *grammar.CaseExpressionItemContext) interface{} {
	exprs := ctx.AllExpression()
	if len(exprs) < 2 {
		return nil
	}
	return &ast.CaseItem{
		When: b.visitExpression(exprs[0]),
		Then: b.visitExpression(exprs[1]),
	}
}

func (b *builder) VisitAggregateExpressionTerm(ctx *grammar.AggregateExpressionTermContext) interface{} {
	expr := b.visitExpression(ctx.Expression())
	text := ctx.GetText()
	if strings.HasPrefix(text, "distinct") {
		return &ast.UnaryExpression{Operator: ast.OpDistinct, Operand: expr}
	}
	return &ast.UnaryExpression{Operator: ast.OpFlatten, Operand: expr}
}

func (b *builder) VisitSetAggregateExpressionTerm(ctx *grammar.SetAggregateExpressionTermContext) interface{} {
	exprs := ctx.AllExpression()
	sa := &ast.SetAggregateExpression{}
	text := ctx.GetText()
	if strings.HasPrefix(text, "expand") {
		sa.Kind = "expand"
	} else {
		sa.Kind = "collapse"
	}
	if len(exprs) > 0 {
		sa.Operand = b.visitExpression(exprs[0])
	}
	if len(exprs) > 1 {
		sa.Per = b.visitExpression(exprs[1])
	}
	return sa
}

// ---------------------------------------------------------------------------
// Terms
// ---------------------------------------------------------------------------

func (b *builder) VisitInvocationTerm(ctx *grammar.InvocationTermContext) interface{} {
	return b.Visit(ctx.Invocation())
}

func (b *builder) VisitLiteralTerm(ctx *grammar.LiteralTermContext) interface{} {
	return b.Visit(ctx.Literal())
}

func (b *builder) VisitExternalConstantTerm(ctx *grammar.ExternalConstantTermContext) interface{} {
	return b.Visit(ctx.ExternalConstant())
}

func (b *builder) VisitIntervalSelectorTerm(ctx *grammar.IntervalSelectorTermContext) interface{} {
	return b.Visit(ctx.IntervalSelector())
}

func (b *builder) VisitTupleSelectorTerm(ctx *grammar.TupleSelectorTermContext) interface{} {
	return b.Visit(ctx.TupleSelector())
}

func (b *builder) VisitInstanceSelectorTerm(ctx *grammar.InstanceSelectorTermContext) interface{} {
	return b.Visit(ctx.InstanceSelector())
}

func (b *builder) VisitListSelectorTerm(ctx *grammar.ListSelectorTermContext) interface{} {
	return b.Visit(ctx.ListSelector())
}

func (b *builder) VisitCodeSelectorTerm(ctx *grammar.CodeSelectorTermContext) interface{} {
	return b.Visit(ctx.CodeSelector())
}

func (b *builder) VisitConceptSelectorTerm(ctx *grammar.ConceptSelectorTermContext) interface{} {
	return b.Visit(ctx.ConceptSelector())
}

func (b *builder) VisitParenthesizedTerm(ctx *grammar.ParenthesizedTermContext) interface{} {
	return b.visitExpression(ctx.Expression())
}

// ---------------------------------------------------------------------------
// Invocations
// ---------------------------------------------------------------------------

func (b *builder) VisitMemberInvocation(ctx *grammar.MemberInvocationContext) interface{} {
	name := referentialIdentifierText(ctx.ReferentialIdentifier())
	return &ast.IdentifierRef{Name: name}
}

func (b *builder) VisitFunctionInvocation(ctx *grammar.FunctionInvocationContext) interface{} {
	return b.Visit(ctx.Function())
}

func (b *builder) VisitThisInvocation(_ *grammar.ThisInvocationContext) interface{} {
	return &ast.ThisExpression{}
}

func (b *builder) VisitIndexInvocation(_ *grammar.IndexInvocationContext) interface{} {
	return &ast.IndexExpression{}
}

func (b *builder) VisitTotalInvocation(_ *grammar.TotalInvocationContext) interface{} {
	return &ast.TotalExpression{}
}

func (b *builder) VisitFunction(ctx *grammar.FunctionContext) interface{} {
	fc := &ast.FunctionCall{}
	if ri := ctx.ReferentialIdentifier(); ri != nil {
		fc.Name = referentialIdentifierText(ri)
	}
	if pl := ctx.ParamList(); pl != nil {
		for _, expr := range pl.AllExpression() {
			fc.Operands = append(fc.Operands, b.visitExpression(expr))
		}
	}
	return fc
}

func (b *builder) VisitQualifiedMemberInvocation(ctx *grammar.QualifiedMemberInvocationContext) interface{} {
	name := referentialIdentifierText(ctx.ReferentialIdentifier())
	return &ast.MemberAccess{Member: name}
}

func (b *builder) VisitQualifiedFunctionInvocation(ctx *grammar.QualifiedFunctionInvocationContext) interface{} {
	return b.Visit(ctx.QualifiedFunction())
}

func (b *builder) VisitQualifiedFunction(ctx *grammar.QualifiedFunctionContext) interface{} {
	fc := &ast.FunctionCall{}
	if id := ctx.IdentifierOrFunctionIdentifier(); id != nil {
		fc.Name = id.GetText()
	}
	if pl := ctx.ParamList(); pl != nil {
		for _, expr := range pl.AllExpression() {
			fc.Operands = append(fc.Operands, b.visitExpression(expr))
		}
	}
	return fc
}

// ---------------------------------------------------------------------------
// Literals
// ---------------------------------------------------------------------------

func (b *builder) VisitBooleanLiteral(ctx *grammar.BooleanLiteralContext) interface{} {
	return &ast.Literal{ValueType: ast.LiteralBoolean, Value: ctx.GetText()}
}

func (b *builder) VisitNullLiteral(_ *grammar.NullLiteralContext) interface{} {
	return &ast.Literal{ValueType: ast.LiteralNull, Value: "null"}
}

func (b *builder) VisitStringLiteral(ctx *grammar.StringLiteralContext) interface{} {
	return &ast.Literal{ValueType: ast.LiteralString, Value: unquoteString(ctx.GetText())}
}

func (b *builder) VisitNumberLiteral(ctx *grammar.NumberLiteralContext) interface{} {
	text := ctx.GetText()
	if strings.Contains(text, ".") {
		return &ast.Literal{ValueType: ast.LiteralDecimal, Value: text}
	}
	return &ast.Literal{ValueType: ast.LiteralInteger, Value: text}
}

func (b *builder) VisitLongNumberLiteral(ctx *grammar.LongNumberLiteralContext) interface{} {
	text := ctx.GetText()
	text = strings.TrimSuffix(text, "L")
	return &ast.Literal{ValueType: ast.LiteralLong, Value: text}
}

func (b *builder) VisitDateTimeLiteral(ctx *grammar.DateTimeLiteralContext) interface{} {
	text := ctx.GetText()
	text = strings.TrimPrefix(text, "@")
	return &ast.Literal{ValueType: ast.LiteralDateTime, Value: text}
}

func (b *builder) VisitDateLiteral(ctx *grammar.DateLiteralContext) interface{} {
	text := ctx.GetText()
	text = strings.TrimPrefix(text, "@")
	return &ast.Literal{ValueType: ast.LiteralDate, Value: text}
}

func (b *builder) VisitTimeLiteral(ctx *grammar.TimeLiteralContext) interface{} {
	text := ctx.GetText()
	text = strings.TrimPrefix(text, "@T")
	return &ast.Literal{ValueType: ast.LiteralTime, Value: text}
}

func (b *builder) VisitQuantityLiteral(ctx *grammar.QuantityLiteralContext) interface{} {
	return &ast.Literal{ValueType: ast.LiteralQuantity, Value: ctx.GetText()}
}

func (b *builder) VisitRatioLiteral(ctx *grammar.RatioLiteralContext) interface{} {
	return &ast.Literal{ValueType: ast.LiteralRatio, Value: ctx.GetText()}
}

// ---------------------------------------------------------------------------
// External Constants
// ---------------------------------------------------------------------------

func (b *builder) VisitExternalConstant(ctx *grammar.ExternalConstantContext) interface{} {
	name := ""
	if id := ctx.Identifier(); id != nil {
		name = identifierText(id)
	} else if kw := ctx.KeywordIdentifier(); kw != nil {
		name = kw.GetText()
	} else if s := ctx.STRING(); s != nil {
		name = unquoteString(s.GetText())
	}
	return &ast.ExternalConstant{Name: name}
}

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

func (b *builder) VisitIntervalSelector(ctx *grammar.IntervalSelectorContext) interface{} {
	ie := &ast.IntervalExpression{}
	exprs := ctx.AllExpression()
	if len(exprs) >= 2 {
		ie.Low = b.visitExpression(exprs[0])
		ie.High = b.visitExpression(exprs[1])
	}
	// Check boundary markers
	text := ctx.GetText()
	ie.LowClosed = strings.Contains(text, "Interval[")
	ie.HighClosed = strings.HasSuffix(strings.TrimSpace(text), "]")
	return ie
}

func (b *builder) VisitTupleSelector(ctx *grammar.TupleSelectorContext) interface{} {
	te := &ast.TupleExpression{}
	for _, tes := range ctx.AllTupleElementSelector() {
		result := b.Visit(tes)
		if elem, ok := result.(*ast.TupleElement); ok {
			te.Elements = append(te.Elements, elem)
		}
	}
	return te
}

func (b *builder) VisitTupleElementSelector(ctx *grammar.TupleElementSelectorContext) interface{} {
	te := &ast.TupleElement{}
	if ri := ctx.ReferentialIdentifier(); ri != nil {
		te.Name = referentialIdentifierText(ri)
	}
	if expr := ctx.Expression(); expr != nil {
		te.Expression = b.visitExpression(expr)
	}
	return te
}

func (b *builder) VisitInstanceSelector(ctx *grammar.InstanceSelectorContext) interface{} {
	ie := &ast.InstanceExpression{}
	if nts := ctx.NamedTypeSpecifier(); nts != nil {
		result := b.Visit(nts)
		if nt, ok := result.(*ast.NamedType); ok {
			ie.Type = nt
		}
	}
	for _, ies := range ctx.AllInstanceElementSelector() {
		result := b.Visit(ies)
		if elem, ok := result.(*ast.TupleElement); ok {
			ie.Elements = append(ie.Elements, elem)
		}
	}
	return ie
}

func (b *builder) VisitInstanceElementSelector(ctx *grammar.InstanceElementSelectorContext) interface{} {
	te := &ast.TupleElement{}
	if ri := ctx.ReferentialIdentifier(); ri != nil {
		te.Name = referentialIdentifierText(ri)
	}
	if expr := ctx.Expression(); expr != nil {
		te.Expression = b.visitExpression(expr)
	}
	return te
}

func (b *builder) VisitListSelector(ctx *grammar.ListSelectorContext) interface{} {
	le := &ast.ListExpression{}
	if ts := ctx.TypeSpecifier(); ts != nil {
		le.TypeSpec = b.visitTypeSpecifier(ts)
	}
	for _, expr := range ctx.AllExpression() {
		le.Elements = append(le.Elements, b.visitExpression(expr))
	}
	return le
}

func (b *builder) VisitCodeSelector(ctx *grammar.CodeSelectorContext) interface{} {
	ce := &ast.CodeExpression{}
	if s := ctx.STRING(); s != nil {
		ce.Code = unquoteString(s.GetText())
	}
	if csid := ctx.CodesystemIdentifier(); csid != nil {
		ce.System = csid.GetText()
	}
	if dc := ctx.DisplayClause(); dc != nil {
		ce.Display = unquoteString(dc.STRING().GetText())
	}
	return ce
}

func (b *builder) VisitConceptSelector(ctx *grammar.ConceptSelectorContext) interface{} {
	ce := &ast.ConceptExpression{}
	for _, cs := range ctx.AllCodeSelector() {
		result := b.Visit(cs)
		if code, ok := result.(*ast.CodeExpression); ok {
			ce.Codes = append(ce.Codes, code)
		}
	}
	if dc := ctx.DisplayClause(); dc != nil {
		ce.Display = unquoteString(dc.STRING().GetText())
	}
	return ce
}

// ---------------------------------------------------------------------------
// Retrieve
// ---------------------------------------------------------------------------

func (b *builder) VisitRetrieve(ctx *grammar.RetrieveContext) interface{} {
	r := &ast.Retrieve{}
	if nts := ctx.NamedTypeSpecifier(); nts != nil {
		result := b.Visit(nts)
		if nt, ok := result.(*ast.NamedType); ok {
			r.ResourceType = nt
		}
	}
	if cp := ctx.CodePath(); cp != nil {
		r.CodePath = cp.GetText()
	}
	if cc := ctx.CodeComparator(); cc != nil {
		r.CodeComparator = cc.GetText()
	}
	if t := ctx.Terminology(); t != nil {
		// terminology can be a qualifiedIdentifierExpression or an expression
		if qie := t.QualifiedIdentifierExpression(); qie != nil {
			r.Codes = &ast.IdentifierRef{Name: qie.GetText()}
		} else if expr := t.Expression(); expr != nil {
			r.Codes = b.visitExpression(expr)
		}
	}
	if ci := ctx.ContextIdentifier(); ci != nil {
		if qie := ci.QualifiedIdentifierExpression(); qie != nil {
			r.Context = &ast.IdentifierRef{Name: qie.GetText()}
		}
	}
	return r
}

// ---------------------------------------------------------------------------
// Query
// ---------------------------------------------------------------------------

func (b *builder) VisitQuery(ctx *grammar.QueryContext) interface{} {
	q := &ast.Query{}
	if sc := ctx.SourceClause(); sc != nil {
		for _, aqs := range sc.AllAliasedQuerySource() {
			result := b.Visit(aqs)
			if as, ok := result.(*ast.AliasedSource); ok {
				q.Sources = append(q.Sources, as)
			}
		}
	}
	if lc := ctx.LetClause(); lc != nil {
		for _, lci := range lc.AllLetClauseItem() {
			result := b.Visit(lci)
			if l, ok := result.(*ast.LetClause); ok {
				q.Let = append(q.Let, l)
			}
		}
	}
	for _, qic := range ctx.AllQueryInclusionClause() {
		if wc := qic.WithClause(); wc != nil {
			result := b.Visit(wc)
			if w, ok := result.(*ast.WithClause); ok {
				q.With = append(q.With, w)
			}
		}
		if woc := qic.WithoutClause(); woc != nil {
			result := b.Visit(woc)
			if w, ok := result.(*ast.WithoutClause); ok {
				q.Without = append(q.Without, w)
			}
		}
	}
	if wc := ctx.WhereClause(); wc != nil {
		q.Where = b.visitExpression(wc.Expression())
	}
	if rc := ctx.ReturnClause(); rc != nil {
		result := b.Visit(rc)
		if r, ok := result.(*ast.ReturnClause); ok {
			q.Return = r
		}
	}
	if ac := ctx.AggregateClause(); ac != nil {
		result := b.Visit(ac)
		if a, ok := result.(*ast.AggregateClause); ok {
			q.Aggregate = a
		}
	}
	if sc := ctx.SortClause(); sc != nil {
		result := b.Visit(sc)
		if s, ok := result.(*ast.SortClause); ok {
			q.Sort = s
		}
	}
	return q
}

func (b *builder) VisitAliasedQuerySource(ctx *grammar.AliasedQuerySourceContext) interface{} {
	as := &ast.AliasedSource{}
	if qs := ctx.QuerySource(); qs != nil {
		if r := qs.Retrieve(); r != nil {
			as.Source = b.Visit(r).(ast.Expression)
		} else if qie := qs.QualifiedIdentifierExpression(); qie != nil {
			as.Source = &ast.IdentifierRef{Name: qie.GetText()}
		} else if expr := qs.Expression(); expr != nil {
			as.Source = b.visitExpression(expr)
		}
	}
	if a := ctx.Alias(); a != nil {
		as.Alias = a.GetText()
	}
	return as
}

func (b *builder) VisitLetClauseItem(ctx *grammar.LetClauseItemContext) interface{} {
	lc := &ast.LetClause{}
	if id := ctx.Identifier(); id != nil {
		lc.Identifier = identifierText(id)
	}
	if expr := ctx.Expression(); expr != nil {
		lc.Expression = b.visitExpression(expr)
	}
	return lc
}

func (b *builder) VisitWithClause(ctx *grammar.WithClauseContext) interface{} {
	wc := &ast.WithClause{}
	if aqs := ctx.AliasedQuerySource(); aqs != nil {
		result := b.Visit(aqs)
		if as, ok := result.(*ast.AliasedSource); ok {
			wc.Source = as
		}
	}
	if expr := ctx.Expression(); expr != nil {
		wc.Condition = b.visitExpression(expr)
	}
	return wc
}

func (b *builder) VisitWithoutClause(ctx *grammar.WithoutClauseContext) interface{} {
	wc := &ast.WithoutClause{}
	if aqs := ctx.AliasedQuerySource(); aqs != nil {
		result := b.Visit(aqs)
		if as, ok := result.(*ast.AliasedSource); ok {
			wc.Source = as
		}
	}
	if expr := ctx.Expression(); expr != nil {
		wc.Condition = b.visitExpression(expr)
	}
	return wc
}

func (b *builder) VisitReturnClause(ctx *grammar.ReturnClauseContext) interface{} {
	rc := &ast.ReturnClause{}
	text := ctx.GetText()
	rc.Distinct = strings.Contains(text, "distinct")
	rc.All = strings.Contains(text, "all")
	if expr := ctx.Expression(); expr != nil {
		rc.Expression = b.visitExpression(expr)
	}
	return rc
}

func (b *builder) VisitAggregateClause(ctx *grammar.AggregateClauseContext) interface{} {
	ac := &ast.AggregateClause{}
	if id := ctx.Identifier(); id != nil {
		ac.Identifier = identifierText(id)
	}
	text := ctx.GetText()
	ac.Distinct = strings.Contains(text, "distinct")
	ac.All = strings.Contains(text, "all")
	if sc := ctx.StartingClause(); sc != nil {
		// Starting can contain a simpleLiteral, quantity, or parenthesized expression
		if expr := sc.Expression(); expr != nil {
			ac.Starting = b.visitExpression(expr)
		} else if sl := sc.SimpleLiteral(); sl != nil {
			ac.Starting = &ast.Literal{ValueType: ast.LiteralString, Value: sl.GetText()}
		}
	}
	if expr := ctx.Expression(); expr != nil {
		ac.Expression = b.visitExpression(expr)
	}
	return ac
}

func (b *builder) VisitSortClause(ctx *grammar.SortClauseContext) interface{} {
	sc := &ast.SortClause{}
	if sd := ctx.SortDirection(); sd != nil {
		if strings.Contains(sd.GetText(), "desc") {
			sc.Direction = ast.SortDesc
		}
	}
	for _, sbi := range ctx.AllSortByItem() {
		result := b.Visit(sbi)
		if item, ok := result.(*ast.SortByItem); ok {
			sc.ByItems = append(sc.ByItems, item)
		}
	}
	return sc
}

func (b *builder) VisitSortByItem(ctx *grammar.SortByItemContext) interface{} {
	sbi := &ast.SortByItem{}
	if et := ctx.ExpressionTerm(); et != nil {
		sbi.Expression = b.visitExpressionTerm(et)
	}
	if sd := ctx.SortDirection(); sd != nil {
		if strings.Contains(sd.GetText(), "desc") {
			sbi.Direction = ast.SortDesc
		}
	}
	return sbi
}
