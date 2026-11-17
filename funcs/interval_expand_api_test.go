package funcs

import (
	"testing"

	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
)

// The exported expansions of main stay, with the semantics of plan 026
// (docs/plans/026-expand-truncates-to-per.md): removing them would break the
// callers of the module (code review 026, round 2). Expected values from CQL
// 1.5.3, Expand: «expand { Interval[1, 10] } per 2» gives whole intervals of 2.
func TestTheExportedExpansionsRemain(t *testing.T) {
	iv := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(5), true, true)
	per := fptypes.NewInteger(2)
	points, err := IntervalExpandPoints(iv, per)
	if err != nil || len(points) != 2 {
		t.Errorf("IntervalExpandPoints([1, 5], 2) = %v, %v; want {1, 3}", points, err)
	}
	ivs, err := IntervalExpandIntervals(iv, per)
	if err != nil || len(ivs) != 2 {
		t.Errorf("IntervalExpandIntervals([1, 5], 2) = %v, %v; want {[1, 2], [3, 4]}", ivs, err)
	}
	if _, err := IntervalExpandIntervalsUpTo(iv, fptypes.NewInteger(1), 3); err == nil {
		t.Errorf("IntervalExpandIntervalsUpTo([1, 5], 1, 3) = no error; want the limit")
	}
}
