package funcs

import (
	"fmt"
	"math"
	"strings"

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
// If the value is nil, returns nil.
func Round(v fptypes.Value, precision int) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	d, ok := toDecimalVal(v)
	if !ok {
		return nil, fmt.Errorf("Round: expected numeric, got %s", v.Type())
	}
	rounded := d.Round(int32(precision))
	return decimalToValue(rounded), nil
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
	floored := d.Floor()
	return fptypes.NewInteger(floored.IntPart()), nil
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
	ceiled := d.Ceil()
	return fptypes.NewInteger(ceiled.IntPart()), nil
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
	if f <= 0 {
		return nil, nil // CQL: undefined for non-positive values
	}
	result := math.Log(f)
	return decimalToValue(decimal.NewFromFloat(result)), nil
}

// Log returns the logarithm of value with the given base.
func Log(v fptypes.Value, base fptypes.Value) (fptypes.Value, error) {
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
		return nil, nil
	}
	return decimalToValue(decimal.NewFromFloat(result)), nil
}

// Power returns value raised to the exponent power.
func Power(v fptypes.Value, exp fptypes.Value) (fptypes.Value, error) {
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

// Precision returns the number of digits of precision in the decimal representation.
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
		d := val.Value()
		s := d.String()
		// Remove sign
		s = strings.TrimLeft(s, "-")
		// Count digits after decimal point
		parts := strings.Split(s, ".")
		if len(parts) == 2 {
			return fptypes.NewInteger(int64(len(parts[1]))), nil
		}
		return fptypes.NewInteger(0), nil
	default:
		// For date/time types, precision could mean the number of specified components
		// Return null for unsupported types
		return nil, nil
	}
}

// HighBoundary returns the high boundary of a value at the given precision.
func HighBoundary(v fptypes.Value, precision fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	switch val := v.(type) {
	case fptypes.Decimal:
		prec := int32(8) // default precision
		if precision != nil {
			if pi, ok := precision.(fptypes.Integer); ok {
				prec = int32(pi.Value())
			}
		}
		d := val.Value()
		// The high boundary is the value + half of the last precision digit
		increment := decimal.New(5, -prec-1)
		result := d.Add(increment)
		return decimalToValue(result.Truncate(prec)), nil
	case fptypes.Integer:
		// For integers, high boundary is value + 0.5, but we return the integer itself
		return val, nil
	default:
		return nil, nil
	}
}

// LowBoundary returns the low boundary of a value at the given precision.
func LowBoundary(v fptypes.Value, precision fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	switch val := v.(type) {
	case fptypes.Decimal:
		prec := int32(8) // default precision
		if precision != nil {
			if pi, ok := precision.(fptypes.Integer); ok {
				prec = int32(pi.Value())
			}
		}
		d := val.Value()
		// The low boundary is the value - half of the last precision digit
		decrement := decimal.New(5, -prec-1)
		result := d.Sub(decrement)
		return decimalToValue(result.Truncate(prec)), nil
	case fptypes.Integer:
		return val, nil
	default:
		return nil, nil
	}
}
