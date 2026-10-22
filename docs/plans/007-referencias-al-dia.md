# 007: las referencias de comparación a su última versión

**Fecha:** 22-10-2026 · **Rama:** `chore/references-latest` · **Versión:** `test:`, sin release (no cambia el motor)

## Por qué

El arnés compara este motor contra el traductor y el engine de Java 3.26.0, y contra cql-execution 3.3.2 con cql-exec-fhir 2.1.6. La última versión de Java es la 5.4.0 (30-09-2026), dos majors después. El 22-10-2026 casi se reportó como defecto vigente uno de la 3.26.0 que la 5.4.0 ya corrige (backlog 1c13). Una referencia vieja produce divergencias falsas y reportes inválidos. El usuario fijó la regla ese día: siempre la última versión, sobre todo en las comparaciones.

| Referencia | Fijada | Última | Fuente |
|---|---|---|---|
| Traductor y engine de Java | `info.cqframework` 3.26.0 | `org.cqframework` 5.4.0 (`cql-to-elm-jvm`, `engine-jvm`, `engine-fhir-jvm`, `quick`) | Maven Central, `maven-metadata.xml` |
| cql-execution | 3.3.2 | 3.3.2 | npm |
| cql-exec-fhir | 2.1.6 | 2.2.0 | npm |

## Revisión previa (sondas del 22-10-2026, en el scratchpad)

- **`gofhir/server`:** no tiene arnés CQL. No hay nada que tomar.
- **La API de Java cambió.** `Evaluate.java` y `Translate.java` dan 51 errores de compilación contra la 5.4.0:
  - los valores de ejecución son una jerarquía sellada, `runtime.Value` (`runtime.String`, `runtime.Integer`, `runtime.List`…), y los recursos FHIR llegan como `ClassInstance`, no como objetos HAPI;
  - `LibrarySourceProvider` devuelve un `kotlinx.io.Source`, y `DefaultLibrarySourceProvider` toma un `kotlinx.io.files.Path`;
  - `CqlEngine.evaluate` toma `EvaluationParams`, y `RetrieveProvider.retrieve` cambió su firma;
  - el manifiesto ya no trae `Implementation-Version`: la versión está en `org.cqframework.engine.BuildConfig.IMPLEMENTATION_VERSION`;
  - `kotlin-logging` escribe una línea en stdout al iniciar, antes del JSON.
- **El port de sonda funciona** (`scratchpad/h540`). Para codificar una respuesta FHIR, reconstruye el objeto HAPI desde el `ClassInstance` (`toHapi`) y lo escribe con el codificador de la 3.26.0, para que el formato no cambie. Es código del oráculo y tiene que probarse con mutaciones.
- **Lo que cambia en Java:** 381 de las 5.829 respuestas, en cinco áreas. Las otras cinco no cambian.

  | Área | Error → valor | Valor → error | Valor distinto |
  |---|---|---|---|
  | arithmetic | 68 | 0 | 28 |
  | types | 94 | 92 | 28 |
  | fhir | 33 | 2 | 13 |
  | membership | 0 | 0 | 22 |
  | fhirmembership | 0 | 0 | 1 |

  Contra las divergencias registradas, en una copia del repo:
  - se van unas 260, sobre todo `reference defect` (arithmetic 60, types 142, fhir 35, membership 22, el defecto de lista-lista con null; fhirmembership 1);
  - aparecen unas 40;
  - cambia la respuesta de Java en unas 220. En muchas cambia solo el texto del mensaje de error, que la fila registra.

  Las cifras definitivas las escribe el comando del plan, no esta sonda.
- **cql-execution no cambia:** las 5.234 respuestas son idénticas, con el ELM de la 5.4.0 y cql-exec-fhir 2.2.0, y el conjunto de definiciones que responde es el mismo.
- **Los nodos del traductor no cambian:** `fhirmembership.nodes.tsv` sale idéntico.
- **`fold` no lee el ELM de la 5.4.0:** se niega a escribir, con «1 definition(s) have no type or no locator: Patient». La 5.x cambió la forma del JSON del ELM: anotaciones, discriminadores `type`, arreglos vacíos.
- **Mensajes de error con un hash de objeto:** en la 5.4.0, algunos mensajes traen la identidad del objeto, como `Cannot cast null to org.hl7.elm.r1.NamedTypeSpecifier@21e57990`. Ese texto cambia en cada corrida, y una fila de divergencia que lo registra no se reproduce.

## Alcance

1. **El arnés Java en la 5.4.0** (`internal/referenceharness`):
   - `pom.xml` pasa a `org.cqframework` 5.4.0;
   - `Translate.java` y `Evaluate.java` portados, con el mismo formato de salida, campo por campo;
   - `pinnedReferenceEngine` lee `org.cqframework:engine-jvm`.
2. **El arnés JS:** cql-exec-fhir 2.2.0 en `package.json` y `package-lock.json`.
3. **`fold`** lee el ELM de la 5.x. Tiene que dar el mismo pliegue donde el traductor decidió lo mismo.
4. **Los mensajes con hash:**
   - el arnés reemplaza solo la identidad de un objeto Java tras un nombre de clase calificado, `(?:[a-z][a-z0-9]*\.)+[A-Z]\w*@[0-9a-f]{1,8}\b`, por `Clase@…`. Una fecha CQL (`@2020-01-01`) no es un hash y queda intacta;
   - es una normalización del registro, no de la respuesta: la clase del error y la respuesta no cambian;
   - se documenta en el README del arnés;
   - un test de Go sobre las referencias versionadas exige que ningún mensaje de error quede vacío, que no quede ninguna identidad sin normalizar y que las fechas `@AAAA-` de los mensajes sigan ahí;
   - se redacta el issue para cqframework, con la reproducción en la 5.4.0, y se publica solo con el visto bueno del usuario.
5. **Regenerar todas las referencias:**
   - las 10 áreas: `*.java.json`, `*.js.json` y `*.nodes.tsv`;
   - `testdata/reference` (`probe.expected.json`, `runtime.expected*.json`);
   - `testdata/membershipmodel` y `testdata/membership`;
   - el ELM de FHIRHelpers que usa JS.
6. **Reclasificar:**
   - `-update-divergences` y los `-classify-<área>` reescriben cada área;
   - cada divergencia nueva tiene la clase que le da el oráculo, con la evidencia de la referencia (su respuesta en la 5.4.0);
   - se borran las que se fueron, y no se reclasifica a mano;
   - las listas de los ADR que escriben los clasificadores se regeneran;
   - si una divergencia nueva revela un defecto propio, va al backlog: no se corrige en este incremento.
7. **Lo que citan los tests y el código:**
   - los casos escritos a mano que nombran la respuesta de Java 3.26.0 se revisan contra la 5.4.0, por ejemplo `membershipModelDiffers` (D7 a D9, el defecto de lista-lista que la 5.4.0 corrige), `engine_code_fields_test.go` y `engine_calendar_bounds_test.go`;
   - cada comentario de producción que cita «Java engine 3.26.0» se vuelve a medir: se corrige el texto, o se marca la afirmación como de la 3.26.0 si ya no vale;
   - los planes cerrados no se tocan: son registro.
8. **El codificador del arnés, con control propio:**
   - `toHapi` conserva las extensiones de un primitivo y de un elemento, y falla si encuentra un hijo que no sabe reconstruir: el arnés sale con error, y nunca escribe `Unencoded` por un `ClassInstance`;
   - el tipo FHIR que se registra es el que da la 5.4.0 (`FHIR.AdministrativeGender`, no `FHIR.code`), para que el archivo diga lo que respondió la referencia;
   - un `runtime.ValueSet` se codifica (`id`, `version`, `codesystems`), y el lector de Go lo acepta;
   - una prueba de ida y vuelta en Java, `toHapi(toCqlValue(r))` igual a `r`, sobre `fhir-data.json` y sobre `hl7.fhir.r4.examples` 4.0.1, con el SHA-256 del package fijado en el repo;
   - `encoder.cql`, en `testdata/reference`, con una definición por rama del codificador: Code, Concept, Tuple, Quantity, cada precisión de DateTime, Date y Time, un intervalo, una lista, un primitivo FHIR con extensión y un recurso. El valor esperado de cada una está escrito a mano desde la especificación, y un test de Go lo compara con la salida versionada de Java.
9. **Ningún veredicto se pierde con la referencia:**
   - `TestThe<Área>EngineFollowsTheOracle`, en cada área con oráculo, compara este motor con el oráculo en todo el corpus, sin pasar por las referencias;
   - los `own defect` conocidos quedan en un archivo propio del área, `*.own-defects.tsv`, con su backlog;
   - así, una fila `own defect` que se va porque Java cambió su respuesta (fhir D240: Java pasa a null, como este motor, y el oráculo dice `FHIR.AdministrativeGender male`) sigue registrada.
10. **El inventario completo de lo que cita la 3.26.0:**
    - `engine_runtime_reference_test.go:72`;
    - `fold/main.go:53`: `fold` lee `translatorVersion` del `CqlToElmInfo` del ELM, en lugar de un texto libre por defecto;
    - `scripts/mutations/002-adr.sh` (224 y 247) y `004-code-fields.sh:60`, cuyas anclas cambian;
    - el ADR 001 (líneas 16 y 38), que se enmienda;
    - el `README.md` del repo y el del arnés;
    - `fhir.divergences.md` y `testdata/reference/*.divergences.md`;
    - `conformance_test.go:169`, `eval/context.go:381`, `eval/evaluator.go:213` y `funcs/type_ops.go:233`;
    - `notOracle` y `decidedRuntimeAnswers` (`engine_runtime_reference_test.go`), que saltan comparaciones: cada entrada se revisa contra la 5.4.0.
11. **Un comando que mide la deriva:** `differential/drift`, que compara dos juegos de referencias definición por definición y clasifica cada cambio (error → valor, valor → error, valor distinto, solo el mensaje). La tabla de la revisión previa sale de él, con fecha, a `docs/evidence/`. Cuenta también:
    - las definiciones que aparecen o desaparecen;
    - por clase, las filas que pierden su veredicto, nuevas, cambiadas o idas, con el veredicto anterior al lado;
    - las alrededor de 320 filas en las que solo cambia el mensaje, que el clasificador vuelve a dar.
12. **Una regeneración reproducible:** `make references`, que hoy sale con error, corre el arnés con Docker dos veces, en directorios separados. Compara las dos corridas byte a byte y los conjuntos de definiciones, y solo entonces copia a `testdata`. Las mutaciones del arnés Java, que necesitan regenerar, corren en ese target y no en `make mutations`, con una variable que lo exige, como `CQL_MUTATIONS_NETWORK`. No se saltan en silencio.

## Fuera del alcance

- `gofhir/fhirpath` (v1.14.1 → v1.16.6) y `shopspring/decimal` (v1.4.0 → v1.6.0): son dependencias del motor y cambian sus respuestas. Van en incrementos propios, después de este.
- Corregir defectos propios que la 5.4.0 destape: van al backlog, cada uno con su fila `own defect`.
- R4B y R5: el arnés usa FHIRHelpers 4.0.1 y el modelo R4, como hoy. No cambia con esta versión.
- Cambiar los generadores o los corpus.

## Criterio de salida

- `TestTheReferencesAreOfTheirCorpus` exige Java 5.4.0 y cql-execution 3.3.2 con cql-exec-fhir 2.2.0, y todas las referencias lo cumplen.
- `TestAnswersMatchTheReferences` en verde: ninguna divergencia `undecided` ni sin clasificar en ningún área.
- `TestStaticConversionsMatchTheReference` y los tests del ELM (`membershipmodel`, nodos) en verde con el ELM de la 5.4.0.
- Dos corridas seguidas del arnés dan archivos idénticos, byte a byte. Se verifica con el target de referencias, no en `make check`, porque usa Docker.
- `make check` en verde, y `make evidence` con el reporte de deriva enlazado.

## Tests y mutaciones

- **El formato del port**, en las diez áreas:
  - el `sourceSha256` es idéntico, porque el corpus no cambió;
  - los `*.js.json` son idénticos, salvo `engine`;
  - los `*.nodes.tsv` son idénticos.

  En `integer`, `scope`, `codes`, `codesets` y `division`, los `*.java.json` son idénticos salvo `engine` y los mensajes. Este control no cubre FHIR ni las fechas, porque esas áreas no tienen valores de esos tipos. Esas ramas las controlan `encoder.cql` y la ida y vuelta.
- **Mutaciones nuevas** (en `scripts/mutations/007-references.sh`), cada una con el test que debe fallar:
  - del arnés Java, que se regeneran en `make references` y se comparan byte a byte con lo versionado, porque el comparador de respuestas no ve el tipo FHIR ni distingue Integer de Long (backlog nuevo):
    - `toHapi` pierde el valor de un primitivo, pierde una extensión o cambia el nombre del tipo FHIR;
    - `encode` lee un `runtime.Integer` como Long;
    - la normalización toca una fecha;
  - `encoder.cql` con un valor esperado cambiado → su test de Go;
  - un `own defect` borrado de `*.own-defects.tsv` → `TestThe<Área>EngineFollowsTheOracle`;
  - el pin lee `info.cqframework:engine` → `TestTheReferencesAreOfTheirCorpus` (reemplaza la mutación de `002-adr.sh`);
  - la normalización del hash borra el mensaje entero, o deja una identidad → el test de Go sobre los mensajes versionados;
  - `fold` deja de leer el locator → `TestStaticConversionsMatchTheReference`;
  - `drift` cuenta un cambio de mensaje como un valor distinto, ignora una definición agregada o ignora un cambio de mensaje → su propio test.
- **Las mutaciones de los incrementos anteriores** (001 a 006) siguen detectándose con las referencias nuevas. Si una deja de detectarse porque su caso dependía de una respuesta de la 3.26.0, se endurece.

## Excepciones a la regla 5

Ninguna. El arnés Java nombra `value`, `id`, `http://hl7.org/fhir` y `Patient`, como hoy, pero es código de test y no de producción.

## Dependencias

Ninguna nueva. Cambian de versión las tres referencias de la tabla; las transitivas, como Kotlin y kotlinx-io, las arrastra Maven.

## Riesgos

- **El port cambia lo que se registra sin que cambie la respuesta:** lo detecta el control de las cinco áreas estables.
- **`toHapi` reconstruye mal un valor:** lo detectan las mutaciones, y la comparación de las 22 respuestas FHIR que no cambian de versión.
- **Una divergencia que se va porque cambió la respuesta de Java, y no porque coincida con este motor:** `-update-divergences` la recalcula contra este motor. No se borra ninguna sin que la comparación lo diga.

## Ronda adversarial (Codex y Fable, 22-10-2026)

| Hallazgo | Verificación | Resolución |
|---|---|---|
| `toHapi` no tiene control real: pierde la extensión de un primitivo, cambia el tipo FHIR registrado, y su falla termina en `Unencoded` (alto, Codex y Fable) | Las cinco áreas estables no tienen ningún valor FHIR; D28 se registra como `FHIR.code` y el mensaje de la 5.4.0 dice `FHIR.AdministrativeGender` | **Aceptado:** alcance 8 (ida y vuelta, `encoder.cql`, falla en lugar de `Unencoded`, tipo de la 5.4.0) |
| Las mutaciones propuestas no ejercitan Java: los tests leen JSON versionado, y el comparador no ve el tipo FHIR ni distingue Integer de Long (alto, Codex) | La sonda de Codex: `Same=true` para `FHIR.string("a")` contra `FHIR.code("a")` y para `Integer(1)` contra `Long(1)` | **Aceptado:** las mutaciones del arnés regeneran en `make references` y se comparan byte a byte. La ceguera del comparador va al backlog: es anterior y no es de este incremento |
| Un `own defect` desaparece sin rastro cuando Java cambia su respuesta (alto, Fable) | fhir D240: Java 3.26.0 da `FHIR.code male` y la 5.4.0, null, como este motor; el oráculo dice `FHIR.AdministrativeGender male` | **Aceptado:** alcance 9 (motor contra oráculo en todo el corpus, `*.own-defects.tsv`) |
| La reclasificación automática no es evidencia independiente: el clasificador escribe y verifica con la misma función (alto, Codex) | `engine_fhir_oracle_test.go:1104-1166` | **Aceptado en parte:** `drift` registra el veredicto anterior, el nuevo y la respuesta de la 5.4.0 de cada fila, y el alcance 9 da un control independiente de la referencia. **Rechazado** pedir una reproducción mínima por fila: cada definición del corpus ya es una expresión sola, evaluada en su propia librería, y su respuesta en la 5.4.0 es la evidencia que pide la carta («un valor que la referencia sí produjo») |
| La expresión regular del hash rompe las fechas CQL (medio, Codex y Fable) | Los mensajes de la 5.4.0 traen `@2020-01-01`; los 180 hashes son de `NamedTypeSpecifier` (126), `ListTypeSpecifier` (34) e `IntervalTypeSpecifier` (20) | **Aceptado:** expresión anclada en un nombre de clase calificado, y un test de Go sobre los mensajes versionados |
| El control de las áreas estables es flojo: exime `sourceSha256` sin razón y no mira JS ni los nodos (medio, Codex y Fable) | `sourceSha256` es idéntico en las diez áreas; JS y los nodos, también | **Aceptado:** control de las diez áreas, en «Tests y mutaciones» |
| Faltaban sitios que citan la 3.26.0, y `notOracle` y `decidedRuntimeAnswers` (medio, Codex y Fable) | grep del repo | **Aceptado:** alcance 10 |
| La regeneración no es reproducible y la sonda de deriva ignora definiciones agregadas y cambios de mensaje (medio, Codex) | `Makefile:39`, `drift.py` | **Aceptado:** alcances 11 y 12 |
| Un `runtime.ValueSet` queda `Unencoded` (bajo, Fable) | fhir D345 y D788 | **Aceptado:** alcance 8 |
| La regla 5 en el arnés (bajo, Fable) | El arnés nombra `value`, `id` y `Patient` | **Aceptado:** se dice en «Excepciones a la regla 5» |

## Aprobación (22-10-2026)

El usuario aprobó el plan ajustado, en dos PR.
- **PR 1, preparación sobre la 3.26.0:** los controles que no dependen de la versión:
  - motor contra oráculo en todo el corpus, con `*.own-defects.tsv`;
  - `encoder.cql`, con los valores esperados escritos a mano;
  - `drift`, con sus tests y mutaciones;
  - el test de los mensajes versionados;
  - `make references` reproducible.
- **PR 2, el cambio de versión:** el port, el pom, la normalización, `toHapi` y su ida y vuelta, regenerar, reclasificar, el inventario y el ADR 001.

El reparto se ajustó al empezar. `TestTheReferencesAreOfTheirCorpus` exige que las referencias nombren la versión del pom, así que el port no puede entrar sin regenerar. Los controles entran primero y vigilan el cambio: `encoder.cql` tiene que dar lo mismo con las dos versiones.

## Implementación del PR 1 (22-10-2026, sin commit)

- **Motor contra oráculo** (`engine_own_defects_test.go`): `TestTheEngineFollowsTheOracles`, en types, fhir, arithmetic, codes, codesets, membership y fhirmembership, con un `*.own-defects.tsv` por área.
  - Contiene todas las filas `own defect` de los manifiestos, y 3 más de fhir (D19, D54 y D794): las que oculta `fhirHiddenByTheComparison`.
  - `referenceDivergences` y el test nuevo leen las respuestas del motor con la misma función, `snapshotEngineAnswers`.
- **El codificador** (`testdata/reference/encoder.cql`, `TestTheReferenceEncoderWritesEachValueAsItIs`): 25 definiciones, una por rama, con el valor esperado escrito a mano. El Patient esperado sale de `fhir-data.json`.
  - El port de sonda a la 5.4.0 da la misma salida, byte a byte.
  - `Patient.gender` quedó fuera: su tipo depende de la versión (HAPI lo tipa `FHIR.code` y la 5.4.0, `FHIR.AdministrativeGender`), no del codificador. Lo cubre el área fhir.
- **Los mensajes** (`TestTheReferenceMessagesAreOfTheAnswer`): ningún error sin mensaje y ninguna identidad de objeto Java en las referencias versionadas. Sobre la salida de la sonda 5.4.0, el patrón encuentra 170 identidades en types, y ninguna fecha CQL.
- **`drift`** (`internal/referenceharness/differential/drift`), con sus tests. Sobre la sonda reproduce la revisión previa: 320 cambios solo de mensaje, 195 errores que pasan a valor, 94 valores que pasan a error y 92 valores distintos.
- **`make references`** (`scripts/references.sh`): con la 3.26.0, las dos corridas son idénticas en los 29 archivos, y 28 coinciden byte a byte con lo versionado.
  - El que no coincidía, `runtime.expected.js.json`, estaba sin regenerar desde #157: `evaluate.js` escribe `Text` para un string sin tipo de CQL, y el archivo seguía diciendo `String`.
  - Regenerado, `TestRuntimeMatchesTheReference` fallaba, porque su comparador no leía `Text`. Se agregó el caso, que lee el texto como lo hace `answers.go`. Pasa con el archivo viejo y con el nuevo.
- **Mutaciones** (`scripts/mutations/007-references.sh`): 11, todas detectadas.
  - La de Docker (`CQL_MUTATIONS_DOCKER=1`, una marca de tiempo en `Evaluate.java`) sobrevivió en la primera corrida sin mostrar por qué. A mano, en una copia, sí se detecta.
  - El runner ahora muestra la cola de la salida de una mutación que sobrevive. Se volvió a correr dos veces.

## Review del PR 1, ronda 1 (Codex y Fable, 22-10-2026)

Cada hallazgo se verificó en el código o con una sonda antes de aceptarlo.

| Hallazgo (severidad, quién) | Verificación | Resolución |
|---|---|---|
| `references.sh check` lista los archivos que difieren de lo versionado, pero sale con 0 (medio, Codex y Fable) | Leído: el modo `check` nunca falla por eso | **Aceptado:** sale con 1. Nueva mutación con Docker: una referencia que el arnés ya no escribe |
| `drift` cuenta Translation → CqlException como «error → valor» (medio, Codex y Fable) | Un caso nuevo de `TestChangeNamesEachKind` falla | **Aceptado:** «refused → accepted», con su mutación |
| `drift` no lista cada fila con su veredicto anterior, el nuevo y la respuesta de la 5.4.0, como prometía la ronda adversarial (medio, Fable) | Leído: solo escribía totales | **Aceptado:** tabla por fila (`rowChanges`, `writeRows`), con dos tests y una mutación |
| El oráculo de aritmética lee `arithmetic.answers.tsv`, no el motor (bajo, Codex y Fable) | Codex: con `Add` cambiado por `Sub` el test seguía verde | **Aceptado:** lee `snapshotAnswers`. Nueva mutación: la suma decimal como resta |
| Un error cuyo mensaje no es texto escapa al test de mensajes (bajo, Codex) | Sonda de Codex | **Aceptado:** se exige un texto no vacío, con su mutación |
| `-update-own-defects` fija las respuestas con `pinned`, pero la lectura compara sin fijar (bajo, Fable) | Sonda de Fable con una barra invertida | **Aceptado:** se compara lo fijado. Con `-update-own-defects`, los 7 archivos se reescriben idénticos |
| La expresión regular de identidades no reconoce `Outer$Inner@…` ni `[L…;@…`, y lee `x.DateTime@2020-01-01` como identidad (bajo, Fable) | `TestJavaIdentityReadsOnlyAnIdentity`, escrito a mano | **Aceptado** |
| `encoder.cql` no tiene el primitivo FHIR con extensión que prometía el alcance 8 (bajo, Fable) | Con la 3.26.0, `FHIR.string { value, extension }` falla en el motor: `ArrayList cannot be cast to String`. La 5.4.0 da exactamente el valor escrito a mano (D25 y D26 en el scratchpad) | **Pasa al PR 2:** el PR 1 está fijado en la 3.26.0, que no puede evaluarlo. El defecto ya está corregido en la 5.4.0 y no se reporta |
| Encontrado al corregir: `own-defects` comparaba la respuesta del motor como texto, y `Exp` escribe otro último dígito en amd64 que en arm64 (D427, D699, D780…) | Memoria «Float por plataforma» | **Corregido:** `sameDefect` usa la tolerancia de `snapshotSame` en las áreas no exactas. Mutación: un defecto registrado con otra respuesta fuera de la tolerancia |

Verificado sin hallazgos: la cobertura del oráculo (types 898 de 900, por diseño; el resto, 100 %); que el caso `Text` no debilita la comparación; los 29 archivos de `references.sh`; el estilo y el lint.

## Review del PR 1, ronda 2 (Codex y Fable, 22-10-2026): converge

Ningún hallazgo de severidad media o mayor. Las correcciones de la ronda 1 se verificaron con sondas:
- la tolerancia acepta el último dígito y rechaza un cambio en el dígito 11 de D427;
- `-update-own-defects` reescribe los 7 archivos idénticos;
- la falla de la 3.26.0 con el primitivo con extensión se reprodujo en Docker.

Hallazgos bajos, todos corregidos:

| Hallazgo (quién) | Resolución |
|---|---|
| `writeRows` no escapa `\|` en la tabla de Markdown (Codex) | Se escapa. `TestRunWritesEachRowWithTheAnswerNow` lo cubre con una respuesta que lo contiene |
| `pinned` se aplicaba dos veces en cinco áreas, y no es idempotente (Fable) | Cada buscador entrega la respuesta del motor ya fijada, y la comparación no vuelve a fijar. Los 7 archivos se reescriben idénticos |
| `references.sh` no ve una referencia del repo que el arnés ya no escribe, y la mutación con ese nombre probaba otra cosa (Fable) | Recorre también lo versionado: `not written by the harness`. La mutación vieja se renombró, y otra nueva saca un área de la lista |
| El README no decía que `make references` falla cuando hay diferencias (Fable) | Corregido |

## Agregado al PR 2: el entorno de las referencias al día (22-10-2026)

Lo pidió el usuario: siempre las últimas versiones. Se midió el 22-10-2026 con `npm view`, `maven-metadata.xml`, nodejs.org, Adoptium y Docker Hub:
- las librerías comparadas ya apuntan a lo último: org.cqframework 5.4.0, cql-execution 3.3.2 y cql-exec-fhir 2.2.0;
- el entorno no:

| Pieza | Hoy | En el PR 2 |
|---|---|---|
| Imagen del arnés JS | `node:20`, sin soporte desde el 30-04-2026 | `node:24.22.1` (última LTS; la 26 es *Current*) |
| Imagen del arnés Java | `maven:3-eclipse-temurin-21` | `maven:3.10.0-eclipse-temurin-25` (última LTS; la 27 no es LTS) |
| `slf4j-nop` | 2.0.13 | 2.0.20 (última estable; la 2.1.0 es alpha) |

- **Versión exacta en las imágenes:** una etiqueta flotante cambia por debajo, y las dos corridas de `references.sh` no lo verían, porque las dos usan la misma imagen.
- **La deriva de cada cambio se mide aparte**, con `drift`: primero el entorno sobre la 3.26.0 y después el salto a la 5.4.0. Así no se mezclan.

**Cierre del review del PR 1 (22-10-2026):**
- `scripts/mutations/007-references.sh` tiene 19 mutaciones: 16 sin Docker y 3 con Docker.
- Las 19 se detectan. En la corrida completa, «a reference the harness no longer writes» falló por la red: Maven cortó la descarga de un jar de HAPI. Repetida a mano en una copia, se detecta (`not written by the harness`, salida 1).
- Las 257 mutaciones de 001 a 006 se detectan todas.
- `make check` pasa: 1027 s y 1,26 GB de RSS.

