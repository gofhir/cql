package ast

import "fmt"

// Position is where a node begins in the CQL source, one-based for both the
// line and the column.
//
// One-based columns are what the rest of the CQL world uses — the ELM locators
// the reference translator emits, and every editor — so a diagnostic here can be
// compared with one from there. ANTLR counts columns from zero, which is why the
// builder adds one when it stamps.
//
// The zero value means unknown, which is what a node built by hand — by a test,
// or by the ELM importer — carries. Diagnostics have to read as well without a
// position as with one, since not every node has come from parsed text.
type Position struct {
	Line int
	Col  int
}

// Pos returns the node's position. Every expression node embeds Position, so
// this is how a diagnostic asks a node where it came from.
func (p Position) Pos() Position { return p }

// Known reports whether a position was recorded.
func (p Position) Known() bool { return p.Line > 0 }

// String renders a position as line:col, or "" when unknown, so it can be
// concatenated into a message without a guard at every call site.
func (p Position) String() string {
	if !p.Known() {
		return ""
	}
	return fmt.Sprintf("%d:%d", p.Line, p.Col)
}

// setPos records where a node started. It is unexported and reached through the
// embedded Position, so the builder can stamp any node without knowing its type.
func (p *Position) setPos(q Position) { *p = q }

// Positioned is implemented by every node that knows where it came from.
type Positioned interface {
	Pos() Position
}

// positionSetter is what the builder uses to stamp a freshly built node.
type positionSetter interface {
	setPos(Position)
}

// SetPosition records where an expression began, when the node supports it.
// Reported as ok so a caller can tell "no position recorded" from "position
// zero".
func SetPosition(expr Expression, line, col int) bool {
	setter, ok := expr.(positionSetter)
	if !ok {
		return false
	}
	setter.setPos(Position{Line: line, Col: col})
	return true
}

// PositionOf returns where an expression began, and whether it knows.
func PositionOf(expr Expression) (Position, bool) {
	p, ok := expr.(Positioned)
	if !ok {
		return Position{}, false
	}
	pos := p.Pos()
	return pos, pos.Known()
}
