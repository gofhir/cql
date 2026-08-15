// Package sema is the semantic phase: it decides, before anything is
// evaluated, what type every expression in a library has.
//
// Until now those decisions were made at evaluation. That works — the engine
// dispatches overloads by the runtime type of its arguments, and converts FHIR
// values where an operator needs a system type — but it can only decide with a
// value in hand. A type error therefore surfaces one row at a time, on data,
// instead of when the library is written; and where two conversions apply, the
// evaluator has nothing to choose between them by, because choosing needs the
// type the surrounding expression wants.
//
// This package answers those questions from the source alone, and reports
// everything it finds in one pass rather than stopping at the first problem.
package sema

import (
	"sort"
	"strings"
)

// Type is a CQL type as the semantic phase understands it.
//
// It is deliberately not ast.TypeSpecifier. That one records what an author
// wrote — a name, possibly unqualified, possibly wrong — whereas this records
// what an expression is, after names have been resolved against the model. The
// two look alike and mean different things, and conflating them is how an
// unresolved name silently becomes a type.
type Type interface {
	// String renders the type the way the CQL specification writes it:
	// qualified names, List<T>, Interval<T>, Tuple{...}, Choice<...>.
	String() string

	semaType()
}

// Named is a type identified by name inside a model: System.Integer,
// FHIR.Encounter, FHIR.ObservationStatus.
type Named struct {
	Model string // "System", "FHIR", …
	Name  string
}

func (*Named) semaType() {}

func (n *Named) String() string {
	if n.Model == "" {
		return n.Name
	}
	return n.Model + "." + n.Name
}

// List is List<Element>.
type List struct{ Element Type }

func (*List) semaType()        {}
func (l *List) String() string { return "List<" + l.Element.String() + ">" }

// Interval is Interval<Point>.
type Interval struct{ Point Type }

func (*Interval) semaType()        {}
func (i *Interval) String() string { return "Interval<" + i.Point.String() + ">" }

// TupleElement is one named field of a tuple type.
type TupleElement struct {
	Name string
	Type Type
}

// Tuple is a structured type: Tuple{name Type, …}.
//
// Elements are kept sorted by name so that two tuples written with their fields
// in different orders compare equal, which is what CQL says they are.
type Tuple struct{ Elements []TupleElement }

func (*Tuple) semaType() {}

func (t *Tuple) String() string {
	parts := make([]string, len(t.Elements))
	for i, e := range t.Elements {
		parts[i] = e.Name + " " + e.Type.String()
	}
	return "Tuple{" + strings.Join(parts, ", ") + "}"
}

// NewTuple builds a tuple type with its elements in canonical order.
func NewTuple(elements []TupleElement) *Tuple {
	sorted := make([]TupleElement, len(elements))
	copy(sorted, elements)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	return &Tuple{Elements: sorted}
}

// Element returns the type of a named field.
func (t *Tuple) Element(name string) (Type, bool) {
	for _, e := range t.Elements {
		if e.Name == name {
			return e.Type, true
		}
	}
	return nil, false
}

// Choice is Choice<A, B, …>: a value that is one of several types, which is how
// FHIR spells its [x] elements.
type Choice struct{ Types []Type }

func (*Choice) semaType() {}

func (c *Choice) String() string {
	parts := make([]string, len(c.Types))
	for i, t := range c.Types {
		parts[i] = t.String()
	}
	return "Choice<" + strings.Join(parts, ", ") + ">"
}

// NewChoice builds a choice, flattening nested choices and dropping duplicates.
// A choice of one is that one type: Choice<Integer> and Integer are the same
// thing, and keeping the wrapper would make every comparison against it fail.
func NewChoice(types []Type) Type {
	var flat []Type
	seen := map[string]bool{}
	var add func(Type)
	add = func(t Type) {
		if t == nil {
			return
		}
		if c, ok := t.(*Choice); ok {
			for _, inner := range c.Types {
				add(inner)
			}
			return
		}
		if key := t.String(); !seen[key] {
			seen[key] = true
			flat = append(flat, t)
		}
	}
	for _, t := range types {
		add(t)
	}
	switch len(flat) {
	case 0:
		return Unknown
	case 1:
		return flat[0]
	}
	return &Choice{Types: flat}
}

// unknownType is what an expression has when the phase could not work out its
// type: an unresolved name, an unmodeled element, a construct not yet inferred.
//
// It is not an error type. Its whole purpose is to stop one problem from
// becoming ten: every operator accepts it and yields it, so a single unresolved
// identifier reports once, at the identifier, instead of once at every
// expression that mentions it.
type unknownType struct{}

func (unknownType) semaType()      {}
func (unknownType) String() string { return "?" }

// Unknown is the singleton unknown type. Compare against it with IsUnknown.
var Unknown Type = unknownType{}

// IsUnknown reports whether a type could not be determined. A nil type counts:
// a node the phase never visited knows as little as one it gave up on.
func IsUnknown(t Type) bool {
	if t == nil {
		return true
	}
	_, ok := t.(unknownType)
	return ok
}

// The System types, as the specification names them.
var (
	Any        = &Named{Model: "System", Name: "Any"}
	Boolean    = &Named{Model: "System", Name: "Boolean"}
	Integer    = &Named{Model: "System", Name: "Integer"}
	Long       = &Named{Model: "System", Name: "Long"}
	Decimal    = &Named{Model: "System", Name: "Decimal"}
	String     = &Named{Model: "System", Name: "String"}
	Date       = &Named{Model: "System", Name: "Date"}
	DateTime   = &Named{Model: "System", Name: "DateTime"}
	Time       = &Named{Model: "System", Name: "Time"}
	Quantity   = &Named{Model: "System", Name: "Quantity"}
	Ratio      = &Named{Model: "System", Name: "Ratio"}
	Code       = &Named{Model: "System", Name: "Code"}
	Concept    = &Named{Model: "System", Name: "Concept"}
	Vocabulary = &Named{Model: "System", Name: "Vocabulary"}
	ValueSet   = &Named{Model: "System", Name: "ValueSet"}
	CodeSystem = &Named{Model: "System", Name: "CodeSystem"}
)

// Equal reports whether two types are the same type.
//
// It compares structurally rather than by pointer: a type parsed from a model
// document and one of the singletons above are the same System.Integer, and an
// identity comparison would say otherwise.
func Equal(a, b Type) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	switch x := a.(type) {
	case unknownType:
		_, ok := b.(unknownType)
		return ok
	case *Named:
		y, ok := b.(*Named)
		return ok && x.Model == y.Model && x.Name == y.Name
	case *List:
		y, ok := b.(*List)
		return ok && Equal(x.Element, y.Element)
	case *Interval:
		y, ok := b.(*Interval)
		return ok && Equal(x.Point, y.Point)
	case *Tuple:
		y, ok := b.(*Tuple)
		if !ok || len(x.Elements) != len(y.Elements) {
			return false
		}
		for i := range x.Elements {
			if x.Elements[i].Name != y.Elements[i].Name || !Equal(x.Elements[i].Type, y.Elements[i].Type) {
				return false
			}
		}
		return true
	case *Choice:
		y, ok := b.(*Choice)
		if !ok || len(x.Types) != len(y.Types) {
			return false
		}
		// Choices are unordered: Choice<Integer, String> is Choice<String,
		// Integer>. Compare as sets.
		for _, t := range x.Types {
			found := false
			for _, u := range y.Types {
				if Equal(t, u) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	return false
}

// ParseTypeName reads a type as a model document writes it: "System.Integer",
// "Interval<System.DateTime>", "List<FHIR.Coding>".
//
// The conversionInfo entries of the official ModelInfo are spelled this way —
// FHIR.Period converts to Interval<System.DateTime> — so inserting a conversion
// statically means knowing what its target type is, not just its name.
func ParseTypeName(name string) Type {
	name = strings.TrimSpace(name)
	if name == "" {
		return Unknown
	}
	if inner, ok := unwrap(name, "List<"); ok {
		return &List{Element: ParseTypeName(inner)}
	}
	if inner, ok := unwrap(name, "Interval<"); ok {
		return &Interval{Point: ParseTypeName(inner)}
	}
	if model, local, found := strings.Cut(name, "."); found {
		return &Named{Model: model, Name: local}
	}
	// An unqualified name in a model document means a System type; anywhere
	// else the caller has already qualified it.
	return &Named{Model: "System", Name: name}
}

// unwrap strips a "Wrapper<" prefix and its closing angle bracket.
func unwrap(name, prefix string) (string, bool) {
	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ">") {
		return "", false
	}
	return name[len(prefix) : len(name)-1], true
}

// IsSystem reports whether a type is one of the CQL system types, as opposed to
// one drawn from a data model.
func IsSystem(t Type) bool {
	n, ok := t.(*Named)
	return ok && n.Model == "System"
}

// Unqualified is the type's name without its model, which is how the reference
// translator writes result types in the ELM it emits.
func Unqualified(t Type) string {
	switch x := t.(type) {
	case *Named:
		return x.Name
	case *List:
		return "List<" + Unqualified(x.Element) + ">"
	case *Interval:
		return "Interval<" + Unqualified(x.Point) + ">"
	}
	if t == nil {
		return "?"
	}
	return t.String()
}
