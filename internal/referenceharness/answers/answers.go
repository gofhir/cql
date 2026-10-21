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
	"maps"
	"slices"
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
	Kind     string // "number", "boolean", "string", "null", "error", "refused", "fhir", "list", "structure", "untyped", "other"
	Text     string
	Num      decimal.Decimal
	Panicked bool
	// Untyped is a value whose type the reference did not write, only what
	// is known of it: a Number, a Text or an Object of cql-execution.
	Untyped bool
	// Parts are a list's elements, an interval's bounds, or a structure's
	// fields, named by Names in their order.
	Parts []Answer
	Names []string
	// form is "list" or "interval" for an answer list or interval built,
	// whatever its Kind says of its elements.
	form string
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
	// A Code's, a Concept's and a tuple's own: each written plain, a Code's
	// fields as JSON strings or null.
	Fields  map[string]json.RawMessage `json:"fields"`
	Codes   []json.RawMessage          `json:"codes"`
	Code    json.RawMessage            `json:"code"`
	System  json.RawMessage            `json:"system"`
	Version json.RawMessage            `json:"version"`
	Display json.RawMessage            `json:"display"`
}

func Reference(raw json.RawMessage) Answer {
	if len(raw) == 0 || string(raw) == "null" {
		return Answer{Kind: "null", Text: "null"}
	}
	var v referenceValue
	if err := json.Unmarshal(raw, &v); err != nil {
		return Answer{Kind: "other", Text: string(raw)}
	}
	if v.Error == "Translation" {
		return Answer{Kind: "refused", Text: "error: " + v.Message}
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
	if a, ok := referenceStructure(v); ok {
		return a
	}
	return Answer{Kind: "other", Text: string(raw)}
}

// referenceStructure is a reference value that is a Code, a Concept or a
// tuple, or one the reference could not type: cql-execution keeps Integer and
// Decimal as one number, String and Long as one text, and a tuple or an
// instance read as Any as a plain object.
func referenceStructure(v referenceValue) (Answer, bool) {
	var s string
	switch v.Type {
	case "Code":
		return code(plainString(v.Code), plainString(v.Display), plainString(v.System), plainString(v.Version)), true
	case "Concept":
		if v.Codes == nil {
			// codes null, which is not codes empty.
			return concept(Answer{Kind: "null", Text: "null"}, plainString(v.Display)), true
		}
		codes := make([]Answer, len(v.Codes))
		for i, c := range v.Codes {
			codes[i] = Reference(c)
		}
		return concept(list(codes), plainString(v.Display)), true
	case "Tuple", "Object":
		names := slices.Sorted(maps.Keys(v.Fields))
		fields := make([]Answer, len(names))
		for i, n := range names {
			fields[i] = Reference(v.Fields[n])
		}
		a := structure(v.Type, names, fields)
		a.Untyped = v.Type == "Object"
		return a, true
	case "Number":
		if json.Unmarshal(v.Value, &s) == nil {
			if d, err := decimal.NewFromString(s); err == nil {
				return Answer{Kind: "untyped", Text: "Number " + s, Num: d, Untyped: true}, true
			}
		}
	case "Text":
		if json.Unmarshal(v.Value, &s) == nil {
			return Answer{Kind: "untyped", Text: "Text " + strconv.Quote(s), Untyped: true}, true
		}
	}
	return Answer{}, false
}

// plainString is a field the reference writes as a JSON string or null.
func plainString(raw json.RawMessage) Answer {
	var s *string
	if json.Unmarshal(raw, &s) != nil || s == nil {
		return Answer{Kind: "null", Text: "null"}
	}
	return Answer{Kind: "string", Text: strconv.Quote(*s)}
}

// code is a Code by its four fields, each a string or null.
func code(codeField, display, system, version Answer) Answer {
	return structure("Code", []string{"code", "display", "system", "version"}, []Answer{codeField, display, system, version})
}

func concept(codes, display Answer) Answer {
	return structure("Concept", []string{"codes", "display"}, []Answer{codes, display})
}

// structure is a value of a type by its named fields, sorted by name.
func structure(typ string, names []string, fields []Answer) Answer {
	parts := make([]string, len(names))
	for i, n := range names {
		parts[i] = n + ": " + partText(fields[i])
	}
	return Answer{Kind: "structure", Text: typ + "{" + strings.Join(parts, ", ") + "}", Parts: fields, Names: names}
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
	}
	if a, ok := partsValue(v); ok {
		return a
	}
	return Answer{Kind: "other", Text: v.Type() + " " + v.String()}
}

// partsValue is Value for a value of parts: a list, a Code, a Concept or a
// tuple.
func partsValue(v fptypes.Value) (Answer, bool) {
	switch n := v.(type) {
	case cqltypes.List:
		items := make([]Answer, len(n.Values))
		for i, e := range n.Values {
			items[i] = Value(e)
		}
		return list(items), true
	case cqltypes.Code:
		return code(stringField(n.Code), stringField(n.Display), stringField(n.System), stringField(n.Version)), true
	case cqltypes.Concept:
		codes := make([]Answer, len(n.Codes))
		for i, c := range n.Codes {
			codes[i] = Value(c)
		}
		return concept(list(codes), stringField(n.Display)), true
	case cqltypes.Tuple:
		names := slices.Sorted(maps.Keys(n.Elements))
		fields := make([]Answer, len(names))
		for i, name := range names {
			fields[i] = Value(n.Elements[name])
		}
		return structure(n.Type(), names, fields), true
	}
	return Answer{}, false
}

// stringField is an element of this engine's Code or Concept: nil is null,
// and the empty String is a String (plan 004).
func stringField(s *string) Answer {
	if s == nil {
		return Answer{Kind: "null", Text: "null"}
	}
	return Answer{Kind: "string", Text: strconv.Quote(*s)}
}

// list is a list of answers, as one text both engines' lists come to:
// numbers by value and kind — see numberText — so {1.0} is {1.00} and not {1},
// strings quoted, and nested lists alike.
// An element of another type — a tuple, an interval, a date — has a text of
// each engine's own, so it keeps the list "other": reported, never matched.
func list(items []Answer) Answer {
	parts := make([]string, len(items))
	kind := "list"
	for i, it := range items {
		switch it.Kind {
		case "number", "null", "boolean", "string", "fhir", "list", "quantity", "temporal", "interval", "structure", "untyped":
		default:
			kind = "other"
		}
		parts[i] = partText(it)
	}
	return Answer{Kind: kind, Text: "{" + strings.Join(parts, ", ") + "}", Parts: items, form: "list"}
}

// partText is an answer inside a list, an interval or a structure.
func partText(a Answer) string {
	if a.Kind == "number" {
		return numberText(a)
	}
	return a.Text
}

// Same compares numbers by value within their kind, and everything else by kind
// and text, exactly: 'abc' and 'ABC' are two strings. Two errors are the same
// whatever their messages say.
//
// A number's kind is Decimal or whole. 1.0 and 1.00 are the same Decimal, which
// the two engines write at different scales; the Integer 3 and the Decimal 3.0
// are not the same answer, though CQL's equality says they are equal: compared
// by value alone, `7.0 div 2` answering an Integer went unseen. An Integer and
// a Long are both whole, a known loss — this engine has no Long at run time
// (D4c), so telling them apart would report every Long it computes.
//
// A Decimal a float64 computed — more significant digits than one holds for
// certain, on either side — is the same within floatTolerance of the other.
// Exp, Ln and a power to a fraction are computed in float64 by every engine
// here, and Go computes those differently by platform, in assembly on amd64
// and in Go on arm64: the same Exp was 183.09405819371815 on CI and
// 183.09405819371818 on a developer's machine, where the Java engine wrote the
// second. Compared exactly, which reference agreed depended on the machine.
func Same(a, b Answer) bool {
	return compare(a, b, sameValue)
}

// sameValue is Same for two answers the text compares: no structure and
// nothing untyped in either.
func sameValue(a, b Answer) bool {
	switch {
	case a.Kind != b.Kind:
		return false
	case a.Kind == "number":
		if isDecimal(a) != isDecimal(b) {
			return false
		}
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
		// Two whole numbers, one of more digits than a float64 writes, are no
		// float's noise but an exact result, `div` above all: 99999999999999999999
		// and 99999999999999999998 are a relative 1e-20 apart. Only two whole
		// numbers: a whole one and a fraction stay within the tolerance, and so
		// do whole numbers padded with zeros; SameExactly is for a corpus that
		// has no float in it.
		if a.Num.IsInteger() && b.Num.IsInteger() &&
			max(writtenDigits(a.Num), writtenDigits(b.Num)) > floatWrittenDigits {
			return false
		}
		scale := decimal.Max(a.Num.Abs(), b.Num.Abs())
		return a.Num.Sub(b.Num).Abs().LessThanOrEqual(scale.Mul(floatTolerance))
	case a.Kind == "error", a.Kind == "refused", a.Kind == "null":
		return true
	}
	return a.Text == b.Text
}

// SameExactly is Same without the tolerance for a float64's last digits: for a
// corpus that computes nothing in floating point, where a number a hair away
// is a wrong answer. The tolerance cannot tell an exact result from a float's
// by its digits alone — 10000000000000000000 and 10000000000000100000 are
// within it, a quotient off by a hundred thousand.
func SameExactly(a, b Answer) bool {
	return compare(a, b, sameExactValue)
}

func sameExactValue(a, b Answer) bool {
	if a.Kind == "number" && b.Kind == "number" {
		return isDecimal(a) == isDecimal(b) && a.Num.Equal(b.Num)
	}
	return sameValue(a, b)
}

// compare compares by the text, with leaf for numbers, unless either answer
// holds a structure or a value of no type: those it compares part by part.
// Only those, so that every answer the text compared before is compared as
// before.
func compare(a, b Answer, leaf func(a, b Answer) bool) bool {
	if !structured(a) && !structured(b) {
		return leaf(a, b)
	}
	if b.Untyped && !a.Untyped {
		a, b = b, a
	}
	if a.Untyped {
		return compareUntyped(a, b, leaf)
	}
	switch {
	case a.form != b.form:
		return false
	case a.form == "list":
		return sameParts(a, b, leaf)
	case a.form == "interval":
		return a.Text[len("Interval")] == b.Text[len("Interval")] && a.Text[len(a.Text)-1] == b.Text[len(b.Text)-1] &&
			sameParts(a, b, leaf)
	case a.Kind != b.Kind:
		return false
	case a.Kind == "structure":
		return structureType(a) == structureType(b) && sameParts(a, b, leaf)
	}
	return leaf(a, b)
}

// compareUntyped is compare for a, a value of no type.
func compareUntyped(a, b Answer, leaf func(a, b Answer) bool) bool {
	switch {
	case b.Untyped:
		return a.Kind == b.Kind && a.Text == b.Text
	case a.Kind == "untyped" && strings.HasPrefix(a.Text, "Number "):
		// A number of no type is the same as a number of either, by its value
		// as one of that type. Its type is confirmed elsewhere: see
		// TestAnUntypedMatchIsConfirmed.
		if b.Kind != "number" {
			return false
		}
		typ, _, _ := strings.Cut(b.Text, " ")
		return leaf(Answer{Kind: "number", Text: typ + " " + a.Num.String(), Num: a.Num}, b)
	case a.Kind == "untyped":
		// A text of no type: a String or a Long, and this engine has no Long
		// at run time (D4c).
		return b.Kind == "string" && b.Text == strings.TrimPrefix(a.Text, "Text ")
	}
	// An object of no type is the same as a structure of any type with the
	// same fields.
	return b.Kind == "structure" && sameParts(a, b, leaf)
}

// structured reports whether an answer holds a structure or a value of no
// type, at any depth.
func structured(a Answer) bool {
	if a.Untyped || a.Kind == "structure" {
		return true
	}
	return slices.ContainsFunc(a.Parts, structured)
}

func structureType(a Answer) string {
	typ, _, _ := strings.Cut(a.Text, "{")
	return typ
}

func sameParts(a, b Answer, leaf func(a, b Answer) bool) bool {
	if !slices.Equal(a.Names, b.Names) || len(a.Parts) != len(b.Parts) {
		return false
	}
	for i := range a.Parts {
		if !compare(a.Parts[i], b.Parts[i], leaf) {
			return false
		}
	}
	return true
}

// HoldsUntyped reports whether an answer holds a value of no type, at any
// depth: a match it takes part in needs its type confirmed by an oracle.
func HoldsUntyped(a Answer) bool {
	return a.Untyped || slices.ContainsFunc(a.Parts, HoldsUntyped)
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
		case "number", "null", "temporal", "quantity", "untyped":
		default:
			kind = "other"
		}
	}
	return Answer{Kind: kind, Text: "Interval" + open + partText(low) + ", " + partText(high) + closeBracket,
		Parts: []Answer{low, high}, form: "interval"}
}

// Failed is the answer of a definition that failed to evaluate, or panicked.
func Failed(err error, panicked bool) Answer {
	if panicked {
		return Answer{Kind: "error", Text: fmt.Sprintf("PANIC: %v", err), Panicked: true}
	}
	return Answer{Kind: "error", Text: "error: " + err.Error()}
}

// Refused is the answer of a definition whose library does not compile: a
// refusal before it runs, which is not the same answer as an error it meets
// while it runs.
func Refused(err error) Answer {
	return Answer{Kind: "refused", Text: "error: " + err.Error()}
}

// floatDigits is the significant digits a float64 holds for certain, and
// floatWrittenDigits the most it writes.
const (
	floatDigits        = 15
	floatWrittenDigits = 17
)

// floatTolerance is how far apart, relative to the larger, two numbers a
// float64 computed may be and be the same answer.
var floatTolerance = decimal.New(1, -13)

func significantDigits(d decimal.Decimal) int {
	return len(strings.TrimLeft(d.Abs().Coefficient().String(), "0"))
}

// writtenDigits is the significant digits of d without its trailing zeros:
// 1907346572495099500000 is the 17 a float64 wrote, padded to its magnitude.
func writtenDigits(d decimal.Decimal) int {
	return len(strings.Trim(d.Abs().Coefficient().String(), "0"))
}

func isDecimal(a Answer) bool { return strings.HasPrefix(a.Text, "Decimal ") }

// numberText is a number inside a list or an interval: its value, and its kind
// for a Decimal, so that an Integer element does not stand for a Decimal one.
func numberText(a Answer) string {
	if isDecimal(a) {
		return "Decimal " + a.Num.String()
	}
	return a.Num.String()
}
