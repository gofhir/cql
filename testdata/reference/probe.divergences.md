# Where this engine and the reference translator disagree

What the probe cannot express, and what it decided. Everything here was found by
translating `probe.cql` with `info.cqframework:cql-to-elm 3.26.0` and comparing,
not by reading the specification and reasoning about it.

## The engine accepts a choice element by its concrete name; the translator does not

```cql
define ValueQuantity: Obs.valueQuantity
```

```
ERROR: CqlSemanticException: Member valueQuantity not found for type Observation.
```

The model declares `Observation.value[x]` as `value`, a choice. The translator
takes that literally: the only way to reach a branch is `Obs.value as
FHIR.Quantity`. This engine accepts both, because the evaluator navigates FHIR
JSON, where the field really is called `valueQuantity`.

**This is not in the probe** — the translator will not produce ELM for it, so
there is nothing to diff.

Worth recording how it got here. The semantic phase was taught to resolve these
names in v1.13.2, to stop it reporting findings on what looked like valid CQL.
The findings were correct: the eight it reported came from **this repository's own
tests**, and neither the published eCQM libraries nor FHIRHelpers writes a choice
element that way. So the extension is real and useful for reading FHIR data, but
it was added on the strength of a false premise, and a library written against it
is not portable to any other CQL tool.

## `start of` on a FHIR.Period evaluates to Date where the reference says DateTime

| | |
| --- | --- |
| reference `resultType` | `DateTime` |
| this engine, statically | `DateTime` — agrees |
| this engine, evaluated | `Date` |

The model declares `FHIR.Period` converts to `Interval<System.DateTime>`, and
both the translator and this engine's semantic phase say so. The evaluator types
the value by its JSON text instead: `"2020-03-01"` becomes a Date and
`"2020-03-05T10:00:00Z"` a DateTime, so one Period can have endpoints of two
different types.

Measured, and worth knowing before anyone prioritizes it: no wrong answer was
found from it. Comparisons, `during`, `overlaps`, `duration in`, and
`difference in` all answer correctly across the boundary, and comparing a Date
endpoint against an explicit DateTime gives null — which is what the
specification requires of mixed precision, not a defect.

`TestResultTypesAgainstTheReference` records it and does not fail, which is the
right shape for it: the static comparison is the one that has to hold.

## Not divergences, though the evaluated comparison lists them

`TestResultTypesAgainstTheReference` compares the type of a *value* against the
type the reference *inferred*, which is a different question and differs for
reasons that are not defects:

- a FHIR object evaluates to `Object` rather than to `Encounter.Hospitalization`;
- a choice evaluates to the branch that was present — `Quantity`, not the whole
  `Choice<…>`;
- `Enc.status` evaluates to `String` where the reference infers
  `EncounterStatus`, its declared code type;
- an interval evaluates to `Interval` without its point type in the name;
- a function is not a definition this engine can evaluate by name.

The four `Test*MatchTheReference` tests are the ones that have to pass, and they
compare what the semantic phase infers, with no data and no evaluation on either
side.
