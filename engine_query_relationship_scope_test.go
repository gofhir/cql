package cql

import (
	"context"
	"strings"
	"testing"
)

// TestARelationshipAliasIsItsConditionsOnly covers the alias a with or without
// clause introduces. It belongs to the such-that condition. Evaluating the
// clause used to leave it bound to the last item tried, so the where, return
// and aggregate clauses after it could read it. Semantic validation refuses
// such a reference, so the case only shows with validation off, or through
// imported ELM, which skips it.
func TestARelationshipAliasIsItsConditionsOnly(t *testing.T) {
	e := NewEngine(WithSemanticValidation(false))
	eval := func(expr string) (string, error) {
		v, err := e.EvaluateExpression(context.Background(), "library T version '1.0'\ndefine R: "+expr, "R", nil, nil)
		if err != nil {
			return "", err
		}
		return valueString(v), nil
	}

	for _, expr := range []string{
		"({1, 2}) X with ({2}) Y such that X = Y return Y",
		"({1, 2}) X without ({1}) Y such that X = Y return Y",
		"({1, 2, 3}) X with ({2}) Y such that X = Y aggregate A starting 0: A + Y",
	} {
		if got, err := eval(expr); err == nil || !strings.Contains(err.Error(), "Y") {
			t.Errorf("%s = %s, %v; want Y unresolved outside its such-that", expr, got, err)
		}
	}

	// Put back as it was, not just removed: a with that reuses the name of a
	// source of its own query hands the source's value back when its condition
	// is done. Validation refuses the duplicate name; this is the case that
	// tells restoring from deleting.
	got, err := eval("({1}) Y with ({7}) Y such that true return Y")
	if err != nil || got != "{1}" {
		t.Errorf("the source Y after a with of the same name = %s, %v; want {1}", got, err)
	}
}

// TestIndexUnderAggregateDistinctIsTheRowsPlace pins $index inside an aggregate.
// It is this engine's: the reference translator refuses $index in a query. It
// is the row's place among the rows the query considered, the index its lets
// and filters saw, not its place among the rows distinct kept.
func TestIndexUnderAggregateDistinctIsTheRowsPlace(t *testing.T) {
	v, err := NewEngine().EvaluateExpression(context.Background(),
		"library T version '1.0'\ndefine R: ({5, 5, 6}) X aggregate distinct A starting 0: A + $index", "R", nil, nil)
	if err != nil || valueString(v) != "2" {
		t.Errorf("$index summed over the distinct rows of {5, 5, 6} = %v, %v; want 2 (places 0 and 2)", v, err)
	}
}
