package funcs

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	fptypes "github.com/gofhir/fhirpath/types"
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

// splitDecimal reads s by the grammar a string is read by as a Decimal:
//
//	[+-]?([0-9]+(\.[0-9]*)?|\.[0-9]+)([eE][+-]?[0-9]+)?
//
// digits with an optional point and an optional exponent, the Java engine's
// BigDecimal read with ASCII digits and no sign after the point. The
// specification's format, (+|-)?#0(.0#)? (CQL Reference, ToDecimal), has
// neither the exponent nor '1.', which the reference engine reads, nor '.5',
// which cql-execution reads and the Java engine answers null; this engine has
// read all three since #134, and keeps '.5' as it was (an own defect, backlog
// 1c23 of docs/plans/README.md). The grammar is the engine's and not the
// decimal library's: what that library accepts moved in v1.5.0 and v1.6.0 (a
// Unicode minus, a bound on the exponent), and with it this engine's answers
// did (docs/plans/009-decimal-1-6.md). It is read byte by byte, as a regular
// expression took over a second on a string of ten million digits under the
// race detector.
func splitDecimal(s string) (negative bool, whole, fraction, exponent string, ok bool) {
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		negative = s[i] == '-'
		i++
	}
	j := digitsFrom(s, i)
	whole = s[i:j]
	if j < len(s) && s[j] == '.' {
		k := digitsFrom(s, j+1)
		fraction = s[j+1 : k]
		j = k
	}
	if whole == "" && fraction == "" {
		return false, "", "", "", false
	}
	if j < len(s) && (s[j] == 'e' || s[j] == 'E') {
		k := j + 1
		if k < len(s) && (s[k] == '+' || s[k] == '-') {
			k++
		}
		end := digitsFrom(s, k)
		if end == k {
			return false, "", "", "", false
		}
		exponent, j = s[j+1:end], end
	}
	return negative, whole, fraction, exponent, j == len(s)
}

// exponentOf reads the written exponent, which has to be within an int32. Its
// leading zeros are dropped before it is read, and more than ten digits are
// past an int32 without reading them: strconv copies the text it refuses into
// its error, as long as the exponent was written.
func exponentOf(text string) (int64, bool) {
	if text == "" {
		return 0, true
	}
	sign := ""
	if text[0] == '+' || text[0] == '-' {
		sign, text = text[:1], text[1:]
	}
	text = strings.TrimLeft(text, "0")
	if len(text) > 10 {
		return 0, false
	}
	e, err := strconv.ParseInt(sign+"0"+text, 10, 32)
	return e, err == nil
}

// digitsFrom is the index past the ASCII digits of s from i on.
func digitsFrom(s string, i int) int {
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return i
}

// ErrNotADecimal reports a string that does not read as a Decimal. ToDecimal and
// convert answer it null, as the specification says: "If the input string is
// not formatted correctly, or cannot be interpreted as a valid Decimal value,
// the result is null" (CQL Reference, ToDecimal). A cast of it fails, as the
// argument is not of the type.
var ErrNotADecimal = errors.New("invalid decimal")

// DecimalFromString is a string read as a Decimal, at the Decimal scale and
// within its range, written as the reference engine writes it, or
// ErrNotADecimal.
//
// The type keeps eight places, and a string can carry more, as a literal
// cannot: ToDecimal('1.000000005') is 1.00000000 in the reference engine —
// eight places, floored as its arithmetic is — and -1.000000005 is
// -1.00000001. Kept whole, the extra digits reached the arithmetic as no
// Decimal written in CQL can, and a logarithm of a base a hair from 1 ran past
// the largest Decimal. A string past the largest Decimal is no Decimal at all,
// and null; so is one whose exponent is past an int32, as in the reference
// engine.
//
// The value is read from its digits, and only from those it keeps: the
// magnitude first, from the position of the first digit that is not zero,
// then at most the digits down to the eighth place, and whether any digit
// past them is not zero, which floors a negative one step down. Reading the
// whole mantissa as a number took a minute for ten million digits, and
// comparing or flooring a string like '1e999999999' rescaled it to a number
// of a billion digits. The decimal library's own reader refuses an exponent
// past its MaxDecodeExponent, which is global; the digits kept here are at
// most twenty-eight, built with decimal.NewFromBigInt, whose caller validates
// them.
func DecimalFromString(s string) (fptypes.Value, error) {
	negative, whole, fraction, exponentText, ok := splitDecimal(s)
	if !ok {
		return nil, fmt.Errorf("%w: %.40s", ErrNotADecimal, s)
	}
	exponent, ok := exponentOf(exponentText)
	if !ok {
		return nil, fmt.Errorf("%w: %.40s", ErrNotADecimal, s)
	}
	// digit is the i-th digit of the mantissa, the point taken away, and its
	// power of ten: the last digit is at the written exponent.
	written := exponent - int64(len(fraction)) // the power of the last digit
	count := len(whole) + len(fraction)
	digit := func(i int) byte {
		if i < len(whole) {
			return whole[i]
		}
		return fraction[i-len(whole)]
	}
	first := strings.IndexFunc(whole, isNonZeroDigit)
	if first < 0 {
		if f := strings.IndexFunc(fraction, isNonZeroDigit); f >= 0 {
			first = len(whole) + f
		}
	}
	if first < 0 {
		return fptypes.NewDecimal("0")
	}
	lead := written + int64(count-1-first) // the power of the first digit
	if lead >= maxDecimalDigitsBeforePoint {
		return nil, nil
	}
	// The digits down to the eighth place, or to the last one when it comes
	// first; the rest only tell whether something is left.
	places, power := max(0, -written), written // the places written, and the power of the last digit kept
	last := count - 1
	rest := false
	if written < -DecimalScale {
		places, power = DecimalScale, -DecimalScale
		last = count - 1 - int(-DecimalScale-written)
		// From the first digit at the least: when even that one is past the
		// scale, last+1 is before it, and may be negative.
		rest = nonZeroFrom(whole, fraction, max(first, last+1))
	}
	kept := make([]byte, 0, max(0, last-first+1)+1)
	for i := first; i <= last; i++ {
		kept = append(kept, digit(i))
	}
	coefficient := new(big.Int)
	if len(kept) > 0 {
		coefficient.SetString(string(kept), 10)
	}
	d := decimal.NewFromBigInt(coefficient, int32(power))
	if d.GreaterThan(maxDecimal) || (d.Equal(maxDecimal) && rest) {
		return nil, nil
	}
	if negative {
		d = d.Neg()
		if rest {
			d = d.Sub(decimal.New(1, -DecimalScale))
		}
	}
	return fptypes.NewDecimal(d.StringFixed(int32(places)))
}

// nonZeroFrom reports whether a digit from the i-th on of the mantissa, the
// point taken away, is not zero.
func nonZeroFrom(whole, fraction string, i int) bool {
	if i < len(whole) && strings.ContainsFunc(whole[i:], isNonZeroDigit) {
		return true
	}
	return strings.ContainsFunc(fraction[max(0, i-len(whole)):], isNonZeroDigit)
}

func isNonZeroDigit(r rune) bool { return r >= '1' && r <= '9' }
