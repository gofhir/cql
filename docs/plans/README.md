# Punto de reanudación

Al reanudar el trabajo o después de una compactación, este archivo es lo primero que se lee. Se actualiza al terminar cada paso. El proceso es el de `NUCLEO.md` §10 (`~/projects/personal/health-platform/workspace/NUCLEO.md`), y lo ejecuta la skill `fhir-incremento`. Rige para cql desde el 18-10-2026.

## Incremento activo

**010: ToQuantity y la lectura de números en un string** (backlog 1c23, primer frente). Plan: [010-to-quantity.md](010-to-quantity.md). Paso: 8 terminado, el review converge en la ronda 3 (ninguno de los dos revisores deja hallazgos medios o mayores). `make check` en verde con 390 mutaciones atrapadas; diferencial 8008 idéntica a main en las 9 áreas después del review; `references.sh check` en verde. Pendiente: commit y PR cuando el usuario lo pida, y la decisión de publicar el issue de gofhir/ucum (la caché retiene el string de quien llama; redactado, sin publicar). Rama `fix/string-readers-1c23`, sin commit.

**009 cerrado:** #165 (dd35249), mergeado el 27-10-2026 con el CI en verde. Plan: [009-decimal-1-6.md](009-decimal-1-6.md). Issue para fhirpath: gofhir/fhirpath#235.

**008 cerrado:** #164 (1b45c9f). Plan: [008-fhirpath-1-16.md](008-fhirpath-1-16.md).

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
1c4. **CERRADO por el 005 (#160).** **`in` y `contains` con un null:** `{1, 2} contains (null as Integer)` y `(null as Integer) in {1, 2}` dan null. La especificación (List Operators, In y Contains) dice false si la lista no tiene ningún null, y las dos referencias dan false. Pasa con todos los tipos y es anterior al 004. Son 10 `own defect` por referencia en `codesets`.
1c5. **Mode no agrupa todo lo que `=` iguala:** fuera de los números y los Codes, la clave de Mode es el `String()` del valor, y separa valores iguales: `Mode({1 'mg', 1 'mg', 1.0 'mg', 1.0 'mg', 2 'mg', 2 'mg', 2 'mg'})` da `2 'mg'`, y `distinct` da dos. Pasa también con `1 'g'`/`1000 'mg'`, `1 week`/`7 days`, Dates de distinta precisión, DateTimes de distinto offset e intervalos cerrados y abiertos de los mismos puntos. Es anterior al 004 (idéntico en main) y lo encontraron Codex y Fable en la ronda 3 del PR 2.
1c6. **`included in` sobre listas no coincide con `in` en la precisión temporal:** `@2020-01 included in { @2020-01-01 }` da null y `@2020-01 in { @2020-01-01 }` da false. Las dos referencias dan false en los dos. `included in` usa una pertenencia de tres valores (`listContainsValueTriState`) que `in` no usa. Lo encontró la ronda adversarial del 005.
1c7. **Quantity con unidades distintas en una lista:** `5 'mg' in { 5000 'ug' }` da true en este motor y en cql-execution, y false en Java. Hace falta un oráculo de UCUM para clasificarlo. Lo encontró la ronda adversarial del 005.
1c8. **Un nombre de una librería incluida no tiene tipo estático, y Any decide solo contra Any:** `{ 1 } includes (null as Any)` queda en null; el traductor escribe `Contains`, por subtipo, y la especificación da false (006, `TestMembershipWithAnAnyOperandStaysUndecided`). Lo del nombre: `sema` tipa `H.X` como Unknown (`sema/infer.go:144`, «Resolving it needs that library's AST, which this pass is not given»). Así, `H.L includes (null as Integer)`, con `H.L` una lista null, da null donde la sobrecarga lista-singleton da false. Pasa lo mismo sin plan, con el ELM importado: `(null as List<String>) included in { 'b', null }` da true donde la sobrecarga lista-lista da null, porque `Includes` e `IncludedIn` llegan a los mismos operadores de tiempo que la sobrecarga singleton. Pasa también con una lista sin tipo (`{} includes (null as Integer)`) o un `null` sin tipo, que siguen en null (ronda 1 del review del 005, Fable). Lo documentan `TestMembershipWithAnUntypedNullStaysUndecided` y `TestMembershipWithNullFromTheReferenceELM` (005).
1c9. **El importador de ELM convierte en null un nodo que no conoce:** `elm/importer.go:443` («Unknown types → literal placeholder») devuelve un null literal. El nodo `Date` que escribe el traductor de Java (`{"type": "Date", "year": …}`) es uno de ellos, así que `{ @2020-01-01 }` importado tiene un null, y `(null as Date) in { @2020-01-01 }` da true. Además, `elm.UnmarshalLibrary` rechaza los selectores de Code y de Tuple del traductor (`value` como objeto). Es parte del backlog 4. Lo documenta `TestMembershipWithNullFromTheReferenceELM` (005).
1c10. **`includes` e `included in` sobre una lista de listas sin null:** `{ 3 } included in List<List<Integer>>{ { 3 } }` es la sobrecarga singleton (el elemento es `List<Integer>`), y las dos referencias dan true. Este motor lo lee como lista-lista, por el valor, y da false. Son D241 y D253 del corpus `membership` (005), `own defect` con las dos referencias. La sobrecarga sale del tipo estático solo cuando hay un null (`nullMembership`).
1c11. **Un elemento que necesita la conversión del modelo:** `Patient.gender included in { 'male', 'female' }`, sin gender, da null, y `Patient.gender in { 'male', 'female' }` da false. El elemento es `FHIR.AdministrativeGender` y la lista es de String: la conversión la registra `sema` para `in` (`inferMembership`, `expectTyped`), pero no para `includes`, `included in` ni `during`, y el evaluador no tiene el modelo para deducirla. Pasa también con el contenedor: `Patient.name.given includes (null as String)` da null, y `O.effective during E.period`, sin period, también da null; Java da false en los dos (ronda 2, Fable). Es la frase que escriben las medidas. Arreglarlo es registrar esa conversión en `inferTiming`, que también cambiaría el valor que recibe el operador cuando no hay null: va en su propio incremento. Lo encontró Fable en las rondas 1 y 2 del review del 005; lo documenta `TestMembershipWithAModelConvertedElementStaysUndecided`.
1c13. **Cerrado sin reportar: el traductor de Java 3.26.0 elige la sobrecarga según las definiciones anteriores, y la 5.4.0 ya no.** Sola, `(null as List<Decimal>) included in List<Integer>{ 1, null }` se traduce como `IncludedIn`. Si antes está `define X: null in { 1, 2 }`, la 3.26.0 la traduce como `In`. Lo encontraron Codex y Fable en la ronda 3 del review del 005. El 22-10-2026 se midió con las opciones por defecto en las dos versiones, en el scratchpad y sin tocar el arnés: la 3.26.0 escribe `In` y `org.cqframework:cql-to-elm-cli` 5.4.0, `IncludedIn` en las dos librerías. No se comenta #435. El engine 3.26.0 responde true en las dos, así que el cambio está en el ELM, no en el resultado de Java. Los tests siguen traduciendo cada expresión sola, porque el arnés está fijado en 3.26.0. Queda abierto actualizar el arnés: desde la 4.x, el group es `org.cqframework`.
1c14. **`in` y `contains` sobre una lista FHIR no convierten sus elementos:** `List<FHIR.string>{ FHIR.string { value: null as String } } contains (null as String)` da false, y Java da true: convierte cada elemento a String, y el que no tiene value es null. La lista conserva el objeto. Lo encontró Codex en la ronda adversarial del 006. Lo amplió el 008 (`testdata/snapshots/crossmodel.cql`): falta la conversión del modelo que inserta el traductor en `=` y `~` de listas, `IndexOf`, `in` y `contains` (anidados y sobre un retrieve), `includes`, `included in` e intervalos de Quantity FHIR (D14 a D31). Un primitivo FHIR, además, es igual a un valor de System en `except` y `union` (D35 a D39). fhirpath v1.16.6 acertaba algunos de estos casos por accidente, con la igualdad de FHIRPath; la regla del 008 (`types.ModelAgainstSystem`) los deja como en main. El arreglo es convertir en `sema`, donde convierte el traductor.
1c15. **`sema` acepta lo que el traductor rechaza por encadenar conversiones:** `Patient.birthDate during (null as FHIR.Period)` y `{ 1.0, 2.0 } includes Patient.multipleBirth` necesitan una conversión del modelo y otra del sistema. Java dice «Could not resolve call to operator». También `(null as FHIR.Period) included in (null as List<Interval<Date>>)`, que este motor decide como `In`: `inferTimingOperand` convierte el Period antes de decidir la sobrecarga, y la conversión del sistema se agrega después. Lo encontró Fable en la ronda adversarial y en la ronda 1 del review del 006.
1c16. **Una instancia `FHIR.dateTime { … }` no se convierte en un operador de tiempo:** `FHIR.dateTime { value: @2020-01-02T00:00:00Z } during First([Encounter]).period`, con un period, da el error «cannot compare dateTime»; Java da true. Pasa también en main. Lo encontró el 006 (`TestMembershipThroughTheModelFollowsTheSpecification`, D12 y D13).
1c17. **El ciclo que se cierra dentro de un pase especulativo no se reporta:** `define function F(): A` con `define A: F()`, y `define A: ({1, 2}) X aggregate R: R + A`, no dan ningún diagnóstico. `typeOfDefine` reporta el ciclo antes de salir del modo especulativo, y `reportf` lo calla. En ejecución, el motor da «maximum expression depth exceeded». Pasa también en main. Lo encontró Fable en la ronda 3 del review del 006.
1c19. **Del 007 (decisiones del usuario del 22-10-2026):**
  - una lista vacía no conserva su tipo en ejecución: `(List<Any>{ List<Integer>{} })[0] as List<String>` da `{}`, y el ADR 003 (decisión 8) dice null. Son D104 y D823 de `types`, `own defect`;
  - `Log` en coma flotante: `LogOfAScaledString` está a 1,16 del valor exacto, y Java también lo calcula mal.
1c21. **`gen` no termina con `-area fhirmembership -n 1500`:** llega a la línea 360 y sigue con la CPU ocupada sin escribir más, probablemente buscando expresiones distintas que ya no existen. Debería fallar con un mensaje. Lo encontró la diferencial del 008.
1c22. **Defectos anteriores que encontró la ronda adversarial del 008:**
  - `Size` de un intervalo continuo no suma el tamaño del punto (`units` D25; CQL Reference, Size);
  - `same hour as` con un offset parcial no sigue la decisión del 008 (`temporal` D22), ni `same or before`: `@2020-01-01T10+05:30 same or before @2020-01-01T05Z` da true, mientras que `<` y `=` dan null;
  - `ToDateTime` acepta una hora con offset (`'2020-01-01T10+05:30'`), y Java da null (`temporal` D31);
  - `12 'mg' / 0` da error y Java null;
  - `convert '25:00' to Time` y `convert '23:59:60' to Time` dan error, y Java null (`temporal` D45);
  - `ToQuantity(5)` da `5 ''` y debería dar `5 '1'`;
  - una Quantity FHIR sin value se lee como null;
  - `Interval[Q, Q] contains -5 'mg'` da error;
  - `2 'cm' * 2 'm'` da `4 'cm.m'` y Java `0.04 'm2'`;
  - `expand { Interval[100 'cm', 2 'm'] } per 50 'cm'` no coincide con ninguna referencia;
  - las duraciones de calendario se comparan a través del mes: `1 day < 1 month` y `1 month = 30 days` dan true, y por la guía de autores (Quantities) no son comparables; por eso `Interval[1 day, 1 month]` es válido, y en Java da error (`units` D56); también `1 month > 29 days` y `1 month ~ 30 days` dan true, y `GeometricMean({1 month, 30 days})` da `1 month` (review del 008);
  - un `time` FHIR no se lee como Time, con 60 o sin él: `First([Observation]).value as FHIR.time` da null con `valueTime` 23:59:58 (medido en main y en la rama);
  - `TestTheDivergencesAreConsistent` acepta que el plan nombre la definición en cualquier parte, aunque sea la homónima de otra área: D47 deliberada de `units` pasa porque el plan nombra D47 de `temporal` (la mutación del 008 tuvo que apuntar a D70, que se nombra una sola vez);
  - el comparador de respuestas compara los DateTime como texto, y `Z` no es `+00:00` (el 008 escribió D8 de `temporal` y D50 de `units` como Boolean por eso).
1c23. **Defectos anteriores que encontró el 009** (son iguales en main y en la rama; ver [009-decimal-1-6.md](009-decimal-1-6.md)):
  - **`ToDecimal('.5')` y `ToQuantity` de un string mal formado:** los toma el 010 ([010-to-quantity.md](010-to-quantity.md)).
  - **Ceros con escala.** `ToDecimal('0.000000000')` da `0`, y Java da `0.00000000`; el comparador de respuestas no lo ve (como en 1c18, tampoco ve la escala de un Decimal). Lo mismo pasa con `'0e-10001'` y `'0.00000000e5'`.
  - **Quantity FHIR que desborda** (`valueQuantity.value` de 1E20000). `.value` da null, `>` da error, y `1E100` se acepta fuera de rango.
  - **`cast` convierte** (backlog de #139). `cast` de un String a DateTime, a Integer o a un tipo FHIR da null cuando la conversión no tiene valor, y Java da error. Son D449 de `types` y D16, D52, D540, D627, D742, D744, D820 y D825 de `fhir`.
  - **Lo que encontró el 010 y no es de su alcance:**
    - **El ELM no importa ninguna conversión:** `ToQuantity`, `ConvertsToQuantity`, `ToDecimal` y los demás `To*` caen en el `default` de `elm/importer.go` y se evalúan como null, sin error (va con el punto 4);
    - **el literal Ratio es un String** en tiempo de ejecución: `(5:2) is String` da true, así que `ToQuantity(Ratio)` no puede dividir. Java convierte las unidades antes de dividir (`5 'mg':2 'g'` da `0.00250000 '1'`);
    - **un literal Decimal de más de ocho decimales** falla: `1.000000005` da «decimal overflow: too many fractional digits», y Java lo lee (D851 de `quantitystrings`);
    - **la unidad de un literal Quantity no se valida** (`5 'foo'`); el traductor de Java la rechaza;
    - **`5 '' = 5 '1'`** da null; Java da true.
    - **`ToQuantity` de un FHIR.Quantity sin cast** (`ToQuantity(O.value)`) da «cannot convert Quantity to Quantity»: no se aplica la conversión implícita de FHIRHelpers, y el tipo se escribe sin su espacio de nombres (Fable, ronda 3 del 010). `ToQuantity(O.value.unit)` da una lista de un null, sin verificar contra Java.
  - **Issue para fhirpath: [gofhir/fhirpath#235](https://github.com/gofhir/fhirpath/issues/235)** (decisión D del 009; relacionado con #232). Con decimal v1.6.0, `TestDecimalRange` falla porque `types.ParseDecimal` recibe de `NewFromString` un error genérico en vez de `ErrDecimalRange`, y además `NewDecimal("1e−2")` da 0.01. Hay que reportarlo antes de que fhirpath suba decimal.
1c18. **El comparador de respuestas no ve el tipo FHIR ni distingue Integer de Long:** `answers.Reference` iguala `FHIR.string "a"` con `FHIR.code "a"`, e `Integer 1` con `Long 1`. Lo encontró Codex en la ronda adversarial del 007 (sonda `compare_probe.go`).
1c12. **Un intervalo en un intervalo null:** `Interval[1, 2] in (null as Interval<Integer>)` da false, y la especificación (Interval Operators, Included In) dice null cuando un argumento es null. Es anterior al 005: la sobrecarga intervalo-intervalo de `in` llega a la rama de List Contains (`evalInContains`). Lo encontró Fable en la ronda 1 del review del 005.
1c. **`make evidence` llega a 1,08 GB de RSS y tarda 515 s** (el 23-10-2026, con la 5.4.0: 1,36 GB y 834 s). El oráculo de `types` no lo explica (0,5 s, 11 MB): hay que medir dónde está el costo y corregirlo.
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
