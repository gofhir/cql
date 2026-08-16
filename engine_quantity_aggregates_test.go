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

// TestProductCompoundsUnits covers the aggregate that has to agree with the
// operator beside it. `2 'mg' * 3 'mg'` is 6 'mg2' in this engine, so a Product
// that refused to multiply quantities would be the same two-policy split these
// aggregates were fixed to remove — and a single-element list has no units to
// compound at all.
func TestProductCompoundsUnits(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`Product({ 4 'mg' })`, "4 'mg'"},
		{`Product({ 2 'mg', 3 'mg' })`, "6 'mg2'"},
		{`Product({ 2 'mg', 3 'mL' })`, "6 'mg.mL'"},
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
	// And it answers what the operator answers, rather than close to it.
	viaOperator, err := evalAggregate(t, `2 'mg' * 3 'mg'`)
	if err != nil {
		t.Fatalf("operator: %v", err)
	}
	viaAggregate, err := evalAggregate(t, `Product({ 2 'mg', 3 'mg' })`)
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}
	if viaOperator != viaAggregate {
		t.Errorf("operator gave %s, Product gave %s", viaOperator, viaAggregate)
	}
}

// TestSpreadAggregatesUnderstandQuantities covers the ones that measure spread.
// Variance, StdDev and PopulationVariance all answered 0 for a list of doses,
// which is not merely wrong but the specific claim that the doses were all the
// same.
//
// A variance is in the square of the unit and its root is back in the original,
// which is why these do not share a return type.
func TestSpreadAggregatesUnderstandQuantities(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{`Variance({ 1 'mg', 2 'mg', 3 'mg' })`, "1 'mg2'"},
		{`Variance({ 1 'mg', 2 'mg' })`, "0.5 'mg2'"},
		{`PopulationVariance({ 1 'mg', 3 'mg' })`, "1 'mg2'"},
		{`StdDev({ 1 'mg', 2 'mg', 3 'mg' })`, "1 'mg'"},

		// Converted first, so two ways of writing the same amount have no
		// spread between them.
		{`StdDev({ 1 'g', 1000 'mg' })`, "0 'g'"},

		// One value says nothing about spread, which is why the sample variance
		// of a single quantity is null — the same answer the numeric one gives.
		{`Variance({ 1 'mg' })`, "null"},
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
		{`Variance({ 1 'mg', 1 's' })`, "incompatible units"},

		// Skipping the bare number is how Sum({ 1 'mg', 2 }) came to be 1 'mg'.
		{`Sum({ 1 'mg', 2 })`, "non-quantity"},
		{`Avg({ 1 'mg', 2 })`, "non-quantity"},
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
		{`Variance({ 1, 2, 3 })`, "1"},
		{`StdDev({ 1, 2, 3 })`, "1"},
		{`PopulationVariance({ 1, 3 })`, "1"},
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
