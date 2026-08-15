// Package eval implements the CQL expression evaluator.
package eval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/model"
	"github.com/gofhir/cql/sema"
	cqltypes "github.com/gofhir/cql/types"
)

// RetrieveRequest describes what a CQL retrieve is asking for.
//
// It is a struct rather than a parameter list so that what a retrieve knows can
// grow without breaking every implementation again: Subject and Limit are both
// things the engine had and could not pass on.
type RetrieveRequest struct {
	// ResourceType is the type named in the retrieve, e.g. "Condition".
	ResourceType string

	// CodePath is the element the codes filter against. When the retrieve does
	// not name one, it comes from the model's primary code path — "code" for
	// Condition, "type" for Encounter, "vaccineCode" for Immunization.
	CodePath string

	// CodeComparator is the operator the retrieve used, e.g. "in".
	CodeComparator string

	// Codes is the value set URL, or the code values, to filter by. Nil when
	// the retrieve is unfiltered.
	Codes interface{}

	// DateRange narrows the retrieve to a period, when the query pushed one down.
	DateRange interface{}

	// Context names the CQL context in force, e.g. "Patient", and ContextID the
	// subject's identifier. ContextSearchParam is the model's search parameter
	// relating this resource type to that context — "subject" for Observation,
	// "patient" for Condition.
	//
	// Honoring these is the provider's job, and it is not optional: CQL scopes
	// a retrieve to the context in force, so a provider that ignores them
	// returns other patients' data and every measure built on it is wrong. The
	// engine cannot do it instead — the model names a search parameter, and
	// Condition has no element called patient.
	Context            string
	ContextID          string
	ContextSearchParam string

	// Limit is the most resources the caller will accept, from
	// WithMaxRetrieveSize. Zero means unbounded. A provider that can bound the
	// query should; the engine refuses the result if more come back, which
	// costs the provider the work of producing them.
	Limit int
}

// DataProvider retrieves FHIR resources for CQL retrieve expressions.
type DataProvider interface {
	Retrieve(ctx context.Context, req RetrieveRequest) ([]json.RawMessage, error)
}

// LibraryLoader resolves included libraries by name and version on demand.
type LibraryLoader interface {
	LoadLibrary(ctx context.Context, name, version string) (*ast.Library, error)
}

// TerminologyProvider checks code membership in value sets.
type TerminologyProvider interface {
	InValueSet(ctx context.Context, code, system, valueSetURL string) (bool, error)
}

// Sentinel errors for the bounds an evaluation is held to. They are values so a
// caller can tell a resource limit from an ordinary evaluation error with
// errors.Is; the engine maps both to ErrTooCostly.
var (
	// ErrMaxDepthExceeded reports that an expression nested past MaxDepth.
	ErrMaxDepthExceeded = errors.New("maximum expression depth exceeded")
	// ErrMaxRetrieveSizeExceeded reports that a retrieve returned more
	// resources than MaxRetrieveSize allows.
	ErrMaxRetrieveSizeExceeded = errors.New("maximum retrieve size exceeded")
)

// Context holds the evaluation state for a CQL evaluation.
// conversionKey names one resolved conversion: which function of which library,
// applied to which type of value.
//
// The library is the library itself and not the name it was reached by. Two
// libraries in one evaluation can each include a different library under the
// same alias — FHIRHelpers is the one everybody aliases — and keying on the
// name let whichever converted first decide the function body the other ran.
type conversionKey struct {
	library  *ast.Library
	function string
	argType  string
}

type Context struct {
	// Go context for cancellation/timeout
	GoCtx context.Context

	// The CQL library being evaluated
	Library *ast.Library

	// Current context (e.g. "Patient") — the focus resource
	ContextValue json.RawMessage

	// Resolved expression definitions (name → result)
	Definitions map[string]fptypes.Value

	// Parameters passed to the evaluation
	Parameters map[string]fptypes.Value

	// Resolved code systems and value sets
	CodeSystems map[string]*cqltypes.Code
	ValueSets   map[string]string // name → URL

	// Aliases from query sources
	Aliases map[string]fptypes.Value

	// Let bindings from queries
	LetBindings map[string]fptypes.Value

	// $this, $index, $total for iteration
	This  fptypes.Value
	Index int
	Total fptypes.Value

	// StatementContext is the CQL context the statement being evaluated declared.
	// A library may switch context part-way — `context Patient` for some
	// definitions and `context Unfiltered` for others — so this follows the
	// statement rather than the library's first declaration.
	StatementContext string

	// InSortKey marks a scope in which a sort key is being evaluated against a
	// result element, which is This. An unqualified identifier then names a
	// column of that element, and one that names no column is null rather than
	// an error — an optional FHIR element is absent from some resources and
	// present on others, and a sort must not fail on the ones that lack it.
	// Whether the key is a typo is decided once against the whole result set,
	// by sortKeyIsTypo, rather than per element.
	//
	// It is a flag rather than the element itself because the element can
	// legitimately be null, and it is deliberately not carried over by
	// ChildScope: it governs the key expression, not a nested query inside one.
	InSortKey bool

	// EvaluationTimestamp is the instant the evaluation request was made, and is
	// what Now, TimeOfDay and Today answer with.
	//
	// CQL requires them to be consistent throughout one evaluation — they name
	// the request's timestamp, not the clock's current reading — so reading the
	// clock per call is a conformance bug, not just a reproducibility one:
	// `Now() = Now()` came out false about once in every 1,700 evaluations.
	// Freezing it also makes a measure repeatable, which is what lets the same
	// data be re-run and give the same answer.
	EvaluationTimestamp time.Time

	// MaxDepth bounds how deeply Eval may nest before giving up, and
	// MaxRetrieveSize how many resources a single retrieve may return. Zero
	// means unbounded for either.
	MaxDepth        int
	MaxRetrieveSize int

	// depth counts the Eval calls currently on the stack. It is a pointer so
	// that every scope of one evaluation shares the same counter: ChildScope
	// hands out new contexts, but the recursion they measure is the same.
	depth *int

	// evalTicks counts Eval calls since the last cancellation check.
	evalTicks *int

	// funcRegistries memoizes each library's name → overloads map, keyed by the
	// library itself. Rebuilding it is what NewEvaluator used to do on every
	// user function call.
	funcRegistries map[*ast.Library]map[string][]*ast.FunctionDef

	// includedRegistries memoizes the alias → name → overloads map for the
	// includes of each library. Keyed by library because a library's aliases are
	// fixed by its own include list, so every scope of one library shares them.
	includedRegistries map[*ast.Library]map[string]map[string][]*ast.FunctionDef

	// libraryScopes memoizes the scope built for each included library, keyed by
	// the library itself. It serves two purposes: a definition of an included
	// library is evaluated once per evaluation rather than once per reference —
	// three mentions of H.Obs must not issue three retrieves — and the six maps
	// a library scope carries are built once rather than on every call into it.
	libraryScopes map[*ast.Library]*Context

	// External providers
	DataProvider        DataProvider
	TerminologyProvider TerminologyProvider
	QuantityConverter   QuantityConverter

	// TraceListener receives events during evaluation (optional, nil = no tracing).
	TraceListener TraceListener

	// ModelInfo provides FHIR type metadata for choice type resolution.
	ModelInfo model.ModelInfo

	// conversionOverloads remembers which overload of a conversion function
	// takes which argument type, for the length of one evaluation. See
	// conversionOverload.
	conversionOverloads map[conversionKey]*ast.FunctionDef

	// Plan is what the semantic phase decided about the library being
	// evaluated: which expressions have to be converted before the context
	// around them can use them.
	//
	// It is consulted per node, so a plan belonging to a different library
	// simply never matches and the evaluator falls back to deciding for itself
	// — which is what happens inside an included library, whose AST the phase
	// is not given.
	Plan *sema.Result

	// Context type and resource type for multi-context support
	contextType         ContextType
	contextResourceType string

	// Cached parsed context value (avoids repeated JSON unmarshal)
	cachedSubjectID string
	cachedSubjectOK bool // true once cachedSubjectID has been computed
	cachedObject    *fptypes.ObjectValue

	// IncludedLibraries maps alias → compiled included library. Aliases are
	// local to the library that wrote them, so this map is rebuilt per library
	// scope rather than inherited.
	IncludedLibraries map[string]*ast.Library

	// LoadedLibraries indexes every library compiled for this evaluation by
	// name and by name/version. Unlike aliases these are global and unambiguous,
	// which is what lets a library's own include list be resolved against the
	// set already loaded.
	LoadedLibraries map[string]*ast.Library

	// LibraryLoader resolves included libraries lazily on demand (optional).
	LibraryLoader LibraryLoader
	// loadingLibs tracks libraries currently being loaded to detect circular deps.
	loadingLibs map[string]bool

	// Parent context (for nested scopes)
	parent *Context
}

// NewContext creates a root evaluation context.
func NewContext(goCtx context.Context, lib *ast.Library) *Context {
	if goCtx == nil {
		goCtx = context.Background()
	}
	c := &Context{
		GoCtx:               goCtx,
		Library:             lib,
		EvaluationTimestamp: time.Now(),
		depth:               new(int),
		evalTicks:           new(int),
		libraryScopes:       make(map[*ast.Library]*Context),
		funcRegistries:      make(map[*ast.Library]map[string][]*ast.FunctionDef),
		includedRegistries:  make(map[*ast.Library]map[string]map[string][]*ast.FunctionDef),
		Definitions:         make(map[string]fptypes.Value),
		Parameters:          make(map[string]fptypes.Value),
		CodeSystems:         make(map[string]*cqltypes.Code),
		ValueSets:           make(map[string]string),
		Aliases:             make(map[string]fptypes.Value),
		LetBindings:         make(map[string]fptypes.Value),
		IncludedLibraries:   make(map[string]*ast.Library),
		LoadedLibraries:     make(map[string]*ast.Library),
		conversionOverloads: make(map[conversionKey]*ast.FunctionDef),
	}
	c.loadDeclarations(lib)
	return c
}

// loadDeclarations populates the terminology a library declares. It is separate
// from NewContext because an included library's own code has to run against its
// own declarations, not the including library's.
func (c *Context) loadDeclarations(lib *ast.Library) {
	if lib == nil {
		return
	}
	for _, cs := range lib.CodeSystems {
		c.CodeSystems[cs.Name] = &cqltypes.Code{System: cs.ID}
	}
	for _, vs := range lib.ValueSets {
		c.ValueSets[vs.Name] = vs.ID
	}
	// Declared codes and concepts are values, so they belong with the other
	// definitions: `code "SBP": '8480-6' from "LOINC"` makes "SBP" a Code.
	// Without this they resolved to nothing, and the identifier fallback
	// answered with the name as a String — so `O.code ~ "SBP"` compared a
	// Code against the text "SBP" and was quietly always false.
	for _, cd := range lib.Codes {
		system := cd.System
		if cs, ok := c.CodeSystems[cd.System]; ok {
			system = cs.System
		}
		c.Definitions[cd.Name] = cqltypes.NewCode(system, cd.ID, cd.Display)
	}
	for _, cd := range lib.Concepts {
		codes := make([]cqltypes.Code, 0, len(cd.Codes))
		for _, name := range cd.Codes {
			if v, ok := c.Definitions[name].(cqltypes.Code); ok {
				codes = append(codes, v)
			}
		}
		c.Definitions[cd.Name] = cqltypes.NewConcept(codes, cd.Display)
	}
}

// functionRegistry returns a library's name → overloads map, building it once.
func (c *Context) functionRegistry(lib *ast.Library) map[string][]*ast.FunctionDef {
	if lib == nil {
		return map[string][]*ast.FunctionDef{}
	}
	if c.funcRegistries != nil {
		if reg, ok := c.funcRegistries[lib]; ok {
			return reg
		}
	}
	reg := make(map[string][]*ast.FunctionDef, len(lib.Functions))
	for _, f := range lib.Functions {
		reg[f.Name] = append(reg[f.Name], f)
	}
	if c.funcRegistries != nil {
		c.funcRegistries[lib] = reg
	}
	return reg
}

// includedFunctionRegistry returns the alias → name → overloads map for this
// scope's includes, building it once per library.
//
// The result is mutable on purpose: ensureLibraryLoaded adds an alias to it when
// a library is resolved on demand, and every scope of that library should see
// the library it just loaded.
func (c *Context) includedFunctionRegistry() map[string]map[string][]*ast.FunctionDef {
	if c.includedRegistries != nil && c.Library != nil {
		if reg, ok := c.includedRegistries[c.Library]; ok {
			return reg
		}
	}
	reg := make(map[string]map[string][]*ast.FunctionDef, len(c.IncludedLibraries))
	for alias, lib := range c.IncludedLibraries {
		if lib == nil {
			continue
		}
		reg[alias] = c.functionRegistry(lib)
	}
	if c.includedRegistries != nil && c.Library != nil {
		c.includedRegistries[c.Library] = reg
	}
	return reg
}

// hasInclude reports whether the library declares an include under this alias.
func (c *Context) hasInclude(alias string) bool {
	if c.Library == nil {
		return false
	}
	for _, inc := range c.Library.Includes {
		name := inc.Alias
		if name == "" {
			name = inc.Name
		}
		if name == alias {
			return true
		}
	}
	return false
}

// includesOf builds the alias→library map for a library's own includes,
// resolving each against the libraries already loaded in this evaluation.
//
// An alias means whatever the library that wrote it says it means, so this
// cannot be inherited from the caller: two libraries commonly pick the same
// short alias for different targets.
func (c *Context) includesOf(lib *ast.Library) map[string]*ast.Library {
	if lib == nil {
		return make(map[string]*ast.Library)
	}
	out := make(map[string]*ast.Library, len(lib.Includes))
	known := c.loadedByName()
	for _, inc := range lib.Includes {
		alias := inc.Alias
		if alias == "" {
			alias = inc.Name
		}
		if target, ok := known[inc.Name+"/"+inc.Version]; ok {
			out[alias] = target
		} else if target, ok := known[inc.Name]; ok {
			out[alias] = target
		}
	}
	return out
}

// loadedByName indexes every library loaded in this evaluation under its own
// declared name, and under name/version when it declares one. Aliases are local
// to whoever wrote them; the library's name is not.
func (c *Context) loadedByName() map[string]*ast.Library {
	if c.LoadedLibraries != nil {
		return c.LoadedLibraries
	}
	return make(map[string]*ast.Library)
}

// RegisterLoadedLibrary indexes a compiled library under its declared name, so
// that any library including it can find it whatever alias it chose.
func (c *Context) RegisterLoadedLibrary(name, version string, lib *ast.Library) {
	if lib == nil || c.LoadedLibraries == nil {
		return
	}
	if _, seen := c.LoadedLibraries[name]; !seen {
		c.LoadedLibraries[name] = lib
	}
	if version != "" {
		c.LoadedLibraries[name+"/"+version] = lib
	}
}

// LibraryScope builds the scope an included library's own code runs in: its
// functions, its definitions and its terminology rather than the including
// library's. The definition cache and terminology tables are fresh, because two
// libraries may name a define alike and those maps are otherwise shared by every
// scope of one evaluation.
func (c *Context) LibraryScope(lib *ast.Library) *Context {
	scope := c.ChildScope()
	scope.Library = lib
	scope.Definitions = make(map[string]fptypes.Value)
	scope.CodeSystems = make(map[string]*cqltypes.Code)
	scope.ValueSets = make(map[string]string)
	// Parameters are per-library in CQL, so the callee must not read a value the
	// caller was given — nor write its own defaults back into the caller's map.
	scope.Parameters = make(map[string]fptypes.Value)
	// Includes are per-library too, and this is the one that bites: aliases are
	// short and collide. With the caller's map in place, a callee's `C.Answer`
	// resolved against whatever the *caller* had called C.
	scope.IncludedLibraries = c.includesOf(lib)
	// No parent: the callee must not see the caller's aliases or definitions.
	// Crossing that line is what would let an included library resolve a name it
	// never declared, which is worse than failing to find one.
	scope.parent = nil
	scope.loadDeclarations(lib)
	return scope
}

// ChildScope creates a nested scope inheriting parent lookups.
func (c *Context) ChildScope() *Context {
	return &Context{
		GoCtx:               c.GoCtx,
		Library:             c.Library,
		ContextValue:        c.ContextValue,
		Definitions:         c.Definitions,
		Parameters:          c.Parameters,
		CodeSystems:         c.CodeSystems,
		ValueSets:           c.ValueSets,
		Aliases:             make(map[string]fptypes.Value),
		LetBindings:         make(map[string]fptypes.Value),
		MaxDepth:            c.MaxDepth,
		MaxRetrieveSize:     c.MaxRetrieveSize,
		EvaluationTimestamp: c.EvaluationTimestamp,
		StatementContext:    c.StatementContext,
		depth:               c.depth,
		evalTicks:           c.evalTicks,
		libraryScopes:       c.libraryScopes,
		funcRegistries:      c.funcRegistries,
		includedRegistries:  c.includedRegistries,
		DataProvider:        c.DataProvider,
		TerminologyProvider: c.TerminologyProvider,
		QuantityConverter:   c.QuantityConverter,
		TraceListener:       c.TraceListener,
		ModelInfo:           c.ModelInfo,
		Plan:                c.Plan,
		conversionOverloads: c.conversionOverloads,
		contextType:         c.contextType,
		contextResourceType: c.contextResourceType,
		cachedSubjectID:     c.cachedSubjectID,
		cachedSubjectOK:     c.cachedSubjectOK,
		cachedObject:        c.cachedObject,
		IncludedLibraries:   c.IncludedLibraries,
		LoadedLibraries:     c.LoadedLibraries,
		LibraryLoader:       c.LibraryLoader,
		loadingLibs:         c.loadingLibs,
		parent:              c,
	}
}

// ResolveIdentifier looks up a name in order: aliases, let bindings, definitions, parameters.
func (c *Context) ResolveIdentifier(name string) (fptypes.Value, bool) {
	if v, ok := c.Aliases[name]; ok {
		return v, true
	}
	if v, ok := c.LetBindings[name]; ok {
		return v, true
	}
	if v, ok := c.Definitions[name]; ok {
		return v, true
	}
	if v, ok := c.Parameters[name]; ok {
		return v, true
	}
	if c.parent != nil {
		return c.parent.ResolveIdentifier(name)
	}
	return nil, false
}

// checkCanceled reports the Go context's error, if any, so that a canceled or
// timed-out evaluation stops where it is rather than running to completion and
// having the caller notice afterwards.
func (c *Context) checkCanceled() error {
	if c.GoCtx == nil {
		return nil
	}
	select {
	case <-c.GoCtx.Done():
		return fmt.Errorf("evaluation stopped: %w", c.GoCtx.Err())
	default:
		return nil
	}
}

// ResolveValueSetURL looks up a value set name and returns its URL.
func (c *Context) ResolveValueSetURL(name string) (string, bool) {
	if url, ok := c.ValueSets[name]; ok {
		return url, true
	}
	if c.parent != nil {
		return c.parent.ResolveValueSetURL(name)
	}
	return "", false
}

// GetContextObject returns a cached ObjectValue for the context resource.
// This avoids repeated JSON parsing when accessing multiple fields.
func (c *Context) GetContextObject() *fptypes.ObjectValue {
	if c.cachedObject != nil {
		return c.cachedObject
	}
	if len(c.ContextValue) == 0 {
		return nil
	}
	c.cachedObject = fptypes.NewObjectValue([]byte(c.ContextValue))
	return c.cachedObject
}

// contextScoper is the part of a model that knows how a resource type relates
// to a context. It is asked for optionally so a hand-built model still works.
type contextScoper interface {
	ContextSearchParam(resourceType, contextName string) (string, bool)
}

// applyRetrieveContext fills in the context a retrieve is scoped to.
//
// CQL evaluates a retrieve within the context in force, so `[Condition]` under
// `context Patient` means that patient's conditions and no one else's. The
// engine cannot enforce it: the model relates a type to a context by search
// parameter, and Condition has no element called patient. What it can do is say
// which patient it means, clearly enough that a provider cannot mistake it.
func (c *Context) applyRetrieveContext(req *RetrieveRequest) {
	contextName := c.StatementContext
	if contextName == "" {
		return
	}
	// Without a subject there is nothing to scope to, and saying "Patient" with
	// an empty id would ask a conforming provider to filter on nothing. A
	// population-level run is exactly that case.
	subject := c.GetContextSubjectID()
	if subject == "" {
		return
	}
	req.Context = contextName
	req.ContextID = subject
	if scoper, ok := c.ModelInfo.(contextScoper); ok {
		if param, found := scoper.ContextSearchParam(req.ResourceType, contextName); found {
			req.ContextSearchParam = param
		}
	}
}
