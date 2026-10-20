# ADR-002: la escala de Decimal

**Estado:** Aceptado. El usuario tomó las decisiones en el #134 (16-10-2026) y aprobó este registro el 19-10-2026 (plan 003, PR 4).
**Fecha:** 19-10-2026 · **Plan:** [`docs/plans/003-clasificar-divergencias.md`](../plans/003-clasificar-divergencias.md), PR 4

## Contexto

CQL 1.5.3 (Types, Decimal) le da al tipo «a precision of 28 and a scale of 8», dentro de (10^28 − 1)/10^8 en ambos signos. También dice: «implementations must support at least the 28 digits of precision and 8 digits of scale, but may support more precise values». No fija en qué operación se aplica la escala ni con qué redondeo. Esa es una elección de cada implementación, y responde distinto según la elección: Java recorta con FLOOR y cql-execution calcula en binary64 sin recortar.

El motor eligió en el #134, y sus excepciones quedaron en `decidedRuntimeAnswers`. Pero la regla no estaba escrita en ningún documento que la gobierne, y NUCLEO pide que no haya nada hardcodeado sin ADR.

## Decisión

**Un Decimal se lleva a la escala 8 con FLOOR,** hacia menos infinito, igual que Java. Un valor que ya está dentro de la escala se deja como está. Lo que pasa del rango del tipo es null: «cannot be represented» (plan 003, PR 4, decisión 2).

| Operación | Escala | Cómo |
|---|---|---|
| `+`, `*`, `mod` | FLOOR a 8 | en decimal exacto |
| `/` | FLOOR a 8 | el cociente exacto, sin redondear antes |
| `-` | no se recorta | Java tampoco la recorta |
| `div` | entera | trunca hacia cero |
| `^` y `Power` con exponente entero | FLOOR a 8 | en decimal exacto; con exponente negativo es Decimal (la conformance pide `Power(10, -7) = 0.0000001`) |
| `^` y `Power` con otro exponente | FLOOR a 8 | en binary64, leído por el decimal más corto que lo imprime (`DecimalPowerShortestFloat`) |
| `Log` | FLOOR a 8 | en binary64, salvo una potencia exacta, que da su logaritmo entero (`DecimalLogNearInteger`) |
| `Ln` | no se recorta | en binary64; la conformance pide `Round(Ln(1000), 8) = 6.90775528` (`DecimalLnUnscaled`) |
| `Exp` | no se recorta | en binary64; Java tampoco lo recorta |
| `Abs`, `Negate`, `Round`, `Truncate`, `Floor`, `Ceiling` | no agregan cifras | `Round` redondea la mitad alejándose de cero (Appendix B) |
| `Sum`, `Product` | paso a paso | cada suma o producto, como en `+` y `*` |
| `Avg` | paso a paso | la suma, y después `/` |
| `Median` | el par, como `/` | la suma de los dos centrales y después `/` 2 |
| `Variance`, `PopulationVariance` | paso a paso | la media, las diferencias (`-`, sin recortar), sus cuadrados, la suma y `/` |
| `StdDev`, `PopulationStdDev` | FLOOR a 8 | la raíz de la varianza, en binary64 |
| `GeometricMean` | no se recorta | la raíz exacta (`DecimalGeometricMeanUnscaled`); no está en el corpus `arithmetic` |
| Quantity | no se recorta | la escala es la del tipo Decimal; una dosis en una unidad chica conserva sus cifras (`QuantityDivisionUnscaled`) |

Una referencia que da una respuesta conforme con otra escala no está equivocada. La diferencia es `deliberate`, y la nombra el plan 003 (PR 4, decisión 1).

## Verificación

- `TestTheScalePolicyIsConformant`: en cada una de las 800 definiciones de `testdata/snapshots/arithmetic.cql` que la especificación decide, la política da una respuesta que la especificación permite. El intervalo conforme se calcula desde la especificación, no desde la política.
- `TestTheEngineFollowsTheScalePolicy`: el motor sigue esta tabla en cada definición fuera de las divergencias.
- `TestTheArithmeticVerdictsFollowTheOracle`: cada divergencia del área tiene el veredicto que computa el oráculo.
- `TestRuntimeMatchesTheReference`: las excepciones de `decidedRuntimeAnswers`.
- `TestConformance`: la suite oficial, con `Round(Ln(1000), 8)` y `Power(10, -7)`.
