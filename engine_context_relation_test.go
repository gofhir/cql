package cql

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"

	"github.com/gofhir/cql/eval"
	"github.com/gofhir/cql/model"
)

// TestRetrieveSaysHowItRelatesToItsContext covers what an empty
// ContextSearchParam used to mean. Three different situations reported it, and
// two of them still need the retrieve scoped, so a provider reading only that
// field returned every patient's data for a Patient retrieve and for the types
// the model relates by a FHIRPath fragment.
func TestRetrieveSaysHowItRelatesToItsContext(t *testing.T) {
	patient := []byte(`{"resourceType":"Patient","id":"p1"}`)
	for _, tt := range []struct {
		resource  string
		wantKind  model.ContextRelationKind
		wantParam string
		wantExpr  string
	}{
		// The retrieve asks for the context type itself. "_id" is a real search
		// parameter, so even a provider that reads nothing but
		// ContextSearchParam scopes this to the one patient.
		{"Patient", model.ContextSelf, "_id", ""},

		// The ordinary case, unchanged.
		{"Condition", model.ContextBySearchParam, "patient", ""},
		{"Observation", model.ContextBySearchParam, "subject", ""},

		// A FHIRPath fragment is not a query, but saying so beats saying
		// nothing: the provider knows scoping is required and can fall back to
		// a compartment definition.
		{"AuditEvent", model.ContextByExpression, "", "where(resolve() is Patient)"},

		// Unrelated to any patient. Not scoping these is correct.
		{"Medication", model.ContextUnrelated, "", ""},
	} {
		t.Run(tt.resource, func(t *testing.T) {
			src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\n\ndefine X: [" + tt.resource + "]\n"
			rec := &requestRecorder{}
			if _, err := NewEngine(WithDataProvider(rec)).
				EvaluateExpression(context.Background(), src, "X", patient, nil); err != nil {
				t.Fatalf("evaluating: %v", err)
			}
			if rec.last.ContextRelation != tt.wantKind {
				t.Errorf("relation = %q, want %q", rec.last.ContextRelation, tt.wantKind)
			}
			if rec.last.ContextSearchParam != tt.wantParam {
				t.Errorf("search param = %q, want %q", rec.last.ContextSearchParam, tt.wantParam)
			}
			if rec.last.ContextExpression != tt.wantExpr {
				t.Errorf("expression = %q, want %q", rec.last.ContextExpression, tt.wantExpr)
			}
			// The subject travels either way: what changes is whether the
			// provider is being told to filter by it.
			if rec.last.ContextID != "p1" {
				t.Errorf("context id = %q, want p1", rec.last.ContextID)
			}
		})
	}
}

// TestUnusableContextSubjectIsAnError covers the run nobody could see was
// wrong. A context resource with no id scoped nothing, so the retrieve reached
// the provider looking exactly like a deliberate population-level query and
// came back with every patient's data.
func TestUnusableContextSubjectIsAnError(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\n\ndefine X: [Condition]\n"

	rec := &requestRecorder{}
	_, err := NewEngine(WithDataProvider(rec)).
		EvaluateExpression(context.Background(), src, "X", []byte(`{"resourceType":"Patient"}`), nil)
	if err == nil {
		t.Fatal("a Patient with no id evaluated silently, want an error")
	}
	if !errors.Is(err, eval.ErrContextSubjectUnusable) {
		t.Errorf("error = %v, want ErrContextSubjectUnusable", err)
	}
	if rec.last.ResourceType != "" {
		t.Errorf("the unscoped retrieve still reached the provider as %+v", rec.last)
	}

	// The deliberate population-level run — no context resource — keeps working.
	if _, err := NewEngine(WithDataProvider(rec)).
		EvaluateExpression(context.Background(), src, "X", nil, nil); err != nil {
		t.Fatalf("population-level run: %v", err)
	}
	if rec.last.ContextID != "" {
		t.Errorf("population run carries subject %q, want none", rec.last.ContextID)
	}
}

// emptyProvider answers every retrieve with nothing.
type emptyProvider struct{}

func (emptyProvider) Retrieve(_ context.Context, _ eval.RetrieveRequest) ([]json.RawMessage, error) {
	return nil, nil
}

// libSource builds a distinct library per n. The version differs, so each
// source hashes differently while staying the same size and shape.
func libSource(n int) string {
	return "library L version '1." + strconv.Itoa(n) + "'\n" +
		"using FHIR version '4.0.1'\n\ndefine X: 1 + 1\n"
}

// TestCompiledCacheEvicts covers a cache that only ever grew. The key is a hash
// of the source, so a server evaluating CQL that arrives over HTTP added an
// entry per distinct source it was ever sent and released none of them — memory
// exhaustion reachable by anyone who can reach the endpoint.
func TestCompiledCacheEvicts(t *testing.T) {
	engine := NewEngine(WithDataProvider(emptyProvider{}), WithCompiledCacheSize(4))
	for i := range 20 {
		if _, err := engine.EvaluateExpression(context.Background(), libSource(i), "X", nil, nil); err != nil {
			t.Fatalf("evaluating library %d: %v", i, err)
		}
	}
	if got := engine.compiledCache.len(); got != 4 {
		t.Errorf("cache holds %d parses, want 4", got)
	}
}

// The most recently used entries are the ones kept.
func TestCompiledCacheKeepsTheRecentlyUsed(t *testing.T) {
	engine := NewEngine(WithDataProvider(emptyProvider{}), WithCompiledCacheSize(2))
	ctx := context.Background()

	for _, i := range []int{0, 1} {
		if _, err := engine.EvaluateExpression(ctx, libSource(i), "X", nil, nil); err != nil {
			t.Fatalf("evaluating library %d: %v", i, err)
		}
	}
	// Touch 0 so 1 becomes the least recently used, then add 2.
	if _, err := engine.EvaluateExpression(ctx, libSource(0), "X", nil, nil); err != nil {
		t.Fatalf("re-evaluating library 0: %v", err)
	}
	if _, err := engine.EvaluateExpression(ctx, libSource(2), "X", nil, nil); err != nil {
		t.Fatalf("evaluating library 2: %v", err)
	}

	if _, ok := engine.compiledCache.load(sourceKey(libSource(0)), libSource(0)); !ok {
		t.Error("library 0 was evicted, want it kept: it was used most recently")
	}
	if _, ok := engine.compiledCache.load(sourceKey(libSource(1)), libSource(1)); ok {
		t.Error("library 1 is still cached, want it evicted as least recently used")
	}
}

// A zero size turns the cache off, for a process that evaluates each library
// once and should not hold any of them.
func TestCompiledCacheCanBeDisabled(t *testing.T) {
	engine := NewEngine(WithDataProvider(emptyProvider{}), WithCompiledCacheSize(0))
	for i := range 5 {
		if _, err := engine.EvaluateExpression(context.Background(), libSource(i), "X", nil, nil); err != nil {
			t.Fatalf("evaluating library %d: %v", i, err)
		}
	}
	if got := engine.compiledCache.len(); got != 0 {
		t.Errorf("cache holds %d parses with caching off, want 0", got)
	}
}

// A negative size restores the unbounded behavior, for a process whose set of
// libraries is closed and known.
func TestCompiledCacheCanBeUnbounded(t *testing.T) {
	engine := NewEngine(WithDataProvider(emptyProvider{}), WithCompiledCacheSize(-1))
	for i := range 20 {
		if _, err := engine.EvaluateExpression(context.Background(), libSource(i), "X", nil, nil); err != nil {
			t.Fatalf("evaluating library %d: %v", i, err)
		}
	}
	if got := engine.compiledCache.len(); got != 20 {
		t.Errorf("cache holds %d parses, want all 20", got)
	}
}

// Repeating one source is still one parse: the ceiling must not cost the reuse
// that the cache exists for.
func TestCompiledCacheStillReusesOneSource(t *testing.T) {
	engine := NewEngine(WithDataProvider(emptyProvider{}), WithCompiledCacheSize(4))
	src := libSource(1)
	for range 10 {
		if _, err := engine.EvaluateExpression(context.Background(), src, "X", nil, nil); err != nil {
			t.Fatalf("evaluating: %v", err)
		}
	}
	if got := engine.compiledCache.len(); got != 1 {
		t.Errorf("cache holds %d parses for one source, want 1", got)
	}
}
