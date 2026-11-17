# 026: `expand … per` trunca los límites a la precisión de `per`

Backlog 1c51 (M2 del 024). Rama `fix/expand-truncates-to-per`. Sigue al 025 (#182, 186d4f7).

## Revisión previa (16-11-2026)

Sondas en `scratchpad/r026` (26 expresiones) y `scratchpad/r026b` (28), contra Java 5.4.0 y cql-execution 3.3.2, a -03:00 y +05:30. En estas sondas, ningún resultado de `expand` ni de `collapse` cambia con el offset de la evaluación. No se afirma en general: un DateTime sin offset sí depende de él (ronda adversarial 1, hallazgo 6).

**Especificación.** CQL 1.5.3, *Expand* (https://cql.hl7.org/N1A/09-b-cqlreference.html#expand):

> If the interval boundaries are more precise than the per quantity, the more precise values will be truncated to the precision specified by the per quantity. In these cases, the resulting list of intervals may be more broad than the input range due to this truncation. For example:
> `expand { Interval[@T10:00, @T12:30] } per hour` → `{ Interval[@T10, @T10], Interval[@T11, @T11], Interval[@T12, @T12] }`
> `expand { Interval[10.0, 12.5] } per 1` → `{ Interval[10, 10], Interval[11, 11], Interval[12, 12] }`

Y para el tamaño: `expand { Interval[@2018-01-01, @2018-01-04] } per 2 days` → `{ Interval[@2018-01-01, @2018-01-02], Interval[@2018-01-03, @2018-01-04] }`, y `expand { Interval[10, 10] } per 0.1` → `{ Interval[10.0, 10.0], Interval[10.1, 10.1], ..., Interval[10.9, 10.9] }`.

**El motor.** `funcs/interval_set.go`: `truncateTemporalToPrecision` trunca solo Time («only truncate Time values for now»). Un Date o un DateTime con un límite más preciso que `per` empieza a contar desde el límite tal cual. Un Decimal o una Quantity, también. Los ejemplos de la especificación pasan porque sus límites bajos caen en la unidad.

| Caso | Motor | Java | cql-execution |
|---|---|---|---|
| `expand { Interval[@…T00:30Z, @…T02:10Z] } per hour` | 00:30, 01:30 | 01, 02 | 00, 01, 02 |
| `expand { Interval[@…01T10:00Z, @…03T09:00Z] } per day` | 01T10:00, 02T10:00 | 02, 03 | 01, 02, 03 |
| `expand { Interval[@2020-01-15, @2020-03-10] } per month` | 01-15, 02-15 | 02, 03 | 01, 02, 03 |
| `expand { Interval[@2020-06-15, @2021-02-01] } per year` | 2020-06-15 | 2021 | 2020, 2021 |
| `expand { Interval[1.5, 4.2] } per 1` | 1.5, 2.5, 3.5 | 2, 3, 4 | 1, 2, 3, 4 |
| `expand { Interval[-1.5, 1.2] } per 1` | -1.5, -0.5, 0.5 | -1, 0, 1 | -1, 0, 1 |
| `expand { Interval[1.5 'mg', 4.2 'mg'] } per 1 'mg'` | 1.5, 2.5, 3.5 mg | 2, 3, 4 mg | 1, 2, 3, 4 mg |
| `expand { Interval[1.5, 4.2] } per 2` | [1.5,1.5], [3.5,3.5] | [2,3] | [1,2], [3,4] |
| `expand { Interval[@T10:30, @T12:30] } per hour` (Time) | 10, 11, 12 | 11, 12 | 10, 11, 12 |

- **cql-execution trunca y empieza en la unidad truncada** en todas esas filas. No en todo: con `per 0.1` da trozos de coma flotante (r026b D4), y en las colas incompletas difiere (r026c).
- **Java no la sigue:** redondea el límite bajo hacia arriba (Time incluido: el motor ya coincide con cql-execution en Time). Con `-1.5` sí trunca hacia cero. Defecto de la referencia.
- **El motor tiene además un defecto propio en Decimal con un `per` entero mayor que 1:** `expandDecimalIntervals` toma un paso entero como intervalo unitario (`end = v`), y `per 2` da `[1.5, 1.5]` donde el tamaño es 2. Para la especificación, `per 2 days` da `[01, 02]`. La fórmula general que ya usa para un paso fraccionario (`end = v + per − una unidad de la precisión de per`) da el resultado correcto también para el entero.

**`gofhir/server` (congelado):** usa el mismo `funcs/interval_set.go` de cql, sin lógica propia de `expand`. Nada que tomar.

**Fuera de lo medido, pero visto:**
- `collapse … per hour` con límites fuera de la unidad (D17 de r026): el motor deja dos intervalos, y las dos referencias juntan uno. El pseudocódigo de la especificación (informativo) no los junta. La especificación y las referencias no coinciden.
- `expand { Interval[150 'cm', 420 'cm'] } per 1 'm'`: el motor da puntos de 100 cm, las dos referencias `[150 cm, 249 cm], [250 cm, 349 cm]`. Es el frente de unidades distintas que ya está en el backlog (`[100 'cm', 2 'm'] per 50 'cm'`).
- `per 0.5` sobre `[1.25, 2.0]`: motor `[1.25, 1.65], [1.75, 2]`, Java `[1.3, 1.7]`, cql-execution `[1.25, 1.74999999]`. Ninguna coincide con otra.

## Plan

Ajustado tras la ronda adversarial 1 (abajo). Una sola regla para `expand` de una lista, y la sobrecarga de un intervalo toma los inicios de lo que da la lista:

1. **`per`.** Si se omite o es null, es una unidad de la precisión más gruesa entre los límites de **toda** la entrada (*Expand*: «a per value will be constructed based on the coarsest precision of the boundaries of the intervals in the input set»). `week` es 7 días, no una precisión.
2. **La entrada.** Se quitan los null, se ordena y se juntan los intervalos que se superponen o se tocan. Las dos referencias lo hacen: `{[1,4], [2,5]} per 2` da `[1,2], [3,4]`, y `{[2,9], [1,1]} per 3` da `[1,3], [4,6], [7,9]` (r026c D3, D35). El resultado sale ordenado.
3. **Los límites.** Primero `start of` y `end of`, a la precisión propia del límite (*Start*: «If the low boundary of the interval is open, this operator returns the successor of the low value»; *End*, el predecesor). Después se truncan los dos a la precisión de `per`, y quedan cerrados. `(@T10:59, @T12:00] per hour` da 11 y 12, y `[@T10:30, @T12:00) per hour` da 10 y 11, como cql-execution (r026c D18, D19). El orden no lo fija *Expand*: es una lectura, y la sostienen los dos casos de cql-execution y el de Java en D18.
   - Date, DateTime y Time se truncan en sus propios dígitos. Un DateTime conserva su offset, escrito o ausente.
   - Decimal y Quantity de la misma unidad se truncan a la escala de `per`. El sentido para un negativo es la **decisión D1**.
4. **Un límite menos preciso que `per`:** para Date, DateTime y Time, el intervalo no aporta nada (*Expand*: «the interval will not contribute any results»). Hoy un Date se lee siempre como día, y `[@2020-01, @2020-03] per day` corre hasta el límite de 100 000; las dos referencias dan `{}` (D13). Para los números, `per` agrega precisión, como hoy.
5. **Los trozos.** Un trozo de tamaño `per` empieza en el inicio truncado y termina una unidad de la precisión de `per` antes del siguiente. Solo se aportan los trozos completos: «end on or before the upper boundary». `[1.0, 3.0] per 2` da `[1, 2]`, `[@…01, @…03] per 2 days` da `[01, 02]`, y `[@…01, @…07] per week` da `[01, 07]`. Las dos referencias lo hacen (D7, D9, D33). Hoy el motor deja un trozo incompleto o de un punto.
   - Con un `per` de varias unidades, dónde empiezan los trozos de un segundo intervalo disjunto es la **decisión D2**.
6. **El calendario.** Se cuenta desde el inicio truncado, sumando `k·per` desde ese inicio y no encadenando sumas: así un 31 de enero que trunca a enero no se vuelve 29. El recorrido termina cuando un trozo pasa del límite alto, cuando sumar sale del calendario (año 9999) o cuando el valor no avanza (un Time que da la vuelta a las 24). `[@T23, @T23] per hour` da `[23, 23]`, como cql-execution; hoy da error por el límite (D25). `[@9999-12-31, @9999-12-31] per day` da su único trozo; las dos referencias dan error (D23), y el motor da `{}` en la lista y un punto en la sobrecarga de intervalo.

### Fuera de alcance

- `collapse … per` con límites fuera de la unidad: la especificación (pseudocódigo informativo) y las dos referencias no coinciden. Al backlog, con D17 de r026. Las funciones compartidas con `collapse` (`expandGetStep`, `defaultTemporalUnit`, `DateAdd`) no cambian de comportamiento para `collapse`; lo cubren tests de regresión.
- Un intervalo cuyos límites tienen offsets distintos, o uno con offset y otro sin él: comparar el final de un trozo a la hora con el límite alto es el tramo del 025, y puede quedar indeciso. Va al backlog, con un test que documenta lo que hace hoy (r026c D29; las referencias tampoco coinciden).
- Quantity en unidades distintas (backlog existente) y un `per` fraccionario que no es una unidad de su escala (`per 0.5`): las tres implementaciones difieren.
- `CompareTemporal` con una hora y offsets distintos (A2 del 024): el motor y las dos referencias dan null en todos los offsets. Sin divergencia; se cierra en el backlog con esa nota.

### Decisiones para el usuario

- **D1. Truncar un Decimal negativo:** (a) hacia cero, `-2.9` → `-2`: como *Truncate* («the integer component of the argument») y como un DateTime trunca sus dígitos; `[-2.9, -1.1] per 1` da `-2, -1`, como cql-execution (D27). (b) hacia abajo (`floor`), `-2.9` → `-3`: una partición de la recta en unidades de `per`, y ninguna referencia lo hace. **Recomendado (a).**
- **D2. Un `per` de varias unidades sobre intervalos disjuntos:** `{[1,4], [6,9]} per 2`.
  - (a) Cada intervalo, ya juntado, cuenta desde su propio inicio truncado: `[1,2], [3,4], [6,7], [8,9]`, como Java (D1, D4). Es lo que hace hoy el motor.
  - (b) Todos desde el inicio del primero, como dice *Expand*: «the 'per' partitions start from the starting boundary of the first input interval, ordered». Así, `[6,9]` aporta solo `[7,8]`. Ninguna referencia lo hace: cql-execution ancla en el primero, pero aporta `[5,6]`, que empieza antes del intervalo.
  - **Recomendado (b),** por el texto de la especificación. Con `per` de una unidad, que es el caso común, las dos dan lo mismo.

## Tests y datos (paso 5)

- Tests a mano con los ejemplos de la especificación (citados) y los casos de las sondas, en las dos sobrecargas, a -03:00 y +05:30, con valores de la especificación o de cql-execution donde sigue la regla.
- **Área nueva `expansions`** (`scripts/expansions-area.py`): cientos de casos de Date, DateTime (con offset, sin offset, offset fraccionario), Time, Decimal, Integer y Quantity, con límites dentro y fuera de la unidad, abiertos y cerrados, `per` de 1 y de varias unidades, y las dos sobrecargas. **Oráculo independiente** en Python, por la regla de la especificación: truncar, sumar `per`, cortar por el límite alto. Respuestas de Java y cql-execution con el arnés (`scripts/references.sh`).
- Cada divergencia con una referencia se clasifica: Java por el redondeo hacia arriba (defecto de la referencia, con la cita), cql-execution donde no tiene la sobrecarga de puntos (devuelve intervalos), y lo que sea propio.
- Casos por regla, con testigos que separan las lecturas:
  - cola incompleta en Integer, Decimal, Date, Time y `per 2 months`;
  - `per` omitido y null, con precisiones mezcladas en la lista;
  - entradas desordenadas, superpuestas y disjuntas;
  - `(@T10:59, …]`, cuyo sucesor cruza la unidad, frente a `(@T10:30, …]`;
  - fines de mes, 29 de febrero, año 9999, `@T23` y el límite de expand apagado;
  - Decimal negativo, entero y abierto;
  - regresión de Integer, Long y Quantity con `per` fraccionario, y `collapse` sin cambios.
- **Mutaciones** (`scripts/mutations/026-expand-truncates.sh`):
  - sin truncar DateTime, Date o Decimal;
  - truncar hacia arriba, o en el sentido que D1 descarta;
  - dejar el trozo incompleto;
  - encadenar sumas de calendario;
  - no juntar la entrada, o no ordenarla;
  - inferir `per` por intervalo en vez de por la lista;
  - `week` como precisión;
  - quitar la guarda de no avanzar;
  - truncar antes de `start of`;
  - truncar en el offset de la evaluación en vez de los dígitos propios;
  - mutar el oráculo del área.
- Diferencial contra main en los nueve corpus y volcado eCQM (se espera idéntico: ningún eCQM usa `expand`; se verifica).

## Criterio de salida

- Los ejemplos de *Expand* de la especificación pasan en las dos sobrecargas.
- El área `expansions` coincide con su oráculo en toda fila que el oráculo responda; toda divergencia con Java o cql-execution queda clasificada con evidencia.
- Mutaciones del 026 detectadas todas, y las anteriores (001-025) sin sobrevivientes.
- `make check` en verde.

Sin dependencias nuevas. Sin excepciones a la regla 5.

## Ronda adversarial 1 (Codex, 16-11-2026)

Codex dejó 10 hallazgos, con sondas en el motor (`/private/tmp/claude-501/r026/.probe/attack026-output.txt`). Los verifiqué en main y medí los casos nuevos contra Java y cql-execution (`scratchpad/r026c`, 35 casos a -03:00).

| # | Sev. | Hallazgo | Resolución |
|---|---|---|---|
| 1 | alta | Trozos incompletos: `[1.0, 3.0] per 2` da `[1,1], [3,3]`, y `[@T10:30, @T12:30] per 2 hours` da `[10,11], [12,12]`. | **Aceptado** (regla 5). Las dos referencias dan solo los trozos completos (D7, D9). |
| 2 | alta | `per` omitido o null se infiere por intervalo; `[@2020-01, @2020-03]` sin `per` corre hasta el límite. | **Aceptado** (regla 1). Java infiere por toda la lista (D16), y las dos referencias dan meses en D14. |
| 3 | alta | Las particiones no se alinean con el primer intervalo; una entrada al revés sale al revés. | **Aceptado en parte** (regla 2 y D2). Las dos referencias juntan y ordenan la entrada (D3, D35). El ancla común con intervalos disjuntos es la decisión D2: las referencias no coinciden. |
| 4 | alta | Calendario: `[@2020-01-31, @2020-05-31] per month` da primero `[01-31, 01-29]`; `@T23 per hour` corre hasta el límite; el año 9999 da distinto en las dos sobrecargas. | **Aceptado** (regla 6). |
| 5 | alta | Un Date se lee siempre como día: `[@2020-01, @2020-03] per day` corre hasta el límite, y las referencias dan `{}`. | **Aceptado** (regla 4). |
| 6 | alta | «No depende del offset» no está probado: con un límite sin offset y otro con offset, el resultado cambia con la evaluación. | **Aceptado.** Se quita la afirmación general. Los límites con offsets distintos van fuera de alcance, con un test que documenta lo que pasa hoy: comparar el final de un trozo con el límite alto es el tramo del 025. |
| 7 | media | Falta un testigo de un límite abierto cuyo sucesor cruza la unidad: `(@T10:59, @T12:00]`. | **Aceptado** (regla 3). Las dos referencias dan 11 y 12 (D18), así que `start of` va antes de truncar. Queda registrado como lectura. |
| 8 | media | El plan pedía `floor` y D1 recomendaba truncar hacia cero. | **Aceptado.** La regla remite a D1. |
| 9 | media | `expandDecimalIntervals` y las funciones de `collapse` son compartidas; Long `per 0.1` da diez trozos. | **Aceptado:** tests de regresión de Integer, Long, Quantity y `collapse`. |
| 10 | media | Es exagerado decir que cql-execution sigue la especificación en todos los casos (D4 de r026b, colas incompletas). | **Aceptado:** las filas se clasifican una por una. |

La sonda r026c encontró además que `per week` se lee como precisión de día y da puntos. Las dos referencias dan trozos de 7 días (D31-D33). Entra en la regla 1.

## Aprobación del usuario (16-11-2026)

Plan aprobado, con **D1 (a)**: un Decimal negativo se trunca hacia cero (`-2.9` → `-2`). Y **D2 (b)**: con un `per` de varias unidades, los trozos se cuentan desde el inicio truncado del primer intervalo (ya juntado y ordenado), y cada intervalo aporta los trozos completos dentro de sus límites: `{[1,4], [6,9]} per 2` da `[1,2], [3,4], [7,8]`.

## Aclaración del orden (paso 7, 16-11-2026)

Las 52 divergencias que los modelos de las referencias no explicaban mostraron un orden que el plan no fijaba. Las dos referencias juntan los intervalos de la lista **después** de truncarlos, cuando se tocan o se superponen a la precisión de `per`. Ejemplo: `{[@…18T00Z, @…20T00Z], [@…15T00Z, @…17T00Z]} per 2 days` da `[15,16], [17,18], [19,20]`, porque a la hora los dos no se tocan y al día sí (expansions D652, y D638). Truncar primero es lo coherente con la regla 3. Se corrigieron el oráculo y el motor, y se agregaron los dos casos a `TestTheInputIsCollapsedOrderedAndAnchoredAtTheFirst`. Esto no cambia ninguna decisión del usuario.

## Validación (paso 7, 16-11-2026)

**Área `expansions`** (`scripts/expansions-area.py`): 836 definiciones, 832 con respuesta del oráculo independiente. El motor coincide con el oráculo en las 832 (`TestTheExpansionsAreTheirOracle`).
- Salen del área las filas de Time cuyo recorrido cruza la medianoche: Java 5.4.0 da la vuelta a la mañana, no pasa nunca el límite alto y se queda sin memoria (D535 y D678 de un borrador). Esos casos están en los tests a mano.
- **Divergencias** (`expansions.divergences.tsv`, clasificadas con `--verdicts`, ninguna sin explicar): Java, 66 deliberadas y 498 defectos de la referencia; cql-execution, 118 deliberadas y 219 defectos de la referencia.
  - **Java, defecto de la referencia:** redondea el límite bajo hacia arriba y el alto hacia abajo, y cuenta cada intervalo desde su inicio. También le falta precisión a un Integer con `per 0.1` (lo contradice el ejemplo de la especificación), falla en el año 9999, y en D770 junta antes de truncar.
  - **cql-execution, defecto de la referencia:** devuelve intervalos en la sobrecarga de puntos; calcula los Decimal en coma flotante, con trozos que terminan 10^-8 antes del siguiente; junta los intervalos a menos de un `per` y cuenta cada uno desde su inicio; da null en una lista vacía o con un límite nulo abierto; y en D697 no trunca.
  - **Deliberadas, los mismos valores escritos de otra forma** (`+00:00` por `Z`, el offset de la evaluación por ninguno, o ningún offset): D75 D76 D77 D78 D99 D100 D101 D102 D103 D104 D105 D106 D107 D108 D109 D128 D129 D131 D134 D136 D138 D139 D140 D141 D142 D143 D164 D165 D166 D167 D169 D173 D175 D176 D177 D178 D179 D180 D181 D182 D183 D184 D185 D186 D224 D225 D226 D227 D228 D229 D254 D255 D256 D257 D258 D259 D290 D291 D292 D293 D294 D392 D394 D414 D415 D416 D418 D421 D423 D443 D445 D446 D447 D448 D449 D450 D452 D454 D456 D457 D458 D459 D479 D480 D481 D482 D483 D485 D486 D487 D489 D490 D494 D495 D496 D497 D562 D563 D564 D565 D566 D567 D571 D572 D573 D596 D597 D599 D600 D601 D602 D603 D604 D605 D607 D608 D609 D610 D611 D612 D614 D618 D621 D625 D626 D627 D628 D632 D634 D635 D654 D655 D657 D658 D659 D660 D661 D662 D663 D664 D665 D668 D669 D670 D671 D796 D801.
  - **Deliberadas por la regla 3 y D1** (un límite numérico abierto es su sucesor o predecesor, truncado hacia cero; las dos referencias descartan la unidad en la que cae): D698 D722 D734 D819 D820.
- `units` D26 (`expand Interval[100 'cm', 2 'm'] per 50 'cm'`) pasa de `{100, 150, 200}` a `{100, 150}` por la regla 5: el trozo `[200 cm, 249 cm]` termina después de 2 m. Las dos divergencias se reclasificaron como defectos de las referencias. Se ajustaron los esperados de `TestWidthAndExpandAnswerInTheMostGranularUnit` y, por la regla 2, `TestExpandPastItsLimitFails`, que ahora usa intervalos disjuntos.

## Mutaciones (paso 7, 16-11-2026)

`sh scripts/mutations/026-expand-truncates.sh` aplica 24 mutaciones, cada una en una copia del repo, y en las 24 el test que nombra falla con su mensaje. Cubren:
- truncar: no truncar, redondear, cortar el DateTime en otro dígito, perder el offset, no darle precisión al Integer;
- trozos enteros y ancla: el último trozo pasa el límite alto, cada intervalo se cuenta desde su inicio (Decimal, Integer y temporal);
- juntar después de truncar: no juntar, o no juntar los que solo se tocan (temporal, Integer y Decimal);
- un límite menos preciso que `per` aporta algo, `per` se infiere del límite más fino, una semana dura un día;
- un límite abierto se lee cerrado (temporal y Decimal);
- el oráculo y sus archivos: otra respuesta en el oráculo o en el snapshot, una fila del oráculo que no está en el área, una fila deliberada que el plan no nombra.

Al principio cuatro mutaciones no compilaban o no se aplicaban; se reescribieron para que compilen y se apliquen, sin cambiar el defecto que inyectan.

## Code review (paso 8)

**Ronda 1 (Codex, 16-11-2026).** Tres hallazgos, los tres reproducidos y aceptados:
1. **Alta.** Un Time con un `per` enorme hace que `amount*ms` desborde a 0 y el motor entra en pánico por división por cero.
2. **Media.** `per … weeks` enorme: `amount*7` da la vuelta y deja un paso chico.
3. **Media.** Se descartaban los intervalos con un límite nulo **antes** de inferir `per`, y eso contradice la regla 1. Un límite definido cuenta para la inferencia aunque su intervalo no aporte nada.

Corrección:
- Multiplicaciones con control de desborde. Un paso que no se puede representar, o que es más largo que el día en un Time, da la lista vacía.
- `temporalStep` (solo lo usa `expand`) rechaza un `per` que no cabe en int64.
- `per` se infiere sobre toda la lista antes de descartar los intervalos que no aportan.
- Tests nuevos: `TestPerIsInferredFromEveryDefinedBoundary` y `TestAPerTooLargeForAnyPartitionGivesNone`.

Después de corregir:
- `go test ./...` en verde y lint sin problemas;
- 24 de 24 mutaciones atrapadas;
- diferencial y eCQM idénticos a main.

**Ronda 2 (Codex, 16-11-2026).** Tres hallazgos:
1. **Media, aceptado.** Un `per` numérico que no cabe en el paso (`per 18446744073709551617.0` sobre Integer) se envolvía con `IntPart()` y quedaba en un paso chico. Ahora da la lista vacía. Test: `TestANumericPerTooLargeGivesNone`.
2. **Media, rechazado para este incremento, al backlog.** Con `per` omitido, un Decimal avanza de a 1 y no de a una unidad de la escala más gruesa de sus límites. Así lo hacía main, y la diferencial no cambia. Las dos referencias tampoco infieren la escala: `expand { Interval[1.5, 3.2] }` da en Java `[1.5, 2.4]` y en cql-execution `[1,1] [2,2] [3,3]` (sonda r026c D17). Decidir qué es la «precisión» de un Decimal sin `per` (escala escrita o 1) es otra decisión de semántica, con su propia diferencial. La regla 1 de este plan queda para los temporales, y para los números cuando `per` es explícito.
3. **Alta, aceptado.** Se habían borrado del paquete `funcs` las funciones exportadas `IntervalExpandPoints`, `IntervalExpandIntervals` e `IntervalExpandIntervalsUpTo`, y eso rompe la API del módulo. Vuelven como envoltorios de la implementación nueva. Test: `TestTheExportedExpansionsRemain`.

**Ronda 3 (Codex, 16-11-2026). No convergió**, y quedan pendientes de la decisión del usuario. Los dos hallazgos se reproducen:
1. **Media.** `expand { Interval[-9223372036854775807L - 1L, 9223372036854775807L] } per 9223372036854775808.0` da `{}`. Lo esperado son los dos trozos de 2^63 puntos, `[min, -1]` y `[0, max]`. La corrección de la ronda 2 (`IsInt64()`) rechaza un paso que no cabe en int64 aunque los trozos sí quepan en el dominio de Long. Es un caso extremo: antes de la ronda 2 el paso se envolvía.
2. **Baja.** En `--verdicts`, la regla del límite abierto (D820 de cql-execution, `expand Interval(1.0, 3.0) per 1`) marca la fila como deliberada. Pero la fila también tiene el defecto de la sobrecarga de puntos de cql-execution, que devuelve intervalos, y la evidencia no lo nombra.

Las sondas de `collapse … per` dan lo mismo que en main. La API restaurada y los casos de `per` enorme pasan.

**Cierre por decisión del usuario (16-11-2026),** opción (a): se corrigen los dos hallazgos de la ronda 3 y no se abre una cuarta ronda, como en el 025.
1. Un paso de Integer o Long se acepta si cabe en uint64: es la cuenta que ya se hacía en uint64. `[min Long, max Long] per 2^63` da `[min, -1], [0, max]`. Un paso mayor no aporta trozos. Test: `TestALongPerCanFillTheDomain`.
2. En `--verdicts`, la evidencia de un límite abierto nombra también el defecto de la sobrecarga de puntos de cql-execution cuando la respuesta trae intervalos (D820).

Después de corregir:
- 24 de 24 mutaciones atrapadas;
- diferencial y eCQM idénticos a main;
- las mutaciones de los incrementos 001 a 025 corrieron todas con rc=0 (82 minutos, sobre el estado de la ronda 2).

## Backlog que deja

- Con `per` omitido, un Decimal avanza de a 1 (como en main). Ninguna referencia infiere la escala (r026c D17). Decidir qué significa la precisión de un Decimal es su propio incremento.
- Fuera de alcance desde el plan: `collapse … per`; límites con offsets distintos; Quantity en unidades distintas; `per 0.5`; A2 CompareTemporal.
