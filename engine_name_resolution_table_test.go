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

// A name applied to a value has three spellings in a sort clause, and the engine
// resolves all three. CQL allows exactly one of them:
//
//	"Because the sort applies after the query results have been determined,
//	 alias references are neither required nor allowed in the sort."
//	   — CQL, Author's Guide, Sorting
//
// so `sort by effective` is the conformant form, `sort by $this.effective` is the
// accessor the sort clause does provide, and `sort by O.effective` is an
// extension this engine accepts. The conformant one was the one going through a
// reader of its own, and it disagreed with the other two.
//
// This table is the reason there is now one reader. It asserts nothing about what
// a name *means* — only that the three spellings cannot answer differently, which
// is the shape of defect five consecutive review rounds kept turning up and the
// shape no amount of case-by-case fixing closed.
var nameResolutionSpellings = []struct {
	name string
	// expr renders the spelling of `element` inside a sort clause where the
	// query alias is X.
	expr func(element string) string
}{
	{"bare", func(el string) string { return el }},
	{"alias", func(el string) string { return "X." + el }},
	{"$this", func(el string) string { return "$this." + el }},
}

type nameTableProvider struct{ rows []string }

func (p nameTableProvider) Retrieve(_ context.Context, r eval.RetrieveRequest) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, 0, len(p.rows))
	for _, s := range p.rows {
		var envelope struct {
			ResourceType string `json:"resourceType"`
		}
		if err := json.Unmarshal([]byte(s), &envelope); err != nil {
			continue
		}
		if envelope.ResourceType == r.ResourceType {
			out = append(out, json.RawMessage(s))
		}
	}
	return out, nil
}

// evalNameTable answers with the value printed, or with "ERROR: …". An error is
// never a value here: every caller that compares two of these has to reject a
// pair that failed rather than call them equal. A comparison tool that compares
// two errors and reports agreement is a defect this repository has shipped once
// already.
func evalNameTable(t *testing.T, p nameTableProvider, expr string) string {
	t.Helper()
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\n" +
		"include FHIRHelpers version '4.0.1' called FH\ncontext Patient\ndefine A: " + expr + "\n"
	got, err := NewEngine(WithDataProvider(p), WithEvaluationTimestamp(
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

// The rows each case sorts, and the element the sort key names. `readback` turns
// the query into something printable: for a resource it is the id, for a bare
// value it is the value itself.
type nameResolutionCase struct {
	what     string
	rows     []string
	source   string // the query source: a retrieve, or a list literal
	element  string
	readback string // appended to First(…)/Last(…), e.g. ".id"
}

func (c nameResolutionCase) queries(spelling func(string) string) []string {
	key := spelling(c.element)
	return []string{
		fmt.Sprintf("First(%s X sort by %s)%s", c.source, key, c.readback),
		fmt.Sprintf("Last(%s X sort by %s)%s", c.source, key, c.readback),
		fmt.Sprintf("First(%s X sort by %s desc)%s", c.source, key, c.readback),
		fmt.Sprintf("Count(%s X sort by %s)", c.source, key),
	}
}

// TestOneNameOneAnswer is the table.
func TestOneNameOneAnswer(t *testing.T) {
	const observations = "[Observation]"
	const conditions = "[Condition]"

	cases := []nameResolutionCase{{
		what:   "a FHIR choice element, which is stored under its type",
		source: observations,
		rows: []string{
			`{"resourceType":"Observation","id":"june","status":"final","effectiveDateTime":"2019-06-01T10:00:00Z"}`,
			`{"resourceType":"Observation","id":"march","status":"final","effectiveDateTime":"2019-03-01T10:00:00Z"}`,
		},
		element: "effective", readback: ".id",
	}, {
		what:   "the concrete spelling of that same choice element",
		source: observations,
		rows: []string{
			`{"resourceType":"Observation","id":"june","status":"final","effectiveDateTime":"2019-06-01T10:00:00Z"}`,
			`{"resourceType":"Observation","id":"march","status":"final","effectiveDateTime":"2019-03-01T10:00:00Z"}`,
		},
		element: "effectiveDateTime", readback: ".id",
	}, {
		what:   "a plain element the type declares itself",
		source: observations,
		rows: []string{
			`{"resourceType":"Observation","id":"b","status":"final"}`,
			`{"resourceType":"Observation","id":"a","status":"registered"}`,
		},
		element: "status", readback: ".id",
	}, {
		// Condition.recordedDate is a FHIR.dateTime, and a value written to the
		// day arrives from JSON as a Date. Only one of the three spellings used to
		// give it the type the model declares.
		what:   "a dateTime element written to the day, alongside one with an instant",
		source: conditions,
		rows: []string{
			`{"resourceType":"Condition","id":"day","recordedDate":"2019-03-01"}`,
			`{"resourceType":"Condition","id":"instant","recordedDate":"2019-02-01T10:00:00Z"}`,
		},
		element: "recordedDate", readback: ".id",
	}, {
		what:   "an element the type inherits — Observation.id lives on Resource",
		source: observations,
		rows: []string{
			`{"resourceType":"Observation","id":"b","status":"final"}`,
			`{"resourceType":"Observation","id":"a","status":"final"}`,
		},
		element: "id", readback: ".id",
	}, {
		what:   "an inherited element none of the rows carries",
		source: observations,
		rows: []string{
			`{"resourceType":"Observation","id":"b","status":"final"}`,
			`{"resourceType":"Observation","id":"a","status":"final"}`,
		},
		element: "language", readback: ".id",
	}, {
		what:   "a declared element some rows carry and others do not",
		source: observations,
		rows: []string{
			`{"resourceType":"Observation","id":"undated","status":"final"}`,
			`{"resourceType":"Observation","id":"june","status":"final","effectiveDateTime":"2019-06-01T10:00:00Z"}`,
			`{"resourceType":"Observation","id":"march","status":"final","effectiveDateTime":"2019-03-01T10:00:00Z"}`,
		},
		element: "effective", readback: ".id",
	}, {
		what:    "a tuple column",
		source:  "({Tuple{k: 2, mark: 'b'}, Tuple{k: 1, mark: 'a'}})",
		element: "k", readback: ".mark",
	}, {
		// Not FHIR at all: a member access knows these and the sort key's own
		// reader did not, so the conformant spelling was refused outright.
		what:    "value on a System.Quantity",
		source:  "({2 'mg', 1 'mg'})",
		element: "value", readback: "",
	}, {
		what:    "unit on a System.Quantity",
		source:  "({2 'mg', 1 'mg'})",
		element: "unit", readback: "",
	}, {
		what:    "code on a System.Code",
		source:  "({Code {code: 'b', system: 's'}, Code {code: 'a', system: 's'}})",
		element: "code", readback: ".code",
	}, {
		what:    "system on a System.Code",
		source:  "({Code {code: 'b', system: 't'}, Code {code: 'a', system: 's'}})",
		element: "system", readback: ".code",
	}, {
		what:    "display on a System.Concept",
		source:  "({Concept {codes: {Code {code: 'b'}}, display: 'z'}, Concept {codes: {Code {code: 'a'}}, display: 'y'}})",
		element: "display", readback: ".display",
	}}

	for _, c := range cases {
		t.Run(c.what, func(t *testing.T) {
			p := nameTableProvider{rows: c.rows}
			// The first spelling is the conformant one and sets the answer the
			// others are held to.
			base := nameResolutionSpellings[0]
			want := make([]string, 0, 4)
			for _, q := range c.queries(base.expr) {
				got := evalNameTable(t, p, q)
				// An expression that does not evaluate is not an answer. Left as a
				// comparable string it would let two spellings "agree on being
				// broken", which is how a table in this repository once ran 432 of
				// its combinations as syntax errors and reported success.
				if strings.HasPrefix(got, "ERROR") {
					t.Fatalf("%s\n  %s", got, q)
				}
				want = append(want, got)
			}

			for _, sp := range nameResolutionSpellings[1:] {
				for i, q := range c.queries(sp.expr) {
					got := evalNameTable(t, p, q)
					if strings.HasPrefix(got, "ERROR") {
						t.Errorf("spelling %q does not evaluate where %q does:\n  %s\n  %s",
							sp.name, base.name, q, got)
						continue
					}
					if got != want[i] {
						t.Errorf("one name, two answers:\n  %-6s %s = %s\n  %-6s %s = %s",
							base.name, c.queries(base.expr)[i], want[i],
							sp.name, q, got)
					}
				}
			}
		})
	}
}

// TestOneNameOneType covers what an ordering cannot show: two spellings can sort
// a set identically while disagreeing about the type of the key they sorted it
// by. That was the second of the three divergences, and it is the one that
// decides `is`, `as` and which branch an if takes:
//
//	recordedDate is Date      was true   — the type the JSON text suggested
//	C.recordedDate is Date    was false  — FHIR.dateTime, which is what is declared
//
// Reading it takes care, because a sort of two rows carries one bit and a tie
// looks like an answer. The bit is read against a reference row served *first*
// whose predicate is reliably false — it has no recordedDate at all, and null is
// of no type. A tie therefore leaves the reference in front, so the subject
// coming first can only mean the predicate held for it.
//
// The first draft of this test got that wrong: it used a reference row with a
// full instant, for which `is DateTime` is true as well, so both probes tied and
// both reported the answer of whichever row the provider happened to serve first.
// It "passed" the broken predicate and failed the correct one.
func TestOneNameOneType(t *testing.T) {
	p := nameTableProvider{rows: []string{
		// The reference, served first: no recordedDate, so every `is` is false.
		`{"resourceType":"Condition","id":"reference"}`,
		// recordedDate is declared FHIR.dateTime. Written to the day it arrives
		// from JSON as a Date, and the model is what says otherwise.
		`{"resourceType":"Condition","id":"subject","recordedDate":"2019-03-01"}`,
	}}

	for _, probe := range []struct {
		predicate string
		holds     bool
		why       string
	}{
		{"is DateTime", true, "a FHIR.dateTime is a DateTime, at whatever precision it was written"},
		{"is Date", false, "the JSON text carried no time; the model is what says which type it is"},
		{"is String", false, "it is not the raw JSON string either"},
	} {
		want := "reference"
		if probe.holds {
			want = "subject"
		}
		for _, sp := range nameResolutionSpellings {
			key := fmt.Sprintf("(if %s %s then 'a' else 'b')", sp.expr("recordedDate"), probe.predicate)
			q := fmt.Sprintf("First([Condition] X sort by %s).id", key)
			got := evalNameTable(t, p, q)
			if strings.HasPrefix(got, "ERROR") {
				t.Errorf("spelling %q: %s\n  %s", sp.name, got, q)
				continue
			}
			if got != want {
				t.Errorf("spelling %q says `recordedDate %s` is %v for the day-precision row, want %v — %s\n  %s",
					sp.name, probe.predicate, got == "subject", probe.holds, probe.why, q)
			}
		}
	}
}

// TestANameNothingDeclaresIsStillAMistake is the other half of the table: making
// the three spellings agree must not be done by making every name resolve.
//
// All three report it, and it is worth recording that the report comes from the
// semantic phase, not from this resolution: sema is a fourth reader of "a name on
// a value", it types the rows without evaluating them, and it is what refuses an
// invented name before there is a row to look at.
func TestANameNothingDeclaresIsStillAMistake(t *testing.T) {
	p := nameTableProvider{rows: []string{
		`{"resourceType":"Observation","id":"a","status":"final"}`,
		`{"resourceType":"Observation","id":"b","status":"final"}`,
	}}
	for _, sp := range nameResolutionSpellings {
		q := fmt.Sprintf("Count([Observation] X sort by %s)", sp.expr("noSuchElement"))
		if got := evalNameTable(t, p, q); !strings.HasPrefix(got, "ERROR") {
			t.Errorf("spelling %q: a sort key naming nothing = %s, want an error\n  %s", sp.name, got, q)
		}
	}
	if got := evalNameTable(t, p, "Count([Observation] X where X.noSuchElement is null)"); !strings.HasPrefix(got, "ERROR") {
		t.Errorf("a member access on a name nothing declares = %s, want the same refusal", got)
	}
}

// TestValueOnASystemPrimitiveIsNotAColumn records a divergence this change did
// *not* remove, asserted rather than described so that closing it trips here.
//
// `.value` on a system primitive is the primitive itself, which is what lets a
// library written against the official FHIRHelpers read `coding.code.value`: the
// official ModelInfo models FHIR.string as an object with a value element, while
// this engine navigates raw JSON where the scalar is already there. The rule is
// deliberately narrow — if `.value` on any value returned that value, a mistyped
// someString.value would answer the string instead of failing.
//
// The semantic phase does not know the rule, so it refuses both spellings on a
// System.String. They refuse differently, which is the seam:
//
//	sort by value      value is not defined
//	sort by X.value    System.String has no element value
//
// Both are refusals and neither is a wrong answer, so this is a wart rather than
// a defect, and widening sema to admit the bridge would widen exactly the typo it
// was made narrow to catch. If it is ever decided, decide it in one place and
// delete this.
func TestValueOnASystemPrimitiveIsNotAColumn(t *testing.T) {
	p := nameTableProvider{}
	for _, sp := range nameResolutionSpellings {
		q := fmt.Sprintf("First(({'b', 'a'}) X sort by %s)", sp.expr("value"))
		if got := evalNameTable(t, p, q); !strings.HasPrefix(got, "ERROR") {
			t.Errorf("`.value` on a System.String answers %s through the %q spelling now — "+
				"decide the rule for all three and remove this test\n  %s", got, sp.name, q)
		}
	}
}
