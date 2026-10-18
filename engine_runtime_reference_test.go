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
	"slices"
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
			// The engine fails reading a let from the aggregate expression:
			// State.resolveVariable returns null for it. cql-execution and
			// this engine both bind it.
			"AggregateLetWithout":      "it fails resolving a let inside aggregate: a NullPointerException in State.resolveVariable",
			"AggregateDistinctLetWith": "it fails resolving a let inside aggregate distinct, after a with: the same NullPointerException",
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
			// The successor of the largest value is an error in the
			// specification, and in the Java engine; cql-execution answers null.
			"IntegerSuccessorOfMaximum": "it answers null where the specification raises an error",
			"LongSuccessorOfMaximum":    "it answers null where the specification raises an error",
			// Decimal arithmetic runs in binary floating point there, below the
			// scale of 8 the specification gives the type.
			"DecimalDivision":                 "it divides in binary floating point: 3.3333333333333335",
			"DecimalSum":                      "it adds in binary floating point: 0.30000000000000004",
			"DecimalLnUnscaled":               "it computes Decimal in binary floating point",
			"DecimalModScaled":                "it computes Decimal in binary floating point",
			"ToDecimalScaleKept":              "it computes Decimal in binary floating point",
			"ToDecimalPrecision":              "it computes Decimal in binary floating point",
			"ToDecimalTrailingZeros":          "it computes Decimal in binary floating point",
			"ToDecimalPastTheLargest":         "it computes Decimal in binary floating point",
			"ToDecimalPastTheSmallest":        "it computes Decimal in binary floating point",
			"ToDecimalLeadingZero":            "it computes Decimal in binary floating point",
			"ToDecimalKeepsItsZeros":          "it computes Decimal in binary floating point",
			"ToDecimalOutOfRange":             "it computes Decimal in binary floating point",
			"ToDecimalScaled":                 "it computes Decimal in binary floating point",
			"ToDecimalNegativeScaled":         "it computes Decimal in binary floating point",
			"LogOfAScaledString":              "it computes Decimal in binary floating point",
			"DecimalSumOfOne":                 "it computes Decimal in binary floating point",
			"DecimalProductOfOne":             "it computes Decimal in binary floating point",
			"DecimalAvgOfOne":                 "it computes Decimal in binary floating point",
			"DecimalSumOfTwo":                 "it computes Decimal in binary floating point",
			"DecimalLogNearOne":               "it computes Decimal in binary floating point",
			"IntegerNegativePower":            "it cannot tell Integer from Decimal; the type it records is the translator's",
			"DecimalWholePowerExact":          "it computes Decimal in binary floating point",
			"DecimalMultiplyScaled":           "it computes Decimal in binary floating point",
			"DecimalMedianEvenScale":          "it computes Decimal in binary floating point",
			"DecimalLogNearInteger":           "it computes Decimal in binary floating point",
			"DecimalDivisionNoDoubleRounding": "it computes Decimal in binary floating point",
			"DecimalPowerShortestFloat":       "it computes Decimal in binary floating point",
			"DecimalDivisionNegative":         "it computes Decimal in binary floating point",
			"DecimalDivisionEquals":           "it computes Decimal in binary floating point",
			"IntegerDivisionScale":            "it computes Decimal in binary floating point",
			"DecimalProductScale":             "it computes Decimal in binary floating point",
			"DecimalAvgScale":                 "it computes Decimal in binary floating point",
			"DecimalAvgNegativeScale":         "it computes Decimal in binary floating point",
			"DecimalProductAggregateScale":    "it computes Decimal in binary floating point",
			"DecimalLogScale":                 "it computes Decimal in binary floating point",
			"DecimalPowerScale":               "it computes Decimal in binary floating point",
			"DecimalVarianceScale":            "it computes Decimal in binary floating point",
			"DecimalStdDevScale":              "it computes Decimal in binary floating point",
			"DecimalPopulationVarianceScale":  "it computes Decimal in binary floating point",
			"DecimalAbsOfFloored":             "it computes Decimal in binary floating point",
		},
	},
}

// knownRuntimeDivergences are the definitions where this engine is known to
// answer differently from the reference, each with this engine's exact answer
// as pinnedAs writes it: describeRuntimeValue, with the elements sorted for a
// definition in unorderedRuntimeResults.
//
// Pinning the answer, not just the name, is what keeps the list honest: a
// divergence that changes into a different wrong answer fails here, and one
// that gets fixed fails too, until it is removed from this list and from
// runtime.divergences.md.
var knownRuntimeDivergences = map[string]struct {
	got string
	why string
}{
	// None at present: every definition the references and the specification
	// agree on, this engine answers as they do.
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
	"DecimalLnUnscaled": {"Decimal(0.6931471805599453)", "Ln is not brought to the Decimal scale: the " +
		"conformance suite asks Round(Ln(1000), 8) for 6.90775528, and flooring first gives 6.90775527"},
	"DecimalGeometricMeanUnscaled": {"Decimal(3.63424118566428)", "the geometric mean is the exact root, not " +
		"Power(Product, 1/n) with 1/n floored, which makes the geometric mean of {1, 2, 4} 1.99999998; the " +
		"numeric and Quantity aggregates agree on it"},
	"ConvertsToDecimalOutOfRange": {"Boolean(false)", "ConvertsToDecimal is whether ToDecimal succeeds, and " +
		"ToDecimal of a string past the largest Decimal is null in both engines; the Java engine answers true here " +
		"all the same"},
	"IntegerNegateMinimum":       {"null", overflowDecided + ": -(minimum Integer) is past the largest one"},
	"IntegerAbsMinimum":          {"null", overflowDecided + ": the specification names Abs(minimum Integer) as null"},
	"IntegerSumOverflow":         {"null", overflowDecided + ", for the sum as a whole"},
	"IntegerProductOverflow":     {"null", overflowDecided + ", for the product as a whole"},
	"IntegerTruncatePastTheType": {"null", overflowDecided + ", as Floor and Ceiling past it are in the conformance suite"},
	"LongAddOverflow":            {"null", overflowDecided + "; cql-execution concatenates the digits"},
	"DecimalWholePowerExact": {"Decimal(0.0441)", "a whole-number power is computed in decimal: through a float " +
		"0.21 ^ 2 is 0.044099999999999993, which floors to 0.04409999 in the Java engine, where 0.21 * 0.21 is 0.0441"},
	"DecimalLogNearInteger": {"Decimal(3)", "an exact power has a whole-number logarithm, checked in decimal: " +
		"ln 1000 / ln 10 is the float 2.9999999999999996, which floors to 2.99999999 in the Java engine; the " +
		"logarithm is 3. A value merely near a power floors as the float says"},
	"DecimalPowerShortestFloat": {"Decimal(0.3)", "a float result is read as the shortest decimal that " +
		"prints as it, not its exact binary value: the conformance suite asks Power(10, -7) for 0.0000001, and " +
		"the float 1e-7 is 0.0000000999…, which floors to 0.00000009"},
	"QuantityDivisionUnscaled": {"Quantity(3.3333333333333333 'mg')", "the Decimal scale is the Decimal " +
		"type's, and a Quantity keeps its digits: a dose in a small unit does not become 0 in that unit"},
	"ListDateDateTime": {"Boolean(false)", "the numeric promotions to a list's common type are applied and Date to " +
		"DateTime is not: a Date compared at the day is read as written and a DateTime is moved to an offset, so " +
		"promoting it would change what every comparison it reaches answers"},
}

// disputedRuntimeAnswers are the definitions the references disagree on, each
// with this engine's exact answer and how the disagreement was settled — by
// the conformance suite or the specification, never by majority. The test
// checks this engine's answer, and that the references still disagree: once
// they agree, the row is no longer a dispute and has to leave the list.
//
// A reference that answers wrong here is not excluded with notOracle. That is
// for what a runtime cannot answer at all, and has no check that it is still
// needed; this list fails the day the reference is fixed.
//
// Each row names the references that agree with this engine, so that the
// test can tell the reference in the wrong being fixed from the one in the
// right changing: the first ends the dispute, the second reopens it.
var disputedRuntimeAnswers = map[string]struct {
	got   string
	agree []string
	why   string
}{
	"AsAny":                           {`String("a")`, []string{"cql-execution"}, anySettled},
	"IsAny":                           {"Boolean(true)", []string{"cql-execution"}, anySettled},
	"CastAsAny":                       {`String("a")`, []string{"cql-execution"}, anySettled},
	"AsAnyInArithmetic":               {"Integer(6)", []string{"cql-execution"}, anySettled},
	"ListOfIntegersIsNotListOfString": {"Boolean(false)", []string{"cql-execution"}, elementTypeSettled},
	"IntervalOfIntegersIsNotIntervalOfString": {"Boolean(false)", []string{"cql-execution"},
		elementTypeSettled},
	"DefinitionUnderAnAlias":   {"List[Integer(2)]", []string{"the Java engine"}, scopeSettled},
	"DefinitionUnderALet":      {"List[Integer(2)]", []string{"the Java engine"}, scopeSettled},
	"DefinitionUnderAWhere":    {"List[Integer(10), Integer(20)]", []string{"the Java engine"}, scopeSettled},
	"DefinitionUnderAnOperand": {"Integer(32)", []string{"the Java engine"}, scopeSettled},
	"IntegerAddOverflow":       {"null", []string{"cql-execution"}, overflowSettled},
	"IntegerMultiplyOverflow":  {"null", []string{"cql-execution"}, overflowSettled},
	"IntegerDivOverflow":       {"null", []string{"cql-execution"}, overflowSettled},
	"IntegerPowerOverflow":     {"null", []string{"cql-execution"}, overflowSettled},
	"LongMultiplyOverflow":     {"null", []string{"cql-execution"}, overflowSettled},
	"NoReturn": {"List[Integer(1), Integer(1), Integer(2)]", []string{"the Java engine"},
		noReturnSettled + ". This row has no sort, which the conformance cases have; it rests on them by " +
			"analogy, since a sort orders the elements a query keeps and does not decide which"},
	"NoReturnSortedConformance": {"List[Integer(1), Integer(1), Integer(2), Integer(4), Integer(5), Integer(6)]",
		[]string{"the Java engine"}, noReturnSettled + ". This row is simpleSortAsc itself, which cql-execution fails"},
	"NoReturnSortedDescConformance": {"List[Integer(6), Integer(5), Integer(4), Integer(2), Integer(1), Integer(1)]",
		[]string{"the Java engine"}, noReturnSettled + ". This row is simpleSortDesc itself, which cql-execution fails"},
	"UnionNullFirst":   {"List[Integer(1), Integer(2)]", []string{"the Java engine"}, setOpNullSettled},
	"UnionNullSecond":  {"List[Integer(1), Integer(2)]", []string{"the Java engine"}, setOpNullSettled},
	"ExceptNullSecond": {"List[Integer(1), Integer(2)]", []string{"the Java engine"}, setOpNullSettled},
}

// setOpNullSettled is how the dispute over a null operand of union and except
// was settled.
const setOpNullSettled = "a null operand of union, or the second of except, is an empty list, and the " +
	"result still has set semantics: Union \"if either argument is null, it is considered an empty list\" " +
	"and returns \"all unique elements\"; Except \"the operation is performed as though the second argument " +
	"was an empty list\" and \"each element will appear in the result at most once\" (CQL 1.5, list " +
	"operators). The Java engine answers so; cql-execution returns the other operand with its duplicates"

// anySettled is how the rows on Any were settled.
const anySettled = "every value is of type Any, the type every other one descends from (CQL 1.5, types: \"System.Any " +
	"is the base type of all types\"), so x as Any and cast x as Any are x and x is Any is true. cql-execution " +
	"answers so for these rows, a scalar each, and null for a list or an interval as Any; the Java engine fails " +
	"at run time resolving the type, org.opencds.cqf.cql.engine.runtime.Any"

// elementTypeSettled is how the rows on the element type of a list or an
// interval were settled.
const elementTypeSettled = "a list is a List<T>, and an interval an Interval<T>, when what it holds is of T: a " +
	"List<Integer> is not a List<String>. cql-execution answers so; the Java engine answers true for any list " +
	"or interval, whatever its elements"

// scopeSettled is how the rows on the scope of a definition were settled.
const scopeSettled = "a definition evaluates in the scope of its library, not in the query or the function it " +
	"is referenced from: an expression definition is a library-level declaration, and the names in it are the " +
	"library's. The Java engine evaluates it so; cql-execution resolves its names in the caller's scope, where an " +
	"alias, a let or an operand of the same name stands in for them"

// overflowDecided is why a result past its type is null where neither
// reference answers so.
const overflowDecided = "a result past its type, 32 bits for an Integer and 64 for a Long, is null, as the " +
	"specification has it; the Java engine wraps it, and cql-execution answers it whole"

// overflowSettled is how the overflow rows cql-execution agrees on were
// settled.
const overflowSettled = "a result past its type is null: \"if the result of the operation cannot be " +
	"represented, the result is null\" (CQL 1.5, arithmetic operators), for an Integer of 32 bits and a Long " +
	"of 64. The Java engine wraps it; cql-execution answers null here, and not for every operator"

// noReturnSettled is how the NoReturn dispute was settled.
const noReturnSettled = "a single source with no return clause keeps its duplicates: the Java engine keeps them, " +
	"cql-execution removes them, and the conformance suite keeps them (CqlListOperatorsTest simpleSortAsc, " +
	"({4, 5, 1, 6, 2, 1}) sL sort asc = {1, 1, 2, 4, 5, 6}). The Developer's Guide says \"if no return clause is " +
	"specified, duplicates will be eliminated\", in its section on multi-source queries, where the translator " +
	"does supply a distinct return clause"

// unorderedRuntimeResults are the definitions whose list result is compared as
// a multiset, each with the reason its order is not part of the answer. Every
// other list is compared in order: a sort that returned its elements reversed
// would pass an unordered comparison, and that is a defect worth finding.
var unorderedRuntimeResults = map[string]string{
	"ReturnDefault":         "distinct fixes which elements remain, not their order",
	"ReturnDistinct":        "same as ReturnDefault",
	"UnionDuplicates":       "Union: no expectation that the order of elements in the inputs will be preserved",
	"ExceptDuplicates":      "Except is a set operation",
	"IntersectDuplicates":   "Intersect is a set operation",
	"DistinctQuantity":      "same as ReturnDefault",
	"DistinctOffsets":       "same as ReturnDefault",
	"UnionNullFirst":        "Union: no expectation that the order of elements in the inputs will be preserved",
	"UnionNullSecond":       "same as UnionNullFirst",
	"ExceptNullSecond":      "Except is a set operation",
	"UnionNullElements":     "same as UnionNullFirst",
	"ExceptNullElement":     "Except is a set operation",
	"IntersectNullElements": "Intersect is a set operation",
	"UnionEmptySecond":      "same as UnionNullFirst",
	"UnionDecimalScale":     "same as UnionNullFirst",
	"ExceptDecimalScale":    "Except is a set operation",
}

// TestRuntimeMatchesTheReference covers what this engine answers against what
// the reference engine answers, value by value.
//
// The comparison is written here rather than borrowed from the engine: decimals
// are compared with math/big, instants with time, and nothing goes through this
// engine's String or Equal. An oracle that used them would inherit the defects
// it exists to find — aggregate distinct was wrong precisely because it trusted
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
	referenceNamed := map[string]bool{}
	for _, r := range runtimeReferences {
		referenceNamed[r.name] = true
	}
	for name, pin := range disputedRuntimeAnswers {
		for _, who := range pin.agree {
			if !referenceNamed[who] {
				t.Errorf("%s is disputed with %q agreeing, which names no reference", name, who)
			}
		}
		if len(pin.agree) == 0 {
			t.Errorf("%s is disputed with no reference agreeing: that is a known divergence, not a dispute", name)
		}
		if _, ok := decidedRuntimeAnswers[name]; ok {
			t.Errorf("%s is both disputed and decided: it is one or the other", name)
		}
		for _, r := range runtimeReferences {
			if _, ok := r.notOracle[name]; ok {
				t.Errorf("%s is disputed and excluded for %s: a reference that answers wrong stays in the row", name, r.name)
			}
		}
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
			case pinnedAs(name, got) != pin.got:
				t.Errorf("%s is decided at %s and now answers %s. The decision: %s. "+
					"Argue with it before moving the pin", name, pin.got, pinnedAs(name, got), pin.why)
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
			// This engine first: a failure here is a regression of this
			// engine's, not a change in the references, and reading it as
			// one would ask for the pin that records the settlement to go.
			switch {
			case evalErr != nil:
				t.Errorf("%s is disputed, pinned at %s (%s), and now fails: %v", name, pin.got, pin.why, evalErr)
			case pinnedAs(name, got) != pin.got:
				t.Errorf("%s is disputed, pinned at %s, and now answers %s. How it was settled: %s",
					name, pin.got, pinnedAs(name, got), pin.why)
			case unanswered:
				// Reported as a reference to regenerate. Read as agreement, a
				// missing answer would ask for the pin to go.
			case len(disagree) == 0:
				t.Errorf("%s is listed as disputed, and every reference now answers as this engine does: "+
					"remove it from disputedRuntimeAnswers and compare it like any other row", name)
			case len(agree) == 0:
				// The reference that agreed has moved to the other side. This
				// engine is alone against every reference now, which is no
				// longer the dispute that was settled.
				t.Errorf("%s was disputed with %s agreeing, and no reference agrees now. It is no longer "+
					"the dispute that was settled: argue it again against the evidence, or move it to "+
					"knownRuntimeDivergences. How it was settled: %s", name, strings.Join(pin.agree, ", "), pin.why)
			case !sameNames(agree, pin.agree):
				t.Errorf("%s is disputed with %s agreeing, and now %s agree. How it was settled: %s",
					name, strings.Join(pin.agree, ", "), strings.Join(agree, ", "), pin.why)
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
			} else if d := pinnedAs(name, got); d != known.got {
				t.Errorf("%s was a known divergence answering %s, because %s; it now answers %s — "+
					"still not the references', but not the one that was understood", name, known.got, known.why, d)
			}
		}
	}
}

// sameNames reports whether two lists of reference names hold the same names,
// in whatever order.
func sameNames(a, b []string) bool {
	x, y := slices.Clone(a), slices.Clone(b)
	slices.Sort(x)
	slices.Sort(y)
	return slices.Equal(x, y)
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

// pinnedAs is how this engine's answer for a definition is compared with its
// pin. A list from a definition whose order is not part of the answer
// (unorderedRuntimeResults) has its elements sorted, so that a reordering the
// specification allows does not read as a changed answer. Such a pin is
// written in that sorted form — by the text of each element, so Integer(10)
// before Integer(9) — and failures print it the same way.
func pinnedAs(name string, v fptypes.Value) string {
	list, isList := v.(cqltypes.List)
	if unorderedRuntimeResults[name] == "" || !isList {
		return describeRuntimeValue(v)
	}
	parts := make([]string, len(list.Values))
	for i, e := range list.Values {
		parts[i] = describeRuntimeValue(e)
	}
	sort.Strings(parts)
	return "List[" + strings.Join(parts, ", ") + "]"
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
