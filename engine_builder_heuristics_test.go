package cql

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"
)

func valueString(v fptypes.Value) string {
	if v == nil {
		return "null"
	}
	return v.String()
}

// TestBuilderTokenInspection covers the cases where the builder used to decide
// semantics with strings.Contains over the flattened node text, so any CQL that
// merely mentioned "not", "union", "distinct" or "properly" inside a literal or
// an identifier was compiled into the wrong AST.
func TestBuilderTokenInspection(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want string
	}{
		{"is null with literal containing not", `'not' is null`, "false"},
		{"is not null with literal containing not", `'not' is not null`, "true"},
		{"except with literal containing union", `{'union','x'} except {'x'}`, "{union}"},
		{"union with literal containing except", `{'except'} union {'x'}`, "{except, x}"},
		{"intersect with literal containing union", `{'union','x'} intersect {'x'}`, "{x}"},
		{"return with distinct inside nested call", `({1,1}) A return Count(distinct {A})`, "{1, 1}"},
		{"return distinct", `({1,1}) A return distinct A`, "{1}"},
		{"between with literal containing properly", `3 between 1 and 5`, "true"},
		{"sort by column", `({3,1,2}) A return A sort by $this`, "{1, 2, 3}"},
	}

	engine := NewEngine()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := "library Test version '1.0'\n\ndefine X: " + tt.expr + "\n"
			got, err := engine.EvaluateExpression(context.Background(), src, "X", nil, nil)
			if err != nil {
				t.Fatalf("evaluating %s: %v", tt.expr, err)
			}
			if s := valueString(got); s != tt.want {
				t.Errorf("%s = %s, want %s", tt.expr, s, tt.want)
			}
		})
	}
}

// TestBuilderKeywordsInFunctionBodies covers the two heuristics whose keyword
// could only ever appear in a nested expression: a function whose body is the
// string 'external' was compiled as an external declaration, and a return clause
// whose expression mentions 'distinct' was compiled as `return distinct`.
func TestBuilderKeywordsInFunctionBodies(t *testing.T) {
	src := `library Test version '1.0'

define function F(): 'external'
define BodyIsExternal: F()
define ReturnsDistinctLiteral: ({1,1}) A return 'distinct'
define ReturnsAllLiteral: ({1,1}) A return 'all'
`
	tests := []struct{ name, want string }{
		{"BodyIsExternal", "external"},
		{"ReturnsDistinctLiteral", "{distinct, distinct}"},
		{"ReturnsAllLiteral", "{all, all}"},
	}

	engine := NewEngine()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.EvaluateExpression(context.Background(), src, tt.name, nil, nil)
			if err != nil {
				t.Fatalf("evaluating %s: %v", tt.name, err)
			}
			if s := valueString(got); s != tt.want {
				t.Errorf("%s = %s, want %s", tt.name, s, tt.want)
			}
		})
	}
}

// TestBuilderIntervalBoundaries covers the boundary markers of an interval
// selector, which used to be read as `Interval[` anywhere in the flattened text
// and a trailing `]` — both of which an operand can supply on its own.
func TestBuilderIntervalBoundaries(t *testing.T) {
	tests := []struct{ expr, want string }{
		{`Interval[1, 5]`, "Interval[1, 5]"},
		{`Interval(1, 5)`, "Interval(1, 5)"},
		{`Interval[1, 5)`, "Interval[1, 5)"},
		{`Interval(1, 5]`, "Interval(1, 5]"},
		// operands that carry brackets of their own
		{`Interval({1,2}[0], 5)`, "Interval(1, 5)"},
		{`Interval(1, {4,5}[1]]`, "Interval(1, 5]"},
		{`Interval(start of Interval[2, 3], 9)`, "Interval(2, 9)"},
	}

	engine := NewEngine()
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			src := "library Test version '1.0'\n\ndefine X: " + tt.expr + "\n"
			got, err := engine.EvaluateExpression(context.Background(), src, "X", nil, nil)
			if err != nil {
				t.Fatalf("evaluating %s: %v", tt.expr, err)
			}
			if s := valueString(got); s != tt.want {
				t.Errorf("%s = %s, want %s", tt.expr, s, tt.want)
			}
		})
	}
}

// TestSortByUnknownColumn covers a sort key that names nothing: it used to
// resolve to its own name, giving every element the same key, so the query came
// back unsorted with no diagnostic at all.
func TestSortByUnknownColumn(t *testing.T) {
	src := `library Test version '1.0'

define X: ({ Tuple{n: 3}, Tuple{n: 1} }) A sort by nope
`
	_, err := NewEngine().EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err == nil {
		t.Fatal("sort by an unknown column should be an error, got none")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("error should name the offending key, got: %v", err)
	}
}

// TestIntervalOpenBoundaryStartEnd covers `start of` and `end of` on an open
// boundary, which used to return the excluded boundary value itself. `width of`
// already accounted for openness, so the two disagreed.
func TestIntervalOpenBoundaryStartEnd(t *testing.T) {
	tests := []struct{ expr, want string }{
		{`start of Interval[1, 5]`, "1"},
		{`end of Interval[1, 5]`, "5"},
		{`start of Interval(1, 5)`, "2"},
		{`end of Interval(1, 5)`, "4"},
		{`start of Interval[1, 5)`, "1"},
		{`end of Interval[1, 5)`, "4"},
		{`start of Interval(1, 5]`, "2"},
		{`end of Interval(1, 5]`, "5"},
		{`start of Interval(@2020-01-01, @2020-12-31)`, "2020-01-02"},
		{`end of Interval(@2020-01-01, @2020-12-31)`, "2020-12-30"},
		// start/end and width must agree: 4 - 2 == 2
		{`width of Interval(1, 5)`, "2"},
		{`end of Interval(1, 5) - start of Interval(1, 5)`, "2"},
	}

	engine := NewEngine()
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			src := "library Test version '1.0'\n\ndefine X: " + tt.expr + "\n"
			got, err := engine.EvaluateExpression(context.Background(), src, "X", nil, nil)
			if err != nil {
				t.Fatalf("evaluating %s: %v", tt.expr, err)
			}
			if s := valueString(got); s != tt.want {
				t.Errorf("%s = %s, want %s", tt.expr, s, tt.want)
			}
		})
	}
}

// TestBuilderSortByColumn covers `sort by <column>`, which used to fall through
// to the unknown-identifier behavior — every element produced the same key, so
// the list came back in its original order.
func TestBuilderSortByColumn(t *testing.T) {
	const tuples = `{ Tuple{n: 3, m: Tuple{k: 1}}, Tuple{n: 1, m: Tuple{k: 3}}, Tuple{n: 2, m: Tuple{k: 2}} }`
	src := `library Test version '1.0'

define ByThis: (` + tuples + `) A return A.n sort by $this
define ByColumn: (` + tuples + `) A return A.n sort by $this
define ByColumnDesc: (` + tuples + `) A sort by n desc
define ByAlias: ({3, 1, 2}) A sort by A
define ByPath: (` + tuples + `) A sort by m.k
define ByTwoKeys: ({ Tuple{a: 1, b: 2}, Tuple{a: 1, b: 1}, Tuple{a: 0, b: 9} }) A sort by a, b
define MultiByFirstAlias: from ({1,2}) A, ({20,10}) B sort by A
define MultiBySecondAlias: from ({1,2}) A, ({20,10}) B sort by B
define ByCallOfColumn: ({ Tuple{n: -3}, Tuple{n: 1}, Tuple{n: -2} }) A sort by Abs(n)
define ByArithmeticOfColumn: (` + tuples + `) A return A.n sort by $this * -1
define AliasShadowedByColumn: ({ Tuple{A: 3}, Tuple{A: 1}, Tuple{A: 2} }) A sort by A
define ByNullColumn: ({ Tuple{n: 3, z: null}, Tuple{n: 1, z: null} }) A sort by z
`
	tests := []struct {
		name string
		want string
	}{
		{"ByThis", "{1, 2, 3}"},
		{"ByColumn", "{1, 2, 3}"},
		{"ByColumnDesc", "{Tuple{m: Tuple{k: 1}, n: 3}, Tuple{m: Tuple{k: 2}, n: 2}, Tuple{m: Tuple{k: 3}, n: 1}}"},
		{"ByAlias", "{1, 2, 3}"},
		{"ByPath", "{Tuple{m: Tuple{k: 1}, n: 3}, Tuple{m: Tuple{k: 2}, n: 2}, Tuple{m: Tuple{k: 3}, n: 1}}"},
		{"ByTwoKeys", "{Tuple{a: 0, b: 9}, Tuple{a: 1, b: 1}, Tuple{a: 1, b: 2}}"},
		// A multi-source query yields a tuple keyed by alias, so its aliases are
		// columns of the result and must be resolved as such — including the
		// first one, which used to reach the comparator as the whole tuple.
		{"MultiByFirstAlias", "{Tuple{A: 1, B: 20}, Tuple{A: 1, B: 10}, Tuple{A: 2, B: 20}, Tuple{A: 2, B: 10}}"},
		{"MultiBySecondAlias", "{Tuple{A: 1, B: 10}, Tuple{A: 2, B: 10}, Tuple{A: 1, B: 20}, Tuple{A: 2, B: 20}}"},
		// A column is resolved wherever it appears in the key, not just when the
		// key is the bare identifier.
		{"ByCallOfColumn", "{Tuple{n: 1}, Tuple{n: -2}, Tuple{n: -3}}"},
		{"ByArithmeticOfColumn", "{3, 2, 1}"},
		// A column takes precedence over a query alias of the same name.
		{"AliasShadowedByColumn", "{Tuple{A: 1}, Tuple{A: 2}, Tuple{A: 3}}"},
		// A column that exists but is null is a stable no-op, not an error.
		{"ByNullColumn", "{Tuple{n: 3, z: null}, Tuple{n: 1, z: null}}"},
	}

	engine := NewEngine()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.EvaluateExpression(context.Background(), src, tt.name, nil, nil)
			if err != nil {
				t.Fatalf("evaluating %s: %v", tt.name, err)
			}
			if s := valueString(got); s != tt.want {
				t.Errorf("%s = %s, want %s", tt.name, s, tt.want)
			}
		})
	}
}

// resourceProvider serves fixed JSON resources.
type resourceProvider struct{ rows []string }

func (p resourceProvider) Retrieve(_ context.Context, _, _, _ string, _, _ interface{}) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, 0, len(p.rows))
	for _, r := range p.rows {
		out = append(out, json.RawMessage(r))
	}
	return out, nil
}

// TestSortByOptionalColumn covers a sort key that some elements carry and others
// do not, which is what any optional FHIR element looks like. Resolving the key
// strictly per element made the whole query fail on the resources that lacked it
// — and inconsistently so, since the qualified form `sort by P.birthDate` had
// always tolerated it.
func TestSortByOptionalColumn(t *testing.T) {
	prov := resourceProvider{rows: []string{
		`{"resourceType":"Patient","id":"a","birthDate":"1980-01-01"}`,
		`{"resourceType":"Patient","id":"b"}`,
		`{"resourceType":"Patient","id":"c","birthDate":"1970-01-01"}`,
	}}
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\n\ndefine X: [Patient] P sort by birthDate\n"

	got, err := NewEngine(WithDataProvider(prov)).
		EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err != nil {
		t.Fatalf("a column absent from one resource should sort as null, not fail: %v", err)
	}
	// Sorted ascending with the missing one last.
	s := valueString(got)
	if !strings.Contains(s, `"id":"c"`) || strings.Index(s, `"id":"c"`) > strings.Index(s, `"id":"a"`) {
		t.Errorf("expected c before a, got %s", s)
	}
	if strings.Index(s, `"id":"b"`) < strings.Index(s, `"id":"a"`) {
		t.Errorf("expected the resource without birthDate last, got %s", s)
	}
}

// TestSortByStillCatchesTypos guards the other half: a key that names nothing
// anywhere is still refused, decided once against the whole result rather than
// per element.
func TestSortByStillCatchesTypos(t *testing.T) {
	for _, src := range []string{
		"library T version '1.0'\n\ndefine X: ({ Tuple{n: 3}, Tuple{n: 1} }) A sort by nope\n",
		"library T version '1.0'\n\ndefine X: ({3, 1, 2}) A sort by nope\n",
	} {
		_, err := NewEngine().EvaluateExpression(context.Background(), src, "X", nil, nil)
		if err == nil {
			t.Errorf("a key naming nothing should still be refused: %s", src)
			continue
		}
		if !strings.Contains(err.Error(), "nope") {
			t.Errorf("the error should name the key, got: %v", err)
		}
	}
}

// TestSortWithNullElement covers a null in the sorted list. The sort scope used
// to be recognized by its element being non-nil, so a null element read as "not
// in a sort key" and the key fell through to the unknown-identifier fallback.
func TestSortWithNullElement(t *testing.T) {
	src := "library T version '1.0'\n\ndefine X: ({Tuple{n: 3}, null, Tuple{n: 1}}) A sort by n\n"
	got, err := NewEngine().EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err != nil {
		t.Fatalf("a null element should sort, not fail: %v", err)
	}
	if s := valueString(got); s != "{Tuple{n: 1}, Tuple{n: 3}, null}" {
		t.Errorf("X = %s, want {Tuple{n: 1}, Tuple{n: 3}, null}", s)
	}
}

// TestSortByRepeatingColumn covers a key that resolves to more than one value.
// A list has no ordering, so it sorts as null instead of failing the query on
// whichever pairs the sort happened to compare.
func TestSortByRepeatingColumn(t *testing.T) {
	prov := resourceProvider{rows: []string{
		`{"resourceType":"Patient","id":"a","name":[{"family":"Zed"}]}`,
		`{"resourceType":"Patient","id":"b","name":[{"family":"Abe"},{"family":"Bee"}]}`,
	}}
	for _, key := range []string{"name", "P.name"} {
		src := "library T version '1.0'\nusing FHIR version '4.0.1'\n\ndefine X: [Patient] P sort by " + key + "\n"
		if _, err := NewEngine(WithDataProvider(prov)).
			EvaluateExpression(context.Background(), src, "X", nil, nil); err != nil {
			t.Errorf("sort by %s: a repeating element should not fail the query: %v", key, err)
		}
	}
}

// TestSuccessorPredecessorRespectDatePrecision covers the Date branch, which
// always stepped one day: the successor of @2020-01 came back as @2020-01-02,
// claiming a precision the operand never had. `start of` and `end of` on an open
// boundary go through here, so they inherited it.
func TestSuccessorPredecessorRespectDatePrecision(t *testing.T) {
	tests := []struct{ expr, want string }{
		{`successor of @2019`, "2020"},
		{`successor of @2020-01`, "2020-02"},
		{`successor of @2020-01-01`, "2020-01-02"},
		{`predecessor of @2020-01`, "2019-12"},
		{`start of Interval(@2020-01, @2020-12)`, "2020-02"},
		{`end of Interval(@2019, @2021)`, "2020"},
	}
	engine := NewEngine()
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			src := "library T version '1.0'\n\ndefine X: " + tt.expr + "\n"
			got, err := engine.EvaluateExpression(context.Background(), src, "X", nil, nil)
			if err != nil {
				t.Fatalf("evaluating %s: %v", tt.expr, err)
			}
			if s := valueString(got); s != tt.want {
				t.Errorf("%s = %s, want %s", tt.expr, s, tt.want)
			}
		})
	}
}

// TestOpenBoundaryWithoutSuccessor covers a point type that has no successor.
// An open boundary cannot name its first included point, so it stands for
// itself rather than failing the expression.
func TestOpenBoundaryWithoutSuccessor(t *testing.T) {
	tests := []struct{ expr, want string }{
		{`start of Interval('a','z')`, "a"},
		{`end of Interval('a','z')`, "z"},
	}
	engine := NewEngine()
	for _, tt := range tests {
		got, err := engine.EvaluateExpression(context.Background(),
			"library T version '1.0'\n\ndefine X: "+tt.expr+"\n", "X", nil, nil)
		if err != nil {
			t.Fatalf("evaluating %s: %v", tt.expr, err)
		}
		if s := valueString(got); s != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, s, tt.want)
		}
	}
}
