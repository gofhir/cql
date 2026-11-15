package eval

import (
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

// A DateTime that writes no offset and has not been told one is somewhere in the
// 26 hours the legal offsets allow, and a comparison with a value that writes one
// holds where it holds at both ends of that window. Each end is read in the
// offset of the evaluation request (plan 024, docs/plans/024-same-precision-offset.md).
//
// At +05:30, 2020-06-15T23:00:00 is at latest 11:00 on the 16th at Z, 16:30
// there; and 11:40Z is 17:10. The hours are 16 and 17, so it is before the hour
// of the second. At UTC the latest end is in the hour 11, the second's own, and
// the answer would be unknown.
func TestTheWindowOfAnUnplacedDateTimeIsReadInTheEvaluationOffset(t *testing.T) {
	unplaced, err := fptypes.NewDateTime("2020-06-15T23:00:00")
	if err != nil {
		t.Fatal(err)
	}
	if _, placed := unplaced.EffectiveOffset(); placed {
		t.Fatal("the value has an offset, and there is no window to read")
	}
	atZ, err := fptypes.NewDateTime("2020-06-16T11:40:00Z")
	if err != nil {
		t.Fatal(err)
	}
	frame := time.FixedZone("", 5*3600+30*60)
	got, err := temporalBefore(frame, unplaced, atZ, "hour")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.String() != "true" {
		t.Errorf("before hour of at +05:30 = %v; want true", got)
	}
	got, err = temporalBefore(time.UTC, unplaced, atZ, "hour")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("before hour of at Z = %v; want null: the latest end is in the hour of the other", got)
	}
}

// An unplaced DateTime next to one that writes an offset is not read in the
// evaluation's offset by the readers of components: they keep the reading
// they had before plan 024 (see readingFrameOf).
func TestAnUnplacedDateTimeIsAtTheOffsetOfTheEvaluation(t *testing.T) {
	unplaced, err := fptypes.NewDateTime("2020-01-01T00:10:00")
	if err != nil {
		t.Fatal(err)
	}
	atZ, err := fptypes.NewDateTime("2020-01-01T00:40:00Z")
	if err != nil {
		t.Fatal(err)
	}
	frame := time.FixedZone("", 5*3600+30*60)
	// Read as before plan 024: the written value at UTC, the unplaced one in
	// its digits. Two frames, a known defect of the readers that do not resolve
	// the window (plan 024, code review round 3, M3; backlog).
	a, _, b, _ := componentsAtPrecision(unplaced, atZ, "hour", frame)
	if a[3] != 0 || b[3] != 0 {
		t.Errorf("the hours of the unplaced DateTime and of 00:40Z are %d and %d; want 0 and 0, as before plan 024", a[3], b[3])
	}
	a, _, b, _ = componentsAtPrecision(unplaced, unplaced, "hour", frame)
	if a[3] != 0 || b[3] != 0 {
		t.Errorf("two unplaced DateTimes are read as written; got hours %d and %d", a[3], b[3])
	}
}
