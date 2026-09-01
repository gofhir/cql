package conformance

import (
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

// parseNotation reads expected-output text with the assumed offset left at zero,
// which is UTC rather than the absence of one — fhirpath reports (0, true) for
// such a value, not (0, false). The difference does not reach the cases below,
// which are about how the corpus's notation maps onto values and compare equal
// either way, and the offset a bare DateTime assumes is fixed separately by
// TestParseAssumesTheRequestOffset.
func parseNotation(raw string) (fptypes.Value, error) {
	return outputParser{}.parse(raw)
}

func TestParseExpectedOutput(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		v, err := parseNotation("null")
		assertNoError(t, err)
		if v != nil {
			t.Errorf("expected nil, got %v", v)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		v, err := parseNotation("")
		assertNoError(t, err)
		if v != nil {
			t.Errorf("expected nil, got %v", v)
		}
	})

	t.Run("boolean true", func(t *testing.T) {
		v, err := parseNotation("true")
		assertNoError(t, err)
		expected := fptypes.NewBoolean(true)
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("boolean false", func(t *testing.T) {
		v, err := parseNotation("false")
		assertNoError(t, err)
		expected := fptypes.NewBoolean(false)
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("integer 42", func(t *testing.T) {
		v, err := parseNotation("42")
		assertNoError(t, err)
		expected := fptypes.NewInteger(42)
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("integer -1", func(t *testing.T) {
		v, err := parseNotation("-1")
		assertNoError(t, err)
		expected := fptypes.NewInteger(-1)
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("integer 0", func(t *testing.T) {
		v, err := parseNotation("0")
		assertNoError(t, err)
		expected := fptypes.NewInteger(0)
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("decimal 5.0", func(t *testing.T) {
		v, err := parseNotation("5.0")
		assertNoError(t, err)
		expected, _ := fptypes.NewDecimal("5.0")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("decimal 3.33333333", func(t *testing.T) {
		v, err := parseNotation("3.33333333")
		assertNoError(t, err)
		expected, _ := fptypes.NewDecimal("3.33333333")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("string hello", func(t *testing.T) {
		v, err := parseNotation("'hello'")
		assertNoError(t, err)
		expected := fptypes.NewString("hello")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("string abc", func(t *testing.T) {
		v, err := parseNotation("'abc'")
		assertNoError(t, err)
		expected := fptypes.NewString("abc")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("string empty", func(t *testing.T) {
		v, err := parseNotation("''")
		assertNoError(t, err)
		expected := fptypes.NewString("")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		v, err := parseNotation("{}")
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
		v, err := parseNotation("{1, 2, 3}")
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
		v, err := parseNotation("{'a','b','c'}")
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
		v, err := parseNotation("Interval[2, 7]")
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
		v, err := parseNotation("Interval(2, 7]")
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
		v, err := parseNotation("Interval(2, 7)")
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
		v, err := parseNotation("Interval[null, 7]")
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
		v, err := parseNotation("@2014")
		assertNoError(t, err)
		expected, _ := fptypes.NewDate("2014")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("date year-month", func(t *testing.T) {
		v, err := parseNotation("@2014-01")
		assertNoError(t, err)
		expected, _ := fptypes.NewDate("2014-01")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("date full", func(t *testing.T) {
		v, err := parseNotation("@2014-01-01")
		assertNoError(t, err)
		expected, _ := fptypes.NewDate("2014-01-01")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("datetime year with T", func(t *testing.T) {
		v, err := parseNotation("@2014T")
		assertNoError(t, err)
		expected, _ := fptypes.NewDateTime("2014")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("datetime date with T suffix", func(t *testing.T) {
		v, err := parseNotation("@2014-01-01T")
		assertNoError(t, err)
		expected, _ := fptypes.NewDateTime("2014-01-01")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("datetime full", func(t *testing.T) {
		v, err := parseNotation("@2016-07-07T06:25:33.910")
		assertNoError(t, err)
		expected, _ := fptypes.NewDateTime("2016-07-07T06:25:33.910")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("time", func(t *testing.T) {
		v, err := parseNotation("@T09:00:00.000")
		assertNoError(t, err)
		expected, _ := fptypes.NewTime("T09:00:00.000")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("quantity simple", func(t *testing.T) {
		v, err := parseNotation("5.0'g'")
		assertNoError(t, err)
		expected, _ := fptypes.NewQuantity("5.0 'g'")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("quantity with space", func(t *testing.T) {
		v, err := parseNotation("19.99 '[lb_av]'")
		assertNoError(t, err)
		expected, _ := fptypes.NewQuantity("19.99 '[lb_av]'")
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("tuple simple", func(t *testing.T) {
		v, err := parseNotation("Tuple { id: 5, name: 'Chris'}")
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
		v, err := parseNotation("Tuple {}")
		assertNoError(t, err)
		tup, ok := v.(cqltypes.Tuple)
		if !ok {
			t.Fatalf("expected cqltypes.Tuple, got %T", v)
		}
		if len(tup.Elements) != 0 {
			t.Errorf("expected empty tuple, got %d elements", len(tup.Elements))
		}
	})

	t.Run("long literal parsed as integer", func(t *testing.T) {
		v, err := parseNotation("3L")
		assertNoError(t, err)
		expected := fptypes.NewInteger(3)
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("whitespace trimming", func(t *testing.T) {
		v, err := parseNotation("  42  ")
		assertNoError(t, err)
		expected := fptypes.NewInteger(42)
		if !expected.Equal(v) {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})

	t.Run("list with nested interval", func(t *testing.T) {
		v, err := parseNotation("{Interval[1, 3], Interval[5, 7]}")
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

// TestParseAssumesTheRequestOffset fixes the rule that lets the harness compare
// at all: an expected DateTime is CQL text, so one written without an offset takes
// the evaluation request's, and one that writes its own keeps it.
//
// Without this the harness compared a value the engine had placed against one its
// own parser had not, and fhirpath answered that two DateTimes printing the same
// string were different values. It answered "equal" before v1.9.1 only because it
// was ignoring the offset — so this is what a version of fhirpath that reads the
// offset correctly requires, not a workaround for one that does not.
func TestParseAssumesTheRequestOffset(t *testing.T) {
	east := outputParser{assumedOffset: 9 * time.Hour}
	west := outputParser{assumedOffset: -5 * time.Hour}

	t.Run("a bare DateTime is placed at the request's offset", func(t *testing.T) {
		v, err := west.parse("@2012-03-10T10:20:00")
		assertNoError(t, err)
		dt, ok := v.(fptypes.DateTime)
		if !ok {
			t.Fatalf("expected DateTime, got %T", v)
		}
		if minutes, placed := dt.EffectiveOffset(); !placed || minutes != -300 {
			t.Errorf("expected an effective offset of -300 minutes, got (%d, %v)", minutes, placed)
		}
	})

	t.Run("the same text under two offsets is two different values", func(t *testing.T) {
		e, err := east.parse("@2012-03-10T10:20:00")
		assertNoError(t, err)
		w, err := west.parse("@2012-03-10T10:20:00")
		assertNoError(t, err)
		if e.String() != w.String() {
			t.Fatalf("the assumed offset must not be written into the value: %v vs %v", e, w)
		}
		if e.Equal(w) {
			t.Error("10:20 at UTC+9 and 10:20 at UTC-5 are different instants and must not compare equal")
		}
	})

	t.Run("a stated offset is left alone", func(t *testing.T) {
		v, err := west.parse("@2012-03-10T10:20:00+09:00")
		assertNoError(t, err)
		dt, ok := v.(fptypes.DateTime)
		if !ok {
			t.Fatalf("expected DateTime, got %T", v)
		}
		if minutes, placed := dt.EffectiveOffset(); !placed || minutes != 540 {
			t.Errorf("expected the written +09:00 to stand, got (%d, %v)", minutes, placed)
		}
	})

	t.Run("a DateTime nested in a container is placed too", func(t *testing.T) {
		v, err := west.parse("{@2012-03-10T10:20:00}")
		assertNoError(t, err)
		list, ok := v.(cqltypes.List)
		if !ok {
			t.Fatalf("expected List, got %T", v)
		}
		dt, ok := list.Values[0].(fptypes.DateTime)
		if !ok {
			t.Fatalf("expected DateTime, got %T", list.Values[0])
		}
		if _, placed := dt.EffectiveOffset(); !placed {
			t.Error("a DateTime inside a list is built by the same parser and must be placed by it")
		}
	})
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
