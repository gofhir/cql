# 003: clasificar las divergencias sin clasificar

**Estado:** PR 1 cerrado, sin commit; siguen los PR 2 (types), 3 (fhir) y 4 (arithmetic). **Fecha:** 18-10-2026.

## Problema

`testdata/snapshots/unclassified-debt.tsv` registra 855 divergencias del motor con Java 3.26.0 o con cql-execution 3.3.2 que nadie clasificó: 472 de `types`, 240 de `arithmetic` y 143 de `fhir`. La carta (§10) exige que cada diferencia sea un defecto propio, un defecto de la referencia o una diferencia deliberada, con evidencia de la referencia. El ADR 001 queda Propuesto hasta que estén clasificadas (decisión del usuario, 18-10-2026).

## 1. Revisión previa

- **`gofhir/server`:** no tiene oráculos contra referencias CQL. **Se descarta:** no hay nada que tomar.
- **Lo que ya existe en el repo:**
  - **Los manifiestos `<área>.divergences.tsv`:** `integer`, `scope` y `division` están clasificados por completo. Sus notas siguen un patrón, el defecto de quién y por qué, que **se toma**.
  - **`testdata/snapshots/fhir.divergences.md`** (#141): inventario de las 326 definiciones de `fhir` que Java acepta, con grupos ya descritos. Por ejemplo, «a FHIR primitive is a FHIR.Element», que son 12 definiciones donde las dos referencias coinciden y el motor difiere. **Se toma como punto de partida** de la clasificación de `fhir`.
- **Lo que mide la deuda,** con sondas en el scratchpad que se reescriben como código en el paso 6:
  - **Formas de la diferencia:** 129 grupos por área, referencia y forma de cada respuesta.
  - **Triangulación con la otra referencia:**

    | Área | La otra coincide con el motor | Las dos difieren del motor |
    |---|---|---|
    | `arithmetic` | 202 | 38 (19 definiciones, una fila por referencia) |
    | `types` | 246 | 226 (113 definiciones) |
    | `fhir` | 95 | 48 (24 definiciones) |

  - **Errores de las referencias, por mensaje:**
    - Java: «Could not resolve type …runtime.Any» (126), «Division by zero» (50), «Character N/I is neither a decimal digit…» (16), «Infinite or NaN» (11), «Non-terminating decimal expansion» (5) y «Cannot cast a value of type …» (≈14);
    - cql-execution: «Cannot cast …» (≈70) y «Patient Source does not support …» (25).
  - **Grupos sin error,** que hay que entender antes de clasificar:
    - cql-execution da null donde el motor da false: 83 en `types` y 39 en `fhir`;
    - cql-execution da true contra false (30) y false contra true (22) en `types`;
    - cql-execution da un Decimal con más cifras que la escala 8: 83 en `arithmetic`;
    - Java da un Decimal distinto: 23 en `arithmetic`.
- **Especificación:** CQL 1.5.3, `cql.hl7.org/N1A` (ADR 001). Cada regla del clasificador cita su sección.

## 2. Plan

### Alcance

1. **Un clasificador como código:** `internal/referenceharness/classify`, con un comando que reescribe las notas.
   - Cada **regla** tiene un nombre, la sección de la especificación que la decide y un predicado sobre cuatro cosas: la expresión de la definición, la respuesta de la referencia, la del motor y la de la otra referencia.
   - Una regla asigna una clase: `own defect`, `reference defect: java|js` o `deliberate`, con una nota que la cita.
   - Por ejemplo, la regla «Java falla al resolver `Any`» exige el mensaje de Java y, en la expresión, `Any` como tipo destino.
   - **Una fila que ninguna regla cubre queda sin clasificar.** Una fila que cubren dos reglas con clases distintas es un error.
2. **Evidencia por fila, no por patrón:**
   - el predicado mira la expresión y las respuestas reales de esa fila, no solo su forma;
   - las reglas `deliberate` y `reference defect` exigen que la otra referencia, o la especificación verificada campo a campo, respalde al motor;
   - una regla que no puede dar evidencia de una fila la deja sin clasificar.
3. **Los defectos propios** se clasifican como `own defect: <descripción>` y entran a la lista de pendientes de `docs/plans/README.md`. **No se arreglan en este incremento:** cada uno es un incremento propio.
4. **Las filas que nadie entiende** se investigan una por una, con sondas sobre las dos referencias, hasta que una regla las cubra o se clasifiquen a mano con su evidencia citada.
5. **Al cerrar:**
   - `unclassified-debt.tsv` queda vacío;
   - el ADR 001 pasa a «Aceptado», con el job de CI que ya exige;
   - `fhir.divergences.md` se alinea con las notas.

### Fuera del alcance

- **Arreglar los defectos propios** que salgan: cada uno es un incremento propio.
- **Regenerar los corpus** o las respuestas de las referencias.

### Criterio de salida

- `TestTheUnclassifiedDivergencesDoNotGrow` pasa con la deuda vacía.
- `TestADRsHaveVerification` pasa con el ADR 001 en «Aceptado».
- El comando del clasificador, corrido de nuevo, no cambia ninguna nota.
- Cada regla tiene al menos una fila que la ejercita.

### Tests y mutaciones

- **Tests del clasificador:**
  - cada regla, con una fila real que la cumple y una que no;
  - una fila sin regla queda sin clasificar;
  - dos reglas en conflicto fallan.
- **Mutaciones:**

  | Mutación | Qué tiene que fallar |
  |---|---|
  | Una regla que acepta cualquier fila | su test de «fila que no la cumple» |
  | Una regla que ignora la otra referencia | el test de evidencia |
  | El motor cambia una respuesta de una fila clasificada | `TestAnswersMatchTheReferences` (la divergencia cambia de texto) o el clasificador, que la deja sin clasificar |
  | Una nota editada a mano, distinta de lo que da el clasificador | el test de que el clasificador no cambia ninguna nota |

### Organización

Son 855 filas en tres áreas. Propongo **un solo plan y tres PRs**, uno por área, en este orden: `arithmetic`, `types`, `fhir`. El clasificador y sus reglas comunes entran en el primero. Cada PR pasa por su propio review. El ADR pasa a «Aceptado» en el último.

**Versiones FHIR:** el área `fhir` corre sobre R4 (`fhir-data.json`). R4B y R5 no aplican, porque los corpus y las referencias son de R4.

**Dependencias:** ninguna nueva.

## 3. Ronda adversarial

Hubo dos revisores. Codex corrió 15 expresiones en una copia con escritura, y Fable sondeó en la suya. Los dos coinciden en que el plan, tal como está escrito, **clasificaría mal**. Todos los hallazgos son aceptados, y el plan cambia en la sección 3b.

| Hallazgo | Evidencia, en filas | Resolución |
|---|---|---|
| «La otra referencia coincide con el motor» no es evidencia: las dos comparten sesgos | `types` D132, D30, D244, D456, D763: `as List<String>` devuelve `{1,2}` y `is List<String>` da false, y Java (borrado de tipos) acompaña. `arithmetic` D558: `10.0 ^ 24` da null siendo representable, y cql-execution acompaña. `arithmetic` D114 y D115: Java y el motor dan error, y cql-execution da null, que es lo correcto | **Aceptado:** ninguna clase se asigna porque una referencia coincide. Cada fila necesita un **valor esperado** que salga de la especificación |
| «Las dos difieren» tampoco prueba un defecto propio | D4: `Round(-0.5)` es −1 según el ejemplo de la especificación, y el motor acierta. D80: `mod` exacto | **Aceptado:** lo decide el valor esperado, no la mayoría |
| Defectos propios escondidos en la deuda | Long sin tipo (D112, D160, D200, D267, D448); `as` con tipos parametrizados (D132…); primitivos FHIR como Element (`fhir` D16, D53, D216, D316, D634, D807, D844…); `10.0 ^ 24` (D558) | **Aceptado:** se clasifican `own defect` con su valor esperado, y van a la lista de pendientes |
| Filas que la especificación no decide | `Log(x, 0)` y `Log(x, 1)` (D52, D696); `null is Integer` en Java (D129); la potencia de enteros con exponente negativo (D207, D703, D790, D25) | **Aceptado:** van a una clase `undecided` con las dos lecturas, para que el usuario las decida. Una vez decididas pasan a `deliberate`, con la decisión registrada |
| Un cambio de respuesta deja la nota obsoleta: los manifiestos comparan claves, no texto. Codex cambió la respuesta de D207 y el test pasó | Lo midió Codex con una mutación | **Aceptado, y es un defecto del oráculo del #143:** el manifiesto fija el texto de las dos respuestas, y el test falla si cambia |
| Un archivo de deuda vacío no prueba lo que pide el ADR: 60 filas de `fhir` dicen «open» o «disputed or representation» | Lo contó Codex | **Aceptado:** el criterio de salida cubre toda nota que no sea una de las clases, no solo `unclassified` |
| El orden: `arithmetic` primero es el equivocado, porque `types` y `fhir` comparten las reglas de `is` y `as` y esconden la mayoría de los defectos propios | — | **Aceptado:** primero el esquema y la guarda; después `types` y `fhir`; al final `arithmetic` |

### 3b. El plan, ajustado

1. **PR 1, el esquema de la evidencia y la guarda contra notas obsoletas.**
   - El manifiesto pasa a llevar, por fila, la clase, el valor esperado, la cláusula citada y el texto exacto de las dos respuestas.
   - `TestAnswersMatchTheReferences` falla si cambia el texto de una respuesta registrada, no solo si cambia la clave.
   - Un validador comprueba la coherencia de cada fila:
     - `own defect` exige que el esperado difiera de la respuesta del motor;
     - `reference defect: X` exige que el esperado sea igual a la respuesta del motor y distinto de la de X;
     - `deliberate` exige una decisión registrada, sea un ADR, un plan o una decisión del usuario;
     - `undecided` exige las dos lecturas.
   - Va con sus mutaciones. Todavía no clasifica nada.
2. **PR 2 y 3, `types` y `fhir`.** Cada fila lleva su veredicto, y su valor esperado sale de la especificación:
   - **a mano**, para las filas que nada decide de forma mecánica;
   - **por una regla que computa el esperado** desde la expresión y la cláusula, y lo compara con las tres respuestas. Ejemplo: `x is T` con `x` no nulo, del tipo de su literal, da true o false según la especificación. Una regla que solo mira la forma de la respuesta no vale.
3. **PR 4, `arithmetic`.** Igual que los anteriores, con las reglas de división por cero, NaN y escala.
4. **Las filas `undecided`** se presentan al usuario por grupo. Su decisión pasa a ser `deliberate`.
5. **El ADR 001 pasa a «Aceptado»** cuando ninguna nota de ningún manifiesto sea `unclassified` ni `undecided`.

## 4. Aprobación

El usuario aprobó el plan ajustado (§3b) el 18-10-2026.

## PR 1: el esquema de la evidencia y la guarda contra notas obsoletas

- **El manifiesto** pasa a siete columnas: nombre, referencia, clase, esperado, evidencia, respuesta de la referencia y respuesta del motor. Las dos respuestas se fijan completas, con `wholeLine`; `oneLine` las recortaba a 120 caracteres.
- **`TestAnswersMatchTheReferences`** falla además con `changed:` si una respuesta registrada cambia. Arregla el hueco del #143 que encontró Codex.
- **`-update-divergences`** conserva un veredicto solo si sus dos respuestas no cambiaron; lo prueba `TestAChangedAnswerLosesItsVerdict`.
- **`TestTheDivergencesAreConsistent`** valida cada clase.
- **La migración** se hizo con una sola corrida, sin decidir nada nuevo, y cada nota vieja pasó a su clase:
  - `reference defect`: las de `integer`, `division` y las 3 de `arithmetic` del 001;
  - `deliberate`: D11 de `division`;
  - `undecided`: las 390 «the Java translator refuses it», las 60 «open» o «disputed» de `fhir`, y las 180 de `scope`;
  - `unclassified`: el resto.
  - La cuenta por clase sale de `make evidence`.
- **Mutaciones:** `scripts/mutations/003-evidence.sh`, con 9 casos. `make check` corre los 55 de los tres scripts en verde.

### Review del PR 1, ronda 1 (Codex y Fable)

Fable verificó la migración contra el historial: las 2115 claves coinciden, y ninguna fila recibió una clase más fuerte que la que su nota vieja justificaba.

| Hallazgo | Severidad | Resolución |
|---|---|---|
| Colapsar los espacios esconde cambios (`"a b"` contra `"a  b"`) (Codex) | Media | **Corregido:** `pinned` escapa las barras invertidas, las tabulaciones y los saltos de línea. Se migraron una vez las 325 respuestas fijadas que cambiaban de forma, conservando el veredicto cuando la respuesta, colapsada, era la misma |
| Las respuestas fijadas exactas fallan entre amd64 y arm64 (D379) (Codex) | Media | **Corregido:** `samePin` compara con la equivalencia del área. Un Decimal suelto de un área no exacta usa la tolerancia de `answers.Same`, y todo lo demás compara el texto. Una respuesta fijada que se conserva no se reescribe. Lo prueba `TestAPinIsTheSameAcrossPlatforms` |
| `undecided` no exigía sus dos lecturas y era una salida de la deuda (Codex y Fable) | Media | **Corregido:** exige `reading 1: … (fuente); reading 2: … (fuente)`. La deuda pasa a `pending-verdicts.tsv`, con `unclassified` más `undecided`: 1485 entradas, y el script de CI acepta el archivo nuevo solo por su SHA-256 inicial. Se reescribió la evidencia de las 630 `undecided` con sus dos lecturas |
| `deliberate` aceptaba una cita inexistente o un prefijo como `docs/plans/003` (Codex y Fable) | Media | **Corregido:** exige la ruta completa de un archivo que existe, y que el esperado sea la respuesta del motor |
| `reference defect` u `own defect` con un esperado inventado pasan, porque el validador no conoce la especificación (Fable) | Baja | **Aceptado para el PR 2 y los siguientes:** cada veredicto nuevo cita su cláusula. Las filas migradas citan la evidencia que tenían |
| El punto de reanudación estaba desactualizado (Fable) | Baja | **Corregido** |

**Mutaciones:** 15 en `003-evidence.sh`. `make check` corre las 61 de los tres scripts en verde.

### Review del PR 1, ronda 2 (Codex y Fable)

Fable no dejó hallazgos medios. Verificó ejecutando:
- los arreglos de la ronda 1;
- que sus ataques de la ronda 1 ahora fallan;
- la fidelidad de las 630 evidencias reescritas y de las 325 respuestas fijadas de nuevo.

Codex dejó dos medios.

| Hallazgo | Severidad | Resolución |
|---|---|---|
| Renombrar la deuda, o reescribir el script con otro SHA-256 inicial, esquiva el trinquete (Codex) | Media | **Rechazado con evidencia:** exige editar a la vez `scripts/debt-only-shrinks.sh` y el lector en Go, que es la subversión deliberada que el ADR 001 deja a la revisión del PR (modelo de amenaza). Borrar el archivo sin tocar el código ya falla en `TestThePendingVerdictsDoNotGrow`, lo verificó Fable. Fable lo clasifica igual, como información |
| La regla de las dos lecturas acepta `reading 1: ( ); reading 2: ( )`, porque un paréntesis de una respuesta contaba como fuente, y rechaza una respuesta con `;` (Codex; Fable, baja) | Media | **Corregido:** el formato es `reading 1: … [source: ruta] \| reading 2: … [source: ruta]`, cada texto no vacío y cada ruta existente. Se reescribieron las 630 evidencias. Lo prueban tres mutaciones |
| Una URL que contiene la ruta de un plan pasa como decisión (Codex) | Baja | **Corregido:** la cita es una ruta del repo, no parte de una URL |
| Un plan que existe pero decide otra cosa pasa como decisión (Fable) | Baja | **Corregido:** el archivo citado tiene que nombrar la divergencia |
| `-update-divergences` reescribía las respuestas equivalentes de una fila `unclassified` con los dígitos de la máquina local (Fable) | Baja | **Corregido:** una respuesta fijada equivalente no se reescribe, en ninguna clase |
| El punto de reanudación seguía diciendo 855 (Fable) | Baja | **Corregido:** son 1485 pendientes |

**Mutaciones:** 20 en `003-evidence.sh`.

### Review del PR 1, ronda 3 (Codex y Fable): converge

Ninguno de los dos dejó hallazgos medios. Fable verificó dos cosas:
- **Las 630 evidencias:** cada lectura trae la respuesta fijada y la fuente correcta, y conserva la nota vieja.
- **`-update-divergences`:** dos corridas seguidas no producen ningún cambio.

Bajos corregidos:
- el nombre de la divergencia se busca como palabra entera, así que `D1` no coincide con `D11`;
- la fuente de una lectura tiene que ser un archivo de `testdata/snapshots/` que nombre la divergencia; `go.mod`, `/etc/passwd` o `..` ya no pasan;
- la cita de una decisión no puede venir pegada a una URL;
- el doble chequeo de `writeDivergences` quedó en uno.

Una ruta con espacios se rechaza a propósito, porque los archivos de los corpus no los tienen. Son 22 mutaciones en `003-evidence.sh`.

**Disco:** la verificación final se cayó porque el disco quedó sin espacio. Los scripts de mutaciones copiaban el repo en un `mktemp -d` por corrida y nunca lo borraban. Ahora lo borran al terminar, si lo crearon ellos.

### Cierre del PR 1

- **Chequeos:** `make check` (68 mutaciones), `make evidence` y `make conformance-provenance` en verde. La cuenta de divergencias por clase sale de `make evidence`.
- **Lo que deja:**
  - el esquema de la evidencia;
  - la guarda contra veredictos obsoletos;
  - el validador de clases;
  - la deuda de veredictos pendientes (`pending-verdicts.tsv`: `unclassified` más `undecided`).
- **Todavía no hay ningún veredicto nuevo:** la clasificación es de los PRs 2 a 4.
