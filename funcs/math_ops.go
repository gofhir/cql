package funcs

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	fptypes "github.com/gofhir/fhirpath/types"
)

// toDecimalVal converts an fptypes.Value to a decimal.Decimal.
// Returns false if the value is not numeric.
func toDecimalVal(v fptypes.Value) (decimal.Decimal, bool) {
	if i, ok := v.(fptypes.Integer); ok {
		return decimal.NewFromInt(i.Value()), true
	}
	if d, ok := v.(fptypes.Decimal); ok {
		return d.Value(), true
	}
	return decimal.Zero, false
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
	// If already an integer, return as-is.
	if _, ok := v.(fptypes.Integer); ok {
		return v, nil
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
	if _, ok := v.(fptypes.Integer); ok {
		return v, nil
	}
	d, ok := toDecimalVal(v)
	if !ok {
		return nil, fmt.Errorf("Ceiling: expected numeric, got %s", v.Type())
	}
	return integerOrNull(d.Ceil()), nil
}

// integerOrNull returns d as an Integer, or null when it falls outside the range
// CQL gives that type. Floor and Ceiling are the two operators that can carry a
// Decimal past the Integer bounds while still being asked for an Integer, and CQL
// answers that with null rather than a wrapped or clamped number.
func integerOrNull(d decimal.Decimal) fptypes.Value {
	if d.LessThan(decimal.NewFromInt(math.MinInt32)) || d.GreaterThan(decimal.NewFromInt(math.MaxInt32)) {
		return nil
	}
	return fptypes.NewInteger(d.IntPart())
}

// Truncate removes the fractional part of a decimal value, returning an integer.
func Truncate(v fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	if _, ok := v.(fptypes.Integer); ok {
		return v, nil
	}
	d, ok := toDecimalVal(v)
	if !ok {
		return nil, fmt.Errorf("Truncate: expected numeric, got %s", v.Type())
	}
	truncated := d.Truncate(0)
	return fptypes.NewInteger(truncated.IntPart()), nil
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
	f, _ := d.Float64()
	if d.IsZero() {
		return nil, fmt.Errorf("Ln: undefined for zero (results in negative infinity)")
	}
	if f < 0 {
		return nil, nil // CQL: Ln of negative value returns null
	}
	result := math.Log(f)
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
	fv, _ := d.Float64()
	fb, _ := b.Float64()
	if fv <= 0 || fb <= 0 || fb == 1 {
		return nil, nil // CQL: undefined
	}
	result := math.Log(fv) / math.Log(fb)
	return decimalToValue(decimal.NewFromFloat(result)), nil
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
	fv, _ := d.Float64()
	fe, _ := e.Float64()
	result := math.Pow(fv, fe)
	if math.IsInf(result, 0) || math.IsNaN(result) {
		return nil, nil
	}
	// If both inputs are integers and exponent is non-negative, return integer
	_, vIsInt := v.(fptypes.Integer)
	_, eIsInt := exp.(fptypes.Integer)
	if vIsInt && eIsInt && fe >= 0 {
		return fptypes.NewInteger(int64(math.Round(result))), nil
	}
	return decimalToValue(decimal.NewFromFloat(result)), nil
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
	// sits: an offset the caller supplied for v applies to the result. Rebuilding
	// through NewDateTime inside would otherwise lose it.
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

// temporalHighBoundary fills in missing components of a temporal value with maximum values.
func temporalHighBoundary(s string, targetDigits int, kind string) (fptypes.Value, error) {
	// Max components: for datetime "9999-12-31T23:59:59.999", date "9999-12-31", time "23:59:59.999"
	maxParts := map[string]string{
		"datetime": "9999-12-31T23:59:59.999",
		"date":     "9999-12-31",
		"time":     "23:59:59.999",
	}
	maxStr := maxParts[kind]

	// Count current digits
	currentDigits := countDigits(s)
	if currentDigits >= targetDigits {
		// Already at or beyond target precision
		switch kind {
		case "datetime":
			return fptypes.NewDateTime(s)
		case "date":
			return fptypes.NewDate(s)
		case "time":
			return fptypes.NewTime(s)
		}
		return nil, nil
	}

	// Build result by taking the existing value and filling remaining from max
	result := fillTemporalBoundary(s, maxStr, targetDigits)
	switch kind {
	case "datetime":
		return fptypes.NewDateTime(result)
	case "date":
		return fptypes.NewDate(result)
	case "time":
		return fptypes.NewTime(result)
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

	currentDigits := countDigits(s)
	if currentDigits >= targetDigits {
		switch kind {
		case "datetime":
			return fptypes.NewDateTime(s)
		case "date":
			return fptypes.NewDate(s)
		case "time":
			return fptypes.NewTime(s)
		}
		return nil, nil
	}

	result := fillTemporalBoundary(s, minStr, targetDigits)
	switch kind {
	case "datetime":
		return fptypes.NewDateTime(result)
	case "date":
		return fptypes.NewDate(result)
	case "time":
		return fptypes.NewTime(result)
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
