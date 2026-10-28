# 010: ToQuantity y la lectura de números en un string

Sale del punto 1c23 del backlog, que juntaba cinco defectos que encontró el 009. Este incremento toma uno solo: la conversión a Quantity, con `ToDecimal('.5')`, porque las dos comparten el lector del número. Los otros cuatro vuelven al backlog, cada uno por su cuenta (ver «Fuera del alcance»).

## Por qué

`funcs.ToQuantity` (`funcs/type_ops.go:143`) pasa cualquier valor a string y se lo da a `fptypes.NewQuantity`, que es el lector de literales de FHIRPath y no el de CQL. Por eso:

- **Ratio:** `ToQuantity(5:2)` da `5 ':2'`. Java y JS dan `2.5 '1'` (fuera del alcance: ver abajo).
- **Integer, Long y Decimal:** `ToQuantity(5)` da una Quantity sin unidad (`.unit` null). La especificación dice `'1'`, y Java y JS también.
- **Un string mal formado** (`'abc'`, `''`, `'1e3 \'mg\''`, `'−5 \'mg\''`) da error, y la especificación dice null.
- **Un número fuera de rango** (`'100000000000000000000 \'mg\''`) se acepta, y Java da null.
- **La unidad:** se lee mal entre comillas (`'5 \'mg'` da la unidad `'mg`; `'5 \'\''` da `''`).
- **`convert … to Quantity`** da error siempre, también desde `'5 \'mg\''` y desde `5`. `convertToType` (`eval/evaluator.go:8678`) solo deja pasar una Quantity.

Además, `ToDecimal('.5')` da 0.5, y la especificación pide un dígito antes del punto (backlog 1c23; D12 a D20 de `decimalstrings`).

## Revisión previa (sondas del 27-10-2026, en el scratchpad `q010/`)

**La especificación** (CQL v2.0.0, Appendix B, Type Operators, ToQuantity):

> For the String overload, the operator accepts strings using the following format: `(+|-)?#0(.0#)?('<unit>')?` […] all optionally followed by a unit designator as a string literal specifying a valid, case-sensitive UCUM unit of measure or calendar duration keyword, singular or plural. Spaces are allowed between the quantity value and the unit designator. […] Note that the decimal value of the quantity returned by this operator must be a valid value in the range representable for Decimal values in CQL. If the input string is not formatted correctly, or cannot be interpreted as a valid Quantity value, the result is null. For the Integer, Long, and Decimal overloads, the operator returns a quantity with the value of the argument and a unit of '1' (the default unit). For the Ratio overload, the operator is equivalent to dividing the numerator of the ratio by the denominator.

ToDecimal pide el mismo formato para el número: `(+|-)?#0(.0#)?`.

**`gofhir/server`** no llama a ToQuantity: no hay nada que tomar ni que descartar.

**fhirpath v1.16.6:** `types.NewQuantity` implementa el literal de FHIRPath (`^([+-]?\d+\.?\d*)\s*(?:'([^']+)'|(\S+))?$`). No es un defecto de fhirpath: es otra gramática. El defecto es del motor, que la usa para CQL.

**Sondas** (152 definiciones: `q.cql`, de 32 strings por tres formas, y `q2.cql`, de 56 expresiones), contra Java 5.4.0 y JS 2.2.0. Tablas en `q010/table.txt` y `q010/table2.txt`. Lo que se mide de Java:

- Lee el número como lo lee su ToDecimal, con exponente (`'1e3 \'mg\''` da `1000 'mg'`) y el rango de CQL (`'1e10001'` y `'100000000000000000000'` dan null), pero **acepta `'.5 \'mg\''`** (0.5), aunque su `ToDecimal('.5')` da null y su `ConvertsToDecimal('.5')` da true.
- Es laxo con la unidad: le quita las comillas y los espacios a todo lo que sigue al número. `'5 mg'` da `mg`; `'5 \'mg'` da `mg`; `'5 \'mg\'x'` da `mgx`; `'5 5'` da `5`; `'5 \'m g\''` da `mg`; `'5 \"mg\"'` da `"mg"`.
- No valida la unidad: `'5 \'foo\''` da `5 'foo'`. JS da error.
- Acepta espacios y tabuladores antes, entre y después; un salto de línea da null. Sin espacio entre el número y la unidad (`'5\'mg\''`) da null.
- Sin unidad, la unidad es `'1'` (`'5'`, `5`, `5.5`, `5L`).
- Ratio: divide (`5 'mg':2 'mg'` da `2.50 '1'`; `5:0` da null).
- `convert … to Quantity` da lo mismo que ToQuantity.

## Alcance (ajustado por la ronda adversarial)

1. **Un lector de Quantity propio para el string**, con la gramática de la especificación:
   - al principio y al final se descarta todo carácter hasta U+0020, como lo hace `String.trim()` de Java, con el salto de línea (review 1, C2);
   - el número sigue la gramática y el rango de ToDecimal, pero **no su escala**: la Quantity conserva las cifras escritas, como el literal y como Java (ADR 002, «Quantity: no se recorta»). Se separa de `DecimalFromString` la parte que lee y valida el rango de la que lleva a 8 decimales;
   - el punto sin decimales (`5.`) y el exponente (`1e3`) se aceptan, como en ToDecimal desde la decisión A del 009; no los admite el formato de la especificación, y el plan lo dice así;
   - entre el número y la unidad, cero o más espacios o tabuladores (la especificación los permite y no los exige);
   - la unidad es un literal entre comillas simples, sin comillas dentro; todo lo que sigue a la comilla de cierre, o una comilla sin cerrar, da null (ejemplo de la especificación: `ToQuantity('444 \'cm')` da null);
   - todo lo demás da null.
2. **Sin unidad, la unidad es `'1'`.** Una unidad vacía (`'5 \'\''`) no es UCUM y da null (decisión A).
3. **Integer y Decimal** dan su valor con la unidad `'1'`. El motor no tiene Long (`5L is Long` da false), así que `5L` va por Integer.
4. **`convert … to Quantity`** desde String, Integer y Decimal da lo mismo que ToQuantity (`convertToType`).
5. **`ConvertsToQuantity`** da true exactamente cuando ToQuantity da un valor.
6. **`ToDecimal('.5')`, `'+.5'` y `'-.5'` dan null:** se quita la alternativa `\.[0-9]+` de `splitDecimal`. Va en este incremento porque el número de ToQuantity lo lee el mismo lector: separarlos dejaría a uno aceptando lo que el otro rechaza.

Las palabras clave de calendario (si la decisión C las acepta sin comillas) salen de `types/calendar_sum.go`, y un test verifica que el conjunto es el de `dateTimePrecision` y `pluralDateTimePrecision` de `grammar/cql.g4`.

`funcs.ToQuantity` solo la llaman `eval/evaluator.go:4835` y `:4968` y `funcs.Convert`. `fptypes.NewQuantity` sigue para los literales (`eval/evaluator.go:833`), `quantity_converter.go` y `conformance/output_parser.go`, que no cambian.

## Diferencias deliberadas con Java (cada una con su definición en el corpus)

- **`'.5 \'mg\''` da null** (Java 0.5): el formato pide `#0`, y es lo que hace el ToDecimal de Java.
- **`ConvertsToDecimal('.5')` da false** (Java true): sigue a ToDecimal. Como el ToDecimal de Java da null, se clasifica como defecto de la referencia, no como diferencia deliberada.
- **`'5\'mg\''`, sin espacio, da `5 'mg'`** (Java null): la especificación permite el espacio, no lo exige.
- **Una unidad sin comillas** (`'5 mg'`, `'5 MG'`, `'5 5'`, `'5 \"mg\"'`) **da null** (Java la acepta).
- **Lo que sigue a la unidad, o una comilla sin cerrar o escapada** (`'5 \'mg\'x'`, `'5 \'mg'`, `'5 \'mg\' \'g\''`, `'5 \'m\\\'g\''`) **da null.** Java une los pedazos.
- **Una unidad que no es UCUM ni palabra de calendario** (`'5 \'foo\''`, `'5 \'m g\''`, `'5 \'\''`) **da null** (decisión A; Java la acepta).
- **Una palabra de calendario sin comillas** (`'5 days'`) **da null** (decisión C; Java la acepta).
- **`ConvertsToQuantity(5L)`** da true; Java da un error de argumento, y es un defecto de la referencia (su ToQuantity(5L) sí funciona).

## Decisiones del usuario (27-10-2026)

- **A. La unidad se valida contra UCUM** (la especificación: «a valid, case-sensitive UCUM unit of measure or calendar duration keyword», y null si no se puede interpretar). Una unidad entre comillas es válida si es una palabra de calendario de la gramática o si `ucum.Service.Validate` la acepta; si no, el resultado es null. Así, `'5 \'foo\''`, `'5 \'m g\''` y `'5 \'\''` dan null (Java da una Quantity en los tres: diferencias deliberadas). La validación es solo de este lector: los literales (`5 'foo'`) no cambian, y su validación queda en el backlog.
- **B. El salto de línea, como Java:** se descarta en los extremos (el conjunto de `trim()`), y entre el número y la unidad solo valen espacio y tabulador.
- **C. Una palabra de calendario sin comillas (`'5 days'`) da null,** como pide la especificación. Es una diferencia deliberada con Java y cambia lo que el motor da hoy. Entre comillas (`'5 \'days\''`) es válida.
- **Plan aprobado** con los cambios de la ronda adversarial.

## Fuera del alcance (al backlog, cada uno aparte)

- **El ELM no importa ninguna conversión:** `ToQuantity`, `ConvertsToQuantity`, `ToDecimal` y los demás `To*` caen en el `default` de `elm/importer.go:443` y se evalúan como null, sin error (Codex, `.review010/main.go`). Va al punto 4 del backlog (nodos de ELM que faltan).
- **El literal Ratio es un String en tiempo de ejecución** (`(5:2) is String` da true; `eval/evaluator.go:834`). `ToQuantity(Ratio)` necesita antes que el Ratio exista como valor: es su propio incremento. Al volver, Java convierte las unidades antes de dividir (`5 'mg':2 'g'` da `0.00250000 '1'`), y la escala del cociente se clasifica según el ADR 002.
- **Ceros con escala** (`ToDecimal('0.000000000')`): depende de 1c18.
- **Quantity FHIR que desborda:** espera a gofhir/fhirpath#235 (y #232).
- **`cast` que convierte** (D449 de `types` y los ocho de `fhir`).
- **Validar la unidad de un literal Quantity** (`5 'foo'`): el traductor de Java lo rechaza; el motor no.
- **`5 '' = 5 '1'`**: Java da true y el motor null; es de la comparación, no del lector.
- Las conversiones a Integer, Boolean, Date y Time de `convertToType`: no se tocan.

## Criterio de salida

- Las definiciones del área nueva `quantitystrings` y las de `decimalstrings` coinciden con Java, salvo las diferencias deliberadas de arriba, cada una nombrada en este plan.
- `make check` en verde, y las mutaciones del 010 todas atrapadas.
- `scripts/references.sh check`: dos corridas idénticas y 0 archivos distintos del repo.
- La diferencial masiva (semilla 8008) contra Java, antes y después del review: ninguna definición fuera de ToQuantity, ConvertsToQuantity, convert y `'.5'` cambia de respuesta.

## Tests

1. **Área de snapshot `quantitystrings`**, escrita a mano como `decimalstrings` (el generador está en el scratchpad, `gen010/gen.py`): 894 definiciones, 288 strings por tres formas (ToQuantity, ConvertsToQuantity, convert), más Integer, Long, Decimal y null. Cubre:
   - el número: signos, ceros a la izquierda, punto sin decimales, exponentes, bordes del rango, cifras más allá de la octava (`1.000000005`, `0.000000001`, `-1.00000001`), `'−'`, `'.-5'`;
   - el espacio en blanco: ninguno, varios, tabulador, salto de línea en los extremos y en el medio, NBSP;
   - la unidad: entre comillas, vacía, con espacio, sin cerrar, escapada, con resto, sin comillas; las 16 palabras de calendario con y sin comillas, y en mayúsculas;
   - los ejemplos de la especificación (`ToQuantity(0.1)`, `13`, `13L`, `'-0.1 \'mg\''`, `'444 \'cm'`).

   Los valores esperados salen de Java, y cada divergencia se clasifica (deliberada, defecto de la referencia o defecto propio).
2. **`decimalstrings`**: D12 a D20 pasan a coincidir con Java.
3. **Tests a mano en `funcs`**, con la especificación citada: el lector de la unidad, el espacio en blanco y la escala.
4. **Propiedades:** `ToQuantity('<n> \'<u>\'') = <n> '<u>'` sobre números y unidades generados, con las cifras exactas, contra un oráculo escrito aparte del lector (`expectedQuantity`). La ida y vuelta por ToString no va: `ToString(5.50 'mg')` escribe `5.5 'mg'` (review 2, F2). Fuzzing (`FuzzQuantityFromString`): sin pánico, y lo que acepta cumple la gramática de arriba, escrita aparte del lector. Se mide el costo con un string de diez millones de bytes, como en el 009.
5. **Metamórficos:** espacio de más en los extremos da lo mismo; `ToQuantity(x)` y `convert x to Quantity` coinciden en todo el corpus.

## Mutaciones (`scripts/mutations/010-quantity.sh`, 27-10-2026)

Son 31 (29 hasta el review, una de la ronda 1 y una de la ronda 2) y todas se atrapan; el arnés da como sobrevivida una mutación que no compila. Cada una nombra el comando que tiene que fallar y el texto de la falla:

- **El número:** `.5` leído otra vez; sin exponente; la Quantity llevada a 8 decimales; sin el rango; sin la precisión (la atrapa el test de recursos); un cero sin límite de decimales.
- **El espacio en blanco:** el salto de línea sin recortar; un salto de línea entre el número y la unidad; el espacio exigido antes de la unidad.
- **La unidad:** sin la unidad `'1'`; una unidad sin comillas aceptada; lo que sigue a la comilla de cierre ignorado; sin validar UCUM; una palabra de calendario de menos (la atrapa el test contra la gramática); una palabra leída sin distinguir mayúsculas.
- **Las sobrecargas y las formas:**
  - Integer y Decimal sin la unidad `'1'`;
  - `convert` que vuelve a dar error;
  - `cast` que convierte a Quantity, que mueve las respuestas de `types`: el cast queda fuera;
  - ConvertsToQuantity true para un null, `convert` distinto de ToQuantity y el espacio de los extremos leído aparte: los tres los atrapa `TestTheQuantityFormsAgree`.
- **El oráculo escrito aparte:** un oráculo que da null a todo y uno que no lee la unidad.
- **El área:**
  - una divergencia deliberada borrada de su archivo;
  - una divergencia que el plan no nombra (la segunda de `'.5'`, sin unidad);
  - un defecto propio que espera la respuesta del motor (D851);
  - el área fuera del manifiesto, o con otra cantidad de definiciones.

**Al correrlas:**

- **Tres fallas del primer intento:**
  - «un cero sin límite de decimales» sobrevivió, porque ningún test tenía un cero de más de 28 decimales. Se agregaron uno al test de propiedades y otro, de diez millones de decimales, al de recursos;
  - el texto esperado de «una palabra de calendario de menos» era otro;
  - «ConvertsToQuantity true para un null» aparecía cinco veces en el código.
- **Las del 009 cuyo texto cambió:**
  - «algo después del exponente» ahora apunta a `ok && end == len(s)`;
  - «un defecto propio que espera la respuesta del motor» apuntaba a D12, que ya no diverge; ahora es un defecto de la referencia que espera la respuesta de la referencia (D100);
  - «un signo después del punto» sobrevivió: al exigir un dígito antes del punto, ningún corpus tenía un `'5.-5'`. Se agregaron al final de `quantitystrings`, para no mover ningún número, `'5.-5'`, `'5.+5'`, `'5.e3'` y `'5.5e-3'` (D876 a D893, en las que Java coincide con el motor). Ahora la mutación termina en un pánico (el signo entre los dígitos deja el coeficiente en nil), y eso es lo que se espera.

  Las 23 del 009 se atrapan.

**La salida estándar del arnés JS:** ucum-lhc escribe con `console.log` las unidades que no lee, y eso caía antes del JSON. `js/evaluate.js` lo manda a la salida de error. Lo verifica `scripts/references.sh check`, que parsea cada `.js.json`.

## Diferencial masiva (semilla 8008, 27-10-2026)

Los corpus de las nueve áreas de la semilla 8008 (`scratchpad/d008`: arithmetic, codes, codesets, fhir, fhirmembership, integer, membership, scope y types) se corrieron contra Java con el comparador compilado desde main y desde la rama. Los informes son idénticos byte a byte en las nueve. Ninguno de esos corpus usa ToQuantity, así que lo que la diferencial muestra es que nada fuera de este incremento cambió de respuesta. Lo que sí cambió está en `quantitystrings` y en D12 a D20 de `decimalstrings`; fuera de esas dos áreas, ningún snapshot se movió. Se repitió después de la ronda 3, con el comparador de la rama compilado otra vez: idénticos en las nueve.

## Clasificación de `quantitystrings` (894 definiciones, 27-10-2026)

Script: `scratchpad/gen010/classify.py`. Contra Java quedan 266 divergencias (en main eran 567 de las 876 primeras): 265 deliberadas, en los grupos de abajo, y un defecto propio, D851. `ToQuantity(1.000000005)` da error porque el literal Decimal de nueve decimales falla («too many fractional digits»), y ya fallaba así en main: no es de este incremento y va al backlog. Contra JS: las 44 en que JS coincide con Java van en la misma clase que la de Java; las demás (444) son defectos de la referencia. cql-execution lee el número como un float que se detiene en el exponente, descarta una unidad sin comillas y da error con una que no lee.

En `decimalstrings`, D12 a D20 ahora coinciden con Java en ToDecimal y convert. `ConvertsToDecimal` de Java da true en D13, D16 y D19, aunque su propio ToDecimal da null: es un defecto de la referencia. Las nueve de JS también son defectos de la referencia.

| Grupo | Definiciones |
|---|---|
| Sin espacio antes de la unidad (`'5\'mg\''`): la especificación lo permite (93) | D6, D7, D8, D15, D16, D17, D24, D25, D26, D33, D34, D35, D42, D43, D44, D51, D52, D53, D60, D61, D62, D69, D70, D71, D78, D79, D80, D87, D88, D89, D96, D97, D98, D105, D106, D107, D114, D115, D116, D123, D124, D125, D132, D133, D134, D141, D142, D143, D150, D151, D152, D159, D160, D161, D168, D169, D170, D177, D178, D179, D186, D187, D188, D195, D196, D197, D204, D205, D206, D213, D214, D215, D222, D223, D224, D231, D232, D233, D240, D241, D242, D249, D250, D251, D258, D259, D260, D276, D277, D278, D321, D322, D323 |
| Sin dígito antes del punto (`'.5'`): el formato pide `#0` (18) | D333, D334, D335, D336, D337, D338, D342, D343, D344, D345, D346, D347, D351, D352, D353, D354, D355, D356 |
| Un dígito que no es ASCII (`'٥'`): decisión A del 009 (6) | D450, D451, D452, D453, D454, D455 |
| Más allá de la precisión de 28 (`'1e-10001'`) (6) | D306, D307, D308, D309, D310, D311 |
| Una unidad que no es UCUM ni palabra de calendario (decisión A) (33) | D513, D514, D515, D522, D523, D524, D555, D556, D557, D579, D580, D581, D582, D583, D584, D585, D586, D587, D588, D589, D590, D591, D592, D593, D594, D595, D596, D597, D598, D599, D735, D736, D737 |
| Una palabra de calendario sin comillas (decisión C) (48) | D648, D649, D650, D651, D652, D653, D654, D655, D656, D657, D658, D659, D660, D661, D662, D663, D664, D665, D666, D667, D668, D669, D670, D671, D672, D673, D674, D675, D676, D677, D678, D679, D680, D681, D682, D683, D684, D685, D686, D687, D688, D689, D690, D691, D692, D693, D694, D695 |
| Una unidad que no es un literal: sin comillas, sin cerrar o con algo después (60) | D696, D697, D698, D699, D700, D701, D702, D703, D704, D705, D706, D707, D708, D709, D710, D711, D712, D713, D714, D715, D716, D717, D718, D719, D720, D721, D722, D723, D724, D725, D726, D727, D728, D729, D730, D731, D732, D733, D734, D738, D739, D740, D759, D760, D761, D762, D763, D764, D765, D766, D767, D768, D769, D770, D771, D772, D773, D825, D826, D827 |
| El literal Quantity conserva las cifras más allá de la escala (ADR 002), y Java no (1) | D868 |

## Review, ronda 1 (Fable y Codex, 27-10-2026)

Ninguno dejó hallazgos de severidad alta. Fable revisó sin encontrar nada de severidad media o mayor: `go test ./...`, `-race`, 4,27 millones de ejecuciones del fuzz, las 29 mutaciones y un cruce de la clasificación con las respuestas de Java y JS. Cada hallazgo se verificó ejecutando:

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| C1 | El oráculo del fuzz rechazaba todo número con exponente mayor que 100 aunque los decimales escritos lo compensen (`'0.' + 101 ceros + '1e102'` es 1), y a un cero le perdía la escala | media | Confirmado con una sonda. El oráculo calcula ahora la potencia de la última cifra con el exponente y los decimales juntos |
| C2, F1 | El código recortaba solo `[ \r\n\t]`, y la decisión B dice «como Java», que recorta todo carácter hasta U+0020 (`String.trim()`; D780: `'5 \'mg\'\u0000'` da `5 'mg'`) | baja | Confirmado. Se recorta hasta U+0020; D780 a D782 coinciden ahora con Java, y quedan 266 divergencias |
| F2 | `convert true to Quantity` daba null, y en main daba error; la especificación no tiene esa sobrecarga | baja | Confirmado. Solo String, Integer, Decimal y Quantity van a ToQuantity; lo demás da el error de antes. Test `TestConvertToQuantityKeepsItsOverloads` y su mutación |
| F3 | La variable local `units` del test tapaba la del paquete | baja | La del paquete pasa a llamarse `ucumService` |
| F4 | La mutación del 009 «un signo después del punto» se atrapaba por un pánico | baja | `decimalAsWritten` ya no deja el coeficiente en nil, y la mutación espera que se muevan las respuestas de `quantitystrings` |

## Review, ronda 2 (Fable y Codex, 27-10-2026)

Fable confirmó las cinco correcciones de la ronda 1 y no dejó nada de severidad media o mayor; leyó `ToQuantityEvaluator.kt` de Java 5.4.0, que recorta igual que `trimmed`. Codex dejó uno de severidad media. Cada hallazgo se verificó ejecutando:

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| C1 | La unidad de la Quantity era un pedazo del string leído: guardar ocho Quantities leídas de strings de diez millones de bytes retenía 80 MB | media | Confirmado con un test que falla sin la corrección. La unidad se copia (`strings.Clone`); test `TestAQuantityDoesNotKeepTheStringItWasReadFrom` y su mutación. La caché de gofhir/ucum retiene lo mismo por su cuenta (la clave es el string de quien llama): es defecto de esa librería, reportado como [gofhir/ucum#61](https://github.com/gofhir/ucum/issues/61) |
| F1 | `ToQuantity(true)` daba null, y en main y en Java da error; la corrección F2 de la ronda 1 cubría solo `convert` | baja | Confirmado. ToQuantity da el error de antes para todo tipo sin sobrecarga, y `convert` lo usa sin guarda propia; `ConvertsToQuantity` de esos tipos sigue en false. Tests `TestConvertToQuantityKeepsItsOverloads` y `TestConvertsToQuantityOfATypeWithNoOverloadIsFalse` |
| F2 | El plan prometía la propiedad `ToQuantity(ToString(q)) = q`, que no se escribió | baja | Confirmado, y la propiedad es falsa: `ToString(5.50 'mg')` escribe `5.5 'mg'`. Sale del plan; la cubre el oráculo de cifras exactas |
| F3 | El README decía 29 mutaciones | baja | Al día |
| F4 | `ConvertsToQuantity` da false si UCUM no carga | info | Sin cambio: `ucum.New()` carga datos embebidos y no falla en un binario sano |

## Review, ronda 3 (Fable y Codex, 27-10-2026): converge

Ninguno dejó hallazgos de severidad media o mayor. Codex no dejó ninguno: `go test -race ./...`, las 31 mutaciones del 010 y las 23 del 009, 255 882 ejecuciones del fuzz y una sonda de 66 casos (Ratio, valores del modelo FHIR, elementos choice, listas, null). Fable corrió la suite, el lint v2.15.0 y las 31 mutaciones, y una diferencial de 90 expresiones entre main y la rama sin regresiones:

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| F1 | `ToQuantity(O.value)` sobre un FHIR.Quantity sin cast da «cannot convert Quantity to Quantity», porque el tipo se escribe sin el espacio de nombres | baja | Anterior (main daba «invalid quantity format») y fuera del alcance: Java convierte por FHIRHelpers, que aquí no se aplica. Al backlog |
| F2 | `ToQuantity(O.value.unit)` da una lista de un null | info | Sin verificar contra Java. Al backlog junto con F1 |

## Ronda adversarial (Codex y Fable, 27-10-2026)

| # | Hallazgo | Resolución |
|---|---|---|
| C1 | El ELM no importa `ToQuantity` ni `ConvertsToQuantity` (null sin error) | Verificado (pasa con todos los `To*`). Rechazado para el 010: es anterior y es del punto 4 del backlog |
| C2, F1 | La escala de ToDecimal recortaría la Quantity (`1.000000005` → `1.00000000`) | Aceptado: se comparten gramática y rango, no la escala |
| C3, F4, F10 | El formato no admite `5.` ni exponente, y permite cero espacios | Aceptado cero espacios. `5.` y exponente siguen la decisión A del 009, dicho de forma explícita |
| C4 | No validar UCUM deja el contrato incompleto | Aceptado: decisión A del usuario, validar ahora |
| C5, F2, F3 | Ratio: la división no da la escala de Java y Java convierte las unidades; además el literal Ratio es un String | Verificado (`(5:2) is String`). Aceptado: Ratio sale del 010, a su propio incremento |
| C6, F11 | El oráculo del fuzzing era tautológico | Aceptado: gramática escrita aparte y propiedades de ida y vuelta |
| C7 | La lista de calendario está escrita a mano | Aceptado: test contra `grammar/cql.g4` |
| F5 | Rechazar `'5 mg'` y aceptar `'5 days'` no es coherente con la especificación | Aceptado: decisión C del usuario |
| F6 | Java descarta el salto de línea en los extremos | Aceptado: decisión B reformulada |
| F7 | La unidad `''` | Aceptado: da null por la decisión A; la comparación `5 '' = 5 '1'` va al backlog |
| F8 | `.5` debería ir aparte | Rechazado: comparte el lector con ToQuantity |
| F9 | Long | Aceptado: va por Integer; `ConvertsToQuantity(5L)` de Java es defecto de la referencia |
| F12 | Los llamadores son seguros; el ELM queda cubierto | Aceptado lo primero. Lo segundo es falso: C1 lo midió |

## Dependencias

- **`github.com/gofhir/ucum/v4` v4.3.1, directa** (decisión A). Ya estaba en el grafo, indirecta por fhirpath en v4.3.0, así que no entra código nuevo al binario. Es la librería UCUM de la plataforma, y la que usa fhirpath. v4.3.1 es la última; su único cambio es `CanonicalKey`, que fhirpath no llama. Costo medido (sonda `q010/u`): `ucum.New()` tarda 2,5 ms y deja 214 KiB en el heap; 100 000 `Validate` tardan 1,1 ms. Se carga una sola vez, con `sync.Once`, y solo cuando un string llega a ToQuantity.
