# Adopción del ModelInfo y FHIRHelpers oficiales

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Cerrar la capa FHIR del motor cargando los artefactos que cqframework ya publica —el ModelInfo oficial y el FHIRHelpers oficial— en lugar de mantener versiones propias reducidas, y tomar de `google/cql` las cuatro decisiones de API que le faltan al motor.

**Architecture:** Los bloqueos están verificados empíricamente, no deducidos. La librería oficial **ya compila** en el motor actual; lo que falla es la evaluación, por tres causas concretas y acotadas que ataca la Etapa 1. El ModelInfo no es prerrequisito para cargar FHIRHelpers, solo para despachar correctamente sus 251 sobrecargas de `ToString`.

**Tech Stack:** Go 1.24, `github.com/gofhir/fhirpath/types`, `go:embed`, ModelInfo XML de cqframework (`urn:hl7-org:elm-modelinfo:r1`).

---

## Estado verificado

Medido sobre `main` @ `426e0be`:

- Conformidad 1820/1823 contra `cqframework/cql-tests@727219f`
- 126 builtins, 38 operadores AST, 8 interfaces conectables
- Los criterios de aceptación del issue #3 pasan **7 de 7**
- Fallan cinco expresiones sobre datos FHIR

```text
First([Condition]).code in "T2DM"                 false   ← debería ser true
[Condition] C where C.code in "T2DM" return C.id  {}      ← vacío
Count([Condition])                                2       ← incluye otro paciente
First([Encounter]).period during "MP"             null    ← Period no es Interval
parameter "MP" … default Interval[…]              "MP"    ← default no aplicado
```

### Experimento: cargar el FHIRHelpers oficial hoy

Las 520 líneas oficiales servidas vía `WithLibraryResolver`, sin cambiar nada del motor:

```text
COMPILA-SOLA   OK — 520 líneas, 297 funciones, sin ModelInfo
ToInterval     null                                    ← .value
ToCode         Code = Tuple{code: null, system: null}  ← despacha bien, campos vacíos por .value
ToConcept      ERR: unknown function: ToCode           ← llamada entre funciones de la misma librería
ToQuantity     ERR: unknown function: ToCalendarUnit   ← ídem
ToRatio        ERR: invalid quantity format: Tuple{…}  ← Tuple no se acepta como Quantity
ToString(uri)  null                                    ← .value
```

**Tres conclusiones que corrigen suposiciones previas:**

1. **El ModelInfo no bloquea la carga.** La librería compila sin él porque el motor no tiene comprobación de tipos, así que `define function ToInterval(period FHIR.Period)` parsea ignorando el tipo declarado. El ModelInfo hace falta para despachar las 251 sobrecargas de `ToString`, no para cargar el fichero.
2. **`ToCode` ya se despacha correctamente** y construye un `Code`: el problema son sus campos, vacíos porque `coding.code.value` devuelve null.
3. **Hay un bloqueo no catalogado hasta ahora**: una función de una librería incluida no puede llamar a otra de la misma librería. `ToConcept` invoca `ToCode` y falla con *unknown function*. Esto rompe estructuralmente el fichero oficial, donde las conversiones se apoyan unas en otras.

---

## Recomendación sobre los ficheros oficiales

**Sí a ambos, y ninguno traducido a mano.**

**FHIRHelpers oficial** aporta las cuatro conversiones que faltan —`CodeableConcept→Concept`, `Coding→Code`, `Period→Interval`, `Quantity`— ya escritas en CQL. La arquitectura actual es la correcta: `fhirhelpers.Source` es una constante de texto que el motor incluye. Falta sustituir 8 identidades por el contenido real, y arreglar los tres bloqueos de la Etapa 1.

**ModelInfo oficial** no es prerrequisito para lo anterior, pero sí para el resto: jerarquía de tipos, `primaryCodePath`, contextos, conversiones declaradas y despacho correcto de `ToString`. El tamaño no es obstáculo: los 3,3 MB bajan a **107 KB** quitando `description`, `definition` y `comment` y comprimiendo, embebible con `go:embed`.

**Ninguno debe traducirse a mano.** Mantener una versión propia significa ser dueño de la divergencia para siempre: cada actualización de upstream pasa a ser una fusión manual, y los errores no fallan ruidosamente sino que devuelven listas vacías. Es exactamente cómo se llegó a los 6 recursos de `model/modelinfo.go` y a las 8 identidades de `fhirhelpers/fhirhelpers.go`.

### Lo que contiene el ModelInfo oficial

| Elemento | Cantidad |
| --- | --- |
| `ClassInfo` | 931, todos con `baseType` |
| Recuperables (`retrievable="true"`) | 147 |
| Con `primaryCodePath` | 67 |
| Elementos declarados | 5.000 |
| `conversionInfo` | 264 |
| `contextInfo` | 5, con `keyElement` |
| En la raíz | `patientClassName`, `patientBirthDatePropertyName` |

Versiones publicadas: FHIR 1.0.2 → 4.0.1, QI-Core hasta 6.0.0, US Core 6.1.0. **No hay ModelInfo de R5** en el repositorio de referencia ni en el IG de CQL, cuyo `Library/FHIR-ModelInfo` está fijado en `version: "4.0.1"`. Si hiciera falta, `tools/xsd-to-modelinfo` de cqframework permite generarlo, asumiendo la corrección del resultado.

---

## Etapa 0: Higiene

**Scope:** independiente del resto.
**Coste:** horas.

- [ ] `WithMaxDepth` y `WithMaxRetrieveSize`: cablearlas a `eval.Context` (contador de profundidad en `Eval`, truncado en `evalRetrieve`). Hoy se asignan en `engine.go:46-47,90-99` y no se leen nunca. **Retirarlas no es alternativa**: la firma es pública desde v1.5.1 y el proxy de módulos cachea versiones para siempre.
- [ ] Comprobar cancelación (`GoCtx.Err()`) dentro de `Eval` y en los bucles de `evalQuery`. Hoy no hay ninguna comprobación en `eval/`, así que el timeout no interrumpe: solo se detecta al terminar (`engine.go:281`).
- [ ] Un identificador sin resolver debe ser error, no devolver su propio nombre como `String` (`eval/evaluator.go:318`). Arregla de paso `sort by` por columna y los `default` de parámetros.
- [ ] Fijar la versión de `golangci-lint` en `.github/workflows/ci.yml`; hoy pide `latest`.

**Verificación:** un test que demuestre que una expresión profunda aborta por `maxDepth`, que un bucle largo se corta por timeout y que `define A: Bogus` falla al compilar.

---

## Etapa 0b: Las heurísticas del builder

**Scope:** `compiler/builder.go`.
**Coste:** ~1 día.
**Prioridad:** la más alta del plan en términos de generalidad.

Varias decisiones semánticas se toman con `strings.Contains` sobre el texto plano del nodo. No es cobertura incompleta: es construcción errónea, y **afecta a cualquier CQL que contenga esas subcadenas**, sea del caso de uso que sea. Verificado hoy:

```text
'not' is null                          → true      (debería ser false)
{'union','x'} except {'x'}             → {union,x} (hace unión, no except)
({1,1}) A return Count(distinct {A})   → {1}       (aplica distinct a la query entera)
sort by <columna>                      → no ordena
```

- [ ] Sustituir la detección por inspección de los tokens hijo (`GetChild`) en: negación de `is null` (`builder.go:567`), operadores de conjunto (`:674`), `distinct`/`all` en `return` y `aggregate` (`:1629`, `:1643`), `external` (`:440`), `properly` (`:614`), `expand`/`collapse` (`:1175`) y los bordes del selector de intervalo (`:1382`).
- [ ] `sort by <columna>`: resolver la clave contra el elemento en lugar de dejarla caer al comportamiento de identificador desconocido (`eval/evaluator.go:3356`).

**Verificación:** los cuatro casos de arriba dan el resultado correcto.

**Por qué va antes que todo lo demás:** el resto del plan mejora la capa de datos, que solo se nota en CQL clínico. Esto corrompe expresiones arbitrarias en silencio, y ninguna suite lo detecta porque la de conformidad no usa literales con esas palabras dentro.

### Lo que sí funciona (medido, contra lo que sugerían informes previos)

`from A, B` multi-fuente, `with`/`without ... such that`, `aggregate ... starting`, `let`, `return distinct` y `sort by $this` evalúan correctamente. El motor de queries está en mejor estado de lo que se había reportado; lo que falla es lo de arriba.

---

## Etapa 1: Los tres bloqueos

**Scope:** `eval/evaluator.go`, `fhirhelpers/`.
**Coste:** días.

Los tres salen del experimento de arriba. Ninguno es estructural.

### Task 1.1 — Resolución de funciones dentro de una librería incluida

`ToConcept` llama a `ToCode` y falla con *unknown function*. Las funciones de una librería incluida se registran para llamadas cualificadas desde fuera (`FH.ToCode`), pero no en el ámbito de la propia librería, de modo que sus funciones no se ven entre sí.

Es el bloqueo más importante: sin él, el fichero oficial no funciona en absoluto, porque sus conversiones se apoyan unas en otras.

### Task 1.2 — Accesor identidad para `.value`

En `evalMemberAccess`, si el miembro es `value` y el origen ya es una primitiva de sistema, devolverlo:

```go
// .value sobre una primitiva es la primitiva: el ModelInfo oficial modela
// FHIR.date como un objeto con un elemento value, mientras el evaluador navega
// JSON donde birthDate ya es el escalar. Esta regla hace que ambos coincidan.
if name == "value" && isPrimitive(src) {
    return src, nil
}
```

**La regla debe ser estrecha.** Si `.value` sobre cualquier valor devuelve el valor, un `someString.value` escrito por error deja de fallar y devuelve la cadena, convirtiendo una errata en un silencio. Limitarla a los tipos primitivos de sistema y a ningún otro caso.

### Task 1.3 — Materializar `Code` y `Concept`, aceptar Tuple como Quantity

`Code{…}` y `Concept{…}` se construyen hoy como `cqltypes.Tuple` con `TypeOverride`, de modo que `extractCodeComponents` (`eval/terminology.go:149`) no los reconoce y `in "ValueSet"` responde `false`. `Quantity{…}` sí se materializa: replicar ese camino.

```go
case "Code":    return buildCode(elements)
case "Concept": return buildConcept(elements)
```

Y aceptar un Tuple con `value`/`unit` donde se espera un Quantity, que es lo que hace fallar a `ToRatio`.

### Task 1.4 — Sustituir el FHIRHelpers propio por el oficial y medir

Reemplazar `fhirhelpers.Source` por las 520 líneas de
`quick/src/main/resources/org/hl7/fhir/FHIRHelpers-4.0.1.cql`.

**La métrica no es «cuántas de las 297».** 251 son sobrecargas de `ToString` que dependen del despacho por tipo (Etapa 2). El criterio es el grupo de conversiones que arregla los cinco fallos:

| Función | Criterio |
| --- | --- |
| `ToInterval(Period)` | devuelve `Interval<DateTime>` con los bordes correctos |
| `ToInterval(Range)` | devuelve `Interval<Quantity>` |
| `ToCode(Coding)` | devuelve `Code` con `code`, `system`, `display`, `version` poblados |
| `ToConcept(CodeableConcept)` | devuelve `Concept` con sus `Code` |
| `ToQuantity(Quantity)` | devuelve `Quantity` con valor y unidad |
| `ToRatio(Ratio)` | devuelve `Ratio` |

**Umbral, fijado de antemano:** si las seis funcionan, las Etapas 2 y 3 son mecánicas y el plan sigue. Si fallan tres o más por razones que no sean estas tres tareas, hay que evaluar la envoltura completa de primitivas (*Option B* del issue #3), que es cara y arrastra a la Etapa 5.

**Verificación adicional:** ejecutar los tests de `gofhir/server` contra la rama vía `replace`. Hoy pasan sus 23 paquetes; el FHIRHelpers oficial cambia el camino de `FHIRHelpers.ToQuantity(o.valueQuantity).value`, así que hay que confirmar que sigue en verde.

---

## Etapa 2: ModelInfo oficial y despacho por tipo

**Scope:** `model/`, `eval/`.
**Coste:** 1–2 semanas. Depende de la Etapa 1.

- [ ] Parser del XML (`urn:hl7-org:elm-modelinfo:r1`): `typeInfo`, `element`, `elementTypeSpecifier`, `conversionInfo`, `contextInfo`.
- [ ] Embeber recortado y comprimido con `go:embed` (107 KB). Exponerlo como datos versionados, al estilo de `FHIRDataModel(version) ([]byte, error)` de `google/cql`.
- [ ] **Despacho de sobrecargas por tipo de argumento.** Es la pieza que las 251 sobrecargas de `ToString` necesitan, y un subconjunto de la Etapa 5 adelantado por necesidad. Hoy `matchesArgType` (`eval/evaluator.go:1396-1464`) solo reconoce literales: cualquier argumento no literal puntúa cero y se elige el primer candidato.
- [ ] Cablear `baseType` → jerarquía real para `is` y `as`. Hoy `funcs/type_ops.go:154` compara nombres con `strings.EqualFold`, así que `x is DomainResource` falla para un Patient.
- [ ] Cablear `primaryCodePath` en `evalRetrieve`: 67 declarados, ninguno consultado. Sin esto `[Encounter: "VS"]` sale con `codePath=""`.
- [ ] Cablear los 5 `contextInfo` con su `keyElement` → filtrado por paciente. Requiere añadir el sujeto a la firma de `DataProvider.Retrieve` (`eval/context.go:16`), que rompe la API pública; la ventana barata es ahora, mientras el único implementador conocido sea propio.
- [ ] Cargar el modelinfo según el `using` y fallar si no hay ninguno para esa versión. Hoy `lib.Usings` se parsea y no se lee, así que `using FHIR version '5.0.0'` se acepta en silencio contra R4.

**Verificación:** `Count([Condition])` deja de incluir recursos de otros pacientes; `[Encounter: "VS"]` llega al proveedor con `codePath="type"`; `ToString` elige la sobrecarga correcta entre 251.

---

## Etapa 3: Coerción por `conversionInfo`

**Scope:** `eval/`.
**Coste:** 1–2 semanas. Depende de las Etapas 1 y 2.

El ModelInfo declara 264 conversiones con su función asociada (`FHIR.CodeableConcept → System.Concept` vía `FHIRHelpers.ToConcept`). En la implementación de referencia el traductor las inserta usando tipos estáticos; sin fase semántica se aplican **en tiempo de evaluación**: cuando un operador recibe un tipo FHIR donde espera uno de sistema, se busca la conversión declarada y se invoca.

Tiene techo: sin tipos estáticos no se puede distinguir una sobrecarga por su tipo de retorno ni decidir entre dos conversiones aplicables. Cubre la mayoría de casos reales y la Etapa 5 la sustituye; cuando llegue, la tabla ya está cargada y solo cambia *cuándo* se decide aplicarla.

**Verificación:** los cinco fallos de arriba pasan a verde.

> **Nota sobre el criterio.** Esos cinco casos salen de un fixture construido para este análisis. Antes de darlos por criterio de aceptación conviene sustituirlos por una measure publicada de verdad, evaluada contra datos que no se hayan elegido para que funcionen.

---

## Etapa 4: Decisiones de API tomadas de `google/cql`

**Scope:** `engine.go`, `eval/`.
**Coste:** 1–2 semanas. Independiente de las anteriores; puede adelantarse.

- [ ] **Reloj inyectable** (`EvaluationTimestamp`). Sin él, `Now()` y `Today()` toman la hora del sistema y una measure deja de ser reproducible: los mismos datos dan resultados distintos según cuándo se ejecute.
- [ ] **`Parse` y `Eval` separados**, con artefacto compilado reutilizable. Hoy `Compile()` solo devuelve `error` y la caché interna indexa por `fnv64a` sin verificar la fuente (`engine.go:169`), de modo que una colisión devuelve el AST equivocado.
- [ ] **Respetar `define private`**, que se parsea y nunca se comprueba.
- [ ] **Procedencia del resultado**: cada valor recuerda la expresión que lo produjo y sus entradas. Es lo que permite justificar por qué un paciente entró en una población.

---

## Etapa 5: Fase semántica

**Scope:** nuevo paquete entre `compiler` y `eval`.
**Coste:** meses. Depende de todo lo anterior. Reducida: el despacho por tipo se adelantó a la Etapa 2.

- [x] Posiciones `{Line, Col}` en cada nodo del AST desde `ctx.GetStart()`. Mecánico, y da diagnósticos útiles desde el primer día. Hoy solo los errores de sintaxis de ANTLR llevan ubicación.
- [x] Tipado estático con recuperación estilo `badExpression()`: reportar todos los errores de una pasada en vez de abortar en el primero. **Paquete `sema/`**, expuesto como `Engine.Check`. La recuperación es el tipo `Unknown`: todo operador lo acepta y lo propaga, así que un nombre sin resolver se reporta una vez y no una por expresión que lo mencione.
- [x] Coste de conversión en la resolución de sobrecargas, como `OverloadMatch()`. Escala en `sema/convert.go`: exacta < subtipo < rama de choice < conversión implícita < conversión declarada por el modelo < promoción a lista.
- [~] Inserción estática de las conversiones que la Etapa 3 aplica en runtime. La fase ya **decide** cuáles hacen falta y dónde (`Result.Conversions`, y agrupadas por definición), y coincide con la referencia en la probe. Falta que el evaluador consuma esa decisión en vez de volver a tomarla con el valor en la mano.

**Verificación:** el traductor de referencia está disponible como microservicio (`cqframework/cql-translation-service`) y como CLI (`cql-to-elm-cli`). Traducir un corpus y comparar qué tipo infiere y dónde inserta cada conversión convierte esta etapa en algo diffable.

### Lo medido al cerrar la primera mitad

| | |
| --- | --- |
| Probe de referencia | los 5 tipos coinciden y las 2 conversiones caen en la misma definición (`TestStatic*MatchTheReference`) |
| FHIRHelpers oficial, 297 funciones | 0 diagnósticos |
| Corpus de conformidad, 1783 expresiones válidas | 0 falsos positivos |
| Las 39 que el corpus marca inválidas y el parser acepta | 0 detectadas, y es lo correcto: son errores de evaluación —`Exp(1000)` desborda, `Ln(0)` no existe, `successor of` el último DateTime no tiene a dónde ir—, no de tipos |

**`start of encounter.period` queda cerrado estáticamente**: la fase dice `DateTime`, como la referencia, porque el modelo declara que `FHIR.Period` se convierte a `Interval<System.DateTime>`. En evaluación sigue dando `Date`, y seguirá hasta que el evaluador consuma la decisión estática.

Tres reglas costaron encontrarse, y las tres salieron de correr la fase sobre el FHIRHelpers oficial en vez de sobre ejemplos propios:

- **`null` en una rama no destipa la expresión.** Casi toda función de FHIRHelpers es `if x is null then null else …`; tratar el tipo de `null` como un tipo más hacía que todas devolvieran `Any`, y con ello fallaban las llamadas que las usan.
- **`System.ValueSet` tiene elementos** (`id`, `version`, `codesystems`), y `ToValueSet` los construye.
- **Un nombre de tipo sin cualificar es del modelo antes que de CQL.** `value as Quantity` dentro de FHIRHelpers es `FHIR.Quantity`; el mismo fichero escribe `System.Quantity` cuando quiere el otro. Solo chocan `Quantity` y `Ratio`, porque FHIR escribe sus primitivas en minúscula — razón de más para preguntárselo al modelo y no a una lista de nombres.

---

## Riesgos

**El puente puede no bastar.** Que `.value` sea identidad resuelve el acceso, pero quedan las primitivas con extensión y sin valor, y `.value` sobre un objeto real. Por eso la Etapa 1 tiene umbral fijado de antemano.

**La Etapa 1 arrastra un trozo de la 5.** El despacho por tipo era trabajo de la fase semántica y hay que adelantarlo para que las 251 sobrecargas de `ToString` funcionen. No es opcional ni se puede aplazar.

**Adoptar el FHIRHelpers oficial cambia comportamiento.** `FHIRHelpers.ToQuantity(o.valueQuantity).value` funciona hoy con el stub; con el oficial toma otro camino. Hay que verificar contra `gofhir/server`, y la publicación debe ser minor, no parche.

**El criterio de aceptación es propio.** Los cinco fallos vienen de un fixture construido para este análisis. Sustituirlos por una measure publicada antes de tratarlos como prueba.

**El orden importa más que las estimaciones.** Los plazos son gruesos y suponen trabajo intermitente: sirven para ordenar, no para comprometer fechas.

---

## Referencias

- Issue #3 de este repositorio — decisión *Option A* / *Option B* sobre envoltura de primitivas
- `fhir-modelinfo-4.0.1.xml` — `quick/src/main/resources/org/hl7/fhir/` en `cqframework/clinical_quality_language`
- `FHIRHelpers-4.0.1.cql` — mismo directorio, 520 líneas, 297 funciones, 251 de ellas `ToString`
- `google/cql` — `docs/implementation.md`; API pública en `pkg.go.dev/github.com/google/cql`
- `cqframework/cql-translation-service` — traductor de referencia como microservicio
