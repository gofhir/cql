package conformance

import (
	"bufio"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheConformanceSuiteIsTheVendoredOne holds testdata/*.xml to
// testdata/SOURCE.tsv: every file listed once, none missing, none added, each
// with the git blob SHA-1 and the SHA-256 recorded when it was vendored. The
// blob SHA-1 is what upstream's git tree records for the same file, so `make
// conformance-provenance` can compare the manifest with
// cqframework/cql-tests at its commit: the manifest is checked here for the
// files, and there against where they came from.
func TestTheConformanceSuiteIsTheVendoredOne(t *testing.T) {
	recorded := readSource(t)
	files, err := filepath.Glob(filepath.Join("testdata", "*.xml"))
	if err != nil {
		t.Fatal(err)
	}
	present := map[string]bool{}
	for _, file := range files {
		name := filepath.Base(file)
		present[name] = true
		want, ok := recorded[name]
		if !ok {
			t.Errorf("%s is not in testdata/SOURCE.tsv", name)
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if got := blobSHA1(raw); got != want.blob {
			t.Errorf("%s: git blob %s, recorded %s", name, got, want.blob)
		}
		sum := sha256.Sum256(raw)
		if got := hex.EncodeToString(sum[:]); got != want.sha256 {
			t.Errorf("%s: SHA-256 %s, recorded %s", name, got, want.sha256)
		}
	}
	for name := range recorded {
		if !present[name] {
			t.Errorf("%s is in testdata/SOURCE.tsv and not in testdata", name)
		}
	}
}

type sourceEntry struct{ blob, sha256 string }

func readSource(t *testing.T) map[string]sourceEntry {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "SOURCE.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	entries := map[string]sourceEntry{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			t.Fatalf("SOURCE.tsv: %q is not file, git blob and SHA-256", line)
		}
		if _, dup := entries[fields[0]]; dup {
			t.Errorf("SOURCE.tsv lists %s twice", fields[0])
		}
		entries[fields[0]] = sourceEntry{blob: fields[1], sha256: fields[2]}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return entries
}

// blobSHA1 is the identity git gives a file's content.
func blobSHA1(content []byte) string {
	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00", len(content))
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}
