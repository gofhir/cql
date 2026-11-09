# 018: el Date junto a un DateTime se convierte con ToDateTime

Rama `fix/mixed-date-duration`. Viene del backlog 1c29, abierto por el 017.

## Por qué

CQL 1.5.3 declara una conversión implícita de Date a DateTime (Appendix B, Convert: la celda Date→DateTime es «Implicit»). El traductor de Java la inserta como `ToDateTime` en cada operador donde un Date se encuentra con un DateTime. El motor no la aplica nunca. sema la conoce para resolver sobrecargas (`implicitConversions` de `sema/convert.go`), pero `convert` (`sema/check.go:451`) decide no registrarla, a diferencia de Integer→Decimal. Así, el Date llega como Date a operadores que esperan un DateTime y cada uno lo lee a su manera.

## Revisión previa (02-11-2026)

### `gofhir/server` (congelado)

No evalúa CQL: usa `gofhir/cql`. No hay nada que tomar.

### Especificación (CQL 1.5.3)

- **Convert:** Date → DateTime es una conversión implícita.
- **ToDateTime** de un Date: «the time components unspecified, except for the timezone offset, which will be set to the timezone offset of the evaluation request timestamp».
- **Duration y Difference:** con operandos mixtos, «the Date values will be implicitly converted to DateTime».

### Lo que escribe el traductor de Java (cql-to-elm de engine 5.4.0)

| CQL | ELM |
|---|---|
| `@2012-01-01 = @2012-01-01T00:00:00.000Z` | `Equal(ToDateTime(Date), DateTime)` |
| `days between @2012-01-01T23:59:59.999 and @2012-01-03` | `DurationBetween(DateTime, ToDateTime(Date))` |
| `difference in hours between @2012-01-01 and @2012-01-02T05:00` | `DifferenceBetween(ToDateTime(Date), DateTime)` |
| `@2012-01-01T10:00:00.000 between @2012-01-01 and @2012-01-05` | `GreaterOrEqual(DateTime, ToDateTime(Date)) and LessOrEqual(...)` |
| `Interval[@2012-01-01, @2012-01-05T00:00:00.000]` | `Interval(ToDateTime(Date), DateTime)` |
| `{ @2012-01-01, @2012-01-01T10:00:00.000 }` | `List(ToDateTime(Date), DateTime)` |
| `if true then @2012-01-01 else <DateTime>` | `If(…, ToDateTime(Date), DateTime)` |
| `@2012-01-01 in { <DateTime> }` | `In(ToDateTime(Date), List(DateTime))` |
| `@2012-01-01 during Interval[<DateTime>, <DateTime>]` | `In(ToDateTime(Date), Interval(DateTime, DateTime))` |
| `Coalesce(null, @2012-01-01, <DateTime>)` | `Coalesce(Null, Date, DateTime)`: **no convierte** (Choice) |
| `{ @2012-01-01 } union { <DateTime> }` | `Union(As(List<Date>), As(List<DateTime>))`: **no convierte** (Choice) |

ELM en `scratchpad/r018/elm/probe.elm.json`.

### Sondas: 44 casos contra Java y cql-execution (`scratchpad/r018/in/probe.cql`)

En main, las respuestas que difieren de las dos referencias son:

| Expresión | Java | cql-execution | Motor |
|---|---|---|---|
| `@2012-01-01 = @2012-01-01T00:00:00.000Z` | false | false | **null** |
| `days between @2012-01-01T23:59:59.999 and @2012-01-03` | 1 | `[1, 2]` | **0** |
| `days between @2012-01-03 and @2012-01-01T23:59:59.999` | -1 | `[-2, -1]` | **0** |
| `hours between @2012-01-01 and @2012-01-02T05:00` | `[6, 29]` | `[5, 29]` | **32** |
| `difference in hours between @2012-01-01 and @2012-01-02T05:00` | `[6, 29]` | `[6, 29]` | **32** |
| `Interval[@2012-01-01, @2012-01-05T00:00:00.000]` | cota baja DateTime | igual | **cota baja Date** |
| `{ @2012-01-01, @2012-01-01T10:00:00.000 }` | dos DateTime | igual | **un Date** |
| `if true then @2012-01-01 else <DateTime>` | DateTime | igual | **Date** |

Además, `testdata/reference/runtime.cql` tiene `ListDateDateTime: First({@2020-01-01, @2020-01-01T10:00:00}) is DateTime`. Java y cql-execution dan true, y el motor da false. `TestRuntimeMatchesTheReference` lo fija como decisión de #132 (16-10-2026, antes del proceso §10): «a Date compared at the day is read as written and a DateTime is moved to an offset, so promoting it would change what every comparison it reaches answers». Este plan revisa esa decisión.

### Un prototipo (`scratchpad/proto018`)

Registré Date→DateTime como se registra una promoción numérica, y el evaluador aplica ToDateTime. Resultado:
- Las 23 áreas de snapshot no cambian ninguna respuesta: no tienen casos mixtos que dependan de esto.
- Alcanza los literales de intervalo y de lista y el `if`. No alcanza `=`, `<`, las duraciones, `difference`, `between` ni las frases de tiempo: sema no busca un tipo común para esos operadores con temporales, así que nunca pide la conversión.
- Convierte en `Coalesce`, donde el traductor no lo hace: es una regresión contra las referencias.
- Rompe dos tests que fijan la decisión vieja: `ListDateDateTime` y `TestANumericPromotionIsRecordedButCallsNothing`.

## Decisión del usuario (02-11-2026)

La conversión general, no solo en las duraciones.

## Cambio de arquitectura

1. **sema registra Date→DateTime** como registra Integer→Decimal: una conversión sin función (`DatePromotion`, junto a `NumericPromotion`), directa o dentro de una lista o un intervalo.
2. **sema busca el tipo común de los temporales** en los operadores donde el traductor inserta ToDateTime: `=`, `!=`, `~`, `!~`, `<`, `<=`, `>`, `>=`, `between`, las frases de tiempo (`same as`, `before`/`after`, `during`, `includes`, `in` de intervalo), `in`/`contains` de lista, `DurationBetween`, `DifferenceBetween`, `duration in`/`difference in` de un intervalo, los literales de intervalo y de lista, `if` y `case`. Si un lado es Date y el otro DateTime (o un intervalo o una lista de DateTime), registra la conversión del lado Date.
3. **No en `Coalesce` ni en `union`/`intersect`/`except`**, que el traductor resuelve a un Choice.
4. **El evaluador aplica la conversión** en `evalAndConvert`, con `funcs.ToDateTime`. El DateTime queda sin offset propio, así que se lee al offset de la evaluación, como dice ToDateTime.
5. **La decisión de #132 se revierte:** `ListDateDateTime` pasa a true, como las dos referencias, y el test que prohibía registrar Date→DateTime pasa a exigirlo. `readAsWritten` no cambia: un DateTime sin offset ya se lee así por debajo de la hora, y `@2012-01-01 same day as @2012-01-01T00:00:00.000Z` sigue dando true, como en las dos referencias (el prototipo no lo cambió).

## Alcance

Los puntos 1 a 5. Una nueva área de snapshot `mixeddates`, con cientos de casos: Date de año, mes y día junto a un DateTime de año a milisegundo, con y sin offset, en cada operador del punto 2, en los dos órdenes, más los casos del punto 3. Se compara contra Java y cql-execution, y el ELM de Java se importa para comparar los dos caminos.

## Fuera del alcance

- `Max`/`Min` de una lista mixta. Con la conversión, el motor da `@2012-01-01T10:00:00.000` y Java el Date convertido. Es una comparación que no se puede decidir dentro de un agregado: otro frente.
- La normalización de offsets por debajo de la hora (1c28) y `CalculateAgeInYears(null)` (1c30).
- Los backlogs 1c26 y 1c27 del 016.

## Consecuencias previstas

- `days between <DateTime> and <Date>` con el Date a la unidad pasa a ser una incertidumbre (`[1, 2]`), como el ejemplo normativo, y difiere de Java como en el 016 y el 017. Pasa lo mismo con `duration in days of Interval[@2012-01-01, @2012-01-03T10:00:00.000]` (main: 2; con la conversión, `[1, 2]`, como cql-execution).
- El impacto en eCQM se mide antes y después: el corpus compara fechas de nacimiento (Date) con el Measurement Period (DateTime).

## Criterio de salida

- Las ocho filas de la tabla de sondas y `ListDateDateTime` dan lo que dan las dos referencias, o la regla del 016 donde las referencias difieren entre sí (las duraciones: `[1, 2]`, `[-2, -1]` y `[5, 29]`; `difference`: `[6, 29]`).
- `Coalesce` y `union` no convierten, como el traductor.
- El área `mixeddates` tiene cada divergencia clasificada con evidencia. Los caminos AST y ELM de Java coinciden.
- La diferencial contra main es idéntica fuera de los casos mixtos, en todas las áreas.
- eCQM da 23 coincidentes y 1 excusado, o cada cambio queda explicado y aceptado por el usuario.
- Los tests nuevos fallan en main, hay mutaciones para cada oráculo y `make check` está en verde.

## Tests y fuentes de datos

- A mano, con la sección de la especificación: las ocho filas, `ListDateDateTime`, `Coalesce`, `union` y `same day as` con Z.
- Un test de sema por cada operador del punto 2: la conversión queda registrada del lado Date, y no se registra en `Coalesce` ni en `union`.
- El área `mixeddates` (generada: forma del Date × forma del DateTime × operador × orden) contra Java y cql-execution.
- El ELM de Java del área importado: el mismo resultado que el AST.
- Metamórficos: cada expresión mixta da lo mismo que la misma con `ToDateTime(...)` explícito (el oráculo es el propio motor sobre la forma explícita, que no depende de este cambio), y el offset de la evaluación cambiado da lo que predice ToDateTime.

## Mutaciones previstas

- `DatePromotion` siempre false, y siempre true; la conversión aplicada en UTC.
- Un operador del punto 2 sin el tipo común.
- `Coalesce` convertido.
- El área fuera del manifiesto y de las referencias; una fila deliberada que cita un plan que no la nombra.


## Ronda adversarial (Opus independiente, 02-11-2026)

Codex sigue sin cuota hasta el 04-11-2026, así que, por decisión del usuario, la ronda la hizo un Opus independiente. Verifiqué los hallazgos altos con una sonda propia contra Java y cql-execution (`scratchpad/r018/in/probe2.cql`).

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| H1 | El mecanismo no cumple la fila 1. `ToDateTime` de un Date da un DateTime sin offset, que se lee «as written»: `ToDateTime(@2012-01-01) = @2012-01-01T00:00:00.000Z` da null, y `ToDateTime(@2012-01-01) < @2012-01-01T02:00:00.000Z` también; Java y cql-execution dan false en los dos. La especificación pide que ToDateTime ponga «the timezone offset of the evaluation request timestamp» | alta | Confirmado. Es un defecto de `ToDateTime` explícito, anterior al cableado de la conversión implícita, y se propone como frente propio |
| H2 | `difference in hours between ToDateTime(@2012-01-01) and @2012-01-02T05:00` da 32, y las dos referencias dan `[6, 29]`: `difference` no aplica la incertidumbre, aunque el Date se convierta | alta | Confirmado. Es un defecto de `DifferenceBetween` y se propone como frente propio |
| H3 | El oráculo metamórfico (la forma con `ToDateTime` explícito) usa el propio motor, que se equivoca en H1 y H2 | alta | Aceptado: es un oráculo del cableado, no de la corrección. La corrección sale de Java y cql-execution |
| M1 | Con el offset puesto, cambian las respuestas con un DateTime en otro offset: `hours between @2012-01-01 and @2012-01-02T05:00:00.000Z` da 29 en main; con ToDateTime, el motor da `[2, 26]` (cql-execution `[2, 26]`, Java `[3, 26]`) | media | Aceptado: un eje de offsets distintos (Z y +14 contra la evaluación) en el área |
| M2 | El camino FHIR no se cubre: tras una función del modelo (FHIR.date → FHIRHelpers.ToDate), `applyPlannedConversion` solo aplica una promoción numérica, así que un `birthDate` comparado con el Measurement Period seguiría siendo Date | media | Aceptado |
| M3 | Faltan los argumentos de funciones: `define function F(x DateTime)` con `F(@2012-01-01)` no convierte, y las funciones de librerías incluidas no se tipan (`inferFunctionCall` da Unknown si `e.Library != ""`) | media | Aceptado |
| M4 | Faltan operadores (`overlaps`, `meets`, `starts`, `ends`, `properly includes`, `within … of`, frases con offset, `contains` de intervalo, `=`/`~` de listas) y la ELM de Java de las operaciones de conjuntos sobre intervalos | media | Aceptado |
| M5 | La medición de eCQM es débil: hacen falta pacientes en el borde (cumpleaños el primer día del período, período escrito en Z evaluando a -03:00, `AgeInDays` de neonatos) | media | Aceptado |
| L1 | `Max` de una lista mixta ya da `@2012-01-01T10:00:00.000` en main: no es una regresión nueva | baja | Aceptado |

**Propuesta de división** (un frente a la vez):
- **018:** `ToDateTime` de un Date lleva el offset de la evaluación (H1 y M1). Es el cimiento de los demás: hoy la forma explícita ya se equivoca.
- **019:** la conversión implícita Date→DateTime registrada en sema y aplicada en el evaluador, en todos los operadores (M4), los argumentos de funciones (M3) y el camino FHIR (M2), con la reversión de #132 y la medición de eCQM (M5).
- **020:** `difference` con incertidumbre (H2).


## Revisión previa del 018 dividido (02-11-2026)

El usuario aprobó dividir en tres. Al medir el primero (`scratchpad/r018/in/probe3.cql`, contra Java y cql-execution, evaluando a -03:00), el defecto resultó más amplio que `ToDateTime`: **el motor no ubica en ningún offset a un DateTime más grueso que la hora**, venga de donde venga.

| Expresión | Java | cql-execution | Motor |
|---|---|---|---|
| `@2012-01-01T = @2012-01-01T00:00:00.000Z` | false | false | null |
| `DateTime(2012, 1, 1) = @2012-01-01T00:00:00.000Z` | false | false | null |
| `@2012-01-01T < @2012-01-01T02:00:00.000Z` | false | false | null |
| `DateTime(2012, 1, 1) < @2012-01-01T02:00:00.000Z` | false | false | null |
| `@2012-01T = @2012-01-01T00:00:00.000Z` | false | false | null |
| `ToDateTime(@2012-01-01) = @2012-01-01T00:00:00.000Z` | false | false | null |
| `@2012-01-01T same day as @2012-01-01T23:00:00.000-05:00` | true | true | true |
| `@2012-01-01T = @2012-01-01T00:00:00.000-03:00` | null | null | null |

Las dos referencias ubican el DateTime de día al offset de la evaluación. Eso dice la especificación de los literales («If no timezone offset is supplied, the timezone offset of the evaluation request timestamp is assumed») y de ToDateTime. `same day as` sigue leyéndose como está escrito, por la regla de Same As (normalizar solo a hora o más fino), y no cambia.

**Dónde está la causa.** Son dos piezas, y una es de otra librería:
1. `gofhir/fhirpath` v1.16.6 (la última), `types.DateTime.WithDefaultOffset`: rechaza a propósito el offset por defecto si la precisión es menor que la hora («@2020 has no instant to place»), y `EffectiveOffset` no da ninguno. El motor no tiene cómo ubicar el valor sin rodear la librería.
2. `gofhir/cql`, `framelessTemporal` (`eval/evaluator.go`): trata a ese DateTime como un Date, sin marco, por la misma razón. Esa decisión es nuestra.

Por la regla de no rodear librerías, el arreglo de la pieza 1 va en fhirpath. Un DateTime de día de CQL sí tiene un instante en el que empieza, y la especificación de CQL le da el offset de la evaluación. Si eso es un defecto de fhirpath o una política que la API debería dejar decidir a quien llama, se decide allá, con un issue.


**Decisión del usuario (02-11-2026):** primero el issue a `gofhir/fhirpath`. El 018 queda detenido hasta que exista una versión de fhirpath que permita ubicar un DateTime de día. Issue publicado con la aprobación del usuario: gofhir/fhirpath#256 (02-11-2026); no hay duplicados (se buscó «WithDefaultOffset», «default offset» y «offset precision day»). Reproducción medida en Go contra v1.16.6: `EffectiveOffset` da `0 false` en día y `-180 true` en hora.

## Revisión previa con fhirpath v1.16.7 (05-11-2026)

gofhir/fhirpath#256 se corrigió en v1.16.7 (68fa7e5, la última versión). Ahora `WithDefaultOffset` acepta cualquier precisión, y `compareCoarsePlaced` compara un valor más grueso que la hora y ya ubicado como el intervalo de tiempo que nombra. Sonda de 42 casos contra Java 5.4.0 y cql-execution 3.3.2, y contra el motor en main y con la versión nueva (`scratchpad/r018b/table018.tsv`). Antes de la versión nueva se corrió `go test ./...` (`scratchpad/r018b/bump-test.log`).

**Subir solo la versión no sirve.** Fallan 15 tests del paquete raíz y uno de `conformance`. Las respuestas que cambian son de tres clases:

1. **Arregla:** `timezoneoffset from @2020-01-01T` da -3, como las dos referencias (main da null).
2. **No arregla, por un defecto de cql:** `@2012-01-01T = @2012-01-01T00:00:00.000Z` sigue en null, y las referencias dan false (D0-D8).
   - fptypes ya da el orden correcto. Lo deshace el paso extra de `CompareTemporal` (`types/temporal_compare.go`), que pasa los dos valores a UTC con `inUTC`, y un día a -03:00 queda en el día 1 a UTC.
   - Prototipo: si cql se salta ese paso cuando un valor es más grueso que la hora y ya está ubicado, D0-D8 coinciden con las referencias.
3. **Rompe, porque falta la conversión de Date a DateTime:** un Date frente a un DateTime de día da null o false, y las referencias dan true o false (D12-D22).
   - Ejemplos: `@2020-03-01 = @2020-03-02T`, `@2020-03-01 in {@2020-03-01T}`, y `PeriodDuringMP` con `parameter MP Interval<DateTime> default Interval[@2020-01-01, @2020-12-31]`, la forma de los eCQM. También deja vacío el pushdown de fechas.
   - fhirpath responde `ErrOffsetMismatch` a propósito: un Date no tiene offset y el DateTime ya está ubicado. Eso es correcto en FHIRPath, así que no es un defecto de la librería.
   - En CQL lo resuelve el traductor, que inserta ToDateTime del lado del Date, y ToDateTime lo ubica al offset de la evaluación. Es la conversión implícita del 019, más el ToDateTime con offset del 018 original (H1).

**Consecuencia:** el 018 (subir la versión, `CompareTemporal` y ToDateTime con offset) no se puede mergear sin el 019 (la conversión implícita), porque rompería los eCQM. Hace falta decidir el alcance.

## Decisión del usuario (05-11-2026): unir el 018 y el 019

Como la versión nueva sola rompe los eCQM, el 018 y el 019 van en un solo incremento. Este plan reemplaza al «Alcance» y al «Criterio de salida» de arriba. Las secciones anteriores quedan como historia.

## Plan del 018 unido

### Alcance

1. **Subir fhirpath a v1.16.7.**
2. **`CompareTemporal`** (`types/temporal_compare.go`). Hoy el paso extra de precisión pasa los dos valores a UTC con `inUTC`. Ese paso se salta cuando un valor es más grueso que la hora y ya está ubicado: la respuesta de fptypes (`compareCoarsePlaced`) ya es la de CQL. El paso extra sigue para lo que existe, la diferencia entre segundos y milisegundos a la hora o más fino.
3. **Conversión implícita Date→DateTime**, el punto 1 del «Cambio de arquitectura»:
   - sema registra `DatePromotion` como registra `NumericPromotion`: una conversión sin función, directa o dentro de una lista o un intervalo;
   - sema busca el tipo común en los operadores del punto 2;
   - no convierte en `Coalesce` ni en `union`/`intersect`/`except`, que el traductor resuelve a un Choice (punto 3).
4. **El evaluador aplica la conversión** con `funcs.ToDateTime` y `situate`, igual que ToDateTime explícito: el resultado queda al offset de la evaluación. El camino FHIR (M2) también convierte, en `applyPlannedConversion`, después de una función del modelo.
5. **Los argumentos de funciones** (M3): `define function F(x DateTime)` llamada con `F(@2012-01-01)` convierte, también en una librería incluida.
6. **Los operadores** de M4 (`overlaps`, `meets`, `starts`, `ends`, `properly includes`, `within … of`, `contains` de intervalo, `=` y `~` de listas) y los parámetros con default (`parameter MP Interval<DateTime> default Interval[@2020-01-01, @2020-12-31]`).
7. **Se revierten las decisiones viejas** que fijan los tests:
   - `ListDateDateTime` pasa a true, como las dos referencias (#132);
   - el test que prohíbe registrar Date→DateTime pasa a exigirlo;
   - `timezoneoffset from @2020-01-01T` da -3, como las dos referencias (`engine_default_offset_doors_test.go`).

### Fuera del alcance

- `Max`/`Min` de una lista mixta (L1), 1c28 y 1c30.
- La forma con que se imprime un DateTime sin offset (decisión del usuario del 05-11-2026 en el 023).

### Criterio de salida

- **Sonda:** las 42 filas de `table018.tsv` dan lo que dan las dos referencias. Donde las referencias difieren entre sí, la regla del 016. Las demás quedan clasificadas con evidencia.
- **Sin regresiones:** los 15 tests que rompe la versión nueva vuelven a pasar, salvo los que fijan una decisión vieja (punto 7). Esos se cambian con la cita y la respuesta de las referencias.
- **Área `mixeddates`:** cientos de casos generados. Cada divergencia queda clasificada con evidencia, y los caminos AST y ELM de Java coinciden.
- **Diferencial contra main:** idéntica fuera de los casos mixtos y de los DateTime más gruesos que la hora, en todas las áreas. Cada cambio queda explicado.
- **eCQM:** 23 coincidentes y 1 excusado, más los pacientes en el borde de M5. Todo cambio lo acepta el usuario.
- **Cierre:** los tests nuevos fallan en main, cada oráculo tiene mutaciones y `make check` pasa.

### Tests y fuentes de datos

- **A mano, con la cita de la 1.5.3 (N1A):**
  - las filas de la sonda;
  - `Coalesce` y `union` sin conversión;
  - `same day as` con Z;
  - el MP con default Date;
  - una función con un argumento DateTime;
  - un `birthDate` de FHIR frente al MP.
- **sema:** un test por operador del punto 2 y de M4. La conversión queda registrada del lado del Date, y no se registra en `Coalesce` ni en `union`.
- **Área `mixeddates`:** generada combinando forma del Date × forma del DateTime (con y sin offset, Z y +14) × operador × orden, contra Java y cql-execution. Se importa el ELM de Java para comparar los dos caminos.
- **Oráculo independiente en Python**, que ubica cada valor al offset de la evaluación y compara por intervalos de tiempo. El oráculo metamórfico con el motor queda descartado (H3).

### Mutaciones previstas

- `DatePromotion` siempre false, y siempre true.
- La conversión aplicada sin `situate`.
- Un operador sin el tipo común.
- `Coalesce` convertido.
- El paso extra de `CompareTemporal` otra vez con `inUTC` para un valor ubicado más grueso que la hora.
- El área fuera del manifiesto y de las referencias.
- Una fila deliberada que cita un plan que no la nombra.
- Una divergencia sin clasificar.

### Dependencias

gofhir/fhirpath v1.16.7, que es una versión nueva de una dependencia que ya existe.

## Dependencias

Ninguna nueva.

### Ronda adversarial 1 del plan unido (Codex, 05-11-2026)

| # | Severidad | Hallazgo | Resolución |
|---|---|---|---|
| 1 | alta | El salto en `CompareTemporal` arregla `<`, pero las frases de tiempo quedan incoherentes: leen el valor con `framelessTemporal` y `readAsWritten`. | **Aceptado, verificado** con `scratchpad/r018b/probe018c.cql`. Con el prototipo, `@2012-01-01T < @2012-01-01T02:00:00.000Z` da false, como Java. `before`, `same or before` y `on or after` con los mismos valores siguen en null. Un DateTime ya ubicado deja de tratarse como un Date en las frases, salvo la regla de Same As (leer como está escrito por debajo de la hora). |
| 2 | alta | La regla de segundos y milisegundos contradice la 1.5.3: `@T10:00:00 = @T10:00:00.000` da null. | **Rechazado para este incremento.** El motor responde igual que Java (sonda D4 y D5). La lectura es la pregunta 2 del borrador para HL7 (`docs/hl7-cql-clarification.md`). El plan ya no presenta esa regla como requisito de CQL: solo no la cambia. |
| 3 | alta | Con la versión nueva y el salto, los 15 tests siguen fallando, y el de `conformance` compara valores iguales con distinto estado interno. | **Aceptado.** Los 15 son de la clase 3 (falta la conversión). `TestParseExpectedOutput` queda en el alcance con una resolución explícita. |
| 4 | alta | sema no tiene todo lo que necesita. Las comparaciones solo comprueban convertibilidad, `convert` descarta Date→DateTime (`sema/check.go:451`), los defaults con tipo declarado no se infieren (`sema/check.go:217`) y las llamadas a una librería incluida dan Unknown (`sema/query.go`). | **Aceptado.** Resolver firmas entre librerías es trabajo de arquitectura aparte. |
| 5 | media | La exclusión de las operaciones de conjuntos no distingue listas de intervalos. Las operaciones sobre Periods FHIR fallan con la versión nueva. | **Aceptado.** Se decide por sobrecarga con el ELM de Java, regenerado. |
| 6 | media | Parte del diagnóstico quedó viejo: `difference` ya da `[6, 29]` desde el 020. Un oráculo solo por intervalos no reproduce `same day as`. | **Aceptado.** El oráculo lleva reglas por operador y precisión. |
| 7 | alta | 23+1 no prueba la seguridad de las poblaciones: Codex no tenía el corpus, y Java y cql-execution comparten traductor. | **Aceptado.** Hacen falta resultados por paciente y por población, con los casos de borde de M5, offsets fraccionarios, medianoche y límites abiertos o null. |

**División que propone Codex:** primero las conversiones sobre v1.16.6, con la suite en verde. Eso abarca defaults, el camino FHIR y las firmas de librerías incluidas. Después, v1.16.7, el salto en `CompareTemporal` y las frases coherentes, con eCQM medido por población. El prototipo del 02-11 (`scratchpad/proto018`) hizo la conversión sobre v1.16.6 y rompió solo los dos tests que fijan la decisión vieja, más `Coalesce`. Esto contradice la decisión del 05-11 de unir el 018 y el 019, así que se consulta al usuario.

**Decisión del usuario (06-11-2026): dividir.** Primero el **019**, la conversión implícita Date→DateTime sobre fhirpath v1.16.6, con la suite en verde: sema, defaults, el camino FHIR, los argumentos y los operadores. Su plan está en [019-date-promotion.md](019-date-promotion.md). Después, el **018**: la subida a v1.16.7, `CompareTemporal` y las frases de tiempo, con eCQM medido por población (hallazgos 1, 3, 6 y 7). Los hallazgos 4 y 5 pasan al 019.

## Revisión previa tras el 019 (06-11-2026)

El 019 está mergeado (#179, 995bf95). La sonda se repitió sobre main con fhirpath v1.16.7, de tres formas: solo la versión, la versión más el salto en `CompareTemporal` del plan unido, y la versión más la regla de la 1.5.3 (abajo). Los logs están en `scratchpad/r018c/test-{bump,skip,written}.log`.

### La premisa del 018 no se sostiene

La 1.5.3 limita la normalización de offsets (Appendix B, cada operador de comparación y de tiempo; Author's Guide, Date and Time Operators): «When comparing DateTime values with different timezone offsets, implementations should normalize to the timezone offset of the evaluation request timestamp, but only when the comparison precision is hours, minutes, seconds, or milliseconds». Contra un año, un mes o un día no se normaliza: los componentes se comparan como están escritos.

El 018 partía de que `@2012-01-01T = @2012-01-01T00:00:00.000Z` da false porque así responden Java 5.4.0 y cql-execution 3.3.2 (D0-D8 de `table018.tsv`). Pero **las dos referencias dependen del orden de los operandos**. Mueven el operando derecho al offset del izquierdo, y deciden solo cuando eso cambia el día. Pares del área `mixeddates`, con la misma respuesta en las dos:

| Definición | Java y cql-execution | Espejo | Java y cql-execution |
|---|---|---|---|
| D165 `@2020-03 = @2020-03-01T10:00:00.000+14:00` | false | D166 `@2020-03-01T10:00:00.000+14:00 = @2020-03` | null |
| D185 `@2020-03-01 = @2020-03-01T10:00:00.000+14:00` | false | D186 `@2020-03-01T10:00:00.000+14:00 = @2020-03-01` | null |
| D188 `@2020-03-01T10:00:00.000-12:00 = @2020-03-01` | false | D187 `@2020-03-01 = @2020-03-01T10:00:00.000-12:00` | null |
| D148 `@2020-03-01T10:00:00.000-12:00 = @2020` | false | D147 `@2020 = @2020-03-01T10:00:00.000-12:00` | null |

Un `=` que cambia al dar vuelta los operandos no es el de la especificación. La respuesta de la 1.5.3 es null en los dos órdenes, y es la que el motor ya da en main. **D0-D8 son defectos de las referencias, no del motor.**

### Lo que hace fhirpath v1.16.7

Las dos variantes del plan unido mueven las mismas filas de `mixeddates`: 64 dejan de divergir y aparecen 46 nuevas. El salto en `CompareTemporal` no cambia nada. Lo que cambia es que `WithDefaultOffset` ahora ubica los valores gruesos, y fptypes los normaliza. Así el motor coincide con las referencias en el orden en que deciden y diverge en el espejo (D166, D186, D226…): es una regresión contra la 1.5.3.

El cambio de fhirpath es opt-in y cumple lo que pidió gofhir/fhirpath#256, así que no es un defecto de la librería. El issue, en cambio, apoyaba la propuesta en las respuestas de las referencias sin la restricción «only when the comparison precision is hours…». Hay que corregirlo con un comentario.

### La regla de la 1.5.3, medida (`scratchpad/p018-written`)

En `CompareTemporal`, cuando la precisión más gruesa del par es año, mes o día, los componentes se comparan como están escritos. Desde la hora para abajo, nada cambia. Con v1.16.7:

- **`mixeddates`:** ninguna divergencia nueva. Dejan de divergir las 12 filas de `union`, `intersect` y `except` de listas (D1067-D1092): el Date y el DateTime ubicado ya no son el mismo elemento, como en las dos referencias.
- **Tests que fijan una decisión vieja contra la 1.5.3:**
  - `engine_default_offset_doors_test.go`: `timezoneoffset from` un DateTime de día ahora da el offset de la evaluación. El Author's Guide dice: «the result of extracting the timezone offset component will be the timezone offset of the evaluation request, not null». Las dos referencias dan -3.
  - `TestRuntimeMatchesTheReference`: `DateTimeYear` ahora lleva -03:00.
  - `TestParseExpectedOutput` (`conformance`): compara valores que se imprimen igual y difieren en el offset ubicado.
- **Huecos del 019 que v1.16.7 deja a la vista:** con v1.16.6, un Date se comparaba sin conversión contra un DateTime sin ubicar. Ahora da `ErrOffsetMismatch`.
  - `Interval[@2020-03-01, @2020-03-05] in MP` da null. El `in` con un intervalo a la izquierda no convierte.
  - `IndexOf({@2020-03-01T}, @2020-03-01)` da -1. No convierte los argumentos.
  - `Tuple{d: @2020-03-01} = Tuple{d: @2020-03-01T}` da false. El traductor rechaza la igualdad de tuplas Date/DateTime, así que no hay respuesta de referencia.

## Plan del 018 (06-11-2026)

### Alcance

1. **Subir fhirpath a v1.16.7**, la última versión.
2. **La regla de la 1.5.3 en `CompareTemporal`:** por encima de la hora, los componentes se comparan como están escritos y no se normalizan offsets. Desde la hora para abajo, no cambia nada, y tampoco cambian las duraciones, que ya siguen la regla (016, 021).
3. **`timezoneoffset from`** un DateTime grueso da el offset de la evaluación. Viene con la versión nueva; se cambia el test que fijaba null.
4. **Los huecos del 019 que v1.16.7 deja a la vista:** la conversión en `in` con un intervalo a la izquierda y en los argumentos de `IndexOf`, donde el ELM de Java convierte (se mide antes). Antes de implementar se recorren las operaciones del área que comparan un Date con un DateTime, para encontrar otros huecos.
5. **Reclasificar en `mixeddates`** los veredictos del 019 que dependían de la premisa:
   - las 52 filas «own defect … resolved by plan 018», salvo las 12 de listas, pasan a defectos de las referencias, con el espejo como evidencia;
   - D467, D475, D515 y D523 (8 filas) también;
   - el texto de las 22 filas `-12:00` dice ahora que las referencias dependen del orden;
   - el backlog 1c41 se cierra.
6. **Corrección pública:** borrador de comentario para gofhir/fhirpath#256 con esta evidencia, y el punto para HL7 en `docs/hl7-cql-clarification.md`: las referencias dependen del orden. Nada se publica sin aprobación.

### Fuera del alcance

- La igualdad de tuplas Date/DateTime: el traductor la rechaza. Se registra la respuesta nueva, y la decisión entre false y null queda para el usuario.
- La forma impresa (023) y `Max`/`Min` (L1).
- 1c40, 1c42, 1c43 y 1c44.

### Criterio de salida

- **Simetría y regla:** test nuevo, con un oráculo propio en Python que compara componentes como están escritos por encima de la hora.
  - Cada par del área y su espejo dan respuestas espejadas (`a = b` igual que `b = a`, y `a < b` igual que `b > a`).
  - Por encima de la hora, la respuesta es la de comparar como está escrito: D166 y D186 dan null.
  - La simetría sola no basta, porque la versión sola también es simétrica (normaliza los dos lados). El test falla con la versión sola por la regla.
- **`mixeddates`:** ninguna divergencia nueva sin clasificar. Las de las referencias citan su espejo.
- **Los tests de decisiones viejas** cambian con la cita de la 1.5.3 y la respuesta de las referencias.
- **Diferencial contra main** en los nueve corpus: cada cambio se explica.
- **eCQM por población:** 23 coincidentes y 1 excusado, sin cambios por paciente (hallazgo 7 del plan unido). Se miden también las poblaciones de borde: medianoche, `+14:00` y `-12:00`, y offsets fraccionarios.
- **Cierre:** mutaciones de cada pieza y `make check` en verde.

### Mutaciones previstas

- La regla aplicada también desde la hora (normaliza a la hora): la detecta la simetría.
- La regla quitada (vuelve a normalizar por encima de la hora).
- La conversión de `in` con intervalo quitada.
- La de `IndexOf` quitada.
- Un veredicto de espejo sin su espejo.

### Dependencias

fhirpath v1.16.7, versión nueva de una dependencia que ya existe.

### Ronda adversarial 1 del plan del 06-11 (Codex)

| # | Severidad | Hallazgo | Resolución |
|---|---|---|---|
| 1 | alta | La lectura central no está establecida por la 1.5.3. Same As dice «finest precision specified in either input», y el Author's Guide equipara los operadores de tiempo sin precisión con los símbolos. Equal y Less describen el recorrido por componentes, pero no dicen que la precisión que controla la normalización sea la más gruesa del par. La asimetría prueba que las referencias tienen un defecto, pero no dice cuál es la respuesta simétrica correcta. | **Aceptado.** Es una lectura que la especificación no decide. Se consulta al usuario y se agrega al borrador para HL7. |
| 2 | alta | `Min`, `Max` y `sort` comparan por `compareValues`, que no pasa por `CompareTemporal`. Con v1.16.7, `Min({@2020-03-01T, @2020-03-01T10:00:00.000+14:00})` cambia de día a instante, también con la regla. | **Aceptado, verificado** (`scratchpad/zz_probe_test.go` en main, en la versión sola y en el prototipo): es una regresión de la versión. Entra al alcance como diferencial obligatoria. |
| 3 | alta | `@2020-01-01T00:10:00Z same hour as @2020-01-01T01:40:00+01:00` da true con la evaluación en `+05:30`, en main y con la versión. En ese offset son las 05:40 y las 06:10. | **Aceptado, verificado.** Es un defecto previo: la frase normaliza a UTC y no al offset de la evaluación. Va al backlog. |
| 4 | media | La mutación «la regla también a la hora» es simétrica: la simetría no la detecta. | **Aceptado.** Se detecta con expectativas explícitas a la hora normalizada, por ejemplo `@2020-01-01T10+05:00 = @2020-01-01T05Z`. |
| 5 | media | La regla necesita un límite de tipo explícito y un recorrido por los consumidores: `in` de intervalo, `IndexOf`, tuplas, listas, `sort`, Same As y Time. | **Aceptado.** |
| 6 | media | El área usa un solo offset de evaluación (-03:00) y cuatro offsets de operando. Faltan otros marcos, offsets fraccionarios, medianoche y cambio de año, y Date convertido en runtime. El eCQM tiene que medirse por paciente y por población. | **Aceptado.** |
| 7 | baja | La asimetría vale para más pares (D167/D168, D823/D824, D829/D830). Además, cql-execution da D129/D130 = true/false sin offset escrito, así que no es solo la normalización. | **Aceptado.** Cada reclasificación cita su espejo. |

## Decisión del usuario (07-11-2026): solo la versión, sin reinterpretar

El usuario eligió subir a fhirpath v1.16.7 conservando las respuestas de main. Un DateTime más grueso que la hora no se ubica para comparar. Se arregla solo lo que la 1.5.3 dice claro. La lectura de la comparación por encima de la hora va al borrador para HL7. Este plan reemplaza al «Plan del 018 (06-11-2026)».

## Plan del 018 (07-11-2026)

### Revisión previa (prototipo `scratchpad/p018-guard`)

Con v1.16.7, los cinco lugares que ubican un DateTime al offset por defecto pasan por `types.WithDefaultOffset`, que no ubica un valor más grueso que la hora. Son `situate`, el literal, `carryDefaultOffsetTo`, `funcs.carryDefaultOffset` y el parser de `conformance`. Así se comporta como v1.16.6.

Resultado: `go test ./...` pasa entero y `mixeddates` no cambia ninguna respuesta. `Min` y `Max` del hallazgo 2 dan lo de main, y el `in` de intervalo, `IndexOf` y las tuplas vuelven a dar lo de main.

### Alcance

1. **Subir fhirpath a v1.16.7**, la última versión.
2. **`types.WithDefaultOffset`:** ubica al offset por defecto solo desde la hora. Los cinco lugares de arriba pasan por ahí. Es la regla de main, ahora escrita en cql y no heredada de la librería. No rodea un defecto: fhirpath ofrece la ubicación en toda precisión como opción, y cql elige no usarla por debajo de la hora (hallazgo 1 de la ronda del 06-11).
3. **`timezoneoffset from`** un DateTime sin offset da el offset de la evaluación. El Author's Guide (Date and Time Operators) dice: «the result of extracting the timezone offset component will be the timezone offset of the evaluation request, not null». Las dos referencias dan -3. Cambia el caso «a DateTime no finer than the day» de `TestWhatIsNotADoor`.
4. **Veredictos del 019 que citan el 018:**
   - las 52 filas «own defect … resolved by plan 018» de `mixeddates` (sin las 12 de listas) y D467, D475, D515 y D523 pasan a deliberadas, con la decisión del 07-11. Las referencias dependen del orden de los operandos (el espejo de cada fila), y el motor da null en los dos órdenes;
   - las 12 de listas siguen como defecto propio, con el texto apuntando al backlog;
   - el texto de las 22 filas `-12:00` cita el espejo;
   - el plan nombra todas las filas deliberadas.
5. **Borradores, sin publicar:**
   - un comentario para gofhir/fhirpath#256: la propuesta se apoyaba en una respuesta de las referencias que depende del orden;
   - el punto para HL7 en `docs/hl7-cql-clarification.md`: qué precisión controla la normalización en `=` y `<` sin precisión explícita.

### Fuera del alcance

- **Al backlog:**
  - `same hour as` y las frases normalizan a UTC y no al offset de la evaluación (hallazgo 3 de la ronda del 06-11);
  - la igualdad de conjuntos de listas Date/DateTime (las 12 filas);
  - el área con un solo marco de evaluación (hallazgo 6).
- 1c40, 1c42, 1c43 y 1c44.

### Criterio de salida

- **Sin cambios de respuesta en las áreas,** salvo `timezoneoffset`. `go test ./...` pasa con v1.16.7.
- **Diferencial contra main en los nueve corpus:** idéntica, salvo `timezoneoffset` y lo que se explique.
- **eCQM:** 23 coincidentes y 1 excusado. Las respuestas por paciente y por población son idénticas a main (hallazgo 6 de la ronda del 06-11).
- **Tests nuevos:**
  - `types.WithDefaultOffset` a cada precisión;
  - `timezoneoffset from` un DateTime de año, de mes y de día, a varios offsets de evaluación;
  - `Min`/`Max` del hallazgo 2, que falla con la versión sola.
  - un test que recorre el código de producción y falla si algo llama a `WithDefaultOffset` de fhirpath fuera de `types.WithDefaultOffset`. Cubre la mutación de un lugar suelto, que el área puede no ver.
- **Cierre:** mutaciones y `make check` en verde.

### Mutaciones previstas

- `WithDefaultOffset` ubica también por debajo de la hora: lo detectan el test de `Min`/`Max` y el área.
- Ubica solo desde el minuto: lo detecta el test por precisión.
- Uno de los cinco lugares vuelve a llamar a fhirpath directo.
- `timezoneoffset` vuelve a null.
- Una fila deliberada sin nombre en el plan.

### Ronda adversarial 2 del plan del 07-11 (Codex)

| # | Severidad | Hallazgo | Resolución |
|---|---|---|---|
| 1 | alta | v1.16.7 cambia más que la ubicación: decimales, primitivos FHIR, números JSON. Un `valueDecimal` de `1e1001` pasa de ausente a Decimal: `exists` pasa de false a true y `ToString` de null a `"1e1001"`. | **Aceptado, verificado** en `/private/tmp/claude-501/r018q/round2-*-resource.log` y en el CHANGELOG de v1.16.7. Se aceptan los cambios de la librería, que son correcciones, varias pedidas desde cql (#235, #232). Revertirlos sería un workaround. Cada entrada del CHANGELOG queda en el plan, con su efecto en CQL medido en la diferencial y en un test cuando llega a una respuesta de CQL. Si un decimal fuera de rango de CQL debe dar null es una pregunta de cql: va al backlog. |
| 2 | alta | Un DateTime grueso que el que llama ya ubicó con `WithDefaultOffset` no pasa por la guarda. `P = @2020-03-01T` pasa de true a null. | **Aceptado como límite conocido.** Con v1.16.6, el que llamaba no podía ubicar un valor grueso: el método no hacía nada. gofhir/server, el único consumidor, no lo llama (medido). Queda un test que documenta la respuesta, y una nota en la documentación de la API. |
| 3 | media | `timezoneoffset` necesita una regla explícita. El offset efectivo o escrito gana. Sin ninguno, va el de la evaluación, a toda precisión, también para un dateTime FHIR y un Date convertido. Minimum y Maximum quedan en UTC. Null da null. | **Aceptado.** Son tests para literales, constructores, ToDateTime, FHIRHelpers, aritmética y extremos, a varios offsets, incluido +05:30. |
| 4 | media | «Sin cambios» no se puede reproducir. El arnés eCQM evalúa el primer Patient sin timestamp fijo y junta null con false. | **Aceptado.** Diferencial por paciente con respuestas tipadas crudas, contra main y con timestamp fijo. Las excepciones van en una lista cerrada: caso, antes y después tipados, justificación. Se fijan la revisión de main y los hashes del corpus. |
| 5 | media | Si la guarda es un workaround depende de cómo se presente. | **Aceptado.** Se registra como la decisión de compatibilidad del usuario. La ubicación para comparar queda separada de la extracción del offset, y no se afirma que la 1.5.3 fije el corte en la hora. |
| 6 | baja | Las mutaciones no están verificadas. Los tests por precisión tienen que mirar `EffectiveOffset` y no `String()`. Faltan tests de comportamiento por cada uno de los cinco lugares, `sort`, offsets escritos y offsets de evaluación fraccionarios. | **Aceptado.** |

**Decisión del usuario (08-11-2026): plan aprobado**, con los hallazgos de las dos rondas incorporados.

## Veredictos del 019 reescritos (paso 7, 08-11-2026)

Lo hizo `scratchpad/r018c/reclassify.py` sobre `testdata/snapshots/mixeddates.divergences.tsv`:

- **Deliberadas por la decisión del 07-11:** el motor da null en los dos órdenes. Las referencias dependen del orden; cada fila cita su espejo cuando el área lo tiene. Filas: D165, D185, D225, D245, D345, D365, D425, D467, D475, D515, D523, D565, D566, D583, D584, D619, D620, D637, D638, D649, D650, D655, D656, D757, D758, D775, D776, D787, D793, D811, D823, D829, D847, D848, D1230, D1240, D1260.
- **Defectos propios, en el backlog 1c40:** `union` e `intersect` de intervalos con offsets (D883, D901).
- **Defectos propios:** las 12 filas de `union`, `intersect` y `except` de listas; el texto ya no dice que las resuelve el 018.
- **Defectos de las referencias:** las filas `-12:00` citan su espejo (D187/D188).
- **El backlog 1c41 se cierra:** D467, D475, D515 y D523 son deliberadas.

## Validación (paso 7, 08-11-2026)

- **`go test ./...`** pasa con fhirpath v1.16.7. Las áreas no cambian ninguna respuesta. `timezoneoffset` cambia, con sus tests.
- **Diferencial contra main (995bf95):** idéntica en los nueve corpus (`sh scratchpad/r016/diff.sh scratchpad/d008`).
- **eCQM por caso:** `engine_ecqm_dump_test.go`, nuevo, escribe la respuesta tipada de cada definición de cada caso, con timestamp fijo y a -03:00 y +14:00. El volcado de main y el de la rama son idénticos: 2226 respuestas en 67 casos, sin errores (`ECQM_CONTENT_DIR=… ECQM_DUMP=… go test -run TestECQMDump .`, con el archivo copiado a una copia de main).
- **Mutaciones:** las 9 de `scripts/mutations/018-default-offset.sh` se detectan.

## Code review, ronda 1 (Codex, 08-11-2026)

| # | Severidad | Hallazgo | Resolución |
|---|---|---|---|
| 1 | media | El volcado eCQM trunca a 120 caracteres y escribe solo `String()`. Así, un String `"false"` y un Boolean `false`, o un DateTime ubicado y uno sin ubicar, se vuelcan igual. | **Aceptado, corregido.** `dumpValue` escribe el tipo en cada nivel y, en un DateTime, su texto y el offset en que está ubicado, sin truncar. |
| 2 | media | El volcado pasa sin evaluar ningún caso: con un corpus inexistente escribió 0 respuestas. | **Aceptado, corregido y verificado:** con un corpus inexistente, ahora falla. Exige 67 casos y 2000 respuestas o más. Una medida sin Measure o un caso sin Patient quedan como una línea del volcado. |
| 3 | baja | El test estático no ve un valor de método (`place := dt.WithDefaultOffset`). | **Aceptado, corregido:** ahora revisa toda selección del método. La mutación de Codex en `conformance/output_parser.go` falla ahora con «selects dt.WithDefaultOffset». |

Después de las correcciones, los volcados de main (995bf95) y de la rama siguen idénticos: 2232 líneas en dos offsets, con 6 omisiones registradas.

## Code review, ronda 2 (Codex, 08-11-2026)

Codex confirmó que las tres correcciones de la ronda 1 se sostienen.

| # | Severidad | Hallazgo | Resolución |
|---|---|---|---|
| 1 | media | `dumpValue` sigue juntando respuestas distintas: no escribe la versión de un Code, no separa un campo nulo de uno vacío, no escribe el tipo FHIR de un objeto ni la escala de un Decimal calculado. Codex cambió las versiones de los Codes del corpus y el volcado salió idéntico. | **Aceptado, corregido.** Code y Concept se escriben campo por campo, con null aparte; un objeto FHIR, con su `Type()` y su JSON; un Decimal, con su exponente, que existe en las dos versiones de fhirpath. `TestDumpValueTellsAnswersApart` (`engine_ecqm_dump_value_test.go`, aparte para que el volcado compile en main) fija los pares que antes salían iguales. Los volcados de main y de la rama siguen idénticos. |

## Code review, ronda 3 (Codex, 08-11-2026): convergió

No quedan hallazgos de severidad media o mayor, y la corrección de la ronda 2 se sostiene: si se cambian las versiones de los Codes del corpus, cambian 134 líneas del volcado, y si se cambian los tipos de los objetos, cambian 4.

Quedaba un hallazgo bajo: `TestDumpValueTellsAnswersApart` no fallaba si el volcado dejaba de escribir el tipo de un objeto o el exponente de un Decimal. **Corregido:** el test tiene ahora un par de objetos con el mismo JSON y distinto tipo, y uno de Decimals calculados con distinta escala. Las dos mutaciones fallan ahora con «two different answers dump alike».

## Cierre (09-11-2026)

`make check` en verde tras la ronda 3: lint sin issues, la suite completa y todas las mutaciones detectadas (log en `scratchpad/r018c/makecheck3.log`). Faltan el commit y el PR, que se hacen solo cuando el usuario los pida.
