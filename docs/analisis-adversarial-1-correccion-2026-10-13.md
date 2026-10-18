# Revisión adversarial 1: corrección, conformidad y oráculos

Realizada después de la [primera entrega](analisis-proyecto-2026-10-13.md). Es una segunda pasada del mismo revisor con hipótesis contrarias, no una auditoría de un tercero.

## Hipótesis a atacar

«La arquitectura semántica y la suite oficial hacen que el motor sea confiable, y basta ampliar los fixtures Java para demostrarlo».

La afirmación es demasiado fuerte: hay caminos que aceptan código no validado, formatos externos que pierden información y tests verdes que no representan equivalencia de resultados.

## A. ¿Los contraejemplos de la primera entrega son CQL aceptado externamente?

Ejecuté el traductor Java **3.26.0** ya disponible localmente, con `Translate.java` del proyecto, sobre cinco definiciones: agregación con `with`, con `without`, colisión de distinct, union y except. La traducción terminó con exit code 0 y produjo ELM con tres `Query`, un `Union` y un `Except`.

Esto refuta que los hallazgos sean simplemente expresiones inventadas que el traductor rechaza. **No ejecuté esos valores en el runtime Java**: aceptar/traducir no demuestra el resultado de ejecución. Los valores esperados de relaciones y deduplicación se derivan del significado de los operadores y de los inputs mínimos, con esa limitación declarada.

La colisión de `distinctCombos` también se reproduce sin nulos ni tipos mixtos:

```cql
define R:
  from ({'a,b','a'}) X, ({'c','b,c'}) Y
    aggregate distinct A starting 0: A + 1
```

Las cuatro parejas son distintas; el motor devuelve **3**. Las parejas `('a,b','c')` y `('a','b,c')` se convierten en la misma clave textual. La reparación necesita igualdad estructural; no se resuelve añadiendo únicamente etiquetas de tipo.

## B. ¿La validación semántica protege todo el grafo?

Nuevo contraejemplo confirmado:

```cql
// Fuente que devuelve LibraryResolver:
library Bad version '1.0'
define X: if true then 1 else 1 + 'bad'

// Biblioteca principal:
library Probe
include Bad version '1.0' called B
define R: B.X
```

Con `NewEngine(WithLibraryResolver(...))`, `R` devuelve **1 sin error**, aun con la validación semántica activada por defecto. `Engine.Check` aplicado directamente a `Bad` produce error en la rama inválida.

La causa es concreta: `newEvalContext` rechaza errores del plan principal; `resolveIncludesInto` calcula y registra el plan incluido, pero no rechaza sus diagnósticos. La rama no ejecutada permite observar éxito. Una variante que ejecuta `1 + 'bad'` sí falla en runtime: **no hay evidencia de que esa variante devuelva aritmética incorrecta**, y no debe afirmarse.

Severidad propuesta: **P1 para el contrato de validación del grafo**. No clasificar la rama muerta como un resultado clínico erróneo demostrado. Un sistema que promete validar la biblioteca debe decidir y documentar si esa garantía incluye las dependencias; recomiendo que las incluya.

Test a introducir: misma biblioteca inválida como raíz, include directo, include transitivo y LibraryLoader; exigir rechazo semántico consistente. Agregar variantes con funciones y branches no recorridas por el paciente.

## C. ¿ELM interoperable está suficientemente demostrado?

Nuevo hallazgo: el traductor oficial emite un documento con raíz `{"library": {...}}`. `elm.UnmarshalLibrary` deserializa directamente a `elm.Library`, cuyos campos son los del contenido interior. Con el documento oficial recién generado devuelve **Identifier=nil, Statements=nil, error=nil**.

Consecuencia: conectar las APIs directamente puede producir una importación vacía aparentemente exitosa. Esto demuestra una incompatibilidad de integración y aceptación silenciosa; **no prueba que el método prometiera aceptar ese envelope**, porque su comentario no distingue documento completo de objeto interno.

Recomendación: exponer/documentar ambos niveles y rechazar estructura vacía no válida. Agregar un fixture producido por Java, no sólo roundtrips del serializador propio.

Además, `Statements.Def` está declarado como `[]*ExpressionDef`, e `importStatements` sólo construye definiciones de expresión. Existe un tipo `FunctionDef`, pero ese camino no lo usa. El soporte real de funciones oficiales debe verificarse antes de habilitar ELM como oráculo alternativo completo.

El `default` que convierte nodos desconocidos en null sigue confirmado. Es especialmente peligroso para un test diferencial: importar ELM incompleto y normalizar sus nulls podría producir falsos acuerdos.

## D. ¿Union y except eran sólo una sospecha?

Contrasté la documentación histórica HL7 de CQL **1.5.0, ballot de mayo de 2020**, para evitar basar el diagnóstico únicamente en la página actual 2.0:

- `union` de listas exige elementos únicos.
- `except` usa semántica de conjunto y limita cada elemento del resultado a una aparición.

Por tanto, para `{1,1,2} union {2}` y `{1,1,2} except {3}`, el resultado debe contener una única aparición de 1 y de 2; el orden no es el punto del hallazgo. El motor conserva los dos unos. Se elevan a **defectos confirmados contra esas reglas documentadas**, con resultado runtime externo aún pendiente.

Fuentes: [Union, HL7 1.5.0](https://cql.hl7.org/2020May/09-b-cqlreference.html#union-1), [Except, HL7 1.5.0](https://cql.hl7.org/2020May/09-b-cqlreference.html#except-1). Al definir el perfil soportado del proyecto debe fijarse la publicación normativa exacta; el ballot histórico no sustituye ese trabajo.

## E. ¿Los propios tests podrían ocultar los errores?

Sí, en cuatro puntos que requieren endurecimiento:

| Ataque | Por qué puede pasar | Gate necesario |
|---|---|---|
| Fixture Java con cero definiciones | El 75% de cero es cero | Manifest e IDs obligatorios, además del mínimo absoluto |
| Operación inválida falla por otra causa | Conformidad acepta cualquier error para `invalid` | Fase y categoría de fallo esperadas |
| Excepción conocida empieza a fallar distinto | Se exige “no pasar”, sin fijar discrepancia | Valor/error exacto autorizado para ese input |
| eCQM con mismatch conocido se rompe en runtime | `diff` contiene tanto diferencias como errores y ambos se excusan | Separar fallo de ejecución y discrepancia del oráculo |

La última brecha existe también en `TestECQMCases`: el caso conocido no compara la naturaleza del diff. Las guardas de mínimo de oráculos y excepciones sobrantes son buenas, pero resuelven problemas diferentes.

El normalizador propuesto debe tener suite propia independiente. Reutilizar `Equal` del motor conserva el riesgo de que el mismo defecto afecte al resultado y al veredicto. También hay que preservar procedencia y precisión temporal y distinguir error, null y colección vacía.

## Cambios exigidos a la recomendación inicial

1. Subir la validación del grafo y la importación estricta al primer lote, junto a agregación y distinct.
2. Usar Java como referencia principal de traducción y ejecución, pero no confundir las dos fases.
3. Mantener JS como segundo runtime; comparte frontend y no constituye independencia total. Google añade otro frontend sólo en su subconjunto compatible.
4. Resolver discrepancias con especificación/versiones y casos mínimos, no por votación ni porque dos motores fallen.
5. Corregir la presentación de conformidad: “1820/1823 en un corpus fijado” es evidencia limitada, no certificación universal.

**Veredicto:** la estrategia de tests diferenciales es adecuada, pero necesita validar antes los adaptadores, el normalizador y los manifests. Ampliar casos sin esas guardas puede aumentar el número de verdes sin aumentar confianza.
