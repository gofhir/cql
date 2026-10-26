package types

import (
	"errors"
	"fmt"

	fptypes "github.com/gofhir/fhirpath/types"
)

// ErrLeapSecond is a DateTime or a Time whose second is 60.
//
// CQL has no place for one. Its formats write "ss - A full two digit second
// value (00..59)", and the range of DateTime ends at
// @9999-12-31T23:59:59.999 and that of Time at @T23:59:59.999 (CQL Reference,
// Appendix B). FHIRPath accepts a leap second, and fhirpath does from v1.16.6;
// the Java engine refuses it, with an error for a literal or a constructor and
// null for a conversion (testdata/snapshots/temporal.cql, D0 to D7).
var ErrLeapSecond = errors.New("a second of 60 is no CQL value: seconds run from 00 to 59")

// NewDateTime parses a DateTime as fptypes does, and refuses the leap second
// CQL does not have.
func NewDateTime(s string) (fptypes.DateTime, error) {
	dt, err := fptypes.NewDateTime(s)
	if err != nil {
		return dt, err
	}
	if dt.Second() > 59 {
		return fptypes.DateTime{}, fmt.Errorf("%s: %w", s, ErrLeapSecond)
	}
	return dt, nil
}

// NewTime parses a Time as fptypes does, and refuses the leap second CQL does
// not have.
func NewTime(s string) (fptypes.Time, error) {
	t, err := fptypes.NewTime(s)
	if err != nil {
		return t, err
	}
	if t.Second() > 59 {
		return fptypes.Time{}, fmt.Errorf("%s: %w", s, ErrLeapSecond)
	}
	return t, nil
}
