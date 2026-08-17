package cql

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestSemanticValidationRefusesALibraryThatDoesNotCheckOut covers the phase the
// specification puts before evaluation: semantic validation "is the process of
// verifying that the meaning of the expression is valid", and what it produces
// — a type for every expression — is what "guarantees that the expression is
// guaranteed to return either a value of that type, or a null".
//
// Without it that guarantee did not hold. Each of these evaluated to something.
func TestSemanticValidationRefusesALibraryThatDoesNotCheckOut(t *testing.T) {
	for _, tt := range []struct{ name, src, want string }{
		{
			// Evaluated to 1: the string was dropped and the answer was
			// indistinguishable from arithmetic that meant it.
			"an operator applied to types it does not accept",
			"library T version '1.0'\n\ndefine A: 1 + 'text'\n",
			"expected a String",
		},
		{
			"a name that is not defined",
			"library T version '1.0'\n\ndefine A: Bogus\n",
			"Bogus is not defined",
		},
		{
			"an element the model does not have",
			"library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\ndefine A: First([Encounter]).notAnElement\n",
			"has no element notAnElement",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEngine().EvaluateExpression(context.Background(), tt.src, "A", nil, nil)
			if err == nil {
				t.Fatal("the library evaluated, want it refused")
			}
			var semantic *ErrSemantic
			if !errors.As(err, &semantic) {
				t.Fatalf("error is %T, want *ErrSemantic: %v", err, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

// TestSemanticErrorsCarryTheirLocation covers what a caller needs to report the
// refusal: every finding, each with its position, rather than one message.
func TestSemanticErrorsCarryTheirLocation(t *testing.T) {
	src := `library T version '1.0'

define A: Bogus
define B: ({1,2}) X sort by nope
`
	_, err := NewEngine().EvaluateExpression(context.Background(), src, "A", nil, nil)
	var semantic *ErrSemantic
	if !errors.As(err, &semantic) {
		t.Fatalf("error is %T, want *ErrSemantic: %v", err, err)
	}
	if len(semantic.Diagnostics) != 2 {
		t.Fatalf("want both findings, got %d: %v", len(semantic.Diagnostics), semantic.Diagnostics)
	}
	for _, d := range semantic.Diagnostics {
		if !d.Position.Known() {
			t.Errorf("no position on %q", d.Message)
		}
		if d.Define == "" {
			t.Errorf("no definition named on %q", d.Message)
		}
	}
	// The sort key, not the query that contains it.
	if !strings.Contains(err.Error(), "4:29") {
		t.Errorf("want the sort key's position (4:29), got %v", err)
	}
}

// TestSemanticValidationCanBeTurnedOff covers the way out. The risk of refusing
// belongs to whoever runs other people's libraries, so they get to decide: with
// it off the evaluator works types out as it goes, which is what it did before.
func TestSemanticValidationCanBeTurnedOff(t *testing.T) {
	src := "library T version '1.0'\n\ndefine A: 1 + 'text'\n"

	got, err := NewEngine(WithSemanticValidation(false)).
		EvaluateExpression(context.Background(), src, "A", nil, nil)
	if err != nil {
		t.Fatalf("with validation off the library should still evaluate: %v", err)
	}
	if s := valueString(got); s != "1" {
		t.Errorf("= %s, want the 1 it answered before", s)
	}
}

// Warnings never block. ELM has them "not critical enough to prevent
// translation", and a library that only warns has to evaluate.
func TestWarningsDoNotBlockEvaluation(t *testing.T) {
	// A library with nothing wrong evaluates, and Check agrees there is nothing
	// of error severity to report.
	src := "library T version '1.0'\n\ndefine A: 1 + 1\n"
	diags, err := NewEngine().Check(src)
	if err != nil {
		t.Fatalf("checking: %v", err)
	}
	if errs := diags.Errors(); len(errs) != 0 {
		t.Fatalf("the library is sound; got %v", errs)
	}
	got, err := NewEngine().EvaluateExpression(context.Background(), src, "A", nil, nil)
	if err != nil {
		t.Fatalf("evaluating: %v", err)
	}
	if s := valueString(got); s != "2" {
		t.Errorf("= %s, want 2", s)
	}
}

// Parse and Check must stay out of the gate: their whole job is to hand back a
// library together with its findings, including one that will not evaluate.
func TestParseAndCheckStillAcceptABrokenLibrary(t *testing.T) {
	src := "library T version '1.0'\n\ndefine A: Bogus\n"

	parsed, err := NewEngine().Parse(src)
	if err != nil {
		t.Fatalf("Parse should accept a library that does not check out: %v", err)
	}
	if parsed == nil {
		t.Fatal("Parse returned nothing")
	}
	diags, err := NewEngine().Check(src)
	if err != nil {
		t.Fatalf("Check should accept it too: %v", err)
	}
	if len(diags.Errors()) != 1 {
		t.Errorf("want the one finding, got %v", diags)
	}
}

// The refusal reaches every entry point, including the ones that take an
// already-parsed library.
func TestSemanticValidationCoversTheParsedEntryPoints(t *testing.T) {
	engine := NewEngine()
	parsed, err := engine.Parse("library T version '1.0'\n\ndefine A: Bogus\n")
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	var semantic *ErrSemantic

	if _, err := engine.EvaluateParsedExpression(
		context.Background(), parsed, "A", nil, nil); !errors.As(err, &semantic) {
		t.Errorf("EvaluateParsedExpression: error is %T, want *ErrSemantic: %v", err, err)
	}
	if _, err := engine.EvaluateParsedLibrary(
		context.Background(), parsed, nil, nil); !errors.As(err, &semantic) {
		t.Errorf("EvaluateParsedLibrary: error is %T, want *ErrSemantic: %v", err, err)
	}
	if _, err := engine.EvaluateLibrary(
		context.Background(), "library T version '1.0'\n\ndefine A: Bogus\n", nil, nil); !errors.As(err, &semantic) {
		t.Errorf("EvaluateLibrary: error is %T, want *ErrSemantic: %v", err, err)
	}
}
