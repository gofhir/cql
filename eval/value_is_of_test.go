package eval

import (
	"context"
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
)

// TestATypeIsReadWithItsModel covers the qualifier of a named type: Any is
// System's, unqualified or qualified, and a System value is of a System type
// of its name. The FHIR model's descent is covered with the model, where it
// applies, by TestTheFHIRModelsDescentIsOnlyAFHIRValues.
func TestATypeIsReadWithItsModel(t *testing.T) {
	e := NewEvaluator(NewContext(context.Background(), nil))
	for _, tt := range []struct {
		namespace, name string
		want            bool
	}{
		{"System", "Any", true},
		{"", "Any", true},
		{"System", "Integer", true},
		{"System", "String", false},
	} {
		got := e.valueIsOf(fptypes.NewInteger(5), &ast.NamedType{Namespace: tt.namespace, Name: tt.name})
		if got != tt.want {
			t.Errorf("5 is %s.%s = %t; want %t", tt.namespace, tt.name, got, tt.want)
		}
	}
}
