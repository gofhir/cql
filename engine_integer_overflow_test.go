package cql

import (
	"context"
	"errors"
	"testing"

	cqltypes "github.com/gofhir/cql/types"
)

// TestAResultPastItsTypeIsNull covers arithmetic past the 32 bits of an
// Integer and the 64 of a Long. Both wrapped, or for an Integer did not even
// do that: 2147483647 + 1 was 2147483648, still typed an Integer. The
// specification makes each null, and the successor of the largest value an
// error. The edge is the static type's — the same 2147483648 is a Long's —
// so a Long beside an Integer carries the sum, and a duration is left whole,
// as the overflow rule is about arithmetic.
func TestAResultPastItsTypeIsNull(t *testing.T) {
	for expr, want := range map[string]string{
		"2147483647 + 1":                       "null",
		"2147483647 + 1L":                      "2147483648",
		"(-2147483647) - 2":                    "null",
		"46341 * 46341":                        "null",
		"46340 * 46340":                        "2147395600",
		"-(-2147483648)":                       "null",
		"Abs(-2147483648)":                     "null",
		"(-2147483648) div (-1)":               "null",
		"(-2147483648) mod (-1)":               "0",
		"2 ^ 31":                               "null",
		"Power(2, 31)":                         "null",
		"Sum({2147483647, 1})":                 "null",
		"Sum({2147483647, 1, -1})":             "2147483647",
		"Product({65536, 65536})":              "null",
		"Product({65536, 65536, 0})":           "0",
		"Truncate(3000000000.5)":               "null",
		"ToInteger(2147483648L)":               "null",
		"ToInteger(5L)":                        "5",
		"9223372036854775807L + 1L":            "null",
		"4611686018427387904L * 2L":            "null",
		"(-9223372036854775807L) - 2L":         "null",
		"Abs(-9223372036854775807L - 1L)":      "null",
		"Sum({9223372036854775807L, 1L, -1L})": "9223372036854775807",
		"Product({4611686018427387904L, 2L})":  "null",
		"3037000499L * 3037000499L":            "9223372030926249001",
		"milliseconds between @2000-01-01T00:00:00.000Z and @2020-01-01T00:00:00.000Z": "631152000000",
	} {
		v, err := NewEngine().EvaluateExpression(context.Background(), "library T version '1.0'\ndefine R: "+expr+"\n", "R", nil, nil)
		got := "null"
		if v != nil {
			got = valueString(v)
		}
		if err != nil || got != want {
			t.Errorf("%s = %s, %v; want %s", expr, got, err, want)
		}
	}
	for _, expr := range []string{
		"successor of 2147483647",
		"predecessor of (-2147483648)",
		"successor of 9223372036854775807L",
	} {
		_, err := NewEngine().EvaluateExpression(context.Background(), "library T version '1.0'\ndefine R: "+expr+"\n", "R", nil, nil)
		if !errors.Is(err, cqltypes.ErrPastRange) {
			t.Errorf("%s: err = %v; want ErrPastRange", expr, err)
		}
	}
}

// TestALongSemaCannotTypeIsNotNulled covers the values sema has no type for: a
// definition or a function of an included library, an alias over a List<Any>.
// Sema types an operation between one and an Integer as an Integer, so the
// static type alone nulled a correct Long — L.Big + 1 was null. The 32-bit
// edge holds only where every operand is an Integer for certain, which a
// definition of this library made of literals is.
func TestALongSemaCannotTypeIsNotNulled(t *testing.T) {
	included := "library L version '1.0'\n" +
		"define Big: 3000000000L\n" +
		"define function F(x Long): x * 2\n"
	engine := NewEngine(WithLibraryResolver(func(_ context.Context, name, _ string) (string, error) {
		return included, nil
	}))
	for expr, want := range map[string]string{
		"L.Big + 1":           "3000000001",
		"L.Big * 1":           "3000000000",
		"Sum({L.Big, 1})":     "3000000001",
		"-L.Big":              "-3000000000",
		"Abs(L.Big)":          "3000000000",
		"L.F(2000000000) + 1": "4000000001",
		"(List<Any>{ 3000000000L }) Q return singleton from ({Q + 1})": "{3000000001}",
		"Max24 + 1": "null",
		"Max24 * 2": "null",
	} {
		src := "library T version '1.0'\ninclude L version '1.0'\n" +
			"define Max24: 2147483647\ndefine R: " + expr + "\n"
		v, err := engine.EvaluateExpression(context.Background(), src, "R", nil, nil)
		got := "null"
		if v != nil {
			got = valueString(v)
		}
		if err != nil || got != want {
			t.Errorf("%s = %s, %v; want %s", expr, got, err, want)
		}
	}
}

// TestTheCertainIntegersAreFollowedThrough covers how far an Integer is one for
// certain. A definition used twice is still one — A * A read the second A as a
// definition that refers to itself, and let 2500000000 through — and so are
// the type's extents, a list definition and a fluent call in lower case. A
// sort key's name is the column first, so a Long column named like a
// definition is not nulled, which had broken the sort.
func TestTheCertainIntegersAreFollowedThrough(t *testing.T) {
	for expr, want := range map[string]string{
		"A * A":                           "null",
		"Max24 + Max24":                   "null",
		"Sum({Max24, Max24})":             "null",
		"Sum(Edge)":                       "null",
		"(maximum Integer) + 1":           "null",
		"-(minimum Integer)":              "null",
		"Abs(minimum Integer)":            "null",
		"(-2147483648).abs()":             "null",
		"({2147483647, 1}).sum()":         "null",
		"Truncate(3000000000L)":           "null",
		"Floor(5)":                        "5",
		"Floor(3000000000L)":              "null",
		"Ceiling(3000000000L)":            "null",
		"Length('abc') + 2147483647":      "null",
		"Precision(1.58700) + 2147483647": "null",
		// A let or an alias of a definition's name is what the name means
		// there, and may be a Long.
		"({1}) Q let Max24: 3000000000L return Max24 + 1":                                             "{3000000001}",
		"({3000000000L}) Max24 return Max24 + 1":                                                      "{3000000001}",
		"({1}) Max24 return Max24 + 1":                                                                "{2}",
		"First((List<Tuple{X Long}>{Tuple{X: 3000000000L}, Tuple{X: 1L}}) T sort by X + 1).X":         "1",
		"Last((List<Tuple{X Long}>{Tuple{X: 3000000000L}, Tuple{X: 4000000000L}}) T sort by X * 1).X": "4000000000",
	} {
		src := "library T version '1.0'\ndefine A: 50000\ndefine X: 5\ndefine Max24: 2147483647\n" +
			"define Edge: {2147483647, 1}\ndefine R: " + expr + "\n"
		v, err := NewEngine().EvaluateExpression(context.Background(), src, "R", nil, nil)
		got := "null"
		if v != nil {
			got = valueString(v)
		}
		if err != nil || got != want {
			t.Errorf("%s = %s, %v; want %s", expr, got, err, want)
		}
	}
	for _, expr := range []string{"successor of maximum Integer", "predecessor of minimum Integer"} {
		_, err := NewEngine().EvaluateExpression(context.Background(),
			"library T version '1.0'\ndefine R: "+expr+"\n", "R", nil, nil)
		if !errors.Is(err, cqltypes.ErrPastRange) {
			t.Errorf("%s: err = %v; want ErrPastRange", expr, err)
		}
	}
}
