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

// baseOf reports the type a type extends, taking either spelling of the name.
//
// It unqualifies defensively rather than assuming: callers reach it both with
// the name as the phase carries it (FHIR.Encounter) and with the name the
// document indexes (Encounter.Hospitalization), and unqualifying an already
// local one has to be a no-op. It was not, while the qualifier was cut at the
// last dot: Encounter.Hospitalization became Hospitalization, so a backbone
// element inherited nothing and every element it gets from Element or
// BackboneElement — extension among them — was reported missing.
func (a *modelInfoAdapter) baseOf(typeName string) (string, bool) {
	ti, ok := a.mi.TypeInfo(typeName)
	if !ok {
		if ti, ok = a.mi.TypeInfo(unqualify(typeName)); !ok {
			return "", false
		}
	}
	if ti.BaseName == "" {
		return "", false
	}
	return ti.BaseName, true
}

func (a *modelInfoAdapter) IsSubtypeOf(concrete, target string) bool {
	return a.mi.IsSubtypeOf(concrete, target)
}

// ConversionsFrom lists what a type converts to, inheriting from its base.
//
// The document declares a conversion where it is introduced, and FHIR's
// primitives are a hierarchy: `id` extends `string`, which is what converts to
// System.String. Asking only about the type itself found nothing for FHIR.id, so
// `'discharged: ' & Encounter.id` was reported as a String operator applied to
// something that is not one — while the evaluator converted it and answered
// correctly, since it walks the same hierarchy at runtime.
// The whole chain is collected rather than the nearest ancestor that declares
// anything: a subtype declaring one conversion would otherwise hide every
// conversion its base declares. The embedded 4.0.1 model has no such type today,
// which is exactly why it is worth not depending on.
func (a *modelInfoAdapter) ConversionsFrom(from string) []ModelConversion {
	var out []ModelConversion
	seenTarget := map[string]bool{}
	for name, depth := from, 0; depth < maxTypeHierarchyDepth; depth++ {
		for _, c := range a.mi.ConversionsFrom(name) {
			// A conversion declared closer to the type wins over the same
			// target declared further up.
			if seenTarget[c.To] {
				continue
			}
			seenTarget[c.To] = true
			out = append(out, ModelConversion{To: c.To, Function: c.Function})
		}
		base, ok := a.baseOf(name)
		if !ok {
			break
		}
		name = base
	}
	return out
}

// maxTypeHierarchyDepth bounds the walk up a base chain. FHIR's deepest is five
// or so; the limit is there because a document that declared a cycle would
// otherwise hang the phase rather than report anything.
const maxTypeHierarchyDepth = 32

// HasType reports whether the model declares this type, matched exactly: FHIR
// declares `integer` and not `Integer`, and treating the two as one would make
// every `x as Integer` refer to a FHIR primitive.
//
// Both spellings are tried, since a nested type's own name contains a dot:
// Encounter.Hospitalization is how the document indexes it, and unqualifying
// that leaves "Hospitalization", which the document does not have.
func (a *modelInfoAdapter) HasType(name string) bool {
	if _, ok := a.mi.TypeInfo(name); ok {
		return true
	}
	_, ok := a.mi.TypeInfo(unqualify(name))
	return ok
}

func (a *modelInfoAdapter) ContextType(contextName string) string {
	return a.mi.ContextType(contextName)
}

// unqualify drops the model qualifier from a type name, leaving the name the
// document indexes it under.
//
// The qualifier is the first segment, not everything before the last dot. FHIR
// names its backbone elements after the type that owns them —
// FHIR.Encounter.Hospitalization is one type, whose name is
// Encounter.Hospitalization — so cutting at the last dot left "Hospitalization",
// which the document has never heard of. Every element of every backbone
// element was reported missing on that basis: dischargeDisposition on a
// hospitalization, code and value on an Observation.component.
func unqualify(name string) string {
	if i := strings.Index(name, "."); i >= 0 {
		return name[i+1:]
	}
	return name
}
