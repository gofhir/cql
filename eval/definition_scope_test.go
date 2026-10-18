package eval

import (
	"context"
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/compiler"
)

// TestADefinitionUnderASwitchedContextIsAboutItsSubject covers SwitchContext. A
// definition reached through another one evaluates in the scope the switch
// opened, for the subject switched to — not at the top of the chain, under the
// subject the switch was made from.
func TestADefinitionUnderASwitchedContextIsAboutItsSubject(t *testing.T) {
	lib, err := compiler.Compile("library T version '1.0'\n\n" +
		"context Patient\n\ndefine Who: Patient.id\ndefine Use: Who\n")
	if err != nil {
		t.Fatalf("compiling: %v", err)
	}
	ctx := NewContext(context.Background(), lib)
	ctx.SetContextResource("Patient", []byte(`{"resourceType":"Patient","id":"A"}`))
	switched := ctx.SwitchContext(ContextPatient, []byte(`{"resourceType":"Patient","id":"B"}`))
	got, err := NewEvaluator(switched).EvaluateExpression("Use")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.String() != "B" {
		t.Errorf("Use under Patient B = %v; want B", got)
	}
}

// TestADefinitionKeepsTheCallersSubjectAndNotItsAliases covers a scope a
// caller opens itself. A definition and a function body evaluate with the
// caller's subject and providers — a child scope given Patient B is about B —
// and without the aliases around it: a context switched to under a query
// alias Y does not hand Y to a definition that reads the definition Y.
func TestADefinitionKeepsTheCallersSubjectAndNotItsAliases(t *testing.T) {
	lib, err := compiler.Compile("library T version '1.0'\n\n" +
		"define Y: 1\ndefine D: Y + 1\n\ncontext Patient\n\n" +
		"define Who: Patient.id\ndefine Use: Who\ndefine function F(): Patient.id\ndefine UseF: F()\n")
	if err != nil {
		t.Fatalf("compiling: %v", err)
	}
	for _, name := range []string{"Use", "UseF"} {
		ctx := NewContext(context.Background(), lib)
		ctx.SetContextResource("Patient", []byte(`{"resourceType":"Patient","id":"A"}`))
		child := ctx.ChildScope()
		child.SetContextResource("Patient", []byte(`{"resourceType":"Patient","id":"B"}`))
		got, evalErr := NewEvaluator(child).EvaluateExpression(name)
		if evalErr != nil || got == nil || got.String() != "B" {
			t.Errorf("%s in a child scope of Patient B = %v, %v; want B", name, got, evalErr)
		}
	}

	ctx := NewContext(context.Background(), lib)
	query := ctx.ChildScope()
	query.Aliases["Y"] = fptypes.NewInteger(10)
	switched := query.SwitchContext(ContextPatient, []byte(`{"resourceType":"Patient","id":"B"}`))
	got, err := NewEvaluator(switched).EvaluateExpression("D")
	if err != nil || got == nil || got.String() != "2" {
		t.Errorf("D under a query alias Y = %v, %v; want 2", got, err)
	}
}
