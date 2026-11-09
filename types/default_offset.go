package types

import (
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

// WithDefaultOffset places dt at offset when it has none and is specified to
// the hour or finer. A DateTime coarser than the hour is returned as it is.
//
// Every place of the engine that gives a DateTime the offset of the evaluation
// request goes through here, and none calls fhirpath's method directly. This
// keeps the engine's comparisons as they were with fhirpath v1.16.6: a DateTime
// coarser than the hour is not placed for comparison. That is the compatibility
// decision of the user of 07-11-2026, recorded in
// docs/plans/018-date-to-datetime.md; it does not claim that CQL 1.5.3 sets the
// cutoff at the hour. Extracting the timezone offset is separate and answers the
// offset of the request at every precision (see the evaluator).
func WithDefaultOffset(dt fptypes.DateTime, offset time.Duration) fptypes.DateTime {
	if dt.Precision() < fptypes.DTHourPrecision {
		return dt
	}
	return dt.WithDefaultOffset(offset)
}
