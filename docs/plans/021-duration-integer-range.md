# 021: las duraciones se miden en todo el calendario, del año 1 al 9999

Rama `fix/duration-integer-range`. Viene del backlog 1c31, abierto en la ronda adversarial del 020. **Por decisión del usuario del 03-11-2026, el plan se dividió:** este incremento corrige la medición (los defectos B y C, abajo). El null para un Integer fuera de rango (el defecto A, el 1c31 original) pasa al 022. La revisión previa y la ronda adversarial se refieren a los tres defectos.

## Revisión previa (03-11-2026)

### `gofhir/server` (congelado)

No evalúa CQL: usa `gofhir/cql`. No hay nada que tomar.

### Especificación (CQL 1.5.3)

- **Types:** un Integer es de 32 bits, de -2^31 a 2^31-1.
- **Duration y Difference** (Appendix B): su resultado es Integer. No dicen nada del desborde.
- La regla general la dan los operadores aritméticos: «operations that cause arithmetic overflow or underflow, or otherwise cannot be performed […] will result in null». Abs, Add, Successor, Floor y otros la repiten: «If the result of the operation cannot be represented, the result is null».
- El motor ya la aplica a la aritmética de Integer (#135) y en los bordes de los tipos (#169, plan 013). En #135 las duraciones quedaron fuera a propósito («Durations, widths and counts are left as they are», `eval/integer_range.go`), por alcance y sin una decisión de fondo.

### Sondas contra Java 5.4.0 y cql-execution 3.3.2 (`scratchpad/r021/in/probe.cql`, 34 casos)

**Defecto A: un resultado fuera del rango de Integer.** Pasa en milisegundos por encima de 24,8 días, en segundos por encima de 68 años y en minutos por encima de 4.085 años.

| Expresión | Java | cql-execution | Motor |
|---|---|---|---|
| `milliseconds between @2012-01-01T00:00:00.000 and @2012-01-25T20:31:23.647` (2^31-1) | 2147483647 | 2147483647 | 2147483647 |
| … y `…23.648` (2^31) | -2147483648 | 2147483648 | 2147483648 |
| `seconds between @1950-01-01T00:00:00.000 and @2020-01-01T00:00:00.000` | -2085978496 | 2208988800 | 2208988800 |
| `CalculateAgeInSecondsAt(@1950-01-01T00:00:00.000, @2020-01-01T00:00:00.000)` | -2085978496 | 2208988800 | 2208988800 |
| `difference in milliseconds between @2012-01-01T00:00:00.000 and @2012-02-01T00:00:00.000` | -1616567296 | 2678400000 | 2678400000 |
| `milliseconds between @2012-01T and @2012-02-25T00:00:00.000` | error | `[2073600001, 4752000000]` | `[2073600001, 4752000000]` |

- Java da la vuelta en 32 bits, con signo cambiado.
- cql-execution devuelve el número de JS, que no es un Integer de CQL.
- Ninguna referencia sigue la regla. El mismo cuadro ya pasó con la aritmética en #135: «Java no es oráculo en esta área».

**Defecto B: los lapsos de más de 292 años.** `time.Time.Sub` satura en `time.Duration`, que tiene unos 292 años, y el motor mide los días, las horas, los minutos, los segundos y los milisegundos con `Sub`. Las dos referencias coinciden entre sí y con la especificación.

| Expresión | Java | cql-execution | Motor |
|---|---|---|---|
| `days between @0001-01-01T00:00:00.000 and @9999-12-31T00:00:00.000` | 3652058 | 3652058 | **106751** |
| `days between @0001-01-01 and @9999-12-31` | 3652058 | 3652058 | **null** |
| `weeks between @0001-01-01 and @9999-12-31` | 521722 | 521722 | **null** |
| `hours between @1700-01-01T00:00:00.000 and @2020-01-01T00:00:00.000` | 2805048 | 2805048 | **2562047** |
| `CalculateAgeInDaysAt(@1700-01-01, @2020-01-01)` | 116877 | 116877 | **106751** |
| `days between @1700 and @2020-01-01` | `[116513, 116877]` | igual | **106751** |
| `difference in days between @0001-01-01 and @9999-12-31` | 3652058 | 3652058 | 3652058 (el 020 ya no usa `Sub`) |

106751 días son exactamente 2^63 ns.

## Alcance (tras la división)

1. **Defecto C: el año 1.** `toGoTime` y los `XBetween` de `funcs/temporal.go` y `funcs/datetime_advanced.go` usan `time.Time{}` (0001-01-01T00:00Z) como «sin valor». Se reemplaza por un `ok bool`, para que el primer instante del calendario se mida como cualquier otro.
2. **Defecto B: los lapsos de más de 292 años.** Días, semanas, horas, minutos, segundos y milisegundos se miden con aritmética entera de milisegundos (`UnixMilli`), no con `time.Duration`. Incluye cada camino de `duration`: el exacto, el conteo de períodos enteros del 015 y las incertidumbres del 016. También las edades, que pasan por ahí.
3. **El resultado fuera del rango de Integer no cambia:** sigue siendo el número, hasta el 022. En segundos y milisegundos, un lapso de más de 292 años da hoy el número grande que corresponde, y el 022 decidirá si es null.

## Fuera del alcance

- **El defecto A** (el null fuera del rango de Integer): es el 022, con su consulta a HL7.
- **El Date junto a un DateTime:** es el 019.
- **Time en `duration`:** es el 1c32.

## Divergencias previstas

Ninguna nueva. En las filas de B y C, las dos referencias coinciden con la especificación. Las filas de segundos y milisegundos de más de 2^31 siguen divergiendo como en el 020: Java da la vuelta en 32 bits, y el motor y cql-execution dan el número.

## Criterio de salida

- Las filas de B y C de las sondas dan lo que dan las dos referencias, por los dos caminos (AST y ELM).
- Una nueva área de snapshot, `durationrange`, con cientos de casos, contra Java y cql-execution:
  - el año 1 (Z, +03:00 a las 03:00, Date, incertidumbre) en lapsos cortos y largos;
  - lapsos de 290, 293 y 9998 años en cada unidad, en los dos signos, con Date y con DateTime;
  - `duration`, `difference`, `in X of` y las edades con un timestamp fijo.
- Cada divergencia del área queda clasificada con evidencia.
- La diferencial contra main es idéntica en las otras áreas, y eCQM da 23 coincidentes y 1 excusado.
- Los tests nuevos fallan en main, cada oráculo tiene mutaciones y `make check` está en verde.

## Tests y fuentes de datos

- **A mano, citando la especificación:** las filas de B y C de las sondas.
- **Área `durationrange`**, generada por un script en el repo e importada también por ELM.
- **Un oráculo independiente en Go** que cuenta con el número de día juliano, sin el paquete `time` ni `funcs`.

## Mutaciones previstas

- El centinela `IsZero` devuelto en un `XBetween`.
- Las horas otra vez con `Sub`, con casos de horas de más de 292 años en los dos signos.
- Los días de un Date otra vez con `Sub`.
- La incertidumbre armada con `NewDateTimeFromTime` desde un cero.
- El área fuera del manifiesto.
- Una respuesta del área que se aparta del oráculo.

## Decisión del usuario (03-11-2026)

El plan se divide: 021 son los defectos B y C, y el 022 es el defecto A (backlog 1c35). Se aprobó el plan reducido sin otra ronda adversarial.

## Ronda adversarial (Opus independiente, 03-11-2026)

Verifiqué los hallazgos alto y medios con `scratchpad/r021/in/probe2.cql`, contra Java y cql-execution.

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| 1 | **El null del defecto B tiene otra causa.** `toGoTime` usa el `time.Time` cero como «sin valor», pero el cero es 0001-01-01T00:00:00Z, un instante real. `days between @0001-01-01 and @0001-01-05` da null (las dos referencias dan 4). `hours between @0001-01-01T00:00:00.000Z and @0001-01-02T00:00:00.000Z` da null (las dos, 24). `CalculateAgeInDaysAt(@0001-01-01, @2020-01-01)` da null (las dos, 737424). Cambiar a `UnixMilli` solamente no lo arregla | alta | Confirmado. Es un **defecto C**: reemplazar el centinela por un `ok bool`, con tests en el año 1 (con Z, con +03:00 a las 03:00, como Date y como incertidumbre) y en lapsos cortos |
| 2 | **La base en la especificación es más débil de lo que dice el plan.** La regla del desborde está en la introducción de *Arithmetic Operators*. Duration y Difference están en *Date and Time Operators* y solo dicen «always an integer». El null es una lectura defendible (por analogía y por los 32 bits de Types), no una cita directa | media | Aceptado. El plan lo llama lectura, y se agrega a la consulta a HL7. Con el null, `(ms between @2012-01T and …) > 0` pasa de true a null; se agrega como test |
| 3 | **Son cuatro los lugares que devuelven un resultado**, no dos: `DifferenceBetweenAt` tiene su incertidumbre, su exacto y el fallback a Duration | media | Aceptado. El chequeo va una sola vez, alrededor de las dos funciones `*At` exportadas, y se suma una mutación por la rama incierta de Difference |
| 4 | Los llamadores y el alcance son seguros: expand, per, comparaciones y `DateAdd` no pasan por aquí | baja | Sin cambio |
| 5 | En los snapshots solo cambian las 50 filas de `differences`; B no cambia ninguna fila existente | baja | Sin cambio |
| 6 | **Detalles.** El borde de los minutos es de unos 4.083 años. B solo se ve en días, semanas, horas y minutos, porque segundos y milisegundos ya son null por A: la mutación «horas con `Sub`» necesita casos de horas de más de 292 años en los dos signos. El oráculo debe usar el número de día juliano, no el paquete `time`. Las edades contra `Now()` necesitan un timestamp fijo. El comentario de `integer_range.go` hay que actualizarlo | baja | Aceptados |
| 7 | **Un frente a la vez.** B y C corrigen la medición, y las dos referencias coinciden con la especificación. A es una lectura de la especificación en la que las dos referencias difieren de ella | baja | Propuesta de división: 021 = B y C; 022 = A |

## Implementación y validación (03-11-2026)

La implementación la hizo un subagente Sonnet.
- **`funcs/temporal.go`:** `elapsed(low, high)` da los segundos enteros y los nanosegundos sobrantes con `Unix()` y `Nanosecond()`, sin `time.Duration`. Los `XBetween` ya no usan el centinela `IsZero`.
- **`funcs/datetime_advanced.go`:** `toGoTime` y `temporalRange` devuelven un `ok`, `parseTemporal` reemplaza el centinela del fallback de texto, y `MillisecondsBetween` se calcula con segundos y nanosegundos.

**Área `durationrange`:** 1675 definiciones, generadas por `scripts/durationrange-area.py`. Tiene 353 divergencias, todas clasificadas por `scratchpad/r021/classify.py`:
- 312 son defecto propio conocido: Integer fuera de rango, de la lectura del 022 (1c35), con null como respuesta esperada (294 contra Java y 18 contra cql-execution);
- 31 de cql-execution: mide un Interval de DateTimes con offset en el año 1 o 9999 corrido por el offset de la evaluación, y Java coincide con el motor;
- 4 de Java: da null cuando los dos operandos son más gruesos que la precisión;
- 2 de Java: devuelve un Interval simple donde va una incertidumbre;
- 1 de Java: D1669, un número en lugar de una incertidumbre;
- 1 de Java: D1666, que da 3 en el primer instante del calendario, donde son 4 días enteros;
- 2 propias, D1672 contra cada referencia: la cota baja de una incertidumbre al año, en horas. Ya pasaba en main: `hours between @2019 and @2020-01-01T00:00:00.000` da `[27, 8763]` en main y en la rama. Va al backlog como 1c36, con la respuesta esperada calculada a mano.

**Tests:** `TestDurationsOverTheCalendarByHand` (AST y ELM) y `TestTheDurationRangeAreaIsItsOracle`, con un oráculo por día juliano que no usa `time` ni `funcs` y revisa 1667 filas. El primero falla en main.

**Mutaciones:** 7 en `scripts/mutations/021-durationrange.sh`, más la de una respuesta del área que se aparta del oráculo.

**Diferencial contra main:** idéntica en las 9 áreas. Las otras 24 áreas de snapshot no cambian; `differences` tampoco, porque sus 50 filas siguen esperando el 022.

**eCQM:** 23 coincidentes y 1 excusado.

**`make check`:** en verde, con 561 mutaciones (`scratchpad/r021/check2.log`). La primera corrida falló porque la mutación «Go's zero time as no value» del 015 ya no se aplicaba: su texto ahora aparece dos veces en `funcs/temporal.go`. Se reancló al contexto de `calendarBetween`.

## Code review

**Ronda 1 (Opus independiente, 03-11-2026): convergió.** Sin hallazgos de severidad media o mayor.
- El revisor verificó que los tests fallan en main.
- Probó a mano los bordes: truncar lapsos negativos, los años 1 y 9999 con offsets, medianoche de Time, edades, semanas y milisegundos de 0001 a 9999.
- No encontró `Sub`, `IsZero` ni `time.Duration` en ningún camino de duraciones, `difference`, edades o incertidumbres.

Hallazgos bajos:
1. **El préstamo de un lapso negativo no estaba cubierto** por los tests del 021. Corregido: hay cinco filas a mano verificadas contra Java y cql-execution (`scratchpad/r021/in/probe4.cql` y `probe5.cql`) y una mutación nueva. Ahora el script atrapa 9 de 9.
2. **La rama de texto de `toGoTime` es inalcanzable** desde CQL. Se deja como está, y la mutación «ok siempre true» sobrevive porque no hay camino que la alcance.
3. **El texto de las 31 filas de cql-execution:** el corrimiento es el offset de la evaluación, no el de la cota. Corregido.
4. **El texto de D1672:** la cota alta también está corrida, 3 horas. Corregido.
5. Las demás clasificaciones revisadas se sostienen.
6. **El script de mutaciones** tenía un `#!/bin/sh` duplicado y un comentario viejo. Corregido.
7. **El plan del 018** estaba en el conjunto de la copia de revisión, pero no se sube.

## Cierre (04-11-2026)

`make check` final en verde, con 562 mutaciones atrapadas y lint sin hallazgos (`scratchpad/r021/check3.log`). Queda pendiente el 022 (backlog 1c35), con su punto para la consulta a HL7.

## Dependencias

Ninguna nueva.
