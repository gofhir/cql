package eval

import (
	"time"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/sema"
)

// situate gives a DateTime the offset the evaluation request assumes, where the
// value states none of its own.
//
// CQL says a DateTime written without an offset carries the request's, and until
// now the engine only applied that to literals. That left two populations of
// value: the ones a literal produced, which were placed, and the ones that came
// in any other way — from FHIR JSON, from a string conversion — which were not.
// The two compare correctly only while the comparison ignores the default, which
// fhirpath did until v1.9.1 and no longer does.
//
// So every door a DateTime comes through calls this. A stated offset is left
// alone, and nothing is written into the value: the default is remembered
// alongside, which is what keeps a literal printing as it was written.
func (e *Evaluator) situate(v fptypes.Value) fptypes.Value {
	dt, isDateTime := v.(fptypes.DateTime)
	if !isDateTime || dt.HasTZ() {
		return v
	}
	_, seconds := e.ctx.EvaluationTimestamp.Zone()
	return dt.WithDefaultOffset(time.Duration(seconds) * time.Second)
}

// situateRefinement places a boundary that is specified more finely than the value
// it was derived from.
//
// A derived value carries its source's offset rather than the request's, which is
// the distinction that keeps a value read from data from silently acquiring the
// offset of whoever asked, and funcs.HighBoundary does that copying. What it
// cannot do is give a boundary a frame its source never had:
// `HighBoundary(@2020-06-15T, 17)` refines a value with no hour into one with a
// millisecond, and there is no offset on the source to copy because a day does not
// have one.
//
// The result does need a frame — it names an instant now — and the only one there
// is belongs to the evaluation request. That is not the confusion the rule guards
// against: there is no frame of the source's to contradict, and "the last instant
// of the 15th" is a question about the asking party's day. Left unplaced it would
// sit in exactly the population of values this line of work exists to empty.
func (e *Evaluator) situateRefinement(source, out fptypes.Value, err error) (fptypes.Value, error) {
	if err != nil || out == nil {
		return out, err
	}
	if src, isDateTime := source.(fptypes.DateTime); isDateTime {
		if _, framed := src.EffectiveOffset(); framed {
			return out, nil // the source had a frame; funcs carried it across
		}
	}
	return e.situate(out), nil
}

// asPlannedType gives a value the type the semantic phase decided the expression
// has, where the two can disagree about a FHIR primitive.
//
// The model says what a FHIR primitive holds: FHIR.dateTime.value is
// System.DateTime, and FHIR.date.value is System.Date. That is the whole of the
// difference between FHIRHelpers' ToDate and ToDateTime, which have the same body
// — `value.value` — and differ only in the type of the parameter they take.
//
// This engine reads the JSON, where a dateTime and a date are both a string, and
// typed the value by its text: "2020-03-01" became a Date because it carried no
// time. So `Enc.period.start` came back a Date although Period.start is declared
// FHIR.dateTime, and one Period could have endpoints of two different types.
//
// It went unnoticed because the operators convert as they go — comparisons,
// `during`, `overlaps` and the duration operators all answer correctly across the
// boundary. The place it shows is where the type *is* the question:
//
//	start of Enc.period is DateTime   was false
//	start of Enc.period as DateTime   was null
//	if x is DateTime then … else …    took the other branch
//
// A DateTime carries its precision, so a dateTime written to the day stays
// written to the day: nothing is invented, the value is the same and only its
// type changes to the one the model declares.
func (e *Evaluator) asPlannedType(node ast.Expression, owner, member string, v fptypes.Value) fptypes.Value {
	date, isDate := v.(fptypes.Date)
	if !isDate {
		return v
	}
	if !e.declaredDateTime(node, owner, member) {
		return v
	}
	return asDateTime(date)
}

// AsDeclaredType gives a value the type a named FHIR element declares, for the
// caller that already knows which one it read.
//
// The choice-element path knows exactly that: resolving `Observation.effective`
// walks the declared branches until one names a field the JSON has, so the branch
// is in hand rather than inferred. It is a separate entry point because the path
// above has to ask two sources which type applies, and this one does not have to
// ask at all.
func AsDeclaredType(declared string, v fptypes.Value) fptypes.Value {
	date, isDate := v.(fptypes.Date)
	if !isDate || !holdsDateTime(sema.ParseTypeName(declared)) {
		return v
	}
	return asDateTime(date)
}

// asDateTime retypes a Date through its text, which is what carries the
// precision: a Date written to the year yields a DateTime to the year. Nothing is
// invented — the value is the same and only its type changes.
func asDateTime(date fptypes.Date) fptypes.Value {
	promoted, err := fptypes.NewDateTime(date.String())
	if err != nil {
		// Unreachable for a value that parsed as a Date; keeping the original is
		// the safe answer if it ever is not.
		return date
	}
	return promoted
}

// declaredDateTime asks both of the things that know: the phase, for the type it
// inferred for this expression, and the model, for the type it declares for this
// element.
//
// The model is asked as well because the phase can only answer about the library
// being evaluated. `start of Enc.period` reaches the value through the body of
// FHIRHelpers.ToInterval, which is CQL from another library and carries its own
// plan, so TypeOf finds nothing for those nodes. The element's declared type does
// not depend on who is asking.
func (e *Evaluator) declaredDateTime(node ast.Expression, owner, member string) bool {
	if e.ctx.Plan != nil && holdsDateTime(e.ctx.Plan.TypeOf(node)) {
		return true
	}
	if e.ctx.ModelInfo == nil || owner == "" || member == "" {
		return false
	}
	declared, ok := e.ctx.ModelInfo.ElementType(owner + "." + member)
	if !ok {
		return false
	}
	return holdsDateTime(sema.ParseTypeName(declared))
}

// holdsDateTime reports whether a type holds a System.DateTime.
//
// Both levels have to be named. The phase types `Enc.period.start` as
// FHIR.dateTime, the element's own type, and only `.value` on it is
// System.DateTime — while the evaluator does not wrap FHIR primitives, so it
// represents that FHIR.dateTime with a system value directly. Checking only for
// System.DateTime therefore matched nothing at the place the value is read.
//
// FHIR.instant is the same underlying value at a fixed precision, and its
// FHIRHelpers accessor is ToDateTime for that reason. FHIR.date stays a Date.
func holdsDateTime(t sema.Type) bool {
	named, ok := t.(*sema.Named)
	if !ok {
		return false
	}
	switch named.Model {
	case "System":
		return named.Name == "DateTime"
	case "FHIR":
		return named.Name == "dateTime" || named.Name == "instant"
	}
	return false
}
