# Punto de reanudación

Al reanudar el trabajo o después de una compactación, este archivo es lo primero que se lee. Se actualiza al terminar cada paso. El proceso es el de `NUCLEO.md` §10 (`~/projects/personal/health-platform/workspace/NUCLEO.md`), y lo ejecuta la skill `fhir-incremento`. Rige para cql desde el 18-10-2026.

## Incremento activo

**021: las duraciones se miden en todo el calendario, del año 1 al 9999** (defectos B y C del 1c31). Rama `fix/duration-integer-range`. Pasos 1-3 hechos el 03-11-2026. La ronda adversarial (Opus independiente) encontró el centinela del año 1 (C), y el usuario dividió el plan: el null fuera del rango de Integer (A) pasa al 022. **Paso 4: aprobado el plan reducido por el usuario el 03-11-2026.** Pasos 5-8 hechos el 03-11-2026: área `durationrange` (1675 casos, 353 divergencias clasificadas, backlog 1c36), diferencial idéntica, eCQM 23+1, `make check` en verde (561 mutaciones); la ronda 1 del review convergió y se corrigieron 6 de los 7 hallazgos bajos. **Paso 9 hecho el 04-11-2026:** `make check` final en verde (562 mutaciones). Falta commit y PR, con la aprobación del usuario. Plan: [021-duration-integer-range.md](021-duration-integer-range.md).

**020 (cerrado): `difference in … between` cuenta fronteras y responde con incertidumbre.** Rama `fix/difference-uncertainty`. Pasos 1-2 hechos el 02-11-2026: 755 casos contra Java y cql-execution, tres defectos propios. Paso 3 hecho: ronda adversarial con un Opus independiente, 6 hallazgos; backlog 1c31-1c33. **Paso 4: aprobado por el usuario el 02-11-2026.** Pasos 5-7 hechos el 03-11-2026: área `differences` (776 casos, 403 divergencias clasificadas), diferencial idéntica, eCQM 23+1, `make check` en verde (553 mutaciones). Paso 8 hecho: la ronda 1 del code review (Opus independiente) convergió sin hallazgos medios. De los 4 bajos se corrigieron 2 (19 filas de semanas mal clasificadas y una errata); los tipos que sema acepta van al backlog (1c34), y `DifferenceBetween` se queda como envoltorio exportado. **Paso 9 hecho el 03-11-2026:** `make check` final en verde (553 mutaciones). **MERGEADO #174 (515b960) el 03-11-2026.** Plan: [020-difference-uncertainty.md](020-difference-uncertainty.md).
1c34. **sema acepta en `difference` (y en `duration`) tipos que la especificación no tiene** (code review del 020). Appendix B admite Date solo de años a días, Time solo de horas a milisegundos, y no tiene sobrecarga Time/DateTime. Aun así, `difference in hours between @T10:00 and @2012-01-01T10:00` da 17636832, `difference in days between @T10 and @T11` da 0 y `difference in hours between @2012-01-01 and @2012-01-02` da `[1, 47]`. Debería rechazarlo sema.
1c35. **El 022: una duración que no cabe en un Integer es null** (el defecto A del 1c31; ver [021-duration-integer-range.md](021-duration-integer-range.md)). Es una lectura: la regla del desborde está en *Arithmetic Operators*, y Duration solo dice «always an integer». Java da la vuelta y cql-execution devuelve el número de JS. Se suma a la consulta a HL7.
1c36. **La cota baja de una incertidumbre de un DateTime al año o al mes, en horas** (021, D1672 de `durationrange`). Ya pasa en main: `hours between @2019 and @2020-01-01T00:00:00.000` da `[27, 8763]`. Al offset de la evaluación (-03:00), el último momento de 2019 está a 0 horas del 1 de enero. Java da `[1, 8760]` y cql-execution `[3, 8763]`, y tampoco coinciden entre sí. `days` no tiene el defecto.

**018: un DateTime más grueso que la hora, al offset de la evaluación** (1c29, dividido en 018/019/020 por decisión del usuario del 02-11-2026). Rama `fix/mixed-date-duration`. DETENIDO: la causa está en parte en gofhir/fhirpath v1.16.6 (`WithDefaultOffset` rechaza la precisión menor que la hora). Issue gofhir/fhirpath#256 publicado el 02-11-2026. Próximo paso: esperar la versión de fhirpath; mientras tanto, el 020 (difference con incertidumbre). Plan: [018-date-to-datetime.md](018-date-to-datetime.md).

**017 (cerrado):** **017: las edades como duración, y el Date junto a un DateTime.** Rama `fix/age-duration`. Paso 1 (revisión previa) y paso 2 (plan) hechos el 02-11-2026: cuatro defectos propios medidos contra Java 5.4.0 y cql-execution 3.3.2 (sondas en `scratchpad/r017`). Paso 3 hecho: ronda adversarial con un Opus independiente (Codex sin cuota hasta el 04-11), 9 hallazgos aceptados. **Paso 4: aprobado y dividido por el usuario el 02-11-2026** (el Date junto a un DateTime en las duraciones pasa al 018). Paso 5-6: tests e implementación con un subagente Sonnet. **Decisión del usuario del 02-11-2026:** AgeInDays/AgeInWeeks con ToDateTime(birthDate) son inciertas para todo paciente, como la especificación y cql-execution (Java exacto, divergencia deliberada); eCQM sin cambio. Pasos 5-7 hechos: área `ages` (462 definiciones, 518 divergencias clasificadas con el oráculo independiente), 23 mutaciones del 017, diferencial idéntica, eCQM 23+1, backlog 1c30; make check antes de la ronda 1 en verde (537 mutaciones; se reanclaron 7 del 015); ronda 1 (Opus) convergió sin hallazgos medios; 4 bajos corregidos (Today al offset, estilo), 24 mutaciones del 017; make check final en verde (538 mutaciones). **Paso 9 hecho el 02-11-2026. MERGEADO #173 (372b228) el 02-11-2026.** Plan: [017-age-duration.md](017-age-duration.md).

**016 (cerrado):** **016: la incertidumbre como operando** (backlog del 015). Rama `fix/duration-uncertainty`. Plan aprobado el 31-10-2026 tras la ronda adversarial (6 hallazgos de Codex, aceptados) y la división en dos: el 016 corrige los operadores puntuales y de lista con una incertidumbre, la comparación de dos incertidumbres, el rango de un DateTime incompleto por offset y segundos con ms como una precisión, y reclasifica D46/D266/D297. **El 017** (siguiente) toma las edades por `DurationBetween` y el Date junto a un DateTime. Pasos 5-7 hechos y verificados el 02-11-2026: make check en verde (508 mutaciones), diferencial idéntica, eCQM 23+1; dos tests viejos que suponían UTC se fijaron con WithEvaluationTimestamp. Paso 8: Codex llegó a su límite de uso (vuelve el 04-11-2026 03:52); por decisión del usuario, hasta que Codex vuelva los reviews los hace un revisor Opus independiente (excepción registrada en el plan). Ronda 1 hecha: 2 hallazgos medios fuera del alcance escrito (al backlog), un defecto por ELM en `between` con límite incierto, corregido; make check antes de la ronda 2 en verde (510 mutaciones); ronda 2 hecha (1 medio introducido por el arreglo de la ronda 1, corregido; 23 mutaciones del 016); make check antes de la ronda 3 en verde (514 mutaciones); ronda 3 hecha: tope alcanzado con un hallazgo media-baja (`properly` con cota abierta, causa en la regla de punto de main). **Decisión del usuario del 02-11-2026:** cerrar el 016 y mandar el defecto al backlog (1c26). Paso 9 hecho: make check en verde (514 mutaciones), diferencial idéntica en las nueve áreas, 23 mutaciones del 016. **MERGEADO #172 (bd997db) el 02-11-2026.** El 017 sigue después. Plan: [016-duration-uncertainty.md](016-duration-uncertainty.md).

**015: `years between` y `months between` por períodos de calendario** (1c25 G3b). Rama `fix/years-between-leap-1c25`, sin commit. Review convergido en la ronda 2 de Codex el 31-10-2026 (ronda 1: un hallazgo medio, el codificador JS reconocía una incertidumbre por un campo; corregido y cubierto por D298). Diferencial contra main idéntica antes y después del review. `make check` final en verde el 31-10-2026 (491 mutaciones detectadas, 0 issues de lint). **015 cerrado:** #171 (1614601), mergeado el 31-10-2026 con el CI en verde. Plan: [015-years-between-leap.md](015-years-between-leap.md).

**014 cerrado:** #170 (e1a83e0), mergeado el 30-10-2026 con el CI en verde. Plan: [014-todate-toratio.md](014-todate-toratio.md).

**013 cerrado:** #169 (a623d74), mergeado el 29-10-2026. Plan: [013-null-donde-va-null.md](013-null-donde-va-null.md).

**012 cerrado:** #168 (4bf4d94), mergeado el 28-10-2026 con el CI en verde. Plan: [012-suite-cql-tests.md](012-suite-cql-tests.md). Issue para cql-tests: cqframework/cql-tests#172.

**011 cerrado:** #167 (2dc918f), mergeado el 28-10-2026. Plan: [011-literal-decimal.md](011-literal-decimal.md).

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
    - **un literal Decimal de más de ocho decimales:** lo toma el 011 ([011-literal-decimal.md](011-literal-decimal.md)), y el error se mantiene;
    - **la unidad de un literal Quantity no se valida** (`5 'foo'`); el traductor de Java la rechaza;
    - **`5 '' = 5 '1'`** da null; Java da true.
    - **`ToQuantity` de un FHIR.Quantity sin cast** (`ToQuantity(O.value)`) da «cannot convert Quantity to Quantity»: no se aplica la conversión implícita de FHIRHelpers, y el tipo se escribe sin su espacio de nombres (Fable, ronda 3 del 010). `ToQuantity(O.value.unit)` da una lista de un null, sin verificar contra Java.
  - **Issue para fhirpath: [gofhir/fhirpath#235](https://github.com/gofhir/fhirpath/issues/235)** (decisión D del 009; relacionado con #232). Con decimal v1.6.0, `TestDecimalRange` falla porque `types.ParseDecimal` recibe de `NewFromString` un error genérico en vez de `ErrDecimalRange`, y además `NewDecimal("1e−2")` da 0.01. Hay que reportarlo antes de que fhirpath suba decimal.
  - **Lo que encontró el 011 y no es de su alcance:**
    - **el control del literal Decimal es de evaluación:** un literal malo hace fallar toda `EvaluateLibrary`, y en una rama que no se evalúa no falla (decisión E del 011; `TestADecimalLiteralPastTheSuiteFailsWhereItIsEvaluated`). Llevarlo a la traducción (sema);
    - **el literal Quantity no tiene control de rango:** `10000000000000000000000000000.0 'mg'` se lee, y cql-execution lo rechaza por el rango de la especificación (D59 de `decimalliterals`, defecto propio);
    - **un literal con exponente en el ELM** (`1E-9`) no pasa por el control: solo los tests llaman a `elm.Import`;
    - **`(maximum Decimal) + 1.0`** da `100000000000000000000.99999999`, y el ADR 002 dice que lo que pasa del rango es null (Java da lo mismo);
    - **ToString de un literal con ceros a la izquierda** los conserva: `ToString(0001.0)` da `"0001.0"`, también en main, y Java da `"1.0"` (D39 de `decimalliterals`). El Decimal de fhirpath imprime el texto del que se leyó, y fhirpath v1.16.6 da lo mismo sola (`0001.0.toString()`). Hay que decidir de quién es antes de reportarlo;
    - **la suite pide 28 cifras enteras y la especificación da 20:** ya estaba en [cqframework/cql-tests#145](https://github.com/cqframework/cql-tests/issues/145), donde se agregó la evidencia. [#153](https://github.com/cqframework/cql-tests/issues/153) discute los tests en que se apoya la decisión del 011: revisarla en 1c24.
1c24. **(Cerrado en el 012, #168.) La suite de conformance estaba atrasada:** `conformance/testdata/SOURCE.tsv` fija cqframework/cql-tests en 727219f, 56 commits detrás de main (f724ee4, 19-10-2026 en −0600), con unas 1 200 líneas de tests nuevas en los 15 archivos. Lo encontró la revisión previa del 011.
1c25. **Las fallas de la suite al día** (cql-tests f724ee4; ver [012-suite-cql-tests.md](012-suite-cql-tests.md)). Cada test está en `knownFailures` (`conformance/known_failures_test.go`) con su grupo, y en `testdata/snapshots/suitegaps.cql` con las respuestas de Java y cql-execution. Un frente por incremento:
  - **G2. Error donde va null** (9): cerrado en el 013 (#169).
  - **Lo que el 013 dejó para después** (ver su plan, «Fuera del alcance»):
    - el borde de 32 bits sin Plan (ELM) o con un Integer no certero;
    - `+`, `-` y `*` de Decimal que pasan de `maximum Decimal`;
    - `width of` da la vuelta;
    - `except` lejos del borde (`Interval[1, 5] except Interval(1, 5]` da `Interval[6, 5]`) y en él (D83 de `limits`);
    - `maximum Quantity` es null;
    - `successor of` de una cantidad entera;
    - `Quantity mod` y `div` de un número;
    - fecha más una duración que pasa del año 9999;
    - el comparador lee un DateTime por su texto, y un DateTime sin offset no se compara con el de Java.
  - **G3a.** `month from DateTime(2012)` da 0 y no null («is not specified to the level of precision being extracted, the result is null»).
  - **G3b** (corregido en el 015, sin merge). `years between @2012-02-29 and @2014-02-28` da 1, y Java y cql-execution dan 2, como la especificación: un año sumado a un 29 de febrero cae en el último día de febrero (Author's Guide, Date and Time Arithmetic).
  - **Lo que el 015 deja para después** (ver su plan, «Fuera del alcance»):
    - la regla que decide cuándo una duración es una incertidumbre: D46, D266 y D297 de `durations` dan 2 donde cql-execution da `[1, 2]`, y el Author's Guide da `[6, 7]` para `days between @2017-08-07T17:00 and @2017-08-14T`;
    - los offsets de `years between` y `months between` (decisión C);
    - `CalculateAgeIn{Hours,Minutes,Seconds}At`, `CalculateAgeAt` de ELM y `CalculateAgeInYears` de dos argumentos;
    - un test ejecutable del codificador JS del arnés (hoy solo lo cubre la regeneración de las referencias).
  - **G3c.** `Distinct` y `Flatten` de una lista null dan `{}` y no null.
  - **G3d.** `Skip` con un número negativo da la lista entera, y la especificación dice vacía (Java da error).
  - **G3e.** `Coalesce` de seis argumentos se evalúa; sus firmas son de 2 a 5.
  - **G3f** (cerrado en el 014, #170). Faltaban `ToDate` y `ToRatio`.
  - **Lo que el 014 deja para después** (ver su plan, «Fuera del alcance»):
    - la equivalencia de Ratio, que debe compararse en cruz;
    - la escala en `ToString` de una Quantity, que un Ratio hereda;
    - `convert … to String` de una Quantity no pasa por `ToString`: `convert (1 'day') to String` da `1 day`, sin comillas, y no vuelve por `ToQuantity`. Es anterior al 014, que lo arregló solo para un Ratio;
    - el lector de `ToDateTime`, que acepta `'2014-01-01T'`, `'2014-01T12:30'` y `.1234`;
    - `cast` de String a Date o a Ratio;
    - las firmas de las conversiones en `sema`;
    - un tipo sin calificar bajo `using FHIR`: Java lee `Ratio` y `Quantity` como los de FHIR.
    - un literal Quantity con una unidad que no es de UCUM (`1 'foo'`) se lee, y el traductor de referencia lo rechaza; el literal Ratio lo comparte (D264 de `conversions`).
  - **G3g.** `div` de dos Quantity de la misma unidad conserva la unidad, y debe ser `'1'` (Java también la conserva).
  - **G3h.** La igualdad de tuplas con un elemento null da null, y es una conjunción (Java también da null).
  - **G3i.** Un límite cerrado en null no se lee como el mínimo o el máximo del tipo (8 tests; Java responde igual que el motor).
  - **G3j.** Un límite abierto en null da false y no null (2 tests; Java también).
  - **G1. CQL 2.0** (17): `MatchesFull`, `~in`, `~contains`, y Predecessor y Successor con la precisión del literal. Fuera del objetivo (ADR 001); se mueven si el objetivo pasa a 2.0.
  - **G5. La suite contra la especificación:** NotEquivalentNullNull marca inválido `null !~ null`, que la especificación da como false. Reportado: [cqframework/cql-tests#172](https://github.com/cqframework/cql-tests/issues/172).
  - **Lo que la ronda adversarial del 012 dejó para después:** en las salidas esperadas, un Long se lee como Integer, así que los tests de Long pasan solo por el valor (como 1c18); `valuesEqual` iguala `1 'm'` con `100 'cm'` e ignora si un límite null es abierto o cerrado; los tests `semantic` se aceptan con un error de evaluación; y DateTimeComponentFromTimezoneOffset (`versionTo="1.3"`) pasa porque el motor guarda `timezone from`, anterior a la 1.4: hay que decidir si se queda y anotarlo en el ADR 001.
1c18. **El comparador de respuestas no ve el tipo FHIR ni distingue Integer de Long:** `answers.Reference` iguala `FHIR.string "a"` con `FHIR.code "a"`, e `Integer 1` con `Long 1`. Lo encontró Codex en la ronda adversarial del 007 (sonda `compare_probe.go`).
1c12. **Un intervalo en un intervalo null:** `Interval[1, 2] in (null as Interval<Integer>)` da false, y la especificación (Interval Operators, Included In) dice null cuando un argumento es null. Es anterior al 005: la sobrecarga intervalo-intervalo de `in` llega a la rama de List Contains (`evalInContains`). Lo encontró Fable en la ronda 1 del review del 005.
1c. **`make evidence` llega a 1,08 GB de RSS y tarda 515 s** (el 23-10-2026, con la 5.4.0: 1,36 GB y 834 s). El oráculo de `types` no lo explica (0,5 s, 11 MB): hay que medir dónde está el costo y corregirlo.
1c26. **`properly` con una cota abierta, en la regla de punto** (ronda 3 del review del 016). `evalIntervalProperlyContainsPoint` (`eval/evaluator.go`) excluye solo las cotas cerradas: `1 properly included in Interval(0, 3)` y `2 properly included in Interval[1, 3)` dan true, y CQL 1.5.3 (ProperIn de punto) compara contra Start y End, así que dan false. La incertidumbre lo hereda: `U properly during Interval(0, 3)` da true. Necesita diferencial masiva contra Java antes de tocarlo.
1c27. **Lo que el 016 dejó fuera, de sus rondas de review:**
  - las frases de orden entre una incertidumbre y un intervalo (`U before Interval[2, 3]` da false, y es null);
  - una lista que contiene una incertidumbre (`1 in { U }` da false, y es null);
  - un intervalo dentro de un intervalo con una cota incierta (`Interval[U, 5] includes Interval[2, 3]` da error, como en main, y es true);
  - frases que Integer no tiene (`U includes 1`, `U overlaps …`, `starts`, `meets`), que leen U como intervalo, y `same as` entre dos Integers (31 filas de `uncertainty`), que sema debería rechazar, junto con los tipos que no concuerdan (`(days between …) in Interval[@2012-01-01, @2012-02-01]` da null);
  - `U less than 1 before 2`: false por AST y null por ELM;
  - huecos de tests: `U in Interval[null, 2]` y la guarda del offset en `uncertainTiming` sin fijar.
1c28. **`DurationBetween` normaliza los offsets por debajo de la hora** (revisión del 017). `days between @2025-12-31T23:00:00.000-03:00 and @2026-01-01T01:00:00.000Z` da 0; CQL 1.5.3 (Duration) normaliza «only when the comparison precision is hours, minutes, seconds, or milliseconds», así que daría 1. Medir Java antes de tocarlo.
1c29. **El 018:** el Date junto a un DateTime en `DurationBetween` y `duration in X of` se convierte con ToDateTime al offset de la evaluación (D21, D25 y D26 de `scratchpad/r017`; ver [017-age-duration.md](017-age-duration.md)). Y aparte, `@2012-01-01 = @2012-01-01T00:00:00.000Z` da null; Java y cql-execution dan false.
1c30. **`CalculateAgeInYears(null)` es ambigua** (017, D37 de `ages`): un null calza igual con la sobrecarga Date y con la DateTime, y el traductor de Java la rechaza; sema la acepta y el motor da null.
1c31. **Un Integer fuera de rango en `duration` y `difference`** (ronda adversarial del 020). `milliseconds between @2011-12-31T12:00:00.000 and @2013-01-01T12:00:00.000` da 31708800000, fuera del rango de Integer de CQL. Por la regla del #169, debería dar null. Java da la vuelta en 32 bits. Ver [020-difference-uncertainty.md](020-difference-uncertainty.md).
1c32. **`duration` de Time sin incertidumbre** (020). `temporalRange` no tiene caso para Time: `minutes between @T10 and @T11:30` da 90; Java da `[31, 90]` y cql-execution `[30, 90]`.
1c33. **`duration`/`difference in X of` ignoran las cotas abiertas y null** (020). `Interval(@2012-01-01, @2012-01-05)` da 4 días y las dos referencias 2; `Interval[@2012-01-01, null]` da null y las dos 2917556.
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
