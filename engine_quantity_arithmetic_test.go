package cql

import (
	"context"
	"strings"
	"testing"
)

func evalArith(t *testing.T, expr string) string {
	t.Helper()
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ndefine A: " + expr + "\n"
	got, err := NewEngine().EvaluateExpression(context.Background(), src, "A", nil, nil)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	if got == nil {
		return "null"
	}
	return got.String()
}

// TestAddingQuantitiesOfDifferentDimensionsIsNull covers the last operator family
// the specification names and this engine still failed on.
//
//	1 'cm' + 1 's'     error: incompatible units — and it took the define with it
//	1 'cm2' - 1 'cm'   the same
//	1 'cm' < 1 's'     null, since the comparison operators were corrected
//
// One rule, read two ways one operator apart. CQL says it in the paragraph that
// defines addition:
//
//	"When adding quantities, the dimensions of each quantity must be the same,
//	 but not necessarily the unit. For example, units of 'cm' and 'm' can be
//	 added, but units of 'cm2' and 'cm' cannot… Attempting to operate on
//	 quantities with invalid or special units will result in a null."
func TestAddingQuantitiesOfDifferentDimensionsIsNull(t *testing.T) {
	for _, expr := range []string{
		"1 'cm' + 1 's'",
		"1 'cm2' - 1 'cm'",
		"1 'mg' + 1 's'",
		"1 's' - 1 'mg'",
	} {
		if got := evalArith(t, expr); got != "null" {
			t.Errorf("%s = %s, want null", expr, got)
		}
	}
}

// TestABareNumberIsAQuantityOfUnitOne covers the half of this that was answering
// a number rather than failing, which is the worse half.
//
//	1 + 1 'cm'     1        the quantity dropped, in silence
//	1.5 + 1 'cm'   1.5      the same
//	1 - 1 'cm'     1        the same
//	1 'cm' + 1     error    the same pair, the other way round
//
// The cause is one guard on one side. The pair fell through to decimal
// arithmetic, where a Quantity reads as zero, so `1 + 0` answered 1 — and the
// check that catches that zero-reading existed for the left operand and not the
// right.
//
// What it should be is in the same paragraph: "when invoked with mixed Decimal and
// Quantity arguments, the Decimal argument will be implicitly converted to
// Quantity", and "when a quantity has no units specified, it is treated as a
// quantity with the default unit ('1')". A dimensionless quantity and a length
// have different dimensions, so the answer is the null above.
func TestABareNumberIsAQuantityOfUnitOne(t *testing.T) {
	for _, expr := range []string{
		"1 + 1 'cm'",
		"1 'cm' + 1",
		"1.5 + 1 'cm'",
		"1 'cm' + 1.5",
		"1 - 1 'cm'",
		"1 'cm' - 1",
		// Written out, which is the same pair and was already null.
		"1 '1' + 1 'cm'",
	} {
		if got := evalArith(t, expr); got != "null" {
			t.Errorf("%s = %s, want null", expr, got)
		}
	}

	// Two dimensionless quantities add, and a bare number is one of them.
	for _, tt := range []struct{ expr, want string }{
		{"1 '1' + 1 '1'", "2 '1'"},
		{"1 '1' + 2", "3 '1'"},
	} {
		if got := evalArith(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestMultiplicationCombinesDimensionsRatherThanRequiringThem is the limit on the
// rule above, and it is the specification's own limit rather than a shortcut.
//
// The sentence about matching dimensions is in the paragraph for `+`. Multiplying
// and dividing quantities is a different paragraph and a different rule: they
// build a unit out of the two rather than requiring one. So `1 'cm' * 1 's'` is
// 1 'cm.s', and a bare number scales rather than converting — which is why the
// promotion above is for `+` and `-` only. Promoting the 2 in `2 * 1 'cm'` would
// make it 2 '1.cm'.
func TestMultiplicationCombinesDimensionsRatherThanRequiringThem(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"1 'cm' * 1 'cm'", "1 'cm2'"},
		{"1 'cm' * 1 's'", "1 'cm.s'"},
		{"2 * 1 'cm'", "2 'cm'"},
		{"1 'cm' * 2", "2 'cm'"},
		{"Product({ 1 'cm2', 1 'cm' })", "1 'cm2.cm'"},
	} {
		if got := evalArith(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestSameDimensionArithmeticStillAnswers is the other half of the measurement:
// the rule must fire on dimensions that differ and nowhere else. Two units of one
// dimension convert and add, which is the ordinary case and the one a measure
// depends on.
func TestSameDimensionArithmeticStillAnswers(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"1 'cm' + 1 'm'", "101 'cm'"},
		{"100 'cm' - 1 'm'", "0 'cm'"},
		{"1 'cm' + 1 'cm'", "2 'cm'"},
		{"Sum({ 1 'cm', 1 'm' })", "101 'cm'"},
		{"Avg({ 1 'cm', 1 'cm' })", "1 'cm'"},
		// Temporal arithmetic is its own family and unmoved: a unit that is not a
		// duration still says so, rather than answering null.
		{"@2020-01-01 + 1 'd'", "2020-01-02"},
	} {
		if got := evalArith(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
	if got := evalArith(t, "@2020-01-01 + 1 'cm'"); got == "null" {
		t.Error("a date plus a length answered null; a unit that is not a duration should say so")
	}
}

// TestDivAndModStateTheRightOperandInTheLeftsUnit covers what the review of this
// change turned up: `div` and `mod` over two quantities ignored units entirely.
//
//	4 'cm' div 2 'm'   was 2 'cm'   — 4 cm div 200 cm is 0
//	5 'cm' mod 2 'm'   was 1 'cm'   — 5 cm mod 200 cm is 5
//	1 'cm' div 1 's'   was 1 'cm'   — no unit either can be stated in
//
// Three wrong answers, and only the third is about dimensions. The first two are
// wrong for units that convert perfectly well: both operators keep the left
// operand's unit, and neither restated the right one in it before dividing. So a
// measure comparing a reading in centimeters against a threshold in meters got a
// number a hundred times off, quietly.
//
// Converting first settles all three, and it is the same step Add and Subtract
// already take. Where there is no unit both can be stated in, the answer is the
// null this change gives every other arithmetic operator.
//
// CQL does not define `div` or `mod` over quantities at all — their signatures
// are Integer, Long and Decimal — so this is an extension of the language either
// way. It may as well be one that does not lie.
func TestDivAndModStateTheRightOperandInTheLeftsUnit(t *testing.T) {
	for _, tt := range []struct{ expr, want, why string }{
		{"4 'cm' div 2 'm'", "0 'cm'", "4 cm div 200 cm"},
		{"4 'cm' div 200 'cm'", "0 'cm'", "the same, written in one unit"},
		{"400 'cm' div 2 'm'", "2 'cm'", "400 cm div 200 cm"},
		{"100 'cm' div 30 'cm'", "3 'cm'", "three times, labeled cm — see below"},
		{"1 'm' div 3 'cm'", "33 'm'", "33 times, across a conversion"},

		// A remainder is in the dimension it was taken from, so mod keeps the
		// unit where div does not: the quotient is a count and the remainder is a
		// length.
		{"5 'cm' mod 2 'm'", "5 'cm'", "5 cm mod 200 cm"},
		{"5 'cm' mod 200 'cm'", "5 'cm'", "the same, written in one unit"},
		{"1 'm' mod 30 'cm'", "0.1 'm'", "0.1 m is left over"},

		// And no unit both can be stated in is the rule of this change.
		{"1 'cm' div 1 's'", "null", "no dimension in common"},
		{"1 'cm' mod 1 's'", "null", "the same"},
	} {
		if got := evalArith(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — %s", tt.expr, got, tt.want, tt.why)
		}
	}

	// The quotient keeps the left operand's unit, which reads oddly: a
	// whole-number quotient is a count, and `100 'cm' div 30 'cm'` says three
	// centimeters where it means three times. It is not this engine's to decide,
	// and the conformance corpus settles it —
	// `TruncatedDivide10By5DQuantity: 10.0 'g' div 5.0 'g' = 2.0 'g'`. A first
	// attempt at this change asked for a dimensionless '1' on the reasoning above,
	// and the corpus said no. What was wrong here was the magnitude, not the label.
	//
	// Division proper is untouched, and builds a unit out of the two rather than
	// restating one in the other, which is the paragraph `*` belongs to.
	for _, tt := range []struct{ expr, want string }{
		{"4 'cm' / 2 'm'", "2.0000000000000000 'cm/m'"},
		{"4 'cm' / 2 'cm'", "2.0000000000000000 '1'"},
	} {
		if got := evalArith(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestArithmeticRefusesWhatItCannotReadAsANumber covers the root of the defect
// above, which the review of this change found by looking at what else the same
// code path answers.
//
// The decimal fallback reads anything it does not recognize as zero, and the
// check that catches that read existed for the left operand only. So every
// operator answered a number for a right-hand operand that is not one:
//
//	1 + 'text'   1      'text' + 1   error
//	1 - 'text'   1      'text' - 1   error
//	1 * 'text'   0      the worst of them: it looks computed
//	2 ^ 'text'   1
//	2 ^ 1 'cm'   1      the same read, with a quantity
//
// That one-sided guard is exactly why `1 + 1 'cm'` answered 1 while
// `1 'cm' + 1` failed. Addition and subtraction no longer reach the fallback for
// a quantity, because a bare number beside one is now a quantity of unit '1';
// every other operator still does, and there is nothing to promote a String to.
//
// The semantic phase refuses all of these before evaluation, so the only path
// this changes is the explicit escape hatch — where an error about arithmetic
// that cannot be done is more use than a wrong number. See
// TestSemanticValidationIsOnByDefault.
func TestArithmeticRefusesWhatItCannotReadAsANumber(t *testing.T) {
	for _, expr := range []string{
		"2 ^ 1 'cm'",
		"1 'cm' ^ 2",
	} {
		if got := evalArith(t, expr); !strings.HasPrefix(got, "ERROR") {
			t.Errorf("%s = %s, want it reported rather than a number", expr, got)
		}
	}

	// Both orders now, which is the property: the guard is symmetric.
	for _, pair := range [][2]string{
		{"1 + 'text'", "'text' + 1"},
		{"1 - 'text'", "'text' - 1"},
		{"1 * 'text'", "'text' * 1"},
	} {
		a, b := evalArithUnchecked(t, pair[0]), evalArithUnchecked(t, pair[1])
		if !strings.HasPrefix(a, "ERROR") || !strings.HasPrefix(b, "ERROR") {
			t.Errorf("one side answered: %s = %s, %s = %s", pair[0], a, pair[1], b)
		}
	}

	// And two numbers are still two numbers.
	for _, tt := range []struct{ expr, want string }{
		{"1 + 2", "3"}, {"2 * 3", "6"}, {"2 ^ 3", "8"}, {"1.5 + 2", "3.5"},
	} {
		if got := evalArith(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// evalArithUnchecked evaluates with the semantic phase off, which is the only way
// to reach the arithmetic for an expression the phase would refuse.
func evalArithUnchecked(t *testing.T, expr string) string {
	t.Helper()
	src := "library T version '1.0'\ndefine A: " + expr + "\n"
	got, err := NewEngine(WithSemanticValidation(false)).
		EvaluateExpression(context.Background(), src, "A", nil, nil)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	if got == nil {
		return "null"
	}
	return got.String()
}
