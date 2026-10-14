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
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

// runtimeReference is what a reference runtime answered for each definition of
// testdata/reference/runtime.cql, recorded by internal/referenceharness
// (Evaluate.java, js/evaluate.js). It is committed for the same reason
// referenceELM is: the runtimes need a JVM and Node, and the point is to diff
// against them, not to re-derive them.
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
	Fields    json.RawMessage `json:"fields,omitempty"`
	Error     string          `json:"error,omitempty"`
	Message   string          `json:"message,omitempty"`
}

// runtimeReferences are the runtimes this engine is diffed against, each with
// the version the harness pins and the definitions it is not an oracle for.
//
// The version is checked because an output regenerated with another one is a
// different oracle and has to be reviewed as such, not slipped in under the
// same test. The exclusions are per runtime because they are about the
// runtime, not the definition: the Java engine, as Evaluate.java configures it,
// does not convert units, and cql-execution does.
//
// Both run the ELM the reference translator produces. Where they agree they
// are two runtimes, not two frontends.
var runtimeReferences = []struct {
	name      string
	file      string
	engine    string
	notOracle map[string]string
}{
	{
		name:   "the Java engine",
		file:   "testdata/reference/runtime.expected.json",
		engine: "3.26.0",
		notOracle: map[string]string{
			"QuantityEqual":       "it does not convert units: 1 'm' = 100 'cm' is null there",
			"QuantityEquivalent":  "it does not convert units: it answers false",
			"DistinctQuantity":    "it does not convert units, and keeps both",
			"AggDistinctQuantity": "it does not convert units, and answers 2",
		},
	},
	{
		name:   "cql-execution",
		file:   "testdata/reference/runtime.expected.js.json",
		engine: "cql-execution 3.3.2",
		notOracle: map[string]string{
			// cql-execution keeps Integer and Decimal as one JavaScript number, so
			// evaluate.js reads a number's type from the translator's annotation.
			// On a row about the type itself, that is the translator answering.
			"ListIntDec": "it cannot tell Integer from Decimal; the type it records is the translator's",
			// Decimal arithmetic runs in binary floating point there, below the
			// scale of 8 the specification gives the type.
			"DecimalDivision": "it divides in binary floating point: 3.3333333333333335",
			"DecimalSum":      "it adds in binary floating point: 0.30000000000000004",
		},
	},
}

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
	"WithAggregate":      {"Integer(6)", "the aggregate loop iterates the unfiltered product and reapplies only where"},
	"WithoutAggregate":   {"Integer(6)", "same cause as WithAggregate"},
	"UnionDuplicates":    {"List[Integer(1), Integer(1), Integer(2)]", "union keeps duplicates from one operand"},
	"ExceptDuplicates":   {"List[Integer(1), Integer(1), Integer(2)]", "except keeps duplicates from the first operand"},
	"AggDistinctNilNull": {"Integer(1)", "distinctCombos keys rows by String(): null and 'nil' collide"},
	"AggDistinctPairs":   {"Integer(3)", "distinctCombos joins keys with a comma that the values can contain"},
	"AggDistinctIntDec":  {"Integer(2)", "distinctCombos compares text, and 1 = 1.0"},
	"AggDistinctOffsets": {"Integer(2)", "distinctCombos compares text, and the two are the same instant"},
	"ListIntDec":         {"List[Integer(1), Decimal(1)]", "a list of Integer and Decimal is not converted to List<Decimal>"},
	"SameDayAcrossOffsets": {"Boolean(false)", "values at different offsets are normalized to UTC; the " +
		"specification normalizes to the evaluation offset, and only below day precision"},
	"AggDistinctQuantity": {"Integer(2)", "distinctCombos compares text, and 1 'm' = 100 'cm'"},
	"DecimalDivision": {"Decimal(3.3333333333333333)", "a quotient is not brought to the Decimal scale of 8; " +
		"the specification gives the type a scale of 8, and both references answer 3.33333333"},
}

// decidedRuntimeAnswers are the definitions where this engine answers
// differently from every reference on purpose, each with its exact answer and
// the reason. The references are not consulted for them: agreeing with them
// would undo the decision, not fix a defect. Any change fails until someone
// argues with the reason and moves the pin.
var decidedRuntimeAnswers = map[string]struct {
	got string
	why string
}{
	"DateTimeYear": {"DateTime(2020-01-01T00:00:00/year, no offset)", "fhirpath gives a DateTime no offset below " +
		"hour precision, and this engine's comparisons depend on it: without the rule a day-precision value was " +
		"normalized as if it were an instant, and answered wrong at UTC+14. Both references assume the " +
		"evaluation offset, as the specification words it"},
}

// disputedRuntimeAnswers are the definitions the references disagree on, each
// with this engine's exact answer and how the disagreement was settled. The
// test checks this engine's answer, and that the references still disagree:
// once they agree, the row is no longer a dispute and has to leave the list.
var disputedRuntimeAnswers = map[string]struct {
	got string
	why string
}{
	"NoReturn": {"List[Integer(1), Integer(1), Integer(2)]", "a single source with no return clause: the " +
		"reference translator emits no return clause, and the Java engine keeps the duplicates; cql-execution " +
		"removes them. The specification's \"by default, queries return a distinct list\" is written about " +
		"the return clause, so this engine follows the translator"},
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
	// Line endings are normalized first, so a checkout that converts them does
	// not read as an edit. The harness hashes the file as committed, with LF.
	src = bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))

	names := runtimeProbeNames(t, string(src))
	inProbe := make(map[string]bool, len(names))
	for _, n := range names {
		inProbe[n] = true
	}

	refs := make([]runtimeReference, len(runtimeReferences))
	for i, r := range runtimeReferences {
		refs[i] = loadRuntimeReference(t, r.file, r.engine, src)
		runtimeCoverage(t, r.name, names, refs[i])
		for name := range r.notOracle {
			if !inProbe[name] {
				t.Errorf("%s is excluded for %s and is not in the probe", name, r.name)
			}
		}
	}

	// One evaluation time for every reference, or their answers to anything
	// that reads it are not answers to the same question.
	evaluatedAt, err := time.Parse(time.RFC3339, refs[0].EvaluationDateTime)
	if err != nil {
		t.Fatalf("reading the reference evaluation time %q: %v", refs[0].EvaluationDateTime, err)
	}
	for i, ref := range refs[1:] {
		other, perr := time.Parse(time.RFC3339, ref.EvaluationDateTime)
		_, o1 := evaluatedAt.Zone()
		_, o2 := other.Zone()
		if perr != nil || !other.Equal(evaluatedAt) || o1 != o2 {
			t.Fatalf("%s evaluates at %q and %s at %q", runtimeReferences[0].name,
				refs[0].EvaluationDateTime, runtimeReferences[i+1].name, ref.EvaluationDateTime)
		}
	}

	engine := NewEngine(WithEvaluationTimestamp(evaluatedAt))
	lib, err := engine.Parse(string(src))
	if err != nil {
		t.Fatalf("parsing the runtime probe: %v", err)
	}

	listed := func(list, name string) {
		if !inProbe[name] {
			t.Errorf("%s is listed in %s and is not in the probe", name, list)
		}
	}
	for name := range knownRuntimeDivergences {
		listed("knownRuntimeDivergences", name)
	}
	for name := range decidedRuntimeAnswers {
		listed("decidedRuntimeAnswers", name)
		if _, ok := knownRuntimeDivergences[name]; ok {
			t.Errorf("%s is both decided and a known divergence: it is one or the other", name)
		}
	}
	for name := range unorderedRuntimeResults {
		listed("unorderedRuntimeResults", name)
	}
	for name := range disputedRuntimeAnswers {
		listed("disputedRuntimeAnswers", name)
		if _, ok := knownRuntimeDivergences[name]; ok {
			t.Errorf("%s is both disputed and a known divergence: it is one or the other", name)
		}
	}

	for _, name := range names {
		got, evalErr := engine.EvaluateParsedExpression(context.Background(), lib, name, nil, nil)

		if pin, decided := decidedRuntimeAnswers[name]; decided {
			switch {
			case evalErr != nil:
				t.Errorf("%s is decided at %s (%s) and now fails: %v", name, pin.got, pin.why, evalErr)
			case describeRuntimeValue(got) != pin.got:
				t.Errorf("%s is decided at %s and now answers %s. The decision: %s. "+
					"Argue with it before moving the pin", name, pin.got, describeRuntimeValue(got), pin.why)
			}
			continue
		}

		// What this engine's answer has against each reference that can settle
		// the definition.
		var agree, disagree []string
		unanswered := false
		for i, r := range runtimeReferences {
			if _, excluded := r.notOracle[name]; excluded {
				continue
			}
			want, answered := refs[i].Results[name]
			if !answered {
				unanswered = true // runtimeCoverage reported it; comparing would read it as null
				continue
			}
			if m := runtimeAnswerMismatch(t, name, want, got, evalErr); m == "" {
				agree = append(agree, r.name)
			} else {
				disagree = append(disagree, r.name+": "+m)
			}
		}

		if pin, disputed := disputedRuntimeAnswers[name]; disputed {
			switch {
			case len(agree) == 0 || len(disagree) == 0:
				t.Errorf("%s is listed as disputed, and the references no longer disagree: "+
					"remove it from disputedRuntimeAnswers and compare it like any other row", name)
			case evalErr != nil:
				t.Errorf("%s is disputed, pinned at %s (%s), and now fails: %v", name, pin.got, pin.why, evalErr)
			case describeRuntimeValue(got) != pin.got:
				t.Errorf("%s is disputed, pinned at %s, and now answers %s. How it was settled: %s",
					name, pin.got, describeRuntimeValue(got), pin.why)
			}
			continue
		}

		known, isKnown := knownRuntimeDivergences[name]
		switch {
		case len(agree) == 0 && len(disagree) == 0 && unanswered:
			// Reported as a reference to regenerate. Asking for a decided pin here
			// would be the wrong fix, and would hide the gap for good.
		case len(agree) == 0 && len(disagree) == 0:
			t.Errorf("%s: no reference can settle it — decide it and pin it in decidedRuntimeAnswers", name)
		case len(agree) > 0 && len(disagree) > 0:
			// Never settled by majority: two runtimes that share a translator can
			// share its mistake.
			t.Errorf("%s: the references disagree — this engine agrees with %s, and not with %s. "+
				"Settle it against the specification", name, strings.Join(agree, ", "), strings.Join(disagree, "; "))
		case len(disagree) == 0 && isKnown:
			t.Errorf("%s now agrees with every reference that can settle it (it diverged because %s): remove it from "+
				"knownRuntimeDivergences and from runtime.divergences.md", name, known.why)
		case len(agree) == 0 && !isKnown:
			t.Errorf("%s: %s", name, strings.Join(disagree, "; "))
		case len(agree) == 0 && isKnown:
			if evalErr != nil {
				t.Errorf("%s was a known divergence (%s, because %s) and now fails: %v",
					name, known.got, known.why, evalErr)
			} else if d := describeRuntimeValue(got); d != known.got {
				t.Errorf("%s was a known divergence answering %s, because %s; it now answers %s — "+
					"still not the references', but not the one that was understood", name, known.got, known.why, d)
			}
		}
	}
}

// loadRuntimeReference reads one reference's answers and checks that they are
// answers to this probe, from the pinned version of that runtime.
func loadRuntimeReference(t *testing.T, file, engine string, src []byte) runtimeReference {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("reading %s: %v", file, err)
	}
	var ref runtimeReference
	if err = json.Unmarshal(data, &ref); err != nil {
		t.Fatalf("decoding %s: %v", file, err)
	}
	if ref.Engine != engine {
		t.Fatalf("%s was produced by %q, the harness pins %q", file, ref.Engine, engine)
	}
	// The names alone cannot tell a stale reference: a definition edited without
	// regenerating keeps its name, and its old answer would read as a divergence
	// of this engine — one that could then be pinned as understood.
	if sum := sha256.Sum256(src); hex.EncodeToString(sum[:]) != ref.SourceSha256 {
		t.Fatalf("runtime.cql changed since %s was produced — regenerate it", file)
	}
	if m := libraryName.FindStringSubmatch(string(src)); len(m) < 2 || m[1] != ref.Library {
		t.Fatalf("%s answers for library %q, which is not the probe's", file, ref.Library)
	}
	return ref
}

// runtimeAnswerMismatch says how this engine's answer, or its failure, differs
// from one reference's, or "" when it does not.
func runtimeAnswerMismatch(t *testing.T, name string, want *runtimeRefValue, got fptypes.Value, evalErr error) string {
	t.Helper()
	switch {
	case want != nil && want.Error != "":
		if evalErr == nil {
			return fmt.Sprintf("it fails (%s: %s) and this engine answers %s",
				want.Error, want.Message, describeRuntimeValue(got))
		}
		return ""
	case evalErr != nil:
		return fmt.Sprintf("this engine fails: %v", evalErr)
	}
	return runtimeMismatch(t, name, want, got, unorderedRuntimeResults[name] == "")
}

// defineName reads the definitions from the source text, deliberately without
// the compiler under test: a manifest derived from the parser would shrink with
// any definition the parser lost.
var (
	defineName  = regexp.MustCompile(`(?m)^define\s+(\w+)\s*:`)
	libraryName = regexp.MustCompile(`(?m)^library\s+(\w+)`)
)

// runtimeProbeNames returns the probe's definitions in source order.
func runtimeProbeNames(t *testing.T, src string) []string {
	t.Helper()
	matches := defineName.FindAllStringSubmatch(src, -1)
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, m[1])
	}
	if len(names) == 0 {
		t.Fatal("the runtime probe defines nothing")
	}
	return names
}

// runtimeCoverage checks that a reference answers exactly the probe's
// definitions. A reference regenerated from another file, or truncated, would
// otherwise pass by comparing what little it has.
func runtimeCoverage(t *testing.T, reference string, names []string, ref runtimeReference) {
	t.Helper()
	for _, n := range names {
		if _, ok := ref.Results[n]; !ok {
			t.Errorf("%s is in the probe and %s has no answer for it — regenerate it", n, reference)
		}
	}
	if len(ref.Results) != len(names) {
		t.Errorf("%s answers %d definitions and the probe has %d", reference, len(ref.Results), len(names))
	}
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
		if want.Instant == "" {
			// A reference DateTime with no offset has no instant to compare.
			return false, "a DateTime with no offset, which this test cannot compare"
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
	case "Tuple":
		var fields map[string]*runtimeRefValue
		mustDecode(t, name, want.Fields, &fields)
		g, ok := got.(cqltypes.Tuple)
		if !ok || len(g.Elements) != len(fields) {
			return false, fmt.Sprintf("a Tuple of %d fields", len(fields))
		}
		for k, w := range fields {
			v, present := g.Elements[k]
			if !present {
				return false, "a Tuple with field " + k
			}
			if ok, why := runtimeMatches(t, name, w, v, true); !ok {
				return false, "field " + k + ": " + why
			}
		}
		return true, ""
	case "Time":
		g, ok := got.(fptypes.Time)
		if !ok {
			return false, "Time"
		}
		// Read from the text by component, with this test's own parsing: the
		// references write 10:30 at minute precision and 10:30:00.000 at
		// millisecond, and a missing component is not a zero.
		var h, m, sec, ms int
		parts := strings.SplitN(strings.Replace(stringOf(t, name, want.Value), ".", ":", 1), ":", 4)
		for i, dst := range []*int{&h, &m, &sec, &ms} {
			if i < len(parts) {
				if _, err := fmt.Sscanf(parts[i], "%d", dst); err != nil {
					return false, fmt.Sprintf("a Time this test cannot read: %v", err)
				}
			}
		}
		return g.Hour() == h && g.Minute() == m && g.Second() == sec && g.Millisecond() == ms &&
			timePrecisionName(g.Precision()) == want.Precision, "same components and precision"
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
	// Never counted as agreeing: a type with no comparison here would otherwise
	// pass as compared. Reported as a mismatch rather than a Fatalf, so one
	// unusual row does not stop every other definition from being compared.
	return false, fmt.Sprintf("a %s, which this test cannot compare yet", want.Type)
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
	case fptypes.Time:
		return fmt.Sprintf("Time(%02d:%02d:%02d.%03d/%s)", x.Hour(), x.Minute(), x.Second(), x.Millisecond(),
			timePrecisionName(x.Precision()))
	case cqltypes.Tuple:
		keys := make([]string, 0, len(x.Elements))
		for k := range x.Elements {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = k + ": " + describeRuntimeValue(x.Elements[k])
		}
		return "Tuple{" + strings.Join(parts, ", ") + "}"
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

// timePrecisionName is the references' name for each Time precision.
func timePrecisionName(p fptypes.TimePrecision) string {
	switch p {
	case fptypes.HourPrecision:
		return "hour"
	case fptypes.MinutePrecision:
		return "minute"
	case fptypes.SecondPrecision:
		return "second"
	case fptypes.MillisPrecision:
		return "millisecond"
	}
	return fmt.Sprintf("precision(%d)", int(p))
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
