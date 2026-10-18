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

## Running the second runtime

`js/evaluate.js` runs the same probe with `cql-execution`, pinned exactly in
`js/package.json` and `js/package-lock.json`. It runs the ELM the reference
translator produces, so it is a second opinion on evaluation, not on
translation. It writes the same encoding as `Evaluate.java`, at the same
evaluation time.

`cql-execution` differs from the Java engine in three representations, and
the encoding accounts for each:

- **Integer and Decimal** are one JavaScript number. A number's type is read
  from the static type the translator annotated, so `cql-execution` is not
  consulted on a row about the type itself (`ListIntDec`).
- **A Decimal** is computed in binary floating point, so `0.1 + 0.2` is
  `0.30000000000000004` there. It is recorded as held, not rounded to the
  scale of 8. Rounding would put an answer in its mouth: `10.0 / 3.0 =
  3.33333333` is false in `cql-execution`. It is not consulted on decimal
  arithmetic.
- **A Time** is a DateTime with no date, so it is recognised before
  DateTime. **A Long** comes back as text, so its type is read first.

```sh
cd internal/referenceharness
docker run --rm -v "$PWD/js":/w -w /w node:20 npm ci
docker run --rm -v "$PWD":/w -v "$PWD/../../testdata/reference":/cql -w /w \
  maven:3-eclipse-temurin-21 java -cp "out:lib/*" Translate /cql/runtime.cql /cql \
  > elm.json
docker run --rm -v "$PWD":/w -v "$PWD/../../testdata/reference":/cql -w /w \
  node:20 node js/evaluate.js elm.json /cql/runtime.cql \
  > runtime-js.json && mv runtime-js.json ../../testdata/reference/runtime.expected.js.json
```

## The comparison

`TestRuntimeMatchesTheReference` diffs this engine against both files. It
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

- **Per-reference exclusions:** a reference is not consulted where it is not
  an oracle. The Java engine, as configured here, does not convert units, so
  it is excluded from the quantity rows. `cql-execution` settles them.
- **References that disagree:** if this engine agrees with one reference and
  not with the other, the test fails and asks for the specification to settle
  it. It is never settled by majority, since two runtimes that share a
  translator can share its mistake.
- **Disputed answers:** where the references disagree with each other, the
  row records how the disagreement was settled and pins this engine's answer.
  The test fails once the references agree, since the row is then no longer
  a dispute. A disagreement goes to the conformance suite and the
  specification, never to a vote, and the row records which settled it, as
  `NoReturn` does. A reference that answers wrong stays in the row rather than
  being excluded: an exclusion (`notOracle`) is for what a runtime cannot
  answer at all, and only a disputed row fails when the reference is fixed.
- **Decided answers:** where this engine differs from every reference on
  purpose, only its own answer is checked, and changing it means arguing with
  the recorded reason.

`testdata/reference/runtime.divergences.md` explains each case.

## Differential testing

The probe and runtime rows are written by hand, and they pin what was decided.
They do not find what nobody thought to write. A change to how expressions
evaluate is checked against the reference engine over generated expressions,
before it is reviewed and again after every fix:

1. **Baseline.** Before changing anything, generate a library for the area the
   change touches and record what the two engines disagree on.
2. **Change, and compare.** Run the same library again, against the baseline.
   Every new difference is read and sorted into one of three:
   - a decision this engine makes against the reference, which goes into
     `decidedRuntimeAnswers` and `runtime.divergences.md`;
   - a failure of the reference, settled by the specification or the
     conformance suite;
   - a defect, which is fixed.
3. **Review** what this cannot see: quantities, nulls, FHIR data past the range
   of a literal, static types, uncertainty and performance.
4. **After each fix**, compare again. Nothing new should appear.

This came out of a change to the Decimal scale that went through ten rounds of
case-by-case review, each one finding defects the last had not looked at. A run
of 2,600 generated expressions found the rest in one pass, and showed after
each fix that nothing else had moved.

```sh
go run ./internal/referenceharness/differential/gen -area arithmetic -seed 7 -n 600 > /tmp/diff/fuzz.cql
cd internal/referenceharness
docker run --rm -v "$PWD":/w -v /tmp/diff:/cql -w /w maven:3-eclipse-temurin-21 \
  java -cp "out:lib/*" Evaluate /cql/fuzz.cql /cql > /tmp/diff/java.json
cd ../..
go run ./internal/referenceharness/differential/compare \
  -cql /tmp/diff/fuzz.cql -reference /tmp/diff/java.json -report /tmp/diff/before.json
# … change the engine …
go run ./internal/referenceharness/differential/compare \
  -cql /tmp/diff/fuzz.cql -reference /tmp/diff/java.json -baseline /tmp/diff/before.json
```

`gen` is reproducible from its seed, so a definition's name is the same
expression in every run, and `-baseline` reports by name what is new and what
is gone. The reference engine's answers only need computing once per library:
they do not change with this engine.

`compare` evaluates at the fixed evaluation time `Evaluate.java` uses. It
compares numbers by value and sorts each difference into one of three:

- `engine-panic`: always a defect, and reported even where the reference
  failed too.
- `reference-error`: the reference threw and this engine answered.
- `different`: the two engines gave different answers.

Two runs of different areas, or of two seeds, cover more than one large run of
one.

An area is a function in `gen`. `arithmetic` covers the operators, the
arithmetic functions and the aggregates over literals, typed nulls and
divisions by zero. A change elsewhere — intervals, temporal arithmetic, lists,
strings — adds the area it needs, with literals chosen for that area's edges, as
`arithmetic` chooses the Decimal scale's.

`integer` covers the operators and functions whose Integer or Long result can
fall past its type, over the edges of both. There the Java engine is not an
oracle: it wraps a result past its type, where the specification makes it
null, so most of the differences in a run are its own. Read them against the
specification — the result as an exact number, null past 32 bits for an
Integer or 64 for a Long — and compare runs with `-baseline`, so that what a
change moves stands apart from what Java always answers.

`scope` covers how a name resolves: a preamble of definitions and functions,
and queries, lets and calls that bind the same names as aliases, lets and
operands, with bodies that refer to both. A definition evaluates in its
library's scope, and the Java engine is the oracle; cql-execution resolves a
definition's names where it is referenced. A let takes a name not bound
already, since one bound twice in a query is not CQL to settle by a run.

`types` covers `is`, `as` and `cast` of a value whose static type is Any — an
element of a `List<Any>` — against its own type, its supertypes and unrelated
ones, so that every type can be asked about as a downcast: the reference
translator refuses a cast between unrelated static types. The Java engine
fails at run time on Any as a target, so those rows are read against the
specification: every value is of type Any.

`fhir` covers FHIR data: `is`, `as` and `cast` of resources, their primitives
and choices, and of FHIR instances built in CQL, against System and FHIR
types, and the operators that read them. Its values come from resources a test
serves by fixed ids (`testdata/snapshots/fhir-data.json`). Both reference
engines run it over the same resources and the same FHIRHelpers 4.0.1 text —
the translator's own copy is the one this engine embeds, line for line — each
definition in a library of its own, since the translator refuses a whole file
for one invalid expression and this corpus holds them on purpose:

```sh
cd internal/referenceharness
docker run --rm -v "$PWD":/w -w /w maven:3-eclipse-temurin-21 \
  mvn -q -B dependency:copy-dependencies -DoutputDirectory=/w/lib
docker run --rm -v "$PWD":/w -w /w maven:3-eclipse-temurin-21 \
  sh -c 'javac -cp "lib/*" -d out src/Translate.java src/Evaluate.java'
mkdir -p /tmp/fhir && cp ../../testdata/snapshots/fhir.cql ../../testdata/snapshots/fhir-data.json \
  ../../fhirhelpers/FHIRHelpers-4.0.1.cql /tmp/fhir/
docker run --rm -v "$PWD":/w -v /tmp/fhir:/cql -w /w maven:3-eclipse-temurin-21 \
  java -cp "out:lib/*" Evaluate -each /cql/fhir.cql /cql /cql/fhir-data.json /cql/fhir.elm.json \
  > /tmp/fhir/java.json && mv /tmp/fhir/java.json ../../testdata/snapshots/fhir.java.json
docker run --rm -v "$PWD":/w -v /tmp/fhir:/cql -w /w maven:3-eclipse-temurin-21 \
  java -cp "out:lib/*" Translate /cql/FHIRHelpers-4.0.1.cql /cql > /tmp/fhir/FHIRHelpers.elm.json
docker run --rm -v "$PWD":/w -v /tmp/fhir:/cql -w /w node:20 node js/evaluate.js --each \
  /cql/fhir.elm.json /cql/fhir.cql /cql/FHIRHelpers.elm.json /cql/fhir-data.json \
  > /tmp/fhir/js.json && mv /tmp/fhir/js.json ../../testdata/snapshots/fhir.js.json
cd ../..
go run ./internal/referenceharness/differential/compare -each -data testdata/snapshots/fhir-data.json \
  -cql testdata/snapshots/fhir.cql -reference testdata/snapshots/fhir.java.json
```

The Java engine reads the resources through engine-fhir's R4 model resolver,
cql-execution through cql-exec-fhir 2.1.6, both pinned. A translation error is
the Java engine's answer for its definition, and cql-execution runs only the
ELM the translator wrote. What the three engines answer is inventoried in
`testdata/snapshots/fhir.divergences.md`; the references are held to the
corpus by `TestTheFHIRReferencesAreOfThisCorpus`.

## Answer snapshots

The differential says which answers are right; it needs a reference that can
answer, and over FHIR data none can yet. A snapshot says which answers moved,
and needs none. Each area's corpus is kept in `testdata/snapshots/<area>.cql`,
written once by `gen` with the seed noted in the manifest of
`engine_snapshot_test.go`, and beside it what this engine answers to every
definition, `<area>.answers.tsv`. `TestAnswersMatchTheSnapshots`, part of
`go test ./...`, fails on any answer that moved and lists them all.

A change that moves answers on purpose regenerates them:

```sh
go test -run TestAnswersMatchTheSnapshots -update-snapshots .
```

and the diff of the pull request is then the list of what it changed, each to
be accounted for before review. A move nobody can account for is a regression,
whatever the review of the code says.

The corpora are not regenerated when `gen` changes; a new or wider corpus is a
change of its own, with its answers as written by the engine it lands on.
Each definition is evaluated in a library of its own, with its area's header,
so that an invalid expression — the corpora hold them on purpose — fails alone.

## Divergences from the references

The snapshot says which answers moved; it does not say whether they moved
toward the reference engines or away. Each area also keeps what the Java
engine (`<area>.java.json`) and cql-execution (`<area>.js.json`) answer to its
corpus, run with `Evaluate -each` and `evaluate.js --each` as for the fhir area
above, and a manifest, `<area>.divergences.tsv`: one line per definition and
reference that differ, with a note saying why.

`TestAnswersMatchTheReferences`, part of `go test ./...`, compares this engine
with both and fails on any divergence the manifest does not record — a
regression, or a change to account for — and on any it records that is gone,
so that what a change settles is struck from the list in the same diff:

```sh
go test -run TestAnswersMatchTheReferences -update-divergences .
```

keeps the note of every entry that stays and marks a new one `unclassified`.
A divergence is not a defect by itself, and the notes say which it is: a
reference's own failure (in the integer area every one is, checked against the
exact-integer oracle of #135; in scope, cql-execution's resolution of a
definition's names), a CQL the Java translator refuses, a decision, or one
still to settle. The unclassified ones are the inventory of what remains to be
read; none of them is excused by being listed.

`TestTheReferencesAreOfTheirCorpus` holds every reference to its corpus by its
SHA-256, and fails rather than skips when one is missing.

## Two things worth knowing

`cql-to-elm-cli`'s published POM is invalid — it declares
`hapi-fhir-structures-r5` with no version — so Maven drops its transitive
dependencies and the CLI cannot be resolved. This depends on the `cql-to-elm`
library directly and drives it from `Translate.java` instead.

Result types are not emitted by default. `EnableResultTypes` is what makes the
translator annotate them, and without it every `resultType` comes back empty.
