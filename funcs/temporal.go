package funcs

import (
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

// Now returns the current date and time as a DateTime value.
func Now() (fptypes.Value, error) {
	return fptypes.NewDateTime(time.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00"))
}

// Today returns the current date as a Date value.
func Today() (fptypes.Value, error) {
	return fptypes.NewDate(time.Now().UTC().Format("2006-01-02"))
}

// TimeOfDay returns the current time as a Time value.
func TimeOfDay() (fptypes.Value, error) {
	return fptypes.NewTime(time.Now().UTC().Format("15:04:05.000"))
}

// YearsBetween calculates the number of whole years between two dates.
func YearsBetween(low, high fptypes.Value) (fptypes.Value, error) {
	tl, err := toTime(low)
	if err != nil || tl.IsZero() {
		return nil, nil
	}
	th, err := toTime(high)
	if err != nil || th.IsZero() {
		return nil, nil
	}
	years := th.Year() - tl.Year()
	if th.YearDay() < tl.YearDay() {
		years--
	}
	return fptypes.NewInteger(int64(years)), nil
}

// MonthsBetween calculates the number of whole months between two dates.
func MonthsBetween(low, high fptypes.Value) (fptypes.Value, error) {
	tl, err := toTime(low)
	if err != nil || tl.IsZero() {
		return nil, nil
	}
	th, err := toTime(high)
	if err != nil || th.IsZero() {
		return nil, nil
	}
	months := (th.Year()-tl.Year())*12 + int(th.Month()) - int(tl.Month())
	if th.Day() < tl.Day() {
		months--
	}
	return fptypes.NewInteger(int64(months)), nil
}

// DaysBetween calculates the number of whole days between two dates.
func DaysBetween(low, high fptypes.Value) (fptypes.Value, error) {
	tl, err := toTime(low)
	if err != nil || tl.IsZero() {
		return nil, nil
	}
	th, err := toTime(high)
	if err != nil || th.IsZero() {
		return nil, nil
	}
	days := int(th.Sub(tl).Hours() / 24)
	return fptypes.NewInteger(int64(days)), nil
}

// HoursBetween calculates the number of whole hours between two datetimes.
func HoursBetween(low, high fptypes.Value) (fptypes.Value, error) {
	tl, err := toTime(low)
	if err != nil || tl.IsZero() {
		return nil, nil
	}
	th, err := toTime(high)
	if err != nil || th.IsZero() {
		return nil, nil
	}
	hours := int(th.Sub(tl).Hours())
	return fptypes.NewInteger(int64(hours)), nil
}

// MinutesBetween calculates the number of whole minutes between two datetimes.
func MinutesBetween(low, high fptypes.Value) (fptypes.Value, error) {
	tl, err := toTime(low)
	if err != nil || tl.IsZero() {
		return nil, nil
	}
	th, err := toTime(high)
	if err != nil || th.IsZero() {
		return nil, nil
	}
	minutes := int(th.Sub(tl).Minutes())
	return fptypes.NewInteger(int64(minutes)), nil
}

// SecondsBetween calculates the number of whole seconds between two datetimes.
func SecondsBetween(low, high fptypes.Value) (fptypes.Value, error) {
	tl, err := toTime(low)
	if err != nil || tl.IsZero() {
		return nil, nil
	}
	th, err := toTime(high)
	if err != nil || th.IsZero() {
		return nil, nil
	}
	seconds := int(th.Sub(tl).Seconds())
	return fptypes.NewInteger(int64(seconds)), nil
}
