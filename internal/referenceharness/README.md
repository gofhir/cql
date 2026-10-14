# Reference harness

Translates CQL with the reference implementation and records what it decided, so
that what this engine decides can be diffed against it rather than argued about.

The plan calls for this in Etapa 5: *"traducir un corpus y comparar qué tipo
infiere y dónde inserta cada conversión convierte esta etapa en algo diffable."*

## What it records

For each definition, three things this engine has to agree with eventually:

| | |
| --- | --- |
| `resultType` | the type the reference infers — what a semantic phase must arrive at |
| `locator` | where the expression begins, one-based, which our `ast.Position` is compared against |
| `conversions` | the FHIRHelpers calls the translator inserts, which Etapa 3 applies at evaluation instead |

## Running it

Needs Docker. It is not part of CI: the reference translator pulls a JVM and
some thirty jars, which is too much to ask of every build. The output is
committed under `testdata/reference/` instead, and the tests named
`Test*TheReference` in the root package diff against that — so the comparison runs everywhere, and only regenerating it
needs the toolchain.

```sh
cd internal/referenceharness
docker run --rm -v "$PWD":/w -w /w maven:3-eclipse-temurin-21 \
  mvn -q -B dependency:copy-dependencies -DoutputDirectory=/w/lib
docker run --rm -v "$PWD":/w -w /w maven:3-eclipse-temurin-21 \
  sh -c 'javac -cp "lib/*" -d out src/Translate.java'
docker run --rm -v "$PWD":/w -v "$PWD/../../testdata/reference":/cql -w /w \
  maven:3-eclipse-temurin-21 java -cp "out:lib/*" Translate /cql/probe.cql /cql \
  > elm.json
```

Then fold it, which used to be a manual read of a 45KB document and is why the
probe sat at five definitions for so long:

```sh
cd ../..
go run ./internal/referenceharness/fold \
  -in internal/referenceharness/elm.json \
  -source probe.cql \
  -out testdata/reference/probe.expected.json
```

The fold keeps the type, the locator and the FHIRHelpers calls, and marks whether
each definition is an expression or a function — the translator types both, and
`sema.Result` carries only the expressions.

A translation that fails prints nothing and exits non-zero, so a reference cannot
be derived from a failed run. That has already been useful: the translator
refuses `Obs.valueQuantity`, which this engine accepts, and finding out that way
is what `testdata/reference/probe.divergences.md` records.

## Running the reference engine

Translation says what the reference decides about types; it says nothing about
values. `Evaluate.java` runs each expression definition with
`info.cqframework:engine`, at the same version as the translator, and records
the value with its CQL type. A definition that throws is recorded as an error
and does not stop the rest.

```sh
cd internal/referenceharness
docker run --rm -v "$PWD":/w -w /w maven:3-eclipse-temurin-21 \
  sh -c 'javac -cp "lib/*" -d out src/Translate.java src/Evaluate.java'
docker run --rm -v "$PWD":/w -v "$PWD/../../testdata/reference":/cql -w /w \
  maven:3-eclipse-temurin-21 java -cp "out:lib/*" Evaluate /cql/runtime.cql /cql \
  > runtime.json && mv runtime.json ../../testdata/reference/runtime.expected.json
```

The output goes through a temporary file on purpose. A failed run prints
nothing, and redirecting straight onto the committed reference would empty it
before the error showed.

The encoding does not go through anything of this engine's. Numbers are kept as
text, so an Integer and a Decimal stay apart. The test then compares decimals by
value, since `1.0 = 1` in CQL, and so it does not check scale. A DateTime keeps its offset and precision, and
also an `instant` in full ISO 8601. That field is the one to compare, because
the engine's `toString()` drops the offset at some precisions and keeps it at
others. null is JSON null and an empty list is still a list. A tuple's elements
go under `fields`, not `elements`, so the key always means one thing. A value
with no encoding is recorded as `Unencoded:<class>` instead of being guessed at.
The SHA-256 of the `.cql` is recorded too.

Evaluation time is fixed at `2026-01-01T00:00:00-03:00`, so `Now()` and
DateTimes without an offset do not depend on the machine. It is deliberately
not UTC. At UTC, an engine that defaulted to UTC instead of the evaluation
offset would give the same answers, and the difference would not show.

`TestRuntimeMatchesTheReference` diffs this engine against that file. It
compares with its own code, not with this engine's `String` or `Equal`.

- **Lists:** compared in order, except the few where the specification fixes
  no order (`union`, `except`, `distinct`). Those are listed by name, each with
  its reason.
- **DateTimes:** compared by instant, offset and precision. The offset is the
  effective one, so a value with none does not pass as UTC.
- **Known divergences:** each is pinned with this engine's exact answer. A
  divergence that gets fixed fails the test until it is removed from the list,
  and so does one that changes to a different wrong answer.
- **Stale reference:** a `.cql` edited without regenerating fails on its hash,
  instead of reading as a divergence of this engine. A reference with
  definitions missing fails against the names read from the `.cql` text.

- **Pinned answers:** some definitions the reference cannot settle. These
  include quantities, since with this configuration the reference does not
  convert units. Only this engine's answer is checked for them, because
  agreeing with the reference there could be a regression.

`testdata/reference/runtime.divergences.md` explains each case.

## Two things worth knowing

`cql-to-elm-cli`'s published POM is invalid — it declares
`hapi-fhir-structures-r5` with no version — so Maven drops its transitive
dependencies and the CLI cannot be resolved. This depends on the `cql-to-elm`
library directly and drives it from `Translate.java` instead.

Result types are not emitted by default. `EnableResultTypes` is what makes the
translator annotate them, and without it every `resultType` comes back empty.
