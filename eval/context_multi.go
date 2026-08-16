package eval

import (
	"encoding/json"
	"fmt"
	"strings"

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
	c.cachedSubjectErr = nil
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
//
// An empty result means only that there is no id to scope by; it does not say
// whether that is because no context resource was supplied. Use subjectID when
// the difference matters.
func (c *Context) GetContextSubjectID() string {
	id, err := c.subjectID()
	if err != nil {
		return ""
	}
	return id
}

// subjectID returns the context subject's id, or the reason there is none.
//
// No context resource is not an error: that is a population-level run, and the
// caller asked for one. A context resource that yields no id is, because the
// two are indistinguishable downstream — both produce an unscoped retrieve —
// and only one of them was intended.
func (c *Context) subjectID() (string, error) {
	if c.cachedSubjectOK {
		return c.cachedSubjectID, c.cachedSubjectErr
	}
	c.cachedSubjectOK = true
	if len(c.ContextValue) == 0 {
		return "", nil
	}
	var resource struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(c.ContextValue, &resource); err != nil {
		c.cachedSubjectErr = fmt.Errorf("%w: %s resource does not parse: %w",
			ErrContextSubjectUnusable, orUnknown(c.contextResourceType), err)
		return "", c.cachedSubjectErr
	}
	if resource.ID == "" {
		c.cachedSubjectErr = fmt.Errorf("%w: %s resource has no id",
			ErrContextSubjectUnusable, orUnknown(c.contextResourceType))
		return "", c.cachedSubjectErr
	}
	c.cachedSubjectID = resource.ID
	return c.cachedSubjectID, nil
}

func orUnknown(resourceType string) string {
	if resourceType == "" {
		return "context"
	}
	return resourceType
}

// ResolveRelatedContext allows querying data from a related context.
// For example, from within a Patient context, query Practitioner data
// using the patient's generalPractitioner reference.
//
// The reference is asked for by id and the answer is checked against it. The
// request used to name no id at all — CodePath was "_id" with nothing to
// compare against — so the provider saw an unfiltered retrieve and the first
// resource of that type in the database came back as though it were the
// referenced one. A provider that ignores IDs still returns everything, which
// is why the id is verified here rather than trusted: this engine cannot make
// a provider filter, but it can refuse to hand back a resource that is not the
// one that was asked for.
func (e *Evaluator) ResolveRelatedContext(targetType, reference string) (fptypes.Value, error) {
	if e.ctx.DataProvider == nil {
		return nil, nil
	}
	id := referenceID(reference)
	if id == "" {
		return nil, nil
	}
	req := RetrieveRequest{
		ResourceType: targetType,
		IDs:          []string{id},
		Limit:        retrieveLimit(e.ctx.MaxRetrieveSize),
	}
	resources, err := e.ctx.DataProvider.Retrieve(e.ctx.GoCtx, req)
	// Every query the engine makes has to appear in a provenance trail, not
	// only the ones a retrieve expression makes: resources fetched here are
	// just as much a reason a decision came out the way it did.
	if observer, ok := e.ctx.TraceListener.(RetrieveObserver); ok {
		observer.OnRetrieve(req, len(resources), err)
	}
	if err != nil {
		return nil, err
	}
	for _, resource := range resources {
		var header struct {
			ResourceType string `json:"resourceType"`
			ID           string `json:"id"`
		}
		if err := json.Unmarshal(resource, &header); err != nil {
			continue
		}
		if header.ID != id {
			continue
		}
		if header.ResourceType != "" && !strings.EqualFold(header.ResourceType, targetType) {
			continue
		}
		return fptypes.NewString(string(resource)), nil
	}
	return nil, nil
}

// referenceID reduces a FHIR reference to the resource id it names.
//
// "Practitioner/123" and "http://example.org/fhir/Practitioner/123" both name
// 123. A bare "123" is taken as an id already. Contained references ("#x") and
// URN references name nothing a retrieve by id can find, so they resolve to
// nothing rather than to a wrong resource.
func referenceID(reference string) string {
	ref := strings.TrimSpace(reference)
	if ref == "" || strings.HasPrefix(ref, "#") || strings.HasPrefix(ref, "urn:") {
		return ""
	}
	// Drop a version suffix before taking the last segment: _history/2 would
	// otherwise be read as the id.
	if i := strings.Index(ref, "/_history/"); i >= 0 {
		ref = ref[:i]
	}
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		ref = ref[i+1:]
	}
	if strings.ContainsAny(ref, "?&") {
		return ""
	}
	return ref
}
