package cql

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// wrongHelper is a library whose semantic phase finds an error, in a
// definition nobody asks for.
const wrongHelper = "library H version '1.0'\n\ndefine Wrong: 1 + 'x'\ndefine Right: 2\n"

// TestAnIncludedLibraryIsHeldToItsSemanticPhase covers an included library
// whose semantic phase finds errors. The evaluated library is refused for its
// own; an included one was evaluated all the same, directly or through
// another library.
func TestAnIncludedLibraryIsHeldToItsSemanticPhase(t *testing.T) {
	libs := map[string]string{
		"H": wrongHelper,
		"M": "library M version '1.0'\ninclude H version '1.0' called H\n\ndefine Right: H.Right\n",
	}
	for _, tt := range []struct{ name, src string }{
		{"included", "library T version '1.0'\ninclude H version '1.0' called H\n\ndefine X: H.Right\n"},
		{"included by an included library", "library T version '1.0'\ninclude M version '1.0' called M\n\ndefine X: M.Right\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine(WithLibraryResolver(resolverOf(libs, nil)))
			got, err := engine.EvaluateExpression(context.Background(), tt.src, "X", nil, nil)
			var semantic *ErrSemantic
			if !errors.As(err, &semantic) {
				t.Fatalf("X = %v, err %T %v; want an ErrSemantic", got, err, err)
			}
			if len(semantic.Diagnostics) != 1 {
				t.Fatalf("diagnostics = %v; want the one in H", semantic.Diagnostics)
			}
			d := semantic.Diagnostics[0]
			if d.Library == nil || d.Library.Name != "H" || d.Library.Version != "1.0" {
				t.Errorf("the finding is in %+v; want H 1.0", d.Library)
			}
			if d.Position.Line != 3 || !strings.Contains(err.Error(), "library H version '1.0'") {
				t.Errorf("err = %v; want the finding at H's line 3, saying it is H's", err)
			}

			// Turned off, it evaluates, as the evaluated library's own would.
			lenient := NewEngine(WithLibraryResolver(resolverOf(libs, nil)), WithSemanticValidation(false))
			if got, err := lenient.EvaluateExpression(context.Background(), tt.src, "X", nil, nil); err != nil || valueString(got) != "2" {
				t.Errorf("without semantic validation: X = %v, err %v; want 2", got, err)
			}
		})
	}
}
