package cql

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"strings"
	"testing"

	"github.com/gofhir/cql/elm"
	"github.com/gofhir/cql/eval"
	"github.com/gofhir/cql/internal/referenceharness/answers"
)

// The rules of CQL 1.5.3 (cql.hl7.org/09-b-cqlreference.html) that decide a
// membership with a null, quoted where each case cites them.
const (
	// List Operators, In: "If the first argument is null, the result is true if
	// the list contains any null elements, and false otherwise. If the second
	// argument is null, the result is false."
	listIn = "List In"
	// List Operators, Contains: "If the first argument is null, the result is
	// false. If the second argument is null, the result is true if the list
	// contains any null elements, and false otherwise."
	listContains = "List Contains"
	// List Operators, Includes and Included In: "For the list-singleton
	// overload, this operator is a synonym for the contains [in] operator."
	listSynonym = "List Includes/Included In, list-singleton overload"
	// List Operators, Includes and Included In: "For the list-list overload, if
	// either argument is null, the result is null."
	listList = "List Includes/Included In, list-list overload"
	// Interval Operators, Contains: "If the first argument is null, the result
	// is false. If the second argument is null, the result is null."
	intervalContains = "Interval Contains"
	// Interval Operators, In (and during): "If the first argument is null, the
	// result is null. If the second argument is null, the result is false."
	intervalIn = "Interval In"
	// Interval Operators, Includes and Included In: "For the point-interval
	// overload, this operator is a synonym for the contains [in] operator."
	intervalSynonym = "Interval Includes/Included In, point-interval overload"
	// Both arguments null, where the two sentences above disagree: the null
	// container decides (docs/adr/003-lo-que-la-especificacion-no-decide.md,
	// decision 6).
	containerDecides = "ADR 003, decision 6"
)

// membershipCase is a membership the specification decides, by the rule it
// cites.
type membershipCase struct {
	expr, want, rule string
}

// membershipCases are the probes of plan 005 (21-10-2026). Each answer is the
// specification's, and both the Java engine 3.26.0 and cql-execution 3.3.2 give
// it, but for the two list-list cases, where Java answers false: there
// cql-execution and the conformance suite (IncludesNullRight) give null.
var membershipCases = []membershipCase{
	{"(null as Integer) in { 1, 2 }", "false", listIn},
	{"{ 1, 2 } contains (null as Integer)", "false", listContains},
	{"null in { 1, 2 }", "false", listIn},
	{"{ 1, 2 } contains null", "false", listContains},
	{"(null as Integer) in { 1, null }", "true", listIn},
	{"{ 1, null } contains (null as Integer)", "true", listContains},
	{"(null as Integer) in ({} as List<Integer>)", "false", listIn},
	{"({} as List<Integer>) contains (null as Integer)", "false", listContains},
	{"{} contains null", "false", listContains},
	{"null in {}", "false", listIn},
	{"{ 1, 2 } includes (null as Integer)", "false", listSynonym},
	{"(null as Integer) included in { 1, 2 }", "false", listSynonym},
	{"{ 1, null } includes (null as Integer)", "true", listSynonym},
	{"(null as Integer) included in { 1, null }", "true", listSynonym},
	{"1 in (null as List<Integer>)", "false", listIn},
	{"(null as List<Integer>) contains 1", "false", listContains},
	{"(null as List<Integer>) contains (null as Integer)", "false", listContains},
	{"(null as Integer) in (null as List<Integer>)", "false", listIn},
	{"(null as String) in { 'a' }", "false", listIn},
	{"(null as Code) in { Code { code: 'x' } }", "false", listIn},
	{"(null as Interval<Integer>) in { Interval[1, 2] }", "false", listIn},
	{"{ Interval[1, 2] } contains (null as Interval<Integer>)", "false", listContains},
	{"(null as Date) in { @2020-01-01 }", "false", listIn},
	{"{ @2020-01-01 } contains (null as Date)", "false", listContains},
	{"(null as Tuple { a Integer }) in { Tuple { a: 1 } }", "false", listIn},
	{"{ { 1 } } contains (null as List<Integer>)", "false", listContains},
	{"not ((null as Integer) in { 1, 2 })", "true", listIn},
	{"(null as Interval<Integer>) contains (null as Integer)", "false", containerDecides},
	{"(null as Interval<Integer>) contains 1", "false", intervalContains},
	{"(null as Integer) in (null as Interval<Integer>)", "false", containerDecides},
	{"1 in (null as Interval<Integer>)", "false", intervalIn},
	{"Interval[1, 2] contains (null as Integer)", "null", intervalContains},
	{"(null as Integer) in Interval[1, 2]", "null", intervalIn},
	{"(null as List<Integer>) includes (null as Integer)", "false", listSynonym},
	{"(null as Integer) included in (null as List<Integer>)", "false", listSynonym},
	{"{ (null as Integer) } contains (null as Integer)", "true", listContains},
	{"List<Integer>{} contains (null as Integer)", "false", listContains},
	{"{ 's', 'a', 'm' } includes (null as List<String>)", "null", listList},
	{"{ 's', 'a', 'm' } includes (null as String)", "false", listSynonym},
	{"(null as List<String>) included in { 's' }", "null", listList},
	{"(null as String) included in { 's' }", "false", listSynonym},
	{"(null as List<Integer>) includes 1", "false", listSynonym},
	{"1 included in (null as List<Integer>)", "false", listSynonym},
	{"{ { 1 } } includes (null as List<Integer>)", "false", listSynonym},
	{"(null as List<Integer>) included in { { 1 } }", "false", listSynonym},
	{"(null as Interval<Integer>) includes 1", "false", intervalSynonym},
	{"1 included in (null as Interval<Integer>)", "false", intervalSynonym},
	{"1 during (null as Interval<Integer>)", "false", intervalSynonym},
	{"(null as Integer) during { 1, 2 }", "false", listSynonym},
	{"{ Interval[1, null] } contains (null as Interval<Integer>)", "false", listContains},
	{"{ Tuple { a: null } } contains (null as Tuple { a Integer })", "false", listContains},
	{"{ { null as Integer } } contains (null as List<Integer>)", "false", listContains},
	{"First(List<Integer>{}) in { 1, 2 }", "false", listIn},
	{"NullList includes (null as Integer)", "false", listSynonym},
	{"(null as List<String>) included in { 'b', null }", "null", listList},
	{"{ 'a', null } includes (null as List<String>)", "null", listList},
	// An element of a type that converts implicitly to the list's or the
	// interval's, which the translator resolves to In and Contains.
	{"(null as Integer) included in List<Decimal>{ 1.0 }", "false", listSynonym},
	{"(null as Integer) included in List<Decimal>{ 1.0, null }", "true", listSynonym},
	{"List<Decimal>{ 1.0 } includes (null as Integer)", "false", listSynonym},
	{"1 included in (null as Interval<Decimal>)", "false", intervalSynonym},
	{"(null as Integer) during (null as Interval<Decimal>)", "false", containerDecides},
	{"(null as Date) included in (null as Interval<DateTime>)", "false", containerDecides},
	{"(null as Interval<DateTime>) includes day of @2020-01-01T00:00:00", "false", intervalSynonym},
	// The container's type converts to the element's (review round 2, Codex),
	// also inside a list.
	{"(null as Decimal) included in List<Integer>{ 1, 2 }", "false", listSynonym},
	{"(null as Decimal) included in List<Integer>{ 1, null }", "true", listSynonym},
	{"(null as List<Integer>) includes (null as Decimal)", "false", listSynonym},
	{"(null as List<Integer>) included in List<List<Decimal>>{ { 1.0 } }", "false", listSynonym},
	{"(null as List<Integer>) included in List<List<Decimal>>{ { 1.0 }, null }", "true", listSynonym},
	{"List<List<Decimal>>{ { 1.0 } } includes (null as List<Integer>)", "false", listSynonym},
	{"(null as Decimal) during (null as Interval<Integer>)", "false", containerDecides},
	{"{ 1, 2 } includes (null as Decimal)", "false", listSynonym},
	{"(null as DateTime) included in { @2020-01-01 }", "false", listSynonym},
	{"1.0 included in (null as Interval<Integer>)", "false", intervalSynonym},
	{"(null as Interval<Date>) includes @2020-01-01T00:00:00", "false", intervalSynonym},
	// The list-list overload with one list widened to the other's: in a library
	// of its own, the translator writes IncludedIn either way (review round 3).
	{"(null as List<Integer>) included in List<Decimal>{ 1.0, null }", "null", listList},
	{"(null as List<Decimal>) included in List<Integer>{ 1, null }", "null", listList},
}

// membershipLibrary declares what the cases name beyond literals.
const membershipLibrary = "library Membership version '1.0'\n" +
	"define NullList: null as List<Integer>\n"

// TestMembershipWithNullFollowsTheSpecification holds in, contains, includes,
// included in and during with a null to the rule each case cites.
func TestMembershipWithNullFollowsTheSpecification(t *testing.T) {
	engine := NewEngine()
	for _, tt := range membershipCases {
		t.Run(tt.expr, func(t *testing.T) {
			src := membershipLibrary + "define R: " + tt.expr + "\n"
			v, err := engine.EvaluateExpression(context.Background(), src, "R", nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := answers.Value(v).Text; got != tt.want {
				t.Errorf("%s = %s, want %s (%s)", tt.expr, got, tt.want, tt.rule)
			}
		})
	}
}

// TestMembershipWithNullFromTheReferenceELM evaluates the cases from the ELM
// the Java translator 3.26.0 writes for testdata/membership/cases.cql, which
// resolves each overload itself: In and Contains for the membership one,
// Includes and IncludedIn for the list-list one. It is the path of an imported
// library, which has no static types.
//
// Regenerate the ELM, from internal/referenceharness:
//
//	docker run --rm -v "$PWD":/w -v "$PWD/../../testdata/membership":/cql -w /w maven:3.10.0-eclipse-temurin-25 \
//	  sh -c 'java -cp "out:lib/*" Translate /cql/cases.cql /cql > /cql/cases.elm.json'
func TestMembershipWithNullFromTheReferenceELM(t *testing.T) {
	src, err := os.ReadFile("testdata/membership/cases.cql")
	if err != nil {
		t.Fatal(err)
	}
	var want strings.Builder
	want.WriteString("library Membership version '1.0'\n\ndefine NullList: null as List<Integer>\n")
	for i, tt := range membershipCases {
		fmt.Fprintf(&want, "define D%d: %s\n", i, tt.expr)
	}
	if string(src) != want.String() {
		t.Fatal("testdata/membership/cases.cql is not the cases of membershipCases; rewrite it and its ELM")
	}
	raw, err := os.ReadFile("testdata/membership/cases.elm.json")
	if err != nil {
		t.Fatal(err)
	}
	// The translator writes the library inside a {"library": …} envelope. Each
	// definition is imported on its own, beside NullList: the ELM reader does not
	// take every node the translator writes (backlog 4), and the ones it refuses
	// are named below rather than failing the rest.
	var envelope struct {
		Library map[string]json.RawMessage `json:"library"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	var statements struct {
		Def []json.RawMessage `json:"def"`
	}
	if err := json.Unmarshal(envelope.Library["statements"], &statements); err != nil {
		t.Fatal(err)
	}
	if len(statements.Def) != len(membershipCases)+1 {
		t.Fatalf("the ELM has %d definitions; want NullList and %d cases", len(statements.Def), len(membershipCases))
	}
	// The ELM reader refuses the Code and Tuple selectors the translator writes,
	// and the Query it writes to convert a list (D63 to D68, D70, D71), whose
	// source is an array; it reads its Date constructor as null, for it turns a node it does not
	// know into a null literal (elm/importer.go): `{ @2020-01-01 }` holds a null
	// there. Backlog 4 and 1c9; each is named so that its fix shows here.
	//
	// Without static types, an imported Includes or IncludedIn of two lists
	// cannot be told from the membership overload when one side is null, and a
	// list holding a null answers true where the list-list overload is null
	// (backlog 1c8): D54 and D55, named with the misread ones. D74 and D75 are
	// misread by the translator: in this library it writes them as In(Any,
	// List<Any>), and in one of their own as IncludedIn (review round 3).
	refused := map[string]bool{"D19": true, "D24": true, "D50": true,
		"D63": true, "D64": true, "D65": true, "D66": true, "D67": true, "D68": true, "D70": true, "D71": true}
	misread := map[string]bool{"D22": true, "D23": true, "D54": true, "D55": true, "D74": true, "D75": true}
	var unread []string
	for i, tt := range membershipCases {
		name := fmt.Sprintf("D%d", i)
		lib := maps.Clone(envelope.Library)
		defs, err := json.Marshal(map[string][]json.RawMessage{"def": {statements.Def[0], statements.Def[i+1]}})
		if err != nil {
			t.Fatal(err)
		}
		lib["statements"] = defs
		one, err := json.Marshal(lib)
		if err != nil {
			t.Fatal(err)
		}
		read, err := elm.UnmarshalLibrary(one)
		if err != nil {
			unread = append(unread, name)
			continue
		}
		imported, err := elm.Import(read)
		if err != nil {
			t.Fatalf("%s: %v", tt.expr, err)
		}
		v, err := eval.NewEvaluator(eval.NewContext(context.Background(), imported)).EvaluateExpression(name)
		if err != nil {
			t.Errorf("%s: %v", tt.expr, err)
			continue
		}
		if got := answers.Value(v).Text; (got == tt.want) == misread[name] {
			t.Errorf("%s from ELM = %s, want %s (%s); misread by the ELM reader: %t", tt.expr, got, tt.want, tt.rule, misread[name])
		}
	}
	for _, name := range unread {
		if !refused[name] {
			t.Errorf("the ELM reader refuses %s", name)
		}
	}
	if len(unread) != len(refused) {
		t.Errorf("the ELM reader refuses %v; want %d of them", unread, len(refused))
	}
}

// TestMembershipWithNullThroughOtherPaths reaches the null by the paths that
// are not a literal: a parameter, a function, an included library and a query
// (List In, List Contains, and their synonyms). A String is not a list, and
// CQL has no contains over one: the check refuses it rather than reading it as
// a membership.
func TestMembershipWithNullThroughOtherPaths(t *testing.T) {
	const included = "library H version '1.0'\n" +
		"define N: null as Integer\n" +
		"define L: null as List<Integer>\n"
	engine := NewEngine(WithLibraryResolver(func(_ context.Context, _, _ string) (string, error) {
		return included, nil
	}))
	const header = "library T version '1.0'\n" +
		"include H version '1.0' called H\n" +
		"parameter P Integer default null\n" +
		"parameter PL List<Integer> default null\n" +
		"define function NullOf(): null as Integer\n"
	for _, tt := range []membershipCase{
		{"P in { 1, 2 }", "false", listIn},
		{"{ 1, 2 } contains P", "false", listContains},
		{"PL includes (null as Integer)", "false", listSynonym},
		{"NullOf() in { 1, 2 }", "false", listIn},
		{"H.N in { 1, 2 }", "false", listIn},
		{"Count(({ 1, 2 }) X where not ((null as Integer) in { X }))", "Integer 2", listIn},
		// properly is not this increment: the answer is this engine's and Java's
		// before and after it, kept so that routing it here shows.
		{"List<Integer>{ null } properly includes (null as Integer)", "false", "List Properly Includes, untouched by plan 005"},
		{"({ 1 }) X return (null as Integer) in { X }", "{false}", listIn},
	} {
		t.Run(tt.expr, func(t *testing.T) {
			v, err := engine.EvaluateExpression(context.Background(), header+"define R: "+tt.expr+"\n", "R", nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := answers.Value(v).Text; got != tt.want {
				t.Errorf("%s = %s, want %s (%s)", tt.expr, got, tt.want, tt.rule)
			}
		})
	}
}

// TestMembershipWithAnUntypedNullStaysUndecided documents a known loss: a
// name qualified by an included library has no static type (sema types it
// Unknown), so a null list behind it cannot be told from a null element, and
// includes and included in answer null where the list-singleton overload is
// false (plan 005, backlog 1c8). A null element needs no type, and is decided.
func TestMembershipWithAnUntypedNullStaysUndecided(t *testing.T) {
	engine := NewEngine(WithLibraryResolver(func(_ context.Context, _, _ string) (string, error) {
		return "library H version '1.0'\ndefine L: null as List<Integer>\n", nil
	}))
	const header = "library T version '1.0'\ninclude H version '1.0' called H\n"
	for expr, want := range map[string]string{
		"H.L includes (null as Integer)":    "null",
		"(null as Integer) included in H.L": "null",
		"H.L contains (null as Integer)":    "false",
		"(null as Integer) in H.L":          "false",
	} {
		v, err := engine.EvaluateExpression(context.Background(), header+"define R: "+expr+"\n", "R", nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := answers.Value(v).Text; got != want {
			t.Errorf("%s = %s, want %s", expr, got, want)
		}
	}
}

func TestAStringIsNoListToBeIn(t *testing.T) {
	_, err := NewEngine().EvaluateExpression(context.Background(),
		"library T version '1.0'\ndefine R: 'abc' contains (null as String)\n", "R", nil, nil)
	if err == nil {
		t.Error("'abc' contains (null as String) evaluated; want the check to refuse it")
	}
}

// TestMembershipWithNullInAFHIRList asks for a null in a FHIR list of no, one
// and several elements: none of them holds a null, so the answer is false
// (List Contains, List In).
func TestMembershipWithNullInAFHIRList(t *testing.T) {
	const header = "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1'\n" +
		"context Patient\n"
	engine := NewEngine()
	for _, patient := range []string{
		`{"resourceType":"Patient","id":"none"}`,
		`{"resourceType":"Patient","id":"one","name":[{"given":["Ana"]}]}`,
		`{"resourceType":"Patient","id":"several","name":[{"given":["Ana","Luz"]},{"given":["Eva"]}]}`,
	} {
		for _, expr := range []string{
			"Patient.name.given contains (null as FHIR.string)",
			"(null as FHIR.string) in Patient.name.given",
			"Patient.name.given includes (null as FHIR.string)",
			// The model converts a FHIR.Period to an interval: Interval
			// Includes and Included In, point overload, container null.
			"(null as FHIR.Period) includes @2020-01-01T00:00:00",
			"@2020-01-01T00:00:00 during (null as FHIR.Period)",
		} {
			v, err := engine.EvaluateExpression(context.Background(), header+"define R: "+expr+"\n", "R", []byte(patient), nil)
			if err != nil {
				t.Fatalf("%s over %s: %v", expr, patient, err)
			}
			if got := answers.Value(v).Text; got != "false" {
				t.Errorf("%s over %s = %s, want false", expr, patient, got)
			}
		}
	}
}
