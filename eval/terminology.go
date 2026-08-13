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
	return e.inValueSetURL(code, vsURL)
}

// inValueSetURL asks the terminology provider about a code, once the value set
// has been resolved to a URL. Separate from evalInValueSet because a value set
// can be named through an include, where the name alone does not locate it.
func (e *Evaluator) inValueSetURL(code fptypes.Value, vsURL string) (fptypes.Value, error) {
	if code == nil {
		return nil, nil
	}
	// A value that carries several codes is in the set when any of them is: a
	// CodeableConcept names one idea with several codings, and a raw FHIR
	// CodeableConcept off a resource is the same shape before conversion.
	// Deciding by whichever code came first would answer by accident.
	if members, ok := codeMembers(code); ok {
		anyKnown := false
		for _, c := range members {
			result, err := e.inValueSetURL(c, vsURL)
			if err != nil {
				return nil, err
			}
			if result == nil {
				continue
			}
			anyKnown = true
			if isTrue(result) {
				return fptypes.NewBoolean(true), nil
			}
		}
		if !anyKnown {
			return nil, nil
		}
		return fptypes.NewBoolean(false), nil
	}

	system, codeVal := extractCodeComponents(code)
	if codeVal == "" {
		return nil, nil
	}
	if e.ctx.TerminologyProvider != nil {
		result, err := e.ctx.TerminologyProvider.InValueSet(e.ctx.GoCtx, codeVal, system, vsURL)
		if observer, ok := e.ctx.TraceListener.(TerminologyObserver); ok {
			observer.OnTerminologyCheck(codeVal, system, vsURL, result, err)
		}
		if err != nil {
			return nil, fmt.Errorf("terminology check failed for ValueSet %s: %w", vsURL, err)
		}
		return fptypes.NewBoolean(result), nil
	}
	// Without a TerminologyProvider, we can only check in-memory ValueSets
	return nil, nil
}

// includedValueSetURL resolves `alias.name` to the URL of a value set declared
// by an included library.
func (e *Evaluator) includedValueSetURL(alias, name string) (string, bool) {
	lib, ok := e.ctx.IncludedLibraries[alias]
	if !ok || lib == nil {
		return "", false
	}
	for _, vs := range lib.ValueSets {
		if vs.Name == name {
			return vs.ID, true
		}
	}
	return "", false
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

// codeMembers reports the individual codes a multi-valued terminology value
// carries, and whether it is one at all. A Concept and a raw FHIR
// CodeableConcept are the same idea before and after conversion; a list is what
// a repeating element resolves to.
func codeMembers(v fptypes.Value) ([]fptypes.Value, bool) {
	switch c := v.(type) {
	case cqltypes.Concept:
		out := make([]fptypes.Value, 0, len(c.Codes))
		for _, code := range c.Codes {
			out = append(out, code)
		}
		return out, true
	case cqltypes.List:
		return c.Values, true
	case *fptypes.ObjectValue:
		// A CodeableConcept carries its codings under `coding`; a Coding does
		// not, and is a single code.
		if codings := c.GetCollection("coding"); codings.Count() > 0 {
			return codings, true
		}
	}
	return nil, false
}

// extractCodeComponents pulls the system and code out of a value.
//
// It answers an empty code for anything it does not recognize, rather than
// stringifying it. The previous fallback rendered a whole value with String(),
// so a raw FHIR Coding reached the terminology server as the JSON text
// `{"system":"http://loinc.org","code":"8480-6"}` in the code field: a query no
// server can answer, and resource content sent outward to boot.
func extractCodeComponents(v fptypes.Value) (system, code string) {
	switch c := v.(type) {
	case cqltypes.Code:
		return c.System, c.Code
	case fptypes.String:
		return "", c.Value()
	case *fptypes.ObjectValue:
		// A raw FHIR Coding, as it comes off a resource before FHIRHelpers has
		// converted it.
		return objectField(c, "system"), objectField(c, "code")
	default:
		return "", ""
	}
}

// objectField reads a single scalar field off a FHIR object, or "" when it is
// absent or repeats.
func objectField(obj *fptypes.ObjectValue, name string) string {
	values := obj.GetCollection(name)
	if values.Count() != 1 {
		return ""
	}
	if s, ok := values[0].(fptypes.String); ok {
		return s.Value()
	}
	return ""
}
