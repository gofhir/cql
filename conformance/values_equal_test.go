package conformance

import (
	"testing"

	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
)

// TestAnUntypedOutputIsReadAsDecimals covers the cases whose expected output
// writes Decimal points as whole numbers: an answer of Decimals passes, and one
// that mixes in an Integer does not, which comparing by value let through.
func TestAnUntypedOutputIsReadAsDecimals(t *testing.T) {
	equal := equalFor(TestCase{Name: "ExpandPer1IntervalOverload"})
	number := func(s string) fptypes.Value {
		d, err := fptypes.NewDecimal(s)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	want := cqltypes.NewList(fptypes.Collection{fptypes.NewInteger(10), fptypes.NewInteger(11)})
	if !equal(cqltypes.NewList(fptypes.Collection{number("10.0"), number("11.0")}), want) {
		t.Error("Decimal points failed an output written as whole numbers")
	}
	if equal(cqltypes.NewList(fptypes.Collection{number("10.0"), fptypes.NewInteger(11)}), want) {
		t.Error("an Integer among the Decimal points passed")
	}
}

// TestValuesEqualTellsAnIntegerFromADecimal holds the suite's comparison to the
// type of its numbers, at the top and inside lists, intervals and tuples: CQL's
// equality says the Integer 3 equals the Decimal 3.0, and through it `10.1 div
// 3.1` answering an Integer passed a suite that expects 3.0.
func TestValuesEqualTellsAnIntegerFromADecimal(t *testing.T) {
	integer := fptypes.NewInteger(3)
	dec, err := fptypes.NewDecimal("3.0")
	if err != nil {
		t.Fatal(err)
	}
	list := func(v fptypes.Value) fptypes.Value { return cqltypes.NewList(fptypes.Collection{v}) }
	interval := func(v fptypes.Value) fptypes.Value {
		return cqltypes.Interval{Low: v, High: v, LowClosed: true, HighClosed: true}
	}
	tuple := func(v fptypes.Value) fptypes.Value {
		return cqltypes.NewTuple(map[string]fptypes.Value{"a": v})
	}
	for name, wrap := range map[string]func(fptypes.Value) fptypes.Value{
		"a number":      func(v fptypes.Value) fptypes.Value { return v },
		"a list":        list,
		"a nested list": func(v fptypes.Value) fptypes.Value { return list(list(v)) },
		"an interval":   interval,
		"a tuple":       tuple,
	} {
		if valuesEqual(wrap(integer), wrap(dec)) {
			t.Errorf("%s: an Integer answered for a Decimal passed", name)
		}
		if !valuesEqual(wrap(dec), wrap(dec)) {
			t.Errorf("%s: a Decimal answered for itself failed", name)
		}
	}
}
