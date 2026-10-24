# Drift 2026-10-22: org.cqframework 5.4.0 and cql-exec-fhir 2.2.0

The Java references move from `info.cqframework` 3.26.0 to `org.cqframework` 5.4.0 (`cql-to-elm-jvm`, `engine-jvm`, `engine-fhir-jvm`, `quick`), and cql-exec-fhir from 2.1.6 to 2.2.0; cql-execution stays at 3.3.2. The environment is the one of 7419c47 in both sets (`docs/evidence/007-drift-environment.md`). Old set: the references at 7419c47; new set: `sh scripts/references.sh write` on top (two identical runs), with the manifests rewritten by `-update-divergences` and the classifiers, and the decisions of plan 007 («Implementación del PR 2»). Written by

```sh
go run ./internal/referenceharness/differential/drift -old <7419c47>/testdata/snapshots -new testdata/snapshots [-ref js] -divergences arithmetic integer scope types fhir codes codesets membership fhirmembership division
```

## Java

| Area | same | message only | error class changed | error → value | value → error | value changed | refused → accepted | accepted → refused | added | removed |
|---|---|---|---|---|---|---|---|---|---|---|
| arithmetic | 683 | 21 | 0 | 68 | 0 | 28 | 0 | 0 | 0 | 0 |
| integer | 1000 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| scope | 600 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| types | 608 | 78 | 0 | 94 | 92 | 28 | 0 | 0 | 0 | 0 |
| fhir | 698 | 154 | 0 | 33 | 2 | 13 | 0 | 0 | 0 | 0 |
| codes | 600 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| codesets | 354 | 46 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| membership | 378 | 0 | 0 | 0 | 0 | 22 | 0 | 0 | 0 | 0 |
| fhirmembership | 178 | 21 | 0 | 0 | 0 | 1 | 0 | 0 | 0 | 0 |
| division | 29 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| all | 5128 | 320 | 0 | 195 | 94 | 92 | 0 | 0 | 0 | 0 |

## cql-execution

cql-execution runs the ELM the 5.4.0 translator writes, and cql-exec-fhir 2.2.0 reads the data; no answer moves.

| Area | same | message only | error class changed | error → value | value → error | value changed | refused → accepted | accepted → refused | added | removed |
|---|---|---|---|---|---|---|---|---|---|---|
| arithmetic | 800 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| integer | 1000 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| scope | 600 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| types | 900 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| fhir | 326 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| codes | 600 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| codesets | 400 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| membership | 400 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| fhirmembership | 179 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| division | 29 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| all | 5234 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |

## Verdicts

The rows of the manifests, before and now; both reports hold the same table, since a manifest holds the rows of both references.

| Area | Rows | Count |
|---|---|---|
| arithmetic | gone: own defect | 1 |
| arithmetic | gone: reference defect | 59 |
| arithmetic | kept: deliberate | 106 |
| arithmetic | kept: own defect | 11 |
| arithmetic | kept: reference defect | 66 |
| arithmetic | new: own defect | 10 |
| integer | kept: reference defect | 621 |
| scope | kept: reference defect | 180 |
| types | gone: deliberate | 9 |
| types | gone: own defect | 3 |
| types | gone: reference defect | 130 |
| types | kept: own defect | 139 |
| types | kept: reference defect | 176 |
| types | new: own defect | 17 |
| types | new: reference defect | 12 |
| types | reclassified: reference defect → deliberate | 1 |
| fhir | gone: deliberate | 7 |
| fhir | gone: own defect | 6 |
| fhir | gone: reference defect | 22 |
| fhir | kept: deliberate | 433 |
| fhir | kept: own defect | 214 |
| fhir | kept: reference defect | 68 |
| fhir | new: reference defect | 1 |
| fhir | reclassified: reference defect → deliberate | 1 |
| codes | kept: reference defect | 157 |
| codesets | kept: reference defect | 83 |
| membership | gone: reference defect | 22 |
| membership | kept: own defect | 4 |
| fhirmembership | gone: reference defect | 1 |
| fhirmembership | kept: own defect | 21 |
| division | kept: deliberate | 1 |
| division | kept: reference defect | 5 |

| Area | Name | Reference | Before | Now | The reference's answer now |
|---|---|---|---|---|---|
| arithmetic | D114 | java | — | own defect | `null` |
| arithmetic | D115 | java | — | own defect | `null` |
| arithmetic | D12 | java | reference defect | gone | `null` |
| arithmetic | D128 | java | reference defect | gone | `{"type":"Decimal","value":"-2.0"}` |
| arithmetic | D143 | java | reference defect | gone | `null` |
| arithmetic | D157 | java | reference defect | gone | `null` |
| arithmetic | D159 | java | reference defect | gone | `null` |
| arithmetic | D17 | java | — | own defect | `null` |
| arithmetic | D188 | java | reference defect | gone | `null` |
| arithmetic | D192 | java | reference defect | gone | `null` |
| arithmetic | D212 | java | — | own defect | `null` |
| arithmetic | D224 | java | reference defect | gone | `null` |
| arithmetic | D235 | java | reference defect | gone | `null` |
| arithmetic | D257 | java | reference defect | gone | `{"type":"Decimal","value":"249955.002025"}` |
| arithmetic | D258 | java | reference defect | gone | `null` |
| arithmetic | D291 | java | reference defect | gone | `null` |
| arithmetic | D296 | java | reference defect | gone | `null` |
| arithmetic | D313 | java | reference defect | gone | `null` |
| arithmetic | D324 | java | reference defect | gone | `null` |
| arithmetic | D328 | java | reference defect | gone | `{"type":"Decimal","value":"-0.93827161"}` |
| arithmetic | D336 | java | reference defect | gone | `{"type":"Decimal","value":"1096.63315842"}` |
| arithmetic | D379 | java | — | own defect | `{"type":"Decimal","value":"1907346572495099527168"}` |
| arithmetic | D386 | java | reference defect | gone | `null` |
| arithmetic | D389 | java | reference defect | gone | `null` |
| arithmetic | D390 | java | reference defect | gone | `null` |
| arithmetic | D395 | java | reference defect | gone | `null` |
| arithmetic | D4 | java | own defect | gone | `{"type":"Decimal","value":"1.00000000"}` |
| arithmetic | D41 | java | reference defect | gone | `null` |
| arithmetic | D411 | java | reference defect | gone | `null` |
| arithmetic | D414 | java | reference defect | gone | `null` |
| arithmetic | D427 | java | — | own defect | `{"type":"Decimal","value":"26881171418161356094253400435962903554686976"}` |
| arithmetic | D43 | java | reference defect | gone | `null` |
| arithmetic | D436 | java | reference defect | gone | `null` |
| arithmetic | D464 | java | reference defect | gone | `null` |
| arithmetic | D466 | java | reference defect | gone | `null` |
| arithmetic | D469 | java | reference defect | gone | `null` |
| arithmetic | D473 | java | reference defect | gone | `null` |
| arithmetic | D476 | java | reference defect | gone | `null` |
| arithmetic | D477 | java | reference defect | gone | `null` |
| arithmetic | D502 | java | reference defect | gone | `null` |
| arithmetic | D507 | java | reference defect | gone | `null` |
| arithmetic | D511 | java | reference defect | gone | `null` |
| arithmetic | D553 | java | reference defect | gone | `null` |
| arithmetic | D565 | java | reference defect | gone | `{"type":"Decimal","value":"1096.63315842845850056619383394718170166015625"}` |
| arithmetic | D566 | java | reference defect | gone | `null` |
| arithmetic | D571 | java | reference defect | gone | `null` |
| arithmetic | D573 | java | — | own defect | `null` |
| arithmetic | D58 | java | reference defect | gone | `null` |
| arithmetic | D604 | java | reference defect | gone | `null` |
| arithmetic | D612 | java | reference defect | gone | `{"type":"Decimal","value":"4.5"}` |
| arithmetic | D614 | java | reference defect | gone | `null` |
| arithmetic | D629 | java | reference defect | gone | `null` |
| arithmetic | D631 | java | reference defect | gone | `null` |
| arithmetic | D651 | java | reference defect | gone | `{"type":"Decimal","value":"2.12132034"}` |
| arithmetic | D680 | java | reference defect | gone | `{"type":"Decimal","value":"8.12784588"}` |
| arithmetic | D681 | java | reference defect | gone | `null` |
| arithmetic | D683 | java | reference defect | gone | `null` |
| arithmetic | D698 | java | reference defect | gone | `null` |
| arithmetic | D699 | java | — | own defect | `{"type":"Decimal","value":"26881171149349812049819514673522596865114112"}` |
| arithmetic | D7 | java | — | own defect | `null` |
| arithmetic | D70 | java | reference defect | gone | `null` |
| arithmetic | D700 | java | reference defect | gone | `null` |
| arithmetic | D739 | java | reference defect | gone | `null` |
| arithmetic | D74 | java | reference defect | gone | `null` |
| arithmetic | D756 | java | reference defect | gone | `null` |
| arithmetic | D768 | java | reference defect | gone | `null` |
| arithmetic | D780 | java | — | own defect | `{"type":"Decimal","value":"26881171418161356094253400435962903554686976"}` |
| arithmetic | D81 | java | reference defect | gone | `null` |
| arithmetic | D87 | java | reference defect | gone | `{"type":"Decimal","value":"0.33333333"}` |
| arithmetic | D94 | java | reference defect | gone | `null` |
| types | D100 | java | — | reference defect | `{"error":"InvalidCast","message":"Cannot cast null to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D104 | java | — | own defect | `{"error":"InvalidCast","message":"Cannot cast {} to org.hl7.elm.r1.ListTypeSpecifier@…."}` |
| types | D112 | java | own defect | gone | `{"type":"Long","value":"5"}` |
| types | D115 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D12 | java | reference defect | gone | `{"type":"Interval","low":{"type":"String","value":"a"},"lowClosed":true,"high":{"type":"String","value":"b"},"highClosed":true}` |
| types | D128 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D129 | java | deliberate | gone | `{"type":"Boolean","value":false}` |
| types | D130 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D132 | java | — | own defect | `{"type":"Boolean","value":true}` |
| types | D139 | java | reference defect | gone | `{"type":"Time","value":"10:00:00","precision":"second"}` |
| types | D149 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast Interval['a', 'b'] to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D155 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast @2020-01-01T10:00:00.000+00:00 to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D16 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D162 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast {\n  1,\n  'a'\n} to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D168 | java | reference defect | gone | `{"type":"String","value":"a"}` |
| types | D171 | java | reference defect | gone | `{"type":"Time","value":"10:00:00","precision":"second"}` |
| types | D172 | java | deliberate | gone | `{"type":"Boolean","value":false}` |
| types | D176 | java | — | reference defect | `{"error":"InvalidCast","message":"Cannot cast null to org.hl7.elm.r1.ListTypeSpecifier@…."}` |
| types | D181 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D188 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D198 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D199 | java | — | reference defect | `{"error":"InvalidCast","message":"Cannot cast null to org.hl7.elm.r1.ListTypeSpecifier@…."}` |
| types | D200 | java | own defect | gone | `{"type":"Long","value":"5"}` |
| types | D205 | java | reference defect | gone | `{"type":"Date","value":"2020-01-01","precision":"day"}` |
| types | D208 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast 'a' to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D215 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D22 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast @T10:00:00 to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D221 | java | reference defect | gone | `{"type":"Interval","low":{"type":"String","value":"a"},"lowClosed":true,"high":{"type":"String","value":"b"},"highClosed":true}` |
| types | D222 | java | — | own defect | `null` |
| types | D223 | java | reference defect | gone | `{"type":"Decimal","value":"2.5"}` |
| types | D230 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D231 | java | — | reference defect | `{"type":"Boolean","value":true}` |
| types | D243 | java | — | own defect | `null` |
| types | D244 | java | — | own defect | `{"type":"Boolean","value":true}` |
| types | D245 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast @2020-01-01T10:00:00.000+00:00 to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D262 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D264 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D265 | java | deliberate | gone | `{"type":"Boolean","value":false}` |
| types | D28 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D289 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D292 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D297 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D3 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D30 | java | — | own defect | `{"type":"Boolean","value":true}` |
| types | D302 | java | reference defect | gone | `null` |
| types | D306 | java | reference defect | gone | `null` |
| types | D309 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast Tuple {\n  a: 1\n} to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D312 | java | — | reference defect | `{"error":"InvalidCast","message":"Cannot cast null to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D314 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D316 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D317 | java | deliberate | gone | `{"type":"Boolean","value":false}` |
| types | D33 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast Interval[1, null] to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D342 | java | reference defect | gone | `{"type":"Time","value":"10:00:00","precision":"second"}` |
| types | D349 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast {\n  1,\n  'a'\n} to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D355 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast {\n  'a'\n} to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D36 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast {\n  'a'\n} to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D370 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D371 | java | reference defect | gone | `{"type":"Interval","low":{"type":"Integer","value":"1"},"lowClosed":true,"high":null,"highClosed":true}` |
| types | D379 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast @2020-01-01 to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D383 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast @T10:00:00 to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D39 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D393 | java | reference defect | gone | `{"type":"Tuple","fields":{"a":{"type":"Integer","value":"1"}}}` |
| types | D40 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D401 | java | — | own defect | `{"error":"InvalidCast","message":"Cannot cast Interval[1, 2] to org.hl7.elm.r1.IntervalTypeSpecifier@…."}` |
| types | D408 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D425 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D426 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast true to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D429 | java | — | reference defect | `{"type":"Boolean","value":true}` |
| types | D43 | java | — | own defect | `{"error":"InvalidCast","message":"Cannot cast {\n  1,\n  'a'\n} to org.hl7.elm.r1.ListTypeSpecifier@…."}` |
| types | D431 | java | reference defect | gone | `{"type":"Interval","low":{"type":"String","value":"a"},"lowClosed":true,"high":{"type":"String","value":"b"},"highClosed":true}` |
| types | D432 | java | reference defect | gone | `{"type":"List","elements":[]}` |
| types | D435 | java | deliberate | gone | `{"type":"Boolean","value":false}` |
| types | D437 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D438 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D444 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D447 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast Code {\n  code: 'x',\n  system: null,\n  version: null,\n  display: null\n} to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D448 | java | own defect | gone | `{"type":"Long","value":"5"}` |
| types | D455 | java | — | own defect | `{"error":"InvalidCast","message":"Cannot cast {\n  1,\n  2\n} to org.hl7.elm.r1.ListTypeSpecifier@…."}` |
| types | D456 | java | — | own defect | `{"type":"Boolean","value":true}` |
| types | D462 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast Interval['a', 'b'] to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D464 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast @T10:00:00 to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D469 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D473 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D480 | java | reference defect | gone | `{"type":"Interval","low":{"type":"Integer","value":"1"},"lowClosed":true,"high":null,"highClosed":true}` |
| types | D482 | java | reference defect | gone | `{"type":"Code","code":"x","system":null,"version":null,"display":null}` |
| types | D483 | java | reference defect | deliberate | `{"type":"Boolean","value":true}` |
| types | D484 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D487 | java | reference defect | gone | `{"type":"Quantity","value":"5","unit":"mg"}` |
| types | D493 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast 5 'mg' to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D497 | java | — | reference defect | `{"type":"Boolean","value":false}` |
| types | D507 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D516 | java | — | own defect | `{"error":"InvalidCast","message":"Cannot cast Interval[1, null] to org.hl7.elm.r1.IntervalTypeSpecifier@…."}` |
| types | D521 | java | — | reference defect | `{"error":"InvalidCast","message":"Cannot cast {} to org.hl7.elm.r1.ListTypeSpecifier@…."}` |
| types | D532 | java | reference defect | gone | `{"type":"List","elements":[]}` |
| types | D533 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast 'a' to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D536 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D546 | java | — | own defect | `{"error":"InvalidCast","message":"Cannot cast {\n  'a'\n} to org.hl7.elm.r1.ListTypeSpecifier@…."}` |
| types | D547 | java | reference defect | gone | `{"type":"Interval","low":{"type":"Integer","value":"1"},"lowClosed":true,"high":{"type":"Integer","value":"2"},"highClosed":true}` |
| types | D550 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D561 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D569 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast 2.5 to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D577 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D578 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D6 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D607 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D608 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast {\n  1,\n  2\n} to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D610 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D616 | java | reference defect | gone | `{"type":"List","elements":[{"type":"String","value":"a"}]}` |
| types | D626 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast @2020-01-01T10:00:00.000+00:00 to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D63 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast Concept {\n  codes: {\n    Code {\n      code: 'x',\n      system: null,\n      version: null,\n      display: null\n    }\n  }\n} to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D631 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D637 | java | reference defect | gone | `{"type":"List","elements":[{"type":"Integer","value":"1"},{"type":"Integer","value":"2"}]}` |
| types | D644 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast Code {\n  code: 'x',\n  system: null,\n  version: null,\n  display: null\n} to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D652 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D657 | java | — | own defect | `null` |
| types | D659 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast true to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D676 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D698 | java | — | reference defect | `{"error":"InvalidCast","message":"Cannot cast {} to org.hl7.elm.r1.ListTypeSpecifier@…."}` |
| types | D699 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast Interval['a', 'b'] to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D70 | java | reference defect | gone | `{"type":"Code","code":"x","system":null,"version":null,"display":null}` |
| types | D704 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast 5 'mg' to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D707 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D71 | java | reference defect | gone | `{"type":"List","elements":[{"type":"Integer","value":"1"},{"type":"String","value":"a"}]}` |
| types | D715 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D716 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast @T10:00:00 to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D725 | java | — | reference defect | `{"type":"Boolean","value":false}` |
| types | D737 | java | reference defect | gone | `{"type":"Date","value":"2020-01-01","precision":"day"}` |
| types | D740 | java | reference defect | gone | `{"type":"Time","value":"10:00:00","precision":"second"}` |
| types | D746 | java | — | reference defect | `{"error":"InvalidCast","message":"Cannot cast null to org.hl7.elm.r1.IntervalTypeSpecifier@…."}` |
| types | D75 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast {} to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D752 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast 5 'mg' to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D758 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast Interval['a', 'b'] to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D763 | java | — | own defect | `{"type":"Boolean","value":true}` |
| types | D764 | java | reference defect | gone | `{"type":"Interval","low":{"type":"Integer","value":"1"},"lowClosed":true,"high":{"type":"Integer","value":"2"},"highClosed":true}` |
| types | D768 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast {\n  1,\n  'a'\n} to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D773 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D774 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D775 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast Concept {\n  codes: {\n    Code {\n      code: 'x',\n      system: null,\n      version: null,\n      display: null\n    }\n  }\n} to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D777 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast 2.5 to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D778 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D780 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast 5 to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D79 | java | reference defect | gone | `{"type":"Code","code":"x","system":null,"version":null,"display":null}` |
| types | D791 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D806 | java | reference defect | gone | `{"type":"List","elements":[{"type":"String","value":"a"}]}` |
| types | D812 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D816 | java | — | own defect | `null` |
| types | D823 | java | — | own defect | `null` |
| types | D827 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D83 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D834 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D836 | java | reference defect | gone | `{"type":"Tuple","fields":{"a":{"type":"Integer","value":"1"}}}` |
| types | D844 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D845 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D846 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D847 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast Concept {\n  codes: {\n    Code {\n      code: 'x',\n      system: null,\n      version: null,\n      display: null\n    }\n  }\n} to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D855 | java | — | own defect | `null` |
| types | D857 | java | deliberate | gone | `{"type":"Boolean","value":false}` |
| types | D86 | java | — | reference defect | `{"error":"InvalidCast","message":"Cannot cast null to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D860 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D862 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| types | D864 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D865 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast Concept {\n  codes: {\n    Code {\n      code: 'x',\n      system: null,\n      version: null,\n      display: null\n    }\n  }\n} to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D870 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast 2.5 to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D878 | java | reference defect | gone | `{"type":"Date","value":"2020-01-01","precision":"day"}` |
| types | D881 | java | deliberate | gone | `{"type":"Boolean","value":false}` |
| types | D885 | java | reference defect | gone | `{"error":"InvalidCast","message":"Cannot cast @2020-01-01 to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| types | D888 | java | reference defect | gone | `{"type":"List","elements":[{"type":"String","value":"a"}]}` |
| types | D896 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D898 | java | reference defect | gone | `{"type":"List","elements":[{"type":"Integer","value":"1"},{"type":"Integer","value":"2"}]}` |
| types | D90 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| types | D92 | java | deliberate | gone | `{"type":"Boolean","value":false}` |
| types | D99 | java | deliberate | gone | `{"type":"Boolean","value":false}` |
| fhir | D117 | java | deliberate | gone | `{"type":"Boolean","value":false}` |
| fhir | D132 | java | reference defect | gone | `null` |
| fhir | D134 | java | deliberate | gone | `{"type":"Boolean","value":false}` |
| fhir | D149 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| fhir | D190 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| fhir | D2 | java | reference defect | gone | `{"type":"Quantity","value":"5","unit":"mg"}` |
| fhir | D207 | java | own defect | gone | `{"type":"Boolean","value":false}` |
| fhir | D223 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| fhir | D240 | java | own defect | gone | `null` |
| fhir | D244 | java | deliberate | gone | `{"type":"Boolean","value":false}` |
| fhir | D28 | java | own defect | gone | `{"type":"FHIR.AdministrativeGender","value":"male"}` |
| fhir | D324 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| fhir | D326 | java | deliberate | gone | `{"type":"Boolean","value":false}` |
| fhir | D438 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| fhir | D443 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| fhir | D49 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| fhir | D494 | java | reference defect | gone | `null` |
| fhir | D516 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| fhir | D530 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| fhir | D577 | java | deliberate | gone | `{"type":"Boolean","value":false}` |
| fhir | D583 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| fhir | D585 | java | — | reference defect | `{"error":"InvalidCast","message":"Cannot cast null to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| fhir | D593 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| fhir | D619 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| fhir | D663 | java | reference defect | gone | `{"type":"FHIR.Period","value":"{\"start\":\"2020-01-01\",\"end\":\"2020-01-03T10:00:00Z\"}"}` |
| fhir | D679 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| fhir | D683 | java | reference defect | gone | `{"type":"Boolean","value":true}` |
| fhir | D688 | java | reference defect | deliberate | `{"type":"Boolean","value":true}` |
| fhir | D70 | java | own defect | gone | `{"type":"FHIR.integer","value":"7"}` |
| fhir | D739 | java | deliberate | gone | `{"type":"Boolean","value":false}` |
| fhir | D773 | java | reference defect | gone | `{"type":"Boolean","value":false}` |
| fhir | D777 | java | reference defect | gone | `{"type":"Quantity","value":"5","unit":"mg"}` |
| fhir | D787 | java | own defect | gone | `{"type":"FHIR.id","value":"p"}` |
| fhir | D797 | java | reference defect | gone | `{"type":"List","elements":[{"type":"Integer","value":"1"},{"type":"Integer","value":"2"}]}` |
| fhir | D84 | java | deliberate | gone | `{"type":"Boolean","value":false}` |
| fhir | D859 | java | reference defect | gone | `{"type":"FHIR.Period","value":"{\"start\":\"2020-01-01\"}"}` |
| fhir | D88 | java | own defect | gone | `{"error":"InvalidCast","message":"Cannot cast FHIR.AdministrativeGender {\n  extension: null,\n  id: null,\n  value: 'male'\n} to org.hl7.elm.r1.NamedTypeSpecifier@…."}` |
| membership | D11 | java | reference defect | gone | `null` |
| membership | D129 | java | reference defect | gone | `null` |
| membership | D131 | java | reference defect | gone | `null` |
| membership | D14 | java | reference defect | gone | `null` |
| membership | D149 | java | reference defect | gone | `null` |
| membership | D164 | java | reference defect | gone | `null` |
| membership | D170 | java | reference defect | gone | `null` |
| membership | D180 | java | reference defect | gone | `null` |
| membership | D214 | java | reference defect | gone | `null` |
| membership | D23 | java | reference defect | gone | `null` |
| membership | D265 | java | reference defect | gone | `null` |
| membership | D279 | java | reference defect | gone | `null` |
| membership | D322 | java | reference defect | gone | `null` |
| membership | D331 | java | reference defect | gone | `null` |
| membership | D34 | java | reference defect | gone | `null` |
| membership | D349 | java | reference defect | gone | `null` |
| membership | D380 | java | reference defect | gone | `null` |
| membership | D395 | java | reference defect | gone | `null` |
| membership | D71 | java | reference defect | gone | `null` |
| membership | D72 | java | reference defect | gone | `null` |
| membership | D88 | java | reference defect | gone | `null` |
| membership | D97 | java | reference defect | gone | `null` |
| fhirmembership | D163 | java | reference defect | gone | `null` |
