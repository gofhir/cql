package types

import (
	"fmt"
	"strings"

	fptypes "github.com/gofhir/fhirpath/types"
)

// Code represents a CQL Code type — a code within a code system.
//
// CQL 1.5 declares it `structured type Code { code String, display String,
// system String, version String }`: each element is a String or null, and
// the empty String is a String. A nil field is null.
type Code struct {
	System  *string
	Code    *string
	Display *string
	Version *string
}

// NewCode creates a new Code; a nil element is null.
func NewCode(system, code, display, version *string) Code {
	return Code{System: system, Code: code, Display: display, Version: version}
}

// Str is a String element that is not null.
func Str(s string) *string {
	return &s
}

// Deref is an element's text, "" for null: for the callers that hold a code
// as plain strings, as the terminology providers do.
func Deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// SameText reports whether two elements are both null or hold the same text.
// It compares what they hold, never where.
func SameText(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// sameTextFold is SameText ignoring case.
func sameTextFold(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return strings.EqualFold(*a, *b)
}

// Type returns "Code".
func (c Code) Type() string {
	return "Code"
}

// Equal checks exact equality (system and code must match).
func (c Code) Equal(other fptypes.Value) bool {
	o, ok := other.(Code)
	if !ok {
		return false
	}
	return SameText(c.System, o.System) && SameText(c.Code, o.Code)
}

// Equivalent checks equivalence (system and code, case-insensitive).
func (c Code) Equivalent(other fptypes.Value) bool {
	o, ok := other.(Code)
	if !ok {
		return false
	}
	return sameTextFold(c.System, o.System) && sameTextFold(c.Code, o.Code)
}

// String returns a text representation.
func (c Code) String() string {
	if Deref(c.Display) != "" {
		return fmt.Sprintf("Code '%s' from %s display '%s'", Deref(c.Code), Deref(c.System), Deref(c.Display))
	}
	return fmt.Sprintf("Code '%s' from %s", Deref(c.Code), Deref(c.System))
}

// IsEmpty returns false for Code.
func (c Code) IsEmpty() bool {
	return false
}

// InValueSet checks if this code is in a set of codes (used for terminology checking).
func (c Code) InValueSet(codes []Code) bool {
	for _, vc := range codes {
		if SameText(c.System, vc.System) && SameText(c.Code, vc.Code) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------

// Concept represents a CQL Concept — one or more equivalent Codes.
type Concept struct {
	Codes   []Code
	Display *string
}

// NewConcept creates a new Concept; a nil display is null.
func NewConcept(codes []Code, display *string) Concept {
	return Concept{Codes: codes, Display: display}
}

// Type returns "Concept".
func (c Concept) Type() string {
	return "Concept"
}

// Equal checks exact equality (all codes must match).
func (c Concept) Equal(other fptypes.Value) bool {
	o, ok := other.(Concept)
	if !ok {
		return false
	}
	if len(c.Codes) != len(o.Codes) {
		return false
	}
	for i, code := range c.Codes {
		if !code.Equal(o.Codes[i]) {
			return false
		}
	}
	return true
}

// Equivalent checks if the concept contains at least one equivalent code.
func (c Concept) Equivalent(other fptypes.Value) bool {
	switch o := other.(type) {
	case Concept:
		for _, cc := range c.Codes {
			for _, oc := range o.Codes {
				if cc.Equivalent(oc) {
					return true
				}
			}
		}
		return false
	case Code:
		for _, cc := range c.Codes {
			if cc.Equivalent(o) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// String returns a text representation.
func (c Concept) String() string {
	parts := make([]string, len(c.Codes))
	for i, code := range c.Codes {
		parts[i] = code.String()
	}
	s := "Concept{" + strings.Join(parts, ", ") + "}"
	if Deref(c.Display) != "" {
		s += fmt.Sprintf(" display '%s'", Deref(c.Display))
	}
	return s
}

// IsEmpty returns false for Concept.
func (c Concept) IsEmpty() bool {
	return false
}
