# 017: las edades como duración, y el Date junto a un DateTime

Rama `fix/age-duration`. Viene del 016, que dividió su alcance el 31-10-2026 por decisión del usuario: las edades por `DurationBetween` y el Date junto a un DateTime pasaron aquí.

## Por qué

CQL define una edad como una duración: `CalculateAgeInYearsAt(@1965, @2019-01-01)` es `years between @1965 and @2019-01-01`. Hoy el motor calcula las edades por su cuenta (`funcs/clinical.go`):
- años y meses llaman a `YearsBetween`/`MonthsBetween`, que nunca dan una incertidumbre;
- semanas y días parsean el texto del valor (`toTime`) y restan horas, así que una fecha parcial cuenta desde su primer día y un DateTime con hora no se lee;
- horas, minutos y segundos no existen.

Además, cuando un Date va junto a un DateTime en una duración, el Date se lee en UTC y el DateTime sin offset en el offset por defecto, en vez de convertir el Date con ToDateTime.

## Revisión previa (02-11-2026)

### `gofhir/server` (congelado, 4592d561)

No implementa edades ni duraciones: usa `gofhir/cql` v1.23.1. No hay nada que tomar.

### Especificación (CQL 1.5.3)

- **CalculateAge / CalculateAgeAt** (Appendix B): hay sobrecargas Date y DateTime. Las Date cubren years, months, weeks y days. Las DateTime cubren además hours, minutes y seconds. El cálculo «is a datetime duration», así que una fecha de nacimiento o un asOf imprecisos para la unidad dan una incertidumbre, y una comparación con ella puede dar null. Una fecha de nacimiento o un asOf null dan null. Ejemplo: `CalculateAgeInYearsAt(@2000-01-01, @2015-01-01)` = 15. CalculateAgeIn… es CalculateAgeIn…At contra la fecha actual. La sección Age toma Today() para años y meses, y Now() para las unidades más finas.
- **Duration** (Appendix B): «the number of whole calendar periods for the specified precision». Con operandos Date y DateTime mezclados, «the Date values will be implicitly converted to DateTime». El ejemplo de incertidumbre es `months between @2012-01-02 and @2012` = `[0, 10]`.
- **ToDateTime** de un Date: «the time components unspecified, except for the timezone offset, which will be set to the timezone offset of the evaluation request timestamp».
- **DateTime**: sin offset, «the timezone offset of the evaluation request timestamp is assumed». ToDateTime de un Date da un DateTime «with the time components unspecified».

### El traductor de Java (cql-to-elm de engine 5.4.0, `using FHIR`, contexto Patient)

| CQL | ELM |
|---|---|
| `AgeInYears()` | `CalculateAge(precision=Year; Patient.birthDate.value)`, un Date |
| `AgeInDays()` | `CalculateAge(precision=Day; ToDateTime(Patient.birthDate.value))` |
| `AgeInHours()` | `CalculateAge(precision=Hour; ToDateTime(Patient.birthDate.value))` |
| `AgeInYearsAt(start of "Measurement Period")` | `CalculateAgeAt(precision=Year; ToDateTime(birthDate), Start(...))` |
| `AgeInYearsAt(@2019-01-01)` | `CalculateAgeAt(precision=Year; birthDate, Date)` |
| `AgeInDaysAt(start of "Measurement Period")` | `CalculateAgeAt(precision=Day; ToDateTime(birthDate), Start(...))` |

La sobrecarga sale del tipo del asOf: Date con Today() para años y meses sin asOf, y DateTime con Now() para días y unidades más finas. ELM en `scratchpad/r017/elm/ages.elm.json`.

### Sondas (evaluación a 2026-01-01T00:00:00-03:00, como el arnés)

Corpus `scratchpad/r017/in/probe.cql`, contra Java 5.4.0 y cql-execution 3.3.2:

| # | Expresión | Java | cql-execution | Motor (main) |
|---|---|---|---|---|
| D1 | `CalculateAgeInYearsAt(@1965, @2019-01-01)` | 54 | `[53, 54]` | 54 |
| D3 | `CalculateAgeInWeeksAt(@2019-01, @2019-03-01)` | `[4, 8]` | `[4, 8]` | **8** |
| D4 | `CalculateAgeInDaysAt(@2019-01, @2019-03-01)` | `[29, 59]` | `[29, 59]` | **59** |
| D18 | `CalculateAgeInMonthsAt(@1965, @2019-01-01)` | `[637, 648]` | `[636, 648]` | **648** |
| D6 | `CalculateAgeInDaysAt(@2019-01-01T10:00, @2019-01-02T09:00)` | 0 | 0 | **null** |
| D8 | `CalculateAgeInHoursAt(@2019-01-01T10:00, @2019-01-02T09:00)` | 23 | `[22, 23]` | **error: unknown function** |
| D15 | `CalculateAgeInMinutesAt(...)`, la misma pareja | 1380 | `[1379, 1380]` | **error** |
| D16 | `CalculateAgeInSecondsAt(@2019-01-01T10:00, @2019-01-01T10:01)` | null | `[0, 119]` | **error** |
| D19 | `CalculateAgeInDaysAt(@2019-01-01, @2019-01-02T09:00:00.000Z)` | 1 | `[0, 1]` | 1 |
| D21 | `days between @2012-01-01T23:59:59.999 and @2012-01-03` | 1 | `[1, 2]` | **0** |
| D26 | `days between @2012-01-03 and @2012-01-01T23:59:59.999` | -1 | `[-2, -1]` | **0** |
| D25 | `hours between @2012-01-01 and @2012-01-02T05:00` | `[6, 29]` | `[5, 29]` | **32** |
| D28 | `@2012-01-01 = @2012-01-01T00:00:00.000Z` | false | false | **null** |

Las demás sondas (D2, D5, D7, D9-D14, D17, D22-D24, D27, D29, D30) dan lo mismo que Java. D1, D2, D7, D9 y D24 difieren de cql-execution por la misma ambigüedad que D46/D266/D297 (016): cql-execution lee todo DateTime incompleto como un rango.

`AgeInHours()` y `AgeInHoursAt(...)` tampoco existen en el motor («unknown function»). Una fecha de nacimiento parcial en el contexto Patient da edades exactas: con `birthDate` 2000-06 y evaluación al 2020-03-01, `AgeInDays()` da 7213.

### Conclusión

Hay cuatro defectos propios, todos en las edades y las duraciones:
1. **Las edades no son duraciones:** D3, D4 y D18 deberían ser inciertas, y lo son en las dos referencias.
2. **Una edad en días o semanas con DateTime da null** (D6).
3. **Faltan las edades en horas, minutos y segundos** (D8, D15, D16), para CalculateAgeIn… (At) y AgeIn… (At).
4. **Un Date junto a un DateTime en una duración no se convierte con ToDateTime** (D21, D26 y D25). Con la conversión, la regla del 016 da `[1, 2]`, `[-2, -1]` y `[5, 29]`: lo mismo que cql-execution, y lo que da el ejemplo normativo de la misma forma (`days between @2017-08-07T17:00 and @2017-08-14T` = `[6, 7]`, donde Java da 6).

D28 (`=` entre un Date y un DateTime) es una comparación, no una duración. Es otro frente y va al backlog.

## Decisión del usuario (02-11-2026)

El usuario aprobó el plan ajustado, dividido: el 017 son las edades, y el Date junto a un DateTime en las duraciones pasa al 018.

## Ronda adversarial (Opus independiente, 02-11-2026)

Codex sigue sin cuota hasta el 04-11-2026; por decisión del usuario, la ronda la hace un Opus independiente. Verifiqué cada hallazgo con una sonda propia (`scratchpad/r017/probe2.cql`, contra Java y cql-execution).

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| H1 | El camino ELM no existe: `elm/importer.go` no tiene `CalculateAge` ni `CalculateAgeAt`, y su `default:` (línea 444) convierte un nodo desconocido en un null literal. `evalImported` pasa por el traductor propio, que escribe `FunctionRef`, así que no lo prueba | alta | Aceptado: el importador lee `CalculateAge` y `CalculateAgeAt` con `precision`, y los tests importan el ELM del traductor de Java (`ages.elm.json`). Que un nodo desconocido sea un error de importación es otro frente: va al backlog (es parte de 1c9) |
| H2 | Los valores esperados contradecían la regla del 016: con ToDateTime explícito, el motor ya da `[1, 2]` en D21 (Java 1), `[-2, -1]` en D26, `[636, 648]` en D18, `[5, 29]` en D25 (Java `[6, 29]`) y `[0, 119]` en D16 (Java null) | alta | Aceptado: el criterio de salida lleva el valor exacto de cada fila y la clase de su divergencia, decidida de antemano (ver abajo) |
| M1 | `evalImported` no fija el timestamp de evaluación: `CalculateAgeInYears(@1965-06-01T10:00)` da 60 por AST y 61 por ELM, con el reloj real | media | Aceptado: el helper fija el mismo timestamp que `evalConversion` |
| M2 | «sema rechaza las horas con un Date» no tiene respaldo: el traductor de Java acepta `CalculateAgeInHoursAt(@2019-01-01, @2019-01-02)` por la conversión implícita (luego su runtime falla con «Unsupported unit: Hour»), y cql-execution da 24 | media | Aceptado: un Date se convierte a DateTime y se usa la sobrecarga DateTime; no se rechaza. Da `[0, 47]`; Java (error) y cql-execution (24) son defectos de la referencia |
| M3 | El punto 4 cambia filas de `durations` y `uncertainty`, fuera de las edades | media | Aceptado: se divide. El punto 4 pasa al **018**, y el 017 queda en las edades. Decisión del usuario pendiente |
| L1 | Un `birthDate` completo convertido tiene precisión de día, así que `AgeInHours()` y las unidades más finas siempre son inciertas con un Patient R4 (`[0, 47]` entre dos días). Java da null | baja | Aceptado: la consecuencia se declara y se compara |
| L2 | `CalculateAgeIn…()` sin asOf también toma Today() con un Date y Now() con un DateTime | baja | Aceptado |
| L3 | `toTime` no se puede borrar: lo usan `DayOfWeek` y dos caminos de `funcs/datetime_advanced.go` | baja | Aceptado: sale solo de `clinical.go` |
| L4 | Faltan bordes: edades negativas, asOf null, y un asOf con Z mientras se evalúa a -03:00 (`CalculateAgeInYearsAt(@2000-01-01, @2026-01-01T00:00:00.000Z)`: Java 26, cql-execution 25) | baja | Aceptado: van a la grilla y a la corrida de eCQM |

## Alcance (ajustado tras la ronda)

1. **Las edades por `DurationBetweenAt`:** CalculateAgeIn{Years,Months,Weeks,Days}At con Date o DateTime pasan a ser la duración en esa unidad, en el marco de la evaluación. `toTime` y la resta de horas salen de `clinical.go`.
2. **Las siete unidades:** CalculateAgeIn{Hours,Minutes,Seconds}(At) y AgeIn{Hours,Minutes,Seconds}(At). Un Date se convierte a DateTime (conversión implícita), como lo resuelve el traductor de Java.
3. **Las sobrecargas sin asOf y con asOf**, como el traductor: un Date de nacimiento con Today() para años y meses, y ToDateTime(birthDate) con Now() para días y unidades más finas, tanto en `AgeIn…()` como en `CalculateAgeIn…()`. Con un asOf DateTime, la fecha de nacimiento se convierte con ToDateTime.
4. **El importador de ELM** lee `CalculateAge` y `CalculateAgeAt` con `precision`. Para que el ELM de Java de `AgeIn…()` se evalúe, también lee `ToDateTime` y un `Property` con ruta (`birthDate.value`) como accesos anidados (agregado en la implementación). Los constructores `Date`/`DateTime` del ELM de Java siguen dando null (backlog 1c9); el test del ELM fija esas definiciones como null, por nombre, para que su arreglo se vea.
5. `evalImported` fija el timestamp de evaluación.

## Fuera del alcance

- **El 018:** el Date junto a un DateTime en `DurationBetween` y `duration in X of` (D21, D25 y D26), que cambia filas de `durations` y `uncertainty`.
- D28 y toda comparación (`=`, `<`, `same as`) entre un Date y un DateTime.
- La regla de incertidumbre de `DurationBetween` (decisión del 016).
- Que el importador falle con un nodo desconocido (backlog 1c9).
- `DurationBetween` normaliza los offsets también en días: `days between @2025-12-31T23:00:00.000-03:00 and @2026-01-01T01:00:00.000Z` da 0, y por el texto de Duration (sin normalizar por debajo de la hora) daría 1. Va al backlog (1c28), con Java medido antes de tocarlo.
- `difference in … between` y el backlog 1c26 y 1c27 del 016.

## Consecuencias previstas

- **Corregido durante la implementación (02-11-2026):** con un `birthDate` completo, `AgeInDays()` y `AgeInWeeks()` también cambian. El traductor los escribe con `ToDateTime(birthDate)`, que queda con precisión de día, contra Now(), que llega a milisegundos. Por la regla del 016 (la del ejemplo normativo `days between @2017-08-07T17:00 and @2017-08-14T` = `[6, 7]`), dan una incertidumbre para todo paciente. Medido con un nacido el 1980-01-01 y evaluación a 2026-01-01T00:00-03:00 (`scratchpad/r017/pp`): Java da `AgeInDays()` = 16802, `AgeInWeeks()` = 2400 y `AgeInDaysAt(@2020-01-01T10:00:00.000Z)` = 14610, todos exactos (y 6 en el ejemplo normativo); cql-execution da `[16801, 16802]`, 2400 y `[14609, 14610]`. **Decisión del usuario (02-11-2026):** se sigue la especificación (incierta), como el ejemplo normativo, cql-execution y el ELM de Java importado. La diferencia con Java queda como deliberada y cita este plan. En eCQM no cambia nada (23 coincidentes y 1 excusado, medido con el código de la rama): el corpus solo usa `AgeInYearsAt` (218) y `AgeInMonthsAt` (28).
- `AgeInDays() = AgeInDaysAt(Now())` puede pasar de true a null.
- `AgeInHours()` y las unidades más finas siempre dan una incertidumbre con un `birthDate` Date de FHIR R4 (Java da `AgeInHours()` = `[403225, 403248]`, también incierta).
- En eCQM, el 016 midió que el corpus no tiene fechas de nacimiento parciales. Se mide antes y después con `TestECQMCases`.

## Criterio de salida

Cada fila da exactamente este valor, por AST y por ELM (el propio y el de Java):

| Fila | Valor | Java | cql-execution | Clase prevista de la divergencia |
|---|---|---|---|---|
| D3 `CalculateAgeInWeeksAt(@2019-01, @2019-03-01)` | `[4, 8]` | igual | igual | — |
| D4 `CalculateAgeInDaysAt(@2019-01, @2019-03-01)` | `[29, 59]` | igual | igual | — |
| D18 `CalculateAgeInMonthsAt(@1965, @2019-01-01)` | `[636, 648]` | `[637, 648]` | igual | defecto de Java: la cota baja de la regla, como en `months between` |
| D6 `CalculateAgeInDaysAt(@2019-01-01T10:00, @2019-01-02T09:00)` | 0 | igual | igual | — |
| D8 `CalculateAgeInHoursAt(@2019-01-01T10:00, @2019-01-02T09:00)` | 23 | igual | `[22, 23]` | deliberada (016, D46/D266/D297) |
| D15 la misma pareja, en minutos | `[1379, 1380]` | 1380 | igual | defecto de Java: los dos operandos están a la precisión de la unidad, como `years between DateTime(2005) and DateTime(2010)` = `[4, 5]` de la suite (corregido en la implementación: el plan decía 1380) |
| D16 `CalculateAgeInSecondsAt(@2019-01-01T10:00, @2019-01-01T10:01)` | `[0, 119]` | null | igual | defecto de Java |
| probe2 D5 `CalculateAgeInHoursAt(@2019-01-01, @2019-01-02)` | `[0, 47]` | error | 24 | defecto de las dos |
| probe2 D9 edad negativa | -10 | igual | igual | — |
| probe2 D10 asOf con Z | 26 | igual | 25 | defecto de cql-execution: Duration normaliza los offsets «only when the comparison precision is hours, minutes, seconds, or milliseconds», y en años se cuentan los campos |

Además:
- Las edades en las siete unidades existen por AST y por ELM.
- Hay una nueva área de snapshot `ages` con cientos de casos (fechas completas y parciales, Date y DateTime, las siete unidades, negativas, nulls, offsets declarados, y el contexto Patient con `birthDate` completo y parcial), contra Java y cql-execution, con toda divergencia clasificada con evidencia.
- Los tests fallan en main, y cada oráculo tiene sus mutaciones.
- La diferencial contra main es idéntica fuera de las edades; las filas de edades de `durations` que cambien se listan una por una.
- eCQM da 23 coincidentes y 1 excusado, y `make check` está en verde.

## Tests y fuentes de datos

- A mano, con la sección de la especificación en un comentario: el ejemplo normativo de CalculateAgeAt, las sondas de esta revisión y los bordes (29 de febrero, fin de mes, edades negativas, asOf null, offset declarado).
- Una grilla de fechas de nacimiento (año, mes, día, minuto, ms) contra asOf (Date y DateTime), en las siete unidades. El oráculo es la definición de la especificación: la edad es `<unit> between birthDate and asOf`. El caso a mano se compara con la duración escrita como CQL (metamórfico), y la grilla contra un oráculo en Go que enumera los extremos del rango sin llamar al motor.
- El ELM de Java (`CalculateAge`, `CalculateAgeAt`) importado.
- El contexto Patient con `birthDate` de `fhir-data.json` y pacientes sintéticos escritos a mano (nunca datos reales).

## Mutaciones previstas

- Las edades de vuelta a `YearsBetween`/`MonthsBetween` y a la resta de horas.
- La sobrecarga Date tomada como DateTime y al revés; Today() por Now().
- El importador sin `CalculateAge`/`CalculateAgeAt`, o con la precisión equivocada.
- Una unidad de las nuevas cambiada por otra.
- `evalImported` sin timestamp fijo.
- Una fila deliberada que cita un plan que no la nombra; el área fuera del manifiesto y de las referencias.


## Encontrado en los tests (02-11-2026)

### El área `ages`

`testdata/snapshots/ages.cql` tiene 462 definiciones. Contra Java 5.4.0 y cql-execution 3.3.2 hay 518 divergencias, todas con veredicto:

| Referencia | Clase | Filas |
|---|---|---|
| java | deliberate | 26 |
| java | own defect | 1 |
| java | reference defect | 320 |
| js | deliberate | 31 |
| js | reference defect | 140 |

Cómo se clasificaron: `TestTheAgesAreaIsItsOracle` comprueba que cada `CalculateAgeIn…At` del área con dos literales sin offset da lo que da el oráculo independiente de la grilla, que enumera en Go sin llamar al motor. Sobre esa base, cada clase sale de una regla que verifica su condición (script `scratchpad/r017/classify.py`):
- Java da error «Unsupported unit» con un Date en horas, minutos o segundos (la especificación convierte el Date).
- Java da null donde hay una incertidumbre.
- Una referencia da un número donde un operando no llega a la precisión de la unidad.
- Java da un Interval simple con las mismas cotas que la incertidumbre.
- Una referencia da otras cotas que los extremos del oráculo.
- cql-execution cuenta las horas sobre el Date, sin convertirlo.

Las 17 filas que ninguna regla cubre se revisaron a mano: los fines de mes sin 29 de febrero, D22, D29, D30, D35, D36, D37, D39 y D40.

**Diferencias deliberadas que este plan nombra:**
- **Fin de mes** (decisión A del plan 015, que una edad hereda porque es esa duración), contra Java: D21, D53, D58, D63, D68, D114, D115, D116, D119, D120, D121, D124, D125, D126, D129, D130, D131, D144, D145, D146, D149, D150, D151, D154, D159.
- **La hora del día en un aniversario** (decisión B del plan 015), contra Java: D40.
- **Dos operandos más finos que la unidad** (decisión del plan 016), contra cql-execution, que lee todo DateTime incompleto como un rango: D8, D18, D38, D48, D50, D51, D53, D55, D56, D58, D59, D60, D61, D63, D64, D68, D69, D115, D116, D119, D120, D121, D124, D129, D145, D146, D149, D150, D151, D154, D159.

**Defecto propio:** D37, `CalculateAgeInYears(null)`. Un null calza igual con la sobrecarga Date y con la DateTime, así que la llamada es ambigua, como informa el traductor de Java; el motor la acepta y da null. Es de sema y va al backlog (1c30).

### Code review

Codex sigue sin cuota (vuelve el 04-11-2026 03:52). Por decisión del usuario, cada ronda la hace un Opus nuevo e independiente.

**Ronda 1 (02-11-2026).** Antes: `make check` en verde (537 mutaciones, 0 issues de lint; `scratchpad/r017/check017b.log`), diferencial idéntica en las nueve áreas, eCQM 23+1. El revisor corrió unas 80 sondas por los dos caminos y por el contexto Patient, comparó contra main, revisó a mano filas de cada clase y corrió los dos scripts de mutaciones (23 y 23). No dejó hallazgos de severidad media o mayor; converge.

| # | Hallazgo (bajo) | Resolución |
|---|---|---|
| 1 | Ningún test fija Today() al offset de la evaluación: con `EvaluationTimestamp.UTC()` todo pasa, porque los arneses evalúan a medianoche con offset negativo | Aceptado: `TestTodayIsTheEvaluationsDate` (2026-01-01T05:00+14:00) y su mutación |
| 2 | El oráculo de la grilla copia la regla de `DurationBetweenAt` para dos fechas más finas que la unidad (semanas y unidades más finas, entre los instantes tal como están escritos) | Aceptado como límite conocido, declarado en su comentario: la regla no cambia en este plan |
| 3 | Se quitan las funciones exportadas `CalculateAgeInYears/Months/Weeks/Days` y `AgeIn*At` de `funcs`: un cambio de API | Aceptado: se declara en el PR. El único consumidor, `gofhir/server`, no importa `funcs` |
| 4 | Estilo: un comentario con otro nombre (`ageUnits`), `TestAPartialBirthDateIsReadAtItsStart` afirmaba lo contrario de su nombre, el mapa `ageUnits` era la identidad, y faltaba la columna Clase en la tabla de este plan | Corregidos: el mapa se quitó y el test se renombró (`TestAPartialBirthDateIsAnUncertaintyOverItsPeriod`) |

### Cierre (02-11-2026)

- Una ronda de review, que convergió; los cuatro hallazgos bajos quedaron corregidos o declarados.
- `make check` en verde sobre el código final: exit 0, 538 mutaciones detectadas y 0 issues de lint (`scratchpad/r017/check017c.log`).
- Diferencial contra main después del review: idéntica en las nueve áreas (`scratchpad/r016/diff.sh`). Las otras 22 áreas de snapshot no cambian ninguna respuesta.
- eCQM: 23 coincidentes y 1 excusado de 24.
- Mutaciones: 24 del 017 y 23 del 015 reancladas, todas detectadas.

### Otras correcciones al plan

- Siete mutaciones de `scripts/mutations/015-durations.sh` apuntaban al código de edades que el 017 reemplazó (`YearsBetween`/`MonthsBetween` en `clinical.go`, los `case` por unidad del evaluador y el espaciado de sema): el primer `make check` dio «NOT APPLIED». Se reanclaron al código nuevo con la misma intención, y las 23 del 015 se detectan.

- D15 es `[1379, 1380]`, no 1380 (ver la tabla del criterio de salida).
- El importador también lee `ToDateTime` y un `Property` con ruta.

## Dependencias

Ninguna nueva.
