package eval

import (
	"fmt"

	fptypes "github.com/gofhir/fhirpath/types"
)

// QuantityConverter converts between UCUM units.
// Implement and set on Context.QuantityConverter to support ConvertQuantity/CanConvertQuantity.
type QuantityConverter interface {
	ConvertQuantity(value float64, fromUnit, toUnit string) (float64, error)
}

func (e *Evaluator) evalConvertQuantity(src fptypes.Value, targetUnit string) (fptypes.Value, error) {
	if src == nil {
		return nil, nil
	}
	if e.ctx.QuantityConverter == nil {
		return nil, nil
	}
	q, ok := src.(fptypes.Quantity)
	if !ok {
		return nil, nil
	}
	val, _ := q.Value().Float64()
	result, err := e.ctx.QuantityConverter.ConvertQuantity(val, q.Unit(), targetUnit)
	if err != nil {
		return nil, nil // null on failed conversion per CQL spec
	}
	return fptypes.NewQuantity(fmt.Sprintf("%g '%s'", result, targetUnit))
}

func (e *Evaluator) evalCanConvertQuantity(src fptypes.Value, targetUnit string) (fptypes.Value, error) {
	if src == nil {
		return nil, nil
	}
	if e.ctx.QuantityConverter == nil {
		return nil, nil
	}
	q, ok := src.(fptypes.Quantity)
	if !ok {
		return fptypes.NewBoolean(false), nil
	}
	_, err := e.ctx.QuantityConverter.ConvertQuantity(1.0, q.Unit(), targetUnit)
	return fptypes.NewBoolean(err == nil), nil
}
