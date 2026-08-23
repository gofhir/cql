package cql

import (
	"strings"
	"testing"
)

// TestGeometricMeanUnderstandsQuantities is the last of the aggregates that did
// not, and the way it did not was hidden rather than honest.
//
// It was recorded as deliberately left out — "returns null, which does not lie."
// It does lie, by accident. numericVal reports zero for a Quantity, and the
// implementation guards non-positive values because the geometric mean of zero is
// not defined, so the dose was read as zero and the zero produced the null. Had
// the conversion answered 1 instead of 0 this would have returned a number.
//
// So the null was never a decision to decline. Every other aggregate on this list
// was fixed by asking the operators that already know about units; this one is
// the same fix, and the unit works out because the geometric mean of n values in
// mg is the nth root of a product in mg^n.
func TestGeometricMeanUnderstandsQuantities(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		// sqrt(2 × 8) = 4, back in the original unit.
		{`GeometricMean({ 2 'mg', 8 'mg' })`, "4 'mg'"},
		{`GeometricMean({ 1 'mg', 4 'mg' })`, "2 'mg'"},
		// The cube root of 1 × 2 × 4 is 2.
		{`GeometricMean({ 1 'mg', 2 'mg', 4 'mg' })`, "2 'mg'"},
		// One value is its own geometric mean, and answered null before.
		{`GeometricMean({ 4 'mg' })`, "4 'mg'"},

		// Converted first, the way Sum and StdDev convert: two spellings of one
		// amount have that amount as their geometric mean.
		{`GeometricMean({ 1 'g', 1000 'mg' })`, "1 'g'"},

		// Nulls are skipped, as everywhere else.
		{`GeometricMean({ 2 'mg', null, 8 'mg' })`, "4 'mg'"},
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

// TestGeometricMeanRefusesWhatItCannot covers the cases where no number is right,
// and where the numeric path already declines — a quantity must not answer where
// a plain number would not.
func TestGeometricMeanRefusesWhatItCannot(t *testing.T) {
	// Incompatible units, which is an error for every other aggregate too.
	if _, err := evalAggregate(t, `GeometricMean({ 1 'mg', 1 's' })`); err == nil {
		t.Error("GeometricMean over mg and s was answered, want an error")
	} else if !strings.Contains(err.Error(), "incompatible units") {
		t.Errorf("error = %v, want it to mention incompatible units", err)
	}

	// Mixing with a bare number: the collection cannot say what unit it carries.
	if _, err := evalAggregate(t, `GeometricMean({ 1 'mg', 2 })`); err == nil {
		t.Error("GeometricMean over a quantity and a number was answered, want an error")
	}

	// Non-positive values, where the geometric mean is not defined. The numeric
	// path answers null for these, so the quantity path has to as well.
	for _, expr := range []string{
		`GeometricMean({ 0 'mg', 4 'mg' })`,
		`GeometricMean({ -1 'mg', 4 'mg' })`,
	} {
		got, err := evalAggregate(t, expr)
		if err != nil {
			continue
		}
		if got != "null" {
			t.Errorf("%s = %s, want null — the geometric mean of a non-positive value "+
				"is not defined, and the numeric path says so too", expr, got)
		}
	}

	// And the numeric path is untouched.
	for _, tt := range []struct{ expr, want string }{
		{`GeometricMean({ 1, 4 })`, "2"},
		{`GeometricMean({ 2, 8 })`, "4"},
		{`GeometricMean({ 0, 4 })`, "null"},
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

// TestGeometricMeanDoesNotOverflowOrRoundToZero covers three defects that shared
// one cause: the product was computed in full and then rounded absolutely.
//
// Review found all three, and the first two are worse than wrong answers.
func TestGeometricMeanDoesNotOverflowOrRoundToZero(t *testing.T) {
	// A product large enough to exceed float64 panicked, because Float64()
	// returned +Inf and decimal.NewFromFloat cannot take one. Nothing in the
	// engine recovers, so it escaped EvaluateExpression entirely — reachable at
	// about 110 values of 1000 'mg'. The geometric mean of n copies of x is x,
	// however many copies there are.
	var many strings.Builder
	for i := range 400 {
		if i > 0 {
			many.WriteString(", ")
		}
		many.WriteString("100 'mg'")
	}
	got, err := evalAggregate(t, "GeometricMean({"+many.String()+"})")
	if err != nil {
		t.Errorf("400 quantities: %v", err)
	} else if got != "100 'mg'" {
		t.Errorf("GeometricMean of 400 × 100 'mg' = %s, want 100 'mg'", got)
	}

	// Round(8) is absolute, so a dose below 1e-8 was rounded to zero — a
	// fabricated zero dose, which is the exact failure this file's header
	// condemns. The numeric path does not round at all.
	got, err = evalAggregate(t, `GeometricMean({ 0.000000001 'mg', 0.000000001 'mg' })`)
	if err != nil {
		t.Errorf("tiny quantities: %v", err)
	} else if got == "0 'mg'" {
		t.Errorf("GeometricMean of two 1e-9 'mg' doses = %s, which is a dose that was "+
			"never measured", got)
	}

	// And the parity claimed with the numeric path has to hold, not be asserted:
	// the same figures with and without a unit must agree digit for digit.
	for _, pair := range [][2]string{
		{`GeometricMean({ 3 'mg', 5 'mg' })`, `GeometricMean({ 3, 5 })`},
		{`GeometricMean({ 2 'mg', 3 'mg', 7 'mg' })`, `GeometricMean({ 2, 3, 7 })`},
	} {
		withUnit, err1 := evalAggregate(t, pair[0])
		bare, err2 := evalAggregate(t, pair[1])
		if err1 != nil || err2 != nil {
			t.Errorf("%s / %s: %v %v", pair[0], pair[1], err1, err2)
			continue
		}
		if strings.TrimSuffix(withUnit, " 'mg'") != bare {
			t.Errorf("%s = %s but %s = %s — the same figures must agree",
				pair[0], withUnit, pair[1], bare)
		}
	}
}

// TestGeometricMeanChecksUnitsBeforeSigns covers the order-dependence review
// found: the non-positive guard ran per element, so it short-circuited before the
// units of a later element were ever looked at.
func TestGeometricMeanChecksUnitsBeforeSigns(t *testing.T) {
	_, errA := evalAggregate(t, `GeometricMean({ 0 'mg', 1 's' })`)
	_, errB := evalAggregate(t, `GeometricMean({ 1 's', 0 'mg' })`)
	if (errA == nil) != (errB == nil) {
		t.Errorf("reordering the list changed whether it is an error: %v vs %v", errA, errB)
	}
	if errA == nil {
		t.Error("mg and s was answered, want an error whichever comes first")
	}
}
