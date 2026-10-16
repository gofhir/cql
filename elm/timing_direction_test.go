package elm

import (
	"encoding/json"
	"testing"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/compiler"
)

// TestSameOrBeforeSurvivesTheRoundTrip takes `same day or before` and
// `on or after` from CQL through ELM JSON and back. They are the SameAs phrase
// with a direction: written out as a plain SameAs the direction was lost, and
// an ELM SameOrBefore read back as a properly before was evaluated as a strict
// before, false on equal endpoints.
func TestSameOrBeforeSurvivesTheRoundTrip(t *testing.T) {
	for _, tt := range []struct {
		expr, elmType string
		before, after bool
	}{
		{"@2020-01-01 same day or before @2020-01-02", "SameOrBefore", true, false},
		{"@2020-01-01 on or after @2020-01-02", "SameOrAfter", false, true},
	} {
		lib, err := compiler.Compile("library T version '1.0'\ndefine R: " + tt.expr)
		if err != nil {
			t.Fatalf("%s: %v", tt.expr, err)
		}
		node := TranslateExpression(lib.Statements[0].Expression)
		if node.Type != tt.elmType {
			t.Errorf("%s is written as %s, want %s", tt.expr, node.Type, tt.elmType)
		}
		data, err := json.Marshal(node)
		if err != nil {
			t.Fatal(err)
		}
		var back ExpressionNode
		if err := json.Unmarshal(data, &back); err != nil {
			t.Fatal(err)
		}
		te, ok := ImportExpression(&back).(*ast.TimingExpression)
		if !ok {
			t.Fatalf("%s comes back as %T", tt.expr, ImportExpression(&back))
		}
		op := te.Operator
		if op.Kind != ast.TimingSameAs || op.Before != tt.before || op.After != tt.after || op.Properly {
			t.Errorf("%s comes back as %+v, want SameAs before=%t after=%t", tt.expr, op, tt.before, tt.after)
		}
	}
}

// TestARangeAtAPrecisionIsNotWrittenAsAnExactOne covers a date filter pushed
// down from a phrase at a precision. The evaluator widens it before a provider
// sees it, and ELM has nowhere to say so: written as a bare dateRange it came
// back as an exact one, and a provider dropped the rows at the edges the phrase
// keeps. Without a precision the range is written as before.
func TestARangeAtAPrecisionIsNotWrittenAsAnExactOne(t *testing.T) {
	for _, tt := range []struct {
		where     string
		wantRange bool
	}{
		{`E.period during day of "MP"`, false},
		{`E.period during "MP"`, true},
		// Not a precision the evaluator compares or widens at, so the range is
		// as exact as without one.
		{`E.period overlaps week of "MP"`, true},
	} {
		lib, err := compiler.Compile("library T version '1.0'\nusing FHIR version '4.0.1'\n" +
			"parameter \"MP\" Interval<DateTime>\ncontext Patient\n" +
			"define R: [Encounter] E where " + tt.where + "\n")
		if err != nil {
			t.Fatalf("%s: %v", tt.where, err)
		}
		node := TranslateExpression(lib.Statements[0].Expression)
		if len(node.SourceClause) != 1 || node.SourceClause[0].Expression == nil {
			t.Fatalf("%s: no query source written", tt.where)
		}
		retrieve := node.SourceClause[0].Expression
		if got := retrieve.DateRange != nil; got != tt.wantRange {
			t.Errorf("%s: dateRange written = %t, want %t", tt.where, got, tt.wantRange)
		}
	}
}
