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

## PR 4: el área `arithmetic` (diseño, 19-10-2026, pendiente de ronda adversarial)

**Qué hay.**
- **El corpus:** 800 definiciones de una gramática cerrada (`gen`, `arithmetic`). Son literales Integer y Decimal, nulls tipados y divisiones por cero, anidados con `+ - * / ^ div mod`, `Abs`, `Negate`, `Ln`, `Exp`, `Round`, `Truncate`, `Ceiling`, `Floor`, `Log`, `Power` y diez agregados.
- **Las divergencias:** 240 pendientes (125 java y 115 js) y 3 `reference defect` del 001. En 189 de las 243 filas aparece `Ln`, `Exp`, `Log`, `Power`, una raíz o una potencia negativa.

**Los grupos, medidos:**

| Filas | Referencia | Motor | Qué es |
|---|---|---|---|
| 85 | js Decimal con más cifras | Decimal a escala 8 | cql-execution usa flotantes y no recorta |
| 50 | java `Division by zero` | null o un valor | `mod` y `/` por cero en Java |
| 23 | java otro Decimal | Decimal | recortes distintos: dónde se aplica la escala |
| 16 | java `Character N is neither…` | null o un valor | Java convierte un NaN a BigDecimal |
| 11 | java `Infinite or NaN` | null | potencia de base negativa con exponente fraccionario |
| 10 + 6 | null | Decimal | `Product` o `Sum` con un null, entre otros |
| 6 | js null | error del motor | `Ln(0)` y `Exp` que desborda |
| 5 | java `Non-terminating decimal` | Decimal o null | `3 ^ -3` en BigDecimal |
| 11 | js `{…` o `Unencoded` | null o Decimal | el arnés no codifica la respuesta |
| el resto | tipo Integer frente a Decimal y casos sueltos | | |

**La especificación** (Appendix B, Arithmetic Operators, verificado en cql.hl7.org/N1A):
- «operations that cause arithmetic overflow or underflow, or otherwise cannot be performed (such as division by 0) will result in null, rather than a run-time error»;
- en cada operador, «If the result … cannot be represented … the result is null», y para `/`, `div` y `mod` también «or the right argument is 0»;
- el tipo Decimal tiene precisión 28 y escala 8, y «implementations must support at least the 28 digits of precision and 8 digits of scale, but may support more precise values».

**La última frase es la que decide el diseño:** una referencia con más cifras que la escala 8 no está, por eso, fuera de la especificación. El motor recorta a 8 con FLOOR en las operaciones que fija el #134. Esa es una decisión del usuario («igual que Java»), y sus excepciones están en `decidedRuntimeAnswers`: `Ln` sin recortar, la media geométrica exacta, las potencias enteras exactas, `Log` cerca de un entero y el flotante leído por su decimal más corto.

**El oráculo** (`engine_arithmetic_oracle_test.go`) lee las 800 definiciones con su propio parser de la gramática y evalúa cada una con dos modelos:
- **(a) el modelo decidido:** escala 8 con FLOOR en las operaciones del #134, y sus excepciones. `Ln`, `Exp`, `Log` y `Power` fraccionario van por `math` de Go, como decide `DecimalPowerShortestFloat`; el resto, en decimal exacto. Es lo que se espera del motor.
- **(b) el modelo exacto:** la misma expresión con 40 cifras y sin recortar, con las funciones trascendentes a alta precisión. Es lo que da una implementación con más escala, que la especificación permite.

Los dos modelos aplican igual las reglas de null de la especificación: argumento null, divisor 0, resultado no representable (fuera del rango Decimal o del Integer de 32 bits), y `Ln` o `Log` de un no positivo. Los agregados descartan los null, y la lista vacía da null.

**Las clases:**
- **La regla de null decide:** el motor se equivoca → `own defect`; la referencia se equivoca → `reference defect`. Es el caso de la división por cero, el NaN, el desborde y el error en ejecución donde corresponde null.
- **El motor no da (a)** → `own defect`.
- **El motor da (a) y la referencia da (b), a su precisión** (la de un flotante para js, la exacta para Java) → `deliberate`, con la decisión del #134. **Hace falta la decisión del usuario**, y el plan nombraría las definiciones que cubre.
- **El motor da (a) y la referencia no da ni (a) ni (b)** (otro tipo o otro valor) → `reference defect`.
- **La especificación no fija el resultado:** por ejemplo, `Round` de un .5 negativo, si la especificación lo dejara abierto (está por verificar), o un tipo de resultado que no decide → `undecided`, con las dos lecturas.
- **El arnés no codifica la respuesta** → `unclassified`, como en los PRs 2 y 3.

**Las guardas,** como en el PR 3:
- toda fila sigue al oráculo;
- el oráculo lee el corpus entero;
- donde los tres motores coinciden, cada referencia da (a) o (b);
- donde las dos referencias coinciden, el oráculo da (b);
- mutaciones: FLOOR cambiado por redondeo, una regla de null desactivada, la lista de operaciones que recortan, un literal corrupto, el parser y la tolerancia del flotante.

**Lo que no entra:** cambiar el motor. Los defectos propios quedan registrados.

**La mutación de canje del 002** usa `arithmetic` D0 java, y ese PR la clasifica: hay que moverla a una fila que siga pendiente (de `types` o de `fhir`).

**Riesgos que veo:**
1. Los flotantes de (a) difieren en la última cifra entre amd64 y arm64 (memoria de float por plataforma). Para los valores sin recortar, la comparación con (a) tolera 1e-13 relativo, igual que el snapshot.
2. La tolerancia entre (b) y un flotante de js tiene que ser lo bastante estricta como para no tomar por buena una respuesta equivocada.
3. El modelo (a) reproduce una decisión que el motor implementa. El riesgo es copiar el código del motor; el oráculo se escribe desde la lista de decisiones, no desde `funcs`.

### Ronda adversarial del PR 4 (Codex y Fable)

Los dos revisores midieron con sondas en sus copias. Todos los hallazgos se aceptan; el diseño de dos modelos se descarta.

| # | Hallazgo | Evidencia | Resolución |
|---|---|---|---|
| 1 | (a) y (b) no cubren toda política de precisión válida: Java da `Ln` exacto recortado a 8, que no es ni (a) ni (b) | D214, D309, D341, D354, D449, D472, D546, D584 y D53 (ambos), y también D441 y D557 | **Aceptado:** la respuesta válida es un conjunto, no un punto |
| 2 | Java pasa por double en `Power`, `Log` y `Exp`: no es «exacto» | D456 (Fable). Además, `answers.Same` ya tolera 1e-13 relativo, así que la lista de divergencias viene filtrada | **Aceptado** |
| 3 | «Las dos referencias coinciden» no es una guarda segura | D52 y D696: `Log` de base 0 da 0 en las dos referencias y null en el motor | **Aceptado:** la guarda se quita, y la base 0 es una decisión |
| 4 | El rango contradice la cláusula «may support more precise values» | El motor da `Exp(100)` = 2,69e43, pero null para 1e24 (D558). Pasan el rango, en el motor D427, D379, D699, D780 y D4, y en Java D261, D535, D558, D65 y D755 | **Aceptado:** es una decisión |
| 5 | `Round` no está abierto: «less than or equal to -0.5 and greater than -1.0 will round to -1» | Fable lo citó de Appendix B | **Aceptado:** es regla de la especificación |
| 6 | `^(Integer, Integer) Integer` con exponente negativo | La conformance exige `Power(10,-7) = 0.0000001`, un Decimal (runtime.divergences.md:119). En D25 (`1000 ^ -3`), js da un Integer | **Aceptado:** el tipo lo fija la conformance, y el valor es cuestión de precisión |
| 7 | El modelo (a) no está escrito: «las operaciones del #134» es un puntero, y el ADR 001 no menciona Decimal | D615: la resta y `Exp` no se recortan | **Aceptado:** la política de escala va a un ADR (002) con la tabla por operación, y el oráculo se escribe desde esa tabla |
| 8 | La decisión cambia valores más allá de la cifra 8 | D380 da 65500 frente a 65499 exacto; también D450, D617, D455 y D80 | **Aceptado:** el conjunto válido lo decide |
| 9 | `Product` con null: «geometric product of the elements», a diferencia de Sum, que dice «non-null elements» | Fable lo citó; son 12 filas de Java | **Aceptado:** se citan las dos frases, y se va a `undecided` si no alcanzan |
| 10 | Las 11 «no codificadas» son NaN tipados, defectos de la referencia; y D711, un crash de js | D223, D241, D276 y otras | **Aceptado** |
| 11 | El signo de `div` y `mod`: trunca hacia cero, y el resto lleva el signo del dividendo | 11 filas con operandos negativos | **Aceptado**, con controles `-7 div 3 = -2` y `-7 mod 3 = -1` |
| 12 | Flotantes por plataforma: un flotante recortado a 1 ulp de la grilla cambia la cifra 8 | El CI corre solo en amd64 | **Aceptado:** esas filas se marcan ambiguas |
| 13 | La tolerancia relativa general esconde defectos o inventa otros | D456 acepta un error de 10 000; falla con cancelaciones | **Aceptado:** cotas por operación, propagadas |
| 14 | Los recuentos | Son 51 de Java por división por cero (D312 es `0 mod 0`) y 12 filas no codificadas | **Aceptado** |

### El diseño del PR 4, ajustado

**El oráculo calcula, para cada definición, un intervalo con todas las respuestas conformes, y la respuesta del motor por la política decidida.**

1. **Las reglas discretas son exactas,** desde la especificación:
   - el tipo de cada operación (con la conformance para `^` con exponente negativo);
   - los null: argumento null, divisor 0, dominio de `Ln` y `Log`, fuera de rango según la decisión 2;
   - los agregados sin null, y la lista sin elementos no null da null;
   - `Variance` y `StdDev` de un elemento dan null: la muestral divide por n-1, y la inferencia queda escrita;
   - `Round` según Appendix B, y `div`/`mod` truncados.

   Un NaN o un Infinito nunca son un Decimal.
2. **El conjunto conforme.** Para cada operación con valor exacto x (con 50 cifras de precisión), una implementación conforme da x redondeado a una escala k ≥ 8, de cualquier modo; o, si pasa por double, x(1 ± ε). El oráculo propaga ese intervalo por la expresión. En una operación monótona usa los extremos; en `Abs`, `^`, `Max`, `Min` y `Median`, los extremos y los puntos críticos.
   - Si el intervalo cruza una discontinuidad (`Floor`, `Ceiling`, `Truncate`, `Round`, `div`, `mod`, el orden en `Max`, `Min` o `Median`, o la frontera del rango), la definición es **ambigua**: `undecided`, salvo que la referencia caiga fuera de todas las ramas.
3. **La respuesta del motor** sale de la política del ADR 002, una tabla de operaciones y cómo se recorta cada una, escrita desde las decisiones del #134 y no desde `funcs`. Para los valores sin recortar se compara con 1e-13 relativo.

**Las clases:**
- Una regla discreta decide la fila: el motor se equivoca → `own defect`; la referencia se equivoca → `reference defect`.
- El motor no da la política del ADR 002 → `own defect`.
- El motor la da y la referencia cae en el intervalo conforme → `deliberate`: la especificación permite las dos, y el motor eligió su escala (decisión 1). El plan nombra las filas.
- El motor la da y la referencia cae fuera del intervalo → `reference defect`.
- Lo ambiguo, o la especificación sin texto → `undecided`.

**Las guardas:**
- toda fila sigue al oráculo;
- se lee el corpus entero;
- donde coinciden los tres motores, cada respuesta cae en el intervalo, con las excepciones nombradas;
- la respuesta del motor por la política coincide fuera de las divergencias, y las excepciones se fijan;
- la decisión nombra sus filas.

Mutaciones: FLOOR por redondeo, una regla de null apagada, la tabla del ADR alterada, un literal corrupto, una cota ε demasiado ancha y `div` euclídea.

**ADR 002, la escala de Decimal:** recoge las decisiones del #134 (FLOOR a 8, las operaciones que recortan y sus excepciones) en un documento con su tabla y su verificación. NUCLEO lo exige («nada hardcodeado sin ADR»), y el ADR 001 no lo cubre.

### Aprobación del PR 4 y decisiones del usuario (19-10-2026)

El usuario aprobó el diseño ajustado y el ADR 002, y tomó tres decisiones:
1. **La precisión:** si la respuesta de una referencia cae en el intervalo conforme y el motor da su escala (FLOOR a 8), la fila es `deliberate`. El plan nombra las filas que cubre.
2. **El rango:** un valor fuera del rango Decimal, |x| > (10^28−1)/10^8, es null («cannot be represented»). «May support more precise values» habla de cifras, no de magnitud.
3. **`Log` en base 0:** es null. Las dos referencias, que dan 0, son `reference defect` (D52 y D696).

   Las filas que cubre la decisión 1, escritas por `-classify-arithmetic`:

<!-- arithmetic decision 1, written by -classify-arithmetic: begin -->
D10 D103 D107 D108 D117 D120 D122 D127 D133 D137 D142 D144 D147 D166 D169 D190 D202 D203 D208 D214 D23 D233 D24 D249 D25 D256 D260 D265 D267 D282 D29 D298 D309 D310 D316 D332 D336 D340 D341 D354 D361 D363 D385 D397 D405 D431 D441 D448 D449 D455 D457 D459 D461 D463 D465 D468 D471 D472 D480 D484 D491 D516 D524 D53 D545 D546 D557 D56 D570 D584 D590 D600 D611 D615 D617 D622 D634 D646 D651 D653 D667 D674 D676 D677 D68 D680 D69 D697 D705 D707 D713 D723 D729 D735 D737 D747 D752 D762 D778 D8
<!-- arithmetic decision 1: end -->

### Implementación del PR 4

**El oráculo** (`engine_arithmetic_oracle_test.go`) lee las 800 definiciones con su propio parser. Para cada una calcula dos cosas:
- **El intervalo conforme.** El valor exacto con 40 cifras, ensanchado en cada operación por el redondeo a una escala de 8 o más (10^-8, solo si el valor pasa la escala) y por el error de binary64 (4·10^-16 relativo, solo si el valor no es un binary64 exacto). Los literales se ensanchan igual. Un escalón dentro del intervalo (`Floor`, `div` y los demás), un divisor que puede ser 0 o el borde del rango dejan la definición `undecided`.
- **La respuesta por la política del ADR 002** (`docs/adr/002-escala-decimal.md`), que es nuevo.

**La especificación, verificada:**
- **Los agregados ignoran los null:** «In general, nulls encountered during aggregation are ignored» (Author's Guide). Esto zanjó 45 filas que el diseño dejaba `undecided`.
- **`Variance` y `StdDev` dividen por n-1,** según los ejemplos de Appendix B. Una muestra de uno da null.

**Las guardas:**
- `TestTheArithmeticVerdictsFollowTheOracle`.
- `TestTheArithmeticOracleReadsTheWholeCorpus`.
- `TestTheScalePolicyIsConformant`: la política cae en el intervalo en todas las definiciones que la especificación decide.
- `TestTheEngineFollowsTheScalePolicy`: el motor sigue la política fuera de las divergencias, y no hay ninguna excepción.
- `TestTheScalePolicyHoldsItsControls`: 14 controles escritos a mano para las filas de la tabla que el corpus no alcanza (la resta pasada la escala, `Round` de las mitades, `div` y `mod` negativos, el cociente negativo), contra la política y contra el motor.
- `TestTheArithmeticOracleAllowsWhatAllAgreeOn`: tiene 10 excepciones nombradas. En todas, Java y el motor dan error con un infinito, o un valor fuera de rango; el defecto del motor queda en la fila js.
- `TestTheArithmeticDecisionNamesItsRows`.

**Resultado:** las 243 filas tienen veredicto. La deuda de `arithmetic` baja de 240 a 5 `undecided`:
- 4 por un cociente truncado en un número entero (D380, D450 y D80, las que la escala lleva a otro entero);
- 1 por una potencia negativa de una base que puede ser 0 (D520).

La evidencia está en `docs/evidence/2026-10-19.md` (`make evidence`).

**Defectos propios del motor, en `arithmetic`** (12 filas):
- `Ln(0)` y el desborde de `Exp` dan error en ejecución, y deberían dar null: D114, D115, D17, D212, D573 y D7;
- `Exp` fuera del rango Decimal da un valor, y debería dar null (decisión 2): D379, D427, D699 y D780;
- D4, en las dos filas.

**La mutación de canje del 002** pasó a `types` D151 js, que sigue pendiente.

**Mutaciones** (se agregaron 14 al script; se detectan las 60 de `003-evidence.sh` y las 30 del 002):
- FLOOR cambiado por redondeo;
- la resta recortada;
- la división por cero que no da null;
- un literal corrupto;
- un error binary64 mucho más ancho;
- ningún redondeo de escala;
- `div` euclídea;
- `Round` de la mitad hacia arriba;
- los null de un agregado conservados;
- una respuesta de cql-execution corrida en una millonésima;
- una excepción que ya no lo es;
- una fila menos en la decisión;
- el motor fuera de la política donde no llega ninguna divergencia.

### Review del PR 4, ronda 1 (Codex y Fable)

| # | Hallazgo | Resolución |
|---|---|---|
| C1 (alto) | El ensanche admite un signo que ningún redondeo da: Codex cambió D25 java a -0.000000001 y la fila siguió `deliberate` | **Aceptado.** Redondear nunca cruza el 0. Mutación «a Java answer of a sign no rounding gives». |
| C2 (alto) | Una fila ambigua se clasifica antes de mirar las respuestas: los errores de Java en D520 y D80 quedaban `undecided`, y lo mismo D380 con un valor imposible | **Aceptado.** Una fila ambigua es `own defect` si el motor no sigue la política. Es `reference defect` si la referencia da un error o un NaN (la aritmética nunca da error en ejecución), un tipo distinto, o un valor fuera de todas las ramas (`hull`, para un escalón). Mutación «an undecided reference answer outside every branch». D520 java y D80 java pasan a `reference defect`. |
| C3 | `Median` pierde los null y la ambigüedad del par central | **Aceptado.** Se propagan. |
| C4 / F1 | El tope de las potencias pierde la paridad, y `exp` por debajo de -60 da un 0 exacto | **Aceptado.** `cappedPower` conserva el signo y `exp` da 10^-60. Controles `Power((-1000.0), -15)` y `Exp(-70)`, y dos mutaciones. |
| C5 | `Round` con precisión null da null; la especificación dice «If precision is not specified or null, 0 is assumed» | **Aceptado**, con su control y su mutación. |
| C6 | `Avg` de Integers desborda antes de convertir | **Aceptado en el intervalo,** que era el que fallaba; la política ya convertía en `add`. Control `Avg({2000000000, 2000000000})` y mutación. |
| F2 | Los flotantes por plataforma: la tolerancia no se propagaba por `Abs`, `-` y `*`, y un flotante recortado puede moverse un paso | **Aceptado.** La política se evalúa también con cada flotante movido una ulp de cada lado (`floatNudge`), y la tolerancia es lo que eso mueve. Con eso se resuelve el hallazgo adversarial 12. Fuera de los flotantes, la comparación es exacta. |
| F3 | Un nombre de función desconocido se leía como `Round` | **Aceptado.** El parser lo rechaza. |
| F4 | Reglas que el corpus no ejercita: el Integer de 32 bits, `Log` en base 1, la varianza de uno, la potencia par, la potencia exacta de `Log` | **Aceptado.** Hay un control escrito a mano para cada una (`scalePolicyControls`, ahora 24), contra la política, el intervalo y el motor. Las dos inferencias (`Log` en base 1 y la varianza de uno) quedan escritas como inferencias en el comentario del archivo. |
| F5 | `Product` con null: la evidencia tiene que citar la frase de Appendix B que la guía resuelve | **Aceptado**, en el comentario de la regla. |

**Resultado:** la deuda de `arithmetic` baja a 3 filas (D380 js, D450 js y D80 js), y la total a 234. Se detectan las 66 mutaciones de `003-evidence.sh` y las 30 del 002.

### Review del PR 4, ronda 2 (Codex y Fable)

| # | Hallazgo | Resolución |
|---|---|---|
| R2-C1 / R2-F1 (alto) | La tolerancia por ulp acepta respuestas equivocadas. Codex: la raíz exacta de D191 aceptaba 0.50000001. Fable: el rango de D116 aceptaba 2.5 entre 0 y 4.99999999 | **Aceptado.** Solo se mueven los floats de `Exp`, `Log` y `Pow`, cuyo libm no está bien redondeado; la raíz y la aritmética de binary64 sí lo están. La respuesta tiene que ser una de las tres evaluaciones (sin mover, +1 ulp, −1 ulp), no algo entre ellas. Mutaciones de D191 y D116. |
| R2-C2 / R2-F2 | `Median` de la política sumaba los Integers centrales antes de convertirlos | **Aceptado**, con el control `Median({2000000000, 2000000000})`. |
| R2-C3 | Una referencia null en una fila ambigua cuya envoltura no tiene ninguna rama null quedaba `undecided` | **Aceptado.** Una envoltura (`hull`) es de un escalón de un valor conocido, y ahí ninguna rama es null. Mutación con D380 js null. |
| R2-F3 | En una fila ambigua, el tipo salía del intervalo, que lo da antes de decidir el tipo | **Aceptado.** El tipo se toma de la política, cuyas reglas de tipo son las de la especificación. |
| R2-F4 (bajo) | Mover todos los floats para el mismo lado subestima los casos mixtos | **Se mantiene:** el revisor midió que en el peor caso produce un `own defect` espurio en la otra plataforma, nunca una aceptación falsa. |

Se detectan las 69 mutaciones de `003-evidence.sh` y las 30 del 002.

### Review del PR 4, ronda 3 (Codex y Fable): no converge

Se llegó al tope de tres rondas con hallazgos de severidad media o mayor. Por el proceso §10, el trabajo se detiene y se consulta al usuario. Ningún hallazgo cambia hoy un veredicto del corpus: todos son holguras de las guardas, verificadas con mutaciones en las copias.

| # | Hallazgo | Evidencia |
|---|---|---|
| R3-C1 (alto) | `Pow(x, 0.5)` se mueve por ulp aunque Go lo calcula con `Sqrt`, que está bien redondeado | D734 acepta 9.99990001, y `Power(4.0, 0.5)` acepta 1.99999999 |
| R3-F1 (medio) | Las alternativas no se someten al intervalo conforme. Los casos exactos por IEEE (`Exp(0)`, `Pow(1, y)`) se mueven igual | D702 acepta 0.99999999, y D27 acepta 1.0000000000000002 |
| R3-C2 (medio) | La envoltura de una fila ambigua pierde la restricción de cada rama: un `div` admite valores no enteros, y un producto de restos no negativos admite un negativo | D450 acepta -9.5, y D80 acepta -0.05 |
| R3-F2 (medio) | `docs/evidence/2026-10-19.md` está desactualizado | Se regenera al cierre |
| R3-F3 (pregunta) | D380 js y D450 js cumplen la condición de la decisión 1 (una rama conforme, y el motor en su escala) y quedan `undecided` | ¿Se extiende la decisión 1 a las envolturas? |

**La señal de diseño:** los hallazgos de las rondas 2 y 3 vienen todos de dos mecanismos, las alternativas por ulp y las envolturas de las filas ambiguas. Los dos intentan aceptar más respuestas que el modelo exacto, y cada corrección dejó otra holgura.

### Decisión del usuario tras la ronda 3: simplificar (19-10-2026)

El usuario eligió quitar los dos mecanismos que originaban los hallazgos, en vez de corregirlos o cerrar con holguras conocidas.
- **Las alternativas por ulp se van.** El motor se compara exactamente con la política, y la única excepción es un float de `Ln` o `Exp` sin recortar (también a través de `-`, `Abs` y `Negate`), que tolera 1e-13 relativo, como el snapshot. Si el libm de otra plataforma da otro último dígito en un valor recortado, el test falla a la vista, y se decide entonces.
- **Las envolturas se van.** Una fila ambigua solo se decide si el motor no sigue la política (`own defect`), o si la referencia da un error, un NaN o un tipo distinto (`reference defect`). Qué valor da cada rama no se decide.
- **Medido:** los 243 veredictos son idénticos a los de antes de simplificar, y la deuda de `arithmetic` sigue en 3. Se quitaron las dos mutaciones de las envolturas, y se detectan las 67 restantes de `003-evidence.sh` y las 30 del 002.

### Review del PR 4, ronda 4 (Codex y Fable), sobre la simplificación

| # | Hallazgo | Resolución |
|---|---|---|
| R4-C1 (medio) | La tolerancia de 1e-13 de un float sin recortar pasa más allá de lo conforme: D27 (`Exp(0)`) acepta 1.0000000000000002. Fable encontró lo mismo en D508, donde el 1e-13 relativo de un valor grande admite el valor recortado | **Aceptado.** La respuesta de un float sin recortar tiene que estar también en el intervalo conforme (`follows`). Mutación con D27. |
| R4-C2 (medio) | El oráculo leía el snapshot, que tolera 1e-13: un cambio del motor en un valor recortado pasaba sin verse | **Aceptado.** `TestTheEngineFollowsTheScalePolicy` evalúa el motor en vivo (0,4 s, 200 MB). En cada definición, el motor sigue la política exactamente donde la sigue el snapshot. Mutación en `funcs/math_ops.go`: la potencia fraccionaria movida un float. |
| R4-F1 (medio, condicional) | 21 veredictos guardaban los 16–17 dígitos del float de la política, y la comparación de texto exacta los rompería en una plataforma cuyo libm dé otro último dígito | **Aceptado.** El texto de un valor sin recortar guarda 13 cifras significativas («about …»). |
| R4-F3 (cosmético) | El nombre de la mutación de D116 describía las alternativas que se quitaron | **Aceptado.** |
| R3-F2 / R4-C3 (bajo) | `docs/evidence/2026-10-19.md` desactualizado | Lo regenera `make evidence`. |

Los veredictos no cambian (243 idénticos). Se detectan las 69 mutaciones de `003-evidence.sh` y las 30 del 002.

## PR 5: el área `scope` (diseño, 19-10-2026, pendiente de ronda adversarial)

El 003 no planificó este PR. Con los PRs 1 a 4 mergeados (#151–#154), quedan 234 filas pendientes, y 180 son de `scope`.

### Revisión previa

- **Qué son las 180 filas:** todas son de cql-execution (`js`) y están `undecided`, con el mismo texto: «cql-execution resolves a definition's names in the caller's scope; disputed (runtime.divergences.md)». Java no tiene ninguna fila: coincide con este motor en las 600 definiciones.
- **El corpus:** `gen -area scope -seed 1 -n 600`, con el preámbulo `scopePreamble` (`internal/referenceharness/differential/gen/main.go`). Las definiciones `A`, `B`, `C`, `DA`, `DB`, `DAB`, `DList` y `DQuery`, y las funciones `F`, `G`, `H` y `Up`. Las consultas ligan `A`, `B`, `C` y `Q` como alias, `let` y operandos, y así ocultan los nombres de las definiciones. La gramática es chica: literales enteros, `+`, `-`, `*`, las llamadas, `Sum`, `Count` y `Max`, y consultas con `let`, `where` y `return`. El motor responde 500 Integer y 100 null.
- **Qué hace la referencia:** el traductor de referencia ya resolvió cada nombre al traducir. En `DQuery` (`({ 2, 3 }) A return A + B`), el ELM tiene `Add(AliasRef A, ExpressionRef B)`, medido traduciendo `scope.cql` con `Translate.java` el 19-10-2026. cql-execution ejecuta **ese mismo ELM** (`evaluate.js`), y aun así busca el nombre en la cadena de contextos de quien llama. Un ejemplo es `D4` (`Sum((({ 3, 4 }) C where G(2) > 5 return Max(DList)))`): Java y este motor dan 100, y cql-execution da 10, porque toma el alias `C` para el `C` de `DList`. *(Corregido en la ronda adversarial: el ejemplo era `D5`, que da 25 en los tres motores.)*
- **La especificación** (CQL 1.5.3, Logical Specification, Reusing Logic y Queries) dice:
  - ExpressionRef: «The result of evaluating an ExpressionReference is the result of evaluating the referenced NamedExpression».
  - AliasRef: «allows for the reference of a specific source within the scope of a query».
  - QueryLetRef: «a specific let definition within the scope of a query».
  - OperandRef: «the value of an operand to be referenced as part of an expression within the body of a function definition».

  De ahí se infiere (es una inferencia, no una frase de la especificación) que el resultado de una referencia no depende de quién la llama. El ámbito de un alias es su consulta, y el de un operando, el cuerpo de su función.
- **`gofhir/server`:** no aplica. Usa esta librería y no evalúa CQL por su cuenta.
- **El registro anterior** (`testdata/reference/runtime.divergences.md`, «Disputed: the scope of a definition») dejó la diferencia como disputa, sin cita. Las cuatro filas de `disputedRuntimeAnswers` son del corpus `runtime`, y quedan fuera de este PR.

### El oráculo

`engine_scope_oracle_test.go` tiene un parser de la gramática del corpus, que rechaza lo que no conoce, y un evaluador con **ámbito léxico**:
- Una definición se evalúa en el ámbito vacío de la biblioteca.
- Una función se evalúa solo con sus operandos.
- Un alias y un `let` se ven solo dentro de su consulta.

Reglas que toma de la especificación, cada una con su cita en un comentario:
- **`return` es distinct** si no dice `all` (Queries, Return clause).
- **`where`** se queda solo con los elementos cuya condición es true: null descarta.
- **`Sum` y `Max`** ignoran los null y dan null sobre una lista vacía. **`Count`** cuenta los no null.
- **La aritmética con null** da null.
- **El Integer es de 32 bits** (ADR 002): un desborde da null.

### Veredictos

- El motor distinto del oráculo es `own defect`.
- La referencia distinta del oráculo es `reference defect`, y la evidencia cita la regla de ExpressionRef y el ELM.
- Una referencia que no se puede leer queda `unclassified`.

No se espera ninguna fila `deliberate` ni `undecided`.

### Guardas y mutaciones

- `TestTheScopeOracleReadsTheWholeCorpus`: el parser lee las 600 definiciones.
- `TestTheScopeOracleAgreesWithJava`: el oráculo da la respuesta de Java en las 600. Es una segunda opinión independiente del motor.
- `TestTheEngineFollowsTheScopeOracle`: el motor, evaluado en vivo, da la respuesta del oráculo en las 600.
- `TestTheScopeVerdictsFollowTheOracle`: cada fila tiene el veredicto que computa el oráculo, regenerable con `-classify-scope`.
- Mutaciones en `003-evidence.sh`:
  - un oráculo con ámbito dinámico, es decir, el defecto de cql-execution;
  - `return` sin distinct;
  - un `where` que deja pasar el null;
  - un `Sum` vacío que da 0;
  - una respuesta de Java corrupta;
  - el motor con una definición evaluada en el ámbito de quien llama;
  - un veredicto cambiado.

### Criterio de salida

Las 180 filas de `scope` tienen veredicto y `pending-verdicts.tsv` las suelta: la deuda baja de 234 a 54. Pasan `make check` y `make evidence`, y se detectan todas las mutaciones.

### Lo que queda fuera

- Las cuatro filas de `disputedRuntimeAnswers` del corpus `runtime`, que caben en la misma regla y van en un PR propio.
- Las 54 filas pendientes de otras áreas: 24 de `types`, 27 de `fhir` y 3 de `arithmetic`. Sin ellas, el ADR 001 no puede pasar a «Aceptado».
- Reportarle el defecto a cql-execution: se redacta el issue, y se publica solo si el usuario lo aprueba (principio 10).

### Ronda adversarial del PR 5 (Codex y Fable)

Los dos confirman la lectura de ámbito léxico de la especificación. Fable, con un evaluador independiente en Python, midió que el modelo léxico, Java y este motor coinciden en las 600 definiciones.

| # | Hallazgo | Resolución |
|---|---|---|
| A1 (alto, los dos) | El ejemplo `D5` es falso: da 25 en los tres motores y no tiene fila de divergencia | **Aceptado.** El ejemplo pasa a ser `D4`, medido (100 contra 10). |
| A2 (alto, Fable) | El mecanismo no es solo «el ámbito de quien llama». cql-execution busca el nombre en la cadena de contextos (`ctx.get`) **y memoiza** el primer valor de la definición en ese contexto. Sin la memoización, el modelo explica 65 de las 180 filas. Depende además del arnés: `evaluate.js` corre cada definición en un contexto nuevo | **Aceptado, y verificado con su sonda:** con memoización, 0 de 180 filas no coinciden; sin ella, 115. La evidencia de los veredictos y el issue de cql-execution describen los dos efectos. |
| A3 (alto, los dos) | Hay reglas que el corpus no ejercita, y sus mutaciones no se pueden detectar: el desborde nunca ocurre, `Count` y `Max` solo reciben `DList` y `DQuery` (sin null ni lista vacía), `Sum` con un null nunca cambia una fila, y una función que ve el ámbito de quien llama no cambia ninguna | **Aceptado como lo propone Codex**, igual que en el PR 4: hay controles escritos a mano, con su cita de la especificación, para el null, las listas mixtas y vacías, una proyección null, un `Max` de negativos y un desborde intermedio. Se comparan contra el oráculo y contra el motor en vivo, y cada regla tiene su mutación. La mutación de ámbito dinámico es de **definiciones** (210 filas); la de funciones se prueba con un control. |
| A4 (medio, Codex) | Faltan casos de ligado: un `let` que oculta un alias exterior, un alias exterior correlacionado, la fuente evaluada antes de ligar su alias, el inicializador de un `let` evaluado antes de ligarlo (`D16`, `let B: B`), y el receptor fluido evaluado en el ámbito de quien llama | **Aceptado.** Cada uno es un control escrito a mano, porque el generador excluye un `let` que oculte un nombre ya ligado. |
| A5 (medio, los dos) | Independencia: el oráculo tiene que leer de `scope.cql` el preámbulo (ocho definiciones y cuatro cuerpos), rechazar la sintaxis desconocida y el texto sobrante, y no llamar al parser ni al evaluador del motor | **Aceptado.** El oráculo lee el preámbulo del archivo, y hay una mutación del preámbulo. `TestTheScopeOracleAgreesWithJava` es la segunda opinión; los valores esperados salen del oráculo. |
| A6 (medio y bajo) | Las cuatro filas de `runtime` ya tienen veredicto (`scopeSettled`); les falta la cita | **Aceptado en este PR**, porque es solo el texto: `scopeSettled` y `runtime.divergences.md` citan ExpressionRef y dejan de decir «Disputed». Los pines quedan como están. |
| A7 (bajo, Fable) | No citar ExpressionDef: su contexto es Patient o Unfiltered, no el ámbito léxico | **Aceptado.** |

### Aprobación del PR 5 (19-10-2026)

El usuario aprobó el diseño ajustado, y pidió dejar redactado el issue para cql-execution sin publicarlo.

### Implementación del PR 5

- **`engine_scope_oracle_test.go`:**
  - Parser de la gramática del corpus, que rechaza la sintaxis desconocida y el texto sobrante. Lee de `scope.cql` el preámbulo, sus 8 definiciones y sus 4 funciones.
  - Evaluador con ámbito léxico, que no comparte código con el motor.
  - 21 controles escritos a mano, cada uno con su regla.
- **Seis tests:**
  - `TestTheScopeOracleReadsTheWholeCorpus`;
  - `TestTheScopeOracleAgreesWithJava`, que da las 600 iguales;
  - `TestTheEngineFollowsTheScopeOracle`, en vivo y contra el snapshot;
  - `TestTheScopeOracleHoldsItsControls`, contra el oráculo y el motor en vivo;
  - `TestTheScopeVerdictsFollowTheOracle`, regenerable con `-classify-scope`;
  - el test de los rechazos, dentro de `ReadsTheWholeCorpus`.
- **Veredictos:** las 180 filas son `reference defect`. La deuda total baja de 234 a **54**: 24 de `types`, 27 de `fhir` y 3 de `arithmetic`.
- **`runtime`:** `scopeSettled` y `runtime.divergences.md` («Settled: the scope of a definition») citan ExpressionRef.
- **Mutaciones:** 12 nuevas en `003-evidence.sh`. Ocho son del oráculo: ámbito dinámico de definiciones, `return` sin distinct, `where` que deja pasar el null, agregado vacío igual a 0, `Count` que cuenta los null, Integer sin límite de 32 bits, alias ligado antes que su fuente, y `let` ligado antes que su valor. Las otras cuatro son el preámbulo mal leído, una respuesta de Java corrupta, el motor con una definición evaluada en el ámbito de quien llama (`eval/evaluator.go`) y un veredicto cambiado. Seis mutaciones antiguas usaban como ejemplo de `undecided` la fila D1 js de `scope`, y pasaron a D380 js de `arithmetic`. Se detectaban entonces las 81 de `003-evidence.sh` y las 30 del 002; después de la ronda 1 son 84.
- **Lo que no se puede verificar:** una función que viera el ámbito de quien llama no cambia ninguna respuesta. Los cuerpos del preámbulo solo nombran sus operandos y definiciones, así que esa regla queda sin mutación detectable.

### Issue para cql-execution (redactado, sin publicar)

No hay duplicado abierto. El #46 («Scoping Issue in CoffeeScript CQL Execution», cerrado en 2021) trataba de un alias dentro de una función, y se atribuyó al traductor.

> **A definition's value depends on the scope it is first referenced from**
>
> cql-execution 3.3.2, with the ELM of the reference translator (cql-to-elm, the same version as engine 3.26.0):
>
> ```cql
> library Repro version '1.0'
> define C: 100
> define DList: { 1, C }
> define Test: ({ 3 }) C return Max(DList)
> ```
>
> ```js
> const lib = new cql.Library(elm);
> (await new cql.Executor(lib).exec(new cql.PatientSource([]))).unfilteredResults.Test; // [100]
> await lib.expressions.Test.exec(new cql.UnfilteredContext(lib, new cql.Results(), undefined, {})); // [3]
> ```
>
> The Java engine gives `{ 100 }` either way. The translator resolves the `C` in `DList` as `ExpressionRef C`. When `DList` is first evaluated from inside the query, cql-execution looks `C` up through the context it is referenced from, where the alias `C` stands in for it, and keeps the value `DList` took there for the rest of the evaluation. `Executor.exec` hides this whenever the definition was evaluated earlier at the top level, in an empty scope, so the result depends on the order of evaluation.
>
> CQL 1.5.3, Logical Specification, ExpressionRef: "The result of evaluating an ExpressionReference is the result of evaluating the referenced NamedExpression". An alias is referenced "within the scope of a query" (AliasRef). In a differential of 600 generated definitions, each evaluated on its own, 180 differ from the Java engine. Looking the name up in the caller's context alone explains 65 of them; with the kept value, it explains all 180.
>
> The fix is probably to evaluate the expression of an `ExpressionRef` in a context holding only the library's definitions and parameters, not the aliases, lets and operands of the context it is referenced from.

Medido el 19-10-2026, con `Translate.java`, `Evaluate.java` y un script de `cql-execution` 3.3.2 en el arnés: Java da `{ 100 }`. cql-execution da `[100]` con `Executor.exec` y `[3]` con `Test` evaluado solo. Este último detalle lo aportó la ronda 1 del review (Codex).

### Review del PR 5, ronda 1 (Codex y Fable)

Fable no deja hallazgos medios ni altos. Codex deja tres medios.

| # | Hallazgo | Resolución |
|---|---|---|
| C1 (medio) | Ningún control tiene un `let` que oculte un alias exterior (A4): un oráculo que dejara el alias en su lugar pasaba todo | **Aceptado.** Control `Sum(({ 9 }) B return Sum(({ 1 }) Q let B: Q return B))` = 1, y su mutación. |
| C2 / F2 (medio) | El distinct de los null no se verifica: los agregados ignoran los null | **Aceptado.** Los controles pueden ser listas, escritas como las escribe el motor: `({ 1, 2 }) Q return null` es `List {null}`, y `({ 3, 4, 3 }) Q return Q` es `List {3, 4}`. Hay mutación. |
| C3 (medio) | El issue de cql-execution no se reproduce con `Executor.exec`, porque las definiciones de arriba llenan su caché | **Aceptado y medido:** `Executor.exec` da `[100]`, y `Test` evaluado solo da `[3]`. El issue trae las dos invocaciones y la dependencia del orden de evaluación. La evidencia de las 180 filas dice que el arnés evalúa cada definición en un contexto propio. |
| F1 (bajo) | La precedencia no se ejercita: el generador pone paréntesis en todo | **Aceptado:** control `(1 + 2 * 3)` = 7. |
| F3 (bajo) | `Count(null)` = 0 no tiene control | **Aceptado:** controles `Count(null)`, `Sum(null)` y `Max(null)`, y una mutación. |
| F4 (bajo) | El oráculo aceptaba lo que CQL rechaza: un literal fuera del rango de Integer y un `where` que no es Boolean | **Aceptado:** los dos dan error, y están en la lista de rechazos. |
| F5 (nit) | `regexp.MustCompile` dentro de los bucles | **Aceptado.** |

Se detectan las 84 mutaciones de `003-evidence.sh`.

### Review del PR 5, ronda 2 (Codex y Fable): converge

Ninguno de los dos deja hallazgos medios ni altos. Codex volvió a traducir y ejecutar el corpus con cql-execution: las 600 respuestas son las del snapshot, con las mismas 180 divergencias, y la reproducción del issue (`[100]` con `Executor.exec` y `[3]` evaluado solo) se repite. Fable dejó dos bajos, aceptados: un agregado rechaza un elemento que no es Integer ni null (`Count({ 1 > 0 })` y `Sum({ { 1 } })` se suman a la lista de rechazos), y la cifra de mutaciones del registro de la implementación quedó corregida. Se detectan las 84 mutaciones.

## PR 6: lo que la especificación no decide (diseño, 20-10-2026)

El 003 no planificó este PR. Con el PR 5 (#155), quedan 54 filas pendientes: 38 `undecided` y 16 `unclassified`.

### Decisiones del usuario (20-10-2026)

Se le presentaron las 38 filas `undecided` en cuatro grupos, con las respuestas de cada motor:

1. **`null is T` es false** (16 filas de Java: 9 de `types` y 7 de `fhir`; más D483 y D688, `is Any`). Java responde null, y este motor y cql-execution, false. Appendix B, Is: true «if the run-time type of the argument is the same as or derived from the type being tested», y false si no; la especificación no dice cuál es el tipo de null. Se lee que null no es de ningún tipo.
2. **`cast null as T` es null**, también cuando T no se relaciona con el tipo estático (14 filas de Java en `fhir`). Java rechaza esos casts al traducir, y este motor da null. *(Corregido en la ronda adversarial: se le dijo al usuario que cql-execution también da null, pero no responde nada: el arnés solo le pasa lo que el traductor acepta, y `fhir.js.json` no tiene esas 14 definiciones. Lo que sí hay: en un cast relacionado de null, D585, los tres dan null.)* Es la misma clase de cast que la decisión 1 del PR 3 dio por CQL válido.
3. **Arithmetic:** en D380 y D450, el piso a 8 del ADR 002 lleva el cociente de un `div` a otro entero, y la fila es `deliberate`. En D80, este motor da el valor exacto, 0, y cql-execution da 0.00999…, porque calcula `mod` en binario: es `reference defect`.
4. **D194 y D239: se queda la respuesta del motor.** En D194, `Sum` de un value choice suma el valor según su tipo en ejecución (-5 'mg'); las dos referencias eligen la sobrecarga de Integer y dan null. En D239, `same day as` toma las dos conversiones seguidas (FHIR.date → Date → DateTime) y da false; Java no resuelve la llamada.

### Revisión previa

- **Qué decide hoy cada oráculo** (sonda del 20-10-2026 sobre los corpus enteros, no solo sobre las divergencias):
  - `types`: 16 definiciones con «null's run-time type», 10 de ellas con divergencia; 2 con «empty list», sin divergencia.
  - `fhir`: 23 con «null's run-time type» (22 con divergencia); 5 con «converting each element» (1, D194); 1 con «two conversions» (D239).
  - `arithmetic`: las 3 filas son ambiguas en «a truncated quotient at a whole number». La ambigüedad sale de `widen`: un literal como 0.1, que no es un binary64, se ensancha para admitir una implementación en binario, y entonces `1.0 mod 0.1` puede caer a cada lado de 10.
- **`gofhir/server`:** no aplica.
- **Las 16 `unclassified`** son respuestas que el arnés no codifica (Code, Concept, Tuple y los números de JS; pendiente 1b del README). Quedan fuera: van en un PR 7 del arnés.

### Mecánica

- **`types` y `fhir`:** el caso de null deja de ser `undecided`: `is` da false y `cast` da null, con una razón que cita la decisión. Si el motor da eso y la referencia no, la fila es `deliberate`. Si la referencia da un error interno, que no es ninguna lectura (D483 y D688: «Could not resolve type …runtime.Any»), es `reference defect`.
- **`fhir`, conversiones:** con `converting each element`, el oráculo calcula el agregado sobre el valor en ejecución de cada elemento; con `two conversions`, sigue por la firma alcanzada, como con una sola conversión. Si el motor coincide y la referencia no, la fila es `deliberate`.
- **`arithmetic`:** en una fila ambigua solo por un cociente truncado, el oráculo calcula además la **lectura decimal exacta**: el mismo árbol a 40 cifras, sin `widen`. Entonces:
  - el motor sigue la política y la referencia da la lectura exacta: `deliberate`, por el ADR 002 (D380 y D450);
  - el motor sigue la política y da la lectura exacta, y la referencia no: `reference defect`, porque el tipo Decimal es decimal (Types, Decimal) (D80);
  - en otro caso, sigue `undecided`.
- Las pruebas «donde todos coinciden» (`TestTheTypesOracleAgreesWhereAllAgree` y la de `fhir`) pasan a revisar las 6 + 1 + 4 definiciones sin divergencia que hoy saltan por `undecided`.

### Guardas y mutaciones

Mutaciones nuevas en `003-evidence.sh`, cada una debe fallar:
- `null is T` da true, y da null;
- `cast null as T` da un error;
- el agregado de un choice da null; las dos conversiones, un error de traducción;
- la lectura exacta con `widen`;
- D80 con la referencia tomada por exacta, y D380 con el motor fuera de la política;
- D483 `deliberate` en vez de `reference defect`.

Se corren también las 84 del 003 y las 30 del 002.

### Criterio de salida

Ninguna fila queda `undecided`. `pending-verdicts.tsv` baja de 54 a 16, todas `unclassified` del arnés. Pasan `make check` y `make evidence`, y se detectan todas las mutaciones.

### Lo que queda fuera

- Las 16 filas que el arnés no codifica (PR 7). Sin ellas, el ADR 001 no pasa a «Aceptado».
- Cambiar el motor: ninguna decisión lo pide.

### Ronda adversarial del PR 6 (Codex y Fable)

| # | Hallazgo | Evidencia | Resolución |
|---|---|---|---|
| A1 (alto, los dos) | Con quitar el `undecided`, D194 queda `own defect`: el oráculo resuelve `Sum` por el tipo estático (`List<System.Integer>`) y lleva la Quantity a null | `engine_fhir_oracle_test.go`, resolución por costo y cast implícito de un choice | **Aceptado.** Con «converting each element», la sobrecarga se elige por el tipo en ejecución de cada elemento. Un elemento que no alcanza ninguna sobrecarga da null: D374 y D792 (`valueString 'abc'`) dan null en los tres motores. D194 entra en el manifiesto de las filas donde las dos referencias se equivocan |
| A2 (alto, Codex; medio, Fable) | 40 cifras sin `widen` no es una lectura exacta: `((1.0 / 3.0) * 3.0) div 1.0` da 0 a 40 cifras y 1 con racionales; Ln, Exp, Power fraccionaria y Sqrt son aproximaciones (`Power(0.00000001, -0.5) mod -5`, D116) | Sondas en Python de los dos | **Aceptado.** La lectura exacta es un intervalo: sin la holgura de implementación de `widen`, y con un margen de 10^-35 en cada operación que redondea (la que deja 40 cifras o es trascendente). Decide solo si todo el intervalo da el mismo entero; si no, la fila sigue `undecided`. D380 (65499,99998), D450 (−9,99999991) y D80 (cociente exacto 10) deciden; `(1.0 / 3.0) * 3.0` y D116 no. La respuesta de la referencia se compara por igualdad con ese entero |
| A3 (medio, Fable) | No hay camino `deliberate` para las filas de null ni para D239: `classifiedByTypesOracle` solo da `own defect` y `reference defect`, y `fhir` da `deliberate` solo con un cast `unrelated` | Código de los dos clasificadores | **Aceptado.** El veredicto lleva la decisión que lo funda, y esa decisión lleva a `deliberate`. Los 14 casts ya caen en `deliberate` por el camino de `unrelated` |
| A4 (medio, Fable) | `TestTheDivergencesAreConsistent` exige que el documento citado nombre cada fila, y el plan no nombra las 16 de null | `engine_oracle_test.go` | **Aceptado.** Las cuatro decisiones van a un **ADR 003**, con un bloque de filas generado y su test, como el de la decisión 1 del PR 3 |
| A5 (medio, Fable) | Las decisiones son reglas del motor para todo programa (null, cast de null, conversiones), y el ADR 002 se escribió porque una regla así no estaba en ningún documento | NUCLEO §2.2; plan, PR 4 | **Aceptado:** ADR 003, «Lo que la especificación no decide», con la sección de la especificación de cada decisión y su verificación |
| A6 (bajo, Codex) | Se le atribuyó a cql-execution una respuesta a los 14 casts que no existe | `fhir.js.json` | **Aceptado:** corregido arriba, y se le informa al usuario |
| A7 (bajo, Fable) | Faltan mutaciones | — | **Aceptado:** la lectura exacta con el piso a 8 (D380 pasa a `reference defect`), una fila de null con la referencia como `reference defect`, la resolución estática en el caso por elemento (D194 `own defect`), y un elemento sin sobrecarga que da error (D792). Además se revisa el diff del TSV, no solo las mutaciones |

**Medido por los revisores:** las definiciones sin divergencia que pasan a decidirse coinciden con las nuevas lecturas. Son D86, D100, D176, D199, D312 y D746 de `types`, y D585 de `fhir`: casts de null que dan null en los tres. D154 da -5 'mg' en los tres, y D367, D374 y D792 dan null. Las cinco ambiguas de `arithmetic` sin divergencia (D116, D406, D55, D555 y D694) coinciden con el motor.

### El diseño del PR 6, ajustado

- **ADR 003**, «Lo que la especificación no decide»: las cuatro decisiones, con su sección de la especificación, las filas que funda (bloque generado) y su verificación.
- **`types` y `fhir`:** `is` de null da false, y `cast` de null da null. La fila es `deliberate` por el ADR 003. D483 y D688 son `reference defect`.
- **`fhir`, conversiones:** la sobrecarga del agregado se elige por el tipo en ejecución de cada elemento, y un elemento sin sobrecarga da null. Las dos conversiones seguidas se toman. D194 y D239 son `deliberate`.
- **`arithmetic`:** la lectura exacta certificada de A2. D380 y D450 son `deliberate` por el ADR 002, y D80 es `reference defect`.
- **Criterio de salida:** sin cambios. 54 → 16 filas pendientes, y el resto queda para el PR 7.

### Aprobación del PR 6 (20-10-2026)

El usuario mantuvo la decisión 2 tras la corrección de A6, y aprobó el diseño ajustado.

### Implementación del PR 6

- **ADR 003** (`docs/adr/003-lo-que-la-especificacion-no-decide.md`), «Aceptado»: las cuatro decisiones, con un bloque de filas por área que escribe el `-classify` de cada una y que comprueba `TestTheUndecidedADRNamesItsRows`.
- **`types` y `fhir`:** el veredicto lleva `decided`. Con la respuesta del motor, la fila es `deliberate` si la referencia no da un error en ejecución (D483 y D688 quedan `reference defect`). Los 14 casts siguen el camino `unrelated` del PR 3, y su evidencia nombra también la decisión 2.
- **`fhir`, conversiones:** `operateByElement` elige la firma por el tipo en ejecución del elemento, y da null si ninguna lo toma. Las dos conversiones seguidas se toman con `decided`. D194 entra en `fhirBothReferencesWrong`. `operateOnNull` deja de devolver un error que siempre era nil (lint).
- **`arithmetic`:** las funciones `conform` pasan a ser métodos de `aReading`. `specification` es la lectura de siempre; `exactly` cambia `widen` por un margen de 10^-35 en cada valor que redondeó el oráculo. `TestTheExactReadingHoldsItsControls` tiene seis controles hechos a mano. La prueba de la decisión 1 del PR 4 cuenta solo las filas que citan el plan.
- **`TestTheDivergencesAreConsistent`:** su regex de cita pasa a ser `decisionCited`, que comparte con `cites`.
- **Resultado:** 38 filas con veredicto: 35 `deliberate` y 3 `reference defect` (D483, D688 y D80). `pending-verdicts.tsv` baja de 54 a 16, todas `unclassified`. Ninguna otra fila de los manifiestos cambió.
- **Mutaciones:** 98 de 98 en `003-evidence.sh` (223 s, 0,53 GB de RSS) y 30 de 30 en el 002. Desde este PR, ninguna fila es `undecided`. Por eso las seis mutaciones de una fila `undecided` primero la fabrican: D380 js, con las lecturas que tenía, y su control comprueba que esa fila pasa tal como queda.

### Review del PR 6, ronda 1 (Codex y Fable)

La primera corrida de Codex no ejecutó nada: su sandbox no escribe en el disco externo, donde ahora está el scratchpad. Se repitió con su copia en `/private/tmp`.

| # | Hallazgo | Evidencia | Resolución |
|---|---|---|---|
| R1-1 (medio, Codex; bajo, Fable) | La lectura exacta deducía el redondeo por las cifras: un cociente que el oráculo redondea a 1 pasaba por exacto, y `div` truncaba el cociente sin su margen | `99999999999999999998.999999999999999999999 div 99999999999999999999.0` daba 1, y el valor verdadero es 0 | **Aceptado.** `aConform.rounded` lo marca cada operación que redondea: un cociente es exacto solo si `q·b = a`. Se marcan también Ln, Log, Exp, la raíz y la potencia topada. `div` y `mod` truncan el cociente con su margen. Los dos casos son controles nuevos |
| R1-2 (medio, Codex) | Bajo −60, `exp` da 10^-60 en vez de la exponencial, y el margen no la contiene | `((Exp(-61) * 10000000000000000000.0) div 0.00000001)` daba 0, y el valor verdadero es 3 | **Aceptado.** La lectura exacta la acota entre 0 y e^-60, y el caso es un control |
| R1-3 (medio, Codex) | Si el cast de null de `types` volvía a `undecided`, los seis casts dejaban de probarse en silencio | mutación de Codex: pasa | **Aceptado.** `TestTheTypesOracleAgreesWhereAllAgree` falla con cualquier `undecided` que no sea el de la lista vacía. En `fhir`, que ya no deja nada sin decidir, se quita el campo `undecided` con sus ramas |
| R1-4 (medio, Fable) | Falta la mutación «D380 con el motor fuera de la política» que pedía el plan | sonda: el código de la guarda no tiene ninguna fila que lo ejercite | **Aceptado:** mutación de datos (D380 da 65501 en `arithmetic.answers.tsv`) |
| R1-5 (bajo, Fable) | El ADR dice que sus bloques tienen todas las filas que funda, pero los 14 casts citan primero el plan | evidencia de D112 | **Aceptado:** el texto ya lo dice |
| R1-6 (bajo, Fable) | Falta la mutación del cast de null en `types` | sonda | **Aceptado:** el cast que falla y el cast `undecided` |
| R1-7 (bajo, Fable) | El campo `exact` se confunde con el método `exact()`; los controles `undecided` no fijan el porqué | — | **Aceptado:** el campo pasa a llamarse `exactReading`, y un control `undecided` exige la ambigüedad del cociente truncado |

Mutaciones nuevas: el cociente redondeado tomado por exacto, `div` sin margen, y la exponencial bajo −60. Corridas completas: 104 de 104 en el 003 y 30 de 30 en el 002. `make check` estuvo en verde antes de estas correcciones (759 s, 1,26 GB de RSS) y se vuelve a correr al cerrar.

### Review del PR 6, ronda 2 (Codex y Fable)

Los dos confirmaron las correcciones de la ronda 1. Fable midió que la lectura de la especificación es idéntica byte a byte en las 800 definiciones, comparada con HEAD.

| # | Hallazgo | Evidencia | Resolución |
|---|---|---|---|
| R2-1 (medio, los dos) | Log y Power fraccionaria ponían el margen después de dividir o multiplicar logaritmos redondeados: un logaritmo chico amplifica su error hasta 10^8 veces | `Log(2, 1.00000001)`: error de 3,2e-25 frente a un margen de 6,9e-28 (Fable); un Log de bases cerca de 1 certificaba 0, y el valor verdadero es -1 (Codex). Ninguna fila del corpus cambia | **Aceptado.** `lnOf` acota cada logaritmo con su margen antes de dividirlo o multiplicarlo, solo en la lectura exacta. Un denominador que llega a 0 queda ambiguo. Son dos controles nuevos |
| R2-2 (medio, Codex) | `roundTraditional` usaba `Div`, que redondea a 16 cifras sin marcarlo | `Round(0.1234567890123456789, 19)` daba `0.1234567890123457`. El corpus redondea a 8 cifras como mucho | **Aceptado:** se escala con `Shift`, en forma exacta. Es un control nuevo |

Mutaciones nuevas: un logaritmo acotado después de dividirlo, y `Round` por `Div`. Corridas completas: 106 de 106 en el 003 y 30 de 30 en el 002.

### Review del PR 6, ronda 3 (Codex y Fable): converge

Ninguno de los dos dejó hallazgos. Lo que verificaron:
- **Codex** repitió sus contraejemplos de la ronda 2. El Log de bases cerca de 1 queda ambiguo, y una cuenta a 90 cifras da −1,99…, que se trunca a −1. El Round a 19 cifras certifica 0. El valor a 90 cifras de `Power(1.0000000000001, 99999999999999.5)` cae dentro de su intervalo.
- **Fable** midió contra 70 cifras que los intervalos de `Log(2, 1.00000001)`, `Log(0.12345678, 0.99999999)` y `Power(1.00001, 4000000.5)` contienen el valor, y que Ln y ExpTaylor a 40 cifras se apartan menos de 1,1e-40 de su valor a 70: el margen de 10^-35 alcanza. La lectura de la especificación es idéntica byte a byte a HEAD en las 800 definiciones: las 66 con Round redondean a 8 cifras como mucho. `go test ./...` pasa.
