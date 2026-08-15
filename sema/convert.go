package sema

// Model is what the semantic phase needs to know about a data model. It is a
// narrow view of model.StaticModelInfo, declared here so the phase can be
// exercised against a hand-built model and so the dependency points one way.
type Model interface {
	// ElementType returns the type of "Type.element" — already a sema type, so
	// that a list-valued or choice-valued element arrives as List<T> or
	// Choice<…> rather than as a name the caller has to interpret.
	ElementType(typeName, element string) (Type, bool)

	// IsSubtypeOf reports whether one model type descends from another.
	IsSubtypeOf(concrete, target string) bool

	// ConversionsFrom lists the conversions the model declares from a type,
	// each naming a target type and the function that performs it.
	ConversionsFrom(from string) []ModelConversion

	// HasType reports whether the model declares a type by this name, which is
	// what decides whether an unqualified name means the model's type or CQL's
	// own.
	HasType(name string) bool

	// ContextType maps a context name to the type retrieved for it.
	ContextType(contextName string) string
}

// ModelConversion is one conversionInfo entry: FHIR.Period converts to
// Interval<System.DateTime> through FHIRHelpers.ToInterval.
type ModelConversion struct {
	To       string
	Function string
}

// Conversion records how one type reaches another, and what it costs.
//
// Cost is what overload resolution compares. The absolute numbers mean nothing;
// their order is the whole content: an exact match must beat a subtype, which
// must beat an implicit conversion, which must beat wrapping a value in a list.
// Without it, `ToString(x)` picking between 251 overloads would resolve by
// whichever the map happened to yield first.
type Conversion struct {
	Cost     int
	Function string // set when the model declares a function for it
	Target   Type   // the type arrived at, which may be wider than requested
}

// Conversion costs, cheapest first. See Conversion.Cost.
const (
	costExact    = 0 // the same type
	costSubtype  = 1 // FHIR.Encounter where FHIR.DomainResource is wanted
	costChoice   = 2 // one branch of a Choice, or anything where Any is wanted
	costImplicit = 3 // Integer to Decimal, Date to DateTime, Code to Concept
	costModel    = 4 // FHIR.Period to Interval<DateTime>, via a declared function
	costPromote  = 5 // T to List<T>, and back
)

// implicitConversions are the system-to-system conversions the specification
// declares, keyed by the type converted from.
//
// Date to DateTime is in this table and matters more than it looks: without it
// comparing a birth date against a measurement period is a type error, and with
// it the comparison carries the uncertainty rules that a missing time implies.
var implicitConversions = map[string][]*Named{
	"System.Integer":  {Long, Decimal, Quantity},
	"System.Long":     {Decimal, Quantity},
	"System.Decimal":  {Quantity},
	"System.Date":     {DateTime},
	"System.Code":     {Concept},
	"System.ValueSet": {Vocabulary},
}

// Convertible reports how `from` reaches `to`, if it can.
//
// Unknown converts to and from anything at no cost. That is what keeps one
// unresolved name from being reported again at every operator that touches it:
// the phase already said what was wrong, once, where it was wrong.
func Convertible(from, to Type, m Model) (Conversion, bool) {
	if IsUnknown(from) || IsUnknown(to) {
		return Conversion{Cost: costExact, Target: to}, true
	}
	if Equal(from, to) {
		return Conversion{Cost: costExact, Target: to}, true
	}
	if isAny(to) {
		return Conversion{Cost: costChoice, Target: to}, true
	}
	// Any converts *into* anything as well, because the expression that has it
	// is nearly always `null`, and null is a value of every type. Refusing it
	// makes `if x is null then null else 1 'mg'` a type error.
	if isAny(from) {
		return Conversion{Cost: costChoice, Target: to}, true
	}

	// A choice converts if any of its branches does, and costs what the
	// cheapest branch costs plus the price of having had to choose.
	if c, ok := from.(*Choice); ok {
		best, found := Conversion{}, false
		for _, branch := range c.Types {
			conv, ok := Convertible(branch, to, m)
			if !ok {
				continue
			}
			if !found || conv.Cost < best.Cost {
				best, found = conv, true
			}
		}
		if found {
			best.Cost += costChoice
			return best, true
		}
		return Conversion{}, false
	}

	// Anything converts into a choice that has a branch it converts to.
	if c, ok := to.(*Choice); ok {
		for _, branch := range c.Types {
			if conv, ok := Convertible(from, branch, m); ok {
				conv.Cost += costChoice
				conv.Target = to
				return conv, true
			}
		}
		return Conversion{}, false
	}

	if conv, ok := convertNamed(from, to, m); ok {
		return conv, true
	}

	// Lists and intervals convert when their element types do, at the cost of
	// converting the elements — a List<Integer> is a List<Decimal> exactly as
	// far as an Integer is a Decimal.
	switch f := from.(type) {
	case *List:
		if t, ok := to.(*List); ok {
			if conv, ok := Convertible(f.Element, t.Element, m); ok {
				return Conversion{Cost: conv.Cost, Target: to}, true
			}
		}
	case *Interval:
		if t, ok := to.(*Interval); ok {
			if conv, ok := Convertible(f.Point, t.Point, m); ok {
				return Conversion{Cost: conv.Cost, Target: to}, true
			}
		}
	case *Tuple:
		if t, ok := to.(*Tuple); ok && tupleConvertible(f, t, m) {
			return Conversion{Cost: costImplicit, Target: to}, true
		}
	}

	// Promotion and demotion, last because they are the widest reach: a value
	// where a list is wanted, and a one-element list where a value is.
	if t, ok := to.(*List); ok {
		if conv, ok := Convertible(from, t.Element, m); ok {
			return Conversion{Cost: conv.Cost + costPromote, Target: to}, true
		}
	}
	if f, ok := from.(*List); ok {
		if conv, ok := Convertible(f.Element, to, m); ok {
			return Conversion{Cost: conv.Cost + costPromote, Target: to}, true
		}
	}
	return Conversion{}, false
}

// convertNamed handles the named-to-anything cases: subtyping within a model,
// the system conversions the specification declares, and the ones a model
// document declares for its own types.
func convertNamed(from, to Type, m Model) (Conversion, bool) {
	f, ok := from.(*Named)
	if !ok {
		return Conversion{}, false
	}

	if t, ok := to.(*Named); ok {
		for _, candidate := range implicitConversions[f.String()] {
			if Equal(candidate, t) {
				return Conversion{Cost: costImplicit, Target: to}, true
			}
		}
		// Two hops: Integer reaches Quantity through Decimal, and an author
		// who wrote neither should not have to care.
		for _, mid := range implicitConversions[f.String()] {
			for _, candidate := range implicitConversions[mid.String()] {
				if Equal(candidate, t) {
					return Conversion{Cost: costImplicit + 1, Target: to}, true
				}
			}
		}
		if m != nil && f.Model == t.Model && f.Model != "System" && m.IsSubtypeOf(f.String(), t.String()) {
			return Conversion{Cost: costSubtype, Target: to}, true
		}
	}

	// What the model declares about its own types: FHIR.Period becomes an
	// Interval<System.DateTime> by calling FHIRHelpers.ToInterval. This is the
	// same table eval/conversion.go consults at evaluation, read here with the
	// target type in hand — which is what lets it choose between the several
	// conversions a type may declare, and the evaluator cannot.
	if m != nil && f.Model != "System" {
		for _, mc := range m.ConversionsFrom(f.String()) {
			target := ParseTypeName(mc.To)
			if Equal(target, to) {
				return Conversion{Cost: costModel, Function: mc.Function, Target: to}, true
			}
			// The declared target may itself need one more step — a FHIR code
			// becomes a System.String, and a String is not a Concept, so stop
			// short of chaining model conversions into each other.
			if conv, ok := Convertible(target, to, m); ok && conv.Function == "" {
				return Conversion{Cost: costModel + conv.Cost, Function: mc.Function, Target: to}, true
			}
		}
	}
	return Conversion{}, false
}

// isAny reports whether a type is System.Any, the supertype every value has.
func isAny(t Type) bool {
	n, ok := t.(*Named)
	return ok && n.Model == "System" && n.Name == "Any"
}

// tupleConvertible reports whether one tuple can stand in for another: same
// field names, each field convertible.
func tupleConvertible(from, to *Tuple, m Model) bool {
	if len(from.Elements) != len(to.Elements) {
		return false
	}
	for i := range from.Elements {
		if from.Elements[i].Name != to.Elements[i].Name {
			return false
		}
		if _, ok := Convertible(from.Elements[i].Type, to.Elements[i].Type, m); !ok {
			return false
		}
	}
	return true
}

// Common returns a type both arguments convert to: what the branches of an
// if-then-else, or the elements of a list literal, together are.
//
// Where neither converts to the other it answers a Choice rather than giving
// up. `{ 1, 'a' }` is a List<Choice<Integer, String>> in CQL, not an error, and
// an element accessed from it is a choice the author has to narrow with `as`.
func Common(a, b Type, m Model) Type {
	switch {
	case IsUnknown(a):
		return b
	case IsUnknown(b):
		return a
	case Equal(a, b):
		return a
	// Any is what a `null` branch has, and it says nothing about the type of
	// the expression it is a branch of: `if x is null then null else 1` is an
	// Integer, and answering Any would lose that all the way up the tree — it
	// is what made every FHIRHelpers conversion function return Any.
	case isAny(a):
		return b
	case isAny(b):
		return a
	}
	toB, okB := Convertible(a, b, m)
	toA, okA := Convertible(b, a, m)
	switch {
	case okB && !okA:
		return b
	case okA && !okB:
		return a
	case okA && okB:
		// Both directions work — Integer and Long convert to each other's
		// neighbors — so keep the one that costs less to reach.
		if toB.Cost <= toA.Cost {
			return b
		}
		return a
	}
	return NewChoice([]Type{a, b})
}
