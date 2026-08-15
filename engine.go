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
	"errors"
	"fmt"
	"hash/fnv"
	"maps"
	"strings"
	"sync"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/compiler"
	"github.com/gofhir/cql/eval"
	"github.com/gofhir/cql/fhirhelpers"
	"github.com/gofhir/cql/model"
	"github.com/gofhir/cql/sema"
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
	libraryLoader       eval.LibraryLoader
	quantityConverter   eval.QuantityConverter
	maxExpressionLen    int
	evalTimeout         time.Duration
	maxRetrieveSize     int
	maxDepth            int
	modelInfoExplicit   bool // caller supplied a model; do not second-guess it
	evaluationTimestamp time.Time
	compiledCache       sync.Map // hash(cqlSource) → cachedLibrary
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
// A retrieve that returns more fails with ErrTooCostly rather than being
// truncated: a measure computed over a silently shortened population is wrong
// without saying so. Zero disables the limit.
func WithMaxRetrieveSize(n int) Option {
	return func(e *Engine) {
		e.maxRetrieveSize = n
	}
}

// WithMaxDepth sets the maximum recursion depth for nested expressions.
// Zero disables the limit, which lets a self-referencing definition recurse
// until the process dies. Exceeding it yields ErrTooCostly.
func WithMaxDepth(n int) Option {
	return func(e *Engine) {
		e.maxDepth = n
	}
}

// WithEvaluationTimestamp fixes the instant Now, Today and TimeOfDay answer
// with, for every evaluation this engine performs.
//
// CQL defines them as the timestamp of the evaluation request, and the engine
// takes the clock's reading when the request arrives. Supplying it instead is
// what makes a measure re-runnable: the same data and the same timestamp give
// the same answer months later, which is the difference between a result you
// can re-derive and one you have to trust.
//
// The zero time means "read the clock", which is the default.
func WithEvaluationTimestamp(t time.Time) Option {
	return func(e *Engine) {
		e.evaluationTimestamp = t
	}
}

// WithLibraryResolver sets the resolver for included libraries.
func WithLibraryResolver(lr LibraryResolver) Option {
	return func(e *Engine) {
		e.libraryResolver = lr
	}
}

// WithLibraryLoader sets the library loader for lazy cross-library include resolution.
// When set, the evaluator will call the loader on demand when a library-qualified
// function call is encountered and the library has not been pre-resolved.
func WithLibraryLoader(loader eval.LibraryLoader) Option {
	return func(e *Engine) {
		e.libraryLoader = loader
	}
}

// WithQuantityConverter sets the quantity converter for UCUM unit conversions.
// When set, CQL ConvertQuantity and CanConvertQuantity functions become available.
func WithQuantityConverter(qc eval.QuantityConverter) Option {
	return func(e *Engine) {
		e.quantityConverter = qc
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
	traceListener       eval.TraceListener
	evaluationTimestamp time.Time
}

// WithCallTraceListener sets a trace listener for a specific call,
// overriding the engine-level trace listener for that call only.
// This enables per-request tracing in concurrent environments.
func WithCallTraceListener(tl eval.TraceListener) EvalOption {
	return func(c *evalConfig) { c.traceListener = tl }
}

// WithCallEvaluationTimestamp fixes the evaluation timestamp for one call,
// overriding the engine's. Re-running a measure as of a past date is a
// per-request question, not a per-engine one.
func WithCallEvaluationTimestamp(t time.Time) EvalOption {
	return func(c *evalConfig) { c.evaluationTimestamp = t }
}

// NewEngine creates a new CQL engine with the given options.
func NewEngine(opts ...Option) *Engine {
	e := &Engine{
		maxExpressionLen: 100 * 1024, // 100KB default
		evalTimeout:      30 * time.Second,
		maxRetrieveSize:  10000,
		// Measured, not guessed: a 100-term `or` chain and a chain of 50 defines
		// each reach depth ~100, and both are ordinary clinical CQL. The limit
		// exists to stop runaway recursion — `define A: A` crashed the process
		// with an unrecoverable stack overflow while this was unwired — so it
		// wants to sit far above real expressions and far below anything that
		// runs for long. Depth 100000 evaluates fine, so 10000 is neither.
		maxDepth: 10000,
	}
	for _, opt := range opts {
		opt(e)
	}
	if e.modelInfo == nil {
		// The official R4 model, which knows 147 retrievable types and their
		// code paths where the hand-built one knew ten. It is parsed once per
		// process; the hand-built model stays as the fallback so a broken
		// embed degrades rather than refusing to build an engine.
		if mi, err := model.LoadR4ModelInfo(); err == nil {
			e.modelInfo = mi
		} else {
			// Degrade rather than refuse to build an engine — and do not then
			// reject every library for asking for the version we just failed to
			// load, which is what checking the usings against the same sticky
			// failure would do.
			e.modelInfo = model.DefaultR4ModelInfo()
			e.modelInfoExplicit = true
		}
	} else {
		e.modelInfoExplicit = true
	}
	return e
}

// compileOrCache compiles CQL source to AST, using a cache to avoid redundant ANTLR parsing.
// compileOrCache parses CQL source to an AST, reusing the parse when the same
// source comes back.

// cachedLibrary is a compiled library together with the source it came from.
//
// The source is kept so a hit can be confirmed. Indexing by hash alone means a
// collision hands back a different library's AST — quietly, and with no way for
// the caller to tell. fnv64a over arbitrary CQL is not a cryptographic digest;
// two sources colliding is unlikely, not impossible, and the failure is silent
// and total.
type cachedLibrary struct {
	source string
	lib    *ast.Library
	plan   *sema.Result
}

func (e *Engine) compileOrCache(cqlSource string) (*ast.Library, *sema.Result, error) {
	h := fnv.New64a()
	h.Write([]byte(cqlSource))
	key := h.Sum64()

	if cached, ok := e.compiledCache.Load(key); ok {
		if entry, ok := cached.(cachedLibrary); ok && entry.source == cqlSource {
			return entry.lib, entry.plan, nil
		}
		// A collision: compile this source rather than answer with the other's
		// AST. The entry is left alone, so whichever source got there first
		// keeps the slot and the other pays the parse each time.
	}

	lib, err := compiler.Compile(cqlSource)
	if err != nil {
		return nil, nil, err
	}
	// The semantic phase runs with the parse and is cached with it. It costs
	// about a hundredth of what parsing does — 8.6µs against 975µs for a
	// measure-sized library — and what it decides is what the evaluator applies
	// instead of deciding for itself.
	plan := sema.Check(lib, e.semanticModel())
	e.compiledCache.LoadOrStore(key, cachedLibrary{source: cqlSource, lib: lib, plan: plan})
	return lib, plan, nil
}

// resolveIncludes compiles a library's includes, and then theirs, so that the
// whole graph is available before evaluation starts.
//
// Only the top-level library's own aliases are registered in IncludedLibraries.
// A transitive library is loaded but not named there: aliases belong to the
// library that wrote them, and registering a nested one under the same map is
// how `C.Answer()` in the top-level library came to resolve against a `C` that
// some included library had chosen. Each library's own alias map is rebuilt from
// LoadedLibraries when its scope is created.
func (e *Engine) resolveIncludes(ctx context.Context, lib *ast.Library, evalCtx *eval.Context) error {
	return e.resolveIncludesInto(ctx, lib, evalCtx, make(map[string]bool), true)
}

func (e *Engine) resolveIncludesInto(ctx context.Context, lib *ast.Library, evalCtx *eval.Context, seen map[string]bool, top bool) error {
	for _, inc := range lib.Includes {
		alias := inc.Alias
		if alias == "" {
			alias = inc.Name
		}
		// A library already compiled elsewhere in the graph still has to be named
		// under this library's alias when this is the top level; only the walk
		// into its own includes is skipped, and that is what stops a cycle.
		key := inc.Name + "/" + inc.Version
		if seen[key] {
			if top {
				if known, ok := evalCtx.LoadedLibraries[key]; ok {
					evalCtx.IncludedLibraries[alias] = known
				} else if known, ok := evalCtx.LoadedLibraries[inc.Name]; ok {
					evalCtx.IncludedLibraries[alias] = known
				}
			}
			continue
		}
		seen[key] = true

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

		incLib, _, err := e.compileOrCache(src)
		if err != nil {
			return fmt.Errorf("compiling library '%s': %w", inc.Name, err)
		}
		if top {
			evalCtx.IncludedLibraries[alias] = incLib
		}
		evalCtx.RegisterLoadedLibrary(inc.Name, inc.Version, incLib)
		if err := e.resolveIncludesInto(ctx, incLib, evalCtx, seen, false); err != nil {
			return err
		}
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
	parsed, err := e.Parse(cqlSource)
	if err != nil {
		return nil, err
	}
	return e.EvaluateParsedLibrary(ctx, parsed, contextResource, params, evalOpts...)
}

// EvaluateParsedLibrary evaluates every public definition of an already-parsed
// library, so the parse is paid once however many times it is evaluated.
func (e *Engine) EvaluateParsedLibrary(
	ctx context.Context,
	parsed *Library,
	contextResource json.RawMessage,
	params map[string]fptypes.Value,
	evalOpts ...EvalOption,
) (map[string]fptypes.Value, error) {
	if parsed == nil || parsed.lib == nil {
		return nil, &ErrEvaluation{Cause: fmt.Errorf("no library to evaluate")}
	}
	ctx, cancel := e.withTimeout(ctx)
	defer cancel()

	evalCtx, err := e.newEvalContext(ctx, parsed.lib, parsed.plan, contextResource, params, evalOpts)
	if err != nil {
		return nil, err
	}
	results, err := eval.NewEvaluator(evalCtx).EvaluateLibrary()
	if err != nil {
		return nil, e.classifyEvalError(ctx, err)
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
	parsed, err := e.Parse(cqlSource)
	if err != nil {
		return nil, err
	}
	return e.EvaluateParsedExpression(ctx, parsed, expressionName, contextResource, params, evalOpts...)
}

// EvaluateParsedExpression evaluates one named expression of an already-parsed
// library.
func (e *Engine) EvaluateParsedExpression(
	ctx context.Context,
	parsed *Library,
	expressionName string,
	contextResource json.RawMessage,
	params map[string]fptypes.Value,
	evalOpts ...EvalOption,
) (fptypes.Value, error) {
	if parsed == nil || parsed.lib == nil {
		return nil, &ErrEvaluation{Cause: fmt.Errorf("no library to evaluate")}
	}
	ctx, cancel := e.withTimeout(ctx)
	defer cancel()

	evalCtx, err := e.newEvalContext(ctx, parsed.lib, parsed.plan, contextResource, params, evalOpts)
	if err != nil {
		return nil, err
	}
	result, err := eval.NewEvaluator(evalCtx).EvaluateExpression(expressionName)
	if err != nil {
		return nil, e.classifyEvalError(ctx, err)
	}
	return result, nil
}

// newEvalContext builds the evaluation context for one request: the engine's
// providers and limits, then whatever the call overrides, then the parameters
// and the include graph. Both evaluation entry points go through it so they
// cannot drift apart — LibraryLoader was already only wired on one of them.
func (e *Engine) newEvalContext(
	ctx context.Context,
	lib *ast.Library,
	plan *sema.Result,
	contextResource json.RawMessage,
	params map[string]fptypes.Value,
	evalOpts []EvalOption,
) (*eval.Context, error) {
	if err := e.checkUsings(lib); err != nil {
		return nil, err
	}

	evalCtx := eval.NewContext(ctx, lib)
	evalCtx.ContextValue = contextResource
	evalCtx.DataProvider = e.dataProvider
	evalCtx.TerminologyProvider = e.terminologyProvider
	evalCtx.TraceListener = e.traceListener
	evalCtx.ModelInfo = e.modelInfo
	evalCtx.Plan = plan
	evalCtx.LibraryLoader = e.libraryLoader
	evalCtx.QuantityConverter = e.quantityConverter
	evalCtx.MaxDepth = e.maxDepth
	evalCtx.MaxRetrieveSize = e.maxRetrieveSize

	var cfg evalConfig
	for _, opt := range evalOpts {
		opt(&cfg)
	}
	if cfg.traceListener != nil {
		evalCtx.TraceListener = cfg.traceListener
	}
	if ts := e.evaluationTimestampFor(cfg); !ts.IsZero() {
		evalCtx.EvaluationTimestamp = ts
	}
	maps.Copy(evalCtx.Parameters, params)

	if err := e.resolveIncludes(ctx, lib, evalCtx); err != nil {
		return nil, &ErrEvaluation{Cause: err}
	}
	return evalCtx, nil
}

// evaluationTimestampFor resolves the instant an evaluation answers Now with:
// the call's if it supplied one, then the engine's, then nothing — which leaves
// the context's own reading of the clock in place.
func (e *Engine) evaluationTimestampFor(cfg evalConfig) time.Time {
	if !cfg.evaluationTimestamp.IsZero() {
		return cfg.evaluationTimestamp
	}
	return e.evaluationTimestamp
}

// checkUsings refuses a library that asks for a model this build cannot serve.
//
// `using FHIR version '5.0.0'` was accepted in silence and then evaluated
// against R4: every path and code path resolved against the wrong version of the
// spec, and nothing said so. A caller who supplied their own ModelInfo is taken
// at their word.
func (e *Engine) checkUsings(lib *ast.Library) error {
	if e.modelInfoExplicit || lib == nil {
		return nil
	}
	for _, u := range lib.Usings {
		if !strings.EqualFold(u.Name, "FHIR") {
			continue
		}
		if _, err := model.FHIRModelInfo(u.Version); err != nil {
			return &ErrEvaluation{Cause: err}
		}
	}
	return nil
}

// classifyEvalError maps an evaluation failure onto the engine's error taxonomy.
// A canceled context is checked first: the evaluator stops on cancellation and
// reports it as an ordinary error, so the reason has to be recovered from the
// context rather than from the error alone.
func (e *Engine) classifyEvalError(ctx context.Context, err error) error {
	if ctx.Err() == context.DeadlineExceeded {
		return &ErrTimeout{Duration: e.evalTimeout}
	}
	if errors.Is(err, eval.ErrMaxDepthExceeded) || errors.Is(err, eval.ErrMaxRetrieveSizeExceeded) {
		return &ErrTooCostly{Msg: err.Error()}
	}
	return &ErrEvaluation{Cause: err}
}

// withTimeout applies the engine's per-evaluation timeout.
func (e *Engine) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if e.evalTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, e.evalTimeout)
}

// Library is a parsed CQL library, ready to be evaluated as many times as
// wanted without being parsed again.
//
// Parsing dominates the first use of a source — a 300-term expression costs
// about 1.4ms to parse and 26µs to evaluate — so separating the two lets a
// caller pay it once, deliberately, rather than relying on a cache keyed by a
// hash of the text.
type Library struct {
	lib *ast.Library
	// plan is what the semantic phase decided about this library, carried
	// alongside it so that evaluating it repeatedly does not re-derive it.
	plan *sema.Result
}

// Name returns the library's declared name, or "" if it is anonymous.
func (l *Library) Name() string {
	if l == nil || l.lib == nil || l.lib.Identifier == nil {
		return ""
	}
	return l.lib.Identifier.Name
}

// Version returns the library's declared version, or "" if it declares none.
func (l *Library) Version() string {
	if l == nil || l.lib == nil || l.lib.Identifier == nil {
		return ""
	}
	return l.lib.Identifier.Version
}

// ExpressionNames returns the names a caller may evaluate, in declaration
// order. Private definitions are the library's own business and are not listed.
func (l *Library) ExpressionNames() []string {
	if l == nil || l.lib == nil {
		return nil
	}
	names := make([]string, 0, len(l.lib.Statements))
	for _, stmt := range l.lib.Statements {
		if stmt.AccessLevel == ast.AccessPrivate {
			continue
		}
		names = append(names, stmt.Name)
	}
	return names
}

// Parse compiles CQL source into a Library that can be evaluated repeatedly.
func (e *Engine) Parse(cqlSource string) (*Library, error) {
	if len(cqlSource) > e.maxExpressionLen {
		return nil, &ErrTooCostly{Msg: fmt.Sprintf("CQL source exceeds maximum length (%d > %d)", len(cqlSource), e.maxExpressionLen)}
	}
	lib, plan, err := e.compileOrCache(cqlSource)
	if err != nil {
		return nil, &ErrSyntaxError{Cause: err}
	}
	return &Library{lib: lib, plan: plan}, nil
}

// Compile parses a CQL source string without evaluating it.
// Useful for syntax validation and data requirements analysis.
//
// Deprecated: use Parse, which returns the parsed library rather than
// discarding it.
func (e *Engine) Compile(cqlSource string) error {
	_, err := e.Parse(cqlSource)
	return err
}

// Check parses a library and reports what the semantic phase finds in it: what
// type every expression has, and everything wrong with the ones that have none.
//
// It answers without data, without a provider and without evaluating anything,
// which is what makes it usable where a measure is authored rather than where
// it is run. A mistyped element name is found once, at the line that has it,
// instead of once per patient and only for patients whose data reaches it.
//
// The error return is for a library that does not parse; everything the
// semantic phase itself finds comes back as diagnostics, all of it in one pass.
// A caller that wants to refuse the library asks Diagnostics.HasErrors — a
// warning is a remark, not a verdict.
func (e *Engine) Check(cqlSource string) (sema.Diagnostics, error) {
	lib, err := e.Parse(cqlSource)
	if err != nil {
		return nil, err
	}
	return e.CheckParsed(lib), nil
}

// CheckParsed runs the semantic phase over an already parsed library, so a
// caller that has paid for the parse does not pay for it again.
func (e *Engine) CheckParsed(lib *Library) sema.Diagnostics {
	if lib == nil || lib.lib == nil {
		return nil
	}
	if lib.plan != nil {
		return lib.plan.Diagnostics
	}
	return sema.Check(lib.lib, e.semanticModel()).Diagnostics
}

// semanticModel is the model the semantic phase asks about types.
//
// A caller may have supplied a ModelInfo of their own making, which knows how
// to answer the questions the evaluator asks and not the ones this phase does —
// element types as types, declared conversions with their targets. Rather than
// widen the public ModelInfo interface for it, an implementation that cannot
// answer yields no model at all: the phase then types what needs no model and
// stays quiet about the rest, which is the honest answer.
func (e *Engine) semanticModel() sema.Model {
	static, ok := e.modelInfo.(*model.StaticModelInfo)
	if !ok {
		return nil
	}
	return sema.FromModelInfo(static)
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
