# Drift 2026-10-22: the environment of the references

Java 3.26.0 and cql-execution 3.3.2 with cql-exec-fhir 2.1.6 in both sets. Only the environment moves: maven:3-eclipse-temurin-21 → maven:3.10.0-eclipse-temurin-25, node:20 → node:24.22.1, slf4j-nop 2.0.13 → 2.0.20. Old set: the references at 1347eb4; new set: `make references-write` on top, with the manifests rewritten by `-update-divergences` and the classifiers. Written by

```sh
go run ./internal/referenceharness/differential/drift -old <1347eb4> -new testdata/snapshots [-ref js] -divergences arithmetic integer scope types fhir codes codesets membership fhirmembership division
```

## Java

| Area | same | message only | error class changed | error → value | value → error | value changed | refused → accepted | accepted → refused | added | removed |
|---|---|---|---|---|---|---|---|---|---|---|
| arithmetic | 784 | 16 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| integer | 1000 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| scope | 600 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| types | 900 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| fhir | 900 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| codes | 600 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| codesets | 400 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| membership | 400 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| fhirmembership | 200 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| division | 29 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| all | 5813 | 16 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |

| Area | Rows | Count |
|---|---|---|
| arithmetic | kept: deliberate | 106 |
| arithmetic | kept: own defect | 12 |
| arithmetic | kept: reference defect | 125 |
| integer | kept: reference defect | 621 |
| scope | kept: reference defect | 180 |
| types | kept: deliberate | 9 |
| types | kept: own defect | 142 |
| types | kept: reference defect | 307 |
| fhir | kept: deliberate | 440 |
| fhir | kept: own defect | 220 |
| fhir | kept: reference defect | 91 |
| codes | kept: reference defect | 157 |
| codesets | kept: reference defect | 83 |
| membership | kept: own defect | 4 |
| membership | kept: reference defect | 22 |
| fhirmembership | kept: own defect | 21 |
| fhirmembership | kept: reference defect | 1 |
| division | kept: deliberate | 1 |
| division | kept: reference defect | 5 |

| Area | Name | Reference | Before | Now | The reference's answer now |
|---|---|---|---|---|---|

## cql-execution

| Area | same | message only | error class changed | error → value | value → error | value changed | refused → accepted | accepted → refused | added | removed |
|---|---|---|---|---|---|---|---|---|---|---|
| arithmetic | 798 | 0 | 0 | 0 | 0 | 2 | 0 | 0 | 0 | 0 |
| integer | 1000 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| scope | 600 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| types | 900 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| fhir | 326 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| codes | 600 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| codesets | 400 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| membership | 400 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| fhirmembership | 179 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| division | 29 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| all | 5232 | 0 | 0 | 0 | 0 | 2 | 0 | 0 | 0 | 0 |
