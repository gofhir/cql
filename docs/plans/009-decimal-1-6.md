# 009: shopspring/decimal v1.4.0 → v1.6.0

**Fecha:** 26-10-2026 · **Rama:** `chore/decimal-1.6` · **Versión:** `fix:` (cambia respuestas del motor)

## Por qué

El motor importa `github.com/shopspring/decimal` v1.4.0 de forma directa, y la última versión es la v1.6.0. La regla del usuario es usar siempre la última versión. fhirpath v1.16.6 pide v1.4.0, pero en la compilación del motor corre con la versión que fije el motor (MVS).

Del changelog de v1.5.0 y v1.6.0, estos son los cambios de comportamiento que tocan funciones que usa el motor:

- **v1.5.0:**
  - `NewFromString` rechaza exponentes mayores que `MaxDecodeExponent` (10.000, global).
  - `Pow` redondea correctamente.
  - `RoundFloor` y sus pares siempre devuelven el exponente `-places`.
  - `Truncate` acepta precisión negativa.
  - `NewFromString` rechaza un signo después del punto (`.-5`).
- **v1.6.0:**
  - `NewFromString` acepta el signo menos Unicode `−` (U+2212).
  - `Ln` redondea correctamente.
  - `Decimal` implementa `fmt.Formatter`, y `%v` y `%s` siguen igual.

## Revisión previa (sondas del 26-10-2026, en el scratchpad)

- **`gofhir/server`:** no fija decimal a través de cql. No hay nada que tomar de él para este incremento.
- **`go test ./...` con v1.6.0:** falla un test, `TestAStringWithAHugeExponentIsAnsweredAtOnce`, porque `ToDecimal('0e999999999')`, `'1e-999999999'`, `'-1e-999999999'` y `'1e999999999'` pasan a dar error de evaluación. Pasan los snapshots, las referencias Java y JS, el oráculo y la conformance.
- **Diferencial generada (seed 8008, 12.350 expresiones, nueve áreas):** los reportes de v1.6.0 son idénticos byte a byte a los de main. Esto incluye la aritmética, `Power`, `Ln`, `Exp`, `Round`, `Truncate`, los agregados y `expand`.
- **`RoundFloor` y el exponente:** `ScaleDecimal` solo llama a `RoundFloor` cuando el valor tiene más de ocho decimales, y en ese caso el exponente siempre fue `-8`. Una sonda de 76 expresiones sensibles a la escala da respuestas idénticas en main y en la rama: `ToString` de sumas, cocientes, agregados, cantidades, potencias, `Ln`, `Exp`, `HighBoundary`, `expand … per`, `predecessor`.
- **Todo lo que cambia pasa por un string leído como Decimal** (`funcs.DecimalFromString`, que usan `ToDecimal`, `ConvertsToDecimal` y `convert … to Decimal`). Sonda `dec.cql`, 30 definiciones:

| Expresión | Java 5.4.0 | cql-execution 3.3.2 | motor main (v1.4.0) | motor v1.6.0 |
|---|---|---|---|---|
| `ToDecimal('−5')` (U+2212) | null | null | error | **-5** |
| `ConvertsToDecimal('−5')` | false | false | false | **true** |
| `ToDecimal('.-5')` | null | null | -0.05 | error |
| `ConvertsToDecimal('.-5')` | false | false | true | false |
| `ToDecimal('1e10001')` | null | null | null | **error** |
| `ToDecimal('0e10001')` | 0 | 0 | 0 | **error** |
| `ToDecimal('1e-10001')` | 0.00000000 | 0 | 0.00000000 | **error** |
| `ToDecimal('1e-10000')` | 0.00000000 | 0 | 0.00000000 | 0.00000000 |
| `ToDecimal('abc')` | null | null | error | error |
| `ToDecimal('1.5.2')` | null | 1.5 | error | error |
| `ToDecimal(' 1')` | null | 1 | error | error |
| `convert 'abc' to Decimal` | null | null | error | error |
| `ToDecimal('.5')` | null | 0.5 | 0.5 | 0.5 |
| `ToDecimal('1.')`, `'+5'`, `'1.5E2'`, `'1e3'`, `'-0'`, `'00.10'` | 1, 5, 150, 1000, 0, 0.10 | iguales (0.1) | iguales | iguales |
| `ConvertsToDecimal('1e10001')` | true | false | false | false |

La causa es una sola. El motor usaba `decimal.NewFromString` como si fuera la gramática de CQL: lo que la librería aceptaba era un Decimal, y lo que rechazaba, un error. v1.5.0 y v1.6.0 cambian ese conjunto, y el motor cambia de respuesta con él. **No es un defecto de decimal.** Los tres cambios están documentados y son deliberados: el tope de exponente es una protección contra entradas que cuelgan el proceso. La gramática de un string CQL le corresponde al motor.

La especificación, en CQL Reference (Apéndice B), ToDecimal, dice: «The operator accepts strings using the following format: (+|-)?#0(.0#)?», y «If the input string is not formatted correctly, or cannot be interpreted as a valid Decimal value, the result is null». La cita está copiada del HTML local de la especificación.

**fhirpath con v1.6.0.**
- `go test github.com/gofhir/fhirpath/...`, corrido desde el módulo del motor (con v1.6.0 en la lista de compilación), falla en `TestDecimalRange` (`math_test.go:149-150`):
  - `Observation.value = 1 'mg'` da false, cuando se espera vacío;
  - `comparable` también da false;
  - `sort()` da «cannot compare Quantity with Quantity».
- Con v1.4.0 el test pasa.
- La causa: con v1.4.0, `types.ParseDecimal` (`types/decimal.go:110`) le daba `ErrDecimalRange` a una Quantity FHIR cuyo valor desborda, como `1E8227822`. Con v1.6.0, `NewFromString` le da antes un error genérico, y la Quantity queda «sin valor».
- `TestSLLPredictionGivesLLTree` falla también en main: no lo causa este cambio.

**En el motor no cambia nada.** La sonda `fq` sobre un `valueQuantity` con `5`, `1E20000`, `1E8227822`, `1E100`, `0E20000` y `1e-20000` da las mismas respuestas en main y en la rama: el motor lee la Quantity por su propio camino. Por eso no se rodea nada aquí; se reporta a fhirpath (decisión D). Las respuestas de main para esos valores (`.value` null, `>` con error, `1E100` aceptado fuera de rango) son defectos anteriores y van al backlog.

Los decimales FHIR llegan como números JSON, así que el `−` de U+2212 no llega por ahí. `NewDecimal("1e−2")` con v1.6.0 da `0.01`, pero eso solo importa en fhirpath, y va en el mismo reporte.

## Alcance

1. **Bump:** `shopspring/decimal` v1.4.0 → v1.6.0.
2. **La gramática es del motor.** `DecimalFromString` valida el string antes de leerlo, y lo que no cumple la gramática da null. La gramática es la de un BigDecimal de Java, con dígitos ASCII y sin signo después del punto:

   `^[+-]?([0-9]+(\.[0-9]*)?|\.[0-9]+)([eE][+-]?[0-9]+)?$`

   - No es la especificación. El formato `(+|-)?#0(.0#)?` no tiene exponente, ni `1.`, ni `.5`, y el motor acepta los tres desde #134.
     - El exponente y `1.` también los lee Java.
     - `.5` lo lee cql-execution, y Java da null. El motor lo conserva como estaba en main y queda clasificado como defecto propio (backlog 1c23). La ronda 1 del review lo señaló: cambiarlo movería respuestas de main fuera del alcance.
   - No es exactamente lo que aceptaba v1.4.0. Codex lo comparó en 597.882 strings:
     - 176 casos los aceptaba solo v1.4.0, y todos son un signo después del punto (`.+5` daba 0.05 y `.-5` daba -0.05). Eran defectos de la librería, y ahora dan null.
     - 5 casos los acepta solo la regex: exponentes fuera de `int32`, que v1.4.0 rechazaba. Los resuelve el punto 3.
   - `'−5'` (U+2212) y `'﹣5'` (U+FE63) dan null, como en Java (decisión A).
   - Los dígitos no ASCII (`'１'`, `'٣'`) dan null. Java los acepta: es una divergencia deliberada contra Java, a favor de la especificación (decisión A).
3. **La magnitud sale del string, con memoria acotada.**
   - El exponente escrito se lee dentro de un `int32`. Primero se le quitan los ceros a la izquierda, y si tiene más de diez dígitos no se lee: `strconv` copia en su error el texto que rechaza (ronda 1 del review). Fuera de `int32` da null. Java acota otra cosa, la escala (el exponente menos los decimales escritos): `'1.0e-2147483648'` da null en Java y 0.00000000 aquí (D204). En Java, `'1e-99999999999'` y `'0e99999999999'` dan null y `'1e2147483647'` también, mientras que en main daban error. `'0e2147483648'` da 0 en Java: es una divergencia clasificada.
   - De la mantisa se quitan los ceros a la izquierda. Se guardan los dígitos que alcanzan hasta la octava posición decimal (como mucho 28 significativos, de la potencia 19 a la −8) y una marca de resto distinto de cero, que sirve para el piso de un negativo. Con eso el motor decide:
     - un cero da 0;
     - más de 20 dígitos antes del punto, o un valor sobre el máximo, da null;
     - por debajo de la escala, da 0.00000000 o -0.00000001.
   - Recién entonces arma el valor, con `decimal.NewFromBigInt` y un coeficiente de a lo más 28 dígitos. Ninguna comparación ni piso toca un exponente enorme ni una mantisa enorme.
   - No es un workaround. La gramática de un string CQL le corresponde al motor (especificación, ToDecimal), y `decimal.go:103-116` de v1.6.0 documenta que `NewFromBigInt` no se valida y que la validación queda a cargo de quien llama. No se toca `decimal.MaxDecodeExponent`, que es global y afectaría a `gofhir/server`.
4. **Un string mal formado da null, no error** (decisión B), en `convert … to Decimal` y `ToDecimal` (el camino es `convertToType`, `eval/evaluator.go:8694-8701`), como responden Java y cql-execution. `funcs.ToDecimal` ya daba null y no tiene llamadores; queda como está.

## Fuera del alcance (al backlog)

- **Ceros con escala.** `'0.000000000'` da `0.00000000` en Java, y en el motor da `0`, igual en main y en la rama. Lo mismo pasa con `'0e-10001'` y `'0.00000000e5'`. Se conserva la respuesta de main. El corpus no lo ve, porque el comparador (`answers.Same`) iguala `0` con `0.00000000`; va al backlog, junto con 1c18.
- **`ToQuantity` de un string mal formado** da error, no null, igual en main y en la rama.
- **Quantity FHIR que desborda.** `.value` da null, `>` da error y `1E100` se acepta. Es un defecto anterior, que se ve igual con las dos versiones.
- **Defectos de Java:** `ConvertsToDecimal` da true mientras `ToDecimal` del mismo string da null o falla. La lista completa está en «Clasificación de `decimalstrings`»; `'.5'` es uno de esos casos (D13, D16 y D19).

## Decisiones del usuario (26-10-2026: A, B, C y D, las recomendadas)

- **A.** La gramática es la de BigDecimal, con dígitos ASCII y sin signo después del punto. `'−5'` da null como en Java. Los dígitos Unicode dan null, lo que es una divergencia deliberada contra Java, a favor de la especificación. *Recomendado.*
- **B.** Un string mal formado da null en vez de error, como dicen la especificación y las dos referencias. Cambia respuestas de main. *Recomendado.*
- **C.** Un exponente fuera de `int32` da null, como en Java en casi todos los casos. En main daba error. *Recomendado.*
- **D.** Reportar a fhirpath, con su test que falla, que con decimal v1.6.0 un valor FHIR que desborda deja de ser `ErrDecimalRange`. El 009 avanza porque el motor no cambia de respuesta (sonda `fq`). *Recomendado*, en vez de esperar a que fhirpath lo corrija.

## Criterio de salida

- Ninguna respuesta se aleja de Java respecto de main, salvo las decisiones A, B y C, y lo que se registra como divergencia deliberada. Lo verifican:
  - la diferencial seed 8008, main contra rama, con `-baseline`;
  - el corpus nuevo;
  - todos los corpus del repo.
- `make check` en verde, con las mutaciones de 001 a 009 cazadas.
- `references.sh check` en verde.
- El issue de fhirpath, redactado y publicado si el usuario lo aprueba.

## Tests

- **Corpus nuevo escrito a mano, `testdata/snapshots/decimalstrings.cql`**, de unas 80 definiciones, con las respuestas de Java 5.4.0 y cql-execution y cada divergencia clasificada. Cita la especificación (ToDecimal, ConvertsToDecimal, Convert) y cubre:
  - **gramática:** signos ASCII y Unicode (U+2212 y U+FE63, también dentro del exponente), dígitos árabes y de ancho completo, espacios, `''`, `'.'`, `'-'`, `'e5'`, `'1e'`, `'1e5.5'`, `'1e+3'`, `'1E-2'`, `'.-5'`, `'.+5'`, `'.+0'`, `'+.5'`, `'1_000'`;
  - **magnitud:** exponentes de ±10.000, ±10.001, ±2147483647 y ±2147483648, `0e…` de cada uno, negativos por debajo de la escala (`'-1e-10001'` da -0.00000001 en Java), 20 y 21 dígitos antes del punto, `000000000000000000001`, `99999999999999999999.999999991`, un string de 10.001 decimales y una mantisa de 31 dígitos que vale 1 (D198). Las mantisas y los exponentes de diez millones de dígitos están solo en el test de Go;
  - **ronda 1 del review (D204 a D215):** `'1.0e-2147483648'`, un resto que termina en ceros (`'-1.0000000010'` y `'99999999999999999999.9999999910'`) y un exponente con ceros a la izquierda;
  - **ceros con escala** (`'0.000000000'`, `'0e-10001'`);
  - `ConvertsToDecimal` y `convert … to Decimal` de cada caso.
- **Presupuesto de recursos, en Go:** una mantisa de 10⁷ dígitos y un exponente compensatorio responden en menos de un segundo, con las asignaciones acotadas (`testing.AllocsPerRun` o `runtime.MemStats`).
- `TestAStringWithAHugeExponentIsAnsweredAtOnce` no cambia. El caso `'-1e-999999999'` pasa a citar a `'-1e-10001'` del corpus, que es la respuesta medida en Java.
- **La suite de fhirpath con v1.6.0** se corre como paso de validación. El resultado va al issue.

## Mutaciones (`scripts/mutations/009-decimal.sh`, 26-10-2026)

Se cazan las 23. La primera corrida tenía 19 y tardó 76 s, con 0,5 GB de RSS máximo:
- **gramática:**
  - acepta U+2212;
  - acepta un signo después del punto;
  - lee un dígito de ancho completo como el ASCII;
  - acepta un exponente sin dígitos;
  - acepta algo después del exponente;
  - lee el exponente en 64 bits;
  - lee el exponente entero, con sus ceros a la izquierda (lo caza el test de presupuesto; ronda 1 del review);
- **magnitud:**
  - el valor se lee primero con `NewFromString`;
  - el último dígito guardado queda a la potencia opuesta;
  - se ignora el resto que queda después de la escala;
  - el piso de un negativo no baja un paso;
  - el máximo con resto se acepta;
  - el resto se decide por el último dígito (ronda 1 del review);
  - el cero se escribe con un decimal;
  - se guardan todos los dígitos de la mantisa (lo caza el test de presupuesto);
- **string mal formado:**
  - `convert` falla;
  - `ToDecimal` falla;
  - `cast` da null (se mueve `types`);
- **oráculo:**
  - una divergencia deliberada se borra del archivo;
  - el plan deja de nombrar una divergencia deliberada;
  - un defecto propio espera la respuesta del motor;
  - el área queda fuera del manifiesto;
  - el área tiene otro número de definiciones.

El primer `make check` dio el test de presupuesto en rojo con `-race`: la expresión regular de la gramática tardaba 1,6 s sobre 10 MB, contra 0,12 s sin `-race`. Se reemplazó por un lector byte a byte de la misma gramática, que tarda 0,06 s con `-race`, y las mutaciones de la gramática pasaron a apuntar al lector.

La mutación «límite de 21 dígitos en vez de 20» que estaba prevista se descartó por equivalente: la comparación con el máximo Decimal da null en el mismo caso. En su lugar está «el máximo con resto».

## Clasificación de `decimalstrings` (216 definiciones, 100 divergencias)

**Deliberadas:**
- **Decisión A, dígitos ASCII:** D51, D53, D54, D56, D57 y D59 (Java). Java lee `'１'`, `'٣'` y `'１.５'`.
- **Decisión C, el exponente escrito se lee dentro de un `int32`:** D150, D152, D153, D155, D156, D157, D158, D159, D160, D161, D162, D163, D164, D204 y D206 (Java o cql-execution). Java acota la escala, no el exponente.
  - `'1e-2147483648'` está dentro de `int32` y da 0.00000000. Java da null.
  - `'0e2147483648'`, `'0e99999999999'` y `'1e-99999999999'` están fuera y dan null. Alguna referencia los lee.

**Defectos de la referencia:**
- **`ConvertsToDecimal` de Java da true donde su `ToDecimal` da null o falla:** D61, D64, D115, D118, D121, D124, D142, D145, D157, D160, D163, D169, D175, D178 y D211.
- **Java falla con «BigInteger would overflow»** en `'1e-2147483647'`: D147 y D149.
- **cql-execution lee el prefijo más largo que es un número, como `parseFloat`:** D48 a D50, D60 a D65, D81 a D86, D96 a D101, D105 a D107 y D111 a D113.
- **cql-execution lee el string como un float binario, sin llevarlo a la escala:** D132, D134, D153, D155, D165, D167, D171, D173 a D180, D182, D183, D185, D189, D191, D201, D203, D207, D209, D210, D211 y D212.

**Defecto propio:** un string sin dígito antes del punto (`'.5'`, `'+.5'`, `'-.5'`) no sigue el formato. Java da null, y el motor lo lee desde #134, también en main. Son D12, D14, D15, D17, D18 y D20 (Java). Va al backlog.

**`cast` no cambia, salvo en lo que deciden A y C.** `cast 'a' as Decimal` sigue siendo error, como en main y en Java. `cast '.-5' as Decimal` pasa de -0.05 a error, porque ya no es un Decimal (decisión A). `DecimalFromString` devuelve `ErrNotADecimal`, y solo `ToDecimal` y `convert … to Decimal` lo convierten en null.
- Una primera versión hacía fallar `cast` cada vez que la conversión no daba valor. Eso movía nueve respuestas de `types` y `fhir` hacia Java (D449 de `types` y D16, D52, D540, D627, D742, D744, D820 y D825 de `fhir`).
- Son defectos anteriores: `cast` convierte, según el backlog de #139. Quedan fuera del alcance.

## Ronda adversarial (Fable y Codex, 26-10-2026)

| # | Hallazgo | Revisor | Resolución |
|---|---|---|---|
| 1 | La gramática no es la de v1.4.0: `.+5` y los exponentes fuera de `int32` | los dos | Aceptado: alcance 2 y 3, decisión C |
| 2 | fhirpath con v1.6.0 falla `TestDecimalRange` | Fable | Aceptado y medido: el motor no cambia (sonda `fq`); decisión D |
| 3 | La decisión A no es «como Java»: Java acepta dígitos Unicode | Fable | Aceptado: es una divergencia deliberada, nombrada |
| 4 | La gramática es la de BigDecimal, no la de la especificación | Fable | Aceptado: el alcance 2 lo dice |
| 5 | Ceros con escala divergen de Java | Fable | Medido igual en main y en la rama: al backlog, como defecto propio |
| 6 | Costo de memoria y tiempo con mantisas enormes y exponentes compensatorios | los dos | Aceptado: alcance 3 con memoria acotada, test de presupuesto y mutación |
| 7 | Hay que decir por qué no es workaround, con la cita | Fable | Aceptado: alcance 3 cita `decimal.go:103-116` |
| 8 | Dos políticas: `funcs.ToDecimal` contra `convertToType` | Fable | Aceptado: alcance 4. `funcs.ToDecimal` no tiene llamadores y no se toca |
| 9 | Más defectos de Java en `ConvertsToDecimal` | Fable | Aceptado: se registran como defectos de la referencia |
| 10 | La afirmación «`0e10001` en JSON pasa a sin valor» es falsa | Codex | Aceptado: retirada. El cambio real es el punto 2 |
| 11 | Fronteras y mutaciones que faltaban, como la de «sin gramática acepta −5», que con `big.Int` no se garantiza | los dos | Aceptado: tests y mutaciones reescritos |

## Review, ronda 1 (Fable y Codex, 26-10-2026)

Codex no encontró discrepancias en 300.000 casos contra un oráculo independiente (regex y `big.Rat`) ni en 1.885.863 ejecuciones de fuzzing. Fable no encontró panics ni caminos de `ErrNotADecimal` sin manejar, y el lint dio 0 issues.

| # | Hallazgo | Revisor | Severidad | Resolución |
|---|---|---|---|---|
| 1 | `strconv.ParseInt` copia en su error un exponente de diez millones de dígitos | Codex | media | Confirmado (10 MB asignados). `exponentOf` quita los ceros y no lee más de diez dígitos. Test de presupuesto y mutación |
| 2 | Ningún test ve un resto que termina en cero: «el último dígito decide» sobrevivía | Codex | baja | Confirmado. Agregados D207 y D210, y la mutación |
| 3 | El comentario y el plan dicen que Java lee `'.5'`; da null | Fable | media | Confirmado. Textos corregidos; la respuesta se conserva como en main (defecto propio, backlog 1c23), para no ampliar en el review |
| 4 | Textos que no coinciden con lo medido: 29 dígitos, la mantisa de un millón, 19 mutaciones, `cast '.-5'` | Fable | baja | Corregidos |
| 5 | El presupuesto tolera 10 MB | Fable | baja | Cota fija de 4 KB (se midieron 208 bytes) |
| 6 | Java acota la escala, no el exponente: `'1.0e-2147483648'` | Fable | baja | Medido en Java (null). Agregado como D204 y clasificado con la decisión C |
| 7 | La evidencia de D147 cita un formato que no tiene exponente | Fable | baja | Reescrita |
| 8 | Los ceros con escala no tienen fila: el comparador los iguala | Fable | baja | Anotado en el plan y en el backlog |
| 9 | D12 a D20 no dicen que el `ConvertsToDecimal` de Java da true | Fable | baja | Evidencia ampliada |
| 10 | Estilo: `a \|\| b && c`, la doc de `ErrNotADecimal` y `max(first, last+1)` | Fable | baja | Paréntesis, «reports» y un comentario |

## Review, ronda 2 (26-10-2026)

- **Codex**, sin hallazgos:
  - el oráculo `big.Rat` coincide en 300.000 casos;
  - 2.001.049 ejecuciones de fuzzing, sin discrepancias;
  - se cazan las 23 mutaciones del script y 6 propias;
  - rechazar un exponente de diez millones de dígitos asigna 112 bytes.
- **Fable** verificó las diez resoluciones de la ronda 1 y dejó dos hallazgos de severidad baja, los dos corregidos:
  - el plan decía «29 dígitos» en un lugar;
  - la lista de defectos de Java en «Fuera del alcance» estaba incompleta.

**Converge en la ronda 2:** no queda ningún hallazgo de severidad media o mayor.

## Cierre (26-10-2026)

- **Diferencial seed 8008, main contra la rama**, después de las correcciones del review: los reportes de las nueve áreas son idénticos byte a byte.
- **`sh scripts/references.sh check`:** dos corridas idénticas, 37 archivos, y 0 archivos difieren del repo.
- **`make check`:** en verde, con 359 mutaciones cazadas de 001 a 009, 23 de ellas del 009.
- **Evidencia:** [docs/evidence/2026-10-26.md](../evidence/2026-10-26.md).
- **Pendientes:**
  - el issue para fhirpath (decisión D), publicado como [gofhir/fhirpath#235](https://github.com/gofhir/fhirpath/issues/235), relacionado con #232;
  - el commit y el PR, que esperan la aprobación del usuario.

## CI de #165 (27-10-2026)

El primer CI falló en el test de presupuesto, en dos jobs:
- **Test** corre con `-race -covermode=atomic`. Con esa cobertura, cada vuelta de un bucle byte a byte del paquete es un contador atómico, y diez millones de dígitos tardaron más de 5 s; en local, sin cobertura, eran 0,06 s.
  - Los recorridos pasaron a `strings.TrimLeft` con conjuntos ASCII, que la cobertura no instrumenta.
  - Con las mismas opciones que el CI, ahora tarda 0,21 s.
- **Zones** midió 6688 bytes contra la cota de 4 KB. `TotalAlloc` cuenta todo el proceso, así que la cota pasó a 64 KiB, todavía muy por debajo de una copia de la mantisa (10 MB).

Las mutaciones (23) se siguen cazando todas, y la diferencial sigue idéntica a main.

## Dependencias

Ninguna nueva. `math/big` y `strconv` son de la biblioteca estándar.
