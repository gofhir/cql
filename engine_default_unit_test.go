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

// TestCollapsePerIsIgnored asserts a defect this change did not cause and does not
// fix, found while checking that `collapse` reaches the check above.
//
// `collapse … per` is accepted and then ignored: intervals separated by less than
// the step should merge, and none of these do.
//
//	collapse {Interval[1, 3], Interval[5, 7]} per 3   should be {Interval[1, 7]}
//	                                                  is two intervals
//
// The gap between 3 and 5 is 2, which is inside a step of 3. Identical on main —
// the evaluator's collapse never looks at Per at all — so it is untouched here and
// gets its own change. Asserted rather than described, with the no-per spelling
// beside it: if a `per` ever starts making a difference, this fails.
func TestCollapsePerIsIgnored(t *testing.T) {
	withoutPer := evalDefaultUnit(t, "collapse {Interval[1, 3], Interval[5, 7]}")
	for _, per := range []string{" per 3", " per 1", " per 10"} {
		got := evalDefaultUnit(t, "collapse {Interval[1, 3], Interval[5, 7]}"+per)
		if got != withoutPer {
			t.Errorf("collapse%s = %s but without per = %s — the step now makes a "+
				"difference. With a step of 3 the right answer is {Interval[1, 7]}, "+
				"since the gap between 3 and 5 is 2.", per, got, withoutPer)
		}
	}
	// The collapsing it does do is unaffected, which is what makes the above a
	// missing feature rather than a broken one.
	if got := evalDefaultUnit(t, "collapse {Interval[1, 4], Interval[3, 7]}"); got != "{Interval[1, 7]}" {
		t.Errorf("collapsing two overlapping intervals = %s, want {Interval[1, 7]}", got)
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
}
