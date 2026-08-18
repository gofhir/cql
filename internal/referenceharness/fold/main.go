// Command fold reduces the reference translator's ELM output to the three things
// this engine is compared against, and writes it in the shape the tests read.
//
// Folding it by hand is what kept the probe at five definitions: every new one
// meant reading a locator out of a 45KB document and transcribing a type name.
// The comparison is only worth what it covers, so the transcription had to stop
// being the reason not to cover more.
//
// Usage:
//
//	fold -in elm.json -source probe.cql -out ../../testdata/reference/probe.expected.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// expected is the shape testdata/reference/*.expected.json is read in.
type expected struct {
	Translator  string       `json:"translator"`
	Source      string       `json:"source"`
	Definitions []definition `json:"definitions"`
}

type definition struct {
	Name string `json:"name"`
	// Kind is "expression" or "function". The translator reports both as
	// definitions and types both; this engine's sema.Result carries only the
	// expressions, so the comparison has to be able to tell them apart rather
	// than read a function as a definition we failed to type.
	Kind        string   `json:"kind"`
	ResultType  string   `json:"resultType"`
	Locator     string   `json:"locator"`
	Conversions []string `json:"conversions"`
}

func main() {
	in := flag.String("in", "", "path to the translator's ELM JSON")
	source := flag.String("source", "", "name of the CQL file it came from")
	out := flag.String("out", "", "path to write the folded reference to")
	translator := flag.String("translator", "info.cqframework:cql-to-elm 3.26.0",
		"translator coordinates, recorded so a diff can say which version decided")
	flag.Parse()
	if *in == "" || *source == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "-in, -source and -out are required")
		os.Exit(2)
	}

	raw, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %v\n", *in, err)
		os.Exit(1)
	}
	var doc struct {
		Library struct {
			Statements struct {
				Def []map[string]any `json:"def"`
			} `json:"statements"`
		} `json:"library"`
	}
	if unmarshalErr := json.Unmarshal(raw, &doc); unmarshalErr != nil {
		fmt.Fprintf(os.Stderr, "parsing %s: %v\n", *in, unmarshalErr)
		os.Exit(1)
	}
	defs := doc.Library.Statements.Def
	if len(defs) == 0 {
		// An ELM document with no statements means the translation produced
		// nothing to compare, and writing it would record an empty reference as
		// authoritative.
		fmt.Fprintln(os.Stderr, "no definitions in the ELM document")
		os.Exit(1)
	}

	result := expected{Translator: *translator, Source: *source}
	for _, def := range defs {
		name, _ := def["name"].(string)
		expr, _ := def["expression"].(map[string]any)
		kind := "expression"
		if def["type"] == "FunctionDef" {
			kind = "function"
		}
		result.Definitions = append(result.Definitions, definition{
			Name:        name,
			Kind:        kind,
			ResultType:  typeName(def),
			Locator:     startOfLocator(expr),
			Conversions: conversionsIn(expr),
		})
	}
	folded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "encoding: %v\n", err)
		os.Exit(1)
	}
	// The mode matters little — git records only the executable bit, and the
	// file is regenerated rather than shipped — so it takes the stricter one the
	// linter asks for.
	if err := os.WriteFile(*out, append(folded, '\n'), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "writing %s: %v\n", *out, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "%d definitions folded into %s\n", len(result.Definitions), *out)
}

// typeName renders the type a definition resulted in, the way the tests spell
// it: unqualified for a named type, and List<T> or Interval<T> for the wrappers.
func typeName(node map[string]any) string {
	if qname, ok := node["resultTypeName"].(string); ok && qname != "" {
		return localPart(qname)
	}
	spec, ok := node["resultTypeSpecifier"].(map[string]any)
	if !ok {
		return ""
	}
	return specifierName(spec)
}

func specifierName(spec map[string]any) string {
	switch spec["type"] {
	case "NamedTypeSpecifier":
		if qname, ok := spec["name"].(string); ok {
			return localPart(qname)
		}
	case "ListTypeSpecifier":
		if inner, ok := spec["elementType"].(map[string]any); ok {
			return "List<" + specifierName(inner) + ">"
		}
	case "IntervalTypeSpecifier":
		if inner, ok := spec["pointType"].(map[string]any); ok {
			return "Interval<" + specifierName(inner) + ">"
		}
	case "ChoiceTypeSpecifier":
		choices, _ := spec["choice"].([]any)
		parts := make([]string, 0, len(choices))
		for _, c := range choices {
			if m, ok := c.(map[string]any); ok {
				parts = append(parts, specifierName(m))
			}
		}
		return "Choice<" + strings.Join(parts, ", ") + ">"
	}
	return ""
}

// localPart drops the namespace from an ELM QName: {http://hl7.org/fhir}Encounter
// is Encounter, and {urn:hl7-org:elm-types:r1}Boolean is Boolean.
func localPart(qname string) string {
	if i := strings.LastIndex(qname, "}"); i >= 0 {
		return qname[i+1:]
	}
	return qname
}

// startOfLocator keeps the beginning of an ELM locator: "10:13-10:30" is 10:13,
// which is what an ast.Position is compared against.
func startOfLocator(expr map[string]any) string {
	locator, _ := expr["locator"].(string)
	if i := strings.Index(locator, "-"); i >= 0 {
		return locator[:i]
	}
	return locator
}

// conversionsIn collects the FHIRHelpers calls the translator inserted anywhere
// under a definition, which is what this engine applies at evaluation instead.
//
// Sorted and deduplicated: the comparison is about which conversions a definition
// needs, not how many times the tree mentions one or in what order a walk found
// them.
func conversionsIn(node any) []string {
	seen := map[string]bool{}
	walk(node, seen)
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func walk(node any, seen map[string]bool) {
	switch n := node.(type) {
	case map[string]any:
		if n["type"] == "FunctionRef" {
			if lib, _ := n["libraryName"].(string); strings.Contains(lib, "FHIRHelpers") {
				if name, _ := n["name"].(string); name != "" {
					seen[name] = true
				}
			}
		}
		for _, v := range n {
			walk(v, seen)
		}
	case []any:
		for _, v := range n {
			walk(v, seen)
		}
	}
}
