# Changelog

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
