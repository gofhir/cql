package types

import (
	"fmt"
	"sort"
	"strings"

	fptypes "github.com/gofhir/fhirpath/types"
)

// Tuple represents a CQL Tuple — an ordered set of named elements.
type Tuple struct {
	Elements     map[string]fptypes.Value
	TypeOverride string // optional type name for instance expressions (e.g., "ValueSet")
}

// NewTuple creates a new Tuple from a map of named values.
func NewTuple(elements map[string]fptypes.Value) Tuple {
	if elements == nil {
		elements = make(map[string]fptypes.Value)
	}
	return Tuple{Elements: elements}
}

// Type returns "Tuple" or the overridden type name for instance expressions.
func (t Tuple) Type() string {
	if t.TypeOverride != "" {
		return t.TypeOverride
	}
	return "Tuple"
}

// Equal checks exact equality: same keys, same values.
// Returns false if structurally different, but note that CQL equality
// with null fields should return null — this is handled at the evaluator level.
func (t Tuple) Equal(other fptypes.Value) bool {
	o, ok := other.(Tuple)
	if !ok {
		return false
	}
	if len(t.Elements) != len(o.Elements) {
		return false
	}
	for k, v := range t.Elements {
		ov, exists := o.Elements[k]
		if !exists {
			return false
		}
		if !valuesEqual(v, ov) {
			return false
		}
	}
	return true
}

// Equivalent checks equivalence.
func (t Tuple) Equivalent(other fptypes.Value) bool {
	o, ok := other.(Tuple)
	if !ok {
		return false
	}
	if len(t.Elements) != len(o.Elements) {
		return false
	}
	for k, v := range t.Elements {
		ov, exists := o.Elements[k]
		if !exists {
			return false
		}
		if !valuesEquivalent(v, ov) {
			return false
		}
	}
	return true
}

// String returns a text representation.
func (t Tuple) String() string {
	if len(t.Elements) == 0 {
		return "Tuple{}"
	}
	keys := make([]string, 0, len(t.Elements))
	for k := range t.Elements {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := t.Elements[k]
		vs := "null"
		if v != nil {
			vs = v.String()
		}
		parts = append(parts, fmt.Sprintf("%s: %s", k, vs))
	}
	return "Tuple{" + strings.Join(parts, ", ") + "}"
}

// IsEmpty returns false for Tuple.
func (t Tuple) IsEmpty() bool {
	return false
}

// Get returns the value for a named element.
func (t Tuple) Get(name string) (fptypes.Value, bool) {
	v, ok := t.Elements[name]
	return v, ok
}
