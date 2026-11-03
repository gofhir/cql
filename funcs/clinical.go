package funcs

import (
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

// CalculateAgeIn is the age of someone born at birthDate, as of asOf, in the unit
// ("years", "months", "weeks", "days", "hours", "minutes" or "seconds"). CQL
// 1.5.3, Appendix B, CalculateAgeAt: "The CalculateAgeAt operators are defined in
// terms of a datetime duration calculation", so the age is `<unit> between
// birthDate and asOf`: whole calendar periods (a person born on 29 February is a
// year older on 28 February of a common year, plan 015), and an uncertainty where
// either date is too imprecise to say (plan 016). With no reference to measure
// against there is no age, which CQL answers with null rather than an age
// against the machine's clock.
//
// The Date overloads cover years to days and the DateTime ones add hours, minutes
// and seconds. A Date beside a DateTime is converted with ToDateTime, as is a
// Date in a unit finer than a day, which only the DateTime overload has: "the
// Date values will be implicitly converted to DateTime" (Appendix B, Duration).
// The conversion is of the date as written, to the day, so its time of day is
// every moment of it and an age in hours is an uncertainty.
//
// evaluation is the offset of the evaluation request, which a DateTime that
// states none is read at, nil for UTC. See DurationBetweenAt.
func CalculateAgeIn(unit string, birthDate, asOf fptypes.Value, evaluation *time.Location) (fptypes.Value, error) {
	if birthDate == nil || asOf == nil {
		return nil, nil
	}
	birthDate, asOf, err := convertAgeDates(unit, birthDate, asOf)
	if err != nil {
		return nil, err
	}
	return DurationBetweenAt(birthDate, asOf, unit, evaluation)
}

// convertAgeDates converts the Date among the two to a DateTime where the age
// calls for one: the other is a DateTime, or the unit is finer than a day.
func convertAgeDates(unit string, birthDate, asOf fptypes.Value) (birth, reference fptypes.Value, err error) {
	_, birthIsDate := birthDate.(fptypes.Date)
	_, asOfIsDate := asOf.(fptypes.Date)
	if !birthIsDate && !asOfIsDate {
		return birthDate, asOf, nil
	}
	if birthIsDate && asOfIsDate && !AgeNeedsDateTime(unit) {
		return birthDate, asOf, nil
	}
	birth, err = ToDateTime(birthDate)
	if err != nil {
		return nil, nil, err
	}
	reference, err = ToDateTime(asOf)
	if err != nil {
		return nil, nil, err
	}
	return birth, reference, nil
}

// AgeNeedsDateTime reports whether an age in the unit is only defined for
// DateTime: CQL has no Date overload of CalculateAgeIn… for hours, minutes and
// seconds.
func AgeNeedsDateTime(unit string) bool {
	switch unit {
	case "hours", "minutes", "seconds":
		return true
	}
	return false
}
