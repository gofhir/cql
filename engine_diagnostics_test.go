package cql

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/eval"
)

// TestEvaluationErrorsCarryTheirLocation covers where an evaluation error
// happened. Only ANTLR's syntax errors used to say, so every runtime diagnostic
// named what went wrong and left the reader to find it.
func TestEvaluationErrorsCarryTheirLocation(t *testing.T) {
	for _, tt := range []struct {
		name string
		src  string
		at   string
	}{
		{
			"unknown identifier",
			"library T version '1.0'\n\ndefine A: Bogus\n",
			"3:11",
		},
		{
			"unknown identifier on its own line",
			"library T version '1.0'\n\ndefine A:\n  Bogus\n",
			"4:3",
		},
		{
			// The sort key, not the query that contains it: pointing at the
			// whole query leaves the reader to find the word.
			"unknown sort key",
			"library T version '1.0'\n\ndefine A: ({1,2}) X sort by nope\n",
			"3:29",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEngine().EvaluateExpression(context.Background(), tt.src, "A", nil, nil)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tt.at) {
				t.Errorf("error does not point at %s: %v", tt.at, err)
			}

			var positioned *eval.PositionedError
			if !errors.As(err, &positioned) {
				t.Fatalf("error carries no position: %v", err)
			}
			if !positioned.Position.Known() {
				t.Error("the position is the zero value")
			}
		})
	}
}

// TestOnlyTheInnermostFailureIsLocated covers how the position travels. An
// error passes through every enclosing expression on its way out, and each of
// them attaching its own would point at the statement rather than at the part
// that went wrong — and would grow the message once per level, which is how a
// recursion error once reached 380 KB.
func TestOnlyTheInnermostFailureIsLocated(t *testing.T) {
	src := "library T version '1.0'\n\ndefine A: 1 + (2 * (3 + Bogus))\n"
	_, err := NewEngine().EvaluateExpression(context.Background(), src, "A", nil, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	// One location, and it is the identifier's rather than the statement's.
	if n := strings.Count(err.Error(), ":"); n > 3 {
		t.Errorf("the message carries more than one location: %v", err)
	}
	// `define A: 1 + (2 * (3 + Bogus))` — Bogus starts at column 25, counting
	// from one the way ELM locators do.
	if !strings.Contains(err.Error(), "3:25") {
		t.Errorf("expected the position of Bogus (3:25), got: %v", err)
	}
}

// TestUnlocatableErrorsStillRead covers an error with nowhere to point. Nodes
// built without the parser — by a test, or by the ELM importer — carry no
// position, and the message must read without a stray separator where the
// location would have been.
func TestUnlocatableErrorsStillRead(t *testing.T) {
	plain := errors.New("something went wrong")
	unlocated := &eval.PositionedError{Err: plain}
	if got := unlocated.Error(); got != "something went wrong" {
		t.Errorf("an unlocated error reads %q, want the message alone", got)
	}

	located := &eval.PositionedError{Position: ast.Position{Line: 3, Col: 11}, Err: plain}
	if got := located.Error(); got != "3:11: something went wrong" {
		t.Errorf("a located error reads %q", got)
	}

	inLib := &eval.PositionedError{
		Position: ast.Position{Line: 8, Col: 34}, Library: "Helper", Err: plain,
	}
	if got := inLib.Error(); got != "Helper 8:34: something went wrong" {
		t.Errorf("an error from an included library reads %q", got)
	}
}

// TestErrorsFromIncludedLibrariesNameIt covers a position that belongs to
// another source. A four-line library reporting an error at line 8 sends the
// reader to a line that is not there; the library the line belongs to has to be
// part of the message.
func TestErrorsFromIncludedLibrariesNameIt(t *testing.T) {
	helper := "library Helper version '1.0'\n\n// 3\n// 4\n// 5\n// 6\n// 7\ndefine function Boom(x Integer): NoSuchThing\n"
	src := "library M version '1.0'\ninclude Helper version '1.0' called H\n\ndefine A: H.Boom(1)\n"

	_, err := NewEngine(WithLibraryResolver(func(_ context.Context, name, _ string) (string, error) {
		if name == "Helper" {
			return helper, nil
		}
		return "", context.Canceled
	})).EvaluateExpression(context.Background(), src, "A", nil, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "Helper") {
		t.Errorf("the error does not say which library line 8 belongs to: %v", err)
	}
}
