package cql

import (
	"context"
	"testing"

	"github.com/gofhir/cql/compiler"
	"github.com/gofhir/cql/elm"
	"github.com/gofhir/cql/eval"
)

// evalAtTheEdge evaluates one expression, and fails the test on an error: each
// case below is one the specification answers with a value or null, never a
// run-time error.
func evalAtTheEdge(t *testing.T, expr string) string {
	t.Helper()
	v, err := NewEngine().EvaluateExpression(context.Background(),
		"library T version '1.0'\ndefine R: "+expr+"\n", "R", nil, nil)
	if err != nil {
		t.Errorf("%s: %v; want a value or null", expr, err)
		return "error"
	}
	return valueString(v)
}

// TestAStepPastItsTypeIsNull covers successor and predecessor at the edge of
// every type that has them. CQL 1.5.3, Appendix B, Successor: "If the argument
// is already the maximum value for the type, a null is returned", and "If the
// result of the operation cannot be represented (i.e. would result in an
// overflow), the result is null"; Predecessor says the same of the minimum.
// The engine raised an error, as cql-tests 727219f asked of the temporal
// cases (invalid="true"); f724ee4 asks null, as the specification does (plan
// 013).
func TestAStepPastItsTypeIsNull(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"successor of 2147483647", "null"},
		{"predecessor of (-2147483648)", "null"},
		{"successor of maximum Integer", "null"},
		{"predecessor of minimum Integer", "null"},
		{"successor of 9223372036854775807L", "null"},
		{"successor of maximum Long", "null"},
		{"predecessor of minimum Long", "null"},
		{"successor of 2147483646", "2147483647"},
		{"successor of 2147483647L", "2147483648"},

		// Decimal: "maximum Decimal" is 99999999999999999999.99999999, and one
		// step past it cannot be represented.
		{"successor of maximum Decimal", "null"},
		{"predecessor of minimum Decimal", "null"},
		{"successor of 99999999999999999999.99999999", "null"},
		{"successor of 99999999999999999999.99999998", "99999999999999999999.99999999"},
		{"successor of 99999999999999999999.99999999 'mg'", "null"},
		{"predecessor of -99999999999999999999.99999999 'mg'", "null"},

		// Date, DateTime and Time: the calendar runs from year 1 to 9999, and
		// the clock from 00:00:00.000 to 23:59:59.999, at every precision.
		{"successor of @9999-12-31", "null"},
		{"successor of @9999-12", "null"},
		{"successor of @9999", "null"},
		{"predecessor of @0001-01-01", "null"},
		{"predecessor of @0001", "null"},
		{"successor of maximum Date", "null"},
		{"predecessor of minimum Date", "null"},
		{"successor of DateTime(9999, 12, 31, 23, 59, 59, 999)", "null"},
		{"successor of DateTime(9999)", "null"},
		{"predecessor of DateTime(0001, 1, 1, 0, 0, 0, 0)", "null"},
		{"predecessor of DateTime(1)", "null"},
		{"successor of DateTime(9999, 12, 31, 23, 59, 59, 999, 14.0)", "null"},
		{"predecessor of DateTime(0001, 1, 1, 0, 0, 0, 0, -12.0)", "null"},
		{"successor of maximum DateTime", "null"},
		{"predecessor of minimum DateTime", "null"},
		{"successor of @T23:59:59.999", "null"},
		{"successor of @T23:59", "null"},
		{"successor of @T23", "null"},
		{"predecessor of @T00:00:00.000", "null"},
		{"predecessor of @T00", "null"},
		{"successor of maximum Time", "null"},
		{"predecessor of minimum Time", "null"},
		{"successor of @9999-12-30", "9999-12-31"},
		{"predecessor of @T00:00:00.001", "00:00:00.000"},

		// A null, not an error, so the expression around it goes on.
		{"Coalesce(successor of maximum Integer, 1)", "1"},
		{"(successor of @T23:59:59.999) is null", "true"},
	} {
		if got := evalAtTheEdge(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestTheStartOfABoundaryAtTheEdgeIsNull covers the same step taken by Start
// and End. CQL 1.5.3, Appendix B, Start: "If the low boundary of the interval
// is open, this operator returns the successor of the low value of the
// interval"; End returns the predecessor of an open high. The successor of the
// maximum is null, so is the start of an interval open there (plan 013,
// decision C). An Integer answered 2147483648, outside its type.
func TestTheStartOfABoundaryAtTheEdgeIsNull(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"start of Interval(2147483647, null]", "null"},
		{"end of Interval[null, -2147483648)", "null"},
		{"start of Interval(maximum Long, null]", "null"},
		{"start of Interval(maximum Decimal, null]", "null"},
		{"end of Interval[null, minimum Decimal)", "null"},
		{"start of Interval(@9999-12-31, null]", "null"},
		{"start of Interval(@9999-12-31T23:59:59.999, null]", "null"},
		{"end of Interval[null, @0001-01-01T00:00:00.000)", "null"},
		{"start of Interval(@T23:59:59.999, null]", "null"},
		{"end of Interval[null, @T00:00:00.000)", "null"},
		{"start of Interval(2147483646, null]", "2147483647"},
		{"end of Interval[null, @T00:00:00.001)", "00:00:00.000"},
	} {
		if got := evalAtTheEdge(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestTheStartOfANamedIntervalAtTheEdgeIsNull covers the 32-bit edge of an
// interval held by a definition of the library, which is an Integer's for
// certain as one written in place is (review 1 of plan 013: it answered
// 2147483648). An operand of a function is not: a Long can stand behind it,
// so there the 64-bit edge is all there is, a known loss of plan 013.
func TestTheStartOfANamedIntervalAtTheEdgeIsNull(t *testing.T) {
	for _, tt := range []struct{ defs, expr, want string }{
		{"define I: Interval(2147483647, null]\n", "start of I", "null"},
		{"define I: Interval[null, -2147483648)\n", "end of I", "null"},
		{"define I: Interval(2147483647, null]\ndefine J: I\n", "start of J", "null"},
		{"define I: Interval(2147483646, null]\n", "start of I", "2147483647"},
		{"define function F(i Interval<Integer>): start of i\n", "F(Interval(2147483647, null])", "2147483648"},
		// A definition may be named as the walk names what it settles about
		// another: "low of I" is the name of a definition here, not the low
		// boundary of I (review 2 of plan 013).
		{"define I: Interval[2147483647, null]\ndefine \"low of I\": start of I\n", "successor of \"low of I\"", "null"},
		{"define I: Interval[null, -2147483648]\ndefine \"high of I\": end of I\n", "predecessor of \"high of I\"", "null"},
		{"define L: {2147483647}\ndefine \"list L\": Max(L)\n", "successor of \"list L\"", "null"},
	} {
		v, err := NewEngine().EvaluateExpression(context.Background(),
			"library T version '1.0'\n"+tt.defs+"define R: "+tt.expr+"\n", "R", nil, nil)
		if err != nil {
			t.Errorf("%s: %v; want %s", tt.expr, err, tt.want)
			continue
		}
		if got := valueString(v); got != tt.want {
			t.Errorf("%s, with %q, = %s; want %s", tt.expr, tt.defs, got, tt.want)
		}
	}
}

// TestAStepThatIsNoStepStaysAnError holds the other errors of a step apart from
// the edge: only a step past the range of its type is null.
func TestAStepThatIsNoStepStaysAnError(t *testing.T) {
	for _, expr := range []string{"successor of 'a'", "predecessor of true"} {
		if _, err := NewEngine().EvaluateExpression(context.Background(),
			"library T version '1.0'\ndefine R: "+expr+"\n", "R", nil, nil); err == nil {
			t.Errorf("%s evaluated; want an error", expr)
		}
	}
}

// TestMeetsAtTheEdgeOfAType covers meets, which steps a boundary on its own and
// must not reach the null of the operators above. A coincident pair does not
// meet: CQL 1.5.3, Appendix B, Meets, requires the end of one to be the
// predecessor of the start of the other. At the last value of a type, with no
// successor to compare, two coincident intervals met, for Date, DateTime,
// Time and Long (plan 013); and so would they at the largest Decimal, had its
// step become an error. Collapse, which rests on meets, still joins two
// intervals that meet at the edge.
func TestMeetsAtTheEdgeOfAType(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"Interval[maximum Decimal, maximum Decimal] meets before Interval[maximum Decimal, maximum Decimal]", "false"},
		{"Interval[maximum Decimal, maximum Decimal] meets Interval[maximum Decimal, maximum Decimal]", "false"},
		{"Interval[99999999999999999999.99999998, 99999999999999999999.99999998] meets before Interval[maximum Decimal, maximum Decimal]", "true"},
		{"Interval[maximum Integer, maximum Integer] meets before Interval[maximum Integer, maximum Integer]", "false"},
		{"Interval[maximum Long, maximum Long] meets before Interval[maximum Long, maximum Long]", "false"},
		{"Interval[@T23:59:59.999, @T23:59:59.999] meets before Interval[@T23:59:59.999, @T23:59:59.999]", "false"},
		{"Interval[@9999-12-31, @9999-12-31] meets before Interval[@9999-12-31, @9999-12-31]", "false"},
		{"Interval[99999999999999999999.99999999 'g', 99999999999999999999.99999999 'g'] meets before Interval[99999999999999999999.99999999 'g', 99999999999999999999.99999999 'g']", "false"},
		{"collapse { Interval[0.0, 99999999999999999999.99999998], Interval[maximum Decimal, maximum Decimal] }",
			"{Interval[0.0, 99999999999999999999.99999999]}"},
		// A type without a successor is read by equality, as it was before
		// plan 013: the specification defines meets by the successor, and
		// String has none. This holds main's reading, not the specification's.
		{"Interval['a', 'a'] meets before Interval['a', 'a']", "true"},
	} {
		if got := evalAtTheEdge(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestWhatCannotBeRepresentedIsNull covers Exp, Ln and the division of a
// Quantity by zero. CQL 1.5.3, Appendix B: "In general, operations that cause
// arithmetic overflow or underflow, or otherwise cannot be performed (such as
// division by 0) will result in null, rather than a run-time error"; Exp and
// Ln: "If the result of the operation cannot be represented, the result is
// null"; Divide: "If the result of the operation cannot be represented, or the
// right argument is 0, the result is null". The largest Decimal is
// 99999999999999999999.99999999 (plan 013, decision B).
func TestWhatCannotBeRepresentedIsNull(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"Exp(1000)", "null"},
		{"Exp(1000.0)", "null"},
		{"Exp(709)", "null"},
		{"Exp(100)", "null"},
		{"Exp(46.06)", "null"},
		{"Exp(-1000)", "0"},
		{"Exp(0)", "1"},
		{"Exp(1000) is null", "true"},
		{"Ln(0)", "null"},
		{"Ln(0.0)", "null"},
		{"Ln(-0)", "null"},
		{"Ln(-0.0)", "null"},
		{"Ln(-1)", "null"},
		{"Ln(1)", "0"},
		{"Coalesce(Ln(0), 7)", "7"},
		{"Ln(0) + 1", "null"},
		{"5.0 'g' / 0", "null"},
		{"5.0 'g' / 0.0", "null"},
		{"5 'g' / 0", "null"},
		{"(5.0 'g' / 0) + 1 'g'", "null"},
		{"5.0 'g' / 2", "2.5 'g'"},
	} {
		if got := evalAtTheEdge(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestExpBelowTheLargestDecimalIsAValue holds the other side of the edge: e to
// 46.05 is 99829958746143620000-odd, below the largest Decimal, and a value.
func TestExpBelowTheLargestDecimalIsAValue(t *testing.T) {
	for _, expr := range []string{"Exp(46.05)", "Exp(46)", "Exp(-46.05)"} {
		if got := evalAtTheEdge(t, expr); got == "null" || got == "error" {
			t.Errorf("%s = %s, want a value", expr, got)
		}
	}
}

// TestAnImportedSuccessorAtTheEdgeIsNull covers the ELM path: Successor read
// from ELM reaches the same operator. Its 32-bit edge does not, as an imported
// library has no Plan to say an operand is an Integer for certain
// (eval/integer_range.go, pastInteger): the successor of the largest Integer
// is 2147483648 there, a known loss of plan 013 that this records.
func TestAnImportedSuccessorAtTheEdgeIsNull(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"successor of @T23:59:59.999", "null"},
		{"predecessor of @0001-01-01", "null"},
		{"successor of 9223372036854775807L", "null"},
		{"successor of maximum Decimal", "null"},
		{"start of Interval(@T23:59:59.999, null]", "null"},
		{"successor of 2147483647", "2147483648"},
	} {
		parsed, err := compiler.Compile("library T version '1.0'\ndefine X: " + tt.expr + "\n")
		if err != nil {
			t.Fatal(err)
		}
		raw, err := elm.MarshalLibrary(elm.Translate(parsed))
		if err != nil {
			t.Fatal(err)
		}
		read, err := elm.UnmarshalLibrary(raw)
		if err != nil {
			t.Fatal(err)
		}
		imported, err := elm.Import(read)
		if err != nil {
			t.Fatal(err)
		}
		got, err := eval.NewEvaluator(eval.NewContext(context.Background(), imported)).EvaluateExpression("X")
		if err != nil {
			t.Errorf("%s from ELM: %v; want %s", tt.expr, err, tt.want)
			continue
		}
		if s := valueString(got); s != tt.want {
			t.Errorf("%s from ELM = %s, want %s", tt.expr, s, tt.want)
		}
	}
}
