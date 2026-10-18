package funcs

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	fptypes "github.com/gofhir/fhirpath/types"
)

// toDecimalVal converts an fptypes.Value to a decimal.Decimal.
// Returns false if the value is not numeric.
func toDecimalVal(v fptypes.Value) (decimal.Decimal, bool) {
	return NumberOf(v)
}

// Round rounds a decimal value to the given precision (number of decimal places).
// A half rounds away from zero, so the sign does not change which way it goes:
// -1.5 → -2, -0.5 → -1, 0.5 → 1, 1.5 → 2. If the value is nil, returns nil.
func Round(v fptypes.Value, precision int) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	d, ok := toDecimalVal(v)
	if !ok {
		return nil, fmt.Errorf("Round: expected numeric, got %s", v.Type())
	}
	if precision < math.MinInt32 || precision > math.MaxInt32 {
		return nil, fmt.Errorf("Round: precision out of range: %d", precision)
	}
	return decimalToValue(d.Round(int32(precision))), nil
}

// Floor returns the largest integer less than or equal to the value.
func Floor(v fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	if i, ok := v.(fptypes.Integer); ok {
		return integerInRange(i), nil
	}
	d, ok := toDecimalVal(v)
	if !ok {
		return nil, fmt.Errorf("Floor: expected numeric, got %s", v.Type())
	}
	return integerOrNull(d.Floor()), nil
}

// Ceiling returns the smallest integer greater than or equal to the value.
func Ceiling(v fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	if i, ok := v.(fptypes.Integer); ok {
		return integerInRange(i), nil
	}
	d, ok := toDecimalVal(v)
	if !ok {
		return nil, fmt.Errorf("Ceiling: expected numeric, got %s", v.Type())
	}
	return integerOrNull(d.Ceil()), nil
}

// integerOrNull returns d as an Integer, or null when it falls outside the range
// CQL gives that type. Floor, Ceiling and Truncate are the operators that can carry a
// Decimal past the Integer bounds while still being asked for an Integer, and CQL
// answers that with null rather than a wrapped or clamped number.
func integerOrNull(d decimal.Decimal) fptypes.Value {
	if d.LessThan(minInteger) || d.GreaterThan(maxInteger) {
		return nil
	}
	return fptypes.NewInteger(d.IntPart())
}

// integerInRange is integerOrNull for an Integer or a Long given to Floor,
// Ceiling or Truncate: an Integer is itself, and a Long past the Integer
// bounds — Floor(3000000000L) — is null, the same rule as for a Decimal.
func integerInRange(i fptypes.Integer) fptypes.Value {
	if v := i.Value(); v < math.MinInt32 || v > math.MaxInt32 {
		return nil
	}
	return i
}

var (
	minInteger = decimal.NewFromInt(math.MinInt32)
	maxInteger = decimal.NewFromInt(math.MaxInt32)
)

// Truncate removes the fractional part of a decimal value, returning an integer.
func Truncate(v fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	if i, ok := v.(fptypes.Integer); ok {
		return integerInRange(i), nil
	}
	d, ok := toDecimalVal(v)
	if !ok {
		return nil, fmt.Errorf("Truncate: expected numeric, got %s", v.Type())
	}
	// As Floor and Ceiling: past the Integer bounds is null. IntPart wrapped,
	// and Truncate(100000000000000000000.0) was 7766279631452241920.
	return integerOrNull(d.Truncate(0)), nil
}

// Ln returns the natural logarithm of the value.
func Ln(v fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	d, ok := toDecimalVal(v)
	if !ok {
		return nil, fmt.Errorf("Ln: expected numeric, got %s", v.Type())
	}
	if d.IsZero() {
		return nil, fmt.Errorf("Ln: undefined for zero (results in negative infinity)")
	}
	if d.Sign() < 0 {
		return nil, nil // CQL: Ln of negative value returns null
	}
	result := lnOf(d)
	// Not brought to the Decimal scale, although the reference engine floors it
	// there: the conformance suite asks Round(Ln(1000), 8) for 6.90775528, and
	// ln 1000 is 6.9077552789…, which floors to 6.90775527 first.
	return decimalToValue(decimal.NewFromFloat(result)), nil
}

// Log returns the logarithm of value with the given base.
func Log(v, base fptypes.Value) (fptypes.Value, error) {
	if v == nil || base == nil {
		return nil, nil
	}
	d, ok := toDecimalVal(v)
	if !ok {
		return nil, fmt.Errorf("Log: expected numeric value, got %s", v.Type())
	}
	b, ok := toDecimalVal(base)
	if !ok {
		return nil, fmt.Errorf("Log: expected numeric base, got %s", base.Type())
	}
	if d.Sign() <= 0 || b.Sign() <= 0 || b.Equal(decimal.NewFromInt(1)) {
		return nil, nil // CQL: undefined
	}
	result := lnOf(d) / lnOf(b)
	// No logarithm the float can give: two values a float rounds to 1 have the
	// ratio 0/0, and reading a whole number off NaN converts unpredictably.
	if math.IsNaN(result) || math.IsInf(result, 0) {
		return nil, nil
	}
	if exact, isExact := exactLog(d, b, result); isExact {
		return decimalToValue(exact), nil
	}
	scaled, ok := FloatScaled(result)
	if !ok {
		return nil, nil
	}
	return decimalToValue(scaled), nil
}

// Exp returns e raised to the power of the value.
func Exp(v fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	d, ok := toDecimalVal(v)
	if !ok {
		return nil, fmt.Errorf("Exp: expected numeric, got %s", v.Type())
	}
	f, _ := d.Float64()
	result := math.Exp(f)
	if math.IsInf(result, 0) || math.IsNaN(result) {
		return nil, fmt.Errorf("Exp: overflow for value %v", d)
	}
	return decimalToValue(decimal.NewFromFloat(result)), nil
}

// Power returns value raised to the exponent power.
func Power(v, exp fptypes.Value) (fptypes.Value, error) {
	if v == nil || exp == nil {
		return nil, nil
	}
	d, ok := toDecimalVal(v)
	if !ok {
		return nil, fmt.Errorf("Power: expected numeric value, got %s", v.Type())
	}
	e, ok := toDecimalVal(exp)
	if !ok {
		return nil, fmt.Errorf("Power: expected numeric exponent, got %s", exp.Type())
	}
	// An Integer to a non-negative Integer power is an Integer, and one past
	// int64 is an overflow: 2 ^ 63 is null, not a value the conversion wrapped.
	_, vIsInt := v.(fptypes.Integer)
	_, eIsInt := exp.(fptypes.Integer)
	integral := vIsInt && eIsInt && e.Sign() >= 0
	exact, isExact, overflow := ExactPower(d, e)
	switch {
	case overflow:
		return nil, nil
	case isExact && integral:
		return IntegerOf(exact), nil
	case isExact:
		return decimalToValue(exact), nil
	}
	// An Integer to a whole power the exact path declined is past the largest
	// Decimal, and so past int64.
	if integral {
		return nil, nil
	}
	fv, _ := d.Float64()
	fe, _ := e.Float64()
	scaled, ok := FloatScaled(math.Pow(fv, fe))
	// Past the largest Decimal is an overflow, finite float or not. Compared as
	// a decimal: the float of the largest Decimal rounds up to 1e20 itself.
	if !ok || scaled.Abs().GreaterThan(maxDecimal) {
		return nil, nil
	}
	return decimalToValue(scaled), nil
}

// IntegerOf is a whole number as an Integer, or null where it does not fit one:
// IntPart wraps a value past int64 in silence, and `1e320 div 3.0` answered
// 6148914691236517205.
func IntegerOf(d decimal.Decimal) fptypes.Value {
	whole := d.BigInt()
	if !whole.IsInt64() {
		return nil
	}
	return fptypes.NewInteger(whole.Int64())
}

// Precision returns the number of digits of precision in the value's representation.
// For Decimal: number of digits after decimal point (trailing zeros preserved).
// For DateTime/Date/Time: number of digits in the string representation.
func Precision(v fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	switch val := v.(type) {
	case fptypes.Integer:
		// For integers, precision is the number of digits
		s := fmt.Sprintf("%d", val.Value())
		s = strings.TrimLeft(s, "-")
		return fptypes.NewInteger(int64(len(s))), nil
	case fptypes.Decimal:
		// CQL Precision for decimal: count digits after decimal point including trailing zeros
		s := val.String()
		s = strings.TrimLeft(s, "-")
		parts := strings.Split(s, ".")
		if len(parts) == 2 {
			return fptypes.NewInteger(int64(len(parts[1]))), nil
		}
		return fptypes.NewInteger(0), nil
	case fptypes.DateTime:
		// Precision = number of digits in the datetime string representation
		s := val.String()
		count := 0
		for _, c := range s {
			if c >= '0' && c <= '9' {
				count++
			}
		}
		return fptypes.NewInteger(int64(count)), nil
	case fptypes.Date:
		s := val.String()
		count := 0
		for _, c := range s {
			if c >= '0' && c <= '9' {
				count++
			}
		}
		return fptypes.NewInteger(int64(count)), nil
	case fptypes.Time:
		s := val.String()
		count := 0
		for _, c := range s {
			if c >= '0' && c <= '9' {
				count++
			}
		}
		return fptypes.NewInteger(int64(count)), nil
	default:
		return nil, nil
	}
}

// HighBoundary returns the high boundary of a value at the given precision.
func HighBoundary(v, precision fptypes.Value) (fptypes.Value, error) {
	// A boundary describes the value it was given, so it sits where that value
	// sits: an offset supplied as v's default applies to the result, and rebuilding
	// through NewDateTime inside would otherwise lose it. An offset v states for
	// itself is kept by splitTemporalBoundary's own handling, which is a different
	// mechanism for a different thing.
	//
	// What this cannot do is give a boundary a frame its source never had.
	// `HighBoundary(@2020-06-15T, 17)` refines a value with no hour into one with a
	// millisecond, and there is no offset on the source to carry — funcs cannot ask
	// for the evaluation request's. The evaluator finishes that case in
	// situateRefinement, which is where the request's offset is reachable.
	out, err := highboundaryOf(v, precision)
	if err != nil {
		return nil, err
	}
	return carryDefaultOffset(v, out), nil
}

func highboundaryOf(v, precision fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	prec := int64(8)
	if precision != nil {
		if pi, ok := precision.(fptypes.Integer); ok {
			prec = pi.Value()
		}
	}
	switch val := v.(type) {
	case fptypes.Decimal:
		// High boundary: keep existing digits, fill remaining precision positions with 9
		d := val.Value()
		s := d.String()
		// Find current number of decimal digits
		dotIdx := strings.IndexByte(s, '.')
		currentDecimals := 0
		if dotIdx >= 0 {
			currentDecimals = len(s) - dotIdx - 1
		}
		if int64(currentDecimals) >= prec {
			// Already at or beyond target precision, just format
			return fptypes.NewDecimal(d.StringFixed(int32(prec)))
		}
		// Fill remaining digits with 9
		result := d.StringFixed(int32(prec))
		// Replace trailing zeros (after current precision) with 9s
		runes := []byte(result)
		resultDotIdx := strings.IndexByte(result, '.')
		if resultDotIdx >= 0 {
			for i := resultDotIdx + 1 + currentDecimals; i < len(runes); i++ {
				runes[i] = '9'
			}
		}
		return fptypes.NewDecimal(string(runes))
	case fptypes.Integer:
		return val, nil
	case fptypes.DateTime:
		return temporalHighBoundary(val.String(), int(prec), "datetime")
	case fptypes.Date:
		return temporalHighBoundary(val.String(), int(prec), "date")
	case fptypes.Time:
		return temporalHighBoundary(val.String(), int(prec), "time")
	default:
		return nil, nil
	}
}

// LowBoundary returns the low boundary of a value at the given precision.
func LowBoundary(v, precision fptypes.Value) (fptypes.Value, error) {
	// A boundary describes the value it was given, so it sits where that value
	// sits: an offset the caller supplied for v applies to the result. Rebuilding
	// through NewDateTime inside would otherwise lose it.
	out, err := lowboundaryOf(v, precision)
	if err != nil {
		return nil, err
	}
	return carryDefaultOffset(v, out), nil
}

func lowboundaryOf(v, precision fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	prec := int64(8)
	if precision != nil {
		if pi, ok := precision.(fptypes.Integer); ok {
			prec = pi.Value()
		}
	}
	switch val := v.(type) {
	case fptypes.Decimal:
		d := val.Value()
		s := d.StringFixed(int32(prec))
		return fptypes.NewDecimal(s)
	case fptypes.Integer:
		return val, nil
	case fptypes.DateTime:
		return temporalLowBoundary(val.String(), int(prec), "datetime")
	case fptypes.Date:
		return temporalLowBoundary(val.String(), int(prec), "date")
	case fptypes.Time:
		return temporalLowBoundary(val.String(), int(prec), "time")
	default:
		return nil, nil
	}
}

// splitWrittenOffset separates a written timezone offset from the digits that say
// how finely a value is specified.
//
// The boundary fill works by counting digits and appending more, and an offset
// left on the end was counted as precision and filled after:
// `HighBoundary(@2020-06-15T23:00:00Z, 17)` built "2020-06-15T23:00:00Z999" and
// parsing that aborted the whole expression rather than returning a boundary. An
// offset says where a value sits, not how precisely it is stated, so it comes off
// before the fill and goes back on after.
func splitWrittenOffset(s string) (body, offset string) {
	start := 0
	if i := strings.IndexByte(s, 'T'); i >= 0 {
		start = i + 1
	} else if !strings.Contains(s, ":") {
		return s, "" // a date carries no offset
	}
	for i := start; i < len(s); i++ {
		switch s[i] {
		case 'Z', '+', '-':
			return s[:i], s[i:]
		}
	}
	return s, ""
}

// lastDayOfMonthFill adjusts the constant a high boundary is filled from, because
// one component of it is not constant.
//
// The fill takes missing components from "9999-12-31T23:59:59.999", so the day it
// supplies was always the 31st whatever month the value names:
//
//	HighBoundary(@2021-02T, 17)   was 2021-02-31T23:59:59.999
//
// February has no 31st, and neither do April, June, September or November. The
// value was malformed, and while it stayed unplaced the comparisons it reached
// declined; once a boundary takes the request's offset it is a value the engine
// answers with, so `days between @2021-02-01 and that` reported 30 where the month
// has 27 left.
//
// The year and month are already known from the value itself, and they are what
// determines the last day — including February in a leap year, which is why this
// asks the calendar rather than a table.
func lastDayOfMonthFill(existing, boundary string) string {
	// A day can only be determined once a year and a month are both stated. Where
	// the month is itself being filled in, the constant's own December is right.
	if len(existing) < 7 || existing[4] != '-' || len(boundary) < 10 {
		return boundary
	}
	year, err := strconv.Atoi(existing[:4])
	if err != nil {
		return boundary
	}
	month, err := strconv.Atoi(existing[5:7])
	if err != nil || month < 1 || month > 12 {
		return boundary
	}
	// Day zero of the next month is the last day of this one, leap years included.
	last := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return fmt.Sprintf("%s%02d%s", boundary[:8], last, boundary[10:])
}

// temporalHighBoundary fills in missing components of a temporal value with maximum values.
func temporalHighBoundary(s string, targetDigits int, kind string) (fptypes.Value, error) {
	// Max components: for datetime "9999-12-31T23:59:59.999", date "9999-12-31", time "23:59:59.999"
	maxParts := map[string]string{
		"datetime": "9999-12-31T23:59:59.999",
		"date":     "9999-12-31",
		"time":     "23:59:59.999",
	}
	s, writtenOffset := splitWrittenOffset(s)
	maxStr := lastDayOfMonthFill(s, maxParts[kind])

	// Count current digits
	currentDigits := countDigits(s)
	if currentDigits >= targetDigits {
		// Already at or beyond target precision
		switch kind {
		case "datetime":
			return fptypes.NewDateTime(s + writtenOffset)
		case "date":
			return fptypes.NewDate(s)
		case "time":
			return fptypes.NewTime(s + writtenOffset)
		}
		return nil, nil
	}

	// Build result by taking the existing value and filling remaining from max
	result := fillTemporalBoundary(s, maxStr, targetDigits)
	switch kind {
	case "datetime":
		return fptypes.NewDateTime(result + writtenOffset)
	case "date":
		return fptypes.NewDate(result)
	case "time":
		return fptypes.NewTime(result + writtenOffset)
	}
	return nil, nil
}

// temporalLowBoundary fills in missing components with minimum values.
func temporalLowBoundary(s string, targetDigits int, kind string) (fptypes.Value, error) {
	minParts := map[string]string{
		"datetime": "0001-01-01T00:00:00.000",
		"date":     "0001-01-01",
		"time":     "00:00:00.000",
	}
	minStr := minParts[kind]

	s, writtenOffset := splitWrittenOffset(s)
	currentDigits := countDigits(s)
	if currentDigits >= targetDigits {
		switch kind {
		case "datetime":
			return fptypes.NewDateTime(s + writtenOffset)
		case "date":
			return fptypes.NewDate(s)
		case "time":
			return fptypes.NewTime(s + writtenOffset)
		}
		return nil, nil
	}

	result := fillTemporalBoundary(s, minStr, targetDigits)
	switch kind {
	case "datetime":
		return fptypes.NewDateTime(result + writtenOffset)
	case "date":
		return fptypes.NewDate(result)
	case "time":
		return fptypes.NewTime(result + writtenOffset)
	}
	return nil, nil
}

// countDigits counts the number of digit characters in a string.
func countDigits(s string) int {
	count := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			count++
		}
	}
	return count
}

// fillTemporalBoundary takes the existing temporal string and appends from the
// boundary string until we reach the target number of digits.
func fillTemporalBoundary(existing, boundary string, targetDigits int) string {
	// We need to extend 'existing' by appending characters from 'boundary' at the same positions.
	if len(existing) >= len(boundary) {
		return existing
	}
	result := existing + boundary[len(existing):]
	// Trim result to targetDigits worth of digits
	digits := 0
	cutoff := len(result)
	for i, c := range result {
		if c >= '0' && c <= '9' {
			digits++
			if digits == targetDigits {
				cutoff = i + 1
				break
			}
		}
	}
	// Include any trailing non-digit separators
	return result[:cutoff]
}

// carryDefaultOffset copies the offset a source value was told to assume onto a
// value derived from it.
//
// A boundary, a rounding, a shifted instant — these transform a value rather than
// make a new one, so the caller's answer about where the original sat applies to
// the result too. Rebuilding through NewDateTime loses it, which left
// `LowBoundary(v)` unplaced while `v` itself was placed, and the two then
// compared as different things.
func carryDefaultOffset(from, to fptypes.Value) fptypes.Value {
	src, srcOK := from.(fptypes.DateTime)
	dst, dstOK := to.(fptypes.DateTime)
	if !srcOK || !dstOK || src.HasTZ() || dst.HasTZ() {
		return to
	}
	minutes, ok := src.EffectiveOffset()
	if !ok {
		return to
	}
	return dst.WithDefaultOffset(time.Duration(minutes) * time.Minute)
}

// lnOf is the natural logarithm of a positive decimal. A value a float64 holds
// at full precision is taken through math.Log on it, as it always was and as
// the reference engine takes it — Log(10, 1.00001) is 230259.66058852 in both;
// one the float rounds to 1.0 through Log1p on its distance from 1, which the
// float keeps — Log(10, 1.0000000000000000001) was null. A distance below what
// a float holds still reads as 0. One
// a float64 does not hold — a FHIR decimal of 320 digits, or one smaller than
// the smallest normal float, whose digits a subnormal loses — is read from the
// decimal's own coefficient and exponent: ln(c·10^e) is ln(c) + e·ln(10).
// Converting it first made it infinity, zero, or off in the visible digits, and
// Log(1e320, 10) null where it is 320.
func lnOf(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	if f == 1 && !d.Equal(decimal.NewFromInt(1)) {
		x, _ := d.Sub(decimal.NewFromInt(1)).Float64()
		return math.Log1p(x)
	}
	if f >= minNormalFloat && !math.IsInf(f, 0) {
		return math.Log(f)
	}
	mantissa := new(big.Float)
	binaryExp := new(big.Float).SetInt(d.Coefficient()).MantExp(mantissa)
	m, _ := mantissa.Float64()
	return math.Log(m) + float64(binaryExp)*math.Ln2 + float64(d.Exponent())*math.Ln10
}

// minNormalFloat is the smallest normal float64, 2^-1022. Below it a float
// keeps fewer significant digits than a logarithm needs.
const minNormalFloat = 2.2250738585072014e-308

// maxExactPower bounds the powers exactLog will compute; a decimal has at
// most a thousand digits on either side of the point, so no exact power of a
// base past 1 lies beyond it.
const maxExactPower = 4000

// maxDecimalDigits bounds the significant digits of any decimal: a thousand
// on either side of the point.
const maxDecimalDigits = 2000

// maxLogDenominator is the largest denominator of a logarithm exactLog tries
// to recognize as an exact fraction.
const maxLogDenominator = 12

// exactLog is a logarithm that is a fraction k/q with a small denominator,
// when it is one exactly. The float around it is not to be trusted with that:
// ln 1000 / ln 10 is 2.9999999999999996, which floors to 2.99999999 — the Java
// engine's answer — where the logarithm is 3, and log base 100 of 1000 floors
// to 1.49999999 where it is 1.5. Where the float is within 1e-9 of k/q, v^q is
// compared with base^k exactly, in decimal, and only an exact match answers
// k/q, at the Decimal scale. A value merely near a power, like
// 999.999999999999, floors as the float says.
func exactLog(v, base decimal.Decimal, result float64) (decimal.Decimal, bool) {
	for q := int64(1); q <= maxLogDenominator; q++ {
		k := math.Round(result * float64(q))
		if math.IsNaN(k) || math.Abs(result*float64(q)-k) >= 1e-9*float64(q) || math.Abs(k) > maxExactPower {
			continue
		}
		if powersEqual(v, q, base, int64(k)) {
			return DivideScaled(decimal.NewFromInt(int64(k)), decimal.NewFromInt(q)), true
		}
	}
	return decimal.Zero, false
}

// powersEqual reports whether v^q is exactly base^k, for q ≥ 1.
//
// A number of n significant digits to the k has at least |k|·(n-1) of them,
// so a power past what any decimal holds is refused before it is computed:
// with a base of a thousand digits that power ran to millions of them, and
// took a third of a second to compare.
func powersEqual(v decimal.Decimal, q int64, base decimal.Decimal, k int64) bool {
	if tooLong(base, k) || tooLong(v, q) {
		return false
	}
	left := v.Pow(decimal.NewFromInt(q))
	if k >= 0 {
		return left.Equal(base.Pow(decimal.NewFromInt(k)))
	}
	return left.Mul(base.Pow(decimal.NewFromInt(-k))).Equal(decimal.NewFromInt(1))
}

// tooLong reports whether d to the power k has more significant digits than a
// decimal holds. Trailing zeros are not digits of the power: 10.0000 is 10.
func tooLong(d decimal.Decimal, k int64) bool {
	if k < 0 {
		k = -k
	}
	n := int64(len(strings.TrimRight(d.Coefficient().String(), "0")))
	return k*(n-1) > maxDecimalDigits
}

// ExactPower is base to a whole-number exponent, computed in decimal and
// brought to the Decimal scale. exact is false where the exponent is not a
// whole number, the power would run past what a decimal holds, or there is none
// to give — zero to a negative power. overflow is true where the power is past
// the largest Decimal: that answer is final, and Power gives null rather than
// asking the float, which can land just under the bound — 0.1 ^ -20 is 1e20,
// and the float made it 99999999999999790000.
//
// Through a float, a power of a decimal lands just off it: 0.21 ^ 2 is the
// float 0.044099999999999993, which floors to 0.04409999 — the Java engine's
// answer — where 0.21 * 0.21 is 0.0441, and the two disagreed. A whole-number
// power is repeated multiplication, and it is computed as one; a negative one
// is its quotient, so 2 ^ -1 is 0.5.
func ExactPower(base, exponent decimal.Decimal) (power decimal.Decimal, exact, overflow bool) {
	if !exponent.IsInteger() {
		return decimal.Zero, false, false
	}
	// Zero, one and minus one answer for any whole exponent, however large,
	// and from the exponent itself: through a float, an odd exponent past 2^53
	// rounds to an even one, and (-1) ^ 9007199254740993 was 1.
	one := decimal.NewFromInt(1)
	switch {
	case exponent.IsZero():
		return one, true, false // 0.0 ^ 0 included, as math.Pow has it
	case base.IsZero():
		if exponent.Sign() < 0 {
			return decimal.Zero, false, false
		}
		return decimal.Zero, true, false
	case base.Equal(one):
		return one, true, false
	case base.Equal(one.Neg()):
		if exponent.Mod(decimal.NewFromInt(2)).IsZero() {
			return one, true, false
		}
		return one.Neg(), true, false
	}
	// Bounded before it is read as an int64, which wraps one past it in
	// silence: 2.0 ^ 18446744073709551616.0 was computed as 2.0 ^ 0.
	if exponent.Abs().GreaterThan(decimal.NewFromInt(maxExactPower)) {
		return decimal.Zero, false, false
	}
	k := exponent.IntPart()
	// A power plainly past the largest Decimal is refused before it is
	// computed, from its magnitude: 1.1 ^ 2000 has 83 digits before the point,
	// and a tiny value to a large negative power as many.
	if magnitude, _ := base.Abs().Float64(); float64(k)*math.Log10(magnitude) > maxDecimalDigitsBeforePoint+1 {
		return decimal.Zero, false, true
	}
	if tooLong(base, k) {
		return decimal.Zero, false, false
	}
	if k > 0 {
		power = ScaleDecimal(base.Pow(decimal.NewFromInt(k)))
	} else {
		power = DivideScaled(one, base.Pow(decimal.NewFromInt(-k)))
	}
	if power.Abs().GreaterThan(maxDecimal) {
		return decimal.Zero, false, true
	}
	return power, true, false
}

// maxDecimal is the largest value of the Decimal type, `maximum Decimal`, and
// maxDecimalDigitsBeforePoint its count of digits before the point.
var maxDecimal = decimal.RequireFromString("99999999999999999999.99999999")

const maxDecimalDigitsBeforePoint = 20
