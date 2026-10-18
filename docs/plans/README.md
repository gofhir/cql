# Punto de reanudación

Al reanudar el trabajo o después de una compactación, este archivo es lo primero que se lee. Se actualiza al terminar cada paso. El proceso es el de `NUCLEO.md` §10 (`~/projects/personal/health-platform/workspace/NUCLEO.md`), y lo ejecuta la skill `fhir-incremento`. Rige para cql desde el 18-10-2026.

## Incremento activo

Ninguno. **002, el ADR único,** se cerró el 18-10-2026 y sigue sin commit: [002-adr-semantica-cql.md](002-adr-semantica-cql.md).

**Próximo paso concreto:** el commit, el push y el PR del 002, cuando el usuario lo pida. Después, el 003.

## Pendientes, en orden

1. **003:** clasificar las 855 divergencias sin clasificar (`types`, `arithmetic` y `fhir`). Con eso, el ADR 001 pasa a «Aceptado» (decisión del usuario, 18-10-2026).
2. **`distinct` y operaciones de conjuntos sobre intervalos** son O(n²). Con un plazo ya no corren de más, desde #144, pero siguen siendo lentas: hace falta un índice por cubetas para intervalos.
3. **D4a, identidad de tipo:** un descriptor `{modelo, nombre}` en vez del texto de `Type()`. Ver [arquitectura-verificacion-2026-10-18.md](../arquitectura-verificacion-2026-10-18.md) §4.
4. **ELM oficial:** el documento completo del traductor y los 54 de 115 tipos de nodo que faltan. Ya hay un arreglo listo para entrar cuando sea: `Abs` se importa como `OpPositive`.
5. **Borrar la carga perezosa de `eval`** (`ensureLibraryLoaded`): desde #146 es inalcanzable desde `Engine`.
6. **Traducir los comentarios del código al español,** como pide la carta. Va en un incremento propio; hasta entonces, los comentarios nuevos van en inglés para no mezclar idiomas en un mismo archivo (decisión del usuario, 18-10-2026).
7. **Verificar contra las referencias** si un `valueQuantity` que solo trae `value` es `FHIR.Quantity`. El motor dice que no; se observó en el 001.

## Decisiones pendientes del usuario

- Publicar el release que acumula #144, #146, #147 y #148.
- Agregar Australia/Lord_Howe a la matriz de zonas de CI.

## Cerrados con el §10

| PR | Qué |
|---|---|
| #148 | 001: la división Decimal y los oráculos que distinguen el tipo de los números |
| — | 002: el ADR 001 (Propuesto), su gobierno en CI, la procedencia de la suite y la deuda de divergencias (sin commit) |

## Cerrados el 18-10-2026, antes de adoptar el §10

| PR | Qué |
|---|---|
| #144 | Cancelación: `ErrCanceled`, el plazo real en `ErrTimeout` y corte de los bucles largos |
| #146 | Resolución de includes: identidad, ciclos, `using` en todo el grafo y `WithLibraryLoader` deprecado |
| #147 | Semántica de includes: `ErrSemantic`, `Diagnostic.Library`, la posición del include y `CheckContext` |

El defecto del loader de `gofhir/server`, que está congelado, quedó en `opensource/innovation/paito_backlog.md` §4b.
