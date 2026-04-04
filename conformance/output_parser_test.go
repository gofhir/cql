package conformance

import (
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

func TestParseExpectedOutput(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		v, err := parseExpectedOutput("null")
		assertNoError(t, err)
		if v != nil {
			t.Errorf("expected nil, got %v", v)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		v, err := parseExpectedOutput("")
		assertNoError(t, err)
		if v != nil {
			t.Errorf("expected nil, got %v", v)
		}
	})

	t.Run("boolean true", func(t *testing.T) {
		v, err := parseExpectedOutput("true")
		assertNoError(t, err)
		expected := fptypes.NewBoolean(true)
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("boolean false", func(t *testing.T) {
		v, err := parseExpectedOutput("false")
		assertNoError(t, err)
		expected := fptypes.NewBoolean(false)
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("integer 42", func(t *testing.T) {
		v, err := parseExpectedOutput("42")
		assertNoError(t, err)
		expected := fptypes.NewInteger(42)
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("integer -1", func(t *testing.T) {
		v, err := parseExpectedOutput("-1")
		assertNoError(t, err)
		expected := fptypes.NewInteger(-1)
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("integer 0", func(t *testing.T) {
		v, err := parseExpectedOutput("0")
		assertNoError(t, err)
		expected := fptypes.NewInteger(0)
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("decimal 5.0", func(t *testing.T) {
		v, err := parseExpectedOutput("5.0")
		assertNoError(t, err)
		expected, _ := fptypes.NewDecimal("5.0")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("decimal 3.33333333", func(t *testing.T) {
		v, err := parseExpectedOutput("3.33333333")
		assertNoError(t, err)
		expected, _ := fptypes.NewDecimal("3.33333333")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("string hello", func(t *testing.T) {
		v, err := parseExpectedOutput("'hello'")
		assertNoError(t, err)
		expected := fptypes.NewString("hello")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("string abc", func(t *testing.T) {
		v, err := parseExpectedOutput("'abc'")
		assertNoError(t, err)
		expected := fptypes.NewString("abc")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("string empty", func(t *testing.T) {
		v, err := parseExpectedOutput("''")
		assertNoError(t, err)
		expected := fptypes.NewString("")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		v, err := parseExpectedOutput("{}")
		assertNoError(t, err)
		list, ok := v.(cqltypes.List)
		if !ok {
			t.Fatalf("expected cqltypes.List, got %T", v)
		}
		if len(list.Values) != 0 {
			t.Errorf("expected empty list, got %d elements", len(list.Values))
		}
	})

	t.Run("list of integers", func(t *testing.T) {
		v, err := parseExpectedOutput("{1, 2, 3}")
		assertNoError(t, err)
		list, ok := v.(cqltypes.List)
		if !ok {
			t.Fatalf("expected cqltypes.List, got %T", v)
		}
		if len(list.Values) != 3 {
			t.Fatalf("expected 3 elements, got %d", len(list.Values))
		}
		for i, expected := range []int64{1, 2, 3} {
			exp := fptypes.NewInteger(expected)
			if !exp.Equal(list.Values[i]) {
				t.Errorf("element %d: expected %v, got %v", i, exp, list.Values[i])
			}
		}
	})

	t.Run("list of strings", func(t *testing.T) {
		v, err := parseExpectedOutput("{'a','b','c'}")
		assertNoError(t, err)
		list, ok := v.(cqltypes.List)
		if !ok {
			t.Fatalf("expected cqltypes.List, got %T", v)
		}
		if len(list.Values) != 3 {
			t.Fatalf("expected 3 elements, got %d", len(list.Values))
		}
		for i, expected := range []string{"a", "b", "c"} {
			exp := fptypes.NewString(expected)
			if !exp.Equal(list.Values[i]) {
				t.Errorf("element %d: expected %v, got %v", i, exp, list.Values[i])
			}
		}
	})

	t.Run("interval closed-closed", func(t *testing.T) {
		v, err := parseExpectedOutput("Interval[2, 7]")
		assertNoError(t, err)
		iv, ok := v.(cqltypes.Interval)
		if !ok {
			t.Fatalf("expected cqltypes.Interval, got %T", v)
		}
		if !iv.LowClosed {
			t.Error("expected low closed")
		}
		if !iv.HighClosed {
			t.Error("expected high closed")
		}
		if !fptypes.NewInteger(2).Equal(iv.Low) {
			t.Errorf("expected low=2, got %v", iv.Low)
		}
		if !fptypes.NewInteger(7).Equal(iv.High) {
			t.Errorf("expected high=7, got %v", iv.High)
		}
	})

	t.Run("interval open-closed", func(t *testing.T) {
		v, err := parseExpectedOutput("Interval(2, 7]")
		assertNoError(t, err)
		iv, ok := v.(cqltypes.Interval)
		if !ok {
			t.Fatalf("expected cqltypes.Interval, got %T", v)
		}
		if iv.LowClosed {
			t.Error("expected low open")
		}
		if !iv.HighClosed {
			t.Error("expected high closed")
		}
	})

	t.Run("interval open-open", func(t *testing.T) {
		v, err := parseExpectedOutput("Interval(2, 7)")
		assertNoError(t, err)
		iv, ok := v.(cqltypes.Interval)
		if !ok {
			t.Fatalf("expected cqltypes.Interval, got %T", v)
		}
		if iv.LowClosed {
			t.Error("expected low open")
		}
		if iv.HighClosed {
			t.Error("expected high open")
		}
	})

	t.Run("interval with null bounds", func(t *testing.T) {
		v, err := parseExpectedOutput("Interval[null, 7]")
		assertNoError(t, err)
		iv, ok := v.(cqltypes.Interval)
		if !ok {
			t.Fatalf("expected cqltypes.Interval, got %T", v)
		}
		if iv.Low != nil {
			t.Errorf("expected low=nil, got %v", iv.Low)
		}
		if !fptypes.NewInteger(7).Equal(iv.High) {
			t.Errorf("expected high=7, got %v", iv.High)
		}
	})

	t.Run("date year only", func(t *testing.T) {
		v, err := parseExpectedOutput("@2014")
		assertNoError(t, err)
		expected, _ := fptypes.NewDate("2014")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("date year-month", func(t *testing.T) {
		v, err := parseExpectedOutput("@2014-01")
		assertNoError(t, err)
		expected, _ := fptypes.NewDate("2014-01")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("date full", func(t *testing.T) {
		v, err := parseExpectedOutput("@2014-01-01")
		assertNoError(t, err)
		expected, _ := fptypes.NewDate("2014-01-01")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("datetime year with T", func(t *testing.T) {
		v, err := parseExpectedOutput("@2014T")
		assertNoError(t, err)
		expected, _ := fptypes.NewDateTime("2014")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("datetime date with T suffix", func(t *testing.T) {
		v, err := parseExpectedOutput("@2014-01-01T")
		assertNoError(t, err)
		expected, _ := fptypes.NewDateTime("2014-01-01")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("datetime full", func(t *testing.T) {
		v, err := parseExpectedOutput("@2016-07-07T06:25:33.910")
		assertNoError(t, err)
		expected, _ := fptypes.NewDateTime("2016-07-07T06:25:33.910")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("time", func(t *testing.T) {
		v, err := parseExpectedOutput("@T09:00:00.000")
		assertNoError(t, err)
		expected, _ := fptypes.NewTime("T09:00:00.000")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("quantity simple", func(t *testing.T) {
		v, err := parseExpectedOutput("5.0'g'")
		assertNoError(t, err)
		expected, _ := fptypes.NewQuantity("5.0 'g'")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("quantity with space", func(t *testing.T) {
		v, err := parseExpectedOutput("19.99 '[lb_av]'")
		assertNoError(t, err)
		expected, _ := fptypes.NewQuantity("19.99 '[lb_av]'")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("tuple simple", func(t *testing.T) {
		v, err := parseExpectedOutput("Tuple { id: 5, name: 'Chris'}")
		assertNoError(t, err)
		tup, ok := v.(cqltypes.Tuple)
		if !ok {
			t.Fatalf("expected cqltypes.Tuple, got %T", v)
		}
		if !fptypes.NewInteger(5).Equal(tup.Elements["id"]) {
			t.Errorf("expected id=5, got %v", tup.Elements["id"])
		}
		if !fptypes.NewString("Chris").Equal(tup.Elements["name"]) {
			t.Errorf("expected name='Chris', got %v", tup.Elements["name"])
		}
	})

	t.Run("tuple empty", func(t *testing.T) {
		v, err := parseExpectedOutput("Tuple {}")
		assertNoError(t, err)
		tup, ok := v.(cqltypes.Tuple)
		if !ok {
			t.Fatalf("expected cqltypes.Tuple, got %T", v)
		}
		if len(tup.Elements) != 0 {
			t.Errorf("expected empty tuple, got %d elements", len(tup.Elements))
		}
	})

	t.Run("long literal returns error", func(t *testing.T) {
		_, err := parseExpectedOutput("3L")
		if err == nil {
			t.Error("expected error for long literal")
		}
	})

	t.Run("whitespace trimming", func(t *testing.T) {
		v, err := parseExpectedOutput("  42  ")
		assertNoError(t, err)
		expected := fptypes.NewInteger(42)
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("list with nested interval", func(t *testing.T) {
		v, err := parseExpectedOutput("{Interval[1, 3], Interval[5, 7]}")
		assertNoError(t, err)
		list, ok := v.(cqltypes.List)
		if !ok {
			t.Fatalf("expected cqltypes.List, got %T", v)
		}
		if len(list.Values) != 2 {
			t.Fatalf("expected 2 elements, got %d", len(list.Values))
		}
		iv1, ok := list.Values[0].(cqltypes.Interval)
		if !ok {
			t.Fatalf("expected Interval, got %T", list.Values[0])
		}
		if !fptypes.NewInteger(1).Equal(iv1.Low) {
			t.Errorf("expected low=1, got %v", iv1.Low)
		}
	})
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
