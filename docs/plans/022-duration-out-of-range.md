# 022: una duración que no cabe en un Integer es null

Rama `fix/duration-out-of-range`. Viene del backlog 1c35, que es el defecto A del 1c31. El 021 separó este defecto por decisión del usuario del 03-11-2026; su revisión previa ([021-duration-integer-range.md](021-duration-integer-range.md)) tiene las sondas.

## Revisión previa (04-11-2026)

### `gofhir/server` (congelado)

No evalúa CQL: usa `gofhir/cql`. No hay nada que tomar.

### Especificación (CQL 1.5.3)

- **Duration y Difference** (Appendix B), las dos con las firmas `(Date, Date)`, `(DateTime, DateTime)` y `(Time, Time)`. Ninguna tiene variante Long.
  - Devuelven `Integer`: «The result of this operation is always an integer; any fractional periods are dropped».
  - Su texto no dice nada del desborde. La única regla de null que traen es «If either argument is null, the result is null».
- **Types:** un Integer es de 32 bits, de -2^31 a 2^31-1.
- **Arithmetic Operators**, en la introducción: «operations that cause arithmetic overflow or underflow, or otherwise cannot be performed (such as division by 0) will result in null, rather than a run-time error». Abs, Add, Successor y otros la repiten: «If the result of the operation cannot be represented, the result is null».
- **CalculateAge…** es Duration desde la fecha de nacimiento (Appendix B, Clinical Operators) y devuelve Integer.

**Lectura:** el resultado de Duration y Difference es un Integer. Cuando el valor no se puede representar como uno, la operación «cannot be performed» y da null. Es la regla general del motor desde #135, donde las duraciones quedaron fuera por alcance (`eval/integer_range.go`: «Durations, widths and counts are left as they are»), no por una decisión de fondo.

**Es una lectura, no una cita textual.** Duration está en *Date and Time Operators*, no en *Arithmetic Operators*. La alternativa, un número que no es un Integer de CQL, contradice la firma. Va a la consulta a HL7 junto con los otros puntos ya redactados.

### cql-tests (HL7)

`conformance/testdata/CqlDateTimeOperatorsTest.xml` no tiene ningún caso de duración ni de diferencia fuera del rango de Integer. Los casos «Overflow» de ese archivo son de acarreo entre componentes (`+ 25 hours`), no de rango.

### Referencias (sondas del 021: `scratchpad/r021/in/probe.cql`)

| Expresión | Java 5.4.0 | cql-execution 3.3.2 | Motor |
|---|---|---|---|
| `milliseconds between @2012-01-01T00:00:00.000 and @2012-01-25T20:31:23.647` (2^31-1) | 2147483647 | 2147483647 | 2147483647 |
| … y `…23.648` (2^31) | -2147483648 | 2147483648 | 2147483648 |
| `seconds between @1950-01-01T00:00:00.000 and @2020-01-01T00:00:00.000` | -2085978496 | 2208988800 | 2208988800 |
| `difference in milliseconds between @2012-01-01T00:00:00.000 and @2012-02-01T00:00:00.000` | -1616567296 | 2678400000 | 2678400000 |
| `milliseconds between @2012-01T and @2012-02-25T00:00:00.000` | error | `[2073600001, 4752000000]` | `[2073600001, 4752000000]` |

- Java da la vuelta en 32 bits.
- cql-execution devuelve el número de JS, que no es un Integer de CQL.
- Ninguna de las dos sigue la regla, el mismo cuadro que en #135.

### Dónde se calcula

- El resultado sale de dos funciones, `funcs.DurationBetweenAt` y `funcs.DifferenceBetweenAt` (`funcs/datetime_advanced.go`).
  - Los caminos AST y ELM llegan ahí por `evalDurationBetween` y `evalDifferenceBetween`.
  - Las edades llegan por `funcs.CalculateAgeIn`, que llama a `DurationBetweenAt`.
  - `DurationBetween` y `DifferenceBetween` las envuelven con el marco UTC.
  - No hay otros llamadores fuera de `funcs` (grep).
- **Qué unidades se pasan del rango.** En 9998 años:
  - nunca se pasan los años, los meses, las semanas, los días (3,65 millones) ni las horas (87,6 millones);
  - los minutos se pasan por encima de 4085 años, los segundos por encima de 68 años y los milisegundos por encima de 24,8 días;
  - un Time llega a lo sumo a 86.400.000 ms, así que nunca se pasa.
- **A diferencia de #135, no hace falta el plan de tipos.** Duration y Difference devuelven Integer siempre, cualquiera sean sus operandos, así que el borde de 32 bits se aplica igual por AST y por ELM importado.

### Lo afectado hoy (medido sobre `testdata/snapshots/*.answers.tsv`)

| Área | Integer fuera de rango | Incertidumbre con una cota fuera |
|---|---|---|
| `durationrange` | 294 | 0 |
| `differences` | 3 | 47 |

En el resto de las áreas, ninguna fila de duración, diferencia o edad sale del rango. Las 128 filas de `integer` y la de `limits` son aritmética de Long y no cambian.

Hoy hay 362 divergencias clasificadas como «defecto propio (1c35)», todas con null como respuesta esperada: 312 en `durationrange` y 50 en `differences`.

## Alcance

1. **Un Integer fuera del rango es null.** Si Duration o Difference dan un número fuera de [-2^31, 2^31-1], el resultado es null. -2^31 sí es representable. Se aplica a las edades, porque pasan por Duration.
2. **Una incertidumbre con una cota fuera del rango es null entera** (decisión D1, abajo).
3. **La duración es un Integer «certero» para la aritmética de #135** (hallazgo 2). `certainInteger` (`eval/integer_range.go`) reconoce `DurationBetween`, `DifferenceBetween`, `DurationOf`, `DifferenceOf` y las funciones de edad (`AgeIn…`, `AgeIn…At`, `CalculateAgeIn…`, `CalculateAgeIn…At`), cuyo resultado es siempre Integer. Así `Abs(d)`, `-d` y `d + 1` dan null fuera del rango por el camino con plan. Por ELM importado sin plan sigue valiendo solo el borde de 64 bits, la limitación conocida de #135. El comentario «Durations, widths and counts are left as they are» se corrige.
4. **Los `XBetween` exportados** (`funcs/temporal.go`, `MillisecondsBetween`) documentan que dan el número sin la garantía de Integer, porque son los ladrillos de `DurationBetweenAt`.

### Decisión D1: la incertidumbre que cruza el borde

`difference in milliseconds between @2012-01T and @2012-01-01T` da hoy `[-2678399999, 86399999]`: la cota baja no es un Integer.

- **(a) null para toda la incertidumbre (recomendada).** La incertidumbre es el conjunto de valores posibles del resultado, y uno de ellos no se puede representar, así que la operación no se puede hacer como Integer. Esas 47 filas se clasificaron con null como esperado en el 020 y el 021, pero eso no prueba D1 (hallazgo 5).
- **(b) recortar la cota al borde.** Inventa un valor que la especificación no da.
- **(c) una cota null**, como `[null, 86399999]`. Un Interval con una cota null significa «sin límite», y eso no es lo que pasa.

## Fuera del alcance

- **El Date junto a un DateTime:** es el 019.
- **Time en `duration`:** es el 1c32.
- **La cota de una incertidumbre al año en horas:** es el 1c36.
- **`width of`, `Count` y las demás operaciones que #135 dejó fuera:** ninguna es Duration ni Difference.
- **Las funciones exportadas `XBetween` de `funcs`** siguen devolviendo el número. Son los ladrillos que usa `DurationBetweenAt`, y un null ahí cambiaría las cotas de las incertidumbres por un camino que esta decisión no busca. Ningún paquete fuera de `funcs` las llama.

## Divergencias previstas

- **Las filas en que Java ya da null** (22 en `differences`, como D127) pasan a coincidir: se borran del archivo de divergencias, y las respuestas fijadas se actualizan (hallazgo 1).
- Las demás de las 362 dejan de ser «defecto propio» y se clasifican fila por fila, porque algunas respuestas de Java son errores y no números envueltos. Contra Java pasan a «reference defect: Java wraps the Integer in 32 bits», y contra cql-execution a «reference defect: a JavaScript number past the Integer range». Las dos clases citan la regla y este plan, como hizo #135.
- Las filas nuevas del área se clasifican igual.
- No hay divergencias deliberadas nuevas.

## Criterio de salida

- Tests a mano, por AST y por ELM:
  - los bordes ±2^31 en milisegundos, segundos y minutos, en los dos signos, con 2^31-1 y -2^31 como números y 2^31 y -2^31-1 como null;
  - Duration, Difference, `duration in X of`, `difference in X of`, y las edades en segundos y minutos (`CalculateAgeInSecondsAt`, `CalculateAgeInMinutesAt`, y `AgeIn…` y `CalculateAgeIn…` sin fecha explícita contra el timestamp fijo). CQL no tiene edad en milisegundos (hallazgo 3);
  - la composición: `Abs(d)`, `-d`, `d + 1` con `d` = -2^31, y como control `d + 1L`, que es Long y da el número;
  - controles que no cambian: Time, una mezcla Date y DateTime, y los años 1 y 9999 en días y horas;
  - una incertidumbre adentro, una que cruza el borde y una enteramente afuera.
- **Área `durationrange` ampliada** con unos 150 casos de borde, generados por `scripts/durationrange-area.py` y contrastados contra Java y cql-execution:
  - cada unidad que puede pasarse del rango, en el borde exacto y a ±1, en los dos signos;
  - Duration, Difference y edades;
  - DateTime con y sin offset, e incertidumbres que cruzan el borde.
- **El oráculo de día juliano** pasa a dar null fuera del rango. El test cuenta cuántas filas revisa por clase (dentro del rango, en el borde, fuera), con un mínimo para cada una (hallazgo 6).
- Las 362 divergencias quedan reclasificadas, y no queda ninguna «defecto propio (1c35)».
- La diferencial contra main cambia solo en filas fuera del rango, si las hay en las 9 áreas. eCQM sigue en 23 coincidentes y 1 excusado.
- Los tests nuevos fallan en main, las mutaciones se atrapan todas y `make check` está en verde.

## Tests y fuentes de datos

- **Tests a mano** en `engine_durationrange_test.go`. Los valores esperados salen de la regla y de la aritmética de las sondas: los números de las dos referencias coinciden con cql-execution donde el valor cabe.
- **El área generada**, también importada por ELM.
- **El oráculo independiente**, sin `time` ni `funcs`.

## Mutaciones previstas

- El borde con `>` en lugar de `>=` en 2^31, y el mismo error de un lado en -2^31.
- Sin control en Difference.
- Sin control en las incertidumbres.
- El control solo en una de las dos cotas.
- Las edades por un camino que no pasa por el control.
- La duración fuera de `certainInteger`: `Abs(d)` vuelve a dar 2^31.
- El control antes de restar las cotas, o con la conversión a 32 bits hecha antes de tiempo (un `int32(...)` que da la vuelta).
- El oráculo sin el borde.
- Una respuesta del área que se aparta del oráculo.

## Ronda adversarial (Codex, 04-11-2026)

| # | Severidad | Hallazgo | Resolución |
|---|---|---|---|
| 1 | Alta | 22 filas de `differences` (como D127) ya dan null en Java. Reclasificarlas como «Java wraps» no pasa `TestTheDivergencesAreConsistent` (`engine_oracle_test.go`). | **Aceptado.** Pasan a coincidir y se borran; las demás se clasifican fila por fila. |
| 2 | Alta | `Abs(d)`, `-d` y `d + 1`, con `d` = -2^31 o 2^31-1 de una duración, dan 2147483648 por AST y por ELM: `certainInteger` no reconoce los nodos temporales. Verificado con una sonda. | **Aceptado** en el camino con plan (alcance 3). El ELM sin plan queda como en #135. |
| 3 | Media | `CalculateAgeInMillisecondsAt` no existe en CQL 1.5.3. Verificado: «unknown function». | **Aceptado.** Se retira, y se agregan `AgeIn…` y `CalculateAgeIn…` sin fecha explícita. |
| 4 | Media | Faltaban `DurationOf` y `DifferenceOf` en el inventario. Hoy dan 2678400000. | **Aceptado.** Ya pasan por `DurationBetweenAt` y `DifferenceBetweenAt` (`eval/evaluator.go:729-752`), así que solo se agregan tests por AST y ELM. Se suman controles de Time y de la mezcla Date y DateTime. |
| 5 | Media | D1 es una interpretación, y las 47 filas esperadas no la prueban. No hay alternativa mejor respaldada. | **Aceptado.** D1 la decide el usuario, y la evidencia queda separada de la decisión. |
| 6 | Media | El oráculo descarta las incertidumbres con filtros. Faltan mutaciones de saturación, de signo y de conversión a 32 bits antes de tiempo. | **Aceptado.** El test cuenta filas por clase, y se agregan las mutaciones y la documentación de los `XBetween`. |

## Decisión del usuario (04-11-2026)

- **D1:** la opción (a). Una incertidumbre con alguna cota fuera del rango es null entera.
- **Plan:** aprobado con los seis hallazgos aceptados, incluida la parte de `certainInteger`.

## Implementación y validación (04-11-2026)

La implementación la hizo un subagente Sonnet.

**Código:**
- `funcs/datetime_advanced.go`: `inIntegerRange` controla el resultado de `DurationBetweenAt` y `DifferenceBetweenAt` (alcance 1 y D1).
- `eval/integer_range.go`: `certainInteger` reconoce los cuatro nodos y las funciones de edad (alcance 3).
- `funcs/temporal.go` y `MillisecondsBetween`: la documentación de los `XBetween` (alcance 4).

**Tests:**
- `TestDurationsAtTheIntegerEdge`: unas 95 filas a mano, por AST y ELM, a -03:00 y +05:45.
- `TestADurationIsAnIntegerForCertain`: la composición, solo por AST.
- El oráculo de día juliano da null fuera del rango y cuenta las filas por clase: 1355 dentro, 176 en el borde y 382 fuera.
- `differenceOracle`, en `engine_differences_test.go`, también da null fuera del rango.
- Una fila de `engine_integer_overflow_test.go` pasa a null.

**Área `durationrange`:** suma 273 casos de borde, 1948 en total, y las referencias se regeneraron. Hay 893 divergencias, todas de referencia, clasificadas fila por fila por `scratchpad/r022/classify.py`. El script comprueba en cada fila la evidencia de la referencia:
- **cql-execution da un número de JS fuera del rango:** 447 filas, 397 en `durationrange` y 50 en `differences`.
- **Java da la vuelta en 32 bits:** 412 filas, 390 en `durationrange` y 22 en `differences`. El script verifica que la diferencia con la cuenta verdadera sea un múltiplo de 2^32. La cuenta verdadera la da cql-execution o, en los Interval en UTC, la calcula Python a partir de los dos instantes.
- **Java da la vuelta en las cotas de una incertidumbre** y lanza «Invalid Interval»: 16 filas, 10 en `durationrange` y 6 en `differences`. El script verifica las dos cotas módulo 2^32.
- **cql-execution corre el Interval por el offset de la evaluación** y su respuesta queda dentro del rango, aunque la cuenta verdadera está fuera: 4 filas. Además hay 6 del mismo defecto, ya conocido del 021, en que la cuenta queda dentro.
- **Java da un Interval simple donde va una incertidumbre:** 8 filas.

Las 22 filas de `differences` en que Java ya daba null ahora coinciden (hallazgo 1). No queda ninguna fila que cite 1c35.

**Mutaciones:**
- `scripts/mutations/022-duration-out-of-range.sh`: atrapa 16 de 16.
- El 021 atrapa 9 de 9 y el 020, 15 de 15.

**Diferencial contra main:** idéntica en las 9 áreas.

**eCQM:** 23 coincidentes y 1 excusado.

## Code review

**Ronda 1 (Codex, 04-11-2026): un hallazgo de severidad media, confirmado con una sonda y corregido.**
- **El hallazgo:** la aritmética sobre una incertidumbre de duración no aplicaba el borde de 32 bits. Con `N` = `milliseconds between @2012-01-25T20:31:23.648Z and @2012-01-01T00:00Z` = `[-2147483648, -2147423649]`, `-N` y `Abs(N)` daban `[2147423649, 2147483648]`. `P + 100000` daba cotas fuera del rango, y `P + 100000 > 2147400000` daba true.
- **La causa:** `pastInteger` miraba solo valores Integer, y la negación de una incertidumbre no pasaba por `withinInteger`.
- **La corrección:** la hizo un subagente Sonnet. `pastInteger32` cubre la incertidumbre de Integers, que queda null entera según D1, y la negación pasa por `withinInteger`.
- **Tests:** seis filas nuevas en `TestADurationIsAnIntegerForCertain`. Los controles siguen dentro del rango: `P - 100000`, `-P` y `P + 100000L` (Long).
- **Mutaciones:** dos nuevas; el script atrapa 18 de 18.
- **Lo demás:** Codex probó tres mutaciones propias y las tres se atraparon. Revisó 15 filas de divergencias, de las cinco clases, y su evidencia se sostiene. No encontró falsos positivos de `certainInteger` con funciones de edad de una librería incluida.

**Ronda 2 (Codex, 04-11-2026): convergió.** No dejó hallazgos de severidad media o mayor.
- Revisó la resta, la multiplicación, `Sum`, las comparaciones, los controles Long y el ELM sin plan; todo se mantiene.
- `Predecessor` y `Successor` ya rechazaban incertidumbres en main (`types/successor.go:114`).
- Un hallazgo bajo, corregido: no había test para la cota baja de una incertidumbre. Se agregaron `N - 1` y `(N - 1) < 0`, los dos null, y una mutación que revisa solo la cota alta. El script del 022 atrapa 19 de 19.

## Cierre (04-11-2026)

- **`make check` en verde,** con 580 mutaciones (`scratchpad/r022/check2.log`). Se corrió después de la corrección de la ronda 1 y antes de la ronda 2.
- **Lo que cambió después:** la ronda 2 solo sumó dos filas de test y una mutación. Para eso se corrieron el test, el script del 022 (19 de 19), `go test ./...` y el lint, sin hallazgos.
- **Diferencial contra main:** idéntica en las 9 áreas.
- **eCQM:** 23 coincidentes y 1 excusado.

## Dependencias

Ninguna.
