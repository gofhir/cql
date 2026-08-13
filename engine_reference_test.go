package cql

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/compiler"
	"github.com/gofhir/cql/eval"
)

// referenceELM is what the reference translator decided, recorded by
// internal/referenceharness. It is committed rather than regenerated in CI: the
// translator needs a JVM and some thirty jars, which is too much to ask of every
// build, and the point is to diff against it, not to re-derive it.
type referenceELM struct {
	Translator  string `json:"translator"`
	Definitions []struct {
		Name        string   `json:"name"`
		ResultType  string   `json:"resultType"`
		Locator     string   `json:"locator"`
		Conversions []string `json:"conversions"`
	} `json:"definitions"`
}

type referenceProvider struct{}

func (referenceProvider) Retrieve(_ context.Context, req eval.RetrieveRequest) ([]json.RawMessage, error) {
	if req.ResourceType != "Encounter" {
		return nil, nil
	}
	return []json.RawMessage{json.RawMessage(
		`{"resourceType":"Encounter","id":"e1","period":{"start":"2020-03-01","end":"2020-03-05"}}`)}, nil
}

// TestPositionsMatchTheReference covers where this engine says an expression
// begins against where the reference translator says it does.
//
// Agreeing matters beyond tidiness: a diagnostic that is off by a column from
// every other CQL tool is worse than useless when the two are read side by side.
// ANTLR counts columns from zero and the ELM locators count from one, which is
// exactly the kind of divergence this comparison exists to catch — it caught
// this one.
func TestPositionsMatchTheReference(t *testing.T) {
	ref := loadReference(t)
	src := loadProbe(t)

	lib, err := compiler.Compile(src)
	if err != nil {
		t.Fatalf("compiling the probe: %v", err)
	}

	for _, want := range ref.Definitions {
		if want.Locator == "" {
			continue // the implicit context definition has no source of its own
		}
		stmt := findDefine(lib, want.Name)
		if stmt == nil {
			t.Errorf("%s: the reference has it and we do not", want.Name)
			continue
		}
		pos, known := ast.PositionOf(stmt.Expression)
		if !known {
			t.Errorf("%s: no position recorded", want.Name)
			continue
		}
		if pos.String() != want.Locator {
			t.Errorf("%s begins at %s, the reference says %s", want.Name, pos, want.Locator)
		}
	}
}

// TestConversionsMatchTheReference covers where the reference translator
// inserts a FHIRHelpers call. It inserts them statically, from the types it
// inferred; this engine applies the same conversions at evaluation, so the
// places should line up even though the mechanism differs.
func TestConversionsMatchTheReference(t *testing.T) {
	ref := loadReference(t)
	src := loadProbe(t)

	engine := NewEngine(WithDataProvider(referenceProvider{}))
	lib, err := engine.Parse(src)
	if err != nil {
		t.Fatalf("parsing the probe: %v", err)
	}
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)

	for _, want := range ref.Definitions {
		if len(want.Conversions) == 0 {
			continue
		}
		// A definition the reference had to convert for must evaluate here
		// rather than failing on the unconverted FHIR type.
		got, err := engine.EvaluateParsedExpression(context.Background(), lib, want.Name, patient, nil)
		if err != nil {
			t.Errorf("%s needs %v and failed: %v", want.Name, want.Conversions, err)
			continue
		}
		if got == nil {
			t.Errorf("%s needs %v and answered null", want.Name, want.Conversions)
		}
	}
}

// TestResultTypesAgainstTheReference records where the types this engine
// produces differ from the ones the reference infers. It reports rather than
// fails: this engine has no semantic phase yet, so a difference is a finding to
// carry into Etapa 5, not a regression to block on.
func TestResultTypesAgainstTheReference(t *testing.T) {
	ref := loadReference(t)
	src := loadProbe(t)

	engine := NewEngine(WithDataProvider(referenceProvider{}))
	lib, err := engine.Parse(src)
	if err != nil {
		t.Fatalf("parsing the probe: %v", err)
	}
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)

	agreed, differed := 0, 0
	for _, want := range ref.Definitions {
		if want.ResultType == "" {
			continue
		}
		got, err := engine.EvaluateParsedExpression(context.Background(), lib, want.Name, patient, nil)
		if err != nil {
			t.Logf("%-9s reference %-10s engine failed: %v", want.Name, want.ResultType, err)
			differed++
			continue
		}
		actual := "null"
		if got != nil {
			actual = got.Type()
		}
		if actual == want.ResultType {
			agreed++
			continue
		}
		differed++
		t.Logf("%-9s reference %-10s engine %-10s", want.Name, want.ResultType, actual)
	}
	t.Logf("agreed on %d result types, differed on %d", agreed, differed)
}

func loadReference(t *testing.T) referenceELM {
	t.Helper()
	raw, err := os.ReadFile("testdata/reference/probe.expected.json")
	if err != nil {
		t.Skipf("no reference recorded: %v", err)
	}
	var ref referenceELM
	if err := json.Unmarshal(raw, &ref); err != nil {
		t.Fatalf("reading the reference: %v", err)
	}
	return ref
}

func loadProbe(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/reference/probe.cql")
	if err != nil {
		t.Skipf("no probe library: %v", err)
	}
	return string(raw)
}

func findDefine(lib *ast.Library, name string) *ast.ExpressionDef {
	for _, stmt := range lib.Statements {
		if stmt.Name == name {
			return stmt
		}
	}
	return nil
}
