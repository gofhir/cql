package cql

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestSemanticPhaseAcceptsPublishedMeasures runs the semantic phase over real
// measure CQL, which is the measurement that decides whether refusing to
// evaluate a library that does not check out is safe.
//
// It is skipped unless the content is on disk, so it runs when someone asks for
// it rather than on every build. To run it:
//
//	git clone --depth 1 https://github.com/cqframework/ecqm-content-r4 /tmp/ecqm
//	ECQM_CONTENT_DIR=/tmp/ecqm/input/cql go test -run PublishedMeasures ./...
//
// The 19 libraries there are CC0, so vendoring them is possible if this should
// ever become a build-time check.
//
// What runs in CI instead is engine_measure_shapes_test.go, which pins down the
// shapes these libraries are made of — a value set name containing a dot, a
// property read where a value set name goes, an inherited element on a backbone
// element, a primitive whose conversion is declared on its base, a Period in a
// set operation. Those are the forms this engine has actually got wrong, and the
// conformance corpus cannot see any of them: it is loose expressions, and a
// measure is a library. Zero findings across the conformance suite is what
// justified turning WithSemanticValidation on the first time, while these
// libraries reported 137.
func TestSemanticPhaseAcceptsPublishedMeasures(t *testing.T) {
	dir := os.Getenv("ECQM_CONTENT_DIR")
	if dir == "" {
		t.Skip("set ECQM_CONTENT_DIR to a checkout of cqframework/ecqm-content-r4's input/cql")
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.cql"))
	if err != nil {
		t.Fatalf("globbing %s: %v", dir, err)
	}
	if len(files) == 0 {
		t.Fatalf("no .cql files under %s", dir)
	}
	sort.Strings(files)

	engine := NewEngine()
	var findings int
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading %s: %v", file, err)
		}
		name := filepath.Base(file)
		diags, err := engine.Check(string(raw))
		if err != nil {
			t.Errorf("%s does not parse: %v", name, err)
			continue
		}
		for _, diag := range diags.Errors() {
			findings++
			t.Errorf("%s:%d in %s: %s", name, diag.Position.Line, diag.Define, diag.Message)
		}
	}
	t.Logf("checked %d published libraries, %d findings", len(files), findings)
}
