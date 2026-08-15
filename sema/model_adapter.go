package sema

import (
	"strings"

	"github.com/gofhir/cql/model"
)

// FromModelInfo adapts the engine's model information to what the semantic
// phase asks of a model.
//
// The adaptation is not a formality. ModelInfo answers in the vocabulary of the
// document it was parsed from — an element is a name plus two booleans, and a
// type is a string — and the phase works in types. Turning "list of choice of
// FHIR.Quantity, System.String" into List<Choice<FHIR.Quantity, System.String>>
// once, here, is what keeps that shape out of every inference rule.
func FromModelInfo(mi *model.StaticModelInfo) Model {
	if mi == nil {
		return nil
	}
	return &modelInfoAdapter{mi: mi}
}

type modelInfoAdapter struct{ mi *model.StaticModelInfo }

// ElementType resolves one element of one type.
//
// The type name arrives qualified, as the phase carries it — FHIR.Encounter —
// and the document indexes its types unqualified, so the qualifier is dropped
// before the lookup. Getting this backwards is what broke every choice element
// when the model grew from 6 types to 931: the document spells its primitives
// in lower case, and a name compared without care matches nothing.
func (a *modelInfoAdapter) ElementType(typeName, element string) (Type, bool) {
	local := unqualify(typeName)
	info, ok := a.mi.ElementInfoByPath(local + "." + element)
	if !ok {
		// A subtype inherits its base's elements, and the document declares
		// each one only where it is introduced: Encounter.id lives on Resource.
		if base, found := a.baseOf(local); found {
			return a.ElementType(base, element)
		}
		return nil, false
	}
	return elementType(info), true
}

// elementType renders one element declaration as a type.
func elementType(info *model.ElementInfo) Type {
	var t Type
	switch {
	case info.IsChoice:
		branches := make([]Type, 0, len(info.ChoiceTypes))
		for _, name := range info.ChoiceTypes {
			branches = append(branches, ParseTypeName(name))
		}
		t = NewChoice(branches)
	case info.Type != "":
		t = ParseTypeName(info.Type)
	default:
		t = Unknown
	}
	if info.IsList {
		return &List{Element: t}
	}
	return t
}

func (a *modelInfoAdapter) baseOf(typeName string) (string, bool) {
	ti, ok := a.mi.TypeInfo(unqualify(typeName))
	if !ok || ti.BaseName == "" {
		return "", false
	}
	return ti.BaseName, true
}

func (a *modelInfoAdapter) IsSubtypeOf(concrete, target string) bool {
	return a.mi.IsSubtypeOf(concrete, target)
}

func (a *modelInfoAdapter) ConversionsFrom(from string) []ModelConversion {
	declared := a.mi.ConversionsFrom(from)
	out := make([]ModelConversion, len(declared))
	for i, c := range declared {
		out[i] = ModelConversion{To: c.To, Function: c.Function}
	}
	return out
}

// HasType reports whether the model declares this type, matched exactly: FHIR
// declares `integer` and not `Integer`, and treating the two as one would make
// every `x as Integer` refer to a FHIR primitive.
func (a *modelInfoAdapter) HasType(name string) bool {
	_, ok := a.mi.TypeInfo(unqualify(name))
	return ok
}

func (a *modelInfoAdapter) ContextType(contextName string) string {
	return a.mi.ContextType(contextName)
}

func unqualify(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}
	return name
}
