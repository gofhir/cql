# 015: `years between` y `months between` por períodos de calendario

Backlog 1c25, grupo G3b. Rama: `fix/years-between-leap-1c25`. Estado: implementado el 30-10-2026; review convergido en la ronda 2 el 31-10-2026; cerrado sin commit.

## Por qué

En la suite al día (cqframework/cql-tests f724ee4, plan 012) fallan dos tests: `YearsBetweenLeapYearDatesEquals2` y `YearsBetweenLeapYearDateTimesEquals2`. Ambos esperan 2 de `years between @2012-02-29 and @2014-02-28`, y el motor da 1.

La edad de una persona se calcula con el mismo criterio, así que el defecto también la alcanza. Una persona nacida un 29 de febrero tenía, según el motor, un año menos en el 28 de febrero de cada año no bisiesto.

## Revisión previa (sondas del 30-10-2026, scratchpad `r015/`)

**`gofhir/server`** (c4f9f4bc) no calcula duraciones ni edades por su cuenta. Solo las nombra en dos tests (`internal/cqltrace/recorder_test.go` e `internal/service/ops/cql/library_test.go`). No hay nada que tomar.

**El dueño es este repo.** `funcs/temporal.go` (`YearsBetween` y `MonthsBetween`) compara a mano el mes y el día. `funcs/clinical.go` (`CalculateAgeInYears` y `CalculateAgeInMonths`) tiene una copia propia del mismo criterio. Ninguna librería interviene: fhirpath solo entrega los valores.

**La especificación.** Las citas son de CQL 1.5.3.
- **Appendix B, Duration:** «returns the number of whole calendar periods for the specified precision between the first and second arguments. If the first argument is after the second argument, the result is negative. […] any fractional periods are dropped».
- **Author's Guide, Date and Time Arithmetic:**
  - «For variable length precisions (months and years), the operation uses the calendar length of the precision to determine the number of periods»;
  - `months between @2014-01-01 and @2014-03-15` da 2;
  - `duration in months between @2014-01-31 and @2014-02-01` da 0, «because there is less than one calendar month between the two dates».
- **Appendix B, CalculateAgeAt:** «The CalculateAgeAt operators are defined in terms of a datetime duration calculation». Las firmas son `CalculateAgeInYearsAt(birthDate Date, asOf Date) Integer` y sus pares de meses, semanas y días, con Date y con DateTime.
- **La suma de un año o un mes** a un día que no existe en el destino cae en el último día del mes. Las tres implementaciones lo hacen igual: `@2012-02-29 + 2 years` da `@2014-02-28`, y `@2014-01-31 + 1 month` da `@2014-02-28` (D47 y D48 de las sondas).

**Las sondas** (50 definiciones, contra Java 5.4.0 y cql-execution 3.3.2):

| Caso | Java | cql-execution | Motor |
|---|---|---|---|
| `years between @2012-02-29 and @2014-02-28` (D0) | 2 | 2 | 1 |
| `years between @2012-02-29 and @2013-02-28` (D1) | 1 | 1 | 0 |
| el mismo con hora igual (D5) | 2 | 2 | 1 |
| el mismo un segundo antes del aniversario (D6) | 2 | 1 | 1 |
| `years between @2014-01-15 and @2012-01-16` (D16) | -1 | -1 | -2 |
| `months between @2014-03-31 and @2014-02-28` (D27) | -1 | -1 | -2 |
| `months between @2014-01-15T10:00:00 and @2014-02-15T09:59:59` (D30) | 0 | 0 | 1 |
| `years between @2014-01-15T10:00:00 and @2015-01-15T09:59:59` (D31) | 0 | 0 | 1 |
| `months between @2014-01-31 and @2014-02-28` (D19) | 0 | 1 | 0 |
| `months between @2014-03-31 and @2014-04-30` (D23) | 0 | 1 | 0 |
| distinto offset, a precisión de años (D18) | 2 | 1 | 1 |
| `CalculateAgeInYearsAt(@2012-02-29, @2014-02-28)` (D34) | 2 | 2 | «unknown function» |

Además de la del 29 de febrero, la tabla muestra dos defectos propios en los que las dos referencias coinciden:
- **en sentido inverso**, el motor resta un período de más (D16 y D27);
- **la hora del día se ignora** al comparar el aniversario (D30 y D31).

**Java no es coherente consigo mismo.** Para años toma el 28 de febrero como aniversario del 29 (D0 da 2), pero para meses no toma el 28 de febrero como el mes cumplido desde el 31 de enero (D19 da 0). Tampoco mira la hora cuando el inicio es un 29 de febrero (D6), y sí la mira en otros días (D31).

## Decisiones tomadas (30-10-2026)

- **A. El fin de mes:** un período entero es el mayor n tal que `low + n períodos <= high`, con la suma del motor, que coincide con las dos referencias. Es una interpretación: la especificación dice «whole calendar periods» y suma con ajuste al fin de mes, pero no dice que la duración sea la inversa de la suma, y las referencias difieren al fin de mes (D19 y D38). Así `months between @2014-01-31 and @2014-02-28` da 1, como cql-execution. Es la misma regla que da 2 años del 29 de febrero de 2012 al 28 de febrero de 2014, donde las dos referencias coinciden.
- **B. La hora cuenta:** un segundo antes del aniversario no se cumple el período (D6 da 1, como cql-execution).
- **C. Los offsets distintos a precisión de años, meses o días**, fuera del alcance. La especificación pide normalizar solo desde horas («only when the comparison precision is hours, minutes, seconds, or milliseconds»), que es la respuesta de Java. Cambiarlo alcanza a `days between` y a todas las frases de timing, así que va al backlog como frente propio.
- **D. Las edades:** pasan por el mismo cálculo, y se agregan `CalculateAgeInYearsAt`, `CalculateAgeInMonthsAt`, `CalculateAgeInWeeksAt` y `CalculateAgeInDaysAt`.

En sentido inverso, la regla es simétrica: el resultado es menos el período de `high` a `low`. Con A, `months between @2014-02-28 and @2014-01-31` da -1, como cql-execution (D25).

## Alcance

1. `YearsBetween` y `MonthsBetween` (`funcs/temporal.go`) cuentan períodos con la regla de A y B, en los dos sentidos.
2. `CalculateAgeInYears` y `CalculateAgeInMonths` (`funcs/clinical.go`) usan esas mismas funciones, y con ellas `AgeInYears`, `AgeInYearsAt`, `AgeInMonths` y `AgeInMonthsAt`.
3. `CalculateAgeIn{Years,Months,Weeks,Days}At` se despachan en `eval` y se tipan en `sema`, como sus pares sin `At`.
4. La suma que usa el conteo es el ajuste al fin de mes que usa el operador `+` del motor, compartido en Go, no una copia. No es el operador mismo: ese falla al pasar del año 9999, y el conteo prueba candidatos que pueden pasarlo.
5. **El aniversario se compara a la precisión común** de los dos operandos, como los operadores de comparación. Así, un Date junto a un DateTime no supone medianoche: ToDateTime deja sin especificar la hora. Se compara hasta el milisegundo.
6. **Los bordes del calendario:**
   - `@0001-01-01` es una fecha válida: el conteo no usa la fecha cero de Go como marca de «sin valor», y hoy `years between @0001-01-01 and @0002-01-01` da null;
   - un candidato que pasa del 9999 cuenta como posterior a `high`.
7. **La incertidumbre:** `durationBetweenUncertain` llama al conteo exacto en sus dos cotas, así que esas cotas cambian con él (`years between @2014 and @2012` da hoy `[-3, -2]`). Entran al alcance como consecuencia, con sus valores esperados. Cada cota es el conteo exacto entre los extremos posibles («Results in an uncertainty from the shortest possible duration, to the longest possible duration», Language Semantics).
8. **El arnés de cql-execution** codifica una incertidumbre con sus dos cotas. Hoy la escribe como `Unencoded:Uncertainty` (`internal/referenceharness/js/evaluate.js`), y dos incertidumbres distintas parecen iguales.
9. Un área de snapshots nueva, `durations`, con las sondas y una grilla de fechas generada, comparada contra las dos referencias. Cada divergencia queda clasificada.
10. Los dos tests G3b salen de `knownFailures`.

## Fuera del alcance (al backlog)

- Los offsets distintos a precisión de años, meses o días (decisión C).
- Java no devuelve incertidumbre: `years between @2012 and @2014` da 2. La especificación y la suite (`Interval[ 4, 5 ]` para `years between DateTime(2005) and DateTime(2010)`) piden el intervalo. Esas filas se clasifican como defecto de Java, sin cambiar el motor.
- `CalculateAgeIn{Hours,Minutes,Seconds}At` y la edad con incertidumbre.
- `CalculateAgeAt` en el ELM: el importador no conoce el nodo (backlog 4).
- `CalculateAgeInYears(@2012-02-29, @2014-02-28)` con dos argumentos: el motor lo acepta, y el traductor de referencia lo rechaza por no tener esa firma (D42).

## Diferencias deliberadas

En el área `durations`, cada una nombrada una sola vez:

- **Decisión A, frente a Java:** el mes que termina antes del día de inicio, al fin de mes. Son D19, D21, D23, D24, D25, D28, D38, D56, D57, D60, D61, D64, D65, D76, D77, D82, D83, D98, D99, D106, D107, D238, D239, D242, D243, D250, D251, D254, D255, D275, D288, D289, D291 y D292. cql-execution coincide con el motor en todas.
- **Decisión B, frente a Java:** la hora del día cuenta en un aniversario de febrero (el 29 o el 28), que Java no mira. Son D6, D17, D40, D210, D211, D216, D217, D222, D223, D234, D235, D263, D267 y D293. cql-execution coincide con el motor en todas.

Las demás filas del área no son deliberadas:

- **Defecto de Java, sin incertidumbre** (la lista de «Fuera del alcance»): D43, D269, D276, D277, D278, D279 y D280. D281 es el mismo defecto con otra forma: Java da el intervalo `[7, 18]`, y la cota inferior verdadera es 6, de `2005-12-31` a `2006-07-01`.
- **Defecto de cql-execution, D18:** normaliza a UTC al contar años. Java y el motor coinciden con la especificación, que normaliza solo desde horas.
- **Defecto propio diferido, D46, D266 y D297:** cql-execution da la incertidumbre `[1, 2]` donde el motor da 2 (como Java), porque a un operando le falta la hora del día. Qué cuenta como insuficiente lo decide `DurationBetween` desde antes de este plan (precisión menor o igual a la unidad), y este plan no lo cambia. Va al backlog junto con la decisión C. El punto 5 del alcance (comparar a la precisión común) solo fija el conteo cuando la duración ya es exacta.

Las filas de la decisión C no son deliberadas: son defectos propios diferidos, porque la especificación da la respuesta de Java. La grilla no las alcanza: el área solo tiene un caso con offsets distintos (D18), y el motor ya da lo que dice la especificación. La regla A no puede normalizar a UTC a precisión de años o meses, porque eso agrandaría el defecto.

## Criterio de salida

- Los dos tests G3b pasan y salen de `knownFailures`.
- En el área `durations`, las divergencias con las dos referencias son solo las deliberadas de A y B, más las de la decisión C, clasificadas.
  - *Ajuste del 31-10-2026 (review, ronda 1):* la decisión C no deja filas. Quedan además dos clases que el plan no previó, cada una con su evidencia en la fila: los defectos de las referencias (Java da un número donde la especificación pide una incertidumbre, o cuenta mal una cota; cql-execution normaliza offsets a precisión de año) y los tres defectos propios diferidos D46, D266 y D297, que dependen de la regla de incertidumbre de `DurationBetween` y van al backlog (ver Diferencias deliberadas).
- `make check` en verde, con las mutaciones nuevas detectadas.
- La diferencial contra main sobre todo el corpus: cada definición que cambia se explica por A, B o los dos defectos propios.

## Tests

- **Casos de la especificación y de la suite:** los ejemplos de Duration y del Author's Guide, `YearsBetweenLeapYear*` y los demás `between` de `CqlDateTimeOperatorsTest.xml`.
- **Una grilla generada:**
  - días de inicio entre el 28 y el 31, y de fin entre el 27 y el 31;
  - meses de enero a abril, en años bisiestos y comunes (2011 a 2013);
  - años y meses, en los dos sentidos, con Date y con DateTime con hora igual, un segundo antes y un segundo después.

  Son unos 300 casos, con las respuestas de las dos referencias en el área.
- **Metamórfico, en Go, sobre la misma grilla:**
  - con `n = years between low and high`, se cumple `low + n years <= high < low + (n+1) years`;
  - lo mismo para meses;
  - y `between(high, low) = -between(low, high)`.

  Usa el `+` del motor, que coincide con las referencias.
- **Edades:**
  - `CalculateAgeInYearsAt` igual a `years between` sobre la grilla;
  - `AgeInYearsAt` y `AgeInYears` con un Patient nacido el 29 de febrero, en R4, con un `asOf` explícito (los snapshots congelan el reloj en el 1 de enero);
  - umbrales clínicos alrededor del cumpleaños: `AgeInYearsAt(x) >= 18` el día antes, el día y el día después.
- **Las entradas:** el mismo caso por `years between`, por `duration in years of` un intervalo y por el ELM importado (`DurationBetween`), con la misma respuesta.
- **Milisegundos y precisiones mezcladas:**
  - `.001` antes y después del aniversario;
  - Date contra DateTime;
  - un DateTime a precisión de día contra uno a segundos.
- **Los bordes:** `@0001-01-01`, `@9999-12-31` y la inversa de cada uno. La aserción metamórfica de `n+1` se salta cuando el candidato pasa del 9999.
- **La incertidumbre:** las cotas de `years between @2014 and @2012`, `months between @2014-03 and @2014-02` y sus inversas, contra cql-execution con el arnés que las codifica.
- **Los valores esperados** salen de la especificación o de las referencias, nunca de correr el motor.

## Implementación

- `funcs/temporal.go`: `YearsBetween` y `MonthsBetween` pasan por `calendarPeriodsBetween`.
  - Lee cada operando como sus componentes escritos (`partsOf`, año a milisegundo) y los compara a la precisión común (`compareParts`).
  - El período entero es el mayor n con `low + n períodos <= high`. El mes o el año de `low + n` es el de `high` o el anterior, así que basta probar n y n-1.
  - El sentido inverso es `-between(high, low)`.
  - La suma es `Date.AddDuration` de fhirpath sobre la fecha de `low`: la misma que usa el `+` del motor, con su ajuste al fin de mes. El ayudante del ajuste (`shiftCalendarMonths`) no se exporta, así que se llama a la operación que lo contiene y no se copia. Un error de rango cuenta como «posterior a high».
  - Ya no usa `toGoTime` ni la fecha cero de Go como marca de «sin valor».
- `funcs/clinical.go`: `CalculateAgeInYears` y `CalculateAgeInMonths` son esos dos conteos. Sin fecha de nacimiento o sin `asOf` siguen dando null. Las semanas y los días no cambian.
- `eval/evaluator.go`: `CalculateAgeIn{Years,Months,Weeks,Days}At` comparten el caso de su par sin `At`.
- `sema/systemfuncs.go`: las cuatro se tipan `Integer`.
- `internal/referenceharness/js/evaluate.js` y `answers/answers.go`: una incertidumbre de cql-execution se escribe como `Uncertainty` con sus dos cotas, y se lee igual que la del motor (`uncertainty`).
- `durationBetweenUncertain` no cambia: llama al conteo exacto en sus dos cotas, y esas cotas cambian con él (punto 7).
- Tests: `engine_durations_test.go` (seis tests), `TestAnUncertaintyIsReadWithBothBounds` y `TestTheAgeAtOperatorsAreTypedAsIntegers`. Antes de la implementación fallaban los seis primeros; los otros dos se verifican con las mutaciones.
- Área `durations` (297 definiciones: 49 sondas y una grilla de 248 desde D50), en el manifiesto y en `scripts/references.sh`; sus referencias están regeneradas. Los dos tests G3b salen de `knownFailures`, y D21 y D22 de `suitegaps`, que pasa a 38.

## Encontrado en los tests

- **La suma compartida no puede pasar del 9999 en el sentido directo.** El año de `low + n períodos` es a lo sumo el de `high`, así que el error de rango de `AddDuration` es inalcanzable al contar. Se deja como guarda, sin mutación: solo la aserción `n+1` del metamórfico pasaría del 9999, y la grilla no llega allí. Los bordes 0001 y 9999 se prueban con conteos.
- **D18 da 2 sin tocar los offsets.** Al comparar los componentes escritos, los dos años salen iguales a Java, y es cql-execution quien normaliza. La decisión C sigue fuera: a precisión de horas o más fina, el offset todavía no se normaliza al contar años o meses.
- **Java ignora la hora en algunos aniversarios de febrero que no son el 29** (D234 y D235: `@2011-02-28T12:34:56` a `@2013-02-28T12:34:55` da 2). La decisión B los cubre; por eso la lista de B no se limita al inicio en 29 de febrero.
- **D281: Java da `[7, 18]`** para `months between DateTime(2005) and DateTime(2006, 7)`, donde la cota inferior es 6. Es un defecto de Java con otra forma que los de la incertidumbre ausente.
- **Date junto a DateTime, o DateTime a precisión de día, contra cql-execution (D46, D266, D297):** cql-execution da la incertidumbre `[1, 2]`, y el motor y Java dan 2. El Author's Guide da `[6, 7]` para `days between @2017-08-07T17:00 and @2017-08-14T`, así que la regla actual de `DurationBetween` (incertidumbre solo si la precisión es menor o igual a la unidad) probablemente es un defecto propio anterior a este plan. Va al backlog.
- **Dos tests anteriores fijaban el comportamiento viejo** (`engine_default_unit_test.go`). `TestAgeIsRightOnTheAnniversary` daba a quien nació un 29 de febrero su aniversario el 1 de marzo, «una convención»; ahora es el 28, como el `+` del motor y las dos referencias. `TestAgeWithoutABirthDate` daba -11 para un nacimiento en el futuro (2019-06-01 contra 2030-01-15); ahora da -10, las 10 edades enteras al revés, como D16 y D27 en sentido inverso. Los valores nuevos salen de la especificación y de las referencias, no del motor.
- **El motor limita la fuente de una librería a 100 KiB**, así que el metamórfico evalúa la grilla en bloques de 120 tramos.
- **Una mutación de `durationBetweenUncertain` no compilaba** por una variable sin uso; se reescribe con `_ = lowLatest`.

## Mutaciones

`scripts/mutations/015-durations.sh`: 23 mutaciones, todas detectadas.

- el día del mes conservado, sin el ajuste al fin de mes;
- la hora del día ignorada;
- el milisegundo ignorado;
- un período de más en sentido inverso;
- la precisión más fina de las dos en lugar de la común;
- la estimación que nunca retrocede un período;
- los meses contados como años;
- la fecha cero de Go como null;
- la cota inferior de la incertidumbre desde el otro extremo;
- la edad en años y en meses por una cuenta propia;
- `CalculateAgeIn{Years,Months,Weeks,Days}At` sin despachar, y `CalculateAgeInYearsAt` sin tipar;
- la incertidumbre de la referencia sin leer, y con una sola cota (referencia y motor);
- una diferencia deliberada que el plan no nombra;
- el área fuera del manifiesto y de las referencias;
- un test G3b de vuelta en `suitegaps`.

## Ronda adversarial (Codex, 30-10-2026)

Por la regla del usuario del 30-10-2026, la ronda adversarial y el review son solo de Codex. Cada hallazgo se verificó con su sonda (`/private/tmp/claude-501/review015.out`).

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| 1 | La incertidumbre llama al conteo exacto, así que sus cotas cambian: `years between @2014 and @2012` da `[-3, -2]` | alta | Aceptado: entra al alcance (punto 7), con valores esperados |
| 2 | La comparación de A no dice qué pasa con precisiones mezcladas, y `.001` antes del aniversario da 1 | alta | Aceptado: se compara a la precisión común, hasta el milisegundo (punto 5), con tests |
| 3 | `@0001-01-01` da null por la fecha cero de Go, y la suma del motor falla en el 10000 | alta | Aceptado (punto 6). La suma compartida es el ajuste, no el operador |
| 4 | A es defendible, pero el texto citado no la prueba como única, y el metamórfico valida la regla elegida, no su corrección | media | Aceptado: A queda como interpretación, y el metamórfico, como coherencia interna |
| 5 | El arnés de cql-execution descarta las cotas de una incertidumbre | alta | Aceptado: se codifican (punto 8) |
| 6 | Las filas de la decisión C son defectos, no alternativas justificadas | media | Aceptado: se clasifican como defecto propio diferido, y A no normaliza a UTC |
| 7 | Faltan entradas (ELM, `duration … of`) y edades alrededor del cumpleaños; los snapshots congelan el 1 de enero | alta | Aceptado: tests por entrada y de umbral, con `asOf` explícito |

## Code review

Solo Codex, sobre una copia en `/private/tmp/claude-501/cql015r1`. Antes de la ronda 1, los 6 tests de `engine_durations_test.go` fallan en main y la diferencial contra main (corpus de la semilla 8008, `r015/diff.sh`) da las nueve áreas idénticas. Ese corpus no usa duraciones ni edades, así que solo descarta cambios colaterales.

### Ronda 1 (31-10-2026)

Codex no encontró defectos en el conteo: probó 16 casos propios (siglos bisiestos, sentido negativo, milisegundos, precisión mezclada y parcial, años 0001 y 9999). Las 23 mutaciones del script fallan, y también las que inyectó a mano (sin el ajuste de fin de mes, sin la hora, con el signo invertido, con las cotas rotas, con el lector Go roto).

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| 1 | `evaluate.js` reconoce una incertidumbre por el campo `isUncertainty`, y una tupla puede tenerlo: `Tuple { isUncertainty: true, low: 1, high: 2 }` sale como `Uncertainty` | media | Confirmado: con el codificador anterior, D298 (nuevo, esa tupla) sale `Uncertainty` en cql-execution, y Java da `Tuple`. Corregido: se compara por la clase, que cql-execution 3.3.2 exporta solo desde `lib/datatypes/uncertainty`. Con la corrección, las dos referencias dan `Tuple` y D298 no diverge. Al regenerar, solo cambió D298 |
| 2 | El test nuevo de `answers` arma el JSON a mano y nunca corre el codificador JS | baja | Aceptado en parte: D298 cubre el caso al regenerar las referencias (`scripts/references.sh check`). Un test ejecutable del JS va al backlog, porque el arnés JS no tiene uno y crearlo amplía el alcance |
| 3 | D46, D266 y D297 chocan con el criterio de salida, que no los permitía | baja | Aceptado: se ajusta el criterio, con la fecha y el motivo, y se informa al usuario |

El `make check` posterior encontró una mutación sin aplicar: su ancla en `015-durations.sh` buscaba la cuenta 297 del manifiesto, y con D298 es 298. Se corrigió el ancla.

### Ronda 2 (31-10-2026): converge

Codex revisó solo los cambios de la ronda 1, sobre la copia `/private/tmp/claude-501/cql015r2`, y no dejó hallazgos. Comprobó cuatro cosas:
- La ruta del `require` funciona con cql-execution 3.3.2.
- El codificador conserva las dos cotas de una incertidumbre real (`years between @2012 and @2014`): sola, dentro de una lista y como cota de un intervalo.
- D298 sale `Tuple`.
- El ajuste del criterio no esconde una corrección prometida, porque la condición que decide cuándo hay incertidumbre es anterior a este plan.

Las 23 mutaciones del 015 se detectan.

## Dependencias

Ninguna nueva.
