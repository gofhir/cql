// Package eval implements the CQL expression evaluator.
package eval

import (
	"context"
	"encoding/json"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/model"
	cqltypes "github.com/gofhir/cql/types"
)

// DataProvider retrieves FHIR resources for CQL retrieve expressions.
type DataProvider interface {
	Retrieve(ctx context.Context, resourceType string, codePath string, codeComparator string, codes interface{}, dateRange interface{}) ([]json.RawMessage, error)
}

// TerminologyProvider checks code membership in value sets.
type TerminologyProvider interface {
	InValueSet(ctx context.Context, code, system, valueSetURL string) (bool, error)
}

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

	// External providers
	DataProvider        DataProvider
	TerminologyProvider TerminologyProvider

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

	// Parent context (for nested scopes)
	parent *Context
}

// NewContext creates a root evaluation context.
func NewContext(goCtx context.Context, lib *ast.Library) *Context {
	if goCtx == nil {
		goCtx = context.Background()
	}
	c := &Context{
		GoCtx:       goCtx,
		Library:     lib,
		Definitions: make(map[string]fptypes.Value),
		Parameters:  make(map[string]fptypes.Value),
		CodeSystems: make(map[string]*cqltypes.Code),
		ValueSets:   make(map[string]string),
		Aliases:     make(map[string]fptypes.Value),
		LetBindings: make(map[string]fptypes.Value),
	}
	// Populate code systems and value sets from library definitions
	if lib != nil {
		for _, cs := range lib.CodeSystems {
			c.CodeSystems[cs.Name] = &cqltypes.Code{System: cs.ID}
		}
		for _, vs := range lib.ValueSets {
			c.ValueSets[vs.Name] = vs.ID
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
		DataProvider:        c.DataProvider,
		TerminologyProvider: c.TerminologyProvider,
		TraceListener:       c.TraceListener,
		ModelInfo:           c.ModelInfo,
		contextType:         c.contextType,
		contextResourceType: c.contextResourceType,
		cachedSubjectID:     c.cachedSubjectID,
		cachedSubjectOK:     c.cachedSubjectOK,
		cachedObject:        c.cachedObject,
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
