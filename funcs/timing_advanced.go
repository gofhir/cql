package funcs

import (
	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

// overlapsCompare is a shared helper for OverlapsBefore and OverlapsAfter.
// It checks overlap, then compares the selected bounds of a and b using the
// given comparator (cmp < 0 for "before", cmp > 0 for "after").
// boundA/boundB select which bound to compare; closedA/closedB select which
// closed flag to use; intAdj adjusts integer effective bounds (+1 for low, -1 for high).
func overlapsCompare(
	a, b cqltypes.Interval,
	boundA, boundB fptypes.Value,
	closedA, closedB bool,
	intAdj int64,
	cmpFunc func(int) bool,
) (fptypes.Value, error) {
	overlap, err := a.Overlaps(b)
	if err != nil {
		if isAmbiguousComparisonErr(err) {
			return nil, nil
		}
		return nil, err
	}
	if !overlap {
		return fptypes.NewBoolean(false), nil
	}
	if boundA == nil || boundB == nil {
		return nil, nil
	}
	// For integer types, compare effective bounds (accounting for open boundaries)
	if ai, ok := boundA.(fptypes.Integer); ok {
		if bi, ok := boundB.(fptypes.Integer); ok {
			aEff := ai.Value()
			if !closedA {
				aEff += intAdj
			}
			bEff := bi.Value()
			if !closedB {
				bEff += intAdj
			}
			return fptypes.NewBoolean(cmpFunc(int(aEff - bEff))), nil
		}
	}
	cmp, err := compareVals(boundA, boundB)
	if err != nil {
		if isAmbiguousComparisonErr(err) {
			return nil, nil
		}
		return nil, err
	}
	if cmp == 0 {
		return fptypes.NewBoolean(closedA && !closedB), nil
	}
	return fptypes.NewBoolean(cmpFunc(cmp)), nil
}

// OverlapsBefore checks if interval a starts before b and overlaps.
func OverlapsBefore(a, b cqltypes.Interval) (fptypes.Value, error) {
	return overlapsCompare(a, b, a.Low, b.Low, a.LowClosed, b.LowClosed, 1, func(d int) bool { return d < 0 })
}

// OverlapsAfter checks if interval a extends past the end of b and overlaps.
func OverlapsAfter(a, b cqltypes.Interval) (fptypes.Value, error) {
	return overlapsCompare(a, b, a.High, b.High, a.HighClosed, b.HighClosed, -1, func(d int) bool { return d > 0 })
}

// SameAs checks if two intervals are the same (equal boundaries).
func SameAs(a, b cqltypes.Interval) fptypes.Value {
	return fptypes.NewBoolean(a.Equal(b))
}

// SameOrBefore checks if interval a ends on or before interval b starts.
func SameOrBefore(a, b cqltypes.Interval) (fptypes.Value, error) {
	if a.High == nil || b.Low == nil {
		return nil, nil
	}
	cmp, err := compareVals(a.High, b.Low)
	if err != nil {
		if isAmbiguousComparisonErr(err) {
			return nil, nil
		}
		return nil, err
	}
	return fptypes.NewBoolean(cmp <= 0), nil
}

// SameOrAfter checks if interval a starts on or after interval b ends.
func SameOrAfter(a, b cqltypes.Interval) (fptypes.Value, error) {
	if a.Low == nil || b.High == nil {
		return nil, nil
	}
	cmp, err := compareVals(a.Low, b.High)
	if err != nil {
		if isAmbiguousComparisonErr(err) {
			return nil, nil
		}
		return nil, err
	}
	return fptypes.NewBoolean(cmp >= 0), nil
}

// Starts checks if interval a starts interval b (a starts at same point as b, a ends within b).
func Starts(a, b cqltypes.Interval) (fptypes.Value, error) {
	if a.Low == nil || b.Low == nil {
		return nil, nil
	}
	cmpLow, err := compareVals(a.Low, b.Low)
	if err != nil {
		if isAmbiguousComparisonErr(err) {
			return nil, nil
		}
		return nil, err
	}
	if cmpLow != 0 {
		return fptypes.NewBoolean(false), nil
	}
	if a.High != nil && b.High != nil {
		cmpHigh, err := compareVals(a.High, b.High)
		if err != nil {
			if isAmbiguousComparisonErr(err) {
				return nil, nil
			}
			return nil, err
		}
		return fptypes.NewBoolean(cmpHigh <= 0), nil
	}
	return fptypes.NewBoolean(true), nil
}

// Ends checks if interval a ends interval b (a ends at same point as b, a starts within b).
func Ends(a, b cqltypes.Interval) (fptypes.Value, error) {
	if a.High == nil || b.High == nil {
		return nil, nil
	}
	cmpHigh, err := compareVals(a.High, b.High)
	if err != nil {
		if isAmbiguousComparisonErr(err) {
			return nil, nil
		}
		return nil, err
	}
	if cmpHigh != 0 {
		return fptypes.NewBoolean(false), nil
	}
	if a.Low != nil && b.Low != nil {
		cmpLow, err := compareVals(a.Low, b.Low)
		if err != nil {
			if isAmbiguousComparisonErr(err) {
				return nil, nil
			}
			return nil, err
		}
		return fptypes.NewBoolean(cmpLow >= 0), nil
	}
	return fptypes.NewBoolean(true), nil
}

// During checks if interval a is during interval b (same as IncludedIn).
func During(a, b cqltypes.Interval) (fptypes.Value, error) {
	return IntervalIncludedIn(a, b)
}

// ConcurrentWith checks if two intervals have the same boundaries (alias for SameAs).
func ConcurrentWith(a, b cqltypes.Interval) fptypes.Value {
	return SameAs(a, b)
}
