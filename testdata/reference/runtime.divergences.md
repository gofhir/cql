# Where this engine and the reference runtimes disagree

`runtime.cql` was run by two reference runtimes at the same fixed evaluation
time:

- `org.cqframework:engine-jvm 5.4.0`, through `Evaluate.java`, into
  `runtime.expected.json`.
- `cql-execution 3.3.2`, through `js/evaluate.js`, into
  `runtime.expected.js.json`.

This engine was run on `main`, at fhirpath v1.14.1. Values below are what each
one returned, not what any of them should return.

`engine_runtime_reference_test.go` pins each row with this engine's answer, in
`knownRuntimeDivergences`, `disputedRuntimeAnswers` and `decidedRuntimeAnswers`,
and records each reference's exclusions in `runtimeReferences[].notOracle`. This file and those
lists have to change together.

Both runtimes run the ELM the reference translator produces. Where they agree
with each other they are two opinions on evaluation, and one on translation.

## The references and the specification agree; this engine does not

None at present. The last, `DecimalDivision`, was a quotient left at sixteen
places: `10.0 / 3.0 = 3.33333333` was false here and true in the Java engine.
A Decimal result is now brought to the scale of 8 the specification gives the
type, toward negative infinity, as the Java engine does (`funcs.ScaleDecimal`).
cql-execution computes Decimal in binary floating point and is not an oracle for
those rows.

## Neither gives the exact value: a logarithm in floating point

```cql
define LogOfAScaledString: Log(10, ToDecimal('0.999999999999999999999'))
// Java: -230258506.99111847   cql-execution: null   this engine: -230258506.99111846
```

The exact value is -230258508.14811202. ToDecimal brings the string to the
scale of 8, 0.99999999, and both engines take the logarithm of each argument
as a float, where ln 0.99999999 has already lost the digits that decide the
quotient. This engine divides the two floats as floats; the Java engine 5.4.0
divides them as decimals, and 3.26.0 answered as this engine does. It is in
`knownRuntimeDivergences`, and an exact Log is this engine's to do.

## Settled: quantities with different units

`QuantityEqual` (`1 'm' = 100 'cm'`), `QuantityEquivalent`, `DistinctQuantity`
and `AggDistinctQuantity` compare quantities with comparable units, which the
specification converts. The Java engine 3.26.0 did not convert, and was not an
oracle for them; 5.4.0 converts, and the three runtimes agree.
`AggDistinctQuantity` is why one reference was not enough: with 3.26.0 alone,
both engines answered 2 and the row looked settled, though both were wrong,
for different reasons. Agreement between two engines does not make the answer
right.

## Where the Java engine fails: a let read inside aggregate

```cql
define AggregateLetWithout: ({1, 2, 3}) X let L: X * 10
  without ({2}) Y such that X = Y aggregate A starting 0: A + L
// Java: CqlException   cql-execution: 40   this engine: 40
```

The Java engine fails whenever the aggregate expression reads a let. In
3.26.0, `State.resolveVariable` returned null for it, and the evaluation
ended in a NullPointerException; 5.4.0 fails too, with a CqlException whose
message is `null`. Two rows show it, and it is excluded from both, which
cql-execution settles:

- `AggregateLetWithout`, a plain aggregate after a without (40).
- `AggregateDistinctLetWith`, an aggregate distinct after a with (30).

If a later version fixes one shape and not the other, each row says which.

## Decided: a DateTime below hour precision has no offset here

```cql
define DateTimeYear: DateTime(2020)
// Java: 2020 at -03:00   cql-execution: 2020 at -03:00   this engine: no offset
```

Both references assume the evaluation offset. The specification words it that
way, with no exception for precision. This engine keeps the rule fhirpath
applies (`WithDefaultOffset`): "An offset only means something once the value
reaches a time of day."

That is a decision, not an oversight. The engine's comparisons depend on it.
Without the rule, a day-precision value was normalized as if it were an
instant, and it answered wrong at UTC+14. `decidedRuntimeAnswers` pins it,
and changing it means arguing with that reason.

## Decided: where the Decimal scale is not applied

```cql
define DecimalLnUnscaled: Ln(2.0)                                   // Java: 0.69314718
define DecimalGeometricMeanUnscaled: GeometricMean({2.0, 8.0, 3.0}) // Java: 3.63424113
define QuantityDivisionUnscaled: 10.0 'mg' / 3.0                    // Java: 3.33333333 'mg'
define DecimalPowerShortestFloat: 0.09 ^ 0.5                         // Java: 0.29999999
define DecimalLogNearInteger: Log(1000.0, 10.0)                      // Java: 2.99999999
define DecimalWholePowerExact: 0.21 ^ 2                              // Java: 0.04409999
define ConvertsToDecimalOutOfRange: ConvertsToDecimal('100000000000000000000') // Java: true
define IntegerNegateMinimum: -(-2147483648)                          // Java: -2147483648
define IntegerAbsMinimum: Abs(-2147483648)                           // Java: -2147483648
define IntegerSumOverflow: Sum({2147483647, 1})                      // Java: -2147483648
define IntegerProductOverflow: Product({65536, 65536})               // Java: 0
define IntegerTruncatePastTheType: Truncate(3000000000.5)            // Java: -1294967296
define LongAddOverflow: 9223372036854775807L + 1L                    // Java: -9223372036854775808
```

The Java engine floors these to 8 places as well, and this engine does not:

- `Ln`: the conformance suite asks `Round(Ln(1000), 8)` for 6.90775528. ln 1000
  is 6.9077552789…, which floors to 6.90775527 before the rounding sees it.
- `GeometricMean`: the Java engine computes `Power(Product(X), 1.0 / Count(X))`
  with the exponent floored, so the geometric mean of `{1, 2, 4}` is 1.99999998
  there. This engine keeps the exact root, and the numeric and Quantity
  aggregates agree on it.
- A Quantity's value: the scale is the Decimal type's. A dose in a small unit
  keeps its digits rather than becoming 0 in that unit.
- A float result (a power, a logarithm, a root): this engine floors the shortest
  decimal that prints as the float, and the Java engine floors the float's exact
  binary value. The float written 0.3 is 0.29999999999999998…, so `0.09 ^ 0.5` is
  0.29999999 there. The conformance suite asks `Power(10, -7)` for 0.0000001, and
  the float 1e-7 is 0.0000000999…, which the exact reading floors to 0.00000009.
- A result past its type, 32 bits for an Integer and 64 for a Long: null, as
  the specification has it ("if the result of the operation cannot be
  represented, the result is null"). The Java engine wraps it to the low bits.
  Where cql-execution also answers null — `+`, `*`, `div` and `^` of an
  Integer, `*` of a Long — the rows are disputed rather than decided. The
  32-bit edge is read from the static type, since the engine holds an Integer
  and a Long alike, and only where every operand is an Integer for certain:
  sema types a value it cannot follow — from an included library, an alias —
  plus an Integer as an Integer, and a Long can stand behind it. There, and
  without a plan, as for imported ELM, only the 64-bit edge is caught.
- `ConvertsToDecimal` of a string past the largest Decimal: it is whether
  ToDecimal succeeds, and ToDecimal of it is null in both engines. The Java
  engine answers true all the same.
- A power with a whole-number exponent: it is computed in decimal, as the
  repeated multiplication it is, and then floored. Through a float, `0.21 ^ 2`
  is 0.044099999999999993, which floors to 0.04409999 in the Java engine, where
  `0.21 * 0.21` is 0.0441. A fractional exponent still goes through the float.
- An Integer to a negative Integer power is the Decimal it is: `2 ^ -1` is 0.5,
  where this engine answered 0. The semantic phase types it as a Decimal when
  the exponent is a negative literal, `2 ^ -1` or `Power(2, -1)`. An exponent
  whose sign only evaluation knows, `2 ^ (0 - 1)`, stays typed as the reference
  translator types it, an Integer, and what consumes it as one — `expand`,
  `ToInteger`, an index — sees the Decimal it evaluates to.
- `Log` of an exact power: log10 1000 computed as ln 1000 / ln 10 is the float
  2.9999999999999996, which floors to 2.99999999 in the Java engine. The
  specification asks for the logarithm, which is 3, as `Log(1000, 100)` is 1.5
  where the Java engine answers 1.49999999. Where the float is within 1e-9 of a
  fraction k/q with q up to 12, and the value to the q is exactly the base to
  the k — checked in decimal — the answer is k/q; anything else floors as the
  Java engine floors it, so `Log(999.999999999999, 10)` is 2.99999999 in both.

## Decided: a Date among DateTimes stays a Date

```cql
define ListDateDateTime: First({@2020-01-01, @2020-01-01T10:00:00}) is DateTime
// Java: true   cql-execution: true   this engine: false
```

The parts of a list, an if or a case reach the type they have in common:
`{1, 1.0}` is two Decimals and `{1 'mg', 2}` two Quantities, as the reference
translator makes them. The other implicit conversion it inserts, Date to
DateTime, this engine does not apply. A Date compared at the day is read as
written, and a DateTime at an offset is moved to it (see `readAsWritten`), so
promoting one would change what every comparison it reaches answers. That is a
larger question than the list's type, and it is left for its own change.

## Disputed, and settled by the specification: a null operand of union or except

```cql
define UnionNullFirst: null union {1, 1, 2}     // Java: {1, 2}   cql-execution: {1, 1, 2}   this engine: {1, 2}
define UnionNullSecond: {1, 1, 2} union null    // the same
define ExceptNullSecond: {1, 1, 2} except null  // the same
```

The specification settles it. Union says "if either argument is null, it is
considered an empty list for the purposes of evaluating the union", and its
result is "all unique elements from both arguments". Except says that with a
null second argument "the operation is performed as though the second
argument was an empty list", and that "each element will appear in the result
at most once". An empty list does not exempt the other operand from set
semantics, so the duplicates go. The Java engine answers so. cql-execution
returns the other operand as it was, and so did this engine until union and
except were rebuilt on distinct.

## Disputed, and settled by the conformance suite: a single source with no return clause

```cql
define NoReturn: ({1, 1, 2}) X
// Java: {1, 1, 2}   cql-execution: {1, 2}   this engine: {1, 1, 2}
define NoReturnSortedConformance: ({4, 5, 1, 6, 2, 1}) sL sort asc
// Java: {1, 1, 2, 4, 5, 6}   cql-execution: {1, 2, 4, 5, 6}   this engine: {1, 1, 2, 4, 5, 6}
define NoReturnSortedDescConformance: ({4, 5, 1, 6, 2, 1}) sL sort desc
// Java: {6, 5, 4, 2, 1, 1}   cql-execution: {6, 5, 4, 2, 1}   this engine: {6, 5, 4, 2, 1, 1}
```

The references disagree. The conformance suite keeps the duplicates of a
single source with no return clause:

```
simpleSortAsc:  ({4, 5, 1, 6, 2, 1}) sL sort asc   =>  {1, 1, 2, 4, 5, 6}
simpleSortDesc: ({4, 5, 1, 6, 2, 1}) sL sort desc  =>  {6, 5, 4, 2, 1, 1}
```

(CqlListOperatorsTest, which this engine runs and passes.) The two sorted rows
are those cases themselves, so the harness checks them directly, and
cql-execution fails them. `NoReturn` has no sort, and no conformance case
does without one. It rests on them by analogy: a sort orders the elements a
query keeps, and does not decide which it keeps.

The specification text points the other way, and the tension stays recorded
here. The Developer's Guide says: "the default for return clauses is
distinct, as opposed to all, so if no return clause is specified, duplicates
will be eliminated from the result". The sentence reads the same in the 1.5
ballot ([May 2020](https://cql.hl7.org/2020May/03-developersguide.html)) and
in v2.0.0 ([current](https://cql.hl7.org/03-developersguide.html)), both
checked on 14 October 2026. It sits in the section on multi-source queries, and there the reference
translator does supply a return clause, a distinct Tuple of the aliases. For
a single source the translator supplies none, and the Java engine and the
conformance suite keep the duplicates. This engine does the same.

The rows stay in `disputedRuntimeAnswers`. That pins this engine's answer,
records how the dispute was settled, and names the Java engine as the
reference that agrees. The test fails the day cql-execution agrees, and the
rows then leave the list. If the Java engine changes instead, it fails with a
different message, since that reopens the dispute rather than settling it.

## Settled: the scope of a definition

A definition evaluates in the scope of its library, as the Java engine has
it. cql-execution resolves the names in it in the scope it is referenced from,
so a query alias, a let or a function operand of the same name stands in for
them: `({10}) Y return D`, with `define Y: 1` and `define D: Y + 1`, is {11}
there and {2} here (`DefinitionUnderAnAlias`).

The specification decides it. CQL 1.5.3, Logical Specification, ExpressionRef:
"The result of evaluating an ExpressionReference is the result of evaluating
the referenced NamedExpression". An alias or a let is referenced "within the
scope of a query" (AliasRef, QueryLetRef). The reference translator resolves the
names that way, and cql-execution runs that same ELM. The `scope` area holds
the same rule to 600 definitions with an oracle of its own
(`engine_scope_oracle_test.go`, plan 003, PR 5).

The rows — `DefinitionUnderAnAlias`, `DefinitionUnderALet`,
`DefinitionUnderAWhere` and `DefinitionUnderAnOperand` — stay in
`disputedRuntimeAnswers`, with the Java engine as the reference that agrees.
The test fails the day cql-execution agrees too, and the rows then leave the
list.

## Settled: Any, and the element type of a list or an interval

Every value is of type Any, so `x as Any` and `cast x as Any` are `x`, and
`x is Any` is true; a list is a `List<T>`, and an interval an `Interval<T>`,
when what it holds is of `T`, so `{ 1, 2 } is List<String>` is false. The
Java engine 3.26.0 failed on Any (`Could not resolve type
org.opencds.cqf.cql.engine.runtime.Any`) and answered true for any list or
interval; 5.4.0 answers as this engine and cql-execution do. `AsAny`, `IsAny`,
`CastAsAny`, `AsAnyInArithmetic`, `ListOfIntegersIsNotListOfString` and
`IntervalOfIntegersIsNotIntervalOfString` left `disputedRuntimeAnswers`.
