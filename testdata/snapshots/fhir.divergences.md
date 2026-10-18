# The fhir area against the reference engines

The `fhir` corpus (`fhir.cql`, over `fhir-data.json`) run by the Java engine
3.26.0 with engine-fhir (`fhir.java.json`) and by cql-execution 3.3.2 with
cql-exec-fhir 2.1.6 (`fhir.js.json`), each definition in a library of its own
and all three engines on the same FHIRHelpers 4.0.1 text. Measured on
18 October 2026, against main at `8090460`.

Of the 900 definitions, the Java translator refuses 574: it types strictly, and
refuses a cast between unrelated types and an operator a choice does not fit,
which this engine accepts. The 326 it accepts compare as follows.

| Outcome | Definitions |
|---|---|
| All three agree | 177 |
| This engine agrees with the Java engine; cql-execution differs | 63 |
| This engine agrees with cql-execution; the Java engine differs | 32 |
| **Both references agree, and this engine differs** | **31** |
| All three differ | 23 |

This is an inventory, not a list of decisions: nothing here is fixed or
excused yet. Each group below is a change of its own, measured against both
references before it lands.

## Both references agree, this engine differs

**A FHIR primitive is a FHIR.Element** (12). `Patient.gender is Element`,
`Patient.birthDate as FHIR.Element`, and the same for `Patient.id`,
`Patient.deceased`, `period.start`, an Observation's `effective` and a
`valueString`, are true or the value in both references, and false or null
here: a primitive read from the data reports its System type's name, and the
model's descent finds nothing for it. D53, D79, D145, D207, D228, D240, D287,
D371, D543, D617, D772, D808.

**Arithmetic on a choice is refused by the semantic check** (6). `Patient.deceased
+ 1 day` is 2020-02-03 in both, and `effective + 1 day` 2020-01-02; this engine
fails the library with "Choice<…>". `-(O.value)` and `effective + 1 'mg'` are
null in both. D104, D111, D156, D296, D299, D390.

**A choice that is no number at run time is null** (3). Settled: `Abs`, `Sum`
and `Avg` of a `valueString` are null here too, as in both, where #139 failed
with "String is not a number or a quantity" (and main before it answered 0).
An operand typed as no number at all still fails. D10, D374, D792.

**A type name is read without its model** (2). `Patient.id is String` and a
`valueInteger is Integer` are false in both: FHIR.id is no System.String, and
FHIR.integer no System.Integer. D114, D600.

**An element is no string** (2). `ToString(effective)` of a Period is null in
both, and `cast O.value as FHIR.string` of a Quantity an error; this engine
answers the element's JSON as text. D322, D369.

**Disputed, or representation** (9):

- `Abs` and `Sum` of a FHIR Quantity are null in both, and the System Quantity
  here, through FHIRHelpers (D194, D509).
- `cast O.value as Quantity` is the FHIR element in both, and the System
  Quantity here, as decided in #139 (D579).
- `cast` of a string as FHIR.integer is an error in both, and null here (D16).
- `as FHIR.dateTime` is the FHIR primitive in both, and a System DateTime here
  (D250, D389).
- `Abs` of a `valueInteger` keeps the FHIR primitive here, where both answer a
  System Integer (D782).
- `cast { 1, 2 } as Any` fails in the Java engine on Any, as recorded in
  `testdata/reference/runtime.divergences.md` (D797).

## Regenerating the references

See "The fhir area" in `internal/referenceharness/README.md`. Both files carry
the SHA-256 of `fhir.cql`, and `TestTheFHIRReferencesAreOfThisCorpus` fails when
the corpus moved without them.
