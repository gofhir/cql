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

// Avg returns the average of the numbers in a collection, skipping nulls, at the
// Decimal scale: each addition scaled as + is, and the quotient as / is.
//
// It is the CQL Avg operator over numbers — the evaluator hands it the list once
// it has ruled out quantities and uncertainties, which it averages itself — and
// the mean Variance and PopulationVariance measure from, so the two scale it
// alike. A value is read through NumberOf, the one reading of a number.
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
		// From the first value, each further addition scaled, as Sum adds.
		if count == 0 {
			sum = numericVal(item)
		} else {
			sum = ScaleDecimal(sum.Add(numericVal(item)))
		}
		count++
	}
	if count == 0 {
		return nil
	}
	return decimalToValue(DivideScaled(sum, decimal.NewFromInt(count)))
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
//
// It needs one value, not two: the population variance of a single value is 0,
// as the reference engine answers, and null is for a list with no values at
// all. Only the sample variance divides by n - 1 and has nothing to say about
// one. Asking for two made PopulationVariance({3.0}) and PopulationStdDev of it
// null.
func PopulationVariance(c fptypes.Collection) fptypes.Value {
	nn := nonNullItems(c)
	if nn.Count() < 1 {
		return nil
	}
	return decimalToValue(scaledVariance(nn, int64(nn.Count())))
}

// scaledVariance is the sum of squared deviations from the mean over divisor,
// built from the operations the reference engine builds it from, each at the
// Decimal scale: the mean is Avg, each square a product and the total a sum,
// and the subtraction is left unscaled as - is. That is why the variance of
// {1.0, 2.0, 4.0} is 2.33333332 there, and here, and not 2.33333333.
func scaledVariance(nn fptypes.Collection, divisor int64) decimal.Decimal {
	mean := numericVal(Avg(nn))
	sumSq := decimal.Zero
	for _, item := range nn {
		diff := numericVal(item).Sub(mean)
		sumSq = ScaleDecimal(sumSq.Add(ScaleDecimal(diff.Mul(diff))))
	}
	return DivideScaled(sumSq, decimal.NewFromInt(divisor))
}

// scaledRoot is the square root of a variance, as a power of 0.5 at the Decimal
// scale, which is how the reference engine takes a standard deviation.
func scaledRoot(v decimal.Decimal) fptypes.Value {
	f, _ := v.Float64()
	root, ok := FloatScaled(math.Pow(f, 0.5))
	if !ok {
		return nil
	}
	return decimalToValue(root)
}

// PopulationStdDev computes the population standard deviation.
func PopulationStdDev(c fptypes.Collection) fptypes.Value {
	variance := PopulationVariance(c)
	if variance == nil {
		return nil
	}
	return scaledRoot(numericVal(variance))
}

// Variance computes the sample variance.
func Variance(c fptypes.Collection) fptypes.Value {
	nn := nonNullItems(c)
	if nn.Count() < 2 {
		return nil
	}
	return decimalToValue(scaledVariance(nn, int64(nn.Count()-1)))
}

// StdDev computes the sample standard deviation.
func StdDev(c fptypes.Collection) fptypes.Value {
	v := Variance(c)
	if v == nil {
		return nil
	}
	return scaledRoot(numericVal(v))
}

func numericVal(v fptypes.Value) decimal.Decimal {
	n, _ := NumberOf(v)
	return n
}

// NumberOf reads a number, reporting whether v is one. Anything that is not —
// a null, a string — is zero and false. It is the one reading of a number: the
// evaluator's toDecimal is this, so Avg, the variances and Median here and Sum
// and Product there cannot read a value differently.
func NumberOf(v fptypes.Value) (decimal.Decimal, bool) {
	switch n := v.(type) {
	case nil:
		return decimal.Zero, false
	case fptypes.Integer:
		return decimal.NewFromInt(n.Value()), true
	case fptypes.Decimal:
		return n.Value(), true
	case fptypes.Numeric:
		return n.ToDecimal().Value(), true
	}
	return decimal.Zero, false
}

func decimalToValue(d decimal.Decimal) fptypes.Value {
	v, err := fptypes.NewDecimal(d.String())
	if err != nil {
		return nil
	}
	return v
}
