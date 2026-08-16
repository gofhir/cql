package compiler

import (
	"strings"
	"testing"

	"github.com/gofhir/cql/ast"
)

const pushdownPreamble = "library T version '1.0'\n" +
	"using FHIR version '4.0.1'\n" +
	"parameter \"Measurement Period\" Interval<DateTime>\n" +
	"valueset \"Diabetes\": 'http://example.org/vs/diabetes'\n" +
	"context Patient\n\n"

// retrieveOf digs out the retrieve a define's query reads from.
func retrieveOf(t *testing.T, body string) *ast.Retrieve {
	t.Helper()
	lib, err := Compile(pushdownPreamble + "define X: " + body + "\n")
	if err != nil {
		t.Fatalf("compiling %q: %v", body, err)
	}
	query, ok := lib.Statements[0].Expression.(*ast.Query)
	if !ok {
		t.Fatalf("expected a query, got %T", lib.Statements[0].Expression)
	}
	retrieve, ok := query.Sources[0].Source.(*ast.Retrieve)
	if !ok {
		t.Fatalf("expected a retrieve, got %T", query.Sources[0].Source)
	}
	return retrieve
}

// TestDateRangePushesDown covers the filter a provider can use. Only the ELM
// importer ever filled DateRange in, so a library evaluated from CQL text —
// which is what a server receives — fetched every row and filtered afterwards.
func TestDateRangePushesDown(t *testing.T) {
	for _, tt := range []struct {
		name, body, wantPath string
	}{
		{
			"during a parameter",
			`[Encounter] E where E.period during "Measurement Period"`,
			"period",
		},
		{
			"included in",
			`[Encounter] E where E.period included in "Measurement Period"`,
			"period",
		},
		{
			// An overlap admits more rows than during does, and the FHIR date
			// search a provider maps this to is an overlap anyway.
			"overlaps",
			`[Encounter] E where E.period overlaps "Measurement Period"`,
			"period",
		},
		{
			"a nested path",
			`[Encounter] E where E.period.start during "Measurement Period"`,
			"period.start",
		},
		{
			"in a stated interval",
			`[Condition] C where C.onset in Interval[@2020-01-01, @2020-12-31]`,
			"onset",
		},
		{
			// Every conjunct is implied by the whole clause, so narrowing by
			// one cannot drop a row the clause would have kept.
			"one conjunct among several",
			`[Encounter] E where E.status = 'finished' and E.period during "Measurement Period"`,
			"period",
		},
		{
			// `properly` admits fewer rows, so the query stays a subset of what
			// the pushed-down interval returns.
			"properly during",
			`[Encounter] E where E.period properly during "Measurement Period"`,
			"period",
		},
		{
			"during at a stated precision",
			`[Encounter] E where E.period during day of "Measurement Period"`,
			"period",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			retrieve := retrieveOf(t, tt.body)
			if retrieve.DatePath != tt.wantPath {
				t.Errorf("date path = %q, want %q", retrieve.DatePath, tt.wantPath)
			}
			if retrieve.DateRange == nil {
				t.Error("date range = nil, want the interval the query named")
			}
		})
	}
}

// TestDateRangeStaysPutWhenPushingItDownWouldChangeTheAnswer is the half of
// this that matters. A filter pushed down wrongly does not fail; it returns
// fewer rows, in silence, in a measure denominator.
func TestDateRangeStaysPutWhenPushingItDownWouldChangeTheAnswer(t *testing.T) {
	for _, tt := range []struct{ name, body, why string }{
		{
			"a disjunction",
			`[Encounter] E where E.period during "Measurement Period" or E.status = 'finished'`,
			"rows outside the interval still qualify through the other branch",
		},
		{
			"a disjunction nested under an and",
			`[Encounter] E where E.status = 'finished' and (E.period during "Measurement Period" or E.class = 'IMP')`,
			"the same, one level down",
		},
		{
			"more than one source",
			`from [Encounter] E, [Condition] C where E.period during "Measurement Period"`,
			"which retrieve to narrow stops having one answer",
		},
		{
			"an interval built from the row itself",
			`[Encounter] E where E.period during Interval[E.period.start, E.period.end]`,
			"the interval cannot be evaluated before the rows it is built from",
		},
		{
			"a value set membership",
			`[Condition] C where C.code in "Diabetes"`,
			"in is membership of any kind, and this one is a terminology question",
		},
		{
			"a path on another name",
			`[Encounter] E where Patient.birthDate during "Measurement Period"`,
			"the predicate does not constrain the retrieved rows",
		},
		{
			"before the interval",
			`[Encounter] E where E.period before "Measurement Period"`,
			"asking for the interval would exclude exactly the rows meant to be kept",
		},
		{
			"after the interval",
			`[Encounter] E where E.period after "Measurement Period"`,
			"the same, in the other direction",
		},
		{
			"includes rather than included in",
			`[Encounter] E where E.period includes "Measurement Period"`,
			"this constrains the interval, not the row",
		},
		{
			// These do meet the interval, so pushing down would be safe. They
			// are refused because the boundary work has not been done, and a
			// slow query beats a wrong one.
			"overlaps with a direction",
			`[Encounter] E where E.period overlaps after "Measurement Period"`,
			"the boundary for each direction has not been worked out",
		},
		{
			"starts before",
			`[Encounter] E where E.period starts before "Measurement Period"`,
			"the rows meant to be kept are outside the interval",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			retrieve := retrieveOf(t, tt.body)
			if retrieve.DateRange != nil || retrieve.DatePath != "" {
				t.Errorf("pushed down %q/%v — %s", retrieve.DatePath, retrieve.DateRange, tt.why)
			}
		})
	}
}

// The where clause is never removed. Pushing a filter down is a request a
// provider may ignore, and the FHIR date search it maps to answers with an
// overlap, which is a superset of what during means. Both are harmless only
// while the clause is still applied to what comes back.
func TestWhereClauseSurvivesThePushdown(t *testing.T) {
	lib, err := Compile(pushdownPreamble +
		"define X: [Encounter] E where E.period during \"Measurement Period\"\n")
	if err != nil {
		t.Fatalf("compiling: %v", err)
	}
	query, _ := lib.Statements[0].Expression.(*ast.Query)
	if query.Where == nil {
		t.Fatal("the where clause was removed, want it kept")
	}
	if _, ok := query.Where.(*ast.TimingExpression); !ok {
		t.Errorf("where clause is %T, want the timing expression untouched", query.Where)
	}
}

// A retrieve that already carries a date range keeps it: the ELM importer
// supplies one, and a second opinion from the where clause would overwrite
// what the translator decided.
func TestExistingDateRangeIsNotOverwritten(t *testing.T) {
	lib, err := Compile(pushdownPreamble +
		"define X: [Encounter] E where E.period during \"Measurement Period\"\n")
	if err != nil {
		t.Fatalf("compiling: %v", err)
	}
	query, _ := lib.Statements[0].Expression.(*ast.Query)
	retrieve, _ := query.Sources[0].Source.(*ast.Retrieve)

	sentinel := &ast.Literal{Value: "already here"}
	retrieve.DateRange = sentinel
	retrieve.DatePath = "chosen"
	pushDownDateRanges(lib)

	if retrieve.DateRange != ast.Expression(sentinel) || retrieve.DatePath != "chosen" {
		t.Errorf("date range replaced with %q/%v", retrieve.DatePath, retrieve.DateRange)
	}
}

// A query with no where clause, and a retrieve read outside a query at all,
// are both left alone rather than reached for.
func TestPushdownLeavesUnrelatedShapesAlone(t *testing.T) {
	for _, body := range []string{
		`[Encounter]`,
		`[Encounter] E return E.status`,
		`[Encounter] E where E.status = 'finished'`,
	} {
		lib, err := Compile(pushdownPreamble + "define X: " + body + "\n")
		if err != nil {
			t.Fatalf("compiling %q: %v", body, err)
		}
		var retrieve *ast.Retrieve
		switch expr := lib.Statements[0].Expression.(type) {
		case *ast.Retrieve:
			retrieve = expr
		case *ast.Query:
			retrieve, _ = expr.Sources[0].Source.(*ast.Retrieve)
		}
		if retrieve == nil {
			t.Fatalf("%q: no retrieve found", body)
		}
		if retrieve.DateRange != nil {
			t.Errorf("%q pushed down %v", body, retrieve.DateRange)
		}
	}
}

// Compiling is what applies the pushdown, so a source that fails to parse must
// not reach it half-built.
func TestPushdownDoesNotRunOnBrokenSource(t *testing.T) {
	if _, err := Compile("library T version '1.0'\ndefine X: [Encounter] E where\n"); err == nil {
		t.Error("a truncated where clause compiled, want a syntax error")
	} else if !strings.Contains(err.Error(), "syntax error") {
		t.Errorf("error = %v, want a syntax error", err)
	}
}
