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
// With -promotions it writes instead the number of ToDateTime nodes the
// translator wrote into each definition, which is the conversion of a Date to a
// DateTime it inserts where a Date is next to a DateTime (plan 019); the ELM of a
// corpus of a thousand definitions is megabytes, and the counts are what the
// repository keeps.
//
// Usage:
//
//	nodes -cql fhirmembership.cql -elm fhirmembership.elm.json > fhirmembership.nodes.tsv
//	nodes -promotions -cql mixeddates.cql -elm mixeddates.elm.json > mixeddates.promotions.tsv
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
)

// count is how many nodes of a type an ELM tree holds.
func count(tree any, nodeType string) int {
	n := 0
	switch v := tree.(type) {
	case map[string]any:
		if v["type"] == nodeType {
			n++
		}
		for _, child := range v {
			n += count(child, nodeType)
		}
	case []any:
		for _, child := range v {
			n += count(child, nodeType)
		}
	}
	return n
}

type library struct {
	Library struct {
		Statements struct {
			Def []struct {
				Name       string          `json:"name"`
				Expression json.RawMessage `json:"expression"`
			} `json:"def"`
		} `json:"statements"`
	} `json:"library"`
}

func main() {
	corpus := flag.String("cql", "", "the corpus")
	path := flag.String("elm", "", "the ELM Evaluate.java -each wrote, a library per definition")
	promotions := flag.Bool("promotions", false, "write the number of ToDateTime nodes of each definition")
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
				if def.Name != name {
					continue
				}
				if *promotions {
					var tree any
					if err := json.Unmarshal(def.Expression, &tree); err != nil {
						fmt.Fprintln(os.Stderr, "nodes:", err)
						os.Exit(1)
					}
					node = strconv.Itoa(count(tree, "ToDateTime"))
					continue
				}
				var expr struct {
					Type string `json:"type"`
				}
				if err := json.Unmarshal(def.Expression, &expr); err != nil {
					fmt.Fprintln(os.Stderr, "nodes:", err)
					os.Exit(1)
				}
				node = expr.Type
			}
		}
		fmt.Printf("%s\t%s\n", name, node)
	}
}

// define is a generated definition of a corpus.
var define = regexp.MustCompile(`(?m)^define (D\d+): `)
