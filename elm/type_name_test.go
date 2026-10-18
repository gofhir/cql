package elm

import (
	"testing"

	"github.com/gofhir/cql/ast"
)

// TestATypeNameKeepsItsModel covers the names ELM writes with the model's URI
// in them, {urn:hl7-org:elm-types:r1}Any: they are imported as the qualified
// names CQL text writes, System.Any, which the evaluator reads by qualifier.
// Left in the name, Any was never recognized, and the System model's URI was
// kept as a qualifier no one reads.
func TestATypeNameKeepsItsModel(t *testing.T) {
	for in, want := range map[string]ast.NamedType{
		"{urn:hl7-org:elm-types:r1}Any": {Namespace: "System", Name: "Any"},
		"{http://hl7.org/fhir}Patient":  {Namespace: "FHIR", Name: "Patient"},
		"Integer":                       {Name: "Integer"},
	} {
		got, ok := importTypeSpecifier(&TypeSpecifier{Type: "NamedTypeSpecifier", Name: in}).(*ast.NamedType)
		if !ok || *got != want {
			t.Errorf("%s imported as %+v; want %+v", in, got, want)
		}
	}
}
