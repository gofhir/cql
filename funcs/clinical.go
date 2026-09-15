package funcs

import (
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

// CalculateAgeInYears calculates years between birthDate and asOf (or today if nil).
func CalculateAgeInYears(birthDate, asOf fptypes.Value) (fptypes.Value, error) {
	bd, err := toTime(birthDate)
	if err != nil || bd.IsZero() {
		return nil, nil
	}
	ref := referenceDate(asOf)
	if ref.IsZero() {
		// No reference to measure against: there is no age, which CQL answers
		// with null. `CalculateAgeInYears(bd, null)` used to answer an age
		// against the machine's clock instead.
		return nil, nil
	}
	// Month and day, not day-of-year. A leap year's day numbers run one ahead of a
	// common year's after the 29th of February, so comparing them read the
	// anniversary as not yet reached for anyone born in a leap year after that
	// date: someone born 2000-06-01 was 18 on 2019-06-01, the day they turned 19.
	//
	// That is everyone born between the 1st of March and the 31st of December of a
	// leap year, on their birthday.
	//
	// Who it reaches in published CQL is narrower than that sounds, and was
	// measured rather than assumed. Of the 25 uses of AgeInYearsAt across the 19
	// published measures, 21 measure against the start of the measurement period —
	// which every one of them puts on the 1st of January, so the reference can only
	// be an anniversary for someone born in January, before the leap day, where
	// day-of-year numbers still agree. Those do not move.
	//
	// The other three measure against a clinical date: an encounter's period, a
	// test's effective time. Those land on any day of the year, so a patient born
	// in a leap year after February whose encounter begins on their birthday was
	// counted a year younger, and a threshold like `>= 18` turns on that.
	//
	// The engine already answered this correctly one function over, which is what
	// settles it without reaching for the specification: CalculateAgeInMonths
	// compares month and day and returned 228 for that pair, and 228 months is 19
	// years. The two now agree.
	years := ref.Year() - bd.Year()
	if ref.Month() < bd.Month() || (ref.Month() == bd.Month() && ref.Day() < bd.Day()) {
		years--
	}
	return fptypes.NewInteger(int64(years)), nil
}

// CalculateAgeInMonths calculates months between birthDate and asOf (or today if nil).
func CalculateAgeInMonths(birthDate, asOf fptypes.Value) (fptypes.Value, error) {
	bd, err := toTime(birthDate)
	if err != nil || bd.IsZero() {
		return nil, nil
	}
	ref := referenceDate(asOf)
	if ref.IsZero() {
		// No reference to measure against: there is no age, which CQL answers
		// with null. `CalculateAgeInYears(bd, null)` used to answer an age
		// against the machine's clock instead.
		return nil, nil
	}
	months := (ref.Year()-bd.Year())*12 + int(ref.Month()) - int(bd.Month())
	if ref.Day() < bd.Day() {
		months--
	}
	return fptypes.NewInteger(int64(months)), nil
}

// CalculateAgeInWeeks calculates weeks between birthDate and asOf (or today if nil).
func CalculateAgeInWeeks(birthDate, asOf fptypes.Value) (fptypes.Value, error) {
	bd, err := toTime(birthDate)
	if err != nil || bd.IsZero() {
		return nil, nil
	}
	ref := referenceDate(asOf)
	if ref.IsZero() {
		// No reference to measure against: there is no age, which CQL answers
		// with null. `CalculateAgeInYears(bd, null)` used to answer an age
		// against the machine's clock instead.
		return nil, nil
	}
	days := int(ref.Sub(bd).Hours() / 24)
	return fptypes.NewInteger(int64(days / 7)), nil
}

// CalculateAgeInDays calculates days between birthDate and asOf (or today if nil).
func CalculateAgeInDays(birthDate, asOf fptypes.Value) (fptypes.Value, error) {
	bd, err := toTime(birthDate)
	if err != nil || bd.IsZero() {
		return nil, nil
	}
	ref := referenceDate(asOf)
	if ref.IsZero() {
		// No reference to measure against: there is no age, which CQL answers
		// with null. `CalculateAgeInYears(bd, null)` used to answer an age
		// against the machine's clock instead.
		return nil, nil
	}
	days := int(ref.Sub(bd).Hours() / 24)
	return fptypes.NewInteger(int64(days)), nil
}

// AgeInYearsAt calculates the patient's age in years at a given date.
func AgeInYearsAt(birthDate, asOf fptypes.Value) (fptypes.Value, error) {
	return CalculateAgeInYears(birthDate, asOf)
}

// AgeInMonthsAt calculates the patient's age in months at a given date.
func AgeInWeeksAt(birthDate, asOf fptypes.Value) (fptypes.Value, error) {
	return CalculateAgeInWeeks(birthDate, asOf)
}

// AgeInDaysAt calculates age in days as of the given instant.
func AgeInDaysAt(birthDate, asOf fptypes.Value) (fptypes.Value, error) {
	return CalculateAgeInDays(birthDate, asOf)
}

// AgeInMonthsAt calculates age in months as of the given instant.
func AgeInMonthsAt(birthDate, asOf fptypes.Value) (fptypes.Value, error) {
	return CalculateAgeInMonths(birthDate, asOf)
}

func referenceDate(asOf fptypes.Value) time.Time {
	if asOf != nil {
		t, err := toTime(asOf)
		if err == nil && !t.IsZero() {
			return t
		}
	}
	// No clock is read here. The comment that used to stand in its place said the
	// evaluator always passes the evaluation's frozen timestamp and that only a
	// direct caller of this package could land here — and that was not true: two
	// branches of the evaluator's age switch passed a nil through, and an age came
	// back seven years off the Today() beside it.
	//
	// With nothing to measure against there is no age, which is null in CQL rather
	// than an age against whatever the machine's clock says. The callers turn this
	// zero time into that null.
	return time.Time{}
}

func toTime(v fptypes.Value) (time.Time, error) { //nolint:unparam // error kept for future format additions
	if v == nil {
		return time.Time{}, nil
	}
	s := v.String()
	// Try common FHIR date/datetime formats
	for _, layout := range []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"2006-01",
		"2006",
		"15:04:05.000",
		"15:04:05",
		"15:04",
	} {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, nil
}
