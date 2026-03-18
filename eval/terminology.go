package eval

import (
	"fmt"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
	cqltypes "github.com/gofhir/cql/types"
)

// evalInValueSet evaluates "Code in ValueSetRef" membership.
func (e *Evaluator) evalInValueSet(code fptypes.Value, vsRef *ast.IdentifierRef) (fptypes.Value, error) {
	if code == nil {
		return nil, nil
	}

	// Resolve the ValueSet URL from the library
	vsURL, found := e.ctx.ResolveValueSetURL(vsRef.Name)
	if !found || vsURL == "" {
		return nil, fmt.Errorf("ValueSet '%s' not found in library", vsRef.Name)
	}

	// Extract system and code from the value
	system, codeVal := extractCodeComponents(code)
	if codeVal == "" {
		return nil, nil
	}

	// Use the TerminologyProvider if available
	if e.ctx.TerminologyProvider != nil {
		result, err := e.ctx.TerminologyProvider.InValueSet(e.ctx.GoCtx, codeVal, system, vsURL)
		if err != nil {
			return nil, fmt.Errorf("terminology check failed for ValueSet %s: %w", vsURL, err)
		}
		return fptypes.NewBoolean(result), nil
	}

	// Without a TerminologyProvider, we can only check in-memory ValueSets
	return nil, nil
}

// evalInCodeSystem evaluates "Code in CodeSystemRef" membership.
func (e *Evaluator) evalInCodeSystem(code fptypes.Value, csRef *ast.IdentifierRef) (fptypes.Value, error) {
	if code == nil {
		return nil, nil
	}

	// Resolve the CodeSystem URL from the library
	csURL := ""
	if e.ctx.Library != nil {
		for _, cs := range e.ctx.Library.CodeSystems {
			if cs.Name == csRef.Name {
				csURL = cs.ID
				break
			}
		}
	}
	if csURL == "" {
		return nil, fmt.Errorf("CodeSystem '%s' not found in library", csRef.Name)
	}

	// Extract system from the code and compare
	system, _ := extractCodeComponents(code)
	return fptypes.NewBoolean(system == csURL), nil
}

// resolveCodeRef resolves a CodeRef to a concrete Code value using library definitions.
func (e *Evaluator) resolveCodeRef(ref cqltypes.CodeRef) fptypes.Value {
	if e.ctx.Library == nil {
		return nil
	}
	for _, codeDef := range e.ctx.Library.Codes {
		if codeDef.Name == ref.Name {
			// Resolve the CodeSystem
			system := ""
			if codeDef.System != "" {
				for _, cs := range e.ctx.Library.CodeSystems {
					if cs.Name == codeDef.System {
						system = cs.ID
						break
					}
				}
			}
			return cqltypes.NewCode(system, codeDef.ID, codeDef.Display)
		}
	}
	return nil
}

func extractCodeComponents(v fptypes.Value) (system, code string) {
	switch c := v.(type) {
	case cqltypes.Code:
		return c.System, c.Code
	case fptypes.String:
		return "", c.Value()
	default:
		return "", v.String()
	}
}
