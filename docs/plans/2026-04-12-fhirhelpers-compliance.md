# FHIRHelpers Compliance Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable the Go CQL engine to evaluate standard clinical CQL that depends on FHIRHelpers, achieving behavioral parity with `@fhir-toolkit/cql`.

**Architecture:** Three gaps implemented sequentially: (1) Library resolver with function overloads and library-qualified dispatch, (2) `value[x]` choice type resolution in member access using existing ModelInfo infrastructure, (3) Built-in FHIRHelpers stub that works with raw JSON values (Option A from the issue). The parser already produces `FunctionCall{Source: IdentifierRef{"FHIRHelpers"}, Name: "ToQuantity"}` — we intercept this pattern in `evalFunctionCall` rather than modifying the grammar.

**Tech Stack:** Go 1.24, ANTLR4 (grammar unchanged), `github.com/gofhir/fhirpath` (ObjectValue, types)

---

## Task 1: Add ModelInfo to eval.Context

ModelInfo currently lives only in `Engine` but the evaluator needs it for choice type resolution (Gap 2) and future type-aware dispatch. Thread it through.

**Files:**
- Modify: `eval/context.go` (Context struct + NewContext + ChildScope)
- Modify: `engine.go` (EvaluateLibrary + EvaluateExpression — set ModelInfo on context)

**Step 1: Add ModelInfo field to Context**

In `eval/context.go`, add the import and field:

```go
import (
    "github.com/gofhir/cql/model"
)
```

Add to Context struct after `TraceListener`:

```go
    // ModelInfo provides FHIR type metadata for choice type resolution.
    ModelInfo model.ModelInfo
```

**Step 2: Propagate in ChildScope**

In `ChildScope()`, add to the returned struct:

```go
    ModelInfo:           c.ModelInfo,
```

**Step 3: Set ModelInfo in engine.go**

In `EvaluateLibrary` and `EvaluateExpression`, after `evalCtx := eval.NewContext(ctx, lib)`, add:

```go
    evalCtx.ModelInfo = e.modelInfo
```

**Step 4: Run tests**

Run: `go test ./... -count=1`
Expected: All existing tests pass (no behavioral change).

**Step 5: Commit**

```bash
git add eval/context.go engine.go
git commit -m "refactor: thread ModelInfo from Engine into eval.Context"
```

---

## Task 2: Support function overloads in Evaluator

FHIRHelpers defines 200+ `ToString` overloads. Change the funcs map from single-function-per-name to a slice of overloads.

**Files:**
- Modify: `eval/evaluator.go` (Evaluator struct, NewEvaluator, evalFunctionCall, evalUserFunction, withContext)

**Step 1: Write failing test**

In `eval/evaluator_test.go`, add:

```go
func TestEvalFunctionOverloads(t *testing.T) {
    lib := &ast.Library{
        Functions: []*ast.FunctionDef{
            {
                Name:     "Convert",
                Operands: []*ast.OperandDef{{Name: "val", TypeSpec: &ast.NamedTypeSpecifier{Name: "Integer"}}},
                Body:     &ast.Literal{ValueType: ast.LiteralString, Value: "from-integer"},
            },
            {
                Name:     "Convert",
                Operands: []*ast.OperandDef{{Name: "val", TypeSpec: &ast.NamedTypeSpecifier{Name: "String"}}},
                Body:     &ast.Literal{ValueType: ast.LiteralString, Value: "from-string"},
            },
        },
        Statements: []*ast.ExpressionDef{
            {
                Name: "ResultInt",
                Expression: &ast.FunctionCall{
                    Name:     "Convert",
                    Operands: []ast.Expression{&ast.Literal{ValueType: ast.LiteralInteger, Value: "42"}},
                },
            },
            {
                Name: "ResultStr",
                Expression: &ast.FunctionCall{
                    Name:     "Convert",
                    Operands: []ast.Expression{&ast.Literal{ValueType: ast.LiteralString, Value: "hello"}},
                },
            },
        },
    }

    ctx := NewContext(context.Background(), lib)
    evaluator := NewEvaluator(ctx)
    results, err := evaluator.EvaluateLibrary()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    ri, ok := results["ResultInt"].(fptypes.String)
    if !ok || ri.Value() != "from-integer" {
        t.Errorf("ResultInt = %v, want 'from-integer'", results["ResultInt"])
    }

    rs, ok := results["ResultStr"].(fptypes.String)
    if !ok || rs.Value() != "from-string" {
        t.Errorf("ResultStr = %v, want 'from-string'", results["ResultStr"])
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./eval/ -run TestEvalFunctionOverloads -v`
Expected: FAIL — second `Convert` overwrites first in `map[string]*ast.FunctionDef`.

**Step 3: Change funcs map to support overloads**

In `eval/evaluator.go`, change the Evaluator struct:

```go
type Evaluator struct {
    ctx           *Context
    funcs         map[string][]*ast.FunctionDef         // local overloads
    includedFuncs map[string]map[string][]*ast.FunctionDef // alias → name → overloads
}
```

Update `NewEvaluator`:

```go
func NewEvaluator(ctx *Context) *Evaluator {
    e := &Evaluator{
        ctx:           ctx,
        funcs:         make(map[string][]*ast.FunctionDef),
        includedFuncs: make(map[string]map[string][]*ast.FunctionDef),
    }
    if ctx.Library != nil {
        for _, f := range ctx.Library.Functions {
            e.funcs[f.Name] = append(e.funcs[f.Name], f)
        }
    }
    return e
}
```

Update `withContext`:

```go
func (e *Evaluator) withContext(ctx *Context) *Evaluator {
    return &Evaluator{ctx: ctx, funcs: e.funcs, includedFuncs: e.includedFuncs}
}
```

**Step 4: Add overload resolution**

Add a new function:

```go
// resolveOverload picks the best FunctionDef matching the given argument count.
// For now, match by operand count. Type-based scoring can be added later.
func resolveOverload(overloads []*ast.FunctionDef, args []ast.Expression) *ast.FunctionDef {
    if len(overloads) == 1 {
        return overloads[0]
    }
    for _, fd := range overloads {
        if len(fd.Operands) == len(args) {
            return fd
        }
    }
    // Fallback to first
    return overloads[0]
}
```

**Step 5: Update evalFunctionCall**

```go
func (e *Evaluator) evalFunctionCall(n *ast.FunctionCall) (fptypes.Value, error) {
    if overloads, ok := e.funcs[n.Name]; ok {
        fd := resolveOverload(overloads, n.Operands)
        return e.evalUserFunction(fd, n.Operands)
    }
    return e.evalBuiltinFunction(n)
}
```

**Step 6: Run tests**

Run: `go test ./eval/ -run TestEvalFunctionOverloads -v`
Expected: PASS

Run: `go test ./... -count=1`
Expected: All tests pass.

**Step 7: Commit**

```bash
git add eval/evaluator.go eval/evaluator_test.go
git commit -m "feat: support function overloads in evaluator"
```

---

## Task 3: Library-qualified function dispatch

When CQL has `FHIRHelpers.ToQuantity(x)`, the parser produces `FunctionCall{Source: IdentifierRef{Name: "FHIRHelpers"}, Name: "ToQuantity", Operands: [x]}`. The evaluator must detect this pattern and dispatch to the included library's functions.

**Files:**
- Modify: `eval/evaluator.go` (evalFunctionCall, NewEvaluator)
- Modify: `eval/context.go` (add IncludedLibraries field)
- Test: `eval/evaluator_test.go`

**Step 1: Write failing test**

In `eval/evaluator_test.go`:

```go
func TestEvalLibraryQualifiedFunctionCall(t *testing.T) {
    // Simulates: include MyLib called MyLib
    // define Result: MyLib.Double(21)
    includedLib := &ast.Library{
        Functions: []*ast.FunctionDef{
            {
                Name:     "Double",
                Operands: []*ast.OperandDef{{Name: "x"}},
                Body: &ast.BinaryExpression{
                    Operator: ast.OpMultiply,
                    Left:     &ast.IdentifierRef{Name: "x"},
                    Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
                },
            },
        },
    }

    mainLib := &ast.Library{
        Includes: []*ast.IncludeDef{
            {Name: "MyLib", Alias: "MyLib"},
        },
        Statements: []*ast.ExpressionDef{
            {
                Name: "Result",
                Expression: &ast.FunctionCall{
                    Source:   &ast.IdentifierRef{Name: "MyLib"},
                    Name:     "Double",
                    Operands: []ast.Expression{&ast.Literal{ValueType: ast.LiteralInteger, Value: "21"}},
                },
            },
        },
    }

    ctx := NewContext(context.Background(), mainLib)
    ctx.IncludedLibraries = map[string]*ast.Library{"MyLib": includedLib}
    evaluator := NewEvaluator(ctx)
    results, err := evaluator.EvaluateLibrary()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    val, ok := results["Result"].(fptypes.Integer)
    if !ok {
        t.Fatalf("Result: expected Integer, got %T (%v)", results["Result"], results["Result"])
    }
    if val.Value() != 42 {
        t.Errorf("Result = %d, want 42", val.Value())
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./eval/ -run TestEvalLibraryQualifiedFunctionCall -v`
Expected: FAIL — library-qualified dispatch not implemented.

**Step 3: Add IncludedLibraries to Context**

In `eval/context.go`, add to Context struct:

```go
    // IncludedLibraries maps alias → compiled included library
    IncludedLibraries map[string]*ast.Library
```

In `NewContext`, initialize it:

```go
    IncludedLibraries: make(map[string]*ast.Library),
```

In `ChildScope`, propagate it:

```go
    IncludedLibraries:   c.IncludedLibraries,
```

**Step 4: Register included library functions in NewEvaluator**

In `NewEvaluator`, after registering local functions:

```go
    // Register included library functions
    for alias, lib := range ctx.IncludedLibraries {
        libFuncs := make(map[string][]*ast.FunctionDef)
        for _, f := range lib.Functions {
            libFuncs[f.Name] = append(libFuncs[f.Name], f)
        }
        e.includedFuncs[alias] = libFuncs
    }
```

**Step 5: Update evalFunctionCall for library-qualified calls**

```go
func (e *Evaluator) evalFunctionCall(n *ast.FunctionCall) (fptypes.Value, error) {
    // Check for library-qualified call: FHIRHelpers.ToQuantity(...)
    // Parser produces: FunctionCall{Source: IdentifierRef{Name: "FHIRHelpers"}, Name: "ToQuantity"}
    if n.Source != nil {
        if idRef, ok := n.Source.(*ast.IdentifierRef); ok {
            if libFuncs, ok := e.includedFuncs[idRef.Name]; ok {
                overloads, ok := libFuncs[n.Name]
                if !ok {
                    return nil, fmt.Errorf("function '%s' not found in library '%s'", n.Name, idRef.Name)
                }
                fd := resolveOverload(overloads, n.Operands)
                return e.evalIncludedFunction(fd, n.Operands, idRef.Name)
            }
        }
    }

    // Local function dispatch (with overload support)
    if overloads, ok := e.funcs[n.Name]; ok {
        fd := resolveOverload(overloads, n.Operands)
        return e.evalUserFunction(fd, n.Operands)
    }
    return e.evalBuiltinFunction(n)
}
```

**Step 6: Add evalIncludedFunction**

This is like `evalUserFunction` but creates a child scope that can resolve identifiers from the included library's own definitions:

```go
func (e *Evaluator) evalIncludedFunction(fd *ast.FunctionDef, args []ast.Expression, libAlias string) (fptypes.Value, error) {
    if fd.External {
        return nil, fmt.Errorf("external function '%s' not implemented", fd.Name)
    }
    child := e.ctx.ChildScope()
    for i, op := range fd.Operands {
        if i < len(args) {
            val, err := e.Eval(args[i])
            if err != nil {
                return nil, err
            }
            child.Aliases[op.Name] = val
        }
    }
    childEval := NewEvaluator(child)
    // Ensure the child evaluator has the same included funcs
    childEval.includedFuncs = e.includedFuncs
    return childEval.Eval(fd.Body)
}
```

**Step 7: Run tests**

Run: `go test ./eval/ -run TestEvalLibraryQualifiedFunctionCall -v`
Expected: PASS

Run: `go test ./... -count=1`
Expected: All tests pass.

**Step 8: Commit**

```bash
git add eval/evaluator.go eval/evaluator_test.go eval/context.go
git commit -m "feat: library-qualified function dispatch for included libraries"
```

---

## Task 4: LibraryResolver option and include resolution in Engine

Wire up the actual library resolution: the Engine compiles included libraries and passes them to the evaluation context.

**Files:**
- Modify: `engine.go` (add LibraryResolver type, option, resolve logic)
- Test: `engine_test.go`

**Step 1: Write failing test**

In `engine_test.go`:

```go
func TestEngine_EvaluateLibrary_WithIncludedLibrary(t *testing.T) {
    mathLib := `library MathHelpers version '1.0'
using FHIR version '4.0.1'
define function Double(x Integer) returns Integer: x * 2`

    resolver := func(ctx context.Context, name, version string) (string, error) {
        if name == "MathHelpers" {
            return mathLib, nil
        }
        return "", fmt.Errorf("library '%s' not found", name)
    }

    e := NewEngine(WithLibraryResolver(resolver))

    cqlSource := `library Test version '1.0'
using FHIR version '4.0.1'
include MathHelpers version '1.0'
define Result: MathHelpers.Double(21)`

    results, err := e.EvaluateLibrary(context.Background(), cqlSource, nil, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    val, ok := results["Result"].(fptypes.Integer)
    if !ok {
        t.Fatalf("Result: expected Integer, got %T (%v)", results["Result"], results["Result"])
    }
    if val.Value() != 42 {
        t.Errorf("Result = %d, want 42", val.Value())
    }
}

func TestEngine_EvaluateLibrary_IncludeWithoutResolver(t *testing.T) {
    e := NewEngine()
    cqlSource := `library Test version '1.0'
using FHIR version '4.0.1'
include SomeLib version '1.0'
define X: 1`

    _, err := e.EvaluateLibrary(context.Background(), cqlSource, nil, nil)
    if err == nil {
        t.Fatal("expected error for missing resolver")
    }
}
```

**Step 2: Run tests to verify they fail**

Run: `go test . -run TestEngine_EvaluateLibrary_WithIncludedLibrary -v`
Expected: FAIL — WithLibraryResolver does not exist.

**Step 3: Add LibraryResolver to Engine**

In `engine.go`, add the type and option:

```go
// LibraryResolver loads CQL source by library name and version.
type LibraryResolver func(ctx context.Context, name, version string) (string, error)

// WithLibraryResolver sets the resolver for included libraries.
func WithLibraryResolver(lr LibraryResolver) Option {
    return func(e *Engine) {
        e.libraryResolver = lr
    }
}
```

Add field to Engine struct:

```go
    libraryResolver     LibraryResolver
```

**Step 4: Resolve includes in EvaluateLibrary**

In `engine.go`, add a helper method and call it from both `EvaluateLibrary` and `EvaluateExpression`, after building the eval context:

```go
// resolveIncludes compiles and registers included libraries.
func (e *Engine) resolveIncludes(ctx context.Context, lib *ast.Library, evalCtx *eval.Context) error {
    if len(lib.Includes) == 0 {
        return nil
    }
    if e.libraryResolver == nil {
        return fmt.Errorf("library '%s' is included but no LibraryResolver was provided", lib.Includes[0].Name)
    }
    for _, inc := range lib.Includes {
        src, err := e.libraryResolver(ctx, inc.Name, inc.Version)
        if err != nil {
            return fmt.Errorf("resolving library '%s' version '%s': %w", inc.Name, inc.Version, err)
        }
        incLib, err := e.compileOrCache(src)
        if err != nil {
            return fmt.Errorf("compiling library '%s': %w", inc.Name, err)
        }
        alias := inc.Alias
        if alias == "" {
            alias = inc.Name
        }
        evalCtx.IncludedLibraries[alias] = incLib
    }
    return nil
}
```

In `EvaluateLibrary`, after building evalCtx and before creating the evaluator:

```go
    // Resolve included libraries
    if err := e.resolveIncludes(ctx, lib, evalCtx); err != nil {
        return nil, &ErrEvaluation{Cause: err}
    }
```

Same in `EvaluateExpression`.

**Step 5: Run tests**

Run: `go test . -run TestEngine_EvaluateLibrary_WithIncludedLibrary -v`
Expected: PASS

Run: `go test . -run TestEngine_EvaluateLibrary_IncludeWithoutResolver -v`
Expected: PASS

Run: `go test ./... -count=1`
Expected: All tests pass.

**Step 6: Commit**

```bash
git add engine.go engine_test.go
git commit -m "feat: add LibraryResolver option and include resolution"
```

---

## Task 5: `value[x]` choice type resolution in evalMemberAccess

When accessing `Observation.value`, the JSON has `valueQuantity`. Use ModelInfo to try choice type suffixes.

**Files:**
- Modify: `eval/evaluator.go` (evalMemberAccess)
- Modify: `model/modelinfo.go` (add ElementInfoByPath helper)
- Test: `eval/evaluator_test.go`

**Step 1: Add ElementInfoByPath to StaticModelInfo**

Currently there's no direct method to get ElementInfo by path. The `TypeInfo` + iterate pattern is cumbersome. Add a convenience method.

In `model/modelinfo.go`:

```go
// ElementInfoByPath returns the ElementInfo for a dot-path like "Observation.value".
func (m *StaticModelInfo) ElementInfoByPath(path string) (*ElementInfo, bool) {
    parts := strings.SplitN(path, ".", 2)
    if len(parts) != 2 {
        return nil, false
    }
    typeName := parts[0]
    elemName := parts[1]
    ti, ok := m.TypeInfo(typeName)
    if !ok {
        return nil, false
    }
    for i := range ti.Elements {
        if ti.Elements[i].Name == elemName {
            return &ti.Elements[i], true
        }
    }
    return nil, false
}
```

Add `"strings"` to the imports.

Also add `ElementInfoByPath` to the `ModelInfo` interface:

```go
    ElementInfoByPath(path string) (*ElementInfo, bool)
```

**Step 2: Write failing test for value[x]**

In `eval/evaluator_test.go`:

```go
func TestEvalMemberAccess_ChoiceType(t *testing.T) {
    obsJSON := json.RawMessage(`{
        "resourceType": "Observation",
        "status": "final",
        "valueQuantity": {
            "value": 128,
            "unit": "cm"
        }
    }`)

    lib := &ast.Library{
        Statements: []*ast.ExpressionDef{
            {
                Name: "ObsValue",
                Expression: &ast.MemberAccess{
                    Source: &ast.IdentifierRef{Name: "$context"},
                    Member: "value",
                },
            },
        },
    }

    ctx := NewContext(context.Background(), lib)
    ctx.ContextValue = obsJSON
    ctx.ModelInfo = model.DefaultR4ModelInfo()
    evaluator := NewEvaluator(ctx)
    results, err := evaluator.EvaluateLibrary()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    val := results["ObsValue"]
    if val == nil {
        t.Fatal("ObsValue should not be nil — value[x] resolution failed")
    }
}
```

Note: The test may need adjustment depending on how `$context` is resolved. The key assertion is that accessing `.value` on an Observation with `valueQuantity` returns non-nil. If `$context` doesn't work, use the context object directly by accessing `Patient` or `Observation` context value. Adjust the test to use whatever pattern the existing tests use for accessing the context resource.

**Step 3: Run test to verify it fails**

Run: `go test ./eval/ -run TestEvalMemberAccess_ChoiceType -v`
Expected: FAIL — `ObsValue` is nil because direct lookup of "value" fails.

**Step 4: Update evalMemberAccess**

In `eval/evaluator.go`, modify `evalMemberAccess`:

```go
func (e *Evaluator) evalMemberAccess(n *ast.MemberAccess) (fptypes.Value, error) {
    source, err := e.Eval(n.Source)
    if err != nil {
        return nil, err
    }
    if source == nil {
        return nil, nil
    }
    // Tuple member access
    if t, ok := source.(cqltypes.Tuple); ok {
        v, _ := t.Get(n.Member)
        return v, nil
    }
    // JSON object member access
    if obj, ok := source.(*fptypes.ObjectValue); ok {
        result := obj.GetCollection(n.Member)
        if result.Count() > 0 {
            if result.Count() == 1 {
                return result[0], nil
            }
            return cqltypes.NewList(result), nil
        }

        // Choice type resolution: check ModelInfo for value[x] patterns
        if e.ctx.ModelInfo != nil {
            typeName := obj.Type() // e.g. "Observation"
            path := typeName + "." + n.Member
            if e.ctx.ModelInfo.IsChoiceType(path) {
                if ei, ok := e.ctx.ModelInfo.ElementInfoByPath(path); ok {
                    for _, choiceType := range ei.ChoiceTypes {
                        // Extract suffix: "FHIR.Quantity" → "Quantity", "System.String" → "String"
                        suffix := choiceType
                        if idx := strings.LastIndex(choiceType, "."); idx >= 0 {
                            suffix = choiceType[idx+1:]
                        }
                        concreteKey := n.Member + suffix
                        result = obj.GetCollection(concreteKey)
                        if result.Count() > 0 {
                            if result.Count() == 1 {
                                return result[0], nil
                            }
                            return cqltypes.NewList(result), nil
                        }
                    }
                }
            }
        }

        return nil, nil
    }
    return nil, nil
}
```

**Step 5: Run tests**

Run: `go test ./eval/ -run TestEvalMemberAccess_ChoiceType -v`
Expected: PASS

Run: `go test ./... -count=1`
Expected: All tests pass.

**Step 6: Commit**

```bash
git add eval/evaluator.go eval/evaluator_test.go model/modelinfo.go
git commit -m "feat: value[x] choice type resolution in member access"
```

---

## Task 6: Built-in FHIRHelpers stub

Provide a built-in FHIRHelpers library that works with raw JSON values (Option A). This is used as fallback when no LibraryResolver is provided or when the resolver doesn't have FHIRHelpers.

**Files:**
- Create: `fhirhelpers/fhirhelpers.go` (CQL source string for built-in FHIRHelpers)
- Modify: `engine.go` (auto-register FHIRHelpers as fallback)
- Test: `engine_test.go`

**Step 1: Write the end-to-end failing test**

In `engine_test.go`:

```go
func TestEngine_FHIRHelpers_ToQuantity(t *testing.T) {
    e := NewEngine()

    cqlSource := `library Test version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'

define Obs: Tuple {
    resourceType: 'Observation',
    valueQuantity: Tuple { value: 7.2, unit: 'mg/dL' }
}

define QuantityValue: FHIRHelpers.ToQuantity(Obs.valueQuantity)`

    results, err := e.EvaluateLibrary(context.Background(), cqlSource, nil, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    // FHIRHelpers.ToQuantity should return a System.Quantity
    val := results["QuantityValue"]
    if val == nil {
        t.Fatal("QuantityValue should not be nil")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test . -run TestEngine_FHIRHelpers_ToQuantity -v`
Expected: FAIL — no resolver for FHIRHelpers.

**Step 3: Create built-in FHIRHelpers source**

Create `fhirhelpers/fhirhelpers.go`:

```go
// Package fhirhelpers provides a built-in FHIRHelpers CQL library
// that works with raw JSON values (no FHIR primitive wrapping).
package fhirhelpers

// Source returns the CQL source for the built-in FHIRHelpers 4.0.1.
// This is a simplified version that passes through raw values,
// since the Go engine accesses JSON fields directly without FHIR primitive wrapping.
const Source = `library FHIRHelpers version '4.0.1'

using FHIR version '4.0.1'

define function ToBoolean(value Boolean): value
define function ToString(value String): value
define function ToInteger(value Integer): value
define function ToDecimal(value Decimal): value
define function ToDateTime(value DateTime): value
define function ToDate(value Date): value
define function ToTime(value Time): value
define function ToQuantity(quantity Quantity): quantity
`
```

**Step 4: Auto-register FHIRHelpers in engine**

In `engine.go`, modify `resolveIncludes` to provide FHIRHelpers as a built-in fallback:

```go
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
```

Add import for `fhirhelpers` package.

**Step 5: Run tests**

Run: `go test . -run TestEngine_FHIRHelpers_ToQuantity -v`
Expected: PASS

Run: `go test ./... -count=1`
Expected: All tests pass.

**Step 6: Commit**

```bash
git add fhirhelpers/fhirhelpers.go engine.go engine_test.go
git commit -m "feat: built-in FHIRHelpers stub with passthrough functions"
```

---

## Task 7: End-to-end integration test

Write a comprehensive test matching the issue's acceptance criteria using the full engine with a Patient context resource.

**Files:**
- Test: `engine_test.go`

**Step 1: Write the integration test**

```go
func TestEngine_FHIRHelpers_EndToEnd(t *testing.T) {
    patient := json.RawMessage(`{
        "resourceType": "Patient",
        "id": "test-1",
        "name": [{"family": "Smith", "given": ["Jane"]}],
        "gender": "female",
        "birthDate": "1962-07-22"
    }`)

    e := NewEngine()

    cqlSource := `library Test version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'

context Patient

define "Family Name":
    Patient.name.first().family

define "Gender":
    Patient.gender`

    results, err := e.EvaluateLibrary(context.Background(), cqlSource, patient, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if fn, ok := results["Family Name"].(fptypes.String); !ok || fn.Value() != "Smith" {
        t.Errorf("Family Name = %v, want 'Smith'", results["Family Name"])
    }

    if g, ok := results["Gender"].(fptypes.String); !ok || g.Value() != "female" {
        t.Errorf("Gender = %v, want 'female'", results["Gender"])
    }
}
```

**Step 2: Run the test**

Run: `go test . -run TestEngine_FHIRHelpers_EndToEnd -v`
Expected: PASS

**Step 3: Commit**

```bash
git add engine_test.go
git commit -m "test: add end-to-end FHIRHelpers integration test"
```

---

## Task 8: Type-aware overload resolution

The simple operand-count-based resolution from Task 2 won't distinguish `ToString(FHIR.string)` from `ToString(FHIR.code)`. Add type scoring.

**Files:**
- Modify: `eval/evaluator.go` (resolveOverload)
- Test: `eval/evaluator_test.go`

**Step 1: Write failing test**

In `eval/evaluator_test.go`:

```go
func TestResolveOverload_ByArgumentType(t *testing.T) {
    // Two overloads with same arity but different operand types
    overloads := []*ast.FunctionDef{
        {
            Name:     "Convert",
            Operands: []*ast.OperandDef{{Name: "val", TypeSpec: &ast.NamedTypeSpecifier{Name: "Integer"}}},
            Body:     &ast.Literal{ValueType: ast.LiteralString, Value: "integer-path"},
        },
        {
            Name:     "Convert",
            Operands: []*ast.OperandDef{{Name: "val", TypeSpec: &ast.NamedTypeSpecifier{Name: "String"}}},
            Body:     &ast.Literal{ValueType: ast.LiteralString, Value: "string-path"},
        },
    }

    // Calling with a string argument should pick the String overload
    args := []ast.Expression{&ast.Literal{ValueType: ast.LiteralString, Value: "hello"}}
    fd := resolveOverload(overloads, args)
    if fd.Operands[0].TypeSpec.(*ast.NamedTypeSpecifier).Name != "String" {
        t.Errorf("expected String overload, got %s", fd.Operands[0].TypeSpec.(*ast.NamedTypeSpecifier).Name)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./eval/ -run TestResolveOverload_ByArgumentType -v`
Expected: FAIL — current resolveOverload doesn't check types.

**Step 3: Enhance resolveOverload with type scoring**

```go
func resolveOverload(overloads []*ast.FunctionDef, args []ast.Expression) *ast.FunctionDef {
    if len(overloads) == 1 {
        return overloads[0]
    }

    // First filter by arity
    var candidates []*ast.FunctionDef
    for _, fd := range overloads {
        if len(fd.Operands) == len(args) {
            candidates = append(candidates, fd)
        }
    }
    if len(candidates) == 0 {
        return overloads[0]
    }
    if len(candidates) == 1 {
        return candidates[0]
    }

    // Score by argument type match
    bestScore := -1
    var best *ast.FunctionDef
    for _, fd := range candidates {
        score := 0
        for i, op := range fd.Operands {
            if i < len(args) && op.TypeSpec != nil {
                if nts, ok := op.TypeSpec.(*ast.NamedTypeSpecifier); ok {
                    if matchesArgType(args[i], nts.Name) {
                        score++
                    }
                }
            }
        }
        if score > bestScore {
            bestScore = score
            best = fd
        }
    }
    if best != nil {
        return best
    }
    return candidates[0]
}

// matchesArgType checks if an AST expression matches the expected type name.
func matchesArgType(expr ast.Expression, typeName string) bool {
    switch e := expr.(type) {
    case *ast.Literal:
        switch e.ValueType {
        case ast.LiteralInteger:
            return typeName == "Integer" || typeName == "System.Integer"
        case ast.LiteralDecimal:
            return typeName == "Decimal" || typeName == "System.Decimal"
        case ast.LiteralString:
            return typeName == "String" || typeName == "System.String"
        case ast.LiteralBoolean:
            return typeName == "Boolean" || typeName == "System.Boolean"
        case ast.LiteralLong:
            return typeName == "Long" || typeName == "System.Long"
        }
    case *ast.QuantityLiteral:
        return typeName == "Quantity" || typeName == "System.Quantity"
    }
    return false
}
```

**Step 4: Run tests**

Run: `go test ./eval/ -run TestResolveOverload_ByArgumentType -v`
Expected: PASS

Run: `go test ./... -count=1`
Expected: All tests pass.

**Step 5: Commit**

```bash
git add eval/evaluator.go eval/evaluator_test.go
git commit -m "feat: type-aware overload resolution for function dispatch"
```

---

## Summary: Implementation Order

| Task | Description | Dependencies |
|------|-------------|--------------|
| 1 | ModelInfo in eval.Context | None |
| 2 | Function overloads in Evaluator | None |
| 3 | Library-qualified function dispatch | Task 2 |
| 4 | LibraryResolver option in Engine | Task 3 |
| 5 | value[x] choice type resolution | Task 1 |
| 6 | Built-in FHIRHelpers stub | Task 4 |
| 7 | End-to-end integration test | Tasks 5, 6 |
| 8 | Type-aware overload resolution | Task 2 |

Tasks 1 and 2 can be done in parallel. Tasks 5 and 8 can be done in parallel (after their deps).
