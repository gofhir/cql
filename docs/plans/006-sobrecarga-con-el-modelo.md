# 006: la sobrecarga de `includes`, `included in` y `during` con el modelo

Backlog 1c11. Rama `fix/membership-model-conversion`.

## Por qué

Con un null de un lado, el 005 elige la sobrecarga de `includes`, `included in` y `during` por los tipos estáticos (`membershipOverload`, `eval/evaluator.go`). Esa elección solo ve las conversiones del sistema (`sema.ImplicitlyConverts`). Cuando un lado necesita la conversión del modelo FHIR para encontrarse con el otro, no se elige ninguna sobrecarga, y la respuesta sigue siendo la de antes del 005: null, donde la especificación dice false.

Es la frase que escriben las medidas:

| Expresión | Este motor | Java 3.26.0 | Especificación |
|---|---|---|---|
| `FHIR.dateTime { value: @2020-01-02T00:00:00 } during (null as FHIR.Period)` | null | false | false (Interval In, segundo argumento null) |
| `O.effective during E.period`, sin period (medido por Fable en la ronda 2 del 005, con un bundle) | null | false | false |
| `Patient.name.given includes (null as String)` | null | false | false (List Contains) |
| `Patient.gender included in { 'male', 'female' }`, sin gender | null | false | false (List In) |
| `Patient.gender in { 'male', 'female' }` | false | false | false |

La última fila es la misma pregunta con otra palabra, y ya da false: `in` no necesita elegir sobrecarga. Medido el 22-10-2026, cada definición sola en su librería, con un Patient sin gender ni name (sonda en el scratchpad; los casos definitivos van al repo).

**Una que no es pregunta:** `Patient.birthDate during (null as FHIR.Period)` da null en Java porque el traductor la rechaza: «Could not resolve call to operator IncludedIn with signature (FHIR.date,FHIR.Period)». Haría falta encadenar dos conversiones, `FHIR.date`→Date y Date→DateTime. Este motor la acepta y da null. Que `sema` la acepte es otro frente: queda fuera, y entra al backlog si el corpus lo confirma.

## Revisión previa

- **`gofhir/server` (congelado):** solo traduce los retrieves (`internal/cqlpushdown`); no evalúa la pertenencia. **Descartar:** no hay nada que tomar.
- **Especificación:** CQL 1.5.3, List Operators (In, Contains, Includes, Included In) e Interval Operators (In, Contains, Includes, Included In, During). La resolución de sobrecargas con conversiones implícitas está en el Developer's Guide, «Implicit Conversions», y en el Author's Guide, «Conversion Precedence».
- **Java 3.26.0:** el traductor resuelve la sobrecarga antes de escribir el ELM, con el modelo: escribe `In`/`Contains` para la del elemento e `IncludedIn`/`Includes` para la lista-lista, con la llamada a FHIRHelpers en el operando. Su elección depende de las definiciones anteriores (backlog 1c13, #435 del traductor). Por eso los casos se traducen solos.
- **cql-execution 3.3.2:** evalúa el ELM de Java; no resuelve nada por su cuenta.
- **Este motor:**
  - `sema` tiene el modelo (`checker.model`) y `Convertible(from, to, model)` (`sema/convert.go:118`), que ya encuentra las conversiones del modelo. `inferTiming` (`sema/infer.go:692`) solo registra la conversión de un operando a intervalo (`convertToInterval`).
  - El evaluador aplica las conversiones que registra `sema` (`evalAndConvert`, `eval/evaluator.go:471`). No tiene el modelo como `sema.Model`, solo `ModelInfo`.

## Diseño (cambio de arquitectura: requiere aprobación)

**Propuesta: que `sema` decida la sobrecarga y la registre; el evaluador solo la lee.**

- Nuevo campo en `sema.Result`: `MembershipOverloads map[ast.Expression]Overload`, con un accesor `OverloadOf(expr)`, como `ConversionFor`. Los valores son `ElementOverload` y `ListOverload`; sin entrada, no hay sobrecarga decidida.
- `inferTiming`, para `includes`, `included in` y `during` sin `properly` ni offset, calcula la sobrecarga con los tipos que ya infirió, con una regla propia, `meets(a, b)`. **No usa `Convertible`** (ronda adversarial): `Convertible` sube y baja listas (`String`↔`List<String>`), encadena una conversión del modelo con una del sistema, y acepta Any y Unknown anidados, y el traductor no hace ninguna de las tres. `meets(a, b)` es verdadero si uno de los dos llega al otro:
  - por igualdad;
  - por una conversión del sistema (`ImplicitlyConverts`);
  - o por **una sola** conversión del modelo cuyo destino es exactamente el otro tipo, sin encadenar otra.

  En listas e intervalos se aplica elemento a elemento, y nunca promueve ni baja una lista. Un Any o un Unknown en cualquier nivel deja la expresión sin decidir.
- La sobrecarga:
  - del elemento, si `meets(elemento, elemento o punto del contenedor)`;
  - si no, lista-lista, si `meets(lista, contenedor)`;
  - sin entrada en cualquier otro caso.
- **Una choice** de un lado: se usa la rama que `narrowChoiceOperand` ya registró, si la hay. Si no, cada rama se evalúa por separado, y se decide solo si todas las ramas que dan alguna sobrecarga dan la misma; si no, queda sin entrada. `O.effective` (choice) contra un intervalo ya se estrecha a su rama DateTime.
- El registro se salta en modo especulativo, como `c.convert` (`speculative > 0`).
- `membershipOverload` y `commonType` salen del evaluador. `nullMembership` lee `Plan.OverloadOf(n)`.
- `sema.ImplicitlyConverts` se queda: `meets` la usa.

**Por qué ahí y no en el evaluador:**
- `sema` ya tiene el modelo y el tipo de cada lado. El evaluador tendría que reconstruir un `sema.Model` desde `ModelInfo`.
- El comentario de `ConversionFor` dice para qué existe: «lets the evaluator apply a decision instead of making one». El 005 dejó una decisión de tipos en el evaluador; esto la devuelve a `sema`.

**Lo que se rechaza:**
- **Registrar la conversión del elemento en `Conversions`, como hace `expectTyped`:** el evaluador la aplicaría también cuando no hay null, y cambiaría el valor que recibe el operador, por ejemplo un `FHIR.string` convertido a String antes de comparar. Eso es otro frente, con su propio riesgo.
- **Pasarle el modelo al evaluador:** duplica la resolución en dos lugares.

## Aprobación

El usuario aprobó el plan ajustado el 22-10-2026, con el cambio de arquitectura.

## Alcance

- Dentro: la elección de la sobrecarga con un null para `includes`, `included in` y `during`, con conversiones del modelo, en los dos sentidos, también dentro de listas e intervalos.
- Fuera:
  - las respuestas sin null: la sobrecarga solo se lee cuando hay un null, como en el 005;
  - una lista FHIR cuyos elementos no se convierten: `List<FHIR.string>{ FHIR.string { value: null as String } } includes (null as String)` da true en Java, que convierte cada elemento a String. Aquí `contains` da false, porque la lista conserva el objeto, e `includes` dará lo mismo. Es un defecto de `in`/`contains` sobre listas FHIR, anterior a este incremento: backlog 1c14, con un test que documenta que `includes` y `contains` coinciden;
  - las expresiones que el traductor rechaza y `sema` acepta, como `Patient.birthDate during (null as FHIR.Period)` (dos conversiones encadenadas): con `meets` quedan sin decidir, en null. Que `sema` las rechace va al backlog (1c15);
  - el valor que tiene la rama de una choice que no es la pedida (`onAnotherBranch`, `eval/evaluator.go:6967`): ese null se decide antes y no cambia;
  - `{}` y `null` sin tipo (1c8);
  - la lista de listas sin null (1c10);
  - `Interval[1, 2] in (null as Interval<Integer>)` (1c12);
  - el ELM importado, donde el traductor ya eligió el nodo.

## Tests y datos

- **Los 76 casos del 005 siguen igual.** Es la prueba de que mover la decisión a `sema` no cambia nada sin modelo.
- **`TestMembershipWithAModelConvertedElementStaysUndecided` cambia de sentido:** pasa a esperar false en las filas de la tabla de arriba, y cambia de nombre.
- **Casos FHIR a mano,** con R4 (4.0.1), que es el único modelo que carga el motor. Los casos se prueban por el motor, que carga `model.LoadR4ModelInfo()`; `DefaultR4ModelInfo()` no trae conversiones. No hay un test de `sema` aparte (ronda 1 del review, Fable):
  - un elemento con un null: `gender`, `birthDate`, `Observation.value` como choice;
  - un contenedor con un null: `name.given`, `Encounter.period`, `Observation.effective` como choice;
  - los dos null;
  - los que deben quedar sin decidir: la lista-lista (`{ 's' } includes (null as List<String>)`, `Patient.name.given includes (null as List<String>)`), un Any anidado (`Patient.name.given includes (null as List<Any>)`) y las conversiones encadenadas (`birthDate`, `multipleBirth`);
  - pacientes con cero, uno y varios elementos.

  Por cada caso, **la sobrecarga esperada es el nodo que escribe el traductor de Java** (In/Contains o IncludedIn/Includes), con la expresión traducida sola en su librería (1c13). La respuesta esperada sale de la regla de la especificación para esa sobrecarga. El valor de Java no alcanza, porque da false también en la lista-lista con un null, que es su defecto conocido (ronda adversarial, Fable).
- **Corpus generado `fhirmembership`:** unos 200 casos sobre `testdata/snapshots/fhir-data.json`, con las respuestas de Java y de JS. **El oráculo no usa `sema` ni el modelo del motor:**
  - la sobrecarga sale del nodo del ELM de Java, que el arnés ya escribe con `-each`;
  - que un lado sea null sale de leer el JSON de los datos;
  - la respuesta sale de la regla de la especificación del 005.
- **Mutaciones:**
  - `sema` deja de registrar la sobrecarga;
  - registra la del elemento para todo;
  - `meets` acepta subir o bajar listas;
  - `meets` encadena dos conversiones;
  - la guarda de Any solo mira el nivel superior;
  - `meets` en un solo sentido;
  - registra en modo especulativo;
  - el evaluador ignora el registro;
  - el oráculo lee la sobrecarga del texto en vez del ELM.
- **Diferencial de todos los corpus, main contra la rama:** sin cambios fuera de las filas con un null y un tipo del modelo.

## Criterio de salida

- Las filas de la tabla de arriba dan false.
- Los 76 casos del 005 y todos los snapshots no cambian, salvo filas con null y modelo, cada una explicada.
- `fhirmembership` no deja divergencias sin clasificar.
- Todas las mutaciones se detectan.
- `make check` y `make evidence` en verde.

## Riesgos

- **Una regla distinta de la del traductor:** por eso `meets` es propia y estrecha, y por eso el oráculo del corpus compara el nodo del ELM, no solo el valor.
- **Una choice:** `narrowChoiceOperand` solo estrecha contra un intervalo, y escribe otro mapa (`narrowings`), no el tipo. La regla de las choices (arriba) lo lee de ahí, y se prueba cada rama y la ausencia.
- **Romper en silencio:** `sema` escribe el registro y el evaluador lo lee. Si uno cambia sin el otro, los 76 casos del 005 lo detectan, porque dejan de tener sobrecarga.

## Versión

`fix:`, v1.24.3. El campo nuevo de `sema.Result` es una adición: no rompe a nadie.

## Ronda adversarial (Codex y Fable, 22-10-2026)

| Hallazgo | Evidencia | Resolución |
|---|---|---|
| `Convertible` sube y baja listas, así que la sobrecarga del elemento le gana siempre a la lista-lista (alto, los dos) | `Convertible(List<String>, String)` es válido con costo 5 (`sema/convert.go:194`); Java escribe `Includes` para `{ 's' } includes (null as List<String>)` | **Aceptado:** `meets` no promueve ni baja listas |
| Una lista FHIR sin convertir sus elementos cambia la respuesta (alto, Codex) | `List<FHIR.string>{ FHIR.string { value: null as String } } includes (null as String)`: Java `Contains`, true. Aquí `contains` da false | **Fuera, al backlog (1c14):** es un defecto de `contains`, anterior. El 006 hace que `includes` coincida con `contains`, con un test que lo documenta |
| La premisa sobre las choices era falsa (medio, Codex) | `narrowChoiceOperand` solo estrecha contra intervalos; `First([Observation]).effective includes (null as DateTime)` da false en Java y null aquí | **Aceptado:** regla explícita para las choices, y casos por rama y por ausencia. El null de `onAnotherBranch` queda fuera y se nombra |
| La guarda de Any solo mira el nivel superior (medio, Fable) | `Patient.name.given includes (null as List<Any>)`: Java `Includes` | **Aceptado:** Any o Unknown en cualquier nivel dejan la expresión sin decidir |
| `Convertible` encadena una conversión del modelo con una del sistema (medio, Fable) | Java rechaza `Patient.birthDate during (null as FHIR.Period)` y `{ 1.0, 2.0 } includes Patient.multipleBirth` | **Aceptado:** una sola conversión del modelo. Las rechazadas quedan sin decidir; que `sema` las rechace va al backlog (1c15) |
| El valor de Java no distingue la sobrecarga (medio, Fable) | Java da false en la lista-lista con un null | **Aceptado:** el esperado es el nodo del ELM más la regla de la especificación |
| La independencia del oráculo (medio, Fable) | | **Aceptado:** el oráculo lee el ELM y el JSON de los datos, no `sema` |
| Tests de `sema` con el modelo sin conversiones; mutaciones que faltan (bajo, Fable) | `DefaultR4ModelInfo()` no tiene `FHIR.string→String` | **Aceptado:** `LoadR4ModelInfo()` y la lista de mutaciones ampliada |

## Implementación (22-10-2026, sin commit)

**`sema`** (`sema/overload.go`): `recordOverload`, llamada desde `inferTiming`, decide la sobrecarga y la registra en `Result.Overloads`; `OverloadOf` la lee. La regla es `meets`, en los dos sentidos, con `reaches`:
- igualdad, o una conversión del sistema (`ImplicitlyConverts`);
- un subtipo del modelo, o **una** conversión del modelo cuyo destino es exactamente el otro tipo;
- listas e intervalos, elemento a elemento.

Un contenedor del modelo ya llega convertido por `inferTimingOperand`; `containerForm` convierte la rama de una choice que es un tipo del modelo, como el Period de `Observation.effective`. Una choice se lee por sus ramas.

**El evaluador:** `nullMembership` lee `Plan.OverloadOf(n)`. Salieron `membershipOverload`, `commonType`, el tipo `overload` y la guarda de `properly` y offset, que ya aplica `sema`. `ImplicitlyConverts` volvió a ser escalar: `reaches` recorre las listas.

**Lo que cambió respecto del plan, con su evidencia:**
- **Any y Unknown no dejan la expresión sin decidir:** `meets` los iguala solo consigo mismos.
  - El traductor también decide con Any: `List<Any>{ null } includes (null as Any)` es `Contains` y `List<Any>{ 1 } includes (null as List<Any>)` es `Includes`.
  - Con la guarda, esos casos quedaban en la respuesta vieja, y su mutación sobrevivía.
  - Queda sin decidir `{ 1 } includes (null as Any)`, que el traductor resuelve como `Contains`, por subtipo (backlog 1c8). Lo documenta `TestMembershipWithAnAnyOperandStaysUndecided`.
- **No se lee la rama que estrechó `narrowChoiceOperand`.** Una choice se lee por sus ramas, y si una rama elige la sobrecarga del elemento y otra la de la lista, gana la del elemento. Así lo resuelve el traductor en `{ 'a', null } includes (null as Choice<List<FHIR.string>, String>)`: escribe `Contains`. Contra un `FHIR.Range`, el traductor elige el intervalo; este motor, el elemento. Es la decisión 7 del ADR 003, que tomó el usuario.
- **La guarda del modo especulativo se queda:** lo que encuentra un pase especulativo es del pase dueño, como en `convert` y `narrow`. El plan la daba por equivalente, y la ronda 2 del review lo refutó: una definición alcanzada primero desde el cuerpo de una función sin tipo de retorno se memorizaba dentro del pase especulativo, y ningún pase la volvía a visitar. Ahora `typeOfDefine` es el pase dueño de la definición, la alcance quien la alcance: pone `speculative` en 0 mientras la infiere.

**Casos a mano** (`engine_membership_model_test.go`):
- `TestMembershipThroughTheModelFollowsTheSpecification`: 38 casos sobre dos juegos de datos (`testdata/membershipmodel/`).
  - Por cada caso, el test comprueba que el nodo coincida con el que escribió el traductor de Java, traducido solo.
  - Las respuestas de Java coinciden con la tabla, salvo las diferencias que nombra `membershipModelDiffers`: su defecto de la lista-lista con un null, el null de Contains con Any, 1c14, 1c15 y los tres casos con un `FHIR.Range` (decisión 7).
  - D12 y D13 no se asertan con datos: con un period, la instancia `FHIR.dateTime` da error, también en main (backlog 1c16).
- `TestMembershipWithAnAnyOperandStaysUndecided`: la pérdida conocida con Any.
- Salió `TestMembershipWithAModelConvertedElementStaysUndecided`: sus filas ahora dan false y están en la tabla nueva (D0, D1 y D5).

**Corpus `fhirmembership`:** `gen -area fhirmembership -seed 1 -n 200`, sobre `fhir-data.json`, con las respuestas de Java y de JS y los nodos del traductor (`differential/nodes`). El oráculo da:
- Java: 21 `own defect` (las expresiones que el traductor rechaza y este motor acepta, 1c15) y 1 `reference defect` (la lista-lista con un null);
- JS: ninguna divergencia.

No queda ninguna sin clasificar.

**Corpus existentes:** ningún snapshot cambió. Es el diferencial de main contra la rama sobre todos los corpus.

**Mutaciones:**
- `scripts/mutations/006-membership-model.sh` tiene 20, todas detectadas.
- `005-membership.sh` se reescribió para apuntar al código que se movió a `sema`: sigue en 22, todas detectadas.

## Review, ronda 1 (Codex y Fable, 22-10-2026)

| Hallazgo | Verificación | Resolución |
|---|---|---|
| Una choice con una lista entre sus ramas: la primera rama decidía (medio, Codex) | `{ 'a', null } includes (null as Choice<List<FHIR.string>, String>)`: el traductor escribe `Contains` y este motor decidía `Includes` | **Aceptado:** gana la rama del elemento; caso D30 y la mutación «la primera rama decide» |
| Una choice dentro de los elementos del contenedor no se recorría (medio, Codex) | `(null as System.Quantity) included in (null as List<Choice<FHIR.string, FHIR.Quantity>>)` quedaba sin decidir; el traductor escribe `In` | **Aceptado:** `reaches` recorre una choice de los dos lados; casos D31 a D33 y dos mutaciones |
| Un operando choice contra un `FHIR.Range` (medio, Fable) | El traductor escribe `IncludedIn` y responde null; este motor, `In` y false. Con un `FHIR.Period`, el traductor elige el punto | **Decisión del usuario:** gana el elemento (ADR 003, decisión 7); casos D35 a D37, con la respuesta de Java nombrada como diferencia |
| Los intervalos no se probaban punto a punto (bajo, Fable) | Sin caso, la mutación sobrevivía | **Aceptado:** caso D34 y su mutación |
| `(null as FHIR.Period) included in (null as List<Interval<Date>>)` encadena conversiones (bajo, Fable) | El traductor lo rechaza | **Al backlog:** 1c15 |
| El oráculo no lee una ruta con varios valores en los datos (bajo, Fable) | El corpus no genera ninguna | **Aceptado como límite:** comentario en el oráculo |

## Review, ronda 2 (Codex y Fable, 22-10-2026)

| Hallazgo | Verificación | Resolución |
|---|---|---|
| Una definición alcanzada primero por un pase especulativo pierde su sobrecarga (medio, Codex) | `define function F(): Later`, `define Entry: F()`, `define Later: { 'a' } includes (null as String)`: `Entry` daba null; en orden, false. Java da false | **Aceptado:** `typeOfDefine` infiere con `speculative` en 0. `TestMembershipInADefinitionReachedFirstFromACallIsDecided` y su mutación. Ningún snapshot cambió |
| El corpus generado no prueba la lista-lista a través del modelo (bajo, Fable) | Las 12 filas IncludedIn/Includes de `fhirmembership.nodes.tsv` son listas `null as` o vacías | **Aceptado como límite:** lo cubren los casos D7 a D9 y la mutación «toda sobrecarga es la del elemento». Ampliar el generador cambia el corpus, y va aparte |
| `containerForm` solo convierte la rama de una choice (bajo, Fable) | Con la conversión del modelo desactivada, solo falla D20 | **Aceptado:** comentario, nombre de la mutación del 005 y texto del plan |
| El plan decía `make check` pendiente (bajo, Fable) | Pasó el 22-10-2026, antes de esta ronda | **Aceptado:** se vuelve a correr con los cambios |

## Review, ronda 3 (Codex y Fable, 22-10-2026): converge

Ninguno de los dos deja hallazgos de severidad media o mayor. Probaron el arreglo de la ronda 2 en los dos órdenes de declaración, alcanzando la definición desde:
- el cuerpo de una función;
- el primer pase de `aggregate`;
- definiciones anidadas, autorreferencias y librerías incluidas.

Cada diagnóstico y cada conversión aparecen una vez, bajo la definición que los contiene. El chequeo de los 14 `.cql` de `testdata` da lo mismo con el arreglo y sin él: ni los snapshots ni `ConversionsByDefine` cambian. En main, esos casos dependían del orden: se perdían también diagnósticos y conversiones.

| Hallazgo | Verificación | Resolución |
|---|---|---|
| Un ciclo que se cierra dentro de un pase especulativo no reporta su diagnóstico (bajo, Fable) | `define function F(): A` con `define A: F()`: cero diagnósticos, en la rama y en main | **Al backlog:** 1c17. Es anterior al 006 y no se amplía en un arreglo de review |

## Cierre

`make check` en verde el 22-10-2026, con el arreglo de la ronda 2. Evidencia: [`docs/evidence/2026-10-22.md`](../evidence/2026-10-22.md), generada por `make evidence`.
