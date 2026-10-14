# Where this engine and the reference runtimes disagree

`runtime.cql` was run by two reference runtimes at the same fixed evaluation
time:

- `info.cqframework:engine 3.26.0`, through `Evaluate.java`, into
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

| Definition | This engine | Java | cql-execution | Specification |
| --- | --- | --- | --- | --- |
| `ListIntDec` | `{1, 1.0}`, Integer and Decimal | two Decimals | (not an oracle: the type is the translator's) | the translator converts the list to `List<Decimal>` |
| `DecimalDivision` | 3.3333333333333333 | 3.33333333 | (not an oracle: 3.3333333333333335, binary floating point) | Decimal: "a precision ... of 28 and a scale (meaning number of possible digits to the right of the decimal) of 8" |
| `SameDayAcrossOffsets` | false | true | true | Same As: "When comparing DateTime values with different timezone offsets, implementations should normalize to the timezone offset of the evaluation request timestamp, but only when the comparison precision is hours, minutes, seconds, or milliseconds" |

`DecimalDivision` is a quotient left at sixteen places. The conformance corpus
does not catch it, because its division cases are wrapped in `Round(…, 8)`. It
matters beyond printing: `10.0 / 3.0 = 3.33333333` is true in the Java engine
and false here. It is false in cql-execution too, which divides in binary
floating point. That runtime is not an oracle for decimal arithmetic, and its
row is the Java engine and the specification alone.

`SameDayAcrossOffsets` was already wrong before fhirpath v1.14. This engine
normalizes both sides to UTC, which moves `23:00Z` into the next day. At day
precision, the specification compares the days as written.

## Where the Java engine is not an oracle: quantities with different units

```cql
define QuantityEqual: 1 'm' = 100 'cm'        // Java: null   cql-execution: true   this engine: true
define QuantityEquivalent: 1 'm' ~ 100 'cm'   // Java: false  cql-execution: true   this engine: true
define DistinctQuantity: distinct {1 'm', 100 'cm'}
                       // Java: both          cql-execution: {1 'm'}            this engine: {1 'm'}
define AggDistinctQuantity: ({1 'm', 100 'cm'}) X aggregate distinct A starting 0: A + 1
                       // Java: 2             cql-execution: 1                  this engine: 1
```

The specification compares quantities with comparable units after converting
them. Configured as `Evaluate.java` configures it, the Java engine does not
convert. `cql-execution` converts through `ucum-lhc`, and it settles these
rows: this engine was right on the first three, and was wrong on the fourth
until aggregate distinct compared rows by equality instead of by their text.

`AggDistinctQuantity` is why one reference was not enough. With Java alone,
both engines answered 2 and the row looked settled, though both were wrong,
for different reasons. Agreement between two engines does not make the answer
right.

## Where the Java engine fails: a let read inside aggregate

```cql
define AggregateLetWithout: ({1, 2, 3}) X let L: X * 10
  without ({2}) Y such that X = Y aggregate A starting 0: A + L
// Java: CqlException   cql-execution: 40   this engine: 40
```

The Java engine fails whenever the aggregate expression reads a let:
`State.resolveVariable` returns null for it, and the evaluation ends in a
NullPointerException. Two rows show it, and it is excluded from both, which
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
