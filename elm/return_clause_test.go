package elm

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/compiler"
)

// TestReturnClauseIsDistinctUnlessItSaysSo covers the default ELM gives a return
// clause's distinct attribute: true, written out only as false, for `return
// all`. The reference translator emits `"return": {"expression": …}` for a plain
// `return X`, and decoding that absence as Go's false turned it into `return all`.
func TestReturnClauseIsDistinctUnlessItSaysSo(t *testing.T) {
	for _, tt := range []struct {
		doc  string
		want bool
	}{
		{`{"expression":{"type":"AliasRef","name":"X"}}`, true},
		{`{"distinct":true,"expression":{"type":"AliasRef","name":"X"}}`, true},
		{`{"distinct":false,"expression":{"type":"AliasRef","name":"X"}}`, false},
	} {
		var rc ReturnClause
		if err := json.Unmarshal([]byte(tt.doc), &rc); err != nil {
			t.Fatalf("%s: %v", tt.doc, err)
		}
		if rc.Distinct != tt.want {
			t.Errorf("%s decodes distinct=%t, want %t", tt.doc, rc.Distinct, tt.want)
		}
	}

	// Written out whatever its value: with omitempty, a false was left out and
	// read back as the default.
	b, err := json.Marshal(ReturnClause{Distinct: false})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"distinct":false`) {
		t.Errorf("return all encodes as %s, which reads back as distinct", b)
	}
}

// TestReturnClauseSurvivesTheRoundTrip takes `return` and `return all` from CQL
// through ELM JSON and back to the AST, the path an exported library takes.
func TestReturnClauseSurvivesTheRoundTrip(t *testing.T) {
	for _, tt := range []struct {
		expr string
		want bool
	}{
		{"({1, 1}) X return X", true},
		{"({1, 1}) X return distinct X", true},
		{"({1, 1}) X return all X", false},
	} {
		lib, err := compiler.Compile("library T version '1.0'\ndefine R: " + tt.expr)
		if err != nil {
			t.Fatalf("%s: %v", tt.expr, err)
		}
		node := TranslateExpression(lib.Statements[0].Expression)
		data, err := json.Marshal(node)
		if err != nil {
			t.Fatalf("%s: %v", tt.expr, err)
		}
		var back ExpressionNode
		if err := json.Unmarshal(data, &back); err != nil {
			t.Fatalf("%s: %v", tt.expr, err)
		}
		q, ok := ImportExpression(&back).(*ast.Query)
		if !ok || q.Return == nil {
			t.Fatalf("%s: imported as %T, not a query with a return clause", tt.expr, ImportExpression(&back))
		}
		if q.Return.Distinct != tt.want {
			t.Errorf("%s comes back distinct=%t, want %t (ELM: %s)", tt.expr, q.Return.Distinct, tt.want, data)
		}
		// All is the other half of the same fact, and the AST documents it as
		// set exactly when Distinct is not.
		if q.Return.All == q.Return.Distinct {
			t.Errorf("%s comes back with All=%t and Distinct=%t", tt.expr, q.Return.All, q.Return.Distinct)
		}
	}
}

// TestSeveralSourcesWriteTheirReturnClause covers a query over several sources
// with no return clause. Its result is a distinct Tuple of the aliases, and the
// ELM has to say so for an engine that runs it to answer as this one does.
func TestSeveralSourcesWriteTheirReturnClause(t *testing.T) {
	lib, err := compiler.Compile("library T version '1.0'\ndefine R: from ({1, 1}) A, ({2}) B")
	if err != nil {
		t.Fatal(err)
	}
	node := TranslateExpression(lib.Statements[0].Expression)
	if node.Return == nil {
		t.Fatal("no return clause written for a query over two sources")
	}
	if !node.Return.Distinct {
		t.Error("the return clause written is not distinct")
	}
	if e := node.Return.Expression; e == nil || e.Type != "Tuple" || len(e.Element) != 2 {
		t.Errorf("the return clause written is %+v, want a Tuple of the two aliases", e)
	}

	// An aggregate has no return clause, and gets none.
	lib, err = compiler.Compile("library T version '1.0'\ndefine R: from ({1}) A, ({2}) B aggregate S starting 0: S + A")
	if err != nil {
		t.Fatal(err)
	}
	if node := TranslateExpression(lib.Statements[0].Expression); node.Return != nil {
		t.Error("a return clause was written for an aggregate")
	}
}
