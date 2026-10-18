# ADR-001: la semántica de CQL se programa desde su especificación

**Estado:** Propuesto. Pasa a «Aceptado» cuando el incremento 003 clasifique las divergencias que los manifiestos todavía marcan `unclassified`; hasta entonces no se cumple que cada diferencia tenga su evidencia (`NUCLEO.md` §10). Decisión del usuario, 18-10-2026.
**Fecha:** 18-10-2026 · **Plan:** [`docs/plans/002-adr-semantica-cql.md`](../plans/002-adr-semantica-cql.md)

## Contexto

La regla 5 de `NUCLEO.md` (§2) pide que todo salga de los packages FHIR. El principio 2 admite como excepción lo que una especificación define en prosa y no en definiciones, con un ADR por excepción, que cita la sección y el test que la verifica.

La semántica de CQL no está en ningún package. La define la prosa de su especificación: operadores, tipos y conversiones, nulos, precisión de fechas e intervalos, consultas. Es la librería entera, y un ADR por operador no aporta nada que la conformance no diga mejor. El principio 2 enumera las excepciones por área de especificación, por ejemplo «funciones de FHIRPath», y este ADR es de esa granularidad.

## Decisión

**La semántica de CQL se programa desde la especificación de CQL 1.5.3,** la edición normativa publicada en `https://cql.hl7.org/N1A` el 7 de marzo de 2025, según `https://cql.hl7.org/package-list.json`.
- **La única excepción de versión** es el grupo `Slice` de `CqlListOperatorsTest`, con sus 10 casos, todos marcados `version="2.0"`, que el motor también resuelve.
- **El traductor de referencia** es el motor Java 3.26.0, con las opciones que fija `internal/referenceharness/src/Evaluate.java`, `EnableResultTypes` y `EnableLocators`, y sin `compatibilityLevel`. Esas opciones deciden qué CQL acepta, y las respuestas de referencia se produjeron con ellas.

**Cada operador se escribe desde su sección,** que se cita en el comentario o en el test que lo prueba.

**Una desviación de la especificación no se escribe en silencio.** Queda en uno de estos manifiestos, con su sección o su evidencia:
- `conformance/conformance_test.go`:
  - `upstreamContradictions`: casos en que la suite contradice a la especificación, que tienen que seguir fallando;
  - `untypedOutputs`: salidas que la suite escribe sin el tipo de sus números.
- `engine_runtime_reference_test.go`: `decidedRuntimeAnswers`, `disputedRuntimeAnswers` y la lista `notOracle` de cada área.
- `testdata/snapshots/<área>.divergences.tsv`: cada respuesta que difiere de Java o de cql-execution, con su clasificación (defecto propio, defecto de la referencia o diferencia deliberada).

### Qué no cubre

- **Los nombres FHIR escritos en el código de producción,** como recursos, elementos y tipos: van en la revisión de dinamismo de `NUCLEO.md` §4, con su propio ADR.
- **El modelo FHIR:** sale del ModelInfo oficial y del FHIRHelpers oficial (plan `docs/plans/2026-08-11-official-modelinfo-fhirhelpers.md`), no de este ADR.
- **La terminología** (`ValueSet`, `CodeSystem`): la resuelve el proveedor que configura quien llama.

## Procedencia de la evidencia

- **La conformance:** `conformance/testdata/*.xml` son `tests/cql/` de `cqframework/cql-tests` en el commit que registra `conformance/testdata/SOURCE.tsv`, con el blob SHA-1 de git y el SHA-256 de cada archivo.
  - Re-vendorizar es `sh scripts/conformance-provenance.sh vendor <commit>`, y no se hace de otra forma.
  - `make conformance-provenance` compara el manifiesto con el árbol upstream; necesita red.
- **Las referencias:** el motor Java está fijado en `internal/referenceharness/pom.xml`, y cql-execution y cql-exec-fhir en `internal/referenceharness/js/package-lock.json`. Cada `<área>.java.json` y `<área>.js.json` registra el motor que lo produjo y el SHA-256 de su corpus.

## Modelo de amenaza de la verificación

**Contra qué protegen.** Las comprobaciones de este ADR protegen contra la deriva accidental:
- un test que deja de correr o se salta;
- un job de CI que alguien cambia sin darse cuenta de lo que verificaba;
- una divergencia nueva que entra sin clasificar;
- una suite o una referencia que cambian de origen.

**Contra qué no.** No protegen contra la subversión deliberada. Quien puede editar en un PR `.github/workflows/ci.yml`, el `Makefile` o los scripts de `scripts/` también puede editar `adr_test.go`, que los comprueba. Ese cambio se ve en el diff del PR, y la revisión humana del PR es el control.

La ronda 3 del plan 002 registra ejemplos de esa subversión: targets redefinidos como `@true`, o un `defaults.run.shell` que vuelve no-op cada comando.

## Verificación

- `TestConformance`: la suite oficial, con los manifiestos `upstreamContradictions` y `untypedOutputs`.
- `TestTheConformanceSuiteIsTheVendoredOne`: la suite es la que registra `SOURCE.tsv`.
- `make conformance-provenance`: `SOURCE.tsv` es el árbol de `cqframework/cql-tests` en su commit.
- `TestAnswersMatchTheReferences`: cada divergencia de Java y de cql-execution está en su manifiesto.
- `TestTheReferencesAreOfTheirCorpus`: las referencias son de su corpus y del motor fijado.
- `TestRuntimeMatchesTheReference`: el arnés del runtime, con `decidedRuntimeAnswers`, `disputedRuntimeAnswers` y `notOracle`.
- `TestTheUnclassifiedDivergencesDoNotGrow`: cada divergencia sin clasificar está en `testdata/snapshots/unclassified-debt.tsv`, de donde solo se puede quitar. Su número, por área, sale de `make evidence`, que además falla si un test nombrado no corre o se salta, incluso a través de un helper.
- `TestADRsHaveVerification`: este ADR declara su verificación, que existe y no se salta. Para que sea «Aceptado», CI tiene que tener, palabra por palabra, el job `adr-verification`. Ese job corre `sh scripts/debt-only-shrinks.sh`, que impide agregar deuda contra la rama base, y `make evidence conformance-provenance`.
