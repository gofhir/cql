package eval

import (
	"errors"
	"fmt"

	"github.com/gofhir/cql/ast"
)

// PositionedError is an evaluation error that knows where in the CQL source it
// happened.
//
// Only the innermost node to fail attaches its position: the error travels up
// through every enclosing expression, and each of them adding its own would
// point at the whole statement rather than at the part that went wrong — and
// would grow the message once per level, which is how a recursion error once
// reached 380 KB.
type PositionedError struct {
	Position ast.Position
	// Library names the included library the position belongs to, when it is
	// not the one being evaluated. Without it a line number from an included
	// library reads as a line number of the caller's own source, and the reader
	// resolves it against the wrong text — a four-line library reporting an
	// error at line 8.
	Library string
	Err     error
}

func (e *PositionedError) Error() string {
	if !e.Position.Known() {
		return e.Err.Error()
	}
	if e.Library != "" {
		return fmt.Sprintf("%s %s: %s", e.Library, e.Position, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Position, e.Err)
}

func (e *PositionedError) Unwrap() error { return e.Err }

// withPosition attaches a node's position to an error that does not have one.
func withPosition(err error, expr ast.Expression) error {
	if err == nil {
		return nil
	}
	var positioned *PositionedError
	if errors.As(err, &positioned) {
		return err
	}
	pos, known := ast.PositionOf(expr)
	if !known {
		return err
	}
	return &PositionedError{Position: pos, Err: err}
}

// inLibrary marks an error as having come from an included library, so its
// position is read against that library's source rather than the caller's.
// Only the innermost crossing records it: the name that matters is where the
// line number is, not every library the error passed back through.
func inLibrary(err error, name string) error {
	if err == nil || name == "" {
		return err
	}
	var positioned *PositionedError
	if !errors.As(err, &positioned) || positioned.Library != "" {
		return err
	}
	positioned.Library = name
	return err
}
