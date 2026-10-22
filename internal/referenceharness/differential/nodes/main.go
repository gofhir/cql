// Command nodes writes the ELM node the reference translator wrote for each
// definition of a corpus, one line per definition: its name, a tab and the
// node's type, or "rejected" where the translator wrote none.
//
// It reads the ELM that Evaluate.java -each writes, one library per
// definition, which leaves out the ones the translator refused; the corpus
// names them all. It keeps only what an oracle needs from it: which overload the
// translator resolved, In and Contains or IncludedIn and Includes. The ELM is
// megabytes; the overloads are a few kilobytes, and they are what the
// repository keeps (plan 006).
//
// Usage:
//
//	nodes -cql fhirmembership.cql -elm fhirmembership.elm.json > fhirmembership.nodes.tsv
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
)

type library struct {
	Library struct {
		Statements struct {
			Def []struct {
				Name       string `json:"name"`
				Expression struct {
					Type string `json:"type"`
				} `json:"expression"`
			} `json:"def"`
		} `json:"statements"`
	} `json:"library"`
}

func main() {
	corpus := flag.String("cql", "", "the corpus")
	path := flag.String("elm", "", "the ELM Evaluate.java -each wrote, a library per definition")
	flag.Parse()
	src, err := os.ReadFile(*corpus)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nodes:", err)
		os.Exit(1)
	}
	raw, err := os.ReadFile(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nodes:", err)
		os.Exit(1)
	}
	var libraries map[string]*library
	if err := json.Unmarshal(raw, &libraries); err != nil {
		fmt.Fprintln(os.Stderr, "nodes:", err)
		os.Exit(1)
	}
	for _, m := range define.FindAllStringSubmatch(string(src), -1) {
		name := m[1]
		node := "rejected"
		if lib := libraries[name]; lib != nil {
			for _, def := range lib.Library.Statements.Def {
				if def.Name == name {
					node = def.Expression.Type
				}
			}
		}
		fmt.Printf("%s\t%s\n", name, node)
	}
}

// define is a generated definition of a corpus.
var define = regexp.MustCompile(`(?m)^define (D\d+): `)
