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
`knownRuntimeDivergences`, `disputedRuntimeAnswers` and `decidedRuntimeAnswers`. This file and those
lists have to change together.

Both runtimes run the ELM the reference translator produces. Where they agree
with each other they are two opinions on evaluation, and one on translation.

## The references and the specification agree; this engine does not

| Definition | This engine | Java | cql-execution | Specification |
| --- | --- | --- | --- | --- |
| `UnionDuplicates` | `{1, 1, 2}` | `{1, 2}` | `{1, 2}` | Union: "returns a list with all unique elements from both arguments" |
| `ExceptDuplicates` | `{1, 1, 2}` | `{1, 2}` | `{1, 2}` | Except: "each element will appear in the result at most once" |
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

## Disputed: a single source with no return clause

```cql
define NoReturn: ({1, 1, 2}) X
// Java: {1, 1, 2}   cql-execution: {1, 2}   this engine: {1, 1, 2}
```

The references disagree. The specification says "By default, queries return a
distinct list of results" in the section on the return clause, and does not
say what a query without one returns.

The reference translator settles it in its ELM. For a single source with no
return clause it emits no return clause, so the Java engine keeps the
duplicates. For several sources with no return clause it emits one, a Tuple of
the aliases with distinct set. This engine follows the translator in both
cases. The disagreement belongs to cql-execution, which removes duplicates
either way. `disputedRuntimeAnswers` pins this engine's answer and fails once
the references agree.
