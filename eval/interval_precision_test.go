package eval

import (
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

// TestWidenForPrecision covers the range a provider gets from a filter at a
// precision: each bound out by one unit and two days, a bound that would leave
// the calendar at the calendar's edge, and anything without a precision this
// engine knows left as it was.
func TestWidenForPrecision(t *testing.T) {
	dt := func(s string) fptypes.Value {
		v, err := fptypes.NewDateTime(s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	d := func(s string) fptypes.Value {
		v, err := fptypes.NewDate(s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	for _, tt := range []struct {
		name      string
		in        cqltypes.Interval
		precision string
		low, high string
	}{
		{"a day", cqltypes.NewInterval(dt("2020-03-10T00:00:00Z"), dt("2020-03-20T00:00:00Z"), true, true), "day",
			"2020-03-07T00:00:00Z", "2020-03-23T00:00:00Z"},
		{"a month", cqltypes.NewInterval(dt("2020-03-10T00:00:00Z"), dt("2020-03-20T00:00:00Z"), true, true), "month",
			"2020-02-08T00:00:00Z", "2020-04-22T00:00:00Z"},
		{"a Date at the hour", cqltypes.NewInterval(d("2020-03-10"), d("2020-03-20"), true, true), "hour",
			"2020-03-08", "2020-03-22"},
		{"past the start of the calendar", cqltypes.NewInterval(dt("0001-01-02T00:00:00Z"), dt("2020-03-20T00:00:00Z"), true, true), "day",
			"0001-01-01T00:00:00.000Z", "2020-03-23T00:00:00Z"},
		{"a bound written to the month", cqltypes.NewInterval(dt("2020-03"), dt("2020-04"), true, true), "day",
			"2020-01", "2020-06"},
		{"a bound written to the year", cqltypes.NewInterval(dt("2019"), dt("2020"), true, true), "month",
			"2017", "2022"},
		// The phrase's unit is the coarser one here, and moving by two months
		// would leave out February at the year.
		{"a bound written to the month, at the year", cqltypes.NewInterval(dt("2020-06"), dt("2020-12"), true, true), "year",
			"2018-06", "2022-12"},
		{"no precision", cqltypes.NewInterval(dt("2020-03-10T00:00:00Z"), dt("2020-03-20T00:00:00Z"), true, true), "",
			"2020-03-10T00:00:00Z", "2020-03-20T00:00:00Z"},
		{"a week, which is not one", cqltypes.NewInterval(dt("2020-03-10T00:00:00Z"), dt("2020-03-20T00:00:00Z"), true, true), "week",
			"2020-03-10T00:00:00Z", "2020-03-20T00:00:00Z"},
	} {
		got, ok := widenForPrecision(tt.in, tt.precision).(cqltypes.Interval)
		if !ok {
			t.Fatalf("%s: not an interval", tt.name)
		}
		if got.Low == nil || got.High == nil || got.Low.String() != tt.low || got.High.String() != tt.high {
			t.Errorf("%s: widened to %v, want [%s, %s]", tt.name, got, tt.low, tt.high)
		}
	}
	// A missing bound stays missing.
	got := widenForPrecision(cqltypes.NewInterval(nil, dt("2020-03-20T00:00:00Z"), false, true), "day").(cqltypes.Interval)
	if got.Low != nil {
		t.Errorf("an open start widened to %v, want it left open", got.Low)
	}
}
