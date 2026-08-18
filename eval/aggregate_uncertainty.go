package eval

import (
	"fmt"

	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

// An uncertainty is an interval, and toDecimal answers zero for one. So an
// aggregate over durations the engine could not pin down came back as 0 — a
// plausible number, in the arithmetic of a continuous-variable measure, and
// indistinguishable from a real answer:
//
//	Sum({ duration in days of Enc.period })    was 0
//	Avg(…)                                     was 0
//	Median(…)                                  was null
//
// Arithmetic on an uncertainty already propagated it. The conformance corpus
// fixes `U + U` at Interval[32, 88] for a U of Interval[16, 44], and the engine
// agrees; the aggregates were the route that did not.

// uncertaintyOperands reports the uncertainties in a collection.
//
// Mixing them with plain numbers is refused rather than averaged. It is the same
// reason a Quantity beside a bare number is refused: nothing says which of the
// two the caller meant, and picking one silently is how an aggregate comes to
// answer confidently about data it did not read.
func uncertaintyOperands(c fptypes.Collection) (intervals []cqltypes.Interval, found bool, err error) {
	var others int
	for _, item := range c {
		if item == nil {
			continue
		}
		iv, ok := item.(cqltypes.Interval)
		if !ok {
			others++
			continue
		}
		intervals = append(intervals, iv)
	}
	if len(intervals) == 0 {
		return nil, false, nil
	}
	if others > 0 {
		return nil, true, fmt.Errorf(
			"cannot aggregate %d uncertain value(s) together with %d certain one(s)",
			len(intervals), others)
	}
	return intervals, true, nil
}

// sumUncertainties adds uncertainties the way the + operator does: the lowest
// each could be, and the highest.
func sumUncertainties(intervals []cqltypes.Interval) (fptypes.Value, error) {
	total := intervals[0]
	for _, iv := range intervals[1:] {
		total = addUncertainty(total, iv)
	}
	return total, nil
}

// avgUncertainties is the sum over the count, which keeps the width of the
// uncertainty proportional rather than collapsing it.
func avgUncertainties(intervals []cqltypes.Interval) (fptypes.Value, error) {
	total, err := sumUncertainties(intervals)
	if err != nil {
		return nil, err
	}
	sum, ok := total.(cqltypes.Interval)
	if !ok {
		return nil, nil
	}
	count := fptypes.NewInteger(int64(len(intervals)))
	return cqltypes.NewInterval(
		divideValues(sum.Low, count), divideValues(sum.High, count), true, true), nil
}

// medianUncertainties orders by what each uncertainty could least be, and takes
// the middle one whole. Averaging the two middle uncertainties on an even count
// would widen the answer past anything the data supports.
func medianUncertainties(intervals []cqltypes.Interval) fptypes.Value {
	sorted := make([]cqltypes.Interval, len(intervals))
	copy(sorted, intervals)
	for i := 1; i < len(sorted); i++ {
		current := sorted[i]
		j := i - 1
		for j >= 0 && lessThanValue(current.Low, sorted[j].Low) {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = current
	}
	return sorted[len(sorted)/2]
}

// addUncertainty adds two uncertainties bound by bound.
func addUncertainty(a, b cqltypes.Interval) cqltypes.Interval {
	return cqltypes.NewInterval(
		addValues(a.Low, b.Low), addValues(a.High, b.High), true, true)
}

// addValues adds two bounds, keeping Integer arithmetic where both sides are
// integers so that a count of days stays a count of days.
func addValues(a, b fptypes.Value) fptypes.Value {
	if a == nil || b == nil {
		return nil
	}
	ai, aok := a.(fptypes.Integer)
	bi, bok := b.(fptypes.Integer)
	if aok && bok {
		return fptypes.NewInteger(ai.Value() + bi.Value())
	}
	return newDecimalFromD(toDecimal(a).Add(toDecimal(b)))
}

func divideValues(a fptypes.Value, count fptypes.Integer) fptypes.Value {
	if a == nil || count.Value() == 0 {
		return nil
	}
	return newDecimalFromD(toDecimal(a).Div(toDecimal(count)))
}

func lessThanValue(a, b fptypes.Value) bool {
	if a == nil || b == nil {
		return false
	}
	return toDecimal(a).LessThan(toDecimal(b))
}
