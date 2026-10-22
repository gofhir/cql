package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestChangeNamesEachKind holds each kind of change to a pair of answers
// written by hand.
func TestChangeNamesEachKind(t *testing.T) {
	for _, tt := range []struct {
		old, cur, want string
	}{
		{`{"type": "Integer", "value": "1"}`, `{"value": "1", "type": "Integer"}`, same},
		{`null`, `null`, same},
		{`{"type": "Integer", "value": "1"}`, `{"type": "Long", "value": "1"}`, valueChanged},
		{`{"type": "Integer", "value": "1"}`, `null`, valueChanged},
		{`{"error": "CqlException", "message": "a"}`, `{"error": "CqlException", "message": "b"}`, messageOnly},
		{`{"error": "CqlException", "message": "a"}`, `{"error": "CqlException", "message": "a"}`, same},
		{`{"error": "CqlException", "message": "a"}`, `{"error": "InvalidCast", "message": "a"}`, errorClass},
		{`{"error": "CqlException", "message": "a"}`, `null`, errorToValue},
		{`null`, `{"error": "CqlException", "message": "a"}`, valueToError},
		{`{"error": "Translation", "message": "a"}`, `{"type": "Boolean", "value": true}`, acceptedNow},
		{`{"type": "Boolean", "value": true}`, `{"error": "Translation", "message": "a"}`, refusedNow},
		{`{"error": "CqlException", "message": "a"}`, `{"error": "Translation", "message": "a"}`, refusedNow},
		{`{"error": "Translation", "message": "a"}`, `{"error": "CqlException", "message": "a"}`, acceptedNow},
		{`{"error": "Translation", "message": "a"}`, `{"error": "Translation", "message": "b"}`, messageOnly},
	} {
		a, err := read(json.RawMessage(tt.old))
		if err != nil {
			t.Fatal(err)
		}
		b, err := read(json.RawMessage(tt.cur))
		if err != nil {
			t.Fatal(err)
		}
		if got := change(a, b); got != tt.want {
			t.Errorf("%s → %s: %s, want %s", tt.old, tt.cur, got, tt.want)
		}
	}
}

// TestCompareCountsEveryDefinition holds the counts to a pair of sets with a
// definition each set lacks.
func TestCompareCountsEveryDefinition(t *testing.T) {
	old := map[string]json.RawMessage{
		"D0": json.RawMessage(`null`),
		"D1": json.RawMessage(`{"error": "CqlException", "message": "a"}`),
		"D2": json.RawMessage(`{"type": "Boolean", "value": true}`),
	}
	cur := map[string]json.RawMessage{
		"D0": json.RawMessage(`null`),
		"D1": json.RawMessage(`{"error": "CqlException", "message": "b"}`),
		"D3": json.RawMessage(`{"type": "Boolean", "value": true}`),
	}
	got, err := compare(old, cur)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{same: 1, messageOnly: 1, removed: 1, added: 1}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("compare = %v, want %v", got, want)
	}
}

// TestVerdictChangesKeepTheClassBefore holds the manifest comparison to a pair
// of manifests written by hand.
func TestVerdictChangesKeepTheClassBefore(t *testing.T) {
	old, err := verdicts(strings.NewReader("# header\nD0\tjava\town defect\tx\nD1\tjava\treference defect\tx\nD2\tjs\tdeliberate\tx\n"))
	if err != nil {
		t.Fatal(err)
	}
	cur, err := verdicts(strings.NewReader("D1\tjava\town defect\ty\nD2\tjs\tdeliberate\ty\nD3\tjava\treference defect\ty\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{
		"gone: own defect": 1,
		"reclassified: reference defect → own defect": 1,
		"kept: deliberate":      1,
		"new: reference defect": 1,
	}
	if got := verdictChanges(old, cur); !reflect.DeepEqual(got, want) {
		t.Errorf("verdictChanges = %v, want %v", got, want)
	}
}

// TestRowChangesListEachRowThatMoved holds the rows the report lists, each
// with its verdict before and now, to a pair of manifests written by hand.
func TestRowChangesListEachRowThatMoved(t *testing.T) {
	old := map[string]string{"D0\tjava": "own defect", "D1\tjava": "reference defect", "D2\tjs": "deliberate"}
	cur := map[string]string{"D1\tjava": "own defect", "D2\tjs": "deliberate", "D3\tjava": "reference defect"}
	want := []rowChange{
		{name: "D0", ref: "java", before: "own defect"},
		{name: "D1", ref: "java", before: "reference defect", now: "own defect"},
		{name: "D3", ref: "java", now: "reference defect"},
	}
	if got := rowChanges(old, cur); !reflect.DeepEqual(got, want) {
		t.Errorf("rowChanges = %v, want %v", got, want)
	}
}

// TestRunWritesEachRowWithTheAnswerNow holds the report to the answer the new
// set gives each row that moved.
func TestRunWritesEachRowWithTheAnswerNow(t *testing.T) {
	oldDir, newDir := t.TempDir(), t.TempDir()
	write := func(dir, file, content string) {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(oldDir, "a.java.json", `{"results": {"D0": {"error": "CqlException", "message": "x"}}}`)
	write(newDir, "a.java.json", `{"results": {"D0": {"type": "String", "value": "a | b"}}}`)
	write(oldDir, "a.divergences.tsv", "D0\tjava\treference defect\tx\n")
	write(newDir, "a.divergences.tsv", "")
	var out strings.Builder
	if err := run(&out, oldDir, newDir, "java", true, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	// A | in the answer is escaped, or it would split the row's cells.
	row := "| a | D0 | java | reference defect | gone | " + "`{\"type\":\"String\",\"value\":\"a \\| b\"}` |"
	if !strings.Contains(out.String(), row) {
		t.Errorf("the report lacks the row %q:\n%s", row, out.String())
	}
}
