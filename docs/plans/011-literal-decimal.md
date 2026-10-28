# 011: el literal Decimal de más de ocho decimales

Backlog 1c23, segundo frente. Rama: `test/decimal-literal-1c23`. Estado: paso 8 terminado, el review converge en la ronda 2. Plan aprobado el 27-10-2026.

## Por qué

El 010 dejó D851 de `quantitystrings` como defecto propio: `ToQuantity(1.000000005)` falla con «decimal overflow: too many fractional digits», y Java y cql-execution dan `1.000000005 '1'`. El error viene del literal, no de ToQuantity: `validateCQLDecimal` (`eval/evaluator.go`) rechaza todo literal Decimal de más de 8 decimales o de más de 28 cifras enteras.

## Revisión previa (sondas del 27-10-2026, scratchpad `q011/`)

**La especificación.** CQL (Types, Decimal) da al tipo «a precision of 28 and a scale of 8», y dice: «implementations must support at least the 28 digits of precision and 8 digits of scale, but may support more precise values» (ADR 002, Contexto). No dice qué hacer con un literal que tiene más.

**La suite oficial** (cqframework/cql-tests, `ValueLiteralsAndSelectors.xml`, igual en el commit fijado y en main de hoy):

- `DecimalTenthStep`, `DecimalPosTenthStep` y `DecimalNegTenthStep` (`0.000000001` y sus signos) son `invalid="true"`: «Either some error saying the literal uses more precision than the Decimal step size, or round to a whole step».
- `Decimal10Pow28` y sus signos (`10000000000000000000000000000.00000000`) son `invalid="true"`: «Some error saying the literal is a greater magnitude than a Decimal supports».
- `DecimalFractionalTooBig` (`5.999999999` da `5.999999999`) está en `CqlTypesTest.xml` dentro de un comentario («REPLACED BY … ValueLiteralsAndSelectors.xml»): no rige.
- El runner (`conformance/conformance_test.go:233`) exige un error en todo test `invalid`.

**Las referencias** (Java 5.4.0 y cql-execution, sonda `q011/lit.cql`):

| Expresión | Java | cql-execution | este motor |
|---|---|---|---|
| `1.000000005` | 1.000000005 | 1.000000005 | error |
| `0.000000001` | 0.000000001 | 1e-9 | error |
| `1.000000005 + 0` | 1.00000000 | 1.000000005 | error |
| `1.000000005 - 0` | 1.000000005 | 1.000000005 | error |
| `Round(1.000000005, 9)` | 1.000000005 | 1.000000005 | error |
| `10000000000000000000000000000.00000000` | lo lee | 1e+28 | error |
| `1.000000005 'mg'` | 1.00000000 'mg' | 1.000000005 'mg' | 1.000000005 'mg' |

Ninguna de las dos referencias hace lo que pide la suite: no fallan ni redondean, leen el literal tal como está escrito. El motor sí cumple la suite: el error es una de las dos respuestas que admite.

**gofhir/server** (congelado) no evalúa CQL por su cuenta: usa esta librería. No hay nada que tomar.

**Conclusión de la revisión.** D851 no es un defecto propio claro: el motor cumple la suite oficial y difiere de las dos referencias, que no la cumplen. Hay que elegir entre tres respuestas (mantener el error, redondear a la escala o leer el literal como está escrito), y la elección es del usuario.

## Decisiones del usuario (27-10-2026)

- **Se mantiene el error** para el literal Decimal que tiene más precisión que el paso de la escala o más magnitud de la que admite la suite: más de 8 decimales con valor, o más de 28 cifras enteras con valor (`DecimalTenthStep` y `Decimal10Pow28`). La especificación permite leerlo («may support more precise values»), y así lo hacen Java y cql-execution; la suite pide un error o un redondeo, y el motor sigue la suite.
- **La suite (1c24) se actualiza después del 011:** los tests de estos literales son iguales en el commit fijado y en main.
- **Decisión D: se cuentan las cifras con valor, no los caracteres.** Los ceros a la derecha de los decimales y a la izquierda del entero no cuentan: `1.000000000` y `000000000000000000000000000001.0` se leen, como en Java, y conservan lo escrito.
- **Decisión E: el alcance del error se documenta y va al backlog.** El error es de evaluación, así que hace fallar toda `EvaluateLibrary`, y un literal en una rama que no se evalúa no falla. Llevar el control a la traducción (sema) es otro frente.

## Alcance

1. **`validateCQLDecimal`** cuenta las cifras con valor (decisión D). Los límites son los de hoy, 28 y 8, y quedan escritos en el ADR 002 con su fuente.
2. **Área `decimalliterals`** (nueva), con Java 5.4.0 y cql-execution: el literal solo, con signo, en `+`, `-`, `*`, `=`, `Round`, en listas e intervalos, en una rama muerta; los bordes de 20, 21, 28 y 29 cifras enteras y de 8 y 9 decimales; los ceros sobrantes; `-0.0` y `-0.00000000`; `ToDecimal` del mismo texto; los literales Quantity de 9 decimales y de 29 cifras. **Regla escrita antes de clasificar:** cql-execution calcula en binary64, que guarda 17 cifras significativas y ninguna escala, así que toda divergencia suya por un valor de más de 17 cifras o por la escala perdida es defecto de la referencia; las sumas y productos de Java llevan FLOOR (ADR 002), y que Java lee el literal solo se ve en el literal solo, `-`, `=` y `Round`.
3. **Un test de la biblioteca entera** fija el alcance del error (decisión E).
4. **D851 de `quantitystrings`** pasa de defecto propio a diferencia deliberada.
5. **Issue para cqframework/cql-tests** (redactado; se publica con la aprobación del usuario): la suite pide un literal de 28 cifras enteras (`Decimal10Pow28ToZeroOneStepDecimalMaxValue`) y la especificación da al tipo un máximo de 20.

**Fuera, al backlog:** el control en la traducción (decisión E); el rango del literal Quantity; el literal con exponente en el ELM; `(maximum Decimal) + 1.0`, que da un valor fuera del rango donde el ADR 002 dice null. El mensaje de error queda como está.

## Diferencias deliberadas

- **El literal de más precisión que el paso de la escala o de más de 28 cifras enteras da error** (decisión del usuario; `DecimalTenthStep` y `Decimal10Pow28`). Java y cql-execution lo leen. D851 de `quantitystrings`, y en `decimalliterals`, contra las dos referencias: D1, D2, D3, D4, D5, D6, D7, D9, D22, D23, D24, D25, D26, D27, D28, D29, D30, D31, D32, D35, D36, D37, D40, D41, D42, D44 y D46.
- **El literal Quantity conserva sus cifras** (ADR 002; `QuantityFractionalTooBig` de la suite): `5.999999999 'g'` da `5.999999999 'g'`, y Java lo recorta a la escala. En `decimalliterals`, contra Java: D55, D56 y D58.

## Criterio de salida

- `decimalliterals` en el manifiesto, con todas sus divergencias clasificadas, y `TestAnswersMatchTheSnapshots` y el oráculo de divergencias en verde.
- `TestConformance` en verde: los seis `invalid` siguen dando error.
- D851 deliberada; `quantitystrings` sin defectos propios contra Java.
- `references.sh check` en verde; diferencial 8008 contra main; `make check` en verde.
- Mutaciones (`scripts/mutations/011-literal.sh`), todas atrapadas: el error quitado; los límites en 9 y en 29; los ceros contados otra vez como caracteres; el control aplicado al literal Quantity; redondear en vez de fallar; una divergencia deliberada que el plan no nombra.

## Tests

- **La suite oficial:** `DecimalTenthStep`, `DecimalPosTenthStep`, `DecimalNegTenthStep`, `Decimal10Pow28`, `DecimalPos10Pow28`, `DecimalNeg10Pow28` y `QuantityFractionalTooBig`.
- **`decimalliterals`:** unas 60 definiciones, con Java y cql-execution.
- **Unitario de `validateCQLDecimal`:** casos escritos a mano desde la suite y la decisión D, sin correr el código para sacar lo esperado.
- **Biblioteca entera:** `EvaluateLibrary` con un literal malo entre definiciones buenas.

## Clasificación de `decimalliterals` (60 definiciones, 27-10-2026)

Contra Java, 30 deliberadas y 1 defecto propio; contra cql-execution, 27 deliberadas, 7 defectos de la referencia y 2 defectos propios. Ninguna otra área movió una respuesta.

- **Deliberadas:** las de la lista de arriba.
- **Defectos de cql-execution:** D18, D19, D20, D21, D38, D52 y D53, por binary64 (la regla del alcance).
- **Defecto propio, contra cql-execution:** D59. El literal Quantity de 29 cifras enteras se lee aquí y en Java; cql-execution lo rechaza con su control del rango del Decimal (`isValidDecimal`, hasta `99999999999999999999.99999999`), que es el de la especificación. El rango del literal Quantity va al backlog (review 1, F1).
- **Defecto propio, anterior y fuera del alcance:** D39, `ToString(000000000000000000000000000001.0)` da `"000000000000000000000000000001.0"`, y Java da `"1.0"`. Pasa igual en main con `ToString(0001.0)`: el Decimal de fhirpath imprime el texto del que se leyó, ceros incluidos, y fhirpath v1.16.6 da lo mismo sola (`0001.0.toString()` es `'0001.0'`). La especificación de CQL, ToString de Decimal, da `(+|-)?#0.0#`. Va al backlog.

## Mutaciones (`scripts/mutations/011-literal.sh`, 27-10-2026)

Son 13 (12 hasta el review, más una de la ronda 1) y todas se atrapan. Cada una nombra el comando que tiene que fallar y el texto de la falla:

- **El error:** quitado, visto por el snapshot, por la suite (`DecimalTenthStep`) y por el test de la biblioteca entera; reemplazado por un redondeo a la escala, la otra respuesta que admite la suite.
- **Los límites:** 9 decimales; 29 cifras enteras.
- **Los ceros (decisión D):** los de la derecha contados otra vez, vistos por el snapshot y por los casos escritos a mano; los de la izquierda contados otra vez.
- **El signo:** el control sin leer el signo, visto por los casos escritos a mano (ronda 1).
- **El literal Quantity:** el control aplicado a él, visto por `QuantityFractionalTooBig`.
- **El área:** una deliberada que el plan no nombra; el área fuera del manifiesto.

La mutación del 010 «an own defect expected as this engine answers» apuntaba a D851, que dejó de ser defecto propio: ahora apunta a D39 de `decimalliterals`, un defecto propio de esa área, y se sigue atrapando.

## Diferencial masiva (semilla 8008, 27-10-2026)

Las nueve áreas de la semilla 8008 dan informes idénticos byte a byte con el comparador compilado desde main (3a2f48e) y desde la rama. Las respuestas que se movieron son solo las de `decimalliterals`, que es nueva: ninguna otra área cambió.

## Riesgo de la decisión: la suite está en discusión

[cqframework/cql-tests#153](https://github.com/cqframework/cql-tests/issues/153) (abierto) argumenta que `DecimalTenthStep` y `Decimal10Pow28` no deberían ser `invalid`, por «may support more precise values», y propone quitarlos. Si la suite cambia, el error deja de ser la única respuesta conforme y la decisión queda como elección propia: se revisa en el 1c24, al actualizar la suite. [cqframework/cql-tests#145](https://github.com/cqframework/cql-tests/issues/145) (abierto) ya señalaba que los tests `…ToZeroOneStepDecimalMaxValue` pasan de `maximum Decimal`; en lugar de un issue nuevo se le agregó la evidencia de este plan ([comentario](https://github.com/cqframework/cql-tests/issues/145#issuecomment-6459208346), 27-10-2026).

## Review, ronda 1 (Fable y Codex, 27-10-2026)

Ninguno dejó hallazgos de severidad alta. Los dos corrieron la suite, el lint y las 12 mutaciones. Cada hallazgo se verificó:

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| F1 | D59 contra cql-execution no es un error interno: la referencia rechaza la Quantity con su control del rango del Decimal, el de la especificación | media | Confirmado en `decimalliterals.js.json` y en `cql-execution/lib/datatypes/quantity.js`. Pasa a defecto propio, ligado al backlog |
| F2 | El comentario del literal describía el conteo por caracteres | baja | Corregido |
| F3, C1 | Quitar la lectura del signo no lo ve ningún test: el parser deja el signo a un operador | baja | Casos con signo en los tests escritos a mano, y su mutación |
| F4 | El test de recursos armaba literales de 10 a 20 MB (110 MB de RSS) | baja | Un millón de cifras: la prueba de cero asignaciones vale igual |
| F5 | La regla de binary64 no cubría D38, que es por la escala | baja | La regla nombra la escala |

## Review, ronda 2 (Fable y Codex, 28-10-2026): converge

Ninguno dejó hallazgos de severidad media o mayor. Codex no dejó ninguno: `go test -race ./...`, las 13 mutaciones del 011 y las 31 del 010, el lint v2.15.0, y su mutación del signo, que ahora falla en los casos con signo. Fable confirmó las correcciones de la ronda 1 (el test de recursos baja de 110 MB a 19 MB de RSS) y dejó dos bajos:

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| F1 | El plan y la mutación del 010 decían que D39 es el único defecto propio del área, y desde la ronda 1 también lo es D59 | baja | Corregido |
| F2 | D59 esperaba el mensaje entero de cql-execution, con su posición en el ELM; la convención para un defecto propio cuya respuesta es un error es `error` (D104 de `types`) | baja | Corregido |

## Ronda adversarial (Codex y Fable, 27-10-2026)

Cada hallazgo se verificó ejecutando, en el motor y en Java 5.4.0 (sonda `q011/b.cql`):

| Expresión | Java | cql-execution | este motor |
|---|---|---|---|
| `maximum Decimal` | 99999999999999999999.99999999 | 1e20 | 99999999999999999999.99999999 |
| `(maximum Decimal) + 1.0` | 100000000000000000000.99999999 | null | 100000000000000000000.99999999 |
| `123456789012345678901.0` (21 cifras) | lo lee | lo lee en binary64 | lo lee |
| `ToDecimal('123456789012345678901.0')` | null | null | null |
| `1.000000000` | 1.000000000 | 1 | error |
| `000000000000000000000000000001.0` | 1.0 | 1 | error |
| `-0.0` | 0.0 | 0 | 0 |
| `10000000000000000000000000000.0 'mg'` | lo lee | error | lo lee |
| `5.999999999 'g'` | 5.99999999 'g' | 5.999999999 'g' | 5.999999999 'g' |
| biblioteca con `define Bad: 0.000000001` y `define Good: 1` | — | — | `EvaluateLibrary` da error y ningún resultado |

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| F1, C2 | Hay dos rangos: el literal admite 28 cifras enteras (la suite) y el tipo llega a 20 (`maximum Decimal`, la especificación: (10^28 − 1)/10^8). El plan decía «28 cifras o más», y son más de 28 | alta | Aceptado. El corpus prueba los bordes de 20, 21, 28 y 29 cifras. El límite del literal sigue en 28: con 20 fallarían los tres tests `Decimal10Pow28ToZeroOneStep…` de la suite, que esperan `9999999999999999999999999999.99999999`. La contradicción entre la suite y la especificación se reporta (borrador de issue para cqframework/cql-tests). Se escribe la regla del literal en el ADR 002 |
| F2, C4 | El error es de evaluación: hace fallar toda la biblioteca, y depende del orden (`if false then 0.000000001 else 1.0` da 1.0) | alta | Confirmado. Decisión del usuario (decisión E) |
| F3, C3 | Se cuentan caracteres, no cifras: `1.000000000` y `0…01.0` fallan, y valen 1 | media | Confirmado. Decisión del usuario (decisión D) |
| F4 | El literal Quantity no tiene control de rango, y la suite activa tiene `QuantityFractionalTooBig` (`5.999999999 'g'` conserva las cifras), que Java no cumple | media | Aceptado: se cita la suite. El rango del literal Quantity va al backlog |
| F5, C6 | Mutaciones: el control aplicado al literal Quantity, los ceros tolerados, redondear en vez de fallar | media | Aceptado. La de «D851 otra vez defecto propio» se retira: la clase es metadato, el oráculo no la puede decidir (1c22), y lo que fija el comportamiento es el snapshot |
| F6 | cql-execution calcula en binary64: toda divergencia de más de 17 cifras es defecto suyo; las sumas y productos de Java llevan FLOOR | baja | Aceptado: la regla se escribe antes de clasificar |
| F7 | En el ELM, un literal con exponente (`1E-9`) no pasa por el control | baja | Al backlog: solo los tests llaman a `elm.Import` |
| C1 | Cumplir la suite no es lo mismo que lo exija la especificación, que permite más precisión | media | Aceptado: el plan separa las dos cosas |
| — | `(maximum Decimal) + 1.0` da un valor fuera del rango, y el ADR 002 dice null (igual en Java) | — | Anterior y fuera del alcance: al backlog |

## Encontrado aparte

- **La suite de conformance está atrasada:** el commit fijado en `conformance/testdata/SOURCE.tsv` (727219f) va 56 commits detrás de main de cqframework/cql-tests (f724ee4, 20-10-2026), con unas 1 200 líneas de tests nuevas en los 15 archivos. Va al backlog como su propio incremento.
