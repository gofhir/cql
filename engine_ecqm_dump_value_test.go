package cql

import (
	"strings"
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

// TestDumpValueTellsAnswersApart holds the pairs that dumped alike before
// (code review of plan 018, rounds 1 and 2): if two answers differ, so do
// their lines in the dump.
func TestDumpValueTellsAnswersApart(t *testing.T) {
	a, b := "1", "2"
	unplaced, err := fptypes.NewDateTime("2020-03-01T10:00")
	if err != nil {
		t.Fatal(err)
	}
	pairs := [][2]fptypes.Value{
		{fptypes.NewString("false"), fptypes.NewBoolean(false)},
		{unplaced, cqltypes.WithDefaultOffset(unplaced, -3*time.Hour)},
		{fptypes.NewString(strings.Repeat("x", 200) + "a"), fptypes.NewString(strings.Repeat("x", 200) + "b")},
		{cqltypes.Code{Code: &a, Version: &a}, cqltypes.Code{Code: &a, Version: &b}},
		{cqltypes.Code{Code: &a}, cqltypes.Code{Code: &a, Version: new(string)}},
		{cqltypes.Concept{Codes: []cqltypes.Code{{Code: &a, Version: &a}}},
			cqltypes.Concept{Codes: []cqltypes.Code{{Code: &a, Version: &b}}}},
		{fptypes.MustDecimal("1.0"), fptypes.MustDecimal("1.00")},
		// Computed Decimals of different scale.
		{fptypes.NewDecimalFromInt(3), fptypes.MustDecimal("1.50").Multiply(fptypes.NewDecimalFromInt(2))},
		// The same JSON read as two FHIR types.
		{fptypes.NewObjectValueWithType([]byte(`{"value":1}`), "Quantity"),
			fptypes.NewObjectValueWithType([]byte(`{"value":1}`), "Age")},
	}
	for _, p := range pairs {
		if dumpValue(p[0]) == dumpValue(p[1]) {
			t.Errorf("two different answers dump alike: %s", dumpValue(p[0]))
		}
	}
}
