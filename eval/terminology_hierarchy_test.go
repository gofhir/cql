package eval

import (
	"context"
	"testing"

	"github.com/gofhir/cql/ast"
	cqltypes "github.com/gofhir/cql/types"
)

// mockSubsumesProvider implements both TerminologyProvider and SubsumesChecker.
type mockSubsumesProvider struct{}

func (m *mockSubsumesProvider) InValueSet(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}

func (m *mockSubsumesProvider) Subsumes(_ context.Context, _, codeA, codeB string) (bool, error) {
	// parent subsumes child
	return codeA == "parent" && codeB == "child", nil
}

// mockTermOnlyProvider implements only TerminologyProvider (no SubsumesChecker).
type mockTermOnlyProvider struct{}

func (m *mockTermOnlyProvider) InValueSet(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}

func TestSubsumes_NilProvider(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.Definitions["codeA"] = cqltypes.NewCode("http://snomed.info/sct", "parent", "")
	ctx.Definitions["codeB"] = cqltypes.NewCode("http://snomed.info/sct", "child", "")
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.FunctionCall{
		Name: "Subsumes",
		Operands: []ast.Expression{
			&ast.IdentifierRef{Name: "codeA"},
			&ast.IdentifierRef{Name: "codeB"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil (null), got %v", val)
	}
}

func TestSubsumes_ProviderWithoutSubsumesChecker(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.TerminologyProvider = &mockTermOnlyProvider{}
	ctx.Definitions["codeA"] = cqltypes.NewCode("http://snomed.info/sct", "parent", "")
	ctx.Definitions["codeB"] = cqltypes.NewCode("http://snomed.info/sct", "child", "")
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.FunctionCall{
		Name: "Subsumes",
		Operands: []ast.Expression{
			&ast.IdentifierRef{Name: "codeA"},
			&ast.IdentifierRef{Name: "codeB"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil (null), got %v", val)
	}
}

func TestSubsumes_NilCodes(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.TerminologyProvider = &mockSubsumesProvider{}
	ctx.Definitions["codeA"] = cqltypes.NewCode("http://snomed.info/sct", "parent", "")
	ev := NewEvaluator(ctx)

	// nil left
	val, err := ev.Eval(&ast.FunctionCall{
		Name: "Subsumes",
		Operands: []ast.Expression{
			&ast.Literal{ValueType: ast.LiteralNull},
			&ast.IdentifierRef{Name: "codeA"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil for null left code, got %v", val)
	}

	// nil right
	val, err = ev.Eval(&ast.FunctionCall{
		Name: "Subsumes",
		Operands: []ast.Expression{
			&ast.IdentifierRef{Name: "codeA"},
			&ast.Literal{ValueType: ast.LiteralNull},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil for null right code, got %v", val)
	}
}

func TestSubsumes_DifferentSystems(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.TerminologyProvider = &mockSubsumesProvider{}
	ctx.Definitions["codeA"] = cqltypes.NewCode("http://snomed.info/sct", "parent", "")
	ctx.Definitions["codeB"] = cqltypes.NewCode("http://loinc.org", "child", "")
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.FunctionCall{
		Name: "Subsumes",
		Operands: []ast.Expression{
			&ast.IdentifierRef{Name: "codeA"},
			&ast.IdentifierRef{Name: "codeB"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBoolean(t, val, false, "different systems")
}

func TestSubsumes_ParentChild(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.TerminologyProvider = &mockSubsumesProvider{}
	ctx.Definitions["codeA"] = cqltypes.NewCode("http://snomed.info/sct", "parent", "")
	ctx.Definitions["codeB"] = cqltypes.NewCode("http://snomed.info/sct", "child", "")
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.FunctionCall{
		Name: "Subsumes",
		Operands: []ast.Expression{
			&ast.IdentifierRef{Name: "codeA"},
			&ast.IdentifierRef{Name: "codeB"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBoolean(t, val, true, "parent subsumes child")
}

func TestSubsumes_NonParent(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.TerminologyProvider = &mockSubsumesProvider{}
	ctx.Definitions["codeA"] = cqltypes.NewCode("http://snomed.info/sct", "child", "")
	ctx.Definitions["codeB"] = cqltypes.NewCode("http://snomed.info/sct", "parent", "")
	ev := NewEvaluator(ctx)

	val, err := ev.Eval(&ast.FunctionCall{
		Name: "Subsumes",
		Operands: []ast.Expression{
			&ast.IdentifierRef{Name: "codeA"},
			&ast.IdentifierRef{Name: "codeB"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBoolean(t, val, false, "child does not subsume parent")
}

func TestSubsumedBy_Reversal(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.TerminologyProvider = &mockSubsumesProvider{}
	ctx.Definitions["child"] = cqltypes.NewCode("http://snomed.info/sct", "child", "")
	ctx.Definitions["parent"] = cqltypes.NewCode("http://snomed.info/sct", "parent", "")
	ev := NewEvaluator(ctx)

	// SubsumedBy(child, parent) should be true because it calls Subsumes(parent, child)
	val, err := ev.Eval(&ast.FunctionCall{
		Name: "SubsumedBy",
		Operands: []ast.Expression{
			&ast.IdentifierRef{Name: "child"},
			&ast.IdentifierRef{Name: "parent"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBoolean(t, val, true, "child subsumed by parent")
}

func TestSubsumedBy_NotSubsumed(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.TerminologyProvider = &mockSubsumesProvider{}
	ctx.Definitions["parent"] = cqltypes.NewCode("http://snomed.info/sct", "parent", "")
	ctx.Definitions["child"] = cqltypes.NewCode("http://snomed.info/sct", "child", "")
	ev := NewEvaluator(ctx)

	// SubsumedBy(parent, child) should be false
	val, err := ev.Eval(&ast.FunctionCall{
		Name: "SubsumedBy",
		Operands: []ast.Expression{
			&ast.IdentifierRef{Name: "parent"},
			&ast.IdentifierRef{Name: "child"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBoolean(t, val, false, "parent not subsumed by child")
}
