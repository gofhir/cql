package cql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gofhir/cql/eval"
)

// oneBranchProvider serves a single Observation whose value[x] takes the branch
// the test names, which is how the eleven branches are told apart.
type oneBranchProvider struct{ field, value string }

func (p oneBranchProvider) Retrieve(_ context.Context, r eval.RetrieveRequest) ([]json.RawMessage, error) {
	if r.ResourceType != "Observation" {
		return nil, nil
	}
	return []json.RawMessage{json.RawMessage(
		`{"resourceType":"Observation","id":"o","status":"final","` + p.field + `":` + p.value + `}`)}, nil
}

func evalOnBranch(t *testing.T, field, value, expr string) string {
	t.Helper()
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\n" +
		"include FHIRHelpers version '4.0.1' called FHIRHelpers\ncontext Patient\ndefine A: " + expr + "\n"
	got, err := NewEngine(WithDataProvider(oneBranchProvider{field, value}), WithEvaluationTimestamp(
		time.Date(2019, 6, 1, 12, 0, 0, 0, time.UTC))).
		EvaluateExpression(context.Background(), src, "A",
			[]byte(`{"resourceType":"Patient","id":"p1"}`), nil)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	if got == nil {
		return "null"
	}
	return got.String()
}

// branchCase is one branch of Observation.value and a literal of that branch's
// own type.
var branchCases = []struct{ name, field, value, literal string }{
	{"Quantity", "valueQuantity", `{"value":190,"unit":"mg/dL"}`, "190 'mg/dL'"},
	{"CodeableConcept", "valueCodeableConcept",
		`{"coding":[{"system":"http://loinc.org","code":"x"}]}`,
		"ToConcept(Code { code: 'x', system: 'http://loinc.org' })"},
	{"string", "valueString", `"abc"`, "'abc'"},
	{"integer", "valueInteger", `190`, "190"},
	{"boolean", "valueBoolean", `true`, "true"},
	{"dateTime", "valueDateTime", `"2019-06-01T10:00:00Z"`, "@2019-06-01T10:00:00Z"},
}

// TestAComparisonAsksAboutOneBranchOfAChoice covers the whole of the rule, across
// every branch and both kinds of comparison.
//
// FHIR stores Observation.value as one of eleven types, and a comparison names
// which one it means by what it compares against. The reference translator writes
// that down as `as Quantity`, and `as` is null for a value on another branch — so
// a row whose value is a CodeableConcept drops out of a query about quantities
// rather than failing it or being counted as unequal.
//
// The engine gave that question three different answers depending on the branch,
// and the table below is what it looked like. Off the diagonal:
//
//	branch           `=` against another branch   `>=` against another branch
//	Quantity         false                        null
//	CodeableConcept  false                        null
//	string           false                        error
//	integer          false                        error
//	boolean          false                        error
//	dateTime         false                        error
//
// A first attempt reached the first two rows only. It asked at evaluation whether
// the value was still raw FHIR JSON, and only Quantity and CodeableConcept are:
// `valueString`, `valueInteger`, `valueBoolean` and `valueDateTime` come out of
// the JSON as ordinary system values, indistinguishable from a literal. By
// evaluation the question has no answer, which is why it is asked of the phase
// that typed the operand instead.
func TestAComparisonAsksAboutOneBranchOfAChoice(t *testing.T) {
	for _, b := range branchCases {
		for _, other := range branchCases {
			for _, op := range []string{"=", "!=", ">=", "<="} {
				expr := fmt.Sprintf("First([Observation] O).value %s %s", op, other.literal)
				got := evalOnBranch(t, b.field, b.value, expr)
				if b.name == other.name {
					// The branch being asked about answers. `>=` over a Boolean is
					// the exception and not this rule's: CQL gives Booleans no
					// ordering, and `true >= true` fails for two literals too.
					if b.name == "boolean" && (op == ">=" || op == "<=") {
						continue
					}
					if got == "null" {
						t.Errorf("%s on the %s branch = null, want an answer", expr, b.name)
					}
					continue
				}
				if got != "null" {
					t.Errorf("the %s branch under `%s %s` = %s, want null — the comparison "+
						"is about the %s branch", b.name, op, other.literal, got, other.name)
				}
			}
		}
	}
}

// TestTheOperatorsAgreeAboutTheBranch is the property the three answers above
// violated, stated on its own: whether a row is being asked about cannot depend
// on which comparison operator asks.
//
//	not (O.value >= 190 'mg/dL')   1   the coded row was unanswerable already
//	not (O.value  = 190 'mg/dL')   2   and here it counted as "not equal"
//
// Over three rows — two quantities and a CodeableConcept — the coded one is not
// being asked about, so it is in neither count.
func TestTheOperatorsAgreeAboutTheBranch(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"Count([Observation] O where O.value >= 190 'mg/dL')", "1"},
		{"Count([Observation] O where not (O.value >= 190 'mg/dL'))", "1"},
		{"Count([Observation] O where O.value = 190 'mg/dL')", "1"},
		{"Count([Observation] O where not (O.value = 190 'mg/dL'))", "1"},
		{"Count([Observation] O where O.value != 190 'mg/dL')", "1"},
	} {
		if got := evalChoiceCompare(t, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — the coded row is in no count either way", tt.expr, got, tt.want)
		}
	}
}

// TestABranchStillConvertsWhereCQLSaysItDoes is the limit: the rule must fire on
// a branch the comparison is not about, and nowhere else.
//
// An implicit conversion keeps a branch in scope. `O.value = 190.0` over the
// integer branch is an Integer against a Decimal, which CQL converts, so the
// comparison is about that branch and answers. Reading the rule as "the type
// names differ" would have made it null and dropped a row the expression
// includes.
func TestABranchStillConvertsWhereCQLSaysItDoes(t *testing.T) {
	for _, tt := range []struct{ expr, want, why string }{
		{"First([Observation] O).value = 190.0", "true", "an Integer branch against a Decimal"},
		{"First([Observation] O).value >= 190.0", "true", "and under ordering too"},
		{"First([Observation] O).value < 200.0", "true", "the conversion is not one-way"},
	} {
		if got := evalOnBranch(t, "valueInteger", "190", tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — %s", tt.expr, got, tt.want, tt.why)
		}
	}

	// And a pair neither of which came from a choice is an authoring mistake, not
	// a narrowing that went wrong: it still fails rather than answering null.
	for _, expr := range []string{"1 'mg' >= 'abc'", "@2020-01-01 >= 1 'mg'"} {
		if got := evalChoiceCompare(t, expr); got == "null" {
			t.Errorf("%s = null, want it reported — nothing here was narrowed", expr)
		}
	}
}

// TestEveryMembershipSpellingAgreesAboutTheBranch covers the operators the first
// pass at this left out, and which CQL makes one question.
//
// `x in Interval[…]` is `x >= low and x <= high`; `contains` is the same operator
// with its operands the other way round; and the specification makes `during` a
// synonym of `included in`. So all four have to name the same branch as the
// comparisons do, and three of them raised an error where the comparisons had
// just been taught to decline — the same shape v1.20.1 removed from the four
// spellings of membership at a precision.
func TestEveryMembershipSpellingAgreesAboutTheBranch(t *testing.T) {
	const coded = `{"coding":[{"system":"http://loinc.org","code":"x"}]}`
	const rng = "Interval[100 'mg/dL', 200 'mg/dL']"

	// The coded branch is not what a range of quantities asks about.
	for _, expr := range []string{
		"First([Observation] O).value in " + rng,
		rng + " contains First([Observation] O).value",
		"First([Observation] O).value during " + rng,
		"First([Observation] O).value included in " + rng,
	} {
		if got := evalOnBranch(t, "valueCodeableConcept", coded, expr); got != "null" {
			t.Errorf("%s = %s, want null", expr, got)
		}
	}

	// And the branch it does ask about answers, in every spelling.
	for _, expr := range []string{
		"First([Observation] O).value in " + rng,
		rng + " contains First([Observation] O).value",
		"First([Observation] O).value during " + rng,
		"First([Observation] O).value included in " + rng,
	} {
		if got := evalOnBranch(t, "valueQuantity", `{"value":190,"unit":"mg/dL"}`, expr); got != "true" {
			t.Errorf("%s over the Quantity branch = %s, want true", expr, got)
		}
	}
}

// TestEquivalenceStillDecidesTheWrongBranch is the limit, and it is the one `~`
// takes everywhere: equivalence never returns null in CQL, so a value on another
// branch is simply not equivalent to what is being asked about.
//
// A first pass at this change had `~` declining with the rest, which made
// `O.value ~ 190 'mg/dL'` null. Neither of the two existing assertions about `~`
// caught it — both compare quantities of different dimensions rather than
// branches of a choice — so this one names the case.
func TestEquivalenceStillDecidesTheWrongBranch(t *testing.T) {
	const coded = `{"coding":[{"system":"http://loinc.org","code":"x"}]}`
	for _, tt := range []struct{ expr, want string }{
		{"First([Observation] O).value ~ 190 'mg/dL'", "false"},
		{"First([Observation] O).value !~ 190 'mg/dL'", "true"},
	} {
		if got := evalOnBranch(t, "valueCodeableConcept", coded, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — equivalence has no null to return", tt.expr, got, tt.want)
		}
	}
	// The branch it is about still answers true.
	if got := evalOnBranch(t, "valueQuantity", `{"value":190,"unit":"mg/dL"}`,
		"First([Observation] O).value ~ 190 'mg/dL'"); got != "true" {
		t.Errorf("the Quantity branch under `~` = %s, want true", got)
	}
}

// TestBetweenOverAChoiceIsStillRefused records what this change does not reach,
// asserted so that reaching it trips here.
//
// `between` over a choice element is refused by the semantic phase for *every*
// branch, the one being asked about included, so there is no narrowing for the
// evaluator to apply — the expression never gets that far. That is a defect in
// how `between` is typed rather than in which branch it means, and it is older
// than this change: `O.value between 100 'mg/dL' and 200 'mg/dL'` does not
// compile whether the value is a Quantity or a CodeableConcept.
func TestBetweenOverAChoiceIsStillRefused(t *testing.T) {
	const expr = "First([Observation] O).value between 100 'mg/dL' and 200 'mg/dL'"
	for _, b := range []struct{ field, value string }{
		{"valueQuantity", `{"value":190,"unit":"mg/dL"}`},
		{"valueCodeableConcept", `{"coding":[{"code":"x"}]}`},
	} {
		got := evalOnBranch(t, b.field, b.value, expr)
		if !strings.HasPrefix(got, "ERROR") {
			t.Errorf("`between` over the %s branch = %s — if it compiles now, it should name a "+
				"branch like `in` does, and this test should go", b.field, got)
		}
	}
}

// TestAListDoesNotDeclineTheWrongBranch is the limit on the membership rule, and
// the specification draws it on exactly this point:
//
//	in an interval: "If the first argument is null, the result is null."
//	in a list:      "If the first argument is null, the result is true if the
//	                 list contains any null elements, and false otherwise."
//
// So a branch a range is not about is unanswerable, and a branch a list does not
// hold is simply not in it. A first pass narrowed both, which made
// `O.value in {190 'mg/dL'}` null while `distinct` and `IndexOf` over the same
// pair still answered "two values" — the split v1.20.0 drew between a container
// that has a null to return and one that does not, broken on one side of it.
func TestAListDoesNotDeclineTheWrongBranch(t *testing.T) {
	const coded = `{"coding":[{"system":"http://loinc.org","code":"x"}]}`
	for _, tt := range []struct{ expr, want string }{
		{"First([Observation] O).value in {190 'mg/dL'}", "false"},
		{"{190 'mg/dL'} contains First([Observation] O).value", "false"},
		// The rest of the list family already answered "two values" and has to
		// keep agreeing with `in`.
		{"Count(distinct {First([Observation] O).value, 190 'mg/dL'})", "2"},
		{"IndexOf({190 'mg/dL'}, First([Observation] O).value)", "-1"},
	} {
		if got := evalOnBranch(t, "valueCodeableConcept", coded, tt.expr); got != tt.want {
			t.Errorf("%s = %s, want %s — a list has no null to return", tt.expr, got, tt.want)
		}
	}

	// And the interval keeps declining, which is the pair this is a limit on.
	if got := evalOnBranch(t, "valueCodeableConcept", coded,
		"First([Observation] O).value in Interval[100 'mg/dL', 200 'mg/dL']"); got != "null" {
		t.Errorf("the interval form = %s, want null", got)
	}
}

// TestTheBranchRuleCrossesTheLibraryBoundary covers what the plan reaches, which
// until this was only the library being evaluated.
//
// A plan is keyed by AST node, and an included library's nodes are not in the
// caller's. So every decision the semantic phase makes was invisible to code
// inside an included library, and the same comparison answered two ways:
//
//	the same expression, in the evaluated library   null
//	                     in an included library     error: cannot compare Concept
//
// Conversions survived it because coerceToSystem asks the model at evaluation and
// needs no plan. A narrowing has no such fallback — nothing at evaluation knows
// which branch a comparison is about — so it was the half that showed.
//
// resolveIncludesInto had each library's plan in hand and discarded it. It is kept
// now, and a scope built for a library is judged by its own.
func TestTheBranchRuleCrossesTheLibraryBoundary(t *testing.T) {
	const main = `library T version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
include Helper version '1.0' called H
context Patient
define A: H.IsHigh(First([Observation]))
`
	helper := func(body string) string {
		return "library Helper version '1.0'\nusing FHIR version '4.0.1'\n" +
			"include FHIRHelpers version '4.0.1' called FHIRHelpers\n" +
			"define function IsHigh(O FHIR.Observation): " + body + "\n"
	}
	run := func(t *testing.T, body, field, value string) string {
		t.Helper()
		resolve := func(_ context.Context, name, _ string) (string, error) {
			if name == "Helper" {
				return helper(body), nil
			}
			return "", fmt.Errorf("no library %q", name)
		}
		got, err := NewEngine(WithDataProvider(oneBranchProvider{field, value}),
			WithLibraryResolver(resolve)).
			EvaluateExpression(context.Background(), main, "A",
				[]byte(`{"resourceType":"Patient","id":"p1"}`), nil)
		if err != nil {
			return "ERROR: " + err.Error()
		}
		if got == nil {
			return "null"
		}
		return got.String()
	}

	// Every branch answers the same inside an included library as in the evaluated
	// one, which is the claim. The four plain-value branches are the ones that
	// raised errors, and they are also the ones a runtime test cannot recognize.
	for _, b := range branchCases {
		const expr = "O.value >= 190 'mg/dL'"
		inside := run(t, expr, b.field, b.value)
		outside := evalOnBranch(t, b.field, b.value, "First([Observation] O).value >= 190 'mg/dL'")
		if inside != outside {
			t.Errorf("the %s branch: inside an included library = %s, in the evaluated one = %s",
				b.name, inside, outside)
		}
	}

	// And the conversions the plan decides now apply there too, which is the other
	// half of what was invisible: this raised an error, because the cast's planned
	// ToQuantity never ran and arithmetic was handed raw FHIR JSON.
	if got := run(t, "(O.value as FHIR.Quantity) + 1 'mg' > 9 'mg'",
		"valueQuantity", `{"value":9.1,"unit":"mg"}`); got != "true" {
		t.Errorf("arithmetic over a cast choice element inside an included library = %s, want true", got)
	}
}

// TestTheImplicitNarrowingMatchesTheExplicitCast is the strongest statement this
// rule can be held to: writing out the cast the reference translator emits gives
// the same answer as the engine now gives without it.
//
// The reference types the choice, sees the comparison wants a Quantity and emits
// `as Quantity`, which is null for a value on another branch. That is where the
// rule comes from, so the two spellings agreeing is not a coincidence to arrange —
// it is the claim.
func TestTheImplicitNarrowingMatchesTheExplicitCast(t *testing.T) {
	for _, b := range []struct{ field, value, want string }{
		{"valueQuantity", `{"value":190,"unit":"mg/dL"}`, "true"},
		{"valueCodeableConcept", `{"coding":[{"code":"x"}]}`, "null"},
	} {
		for _, op := range []string{"=", ">=", "<="} {
			implicit := fmt.Sprintf("First([Observation] O).value %s 190 'mg/dL'", op)
			explicit := fmt.Sprintf("(First([Observation] O).value as FHIR.Quantity) %s 190 'mg/dL'", op)
			a := evalOnBranch(t, b.field, b.value, implicit)
			c := evalOnBranch(t, b.field, b.value, explicit)
			if a != c {
				t.Errorf("on the %s branch, `%s` = %s but the cast written out = %s",
					b.field, implicit, a, c)
			}
			if a != b.want {
				t.Errorf("on the %s branch, `%s` = %s, want %s", b.field, implicit, a, b.want)
			}
		}
	}
}

// TestTheChoiceMayBeOnEitherSide covers the symmetry, which the narrowing is
// recorded for on both operands and which nothing else here exercises.
func TestTheChoiceMayBeOnEitherSide(t *testing.T) {
	for _, b := range []struct{ field, value, want string }{
		{"valueQuantity", `{"value":190,"unit":"mg/dL"}`, "true"},
		{"valueCodeableConcept", `{"coding":[{"code":"x"}]}`, "null"},
	} {
		for _, op := range []string{"=", ">=", "<="} {
			expr := fmt.Sprintf("190 'mg/dL' %s First([Observation] O).value", op)
			if got := evalOnBranch(t, b.field, b.value, expr); got != b.want {
				t.Errorf("%s on the %s branch = %s, want %s", expr, b.field, got, b.want)
			}
		}
	}
}

// TestThePlanReachesEveryDepthOfTheIncludeGraph covers the depth, which is a
// separate property from crossing one boundary: a library included by an included
// library is two plans away from the one being evaluated.
//
// It works because each level is registered as the graph is walked, before the
// check that decides which libraries the top level gets an alias for. Measured
// against the previous code, where two levels deep raised an error for every
// branch a comparison is not about.
func TestThePlanReachesEveryDepthOfTheIncludeGraph(t *testing.T) {
	const deep = `library H2 version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
define function Deep(O FHIR.Observation): O.value >= 190 'mg/dL'
`
	const mid = `library H1 version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
include H2 version '1.0' called D
define function Mid(O FHIR.Observation): D.Deep(O)
`
	const main = `library T version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
include H1 version '1.0' called M
context Patient
define A: M.Mid(First([Observation]))
`
	resolve := func(_ context.Context, name, _ string) (string, error) {
		switch name {
		case "H1":
			return mid, nil
		case "H2":
			return deep, nil
		}
		return "", fmt.Errorf("no library %q", name)
	}
	for _, b := range branchCases {
		got, err := NewEngine(WithDataProvider(oneBranchProvider{b.field, b.value}),
			WithLibraryResolver(resolve)).
			EvaluateExpression(context.Background(), main, "A",
				[]byte(`{"resourceType":"Patient","id":"p1"}`), nil)
		if err != nil {
			t.Errorf("the %s branch, two includes deep: %v", b.name, err)
			continue
		}
		want := "null"
		if b.name == "Quantity" {
			want = "true"
		}
		out := "null"
		if got != nil {
			out = got.String()
		}
		if out != want {
			t.Errorf("the %s branch, two includes deep = %s, want %s", b.name, out, want)
		}
	}
}
