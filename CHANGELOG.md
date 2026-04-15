# Changelog

## [1.5.0](https://github.com/gofhir/cql/compare/v1.4.0...v1.5.0) (2026-04-15)


### Features

* add DateTime/Date/Time and Quantity arithmetic support ([f06ce74](https://github.com/gofhir/cql/commit/f06ce74fa7366e8de177f990625b3c1db89ef015))
* add LibraryResolver option and include resolution ([c54aa56](https://github.com/gofhir/cql/commit/c54aa567d57e348d29b2f2df93ef0d606a302d23))
* add math functions (Round, Floor, Ceiling, Truncate, Ln, Log, Exp, Power, Precision, HighBoundary, LowBoundary) ([d2462af](https://github.com/gofhir/cql/commit/d2462af8e62cfea7a5f8659502e5662af6c48a74))
* add missing string, conversion, and quantity comparison functions ([3dd07ae](https://github.com/gofhir/cql/commit/3dd07ae73dd4cb6b9f74f88239cd9442e17f0b5c))
* built-in FHIRHelpers stub with passthrough functions ([742c3bc](https://github.com/gofhir/cql/commit/742c3bc5c61e965fd9344bef33972f7ffd19fe5f))
* **conformance:** add cql-tests runner with baseline results (791/1731 = 45.7%) ([957cf3f](https://github.com/gofhir/cql/commit/957cf3f99cbb7d6a1a5d72c3765d576ff5de81b6))
* **conformance:** add output parser for CQL test expected values ([1a41a09](https://github.com/gofhir/cql/commit/1a41a0967b4d9c4ad180a7aedcf9352400dc4d68))
* **conformance:** add test runner for cqframework/cql-tests ([9ec8c60](https://github.com/gofhir/cql/commit/9ec8c606deb1527784ad098dbb33c7f3323845e9))
* **conformance:** add XML model structs for cql-tests format ([cd8bc7e](https://github.com/gofhir/cql/commit/cd8bc7ee717e876b0bbb3e41a65329c08bfbc865))
* **cql:** add AnyInValueSet and AnyInCodeSystem operators ([6953a03](https://github.com/gofhir/cql/commit/6953a039123ce594774dee4a70874fd4658e72d1))
* **cql:** add ConvertQuantity/CanConvertQuantity with pluggable QuantityConverter interface ([c193241](https://github.com/gofhir/cql/commit/c19324164a9fb2c400b0c4cd6c1d97dd8223659f))
* **cql:** add ConvertsTo* predicates (Decimal, Long, Quantity, Date, DateTime, Time, Ratio) ([a978590](https://github.com/gofhir/cql/commit/a978590f294a70a6b8f640f6c67ed833f18b12fb))
* **cql:** add ConvertsToBoolean, ConvertsToString, ConvertsToInteger predicates ([56a439f](https://github.com/gofhir/cql/commit/56a439fda28f662366d694e4d30baf3f3403b8de))
* **cql:** add ExpandValueSet operator with optional ValueSetExpander interface ([cbbab8b](https://github.com/gofhir/cql/commit/cbbab8b46e49191b01037917e031d379956af82c))
* **cql:** add LibraryLoader for cross-library include resolution with recursion guard ([08b8d36](https://github.com/gofhir/cql/commit/08b8d36156a0b8a64ace9714999263bfcfd9e46f))
* **cql:** add SubList and SplitOnMatches operators ([31a599f](https://github.com/gofhir/cql/commit/31a599fae863787330de48d4b0cf8f874c0b48ea))
* **cql:** add Subsumes/SubsumedBy with optional SubsumesChecker interface ([a609423](https://github.com/gofhir/cql/commit/a6094237a87fd5951bda00bd8004355c45929353))
* **cql:** add ToLong and Children operators ([1e98532](https://github.com/gofhir/cql/commit/1e98532c3c8da2af4db6cdc9601ff58f28a004d4))
* **engine:** add WithQuantityConverter option for UCUM unit conversions ([31e3393](https://github.com/gofhir/cql/commit/31e33930dac6fe367915cbc1a1e3b7084c45c7e7))
* FHIRHelpers compliance — library resolver, value[x] resolution, FHIR primitive wrapping ([e491b02](https://github.com/gofhir/cql/commit/e491b020afa1d5db48b71c4ad15afe03d7145a65))
* implement precision-aware temporal comparisons and fix DateTime operations ([7b735db](https://github.com/gofhir/cql/commit/7b735dbf92efe6a3be9e627e65e8d8a5f36a902e))
* initial commit — CQL engine for FHIR R4 ([5a5f8d7](https://github.com/gofhir/cql/commit/5a5f8d713507b558ed5f7349a5518a822cdb5917))
* library-qualified function dispatch for included libraries ([b20b6ac](https://github.com/gofhir/cql/commit/b20b6acd49bbcf12b599a780c904b826ee2349f9))
* support function overloads in evaluator ([3318d60](https://github.com/gofhir/cql/commit/3318d60d81195e48f44bc8645302b866be3c52f1))
* type-aware overload resolution for function dispatch ([21fdd69](https://github.com/gofhir/cql/commit/21fdd6971cdcaedb71ecdde50c10c628486f7cde))
* value[x] choice type resolution in member access ([f779c76](https://github.com/gofhir/cql/commit/f779c76b82dd7c90e865f703de5cd68f9df4b107))


### Bug Fixes

* **ci:** use golangci-lint-action v7 for golangci-lint v2 config ([d9749bf](https://github.com/gofhir/cql/commit/d9749bfb8ad13e143a998bf18ae91901d28413f7))
* resolve final conformance gaps (multi-source queries, time precision, output parser) ([cd6d4f2](https://github.com/gofhir/cql/commit/cd6d4f26257c9e979bd6ffc48e2d3bd10be0acd9))
* resolve final conformance gaps (time expand, null intervals, string concat, validation) ([be58ab4](https://github.com/gofhir/cql/commit/be58ab4bc6d36ae83bfd6cedd16b61c585de35c5))
* resolve remaining conformance failures (Long literals, null intervals, ambiguous comparisons) ([b9f7104](https://github.com/gofhir/cql/commit/b9f710414d5b7beb05090f851ef040263861d9e9))
* resolve remaining conformance gaps (expand, list ops, string concat, edge cases) ([fb090a5](https://github.com/gofhir/cql/commit/fb090a54dc5b37922fd15fe11b7f5d180de55483))
* resolve source/operands dispatch and fix remaining conformance failures ([2809754](https://github.com/gofhir/cql/commit/2809754594108e17dc55780a1aec401690e3d4db))

## [1.4.0](https://github.com/gofhir/cql/compare/v1.3.0...v1.4.0) (2026-04-15)


### Features

* **engine:** add WithQuantityConverter option for UCUM unit conversions ([31e3393](https://github.com/gofhir/cql/commit/31e33930dac6fe367915cbc1a1e3b7084c45c7e7))

## [1.3.0](https://github.com/gofhir/cql/compare/v1.2.0...v1.3.0) (2026-04-15)


### Features

* **cql:** add AnyInValueSet and AnyInCodeSystem operators ([6953a03](https://github.com/gofhir/cql/commit/6953a039123ce594774dee4a70874fd4658e72d1))
* **cql:** add ConvertQuantity/CanConvertQuantity with pluggable QuantityConverter interface ([c193241](https://github.com/gofhir/cql/commit/c19324164a9fb2c400b0c4cd6c1d97dd8223659f))
* **cql:** add ConvertsTo* predicates (Decimal, Long, Quantity, Date, DateTime, Time, Ratio) ([a978590](https://github.com/gofhir/cql/commit/a978590f294a70a6b8f640f6c67ed833f18b12fb))
* **cql:** add ConvertsToBoolean, ConvertsToString, ConvertsToInteger predicates ([56a439f](https://github.com/gofhir/cql/commit/56a439fda28f662366d694e4d30baf3f3403b8de))
* **cql:** add ExpandValueSet operator with optional ValueSetExpander interface ([cbbab8b](https://github.com/gofhir/cql/commit/cbbab8b46e49191b01037917e031d379956af82c))
* **cql:** add LibraryLoader for cross-library include resolution with recursion guard ([08b8d36](https://github.com/gofhir/cql/commit/08b8d36156a0b8a64ace9714999263bfcfd9e46f))
* **cql:** add SubList and SplitOnMatches operators ([31a599f](https://github.com/gofhir/cql/commit/31a599fae863787330de48d4b0cf8f874c0b48ea))
* **cql:** add Subsumes/SubsumedBy with optional SubsumesChecker interface ([a609423](https://github.com/gofhir/cql/commit/a6094237a87fd5951bda00bd8004355c45929353))
* **cql:** add ToLong and Children operators ([1e98532](https://github.com/gofhir/cql/commit/1e98532c3c8da2af4db6cdc9601ff58f28a004d4))

## 1.3.0 (2026-04-14)

### Features

* add 10 `ConvertsTo*` safe conversion predicates (Boolean, String, Integer, Decimal, Long, Quantity, Date, DateTime, Time, Ratio)
* add `AnyInValueSet` and `AnyInCodeSystem` terminology operators
* add `Subsumes`/`SubsumedBy` with optional `SubsumesChecker` interface
* add `ExpandValueSet` operator with optional `ValueSetExpander` interface
* add `ToLong` and `Children` operators
* add `SubList` and `SplitOnMatches` operators
* add `ConvertQuantity`/`CanConvertQuantity` with pluggable `QuantityConverter` interface
* add `LibraryLoader` interface for cross-library include resolution with recursion guard

### Refactoring

* consolidate evalIncludedFunction into evalUserFunction

## [1.2.0](https://github.com/gofhir/cql/compare/v1.1.0...v1.2.0) (2026-04-12)


### Features

* add LibraryResolver option and include resolution ([c54aa56](https://github.com/gofhir/cql/commit/c54aa567d57e348d29b2f2df93ef0d606a302d23))
* built-in FHIRHelpers stub with passthrough functions ([742c3bc](https://github.com/gofhir/cql/commit/742c3bc5c61e965fd9344bef33972f7ffd19fe5f))
* FHIRHelpers compliance — library resolver, value[x] resolution, FHIR primitive wrapping ([e491b02](https://github.com/gofhir/cql/commit/e491b020afa1d5db48b71c4ad15afe03d7145a65))
* library-qualified function dispatch for included libraries ([b20b6ac](https://github.com/gofhir/cql/commit/b20b6acd49bbcf12b599a780c904b826ee2349f9))
* support function overloads in evaluator ([3318d60](https://github.com/gofhir/cql/commit/3318d60d81195e48f44bc8645302b866be3c52f1))
* type-aware overload resolution for function dispatch ([21fdd69](https://github.com/gofhir/cql/commit/21fdd6971cdcaedb71ecdde50c10c628486f7cde))
* value[x] choice type resolution in member access ([f779c76](https://github.com/gofhir/cql/commit/f779c76b82dd7c90e865f703de5cd68f9df4b107))

## [1.1.0](https://github.com/gofhir/cql/compare/v1.0.0...v1.1.0) (2026-04-05)


### Features

* add DateTime/Date/Time and Quantity arithmetic support ([f06ce74](https://github.com/gofhir/cql/commit/f06ce74fa7366e8de177f990625b3c1db89ef015))
* add math functions (Round, Floor, Ceiling, Truncate, Ln, Log, Exp, Power, Precision, HighBoundary, LowBoundary) ([d2462af](https://github.com/gofhir/cql/commit/d2462af8e62cfea7a5f8659502e5662af6c48a74))
* add missing string, conversion, and quantity comparison functions ([3dd07ae](https://github.com/gofhir/cql/commit/3dd07ae73dd4cb6b9f74f88239cd9442e17f0b5c))
* **conformance:** add cql-tests runner with baseline results (791/1731 = 45.7%) ([957cf3f](https://github.com/gofhir/cql/commit/957cf3f99cbb7d6a1a5d72c3765d576ff5de81b6))
* **conformance:** add output parser for CQL test expected values ([1a41a09](https://github.com/gofhir/cql/commit/1a41a0967b4d9c4ad180a7aedcf9352400dc4d68))
* **conformance:** add test runner for cqframework/cql-tests ([9ec8c60](https://github.com/gofhir/cql/commit/9ec8c606deb1527784ad098dbb33c7f3323845e9))
* **conformance:** add XML model structs for cql-tests format ([cd8bc7e](https://github.com/gofhir/cql/commit/cd8bc7ee717e876b0bbb3e41a65329c08bfbc865))
* implement precision-aware temporal comparisons and fix DateTime operations ([7b735db](https://github.com/gofhir/cql/commit/7b735dbf92efe6a3be9e627e65e8d8a5f36a902e))


### Bug Fixes

* resolve final conformance gaps (multi-source queries, time precision, output parser) ([cd6d4f2](https://github.com/gofhir/cql/commit/cd6d4f26257c9e979bd6ffc48e2d3bd10be0acd9))
* resolve final conformance gaps (time expand, null intervals, string concat, validation) ([be58ab4](https://github.com/gofhir/cql/commit/be58ab4bc6d36ae83bfd6cedd16b61c585de35c5))
* resolve remaining conformance failures (Long literals, null intervals, ambiguous comparisons) ([b9f7104](https://github.com/gofhir/cql/commit/b9f710414d5b7beb05090f851ef040263861d9e9))
* resolve remaining conformance gaps (expand, list ops, string concat, edge cases) ([fb090a5](https://github.com/gofhir/cql/commit/fb090a54dc5b37922fd15fe11b7f5d180de55483))
* resolve source/operands dispatch and fix remaining conformance failures ([2809754](https://github.com/gofhir/cql/commit/2809754594108e17dc55780a1aec401690e3d4db))

## 1.0.0 (2026-03-18)


### Features

* initial commit — CQL engine for FHIR R4 ([5a5f8d7](https://github.com/gofhir/cql/commit/5a5f8d713507b558ed5f7349a5518a822cdb5917))


### Bug Fixes

* **ci:** use golangci-lint-action v7 for golangci-lint v2 config ([d9749bf](https://github.com/gofhir/cql/commit/d9749bfb8ad13e143a998bf18ae91901d28413f7))
