# 003: clasificar las divergencias sin clasificar

**Estado:** PR 1 subido (`test/divergence-evidence`); PR 2 (types) en review. **Fecha:** 18-10-2026; actualizado el 19-10-2026.

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

## PR 2: el área `types`

**El oráculo** (`engine_types_oracle_test.go`). El corpus de `types` es una gramática cerrada: `is`, `as`, `as … is null` y `cast` de un elemento de `List<Any>` contra un nombre de tipo, con 20 literales y 21 tipos (`gen`, `typedValues` y `typeNames`). El oráculo computa el esperado desde el tipo en ejecución del literal y la regla, no desde ninguna respuesta:
- **Las reglas,** de CQL 1.5.3 (N1A):
  - Appendix B, Type Operators: `Is` es verdadero si el tipo es «the same as or derived from»; `As` da null si no es del tipo, y `cast` lanza una excepción.
  - Developer's Guide, Types: «All the system-defined types derive directly from System.Any», así que `Code` y `Concept` no son `Vocabulary`, `Long` no es `Integer` e `Integer` no es `Decimal`. `List<E'>` es subtipo de `List<E>`, e `Interval` igual; las choices se comparan como subconjuntos.
- **`undecided`:**
  - el tipo en ejecución del null en `is` y en `cast`, que la especificación no define (10 filas);
  - si el especificador de una lista vacía es su tipo en ejecución.
- **`TestTheTypesOracleReadsTheWholeCorpus`:** el oráculo lee las 900 definiciones.
- **`TestTheTypesVerdictsFollowTheOracle`:** toda fila del área, de cualquier clase, es la que el oráculo computa, en clase, esperado y evidencia, cite lo que cite. Lo reescribe `-classify-types`.
- **`TestTheTypesOracleAgreesWhereAllAgree`:** en las 541 definiciones donde el motor y las dos referencias dan la misma respuesta, el oráculo da esa respuesta o la deja `undecided`. Es lo que lo controla en las relaciones de tipo a las que no llega ninguna divergencia.

**La validación del oráculo** (la primera medición, a mano; desde la ronda 1 la hace `TestTheTypesOracleAgreesWhereAllAgree`, comparando como compara el comparador: coincide en 533 y deja 8 `undecided`, sin diferencias de formato). Sobre las 541 definiciones donde las tres implementaciones coinciden, el oráculo coincide en 520 y deja 6 `undecided`. De las 15 restantes:
- **13 son de formato:** el snapshot escribe `List[Integer 1, Integer 2]` y el comparador `{1, 2}`.
- **2 son reales:** D104 y D823, con `List<Integer>{}` contra `List<String>`. Las tres implementaciones tratan la lista vacía como compatible y el oráculo no. Por eso la lista vacía pasó a `undecided`: le dieron la razón a las implementaciones contra el supuesto del oráculo.

**La clasificación.** Las 472 filas quedaron así: 306 `reference defect`, 142 `own defect`, 10 `undecided` y 14 `unclassified`. Las 14 son filas donde la referencia da el valor que el oráculo espera, pero el arnés no sabe escribirlo (`Unencoded:` de Code, Concept, Tuple y los números de JS): nada dice que la referencia esté mal, y quedan sin veredicto hasta que el arnés las lea. La deuda bajó en 448 filas, a 1037. Los defectos propios son tres defectos del motor, que pasan a pendientes:
1. **`cast` y `as` hacia un tipo genérico no comprueban el tipo** (58 definiciones): `cast` no falla, y `as` no da null. Por ejemplo, D13, D29, D43, D110, D125 y D132.
2. **`cast` convierte en vez de fallar** (11): D37 (`cast true as Integer` da 1), D109, D307, D363, D374 y D441. Aparte, D449 (`cast 'a' as DateTime`) da null en vez de fallar: un `cast` que se porta como `as`. Y D533 (`cast 'a' as Decimal`) falla, pero porque intentó convertir el texto (`invalid decimal: a`): el veredicto es correcto y es el mismo defecto 2, de String a Decimal.
3. **Long sin tipo en ejecución** (10; es D4c): D31, D112, D160, D200, D267, D448 y D811.

**Mutaciones del oráculo,** en `003-evidence.sh`: el subtipado genérico ignorado, `Long` tomado por `Integer`, el null decidido, las clases invertidas y un literal faltante. En la ronda 1 se sumaron cinco: un veredicto que no cita al oráculo, una evidencia que no es la suya, el oráculo equivocado donde no llega ninguna divergencia, culpar a una referencia que el arnés no lee y no leer ninguna definición. Las 32 se detectan.

### Review del PR 2, ronda 1 (Codex y Fable)

Ninguno dejó hallazgos altos. Fable leyó las 472 filas contra el texto de la especificación y no encontró un esperado equivocado; Codex tampoco.

| Hallazgo | Severidad | Resolución |
|---|---|---|
| Las filas que se revisan se eligen por el texto de su evidencia: cambiar la cita esquiva el oráculo (Codex y Fable, con mutación) | Media | **Corregido:** se revisa toda fila no `deliberate`, y `-classify-types` reescribe todas |
| El oráculo no se controla fuera de las divergencias: `Time` derivado de `Date` pasaba (Codex, con mutación) | Media | **Corregido:** `TestTheTypesOracleAgreesWhereAllAgree` lo controla en las 541 definiciones unánimes |
| 14 `reference defect` son respuestas que el arnés no sabe codificar, no errores de la referencia (Fable) | Media | **Corregido:** quedan `unclassified`, en la deuda. Codificar Code, Concept, Tuple y los números de JS es un arreglo del arnés, en su propio PR |
| `TestTheTypesOracleReadsTheWholeCorpus` pasaba sin leer nada (Fable, con mutación) | Baja | **Corregido:** exige las 900 definiciones |
| La guarda no compara la evidencia (Fable) | Baja | **Corregido:** compara clase, esperado y evidencia |
| La validación y D449 estaban mal descritos; el punto de reanudación, desactualizado (Fable) | Baja | **Corregido** |
| `expectedMatches` toma por prefijo todo esperado que termina en «…», en cualquier área (Fable) | Baja | **Sin cambio:** hoy ningún esperado de otra área termina así; se revisa si alguna lo necesita |

### Review del PR 2, ronda 2 (Codex y Fable)

Los arreglos de la ronda 1 cierran sus huecos: los dos ataques de entonces ahora fallan. Fable revisó a mano las 20 filas `Unencoded:`: las 14 `unclassified` son `as` y `cast` de identidad cuya respuesta cruda es el literal mismo, y las 6 `reference defect` esperan error o null y la referencia dio un valor.

| Hallazgo | Severidad | Resolución |
|---|---|---|
| Marcar una fila `deliberate` y citar el plan 003, que nombra la definición, la saca del oráculo (Codex y Fable, con mutación) | Media | **Corregido:** el oráculo revisa toda fila, de cualquier clase. El área no tiene diferencias deliberadas; si alguna vez tiene una, cambia este test |
| La regla `Unencoded` busca una subcadena: un error que dice «Unencoded:» también la dispara (Codex) | Media | **Corregido:** exige un objeto `"type": "Unencoded:…"`. **No se suma** comprobar que el tipo sin codificar sea el esperado: la regla solo puede dejar una fila sin veredicto, en la deuda, nunca darle uno, y se borra cuando el arnés codifique esos valores |
| `TestTheTypesOracleAgreesWhereAllAgree` descartaba una definición si faltaba una de las referencias (Codex) | Media | **Corregido:** basta una referencia que responda y que ninguna diverja. Hoy son las mismas 541 |
| Conteos viejos en la evidencia y el README, la fecha del plan y un motivo del oráculo que afirmaba lo que la especificación no decide (Codex y Fable) | Baja | **Corregido.** La evidencia se regenera con `make evidence` |

Dos mutaciones nuevas: la fila hecha `deliberate` y el error que nombra `Unencoded:`. Son 34, y todas se detectan.

### Review del PR 2, ronda 3 (Codex y Fable): converge

Ninguno dejó hallazgos medios. Los ataques de la ronda 2 ahora fallan. Fable atacó además las columnas de las respuestas fijadas: falsificó la respuesta del motor, la de la referencia como `Unencoded:` y borró una fila, y las guardas lo detectaron las tres veces. Bajos corregidos: dos frases desactualizadas del plan y del README, la deuda contada desde las 1485 iniciales y D533 sumado al inventario del defecto 2.

**Costo:** el oráculo corre en 0,5 s con 11 MB (lo midió Codex), así que no explica el pico de 1,08 GB de `make evidence`. La causa está en otra parte de la evidencia, y queda por medir.

## PR 3: el área `fhir` (diseño, antes de la ronda adversarial)

**Las filas:** 593, de 540 definiciones: 143 `unclassified` (56 de Java, 87 de JS) y 450 `undecided`. Las `undecided` son 390 que el traductor Java rechaza y 60 de los grupos «open» o «disputed» de `fhir.divergences.md`.

**El corpus es una gramática cerrada** (`internal/referenceharness/differential/gen/main.go`, `fhirArea`), como el de `types`:
- **Operandos:** 19 valores, cada uno de uno de estos orígenes:
  - elementos de `fhir-data.json`: `Patient`, `gender`, `birthDate`, `deceased`, `id`, el `period` del Encounter y su `start`, y el `value`, `effective` y `code` de tres Observations;
  - instancias FHIR construidas en CQL: `FHIR.Coding` y `FHIR.code`;
  - valores System: `5 'mg'`, `{ 1, 2 }`, `System.ValueSet` y `null as Integer`.
- **Operaciones:**
  - de tipo: `is`, `as`, `as … is null` y `cast`, contra 26 nombres de tipo;
  - 9 operadores: `-`, `Abs`, `Sum`, `Avg`, `+ 1 day`, `+ 1 'mg'`, `same day as`, `ToString` e `is null`.

### Revisión previa

- **`gofhir/server`:** no tiene oráculos contra referencias CQL. **Se descarta.**
- **La especificación** (CQL 1.5.3, N1A; cada cita se verificó con su URL):
  - **Q1, un `as` o `cast` entre tipos sin relación estática:** la especificación no dice nada.
    - Appendix B declara `as<T>(argument Any) T` y `cast as<T>(argument Any) T`, y todas sus reglas son de ejecución: «If the argument is not of the specified type at run-time the result is null»; con `cast`, «an exception is thrown».
    - Ningún texto convierte un `as` sin relación en un error de compilación. El «Expression of type 'X' cannot be cast as a value of type 'Y'» del traductor Java (364 filas) no tiene base en la especificación.
  - **Q2, una llamada sin sobrecarga que calce, sin conversión que aplique, es un error:** «CQL is a statically typed language … for any given operator invocation, the type of the arguments must match the types of the operands» (03-developersguide, Types). El Author's Guide (Comparison Operators) da como ejemplo de error un `=` entre un número y un texto. La precedencia de conversiones está en 03-developersguide, Conversion Precedence.
  - **Q3:** el tipo de `null` es Any, y `null is T` no está definido. Esas filas quedan `undecided`, igual que en `types`.
  - **Q4, las primitivas FHIR:** el FHIR CQL IG (using-modelinfo) dice que las primitivas FHIR son clases con un elemento `value`, convertibles a las primitivas CQL de forma implícita por FHIRHelpers. El modelinfo oficial, embebido en `model/fhir-modelinfo-4.0.1.xml.gz`, pone `FHIR.date` bajo `FHIR.Element`, y `FHIR.id` y `FHIR.code` bajo `FHIR.string`. `Is` prueba el tipo en ejecución, y nada dice que la conversión implícita se aplique a su operando: `Patient.birthDate is Date` es false, y `Patient.birthDate is FHIR.Element`, true.
- **El modelo de primitivas del motor (Option A, issue #3)** lee una primitiva FHIR como su valor System. Es la causa del grupo «a FHIR primitive is a FHIR.Element». Este PR no lo cambia: lo clasifica, como ya decidió la ronda adversarial (§3, «Defectos propios escondidos en la deuda»).

### El oráculo de las operaciones de tipo

Va en `engine_fhir_oracle_test.go`, con las mismas reglas que `types` (Appendix B y Developer's Guide, Types). Agrega dos fuentes, ninguna de ellas el código del motor:
- **La jerarquía de tipos FHIR:** el oráculo lee `name` y `baseType` de cada `typeInfo` del modelinfo oficial con su propio lector XML. No pasa por `model/`.
- **El tipo en ejecución de cada operando:**
  - el tipo declarado del elemento en el modelinfo (`Patient.gender` es `FHIR.AdministrativeGender`, `Patient.id` es `FHIR.id`);
  - para una choice, la rama que trae el dato: `deceasedDateTime` es `FHIR.dateTime` y `valueQuantity` es `FHIR.Quantity`.
- **Los nombres de tipo sin calificar** (`Quantity`, `Element`, `Patient`, `CodeableConcept`) se resuelven por la regla que la especificación da para los modelos en uso, citada en el oráculo. Java resuelve `Quantity` como `FHIR.Quantity` (D710 y D556). Si la especificación no decide un nombre, sus filas quedan `undecided`.
- **El esperado de un `as` o un `cast` que tiene éxito** es el operando mismo, y se compara por su forma, como un prefijo, igual que las estructuras de `types`. Si el motor devuelve el valor System en vez del elemento FHIR (Option A), es `own defect`.

### Las 390 que Java rechaza: una decisión del usuario

- **Las 364 de «cannot be cast».** La especificación permite rechazarlas, pero no lo exige: es la regla de §3b.4, una fila que la especificación no decide se le presenta al usuario. Las dos lecturas:
  - **(a) Son CQL válida, y el motor las acepta** (`deliberate`). Lo apoyan la firma `Any` de `As` y que ninguna regla las convierte en error. Su esperado es la respuesta en ejecución que computa el oráculo: así, aun dentro de las 364, un motor que responde mal es `own defect`.
  - **(b) Se rechazan como Java** (`own defect`, esperado `error`). El motor tendría que rechazarlas en la fase semántica.
- **Las 26 de «Could not resolve call»** (`ToString` y `SameAs`) se deciden por Q2, fila por fila:
  - con una conversión implícita del modelinfo (`conversionInfo`) o de FHIRHelpers que lleva a una sobrecarga, Java se equivoca: es `reference defect`, con el esperado de la llamada convertida;
  - sin ninguna, el motor tiene que rechazarla: es `own defect`, con esperado `error`.

### El resto, a mano por grupo, con su cita

- **Java con «Could not resolve type …runtime.Any»:** `reference defect`. El defecto ya está en `testdata/reference/runtime.divergences.md`, y el esperado lo da el oráculo.
- **JS que responde null a un `x is T` con `x` no nulo:** `reference defect`, porque Appendix B dice «otherwise, the result is false». Si da «Encountered unexpected error» con un tipo FHIR, también.
- **Los operadores** (`Abs`, `Sum`, `Avg`, `-`, `+`, `same day as`, `ToString`): por grupo, con la conversión de FHIRHelpers que aplica y la cláusula del operador. El inventario de `fhir.divergences.md` es el punto de partida.

### Criterio de salida

- Ninguna fila de `fhir.divergences.tsv` queda `unclassified`.
- Las `undecided` son solo las que la especificación no decide, con sus dos lecturas. Se presentan al usuario por grupo.
- `TestTheFHIRVerdictsFollowTheOracle` sigue a `types`: toda fila de una operación de tipo, de cualquier clase, es lo que el oráculo computa. Hay además un control del oráculo sobre las definiciones donde todos coinciden, y mutaciones del oráculo.

### Lo que queda fuera

- Cambiar el modelo de primitivas (Option A) y arreglar los defectos propios: van a pendientes.
- Rechazar en sema los casts sin relación: solo si el usuario elige (b), y en su propio incremento.

### Ronda adversarial del PR 3 (Codex y Fable)

Los dos midieron en copias del repo. Coinciden en que el diseño, tal como estaba, clasificaría mal.

| Hallazgo | Quién | Resolución |
|---|---|---|
| «Las 390 que Java rechaza» es lo que ve el comparador, no lo que hace Java. `fhir.java.json` tiene 574 errores de traducción, y 184 no figuran en el manifiesto porque el motor también falla y el comparador iguala un error con otro. De esos, 115 son «cannot be cast»: la decisión cubre **479** definiciones, no 364. Esconde defectos propios, como D40: `cast valueString as FHIR.Element` tiene que funcionar, porque `FHIR.string` deriva de `FHIR.Element`, y el motor falla | Los dos, medido | **Aceptado.** Un error de traducción no es la misma respuesta que un error de ejecución. Medido por área: solo pasa en `fhir` (184; en las otras cinco áreas, 0) |
| Las reglas por mensaje («JS da *unexpected error*, entonces es defecto de JS») clasifican mal. D16 y D322 son casts que tienen que fallar, y el motor responde null o texto | Codex | **Aceptado.** Primero se computa el esperado; la clase sale de comparar el esperado con cada respuesta. No queda ninguna regla por mensaje |
| La comparación por prefijo deja pasar corrupciones. Un Coding con `code` cambiado a `CORRUPT` pasa, y `answers.Same` iguala `FHIR.code` con `FHIR.id` cuando los dos dicen `x` | Codex, con sonda | **Aceptado.** El esperado es un tipo calificado más un valor canónico. Hay mutaciones de campo, de valor y de tipo |
| Cada referencia escribe distinto los elementos FHIR (`Patient/p` en Java, `p` en JS), y JS escribe `Unencoded:Object` para los literales construidos | Los dos | **Aceptado.** Se separan representación, tipo y valor. Una forma que el arnés no lee queda `unclassified`, como en el PR 2: no se culpa a la referencia. El criterio de salida se ajusta |
| 5 de las 26 son llamadas ambiguas, no llamadas sin sobrecarga: D126, D205, D328, D346 y D490 | Los dos | **Aceptado.** Function Resolution exige error cuando la ambigüedad empata. La regla de la choice es Choice Types, y donde la especificación no decide el empate, la fila es `undecided` |
| `ToString` no tiene sobrecarga `ToString(String)` en Appendix B: con `FHIR.id` o `FHIR.string` (D383, D419 y D109) ningún camino de conversión llega, y es el motor el que se equivoca | Fable | **Aceptado**, fila por fila: cada fila escribe el camino de conversión probado, o que no hay ninguno. Cuando hacen falta dos conversiones, como en D239 (`FHIR.date → Date → DateTime`), se escribe la regla usada y su cita |
| Los nombres sin calificar los decide la especificación: Developer's Guide, Multiple Data Models («each model used in the current library is searched for an exact match … when the System model declaration is implicit, it is not considered»). `Quantity`, `Element`, `Patient` y `CodeableConcept` son FHIR; `Code`, `Concept` e `Integer` son System, por coincidencia exacta | Fable, con cita; Codex | **Aceptado.** La regla queda en el oráculo, ejecutable. Java acierta en D556 |
| El constructor `FHIR.code { value: 'x' }` pierde el valor (da `Code '' from`), y `FHIR.Coding { … }` igual. Son 2 de los 19 operandos, unas 95 definiciones, y el oráculo culparía a `is`, `as` y `cast` | Fable, con sonda; verificado con D317 y D109 | **Aceptado.** Es un grupo `own defect` propio: «el constructor de una instancia FHIR pierde el valor» |
| El motor pliega `FHIR.Quantity` en `System.Quantity`: `5 'mg' is FHIR.Quantity` da true | Fable, con sonda | **Aceptado:** el oráculo lo clasifica `own defect` |
| La decisión de fondo es la Option A frente a Q4: con el modelinfo, el motor se equivoca en 75 operaciones de tipo. La Option A se decidió como representación (issue #3) | Fable | **Aceptado:** se le presenta al usuario como una decisión, con sus cifras |
| La decisión sobre los 479 estaba presentada con demasiada seguridad. Appendix B, `As`, trae como ejemplo `cast P as Observation`, con P un Patient, y lo da como error **en ejecución** | Codex | **Aceptado:** el ejemplo sostiene la lectura (a). Las dos opciones se presentan con sus consecuencias |
| El control «donde todos coinciden» no puede heredar error≡error: una traducción fallida es «sin respuesta» | Los dos | **Aceptado** |
| El tipo de cada operando sale de las estructuras del modelinfo (espacios de nombres, `Resource.id` heredado, el binding de `AdministrativeGender`, los especificadores de choice anidados), con mutaciones del modelo y de los datos | Codex | **Aceptado** |
| Las cifras se generan, no se escriben a mano. Son 539 definiciones, no 540; D797 no es «con un tipo FHIR» | Los dos | **Aceptado** |

### El diseño del PR 3, ajustado

1. **El comparador distingue los errores por su fase.** El error de traducción de Java no es igual al error en ejecución del motor. Solo afecta a `fhir`: aparecen las 184 divergencias que estaban escondidas, y entran clasificadas en el mismo PR, así que la deuda no crece.
2. **El oráculo de `fhir`** (`engine_fhir_oracle_test.go`):
   - lee las 900 definiciones;
   - computa, para cada operación de tipo, el esperado como tipo calificado más valor canónico, con:
     - la jerarquía y los elementos del modelinfo oficial, leídos por su cuenta;
     - la rama de cada choice, que sale de los datos;
     - la regla Multiple Data Models para los nombres;
   - para los operadores, computa el esperado desde la regla de conversiones (Conversion Precedence, Choice Types, Function Resolution y `conversionInfo`), escribiendo el camino que prueba o que no hay ninguno.
3. **La clase sale del esperado:**
   - `own defect` si el motor no da el esperado;
   - `reference defect` si lo da y la referencia no;
   - `unclassified` si el arnés no lee la respuesta de la referencia;
   - `undecided` donde la especificación no decide (el tipo en ejecución del null, el empate de una choice), con sus dos lecturas;
   - `deliberate` según lo que decida el usuario sobre los casts sin relación y sobre la Option A.
4. **Las guardas:**
   - toda fila de `fhir` es lo que el oráculo computa;
   - el oráculo se controla donde todos coinciden, contando una traducción fallida como «sin respuesta»;
   - mutaciones del modelinfo, de los datos, de los nombres, de los campos y de los tipos.
5. **Criterio de salida:** toda fila tiene la clase que computa el oráculo. Las `unclassified` son solo las que el arnés no lee, y quedan en la deuda con su causa.

### Decisiones del usuario

1. **Los casts sin relación estática** (479 definiciones):
   - **(a) Son CQL válida** (`deliberate`): el oráculo exige la respuesta en ejecución. Lo apoyan la firma `Any` y el ejemplo de Appendix B, que da `cast P as Observation` como error en ejecución. Como consecuencia, una biblioteca que este motor acepta, el traductor Java la rechaza.
   - **(b) Se rechazan como en Java** (`own defect`, con esperado «error de traducción»). Queda pendiente rechazarlas en sema.
2. **La Option A frente a Q4** (75 operaciones de tipo donde el motor responde por el valor System):
   - **`own defect`,** como ya aprobó el plan (§3): quedan como defectos pendientes;
   - **o `deliberate`,** documentando cada definición como consecuencia aceptada de la Option A.

### Aprobación del PR 3

El 19-10-2026 el usuario aprobó el plan ajustado, con estas dos decisiones:
1. **Los casts sin relación estática son CQL válida** y se clasifican `deliberate`. El motor los acepta, y el oráculo exige la respuesta que dan en ejecución: null con `as` y error con `cast`. Un motor que responde otra cosa, como en D40, es `own defect`. Las definiciones que cubre esta decisión las computa el oráculo, se listan en este plan y un test exige que la lista coincida con lo que computa.

   Son los `as` y `cast` que el traductor Java rechaza por sus tipos estáticos (`castable` en `engine_fhir_oracle_test.go`, que coincide con Java en todas las operaciones de tipo del corpus):

<!-- fhir decision 1, written by -classify-fhir: begin -->
D0 D100 D102 D103 D106 D107 D110 D112 D113 D115 D116 D118 D12 D120 D121 D122 D123 D124 D129 D135 D137 D138 D14 D142 D143 D148 D15 D152 D153 D158 D161 D162 D163 D164 D165 D166 D167 D17 D170 D173 D174 D175 D176 D178 D180 D181 D183 D187 D188 D189 D191 D192 D195 D199 D20 D200 D203 D204 D206 D208 D210 D211 D212 D215 D218 D219 D22 D220 D222 D226 D23 D232 D234 D235 D238 D24 D242 D243 D246 D247 D248 D25 D251 D252 D253 D254 D256 D257 D26 D260 D262 D263 D264 D267 D269 D270 D271 D272 D274 D275 D278 D279 D280 D281 D282 D290 D291 D292 D293 D294 D295 D297 D298 D300 D301 D308 D31 D313 D314 D321 D323 D325 D329 D330 D332 D333 D334 D339 D341 D342 D344 D347 D348 D349 D35 D351 D353 D354 D357 D36 D360 D361 D363 D364 D365 D366 D372 D376 D377 D38 D382 D385 D386 D387 D39 D391 D394 D396 D397 D40 D400 D402 D403 D407 D408 D409 D410 D411 D414 D415 D417 D418 D421 D424 D426 D429 D431 D433 D435 D436 D44 D440 D441 D448 D449 D450 D453 D454 D455 D456 D457 D458 D459 D460 D462 D463 D464 D465 D466 D467 D468 D470 D472 D474 D475 D476 D477 D478 D481 D483 D484 D485 D486 D487 D489 D491 D496 D497 D498 D499 D500 D501 D502 D503 D506 D507 D508 D51 D511 D512 D517 D518 D519 D52 D520 D523 D524 D525 D526 D527 D528 D531 D532 D533 D540 D541 D542 D545 D547 D548 D55 D550 D551 D552 D553 D554 D555 D556 D557 D558 D559 D56 D560 D562 D569 D570 D573 D576 D58 D580 D581 D582 D584 D586 D587 D589 D59 D590 D591 D592 D594 D595 D596 D598 D599 D6 D60 D602 D603 D605 D606 D610 D611 D614 D616 D621 D622 D623 D624 D625 D627 D629 D630 D633 D635 D637 D638 D639 D64 D640 D641 D642 D644 D646 D647 D649 D65 D650 D651 D652 D653 D655 D656 D659 D66 D660 D664 D667 D669 D670 D672 D673 D677 D681 D682 D684 D685 D686 D689 D69 D690 D691 D692 D696 D698 D699 D701 D702 D709 D71 D711 D713 D715 D716 D717 D719 D72 D720 D722 D723 D725 D729 D73 D730 D732 D734 D738 D741 D742 D743 D744 D745 D746 D747 D749 D751 D752 D756 D757 D758 D759 D762 D765 D766 D767 D768 D770 D774 D775 D776 D779 D781 D783 D784 D785 D789 D790 D791 D799 D8 D80 D800 D801 D803 D804 D805 D809 D81 D810 D811 D813 D814 D817 D82 D820 D822 D823 D824 D825 D828 D829 D83 D831 D832 D833 D834 D835 D836 D837 D838 D840 D841 D843 D845 D847 D848 D849 D850 D851 D853 D854 D858 D86 D860 D861 D864 D865 D866 D867 D869 D87 D870 D871 D873 D874 D875 D876 D878 D879 D881 D882 D883 D884 D885 D887 D888 D889 D89 D892 D893 D894 D895 D898 D899 D91 D92 D93 D94 D95
<!-- fhir decision 1: end -->
2. **Las operaciones de tipo que la Option A responde mal son `own defect`:** quedan como defectos pendientes, como ya decía §3.

### Implementación del PR 3

**El comparador separa las fases.** `answers.Refused` es el rechazo de una biblioteca, del motor (`ErrSemantic`, `ErrSyntaxError`) o del traductor Java (`"error": "Translation"`), y no es la misma respuesta que un error en ejecución. Solo cambia `fhir`: aparecen 160 divergencias, que ya estaban escondidas. Las otras 24 de las 184 las rechaza el motor también. Lo prueba `TestARefusalIsNotARunTimeError`.

**El oráculo** (`engine_fhir_oracle_test.go`) lee las 900 definiciones con el modelinfo oficial, leído por su cuenta, y con `fhir-data.json`. Compara cada respuesta como texto tipado, con su tipo calificado y su valor. Una primitiva FHIR del motor (Option A) se escribe aparte, sin el tipo de FHIR.

**Las guardas:**
- `TestTheFHIRVerdictsFollowTheOracle`: toda fila, de cualquier clase, es lo que computa el oráculo.
- `TestTheFHIROracleTypesStaticallyAsJava`: `castable` rechaza exactamente lo que rechaza Java en todas las operaciones de tipo. Java acepta el cast de una choice solo a uno de sus tipos o a `Any`.
- `TestTheFHIROracleAgreesWithTheReferencesWhereAllAgree`: donde los tres responden lo mismo, cada referencia da lo que espera el oráculo.
- `TestTheFHIROracleAgreesWhereTheReferencesAgree`: donde las dos referencias responden lo mismo y se pueden leer, el oráculo coincide. La excepción es D797, nombrada.
- `TestTheFHIROracleValuesAreTheReferences`: donde una referencia da un valor del tipo esperado, es el mismo valor. La excepción nombrada es el id que Java escribe `Patient/p`.
- `TestTheFHIRDecisionNamesItsCasts`: este plan nombra las 479 definiciones de la decisión 1.

**Lo que decidió la implementación, con evidencia:**
- **Las choices empatadas son un error de resolución ambigua** (Function Resolution, «an ambiguous resolution error is thrown»), con Conversion Precedence: una conversión a un tipo simple va antes que una a un tipo clase. Con ese orden, `Abs` de un `valueInteger` elige `Abs(Integer)`, y el control de las unánimes lo confirma en los tres motores. Además explica un grupo que el inventario daba por disputado: `Abs` y `Sum` de un `valueQuantity` dan null en las dos referencias, porque `Abs(Integer)` le gana a `Abs(Quantity)` y el cast implícito de la rama Quantity da null. Esto contradice la propuesta de Fable en la ronda adversarial (dejar `undecided` las choices con varias firmas). El texto de la especificación y la medición la descartan.
- **D132** (`5 'mg' + 1 day`) es null: Appendix B, Add, «When adding quantities, the dimensions of each quantity must be the same», con el ejemplo `QuantityAddIsNull`. Java responde `6 'mg'`.
- **D258** (`deceased + 1 'mg'`) es un error en ejecución. Para un Date, Add dice «the quantity unit must be one of: years, months, weeks, or days». La unidad es un valor, no un tipo, así que no hay error de traducción. El motor rechaza la biblioteca.
- **Siguen `undecided`:** el tipo en ejecución de null, dos conversiones seguidas (D239: `FHIR.date → Date → DateTime`) y una lista convertida elemento por elemento (D194).

**El resultado** está en `docs/evidence/2026-10-19.md` (`make evidence`). La deuda de `fhir` baja de 593 a 27 filas, y no crece: las 160 nuevas entran todas clasificadas.

**Mutaciones,** en `003-evidence.sh`:
- la base de un tipo FHIR ignorada;
- un nombre sin calificar buscado primero en System;
- la rama de choice equivocada;
- Conversion Precedence ignorada;
- un literal con un valor corrupto;
- ninguna definición leída;
- un cast menos en la decisión;
- un defecto propio pasado a `deliberate`;
- la representación de una referencia tomada por un valor;
- un rechazo leído como error en ejecución.

Se detectan las 44 del script (46 después de la ronda 1), y las 30 del 002.

**Defectos propios que encontró el PR 3, en `fhir`:**
1. **Option A:** una primitiva FHIR se lee como su valor System, así que `is`, `as` y `cast` responden por el tipo System. En D19, D54 y D794 el comparador no lo ve, porque lee la primitiva por su valor; los nombra `TestTheFHIROracleAgreesWithTheReferencesWhereAllAgree`.
2. **El constructor de una instancia FHIR pierde el valor:** `FHIR.code { value: 'x' }` da `Code '' from`, y lo mismo pasa con `FHIR.Coding`.
3. **`FHIR.Quantity` se pliega en `System.Quantity`** (D579 y D844).
4. **Un `cast` o un `as` de una primitiva a `FHIR.Element` falla en ejecución** (D40, D819 y D88).
5. **Una llamada sin firma posible no se rechaza:** `ToString` de un `FHIR.id` o de un `Period`, `Sum` y `Abs` de algo que no es número, `same day as` de algo que no es fecha. Falla en ejecución o responde.
6. **La aritmética sobre una choice se rechaza en sema,** aunque una sola firma la toma (D111, D299 y D104).
7. **Una llamada ambigua sobre una choice se evalúa** (`ToString`: D126, D328 y D490).

### Review del PR 3, ronda 1 (Codex y Fable)

| # | Hallazgo | Resolución |
|---|---|---|
| C1 | D239 (`FHIR.date → Date → DateTime`) debería ser `reference defect` | **Rechazado.** En Implicit Conversions, Conversion Precedence y Function Resolution no hay ninguna frase sobre encadenar conversiones; lo más cercano es «the least converting overall conversion path» (verificado en cql.hl7.org/N1A). Queda `undecided` y va al usuario. |
| C2 / F1 | La guarda de las unánimes solo registra (`t.Logf`) las respuestas que el comparador esconde (D19, D54 y D794) | **Aceptado.** Las fija `fhirHiddenByTheComparison`, cada una con la respuesta del motor, y cualquier otra falla. Se agregaron dos mutaciones: la excepción cambiada y los Booleans del motor negados. |
| F2 | La evidencia de `docs/evidence/2026-10-19.md` no estaba al día | **Ya resuelto:** `make evidence` corrió después de que el revisor copiara el repo, y la fila `fhir` es la del PR 3. |
| F3 | `classifiedByFHIROracle` no consulta `fhirReferenceRenderings` | **Rechazado.** Hoy no se alcanza: el motor responde mal D216, D316 y D819. Si los arregla, `TestTheFHIRVerdictsFollowTheOracle` falla y obliga a revisarlo; no pasa en silencio. |
| F4 | La cita de D258 dice más que el texto | **Aceptado.** La razón ahora cita «must be one of», dice que la especificación no da ni null ni error, y deja el resto como argumento: la unidad es un valor y las dos referencias fallan en ejecución. |
| F5 | Paréntesis colgante en la evidencia | **Aceptado.** |
| F6 | Código muerto y duplicado | **Aceptado en parte.** Se borró el caso Choice de `derives`, que no se alcanza (verificado con un `panic` de sonda), y la rama duplicada de `+ 1 day`; se usa `bytes.NewReader`. Se rechaza el helper común de `Refused`: llevarlo a `answers` haría que `answers` importe el paquete `cql`, y no hace falta para este PR. |
| F7 | En `compare`, un rechazo de la referencia cae en `different` | **Aceptado.** Cae en `reference-error`. |
| F8 | `fhirReferenceText` lee una respuesta ausente como null | **Rechazado.** Todo el que la llama comprueba antes que la respuesta exista. |
| F9 | El README dice «seis guardas» | **Aceptado:** son siete tests. |

También se corrigió el lint (prealloc y gocritic). Las 46 mutaciones del 003 se detectan, y las del 002 también.

### Review del PR 3, ronda 2 (Codex y Fable): converge

Ninguno de los dos deja hallazgos de severidad media o mayor. Los dos corrieron las 46 mutaciones (todas detectadas) y revisaron las correcciones y los rechazos de la ronda 1. Las fuentes que citan son la especificación (Add de Date y Time, Conversion Precedence) y sondas en sus copias.

| # | Hallazgo (bajo) | Resolución |
|---|---|---|
| R2-C1 | El rechazo de F8 era falso: `TestTheFHIROracleTypesStaticallyAsJava` no comprobaba que hubiera respuesta Java, y una ausente pasaba como null. Codex lo confirmó borrando la respuesta de D19. | **Aceptado.** La guarda exige la respuesta. Se agregó la mutación «the Java engine's answer to an as missing». |
| R2-F1 | Los recuentos de mutaciones quedaron en 44 en el README y en este plan | **Aceptado.** Se corrigieron, y `make evidence` se vuelve a correr al cierre. |
| R2-F2 | F3: si el motor arregla D216, la fila java quedaría `reference defect` y la js `unclassified` | **Se mantiene el rechazo:** `TestTheFHIRVerdictsFollowTheOracle` falla en ese momento, y la fila se revisa entonces. |

Se detectan las 47 mutaciones de `003-evidence.sh`.
