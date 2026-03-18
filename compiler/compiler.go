// Package compiler transforms CQL source text into an AST representation.
package compiler

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/parser/grammar"
)

// Compile parses CQL source text and returns a Library AST node.
func Compile(source string) (*ast.Library, error) {
	input := antlr.NewInputStream(source)
	lexer := grammar.NewcqlLexer(input)

	errListener := &errorCollector{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(errListener)

	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := grammar.NewcqlParser(stream)
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errListener)

	tree := parser.Library()
	if len(errListener.errors) > 0 {
		return nil, fmt.Errorf("CQL syntax error: %s", errListener.errors[0])
	}

	builder := newBuilder()
	result := builder.Visit(tree)
	if builder.err != nil {
		return nil, builder.err
	}
	lib, ok := result.(*ast.Library)
	if !ok {
		return nil, fmt.Errorf("expected Library node, got %T", result)
	}
	return lib, nil
}

// errorCollector collects ANTLR parse errors.
type errorCollector struct {
	antlr.DefaultErrorListener
	errors []string
}

func (e *errorCollector) SyntaxError(_ antlr.Recognizer, _ interface{}, line, column int, msg string, _ antlr.RecognitionException) {
	e.errors = append(e.errors, fmt.Sprintf("line %d:%d %s", line, column, msg))
}
