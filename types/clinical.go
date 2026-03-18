package types

import (
	"fmt"
	"strings"

	fptypes "github.com/gofhir/fhirpath/types"
)

// Code represents a CQL Code type — a code within a code system.
type Code struct {
	System  string
	Code    string
	Display string
	Version string
}

// NewCode creates a new Code.
func NewCode(system, code, display string) Code {
	return Code{System: system, Code: code, Display: display}
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
	return c.System == o.System && c.Code == o.Code
}

// Equivalent checks equivalence (system and code, case-insensitive).
func (c Code) Equivalent(other fptypes.Value) bool {
	o, ok := other.(Code)
	if !ok {
		return false
	}
	return strings.EqualFold(c.System, o.System) && strings.EqualFold(c.Code, o.Code)
}

// String returns a text representation.
func (c Code) String() string {
	if c.Display != "" {
		return fmt.Sprintf("Code '%s' from %s display '%s'", c.Code, c.System, c.Display)
	}
	return fmt.Sprintf("Code '%s' from %s", c.Code, c.System)
}

// IsEmpty returns false for Code.
func (c Code) IsEmpty() bool {
	return false
}

// InValueSet checks if this code is in a set of codes (used for terminology checking).
func (c Code) InValueSet(codes []Code) bool {
	for _, vc := range codes {
		if c.System == vc.System && c.Code == vc.Code {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------

// Concept represents a CQL Concept — one or more equivalent Codes.
type Concept struct {
	Codes   []Code
	Display string
}

// NewConcept creates a new Concept.
func NewConcept(codes []Code, display string) Concept {
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
	if c.Display != "" {
		s += fmt.Sprintf(" display '%s'", c.Display)
	}
	return s
}

// IsEmpty returns false for Concept.
func (c Concept) IsEmpty() bool {
	return false
}
