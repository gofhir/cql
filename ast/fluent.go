package ast

// FluentOverloads is the overloads of a function that are declared fluent,
// the only ones a method-style call `x.F()` can reach: the same slice when
// every one is, as for the functions written to be called so, and a new one
// otherwise. The checker and the evaluator both pick from it, so that they
// agree on what such a call is.
func FluentOverloads(overloads []*FunctionDef) []*FunctionDef {
	all := true
	for _, fd := range overloads {
		all = all && fd.Fluent
	}
	if all {
		return overloads
	}
	var fluent []*FunctionDef
	for _, fd := range overloads {
		if fd.Fluent {
			fluent = append(fluent, fd)
		}
	}
	return fluent
}
