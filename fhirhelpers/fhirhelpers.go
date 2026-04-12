// Package fhirhelpers provides a built-in FHIRHelpers CQL library
// that works with raw JSON values (no FHIR primitive wrapping).
package fhirhelpers

// Source is the CQL source for the built-in FHIRHelpers 4.0.1.
// This is a simplified version that passes through raw values,
// since the Go engine accesses JSON fields directly without FHIR primitive wrapping.
const Source = `library FHIRHelpers version '4.0.1'

using FHIR version '4.0.1'

define function ToBoolean(value Boolean): value
define function ToString(value String): value
define function ToInteger(value Integer): value
define function ToDecimal(value Decimal): value
define function ToDateTime(value DateTime): value
define function ToDate(value Date): value
define function ToTime(value Time): value
define function ToQuantity(quantity Quantity): quantity
`
