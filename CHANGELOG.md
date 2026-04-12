# Changelog

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

## 1.1.0 (2026-04-04)

### Features

* add CQL conformance test suite — 1731/1731 (100%) official cqframework/cql-tests passing
* add DateTime/Date/Time arithmetic with Quantity operands (`DateTime(2005, 10, 10) + 5 years`)
* add math functions: Round, Floor, Ceiling, Truncate, Ln, Log, Exp, Power, Precision, HighBoundary, LowBoundary
* add Quantity arithmetic (add, subtract, multiply, divide between quantities)
* add string functions: Indexer, Concatenate, and string `+` concatenation
* add conversion functions: ToDateTime, ToTime, ToBoolean, ToQuantity, ToConcept
* add precision-aware temporal comparisons (same as, before, after, same or before, same or after)
* add Message function
* add multi-source query support (cartesian product)
* add Interval Expand for Integer, Decimal, DateTime, Date, Time, and Quantity types
* add aggregate clause support with custom accumulators

### Bug Fixes

* fix source/operands dispatch for standalone function calls (`Floor(1.0)` now works)
* fix null propagation in interval operations (null bounds = unbounded)
* fix three-valued logic in list membership and interval containment
* fix DateTime constructor to produce precision-aware values
* fix DurationBetween/DifferenceBetween for partial date/time precision
* fix Quantity comparison with Decimal values
* fix list Equivalent operator (order-independent)
* fix successor/predecessor for DateTime, Date, Time types
* fix overlaps before/after with open boundaries
* fix interval meets with temporal types
* fix Count to skip null elements
* fix StdDev/PopulationStdDev rounding

## 1.0.0 (2026-03-18)


### Features

* initial commit — CQL engine for FHIR R4 ([5a5f8d7](https://github.com/gofhir/cql/commit/5a5f8d713507b558ed5f7349a5518a822cdb5917))


### Bug Fixes

* **ci:** use golangci-lint-action v7 for golangci-lint v2 config ([d9749bf](https://github.com/gofhir/cql/commit/d9749bfb8ad13e143a998bf18ae91901d28413f7))
