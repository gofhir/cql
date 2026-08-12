// Package eval implements the CQL expression evaluator.
package eval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/model"
	cqltypes "github.com/gofhir/cql/types"
)

// DataProvider retrieves FHIR resources for CQL retrieve expressions.
type DataProvider interface {
	Retrieve(ctx context.Context, resourceType string, codePath string, codeComparator string, codes interface{}, dateRange interface{}) ([]json.RawMessage, error)
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

	// External providers
	DataProvider        DataProvider
	TerminologyProvider TerminologyProvider
	QuantityConverter   QuantityConverter

	// TraceListener receives events during evaluation (optional, nil = no tracing).
	TraceListener TraceListener

	// ModelInfo provides FHIR type metadata for choice type resolution.
	ModelInfo model.ModelInfo

	// Context type and resource type for multi-context support
	contextType         ContextType
	contextResourceType string

	// Cached parsed context value (avoids repeated JSON unmarshal)
	cachedSubjectID string
	cachedSubjectOK bool // true once cachedSubjectID has been computed
	cachedObject    *fptypes.ObjectValue

	// IncludedLibraries maps alias → compiled included library
	IncludedLibraries map[string]*ast.Library

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
		GoCtx:             goCtx,
		Library:           lib,
		depth:             new(int),
		evalTicks:         new(int),
		Definitions:       make(map[string]fptypes.Value),
		Parameters:        make(map[string]fptypes.Value),
		CodeSystems:       make(map[string]*cqltypes.Code),
		ValueSets:         make(map[string]string),
		Aliases:           make(map[string]fptypes.Value),
		LetBindings:       make(map[string]fptypes.Value),
		IncludedLibraries: make(map[string]*ast.Library),
	}
	// Populate code systems and value sets from library definitions
	if lib != nil {
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
	return c
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
		depth:               c.depth,
		evalTicks:           c.evalTicks,
		DataProvider:        c.DataProvider,
		TerminologyProvider: c.TerminologyProvider,
		QuantityConverter:   c.QuantityConverter,
		TraceListener:       c.TraceListener,
		ModelInfo:           c.ModelInfo,
		contextType:         c.contextType,
		contextResourceType: c.contextResourceType,
		cachedSubjectID:     c.cachedSubjectID,
		cachedSubjectOK:     c.cachedSubjectOK,
		cachedObject:        c.cachedObject,
		IncludedLibraries:   c.IncludedLibraries,
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
