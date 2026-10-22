package cql

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// javaIdentity is a Java object's identity after its class name, as
// org.hl7.elm.r1.NamedTypeSpecifier@21e57990, a nested class's
// (Outer$Inner@…) or an array's ([Lorg.x.Y;@…): a hash of the run, not of
// the answer. A CQL date in a message, @2020-01-01, is not one.
var javaIdentity = regexp.MustCompile(`(?:[a-z][a-z0-9]*\.)+[A-Z][\w$]*;?@[0-9a-f]{1,8}(?:$|[^\w-])`)

// TestJavaIdentityReadsOnlyAnIdentity holds javaIdentity to messages written
// by hand.
func TestJavaIdentityReadsOnlyAnIdentity(t *testing.T) {
	for msg, want := range map[string]bool{
		"Cannot cast org.hl7.elm.r1.NamedTypeSpecifier@21e57990.":                                true,
		"Cannot cast org.hl7.elm.r1.NamedTypeSpecifier@21e57990":                                 true,
		"a org.opencds.Outer$Inner@1a2b in the list":                                             true,
		"a [Lorg.opencds.cqf.cql.engine.runtime.Code;@5f3c, not a Code":                          true,
		"Cannot compare org.x.DateTime@2020-01-01 to a Time":                                     false,
		"Invalid date @2020-01-01":                                                               false,
		"Cannot cast a value of type java.lang.Long as org.opencds.cqf.cql.engine.runtime.Time.": false,
	} {
		if got := javaIdentity.MatchString(msg); got != want {
			t.Errorf("%q: %v, want %v", msg, got, want)
		}
	}
}

// TestTheReferenceMessagesAreOfTheAnswer holds every error a reference file
// records: its message is not empty, and holds no Java object's identity,
// which would make the file differ from one run to the next and a verdict
// pinned to it unreproducible (plan 007).
func TestTheReferenceMessagesAreOfTheAnswer(t *testing.T) {
	var files []string
	for _, pattern := range []string{
		"testdata/snapshots/*.java.json", "testdata/snapshots/*.js.json",
		"testdata/reference/*.java.json", "testdata/reference/*.expected*.json",
		"testdata/membershipmodel/*.java.json",
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, matches...)
	}
	if len(files) < 20 {
		t.Fatalf("%d reference files: the check reads almost nothing", len(files))
	}
	errorsRead := 0
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var ref struct {
			Results map[string]json.RawMessage `json:"results"`
		}
		if err := json.Unmarshal(raw, &ref); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		for name, r := range ref.Results {
			var e map[string]json.RawMessage
			if json.Unmarshal(r, &e) != nil {
				continue
			}
			if _, isError := e["error"]; !isError {
				continue
			}
			errorsRead++
			var msg string
			switch {
			case json.Unmarshal(e["message"], &msg) != nil || msg == "":
				t.Errorf("%s %s: an error with no message, or one that is no text: %s", path, name, e["message"])
			case javaIdentity.MatchString(msg):
				t.Errorf("%s %s: the message holds a Java object's identity: %s", path, name, javaIdentity.FindString(msg))
			}
		}
	}
	if errorsRead == 0 {
		t.Fatal("no error read: the check holds nothing")
	}
}
