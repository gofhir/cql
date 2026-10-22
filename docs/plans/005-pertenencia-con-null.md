# 005: la pertenencia con un null sigue la especificación

Estado: aprobado por el usuario el 21-10-2026, con la decisión 6 del ADR 003 (decide el contenedor). Rama: `fix/membership-null`.

## Por qué

Viene del backlog (1c4 de `docs/plans/README.md`), y lo encontró el corpus `codesets` del 004. `{1, 2} contains (null as Integer)` y `(null as Integer) in {1, 2}` dan null en este motor. La especificación dice false, y las dos referencias dan false. Pasa con todos los tipos, y es anterior al 004. En `codesets` son 10 `own defect` por referencia.

## Revisión previa

**Especificación** (texto vigente en `cql.hl7.org/09-b-cqlreference.html`, igual al de 1.5.3 en estas frases):
- *List In*: «returns true if the given element is in the given list using equality semantics, with the exception that null elements are considered equal. If the first argument is null, the result is true if the list contains any null elements, and false otherwise. If the second argument is null, the result is false.»
- *List Contains*: la misma regla con los argumentos al revés. «If the first argument is null, the result is false. If the second argument is null, the result is true if the list contains any null elements, and false otherwise.»
- *List Includes* e *Included In*: en la sobrecarga lista-singleton son sinónimos de `contains` y de `in`. En la lista-lista, «if either argument is null, the result is null».
- *Interval Contains* (punto): «If the first argument is null, the result is false. If the second argument is null, the result is null.»
- *Interval In* (punto, también `during`): «If the first argument is null, the result is null. If the second argument is null, the result is false.»
- *Interval Includes* e *Included In*: en la sobrecarga punto-intervalo son sinónimos de `contains` y de `in`. En la intervalo-intervalo, null si un argumento es null.

**Dónde la especificación no decide** (hallazgo 4 de Fable). Las dos frases chocan cuando los dos argumentos son null:
- `(null as Interval<Integer>) contains (null as Integer)`: la primera frase dice false y la segunda null;
- `(null as Integer) in (null as Interval<Integer>)`: la primera dice null y la segunda false.

Las dos referencias dan false en los dos casos. En List In y List Contains las dos frases coinciden en false, así que no hay conflicto.

**Sondas del 21-10-2026** (motor, Java 3.26.0 y cql-execution 3.3.2). Están en el scratchpad, en `probe005/` (`p.cql`, `q.cql` y `r.cql`), y son 60 definiciones. El motor difiere de las dos referencias en 38, y en todas da null donde las referencias dan false. Una es la precisión temporal, que queda fuera; las otras 37 son:
- `in`, `contains`, `includes`, `included in` y `during` sobre listas, con un elemento null y una lista sin null, una lista vacía o una lista null. Con Integer, String, Code, Date, Interval, Tuple y List: `{ Interval[1, null] } contains (null as Interval<Integer>)` también es false, porque el null está dentro del elemento, no es el elemento.
- La sobrecarga singleton con una lista null: `(null as List<Integer>) includes 1` y `1 included in (null as List<Integer>)`, y también con una definición `L` null.
- La sobrecarga punto con un intervalo null: `(null as Interval<Integer>) includes 1`, `1 included in (null as Interval<Integer>)` y `1 during (null as Interval<Integer>)`.
- Un null calculado: `First(List<Integer>{}) in { 1, 2 }`.
- `not ((null as Integer) in { 1, 2 })`, que da true en las referencias.

Coinciden ya:
- un null buscado en una lista con un null (true);
- `null in {}` (false);
- una lista null con un elemento no null en `in` y `contains` (false);
- `Interval[1, 2] contains (null as Integer)` y su inversa (null en los tres);
- `properly includes`/`properly included in` con un null (false en Java; cql-execution no los evalúa).

**La sobrecarga lista-lista con null** (hallazgo alto de los dos revisores). `{ 's', 'a', 'm' } includes (null as List<String>)` da null en este motor y en cql-execution, y la conformidad (`IncludesNullRight`) pide null. Java da false: es un defecto de Java. En ejecución, ese par (lista, null) es idéntico al de `{ 's', 'a', 'm' } includes (null as String)`, que debe dar false. Solo el tipo estático los distingue:
- el traductor lo decide así: con el tipo del elemento igual al de los elementos de la lista, es la sobrecarga singleton y emite `Contains`/`In`. Así, `{ { 1 } } includes (null as List<Integer>)` es singleton (`Contains`, false en las dos referencias), y `{ 's' } includes (null as List<String>)` es lista-lista (`Includes`);
- este motor tiene el tipo de cada subexpresión (`sema.Result.Types`, que se lee con `e.ctx.Plan.TypeOf(nodo)`). `TypeOf` de la expresión entera es Boolean, así que hay que leer el de cada operando (hallazgo de Codex).

**El código** (`eval/evaluator.go`):
- `evalInContains`, ramas de lista de `in` y de `contains`: un elemento null en una lista sin null devuelve null a propósito, con el comentario `null in {1,2} = null`, que la especificación contradice. Viene de fb090a5. La conformidad no tiene ningún caso de una lista sin null.
- `evalListTimingOp`, ramas singleton de `includes` y de `included in`/`during`: la misma regla.
- `evalTimingExpr`: con un contenedor null, ninguno de los dos operandos es una `cqltypes.List` ni un `cqltypes.Interval` en ejecución, y la propagación de null responde null.
- **ELM importado:** el traductor ya resuelve la sobrecarga y emite `In`/`Contains` para la singleton y la punto. El importador los lleva a `ast.OpIn`/`ast.OpContains`, que llegan a `evalInContains` (`elm/importer.go:544`). Codex lo midió en el ELM de `q.cql`: D6 da null hoy. Con el arreglo de `evalInContains`, ese camino queda resuelto sin tipos.
- **Librerías incluidas:** sus definiciones sí tienen plan (`engine.go:474`, `eval/context.go:562`). El plan anterior decía lo contrario (hallazgo de Codex).

**`gofhir/server` (congelado):** no evalúa la pertenencia; la delega a esta librería. `internal/cqlpushdown` solo traduce a búsqueda el `in` contra un ValueSet (`:in`). **Descartar:** no hay nada que tomar.

## Alcance

1. **`in` y `contains` sobre listas** (`evalInContains`): un elemento null da true si la lista tiene un null en el primer nivel y false si no, incluida la lista vacía. Una lista null da false con cualquier elemento, también null. Esto cubre también el ELM importado.
2. **`includes`, `included in` y `during` en la sobrecarga singleton o punto, con un null de un lado:** se lleva a `evalInContains`, con `contains` para `includes` y con `in` para `included in` y `during`. La sobrecarga sale del tipo estático de los dos operandos:
   - es singleton si el contenedor es `List<T>` y el otro operando es de tipo T, aunque T sea una lista;
   - es punto si el contenedor es `Interval<P>` y el otro es de tipo P;
   - si no hay tipo (Unknown, Any, una elección), queda como está. Ninguna conversión se infiere del valor.
3. **Los dos argumentos null en un intervalo:** false, por la decisión 6 del ADR 003 (ver «Decisión del usuario»).

**Fuera:**
- La sobrecarga lista-lista y la intervalo-intervalo: siguen en null, como pide la conformidad.
- Un punto null en un intervalo no null: sigue en null, como dice la especificación.
- **La pertenencia indecidible por precisión temporal.** `@2020-01 included in { @2020-01-01 }` da null en este motor; `in` da false, y las dos referencias dan false. Va al backlog (1c6): `included in` sobre listas usa una pertenencia de tres valores que `in` no usa.
- **Quantity con unidades distintas en una lista:** `5 'mg' in { 5000 'ug' }` da true en este motor y en cql-execution, y false en Java. Va al backlog (1c7), para clasificarlo con un oráculo de UCUM.
- `properly includes` y `properly included in`, Mode (1c5) y el resto del backlog.

## Decisión del usuario (tomada el 21-10-2026)

**Decisión 6 del ADR 003: con los dos argumentos null, decide el contenedor.** `(null as Interval<Integer>) contains (null as Integer)` y `(null as Integer) in (null as Interval<Integer>)` dan false. Sigue a la frase del contenedor null, que en List In y List Contains es la única respuesta, y las dos referencias dan false. La alternativa es la frase del punto null, que da null en los dos casos (ronda 1 del review, Fable: el texto anterior decía que no había una lectura así, y era falso). El usuario eligió esta, y la alternativa queda como lectura que se deja de lado.

## Versión

Corrige resultados que la especificación fija: `fix:`, para v1.24.2. Nota del release: `not (x in L)` con x null pasa de null a true, y un filtro que antes excluía ese caso pasa a incluirlo.

## Tests y datos

- **Casos a mano,** `TestMembershipWithNullFollowsTheSpecification`: las sondas de este frente (ver «Implementación»), cada una con la frase de la especificación que la decide, o con la decisión 6 en los dos casos que ninguna frase decide. El valor esperado sale de la especificación, y las referencias lo confirman, salvo el defecto de Java en `IncludesNullRight`, que se anota. Se agregan además (hallazgos de Codex):
  - un null de un parámetro, de una definición y de una función, y una definición de una librería incluida;
  - `where x in L` en una query, y un `return` que lo usa;
  - una lista FHIR (`Patient.name.given`) con cero, uno y varios elementos;
  - el ELM importado de las sondas, evaluado con `elm.Import`;
  - el negativo: `'abc' contains (null as String)` sigue siendo el operador de String (null).
- **Corpus generado, área nueva `membership`:** `gen -area membership -seed 1 -n 400`, con respuestas de Java y de JS. Combina:
  - los cinco operadores (`in`, `contains`, `includes`, `included in`, `during`);
  - elementos de Integer, Decimal, String, Boolean, Date, DateTime, Time, Quantity, Code, Interval, Tuple y List, del mismo tipo que la lista;
  - un contenedor lista o intervalo;
  - el elemento null (`null`, `(null as T)`, `First(List<T>{})`) o un valor;
  - la lista vacía, una lista null, una lista sin null y una lista con uno o más null;
  - la sobrecarga lista-lista con null, para que no se mueva.

  Para no mezclar frentes, las fechas y horas de una fila tienen una sola precisión, y las Quantity, una sola unidad.
- **Oráculo de `membership`:** lee la lista con un lector de primer nivel, no con una expresión regular, porque un null dentro de un elemento (`Interval[1, null]`, `Tuple { a: null }`) no es un elemento null (hallazgo 5 de Fable). Decide:
  - con un null del lado del elemento o del contenedor, por las frases citadas, por la decisión 6 y por la sobrecarga que da el tipo;
  - con valores sin nulls anidados, escritos en forma canónica: pertenece si su texto es el de un elemento de la lista. El generador escribe cada valor de una sola manera.

  Las filas restantes no las decide, y las clasifica la diferencial con su evidencia.
- **Corpus existentes:** los 20 `own defect` de `codesets` deben pasar a coincidir, y ninguna otra respuesta de los snapshots debe moverse. Lo comprueba la diferencial de main contra la rama, sobre todos los corpus.
- **Mutaciones** (`scripts/mutations/005-membership.sh`):
  - restaurar cada rama: las dos de `evalInContains`, las dos de `evalListTimingOp` y la de `evalTimingExpr`;
  - la sobrecarga leída del valor en vez del tipo, que debe romper `IncludesNullRight` y la lista de listas;
  - un elemento null que siempre da true;
  - la decisión 6 invertida;
  - en el oráculo: un null anidado leído como elemento; una lista null leída como una con null; la sobrecarga lista-lista leída como singleton;
  - el corpus cambiado.

  Una lista null leída como vacía no es una mutación válida: dan lo mismo en las cuatro sobrecargas (hallazgo de Codex).
- **Las tres versiones FHIR:** no aplica. Es un cambio de operadores de System, y la lista FHIR del caso a mano es de R4, la única que trae el modelo del motor.

## Criterio de salida

- Los casos a mano dan lo que dice la especificación o la decisión 6.
- `membership` no deja `own defect` ni filas sin clasificar.
- `codesets` queda sin `own defect`, y ningún otro snapshot cambia.
- Las mutaciones se detectan todas, y `make check` y `make evidence` pasan.

## Riesgos

- **Un contenedor null se ve igual en ejecución, sea lista o intervalo.** Con el tipo estático, cada uno toma su frase; sin tipo, el motor no lo adivina y queda como está.
- **Las medidas que dependen de un null en un `in`:** ver la nota del release.

## Ronda adversarial (Codex y Fable, 21-10-2026)

| Hallazgo | Verificación | Resolución |
|---|---|---|
| La sobrecarga lista-lista con null se confunde con la singleton (alto, los dos) | Sonda: `{ 's', 'a', 'm' } includes (null as List<String>)` debe seguir en null (conformidad `IncludesNullRight`, cql-execution), y `{ { 1 } } includes (null as List<Integer>)` es singleton (el traductor emite `Contains`) | **Aceptado:** la sobrecarga sale del tipo de los dos operandos (alcance 2), con mutación y casos |
| El ELM importado queda mal (alto, Codex) | El ELM de `q.cql`, D6, da null | **Aceptado, y resuelto por el alcance 1:** el traductor emite `In`/`Contains`, que llegan a `evalInContains`. Hay un caso a mano con `elm.Import` |
| Las librerías incluidas sí tienen plan (medio, Codex) | `engine.go:474`, `eval/context.go:562` | **Aceptado:** la premisa era falsa. Hay un caso a mano |
| Mutación equivalente: lista null leída como vacía (medio, Codex) | Las dos dan false en las cuatro sobrecargas | **Aceptado:** se reemplaza |
| Faltan los caminos de ejecución que no son literales (medio, Codex) | `evaluator.go:6947`, `:2016`, `:5619` | **Aceptado:** casos de null calculado, parámetro, query, lista FHIR y `during`. Los filtros del retrieve los resuelve el proveedor, y no entran |
| Las sondas de intervalo con null sí tienen respuesta (bajo, Codex; medio, Fable) | `qjava.json` y `qjs.json` dan null en D4 y D5 | **Aceptado:** la tabla decía que no respondían. Coinciden los tres |
| Los sinónimos sobre un intervalo null quedan inconsistentes (medio, Fable) | Sonda: `(null as Interval<Integer>) includes 1` da null; Java y JS dan false; el traductor emite `Contains` | **Aceptado:** entra en el alcance 2 |
| Casos donde la especificación no decide se citaban como decididos (medio, Fable) | Interval Contains e Interval In con los dos null: las frases chocan | **Aceptado:** decisión 6 del ADR 003, a decidir por el usuario |
| El oráculo lee un null anidado como elemento (medio, Fable) | `{ Interval[1, null] } contains (null as Interval<Integer>)` es false en las dos referencias | **Aceptado:** lector de primer nivel y mutación |
| Las filas sin null no cierran «sin filas sin clasificar» (medio, Fable) | Sondas: la precisión temporal y la Quantity con unidades distintas difieren | **Aceptado:** el generador las evita, y van al backlog (1c6, 1c7). El oráculo decide los valores canónicos |
| Faltan mutaciones del camino del tipo y del oráculo en filas sin null (bajo, Fable) | — | **Aceptado** |
| Nombre: `evalTimingExpr`, no `evalTimingExpression` (bajo, Fable) | `evaluator.go:6904` | **Aceptado** |

## Implementación (21-10-2026, sin commit)

**El motor** (`eval/evaluator.go`):
- `evalInContains`: `in` y `contains` sobre listas responden con `listContainsValue`, que encuentra un null solo en una lista que lo tiene. Una lista null, y también un intervalo null, no tienen nada.
- `nullMembership`, que llama `evalTimingExpr` antes que el resto: con un null de un lado, `includes`, `included in` y `during` toman la sobrecarga que dan los tipos estáticos (`membershipOverload`). La sobrecarga singleton o punto va a `evalInContains`. La lista-lista da null. Sin tipo, el operador sigue como antes.
- `evalListTimingOp` no cambia. Sin tipo, un par (lista, null) puede ser cualquiera de las dos sobrecargas.

**Lo que el corpus encontró y entró:** la sobrecarga lista-lista con un null daba true cuando la lista tenía un null (`(null as List<Boolean>) included in List<Boolean>{ true, null }`). La especificación y la conformidad dicen null. Era el mismo mecanismo, y lo resuelve `listOverload`. Java también da true.

**Lo que no entró, con su evidencia** (backlog de `docs/plans/README.md`):
- **1c8:** un nombre de una librería incluida no tiene tipo estático (`sema/infer.go:144`), y sin tipo no hay sobrecarga. `H.L includes (null as Integer)` sigue en null. Lo mismo pasa con el ELM importado y la lista-lista con un null en la lista. Lo documentan `TestMembershipWithAnUntypedNullStaysUndecided` y `TestMembershipWithNullFromTheReferenceELM`.
- **1c9:** el importador de ELM convierte en null un nodo que no conoce (`elm/importer.go:443`), y el `Date` de Java es uno de ellos. Además rechaza los selectores de Code y de Tuple. El test del ELM nombra esas definiciones.
- **1c10:** `{ 3 } included in List<List<Integer>>{ { 3 } }` sin null: es la sobrecarga singleton, y el motor la lee como lista-lista. Son D241 y D253 de `membership`, los únicos `own defect` del área.

**Casos a mano:**
- `TestMembershipWithNullFollowsTheSpecification`: 76 casos. Son las 60 sondas menos las 6 de otro frente (precisión temporal, Quantity con unidades distintas, una tupla con null en `contains`, y `properly`), más los 2 de la lista-lista con un null en la lista los 7 de las conversiones implícitas (ronda 1 del review) y los 13 de la ronda 2.
- `TestMembershipWithNullThroughOtherPaths`: parámetro, función, librería incluida y query.
- `TestMembershipWithNullInAFHIRList`: listas FHIR de cero, uno y varios elementos.
- `TestMembershipWithNullFromTheReferenceELM`: los 76 casos desde el ELM del traductor de Java (`testdata/membership/`).
- `TestMembershipWithAModelConvertedElementStaysUndecided`: la pérdida conocida de 1c11.
- `TestAStringIsNoListToBeIn`: `'abc' contains (null as String)` no es una pertenencia, y el chequeo lo rechaza.

**Corpus `membership`:** `gen -area membership -seed 1 -n 400`, con las respuestas de Java y de JS. El oráculo da:
- Java: 22 `reference defect` (la lista-lista con un null) y 2 `own defect` (1c10);
- JS: 2 `own defect` (1c10).

No queda ninguna sin clasificar.

**Corpus existentes:** solo se movieron las 10 definiciones de `codesets` con `contains`/`in` de un null, de null a false. `codesets` queda sin `own defect` (antes eran 20). Ningún otro snapshot cambió.

**Mutaciones:** `scripts/mutations/005-membership.sh` tiene 22: quince del motor, seis del oráculo y una del corpus. Las cuatro nuevas son de la ronda 1 del review: la guarda de precisión, `properly` quitado, `ImplicitlyConverts` cambiado por `Equal` y la guarda de `Conversions` desactivada. Las tres de la ronda 2: la conversión en un solo sentido, la lista-lista con tipos iguales y la lista que no se convierte elemento a elemento.

**Criterio de salida, ajustado:** `membership` no deja filas sin clasificar, y sus únicos `own defect` son los de 1c10.


## Review, ronda 1 (Codex y Fable, 21-10-2026)

| Hallazgo | Evidencia | Resolución |
|---|---|---|
| Una conversión implícita del sistema no elige la sobrecarga (medio) | `(null as Integer) included in List<Decimal>{ 1.0 }` daba null | **Aceptado:** `sema.ImplicitlyConverts` en `membershipOverload`, y 7 casos nuevos |
| Un contenedor FHIR.Period no se lee como intervalo (medio) | `(null as FHIR.Period) includes @2020-01-01T00:00:00` | **Aceptado:** `membershipOverload` lee `Plan.Conversions[container].Target` |
| Un elemento que necesita la conversión del modelo (medio, Fable) | `Patient.gender included in { 'male', 'female' }` da null; `in` da false | **Al backlog (1c11):** arreglarlo es registrar la conversión en `inferTiming`, y eso cambia el valor del operando también sin null. Es otro frente. Lo documenta `TestMembershipWithAModelConvertedElementStaysUndecided` |
| La decisión 6 del ADR describía mal la lectura rechazada (medio, Fable) | Interval In da null con el punto null primero; Contains, con el punto null segundo | **Aceptado:** el ADR y §Decisión dicen ahora que esa lectura da null en los dos |
| `{}` o `null` sin tipo con `includes` siguen en null (bajo) | sin tipo no hay sobrecarga | **Al backlog (1c8)** |
| `Interval[1, 2] in (null as Interval<Integer>)` da false (bajo) | Interval Included In dice null | **Al backlog (1c12):** es anterior al 005. El comentario de `evalInContains` lo nombra |
| Puntuación del ADR y 1c4 sin marcar como cerrado (bajo) | | **Aceptado** |

## Review, ronda 2 (Codex y Fable, 21-10-2026)

| Hallazgo | Evidencia | Resolución |
|---|---|---|
| La conversión se buscaba en un solo sentido: un contenedor que se convierte al tipo del elemento no elegía la sobrecarga (medio, Codex y Fable) | `(null as Decimal) included in List<Integer>{ 1, 2 }` y `1.0 included in (null as Interval<Integer>)` daban null; Java da false | **Aceptado:** `commonType` busca la conversión en los dos sentidos para el elemento y el punto |
| `ImplicitlyConverts` no convertía listas ni intervalos (medio, Codex) | `(null as List<Integer>) included in List<List<Decimal>>{ { 1.0 } }` daba null; Java y JS dan false | **Aceptado:** convierte elemento a elemento |
| 1c11 era más angosto que la pérdida (medio, Fable) | `Patient.name.given includes (null as String)` da null; Java da false | **Al backlog, ampliado:** 1c11 nombra el contenedor, y el test lo documenta |
| La lista-lista con tipos de lista que se convierten (bajo, Fable) | `(null as List<Integer>) included in List<Decimal>{ 1.0, null }` daba true; el traductor escribe `IncludedIn` | **Aceptado:** la lista-lista se elige también con listas que se convierten. La regla quedó en un solo sentido y era un error: ver la ronda 3 |
| El README decía 15 mutaciones (bajo, Fable) | | Ya estaba corregido en el repo; la copia era anterior |

Los snapshots no cambiaron con estos arreglos. En el ELM, los casos nuevos que convierten una lista los rechaza el lector (un `Query` con `source` en arreglo, backlog 4), y D74 es la pérdida de 1c8.

## Review, ronda 3 (Codex y Fable, 21-10-2026)

| Hallazgo | Evidencia | Resolución |
|---|---|---|
| La regla lista-lista iba en un solo sentido, por una evidencia que era un artefacto del orden de las definiciones (medio, Codex y Fable) | Solos en su librería, `(null as List<Decimal>) included in List<Integer>{ 1, null }` y `List<Integer>{ 1, null } includes (null as List<Decimal>)` son `IncludedIn` e `Includes`. En `cases.cql`, D74 y D75 salen `In(Any, List<Any>)`, porque `null in { 1, 2 }` (D2) instancia antes esa sobrecarga, y el traductor la reusa | **Aceptado:** la lista-lista usa `commonType`, en los dos sentidos. D75 espera null. D74 y D75 quedan como `misread` en el test del ELM |

La elección del traductor de Java depende de lo que la librería tiene antes: es un defecto de la referencia (principio 10), con su reproducción en `docs/plans/README.md` (1c13). Nada en este motor lo rodea.

La ronda 3 dejó un hallazgo medio, así que no convergió. Llegó al tope de tres rondas y se consulta al usuario. El arreglo ya está aplicado y verificado: 76 casos y 22 mutaciones, todas detectadas.

## Review, ronda 4 (Codex y Fable, 21-10-2026)

El usuario pidió una ronda 4 corta, solo sobre el arreglo de la ronda 3. **Converge:** ninguno de los dos dejó hallazgos. Los dos confirmaron con sondas:
- la lista-lista da null en los dos sentidos;
- la regla del elemento se mira primero, así que la lista-lista no le quita ningún caso;
- los comentarios y este plan coinciden con el ELM.

## Cierre (21-10-2026)

`make check` y `make evidence` en verde. La evidencia es [docs/evidence/2026-10-21.md](../evidence/2026-10-21.md), que genera `make evidence`.
