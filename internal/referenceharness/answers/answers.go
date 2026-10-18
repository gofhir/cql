// Package answers reads what the reference engines and this engine answer to a
// definition into one comparable form: a number by its value, a list by its
// elements, a FHIR value by its JSON read back. compare and the root package's
// TestAnswersMatchTheReferences share it, so that a run by hand and the test
// read an answer the same way.
//
// It does not go through anything of this engine's beyond evaluating the
// definition: an oracle that used its String() or its equality would inherit
// the defects it is there to find.
package answers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
)

// Answer is one engine's answer to a definition: a number, a boolean, a
// string, null, an error, a FHIR value, a list of those, or anything else as
// its text.
type Answer struct {
	Kind     string // "number", "boolean", "string", "null", "error", "fhir", "list", "other"
	Text     string
	Num      decimal.Decimal
	Panicked bool
}

// referenceValue is one value as the reference engines encode it.
type referenceValue struct {
	Type       string            `json:"type"`
	Value      json.RawMessage   `json:"value"`
	Unit       string            `json:"unit"`
	Elements   []json.RawMessage `json:"elements"`
	Low        json.RawMessage   `json:"low"`
	High       json.RawMessage   `json:"high"`
	LowClosed  bool              `json:"lowClosed"`
	HighClosed bool              `json:"highClosed"`
	Error      string            `json:"error"`
	Message    string            `json:"message"`
}

func Reference(raw json.RawMessage) Answer {
	if len(raw) == 0 || string(raw) == "null" {
		return Answer{Kind: "null", Text: "null"}
	}
	var v referenceValue
	if err := json.Unmarshal(raw, &v); err != nil {
		return Answer{Kind: "other", Text: string(raw)}
	}
	if v.Error != "" {
		return Answer{Kind: "error", Text: "error: " + v.Message}
	}
	// A FHIR value: a primitive as its value, a resource or an element as its
	// JSON, read back so that key order and spacing do not matter.
	if fhirType, ok := strings.CutPrefix(v.Type, "FHIR."); ok {
		var s string
		if json.Unmarshal(v.Value, &s) == nil {
			if strings.HasPrefix(strings.TrimSpace(s), "{") {
				return Answer{Kind: "fhir", Text: canonicalJSON([]byte(s))}
			}
			return Answer{Kind: "fhir", Text: s}
		}
		return Answer{Kind: "other", Text: fhirType + " " + string(v.Value)}
	}
	if v.Type == "List" {
		items := make([]Answer, len(v.Elements))
		for i, e := range v.Elements {
			items[i] = Reference(e)
		}
		return list(items)
	}
	if a, ok := referenceScalar(v); ok {
		return a
	}
	return Answer{Kind: "other", Text: string(raw)}
}

// referenceScalar is a reference value that is one of CQL's System scalars, or
// an interval of them.
func referenceScalar(v referenceValue) (Answer, bool) {
	var s string
	switch v.Type {
	case "Quantity":
		if json.Unmarshal(v.Value, &s) == nil {
			if d, err := decimal.NewFromString(s); err == nil {
				return quantity(d, v.Unit), true
			}
		}
	case "Date", "DateTime", "Time":
		if json.Unmarshal(v.Value, &s) == nil {
			return Answer{Kind: "temporal", Text: v.Type + " " + s}, true
		}
	case "Interval":
		return interval(Reference(v.Low), Reference(v.High), v.LowClosed, v.HighClosed), true
	case "Integer", "Long", "Decimal":
		if json.Unmarshal(v.Value, &s) == nil {
			if d, err := decimal.NewFromString(s); err == nil {
				return Answer{Kind: "number", Text: v.Type + " " + s, Num: d}, true
			}
		}
	case "Boolean":
		var b bool
		if json.Unmarshal(v.Value, &b) == nil {
			return Answer{Kind: "boolean", Text: fmt.Sprint(b)}, true
		}
	case "String":
		if json.Unmarshal(v.Value, &s) == nil {
			return Answer{Kind: "string", Text: strconv.Quote(s)}, true
		}
	}
	return Answer{}, false
}

// Value is Evaluate for a value already evaluated: an element of a
// list.
func Value(v fptypes.Value) Answer {
	if o, ok := v.(*fptypes.ObjectValue); ok {
		return Answer{Kind: "fhir", Text: canonicalJSON(o.Data())}
	}
	if fptypes.IsFHIRPrimitive(v) {
		return Answer{Kind: "fhir", Text: v.String()}
	}
	switch n := v.(type) {
	case fptypes.Quantity:
		return quantity(n.Value(), n.Unit())
	case fptypes.Date:
		return Answer{Kind: "temporal", Text: "Date " + n.String()}
	case fptypes.DateTime:
		return Answer{Kind: "temporal", Text: "DateTime " + n.String()}
	case fptypes.Time:
		return Answer{Kind: "temporal", Text: "Time " + n.String()}
	case cqltypes.Interval:
		return interval(Value(n.Low), Value(n.High), n.LowClosed, n.HighClosed)
	case nil:
		return Answer{Kind: "null", Text: "null"}
	case fptypes.Integer:
		return Answer{Kind: "number", Text: "Integer " + n.String(), Num: decimal.NewFromInt(n.Value())}
	case fptypes.Decimal:
		return Answer{Kind: "number", Text: "Decimal " + n.String(), Num: n.Value()}
	case fptypes.Boolean:
		return Answer{Kind: "boolean", Text: fmt.Sprint(n.Bool())}
	case fptypes.String:
		return Answer{Kind: "string", Text: strconv.Quote(n.Value())}
	case cqltypes.List:
		items := make([]Answer, len(n.Values))
		for i, e := range n.Values {
			items[i] = Value(e)
		}
		return list(items)
	}
	return Answer{Kind: "other", Text: v.Type() + " " + v.String()}
}

// list is a list of answers, as one text both engines' lists come to:
// numbers by value, so {1.0} is {1}, strings quoted, and nested lists alike.
// An element of another type — a tuple, an interval, a date — has a text of
// each engine's own, so it keeps the list "other": reported, never matched.
func list(items []Answer) Answer {
	parts := make([]string, len(items))
	kind := "list"
	for i, it := range items {
		switch it.Kind {
		case "number":
			parts[i] = it.Num.String()
		case "null", "boolean", "string", "fhir", "list", "quantity", "temporal", "interval":
			parts[i] = it.Text
		default:
			kind = "other"
			parts[i] = it.Text
		}
	}
	return Answer{Kind: kind, Text: "{" + strings.Join(parts, ", ") + "}"}
}

// Same compares numbers by value, since 1.0 = 1 in CQL, and everything else by
// kind and text, exactly: 'abc' and 'ABC' are two strings. Two errors are the
// same whatever their messages say.
//
// A Decimal a float64 computed — more significant digits than one holds for
// certain, on either side — is the same within floatTolerance of the other.
// Exp, Ln and a power to a fraction are computed in float64 by every engine
// here, and Go computes those differently by platform, in assembly on amd64
// and in Go on arm64: the same Exp was 183.09405819371815 on CI and
// 183.09405819371818 on a developer's machine, where the Java engine wrote the
// second. Compared exactly, which reference agreed depended on the machine.
func Same(a, b Answer) bool {
	switch {
	case a.Kind != b.Kind:
		return false
	case a.Kind == "number":
		if a.Num.Equal(b.Num) {
			return true
		}
		// Only a Decimal: an Integer or a Long is exact in every engine, and a
		// Long one past its range differs from the wrapped one by a hair of
		// its size.
		if !isDecimal(a) || !isDecimal(b) ||
			significantDigits(a.Num) <= floatDigits && significantDigits(b.Num) <= floatDigits {
			return false
		}
		scale := decimal.Max(a.Num.Abs(), b.Num.Abs())
		return a.Num.Sub(b.Num).Abs().LessThanOrEqual(scale.Mul(floatTolerance))
	case a.Kind == "error", a.Kind == "null":
		return true
	}
	return a.Text == b.Text
}

// canonicalJSON is a JSON document read and written back, with its keys
// sorted, so that two engines' encodings of one resource compare equal.
func canonicalJSON(raw []byte) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	out, err := json.Marshal(v)
	if err != nil {
		return string(raw)
	}
	return string(out)
}

// quantity is a quantity by its value as a number and its unit, as
// both engines write it.
func quantity(value decimal.Decimal, unit string) Answer {
	return Answer{Kind: "quantity", Text: value.String() + " '" + unit + "'"}
}

// interval is an interval of two answers, with its closed and open
// bounds, which a reader of either engine would be misled to drop. An
// interval of points neither engine can say more about stays "other".
func interval(low, high Answer, lowClosed, highClosed bool) Answer {
	open, closeBracket := "(", ")"
	if lowClosed {
		open = "["
	}
	if highClosed {
		closeBracket = "]"
	}
	kind := "interval"
	for _, b := range []Answer{low, high} {
		switch b.Kind {
		case "number", "null", "temporal", "quantity":
		default:
			kind = "other"
		}
	}
	text := func(b Answer) string {
		if b.Kind == "number" {
			return b.Num.String()
		}
		return b.Text
	}
	return Answer{Kind: kind, Text: "Interval" + open + text(low) + ", " + text(high) + closeBracket}
}

// Failed is the answer of a definition that failed to evaluate, or panicked.
func Failed(err error, panicked bool) Answer {
	if panicked {
		return Answer{Kind: "error", Text: fmt.Sprintf("PANIC: %v", err), Panicked: true}
	}
	return Answer{Kind: "error", Text: "error: " + err.Error()}
}

// floatDigits is the significant digits a float64 holds for certain; it writes
// up to 17.
const floatDigits = 15

// floatTolerance is how far apart, relative to the larger, two numbers a
// float64 computed may be and be the same answer.
var floatTolerance = decimal.New(1, -13)

func significantDigits(d decimal.Decimal) int {
	return len(strings.TrimLeft(d.Abs().Coefficient().String(), "0"))
}

func isDecimal(a Answer) bool { return strings.HasPrefix(a.Text, "Decimal ") }
