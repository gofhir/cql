package funcs

import (
	"fmt"
	"strconv"
	"strings"

	fptypes "github.com/gofhir/fhirpath/types"
)

// ToString converts a value to its string representation.
func ToString(v fptypes.Value) fptypes.Value {
	if v == nil {
		return nil
	}
	return fptypes.NewString(v.String())
}

// ToInteger converts a value to an integer.
func ToInteger(v fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	switch val := v.(type) {
	case fptypes.Integer:
		return val, nil
	case fptypes.String:
		i, err := strconv.ParseInt(val.Value(), 10, 64)
		if err != nil {
			return nil, nil // CQL: failed conversions return null
		}
		return fptypes.NewInteger(i), nil
	case fptypes.Boolean:
		if val.Bool() {
			return fptypes.NewInteger(1), nil
		}
		return fptypes.NewInteger(0), nil
	case fptypes.Decimal:
		return fptypes.NewInteger(val.Value().IntPart()), nil
	default:
		return nil, nil
	}
}

// ToDecimal converts a value to a decimal.
func ToDecimal(v fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	switch val := v.(type) {
	case fptypes.Decimal:
		return val, nil
	case fptypes.Integer:
		return fptypes.NewDecimalFromInt(val.Value()), nil
	case fptypes.String:
		d, err := fptypes.NewDecimal(val.Value())
		if err != nil {
			return nil, nil
		}
		return d, nil
	default:
		return nil, nil
	}
}

// ToBoolean converts a value to a boolean.
func ToBoolean(v fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	switch val := v.(type) {
	case fptypes.Boolean:
		return val, nil
	case fptypes.String:
		s := strings.ToLower(val.Value())
		switch s {
		case "true", "t", "yes", "y", "1", "1.0":
			return fptypes.NewBoolean(true), nil
		case "false", "f", "no", "n", "0", "0.0":
			return fptypes.NewBoolean(false), nil
		default:
			return nil, nil
		}
	case fptypes.Integer:
		return fptypes.NewBoolean(val.Value() != 0), nil
	case fptypes.Decimal:
		return fptypes.NewBoolean(!val.Value().IsZero()), nil
	default:
		return nil, nil
	}
}

// ToDateTime converts a value to a DateTime.
func ToDateTime(v fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	switch v.(type) {
	case fptypes.DateTime:
		return v, nil
	case fptypes.Date:
		return fptypes.NewDateTime(v.String() + "T00:00:00")
	case fptypes.String:
		return fptypes.NewDateTime(v.String())
	default:
		return nil, nil
	}
}

// ToDate converts a value to a Date.
func ToDate(v fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	switch v.(type) {
	case fptypes.Date:
		return v, nil
	case fptypes.DateTime:
		// Extract date part
		s := v.String()
		if idx := strings.IndexByte(s, 'T'); idx >= 0 {
			s = s[:idx]
		}
		return fptypes.NewDate(s)
	case fptypes.String:
		return fptypes.NewDate(v.String())
	default:
		return nil, nil
	}
}

// ToQuantity converts a value to a Quantity.
func ToQuantity(v fptypes.Value) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	if _, ok := v.(fptypes.Quantity); ok {
		return v, nil
	}
	return fptypes.NewQuantity(v.String())
}

// IsType checks if a value matches the given type name.
func IsType(v fptypes.Value, typeName string) bool {
	if v == nil {
		return false
	}
	return strings.EqualFold(v.Type(), typeName)
}

// AsType performs a safe cast, returning nil if the type doesn't match.
func AsType(v fptypes.Value, typeName string) fptypes.Value {
	if v == nil {
		return nil
	}
	if strings.EqualFold(v.Type(), typeName) {
		return v
	}
	return nil
}

// Convert attempts to convert a value to the specified type.
func Convert(v fptypes.Value, typeName string) (fptypes.Value, error) {
	if v == nil {
		return nil, nil
	}
	switch strings.ToLower(typeName) {
	case "string", "system.string":
		return ToString(v), nil
	case "integer", "system.integer":
		return ToInteger(v)
	case "decimal", "system.decimal":
		return ToDecimal(v)
	case "boolean", "system.boolean":
		return ToBoolean(v)
	case "datetime", "system.datetime":
		return ToDateTime(v)
	case "date", "system.date":
		return ToDate(v)
	case "quantity", "system.quantity":
		return ToQuantity(v)
	default:
		return nil, fmt.Errorf("cannot convert to %s", typeName)
	}
}
