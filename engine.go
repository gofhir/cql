// Package cql provides a native CQL (Clinical Quality Language) engine for Go.
//
// The engine parses CQL text into an AST, evaluates expressions, and supports
// FHIR data retrieval through pluggable data and terminology providers.
//
// Basic usage:
//
//	engine := cql.NewEngine(
//	    cql.WithDataProvider(myDataProvider),
//	    cql.WithTimeout(30 * time.Second),
//	)
//	results, err := engine.EvaluateLibrary(ctx, cqlSource, patientJSON, params)
package cql

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/compiler"
	"github.com/gofhir/cql/eval"
	"github.com/gofhir/cql/fhirhelpers"
	"github.com/gofhir/cql/model"
)

// LibraryResolver loads CQL source by library name and version.
type LibraryResolver func(ctx context.Context, name, version string) (string, error)

// Engine is the public API for the CQL engine.
type Engine struct {
	dataProvider        eval.DataProvider
	terminologyProvider eval.TerminologyProvider
	modelInfo           model.ModelInfo
	traceListener       eval.TraceListener
	libraryResolver     LibraryResolver
	maxExpressionLen    int
	evalTimeout         time.Duration
	maxRetrieveSize     int
	maxDepth            int
	compiledCache       sync.Map // hash(cqlSource) → *ast.Library
}

// Option configures the Engine.
type Option func(*Engine)

// WithDataProvider sets the data provider for retrieve expressions.
func WithDataProvider(dp eval.DataProvider) Option {
	return func(e *Engine) {
		e.dataProvider = dp
	}
}

// WithTerminologyProvider sets the terminology provider for valueset checks.
func WithTerminologyProvider(tp eval.TerminologyProvider) Option {
	return func(e *Engine) {
		e.terminologyProvider = tp
	}
}

// WithModelInfo sets the FHIR model information.
func WithModelInfo(mi model.ModelInfo) Option {
	return func(e *Engine) {
		e.modelInfo = mi
	}
}

// WithMaxExpressionLen sets the maximum CQL source length (DoS protection).
func WithMaxExpressionLen(n int) Option {
	return func(e *Engine) {
		e.maxExpressionLen = n
	}
}

// WithTimeout sets the per-evaluation timeout.
func WithTimeout(d time.Duration) Option {
	return func(e *Engine) {
		e.evalTimeout = d
	}
}

// WithMaxRetrieveSize sets the maximum number of resources per retrieve.
func WithMaxRetrieveSize(n int) Option {
	return func(e *Engine) {
		e.maxRetrieveSize = n
	}
}

// WithMaxDepth sets the maximum recursion depth for nested expressions.
func WithMaxDepth(n int) Option {
	return func(e *Engine) {
		e.maxDepth = n
	}
}

// WithLibraryResolver sets the resolver for included libraries.
func WithLibraryResolver(lr LibraryResolver) Option {
	return func(e *Engine) {
		e.libraryResolver = lr
	}
}

// WithTraceListener sets a trace listener for expression-level debugging.
// When set, OnEnter/OnExit are called for every expression evaluation,
// allowing construction of trace trees for debugging and profiling.
func WithTraceListener(tl eval.TraceListener) Option {
	return func(e *Engine) {
		e.traceListener = tl
	}
}

// EvalOption configures a single evaluation call.
type EvalOption func(*evalConfig)

type evalConfig struct {
	traceListener eval.TraceListener
}

// WithCallTraceListener sets a trace listener for a specific call,
// overriding the engine-level trace listener for that call only.
// This enables per-request tracing in concurrent environments.
func WithCallTraceListener(tl eval.TraceListener) EvalOption {
	return func(c *evalConfig) { c.traceListener = tl }
}

// NewEngine creates a new CQL engine with the given options.
func NewEngine(opts ...Option) *Engine {
	e := &Engine{
		maxExpressionLen: 100 * 1024, // 100KB default
		evalTimeout:      30 * time.Second,
		maxRetrieveSize:  10000,
		maxDepth:         100,
	}
	for _, opt := range opts {
		opt(e)
	}
	if e.modelInfo == nil {
		e.modelInfo = model.DefaultR4ModelInfo()
	}
	return e
}

// compileOrCache compiles CQL source to AST, using a cache to avoid redundant ANTLR parsing.
func (e *Engine) compileOrCache(cqlSource string) (*ast.Library, error) {
	h := fnv.New64a()
	h.Write([]byte(cqlSource))
	key := h.Sum64()

	if cached, ok := e.compiledCache.Load(key); ok {
		return cached.(*ast.Library), nil
	}

	lib, err := compiler.Compile(cqlSource)
	if err != nil {
		return nil, err
	}
	e.compiledCache.Store(key, lib)
	return lib, nil
}

// resolveIncludes compiles and registers included libraries into the evaluation context.
func (e *Engine) resolveIncludes(ctx context.Context, lib *ast.Library, evalCtx *eval.Context) error {
	for _, inc := range lib.Includes {
		alias := inc.Alias
		if alias == "" {
			alias = inc.Name
		}

		// Try user-provided resolver first
		var src string
		var resolved bool
		if e.libraryResolver != nil {
			s, err := e.libraryResolver(ctx, inc.Name, inc.Version)
			if err == nil {
				src = s
				resolved = true
			}
		}

		// Fall back to built-in FHIRHelpers
		if !resolved && inc.Name == "FHIRHelpers" {
			src = fhirhelpers.Source
			resolved = true
		}

		if !resolved {
			return fmt.Errorf("library '%s' version '%s' could not be resolved (no LibraryResolver provided)", inc.Name, inc.Version)
		}

		incLib, err := e.compileOrCache(src)
		if err != nil {
			return fmt.Errorf("compiling library '%s': %w", inc.Name, err)
		}
		evalCtx.IncludedLibraries[alias] = incLib
	}
	return nil
}

// EvaluateLibrary parses and evaluates a CQL library, returning named expression results.
// Optional EvalOption arguments allow per-call configuration (e.g., WithCallTraceListener).
func (e *Engine) EvaluateLibrary(
	ctx context.Context,
	cqlSource string,
	contextResource json.RawMessage,
	params map[string]fptypes.Value,
	evalOpts ...EvalOption,
) (map[string]fptypes.Value, error) {
	// DoS protection: check source length
	if len(cqlSource) > e.maxExpressionLen {
		return nil, &ErrTooCostly{Msg: fmt.Sprintf("CQL source exceeds maximum length (%d > %d)", len(cqlSource), e.maxExpressionLen)}
	}

	// Apply timeout
	if e.evalTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.evalTimeout)
		defer cancel()
	}

	// Parse CQL source to AST (cached)
	lib, err := e.compileOrCache(cqlSource)
	if err != nil {
		return nil, &ErrSyntaxError{Cause: err}
	}

	// Build evaluation context
	evalCtx := eval.NewContext(ctx, lib)
	evalCtx.ContextValue = contextResource
	evalCtx.DataProvider = e.dataProvider
	evalCtx.TerminologyProvider = e.terminologyProvider
	evalCtx.TraceListener = e.traceListener
	evalCtx.ModelInfo = e.modelInfo
	// Apply per-call options (may override engine-level trace listener)
	var cfg evalConfig
	for _, opt := range evalOpts {
		opt(&cfg)
	}
	if cfg.traceListener != nil {
		evalCtx.TraceListener = cfg.traceListener
	}
	for k, v := range params {
		evalCtx.Parameters[k] = v
	}

	// Resolve included libraries
	if err = e.resolveIncludes(ctx, lib, evalCtx); err != nil {
		return nil, &ErrEvaluation{Cause: err}
	}

	// Evaluate all definitions
	evaluator := eval.NewEvaluator(evalCtx)
	results, err := evaluator.EvaluateLibrary()
	if err != nil {
		// Check for timeout
		if ctx.Err() == context.DeadlineExceeded {
			return nil, &ErrTimeout{Duration: e.evalTimeout}
		}
		return nil, &ErrEvaluation{Cause: err}
	}

	return results, nil
}

// EvaluateExpression parses CQL source and evaluates a single named expression.
// Optional EvalOption arguments allow per-call configuration (e.g., WithCallTraceListener).
func (e *Engine) EvaluateExpression(
	ctx context.Context,
	cqlSource string,
	expressionName string,
	contextResource json.RawMessage,
	params map[string]fptypes.Value,
	evalOpts ...EvalOption,
) (fptypes.Value, error) {
	// DoS protection
	if len(cqlSource) > e.maxExpressionLen {
		return nil, &ErrTooCostly{Msg: fmt.Sprintf("CQL source exceeds maximum length (%d > %d)", len(cqlSource), e.maxExpressionLen)}
	}

	// Apply timeout
	if e.evalTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.evalTimeout)
		defer cancel()
	}

	// Parse CQL source (cached)
	lib, err := e.compileOrCache(cqlSource)
	if err != nil {
		return nil, &ErrSyntaxError{Cause: err}
	}

	// Build evaluation context
	evalCtx := eval.NewContext(ctx, lib)
	evalCtx.ContextValue = contextResource
	evalCtx.DataProvider = e.dataProvider
	evalCtx.TerminologyProvider = e.terminologyProvider
	evalCtx.TraceListener = e.traceListener
	evalCtx.ModelInfo = e.modelInfo
	// Apply per-call options (may override engine-level trace listener)
	var cfg evalConfig
	for _, opt := range evalOpts {
		opt(&cfg)
	}
	if cfg.traceListener != nil {
		evalCtx.TraceListener = cfg.traceListener
	}
	for k, v := range params {
		evalCtx.Parameters[k] = v
	}

	// Resolve included libraries
	if err = e.resolveIncludes(ctx, lib, evalCtx); err != nil {
		return nil, &ErrEvaluation{Cause: err}
	}

	// Evaluate specified expression
	evaluator := eval.NewEvaluator(evalCtx)
	result, err := evaluator.EvaluateExpression(expressionName)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, &ErrTimeout{Duration: e.evalTimeout}
		}
		return nil, &ErrEvaluation{Cause: err}
	}

	return result, nil
}

// Compile parses a CQL source string without evaluating it.
// Useful for syntax validation and data requirements analysis.
func (e *Engine) Compile(cqlSource string) error {
	if len(cqlSource) > e.maxExpressionLen {
		return &ErrTooCostly{Msg: "CQL source exceeds maximum length"}
	}
	_, err := compiler.Compile(cqlSource)
	if err != nil {
		return &ErrSyntaxError{Cause: err}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Error types
// ---------------------------------------------------------------------------

// ErrSyntaxError indicates a CQL parse error (HTTP 400).
type ErrSyntaxError struct {
	Cause error
}

func (e *ErrSyntaxError) Error() string {
	return fmt.Sprintf("CQL syntax error: %v", e.Cause)
}

func (e *ErrSyntaxError) Unwrap() error {
	return e.Cause
}

// ErrEvaluation indicates a runtime evaluation error (HTTP 422).
type ErrEvaluation struct {
	Cause error
}

func (e *ErrEvaluation) Error() string {
	return fmt.Sprintf("CQL evaluation error: %v", e.Cause)
}

func (e *ErrEvaluation) Unwrap() error {
	return e.Cause
}

// ErrTimeout indicates the evaluation exceeded the configured timeout (HTTP 408).
type ErrTimeout struct {
	Duration time.Duration
}

func (e *ErrTimeout) Error() string {
	return fmt.Sprintf("CQL evaluation timed out after %v", e.Duration)
}

// ErrTooCostly indicates the evaluation is too expensive (HTTP 422).
type ErrTooCostly struct {
	Msg string
}

func (e *ErrTooCostly) Error() string {
	return fmt.Sprintf("CQL evaluation too costly: %s", e.Msg)
}
