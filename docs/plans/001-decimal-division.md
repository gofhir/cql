# 001: división Decimal

**Estado:** cerrado el 18-10-2026, sin commit. **Fecha:** 18-10-2026.

## Problema

Con un operando Decimal, `div` devuelve un Integer en vez de un Decimal: `7.0 div 2`, `10.1 div 3.1` y `1000.0 div 1000` son Integer.

Ningún oráculo lo detectaba, porque tanto la conformance (`valuesEqual`) como el comparador de las referencias (`answers.Same`) comparan los números por valor, y para los dos el Integer 3 es igual al Decimal 3.0.

## 1. Revisión previa

- **`gofhir/server`:** no implementa `div` de CQL. Las coincidencias de «div» en su código son `<div>` de la narrativa HTML, y la evaluación la delega en go-cql. **Se descarta:** no hay nada que tomar.
- **Especificación:** CQL, Appendix B, Arithmetic Operators, «Truncated Divide» (`#truncated-divide`).
  - Firmas: `div(Integer, Integer) Integer`, `div(Decimal, Decimal) Decimal`, `div(Long, Long) Long` y `div(Quantity, Quantity) Quantity`. Las de Long y Quantity son de uso a prueba desde la 1.5.
  - Con argumentos mixtos, el Integer se convierte a Decimal o a Long, y el Long a Decimal.
  - El resultado es null si un argumento es null, si el divisor es 0 o si el resultado no se puede representar.
  - Página leída: la versión 2.0.0 (R2 STU 1); la entrada viene de la 1.5.
- **Conformance oficial:** `conformance/testdata/CqlArithmeticFunctionsTest.xml`, grupo «Truncated Divide». Espera Decimal en `10.1 div 3.1` = `3.0`, `-10.1 div -3.1` = `3.0` y los demás casos con Decimal.
- **Otras implementaciones:** el motor Java 3.26.0 y cql-execution 3.3.2 devuelven Decimal en todos los casos con un operando Decimal.
  - Evidencia en el repo: en `testdata/snapshots/arithmetic.java.json` y `arithmetic.js.json`, 20 definiciones con `div` sobre Decimal, D55, D89, D100, D146, D204, D255, D326, D364, D368, D369, D380, D417, D421, D450, D549, D582, D606, D668, D687 y D764, son `Decimal` en ambas referencias e `Integer` en este motor.
  - En Quantity difieren entre sí: Java da `7.5 'cm' div 2 'cm'` = `3.0 'cm'`, y cql-execution da null.
- **El código propio:**
  - `eval/evaluator.go`, en la rama Decimal de `evalArithmetic`, devuelve `funcs.IntegerOf(quotient)` desde el commit inicial. #134 solo le agregó la guarda contra el desborde de int64.
  - `sema` ya tipa `Decimal div Decimal` como Decimal: la evaluación contradecía al tipo estático.
- **Los oráculos, medidos en una copia del repo:**
  - Si `valuesEqual` distingue Integer de Decimal, fallan exactamente 5 casos de la conformance, todos de «Truncated Divide»: `TruncatedDivide10By5D`, `10d1By3D1`, `10d1ByNeg3D1`, `Neg10d1By3D1` y `Neg10d1ByNeg3D1`.
  - Si `answers.Same` distinguiera también el tipo, en `arithmetic` aparecerían las 20 de Java y 27 de cql-execution: las mismas 20 más 7 suyas, en las que rotula Integer una potencia negativa de un entero, como `10 ^ -1` = «Integer 0.1». En `integer` aparecerían 235 de Java y 124 de cql-execution donde ellas dicen Long, por la deuda D4c: Long no tiene tipo propio en ejecución.

## 2. Plan

### Alcance

1. **El motor:** `div` con un operando Decimal devuelve un Decimal, el cociente truncado hacia cero, y null si queda fuera del rango Decimal (`maxDecimal`). `Integer div Integer` no cambia.
2. **La conformance:** `valuesEqual` distingue Integer de Decimal. Los Long de la salida esperada se leen como Integer, así que no cambian.
3. **El comparador de las referencias:** `answers.Same` distingue un número entero, Integer o Long, de un Decimal, también dentro de listas e intervalos (hallazgos C1 y F3). **Integer y Long siguen siendo equivalentes** hasta D4c; esa pérdida conocida queda documentada en un test.
4. **La conformance en contenedores:** `valuesEqual` aplica la misma distinción de forma recursiva en listas, intervalos y tuplas (C1). Antes de fijarlo se mide cuántos casos cambian.

### Fuera del alcance

- **La unidad del cociente de Quantity:** la especificación pide «una unidad UCUM apropiada», y las referencias no coinciden entre sí.
- **`div` sobre intervalos de incertidumbre:** hoy es un error.
- **La escala del resultado:** Java conserva `1111.000000`. El oráculo compara por valor, y la especificación no fija una escala para `div`.
- **El tipo Long en ejecución:** es la deuda D4c.
- **`div` entre una Quantity y un número** (`10 'cm' div 3`): hoy es un error, aunque la especificación convierte el número a Quantity (F5).
- **`div` sobre intervalos numéricos comunes,** además de los de incertidumbre (C4).
- **La tolerancia de 1e-13 del comparador** para Decimals de más de 15 cifras (C3): ver la resolución.

### Criterio de salida

- Los 5 casos de la conformance pasan con el comparador estricto.
- `TestAnswersMatchTheReferences`, con el `Same` nuevo, no tiene divergencias de tipo contra Java en `arithmetic`. Las 7 de cql-execution quedan en el manifiesto como defecto de la referencia, con su evidencia: un Integer no puede valer 0,1, y Java coincide con este motor.
- El snapshot de `arithmetic` cambia exactamente en esas 20 respuestas, de `Integer` a `Decimal` con el mismo valor, y en ninguna otra. Los de las otras áreas no cambian, o cada cambio queda explicado.

### Tests

- **Conformance:** los casos de «Truncated Divide», con el comparador estricto. Son casos de la especificación.
- **Un test del motor** con valores que salen de la especificación y de las referencias, no de correr el código propio. Cada valor esperado se mide con Java y cql-execution, y la medición queda en el repo como un corpus del arnés:
  - `-7.5 div 2` = -3.0, porque se trunca hacia cero, y `-1.5 div 2` = 0;
  - `7 div 2.0` y `7.0 div 2`, en ambos órdenes de argumentos mixtos;
  - `7L div 2.0`, Long con Decimal;
  - `10.1 div 0.0` = null;
  - **cocientes entre int64 y `maxDecimal`,** en ambos signos (C2 y F2): `10000000000000000000.0 div 1.0` y `99999999999999999999.99999999 div 1`. Hoy dan null, aunque son Decimals válidos;
  - **un cociente fuera del rango con operandos válidos** (`99999999999999999999.99999999 div 0.00000001`) = null, porque «no se puede representar». Java no acota BigDecimal, así que su respuesta será un número: el null sale de la especificación, y la diferencia entra al manifiesto como deliberada, con la sección citada (F2);
  - **cocientes chicos y casi enteros:** `0.00000001 div 3` = 0, y `29999999999.99999999 div 10000000000.0` = 2, que el redondeo a 16 cifras volvería 3 (F4);
  - **`mod` con signo,** con la reconstrucción `a = (a div b) * b + (a mod b)` dentro del rango (C5): `-7.5 mod 2` y `7.5 mod -2`;
  - **operandos FHIR** (C4 y F1): `(Observation.value as FHIR.Quantity).value div 2` da Decimal, y `FHIR.integer div FHIR.integer` sigue siendo Integer, como guarda para que el arreglo no desvíe los enteros a la rama Decimal;
  - **ELM importado:** `TruncatedDivide` con un operando Decimal (C4).
- **El test de la pérdida conocida:** Integer y Long son equivalentes en `answers.Same`.
- **Versiones FHIR:** `div` es aritmética del sistema y no lee el modelo. Un `FHIR.decimal` llega convertido a System.Decimal por FHIRHelpers. El área `fhir` **no** tiene `div`: lo medí, con 0 expresiones, así que la cobertura sale de los casos con operandos FHIR del test del motor (F1).

### Fuentes de datos

- La conformance oficial.
- Los corpus generados de `arithmetic`, con 800 expresiones, y de `fhir`, con 900.
- Las respuestas de Java y cql-execution ya versionadas.

### Mutaciones de los oráculos, antes del review

| Mutación | Qué tiene que fallar |
|---|---|
| Volver a devolver Integer | La conformance (5) y el manifiesto (20 nuevas) |
| Devolver el cociente sin truncar (`Div` en vez de `QuoRem`) | La conformance y el snapshot |
| Truncar hacia abajo en vez de hacia cero (`Floor`) | `-7.5 div 2` en el test y el snapshot |
| Que `answers.Same` vuelva a comparar solo por valor | El test de `Same` |
| Que `Same` ignore el tipo dentro de listas e intervalos | El test de `Same` con contenedores |
| Que `valuesEqual` vuelva a solo `Equal` | La conformance (5 casos) |
| Volver a `Div(...).Truncate(0)`, que redondea a 16 cifras | `29999999999.99999999 div 10000000000.0` |
| Quitar la guarda de `maxDecimal` | El cociente fuera de rango |
| Mantener la guarda de int64 (`IntegerOf`) | Los cocientes entre int64 y `maxDecimal` |
| Escalar antes de truncar (`ScaleDecimal`, que hace FLOOR) | Los cocientes chicos negativos |
| Mover un cociente grande en una unidad | Ver la resolución de C3 |

Cada mutación se corre en una copia del repo, y el plan registra el comando y su resultado (C6).

### Excepciones a la regla 5 y dependencias

**Excepciones:** `div` es semántica de CQL definida en la prosa de su especificación, no algo que se lea de un package FHIR. Como la semántica entera de la librería es así, propongo **un ADR único para cql** en vez de uno por operador (C6). Ese ADR declararía que la semántica de CQL se programa desde la especificación de CQL, y que se verifica con la conformance oficial y con las dos referencias en `go test`. Queda como decisión pendiente del usuario.

**Dependencias:** ninguna nueva.

**`make check`:** el repo no tiene Makefile. Su equivalente es gofmt, golangci-lint v2.15.0, `go test -race ./...` y las seis zonas horarias. Agregar el Makefile queda como decisión pendiente (C6).

## 3. Ronda adversarial

Hubo dos revisores en paralelo: Codex, que leyó el código sin ejecutar nada, y Fable, que corrió sondas. Los hallazgos C1 a C6 son de Codex y F1 a F7 de Fable. Verifiqué con una sonda los marcados como verificados.

| # | Hallazgo | Resolución |
|---|---|---|
| C1, F3 | `Same` y `valuesEqual` siguen ciegos al tipo dentro de listas, intervalos y tuplas | **Aceptado:** alcance 3 y 4, con mutaciones |
| C2, F2 | Falta el rango entre int64 y `maxDecimal`. Verificado: `1e19 div 1.0` da null hoy | **Aceptado:** casos en ambos signos, con mutación de la guarda de int64 |
| F2 | Java no acota BigDecimal, así que el null fuera de rango será una divergencia | **Aceptado:** se anticipa como diferencia deliberada, con la sección citada |
| C3 | La tolerancia de 1e-13 deja pasar un cociente grande que difiere en una unidad | **Aceptado en parte:** los tests del motor comparan exacto. En el comparador, mido si algún ruido de float de las referencias tiene valor entero; si ninguno lo tiene, la comparación de valores enteros pasa a ser exacta. Si alguno lo tiene, la tolerancia queda como está y el punto ciego se documenta |
| C4, F1 | El área `fhir` no tiene `div`. Verificado: 0 expresiones | **Aceptado:** casos con operandos FHIR y con ELM importado en el test del motor |
| C4 | Operandos de funciones y agregados | **Rechazado:** F6 verificó que el único camino de `div` es `evalArithmetic`, y las funciones y los agregados llegan ahí con valores ya evaluados |
| C4, F5 | Quantity con número, e intervalos numéricos | **Aceptado como exclusión,** declarada en «Fuera del alcance» |
| C5 | `mod` con signo no se prueba | **Aceptado:** casos con signo y la reconstrucción. F6 verificó que `mod` usa el mismo `QuoRem` |
| F4 | Falta la mutación del redondeo a 16 cifras | **Aceptado:** caso y fila en la tabla |
| C6 | Mutaciones sin comando registrado; falta `make check`; ADR | **Aceptado:** comandos en el plan. El ADR único y el Makefile quedan para el usuario |
| C6 | El manifiesto mira qué definiciones divergen, no qué responden | **Rechazado para este incremento:** `TestAnswersMatchTheReferences` compara la respuesta actual de las referencias con la del motor en cada corrida, así que una respuesta distinta aparece como divergencia nueva |
| F7 | La conformance espera `3.0` y el motor dará «Decimal 3» | **Aceptado:** el criterio «mismo valor» se refiere al número, no al texto |

## 4. Aprobación

El usuario aprobó el plan ajustado el 18-10-2026. También decidió dos cosas:
- **Un ADR único** para la semántica de CQL, en un incremento propio.
- **El Makefile** con `make check` se agrega en el cierre de este incremento.

## 5 a 7. Tests, implementación y validación

- **Corpus `division`** (`testdata/snapshots/division.cql`), escrito a mano, con 29 casos:
  - sus respuestas de Java 3.26.0 y cql-execution 3.3.2 están versionadas (`division.java.json` y `division.js.json`);
  - lo cubren `TestAnswersMatchTheReferences` y `TestAnswersMatchTheSnapshots`.
- **Tests:**
  - `engine_decimal_division_test.go`: operandos FHIR, ELM importado y el cociente fuera de rango;
  - `conformance/values_equal_test.go`: el comparador de la conformance;
  - `TestNumbersAreComparedByValueWithinTheirType` y `TestAContainersNumbersKeepTheirType`, en `answers`;
  - dos casos de enteros grandes en `TestAFloat64sNumbersAreTheSameAcrossPlatforms`.
- **Lo que se encontró en el camino:**
  - **El XML oficial escribe sin tipo cuatro salidas de `expand … per 1`.** Java responde puntos Decimal, como el motor. Esos casos quedan en `untypedOutputs` y se comparan por valor; si una entrada deja de hacer falta, la conformance falla.
  - **`TestListsAreComparedByTheirElements`** tenía escrita a mano una referencia con Integer que Java no produce: Java promueve la lista mixta a Decimal, como en D22.
  - **La tolerancia de 1e-13** salva 11 pares de ruido de float, y en uno, D456 de `arithmetic`, Java escribió un valor entero. Por eso la comparación exacta se limita a dos enteros cuando uno tiene más de 17 cifras, las que un float64 escribe. Queda como punto ciego conocido un entero de 16 o 17 cifras que difiera en una unidad.
  - **Observación pendiente, sin verificar contra las referencias:** un `valueQuantity` que solo tiene `value` no es `FHIR.Quantity` para el motor (`First([Observation]).value is FHIR.Quantity` da false). Es otro incremento.
- **Divergencias nuevas, clasificadas con su evidencia en los manifiestos:**
  - de cql-execution: D7 a D10 (null dentro del rango Decimal) y D14 (redondeo de float) de `division`, y D154, D749 y D767 (potencias negativas rotuladas Integer) de `arithmetic`;
  - de Java: D11 de `division`, la diferencia deliberada por el rango.
- **Snapshot:** cambian exactamente las 20 respuestas de `arithmetic` que el plan anticipaba, de Integer a Decimal con el mismo valor. Ninguna otra área cambia.
- **Mutaciones:** `sh scripts/mutations/001-decimal-division.sh` aplica cada defecto en una copia del repo.
  - Antes exige una línea base en verde.
  - Una mutación que no compila cuenta como sobreviviente, y cada una tiene que hacer fallar el test que declara.
  - El script termina con código distinto de 0 si alguna sobrevive.
  - Tras la ronda 1 son 15 mutaciones.
  - Las de contenedores (`Same` dentro de listas e intervalos, `valuesEqual` recursivo) las detectan solo sus tests unitarios: ningún caso del corpus enfrenta un elemento Integer con uno Decimal (F3 de la ronda 1).
- **Chequeos:** gofmt, golangci-lint v2.15.0, `go test -race ./...` y las seis zonas horarias, en verde.

## 8. Code review

### Ronda 1 (Codex y Fable)

Fable ejecutó todo en una copia del repo, incluidas 43 sondas propias, y no encontró defectos en el arreglo del motor. Codex leyó el código sin ejecutar nada.

| Hallazgo | Severidad | Resolución |
|---|---|---|
| El script cuenta como detectada una falla de compilación o de una línea base rota (Codex) | Media | **Corregido:** línea base obligatoria, compilación fallida cuenta como sobreviviente, y un test esperado por mutación |
| `untypedOutputs` acepta tipos mezclados, como `{Decimal 10, Integer 11}` (Codex) | Media | **Corregido:** la salida esperada se lee como Decimales y se compara estricto. Lo prueba `TestAnUntypedOutputIsReadAsDecimals` |
| Con ceros finales, `10000000000000000000` y `10000000000000100000` pasan como iguales (Codex) | Media | **Corregido:** `SameExactly` compara sin tolerancia en las áreas sin float (`exact` en `snapshotAreas`; `division` lo es). Lo prueba `TestACorpusWithoutFloatsIsComparedExactly` |
| El test de `Same` no cubría referencia Decimal contra motor Integer, la forma del defecto; la mutación asimétrica sobrevivía (Fable) | Media | **Corregido:** se agregaron el caso y la mutación |
| La regla exacta de `Same` vale solo para dos enteros (Fable) | Baja | Documentado en el comentario |
| Las mutaciones de contenedores las detectan solo tests unitarios (Fable) | Baja | Registrado arriba |
| Mutantes equivalentes: `LessThan` en `WithinDecimal`, y `Mod` contra `QuoRem` (Fable) | Baja | Sin acción: no se distinguen por construcción |
| El script no corría `./eval/` ni `./funcs/` (Fable) | Baja | **Corregido** |

### Ronda 2 (Codex y Fable)

Los revisores recibieron solo los hallazgos de la ronda 1 y lo que cambió. **La ronda converge:** ninguno dejó hallazgos medios o mayores.
- **Fable lo verificó ejecutando** en una copia del repo:
  - las 15 mutaciones se detectan, cada una por su test;
  - el script informa SURVIVED y BUILD FAILED cuando corresponde;
  - mutó también los arreglos de la ronda 1: `asDecimals` sin intervalos, y `SameExactly` sin distinguir el tipo. Las dos mutaciones se detectan;
  - la verificación de entradas obsoletas de `untypedOutputs` funciona.
- **Codex** revisó el diff sin ejecutar nada.

| Hallazgo | Severidad | Resolución |
|---|---|---|
| Los comentarios del script estaban en español, contra la decisión de mantenerlos en inglés (Codex) | Baja | **Corregido** |
| El snapshot de un área exacta toleraba 1e-13: 10^19 + 1 pasaba por 10^19, aunque la referencia exacta lo detectaba (Fable) | Baja | **Corregido:** en un área exacta, el snapshot compara el texto. Se agregó la mutación «10^19 answered as 10^19 + 1»: ya son 16 |

## 9. Cierre

- **`make check`:** el `Makefile` es nuevo, como pidió el usuario. Corre gofmt, golangci-lint v2.15.0, `go test -race ./...`, las seis zonas horarias y las mutaciones de este incremento. Para comprobar que está en verde hay que correrlo: el registro no copia su salida.
- **`make references`:** remite al README del arnés, porque los comandos dependen del área y necesitan Docker.
- **Se cumple el criterio de salida:**
  - pasan los 5 casos de la conformance con el comparador estricto;
  - no queda ninguna divergencia de tipo contra Java en `arithmetic`;
  - el snapshot de `arithmetic` cambió exactamente en las 20 respuestas previstas.
- **Queda pendiente, fuera de este incremento:**
  - el ADR único de la semántica de CQL;
  - la observación de `valueQuantity` sin unidad, que no es `FHIR.Quantity` para el motor, sin verificar contra las referencias;
  - la unidad del cociente de Quantity;
  - `div` entre una Quantity y un número;
  - el tipo Long en ejecución (D4c).
