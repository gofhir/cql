package conformance

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// knownFailure is a test of the suite this engine fails, classified (plan 012,
// docs/plans/012-suite-cql-tests.md).
type knownFailure struct {
	// group is the class of the failure, a key of failureGroups.
	group string
	// errs is how the engine fails it now: with an error, or with a value. A
	// failure that changes from one to the other is not taken for a fix.
	errs bool
	// suite is what the suite asks of the test, as suiteContract writes it. A
	// test upstream changes no longer matches, and has to be classified again.
	suite string
}

// failureGroups are the classes of knownFailures, each with the item of
// docs/plans/README.md that holds it.
var failureGroups = map[string]string{
	"G1":  "1c25 G1: CQL 2.0, after the 1.5.3 this engine is written from (ADR 001)",
	"G2":  "1c25 G2: an error where CQL 1.5.3 and Java answer null",
	"G3a": "1c25 G3a: a component the DateTime does not have answers 0, not null",
	"G3b": "1c25 G3b: years between dates on a leap day",
	"G3c": "1c25 G3c: Distinct and Flatten of a null list answer an empty list, not null",
	"G3d": "1c25 G3d: Skip of a negative number answers the list, not an empty one",
	"G3e": "1c25 G3e: Coalesce of six arguments is evaluated, past its signatures",
	"G3f": "1c25 G3f: ToDate and ToRatio are missing",
	"G3g": "1c25 G3g: div of two Quantities of a unit keeps the unit, not '1'",
	"G3h": "1c25 G3h: tuple equality with a null element is not a conjunction",
	"G3i": "1c25 G3i: a closed null boundary is not the point type's minimum or maximum",
	"G3j": "1c25 G3j: an open null boundary answers false, not null",
	"G5":  "1c25 G5: the suite contradicts CQL 1.5.3, and the engine follows the specification",
}

// knownFailures are the tests of cql-tests f724ee4 this engine fails (plan 012).
// Like upstreamContradictions, each still runs and has to keep failing: when
// one passes, its entry has to go. Each has a definition in
// testdata/snapshots/suitegaps.cql, which holds what the engine answers it and
// what Java and cql-execution do.
var knownFailures = map[string]knownFailure{
	// CqlArithmeticFunctionsTest.xml
	"DivideQuantityByZero":                {group: "G2", errs: true, suite: `1.0.. invalid="" outputs=["null"]`},
	"Exp1000":                             {group: "G2", errs: true, suite: `1.0.. invalid="" outputs=["null"]`},
	"Exp1000D":                            {group: "G2", errs: true, suite: `1.0.. invalid="" outputs=["null"]`},
	"Ln0":                                 {group: "G2", errs: true, suite: `1.4.. invalid="" outputs=["null"]`},
	"LnNeg0":                              {group: "G2", errs: true, suite: `1.4.. invalid="" outputs=["null"]`},
	"PredecessorOf101DPrecision":          {group: "G1", errs: false, suite: `2.0.. invalid="" outputs=["1.00"]`},
	"PredecessorOf1DPrecision":            {group: "G1", errs: false, suite: `2.0.. invalid="" outputs=["0.9"]`},
	"PredecessorOf1QCMPrecision":          {group: "G1", errs: false, suite: `2.0.. invalid="" outputs=["0.9'cm'"]`},
	"PredecessorUnderflowDt":              {group: "G2", errs: true, suite: `1.0.. invalid="" outputs=["null"]`},
	"PredecessorUnderflowT":               {group: "G2", errs: true, suite: `1.0.. invalid="" outputs=["null"]`},
	"SuccessorOf101DPrecision":            {group: "G1", errs: false, suite: `2.0.. invalid="" outputs=["1.02"]`},
	"SuccessorOf1DPrecision":              {group: "G1", errs: false, suite: `2.0.. invalid="" outputs=["1.1"]`},
	"SuccessorOverflowDt":                 {group: "G2", errs: true, suite: `1.0.. invalid="" outputs=["null"]`},
	"SuccessorOverflowT":                  {group: "G2", errs: true, suite: `1.0.. invalid="" outputs=["null"]`},
	"TruncatedDivide10By5DQuantity":       {group: "G3g", errs: false, suite: `1.0.. invalid="" outputs=["2.0 '1'"]`},
	"TruncatedDivide10d1ByNeg3D1Quantity": {group: "G3g", errs: false, suite: `1.0.. invalid="" outputs=["-3.0 '1'"]`},
	"TruncatedDivide414By206DQuantity":    {group: "G3g", errs: false, suite: `1.0.. invalid="" outputs=["2.0 '1'"]`},
	// CqlComparisonOperatorsTest.xml
	"NotEquivalentNullNull":                 {group: "G5", errs: false, suite: `1.0.. invalid="semantic" outputs=[]`},
	"TupleEqDifferentNamesWithOneNullId":    {group: "G3h", errs: false, suite: `1.5.. invalid="" outputs=["false"]`},
	"TupleNotEqDifferingNamesWithOneNullId": {group: "G3h", errs: false, suite: `1.5.. invalid="" outputs=["true"]`},
	// CqlDateTimeOperatorsTest.xml
	"DateTimeComponentFromMonthIsNull":     {group: "G3a", errs: false, suite: `1.4.. invalid="" outputs=["null"]`},
	"YearsBetweenLeapYearDateTimesEquals2": {group: "G3b", errs: false, suite: `1.4.. invalid="" outputs=["2"]`},
	"YearsBetweenLeapYearDatesEquals2":     {group: "G3b", errs: false, suite: `1.4.. invalid="" outputs=["2"]`},
	// CqlIntervalOperatorsTest.xml
	"TestIntegerInMaxIntervalTrue":      {group: "G3i", errs: false, suite: `1.0.. invalid="" outputs=["true"]`},
	"TestIntegerInUnknownIntervalNull":  {group: "G3j", errs: false, suite: `1.0.. invalid="" outputs=["null"]`},
	"TestMaxIntervalEndsFalse":          {group: "G3i", errs: false, suite: `1.0.. invalid="" outputs=["false"]`},
	"TestMaxIntervalOverlapsAfterTrue":  {group: "G3i", errs: false, suite: `1.0.. invalid="" outputs=["true"]`},
	"TestMaxIntervalOverlapsBeforeTrue": {group: "G3i", errs: false, suite: `1.0.. invalid="" outputs=["true"]`},
	"TestMaxIntervalOverlapsTrue":       {group: "G3i", errs: false, suite: `1.0.. invalid="" outputs=["true"]`},
	"TestMaxIntervalStartsFalse":        {group: "G3i", errs: false, suite: `1.0.. invalid="" outputs=["false"]`},
	"TestMeetsAfterNull":                {group: "G3j", errs: false, suite: `1.0.. invalid="" outputs=["null"]`},
	"TestPointFromMaxIntervalError":     {group: "G3i", errs: false, suite: `1.0.. invalid="true" outputs=[]`},
	"TestUnionMaxInterval":              {group: "G3i", errs: false, suite: `1.0.. invalid="" outputs=["Interval[null, null]"]`},
	// CqlListOperatorsTest.xml
	"DistinctNull":                      {group: "G3c", errs: false, suite: `1.0.. invalid="" outputs=["null"]`},
	"EquivalentContainsIsFalse":         {group: "G1", errs: true, suite: `2.0.. invalid="" outputs=["false"]`},
	"EquivalentContainsIsTrue":          {group: "G1", errs: true, suite: `2.0.. invalid="" outputs=["true"]`},
	"EquivalentContainsNullIsFalse":     {group: "G1", errs: true, suite: `2.0.. invalid="" outputs=["false"]`},
	"EquivalentContainsNullIsTrue":      {group: "G1", errs: true, suite: `2.0.. invalid="" outputs=["true"]`},
	"EquivalentContainsNullLeftIsFalse": {group: "G1", errs: true, suite: `2.0.. invalid="" outputs=["false"]`},
	"EquivalentInIsFalse":               {group: "G1", errs: true, suite: `2.0.. invalid="" outputs=["false"]`},
	"EquivalentInIsTrue":                {group: "G1", errs: true, suite: `2.0.. invalid="" outputs=["true"]`},
	"EquivalentInNullRightIsFalse":      {group: "G1", errs: true, suite: `2.0.. invalid="" outputs=["false"]`},
	"FlattenNull":                       {group: "G3c", errs: false, suite: `1.0.. invalid="" outputs=["null"]`},
	"NullEquivalentInIsTrue":            {group: "G1", errs: true, suite: `2.0.. invalid="" outputs=["true"]`},
	"SkipNegative":                      {group: "G3d", errs: false, suite: `1.0.. invalid="" outputs=["{ }"]`},
	// CqlNullologicalOperatorsTest.xml
	"CoalesceMoreThanFiveArgs": {group: "G3e", errs: false, suite: `1.0.. invalid="semantic" outputs=[]`},
	// CqlStringOperatorsTest.xml
	"MatchesFullFalse": {group: "G1", errs: true, suite: `2.0.. invalid="" outputs=["false"]`},
	"MatchesFullNull":  {group: "G1", errs: true, suite: `2.0.. invalid="" outputs=["null"]`},
	"MatchesFullTrue":  {group: "G1", errs: true, suite: `2.0.. invalid="" outputs=["true"]`},
	// CqlTypeOperatorsTest.xml
	"StringMalformedToRatio": {group: "G3f", errs: true, suite: `1.5.. invalid="" outputs=["null"]`},
	"StringToRatio":          {group: "G3f", errs: true, suite: `1.5.. invalid="" outputs=["true"]`},
	"ToDate1":                {group: "G3f", errs: true, suite: `1.5.. invalid="" outputs=["@2014-01-01"]`},
	"ToDateFromDateTime":     {group: "G3f", errs: true, suite: `1.5.. invalid="" outputs=["@2014-01-01"]`},
	"ToDateIgnoresTime":      {group: "G3f", errs: true, suite: `1.5.. invalid="" outputs=["@2014-01-01"]`},
	"ToDateMalformed":        {group: "G3f", errs: true, suite: `1.5.. invalid="" outputs=["null"]`},
	"ToDateYear":             {group: "G3f", errs: true, suite: `1.5.. invalid="" outputs=["@2014"]`},
	"ToDateYearMonth":        {group: "G3f", errs: true, suite: `1.5.. invalid="" outputs=["@2014-01"]`},
}

// suiteContract writes what the suite asks of a test: the versions of CQL it
// applies to, whether it is invalid, and its outputs.
func suiteContract(tc TestCase, version string) string {
	outputs := make([]string, len(tc.Outputs))
	for i, o := range tc.Outputs {
		outputs[i] = strings.TrimSpace(o.Value)
		if o.Type != "" {
			outputs[i] = o.Type + ":" + outputs[i]
		}
	}
	return fmt.Sprintf("%s..%s invalid=%q outputs=%q", version, tc.VersionTo, tc.Expression.Invalid, outputs)
}

// targetVersion is the CQL this engine is written from (ADR 001).
const targetVersion = "1.5.3"

// covers reports whether the version range from..to of a test, to open when
// empty, holds v.
func covers(v, from, to string) bool {
	return compareVersions(from, v) <= 0 && (to == "" || compareVersions(v, to) <= 0)
}

// compareVersions orders two dotted versions by their numbers, a missing one
// read as 0: 1.5 is 1.5.0.
func compareVersions(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := range max(len(as), len(bs)) {
		var x, y int
		if i < len(as) {
			x, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			y, _ = strconv.Atoi(bs[i])
		}
		if x != y {
			return x - y
		}
	}
	return 0
}

// suiteCase is a test of the suite with the version it inherits.
type suiteCase struct {
	tc      TestCase
	version string
}

// readSuite reads every test of testdata/*.xml by name, with the version its
// group or file gives it when it gives none.
func readSuite(t *testing.T) map[string]suiteCase {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("testdata", "*.xml"))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]suiteCase{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var suite TestSuite
		if err := xml.Unmarshal(data, &suite); err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}
		for _, group := range suite.Groups {
			for _, tc := range group.Tests {
				version := effectiveVersion(tc.Version, group.Version, suite.Version)
				if _, dup := cases[tc.Name]; dup && tc.Name != "" {
					t.Errorf("%s: two tests of the suite are named %s", file, tc.Name)
				}
				cases[tc.Name] = suiteCase{tc, version}
			}
		}
	}
	return cases
}

// effectiveVersion is the version of a test: its own, or else its group's,
// or else its file's.
func effectiveVersion(test, group, file string) string {
	for _, v := range []string{test, group} {
		if v != "" {
			return v
		}
	}
	return file
}

// suitegapsDefine is a definition of testdata/snapshots/suitegaps.cql: the
// suite's expression, and the name of its test after it.
var suitegapsDefine = regexp.MustCompile(`(?m)^define D\d+: (.*) // (\w+)$`)

// TestEveryKnownFailureIsOfTheSuite holds knownFailures to the suite and to
// testdata/snapshots/suitegaps.cql: each entry names a test of the suite, in a
// group of failureGroups, asking what the entry says it asks; a test of G1
// is of a CQL after 1.5.3, and any other of a range that holds it; and the area
// has one definition for each, with the suite's expression.
func TestEveryKnownFailureIsOfTheSuite(t *testing.T) {
	t.Parallel()
	cases := readSuite(t)
	for name, kf := range knownFailures {
		c, ok := cases[name]
		if !ok {
			t.Errorf("knownFailures names %s, which the suite does not have", name)
			continue
		}
		if _, ok := failureGroups[kf.group]; !ok {
			t.Errorf("%s: group %q is not in failureGroups", name, kf.group)
		}
		if got := suiteContract(c.tc, c.version); got != kf.suite {
			t.Errorf("%s: the suite asks %s, and knownFailures records %s; classify it again", name, got, kf.suite)
		}
		switch after := compareVersions(c.version, targetVersion) > 0; {
		case kf.group == "G1" && !after:
			t.Errorf("%s: of CQL %s..%s, and in G1; G1 is for the tests of a CQL after %s",
				name, c.version, c.tc.VersionTo, targetVersion)
		case kf.group != "G1" && !covers(targetVersion, c.version, c.tc.VersionTo):
			t.Errorf("%s: of CQL %s..%s, which does not hold %s, and in group %s; classify it again",
				name, c.version, c.tc.VersionTo, targetVersion, kf.group)
		}
	}

	raw, err := os.ReadFile(filepath.Join("..", "testdata", "snapshots", "suitegaps.cql"))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, m := range suitegapsDefine.FindAllStringSubmatch(string(raw), -1) {
		expr, name := m[1], m[2]
		if seen[name] {
			t.Errorf("suitegaps.cql holds %s twice", name)
		}
		seen[name] = true
		if _, ok := knownFailures[name]; !ok {
			t.Errorf("suitegaps.cql holds %s, which is not in knownFailures", name)
			continue
		}
		if want := strings.TrimSpace(cases[name].tc.Expression.Value); expr != want {
			t.Errorf("suitegaps.cql: %s is %s, and the suite has %s", name, expr, want)
		}
	}
	for name := range knownFailures {
		if !seen[name] {
			t.Errorf("%s is in knownFailures and has no definition in suitegaps.cql", name)
		}
	}
}

// TestTheVersionsOfATestAreARange holds covers to tests of the suite whose
// versions say whether they apply to CQL 1.5.3: PredecessorOf1D is of 1.0 to
// 1.5.3, and its sibling PredecessorOf1DPrecision of 2.0, where the step of a
// Decimal became its precision; DateTimeComponentFromTimezoneOffset, of
// `timezone from`, ends at 1.3, and its group begins at 1.4: a range that
// holds no version (backlog 1c25).
func TestTheVersionsOfATestAreARange(t *testing.T) {
	t.Parallel()
	cases := readSuite(t)
	for name, applies := range map[string]bool{
		"PredecessorOf1D":                     true,
		"PredecessorOf1DPrecision":            false,
		"DateTimeComponentFromTimezoneOffset": false,
		"SkipNegative":                        true,
	} {
		c, ok := cases[name]
		if !ok {
			t.Errorf("the suite has no %s", name)
			continue
		}
		if got := covers(targetVersion, c.version, c.tc.VersionTo); got != applies {
			t.Errorf("%s, of CQL %s..%s: applies to %s %v; want %v", name, c.version, c.tc.VersionTo, targetVersion, got, applies)
		}
	}
}

// TestTheContractHoldsWhatTheSuiteAsks holds suiteContract to each field it
// writes: no test in knownFailures has a versionTo or a typed output today, so
// a contract that dropped either would not be seen until one had.
func TestTheContractHoldsWhatTheSuiteAsks(t *testing.T) {
	t.Parallel()
	base := TestCase{
		VersionTo:  "1.5.3",
		Expression: Expression{Invalid: "semantic"},
		Outputs:    []Output{{Type: "integer", Value: " 2 "}},
	}
	if got, want := suiteContract(base, "1.0"), `1.0..1.5.3 invalid="semantic" outputs=["integer:2"]`; got != want {
		t.Errorf("suiteContract = %s; want %s", got, want)
	}
	for field, changed := range map[string]TestCase{
		"versionTo": {VersionTo: "1.4", Expression: base.Expression, Outputs: base.Outputs},
		"invalid":   {VersionTo: base.VersionTo, Expression: Expression{Invalid: "true"}, Outputs: base.Outputs},
		"type":      {VersionTo: base.VersionTo, Expression: base.Expression, Outputs: []Output{{Type: "decimal", Value: "2"}}},
		"output":    {VersionTo: base.VersionTo, Expression: base.Expression, Outputs: []Output{{Type: "integer", Value: "3"}}},
	} {
		if suiteContract(changed, "1.0") == suiteContract(base, "1.0") {
			t.Errorf("suiteContract does not see a change of %s", field)
		}
	}
}

// TestVersionsAreComparedByTheirNumbers holds compareVersions and covers to
// what a dotted version means: 1.10 is after 1.9, 1.5 is 1.5.0, and a range
// holds its two ends.
func TestVersionsAreComparedByTheirNumbers(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		a, b string
		sign int
	}{
		{"1.10", "1.9", 1},
		{"1.5", "1.5.0", 0},
		{"1.5", "1.5.3", -1},
		{"1.5.3", "1.5", 1},
		{"1.5.3", "1.5.10", -1},
		{"2.0", "1.5.3", 1},
		{"1.4", "1.4", 0},
	} {
		if got := compareVersions(tt.a, tt.b); (got > 0) != (tt.sign > 0) || (got < 0) != (tt.sign < 0) {
			t.Errorf("compareVersions(%s, %s) = %d; want the sign of %d", tt.a, tt.b, got, tt.sign)
		}
	}
	for _, tt := range []struct {
		v, from, to string
		want        bool
	}{
		{"1.5.3", "1.5.3", "1.5.3", true},
		{"1.5.3", "1.0", "1.5.2", false},
		{"1.5.3", "1.0", "1.5", false},
		{"1.5.3", "1.5.4", "", false},
		{"1.5.3", "1.0", "", true},
	} {
		if got := covers(tt.v, tt.from, tt.to); got != tt.want {
			t.Errorf("covers(%s, %s, %s) = %v; want %v", tt.v, tt.from, tt.to, got, tt.want)
		}
	}
}

// TestATestInheritsItsVersion holds effectiveVersion to its order, and
// readSuite to using it: LowBoundaryNull declares no version, in a group of
// 1.4 in a file of 1.0. No test of the suite inherits its file's.
func TestATestInheritsItsVersion(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ test, group, file, want string }{
		{"2.0", "1.4", "1.0", "2.0"},
		{"", "1.4", "1.0", "1.4"},
		{"", "", "1.0", "1.0"},
	} {
		if got := effectiveVersion(tt.test, tt.group, tt.file); got != tt.want {
			t.Errorf("effectiveVersion(%q, %q, %q) = %q; want %q", tt.test, tt.group, tt.file, got, tt.want)
		}
	}
	if got := readSuite(t)["LowBoundaryNull"].version; got != "1.4" {
		t.Errorf("LowBoundaryNull has version %q; want its group's, 1.4", got)
	}
}
