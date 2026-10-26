# 008: fhirpath v1.14.1 → v1.16.6

**Fecha:** 24-10-2026 · **Rama:** `chore/fhirpath-1.16` · **Versión:** `fix:` (cambia respuestas del motor)

## Por qué

El motor importa solo `github.com/gofhir/fhirpath/types`, en v1.14.1. La última versión es la v1.16.6 (18-10-2026), 22 releases después. Arrastra `github.com/gofhir/ucum/v4` de v4.2.0 a v4.3.0, que es la última. La regla del usuario es usar siempre la última versión. `shopspring/decimal` v1.6.0 queda para el incremento siguiente.

De los cambios de esas 22 releases, los que tocan `types` son:
- #181, offsets y segundos intercalares;
- #186, `=` sobre colecciones, número contra cantidad, y una Quantity FHIR igual a una System.Quantity;
- #197, la aritmética toma la unidad que dice el texto;
- #200, las conversiones;
- #206, `~` en un primitivo con su elemento;
- #227, los sufijos de un elemento choice.

El resto (`funcs`, `eval`, el parser y el análisis estático) no lo usa el motor.

## Revisión previa (sondas del 24-10-2026, en el scratchpad)

- **`gofhir/server`:** no fija fhirpath a través de cql. Este incremento no le cambia nada, salvo lo que reciba cuando suba de versión.
- **Con v1.16.6 fallan tres tests.** Los snapshots de los corpus, las referencias Java y JS, y la conformance pasan. Los tres apuntan a dos cambios, las secciones 1 y 2. La ronda adversarial encontró más, en la sección 4.

### 1. La unidad del resultado de `+` y `-` (#197)

La especificación (CQL Reference, Apéndice B, Add y Subtract) dice: «The unit of the result will be the most granular unit of either input». Con v1.14.1, el motor daba la unidad del operando izquierdo; con v1.16.6, la más granular. Sonda `fu.cql`, con 22 casos:

| Expresión | Java 5.4.0 | cql-execution 3.3.2 | motor v1.14.1 | motor v1.16.6 |
|---|---|---|---|---|
| `2 'm' - 100 'cm'` | 1.0 'm' | 1 'm' | 1 'm' | 100 'cm' |
| `100 'cm' + 2 'm'` | 3.0 'm' | 300 'cm' | 300 'cm' | 300 'cm' |
| `width of Interval[100 'cm', 2 'm']` | 1.0 'm' | error | 1 'm' | 100 'cm' |
| `1 'g' + 500 'mg'` | 1.50 'g' | 1500 'mg' | 1.5 'g' | 1500 'mg' |
| `3 'm' - 1 'km'` | -0.9970 'km' | -0.997 'km' | -997 'm' | -997 'm' |
| `Sum({ 1 'g', 500 'mg' })` | 1.50 'g' | 1.5 'g' | 1.5 'g' | 1500 'mg' |
| `1 'L' - 250 'mL'` | 0.75 'L' | 0.75 'L' | 0.75 'L' | 750 'mL' |
| `1 day + 1 hour` | null | 25 'hour' | 1.0416… 'day' | 25 'hour' |
| `1 '[in_i]' + 1 'cm'` | 1.394 '[in_i]' | 3.54 'cm' | 1.3937… '[in_i]' | 3.54 'cm' |

- **Java** siempre da la unidad menos granular: es la regla de la comparación, no la de la suma.
- **cql-execution** no sigue una sola regla.
- **El motor con v1.16.6** sigue el texto de la especificación en los 22 casos. El valor es el mismo en los tres motores; cambia la unidad en que se expresa.

Los tests que esperaban la unidad del operando izquierdo son `TestSizeFollowsWidth` y `TestWidthAndExpandReportDifferentUnitsOnPurpose`. Sus valores esperados salieron del código propio, no de la especificación. Ningún corpus con referencias suma ni resta unidades distintas: hay un solo caso, `D50` de `arithmetic`, y es de la misma unidad. La comparación de respuestas compara la unidad como texto, así que lo habría visto. Es un hueco de cobertura.

### 2. Una Quantity FHIR igual a una System.Quantity (#186)

En v1.16.6, `ObjectValue.Equal` convierte una Quantity FHIR a System.Quantity cuando la compara con una cantidad o un número, por el mapeo de FHIR a FHIRPath. Es correcto en FHIRPath. Dos Quantities FHIR, en cambio, se siguen comparando hijo por hijo (FHIRPath §6.1.1, tipos complejos). La igualdad deja de ser transitiva: `{1 g}` FHIR es igual a `1000 'mg'` y a `1 'g'`, pero no a `{1000 mg}` FHIR.

El índice por cubetas de `distinct`, `union`, `intersect` y `except` (`valueIndex`) archiva un objeto por su JSON y nunca lo compara con las cantidades. Por eso `TestBucketedDistinctAnswersAsTheQuadraticOne` encuentra que el índice y la comparación por pares ya no coinciden.

**Qué dice CQL.** El traductor decide la conversión de forma estática (sonda `fq.cql`: la Observation `q`, de -5 mg, contra `-0.005 'g'`, en los tres motores):

| Expresión | ELM de Java | Java = JS | motor v1.14.1 | motor v1.16.6 |
|---|---|---|---|---|
| `Q = -0.005 'g'`, `in`, `contains`, `distinct` | `ToQuantity` | igual | = | = |
| `{ Q } = { -0.005 'g' }` (D7) | `Query` + `ToQuantity` | true | **false** | true |
| `IndexOf({ -0.005 'g' }, Q)` (D9) | `ToQuantity` | 0 | **-1** | 0 |
| `{ Q } except { -0.005 'g' }` (D1) | `As List<Choice<FHIR.Quantity, System.Quantity>>`, sin conversión | `{ Q }` | = | **`{}`** |
| `{ Q } intersect { -0.005 'g' }` (D5) | ídem | `{}` | = | **`{ Q }`** |
| `{ Q } union { -0.005 'g' }` (D8) | ídem | los dos | = | **solo `Q`** |

Donde el traductor no inserta la conversión, los elementos son de tipos distintos en ejecución (FHIR.Quantity y System.Quantity), y las dos referencias los dan por distintos. La igualdad entre elementos de CQL es responsabilidad de este motor, no de fhirpath: fhirpath implementa la de FHIRPath, y la implementa bien. Con v1.16.6, el motor tomó la de FHIRPath en tres operaciones (D1, D5, D8) y se alejó de las dos referencias. En otras dos (D7, D9) acierta, pero por casualidad: falta la conversión que inserta el traductor, y con v1.14.1 esas dos ya eran defectos propios.

### 3. La diferencial generada

Seed 8008, 1500 expresiones por área, en las nueve áreas del generador, en Java y en el motor con las dos versiones. Ver «Diferencial previa».

### 4. Lo que los tests no ven (ronda adversarial, medido)

Los tres tests que fallan no son todo lo que cambia. Codex y Fable midieron 280 expresiones más con Java 5.4.0, cql-execution y el motor en las dos versiones. Hay cambios de v1.14.1 a v1.16.6 que ningún corpus ni ningún generador ejercitan:

| Cambio | v1.14.1 | v1.16.6 | Java | Lectura |
|---|---|---|---|---|
| `ToDateTime('2020-01-01+05:00')` | valor | null | null | se acerca |
| `@2020-01-01T10+05:30 > @2020-01-01T04Z` y `=`, `in`, `contains` con offsets | null | el de Java | — | se acerca (#181) |
| `@2020-01-01T10+05:30 < @2020-01-01T05Z` | null | **true** | false | se aleja |
| `~` entre dos `HumanName` con extensiones en primitivos distintos | true | false | false | se acerca (#206) |
| `1 'Cel' + 1 'K'` | -271.15 'Cel' | null | null | se acerca |
| `@2016-12-31T23:59:60Z`, `DateTime(…, 60)`, `Time(23, 59, 60)`, `ToTime('23:59:60')`, `ToDateTime('…:60Z')` | error o null | **valor** | error o null | se aleja |
| un `effectiveDateTime` de datos FHIR en `23:59:60Z` | — | 23:59:60, y `second from` da 60 | 2017-01-01T00:00:00Z | se aleja |
| `1 month + 1 day`, `1 year + 1 month` | — | `31 'day'`, `13 'month'` | null | se aleja |
| `@2020-01-31 + (1 month + 1 day)` | 2020-02-29 | **2020-03-02** | null | se aleja |
| `predecessor of (1 'g' + 1 'mg')` | 1.00099999 'g' | 1000.99999999 'mg' | 1.00099999 'g' | otro valor físico |
| `Interval(2 'm' - 100 'cm', 2 'm'] = Interval[1.00000001 'm', 2 'm']` | true | false | true | otro valor físico |

- **El segundo intercalar no es CQL.** La referencia dice: «**ss** - A full two digit second value (00..59)». El rango de DateTime termina en `@9999-12-31T23:59:59.999` y el de Time en `@T23:59:59.999`. fhirpath lo acepta por FHIRPath. Este motor ya rechaza `@T23:59:60`.
- **Un mes no son 30 días en CQL.** La guía de autores dice: «Durations above days (and weeks) are calendar durations that are not comparable with definite quantity UCUM duration units». fhirpath aplica el mes de 30 días de UCUM, que es la regla de FHIRPath 3.0.0.
- **`successor` y `predecessor` dan un paso de 10⁻⁸ en la unidad del resultado.** Con la unidad más granular, el paso es otro valor físico. Es una consecuencia de la regla de la especificación, y no un defecto aparte.

**Observación de arquitectura.** El motor delega en `fhirpath/types` la igualdad, la aritmética de cantidades y el parseo temporal. fhirpath sigue FHIRPath 3.0.0, y donde FHIRPath y CQL difieren, cada bump trae la diferencia al motor sin que nada falle: la igualdad entre modelo y System, el mes de UCUM, el segundo intercalar. En este incremento, cada diferencia se resuelve en el motor, con la regla de CQL. Que el motor declare en un solo lugar dónde la semántica de CQL se aparta de la de FHIRPath es una decisión de arquitectura: se propone como ADR aparte, no se hace aquí.

### 5. Qué llega al motor

El motor importa `fhirpath/types` y nada más: ni `funcs`, ni `eval`, ni `TypedChildren`, ni `IsValuelessQuantity`. La búsqueda está en las sondas de Codex.

| PR de fhirpath | Llega | Cómo |
|---|---|---|
| #181 offsets y segundos intercalares | sí | parseo y comparación de DateTime y Time |
| #186 igualdad | sí | `ObjectValue.Equal` y `Quantity.Equal` |
| #197 unidad de la aritmética | sí | `Quantity.Add` y `Quantity.Subtract`. El redondeo de Math de FHIRPath no llega: `Round` es del motor |
| #200 conversiones | en parte | `ToDateTime('2020-01-01+05:00')` |
| #206 `~` de primitivos | sí | `ObjectValue.Equivalent` |
| #227 sufijos de choice | no se observó | `valueQuantity.value` da -5 en las dos versiones |
| #183, #189, #191, #193, #195, #200 (funciones), #202, #204, #209, #211, #217, #220, #226 | no | son de `funcs`, `eval`, el parser o el análisis estático |

## Alcance

1. Subir `github.com/gofhir/fhirpath` a v1.16.6, y `github.com/gofhir/ucum/v4` a v4.3.0, que es indirecta.
2. **La unidad de `+` y `-` sigue la especificación** (CQL Reference, Add y Subtract: «the most granular unit of either input»), como ya lo hace fhirpath. Si las dos unidades son igual de granulares, se usa la del operando izquierdo, como fhirpath. En el código del motor no cambia nada; cambian sus tests y el corpus:
   - `TestSizeFollowsWidth` y `TestWidthAndExpandReportDifferentUnitsOnPurpose` esperan la unidad más granular y citan la especificación;
   - un tramo de `runtime.cql` con sumas y restas de unidades distintas, en los dos órdenes, y lo que depende de ellas: `Sum`, `Avg`, `Median`, `successor`, `predecessor`, `.value`, `width`, `Size`, `expand`, `collapse`, temperatura y unidades adimensionales;
   - las respuestas de Java (la unidad menos granular) y de cql-execution van a `runtime.divergences.md` como decisiones registradas. **Decisión del usuario.**
3. **Un elemento del modelo no es igual a un valor de System en la igualdad entre elementos** (`sameValue`, `elementEquality` y `valuesEqual` de `types`). Va en un solo lugar de `types`, que llaman los tres. El resultado es el de main en todos los casos medidos:
   - D1, D5 y D8 (`except`, `intersect`, `union`) vuelven a coincidir con las dos referencias;
   - el índice por cubetas vuelve a coincidir con la comparación por pares;
   - lo que v1.16.6 acierta por accidente vuelve a ser el defecto de main: `=` de listas, `IndexOf`, `in` y `contains` anidados, `Q in {Q}` y `Interval[Q, Q] = Interval[…]`. El arreglo correcto es convertir donde convierte el traductor, en `sema`: es 1c14 del backlog, y se amplía con estos casos;
   - **decisión del usuario,** frente a la alternativa de no poner la regla. Esa alternativa conserva los aciertos accidentales, pierde D1, D5 y D8, y obliga a que el índice reproduzca una igualdad no transitiva. Partir la regla por operación tampoco sirve, porque `in` e `IndexOf` pasan por `sameValue` (`listContainsValue`): la igualdad volvería a ser distinta según el operador, que es lo que `sameValue` vino a unificar.
4. **El segundo intercalar se rechaza** en los literales, en `DateTime()` y `Time()` (error, como Java) y en `ToDateTime` y `ToTime` (null, como Java), por el rango de la especificación. Para un dato FHIR con `:60` se mide primero lo que da main. Si main no lo aceptaba, se conserva su respuesta; si lo aceptaba, va al backlog con un test que lo documenta.
5. **Un mes o un año no se suman ni se restan con días o unidades menores:** el resultado es null, como en Java, según la guía de autores. Con un Date o un DateTime, la suma por partes (`(@2020-01-31 + 1 month) + 1 day`) no cambia.
6. **Offsets:** un tramo de corpus con comparaciones entre offsets enteros y parciales, a toda precisión, con Java y JS. Se clasifica cada caso. `@…T10+05:30 < @…T05Z`, que da true donde Java da false, se lleva a la especificación primero. Si el defecto es de fhirpath, se reporta allá con el borrador del issue y no se rodea: queda como pérdida conocida, con un test.
7. Corregir el punto de reanudación: #163 ya está mergeado.

## Fuera del alcance

- **Las conversiones del modelo que faltan** (1c14, ampliado): `=` de listas, `IndexOf`, `in`/`contains` anidados y sobre retrieves, intervalos de Quantity FHIR, `~` de listas en un sentido, y los primitivos FHIR en `except` (`{Patient.gender} except {'male'}` da `{}`; Java conserva el elemento).
- **Defectos anteriores que encontró la ronda**, para el backlog (1c22):
  - `12 'mg' / 0` da error y Java null;
  - `ToQuantity(5)` da `5 ''` y debería dar `5 '1'`;
  - una Quantity FHIR sin value se lee como null;
  - `Interval[Q, Q] contains -5 'mg'` da error;
  - `2 'cm' * 2 'm'` da `4 'cm.m'` y Java `0.04 'm2'`;
  - `expand { Interval[…] } per 50 'cm'` no coincide con ninguna referencia;
  - `Size` de un intervalo continuo no suma el tamaño del punto;
  - `ToDateTime` acepta una hora con offset (`'2020-01-01T10+05:30'`) y Java da null.
- `Mode` con cantidades de distinta unidad: ya está en 1c5.
- El generador que se cuelga (1c21).
- `shopspring/decimal` v1.6.0: es el incremento siguiente. R4B y R5: el motor no cambia de modelo.

## Criterio de salida

- `go.mod` con fhirpath v1.16.6 y ucum v4.3.0.
- **Ninguna respuesta se aleja de las referencias respecto de main**, en los corpus, en la diferencial generada (seed 8008) y en los tramos nuevos. Las únicas excepciones son las decisiones registradas de la unidad. Lo verifica la diferencial main contra rama, con `-baseline`, sobre todo.
- `TestBucketedDistinctAnswersAsTheQuadraticOne` en verde, con objetos que parecen Quantity en su pozo, sin cambiar el índice.
- Tramos nuevos en el corpus, cada uno con las respuestas de Java y de JS: igualdad entre modelo y System (las sondas `fq` y las de la ronda, con `~`), unidades, duraciones de calendario, segundos intercalares y offsets. Cada tramo tiene un test que exige que el área tenga ese tipo de expresión. Así, un corpus que pierde el tramo falla, y la cifra de diferencias no puede quedar igual por falta de casos.
- `make check` y `sh scripts/references.sh check` en verde, y `make evidence` con su reporte.

## Tests y mutaciones

- **Tests primero,** con los valores esperados de la especificación o de las referencias, citados:
  - unidades: valor y unidad, en los dos órdenes, con la cita de Add y Subtract;
  - `except`, `intersect`, `union`, `distinct`, `=` de listas y de tuplas, `~`, `in`, `contains` e `IndexOf`, con una Quantity FHIR frente a una System.Quantity, en los dos órdenes y anidados. Los que hoy son defecto quedan como `own defect`;
  - el segundo intercalar en las seis formas de construirlo;
  - mes y año más días, horas o minutos, solos y sumados a un Date;
  - el pozo de la propiedad del índice, con `{"value":1,"unit":"g"}`, `{"value":1,"code":"g","system":"http://unitsofmeasure.org"}` y `{"value":1,"unit":"1"}`.
- **Mutaciones** (`scripts/mutations/008-fhirpath.sh`), separadas entre las de traducción y las de ejecución:
  - quitar la regla de modelo contra System → fallan `except`, `intersect` y `union` y la propiedad del índice;
  - la regla solo en `sameValue` → fallan las listas y las tuplas;
  - forzar la unidad del operando izquierdo → fallan los tests de unidad y el tramo;
  - aceptar el segundo 60 → fallan las seis formas;
  - dejar que un mes se sume con días → falla el tramo de calendario;
  - borrar una decisión registrada → falla el test del corpus;
  - borrar un tramo → falla el test de cobertura.
- **Las mutaciones de 001 a 007** siguen detectándose.

## Excepciones a la regla 5

Ninguna. La regla de modelo contra System distingue por la representación (`*ObjectValue`), no por un nombre FHIR. Los límites del segundo y la regla del mes salen de la especificación de CQL.

## Dependencias

Ninguna nueva. Cambian de versión fhirpath y, por arrastre, ucum.

## Riesgos

- **Otro cambio de `types` que nadie mide:** los tramos nuevos y la tabla de alcance de la sección 5. La diferencial generada no basta, porque sus generadores no producen estos valores.
- **Una regla del motor que tape un defecto de fhirpath:** cada regla cita la especificación de CQL, donde CQL difiere de FHIRPath. Si fhirpath se equivoca en FHIRPath, se reporta allá.
- **Que el criterio «nada se aleja respecto de main» esconda aciertos accidentales:** quedan enumerados en 1c14.

## Diferencial previa (24-10-2026)

Seed 8008, 1500 expresiones por área, salvo `fhirmembership`, que tiene 350. Se corrió en Java 5.4.0 y en el motor con v1.14.1 y con v1.16.6, con `compare -each` y su reporte. Son 12.350 expresiones en total.

| Área | Difieren de Java (v1.14.1) | Difieren de Java (v1.16.6) |
|---|---|---|
| arithmetic | 127 | 127 |
| integer | 489 | 489 |
| scope | 0 | 0 |
| types | 177 | 177 |
| fhir | 993 | 993 |
| codes | 235 | 235 |
| codesets | 171 | 171 |
| membership | 11 | 11 |
| fhirmembership | 49 | 49 |

Los reportes de las dos versiones son idénticos en las nueve áreas, con las mismas definiciones y los mismos valores. Los dos cambios de la revisión previa no aparecen porque ningún generador mezcla unidades ni pone una Quantity FHIR junto a una System.Quantity. Los cubren las sondas `fu.cql` y `fq.cql`, que pasan a ser corpus del repositorio. Las cifras de diferencias contra Java son de seeds nuevas, sin clasificar, y no son objeto de este incremento: solo importa que no se muevan.

**Defecto del generador:** con `-area fhirmembership -n 1500`, `gen` no termina. Llega a la línea 360 y sigue doce minutos de CPU sin escribir más, probablemente buscando expresiones distintas que ya no existen. Va al backlog (1c21): debería fallar con un mensaje en vez de colgarse.

## Ronda adversarial (Codex y Fable, 24-10-2026)

Los dos atacaron el plan sobre copias con v1.16.6, cada uno con sus sondas en Java 5.4.0 y cql-execution: 55 expresiones Codex, 212 Fable. Cada hallazgo se verificó contra sus resultados, y los temporales también con una sonda propia (`ls.cql`).

| # | Hallazgo | Quién | Resolución |
|---|---|---|---|
| 1 | La regla vuelve a romper lo que v1.16.6 acierta (`=` de listas, `IndexOf`, `in` y `contains` anidados, `Q in {Q}`, intervalos) | los dos | **Aceptado en parte.** Son los mismos defectos de main (con la regla, `guard` = `old` en todas las filas de Codex). Se enumeran en 1c14. Partir la regla por operación se rechaza: `in` e `IndexOf` pasan por `sameValue`. La regla queda como decisión del usuario, con su alternativa (alcance 3) |
| 2 | Hay caminos de igualdad fuera del plan (`~`, `Mode`, intervalos) | los dos | Aceptado: tests de `~` en los dos órdenes. `Mode` ya está en 1c5 |
| 3 | Hay cambios de #181, #200 y #206 que la diferencial no ve | los dos | Aceptado: sección 4, alcance 4 a 6, tramos nuevos. D40 de Codex se reclasificó: Java da null porque su `ToDateTime` no acepta `T10+05:30`; con literales, Java da true, igual que v1.16.6 |
| 4 | Falta la tabla de qué llega al motor | Codex | Aceptado: sección 5 |
| 5 | «Solo cambia la unidad» es falso para `successor`, `predecessor` y `.value` | los dos | Aceptado: sección 4 y el tramo de unidades |
| 6 | Las duraciones de calendario no son una cuestión de granularidad | Fable | Aceptado: alcance 5, con la cita de la guía de autores |
| 7 | Los tests de aceptación pueden conservar defectos (`expand` por 50 cm) | Codex | Aceptado: al backlog 1c22. No cambia con el bump |
| 8 | Los generadores no producen estos valores, así que la cifra igual no prueba nada | Codex | Aceptado: un test de cobertura por tramo |
| 9 | Citar la sección Add, y decir qué pasa con un empate | Fable | Aceptado: alcance 2 |
| 10 | Defectos anteriores (`/ 0`, `ToQuantity(5)`, Quantity sin value, `cm.m`) | Fable | Aceptado: al backlog 1c22 |

## Aprobación (24-10-2026)

El usuario aprobó el plan con sus dos decisiones:
- **Unidad de `+` y `-`:** la de la especificación, la más granular. La diferencia con Java se registra como decisión.
- **Igualdad entre modelo y System:** la regla, como en main. Los aciertos accidentales de v1.16.6 van a 1c14.

El alcance de los puntos 4 a 6 también quedó aprobado, sin dividir.

## Decisión del usuario: offsets parciales (25-10-2026)

Comparar un DateTime cuyo offset no es de horas enteras, a una precisión más gruesa que el minuto (`@2020-01-01T10+05:30 < @2020-01-01T05Z`, D12 a D27 de `temporal`), da **null**, como en main. La especificación compara precisión por precisión y no dice cómo normalizar un offset parcial. Normalizado a UTC, `10+05:30` queda en 04:30 y tiene un valor para el minuto que `05Z` no tiene. La regla de Equal y Less dice: «If one input has a value for the precision and the other does not, the comparison stops and the result is null». Cada motor responde otra cosa:
- Java lleva el segundo valor al offset del primero;
- fhirpath v1.16.6 lo lee como el instante 04:30Z;
- cql-execution sigue una tercera regla.

Se registra como decisión, en un solo lugar del motor: `compareAcrossPartialOffsets` (types/temporal_compare.go). Cada valor es un tramo de su reloj, tan largo como su precisión, y el orden se conoce solo cuando los tramos no se tocan.

**Extensión en el review (ronda 1):** la primera versión aplicaba la regla solo entre dos horas, y una hora contra un minuto o algo más fino quedaba ordenada por fhirpath (`@2020-01-01T10+05:30 < @2020-01-01T05:00Z` daba true, y main null). Ahora la regla se aplica a una hora contra cualquier precisión desde la hora (D34 a D42 de `temporal`). En 8 de los 9 casos repite a main. En D40 (`> @2020-01-01T04:00Z`) da true, como Java, porque el minuto 04:00Z queda fuera de la hora 04:30Z–05:30Z; main daba null.

## Mutaciones (25-10-2026)

`sh scripts/mutations/008-fhirpath.sh` aplica 26 mutaciones, cada una en una copia, y exige que el comando que nombra falle con un texto dado. Cubren la regla de modelo contra System en sus tres lugares, la unidad de la suma, las duraciones de calendario en `+`, `Sum` y `Width`, el segundo 60 en sus seis formas y en los datos FHIR, los offsets parciales, y las tres áreas: una divergencia borrada, una que el plan no nombra, un defecto propio con la respuesta del motor, el área sin datos FHIR y un número de definiciones que no coincide.

En la primera corrida sobrevivieron tres: sacar la semana de las unidades menores que el mes, y volver a sumar a través del mes en `Sum` y en `Width`. El corpus no tenía esos casos, porque main ya sumaba mes con semana. Se agregaron D52 a D57 a `units`, y ahora se cazan las 26. Las de 001 a 007 se siguen cazando todas.

Después de la ronda 1 del review son 48: una por palabra clave de calendario (16), la unidad de la dispersión, las tres conversiones del segundo 60 y dos de los offsets con precisiones mixtas. Se cazan todas, y las de 001 a 007 también (285).

En el cierre se agregó una más: sacar un área entera de `snapshotAreas` no fallaba en ningún test, y el criterio de salida lo pide. Ahora lo exige `TestEveryCorpusIsInTheManifest`. Son 51.

## Segundo intercalar en datos FHIR (alcance 4, medido)

Medido con `deceasedDateTime` 2016-12-31T23:59:60Z, en main, en la rama sin la regla, en Java 5.4.0 y en cql-execution 3.3.2:

| Expresión | main | rama sin la regla | Java | cql-execution |
|---|---|---|---|---|
| `Patient.deceased as FHIR.dateTime` | null | 2016-12-31T23:59:60Z | 2017-01-01T00:00:00Z | el valor FHIR, con 60 |
| `… > @2016-12-31T23:59:59Z` | null | true | true | null |
| `second from …` | null | 60 | 0 | null |
| `… + 1 second` | null | 2017-01-01T00:00:01Z | 2017-01-01T00:00:01 | null |

Las conversiones también lo dejaban pasar: `ConvertsToDateTime` daba true, `convert … to DateTime` devolvía un String y `convert … to Time` un Time con 60. Ahora pasan por los mismos analizadores y repiten a main: false, null y error (D43 a D46 de `temporal`; Java da false, null y null).

Main no lo aceptaba: fhirpath v1.14.1 no podía leerlo y lo dejaba como texto. Se conserva esa respuesta: `situate` (eval/planned_type.go), por donde pasa cada DateTime leído de JSON, devuelve el texto. Lo fija `TestALeapSecondReadFromFHIRDataIsNoCQLValue`, y la mutación «a leap second read from FHIR data». Un `time` FHIR no se lee como Time ni siquiera sin el 60, así que no se alcanza (1c22).

## Review, ronda 1 (25-10-2026)

Codex y Opus, este último en lugar de Fable porque Fable llegó a su límite de uso. Cada hallazgo se verificó con una sonda en main y en la rama, y en Java cuando hacía falta el valor esperado.

| # | Hallazgo | Quién | Resolución |
|---|---|---|---|
| 1 | `StdDev` y `PopulationStdDev` ponen a la raíz la unidad del primer valor: `{1 'g', 500 'mg'}` da `250 'g'` (main 176.78 g, Java 0.25 g) | Codex | **Confirmado (alto).** La raíz va en la unidad de las desviaciones, `250 'mg'` en los dos órdenes. Tests con valores de Java, D69 a D72 de `units` y una mutación |
| 2 | El segundo 60 entra por `ConvertsToDateTime` y por `convert … to DateTime/Time` | los dos | **Confirmado (medio).** Por los analizadores de CQL; D43 a D46 de `temporal` y tres mutaciones |
| 3 | Offsets parciales entre una hora y un minuto o algo más fino | Opus | **Confirmado (medio).** Regla extendida (sección de la decisión), D34 a D42 y dos mutaciones |
| 4 | Ninguna prueba cubre la mayoría de las palabras clave de calendario | Opus | **Confirmado (medio).** D58 a D68 de `units` y una mutación por palabra clave (16) |
| 5 | Textos de evidencia equivocados en D35, D39, D40 y D57 | Opus | **Confirmado (bajo).** D35 (Java) y D57 (JS) son la decisión de la unidad; D36, D37, D39 y D40 (JS) son otra grafía de la unidad; D40 (Java) es un defecto de la referencia, que da null en `1 year + 2 years` |
| 6 | `AddQuantities` en la desviación no se alcanza: `Sum` ya falla antes | Opus | **Confirmado (bajo).** Vuelve a `Subtract` |

Los defectos que Opus encontró y que también tiene main van a 1c22: `1 month = 30 days`, `1 month > 29 days`, `GeometricMean({1 month, 30 days})`, y `same or before` con un offset parcial, de la familia de D22.

## Review, ronda 2 (25-10-2026)

Codex no dejó hallazgos: corrió la suite, sus sondas y las 48 mutaciones. Opus confirmó los seis arreglos contra main, con unas 70 expresiones, y dejó tres hallazgos bajos, todos aceptados:
- el tramo de un segundo en `clockSpan` no tenía caso: D47 de `temporal` y una mutación;
- el borde del tramo posterior no tenía caso: D48 de `temporal` y una mutación;
- la conversión de la desviación en `spreadOfQuantities` no se alcanza, porque la media ya está en la unidad más granular: se borró, con un comentario.

La ronda converge: ningún hallazgo de severidad media o mayor.

## Clasificación de las áreas nuevas (25-10-2026)

Las divergencias de `units`, `temporal` y `crossmodel` respecto de Java y de cql-execution, después de las reglas del alcance. Contra main (`compare` sobre las tres áreas, en el motor con v1.14.1), ninguna respuesta se aleja de las referencias, salvo las de las decisiones de este plan.

**`units`, deliberadas por la decisión de la unidad** (CQL Reference, Add y Subtract: «the most granular unit of either input»; Java da la menos granular):
- D0, D1, D2, D3, D4, D5, D6, D7, D8, D9, D10, D11, D12, D13, D16, D19, D20, D21, D23, D24, D35 (Java), D47 y D57 (JS);
- sus consecuencias en el paso de `successor` y `predecessor` y en `.value`: D28, D29, D30, D31 y D32.

**`units`, deliberadas por las duraciones de calendario.** La guía de autores, Quantities: «For all but years and months, calendar durations are both equal and equivalent to the corresponding UCUM definite-time unit». Un año son doce meses. Java da null en toda suma de duraciones de calendario de unidades distintas, y el motor suma:
- D34, D36, D37 y D38: días y unidades menores;
- D42, D50 y D57: años y meses (D57 por `Sum`).

**`units`, deliberadas por la grafía:** D36, D37, D39 y D40, frente a cql-execution. Es la misma cantidad, y cql-execution escribe la unidad UCUM (`h`, `minute`, `mo_g`, `a_g`) donde el motor escribe la palabra clave de calendario.

**`units`, deliberadas por la dispersión:** D69, D70, D71 y D72. `StdDev`, `PopulationStdDev` y `Variance` se toman sobre las desviaciones, que son diferencias, en la unidad más granular; Java da la misma cantidad en la menos granular (`0.25 'g'` frente a `250 'mg'`).

**`units`, deliberadas por la temperatura:** D17 y D18. Una temperatura en grados Celsius más una en kelvin da null, como en Java y en fhirpath; cql-execution suma en kelvin.

**`units`, defectos de las referencias:**
- D41, D43, D44, D45, D46, D48, D51, D52, D53, D54, D55, D58, D59, D60, D61, D62, D63, D64, D65, D66, D67 y D68: cql-execution suma un mes o un año y una semana o menos (por la guía de autores, no son comparables); D52 a D55 llegan por la semana, por `-` y por `Sum`, y main también los sumaba;
- D72 (JS): cql-execution da la varianza de gramos en `g`, que es la unidad de la desviación, y Java en `g2`;
- D40 (Java): da null en `1 year + 2 years`, que es una sola unidad («calendar duration keyword, singular or plural», CQL Reference, ToQuantity);
- D26: Java incluye `250 'cm'` en la expansión de `Interval[100 'cm', 2 'm']`, más allá del límite superior; cql-execution responde intervalos, que son la sobrecarga de lista, a la sobrecarga de intervalo;
- D24 (JS): cql-execution da error en `width` de un intervalo con unidades distintas.

**`units`, defectos propios, anteriores y en 1c22:**
- D56: `width of Interval[1 day, 1 month]` da null, y las dos referencias dan error: el intervalo no es válido porque `1 day < 1 month` no se decide, y el motor lo decide (true, en main y en la rama);
- D25, frente a las dos referencias (cql-execution da error). `Size` es `width` más el tamaño de un punto (CQL Reference, Size), y el motor no lo suma. Main daba `1 'm'`.

**`temporal`, deliberadas por la decisión de los offsets parciales:** D12, D13, D14, D15, D16, D20, D24, D25, D27, D36, D40, D42, D47 y D48 (D36 y D42 frente a Java, que ordena dentro de la hora; D40, D47 y D48 frente a cql-execution, que da null fuera de ella).

**`temporal`, defectos de las referencias:**
- D0, D1, D5 y D6: cql-execution acepta el segundo 60;
- D11: cql-execution acepta el minuto 60.

**`temporal`, deliberada:** D4. Las dos rechazan `@T23:59:60`; Java al traducir y el motor al evaluar.

**`temporal`, defectos propios, anteriores y en 1c22:**
- D22: `same hour as` con un offset parcial no sigue la decisión de los offsets;
- D31: `ToDateTime` acepta una hora con offset;
- D45: `convert '23:59:60' to Time` da error, como main, y las dos referencias dan null («If the input string is not formatted correctly, or does not represent a valid time-of-day value, the result is null», CQL Reference, ToTime).

**`crossmodel`, defectos propios, anteriores y en 1c14:**
- D14, D16, D17, D18, D19, D21, D22, D23, D24, D25, D26, D27, D30 y D31: falta la conversión del modelo que inserta el traductor;
- D35, D36, D37, D38 y D39: un primitivo FHIR es igual a un valor de System en `except` y `union`.

## Cierre (25-10-2026)

- Diferencial seed 8008 de la rama contra main, después del review: 12.350 expresiones, y en las nueve áreas 0 nuevas y 0 perdidas.
- Las tres áreas nuevas (`units` 73 definiciones, `temporal` 49, `crossmodel` 40), con las respuestas de Java 5.4.0 y cql-execution, todas clasificadas.
- `sh scripts/references.sh check`: dos corridas idénticas y 0 archivos distintos del repositorio.
- `make check` en verde, con las mutaciones de 001 a 008.
- Reporte: [docs/evidence/2026-10-25.md](../evidence/2026-10-25.md).
- Sin commit ni push: falta la aprobación del usuario.
