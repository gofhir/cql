# 004: los campos de Code y Concept distinguen null de `''`

Estado: plan, 21-10-2026. Pendiente de la ronda adversarial y de la aprobación del usuario.

## Por qué

`types.Code` y `types.Concept` guardan sus campos como `string`, y el motor lee `""` como null. Un `''` explícito no se puede distinguir de un campo que nadie fijó. Viene del backlog (1b de `docs/plans/README.md`, R3-2 del PR 7 del 003).

El usuario eligió, el 21-10-2026, **campos `*string`** (nil es null), en vez de marcas de presencia o de dejarlo documentado.

## Revisión previa

**Especificación** (CQL 1.5, `cql.hl7.org/09-b-cqlreference.html`):
- *Types, Code*: `structured type Code { code String, display String, system String, version String }`. `''` es un String, no null.
- *Equal (=)*: un tipo estructurado se compara elemento por elemento, como una conjunción. Los ejemplos de tuplas dan:
  - true si los dos lados tienen null en el mismo elemento;
  - null si un lado tiene null donde el otro tiene un valor y nada más difiere;
  - false si algún par de valores difiere.

  El texto no nombra Code ni Concept, así que se aplica la regla de los tipos estructurados.
- *Equivalent (~)*: «For Code values, equivalence is defined based on the code and system elements only… The version and display elements are ignored». «For Concept values, equivalence is defined as a non-empty intersection of the codes». Siempre da true o false, nunca null.

**Java 3.26.0 contra este motor** (sondas del 21-10-2026, `P.cql` y `Q.cql` en el scratchpad; se pasan a tests del repo):

| Definición | Java | Este motor |
|---|---|---|
| `Code{code:'x', system:''}.system is null` | false | true |
| `Code{code:'x', system:''}.system` | `''` | null |
| `Code{code:''}.code is null` | false | true |
| `Concept{codes:{…}, display:''}.display is null` | false | true |
| `Code{code:'x', system:''} = Code{code:'x'}` | null | true |
| `Code{code:'x', system:''} ~ Code{code:'x'}` | false | true |
| `Code{…display:'a'} = Code{…display:'b'}` | false | true |
| `Code{…version:'1'} = Code{…version:'2'}` | false | true |
| `Code{code:'x'} = Code{code:'x', system:'s'}` | null | false |
| `Concept{…display:'a'} = Concept{…display:'b'}` | false | true |
| `Code{code:'x'} = Code{code:'x'}` | true | true |
| `Code{code:'x'} ~ Code{code:'x'}` | true | true |
| `Concept{display:'d'}.codes` | `[]` | `{}` (coinciden) |

La primera mitad de la tabla es el `''`. La segunda es otro defecto: la igualdad de Code y Concept ignora `display` y `version`, y no propaga el null. Va en el mismo incremento porque el cambio de representación obliga a reescribir esa igualdad.

**Este motor:**
- 13 lugares construyen un Code o un Concept:
  - `eval/evaluator.go`: el selector, `buildCode` y `buildConcept`, y las definiciones `code` y `concept`;
  - `eval/context.go` y `eval/terminology.go`: las definiciones de la librería;
  - `eval/terminology_expand.go`;
  - `funcs/type_ops.go`: `ToConcept`;
  - `conformance/output_parser.go`.
- Unos 30 lugares leen sus campos.
- La lectura de un campo pasa por `optionalString`, que convierte `""` en null.
- `Code.Equal` compara solo `system` y `code`.
- `tupleEqual` ya implementa la regla de la especificación para tuplas.

**`gofhir/server`** (congelado, único consumidor): no construye ni lee campos de Code o Concept (grep del 21-10-2026). Solo usa `cqltypes.List` y `cqltypes.Interval`.

**Tomar, mejorar, descartar:**
- **Tomar** `tupleEqual` para la igualdad de Code y Concept, vistos como tuplas de sus elementos.
- **Descartar** las marcas de presencia (decisión del usuario).

## Alcance (rehecho tras la ronda adversarial, 21-10-2026)

**PR 1, representación, construcción, lectura e igualdad:**
- `types.Code{System, Code, Display, Version *string}` y `types.Concept{Codes []Code; Display *string}`. `NewCode` recibe también la versión. Hay un ayudante `Str(s) *string`. La igualdad de los campos compara la presencia y el contenido, nunca la dirección (A4).
- **Construcción:** cada lugar dice si un campo es null.
  - El selector usa null para un elemento que no se fijó o que vale null, y `''` para uno que vale `''`.
  - La definición `code` toma su `display` y la `version` del codesystem, y la conserva desde el AST, el compilador y el ELM, donde hoy `Display string` y `omitempty` la pierden (A3, A9).
  - FHIRHelpers pasa el null de un `Coding` sin `system`.
  - `ToConcept` deja el `display` del Concept en null (A10).
- **Lectura:** un campo nil se lee como null, y `''` como `''`.
- **Igualdad** (`=`, escalar y dentro de listas, intervalos y tuplas, por `elementEquality`), A1, A7 y A8:
  - una conjunción de los cuatro elementos de un Code, y de `codes` (como lista, en orden) y `display` de un Concept;
  - si un par difiere, da false; si no, y algún par es indeterminado, da null; si no, true.
  - Es la lectura de la especificación («a conjunction of equality comparisons»). Java no la sigue: da null si cualquier elemento es null en un solo lado, aunque otro difiera. Esas divergencias se deciden por la especificación, con su veredicto (decisión del usuario, ver abajo).
- **Equivalencia** (`~`): `code` y `system`, sin distinguir mayúsculas, como hoy. `''` y null son distintos, y el resultado nunca es null.

**PR 2, las operaciones que comparan Codes sin pasar por `=`** (A5 y A6):
- `distinct`, `union`, `intersect`, `except`, `in`, `contains`, `IndexOf`, `Mode` y el `return` de una consulta, que es distinct por defecto.
- Hoy pasan por `sameValue` o usan el `String()` como clave, y juntan Codes que difieren en `display` o `version`. Java da `Count(distinct …) = 2` y este motor 1.
- Pasan a la igualdad del PR 1, y un resultado indeterminado cuenta como distinto, como en Java.
- El parser de conformidad lee `null` como null y lee la `version` (A11).

**Fuera** (al backlog, con su sonda):
- **La terminología** (`TerminologyProvider.InValueSet`, `ExpandedCode`, `extractCodeComponents`) sigue aplanando a `string`. Llevar la presencia hasta el proveedor sería una segunda ruptura de API. Una expansión con `''` se lee como null, igual que hoy (A2).
- **La igualdad de tuplas** (`tupleEqual`) se decide en el primer elemento por orden alfabético, no por conjunción. Java hace lo mismo, y los dos contradicen el ejemplo de la especificación (`{x:1,y:1} = {x:null,y:2}` es false). Es un frente aparte (A1).
- **`Concept.codes` null:** Java da `[]`, igual que este motor.

## Versión

El cambio de tipo de los campos rompe la compatibilidad de `types.Code` y `types.Concept`. Hay dos salidas: v1.24.0 o v2. El único consumidor, `gofhir/server`, importa `cqltypes` solo para `List` e `Interval` (verificado).

Recomendación: **v1.24.0**. Los commits van como `fix:` o `feat:` y la ruptura se anota a mano en el CHANGELOG: `release-please` (`release-type: go`) subiría a v2.0.0 con un `!` o un `BREAKING CHANGE`, y Go rechaza una v2 sin la ruta `/v2` (A12). **Lo decide el usuario.**

## Tests y datos

- **Casos a mano:** las 13 sondas de la tabla, como test del paquete raíz. El valor esperado sale de Java (las sondas) y de la especificación, citada en el comentario.
- **Diferencial masiva** contra Java antes y después (regla permanente):
  - las áreas `types` y `fhir` completas;
  - un corpus nuevo, `codes`, generado por `gen`, con Code y Concept que mezclan `''`, null y valores en sus cuatro campos, con `=`, `~`, el acceso a campos, `is null`, `in` sobre listas y conversiones.
  - Cientos de definiciones, con sus referencias de Java y de cql-execution, y su manifiesto de divergencias, que tiene que quedar sin `unclassified`.
- **Metamórficos:** reordenar los elementos de un selector da el mismo Code, y `a = b` es igual a `b = a`.
- **Regresión:** los casos eCQM, la conformidad de cql-tests, `runtime` y los snapshots. Toda respuesta que se mueva se explica en el plan.
- **Sondas adicionales** de la ronda (Fable, 36 definiciones en `R.cql`, con sus respuestas de Java): se pasan al repo como test.
- **Mutaciones**, en un script nuevo, `scripts/mutations/004-code-fields.sh`:
  - la igualdad compara direcciones en vez de textos;
  - se pierde la `version` del codesystem;
  - `elementEquality` vuelve a la igualdad booleana de Code;
  - se ignora el orden de los codes de un Concept;
  - `ToConcept` copia el `display`;
  - `distinct` junta Codes que difieren en `display`;
  - la igualdad decide en el primer elemento, no por conjunción;
  - `optionalString` vuelve a convertir `""` en null;
  - el selector toma `''` donde no se fijó el elemento;
  - la igualdad ignora `display`;
  - la igualdad ignora `version`;
  - la igualdad da false donde la especificación dice null;
  - `~` mira `display`;
  - `~` da null;
  - FHIRHelpers toma `''` por un `system` ausente.

## Criterio de salida

- Las 13 sondas dan lo mismo que Java.
- El corpus `codes` no deja divergencias `unclassified`, y las que quedan tienen veredicto.
- Ninguna otra área cambia de respuesta sin explicación.
- Se detectan todas las mutaciones, y pasan `make check` y `make evidence`.

## Riesgos

- **Escritura y lectura que cambian juntas:** cada lector de campos tiene que manejar nil. El compilador marca todos los usos de `string`, así que no puede quedar uno en silencio.
- **Terminología:** queda fuera, y sigue aplanando como hoy. Lo confirman los casos eCQM y la conformidad.
- **`distinct` y el `return` de una consulta** dejan de juntar Codes que difieren en `display` o `version`. Se mide sobre los casos eCQM y sobre todos los corpus.

## Ronda adversarial (Codex y Fable, 21-10-2026)

| # | Hallazgo | Resolución |
|---|---|---|
| A1 (alto, los dos) | `tupleEqual` decide en el primer elemento (por orden alfabético), no por conjunción. Java tampoco sigue la conjunción: da null si un elemento es null en un solo lado | **Aceptado:** la conjunción es propia del Code y el Concept. La de las tuplas va al backlog. La divergencia con Java la decide el usuario |
| A2 (alto, los dos) | La terminología aplana a `string` (`InValueSet`, `ExpandedCode`, `extractCodeComponents`) | **Aceptado:** fuera de alcance, explícito |
| A3 (alto, los dos) | La `version` del codesystem nunca llega al Code de una definición `code`. Java: `"A".version` da '2', y este motor null | **Aceptado:** va en el PR 1. Si no, `=` contra un Code con versión retrocede de true a null |
| A4 (alto, Codex) | Con punteros, un `==` compara direcciones y sigue compilando | **Aceptado:** la igualdad de los campos compara presencia y contenido, con su mutación |
| A5 (medio, los dos) | `distinct`, `union`, `in` y `return` juntan Codes que difieren en `display` o `version` (Java: 2, este motor: 1) | **Aceptado:** PR 2 |
| A6 (medio, Codex) | `Distinct` y `Mode` usan el `String()` como clave | **Aceptado:** PR 2 |
| A7 (medio, Fable) | `elementEquality` usa el `Equal` booleano: un Code dentro de una lista no propaga el null | **Aceptado:** PR 1 |
| A8 (medio, Fable) | El PR 1 no puede ser «solo representación»: el `Equal` de un `*string` ya cambia `=` | **Aceptado:** la igualdad pasa al PR 1 |
| A9 (alto, Codex) | El AST, el compilador y el ELM guardan `Display string` con `omitempty`, así que pierden el `display ''` (Java: `"C2".display is null` da false) | **Aceptado:** PR 1 |
| A10 (bajo, Fable) | `ToConcept` copia el `display` del Code al Concept (Java: null) | **Aceptado:** PR 1 |
| A11 (medio, Codex; bajo, Fable) | El parser de conformidad lee `null` como texto e ignora la `version`, y su test usa `Equal` | **Aceptado:** PR 2, con expectativas que no dependen del motor |
| A12 (medio, Fable) | `release-please` sube a v2.0.0 con un commit `!` o `BREAKING CHANGE` | **Aceptado:** se commitea como `fix:` o `feat:`, y la ruptura se anota a mano |

## Aprobación (21-10-2026)

El usuario aprobó el plan rehecho, con A1 a A12, y decidió:
- **Igualdad:** se sigue la especificación (conjunción). Como la especificación decide, una divergencia de Java en la igualdad es `reference defect`, con la cita del texto: no es `deliberate` y no necesita un ADR.
- **Versión:** v1.24.0. Los commits van como `fix:` o `feat:`, y la ruptura se anota a mano en el CHANGELOG.

## Implementación del PR 1 (21-10-2026, sin commit)

- **Representación:** `types.Code` guarda `System`, `Code`, `Display` y `Version` como `*string`, y `types.Concept` guarda `Display` como `*string`. También son `*string` la versión del codesystem y el `display` del AST y del ELM. Un `display ''` declarado llega como String.
- **Igualdad:**
  - `codeVerdict` y `conceptVerdict` son una conjunción, dentro de `containerVerdict`, así que valen para `=` escalar y para un Code dentro de una lista, un intervalo o una tupla.
  - `~` sigue comparando solo código y sistema, sin mayúsculas, y distingue `''` de null.
- **Tests a mano:** `engine_code_fields_test.go`, 42 casos. El valor esperado sale de Java o de la especificación, y los dos casos de conjunción están marcados.
- **Corpus `codes`:** `gen -area codes -seed 1 -n 600` (`testdata/snapshots/codes.cql`).
  - Su preámbulo declara un codesystem con versión, codes con display `''` y sin display, y dos concepts.
  - Las definiciones combinan acceso a campos, `is null`, `=`, `!=`, `~` y `!~` sobre Code y Concept, y `=` y `!=` sobre listas de Codes, incluidas listas con null en la misma posición.
  - Las referencias se corrieron con `Evaluate -each` y `evaluate.js --each`.
- **Oráculo:** `engine_codes_oracle_test.go`.
  - Lee el texto del corpus (el preámbulo y los selectores) y calcula la respuesta con la especificación.
  - Tiene tres tests: los veredictos, la coincidencia donde los tres concuerdan y la lectura del corpus entero.
  - El flag `-classify-codes` escribe los veredictos.
  - El 21-10-2026 la deuda queda en cero. El manifiesto, `testdata/snapshots/codes.divergences.tsv`, queda así:

    | Referencia | Clase | Cuántas | Qué son |
    |---|---|---|---|
    | java | reference defect | 90 | la conjunción de los campos de un Code: Java da null cuando un null va primero |
    | js | reference defect | 67 | `~` distingue mayúsculas, entre otras |

    (Cifras tras la ronda 2 de review. Antes había 7 `own defect` de listas, ver «Review, ronda 1» y «Review, ronda 2».)

- **Listas de dos Codes:** al principio el oráculo las leía como una conjunción, y daba 7 `own defect`. En la ronda 1 se vio que la especificación no lo decide (ver «Review, ronda 1»).
- **El oráculo y la especificación:** en una lista, «null elements are considered equal». Lo confirma el ejemplo `ListEqualIsTrue`, y el oráculo lo aplica.
- **Mutaciones:** `scripts/mutations/004-code-fields.sh`, 25 en total al principio (31 tras la ronda 1); se detectan todas.
  - Tuvieron que endurecerse tres:
    - la de `ToConcept`, que no se aplicaba (el texto estaba dos veces);
    - la del arnés que escribía `''` como null: se agregó el caso «a Code writes an empty field as a String», que Java confirma con `system: ''`;
    - la del oráculo con dos null en la misma posición: el generador ahora pone `(null as Code)` en las dos listas.
  - Quedó fuera «`~` da null»: `cqlEquivalent` devuelve un bool, así que en esa capa no se puede escribir.
  - `distinct` y FHIRHelpers van con el PR 2.

## Review, ronda 1 (Codex y Fable, 21-10-2026)

| Hallazgo | Verificación | Resolución |
|---|---|---|
| R1-1 (medio, los dos) | `Code '1' from "CS"` guardaba el sistema con las comillas: daba `.system` = `"\"CS\""` y `.version` null. Sonda en una copia | **Corregido:** `VisitCodeSelector` usa `undelimitedIdentifier`, como la definición de un code. Java confirma `version` '2' y el id del sistema. Hay 6 casos nuevos a mano, el generador escribe estos selectores y el oráculo los lee |
| R1-2 (medio, Codex) | Ninguna mutación cubría que el ELM JSON lea un display `''` como null | **Corregido:** `elm/code_fields_test.go` hace la ida y vuelta CQL → ELM JSON → AST de display y versión, con `''` y sin la cláusula. Tiene su mutación |
| R1-3 (medio, Fable) | `buildConcept` descarta un `(null as Code)` de `codes` (anterior a este PR), y el generador los escribía | **Fuera del alcance** (Concept.codes null, en el backlog): el generador ya no pone null dentro de un Concept |
| R1-4 (bajo, Fable) | `textOrNull` no tenía test | **Corregido:** `TestExpandValueSet_EmptyIsNull`, con su mutación |
| R1-5 (bajo, Fable) | El sistema de respaldo de un code con codesystem no declarado no tiene test | **Rechazado:** un test fijaría algo que sema debería rechazar. Va al backlog |
| R1-6 (bajo, Fable) | El comentario de `optionalString` había quedado sobre `optionalElement`; errata `conceptNams` | **Corregido** |
| R1-7 (bajo, Fable) | `~` entre Code y Concept es asimétrico (anterior a este PR) | **Backlog** |

**Lo que encontró el corpus al regenerarlo** (D334, después de R1-1):
- En una lista de dos Codes con un par indeciso y otro desigual, la especificación no dice qué resulta. Su texto para listas («true if and only if… each element… equal») no es una conjunción, a diferencia del de las tuplas.
- Las referencias tampoco coinciden. Con un elemento `(null as Code)` contra un Code, los tres motores dan null. Con un Code de campos null, JS da false y Java null, igual que este motor.
- Por eso el corpus solo escribe lo que la especificación decide: listas de un Code, o con null contra null en la misma posición antes del Code («null elements are considered equal»).
- La pregunta queda en el backlog (1c2) para que la decida el usuario, y con ella desaparecen los 7 `own defect` de listas.

## Review, ronda 2 (Codex y Fable, 21-10-2026)

| Hallazgo | Verificación | Resolución |
|---|---|---|
| R2-1 (medio, los dos) | La exclusión de la ronda 1 solo cubría las listas sueltas. Los `codes` de un Concept son la misma lista, y el corpus tenía 7 definiciones (D17 y otras) con un par indeciso y otro desigual: 6 eran `reference defect` de Java, por una regla que la especificación no decide | **Corregido:**<br>• el generador compara con `=` y `!=` solo Concepts de un code, o declarados;<br>• el oráculo rechaza una lista así («the specification does not decide»), así que `TestTheCodesOracleReadsTheWholeCorpus` falla si el corpus vuelve a tenerla. Contra el corpus anterior da 13 |
| R2-2 (medio, Codex; bajo, Fable) | `Code '1' from H."CS"`, con un codesystem de una librería incluida, no lo resuelve: da `system` "H.CS" y `version` null | **Backlog:** la resolución de codesystems entre librerías es anterior a este PR (también falla en un code declarado) y es otro frente |
| R2-3 (bajo, Fable) | El script de mutaciones necesita un checkout de git (`git ls-files`) | **Rechazado:** es igual que el de 003; el repositorio siempre es un checkout |
| R2-4 (bajo, Fable) | `gofmt` había convertido `''` en `”` en un comentario | **Corregido** |

**Al rehacer el corpus,** dos mutaciones del oráculo, la versión de un selector y la de un code declarado, dejaron de tener un caso que decidiera. Ahora el generador lee los campos de los codes declarados y de los selectores con codesystem, y se vuelven a detectar las 31.

## Review, ronda 3 (Codex y Fable, 21-10-2026): converge

- Ninguno de los dos dejó hallazgos de severidad media o mayor.
  - Codex probó la guarda con una lista suelta y con unos `codes` ambiguos, y el oráculo leyó 100.000 expresiones generadas con la semilla 37.
  - Fable revisó las 157 filas. Las de Code y Concept se apoyan en «a conjunction of equality comparisons», y las de listas con `(null as Code)` en «null elements are considered equal». Ninguna se apoya en una lectura que la especificación deje abierta.
- Un hallazgo bajo, corregido: la etiqueta «backlog 1d» ahora dice «1c2».
- Al hacerlo, el lint marcó un `dupCase` en `fieldOperand`. Se reescribió con `if` encadenados, que consumen el mismo azar, y el corpus sale idéntico (`cmp`).

## Evidencia del PR 1

- `make check`, en verde el 21-10-2026, después de la ronda 3: lint, `-race`, las seis zonas y las mutaciones de 001 a 004.
- `make evidence`: [docs/evidence/2026-10-21.md](../evidence/2026-10-21.md). El área `codes` no tiene ninguna divergencia `unclassified`, y `004-code-fields.sh` termina con código 0.
- **Al hacer el commit:** para que release-please publique v1.24.0, el commit tiene que ser `feat:`; un `fix:` daría v1.23.4. La ruptura de `types.Code` y `types.Concept` se anota a mano en el PR del release.

## Implementación del PR 2 (21-10-2026, sin commit)

- **La causa:**
  - Las operaciones que guardan valores comparan con `sameValue`, que termina en `Code.Equal`, y `Code.Equal` comparaba solo el sistema y el código.
  - `Mode` agrupaba por `Type()+String()`, y `String()` no lleva la versión y escribe el null como vacío.
- **El cambio:**
  - `Code.Equal` compara los cuatro elementos por presencia y texto (`SameText`).
  - `Concept.Equal` compara además el display.
  - `Mode` agrupa con `modeKey`, que cita cada elemento y marca el null.
  - Un `=` indeterminado cuenta como distinto, como dicen las dos referencias.
  - Las cubetas de `distinct` no cambian: los Codes no tienen cubeta y se comparan de a pares.
- **A11:** `parseCode` lee `null` como null y lee la `version`, y el display de un Concept pasa por el mismo `codeElement`. El corpus de conformidad tiene una sola salida de Code (`CodeToConcept1`, solo `code`), así que la cubre `TestParseExpectedOutput`.
- **Casos a mano:** `TestCodeOperationsTellCodesApart`, 34 casos. Las sondas de Java y JS del 21-10-2026 están en el scratchpad, en `ops.cql` y `ops2.cql`. En main fallaban 14.
  - El IndexOf de Java compara por equivalencia (`IndexOf({Code{code:'X'}}, Code{code:'x'})` da 0), y el Mode de Codes falla («Code is not comparable»). Ahí manda JS, que coincide con la especificación («using equality semantics»).
  - Mode no lo decide la especificación, sino la decisión 5 del ADR 003 (ver «Review del PR 2, ronda 1»).
- **Corpus `codesets`:** `gen -area codesets -seed 1 -n 400`. Comparte el preámbulo y el oráculo con `codes`, y el oráculo lee sus operaciones. El oráculo rechaza un Mode con empate.
  - Manifiesto, tras la ronda 2:
    - Java, 52 `reference defect`: los errores de Mode y el IndexOf por equivalencia;
    - JS, 31 `reference defect`: Mode. cql-execution 3.3.2 cuenta con `counts[elem]` (`lib/elm/aggregate.js`), y como Code no define `toString`, todo Code es la clave `"[object Object]"`: cuenta todos los Codes como un solo valor y devuelve el último;
    - 10 `own defect` con cada referencia: `contains null` (ver la ronda 1).
  - No queda ninguna sin clasificar.
- **Mutaciones:** `004-code-fields.sh` tiene 45 (14 más que en el PR 1), y se detectan todas.
  - Dos tuvieron que cambiar. La de «el `return` no es distinct» sobrevivía porque las listas casi nunca repetían un Code; ahora el generador repite uno.
  - La de «Mode con `>=`» era equivalente al código correcto: con `{a, a, b}` no hay empate. Se reemplazó por «Mode toma el último valor distinto».
  - La de «Mode guarda null como vacío» necesitó un caso a mano.
- **Sin tocar:** `funcs.Distinct`, que solo usa su test.

## Review del PR 2, ronda 1 (Codex y Fable, 21-10-2026)

| Hallazgo | Verificación | Resolución |
|---|---|---|
| P2-1 (medio, los dos) | `modeKey` usaba `String()` dentro de una tupla o una lista: `Mode({Tuple{c: Code v1}, Tuple{c: Code v1}, Tuple{c: Code v2}}).c.version` daba '2' | **Corregido:** la clave es recursiva para tuplas y listas. Hay 3 casos a mano y 2 mutaciones |
| P2-2 (bajo, Fable) | La sección Mode de la especificación no dice cómo compara ni cómo desempata (verificado en cql.hl7.org: solo «the statistical mode») | **Lo decidió el usuario** el 21-10-2026: Mode por igualdad, como `distinct`. Es la **decisión 5 del ADR 003**:<br>• el oráculo calcula también la lectura por equivalencia;<br>• una referencia que la da es `deliberate`;<br>• una que no da ninguna de las dos es `reference defect`;<br>• el bloque `codesets` del ADR queda vacío: con la forma `{a, a, b}` del generador, la lectura por equivalencia o coincide con la igualdad o tiene más de una respuesta, así que el corpus no llega a la rama (ronda 2, H2). `TestTheModeDecisionIsTheOnlyReadingLeftAside` cubre esa rama |
| P2-3 (bajo, Fable) | El corpus no tenía elementos null ni Concepts | **Corregido en parte:**<br>• el generador pone `(null as Code)` en las listas y en lo que se busca, y el oráculo aplica las reglas de null de Count, IndexOf, In y Contains;<br>• los Concepts siguen solo en los casos a mano: en una lista de dos codes, la igualdad es la que no decide la especificación (1c2). Se sumó uno de `union` |

**Lo que encontró el corpus con los null:**
- `{ 1, 2 } contains (null as Integer)` y `(null as Integer) in { 1, 2 }` dan null. La especificación dice false («true if the list contains any null elements, and false otherwise»), y las dos referencias dan false.
- Pasa con cualquier tipo y es anterior a este PR: va al backlog (1c4).
- En `codesets` son 10 `own defect` con cada referencia, con su veredicto.

Las mutaciones son 52, y se detectan todas.

## Review del PR 2, ronda 2 (Codex y Fable, 21-10-2026)

Codex no terminó: el disco del sistema se llenó (ENOSPC) mientras guardaba su estado. Antes dejó dos pruebas en su copia, y las dos se ejecutaron.

| Hallazgo | Verificación | Resolución |
|---|---|---|
| R2-1 (medio, Codex) | La clave recursiva de Mode escribía el tipo de cada elemento: `Tuple{n: 1}` y `Tuple{n: 1.0}` son iguales (`=` da true, `distinct` da uno), pero Mode los contaba aparte. Su prueba fallaba: daba 2, no 1. La clave de la ronda 1 los agrupaba | **Corregido:** un Integer y un Decimal se escriben como el número. Hay un caso a mano y una mutación |
| R2-2 (bajo, Codex; H3 de Fable) | Si la clase por equivalencia difiere en el elemento leído, el oráculo no tiene respuesta alternativa, y una referencia que diera uno de sus valores quedaría `reference defect` | **Rechazado, con evidencia:** aceptar cualquier valor de la clase reclasificó D13 y D326 de JS como `deliberate`. Pero cql-execution no lee por equivalencia: cuenta todos los Codes como uno (ver el manifiesto) y coincide por casualidad. Cuál valor de la clase se queda no es una lectura de la especificación. `TestTheModeDecisionIsTheOnlyReadingLeftAside` fija ese caso como `reference defect` |
| H1 (medio, Fable) | El manifiesto atribuía los 31 de JS a «Mode con elementos null»: 30 no tienen ningún null | **Corregido:** la causa, verificada en el código de cql-execution, está en el manifiesto |
| H2 (medio-bajo, Fable) | La tabla de la ronda 1 atribuía a JS que el bloque `codesets` quede vacío; es la forma del generador | **Corregido** en la tabla |
| H4 (bajo, Fable) | `g.codeOr(x, "(null as Code)")` es `g.code(x)` | **Corregido;** el corpus regenerado es idéntico byte a byte |

Las mutaciones son 53.

## Review del PR 2, ronda 3 (Codex y Fable, 21-10-2026)

**Converge:** ninguno deja hallazgos de este PR de severidad media o mayor. Los dos sondearon la clave numérica de Mode (Integer y Decimal, escala, cero negativo, Long, números en listas y tuplas anidadas, `as Any`): agrupa todo lo que `=` iguala, y main no agrupaba ninguno. También separa ahora un String de un número en una tupla (`'1'` y `1`), que main confundía. Las mutaciones dan 53 detectadas en las dos copias.

| Hallazgo | Verificación | Resolución |
|---|---|---|
| Mode separa cantidades, Dates e intervalos iguales (medio para Codex; bajo, B1, para Fable) | `Mode({1 'mg', 1 'mg', 1.0 'mg', 1.0 'mg', 2 'mg', 2 'mg', 2 'mg'})` da `2 'mg'`; idéntico con la clave de main | **Al backlog (1c5):** es anterior al PR, y arreglarlo aquí ampliaría el review |
| R2-2 (bajo, Codex) | Codex ejecutó cql-execution: D13 da `""`, D326 da null, y `{x, x, y}` da `y`. Una lectura por equivalencia que se quede con el último también daría D13 y D326 | **Se mantiene, con su fundamento:** la clasificación descansa en cómo está hecha la referencia (todo Code es `"[object Object]"`, verificado en su código y con el control), no en las dos filas solas |
| B2 (bajo, Fable) | La decisión 5 decía «con los null iguales entre sí», y Mode ignora los null de la lista | **Corregido:** se refiere a los elementos null de un Code o de una tupla |
| B3 (bajo, Fable) | El punto de reanudación nombraba la ronda 2 como pendiente | **Ya corregido** en el repo; la copia de review era anterior |

