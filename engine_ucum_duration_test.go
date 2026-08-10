package cql

import (
	"context"
	"errors"
	"testing"

	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
)

// evalCQL evaluates a single CQL expression through the full parse-and-evaluate path,
// which is where the UCUM units arrive as written rather than as normalized keywords.
func evalCQL(t *testing.T, expression string) (fptypes.Value, error) {
	t.Helper()
	return NewEngine().EvaluateExpression(
		context.Background(), "define X: "+expression, "X", nil, nil)
}

// TestTemporalArithmetic_UCUMUnits covers a CQL duration written the way measures
// actually write it — with a UCUM code rather than a calendar keyword. Every one of
// these silently returned the operand unchanged, reporting success.
func TestTemporalArithmetic_UCUMUnits(t *testing.T) {
	cases := []struct {
		expression string
		want       string
	}{
		{"@2020-01-01T00:00:00 + 5 'wk'", "2020-02-05T00:00:00"},
		{"@2020-01-01T00:00:00 + 5 'd'", "2020-01-06T00:00:00"},
		{"@2020-01-01T00:00:00 + 5 'h'", "2020-01-01T05:00:00"},
		{"@2020-01-01T00:00:00 + 90 'min'", "2020-01-01T01:30:00"},
		{"@2020-01-01T00:00:00 + 30 's'", "2020-01-01T00:00:30"},
		{"@2020-01-01T00:00:00 - 5 'd'", "2019-12-27T00:00:00"},
		{"@2020-01-01 + 5 'd'", "2020-01-06"},
	}

	for _, tc := range cases {
		t.Run(tc.expression, func(t *testing.T) {
			got, err := evalCQL(t, tc.expression)
			if err != nil {
				t.Fatalf("%s: %v", tc.expression, err)
			}
			if got == nil {
				t.Fatalf("%s returned null", tc.expression)
			}
			if got.String() != tc.want {
				t.Errorf("%s = %s, want %s", tc.expression, got.String(), tc.want)
			}
		})
	}
}

// TestTemporalArithmetic_UCUMEqualsCalendarKeyword states the property directly:
// at a week and below the two duration systems are interchangeable in CQL.
func TestTemporalArithmetic_UCUMEqualsCalendarKeyword(t *testing.T) {
	pairs := [][2]string{
		{"@2020-01-01T00:00:00 + 5 'wk'", "@2020-01-01T00:00:00 + 5 weeks"},
		{"@2020-01-01T00:00:00 + 5 'd'", "@2020-01-01T00:00:00 + 5 days"},
		{"@2020-01-01T00:00:00 + 5 'h'", "@2020-01-01T00:00:00 + 5 hours"},
	}

	for _, pair := range pairs {
		t.Run(pair[0], func(t *testing.T) {
			viaUCUM, err := evalCQL(t, pair[0])
			if err != nil {
				t.Fatalf("%s: %v", pair[0], err)
			}
			viaKeyword, err := evalCQL(t, pair[1])
			if err != nil {
				t.Fatalf("%s: %v", pair[1], err)
			}
			if !viaUCUM.Equal(viaKeyword) {
				t.Errorf("%s = %v but %s = %v", pair[0], viaUCUM, pair[1], viaKeyword)
			}
		})
	}
}

// TestTemporalArithmetic_UCUMYearAndMonthRefused covers the two codes CQL keeps
// apart from the calendar. A UCUM year is a fixed 365.25 days, so `+ 5 'a'` has no
// single right answer and must be reported rather than guessed at.
func TestTemporalArithmetic_UCUMYearAndMonthRefused(t *testing.T) {
	for _, expression := range []string{
		"@2020-01-01T00:00:00 + 5 'a'",
		"@2020-01-01T00:00:00 + 5 'mo'",
		"@2020-01-01 - 1 'a'",
	} {
		t.Run(expression, func(t *testing.T) {
			got, err := evalCQL(t, expression)
			if err == nil {
				t.Fatalf("%s returned %v, want an error", expression, got)
			}
			if !errors.Is(err, fptypes.ErrCalendarConversionRequired) {
				t.Errorf("%s gave %v, want ErrCalendarConversionRequired", expression, err)
			}
		})
	}
}

// TestRelationalOperators_PrecisionUncertainty covers the relational operators at the
// second/millisecond boundary, where CQL asks for null because the two values agree on
// everything they share while one is specified more precisely. Nothing in the
// conformance suite pins this for `<` and friends, and it regressed once.
func TestRelationalOperators_PrecisionUncertainty(t *testing.T) {
	uncertain := []string{
		"@2017-09-01T00:00:00 < @2017-09-01T00:00:00.000",
		"@2017-09-01T00:00:00 > @2017-09-01T00:00:00.000",
		"@2017-09-01T00:00:00 <= @2017-09-01T00:00:00.000",
		"@2017-09-01T00:00:00 >= @2017-09-01T00:00:00.000",
		"@T15:59:59 < @T15:59:59.999",
	}
	for _, expression := range uncertain {
		t.Run("null/"+expression, func(t *testing.T) {
			got, err := evalCQL(t, expression)
			if err != nil {
				t.Fatalf("%s: %v", expression, err)
			}
			if got != nil {
				t.Errorf("%s = %v, want null", expression, got)
			}
		})
	}

	// A comparison settled by a component both values specify stays settled, and
	// ordinary values are untouched by the temporal rule
	decided := []string{
		"@T15:59:59 < @T20:00:00.999",
		"@2017-09-01T00:00:00 < @2017-09-01T00:00:01",
		"@2020-01-01 < @2020-06-01",
		"1 < 2",
	}
	for _, expression := range decided {
		t.Run("true/"+expression, func(t *testing.T) {
			got, err := evalCQL(t, expression)
			if err != nil {
				t.Fatalf("%s: %v", expression, err)
			}
			want := fptypes.NewBoolean(true)
			if got == nil || !got.Equal(want) {
				t.Errorf("%s = %v, want true", expression, got)
			}
		})
	}
}

// TestIntervalExpand_UCUMUnits covers the same gap reached through `expand`, where a
// unit that never advanced the cursor did not merely give a wrong answer: the loop ran
// to its iteration cap and returned thousands of copies of a degenerate interval.
func TestIntervalExpand_UCUMUnits(t *testing.T) {
	viaUCUM, err := evalCQL(t, "expand { Interval[@2020-01-01, @2020-01-05] } per 1 'd'")
	if err != nil {
		t.Fatalf("expand per 1 'd': %v", err)
	}
	viaKeyword, err := evalCQL(t, "expand { Interval[@2020-01-01, @2020-01-05] } per 1 day")
	if err != nil {
		t.Fatalf("expand per 1 day: %v", err)
	}

	ucumList, ok := viaUCUM.(cqltypes.List)
	if !ok {
		t.Fatalf("expand per 1 'd' returned %T, want a list", viaUCUM)
	}
	keywordList, ok := viaKeyword.(cqltypes.List)
	if !ok {
		t.Fatalf("expand per 1 day returned %T, want a list", viaKeyword)
	}
	if len(ucumList.Values) != len(keywordList.Values) {
		t.Fatalf("expand per 1 'd' gave %d intervals, want %d (as per 1 day)",
			len(ucumList.Values), len(keywordList.Values))
	}
	if viaUCUM.String() != viaKeyword.String() {
		t.Errorf("expand per 1 'd' = %s, want %s", viaUCUM.String(), viaKeyword.String())
	}
}

// TestIntervalExpand_UCUMYearRefused makes sure the refusal reaches `expand` too,
// rather than quietly falling back to a partial result.
func TestIntervalExpand_UCUMYearRefused(t *testing.T) {
	got, err := evalCQL(t, "expand { Interval[@2020-01-01, @2025-01-01] } per 1 'a'")
	if err == nil {
		t.Fatalf("expand per 1 'a' returned %v, want an error", got)
	}
	if !errors.Is(err, fptypes.ErrCalendarConversionRequired) {
		t.Errorf("expand per 1 'a' gave %v, want ErrCalendarConversionRequired", err)
	}
}

// TestIntervalExpand_UnknownUnitRefused covers a unit that names nothing at all.
// It used to step the cursor nowhere and run the loop to its 10000-iteration cap,
// handing back that many copies of a degenerate interval and reporting success.
func TestIntervalExpand_UnknownUnitRefused(t *testing.T) {
	got, err := evalCQL(t, "expand { Interval[@2020-01-01, @2020-01-05] } per 1 'foo'")
	if err == nil {
		t.Errorf("expand per 1 'foo' returned %v, want an error", got)
	}
}

// TestIntervalExpand_ValidUnitsUnaffected is the other side of that strictness: every
// shape of `per` the language allows still expands.
func TestIntervalExpand_ValidUnitsUnaffected(t *testing.T) {
	cases := []struct {
		expression string
		want       int
	}{
		{"expand { Interval[@2020-01-01, @2020-01-05] } per 1 day", 5},
		{"expand { Interval[@2020-01-01, @2020-01-05] } per 1 'd'", 5},
		{"expand { Interval[@2020-01-01, @2020-01-05] }", 5},
		{"expand { Interval[@2020, @2024] } per 1 year", 5},
		{"expand { Interval[@T10, @T14] } per 1 hour", 5},
		{"expand { Interval[@T10:00:00, @T10:00:04] } per 1 's'", 5},
		{"expand { Interval[1, 10] } per 2", 5},
	}

	for _, tc := range cases {
		t.Run(tc.expression, func(t *testing.T) {
			got, err := evalCQL(t, tc.expression)
			if err != nil {
				t.Fatalf("%s: %v", tc.expression, err)
			}
			list, ok := got.(cqltypes.List)
			if !ok {
				t.Fatalf("%s returned %T, want a list", tc.expression, got)
			}
			if len(list.Values) != tc.want {
				t.Errorf("%s gave %d elements, want %d", tc.expression, len(list.Values), tc.want)
			}
		})
	}
}
