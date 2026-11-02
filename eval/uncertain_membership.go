package eval

import (
	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/ast"
	cqltypes "github.com/gofhir/cql/types"
)

// An uncertainty is a point whose value is not known, not an interval of points.
//
// CQL 1.5.3, Language Semantics, Uncertainty: a comparison with an uncertainty
// is true if it holds for every value the uncertainty could be, false if it holds
// for none, and null otherwise. `U between a and b` is `U >= a and U <= b`, which
// is how the translator writes it, and `U in Interval[a, b)` is `U >= a and
// U < b`: the point is asked about each bound of the interval, open or closed,
// with three-valued `and`.
//
// The operators that take a point used to take an uncertainty for an ordinary
// interval, which is the shape it prints as: `between`, `in` and `contains`
// raised "cannot compare Interval", and `during`, `includes`, `before` and `in`
// over a list answered false where the answer was null. Everything here reads it
// as the point it is.

// uncertainRange reads a point or an uncertainty as the closed range of values it
// could be.
func uncertainRange(v fptypes.Value) cqltypes.Interval {
	if u, ok := v.(cqltypes.Uncertainty); ok {
		return u.AsInterval()
	}
	return cqltypes.NewInterval(v, v, true, true)
}

// uncertainCompare answers `a op b` for two operands that are each a point or an
// uncertainty, with op one of < <= > >=.
func uncertainCompare(a, b fptypes.Value, op ast.BinaryOp) (fptypes.Value, error) {
	return compareIntervalWithScalar(uncertainRange(a), b, op)
}

// allOfThree is `and` over three-valued answers: false if any is false, null if
// any is null, true otherwise.
func allOfThree(parts ...fptypes.Value) fptypes.Value {
	unknown := false
	for _, p := range parts {
		if p == nil {
			unknown = true
		} else if !isTrue(p) {
			return fptypes.NewBoolean(false)
		}
	}
	if unknown {
		return nil
	}
	return fptypes.NewBoolean(true)
}

// uncertainWithin answers whether a point or an uncertainty is in an interval,
// asking it of each bound with the closure the bound has. A null bound is
// unbounded, as it is for a point. properly asks for the interior: both bounds
// strict.
func uncertainWithin(x fptypes.Value, iv cqltypes.Interval, properly bool) (fptypes.Value, error) {
	var parts []fptypes.Value
	if iv.Low != nil {
		op := ast.OpGreaterOrEqual
		if !iv.LowClosed || properly {
			op = ast.OpGreater
		}
		part, err := uncertainCompare(x, iv.Low, op)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	if iv.High != nil {
		op := ast.OpLessOrEqual
		if !iv.HighClosed || properly {
			op = ast.OpLess
		}
		part, err := uncertainCompare(x, iv.High, op)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	return allOfThree(parts...), nil
}

// uncertainInList answers membership of a list by equality with each element: an
// uncertainty can be an element wherever their ranges meet, and cannot where
// they do not, which is null and false. A null element is not a match, as it is
// not for a point. properly asks for a list with another element besides.
func uncertainInList(x fptypes.Value, items cqltypes.List, properly bool) fptypes.Value {
	if properly && len(items.Values) <= 1 {
		return fptypes.NewBoolean(false)
	}
	unknown := false
	for _, item := range items.Values {
		if item == nil {
			continue
		}
		res, handled := uncertainEquality(x, item)
		if !handled || res == nil {
			unknown = true
		} else if isTrue(res) {
			return fptypes.NewBoolean(true)
		}
	}
	if unknown {
		return nil
	}
	return fptypes.NewBoolean(false)
}

// uncertainContained answers `x in container` for an uncertainty x and an
// interval or a list, or for a point x and an interval one of whose bounds is
// an uncertainty, and reports handled=false for anything else. The second is
// how `5 between U and 10` reaches here from ELM, as `5 in Interval[U, 10]`.
func uncertainContained(x, container fptypes.Value, properly bool) (fptypes.Value, bool, error) {
	switch c := container.(type) {
	case cqltypes.Interval:
		if !isUncertainty(x) && !holdsUncertainty(c) {
			return nil, false, nil
		}
		// Only a point is asked about the bounds; an interval inside an interval
		// with an uncertain bound is not a question about one number.
		if !pointOrUncertainty(x) {
			return nil, false, nil
		}
		res, err := uncertainWithin(x, c, properly)
		return res, true, err
	case cqltypes.List:
		if !isUncertainty(x) {
			return nil, false, nil
		}
		return uncertainInList(x, c, properly), true, nil
	}
	return nil, false, nil
}

// holdsUncertainty reports whether a value is an uncertainty or an interval one
// of whose bounds is one.
func holdsUncertainty(v fptypes.Value) bool {
	if iv, ok := v.(cqltypes.Interval); ok {
		return isUncertainty(iv.Low) || isUncertainty(iv.High)
	}
	return isUncertainty(v)
}

// uncertainTiming answers a timing phrase one of whose sides is an uncertainty,
// where it asks about a point: membership (`in`, `during`, `included in`,
// `includes`) and order (`before`, `after`, `same or before`, `same or after`).
// It reports handled=false for every other phrase, and for one that names a
// precision, a boundary or an offset, which are not questions about a whole
// number.
func uncertainTiming(op ast.TimingOp, left, right fptypes.Value) (fptypes.Value, bool, error) {
	if left == nil || right == nil || (!holdsUncertainty(left) && !holdsUncertainty(right)) {
		return nil, false, nil
	}
	if op.Precision != "" || op.Boundary != "" || op.Offset != "" {
		return nil, false, nil
	}
	switch op.Kind {
	case ast.TimingIncludedIn, ast.TimingDuring:
		return uncertainContained(left, right, op.Properly)
	case ast.TimingIncludes:
		return uncertainContained(right, left, op.Properly)
	case ast.TimingBeforeOrAfter, ast.TimingSameAs:
		return uncertainOrder(op, left, right)
	}
	return nil, false, nil
}

// uncertainOrder answers `before`, `after`, `same or before` and `same or after`
// between an uncertainty and a point, read as `<`, `>`, `<=` and `>=`. `same as`
// alone has no overload for two Integers: the reference translator cannot
// resolve it (plan 016), so it is left to the path it took before.
func uncertainOrder(op ast.TimingOp, left, right fptypes.Value) (fptypes.Value, bool, error) {
	if op.Kind == ast.TimingSameAs && !op.Before && !op.After {
		return nil, false, nil
	}
	if !pointOrUncertainty(left) || !pointOrUncertainty(right) {
		return nil, false, nil
	}
	var cmp ast.BinaryOp
	switch {
	case op.Kind == ast.TimingSameAs && op.Before:
		cmp = ast.OpLessOrEqual
	case op.Kind == ast.TimingSameAs:
		cmp = ast.OpGreaterOrEqual
	case op.Before:
		cmp = ast.OpLess
	default:
		cmp = ast.OpGreater
	}
	res, err := uncertainCompare(left, right, cmp)
	return res, true, err
}

// pointOrUncertainty reports whether a value is something a point comparison can
// be asked of: not an interval of the author's and not a list.
func pointOrUncertainty(v fptypes.Value) bool {
	switch v.(type) {
	case nil, cqltypes.Interval, cqltypes.List:
		return false
	}
	return true
}
