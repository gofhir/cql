package types

import (
	"errors"
	"math"
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"
)

// TestAStepOffTheTypeIsErrPastRange covers the one error every type gives for
// a step past its range, so that the operator can answer null for all of them
// (CQL 1.5.3, Appendix B, Successor and Predecessor; plan 013) and an interval
// boundary can read "no step" from the same error. Date and DateTime gave
// fhirpath's ErrInvalidTemporal, and Time an error of its own.
func TestAStepOffTheTypeIsErrPastRange(t *testing.T) {
	date := func(s string) fptypes.Value { v, _ := fptypes.NewDate(s); return v }
	dateTime := func(s string) fptypes.Value { v, _ := fptypes.NewDateTime(s); return v }
	clock := func(s string) fptypes.Value { v, _ := fptypes.NewTime(s); return v }
	for _, tt := range []struct {
		name string
		v    fptypes.Value
		step func(fptypes.Value) (fptypes.Value, error)
	}{
		{"largest Long", fptypes.NewInteger(math.MaxInt64), Successor},
		{"least Long", fptypes.NewInteger(math.MinInt64), Predecessor},
		{"last day", date("9999-12-31"), Successor},
		{"last month", date("9999-12"), Successor},
		{"first year", date("0001"), Predecessor},
		{"last millisecond", dateTime("9999-12-31T23:59:59.999"), Successor},
		{"last millisecond at +14:00", dateTime("9999-12-31T23:59:59.999+14:00"), Successor},
		{"first millisecond", dateTime("0001-01-01T00:00:00.000"), Predecessor},
		{"first year of a DateTime", dateTime("0001"), Predecessor},
		{"last time", clock("23:59:59.999"), Successor},
		{"last hour", clock("23"), Successor},
		{"first time", clock("00:00:00.000"), Predecessor},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.v == nil {
				t.Fatal("the operand did not parse")
			}
			v, err := tt.step(tt.v)
			if !errors.Is(err, ErrPastRange) {
				t.Errorf("= %v, %v; want ErrPastRange", v, err)
			}
		})
	}
}

// TestAStepInsideTheTypeIsAValue holds the other side of each edge.
func TestAStepInsideTheTypeIsAValue(t *testing.T) {
	date, _ := fptypes.NewDate("9999-12-30")
	clock, _ := fptypes.NewTime("23:59:59.998")
	for _, tt := range []struct {
		v    fptypes.Value
		want string
	}{
		{fptypes.NewInteger(math.MaxInt64 - 1), "9223372036854775807"},
		{date, "9999-12-31"},
		{clock, "23:59:59.999"},
	} {
		v, err := Successor(tt.v)
		if err != nil || v == nil || v.String() != tt.want {
			t.Errorf("successor of %v = %v, %v; want %s", tt.v, v, err, tt.want)
		}
	}
}
