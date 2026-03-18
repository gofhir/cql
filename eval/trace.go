package eval

import (
	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
)

// TraceListener receives events during CQL expression evaluation.
// OnEnter is called before evaluating an expression node and OnExit is
// called after, forming a natural stack that can be used to build a trace tree.
//
// Implementations must be goroutine-safe if used with parallel evaluation.
type TraceListener interface {
	// OnEnter is called before evaluating an expression node.
	OnEnter(expr ast.Expression)

	// OnExit is called after evaluating an expression node with its result and any error.
	OnExit(expr ast.Expression, result fptypes.Value, err error)
}
