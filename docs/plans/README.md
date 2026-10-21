# Punto de reanudación

Al reanudar el trabajo o después de una compactación, este archivo es lo primero que se lee. Se actualiza al terminar cada paso. El proceso es el de `NUCLEO.md` §10 (`~/projects/personal/health-platform/workspace/NUCLEO.md`), y lo ejecuta la skill `fhir-incremento`. Rige para cql desde el 18-10-2026.

## Incremento activo

**003: dar veredicto a las divergencias pendientes.** Plan y registro: [003-clasificar-divergencias.md](003-clasificar-divergencias.md). Son cuatro PRs: el esquema, `types`, `fhir` y `arithmetic`.

**Estado al 19-10-2026:**
- **PRs 1 a 4 mergeados** con squash: #151 (esquema), #152 (`types`), #153 (`fhir`) y #154 (`arithmetic`, ADR 002). `main` está en 093572c. Como cada squash rompe la pila, el PR siguiente se rebasó con `--onto` antes de cada merge.
- **PR 5, `scope`:** está en el PR #155 (rama `test/classify-scope`), sin merge. El usuario aprobó el diseño. Las 180 filas son `reference defect`, y la deuda baja de 234 a 54. Se detectan las 84 mutaciones de `003-evidence.sh` y las 30 del 002. El issue de cql-execution está redactado y sin publicar. El review convergió en la ronda 2. Registro: plan 003, «Implementación del PR 5» y «Review del PR 5».

**Próximo paso concreto:**
1. PRs 1 a 6 mergeados (#151–#156); `main` en b508841. ADR 003 aceptado.
2. **PR 7** (rama `test/harness-encodes-all`): el arnés codifica Code, Concept, los números de JS sin tipo estático y los objetos planos, para las 16 filas `unclassified`. Implementado. El review no convergió en 3 rondas, y el usuario decidió R3-2: el campo vacío del motor se escribe como el null que responde. `make check` (725 s, 1,15 GB) y `make evidence` (651 s, 1,24 GB) en verde el 20-10-2026. Falta el commit y el PR, si el usuario los pide.
3. El ADR 001 pasa a «Aceptado» cuando `pending-verdicts.tsv` quede vacío, al cerrar el PR 7.

## Pendientes, en orden

1. **003:** cerrado con el PR 7. `pending-verdicts.tsv` está vacío y el ADR 001 está «Aceptado».
1b. **Un `''` explícito en un Code se vuelve null:** `Code{code:'x', system: ''}.system is null` da true (sonda del 20-10-2026, R3-2 del PR 7). `types.Code` y `types.Concept` guardan sus campos como `string`, así que no distinguen `''` de null. Ninguna definición del corpus lo ejercita. Es un cambio del motor y va en su propio PR. Pasa lo mismo con `Concept.codes` null, que se vuelve `{}`.
1c. **`make evidence` llega a 1,08 GB de RSS y tarda 515 s.** El oráculo de `types` no lo explica (0,5 s, 11 MB): hay que medir dónde está el costo y corregirlo.
2. **`distinct` y operaciones de conjuntos sobre intervalos** son O(n²). Con un plazo ya no corren de más, desde #144, pero siguen siendo lentas: hace falta un índice por cubetas para intervalos.
3. **D4a, identidad de tipo:** un descriptor `{modelo, nombre}` en vez del texto de `Type()`. Ver [arquitectura-verificacion-2026-10-18.md](../arquitectura-verificacion-2026-10-18.md) §4.
4. **ELM oficial:** el documento completo del traductor y los 54 de 115 tipos de nodo que faltan. Ya hay un arreglo listo para entrar cuando sea: `Abs` se importa como `OpPositive`.
5. **Borrar la carga perezosa de `eval`** (`ensureLibraryLoaded`): desde #146 es inalcanzable desde `Engine`.
6. **Traducir los comentarios del código al español,** como pide la carta. Va en un incremento propio; hasta entonces, los comentarios nuevos van en inglés para no mezclar idiomas en un mismo archivo (decisión del usuario, 18-10-2026).
7. **Verificar contra las referencias** si un `valueQuantity` que solo trae `value` es `FHIR.Quantity`. El motor dice que no; se observó en el 001.

- **Defectos propios que encontró el 003, en el área `types`:**
  - `cast` y `as` hacia un tipo genérico (`List<T>`, `Interval<T>`) no comprueban el tipo (58 definiciones);
  - `cast` convierte en vez de fallar (11);
  - Long sin tipo en ejecución (10; D4c).
- **Defectos propios que encontró el 003, en el área `fhir`:** siete grupos, listados en el plan 003, «Implementación del PR 3». Entre ellos, el constructor de una instancia FHIR que pierde el valor y las llamadas sin firma posible que no se rechazan.

## Decisiones pendientes del usuario

- Publicar el release que acumula #144, #146, #147 y #148.
- Agregar Australia/Lord_Howe a la matriz de zonas de CI.

## Cerrados con el §10

| PR | Qué |
|---|---|
| #148 | 001: la división Decimal y los oráculos que distinguen el tipo de los números |
| #150 | 002: el ADR 001 (Propuesto), su gobierno en CI, la procedencia de la suite y la deuda de divergencias |

## Cerrados el 18-10-2026, antes de adoptar el §10

| PR | Qué |
|---|---|
| #144 | Cancelación: `ErrCanceled`, el plazo real en `ErrTimeout` y corte de los bucles largos |
| #146 | Resolución de includes: identidad, ciclos, `using` en todo el grafo y `WithLibraryLoader` deprecado |
| #147 | Semántica de includes: `ErrSemantic`, `Diagnostic.Library`, la posición del include y `CheckContext` |

El defecto del loader de `gofhir/server`, que está congelado, quedó en `opensource/innovation/paito_backlog.md` §4b.
