package cql

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofhir/cql/eval"
)

// measureProvider serves the shapes a measure walks: conditions belonging to
// two different patients, and an encounter with a period.
type measureProvider struct{ encounterBody string }

func (p *measureProvider) Retrieve(_ context.Context, req eval.RetrieveRequest) ([]json.RawMessage, error) {
	switch req.ResourceType {
	case "Condition":
		rows := []json.RawMessage{
			json.RawMessage(`{"resourceType":"Condition","id":"c1","subject":{"reference":"Patient/p1"},` +
				`"code":{"coding":[{"system":"http://snomed.info/sct","code":"44054006","display":"T2DM"}]}}`),
			json.RawMessage(`{"resourceType":"Condition","id":"c2","subject":{"reference":"Patient/other"},` +
				`"code":{"coding":[{"system":"http://snomed.info/sct","code":"44054006"}]}}`),
		}
		// A provider that honors the context returns only the subject's own.
		if req.ContextID == "p1" && req.ContextSearchParam != "" {
			return rows[:1], nil
		}
		return rows, nil
	case "Encounter":
		body := p.encounterBody
		if body == "" {
			body = `{"resourceType":"Encounter","id":"e1","period":{"start":"2020-03-01","end":"2020-03-05"}}`
		}
		return []json.RawMessage{json.RawMessage(body)}, nil
	}
	return nil, nil
}

type snomedTerminology struct{}

func (snomedTerminology) InValueSet(_ context.Context, code, _, valueSet string) (bool, error) {
	return code == "44054006" && strings.Contains(valueSet, "t2dm"), nil
}

const measureLibrary = `library Measure version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
valueset "T2DM": 'http://example.org/vs/t2dm'
parameter MP Interval<DateTime> default Interval[@2020-01-01, @2020-12-31]

context Patient

define CodeInValueSet:   First([Condition]).code in "T2DM"
define FilteredByCode:   [Condition] C where C.code in "T2DM" return C.id
define PatientScoped:    Count([Condition])
define PeriodDuringMP:   First([Encounter]).period during MP
define MeasurePeriod:    MP
`

// TestMeasureShapedExpressions covers the five expressions the plan tracks as
// its acceptance criterion for the FHIR data path. Each failed for its own
// reason: a Code that was in no value set, a query that filtered on it, a
// retrieve that returned other patients' rows, a FHIR Period where an Interval
// was wanted, and a parameter default that was never evaluated.
func TestMeasureShapedExpressions(t *testing.T) {
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)
	engine := NewEngine(
		WithDataProvider(&measureProvider{}),
		WithTerminologyProvider(snomedTerminology{}),
	)
	for _, tt := range []struct{ name, want string }{
		{"CodeInValueSet", "true"},
		{"FilteredByCode", "{c1}"},
		{"PatientScoped", "1"},
		{"PeriodDuringMP", "true"},
		{"MeasurePeriod", "Interval[2020-01-01, 2020-12-31]"},
	} {
		got, err := engine.EvaluateExpression(context.Background(), measureLibrary, tt.name, patient, nil)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if s := valueString(got); s != tt.want {
			t.Errorf("%s = %s, want %s", tt.name, s, tt.want)
		}
	}
}

// TestCoercionAtOperators covers a FHIR type reaching an operator that works on
// CQL system types. The model declares 264 conversions — FHIR.Period to
// Interval<System.DateTime> through FHIRHelpers.ToInterval — and the reference
// implementation inserts the calls in its translator from static types. With no
// semantic phase they are applied here, when the operator is handed a type it
// cannot work with.
func TestCoercionAtOperators(t *testing.T) {
	src := `library M version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
parameter MP Interval<DateTime> default Interval[@2020-01-01, @2020-12-31]

define Enc: First([Encounter])
define During: Enc.period during MP
define StartsBefore: Enc.period starts before end of MP
define Overlaps: Enc.period overlaps MP
define IncludedIn: Enc.period included in MP
define StartOf: start of Enc.period
define EndOf: end of Enc.period
define Duration: duration in days of Enc.period
define Compared: end of Enc.period > start of MP
`
	engine := NewEngine(WithDataProvider(&measureProvider{}))
	for _, tt := range []struct{ name, want string }{
		{"During", "true"},
		{"StartsBefore", "true"},
		{"Overlaps", "true"},
		{"IncludedIn", "true"},
		{"StartOf", "2020-03-01"},
		{"EndOf", "2020-03-05"},
		// Both endpoints are written to the day, and a FHIR dateTime without a
		// time means the time is unknown rather than midnight. Two indeterminate
		// instants a few days apart are 3 or 4 days apart, which is what CQL
		// models as an uncertainty and what the conformance corpus confirms:
		// `years between DateTime(2005) and DateTime(2010)` is Interval[4, 5].
		//
		// It answered 4 while the endpoints came back as Dates — a Date *is* the
		// day, so counting days off it is exact. Reading them as the dateTimes
		// the model declares is what moved this onto the honest answer.
		{"Duration", "Interval[3, 4]"},
		{"Compared", "true"},
	} {
		got, err := engine.EvaluateExpression(context.Background(), src, tt.name, nil, nil)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if s := valueString(got); s != tt.want {
			t.Errorf("%s = %s, want %s", tt.name, s, tt.want)
		}
	}
}

// TestCoercionStaysInItsLane covers what conversion must not do. Navigating to
// a FHIR value still yields the FHIR value — only an operator that cannot work
// with one converts — and a library that never included FHIRHelpers does not
// get it pulled in behind the author's back.
func TestCoercionStaysInItsLane(t *testing.T) {
	withHelpers := `library M version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH

define Enc: First([Encounter])
define Raw: Enc.period
define RawStart: Enc.period.start
define StillAPeriod: Enc.period is Period
`
	engine := NewEngine(WithDataProvider(&measureProvider{}))
	raw, err := engine.EvaluateExpression(context.Background(), withHelpers, "Raw", nil, nil)
	if err != nil {
		t.Fatalf("Raw: %v", err)
	}
	if raw == nil || raw.Type() != "Period" {
		t.Errorf("Raw is %v, want a FHIR Period — navigation must not convert", raw)
	}
	for _, tt := range []struct{ name, want string }{
		{"RawStart", "2020-03-01"},
		{"StillAPeriod", "true"},
	} {
		got, evalErr := engine.EvaluateExpression(context.Background(), withHelpers, tt.name, nil, nil)
		if evalErr != nil {
			t.Fatalf("%s: %v", tt.name, evalErr)
		}
		if s := valueString(got); s != tt.want {
			t.Errorf("%s = %s, want %s", tt.name, s, tt.want)
		}
	}

	// Without the include there is no conversion function to call, and the
	// engine does not reach for one the author never named.
	without := `library M version '1.0'
using FHIR version '4.0.1'
parameter MP Interval<DateTime> default Interval[@2020-01-01, @2020-12-31]
define During: First([Encounter]).period during MP
`
	got, err := NewEngine(WithDataProvider(&measureProvider{})).
		EvaluateExpression(context.Background(), without, "During", nil, nil)
	if err != nil {
		t.Fatalf("without FHIRHelpers: %v", err)
	}
	if got != nil {
		t.Errorf("without FHIRHelpers = %v, want null", got)
	}
}

// TestCoercionOnAwkwardData covers period shapes that occur in real resources
// rather than in a fixture chosen to work. An Encounter carrying an empty
// `period: {}` has nothing to infer a type from, so no conversion applies; it
// used to fail the whole expression with "cannot compare Object".
func TestCoercionOnAwkwardData(t *testing.T) {
	src := `library M version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
parameter MP Interval<DateTime> default Interval[@2020-01-01, @2020-12-31]

define During: First([Encounter]).period during MP
define StartOf: start of First([Encounter]).period
`
	for _, tt := range []struct{ name, body, during, start string }{
		{"complete", `{"resourceType":"Encounter","id":"e","period":{"start":"2020-03-01","end":"2020-03-05"}}`, "true", "2020-03-01"},
		{"no end", `{"resourceType":"Encounter","id":"e","period":{"start":"2020-03-01"}}`, "false", "2020-03-01"},
		{"no start", `{"resourceType":"Encounter","id":"e","period":{"end":"2020-03-05"}}`, "false", "null"},
		{"empty period", `{"resourceType":"Encounter","id":"e","period":{}}`, "null", "null"},
		{"no period", `{"resourceType":"Encounter","id":"e"}`, "null", "null"},
		{"with time", `{"resourceType":"Encounter","id":"e","period":{"start":"2020-03-01T10:00:00Z","end":"2020-03-05T18:30:00Z"}}`, "true", "2020-03-01T10:00:00Z"},
		{"outside MP", `{"resourceType":"Encounter","id":"e","period":{"start":"2019-01-01","end":"2019-02-01"}}`, "false", "2019-01-01"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine(WithDataProvider(&measureProvider{encounterBody: tt.body}))
			during, err := engine.EvaluateExpression(context.Background(), src, "During", nil, nil)
			if err != nil {
				t.Fatalf("During: %v", err)
			}
			if s := valueString(during); s != tt.during {
				t.Errorf("During = %s, want %s", s, tt.during)
			}
			start, err := engine.EvaluateExpression(context.Background(), src, "StartOf", nil, nil)
			if err != nil {
				t.Fatalf("StartOf: %v", err)
			}
			if s := valueString(start); s != tt.start {
				t.Errorf("StartOf = %s, want %s", s, tt.start)
			}
		})
	}
}

// TestMembershipWithFHIRResources covers list membership, which asks whether a
// collection contains a value rather than how that value converts. A FHIR
// resource type has no declared conversion, so guarding the timing operators
// against unconverted values had turned every such question into null.
func TestMembershipWithFHIRResources(t *testing.T) {
	src := `library M version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH

define Encs: [Encounter]
define One: First([Encounter])
define IncludedIn: One included in Encs
define Includes: Encs includes One
define ProperlyIncluded: One properly included in Encs
`
	engine := NewEngine(WithDataProvider(&measureProvider{}))
	for _, name := range []string{"IncludedIn", "Includes"} {
		got, err := engine.EvaluateExpression(context.Background(), src, name, nil, nil)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if s := valueString(got); s != "true" {
			t.Errorf("%s = %s, want true", name, s)
		}
	}
}

// TestMembershipCoercesLikeTiming covers `in` and `contains`, which are the more
// common spellings of the same question `during` asks. Conversion was wired into
// the timing operators and not these, so `encounter.period in MP` failed where
// `encounter.period during MP` worked.
//
// It also covers an interval on both sides: CQL reads `X in Y` as IncludedIn
// when X is an interval, a distinction the translator makes from static types
// and this decides from the values.
func TestMembershipCoercesLikeTiming(t *testing.T) {
	src := `library M version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
parameter MP Interval<DateTime> default Interval[@2020-01-01, @2020-12-31]

define PeriodIn: First([Encounter]).period in MP
define MPContains: MP contains First([Encounter]).period
define IntervalIn: Interval[@2020-03-01, @2020-03-05] in MP
define PointIn: @2020-03-01 in MP
define OutsideIn: Interval[@2019-01-01, @2019-02-01] in MP
`
	engine := NewEngine(WithDataProvider(&measureProvider{}))
	for _, tt := range []struct{ name, want string }{
		{"PeriodIn", "true"},
		{"MPContains", "true"},
		{"IntervalIn", "true"},
		{"PointIn", "true"},
		{"OutsideIn", "false"},
	} {
		got, err := engine.EvaluateExpression(context.Background(), src, tt.name, nil, nil)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if s := valueString(got); s != tt.want {
			t.Errorf("%s = %s, want %s", tt.name, s, tt.want)
		}
	}
}

// TestConversionErrorsPropagate covers a conversion that fails for a real
// reason. Reporting it as "no conversion applied" would turn a canceled
// evaluation, a depth limit or a broken conversion library into an ordinary
// answer.
func TestConversionErrorsPropagate(t *testing.T) {
	const broken = `library FHIRHelpers version '4.0.1'
using FHIR version '4.0.1'

define function ToInterval(period FHIR.Period): NoSuchIdentifier
`
	src := `library M version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FH
parameter MP Interval<DateTime> default Interval[@2020-01-01, @2020-12-31]
define X: First([Encounter]).period during MP
`
	engine := NewEngine(WithDataProvider(&measureProvider{}),
		WithLibraryResolver(func(_ context.Context, name, _ string) (string, error) {
			if name == "FHIRHelpers" {
				return broken, nil
			}
			return "", context.Canceled
		}))
	_, err := engine.EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err == nil {
		t.Fatal("a conversion that fails should surface, not answer null")
	}
	if !strings.Contains(err.Error(), "NoSuchIdentifier") {
		t.Errorf("the error should carry its cause, got: %v", err)
	}
}

// TestConversionNeedsTheAuthorsInclude covers reaching through the include
// graph. A library that included something which in turn includes FHIRHelpers
// has not asked for conversions itself, and converting on the strength of
// somebody else's dependency would make the behavior depend on what a
// transitive library happened to import.
func TestConversionNeedsTheAuthorsInclude(t *testing.T) {
	libs := map[string]string{
		"Middle": "library Middle version '1.0'\nusing FHIR version '4.0.1'\n" +
			"include FHIRHelpers version '4.0.1' called FH\n\ndefine Anchor: 1\n",
	}
	src := `library M version '1.0'
using FHIR version '4.0.1'
include Middle version '1.0' called Mid
parameter MP Interval<DateTime> default Interval[@2020-01-01, @2020-12-31]
define X: First([Encounter]).period during MP
`
	engine := NewEngine(WithDataProvider(&measureProvider{}),
		WithLibraryResolver(func(_ context.Context, name, _ string) (string, error) {
			if s, ok := libs[name]; ok {
				return s, nil
			}
			return "", context.Canceled
		}))
	got, err := engine.EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err != nil {
		t.Fatalf("evaluating: %v", err)
	}
	if got != nil {
		t.Errorf("X = %v, want null — the author never included FHIRHelpers", got)
	}
}
