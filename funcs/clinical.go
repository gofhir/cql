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
	years := ref.Year() - bd.Year()
	if ref.YearDay() < bd.YearDay() {
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
