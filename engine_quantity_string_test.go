package cql

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// quantityStringForm is a definition of testdata/snapshots/quantitystrings.cql
// that reads a string literal: its name, the form, and the literal.
var quantityStringForm = regexp.MustCompile(`(?m)^define (D\d+): (ToQuantity\(|ConvertsToQuantity\(|convert )('(?:[^'\\]|\\.)*')(\)| to Quantity)$`)

// TestTheQuantityFormsAgree holds the three ways quantitystrings reads each
// string to one another, by relations the specification states rather than by
// any answer (plan 010):
//   - convert to Quantity is ToQuantity, which the translator lowers it to;
//   - ConvertsToQuantity is true exactly when ToQuantity has a value: "If the
//     input string is not formatted correctly, or cannot be interpreted as a
//     valid Quantity value, the result is null" (CQL Reference, ToQuantity),
//     and ConvertsToQuantity is whether the value "can be converted";
//   - whitespace of the grammar around the string does not change the answer.
func TestTheQuantityFormsAgree(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", "quantitystrings.cql"))
	if err != nil {
		t.Fatal(err)
	}
	answers := snapshotAnswers(t, "quantitystrings")
	read := map[string]string{} // each literal, as the corpus writes it, to its ToQuantity
	forms := map[string][3]string{}
	for _, m := range quantityStringForm.FindAllStringSubmatch(string(raw), -1) {
		f := forms[m[3]]
		switch m[2] {
		case "ToQuantity(":
			f[0] = answers[m[1]]
		case "ConvertsToQuantity(":
			f[1] = answers[m[1]]
		default:
			f[2] = answers[m[1]]
		}
		forms[m[3]] = f
	}
	if len(forms) < 250 {
		t.Fatalf("%d strings read from the corpus; the pattern lost them", len(forms))
	}
	for literal, f := range forms {
		if f[0] == "" || f[1] == "" || f[2] == "" {
			t.Errorf("%s: not read in all three forms (%q)", literal, f)
			continue
		}
		if f[2] != f[0] {
			t.Errorf("convert %s to Quantity = %s, and ToQuantity = %s", literal, f[2], f[0])
		}
		if converts := fmt.Sprint(f[0] != "null"); f[1] != "Boolean "+converts {
			t.Errorf("ConvertsToQuantity(%s) = %s, and ToQuantity = %s", literal, f[1], f[0])
		}
		read[literal] = f[0]
	}
	// The grammar's whitespace, as the corpus escapes it.
	ws := regexp.MustCompile(`^'((?:\\[tnr]| )*)(.*?)((?:\\[tnr]| )*)'$`)
	compared := 0
	for literal, answer := range read {
		m := ws.FindStringSubmatch(literal)
		if len(m) < 4 || m[1] == "" && m[3] == "" {
			continue
		}
		bare := "'" + m[2] + "'"
		if strings.HasSuffix(m[2], "\\") {
			continue // the escape of a quote, not the end of the literal
		}
		if want, ok := read[bare]; ok {
			compared++
			if answer != want {
				t.Errorf("ToQuantity(%s) = %s, and ToQuantity(%s) = %s", literal, answer, bare, want)
			}
		}
	}
	if compared < 10 {
		t.Errorf("only %d strings compared with and without whitespace around", compared)
	}
}

// TestConvertToQuantityKeepsItsOverloads holds ToQuantity and convert to
// Quantity to the operands ToQuantity has an overload of (CQL Reference,
// ToQuantity: Decimal, Integer, Long, Ratio and String). Any other is no
// conversion, which the reference translator refuses and the Java engine
// fails on ("Cannot cast a value of type … as Quantity"), and stays the error
// it was before plan 010; ConvertsToQuantity of it is false.
func TestConvertToQuantityKeepsItsOverloads(t *testing.T) {
	for _, expr := range []string{"convert true to Quantity", "convert @2020-01-01 to Quantity", "convert @T10:00 to Quantity",
		"ToQuantity(true)", "ToQuantity(@2020-01-01)", "ToQuantity({1})"} {
		got, err := NewEngine().EvaluateExpression(context.Background(),
			"library T version '1.0'\ndefine R: "+expr+"\n", "R", nil, nil)
		if err == nil || !strings.Contains(err.Error(), "cannot convert") {
			t.Errorf("%s = %v, %v; want the error that it cannot convert", expr, got, err)
		}
	}
}

// TestConvertsToQuantityOfATypeWithNoOverloadIsFalse: ConvertsToQuantity is
// false where ToQuantity has no overload, as it was before plan 010.
func TestConvertsToQuantityOfATypeWithNoOverloadIsFalse(t *testing.T) {
	for _, expr := range []string{"ConvertsToQuantity(true)", "ConvertsToQuantity(@2020-01-01)"} {
		got, err := NewEngine().EvaluateExpression(context.Background(),
			"library T version '1.0'\ndefine R: "+expr+"\n", "R", nil, nil)
		if err != nil || got == nil || got.String() != "false" {
			t.Errorf("%s = %v, %v; want false", expr, got, err)
		}
	}
}
