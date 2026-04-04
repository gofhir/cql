# CQL Conformance Test Suite Design

## Goal

Validate the go-cql engine against the official CQL conformance tests from
[cqframework/cql-tests](https://github.com/cqframework/cql-tests). Run all 16
XML test suites as native Go tests, report PASS/FAIL per test case, and use
failures as the roadmap to implement missing features until we reach full
conformance.

## Test Source

The `cqframework/cql-tests` repository contains 16 XML files under `tests/cql/`:

| File | Category |
|------|----------|
| ValueLiteralsAndSelectors.xml | Null, Boolean, Integer, Decimal, Long, String, Date, DateTime, Time, Quantity literals |
| CqlTypesTest.xml | Any, Boolean, DateTime, Integer, Decimal, String, Quantity, Time, Interval, List, Tuple |
| CqlArithmeticFunctionsTest.xml | Abs, Add, Ceiling, Divide, Exp, Floor, Log, Ln, Modulo, Multiply, Negate, Power, Predecessor, Round, Subtract, Successor, Truncate, TruncatedDivide |
| CqlComparisonOperatorsTest.xml | Between, Equal, Equivalent, Greater, GreaterOrEqual, Less, LessOrEqual, NotEqual, NotEquivalent |
| CqlConditionalOperatorsTest.xml | if-then-else, standard case, selected case |
| CqlDateTimeOperatorsTest.xml | Add, Subtract, DurationBetween, DifferenceBetween, DateTimeComponentFrom, Now, SameAs, SameOrBefore, SameOrAfter |
| CqlIntervalOperatorsTest.xml | Contains, Except, In, Includes, Intersect, Meets, Overlaps, ProperContains, Union, Width |
| CqlListOperatorsTest.xml | Sort, Contains, Distinct, Equal, Except, Exists, Flatten, First, Last, IndexOf, In, Length, Intersect, Union |
| CqlLogicalOperatorsTest.xml | And, Implies, Not, Or, Xor |
| CqlNullologicalOperatorsTest.xml | Coalesce, IsNull, IsFalse, IsTrue |
| CqlStringOperatorsTest.xml | Combine, Concatenate, EndsWith, Indexer, Length, Lower, Matches, Upper, StartsWith, Substring |
| CqlTypeOperatorsTest.xml | As, Convert, Is, ToBoolean, ToConcept, ToDateTime, ToDecimal, ToInteger, ToString, ToTime, ToQuantity |
| CqlAggregateFunctionsTest.xml | Count, Sum, Min, Max, Avg, Median, Mode |
| CqlAggregateTest.xml | Aggregate operator |
| CqlQueryTests.xml | Query expressions |
| CqlErrorsAndMessagingOperatorsTest.xml | Message function, error handling |

## XML Schema

Tests follow the hierarchy: `Tests > Group > Test > (Expression + Output)`.

```xml
<tests name="CqlArithmeticFunctionsTest" xmlns="http://hl7.org/fhirpath/tests">
  <group name="Add">
    <test name="Add11">
      <expression>1 + 1</expression>
      <output>2</output>
    </test>
    <test name="AddNull">
      <expression>1 + null</expression>
      <output>null</output>
    </test>
    <test name="DateTimeError">
      <expression invalid="true">DateTime(10000)</expression>
    </test>
  </group>
</tests>
```

Key attributes:
- `expression.invalid`: `"false"` (default), `"syntax"`, `"semantic"`, `"execution"`, `"true"` (legacy = execution error)
- `output.type`: optional hint — `boolean`, `integer`, `decimal`, `string`, `date`, `dateTime`, `time`, `Quantity`, `code`
- `test.ordered`: whether list ordering matters for comparison

## Architecture

### Directory Structure

```
conformance/
  testdata/                    <- 16 XML files vendored from cqframework/cql-tests
    CqlArithmeticFunctionsTest.xml
    ...
  conformance_test.go          <- test runner: parses XML, generates subtests
  xmlmodel.go                  <- Go structs for XML deserialization
  output_parser.go             <- parses expected output strings to fptypes.Value
```

### XML Model (Go structs)

```go
type TestSuite struct {
    XMLName xml.Name `xml:"tests"`
    Name    string   `xml:"name,attr"`
    Groups  []Group  `xml:"group"`
}

type Group struct {
    Name  string `xml:"name,attr"`
    Tests []Test `xml:"test"`
}

type Test struct {
    Name       string     `xml:"name,attr"`
    Expression Expression `xml:"expression"`
    Outputs    []Output   `xml:"output"`
    Ordered    string     `xml:"ordered,attr"`
}

type Expression struct {
    Invalid string `xml:"invalid,attr"`
    Value   string `xml:",chardata"`
}

type Output struct {
    Type  string `xml:"type,attr"`
    Value string `xml:",chardata"`
}
```

### Test Runner Flow

For each XML file, for each group, for each test case:

1. **Wrap** the CQL expression in a library: `library Test version '1.0' define "result": <expression>`
2. **Evaluate** with `engine.EvaluateExpression(ctx, wrappedCQL, "result", nil, nil)`
3. **Assert** based on the test case:
   - If `expression.invalid` is set: expect an error from the engine (syntax, semantic, or execution)
   - If output is `null`: expect nil result
   - If output has value: parse the expected string to `fptypes.Value` and compare with the actual result

Subtests are hierarchical for easy filtering:
```
TestConformance/Arithmetic/Add/Add11
TestConformance/Arithmetic/Add/AddNull
TestConformance/Comparison/Equal/EqualTrue
```

### Output Parser

Converts XML output strings to `fptypes.Value` for comparison:

| Output format | Go type |
|---|---|
| `null` | nil |
| `true` / `false` | fptypes.Boolean |
| `42` (no dot) | fptypes.Integer |
| `5.0` (with dot) | fptypes.Decimal |
| `'abc'` (single-quoted) | fptypes.String |
| `@2014-01-01T` | fptypes.DateTime |
| `@2014-01-01` | fptypes.Date |
| `@T09:00:00` | fptypes.Time |
| `5.0'g'` | fptypes.Quantity |
| `{1, 2, 3}` | fptypes.List |
| `Interval[2, 7]` | fptypes.Interval |
| `Tuple { ... }` | fptypes.Tuple |

### Running Tests

```bash
go test ./conformance/... -v                              # all conformance tests
go test ./conformance/... -run TestConformance/Arithmetic  # one category
go test ./conformance/... -run TestConformance/Arithmetic/Add/Add11  # one test
go test ./conformance/... -json                            # JSON output for reporting
```

## Vendoring Strategy

The 16 XML files are copied directly into `conformance/testdata/`. No git
submodule — they are static files updated manually when a new spec version is
released.

## Baseline Results (2026-04-04)

**Overall: 791/1731 (45.7%)**

| Suite | Pass | Total | Rate |
|-------|------|-------|------|
| CqlLogicalOperatorsTest | 39 | 39 | 100.0% |
| CqlComparisonOperatorsTest | 204 | 223 | 91.5% |
| CqlConditionalOperatorsTest | 7 | 9 | 77.8% |
| CqlTypesTest | 19 | 28 | 67.9% |
| CqlListOperatorsTest | 127 | 212 | 59.9% |
| CqlNullologicalOperatorsTest | 11 | 22 | 50.0% |
| CqlIntervalOperatorsTest | 192 | 412 | 46.6% |
| ValueLiteralsAndSelectors | 27 | 66 | 40.9% |
| CqlArithmeticFunctionsTest | 77 | 212 | 36.3% |
| CqlQueryTest | 4 | 12 | 33.3% |
| CqlStringOperatorsTest | 22 | 81 | 27.2% |
| CqlErrorsAndMessagingOperatorsTest | 1 | 4 | 25.0% |
| CqlAggregateFunctionsTest | 12 | 50 | 24.0% |
| CqlDateTimeOperatorsTest | 45 | 317 | 14.2% |
| CqlTypeOperatorsTest | 4 | 35 | 11.4% |
| CqlAggregateTest | 0 | 9 | 0.0% |

## Conformance Roadmap

Priority order by impact (most failing tests first):

1. **CqlDateTimeOperatorsTest** — 272 failures (biggest gap)
2. **CqlIntervalOperatorsTest** — 220 failures
3. **CqlArithmeticFunctionsTest** — 135 failures
4. **CqlListOperatorsTest** — 85 failures
5. **CqlStringOperatorsTest** — 59 failures
6. **CqlAggregateFunctionsTest** — 38 failures
7. **ValueLiteralsAndSelectors** — 39 failures
8. **CqlTypeOperatorsTest** — 31 failures
9. **CqlComparisonOperatorsTest** — 19 failures
10. **CqlNullologicalOperatorsTest** — 11 failures
11. **CqlAggregateTest** — 9 failures
12. **CqlQueryTest** — 8 failures
13. **CqlTypesTest** — 9 failures
14. **CqlErrorsAndMessagingOperatorsTest** — 3 failures
15. **CqlConditionalOperatorsTest** — 2 failures
