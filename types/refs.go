package types

import (
	"fmt"

	fptypes "github.com/gofhir/fhirpath/types"
)

// CodeRef is a deferred reference to a Code defined in a library's codesystem/code definitions.
// It is resolved during evaluation against the context's CodeSystems and Codes.
type CodeRef struct {
	Name    string // name of the code definition
	Library string // optional library alias
}

// NewCodeRef creates a new CodeRef.
func NewCodeRef(name, library string) CodeRef {
	return CodeRef{Name: name, Library: library}
}

func (c CodeRef) Type() string                        { return "CodeRef" }
func (c CodeRef) Equal(other fptypes.Value) bool      { return false }
func (c CodeRef) Equivalent(other fptypes.Value) bool { return false }
func (c CodeRef) IsEmpty() bool                       { return false }
func (c CodeRef) String() string {
	if c.Library != "" {
		return fmt.Sprintf("CodeRef(%s.%s)", c.Library, c.Name)
	}
	return fmt.Sprintf("CodeRef(%s)", c.Name)
}

// CodeSystemRef is a deferred reference to a CodeSystem defined in a library.
type CodeSystemRef struct {
	Name    string
	Library string
}

// NewCodeSystemRef creates a new CodeSystemRef.
func NewCodeSystemRef(name, library string) CodeSystemRef {
	return CodeSystemRef{Name: name, Library: library}
}

func (c CodeSystemRef) Type() string                        { return "CodeSystemRef" }
func (c CodeSystemRef) Equal(other fptypes.Value) bool      { return false }
func (c CodeSystemRef) Equivalent(other fptypes.Value) bool { return false }
func (c CodeSystemRef) IsEmpty() bool                       { return false }
func (c CodeSystemRef) String() string {
	if c.Library != "" {
		return fmt.Sprintf("CodeSystemRef(%s.%s)", c.Library, c.Name)
	}
	return fmt.Sprintf("CodeSystemRef(%s)", c.Name)
}

// ValueSetRef is a deferred reference to a ValueSet defined in a library.
type ValueSetRef struct {
	Name    string
	Library string
}

// NewValueSetRef creates a new ValueSetRef.
func NewValueSetRef(name, library string) ValueSetRef {
	return ValueSetRef{Name: name, Library: library}
}

func (v ValueSetRef) Type() string                        { return "ValueSetRef" }
func (v ValueSetRef) Equal(other fptypes.Value) bool      { return false }
func (v ValueSetRef) Equivalent(other fptypes.Value) bool { return false }
func (v ValueSetRef) IsEmpty() bool                       { return false }
func (v ValueSetRef) String() string {
	if v.Library != "" {
		return fmt.Sprintf("ValueSetRef(%s.%s)", v.Library, v.Name)
	}
	return fmt.Sprintf("ValueSetRef(%s)", v.Name)
}
