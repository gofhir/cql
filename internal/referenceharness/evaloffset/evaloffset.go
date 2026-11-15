// Package evaloffset is the offset of the evaluation request each area of
// testdata/snapshots is evaluated at, by the two reference engines and by this
// one (plan 024, docs/plans/024-same-precision-offset.md).
//
// Every area is evaluated at 2026-01-01T00:00:00-03:00 but the ones listed
// here: an offset of whole hours cuts the hours of the day where UTC does, so
// only an area at a fractional offset can tell a comparison that reads the
// hour in the offset of the evaluation from one that reads it at UTC.
package evaloffset

import "fmt"

// Default is the offset of every area not in fractional, in seconds.
const Default = -3 * 3600

// fractional is the offset of the areas that do not use the default, in seconds.
var fractional = map[string]int{
	"fractionaloffset": 5*3600 + 30*60, // +05:30
}

// Seconds is the offset an area is evaluated at, east of UTC.
func Seconds(area string) int {
	if s, ok := fractional[area]; ok {
		return s
	}
	return Default
}

// Text is the offset an area is evaluated at as the references read it from
// CQL_EVALUATION_OFFSET: "-03:00", "+05:30".
func Text(area string) string {
	s := Seconds(area)
	sign := '+'
	if s < 0 {
		sign, s = '-', -s
	}
	return fmt.Sprintf("%c%02d:%02d", sign, s/3600, s%3600/60)
}

// Timestamp is the evaluation instant of an area, as the references write it
// to evaluationDateTime for the Java engine.
func Timestamp(area string) string {
	return "2026-01-01T00:00:00" + Text(area)
}
