package eval

import (
	"fmt"
	"math"

	"github.com/shopspring/decimal"

	fptypes "github.com/gofhir/fhirpath/types"
)

// The aggregates did not understand Quantity, and the way they did not was the
// dangerous one: they answered anyway.
//
//	Avg({ 1 'mg', 2 'mg' })     was 0        — not null, a number
//	Median({ 1 'mg', 2 'mg' })  was null
//	Sum({ 1 'mg', 2 'kg' })     was 3 'mg'   — 2 kg counted as 2
//	Sum({ 1 'mg', 1 'g' })      was 2 'mg'
//	Product({ 2 'mg', 3 'mg' }) was 0
//
// A dose of 0 mg or a total that silently drops six orders of magnitude reads
// as a real measurement wherever it lands. The rules for this already existed
// and the aggregates were not using them: `1 'mg' + 1 'g'` is 1001 'mg' and
// `1 'mg' + 1 's'` is an error, because the + operator asks Quantity.Add.
// These do the same, so there is one policy on units rather than two.

// quantityOperands reports the Quantities in a collection.
//
// Mixing Quantities with bare numbers is refused rather than averaged: `1 'mg'
// + 2` is already an error, and a collection is no more able to say what unit
// the 2 carries. Silently skipping it is how Sum({ 1 'mg', 2 }) came to be
// 1 'mg'.
func quantityOperands(c fptypes.Collection) (quantities []fptypes.Quantity, found bool, err error) {
	var others int
	for _, item := range c {
		if item == nil {
			continue
		}
		q, ok := item.(fptypes.Quantity)
		if !ok {
			others++
			continue
		}
		quantities = append(quantities, q)
	}
	if len(quantities) == 0 {
		return nil, false, nil
	}
	if others > 0 {
		return nil, true, fmt.Errorf("cannot aggregate Quantity values together with %d non-quantity value(s)", others)
	}
	return quantities, true, nil
}

// sumQuantities adds Quantities the way the + operator does, converting units
// that are comparable and refusing units that are not.
func sumQuantities(quantities []fptypes.Quantity) (fptypes.Quantity, error) {
	total := quantities[0]
	for _, q := range quantities[1:] {
		sum, err := total.Add(q)
		if err != nil {
			return fptypes.Quantity{}, err
		}
		total = sum
	}
	return total, nil
}

// avgQuantities is the sum over the count, keeping the unit.
func avgQuantities(quantities []fptypes.Quantity) (fptypes.Value, error) {
	total, err := sumQuantities(quantities)
	if err != nil {
		return nil, err
	}
	avg, err := total.Divide(decimal.NewFromInt(int64(len(quantities))))
	if err != nil {
		return nil, err
	}
	return avg, nil
}

// productOfQuantities multiplies through, compounding units the way the *
// operator does: mg by mg is mg2, mg by mL is mg.mL.
func productOfQuantities(quantities []fptypes.Quantity) fptypes.Value {
	value := quantities[0].Value()
	unit := quantities[0].Unit()
	for _, q := range quantities[1:] {
		value = value.Mul(q.Value())
		unit = multiplyUnits(unit, q.Unit())
	}
	return fptypes.NewQuantityFromDecimal(value, unit)
}

// varianceOfQuantities is the mean squared deviation, in the square of the
// unit — the variance of milligrams is in mg2, and its square root is back in
// mg. Sample variance divides by n-1, population variance by n.
func varianceOfQuantities(quantities []fptypes.Quantity, sample bool) (fptypes.Value, error) {
	divisor := len(quantities)
	if sample {
		divisor--
	}
	if divisor <= 0 {
		// One sample says nothing about spread, which is why CQL's sample
		// variance of a single value is null rather than zero.
		return nil, nil
	}
	mean, err := meanQuantity(quantities)
	if err != nil {
		return nil, err
	}
	total := decimal.Zero
	for _, q := range quantities {
		// Subtract through Quantity so that a list mixing mg and g is brought
		// to one unit before the deviation is squared.
		deviation, err := q.Subtract(mean)
		if err != nil {
			return nil, err
		}
		total = total.Add(deviation.Value().Mul(deviation.Value()))
	}
	variance := trimTrailingZeros(total.Div(decimal.NewFromInt(int64(divisor))))
	return fptypes.NewQuantityFromDecimal(variance, multiplyUnits(mean.Unit(), mean.Unit())), nil
}

// trimTrailingZeros drops the zeros a division leaves behind, so that a
// variance of one prints as 1 rather than 1.0000000000000000 — decimal keeps
// the scale it divided at, and the numeric aggregates print the same figure
// without it.
func trimTrailingZeros(d decimal.Decimal) decimal.Decimal {
	for d.Exponent() < 0 {
		shorter := d.Truncate(-d.Exponent() - 1)
		if !shorter.Equal(d) {
			break
		}
		d = shorter
	}
	return d
}

// stdDevOfQuantities is the square root of the variance, which returns the
// original unit.
func stdDevOfQuantities(quantities []fptypes.Quantity, sample bool) (fptypes.Value, error) {
	variance, err := varianceOfQuantities(quantities, sample)
	if err != nil || variance == nil {
		return nil, err
	}
	v, ok := variance.(fptypes.Quantity)
	if !ok {
		return nil, nil
	}
	// Through float64 and rounded to 8 places, the same way funcs.StdDev takes
	// its root, so a quantity and a plain number do not disagree in the last
	// digits over the same figures.
	f, _ := v.Value().Float64()
	root := trimTrailingZeros(decimal.NewFromFloat(math.Sqrt(f)).Round(8))
	return fptypes.NewQuantityFromDecimal(root, quantities[0].Unit()), nil
}

// meanQuantity is the average, kept separate because variance needs it before
// it can measure anything.
func meanQuantity(quantities []fptypes.Quantity) (fptypes.Quantity, error) {
	total, err := sumQuantities(quantities)
	if err != nil {
		return fptypes.Quantity{}, err
	}
	return total.Divide(decimal.NewFromInt(int64(len(quantities))))
}

// medianQuantities sorts by Quantity.Compare, which already knows that 1 'g' is
// more than 1 'mg', and averages the middle pair when the count is even.
func medianQuantities(quantities []fptypes.Quantity) (fptypes.Value, error) {
	sorted, err := sortQuantities(quantities)
	if err != nil {
		return nil, err
	}
	mid := len(sorted) / 2
	if len(sorted)%2 != 0 {
		return sorted[mid], nil
	}
	return avgQuantities([]fptypes.Quantity{sorted[mid-1], sorted[mid]})
}

// sortQuantities orders a copy ascending, reporting the first pair it cannot
// compare instead of settling for whatever order that leaves.
func sortQuantities(quantities []fptypes.Quantity) ([]fptypes.Quantity, error) {
	sorted := make([]fptypes.Quantity, len(quantities))
	copy(sorted, quantities)
	// An insertion sort, so that a comparison failure stops the whole
	// operation: sort.Slice has nowhere to put an error, and a median taken
	// over a partly ordered list is a wrong answer that looks like a right one.
	for i := 1; i < len(sorted); i++ {
		current := sorted[i]
		j := i - 1
		for j >= 0 {
			cmp, err := sorted[j].Compare(current)
			if err != nil {
				return nil, err
			}
			if cmp <= 0 {
				break
			}
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = current
	}
	return sorted, nil
}
