# 016: la incertidumbre de una duración

Backlog del 015 (D46, D266 y D297 de `durations`). Rama: `fix/duration-uncertainty`. Estado: implementado el 31-10-2026 (código, tests, área `uncertainty`, mutaciones); falta el cierre con `make check` y la revisión. Alcance decidido por el usuario el 31-10-2026: los tres defectos, sin cambiar la regla.

## Por qué

El 015 dejó D46, D266 y D297 como defecto propio diferido: el motor da 2 donde cql-execution da `[1, 2]`. La hipótesis era que la regla de `DurationBetween` para decidir cuándo hay incertidumbre estaba mal, apoyada en el ejemplo `[6, 7]` de la especificación.

## Revisión previa (31-10-2026, scratchpad `r016/`)

**gofhir/server** (c83d659b): no calcula duraciones. Solo recorre `DurationBetween` en el AST para `$data-requirements`, así que no hay nada que tomar.

**La especificación** (CQL 1.5.3):
- Author's Guide, Date and Time Arithmetic: «if either date or time value involved is not specified to the level of precision for the duration or difference being calculated, the result will be an uncertainty».
- Language Semantics, Determining Difference and Duration: `days between Date(2014, 1, 15) and Date(2014, 2)` es `[17, 44]`. `days between @2017-08-07T17:00 and @2017-08-14T` es `[6, 7]`, «because the starting datetime value is specified to the minute, 17:00, while the second value is only specified to the day». `days between @2012-01 and @2012-02` es `[1, 59]`. `hours between @2012-01-01T01:00:00 and @2012-01-01T02:00:00.0` es 1 (segundos y milisegundos son una sola precisión).
- La suite cql-tests: `years between DateTime(2005) and DateTime(2010)` es `Interval[4, 5]`, aunque los dos operandos llegan a la unidad.

**La regla actual** (`funcs/datetime_advanced.go`, `DurationBetween`): hay incertidumbre cuando un operando está por debajo del milisegundo y alguno no es más fino que la unidad. Da todos los ejemplos de arriba, y Java 5.4.0 coincide con ella en cada caso donde los dos difieren de cql-execution.

**cql-execution 3.3.2** lee cada DateTime incompleto como un rango y siempre calcula las dos cotas. También da todos los ejemplos de arriba. Las dos lecturas difieren solo donde ni la especificación ni la suite tienen un ejemplo: los dos operandos más finos que la unidad, a la misma precisión o a precisiones distintas, o un Date junto a un DateTime.

Sondas (área `probe016` en una copia del repo, contra las dos referencias):

| Expresión | Motor | Java | cql-execution |
|---|---|---|---|
| `days between DateTime(2012, 1, 1) and DateTime(2012, 1, 3)` | `[1, 2]` | 2 | `[1, 2]` |
| `years between DateTime(2012, 2, 29) and DateTime(2014, 2, 28)` (D46) | 2 | 2 | `[1, 2]` |
| `days between @2017-08-07T17:00 and @2017-08-14T` | `[6, 7]` | 6 | `[6, 7]` |
| `days between @2012-01 and @2012-02` | `[1, 59]` | null | `[1, 59]` |
| `years between @2000-01-01 and @2019-01-01T00:00:00.000` | 19 | 19 | `[18, 19]` |
| `days between @2012-01-01T09 and @2012-01-03T09:30` | 2 | 2 | `[1, 2]` |
| `hours between @2012-01-01T01:00 and @2012-01-01T03:00:00` | 2 | 2 | `[1, 2]` |
| `days between @2012-01-01T23:59:59.999 and @2012-01-03` | **0** | 1 | `[1, 2]` |

**Conclusión:** la hipótesis del 015 no se sostiene. La regla actual no contradice ningún texto ni test de la especificación. D46, D266 y D297 son una ambigüedad de la especificación, no un defecto propio, y se reporta como tal (principio 10, punto 4).

**Defectos reales encontrados** (medidos en main con `evalConversion`):
1. **`between`, `in` y `contains` fallan con una incertidumbre:** `(years between @2012 and @2014) between 0 and 5` da el error «cannot compare Interval», cuando `=` y `>=` dan null como corresponde. `evalBetween` arma un `Interval` y llama a `Contains`, que no acepta una `Uncertainty`. Las 21 medidas de edad del corpus eCQM usan `AgeInYearsAt(...) between a and b`.
2. **Las edades nunca son inciertas:** `CalculateAgeInYearsAt(@1965, @2019-01-01)` da 54. Las edades llaman a `YearsBetween`/`MonthsBetween` y no a `DurationBetween`. Para la especificación, CalculateAgeAt es una duración, así que debería dar `[53, 54]`. Si se arregla sin el punto 1, una edad incierta en `between` rompe la población entera con un error.
3. **Un Date junto a un DateTime en el camino incierto da una cota fuera de rango:** `days between @2012-01-01T23:59:59.999 and @2012-01-03` da 0, y con `Z` da 1. El rango de un Date se arma en UTC, y el DateTime sin offset va al offset por defecto.

**Impacto medido en eCQM** (`TestECQMCases` contra cqframework/ecqm-content-r4 022b8b1): en main, 23 casos coinciden y 1 está excusado de 24 con oráculo. Con las edades por `DurationBetween` y la regla actual, el resultado es igual, porque el corpus no tiene fechas de nacimiento parciales. Con la lectura de cql-execution (un Date junto a un DateTime da incertidumbre), 4 casos pasan a error: los 12 de 78 pacientes nacidos un 1 de enero tienen edad incierta al inicio del Measurement Period, y `between` no la acepta.

## Decisión del usuario (31-10-2026)

Se corrigen los tres defectos y la regla de incertidumbre de `DurationBetween` se queda como está. D46, D266 y D297 se reclasifican como ambigüedad de la especificación, y se redacta un pedido de aclaración a HL7, que no se publica sin el visto bueno del usuario.

## Alcance (final, tras la ronda adversarial y la división del 31-10-2026)

El usuario dividió el alcance en dos el 31-10-2026. Las edades por `DurationBetween` y el Date junto a un DateTime (los defectos 2 y 3 de la revisión previa, y el hallazgo 3 de la ronda) pasan al 017. El 016 queda así:

1. **Una incertidumbre como operando puntual.** Language Semantics, Uncertainty: una comparación con una incertidumbre da true si se cumple para todos sus valores, false si no se cumple para ninguno y null si depende. Para una incertidumbre `U = [l, h]` y un punto o intervalo de enteros:
   - `U between a and b` es `U >= a and U <= b` (así lo traduce cql-to-elm), y `properly between` es `U > a and U < b`;
   - `U in I`, `U during I`, `U included in I`, `I contains U` e `I includes U`: el punto contra cada límite de `I`, abierto o cerrado, con la lógica de tres valores (`and`);
   - `U before x`, `U after x`, `U same or before x` y `U same or after x` contra un punto: como `<`, `>`, `<=` y `>=`. El traductor de referencia las resuelve entre dos Integer y cql-execution las evalúa igual que el motor (verificado el 31-10-2026 en `uncertainty.js.json`). **`U same as x` no:** el traductor no tiene sobrecarga de `same as` para dos Integer («Could not resolve call to operator SameAs»), así que el motor no le agrega nada. Que hoy la evalúe es un defecto propio diferido (31 filas de `uncertainty`);
   - `U in {…}` y `{…} contains U`: la pertenencia a una lista por igualdad con cada elemento, que con una incertidumbre da null donde puede coincidir;
   - el error «cannot compare Interval» y las respuestas false silenciosas desaparecen en todos esos caminos. Una incertidumbre no se lee nunca como un `Interval` ordinario.
2. **La comparación de dos incertidumbres.** `[a, b] < [c, d]` es true si `b < c`, false si `a >= d`, null si no; igual con `<=`, `>`, `>=` y `=`. Hoy `[1, 2] >= [0, 1]` da null, y es true.
3. **El rango de un DateTime incompleto se arma en el marco del propio valor.** `temporalRange` corre el fin del día, del mes o del año fuera del marco del valor, y el ejemplo normativo `days between @2017-08-07T17:00 and @2017-08-14T` da `[5, 6]` con algunos offsets de evaluación. Debe dar `[6, 7]` en todos.
4. **Segundos y milisegundos son una sola precisión** en la regla de `DurationBetween` (Language Semantics: «consider seconds and milliseconds as a single combined precision»). `seconds between @2012-01-01T01:00:00 and @2012-01-01T01:00:01.0` da 1, no `[0, 1]`.
5. **D46, D266 y D297** de `durations` pasan de defecto propio diferido a diferencia deliberada contra cql-execution, citando la revisión previa. Se redacta el pedido de aclaración a HL7 (especificación CQL) con la tabla de sondas, y no se publica sin el visto bueno del usuario.
6. **Área nueva `uncertainty`** en snapshots, contra Java 5.4.0 y cql-execution 3.3.2, con los casos de los puntos 1 a 4.

## Fuera del alcance (al backlog)

- **Rechazar `same as` entre dos puntos Integer** (sema), que hoy el motor evalúa sin sobrecarga. Son 31 filas de `uncertainty` como defecto propio.
- **Hallazgos de las rondas 1 y 2 del review fuera del alcance:** un intervalo dentro de un intervalo con una cota incierta (debería dar true, da error como en main); un tipo que no concuerda junto a una incertidumbre, que sema debería rechazar; y de la ronda 1: las frases de orden entre una incertidumbre y un intervalo, una lista que contiene una incertidumbre, y las frases que Integer no tiene (`includes`, `overlaps`, `starts`, `meets` con una incertidumbre como intervalo).
- **El 017:** las edades por `DurationBetween` (fechas de nacimiento parciales, Today() o Now() según la sobrecarga del traductor, con los ELM capturados) y el Date junto a un DateTime leído como ToDateTime (`days between @2012-01-01T23:59:59.999 and @2012-01-03` da 0).
- Cambiar la regla de cuándo hay incertidumbre (la lectura de cql-execution).
- Min, Max y `sort` con una incertidumbre: hoy dan error. Un test fija ese error.
- La aritmética con una incertidumbre (`(years between @2012 and @2014) + 1`).
- Los límites null de un intervalo (G3i y G3j de 1c25): contra una incertidumbre siguen la regla que el motor ya aplica a un punto.
- `difference in … between` y `CalculateAgeIn{Hours,Minutes,Seconds}At`.

## Diferencias deliberadas previstas

- **Contra Java:** Java casi nunca produce una incertidumbre en una duración (D43 del 015), así que los casos del área que la producen divergen de Java. Se clasifican como defecto de la referencia cuando la especificación da la incertidumbre, con la cita.
- **Contra cql-execution:** D46, D266 y D297, y sus equivalentes en el área nueva (dos operandos más finos que la unidad), deliberados por la decisión del usuario, citando la revisión previa.

## Criterio de salida

- Ninguna expresión de los puntos 1 a 4 termina en error ni en una respuesta que contradiga el oráculo de extremos.
- En `uncertainty` y en `durations`, cada divergencia está clasificada con su evidencia.
- `TestECQMCases` contra ecqm-content-r4 022b8b1 sigue en 23 casos coincidentes y 1 excusado de 24 con oráculo.
- La diferencial contra main sobre el corpus de la semilla 8008: cada cambio se explica por los puntos 1 a 4.
- `make check` en verde, con las mutaciones nuevas detectadas.

## Tests

- **Escritos a mano, con la cita de la especificación:** los ejemplos de Language Semantics (`[17, 44] > 2` true, `> 50` false, `> 20` null), `[6, 7]`, `[17, 44]`, `[1, 59]` y el de segundos y milisegundos; cada operador del punto 1 con la incertidumbre adentro, afuera y a caballo de cada límite, abierto y cerrado; los casos de la ronda (`U during Interval(0, 2)`, `Interval[0, 2) includes U`, `U before 2`, `U in {1, 2}`, `[1, 2] >= [0, 1]`).
- **Oráculo de extremos, independiente de los comparadores del motor:** escrito en el test desde la fórmula de la especificación. Sobre una grilla de incertidumbres `[l, h]`, puntos, intervalos abiertos y cerrados y pares de incertidumbres (cientos de combinaciones), el valor esperado es true si la relación se cumple para todos los valores enteros del rango, false si no se cumple para ninguno y null si no. El test enumera esos valores en Go; no evalúa CQL para obtener lo esperado.
- **Metamórfico, como coherencia interna:** `U between a and b` igual a `U >= a and U <= b`, `U in Interval[a, b)` igual a `U >= a and U < b`.
- **Offsets:** los ejemplos normativos con la evaluación en +14:00, +05:45, 00:00, −03:00 y −12:00.
- **Lo que queda en error:** Min, Max y `sort` con una incertidumbre, fijados por un test.
- **El corpus eCQM:** `TestECQMCases` antes y después.

## Mutaciones (previstas)

En `scripts/mutations/016-uncertainty.sh`: `between` sin la rama de la incertidumbre; un límite abierto leído como cerrado; las cotas intercambiadas en la comparación de dos incertidumbres; `temporalRange` de vuelta al instante corrido; segundos y milisegundos separados otra vez; la pertenencia a una lista que vuelve a dar false; el área fuera del manifiesto.

## Ronda adversarial (Codex, 31-10-2026)

Solo Codex, sobre la copia `/private/tmp/claude-501/cql016` (sondas en `tmp/review016-*.txt`). Verifiqué cada hallazgo leyendo esas salidas.

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| 1 | El ejemplo normativo `days between @2017-08-07T17:00 and @2017-08-14T` da `[5, 6]` en vez de `[6, 7]` con algunos offsets de evaluación (−12, y el local de la copia). `temporalRange` arma el fin del día a partir de un instante corrido de marco | alta | Aceptado: el punto 3 se amplía a todo DateTime incompleto (cada rango se arma en el marco del propio valor), con los ejemplos normativos probados en +14, +05:45, 0, −03 y −12 |
| 2 | Una incertidumbre pasa por un intervalo en otros operadores, y la respuesta sale mal en silencio. Con `U = years between @2012 and @2014` (`[1, 2]`): `U during Interval(0, 2)`, `Interval[0, 2) includes U`, `U before 2` y `U in {1, 2}` dan false, y deberían dar null. Min y `sort` dan error | alta | Aceptado: el punto 1 cubre `in`/`during`/`included in`/`includes`/`contains`/`between` (también `properly`) con un punto incierto, `before`/`after`/`same as` contra un punto, y la pertenencia a una lista. Min, Max y `sort` con una incertidumbre van al backlog, con un test que fija el error actual. Product ya da un error deliberado |
| 3 | Con el punto 3, `AgeInDays() = AgeInDaysAt(Now())` puede pasar de true a null: un `birthDate` Date convertido da una incertidumbre en días, y dos incertidumbres iguales se comparan como null | alta | Aceptado: la consecuencia se declara, y se capturan los ELM del traductor de Java (con `using FHIR`) para AgeIn…() y AgeIn…At(start of "Measurement Period"). La sobrecarga que elige el traductor (Date con Today() o DateTime con Now()) es la que sigue el motor, y los tests llevan umbrales en días y semanas con fechas completas |
| 4 | Los tests de edad tienen más supuestos que uno: `TestTheWholeAgeMatrixAgrees` iguala AgeInDays() con CalculateAgeInDays(Date), y `TestTheFourUnitsStayCoherentOverAWideSweep` lee todo como entero | media | Aceptado: se revisan esos tres tests. Donde la respuesta pase a ser una incertidumbre, se comparan las cotas, con los valores que den las referencias |
| 5 | El metamórfico no es independiente, porque reusa los comparadores, y estos ya fallan: `[1, 2] >= [0, 1]` da null, y por la fórmula de Language Semantics es true | alta | Aceptado: un oráculo de extremos escrito en el test desde la fórmula de la especificación (true si se cumple para las dos cotas extremas, false si no se cumple para ninguna, null si no), sobre una grilla con uno y con dos operandos inciertos. La comparación de dos incertidumbres entra al alcance |
| 6 | Las referencias de `probe016` llegan solo a D29 y su hash no coincide con el CQL, que tiene D30–D37 | media | Aceptado: la tabla de la revisión previa cita solo D0–D29, que sí corresponden a esos JSON. Toda la evidencia del incremento sale del área `uncertainty`, regenerada con `scripts/references.sh` |

**Encontrado al verificar (31-10-2026):** `seconds between @2012-01-01T01:00:00 and @2012-01-01T01:00:01.0` da `[0, 1]`. Language Semantics dice que «uncertainty calculations … consider seconds and milliseconds as a single combined precision», así que es 1. La regla actual trata los segundos como una precisión aparte. Entra al alcance (punto 4 del alcance final).


## Code review

**Excepción a la regla del usuario (02-11-2026):** la regla manda que los reviews los haga solo Codex. Codex llegó a su límite de uso (vuelve el 04-11-2026 a las 03:52), y el usuario decidió que, hasta que Codex vuelva, los reviews los haga un revisor Opus independiente, sin el razonamiento del orquestador.

Antes de la ronda 1: `make check` en verde (508 mutaciones detectadas, 0 issues de lint), la diferencial contra main idéntica en las nueve áreas, `TestECQMCases` en 23 coincidentes y 1 excusado de 24, y los tests nuevos de comportamiento fallan en main.


### Ronda 1 (Opus, 02-11-2026)

El revisor no encontró nada de severidad alta. Comprobó que el emparejamiento de cotas en `compareIntervalWithScalar` es correcto en los cuatro operadores, que todos los caminos de `DurationBetween` (incluido `duration in X of` y el ELM) pasan por `DurationBetweenAt`, que los tests tocados pasan en Kiritimati, Midway y Kathmandu, y que el oráculo de la grilla es independiente (enumera en Go). Verifiqué cada hallazgo con una sonda propia en el repo.

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| 1 | Las frases de orden entre una incertidumbre y un **intervalo** siguen leyéndola como intervalo: `U before Interval[2, 3]`, `U after Interval[0, 1]`, `Interval[2, 3] after U` y `U same or before Interval[1, 3]` dan false, y es null | media | Confirmado (igual en main). Fuera del alcance escrito: el punto 1 cubre las frases de orden «contra un punto». Por la regla de no ampliar en un arreglo de review, va al backlog |
| 2 | Una lista que **contiene** una incertidumbre: `1 in { U }`, `{ U } contains 1` y `1 in { U, 5 }` dan false, y es null | media-baja | Confirmado (igual en main). Fuera del alcance escrito: el punto 1 cubre `U in {…}`, con U como el valor buscado. Va al backlog |
| 3 | Huecos de tests: con `evalBetween` mirando solo el operando, sobreviven las mutaciones, porque nadie fija `5 between U and 10` ni `2 properly between 0 and U`; tampoco `U in { 5, null }` | baja | Aceptado. Al agregar esos casos a mano apareció un defecto dentro del alcance que el revisor no vio, porque solo probó el AST: por ELM, `5 between U and 10` llega como `5 included in Interval[U, 10]`, y `uncertainTiming` no miraba los límites del intervalo, así que daba «type error in comparison». Corregido con `holdsUncertainty`. Los casos nuevos pasan por AST y por ELM, y hay dos mutaciones nuevas (19 en total, todas detectadas) |
| 4 | Una incertidumbre sigue leyéndose como intervalo en frases que Integer no tiene (`U includes 1`, `U overlaps …`, `U starts`/`meets`, `Interval[1, 2] included in U`), y comparaciones imposibles se tragan como null (`U during Interval[@2012, @2014]`) | baja | Igual en main, fuera del alcance. Va al backlog, junto con `same as` |
| — | D297 decía «this engine keeps its rule», y una edad no pasa todavía por esa regla | baja | Corregido el texto: el valor es el mismo, y la edad es del 017 |


### Ronda 2 (Opus nuevo, 02-11-2026)

El revisor es nuevo, distinto del de la ronda 1, y recibió el diff completo y el plan, sin el razonamiento del orquestador. Probó cada caso por AST y por ELM, y verificó: cotas abiertas, cerradas y nulas; incertidumbres negativas; operandos invertidos; variantes `properly`; listas con null; consultas; puntos Decimal y Quantity; dos incertidumbres por sus extremos; segundos y milisegundos; y los tests tocados en Kiritimati, Midway y Kathmandu. Las 19 mutaciones se detectan. Revisó a mano 7 filas de la clase de número único. Verifiqué cada hallazgo con una sonda propia.

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| 1 | Un intervalo preguntado contra un intervalo del autor con una cota incierta se leía como punto y daba null: `Interval[U, 5] includes Interval[2, 3]`, `Interval[2, 3] during/included in/in Interval[U, 5]`. Lo introdujo el arreglo de la ronda 1 (`holdsUncertainty`); main daba error | media | Confirmado y corregido: el caso de intervalo de `uncertainContained` exige `pointOrUncertainty(x)`, así que estos casos vuelven al camino de main. Lo fija `TestAnIntervalInAnIntervalWithAnUncertainBoundIsNotAPoint`, por los dos caminos, con su mutación. Que respondan true, como da la especificación, queda fuera del alcance: va al backlog con los hallazgos 1, 2 y 4 de la ronda 1 |
| 2 | Huecos de tests: `duration in X of` de vuelta a UTC, `includes` sin `properly` y la guarda de la lista de un elemento sobrevivían a mutaciones a mano | baja | Aceptado: tres casos a mano (`duration in days of Interval[@2017-08-07T17:00, @2017-08-14T]` = [6, 7] a −12:00; `Interval[1, 3] properly includes U` = null; `{ 1 } properly includes U` = false) y tres mutaciones. Son 23 en total, todas detectadas |
| 3 | Un tipo que no concuerda ahora da null: `(days between @2012-01-01 and @2012-02) in Interval[@2012-01-01, @2012-02-01]`, donde main daba «cannot compare Interval». El traductor de referencia no tiene sobrecarga | baja | Confirmado. Lo correcto es que sema lo rechace, igual que `same as` entre Integers; va al backlog junto con ese defecto propio |


### Ronda 3 (Opus nuevo, 02-11-2026)

La primera ejecución se cortó por un error de conexión con la API, sin informe, y no cuenta como ronda. La segunda, con otro Opus y una copia limpia, probó unas 180 expresiones por los dos caminos, en la rama y en main. Comprobó que las cotas inciertas, las consultas, los puntos Decimal, los extremos de dos incertidumbres y el marco funcionan; que pasan Kiritimati y Midway; que lint da 0 issues; y que las 23 mutaciones se detectan. Verifiqué cada hallazgo con una sonda propia.

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| 1 | `properly` con una cota abierta: `U properly during Interval(0, 3)` da true y `U properly included in Interval[1, 3)` da null; main daba false en ambos. La especificación (ProperIn de punto) compara contra Start y End, que para `Interval(0, 3)` son 1 y 2 | media-baja | Confirmado. La causa es anterior al 016: la regla de punto de main ya da `1 properly included in Interval(0, 3)` = true y `2 properly included in Interval[1, 3)` = true. Se llegó al tope de 3 rondas. **Decisión del usuario (02-11-2026):** cerrar el 016 y mandarlo al backlog (1c26 en `README.md`), con diferencial contra Java antes de tocarlo |
| 2 | `U less than 1 before 2` da false por AST y null por ELM (main daba false en los dos): el AST salta la frase con offset, y el ELM llega como `in` de un intervalo abierto | baja | Confirmado. Por el álgebra de la especificación (`U > 1 and U < 2`, cada una null), null es la lectura del traductor; la disparidad entre caminos va al backlog |
| 3 | Huecos de tests: `U in Interval[null, 2]` sin fijar, y la guarda del offset en `uncertainTiming` | baja | Al backlog (1c27), con el resto de lo que el review dejó fuera |
| 4 | El plan decía 521 divergencias (son 520) y 17 mutaciones (son 23) | baja | Corregido |


### Cierre (02-11-2026)

- Rondas: 3 (Opus independiente cada una, por la falta de cuota de Codex). La ronda 3 dejó un hallazgo media-baja con causa anterior al 016; por decisión del usuario, va al backlog (1c26) y el 016 cierra.
- `make check` en verde sobre el código final (`scratchpad/r016/check016f.log`: exit 0, 514 mutaciones detectadas, 0 issues de lint). Después solo cambió este plan y el README.
- Diferencial contra main después del review: idéntica en las nueve áreas (`scratchpad/r016/diff.sh`).
- Mutaciones del 016: 23, todas detectadas (`scratchpad/r016/mut016f.log`).

## Dependencias

Ninguna nueva.

## Implementación (31-10-2026)

- **Punto 1 y 2:** `eval/uncertain_membership.go`. Una incertidumbre es un punto de valor desconocido: `uncertainWithin` la pregunta contra cada límite de un intervalo (abierto o cerrado, `properly` estricto) con `and` de tres valores; `uncertainInList` compara con cada elemento por `=`; `uncertainTiming` despacha `in`/`during`/`included in`/`includes` y `before`/`after`/`same as`/`same or before`/`same or after`, y no toca las frases con precisión, palabra de frontera u offset. Los ganchos están en `evalBetween`, `evalInContains` (`in`/`contains`) y `evalTimingExpr` (antes de la rama de listas). `compareIntervalWithScalar` acepta una incertidumbre como segundo operando: el mínimo del primero contra el máximo del segundo, y el máximo contra el mínimo, que para un punto es la misma comparación de antes.
- **Punto 3:** `funcs.DurationBetweenAt` recibe el offset de la evaluación (`Evaluator.evaluationFrame`, el mismo que reciben los literales) y `temporalRange` arma ahí el rango de un DateTime sin offset propio. `DurationBetween` conserva su firma y usa UTC. Primera idea descartada: tomar el marco del otro operando; no es el marco de la especificación cuando el otro operando declara un offset distinto del de la evaluación.
- **Punto 4:** `valuePrecisionIndex` da el índice del milisegundo al segundo.
- **Punto 5 y 6:** abajo.

## Encontrado en los tests

- fhirpath no pone offset a un DateTime por debajo de la hora (`WithDefaultOffset` lo ignora: «@2020 has no instant to place»), así que el offset de la evaluación no se alcanza desde el valor y hubo que pasarlo desde el evaluador. No es un rodeo de fhirpath: es contexto del propio motor.
- `Interval[a, a)` es un intervalo inválido, y `Interval(a, a + 1)` no tiene enteros: la tercera lógica de `x > a and x < a + 1` da null sobre una incertidumbre, donde la enumeración de valores daría false. La grilla evita los intervalos vacíos; `U properly between 1 and 2` es null por la definición del operador (`> and <`), y el oráculo de la grilla compone los dos resultados de tres valores en vez de enumerar los valores estrictamente entre ambos. Lo mismo con `U in { 1, 2 }`: por la definición de `In` sobre listas (igualdad con cada elemento) es null, no el true de «todo valor es miembro».
- La frase `same as`, `before`, `after` sobre Integer no la traduce Java (translation error) y cql-execution solo corre lo que el traductor acepta: por eso el JSON de cql-execution tiene 1206 resultados contra 1237 de Java (31 `same as`).

- **Dos tests viejos suponían UTC sin decirlo (31-10-2026, `make check`, paso `zones`):** `TestDurationBecomesUncertain` y `TestSumMixesCertainWithUncertain` evalúan un dateTime FHIR escrito al día o al mes (`"2020-03-01"`), y dejaban la evaluación en la zona del equipo. Con el punto 3 ese valor se lee en el offset de la evaluación, así que en Pacific/Kiritimati (+14:00) `duration in days of First([Encounter]).period` da `[4, 5]` y no `[3, 4]`. Las dos cuentas son correctas para su offset (de 2020-03-01 a las 00:00+14:00 hasta 2020-03-05T10:00Z hay 5 días). Los helpers de esos tests fijan ahora la evaluación en UTC con `WithEvaluationTimestamp`, y sus valores esperados, que son los de UTC, no cambian. Pasan en UTC, America/Santiago, Asia/Kathmandu, Pacific/Kiritimati y Pacific/Midway.

## Área `uncertainty`: clasificación

1237 definiciones (1235 más D1235 y D1236, el marco de la evaluación con un offset declarado: `days between @2017-08-07T07:59+05:00 and @2017-08-14T` evaluado a -03:00). Contra cql-execution: `[7, 8]` y `[-8, -7]`, confirmando los valores derivados de la lectura de la especificación; Java da 6 y -6. 520 divergencias, todas con veredicto:

| Clase | Filas | Qué es |
|---|---|---|
| reference defect, Java | 224 | Java responde true o false desde un solo número donde la duración es una incertidumbre; el motor da la respuesta de la definición |
| reference defect, Java | 159 | Java responde null porque lee la incertidumbre como un intervalo que no puede comparar; el motor, true o false por la definición |
| reference defect, Java | 56 | `before`, `after`, `same or before` y `same or after` con una incertidumbre: Java falla en tiempo de ejecución o responde desde un solo número; cql-execution coincide con el motor |
| reference defect, Java | 14 | pertenencia a una lista: Java responde un Boolean donde un elemento puede coincidir |
| reference defect, Java | 12 | la duración: Java da un número, null o un Interval donde la especificación da la incertidumbre |
| reference defect, Java | 8 | dos incertidumbres comparadas: Java falla |
| reference defect, Java y JS | 2 | `milliseconds between` con segundos (un solo número, regla del punto 4) |
| reference defect, JS | 14 | pertenencia a una lista: cql-execution da false |
| own defect, Java | 31 | `same as` entre dos puntos Integer: el traductor de referencia no lo resuelve y el motor lo evalúa |

Cifras del 31-10-2026, de `awk -F'\t' 'NR>1{print $2, $3, substr($5,1,60)}' testdata/snapshots/uncertainty.divergences.tsv | sort | uniq -c`.

*Corrección del 31-10-2026, al verificar:* el subagente había dejado 71 filas de `before`, `after`, `same as` y variantes como deliberadas. Las cuatro frases de orden se traducen y cql-execution coincide con el motor, así que son defectos de Java (Java falla en tiempo de ejecución o responde un número fijo) y entran en las filas de defecto de la referencia. Solo `same as` no se traduce.

**Qué referencia coincide con qué:** `between`, `properly between`, `in`/`during`/`included in`/`includes`/`contains` sobre intervalos, `before`/`after` y la comparación de una incertidumbre con un punto: cql-execution coincide con el motor en todos (no hay filas de JS), Java no. Pertenencia a una lista (`U in {1, 2}`, `{1, 2} contains U`): ninguna coincide cuando un elemento puede coincidir (cql-execution da false, Java un Boolean); se mantiene la composición de la definición del operador, por decisión del 31-10-2026. `properly between`: coincide cql-execution.

## Diferencias deliberadas

- **D46, D266, D297** de `durations` (los operandos más finos que la unidad, contra cql-execution): pasan de defecto propio diferido a diferencia deliberada, por la decisión del usuario del 31-10-2026 y la revisión previa de este plan (la especificación no da regla para ese caso, y ninguno de sus ejemplos ni la suite lo contradice).
- **Defectos propios diferidos, no deliberados:** las 31 filas de `uncertainty` en las que el motor evalúa `same as` entre dos puntos Integer, que no tiene sobrecarga.

## Pedido de aclaración a HL7 (borrador, sin publicar)

Para la lista de la especificación CQL (cqframework / HL7 CQL-FHIR). Título: «When is a duration an uncertainty if both operands are finer than the unit?».

La sección Determining Difference and Duration y el Author's Guide dicen que el resultado es una incertidumbre cuando un operando no está especificado a la precisión de la unidad, y dan ejemplos solo con un operando más grueso que la unidad (`days between @2017-08-07T17:00 and @2017-08-14T` es `[6, 7]`; `days between @2012-01 and @2012-02` es `[1, 59]`). La suite cql-tests da `years between DateTime(2005) and DateTime(2010)` como `Interval[4, 5]`, con los dos operandos a la unidad. No hay ejemplo ni regla donde ambos operandos son más finos que la unidad:

| Expresión | Java 5.4.0 | cql-execution 3.3.2 |
|---|---|---|
| `years between DateTime(2012, 2, 29) and DateTime(2014, 2, 28)` | 2 | `[1, 2]` |
| `days between @2012-01-01T09 and @2012-01-03T09:30` | 2 | `[1, 2]` |
| `hours between @2012-01-01T01:00 and @2012-01-01T03:00:00` | 2 | `[1, 2]` |
| `days between @2012-01-01T23:59:59.999 and @2012-01-03` | 1 | `[1, 2]` |

Preguntas: (1) ¿cuenta la ausencia de los componentes más finos que la unidad como incertidumbre aunque ambos operandos los tengan? (2) ¿Cómo se leen los segundos y los milisegundos (una sola precisión) en esa regla? (3) ¿Qué operadores se definen sobre una incertidumbre además de la aritmética y la comparación: `between`, `in`, `during`, `before`, `after`, pertenencia a una lista (hoy cada implementación contesta distinto)? (4) ¿Una incertidumbre iguala a otra idéntica (`[1, 2] = [1, 2]`: null, o true)?

Evidencia: el área `uncertainty` de `testdata/snapshots` y la tabla de la revisión previa. No se publica sin el visto bueno del usuario.

## Mutaciones (resultado, 31-10-2026)

`sh scripts/mutations/016-uncertainty.sh`: 17 mutaciones al cierre de la implementación (23 después de las rondas 1 y 2 del review, ver «Code review»), todas detectadas, ninguna sin aplicar ni sin compilar (registro en `scratchpad/r016/mut016c.log`, tras la verificación del 31-10-2026: se reancló la mutación de la fila deliberada no nombrada, que ahora hace citar a D46 un plan que no lo nombra, y la de las frases de orden, movida a `uncertainOrder` por el lint). Cubren: `between`, `in`/`contains` y las frases de tiempo sin la rama de la incertidumbre; `includes` fuera de la familia; un límite abierto leído como cerrado (bajo y alto); `properly between` leído como `between`; la lista que vuelve a dar false o que cuenta como desconocido un elemento que no puede coincidir; las frases de orden leídas como `>=`; las cotas de dos incertidumbres intercambiadas; el rango de un DateTime grueso de vuelta en UTC y sin el offset de la evaluación; segundos y milisegundos separados otra vez; una fila deliberada que el plan no nombra; y el área fuera del manifiesto o de las referencias. Min, Max y `sort` con una incertidumbre quedan fijados por `TestMinMaxAndSortOfAnUncertaintyStillRefuse` (sin mutación: el error ya es el comportamiento).
