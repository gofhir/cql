# Punto de reanudación

Al reanudar el trabajo o después de una compactación, este archivo es lo primero que se lee. Se actualiza al terminar cada paso. El proceso es el de `NUCLEO.md` §10 (`~/projects/personal/health-platform/workspace/NUCLEO.md`), y lo ejecuta la skill `fhir-incremento`. Rige para cql desde el 18-10-2026.

## Incremento activo

**003: dar veredicto a las divergencias pendientes.** Plan y registro: [003-clasificar-divergencias.md](003-clasificar-divergencias.md). Son cuatro PRs: el esquema, `types`, `fhir` y `arithmetic`.

**Estado al 19-10-2026:**
- **PRs 1 a 4 mergeados** con squash: #151 (esquema), #152 (`types`), #153 (`fhir`) y #154 (`arithmetic`, ADR 002). `main` está en 093572c. Como cada squash rompe la pila, el PR siguiente se rebasó con `--onto` antes de cada merge.
- **PR 5, `scope`:** está implementado **sin commit** en la rama `test/classify-scope`. El usuario aprobó el diseño. Las 180 filas son `reference defect`, y la deuda baja de 234 a 54. Se detectan las 84 mutaciones de `003-evidence.sh` y las 30 del 002. El issue de cql-execution está redactado y sin publicar. El review convergió en la ronda 2. Registro: plan 003, «Implementación del PR 5» y «Review del PR 5».

**Próximo paso concreto:**
1. `make check` (665 s) y `make evidence` (422 s, `docs/evidence/2026-10-20.md`) quedaron en verde el 20-10. Falta el commit y el PR del PR 5, que pide el usuario, y publicar el issue de cql-execution si el usuario lo aprueba.
2. Las filas `undecided` que la especificación no decide se le presentan al usuario por grupo.
3. El ADR 001 pasa a «Aceptado» cuando `pending-verdicts.tsv` quede vacío. Después del PR 5 quedarían 54 filas.

## Pendientes, en orden

1. **003:** dar veredicto a las divergencias pendientes de `testdata/snapshots/pending-verdicts.tsv`: eran 1485 (855 `unclassified` y 630 `undecided`); con el PR 2 quedaron 1037, con el PR 3, 471, y con el PR 4, 234. De esas 234, 180 son de `scope`, un área que el 003 no planificó: hace falta un PR 5 para que el ADR 001 pase a «Aceptado». Con eso, el ADR 001 pasa a «Aceptado» (decisión del usuario, 18-10-2026).
1b. **El arnés de referencia no codifica** Code ni Concept (Java y JS), ni Tuple ni los números de JS: escribe `Unencoded:`. Deja 14 filas de `types` sin veredicto (003, PR 2).
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
