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

// RetrieveObserver is an optional extension a TraceListener may implement to be
// told what each retrieve asked the data provider for, and what came back.
//
// The expression tree cannot say this on its own. `[Condition: "Diabetes"]`
// carries no code path in its syntax — it comes from the model — and no subject:
// the context and the patient are resolved at evaluation. Justifying why a
// patient entered a population means being able to say which query produced the
// resources it was decided from, and that is only knowable here.
type RetrieveObserver interface {
	OnRetrieve(req RetrieveRequest, resultCount int, err error)
}

// TerminologyObserver is an optional extension a TraceListener may implement to
// be told what was asked of the terminology provider.
//
// `code in "Diabetes"` records true or false in the tree, which is the answer
// but not the reason: the code, the system and the value set that was consulted
// are what make the decision reviewable, and an expression tree shows none of
// them.
type TerminologyObserver interface {
	OnTerminologyCheck(code, system, valueSetURL string, inValueSet bool, err error)
}
