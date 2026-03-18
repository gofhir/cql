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

## Features

- Full CQL parsing via ANTLR4 grammar
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
