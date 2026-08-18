# Changelog

## [1.17.0](https://github.com/gofhir/cql/compare/v1.16.0...v1.17.0) (2026-08-18)


### Features

* **sema:** warn that a concrete choice name is not portable ([#49](https://github.com/gofhir/cql/issues/49)) ([ac43126](https://github.com/gofhir/cql/commit/ac431260f6135b33243f5fc0665e8a6c8390026b))

## [1.16.0](https://github.com/gofhir/cql/compare/v1.15.2...v1.16.0) (2026-08-18)


### Features

* **referenceharness:** fold the reference automatically, and probe 32 definitions ([#47](https://github.com/gofhir/cql/issues/47)) ([ee4b952](https://github.com/gofhir/cql/commit/ee4b9524e74b4f0cdc276739f3f6fd04acc3161f))

## [1.15.2](https://github.com/gofhir/cql/compare/v1.15.1...v1.15.2) (2026-08-18)


### Bug Fixes

* **types:** decide interval equality by Start and End ([#45](https://github.com/gofhir/cql/issues/45)) ([ffea8a1](https://github.com/gofhir/cql/commit/ffea8a16dd7c5303dfdb403279de7ba69cd97b11))

## [1.15.1](https://github.com/gofhir/cql/compare/v1.15.0...v1.15.1) (2026-08-18)


### Bug Fixes

* **sema:** spell a qualified type name one way ([#43](https://github.com/gofhir/cql/issues/43)) ([20aae0e](https://github.com/gofhir/cql/commit/20aae0e7c906c493c0cfe4d53ced95b63fdc2d70))

## [1.15.0](https://github.com/gofhir/cql/compare/v1.14.1...v1.15.0) (2026-08-18)


### Features

* **engine:** make the published measures check out, and validate by default ([#41](https://github.com/gofhir/cql/issues/41)) ([d62d959](https://github.com/gofhir/cql/commit/d62d9597d3c9159f4485ff16f73207a06f55de20))

## [1.14.1](https://github.com/gofhir/cql/compare/v1.14.0...v1.14.1) (2026-08-17)


### Bug Fixes

* **compiler:** unquote delimited type names, and stop validating by default ([#39](https://github.com/gofhir/cql/issues/39)) ([e1876cc](https://github.com/gofhir/cql/commit/e1876ccdc012f089162d8c802d78ff112d4b7f01))

## [1.14.0](https://github.com/gofhir/cql/compare/v1.13.2...v1.14.0) (2026-08-17)


### Features

* **engine:** refuse to evaluate a library whose meaning does not check out ([#37](https://github.com/gofhir/cql/issues/37)) ([99692cf](https://github.com/gofhir/cql/commit/99692cf3734d39b24f4b3ad643e549e929405b39))

## [1.13.2](https://github.com/gofhir/cql/compare/v1.13.1...v1.13.2) (2026-08-17)


### Bug Fixes

* **sema:** stop reporting included libraries and choice elements as mistakes ([#35](https://github.com/gofhir/cql/issues/35)) ([b77ceb0](https://github.com/gofhir/cql/commit/b77ceb0eec311d56261a49d3b83822b4726d22d5))

## [1.13.1](https://github.com/gofhir/cql/compare/v1.13.0...v1.13.1) (2026-08-16)


### Bug Fixes

* **eval:** teach the aggregates what a Quantity is ([#33](https://github.com/gofhir/cql/issues/33)) ([cba41a1](https://github.com/gofhir/cql/commit/cba41a11fffb618d75ea61961eee4d51202c4400))

## [1.13.0](https://github.com/gofhir/cql/compare/v1.12.0...v1.13.0) (2026-08-16)


### Features

* **compiler:** push a query's date filter down to the retrieve it reads from ([#31](https://github.com/gofhir/cql/issues/31)) ([b9803e3](https://github.com/gofhir/cql/commit/b9803e3af75ce309847fa39252086c9e9bcc7a12))
* **eval:** say which subject a retrieve means, and stop when it cannot ([#30](https://github.com/gofhir/cql/issues/30)) ([c24342c](https://github.com/gofhir/cql/commit/c24342c188a29dd20e2ec046c6ab6dcaecf5496f))

## [1.12.0](https://github.com/gofhir/cql/compare/v1.11.0...v1.12.0) (2026-08-15)


### Features

* **eval:** apply the conversions the semantic phase decided on, instead of deciding again ([#28](https://github.com/gofhir/cql/issues/28)) ([aea0f45](https://github.com/gofhir/cql/commit/aea0f4514f63006bcbc57a70a5ad2608db9a4508))

## [1.11.0](https://github.com/gofhir/cql/compare/v1.10.0...v1.11.0) (2026-08-15)


### Features

* **sema:** work out what every expression is before anything is evaluated ([#26](https://github.com/gofhir/cql/issues/26)) ([c2b9cc5](https://github.com/gofhir/cql/commit/c2b9cc55f97ee6555dadbe7acf23b8dfc192de67))

## [1.10.0](https://github.com/gofhir/cql/compare/v1.9.0...v1.10.0) (2026-08-13)


### Features

* **ast:** record where each expression came from, and diff against the reference ([#24](https://github.com/gofhir/cql/issues/24)) ([c5f0a6e](https://github.com/gofhir/cql/commit/c5f0a6edb1311bd5b7dd987b9348e9e475d036f2))

## [1.9.0](https://github.com/gofhir/cql/compare/v1.8.0...v1.9.0) (2026-08-13)


### Features

* **engine:** reusable parsed libraries, an injectable clock, and provenance ([#23](https://github.com/gofhir/cql/issues/23)) ([73ea8eb](https://github.com/gofhir/cql/commit/73ea8eb42c66d16ac2873a4eae20e35082dd040c))
* **eval:** convert FHIR values where an operator needs a system type ([#21](https://github.com/gofhir/cql/issues/21)) ([4093b93](https://github.com/gofhir/cql/commit/4093b93b97373a4a8932f1e40e603a14070377ab))

## [1.8.0](https://github.com/gofhir/cql/compare/v1.7.0...v1.8.0) (2026-08-13)


### Features

* **model:** carry the official FHIR ModelInfo, and consult it ([#19](https://github.com/gofhir/cql/issues/19)) ([e90b4ef](https://github.com/gofhir/cql/commit/e90b4ef1150794061bc916a6274ffba17031f0f4))

## [1.7.0](https://github.com/gofhir/cql/compare/v1.6.0...v1.7.0) (2026-08-13)


### Features

* **eval:** adopt the official FHIRHelpers, and make it work ([#17](https://github.com/gofhir/cql/issues/17)) ([7cf47cb](https://github.com/gofhir/cql/commit/7cf47cbc3f85622193286a5af21d3c51e0884696))

## [1.6.0](https://github.com/gofhir/cql/compare/v1.5.1...v1.6.0) (2026-08-12)


### Features

* **eval:** align operator semantics with the current conformance suite ([#12](https://github.com/gofhir/cql/issues/12)) ([6b1d01f](https://github.com/gofhir/cql/commit/6b1d01fcce1ccf3c61c52868350332ef04809cfe))


### Bug Fixes

* **compiler:** decide keywords from the token stream, not the node text ([#15](https://github.com/gofhir/cql/issues/15)) ([cfdc840](https://github.com/gofhir/cql/commit/cfdc8407b84f95189b7d44121a859b2dbd57663b))
* **eval:** enforce the limits the engine already advertised ([#16](https://github.com/gofhir/cql/issues/16)) ([19c90df](https://github.com/gofhir/cql/commit/19c90df343bc215241b08bfa5e2ce81146f04af0))

## [1.5.1](https://github.com/gofhir/cql/compare/v1.5.0...v1.5.1) (2026-08-10)


### Bug Fixes

* **eval:** answer null consistently when temporal precision leaves a comparison unknown ([#11](https://github.com/gofhir/cql/issues/11)) ([426e0be](https://github.com/gofhir/cql/commit/426e0be51c055015d37ef072b92db13b7eab5a42))
* **funcs:** honor UCUM duration units in temporal arithmetic ([#9](https://github.com/gofhir/cql/issues/9)) ([20cebc6](https://github.com/gofhir/cql/commit/20cebc63a3dedfed428f73705f0cea975eb9166a))

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
