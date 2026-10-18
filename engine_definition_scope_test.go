package cql

import (
	"context"
	"testing"
)

// TestADefinitionEvaluatesInItsLibrarysScope covers a definition referenced
// from inside a query or a function. It evaluated in the scope it was
// referenced from, so an alias, a let or an operand of the same name stood in
// for a name of the definition — `({10}) Y return D`, with D: Y + 1, was {11} —
// and that answer was memoized as D for every reference after. And an alias
// of an enclosing query lost to a definition of its name once that definition
// had been memoized, the definitions being checked at each scope before the
// aliases of the next.
func TestADefinitionEvaluatesInItsLibrarysScope(t *testing.T) {
	// UnderAnAliasFirst comes before D, so that EvaluateLibrary meets D first
	// under the alias, and what it memoizes then is what DAfterward reads.
	const library = "library T version '1.0'\n" +
		"define UnderAnAliasFirst: ({10}) Y return D\n" +
		"define Y: 1\n" +
		"define D: Y + 1\n" +
		"define Q: ({ 2, 3 }) X return X + Y\n" +
		"define function F(Y Integer): D + Y\n" +
		"define DAfterward: D\n"
	for expr, want := range map[string]string{
		"({10}) Y return D":                         "{2}",
		"({1}) Z let Y: 20 return D":                "{2}",
		"({10, 20}) Y where D = 2":                  "{10, 20}",
		"({10}) Y return ({5}) Z return D":          "{{2}}",
		"F(30)":                                     "32",
		"({10}) Y return Y":                         "{10}",
		"(Q) Y return Sum((({ 3, 4 }) C return Y))": "{3, 4}",
		"({7}) Y let Z: Y + D return Z":             "{9}",
	} {
		v, err := NewEngine().EvaluateExpression(context.Background(), library+"define R: "+expr+"\n", "R", nil, nil)
		if err != nil || valueString(v) != want {
			t.Errorf("%s = %s, %v; want %s", expr, valueString(v), err, want)
		}
	}

	// Evaluated under an alias first, D must still be 2 for what reads it next.
	results, err := NewEngine().EvaluateLibrary(context.Background(), library, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := valueString(results["UnderAnAliasFirst"]); got != "{2}" {
		t.Errorf("UnderAnAliasFirst = %s; want {2}", got)
	}
	if got := valueString(results["DAfterward"]); got != "2" {
		t.Errorf("DAfterward = %s; want 2", got)
	}
}

// TestAFluentCallIsToAFluentFunction covers fluent calls. The source is the
// first operand — 5.AddK() bound nothing, and its body read whatever the
// caller had called x — and a name bound in scope is the source, not an
// include of the same name. A function that is not fluent cannot be called as
// one. An included library's fluent function of a system function's name
// does not stand in for the system function: 5.ToString() is '5'.
func TestAFluentCallIsToAFluentFunction(t *testing.T) {
	const included = "library Inc version '1.0'\n" +
		"define fluent function ToString(x String): 'inc:' + x\n" +
		"define fluent function Abs(x Integer): 999\n"
	const library = "library T version '1.0'\n" +
		"include Inc version '1.0' called Inc\n" +
		"define K: 3\n" +
		"define fluent function AddK(x Integer): x + K\n" +
		"define fluent function Plus(x Integer, y Integer): x + y + K\n" +
		"define fluent function Thrice(x Integer): x * 3\n" +
		"define function Local(x Integer): x + 100\n"
	engine := NewEngine(WithLibraryResolver(func(_ context.Context, _, _ string) (string, error) {
		return included, nil
	}))
	for expr, want := range map[string]string{
		"5.AddK()":                       "8",
		"({10}) x return 5.AddK()":       "{8}",
		"({10}) x return x.AddK()":       "{13}",
		"({10}) K return 5.AddK()":       "{8}",
		"5.Plus(1)":                      "9",
		"5.Thrice().AddK()":              "18",
		"({10}) Inc return Inc.Thrice()": "{30}",
		"5.ToString()":                   "5",
		"(-3).Abs()":                     "3",
	} {
		v, err := engine.EvaluateExpression(context.Background(), library+"define R: "+expr+"\n", "R", nil, nil)
		if err != nil || valueString(v) != want {
			t.Errorf("%s = %s, %v; want %s", expr, valueString(v), err, want)
		}
	}
	// A function of this library that is not fluent is refused, as the
	// reference translator refuses it, both when the library is checked and
	// when it runs — not passed over for a system function of the name.
	for _, expr := range []string{"5.Local()", "(-3).Abs()"} {
		src := library + "define function Abs(x Integer): 999\ndefine R: " + expr + "\n"
		if _, err := engine.EvaluateExpression(context.Background(), src, "R", nil, nil); err == nil {
			t.Errorf("%s was evaluated; a function that is not fluent cannot be called as one", expr)
		}
		diags, err := engine.Check(src)
		if err != nil || !diags.HasErrors() {
			t.Errorf("%s checked clean (%v, %v); want an error", expr, diags, err)
		}
	}
	// A query alias named like an include is the row to sema as well.
	if diags, err := engine.Check(library + "define R: ({10}) Inc return Inc.Thrice()\n"); err != nil || diags.HasErrors() {
		t.Errorf("({10}) Inc return Inc.Thrice() did not check clean: %v, %v", diags, err)
	}
}
