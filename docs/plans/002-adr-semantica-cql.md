# 002: ADR único de la semántica de CQL

**Estado:** cerrado el 18-10-2026, sin commit. **Fecha:** 18-10-2026.

## Problema

La regla 5 de la carta (`NUCLEO.md` §2) pide que todo salga de los packages FHIR, y el principio 2 dice que lo que una especificación define en prosa se programa, con **un ADR por excepción**. La semántica de CQL entera es prosa de su especificación: operadores, tipos, conversiones, nulos, precisión. Un ADR por operador no tiene sentido.

El usuario decidió el 18-10-2026 que se escriba **un ADR único**: la semántica de CQL se programa desde su especificación, y se verifica con la conformance oficial y con las dos referencias en `go test`.

Además, la carta (§3) exige que cada ADR declare su verificación, que un test del repo compruebe que esa verificación existe, y que un ADR solo sea «Aceptado» si CI la corre en verde. cql no tiene ADRs ni ese test.

## 1. Revisión previa

- **`gofhir/server`:** tiene `docs/adr/ADR-001` a `ADR-007`, en inglés, sin sección de verificación y sin un test que la compruebe. **Se descarta el formato,** porque no cumple la carta §3.
- **`gofhir/searchindex`** (el frente actual):
  - tiene `docs/adr/001…004`, en español, con **Estado** y una sección **«## Verificación»**;
  - su `adr_test.go` (`TestADRsHaveVerification`) exige el estado y la sección, y que cada test (`Test…`, `Fuzz…`, `Benchmark…`) y cada `make <target>` nombrado exista.
  - **Se toma, con una mejora:** no comprueba que CI corra la verificación, que es la condición de la carta para «Aceptado». En cql, un ADR «Aceptado» tiene que nombrar solo tests que corren con `go test ./...`, y CI tiene que correr `go test ./...`. Si nombra un `make <target>`, CI tiene que correr ese target.
- **Especificación:**
  - La versión es la que implementa el traductor de referencia 3.26.0. Su ELM declara `translatorVersion` 3.26.0 y no fija `compatibilityLevel`, así que usa la versión por defecto de esa línea, 1.5.
  - Las secciones se citan de `cql.hl7.org`; la página leída el 18-10-2026 era la 2.0.0, que conserva las de la 1.5.
  - No pude fijar una URL versionada de la 1.5.x: la página de historia se carga con JavaScript y no devolvió contenido. El ADR lo dice así, en vez de citar una URL sin verificar.
- **Conformance:** `conformance/testdata/` tiene 16 archivos de `cqframework/cql-tests`.
  - El origen solo está en un mensaje de commit (#12: `cql-tests@727219f`).
  - **Lo medí el 18-10-2026:** los 16 son idénticos byte a byte a los de `cqframework/cql-tests@727219f492c0c84bf7a9c3589b1214e3b4e60667` (`tests/cql/`).
  - La carta pide fuentes «versionadas o descargadas con SHA-256 fijado»: están versionadas, pero sin procedencia ni SHA-256 en el repo. **Se mejora:** un manifiesto con el commit y el SHA-256 de cada archivo, más un test que lo comprueba.
- **Referencias:** el motor Java 3.26.0 está fijado en `internal/referenceharness/pom.xml`, y cql-execution 3.3.2 en `js/package-lock.json`.
  - Cada `<area>.java.json` y `.js.json` lleva el SHA-256 de su corpus, y `TestTheReferencesAreOfTheirCorpus` lo comprueba.
  - **Se toma como está.**
- **Nombres FHIR escritos en el código de producción:**
  - Hay del orden de 60 ocurrencias, sobre todo en `model/modelinfo.go` y `eval/evaluator.go`. El conteo se hizo con un grep aproximado y queda para el incremento que los inventaríe.
  - Son el eje «Dinamismo» de la carta §4, que pide eliminar o documentar cada caso.
  - **Fuera de este incremento:** es la revisión de cuatro ejes de §4, un incremento propio. El ADR único no cubre nombres FHIR, solo la semántica de CQL.

## 2. Plan

### Alcance

1. **`docs/adr/001-semantica-cql-desde-su-especificacion.md`,** en español. Contiene:
   - **Contexto:** la regla 5 y el principio 2.
   - **Decisión:** la semántica de CQL se programa desde su especificación, en la versión del traductor de referencia, y las secciones se citan en los comentarios y los tests de cada operador.
   - **Qué no cubre:** los nombres FHIR del código (§4, pendiente) y el modelo FHIR, que sale del ModelInfo oficial y del FHIRHelpers oficial (plan `2026-08-11-official-modelinfo-fhirhelpers.md`).
   - **Diferencias conocidas,** cada una registrada donde se verifica:
     - `upstreamContradictions` y `untypedOutputs` de la conformance;
     - los manifiestos `<area>.divergences.tsv`.
   - **Verificación:**
     - `TestConformance`, la suite oficial, con su procedencia fijada por `TestTheConformanceSuiteIsTheVendoredOne`;
     - `TestAnswersMatchTheReferences` y `TestTheReferencesAreOfTheirCorpus`;
     - `TestRuntimeMatchesTheReference`;
     - `TestADRsHaveVerification`.
2. **`adr_test.go`,** tomado de searchindex y adaptado: `TestADRsHaveVerification`, con la mejora de CI.
3. **`conformance/testdata/SOURCE.md`,** con el repo, el commit completo, la ruta y el SHA-256 de cada archivo. Lo comprueba `TestTheConformanceSuiteIsTheVendoredOne`, en `conformance/`.
4. **El estado del ADR:** «Aceptado» en el mismo PR, porque CI corre `go test ./...`, que incluye toda su verificación. El test lo exige.

### Fuera del alcance

- El inventario de nombres FHIR del código y los otros ejes de la revisión §4.
- Traducir los comentarios.
- Re-vendorizar la suite a un commit más nuevo.

### Criterio de salida

- `TestADRsHaveVerification` pasa, y falla ante un ADR sin estado, sin verificación, con un test inexistente, o con un `make` target que no existe o que CI no corre siendo «Aceptado».
- `TestTheConformanceSuiteIsTheVendoredOne` pasa, y falla ante un byte cambiado, un archivo de más o uno de menos.
- `make check` en verde.

### Tests y mutaciones

| Mutación | Qué tiene que fallar |
|---|---|
| El ADR nombra un test inexistente | `TestADRsHaveVerification` |
| Se quita la sección «## Verificación» | `TestADRsHaveVerification` |
| Un ADR «Aceptado» nombra un `make` target que CI no corre | `TestADRsHaveVerification` |
| Se cambia un byte de un XML de la suite | `TestTheConformanceSuiteIsTheVendoredOne` |
| Se agrega un XML a la suite sin registrarlo | `TestTheConformanceSuiteIsTheVendoredOne` |

Las mutaciones van en `scripts/mutations/002-adr.sh`, con la misma forma que las del 001, y entran en `make mutations`.

**Versiones FHIR:** no aplican. Es gobierno del repo, no semántica.

**Dependencias:** ninguna nueva.

## 3. Ronda adversarial

Hubo dos revisores en paralelo, y los dos ejecutaron en copias del repo: Codex en una copia con escritura, Fable en la suya. Los hallazgos C son de Codex y los F de Fable. Verifiqué con una sonda los marcados.

**Lo que los dos confirmaron sin refutar:**
- los cuatro tests que nombra el ADR corren sin saltos; `TestConformance` da 2031 PASS, 0 SKIP;
- CI corre `go test ./...` en dos jobs;
- los 16 XML coinciden con `cql-tests@727219f`.

| # | Hallazgo | Resolución |
|---|---|---|
| C1, F1 | El test de searchindex se satisface en vacío. Fable lo probó: acepta un test bajo `testdata/` que `go test ./...` no corre, y un `make references` que CI no corre. Codex lo probó con un test con `t.Skip`, uno con build tag y un benchmark | **Aceptado.** Los tests se recogen de los packages de `go list ./...`. CI tiene que tener una línea `run:` no comentada con `go test` sobre `./...` y sin `-run`. Un target que nombre un ADR «Aceptado» tiene que estar en una línea `run:` de CI. Se rechazan `t.Skip` y `testing.Short` en el cuerpo del test nombrado, una heurística declarada como tal. Un `Benchmark` o `Fuzz` exige que CI corra `-bench` o `-fuzz`. Cada caso es una mutación |
| C2 | Cambiar el `engine` de un JSON de referencia a `FAKE-0.0` no falla nada. **Verificado:** `readReference` solo compara el SHA-256 | **Aceptado:** `TestTheReferencesAreOfTheirCorpus` compara el `engine` con la versión fijada en `pom.xml` y en `package-lock.json`. Va con su mutación |
| C2, F3 | La versión de CQL es evidencia débil. **Verificado:** `cql.hl7.org/package-list.json` publica la 1.5.3 normativa en `cql.hl7.org/N1A` (2025-03-07); la página leída era la 2.0.0 (R2, uso a prueba, 2026-07-29) | **Aceptado:** el ADR cita la 1.5.3 en N1A. **Verificado:** la suite trae 11 casos de versión 2.0 (`Slice*`), y el ADR los nombra como lo único de la 2.0. Registra además las opciones del traductor con que se produjeron las referencias, las de `Evaluate.java`, porque cambian qué CQL se acepta |
| F2 | Las diferencias no están todas clasificadas. **Verificado:** 855 de 2115 filas de los manifiestos dicen `unclassified`: 240 de 243 en `arithmetic`, 143 de 593 en `fhir` y 472 de 472 en `types`. `-update-divergences` las escribe solas | **Aceptado en parte.** En este incremento entra un trinquete: ningún área puede sumar filas sin clasificar, así que una divergencia nueva tiene que llegar con su nota, y el número sale del test, no se copia. **Clasificar las 855 es un incremento propio.** El estado del ADR mientras tanto lo decide el usuario |
| C4, F4 | El SHA-256 acredita integridad, no procedencia: editar un XML y su hash deja todo verde | **Aceptado:** el manifiesto (`conformance/testdata/SOURCE.tsv`, en inglés porque lo lee el código) guarda la ruta, el blob SHA-1 de git y el SHA-256. El unitario compara contra el manifiesto, y `make conformance-provenance`, con red y separado de los unitarios, compara los blobs con el árbol de `cql-tests` en el commit vía la API de GitHub. Ese target es el procedimiento de re-vendorización, y el ADR lo describe. Las mutaciones cubren un byte cambiado, un archivo de más, uno de menos, una entrada duplicada y el manifiesto regenerado después de editar |
| C3, F5 | Un ADR único puede volverse un cheque en blanco | **Aceptado como límite:** el ADR cierra su alcance por exclusión, con lo que no cubre y dónde va: los nombres FHIR del código, el ModelInfo, FHIRHelpers y la terminología. Exige que toda desviación de la especificación quede en un manifiesto con su sección, y los enumera todos, incluidos `decidedRuntimeAnswers`, `disputedRuntimeAnswers` y `notOracle`. El principio 2 enumera excepciones por área de especificación («funciones de FHIRPath»), y el ADR es de esa granularidad |
| C3 | Una matriz sección → regla → test | **Rechazado:** cada XML de la suite declara la sección que cubre (atributo `reference`), así que la matriz se deriva de la suite y no se escribe a mano |
| C5 | El cierre no tiene un reporte generado | **Aceptado:** `make evidence` escribe un reporte con fecha. Lista por cada test nombrado cuántos RUN, PASS y SKIP tuvo (`go test -json`), las filas sin clasificar por área y el resultado de las mutaciones. El registro lo enlaza |
| F6 | `make mutations` corre solo el 001, y la tabla omite «sin Estado» | **Aceptado:** `make mutations` corre todos los scripts de `scripts/mutations/`, y se agrega la mutación |

## 4. Aprobación

El usuario aprobó el plan ajustado el 18-10-2026. También decidió:
- **El estado del ADR:** queda **Propuesto** mientras haya divergencias sin clasificar, y pasa a «Aceptado» cuando el incremento 003 las clasifique. El trinquete impide que crezcan mientras tanto.
- **El incremento 003** es la clasificación de las 855 divergencias, y va justo después del 002.

## 5 a 7. Tests, implementación y validación

- **`adr_test.go`** (`TestADRsHaveVerification`) toma su forma de searchindex y lo endurece con lo que encontró la ronda:
  - recoge los tests de los packages de `go list ./...`;
  - rechaza los saltos en el cuerpo del test nombrado;
  - a un ADR «Aceptado» le exige que CI corra `go test ./...` sin `-run` y que corra cada target que nombra;
  - un `Benchmark` o un `Fuzz` exige `-bench` o `-fuzz` en CI.
- **`conformance/source_test.go`** (`TestTheConformanceSuiteIsTheVendoredOne`) y **`conformance/testdata/SOURCE.tsv`**: el manifiesto lo escribió `sh scripts/conformance-provenance.sh vendor 727219f`. Los 16 XML no cambiaron, y `check` los confirma contra el árbol upstream.
- **`TestTheReferencesAreOfTheirCorpus`** compara además el `engine` de cada referencia con `pom.xml` y `package-lock.json`.
- **`TestTheUnclassifiedDivergencesDoNotGrow`**, el trinquete: deja bajar el número de filas sin clasificar y nunca subirlo.
- **`docs/adr/001-semantica-cql-desde-su-especificacion.md`**, en estado Propuesto.
- **El `Makefile`:**
  - `conformance-provenance` y `evidence` son targets nuevos;
  - `mutations` corre todos los scripts de `scripts/mutations/`.
- **`make evidence`** escribe `docs/evidence/<fecha>.md`. El de esta fecha es [`docs/evidence/2026-10-18.md`](../evidence/2026-10-18.md).
- **Mutaciones:** `sh scripts/mutations/002-adr.sh`.
  - Cada mutación declara el comando que tiene que fallar y el texto que su salida tiene que contener, así que se detecta por la razón correcta.
  - Las dos de CI neutralizan los targets nombrados, de modo que prueban la comprobación de CI por sí sola.
  - La de un manifiesto regenerado después de editar necesita red.
  - El resultado está en el reporte de evidencia.
- **Chequeos:** `make check` en verde. Incluye las mutaciones del 001 y del 002.

## 8. Code review

### Ronda 1 (Codex y Fable)

Los dos ejecutaron en sus copias y cada hallazgo vino con una sonda que lo demostraba. Codex dejó 8 medios y uno bajo; Fable, 2 medios y 5 bajos. La ronda no converge.

| Hallazgo | Severidad | Resolución |
|---|---|---|
| Un test detrás de una build tag pasa la guarda (Codex y Fable) | Media | **Corregido:** los archivos salen de `TestGoFiles` y `XTestGoFiles` de `go list -json`. Lo prueba una mutación |
| Un salto a través de un helper no se ve (Codex) | Media | **Corregido en `make evidence`:** falla si un test nombrado no corre, se salta (también por un helper, porque lo ve `go test -json`) o falla. La heurística del test queda declarada. Lo prueba una mutación |
| `if: false` en un paso o en un job sigue contando (Codex y Fable) | Media | **Corregido sin dependencia nueva:** se lee la estructura del workflow por indentación, y solo cuentan los `run:` de pasos y jobs sin `if:`. Lo prueban dos mutaciones |
| `make references-shadow` pasa por `make references` (Codex y Fable) | Media | **Corregido:** se compara el target como palabra entre los argumentos de `make`. Lo prueba una mutación |
| `vendor` borra la suite antes de terminar de descargar (Codex) | Media | **Corregido:** se descarga y verifica todo en staging antes de reemplazar. Lo verifiqué con un fallo inyectado en la segunda descarga: `vendor` sale con 1 y la suite queda idéntica (`diff -rq`) |
| El regex del `pom.xml` lee comentarios (Codex) | Media | **Corregido:** se lee como XML y se elige `info.cqframework:engine`. Lo prueba una mutación |
| El trinquete por cantidad deja canjear deuda (Codex) | Media | **Corregido:** `testdata/snapshots/unclassified-debt.tsv` fija las 855 identidades, y solo se puede quitar de él. Lo prueba una mutación con un canje |
| `evidence.sh` no ve familias ni nombres sin comillas, y sale con 0 ante un error (Codex y Fable) | Media | **Corregido:** usa la misma regla que `adr_test.go`, expande las familias con `go test -list` y falla ante errores, saltos, tests que no corren y mutaciones sobrevivientes |
| `make check` necesita red (Fable) | Media | **Corregido:** la mutación de red corre en `make conformance-provenance` (`CQL_MUTATIONS_NETWORK=1`). Lo verifiqué sin red, con un proxy inválido: las mutaciones pasan |
| Una segunda sección «Verificación» se ignora (Fable) | Baja | **Corregido:** se exige exactamente una. Lo prueba una mutación |
| «11 casos de la 2.0» eran 10 casos y su grupo (Codex) | Baja | **Corregido** en el ADR |
| Un ADR «Aceptado» va a exigir que CI corra `make conformance-provenance` y `make evidence` (Fable) | Baja | **Registrado para el 003:** antes de pasar a «Aceptado», CI tiene que correr esos dos targets, o salen de la verificación |
| Una clave del trinquete sin área (Fable) | Baja | Desaparece con la deuda por identidades: una entrada de un área inexistente aparece como «classified now» |

### Ronda 2 (Codex y Fable): no converge, y es señal de diseño

Los dos verificaron ejecutando. Los 22 arreglos y mutaciones de la ronda 1 se sostienen: las sondas que antes sobrevivían ahora se detectan. Pero el código nuevo abre la misma clase de huecos:

- **El lector del workflow** (Fable y Codex, media): depende del orden, así que un `if:` después del `run:`, o un `if:` de job después de `steps:`, se escapa. Ignora `continue-on-error: true`. `make -n evidence` y `make -C evidence otro` pasan por `make evidence`. `go test -skip . ./...` y `-list` pasan por `go test ./...`, y la forma *flow* y los `run:` entre comillas no se leen.
- **La deuda** (Codex, media): desclasificar una fila y agregar su identidad a `unclassified-debt.tsv` pasa. El test compara dos archivos que el mismo PR puede editar, así que «solo quitar» no lo puede imponer un test del repo.
- **Bajos:** el corte de la sección de verificación en el primer `## Verificación`, aunque esté en un `### Verificación manual`, y un error poco claro ante CRLF en la deuda.

**Diagnóstico:** cada arreglo del lector parcial de YAML y shell abre casos nuevos de sintaxis válida, y la carrera no se cierra. Por la carta §10 y la skill, se consulta al usuario antes de la ronda 3, con un rediseño.

### Rediseño aprobado por el usuario (vuelta al paso 2)

**CI: un paso canónico más la evidencia.** Se deja de leer YAML y shell.
- CI tiene un job fijo, `adr-verification`, que corre `sh scripts/debt-only-shrinks.sh` y `make evidence conformance-provenance`.
- `adr_test.go` solo verifica que ese job exista **con su texto exacto**: sin `if:`, sin `continue-on-error` y sin claves agregadas.
- La prueba de que cada test nombrado corrió y no se saltó la da `make evidence` (`go test -json`), dentro de CI.
- Un ADR «Aceptado» puede nombrar tests, que `make evidence` corre, y los targets `evidence` y `conformance-provenance`. No puede nombrar un `Benchmark` ni un `Fuzz`, porque la evidencia no los corre.

**La deuda: CI contra la rama base.**
- `scripts/debt-only-shrinks.sh` exige que el diff de `testdata/snapshots/unclassified-debt.tsv` contra la rama base solo quite líneas.
- En un PR compara contra `origin/$GITHUB_BASE_REF`; en un push, contra el commit anterior.
- El job hace checkout con `fetch-depth: 0`.

**Bajos de la ronda 2:**
- la sección se corta en `\n## Verificación\n`, como se cuenta;
- la deuda con CRLF da un error claro.

**Implementado:**
- **`adr_test.go`:** se fue el lector del workflow, y queda la comparación exacta de `adrVerificationJob`.
- **`.github/workflows/ci.yml`:** tiene el job `adr-verification`.
- **`scripts/debt-only-shrinks.sh`** compara la deuda contra la rama base.
- **Mutaciones:** se fueron las del lector y entraron las del job: sin el job, con `if: false` en el job, con `continue-on-error`, con un `if:` después del `run:`, con `make -n`, nombrando `make check` o un benchmark, y deuda agregada contra la base. Son 24 sin red, y la de red corre en `make conformance-provenance`.
- **Control de la razón:** con el job intacto, un ADR «Aceptado» pasa; lo comprobé en una copia. Así, las mutaciones del job se detectan por la razón correcta.
- **`make check`, `make evidence` y `make conformance-provenance`:** en verde.

### Ronda 3 (Codex y Fable): no converge; se llega al tope y se consulta al usuario

Los dos confirmaron el diseño nuevo:
- las 24 mutaciones se detectan;
- con el job intacto, un ADR «Aceptado» pasa;
- un `if:` agregado, una tab, un ancla de YAML y una línea agregada se detectan.

Lo que queda se agrupa en dos clases.

**Defectos reales, baratos de arreglar:**
- **El corte de la sección en `evidence.sh`** (Codex, media): usa otro delimitador que `adr_test.go`, así que un `### Verificación manual` anterior deja un test nombrado fuera de la evidencia.
- **La deuda falla abierta** (Codex y Fable, media):
  - una base que no existe, o que no tiene el archivo, deja pasar entradas nuevas;
  - `DEBT_BASE` se puede fijar desde el `env:` del workflow sin tocar el job;
  - un push de varios commits se compara solo contra `HEAD~1`.
- **El encabezado del workflow no está fijado** (Fable, media): `on: workflow_dispatch`, `paths-ignore: ["**"]`, o el job escondido en un escalar de `name:` dejan el job intacto y sin correr en un PR.
- **Un comentario después del job lo invalida** (Fable, baja).

**Subversión deliberada, un límite del modelo de amenaza** (Codex y Fable): redefinir los targets como `@true`, `defaults.run.shell` como no-op, o cambiar los scripts. Quien puede editar `ci.yml`, el `Makefile` o los scripts en un PR también puede editar `adr_test.go`. Ningún test del repo cierra eso; lo cierra la revisión del diff.

**Decisión del usuario ante el tope:** arreglar los defectos reales, cada uno con su mutación, verificarlos con `make check` y cerrar sin una cuarta ronda. El límite de la subversión deliberada se declara en el ADR como modelo de amenaza.

**Arreglado tras la ronda 3:**
- **`evidence.sh`** corta la sección como `adr_test.go` y exige una sola.
- **`debt-only-shrinks.sh`:**
  - recibe la base como argumento, o la toma del PR o del `before` del push, nunca de una variable;
  - falla si la base no resuelve;
  - sin base, acepta solo el contenido inicial, fijado por su SHA-256.
- **`adr_test.go`:**
  - fija también el encabezado del workflow (`ciWorkflowHeader`) y no admite claves de nivel superior después de `jobs:`;
  - un comentario después del job ya no lo invalida.
- **El ADR** tiene la sección «Modelo de amenaza de la verificación».
- **Mutaciones nuevas:** base que no resuelve, deuda sin base distinta de la registrada, push de dos commits, workflow sin `pull_request`, `defaults` después de los jobs y `### Verificación manual`.
  - Los casos que necesitan un repo git tienen un control, la misma preparación sin el defecto, que la línea base exige que pase.
  - Son 30 sin red, más la de red.

## 9. Cierre

- **Chequeos:** `make check`, `make evidence` y `make conformance-provenance` en verde. La evidencia está en [`docs/evidence/2026-10-18.md`](../evidence/2026-10-18.md), y se regenera con `make evidence`.
- **El ADR 001** queda **Propuesto**. Pasa a «Aceptado» cuando el 003 clasifique la deuda. El job `adr-verification` ya está en CI, así que en ese momento no falta nada más.
- **Queda pendiente:**
  - el incremento 003, que clasifica las 855 divergencias;
  - la revisión de dinamismo de §4, con los nombres FHIR del código;
  - la traducción de los comentarios.
