package conformance

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTestSuite(t *testing.T) {
	raw := `<?xml version="1.0" encoding="utf-8"?>
<tests xmlns="http://hl7.org/fhirpath/tests" name="SampleSuite" version="1.0">
  <capability code="arithmetic-operators"/>
  <group name="Addition" version="1.0">
    <test name="AddIntegers" version="1.0">
      <expression>2 + 3</expression>
      <output>5</output>
    </test>
    <test name="AddNull" version="1.0">
      <expression>2 + null</expression>
    </test>
    <test name="InvalidExpression" version="1.0">
      <expression invalid="true">@invalidliteral</expression>
    </test>
  </group>
</tests>`

	var suite TestSuite
	if err := xml.Unmarshal([]byte(raw), &suite); err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	if suite.Name != "SampleSuite" {
		t.Errorf("expected suite name %q, got %q", "SampleSuite", suite.Name)
	}
	if suite.Version != "1.0" {
		t.Errorf("expected suite version %q, got %q", "1.0", suite.Version)
	}
	if len(suite.Capabilities) != 1 {
		t.Fatalf("expected 1 suite capability, got %d", len(suite.Capabilities))
	}
	if suite.Capabilities[0].Code != "arithmetic-operators" {
		t.Errorf("expected capability code %q, got %q", "arithmetic-operators", suite.Capabilities[0].Code)
	}
	if len(suite.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(suite.Groups))
	}

	group := suite.Groups[0]
	if group.Name != "Addition" {
		t.Errorf("expected group name %q, got %q", "Addition", group.Name)
	}
	if len(group.Tests) != 3 {
		t.Fatalf("expected 3 tests, got %d", len(group.Tests))
	}

	// Test 1: normal expression with output
	tc := group.Tests[0]
	if tc.Name != "AddIntegers" {
		t.Errorf("expected test name %q, got %q", "AddIntegers", tc.Name)
	}
	if strings.TrimSpace(tc.Expression.Value) != "2 + 3" {
		t.Errorf("expected expression %q, got %q", "2 + 3", tc.Expression.Value)
	}
	if len(tc.Outputs) != 1 {
		t.Fatalf("expected 1 output, got %d", len(tc.Outputs))
	}
	if strings.TrimSpace(tc.Outputs[0].Value) != "5" {
		t.Errorf("expected output %q, got %q", "5", tc.Outputs[0].Value)
	}

	// Test 2: null output (no <output> element)
	tc2 := group.Tests[1]
	if tc2.Name != "AddNull" {
		t.Errorf("expected test name %q, got %q", "AddNull", tc2.Name)
	}
	if len(tc2.Outputs) != 0 {
		t.Errorf("expected 0 outputs for null test, got %d", len(tc2.Outputs))
	}

	// Test 3: invalid expression
	tc3 := group.Tests[2]
	if tc3.Name != "InvalidExpression" {
		t.Errorf("expected test name %q, got %q", "InvalidExpression", tc3.Name)
	}
	if tc3.Expression.Invalid != "true" {
		t.Errorf("expected invalid attr %q, got %q", "true", tc3.Expression.Invalid)
	}
}

func TestParseRealFile(t *testing.T) {
	files, err := filepath.Glob("testdata/*.xml")
	if err != nil {
		t.Fatalf("glob error: %v", err)
	}
	if len(files) == 0 {
		t.Skip("no XML files found in testdata/")
	}

	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("failed to read %s: %v", f, err)
			}

			var suite TestSuite
			if err := xml.Unmarshal(data, &suite); err != nil {
				t.Fatalf("failed to unmarshal %s: %v", f, err)
			}

			if suite.Name == "" {
				t.Errorf("suite name is empty in %s", f)
			}

			totalTests := 0
			for _, g := range suite.Groups {
				totalTests += len(g.Tests)
			}
			t.Logf("suite=%s groups=%d tests=%d", suite.Name, len(suite.Groups), totalTests)
		})
	}
}
