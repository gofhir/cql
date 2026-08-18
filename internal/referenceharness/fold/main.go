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

// includeDef is one `include` the translator recorded, which is how an alias is
// resolved back to the library it names.
type includeDef struct {
	LocalIdentifier string `json:"localIdentifier"`
	Path            string `json:"path"`
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
			Includes struct {
				Def []includeDef `json:"def"`
			} `json:"includes"`
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

	result := expected{
		Translator:  *translator,
		Source:      *source,
		Definitions: definitionsOf(defs, helperAliasesOf(doc.Library.Includes.Def)),
	}
	if blank := unannotated(result.Definitions); len(blank) > 0 {
		fmt.Fprintf(os.Stderr,
			"refusing to write: %d definition(s) have no type or no locator: %s\n"+
				"the translator was probably run without EnableResultTypes or EnableLocators\n",
			len(blank), strings.Join(blank, ", "))
		os.Exit(1)
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

// helperAliasesOf reports the names this library can call FHIRHelpers by.
//
// A FunctionRef names the library by the alias the CQL gave it, so
// `include FHIRHelpers called FH` produces libraryName "FH". Matching the alias
// against "FHIRHelpers" found nothing and folded a probe written that way to zero
// conversions — a silently empty answer, which is the one thing a reference must
// not contain.
func helperAliasesOf(includes []includeDef) map[string]bool {
	aliases := map[string]bool{}
	for _, inc := range includes {
		if strings.Contains(inc.Path, "FHIRHelpers") {
			aliases[inc.LocalIdentifier] = true
		}
	}
	if len(aliases) == 0 {
		fmt.Fprintln(os.Stderr,
			"note: the library includes no FHIRHelpers, so no conversions will be recorded")
	}
	return aliases
}

// definitionsOf reduces each ELM definition to what is compared.
func definitionsOf(defs []map[string]any, helperAliases map[string]bool) []definition {
	out := make([]definition, 0, len(defs))
	for _, def := range defs {
		name, _ := def["name"].(string)
		expr, _ := def["expression"].(map[string]any)
		kind := "expression"
		if def["type"] == "FunctionDef" {
			kind = "function"
		}
		out = append(out, definition{
			Name:        name,
			Kind:        kind,
			ResultType:  typeName(def),
			Locator:     startOfLocator(expr),
			Conversions: conversionsIn(expr, helperAliases),
		})
	}
	return out
}

// unannotated names the definitions the translator recorded nothing about.
//
// A reference with nothing in it passes every comparison, which is worse than
// having none: the tests skip a definition whose type is empty, so a
// regeneration that lost the annotations would go green having compared nothing.
// The README's own warning is that dropping EnableResultTypes empties every
// result type, which puts this mistake within easy reach.
func unannotated(defs []definition) []string {
	var blank []string
	for _, def := range defs {
		if def.Name == "Patient" && def.ResultType == "" {
			continue // the implicit context definition, which has no expression
		}
		if def.ResultType == "" || def.Locator == "" {
			blank = append(blank, def.Name)
		}
	}
	return blank
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
// Sorted but not deduplicated. A definition with two FHIR operands needs two
// conversions — `Enc.id & Enc.status` needs ToString twice — and the engine side
// reports both, so folding them to one invented a divergence at every such
// definition.
func conversionsIn(node any, helperAliases map[string]bool) []string {
	// Empty rather than nil, so a definition with no conversions reads as [] in
	// the committed file instead of null.
	found := []string{}
	walk(node, helperAliases, &found)
	sort.Strings(found)
	return found
}

func walk(node any, helperAliases map[string]bool, found *[]string) {
	switch n := node.(type) {
	case map[string]any:
		if n["type"] == "FunctionRef" {
			if lib, _ := n["libraryName"].(string); helperAliases[lib] {
				if name, _ := n["name"].(string); name != "" {
					*found = append(*found, name)
				}
			}
		}
		// Sorted keys, so a walk of the same document twice finds the same
		// conversions in the same order and the fold is reproducible.
		keys := make([]string, 0, len(n))
		for k := range n {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			walk(n[k], helperAliases, found)
		}
	case []any:
		for _, v := range n {
			walk(v, helperAliases, found)
		}
	}
}
