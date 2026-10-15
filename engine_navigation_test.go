package cql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofhir/cql/eval"
)

// TestANameAppliedToAListIsReadOffEachEntry covers path navigation through a
// repeating FHIR element.
//
// One entry arrives as that entry and two as a list of them, and a list had no
// elements: with two names, Patient.name.family was null and
// Count(Patient.name.period) was 0. The reference translator writes a name
// applied to a list as a query over it — entries without the element dropped,
// duplicates kept, flattened when the element repeats too — so that is what
// these answers are, for none, one and two names.
func TestANameAppliedToAListIsReadOffEachEntry(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1' called FH\n" +
		"context Patient\n" +
		"define Families: Patient.name.family\n" +
		"define PeriodCount: Count(Patient.name.period)\n" +
		"define Givens: Patient.name.given\n" +
		"define HasB: 'B' in Patient.name.family\n" +
		"define Starts: Count(Patient.name.period.start)\n"
	e := NewEngine()
	for _, tt := range []struct {
		name, patient string
		want          map[string]string
	}{
		// An absent element is null here, and one entry is that entry, not a
		// list of it. The reference answers {} and {A}: both are how this engine
		// represents FHIR repetition, which this change does not touch, and Count
		// and `in` answer alike either way.
		{"no names", `{"resourceType":"Patient","id":"p"}`,
			map[string]string{"Families": "null", "PeriodCount": "0", "Givens": "null", "HasB": "false", "Starts": "0"}},
		{"one name", `{"resourceType":"Patient","id":"p","name":[{"family":"A","given":["x","y"],"period":{"start":"2020-01-01"}}]}`,
			map[string]string{"Families": "A", "PeriodCount": "1", "Givens": "{x, y}", "HasB": "false", "Starts": "1"}},
		{"two names", `{"resourceType":"Patient","id":"p","name":[` +
			`{"family":"A","given":["x","y"],"period":{"start":"2020-01-01"}},` +
			`{"family":"B","given":["z"],"period":{"start":"2021-01-01"}}]}`,
			map[string]string{"Families": "{A, B}", "PeriodCount": "2", "Givens": "{x, y, z}", "HasB": "true", "Starts": "2"}},
		// The same family twice is two entries, as the reference's `return all`
		// keeps them; a name without a family is not one.
		{"repeated and missing", `{"resourceType":"Patient","id":"p","name":[{"family":"A"},{"given":["z"]},{"family":"A"}]}`,
			map[string]string{"Families": "{A, A}", "PeriodCount": "0", "Givens": "{z}", "HasB": "false", "Starts": "0"}},
		// No entry has the element: an empty list, not null.
		{"none has it", `{"resourceType":"Patient","id":"p","name":[{"given":["x"]},{"given":["z"]}]}`,
			map[string]string{"Families": "{}", "PeriodCount": "0", "Givens": "{x, z}", "HasB": "false", "Starts": "0"}},
	} {
		for def, want := range tt.want {
			v, err := e.EvaluateExpression(context.Background(), src, def, []byte(tt.patient), nil)
			got := "null"
			if v != nil {
				got = valueString(v)
			}
			if err != nil || got != want {
				t.Errorf("%s: %s = %s, %v; want %s", tt.name, def, got, err, want)
			}
		}
	}
}

// TestANameAppliedToACQLListIsReadOffEachEntry covers the same rule on a list
// written in CQL, which the translator treats no differently: a list stays a
// list whatever it holds, one entry included.
func TestANameAppliedToACQLListIsReadOffEachEntry(t *testing.T) {
	e := NewEngine()
	for expr, want := range map[string]string{
		"({Tuple{a: 1}, Tuple{a: null}, Tuple{a: 1}}).a": "{1, 1}",
		"({Tuple{a: {1, 2}}, Tuple{a: {3}}}).a":          "{1, 2, 3}",
		"({Tuple{a: 1}}).a":                              "{1}",
		"(List<Tuple{a Integer}>{}).a":                   "{}",
		// Flatten keeps a null inside an entry's list; only an entry whose
		// element is null is dropped.
		"({Tuple{a: {1, null}}, Tuple{a: {3}}}).a": "{1, null, 3}",
		// Not CQL — the translator refuses a list of lists here — but the
		// semantic phase types it List<Integer>, and the value is what it says.
		"({{Tuple{a: 1}}, Tuple{a: 2}}).a": "{1, 2}",
	} {
		src := "library T version '1.0'\ndefine R: " + expr + "\n"
		v, err := e.EvaluateExpression(context.Background(), src, "R", nil, nil)
		got := "null"
		if v != nil {
			got = valueString(v)
		}
		if err != nil || got != want {
			t.Errorf("%s = %s, %v; want %s", expr, got, err, want)
		}
	}
}

// TestASortKeyIsNotAColumnOfAListRow covers a sort key that names a
// definition, over rows that are lists. A list row has no columns — the
// translator rejects `sort by a` over one — so the key is the definition for
// every row and the order is the rows' own. When elementOf read names off list
// entries, a row's {} or {2} became its key instead.
func TestASortKeyIsNotAColumnOfAListRow(t *testing.T) {
	for rows, want := range map[string]string{
		"{{2}, {}, {1}}":                 "{{2}, {}, {1}}",
		"{{Tuple{D: 2}}, {Tuple{D: 1}}}": "{{Tuple{D: 2}}, {Tuple{D: 1}}}",
	} {
		src := "library T version '1.0'\ndefine D: 5\ndefine R: (" + rows + ") L sort by D\n"
		v, err := NewEngine().EvaluateExpression(context.Background(), src, "R", nil, nil)
		if err != nil || valueString(v) != want {
			t.Errorf("sort by a definition over %s = %v, %v; want %s", rows, v, err, want)
		}
	}
}

type reactionProvider struct{ reactions string }

func (p reactionProvider) Retrieve(_ context.Context, r eval.RetrieveRequest) ([]json.RawMessage, error) {
	if r.ResourceType != "Immunization" {
		return nil, nil
	}
	return []json.RawMessage{json.RawMessage(`{"resourceType":"Immunization","id":"i","status":"completed",` +
		`"vaccineCode":{"text":"x"},"occurrenceDateTime":"2020-01-01","reaction":[` + p.reactions + `]}`)}, nil
}

// TestADateTimeReadThroughARepeatingElementIsOne covers a FHIR dateTime written
// as a date and read through a repeating backbone element, whose owner type the
// value cannot name. The semantic phase types Immunization.reaction.date as a
// list of dateTime, and the check for a dateTime did not look inside a list, so
// the value stayed a Date — with one reaction as with two.
func TestADateTimeReadThroughARepeatingElementIsOne(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1' called FH\n" +
		"context Patient\n" +
		"define R: First([Immunization] I return First(I.reaction.date) is DateTime)\n"
	for _, reactions := range []string{`{"date":"2020-01-01"}`, `{"date":"2020-01-01"},{"date":"2021-01-01"}`} {
		e := NewEngine(WithDataProvider(reactionProvider{reactions}))
		v, err := e.EvaluateExpression(context.Background(), src, "R", []byte(`{"resourceType":"Patient","id":"p"}`), nil)
		if err != nil || valueString(v) != "true" {
			t.Errorf("reaction.date is DateTime with reactions %s = %v, %v; want true", reactions, v, err)
		}
	}
}
