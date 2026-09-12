package cql

import (
	"context"
	"regexp"
	"strings"
	"testing"
)

func evalDefaultUnit(t *testing.T, expr string) string {
	t.Helper()
	src := "library T version '1.0'\ndefine X: " + expr + "\n"
	got, err := NewEngine().EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	if got == nil {
		return "null"
	}
	return got.String()
}

// bareNumberSpellings are the three types a bare number can have in CQL. The
// literal an author writes is the first of them, and it was the one the rule
// missed: the half of this that existed asked whether the value was a Decimal.
var bareNumberSpellings = []string{"150", "150L", "150.0"}

// TestEveryOperatorReadsABareNumberAsAQuantityOfUnitOne is the rule, across the
// operators that put a bare number next to a quantity.
//
// CQL states it where it defines addition — "when a quantity has no units
// specified, it is treated as a quantity with the default unit ('1')" — but it is
// a fact about what the number *is*, so every operator owes it the same reading.
// Arithmetic had it. Comparison had half of it. The rest had none:
//
//	150.0 >= 100 '1'   true       the Decimal spelling of the operand
//	150   >= 100 '1'   error      the same question, written as authors write it
//	150   =  150 '1'   false      equality, which nobody promoted
//	distinct {150, 150 '1'}       kept both, so one value counted as two
//
// The last of those is the shape of the damage: a silently wrong answer rather
// than a refusal. `IndexOf` said -1 for a value the list held, and `in` said
// false, so anything selecting on a quantity-valued element could drop it.
func TestEveryOperatorReadsABareNumberAsAQuantityOfUnitOne(t *testing.T) {
	// Written with the operand's spelling substituted, so all three numeric types
	// go through every operator rather than one type going through all of them.
	for _, tt := range []struct{ expr, want string }{
		{"N >= 100 '1'", "true"},
		{"N > 150 '1'", "false"},
		{"N <= 200 '1'", "true"},
		{"100 '1' < N", "true"},
		{"N = 150 '1'", "true"},
		{"N != 150 '1'", "false"},
		{"N ~ 150 '1'", "true"},
		{"N in Interval[100 '1', 200 '1']", "true"},
		{"Interval[100 '1', 200 '1'] contains N", "true"},
		{"N between 100 '1' and 200 '1'", "true"},
		{"N in {150 '1'}", "true"},
		{"{150 '1'} contains N", "true"},
		{"IndexOf({150 '1'}, N)", "0"},
		{"Count(distinct {N, 150 '1'})", "1"},
		{"Count({N} intersect {150 '1'})", "1"},
		{"Count({N} except {150 '1'})", "0"},
		{"Min({N, 100 '1'})", "100 '1'"},
		{"Max({N, 100 '1'})", "150"},
		// Arithmetic already had the rule and keeps it.
		{"N + 100 '1'", "250 '1'"},
	} {
		for _, spelling := range bareNumberSpellings {
			expr := strings.ReplaceAll(tt.expr, "N", spelling)
			want := tt.want
			// Max returns the element itself, so the answer is however that element
			// renders: a Long prints the way an Integer does, a Decimal keeps its
			// point.
			if strings.Contains(tt.expr, "Max(") {
				want = strings.TrimSuffix(spelling, "L")
			}
			if strings.HasPrefix(tt.expr, "N +") && spelling == "150.0" {
				want = "250.0 '1'"
			}
			if got := evalDefaultUnit(t, expr); got != want {
				t.Errorf("%s = %s, want %s", expr, got, want)
			}
		}
	}
}

// TestAContainerIsNoSurerThanTheValuesItHolds carries the rule into list, tuple,
// interval and ratio equality, which decide it through one shared function.
//
// Leaving them out made a container *less* sure than its elements rather than
// more: `150 in {150 '1'}` answered true while `{150} = {150 '1'}` answered
// false. Equivalence had already been carried over by the element rule in types,
// so the two halves of the same question disagreed with each other as well.
func TestAContainerIsNoSurerThanTheValuesItHolds(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"{150} = {150 '1'}", "true"},
		{"{150} ~ {150 '1'}", "true"},
		{"Tuple{a: 150} = Tuple{a: 150 '1'}", "true"},
		{"Tuple{a: 150} ~ Tuple{a: 150 '1'}", "true"},
		{"Interval[100, 200] = Interval[100 '1', 200 '1']", "true"},
		// And the undecidable pair stays undecidable one level up, which is the
		// property this shared function was written for.
		{"{150} = {150 'mg'}", "null"},
		{"{150} != {150 'mg'}", "null"},
		{"Interval[100, 200] = Interval[100 'mg', 200 'mg']", "null"},
		// The pair that rule was originally about is untouched.
		{"{1 'cm2'} = {1 'cm'}", "null"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestADifferentDimensionIsStillUndecidable is the limit on the rule above, and
// the half that keeps it from being a permit to compare anything.
//
// A bare number is dimensionless, not unitless. So against 'mg' it is two
// dimensions with no unit both can be stated in, which this engine has answered
// null since it learned that quantities of different dimensions do not compare —
// the same answer `1 'cm' < 1 's'` gives. Reading the default unit as "whatever
// the other side has" would have turned every one of these into a confident
// wrong answer instead.
func TestADifferentDimensionIsStillUndecidable(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"150 >= 100 'mg'", "null"},
		{"150 < 100 'mg'", "null"},
		{"150 = 150 'mg'", "null"},
		{"150 != 150 'mg'", "null"},
		{"150 between 100 'mg' and 200 'mg'", "null"},
		{"150 in Interval[100 'mg', 200 'mg']", "null"},
		// Equivalence never yields null in CQL, so it answers false rather than
		// declining — the same split this engine already draws for temporals.
		{"150 ~ 150 'mg'", "false"},
		// A container has no null to hold, so it answers "not the same value".
		{"150 in {150 'mg'}", "false"},
		{"IndexOf({150 'mg'}, 150)", "-1"},
		{"Count(distinct {150, 150 'mg'})", "2"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestTheDefaultUnitDoesNotReachOperatorsThatOnlyRenderTheirOperands is the other
// limit, and the reason the rule is applied per operator rather than to every
// pair of operands on the way in.
//
// Concatenation renders what it is given. Promoting there would have made
// `150 + ' mg'` produce "150 '1' mg", which is a new wrong answer introduced by
// a fix for wrong answers.
func TestTheDefaultUnitDoesNotReachOperatorsThatOnlyRenderTheirOperands(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"'reading: ' + ToString(150)", "reading: 150"},
		{"ToString(150)", "150"},
		{"'a' + 'b'", "ab"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
	// And a bare number on its own is still a bare number: nothing about this rule
	// makes an engine that was asked for 150 answer with a quantity.
	for _, spelling := range bareNumberSpellings {
		if got := evalDefaultUnit(t, spelling); strings.Contains(got, "'1'") {
			t.Errorf("%s on its own evaluated to %s — the default unit is for pairing, not for storing", spelling, got)
		}
	}
}

// TestTheRuleReachesFHIRData is the measurement that matters, because a literal
// is not what a measure compares.
//
// `value as FHIR.Quantity` is the spelling the reference translator emits for a
// choice element in a comparison, so it is what published CQL actually contains.
// Against a bare number that spelling was giving a silently wrong answer:
//
//	(O.value as FHIR.Quantity) = 150     was false, with the reading at 150 '1'
//	(O.value as FHIR.Quantity) >= 100    was an error
//
// The null rows are a different rule and are not affected by this one: an
// unqualified `O.value` against a bare number names the FHIR.integer branch, and
// a Quantity is on another branch. That is the choice-narrowing this engine
// already does, and the last two rows are the same question where the branch does
// match.
func TestTheRuleReachesFHIRData(t *testing.T) {
	const dimensionless = `{"value":150,"unit":"1"}`
	const milligrams = `{"value":150,"unit":"mg"}`
	for _, tt := range []struct{ field, value, expr, want string }{
		{"valueQuantity", dimensionless, "(First([Observation] O).value as FHIR.Quantity) = 150", "true"},
		{"valueQuantity", dimensionless, "(First([Observation] O).value as FHIR.Quantity) >= 100", "true"},
		{"valueQuantity", dimensionless, "(First([Observation] O).value as FHIR.Quantity) in Interval[100, 200]", "true"},
		// A different dimension is undecidable here as everywhere.
		{"valueQuantity", milligrams, "(First([Observation] O).value as FHIR.Quantity) = 150", "null"},
		{"valueQuantity", milligrams, "(First([Observation] O).value as FHIR.Quantity) >= 100", "null"},
		// And the ordinary case a measure depends on keeps answering.
		{"valueQuantity", milligrams, "First([Observation] O).value >= 100 'mg'", "true"},
		{"valueInteger", `150`, "First([Observation] O).value = 150", "true"},
	} {
		if got := evalOnBranch(t, tt.field, tt.value, tt.expr); got != tt.want {
			t.Errorf("%s over %s = %s, want %s", tt.expr, tt.value, got, tt.want)
		}
	}
}

// TestContainersAgreeHoweverDeeplyTheyNest covers that the two functions which
// decide whether two held values are the same — elementEquality in eval and
// valuesEqual in types — reach the same answer, at every depth.
//
// They are two on purpose: one reports a three-way verdict so `=` can answer
// null, the other a boolean for the containers that have no null to return. Two
// functions carrying one rule is exactly the shape that has drifted in this
// repository before, so the agreement is asserted rather than assumed.
func TestContainersAgreeHoweverDeeplyTheyNest(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"{{150}} = {{150 '1'}}", "true"},
		{"{150} in {{150 '1'}}", "true"},
		{"Count(distinct {{150}, {150 '1'}})", "1"},
		{"Tuple{a: {150}} = Tuple{a: {150 '1'}}", "true"},
		// And the split the two functions exist for survives the nesting: `=`
		// declines, membership answers "not the same value".
		{"{{1 'cm'}} = {{1 's'}}", "null"},
		{"{1 'cm'} in {{1 's'}}", "false"},
		{"Count(distinct {{1 'cm'}, {1 's'}})", "2"},
		{"IndexOf({{1 's'}}, {1 'cm'})", "-1"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestOverlapsAnswersTheSameInEitherOrder covers a defect this change uncovered
// rather than caused, and would have shipped as a silently wrong answer.
//
// `overlaps` is symmetric, and it was not:
//
//	Interval[100, 200] overlaps Interval[150 '1', 250 '1']   false
//	Interval[100 '1', 200 '1'] overlaps Interval[150, 250]   true
//
// The cause is the adjustment for open integer boundaries, which needs integers
// on both sides and checked only one. The other side, read through a helper that
// answered 0 for anything that is not an integer, came back as the boundary 0 —
// so 100 looked to be past the end of the second interval. Which of the two
// orders is wrong depends only on which side happened to be the integer.
//
// It was unreachable while a bare number against a quantity raised an error one
// line earlier. Teaching comparison that pair turned the error into an answer,
// and the answer was wrong, so the helper now reports whether it could read the
// boundary at all.
func TestOverlapsAnswersTheSameInEitherOrder(t *testing.T) {
	for _, tt := range []struct{ a, b, want string }{
		{"Interval[100, 200]", "Interval[150 '1', 250 '1']", "true"},
		{"Interval[100, 200]", "Interval[250 '1', 350 '1']", "false"},
		// Open boundaries are what the adjustment is for, so they are the rows
		// that must keep working once it is guarded.
		{"Interval[100, 150)", "Interval[150, 250]", "false"},
		{"Interval[100, 150]", "Interval[150, 250]", "true"},
		{"Interval(100, 150]", "Interval[100, 100]", "false"},
		{"Interval[100, 150]", "Interval[100, 100]", "true"},
	} {
		forward := evalDefaultUnit(t, tt.a+" overlaps "+tt.b)
		backward := evalDefaultUnit(t, tt.b+" overlaps "+tt.a)
		if forward != backward {
			t.Errorf("%s overlaps %s = %s, but the other way round = %s — overlaps is symmetric",
				tt.a, tt.b, forward, backward)
		}
		if forward != tt.want {
			t.Errorf("%s overlaps %s = %s, want %s", tt.a, tt.b, forward, tt.want)
		}
	}
}

// TestTheRuleReachesTheOperatorsThatOnlyBorrowTheComparison lists what came along
// without being touched, because every one of them reaches CompareTemporal.
//
// They are asserted so that the single point stays single: if an ordering ever
// stops passing through there, these fail rather than quietly going back to
// raising errors.
func TestTheRuleReachesTheOperatorsThatOnlyBorrowTheComparison(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		// A sort key, which reaches it through compareSortKeys.
		{"First(({ Tuple{k: 150}, Tuple{k: 100 '1'} }) T sort by k).k", "100 '1'"},
		{"Last(({ Tuple{k: 150}, Tuple{k: 100 '1'} }) T sort by k).k", "150"},
		// The interval operators in funcs.
		{"Interval[100 '1', 200 '1'] includes Interval[120, 180]", "true"},
		{"Interval[100 '1', 200 '1'] starts Interval[100, 300]", "true"},
		{"Interval[100, 200] includes 150 '1'", "true"},
		// `start of` against a bare number, which was false rather than an error —
		// a wrong answer already, before any of this.
		{"start of Interval[100 '1', 200 '1'] = 100", "true"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestMixedBoundIntervalsWork covers the space the default-unit rule opens up.
//
// An interval with a bare number at one end and a quantity at the other was
// unusable: every question about it raised an error. It now answers, and agrees
// with the interval whose bounds are both written out.
//
// `width` and `Size` were the exception and were asserted here as failing, with
// a message naming the subtraction they had to agree with. They agree with it
// now; see TestWidthIsTheSubtractionItIsDefinedAs for what that took.
func TestMixedBoundIntervalsWork(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"Interval[100, 200 '1'] contains 150", "true"},
		{"Interval(100, 200 '1') contains 100", "false"},
		{"start of Interval[100, 200 '1']", "100"},
		{"end of Interval[100, 200 '1']", "200 '1'"},
		{"Interval[100, 200 '1'] = Interval[100 '1', 200 '1']", "true"},
		{"width of Interval[100, 200 '1']", "100 '1'"},
		{"Size(Interval[100, 200 '1'])", "100 '1'"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestWidthIsTheSubtractionItIsDefinedAs covers what asking the subtraction
// instead of repeating it fixed, which was more than the null it was reached
// through.
//
// IntervalWidth carried its own arithmetic: it subtracted the two magnitudes and
// kept the low bound's unit. Any interval written in two scales of one dimension
// therefore came out scaled by the ratio between them — and negative whenever the
// high bound's unit was the larger one, which is not a width any interval has:
//
//	width of Interval[100 'cm', 2 'm']   was -98 'cm'    2 'm' - 100 'cm' is 1 'm'
//	width of Interval[500 'mg', 1 'g']   was -499 'mg'   and 0.5 'g'
//	width of Interval[1 'cm', 1 's']     was 0 'cm'      and null
//	width of Interval[1, 2.5]            was null        and 1.5
//
// Every row is checked against that subtraction rather than against a written-out
// number, because agreeing with it is the property: width has no separate
// definition to be right about on its own.
func TestWidthIsTheSubtractionItIsDefinedAs(t *testing.T) {
	for _, tt := range []struct{ low, high string }{
		{"100 'cm'", "2 'm'"},
		{"500 'mg'", "1 'g'"},
		{"1 'g'", "2 'g'"},
		{"1 'cm'", "1 's'"},
		{"1", "2.5"},
		{"100", "200 '1'"},
		{"1.5", "2.5"},
	} {
		width := evalDefaultUnit(t, "width of Interval["+tt.low+", "+tt.high+"]")
		subtraction := evalDefaultUnit(t, tt.high+" - "+tt.low)
		if width != subtraction {
			t.Errorf("width of Interval[%s, %s] = %s, but %s - %s = %s — width is that subtraction",
				tt.low, tt.high, width, tt.high, tt.low, subtraction)
		}
		// A width is never negative: an interval whose high bound is below its low
		// bound is empty, not backwards.
		if strings.HasPrefix(width, "-") {
			t.Errorf("width of Interval[%s, %s] = %s, and no interval has a negative width",
				tt.low, tt.high, width)
		}
	}

	// Integers keep the discrete reading, where an open boundary moves in by one
	// before the subtraction rather than after it — so these do *not* equal the
	// plain subtraction, and that is the one deliberate exception.
	for _, tt := range []struct{ expr, want string }{
		{"width of Interval[1, 5]", "4"},
		{"width of Interval(1, 5)", "2"},
		{"Size(Interval[1, 5])", "5"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}

	// A continuous interval has no such adjustment: between two quantities or two
	// decimals there is no next value to move to, so open and closed are the same
	// width. Asserted because the integer case above is an exception, and an
	// exception is only safe while what it is an exception *to* is pinned.
	for _, pair := range [][2]string{{"100 'cm'", "200 'cm'"}, {"1.0", "2.0"}} {
		closed := evalDefaultUnit(t, "width of Interval["+pair[0]+", "+pair[1]+"]")
		open := evalDefaultUnit(t, "width of Interval("+pair[0]+", "+pair[1]+")")
		if closed != open {
			t.Errorf("width of [%s, %s] = %s but of (%s, %s) = %s — a continuous interval has no step to move by",
				pair[0], pair[1], closed, pair[0], pair[1], open)
		}
	}

	// What this did not touch. Temporal intervals still refuse, rather than
	// falling into the numeric path now that the pair is promoted first.
	for _, expr := range []string{
		"width of Interval[@2020-01-01, @2020-06-01]",
		"width of Interval[@2020-01-01T00:00:00, @2020-06-01T00:00:00]",
		"width of Interval[@T10:00:00, @T12:00:00]",
	} {
		if got := evalDefaultUnit(t, expr); !strings.Contains(got, "width is not defined") {
			t.Errorf("%s = %s, want the refusal it has always given", expr, got)
		}
	}

	// And a calendar duration against its UCUM code is the pair CQL declines to
	// decide, which Subtract reports and this reads as null rather than inventing
	// a width for it.
	if got := evalDefaultUnit(t, "width of Interval[1 'a', 2 'year']"); got != "null" {
		t.Errorf("width over a calendar/UCUM pair = %s, want null", got)
	}
}

// TestSizeFollowsWidth checks the function that inherits this one, because Size
// is width plus the point size and only the width half was rewritten.
func TestSizeFollowsWidth(t *testing.T) {
	for _, tt := range []struct{ interval, width, size string }{
		// Continuous: a size is the width, there being no points to count.
		{"Interval[100 'cm', 2 'm']", "1 'm'", "1 'm'"},
		{"Interval[1.0, 2.0]", "1", "1"},
		{"Interval[1 'cm', 1 's']", "null", "null"},
		// Discrete: a size counts the points, so it is the width plus one.
		{"Interval[1, 5]", "4", "5"},
	} {
		if got := evalDefaultUnit(t, "width of "+tt.interval); got != tt.width {
			t.Errorf("width of %s = %s, want %s", tt.interval, got, tt.width)
		}
		if got := evalDefaultUnit(t, "Size("+tt.interval+")"); got != tt.size {
			t.Errorf("Size(%s) = %s, want %s", tt.interval, got, tt.size)
		}
	}
}

// TestExpandOverQuantitiesMatchesTheIntegerSpelling covers expansion over
// quantities, which had no case at all.
//
// The ladder in both expand functions ran Integer, Decimal, DateTime, Date, Time
// and then fell off the end, so every quantity interval expanded to nothing, in
// silence — while the same shape written in integers gave three intervals. This
// test was here asserting that emptiness, with a message naming the integer
// spelling to check against; it broke when the case was added, which is what it
// was for.
//
// Each row is checked against that integer spelling rather than against a written
// out list, because matching it is the property: a quantity interval is the same
// expansion wearing a unit.
func TestExpandOverQuantitiesMatchesTheIntegerSpelling(t *testing.T) {
	// The step written in the interval's unit, written without a unit, and left
	// out entirely — all three are one step of what the interval is measured in.
	number := regexp.MustCompile(`\d+`)
	for _, per := range []string{" per 1 'cm'", " per 1", ""} {
		quantities := evalDefaultUnit(t, "expand {Interval[1 'cm', 3 'cm']}"+per)
		integers := evalDefaultUnit(t, "expand {Interval[1, 3]}"+strings.ReplaceAll(per, " 'cm'", ""))
		// The integer expansion with a unit written on every bound *is* the
		// expected answer, derived rather than typed out.
		want := number.ReplaceAllString(integers, "$0 'cm'")
		if quantities != want {
			t.Errorf("expand over quantities%s = %s, but the integer spelling gives %s, which in centimeters is %s",
				per, quantities, integers, want)
		}
		if integers == "{}" {
			t.Fatalf("the integer spelling is empty too, so this test is measuring against nothing")
		}
	}

	// The single-interval overload expands to points rather than unit intervals,
	// and takes the same route, so the two cannot come to disagree about what a
	// quantity step means.
	if got := evalDefaultUnit(t, "expand Interval[1 'cm', 3 'cm'] per 1 'cm'"); got != "{1 'cm', 2 'cm', 3 'cm'}" {
		t.Errorf("the single-interval overload = %s, want {1 'cm', 2 'cm', 3 'cm'}", got)
	}

	// A step in another scale of the same dimension converts, which is the half a
	// separate quantity path would most likely have got wrong.
	if got := evalDefaultUnit(t, "expand {Interval[0 'm', 2 'm']} per 50 'cm'"); !strings.HasPrefix(got, "{Interval[0 'm', 0.4 'm']") {
		t.Errorf("a step in centimeters over an interval in meters = %s", got)
	}

	// A step in another dimension converts to nothing, and the empty list is then
	// an answer rather than the accident it used to be.
	if got := evalDefaultUnit(t, "expand {Interval[1 'cm', 3 'cm']} per 1 's'"); got != "{}" {
		t.Errorf("a step in seconds over an interval in centimeters = %s, want {}", got)
	}

	// The neighbor that always handled quantities is unchanged.
	if got := evalDefaultUnit(t, "collapse {Interval[1 'cm', 2 'cm'], Interval[2 'cm', 3 'cm']}"); got != "{Interval[1 'cm', 3 'cm']}" {
		t.Errorf("collapse over quantities = %s, want {Interval[1 'cm', 3 'cm']}", got)
	}
}
