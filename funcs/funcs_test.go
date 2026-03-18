package funcs

import (
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"
	"github.com/shopspring/decimal"

	cqltypes "github.com/gofhir/cql/types"
)

// ---------------------------------------------------------------------------
// Aggregate tests
// ---------------------------------------------------------------------------

func TestCount(t *testing.T) {
	c := fptypes.Collection{
		fptypes.NewInteger(1),
		fptypes.NewInteger(2),
		fptypes.NewInteger(3),
	}
	result := Count(c)
	assertInteger(t, result, 3, "Count")
}

func TestSum(t *testing.T) {
	c := fptypes.Collection{
		fptypes.NewInteger(10),
		fptypes.NewInteger(20),
		fptypes.NewInteger(30),
	}
	result := Sum(c)
	assertDecimalString(t, result, "60", "Sum")
}

func TestSum_Empty(t *testing.T) {
	result := Sum(fptypes.Collection{})
	if result != nil {
		t.Error("Sum of empty collection should be nil")
	}
}

func TestAvg(t *testing.T) {
	c := fptypes.Collection{
		fptypes.NewInteger(10),
		fptypes.NewInteger(20),
		fptypes.NewInteger(30),
	}
	result := Avg(c)
	assertDecimalString(t, result, "20", "Avg")
}

func TestMin(t *testing.T) {
	c := fptypes.Collection{
		fptypes.NewInteger(30),
		fptypes.NewInteger(10),
		fptypes.NewInteger(20),
	}
	result := Min(c)
	assertInteger(t, result, 10, "Min")
}

func TestMax(t *testing.T) {
	c := fptypes.Collection{
		fptypes.NewInteger(30),
		fptypes.NewInteger(10),
		fptypes.NewInteger(20),
	}
	result := Max(c)
	assertInteger(t, result, 30, "Max")
}

func TestAllTrue(t *testing.T) {
	all := fptypes.Collection{fptypes.NewBoolean(true), fptypes.NewBoolean(true)}
	result := AllTrue(all)
	assertBool(t, result, true, "AllTrue(true,true)")

	mixed := fptypes.Collection{fptypes.NewBoolean(true), fptypes.NewBoolean(false)}
	result = AllTrue(mixed)
	assertBool(t, result, false, "AllTrue(true,false)")
}

func TestAnyTrue(t *testing.T) {
	none := fptypes.Collection{fptypes.NewBoolean(false), fptypes.NewBoolean(false)}
	result := AnyTrue(none)
	assertBool(t, result, false, "AnyTrue(false,false)")

	mixed := fptypes.Collection{fptypes.NewBoolean(false), fptypes.NewBoolean(true)}
	result = AnyTrue(mixed)
	assertBool(t, result, true, "AnyTrue(false,true)")
}

func TestPopulationStdDev(t *testing.T) {
	c := fptypes.Collection{
		fptypes.NewInteger(2),
		fptypes.NewInteger(4),
		fptypes.NewInteger(4),
		fptypes.NewInteger(4),
		fptypes.NewInteger(5),
		fptypes.NewInteger(5),
		fptypes.NewInteger(7),
		fptypes.NewInteger(9),
	}
	result := PopulationStdDev(c)
	if result == nil {
		t.Fatal("PopulationStdDev returned nil")
	}
	d, ok := result.(fptypes.Decimal)
	if !ok {
		t.Fatalf("expected Decimal, got %T", result)
	}
	// Population stddev of {2,4,4,4,5,5,7,9} = 2.0
	f, _ := d.Value().Float64()
	if f < 1.9 || f > 2.1 {
		t.Errorf("PopulationStdDev = %v, want ~2.0", f)
	}
}

// ---------------------------------------------------------------------------
// String ops tests
// ---------------------------------------------------------------------------

func TestUpper(t *testing.T) {
	result := Upper(fptypes.NewString("hello"))
	assertString(t, result, "HELLO", "Upper")
}

func TestLower(t *testing.T) {
	result := Lower(fptypes.NewString("HELLO"))
	assertString(t, result, "hello", "Lower")
}

func TestLength(t *testing.T) {
	result := Length(fptypes.NewString("hello"))
	assertInteger(t, result, 5, "Length")
}

func TestStartsWith(t *testing.T) {
	result := StartsWith(fptypes.NewString("hello world"), fptypes.NewString("hello"))
	assertBool(t, result, true, "StartsWith")

	result = StartsWith(fptypes.NewString("hello world"), fptypes.NewString("world"))
	assertBool(t, result, false, "StartsWith(not)")
}

func TestEndsWith(t *testing.T) {
	result := EndsWith(fptypes.NewString("hello world"), fptypes.NewString("world"))
	assertBool(t, result, true, "EndsWith")
}

func TestSubstring(t *testing.T) {
	result := Substring(fptypes.NewString("hello world"), 6, 5)
	assertString(t, result, "world", "Substring")
}

func TestIndexOf(t *testing.T) {
	result := IndexOf(fptypes.NewString("hello world"), fptypes.NewString("world"))
	assertInteger(t, result, 6, "IndexOf")

	result = IndexOf(fptypes.NewString("hello"), fptypes.NewString("xyz"))
	assertInteger(t, result, -1, "IndexOf(not found)")
}

func TestCombine(t *testing.T) {
	c := fptypes.Collection{
		fptypes.NewString("a"),
		fptypes.NewString("b"),
		fptypes.NewString("c"),
	}
	result := Combine(c, ", ")
	assertString(t, result, "a, b, c", "Combine")
}

func TestSplit(t *testing.T) {
	result := Split(fptypes.NewString("a,b,c"), ",")
	if result == nil {
		t.Fatal("Split returned nil")
	}
	if result.Type() != "List" {
		t.Errorf("Split result type = %s, want List", result.Type())
	}
}

func TestMatches(t *testing.T) {
	// Full regex match: pattern must match entire string
	result := Matches(fptypes.NewString("hello world"), fptypes.NewString("hello.*"))
	assertBool(t, result, true, "Matches regex")

	// Partial match should fail (regex anchored)
	result2 := Matches(fptypes.NewString("hello world"), fptypes.NewString("world"))
	assertBool(t, result2, false, "Matches partial")
}

func TestReplaceMatches(t *testing.T) {
	result := ReplaceMatches(
		fptypes.NewString("hello world"),
		fptypes.NewString("world"),
		fptypes.NewString("Go"),
	)
	assertString(t, result, "hello Go", "ReplaceMatches")
}

// ---------------------------------------------------------------------------
// Type ops tests
// ---------------------------------------------------------------------------

func TestToString(t *testing.T) {
	result := ToString(fptypes.NewInteger(42))
	assertString(t, result, "42", "ToString(integer)")
}

func TestToInteger(t *testing.T) {
	result, err := ToInteger(fptypes.NewString("42"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 42, "ToInteger(string)")

	// Boolean true → 1
	result, err = ToInteger(fptypes.NewBoolean(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 1, "ToInteger(true)")

	// Invalid string → nil (CQL: null)
	result, err = ToInteger(fptypes.NewString("abc"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("ToInteger(non-numeric string) should return nil")
	}
}

func TestToDecimal(t *testing.T) {
	result, err := ToDecimal(fptypes.NewInteger(42))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d, ok := result.(fptypes.Decimal)
	if !ok {
		t.Fatalf("expected Decimal, got %T", result)
	}
	if !d.Value().Equal(decimal.NewFromInt(42)) {
		t.Errorf("ToDecimal(42) = %v, want 42", d.Value())
	}
}

func TestToBoolean(t *testing.T) {
	tests := []struct {
		input    fptypes.Value
		expected bool
	}{
		{fptypes.NewString("true"), true},
		{fptypes.NewString("false"), false},
		{fptypes.NewString("yes"), true},
		{fptypes.NewString("no"), false},
		{fptypes.NewInteger(1), true},
		{fptypes.NewInteger(0), false},
	}
	for _, tt := range tests {
		result, err := ToBoolean(tt.input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertBool(t, result, tt.expected, "ToBoolean("+tt.input.String()+")")
	}
}

func TestToDateTime(t *testing.T) {
	d, _ := fptypes.NewDate("2024-01-15")
	result, err := ToDateTime(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("ToDateTime returned nil")
	}
	if result.Type() != "DateTime" {
		t.Errorf("expected DateTime type, got %s", result.Type())
	}
}

func TestIsType(t *testing.T) {
	if !IsType(fptypes.NewInteger(1), "Integer") {
		t.Error("IsType(Integer, 'Integer') should be true")
	}
	if IsType(fptypes.NewString("x"), "Integer") {
		t.Error("IsType(String, 'Integer') should be false")
	}
}

func TestConvert(t *testing.T) {
	result, err := Convert(fptypes.NewString("42"), "integer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 42, "Convert to integer")

	result, err = Convert(fptypes.NewInteger(1), "boolean")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, true, "Convert integer to boolean")
}

// ---------------------------------------------------------------------------
// Clinical function tests
// ---------------------------------------------------------------------------

func TestCalculateAgeInYears(t *testing.T) {
	bd, _ := fptypes.NewDate("1990-01-15")
	asOf, _ := fptypes.NewDate("2024-06-01")
	result, err := CalculateAgeInYears(bd, asOf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 34, "CalculateAgeInYears")
}

func TestCalculateAgeInMonths(t *testing.T) {
	bd, _ := fptypes.NewDate("2000-03-15")
	asOf, _ := fptypes.NewDate("2000-06-20")
	result, err := CalculateAgeInMonths(bd, asOf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 3, "CalculateAgeInMonths")
}

func TestCalculateAgeInDays(t *testing.T) {
	bd, _ := fptypes.NewDate("2024-01-01")
	asOf, _ := fptypes.NewDate("2024-01-11")
	result, err := CalculateAgeInDays(bd, asOf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 10, "CalculateAgeInDays")
}

func TestCalculateAgeInWeeks(t *testing.T) {
	bd, _ := fptypes.NewDate("2024-01-01")
	asOf, _ := fptypes.NewDate("2024-01-15")
	result, err := CalculateAgeInWeeks(bd, asOf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 2, "CalculateAgeInWeeks")
}

// ---------------------------------------------------------------------------
// Temporal function tests
// ---------------------------------------------------------------------------

func TestNow(t *testing.T) {
	result, err := Now()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Now returned nil")
	}
	if result.Type() != "DateTime" {
		t.Errorf("Now type = %s, want DateTime", result.Type())
	}
}

func TestToday(t *testing.T) {
	result, err := Today()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Today returned nil")
	}
	if result.Type() != "Date" {
		t.Errorf("Today type = %s, want Date", result.Type())
	}
}

func TestYearsBetween(t *testing.T) {
	d1, _ := fptypes.NewDate("2000-01-01")
	d2, _ := fptypes.NewDate("2024-06-15")
	result, err := YearsBetween(d1, d2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 24, "YearsBetween")
}

func TestDaysBetween(t *testing.T) {
	d1, _ := fptypes.NewDate("2024-01-01")
	d2, _ := fptypes.NewDate("2024-01-11")
	result, err := DaysBetween(d1, d2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 10, "DaysBetween")
}

// ---------------------------------------------------------------------------
// Interval function tests
// ---------------------------------------------------------------------------

func TestIntervalContains(t *testing.T) {
	iv := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(10), true, true)
	result, err := IntervalContains(iv, fptypes.NewInteger(5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, true, "IntervalContains(5)")

	result, err = IntervalContains(iv, fptypes.NewInteger(15))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, false, "IntervalContains(15)")
}

func TestIntervalIncludes(t *testing.T) {
	a := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(10), true, true)
	b := cqltypes.NewInterval(fptypes.NewInteger(3), fptypes.NewInteger(7), true, true)
	result, err := IntervalIncludes(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, true, "IntervalIncludes")
}

func TestIntervalOverlaps(t *testing.T) {
	a := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(5), true, true)
	b := cqltypes.NewInterval(fptypes.NewInteger(3), fptypes.NewInteger(8), true, true)
	result, err := IntervalOverlaps(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, true, "IntervalOverlaps")
}

func TestIntervalBefore(t *testing.T) {
	a := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(3), true, true)
	b := cqltypes.NewInterval(fptypes.NewInteger(5), fptypes.NewInteger(8), true, true)
	result, err := IntervalBefore(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, true, "IntervalBefore")
}

func TestIntervalMeets(t *testing.T) {
	a := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(5), true, true)
	b := cqltypes.NewInterval(fptypes.NewInteger(5), fptypes.NewInteger(10), true, true)
	result, err := IntervalMeets(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, true, "IntervalMeets")
}

func TestIntervalStartOf(t *testing.T) {
	iv := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(10), true, true)
	result := IntervalStartOf(iv)
	assertInteger(t, result, 1, "IntervalStartOf")
}

func TestIntervalEndOf(t *testing.T) {
	iv := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(10), true, true)
	result := IntervalEndOf(iv)
	assertInteger(t, result, 10, "IntervalEndOf")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func assertInteger(t *testing.T, v fptypes.Value, expected int64, label string) {
	t.Helper()
	if v == nil {
		t.Fatalf("%s: got nil", label)
	}
	i, ok := v.(fptypes.Integer)
	if !ok {
		t.Fatalf("%s: expected Integer, got %T (%v)", label, v, v)
	}
	if i.Value() != expected {
		t.Errorf("%s = %d, want %d", label, i.Value(), expected)
	}
}

func assertBool(t *testing.T, v fptypes.Value, expected bool, label string) {
	t.Helper()
	if v == nil {
		t.Fatalf("%s: got nil", label)
	}
	b, ok := v.(fptypes.Boolean)
	if !ok {
		t.Fatalf("%s: expected Boolean, got %T (%v)", label, v, v)
	}
	if b.Bool() != expected {
		t.Errorf("%s = %v, want %v", label, b.Bool(), expected)
	}
}

func assertString(t *testing.T, v fptypes.Value, expected string, label string) {
	t.Helper()
	if v == nil {
		t.Fatalf("%s: got nil", label)
	}
	s, ok := v.(fptypes.String)
	if !ok {
		t.Fatalf("%s: expected String, got %T (%v)", label, v, v)
	}
	if s.Value() != expected {
		t.Errorf("%s = %q, want %q", label, s.Value(), expected)
	}
}

func assertDecimalString(t *testing.T, v fptypes.Value, expected string, label string) {
	t.Helper()
	if v == nil {
		t.Fatalf("%s: got nil", label)
	}
	d, ok := v.(fptypes.Decimal)
	if !ok {
		t.Fatalf("%s: expected Decimal, got %T (%v)", label, v, v)
	}
	exp, _ := decimal.NewFromString(expected)
	if !d.Value().Equal(exp) {
		t.Errorf("%s = %v, want %s", label, d.Value(), expected)
	}
}

// ---------------------------------------------------------------------------
// Phase 2 — Advanced String Operations
// ---------------------------------------------------------------------------

func TestPositionOf(t *testing.T) {
	result := PositionOf(fptypes.NewString("world"), fptypes.NewString("hello world"))
	assertInteger(t, result, 6, "PositionOf(world)")

	result = PositionOf(fptypes.NewString("xyz"), fptypes.NewString("hello"))
	assertInteger(t, result, -1, "PositionOf(not found)")
}

func TestLastPositionOf(t *testing.T) {
	result := LastPositionOf(fptypes.NewString("l"), fptypes.NewString("hello world"))
	assertInteger(t, result, 9, "LastPositionOf(l)")

	result = LastPositionOf(fptypes.NewString("z"), fptypes.NewString("hello"))
	assertInteger(t, result, -1, "LastPositionOf(not found)")
}

func TestMatches_Regex(t *testing.T) {
	// Full match with digit pattern
	result := Matches(fptypes.NewString("12345"), fptypes.NewString(`\d+`))
	assertBool(t, result, true, "Matches(digits)")

	// Partial should NOT match
	result = Matches(fptypes.NewString("abc123"), fptypes.NewString(`\d+`))
	assertBool(t, result, false, "Matches(partial digits)")

	// Invalid regex returns false
	result = Matches(fptypes.NewString("test"), fptypes.NewString("[invalid"))
	assertBool(t, result, false, "Matches(invalid regex)")
}

func TestReplaceMatches_Regex(t *testing.T) {
	// Replace digits with X
	result := ReplaceMatches(
		fptypes.NewString("abc123def456"),
		fptypes.NewString(`\d+`),
		fptypes.NewString("X"),
	)
	assertString(t, result, "abcXdefX", "ReplaceMatches regex")
}

// ---------------------------------------------------------------------------
// Phase 2 — Advanced Date/Time Operations
// ---------------------------------------------------------------------------

func TestDateTimeComponentFrom(t *testing.T) {
	dt, _ := fptypes.NewDateTime("2024-03-15T10:30:45.123Z")

	tests := []struct {
		component string
		expected  int64
	}{
		{"year", 2024},
		{"month", 3},
		{"day", 15},
		{"hour", 10},
		{"minute", 30},
		{"second", 45},
		{"millisecond", 123},
	}
	for _, tt := range tests {
		result, err := DateTimeComponentFrom(dt, tt.component)
		if err != nil {
			t.Fatalf("DateTimeComponentFrom(%s): unexpected error: %v", tt.component, err)
		}
		assertInteger(t, result, tt.expected, "DateTimeComponentFrom("+tt.component+")")
	}
}

func TestDateTimeComponentFrom_Date(t *testing.T) {
	// "date" component should return a Date type
	dt, _ := fptypes.NewDateTime("2024-03-15T10:30:45Z")
	result, err := DateTimeComponentFrom(dt, "date")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for date component")
	}
	if result.Type() != "Date" {
		t.Errorf("type = %s, want Date", result.Type())
	}
}

func TestDateTimeComponentFrom_Null(t *testing.T) {
	result, err := DateTimeComponentFrom(nil, "year")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil for null operand")
	}
}

func TestDayOfWeek(t *testing.T) {
	// 2024-03-15 is a Friday (5)
	d, _ := fptypes.NewDate("2024-03-15")
	result, err := DayOfWeek(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 5, "DayOfWeek(Friday)")
}

func TestDateConstructor(t *testing.T) {
	result, err := DateConstructor(
		fptypes.NewInteger(2024),
		fptypes.NewInteger(6),
		fptypes.NewInteger(15),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil date")
	}
	if result.Type() != "Date" {
		t.Errorf("type = %s, want Date", result.Type())
	}
}

func TestDateConstructor_YearOnly(t *testing.T) {
	result, err := DateConstructor(
		fptypes.NewInteger(2024),
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil date")
	}
}

func TestDateTimeConstructor(t *testing.T) {
	result, err := DateTimeConstructor(
		fptypes.NewInteger(2024),
		fptypes.NewInteger(3),
		fptypes.NewInteger(15),
		fptypes.NewInteger(10),
		fptypes.NewInteger(30),
		fptypes.NewInteger(0),
		fptypes.NewInteger(0),
		nil, // tz offset
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil datetime")
	}
	if result.Type() != "DateTime" {
		t.Errorf("type = %s, want DateTime", result.Type())
	}
}

func TestTimeConstructor(t *testing.T) {
	result, err := TimeConstructor(
		fptypes.NewInteger(14),
		fptypes.NewInteger(30),
		fptypes.NewInteger(0),
		fptypes.NewInteger(0),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil time")
	}
	if result.Type() != "Time" {
		t.Errorf("type = %s, want Time", result.Type())
	}
}

func TestDurationBetween_Years(t *testing.T) {
	d1, _ := fptypes.NewDate("2000-01-01")
	d2, _ := fptypes.NewDate("2024-01-01")
	result, err := DurationBetween(d1, d2, "years")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 24, "DurationBetween(years)")
}

func TestDurationBetween_Days(t *testing.T) {
	d1, _ := fptypes.NewDate("2024-01-01")
	d2, _ := fptypes.NewDate("2024-01-11")
	result, err := DurationBetween(d1, d2, "days")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 10, "DurationBetween(days)")
}

func TestDurationBetween_Null(t *testing.T) {
	result, err := DurationBetween(nil, fptypes.NewInteger(1), "days")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil for null operand")
	}
}

func TestMillisecondsBetween(t *testing.T) {
	d1, _ := fptypes.NewDateTime("2024-01-01T00:00:00.000Z")
	d2, _ := fptypes.NewDateTime("2024-01-01T00:00:01.500Z")
	result, err := MillisecondsBetween(d1, d2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 1500, "MillisecondsBetween")
}

func TestDateAdd_Years(t *testing.T) {
	d, _ := fptypes.NewDate("2020-01-15")
	result, err := DateAdd(d, 4, "years")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil date")
	}
	if result.Type() != "Date" {
		t.Errorf("type = %s, want Date", result.Type())
	}
}

func TestDateAdd_Days(t *testing.T) {
	d, _ := fptypes.NewDate("2024-01-01")
	result, err := DateAdd(d, 10, "days")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil")
	}
}

// ---------------------------------------------------------------------------
// Phase 2 — Advanced Interval Operations
// ---------------------------------------------------------------------------

func TestIntervalWidth_Integer(t *testing.T) {
	iv := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(10), true, true)
	result, err := IntervalWidth(iv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 9, "IntervalWidth([1,10])")
}

func TestIntervalWidth_OpenBoundary(t *testing.T) {
	// (1, 10) → effective 2..9, width = 7
	iv := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(10), false, false)
	result, err := IntervalWidth(iv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 7, "IntervalWidth((1,10))")
}

func TestIntervalSize_Integer(t *testing.T) {
	iv := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(10), true, true)
	result, err := IntervalSize(iv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 10, "IntervalSize([1,10])")
}

func TestIntervalPointFrom(t *testing.T) {
	// Unit interval [5,5]
	iv := cqltypes.NewInterval(fptypes.NewInteger(5), fptypes.NewInteger(5), true, true)
	result, err := IntervalPointFrom(iv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertInteger(t, result, 5, "IntervalPointFrom([5,5])")

	// Non-unit interval [1,5] → nil
	iv2 := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(5), true, true)
	result, err = IntervalPointFrom(iv2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("IntervalPointFrom of non-unit interval should be nil")
	}
}

func TestIntervalProperlyIncludes(t *testing.T) {
	a := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(10), true, true)
	b := cqltypes.NewInterval(fptypes.NewInteger(3), fptypes.NewInteger(7), true, true)

	result, err := IntervalProperlyIncludes(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, true, "ProperlyIncludes([1,10], [3,7])")

	// Same interval → false
	result, err = IntervalProperlyIncludes(a, a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, false, "ProperlyIncludes(self)")
}

func TestIntervalMeetsBefore(t *testing.T) {
	a := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(5), true, true)
	b := cqltypes.NewInterval(fptypes.NewInteger(6), fptypes.NewInteger(10), true, true)

	result, err := IntervalMeetsBefore(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, true, "MeetsBefore([1,5], [6,10])")

	// Non-meeting
	c := cqltypes.NewInterval(fptypes.NewInteger(8), fptypes.NewInteger(10), true, true)
	result, err = IntervalMeetsBefore(a, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, false, "MeetsBefore([1,5], [8,10])")
}

func TestIntervalCollapse(t *testing.T) {
	intervals := []cqltypes.Interval{
		cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(5), true, true),
		cqltypes.NewInterval(fptypes.NewInteger(3), fptypes.NewInteger(8), true, true),
		cqltypes.NewInterval(fptypes.NewInteger(10), fptypes.NewInteger(15), true, true),
	}
	result, err := IntervalCollapse(intervals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 collapsed intervals, got %d", len(result))
	}
}

func TestIntervalExpand(t *testing.T) {
	iv := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(5), true, true)
	result, err := IntervalExpand(iv, decimal.Zero)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 5 {
		t.Fatalf("expected 5 expanded points, got %d", len(result))
	}
	assertInteger(t, result[0], 1, "Expand[0]")
	assertInteger(t, result[4], 5, "Expand[4]")
}

// ---------------------------------------------------------------------------
// Phase 2 — Advanced List Operations
// ---------------------------------------------------------------------------

func TestFlatten(t *testing.T) {
	inner := collectionToList(fptypes.Collection{
		fptypes.NewInteger(3),
		fptypes.NewInteger(4),
	})
	c := fptypes.Collection{
		fptypes.NewInteger(1),
		fptypes.NewInteger(2),
		inner,
	}
	result := Flatten(c)
	if len(result) != 4 {
		t.Fatalf("expected 4 items after flatten, got %d", len(result))
	}
}

func TestDistinct(t *testing.T) {
	c := fptypes.Collection{
		fptypes.NewInteger(1),
		fptypes.NewInteger(2),
		fptypes.NewInteger(1),
		fptypes.NewInteger(3),
		fptypes.NewInteger(2),
	}
	result := Distinct(c)
	if len(result) != 3 {
		t.Fatalf("expected 3 distinct items, got %d", len(result))
	}
}

func TestMode(t *testing.T) {
	c := fptypes.Collection{
		fptypes.NewInteger(1),
		fptypes.NewInteger(2),
		fptypes.NewInteger(2),
		fptypes.NewInteger(3),
	}
	result := Mode(c)
	assertInteger(t, result, 2, "Mode")
}

func TestMode_Empty(t *testing.T) {
	result := Mode(fptypes.Collection{})
	if result != nil {
		t.Error("Mode of empty collection should be nil")
	}
}

func TestMedian_Odd(t *testing.T) {
	c := fptypes.Collection{
		fptypes.NewInteger(3),
		fptypes.NewInteger(1),
		fptypes.NewInteger(2),
	}
	result := Median(c)
	assertDecimalString(t, result, "2", "Median(odd)")
}

func TestMedian_Even(t *testing.T) {
	c := fptypes.Collection{
		fptypes.NewInteger(1),
		fptypes.NewInteger(2),
		fptypes.NewInteger(3),
		fptypes.NewInteger(4),
	}
	result := Median(c)
	assertDecimalString(t, result, "2.5", "Median(even)")
}

func TestGeometricMean(t *testing.T) {
	c := fptypes.Collection{
		fptypes.NewInteger(4),
		fptypes.NewInteger(9),
	}
	result := GeometricMean(c)
	if result == nil {
		t.Fatal("GeometricMean returned nil")
	}
	d, ok := result.(fptypes.Decimal)
	if !ok {
		t.Fatalf("expected Decimal, got %T", result)
	}
	f, _ := d.Value().Float64()
	if f < 5.9 || f > 6.1 {
		t.Errorf("GeometricMean(4,9) = %v, want ~6.0", f)
	}
}

func TestGeometricMean_NonPositive(t *testing.T) {
	c := fptypes.Collection{
		fptypes.NewInteger(4),
		fptypes.NewInteger(0),
	}
	result := GeometricMean(c)
	if result != nil {
		t.Error("GeometricMean with zero should return nil")
	}
}

func TestFirst(t *testing.T) {
	c := fptypes.Collection{fptypes.NewInteger(10), fptypes.NewInteger(20)}
	result := First(c)
	assertInteger(t, result, 10, "First")

	if First(fptypes.Collection{}) != nil {
		t.Error("First of empty should be nil")
	}
}

func TestLast(t *testing.T) {
	c := fptypes.Collection{fptypes.NewInteger(10), fptypes.NewInteger(20)}
	result := Last(c)
	assertInteger(t, result, 20, "Last")
}

func TestSingletonFrom(t *testing.T) {
	single := fptypes.Collection{fptypes.NewInteger(42)}
	result := SingletonFrom(single)
	assertInteger(t, result, 42, "SingletonFrom")

	multi := fptypes.Collection{fptypes.NewInteger(1), fptypes.NewInteger(2)}
	if SingletonFrom(multi) != nil {
		t.Error("SingletonFrom of multi-element should be nil")
	}
}

func TestExists(t *testing.T) {
	result := Exists(fptypes.Collection{fptypes.NewInteger(1)})
	assertBool(t, result, true, "Exists(non-empty)")

	result = Exists(fptypes.Collection{})
	assertBool(t, result, false, "Exists(empty)")
}

func TestIndexer(t *testing.T) {
	c := fptypes.Collection{fptypes.NewInteger(10), fptypes.NewInteger(20), fptypes.NewInteger(30)}
	result := Indexer(c, 1)
	assertInteger(t, result, 20, "Indexer[1]")

	if Indexer(c, -1) != nil {
		t.Error("Indexer(-1) should be nil")
	}
	if Indexer(c, 5) != nil {
		t.Error("Indexer(out of range) should be nil")
	}
}

func TestTake(t *testing.T) {
	c := fptypes.Collection{fptypes.NewInteger(1), fptypes.NewInteger(2), fptypes.NewInteger(3)}
	result := Take(c, 2)
	if len(result) != 2 {
		t.Fatalf("Take(2) got %d items, want 2", len(result))
	}

	if Take(c, 0) != nil {
		t.Error("Take(0) should be nil")
	}
}

func TestSkip(t *testing.T) {
	c := fptypes.Collection{fptypes.NewInteger(1), fptypes.NewInteger(2), fptypes.NewInteger(3)}
	result := Skip(c, 1)
	if len(result) != 2 {
		t.Fatalf("Skip(1) got %d items, want 2", len(result))
	}
	assertInteger(t, result[0], 2, "Skip[0]")
}

func TestTail(t *testing.T) {
	c := fptypes.Collection{fptypes.NewInteger(1), fptypes.NewInteger(2), fptypes.NewInteger(3)}
	result := Tail(c)
	if len(result) != 2 {
		t.Fatalf("Tail got %d items, want 2", len(result))
	}
	assertInteger(t, result[0], 2, "Tail[0]")

	if Tail(fptypes.Collection{}) != nil {
		t.Error("Tail of empty should be nil")
	}
}

// ---------------------------------------------------------------------------
// Phase 2 — Advanced Timing Operators
// ---------------------------------------------------------------------------

func TestOverlapsBefore(t *testing.T) {
	a := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(5), true, true)
	b := cqltypes.NewInterval(fptypes.NewInteger(3), fptypes.NewInteger(8), true, true)

	result, err := OverlapsBefore(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, true, "OverlapsBefore([1,5], [3,8])")

	// Non-overlapping
	c := cqltypes.NewInterval(fptypes.NewInteger(6), fptypes.NewInteger(10), true, true)
	result, err = OverlapsBefore(a, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, false, "OverlapsBefore non-overlapping")
}

func TestOverlapsAfter(t *testing.T) {
	a := cqltypes.NewInterval(fptypes.NewInteger(3), fptypes.NewInteger(8), true, true)
	b := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(5), true, true)

	result, err := OverlapsAfter(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, true, "OverlapsAfter([3,8], [1,5])")
}

func TestSameAs(t *testing.T) {
	a := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(5), true, true)
	b := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(5), true, true)
	c := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(10), true, true)

	result := SameAs(a, b)
	assertBool(t, result, true, "SameAs(equal)")

	result = SameAs(a, c)
	assertBool(t, result, false, "SameAs(different)")
}

func TestSameOrBefore(t *testing.T) {
	a := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(3), true, true)
	b := cqltypes.NewInterval(fptypes.NewInteger(5), fptypes.NewInteger(10), true, true)

	result, err := SameOrBefore(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, true, "SameOrBefore([1,3], [5,10])")
}

func TestSameOrAfter(t *testing.T) {
	a := cqltypes.NewInterval(fptypes.NewInteger(5), fptypes.NewInteger(10), true, true)
	b := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(3), true, true)

	result, err := SameOrAfter(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, true, "SameOrAfter([5,10], [1,3])")
}

func TestStarts(t *testing.T) {
	a := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(5), true, true)
	b := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(10), true, true)

	result, err := Starts(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, true, "Starts([1,5], [1,10])")

	// Different start → false
	c := cqltypes.NewInterval(fptypes.NewInteger(2), fptypes.NewInteger(5), true, true)
	result, err = Starts(c, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, false, "Starts different start")
}

func TestEnds(t *testing.T) {
	a := cqltypes.NewInterval(fptypes.NewInteger(5), fptypes.NewInteger(10), true, true)
	b := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(10), true, true)

	result, err := Ends(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, true, "Ends([5,10], [1,10])")
}

func TestDuring(t *testing.T) {
	a := cqltypes.NewInterval(fptypes.NewInteger(3), fptypes.NewInteger(7), true, true)
	b := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(10), true, true)

	result, err := During(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBool(t, result, true, "During([3,7], [1,10])")
}

func TestConcurrentWith(t *testing.T) {
	a := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(5), true, true)
	b := cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(5), true, true)

	result := ConcurrentWith(a, b)
	assertBool(t, result, true, "ConcurrentWith(equal)")
}
