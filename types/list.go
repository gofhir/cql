package types

import (
	"fmt"
	"strings"

	fptypes "github.com/gofhir/fhirpath/types"
)

// List represents a CQL List<T> — a typed, ordered collection.
// Wraps fhirpath Collection with CQL-specific semantics.
type List struct {
	Values   fptypes.Collection
	TypeName string // optional element type hint (e.g. "Integer")
}

// NewList creates a new List from a collection.
func NewList(values fptypes.Collection) List {
	return List{Values: values}
}

// Type returns "List".
func (l List) Type() string {
	if l.TypeName != "" {
		return fmt.Sprintf("List<%s>", l.TypeName)
	}
	return "List"
}

// Equal checks exact equality.
func (l List) Equal(other fptypes.Value) bool {
	o, ok := other.(List)
	if !ok {
		return false
	}
	if len(l.Values) != len(o.Values) {
		return false
	}
	for i, v := range l.Values {
		if !valuesEqual(v, o.Values[i]) {
			return false
		}
	}
	return true
}

// Equivalent checks equivalence (order-independent for lists).
func (l List) Equivalent(other fptypes.Value) bool {
	o, ok := other.(List)
	if !ok {
		return false
	}
	if len(l.Values) != len(o.Values) {
		return false
	}
	// Order-independent: every element in l must match an element in o and vice versa
	used := make([]bool, len(o.Values))
	for _, v := range l.Values {
		found := false
		for j, ov := range o.Values {
			if used[j] {
				continue
			}
			if valuesEquivalent(v, ov) {
				used[j] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// String returns a text representation.
func (l List) String() string {
	if len(l.Values) == 0 {
		return "{}"
	}
	parts := make([]string, len(l.Values))
	for i, v := range l.Values {
		if v == nil {
			parts[i] = "null"
		} else {
			parts[i] = v.String()
		}
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// IsEmpty returns true if the list has no elements.
func (l List) IsEmpty() bool {
	return len(l.Values) == 0
}
