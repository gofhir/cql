package cql

import (
	"context"
	"errors"
	"testing"
)

// TestCheckContextChecksTheIncludedLibraries covers a check of the whole
// include graph. Check reports the library it is given and nothing it
// includes; CheckContext reports both, each finding with its library.
func TestCheckContextChecksTheIncludedLibraries(t *testing.T) {
	libs := map[string]string{
		"H": wrongHelper,
		"M": "library M version '1.0'\ninclude H version '1.0' called H\ninclude Gone version '1.0' called G\n\ndefine Right: H.Right\n",
	}
	engine := NewEngine(WithLibraryResolver(resolverOf(libs, nil)))
	src := "library T version '1.0'\ninclude H version '1.0' called H\n\ndefine Own: 1 + 'y'\ndefine X: H.Right\n"

	own, err := engine.Check(src)
	if err != nil || len(own.Errors()) != 1 {
		t.Fatalf("Check = %v, %v; want the one error of the library given", own, err)
	}

	diags, err := engine.CheckContext(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	errs := diags.Errors()
	if len(errs) != 2 || errs[0].Library != nil || errs[1].Library == nil || errs[1].Library.Name != "H" {
		t.Fatalf("CheckContext = %v; want T's own error, then H's", diags)
	}

	// An include that cannot be resolved is a finding where it is written: in
	// the library given, or in the included library that writes it.
	for _, tt := range []struct {
		name, src, from string
		line            int
	}{
		{"in the library given", "library T version '1.0'\n\ninclude Gone version '1.0' called G\n\ndefine X: 1\n", "", 3},
		{"in an included library", "library T version '1.0'\ninclude M version '1.0' called M\n\ndefine X: 1\n", "M", 3},
	} {
		diags, err := engine.CheckContext(context.Background(), tt.src)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		var found bool
		for _, d := range diags.Errors() {
			from := ""
			if d.Library != nil {
				from = d.Library.Name
			}
			if from == tt.from && d.Position.Line == tt.line && d.Position.Col == 1 {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: diagnostics = %v; want an error at %s line %d", tt.name, diags, tt.from, tt.line)
		}
	}
}

// TestCheckContextStopsWithItsContext covers CheckContext's resolver calls,
// which a caller that has gone does not wait for: canceled before the check,
// or while the resolver is asked and answers all the same.
func TestCheckContextStopsWithItsContext(t *testing.T) {
	const src = "library T version '1.0'\ninclude H version '1.0' called H\n\ndefine X: H.Right\n"
	before, cancelBefore := context.WithCancel(context.Background())
	cancelBefore()
	engine := NewEngine(WithLibraryResolver(resolverOf(map[string]string{"H": wrongHelper}, nil)))
	_, err := engine.CheckContext(before, src)
	var canceled *ErrCanceled
	if !errors.As(err, &canceled) {
		t.Errorf("canceled before: err = %T %v; want an ErrCanceled", err, err)
	}

	during, cancelDuring := context.WithCancel(context.Background())
	defer cancelDuring()
	answering := NewEngine(WithLibraryResolver(func(context.Context, string, string) (string, error) {
		cancelDuring()
		return wrongHelper, nil
	}))
	diags, err := answering.CheckContext(during, src)
	if !errors.As(err, &canceled) {
		t.Errorf("canceled while resolving: %v, err %T %v; want an ErrCanceled", diags, err, err)
	}
}

// TestCheckContextAcceptsFHIRHelpers covers the include nearly every FHIR
// library has: the built-in FHIRHelpers has nothing to report.
func TestCheckContextAcceptsFHIRHelpers(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1' called FH\n\ndefine X: 1\n"
	diags, err := NewEngine().CheckContext(context.Background(), src)
	if err != nil || len(diags) != 0 {
		t.Errorf("CheckContext = %v, %v; want nothing", diags, err)
	}
}

// TestAnIncludedLibrarysWarningsDoNotBlock covers a warning in an included
// library: CheckContext reports it with its library, and the evaluation does
// not refuse it, as it does not refuse the evaluated library's own.
func TestAnIncludedLibrarysWarningsDoNotBlock(t *testing.T) {
	libs := map[string]string{"W": "library W version '1.0'\n\ndefine Never: 1 = 'a'\ndefine V: 3\n"}
	engine := NewEngine(WithLibraryResolver(resolverOf(libs, nil)))
	src := "library T version '1.0'\ninclude W version '1.0' called W\n\ndefine X: W.V\n"
	diags, err := engine.CheckContext(context.Background(), src)
	if err != nil || len(diags) != 1 || diags.HasErrors() || diags[0].Library == nil || diags[0].Library.Name != "W" {
		t.Errorf("CheckContext = %v, %v; want W's one warning", diags, err)
	}
	if got, err := engine.EvaluateExpression(context.Background(), src, "X", nil, nil); err != nil || valueString(got) != "3" {
		t.Errorf("X = %v, err %v; want 3", got, err)
	}
}
