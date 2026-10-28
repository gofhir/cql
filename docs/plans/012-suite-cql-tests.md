# 012: la suite de conformance al día (backlog 1c24)

Estado: paso 8 terminado, el review converge en la ronda 3.

## Por qué

`conformance/testdata/SOURCE.tsv` fija cqframework/cql-tests en 727219f, 56 commits detrás de main (f724ee4, 19-10-2026 en −0600, el 20-10 en UTC). Los 15 archivos cambiaron: 1 292 líneas agregadas y 82 quitadas. Hasta hoy `TestConformance` exige que pase cada test de la suite, salvo las tres contradicciones de `upstreamContradictions`. Con la suite atrasada, eso promete menos de lo que parece.

## Revisión previa (28-10-2026)

**gofhir/server:** no aplica. Consume cql como librería y no tiene suite propia de CQL.

**Medición.** Copié los 15 archivos de f724ee4 sobre un worktree de main (2dc918f) y corrí `go test ./conformance -run 'TestConformance$'`: 1 978 pasan y 59 fallan (2,5 s, 512 MB de RSS). Cada caso que falla se corrió solo, en un archivo propio, contra Java 5.4.0 y cql-execution 3.3.2 con el arnés de `internal/referenceharness`. La sonda y sus respuestas están en el scratchpad (`q012/p`); la evidencia que quede va al repo en el paso 7.

**Versiones de los tests.** Cada test declara la versión de CQL que prueba. En f724ee4 hay 1 428 de la 1.0, 35 de la 1.3, 365 de la 1.4, 182 de la 1.5 y 27 de la 2.0. El motor se programa desde CQL 1.5.3 (ADR 001); `MatchesFull`, `~in` y `~contains` no están en la referencia de la 1.5.3.

**cql-tests#153** sigue abierto, igual que #145. Los tests de Decimal en que se apoya el 011 (DecimalTenthStep, Decimal10Pow28 y sus signos) no cambiaron en f724ee4. La decisión del 011 se mantiene.

### Las 59 fallas, por clase

Las citas son de CQL 1.5.3, Appendix B (`https://cql.hl7.org/N1A/09-b-cqlreference.html`).

| Grupo | Tests | Motor | Java 5.4.0 | Suite | Especificación |
|---|---|---|---|---|---|
| G0. Contradicción resuelta arriba | FloorIntegerGreaterThanMaxInteger, FloorIntegerLessThanMinInteger | error | error | ahora `invalid="semantic"` | — |
| G1. CQL 2.0, fuera de la 1.5.3 | MatchesFull (3), EquivalentIn (4), EquivalentContains (5), Predecessor/SuccessorOf…DPrecision y …QCMPrecision (5) | error, o el paso de 10^-8 | error de traducción, o 10^-8 | lo nuevo de 2.0 | no está en 1.5.3; sus hermanos `versionTo="1.5.3"` (PredecessorOf1D y otros) pasan |
| G2. Error donde va null | DivideQuantityByZero, Exp1000, Exp1000D, Ln0, LnNeg0, PredecessorUnderflowDt, PredecessorUnderflowT, SuccessorOverflowDt, SuccessorOverflowT | error | null | null (ocho antes inválidos; DivideQuantityByZero es nuevo) | «the result is null» (Exp, Ln, `/`); «a null is returned» (Predecessor, Successor). Exp y Ln ya son defectos propios en `arithmetic.own-defects.tsv` (D17, D114, D115, D212) |
| G3a. Componente ausente | DateTimeComponentFromMonthIsNull (`month from DateTime(2012)`) | 0 | null | null | DateTime Component From: «is not specified to the level of precision being extracted, the result is null» |
| G3b. Años entre fechas bisiestas | YearsBetweenLeapYearDatesEquals2, …DateTimesEquals2 | 1 | 2 | 2 | Duration Between: «whole calendar periods»; Author's Guide, Date and Time Arithmetic: si el día no existe en el año resultante, «the last day of the calendar month is used» |
| G3c. Lista null | DistinctNull, FlattenNull | `{}` | null | null | «If the argument is null, the result is null» |
| G3d. Skip negativo | SkipNegative | `{1, 2, 3}` | error | `{}` | «less than zero, the result is an empty list» |
| G3e. Coalesce de seis argumentos | CoalesceMoreThanFiveArgs | evalúa | error de traducción | inválido | firmas de 2 a 5 argumentos |
| G3f. Funciones que faltan | ToDate (6), ToRatio (2) | «unknown function» | responde | responde | están en 1.5.3; ToDate «will ignore the time portions», y en ToDateIgnoresTime las dos referencias dan null |
| G3g. `div` de Quantity | TruncatedDivide…DQuantity (3) | unidad del operando | unidad del operando | `'1'` | Divide: «quantities with the same units … the default UCUM unit ('1')»; Truncated Divide: «the appropriate unit» (se lee como Divide) |
| G3h. Tupla con un null | TupleEqDifferentNamesWithOneNullId, TupleNotEq… | null | null | false / true | «a conjunction of equality comparisons»; ejemplo TupleEqualMixedNullFalse |
| G3i. Límite null cerrado | TestMaxIntervalEnds/Starts/Overlaps/OverlapsAfter/OverlapsBefore, TestIntegerInMaxIntervalTrue, TestPointFromMaxIntervalError, TestUnionMaxInterval | null o false | null o false | el intervalo máximo | Start/End: un límite cerrado en null es «the minimum/maximum value of the point type»; Point From: «If the argument is not a unit interval, a run-time error is thrown» |
| G3j. Límite null abierto | TestIntegerInUnknownIntervalNull, TestMeetsAfterNull | false | false | null | Start/End: «if the low value of the interval is null, the result is null» |
| G5. La suite contra la especificación | NotEquivalentNullNull | false | error de traducción | `invalid="semantic"` | ejemplo NullNotEquivalentIsFalse: `null !~ null` es false |

Suman 2 + 17 + 9 + 30 + 1 = 59. Las 39 de G2 y G3 son defectos propios; en G3f (ToDateIgnoresTime) y de G3g a G3j, Java también contradice a la especificación. En G5 el motor cumple la especificación y la suite no.

## Decisiones

Del usuario, el 28-10-2026:

- **Alcance:** traer la suite y clasificar. El 012 no cambia la semántica; cada grupo de G2 a G4 va al backlog como su propio incremento (G4 se disolvió en G3 en la ronda adversarial).
- **CQL 2.0:** los tests que fallan entran en la lista por nombre, como las demás fallas. Los de 2.0 que pasan, como Slice, se siguen exigiendo.

## Alcance

1. **La suite.** Los 15 archivos de cql-tests f724ee4 reemplazan a los de 727219f, y `SOURCE.tsv` registra el commit nuevo con el blob SHA-1 y el SHA-256 de cada uno. `make conformance-provenance` los compara con el árbol de cql-tests en ese commit.
2. **Las contradicciones resueltas.** FloorIntegerGreaterThanMaxInteger y FloorIntegerLessThanMinInteger salen de `upstreamContradictions`: la suite ahora los marca inválidos, el motor da error, y el test exige que la entrada salga. DateTimeDurationBetweenUncertainInterval sigue ahí: f724ee4 no lo cambió, y sigue fallando.
3. **Las fallas conocidas.** Un mapa `knownFailures` en `conformance/known_failures_test.go`, del nombre del test a su grupo (G1, G2, G3a a G3j, G5) y su ítem del backlog. Funciona como `upstreamContradictions`: el test se corre, tiene que seguir fallando, y si pasa, la prueba falla pidiendo quitar la entrada. Además:
   - cada entrada dice cómo falla, con un valor o con un error, y la prueba lo exige: un test inválido que empieza a dar error por otra causa no se lee como resuelto (Fable 2);
   - cada entrada fija lo que la suite pide: versión, `versionTo`, `invalid` y salidas esperadas. Si la suite cambia un test, la entrada falla y hay que reclasificarla (Codex 7);
   - una entrada de G1 exige que el test sea de una versión posterior a la 1.5.3, y una de otro grupo, que su rango (`version` a `versionTo`) contenga la 1.5.3 (Fable 5);
   - un nombre de la lista que no está en la suite es un error, para que una entrada no sobreviva a su test.
4. **La evidencia de las referencias.** Un área de snapshots nueva, `suitegaps`, con una definición por cada falla conocida: la expresión de la suite tal cual. Sus `java.json` y `js.json` salen de `sh scripts/references.sh write`; su `answers.tsv` guarda la respuesta actual del motor, y su `divergences.tsv` clasifica cada diferencia con Java y con cql-execution. Así, si una falla conocida cambia de respuesta sin empezar a pasar, el snapshot lo ve. Cada definición del área lleva el nombre de su test en un comentario de la misma línea, y un test exige que la lista y el área tengan los mismos tests, con la misma expresión que la suite. Las filas de Exp y Ln citan las de `arithmetic.own-defects.tsv` en vez de decidirlas de nuevo (Fable 4).
5. **El modelo XML.** `xmlmodel.go` lee `versionTo`.
6. **Lo que el 012 deja escrito y no arregla.** El comentario de `sema_test.go` («the 39 expressions it marks invalid», con Exp(1000) y Ln(0) como ejemplos) queda al día con la suite nueva (Fable 3). Un test ata la lista `areas` de `scripts/references.sh` al manifiesto de snapshots (Codex 7, Fable 8).
7. **El backlog.** Un ítem por grupo de G2 y G3 en `docs/plans/README.md`, cada uno con sus tests y la cita de la especificación. G1 queda como «CQL 2.0, fuera del objetivo (ADR 001)». Para G5 se redacta un issue para cql-tests, que se publica solo con aprobación y después de buscar duplicados.

**Fuera:** arreglar cualquiera de las fallas; exigir que un test `semantic` se rechace al compilar; distinguir Long de Integer en las salidas; actualizar el objetivo a CQL 2.0.

**Dependencias nuevas:** ninguna.

## Diferencias deliberadas

Con Java, en `testdata/snapshots/suitegaps.divergences.tsv`: D36 y D37 (`~contains`) y D46, D47 y D48 (`MatchesFull`). Son tests de CQL 2.0: el motor los rechaza, como Java, pero con otro error. Todo lo demás del área es un defecto propio o de una referencia, con su cita.

## Criterio de salida

- `TestConformance` sobre f724ee4: pasa todo test que no esté en `knownFailures` ni en `upstreamContradictions`, y falla todo test que sí esté.
- `knownFailures` tiene 57 entradas, cada una con grupo, forma de la falla, lo que pide la suite e ítem del backlog; `suitegaps` tiene las mismas 57, con sus divergencias clasificadas.
- `sh scripts/references.sh check`: dos corridas idénticas, ningún archivo distinto.
- La diferencial contra main (semilla 8008) es idéntica: el motor no cambia.
- `make check` en verde, con las mutaciones del 012 atrapadas.

## Tests

- `TestConformance` con la suite nueva y la lista.
- `TestTheConformanceSuiteIsTheVendoredOne`, sobre el `SOURCE.tsv` nuevo.
- Un test que ata `knownFailures`, la suite y `suitegaps`: los mismos nombres y la misma expresión.
- El snapshot de `suitegaps` contra sus dos referencias.

## Mutaciones previstas

- Quitar una entrada de `knownFailures`: el test vuelve a fallar.
- Agregar a la lista un test que pasa: la prueba pide quitarlo.
- Un nombre que no está en la suite.
- Una entrada de G1 sobre un test que no es 2.0.
- Una definición de `suitegaps` con otra expresión que la de la suite, o una que falta.
- Un byte cambiado en un XML, o un SHA de `SOURCE.tsv` corrupto.
- Volver a poner las contradicciones de Floor.
- Una respuesta del motor que cambia en una falla conocida (la ve el snapshot).
- Una falla conocida que cambia de forma: de un valor a un error sin relación.
- La suite que cambia la salida esperada, `invalid` o la versión de un test de la lista.
- `suitegaps` fuera de la lista `areas` de `scripts/references.sh`.
- Un test de G1 con `versionTo` que llega a la 1.5.3.

## Ronda adversarial (28-10-2026)

Fable y Codex, en paralelo, cada uno sobre su copia. Cada hallazgo se verificó contra la especificación o la sonda antes de decidir.

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| Codex 1, Fable 1 | NotEquivalentNullNull no es un defecto propio: la 1.5.3 da `null !~ null` como false | alta | Aceptado: G5, la suite contra la especificación. Issue redactado para cql-tests |
| Codex 2, Fable 7 | G4b y G4c se pueden decidir ya | alta | Aceptado: la especificación decide; pasan a G3g y G3h, defectos propios y de Java |
| Codex 3, Fable 6 | G4a mezcla límites cerrados y abiertos; PointFrom mal descrito; la cuenta no daba | media | Aceptado: G3i (cerrados, 8) y G3j (abiertos, 2). El motor da null en PointFrom, como Java |
| Codex 4 | En ToDateIgnoresTime las dos referencias dan null, y la especificación dice ignorar la hora | alta | Aceptado: se registra en G3f como defecto de las dos referencias |
| Codex 5 | Un Long esperado se lee como Integer: los tests nuevos de Long pasan solo por valor | media | Rechazado para el 012, al backlog: es anterior, ya está escrito en `valuesEqual`, y es del mismo tipo que 1c18 |
| Codex 6, Fable 2 | Un test inválido pasa con cualquier error; en la lista, un error sin relación se leería como resuelto | media | Aceptado para la lista: cada entrada fija la forma de la falla. Para los tests que pasan, al backlog: Codex no encontró ningún inválido que pase por un error sin relación |
| Fable 3 | Los `semantic` se rechazan al evaluar, no al compilar; el comentario de `sema_test.go` quedó viejo | media | Aceptado el comentario. Exigir el rechazo estático va al backlog, con el ítem «el control va en sema» de 1c23 |
| Codex 7, Fable 8 | Atar también versión, `invalid` y salidas; `areas` de `references.sh` no se controla | media | Aceptados los dos |
| Fable 4 | G2 duplica filas de `arithmetic.own-defects.tsv` | media | Aceptado: se citan |
| Fable 5 | El modelo ignora `versionTo`; DateTimeComponentFromTimezoneOffset (`versionTo="1.3"`, `timezone from`) pasa porque el motor guarda la grafía anterior a la 1.4 | media | Aceptado `versionTo`. Si `timezone from` se queda es una decisión aparte: al backlog, con la nota en ADR 001 pendiente |
| Fable 9 | `valuesEqual` iguala `1 'm'` con `100 'cm'` e ignora si un límite null es abierto o cerrado | baja | Al backlog: ningún test que pasa depende de eso hoy, según la sonda de Fable; importa para G3i cuando se arregle |
| Fable 10 | Faltan mutaciones: grupo que no coincide con la falla, inválido resuelto por otro error, `versionTo` | baja | Aceptado en parte: la falla que cambia de forma y `versionTo` están en las mutaciones. El grupo no: es la clasificación del plan, y ningún test puede verla (corregido en el review, ronda 1) |

## Implementación (28-10-2026)

Aprobado por el usuario el 28-10. Lo que quedó:

- **La suite:** los 15 archivos de f724ee4 y `SOURCE.tsv` con su commit, blob SHA-1 y SHA-256.
- **`knownFailures`** en `conformance/known_failures_test.go`: 57 entradas (17 de G1, 9 de G2, 30 de G3a a G3j, 1 de G5), cada una con grupo, `errs` y `suite`. Las dos de Floor salen de `upstreamContradictions`.
- **Tests nuevos:** `TestEveryKnownFailureIsOfTheSuite` ata la lista, la suite y `suitegaps.cql`; `TestTheVersionsOfATestAreARange` prueba el rango con tests de la suite; `TestEveryAreaIsRunByTheReferences` ata `areas` de `references.sh` al manifiesto.
- **El área `suitegaps`:** 57 definiciones generadas de la suite, con sus referencias (`sh scripts/references.sh write`, dos corridas idénticas). Divergencias: 58 defectos propios, 4 de las referencias (D5, D7 y D10 de cql-execution en la precisión de Predecessor y Successor; D19 de Java en G5) y 5 deliberadas.
- **`sema_test.go`:** el comentario, medido con una sonda sobre los 38 inválidos: el motor rechaza 1 por sintaxis y 34 al evaluar, y evalúa 3 (los de la lista).
- **Medido de paso:** DateTimeComponentFromTimezoneOffset hereda la versión 1.4 de su grupo y declara `versionTo="1.3"`, un rango vacío.

## Mutaciones

`sh scripts/mutations/012-suite.sh`, 17 mutaciones en la implementación y 19 después del review, todas atrapadas (34 s): una falla conocida fuera de la lista; un test que pasa dentro de ella; un nombre que la suite no tiene; un grupo que no existe; un test de 1.0 en G1; una falla que cambia de forma; la suite que cambia lo que pide; el área con otra expresión, o sin la definición; la respuesta de una falla conocida que se mueve (`Skip`); un byte de un archivo de la suite; un SHA-256 del manifiesto; la contradicción de Floor que vuelve; el área fuera de `references.sh`; `versionTo` sin leer; el fin del rango ignorado; una deliberada que el plan no nombra. Las dos del review: un test cuyo rango terminó antes de la 1.5.3, en G1; y una falla con `errs` que empieza a pasar. La primera corrida dejó una sobreviviente aparente: el test falló, pero con otro nombre que el esperado; se corrigió el texto esperado.

## Diferencial

Contra main (2dc918f), semilla 8008, nueve áreas: idéntica. Fuera de los tests, el 012 solo cambia `conformance/xmlmodel.go`.

## Review, ronda 1 (28-10-2026)

Fable y Codex, cada uno sobre su copia, con mutaciones propias. Ninguno encontró un defecto del motor. Cada hallazgo se verificó antes de decidir.

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| Codex 1 | G1 aceptaba un test cuyo rango terminó antes de la 1.5.3 | media | Aceptado: G1 exige una versión posterior a la 1.5.3, y otro grupo, un rango que la contenga |
| Codex 2 | Quitar del runner el control de la forma o el de «pasa ahora» no hace fallar `go test` | media | Rechazado: lo atrapan las mutaciones «a known failure that fails another way» y «a test that passes put in the list», que corren en `make check`. Así se prueban los oráculos en este repo |
| Codex 3, Fable 2 | `suiteContract` ignora `versionTo` y el tipo de las salidas sin que nada lo vea: ninguna entrada los tiene | media | Aceptado: `TestTheContractHoldsWhatTheSuiteAsks`, con casos sintéticos para cada campo |
| Codex 4 | Sobreviven mutaciones de `compareVersions`, `covers` y la herencia de versión | media | Aceptado: `TestVersionsAreComparedByTheirNumbers` y `TestATestInheritsItsVersion`; la herencia pasa a `effectiveVersion`, porque ningún test de la suite hereda la versión de su archivo. Quitar las anclas de `suitegapsDefine` no cambia qué líneas lee en este archivo: no se agrega |
| Codex 5, Fable 1 | Un test en otro grupo no se detecta, y el plan decía lo contrario | media | Rechazado el control: el grupo es la clasificación del plan, y un predicado por grupo sería una heurística. Corregida la tabla de la ronda adversarial |
| Codex 6, Fable 9 | G3b se apoya solo en las referencias | media | Aceptado. En la ronda 2, Fable encontró la regla en el Author's Guide, y la evidencia la cita |
| Codex 7, Fable 3 | La cuenta «30 de G2 y G3» (son 39) y la ruta de `knownFailures` | baja | Corregidos el plan y `suitegaps.cql` |
| Fable 4 | G3a tiene cita en la especificación | baja | Aceptado: DateTime Component From, en el plan, el README y D20 |
| Fable 5 | D5, D7 y D10 citaban una frase que no está en la especificación | baja | Aceptado: «with a step size of 10-8» |
| Fable 6 | DivideQuantityByZero es nuevo, no un inválido que pasó a null | baja | Aceptado |
| Fable 7 | Fecha de f724ee4 distinta en el plan y el README | baja | Aceptado: 19-10-2026 en −0600 |
| Fable 8 | Filas donde la referencia también se equivoca y la nota no lo dice (D8, D14 a D16, D44) | baja | Aceptado |
| Fable 10 | Cada falla conocida se evalúa dos veces | baja | Aceptado: `runsClean` devuelve el error |

## Review, ronda 2 (28-10-2026)

Antes de la ronda, `make check` en verde: 420 mutaciones atrapadas (31 min). Codex no dejó hallazgos y confirmó los dos rechazos con el script de mutaciones: con el control de la forma o el de «pasa ahora» desactivado, `012-suite.sh` reporta una sobreviviente y falla.

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| Fable 1 | La regla nueva de G1 (Codex 1) no la ataba ningún test: volver a la anterior dejaba todo en verde | media | Aceptado: mutación «a test whose range ended before 1.5.3 in G1», con DateTimeComponentFromTimezoneOffset |
| Fable 2 | `max` → `min` en `compareVersions` sobrevive | baja | Aceptado: caso `1.5` contra `1.5.3`. `> 0` → `>= 0` en la regla de G1 no se puede atar: la suite no tiene tests de la versión 1.5.3 |
| Fable 4 | Una entrada con `errs: true` que empieza a pasar se reporta como «fails another way» y no como «passes now» | baja | Aceptado: el runner mira primero si pasa; mutación «a known failure with errs that passes» |
| Fable 5 | G3b sí tiene regla: Author's Guide, Date and Time Arithmetic | baja | Aceptado: D21, D22, el plan y el README la citan |
| Fable (b) | El CI no corre `make mutations`: el control de los rechazos es local | — | Es como el repo prueba sus oráculos (`make check` antes de cerrar); queda anotado |

## Review, ronda 3 (28-10-2026)

Antes de la ronda, `make check` en verde: 422 mutaciones atrapadas (35 min). Converge: ninguno de los dos deja hallazgos de severidad media o mayor. Codex revirtió cada arreglo de la ronda 2 en su copia, y los tests o el script atraparon los tres. Fable confirmó la cita del Author's Guide y que la mutación de G1 distingue las dos reglas, no solo el mensaje.

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| Fable 1 | `range len(bs)` en `compareVersions` sobrevive: `1.5.3` contra `1.5` daría 0 | baja | Aceptado: casos `1.5.3` contra `1.5`, y `covers` con `versionTo` 1.5 |
| Fable 2 | La lista de mutaciones del plan no nombraba las dos del review | baja | Aceptado |
