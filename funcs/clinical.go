package funcs

import (
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

// AgeInYears calculates the patient's age in years from birthDate to "today".
func AgeInYears(birthDate fptypes.Value) (fptypes.Value, error) {
	return CalculateAgeInYears(birthDate, nil)
}

// AgeInMonths calculates the patient's age in months from birthDate to "today".
func AgeInMonths(birthDate fptypes.Value) (fptypes.Value, error) {
	return CalculateAgeInMonths(birthDate, nil)
}

// AgeInWeeks calculates the patient's age in weeks from birthDate to "today".
func AgeInWeeks(birthDate fptypes.Value) (fptypes.Value, error) {
	return CalculateAgeInWeeks(birthDate, nil)
}

// AgeInDays calculates the patient's age in days from birthDate to "today".
func AgeInDays(birthDate fptypes.Value) (fptypes.Value, error) {
	return CalculateAgeInDays(birthDate, nil)
}

// CalculateAgeInYears calculates years between birthDate and asOf (or today if nil).
func CalculateAgeInYears(birthDate, asOf fptypes.Value) (fptypes.Value, error) {
	bd, err := toTime(birthDate)
	if err != nil || bd.IsZero() {
		return nil, nil
	}
	ref := referenceDate(asOf)
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
	days := int(ref.Sub(bd).Hours() / 24)
	return fptypes.NewInteger(int64(days)), nil
}

// AgeInYearsAt calculates the patient's age in years at a given date.
func AgeInYearsAt(birthDate, asOf fptypes.Value) (fptypes.Value, error) {
	return CalculateAgeInYears(birthDate, asOf)
}

// AgeInMonthsAt calculates the patient's age in months at a given date.
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
	return time.Now().UTC()
}

func toTime(v fptypes.Value) (time.Time, error) {
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
	} {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, nil
}
