package types_test

import (
	"errors"
	"fmt"
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

// TestTheTwoQuestionsDifferWhereTheyAreMeantTo pins what each predicate answers.
//
// They exist as two because two groups of callers want different things from
// "this comparison could not be made": most want null, and four want to retry at
// a shared precision or to treat the pair as two values. That distinction is the
// whole design, so it is asserted rather than left to the doc comments.
func TestTheTwoQuestionsDifferWhereTheyAreMeantTo(t *testing.T) {
	for _, tt := range []struct {
		what             string
		err              error
		undecidable      bool
		temporallyAmbig  bool
		incompatibleUnit bool
	}{
		{"nothing went wrong", nil, false, false, false},
		{"precisions differ", fptypes.ErrPrecisionMismatch, true, true, false},
		{"one side states an offset", fptypes.ErrOffsetMismatch, true, true, false},
		{"dimensions differ", fptypes.ErrIncompatibleUnits, true, false, true},
		// Wrapped, which is how every one of them actually arrives.
		{"wrapped precision", fmt.Errorf("comparing: %w", fptypes.ErrPrecisionMismatch), true, true, false},
		{"wrapped units", fmt.Errorf("%w: cm2 and cm", fptypes.ErrIncompatibleUnits), true, false, true},
		// Anything else is a failure, not an answer.
		{"an ordinary error", errors.New("cannot compare Object"), false, false, false},
		// The wording the two replaced copies matched by text. Nothing produces
		// it — not fhirpath v1.9.1, not this repository — and matching it was
		// matching nothing, so a value that merely says so is not undecidable.
		{"the obsolete wording alone", errors.New("ambiguous comparison"), false, false, false},
		{"an offset mentioned in prose", errors.New("bad timezone offset in literal"), false, false, false},
	} {
		t.Run(tt.what, func(t *testing.T) {
			if got := cqltypes.UndecidableComparison(tt.err); got != tt.undecidable {
				t.Errorf("UndecidableComparison = %v, want %v", got, tt.undecidable)
			}
			if got := cqltypes.AmbiguousTemporalComparison(tt.err); got != tt.temporallyAmbig {
				t.Errorf("AmbiguousTemporalComparison = %v, want %v", got, tt.temporallyAmbig)
			}
			if got := cqltypes.IncompatibleUnits(tt.err); got != tt.incompatibleUnit {
				t.Errorf("IncompatibleUnits = %v, want %v", got, tt.incompatibleUnit)
			}
		})
	}
}

// TestTheWideQuestionIsExactlyTheTwoNarrowOnes keeps the three from drifting into
// three separate readings, which is the state this replaced: one rule with two
// implementations that disagreed about two of its four rows.
func TestTheWideQuestionIsExactlyTheTwoNarrowOnes(t *testing.T) {
	for _, err := range []error{
		nil,
		fptypes.ErrPrecisionMismatch,
		fptypes.ErrOffsetMismatch,
		fptypes.ErrIncompatibleUnits,
		errors.New("something else"),
		fmt.Errorf("wrapped: %w", fptypes.ErrOffsetMismatch),
	} {
		want := cqltypes.AmbiguousTemporalComparison(err) || cqltypes.IncompatibleUnits(err)
		if got := cqltypes.UndecidableComparison(err); got != want {
			t.Errorf("for %v: the wide question is %v and its two halves are %v", err, got, want)
		}
	}
}
