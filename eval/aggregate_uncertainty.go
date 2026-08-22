package eval

import (
	"fmt"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
	cqltypes "github.com/gofhir/cql/types"
)

// The aggregates did not understand an uncertainty, and the way they did not was
// the same dangerous one the Quantity aggregates had: they answered anyway.
//
//	Sum({ U, U })     was 0
//	Avg({ U, U })     was 0
//	Product({ U, U }) was 0
//
// U here is a duration the engine could not pin down — `months between
// DateTime(2005) and DateTime(2006, 7)` is somewhere in [6, 18] — which is
// exactly what a measure computes when a FHIR period carries a partial date. A
// total of 0 months reads as a real measurement wherever it lands.
//
// The cause was the same too: reading each element through a private decimal
// conversion that reports zero for anything that is not an Integer or a Decimal.
//
// The specification defines `+`, `-`, `*` and `/` on uncertainties, and the
// conformance corpus states the answers — `U + U` is `Interval[32, 88]`. So Sum
// and Avg have a rule to follow, and they follow it by asking the operator rather
// than by reimplementing it.
//
// It defines no aggregate over uncertainties at all. The rest therefore have no
// rule, and rather than invent one they say so.

// uncertainOperands reports the values of a collection that holds at least one
// uncertainty, and found=false when none of them is one.
//
// A plain number beside an uncertainty is kept rather than refused. A first
// version refused it, modeled on refusing a bare number beside a Quantity, and
// the two are not alike: a number next to a Quantity has no unit and the
// collection cannot supply one, while a certain number next to an uncertainty has
// an obvious sum — add it to both bounds, which is what `+` already does.
//
// The mistake could not show until the FHIR dateTime promotion landed. A Period
// whose endpoints were typed off their JSON text always yielded a certain
// duration, so Sum never saw a mixture; promote the endpoints and a patient with
// one complete encounter and one partially dated one — the ordinary case —
// produces one. CumulativeDays went from answering 0 to answering an error.
//
// What cannot be added is still refused, one operand at a time, by the operator
// that knows: sumUncertainties asks `+`, and `1 'mg' + 1 's'` is an error there
// already. Deciding it here would be a second policy on what may be summed.
func uncertainOperands(c fptypes.Collection) (values []fptypes.Value, found bool) {
	for _, item := range c {
		if item == nil {
			continue
		}
		if _, isUncertain := item.(cqltypes.Uncertainty); isUncertain {
			found = true
		}
		values = append(values, item)
	}
	if !found {
		return nil, false
	}
	return values, true
}

// sumUncertainties adds uncertainties the way the + operator does, so there is
// one policy on propagating a range rather than two.
func (e *Evaluator) sumUncertainties(values []fptypes.Value) (fptypes.Value, error) {
	total := values[0]
	for _, v := range values[1:] {
		sum, err := e.evalArithmetic(ast.OpAdd, total, v)
		if err != nil {
			return nil, err
		}
		if sum == nil {
			return nil, nil
		}
		total = sum
	}
	return total, nil
}

// avgUncertainties is the sum over the count, through the same operators.
func (e *Evaluator) avgUncertainties(values []fptypes.Value) (fptypes.Value, error) {
	total, err := e.sumUncertainties(values)
	if err != nil || total == nil {
		return nil, err
	}
	count := fptypes.NewInteger(int64(len(values)))
	return e.evalArithmetic(ast.OpDivide, total, count)
}

// undefinedOverUncertainty reports that an aggregate has no rule to follow. It is
// an error and not null deliberately: null is what an empty collection gives, and
// an author reading null cannot tell that the engine declined to answer a
// question it was asked.
func undefinedOverUncertainty(name string) error {
	return fmt.Errorf(
		"%s is not defined over uncertain values: %s of a duration between imprecise "+
			"dates has no answer in the specification, and answering with the bounds "+
			"read as numbers would report a total that was never measured", name, name)
}
