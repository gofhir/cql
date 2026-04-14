package eval

import (
	"context"
	"fmt"

	fptypes "github.com/gofhir/fhirpath/types"
)

// SubsumesChecker is an optional extension to TerminologyProvider.
// Implement this alongside TerminologyProvider to support CQL Subsumes/SubsumedBy.
type SubsumesChecker interface {
	Subsumes(ctx context.Context, systemURL, codeA, codeB string) (bool, error)
}

func (e *Evaluator) evalSubsumes(codeA, codeB fptypes.Value) (fptypes.Value, error) {
	if codeA == nil || codeB == nil {
		return nil, nil
	}
	checker, ok := e.ctx.TerminologyProvider.(SubsumesChecker)
	if !ok || checker == nil {
		return nil, nil
	}
	systemA, valA := extractCodeComponents(codeA)
	systemB, valB := extractCodeComponents(codeB)
	if systemA != systemB || systemA == "" {
		return fptypes.NewBoolean(false), nil
	}
	result, err := checker.Subsumes(e.ctx.GoCtx, systemA, valA, valB)
	if err != nil {
		return nil, fmt.Errorf("Subsumes check failed: %w", err)
	}
	return fptypes.NewBoolean(result), nil
}

func (e *Evaluator) evalSubsumedBy(codeA, codeB fptypes.Value) (fptypes.Value, error) {
	return e.evalSubsumes(codeB, codeA)
}
