package funcs

import (
	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

// OverlapsBefore checks if interval a starts before b and overlaps.
func OverlapsBefore(a, b cqltypes.Interval) (fptypes.Value, error) {
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
	if a.Low == nil || b.Low == nil {
		return nil, nil
	}
	cmp, err := compareVals(a.Low, b.Low)
	if err != nil {
		if isAmbiguousComparisonErr(err) {
			return nil, nil
		}
		return nil, err
	}
	return fptypes.NewBoolean(cmp < 0), nil
}

// OverlapsAfter checks if interval a extends past the end of b and overlaps.
func OverlapsAfter(a, b cqltypes.Interval) (fptypes.Value, error) {
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
	if a.High == nil || b.High == nil {
		return nil, nil
	}
	cmp, err := compareVals(a.High, b.High)
	if err != nil {
		if isAmbiguousComparisonErr(err) {
			return nil, nil
		}
		return nil, err
	}
	return fptypes.NewBoolean(cmp > 0), nil
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
