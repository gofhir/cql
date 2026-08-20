package funcs

import (
	"math"

	"github.com/shopspring/decimal"

	fptypes "github.com/gofhir/fhirpath/types"
)

// Count returns the number of non-null elements in a collection.
func Count(c fptypes.Collection) fptypes.Value {
	count := int64(0)
	for _, item := range c {
		if item != nil {
			count++
		}
	}
	return fptypes.NewInteger(count)
}

// Avg returns the average of all numeric values in a collection (skipping nulls).
//
// Not the CQL Avg operator, which the evaluator implements: this reads a value
// through numericVal, which reports zero for a Quantity or anything else that is
// not an Integer or a Decimal. It stays because Variance and PopulationVariance
// need a mean, and they are only reached on the numeric path — the evaluator
// intercepts quantities before either is called.
//
// Sum, Min and Max sat beside it with the same limitation and nothing calling
// them, which is how a caller comes to use the version that answers zero.
func Avg(c fptypes.Collection) fptypes.Value {
	if c.Empty() {
		return nil
	}
	sum := decimal.Zero
	count := int64(0)
	for _, item := range c {
		if item == nil {
			continue
		}
		sum = sum.Add(numericVal(item))
		count++
	}
	if count == 0 {
		return nil
	}
	return decimalToValue(sum.Div(decimal.NewFromInt(count)))
}

// AllTrue returns true if all non-null items in the collection are true.
// CQL: null values are ignored.
func AllTrue(c fptypes.Collection) fptypes.Value {
	for _, item := range c {
		if item == nil {
			continue // CQL: ignore nulls
		}
		b, ok := item.(fptypes.Boolean)
		if !ok || !b.Bool() {
			return fptypes.NewBoolean(false)
		}
	}
	return fptypes.NewBoolean(true)
}

// AnyTrue returns true if any item in the collection is true.
func AnyTrue(c fptypes.Collection) fptypes.Value {
	for _, item := range c {
		if item == nil {
			continue
		}
		b, ok := item.(fptypes.Boolean)
		if ok && b.Bool() {
			return fptypes.NewBoolean(true)
		}
	}
	return fptypes.NewBoolean(false)
}

// nonNullItems filters out nil elements from a collection.
func nonNullItems(c fptypes.Collection) fptypes.Collection {
	result := make(fptypes.Collection, 0, len(c))
	for _, item := range c {
		if item != nil {
			result = append(result, item)
		}
	}
	return result
}

// PopulationVariance computes the population variance.
func PopulationVariance(c fptypes.Collection) fptypes.Value {
	nn := nonNullItems(c)
	if nn.Count() < 2 {
		return nil
	}
	mean := numericVal(Avg(nn))
	sumSq := decimal.Zero
	for _, item := range nn {
		diff := numericVal(item).Sub(mean)
		sumSq = sumSq.Add(diff.Mul(diff))
	}
	return decimalToValue(sumSq.Div(decimal.NewFromInt(int64(nn.Count()))))
}

// PopulationStdDev computes the population standard deviation.
func PopulationStdDev(c fptypes.Collection) fptypes.Value {
	variance := PopulationVariance(c)
	if variance == nil {
		return nil
	}
	v := numericVal(variance)
	f, _ := v.Float64()
	return decimalToValue(decimal.NewFromFloat(math.Sqrt(f)).Round(8))
}

// Variance computes the sample variance.
func Variance(c fptypes.Collection) fptypes.Value {
	nn := nonNullItems(c)
	if nn.Count() < 2 {
		return nil
	}
	mean := numericVal(Avg(nn))
	sumSq := decimal.Zero
	for _, item := range nn {
		diff := numericVal(item).Sub(mean)
		sumSq = sumSq.Add(diff.Mul(diff))
	}
	return decimalToValue(sumSq.Div(decimal.NewFromInt(int64(nn.Count() - 1))))
}

// StdDev computes the sample standard deviation.
func StdDev(c fptypes.Collection) fptypes.Value {
	v := Variance(c)
	if v == nil {
		return nil
	}
	val := numericVal(v)
	f, _ := val.Float64()
	return decimalToValue(decimal.NewFromFloat(math.Sqrt(f)).Round(8))
}

func numericVal(v fptypes.Value) decimal.Decimal {
	if v == nil {
		return decimal.Zero
	}
	if i, ok := v.(fptypes.Integer); ok {
		return decimal.NewFromInt(i.Value())
	}
	if d, ok := v.(fptypes.Decimal); ok {
		return d.Value()
	}
	return decimal.Zero
}

func decimalToValue(d decimal.Decimal) fptypes.Value {
	v, err := fptypes.NewDecimal(d.String())
	if err != nil {
		return nil
	}
	return v
}
