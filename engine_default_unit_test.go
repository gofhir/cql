package cql

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
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

// TestExpandDeclinesThePairCQLWillNotDecide is the limit review found on the
// expansion above.
//
// A calendar duration against its UCUM code is the one pair CQL declines to
// settle, and every other operator over it answers null:
//
//	1 'year' = 1 'a'                     null
//	3 'a' - 1 'year'                     null
//	width of Interval[1 'year', 3 'a']   null
//
// fptypes.ConvertTo is happy to turn one into the other, so reducing a quantity
// interval through it made expand the only operator that treated the pair as
// settled — it expanded where the rest decline. The reduction now asks the same
// question the equality path asks, and the units that are genuinely one dimension
// still expand.
func TestExpandDeclinesThePairCQLWillNotDecide(t *testing.T) {
	for _, expr := range []string{
		"expand {Interval[1 'year', 3 'a']} per 1 'year'",
		"expand {Interval[1 'year', 3 'year']} per 1 'a'",
	} {
		if got := evalDefaultUnit(t, expr); got != "{}" {
			t.Errorf("%s = %s, want {} — CQL does not decide a calendar duration against its UCUM code",
				expr, got)
		}
	}
	// Either unit on its own is a dimension like any other and still expands.
	for _, expr := range []string{
		"expand {Interval[1 'year', 3 'year']} per 1 'year'",
		"expand {Interval[1 'a', 3 'a']} per 1 'a'",
	} {
		if got := evalDefaultUnit(t, expr); got == "{}" {
			t.Errorf("%s = {} — declining the mixed pair must not stop a single unit from expanding", expr)
		}
	}
}

// TestANegativeStepExpandsNothing settles what the four expansion paths used to
// answer four different ways.
//
// An expansion starts at the low bound and walks toward the high one, so a step
// pointing the other way names no sequence. Each path had invented its own
// answer:
//
//	expand {Interval[1, 3]} per -1          normalized the step to 1
//	expand {Interval[1.0, 3.0]} per -1.0    10001 intervals, down to -10000
//	expand {…@2018-01-04]} per -1 day       10001 of them, each one backwards
//	expand {Interval[1 'cm', 3 'cm']}…      the empty list
//
// The middle two are why this is not left alone: every value they produced lay
// outside the interval asked about, and the temporal one reported intervals whose
// high bound preceded their low. Ten thousand of each, stopped only by the
// runaway guard.
//
// The empty list is now shared by all four. It is the one answer that invents
// nothing — normalizing to 1 reads `per -1` as `per 1`, which is a guess at what
// the author meant — and the conformance corpus has no case for a negative step
// to prefer one reading over the other. Its twenty-odd expand cases all still
// pass.
func TestANegativeStepExpandsNothing(t *testing.T) {
	for _, interval := range []string{
		"Interval[1, 3]",
		"Interval[1.0, 3.0]",
		"Interval[1 'cm', 3 'cm']",
		"Interval[@2018-01-01, @2018-01-04]",
		"Interval[@T10:00, @T12:30]",
	} {
		var per string
		switch {
		case strings.Contains(interval, "@T"):
			per = " per -1 hour"
		case strings.Contains(interval, "@"):
			per = " per -1 day"
		case strings.Contains(interval, "'cm'"):
			per = " per -1 'cm'"
		case strings.Contains(interval, "."):
			per = " per -1.0"
		default:
			per = " per -1"
		}
		// Both overloads: the list one answers with unit intervals, the single one
		// with points, and neither has a sequence to report.
		if got := evalDefaultUnit(t, "expand {"+interval+"}"+per); got != "{}" {
			t.Errorf("expand {%s}%s = %s, want {}", interval, per, got)
		}
		if got := evalDefaultUnit(t, "expand "+interval+per); got != "{}" {
			t.Errorf("expand %s%s = %s, want {}", interval, per, got)
		}
	}

	// However the step is spelled. The temporal paths take a unit keyword or a
	// UCUM code, and a fractional or tiny step is negative just the same.
	for _, expr := range []string{
		"expand {Interval[@2018-01-01, @2018-01-04]} per -1 'd'",
		"expand {Interval[@T10:00, @T12:30]} per -1 hour",
		"expand {Interval[1, 3]} per -0.5",
		"expand {Interval[1, 3]} per -0.0000001",
	} {
		if got := evalDefaultUnit(t, expr); got != "{}" {
			t.Errorf("%s = %s, want {}", expr, got)
		}
	}

	// A zero step is a different question and keeps its answer: all four already
	// read it as "no step given", which is the unit interval of the point type.
	// Negative zero is zero, not negative, and belongs on this side of the line.
	for _, expr := range []string{
		"expand {Interval[1, 3]} per 0",
		"expand {Interval[1.0, 3.0]} per 0.0",
		"expand {Interval[1 'cm', 3 'cm']} per 0 'cm'",
		"expand {Interval[1, 3]} per -0.0",
	} {
		if got := evalDefaultUnit(t, expr); got == "{}" {
			t.Errorf("%s = {} — a zero step means the unit step, not no step", expr)
		}
	}

	// And the runaway guard is still what stops a positive step that is merely
	// very small, which is the case it was there for and which this does not
	// replace.
	if got := evalDefaultUnit(t, "Count(expand Interval[0.0, 1.0] per 0.00001)"); got != "10001" {
		t.Errorf("a tiny positive step gives %s points, want the guard's 10001", got)
	}
}

// TestQuantityExpansionBehavesLikeTheDecimalOneItIsBuiltOn checks the properties
// that come from reducing to the decimal case rather than writing a new one, and
// so would be the first things a separate implementation got wrong.
func TestQuantityExpansionBehavesLikeTheDecimalOneItIsBuiltOn(t *testing.T) {
	// The cap on how many points an expansion produces is the same cap.
	quantities := evalDefaultUnit(t, "Count(expand Interval[0 'cm', 100000 'cm'] per 1 'cm')")
	decimals := evalDefaultUnit(t, "Count(expand Interval[0.0, 100000.0] per 1.0)")
	if quantities != decimals {
		t.Errorf("a long expansion gives %s points over quantities and %s over decimals", quantities, decimals)
	}

	// A fractional step accumulates the same way, so the sequence does not drift
	// apart from the decimal one it is made of.
	q := evalDefaultUnit(t, "expand Interval[1 'cm', 2 'cm'] per 0.1 'cm'")
	d := evalDefaultUnit(t, "expand Interval[1.0, 2.0] per 0.1")
	if stripped := strings.ReplaceAll(q, " 'cm'", ""); stripped != d {
		t.Errorf("a fractional step gives\n  %s\nover quantities and\n  %s\nover decimals", q, d)
	}

	// A step in another scale converts, in both overloads.
	if got := evalDefaultUnit(t, "expand Interval[0 'm', 2 'm'] per 50 'cm'"); got != "{0 'm', 0.5 'm', 1 'm', 1.5 'm', 2 'm'}" {
		t.Errorf("a step in centimeters over an interval in meters = %s", got)
	}
}

// TestWidthAndExpandReportDifferentUnitsOnPurpose records an asymmetry that is
// measured rather than accidental, so that changing it is a decision.
//
// Over the same interval the two answer in different units:
//
//	width of Interval[100 'cm', 2 'm']             1 'm'
//	expand Interval[100 'cm', 2 'm'] per 50 'cm'   {100 'cm', 150 'cm', 200 'cm'}
//
// Each inherits the unit from the operation it is defined as. A width is
// `high - low`, and a subtraction answers in the unit of what it subtracts from.
// An expansion walks from the low bound, so it answers in that bound's unit.
// Neither is wrong about the quantity — 1 'm' is 100 'cm' — and forcing them to
// agree would mean overriding one of the two definitions.
func TestWidthAndExpandReportDifferentUnitsOnPurpose(t *testing.T) {
	if got := evalDefaultUnit(t, "width of Interval[100 'cm', 2 'm']"); got != "1 'm'" {
		t.Errorf("width = %s, want 1 'm' — the unit of the subtraction's left side", got)
	}
	if got := evalDefaultUnit(t, "expand Interval[100 'cm', 2 'm'] per 50 'cm'"); got != "{100 'cm', 150 'cm', 200 'cm'}" {
		t.Errorf("expand = %s, want centimeters — the unit it starts walking from", got)
	}
	// And they agree about the quantity, which is the part that matters.
	if got := evalDefaultUnit(t, "width of Interval[100 'cm', 2 'm'] = 100 'cm'"); got != "true" {
		t.Errorf("the two units disagree about the value: %s", got)
	}
}

// TestQuantityExpansionIsTheDecimalOneWithAUnit compares against the decimal
// spelling rather than the integer one, which is the oracle that actually
// applies: a quantity interval is continuous, and it is the decimal expansion
// this is built on.
//
// The integer comparison elsewhere in this file holds only because a step of one
// makes all three agree. These rows are where discrete and continuous part ways,
// and quantities have to follow the continuous side:
//
//	collapse (expand {Interval[1, 5]})           {Interval[1, 5]}
//	collapse (expand {Interval[1.0, 5.0]})       five point intervals
//	collapse (expand {Interval[1 'cm', 5 'cm']}) five point intervals
//
// Expanding and collapsing gets the interval back only where the points are
// adjacent, which they are in the integers and are not between 1 'cm' and 2 'cm'.
// A quantity behaving like the integer there would mean claiming 1.5 'cm' is not
// in the interval.
func TestQuantityExpansionIsTheDecimalOneWithAUnit(t *testing.T) {
	for _, per := range []string{"0.5", "1.0", "0.25"} {
		quantities := evalDefaultUnit(t,
			"expand {Interval[1 'cm', 3 'cm']} per "+per+" 'cm'")
		decimals := evalDefaultUnit(t, "expand {Interval[1.0, 3.0]} per "+per)
		if stripped := strings.ReplaceAll(quantities, " 'cm'", ""); stripped != decimals {
			t.Errorf("per %s gives\n  %s\nover quantities and\n  %s\nover decimals", per, quantities, decimals)
		}
	}

	// Round-tripping follows the continuous reading, not the discrete one.
	q := evalDefaultUnit(t, "collapse (expand {Interval[1 'cm', 5 'cm']})")
	d := evalDefaultUnit(t, "collapse (expand {Interval[1.0, 5.0]})")
	if stripped := strings.ReplaceAll(q, " 'cm'", ""); stripped != d {
		t.Errorf("expanding and collapsing gives\n  %s\nover quantities and\n  %s\nover decimals", q, d)
	}
	// And the integer spelling does get its interval back, which is what makes the
	// above a property of continuity rather than a failure of collapse.
	if got := evalDefaultUnit(t, "collapse (expand {Interval[1, 5]})"); got != "{Interval[1, 5]}" {
		t.Errorf("the integer round trip = %s, want {Interval[1, 5]}", got)
	}

	// Each interval in a list expands in its own unit, and one the step cannot
	// reach drops out rather than taking the others with it.
	mixed := evalDefaultUnit(t, "expand {Interval[1 'cm', 2 'cm'], Interval[1 'm', 2 'm']} per 1 'cm'")
	for _, want := range []string{"1 'cm'", "1 'm'"} {
		if !strings.Contains(mixed, want) {
			t.Errorf("expanding a list of two units = %s, missing %s", mixed, want)
		}
	}
	if got := evalDefaultUnit(t, "expand {Interval[1 'cm', 2 'cm'], Interval[1 's', 2 's']} per 1 'cm'"); strings.Contains(got, "'s'") {
		t.Errorf("an interval the step cannot reach was expanded anyway: %s", got)
	}
}

// TestAStepOfTheWrongTypeIsReported closes what the previous release left
// asserted as broken.
//
// A `per` of a type the evaluator does not recognize was read as "no step given",
// so the expansion ran with the unit step and the author was told nothing:
//
//	expand {Interval[1, 10]} per 'abc'   expanded by one, silently
//	expand {Interval[1, 10]} per {2}     expanded by one, not by two
//
// The semantic phase inferred the step and threw the type away. It checks it now.
// The fix belongs there rather than in the reader: making the reader decline would
// answer the empty list, and an empty list is not what an author who wrote
// `per 'abc'` needs to see either.
func TestAStepOfTheWrongTypeIsReported(t *testing.T) {
	for _, tt := range []struct{ expr, named string }{
		{"expand {Interval[1, 10]} per 'abc'", "System.String"},
		{"expand {Interval[1, 10]} per true", "System.Boolean"},
		{"expand {Interval[1, 10]} per Interval[1,2]", "Interval<System.Integer>"},
		{"expand {Interval[1, 10]} per {1}", "List<System.Integer>"},
	} {
		got := evalDefaultUnit(t, tt.expr)
		if !strings.Contains(got, "semantic error") {
			t.Errorf("%s = %s, want a diagnostic", tt.expr, got)
		}
		// The message names the type it was given, because a diagnostic that
		// does not is half a diagnostic.
		if !strings.Contains(got, tt.named) {
			t.Errorf("the diagnostic for %s does not name %s: %s", tt.expr, tt.named, got)
		}
	}

	// Every spelling CQL does allow still compiles and still answers, so closing
	// the hole did not narrow the door.
	for _, tt := range []struct{ expr, want string }{
		{"Count(expand {Interval[1, 10]} per 2)", "5"},
		{"Count(expand {Interval[1.0, 3.0]} per 0.5)", "5"},
		{"Count(expand {Interval[1 'cm', 3 'cm']} per 1 'cm')", "3"},
		{"Count(expand {Interval[@2018-01-01, @2018-01-04]} per day)", "4"},
		{"Count(expand {Interval[@2018-01-01, @2018-01-04]} per 2 days)", "2"},
		{"Count(expand {Interval[@T10:00, @T12:30]} per hour)", "3"},
		{"Count(expand {Interval[1, 10]} per 2L)", "5"},
		{"Count(expand {Interval[1, 10]} per null)", "10"},
		{"Count(expand {Interval[1, 10]})", "10"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestAOneElementListIsNotAStep draws the line the check had to be careful about.
//
// CQL declares a one-element list implicitly convertible to its element, so
// asking Convertible alone accepted `per {2}` — and the evaluator, which performs
// no such conversion anywhere, expanded by one instead of by two. The engine does
// not do that conversion for `{2} + 3` either, which is an error.
//
// So a list is refused here rather than accepted and ignored. A model type that
// really does convert — a FHIR.Quantity, which the evaluator coerces — still
// passes, and that is asserted beside it so the refusal cannot widen into one.
func TestAOneElementListIsNotAStep(t *testing.T) {
	if got := evalDefaultUnit(t, "expand {Interval[1, 10]} per {2}"); !strings.Contains(got, "semantic error") {
		t.Errorf("per {2} = %s, want a diagnostic — it expanded by one, not by two", got)
	}
	// The conversion CQL declares is not one this engine performs anywhere.
	if got := evalDefaultUnit(t, "{2} + 3"); !strings.Contains(got, "ERROR") {
		t.Errorf("{2} + 3 = %s — the engine has learned the one-element list conversion; "+
			"if it now performs it everywhere, a list step may be allowed again", got)
	}
}

// TestTheDiagnosticNamesTheOperationTheAuthorWrote covers that `collapse` reaches
// the same check, since both spellings carry a `per` and share one function.
//
// A diagnostic about a `collapse` that says "expand" sends the reader looking at
// the wrong line, which is the kind of half-diagnostic this repository has had to
// fix before.
func TestTheDiagnosticNamesTheOperationTheAuthorWrote(t *testing.T) {
	// The article is checked along with the noun. Accepting either one let "the
	// step of a expand" through, which is what the first version of this test did.
	for _, tt := range []struct{ expr, phrase string }{
		{"expand {Interval[1, 3]} per 'abc'", "the step of an expand"},
		{"collapse {Interval[1, 3]} per 'abc'", "the step of a collapse"},
		{"collapse {Interval[1, 3]} per {2}", "the step of a collapse"},
	} {
		if got := evalDefaultUnit(t, tt.expr); !strings.Contains(got, tt.phrase) {
			t.Errorf("%s reported: %s — it should say %q", tt.expr, got, tt.phrase)
		}
	}
}

// TestCollapsePerMergesWhatIsWithinAStep covers `collapse … per`, which was
// accepted and then ignored: the evaluator never passed the step down and the
// collapse never took one, so `per 1`, `per 3` and `per 10` all answered what
// writing no `per` answers.
//
// Two intervals merge when the high bound of one, moved up by the step, reaches
// the low bound of the next. It is posed that way — rather than as "is the gap
// smaller than the step" — because advancing a value by a quantity is arithmetic
// this engine does for every point type, temporal included, while subtracting two
// bounds to get a gap is not defined over temporals at all.
//
// The reach is measured to the successor, because that is how two bounds touching
// is already defined here: `[1, 3]` and `[5, 7]` are one point apart, so `per 1`
// closes that gap and no step at all does not. Which also means a step of nothing
// means exactly what it meant before.
func TestCollapsePerMergesWhatIsWithinAStep(t *testing.T) {
	const twoApart = "collapse {Interval[1, 3], Interval[5, 7]}"
	for _, tt := range []struct{ per, want string }{
		// One point lies between them, so one step closes it and none does not.
		{" per 1", "{Interval[1, 7]}"},
		{" per 3", "{Interval[1, 7]}"},
		{" per 0", "{Interval[1, 3], Interval[5, 7]}"},
		{"", "{Interval[1, 3], Interval[5, 7]}"},
	} {
		if got := evalDefaultUnit(t, twoApart+tt.per); got != tt.want {
			t.Errorf("collapse%s = %s, want %s", tt.per, got, tt.want)
		}
	}

	// Far enough apart is still far enough apart, whatever the step.
	if got := evalDefaultUnit(t, "collapse {Interval[1, 3], Interval[8, 9]} per 1"); got != "{Interval[1, 3], Interval[8, 9]}" {
		t.Errorf("a gap wider than the step = %s, want two intervals", got)
	}

	// Every point type takes the same reading, which is the property that makes
	// this one rule rather than four.
	for _, tt := range []struct{ what, expr, want string }{
		{"decimals", "collapse {Interval[1.0, 3.0], Interval[5.0, 7.0]} per 2.0", "{Interval[1.0, 7.0]}"},
		{"quantities", "collapse {Interval[1 'cm', 3 'cm'], Interval[5 'cm', 7 'cm']} per 2 'cm'", "{Interval[1 'cm', 7 'cm']}"},
		// A step written in another scale of the same dimension converts first.
		{"a step in millimeters", "collapse {Interval[1 'cm', 3 'cm'], Interval[5 'cm', 7 'cm']} per 20 'mm'", "{Interval[1 'cm', 7 'cm']}"},
		{"dates, one day apart", "collapse {Interval[@2020-01-01, @2020-01-03], Interval[@2020-01-05, @2020-01-07]} per 1 day", "{Interval[2020-01-01, 2020-01-07]}"},
		{"dates, two days apart", "collapse {Interval[@2020-01-01, @2020-01-03], Interval[@2020-01-06, @2020-01-07]} per 1 day", "{Interval[2020-01-01, 2020-01-03], Interval[2020-01-06, 2020-01-07]}"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s: %s = %s, want %s", tt.what, tt.expr, got, tt.want)
		}
	}

	// Widening the step can only merge more, never less. It could not: advancing
	// an integer bound by a fractional step gives a decimal, whose successor is an
	// epsilon rather than the next integer, so `per 1` merged a one-point gap,
	// `per 1.5` did not, and `per 2` did again. Taking the successor of the bound
	// before stepping asks the question in the bound's own type, where the
	// discreteness is.
	var merged []string
	for _, per := range []string{"0.5", "1", "1.5", "2", "2.5", "3"} {
		got := evalDefaultUnit(t, twoApart+" per "+per)
		merged = append(merged, per+"="+got)
		if got == "{Interval[1, 7]}" {
			continue
		}
		// Once a step has merged, no wider one may stop merging.
		for _, earlier := range merged[:len(merged)-1] {
			if strings.HasSuffix(earlier, "{Interval[1, 7]}") {
				t.Errorf("a wider step merges less than a narrower one: %v", merged)
				break
			}
		}
	}

	// An open bound is not part of its interval, so it widens the gap by one point
	// and the step has to grow to match. Without this the four combinations of open
	// and closed answered the same, which the rest of the package does not do.
	for _, tt := range []struct{ expr, want string }{
		// [1, 3) holds up to 2, so the gap to [5, 7] is two points, not one.
		{"collapse {Interval[1, 3), Interval[5, 7]} per 1", "{Interval[1, 3), Interval[5, 7]}"},
		{"collapse {Interval[1, 3), Interval[5, 7]} per 2", "{Interval[1, 7]}"},
		// (5, 7] starts at 6, so it is the low bound that moves in.
		{"collapse {Interval[1, 3], Interval(5, 7]} per 1", "{Interval[1, 3], Interval(5, 7]}"},
		{"collapse {Interval[1, 3], Interval(5, 7]} per 2", "{Interval[1, 7]}"},
		// Both open: three points between them.
		{"collapse {Interval[1, 3), Interval(5, 7]} per 2", "{Interval[1, 3), Interval(5, 7]}"},
		{"collapse {Interval[1, 3), Interval(5, 7]} per 3", "{Interval[1, 7]}"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}

	// A chain merges through, and one link too far away breaks it.
	for _, tt := range []struct{ expr, want string }{
		{"collapse {Interval[1, 3], Interval[5, 7], Interval[9, 11]} per 1", "{Interval[1, 11]}"},
		{"collapse {Interval[1, 3], Interval[5, 7], Interval[20, 21]} per 1", "{Interval[1, 7], Interval[20, 21]}"},
		// The order they are written in does not decide it; they are sorted first.
		{"collapse {Interval[9, 11], Interval[1, 3], Interval[5, 7]} per 1", "{Interval[1, 11]}"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}

	// What collapse already did is untouched: overlapping and touching intervals
	// merge with no step at all.
	for _, tt := range []struct{ expr, want string }{
		{"collapse {Interval[1, 4], Interval[3, 7]}", "{Interval[1, 7]}"},
		{"collapse {Interval[1, 3], Interval[4, 7]}", "{Interval[1, 7]}"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestOneSuccessorForEveryOperator covers the repair of a split this file used to
// assert: there were two implementations of "the next value", and one did not know
// Date.
//
// cqltypes.Successor knows Integer, Decimal, DateTime, Date, Time and Quantity and
// guards the representable range. funcs.intervalSuccessor knew everything but
// Date, and sat under `meets`, `except` and the collapse that rests on `meets` — so
// two consecutive days did not touch while two consecutive integers did, though
// the engine answered `successor of @2020-01-03` correctly all along. That is what
// made it a contradiction rather than a limit.
//
// Every row is the integer spelling beside the date one, because agreeing with
// itself across point types is the property, not any particular answer.
func TestOneSuccessorForEveryOperator(t *testing.T) {
	for _, tt := range []struct{ what, integers, dates, want string }{
		{"meets",
			"Interval[1, 3] meets Interval[4, 7]",
			"Interval[@2020-01-01, @2020-01-03] meets Interval[@2020-01-04, @2020-01-07]", "true"},
		{"meets before",
			"Interval[1, 3] meets before Interval[4, 7]",
			"Interval[@2020-01-01, @2020-01-03] meets before Interval[@2020-01-04, @2020-01-07]", "true"},
		{"meets after",
			"Interval[4, 7] meets after Interval[1, 3]",
			"Interval[@2020-01-04, @2020-01-07] meets after Interval[@2020-01-01, @2020-01-03]", "true"},
		{"not meeting",
			"Interval[1, 3] meets Interval[5, 7]",
			"Interval[@2020-01-01, @2020-01-03] meets Interval[@2020-01-05, @2020-01-07]", "false"},
		{"overlaps, which they do not",
			"Interval[1, 3] overlaps Interval[4, 7]",
			"Interval[@2020-01-01, @2020-01-03] overlaps Interval[@2020-01-04, @2020-01-07]", "false"},
	} {
		gotInt := evalDefaultUnit(t, tt.integers)
		gotDate := evalDefaultUnit(t, tt.dates)
		if gotInt != tt.want || gotDate != tt.want {
			t.Errorf("%s: integers = %s, dates = %s, want %s for both",
				tt.what, gotInt, gotDate, tt.want)
		}
	}

	// union is the fourth caller, and the one where the copy cost a wrong answer
	// rather than a differently written one: two intervals that touch have a union,
	// and over dates it was null.
	for _, tt := range []struct{ expr, want string }{
		{"Interval[@2020-01-01, @2020-01-03] union Interval[@2020-01-04, @2020-01-07]",
			"Interval[2020-01-01, 2020-01-07]"},
		// Not touching is still no union, and overlapping still unions.
		{"Interval[@2020-01-01, @2020-01-03] union Interval[@2020-01-05, @2020-01-07]", "null"},
		{"Interval[@2020-01-01, @2020-01-05] union Interval[@2020-01-03, @2020-01-07]",
			"Interval[2020-01-01, 2020-01-07]"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}

	// collapse rests on meets, and except builds a new boundary out of the step.
	// Both were reading the copy that did not know Date.
	for _, tt := range []struct{ integers, dates string }{
		{"collapse {Interval[1, 3], Interval[4, 7]}",
			"collapse {Interval[@2020-01-01, @2020-01-03], Interval[@2020-01-04, @2020-01-07]}"},
		{"Interval[1, 10] except Interval[1, 3]",
			"Interval[@2020-01-01, @2020-01-10] except Interval[@2020-01-01, @2020-01-03]"},
		{"Interval[1, 10] except Interval[8, 10]",
			"Interval[@2020-01-01, @2020-01-10] except Interval[@2020-01-08, @2020-01-10]"},
	} {
		gotInt := evalDefaultUnit(t, tt.integers)
		gotDate := evalDefaultUnit(t, tt.dates)
		// The shapes differ by their values; what has to match is that neither
		// answer is written with an open bound where the other has a closed one.
		if strings.ContainsAny(gotInt, "()") != strings.ContainsAny(gotDate, "()") {
			t.Errorf("%s gave %s but %s gave %s — one built a closed boundary and the other an open one",
				tt.integers, gotInt, tt.dates, gotDate)
		}
	}

	// `except` used to write its new boundary open where the integer spelling
	// writes it closed. Those describe the same interval, which is measured here
	// rather than asserted in a comment: the engine agrees through `=`, `~`,
	// `start of`, `contains` and `width`.
	for _, expr := range []string{
		"Interval(@2020-01-03, @2020-01-10] = Interval[@2020-01-04, @2020-01-10]",
		"Interval(@2020-01-03, @2020-01-10] ~ Interval[@2020-01-04, @2020-01-10]",
		"start of Interval(@2020-01-03, @2020-01-10] = start of Interval[@2020-01-04, @2020-01-10]",
		"width of Interval(3, 10] = width of Interval[4, 10]",
	} {
		if got := evalDefaultUnit(t, expr); got != "true" {
			t.Errorf("%s = %s — the open and closed spellings are not the same interval after all", expr, got)
		}
	}

	// And the operator that was right all along still is.
	if got := evalDefaultUnit(t, "successor of @2020-01-03"); got != "2020-01-04" {
		t.Errorf("successor of a date = %s", got)
	}
}

// TestTheStepCheckDoesNotCryWolf is the limit on the diagnostic above, and the
// half that decides whether a check is worth having.
//
// A checker that reports a step it merely could not work out is worse than one
// that stays quiet, so everything a step can legitimately be written as has to go
// through. Each of these types the step differently — a parameter, a define, a
// call into an included library whose type this phase cannot see, a FHIR choice
// element, a conditional — and none of them is refused.
func TestTheStepCheckDoesNotCryWolf(t *testing.T) {
	const fhir = "using FHIR version '4.0.1'\n" +
		"include FHIRHelpers version '4.0.1' called FHIRHelpers\ncontext Patient\n"
	for _, tt := range []struct{ what, header, expr string }{
		{"a quantity parameter", "parameter P Quantity\n", "expand {Interval[1 'cm', 9 'cm']} per P"},
		{"an integer parameter", "parameter P Integer\n", "expand {Interval[1, 9]} per P"},
		{"a define", "define S: 2\n", "expand {Interval[1, 9]} per S"},
		{"a quantity define", "define S: 2 'cm'\n", "expand {Interval[1 'cm', 9 'cm']} per S"},
		{"a call this phase cannot type", "include FHIRHelpers version '4.0.1' called FH\n",
			"expand {Interval[1, 9]} per FH.ToString(2)"},
		{"a choice element", fhir, "expand {Interval[1 'cm', 9 'cm']} per First([Observation] O).value"},
		{"a choice element cast", fhir,
			"expand {Interval[1 'cm', 9 'cm']} per (First([Observation] O).value as FHIR.Quantity)"},
		{"a conditional", "", "expand {Interval[1 'cm', 9 'cm']} per (if true then 1 'cm' else 2 'cm')"},
	} {
		src := "library T version '1.0'\n" + tt.header + "define X: " + tt.expr + "\n"
		diags, err := NewEngine().Check(src)
		if err != nil {
			t.Errorf("%s did not check: %v", tt.what, err)
			continue
		}
		for _, d := range diags {
			if strings.Contains(d.Message, "the step of") {
				t.Errorf("%s was refused: %s", tt.what, d.Message)
			}
		}
	}

	// And a type that is genuinely not a step is still refused, so the above is
	// not passing because the check stopped running.
	if got := evalDefaultUnit(t, "expand {Interval[1, 9]} per 1:2"); !strings.Contains(got, "the step of") {
		t.Errorf("a ratio step = %s, want the diagnostic — the check is not running", got)
	}

	// Both spellings of expand carry a step, and the check has to reach wherever
	// one is written rather than only at the top of a define.
	for _, expr := range []string{
		"expand Interval[1, 3] per 'abc'",
		"expand {Interval[1, 3]} per 'abc'",
		"Count(expand Interval[1, 3] per 'abc')",
		"if true then expand Interval[1, 3] per 'abc' else {}",
	} {
		if got := evalDefaultUnit(t, expr); !strings.Contains(got, "the step of") {
			t.Errorf("%s = %s, want the diagnostic", expr, got)
		}
	}

	// Two bad steps are two diagnostics. The semantic phase is meant to reach the
	// end of the library rather than stop at the first thing it finds, and a check
	// that aborted the walk would report one of these and hide the other.
	src := "library T version '1.0'\n" +
		"define X: expand (expand {Interval[1, 3]} per 'abc') per 'xyz'\n"
	diags, err := NewEngine().Check(src)
	if err != nil {
		t.Fatalf("checking two bad steps: %v", err)
	}
	var steps int
	for _, d := range diags {
		if strings.Contains(d.Message, "the step of") {
			steps++
		}
	}
	if steps != 2 {
		t.Errorf("two bad steps gave %d diagnostics, want 2", steps)
	}
}

// TestTheStepOfACollapseFollowsTheArithmeticItIsMadeOf covers what a step may be,
// which is decided by whether the engine can add it to a bound.
//
// Every row here is the arithmetic answering, not a rule written for collapse: a
// step in another dimension cannot be added, a bare step is read in the bound's
// own unit, and the UCUM year and month codes are refused by this engine's date
// arithmetic on purpose — `@2020-01-01 + 1 'a'` is an error, and a step of `1 'a'`
// therefore closes no gap while `1 year` does.
func TestTheStepOfACollapseFollowsTheArithmeticItIsMadeOf(t *testing.T) {
	for _, tt := range []struct{ what, expr, want string }{
		{"another dimension", "collapse {Interval[1 'cm', 3 'cm'], Interval[5 'cm', 7 'cm']} per 1 's'",
			"{Interval[1 'cm', 3 'cm'], Interval[5 'cm', 7 'cm']}"},
		{"a bare step over quantities", "collapse {Interval[1 'cm', 3 'cm'], Interval[5 'cm', 7 'cm']} per 2",
			"{Interval[1 'cm', 7 'cm']}"},
		{"the UCUM day code", "collapse {Interval[@2020-01-01, @2020-01-03], Interval[@2020-01-05, @2020-01-07]} per 1 'd'",
			"{Interval[2020-01-01, 2020-01-07]}"},
		{"a calendar year", "collapse {Interval[@2020-01-01, @2020-06-01], Interval[@2021-01-01, @2021-06-01]} per 1 year",
			"{Interval[2020-01-01, 2021-06-01]}"},
		{"the UCUM year code", "collapse {Interval[@2020-01-01, @2020-06-01], Interval[@2021-01-01, @2021-06-01]} per 1 'a'",
			"{Interval[2020-01-01, 2020-06-01], Interval[2021-01-01, 2021-06-01]}"},
		{"months, at month precision", "collapse {Interval[@2020-01, @2020-03], Interval[@2020-05, @2020-07]} per 1 month",
			"{Interval[2020-01, 2020-07]}"},
		{"hours", "collapse {Interval[@2020-01-01T10:00:00, @2020-01-01T11:00:00], Interval[@2020-01-01T13:00:00, @2020-01-01T14:00:00]} per 2 hours",
			"{Interval[2020-01-01T10:00:00, 2020-01-01T14:00:00]}"},
		{"bounds of different types", "collapse {Interval[1, 3], Interval[5 'cm', 7 'cm']} per 1",
			"{Interval[1, 3], Interval[5 'cm', 7 'cm']}"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s: %s = %s, want %s", tt.what, tt.expr, got, tt.want)
		}
	}

	// The UCUM year row above is only right while the arithmetic refuses it. If
	// that policy changes, the step should follow it rather than stay behind.
	if got := evalDefaultUnit(t, "@2020-01-01 + 1 'a'"); !strings.Contains(got, "ERROR") {
		t.Errorf("@2020-01-01 + 1 'a' = %s — the engine has learned the UCUM year; "+
			"a step of 1 'a' should now close a one-year gap too", got)
	}
}

// TestExpandAndCollapseWithTheSameStepIsTheIdentity is the property the two
// operators owe each other, and the closest thing to an oracle either of them has.
//
// Expanding an interval by a step and collapsing the pieces by that same step must
// give the interval back. Neither operator was written against the other, so their
// agreeing is evidence about both — and it is what a step of nothing cannot do
// over a continuous type: the pieces of `[1 'cm', 5 'cm']` are a centimeter apart,
// which is a gap until a step says it is not.
func TestExpandAndCollapseWithTheSameStepIsTheIdentity(t *testing.T) {
	// The interval has to divide by the step, because expand drops a final piece
	// that would not fit: `expand {Interval[1, 9]} per 2` stops at [7, 8] and the
	// 9 is gone before collapse ever sees it. That is expand's documented
	// behavior and the corpus pins it, so the identity is stated where it holds.
	//
	// Compared with the engine's own `=` rather than by rendering both: a decimal
	// interval comes back printed 1 where it was written 1.0, which is the same
	// interval and a different string.
	for _, tt := range []struct{ interval, per string }{
		{"Interval[1, 5]", "1"},
		{"Interval[1, 8]", "4"},
		{"Interval[1 'cm', 5 'cm']", "1 'cm'"},
		{"Interval[1.0, 3.0]", "0.5"},
		{"Interval[@2020-01-01, @2020-01-05]", "1 day"},
	} {
		// Parenthesized, because `per` swallows what follows it: written without
		// them, `collapse … per 1 = {…}` parses as `per (1 = {…})` and the step
		// becomes a Boolean. The step diagnostic caught that while this test was
		// being written, which is the first thing it has been useful for.
		expr := "(collapse (expand {" + tt.interval + "} per " + tt.per + ") per " + tt.per +
			") = {" + tt.interval + "}"
		if got := evalDefaultUnit(t, expr); got != "true" {
			t.Errorf("expanding %s per %s and collapsing it back is not the same interval (%s): got %s, wanted %s",
				tt.interval, tt.per, got,
				evalDefaultUnit(t, "collapse (expand {"+tt.interval+"} per "+tt.per+") per "+tt.per),
				evalDefaultUnit(t, "{"+tt.interval+"}"))
		}
	}

	// Collapsing twice is collapsing once.
	once := evalDefaultUnit(t, "collapse {Interval[1, 3], Interval[5, 7]} per 1")
	twice := evalDefaultUnit(t, "collapse (collapse {Interval[1, 3], Interval[5, 7]} per 1) per 1")
	if once != twice {
		t.Errorf("collapsing twice gave %s and once gave %s", twice, once)
	}

	// A step changes nothing about intervals that already overlap, however wide it
	// is, and the nulls a collapse drops are dropped the same way with one.
	for _, tt := range []struct{ expr, want string }{
		{"collapse {Interval[1, 5], Interval[3, 8]} per 100", "{Interval[1, 8]}"},
		{"collapse {Interval[1, 3], null} per 1", "{Interval[1, 3]}"},
		{"collapse {Interval(null, null)} per 1", "{}"},
		{"collapse {Interval[1, 3]} per 1", "{Interval[1, 3]}"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestAStepThatCannotBeTakenStillAsksWhetherTheBoundsTouch covers the ends of the
// representable range, where advancing a bound is not possible at all.
//
//	collapse {Interval[@9999-12-28, @9999-12-30],
//	          Interval[@9999-12-31, @9999-12-31]} per 1 day
//
// Those two are consecutive days. Stepping a day past the 31st leaves the calendar
// — there is no 10000-01-01 — so the step was discarded and the intervals stayed
// apart, even though the successor had already arrived before the step was taken.
//
// A step that cannot be applied now falls back to the question with no step in it,
// which is the right answer in both cases it happens: two consecutive days touch
// whether or not a further day exists, and a step in seconds carries a bound in
// centimeters nowhere, so the fallback does not merge what a wrong unit should
// have left alone.
func TestAStepThatCannotBeTakenStillAsksWhetherTheBoundsTouch(t *testing.T) {
	for _, tt := range []struct{ what, expr, want string }{
		{"the end of the calendar",
			"collapse {Interval[@9999-12-28, @9999-12-30], Interval[@9999-12-31, @9999-12-31]} per 1 day",
			"{Interval[9999-12-28, 9999-12-31]}"},
		{"a step past the end",
			"collapse {Interval[@9999-12-28, @9999-12-31], Interval[@9999-12-31, @9999-12-31]} per 1000 years",
			"{Interval[9999-12-28, 9999-12-31]}"},
		{"the start of the calendar",
			"collapse {Interval[@0001-01-01, @0001-01-02], Interval[@0001-01-04, @0001-01-05]} per 1 day",
			"{Interval[0001-01-01, 0001-01-05]}"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s: %s = %s, want %s", tt.what, tt.expr, got, tt.want)
		}
	}

	// And the fallback does not turn an unusable unit into a merge: these are the
	// rows from the arithmetic test above, which have to keep declining.
	for _, tt := range []struct{ expr, want string }{
		{"collapse {Interval[1 'cm', 3 'cm'], Interval[5 'cm', 7 'cm']} per 1 's'",
			"{Interval[1 'cm', 3 'cm'], Interval[5 'cm', 7 'cm']}"},
		{"collapse {Interval[@2020-01-01, @2020-06-01], Interval[@2021-01-01, @2021-06-01]} per 1 'a'",
			"{Interval[2020-01-01, 2020-06-01], Interval[2021-01-01, 2021-06-01]}"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — the fallback must not merge on a unit that does not apply", tt.expr, got, tt.want)
		}
	}
}

// TestTimeStepsWrapAtMidnight covers the one point type whose arithmetic wraps.
//
// `@T23:00:00 + 1 hour` is `00:00:00` in this engine — not an error and not 24:00.
// So a step taken near midnight lands *before* where it set off from, and
// comparing it against the next low bound answered "does not reach" for a gap the
// step covers several times over: `[@T22:00:00, @T23:00:00]` and
// `[@T23:59:59, @T23:59:59]` are a minute short of touching and stayed apart under
// a step of an hour.
//
// Having wrapped means the step reached the end of the day, so it reaches any
// bound at or after where it started — and no further, which is why the last row
// does not join the beginning of one day to the end of it.
func TestTimeStepsWrapAtMidnight(t *testing.T) {
	for _, tt := range []struct{ what, expr, want string }{
		{"a step that wraps still reaches what is left of the day",
			"collapse {Interval[@T22:00:00, @T23:00:00], Interval[@T23:59:59, @T23:59:59]} per 1 hour",
			"{Interval[22:00:00, 23:59:59]}"},
		{"and reaches no further than it should",
			"collapse {Interval[@T22:00:00, @T23:00:00], Interval[@T23:30:00, @T23:59:59]} per 1 minute",
			"{Interval[22:00:00, 23:00:00], Interval[23:30:00, 23:59:59]}"},
		{"midnight is not next to the end of the day",
			"collapse {Interval[@T00:00:00, @T01:00:00], Interval[@T23:00:00, @T23:30:00]} per 1 hour",
			"{Interval[00:00:00, 01:00:00], Interval[23:00:00, 23:30:00]}"},
		{"and away from midnight nothing changed",
			"collapse {Interval[@T10:00:00, @T11:00:00], Interval[@T13:00:00, @T14:00:00]} per 2 hours",
			"{Interval[10:00:00, 14:00:00]}"},
		{"including where it should not merge",
			"collapse {Interval[@T10:00:00, @T11:00:00], Interval[@T13:00:00, @T14:00:00]} per 1 hour",
			"{Interval[10:00:00, 11:00:00], Interval[13:00:00, 14:00:00]}"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s: %s = %s, want %s", tt.what, tt.expr, got, tt.want)
		}
	}

	// The wrap is the arithmetic's, not this function's, and the rule here is to
	// follow it. If Time ever stops wrapping, the row above stops being right.
	if got := evalDefaultUnit(t, "@T23:00:00 + 1 hour"); got != "00:00:00" {
		t.Errorf("@T23:00:00 + 1 hour = %s — Time no longer wraps, so a step near "+
			"midnight needs rereading", got)
	}
}

// TestALongBoundIsAnIntegerBound covers the last point type, which behaves as the
// integer one does at every step width.
func TestALongBoundIsAnIntegerBound(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"collapse {Interval[1L, 3L], Interval[5L, 7L]} per 1", "{Interval[1, 7]}"},
		{"collapse {Interval[1L, 3L], Interval[5L, 7L]} per 1L", "{Interval[1, 7]}"},
		{"collapse {Interval[1L, 3L], Interval[8L, 9L]} per 1", "{Interval[1, 3], Interval[8, 9]}"},
		{"collapse {Interval[1, 3], Interval[5, 7]} per 1L", "{Interval[1, 7]}"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

// TestTheOneSuccessorReadsEveryPointTypeTheSameWay enumerates what the two
// implementations could have differed on, since a shared implementation replacing
// a copy moves whatever the copy read differently.
//
// The dimensions are: which point types it knows, what it does at the ends of the
// representable range, and how coarse a precision it steps at. The decimal step is
// the fourth and was checked before the change rather than after — the two
// constants are the same 0.00000001, and a delegation between different ones would
// have moved every decimal boundary in the engine without failing a test.
func TestTheOneSuccessorReadsEveryPointTypeTheSameWay(t *testing.T) {
	// A Long boundary reads as an Integer one, in both operators.
	for _, tt := range [][2]string{
		{"Interval[1, 3] meets Interval[4, 7]", "Interval[1L, 3L] meets Interval[4L, 7L]"},
		{"Interval[1, 10] except Interval[1, 3]", "Interval[1L, 10L] except Interval[1L, 3L]"},
	} {
		if a, b := evalDefaultUnit(t, tt[0]), evalDefaultUnit(t, tt[1]); a != b {
			t.Errorf("%s = %s but %s = %s", tt[0], a, tt[1], b)
		}
	}

	// The ends of the calendar, where the shared implementation guards the range
	// and the copy did not.
	for _, tt := range []struct{ expr, want string }{
		{"Interval[@9999-12-01, @9999-12-30] meets Interval[@9999-12-31, @9999-12-31]", "true"},
		{"Interval[@0001-01-01, @0001-01-02] meets Interval[@0001-01-03, @0001-01-04]", "true"},
		{"Interval[@9999-12-01, @9999-12-31] except Interval[@9999-12-31, @9999-12-31]",
			"Interval[9999-12-01, 9999-12-30]"},
		// Sharing a bound is overlapping, not meeting, at either end of the day.
		{"Interval[@T22:00:00, @T23:59:59] meets Interval[@T23:59:59, @T23:59:59]", "false"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}

	// A step is taken at the boundary's own precision: the month after @2020-03 is
	// @2020-04, not the next day.
	for _, expr := range []string{
		"Interval[@2020-01, @2020-03] meets Interval[@2020-04, @2020-06]",
		"Interval[@2020, @2021] meets Interval[@2022, @2023]",
		"Interval[@2020-01-01T10, @2020-01-01T11] meets Interval[@2020-01-01T12, @2020-01-01T13]",
	} {
		if got := evalDefaultUnit(t, expr); got != "true" {
			t.Errorf("%s = %s, want true — the step should be one of whatever the bound states", expr, got)
		}
	}
}

// TestThePublishedCollapseDoesNotMove is the measurement that decides what this
// change costs published CQL, rather than what it costs a fixture.
//
// The transitive closure of the two functions this branch replaced is six
// operators: meets, meets before, meets after, union, except and collapse. Of
// those, the 19 published measures use `union` 74 times across 11 libraries and
// `collapse` once; `meets` and `except` not at all. The unions are all unions of
// lists of resources — `union [Encounter: "ED"]` — which this does not touch.
//
// The one collapse is this function, and its intervals are DateTime, which the
// replaced copy already knew. Only Date was missing from it. So the answer here is
// the same before and after, and the change reaches published CQL nowhere:
//
//	CumulativeDays over DateTime intervals    18, 19, 18, 19 — identical on main
//	the same shape written with Date          two intervals on main, one here
func TestThePublishedCollapseDoesNotMove(t *testing.T) {
	const fn = "define function CumulativeDays(Intervals List<Interval<DateTime>>):\n" +
		"  Sum((collapse Intervals) CollapsedInterval return all duration in days of CollapsedInterval)\n"
	for _, tt := range []struct{ what, call, want string }{
		{"consecutive days", "CumulativeDays({Interval[@2020-01-01T00:00:00, @2020-01-10T00:00:00], " +
			"Interval[@2020-01-11T00:00:00, @2020-01-20T00:00:00]})", "18"},
		{"overlapping", "CumulativeDays({Interval[@2020-01-01T00:00:00, @2020-01-10T00:00:00], " +
			"Interval[@2020-01-05T00:00:00, @2020-01-20T00:00:00]})", "19"},
		{"far apart", "CumulativeDays({Interval[@2020-01-01T00:00:00, @2020-01-10T00:00:00], " +
			"Interval[@2020-02-01T00:00:00, @2020-02-10T00:00:00]})", "18"},
		{"a millisecond apart", "CumulativeDays({Interval[@2020-01-01T00:00:00.000, @2020-01-10T00:00:00.000], " +
			"Interval[@2020-01-10T00:00:00.001, @2020-01-20T00:00:00.000]})", "19"},
	} {
		src := "library T version '1.0'\n" + fn + "define X: " + tt.call + "\n"
		got, err := NewEngine().EvaluateExpression(context.Background(), src, "X", nil, nil)
		if err != nil {
			t.Errorf("%s: %v", tt.what, err)
			continue
		}
		if got == nil || got.String() != tt.want {
			t.Errorf("%s: CumulativeDays = %v, want %s", tt.what, got, tt.want)
		}
	}

	// And the Date spelling is where the change does land, which is what makes the
	// rows above a measurement rather than a coincidence.
	if got := evalDefaultUnit(t,
		"collapse {Interval[@2020-01-01, @2020-01-10], Interval[@2020-01-11, @2020-01-20]}"); got != "{Interval[2020-01-01, 2020-01-20]}" {
		t.Errorf("the Date spelling = %s, want one merged interval", got)
	}
}

// TestTheStepIsTakenAtThePrecisionTheValueStates is what a sweep of every
// interval operator over Date against DateTime turned up, and it is a property
// rather than a defect — though it looked like four defects first.
//
// Comparing `Interval[@2020-01-01, @2020-01-03]` against
// `Interval[@2020-01-01T00:00:00, @2020-01-03T00:00:00]` as though they were the
// same shape makes meets, meets before, union and collapse all disagree between
// the two. They are not the same shape: the successor of a value is one of
// whatever precision it states, so the date after the 3rd is the 4th while the
// second after 00:00:00 on the 3rd is 00:00:01 on the 3rd. Written at matching
// precision, the two agree everywhere.
//
// Worth pinning because the disagreement reads exactly like a defect, and because
// it is the rule the whole successor rests on.
func TestTheStepIsTakenAtThePrecisionTheValueStates(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"successor of @2020-01-03", "2020-01-04"},
		{"successor of @2020-01-03T00:00:00", "2020-01-03T00:00:01"},
		{"successor of @2020-01-03T", "2020-01-04"},
		{"successor of @2020-01", "2020-02"},
	} {
		if got := evalDefaultUnit(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
		}
	}

	// At matching precision, a DateTime interval meets exactly where a Date one
	// does — which is what makes the four operators agree once the shapes really
	// are the same.
	for _, expr := range []string{
		"Interval[@2020-01-01, @2020-01-03] meets Interval[@2020-01-04, @2020-01-07]",
		"Interval[@2020-01-01T, @2020-01-03T] meets Interval[@2020-01-04T, @2020-01-07T]",
		"Interval[@2020-01-01T00:00:00, @2020-01-03T00:00:00] meets Interval[@2020-01-03T00:00:01, @2020-01-07T00:00:00]",
	} {
		if got := evalDefaultUnit(t, expr); got != "true" {
			t.Errorf("%s = %s, want true", expr, got)
		}
	}

	// And a day apart at second precision is not touching, which is the row that
	// looked like a defect.
	if got := evalDefaultUnit(t,
		"Interval[@2020-01-01T00:00:00, @2020-01-03T00:00:00] meets Interval[@2020-01-04T00:00:00, @2020-01-07T00:00:00]"); got != "false" {
		t.Errorf("a day apart at second precision = %s, want false", got)
	}
}

// TestAgeIsMeasuredFromTheEvaluationTimestamp is the property the deleted
// wrappers could have broken, and the reason they went rather than a tidiness
// argument.
//
// funcs.AgeInDays, AgeInMonths and AgeInWeeks were one-line wrappers that passed a
// nil reference date, and referenceDate says what that means in its own comment:
// "Reading the clock here is the last resort. The evaluator passes the evaluation's
// frozen timestamp, so that an age agrees with the Today() in the same expression;
// only a direct caller of this package lands here." Those three wrappers were the
// only way to be that direct caller, and nothing called them.
//
// So they existed solely to reach the behavior the engine avoids: an age measured
// against the machine's clock rather than the request's timestamp, which would
// disagree with the Today() beside it. This engine learned that lesson in v1.19.0,
// when Now() and Today() answering in UTC moved whole populations.
//
// The test asserts the property rather than the deletion: an age and the Today()
// in the same expression are measured from the same instant, whatever the clock
// says.
func TestAgeIsMeasuredFromTheEvaluationTimestamp(t *testing.T) {
	const src = "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\n" +
		"define Y: AgeInYearsAt(Today())\n" +
		"define D: AgeInDaysAt(Today())\n" +
		"define T: Today()\n"
	// A timestamp far from the machine's clock: an age read off the real clock
	// would be wrong by decades.
	engine := NewEngine(WithEvaluationTimestamp(time.Date(2019, 6, 1, 12, 0, 0, 0, time.UTC)))
	patient := []byte(`{"resourceType":"Patient","id":"p1","birthDate":"2000-01-15"}`)
	// The birth date is deliberately away from the anniversary: on the anniversary
	// itself this engine answers a year short for anyone born in a leap year after
	// February, which is a defect of its own and is asserted just below.
	for _, tt := range []struct{ define, want string }{
		{"Y", "19"},
		{"T", "2019-06-01"},
	} {
		got, err := engine.EvaluateExpression(context.Background(), src, tt.define, patient, nil)
		if err != nil {
			t.Errorf("%s: %v", tt.define, err)
			continue
		}
		if got == nil || got.String() != tt.want {
			t.Errorf("%s = %v, want %s — the age is not being read from the evaluation timestamp",
				tt.define, got, tt.want)
		}
	}
}

// TestAgeIsRightOnTheAnniversary covers what day-of-year arithmetic got wrong, for
// everyone born in a leap year after February.
//
// CalculateAgeInYears compared day-of-year numbers. A leap year's run one ahead of
// a common year's after the 29th of February, so the anniversary read as not yet
// reached: someone born 2000-06-01 was 18 on 2019-06-01, the day they turned 19.
// That is everyone born between the 1st of March and the 31st of December of a
// leap year — roughly one person in five — on exactly the day it matters most,
// because age decides populations. 30 of the 36 anniversaries swept below were
// wrong.
//
// No specification was needed to settle it: CalculateAgeInMonths compares month
// and day and answered 228 months for that pair, and 228 months is 19 years. The
// engine contradicted itself one function over, and the months were right.
//
// So the test asserts the agreement rather than a list of ages: the months divide
// into the years at every anniversary.
func TestAgeIsRightOnTheAnniversary(t *testing.T) {
	ask := func(birth, at, define string) string {
		parsed, err := time.Parse("2006-01-02", at)
		if err != nil {
			t.Fatalf("bad date in test: %v", err)
		}
		src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\n" +
			"define Y: AgeInYears()\n" +
			"define Agree: AgeInMonths() div 12 = AgeInYears()\n"
		got, err := NewEngine(WithEvaluationTimestamp(parsed.Add(12*time.Hour))).EvaluateExpression(
			context.Background(), src, define,
			[]byte(`{"resourceType":"Patient","id":"p1","birthDate":"`+birth+`"}`), nil)
		if err != nil {
			return "ERROR: " + err.Error()
		}
		if got == nil {
			return "null"
		}
		return got.String()
	}

	// Swept rather than sampled: every month of a leap year, early, middle and
	// late, on the anniversary itself.
	for m := 1; m <= 12; m++ {
		for _, d := range []int{1, 15, 28} {
			birth := fmt.Sprintf("2000-%02d-%02d", m, d)
			at := fmt.Sprintf("2019-%02d-%02d", m, d)
			if got := ask(birth, at, "Y"); got != "19" {
				t.Errorf("born %s, on %s = %s, want 19 — that is the anniversary", birth, at, got)
			}
			if got := ask(birth, at, "Agree"); got != "true" {
				t.Errorf("born %s, on %s: the months and the years disagree", birth, at)
			}
		}
	}

	// The day before and the day after, so the boundary is in the right place
	// rather than merely moved.
	for _, tt := range []struct{ birth, at, want string }{
		{"2000-06-01", "2019-05-31", "18"},
		{"2000-06-01", "2019-06-01", "19"},
		{"2000-06-01", "2019-06-02", "19"},
	} {
		if got := ask(tt.birth, tt.at, "Y"); got != tt.want {
			t.Errorf("born %s, on %s = %s, want %s", tt.birth, tt.at, got, tt.want)
		}
	}

	// Someone born on the 29th of February has no anniversary in a common year.
	// This engine gives it to them on the 1st of March — recorded because it is a
	// convention rather than arithmetic, and jurisdictions differ.
	for _, tt := range []struct{ at, want string }{
		{"2019-02-28", "18"},
		{"2019-03-01", "19"},
		{"2020-02-29", "20"},
	} {
		if got := ask("2000-02-29", tt.at, "Y"); got != tt.want {
			t.Errorf("born on a leap day, on %s = %s, want %s", tt.at, got, tt.want)
		}
	}
}

// TestTheWholeAgeMatrixAgrees walks the family rather than sampling it, which is
// what the last few rounds of this branch kept showing was the difference between
// finding a defect and missing one.
//
// Four units, three spellings, three kinds of reference. Every column has to agree
// with the others about the same span: AgeInX() and CalculateAgeInX(bd) both
// measure to the evaluation timestamp, AgeInXAt(d) and CalculateAgeInX(bd, d) both
// measure to d, and a null reference is null throughout.
func TestTheWholeAgeMatrixAgrees(t *testing.T) {
	at := time.Date(2019, 6, 1, 12, 0, 0, 0, time.UTC)
	patient := []byte(`{"resourceType":"Patient","id":"p1","birthDate":"2000-01-15"}`)
	ask := func(expr string) string {
		src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\ndefine X: " + expr + "\n"
		got, err := NewEngine(WithEvaluationTimestamp(at)).EvaluateExpression(
			context.Background(), src, "X", patient, nil)
		if err != nil {
			return "ERROR: " + err.Error()
		}
		if got == nil {
			return "null"
		}
		return got.String()
	}
	for _, unit := range []string{"Years", "Months", "Weeks", "Days"} {
		toEvaluation := ask("AgeIn" + unit + "()")
		if got := ask("CalculateAgeIn" + unit + "(@2000-01-15)"); got != toEvaluation {
			t.Errorf("%s: AgeIn…() = %s but CalculateAgeIn…(bd) = %s", unit, toEvaluation, got)
		}
		toDate := ask("AgeIn" + unit + "At(@2010-01-15)")
		if got := ask("CalculateAgeIn" + unit + "(@2000-01-15, @2010-01-15)"); got != toDate {
			t.Errorf("%s: AgeIn…At(d) = %s but CalculateAgeIn…(bd, d) = %s", unit, toDate, got)
		}
		if toDate == toEvaluation {
			t.Errorf("%s: measuring to 2010 and to 2019 gave the same answer (%s), so this row proves nothing",
				unit, toDate)
		}
		for _, expr := range []string{
			"AgeIn" + unit + "At(null)",
			"CalculateAgeIn" + unit + "(@2000-01-15, null)",
		} {
			if got := ask(expr); got != "null" {
				t.Errorf("%s = %s, want null", expr, got)
			}
		}
	}
}

// TestAgeWithoutABirthDate pins the edges, which are answers rather than errors.
//
// A patient with no birth date has no age: null, not zero and not a failure. A
// birth date stated only to the year is read at the start of it. A birth date in
// the future gives a negative age — that is what the arithmetic says, the corpus
// has no case for it, and it is recorded rather than decided here, so that
// choosing null instead is a deliberate change with a test to break.
func TestAgeWithoutABirthDate(t *testing.T) {
	at := time.Date(2019, 6, 1, 12, 0, 0, 0, time.UTC)
	ask := func(patient, expr string) string {
		src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\ndefine X: " + expr + "\n"
		got, err := NewEngine(WithEvaluationTimestamp(at)).EvaluateExpression(
			context.Background(), src, "X", []byte(patient), nil)
		if err != nil {
			return "ERROR"
		}
		if got == nil {
			return "null"
		}
		return got.String()
	}
	const none = `{"resourceType":"Patient","id":"p1"}`
	const empty = `{"resourceType":"Patient","id":"p1","birthDate":""}`
	const yearOnly = `{"resourceType":"Patient","id":"p1","birthDate":"2000"}`
	const future = `{"resourceType":"Patient","id":"p1","birthDate":"2030-01-15"}`

	for _, p := range []string{none, empty} {
		for _, expr := range []string{"AgeInYears()", "AgeInYearsAt(@2010-01-15)", "AgeInDaysAt(@2010-01-15)"} {
			if got := ask(p, expr); got != "null" {
				t.Errorf("with no birth date, %s = %s, want null", expr, got)
			}
		}
	}
	if got := ask(yearOnly, "AgeInYears()"); got != "19" {
		t.Errorf("a birth date of 2000 alone gives %s at a 2019 evaluation, want 19", got)
	}
	if got := ask(future, "AgeInYears()"); got != "-11" {
		t.Errorf("a birth date in the future gives %s; it was -11. If this is now null, "+
			"that is a decision worth keeping — check the other units follow it.", got)
	}
}

// TestTheFourUnitsStayCoherentOverAWideSweep is the wide net around the
// anniversary fix, and it measures something different from the anniversary
// sweep: that one lands on the anniversary itself, where the defect was, while
// this one walks 576 unrelated pairs looking for any place the four units stop
// agreeing with each other.
//
// Only one of the 576 was incoherent before the fix, precisely because a
// reference on the 1st, 15th or 28th rarely falls on an anniversary. Two nets of
// different shapes: the narrow one proves the repair, the wide one proves the
// repair broke nothing.
//
// The births are chosen for the calendar's awkward spots — the 31st of a month,
// the 29th of February, the 30th of April, the 31st of December — against
// references in both a common year and a leap one.
func TestTheFourUnitsStayCoherentOverAWideSweep(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\n" +
		"define Y: AgeInYears()\ndefine M: AgeInMonths()\n" +
		"define W: AgeInWeeks()\ndefine D: AgeInDays()\n"
	ask := func(birth string, at time.Time, define string) int64 {
		got, err := NewEngine(WithEvaluationTimestamp(at)).EvaluateExpression(
			context.Background(), src, define,
			[]byte(`{"resourceType":"Patient","id":"p1","birthDate":"`+birth+`"}`), nil)
		if err != nil || got == nil {
			t.Fatalf("born %s at %s, %s: %v (%v)", birth, at.Format("2006-01-02"), define, got, err)
		}
		var n int64
		if _, err := fmt.Sscanf(got.String(), "%d", &n); err != nil {
			t.Fatalf("not a number: %s", got.String())
		}
		return n
	}
	var checked int
	for _, birth := range []string{
		"2000-01-31", "2000-02-29", "2000-03-31", "2000-04-30",
		"2001-01-31", "1999-12-31", "2000-12-31", "2000-06-15",
	} {
		for _, year := range []int{2019, 2020} {
			for month := 1; month <= 12; month++ {
				for _, day := range []int{1, 15, 28} {
					at := time.Date(year, time.Month(month), day, 12, 0, 0, 0, time.UTC)
					years, months := ask(birth, at, "Y"), ask(birth, at, "M")
					weeks, days := ask(birth, at, "W"), ask(birth, at, "D")
					checked++
					if months/12 != years {
						t.Errorf("born %s at %s: %d years but %d months (%d/12 = %d)",
							birth, at.Format("2006-01-02"), years, months, months, months/12)
					}
					if days/7 != weeks {
						t.Errorf("born %s at %s: %d weeks but %d days (%d/7 = %d)",
							birth, at.Format("2006-01-02"), weeks, days, days, days/7)
					}
					if days < 0 || years < 0 {
						t.Errorf("born %s at %s: negative age (%d years, %d days)",
							birth, at.Format("2006-01-02"), years, days)
					}
				}
			}
		}
	}
	if checked != 576 {
		t.Fatalf("the sweep covered %d pairs, want 576", checked)
	}
}

// TestAPartialBirthDateIsReadAtItsStart covers the dates FHIR allows and real
// data uses, since a birth date need not be complete.
func TestAPartialBirthDateIsReadAtItsStart(t *testing.T) {
	at := time.Date(2019, 6, 15, 12, 0, 0, 0, time.UTC)
	ask := func(birth, define string) string {
		src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\n" +
			"define Y: AgeInYears()\ndefine Agree: AgeInMonths() div 12 = AgeInYears()\n"
		got, err := NewEngine(WithEvaluationTimestamp(at)).EvaluateExpression(
			context.Background(), src, define,
			[]byte(`{"resourceType":"Patient","id":"p1","birthDate":"`+birth+`"}`), nil)
		if err != nil {
			return "ERROR"
		}
		if got == nil {
			return "null"
		}
		return got.String()
	}
	for _, tt := range []struct{ birth, years string }{
		{"2000", "19"},       // read at 2000-01-01
		{"2000-06", "19"},    // at 2000-06-01
		{"2000-06-15", "19"}, // the anniversary itself
		{"2000-06-16", "18"}, // a day short of it
		{"2000-07", "18"},    // next month, so not yet
	} {
		if got := ask(tt.birth, "Y"); got != tt.years {
			t.Errorf("birthDate %s = %s years, want %s", tt.birth, got, tt.years)
		}
		if got := ask(tt.birth, "Agree"); got != "true" {
			t.Errorf("birthDate %s: the months and the years disagree", tt.birth)
		}
	}
}

// TestTheAnniversaryFallsOnTheRequestsLocalDate is the dimension the sweeps above
// do not reach: they fix the timestamp in UTC, so none of them can tell whether an
// age reads the request's local date or the instant behind it.
//
// It matters here more than anywhere. An anniversary is a date, and at half past
// midnight in Kiritimati the UTC instant is still the previous day — so an age
// that normalized to UTC would hand someone their birthday a day late at +14 and a
// day early at -11, and would disagree with the Today() beside it. That is the
// shape of the defect v1.19.0 was about, where a request at 23:30 in UTC-5 dated
// Today() to the following day and moved whole populations.
//
// Asserted as the agreement rather than as six ages: whatever Today() says the
// date is, the age is measured to that date.
func TestTheAnniversaryFallsOnTheRequestsLocalDate(t *testing.T) {
	const birth = `{"resourceType":"Patient","id":"p1","birthDate":"2000-06-01"}`
	const src = "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\n" +
		"define Y: AgeInYears()\ndefine T: Today()\n"
	ask := func(at time.Time, define string) string {
		got, err := NewEngine(WithEvaluationTimestamp(at)).EvaluateExpression(
			context.Background(), src, define, []byte(birth), nil)
		if err != nil {
			return "ERROR: " + err.Error()
		}
		if got == nil {
			return "null"
		}
		return got.String()
	}
	for _, zone := range []struct {
		name  string
		hours int
	}{{"UTC", 0}, {"Kiritimati", 14}, {"Midway", -11}} {
		loc := time.FixedZone(zone.name, zone.hours*3600)

		// Half past midnight on the anniversary: the local date is the birthday
		// even where the UTC instant is not.
		onIt := time.Date(2019, 6, 1, 0, 30, 0, 0, loc)
		if got := ask(onIt, "T"); got != "2019-06-01" {
			t.Errorf("%s: Today() at 00:30 on the anniversary = %s", zone.name, got)
		}
		if got := ask(onIt, "Y"); got != "19" {
			t.Errorf("%s: age at 00:30 on the anniversary = %s, want 19 (UTC instant is %s)",
				zone.name, got, onIt.UTC().Format("2006-01-02 15:04"))
		}

		// Half past eleven the night before: still not the birthday, even where
		// the UTC instant has already rolled over.
		nightBefore := time.Date(2019, 5, 31, 23, 30, 0, 0, loc)
		if got := ask(nightBefore, "T"); got != "2019-05-31" {
			t.Errorf("%s: Today() at 23:30 the night before = %s", zone.name, got)
		}
		if got := ask(nightBefore, "Y"); got != "18" {
			t.Errorf("%s: age at 23:30 the night before = %s, want 18 (UTC instant is %s)",
				zone.name, got, nightBefore.UTC().Format("2006-01-02 15:04"))
		}
	}
}
