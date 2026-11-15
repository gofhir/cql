# 024: las comparaciones con precisión normalizan al offset de la evaluación (backlog 1c45)

Rama `fix/same-precision-offset`, desde main 13974e2 (14-11-2026).

## Revisión previa (14-11-2026)

### El defecto

Las comparaciones con precisión pasan los valores a UTC antes de leer sus componentes. Son `same X as`, `same X or before/after`, `before/after X of`, las frases de timing con precisión y los operadores de intervalos con precisión. Esa lectura está en `temporalComponentsAgainst` (`eval/evaluator.go`), y todas pasan por `componentsAtPrecision`.

Con un offset de evaluación de horas enteras, como el `-03:00` del arnés, UTC y el offset de la evaluación cortan las horas en los mismos instantes, y nada cambia. Con un offset fraccionario (`+05:30`, `+05:45`, `-09:30`) no es así: dos instantes de la misma hora local pueden caer en dos horas UTC distintas.

CQL 1.5.3, Appendix B, Same As (https://cql.hl7.org/N1A/09-b-cqlreference.html): «When comparing DateTime values with different timezone offsets, implementations should normalize to the timezone offset of the evaluation request timestamp, but only when the comparison precision is hours, minutes, seconds, or milliseconds». Lo mismo dicen Same Or Before, Same Or After, Before y After.

`difference in hours` ya normaliza al offset de la evaluación (`funcs.DifferenceBetweenAt` con `e.evaluationFrame()`). El motor hoy no es coherente consigo mismo.

### Sondas (`scratchpad/r024`)

El motor y las dos referencias, Java 5.4.0 y cql-execution 3.3.2, en copias del arnés con la evaluación a `-03:00`, `+05:30` y `+05:45` (`scratchpad/r024/out/*.json`, la tabla en la conversación del 14-11). 21 expresiones. En `-03:00`, las tres coinciden en todas menos D4.

**Defectos propios, con las dos referencias de acuerdo:**

| # | Expresión | Offset | Motor | Java | cql-execution |
|---|---|---|---|---|---|
| D6 | `@2020-01-01T00:10:00 same hour as @2020-01-01T00:40:00` | +05:30 | false | true | true |
| D20 | `@2020-01-01T00:10:00 before hour of @2020-01-01T00:40:00` | +05:30 | true | false | false |
| D21 | `@2020-01-01T00:10:00 same hour as @2020-01-01T00:40:00+05:30` | +05:30, +05:45 | false | true | true |

Ninguno de los valores sin offset cambia de hora en el offset de la evaluación. El motor los pasa a UTC, donde las 00:10 y las 00:40 de `+05:30` son las 18:40 y las 19:10 del día anterior.

**Donde la lectura literal se aparta de las referencias:** con offsets escritos distintos, las dos referencias normalizan al offset del operando izquierdo, como ya mostró el 018 en `mixeddates`. `@2020-01-01T00:10:00Z same hour as @2020-01-01T00:40:00Z` da true en las dos a `+05:30`. Normalizado al offset de la evaluación son las 05:40 y las 06:10, y la respuesta es false.

**D4:** `difference in hours between @…00:10:00Z and @…00:40:00Z` da 1 en el motor a `+05:30` y 0 en las dos referencias. El motor ya normaliza ahí al offset de la evaluación; es el punto 3 del borrador para HL7.

### gofhir/server

No implementa CQL: usa esta librería (único consumidor). No hay nada que tomar.

### Decisión del usuario (14-11-2026)

**Se normaliza al offset de la evaluación**, como dice la 1.5.3 y como ya hace `difference`. Los casos con offsets escritos distintos a un offset fraccionario, donde las referencias dan otra cosa, quedan como deliberados.

## Plan (14-11-2026)

### Alcance

1. **Leer en el offset de la evaluación.** `temporalComponentsAgainst` recibe el marco (`*time.Location`) y lee los componentes de un DateTime con offset en ese marco, no en UTC. El marco baja desde el `Evaluator` (`e.evaluationFrame()`) por `componentsAtPrecision` y sus llamadores:
   - `temporalCompareAtPrecision` y sus envoltorios `temporalSameAs`, `temporalBefore`, `temporalAfter`, `temporalSameOrBefore` y `temporalSameOrAfter`;
   - `membershipAtPrecision`, `intervalIncludesAtPrecision`, `evalIntervalProperlyContainsPoint`, `successorOf` y `widenForPrecision`;
   - `compareValues` solo si la llamada lleva precisión.

   Hoy son funciones sueltas; pasan a recibir el marco como parámetro, o a ser métodos del `Evaluator`, lo que cambie menos.
2. **Lo que no cambia:**
   - la lectura tal como está escrita (`readAsWritten`): con un Date o un Time, y a una precisión más gruesa que la hora, se leen los dígitos;
   - la ventana de offset (`offsetWindowOf`) para un valor con offset junto a uno sin offset;
   - `types.CompareTemporal` (`=`, `<` sin precisión), que normaliza a UTC con `inUTC`. Con instantes completos, el marco no cambia el orden. Si en la validación aparece un caso con precisiones distintas donde sí cambia, va al backlog y no se toca aquí.
3. **El arnés de referencia con otro offset de evaluación.** Hoy las dos referencias evalúan siempre a `-03:00` (`Evaluate.java`, `js/evaluate.js`).
   - Ambos leen el offset de una variable de entorno, con `-03:00` como valor por defecto, para que las áreas existentes no cambien.
   - Un área nueva, `fractionaloffset`, se evalúa a `+05:30` en las dos referencias y en el motor. Lleva las 21 sondas más una enumeración: cada operador con precisión, de hora a milisegundo, con pares sin offset, con el mismo offset, con offsets distintos y en el cambio de día.
   - `scripts/references.sh` la genera; sus divergencias se clasifican en `fractionaloffset.divergences.tsv`.
   - Con eso queda cubierto en parte el 1c48 (el área `mixeddates` usa un solo marco).
4. **Veredictos.**
   - Los casos con offsets escritos distintos donde las referencias normalizan al operando izquierdo son deliberados: citan este plan, la decisión del 14-11 y su espejo cuando lo tiene.
   - D4 queda como estaba, deliberado por el borrador a HL7.

### Tests (antes del código)

- `engine_same_precision_offset_test.go`: D6, D20 y D21 a `+05:30` y `+05:45`, con lo que dan las dos referencias. Cada caso de la lectura literal, citando Appendix B. Los mismos casos a `-03:00` y `Z`, que no cambian. Un caso a `-09:30`.
- Cada operador del alcance 1 con un caso que falla en main.
- Un test que fija que `difference in hours` y `same hour as` cortan las horas en el mismo marco: si dos instantes están a 0 límites de hora, son de la misma hora, a todo offset de la lista.

### Mutaciones (`scripts/mutations/024-same-precision-offset.sh`)

- La lectura vuelve a UTC.
- La lectura usa el offset escrito del primer operando (lo que hacen las referencias).
- Un llamador pasa `time.UTC` en vez del marco.
- El arnés ignora la variable de entorno: el área `fractionaloffset` tiene que fallar.
- Una fila deliberada que cita un plan que no la nombra.

### Criterio de salida

- Los tests nuevos pasan; fallan en main.
- `go test ./...` y `make check` en verde.
- La diferencial contra main es idéntica en los corpus existentes, todos a `-03:00`.
- El volcado eCQM es idéntico a `-03:00` y a `+14:00`, offsets de horas enteras. Se agrega `+05:30` y sus diferencias contra main se listan y se explican.
- Las referencias regeneradas solo agregan el área nueva.

### Fuera

- `types.CompareTemporal` sin precisión.
- El orden de normalizar y truncar en `difference` con offsets fraccionarios: es el punto 3 del borrador a HL7.
- Otros offsets de evaluación en `mixeddates` (resto del 1c48).

### Dependencias

Ninguna nueva.

## Ronda adversarial 1 del plan (Codex, 14-11-2026)

La evidencia está en `/private/tmp/claude-501/r024p/.probe/`: `attack.cql`, `attack.{java,js}.json` y `attack.go.txt`. Verificada leyendo esos archivos.

| # | Severidad | Hallazgo | Resolución |
|---|---|---|---|
| A1 | alta | La 1.5.3 normaliza solo «when comparing DateTime values with different timezone offsets». Con el mismo offset escrito, como `+01:00` contra `+01:00` a `+05:30`, Java, cql-execution y el motor dan true. Normalizar siempre daría false. El ejemplo `Z` contra `Z` de la revisión previa no tiene offsets distintos. | **Aceptado.** La regla queda así: con el mismo offset efectivo, los componentes se leen como están escritos; con offsets distintos, en el offset de la evaluación. Un valor sin offset toma el de la evaluación, así que dos valores sin offset tienen el mismo. Es la letra de la 1.5.3 y está dentro de la decisión del usuario. Con esta regla, `Z` contra `Z` da true, como las referencias. Quedan deliberados solo los pares de offsets distintos donde las referencias normalizan al operando izquierdo. |
| A2 | alta | `types.CompareTemporal` queda fuera. `@2020-01-01T00Z = @2020-01-01T01:40+01:00` da null; escrito en `+05:30`, da false. | **Aceptado como decisión explícita, fuera de alcance.** El motor y las dos referencias dan null en todos los offsets (D20, D12 del ataque). No hay divergencia que medir. Un valor a la hora no se puede llevar a un offset fraccionario sin perder su hora. Va al backlog con estos casos. |
| A3 | alta | cql-execution 3.3.2 trunca un offset de evaluación fraccionario al ubicar un literal sin offset: `timezoneoffset from @2020-01-01T00:10:00` da 5 a `+05:30`, y `Now()` da 5.5. Viene de `lib/elm/datetime.js:56` y `literal.js:49` (`parseInt`). | **Aceptado, verificado** en `attack.js.json` (D17 5.5, D18 5) y en el código del módulo. Es un defecto de la referencia: en los casos con valores sin offset, el oráculo del área es Java. Hay un test centinela de D17 a D19 que falla si cql-execution lo corrige, y un borrador de issue para cql-execution, que se publica solo con aprobación. No se compensa en Go. |
| M1 | media | Tres consumidores del arnés fijan `-03:00`: `engine_snapshot_test.go:182`, `engine_oracle_test.go:133` e `internal/referenceharness/differential/compare/main.go:188`. Además, una variable exportada en `references.sh` contaminaría las demás áreas. | **Aceptado.** El offset es por área, en una tabla de Go que leen los tres. La variable se pasa solo a la invocación del área nueva. Un test verifica que el `evaluationDateTime` de cada snapshot es el que la tabla espera para esa área. |
| M2 | media | El alcance no está completo. `meets hour of` da true a `+05:30` y las referencias dan false (D8). `expand … per hour` no trunca a la hora (D5): Java da una lista vacía y cql-execution trunca. `collapse … per hour` deja dos intervalos donde las referencias juntan uno (D7). | **Aceptado en parte.** `meets` entra: el marco tiene que llegar a `successorOf`. `expand` y `collapse` con límites que no caen en la unidad no dependen del offset de la evaluación: pasan igual a `-03:00` (verificarlo en el paso 5). Van al backlog como defectos aparte. |
| M3 | media | Faltan mutaciones independientes por cada rama del marco, por el mismo offset, por la selección del área en Go y por los metadatos del offset. Una mutación del arnés tiene que regenerar las salidas de las referencias. La invariante entre `difference` y `same` pasa si las dos normalizan mal. | **Aceptado.** Las mutaciones quedan en la lista de abajo. La invariante se reemplaza por casos fijos con respuesta de Java. |

### Plan ajustado (14-11-2026)

**Regla:** en una comparación con precisión de hora o más fina, si los offsets efectivos de los dos valores son iguales, se comparan los componentes escritos. Si son distintos, se leen los dos en el offset de la evaluación. Un valor sin offset tiene el de la evaluación. A una precisión más gruesa que la hora, y con un Date o un Time, no cambia nada (`readAsWritten`).

**Alcance:**
1. La regla, en `componentsAtPrecision` y `temporalComponentsAgainst`. El marco baja desde el `Evaluator` a todos los llamadores, incluido `successorOf` (`meets`).
2. **Arnés:**
   - `Evaluate.java` y `js/evaluate.js` leen el offset de una variable de entorno, con `-03:00` por defecto.
   - En `references.sh`, la variable se pasa solo al área nueva.
   - El offset por área va en una tabla de Go que usan `engine_snapshot_test.go`, `engine_oracle_test.go` y `differential/compare`.
   - Un test verifica que cada snapshot tiene el `evaluationDateTime` que su área espera.
3. **Área `fractionaloffset` a `+05:30`.** Lleva las sondas de la revisión previa y las del ataque, y una enumeración: cada operador con precisión de hora a milisegundo, sobre pares sin offset, con el mismo offset, con offsets distintos y en el cambio de día.
4. **Veredictos:**
   - Los pares de offsets distintos donde las referencias normalizan al operando izquierdo son deliberados y citan este plan.
   - Los valores sin offset en cql-execution son defectos de la referencia (A3).
   - `difference` (D4) se queda como está.

**Tests (primero):**
- D6, D20, D21 y D8 a `+05:30`, `+05:45` y `-09:30`, con la respuesta de Java.
- El mismo offset escrito distinto al de la evaluación (A1).
- Un par de offsets distintos con la lectura literal, citando Appendix B.
- `-03:00` y `Z` sin cambios.
- El centinela de cql-execution (A3).
- El test de los metadatos del offset por área.

**Mutaciones:**
- La lectura vuelve a UTC.
- La lectura normaliza aunque los offsets coincidan.
- La lectura usa el offset del operando izquierdo.
- `successorOf` recibe `time.UTC`, y lo mismo cada uno de los demás llamadores.
- La tabla de offsets da `-03:00` para el área nueva.
- Un snapshot regenerado sin la variable de entorno.
- Una respuesta esperada alterada.
- Una fila deliberada sin nombrar en el plan.

**Backlog nuevo:**
- `CompareTemporal` con una precisión de hora y offsets distintos (A2).
- `expand … per` y `collapse … per` con límites fuera de la unidad (M2).
- El issue para cql-execution sobre el offset fraccionario (A3).

**Decisión del usuario (14-11-2026): plan ajustado aprobado.**

## Veredictos del área `fractionaloffset` (paso 7, 14-11-2026)

El área tiene 1444 definiciones evaluadas a +05:30, y 232 divergencias:
- **94 deliberadas.** Son pares de offsets escritos distintos, donde las referencias leen en el offset de un operando. Por la decisión del 14-11, el motor lee los dos en el offset de la evaluación. Cubren D4 y las filas D1, D25, D31, D34, D99, D101, D103, D104, D108, D110, D112, D125, D127, D128, D132, D134, D135, D136, D146, D147, D148, D207, D209, D211, D212, D216, D218, D220, D233, D235, D236, D240, D242, D243, D244, D254, D255, D256, D315, D317, D319, D320, D324, D326, D328, D341, D343, D344, D348, D350, D351, D352, D362, D363, D364, D423, D425, D427, D428, D432, D434, D436, D449, D451, D452, D456, D458, D459, D460, D470, D471, D472.
- **138 defectos de las referencias.** Son 137 de cql-execution, que trunca +05:30 a +05:00 (A3), y uno de Java: `Invalid Interval` en D27.
- **Ningún defecto propio.**

## Validación (paso 7, 14-11-2026)

- **Java a -09:30.** Hice una copia del arnés con `CQL_EVALUATION_OFFSET=-09:30`. Da D6 true, D20 false, D21 false y `meets hour of` false, que es lo que esperan los tests.
- **`go test ./...`** pasa. Ningún área existente cambia.
- **Diferencial contra main:** idéntica en los nueve corpus.
- **Volcado eCQM:** ahora también evalúa a +05:30, además de -03:00 y +14:00. Main y la rama dan las mismas 3345 líneas: el corpus no tiene comparaciones con precisión entre offsets que se aparten de UTC.
- **Mutaciones:** `scripts/mutations/024-same-precision-offset.sh` detecta las 30.

## Code review, ronda 1 (Codex, 14-11-2026)

| # | Severidad | Hallazgo | Resolución |
|---|---|---|---|
| 1 | media | `scripts/references.sh` deja exportado `CQL_EVALUATION_OFFSET=+05:30` después del último área. Las evaluaciones que siguen lo heredan: `encoder`, `runtime` y `membershipmodel` en Java, y `runtime` en cql-execution. | **Aceptado, verificado leyendo el script, y corregido.** Cada bucle corta la variable al terminar. `TestTheReferencesScriptTellsTheHarnessTheOffsetOfEachArea` falla si falta ese corte. |
| 2 | media | En la ventana asimétrica, que ambos extremos coincidan ya no prueba nada. El offset igual al del otro valor se lee como está escrito, y los demás en el offset de la evaluación. Codex dio este ejemplo: un parámetro `00:50` sin offset contra `01:10-12:00` con `before hour of`, a +05:30, dio true, y es incierto. | **Aceptado, confirmado con un test que fallaba, y corregido.** Los dos extremos se leen en el offset de la evaluación, que es monótono a lo largo de la ventana, y además se compara el punto donde el valor sin offset toma el offset del otro. `TestAnUnplacedValueIsUncertainWhereTheWindowDisagrees` cubre los dos casos: el de Codex, y uno donde los extremos coinciden y solo ese punto difiere. |
| 3 | media | D118, D119, D334 y D335 de cql-execution figuraban como deliberadas, pero son defectos de la referencia: un `+05:30` escrito contra un valor sin offset, que también está en +05:30. Lo que los separa es solo la truncación (A3). | **Aceptado, verificado y corregido en el generador.** Si el único offset escrito es el de la evaluación, la fila es un defecto de la referencia. Al regenerar, cambian solo esas cuatro filas. Ahora hay 94 deliberadas y 138 defectos de las referencias. |
| 4 | baja | Los tests del arnés miran el texto de `Evaluate.java`: un Java que ignora el valor de la variable los pasa igual. | **Aceptado como límite conocido.** Probar el comportamiento necesita Docker. Hay evidencia de comportamiento: la corrida a -09:30 de la validación dio `evaluationDateTime` -09:30, y `TestEachAreaIsEvaluatedAtItsOffset` verifica ese metadato en cada snapshot generado. |

Mutaciones después de la ronda: el script detecta las 32, con dos nuevas, una por el hallazgo 1 y otra por el 2. Además reancló una que había roto la corrección del 2.

## Code review, ronda 2 (Codex, 14-11-2026)

Codex confirmó que las tres correcciones de la ronda 1 se sostienen. Sobre la ventana del hallazgo 2 dice que la prueba vale: todo offset distinto del otro se lee en un solo marco, y es monótono; el único que se lee como está escrito es el del otro. Lo comprobó con un barrido de 384.006 lecturas.

| # | Severidad | Hallazgo | Resolución |
|---|---|---|---|
| 1 | media | Un valor a la hora corrido a un offset fraccionario se leía por su primer instante, y así la hora quedaba como conocida. `@2020-01-01T00Z before minute of @2020-01-01T01:40+01:00` a +05:30 pasó de null en main a true: `00Z` es de 05:30 a 06:29, y eso abarca las 06:10. Lo mismo pasaba con un parámetro `2020-01-01T00` sin offset. | **Aceptado.** Confirmado con `TestAnHourMovedByAFractionalOffsetIsTheWholeHour`, que falla sin la corrección; los valores esperados son de Java a +05:30 y -09:30, salvo `same hour as`, que es deliberado. **Corrección:** en una comparación con precisión, un valor cuya unidad cae sobre dos unidades del marco se compara por su primer instante y por el último, y es cierta solo si los dos coinciden. En los demás lectores de componentes, la precisión se reduce a la que comparten los dos extremos. |

La corrección también resuelve un defecto propio anterior. Es `temporal` D22, `@2020-01-01T10+05:30 same hour as @2020-01-01T04Z`, que esperaba null según la decisión de offsets parciales del 008 (backlog 1c22). Ahora el motor da null: D22 pasa a deliberado, con las dos referencias en false.

Después de la ronda 2:
- `go test ./...` pasa.
- La diferencial contra main es idéntica en los nueve corpus.
- El volcado eCQM es idéntico en sus 3345 líneas.
- El script detecta las 34 mutaciones: hay dos nuevas, y reanclé cuatro que había movido la factorización de `readingFrameOf`.

**Ronda 3 (14-11-2026):** Codex se quedó sin cuota hasta las 22:02. Por decisión del usuario del 14-11, la ronda 3 la hace un revisor Opus independiente, que no conoce mi razonamiento. Es una excepción a la regla de que revisa solo Codex.

`make check` después de la ronda 2 falló en `scripts/mutations/019-date-promotion.sh`: una mutación anclada en el final de la línea `areas=` no se aplicó, porque esa línea ahora termina con `fractionaloffset`. Reanclé la mutación y el script vuelve a detectar todas.

## Code review, ronda 3 (revisor Opus independiente, 14-11-2026): no convergió

La evidencia está en `/private/tmp/claude-501/r024r/.probe/`. Verifiqué M1 y M2 con sondas en la rama y en main, a -03:00, +05:30 y +05:45.

| # | Severidad | Hallazgo | Estado |
|---|---|---|---|
| M1 | media | Los extremos se comparan exigiendo el mismo `cmp`. `same or after/before` solo necesita que el predicado coincida: si un extremo da 0 y el otro 1, `>=` es true igual. A -03:00, `@2020-01-01T10+05:30 same hour or after @2020-01-01T01:00` da null en la rama y true en main (verificado). También afecta a `overlaps hour of` por `endpoints.phrase`. | **Confirmado. Lo introdujo la corrección de la ronda 2.** Pendiente. |
| M2 | media | La reducción de precisión de los otros lectores de componentes baja a día, y `in`, `includes`, `during` y `properly` a la hora pierden respuestas ciertas. A -03:00, `@2020-01-01T10+05:30 in hour of Interval[@2020-01-01T01:00, @2020-01-01T03:00]` da null en la rama y true en main (verificado). | **Confirmado. Lo introdujo la corrección de la ronda 2.** Pendiente. |
| M3 | media | `intervalIncludesAtPrecision` no usa la ventana para un parámetro sin offset: lo lee en el offset de la evaluación. `P = 2020-01-01T00:50` con `P in hour of Interval[@…T00:00Z, @…T01:00Z]` da false a -03:00 y true en main. Con `same hour as`, en cambio, la ventana da null. | Sin verificar. Main también estaba mal: daba una respuesta cierta. Pendiente. |
| L | baja | Hay mutaciones que sobreviven: el extremo bajo de la ventana con `time.UTC`, y código muerto en `lastInstantOf` (casos 0-2, 4 y 5) y en `endsIn`. `TestTheSameWrittenOffsetIsReadAsWritten` y `TestACoarserPrecisionAndADateKeepTheDigitsWritten` pasan en main. Hay comentarios desactualizados (`framedOffsets`, «moves the written one to UTC»). | Pendiente. |

Con tres rondas sin converger, la regla es detenerse y consultar al usuario.

## Decisión del usuario (14-11-2026): revertir la ronda 2 y cerrar

- **Se revierte la corrección de la ronda 2:** `endsIn`, `lastInstantOf` y la reducción de precisión. Se van su test, sus dos mutaciones y el cambio de `temporal` D22, que vuelve a ser un defecto propio del 1c22. Se queda `readingFrameOf`, que es solo una factorización. Sin la corrección desaparecen M1 y M2 de la ronda 3: a -03:00, los dos casos vuelven a dar true, como en main (verificado).
- **M3:** en los lectores de componentes que no resuelven la ventana (`in`, `includes` y `during` por componentes), un par con un DateTime sin ningún offset vuelve a leerse como en main (`readingFrameOf`, caso `unplaced`). Así el 024 no cambia esa respuesta a -03:00. La respuesta de main está mal; tendría que ser null, por la ventana. `TestAnUnplacedValueInAnIntervalIsReadAsBefore` la documenta como defecto conocido, y una mutación la cubre.
- **L1:** corregidos los tres comentarios desactualizados.
- **Queda abierto (backlog):** el valor a la hora corrido a un offset fraccionario (hallazgo de la ronda 2). `@2020-01-01T00Z before minute of @2020-01-01T01:40+01:00` da true a +05:30 y +05:45, y null en main y en Java. Solo pasa con un offset de evaluación fraccionario. La corrección tiene que evaluar el predicado en cada extremo, también en los intervalos, y no comparar `cmp`.

Después de revertir:
- `go test ./...` pasa.
- La diferencial contra main es idéntica en los nueve corpus.
- El volcado eCQM es idéntico en sus 3345 líneas, a -03:00, +14:00 y +05:30.
- El script detecta las 32 mutaciones.
