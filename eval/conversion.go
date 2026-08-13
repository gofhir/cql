package eval

import (
	"strings"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
)

// conversionResolver is the part of a model that declares how its own types
// convert to CQL system types. It is asked for optionally, so a model built by
// hand still works without one.
type conversionResolver interface {
	ConversionFunction(from, to string) (string, bool)
	ConversionsFrom(from string) map[string]string
}

// coerceToSystem applies the conversion the model declares for a FHIR value,
// so that an operator expecting a CQL system type gets one.
//
// The reference implementation inserts these calls in the translator, using the
// static types it inferred. Without a semantic phase they are applied here
// instead: when an operator is handed a FHIR type it cannot work with, the model
// is asked what converts it and the declared function is invoked. The model
// declares 264 of them — FHIR.Period to Interval<System.DateTime> through
// FHIRHelpers.ToInterval, and so on.
//
// This has a ceiling the plan is explicit about: with no static types there is
// no way to choose between two applicable conversions by the return type the
// context wants. Where the model declares exactly one conversion from a type,
// that ambiguity does not arise, and that covers the clinical types measures
// actually use. Anything else is left alone rather than guessed at.
func (e *Evaluator) coerceToSystem(v fptypes.Value) fptypes.Value {
	obj, ok := v.(*fptypes.ObjectValue)
	if !ok || obj == nil {
		return v
	}
	resolver, ok := e.ctx.ModelInfo.(conversionResolver)
	if !ok {
		return v
	}
	from := "FHIR." + obj.Type()
	targets := resolver.ConversionsFrom(from)
	if len(targets) != 1 {
		// None declared, or more than one and nothing to choose between them.
		return v
	}
	var fn string
	for _, name := range targets {
		fn = name
	}
	converted, err := e.callConversion(fn, v)
	if err != nil || converted == nil {
		return v
	}
	return converted
}

// callConversion invokes a conversion function named "Library.Function".
//
// The library has to be reachable: a measure that converts is one that includes
// FHIRHelpers, which is what the model names. When it is not included the value
// is returned unchanged and the operator fails as it did before — quietly
// pulling in a library the author never mentioned would be worse than not
// converting.
func (e *Evaluator) callConversion(qualified string, arg fptypes.Value) (fptypes.Value, error) {
	dot := strings.LastIndex(qualified, ".")
	if dot < 0 {
		return nil, nil
	}
	libName, fnName := qualified[:dot], qualified[dot+1:]

	lib := e.conversionLibrary(libName)
	if lib == nil {
		return nil, nil
	}
	overloads := e.ctx.functionRegistry(lib)[fnName]
	if len(overloads) == 0 {
		return nil, nil
	}
	fd := e.resolveOverloadByValues(overloads, []fptypes.Value{arg})
	if fd == nil {
		return nil, nil
	}
	return e.runFunction(fd, []fptypes.Value{arg}, lib)
}

// conversionLibrary finds the library a conversion names, by the alias it was
// included under or by its own name.
func (e *Evaluator) conversionLibrary(name string) *ast.Library {
	if lib, ok := e.ctx.IncludedLibraries[name]; ok {
		return lib
	}
	for alias, lib := range e.ctx.IncludedLibraries {
		if lib != nil && lib.Identifier != nil && lib.Identifier.Name == name {
			_ = alias
			return lib
		}
	}
	if lib, ok := e.ctx.LoadedLibraries[name]; ok {
		return lib
	}
	return nil
}
