package cql

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestSemanticPhaseAcceptsPublishedMeasures runs the semantic phase over real
// measure CQL, which is the measurement that decides whether refusing to
// evaluate a library that does not check out is defensible.
//
// It is skipped unless the content is on disk, because the libraries belong to
// cqframework and are not vendored here. To run it:
//
//	git clone --depth 1 https://github.com/cqframework/ecqm-content-r4 /tmp/ecqm
//	ECQM_CONTENT_DIR=/tmp/ecqm/input/cql go test -run PublishedMeasures ./...
//
// Why it is worth the trouble: this repo's own corpus is the conformance suite,
// which is loose expressions rather than libraries. Zero findings across it says
// nothing about a measure — includes, choice elements, backbone elements and
// FHIR-to-System conversions only appear once you have a library. Turning
// WithSemanticValidation on was justified against the wrong corpus once, and the
// libraries here reported 137 findings at the time.
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
