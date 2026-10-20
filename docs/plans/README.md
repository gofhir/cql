# Punto de reanudación

Al reanudar el trabajo o después de una compactación, este archivo es lo primero que se lee. Se actualiza al terminar cada paso. El proceso es el de `NUCLEO.md` §10 (`~/projects/personal/health-platform/workspace/NUCLEO.md`), y lo ejecuta la skill `fhir-incremento`. Rige para cql desde el 18-10-2026.

## Incremento activo

**003: dar veredicto a las divergencias pendientes.** Plan y registro: [003-clasificar-divergencias.md](003-clasificar-divergencias.md). Son cuatro PRs: el esquema, `types`, `fhir` y `arithmetic`.

**Estado al 19-10-2026:**
- **PR 1, el esquema de la evidencia y la guarda contra veredictos obsoletos:** cerrado, con 3 rondas de review que convergieron. Está en el commit `cdda704` de la rama `test/divergence-evidence`, **ya subida**. **Falta que el usuario abra el PR** (`gh pr create … --head test/divergence-evidence`), seguir el CI y mergear.
- **PR 2, el área `types`:** implementado **sin commit** en la rama `test/classify-types`, apilada sobre la del PR 1.
  - El oráculo está en `engine_types_oracle_test.go`, y la clasificación quedó así: 306 `reference defect`, 142 `own defect`, 10 `undecided` y 14 `unclassified` (el arnés no codifica la respuesta de la referencia). La deuda queda en 1037.
  - Las 34 mutaciones de `003-evidence.sh` se detectan.
  - Las mutaciones del 002 se arreglaron para la deuda que se achica: el control del arranque toma el contenido inicial del historial de git, y el canje usa `arithmetic` D0 java.
  - **Al clasificar `arithmetic` (PR 4), esa mutación hay que moverla a otra fila pendiente.**
  - `make check` (521 s, 1,22 GB de RSS) y `make evidence` (538 s, 1,12 GB) en verde el 19-10-2026, después de las tres rondas.

**Próximo paso concreto:**
1. El PR 2 convergió en la ronda 3 de review, con `make check` y `make evidence` en verde (19-10-2026), y está commiteado en `test/classify-types`, **sin push**.
2. Push y el PR del PR 2, con base en `test/divergence-evidence` o en `main` una vez mergeado el PR 1.
3. El PR 3, el área `fhir`, con 143 `unclassified` más 450 `undecided`. Entre las `undecided` hay 390 que el traductor Java rechaza y que hay que juzgar como CQL válida o no.
4. El PR 4, `arithmetic`, con 240.
5. Las filas `undecided` que la especificación no decide se le presentan al usuario por grupo.
6. El ADR 001 pasa a «Aceptado» cuando `pending-verdicts.tsv` quede vacío.

## Pendientes, en orden

1. **003:** dar veredicto a las divergencias pendientes de `testdata/snapshots/pending-verdicts.tsv`: eran 1485 (855 `unclassified` y 630 `undecided`) y, con el PR 2, quedan 1037. Con eso, el ADR 001 pasa a «Aceptado» (decisión del usuario, 18-10-2026).
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
