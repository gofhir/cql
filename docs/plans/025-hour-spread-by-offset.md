# 025: una hora corrida por un offset que no es de horas enteras, y el predicado por extremo

Backlog 1c49 y 1c50 del [README](README.md), más `temporal` D22 (1c22). Rama `fix/hour-spread-by-offset`. Sigue al 024 (#181, c56e141).

## Revisión previa (15-11-2026)

**Especificación.** CQL 1.5.3, Appendix B (https://cql.hl7.org/N1A/09-b-cqlreference.html):

- *Same As*, *Same Or Before*, *Same Or After*, *Before* y *After*: «When comparing DateTime values with different timezone offsets, implementations should normalize to the timezone offset of the evaluation request timestamp, but only when the comparison precision is hours, minutes, seconds, or milliseconds». Y: «if either input has no value for the precision, the comparison stops and the result is null».
- *In*, *Includes* e *Included In*: «If precision is specified and the point type is a Date, DateTime, or Time type, comparisons used in the operation are performed at the specified precision». Son las mismas comparaciones de arriba.

La especificación no dice qué componentes tiene una hora llevada a un offset que la corta por la mitad. `@2020-01-01T00Z` a `+05:30` es de 05:30 a 06:29, y su hora no es ni 5 ni 6.

**Decisión previa del motor.** El 008 (decisión del usuario del 25-10-2026, `compareAcrossPartialOffsets` en `types/temporal_compare.go`) resolvió esto para `<`, `=` y los demás operadores sin precisión: «cada valor es un tramo de su reloj, tan largo como su precisión, y el orden se conoce solo cuando los tramos no se tocan». Los operadores con precisión nunca adoptaron esa regla.

**Cómo lo hace hoy el motor.** Lee una hora corrida por su primer instante, con precisión de hora: `00Z` a `+05:30` es «la hora 5». Con eso:

- `@2020-01-01T00Z before minute of @2020-01-01T01:40+01:00` da true a `+05:30` y `+05:45`. El otro valor es 06:10, dentro del tramo, así que tendría que dar null. Java y cql-execution dan null.
- No es solo el offset de la evaluación. Un offset escrito fraccionario a `-03:00` también corre la hora: `@2020-01-01T10+05:30 same hour as @2020-01-01T02:00-03:00` da false y `… same hour as @2020-01-01T01:00-03:00` da true. El tramo es de 01:30 a 02:29, y las dos tendrían que dar null. Es la misma familia que `temporal` D22.
- La ventana de un DateTime sin offset (`temporalCompareAtPrecision`) pide el mismo `cmp` en los dos extremos. Es el error que hizo fallar la ronda 2 del 024: un `same hour or after` que da 0 en un extremo y 1 en el otro es true en los dos, y el motor da null.
- `in`, `includes` y `during` con precisión (`intervalIncludesAtPrecision`) no resuelven la ventana (M3 del 024, 1c50). Además dan null apenas el punto tiene menos precisión que la pedida, antes de comparar. `@2020-01-01T00Z in minute of Interval[@2020-01-01T00:10+01:00, @2020-01-01T02:50+01:00]` da null en los tres offsets, y Java, cql-execution y la regla de componentes dan true: la hora ya decide.
- `meets` (`successorOf`) lee los componentes una sola vez, así que tampoco ve la hora corrida.

**Medición.** 31 sondas del motor, Java 5.4.0 y cql-execution 3.3.2 a `-03:00`, `+05:30` y `+05:45`, en `scratchpad/r025/table.tsv`. Hay dos oráculos independientes en Python (`oracle.py`, tramos; `oracle2.py`, regla de componentes sobre cada lectura). En 17 de las 31 sondas el motor difiere del oráculo en al menos un offset.

**gofhir/server:** está congelado y usa esta librería; no tiene lector propio. Nada que tomar.

**Otras implementaciones.** Java lleva todo al offset del operando izquierdo. cql-execution hace lo mismo, y además trunca el offset de la evaluación a horas (A3 del 024). Ninguna tiene un tramo corrido, porque nunca leen en un tercer offset. Sirven de comparación, no de oráculo, para estos casos.

## Plan

### Regla

1. **Una hora corrida** es un DateTime a la hora que, leído en el offset de la comparación, no empieza en una hora entera. Pasa cuando la diferencia entre su offset y el de lectura no es de horas enteras. Se compara como su tramo, con la regla del 008: se toman el primer y el último instante, se truncan a la precisión pedida y se comparan las esquinas. Con eso se tiene el conjunto de resultados posibles (−1, 0, 1). Un par sin hora corrida sigue con la regla de componentes, sin cambios. **Decisión D1 del usuario** (abajo).
2. **El predicado se evalúa sobre el conjunto, no sobre un `cmp`.** Un solo lector, `temporalCompareSet`, devuelve el conjunto de resultados posibles de un par, más si alguno es incierto. Junta la ventana del valor sin offset (sus extremos, rellenando entre ellos porque truncar es monótono, más el punto que se lee como está escrito) y los tramos. Cada operador aplica su predicado: si da lo mismo en todo el conjunto, esa es la respuesta; si no, null. Las respuestas de `temporalBefore`, `temporalAfter`, `temporalSameAs`, `temporalSameOrBefore` y `temporalSameOrAfter` salen de ahí, y también las de las frases de intervalos, que las usan.
3. **`in`, `includes` y `during` con precisión** pasan a componerse con esas comparaciones (`same or before`, o `before` en un límite abierto), con `and` de tres valores, como ya hacen las frases. Así tienen la ventana (1c50) y el tramo. **Decisión D2 del usuario:** se va el null anticipado.
4. **`meets` (`successorOf`)** se evalúa en cada lectura de una hora corrida (las dos horas que toca), y se responde solo si las dos coinciden. Con la ventana sigue dando null, como hoy.
5. **`temporalCompareAtPrecision`** se queda para los que necesitan un solo `cmp` (las frases con cantidad, `timingCompare`). Devuelve un `cmp` cierto solo si el conjunto tiene un único valor.

### Fuera de alcance

- El orden de `sort`, `Min` y `Max` (`compareValues`): un orden total no tiene null.
- `types.CompareTemporal` sin precisión (A2 del 024; ya tiene la regla del 008).
- `expand` y `collapse … per`, y `hours between` (1c51).

### Decisiones para el usuario

- **D1, la regla de la hora corrida.**
  - (a) **Tramo, como el 008 (recomendada).** El motor queda con una sola regla para un offset parcial, con precisión o sin ella. `@2020-01-01T00Z after minute of @2020-01-01T05:10+05:30` da true a `+05:30`, como Java: todo el tramo, de 05:30 a 06:29, queda después de 05:10.
  - (b) **Regla de componentes en cada lectura (hora 5 y hora 6).** El mismo caso da null, porque la lectura «hora 5» no tiene minuto.

  Las dos dan null en 1c49 y en D22.
- **D2, el null anticipado de `in`, `includes` y `during`.**
  - (a) **Se va (recomendada).** El lector se reescribe igual para tener la ventana. Java y cql-execution dan la respuesta de las comparaciones.
  - (b) Se queda, y va al backlog.

### Tests y datos (primero)

- **`scripts/fractionaloffset-area.py`**, el área del 024 a `+05:30`:
  - Se agrega una enumeración de horas corridas. Valores a la hora con offsets `Z`, `+01:00`, `+05:30`, `+05:45`, `-03:00` y `-09:30`, contra valores a la hora, al minuto y al segundo, en posiciones antes del tramo, en su borde, dentro, en el otro borde y después. Se recorren los cinco operadores puntuales, `in`, `includes` y `meets`, a la hora y al minuto, cruzando un cambio de día. Son cientos de casos.
  - El oráculo de Python, que no comparte nada con el motor, agrega la regla de D1 y el predicado sobre el conjunto.
  - Se regeneran las referencias del área. Cada divergencia se clasifica con evidencia: Java y cql-execution leen en el offset del operando izquierdo, así que ahí son deliberadas y citan este plan.
- **`temporal` D22:** pasa de defecto propio a respuesta null, y queda deliberada frente a las dos referencias.
- **Tests en Go** (`engine_hour_spread_test.go`):
  - Cada caso de la revisión previa, citando Appendix B.
  - La ventana con predicado: un parámetro sin offset con `same hour or after`, que da 0 en un extremo y 1 en el otro, tiene que dar true. Y el caso de Codex de la ronda 1 del 024, que tiene que seguir en null.
  - M3: `P in hour of Interval[00Z, 01Z]` con P = `00:50` sin offset da null. `TestAnUnplacedValueInAnIntervalIsReadAsBefore` se reemplaza.
  - Las regresiones de la ronda 3 del 024 (M1, M2) a `-03:00`, como guardas: siguen dando true.
  - Todos esos casos a `-03:00`, `+05:30`, `+05:45` y `+14:00`.
- **Mutaciones** (`scripts/mutations/025-hour-spread.sh`): el tramo leído por su primer instante, el predicado vuelto a `cmp` igual, la ventana sin relleno, `in` sin la ventana, `meets` con una sola lectura, el null anticipado de vuelta, una fila del oráculo corrompida y una divergencia que cita un plan que no la nombra. Además, las mutaciones del 001 al 024 completas.

### Criterio de salida

- El motor coincide con el oráculo en todos los casos del área y de `engine_hour_spread_test.go`.
- `temporal` D22 da null.
- La diferencial contra main cambia solo en las filas de `fractionaloffset` y `temporal` que nombra este plan.
- El volcado eCQM es idéntico a `-03:00`, `+14:00` y `+05:30`, o cada diferencia está explicada.
- Las mutaciones se detectan todas y `make check` está en verde.

### Dependencias nuevas

Ninguna.

### Excepciones a la regla 5

Ninguna: no hay nombres FHIR en juego.

## Ronda adversarial 1 (Codex, 15-11-2026)

Verifiqué cada hallazgo con el motor y con Java 5.4.0, a −03:00 y a +05:30. Las sondas están en `scratchpad/r025v`.

| # | Sev. | Hallazgo | Resolución |
|---|---|---|---|
| 1 | alta | Las frases con cantidad (`evalOffsetTiming` a través de `timingCompare`) usan un `cmp` único. `@2020-01-01T10+05:30 2 hours or less before hour of @2020-01-01T03:00-03:00` da true (Java también). Con un `cmp` cierto solo para un conjunto de un elemento, pasaría a null: contra el límite lejano, el conjunto es {0, 1}. | **Aceptado.** La regla 5 cambia: `evalOffsetTiming` aplica cada predicado al conjunto. El caso queda como guarda, con una mutación propia. |
| 2 | alta | `Interval(@…T00:10Z, @…T02:00Z] includes hour of Interval[@…T00:45Z, @…T01:00Z]` da false en todos los offsets. Java da true (verificado a −03:00 y +05:30). Entre intervalos, Includes compara `Start` y `End`, y el comienzo efectivo del abierto, 00:10:00.001, está en la hora 00. | **Aceptado.** Es un defecto previo, pero está en el lector que se reescribe. Entre intervalos se comparan los límites efectivos (`start of` y `end of`, con `same or before`). Contra un punto se mantiene la comparación exclusiva del límite abierto. |
| 3 | alta | `Interval[@…T00:00Z, @…T02:00Z] properly includes hour of @…T00:45Z` da false, y según Codex debería dar true. | **Rechazado para este plan.** Java también da false en los dos offsets, y `properlyContainsPointAtPrecision` sigue a Java por diseño (su comentario). Es otra lectura de *Properly Includes*, sin relación con el tramo. Va al backlog (1c52) con la cita de N1A. |
| 4 | alta | Las esquinas solas no dan el conjunto. `@2020-01-01T00Z same minute as @2020-01-01T01:30+01:00` a +05:30: las esquinas dan {−1, 1}, el conjunto real es {−1, 0, 1}, y el motor da false (Java null). | **Aceptado.** La regla 1 dice ahora que el conjunto de un tramo se rellena entre su mínimo y su máximo, igual que el de la ventana: truncar es monótono y el tramo es continuo. Hay una mutación que quita ese relleno y conserva el de la ventana. |
| 5 | media | Para `meets`, leer el tramo como dos horas enteras lo ensancha. `Interval[@2019-12-31T23Z, @2020-01-01T00Z] meets before minute of Interval[@2020-01-01T06:50+05:30, …]` a +05:30 da false (Java también), y las dos lecturas darían false y null. | **Aceptado.** La regla 4 cambia: `successorOf` usa el tramo exacto, es decir, sus instantes primero y último truncados a la precisión, no dos horas enteras. |
| 6 | alta | Los dos oráculos de borrador esperan true en `@2020-01-01T00Z same day as @2019-12-31T23:00-03:00` (el motor da false, que es lo correcto: más grueso que la hora se lee como está escrito). `oracle.py` aplica el tramo también a pares sin hora corrida. | **Aceptado.** Eran borradores. El oráculo del área aplica el tramo solo a un par con hora corrida, y lee como está escrito por encima de la hora. Se agregan guardas para una precisión más gruesa, para offsets iguales y para la precisión vacía. |

Con los cambios 1, 2, 4 y 5 el plan conserva su forma. Lo que cambia:
- El conjunto se rellena entre su mínimo y su máximo en todas las lecturas.
- Las frases con cantidad y `meets` entran con el tramo.
- Entre intervalos, Includes compara límites efectivos.

Solo el 3 sale del alcance.

## Aprobación del usuario (15-11-2026)

Plan aprobado.
- **D1 (a), tramo como el 008:** una hora corrida se compara por su tramo exacto, y el conjunto de resultados se rellena entre las esquinas.
- **D2 (a):** se quita el null anticipado de `in`, `includes` y `during`.

## Divergencias deliberadas (implementación, 15-11-2026)

El área `fractionaloffset` crece de 1444 a 2834 definiciones (1390 filas nuevas, las últimas del archivo, de D1445 en adelante); el motor coincide con el oráculo en las 1350 que tienen respuesta de oráculo, 496 de ellas null. Las referencias leen un par en el offset del operando izquierdo y una hora cortada por el offset como la hora de su primer instante; el motor lee un par con offsets distintos en el offset de la evaluación (024) y una hora cortada como su tramo (D1 de este plan). Por eso las 364 filas de Java y las 388 de cql-execution que difieren son deliberadas. De ellas, 72 son `meets` al minuto con un límite a la hora, donde la referencia da una respuesta que ninguna de las lecturas del script reproduce, y el motor compara las unidades que el tramo puede ocupar. Con ellas, `temporal` D22 (que pasa de defecto propio a deliberada, con las dos referencias en false y el motor en null; cierra el 1c22).

Filas de `fractionaloffset` que este plan nombra:

D1455, D1458, D1465, D1467, D1468, D1476, D1479, D1482, D1483, D1485, D1488, D1489, D1491, D1495, D1500, D1501, D1524, D1525, D1527, D1533, D1534, D1536, D1537, D1539, D1542, D1545, D1548, D1549, D1551, D1552, D1554, D1555, D1561, D1569, D1583, D1584, D1585, D1587, D1589, D1591, D1593, D1594, D1595, D1597, D1599, D1601, D1602, D1605, D1607, D1608, D1609, D1610, D1611, D1612, D1614, D1615, D1619, D1621, D1625, D1626, D1629, D1631, D1643, D1645, D1647, D1650, D1653, D1655, D1657, D1658, D1661, D1662, D1663, D1665, D1668, D1669, D1671, D1674, D1675, D1679, D1680, D1681, D1685, D1689, D1691, D1695, D1712, D1714, D1715, D1717, D1724, D1726, D1730, D1732, D1735, D1736, D1738, D1742, D1744, D1751, D1772, D1774, D1789, D1790, D1792, D1796, D1798, D1802, D1804, D1805, D1814, D1816, D1823, D1825, D1833, D1835, D1836, D1837, D1838, D1840, D1841, D1843, D1844, D1845, D1847, D1848, D1851, D1854, D1855, D1859, D1860, D1861, D1864, D1865, D1866, D1868, D1869, D1870, D1871, D1875, D1878, D1879, D1881, D1885, D1895, D1897, D1898, D1900, D1902, D1903, D1905, D1907, D1908, D1913, D1914, D1915, D1919, D1920, D1921, D1922, D1924, D1925, D1926, D1928, D1929, D1931, D1932, D1935, D1939, D1940, D1941, D1942, D1945, D1962, D1965, D1967, D1973, D1974, D1979, D1980, D1985, D1986, D1989, D1991, D1995, D2001, D2004, D2009, D2025, D2027, D2028, D2033, D2034, D2037, D2039, D2040, D2045, D2046, D2049, D2051, D2055, D2058, D2061, D2069, D2090, D2092, D2093, D2095, D2102, D2104, D2108, D2110, D2111, D2114, D2116, D2120, D2122, D2129, D2131, D2147, D2150, D2152, D2165, D2168, D2170, D2174, D2176, D2180, D2182, D2185, D2192, D2194, D2205, D2206, D2217, D2218, D2219, D2220, D2221, D2222, D2229, D2230, D2243, D2244, D2247, D2248, D2253, D2254, D2269, D2270, D2279, D2280, D2281, D2282, D2285, D2286, D2293, D2294, D2301, D2302, D2311, D2312, D2313, D2314, D2315, D2316, D2317, D2318, D2325, D2326, D2345, D2346, D2347, D2348, D2371, D2372, D2377, D2378, D2379, D2380, D2397, D2398, D2407, D2408, D2409, D2410, D2411, D2412, D2413, D2414, D2421, D2422, D2429, D2430, D2435, D2436, D2439, D2440, D2441, D2442, D2445, D2446, D2453, D2454, D2471, D2472, D2477, D2478, D2493, D2494, D2503, D2504, D2505, D2506, D2507, D2508, D2517, D2518, D2537, D2538, D2539, D2540, D2563, D2564, D2569, D2570, D2571, D2572, D2587, D2588, D2596, D2599, D2600, D2607, D2608, D2616, D2619, D2620, D2626, D2627, D2628, D2633, D2636, D2637, D2639, D2640, D2641, D2646, D2647, D2648, D2653, D2656, D2657, D2659, D2660, D2661, D2666, D2672, D2679, D2680, D2686, D2692, D2699, D2700, D2706, D2707, D2708, D2713, D2716, D2717, D2719, D2720, D2721, D2726, D2727, D2728, D2733, D2736, D2737, D2739, D2740, D2741, D2746, D2747, D2748, D2753, D2756, D2757, D2759, D2760, D2761, D2766, D2767, D2768, D2773, D2776, D2777, D2779, D2780, D2781, D2786, D2792, D2799, D2800, D2806, D2812, D2819, D2820.

Y la fila D22 de `temporal`.

## Implementación y validación (15-11-2026)

La implementación la hizo un subagente Sonnet. Lo que dejó:
- `eval/compare_set.go` define `comparisons`, el conjunto de resultados con su marca de incierto, y `temporalCompareSet`, que junta la ventana, el tramo (`hourSpread`, `spreadComparisons`) y el relleno.
- Los cinco predicados, `evalOffsetTiming`, `pointInIntervalAtPrecision` e `intervalIncludesAtPrecision` (este con límites efectivos) evalúan sus predicados sobre ese conjunto.
- `successorOf` usa el tramo exacto.
- Se borró `temporalCompareAtPrecision`, que ya no se usaba.

Validación:
- Se agregaron 1390 casos al área, de D1445 en adelante. El motor coincide con el oráculo independiente en los 1350 que tienen respuesta de oráculo. Hay 752 divergencias deliberadas nuevas, nombradas abajo, y `temporal` D22 pasa a deliberada.
- Las 38 sondas de la revisión previa y de la ronda adversarial, que verifiqué a mano, dan la regla del tramo.
- Las 72 filas de `meets minute of` que ningún modelo de lectura reproduce las revisé con muestras: D2587 da null y D2588 false, que es lo que da la regla del tramo.
- La diferencial contra main es idéntica en los nueve corpus. El volcado eCQM es idéntico en sus 3345 líneas.
- Mutaciones: el 025 detecta 23 de 23 y el 024, 32 de 32.

## Code review, ronda 1 (Codex, 15-11-2026)

| # | Sev. | Hallazgo | Resolución |
|---|---|---|---|
| 1 | alta | `Interval(@2020-01-01T00:10Z, @2020-01-01T02:00Z] includes hour of @2020-01-01T00:45Z` y `@2020-01-01T00:45Z during hour of …` dan true en los tres offsets. `in hour of` da false, igual que main. Para promoverlo, el punto pasaba a `[p, p]` y caía en la comparación entre intervalos, por los límites efectivos. | **Confirmado** con su sonda. Las tres grafías tienen que dar una sola respuesta. Busqué las referencias a -03:00: cql-execution da false en las tres, y Java true. Java tampoco es regular: `@…T00:10Z in minute of Interval(@…T00:10Z, …]` da false, y con los límites efectivos daría true. Se mantiene la regla del plan, comparación exclusiva contra un punto, que es la de main y la de cql-execution. Ahora `includes` y `during` contra un punto, con precisión, van a `membershipAtPrecision` antes de que el punto se promueva. Las filas de Java quedan como deliberadas cuando aparecen. |
| 2 | media | Si el límite bajo abierto se vuelve inclusivo solo a la hora, la mutación sobrevive. | **Confirmado.** `TestThePointSpellingsOfMembershipAgree` cubre `in`, `includes`, `during` e `included in` con el comienzo abierto y con el final abierto, a la hora. Tiene tres mutaciones nuevas: la promoción del punto, el comienzo inclusivo solo a la hora y el final inclusivo solo a la hora. |

Después de corregir: `go test ./...` en verde, y las mutaciones del 025 dan 26 de 26.

## Code review, ronda 2 (Codex, 15-11-2026)

| # | Sev. | Hallazgo | Resolución |
|---|---|---|---|
| 1 | media | `properly` decidía la desigualdad con los límites escritos (`cmpAtPrec`), mientras que la inclusión ya compara los efectivos. `Interval[@2020-01-01, @2020-01-03] properly includes day of Interval(@2019-12-31, @2020-01-03]` daba true; main y `properly includes` sin precisión dan false. `Interval[@…T00Z, @…T03Z] properly includes minute of Interval[@…T01Z, @…T02Z]` daba false, y main null; la hora sola decide true. | **Confirmado y corregido.** `properly` es ahora la inclusión más un comienzo estrictamente anterior o un final estrictamente posterior, sobre los mismos límites efectivos y con `or` de tres valores. Se borró `cmpAtPrec`. Lo cubre `TestProperlyIncludesIsStrictOnTheEffectiveBounds`, con siete casos (incluye un comienzo exterior abierto que la lectura de límites escritos no ve) y dos mutaciones. Las mutaciones del 025 dan 28 de 28. |

Codex no encontró ningún hallazgo en la corrección de la ronda 1.

## Code review, ronda 3 (Codex, 15-11-2026): no convergió

Verifiqué los dos hallazgos en la rama y en main, a −03:00, +05:30 y +05:45. Los dos son regresiones del 025.

| # | Sev. | Hallazgo | Verificación |
|---|---|---|---|
| 1 | media | En `properly`, la salida anticipada cuando la inclusión no es true rompe el `and` de tres valores. `Interval[@…T00Z, @…T03:00Z] properly includes hour of Interval[@…T05:30+05:30, @…T03:00Z]` da null a +05:30 y +05:45. Lo correcto es false: la inclusión es null, y las dos comparaciones estrictas son false. | **Confirmado.** La rama da false/null/null y main false en los tres offsets. La corrección es devolver `and(inclusión, or(antes, después))`, y salir antes solo con un false. |
| 2 | media | Un límite null se salta como si la comparación fuera true. `Interval[@2019-12-29T00Z, @…T03:00Z] includes minute of Interval[@…T00:00Z, null]` da true; un null cerrado es el máximo, así que da false. Con `null)`, la respuesta es null. `@2020-01-02 included in hour of Interval(null, @2020-01-03T00:00Z]` da true y tendría que dar null. | **Confirmado.** La rama da true, y main null en los cuatro casos. Antes lo tapaba el null anticipado que quitó D2. La corrección: un null cerrado se resuelve por `startOf`/`endOf`, como en las frases, y un null abierto queda desconocido. Va en los dos lectores, el de intervalos y el de puntos. |

Las dos correcciones son locales, en `intervalIncludesAtPrecision` y `pointInIntervalAtPrecision`. Por la regla de las 3 rondas, me detengo y se lo consulto al usuario.

## Decisión del usuario (15-11-2026): corregir y cerrar, sin cuarta ronda

Corrección de los dos hallazgos de la ronda 3:
- **`properly`** devuelve ahora `and(inclusión, or(antes, después))`, y sale antes solo con un false.
- **Los límites null:** un null cerrado es el mínimo o el máximo del tipo (`startOf`/`endOf`), y uno abierto queda desconocido (`boundsCompare`), en los dos lectores. Verifiqué los siete casos a -03:00 y +05:30 contra Java 5.4.0 y cql-execution 3.3.2, y las dos referencias coinciden con la corrección. `@…T10:00Z in hour of Interval(null, @2020-01-03T00:00Z]` pasa de true (main) a null, como en las dos referencias.

Tests y mutaciones:
- `TestANullBoundAtAPrecision` cubre los ocho casos, con tres mutaciones nuevas. Reanclé nueve mutaciones del 025 a los lectores reescritos.
- Una mutación del 024 (`includes` a precisión con UTC) quedaba sin cubrir: su caso de un punto ahora va por `in`. Se agregó un caso entre intervalos a `TestEveryPhraseWithAPrecisionReadsTheHourInTheEvaluationOffset`.

Resultados:
- `go test ./...` y el lint en verde.
- Las mutaciones dan 31 de 31 en el 025 y 32 de 32 en el 024.
- La diferencial contra main es idéntica en los nueve corpus, y el volcado eCQM también.
