package funcs

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	fptypes "github.com/gofhir/fhirpath/types"
)

// TestDivideScaledFloorsExactly covers quotients near a multiple of 1e-8. A
// rounding before the floor, at any fixed number of places, can lift one just
// below the multiple onto it; the quotient is floored from its exact remainder
// instead, toward negative infinity as the reference engine floors it.
func TestDivideScaledFloorsExactly(t *testing.T) {
	d := decimal.RequireFromString
	for _, tt := range []struct{ a, b, want string }{
		{"10.0", "3.0", "3.33333333"},
		{"-2.0", "3.0", "-0.66666667"},
		{"2.0", "-3.0", "-0.66666667"},
		{"-2.0", "-3.0", "0.66666666"},
		{"0.00000002", "2.00000001", "0"},
		// Within 1e-43 of 1e-8: rounding at 40 places first made it 0.00000001.
		{"10000000000000000000.0", "1000000000000000000000000000.00000001", "0"},
		{"1.0", "8.0", "0.125"},
		{"6", "3", "2"},
	} {
		if got := DivideScaled(d(tt.a), d(tt.b)); !got.Equal(d(tt.want)) {
			t.Errorf("%s / %s = %s, want %s", tt.a, tt.b, got, tt.want)
		}
	}
}

// TestLogAnswersAnExactPowerExactly covers logarithms near a whole number. An
// exact power has a whole-number logarithm, which the float around it misses:
// ln 1000 / ln 10 is 2.9999999999999996. Only an exact power is answered with
// the whole number — checked in decimal — and a value merely near one floors
// as the float says, as the reference engine floors it.
func TestLogAnswersAnExactPowerExactly(t *testing.T) {
	d := decimal.RequireFromString
	for _, tt := range []struct{ v, base, want string }{
		{"1000", "10", "3"},
		{"1000000", "10", "6"},
		{"0.001", "10", "-3"},
		{"0.125", "2", "-3"},
		{"16", "2", "4"},
		// Near a power and not one: the floor the float gives.
		{"999.999999999999", "10", "2.99999999"},
		{"0.00099999999999999", "10", "-3.00000001"},
		{"10.0", "3.0", "2.09590327"},
		// An exact fraction, as an exact whole number is.
		{"1000", "100", "1.5"},
		{"1000", "10000", "0.75"},
		{"8", "4", "1.5"},
		{"2", "8", "0.33333333"},
		// Trailing zeros of the base do not make it longer.
		{"1000", "10.0000", "3"},
		// A tiny value read from its own digits, not a subnormal float.
		{"1e-320", "10", "-320"},
		// A base a float rounds to 1.0 still has a logarithm.
		{"10", "1.0000000000000000001", "23025850929940456840.17998"},
	} {
		got, err := Log(decimalValue(d(tt.v)), decimalValue(d(tt.base)))
		if err != nil || got == nil {
			t.Errorf("Log(%s, %s) = %v, %v; want %s", tt.v, tt.base, got, err, tt.want)
			continue
		}
		if tt.base == "1.0000000000000000001" {
			// Only the magnitude: the float carries 16 digits of it.
			if !strings.HasPrefix(got.String(), "230258509299404") {
				t.Errorf("Log(%s, %s) = %s, want about %s", tt.v, tt.base, got, tt.want)
			}
			continue
		}
		if got.String() != tt.want {
			t.Errorf("Log(%s, %s) = %s, want %s", tt.v, tt.base, got, tt.want)
		}
	}
}

func decimalValue(d decimal.Decimal) fptypes.Value {
	return decimalToValue(d)
}

// TestPowersEqualRefusesAPowerNoDecimalHolds covers a base of a thousand
// digits raised near the bound: the power would run to millions of digits and
// took a third of a second to compute and compare, for every row that reached
// it. One that long is refused from its digit count.
func TestPowersEqualRefusesAPowerNoDecimalHolds(t *testing.T) {
	base := decimal.RequireFromString("2." + strings.Repeat("0", 999) + "1")
	v := decimal.RequireFromString("1" + strings.Repeat("0", 900))
	start := time.Now()
	if powersEqual(v, 1, base, 3300) || powersEqual(v, 1, base, -3300) {
		t.Error("a power past any decimal was reported exact")
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Errorf("refusing it took %s", elapsed)
	}
}

// TestLogOfTwoValuesAFloatCallsOneIsNull covers a logarithm whose float ratio
// is 0/0: both values are past 1 by less than a float holds, so both natural
// logarithms are 0. Reading a whole number off the NaN converted it to the
// smallest int64 on amd64, and computing that power never returned.
func TestLogOfTwoValuesAFloatCallsOneIsNull(t *testing.T) {
	near := decimal.RequireFromString("1." + strings.Repeat("0", 330) + "1")
	got, err := Log(decimalValue(near), decimalValue(near))
	if err != nil || got != nil {
		t.Errorf("Log of two values a float rounds to 1 = %v, %v; want null", got, err)
	}
}
