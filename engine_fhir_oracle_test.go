package cql

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	cqltypes "github.com/gofhir/cql/types"
	fptypes "github.com/gofhir/fhirpath/types"
)

// The fhir area's oracle (docs/plans/003-clasificar-divergencias.md, PR 3):
// what the specification makes of each definition of testdata/snapshots/fhir.cql,
// computed from the FHIR ModelInfo, the data the definitions read and the
// rules of CQL 1.5.3, not from any engine's answer. Its corpus is a closed
// grammar (internal/referenceharness/differential/gen, fhirArea): is, as,
// `as … is null` and cast of a value against a type name, and nine operators
// over the same values.
//
// The rules, CQL 1.5.3 (cql.hl7.org/N1A):
//   - Appendix B, Type Operators: Is is true "if the run-time type of the
//     argument is the same as or derived from the type being tested"; As is
//     null "if the argument is not of the specified type at run-time"; cast
//     throws. The argument is of type Any, and nothing makes a cast between
//     types with no static relation an error: it is valid CQL, as the user
//     decided (PR 3, decision 1), and Appendix B gives `cast P as Observation`
//     as an error at run time.
//   - The type of a FHIR value is the one the ModelInfo declares, and a
//     FHIR primitive is a class of it, not the System type of its value (FHIR
//     CQL IG, using-modelinfo); nothing applies an implicit conversion to the
//     operand of a type operator.
//   - Developer's Guide, Multiple Data Models: an unqualified type name is
//     searched for, exactly, in the models the library uses; System, declared
//     implicitly, is not considered.
//   - Developer's Guide, Types: "for any given operator invocation, the type of
//     the arguments must match the types of the operands", through the
//     conversions of Conversion Precedence; an invocation nothing matches is a
//     translation error. Choice Types: a choice is used where one of its
//     types is expected, through an implicit cast.
//   - Function Resolution: the least converting signature is chosen, by
//     Conversion Precedence (a conversion to a simple type before one to a
//     class type); two that tie are "an ambiguous resolution error", for a
//     choice too.
//   - Undecided: null's run-time type; a call that needs two conversions in a
//     row; a list converted element by element.
const fhirOracleRule = "fhir oracle (engine_fhir_oracle_test.go), CQL 1.5.3 (cql.hl7.org/N1A): " +
	"Appendix B, Type Operators; Developer's Guide, Types and Multiple Data Models; FHIR ModelInfo 4.0.1"

// fhirDecisionPlan is where the user's decisions about the area are recorded,
// with the definitions they cover.
const fhirDecisionPlan = "docs/plans/003-clasificar-divergencias.md"

var classifyFHIR = flag.Bool("classify-fhir", false,
	"give the divergences of the fhir area the verdict of its oracle")

// fType is a type of the fhir oracle: a qualified named type, or List,
// Interval or Choice of others.
type fType struct {
	name string // "FHIR.date", "System.Integer", or "List", "Interval", "Choice"
	elem []fType
}

func fNamed(name string) fType { return fType{name: name} }

func (t fType) String() string {
	if len(t.elem) == 0 {
		return t.name
	}
	parts := make([]string, len(t.elem))
	for i, e := range t.elem {
		parts[i] = e.String()
	}
	return t.name + "<" + strings.Join(parts, ", ") + ">"
}

// systemTypes is the System model's named types (Developer's Guide, Types),
// each derived directly from System.Any.
var systemTypes = []string{"Any", "Boolean", "Integer", "Long", "Decimal", "String", "Date", "DateTime", "Time",
	"Quantity", "Ratio", "Code", "Concept", "Vocabulary", "ValueSet", "CodeSystem"}

// fhirModel is what the oracle reads of the FHIR ModelInfo: each type's base,
// its elements' types, the System type a primitive's value has, and the
// implicit conversions FHIRHelpers implements.
type fhirModel struct {
	base        map[string]string
	elements    map[string]map[string]fType
	conversions map[string]fType
}

type modelInfoXML struct {
	TypeInfos   []typeInfoXML `xml:"typeInfo"`
	Conversions []struct {
		From string `xml:"fromType,attr"`
		To   string `xml:"toType,attr"`
	} `xml:"conversionInfo"`
}

type typeInfoXML struct {
	Namespace string `xml:"namespace,attr"`
	Name      string `xml:"name,attr"`
	BaseType  string `xml:"baseType,attr"`
	Elements  []struct {
		Name        string            `xml:"name,attr"`
		ElementType string            `xml:"elementType,attr"`
		Specifier   *typeSpecifierXML `xml:"elementTypeSpecifier"`
	} `xml:"element"`
}

type typeSpecifierXML struct {
	Kind        string             `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
	ElementType string             `xml:"elementType,attr"`
	Namespace   string             `xml:"namespace,attr"`
	Name        string             `xml:"name,attr"`
	Choices     []typeSpecifierXML `xml:"choice"`
}

var (
	fhirModelOnce sync.Once
	fhirModelRead *fhirModel
	fhirModelErr  error
)

// loadFHIRModel reads model/fhir-modelinfo-4.0.1.xml.gz, the document
// cqframework publishes, by itself: the oracle does not go through the
// model package, whose reading is part of what it checks.
func loadFHIRModel(t *testing.T) *fhirModel {
	t.Helper()
	fhirModelOnce.Do(func() {
		fhirModelRead, fhirModelErr = readFHIRModel(filepath.Join("model", "fhir-modelinfo-4.0.1.xml.gz"))
	})
	if fhirModelErr != nil {
		t.Fatal(fhirModelErr)
	}
	return fhirModelRead
}

func readFHIRModel(path string) (*fhirModel, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	var doc modelInfoXML
	if err := xml.NewDecoder(z).Decode(&doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	m := &fhirModel{base: map[string]string{}, elements: map[string]map[string]fType{}, conversions: map[string]fType{}}
	for _, ti := range doc.TypeInfos {
		name := ti.Namespace + "." + ti.Name
		m.base[name] = ti.BaseType
		els := map[string]fType{}
		for _, e := range ti.Elements {
			switch {
			case e.ElementType != "":
				els[e.Name] = parseQualified(e.ElementType)
			case e.Specifier != nil:
				els[e.Name] = specifierType(*e.Specifier)
			}
		}
		m.elements[name] = els
	}
	for _, c := range doc.Conversions {
		m.conversions[c.From] = parseQualified(c.To)
	}
	if len(m.base) == 0 {
		return nil, fmt.Errorf("%s holds no type", path)
	}
	return m, nil
}

func specifierType(s typeSpecifierXML) fType {
	switch {
	case strings.HasSuffix(s.Kind, "ChoiceTypeSpecifier"):
		choice := fType{name: "Choice"}
		for _, c := range s.Choices {
			choice.elem = append(choice.elem, specifierType(c))
		}
		return choice
	case strings.HasSuffix(s.Kind, "ListTypeSpecifier"):
		return fType{"List", []fType{parseQualified(s.ElementType)}}
	}
	return fNamed(s.Namespace + "." + s.Name)
}

// parseQualified reads a type name the ModelInfo writes, qualified:
// "FHIR.date", "Interval<System.DateTime>".
func parseQualified(s string) fType {
	if open := strings.IndexByte(s, '<'); open > 0 && strings.HasSuffix(s, ">") {
		return fType{s[:open], []fType{parseQualified(s[open+1 : len(s)-1])}}
	}
	return fNamed(s)
}

// resolveTypeName reads a type name of the corpus as a library using FHIR
// does (Developer's Guide, Multiple Data Models): a qualified name as it is,
// an unqualified one in FHIR, exactly, and in System only when FHIR has none.
func (m *fhirModel) resolveTypeName(s string) (fType, error) {
	if open := strings.IndexByte(s, '<'); open > 0 && strings.HasSuffix(s, ">") {
		generic := s[:open]
		if generic != "List" && generic != "Interval" {
			return fType{}, fmt.Errorf("a generic type the oracle does not know: %s", s)
		}
		e, err := m.resolveTypeName(s[open+1 : len(s)-1])
		return fType{generic, []fType{e}}, err
	}
	if model, name, ok := strings.Cut(s, "."); ok {
		switch {
		case model == "System" && slices.Contains(systemTypes, name), model == "FHIR" && m.base[s] != "":
			return fNamed(s), nil
		}
		return fType{}, fmt.Errorf("a type neither model has: %s", s)
	}
	if m.base["FHIR."+s] != "" {
		return fNamed("FHIR." + s), nil
	}
	if slices.Contains(systemTypes, s) {
		return fNamed("System." + s), nil
	}
	return fType{}, fmt.Errorf("a type neither model has: %s", s)
}

// derives reports whether a value of type a is of type b: the same type, or
// one derived from it by its bases; a list or an interval by its element. A
// choice is never one: its types are taken one by one.
func (m *fhirModel) derives(a, b fType) bool {
	switch {
	case b.name == "System.Any" && len(b.elem) == 0:
		return true
	case len(a.elem) > 0 || len(b.elem) > 0:
		return a.name == b.name && len(a.elem) == 1 && len(b.elem) == 1 && m.derives(a.elem[0], b.elem[0])
	}
	for name := a.name; name != ""; name = m.base[name] {
		if name == b.name {
			return true
		}
		if strings.HasPrefix(name, "System.") {
			return false // a System type derives directly from Any
		}
	}
	return false
}

// castable reports whether the Java translator accepts an as or a cast of a
// value of a static type to a target type: one derives from the other, and
// for a choice, the target is Any or one of its types exactly. The
// specification makes no such requirement (PR 3, decision 1); this is what
// the translator does, measured on every as and cast of the corpus, and the
// set of casts the decision covers.
func (m *fhirModel) castable(static, target fType) bool {
	if static.name == "Choice" {
		return target.name == "System.Any" || slices.ContainsFunc(static.elem, func(c fType) bool { return c.String() == target.String() })
	}
	return m.derives(static, target) || m.derives(target, static)
}

// fhirOperand is a value of the corpus: its static type, its run-time type
// (none for null), and its value, written as the oracle compares answers.
type fhirOperand struct {
	static  fType
	runtime *fType
	value   string // the typed text of the value, see typedText
	node    any    // the value as JSON, for a FHIR value the data holds or CQL builds
}

var (
	fhirDataPath  = regexp.MustCompile(`^(Patient|First\(\[(\w+)\]\)|\(First\(\[(\w+)\] O where O\.id = '(\w+)'\)\))((?:\.\w+)*)$`)
	fhirInstance  = regexp.MustCompile(`^(FHIR\.\w+) \{ (.*) \}$`)
	fhirQuantity  = regexp.MustCompile(`^(-?\d+(?:\.\d+)?) '(\w+)'$`)
	fhirIntegers  = regexp.MustCompile(`^\{ (\d+(?:, \d+)*) \}$`)
	fhirValueSet  = regexp.MustCompile(`^System\.ValueSet \{ id: '(\w+)' \}$`)
	fhirNullTyped = regexp.MustCompile(`^\(null as (\w+)\)$`)
)

// fhirCorpus is what the oracle reads the corpus with: the model and the
// data the test serves.
type fhirCorpus struct {
	model *fhirModel
	data  map[string]any
}

func readFHIRCorpus(t *testing.T) *fhirCorpus {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", "fhir-data.json"))
	if err != nil {
		t.Fatal(err)
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var data map[string]any
	if err := d.Decode(&data); err != nil {
		t.Fatal(err)
	}
	return &fhirCorpus{model: loadFHIRModel(t), data: data}
}

// operand reads a value of the corpus.
func (c *fhirCorpus) operand(s string) (fhirOperand, error) {
	m := c.model
	switch {
	case fhirNullTyped.MatchString(s):
		t, err := m.resolveTypeName(fhirNullTyped.FindStringSubmatch(s)[1])
		return fhirOperand{static: t, value: "null"}, err
	case fhirQuantity.MatchString(s):
		q := fhirQuantity.FindStringSubmatch(s)
		t := fNamed("System.Quantity")
		return fhirOperand{static: t, runtime: &t, value: quantityText(q[1], q[2])}, nil
	case fhirIntegers.MatchString(s):
		var items []string
		for _, n := range strings.Split(fhirIntegers.FindStringSubmatch(s)[1], ", ") {
			items = append(items, "System.Integer "+n)
		}
		t := fType{"List", []fType{fNamed("System.Integer")}}
		return fhirOperand{static: t, runtime: &t, value: "List{" + strings.Join(items, ", ") + "}"}, nil
	case fhirValueSet.MatchString(s):
		t := fNamed("System.ValueSet")
		return fhirOperand{static: t, runtime: &t, value: "System.ValueSet {id: System.String " + fhirValueSet.FindStringSubmatch(s)[1] + "}"}, nil
	case fhirInstance.MatchString(s):
		node, typ, err := c.instance(s)
		if err != nil {
			return fhirOperand{}, err
		}
		t := fNamed(typ)
		return fhirOperand{static: t, runtime: &t, value: fhirText(typ, node), node: node}, nil
	case fhirDataPath.MatchString(s):
		return c.dataPath(fhirDataPath.FindStringSubmatch(s))
	}
	return fhirOperand{}, fmt.Errorf("a value the oracle does not know: %s", s)
}

// instance builds the JSON of a FHIR instance selector, `FHIR.code { value: 'x' }`:
// a primitive is its value, a complex type an object of its elements.
func (c *fhirCorpus) instance(s string) (node any, typ string, err error) {
	m := fhirInstance.FindStringSubmatch(s)
	typ, body := m[1], m[2]
	field, value, ok := strings.Cut(body, ": ")
	if !ok {
		return nil, "", fmt.Errorf("an instance the oracle does not know: %s", s)
	}
	var v any
	switch {
	case strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'"):
		v = strings.Trim(value, "'")
	case fhirInstance.MatchString(value):
		inner, _, err := c.instance(value)
		if err != nil {
			return nil, "", err
		}
		v = inner
	default:
		return nil, "", fmt.Errorf("an instance element the oracle does not know: %s", value)
	}
	if _, declared := c.model.elementOf(typ, field); !declared {
		return nil, "", fmt.Errorf("%s has no element %s", typ, field)
	}
	if c.model.primitive(typ) != "" && field == "value" {
		return v, typ, nil
	}
	return map[string]any{field: v}, typ, nil
}

// elementOf is the type an element of a type has, its own or one of its
// bases'.
func (m *fhirModel) elementOf(typ, name string) (fType, bool) {
	for t := typ; t != ""; t = m.base[t] {
		if e, ok := m.elements[t][name]; ok {
			return e, true
		}
	}
	return fType{}, false
}

// primitive is the System type a FHIR primitive's value has, or "" for a type
// that is no primitive: a primitive is a class whose value element is of a
// System type (FHIR CQL IG, using-modelinfo).
func (m *fhirModel) primitive(typ string) string {
	if v, ok := m.elements[typ]["value"]; ok && strings.HasPrefix(v.name, "System.") {
		return v.name
	}
	return ""
}

// dataPath reads a path into the data: a resource of the context or the
// first of a retrieve, then its elements, a choice by the branch the data
// holds.
func (c *fhirCorpus) dataPath(m []string) (fhirOperand, error) {
	var node any
	var typ string
	switch {
	case m[1] == "Patient":
		node, typ = c.data["Patient"], "FHIR.Patient"
	case m[2] != "":
		list, _ := c.data[m[2]].([]any)
		if len(list) == 0 {
			return fhirOperand{}, fmt.Errorf("no %s in the data", m[2])
		}
		node, typ = list[0], "FHIR."+m[2]
	default:
		list, _ := c.data[m[3]].([]any)
		for _, r := range list {
			if r.(map[string]any)["id"] == m[4] {
				node = r
			}
		}
		if node == nil {
			return fhirOperand{}, fmt.Errorf("no %s %s in the data", m[3], m[4])
		}
		typ = "FHIR." + m[3]
	}
	static := fNamed(typ)
	for _, name := range strings.Split(strings.TrimPrefix(m[5], "."), ".") {
		if name == "" {
			continue
		}
		spec, ok := c.model.elementOf(static.name, name)
		if !ok {
			return fhirOperand{}, fmt.Errorf("%s has no element %s", static.name, name)
		}
		obj, _ := node.(map[string]any)
		static = spec
		if spec.name == "Choice" {
			// The branch the data holds names its type: valueQuantity is a
			// FHIR.Quantity, deceasedDateTime a FHIR.dateTime.
			node = nil
			for _, choice := range spec.elem {
				local := strings.TrimPrefix(choice.name, "FHIR.")
				if v, ok := obj[name+strings.ToUpper(local[:1])+local[1:]]; ok {
					node, typ = v, choice.name
				}
			}
		} else {
			node, typ = obj[name], spec.name
		}
	}
	if node == nil {
		return fhirOperand{static: static, value: "null"}, nil
	}
	rt := fNamed(typ)
	return fhirOperand{static: static, runtime: &rt, value: fhirText(typ, node), node: node}, nil
}

// fhirText is a FHIR value as the oracle compares answers: its type and, for
// a primitive, its value, for a complex type its JSON with sorted keys.
func fhirText(typ string, node any) string {
	if obj, ok := node.(map[string]any); ok {
		out, _ := json.Marshal(obj)
		return typ + " " + string(out)
	}
	return typ + " " + fmt.Sprint(node)
}

func quantityText(value, unit string) string {
	return "System.Quantity " + decimalText(value) + " '" + unit + "'"
}

func decimalText(s string) string {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return s
	}
	return d.String()
}

// fhirVerdict is the oracle's answer to a definition: the answer it expects,
// as typed text, and why. It decides every definition of the corpus, some
// by ADR 003 (decided). A value is "Type value",
// "null", "translation error" or "run-time error".
type fhirVerdict struct {
	expected, reason string
	// unrelated marks an as or a cast the Java translator refuses for its
	// static types: the casts the user decided are valid CQL (PR 3,
	// decision 1).
	unrelated bool
	// decided marks a verdict the specification leaves open and ADR 003
	// decides: a reference that gives another answer reads it otherwise.
	decided bool
}

const (
	translationError = "translation error"
	runTimeError     = "run-time error"
)

var (
	fhirIsAs   = regexp.MustCompile(`^\(\((.*)\) (is|as) ([\w.<>]+)\)$`)
	fhirAsNull = regexp.MustCompile(`^\({3}(.*)\) as ([\w.<>]+)\) is null\)$`)
	fhirCast   = regexp.MustCompile(`^\(cast \((.*)\) as ([\w.<>]+)\)$`)
	fhirUnary  = regexp.MustCompile(`^(-|Abs|ToString)\(\((.*)\)\)$`)
	fhirAggr   = regexp.MustCompile(`^(Sum|Avg)\(\{ \((.*)\) \}\)$`)
	fhirBinary = regexp.MustCompile(`^\(\((.*)\)\) (\+ 1 day|\+ 1 'mg'|same day as @2020-01-01T|is null)$`)
)

// fhirExpected is the oracle's verdict on a definition.
func (c *fhirCorpus) fhirExpected(expr string) (fhirVerdict, error) {
	var op, value, target string
	switch {
	case fhirAsNull.MatchString(expr):
		m := fhirAsNull.FindStringSubmatch(expr)
		op, value, target = "as-null", m[1], m[2]
	case fhirCast.MatchString(expr):
		m := fhirCast.FindStringSubmatch(expr)
		op, value, target = "cast", m[1], m[2]
	case fhirIsAs.MatchString(expr):
		m := fhirIsAs.FindStringSubmatch(expr)
		op, value, target = m[2], m[1], m[3]
	default:
		return c.operatorExpected(expr)
	}
	v, err := c.operand(value)
	if err != nil {
		return fhirVerdict{}, err
	}
	t, err := c.model.resolveTypeName(target)
	if err != nil {
		return fhirVerdict{}, err
	}
	unrelated := op != "is" && !c.model.castable(v.static, t)
	if v.runtime == nil {
		switch op {
		case "as":
			return fhirVerdict{expected: "null", reason: "null as " + t.String() + " is null", unrelated: unrelated}, nil
		case "as-null":
			return fhirVerdict{expected: "System.Boolean true", reason: "null as " + t.String() + " is null", unrelated: unrelated}, nil
		}
		if op == "is" {
			return fhirVerdict{expected: "System.Boolean false", decided: true, unrelated: unrelated,
				reason: "null is of no type, so null is " + t.String() + " is false (" + undecidedADR + ", decision 1)"}, nil
		}
		return fhirVerdict{expected: "null", decided: true, unrelated: unrelated,
			reason: "null is of no type, so cast null as " + t.String() + " is null (" + undecidedADR + ", decision 2)"}, nil
	}
	of := c.model.derives(*v.runtime, t)
	relation := " is not of "
	if of {
		relation = " is of "
	}
	verdict := fhirVerdict{reason: "a " + v.runtime.String() + relation + t.String(), unrelated: unrelated}
	switch {
	case op == "is":
		verdict.expected = fmt.Sprintf("System.Boolean %v", of)
	case op == "as-null":
		verdict.expected = fmt.Sprintf("System.Boolean %v", !of)
	case of:
		verdict.expected = v.value
	case op == "as":
		verdict.expected = "null"
	default:
		verdict.expected = runTimeError
	}
	return verdict, nil
}

// fhirOperators is the operators of the corpus with their signatures in
// Appendix B, by the first operand: the second is 1 day or 1 'mg', a
// System.Quantity, for Add, and @2020-01-01T, a System.DateTime, for SameAs.
var fhirOperators = map[string][]string{
	"-":                        {"System.Integer", "System.Long", "System.Decimal", "System.Quantity"},
	"Abs":                      {"System.Integer", "System.Long", "System.Decimal", "System.Quantity"},
	"Sum":                      {"List<System.Integer>", "List<System.Long>", "List<System.Decimal>", "List<System.Quantity>"},
	"Avg":                      {"List<System.Decimal>", "List<System.Quantity>"},
	"+ 1 day":                  {"System.Quantity", "System.Date", "System.DateTime", "System.Time"},
	"+ 1 'mg'":                 {"System.Quantity", "System.Date", "System.DateTime", "System.Time"},
	"same day as @2020-01-01T": {"System.DateTime"},
	"ToString": {"System.Boolean", "System.Integer", "System.Long", "System.Decimal", "System.Quantity", "System.Ratio",
		"System.Date", "System.DateTime", "System.Time"},
	"is null": {"System.Any"},
}

// systemPromotions is the implicit conversions between System types
// (Developer's Guide, Implicit Conversions).
var systemPromotions = map[string][]string{
	"System.Integer": {"System.Long", "System.Decimal", "System.Quantity"},
	"System.Long":    {"System.Decimal"},
	"System.Decimal": {"System.Quantity"},
	"System.Date":    {"System.DateTime"},
}

// The cost of reaching a parameter, by Conversion Precedence (Developer's
// Guide): a value of the type, then an implicit conversion to a simple type,
// then one to a class type. Two conversions in a row, and a list converted
// element by element, are ranked after every single conversion, and the
// oracle does not decide them.
const (
	toSimpleType  = 1
	toClassType   = 2
	chainedCost   = 50
	elementwise   = 100
	unreachable   = -1
	notConversion = 0
)

// classTypes is the System types that are classes, not simple types
// (Developer's Guide, Types): a conversion to one ranks after a conversion to
// a simple type.
var classTypes = []string{"System.Quantity", "System.Ratio", "System.Code", "System.Concept", "Interval"}

func conversionCost(to fType) int {
	if slices.Contains(classTypes, to.name) {
		return toClassType
	}
	return toSimpleType
}

// conversionsTo is the cost of taking a value of type a to a parameter of
// type p, unreachable when no conversion does.
func (m *fhirModel) conversionsTo(a, p fType) int {
	if m.derives(a, p) {
		return notConversion
	}
	if a.name == "List" && p.name == "List" {
		if n := m.conversionsTo(a.elem[0], p.elem[0]); n > 0 {
			return elementwise + n
		}
		return unreachable
	}
	if to, ok := m.conversions[a.name]; ok && len(a.elem) == 0 {
		if m.derives(to, p) {
			return conversionCost(to)
		}
		for _, q := range systemPromotions[to.name] {
			if m.derives(fNamed(q), p) {
				return chainedCost + conversionCost(fNamed(q))
			}
		}
	}
	for _, q := range systemPromotions[a.name] {
		if m.derives(fNamed(q), p) {
			return conversionCost(fNamed(q))
		}
	}
	return unreachable
}

// operatorExpected is the oracle's verdict on an operator of the corpus:
// which signature its operand's static type reaches, and the result at run
// time.
func (c *fhirCorpus) operatorExpected(expr string) (fhirVerdict, error) {
	var op, value string
	switch {
	case fhirUnary.MatchString(expr):
		m := fhirUnary.FindStringSubmatch(expr)
		op, value = m[1], m[2]
	case fhirAggr.MatchString(expr):
		m := fhirAggr.FindStringSubmatch(expr)
		op, value = m[1], m[2]
	case fhirBinary.MatchString(expr):
		m := fhirBinary.FindStringSubmatch(expr)
		op, value = m[2], m[1]
	default:
		return fhirVerdict{}, fmt.Errorf("not of the corpus's grammar: %s", expr)
	}
	v, err := c.operand(value)
	if err != nil {
		return fhirVerdict{}, err
	}
	arg := v.static
	if op == "Sum" || op == "Avg" {
		arg = fType{"List", []fType{v.static}}
	}
	// Each type the operand may have statically: a choice's, each of its own.
	statics := []fType{arg}
	if v.static.name == "Choice" {
		statics = nil
		for _, e := range v.static.elem {
			if op == "Sum" || op == "Avg" {
				e = fType{"List", []fType{e}}
			}
			statics = append(statics, e)
		}
	}
	type reach struct {
		param fType
		cost  int
		from  []fType
	}
	var reached []reach
	for _, p := range fhirOperators[op] {
		param := parseQualified(p)
		best, from := -1, make([]fType, 0, len(statics))
		for _, s := range statics {
			n := c.model.conversionsTo(s, param)
			switch {
			case n < 0:
			case best < 0 || n < best:
				best, from = n, append(from[:0], s)
			case n == best:
				from = append(from, s)
			}
		}
		if best >= 0 {
			reached = append(reached, reach{param, best, from})
		}
	}
	name := op + " of " + arg.String()
	if len(reached) == 0 {
		return fhirVerdict{expected: translationError, reason: "no signature of " + op + " takes a " + arg.String()}, nil
	}
	slices.SortFunc(reached, func(a, b reach) int { return a.cost - b.cost })
	if len(reached) > 1 && reached[0].cost == reached[1].cost {
		return fhirVerdict{expected: translationError, reason: name + " is ambiguous between " +
			reached[0].param.String() + " and " + reached[1].param.String() + " (Function Resolution)"}, nil
	}
	sig := reached[0]
	if sig.cost >= elementwise {
		return c.operateByElement(op, v, name)
	}
	reason := name + " is " + op + "(" + sig.param.String() + ")"
	decided := sig.cost >= chainedCost
	if decided {
		// The specification does not say a call takes two conversions in a
		// row; ADR 003 (decision 4) takes them.
		reason += " through two conversions in a row (" + undecidedADR + ", decision 4)"
	}
	// A choice takes the signature through an implicit cast: a value of
	// another of its types is null by then (Choice Types).
	if v.runtime == nil || v.static.name == "Choice" && !slices.ContainsFunc(sig.from, func(s fType) bool {
		e := s
		if s.name == "List" {
			e = s.elem[0]
		}
		return c.model.derives(*v.runtime, e)
	}) {
		r := c.operateOnNull(op, reason)
		r.decided = decided
		return r, nil
	}
	r, err := c.operate(op, v, reason)
	r.decided = decided
	return r, err
}

// operateByElement is an aggregate whose list reaches a signature only by
// converting each element, which the specification does not say a list does.
// ADR 003 (decision 4) takes the signature each element's run-time type
// reaches: a Quantity is summed as one (D194), and an element that reaches
// none is null (D374, D792).
func (c *fhirCorpus) operateByElement(op string, v fhirOperand, name string) (fhirVerdict, error) {
	reason := name + " takes the signature of its element's run-time type (" + undecidedADR + ", decision 4)"
	if v.runtime == nil {
		r := c.operateOnNull(op, reason)
		r.decided = true
		return r, nil
	}
	best, ties := unreachable, 0
	var param fType
	for _, p := range fhirOperators[op] {
		q := parseQualified(p)
		switch n := c.model.conversionsTo(*v.runtime, q.elem[0]); {
		case n < 0:
		case best < 0 || n < best:
			best, ties, param = n, 1, q
		case n == best:
			ties++
		}
	}
	switch {
	case best < 0:
		r := c.operateOnNull(op, reason+": no signature takes a "+v.runtime.String())
		r.decided = true
		return r, nil
	case ties > 1:
		return fhirVerdict{}, fmt.Errorf("%s: a %s reaches more than one signature at one cost", name, v.runtime)
	}
	r, err := c.operate(op, v, reason+": "+op+"("+param.String()+")")
	r.decided = true
	return r, err
}

// operateOnNull is an operator's result on null.
func (c *fhirCorpus) operateOnNull(op, reason string) fhirVerdict {
	if op == "is null" {
		return fhirVerdict{expected: "System.Boolean true", reason: reason}
	}
	return fhirVerdict{expected: "null", reason: reason + ", on null"}
}

// systemValue is a value converted to its System type, as FHIRHelpers does
// it: a primitive to its value, a Quantity to a System Quantity.
type systemValue struct {
	typ        string
	text, unit string
}

func (c *fhirCorpus) systemValue(v fhirOperand) (systemValue, bool) {
	rt := v.runtime.name
	if p := c.model.primitive(rt); p != "" {
		return systemValue{typ: p, text: fmt.Sprint(v.node)}, true
	}
	switch rt {
	case "FHIR.Quantity":
		q := v.node.(map[string]any)
		return systemValue{typ: "System.Quantity", text: fmt.Sprint(q["value"]), unit: fmt.Sprint(q["unit"])}, true
	case "System.Quantity":
		if q := fhirQuantity.FindStringSubmatch(strings.TrimPrefix(v.value, "System.Quantity ")); q != nil {
			return systemValue{typ: rt, text: q[1], unit: q[2]}, true
		}
	}
	return systemValue{}, false
}

// operate is an operator's result on a value that reached its signature.
func (c *fhirCorpus) operate(op string, v fhirOperand, reason string) (fhirVerdict, error) {
	if op == "is null" {
		return fhirVerdict{expected: "System.Boolean false", reason: reason}, nil
	}
	s, ok := c.systemValue(v)
	if !ok {
		return fhirVerdict{}, fmt.Errorf("%s: no System value for a %s", reason, v.runtime)
	}
	num := func() decimal.Decimal { d, _ := decimal.NewFromString(s.text); return d }
	out := func(typ string, d decimal.Decimal) string {
		if typ == "System.Quantity" {
			return quantityText(d.String(), s.unit)
		}
		return typ + " " + d.String()
	}
	switch op {
	case "-":
		return fhirVerdict{expected: out(s.typ, num().Neg()), reason: reason}, nil
	case "Abs":
		return fhirVerdict{expected: out(s.typ, num().Abs()), reason: reason}, nil
	case "Sum":
		return fhirVerdict{expected: out(s.typ, num()), reason: reason + ", of one element"}, nil
	case "Avg":
		typ := s.typ
		if typ != "System.Quantity" {
			typ = "System.Decimal"
		}
		return fhirVerdict{expected: out(typ, num()), reason: reason + ", of one element"}, nil
	case "ToString":
		text := s.text
		if s.typ == "System.Quantity" {
			text = decimalText(s.text) + " '" + s.unit + "'"
		}
		return fhirVerdict{expected: "System.String " + text, reason: reason}, nil
	case "+ 1 day":
		if s.typ == "System.Quantity" {
			return fhirVerdict{expected: "null", reason: reason + ": a mass and a time are no one dimension, and Appendix B, Add, makes that null (QuantityAddIsNull)"}, nil
		}
		next, ok := nextDay(s.text)
		if !ok {
			return fhirVerdict{}, fmt.Errorf("%s: a date the oracle cannot shift: %s", reason, s.text)
		}
		return fhirVerdict{expected: s.typ + " " + next, reason: reason}, nil
	case "+ 1 'mg'":
		if s.typ == "System.Quantity" {
			return fhirVerdict{expected: quantityText(num().Add(decimal.NewFromInt(1)).String(), s.unit), reason: reason}, nil
		}
		// Appendix B requires the unit and says neither null nor error for one
		// that breaks it; the unit is a value, so only the addition can find
		// it, and both references fail as it runs.
		return fhirVerdict{expected: runTimeError, reason: reason + ": the unit of a quantity added to a date \"must be one of\" the time units " +
			"(Appendix B, Date and Time Operators, Add), which a unit, a value, breaks only as the addition runs; both references fail there"}, nil
	case "same day as @2020-01-01T":
		return fhirVerdict{expected: fmt.Sprintf("System.Boolean %v", strings.HasPrefix(s.text, "2020-01-01")), reason: reason}, nil
	}
	return fhirVerdict{}, fmt.Errorf("an operator the oracle does not know: %s", op)
}

// nextDay is a date, or a date and time, a day later, at the precision it is
// written to.
func nextDay(s string) (string, bool) {
	date, rest := s, ""
	if len(s) > 10 {
		date, rest = s[:10], s[10:]
	}
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", false
	}
	return d.AddDate(0, 0, 1).Format("2006-01-02") + rest, true
}

// fhirDefinitions is the definitions of testdata/snapshots/fhir.cql.
func fhirDefinitions(t *testing.T) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", "fhir.cql"))
	if err != nil {
		t.Fatal(err)
	}
	defs := map[string]string{}
	for _, m := range regexp.MustCompile(`(?m)^define (D\d+): (.*)$`).FindAllStringSubmatch(string(raw), -1) {
		defs[m[1]] = m[2]
	}
	return defs
}

// fhirCorpusSize is the number of definitions of testdata/snapshots/fhir.cql.
const fhirCorpusSize = 900

// TestTheFHIROracleReadsTheWholeCorpus holds the oracle to every definition of
// the corpus, divergent or not: one it cannot read is a gap in it.
func TestTheFHIROracleReadsTheWholeCorpus(t *testing.T) {
	defs := fhirDefinitions(t)
	if len(defs) != fhirCorpusSize {
		t.Fatalf("the oracle reads %d definitions of fhir.cql; the corpus has %d", len(defs), fhirCorpusSize)
	}
	c := readFHIRCorpus(t)
	for _, name := range slices.Sorted(maps.Keys(defs)) {
		if _, err := c.fhirExpected(defs[name]); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestTheFHIROracleTypesStaticallyAsJava holds the oracle's static types to the
// Java translator's: it refuses an as or a cast exactly where castable says
// it does, which is the set PR 3's decision 1 covers.
func TestTheFHIROracleTypesStaticallyAsJava(t *testing.T) {
	defs := fhirDefinitions(t)
	c := readFHIRCorpus(t)
	java := readReferenceResults(t, "fhir", "java")
	for _, name := range slices.Sorted(maps.Keys(defs)) {
		if !fhirAsNull.MatchString(defs[name]) && !fhirCast.MatchString(defs[name]) && !fhirIsAs.MatchString(defs[name]) {
			continue
		}
		v, err := c.fhirExpected(defs[name])
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		answer, answered := java[name]
		if !answered {
			t.Errorf("%s: the Java engine gives no answer: the translator's refusals are measured on every as and cast", name)
			continue
		}
		refused := fhirReferenceText(answer) == translationError
		if refused != v.unrelated {
			t.Errorf("%s: the Java translator refuses it: %v; the oracle says it does: %v (%s)",
				name, refused, v.unrelated, defs[name])
		}
	}
}

// fhirReferenceText is a reference's answer as typed text, the oracle's
// form; "unreadable" for one the harness could not encode.
func fhirReferenceText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return "null"
	}
	var v referenceTyped
	if err := json.Unmarshal(raw, &v); err != nil {
		return "unreadable"
	}
	switch {
	case v.Error == "Translation":
		return translationError
	case v.Error != "":
		return runTimeError
	case strings.HasPrefix(v.Type, "Unencoded:"):
		return "unreadable"
	case strings.HasPrefix(v.Type, "FHIR."):
		var s string
		if json.Unmarshal(v.Value, &s) != nil {
			return v.Type + " " + string(v.Value)
		}
		if strings.HasPrefix(strings.TrimSpace(s), "{") {
			var obj map[string]any
			d := json.NewDecoder(strings.NewReader(s))
			d.UseNumber()
			if d.Decode(&obj) == nil {
				return fhirText(v.Type, obj)
			}
		}
		return v.Type + " " + s
	case v.Type == "List":
		items := make([]string, len(v.Elements))
		for i, e := range v.Elements {
			items[i] = fhirReferenceText(e)
		}
		return "List{" + strings.Join(items, ", ") + "}"
	case v.Type == "ValueSet" || v.Type == "CodeSystem":
		// Java writes every element, null the ones a selector did not give;
		// this engine keeps the given ones, as answers.Reference reads them.
		// Its texts are written plain, as a Code's; its code systems as a list.
		fields := make([]string, 0, len(v.Fields))
		for _, k := range slices.Sorted(maps.Keys(v.Fields)) {
			f := fhirReferenceText(v.Fields[k])
			var text *string
			if k != "codesystems" && json.Unmarshal(v.Fields[k], &text) == nil {
				f = "null"
				if text != nil {
					f = "System.String " + *text
				}
			}
			if f != "null" {
				fields = append(fields, k+": "+f)
			}
		}
		return "System." + v.Type + " {" + strings.Join(fields, ", ") + "}"
	}
	var s string
	switch v.Type {
	case "Boolean":
		var b bool
		if json.Unmarshal(v.Value, &b) == nil {
			return fmt.Sprintf("System.Boolean %v", b)
		}
	case "Integer", "Long", "Decimal":
		if json.Unmarshal(v.Value, &s) == nil {
			return "System." + v.Type + " " + decimalText(s)
		}
	case "String", "Date", "DateTime", "Time":
		if json.Unmarshal(v.Value, &s) == nil {
			return "System." + v.Type + " " + s
		}
	case "Quantity":
		if json.Unmarshal(v.Value, &s) == nil {
			return quantityText(s, v.Unit)
		}
	}
	return "unreadable"
}

type referenceTyped struct {
	Type     string            `json:"type"`
	Value    json.RawMessage   `json:"value"`
	Unit     string            `json:"unit"`
	Elements []json.RawMessage `json:"elements"`
	Error    string            `json:"error"`
	// A ValueSet's and a CodeSystem's elements, as the harness writes them.
	Fields map[string]json.RawMessage `json:"fields"`
}

// fhirEngineText is this engine's answer to a definition as typed text, the
// oracle's form: a FHIR value it holds as an element by the element's type.
// A FHIR primitive it holds has no FHIR type, only its value's System type
// (Option A), and is written apart: it is neither the FHIR element nor the
// System value.
func fhirEngineText(v fptypes.Value, err error) string {
	var semantic *ErrSemantic
	var syntax *ErrSyntaxError
	switch {
	case errors.As(err, &semantic), errors.As(err, &syntax):
		return translationError
	case err != nil:
		return runTimeError
	}
	return engineValueText(v)
}

func engineValueText(v fptypes.Value) string {
	if v == nil {
		return "null"
	}
	if o, ok := v.(*fptypes.ObjectValue); ok {
		var obj map[string]any
		d := json.NewDecoder(strings.NewReader(string(o.Data())))
		d.UseNumber()
		if d.Decode(&obj) == nil {
			return fhirText("FHIR."+o.Type(), obj)
		}
	}
	if fptypes.IsFHIRPrimitive(v) {
		return "FHIR primitive of System." + v.Type() + " " + v.String()
	}
	switch n := v.(type) {
	case fptypes.Quantity:
		return quantityText(n.Value().String(), n.Unit())
	case fptypes.Integer:
		return "System.Integer " + n.String()
	case fptypes.Decimal:
		return "System.Decimal " + n.Value().String()
	case fptypes.Boolean:
		return fmt.Sprintf("System.Boolean %v", n.Bool())
	case fptypes.String:
		return "System.String " + n.Value()
	case fptypes.Date, fptypes.DateTime, fptypes.Time:
		return "System." + v.Type() + " " + v.String()
	case cqltypes.List:
		items := make([]string, len(n.Values))
		for i, e := range n.Values {
			items[i] = engineValueText(e)
		}
		return "List{" + strings.Join(items, ", ") + "}"
	case cqltypes.Tuple:
		fields := make([]string, 0, len(n.Elements))
		for _, k := range slices.Sorted(maps.Keys(n.Elements)) {
			fields = append(fields, k+": "+engineValueText(n.Elements[k]))
		}
		return "System." + n.Type() + " {" + strings.Join(fields, ", ") + "}"
	}
	return "other " + v.Type() + " " + v.String()
}

// fhirEngineAnswers is this engine's answer to every definition of the
// corpus, as typed text.
func fhirEngineAnswers(t *testing.T, defs map[string]string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", "fhir.cql"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(raw)
	header := src[:snapshotDefine.FindStringIndex(src)[0]]
	provider, patient := snapshotProvider(t)
	engine := NewEngine(WithDataProvider(provider))
	out := make(map[string]string, len(defs))
	for name, expr := range defs {
		v, err := engine.EvaluateExpression(context.Background(), header+"define "+name+": "+expr+"\n", name, patient, nil)
		out[name] = fhirEngineText(v, err)
	}
	return out
}

// classifiedByFHIROracle is the verdict the oracle gives a divergence.
func classifiedByFHIROracle(key string, d divergence, v fhirVerdict, engine string, reference json.RawMessage) divergence {
	name, _, _ := strings.Cut(key, "\t")
	evidence := "fhir oracle: " + v.reason + "; " + strings.TrimPrefix(fhirOracleRule, "fhir oracle (engine_fhir_oracle_test.go), ")
	if engine != v.expected {
		d.Class, d.Expected, d.Evidence = ownDefect, v.expected, evidence+"; this engine answers "+engine
		return d
	}
	refText := fhirReferenceText(reference)
	switch {
	case refText == "unreadable" && v.expected != "null" && !strings.HasPrefix(v.expected, "System.Boolean") &&
		v.expected != translationError && v.expected != runTimeError:
		d.Class, d.Expected = unclassified, ""
		d.Evidence = "the reference harness cannot encode the reference's answer; fhir oracle expects " + v.expected
	case refText == translationError && v.unrelated:
		d.Class, d.Expected = deliberate, d.Engine
		d.Evidence = "a cast between types with no static relation is valid CQL, decided in " + fhirDecisionPlan +
			" (PR 3, decision 1): the Java translator refuses it; " + evidence
	case refText == v.expected:
		d.Class, d.Expected = unclassified, ""
		d.Evidence = "the comparison reads the two answers apart, and typed they are one: " + v.expected + " (" + name + ")"
	case v.decided && refText != runTimeError:
		// The reference reads otherwise what the specification leaves
		// open. A run-time error is no reading: it stays a defect (D688).
		d.Class, d.Expected = deliberate, d.Engine
		d.Evidence = "fhir oracle: " + v.reason + ", which the reference reads otherwise; " +
			strings.TrimPrefix(fhirOracleRule, "fhir oracle (engine_fhir_oracle_test.go), ") + "; the reference answers " + refText
	default:
		d.Class, d.Expected, d.Evidence = referenceDefect, d.Engine, evidence+"; the reference answers "+refText
	}
	return d
}

// TestTheFHIRVerdictsFollowTheOracle holds every verdict of the fhir area to
// what the oracle computes for its definition: its class, its expected
// answer and its evidence. -classify-fhir rewrites them.
func TestTheFHIRVerdictsFollowTheOracle(t *testing.T) {
	defs := fhirDefinitions(t)
	c := readFHIRCorpus(t)
	engine := fhirEngineAnswers(t, defs)
	refs := map[string]map[string]json.RawMessage{
		"java": readReferenceResults(t, "fhir", "java"),
		"js":   readReferenceResults(t, "fhir", "js"),
	}
	path := filepath.Join("testdata", "snapshots", "fhir.divergences.tsv")
	recorded := readDivergences(t, path)
	classify := func(key string, d divergence) divergence {
		name, ref, _ := strings.Cut(key, "\t")
		v, err := c.fhirExpected(defs[name])
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return classifiedByFHIROracle(key, d, v, engine[name], refs[ref][name])
	}
	if *classifyFHIR {
		for key, d := range recorded {
			recorded[key] = classify(key, d)
		}
		writeVerdicts(t, path, recorded)
		return
	}
	for _, key := range slices.Sorted(maps.Keys(recorded)) {
		d := recorded[key]
		want := classify(key, d)
		if want.Class != d.Class || want.Expected != d.Expected || want.Evidence != d.Evidence {
			t.Errorf("%s: recorded %s expecting %q (%s); the oracle gives %s expecting %q (%s)",
				oneLine(key), d.Class, d.Expected, oneLine(d.Evidence), want.Class, want.Expected, oneLine(want.Evidence))
		}
	}
}

// fhirHiddenByTheComparison is the definitions where this engine agrees with
// the references only because the comparison reads a FHIR primitive by its
// value: this engine's typed answer, a FHIR primitive with no FHIR type
// (Option A, an own defect the user decided, PR 3, decision 2).
// D28, D70 and D787 joined them with Java 5.4.0, which answers their cast or `as` to Any
// with the FHIR value, where 3.26.0 failed resolving Any; fhir.own-defects.tsv
// keeps each defect (plan 007).
var fhirHiddenByTheComparison = map[string]string{
	"D19":  "FHIR primitive of System.Date 1980-01-01",
	"D28":  "FHIR primitive of System.String male",
	"D54":  "FHIR primitive of System.Date 1980-01-01",
	"D70":  "FHIR primitive of System.Integer 7",
	"D787": "FHIR primitive of System.String p",
	"D794": "FHIR primitive of System.String abc",
}

// TestTheFHIROracleAgreesWithTheReferencesWhereAllAgree holds the oracle to the
// definitions where this engine and the references give one answer: there,
// each reference that answers gives what the oracle expects, typed. It checks the oracle on what no divergence
// reaches. This engine is held to it too, but for the definitions of
// fhirHiddenByTheComparison, each with the answer it gives.
func TestTheFHIROracleAgreesWithTheReferencesWhereAllAgree(t *testing.T) {
	defs := fhirDefinitions(t)
	c := readFHIRCorpus(t)
	engine := fhirEngineAnswers(t, defs)
	recorded := readDivergences(t, filepath.Join("testdata", "snapshots", "fhir.divergences.tsv"))
	refs := map[string]map[string]json.RawMessage{
		"java": readReferenceResults(t, "fhir", "java"),
		"js":   readReferenceResults(t, "fhir", "js"),
	}
	agreed, hidden := 0, map[string]string{}
	for _, name := range slices.Sorted(maps.Keys(defs)) {
		_, javaDiffers := recorded[name+"\tjava"]
		_, jsDiffers := recorded[name+"\tjs"]
		if javaDiffers || jsDiffers {
			continue
		}
		v, err := c.fhirExpected(defs[name])
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, ref := range []string{"java", "js"} {
			raw, answered := refs[ref][name]
			if !answered {
				continue
			}
			agreed++
			if got := fhirReferenceText(raw); got != "unreadable" && got != v.expected {
				t.Errorf("%s: %s answers %s; the oracle expects %s (%s)", name, ref, got, v.expected, v.reason)
			}
		}
		if engine[name] != v.expected {
			hidden[name] = engine[name]
		}
	}
	if agreed == 0 {
		t.Fatal("no definition where all agree: the oracle is checked on nothing")
	}
	if !maps.Equal(hidden, fhirHiddenByTheComparison) {
		t.Errorf("where all agree, this engine's typed answer is not the expected one on %v; the known ones are %v",
			hidden, fhirHiddenByTheComparison)
	}
}

// The block of the plan that names the casts PR 3's decision 1 covers.
const (
	fhirDecisionBegin = "<!-- fhir decision 1, written by -classify-fhir: begin -->"
	fhirDecisionEnd   = "<!-- fhir decision 1: end -->"
)

// fhirUnrelatedCasts is every definition decision 1 covers: an as or a cast
// the Java translator refuses for its static types.
func fhirUnrelatedCasts(t *testing.T, c *fhirCorpus, defs map[string]string) []string {
	t.Helper()
	var names []string
	for name, expr := range defs {
		v, err := c.fhirExpected(expr)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if v.unrelated {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// TestTheFHIRDecisionNamesItsCasts holds the plan to the casts decision 1
// covers: the deliberate differences cite it, and it names each of them,
// as the oracle computes them. -classify-fhir writes the list.
func TestTheFHIRDecisionNamesItsCasts(t *testing.T) {
	defs := fhirDefinitions(t)
	want := strings.Join(fhirUnrelatedCasts(t, readFHIRCorpus(t), defs), " ")
	raw, err := os.ReadFile(fhirDecisionPlan)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	begin, end := strings.Index(doc, fhirDecisionBegin), strings.Index(doc, fhirDecisionEnd)
	if *classifyFHIR {
		block := fhirDecisionBegin + "\n" + want + "\n" + fhirDecisionEnd
		if begin < 0 || end < begin {
			t.Fatalf("%s has no block for decision 1 to write: %s … %s", fhirDecisionPlan, fhirDecisionBegin, fhirDecisionEnd)
		}
		doc = doc[:begin] + block + doc[end+len(fhirDecisionEnd):]
		if err := os.WriteFile(fhirDecisionPlan, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if begin < 0 || end < begin {
		t.Fatalf("%s does not name the casts decision 1 covers; write them with -classify-fhir", fhirDecisionPlan)
	}
	if got := strings.TrimSpace(doc[begin+len(fhirDecisionBegin) : end]); got != want {
		t.Errorf("%s names other casts than decision 1 covers; rewrite them with -classify-fhir", fhirDecisionPlan)
	}
}

// fhirBothReferencesWrong is the definitions where both references give one
// answer and it is not the specification's, with why.
var fhirBothReferencesWrong = map[string]string{
	"D194": "both take the Integer signature of Sum by the choice's static type, and the Quantity is null by then; ADR 003 (decision 4) takes the signature of the element's run-time type",
}

// TestTheFHIROracleAgreesWhereTheReferencesAgree holds the oracle to every
// definition where both references give one readable answer, whatever this
// engine answers: there, the oracle expects that answer. It reaches what the comparison of this engine cannot: a value of
// an instance this engine builds wrong, which no definition where all three
// agree holds.
func TestTheFHIROracleAgreesWhereTheReferencesAgree(t *testing.T) {
	defs := fhirDefinitions(t)
	c := readFHIRCorpus(t)
	java, js := readReferenceResults(t, "fhir", "java"), readReferenceResults(t, "fhir", "js")
	checked := 0
	used := map[string]bool{}
	for _, name := range slices.Sorted(maps.Keys(defs)) {
		a, okJava := java[name]
		b, okJS := js[name]
		if !okJava || !okJS {
			continue
		}
		answer := fhirReferenceText(a)
		if answer != fhirReferenceText(b) || answer == "unreadable" {
			continue
		}
		v, err := c.fhirExpected(defs[name])
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		checked++
		why, excepted := fhirBothReferencesWrong[name]
		used[name] = excepted
		switch {
		case excepted && v.expected == answer:
			t.Errorf("%s: both references answer %s, as the oracle expects; it is no exception (%s)", name, answer, why)
		case !excepted && v.expected != answer:
			t.Errorf("%s: both references answer %s; the oracle expects %s (%s)", name, answer, v.expected, v.reason)
		}
	}
	// An exception where the references no longer agree is checked nowhere
	// (D797, when the Java engine 5.4.0 stopped failing on Any).
	for _, name := range slices.Sorted(maps.Keys(fhirBothReferencesWrong)) {
		if !used[name] {
			t.Errorf("%s: the references no longer agree on it; remove it from fhirBothReferencesWrong", name)
		}
	}
	if checked == 0 {
		t.Fatal("no definition where both references agree: the oracle is checked on nothing")
	}
}

// fhirReferenceDefectValues are the values a reference answers wrong, by
// definition and reference, with why: the value differs from the oracle's, and
// the divergence manifest records it as the reference's defect. Each has to
// differ still.
var fhirReferenceDefectValues = map[string]string{
	"D345 java": valueSetCodesystems,
	"D788 java": valueSetCodesystems,
}

// valueSetCodesystems is why the Java engine 5.4.0 answers a ValueSet built
// with no codesystems otherwise than the oracle: it gives the element an empty
// list, and an element an instance selector does not name is null (CQL,
// Structured Values).
const valueSetCodesystems = "the Java engine 5.4.0 gives a ValueSet built without codesystems an empty list of them"

// TestTheFHIROracleValuesAreTheReferences holds the values the oracle expects
// to the references: wherever a reference answers a value of the type the
// oracle expects, it is the same value, whatever the verdict. A value the
// oracle reads or builds wrong — an instance's element, a choice's branch —
// differs from the reference that read it right.
func TestTheFHIROracleValuesAreTheReferences(t *testing.T) {
	defs := fhirDefinitions(t)
	c := readFHIRCorpus(t)
	checked := 0
	for _, ref := range []string{"java", "js"} {
		answers := readReferenceResults(t, "fhir", ref)
		for _, name := range slices.Sorted(maps.Keys(defs)) {
			raw, ok := answers[name]
			if !ok {
				continue
			}
			v, err := c.fhirExpected(defs[name])
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			got := fhirReferenceText(raw)
			wantType, _, _ := strings.Cut(v.expected, " ")
			gotType, _, _ := strings.Cut(got, " ")
			// A Boolean is a verdict, not a value read: the verdicts hold it.
			if wantType == "System.Boolean" || gotType != wantType ||
				!strings.HasPrefix(wantType, "FHIR.") && !strings.HasPrefix(wantType, "System.") {
				continue
			}
			checked++
			if why := fhirReferenceDefectValues[name+" "+ref]; why != "" {
				if got == v.expected {
					t.Errorf("%s: %s answers the oracle's value now; it is no reference defect any longer (%s)", name, ref, why)
				}
				continue
			}
			if got != v.expected {
				t.Errorf("%s: %s answers %s; the oracle expects %s (%s)", name, ref, got, v.expected, v.reason)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no value of the type the oracle expects: the oracle's values are checked on nothing")
	}
}
