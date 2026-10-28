package cql

import (
	"context"
	"strings"
	"testing"
)

// TestADecimalLiteralPastTheSuiteFailsWhereItIsEvaluated holds where the
// error of a Decimal literal past what cql-tests reads lands (plan 011,
// decision E). It is an error of evaluation, not of translation: a library
// that holds one fails as a whole in EvaluateLibrary, while a definition
// beside it evaluates alone, and a branch that is not taken does not fail.
// Moving the check to translation is in the backlog.
func TestADecimalLiteralPastTheSuiteFailsWhereItIsEvaluated(t *testing.T) {
	lib := "library L version '1.0'\n" +
		"define Bad: 0.000000001\n" +
		"define Good: 1\n" +
		"define Untaken: if false then 0.000000001 else 1.0\n"
	e := NewEngine()
	if _, err := e.Parse(lib); err != nil {
		t.Fatalf("Parse: %v; want the library to translate", err)
	}
	results, err := e.EvaluateLibrary(context.Background(), lib, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "too many fractional digits") || len(results) != 0 {
		t.Errorf("EvaluateLibrary = %v, %v; want the literal's error and no results", results, err)
	}
	for name, want := range map[string]string{"Good": "1", "Untaken": "1.0"} {
		got, err := e.EvaluateExpression(context.Background(), lib, name, nil, nil)
		if err != nil || got == nil || got.String() != want {
			t.Errorf("%s = %v, %v; want %s", name, got, err, want)
		}
	}
}
