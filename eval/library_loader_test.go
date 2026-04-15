package eval

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofhir/cql/ast"
)

// mockLibraryLoader implements LibraryLoader for tests.
type mockLibraryLoader struct {
	libraries map[string]*ast.Library
}

func (m *mockLibraryLoader) LoadLibrary(_ context.Context, name, version string) (*ast.Library, error) {
	key := name + "/" + version
	if lib, ok := m.libraries[key]; ok {
		return lib, nil
	}
	return nil, nil
}

// errLibraryLoader always returns an error.
type errLibraryLoader struct{}

func (e *errLibraryLoader) LoadLibrary(_ context.Context, name, version string) (*ast.Library, error) {
	return nil, fmt.Errorf("storage unavailable for %s/%s", name, version)
}

// helperLibrary builds a library with a single function: Double(x) = x * 2
func helperLibrary() *ast.Library {
	return &ast.Library{
		Identifier: &ast.LibraryIdentifier{Name: "Helpers", Version: "1.0"},
		Functions: []*ast.FunctionDef{
			{
				Name: "Double",
				Operands: []*ast.OperandDef{
					{Name: "x"},
				},
				Body: &ast.BinaryExpression{
					Operator: ast.OpMultiply,
					Left:     &ast.IdentifierRef{Name: "x"},
					Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
				},
			},
		},
	}
}

// mainLibraryWithInclude builds a library that includes Helpers/1.0 as "H"
// and defines Result = H.Double(21).
func mainLibraryWithInclude() *ast.Library {
	lib := &ast.Library{
		Identifier: &ast.LibraryIdentifier{Name: "Main", Version: "1.0"},
		Includes: []*ast.IncludeDef{
			{Name: "Helpers", Version: "1.0", Alias: "H"},
		},
		Statements: []*ast.ExpressionDef{
			{
				Name: "Result",
				Expression: &ast.FunctionCall{
					Source:   &ast.IdentifierRef{Name: "H"},
					Name:     "Double",
					Operands: []ast.Expression{&ast.Literal{ValueType: ast.LiteralInteger, Value: "21"}},
				},
			},
		},
	}
	return lib
}

func TestCrossLibraryFunctionCall_Source(t *testing.T) {
	helperLib := helperLibrary()
	mainLib := mainLibraryWithInclude()

	loader := &mockLibraryLoader{
		libraries: map[string]*ast.Library{
			"Helpers/1.0": helperLib,
		},
	}

	ctx := NewContext(context.Background(), mainLib)
	ctx.LibraryLoader = loader
	ev := NewEvaluator(ctx)

	result, err := ev.EvaluateExpression("Result")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 42)
}

func TestCrossLibraryFunctionCall_LibraryField(t *testing.T) {
	helperLib := helperLibrary()
	// Build main lib using Library field instead of Source
	mainLib := &ast.Library{
		Identifier: &ast.LibraryIdentifier{Name: "Main", Version: "1.0"},
		Includes: []*ast.IncludeDef{
			{Name: "Helpers", Version: "1.0", Alias: "H"},
		},
		Statements: []*ast.ExpressionDef{
			{
				Name: "Result",
				Expression: &ast.FunctionCall{
					Library:  "H",
					Name:     "Double",
					Operands: []ast.Expression{&ast.Literal{ValueType: ast.LiteralInteger, Value: "21"}},
				},
			},
		},
	}

	loader := &mockLibraryLoader{
		libraries: map[string]*ast.Library{
			"Helpers/1.0": helperLib,
		},
	}

	ctx := NewContext(context.Background(), mainLib)
	ctx.LibraryLoader = loader
	ev := NewEvaluator(ctx)

	result, err := ev.EvaluateExpression("Result")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 42)
}

func TestCrossLibraryFunctionCall_DefaultAlias(t *testing.T) {
	// When no alias is specified, the library name is used as alias.
	helperLib := helperLibrary()
	mainLib := &ast.Library{
		Identifier: &ast.LibraryIdentifier{Name: "Main", Version: "1.0"},
		Includes: []*ast.IncludeDef{
			{Name: "Helpers", Version: "1.0"}, // no alias
		},
		Statements: []*ast.ExpressionDef{
			{
				Name: "Result",
				Expression: &ast.FunctionCall{
					Source:   &ast.IdentifierRef{Name: "Helpers"},
					Name:     "Double",
					Operands: []ast.Expression{&ast.Literal{ValueType: ast.LiteralInteger, Value: "10"}},
				},
			},
		},
	}

	loader := &mockLibraryLoader{
		libraries: map[string]*ast.Library{
			"Helpers/1.0": helperLib,
		},
	}

	ctx := NewContext(context.Background(), mainLib)
	ctx.LibraryLoader = loader
	ev := NewEvaluator(ctx)

	result, err := ev.EvaluateExpression("Result")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 20)
}

func TestCircularDependencyDetection(t *testing.T) {
	t.Run("direct guard", func(t *testing.T) {
		// If a library is already in the loading set, ensureLibraryLoaded returns an error.
		libA := &ast.Library{
			Identifier: &ast.LibraryIdentifier{Name: "A", Version: "1.0"},
			Includes:   []*ast.IncludeDef{{Name: "B", Version: "1.0"}},
		}

		loader := &mockLibraryLoader{
			libraries: map[string]*ast.Library{
				"B/1.0": {Identifier: &ast.LibraryIdentifier{Name: "B", Version: "1.0"}},
			},
		}

		ctx := NewContext(context.Background(), libA)
		ctx.LibraryLoader = loader
		ctx.loadingLibs = map[string]bool{"B/1.0": true} // simulate B already loading
		ev := NewEvaluator(ctx)

		err := ev.ensureLibraryLoaded("B")
		if err == nil {
			t.Fatal("expected circular dependency error, got nil")
		}
		expected := "circular library dependency detected: B/1.0"
		if got := err.Error(); got != expected {
			t.Fatalf("unexpected error: %q, want %q", got, expected)
		}
	})

	t.Run("via evalFunctionCall", func(t *testing.T) {
		// Library A tries to call B.Foo, but B/1.0 is already in loading set.
		libA := &ast.Library{
			Identifier: &ast.LibraryIdentifier{Name: "A", Version: "1.0"},
			Includes:   []*ast.IncludeDef{{Name: "B", Version: "1.0"}},
			Statements: []*ast.ExpressionDef{
				{
					Name: "Result",
					Expression: &ast.FunctionCall{
						Source:   &ast.IdentifierRef{Name: "B"},
						Name:     "Foo",
						Operands: []ast.Expression{&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"}},
					},
				},
			},
		}

		loader := &mockLibraryLoader{
			libraries: map[string]*ast.Library{
				"B/1.0": {Identifier: &ast.LibraryIdentifier{Name: "B", Version: "1.0"}},
			},
		}

		ctx := NewContext(context.Background(), libA)
		ctx.LibraryLoader = loader
		ctx.loadingLibs = map[string]bool{"B/1.0": true}
		ev := NewEvaluator(ctx)

		_, err := ev.EvaluateExpression("Result")
		if err == nil {
			t.Fatal("expected circular dependency error")
		}
	})
}

func TestLibraryLoaderNotSet(t *testing.T) {
	// No loader configured. Library-qualified call with no pre-loaded lib
	// should fall through to builtin resolution (and fail).
	mainLib := mainLibraryWithInclude()

	ctx := NewContext(context.Background(), mainLib)
	// no LibraryLoader set
	ev := NewEvaluator(ctx)

	_, err := ev.EvaluateExpression("Result")
	if err == nil {
		t.Fatal("expected error when library is not loaded and no loader is set")
	}
}

func TestLibraryLoaderReturnsNil(t *testing.T) {
	// Loader returns nil (library not found). Should not panic.
	mainLib := mainLibraryWithInclude()

	loader := &mockLibraryLoader{
		libraries: map[string]*ast.Library{}, // empty
	}

	ctx := NewContext(context.Background(), mainLib)
	ctx.LibraryLoader = loader
	ev := NewEvaluator(ctx)

	_, err := ev.EvaluateExpression("Result")
	if err == nil {
		t.Fatal("expected error when library not found by loader")
	}
}

func TestLibraryLoaderError(t *testing.T) {
	mainLib := mainLibraryWithInclude()

	ctx := NewContext(context.Background(), mainLib)
	ctx.LibraryLoader = &errLibraryLoader{}
	ev := NewEvaluator(ctx)

	_, err := ev.EvaluateExpression("Result")
	if err == nil {
		t.Fatal("expected error from loader")
	}
}

func TestLibraryLoaderIdempotent(t *testing.T) {
	// Calling a library-qualified function twice should load the library only once.
	helperLib := helperLibrary()
	mainLib := &ast.Library{
		Identifier: &ast.LibraryIdentifier{Name: "Main", Version: "1.0"},
		Includes: []*ast.IncludeDef{
			{Name: "Helpers", Version: "1.0", Alias: "H"},
		},
		Statements: []*ast.ExpressionDef{
			{
				Name: "A",
				Expression: &ast.FunctionCall{
					Source:   &ast.IdentifierRef{Name: "H"},
					Name:     "Double",
					Operands: []ast.Expression{&ast.Literal{ValueType: ast.LiteralInteger, Value: "5"}},
				},
			},
			{
				Name: "B",
				Expression: &ast.FunctionCall{
					Source:   &ast.IdentifierRef{Name: "H"},
					Name:     "Double",
					Operands: []ast.Expression{&ast.Literal{ValueType: ast.LiteralInteger, Value: "10"}},
				},
			},
		},
	}

	callCount := 0
	loader := &countingLibraryLoader{
		libraries: map[string]*ast.Library{"Helpers/1.0": helperLib},
		count:     &callCount,
	}

	ctx := NewContext(context.Background(), mainLib)
	ctx.LibraryLoader = loader
	ev := NewEvaluator(ctx)

	results, err := ev.EvaluateLibrary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, results["A"], 10)
	assertInteger(t, results["B"], 20)

	// The loader should have been called exactly once for Helpers/1.0
	if callCount != 1 {
		t.Fatalf("expected loader to be called 1 time, got %d", callCount)
	}
}

// countingLibraryLoader wraps mockLibraryLoader and counts calls.
type countingLibraryLoader struct {
	libraries map[string]*ast.Library
	count     *int
}

func (c *countingLibraryLoader) LoadLibrary(_ context.Context, name, version string) (*ast.Library, error) {
	*c.count++
	key := name + "/" + version
	if lib, ok := c.libraries[key]; ok {
		return lib, nil
	}
	return nil, nil
}

func TestChildScopePropagatesLibraryLoader(t *testing.T) {
	loader := &mockLibraryLoader{}
	ctx := NewContext(context.Background(), nil)
	ctx.LibraryLoader = loader
	ctx.loadingLibs = map[string]bool{"test": true}

	child := ctx.ChildScope()
	if child.LibraryLoader != loader {
		t.Fatal("ChildScope did not propagate LibraryLoader")
	}
	if child.loadingLibs == nil || !child.loadingLibs["test"] {
		t.Fatal("ChildScope did not propagate loadingLibs")
	}
}
