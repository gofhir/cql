# Pedido de aclaración a HL7 sobre CQL 1.5.3 (borrador, sin publicar)

Reúne el borrador del 016 (`docs/plans/016-duration-uncertainty.md`, «Pedido de aclaración a HL7») con los puntos pendientes del 020 y del 022. Las respuestas de las referencias salen de los snapshots del repo (`testdata/snapshots/<área>.java.json` y `.js.json`, con el nombre de la definición entre paréntesis), evaluadas a -03:00. Las filas marcadas «sonda 016» salen de las sondas de la revisión previa del 016 (31-10-2026), con las mismas versiones de las referencias. Las citas se cotejaron el 05-11-2026 contra la versión 1.5.3 publicada en cql.hl7.org/N1A, no contra la página sin versión, que hoy es la 2.0.0. No se publica sin el visto bueno del usuario.

---

**Title:** Duration and Difference in CQL 1.5.3: when the result is an uncertainty, seconds with milliseconds, non-hour offsets, and the Integer range

We maintain a Go implementation of CQL 1.5.3 (https://cql.hl7.org/N1A/) and compare it against the Java engine (5.4.0) and cql-execution (3.3.2). On the points below the two references answer differently from each other or from the text, and we could not settle the reading from the specification alone. All results are with an evaluation request at offset -03:00.

### 1. What makes a duration an uncertainty?

*Language Semantics, Determining Difference and Duration* says the between operator returns an uncertainty "to support the case where one or the other comparand in the duration operation does not specify components to the level of precision being determined". Its two examples are:

- `days between @2017-08-07T17:00 and @2017-08-14T // [6, 7]`, explained as "the starting datetime value is specified to the minute, 17:00, while the second value is only specified to the day". Both operands are specified to the day, the unit, so this reads as the precisions being different, not as an operand coarser than the unit.
- `days between @2012-01 and @2012-02`, where both operands are coarser than the unit.

cql-tests expects `years between DateTime(2005) and DateTime(2010)` to be `Interval[4, 5]`, with both operands at the unit and the same precision. The references disagree when both operands are finer than the unit:

| Expression | Java 5.4.0 | cql-execution 3.3.2 |
|---|---|---|
| `days between @2017-08-07T17:00 and @2017-08-14T` (the specification's example; sonda 016) | 6 | `[6, 7]` |
| `days between @2012-01 and @2012-02` (the specification's example; sonda 016) | null | `[1, 59]` |
| `years between DateTime(2012, 2, 29) and DateTime(2014, 2, 28)` (`durations`, D46) | 2 | `[1, 2]` |
| `days between @2012-01-01T09 and @2012-01-03T09:30` (sonda 016) | 2 | `[1, 2]` |
| `hours between @2012-01-01T01:00 and @2012-01-01T03:00:00` (sonda 016) | 2 | `[1, 2]` |

Questions:
1. Is the result an uncertainty when the operands have different precisions, or only when an operand is not specified to the unit? Is it one when both operands are finer than the unit, at the same precision or at different ones?
2. Which operators are defined on an uncertainty besides arithmetic and comparison: `between`, `in`, `during`, `before`, `after`, membership in a list? Today each implementation answers differently.
3. Does an uncertainty equal an identical one (`[1, 2] = [1, 2]`): null, or true?

### 2. Seconds and milliseconds as a single precision

Duration and Difference both say "seconds and milliseconds are considered a single precision with decimal semantics". The definition of DateTime says they "are combined and represented as a Decimal for the purposes of comparison, duration, and difference calculation. When milliseconds are null, they are combined as .0. In other words, if milliseconds are not specified, calculations for precisions above milliseconds should be performed as though milliseconds had been specified as 0."

| Expression | Java 5.4.0 | cql-execution 3.3.2 |
|---|---|---|
| `difference in seconds between @2012-01-31T23:59:59 and @2012-02-01T00:00:00.000` (`differences`, D82) | 1 | 1 |
| `difference in milliseconds between @2012-01-31T23:59:59 and @2012-02-01T00:00:00.000` (`differences`, D83) | `Interval[1, 1000]` | `Uncertainty[1, 1000]` |
| `milliseconds between @2012-01-01T01:00:00 and @2012-01-01T01:00:01` (`uncertainty`, D12) | null | `Uncertainty[1, 1000]` |

Read literally, `23:59:59` is `59.0` seconds and the second row is 1000. But the last sentence speaks of "precisions above milliseconds", which a duration in milliseconds is not. Questions:
1. With a unit of milliseconds, does a value without milliseconds read as `.0` (1000 in the second and third rows), or as unknown milliseconds (an uncertainty)?
2. If the answer depends on whether the operands have the same precision (as in point 1), does the third row, with both operands to the second, differ from the second?

### 3. Normalize, then truncate? Offsets that are not whole hours

Difference says: "When computing the difference between DateTime values with different timezone offsets, implementations should normalize to the timezone offset of the evaluation request timestamp, but only when the comparison precision is hours, minutes, seconds, or milliseconds." The text does not say whether each value is truncated to the precision before or after it is normalized. With an offset of +05:45 the order matters:

| Expression | Normalize, then count | Java 5.4.0 | cql-execution 3.3.2 |
|---|---|---|---|
| `difference in hours between @2012-01-01T10:10:00.000+05:45 and @2012-01-01T10:00:00.000Z` (`differences`, D766) | 6 (01:25 to 07:00 at -03:00) | 5 | 5 |
| `difference in hours between @2012-01-01T and @2012-01-01T10:10:00.000+05:45` (`differences`, D769) | `[-22, 1]` | `Interval[-21, 1]` | `Uncertainty[-21, 1]` |

Both references appear to truncate each value to the hour in its own offset and then normalize. Our reading is to normalize first, since the boundaries crossed are those of the evaluation request's offset. Which order is intended?

### 4. A duration or difference that does not fit an Integer

Duration and Difference return Integer, with no Long signature, and "the result of this operation is always an integer". *Arithmetic Operators* says "operations that cause arithmetic overflow or underflow, or otherwise cannot be performed (such as division by 0) will result in null, rather than a run-time error", but Duration and Difference are in *Date and Time Operators*. Seconds overflow 32 bits past 68 years, milliseconds past 24.8 days and minutes past 4085 years:

| Expression | True count | Java 5.4.0 | cql-execution 3.3.2 |
|---|---|---|---|
| `seconds between @1950-01-01T00:00:00.000 and @2020-01-01T00:00:00.000` (`durationrange`, D1946) | 2208988800 | -2085978496 | 2208988800 |
| `minutes between @0001-01-01T00:00:00.000 and @4084-01-24T02:08:00.000` (`durationrange`, D1883) | 2147483648 | -2147483648 | 2147483648 |
| `seconds between @2000-01-01T00:00:00.000Z and @2068-01-19T03:14Z` (`durationrange`, D1935) | `[2147483640, 2147483699]` | error: "Invalid Interval - the ending boundary (-2147483597) must be greater than or equal to the starting boundary (2147483640)." | `Uncertainty[2147483640, 2147483699]` |
| `difference in milliseconds between @2012-01T and @2012-01-01T` (`durationrange`, D1944) | `[-2678399999, 86399999]` | null | `Uncertainty[-2678399999, 86399999]` |

Java wraps the count in 32 bits; cql-execution answers a number that is not a CQL Integer. Our reading is that the overflow rule of *Arithmetic Operators* applies, so a count outside [-2^31, 2^31-1] is null, and an uncertainty with any bound out of range is null as a whole. Questions:
1. Is null the intended result?
2. If so, is a partly out-of-range uncertainty null, or does it keep the bounds that fit?
3. Does the same hold for `AgeIn...` and `CalculateAgeIn...`, which "are defined in terms of a datetime duration calculation"?

### 5. Which precision controls offset normalization in a comparison with no precision?

Every comparison operator in Appendix B says: "When comparing DateTime values with different timezone offsets, implementations should normalize to the timezone offset of the evaluation request timestamp, but only when the comparison precision is hours, minutes, seconds, or milliseconds." For `=` or `<` between a DateTime specified to the day or coarser and one specified to the millisecond with another offset, the text does not say what "the comparison precision" is. It could be the coarser precision of the pair, where the comparison stops, so the values are not normalized. Or it could be the finer one, as Same As uses "the finest precision specified in either input", so they are. Both references answer differently depending on the order of the operands (`mixeddates` area):

| Expression | Java 5.4.0 | cql-execution 3.3.2 |
|---|---|---|
| `@2020-03-01 = @2020-03-01T10:00:00.000+14:00` (D185) | false | false |
| `@2020-03-01T10:00:00.000+14:00 = @2020-03-01` (D186) | null | null |
| `@2020-03 = @2020-03-01T10:00:00.000+14:00` (D165) | false | false |
| `@2020-03-01T10:00:00.000+14:00 = @2020-03` (D166) | null | null |
| `@2020-03-01T10:00:00.000-12:00 = @2020-03-01` (D188) | false | false |
| `@2020-03-01 = @2020-03-01T10:00:00.000-12:00` (D187) | null | null |

Both references appear to move the right operand to the left operand's offset. Normalizing to the evaluation request's offset puts the +14:00 value on February 29, and the -12:00 one stays on March 1. Equality ought to be symmetric. Our engine answers null in both orders: it compares the components as written and does not normalize below the hour. Questions:
1. In `=`, `<` and the other comparisons with no precision, is the comparison precision the coarser of the pair, so offsets are not normalized and the answers above are null? Or is it the finer one, so they are normalized and D185, D186, D165 and D166 are false?
2. When a DateTime written with no offset takes the evaluation request's offset (Literals; ToDateTime of a Date), does that offset take part in such a comparison when the value is coarser than the hour?
