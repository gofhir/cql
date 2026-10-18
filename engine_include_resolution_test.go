package cql

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/gofhir/cql/ast"
)

// resolverOf is a LibraryResolver over a fixed set of sources by name, that
// counts what it was asked for.
func resolverOf(libs map[string]string, asked map[string]int) LibraryResolver {
	return func(_ context.Context, name, version string) (string, error) {
		if asked != nil {
			asked[name+"/"+version]++
		}
		if s, ok := libs[name]; ok {
			return s, nil
		}
		return "", fmt.Errorf("no library %q", name)
	}
}

// TestAnIncludeIsTheLibraryItAskedFor covers a resolver that hands back
// another library than the one asked for. It was evaluated as the one asked
// for: version '1.0' for an include of '2.0', or a library of another name —
// which a resolver that searches by name prefix returns for `include Common`
// when only CommonFunctions is stored.
func TestAnIncludeIsTheLibraryItAskedFor(t *testing.T) {
	libs := map[string]string{
		"H":      "library H version '1.0'\n\ndefine V: 1\n",
		"Common": "library CommonFunctions version '1.0'\n\ndefine V: 1\n",
		"Anon":   "define V: 1\n",
	}
	engine := NewEngine(WithLibraryResolver(resolverOf(libs, nil)))
	for _, tt := range []struct {
		include string
		wantErr bool
	}{
		{"include H version '2.0' called L", true},
		{"include Common version '1.0' called L", true},
		{"include Anon called L", true},
		{"include H version '1.0' called L", false},
		// An include that names no version takes the one there is.
		{"include H called L", false},
	} {
		src := "library T version '1.0'\n" + tt.include + "\n\ndefine X: L.V\n"
		got, err := engine.EvaluateExpression(context.Background(), src, "X", nil, nil)
		if tt.wantErr != (err != nil) {
			t.Errorf("%s: X = %v, err %v; want an error: %t", tt.include, got, err, tt.wantErr)
		}
	}
}

// TestACircularIncludeIsRefused covers a library included back by its own
// includes. The library at the top was not on the walk, so the resolver was
// asked for it again and the copy it gave was read in its place: B's A.Y was
// the resolver's A, not the one evaluated.
func TestACircularIncludeIsRefused(t *testing.T) {
	libs := map[string]string{
		"A": "library A version '1.0'\ninclude B version '1.0' called B\n\ndefine Y: 'resolved'\n",
		"B": "library B version '1.0'\ninclude A version '1.0' called A\n\ndefine Z: A.Y\n",
		"C": "library C version '1.0'\ninclude C version '1.0' called C\n\ndefine Z: 1\n",
	}
	engine := NewEngine(WithLibraryResolver(resolverOf(libs, nil)))
	for _, src := range []string{
		"library A version '1.0'\ninclude B version '1.0' called B\n\ndefine Y: 'evaluated'\ndefine X: B.Z\n",
		"library T version '1.0'\ninclude B version '1.0' called B\n\ndefine X: B.Z\n",
		"library T version '1.0'\ninclude C version '1.0' called C\n\ndefine X: C.Z\n",
	} {
		got, err := engine.EvaluateExpression(context.Background(), src, "X", nil, nil)
		if err == nil || !strings.Contains(err.Error(), "circular") {
			t.Errorf("%q: X = %v, err %v; want a circular reference refused", strings.SplitN(src, "\n", 2)[0], got, err)
		}
	}

	// A library two others include is not a cycle, and is resolved once.
	asked := map[string]int{}
	diamond := NewEngine(WithLibraryResolver(resolverOf(map[string]string{
		"L": "library L version '1.0'\ninclude D version '1.0' called D\n\ndefine V: D.V\n",
		"R": "library R version '1.0'\ninclude D version '1.0' called D\n\ndefine V: D.V\n",
		"D": "library D version '1.0'\n\ndefine V: 1\n",
	}, asked)))
	src := "library T version '1.0'\ninclude L version '1.0' called L\ninclude R version '1.0' called R\n\ndefine X: L.V + R.V\n"
	got, err := diamond.EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err != nil || valueString(got) != "2" {
		t.Errorf("a diamond: X = %v, err %v; want 2", got, err)
	}
	if asked["D/1.0"] != 1 {
		t.Errorf("a diamond asked for D %d times; want once", asked["D/1.0"])
	}
}

// TestAnIncludedLibrarysModelIsChecked covers the using of an included
// library, which was not checked: one written for FHIR 5.0.0 was evaluated
// against R4, where the library that evaluates it would have been refused.
func TestAnIncludedLibrarysModelIsChecked(t *testing.T) {
	libs := map[string]string{"H": "library H version '1.0'\nusing FHIR version '5.0.0'\n\ndefine V: 1\n"}
	engine := NewEngine(WithLibraryResolver(resolverOf(libs, nil)))
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude H version '1.0' called H\n\ndefine X: H.V\n"
	got, err := engine.EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "5.0.0") {
		t.Errorf("X = %v, err %v; want the included library's FHIR 5.0.0 refused", got, err)
	}
}

// nothingLoader is a LibraryLoader with no libraries.
type nothingLoader struct{}

func (nothingLoader) LoadLibrary(context.Context, string, string) (*ast.Library, error) {
	return nil, nil
}

// TestALoaderIsNotConsultedForIncludes covers an engine given a LibraryLoader
// and no resolver: its includes are not resolved, and the message said no
// resolver was provided, which left the loader looking like the way to give
// one.
func TestALoaderIsNotConsultedForIncludes(t *testing.T) {
	engine := NewEngine(WithLibraryLoader(nothingLoader{}))
	src := "library T version '1.0'\ninclude H version '1.0' called H\n\ndefine X: H.V\n"
	_, err := engine.EvaluateExpression(context.Background(), src, "X", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "WithLibraryResolver") {
		t.Errorf("err = %v; want it to say includes go through WithLibraryResolver", err)
	}
}
