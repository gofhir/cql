# Revisión adversarial 2: arquitectura, performance y calidad

Realizada después de la primera entrega y de la revisión de corrección. Segunda perspectiva del mismo revisor, no evaluación independiente.

## Hipótesis a atacar

«La implementación es pesada, su cobertura es débil y su arquitectura necesita una reescritura para competir».

Tampoco está demostrada. Hay compromisos de diseño razonables, cobertura transversal sustancial y optimizaciones útiles. Los límites de escalabilidad identificados no autorizan a predecir que otro motor será más rápido en una carga particular.

## A. ¿La primera medición de cobertura justificaba desconfianza?

No. Ejecuté una segunda medición:

```sh
go test ./... -coverpkg=./... -coverprofile=/tmp/go-cql-review-cross-coverage.out
```

Al agregar bloques cubiertos por todos los paquetes y excluir `parser/grammar`:

| Alcance | Cobertura transversal de statements |
|---|---:|
| Código no generado, incluyendo utilidades internas y harness | **80,6%** |
| `eval` | **80,9%** |
| `funcs` | **79,8%** |
| `compiler` | **84,6%** |
| `sema` | **82,6%** |
| `types` | **84,7%** |

El 30,2% inicial de `eval` correspondía al perfil por paquete, no a toda la suite de integración. La diferencia cambia la evaluación: el problema es **combinaciones semánticas omitidas**, no ausencia general de ejecución de líneas. Los bugs de agregación son evidencia de que cobertura alta no garantiza cobertura de interacciones.

`go test -race ./...` también pasa. No hay evidencia en esta revisión de una data race reproducible. Esto no garantiza que un proveedor o listener mutable del usuario sea seguro, ni que se haya cubierto toda intercalación concurrente.

## B. ¿Un archivo grande significa mal diseño?

No por sí solo. CQL tiene numerosas reglas y operadores, y un dispatcher explícito puede ser más auditable que cientos de abstracciones pequeñas. La separación `compiler`/`sema`/`eval`/`types`/`model` funciona y debe conservarse.

El argumento de refactorización debe apoyarse en defectos: la agregación reimplementa parcialmente el flujo de filtrado y usa otra política de igualdad. El objetivo es compartir la semántica y el flujo de filas. “Bajar líneas por archivo” o “hacer pasar gocyclo” no son criterios de aceptación suficientes.

Las dependencias fijadas, CI con race, zonas adicionales y regresiones localizadas son señales favorables de calidad. Las exclusiones de complejidad del linter reducen una señal estática, pero no prueban por sí mismas deuda inaceptable.

## C. ¿Los benchmarks prueban que el parser es 64 veces más lento?

No. Frío y caliente difieren en construcción del Engine, parsing/check, preparación de includes y caché. Sólo se puede afirmar que **el escenario completo frío medido cuesta unas 62–65 veces el caliente** en esa biblioteca.

Tampoco convertir 262 µs a pacientes/segundo representa throughput clínico general: es una definición con 20 Encounters y 20 Observations, un proveedor local particular y sin red. No mide todas las poblaciones de una medida ni todos los pacientes.

B/op es asignación acumulada, no memoria residente. Los aproximadamente 18 MB/op desde frío no significan que cada biblioteca retenga 18 MB. Del mismo modo, una LRU de 64 entradas no garantiza 22 MB máximos: la entrada depende del AST y del plan concretos.

## D. ¿Las optimizaciones sugeridas atacan costos observados?

Añadí perfiles al benchmark caliente existente:

```sh
go test -run '^$' -bench BenchmarkEvaluateMeasureShapedLibrary -benchtime=2s \
  -cpuprofile=/tmp/go-cql-review-cpu.out \
  -memprofile=/tmp/go-cql-review-mem.out \
  -o /tmp/go-cql-review.test .
go tool pprof -top -alloc_space /tmp/go-cql-review.test /tmp/go-cql-review-mem.out
```

El perfil de asignaciones muestreado atribuye aproximadamente:

| Función, asignación directa (`flat`) | Fracción del perfil |
|---|---:|
| `Context.ChildScope` | 21,2% |
| `Evaluator.evalQuery` | 13,9% |
| `Context.functionRegistry` | 10,8% |
| `types.tryParseTemporalString` de FHIRPath | 7,9% |
| `benchProvider.Retrieve` | 3,6% |

Son porcentajes de **asignaciones muestreadas acumuladas durante la ejecución**, no de memoria retenida ni proporciones universales de CPU. El perfil incluye preparación y no aísla perfectamente warmup. Los costos acumulativos de llamadas se solapan: no deben sumarse como componentes independientes.

Esto respalda investigar ámbitos de funciones/FHIRHelpers, registros por evaluación, query y parsing temporal. También refuta que el proveedor artificial explique por sí solo toda la asignación del benchmark. La reutilización de un scope por query ya implementada no elimina los scopes creados por funciones.

El perfil CPU corto en macOS incluye mucho runtime (`kevent`, pthreads); no lo uso para fijar un ranking de optimizaciones de producción. Se necesita repetirlo en carga representativa y runner estable antes de atribuir ahorro de CPU.

## E. ¿El producto cartesiano implica una vulnerabilidad demostrada?

La materialización es verificable en el código; 10.000 × 10.000 implica 100 millones de mapas potenciales. **No ejecuté esa carga ni observé un OOM**. La gravedad operacional depende de quién puede suministrar CQL/datos, concurrencia y límites externos.

El riesgo sigue siendo relevante porque el límite por retrieve no limita las combinaciones ni el total de bytes. Un timeout cooperativo no es una cuota de memoria. La respuesta proporcionada es un presupuesto de filas/pasos y evaluación incremental; evitar afirmar que ya se midió una caída.

Un cambio a streaming debe preservar:

- Orden y multiplicidad definidos por cada operador.
- Alcance de aliases y `let` y semántica de relaciones.
- Propagación de errores, null e incertidumbre.
- Comportamiento de aggregate/distinct y sort.
- Cancelación, trazas y relación con el proveedor.

Sort puede seguir necesitando materialización; no todo el motor se convierte en streaming por cambiar el producto de fuentes.

## F. ¿Agregar más motores desde el primer día reduce el riesgo?

Puede aumentarlo si se dispersa el esfuerzo. Cada adaptador añade fuentes de desacuerdo: modelinfo, helpers, terminología, resolución de bibliotecas, serialización y preparación de datos. Java y JS comparten traducción; Google tiene capacidades diferentes.

Inicio recomendado: **go-cql frente a Java, corpus pequeño, normalizador probado y manifest estricto**. Después añadir JS. Google se incorpora cuando el subconjunto y las cargas comparables estén definidos. Una comparación correcta entre dos motores vale más que cuatro integraciones ambiguas.

En rendimiento, reportar por separado:

1. Preparación de CQL/modelos/dependencias.
2. Runtime con biblioteca preparada y proveedor equivalente.
3. Costo de servicio desde CQL, incluyendo lo que realmente hace cada implementación.

La JVM debe calentarse y los procesos deben persistir. Los adaptadores de rendimiento no deben ejecutar una JVM nueva por paciente ni compararla con un Engine Go caliente. Para conocer experiencia de cold start, ese costo se mide en un escenario separado.

## Conclusiones corregidas y prioridad

| Conclusión | Resultado adversarial |
|---|---|
| Arquitectura con bases sólidas | Se mantiene; no hay justificación para reescribirla ahora |
| Falta cobertura | Reformular: faltan combinaciones, oráculos externos obligatorios y guardas |
| Optimizaciones ya útiles | Confirmadas conceptualmente; profile muestra oportunidades restantes |
| Motor superior/inferior a Java/Google | Sin evidencia: no hubo benchmark cruzado |
| Riesgo de explosión de queries | Fundado por algoritmo; magnitud operacional pendiente de medir |
| Singleflight, pooling o bytecode prioritarios | No demostrado; dependen de perfiles y patrón de carga |

Orden final: corregir resultados → fortalecer oráculos y contratos → medir escalabilidad → optimizar scopes/queries/temporalidad según perfiles → ampliar motores y gates estadísticos. Conservar el diseño por fases y reutilizar el harness existente.

**Veredicto:** el proyecto muestra disciplina de desarrollo y orientación real a performance. Su punto débil principal es la distancia entre conformidad de operadores aislados y comportamiento de bibliotecas/queries combinadas. Esa distancia debe cerrarse antes de vender conformidad total o superioridad de rendimiento.
