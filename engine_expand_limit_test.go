package cql

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofhir/cql/compiler"
	"github.com/gofhir/cql/eval"
)

// TestExpandPastItsLimitFails covers the size limit on expand. It used to be a
// guard that stopped at 10001 elements and returned them as the answer, so a
// week per minute had 10001 minutes and Count over it said so without an error.
// A limit that shortens a result in silence makes a wrong answer look like a
// right one; a retrieve past its limit already failed with ErrTooCostly, and an
// expand now does too.
func TestExpandPastItsLimitFails(t *testing.T) {
	evaluate := func(t *testing.T, e *Engine, expr string) (string, error) {
		t.Helper()
		v, err := e.EvaluateExpression(context.Background(), "library T version '1.0'\ndefine R: "+expr, "R", nil, nil)
		if err != nil {
			return "", err
		}
		return valueString(v), nil
	}
	tooCostly := func(t *testing.T, err error, what string) {
		t.Helper()
		var costly *ErrTooCostly
		if !errors.As(err, &costly) {
			t.Errorf("%s: err = %v, want ErrTooCostly", what, err)
		}
	}

	t.Run("the default limit admits exactly its size", func(t *testing.T) {
		got, err := evaluate(t, NewEngine(), "Count(expand Interval[1, 100000])")
		if err != nil || got != "100000" {
			t.Errorf("Count(expand Interval[1, 100000]) = %s, %v; want 100000", got, err)
		}
		_, err = evaluate(t, NewEngine(), "Count(expand Interval[1, 100001])")
		tooCostly(t, err, "one element past the default")
	})

	t.Run("a week per minute is every minute of it", func(t *testing.T) {
		got, err := evaluate(t, NewEngine(),
			"Count(expand { Interval[@2024-01-01T00:00:00Z, @2024-01-08T00:00:00Z) } per minute)")
		if err != nil || got != "10080" {
			t.Errorf("a week per minute = %s, %v; want 10080", got, err)
		}
	})

	t.Run("the list overload is limited as a whole", func(t *testing.T) {
		// Each interval is under the limit; the two together are past it.
		_, err := evaluate(t, NewEngine(), "expand { Interval[1, 60000], Interval[1, 60000] }")
		tooCostly(t, err, "two intervals of 60000")
	})

	t.Run("a configured limit holds inside a query", func(t *testing.T) {
		e := NewEngine(WithMaxExpandSize(10))
		if got, err := evaluate(t, e, "({1}) X return Count(expand Interval[1, 10])"); err != nil || got != "{10}" {
			t.Errorf("at the limit = %s, %v; want {10}", got, err)
		}
		_, err := evaluate(t, e, "({1}) X return Count(expand Interval[1, 11])")
		tooCostly(t, err, "one past a limit of 10, inside a query")
	})

	t.Run("zero disables it", func(t *testing.T) {
		got, err := evaluate(t, NewEngine(WithMaxExpandSize(0)), "Count(expand Interval[1, 200000])")
		if err != nil || got != "200000" {
			t.Errorf("unlimited = %s, %v; want 200000", got, err)
		}
	})
}

// TestExpandEndsWithoutTheLimit covers the walks the limit used to be the only
// thing stopping. With it disabled they have to end on their own, and each runs
// against a deadline so that one that does not fails here instead of hanging
// the suite.
func TestExpandEndsWithoutTheLimit(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		// A step under one unit truncates to none. The reference engine answers
		// the empty list; cql-execution, like this engine did, never returns.
		{"expand { Interval[@2024-01-01, @2024-01-03] } per 0.5 days", "{}"},
		{"expand Interval[@2024-01-01, @2024-01-03] per 0.5 days", "{}"},
		{"Count(expand { Interval[@2024-01-01T00:00:00Z, @2024-01-01T02:00:00Z] } per 0.5 hours)", "0"},
		// At the top of the Long range the next step wraps round below the
		// high bound, and the end of an interval with it.
		{"Count(expand Interval[9223372036854775805L, 9223372036854775807L])", "3"},
		{"Count(expand { Interval[9223372036854775805L, 9223372036854775807L] })", "3"},
		{"expand { Interval[9223372036854775804L, 9223372036854775807L] } per 2",
			"{Interval[9223372036854775804, 9223372036854775805], Interval[9223372036854775806, 9223372036854775807]}"},
		// A step wider than what is left: the same shape answers {} lower down,
		// and has to at the top too, where v + step - 1 wraps.
		{"expand Interval[2147483646, 2147483647] per 5", "{}"},
		{"expand Interval[9223372036854775806L, 9223372036854775807L] per 5", "{}"},
		// A fractional step over one unit is cut to its whole units, as the
		// reference engine cuts it: ten days, a day at a time.
		{"Count(expand Interval[@2024-01-01, @2024-01-10] per 1.5 days)", "10"},
	} {
		done := make(chan string, 1)
		go func() {
			v, err := NewEngine(WithMaxExpandSize(0)).EvaluateExpression(context.Background(),
				"library T version '1.0'\ndefine R: "+tt.expr, "R", nil, nil)
			if err != nil {
				done <- "ERROR: " + err.Error()
				return
			}
			done <- valueString(v)
		}()
		select {
		case got := <-done:
			if got != tt.want {
				t.Errorf("%s = %s, want %s", tt.expr, got, tt.want)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("%s did not end with the limit disabled", tt.expr)
		}
	}
}

// TestAContextWithoutTheEngineKeepsTheLimit covers an evaluation built straight
// from eval.NewContext. expand had a guard of its own before the limit existed,
// and such an evaluation was protected by it; it is protected by the default
// limit now, not left without one.
func TestAContextWithoutTheEngineKeepsTheLimit(t *testing.T) {
	lib, err := compiler.Compile("library T version '1.0'\ndefine R: Count(expand Interval[1, 100001])")
	if err != nil {
		t.Fatal(err)
	}
	_, err = eval.NewEvaluator(eval.NewContext(context.Background(), lib)).EvaluateExpression("R")
	if !errors.Is(err, eval.ErrMaxExpandSizeExceeded) {
		t.Errorf("err = %v, want ErrMaxExpandSizeExceeded", err)
	}
}
