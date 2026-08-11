package funcs

import (
	"fmt"

	fptypes "github.com/gofhir/fhirpath/types"
)

// The calendar precision keywords the temporal operators in this package work in.
// Only the three that appear often enough to be worth naming are here; the coarser
// keywords stay as literals at their handful of use sites.
const (
	precMinute      = "minute"
	precSecond      = "second"
	precMillisecond = "millisecond"
)

// ucumDurationUnits maps the UCUM codes for definite durations of a week or less
// onto the calendar keyword they convert to exactly. CQL treats the two systems as
// interchangeable at this scale, so `5 'd'` shifts a date exactly like `5 days`.
var ucumDurationUnits = map[string]string{
	"wk":  "week",
	"d":   "day",
	"h":   "hour",
	"min": precMinute,
	"s":   precSecond,
	"ms":  precMillisecond,
}

// ucumCalendarUnits are the UCUM durations that have no exact calendar meaning.
// A UCUM year is a fixed 365.25 days and a UCUM month a fixed 30.44, while adding
// a calendar year or month lands on the same date in another year or month. CQL
// keeps the two systems apart above weeks and asks for an explicit conversion to
// cross between them, so `@2020-01-01 + 1 'a'` is refused rather than guessed at.
var ucumCalendarUnits = map[string]string{
	"a":  "year",
	"mo": "month",
}

// calendarDurationUnits maps the calendar keywords that have no exact UCUM
// counterpart onto the UCUM code covering the same nominal span.
var calendarDurationUnits = map[string]string{
	"year":   "a",
	"years":  "a",
	"month":  "mo",
	"months": "mo",
}

// IsCalendarUCUMDurationPair reports whether one unit is a calendar year or month
// and the other the UCUM code naming the same span.
//
// CQL will not call those two equal. A calendar year runs 365 days or 366, while
// UCUM's 'a' is a fixed 365.25, so whether `1 year = 1 'a'` holds depends on which
// year is meant and the answer is unknown rather than false. They stay equivalent,
// because equivalence asks whether the two name the same thing and they do.
func IsCalendarUCUMDurationPair(a, b string) bool {
	if code, ok := calendarDurationUnits[a]; ok && code == b {
		return true
	}
	code, ok := calendarDurationUnits[b]
	return ok && code == a
}

// normalizeDurationUnit maps a duration unit onto the calendar keyword that the
// temporal arithmetic in this package works in.
//
// Calendar keywords pass through unchanged, as does any unit this table does not
// know — those reach fptypes, which reports them rather than silently ignoring them.
// A UCUM year or month is refused here, wrapping the fptypes error so that callers
// can tell the two failures apart with errors.Is.
func normalizeDurationUnit(unit string) (string, error) {
	if calendar, ok := ucumCalendarUnits[unit]; ok {
		return "", fmt.Errorf("%w: %q is a definite-quantity duration, use %s instead",
			fptypes.ErrCalendarConversionRequired, unit, calendar)
	}
	if keyword, ok := ucumDurationUnits[unit]; ok {
		return keyword, nil
	}
	return unit, nil
}
