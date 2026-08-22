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

// uncertainOperands reports the uncertainties in a collection.
//
// Mixing them with plain numbers is refused, for the reason mixing Quantities
// with numbers is: the aggregate would have to decide what the bare number means
// alongside a range, and nothing in the collection says. Silently skipping either
// side is how these came to answer 0.
func uncertainOperands(c fptypes.Collection) (uncertainties []cqltypes.Uncertainty, found bool, err error) {
	var others int
	for _, item := range c {
		if item == nil {
			continue
		}
		u, ok := item.(cqltypes.Uncertainty)
		if !ok {
			others++
			continue
		}
		uncertainties = append(uncertainties, u)
	}
	if len(uncertainties) == 0 {
		return nil, false, nil
	}
	if others > 0 {
		return nil, true, fmt.Errorf(
			"cannot aggregate uncertain values together with %d value(s) that are not uncertain", others)
	}
	return uncertainties, true, nil
}

// sumUncertainties adds uncertainties the way the + operator does, so there is
// one policy on propagating a range rather than two.
func (e *Evaluator) sumUncertainties(uncertainties []cqltypes.Uncertainty) (fptypes.Value, error) {
	var total fptypes.Value = uncertainties[0]
	for _, u := range uncertainties[1:] {
		sum, err := e.evalArithmetic(ast.OpAdd, total, u)
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
func (e *Evaluator) avgUncertainties(uncertainties []cqltypes.Uncertainty) (fptypes.Value, error) {
	total, err := e.sumUncertainties(uncertainties)
	if err != nil || total == nil {
		return nil, err
	}
	count := fptypes.NewInteger(int64(len(uncertainties)))
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
