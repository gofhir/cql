# CQL Conformance Test Suite Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Run the 16 official cqframework/cql-tests XML suites as native Go subtests to validate conformance and identify missing features.

**Architecture:** A `conformance/` package with XML model structs, an output parser that converts expected output strings to `fptypes.Value`, and a test runner that wraps each CQL expression in a library, evaluates it, and compares against expected output. Tests are hierarchical: `TestConformance/<Suite>/<Group>/<Test>`.

**Tech Stack:** Go `encoding/xml`, `testing`, `github.com/gofhir/cql` (engine), `github.com/gofhir/fhirpath/types` (fptypes), `github.com/gofhir/cql/types` (cqltypes)

---

### Task 1: Vendor the XML test files

**Files:**
- Create: `conformance/testdata/` directory
- Create: 16 XML files downloaded from `https://github.com/cqframework/cql-tests/tree/main/tests/cql/`

**Step 1: Create the testdata directory and download all 16 XML files**

```bash
mkdir -p conformance/testdata
cd conformance/testdata

FILES=(
  CqlAggregateFunctionsTest.xml
  CqlAggregateTest.xml
  CqlArithmeticFunctionsTest.xml
  CqlComparisonOperatorsTest.xml
  CqlConditionalOperatorsTest.xml
  CqlDateTimeOperatorsTest.xml
  CqlErrorsAndMessagingOperatorsTest.xml
  CqlIntervalOperatorsTest.xml
  CqlListOperatorsTest.xml
  CqlLogicalOperatorsTest.xml
  CqlNullologicalOperatorsTest.xml
  CqlQueryTests.xml
  CqlStringOperatorsTest.xml
  CqlTypeOperatorsTest.xml
  CqlTypesTest.xml
  ValueLiteralsAndSelectors.xml
)

BASE=https://raw.githubusercontent.com/cqframework/cql-tests/main/tests/cql

for f in "${FILES[@]}"; do
  curl -sL "$BASE/$f" -o "$f"
done
```

**Step 2: Verify files downloaded correctly**

```bash
ls -la conformance/testdata/*.xml | wc -l
# Expected: 16
```

**Step 3: Commit**

```bash
git add conformance/testdata/
git commit -m "chore: vendor cqframework/cql-tests XML suites (16 files)"
```

---

### Task 2: Create XML model structs

**Files:**
- Create: `conformance/xmlmodel.go`

**Step 1: Write the XML model structs**

```go
package conformance

import "encoding/xml"

// TestSuite is the root <tests> element.
type TestSuite struct {
	XMLName      xml.Name     `xml:"tests"`
	Name         string       `xml:"name,attr"`
	Version      string       `xml:"version,attr"`
	Groups       []TestGroup  `xml:"group"`
	Capabilities []Capability `xml:"capability"`
}

// TestGroup is a <group> element containing related tests.
type TestGroup struct {
	Name         string       `xml:"name,attr"`
	Version      string       `xml:"version,attr"`
	Tests        []TestCase   `xml:"test"`
	Capabilities []Capability `xml:"capability"`
}

// TestCase is a single <test> element.
type TestCase struct {
	Name         string       `xml:"name,attr"`
	Version      string       `xml:"version,attr"`
	Ordered      string       `xml:"ordered,attr"`
	Expression   Expression   `xml:"expression"`
	Outputs      []Output     `xml:"output"`
	Capabilities []Capability `xml:"capability"`
}

// Expression is the <expression> element containing CQL to evaluate.
type Expression struct {
	Invalid string `xml:"invalid,attr"`
	Value   string `xml:",chardata"`
}

// Output is an <output> element with the expected result.
type Output struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

// Capability declares a required capability for a test.
type Capability struct {
	Code  string `xml:"code,attr"`
	Value string `xml:"value,attr"`
}
```

**Step 2: Write a unit test to verify XML parsing works**

Create: `conformance/xmlmodel_test.go`

```go
package conformance

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestParseTestSuite(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="utf-8"?>
<tests xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
       xmlns="http://hl7.org/fhirpath/tests"
       name="TestSuite" version="1.0">
  <group name="Add">
    <test name="Add11">
      <expression>1 + 1</expression>
      <output>2</output>
    </test>
    <test name="AddNull">
      <expression>1 + null</expression>
      <output>null</output>
    </test>
    <test name="AddError">
      <expression invalid="true">overflow expression</expression>
    </test>
  </group>
</tests>`

	var suite TestSuite
	err := xml.NewDecoder(strings.NewReader(xmlData)).Decode(&suite)
	if err != nil {
		t.Fatalf("failed to parse XML: %v", err)
	}

	if suite.Name != "TestSuite" {
		t.Errorf("suite name = %q, want %q", suite.Name, "TestSuite")
	}
	if len(suite.Groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(suite.Groups))
	}

	g := suite.Groups[0]
	if g.Name != "Add" {
		t.Errorf("group name = %q, want %q", g.Name, "Add")
	}
	if len(g.Tests) != 3 {
		t.Fatalf("tests = %d, want 3", len(g.Tests))
	}

	// Test with output
	tc := g.Tests[0]
	if tc.Name != "Add11" {
		t.Errorf("test name = %q, want %q", tc.Name, "Add11")
	}
	if strings.TrimSpace(tc.Expression.Value) != "1 + 1" {
		t.Errorf("expression = %q, want %q", tc.Expression.Value, "1 + 1")
	}
	if len(tc.Outputs) != 1 || strings.TrimSpace(tc.Outputs[0].Value) != "2" {
		t.Errorf("output = %v, want [2]", tc.Outputs)
	}

	// Test with invalid expression
	tc2 := g.Tests[2]
	if tc2.Expression.Invalid != "true" {
		t.Errorf("invalid = %q, want %q", tc2.Expression.Invalid, "true")
	}
	if len(tc2.Outputs) != 0 {
		t.Errorf("invalid test should have no outputs, got %d", len(tc2.Outputs))
	}
}
```

**Step 3: Run the test**

```bash
go test ./conformance/ -run TestParseTestSuite -v
```

Expected: PASS

**Step 4: Test parsing against a real XML file**

Add to `conformance/xmlmodel_test.go`:

```go
func TestParseRealFile(t *testing.T) {
	files, err := filepath.Glob("testdata/*.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Skip("no XML files in testdata/")
	}

	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			var suite TestSuite
			if err := xml.Unmarshal(data, &suite); err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if suite.Name == "" {
				t.Error("suite name is empty")
			}
			total := 0
			for _, g := range suite.Groups {
				total += len(g.Tests)
			}
			t.Logf("parsed %d groups, %d tests", len(suite.Groups), total)
		})
	}
}
```

Add imports: `"os"`, `"path/filepath"`.

**Step 5: Run test against real files**

```bash
go test ./conformance/ -run TestParseRealFile -v
```

Expected: PASS for all 16 files with test counts logged.

**Step 6: Commit**

```bash
git add conformance/xmlmodel.go conformance/xmlmodel_test.go
git commit -m "feat(conformance): add XML model structs for cql-tests format"
```

---

### Task 3: Create the output parser

**Files:**
- Create: `conformance/output_parser.go`
- Create: `conformance/output_parser_test.go`

The output parser converts expected output strings from the XML (e.g., `"42"`, `"'hello'"`, `"@2014-01-01T"`, `"{1, 2, 3}"`) into `fptypes.Value` for comparison with engine results.

**Step 1: Write the output parser**

```go
package conformance

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	fptypes "github.com/gofhir/fhirpath/types"
	cqltypes "github.com/gofhir/cql/types"
)

// parseExpectedOutput converts an XML output string to a fptypes.Value.
// It handles: null, boolean, integer, long, decimal, string, datetime, date,
// time, quantity, list, interval, and tuple formats.
func parseExpectedOutput(raw string) (fptypes.Value, error) {
	s := strings.TrimSpace(raw)

	// null
	if s == "null" {
		return nil, nil
	}

	// boolean
	if s == "true" {
		return fptypes.NewBoolean(true), nil
	}
	if s == "false" {
		return fptypes.NewBoolean(false), nil
	}

	// string (single-quoted)
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		return fptypes.NewString(s[1 : len(s)-1]), nil
	}

	// datetime (@2014-01-01T or @2014-01-01T10:00:00)
	if strings.HasPrefix(s, "@T") {
		// Time: @T10:00:00.000
		timeStr := s[2:] // strip "@T"
		val, err := fptypes.NewTime(timeStr)
		if err != nil {
			return nil, fmt.Errorf("parse time %q: %w", s, err)
		}
		return val, nil
	}
	if strings.HasPrefix(s, "@") {
		// DateTime or Date
		dateStr := s[1:] // strip "@"
		// If it ends with T and nothing after, or contains T with time parts, it's DateTime
		// Date examples: @2014, @2014-01, @2014-01-01 (no trailing T)
		// DateTime examples: @2014T, @2014-01T, @2014-01-01T, @2014-01-01T10:00:00
		if strings.Contains(dateStr, "T") {
			// DateTime - strip trailing T if it's just a marker
			val, err := fptypes.NewDateTime(dateStr)
			if err != nil {
				return nil, fmt.Errorf("parse datetime %q: %w", s, err)
			}
			return val, nil
		}
		// Date
		val, err := fptypes.NewDate(dateStr)
		if err != nil {
			return nil, fmt.Errorf("parse date %q: %w", s, err)
		}
		return val, nil
	}

	// empty list
	if s == "{}" {
		return cqltypes.NewList(nil), nil
	}

	// list: {1, 2, 3}
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		inner := strings.TrimSpace(s[1 : len(s)-1])
		elements, err := splitListElements(inner)
		if err != nil {
			return nil, fmt.Errorf("parse list %q: %w", s, err)
		}
		var values fptypes.Collection
		for _, elem := range elements {
			v, err := parseExpectedOutput(elem)
			if err != nil {
				return nil, fmt.Errorf("parse list element %q: %w", elem, err)
			}
			values = append(values, v)
		}
		return cqltypes.NewList(values), nil
	}

	// interval: Interval[2, 7] or Interval(2, 7] etc.
	if strings.HasPrefix(s, "Interval") {
		return parseInterval(s)
	}

	// tuple: Tuple { key: value, ... }
	if strings.HasPrefix(s, "Tuple") {
		return parseTuple(s)
	}

	// quantity: 5.0'g' or 19.99 '[lb_av]'
	if quantityRe.MatchString(s) {
		val, err := fptypes.NewQuantity(s)
		if err != nil {
			return nil, fmt.Errorf("parse quantity %q: %w", s, err)
		}
		return val, nil
	}

	// long (integer with L suffix)
	if strings.HasSuffix(s, "L") {
		numStr := s[:len(s)-1]
		_, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse long %q: %w", s, err)
		}
		// TODO: implement Long type when supported
		return nil, fmt.Errorf("Long type not yet supported: %s", s)
	}

	// decimal (contains a dot, no quotes)
	if strings.Contains(s, ".") {
		val, err := fptypes.NewDecimal(s)
		if err != nil {
			return nil, fmt.Errorf("parse decimal %q: %w", s, err)
		}
		return val, nil
	}

	// integer
	n, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		return fptypes.NewInteger(n), nil
	}

	return nil, fmt.Errorf("unrecognized output format: %q", s)
}

var quantityRe = regexp.MustCompile(`^-?[\d.]+\s*'[^']*'$`)

// splitListElements splits comma-separated list elements, respecting nesting.
func splitListElements(s string) ([]string, error) {
	var result []string
	depth := 0
	current := strings.Builder{}

	for _, ch := range s {
		switch ch {
		case '{', '[', '(':
			depth++
			current.WriteRune(ch)
		case '}', ']', ')':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				result = append(result, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		case '\'':
			// scan to closing quote
			current.WriteRune(ch)
			// handled by the loop naturally, single quotes don't nest
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}

	return result, nil
}

// parseInterval parses "Interval[2, 7]", "Interval(2, 7]", etc.
func parseInterval(s string) (fptypes.Value, error) {
	// Find opening bracket/paren
	rest := strings.TrimPrefix(s, "Interval")
	rest = strings.TrimSpace(rest)
	if len(rest) < 2 {
		return nil, fmt.Errorf("invalid interval: %q", s)
	}

	lowClosed := rest[0] == '['
	highClosed := rest[len(rest)-1] == ']'

	inner := strings.TrimSpace(rest[1 : len(rest)-1])
	parts, err := splitListElements(inner)
	if err != nil || len(parts) != 2 {
		return nil, fmt.Errorf("invalid interval parts in %q", s)
	}

	low, err := parseExpectedOutput(parts[0])
	if err != nil {
		return nil, fmt.Errorf("parse interval low %q: %w", parts[0], err)
	}
	high, err := parseExpectedOutput(parts[1])
	if err != nil {
		return nil, fmt.Errorf("parse interval high %q: %w", parts[1], err)
	}

	return cqltypes.NewInterval(low, high, lowClosed, highClosed), nil
}

// parseTuple parses "Tuple { key: value, key2: value2 }".
func parseTuple(s string) (fptypes.Value, error) {
	rest := strings.TrimPrefix(s, "Tuple")
	rest = strings.TrimSpace(rest)
	if !strings.HasPrefix(rest, "{") || !strings.HasSuffix(rest, "}") {
		return nil, fmt.Errorf("invalid tuple: %q", s)
	}

	inner := strings.TrimSpace(rest[1 : len(rest)-1])
	if inner == "" {
		return cqltypes.NewTuple(nil), nil
	}

	parts, err := splitListElements(inner)
	if err != nil {
		return nil, fmt.Errorf("parse tuple %q: %w", s, err)
	}

	elements := make(map[string]fptypes.Value)
	for _, part := range parts {
		colonIdx := strings.Index(part, ":")
		if colonIdx < 0 {
			return nil, fmt.Errorf("invalid tuple element %q in %q", part, s)
		}
		key := strings.TrimSpace(part[:colonIdx])
		valStr := strings.TrimSpace(part[colonIdx+1:])
		val, err := parseExpectedOutput(valStr)
		if err != nil {
			return nil, fmt.Errorf("parse tuple value for key %q: %w", key, err)
		}
		elements[key] = val
	}

	return cqltypes.NewTuple(elements), nil
}
```

**Step 2: Write tests for the output parser**

```go
package conformance

import (
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"
	cqltypes "github.com/gofhir/cql/types"
)

func TestParseExpectedOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, v fptypes.Value)
	}{
		{
			name:  "null",
			input: "null",
			check: func(t *testing.T, v fptypes.Value) {
				if v != nil {
					t.Errorf("expected nil, got %v", v)
				}
			},
		},
		{
			name:  "true",
			input: "true",
			check: func(t *testing.T, v fptypes.Value) {
				b, ok := v.(fptypes.Boolean)
				if !ok || !b.Bool() {
					t.Errorf("expected true, got %v", v)
				}
			},
		},
		{
			name:  "false",
			input: "false",
			check: func(t *testing.T, v fptypes.Value) {
				b, ok := v.(fptypes.Boolean)
				if !ok || b.Bool() {
					t.Errorf("expected false, got %v", v)
				}
			},
		},
		{
			name:  "integer",
			input: "42",
			check: func(t *testing.T, v fptypes.Value) {
				i, ok := v.(fptypes.Integer)
				if !ok || i.Value() != 42 {
					t.Errorf("expected 42, got %v", v)
				}
			},
		},
		{
			name:  "negative integer",
			input: "-1",
			check: func(t *testing.T, v fptypes.Value) {
				i, ok := v.(fptypes.Integer)
				if !ok || i.Value() != -1 {
					t.Errorf("expected -1, got %v", v)
				}
			},
		},
		{
			name:  "decimal",
			input: "5.0",
			check: func(t *testing.T, v fptypes.Value) {
				_, ok := v.(fptypes.Decimal)
				if !ok {
					t.Errorf("expected Decimal, got %T", v)
				}
			},
		},
		{
			name:  "string",
			input: "'hello'",
			check: func(t *testing.T, v fptypes.Value) {
				s, ok := v.(fptypes.String)
				if !ok || s.Value() != "hello" {
					t.Errorf("expected 'hello', got %v", v)
				}
			},
		},
		{
			name:  "empty list",
			input: "{}",
			check: func(t *testing.T, v fptypes.Value) {
				l, ok := v.(cqltypes.List)
				if !ok {
					t.Errorf("expected List, got %T", v)
				}
				if len(l.Values) != 0 {
					t.Errorf("expected empty list, got %d elements", len(l.Values))
				}
			},
		},
		{
			name:  "integer list",
			input: "{1, 2, 3}",
			check: func(t *testing.T, v fptypes.Value) {
				l, ok := v.(cqltypes.List)
				if !ok {
					t.Fatalf("expected List, got %T", v)
				}
				if len(l.Values) != 3 {
					t.Fatalf("expected 3 elements, got %d", len(l.Values))
				}
			},
		},
		{
			name:  "interval closed",
			input: "Interval[2, 7]",
			check: func(t *testing.T, v fptypes.Value) {
				iv, ok := v.(cqltypes.Interval)
				if !ok {
					t.Fatalf("expected Interval, got %T", v)
				}
				if !iv.LowClosed || !iv.HighClosed {
					t.Error("expected closed interval")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := parseExpectedOutput(tt.input)
			if err != nil {
				t.Fatalf("parseExpectedOutput(%q) error: %v", tt.input, err)
			}
			tt.check(t, v)
		})
	}
}
```

**Step 3: Run tests**

```bash
go test ./conformance/ -run TestParseExpectedOutput -v
```

Expected: PASS

**Step 4: Commit**

```bash
git add conformance/output_parser.go conformance/output_parser_test.go
git commit -m "feat(conformance): add output parser for CQL test expected values"
```

---

### Task 4: Create the conformance test runner

**Files:**
- Create: `conformance/conformance_test.go`

**Step 1: Write the test runner**

```go
package conformance

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cql "github.com/gofhir/cql"
	fptypes "github.com/gofhir/fhirpath/types"
)

// wrapExpression wraps a bare CQL expression in a minimal library.
func wrapExpression(expr string) string {
	return fmt.Sprintf("library ConformanceTest version '1.0'\ndefine \"result\": %s", expr)
}

// valuesEqual compares two fptypes.Value instances.
// Returns true if both are nil, or if both are non-nil and Equal returns true.
func valuesEqual(got, want fptypes.Value) bool {
	if got == nil && want == nil {
		return true
	}
	if got == nil || want == nil {
		return false
	}
	return got.Equal(want)
}

func TestConformance(t *testing.T) {
	files, err := filepath.Glob("testdata/*.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no XML test files found in testdata/")
	}

	engine := cql.NewEngine()

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}

		var suite TestSuite
		if err := xml.Unmarshal(data, &suite); err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}

		suiteName := strings.TrimSuffix(filepath.Base(f), ".xml")
		t.Run(suiteName, func(t *testing.T) {
			for _, group := range suite.Groups {
				t.Run(group.Name, func(t *testing.T) {
					for _, tc := range group.Tests {
						tc := tc // capture
						t.Run(tc.Name, func(t *testing.T) {
							t.Parallel()
							runTestCase(t, engine, tc)
						})
					}
				})
			}
		})
	}
}

func runTestCase(t *testing.T, engine *cql.Engine, tc TestCase) {
	expr := strings.TrimSpace(tc.Expression.Value)
	if expr == "" {
		t.Skip("empty expression")
	}

	invalid := tc.Expression.Invalid

	// If invalid is set, we expect an error
	if invalid != "" && invalid != "false" {
		cqlSource := wrapExpression(expr)
		_, err := engine.EvaluateExpression(
			context.Background(), cqlSource, "result", nil, nil,
		)
		if err == nil {
			t.Errorf("expected error (invalid=%q) but got success for: %s", invalid, expr)
		}
		return
	}

	// Normal test: evaluate and compare output
	cqlSource := wrapExpression(expr)
	got, err := engine.EvaluateExpression(
		context.Background(), cqlSource, "result", nil, nil,
	)
	if err != nil {
		t.Fatalf("evaluation error: %v\nexpression: %s", err, expr)
	}

	// No output means we just check it doesn't error
	if len(tc.Outputs) == 0 {
		return
	}

	// Single output comparison
	if len(tc.Outputs) == 1 {
		outputStr := strings.TrimSpace(tc.Outputs[0].Value)
		want, err := parseExpectedOutput(outputStr)
		if err != nil {
			t.Fatalf("parse expected output %q: %v", outputStr, err)
		}

		if !valuesEqual(got, want) {
			t.Errorf("expression: %s\ngot:  %v (%T)\nwant: %v (%T)",
				expr, got, got, want, want)
		}
		return
	}

	// Multiple outputs = expected list result
	// Parse each output and compare as a list
	t.Logf("TODO: multi-output comparison for %s", tc.Name)
}
```

**Step 2: Run the conformance tests to get a baseline**

```bash
go test ./conformance/ -run TestConformance -v -count=1 2>&1 | tail -50
```

This will show us which tests pass and fail. Many will fail initially - that's expected and becomes our implementation roadmap.

**Step 3: Get a summary count**

```bash
go test ./conformance/ -run TestConformance -count=1 -json 2>/dev/null | \
  grep -c '"Test".*"Pass":true' || true

go test ./conformance/ -run TestConformance -v -count=1 2>&1 | grep -c "PASS:" || true
go test ./conformance/ -run TestConformance -v -count=1 2>&1 | grep -c "FAIL:" || true
go test ./conformance/ -run TestConformance -v -count=1 2>&1 | grep -c "SKIP:" || true
```

**Step 4: Commit**

```bash
git add conformance/conformance_test.go
git commit -m "feat(conformance): add test runner for cqframework/cql-tests"
```

---

### Task 5: Run baseline conformance and document results

**Files:**
- Modify: `docs/plans/2026-04-04-conformance-tests-design.md` (append baseline results)

**Step 1: Run full conformance suite and capture output**

```bash
go test ./conformance/ -run TestConformance -v -count=1 2>&1 | tee /tmp/conformance-baseline.txt
```

**Step 2: Count results by category**

```bash
echo "=== CONFORMANCE BASELINE ==="
echo "PASS: $(grep -c '--- PASS' /tmp/conformance-baseline.txt)"
echo "FAIL: $(grep -c '--- FAIL' /tmp/conformance-baseline.txt)"
echo "SKIP: $(grep -c '--- SKIP' /tmp/conformance-baseline.txt)"
echo ""
echo "=== FAILURES BY SUITE ==="
grep '--- FAIL' /tmp/conformance-baseline.txt | \
  sed 's/.*TestConformance\/\([^/]*\).*/\1/' | sort | uniq -c | sort -rn
```

**Step 3: Document the baseline in the design doc**

Append a "Baseline Results" section to the design doc with the pass/fail/skip counts per suite. This becomes the roadmap for implementation work.

**Step 4: Commit**

```bash
git add docs/plans/
git commit -m "docs: add conformance baseline results"
```

---

## Notes for the Implementer

### Key implementation details

1. **XML namespace**: The files use `xmlns="http://hl7.org/fhirpath/tests"`. Go's `encoding/xml` handles this - the struct tags don't need the namespace prefix because `xml.Unmarshal` strips default namespaces automatically.

2. **Expression wrapping**: Each test expression must be wrapped in a CQL library. The minimal wrapper is:
   ```
   library ConformanceTest version '1.0'
   define "result": <expression>
   ```

3. **Output parsing edge cases**:
   - Whitespace is inconsistent in XML outputs (e.g., `{1, 2, 3}` vs `{1,2,3}`)
   - DateTime with trailing `T` (e.g., `@2014-01-01T`) vs Date without (e.g., `@2014-01-01`)
   - Quantity format: `5.0'g'` or `19.99 '[lb_av]'` (space before unit is optional)
   - `splitListElements` must handle nested structures (lists within lists, intervals)

4. **Value comparison**: Use `fptypes.Value.Equal()` for comparison. For lists, ordering may or may not matter (check `test.ordered` attribute).

5. **fptypes constructors**:
   - `fptypes.NewInteger(int64)`, `fptypes.NewBoolean(bool)`, `fptypes.NewString(string)`
   - `fptypes.NewDecimal(string)` returns `(Decimal, error)`
   - `fptypes.NewDateTime(string)`, `fptypes.NewDate(string)`, `fptypes.NewTime(string)` return `(T, error)`
   - `fptypes.NewQuantity(string)` returns `(Quantity, error)`
   - `cqltypes.NewList(fptypes.Collection)`, `cqltypes.NewInterval(low, high, lowClosed, highClosed)`
   - `cqltypes.NewTuple(map[string]fptypes.Value)`

6. **Error types**: The engine returns `*cql.ErrSyntaxError` for parse errors, `*cql.ErrEvaluation` for runtime errors. For `invalid="semantic"` tests, you may get either type depending on when the engine catches the error.
