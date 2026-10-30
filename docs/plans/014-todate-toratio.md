# 014: `ToDate` y `ToRatio`

Backlog 1c25, grupo G3f. Rama: `fix/todate-toratio-1c25`. Estado: paso 7 terminado (tests, implementación, área y mutaciones); sigue la diferencial y el review. Plan aprobado el 29-10-2026.

## Por qué

En la suite al día (cqframework/cql-tests f724ee4, plan 012), ocho tests fallan porque el motor responde «unknown function»:
- `ToDate1`, `ToDateYearMonth`, `ToDateYear`, `ToDateIgnoresTime`, `ToDateFromDateTime` y `ToDateMalformed`;
- `StringToRatio` y `StringMalformedToRatio`.

Los dos operadores son de CQL 1.5.3. `sema` ya los tipa (`sema/systemfuncs.go`, `todate` y `toratio`) y `funcs.ToDate` existe, pero el evaluador no los despacha.

## Revisión previa (sondas del 29-10-2026, scratchpad `r014/`)

**`gofhir/server`** no usa `ToDate`, `ToRatio` ni `ConvertsToRatio` (búsqueda en 35459972). No hay nada que tomar.

**La especificación.** Las citas son de CQL 1.5.3, Appendix B (`https://cql.hl7.org/N1A/09-b-cqlreference.html`).

- **ToDate:**
  - firmas `ToDate(argument DateTime) Date` y `ToDate(argument String) Date`;
  - «For the DateTime overload, the operator is equivalent to invoking date from on the DateTime value»;
  - para el String, «the string must be interpretable as a valid date value», con los ejemplos `'2014-01'`, `'2014-01-01'` y `'2014-01-01T12:30:00' // Still January 1st, 2014, time portion will be ignored`;
  - «If the input string is not formatted correctly, or does not represent a valid date value, the result is null».
- **ToRatio:**
  - firma `ToRatio(argument String) Ratio`;
  - «The operator accepts strings using the following format: `<quantity>:<quantity>` where `<quantity>` is the format used to by the ToQuantity operator»;
  - «If the input string is not formatted correctly, or cannot be interpreted as a valid Ratio value, the result is null».
- **ConvertsToDate y ConvertsToRatio:** «returns true if its argument is or can be converted to a Date [Ratio] value. See the ToDate [ToRatio] operator for a description of the supported conversions». Si no, false; null si el argumento es null.
- **ToString:** el Ratio se escribe `<quantity>:<quantity>`, y «The result of any ToString must be round-trippable back to the source value».

**Las referencias** (Java 5.4.0 y cql-execution 3.3.2). Son 216 sondas en `r014/probe014.cql` y 49 más, de la ronda adversarial, en `r014/probe014b.cql`. Las tablas completas, con la respuesta actual del motor, están en `r014/probe-table3.txt` y `r014/probe-table-b.txt`.

| Expresión | este motor (main) | Java | cql-execution | especificación |
|---|---|---|---|---|
| `ToDate('2014-01-01')`, `'2014-01'`, `'2014'`, `'0001-01-01'`, `'9999-12-31'`, `'2016-02-29'` | unknown function | la Date | la Date | la Date |
| `ToDate('2014-01-01T12:30:00')` y otras 13 con hora (con offset, con Z o solo con `T12`) | unknown function | null | null | `@2014-01-01` |
| `ToDate('2014-01-01T25:00:00')`, `'2014-01-01T'`, `'…Tabc'`, `'2014-02-30T10:00'` | unknown function | null | null | sin decir; ver la decisión B |
| `ToDate('2014-02-30')`, `'2014-13'`, `'2014-1-1'`, `' 2014-01-01'`, `'20140101'`, `'2014/01/01'`, `''`, `'2014-01-01Z'` | unknown function | null | null | null |
| `ToDate('0000-01-01')` | unknown function | null | `@0000-01-01` | null (el año va de 1 a 9999) |
| `ToDate(DateTime(2014, 1, 1, 23, 0, 0, 0, -5.0))`, `ToDate(@2014-01-01T23:59:59.999+14:00)` | unknown function | `@2014-01-01` | `@2014-01-01` | `date from`: `@2014-01-01` |
| `ToDate(@2014-01-01)` (no hay sobrecarga Date: el traductor pasa por DateTime) | unknown function | `@2014-01-01` | `@2014-01-01` | `@2014-01-01` |
| `ConvertsToDate('2014-01-01T12:30:00')` | false | false | false | true |
| `ConvertsToDate(@2014-01-01T10:00)`, `ConvertsToDate(5)` | true, false | error | true, false | true, false |
| `ToRatio('1.0 \'mg\':2.0 \'mg\'')`, `'1:2'`, `'1 : 2'`, `'1 \'mg\':2 \'mL\''`, `'1:0'`, `'0:0'`, `'1:2 '` | unknown function | el Ratio | el Ratio | el Ratio |
| `ToRatio('1.0 \'mg\';2.0 \'mg\'')`, `'1'`, `'1:2:3'`, `''` | unknown function | null | null | null |
| `ToRatio(':2')`, `'1:'`, `':'` | unknown function | error | null | null |
| `ToRatio(' 1:2')`, `'-1:2'`, `'1:-2'`, `'1e2:3'` | unknown function | el Ratio | null | lo que dé `ToQuantity` en cada lado: el Ratio |
| `ToRatio('1 \'foo\':2')`, `'1 mg:2 mg'`, `'1 day:2 days'` | unknown function | el Ratio | null | lo que dé `ToQuantity`: null (plan 010, decisión A) |
| `ToRatio('1\'mg\':2\'mg\'')` | unknown function | error | el Ratio | lo que dé `ToQuantity`: el Ratio |
| `ToRatio('1 \'{a:b}\':2')`, un `:` dentro de la unidad | unknown function | null | el Ratio | ver la decisión C |
| `ConvertsToRatio('1.0 \'mg\':2.0 \'mg\'')` | false | devuelve el String | true | true |
| `ConvertsToRatio(' 1:2')`, `'-1:2'`, `'1e2:3'` | true | devuelve el String | false | lo mismo que `ToRatio` |
| `ConvertsToRatio(1 'mg':2 'mg')`, un Ratio | false | devuelve el Ratio | true | true |
| `ToString(ToRatio('1.0 \'mg\':2.0 \'mg\''))` | unknown function | `1.0 'mg':2.0 'mg'` | `1 'mg' : 2 'mg'` | `<quantity>:<quantity>` |
| `(1 'mg':2 'mg').numerator`, un literal Ratio | null: el literal es un String | `1 'mg'` | `1 'mg'` | `1 'mg'` |
| `ToString(1.0 'mg':2.0 'mg')` | `1.0'mg':2.0'mg'`, el texto fuente | `1.0 'mg':2.0 'mg'` | `1 'mg' : 2 'mg'` | `<quantity>:<quantity>` |
| `(1:2) = (2:4)` / `(1:2) ~ (2:4)` | false / false, por texto | false / true | true / true | Equal por componentes; Equivalent cruzado |
| `convert '2014-01-01T12:00:00' to Date`, `convert 'abc' to Date` | error | null | null | null (Convert: «If the conversion cannot be performed, the result is null») |
| `convert @2014-01-01T10:00 to Date` | error | `@2014-01-01` | `@2014-01-01` | `@2014-01-01` |
| `convert '1:2' to Ratio`, `'1:2' as Ratio` | error, null | el traductor lo rechaza | el traductor lo rechaza | no hay conversión implícita ni explícita de String a Ratio |
| `ToRatio('\t1:2')`, `'1\n:2'` | unknown function | el Ratio | null, el Ratio | lo que dé `ToQuantity` |
| `ToRatio('1 \'mg:2')`, una comilla sin cerrar | unknown function | un Ratio | null | null |
| `ToRatio('1L:2')` | unknown function | error | null | null (el formato de `ToQuantity` no tiene Long) |
| `ConvertsToDate(FHIR.date { value: @2014-01-01 })` | false | error | false | false: el argumento es Any, sin conversión del modelo |

**Lo que se toma, lo que se mejora y lo que se descarta:**
- **`funcs.ToDate`** se toma en la rama DateTime: corta el texto antes de la `T`, que es `date from` en el offset del propio valor, como hacen las dos referencias. La rama String se mejora: hoy usa el lector de Date y rechaza un string con hora.
- **`ConvertsToRatio`** (`eval/evaluator.go`) se descarta. Lee los dos lados con `strconv.ParseFloat`, sin unidad, y no reconoce un Ratio, cuyo tipo en ejecución es `cqltypes.Ratio`. Pasa a ser «`ToRatio` no da null».
- **`ConvertsToDate`** se mejora: pasa a ser «`ToDate` no da null», con la misma lectura.
- **Java:**
  - su `ConvertsToRatio` devuelve el argumento y no un Boolean;
  - su `ConvertsToDate` de un DateTime da error;
  - falla en `':2'`.
  Las tres son defectos de la referencia. Se registran con la evidencia de esta tabla, sin reportarse al proyecto: la 5.4.0 es la última versión, y los reportes de Java no son de esta librería.

## Alcance

1. **`ToDate`** en el evaluador. Por sobrecarga:
   - **String:** lo que lee el lector de DateTime del motor, el de `ToDateTime` (decisiones A y B). La Date sale de la parte de fecha tal como está escrita, sin pasar el offset a otro: no se sitúa antes de cortar.
   - **DateTime:** `date from`.
   - **Date:** una Date de System con la misma fecha, como responden las dos referencias. Un `FHIR.date` llega como Date con su tipo FHIR, y sale como `System.Date`. La especificación no tiene esta sobrecarga: el traductor de Java pasa por DateTime.
   - **null:** null.
   - **Cualquier otro tipo:** error, como `ToQuantity` (plan 010): no hay sobrecarga, y el traductor de referencia lo rechaza.
2. **`ToRatio`** en `funcs`, junto a `QuantityFromString`:
   - parte el string en el único `:` que está fuera de comillas simples (decisión C);
   - lee cada lado con `QuantityFromString`, sin un lector nuevo, porque la especificación remite al formato de `ToQuantity`;
   - da null si no hay exactamente un separador fuera de comillas, si una comilla queda sin cerrar o si un lado no es una Quantity;
   - un argumento null da null, y uno de otro tipo, error.
3. **El literal Ratio** (decisión D). `ast.LiteralRatio` se evalúa como un `cqltypes.Ratio` de dos Quantity, y `numerator` y `denominator` se navegan. Comparte con `ToRatio` el separador, pero cada lado se lee como un literal Quantity, que admite una palabra de calendario sin comillas; un número sin unidad lleva la unidad `'1'` (review 1). Hoy el literal es un String: `is Ratio` da false y `.numerator` da null. La igualdad queda por componentes, como `cqltypes.Ratio.Equal`, que coincide con Java (`(1:2) = (2:4)` da false).
4. **`ConvertsToDate` y `ConvertsToRatio`** pasan a ser «la conversión no da null». Un Ratio convierte a Ratio, y un Date o un DateTime convierte a Date; cualquier otro tipo no convierte. Se conservan los tests de `eval/evaluator_test.go` (`"1:128"` da true; `"Inf:NaN"` y `"hello:world"`, false).
5. **`convert … to Date`** en `evalConvert`, junto al caso de Quantity del plan 010, y no en `convertToType`, que `cast` comparte. Un String que no convierte da null, como dice la especificación, en vez de un error; un DateTime da `date from`. Esto cambia un comportamiento existente (`convert 'abc' to Date` daba error) y tiene su propio test y su propia mutación. `cast` no cambia.
6. **La suite:** los ocho tests de G3f salen de `knownFailures`, y sus definiciones (D49 a D56) salen de `suitegaps.cql`.
7. **Un área nueva, `conversions`,** con las 265 sondas, comparadas exactamente contra las dos referencias. Cada divergencia se clasifica.

## Fuera del alcance (al backlog)

- **Equivalencia de Ratio:** `(1:2) ~ (2:4)` da true en las dos referencias, porque la especificación compara en cruz. `cqltypes.Ratio.Equivalent` compara por componentes, y con el literal como Ratio da false (defecto propio, clasificado en `conversions`).
- **La escala en `ToString` de una Quantity:** `ToString(1.0 'mg')` da `1 'mg'`, y la tabla 9-G da `(-)?#0.0# '<unit>'`. Un Ratio se escribe ahora `<quantity>:<quantity>`, cada lado como `ToString` de su Quantity (review 1), así que hereda esa pérdida. La ida y vuelta por `ToRatio` vale por igualdad de Quantity, no por el texto; el test lo dice.
- **El lector de `ToDateTime`:** acepta `'2014-01-01T'` y `'2014-01T12:30'`, y recorta `.1234` a `.123`. Las dos referencias dan null. Por la decisión B, `ToDate` hereda esa lectura; las divergencias se clasifican como de `ToDateTime`, en un frente propio.
- **Un literal Quantity con una unidad que no es de UCUM** (`1 'foo'`): se lee, y el traductor de referencia lo rechaza. El literal Ratio, que son sus dos literales Quantity, lo comparte (D264). Es un defecto del literal Quantity, anterior al 014.
- **`cast '2014-01-01' as Date`:** da una Date, y el traductor de referencia lo rechaza. Es de `cast`, no de la conversión.
- ~~**`convert '1:2' to Ratio`:** sigue en error.~~ Entró en la review 1: la tabla 9-E de la especificación da String a Ratio como explícita, y el error de Java era de `FHIR.Ratio`, por el nombre sin calificar bajo `using FHIR`.
- **El ELM importado:** el importador no conoce ningún nodo de conversión (backlog 4).
- **Las firmas en `sema`:** `fixed(Date)` y `fixed(Ratio)` no comprueban el argumento, así que `ToDate(5)` se rechaza en ejecución y no en `sema`. Es lo mismo que pasa hoy con las demás conversiones; queda en un frente de firmas.

## Decisiones para el usuario

- **A. `ToDate` de un string con hora.** La especificación y la suite (`ToDateIgnoresTime`) dicen que la hora se ignora: `ToDate('2014-01-01T12:05:05.955')` da `@2014-01-01`. Java y cql-execution dan null en las 23 sondas de un string con hora. Recomendado: la especificación, con una diferencia deliberada frente a las dos referencias.
- **B. Qué strings con hora cuentan como válidos.** La especificación pide «a valid date value» y no dice nada de la hora. Recomendado: los que acepta el lector de DateTime del motor, el de `ToDateTime`. Así `ConvertsToDate` y `ConvertsToDateTime` coinciden, y no se escribe un lector nuevo. Consecuencias:
  - `'2014-01-01T25:00:00'` y `'2014-02-30T10:00'` dan null;
  - `'2014-01-01T'` y `'2014-01-01T12:30:00.1234'` dan una Date, porque ese lector los acepta. Lo que el lector tenga de defecto se arregla en él, no aquí.
  La otra opción es tomar lo que haya antes de la `T`. Esa daría una Date a partir de un texto que no es una fecha ni una fecha con hora.
- **C. Un `:` dentro de una unidad entre comillas** (`'1 \'{a:b}\':2'`, una anotación UCUM válida). Recomendado: partir en el `:` que está fuera de las comillas. Es la gramática `<quantity>:<quantity>` y coincide con cql-execution. Java parte en cada `:` y da null.
- **D. El literal Ratio.** Hoy es un String. Recomendado: incluirlo, con el mismo lector que `ToRatio`; sin eso, `ToRatio('1:2') = 1:2` compara un Ratio con un String. La otra opción es dejarlo como defecto aparte y sacar del criterio de salida las comparaciones con literales.

## Decisiones tomadas (29-10-2026)

- **A:** como la especificación.
- **B:** lo que lee `ToDateTime`.
- **C:** se parte fuera de las comillas.
- **D:** el literal Ratio entra.

Las cuatro fueron las recomendadas.

## Diferencias deliberadas

En el área `conversions`, cada una nombrada una sola vez:

- **Decisiones A y B, frente a las dos referencias:** `ToDate`, `ConvertsToDate` y `convert … to Date` de un string con una hora válida. Son D6, D7, D8, D9, D14, D15, D16, D17, D18, D19, D20, D21, D22, D23, D81, D83, D244 y D245.
- **Decisión C, frente a Java:** un `:` dentro de una unidad entre comillas. Son D166, D167, D168 y D249.
- **Lo que `ToQuantity` lee en cada lado (plan 010, decisión A), frente a Java,** que no valida la unidad: D162, D163, D164, D186, D187, D188, D190, D191 y D192.
- **Las dos partes lo rechazan:** el traductor de referencia en la compilación, y este motor en la evaluación. Es una conversión sin sobrecarga: D258 y D259.

## Criterio de salida

- Los ocho tests de G3f pasan, y salen de `knownFailures` y de `suitegaps`.
- En el área `conversions`, cada divergencia está clasificada, y las deliberadas nombran este plan.
- Para todo String de las sondas, `ConvertsToDate(x)` es `ToDate(x) is not null` y `ConvertsToRatio(x)` es `ToRatio(x) is not null` (test metamórfico).
- `ToRatio(ToString(r)) = r` para cada Ratio de las sondas, por igualdad de valor. El texto no es el de la especificación (fuera del alcance).
- `make check` en verde, con las mutaciones nuevas detectadas.

## Tests

- **Escritos a mano, con la cita de la especificación,** en `engine_conversion_test.go`. El archivo no existe; se comprueba antes de crearlo. Los valores esperados salen de la especificación, no de las referencias, porque las dos contradicen la decisión A:
  - las ocho de la suite;
  - las de la tabla;
  - las del offset, evaluadas a un offset fijo distinto del del string (por ejemplo, -12:00 con `'2014-01-01T00:30:00+14:00'`, que da `@2014-01-01`);
  - las de ida y vuelta, y la equivalencia de `ConvertsTo` con su `To`;
  - el literal Ratio: `is Ratio`, `numerator`, `denominator` e igualdad;
  - `convert` a Date: un String válido, uno inválido (null) y un DateTime. `cast` queda como estaba.
- **El área `conversions`,** contra Java y cql-execution. Un script del scratchpad clasifica las divergencias, como en el 013.
- **Mutaciones** (`scripts/mutations/014-conversions.sh`):
  - `ToDate` de un String con el lector de Date;
  - `ToDate` que toma lo que hay antes de la `T`;
  - `ToDate` que sitúa el valor antes de cortar;
  - partir en cada `:`;
  - un lado sin validar (lo detecta `"Inf:NaN"`);
  - `ConvertsToRatio` con el lector viejo;
  - `ConvertsToDate` que no reconoce un DateTime;
  - el literal Ratio como String;
  - `convert` a Date que da error en vez de null;
  - `cast` que pasa a dar null;
  - una divergencia deliberada que el plan no nombra;
  - una definición de G3f que vuelve a `suitegaps`.

## Encontrado en los tests

- **`is FHIR.date` no distingue una Date de System de una FHIR:** el motor responde true también a `@2014-01-01 is FHIR.date`. Es el modelo de primitivas vigente, y no se decide aquí. Ningún test escrito en CQL puede distinguirlas. Por eso, en la review 1, el test pregunta en Go (`fptypes.IsFHIRPrimitive`), y la mutación «`ToDate(Date)` que devuelve el valor FHIR» vuelve a la lista.

- **Un tipo sin calificar bajo `using FHIR`:**
  - con el modelo FHIR en la librería, el traductor de Java lee `is Ratio` e `is Quantity` como `FHIR.Ratio` y `FHIR.Quantity`: `ToRatio('1:2') is Ratio` da false en Java, y cql-execution falla;
  - este motor los lee como los de System, y da true;
  - sin `using FHIR`, las dos referencias dan true (sondas de `r014/probe014.cql`).
  
  Es la resolución de nombres de tipo entre modelos, no la conversión. El área `conversions` califica `System.Ratio` y `System.Quantity`, y la diferencia va al backlog con esta evidencia (primera corrida del área, `r014/conv-div-run1.txt`).

- **Una comilla sin cerrar no necesita su propia comprobación:** deja un apóstrofo en uno de los lados, y `QuantityFromString` rechaza cualquiera que quede fuera de la unidad (sonda en el scratchpad, `r014/`). La comprobación no podía observarse, así que sale del código, y su mutación sale de la lista. Los casos quedan en los tests (`'1 \'mg:2'`).

## Implementación

- **`funcs/ratio_string.go`** (nuevo): `RatioFromString` y `ToRatio`.
- **`funcs/type_ops.go`:** `ToDate` lee un string con hora por el lector de `ToDateTime` y corta la fecha escrita. De un Date o un DateTime da una Date de System, y de otro tipo, error.
- **`eval/evaluator.go`:**
  - `todate` y `toratio`;
  - `ast.LiteralRatio` como Ratio;
  - `numerator` y `denominator` de un Ratio;
  - `ConvertsToDate` y `ConvertsToRatio` como «la conversión no da null»;
  - `convert … to Date` por `ToDate`.
- **La suite:** los ocho tests de G3f salen de `knownFailures` y de `suitegaps`.
- **El área `conversions`:** 263 definiciones, sin las dos de `ConvertsToRatio` de un Ratio, porque Java devuelve el Ratio y su arnés no lo codifica. Sus divergencias se clasifican con `r014/classify_conversions.py`.

## Mutaciones

`scripts/mutations/014-conversions.sh` tenía 15 mutaciones antes del review, y todas se detectaban (corrida del 29-10-2026, `r014/mut1.txt`). La review 1 agregó cinco: el literal por la gramática del string, la unidad por defecto, `ToString` de un Ratio, `convert … to Ratio` y `ToDate` de un `FHIR.date` que conserva su tipo. Las 20 se detectan (`r014/mut2.txt`). Cubre:
- **ToDate:** la hora, la hora inválida, el offset, otro tipo y `ConvertsToDate` de un DateTime;
- **convert a Date;**
- **ToRatio:**
  - partir en cada `:`;
  - un lado sin leer;
  - un Ratio;
  - `ConvertsToRatio` que no sigue a `ToRatio`;
- **el literal Ratio;**
- **el área y su clasificación:** la divergencia deliberada sin nombrar, el manifiesto, las referencias y G3f de vuelta en `suitegaps`.

`cast` no tiene mutación propia: el 014 no lo toca, y lo cubre `TestConvertToDateIsNullWhereItCannot`.

## Diferencial (29-10-2026)

La diferencial masiva contra main (corpus de la semilla 8008, `r014/diff.sh`) da las nueve áreas idénticas. Ese corpus no llama a `ToDate`, `ToRatio`, `ConvertsToDate`, `ConvertsToRatio`, `convert … to Date` ni a un literal Ratio (búsqueda en el corpus). Así que la diferencial solo muestra que no hay cambios colaterales. Lo que cambió lo cubren el área `conversions`, contra las dos referencias, y `suitegaps`. Ninguna otra área de snapshots movió una respuesta.

## Ronda adversarial (Fable y Codex, 29-10-2026)

| # | Hallazgo | Quién | Severidad | Resolución |
|---|---|---|---|---|
| 1 | El literal Ratio es un String en ejecución: `evalLiteral` no tiene `LiteralRatio`, y `numerator` da null. Los criterios con literales comparaban un Ratio con un String | los dos | alta | Aceptado, verificado con una sonda. Entra en el alcance (3) como decisión D |
| 2 | `convert … to` no pasa por `funcs.Convert`, que no tiene llamadores fuera de los tests; va por `convertToType`, que `cast` comparte | los dos | alta | Aceptado, verificado: `convert '2014/01' to Date` da error. El alcance 5 va en `evalConvert`; Ratio sale, porque el traductor lo rechaza (sonda D12) |
| 3 | La exclusión de FHIR supone una conversión que no ocurre: `ConvertsToDate(FHIR.date {…})` da false | Codex | alta | Rechazado como alta. Medido: cql-execution también da false, y Java da error. El argumento es Any, así que no hay conversión del modelo. Fable agregó que un `FHIR.date` recuperado llega como Date con tipo FHIR; eso sí entra (alcance 1, Date) |
| 4 | La decisión B contradice su lector: `ToDateTime` acepta `'2014-01-01T'` y recorta `.1234` | los dos | media | Aceptado. La tabla y B dicen ahora qué da cada caso; los defectos del lector quedan en él (fuera del alcance) |
| 5 | Un tipo sin sobrecarga no está decidido: `funcs.ToDate` da null en silencio, y `sema` no comprueba firmas | los dos | media | Aceptado: error en ejecución, como `ToQuantity`. Las firmas en `sema` quedan fuera, igual que para las demás conversiones |
| 6 | `todatetime` sitúa el resultado; las sondas con +14:00 solo detectan «situar y cortar» si el offset de la evaluación es otro | Fable | media | Aceptado: tests a offset fijo y una mutación |
| 7 | Un Ratio necesita una política de validez: `NewRatio(nil, nil)` convierte. Y 1:0 y 0:0 | Codex | media | Aceptado en parte. Los Ratio que crea el motor siempre tienen dos Quantity; 1:0 y 0:0 son válidos en las dos referencias y quedan así, con test. Un `cqltypes.Ratio` armado a mano por quien usa la librería no se valida aquí |
| 8 | La decisión C necesita valores propios: comillas sin cerrar, comillas dobladas, tabulaciones, saltos de línea, separadores repetidos | los dos | media/baja | Aceptado: 49 sondas nuevas (`r014/probe014b.cql`) y sus valores en los tests escritos a mano |
| 9 | Coincidir con las referencias no sirve de oráculo aquí, porque contradicen la decisión A, y cql-execution redondea decimales (D199) | Codex | media | Aceptado: los valores esperados salen de la especificación, y las divergencias se clasifican después |
| 10 | `ToString(1.0 'mg')` pierde la escala, así que la ida y vuelta solo prueba la igualdad de valor | Fable | baja | Aceptado: el criterio lo dice, y el formato queda fuera del alcance |
| 11 | Los tests existentes de `ConvertsToDate` y `ConvertsToRatio` | Fable | baja | Aceptado: se conservan, y `"Inf:NaN"` detecta la mutación «un lado sin validar» |

## Dependencias

Ninguna nueva.

## Review, ronda 1 (Fable y Codex, 29-10-2026)

| # | Hallazgo | Quién | Severidad | Resolución |
|---|---|---|---|---|
| 1 | El literal Ratio se leía con la gramática de `ToRatio`, más estrecha que la del literal: `1 day:2 days`, `1 week:2 weeks` y `1 'foo':2 'foo'` fallaban en la evaluación, y en main daban un String sin error | los dos | media | Confirmado con un test que fallaba. Cada lado se lee como un literal Quantity (`fptypes.NewQuantity`), con el separador de `RatioSides`, que comparte con `ToRatio`. Un número sin unidad lleva `'1'`. Tests y dos mutaciones |
| 2 | `ToString` de un Ratio salía de `Ratio.String`, que no pone comillas a una unidad de calendario, así que `ToRatio(ToString(1 'day':2 'days'))` daba null; la especificación pide «round-trippable» | Codex | media | Confirmado con un test que fallaba. `funcs.ToString` escribe cada lado como su Quantity, separados por `:` sin espacios (tabla 9-G). Test y mutación |
| 3 | D226 a D228: la tabla 9-E sí tiene String a Ratio, y el error de Java era de `FHIR.Ratio`, por el nombre sin calificar | Codex | media | Confirmado contra la especificación y con la sonda calificada. `convert … to Ratio` es `ToRatio`, como `convert … to Date` es `ToDate`. Las sondas del área quedan calificadas como `System.Ratio`. Test y mutación |
| 4 | `TestToDateOfAFHIRDateIsASystemDate` no detectaba un `ToDate` que deja el tipo FHIR: el `is` de CQL no los distingue, pero `fptypes.IsFHIRPrimitive` sí | los dos | baja | Aceptado. El test pregunta en Go, y la mutación vuelve a la lista, ahora detectada |
| 5 | El test de `cast` probaba solo una fecha válida | Codex | baja | Aceptado: `cast 'abc' as Date` y `cast '2014-01-01T10:00' as Date` siguen en error, con test |
| 6 | Tres definiciones duplicadas en el área (D10 y D236, D11 y D237, D81 y D229) | Fable | baja | Aceptado: salen D229, D236 y D237 |
| 7 | `ConvertsToDate` convertía un error en false, y `ConvertsToRatio` lo propagaba | Fable | baja | Aceptado: los dos lo propagan |

## Review, ronda 2 (Fable y Codex, 29-10-2026)

| # | Hallazgo | Quién | Severidad | Resolución |
|---|---|---|---|---|
| 1 | `convert … to String` de un Ratio no pasaba por `ToString`: `convert (1 'day':2 'days') to String` daba `1 day : 2 days`, y `ToRatio` no lo leía de vuelta | Codex (media), Fable (baja) | media | Confirmado con un test que fallaba. `evalConvert` escribe un Ratio a String con `funcs.ToString`. Test y mutación. La Quantity tiene el mismo problema desde antes del 014 (`convert (1 'day') to String` da `1 day`); va al backlog, sin ampliar el alcance |
| 2 | `(1:2) ~ (2:4)` da false | Fable | baja | Ya estaba fuera del alcance y en el backlog (la equivalencia de Ratio) |

Fable verificó, sin hallazgos:
- **El literal:** `GetText` quita los espacios, y `NewQuantity` lee las dos formas.
- **La unidad por defecto** coincide con las dos referencias (D265).
- **Los lados nil:** `NewRatio` solo los recibe no nil.
- **`convert` de otro tipo a Ratio:** da el mismo error que en main.
- **Las filas reclasificadas:** D211, D212, D219, D220, D264 y D266.

## Review, ronda 3 (Fable y Codex, 29-10-2026): converge

Ninguno de los dos deja hallazgos de severidad media o mayor, y las 21 mutaciones se detectan.

| # | Hallazgo | Quién | Severidad | Resolución |
|---|---|---|---|---|
| 1 | La corrección de la ronda 2 estaba en `convertToType`, que comparte `cast`, así que también cambiaba `cast (1 'day':2 'days') as String`, contra lo que dice el plan | Fable | baja | Aceptado: la comprobación pasa a `evalConvert`, junto a las de Date y Ratio, y `cast` queda como en main. El test no cambia, y la mutación se reubica |
