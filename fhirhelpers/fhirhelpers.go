// Package fhirhelpers provides the FHIRHelpers CQL library the engine falls back
// to when an include of it is not satisfied by the caller's LibraryResolver.
package fhirhelpers

import _ "embed"

// Source is the official FHIRHelpers 4.0.1 published by cqframework, embedded
// verbatim from quick/src/main/resources/org/hl7/fhir in
// cqframework/clinical_quality_language.
//
// It is carried rather than translated on purpose. A hand-maintained subset
// means owning the divergence forever: every upstream change becomes a manual
// merge, and the errors do not fail loudly — they return empty lists. The
// version this replaced had eight identity functions and no ToCode, ToConcept,
// ToInterval or ToRatio at all, so the four conversions that turn FHIR data into
// CQL system types were simply missing.
//
//go:embed FHIRHelpers-4.0.1.cql
var Source string
