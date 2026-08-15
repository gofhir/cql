package sema

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gofhir/cql/ast"
)

// Severity separates what makes a library wrong from what merely makes it
// suspect.
type Severity int

const (
	// SeverityError marks something the library cannot mean: an operator
	// applied to types it has no meaning for, a name that resolves to nothing.
	SeverityError Severity = iota
	// SeverityWarning marks something legal and probably unintended — an
	// implicit conversion that loses information, a comparison that cannot be
	// true.
	SeverityWarning
)

func (s Severity) String() string {
	if s == SeverityWarning {
		return "warning"
	}
	return "error"
}

// Diagnostic is one thing the semantic phase found, and where.
type Diagnostic struct {
	Position ast.Position
	Severity Severity
	Message  string
	// Define names the statement being checked when the diagnostic was
	// produced. A position alone answers "where"; this answers "in what", which
	// is what a reader scanning twenty findings sorts by.
	Define string
}

func (d Diagnostic) String() string {
	var b strings.Builder
	if d.Position.Known() {
		fmt.Fprintf(&b, "%s: ", d.Position)
	}
	b.WriteString(d.Severity.String())
	b.WriteString(": ")
	if d.Define != "" {
		fmt.Fprintf(&b, "in %s: ", d.Define)
	}
	b.WriteString(d.Message)
	return b.String()
}

// Diagnostics is everything one pass found.
//
// The phase collects rather than returns at the first problem, which is the
// whole point of checking statically: an author who mistyped three element
// names wants all three, not the first one three runs in a row. This is what
// the reference translator's badExpression() recovery buys, arrived at by
// giving every failed inference the Unknown type and carrying on.
type Diagnostics []Diagnostic

// HasErrors reports whether anything found makes the library wrong.
func (d Diagnostics) HasErrors() bool {
	for _, diag := range d {
		if diag.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Errors returns just the errors, dropping the warnings.
func (d Diagnostics) Errors() Diagnostics {
	var out Diagnostics
	for _, diag := range d {
		if diag.Severity == SeverityError {
			out = append(out, diag)
		}
	}
	return out
}

// Error renders every diagnostic, one per line, so a Diagnostics can be
// returned where an error is expected.
func (d Diagnostics) Error() string {
	parts := make([]string, len(d))
	for i, diag := range d {
		parts[i] = diag.String()
	}
	return strings.Join(parts, "\n")
}

// sorted orders diagnostics by where they are, so that the output of a pass
// reads in source order however the checker happened to walk the tree.
// Positionless ones come last: they are the ones with nothing to sort by, and
// leading with them would bury the ones a reader can act on.
func (d Diagnostics) sorted() Diagnostics {
	out := make(Diagnostics, len(d))
	copy(out, d)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].Position, out[j].Position
		switch {
		case a.Known() != b.Known():
			return a.Known()
		case a.Line != b.Line:
			return a.Line < b.Line
		default:
			return a.Col < b.Col
		}
	})
	return out
}
