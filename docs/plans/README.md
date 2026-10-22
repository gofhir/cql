# Punto de reanudación

Al reanudar el trabajo o después de una compactación, este archivo es lo primero que se lee. Se actualiza al terminar cada paso. El proceso es el de `NUCLEO.md` §10 (`~/projects/personal/health-platform/workspace/NUCLEO.md`), y lo ejecuta la skill `fhir-incremento`. Rige para cql desde el 18-10-2026.

## Incremento activo

**005: `in` y `contains` con un null siguen la especificación** (backlog 1c4). Plan: [005-pertenencia-con-null.md](005-pertenencia-con-null.md).

**Estado al 21-10-2026:** el 004 está cerrado: el PR 1 es #158 (8a6d2f9) y el PR 2 es #159 (a01570d). El plan 005 está escrito.

**Próximo paso concreto:** el plan está aprobado (21-10-2026), con la decisión 6 del ADR 003: decide el contenedor. Rama `fix/membership-null`. Implementado, sin commit (ver «Implementación» del plan 005). El review convergió en la ronda 4, una ronda corta que pidió el usuario tras el tope de tres. Hay 22 mutaciones, todas detectadas, y `make check` está en verde. `make evidence` está en verde ([2026-10-21](../evidence/2026-10-21.md)). Falta el commit (`fix:`, v1.24.2), cuando el usuario lo apruebe. Después: `make evidence`, y preguntar al usuario por el commit (`fix:`, v1.24.2).

## Pendientes, en orden

1. **003:** cerrado con el PR 7. `pending-verdicts.tsv` está vacío y el ADR 001 está «Aceptado».
1b. **Un `''` explícito en un Code se vuelve null:** `Code{code:'x', system: ''}.system is null` da true (sonda del 20-10-2026, R3-2 del PR 7). `types.Code` y `types.Concept` guardan sus campos como `string`, así que no distinguen `''` de null. Ninguna definición del corpus lo ejercita. Es un cambio del motor y va en su propio PR. Pasa lo mismo con `Concept.codes` null, que se vuelve `{}`.
1c2. **Igualdad de listas con un par indeciso y otro desigual:** la especificación no lo decide, y no es un defecto. Hay tres lecturas:
  - la del motor y Java, que recorren en orden y se detienen en el primer par indeciso;
  - la de JS, que da false si un Code difiere;
  - la de los tres motores con un elemento null, que dan null.
  Lo decide el usuario. Es el mismo frente que `tupleEqual` (A1 del plan 004). Ver «Review, ronda 1» del plan 004 (D334, D593).
1c3. **Restos del 004 que no son de su alcance:**
  - un codesystem de una librería incluida (`Code '1' from H."CS"`, o un code declarado así) no se resuelve: da `system` "H.CS" y `version` null;
  - `buildConcept` descarta un `(null as Code)` de `codes`;
  - `~` entre Code y Concept es asimétrico;
  - sema acepta un code de un codesystem no declarado.
1c4. **CERRADO por el 005 (sin commit).** **`in` y `contains` con un null:** `{1, 2} contains (null as Integer)` y `(null as Integer) in {1, 2}` dan null. La especificación (List Operators, In y Contains) dice false si la lista no tiene ningún null, y las dos referencias dan false. Pasa con todos los tipos y es anterior al 004. Son 10 `own defect` por referencia en `codesets`.
1c5. **Mode no agrupa todo lo que `=` iguala:** fuera de los números y los Codes, la clave de Mode es el `String()` del valor, y separa valores iguales: `Mode({1 'mg', 1 'mg', 1.0 'mg', 1.0 'mg', 2 'mg', 2 'mg', 2 'mg'})` da `2 'mg'`, y `distinct` da dos. Pasa también con `1 'g'`/`1000 'mg'`, `1 week`/`7 days`, Dates de distinta precisión, DateTimes de distinto offset e intervalos cerrados y abiertos de los mismos puntos. Es anterior al 004 (idéntico en main) y lo encontraron Codex y Fable en la ronda 3 del PR 2.
1c6. **`included in` sobre listas no coincide con `in` en la precisión temporal:** `@2020-01 included in { @2020-01-01 }` da null y `@2020-01 in { @2020-01-01 }` da false. Las dos referencias dan false en los dos. `included in` usa una pertenencia de tres valores (`listContainsValueTriState`) que `in` no usa. Lo encontró la ronda adversarial del 005.
1c7. **Quantity con unidades distintas en una lista:** `5 'mg' in { 5000 'ug' }` da true en este motor y en cql-execution, y false en Java. Hace falta un oráculo de UCUM para clasificarlo. Lo encontró la ronda adversarial del 005.
1c8. **Un nombre de una librería incluida no tiene tipo estático:** `sema` tipa `H.X` como Unknown (`sema/infer.go:144`, «Resolving it needs that library's AST, which this pass is not given»). Así, `H.L includes (null as Integer)`, con `H.L` una lista null, da null donde la sobrecarga lista-singleton da false. Pasa lo mismo sin plan, con el ELM importado: `(null as List<String>) included in { 'b', null }` da true donde la sobrecarga lista-lista da null, porque `Includes` e `IncludedIn` llegan a los mismos operadores de tiempo que la sobrecarga singleton. Pasa también con una lista sin tipo (`{} includes (null as Integer)`) o un `null` sin tipo, que siguen en null (ronda 1 del review del 005, Fable). Lo documentan `TestMembershipWithAnUntypedNullStaysUndecided` y `TestMembershipWithNullFromTheReferenceELM` (005).
1c9. **El importador de ELM convierte en null un nodo que no conoce:** `elm/importer.go:443` («Unknown types → literal placeholder») devuelve un null literal. El nodo `Date` que escribe el traductor de Java (`{"type": "Date", "year": …}`) es uno de ellos, así que `{ @2020-01-01 }` importado tiene un null, y `(null as Date) in { @2020-01-01 }` da true. Además, `elm.UnmarshalLibrary` rechaza los selectores de Code y de Tuple del traductor (`value` como objeto). Es parte del backlog 4. Lo documenta `TestMembershipWithNullFromTheReferenceELM` (005).
1c10. **`includes` e `included in` sobre una lista de listas sin null:** `{ 3 } included in List<List<Integer>>{ { 3 } }` es la sobrecarga singleton (el elemento es `List<Integer>`), y las dos referencias dan true. Este motor lo lee como lista-lista, por el valor, y da false. Son D241 y D253 del corpus `membership` (005), `own defect` con las dos referencias. La sobrecarga sale del tipo estático solo cuando hay un null (`nullMembership`).
1c11. **Un elemento que necesita la conversión del modelo:** `Patient.gender included in { 'male', 'female' }`, sin gender, da null, y `Patient.gender in { 'male', 'female' }` da false. El elemento es `FHIR.AdministrativeGender` y la lista es de String: la conversión la registra `sema` para `in` (`inferMembership`, `expectTyped`), pero no para `includes`, `included in` ni `during`, y el evaluador no tiene el modelo para deducirla. Pasa también con el contenedor: `Patient.name.given includes (null as String)` da null, y `O.effective during E.period`, sin period, también da null; Java da false en los dos (ronda 2, Fable). Es la frase que escriben las medidas. Arreglarlo es registrar esa conversión en `inferTiming`, que también cambiaría el valor que recibe el operador cuando no hay null: va en su propio incremento. Lo encontró Fable en las rondas 1 y 2 del review del 005; lo documenta `TestMembershipWithAModelConvertedElementStaysUndecided`.
1c13. **Defecto del traductor de Java 3.26.0, para reportar:** la sobrecarga que elige depende de las definiciones anteriores. Solo, `(null as List<Decimal>) included in List<Integer>{ 1, null }` se traduce como `IncludedIn`. Si antes en la librería está `define X: null in { 1, 2 }`, la misma expresión sale `In(Any, List<Any>)`: el traductor reusa la sobrecarga genérica que ya instanció. Lo encontraron Codex y Fable en la ronda 3 del review del 005. Hay que redactar el issue para cqframework/clinical_quality_language y publicarlo cuando el usuario lo apruebe.
1c12. **Un intervalo en un intervalo null:** `Interval[1, 2] in (null as Interval<Integer>)` da false, y la especificación (Interval Operators, Included In) dice null cuando un argumento es null. Es anterior al 005: la sobrecarga intervalo-intervalo de `in` llega a la rama de List Contains (`evalInContains`). Lo encontró Fable en la ronda 1 del review del 005.
1c. **`make evidence` llega a 1,08 GB de RSS y tarda 515 s.** El oráculo de `types` no lo explica (0,5 s, 11 MB): hay que medir dónde está el costo y corregirlo.
2. **`distinct` y operaciones de conjuntos sobre intervalos** son O(n²). Con un plazo ya no corren de más, desde #144, pero siguen siendo lentas: hace falta un índice por cubetas para intervalos.
3. **D4a, identidad de tipo:** un descriptor `{modelo, nombre}` en vez del texto de `Type()`. Ver [arquitectura-verificacion-2026-10-18.md](../arquitectura-verificacion-2026-10-18.md) §4.
4. **ELM oficial:** el documento completo del traductor y los 54 de 115 tipos de nodo que faltan. Ya hay un arreglo listo para entrar cuando sea: `Abs` se importa como `OpPositive`.
5. **Borrar la carga perezosa de `eval`** (`ensureLibraryLoaded`): desde #146 es inalcanzable desde `Engine`.
6. **Traducir los comentarios del código al español,** como pide la carta. Va en un incremento propio; hasta entonces, los comentarios nuevos van en inglés para no mezclar idiomas en un mismo archivo (decisión del usuario, 18-10-2026).
7. **Verificar contra las referencias** si un `valueQuantity` que solo trae `value` es `FHIR.Quantity`. El motor dice que no; se observó en el 001.

- **Defectos propios que encontró el 003, en el área `types`:**
  - `cast` y `as` hacia un tipo genérico (`List<T>`, `Interval<T>`) no comprueban el tipo (58 definiciones);
  - `cast` convierte en vez de fallar (11);
  - Long sin tipo en ejecución (10; D4c).
- **Defectos propios que encontró el 003, en el área `fhir`:** siete grupos, listados en el plan 003, «Implementación del PR 3». Entre ellos, el constructor de una instancia FHIR que pierde el valor y las llamadas sin firma posible que no se rechazan.

## Decisiones pendientes del usuario

- Publicar el release que acumula #144, #146, #147 y #148.
- Agregar Australia/Lord_Howe a la matriz de zonas de CI.

## Cerrados con el §10

| PR | Qué |
|---|---|
| #148 | 001: la división Decimal y los oráculos que distinguen el tipo de los números |
| #150 | 002: el ADR 001 (Propuesto), su gobierno en CI, la procedencia de la suite y la deuda de divergencias |

## Cerrados el 18-10-2026, antes de adoptar el §10

| PR | Qué |
|---|---|
| #144 | Cancelación: `ErrCanceled`, el plazo real en `ErrTimeout` y corte de los bucles largos |
| #146 | Resolución de includes: identidad, ciclos, `using` en todo el grafo y `WithLibraryLoader` deprecado |
| #147 | Semántica de includes: `ErrSemantic`, `Diagnostic.Library`, la posición del include y `CheckContext` |

El defecto del loader de `gofhir/server`, que está congelado, quedó en `opensource/innovation/paito_backlog.md` §4b.
