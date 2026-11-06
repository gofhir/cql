# 019: un Date junto a un DateTime se convierte implícitamente

**Origen:** el 018 se dividió (decisión del usuario del 06-11-2026). Ver [018-date-to-datetime.md](018-date-to-datetime.md), «Revisión previa con fhirpath v1.16.7» y la ronda adversarial 1 del plan unido. El 019 va primero, sobre fhirpath v1.16.6, con la suite en verde. El 018 sube después a v1.16.7 y ubica el DateTime de día.

## Revisión previa (06-11-2026)

### `gofhir/server` (congelado)

No evalúa CQL: usa `gofhir/cql`. No hay nada que tomar.

### Especificación (CQL 1.5.3, https://cql.hl7.org/N1A/)

- **Developer's Guide, Implicit Conversions, tabla 3-J:** de Date a DateTime la conversión es «Implicit». «The language will also automatically apply implicit conversions when appropriate to produce a correctly typed expression».
- **La misma sección:** «ToDateTime(Date) Converts a Date value to a DateTime with all time components unspecified and the timezone offset of the request».

### Lo que escribe el traductor de Java (cql-to-elm de engine 5.4.0)

Sonda de 93 expresiones, cada operador en los dos órdenes, más una librería entera con parámetros. Los datos están en `scratchpad/r019/elmconv.tsv`, `table019.tsv` e `in/probe019*.cql`.

- **Convierte el lado Date con `ToDateTime`** en:
  - `=`, `!=`, `~`, `!~`, `<`, `>=` y `between`;
  - las frases de tiempo: `same day as`, `before`, `after day of`, `same or before`, `within … of`, `1 day or less before`;
  - `in`, `during`, `includes` y `contains` de un punto en un intervalo;
  - `in` y `contains` de lista;
  - los literales de lista y de intervalo, `if`, `case` y `Coalesce(Date, DateTime)`;
  - `overlaps`, `meets`, `starts`, `ends`, `during`, `includes`, `properly includes` y `=` entre intervalos, también con precisión (`overlaps day of`);
  - **`union`, `intersect` y `except` de intervalos**;
  - `=` de listas;
  - `between`, `difference`, `duration` y `Max` de una lista mixta;
  - los argumentos de una función con parámetro DateTime, o `Interval<DateTime>` (`F(@2020-03-01)`);
  - el default de un parámetro con tipo declarado (`parameter MP Interval<DateTime> default Interval[@2020-03-01, @2020-03-05]`).
- **No convierte `union` ni `intersect` de listas.** Los resuelve con `As` a un Choice: `{Date, DateTime}`.
- **Corrige al 018:** el plan viejo decía que `Coalesce` no convierte. Con `Coalesce(@2020-03-01, <DateTime>)` sí convierte. La sonda del 02-11, `Coalesce(null, @2012-01-01, <DateTime>)`, no lo hacía. Las dos formas entran en el área.
- **Firmas que el traductor rechaza:** Java rechaza `Interval<DateTime> during Date` y `Date includes Interval<DateTime>`, y el motor da null. Quedan fuera (ver más abajo).

### El motor en main contra las dos referencias (`table019.tsv`)

Las comparaciones mixtas ya coinciden. Con fhirpath v1.16.6 el día se lee tal como está escrito, del mismo modo que el Date. Difiere lo siguiente:

1. **El tipo del resultado:** donde las referencias dan DateTime, el motor da Date. Pasa en listas, `if`, `case`, `Coalesce`, `Max` y argumentos de funciones (D38, D42, D44, D46, D82, D88 y D89).
2. **`except` de intervalos mixtos:** el motor falla con «temporal values are specified to different precisions» (D68 y D69). cql-execution da null. Java también falla, con «led to an undefined Except result», porque los límites convertidos tienen precisiones distintas.
3. **`=` de listas mixtas:** el motor da null, cql-execution false, y Java coincide con el motor (D74 y D75).
4. **`@2020-03-01T10:00:00.000 1 day or less before @2020-03-01`:** el motor da null, Java false, y cql-execution coincide con el motor (D25).
5. **La forma de un DateTime sin offset** (`-03:00` en las referencias). Es la diferencia deliberada que decidió el usuario el 05-11-2026 en el 023. No cambia.

### Lo que hay en sema (ronda adversarial del 018, hallazgo 4, verificado en el código)

- `sema/convert.go` ya tiene `Convertible`, `Common` y las conversiones dentro de listas e intervalos, con `NumericPromotion` como modelo.
- **`convert` (`sema/check.go:451`) descarta Date→DateTime a propósito.** Es la decisión vieja del #132, que se revierte acá.
- Los operadores de comparación solo comprueban que se pueda convertir; no registran la conversión (`sema/operators.go`).
- **Los defaults de un parámetro con tipo declarado no se infieren** contra ese tipo (`sema/check.go`, alrededor de la línea 217).
- **Una llamada a una función de una librería incluida da Unknown** (`sema/query.go`), así que su argumento no se puede convertir.
- El evaluador aplica las conversiones registradas en `evalAndConvert` (`eval/evaluator.go`, alrededor de las líneas 475-494). Después de una función del modelo, `applyPlannedConversion` solo aplica una promoción numérica (M2).

## Alcance

1. **sema registra `DatePromotion`**, una conversión sin función, como `NumericPromotion`, directa o dentro de una lista o un intervalo. Se registra en cada lugar donde el traductor inserta `ToDateTime`, según la lista de arriba.
2. **No se registra en `union` ni `intersect` de listas**, que van a un Choice. Sí se registra en las operaciones de intervalos. En `Coalesce`, se sigue al traductor en cada forma medida.
3. **El evaluador aplica la conversión** con `funcs.ToDateTime` y `situate`, como ToDateTime explícito. Con v1.16.6, un DateTime de día no queda ubicado, igual que hoy. Eso lo resuelve el 018.
4. **El camino FHIR (M2):** `applyPlannedConversion` aplica también `DatePromotion` después de una función del modelo. Ejemplo: `FHIR.date` → `FHIRHelpers.ToDate` frente a un DateTime.
5. **Los argumentos de funciones (M3):** una función de la misma librería con parámetro DateTime, o `Interval<DateTime>`, convierte su argumento Date.
6. **El default de un parámetro con tipo declarado** se infiere contra ese tipo y se convierte.
7. **Las formas compuestas** (hallazgo 5):
   - las tuplas y las consultas (`where`, `let`, `return`, `sort`);
   - las listas anidadas y las listas de intervalos;
   - los agregados (`Min`, `Count`, `Mode`, además de `Max`);
   - `Coalesce` por su propio camino (`promoteRecorded`, `sema/infer.go:967`), que hoy filtra las promociones numéricas aparte.

   Lo que hace Java en cada una se mide con su ELM antes de implementar: `Coalesce` con null primero, las listas anidadas, las tuplas y el `except` de listas.
8. **Un chequeo de promociones en el arnés** (hallazgo 4). Hoy `ConversionsByDefine` deja fuera las conversiones sin función, y el fold solo cuenta FunctionRef de FHIRHelpers. Un test nuevo compara, definición por definición, los `ToDateTime` del ELM de Java con los `DatePromotion` que registra sema.
9. **Se revierten las decisiones viejas:**
   - `ListDateDateTime` pasa a true, como las dos referencias;
   - `TestANumericPromotionIsRecordedButCallsNothing` pasa a exigir el registro de Date→DateTime;
   - el comentario de `convert` cambia.

## Fuera del alcance

- **Las llamadas a funciones de una librería incluida:** sema les da Unknown. Resolver firmas entre librerías es trabajo de arquitectura aparte (hallazgo 4). Queda en el backlog con un test que documenta la pérdida.
- **El 018:** fhirpath v1.16.7, `CompareTemporal` y las frases de tiempo.
- **Las firmas que el traductor rechaza** (D29 y D31): son un frente de sema aparte, al backlog.
- `Max`/`Min` cuando las referencias difieren del motor por el instante (L1), y los backlogs 1c28 y 1c30.
- **Lo que la conversión no arregla** (hallazgo 1). Va a un frente nuevo del backlog, en el 018 o después: `union`, `intersect` y `except` de intervalos con límites de precisiones distintas; `=` de listas mixtas (D74 y D75); y `1 day or less before` con un DateTime de día (D25).
- **El ELM que exporta el motor** sigue sin escribir los `ToDateTime` implícitos. `elm.Translate` no lee el plan. Se suma al 1c37.
- **El importador de ELM:** las tres fallas del hallazgo 2, al backlog.
- La regla de segundos y milisegundos (hallazgo 2 del 018; pregunta 2 del borrador para HL7).

## Divergencias que se esperan

- **La forma de un DateTime sin offset:** deliberada, como en el 023.
- **Las comparaciones contra un DateTime de día:** v1.16.6 todavía no lo ubica, así que el área mide lo que las referencias ya resolvieron con el offset (`@2012-01-01 = @2012-01-01T00:00:00.000Z`). Esas filas son defectos propios con el 018 como causa, y su respuesta esperada se calcula en el oráculo.

## Criterio de salida

- **Sonda:** las filas de la clase 1 (el tipo del resultado) dan lo que dan las referencias, salvo D82. En D82, `Max` elige otro valor que las referencias aun con ToDateTime explícito: es el L1, fuera del alcance. Las de la clase 5 siguen deliberadas.
- **Fuera del criterio:** las clases 2 a 4 no las arregla la conversión. El prototipo de Codex lo muestra con ToDateTime explícito: `union` de intervalos falla, `intersect` da null, y D25 y `=` de listas siguen igual. Son defectos temporales y van al backlog (ver «Fuera del alcance»). Las filas D64 a D67 son de la misma clase.
- **intervalbounds:**
  - D590 pasa a `Uncertainty[21, 44]`, como pide `TestAnIntervalPromotedToDateTimeKeepsItsDate`, y ese test cambia.
  - D591 pasa a `Uncertainty[3, 4]`: coincide con cql-execution y se aleja de Java (4). Se clasifica con la regla del 016, no como regresión automática.
- **Área `mixeddates`:**
  - cientos de casos generados, con cada divergencia clasificada con evidencia;
  - un oráculo independiente en Python con reglas por operador y precisión (hallazgo 6), que chequea la mayoría;
  - el ELM de Java importado da lo mismo que la forma con `ToDateTime` explícito por el camino AST.
  - Toda fila donde no da lo mismo es un defecto del importador. Se registra fila por fila, va al backlog y no bloquea el criterio. Las conocidas son D25 (no deserializa un número), D74 (no deserializa un arreglo), D38 (da `{null, null}`) y D46 (da null; la forma explícita da `2020-03-01`).
- **Any y Unknown** (hallazgo 6): un valor sin tipo estático, por ejemplo de una función de una librería incluida, no se convierte. Un test fija lo que da hoy, para que no cambie por accidente.
- **sema:** un test por operador del alcance. La conversión queda registrada del lado Date, y no se registra en `union`/`intersect` de listas.
- **Diferencial contra main:** idéntica fuera de los casos mixtos, en todas las áreas. Cada cambio queda explicado.
- **eCQM:** el corpus con `ECQM_CONTENT_DIR`, sin tests salteados, y resultados por paciente y por población antes y después, que deben dar igual. Si algo cambia, lo acepta el usuario.
- **Cierre:** los tests nuevos fallan en main, cada oráculo tiene mutaciones y `make check` pasa.

## Tests y fuentes de datos

- **A mano, con la cita de N1A:**
  - las filas 1 a 4 de la sonda;
  - `ListDateDateTime`;
  - `Coalesce` en sus dos formas;
  - `union` de listas sin conversión;
  - `except` de intervalos;
  - una función con un argumento DateTime;
  - un default de parámetro;
  - un `birthDate` de FHIR frente a un DateTime.
- **Un test de sema por operador.**
- **Área `mixeddates`**, generada combinando:
  - la forma del Date: año, mes y día;
  - la forma del DateTime: de año a milisegundo, sin offset, con Z y con +14;
  - el operador;
  - el orden de los operandos.

  Se compara contra Java y cql-execution, y se importa el ELM de Java.
- **Una pérdida documentada:** un test fija lo que hoy da una función de una librería incluida.

- **El oráculo comprueba consecuencias, no solo conversiones** (hallazgo 7). Agrega invariantes entre caminos: una expresión implícita da lo mismo que su forma con `ToDateTime` explícito, cuya respuesta sale de las referencias, no del motor. Cubre límites abiertos y null, offsets positivos y negativos (de -12 a +14), y `difference` y `duration`, que sí cambian: `difference in hours between @2012-01-01 and @2012-01-02T05:00` pasa de `[18, 41]` a `[6, 29]`, que es lo que dan las referencias. Para eCQM, pacientes sintéticos en el borde de edad y del Measurement Period, comparando la pertenencia a cada población.

## Mutaciones previstas

- `DatePromotion` siempre false, y siempre true.
- `Coalesce` sin convertir.
- El chequeo de promociones del arnés que acepta un `ToDateTime` faltante.
- Un operador sin registrar.
- `union` de listas convertido.
- El default del parámetro sin convertir.
- El camino FHIR sin convertir.
- El área fuera del manifiesto y de las referencias.
- Una fila deliberada que cita un plan que no la nombra.
- Una divergencia sin clasificar.

## Dependencias

Ninguna nueva.

## Ronda adversarial 1 (Codex, 06-11-2026)

Codex prototipó el cambio mínimo: registrar la conversión en `convert` y aplicarla en `evalAndConvert`. La evidencia está en `/private/tmp/claude-501/cql019/probes/`. Verifiqué los hallazgos 1, 3 y 4 en sus logs y en el código.

| # | Severidad | Hallazgo | Resolución |
|---|---|---|---|
| 1 | alta | El criterio pedía arreglos que la conversión no produce. Aun con ToDateTime explícito, `union` de intervalos falla, `intersect` da null y D25 sigue en null. | **Aceptado, verificado** en `review-extra-prototype.log`. El criterio queda en la clase 1. Las clases 2 a 4 van al backlog como defectos temporales. |
| 2 | alta | La equivalencia AST/ELM depende del importador: D25 y D74 no deserializan, y D38 da `{null, null}`. | **Aceptado.** La equivalencia se mide en las filas que el importador lee. Las fallas van al backlog. |
| 3 | media | Cambian también D590 y D591 de `intervalbounds`, más los snapshots y la clasificación. | **Aceptado, verificado** en `review-prototype-tests.log`. D590 era lo previsto. D591 se clasifica con la regla del 016. |
| 4 | media | Registrar la conversión no cambia lo que compara el arnés ni el ELM que se exporta. | **Aceptado, verificado** en `sema/check.go:447` y en `exported ToDateTime=0`. Se agrega un chequeo de promociones contra el ELM de Java. La exportación pasa al 1c37. |
| 5 | media | Faltan formas compuestas: tuplas, consultas, listas anidadas, agregados y el camino propio de `Coalesce`. | **Aceptado.** Entran al alcance, y lo que hace Java en cada una se mide antes de implementar. |
| 6 | alta | No está definido qué pasa con Any, Unknown y las librerías incluidas. | **Aceptado.** No se convierte sin tipo estático, y un test lo fija. |
| 7 | media | El oráculo tiene que comprobar consecuencias: `difference` y `duration` sí cambian. | **Aceptado.** Se agregan invariantes, offsets de -12 a +14 y pacientes sintéticos por población. |

Codex no corrió eCQM ni pudo traducir con Java (sin acceso a Docker). Esas mediciones se hacen en la validación.

## Ronda adversarial 2 (Codex, 06-11-2026)

| # | Severidad | Hallazgo | Resolución |
|---|---|---|---|
| 1 | alta | La equivalencia AST/ELM seguía exigiendo arreglos del importador: D46 importa bien y da null. | **Aceptado, verificado** en `review-round2.log`. Cada fila donde no da lo mismo se registra como defecto del importador y no bloquea. |
| 2 | media | D82 (`Max`) seguía en el criterio de la clase 1, aunque `Max` es el L1, fuera del alcance. | **Aceptado, verificado:** con ToDateTime explícito, el motor sigue dando `10:00:00.000`. D82 sale del criterio. |
| 3 | media | Con v1.16.6, la mutación «sin `situate`» es equivalente: `WithDefaultOffset` no cambia un valor más grueso que la hora. | **Aceptado, verificado** en las tres precisiones y los cuatro offsets. La mutación pasa al 018. |

**Decisión del usuario (06-11-2026):** plan aprobado como quedó después de la ronda 2.

## Formas compuestas en el ELM de Java (paso 5, 06-11-2026)

Sonda de 29 expresiones (`scratchpad/r019/in/probe019c.cql`, tabla en `elmconv-c.tsv`).

- **Convierte** en:
  - `Coalesce` de listas;
  - las listas anidadas y las listas de intervalos;
  - las consultas: `where`, `return` y `let`;
  - `Min`, `Count`, `Mode`, `distinct` y `First`;
  - una lista con null en el medio;
  - `contains` de un intervalo con un límite null;
  - `includes` de lista;
  - `collapse` y `expand`.
- **No convierte** en:
  - `Coalesce(null, Date, DateTime)`, que da un Choice; el motor tiene que seguir esa forma;
  - las tuplas: `{ Tuple { a: Date }, Tuple { a: DateTime } }` no se convierte;
  - `except` de listas, que va a un Choice como `union` e `intersect`;
  - `sort by` de una lista de un solo tipo, donde no hay nada que convertir.
- **El traductor rechaza** `Tuple { a: Date } = Tuple { a: DateTime }`, y también `@2020-03-01 as DateTime` («cannot be cast»), porque `as` no convierte.

## Clasificación del área mixeddates (paso 7, 06-11-2026)

Las 500 divergencias nuevas del área tienen veredicto. Cada regla del clasificador comprueba su evidencia fila por fila, y ninguna fila que tiene oráculo (`testdata/mixeddates/oracle.tsv`) quedó como defecto propio. Para separar lo que resuelve el 018, el área se evaluó también con el prototipo del 018: fhirpath v1.16.7 más el salto de `CompareTemporal` para valores con offset más gruesos que la hora.

- **Deliberadas (la forma impresa):** el mismo valor, impreso con el offset de la evaluación (`-03:00`) o como `Object` en cql-execution. Sigue la decisión del usuario del 05-11 para la misma forma (023). Filas: D38, D39, D40, D41, D43, D45, D47, D49, D76, D77, D81, D95, D99, D107, D111, D112, D113, D116, D119, D124, D1061, D1062, D1065, D1066, D1069, D1070, D1086, D1090, D1094, D1133, D1134, D1135, D1136, D1138, D1140, D1142, D1144, D1171, D1172, D1176, D1190, D1194, D1202, D1206, D1207, D1208, D1211, D1214, D1219, D1380.
- **Defectos propios que resuelve el 018:** un DateTime más grueso que la hora no queda situado en el offset de la evaluación con fhirpath v1.16.6. El prototipo del 018 da la respuesta de la referencia.
- **Defectos propios para el backlog:**
  - `union`, `intersect` y `except` de intervalos con límites de precisiones distintas. El motor falla, o da null donde las dos referencias coinciden. En `union`, que es conmutativa, se espera lo que las dos referencias dan en el otro orden (D69, D1164).
  - `Max` entre un día y un instante de ese día (L1).
  - Las formas que el traductor rechaza y el motor evalúa: `Date as DateTime`, `during` e `includes` entre un intervalo y un punto.
  - `before` y `same or before` de un mes o un día contra un DateTime en `+14:00` (D467, D475, D515, D523): normalizado a `-03:00`, el DateTime cae el 29 de febrero, y la comparación queda decidida. El motor da null, y también el prototipo del 018.
- **Defectos de las referencias:**
  - cql-execution decide comparaciones que la regla de precisión deja en null; Java da null, como el motor.
  - cql-execution sitúa el Date convertido en UTC.
  - cql-execution da `false` entre operandos de la misma precisión.
  - Mode: cql-execution devuelve una lista. Collapse: cql-execution junta un día con un instante de ese día.
  - Java da un Interval donde va una incertidumbre, null donde va un rango, o un número.
  - Java tiene un límite corrido en uno.
  - Java lanza «Invalid Interval» sin saber si los límites están invertidos.
  - Java falla en `except` donde el operador da null.
  - Las dos referencias mueven el primer instante del valor grueso a `-12:00` y deciden. La misma comparación sin offset da null en Java.
- **Cambio del motor fuera del texto original del plan:** `Mode` desempata por la primera aparición (`funcs/list_advanced.go`), como Java. Sin eso, el resultado dependía del orden de un mapa.
- **Test retirado:** el de equivalencia AST/ELM por importador. El importador no deserializa varias filas (D25, D74, D38, D46). Va al backlog como defecto del importador, según el hallazgo 2 de la ronda 1.

## Code review, ronda 1 (Codex, 06-11-2026)

| # | Severidad | Hallazgo | Resolución |
|---|---|---|---|
| 1 | alta | Los defaults tipados se inferían en `collectParameters`, antes de registrar los parámetros, las funciones y las definiciones que vienen después. `parameter N Integer default M` con `M` declarado después daba «M is not defined»; en main da 1. Y `parameter PF DateTime default G(@2020-03-01)`, con `G` declarada después, no convertía. | **Aceptado, verificado.** Los defaults tipados se infieren en `promoteParameterDefaults`, cuando la biblioteca ya está registrada. Hay dos casos nuevos en `TestTheDefaultOfATypedParameterIsConverted`, que fallan sin el arreglo. |
| 2 | media | La comparación con el traductor solo mira si hay alguna conversión. Una pérdida parcial pasa, y un nombre repetido en `mixeddates.promotions.tsv` deja una definición sin comparar. | **Aceptado en parte.** Un nombre repetido ahora falla; la mutación de Codex, D1 cambiado por D0, se detecta. **Rechazado** comparar los conteos: el traductor escribe un nodo por límite de intervalo y uno por comparación de `between`, y medido así difieren 312 de 1097 definiciones. La pérdida parcial la detecta `sema/date_promotion_test.go`, como dice el mismo hallazgo. |
| 3 | baja | El oráculo y los pares comparan `answers.tsv` sin evaluar. | **Sin cambio.** El test de snapshots evalúa el área y compara con `answers.tsv`, así que la cadena oráculo → respuestas → motor está cerrada. La validación corre los tres juntos. |
| 4 | baja | La evidencia de D1067 decía «la comparación queda en null». El motor junta el Date y el DateTime en un `union` de listas. | **Aceptado, verificado.** Las filas de `union`, `intersect` y `except` de listas tienen un texto propio: el traductor no convierte, los valores son de dos tipos y el motor los junta. El texto del 018 dice ahora «stays null, or fails». |

## Code review, ronda 2 (Codex, 06-11-2026): convergió

Ningún hallazgo. Codex probó el arreglo del hallazgo 1 con defaults que nombran una definición posterior y con defaults tipados encadenados, en los dos órdenes. El error de un default sin resolver sale en su posición, y el de una definición referenciada sale una sola vez. Los defaults sin tipo se comportan como en main. El ciclo entre defaults tipados llega al límite de profundidad, igual que en main: es previo al 019. Las mutaciones del arreglo y del chequeo de nombres repetidos se detectan. Los 12 textos de operaciones de conjunto de listas coinciden con los dos archivos de referencia.
