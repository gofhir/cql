package cql

import (
	"context"
	"strings"
	"testing"
)

func evalAggregate(t *testing.T, expr string) (string, error) {
	t.Helper()
	src := "library T version '1.0'\ndefine X: " + expr + "\n"
	got, err := NewEngine().EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err != nil {
		return "", err
	}
	return valueString(got), nil
}

// TestAggregatesUnderstandQuantities covers aggregates that answered anyway.
// None of these returned null, which is what makes them worth a test: a dose of
// 0 mg, or a total short by six orders of magnitude, reads as a real
// measurement wherever it lands.
//
//	Avg({ 1 'mg', 2 'mg' })    was 0
//	Median({ 1 'mg', 2 'mg' }) was null
//	Sum({ 1 'mg', 1 'g' })     was 2 'mg'
//	Sum({ 1 'mg', 2 'kg' })    was 3 'mg'
func TestAggregatesUnderstandQuantities(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`Avg({ 1 'mg', 2 'mg' })`, "1.5 'mg'"},
		{`Avg({ 1 'mg' })`, "1 'mg'"},
		{`Sum({ 1 'mg', 2 'mg' })`, "3 'mg'"},

		// Odd counts take the middle element, even counts average the middle
		// pair — the same rule the numeric median follows.
		{`Median({ 1 'mg', 2 'mg', 3 'mg' })`, "2 'mg'"},
		{`Median({ 1 'mg', 2 'mg' })`, "1.5 'mg'"},

		// Order is by magnitude, not by the number in front of the unit.
		{`Median({ 3 'g', 1 'mg', 2 'g' })`, "2 'g'"},
		{`Median({ 1 'g', 2 'mg', 3 'mg' })`, "3 'mg'"},
	} {
		got, err := evalAggregate(t, tt.expr)
		if err != nil {
			t.Errorf("%s: %v", tt.expr, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestAggregatesConvertUnitsLikeTheOperatorDoes covers the rule these were not
// following. `1 'mg' + 1 'g'` has been 1001 'mg' all along, because the +
// operator asks Quantity.Add; the aggregates added the bare numbers, so a
// kilogram counted the same as a milligram.
func TestAggregatesConvertUnitsLikeTheOperatorDoes(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`Sum({ 1 'mg', 1 'g' })`, "1001 'mg'"},
		{`Sum({ 1 'mg', 2 'kg' })`, "2000001 'mg'"},
		{`Avg({ 1 'mg', 1 'g' })`, "500.5 'mg'"},
	} {
		got, err := evalAggregate(t, tt.expr)
		if err != nil {
			t.Errorf("%s: %v", tt.expr, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestAggregatesRefuseWhatTheyCannotAdd covers the cases where no number is the
// right answer. `1 'mg' + 1 's'` and `1 'mg' + 2` are already errors, and a
// collection is no better placed to guess.
func TestAggregatesRefuseWhatTheyCannotAdd(t *testing.T) {
	for _, tt := range []struct{ expr, wantErr string }{
		{`Sum({ 1 'mg', 1 's' })`, "incompatible units"},
		{`Avg({ 1 'mg', 1 's' })`, "incompatible units"},
		{`Median({ 1 'mg', 1 's' })`, "incompatible units"},

		// Skipping the bare number is how Sum({ 1 'mg', 2 }) came to be 1 'mg'.
		{`Sum({ 1 'mg', 2 })`, "non-quantity"},
		{`Avg({ 1 'mg', 2 })`, "non-quantity"},

		// Multiplying quantities compounds their units; this reduced them all
		// to zero instead, so any list of doses had a product of 0.
		{`Product({ 2 'mg', 3 'mg' })`, "not supported"},
	} {
		got, err := evalAggregate(t, tt.expr)
		if err == nil {
			t.Errorf("%s = %s, want an error mentioning %q", tt.expr, got, tt.wantErr)
			continue
		}
		if !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("%s: error = %v, want it to mention %q", tt.expr, err, tt.wantErr)
		}
	}
}

// A null beside a quantity is skipped rather than treated as a foreign type:
// aggregates ignore nulls, and that has to keep working now that a non-quantity
// is refused.
func TestNullsAreStillSkipped(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`Sum({ 1 'mg', null, 2 'mg' })`, "3 'mg'"},
		{`Avg({ 1 'mg', null, 3 'mg' })`, "2 'mg'"},
		{`Median({ 1 'mg', null, 3 'mg' })`, "2 'mg'"},
	} {
		got, err := evalAggregate(t, tt.expr)
		if err != nil {
			t.Errorf("%s: %v", tt.expr, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// The numeric paths are untouched. Quantities take a branch of their own, and
// nothing that worked before goes through it.
func TestNumericAggregatesAreUnchanged(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`Sum({ 1, 2, 3 })`, "6"},
		{`Sum({ 1.5, 2.5 })`, "4"},
		{`Avg({ 1.0, 2.0 })`, "1.5"},
		{`Avg({ 1, 2, 3, 4 })`, "2.5"},
		{`Median({ 1, 2, 3 })`, "2"},
		{`Median({ 1, 2, 3, 4 })`, "2.5"},
		{`Product({ 2, 3, 4 })`, "24"},
		{`Min({ 3, 1, 2 })`, "1"},
		{`Max({ 3, 1, 2 })`, "3"},
	} {
		got, err := evalAggregate(t, tt.expr)
		if err != nil {
			t.Errorf("%s: %v", tt.expr, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}
