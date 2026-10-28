package eval

import (
	"strings"
	"testing"
)

// TestADecimalLiteralIsBoundedByItsDigitsOfValue holds the bounds of a Decimal
// literal (plan 011). cql-tests, ValueLiteralsAndSelectors.xml: DecimalTenthStep
// (0.000000001) is invalid, "the literal uses more precision than the Decimal
// step size", and Decimal10Pow28 (10000000000000000000000000000.00000000) is
// invalid, "a greater magnitude than a Decimal supports", while
// Decimal10Pow28ToZeroOneStepDecimalMaxValue reads a literal of 28 integer
// digits. A zero to the right of the places or to the left of the integer adds
// neither precision nor magnitude, and does not count (plan 011, decision D).
func TestADecimalLiteralIsBoundedByItsDigitsOfValue(t *testing.T) {
	for _, tt := range []struct {
		literal string
		ok      bool
	}{
		{"1.00000000", true},
		{"0.00000001", true},
		{"0.000000001", false}, // DecimalTenthStep
		{"1.000000005", false},
		{"5.999999999", false},
		{"0.0000000010", false}, // nine places of value, ten written
		{"1.000000000", true},   // decision D: nine written, none of value past the eighth
		{"1.0000000050", false},
		{"1.00000000000000000000000000000", true},
		{"0.000000000", true},
		{"0", true},
		{"0.0", true},
		{"1234567890123456789012345678.0", true},        // 28 integer digits
		{"9999999999999999999999999999.99999999", true}, // Decimal10Pow28ToZeroOneStepDecimalMaxValue
		{"1000000000000000000000000000.00000000", true},
		{"10000000000000000000000000000.00000000", false}, // Decimal10Pow28
		{"10000000000000000000000000000", false},
		{"000000000000000000000000000001.0", true}, // decision D: one integer digit of value
		{"0000000000000000000000000000000000.5", true},
		{"00000000000000000000000000000010000000000000000000000000000.0", false},
		{"123456789012345678901.0", true}, // past the type's 20 digits, within the suite's 28
		// The parser leaves the sign to a unary operator, but the check reads one.
		{"+1234567890123456789012345678.0", true},
		{"-1234567890123456789012345678.0", true},
		{"-0.000000001", false},                            // DecimalNegTenthStep
		{"+10000000000000000000000000000.00000000", false}, // DecimalPos10Pow28
	} {
		err := validateCQLDecimal(tt.literal)
		if (err == nil) != tt.ok {
			t.Errorf("validateCQLDecimal(%s) = %v; want ok %v", tt.literal, err, tt.ok)
		}
	}
}

// TestADecimalLiteralOfMillionsOfDigitsIsCheckedInBoundedResources: the
// check counts digits, and a literal of a million zeros is no reason to
// allocate.
func TestADecimalLiteralOfMillionsOfDigitsIsCheckedInBoundedResources(t *testing.T) {
	zeros := strings.Repeat("0", 1_000_000)
	for _, tt := range []struct {
		literal string
		ok      bool
	}{
		{zeros + "1." + zeros, true},
		{"1." + zeros + "1", false},
		{"1" + zeros + ".0", false},
	} {
		allocs := testing.AllocsPerRun(1, func() {
			if err := validateCQLDecimal(tt.literal); (err == nil) != tt.ok {
				t.Errorf("validateCQLDecimal of %d bytes: ok %v; want %v", len(tt.literal), err == nil, tt.ok)
			}
		})
		if !tt.ok {
			continue // the error message formats the literal
		}
		if allocs > 0 {
			t.Errorf("validateCQLDecimal of %d bytes allocated %v times", len(tt.literal), allocs)
		}
	}
}
