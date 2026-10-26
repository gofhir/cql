package types

import (
	"fmt"

	fptypes "github.com/gofhir/fhirpath/types"
)

// AddQuantities is l + r, or l - r when subtract is set, as CQL has it for
// calendar durations: a year or a month and a week or anything shorter have
// no sum, and the pair answers fptypes.ErrIncompatibleUnits, which every
// caller reads as null (IncompatibleUnits).
//
// The Author's Guide, Quantities: "Durations above days (and weeks) are
// calendar durations that are not comparable with definite quantity UCUM
// duration units"; a month has no fixed number of days. fhirpath, from
// v1.16.6, restates both in the finer unit through UCUM's month of 30 days, as
// FHIRPath 3.0.0 has it, so `1 month + 1 day` was `31 'day'` and
// `@2020-01-31 + (1 month + 1 day)` landed on 2020-03-02. The Java engine
// answers null (testdata/snapshots/units.cql, D41 to D56;
// docs/plans/008-fhirpath-1-16.md). A year and a month have a sum, twelve
// months to the year, and so do the durations from the week down.
func AddQuantities(l, r fptypes.Quantity, subtract bool) (fptypes.Quantity, error) {
	if acrossTheMonth(l.Unit(), r.Unit()) {
		return fptypes.Quantity{}, fmt.Errorf("%s and %s: %w", l, r, fptypes.ErrIncompatibleUnits)
	}
	if subtract {
		return l.Subtract(r)
	}
	return l.Add(r)
}

// yearsAndMonths and belowTheMonth are CQL's calendar duration keywords, on
// either side of the month.
var (
	yearsAndMonths = map[string]bool{"year": true, "years": true, "month": true, "months": true}
	belowTheMonth  = map[string]bool{
		"week": true, "weeks": true, "day": true, "days": true, "hour": true, "hours": true,
		"minute": true, "minutes": true, "second": true, "seconds": true,
		"millisecond": true, "milliseconds": true,
	}
)

func acrossTheMonth(a, b string) bool {
	return yearsAndMonths[a] && belowTheMonth[b] || yearsAndMonths[b] && belowTheMonth[a]
}
