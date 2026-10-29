# 013: null donde la especificación dice null

Backlog 1c25, grupo G2. Rama: `fix/null-where-due-1c25`. Estado: paso 9, cerrado; el review convergió en la ronda 3 (29-10-2026). Sin commit. Plan aprobado el 28-10-2026.

## Por qué

En la suite al día (cqframework/cql-tests f724ee4, plan 012), nueve tests piden null y este motor da un error: DivideQuantityByZero, Exp1000, Exp1000D, Ln0, LnNeg0, PredecessorUnderflowDt, PredecessorUnderflowT, SuccessorOverflowDt y SuccessorOverflowT. Ocho de ellos eran `invalid="true"` en la suite anterior (727219f), y el motor siguió a la suite. En f724ee4 la suite sigue a la especificación.

## Revisión previa (sondas del 28-10-2026, scratchpad `r013/`)

**`gofhir/server`** no tiene código propio aquí: consume `gofhir/cql`. No hay nada que tomar.

**La especificación.** Las citas son de CQL 1.5.3, Appendix B (`https://cql.hl7.org/N1A/09-b-cqlreference.html`). La edición actual y el build dicen lo mismo de Successor y Predecessor (sonda `r013/spec/`):

- Arithmetic Operators: «In general, operations that cause arithmetic overflow or underflow, or otherwise cannot be performed (such as division by 0) will result in null, rather than a run-time error».
- Successor: «If the argument is already the maximum value for the type, a null is returned», y «If the result of the operation cannot be represented (i.e. would result in an overflow), the result is null». Predecessor dice lo mismo con el mínimo y el underflow.
- Exp y Ln: «If the result of the operation cannot be represented, the result is null».
- Divide: «If the result of the operation cannot be represented, or the right argument is 0, the result is null».
- El máximo de Decimal es `maximum Decimal`, 99999999999999999999.99999999, que el motor ya tiene (`funcs/math_ops.go`, `maxDecimal`).

**De dónde viene el error.** El comentario de `types.ErrPastRange` dice «The specification makes it an error, not null». Ninguna edición lo dice: la regla salió de la suite anterior y de Java. El oráculo de enteros de #135 que citan las filas de `integer.divergences.tsv` no está en el repo, porque quedó en el scratchpad de esa sesión.

**Las referencias** (Java 5.4.0 y cql-execution 3.3.2, sondas `r013/in/probe.cql` y `probe2.cql`, 85 expresiones):

| Expresión | este motor | Java | cql-execution | especificación |
|---|---|---|---|---|
| `successor of DateTime(9999, 12, 31, 23, 59, 59, 999)`, `successor of maximum Date`, `predecessor of @0001`, `successor of @T23`, y otras 14 de fecha y hora | error | null | null | null |
| `predecessor of DateTime(1)`, `predecessor of minimum DateTime` | error | null | `0000-12-31T…`, el año 0 | null |
| `successor of maximum Integer`, `successor of 2147483647`, `successor of maximum Long` | error | error (TypeOverflow) | null; Long: error | null |
| `successor of maximum Decimal`, `successor of 99999999999999999999.99999999 'mg'` | 10^20, fuera del tipo | error (TypeOverflow) | null | null |
| `Exp(1000)`, `Exp(1000.0)`, `Exp(1000) is null`, `Coalesce(Ln(0), 7)` | error | null, true, 7 | null, true, 7 | null, true, 7 |
| `Exp(46.06)`, `Exp(50)`, `Exp(100)`, `Exp(709)` (más que `maximum Decimal`) | el valor | el valor | null | null |
| `Exp(46)` (menos que `maximum Decimal`) | el valor | el valor | null | el valor |
| `Ln(0)`, `Ln(0.0)`, `Ln(-0)`, `Ln(-0.0)` | error | null | null | null |
| `5.0 'g' / 0` | error | null | null | null |
| `5.0 'g' / 0 'g'`, `5 / 0`, `5 mod 0`, `5.0 'g' mod 0 'g'` | null | null | null | null |

**Las tres causas:**

1. `types.step` (`types/successor.go`) da error en el borde de cada tipo. En Integer y Long es `ErrPastRange`. En Date y DateTime es el `ErrInvalidTemporal` de fhirpath, que `AddDuration` devuelve fuera de los años 1 a 9999. En Time es un error propio. Decimal y Quantity no tienen tope, así que pasan de `maximum Decimal`. El evaluador (`evalSuccessorPredecessor` y `stepWithinInteger`) deja pasar el error, y para Integer de 32 bits fabrica el suyo.
2. `funcs.Exp` da error si el float es infinito, y no mira `maximum Decimal`. `funcs.Ln` da error en cero.
3. `Quantity / número` llama a `fptypes.Quantity.Divide`, que devuelve `ErrDivideByZero`, sin mirar el cero antes, como hacen las otras ramas de `/` (`eval/evaluator.go`).

Fhirpath acierta en las tres. Una Date fuera de los años 1 a 9999 no existe en FHIRPath, y devolver error es su contrato. El null es de CQL y le toca al operador de CQL.

## Alcance

Ajustado tras la ronda adversarial (ver abajo).

- **Successor y Predecessor en el borde dan null:**
  - Integer certero, con los 32 bits del tipo estático, y los 64 bits;
  - Long;
  - Decimal y Quantity, cuando el resultado pasa de ±`maximum Decimal`;
  - Date, DateTime y Time.
- **`start of` y `end of` de un límite abierto en el borde también dan null.** Start es «the successor of the low value» (End, el predecessor del high), y el successor del máximo es null. Hoy `start of Interval(2147483647, null]` da `2147483648`, fuera del tipo. Es la decisión C.
- **Cómo:**
  - `types.step` marca con `ErrPastRange` el paso fuera del calendario. Envuelve por `errors.Is` el `ErrInvalidTemporal` de fhirpath, que `AddDuration(1, unidad de la precisión)` solo devuelve por el rango de años (fhirpath `types/date.go:237`, `types/datetime.go:520`). Time recibe el mismo error. Se borran las guardas `result.Year()`, que son código muerto.
  - Decimal y Quantity no cambian en `types`. El tope de `maximum Decimal` lo pone el operador, sobre el resultado. Así, `meets`, `except` y `collapse` siguen recibiendo lo mismo que hoy. Si `types` diera error, `Interval[maximum Decimal, maximum Decimal] meets before` el mismo intervalo pasaría de false a true (Codex, hallazgo 1).
  - El evaluador lleva el resultado de los cuatro operadores (`successor of`, `predecessor of`, `start of`, `end of`) por una sola función:
    - `ErrPastRange` da null;
    - un Decimal o una Quantity fuera de ±`maximum Decimal` da null;
    - un Integer certero fuera de 32 bits da null.
    
    `stepWithinInteger` deja de fabricar un error.
- **Exp da null** si el resultado es infinito, NaN o pasa de ±`maximum Decimal`.
- **Ln da null en cero** (`0`, `0.0`, `-0`, `-0.0`).
- **`Quantity / número` da null si el número es cero.**
- **Tests que hoy fijan el error y cambian:**
  - `engine_integer_overflow_test.go:63` y `:157`;
  - `engine_interval_equality_test.go:102`;
  - `engine_diagnostics_test.go:46`, que usa `Exp(1000)` y `Ln(0)` como error de diagnóstico. Pasa a usar un error que sigue existiendo (`point from` de un intervalo que no es unitario), y conserva sus verificaciones de ubicación.
- **El comentario de `ErrPastRange`** se corrige: la especificación dice null.

## Fuera del alcance (al backlog, con la evidencia de la ronda)

- **El borde de 32 bits sin Plan o con un Integer no certero** (Codex 2, Fable 2):
  - `Successor` importado de ELM da `2147483648`, porque `pastInteger` exige Plan (`eval/integer_range.go:66`);
  - `successor of (if true then 2147483647 else 0)` y `successor of First({2147483647})` dan lo mismo, porque `certainInteger` no los reconoce.
  
  Es la deuda que dejó #135: la raíz es que `sema` marque los tipos inferidos desde un Unknown. Un test la documenta.
- **`+`, `-` y `*` de Decimal que pasan de `maximum Decimal`** (Fable 4): `maximum Decimal + 0.00000001` da 10^20 en el motor y en Java, y null en cql-execution. La especificación (Add, Subtract, Multiply) dice null. Tras el 013, `successor of maximum Decimal` dará null y la suma no. Va como frente propio, porque toca toda la aritmética Decimal.
- **`width of` da la vuelta** (Fable 5): `width of Interval[minimum Long, maximum Long]` da `-1`, e `Interval[minimum Integer, maximum Integer]` da `4294967295`.
- **`except` falla lejos del borde** (Fable 3): `Interval[1, 5] except Interval(1, 5]` da `Interval[6, 5]`, y debe ser `Interval[1, 1]`. La rama `cmpLow <= 0` de `funcs/interval.go:251` trata b como si cubriera el extremo bajo.
- **Quantity** (Fable 8): `maximum Quantity` da null, y `successor of 5 'g'` da `5.00000001 'g'`; la especificación da +1 para una cantidad entera.
- **`Quantity mod número` y `Quantity div número`** dan error («cannot perform arithmetic on …»). Java convierte el número en Quantity y responde `5.0 'g' mod 2` = `1.00000000 'g'`. Es la conversión implícita, no el cero.
- **Fecha más una duración que pasa del año 9999** (`@9999-12-31 + 1 day`): el motor y Java dan error, y cql-execution null. Add de Date y Time no dice nada del desborde.

## Decisiones del usuario (28-10-2026)

A, B y C: null, como recomendaba el plan. El plan quedó aprobado el 28-10-2026.

## Las decisiones, como se plantearon

- **A. Successor y Predecessor de Integer, Long, Decimal y Quantity en el borde.** La especificación y cql-execution dicen null (en Long, cql-execution da error). Java da error (TypeOverflow). Recomendación: null, como la especificación (ADR 001). Las filas de Java quedan como defecto de la referencia, con la cita.
- **C. `start of` y `end of` de un límite abierto en el borde.** La composición de la especificación da null (Start es el successor del low). Java da error (TypeOverflow) y cql-execution también. Recomendación: null.
- **B. Exp mayor que `maximum Decimal`.** La especificación y cql-execution dicen null, y Java da el valor. El oráculo de escala del repo ya pide null en `arithmetic.own-defects.tsv` (D4, D379, D427, D699, D780). Recomendación: null.

## Diferencias deliberadas

Ninguna nueva. En `limitsexp`, tres filas de Java son la decisión del ADR 002 (`docs/adr/002-escala-decimal.md`): Exp y Ln se calculan en binary64 y no se recortan a la escala de Decimal. Java da los dígitos de BigDecimal:

- D93, `Exp(46.05)`;
- D94, `Exp(46)`;
- D106, `Ln(0.00000001)`.

Las filas donde Java difiere por el borde quedan como defecto de la referencia, con la cita de la especificación (decisiones A, B y C).

## Encontrado en los tests (28-10-2026)

- **`meets` en el borde:** `Interval[@T23:59:59.999, @T23:59:59.999] meets before` el mismo intervalo daba true, ya en main. Pasa igual con `@9999-12-31`, y con Decimal en la rama (hallazgo 1 de Codex). Si no hay successor de a.High, `intervalEndMeetsStart` (`funcs/interval.go`) comparaba a.High con b.Low, y dos límites iguales se solapan. Ahora da false cuando el tipo tiene successor y el paso no existe; un tipo sin successor (String) sigue leyéndose por igualdad. Es el mismo borde y una línea, así que entra en el 013. Cambia también en Date y Long (de true a false, review 1). Java da false en todos. cql-execution da false en Time y Date, y true en Decimal y Quantity (D75, D76 y D82 de `limits`, defectos de la referencia).
- **Un DateTime sin offset no se compara con el de Java:** el comparador lee un valor temporal por su texto, y el motor escribe `DateTime(…)` sin offset. En `limits`, D47 lleva offset explícito (`0.0`). No es de este frente.

## Criterio de salida

1. Los nueve tests de G2 pasan y salen de `knownFailures`. La lista se limpia sola: si uno pasa y sigue en la lista, el test falla.
2. Un área nueva de snapshots, `limits`, con las expresiones de las sondas y sus variantes, y las respuestas de Java y cql-execution (`sh scripts/references.sh write`, dos corridas idénticas). No queda ningún defecto propio de este alcance. Cada divergencia queda clasificada con la cita.
3. En `arithmetic.own-defects.tsv` desaparecen las filas de Exp y Ln: D4, D7, D17, D114, D115, D212, D379, D427, D573, D699 y D780.
4. En `integer`, las diez definiciones de successor y predecessor en el borde pasan de error a null. Con eso, las diez filas de cql-execution desaparecen, porque coincide. Las diez de Java quedan como defecto de la referencia: nueve dan TypeOverflow o TypeUnderflow, y D910 da la vuelta (`Long -3037000500`). D205, D328, D684 y D906 no cambian. En `suitegaps`, las nueve filas dejan de ser defecto propio.
5. La diferencial masiva (semilla 8008, corpus d008, las 9 áreas) contra main solo difiere en definiciones con estos operadores (successor, predecessor, start, end, Exp, Ln, `/`), y cada diferencia queda listada en este plan. Antes y después, con `=`, `meets`, `except` y `collapse` en el borde Decimal: no deben cambiar.
6. `scripts/mutations/013-null.sh` atrapa cada parte del arreglo.
7. `make check` en verde.

## Tests

- **Conformance:** los nueve de G2, sin entrada en `knownFailures`.
- **`limits` (snapshot):**
  - el borde de cada tipo, por literal y por `maximum` y `minimum`;
  - DateTime con offset (`+14`, `-12`) y con cada precisión;
  - Time con cada precisión;
  - Decimal un paso antes del borde, que es un valor;
  - Exp un poco antes y un poco después de `maximum Decimal`, y Exp de un negativo grande, que da 0;
  - Ln de los cuatro ceros y de un negativo;
  - `Quantity / 0` y `/ 0.0`, y los ya null como control;
  - el null dentro de otra expresión, que distingue null de error: `Exp(1000) is null`, `Coalesce(Ln(0), 7)`, `Coalesce(successor of maximum Integer, 1)`, `(5.0 'g' / 0) + 1 'g'`;
  - `start of` y `end of` con el límite abierto en el borde y el otro null, en Integer, Long, Decimal, Date, DateTime y Time;
  - `meets`, `except` y `collapse` con intervalos de borde Decimal y Quantity, coincidentes y adyacentes, cuya respuesta no cambia;
  - Exp(46.05), que es un valor, frente a Exp(46.06), que es null. Cerca del máximo, el último dígito difiere de Java (float frente a BigDecimal), y lo cubre la tolerancia del snapshot.
- **`types` (unitario):** `errors.Is(err, ErrPastRange)` en el borde de cada tipo, y el valor un paso antes.
- **Por ELM:** `Successor` importado llega al mismo operador (`elm/importer.go`). Un caso de DateTime lo fija, y otro documenta que el borde de 32 bits no llega sin Plan.

## Dependencias

Ninguna nueva.

## Ronda adversarial (Codex y Fable, 28-10-2026)

| # | Hallazgo | Quién | Resolución |
|---|---|---|---|
| 1 | Un error de Decimal en `types` cambia `meets before` en el borde de false a true (prototipo de Codex) | Codex | Aceptado: el tope de Decimal va en el operador, no en `types` |
| 2 | Start/End deben dar null por composición; hoy un Integer da 2147483648 | los dos | Aceptado: decisión C y alcance |
| 3 | El borde de 32 bits no llega por ELM ni a Integers no certeros | los dos | Fuera del alcance, como deuda de #135, con un test que la documenta |
| 4 | Tests existentes fijan el error | Codex | Aceptado: la lista está en el alcance |
| 5 | `maximum Long` falla en cql-execution antes que successor | Codex | Aceptado: en `limits`, la falla se atribuye al operando |
| 6 | El error nuevo en Decimal cambia `except`/`collapse`; `except` ya falla lejos del borde | Fable | Aceptado: el 1 lo evita, y el defecto de `except` va al backlog |
| 7 | `+`, `-` y `*` de Decimal desbordan sin null | Fable | Al backlog, como frente propio |
| 8 | `width of` da la vuelta | Fable | Al backlog |
| 9 | Envolver `ErrInvalidTemporal` es seguro solo porque la unidad sale de la precisión; las guardas de año son código muerto | Fable | Aceptado: `errors.Is`, comentario y se borran las guardas |
| 10 | Las cuentas de `integer` son 11 filas de 10 definiciones; defectos de cql-execution a clasificar | Fable | Aceptado: criterio 4 corregido, se clasifican en `limits` |
| 11 | Quantity: `maximum Quantity` null y el paso de una cantidad entera | Fable | Al backlog |
| 12 | Faltan tests de null dentro de otra expresión y de Exp cerca del máximo | Fable | Aceptado: en la lista de tests |

## Implementación (28-10-2026)

- **`types/successor.go`:** `calendarStep` envuelve el `ErrInvalidTemporal` de fhirpath en `ErrPastRange`, y Time da el mismo error (`stepError`). Las guardas `result.Year()` se fueron. Decimal y Quantity no cambian.
- **`eval/integer_range.go`:** `stepWithinItsType` reemplaza a `stepWithinInteger` y sirve a `successor of`, `predecessor of`, `start of` y `end of` (`eval/evaluator.go`). `certainInteger` lee el límite de `start of` y `end of` de un intervalo escrito.
- **`funcs/interval.go`:** `intervalEndMeetsStart` da false si el tipo tiene successor y el paso no existe.
- **`funcs/math_ops.go`:** Exp da null si el resultado es infinito, NaN o pasa de `maximum Decimal`, y Ln da null en cero.
- **`eval/evaluator.go`:** `Quantity / número` da null si el número es cero.
- **Tests:**
  - `engine_type_edge_test.go`, con los casos de la especificación;
  - `types/successor_test.go`;
  - se migraron `engine_integer_overflow_test.go`, `engine_interval_equality_test.go` (`TestSteppingPastTheLimitIsNull`) y `engine_diagnostics_test.go`.
  
  En `engine_runtime_reference_test.go`, `IntegerSuccessorOfMaximum` y `LongSuccessorOfMaximum` pasan de la lista de cql-execution a la de Java, y `runtime.cql` deja de marcarlas como control. `arithmeticAgreedOutsideTheSpecification` queda vacía: D4 era su única entrada.
- **La suite:** los nueve tests de G2 salen de `knownFailures`, y sus definiciones de `suitegaps.cql`, que pasa a 48.
- **`limits`:** 119 definiciones al principio; en la review 1 se separaron en `limits` (96, comparadas exactamente) y `limitsexp` (23, Exp y Ln, con la tolerancia del float). D69 (`start of Interval(@9999-12-31T23:59:59.999, null]`) se quitó, porque cql-execution responde un Uncertainty que el arnés no codifica (`TestTheReferencesHoldNothingUnencoded`); el caso queda en `engine_type_edge_test.go`. D47 lleva offset `-3.0`: el comparador lee un valor temporal por su texto, y Java escribe `+00:00` donde el motor escribe `Z`. La clasificación: 54 defectos de las referencias, 3 deliberadas (las de «Diferencias deliberadas») y 2 defectos propios. Estos dos son D83, `except` en el borde de Decimal; el defecto es anterior a este plan, y va al backlog.
- **`arithmetic`:** el oráculo (`-classify-arithmetic`) da como defecto de Java las filas de Exp que pasan de `maximum Decimal` (D4, D379, D427, D699, D780). Las once filas de `arithmetic.own-defects.tsv` desaparecen.

## Mutaciones (`scripts/mutations/013-null.sh`, 28-10-2026)

20 mutaciones, todas atrapadas. Cada una cae en el test que nombra:

- `ErrPastRange` dejado como error, en la suite y en el área;
- un Decimal o una Quantity fuera de `maximum Decimal` que se conservan;
- el borde de 32 bits sin aplicar;
- el rechazo del calendario sin leer como el borde, en `types` y en la suite;
- el borde del reloj con un error propio;
- `start of` sin el guardián, y un Integer de `start of` no certero;
- `meets` leído por igualdad en el borde;
- Exp pasado de `maximum Decimal`, en los tests y en `arithmetic`;
- Exp infinito como error;
- Ln de cero como error;
- `Quantity / 0` como error;
- el área fuera del manifiesto y fuera de `references.sh`;
- una deliberada que el plan no nombra;
- el TypeOverflow de Java tomado por oráculo.

## Diferencial masiva (semilla 8008, corpus d008, 28-10-2026)

Main contra la rama: siete áreas idénticas. En `arithmetic` y en `integer` se movieron 35 definiciones, y todas llevan uno de estos operadores (`scratchpad/r013/diff-moved.txt`):

- **`arithmetic`, 22:**
  - 17 iban de error a null, que es lo que responde Java: Exp(1000) y Ln(0) dentro de otras expresiones;
  - 5 iban de un valor fuera de `maximum Decimal` a null (decisión B): D18, D152, D756, D892 y D1116.
- **`integer`, 13:** successor y predecessor en el borde pasan del error, el mismo de Java, a null (decisión A).

## Review, ronda 1 (Fable y Codex, 29-10-2026)

| # | Hallazgo | Quién | Severidad | Resolución |
|---|---|---|---|---|
| 1 | `start of` y `end of` de un intervalo nombrado (`define I: Interval(2147483647, null]`) dan 2147483648 | los dos | media | Confirmado con un test que fallaba (`TestTheStartOfANamedIntervalAtTheEdgeIsNull`). `certainBoundary` sigue una definición de la librería hasta su intervalo. Un operando de función no cuenta como certero, igual que en #135: queda documentado en el test y en el backlog |
| 2 | La tolerancia del float en todo `limits` esconde un paso de Decimal equivocado cerca del borde | Codex | media | Confirmado con la mutación de Codex, que ahora está en `013-null.sh` («a Decimal step that stays near the least»). `limits` se compara exactamente, y Exp y Ln pasan a `limitsexp` con los mismos D |
| 3 | El test de intervalos dice cubrir `except` y `collapse`, y solo prueba `meets`; su nombre dice Decimal | los dos | baja | Ahora es `TestMeetsAtTheEdgeOfAType`, con `collapse`, Long y Date. `except` en el borde es defectuoso (D83) y queda al backlog |
| 4 | Las evidencias de D65 (cql-execution) y D7 (Java) atribuyen la falla al paso, y es del operando | los dos | baja | Corregidas |
| 5 | El plan decía que cql-execution da true en Time; da false | Fable | baja | Corregido |
| 6 | Ramas sin test: un error que no es del borde (`successor of 'a'`) y la igualdad de un tipo sin successor en `meets` | Fable | baja | Tests nuevos (`TestAStepThatIsNoStepStaysAnError` y la fila de String) |
| 7 | Un comentario sin reenvolver en `engine_integer_overflow_test.go` | Fable | baja | Corregido |

**Cambios de comportamiento que el plan no nombraba** (Fable, verificados): `{ successor of maximum Integer }` da `{null}`, y `Count` de eso da 0; antes, error. Es la consecuencia directa de la decisión A.

## Review, ronda 2 (Fable y Codex, 29-10-2026)

| # | Hallazgo | Quién | Severidad | Resolución |
|---|---|---|---|---|
| 1 | Las claves del memo (`"low of "+nombre`, `"high of "+nombre` y, desde #135, `"list "+nombre`) pueden ser el nombre de una definición: con `define "low of I": start of I`, `successor of "low of I"` da 2147483648. Fable dio otro ejemplo, en que la respuesta depende del orden de los operandos | Codex (media), Fable (baja) | media | Confirmado con tres casos que fallaban en `TestTheStartOfANamedIntervalAtTheEdgeIsNull`. La clave ahora es `settled{what, name}`, con un tipo distinto por pregunta. Cierra también la de `list`, que era anterior: el mismo arreglo, un solo mecanismo. Mutación nueva en `013-null.sh` |

Fable verificó, sin hallazgos:
- **Los ciclos:** `sema` los rechaza antes, y `walking` daría false igual.
- **Los Long:** un intervalo Long, en el lugar o nombrado, sigue dando 2147483648.
- **Las rutas no certeras:** las documentadas.
- **La separación de `limits` y `limitsexp`.**
- **Las divergencias.**
- **Los scripts 003, 007 y 012.**
- **Las zonas Lord_Howe, Kiritimati y Santiago.**

## Review, ronda 3 (Fable y Codex, 29-10-2026): converge

Ninguno deja hallazgos de severidad media o mayor.

- **Fable** probó 20 casos de nombres de definición que coinciden con lo que el memo decide, en los dos órdenes. Con las claves de texto fallan 5 de ellos, más las tres filas nuevas; con `settled` pasan todos.
- **Codex** inyectó la colisión por separado en el low, el high y la lista, y cada una falla en su fila. También corrió `go test -race ./...` y seis zonas.
- **Queda una nota de Fable:** en `certainBoundary`, la clave se asigna y luego se sobrescribe. No cambia nada, y se deja así.

## Cierre (29-10-2026)

- `make check` en verde, con las mutaciones de `scripts/mutations/` (`013-null.sh` incluido), lint y zonas. El último código lo probó la corrida posterior a la ronda 2; después solo cambió este plan.
- La diferencial contra main (semilla 8008) mueve solo las 35 definiciones de arriba.
- **Lo que queda,** en el backlog 1c25 de `docs/plans/README.md`, «Lo que el 013 dejó para después».
