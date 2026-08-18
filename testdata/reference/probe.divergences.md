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
it was added on the strength of a false premise.

**The engine says so now.** Resolving one of these names produces a warning
naming the portable form:

```
Observation.valueQuantity is the FHIR JSON field name, not an element the model
declares; write `value as FHIR.Quantity` — the reference translator refuses this
form
```

A warning and not an error, deliberately: the evaluator reads FHIR JSON, where
the field really is called valueQuantity, and refusing to evaluate would break
libraries that work. What was missing was telling the author that theirs will not
translate anywhere else. Finding this is what the probe was for; it had been
silent since v1.13.2.

## `start of` on a FHIR.Period evaluated to Date where the reference says DateTime

**Resolved.** Kept here because how it was found and what it cost are worth
reading together.

The model declares that FHIR.dateTime holds a System.DateTime, and both the
translator and this engine's semantic phase said so. The evaluator typed the
value by its JSON text instead: `"2020-03-01"` became a Date because it carried
no time, so one Period could have endpoints of two different types.

The damage is where the type *is* the question:

    start of Enc.period is DateTime   was false
    start of Enc.period as DateTime   was null
    if x is DateTime then … else …    took the other branch

Fixing it changes what a duration answers, and that is not a regression. A FHIR
dateTime written to the day means the time is unknown, not midnight, so two
indeterminate instants are an uncertain number of days apart — `duration in days`
over such a Period is now Interval[3, 4] where it was 4. The conformance corpus
holds the same rule: `years between DateTime(2005) and DateTime(2010)` is
Interval[4, 5], while the same to month precision is exactly 4. A `Date` stays
exact, because a Date *is* the day rather than an instant inside it.

That distinction is what a detour nearly got wrong. The uncertainty looked like a
contradiction — two spellings of the same instant giving different answers — and
the corpus is what settled it: the engine passes all four official cases, and
Date and DateTime differ here because they mean different things.

Agreements with the reference went from 19 to 22.

### The search for damage looked in the wrong place twice

This entry used to say that no wrong answer came of the mixed endpoints at all,
on the grounds that comparisons, `during`, `overlaps`, `duration in` and
`difference in` answer correctly across the boundary. They do not:

```cql
E.period during "Measurement Period"    →  null
```

That is the comparison deciding which population a patient belongs to, and 11 of
the 19 published eCQM libraries write it, 38 times between them.

**The cause is not the mixed types.** A FHIR dateTime carrying a time must carry
a timezone offset, so a served period reads `"2020-03-05T10:00:00Z"`, while every
published library declares its measurement period without one. Comparing a value
that writes an offset against one that does not reported a precision mismatch —
even where both were specified to the same precision, which is what gave the
diagnosis away. It was never a precision that was missing.

Two things are worth keeping from how the claim came to be wrong. It was measured,
and the measurement was real: a Date endpoint against an explicit DateTime does
give null, and that is the specification's mixed-precision rule rather than a
defect. What was not measured was the pair that actually occurs in production —
both endpoints DateTime, one carrying an offset. **The probe compares types. It
has no data, so it cannot find a wrong answer, and an entry in this file saying
none exists is claiming something the probe was never able to check.**

## Not divergences, though the evaluated comparison lists them

`TestResultTypesAgainstTheReference` compares the type of a *value* against the
type the reference *inferred*, which is a different question and differs for
reasons that are not defects:

- a FHIR object evaluates to `Object` rather than to `Encounter.Hospitalization`;
- a repeated element evaluates to `List`, where the reference names the element
  type: `List<Encounter.Location>`. The fixture used to serve one location and a
  single-element repeated element unwraps to a scalar, so this read as `Object`
  and was filed under the line above — it is a different thing, and the fixture
  serves two now;
- a choice evaluates to the branch that was present — `Quantity`, not the whole
  `Choice<…>`;
- `Enc.status` evaluates to `String` where the reference infers
  `EncounterStatus`, its declared code type;
- an interval evaluates to `Interval` without its point type in the name;
- a function is not a definition this engine can evaluate by name.

The four `Test*MatchTheReference` tests are the ones that have to pass, and they
compare what the semantic phase infers, with no data and no evaluation on either
side.
