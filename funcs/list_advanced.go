package funcs

import (
	"math"
	"sort"

	fptypes "github.com/gofhir/fhirpath/types"
	"github.com/shopspring/decimal"
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
	product := 1.0
	count := 0
	for _, item := range c {
		if item == nil {
			continue
		}
		n := numericVal(item)
		f, _ := n.Float64()
		if f <= 0 {
			return nil // geometric mean undefined for non-positive values
		}
		product *= f
		count++
	}
	if count == 0 {
		return nil
	}
	result := math.Pow(product, 1.0/float64(count))
	return fptypes.NewDecimalFromFloat(result)
}

// First returns the first element of a collection.
func First(c fptypes.Collection) fptypes.Value {
	if c.Empty() {
		return nil
	}
	return c[0]
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
	return c[1:]
}
