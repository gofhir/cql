package elm

import (
	"testing"

	"github.com/gofhir/cql/ast"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Type operations: Is, As, Cast, Convert
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_IsExpression(t *testing.T) {
	node := TranslateExpression(&ast.IsExpression{
		Operand: &ast.IdentifierRef{Name: "val"},
		Type:    &ast.NamedType{Namespace: "FHIR", Name: "Patient"},
	})
	if node.Type != "Is" {
		t.Errorf("expected Is, got %s", node.Type)
	}
	if node.IsTypeSpecifier == nil {
		t.Fatal("expected IsTypeSpecifier")
	}
	if node.IsTypeSpecifier.Name != "Patient" {
		t.Errorf("expected Patient, got %s", node.IsTypeSpecifier.Name)
	}
}

func TestImport_IsExpression(t *testing.T) {
	node := ImportExpression(&ExpressionNode{
		Type:    "Is",
		Operand: &ExpressionNode{Type: "ExpressionRef", Name: "val"},
		IsTypeSpecifier: &TypeSpecifier{
			Type:      "NamedTypeSpecifier",
			Namespace: "FHIR",
			Name:      "Patient",
		},
	})
	is, ok := node.(*ast.IsExpression)
	if !ok {
		t.Fatalf("expected IsExpression, got %T", node)
	}
	nt := is.Type.(*ast.NamedType)
	if nt.Name != "Patient" {
		t.Errorf("expected Patient, got %s", nt.Name)
	}
}

func TestTranslate_AsExpression(t *testing.T) {
	node := TranslateExpression(&ast.AsExpression{
		Operand: &ast.IdentifierRef{Name: "val"},
		Type:    &ast.NamedType{Name: "String"},
	})
	if node.Type != "As" {
		t.Errorf("expected As, got %s", node.Type)
	}
	if node.Strict {
		t.Error("expected Strict=false for AsExpression")
	}
	if node.AsTypeSpecifier == nil || node.AsTypeSpecifier.Name != "String" {
		t.Error("expected AsTypeSpecifier with String")
	}
}

func TestTranslate_CastExpression(t *testing.T) {
	node := TranslateExpression(&ast.CastExpression{
		Operand: &ast.IdentifierRef{Name: "val"},
		Type:    &ast.NamedType{Name: "Integer"},
	})
	if node.Type != "As" {
		t.Errorf("expected As (strict), got %s", node.Type)
	}
	if !node.Strict {
		t.Error("expected Strict=true for CastExpression")
	}
}

func TestImport_As_Strict(t *testing.T) {
	node := ImportExpression(&ExpressionNode{
		Type:    "As",
		Operand: &ExpressionNode{Type: "ExpressionRef", Name: "val"},
		AsTypeSpecifier: &TypeSpecifier{
			Type: "NamedTypeSpecifier",
			Name: "Integer",
		},
		Strict: true,
	})
	cast, ok := node.(*ast.CastExpression)
	if !ok {
		t.Fatalf("expected CastExpression for strict As, got %T", node)
	}
	nt := cast.Type.(*ast.NamedType)
	if nt.Name != "Integer" {
		t.Errorf("expected Integer, got %s", nt.Name)
	}
}

func TestTranslate_ConvertExpression(t *testing.T) {
	node := TranslateExpression(&ast.ConvertExpression{
		Operand: &ast.Literal{ValueType: ast.LiteralString, Value: "42"},
		ToType:  &ast.NamedType{Name: "Integer"},
	})
	if node.Type != "Convert" {
		t.Errorf("expected Convert, got %s", node.Type)
	}
	if node.ToType == nil || node.ToType.Name != "Integer" {
		t.Error("expected ToType Integer")
	}
}

func TestImport_ConvertExpression(t *testing.T) {
	node := ImportExpression(&ExpressionNode{
		Type:    "Convert",
		Operand: &ExpressionNode{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}String", Value: "42"},
		ToType:  &TypeSpecifier{Type: "NamedTypeSpecifier", Name: "Integer"},
	})
	conv, ok := node.(*ast.ConvertExpression)
	if !ok {
		t.Fatalf("expected ConvertExpression, got %T", node)
	}
	nt := conv.ToType.(*ast.NamedType)
	if nt.Name != "Integer" {
		t.Errorf("expected Integer, got %s", nt.Name)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// BooleanTest translate
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_BooleanTest_IsTrue(t *testing.T) {
	node := TranslateExpression(&ast.BooleanTestExpression{
		Operand:   &ast.IdentifierRef{Name: "x"},
		TestValue: "true",
		Not:       false,
	})
	if node.Type != "IsTrue" {
		t.Errorf("expected IsTrue, got %s", node.Type)
	}
}

func TestTranslate_BooleanTest_IsFalse(t *testing.T) {
	node := TranslateExpression(&ast.BooleanTestExpression{
		Operand:   &ast.IdentifierRef{Name: "x"},
		TestValue: "false",
		Not:       false,
	})
	if node.Type != "IsFalse" {
		t.Errorf("expected IsFalse, got %s", node.Type)
	}
}

func TestTranslate_BooleanTest_IsNull(t *testing.T) {
	node := TranslateExpression(&ast.BooleanTestExpression{
		Operand:   &ast.IdentifierRef{Name: "x"},
		TestValue: "null",
		Not:       false,
	})
	if node.Type != "IsNull" {
		t.Errorf("expected IsNull, got %s", node.Type)
	}
}

func TestTranslate_BooleanTest_IsNotNull(t *testing.T) {
	node := TranslateExpression(&ast.BooleanTestExpression{
		Operand:   &ast.IdentifierRef{Name: "x"},
		TestValue: "null",
		Not:       true,
	})
	if node.Type != "Not" {
		t.Errorf("expected Not (wrapping IsNull), got %s", node.Type)
	}
	inner, ok := node.Operand.(*ExpressionNode)
	if !ok {
		t.Fatalf("expected single operand, got %T", node.Operand)
	}
	if inner.Type != "IsNull" {
		t.Errorf("expected inner IsNull, got %s", inner.Type)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Case expression translate
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_CaseExpression(t *testing.T) {
	node := TranslateExpression(&ast.CaseExpression{
		Comparand: &ast.IdentifierRef{Name: "status"},
		Items: []*ast.CaseItem{
			{
				When: &ast.Literal{ValueType: ast.LiteralString, Value: "active"},
				Then: &ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
			},
			{
				When: &ast.Literal{ValueType: ast.LiteralString, Value: "inactive"},
				Then: &ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
			},
		},
		Else: &ast.Literal{ValueType: ast.LiteralInteger, Value: "0"},
	})
	if node.Type != "Case" {
		t.Errorf("expected Case, got %s", node.Type)
	}
	if node.Comparand == nil {
		t.Error("expected non-nil Comparand")
	}
	if len(node.CaseItem) != 2 {
		t.Errorf("expected 2 case items, got %d", len(node.CaseItem))
	}
	if node.Else == nil || node.Else.Value != "0" {
		t.Error("expected Else with value 0")
	}
}

func TestImport_CaseExpression(t *testing.T) {
	node := ImportExpression(&ExpressionNode{
		Type:      "Case",
		Comparand: &ExpressionNode{Type: "ExpressionRef", Name: "status"},
		CaseItem: []*CaseItem{
			{
				When: &ExpressionNode{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}String", Value: "active"},
				Then: &ExpressionNode{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Integer", Value: "1"},
			},
		},
		Else: &ExpressionNode{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Integer", Value: "0"},
	})
	c, ok := node.(*ast.CaseExpression)
	if !ok {
		t.Fatalf("expected CaseExpression, got %T", node)
	}
	if c.Comparand == nil {
		t.Error("expected non-nil Comparand")
	}
	if len(c.Items) != 1 {
		t.Errorf("expected 1 case item, got %d", len(c.Items))
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Duration/Difference between
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_DurationBetween(t *testing.T) {
	node := TranslateExpression(&ast.DurationBetween{
		Precision: "Year",
		Low:       &ast.Literal{ValueType: ast.LiteralDate, Value: "2020-01-01"},
		High:      &ast.Literal{ValueType: ast.LiteralDate, Value: "2024-01-01"},
	})
	if node.Type != "DurationBetween" {
		t.Errorf("expected DurationBetween, got %s", node.Type)
	}
	if node.Precision != "Year" {
		t.Errorf("expected Year precision, got %s", node.Precision)
	}
	ops, ok := node.Operand.([]*ExpressionNode)
	if !ok || len(ops) != 2 {
		t.Error("expected 2 operands")
	}
}

func TestImport_DurationBetween(t *testing.T) {
	node := ImportExpression(&ExpressionNode{
		Type:      "DurationBetween",
		Precision: "Month",
		Operand: []*ExpressionNode{
			{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Date", Value: "2024-01-01"},
			{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Date", Value: "2024-06-01"},
		},
	})
	db, ok := node.(*ast.DurationBetween)
	if !ok {
		t.Fatalf("expected DurationBetween, got %T", node)
	}
	if db.Precision != "Month" {
		t.Errorf("expected Month, got %s", db.Precision)
	}
}

func TestTranslate_DifferenceBetween(t *testing.T) {
	node := TranslateExpression(&ast.DifferenceBetween{
		Precision: "Day",
		Low:       &ast.Literal{ValueType: ast.LiteralDate, Value: "2024-01-01"},
		High:      &ast.Literal{ValueType: ast.LiteralDate, Value: "2024-01-31"},
	})
	if node.Type != "DifferenceBetween" {
		t.Errorf("expected DifferenceBetween, got %s", node.Type)
	}
	if node.Precision != "Day" {
		t.Errorf("expected Day, got %s", node.Precision)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Code / Concept expressions
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_CodeExpression(t *testing.T) {
	node := TranslateExpression(&ast.CodeExpression{
		Code:    "1234-5",
		System:  "LOINC",
		Display: "Hemoglobin A1c",
	})
	if node.Type != "Code" {
		t.Errorf("expected Code, got %s", node.Type)
	}
	if node.CodeValue != "1234-5" {
		t.Errorf("expected 1234-5, got %s", node.CodeValue)
	}
	if node.System == nil || node.System.Name != "LOINC" {
		t.Error("expected LOINC system")
	}
	if node.Display != "Hemoglobin A1c" {
		t.Errorf("expected Hemoglobin A1c, got %s", node.Display)
	}
}

func TestImport_CodeExpression(t *testing.T) {
	node := ImportExpression(&ExpressionNode{
		Type:      "Code",
		CodeValue: "1234-5",
		System:    &ExpressionNode{Type: "CodeSystemRef", Name: "LOINC"},
		Display:   "Hemoglobin A1c",
	})
	ce, ok := node.(*ast.CodeExpression)
	if !ok {
		t.Fatalf("expected CodeExpression, got %T", node)
	}
	if ce.Code != "1234-5" {
		t.Errorf("expected 1234-5, got %s", ce.Code)
	}
	if ce.System != "LOINC" {
		t.Errorf("expected LOINC, got %s", ce.System)
	}
}

func TestTranslate_ConceptExpression(t *testing.T) {
	node := TranslateExpression(&ast.ConceptExpression{
		Display: "Diabetes",
		Codes: []*ast.CodeExpression{
			{Code: "73211009", System: "SNOMED"},
			{Code: "E11", System: "ICD10"},
		},
	})
	if node.Type != "Concept" {
		t.Errorf("expected Concept, got %s", node.Type)
	}
	if node.Display != "Diabetes" {
		t.Errorf("expected Diabetes display, got %s", node.Display)
	}
	if len(node.Element) != 2 {
		t.Errorf("expected 2 code elements, got %d", len(node.Element))
	}
}

func TestImport_ConceptExpression(t *testing.T) {
	node := ImportExpression(&ExpressionNode{
		Type:    "Concept",
		Display: "Diabetes",
		Element: []*ExpressionNode{
			{Type: "Code", CodeValue: "73211009", System: &ExpressionNode{Type: "CodeSystemRef", Name: "SNOMED"}},
		},
	})
	ce, ok := node.(*ast.ConceptExpression)
	if !ok {
		t.Fatalf("expected ConceptExpression, got %T", node)
	}
	if ce.Display != "Diabetes" {
		t.Errorf("expected Diabetes, got %s", ce.Display)
	}
	if len(ce.Codes) != 1 || ce.Codes[0].Code != "73211009" {
		t.Error("expected 1 code with 73211009")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Instance expression
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_InstanceExpression(t *testing.T) {
	node := TranslateExpression(&ast.InstanceExpression{
		Type: &ast.NamedType{Namespace: "FHIR", Name: "Quantity"},
		Elements: []*ast.TupleElement{
			{Name: "value", Expression: &ast.Literal{ValueType: ast.LiteralDecimal, Value: "5.0"}},
			{Name: "unit", Expression: &ast.Literal{ValueType: ast.LiteralString, Value: "mg"}},
		},
	})
	if node.Type != "Instance" {
		t.Errorf("expected Instance, got %s", node.Type)
	}
	if node.ClassType != "{http://hl7.org/fhir}Quantity" {
		t.Errorf("expected {http://hl7.org/fhir}Quantity, got %s", node.ClassType)
	}
	if len(node.Element) != 2 {
		t.Errorf("expected 2 elements, got %d", len(node.Element))
	}
}

func TestImport_InstanceExpression(t *testing.T) {
	node := ImportExpression(&ExpressionNode{
		Type:      "Instance",
		ClassType: "{http://hl7.org/fhir}Quantity",
		Element: []*ExpressionNode{
			{Type: "InstanceElement", Name: "value", Source: &ExpressionNode{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Decimal", Value: "5.0"}},
		},
	})
	ie, ok := node.(*ast.InstanceExpression)
	if !ok {
		t.Fatalf("expected InstanceExpression, got %T", node)
	}
	if ie.Type.Name != "Quantity" {
		t.Errorf("expected Quantity, got %s", ie.Type.Name)
	}
	if ie.Type.Namespace != "FHIR" {
		t.Errorf("expected FHIR namespace, got %s", ie.Type.Namespace)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Special tokens: This, IterationIndex, Total
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_SpecialTokens(t *testing.T) {
	tests := []struct {
		expr     ast.Expression
		expected string
	}{
		{&ast.ThisExpression{}, "This"},
		{&ast.IndexExpression{}, "IterationIndex"},
		{&ast.TotalExpression{}, "Total"},
	}
	for _, tt := range tests {
		node := TranslateExpression(tt.expr)
		if node.Type != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, node.Type)
		}
	}
}

func TestImport_SpecialTokens(t *testing.T) {
	tests := []struct {
		elmType    string
		expectType string
	}{
		{"This", "*ast.ThisExpression"},
		{"IterationIndex", "*ast.IndexExpression"},
		{"Total", "*ast.TotalExpression"},
	}
	for _, tt := range tests {
		node := ImportExpression(&ExpressionNode{Type: tt.elmType})
		switch tt.elmType {
		case "This":
			if _, ok := node.(*ast.ThisExpression); !ok {
				t.Errorf("expected ThisExpression for %s, got %T", tt.elmType, node)
			}
		case "IterationIndex":
			if _, ok := node.(*ast.IndexExpression); !ok {
				t.Errorf("expected IndexExpression for %s, got %T", tt.elmType, node)
			}
		case "Total":
			if _, ok := node.(*ast.TotalExpression); !ok {
				t.Errorf("expected TotalExpression for %s, got %T", tt.elmType, node)
			}
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Timing expressions
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_TimingBefore(t *testing.T) {
	node := TranslateExpression(&ast.TimingExpression{
		Left:  &ast.IdentifierRef{Name: "A"},
		Right: &ast.IdentifierRef{Name: "B"},
		Operator: ast.TimingOp{
			Kind:      ast.TimingBeforeOrAfter,
			Before:    true,
			Precision: "Day",
		},
	})
	if node.Type != "Before" {
		t.Errorf("expected Before, got %s", node.Type)
	}
	if node.Precision != "Day" {
		t.Errorf("expected Day, got %s", node.Precision)
	}
}

func TestTranslate_TimingAfter(t *testing.T) {
	node := TranslateExpression(&ast.TimingExpression{
		Left:  &ast.IdentifierRef{Name: "A"},
		Right: &ast.IdentifierRef{Name: "B"},
		Operator: ast.TimingOp{
			Kind:  ast.TimingBeforeOrAfter,
			After: true,
		},
	})
	if node.Type != "After" {
		t.Errorf("expected After, got %s", node.Type)
	}
}

func TestImport_TimingBefore(t *testing.T) {
	node := ImportExpression(&ExpressionNode{
		Type:      "Before",
		Precision: "Day",
		Operand: []*ExpressionNode{
			{Type: "ExpressionRef", Name: "A"},
			{Type: "ExpressionRef", Name: "B"},
		},
	})
	te, ok := node.(*ast.TimingExpression)
	if !ok {
		t.Fatalf("expected TimingExpression, got %T", node)
	}
	if te.Operator.Kind != ast.TimingBeforeOrAfter {
		t.Errorf("expected TimingBeforeOrAfter kind")
	}
	if !te.Operator.Before {
		t.Error("expected Before=true")
	}
	if te.Operator.Precision != "Day" {
		t.Errorf("expected Day precision, got %s", te.Operator.Precision)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Membership expression
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_MembershipExpression_In(t *testing.T) {
	node := TranslateExpression(&ast.MembershipExpression{
		Operator:  "in",
		Left:      &ast.IdentifierRef{Name: "code"},
		Right:     &ast.IdentifierRef{Name: "vs"},
		Precision: "Day",
	})
	if node.Type != "In" {
		t.Errorf("expected In, got %s", node.Type)
	}
	if node.Precision != "Day" {
		t.Errorf("expected Day, got %s", node.Precision)
	}
}

func TestTranslate_MembershipExpression_Contains(t *testing.T) {
	node := TranslateExpression(&ast.MembershipExpression{
		Operator: "contains",
		Left:     &ast.IdentifierRef{Name: "list"},
		Right:    &ast.IdentifierRef{Name: "item"},
	})
	if node.Type != "Contains" {
		t.Errorf("expected Contains, got %s", node.Type)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Tuple expression translate
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_TupleExpression(t *testing.T) {
	node := TranslateExpression(&ast.TupleExpression{
		Elements: []*ast.TupleElement{
			{Name: "name", Expression: &ast.Literal{ValueType: ast.LiteralString, Value: "John"}},
			{Name: "age", Expression: &ast.Literal{ValueType: ast.LiteralInteger, Value: "30"}},
		},
	})
	if node.Type != "Tuple" {
		t.Errorf("expected Tuple, got %s", node.Type)
	}
	if len(node.Element) != 2 {
		t.Errorf("expected 2 elements, got %d", len(node.Element))
	}
	if node.Element[0].Name != "name" {
		t.Errorf("expected 'name', got %s", node.Element[0].Name)
	}
}

func TestImport_TupleExpression(t *testing.T) {
	node := ImportExpression(&ExpressionNode{
		Type: "Tuple",
		Element: []*ExpressionNode{
			{Type: "TupleElement", Name: "x", Source: &ExpressionNode{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Integer", Value: "1"}},
		},
	})
	te, ok := node.(*ast.TupleExpression)
	if !ok {
		t.Fatalf("expected TupleExpression, got %T", node)
	}
	if len(te.Elements) != 1 {
		t.Errorf("expected 1 element, got %d", len(te.Elements))
	}
	if te.Elements[0].Name != "x" {
		t.Errorf("expected 'x', got %s", te.Elements[0].Name)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// IndexAccess (Indexer)
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_IndexAccess(t *testing.T) {
	node := TranslateExpression(&ast.IndexAccess{
		Source: &ast.IdentifierRef{Name: "list"},
		Index:  &ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
	})
	if node.Type != "Indexer" {
		t.Errorf("expected Indexer, got %s", node.Type)
	}
	ops, ok := node.Operand.([]*ExpressionNode)
	if !ok || len(ops) != 2 {
		t.Error("expected 2 operands for Indexer")
	}
}

func TestImport_IndexAccess(t *testing.T) {
	node := ImportExpression(&ExpressionNode{
		Type: "Indexer",
		Operand: []*ExpressionNode{
			{Type: "ExpressionRef", Name: "list"},
			{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Integer", Value: "2"},
		},
	})
	ia, ok := node.(*ast.IndexAccess)
	if !ok {
		t.Fatalf("expected IndexAccess, got %T", node)
	}
	if ia.Source == nil || ia.Index == nil {
		t.Error("expected non-nil Source and Index")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// ExternalConstant / ParameterRef
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_ExternalConstant(t *testing.T) {
	node := TranslateExpression(&ast.ExternalConstant{Name: "MeasurementPeriod"})
	if node.Type != "ParameterRef" {
		t.Errorf("expected ParameterRef, got %s", node.Type)
	}
	if node.Name != "MeasurementPeriod" {
		t.Errorf("expected MeasurementPeriod, got %s", node.Name)
	}
}

func TestImport_ParameterRef(t *testing.T) {
	node := ImportExpression(&ExpressionNode{Type: "ParameterRef", Name: "MeasurementPeriod"})
	ec, ok := node.(*ast.ExternalConstant)
	if !ok {
		t.Fatalf("expected ExternalConstant, got %T", node)
	}
	if ec.Name != "MeasurementPeriod" {
		t.Errorf("expected MeasurementPeriod, got %s", ec.Name)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// MemberAccess / Property
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_MemberAccess(t *testing.T) {
	node := TranslateExpression(&ast.MemberAccess{
		Source: &ast.IdentifierRef{Name: "Patient"},
		Member: "birthDate",
	})
	if node.Type != "Property" {
		t.Errorf("expected Property, got %s", node.Type)
	}
	if node.Path != "birthDate" {
		t.Errorf("expected birthDate, got %s", node.Path)
	}
}

func TestImport_Property(t *testing.T) {
	node := ImportExpression(&ExpressionNode{
		Type:   "Property",
		Path:   "birthDate",
		Source: &ExpressionNode{Type: "ExpressionRef", Name: "Patient"},
	})
	ma, ok := node.(*ast.MemberAccess)
	if !ok {
		t.Fatalf("expected MemberAccess, got %T", node)
	}
	if ma.Member != "birthDate" {
		t.Errorf("expected birthDate, got %s", ma.Member)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Query with sort/aggregate/relationships
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_QueryWithSort(t *testing.T) {
	node := TranslateExpression(&ast.Query{
		Sources: []*ast.AliasedSource{
			{Source: &ast.Retrieve{ResourceType: &ast.NamedType{Name: "Observation"}}, Alias: "O"},
		},
		Sort: &ast.SortClause{
			ByItems: []*ast.SortByItem{
				{
					Direction:  ast.SortDesc,
					Expression: &ast.MemberAccess{Source: &ast.IdentifierRef{Name: "O"}, Member: "effectiveDateTime"},
				},
			},
		},
	})
	if node.Sort == nil {
		t.Fatal("expected Sort clause")
	}
	if len(node.Sort.By) != 1 {
		t.Errorf("expected 1 sort item, got %d", len(node.Sort.By))
	}
	if node.Sort.By[0].Direction != "desc" {
		t.Errorf("expected desc, got %s", node.Sort.By[0].Direction)
	}
}

func TestTranslate_QueryWithRelationships(t *testing.T) {
	node := TranslateExpression(&ast.Query{
		Sources: []*ast.AliasedSource{
			{Source: &ast.Retrieve{ResourceType: &ast.NamedType{Name: "Encounter"}}, Alias: "E"},
		},
		With: []*ast.WithClause{
			{
				Source:    &ast.AliasedSource{Source: &ast.Retrieve{ResourceType: &ast.NamedType{Name: "Condition"}}, Alias: "C"},
				Condition: &ast.BinaryExpression{Operator: ast.OpEqual, Left: &ast.IdentifierRef{Name: "E"}, Right: &ast.IdentifierRef{Name: "C"}},
			},
		},
		Without: []*ast.WithoutClause{
			{
				Source:    &ast.AliasedSource{Source: &ast.Retrieve{ResourceType: &ast.NamedType{Name: "Procedure"}}, Alias: "P"},
				Condition: &ast.BinaryExpression{Operator: ast.OpEqual, Left: &ast.IdentifierRef{Name: "E"}, Right: &ast.IdentifierRef{Name: "P"}},
			},
		},
	})
	if len(node.Relationship) != 2 {
		t.Fatalf("expected 2 relationships (with + without), got %d", len(node.Relationship))
	}
	if node.Relationship[0].Type != "With" {
		t.Errorf("expected With, got %s", node.Relationship[0].Type)
	}
	if node.Relationship[1].Type != "Without" {
		t.Errorf("expected Without, got %s", node.Relationship[1].Type)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Include definition
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_IncludeDef(t *testing.T) {
	lib := &ast.Library{
		Includes: []*ast.IncludeDef{
			{Name: "FHIRHelpers", Version: "4.0.1", Alias: "FHIRHelpers"},
		},
	}
	elm := Translate(lib)
	if elm.Includes == nil || len(elm.Includes.Def) != 1 {
		t.Fatal("expected 1 include")
	}
	inc := elm.Includes.Def[0]
	if inc.Path != "FHIRHelpers" {
		t.Errorf("expected FHIRHelpers path, got %s", inc.Path)
	}
	if inc.LocalIdentifier != "FHIRHelpers" {
		t.Errorf("expected FHIRHelpers alias, got %s", inc.LocalIdentifier)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// TypeSpecifier: TupleType and ChoiceType
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_TupleTypeSpecifier(t *testing.T) {
	ts := translateTypeSpecifier(&ast.TupleType{
		Elements: []*ast.TupleElementDef{
			{Name: "x", Type: &ast.NamedType{Name: "Integer"}},
			{Name: "y", Type: &ast.NamedType{Name: "String"}},
		},
	})
	if ts.Type != "TupleTypeSpecifier" {
		t.Errorf("expected TupleTypeSpecifier, got %s", ts.Type)
	}
	if len(ts.Element) != 2 {
		t.Errorf("expected 2 elements, got %d", len(ts.Element))
	}
}

func TestTranslate_ChoiceTypeSpecifier(t *testing.T) {
	ts := translateTypeSpecifier(&ast.ChoiceType{
		Types: []ast.TypeSpecifier{
			&ast.NamedType{Name: "Integer"},
			&ast.NamedType{Name: "String"},
		},
	})
	if ts.Type != "ChoiceTypeSpecifier" {
		t.Errorf("expected ChoiceTypeSpecifier, got %s", ts.Type)
	}
	if len(ts.Choice) != 2 {
		t.Errorf("expected 2 choices, got %d", len(ts.Choice))
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// BetweenExpression
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_BetweenExpression(t *testing.T) {
	node := TranslateExpression(&ast.BetweenExpression{
		Operand: &ast.IdentifierRef{Name: "x"},
		Low:     &ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
		High:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "10"},
	})
	if node.Type != "IncludedIn" {
		t.Errorf("expected IncludedIn, got %s", node.Type)
	}
}

func TestTranslate_BetweenExpression_Properly(t *testing.T) {
	node := TranslateExpression(&ast.BetweenExpression{
		Operand:  &ast.IdentifierRef{Name: "x"},
		Low:      &ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
		High:     &ast.Literal{ValueType: ast.LiteralInteger, Value: "10"},
		Properly: true,
	})
	if node.Type != "ProperIncludedIn" {
		t.Errorf("expected ProperIncludedIn, got %s", node.Type)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// DateTimeComponentFrom
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_DateTimeComponentFrom(t *testing.T) {
	node := TranslateExpression(&ast.DateTimeComponentFrom{
		Component: "Year",
		Operand:   &ast.IdentifierRef{Name: "birthDate"},
	})
	if node.Type != "DateTimeComponentFrom" {
		t.Errorf("expected DateTimeComponentFrom, got %s", node.Type)
	}
	if node.Precision != "Year" {
		t.Errorf("expected Year, got %s", node.Precision)
	}
}

func TestImport_DateTimeComponentFrom(t *testing.T) {
	node := ImportExpression(&ExpressionNode{
		Type:      "DateTimeComponentFrom",
		Precision: "Month",
		Operand:   &ExpressionNode{Type: "ExpressionRef", Name: "dt"},
	})
	dtc, ok := node.(*ast.DateTimeComponentFrom)
	if !ok {
		t.Fatalf("expected DateTimeComponentFrom, got %T", node)
	}
	if dtc.Component != "Month" {
		t.Errorf("expected Month, got %s", dtc.Component)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Nil expression handling
// ═══════════════════════════════════════════════════════════════════════════════

func TestTranslate_NilExpression(t *testing.T) {
	node := TranslateExpression(nil)
	if node != nil {
		t.Error("expected nil for nil input")
	}
}

func TestImport_NilExpression(t *testing.T) {
	node := ImportExpression(nil)
	if node != nil {
		t.Error("expected nil for nil input")
	}
}
