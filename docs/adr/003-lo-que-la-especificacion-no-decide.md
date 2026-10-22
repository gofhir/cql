# ADR-003: lo que la especificación no decide

**Estado:** Aceptado. El usuario tomó las decisiones 1 a 4 el 20-10-2026 (plan 003, PR 6), la 5 el 21-10-2026 (plan 004, PR 2) y la 6 el 21-10-2026 (plan 005).
**Fecha:** 20-10-2026 · **Plan:** [`docs/plans/003-clasificar-divergencias.md`](../plans/003-clasificar-divergencias.md), PR 6

## Contexto

El ADR 001 pide que cada divergencia con las referencias tenga un veredicto. Los oráculos de `types`, `fhir` y `arithmetic` dejaron 38 filas `undecided`, en cuatro grupos donde la especificación (CQL 1.5.3) no dice qué responder. Este motor ya respondía en cada caso. Falta decidir qué respuesta es la de este motor por diseño, y registrarlo, porque NUCLEO pide que no haya nada hardcodeado sin ADR.

## Decisión

1. **`null is T` es false.** Appendix B, Is: true «if the run-time type of the argument is the same as or derived from the type being tested», y false si no. La especificación no dice cuál es el tipo en ejecución de null, y se lee que null no es de ningún tipo. Java responde null; cql-execution, false. El error interno de Java con `is Any` («Could not resolve type …runtime.Any») no es una lectura: es `reference defect`.
2. **`cast null as T` es null,** también cuando T no se relaciona con el tipo estático del operando. Appendix B, cast: la excepción se lanza «if the argument is not of the specified type at run-time», y null no tiene tipo en ejecución. El traductor de Java rechaza el cast entre tipos sin relación estática, y el plan 003 (PR 3, decisión 1) ya lo dio por CQL válido.
3. **Un cociente que se trunca en un número entero lo decide la lectura decimal exacta.** Types, Decimal: el tipo es decimal. Si la lectura de la especificación, que admite implementaciones en decimal y en binary64, deja un `div` o un `mod` a los dos lados de un entero, el oráculo calcula el árbol en decimal, a 40 cifras, con un margen de 10^-35 en cada valor que redondea (un cociente que no es exacto, un logaritmo, una exponencial, una raíz o una potencia topada), y decide solo si todo el intervalo da un mismo valor. Más allá de −60, donde el oráculo no calcula la exponencial, la acota entre 0 y e^−60. Si la referencia da ese valor y este motor, por la escala del ADR 002, da otro, la fila es `deliberate`. Si este motor lo da y la referencia no, es `reference defect`.
4. **Una llamada toma las conversiones que necesita.** Function Resolution no dice si una llamada toma dos conversiones implícitas seguidas, ni si una lista convierte cada elemento. Se toman las dos seguidas (FHIR.date → Date → DateTime). Un agregado sobre una lista que solo alcanza una firma convirtiendo cada elemento toma la firma del tipo en ejecución del elemento, y un elemento que no alcanza ninguna es null.

5. **Mode cuenta por igualdad.** Aggregate Functions, Mode, dice solo que devuelve «the statistical mode of the elements in source». No dice cómo compara los elementos ni cómo desempata, a diferencia de Distinct, In o IndexOf («using equality semantics»). Se cuenta como `distinct`:
   - por igualdad: un elemento null de un Code o de una tupla es igual a otro null, y un `=` indeterminado cuenta como distinto (los null de la lista misma, Mode los ignora);
   - así, dos Codes que solo difieren en display o versión son dos valores.

   Java falla en Mode de Codes («Code is not comparable»), y eso es `reference defect`. La lectura que se deja de lado es la equivalencia: el oráculo la calcula, y una referencia que responde por ella lee de otro modo lo que la especificación deja abierto.
6. **Con los dos argumentos null, en la pertenencia a un intervalo, decide el contenedor.** Interval Operators, Contains, dice «If the first argument is null, the result is false. If the second argument is null, the result is null», e In dice lo inverso. Con los dos null, las dos frases chocan. `(null as Interval<Integer>) contains (null as Integer)` y `(null as Integer) in (null as Interval<Integer>)` dan false, como en List In y List Contains, donde las dos frases coinciden. Java 3.26.0 y cql-execution 3.3.2 dan false. La lectura que se deja de lado es la del punto null, que da null en los dos: Interval In dice null para un punto null como primer argumento, y Contains, como segundo. Lo decidió el usuario el 21-10-2026 (plan 005).

Una referencia que responde otra cosa en estos casos lee de otro modo lo que la especificación deja abierto: la diferencia es `deliberate`. Estas son las filas `deliberate` que citan este ADR como su decisión, en cada área. Los 14 casts de null de `fhir` de la decisión 2 citan primero la decisión 1 del PR 3, y los nombra el plan 003:

- `types`:
<!-- ADR 003, types, written by -classify-types: begin -->
D129 D172 D265 D317 D435 D857 D881 D92 D99
<!-- ADR 003, types: end -->
- `fhir`:
<!-- ADR 003, fhir, written by -classify-fhir: begin -->
D117 D134 D194 D239 D244 D326 D577 D739 D84
<!-- ADR 003, fhir: end -->
- `arithmetic`:
<!-- ADR 003, arithmetic, written by -classify-arithmetic: begin -->
D380 D450
<!-- ADR 003, arithmetic: end -->
- `codesets`:
<!-- ADR 003, codesets, written by -classify-codesets: begin -->

<!-- ADR 003, codesets: end -->

## Verificación

- `TestTheTypesVerdictsFollowTheOracle`, `TestTheFHIRVerdictsFollowTheOracle`, `TestTheArithmeticVerdictsFollowTheOracle`, `TestTheCodesVerdictsFollowTheOracle` (decisión 5) y `TestTheMembershipVerdictsFollowTheOracle` (decisión 6): cada divergencia tiene el veredicto que el oráculo computa con estas decisiones.
- `TestTheTypesOracleAgreesWhereAllAgree` y `TestTheFHIROracleAgreesWithTheReferencesWhereAllAgree`: donde los tres motores coinciden, el oráculo da esa respuesta también en los casos que decide este ADR.
- `TestTheExactReadingHoldsItsControls`: la lectura exacta decide un cociente solo cuando todo su intervalo da un valor; `(1.0 / 3.0) * 3.0` y `Ln(Exp(2.0))` quedan sin decidir.
- `TestMembershipWithNullFollowsTheSpecification` (decisión 6): los dos casos de la decisión, escritos a mano, dan false.
- `TestTheUndecidedADRNamesItsRows`: este ADR nombra cada fila `deliberate` que lo cita, y ninguna otra.
- `TestTheDivergencesAreConsistent`: una fila `deliberate` cita el documento que la decidió, y ese documento la nombra.
