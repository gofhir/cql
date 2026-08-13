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
	Err      error
}

func (e *PositionedError) Error() string {
	if !e.Position.Known() {
		return e.Err.Error()
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
