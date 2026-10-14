package cql

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

// runtimeReference is what the reference engine answered for each definition of
// testdata/reference/runtime.cql, recorded by internal/referenceharness
// (Evaluate.java). It is committed for the same reason referenceELM is: the
// engine needs a JVM, and the point is to diff against it, not to re-derive it.
type runtimeReference struct {
	Library            string                      `json:"library"`
	SourceSha256       string                      `json:"sourceSha256"`
	Engine             string                      `json:"engine"`
	EvaluationDateTime string                      `json:"evaluationDateTime"`
	Results            map[string]*runtimeRefValue `json:"results"`
}

// runtimeRefValue is one value as Evaluate.java encodes it. A JSON null decodes
// to a nil pointer, which is how a null result is told apart from an empty list.
type runtimeRefValue struct {
	Type      string          `json:"type"`
	Value     json.RawMessage `json:"value,omitempty"`
	Unit      string          `json:"unit,omitempty"`
	Precision string          `json:"precision,omitempty"`
	Instant   string          `json:"instant,omitempty"`
	Elements  json.RawMessage `json:"elements,omitempty"`
	Error     string          `json:"error,omitempty"`
	Message   string          `json:"message,omitempty"`
}

// runtimeReferenceEngine is the version the harness pins in its pom.xml. A
// fixture regenerated with another one is a different oracle and has to be
// reviewed as such, not slipped in under the same test.
const runtimeReferenceEngine = "3.26.0"

// knownRuntimeDivergences are the definitions where this engine is known to
// answer differently from the reference, each with this engine's exact answer
// as describeRuntimeValue writes it.
//
// Pinning the answer, not just the name, is what keeps the list honest: a
// divergence that changes into a different wrong answer fails here, and one
// that gets fixed fails too, until it is removed from this list and from
// runtime.divergences.md.
var knownRuntimeDivergences = map[string]struct {
	got string
	why string
}{
	// The reference and the specification agree; this engine does not.
	"WithAggregate":           {"Integer(6)", "the aggregate loop iterates the unfiltered product and reapplies only where"},
	"WithoutAggregate":        {"Integer(6)", "same cause as WithAggregate"},
	"ReturnDefault":           {"List[Integer(1), Integer(1), Integer(2)]", "return without all is not distinct"},
	"UnionDuplicates":         {"List[Integer(1), Integer(1), Integer(2)]", "union keeps duplicates from one operand"},
	"ExceptDuplicates":        {"List[Integer(1), Integer(1), Integer(2)]", "except keeps duplicates from the first operand"},
	"AggDistinctNilNull":      {"Integer(1)", "distinctCombos keys rows by String(): null and 'nil' collide"},
	"AggDistinctPairs":        {"Integer(3)", "distinctCombos joins keys with a comma that the values can contain"},
	"AggDistinctIntDec":       {"Integer(2)", "distinctCombos compares text, and 1 = 1.0"},
	"AggDistinctOffsets":      {"Integer(2)", "distinctCombos compares text, and the two are the same instant"},
	"ListIntDec":              {"List[Integer(1), Decimal(1)]", "a list of Integer and Decimal is not converted to List<Decimal>"},
	"ExpandWeekPerMinute":     {"Integer(10001)", "expand stops at 10001 elements without an error"},
	"ExpandThirtyYearsPerDay": {"Integer(10001)", "same guard as ExpandWeekPerMinute"},
	"ExpandIntegers":          {"Integer(10001)", "same guard as ExpandWeekPerMinute"},
	"SameDayAcrossOffsets": {"Boolean(false)", "values at different offsets are normalized to UTC; the " +
		"specification normalizes to the evaluation offset, and only below day precision"},
}

// pinnedRuntimeAnswers are the definitions the reference cannot settle, each
// with this engine's exact answer and the reason the reference is not consulted.
//
// They are kept apart from knownRuntimeDivergences on purpose. There, agreeing
// with the reference means fixed; here it can mean the opposite — this engine
// breaking unit conversion would agree with a reference that never had it. So
// only this engine's answer is checked, and any change to it fails until
// someone checks it against the specification and moves the pin.
var pinnedRuntimeAnswers = map[string]struct {
	got string
	why string
}{
	"QuantityEqual":      {"Boolean(true)", "the reference does not convert units: 1 'm' = 100 'cm' is null there"},
	"QuantityEquivalent": {"Boolean(true)", "the reference does not convert units: it answers false"},
	"DistinctQuantity":   {"List[Quantity(1 'm')]", "the reference does not convert units and keeps both"},
	"AggDistinctQuantity": {"Integer(2)", "the reference answers 2 too, but distinct is defined by equality and " +
		"1 'm' = 100 'cm', so the specification says 1: both are probably wrong, for different reasons"},
	"DateTimeYear": {"DateTime(2020-01-01T00:00:00/year, no offset)", "fhirpath's WithDefaultOffset gives no offset " +
		"below hour precision, deliberately; the reference assumes the evaluation offset, as the specification words it"},
}

// unorderedRuntimeResults are the definitions whose list result is compared as
// a multiset, each with the reason its order is not part of the answer. Every
// other list is compared in order: a sort that returned its elements reversed
// would pass an unordered comparison, and that is a defect worth finding.
var unorderedRuntimeResults = map[string]string{
	"ReturnDefault":       "distinct fixes which elements remain, not their order",
	"ReturnDistinct":      "same as ReturnDefault",
	"UnionDuplicates":     "Union: no expectation that the order of elements in the inputs will be preserved",
	"ExceptDuplicates":    "Except is a set operation",
	"IntersectDuplicates": "Intersect is a set operation",
	"DistinctQuantity":    "same as ReturnDefault",
	"DistinctOffsets":     "same as ReturnDefault",
}

// TestRuntimeMatchesTheReference covers what this engine answers against what
// the reference engine answers, value by value.
//
// The comparison is written here rather than borrowed from the engine: decimals
// are compared with math/big, instants with time, and nothing goes through this
// engine's String or Equal. An oracle that used them would inherit the defects
// it exists to find — distinctCombos is wrong precisely because it trusted
// String.
func TestRuntimeMatchesTheReference(t *testing.T) {
	src, err := os.ReadFile("testdata/reference/runtime.cql")
	if err != nil {
		t.Fatalf("reading the runtime probe: %v", err)
	}
	data, err := os.ReadFile("testdata/reference/runtime.expected.json")
	if err != nil {
		t.Fatalf("reading the runtime reference: %v", err)
	}
	var ref runtimeReference
	if err = json.Unmarshal(data, &ref); err != nil {
		t.Fatalf("decoding the runtime reference: %v", err)
	}
	if ref.Engine != runtimeReferenceEngine {
		t.Fatalf("the reference was produced by engine %q, the harness pins %q", ref.Engine, runtimeReferenceEngine)
	}

	// The names alone cannot tell a stale reference: a definition edited without
	// regenerating keeps its name, and its old answer would read as a divergence
	// of this engine — one that could then be pinned below as understood.
	// Line endings are normalized first, so a checkout that converts them does
	// not read as an edit. Evaluate.java hashes the file as committed, with LF.
	src = bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))
	if sum := sha256.Sum256(src); hex.EncodeToString(sum[:]) != ref.SourceSha256 {
		t.Fatalf("runtime.cql changed since the reference was produced — regenerate runtime.expected.json")
	}
	if m := libraryName.FindStringSubmatch(string(src)); len(m) < 2 || m[1] != ref.Library {
		t.Fatalf("the reference answers for library %q, which is not the probe's", ref.Library)
	}

	names := runtimeManifest(t, string(src), ref)

	evaluatedAt, err := time.Parse(time.RFC3339, ref.EvaluationDateTime)
	if err != nil {
		t.Fatalf("reading the reference evaluation time %q: %v", ref.EvaluationDateTime, err)
	}
	engine := NewEngine(WithEvaluationTimestamp(evaluatedAt))
	lib, err := engine.Parse(string(src))
	if err != nil {
		t.Fatalf("parsing the runtime probe: %v", err)
	}

	for name := range knownRuntimeDivergences {
		if _, ok := ref.Results[name]; !ok {
			t.Errorf("%s is listed as a known divergence and is not in the probe", name)
		}
	}
	for name := range pinnedRuntimeAnswers {
		if _, ok := ref.Results[name]; !ok {
			t.Errorf("%s is pinned and is not in the probe", name)
		}
		if _, ok := knownRuntimeDivergences[name]; ok {
			t.Errorf("%s is both pinned and a known divergence: it is one or the other", name)
		}
	}
	for name := range unorderedRuntimeResults {
		if _, ok := ref.Results[name]; !ok {
			t.Errorf("%s is listed as unordered and is not in the probe", name)
		}
	}

	for _, name := range names {
		want, answered := ref.Results[name]
		if !answered {
			continue // runtimeManifest reported it; comparing would read it as null
		}
		got, evalErr := engine.EvaluateParsedExpression(context.Background(), lib, name, nil, nil)

		if pin, pinned := pinnedRuntimeAnswers[name]; pinned {
			switch {
			case evalErr != nil:
				t.Errorf("%s is pinned at %s (%s) and now fails: %v", name, pin.got, pin.why, evalErr)
			case describeRuntimeValue(got) != pin.got:
				t.Errorf("%s is pinned at %s and now answers %s. The reference cannot settle it (%s): "+
					"check the change against the specification, then move the pin",
					name, pin.got, describeRuntimeValue(got), pin.why)
			}
			continue
		}

		var mismatch string
		switch {
		case want != nil && want.Error != "":
			if evalErr == nil {
				mismatch = fmt.Sprintf("the reference fails (%s: %s) and this engine answers %s",
					want.Error, want.Message, describeRuntimeValue(got))
			}
		case evalErr != nil:
			mismatch = fmt.Sprintf("this engine fails: %v", evalErr)
		default:
			mismatch = runtimeMismatch(t, name, want, got, unorderedRuntimeResults[name] == "")
		}

		known, isKnown := knownRuntimeDivergences[name]
		switch {
		case mismatch == "" && isKnown:
			t.Errorf("%s now agrees with the reference (it diverged because %s): remove it from "+
				"knownRuntimeDivergences and from runtime.divergences.md", name, known.why)
		case mismatch != "" && !isKnown:
			t.Errorf("%s: %s", name, mismatch)
		case mismatch != "" && isKnown:
			if evalErr != nil {
				t.Errorf("%s was a known divergence (%s, because %s) and now fails: %v",
					name, known.got, known.why, evalErr)
			} else if d := describeRuntimeValue(got); d != known.got {
				t.Errorf("%s was a known divergence answering %s, because %s; it now answers %s — "+
					"still not the reference's, but not the one that was understood", name, known.got, known.why, d)
			}
		}
	}
}

// defineName reads the definitions from the source text, deliberately without
// the compiler under test: a manifest derived from the parser would shrink with
// any definition the parser lost.
var (
	defineName  = regexp.MustCompile(`(?m)^define\s+(\w+)\s*:`)
	libraryName = regexp.MustCompile(`(?m)^library\s+(\w+)`)
)

// runtimeManifest returns the probe's definitions in source order, after
// checking that the reference has exactly those. A reference regenerated from
// another file, or truncated, would otherwise pass by comparing what little it
// has.
func runtimeManifest(t *testing.T, src string, ref runtimeReference) []string {
	t.Helper()
	matches := defineName.FindAllStringSubmatch(src, -1)
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, m[1])
	}
	if len(names) == 0 {
		t.Fatal("the runtime probe defines nothing")
	}
	for _, n := range names {
		if _, ok := ref.Results[n]; !ok {
			t.Errorf("%s is in the probe and the reference has no answer for it — regenerate it", n)
		}
	}
	if len(ref.Results) != len(names) {
		t.Errorf("the reference answers %d definitions and the probe has %d", len(ref.Results), len(names))
	}
	return names
}

// runtimeMismatch says how got differs from the reference's answer, or "" when
// it does not.
func runtimeMismatch(t *testing.T, name string, want *runtimeRefValue, got fptypes.Value, ordered bool) string {
	t.Helper()
	if ok, why := runtimeMatches(t, name, want, got, ordered); !ok {
		return fmt.Sprintf("this engine answers %s, the reference %s (%s)",
			describeRuntimeValue(got), describeRuntimeRef(want), why)
	}
	return ""
}

// runtimeMatches compares one value. ordered applies to a list at this level;
// lists nested inside it are always compared in order.
func runtimeMatches(t *testing.T, name string, want *runtimeRefValue, got fptypes.Value, ordered bool) (matched bool, compared string) {
	t.Helper()
	if want == nil {
		return got == nil, "the reference answers null"
	}
	if got == nil {
		return false, "this engine answers null"
	}
	switch want.Type {
	case "Boolean":
		var b bool
		mustDecode(t, name, want.Value, &b)
		g, ok := got.(fptypes.Boolean)
		return ok && g.Bool() == b, "Boolean"
	case "Integer":
		g, ok := got.(fptypes.Integer)
		return ok && strconv.FormatInt(g.Value(), 10) == stringOf(t, name, want.Value), "Integer"
	case "Decimal":
		g, ok := got.(fptypes.Decimal)
		return ok && sameNumber(t, name, g.Value().String(), stringOf(t, name, want.Value)), "Decimal"
	case "String":
		g, ok := got.(fptypes.String)
		return ok && g.Value() == stringOf(t, name, want.Value), "String"
	case "Quantity":
		g, ok := got.(fptypes.Quantity)
		return ok && g.Unit() == want.Unit &&
			sameNumber(t, name, g.Value().String(), stringOf(t, name, want.Value)), "Quantity"
	case "DateTime":
		g, ok := got.(fptypes.DateTime)
		if !ok {
			return false, "DateTime"
		}
		instant := javaDateTime(t, name, want)
		// The offset is part of the answer, not only the instant: the same
		// moment written at -03:00 is not what the reference returned at Z.
		// EffectiveOffset, not ToTime's zone: ToTime puts a value with no offset
		// at UTC, which would agree with a reference at Z by accident.
		gotOffset, hasOffset := g.EffectiveOffset()
		_, wantOffset := instant.Zone()
		return hasOffset && g.ToTime().Equal(instant) && gotOffset*60 == wantOffset &&
				precisionName(g.Precision()) == want.Precision,
			"same instant, offset and precision"
	case "List":
		var elems []*runtimeRefValue
		mustDecode(t, name, want.Elements, &elems)
		list, ok := got.(cqltypes.List)
		g := list.Values
		if !ok || len(g) != len(elems) {
			return false, fmt.Sprintf("a list of %d", len(elems))
		}
		if ordered {
			for i, w := range elems {
				if ok, _ := runtimeMatches(t, name, w, g[i], true); !ok {
					return false, fmt.Sprintf("the same elements in the same order, from element %d", i)
				}
			}
			return true, ""
		}
		// Unordered only where unorderedRuntimeResults says the specification
		// fixes no order: demanding the reference's order there would report a
		// difference the specification does not make.
		used := make([]bool, len(g))
	next:
		for _, w := range elems {
			for i, e := range g {
				if !used[i] {
					if ok, _ := runtimeMatches(t, name, w, e, true); ok {
						used[i] = true
						continue next
					}
				}
			}
			return false, "the same elements"
		}
		return true, ""
	}
	// Failing loudly is the point: a type with no comparison here would
	// otherwise be counted as compared.
	t.Fatalf("%s: the reference answers a %s, which this test cannot compare yet", name, want.Type)
	return false, ""
}

// describeRuntimeValue writes a value of this engine with its type, the way the
// known divergences pin it. It reads the values but decides nothing about them.
func describeRuntimeValue(v fptypes.Value) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case fptypes.Boolean:
		return fmt.Sprintf("Boolean(%t)", x.Bool())
	case fptypes.Integer:
		return fmt.Sprintf("Integer(%d)", x.Value())
	case fptypes.Decimal:
		return fmt.Sprintf("Decimal(%s)", x.Value().String())
	case fptypes.String:
		return fmt.Sprintf("String(%q)", x.Value())
	case fptypes.Quantity:
		return fmt.Sprintf("Quantity(%s '%s')", x.Value().String(), x.Unit())
	case fptypes.DateTime:
		// ToTime places a value with no offset at UTC; written that way it would
		// claim a Z the value does not have.
		if _, ok := x.EffectiveOffset(); !ok {
			return fmt.Sprintf("DateTime(%s/%s, no offset)",
				x.ToTime().Format("2006-01-02T15:04:05.999999999"), precisionName(x.Precision()))
		}
		return fmt.Sprintf("DateTime(%s/%s)", x.ToTime().Format(time.RFC3339Nano), precisionName(x.Precision()))
	case cqltypes.List:
		parts := make([]string, len(x.Values))
		for i, e := range x.Values {
			parts[i] = describeRuntimeValue(e)
		}
		return "List[" + strings.Join(parts, ", ") + "]"
	}
	return fmt.Sprintf("Unencoded(%T)", v)
}

func describeRuntimeRef(v *runtimeRefValue) string {
	if v == nil {
		return "null"
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// precisionName is the reference's name for each precision.
func precisionName(p fptypes.DateTimePrecision) string {
	switch p {
	case fptypes.DTYearPrecision:
		return "year"
	case fptypes.DTMonthPrecision:
		return "month"
	case fptypes.DTDayPrecision:
		return "day"
	case fptypes.DTHourPrecision:
		return "hour"
	case fptypes.DTMinutePrecision:
		return "minute"
	case fptypes.DTSecondPrecision:
		return "second"
	case fptypes.DTMillisPrecision:
		return "millisecond"
	}
	return fmt.Sprintf("precision(%d)", int(p))
}

// javaDateTime reads the reference's DateTime from its instant, which
// Evaluate.java writes in full at every precision. Its value is the engine's
// toString(), which drops the offset at some precisions and keeps it at others.
func javaDateTime(t *testing.T, name string, v *runtimeRefValue) time.Time {
	t.Helper()
	if v.Instant == "" {
		t.Fatalf("%s: the reference DateTime carries no instant — regenerate it", name)
	}
	instant, err := time.Parse(time.RFC3339Nano, v.Instant)
	if err != nil {
		t.Fatalf("%s: reading the reference DateTime: %v", name, err)
	}
	return instant
}

func sameNumber(t *testing.T, name, a, b string) bool {
	t.Helper()
	x, ok := new(big.Rat).SetString(a)
	if !ok {
		t.Fatalf("%s: %q is not a number", name, a)
	}
	y, ok := new(big.Rat).SetString(b)
	if !ok {
		t.Fatalf("%s: %q is not a number", name, b)
	}
	return x.Cmp(y) == 0
}

func stringOf(t *testing.T, name string, raw json.RawMessage) string {
	t.Helper()
	var s string
	mustDecode(t, name, raw, &s)
	return s
}

func mustDecode(t *testing.T, name string, raw json.RawMessage, into any) {
	t.Helper()
	if err := json.Unmarshal(raw, into); err != nil {
		t.Fatalf("%s: decoding the reference value %s: %v", name, raw, err)
	}
}
