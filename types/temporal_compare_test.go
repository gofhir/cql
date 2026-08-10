package types

import (
	"errors"
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"
)

func mustTime(t *testing.T, s string) fptypes.Value {
	t.Helper()
	v, err := fptypes.NewTime(s)
	if err != nil {
		t.Fatalf("NewTime(%q): %v", s, err)
	}
	return v
}

func mustDate(t *testing.T, s string) fptypes.Value {
	t.Helper()
	v, err := fptypes.NewDate(s)
	if err != nil {
		t.Fatalf("NewDate(%q): %v", s, err)
	}
	return v
}

func mustDateTime(t *testing.T, s string) fptypes.Value {
	t.Helper()
	v, err := fptypes.NewDateTime(s)
	if err != nil {
		t.Fatalf("NewDateTime(%q): %v", s, err)
	}
	return v
}

// TestCompareTemporal_MillisecondIsItsOwnPrecision covers the one rule CQL does not
// share with FHIRPath. fptypes compares seconds and milliseconds as a single decimal
// precision, which makes @T15:59:59 sort below @T15:59:59.999; CQL treats them as
// different precisions, so the comparison is unknown and the operator yields null.
func TestCompareTemporal_MillisecondIsItsOwnPrecision(t *testing.T) {
	cases := []struct {
		name string
		a, b fptypes.Value
	}{
		{"time, second against millisecond", mustTime(t, "15:59:59"), mustTime(t, "15:59:59.999")},
		{"time, the other way round", mustTime(t, "15:59:59.999"), mustTime(t, "15:59:59")},
		{"datetime, second against millisecond",
			mustDateTime(t, "2017-09-01T00:00:00"), mustDateTime(t, "2017-09-01T00:00:00.000")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CompareTemporal(tc.a, tc.b)
			if !errors.Is(err, fptypes.ErrPrecisionMismatch) {
				t.Errorf("CompareTemporal(%v, %v) gave %v, want ErrPrecisionMismatch", tc.a, tc.b, err)
			}
		})
	}
}

// TestCompareTemporal_DecidedAtSharedComponent is the other half of the rule: a
// difference in a component both values specify settles the comparison, and the
// finer precision of one of them is then beside the point.
func TestCompareTemporal_DecidedAtSharedComponent(t *testing.T) {
	cases := []struct {
		name string
		a, b fptypes.Value
		want int
	}{
		{"different hour", mustTime(t, "15:59:59"), mustTime(t, "20:59:59.999"), -1},
		{"different hour, reversed", mustTime(t, "20:59:59"), mustTime(t, "15:59:59.999"), 1},
		{"same precision, equal",
			mustDateTime(t, "2017-09-01T00:00:00"), mustDateTime(t, "2017-09-01T00:00:00"), 0},
		{"same precision, ordered",
			mustDateTime(t, "2017-09-01T00:00:00"), mustDateTime(t, "2017-09-01T00:00:01"), -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CompareTemporal(tc.a, tc.b)
			if err != nil {
				t.Fatalf("CompareTemporal(%v, %v): %v", tc.a, tc.b, err)
			}
			if got != tc.want {
				t.Errorf("CompareTemporal(%v, %v) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

// TestCompareTemporal_AgreesWithFptypesOnOffsets pins the normalization rule against
// the library it wraps. Reading one value at UTC while the other stays in local
// components would compare two frames at once, and would report a comparison fptypes
// settles as unknown — the first case below is exactly that, and it regressed once.
func TestCompareTemporal_AgreesWithFptypesOnOffsets(t *testing.T) {
	cases := []struct {
		name string
		a, b fptypes.Value
	}{
		{"offset against a bare date, crossing the day at UTC",
			mustDateTime(t, "2020-06-01T02:00:00+05:00"), mustDate(t, "2020-05-31")},
		{"different offsets, different precision",
			mustDateTime(t, "2020-06-01T10:00:00+05:00"), mustDateTime(t, "2020-06-01T10:00+02:00")},
		{"different offsets naming the same instant",
			mustDateTime(t, "2020-06-01T10:00:00+05:00"), mustDateTime(t, "2020-06-01T07:00+02:00")},
		{"same offset, different precision",
			mustDateTime(t, "2020-06-01T10:00:00+05:00"), mustDateTime(t, "2020-06-01T10:00+05:00")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want, wantErr := tc.a.(fptypes.Comparable).Compare(tc.b)
			got, gotErr := CompareTemporal(tc.a, tc.b)

			// Only the second/millisecond boundary may diverge, and none of these reach it
			if (wantErr != nil) != (gotErr != nil) {
				t.Fatalf("CompareTemporal gave err=%v, fptypes gave err=%v", gotErr, wantErr)
			}
			if wantErr == nil && got != want {
				t.Errorf("CompareTemporal = %d, fptypes = %d", got, want)
			}
		})
	}
}

// TestCompareTemporal_NonTemporalUnaffected makes sure wrapping the comparison did
// not change how ordinary values order.
func TestCompareTemporal_NonTemporalUnaffected(t *testing.T) {
	got, err := CompareTemporal(fptypes.NewInteger(1), fptypes.NewInteger(2))
	if err != nil {
		t.Fatalf("CompareTemporal on integers: %v", err)
	}
	if got != -1 {
		t.Errorf("CompareTemporal(1, 2) = %d, want -1", got)
	}
}
