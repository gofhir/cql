package eval

import (
	"encoding/json"

	fptypes "github.com/gofhir/fhirpath/types"
)

// ContextType represents the type of evaluation context.
type ContextType string

const (
	ContextPatient      ContextType = "Patient"
	ContextPractitioner ContextType = "Practitioner"
	ContextEncounter    ContextType = "Encounter"
	ContextUnfiltered   ContextType = "Unfiltered"
)

// SetContextType sets the evaluation context type.
// In "Patient" context, data retrieval is scoped to the current patient.
// In "Unfiltered" context, data retrieval returns all data.
func (c *Context) SetContextType(ct ContextType) {
	c.contextType = ct
}

// GetContextType returns the current evaluation context type.
func (c *Context) GetContextType() ContextType {
	return c.contextType
}

// SetContextResource sets the resource for the current evaluation context.
// Invalidates the cached subject ID since the context value changed.
func (c *Context) SetContextResource(resourceType string, data json.RawMessage) {
	c.ContextValue = data
	c.contextResourceType = resourceType
	c.cachedSubjectID = ""
	c.cachedSubjectOK = false
	c.cachedObject = nil
}

// GetContextResourceType returns the resource type of the current context.
func (c *Context) GetContextResourceType() string {
	return c.contextResourceType
}

// SwitchContext creates a new child context with a different context type.
// This is used when a CQL library has multiple context definitions
// (e.g., both "context Patient" and "context Practitioner").
func (c *Context) SwitchContext(ct ContextType, resource json.RawMessage) *Context {
	child := c.ChildScope()
	child.contextType = ct
	child.ContextValue = resource
	return child
}

// IsUnfilteredContext returns true if the context is Unfiltered.
func (c *Context) IsUnfilteredContext() bool {
	return c.contextType == ContextUnfiltered
}

// GetContextSubjectID extracts the subject ID from the context value.
// Used by DataProvider to scope data retrieval to the current subject.
// The result is cached to avoid repeated JSON unmarshaling.
func (c *Context) GetContextSubjectID() string {
	if c.cachedSubjectOK {
		return c.cachedSubjectID
	}
	c.cachedSubjectOK = true
	if c.ContextValue == nil {
		return ""
	}
	var resource struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(c.ContextValue, &resource); err != nil {
		return ""
	}
	c.cachedSubjectID = resource.ID
	return c.cachedSubjectID
}

// ResolveRelatedContext allows querying data from a related context.
// For example, from within a Patient context, query Practitioner data
// using the patient's generalPractitioner reference.
func (e *Evaluator) ResolveRelatedContext(targetType string, reference string) (fptypes.Value, error) {
	if e.ctx.DataProvider == nil {
		return nil, nil
	}
	// Retrieve the referenced resource
	resources, err := e.ctx.DataProvider.Retrieve(
		e.ctx.GoCtx,
		targetType,
		"_id",
		"=",
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}
	if len(resources) == 0 {
		return nil, nil
	}
	return fptypes.NewString(string(resources[0])), nil
}
