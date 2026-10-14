# Where this engine and the reference engine disagree at runtime

`runtime.cql` run by `info.cqframework:engine 3.26.0` through
`internal/referenceharness/src/Evaluate.java`, at the fixed evaluation time
recorded in `runtime.expected.json`, and by this engine at the commit the
branch started from (`2479015`). Values below are what each one returned, not
what either should return. `knownRuntimeDivergences` in
`engine_runtime_reference_test.go` pins each row with this engine's answer, so
this file and that list have to change together.

## The reference engine and the specification agree; this engine does not

| Definition | This engine | Reference | Specification |
| --- | --- | --- | --- |
| `WithAggregate` | 6 | 2 | `with` filters the rows the aggregate reduces |
| `WithoutAggregate` | 6 | 4 | same, for `without` |
| `ReturnDefault` | `{1, 1, 2}` | `{1, 2}` | Author's Guide 3.2: "By default, queries return a distinct list of results" |
| `UnionDuplicates` | `{1, 1, 2}` | `{1, 2}` | Union: "returns a list with all unique elements from both arguments" |
| `ExceptDuplicates` | `{1, 1, 2}` | `{1, 2}` | Except: "each element will appear in the result at most once" |
| `AggDistinctNilNull` | 1 | 2 | `'nil'` is not null |
| `AggDistinctPairs` | 3 | 4 | four distinct pairs |
| `AggDistinctIntDec` | 2 | 1 | `1 = 1.0`; this engine's own `distinct {1, 1.0}` answers `{1}` |
| `AggDistinctOffsets` | 2 | 1 | the same instant; this engine's own `distinct` agrees |
| `ListIntDec` | `{1, 1.0}`, Integer and Decimal | two Decimals | the translator converts the list to `List<Decimal>` |
| `ExpandWeekPerMinute` | 10001 | 10080 | the expansion has no length limit |
| `ExpandThirtyYearsPerDay` | 10001 | 10958 | same |
| `ExpandIntegers` | 10001 | 100000 | same |
| `SameDayAcrossOffsets` | false | true | Same As: "When comparing DateTime values with different timezone offsets, implementations should normalize to the timezone offset of the evaluation request timestamp, but only when the comparison precision is hours, minutes, seconds, or milliseconds" |

`SameDayAcrossOffsets` was already wrong before fhirpath v1.14. This engine
normalizes both sides to UTC, which moves `23:00Z` into the next day. At day
precision, the specification compares the days as written.

The three `Expand*` rows are a guard in `funcs/interval_set.go` that stops at
10001 elements and returns what it has, without an error.

The four `AggDistinct*` rows share one cause: `distinctCombos` keys rows by
their `String()`, which merges values that differ and splits values that are
equal. `aggregate` ignoring `with`/`without` is a separate cause: the aggregate
loop iterates the unfiltered product and reapplies only `where`.

## The reference cannot settle these

`pinnedRuntimeAnswers` checks only this engine's answer for these definitions.
In this group, agreeing with the reference can be a regression rather than a
fix.

### Quantities with different units

```cql
define QuantityEqual: 1 'm' = 100 'cm'        // reference: null   this engine: true
define QuantityEquivalent: 1 'm' ~ 100 'cm'   // reference: false  this engine: true
define DistinctQuantity: distinct {1 'm', 100 'cm'}
                       // reference: both     this engine: {1 'm'}
define AggDistinctQuantity: ({1 'm', 100 'cm'}) X aggregate distinct A starting 0: A + 1
                       // reference: 2        this engine: 2
```

The specification compares quantities with comparable units after converting
them. With the configuration `Evaluate.java` uses, the reference engine does
not convert, so on unit conversion it is **not an oracle**. Agreeing with it
would be agreeing with a configuration.

`AggDistinctQuantity` is the warning case. Both engines answer 2, so the row
looks settled, but both are probably wrong, for different reasons: the
reference does not convert units, and this engine compares text. Through
`distinct`'s equality, the specification says 1. Agreement between two engines
does not make the answer right. A second runtime that converts units
(`cql-execution` with `ucum-lhc`) is what has to settle these rows.

### A DateTime below hour precision has no offset here

```cql
define DateTimeYear: DateTime(2020)
// reference: 2020 at -03:00, the evaluation offset   this engine: no offset
```

This comes from a deliberate decision in `fhirpath` (`WithDefaultOffset`,
v1.9.1): "An offset only means something once the value reaches a time of day."
The specification says that when no offset is given, the evaluation request's
offset is assumed. It makes no exception for precision, and the reference
follows it.

So this is a decision that departs from the wording of the specification, not
an oversight. It showed up only when the evaluation time moved off UTC. Whether
to keep it is a question for `fhirpath`, not for this harness.
