package funcs

import (
	"math"

	fptypes "github.com/gofhir/fhirpath/types"
	"github.com/shopspring/decimal"
)

// Count returns the number of elements in a collection.
func Count(c fptypes.Collection) fptypes.Value {
	return fptypes.NewInteger(int64(c.Count()))
}

// Sum returns the sum of all numeric values in a collection.
func Sum(c fptypes.Collection) fptypes.Value {
	if c.Empty() {
		return nil
	}
	sum := decimal.Zero
	for _, item := range c {
		sum = sum.Add(numericVal(item))
	}
	return decimalToValue(sum)
}

// Avg returns the average of all numeric values in a collection.
func Avg(c fptypes.Collection) fptypes.Value {
	if c.Empty() {
		return nil
	}
	sum := decimal.Zero
	for _, item := range c {
		sum = sum.Add(numericVal(item))
	}
	return decimalToValue(sum.Div(decimal.NewFromInt(int64(c.Count()))))
}

// Min returns the minimum value in a collection.
func Min(c fptypes.Collection) fptypes.Value {
	if c.Empty() {
		return nil
	}
	result := c[0]
	for _, item := range c[1:] {
		comp, ok := result.(fptypes.Comparable)
		if !ok {
			continue
		}
		cmp, err := comp.Compare(item)
		if err != nil {
			continue
		}
		if cmp > 0 {
			result = item
		}
	}
	return result
}

// Max returns the maximum value in a collection.
func Max(c fptypes.Collection) fptypes.Value {
	if c.Empty() {
		return nil
	}
	result := c[0]
	for _, item := range c[1:] {
		comp, ok := result.(fptypes.Comparable)
		if !ok {
			continue
		}
		cmp, err := comp.Compare(item)
		if err != nil {
			continue
		}
		if cmp < 0 {
			result = item
		}
	}
	return result
}

// AllTrue returns true if all items in the collection are true.
func AllTrue(c fptypes.Collection) fptypes.Value {
	for _, item := range c {
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
		b, ok := item.(fptypes.Boolean)
		if ok && b.Bool() {
			return fptypes.NewBoolean(true)
		}
	}
	return fptypes.NewBoolean(false)
}

// PopulationVariance computes the population variance.
func PopulationVariance(c fptypes.Collection) fptypes.Value {
	if c.Count() < 2 {
		return nil
	}
	mean := numericVal(Avg(c))
	sumSq := decimal.Zero
	for _, item := range c {
		diff := numericVal(item).Sub(mean)
		sumSq = sumSq.Add(diff.Mul(diff))
	}
	return decimalToValue(sumSq.Div(decimal.NewFromInt(int64(c.Count()))))
}

// PopulationStdDev computes the population standard deviation.
func PopulationStdDev(c fptypes.Collection) fptypes.Value {
	variance := PopulationVariance(c)
	if variance == nil {
		return nil
	}
	v := numericVal(variance)
	f, _ := v.Float64()
	return fptypes.NewDecimalFromFloat(math.Sqrt(f))
}

// Variance computes the sample variance.
func Variance(c fptypes.Collection) fptypes.Value {
	if c.Count() < 2 {
		return nil
	}
	mean := numericVal(Avg(c))
	sumSq := decimal.Zero
	for _, item := range c {
		diff := numericVal(item).Sub(mean)
		sumSq = sumSq.Add(diff.Mul(diff))
	}
	return decimalToValue(sumSq.Div(decimal.NewFromInt(int64(c.Count() - 1))))
}

// StdDev computes the sample standard deviation.
func StdDev(c fptypes.Collection) fptypes.Value {
	v := Variance(c)
	if v == nil {
		return nil
	}
	val := numericVal(v)
	f, _ := val.Float64()
	return fptypes.NewDecimalFromFloat(math.Sqrt(f))
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
	v, _ := fptypes.NewDecimal(d.String())
	return v
}
