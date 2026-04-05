# go-cql

[![CI](https://github.com/gofhir/cql/actions/workflows/ci.yml/badge.svg)](https://github.com/gofhir/cql/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/gofhir/cql.svg)](https://pkg.go.dev/github.com/gofhir/cql)
[![codecov](https://codecov.io/gh/gofhir/cql/branch/main/graph/badge.svg)](https://codecov.io/gh/gofhir/cql)
[![Go Report Card](https://goreportcard.com/badge/github.com/gofhir/cql)](https://goreportcard.com/report/github.com/gofhir/cql)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A native **Clinical Quality Language (CQL)** engine for Go, designed for evaluating CQL expressions against FHIR R4 resources.

## Installation

```bash
go get github.com/gofhir/cql
```

## Quick Start

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gofhir/cql"
)

func main() {
	engine := cql.NewEngine(
		cql.WithTimeout(30 * time.Second),
	)

	cqlSource := `
		library Example version '1.0'
		using FHIR version '4.0.1'
		context Patient
		define IsAdult: AgeInYears() >= 18
	`

	patient := json.RawMessage(`{"resourceType": "Patient", "birthDate": "1990-01-01"}`)

	results, err := engine.EvaluateLibrary(context.Background(), cqlSource, patient, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("IsAdult:", results["IsAdult"])
}
```

## Conformance

**1731/1731 (100%)** — passes all official [cqframework/cql-tests](https://github.com/cqframework/cql-tests) conformance suites.

| Suite | Tests | Status |
|---|---|---|
| Aggregate Functions | 50/50 | :white_check_mark: |
| Aggregate Operator | 9/9 | :white_check_mark: |
| Arithmetic Functions | 212/212 | :white_check_mark: |
| Comparison Operators | 223/223 | :white_check_mark: |
| Conditional Operators | 9/9 | :white_check_mark: |
| DateTime Operators | 317/317 | :white_check_mark: |
| Errors and Messaging | 4/4 | :white_check_mark: |
| Interval Operators | 412/412 | :white_check_mark: |
| List Operators | 212/212 | :white_check_mark: |
| Logical Operators | 39/39 | :white_check_mark: |
| Nullological Operators | 22/22 | :white_check_mark: |
| Query Expressions | 12/12 | :white_check_mark: |
| String Operators | 81/81 | :white_check_mark: |
| Type Operators | 35/35 | :white_check_mark: |
| Types | 28/28 | :white_check_mark: |
| Value Literals & Selectors | 66/66 | :white_check_mark: |

Run conformance tests locally:

```bash
go test ./conformance/... -v
```

## Features

- Full CQL parsing via ANTLR4 grammar
- 100% conformance with the official CQL test suite
- Expression evaluation with FHIR R4 context
- Pluggable data and terminology providers
- Compiled expression caching
- Configurable timeouts and resource limits
- Trace listener support for debugging

## API

### Engine

```go
// Create engine with options
engine := cql.NewEngine(
    cql.WithDataProvider(dp),
    cql.WithTerminologyProvider(tp),
    cql.WithTimeout(30 * time.Second),
    cql.WithMaxDepth(100),
)

// Evaluate all definitions in a CQL library
results, err := engine.EvaluateLibrary(ctx, cqlSource, resource, params)

// Evaluate a single named expression
value, err := engine.EvaluateExpression(ctx, cqlSource, "ExpressionName", resource, params)

// Validate CQL syntax without evaluation
err := engine.Compile(cqlSource)
```

### Options

| Option | Description | Default |
|---|---|---|
| `WithDataProvider` | Data provider for retrieve expressions | `nil` |
| `WithTerminologyProvider` | Terminology provider for valueset checks | `nil` |
| `WithModelInfo` | FHIR model information | R4 |
| `WithTimeout` | Per-evaluation timeout | 30s |
| `WithMaxExpressionLen` | Maximum CQL source length | 100KB |
| `WithMaxRetrieveSize` | Maximum resources per retrieve | 10000 |
| `WithMaxDepth` | Maximum recursion depth | 100 |
| `WithTraceListener` | Trace listener for debugging | `nil` |

## Error Handling

The engine returns typed errors for different failure modes:

| Error Type | Description |
|---|---|
| `ErrSyntaxError` | CQL parse error |
| `ErrEvaluation` | Runtime evaluation error |
| `ErrTimeout` | Evaluation exceeded timeout |
| `ErrTooCostly` | Expression exceeds size limits |

## License

[MIT](LICENSE) - Copyright (c) 2025 Roberto Araneda
