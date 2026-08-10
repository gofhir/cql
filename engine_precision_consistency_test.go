package cql

import (
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"
)

// evalOrFail evaluates a CQL expression and fails the test if it errors.
func evalOrFail(t *testing.T, expression string) fptypes.Value {
	t.Helper()
	got, err := evalCQL(t, expression)
	if err != nil {
		t.Fatalf("%s: %v", expression, err)
	}
	return got
}

// TestPrecisionUncertainty_ConsistentAcrossOperators is the property this file
// exists for: one pair of values, specified to different precisions, must give the
// same answer whichever operator asks. Each of these reached the comparison by a
// different route — fptypes ordering, the interval helpers, and the timing
// operators' own component walk — and they used to disagree, with `<` saying
// unknown while `=` said true and `before` said false.
func TestPrecisionUncertainty_ConsistentAcrossOperators(t *testing.T) {
	expressions := []string{
		// Ordering, through types.CompareTemporal
		"@2017-09-01T00:00:00 < @2017-09-01T00:00:00.000",
		"@2017-09-01T00:00:00 > @2017-09-01T00:00:00.000",
		"@2017-09-01T00:00:00 <= @2017-09-01T00:00:00.000",
		// Equality, which has no error channel of its own
		"@2017-09-01T00:00:00 = @2017-09-01T00:00:00.000",
		"@2017-09-01T00:00:00 != @2017-09-01T00:00:00.000",
		// Timing operators, through temporalCompareAtPrecision
		"@T15:59:59 before @T15:59:59.999",
		"@T15:59:59 after @T15:59:59.999",
		"@T15:59:59 same as @T15:59:59.999",
		"@T15:59:59 same or before @T15:59:59.999",
		"@T15:59:59 same or after @T15:59:59.999",
		// A date against a datetime is the same question at a coarser level
		"@2020-01-01 same as @2020-01-01T10:00:00",
		// Interval membership, through the interval helpers
		"Interval[@T15:59:59.999, @T20:00:00.000] properly includes @T15:59:59",
	}

	for _, expression := range expressions {
		t.Run(expression, func(t *testing.T) {
			if got := evalOrFail(t, expression); got != nil {
				t.Errorf("%s = %v, want null", expression, got)
			}
		})
	}
}

// TestPrecisionUncertainty_ExplicitPrecisionStillDecides is the boundary of that
// rule. Naming a precision says how far to look, so agreeing that far settles the
// question rather than leaving it open.
func TestPrecisionUncertainty_ExplicitPrecisionStillDecides(t *testing.T) {
	cases := []struct {
		expression string
		want       bool
	}{
		{"@T15:59:59 same second as @T15:59:59.999", true},
		{"@T15:59:59 before second of @T15:59:59.999", false},
		{"@2017-09-01T00:00:00 same day as @2017-09-01T00:00:00.000", true},
		// Equivalence never yields null in CQL, whatever the precisions
		{"@2017-09-01T00:00:00 ~ @2017-09-01T00:00:00.000", true},
	}

	for _, tc := range cases {
		t.Run(tc.expression, func(t *testing.T) {
			got := evalOrFail(t, tc.expression)
			want := fptypes.NewBoolean(tc.want)
			if got == nil || !got.Equal(want) {
				t.Errorf("%s = %v, want %v", tc.expression, got, tc.want)
			}
		})
	}
}

// TestPrecisionUncertainty_DecidedComparisonsUnaffected guards the other direction.
// Turning a definite answer into null is the failure mode this change could
// introduce, so the comparisons that a shared component settles are pinned here,
// alongside ordinary values that the temporal rule must not touch at all.
func TestPrecisionUncertainty_DecidedComparisonsUnaffected(t *testing.T) {
	cases := []struct {
		expression string
		want       bool
	}{
		{"@T15:59:59 before @T20:00:00.999", true},
		{"@2017-09-01T00:00:00 < @2017-09-01T00:00:01", true},
		{"@2020-01-01 < @2020-06-01", true},
		{"@2020-01-01 = @2020-01-01", true},
		{"@2020-01-01 = @2020-06-01", false},
		{"@2020-01-01 != @2020-06-01", true},
		{"1 < 2", true},
		{"1 = 1", true},
		{"'a' = 'a'", true},
		{"1 != 2", true},
	}

	for _, tc := range cases {
		t.Run(tc.expression, func(t *testing.T) {
			got := evalOrFail(t, tc.expression)
			want := fptypes.NewBoolean(tc.want)
			if got == nil || !got.Equal(want) {
				t.Errorf("%s = %v, want %v", tc.expression, got, tc.want)
			}
		})
	}
}

// TestPrecisionUncertainty_OrderingOperationsStayTotal covers the operations left
// deliberately outside the rule. Min, Max and sort need a total order; answering
// "unknown" there would drop values from a result rather than describe them, so
// they keep ordering at the shared precision.
func TestPrecisionUncertainty_OrderingOperationsStayTotal(t *testing.T) {
	cases := []struct {
		expression string
		want       string
	}{
		{"Min({ @2020-01-01, @2019-01-01, @2021-01-01 })", "2019-01-01"},
		{"Max({ @2020-01-01, @2019-01-01, @2021-01-01 })", "2021-01-01"},
		{"Min({ @2017-09-01T00:00:00, @2017-09-01T00:00:00.000 })", "2017-09-01T00:00:00"},
	}

	for _, tc := range cases {
		t.Run(tc.expression, func(t *testing.T) {
			got := evalOrFail(t, tc.expression)
			if got == nil {
				t.Fatalf("%s returned null, want %s", tc.expression, tc.want)
			}
			if got.String() != tc.want {
				t.Errorf("%s = %s, want %s", tc.expression, got.String(), tc.want)
			}
		})
	}

	sorted := evalOrFail(t, "({ @2020-03-01, @2020-01-01, @2020-02-01 }) S sort asc")
	list, ok := sorted.(interface{ String() string })
	if !ok || sorted == nil {
		t.Fatalf("sort returned %v", sorted)
	}
	want := "{2020-01-01, 2020-02-01, 2020-03-01}"
	if list.String() != want {
		t.Errorf("sort asc = %s, want %s", list.String(), want)
	}
}
