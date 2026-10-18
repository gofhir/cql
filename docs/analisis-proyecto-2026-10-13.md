# Evaluación de go-cql y estrategia de pruebas diferenciales

Fecha: 13 de octubre de 2026. Código revisado: `24790151c541a25422b3e9d39d94ef152df8f0b5`. Se conserva el cuerpo de la primera entrega; la sección 10 incorpora el cierre de las dos revisiones adversariales posteriores.

## Dictamen

El proyecto tiene una base técnica sólida para un motor CQL nativo de Go: separación entre parsing, análisis semántico y evaluación; conversiones planificadas; modelo FHIR R4 oficial embebido; proveedores reemplazables; y abundantes regresiones sobre errores reales. Sin embargo, la evidencia actual no permite afirmar conformidad completa, equivalencia clínica con motores externos ni superioridad de rendimiento. Encontré errores de resultados que sobreviven a toda la suite existente.

La inversión inmediata de mayor retorno es ampliar la verificación diferencial de **valores**, bibliotecas y datos FHIR. Optimizar después los caminos que esa verificación cubra, especialmente queries y deduplicación. No recomiendo una reescritura de arquitectura antes de establecer esa red de seguridad.

## 1. Alcance y evidencia

Leí la API pública, caché, compilador, partes críticas del evaluador, contextos, análisis semántico, importador ELM, tests de referencia, conformidad y eCQM, benchmarks y CI. Es una revisión dirigida por riesgo, no una demostración formal ni inspección exhaustiva de cada operador.

Verificaciones realizadas:

- `go test ./... -coverprofile=/tmp/go-cql-review-coverage.out`: todos los paquetes pasan.
- Go `1.25.9`, macOS ARM64, Apple M4 Pro. CI configura Go `1.24` sobre Linux; estas mediciones no sustituyen ese entorno.
- `go test -run '^$' -bench BenchmarkEvaluate -benchmem -benchtime=1s -count=3 .`: tres muestras por escenario.
- Sondas adicionales independientes, guardadas inicialmente en `/tmp/go-cql-review-probes.go`, reprodujeron los resultados de la sección 3.
- Consulta de documentación primaria de CQFramework y Google para seleccionar comparadores.
- La ejecución inicial restringida falló por acceso a la caché; la suite normal se verificó con acceso autorizado. Detector de carreras pendiente en esta primera entrega.

La cobertura por paquete sin `-coverpkg` NO mide todo lo ejercitado por tests de integración de otros paquetes: el 30,2% de `eval` y el 32,1% de `funcs` no permiten concluir que el resto nunca se ejecuta. El total bruto 20,9% además incluye código ANTLR generado. Se necesita medir cobertura transversal y separar código generado antes de usarla como KPI.

Los tests `TestSemanticPhaseAcceptsPublishedMeasures` y `TestECQMCases` requieren `ECQM_CONTENT_DIR`; no se ejecutaron sobre el corpus externo en esta revisión. La CI visible tampoco prepara ese checkout. Existen regresiones locales de formas de medidas, que sí forman parte de la suite normal.

## 2. Arquitectura: ventajas y compromisos

Flujo principal: CQL → ANTLR → AST → `sema.Result` → evaluador → valores basados en FHIRPath y tipos CQL propios. ELM tiene un camino separado de importación/exportación. La API `Library` conserva AST y plan para repetición.

| Decisión | Ventaja | Costo o límite |
|---|---|---|
| Parser ANTLR | Gramática explícita, posiciones y ecosistema conocido | Parsing costoso y gran cantidad de código generado; no implica por sí solo cobertura completa del lenguaje |
| Análisis semántico antes de evaluar | Diagnósticos sin pacientes; evita recalcular conversiones en cada operador | Reglas de tipos y ejecución deben mantenerse alineadas; algunas llamadas entre bibliotecas no quedan comprobadas integralmente |
| Plan ligado a nodos AST | Conversiones y estrechamiento de choices en el punto correcto | Una transformación posterior del AST requiere reconstruir el plan; hay que documentar inmutabilidad |
| Contexto por evaluación | Buen punto de partida para concurrencia y aislamiento | Proveedores, modelos y listeners externos también deben ser seguros; `-race` no prueba todos los contratos |
| Caché LRU acotada y verificación del texto | Evita crecimiento ilimitado por cantidad de fuentes y errores por colisión FNV | Límite por entradas, no por bytes; mutex en hits y compilación duplicada ante misses concurrentes |
| `Parse` y evaluación de `Library` | Permite amortizar parsing deliberadamente | Una Library creada por un Engine puede llevar un plan del modelo de ese Engine al usarse con otro; contrato por verificar |
| Modelo oficial FHIR R4 embebido | Mayor cobertura de tipos y conversiones sin acceso remoto | Fallback silencioso a modelo simplificado puede ocultar degradación; conviene exponer estado del modelo |
| Dependencia de FHIRPath | Reutiliza tipos, acceso JSON y operaciones | CQL tiene reglas diferentes: wrappers y correcciones temporales/cantidades aumentan el trabajo de mantenimiento |
| Pushdown temporal conservando `where` | Permite reducir I/O y conserva comprobación local | Sólo cubre ciertas queries; un proveedor que elimina candidatos válidos no puede corregirse después |

El proyecto interpreta AST; “compilado” no significa bytecode, JIT ni ejecución vectorizada. Esto mantiene el diseño accesible y facilita diagnósticos, pero deja despacho dinámico, interfaces y asignaciones en los caminos calientes.

`eval/evaluator.go` tiene 7.563 líneas y `compiler/builder.go` 2.007. Las exclusiones de complejidad del linter abarcan amplios paquetes. La longitud no es un defecto por sí sola: el riesgo concreto es que filtros, agregación, deduplicación y ordenación implementen políticas distintas, como ya sucede. Dividir por responsabilidades conservando las reglas compartidas es preferible a fragmentar mecánicamente los switches.

## 3. Hallazgos reproducidos

### P1 — `aggregate` pierde los filtros `with` y `without`

En `eval/evaluator.go`, `evalQuery` aplica relaciones en un primer recorrido, pero la rama de agregación vuelve a recorrer `combos` originales y sólo repite `let` y `where`. No conserva la selección producida por las relaciones.

```cql
define R:
  ({1,2,3}) X
    with ({2}) Y such that X = Y
    aggregate A starting 0: A + X
```

Resultado observado: **6**. Resultado esperado por la relación: **2**. Cambiando `with` por `without`, se observa **6** cuando deberían sumarse 1 y 3, dando **4**. El control `where X = 2 aggregate ...` devuelve **2**.

Impacto: agregar recursos con una condición de existencia o ausencia puede incluir filas excluidas y alterar un numerador, denominador o total. No es necesario un caso voluminoso ni datos inválidos.

Corrección propuesta: un único flujo de filas que aplique fuentes, bindings y todas las relaciones/filtros, consumido después por retorno o reducción. Añadir regresiones combinatorias y comprobar los casos en Java antes de consolidar el cambio.

### P1 — deduplicación de agregación mediante `String()`

`distinctCombos` concatena la representación textual de cada valor con comas, y usa `nil,` para nulos. La representación no incluye un encoding inequívoco de tipo y límites.

```cql
define R:
  ({'nil', null}) X
    aggregate distinct A starting 0: A + 1
```

Resultado observado: **1**. Son dos valores diferentes y deberían producir **2**. Esto demuestra una colisión real, sin depender de una hipótesis sobre hashes. La sonda `{1, 1.0}` produce 2; su interpretación necesita verificar promoción y representación semántica antes de declararla otro defecto.

Corrección propuesta: igualdad CQL explícita para filas; si se incorpora hashing, verificar igualdad dentro de buckets y mantener separaciones de tipo/estructura. No reemplazarlo simplemente por otro `String()`.

### P1 — ELM desconocido se acepta como nulo

El `default` de `elm.ImportExpression` crea un literal null. `elm.Import` sólo rechaza la biblioteca nil y no devuelve un diagnóstico para ese nodo.

Sonda: `elm.ImportExpression(&elm.ExpressionNode{Type:"DefinitelyUnsupportedOperator"})` produce `*ast.Literal` de tipo null; evaluarlo retorna `nil` sin error.

Impacto: un ELM que usa una operación no implementada puede transformarse en un resultado plausible de datos ausentes. El hallazgo corresponde al API ELM de bajo nivel; la API Engine de texto no lo usa como camino normal.

Corrección propuesta: importación estricta con error localizado y lista explícita de operadores soportados. Hay que considerar compatibilidad del API público `ImportExpression`, que actualmente no retorna error.

### P2 — contexto ya cancelado puede devolver éxito

`EvaluateExpression` con un contexto cancelado y `define R: 42` devuelve **42**, sin error. `Eval` consulta cancelación cada 256 nodos, y no hay comprobación inicial en ese camino.

Es una inconsistencia del contrato de cancelación; no demuestra una ejecución infinita. El timeout es cooperativo: no puede interrumpir un proveedor que ignora el contexto, y el parsing de la biblioteca principal sucede antes de `withTimeout`. Conviene comprobar el contexto al entrar y delimitar claramente si el timeout cubre compilación o sólo evaluación.

### Pendiente de contraste normativo — duplicados en operaciones de conjuntos

`({1,1,2}) union ({2})` devuelve `{1,1,2}`; `({1,1,2}) except ({3})` devuelve `{1,1,2}`. El primer caso copia íntegro el operando izquierdo. Antes de elevar ambos a defectos hay que fijar la versión CQL objetivo y contrastar cada operador con especificación y motor externo; `except` no debe suponerse equivalente a `union` en su manejo de multiplicidad.

## 4. Calidad de la verificación actual

Fortalezas:

- Suite de expresiones oficial vendorizada y muchas regresiones sobre FHIR, choices, temporalidad, UCUM, nulls y resolución de nombres.
- CI con `-race`, linter fijado, `go mod verify` y tres zonas adicionales a UTC, incluyendo offset fraccionario y extremos.
- Reloj inyectable por Engine y por llamada; facilita reproducir reglas temporales.
- Casos de límites de profundidad, retrieve y cancelación; no trunca silenciosamente resultados grandes.
- El harness externo ya fija traductor Java `3.26.0` y registra tipos, locators y conversiones.
- Hay guardas contra comparaciones vacías y contra excepciones que empiezan a pasar.

Brechas:

1. El fixture Java cubre una biblioteca probe; no contiene resultados de ejecución Java con los mismos pacientes.
2. `TestResultTypesAgainstTheReference` informa diferencias pero no falla. Es razonable para tipos estáticos frente a runtime, pero no constituye equivalencia de resultados.
3. En `engine_reference_test.go`, los fixtures ausentes producen skip; una CI de referencia debe fallar si falta un artefacto obligatorio.
4. La guardia del 75% no basta si `Definitions` está vacío: el mínimo calculado es cero. Debe existir un manifest de casos esperado.
5. En conformidad, `invalid` acepta cualquier error; un error de infraestructura o una operación no soportada puede satisfacer un caso que debía fallar por otra razón.
6. Las tres excepciones oficiales se consideran satisfactorias mientras “no pasen”; un fallo distinto al documentado también mantiene verde el test. Es mejor exigir la discrepancia exacta, su fase y su resultado.
7. No encontré funciones `Fuzz` ni gates de rendimiento en CI; hay dos benchmarks, ambos de una misma biblioteca y 20 recursos por tipo.
8. El README afirma 1820/1823 y también “100% conformance”; ambas afirmaciones no expresan el mismo alcance. Pasar estos tests no certifica CQL/FHIR completo.
9. El ejemplo Quick Start usa `time.Second` sin importar `time`; compilar ejemplos evitaría este tipo de deriva.

## 5. Rendimiento medido y límites de escalabilidad

| Escenario existente | Tiempo, 3 muestras | Bytes asignados/op | Asignaciones/op |
|---|---:|---:|---:|
| Biblioteca con caché caliente | 261,8–269,2 µs | 230.677–231.428 | 2.172 |
| Engine nuevo y biblioteca fría | 16,69–17,12 ms | 18.146.647–18.154.136 | 217.502–217.511 |

Frío cuesta aproximadamente 62–65 veces el escenario caliente en esta biblioteca. La comparación incluye construcción del Engine, parsing/check e includes; no mide sólo el parser. B/op son bytes asignados acumulados por operación, no heap retenido ni pico RSS. El proveedor del benchmark construye JSON en cada llamada, de modo que parte del costo medido es del fixture.

Decisiones favorables: reutilización de AST y plan; caché LRU; contexto hijo reutilizado por query ordinaria; registros de funciones compartidos dentro de una evaluación; JSON crudo para recursos; pushdown temporal; comprobación de cancelación amortizada.

Riesgos estructurales:

- `evalQuery` materializa el producto cartesiano completo como mapas antes de filtrar. Dos fuentes de 10.000 recursos permiten hasta **100 millones de combinaciones**, aunque cada retrieve respete su límite. Es una deducción del algoritmo, no una carga ejecutada durante esta revisión.
- No observé presupuesto global de filas intermedias, bytes o pasos. Un timeout de 30s no limita memoria ni impide picos antes de cancelarse.
- `nullSafeDistinct`, intersección y búsqueda de equivalentes recorren listas repetidamente: casos cuadráticos.
- Ordenación por expresión calcula claves durante comparaciones; precomputarlas puede reducir trabajo, siempre conservando errores y reglas de null.
- Los misses concurrentes compilan fuera del lock; varias solicitudes de la misma fuente pueden duplicar parsing. Medir antes de introducir `singleflight`.
- El LRU limita entradas, no el tamaño acumulado. La estimación de 22 MB de los comentarios usa FHIRHelpers como ejemplo; no es un techo contractual de memoria.

Prioridad: filas en streaming y presupuesto global; deduplicación correcta; claves de sort precalculadas; luego asignaciones y sincronización según perfiles. Bytecode, pooling amplio o paralelismo dentro de un paciente requieren evidencia antes de aumentar complejidad.

## 6. Con qué comparar

| Referencia | Qué comparar | Limitación |
|---|---|---|
| CQFramework `clinical_quality_language`: traductor y engine Java | Tipos, aceptación/rechazo, conversiones y resultados; referencia principal | Traductor y runtime son componentes distintos; fijar ambos y sus dependencias |
| CQFramework `cql-execution` + `cql-exec-fhir` | Segundo runtime sobre ELM oficial y pacientes FHIR | Comparte traductor con Java; acuerdo entre ambos no valida de forma independiente el frontend |
| `google/cql` | Alternativa Go y escalabilidad en el subconjunto común | Se declara experimental; documenta huecos en operadores, promoción, incertidumbre y ELM |
| `cqframework/cql-tests` + corpus eCQM con oráculos | Expectativas independientes de nuestra implementación | Una suite finita no representa todas las combinaciones ni garantiza cobertura clínica |

No usar `github.com/gofhir/fhirpath` como árbitro independiente de todo el motor: ya es dependencia y FHIRPath no es CQL. Sí sirve para pruebas contractuales del adaptador.

Fuentes primarias consultadas: [CQFramework](https://github.com/cqframework/clinical_quality_language), [runtime JS](https://github.com/cqframework/cql-execution), [adaptador FHIR](https://github.com/cqframework/cql-exec-fhir), [Google CQL](https://github.com/google/cql), [suite oficial](https://github.com/cqframework/cql-tests). Las capacidades deben fijarse por versión/commit al implementar; no recomiendo usar `latest` como baseline.

## 7. Diseño concreto de tests automáticos

### Un caso, varias rutas

Cada caso debe contener CQL, definición, parámetros tipados, bundle FHIR, dependencias, terminología local, timestamp y offset explícitos, versión CQL/ELM/modelinfo y manifest de capacidades requeridas.

Rutas a ejecutar:

1. CQL original → go-cql → resultado.
2. CQL original → traductor oficial → ELM → engine Java → resultado.
3. Ese mismo ELM → runtime JS + adaptador FHIR → resultado.
4. Opcional: ELM oficial → importador go-cql → evaluador. Activar sólo cuando rechace estrictamente nodos desconocidos y maneje funciones/dependencias necesarias.
5. Subconjunto declarado → Google CQL, principalmente para un segundo frontend y rendimiento Go.

Esto permite distinguir errores de frontend, ejecución, importación y proveedor. No generar el ELM de referencia con nuestro propio exportador: se compartirían errores de origen.

### Protocolo y comparación

Recomiendo procesos persistentes por motor, con entrada/salida JSON Lines y adaptadores fijados en contenedores. Separar stdout de resultados y stderr de logs. Añadir timeout de proceso y límite de memoria para fuzzing, además del timeout lógico del motor.

Formato de salida orientativo:

```json
{"case":"aggregate-with-001","status":"value","value":{"kind":"Integer","value":"2"}}
```

Estados distintos: `value`, `error`, `unsupported`, `infra_error`. `null` es un valor, no un error ni una ausencia de respuesta. Guardar fase del error (parse/check/eval), código estable y localización; comparar mensajes textuales sólo cuando sean parte del contrato.

El normalizador es parte crítica del sistema:

- Decimal como cadena exacta con las reglas de escala del perfil; no pasar por float64.
- Date/DateTime/Time conservando precisión, offset explícito y offset asumido de evaluación.
- Intervalos con extremos, apertura/cierre y distinción entre límite abierto y límite desconocido cuando el tipo lo requiera.
- Listas preservando orden, duplicados y null. Comparación como multiconjunto únicamente si el contrato específico lo exige; no ordenar todas las listas para ocultar diferencias.
- Tuplas por campos canónicos; Code con sistema, versión y código; cantidades con unidad y política UCUM explícita.
- No usar `String()`, `Value.Equal` de go-cql ni una conversión implementada por el motor probado como único oráculo.

Probar el normalizador con pares deliberadamente distintos y conocidos iguales. Por ejemplo `'nil'` frente a null, `false` frente a null, lista vacía frente a null, límites abiertos/cerrados y fechas a distinta precisión. Un normalizador defectuoso puede hacer concordar dos resultados distintos.

### Generación dirigida por tipos

Primero sembrar el corpus con los hallazgos de esta revisión, suite oficial y medidas publicadas. Después generar ASTs pequeños y bien tipados, imprimir CQL y evaluarlo en los motores. Generar también programas inválidos de forma deliberada; separarlos de los válidos.

Dimensiones prioritarias:

| Área | Combinaciones que faltan o merecen presión |
|---|---|
| Query | fuentes × `let` × `with`/`without` × `where` × `return all/distinct` × `aggregate` × `sort` |
| Null y colecciones | null tipado, lista vacía, null interior, duplicados y listas anidadas |
| Temporal | precisión parcial, offset explícito/implícito, límites del periodo, día bisiesto y transiciones DST |
| FHIR | campo ausente, primitivo, choice en otra rama, backbone, repetición 0/1/N |
| Cantidades | unidades equivalentes, incompatibles, affine, calendario frente a duración y límites numéricos |
| Bibliotecas | includes transitivos, alias, versiones, ciclos, overloads y acceso privado |
| Concurrencia | misma Library/pacientes distintos, misses simultáneos, aislamiento de parámetros y trazas |

Usar `testing.F` para imprimir/reducir casos Go y un generador determinista para campañas entre procesos. Guardar seed y versión del generador. El shrinker debe mantener tipos y capacidades mientras reduce CQL, recursos y terminología. Cada discrepancia confirmada entra como regresión mínima vendorizada.

### Tests metamórficos

Ejemplos con precondiciones explícitas: alpha-renaming de aliases; envoltura en una definición equivalente; equivalencia entre resultado con caché activada/desactivada; repetir evaluación con reloj y entradas fijadas; provider sin pushdown frente a provider conservador; orden de entrada cuando el resultado tiene semántica de conjunto.

No aplicar identidades algebraicas universales sobre null, overflow, cantidades o tiempo. La precisión parcial y las conversiones hacen que muchas identidades intuitivas sean falsas.

### Gates que eviten falsos verdes

- Manifest fijo de casos esperados, mínimo absoluto y conteos de ejecutados/comparados/omitidos/errores de infraestructura.
- `unsupported` fuera del denominador de igualdad, pero dentro del reporte de capacidad. Nunca contarlo como conformidad.
- No aceptar “ambos fallaron” sin comparar fase y clase. Tampoco mayoría de motores como prueba de verdad.
- Excepción con input exacto, resultado observado exacto, razón, versión afectada, responsable y criterio de retiro. Un crash nuevo debe fallar aunque el caso ya tuviera una excepción.
- Registrar SHA de inputs, CQL/ELM/modelinfo/FHIRHelpers/terminología y digest de imagen.
- Comparar por paciente y definición; para medidas, comprobar pertenencia y conteos de poblaciones, no sólo el escalar final o que no haya errores.

## 8. Benchmarks comparables y automatización

Separar tres productos de medición: compilación; evaluación con biblioteca preparada; costo total desde CQL. JS y Java evalúan ELM, por lo que medir su runtime contra nuestro parse+eval sería una comparación asimétrica. La preparación del modelo también debe estar claramente incluida o excluida.

Matriz propuesta: 10/100/1.000/10.000 recursos, selectividad de filtro 0/1/10/100%, joins con cardinalidades controladas, distinct con alta/baja duplicación, sort, terminología y varias bibliotecas. Preconstruir datasets fuera del loop y añadir otro escenario de provider/serialización si ese costo interesa.

Medir ns/op, asignaciones y B/op para Go; throughput, p50/p95/p99, CPU, GC, pico RSS y memoria retenida para todos. Calentar JVM/Node, mantener procesos vivos, fijar CPU/memoria, repetir en orden intercalado y comprobar igualdad de resultado antes de aceptar una muestra. Para comparaciones Go entre commits, usar `benchstat` con múltiples muestras; un runner compartido no justifica un gate estricto de pocos puntos porcentuales.

CI propuesta:

- PR: suite actual, regresiones mínimas y fixtures de referencia; faltantes obligatorios fallan.
- Nightly: traductor/runtime Java + JS, corpus fijado, generación con semillas conocidas y límites de recursos. Publicar divergencias y casos minimizados.
- Semanal/manual: campañas largas de fuzz y benchmark de escalabilidad en runner estable; Google en subconjunto compatible.
- Actualización de oráculos: PR separado con diff revisable, versiones y conteos. No regenerar expectativas automáticamente al mismo tiempo que cambia la implementación.

## 9. Orden de ejecución recomendado

1. Corregir agregación/relaciones, deduplicación y rechazo ELM desconocido, empezando por regresiones mínimas. Resolver contrato de cancelación.
2. Hacer obligatorio un corpus externo fijado en un job dedicado; endurecer manifests, fases de error y excepciones existentes.
3. Extender `internal/referenceharness` con ejecución Java y resultados tipados; reutilizarlo en vez de crear un framework paralelo sin necesidad.
4. Añadir adaptador JS, normalizador independiente y matriz de capacidades. Incorporar generación tipada y shrinker después de estabilizar casos manuales.
5. Perfilar queries grandes; reemplazar producto materializado por iteración y aplicar presupuesto global. Medir beneficio con resultados diferenciales iguales.
6. Incorporar comparador Google y seguimiento estadístico de rendimiento cuando el subconjunto y la metodología estén estabilizados.

Criterio de éxito: resultados y errores trazables, ninguna divergencia nueva sin explicación revisada, ningún caso obligatorio silenciosamente omitido y curvas de tiempo/memoria conocidas. El objetivo no es alcanzar un porcentaje atractivo aumentando excepciones.

## 10. Revisiones adversariales

Completadas después de la primera entrega:

1. [Corrección, conformidad y oráculos](analisis-adversarial-1-correccion-2026-10-13.md).
2. [Arquitectura, performance y calidad](analisis-adversarial-2-performance-2026-10-13.md).

Son dos pasadas adversariales del mismo revisor, no auditorías independientes.

Actualizaciones que prevalecen sobre los pendientes de la primera entrega:

- **`go test -race ./...` pasa.** La primera ejecución restringida falló por el entorno; la ejecución autorizada terminó correctamente.
- La medición con `-coverpkg=./...` cubre **80,6% del código no generado**, incluyendo utilidades internas y harness. `eval` alcanza 80,9%. No interpretar las cifras iniciales por paquete como cobertura integral.
- El traductor Java 3.26.0 acepta los cinco contraejemplos del fixture añadido. Se verificó traducción, no ejecución Java.
- Las reglas documentadas de conjuntos en HL7 1.5.0 confirman que los duplicados observados de `union` y `except` son defectos, además de las colisiones de `aggregate distinct`.
- Un include con una rama semánticamente inválida no ejecutada devuelve éxito con validación activada. Se confirma una brecha de validación del grafo, sin afirmar que esa rama produzca un valor erróneo.
- `elm.UnmarshalLibrary` recibe el documento completo del traductor oficial y devuelve una biblioteca vacía sin error; la integración debe manejar el envelope y validar la estructura.
- El perfil de asignaciones respalda investigar `ChildScope`, `evalQuery`, registros de funciones y parsing temporal. No se ha probado un cuello de botella de CPU ni superioridad frente a otro motor.

La recomendación final mantiene el diseño por fases: corregir resultados y contratos, comparar valores con Java, incorporar JS después y optimizar con perfiles y curvas de escalabilidad. [Evidencia reproducible](analisis-proyecto-2026-10-13-evidencia/README.md).
