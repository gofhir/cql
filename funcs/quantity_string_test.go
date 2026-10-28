package funcs

import (
	"math/big"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
	ucum "github.com/gofhir/ucum/v4"
)

// TestToQuantityReadsTheSpecificationFormat holds the String overload of
// ToQuantity to the CQL Reference (Appendix B, ToQuantity): "(+|-)?#0(.0#)?('<unit>')?",
// the unit "a string literal specifying a valid, case-sensitive UCUM unit of
// measure or calendar duration keyword", "Spaces are allowed between the
// quantity value and the unit designator", and null when "the input string is
// not formatted correctly, or cannot be interpreted as a valid Quantity value".
// The expected values are the specification's examples and the Java engine's
// answers (D-numbers of testdata/snapshots/quantitystrings.cql); where the two
// differ, docs/plans/010-to-quantity.md names the difference. An empty want is
// null.
func TestToQuantityReadsTheSpecificationFormat(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		// The specification's examples.
		{"-0.1 'mg'", "-0.1 'mg'"},
		{"444 'cm", ""},
		// Without a unit, the default unit (D3, D30).
		{"5", "5 '1'"},
		{"0", "0 '1'"},
		{"5 'mg'", "5 'mg'"},
		{"+5 'mg'", "5 'mg'"},
		{"-5 'mg'", "-5 'mg'"},
		{"-0 'mg'", "0 'mg'"},
		// The digits as written, past the Decimal scale too (D63, D90, D189,
		// D198, D207, D216, D234): a Quantity is not brought to it (ADR 002).
		{"5.50 'mg'", "5.50 'mg'"},
		{"005.500", "5.500 '1'"},
		{"1.000000005 'mg'", "1.000000005 'mg'"},
		{"0.000000001 'mg'", "0.000000001 'mg'"},
		{"-0.000000001 'mg'", "-0.000000001 'mg'"},
		{"-1.00000001 'mg'", "-1.00000001 'mg'"},
		{"0.000000000 'mg'", "0.000000000 'mg'"},
		{"1e-10 'mg'", "0.0000000001 'mg'"},
		{"12345678901234567890.12345678 'mg'", "12345678901234567890.12345678 'mg'"},
		// The point with no digit after it and the exponent, as ToDecimal reads
		// them (decision A of plan 009; D99, D117, D135, D162).
		{"5. 'mg'", "5 'mg'"},
		{"1e3 'mg'", "1000 'mg'"},
		{"1e-3 'mg'", "0.001 'mg'"},
		{"1.5e2 'mg'", "150 'mg'"},
		// The range of the Decimal type (D252, D261, D270, D279, D297, D315,
		// D324).
		{"99999999999999999999 'mg'", "99999999999999999999 'mg'"},
		{"100000000000000000000 'mg'", ""},
		{"-99999999999999999999.99999999 'mg'", "-99999999999999999999.99999999 'mg'"},
		{"99999999999999999999.999999991 'mg'", ""},
		{"1e19 'mg'", "10000000000000000000 'mg'"},
		{"1e20 'mg'", ""},
		{"1e10001 'mg'", ""},
		// Past the Decimal's precision of 28 digits: the Java engine writes
		// ten thousand zeros (D306); see the plan.
		{"1e-10001 'mg'", ""},
		{"0." + strings.Repeat("0", 28) + "1 'mg'", ""},
		// Not a number of the format (D333, D360, D369, D378, D396, D414, D423,
		// D441, D459, D477); '.5' as ToDecimal, which the Java engine reads as
		// 0.5 here, and an Arabic-Indic digit, which it reads as 5 (D450).
		{".5 'mg'", ""},
		{"+.5", ""},
		{"−5 'mg'", ""},
		{".-5 'mg'", ""},
		{"5..5 'mg'", ""},
		{"--5 'mg'", ""},
		{"e3 'mg'", ""},
		{"1e 'mg'", ""},
		{"0x10 'mg'", ""},
		{"5,5 'mg'", ""},
		{"Infinity 'mg'", ""},
		{"٥ 'mg'", ""},
		{"", ""},
		{"abc", ""},
		{"'mg'", ""},
		// Spaces between, or none: allowed, not required (the Java engine
		// answers null to the second; D6).
		{"5    'mg'", "5 'mg'"},
		{"5'mg'", "5 'mg'"},
		{"5\t'mg'", "5 'mg'"},
		// Whatever is up to U+0020 around the quantity, as the Java engine
		// trims it (D780, D783 to D837); a line break between the number and
		// the unit is null there (D810, D813), and so is a non-breaking space
		// (D819, D822).
		{" 5 'mg' ", "5 'mg'"},
		{"\r\n5 'mg'\r\n", "5 'mg'"},
		{"\n5\n", "5 '1'"},
		{"\f5 'mg'\v", "5 'mg'"},
		{"\x005 'mg'\x00", "5 'mg'"},
		{"5 'mg'\x00x", ""},
		{"5\n'mg'", ""},
		{"5\r\n'mg'", ""},
		{" 5 'mg'", ""},
		{"5 'mg'", ""},
		{"5 'mg' ", ""},
		// The unit is a string literal: unquoted, unclosed or followed by
		// anything, it is null (the Java engine joins the pieces).
		{"5 mg", ""},
		{"5 'mg", ""},
		{"5 'mg'x", ""},
		{"5 'mg' 'g'", ""},
		{"5 'mg''", ""},
		{"5 ''mg'", ""},
		{"5 \"mg\"", ""},
		{"5 'mg' 5", ""},
		// A calendar duration keyword is a unit designator too, quoted as any
		// other; unquoted it is null (decision C).
		{"5 'days'", "5 days"},
		{"5 'millisecond'", "5 millisecond"},
		{"5 days", ""},
		{"5 'DAYS'", ""},
		// A valid UCUM unit, case-sensitive (decision A): an unknown one, one
		// with a space and the empty one are null.
		{"5 'mg/dL'", "5 'mg/dL'"},
		{"5 '[lb_av]'", "5 '[lb_av]'"},
		{"5 '{tablet}'", "5 '{tablet}'"},
		{"5 'd'", "5 'd'"},
		{"5 '1'", "5 '1'"},
		{"5 'foo'", ""},
		{"5 'm g'", ""},
		{"5 ''", ""},
	} {
		got, err := ToQuantity(fptypes.NewString(tt.in))
		if err != nil {
			t.Errorf("ToQuantity(%q): %v", tt.in, err)
			continue
		}
		if tt.want == "" {
			if got != nil {
				t.Errorf("ToQuantity(%q) = %v, want null", tt.in, got)
			}
			continue
		}
		q, ok := got.(fptypes.Quantity)
		if !ok {
			t.Errorf("ToQuantity(%q) = %v, want %s", tt.in, got, tt.want)
			continue
		}
		if q.String() != tt.want {
			t.Errorf("ToQuantity(%q) = %s, want %s", tt.in, q, tt.want)
		}
	}
}

// TestToQuantityGivesANumberTheDefaultUnit covers the numeric overloads: "the
// operator returns a quantity with the value of the argument and a unit of '1'
// (the default unit)" (CQL Reference, ToQuantity; its examples ToQuantity(0.1)
// and ToQuantity(13)).
func TestToQuantityGivesANumberTheDefaultUnit(t *testing.T) {
	for _, tt := range []struct {
		in   fptypes.Value
		want string
	}{
		{fptypes.NewInteger(13), "13 '1'"},
		{fptypes.NewInteger(-2147483648), "-2147483648 '1'"},
		{fptypes.NewDecimalFromDecimal(decimal.RequireFromString("0.1")), "0.1 '1'"},
		{fptypes.NewDecimalFromDecimal(decimal.RequireFromString("5.50")), "5.50 '1'"},
	} {
		got, err := ToQuantity(tt.in)
		if err != nil {
			t.Errorf("ToQuantity(%v): %v", tt.in, err)
			continue
		}
		if q, ok := got.(fptypes.Quantity); !ok || q.String() != tt.want {
			t.Errorf("ToQuantity(%v) = %v, want %s", tt.in, got, tt.want)
		}
	}
}

// quantityFormat is the String overload's format, written apart from the reader
// it checks: the number as ToDecimal reads it, spaces or tabs, an optional
// quoted unit, and anything up to U+0020 around, as Java's String.trim takes
// it away (plan 010).
var quantityFormat = regexp.MustCompile(`^[\x00- ]*([+-]?)([0-9]+)(?:\.([0-9]*))?(?:[eE]([+-]?[0-9]+))?[ \t]*(?:('[^']*'))?[\x00- ]*$`)

// oracleUnits is the oracle's own UCUM service, loaded once.
var oracleUnits = sync.OnceValues(ucum.New)

// expectedQuantity is what the specification makes of s, by quantityFormat and
// the decimal library's own reader rather than QuantityFromString's: the
// number below 10^20, the Decimal's range, with at most 28 digits in fixed
// notation, the integer digits and the places written, the Decimal's
// precision; and the unit a calendar keyword or a valid UCUM code. ok is false
// for null.
func expectedQuantity(t *testing.T, s string) (want fptypes.Quantity, ok bool) {
	t.Helper()
	m := quantityFormat.FindStringSubmatch(s)
	if m == nil {
		return fptypes.Quantity{}, false
	}
	// The value from its digits and the power of the last one, which the
	// exponent and the places written decide together: '0.001e3' is 1.
	exponent := int64(0)
	if m[4] != "" {
		// Within an int32, as ToDecimal reads it (decision C of plan 009).
		e, err := strconv.ParseInt(m[4], 10, 32)
		if err != nil {
			return fptypes.Quantity{}, false
		}
		exponent = e
	}
	power := exponent - int64(len(m[3]))
	places := max(0, -power)
	var value decimal.Decimal
	if digits := strings.TrimLeft(m[2]+m[3], "0"); digits == "" {
		if places > 28 {
			return fptypes.Quantity{}, false
		}
		value = decimal.New(0, int32(-places))
	} else {
		// A power this far out is past the range or the precision either way,
		// and the decimal library would write the number out to check it.
		if power > 100 || power < -100 {
			return fptypes.Quantity{}, false
		}
		coefficient, _ := new(big.Int).SetString(m[1]+digits, 10)
		value = decimal.NewFromBigInt(coefficient, int32(power))
	}
	integerDigits := len(strings.TrimLeft(value.Abs().Truncate(0).String(), "0"))
	if !value.Abs().LessThan(decimal.New(1, 20)) || integerDigits+int(places) > 28 {
		return fptypes.Quantity{}, false
	}
	unit := "1"
	if m[5] != "" {
		unit = m[5][1 : len(m[5])-1]
		service, err := oracleUnits()
		if err != nil {
			t.Fatal(err)
		}
		if !cqltypes.IsCalendarKeyword(unit) && service.Validate(unit) != nil {
			return fptypes.Quantity{}, false
		}
	}
	return fptypes.NewQuantityFromDecimal(value, unit), true
}

// checkQuantity holds QuantityFromString(s) to expectedQuantity: the same
// Quantity, written the same, or null for both.
func checkQuantity(t *testing.T, s string) {
	t.Helper()
	got, err := QuantityFromString(s)
	if err != nil {
		t.Fatalf("QuantityFromString(%q): %v", s, err)
	}
	want, ok := expectedQuantity(t, s)
	switch {
	case !ok && got != nil:
		t.Errorf("QuantityFromString(%q) = %v, want null", s, got)
	case ok && got == nil:
		t.Errorf("QuantityFromString(%q) = null, want %s", s, want)
	case ok:
		q := got.(fptypes.Quantity)
		if q.String() != want.String() || !q.Value().Equal(want.Value()) || q.Unit() != want.Unit() {
			t.Errorf("QuantityFromString(%q) = %s, want %s", s, q, want)
		}
	}
}

// TestToQuantityReadsWhatItIsGiven writes numbers and units into the format
// and reads them back: '<n> \'<u>\” is <n> '<u>', digits and places as
// written, or null where the specification has it null.
func TestToQuantityReadsWhatItIsGiven(t *testing.T) {
	numbers := []string{"0", "-0", "1", "-1", "+7", "0.5", "-0.5", "3.14159", "10.000", "007",
		"1e2", "1E-2", "-2.5e+3", "0e9", "0.0000000000000000000000000001", "0.00000000000000000000000000001",
		"9999999999999999999.999999999", "99999999999999999999.99999999", "100000000000000000000",
		"1234567890123456789", "1e19", "1e20", "1e-28", "1e-29", "5.", "0.000000001",
		"0.0000000000000000000000000000", "0.00000000000000000000000000000", "0e-29", "-0.0"}
	units := []string{"", "mg", "mg/dL", "[lb_av]", "{tablet}", "1", "%", "Cel", "10*3/uL", "kg.m/s2",
		"days", "year", "milliseconds", "d", "mo", "a", "foo", "m g", "MG", "Days"}
	separators := []string{" ", "", "\t", "  "}
	for _, n := range numbers {
		for _, u := range units {
			for _, sep := range separators {
				checkQuantity(t, n)
				checkQuantity(t, n+sep+"'"+u+"'")
				checkQuantity(t, " "+n+sep+"'"+u+"'\n")
			}
		}
	}
}

// FuzzQuantityFromString holds the reader to the format written apart
// (expectedQuantity) on any string: it does not panic, and what it reads, and
// whether it reads it, is what the format and UCUM say.
func FuzzQuantityFromString(f *testing.F) {
	for _, s := range []string{"5 'mg'", "-0.1 'mg'", "444 'cm", "5", "5.50 'mg'", "1e3 'mg'", ".5", "5'mg'",
		"5 days", "5 'days'", "5 'foo'", " 5 'mg' ", "5\n'mg'", "1e-10001", "0.000000000 'mg'", "5 ''", "5 'mg'x"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		checkQuantity(t, s)
	})
}

// TestAQuantityStringOfMillionsOfBytesIsReadInBoundedResources reads strings
// of ten million bytes, which data can carry, in bounded time and memory: the
// bounds are decided from where the digits are, before any is read, and what
// is kept is at most 28 digits. The unit too is bounded: UCUM refuses a long
// code without parsing it (gofhir/ucum v4.2.0).
func TestAQuantityStringOfMillionsOfBytesIsReadInBoundedResources(t *testing.T) {
	const n = 10_000_000
	zeros := strings.Repeat("0", n)
	for _, tt := range []struct{ name, s, want string }{
		{"one written with ten million zeros", "1" + zeros + "e-10000000 'mg'", "null"},
		{"below the precision", "0." + zeros + "1 'mg'", "null"},
		{"a zero of ten million places", "0." + zeros + " 'mg'", "null"},
		{"past the range", "1" + zeros + " 'mg'", "null"},
		{"an exponent of ten million digits", "1e" + strings.Repeat("9", n) + " 'mg'", "null"},
		{"an exponent of ten million leading zeros", "1e-" + zeros + "5 'mg'", "0.00001 'mg'"},
		{"ten million leading zeros", zeros + "5 'mg'", "5 'mg'"},
		{"ten million spaces around", strings.Repeat(" ", n) + "5 'mg'" + strings.Repeat("\n", n), "5 'mg'"},
		{"ten million spaces between", "5" + strings.Repeat(" ", n) + "'mg'", "5 'mg'"},
		{"a unit of ten million bytes", "5 '" + strings.Repeat("m", n) + "'", "null"},
		{"ten million quotes", "5 " + strings.Repeat("'", n), "null"},
	} {
		if _, err := ucumService(); err != nil {
			t.Fatal(err)
		}
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		start := time.Now()
		done := make(chan string, 1)
		go func() {
			v, err := QuantityFromString(tt.s)
			switch {
			case err != nil:
				done <- "error: " + err.Error()
			case v == nil:
				done <- "null"
			default:
				done <- v.String()
			}
		}()
		select {
		case got := <-done:
			runtime.ReadMemStats(&after)
			if got != tt.want {
				t.Errorf("%s = %s; want %s", tt.name, got, tt.want)
			}
			if elapsed := time.Since(start); elapsed > time.Second {
				t.Errorf("%s took %s", tt.name, elapsed)
			}
			// As in TestAStringOfMillionsOfDigitsIsReadInBoundedResources: a
			// copy of the string is megabytes.
			if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 64<<10 {
				t.Errorf("%s allocated %d bytes, more than 64 KiB", tt.name, allocated)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s did not return in 5s", tt.name)
		}
	}
}

// TestAQuantityDoesNotKeepTheStringItWasReadFrom reads short Quantities from
// strings of ten million bytes and keeps nothing of the strings once they are
// gone: neither the Quantities read nor what reading them leaves behind hold
// on to them (review 2 of plan 010, where eight such strings kept 80 MB).
func TestAQuantityDoesNotKeepTheStringItWasReadFrom(t *testing.T) {
	if _, err := ucumService(); err != nil {
		t.Fatal(err)
	}
	read := func(i int) fptypes.Value {
		v, err := QuantityFromString(strings.Repeat("0", 10_000_000) + "5 'mg{retained" + strconv.Itoa(i) + "}'")
		if err != nil || v == nil {
			t.Fatalf("read %d: %v, %v", i, v, err)
		}
		return v
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	kept := make([]fptypes.Value, 0, 8)
	for i := range 8 {
		kept = append(kept, read(i))
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	// Eight Quantities and what UCUM keeps of their units are a few
	// kilobytes; one string of them is ten megabytes.
	if retained := int64(after.HeapAlloc) - int64(before.HeapAlloc); retained > 1<<20 {
		t.Errorf("eight Quantities read from strings of ten million bytes keep %d bytes", retained)
	}
	runtime.KeepAlive(kept)
}
