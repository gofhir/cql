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

Then fold `elm.json` into `testdata/reference/probe.expected.json`.

## Two things worth knowing

`cql-to-elm-cli`'s published POM is invalid — it declares
`hapi-fhir-structures-r5` with no version — so Maven drops its transitive
dependencies and the CLI cannot be resolved. This depends on the `cql-to-elm`
library directly and drives it from `Translate.java` instead.

Result types are not emitted by default. `EnableResultTypes` is what makes the
translator annotate them, and without it every `resultType` comes back empty.
