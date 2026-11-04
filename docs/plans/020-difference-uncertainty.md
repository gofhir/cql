# 020: `difference in … between` cuenta fronteras y responde con incertidumbre

Rama `fix/difference-uncertainty`. Viene de la división del 018, decidida por el usuario el 02-11-2026 (ver [018-date-to-datetime.md](018-date-to-datetime.md)). El 018 está detenido por gofhir/fhirpath#256.

## Revisión previa (02-11-2026)

### `gofhir/server` (congelado)

No evalúa CQL: usa `gofhir/cql`. No hay nada que tomar.

### Especificación (CQL 1.5.3)

- **Appendix B, Difference:** «returns the number of boundaries crossed for the specified precision between the first and second arguments. […] any fractional boundaries are dropped».
  - «For calculations involving weeks, Sunday is considered to be the first day of the week for the purposes of determining the number of boundaries crossed.»
  - «When computing the difference between DateTime values with different timezone offsets, the values are normalized to the timezone offset of the evaluation request timestamp, but only when the comparison precision is hours, minutes, seconds, or milliseconds.»
  - Ejemplo normativo: `difference in months between @2012-01-02 and @2012` = `[0, 11]`, una incertidumbre.
- **Language Semantics, Determining Difference and Duration:** la cota baja de una incertidumbre sale del máximo posible del primer operando y del mínimo posible del segundo. La cota alta sale del mínimo del primero y del máximo del segundo.

### Sonda: 755 casos contra Java 5.4.0 y cql-execution 3.3.2

La sonda es `scratchpad/r020/in/probe.cql`, generada por `scratchpad/r020/gen.py`, y su tabla, `scratchpad/r020/table.tsv`. Cubre ocho pares de instantes:
- un ms antes de medianoche;
- 10:59:30 → 11:01:10;
- más de un año;
- un par invertido;
- sábado → domingo;
- offsets distintos;
- dos meses;
- un año menos un día.

Cada operando se recorta a diez combinaciones de precisión, en los ocho `difference in X`, como Date y como DateTime. Se suman Time, null, `difference in X of Interval` y el ejemplo normativo.

| Clase | Casos |
|---|---|
| Las dos referencias y el motor coinciden | 348 |
| Las referencias coinciden y el motor no | 208 |
| Las referencias difieren entre sí | 199 |

Las referencias coinciden y el motor no por **tres defectos propios**:

1. **Por debajo del día, el motor mide el tiempo transcurrido, no cuenta fronteras.** Trunca `th.Sub(tl)`.
   - `difference in hours between @2012-01-31T23:59:59.999 and @2012-02-01T00:00:00.000`: las dos dan 1; el motor, 0.
   - `difference in minutes between @T10:59:30.000 and @T11:01:10.000`: las dos dan 2; el motor, 1.
   - 19 filas, 3 de ellas de Time.
2. **El motor nunca responde con incertidumbre.** 189 filas.
   - El ejemplo normativo `[0, 11]` da 0.
   - `difference in hours between @2012-01-31T and @2012-02-01T00:00:00.000` da 27; las dos dan `[1, 24]`.
   - Con un DateTime de día junto a uno con offset, el motor además mezcla marcos: `difference in hours between @2012-01-01T23:30:00.000-03:00 and @2012-01-02T` da -2, y las dos dan `[1, 24]`.
3. **Las semanas son días / 7, no domingos cruzados.**
   - De sábado 2011-12-31 a martes 2013-01-01 se cruzan 53 domingos.
   - cql-execution da 53; Java y el motor dan 52.
   - Es un defecto de Java (ver abajo).

**La regla de la incertidumbre que siguen las dos referencias** cuando coinciden es otra que la de `duration` (016). `difference` es incierta cuando algún operando es **más grueso** que la precisión pedida. A la misma precisión es exacta:
- `difference in days between @2012-01-31T and @2012-02-01T` = 1;
- `difference in months between @2012-01T and @2012-02T` = 1.

Para esta comprobación, el segundo y el milisegundo son precisiones distintas: `difference in milliseconds between @2012-01-31T23:59:59 and @2012-02-01T00:00:00.000` = `[1, 1000]` en las dos.

**Las referencias difieren entre sí en tres patrones, todos defectos de Java:**
- **Java da null cuando los dos operandos son más gruesos que la precisión.** 144 filas; por ejemplo, `difference in hours between @2012-01-01T and @2012-01-01T`. cql-execution da la incertidumbre (`[-23, 23]`), como pide la regla de Language Semantics para todo operando sin la precisión.
- **Java desborda en milisegundos.** 28 filas: aritmética de 32 bits (1644028928 donde son 31708800000) o un error.
- **Java cuenta las semanas como días / 7.** 27 filas. cql-execution cuenta domingos, como dice la especificación.

### El ELM de Java

El traductor escribe `DifferenceBetween(precision; low, high)`, y `elm/importer.go` ya lo lee. El camino ELM llega a la misma función.

## Alcance

En `funcs.DifferenceBetween`, que pasa a recibir el marco de la evaluación como `DurationBetweenAt`:

1. **Fronteras por debajo del día.**
   - DateTime: cada instante se normaliza al offset de la evaluación (la regla de la especificación para horas o más fino), se trunca a la precisión pedida y se cuentan las unidades entre los dos truncados.
   - Time: se trunca y se cuenta igual, sin offset.
   - La diferencia en segundos trunca los dos al segundo, como las dos referencias, aunque Appendix B diga «single precision with decimal semantics» (ronda adversarial, hallazgo 2).
2. **Semanas por domingos.** Se lleva cada día (nominal, como hoy en día/mes/año) al domingo de su semana, y se cuentan las semanas entre esos domingos.
3. **Incertidumbre.**
   - Cuándo: si algún operando es más grueso que la precisión pedida (segundo y milisegundo distintos).
   - Cómo: los extremos posibles de cada operando salen de `temporalRange` con el marco de la evaluación, como en el 016. La cota baja es la diferencia exacta del máximo del primero al mínimo del segundo, y la alta, del mínimo del primero al máximo del segundo.
   - Si las dos cotas son iguales, la respuesta es el entero.
4. **`difference in X of Interval[…]`** pasa por la misma función y el mismo marco. Las cotas abiertas y null quedan fuera (1c33).

## Fuera del alcance

- **El Date junto a un DateTime** (019) y el DateTime de día al offset de la evaluación (018, gofhir/fhirpath#256). Una fila de la sonda donde el resultado depende de ubicar un DateTime de día con un offset propio, y no de la incertidumbre, se clasifica como dependiente del 018.
- **`duration`** (016, 017) y el backlog 1c26-1c30.

## Divergencias previstas

- **Con Java:**
  - null donde los dos operandos son más gruesos: defecto de la referencia, con la regla de Language Semantics, y cql-execution de acuerdo.
  - Milisegundos desbordados: defecto de la referencia (aritmética de 32 bits).
  - Semanas por días / 7: defecto de la referencia, con la regla del domingo.
- **Con cql-execution:** ninguna prevista. La que aparezca se clasifica con evidencia.

## Criterio de salida

- Las 208 filas en que las referencias coinciden dan lo mismo que ellas.
- En las 199 en que difieren, el motor da la respuesta de cql-execution, y cada divergencia con Java queda clasificada con su evidencia.
- Las 24 pruebas de `difference` de la suite de conformance siguen pasando.
- Los caminos AST y ELM de Java coinciden en el área nueva.
- La diferencial contra main es idéntica en todas las otras áreas.
- eCQM da 23 coincidentes y 1 excusado.
- Los tests nuevos fallan en main, cada oráculo tiene mutaciones y `make check` está en verde.

## Tests y fuentes de datos

- **A mano, citando la especificación:**
  - el ejemplo normativo `[0, 11]`;
  - un ms antes de medianoche en horas, minutos y segundos;
  - 10:59:30 → 11:01:10 en DateTime y en Time;
  - sábado → domingo y los 53 domingos;
  - offsets distintos a horas y a días;
  - la incertidumbre en los dos órdenes;
  - null.
- **Una nueva área de snapshot, `differences`**, generada por un script en el repo (`gen.py` llevado a Go o a `scripts/`), contra Java y cql-execution, con su ELM importado.
- **Un oráculo independiente en Go** (`differenceOracle`), escrito desde la definición y no desde el código:
  - fronteras: truncar y restar;
  - domingos: la semana del día;
  - incertidumbre: los extremos de cada operando.
  - Se contrasta contra el área entera, como `ageOracle` en el 017.
- **Metamórficos:**
  - invertir los operandos da el negativo (en una incertidumbre, las cotas negadas e intercambiadas);
  - el mismo instante escrito en otro offset da lo mismo por debajo del día.

## Mutaciones previstas

- Volver a medir el tiempo transcurrido en horas.
- Semanas como días / 7.
- La incertidumbre nunca, y también a la misma precisión.
- Segundo y milisegundo como una sola precisión en la comprobación de incertidumbre.
- El marco en UTC en lugar del de la evaluación.
- Las cotas intercambiadas.
- El área fuera del manifiesto o de `scripts/references.sh`.
- Una fila deliberada que cita un plan que no la nombra.

## Decisión del usuario (02-11-2026)

Plan aprobado como quedó tras la ronda adversarial. Los defectos compartidos con `duration` van al backlog como 1c31, 1c32 y 1c33.

## Decisión del usuario sobre los offsets que no son de horas enteras (03-11-2026)

En D766 y D769 del área `differences`, el motor y las dos referencias difieren:
- D766: `difference in hours between @2012-01-01T10:10:00.000+05:45 and @2012-01-01T10:00:00.000Z`.
- D769: `difference in hours between @2012-01-01T and @2012-01-01T10:10:00.000+05:45`.

Appendix B pide normalizar al offset de la evaluación (-03:00) y después contar fronteras. En D766 son de 01:25 a 07:00, 6 horas, y eso da el motor. Java y cql-execution dan 5: truncan cada valor a la hora en su propio offset antes de normalizar. En D769, el motor da `[-22, 1]` y las dos referencias, `[-21, 1]`. El usuario eligió la especificación. Las dos filas son divergencias deliberadas, y el punto se suma a la consulta a HL7 que está redactada.

## Ronda adversarial (Opus independiente, 02-11-2026)

Codex sigue sin cuota hasta el 04-11-2026, así que la ronda la hizo un Opus independiente, por decisión temporal del usuario. El revisor escribió un oráculo propio en Python con la regla del plan. Ese oráculo reproduce las 556 filas en que las referencias coinciden y las 199 respuestas de cql-execution donde difieren, así que el disparador, la regla nominal desde el día, Time y los domingos se sostienen en la sonda. Verifiqué los hallazgos medios y altos con una sonda propia (`scratchpad/r020/in/probe2.cql`).

| # | Hallazgo | Severidad | Resolución |
|---|---|---|---|
| 1 | **Resultados fuera del rango de Integer.** En 50 filas, la respuesta de cql-execution sale de -2^31..2^31-1 (D267: 31708800000). Java da la vuelta en 32 bits, y el motor ya devuelve el número grande. La regla del #169 es null donde el resultado no se puede representar | alta | Confirmado, y también en `duration`: `milliseconds between @2011-12-31T12:00:00.000 and @2013-01-01T12:00:00.000` da 31708800000 en el motor y en cql-execution, y 1644028928 en Java. Es un defecto compartido con `duration`, así que va al backlog como frente propio (1c31). En el área, esas filas quedan como defecto propio conocido, con null como respuesta esperada y la cita de 1c31 |
| 2 | **La semántica decimal de segundos y milisegundos.** Appendix B, Difference, dice «seconds and milliseconds are considered a single precision with decimal semantics». Las dos referencias no la aplican: D83 da `[1, 1000]`, y D6 (`seconds` de 23:59:59.999 a 00:00:00.000) da 1. El plan llamaba «semántica decimal» a truncar | media | Aceptado. Se sigue a las dos referencias contra el texto, con esta evidencia, y la frase se agrega a la consulta a HL7 que está redactada. Se corrige el punto 1 del alcance |
| 3 | **`temporalRange` no tiene caso para Time** y devuelve un punto. `duration` tiene el mismo defecto: `minutes between @T10 and @T11:30` da 90, Java `[31, 90]` y cql-execution `[30, 90]`. Agregar Time al `temporalRange` compartido cambia `duration` | media | Confirmado. Para no ampliar el alcance a `duration`, el rango de Time se calcula solo en la incertidumbre de `difference`. `duration` de Time va al backlog (1c32) |
| 4 | **`difference in X of` y `duration in X of` ignoran las cotas abiertas y las null.** `difference in days of Interval(@2012-01-01, @2012-01-05)` da 4, y las dos referencias dan 2. `Interval[@2012-01-01, null]` da null, y las dos dan 2917556 (hasta el máximo DateTime) | media | Confirmado, y compartido con `duration`. Va al backlog (1c33). Se quita el punto 4 del alcance: el camino del intervalo no cambia, salvo que llega a la misma función con el marco |
| 5 | **Huecos de la sonda:** un solo offset de evaluación; ninguno que no sea de horas enteras; ningún operando incierto junto a un offset ajeno; ningún febrero bisiesto como máximo de mes; ninguna semana negativa incierta con el 1 de enero en domingo; `ToDateTime(@2012-01-01)` en horas, que depende del 018/019 | baja | Aceptado: esas filas se agregan al área, y la de ToDateTime queda listada como dependiente del 019. El oráculo en Go corre además a +05:45 y +14:00. Rechazado con motivo: volver a correr las referencias con otro offset, porque el arnés fija el timestamp para todas las áreas y es un cambio del arnés, no del 020 |
| 6 | Las tres clasificaciones como defecto de Java se sostienen. Las mutaciones listadas las mata al menos una fila en que las referencias coinciden | baja | Sin cambio |

## Implementación y validación (03-11-2026)

La implementación la hizo un subagente Sonnet. En `funcs/datetime_advanced.go` agregó `DifferenceBetweenAt` y sus ayudantes (`differenceRange`, `differencePrecision`, `boundariesCrossed`, `dayNumber`, `sundayOf`); se quitó `nominalComponents`, que quedó sin uso. El evaluador pasa `evaluationFrame()` en las dos llamadas.

- **Área `differences`:** 776 definiciones, generadas por `scripts/differences-area.py`. Se corrigió D771, que no parseaba: `(null as Date)`. Las dos corridas de `scripts/references.sh` dieron lo mismo, y solo cambiaron los dos archivos del área.
- **Divergencias:** 403, todas clasificadas por `scratchpad/r020/classify.py`:
  - Java, 176: devuelve un Interval simple donde va una incertidumbre;
  - Java, 122: da null cuando los dos operandos son más gruesos;
  - Java, 51: cuenta las semanas como días / 7;
  - 50 propias (1c31): un Integer fuera de rango, con null como respuesta esperada;
  - 4 deliberadas: D766 y D769, contra cada referencia.
- **Contra la sonda:** el motor coincide con cql-execution en las 755 filas.
- **Tests:** `TestDifferencesByHand` (AST y ELM), `TestDifferencesAtOtherOffsets` (-03:00, +05:45, +14:00), `TestTheDifferencesAreaIsItsOracle` (con `differenceOracle`, independiente de `funcs`) y dos metamórficos. Los dos primeros fallan en main; los metamórficos pasan en main, porque son invariantes que el código viejo también cumplía.
- **Mutaciones:** 15 en `scripts/mutations/020-differences.sh`, todas atrapadas. Subir la incertidumbre a la misma precisión es un mutante equivalente (truncar deja las cotas iguales), así que no se incluyó.
- **Diferencial contra main:** idéntica en las 9 áreas. Las otras 23 áreas de snapshot no cambian.
- **eCQM:** 23 coincidentes y 1 excusado.
- **`make check`:** en verde (553 mutaciones), después de corregir 5 avisos del lint en el test nuevo. Log en `scratchpad/r020/check2.log`.

## Code review

**Ronda 1 (Opus independiente, 03-11-2026): convergió.** Sin hallazgos de severidad media o mayor.
- El revisor verificó que los tests nuevos fallan en main (164 líneas de error) y que `differenceOracle` no usa `funcs`.
- Probó extremos (años 0001 y 9999, 1900, +05:45, semanas en 0001-01), y todos coincidieron con su derivación a mano.
- Sus mutaciones o se atraparon o son equivalentes.

Hallazgos bajos:
1. **19 filas de semanas** citaban la regla de días / 7, pero Java tiene las mismas cotas que el motor; lo único que cambia es el Interval simple. Corregido: la clase del Interval simple se evalúa antes. Ahora son 195 del Interval simple y 32 de semanas, y el script de chequeo del revisor confirma las 403.
2. **sema acepta tipos que la especificación no tiene** (Time con DateTime, Time en días, Date en horas). Ya pasaba en main. Va al backlog como 1c34.
3. **Código muerto:** `DifferenceBetween` se queda como el envoltorio exportado, igual que `DurationBetween`; la rama de cotas iguales se queda como defensiva.
4. **Errata en un comentario.** Corregida.

## Cierre (03-11-2026)

- `make check` final en verde: 553 mutaciones atrapadas, lint sin hallazgos. Log en `scratchpad/r020/check3.log`.
  - Tardó 3 h de reloj, contra 36 min de la corrida anterior, con el mismo trabajo de CPU (unos 3.500 s de usuario). Parece contención de la máquina, no un costo del cambio.
- Queda pendiente sumar a la consulta a HL7 que está redactada dos puntos:
  - la frase «single precision with decimal semantics», que las dos referencias no aplican;
  - D766 y D769, el orden entre normalizar y truncar con un offset que no es de horas enteras.

## Dependencias

Ninguna nueva.
