package funcs

import (
	"math/big"
	"strings"
	"sync"

	"github.com/shopspring/decimal"

	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
	ucum "github.com/gofhir/ucum/v4"
)

// maxDecimalPrecision is the number of digits a CQL Decimal holds: the
// specification gives the type "a precision of 28 and a scale ... of 8".
const maxDecimalPrecision = 28

// unitSeparator is what goes between the number and the unit: spaces and tabs.
// A line break there is null, as in the Java engine (plan 010, decision B).
const unitSeparator = " \t"

// trimmed is s without what the Java engine trims from around it, as
// String.trim does: every character up to U+0020, the line breaks among them
// (plan 010, decision B). A cutset of ASCII, so the standard library trims
// it without a call of this package per character, which coverage counts:
// ten million spaces took 3.4 s under -race -covermode=atomic in CI.
func trimmed(s string) string {
	return strings.Trim(s, javaTrimmed)
}

// javaTrimmed is every character up to U+0020.
const javaTrimmed = "\x00\x01\x02\x03\x04\x05\x06\x07\x08\t\n\x0b\x0c\r\x0e\x0f" +
	"\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f "

// ucumService validates a unit against UCUM. The definitions load once, on the first
// string ToQuantity reads with a unit (2.5 ms and 214 KiB, plan 010); the
// service is safe for concurrent use.
var ucumService = sync.OnceValues(ucum.New)

// QuantityFromString is a string read as a Quantity, or null, by the CQL
// Reference (Appendix B, ToQuantity): "(+|-)?#0(.0#)?('<unit>')?", the unit "a
// string literal specifying a valid, case-sensitive UCUM unit of measure or
// calendar duration keyword", "Spaces are allowed between the quantity value
// and the unit designator", and "If the input string is not formatted
// correctly, or cannot be interpreted as a valid Quantity value, the result is
// null". Without a unit the unit is '1', the default unit.
//
// The number is read as ToDecimal reads one (see splitDecimal), and within the
// Decimal's range, but it keeps the digits written: the scale is the Decimal
// type's, and a Quantity is not brought to it (ADR 002). The digits written
// are at most the Decimal's precision of 28, counted in fixed notation, which
// also bounds what a long string costs; past it the value is null, where the
// Java engine keeps them all (docs/plans/010-to-quantity.md names it).
//
// The error is only that UCUM's definitions did not load.
func QuantityFromString(s string) (fptypes.Value, error) {
	s = trimmed(s)
	parts, end, ok := scanDecimal(s)
	if !ok {
		return nil, nil
	}
	value, ok := decimalAsWritten(parts)
	if !ok {
		return nil, nil
	}
	rest := strings.TrimLeft(s[end:], unitSeparator)
	if rest == "" {
		return fptypes.NewQuantityFromDecimal(value, "1"), nil
	}
	// A string literal: a quote, the unit, and the closing quote last.
	if len(rest) < 2 || rest[0] != '\'' || strings.IndexByte(rest[1:], '\'') != len(rest)-2 {
		return nil, nil
	}
	unit := rest[1 : len(rest)-1]
	// A unit longer than UCUM reads is none, and is not copied; the one kept is
	// a copy, as a slice of s would keep all of s alive in the Quantity, ten
	// million bytes for 'mg' (review 2 of plan 010).
	if len(unit) > ucum.MaxCodeLength {
		return nil, nil
	}
	unit = strings.Clone(unit)
	if !cqltypes.IsCalendarKeyword(unit) {
		service, err := ucumService()
		if err != nil {
			return nil, err
		}
		if service.Validate(unit) != nil {
			return nil, nil
		}
	}
	return fptypes.NewQuantityFromDecimal(value, unit), nil
}

// decimalAsWritten is the number scanDecimal read, with the
// places written, or false when it is past the Decimal's range or precision.
// Both are decided from the positions of the digits before any is read, so a
// string of ten million digits is not made a number.
func decimalAsWritten(p decimalParts) (decimal.Decimal, bool) {
	whole, fraction := p.whole, p.fraction
	exponent, ok := exponentOf(p.exponent)
	if !ok {
		return decimal.Decimal{}, false
	}
	written := exponent - int64(len(fraction)) // the power of the last digit
	places := max(0, -written)
	count := len(whole) + len(fraction)
	first := firstNonZero(whole)
	if first < 0 {
		if f := firstNonZero(fraction); f >= 0 {
			first = len(whole) + f
		}
	}
	if first < 0 {
		if places > maxDecimalPrecision {
			return decimal.Decimal{}, false
		}
		return decimal.New(0, int32(-places)), true
	}
	lead := written + int64(count-1-first) // the power of the first digit
	if lead >= maxDecimalDigitsBeforePoint || max(lead+1, 0)+places > maxDecimalPrecision {
		return decimal.Decimal{}, false
	}
	// At most 28 digits are left from the first that is not zero on.
	var kept string
	if first < len(whole) {
		kept = whole[first:] + fraction
	} else {
		kept = fraction[first-len(whole):]
	}
	coefficient := new(big.Int)
	coefficient.SetString(kept, 10)
	if p.negative {
		coefficient.Neg(coefficient)
	}
	return decimal.NewFromBigInt(coefficient, int32(written)), true //nolint:gosec // within [-28, 19] by the bounds above
}
