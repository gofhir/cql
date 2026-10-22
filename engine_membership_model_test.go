package cql

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/gofhir/cql/internal/referenceharness/answers"
)

// The overloads a membership can take, as the reference translator names the
// node it writes for each (CQL 1.5.3, List and Interval Operators).
const (
	nodeIn         = "In"         // element in a list or a point in an interval
	nodeContains   = "Contains"   // the same, from the container's side
	nodeIncludedIn = "IncludedIn" // a list in a list, or an interval in one
	nodeIncludes   = "Includes"   // the same, from the container's side
	nodeRejected   = "rejected"   // the translator resolves no overload
)

// membershipModelCase is a membership whose operands meet only through the FHIR
// model's conversions (backlog 1c11, plan 006). node is the overload the Java
// translator 3.26.0 writes for the expression alone in its library, since its
// choice depends on the definitions before it (backlog 1c13). absent and present
// are the answers over testdata/membershipmodel/absent.json and present.json:
// the specification's rule for that overload where a side is null, the
// reference's answer where none is.
type membershipModelCase struct {
	expr            string
	node            string
	absent, present string
}

// membershipModelCases is testdata/membershipmodel/cases.cql, in its order.
var membershipModelCases = []membershipModelCase{
	// An element of a model type, a list of System ones: List In and Contains.
	// "If the first argument is null, the result is true if the list contains
	// any null elements, and false otherwise."
	{"Patient.gender included in { 'male', 'female' }", nodeIn, "false", "true"},
	{"{ 'male', 'female' } includes Patient.gender", nodeContains, "false", "true"},
	{"Patient.gender during { 'male', 'female' }", nodeIn, "false", "true"},
	{"Patient.gender included in { 'female', null }", nodeIn, "true", "false"},
	// A null list and a null element: the container decides (ADR 003, decision 6).
	{"Patient.gender included in (null as List<String>)", nodeIn, "false", "false"},
	// A list of model elements: Patient.name.given is a List<FHIR.string>.
	{"Patient.name.given includes (null as String)", nodeContains, "false", "false"},
	{"(null as String) included in Patient.name.given", nodeIn, "false", "false"},
	// Two lists: List Includes and Included In, "If either argument is null, the
	// result is null". Java answers false over present.json, its known defect.
	// The List<Any> one is no overload here (Any meets only Any, backlog 1c8),
	// and the operator answers null as before: the same answer.
	{"Patient.name.given includes (null as List<String>)", nodeIncludes, "null", "null"},
	{"Patient.name.given includes (null as List<Any>)", nodeIncludes, "null", "null"},
	{"(null as List<String>) included in Patient.name.given", nodeIncludedIn, "null", "null"},
	// A choice element: Patient.deceased is boolean or dateTime.
	{"Patient.deceased included in { true }", nodeIn, "false", "true"},
	{"{ true } includes Patient.deceased", nodeContains, "false", "true"},
	// A FHIR.Period container, which converts to Interval<DateTime>: Interval In
	// and Contains. A null interval contains no point (In: "If the second
	// argument is null, the result is false"); a null point in an interval is
	// null (Contains: "If the second argument is null, the result is null").
	// Over present.json, with no null, the FHIR.dateTime instance reaches the
	// comparison unconverted and fails (backlog 1c16, on main too): Java's true
	// is not asserted there.
	{"FHIR.dateTime { value: @2020-01-02T00:00:00Z } during First([Encounter]).period", nodeIn, "false", ""},
	{"First([Encounter]).period includes FHIR.dateTime { value: @2020-01-02T00:00:00Z }", nodeContains, "false", ""},
	{"First([Encounter]).period includes (null as DateTime)", nodeContains, "false", "null"},
	{"(null as DateTime) during First([Encounter]).period", nodeIn, "false", "null"},
	{"(null as DateTime) included in First([Encounter]).period", nodeIn, "false", "null"},
	// A choice point, Observation.effective, against a converted Period.
	{"First([Observation]).effective during First([Encounter]).period", nodeIn, "false", "true"},
	{"First([Observation]).effective included in First([Encounter]).period", nodeIn, "false", "true"},
	{"First([Encounter]).period includes First([Observation]).effective", nodeContains, "false", "true"},
	// A choice container. present.json holds a dateTime, not the Period branch:
	// the engine answers on the branch before the overload (onAnotherBranch,
	// out of plan 006), so its answer there is not this increment's.
	{"First([Observation]).effective includes (null as DateTime)", nodeContains, "false", ""},
	// A choice element against a list of System quantities.
	{"First([Observation]).value included in { 5 'mg' }", nodeIn, "false", "true"},
	{"{ 5 'mg' } includes First([Observation]).value", nodeContains, "false", "true"},
	// An interval in a list of intervals: the element overload, a null list.
	{"First([Encounter]).period included in (null as List<Interval<DateTime>>)", nodeIn, "false", "false"},
	// Two conversions chained, a model one and a System one: the translator
	// resolves none, and this engine decides none (backlog 1c15).
	{"Patient.birthDate during (null as FHIR.Period)", nodeRejected, "null", ""},
	{"{ 1.0, 2.0 } includes Patient.multipleBirth", nodeRejected, "null", "null"},
	// A list of FHIR.string whose element has no value: the translator converts
	// each element, and the one with no value is null, so Java answers true.
	// This engine keeps the element as it is, and answers false, in contains as
	// in includes (backlog 1c14).
	{"List<FHIR.string>{ FHIR.string { value: null as String } } includes (null as String)", nodeContains, "false", "false"},
	{"List<FHIR.string>{ FHIR.string { value: null as String } } contains (null as String)", nodeContains, "false", "false"},
	// Any meets Any: the element overload, and the list-list one.
	{"List<Any>{ null } includes (null as Any)", nodeContains, "true", "true"},
	{"List<Any>{ 1 } includes (null as List<Any>)", nodeIncludes, "null", "null"},
	// A choice with a list among its branches, against a list of Strings: the
	// String branch is the element overload, the list one the list-list
	// overload, and the translator writes Contains (review round 1, Codex).
	{"{ 'a', null } includes (null as Choice<List<FHIR.string>, String>)", nodeContains, "true", "true"},
	// A choice inside the container's elements (review round 1, Codex).
	{"(null as System.Quantity) included in (null as List<Choice<FHIR.string, FHIR.Quantity>>)", nodeIn, "false", "false"},
	{"First([Observation]).component.value includes (null as System.Quantity)", nodeContains, "false", "false"},
	{"(null as FHIR.Quantity) included in (null as List<Choice<System.Quantity, String>>)", nodeIn, "false", "false"},
	// Intervals convert point by point (review round 1, Fable).
	{"(null as Interval<Date>) included in (null as List<Interval<DateTime>>)", nodeIn, "false", "false"},
	// A choice element whose branches select different overloads against a
	// FHIR.Range, which converts to Interval<Quantity>: the Quantity branch is a
	// point, the Range one an interval. The element overload wins (ADR 003,
	// decision 7), and a null interval contains no point (Interval In). The
	// translator writes the interval one, and answers null.
	{"First([Observation]).value during (null as FHIR.Range)", nodeIncludedIn, "false", "false"},
	{"First([Observation]).value included in (null as FHIR.Range)", nodeIncludedIn, "false", "false"},
	{"(null as FHIR.Range) includes First([Observation]).value", nodeIncludes, "false", "false"},
}

// membershipModelDiffers names where an answer above is not Java's, and why.
var membershipModelDiffers = map[string]string{
	"D7/present":  "List Includes with a null is null; Java answers false",
	"D8/present":  "List Includes with a null is null; Java answers false",
	"D9/present":  "List Included In with a null is null; Java answers false",
	"D24/absent":  "the translator refuses it; this engine decides no overload (backlog 1c15)",
	"D25/absent":  "the translator refuses it; this engine decides no overload (backlog 1c15)",
	"D25/present": "the translator refuses it; this engine decides no overload (backlog 1c15)",
	"D26/absent":  "the list's element is not converted (backlog 1c14)",
	"D26/present": "the list's element is not converted (backlog 1c14)",
	"D27/absent":  "the list's element is not converted (backlog 1c14)",
	"D27/present": "the list's element is not converted (backlog 1c14)",
	"D28/absent":  "List Contains of a null in a list that holds one is true; Java answers null",
	"D28/present": "List Contains of a null in a list that holds one is true; Java answers null",
	"D29/absent":  "List Includes with a null is null; Java answers false",
	"D29/present": "List Includes with a null is null; Java answers false",
	"D35/absent":  "the element overload wins over the interval one (ADR 003, decision 7); Java answers null",
	"D35/present": "the element overload wins over the interval one (ADR 003, decision 7); Java answers null",
	"D36/absent":  "the element overload wins over the interval one (ADR 003, decision 7); Java answers null",
	"D36/present": "the element overload wins over the interval one (ADR 003, decision 7); Java answers null",
	"D37/absent":  "the element overload wins over the interval one (ADR 003, decision 7); Java answers null",
	"D37/present": "the element overload wins over the interval one (ADR 003, decision 7); Java answers null",
}

var membershipModelDefine = regexp.MustCompile(`(?m)^define (D\d+): (.*)$`)

// TestMembershipThroughTheModelFollowsTheSpecification evaluates every case
// over both data files, and checks the table against what the reference wrote:
// the overload in its ELM and, where the table does not name a difference, its
// answer.
//
// Regenerate the reference files, from internal/referenceharness, with
// FHIRHelpers-4.0.1.cql copied beside the cases:
//
//	docker run --rm -v "$PWD":/w -v <dir>:/cql -w /w maven:3-eclipse-temurin-21 \
//	  java -cp "out:lib/*" Evaluate -each /cql/cases.cql /cql /cql/absent.json /cql/cases.elm.json > absent.java.json
//
// and the same with present.json.
func TestMembershipThroughTheModelFollowsTheSpecification(t *testing.T) {
	dir := filepath.Join("testdata", "membershipmodel")
	src, err := os.ReadFile(filepath.Join(dir, "cases.cql"))
	if err != nil {
		t.Fatal(err)
	}
	defines := membershipModelDefine.FindAllStringSubmatch(string(src), -1)
	if len(defines) != len(membershipModelCases) {
		t.Fatalf("cases.cql has %d definitions; want %d", len(defines), len(membershipModelCases))
	}
	header := string(src[:membershipModelDefine.FindStringIndex(string(src))[0]])
	nodes := membershipModelNodes(t, filepath.Join(dir, "cases.elm.json"))
	for i, tt := range membershipModelCases {
		name := fmt.Sprintf("D%d", i)
		if defines[i][1] != name || defines[i][2] != tt.expr {
			t.Fatalf("cases.cql %s is %q; want %q", defines[i][1], defines[i][2], tt.expr)
		}
		if got := nodes[name]; got != tt.node {
			t.Errorf("%s: the translator writes %s; the table says %s", tt.expr, got, tt.node)
		}
	}
	for _, data := range []string{"absent", "present"} {
		provider, patient := membershipModelData(t, filepath.Join(dir, data+".json"))
		reference := membershipModelReference(t, filepath.Join(dir, data+".java.json"))
		engine := NewEngine(WithDataProvider(provider))
		for i, tt := range membershipModelCases {
			name := fmt.Sprintf("D%d", i)
			want := tt.absent
			if data == "present" {
				want = tt.present
			}
			if want == "" {
				continue
			}
			key := name + "/" + data
			if _, named := membershipModelDiffers[key]; (reference[name] != want) != named {
				t.Errorf("%s over %s: Java answers %s, the table %s; a named difference: %t", tt.expr, data, reference[name], want, named)
			}
			v, err := engine.EvaluateExpression(context.Background(), header+"define "+name+": "+tt.expr+"\n", name, patient, nil)
			if err != nil {
				t.Errorf("%s over %s: %v", tt.expr, data, err)
				continue
			}
			if got := answers.Value(v).Text; got != want {
				t.Errorf("%s over %s = %s, want %s (%s)", tt.expr, data, got, want, tt.node)
			}
		}
	}
}

// membershipModelNodes reads the node the translator wrote for each definition,
// each translated in a library of its own.
func membershipModelNodes(t *testing.T, path string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var libraries map[string]*struct {
		Library struct {
			Statements struct {
				Def []struct {
					Name       string `json:"name"`
					Expression struct {
						Type string `json:"type"`
					} `json:"expression"`
				} `json:"def"`
			} `json:"statements"`
		} `json:"library"`
	}
	if err := json.Unmarshal(raw, &libraries); err != nil {
		t.Fatal(err)
	}
	nodes := map[string]string{}
	for name := range membershipModelCases {
		key := fmt.Sprintf("D%d", name)
		nodes[key] = nodeRejected
		lib := libraries[key]
		if lib == nil {
			continue
		}
		for _, def := range lib.Library.Statements.Def {
			if def.Name == key {
				nodes[key] = def.Expression.Type
			}
		}
	}
	return nodes
}

// membershipModelReference reads Java's answers as text: true, false, null, or
// "error" for a definition it did not translate.
func membershipModelReference(t *testing.T, path string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Results map[string]json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for name, r := range file.Results {
		var v struct {
			Value *bool  `json:"value"`
			Error string `json:"error"`
		}
		switch {
		case string(r) == "null":
			out[name] = "null"
		case json.Unmarshal(r, &v) == nil && v.Error != "":
			out[name] = "error"
		case v.Value != nil:
			out[name] = fmt.Sprint(*v.Value)
		default:
			t.Fatalf("%s: %s is no Boolean", name, r)
		}
	}
	return out
}

// membershipModelData serves one data file's resources, and its patient.
func membershipModelData(t *testing.T, path string) (resources snapshotData, patient json.RawMessage) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	resources = snapshotData{}
	for kind, v := range data {
		if kind == "Patient" {
			continue
		}
		var list []json.RawMessage
		if err := json.Unmarshal(v, &list); err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		resources[kind] = list
	}
	return resources, data["Patient"]
}

// TestMembershipWithAnAnyOperandStaysUndecided documents a known loss (backlog
// 1c8): Any meets only Any here, and the translator reads a container of
// another type against an Any as the element overload, by subtype. A function
// body is decided by its declared operand types, not a call site's.
func TestMembershipWithAnAnyOperandStaysUndecided(t *testing.T) {
	const header = "library T version '1.0'\ndefine function F(x Any): { 'a' } includes x\n"
	for expr, want := range map[string]string{
		// The translator writes Contains, and List Contains of a null in a list
		// that holds none is false.
		"{ 1 } includes (null as Any)": "null",
		"F(null as String)":            "null",
	} {
		v, err := NewEngine().EvaluateExpression(context.Background(), header+"define R: "+expr+"\n", "R", nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := answers.Value(v).Text; got != want {
			t.Errorf("%s = %s, want %s", expr, got, want)
		}
	}
}

// TestMembershipInADefinitionReachedFirstFromACallIsDecided is a definition
// typed first from inside a call to a function with no declared return type,
// whose body is typed speculatively at each call site (review round 2 of plan
// 006, Codex). The overload is the definition's own, wherever it is declared:
// List Contains of a null in a list that holds none is false. Java 3.26.0
// answers false in both orders.
func TestMembershipInADefinitionReachedFirstFromACallIsDecided(t *testing.T) {
	const later = "define Later: { 'a' } includes (null as String)\n"
	const call = "define function F(): Later\ndefine Entry: F()\n"
	for _, src := range []string{call + later, later + call} {
		v, err := NewEngine().EvaluateExpression(context.Background(), "library T version '1.0'\n"+src, "Entry", nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := answers.Value(v).Text; got != "false" {
			t.Errorf("%q: Entry = %s, want false", src, got)
		}
	}
}
