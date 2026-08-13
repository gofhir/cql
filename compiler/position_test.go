package compiler

import (
	"testing"

	"github.com/gofhir/cql/ast"
)

// TestExpressionsCarryTheirPosition covers where each expression began in the
// source. Until now only ANTLR's own syntax errors carried a location, so an
// evaluation error said what went wrong and never where.
func TestExpressionsCarryTheirPosition(t *testing.T) {
	src := `library T version '1.0'

define A: 1 + 2
define B:
  Bogus
define C: ({1,2}) X
  where X > 1
  return X
`
	lib, err := Compile(src)
	if err != nil {
		t.Fatalf("compiling: %v", err)
	}
	if len(lib.Statements) != 3 {
		t.Fatalf("got %d statements, want 3", len(lib.Statements))
	}

	// Columns are one-based, the way ELM locators and editors count them, so
	// these can be compared with what the reference translator reports.
	for _, tt := range []struct {
		name string
		line int
		col  int
	}{
		{"A", 3, 11},
		{"B", 5, 3}, // the definition's body is on its own line
		{"C", 6, 11},
	} {
		var stmt = findStatement(lib, tt.name)
		if stmt == nil {
			t.Errorf("no statement named %s", tt.name)
			continue
		}
		pos, known := ast.PositionOf(stmt.Expression)
		if !known {
			t.Errorf("%s has no position", tt.name)
			continue
		}
		if pos.Line != tt.line || pos.Col != tt.col {
			t.Errorf("%s at %v, want %d:%d", tt.name, pos, tt.line, tt.col)
		}
	}
}

// TestInnerExpressionsKeepTheirOwnPosition covers the parts of an expression.
// A diagnostic about one operand should point at the operand, not at the
// statement that contains it.
func TestInnerExpressionsKeepTheirOwnPosition(t *testing.T) {
	lib, err := Compile("library T version '1.0'\n\ndefine A: 1 + Bogus\n")
	if err != nil {
		t.Fatalf("compiling: %v", err)
	}
	binary, ok := findStatement(lib, "A").Expression.(*ast.BinaryExpression)
	if !ok {
		t.Fatalf("expected a BinaryExpression, got %T", findStatement(lib, "A").Expression)
	}
	outer, _ := ast.PositionOf(binary)
	right, known := ast.PositionOf(binary.Right)
	if !known {
		t.Fatal("the right operand has no position")
	}
	if right.Col <= outer.Col {
		t.Errorf("the operand is at %v and the expression at %v; the operand should be further along",
			right, outer)
	}
}

// TestHandBuiltNodesHaveNoPosition covers the zero value. Nodes built by a test
// or by the ELM importer never went through the parser, and a diagnostic has to
// read as well without a position as with one.
func TestHandBuiltNodesHaveNoPosition(t *testing.T) {
	node := &ast.Literal{ValueType: ast.LiteralInteger, Value: "1"}
	if pos, known := ast.PositionOf(node); known {
		t.Errorf("a hand-built node reported position %v", pos)
	}
	if got := (ast.Position{}).String(); got != "" {
		t.Errorf("an unknown position renders as %q, want empty", got)
	}
}

func findStatement(lib *ast.Library, name string) *ast.ExpressionDef {
	for _, stmt := range lib.Statements {
		if stmt.Name == name {
			return stmt
		}
	}
	return nil
}
