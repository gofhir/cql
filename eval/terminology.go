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

// evalAnyInValueSet checks if any code in a list is a member of the given ValueSet.
func (e *Evaluator) evalAnyInValueSet(codes fptypes.Value, vsRefName string) (fptypes.Value, error) {
	if codes == nil {
		return nil, nil
	}
	list, ok := codes.(cqltypes.List)
	if !ok {
		// Single code: delegate to evalInValueSet
		return e.evalInValueSet(codes, &ast.IdentifierRef{Name: vsRefName})
	}
	if len(list.Values) == 0 {
		return fptypes.NewBoolean(false), nil
	}
	vsRef := &ast.IdentifierRef{Name: vsRefName}
	for _, code := range list.Values {
		if code == nil {
			continue
		}
		result, err := e.evalInValueSet(code, vsRef)
		if err != nil {
			return nil, err
		}
		if isTrue(result) {
			return fptypes.NewBoolean(true), nil
		}
	}
	return fptypes.NewBoolean(false), nil
}

// evalAnyInCodeSystem checks if any code in a list is a member of the given CodeSystem.
func (e *Evaluator) evalAnyInCodeSystem(codes fptypes.Value, csRefName string) (fptypes.Value, error) {
	if codes == nil {
		return nil, nil
	}
	list, ok := codes.(cqltypes.List)
	if !ok {
		// Single code: delegate to evalInCodeSystem
		return e.evalInCodeSystem(codes, &ast.IdentifierRef{Name: csRefName})
	}
	if len(list.Values) == 0 {
		return fptypes.NewBoolean(false), nil
	}
	csRef := &ast.IdentifierRef{Name: csRefName}
	for _, code := range list.Values {
		if code == nil {
			continue
		}
		result, err := e.evalInCodeSystem(code, csRef)
		if err != nil {
			return nil, err
		}
		if isTrue(result) {
			return fptypes.NewBoolean(true), nil
		}
	}
	return fptypes.NewBoolean(false), nil
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
