package eval

import (
	"context"
	"fmt"

	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

// ExpandedCode represents a single code from a ValueSet expansion.
type ExpandedCode struct {
	System  string
	Code    string
	Display string
}

// ValueSetExpander is an optional extension to TerminologyProvider.
// Implement this alongside TerminologyProvider to support CQL expandValueSet.
type ValueSetExpander interface {
	ExpandValueSet(ctx context.Context, valueSetURL string) ([]ExpandedCode, error)
}

func (e *Evaluator) evalExpandValueSet(vsURL string) (fptypes.Value, error) {
	if vsURL == "" {
		return nil, nil
	}
	expander, ok := e.ctx.TerminologyProvider.(ValueSetExpander)
	if !ok || expander == nil {
		return nil, nil
	}
	codes, err := expander.ExpandValueSet(e.ctx.GoCtx, vsURL)
	if err != nil {
		return nil, fmt.Errorf("ExpandValueSet failed: %w", err)
	}
	vals := make(fptypes.Collection, len(codes))
	for i, c := range codes {
		vals[i] = cqltypes.NewCode(c.System, c.Code, c.Display)
	}
	return cqltypes.NewList(vals), nil
}
