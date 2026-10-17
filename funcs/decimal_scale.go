package funcs

import (
	"math"

	"github.com/shopspring/decimal"
)

// DecimalScale is the number of digits a CQL Decimal keeps to the right of the
// point: the specification gives the type "a precision of 28 and a scale ... of
// 8".
const DecimalScale = 8

// ScaleDecimal brings the result of Decimal arithmetic to that scale, toward
// negative infinity, as the reference engine does: 10.0 / 3.0 is 3.33333333,
// and -2.0 / 3.0 is -0.66666667. The specification names the scale and not the
// rounding; the Java engine's FLOOR is the rule taken, so that `10.0 / 3.0 =
// 3.33333333` is true here as it is there.
//
// It applies where the reference engine applies it — the quotient, the product,
// the sum, the remainder, the average, Log and powers, and the variance built
// from them; Log answering an exact power exactly (see Log), and a power with a
// whole-number exponent computed exactly (see ExactPower) — and
// not to subtraction or Exp, which it leaves unscaled: Exp(1.0)
// and Exp(1.0) - 0.0 are both past 2.71828182 there, and Exp(1.0) * 1.0 is not. Two places it is not
// applied though the reference engine applies it: Ln, where the conformance
// suite's Round(Ln(1000), 8) = 6.90775528 needs the digit flooring would drop,
// and GeometricMean, where flooring the exponent 1/3 makes the geometric mean
// of {1, 2, 4} 1.99999998 rather than 2. A value already within the scale is
// returned as it is, keeping its own trailing digits. A Quantity is not brought
// to it: the scale is the Decimal type's, and a dose in a small unit keeps its
// digits.
func ScaleDecimal(d decimal.Decimal) decimal.Decimal {
	if d.Exponent() >= -DecimalScale {
		return d
	}
	return d.RoundFloor(DecimalScale)
}

// DivideScaled is a quotient at the Decimal scale, floored exactly. decimal's
// Div rounds to 16 places first, half up, so a quotient just below a multiple
// of 1e-8 rounded up to it and the floor kept it: 0.00000002 / 2.00000001 is
// 0.0000000099999…, which is 0.00000000 in the reference engine and was
// 0.00000001 here. Rounding further out first only narrows the window, so the
// quotient is truncated at the scale and moved down a step where truncating
// went toward zero from below it.
func DivideScaled(a, b decimal.Decimal) decimal.Decimal {
	quotient, remainder := a.QuoRem(b, DecimalScale)
	if remainder.IsZero() {
		return quotient
	}
	if remainder.Sign()*b.Sign() < 0 {
		quotient = quotient.Sub(decimal.New(1, -DecimalScale))
	}
	return quotient
}

// FloatScaled is a float64 result — a power, a logarithm, a root — at the
// Decimal scale. The float is read as the shortest decimal that prints as it,
// and that is floored. The reference engine floors the float's exact binary
// value instead, which makes 0.09 ^ 0.5 0.29999999 there, the float written 0.3
// being 0.29999999999999998…; but the conformance suite asks Power(10, -7) for
// 0.0000001, and the float 1e-7 is 0.0000000999…, which that reading floors to
// 0.00000009.
//
// ok is false for a float that is not a number — an overflow to infinity, or
// NaN — which has no Decimal to give, and which decimal.NewFromFloat panics on.
func FloatScaled(f float64) (d decimal.Decimal, ok bool) {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return decimal.Zero, false
	}
	return ScaleDecimal(decimal.NewFromFloat(f)), true
}
