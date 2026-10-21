# Punto de reanudación

Al reanudar el trabajo o después de una compactación, este archivo es lo primero que se lee. Se actualiza al terminar cada paso. El proceso es el de `NUCLEO.md` §10 (`~/projects/personal/health-platform/workspace/NUCLEO.md`), y lo ejecuta la skill `fhir-incremento`. Rige para cql desde el 18-10-2026.

## Incremento activo

**004: los campos de Code y Concept distinguen null de `''`.** Plan: [004-campos-de-code.md](004-campos-de-code.md). Son dos PRs: la representación y lectura, y la igualdad y equivalencia.

**Estado al 21-10-2026:**
- El 003 está cerrado: los PRs 1 a 7 están mergeados (#151–#157), `main` está en 5be6763, `pending-verdicts.tsv` está vacío y el ADR 001 está «Aceptado».
- El usuario eligió los campos `*string`.
- Plan aprobado el 21-10-2026, con A1 a A12. La igualdad sigue la especificación, y sale como v1.24.0. Rama: `fix/code-empty-field`.

**Próximo paso concreto:** el PR 1 está MERGEADO como #158 (8a6d2f9). El PR 2 está implementado y sin commit, en la rama `fix/code-fields-sets`. Las rondas 1 y 2 de review están resueltas (ver el plan 004), y la decisión 5 del ADR 003 (Mode por igualdad) la tomó el usuario. Las mutaciones de 004 son 53, todas detectadas. La ronda 3 converge, y `make check` pasó. Falta `make evidence` y pedir el commit.

## Pendientes, en orden

1. **003:** cerrado con el PR 7. `pending-verdicts.tsv` está vacío y el ADR 001 está «Aceptado».
1b. **Un `''` explícito en un Code se vuelve null:** `Code{code:'x', system: ''}.system is null` da true (sonda del 20-10-2026, R3-2 del PR 7). `types.Code` y `types.Concept` guardan sus campos como `string`, así que no distinguen `''` de null. Ninguna definición del corpus lo ejercita. Es un cambio del motor y va en su propio PR. Pasa lo mismo con `Concept.codes` null, que se vuelve `{}`.
1c2. **Igualdad de listas con un par indeciso y otro desigual:** la especificación no lo decide, y no es un defecto. Hay tres lecturas:
  - la del motor y Java, que recorren en orden y se detienen en el primer par indeciso;
  - la de JS, que da false si un Code difiere;
  - la de los tres motores con un elemento null, que dan null.
  Lo decide el usuario. Es el mismo frente que `tupleEqual` (A1 del plan 004). Ver «Review, ronda 1» del plan 004 (D334, D593).
1c3. **Restos del 004 que no son de su alcance:**
  - un codesystem de una librería incluida (`Code '1' from H."CS"`, o un code declarado así) no se resuelve: da `system` "H.CS" y `version` null;
  - `buildConcept` descarta un `(null as Code)` de `codes`;
  - `~` entre Code y Concept es asimétrico;
  - sema acepta un code de un codesystem no declarado.
1c4. **`in` y `contains` con un null:** `{1, 2} contains (null as Integer)` y `(null as Integer) in {1, 2}` dan null. La especificación (List Operators, In y Contains) dice false si la lista no tiene ningún null, y las dos referencias dan false. Pasa con todos los tipos y es anterior al 004. Son 10 `own defect` por referencia en `codesets`.
1c5. **Mode no agrupa todo lo que `=` iguala:** fuera de los números y los Codes, la clave de Mode es el `String()` del valor, y separa valores iguales: `Mode({1 'mg', 1 'mg', 1.0 'mg', 1.0 'mg', 2 'mg', 2 'mg', 2 'mg'})` da `2 'mg'`, y `distinct` da dos. Pasa también con `1 'g'`/`1000 'mg'`, `1 week`/`7 days`, Dates de distinta precisión, DateTimes de distinto offset e intervalos cerrados y abiertos de los mismos puntos. Es anterior al 004 (idéntico en main) y lo encontraron Codex y Fable en la ronda 3 del PR 2.
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
