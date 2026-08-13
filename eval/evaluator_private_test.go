package eval

import (
	"context"
	"testing"

	"github.com/gofhir/cql/compiler"
)

// TestPrivateNotReachableThroughTheMemo covers the exported Evaluator, where a
// caller keeps one across several expressions. The access check has to sit
// above the memoized definitions: `Pub` reads `Hidden`, which caches it, and
// asking for `Hidden` afterwards used to hand it over.
func TestPrivateNotReachableThroughTheMemo(t *testing.T) {
	lib, err := compiler.Compile("library T version '1.0'\n\ndefine private Hidden: 42\ndefine Pub: Hidden + 1\n")
	if err != nil {
		t.Fatalf("compiling: %v", err)
	}

	for _, warmUp := range []string{"", "Pub", "*"} {
		ev := NewEvaluator(NewContext(context.Background(), lib))
		switch warmUp {
		case "Pub":
			if _, err := ev.EvaluateExpression("Pub"); err != nil {
				t.Fatalf("warming up with Pub: %v", err)
			}
		case "*":
			if _, err := ev.EvaluateLibrary(); err != nil {
				t.Fatalf("warming up with EvaluateLibrary: %v", err)
			}
		}
		if _, err := ev.EvaluateExpression("Hidden"); err == nil {
			t.Errorf("after warm-up %q, a private definition was returned", warmUp)
		}
	}
}
