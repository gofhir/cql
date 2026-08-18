package sema

import (
	"fmt"
	"strings"

	"github.com/gofhir/cql/ast"
)

// Result is what one pass over a library decided.
type Result struct {
	// Types is the type of every expression the pass reached, keyed by node.
	// One it reached and could not work out is present, with Unknown: absence
	// means the pass never got there, which is a different thing and worth
	// being able to tell apart. Use TypeOf, which answers Unknown for both.
	Types map[ast.Expression]Type

	// Conversions is where an implicit conversion has to happen for an
	// expression to mean what it says, keyed by the node being converted.
	//
	// The reference translator inserts these into the tree it emits. Recording
	// them beside the tree instead keeps the AST a record of what was written —
	// which is what the ELM importer, the evaluator and the position stamps all
	// assume — while still saying where a FHIRHelpers call belongs.
	Conversions map[ast.Expression]Conversion

	// Defines is the type of each named expression in the library.
	Defines map[string]Type

	// ConversionsByDefine is the same information grouped by the definition the
	// conversion falls inside, which is the shape the reference translator's
	// output is compared against: it says what a definition needs converted,
	// not which node needs it.
	ConversionsByDefine map[string][]Conversion

	// Diagnostics is everything found, in source order.
	Diagnostics Diagnostics
}

// ConversionFor returns the conversion this phase decided an expression needs
// before the context around it can use it.
//
// It is what lets the evaluator apply a decision instead of making one. The
// evaluator can only ask "what does the model convert this value's type to",
// which answers nothing when a type declares more than one conversion and
// nothing at all where the operator never thought to ask — which is most of
// arithmetic and all of comparison.
func (r *Result) ConversionFor(expr ast.Expression) (Conversion, bool) {
	if r == nil || expr == nil {
		return Conversion{}, false
	}
	conv, ok := r.Conversions[expr]
	return conv, ok
}

// TypeOf returns the type inferred for an expression.
func (r *Result) TypeOf(expr ast.Expression) Type {
	if t, ok := r.Types[expr]; ok {
		return t
	}
	return Unknown
}

// Check infers the type of every expression in a library and reports what it
// finds, all of it, in one pass.
//
// It never stops early. An expression it cannot type becomes Unknown, which
// every rule accepts and propagates, so one unresolved name yields one
// diagnostic rather than one per enclosing expression — the recovery the
// reference translator gets from badExpression(), by another route.
//
// The model may be nil. A library that uses no data model still has types, and
// refusing to check it would make the phase unusable for the arithmetic and
// interval tests that need no model at all.
func Check(lib *ast.Library, m Model) *Result {
	c := &checker{
		model:        m,
		lib:          lib,
		types:        map[ast.Expression]Type{},
		conversions:  map[ast.Expression]Conversion{},
		convByDefine: map[string][]Conversion{},
		defines:      map[string]Type{},
		defNodes:     map[string]*ast.ExpressionDef{},
		funcNodes:    map[string][]*ast.FunctionDef{},
		inProgress:   map[string]bool{},
		params:       map[string]Type{},
		terminology:  map[string]Type{},
	}
	c.run()
	return &Result{
		Types:               c.types,
		Conversions:         c.conversions,
		ConversionsByDefine: c.convByDefine,
		Defines:             c.defines,
		Diagnostics:         c.diags.sorted(),
	}
}

// checker holds one pass's state.
type checker struct {
	model Model
	lib   *ast.Library

	types        map[ast.Expression]Type
	conversions  map[ast.Expression]Conversion
	convByDefine map[string][]Conversion
	diags        Diagnostics

	defines    map[string]Type
	defNodes   map[string]*ast.ExpressionDef
	funcNodes  map[string][]*ast.FunctionDef
	inProgress map[string]bool

	params      map[string]Type
	terminology map[string]Type // codesystems, valuesets, codes and concepts

	// scopes are the names bound by enclosing constructs: query aliases, let
	// bindings, function operands. Innermost last.
	scopes []map[string]Type

	// implicit is what an unqualified property access resolves against — the
	// query alias in scope, or the context type outside a query. It is what
	// makes `period` mean `Enc.period` inside `from [Encounter] Enc`.
	implicit Type

	currentDefine string
	contextType   Type

	// speculative counts how deep the pass is inside an inference whose
	// findings another pass will arrive at again. See speculate.
	speculative int
}

func (c *checker) run() {
	c.collectTerminology()
	c.collectParameters()
	c.resolveContext()

	for _, fn := range c.lib.Functions {
		c.funcNodes[fn.Name] = append(c.funcNodes[fn.Name], fn)
	}
	for _, def := range c.lib.Statements {
		c.defNodes[def.Name] = def
	}
	// Walk in source order rather than following references, so that a
	// diagnostic's position is the first place a reader would look. Definitions
	// referenced before they are declared are resolved on demand and memoized.
	for _, def := range c.lib.Statements {
		c.typeOfDefine(def.Name)
	}
	for _, fn := range c.lib.Functions {
		c.checkFunction(fn)
	}
}

// collectTerminology records what the vocabulary declarations name. A valueset
// is a ValueSet, and `code in "Diabetes"` needs that to be true statically as
// much as it does at evaluation.
func (c *checker) collectTerminology() {
	for _, cs := range c.lib.CodeSystems {
		c.terminology[cs.Name] = CodeSystem
	}
	for _, vs := range c.lib.ValueSets {
		c.terminology[vs.Name] = ValueSet
	}
	for _, code := range c.lib.Codes {
		c.terminology[code.Name] = Code
	}
	for _, concept := range c.lib.Concepts {
		c.terminology[concept.Name] = Concept
	}
}

func (c *checker) collectParameters() {
	for _, p := range c.lib.Parameters {
		switch {
		case p.Type != nil:
			c.params[p.Name] = c.resolveTypeSpecifier(p.Type)
		case p.Default != nil:
			// A parameter with no declared type is the type of its default,
			// which is how `parameter MP default Interval[...]` is read.
			c.params[p.Name] = c.infer(p.Default)
		default:
			c.params[p.Name] = Unknown
		}
	}
}

// resolveContext works out what the context statement names, which is the type
// an unqualified property access resolves against outside a query.
//
// A library may change context part-way through, and the last one wins here
// because the AST does not say which statements followed which context:
// Library.Contexts is a list of its own, detached from Library.Statements.
// Getting that right needs the builder to record the context each definition
// was written under, which is a change to the AST and not to this phase.
func (c *checker) resolveContext() {
	c.contextType = Unknown
	for _, ctx := range c.lib.Contexts {
		name := ctx.Name
		if c.model != nil {
			name = c.model.ContextType(ctx.Name)
		}
		model := ctx.Model
		if model == "" {
			model = c.defaultModel()
		}
		c.contextType = &Named{Model: model, Name: unqualify(name)}
	}
}

// defaultModel is the model a bare type name belongs to: the one the library
// declared with `using`, or System when it declared none.
func (c *checker) defaultModel() string {
	for _, u := range c.lib.Usings {
		if u.Name != "" && u.Name != systemModel {
			return u.Name
		}
	}
	return systemModel
}

// typeOfDefine returns the type of a named expression, computing it on first
// use and remembering it.
//
// A definition that refers to itself is given Unknown rather than being
// followed: the alternative is a stack overflow, and CQL has no recursive
// definitions to lose by refusing.
func (c *checker) typeOfDefine(name string) Type {
	if t, ok := c.defines[name]; ok {
		return t
	}
	def, ok := c.defNodes[name]
	if !ok {
		return Unknown
	}
	if c.inProgress[name] {
		c.reportf(def.Expression, SeverityError, "%s is defined in terms of itself", name)
		c.defines[name] = Unknown
		return Unknown
	}
	c.inProgress[name] = true
	t := c.isolated(name, func() Type { return c.infer(def.Expression) })
	delete(c.inProgress, name)

	c.defines[name] = t
	return t
}

// isolated runs an inference as though it were the only one in the library:
// with an empty scope stack, the library's context as what unqualified
// properties resolve against, and its own name on any diagnostic.
//
// It exists because inference is re-entrant. A definition may be reached for
// the first time from the middle of another one's query, and a function body
// from a call site inside one — and whatever was in scope there is not in scope
// here. Leaving the caller's scopes in place let a definition resolve a free
// name against the *caller's* query alias: `define Q: [Encounter] E where Later
// is not null` with `define Later: E` typed Later as an Encounter and reported
// nothing, and memoized that. Which of the two was declared first decided
// whether the library checked clean.
func (c *checker) isolated(define string, infer func() Type) Type {
	savedScopes, savedDefine, savedImplicit := c.scopes, c.currentDefine, c.implicit
	c.scopes, c.currentDefine, c.implicit = nil, define, c.contextType
	t := infer()
	c.scopes, c.currentDefine, c.implicit = savedScopes, savedDefine, savedImplicit
	return t
}

// speculate runs an inference whose findings will be arrived at again, and
// drops what it finds on the way.
//
// A function body with no declared return type has to be typed from each call
// site, to bind operands the declaration left untyped — and again by
// checkFunction, which is the pass that owns it. Only one of them may report:
// otherwise a single mistake in a helper comes back once per caller, and a
// conversion inside the helper is filed under whoever called it rather than
// under the helper.
func (c *checker) speculate(infer func() Type) Type {
	c.speculative++
	t := infer()
	c.speculative--
	return t
}

// checkFunction types a function body with its operands in scope. This is the
// pass that owns a body: the diagnostics and conversions inside it are recorded
// here, under the function's own name, however many call sites it has.
func (c *checker) checkFunction(fn *ast.FunctionDef) {
	if fn.Body == nil {
		return // external: the declaration is all there is
	}
	scope := map[string]Type{}
	for _, op := range fn.Operands {
		scope[op.Name] = c.resolveTypeSpecifier(op.Type)
	}
	body := c.isolated(fn.Name, func() Type {
		c.pushScope(scope)
		defer c.popScope()
		return c.infer(fn.Body)
	})

	// A declared return type is a claim about the body, and one worth checking:
	// `define function F() returns Integer: 'x'` is wrong wherever it is called
	// from, and saying so at the definition is the only place it can be fixed.
	if fn.ReturnType == nil {
		return
	}
	declared := c.resolveTypeSpecifier(fn.ReturnType)
	if IsUnknown(declared) || IsUnknown(body) {
		return
	}
	if _, ok := Convertible(body, declared, c.model); !ok {
		c.reportf(fn.Body, SeverityError,
			"%s returns %s but its body is %s", fn.Name, declared, body)
	}
}

// typeOfFunction returns what a function defined in this library returns.
//
// A declared return type is taken at its word; without one the body is typed,
// which needs the operands bound to the argument types, since a function whose
// operand is untyped is only as specific as its call site.
func (c *checker) typeOfFunction(fn *ast.FunctionDef, args []Type) Type {
	if fn.ReturnType != nil {
		return c.resolveTypeSpecifier(fn.ReturnType)
	}
	if fn.Body == nil {
		return Unknown
	}
	key := "function " + fn.Name
	if c.inProgress[key] {
		return Unknown // recursive, and nothing to unwind to
	}
	c.inProgress[key] = true
	scope := map[string]Type{}
	for i, op := range fn.Operands {
		declared := c.resolveTypeSpecifier(op.Type)
		if IsUnknown(declared) && i < len(args) {
			declared = args[i]
		}
		scope[op.Name] = declared
	}
	t := c.speculate(func() Type {
		return c.isolated(fn.Name, func() Type {
			c.pushScope(scope)
			defer c.popScope()
			return c.infer(fn.Body)
		})
	})
	delete(c.inProgress, key)
	return t
}

// --- scopes ---

func (c *checker) pushScope(scope map[string]Type) { c.scopes = append(c.scopes, scope) }
func (c *checker) popScope()                       { c.scopes = c.scopes[:len(c.scopes)-1] }

// lookup finds a name in the innermost scope that binds it.
func (c *checker) lookup(name string) (Type, bool) {
	for i := len(c.scopes) - 1; i >= 0; i-- {
		if t, ok := c.scopes[i][name]; ok {
			return t, true
		}
	}
	return nil, false
}

// --- diagnostics ---

func (c *checker) reportf(at ast.Expression, sev Severity, format string, args ...any) {
	if c.speculative > 0 {
		return
	}
	pos, _ := ast.PositionOf(at)
	c.diags = append(c.diags, Diagnostic{
		Position: pos,
		Severity: sev,
		Message:  fmt.Sprintf(format, args...),
		Define:   c.currentDefine,
	})
}

// convert records that an expression has to be converted before the context it
// sits in can use it.
//
// Recording it twice — once against the node, once against the definition it
// falls inside — is what lets the same decision be read two ways: a rewriter
// needs the node, and a diff against the reference translator needs the
// definition, because that is the granularity its ELM reports.
func (c *checker) convert(expr ast.Expression, conv Conversion) {
	if conv.Function == "" || c.speculative > 0 {
		return
	}
	if _, already := c.conversions[expr]; already {
		return
	}
	c.conversions[expr] = conv
	c.convByDefine[c.currentDefine] = append(c.convByDefine[c.currentDefine], conv)
}

// record remembers an expression's type and hands it back, so that inference
// rules can end in `return c.record(expr, t)`.
func (c *checker) record(expr ast.Expression, t Type) Type {
	if t == nil {
		t = Unknown
	}
	c.types[expr] = t
	return t
}

// --- type specifiers ---

// resolveTypeSpecifier turns what an author wrote into a type.
func (c *checker) resolveTypeSpecifier(spec ast.TypeSpecifier) Type {
	switch s := spec.(type) {
	case nil:
		return Unknown
	case *ast.NamedType:
		return c.resolveNamedType(s)
	case *ast.ListType:
		return &List{Element: c.resolveTypeSpecifier(s.ElementType)}
	case *ast.IntervalType:
		return &Interval{Point: c.resolveTypeSpecifier(s.PointType)}
	case *ast.TupleType:
		elements := make([]TupleElement, 0, len(s.Elements))
		for _, e := range s.Elements {
			elements = append(elements, TupleElement{Name: e.Name, Type: c.resolveTypeSpecifier(e.Type)})
		}
		return NewTuple(elements)
	case *ast.ChoiceType:
		branches := make([]Type, 0, len(s.Types))
		for _, t := range s.Types {
			branches = append(branches, c.resolveTypeSpecifier(t))
		}
		return NewChoice(branches)
	}
	return Unknown
}

// resolveNamedType decides what model an unqualified type name belongs to.
//
// The data model wins where it declares the name, and CQL's own types answer
// for everything else. The two collide in exactly two places — FHIR declares a
// Quantity and a Ratio, and so does CQL — and there the model is what an author
// means: `value as Quantity` inside FHIRHelpers narrows a FHIR choice element,
// and the same file writes `System.Quantity` when it means the other one.
//
// Nothing else collides, because FHIR spells its primitives in lower case:
// `Integer` is System.Integer and FHIR.integer is a different type, spelled
// differently. Which is why this asks the model rather than guessing from a
// list of names.
// resolveNamedType turns a written type specifier into a type, spelled the same
// way every other route spells it.
//
// There is one canonical split and it is the model qualifier from the rest:
// FHIR.Encounter.Hospitalization is model FHIR and name
// Encounter.Hospitalization, because FHIR names a backbone element after the
// type that owns it. The parser reports the last segment as the name and
// everything before it as the namespace, which for a nested type puts the dot in
// the wrong place — model "FHIR.Encounter", name "Hospitalization". That prints
// identically to what ParseTypeName produces from the model document and
// compares as a different type, so `define function "H"(h
// FHIR.Encounter.Hospitalization)` rejected an argument of exactly that type.
func (c *checker) resolveNamedType(n *ast.NamedType) Type {
	if n == nil || n.Name == "" {
		return Unknown
	}
	if n.Namespace != "" {
		// Re-join, then split once at the qualifier — which is only there when
		// the first segment names a model. `Encounter.Hospitalization` is a
		// nested type of the model in force, not a type called Hospitalization
		// in a model called Encounter.
		full := n.Namespace + "." + n.Name
		if first, _, _ := strings.Cut(full, "."); first == c.defaultModel() || first == systemModel {
			return ParseTypeName(full)
		}
		return &Named{Model: c.defaultModel(), Name: full}
	}
	model := c.defaultModel()
	if model != systemModel && c.model != nil && c.model.HasType(n.Name) {
		return &Named{Model: model, Name: n.Name}
	}
	if isSystemTypeName(n.Name) {
		return &Named{Model: systemModel, Name: n.Name}
	}
	return &Named{Model: model, Name: n.Name}
}

// systemTypeNames are the type names CQL defines itself.
var systemTypeNames = map[string]bool{
	"Any": true, "Boolean": true, "Integer": true, "Long": true,
	"Decimal": true, "String": true, "Date": true, "DateTime": true,
	"Time": true, "Quantity": true, "Ratio": true, "Code": true,
	"Concept": true, "Vocabulary": true, "ValueSet": true, "CodeSystem": true,
}

func isSystemTypeName(name string) bool { return systemTypeNames[name] }
