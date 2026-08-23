package funcs

import (
	"math"
	"sort"

	"github.com/shopspring/decimal"

	fptypes "github.com/gofhir/fhirpath/types"
)

// Flatten takes a collection of collections and flattens into a single collection.
func Flatten(c fptypes.Collection) fptypes.Collection {
	var result fptypes.Collection
	for _, item := range c {
		if item == nil {
			continue
		}
		// Check for listValue wrapper (used internally by funcs package)
		if lw, ok := item.(listValue); ok {
			result = append(result, lw.items...)
			continue
		}
		result = append(result, item)
	}
	return result
}

// Distinct removes duplicate values from a collection (by equality).
// Uses hash-based dedup for O(n) performance instead of O(n²) nested comparison.
func Distinct(c fptypes.Collection) fptypes.Collection {
	if c.Count() <= 1 {
		return c
	}
	seen := make(map[string]struct{}, c.Count())
	result := make(fptypes.Collection, 0, c.Count())
	hasNil := false
	for _, item := range c {
		if item == nil {
			if !hasNil {
				hasNil = true
				result = append(result, item)
			}
			continue
		}
		key := item.Type() + ":" + item.String()
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// Mode returns the most frequently occurring value in a collection.
func Mode(c fptypes.Collection) fptypes.Value {
	if c.Empty() {
		return nil
	}
	// Count occurrences using string representation as key
	counts := make(map[string]int)
	valMap := make(map[string]fptypes.Value)
	for _, item := range c {
		if item == nil {
			continue
		}
		key := item.Type() + ":" + item.String()
		counts[key]++
		valMap[key] = item
	}
	if len(counts) == 0 {
		return nil
	}
	maxKey := ""
	maxCount := 0
	for key, count := range counts {
		if count > maxCount {
			maxCount = count
			maxKey = key
		}
	}
	return valMap[maxKey]
}

// Median returns the median value of a numeric collection.
func Median(c fptypes.Collection) fptypes.Value {
	if c.Empty() {
		return nil
	}
	// Extract numeric values
	nums := make([]decimal.Decimal, 0, c.Count())
	for _, item := range c {
		if item == nil {
			continue
		}
		n := numericVal(item)
		if !n.IsZero() || item.String() == "0" {
			nums = append(nums, n)
		}
	}
	if len(nums) == 0 {
		return nil
	}
	sort.Slice(nums, func(i, j int) bool {
		return nums[i].LessThan(nums[j])
	})
	mid := len(nums) / 2
	if len(nums)%2 == 0 {
		avg := nums[mid-1].Add(nums[mid]).Div(decimal.NewFromInt(2))
		return decimalToValue(avg)
	}
	return decimalToValue(nums[mid])
}

// GeometricMean returns the geometric mean of a collection of positive numbers.
func GeometricMean(c fptypes.Collection) fptypes.Value {
	if c.Empty() {
		return nil
	}
	var values []float64
	for _, item := range c {
		if item == nil {
			continue
		}
		f, _ := numericVal(item).Float64()
		values = append(values, f)
	}
	if len(values) == 0 {
		return nil
	}
	root, ok := GeometricMeanOfFloats(values)
	if !ok {
		return nil
	}
	return decimalToValue(GeometricMeanRound(root))
}

// GeometricMeanOfFloats is the nth root of the product, reporting ok=false where
// the geometric mean is not defined or cannot be computed — a value of zero or
// less, or one that does not fit a float64 at all.
//
// It is exported so that the Quantity aggregate computes the same figures by
// calling it rather than by reimplementing it. Claiming that a quantity and a
// plain number agree in their digits is not the same as their sharing the
// arithmetic that produces those digits, and a reimplementation had already
// drifted: it rounded absolutely to 8 places, which turned a dose of 1e-9 into
// zero.
//
// The product is taken directly where it fits a float64, which is the common case
// and the exact one. Where it does not, the logs are summed instead: multiplying
// 400 values of 100 overflows to +Inf, and +Inf reached decimal.NewFromFloat,
// which panics rather than erroring — a panic with nothing to recover it, out of
// an aggregate a measure could reach with about 110 patients.
//
// Each value is checked as well as the product, which a first version missed: a
// number too large for a float64 arrives here already +Inf, so no amount of care
// with the product saves it.
func GeometricMeanOfFloats(values []float64) (float64, bool) {
	for _, v := range values {
		if v <= 0 || math.IsInf(v, 0) || math.IsNaN(v) {
			return 0, false
		}
	}

	// Normalized by the first value, so the product stays near 1 however large or
	// small the values are: the geometric mean of v0·r1 … v0·rn is v0 times the
	// geometric mean of the ratios. 400 values of 100 have ratios of exactly 1,
	// which is why this returns 100 rather than 99.9999999999984 — the log-sum
	// path below is correct but loses more than a float64's last digit, and no
	// amount of rounding recovers a figure the arithmetic never had.
	base := values[0]
	product := 1.0
	for _, v := range values {
		product *= v / base
	}

	root := 0.0
	switch {
	case !math.IsInf(product, 0) && product != 0:
		root = base * math.Pow(product, 1.0/float64(len(values)))
	default:
		// Ratios that still overflow or underflow: sum their logs instead, which
		// holds a product no float64 can.
		sum := 0.0
		for _, v := range values {
			sum += math.Log(v / base)
		}
		root = base * math.Exp(sum/float64(len(values)))
	}
	if math.IsInf(root, 0) || math.IsNaN(root) || root == 0 {
		return 0, false
	}
	return root, true
}

// GeometricMeanRound trims the noise the float arithmetic leaves in the last
// bits, at fifteen significant digits.
//
// Significant digits rather than decimal places, which is the distinction that
// matters at the small end: rounding to a fixed number of places turned a
// geometric mean of 1e-9 into zero, while leaving the value alone entirely
// reported the cube root of 8 as 1.9999999999999998 — noise from the method, not
// a figure from the data.
//
// Fifteen and not twelve, which is the distinction that matters at the large end.
// A float64 carries about fifteen significant digits, so rounding to twelve threw
// away three that were measured: 1000000000001 came back 1000000000000, which is
// the same "figure nobody measured" this rounding exists to prevent, at the other
// end of the scale. Fifteen discards only what the float cannot vouch for.
//
// Both the numeric and the Quantity path round here, so the two cannot drift
// apart in their last digits.
func GeometricMeanRound(root float64) decimal.Decimal {
	d := decimal.NewFromFloat(root)
	if d.IsZero() {
		return d
	}
	// The exponent of the leading digit, so the rounding follows the magnitude.
	leading := int32(math.Floor(math.Log10(math.Abs(root))))
	rounded := d.Round(significantDigits - 1 - leading)
	// Drop the zeros the rounding leaves, so a clean answer prints as one.
	for rounded.Exponent() < 0 {
		shorter := rounded.Truncate(-rounded.Exponent() - 1)
		if !shorter.Equal(rounded) {
			break
		}
		rounded = shorter
	}
	return rounded
}

// What a float64 can vouch for. Beyond this the digits are the method's, not the
// data's; inside it they are the data's and must not be touched.
const significantDigits = 15

// First returns the first element of a collection.
func First(c fptypes.Collection) fptypes.Value {
	if c.Empty() {
		return nil
	}
	return c[0] //nolint:gosec // bounds checked by Empty()
}

// Last returns the last element of a collection.
func Last(c fptypes.Collection) fptypes.Value {
	if c.Empty() {
		return nil
	}
	return c[c.Count()-1]
}

// SingletonFrom returns the single element if the collection has exactly one item.
func SingletonFrom(c fptypes.Collection) fptypes.Value {
	if c.Count() != 1 {
		return nil
	}
	return c[0]
}

// Exists returns true if the collection has any elements.
func Exists(c fptypes.Collection) fptypes.Value {
	return fptypes.NewBoolean(!c.Empty())
}

// Indexer returns the element at a 0-based index.
func Indexer(c fptypes.Collection, index int) fptypes.Value {
	if index < 0 || index >= c.Count() {
		return nil
	}
	return c[index]
}

// Slice returns the elements between two zero-based bounds, the start included and
// the end excluded. A negative bound counts back from the end, so the last two
// elements are Slice(c, -2, c.Count()).
//
// Bounds are clamped rather than rejected: asking for more list than there is
// yields what there is, and a start at or past the end yields an empty list. That
// is what keeps Slice total — every pair of bounds names some slice of the list.
func Slice(c fptypes.Collection, start, end int) fptypes.Collection {
	start = sliceBound(start, c.Count())
	end = sliceBound(end, c.Count())
	if start >= end {
		return fptypes.Collection{}
	}
	out := make(fptypes.Collection, end-start)
	copy(out, c[start:end])
	return out
}

// sliceBound resolves one Slice bound into an offset within a list of length n.
func sliceBound(i, n int) int {
	if i < 0 {
		i += n
	}
	if i < 0 {
		return 0
	}
	if i > n {
		return n
	}
	return i
}

// Take returns the first n elements.
func Take(c fptypes.Collection, n int) fptypes.Collection {
	if n <= 0 {
		return nil
	}
	if n >= c.Count() {
		return c
	}
	return c[:n]
}

// Skip returns elements after skipping the first n.
func Skip(c fptypes.Collection, n int) fptypes.Collection {
	if n <= 0 {
		return c
	}
	if n >= c.Count() {
		return nil
	}
	return c[n:]
}

// Tail returns all elements except the first.
func Tail(c fptypes.Collection) fptypes.Collection {
	if c.Count() <= 1 {
		return nil
	}
	return c[1:] //nolint:gosec // bounds checked by Count()
}
