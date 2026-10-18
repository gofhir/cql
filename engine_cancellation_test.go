package cql

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestACanceledContextIsReported covers a context already done when the
// evaluation starts. An expression short enough to finish before the
// evaluator's first check answered as if nobody had canceled it.
func TestACanceledContextIsReported(t *testing.T) {
	const src = "library T version '1.0'\n\ndefine X: 42\n"
	engine := NewEngine()
	parsed, err := engine.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, cancelExpired := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelExpired()

	entries := map[string]func(context.Context) error{
		"EvaluateExpression": func(ctx context.Context) error {
			_, err := engine.EvaluateExpression(ctx, src, "X", nil, nil)
			return err
		},
		"EvaluateLibrary": func(ctx context.Context) error {
			_, err := engine.EvaluateLibrary(ctx, src, nil, nil)
			return err
		},
		"EvaluateParsedExpression": func(ctx context.Context) error {
			_, err := engine.EvaluateParsedExpression(ctx, parsed, "X", nil, nil)
			return err
		},
		"EvaluateParsedLibrary": func(ctx context.Context) error {
			_, err := engine.EvaluateParsedLibrary(ctx, parsed, nil, nil)
			return err
		},
	}
	for name, evaluate := range entries {
		var asCanceled *ErrCanceled
		if err := evaluate(canceled); !errors.As(err, &asCanceled) || !errors.Is(err, context.Canceled) {
			t.Errorf("%s with a canceled context: %T %v; want an ErrCanceled of context.Canceled", name, err, err)
		}
		var asTimeout *ErrTimeout
		if err := evaluate(expired); !errors.As(err, &asTimeout) || !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("%s past its deadline: %T %v; want an ErrTimeout of context.DeadlineExceeded", name, err, err)
		}
	}
}

// TestACancellationDuringTheEvaluationIsReported covers a context canceled
// while the evaluation runs, here by the resolver of an include: what is left
// is too short for the evaluator to check, and its answer came back as if in
// time.
func TestACancellationDuringTheEvaluationIsReported(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine := NewEngine(WithLibraryResolver(func(_ context.Context, _, _ string) (string, error) {
		cancel()
		return "library H version '1.0'\n\ndefine A: 1\n", nil
	}))
	src := "library T version '1.0'\ninclude H version '1.0' called H\n\ndefine X: H.A + 41\n"
	got, err := engine.EvaluateExpression(ctx, src, "X", nil, nil)
	var asCanceled *ErrCanceled
	if !errors.As(err, &asCanceled) {
		t.Errorf("X = %v, %T %v; want an ErrCanceled", got, err, err)
	}
}

// TestAResolverThatStopsWithTheContextIsACancellation covers a resolver that
// gives up because the evaluation's context did: that was reported as a
// library nobody could resolve.
func TestAResolverThatStopsWithTheContextIsACancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine := NewEngine(WithLibraryResolver(func(ctx context.Context, _, _ string) (string, error) {
		cancel()
		return "", ctx.Err()
	}))
	src := "library T version '1.0'\ninclude H version '1.0' called H\n\ndefine X: H.A\n"
	_, err := engine.EvaluateExpression(ctx, src, "X", nil, nil)
	var asCanceled *ErrCanceled
	if !errors.As(err, &asCanceled) {
		t.Errorf("%T %v; want an ErrCanceled", err, err)
	}
}

// TestAResolversErrorIsReported covers a resolver that fails for a reason of
// its own. The reason was dropped, and the message said no resolver had been
// provided to a caller who had provided one.
func TestAResolversErrorIsReported(t *testing.T) {
	engine := NewEngine(WithLibraryResolver(func(context.Context, string, string) (string, error) {
		return "", errors.New("library store unavailable")
	}))
	src := "library T version '1.0'\ninclude H version '1.0' called H\n\ndefine X: H.A\n"
	_, err := engine.EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "library store unavailable") {
		t.Errorf("error = %v; want the resolver's reason", err)
	}
	if err != nil && strings.Contains(err.Error(), "no LibraryResolver provided") {
		t.Errorf("error = %v; a resolver was provided", err)
	}
}

// TestALongListOperationStops covers the loops that run long inside one node:
// a distinct, a set operation or a containment between lists of intervals
// compares each with every other, and did not look at the context until it
// was done.
func TestALongListOperationStops(t *testing.T) {
	const src = "library T version '1.0'\n\n" +
		"define Spans: expand { Interval[1, 20000] }\n" +
		"define Distinct: distinct Spans\n" +
		"define Union: Spans union Spans\n" +
		"define Intersect: Spans intersect Spans\n" +
		"define Except: Spans except Spans\n" +
		"define Includes: Spans includes Spans\n" +
		"define IncludedIn: Spans included in Spans\n" +
		"define SameAs: Spans same as Spans\n"
	engine := NewEngine()
	parsed, err := engine.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Distinct", "Union", "Intersect", "Except", "Includes", "IncludedIn", "SameAs"} {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		start := time.Now()
		_, err := engine.EvaluateParsedExpression(ctx, parsed, name, nil, nil)
		took := time.Since(start)
		cancel()
		var asTimeout *ErrTimeout
		if !errors.As(err, &asTimeout) {
			t.Errorf("%s: %T %v; want an ErrTimeout", name, err, err)
			continue
		}
		if took > time.Second {
			t.Errorf("%s stopped %v after a deadline of 20ms", name, took)
		}
		if asTimeout.Duration > 20*time.Millisecond {
			t.Errorf("%s: ErrTimeout.Duration = %v; the deadline that ended it was 20ms", name, asTimeout.Duration)
		}
	}
}
