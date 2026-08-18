package sema

import "strings"

// systemFunctionType returns what one of CQL's own functions answers, given
// what it was called with.
//
// The table covers the functions this engine implements, matched
// case-insensitively as the evaluator matches them. It is deliberately partial:
// a name it does not know yields false, and the caller reports that as a
// warning rather than a refusal, because a library may reach a function through
// an include this pass cannot see.
func systemFunctionType(name string, args []Type, m Model) (Type, bool) {
	if rule, ok := systemFunctions[strings.ToLower(name)]; ok {
		return rule(args, m), true
	}
	return nil, false
}

// A rule works out a return type from the argument types. Most ignore them.
type funcRule func(args []Type, m Model) Type

// systemArgFunctions are the functions that only work on CQL's own types, so an
// argument drawn from a data model has to be converted before they see it.
//
// The table above gives return types and not operand types, which is why an
// argument needing conversion to reach one is normally left alone. For the
// aggregates that is not good enough: summing a list of FHIR quantities is what
// a measure does all day, and Sum walking straight past the ones it does not
// recognize answers with a number that is quietly too small.
var systemArgFunctions = map[string]bool{
	"sum": true, "product": true, "avg": true, "median": true,
	"min": true, "max": true, "mode": true, "geometricmean": true,
	"stddev": true, "variance": true, "populationstddev": true,
	"populationvariance": true, "alltrue": true, "anytrue": true,
}

// wantsSystemArguments reports whether a function needs its arguments in CQL's
// own types.
func wantsSystemArguments(name string) bool {
	return systemArgFunctions[strings.ToLower(name)]
}

// systemFormOf is the CQL type a model type stands for, as the model declares
// it: FHIR.Quantity is a System.Quantity, and a list of them is a list of those.
// It answers nil when the type is already CQL's own, or when the model declares
// no conversion for it.
func systemFormOf(t Type, m Model) Type {
	if m == nil {
		return nil
	}
	switch x := t.(type) {
	case *List:
		if inner := systemFormOf(x.Element, m); inner != nil {
			return &List{Element: inner}
		}
	case *Interval:
		if inner := systemFormOf(x.Point, m); inner != nil {
			return &Interval{Point: inner}
		}
	case *Named:
		if x.Model == systemModel {
			return nil
		}
		for _, mc := range m.ConversionsFrom(x.String()) {
			if target := ParseTypeName(mc.To); IsSystem(target) {
				return target
			}
		}
	}
	return nil
}

// fixed returns the same type whatever the arguments.
func fixed(t Type) funcRule { return func([]Type, Model) Type { return t } }

// argument answers in the first argument's own type, which is what the
// functions that preserve what they were given do — Abs, Distinct, Message.
func argument() funcRule {
	return func(args []Type, _ Model) Type {
		if len(args) > 0 {
			return args[0]
		}
		return Unknown
	}
}

// elementOf answers with the element type of a list argument, for the functions
// that pick one out of it.
func elementOf() funcRule {
	return func(args []Type, _ Model) Type {
		if len(args) == 0 {
			return Unknown
		}
		if l, ok := args[0].(*List); ok {
			return l.Element
		}
		if IsUnknown(args[0]) {
			return Unknown
		}
		// Given a single value where a list was expected, the answer is that
		// value: CQL promotes it to a one-element list, and the element of that
		// is what came in.
		return args[0]
	}
}

// numericOf answers in the argument's own numeric type where it has one, and
// Decimal otherwise — which is what the aggregates that average do.
func numericOf() funcRule {
	return func(args []Type, _ Model) Type {
		if len(args) == 0 {
			return Unknown
		}
		t := args[0]
		if l, ok := t.(*List); ok {
			t = l.Element
		}
		if isNumeric(t) {
			return t
		}
		return Decimal
	}
}

// commonOf answers the type all the arguments have in common, which is what
// Coalesce does.
func commonOf(args []Type, m Model) Type {
	result := Unknown
	for _, a := range args {
		result = Common(result, a, m)
	}
	return result
}

var systemFunctions = map[string]funcRule{
	// --- Aggregates over a list ---
	"count":              fixed(Integer),
	"exists":             fixed(Boolean),
	"first":              elementOf(),
	"last":               elementOf(),
	"singletonfrom":      elementOf(),
	"min":                elementOf(),
	"max":                elementOf(),
	"mode":               elementOf(),
	"sum":                numericOf(),
	"product":            numericOf(),
	"avg":                numericOf(),
	"median":             numericOf(),
	"geometricmean":      fixed(Decimal),
	"stddev":             fixed(Decimal),
	"variance":           fixed(Decimal),
	"populationstddev":   fixed(Decimal),
	"populationvariance": fixed(Decimal),
	"alltrue":            fixed(Boolean),
	"anytrue":            fixed(Boolean),

	// --- List shaping ---
	"distinct": argument(),
	"tail":     argument(),
	"take":     argument(),
	"skip":     argument(),
	"sublist":  argument(),
	"slice":    argument(),
	"where":    argument(),
	"indexof":  fixed(Integer),
	"flatten":  flattenRule,
	"expand":   argument(),
	"collapse": argument(),
	"indexer":  elementOf(),

	// --- Strings ---
	"tostring":       fixed(String),
	"upper":          fixed(String),
	"lower":          fixed(String),
	"substring":      fixed(String),
	"combine":        fixed(String),
	"concatenate":    fixed(String),
	"replacematches": fixed(String),
	"startswith":     fixed(Boolean),
	"endswith":       fixed(Boolean),
	"matches":        fixed(Boolean),
	"positionof":     fixed(Integer),
	"lastpositionof": fixed(Integer),
	"split":          fixed(&List{Element: String}),
	"splitonmatches": fixed(&List{Element: String}),
	"length":         fixed(Integer),
	"size":           fixed(Integer),

	// --- Numbers ---
	"abs":         argument(),
	"ceiling":     fixed(Integer),
	"floor":       fixed(Integer),
	"truncate":    fixed(Integer),
	"round":       fixed(Decimal),
	"ln":          fixed(Decimal),
	"log":         fixed(Decimal),
	"exp":         fixed(Decimal),
	"power":       argument(),
	"precision":   fixed(Integer),
	"successor":   argument(),
	"predecessor": argument(),

	// --- Dates and times ---
	"now":       fixed(DateTime),
	"today":     fixed(Date),
	"timeofday": fixed(Time),
	"date":      fixed(Date),
	"datetime":  fixed(DateTime),
	"time":      fixed(Time),
	"timezone":  fixed(Decimal),

	"year": fixed(Integer), "years": fixed(Integer),
	"month": fixed(Integer), "months": fixed(Integer),
	"week": fixed(Integer), "weeks": fixed(Integer),
	"day": fixed(Integer), "days": fixed(Integer),
	"hour": fixed(Integer), "hours": fixed(Integer),
	"minute": fixed(Integer), "minutes": fixed(Integer),
	"second": fixed(Integer), "seconds": fixed(Integer),
	"millisecond": fixed(Integer), "milliseconds": fixed(Integer),

	"ageinyears":           fixed(Integer),
	"ageinyearsat":         fixed(Integer),
	"ageinmonths":          fixed(Integer),
	"ageinmonthsat":        fixed(Integer),
	"ageinweeks":           fixed(Integer),
	"ageindays":            fixed(Integer),
	"calculateageinyears":  fixed(Integer),
	"calculateageinmonths": fixed(Integer),
	"calculateageinweeks":  fixed(Integer),
	"calculateageindays":   fixed(Integer),

	// --- Conversions ---
	"toboolean":  fixed(Boolean),
	"tointeger":  fixed(Integer),
	"tolong":     fixed(Long),
	"todecimal":  fixed(Decimal),
	"todate":     fixed(Date),
	"todatetime": fixed(DateTime),
	"totime":     fixed(Time),
	"toquantity": fixed(Quantity),
	"toratio":    fixed(Ratio),
	"toconcept":  fixed(Concept),
	"tocode":     fixed(Code),

	"convertstoboolean":  fixed(Boolean),
	"convertstointeger":  fixed(Boolean),
	"convertstolong":     fixed(Boolean),
	"convertstodecimal":  fixed(Boolean),
	"convertstodate":     fixed(Boolean),
	"convertstodatetime": fixed(Boolean),
	"convertstotime":     fixed(Boolean),
	"convertstoquantity": fixed(Boolean),
	"convertstoratio":    fixed(Boolean),
	"convertstostring":   fixed(Boolean),
	"canconvertquantity": fixed(Boolean),
	"convertquantity":    fixed(Quantity),

	// --- Clinical and terminology ---
	"code":            fixed(Code),
	"codes":           fixed(&List{Element: Code}),
	"concept":         fixed(Concept),
	"display":         fixed(String),
	"unit":            fixed(String),
	"version":         fixed(String),
	"system":          fixed(String),
	"value":           valueRule,
	"subsumes":        fixed(Boolean),
	"subsumedby":      fixed(Boolean),
	"anyinvalueset":   fixed(Boolean),
	"anyincodesystem": fixed(Boolean),
	"expandvalueset":  fixed(&List{Element: Code}),
	"invalueset":      fixed(Boolean),

	// --- Interval and boundary ---
	"width":        fixed(Quantity),
	"lowboundary":  argument(),
	"highboundary": argument(),

	// --- Everything else ---
	"coalesce":    commonOf,
	"isnull":      fixed(Boolean),
	"istrue":      fixed(Boolean),
	"isfalse":     fixed(Boolean),
	"not":         fixed(Boolean),
	"message":     argument(),
	"children":    fixed(&List{Element: Any}),
	"descendents": fixed(&List{Element: Any}),
	"select":      argument(),
	"quantity":    fixed(Quantity),
	"string":      fixed(String),
	"boolean":     fixed(Boolean),
	"integer":     fixed(Integer),
	"long":        fixed(Long),
	"decimal":     fixed(Decimal),
	"null":        fixed(Any),
	"true":        fixed(Boolean),
	"false":       fixed(Boolean),
}

// flattenRule strips one level of list nesting.
func flattenRule(args []Type, _ Model) Type {
	if len(args) == 0 {
		return Unknown
	}
	if outer, ok := args[0].(*List); ok {
		if inner, ok := outer.Element.(*List); ok {
			return inner
		}
	}
	return args[0]
}

// valueRule types the `.value` accessor.
//
// On a FHIR primitive it is the identity accessor the engine added in Etapa 1 —
// `Enc.status.value` is the string inside the wrapper — so the answer is what
// the model says that wrapper converts to. On a Quantity it is the number.
func valueRule(args []Type, m Model) Type {
	if len(args) == 0 {
		return Unknown
	}
	if Equal(args[0], Quantity) {
		return Decimal
	}
	named, ok := args[0].(*Named)
	if !ok || m == nil || named.Model == systemModel {
		return Unknown
	}
	for _, mc := range m.ConversionsFrom(named.String()) {
		if target := ParseTypeName(mc.To); IsSystem(target) {
			return target
		}
	}
	return Unknown
}
