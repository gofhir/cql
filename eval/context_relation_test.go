package eval

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"
)

// idRecorder answers retrieves with a fixed set of resources, remembering what
// it was asked. It deliberately ignores every filter in the request, which is
// how a provider that has not implemented one behaves.
type idRecorder struct {
	last      RetrieveRequest
	resources []json.RawMessage
}

func (r *idRecorder) Retrieve(_ context.Context, req RetrieveRequest) ([]json.RawMessage, error) {
	r.last = req
	return r.resources, nil
}

func newRelatedEvaluator(dp DataProvider) *Evaluator {
	ctx := NewContext(context.Background(), nil)
	ctx.DataProvider = dp
	return NewEvaluator(ctx)
}

// TestResolveRelatedContextAsksForTheReferencedResource covers a retrieve that
// named no id at all: CodePath was "_id" with nothing to compare against, so
// the provider saw an unfiltered query and the first resource of that type in
// the database came back as though it were the referenced one.
func TestResolveRelatedContextAsksForTheReferencedResource(t *testing.T) {
	for _, tt := range []struct {
		name      string
		reference string
		wantIDs   []string
	}{
		{"relative", "Practitioner/pr2", []string{"pr2"}},
		{"absolute", "http://example.org/fhir/Practitioner/pr2", []string{"pr2"}},
		{"versioned", "Practitioner/pr2/_history/3", []string{"pr2"}},
		{"bare id", "pr2", []string{"pr2"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := &idRecorder{resources: []json.RawMessage{
				json.RawMessage(`{"resourceType":"Practitioner","id":"pr2"}`),
			}}
			got, err := newRelatedEvaluator(rec).ResolveRelatedContext("Practitioner", tt.reference)
			if err != nil {
				t.Fatalf("resolving: %v", err)
			}
			if len(rec.last.IDs) != 1 || rec.last.IDs[0] != tt.wantIDs[0] {
				t.Errorf("request IDs = %v, want %v", rec.last.IDs, tt.wantIDs)
			}
			if got == nil {
				t.Fatal("resolved to nothing, want the referenced Practitioner")
			}
			if s, ok := got.(fptypes.String); !ok || !strings.Contains(s.Value(), `"pr2"`) {
				t.Errorf("resolved to %v, want the pr2 resource", got)
			}
		})
	}
}

// TestResolveRelatedContextRefusesAnUnrelatedResource is the case that made the
// old behavior dangerous rather than merely wasteful. A provider that ignores
// the id filter returns whatever it has; the engine cannot make it filter, but
// it can check the id of what came back rather than hand over resources[0].
func TestResolveRelatedContextRefusesAnUnrelatedResource(t *testing.T) {
	rec := &idRecorder{resources: []json.RawMessage{
		json.RawMessage(`{"resourceType":"Practitioner","id":"someone-else"}`),
		json.RawMessage(`{"resourceType":"Practitioner","id":"another-one"}`),
	}}
	got, err := newRelatedEvaluator(rec).ResolveRelatedContext("Practitioner", "Practitioner/pr2")
	if err != nil {
		t.Fatalf("resolving: %v", err)
	}
	if got != nil {
		t.Errorf("resolved to %v, want nothing: none of these is pr2", got)
	}
}

// The right id of the wrong type is still the wrong resource.
func TestResolveRelatedContextRefusesTheWrongType(t *testing.T) {
	rec := &idRecorder{resources: []json.RawMessage{
		json.RawMessage(`{"resourceType":"Organization","id":"pr2"}`),
	}}
	got, err := newRelatedEvaluator(rec).ResolveRelatedContext("Practitioner", "Practitioner/pr2")
	if err != nil {
		t.Fatalf("resolving: %v", err)
	}
	if got != nil {
		t.Errorf("resolved to %v, want nothing: that is an Organization", got)
	}
}

// It finds the referenced resource among others rather than assuming position.
func TestResolveRelatedContextPicksTheMatchOutOfMany(t *testing.T) {
	rec := &idRecorder{resources: []json.RawMessage{
		json.RawMessage(`{"resourceType":"Practitioner","id":"pr1"}`),
		json.RawMessage(`{"resourceType":"Practitioner","id":"pr2"}`),
		json.RawMessage(`{"resourceType":"Practitioner","id":"pr3"}`),
	}}
	got, err := newRelatedEvaluator(rec).ResolveRelatedContext("Practitioner", "Practitioner/pr2")
	if err != nil {
		t.Fatalf("resolving: %v", err)
	}
	s, ok := got.(fptypes.String)
	if !ok {
		t.Fatalf("resolved to %T, want the pr2 resource", got)
	}
	if !strings.Contains(s.Value(), `"pr2"`) {
		t.Errorf("resolved to %s, want the pr2 resource", s.Value())
	}
}

// References that name nothing a retrieve by id can find resolve to nothing,
// and do not reach the provider at all: a contained reference asked as an id
// would match whatever the provider felt like returning.
func TestResolveRelatedContextSkipsUnqueryableReferences(t *testing.T) {
	for _, reference := range []string{"", "   ", "#contained", "urn:uuid:9f2a", "Practitioner?name=x"} {
		rec := &idRecorder{resources: []json.RawMessage{
			json.RawMessage(`{"resourceType":"Practitioner","id":"pr1"}`),
		}}
		got, err := newRelatedEvaluator(rec).ResolveRelatedContext("Practitioner", reference)
		if err != nil {
			t.Fatalf("resolving %q: %v", reference, err)
		}
		if got != nil {
			t.Errorf("reference %q resolved to %v, want nothing", reference, got)
		}
		if rec.last.ResourceType != "" {
			t.Errorf("reference %q reached the provider, want no query at all", reference)
		}
	}
}

// TestUnusableSubjectStopsTheRetrieve covers the case a caller cannot see: a
// context resource that yields no id produces the same unscoped retrieve as a
// deliberate population-level run, and returns every subject's data.
func TestUnusableSubjectStopsTheRetrieve(t *testing.T) {
	for _, tt := range []struct {
		name    string
		value   json.RawMessage
		wantErr bool
	}{
		{"no context resource at all", nil, false},
		{"empty context resource", json.RawMessage(``), false},
		{"resource without id", json.RawMessage(`{"resourceType":"Patient"}`), true},
		{"resource with empty id", json.RawMessage(`{"resourceType":"Patient","id":""}`), true},
		{"resource that does not parse", json.RawMessage(`{"resourceType":`), true},
		{"usable resource", json.RawMessage(`{"resourceType":"Patient","id":"p1"}`), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext(context.Background(), nil)
			ctx.StatementContext = "Patient"
			ctx.SetContextResource("Patient", tt.value)

			req := RetrieveRequest{ResourceType: "Condition"}
			err := ctx.applyRetrieveContext(&req)
			if tt.wantErr {
				if !errors.Is(err, ErrContextSubjectUnusable) {
					t.Fatalf("error = %v, want ErrContextSubjectUnusable", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// A population-level run — no context resource — still reaches the provider
// unscoped, because that is the query the caller asked for.
func TestPopulationRunStaysUnscoped(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	ctx.StatementContext = "Patient"

	req := RetrieveRequest{ResourceType: "Condition"}
	if err := ctx.applyRetrieveContext(&req); err != nil {
		t.Fatalf("population run: %v", err)
	}
	if req.Context != "" || req.ContextID != "" || req.ContextRelation != "" {
		t.Errorf("request carries a context (%q/%q/%q), want none",
			req.Context, req.ContextID, req.ContextRelation)
	}
}
