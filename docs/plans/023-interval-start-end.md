# 023: `duration of` y `difference of` miden entre el inicio y el fin del intervalo

Backlog 1c33. Rama `fix/interval-start-end`. El 05-11-2026 el usuario eligió primero los dos defectos (1c33 y G3i). Después de la ronda adversarial 1 los dividió: el 023 queda para el 1c33, y el G3i, que necesita el tipo del punto en el valor, pasa al 024.

## Revisión previa (05-11-2026)

### Especificación (CQL 1.5.3, https://cql.hl7.org/N1A/)

- **Author's Guide, Interval Operators:** «duration in days of X / difference in days of X. These expressions are equivalent to: duration in days between start of X and end of X / difference in days between start of X and end of X».
- **Appendix B, Start:** «If the low boundary of the interval is open, this operator returns the successor of the low value of the interval. Note that if the low value of the interval is null, the result is null. If the low boundary of the interval is closed and the low value of the interval is not null, this operator returns the low value of the interval. Otherwise, the result is the minimum value of the point type of the interval».
- **Appendix B, End:** es simétrico, con el predecessor y el máximo.
- **Un límite abierto en null da null** en la 1.5.3. La incertidumbre y la cláusula «point type unknown» son de la 2.0.0 (ronda adversarial 1, hallazgo 1).
- **Appendix B, Minimum y Maximum:** Date va de 0001-01-01 a 9999-12-31, y DateTime y Time llegan hasta el milisegundo.

### Cómo lo resuelve el motor hoy (sondas en `scratchpad/r023`, 70 expresiones a -03:00)

- **`duration`/`difference in X of I`** (`eval/evaluator.go`, `*ast.DurationOf` y `*ast.DifferenceOf`) mide entre `iv.Low` e `iv.High` tal cual: ignora el cierre y lee el null como null. `duration in days of Interval(@2012-01-01, @2012-01-05)` da 4; las dos referencias, 2. `duration in days of Interval[@2012-01-01, null]` da null; las dos, 2917556.
- **El traductor propio** escribe un nodo `DurationOf`/`DifferenceOf` que no existe en ELM. El de Java escribe `DurationBetween(Start(I), End(I))`. Por el ELM de Java, la parte abierta ya sale bien; la parte null, no.
- **`types.Interval.Start()`/`End()`** aplican el successor o el predecessor a un límite abierto, pero devuelven null para un límite cerrado en null. `eval/interval_precision.go` ya tiene `startOf`/`endOf`, que leen el null cerrado como el extremo, pero solo para tipos temporales (`extremeLike`) y solo en las frases con precisión.
- **El tipo del punto no está en el valor:** Integer y Long son el mismo `fptypes.Integer`. Lo que los distingue es el tipo estático (`e.ctx.Plan.TypeOf`), que existe solo con plan, es decir, al compilar desde CQL. El importador de ELM no lee `resultTypeName`.

### Referencias

| Expresión | Motor | Java 5.4.0 | cql-execution 3.3.2 |
|---|---|---|---|
| `duration in days of Interval(@2012-01-01, @2012-01-05)` | 4 | 2 | 2 |
| `duration in minutes of Interval(@2012-01-01T10:00, @2012-01-01T12:00)` | `[119, 120]` | 118 | `[117, 118]` |
| `duration in months of Interval(@2012-01, @2012-05)` | `[3, 4]` | 2 | `[1, 2]` |
| `duration in days of Interval[@2012-01-01, null]` | null | 2917556 | 2917556 |
| `duration in hours of Interval[@T10:00, null]` | null | 13 | 13 |
| `end of Interval[1, null]` | null | 2147483647 | 2147483647 |
| `end of Interval[1L, null]` | null | 9223372036854775807 | null |
| `start of Interval[null, 5.0]` | null | -99999999999999999999.99999999 | -2147483648 |
| `end of Interval[1 'mg', null]` | null | 99999999999999999999.99999999 'mg' | 2147483647 'mg' |
| `end of Interval[@2012-01-01T00:00:00.000, null]` | null | 9999-12-31T23:59:59.999+00:00 | 9999-12-31T23:59:59.999-03:00 |
| `Interval[1, null] = Interval[1, 2147483647]` | null | true | true |
| `Interval[1, null] ~ Interval[1, 2147483647]` | false | true | false |
| `Interval[1, null] starts Interval[1, 5]` | true | false | null |
| `Interval[1, null] union Interval[5, 10]` | `Interval[1, 10]` | `Interval[1, 2147483647]` | `Interval[1, null]` |
| `collapse { Interval[1, null], Interval[3, 5] }` | `{Interval[1, null]}` | `{Interval[1, 2147483647]}` | `{Interval[1, null], Interval[3, 5]}` |
| `point from Interval[2147483647, null]` | error | 2147483647 | null |
| `width of Interval[null, 1]` | null | -2147483647 (vuelta en 32 bits) | 2147483649 |
| `end of Interval[@2012-01-01, null)` | null | null | null |
| `end of (Interval[null, null] as Interval<Integer>)` | null | null | null |

Java sigue la especificación en todos los casos salvo `width` (la vuelta en 32 bits, ya clasificada como defecto en el #135). cql-execution no lee el null cerrado como extremo en las operaciones que lo devuelven (union, collapse, Long y Decimal).

### gofhir/server

Congelado y sin CQL propio: no hay nada que tomar.

## Alcance

`duration of` y `difference of` solo existen para intervalos de Date, DateTime y Time, que en tiempo de ejecución son tipos distintos. Así, el tipo del punto lo dice el otro límite, y el problema del tipo que dejó la ronda 1 no aparece.

1. **`duration`/`difference in X of I`** miden entre `start of I` y `end of I` (Author's Guide).
   - Un límite abierto pasa por su successor o su predecessor. Si eso cae fuera del calendario, el resultado es null, por `stepWithinItsType`, como `start of`.
   - Un límite cerrado en null es el mínimo o el máximo del tipo del otro límite. Se usa `temporalExtreme`, el mismo valor que `minimum DateTime` y el resto.
   - Un límite abierto en null da null (1.5.3).
   - Si los dos límites son null, el intervalo llega como nil y el resultado es null. Esto queda para el 024.
2. **`start of` y `end of` de un intervalo temporal** leen el null cerrado de la misma manera.
   - Así, `duration in days of I` y `days between start of I and end of I` dan lo mismo, por el AST y también por el ELM de Java, que escribe `DurationBetween(Start(I), End(I))`.
   - Para Integer, Long, Decimal y Quantity siguen dando null hasta el 024.
3. **Un solo lugar:**
   - `startOf` y `endOf` de `eval/interval_precision.go` ya leen el null cerrado temporal como el extremo, con `extremeLike`.
   - `start of`, `end of`, `duration of` y `difference of` pasan a usarlos.
   - `stepWithinItsType` se aplica a lo que devuelven.

## Fuera del alcance (al 024 o al backlog)

- **El G3i para Integer, Long, Decimal y Quantity:**
  - el tipo del punto en el valor;
  - el constructor de `Interval[null, null]`;
  - la igualdad, `distinct`, `Size`, `collapse` y `union`/`except`.
  - Son los hallazgos 2 a 8 de la ronda 1, y van al 024.
- **`Interval[1, null] union Interval[5, 10]`**, que da `Interval[1, 10]`, va al 024.
- **Un intervalo que el compilador tipa `Interval<DateTime>` pero que en ejecución conserva un Date** (un `if` entre un intervalo Date y uno DateTime, o `Interval[@2012-01-01, @2012-01-05T00:00:00.000]`): el extremo sale de Date. Es el defecto del 019, Date al lado de DateTime (`sema/check.go:454` no convierte), y se resuelve ahí. Se deja un test que documenta la respuesta de hoy (ronda 2, hallazgo 1).
- **Nuestro traductor escribe un nodo `DurationOf`/`DifferenceOf`**, que no existe en ELM; Java escribe `DurationBetween(Start, End)`. Va al backlog como 1c37.

## Divergencias esperadas

- **`duration in minutes of Interval(@2012-01-01T10:00, @2012-01-01T12:00)`:**
  - el motor dará la regla de incertidumbre del 016 entre 10:01 y 11:59;
  - Java da 118 y cql-execution `[117, 118]`.
  - Es la misma diferencia que el 016 ya clasificó para `minutes between`.
- **El fin de un DateTime con null cerrado:**
  - Java lo pone en 9999-12-31T23:59:59.999Z y cql-execution a -03:00;
  - el motor usa UTC, como `maximum DateTime`.
  - El conteo en días coincide en las dos (2917556). En horas o menos puede diferir con cql-execution, y se clasifica fila por fila.

- **Un intervalo sin puntos** (`Interval(@2012-01-01, @2012-01-02)`): el motor da -1 en `duration` y `difference`, por la composición del Author's Guide (del 2 al 1 de enero). cql-execution da -1, y Java da error («Invalid Interval»). Es una divergencia deliberada contra Java (ronda 2, hallazgo 2).
- **Las horas hasta el fin de un DateTime con null cerrado:** `duration in hours of Interval[@9999-12-31T00:00:00.000-03:00, null]` da 20, como Java, porque el máximo está en UTC. cql-execution da 23, con el máximo a -03:00. Es una diferencia deliberada, no un defecto: la 1.5.3 (Maximum y Minimum) dice que las implementaciones «may choose to represent the maximum DateTime value using a constant offset such as UTC», así que las dos lecturas valen. El motor elige UTC, como su `maximum DateTime` y como Java (ronda 2, hallazgo 3; corregido en la validación del 05-11-2026, que antes lo daba como defecto de cql-execution).

## Criterio de salida

1. Las filas temporales de `scratchpad/r023/table.tsv` y `tableb.tsv` dan lo de la especificación por el AST, por nuestro ELM y por el ELM de Java:
   - `duration of`, `difference of`, `start of` y `end of`;
   - con límites abiertos y con null cerrado;
   - en Date, DateTime y Time.
2. Hay una nueva área `intervalbounds` con cada divergencia clasificada con evidencia.
3. La diferencial contra main sobre el resto del corpus solo cambia en filas explicadas. eCQM, corrido con `ECQM_CONTENT_DIR`, queda en 23 coincidentes y 1 excusado.
4. Todas las mutaciones se detectan y `make check` queda en verde.

## Tests y fuentes de datos

- **`scripts/intervalbounds-area.py`** genera unos 500 casos.
  - `duration in X of` y `difference in X of`, en todas las unidades de cada tipo.
  - Date, DateTime (con y sin offset) y Time.
  - Las cuatro formas de cada límite: cerrado, abierto, null cerrado y null abierto.
  - Precisiones distintas y el borde del calendario (successor de 9999-12-31).
  - `start of` y `end of` de los mismos intervalos.
- **Oráculo independiente en Python:**
  - calcula Start y End por la 1.5.3, con successor, predecessor y extremos;
  - cuenta los días y los meses por fecha juliana, como el 021;
  - cubre las filas sin incertidumbre;
  - cuenta las horas, los minutos y los segundos por instantes en UTC, con el máximo y el mínimo en UTC, como Java (ronda 2, hallazgo 3).
- **Controles:** el intervalo promovido de Date a DateTime (limitación del 019), un intervalo que llega por una query y por una biblioteca incluida, una duración incierta, el predecessor del mínimo (`Interval[null, @0001-01-01)`) y los bordes de Time.
- **Tests a mano** (`engine_intervalbounds_test.go`):
  - las filas de la tabla, por el AST, por nuestro ELM y por el ELM de Java (las capturas en `testdata/intervalbounds/`);
  - citan la 1.5.3 en N1A;
  - los valores salen de la especificación y coinciden con Java.

## Mutaciones previstas (`scripts/mutations/023-interval-start-end.sh`)

- `duration of` vuelve a leer `Low`/`High`, y lo mismo `difference of`.
- El null cerrado vuelve a null en el inicio, y también en el fin.
- El mínimo y el máximo se intercambian.
- El fin de DateTime pasa a la zona de la evaluación.
- `start of` deja de usar `startOf`.
- Se quita `stepWithinItsType`: el successor de 9999-12-31, y el predecessor de 0001-01-01 por la rama de End.
- El null abierto se lee como el extremo.
- Una divergencia queda sin clasificar, y una respuesta del área se mueve del oráculo.

## Dependencias

Ninguna nueva.

## Ronda adversarial 1 (Codex, 05-11-2026)

Codex hizo los diagnósticos sobre la copia `/private/tmp/claude-501/cql023`: `probes/audit-results.txt` y `probes/read-sites.txt`. Verifiqué cada hallazgo contra su salida ejecutada.

| # | Hallazgo | Sev. | Resolución |
|---|---|---|---|
| 1 | El plan cita la 2.0.0 como si fuera la 1.5.3. La página sin versión es la 2.0.0, y la copia del scratchpad también. En la 1.5.3, Start de un límite abierto en null da null, y no existe la cláusula «point type unknown» | alta | Aceptado. Las citas pasan a https://cql.hl7.org/N1A/. D3 deja de ser decisión: la 1.5.3 da null, como las dos referencias. También se corrigieron dos citas del borrador para HL7 (#177) |
| 2 | El constructor devuelve nil cuando los dos límites son null (`eval/evaluator.go:6056`), sin importar el cierre ni el tipo, y ningún lector de límites lo recupera | alta | Aceptado, verificado: `end of (Interval[null, null] as Interval<Long>)` da nil |
| 3 | El tipo «con que el plan resolvió la operación» no existe: sema da Boolean para `starts`, con operandos `Interval<?>` e `Interval<Long>` | alta | Aceptado, verificado |
| 4 | Con plan también se pierde Long: una llamada a otra biblioteca devuelve Unknown, y un condicional da una elección de tipos. El ELM importado conserva el literal Long en el AST | alta | Aceptado |
| 5 | La igualdad vive en `types.Interval.Equal`, sin tipo estático. `distinct`, `union` y `except` de listas, y la igualdad anidada, pasan por ahí | alta | Aceptado, verificado: `distinct {Interval[1, null], Interval[1, 2147483647]}` deja los dos |
| 6 | Faltan `Size` y dos filtros de `collapse`: `funcs/interval_set.go:16` y `:326`, y `eval/evaluator.go:8073` | alta | Aceptado, verificado: `collapse {Interval[1, 5], Interval[3, null]}` da `{Interval[1, 5]}` |
| 7 | D2 no se puede comprobar con el comparador de cql-tests, que usa el `Equal` de producción. `expand` permite saltear los intervalos con null | media | Aceptado |
| 8 | `maximum Quantity` da nil, así que `evalTypeExtent` no alcanza para el Quantity. Que el DateTime vaya en UTC es una política propia, no la especificación | media | Aceptado |
| 9 | La validación no cubre los caminos 2 a 6, y faltan sus mutaciones. Hay que actualizar `suitegaps`. eCQM no se corrió en la copia | media | Aceptado |

**Conclusión:** el tipo del punto no se puede recuperar donde se lee el límite. El diseño del alcance 1 no sirve como está.

## Ronda adversarial 2 (Codex, 05-11-2026)

Evidencia: `probes/round2*.{cql,txt,json}` y `eval/review023_round2_test.go`, en la copia. La verifiqué con su salida.

| # | Hallazgo | Sev. | Resolución |
|---|---|---|---|
| 1 | El otro límite no siempre dice el tipo: un intervalo `Interval<DateTime>` que en ejecución conserva un Date (promoción por `if`, por query, o límites mixtos) toma el extremo de Date. 24 horas contra `[21, 44]` de Java | alta | Aceptado como limitación explícita: es el defecto del 019 (`sema/check.go:454`), que se resuelve ahí. Hay un test que documenta la respuesta de hoy |
| 2 | Un intervalo sin puntos da -1. Java da error, cql-execution -1 | media | Aceptado: divergencia deliberada contra Java, por la composición del Author's Guide |
| 3 | El oráculo no cubre horas ni minutos, y UTC cambia conteos (20 contra 23 de cql-execution). Faltan controles y la mutación del predecessor | media | Aceptado: el oráculo cuenta por instantes en UTC; controles y mutación agregados |

Sin regresiones fuera del alcance: los 11 nodos de Start/End de `limits.cql` dan lo mismo por los dos caminos, incluida la certeza de Integer.

## Decisión del usuario (05-11-2026)

Plan aprobado como quedó después de la ronda 2.

## Validación (paso 7, 05-11-2026)

**Área `intervalbounds`:** 597 casos generados por `scripts/intervalbounds-area.py`, de los cuales 494 chequea el oráculo independiente (`TestTheIntervalBoundsAreTheirOracle`). Las dos filas de un FHIR Period salieron del área, porque el arnés no les da modelo FHIR a las referencias; quedaron en `TestAPeriodOpenAtTheEndLastsToTheLastInstant`.

**Divergencias:** 259 contra Java 5.4.0 y cql-execution 3.3.2, clasificadas fila por fila por un clasificador que verifica la evidencia de cada una (en `scratchpad/r023/classify.py`, no en el repo).

- **Defectos de las referencias (139):**
  - Java envuelve en 32 bits los conteos fuera de rango.
  - cql-execution da el número de JavaScript.
  - Java da un Interval donde va una incertidumbre, o un número o null donde un extremo es más grueso que la unidad.
  - Java cuenta las semanas por días/7; el motor cuenta los domingos, y el clasificador también.
  - cql-execution normaliza los offsets en años y meses (D364, D366), y falla o da un valor fuera del calendario donde no hay successor o predecessor.
- **Defectos propios (38), con su respuesta esperada:**
  - **1c32 (Time sin incertidumbre):** 35 filas. Es el mismo defecto anotado en el backlog, ahora con el rango esperado contado por el clasificador.
  - **1c28:** D370, porque en días el motor normaliza los offsets.
  - **019:** D590 y D591, la limitación de la promoción de Date a DateTime.
- **Diferencias deliberadas (82)**, decididas en este plan:
  - **Intervalo sin puntos** (ronda 2, hallazgo 2): D40, D41, D42, D43, D44, D45, D46, D47, D48, D49, D472, D473, D474, D475, D476, D477, D478, D479, D480, D481, D482, D483, D484, D485, D486, D487, D488 y D489. El motor mide de atrás para adelante y Java rechaza el intervalo.
  - **El máximo y el mínimo de DateTime en UTC**, que cql-execution pone al offset de la evaluación; la 1.5.3 permite las dos cosas («may choose … a constant offset such as UTC»): D372, D373, D381, D390, D391, D392, D393, D398, D408, D409, D410, D411, D412, D413, D414, D415, D417, D435 y D452.
  - **La regla de incertidumbre del 016** (los dos extremos más finos que la unidad): D246 y D490.
  - **La forma de un DateTime:** D236, D237, D254, D255, D272, D273, D290, D291, D308, D309, D327, D344, D362, D363, D381, D398, D416, D417, D435, D452, D470 y D471.
    - Es el mismo valor escrito de otra manera. Un literal sin offset tiene el de la evaluación (1.5.3, Literals), y las referencias lo imprimen (`-03:00`) mientras el motor lo deja implícito. `+00:00` es el offset `Z`.
    - **Confirmado por el usuario el 05-11-2026:** quedan como deliberadas. El 019 revisa si un literal sin offset debe llevar el de la evaluación, y si cambia, las reclasifica.

## Code review

### Ronda 1 (Codex, 05-11-2026)

Codex corrió las 19 mutaciones (las atrapa todas) y una propia: cambiar `DifferenceOf` por la duración, que atrapa el snapshot con 34 respuestas cambiadas. También comprobó que las 597 expresiones dan lo mismo compiladas que por el ELM del motor, y recalculó a mano las filas de la forma de un DateTime y los rangos esperados de Time.

| # | Severidad | Hallazgo | Resolución |
|---|---|---|---|
| 1 | media | El oráculo cuenta `duration in days` (y semanas) por instantes aunque los offsets difieran. Así respalda el defecto 1c28: da 2917557 en D370, donde la 1.5.3 da 2917556. Codex lo verificó con una copia que cuenta días de calendario, y el oráculo rechazó la respuesta correcta. | **Aceptado.** Con offsets distintos, el oráculo deja la duración en semanas y días a las referencias, como ya hacía con la diferencia. Salen 6 filas del oráculo: D368, D370, D386, D388, D404 y D406. Quedan 494. |
| 2 | media | El extremo de DateTime al offset de la evaluación está permitido en la 1.5.3 («may choose … a constant offset such as UTC»), así que no es un defecto de cql-execution. D364 y D366 no se explican por el offset: con cualquiera de los dos máximos dan 7987 y 95855. | **Aceptado.** Ya estaba corregido para 19 filas, que pasaron a deliberadas. D364 y D366 son el defecto de normalización de cql-execution en años y meses: con el inicio movido a -03:00 salen sus 7988 y 95856. Es la misma clase que `ages` D22 y `durations` D18. |

`go vet ./...` falla en código inalcanzable del parser generado, que este cambio no toca. Queda fuera del alcance.

### Ronda 2 (Codex, 05-11-2026)

Sin hallazgos. Codex confirmó las dos correcciones, la evidencia de D364 y D366, y que las 82 filas deliberadas están nombradas en este plan. Las 19 mutaciones se atrapan. **El review convergió en la ronda 2.**

Backlog nuevo: 1c37 (el nodo `DurationOf` del traductor), 1c38 (el plan 024, G3i) y 1c39 (citas de la 2.0.0 en textos de divergencias ya en el repo).

## Cierre (05-11-2026)

- **`make check`:** pasaron el lint (0 hallazgos) y `go test -race` en las seis zonas.
- **Mutaciones:** tres mutaciones de incrementos anteriores apuntaban al código que este cambio reescribió:
  - 013, «start of not stepped within the type»;
  - 016, «duration in X of an interval back in UTC»;
  - 020, la misma, para la diferencia.

  Se reanclaron a `startOf(iv)` y a `start, end`, sin cambiar lo que prueban. Con eso pasan las 600 mutaciones del 001 al 023.
