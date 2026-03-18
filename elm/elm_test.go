package elm

import (
	"encoding/json"
	"testing"

	"github.com/gofhir/cql/ast"
)

// ---------------------------------------------------------------------------
// Translate (AST → ELM)
// ---------------------------------------------------------------------------

func TestTranslate_SimpleLibrary(t *testing.T) {
	lib := &ast.Library{
		Identifier: &ast.LibraryIdentifier{Name: "TestLib", Version: "1.0.0"},
		Usings: []*ast.UsingDef{
			{Name: "FHIR", Version: "4.0.1"},
		},
		Contexts: []*ast.ContextDef{
			{Name: "Patient"},
		},
		Statements: []*ast.ExpressionDef{
			{
				Name:    "InDemographic",
				Context: "Patient",
				Expression: &ast.BinaryExpression{
					Operator: ast.OpGreaterOrEqual,
					Left: &ast.FunctionCall{
						Name:     "AgeInYears",
						Operands: nil,
					},
					Right: &ast.Literal{ValueType: ast.LiteralInteger, Value: "18"},
				},
			},
		},
	}

	elm := Translate(lib)
	if elm == nil {
		t.Fatal("expected non-nil ELM library")
	}

	if elm.Identifier.ID != "TestLib" {
		t.Errorf("expected ID=TestLib, got %s", elm.Identifier.ID)
	}
	if elm.Identifier.Version != "1.0.0" {
		t.Errorf("expected version=1.0.0, got %s", elm.Identifier.Version)
	}
	if elm.SchemaIdentifier.ID != "urn:hl7-org:elm" {
		t.Errorf("expected schema=urn:hl7-org:elm, got %s", elm.SchemaIdentifier.ID)
	}
	if len(elm.Usings.Def) != 1 {
		t.Errorf("expected 1 using, got %d", len(elm.Usings.Def))
	}
	if elm.Usings.Def[0].LocalIdentifier != "FHIR" {
		t.Errorf("expected FHIR using, got %s", elm.Usings.Def[0].LocalIdentifier)
	}
	if len(elm.Contexts.Def) != 1 || elm.Contexts.Def[0].Name != "Patient" {
		t.Error("expected Patient context")
	}
	if len(elm.Statements.Def) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(elm.Statements.Def))
	}
	stmt := elm.Statements.Def[0]
	if stmt.Name != "InDemographic" {
		t.Errorf("expected InDemographic, got %s", stmt.Name)
	}
	if stmt.Expression.Type != "GreaterOrEqual" {
		t.Errorf("expected GreaterOrEqual, got %s", stmt.Expression.Type)
	}
}

func TestTranslate_NilLibrary(t *testing.T) {
	if Translate(nil) != nil {
		t.Error("expected nil for nil input")
	}
}

func TestTranslate_Retrieve(t *testing.T) {
	lib := &ast.Library{
		Statements: []*ast.ExpressionDef{
			{
				Name: "Conditions",
				Expression: &ast.Retrieve{
					ResourceType: &ast.NamedType{Namespace: "FHIR", Name: "Condition"},
					CodePath:     "code",
					Codes:        &ast.IdentifierRef{Name: "Diabetes VS"},
				},
			},
		},
	}

	elm := Translate(lib)
	expr := elm.Statements.Def[0].Expression
	if expr.Type != "Retrieve" {
		t.Errorf("expected Retrieve, got %s", expr.Type)
	}
	if expr.DataType != "{http://hl7.org/fhir}Condition" {
		t.Errorf("expected FHIR Condition, got %s", expr.DataType)
	}
	if expr.CodeProperty != "code" {
		t.Errorf("expected code property, got %s", expr.CodeProperty)
	}
	if expr.Codes == nil || expr.Codes.Type != "ExpressionRef" {
		t.Error("expected ExpressionRef for codes")
	}
}

func TestTranslate_AllLiteralTypes(t *testing.T) {
	tests := []struct {
		literal  ast.LiteralType
		expected string
	}{
		{ast.LiteralNull, "Null"},
		{ast.LiteralBoolean, "{urn:hl7-org:elm-types:r1}Boolean"},
		{ast.LiteralString, "{urn:hl7-org:elm-types:r1}String"},
		{ast.LiteralInteger, "{urn:hl7-org:elm-types:r1}Integer"},
		{ast.LiteralDecimal, "{urn:hl7-org:elm-types:r1}Decimal"},
		{ast.LiteralDate, "{urn:hl7-org:elm-types:r1}Date"},
		{ast.LiteralDateTime, "{urn:hl7-org:elm-types:r1}DateTime"},
		{ast.LiteralTime, "{urn:hl7-org:elm-types:r1}Time"},
	}

	for _, tt := range tests {
		node := TranslateExpression(&ast.Literal{ValueType: tt.literal, Value: "test"})
		if tt.literal == ast.LiteralNull {
			if node.Type != "Null" {
				t.Errorf("null literal: expected type=Null, got %s", node.Type)
			}
		} else if node.ValueType != tt.expected {
			t.Errorf("literal %d: expected %s, got %s", tt.literal, tt.expected, node.ValueType)
		}
	}
}

func TestTranslate_BinaryOps(t *testing.T) {
	ops := map[ast.BinaryOp]string{
		ast.OpAdd:      "Add",
		ast.OpEqual:    "Equal",
		ast.OpAnd:      "And",
		ast.OpOr:       "Or",
		ast.OpUnion:    "Union",
		ast.OpIn:       "In",
		ast.OpContains: "Contains",
	}

	for op, expected := range ops {
		node := TranslateExpression(&ast.BinaryExpression{
			Operator: op,
			Left:     &ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
			Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
		})
		if node.Type != expected {
			t.Errorf("op %d: expected %s, got %s", op, expected, node.Type)
		}
		ops, ok := node.Operand.([]*ExpressionNode)
		if !ok || len(ops) != 2 {
			t.Errorf("op %d: expected 2 operands", op)
		}
	}
}

func TestTranslate_UnaryOps(t *testing.T) {
	ops := map[ast.UnaryOp]string{
		ast.OpNot:      "Not",
		ast.OpExists:   "Exists",
		ast.OpDistinct: "Distinct",
		ast.OpFlatten:  "Flatten",
	}

	for op, expected := range ops {
		node := TranslateExpression(&ast.UnaryExpression{
			Operator: op,
			Operand:  &ast.Literal{ValueType: ast.LiteralBoolean, Value: "true"},
		})
		if node.Type != expected {
			t.Errorf("op %d: expected %s, got %s", op, expected, node.Type)
		}
	}
}

func TestTranslate_IfThenElse(t *testing.T) {
	node := TranslateExpression(&ast.IfThenElse{
		Condition: &ast.Literal{ValueType: ast.LiteralBoolean, Value: "true"},
		Then:      &ast.Literal{ValueType: ast.LiteralString, Value: "yes"},
		Else:      &ast.Literal{ValueType: ast.LiteralString, Value: "no"},
	})
	if node.Type != "If" {
		t.Errorf("expected If, got %s", node.Type)
	}
	if node.Condition == nil || node.Then == nil || node.Else == nil {
		t.Error("expected non-nil condition/then/else")
	}
}

func TestTranslate_Interval(t *testing.T) {
	node := TranslateExpression(&ast.IntervalExpression{
		LowClosed:  true,
		HighClosed: false,
		Low:        &ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
		High:       &ast.Literal{ValueType: ast.LiteralInteger, Value: "10"},
	})
	if node.Type != "Interval" {
		t.Errorf("expected Interval, got %s", node.Type)
	}
	if node.LowClosed == nil || *node.LowClosed != true {
		t.Error("expected lowClosed=true")
	}
	if node.HighClosed == nil || *node.HighClosed != false {
		t.Error("expected highClosed=false")
	}
}

func TestTranslate_TypeSpecifiers(t *testing.T) {
	tests := []struct {
		input    ast.TypeSpecifier
		expected string
	}{
		{&ast.NamedType{Namespace: "FHIR", Name: "Patient"}, "NamedTypeSpecifier"},
		{&ast.ListType{ElementType: &ast.NamedType{Name: "String"}}, "ListTypeSpecifier"},
		{&ast.IntervalType{PointType: &ast.NamedType{Name: "Integer"}}, "IntervalTypeSpecifier"},
	}
	for _, tt := range tests {
		ts := translateTypeSpecifier(tt.input)
		if ts.Type != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, ts.Type)
		}
	}
}

func TestTranslate_Query(t *testing.T) {
	node := TranslateExpression(&ast.Query{
		Sources: []*ast.AliasedSource{
			{
				Source: &ast.Retrieve{ResourceType: &ast.NamedType{Name: "Condition"}},
				Alias:  "C",
			},
		},
		Where: &ast.BinaryExpression{
			Operator: ast.OpEqual,
			Left:     &ast.MemberAccess{Source: &ast.IdentifierRef{Name: "C"}, Member: "clinicalStatus"},
			Right:    &ast.Literal{ValueType: ast.LiteralString, Value: "active"},
		},
	})
	if node.Type != "Query" {
		t.Errorf("expected Query, got %s", node.Type)
	}
	if len(node.SourceClause) != 1 {
		t.Errorf("expected 1 source, got %d", len(node.SourceClause))
	}
	if node.Where == nil || node.Where.Type != "Equal" {
		t.Error("expected Equal where clause")
	}
}

func TestTranslate_Definitions(t *testing.T) {
	lib := &ast.Library{
		CodeSystems: []*ast.CodeSystemDef{
			{Name: "LOINC", ID: "http://loinc.org", Version: "2.72"},
		},
		ValueSets: []*ast.ValueSetDef{
			{Name: "Diabetes VS", ID: "http://example.org/vs/diabetes"},
		},
		Codes: []*ast.CodeDef{
			{Name: "A1C", ID: "4548-4", System: "LOINC", Display: "Hemoglobin A1c"},
		},
		Parameters: []*ast.ParameterDef{
			{Name: "MeasurementPeriod", Type: &ast.IntervalType{PointType: &ast.NamedType{Name: "DateTime"}}},
		},
	}

	elm := Translate(lib)
	if len(elm.CodeSystems.Def) != 1 || elm.CodeSystems.Def[0].Name != "LOINC" {
		t.Error("expected LOINC code system")
	}
	if len(elm.ValueSets.Def) != 1 || elm.ValueSets.Def[0].Name != "Diabetes VS" {
		t.Error("expected Diabetes VS")
	}
	if len(elm.Codes.Def) != 1 || elm.Codes.Def[0].ID != "4548-4" {
		t.Error("expected A1C code")
	}
	if len(elm.Parameters.Def) != 1 {
		t.Error("expected 1 parameter")
	}
	if elm.Parameters.Def[0].ParameterType.Type != "IntervalTypeSpecifier" {
		t.Error("expected IntervalTypeSpecifier")
	}
}

// ---------------------------------------------------------------------------
// Import (ELM → AST)
// ---------------------------------------------------------------------------

func TestImport_SimpleLibrary(t *testing.T) {
	elmLib := &Library{
		Identifier: &VersionedIdentifier{ID: "TestLib", Version: "1.0.0"},
		Usings: &UsingDefs{Def: []*UsingDef{
			{LocalIdentifier: "FHIR", Version: "4.0.1"},
		}},
		Contexts: &ContextDefs{Def: []*ContextDef{
			{Name: "Patient"},
		}},
		Statements: &Statements{Def: []*ExpressionDef{
			{
				Name:    "IsAdult",
				Context: "Patient",
				Expression: &ExpressionNode{
					Type: "GreaterOrEqual",
					Operand: []*ExpressionNode{
						{Type: "FunctionRef", Name: "AgeInYears"},
						{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Integer", Value: "18"},
					},
				},
			},
		}},
	}

	lib, err := Import(elmLib)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if lib.Identifier.Name != "TestLib" {
		t.Errorf("expected TestLib, got %s", lib.Identifier.Name)
	}
	if len(lib.Usings) != 1 || lib.Usings[0].Name != "FHIR" {
		t.Error("expected FHIR using")
	}
	if len(lib.Contexts) != 1 || lib.Contexts[0].Name != "Patient" {
		t.Error("expected Patient context")
	}
	if len(lib.Statements) != 1 {
		t.Fatal("expected 1 statement")
	}

	expr := lib.Statements[0].Expression
	binExpr, ok := expr.(*ast.BinaryExpression)
	if !ok {
		t.Fatalf("expected BinaryExpression, got %T", expr)
	}
	if binExpr.Operator != ast.OpGreaterOrEqual {
		t.Errorf("expected GreaterOrEqual, got %d", binExpr.Operator)
	}
}

func TestImport_NilLibrary(t *testing.T) {
	_, err := Import(nil)
	if err == nil {
		t.Error("expected error for nil library")
	}
}

func TestImport_Retrieve(t *testing.T) {
	node := ImportExpression(&ExpressionNode{
		Type:         "Retrieve",
		DataType:     "{http://hl7.org/fhir}Condition",
		CodeProperty: "code",
		Codes:        &ExpressionNode{Type: "ExpressionRef", Name: "Diabetes"},
	})

	r, ok := node.(*ast.Retrieve)
	if !ok {
		t.Fatalf("expected Retrieve, got %T", node)
	}
	if r.ResourceType.Namespace != "FHIR" || r.ResourceType.Name != "Condition" {
		t.Errorf("expected FHIR.Condition, got %s.%s", r.ResourceType.Namespace, r.ResourceType.Name)
	}
	if r.CodePath != "code" {
		t.Errorf("expected code, got %s", r.CodePath)
	}
}

func TestImport_BinaryOps(t *testing.T) {
	tests := map[string]ast.BinaryOp{
		"Add":            ast.OpAdd,
		"Equal":          ast.OpEqual,
		"And":            ast.OpAnd,
		"Or":             ast.OpOr,
		"Union":          ast.OpUnion,
		"GreaterOrEqual": ast.OpGreaterOrEqual,
	}

	for typeName, expected := range tests {
		node := ImportExpression(&ExpressionNode{
			Type: typeName,
			Operand: []*ExpressionNode{
				{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Integer", Value: "1"},
				{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Integer", Value: "2"},
			},
		})
		binExpr, ok := node.(*ast.BinaryExpression)
		if !ok {
			t.Fatalf("%s: expected BinaryExpression, got %T", typeName, node)
		}
		if binExpr.Operator != expected {
			t.Errorf("%s: expected op %d, got %d", typeName, expected, binExpr.Operator)
		}
	}
}

func TestImport_Interval(t *testing.T) {
	lc := true
	hc := false
	node := ImportExpression(&ExpressionNode{
		Type:       "Interval",
		Low:        &ExpressionNode{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Integer", Value: "1"},
		High:       &ExpressionNode{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Integer", Value: "10"},
		LowClosed:  &lc,
		HighClosed: &hc,
	})

	ie, ok := node.(*ast.IntervalExpression)
	if !ok {
		t.Fatalf("expected IntervalExpression, got %T", node)
	}
	if !ie.LowClosed {
		t.Error("expected lowClosed=true")
	}
	if ie.HighClosed {
		t.Error("expected highClosed=false")
	}
}

// ---------------------------------------------------------------------------
// JSON marshaling
// ---------------------------------------------------------------------------

func TestJSON_MarshalUnmarshal_SingleOperand(t *testing.T) {
	node := &ExpressionNode{
		Type:    "Not",
		Operand: &ExpressionNode{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Boolean", Value: "true"},
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ExpressionNode
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Type != "Not" {
		t.Errorf("expected Not, got %s", decoded.Type)
	}

	op, ok := decoded.Operand.(*ExpressionNode)
	if !ok {
		t.Fatalf("expected single *ExpressionNode operand, got %T", decoded.Operand)
	}
	if op.Type != "Literal" || op.Value != "true" {
		t.Errorf("expected Literal true, got %s %s", op.Type, op.Value)
	}
}

func TestJSON_MarshalUnmarshal_MultiOperand(t *testing.T) {
	node := &ExpressionNode{
		Type: "Add",
		Operand: []*ExpressionNode{
			{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Integer", Value: "1"},
			{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Integer", Value: "2"},
		},
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ExpressionNode
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Type != "Add" {
		t.Errorf("expected Add, got %s", decoded.Type)
	}

	ops, ok := decoded.Operand.([]*ExpressionNode)
	if !ok {
		t.Fatalf("expected []*ExpressionNode, got %T", decoded.Operand)
	}
	if len(ops) != 2 {
		t.Errorf("expected 2 operands, got %d", len(ops))
	}
	if ops[0].Value != "1" || ops[1].Value != "2" {
		t.Error("operand values mismatch")
	}
}

func TestJSON_MarshalLibrary(t *testing.T) {
	lib := &Library{
		Identifier: &VersionedIdentifier{ID: "Test", Version: "1.0"},
		Statements: &Statements{Def: []*ExpressionDef{
			{
				Name:       "Simple",
				Expression: &ExpressionNode{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Integer", Value: "42"},
			},
		}},
	}

	data, err := MarshalLibrary(lib)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	decoded, err := UnmarshalLibrary(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Identifier.ID != "Test" {
		t.Errorf("expected Test, got %s", decoded.Identifier.ID)
	}
	if len(decoded.Statements.Def) != 1 {
		t.Fatal("expected 1 statement")
	}
	if decoded.Statements.Def[0].Expression.Value != "42" {
		t.Errorf("expected 42, got %s", decoded.Statements.Def[0].Expression.Value)
	}
}

// ---------------------------------------------------------------------------
// Roundtrip (AST → ELM → JSON → ELM → AST)
// ---------------------------------------------------------------------------

func TestRoundtrip_SimpleExpression(t *testing.T) {
	// Build AST
	original := &ast.Library{
		Identifier: &ast.LibraryIdentifier{Name: "RoundtripTest", Version: "1.0"},
		Usings:     []*ast.UsingDef{{Name: "FHIR", Version: "4.0.1"}},
		Contexts:   []*ast.ContextDef{{Name: "Patient"}},
		Statements: []*ast.ExpressionDef{
			{
				Name:    "IsAdult",
				Context: "Patient",
				Expression: &ast.BinaryExpression{
					Operator: ast.OpGreaterOrEqual,
					Left:     &ast.FunctionCall{Name: "AgeInYears"},
					Right:    &ast.Literal{ValueType: ast.LiteralInteger, Value: "18"},
				},
			},
		},
	}

	// AST → ELM
	elmLib := Translate(original)

	// ELM → JSON
	jsonData, err := MarshalLibrary(elmLib)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// JSON → ELM
	elmLib2, err := UnmarshalLibrary(jsonData)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// ELM → AST
	roundtripped, err := Import(elmLib2)
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	// Verify
	if roundtripped.Identifier.Name != "RoundtripTest" {
		t.Errorf("expected RoundtripTest, got %s", roundtripped.Identifier.Name)
	}
	if len(roundtripped.Statements) != 1 {
		t.Fatal("expected 1 statement")
	}

	expr := roundtripped.Statements[0].Expression
	binExpr, ok := expr.(*ast.BinaryExpression)
	if !ok {
		t.Fatalf("expected BinaryExpression, got %T", expr)
	}
	if binExpr.Operator != ast.OpGreaterOrEqual {
		t.Errorf("expected GreaterOrEqual, got %d", binExpr.Operator)
	}

	// Right should be literal 18
	lit, ok := binExpr.Right.(*ast.Literal)
	if !ok {
		t.Fatalf("expected Literal, got %T", binExpr.Right)
	}
	if lit.Value != "18" || lit.ValueType != ast.LiteralInteger {
		t.Errorf("expected integer 18, got %s %d", lit.Value, lit.ValueType)
	}
}

func TestRoundtrip_RetrieveWithQuery(t *testing.T) {
	original := &ast.Library{
		Statements: []*ast.ExpressionDef{
			{
				Name: "ActiveConditions",
				Expression: &ast.Query{
					Sources: []*ast.AliasedSource{
						{
							Source: &ast.Retrieve{
								ResourceType: &ast.NamedType{Namespace: "FHIR", Name: "Condition"},
								CodePath:     "code",
							},
							Alias: "C",
						},
					},
					Where: &ast.BinaryExpression{
						Operator: ast.OpEqual,
						Left: &ast.MemberAccess{
							Source: &ast.IdentifierRef{Name: "C"},
							Member: "clinicalStatus",
						},
						Right: &ast.Literal{ValueType: ast.LiteralString, Value: "active"},
					},
				},
			},
		},
	}

	// Full roundtrip
	elmLib := Translate(original)
	jsonData, err := MarshalLibrary(elmLib)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	elmLib2, err := UnmarshalLibrary(jsonData)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	roundtripped, err := Import(elmLib2)
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	if len(roundtripped.Statements) != 1 {
		t.Fatal("expected 1 statement")
	}

	query, ok := roundtripped.Statements[0].Expression.(*ast.Query)
	if !ok {
		t.Fatalf("expected Query, got %T", roundtripped.Statements[0].Expression)
	}
	if len(query.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(query.Sources))
	}
	if query.Sources[0].Alias != "C" {
		t.Errorf("expected alias C, got %s", query.Sources[0].Alias)
	}

	retrieve, ok := query.Sources[0].Source.(*ast.Retrieve)
	if !ok {
		t.Fatalf("expected Retrieve, got %T", query.Sources[0].Source)
	}
	if retrieve.ResourceType.Name != "Condition" {
		t.Errorf("expected Condition, got %s", retrieve.ResourceType.Name)
	}
}

func TestTranslate_RetrieveWithDateRange(t *testing.T) {
	lib := &ast.Library{
		Statements: []*ast.ExpressionDef{
			{
				Name: "RecentConditions",
				Expression: &ast.Retrieve{
					ResourceType: &ast.NamedType{Namespace: "FHIR", Name: "Condition"},
					CodePath:     "code",
					DatePath:     "onset",
					DateRange: &ast.IntervalExpression{
						LowClosed:  true,
						HighClosed: true,
						Low:        &ast.Literal{ValueType: ast.LiteralDate, Value: "2024-01-01"},
						High:       &ast.Literal{ValueType: ast.LiteralDate, Value: "2024-12-31"},
					},
				},
			},
		},
	}

	elm := Translate(lib)
	expr := elm.Statements.Def[0].Expression
	if expr.Type != "Retrieve" {
		t.Errorf("expected Retrieve, got %s", expr.Type)
	}
	if expr.DateProperty != "onset" {
		t.Errorf("expected dateProperty=onset, got %s", expr.DateProperty)
	}
	if expr.DateRange == nil {
		t.Fatal("expected non-nil dateRange")
	}
	if expr.DateRange.Type != "Interval" {
		t.Errorf("expected dateRange type=Interval, got %s", expr.DateRange.Type)
	}
}

func TestImport_RetrieveWithDateRange(t *testing.T) {
	node := ImportExpression(&ExpressionNode{
		Type:         "Retrieve",
		DataType:     "{http://hl7.org/fhir}Condition",
		DateProperty: "onset",
		DateRange: &ExpressionNode{
			Type: "Interval",
			Low:  &ExpressionNode{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Date", Value: "2024-01-01"},
			High: &ExpressionNode{Type: "Literal", ValueType: "{urn:hl7-org:elm-types:r1}Date", Value: "2024-12-31"},
		},
	})

	r, ok := node.(*ast.Retrieve)
	if !ok {
		t.Fatalf("expected Retrieve, got %T", node)
	}
	if r.DatePath != "onset" {
		t.Errorf("expected DatePath=onset, got %s", r.DatePath)
	}
	if r.DateRange == nil {
		t.Fatal("expected non-nil DateRange")
	}
	interval, ok := r.DateRange.(*ast.IntervalExpression)
	if !ok {
		t.Fatalf("expected IntervalExpression, got %T", r.DateRange)
	}
	if interval.Low == nil || interval.High == nil {
		t.Error("expected Low and High in DateRange interval")
	}
}

func TestRoundtrip_RetrieveWithDateRange(t *testing.T) {
	original := &ast.Library{
		Statements: []*ast.ExpressionDef{
			{
				Name: "FilteredRetrieve",
				Expression: &ast.Retrieve{
					ResourceType: &ast.NamedType{Namespace: "FHIR", Name: "Encounter"},
					DatePath:     "period",
					DateRange: &ast.IntervalExpression{
						LowClosed:  true,
						HighClosed: false,
						Low:        &ast.Literal{ValueType: ast.LiteralDate, Value: "2024-01-01"},
						High:       &ast.Literal{ValueType: ast.LiteralDate, Value: "2025-01-01"},
					},
				},
			},
		},
	}

	elmLib := Translate(original)
	jsonData, err := MarshalLibrary(elmLib)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	elmLib2, err := UnmarshalLibrary(jsonData)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	roundtripped, err := Import(elmLib2)
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	r, ok := roundtripped.Statements[0].Expression.(*ast.Retrieve)
	if !ok {
		t.Fatalf("expected Retrieve, got %T", roundtripped.Statements[0].Expression)
	}
	if r.DatePath != "period" {
		t.Errorf("expected DatePath=period, got %s", r.DatePath)
	}
	if r.DateRange == nil {
		t.Fatal("expected non-nil DateRange after roundtrip")
	}
	ie, ok := r.DateRange.(*ast.IntervalExpression)
	if !ok {
		t.Fatalf("expected IntervalExpression, got %T", r.DateRange)
	}
	if !ie.LowClosed || ie.HighClosed {
		t.Error("interval boundary mismatch after roundtrip")
	}
}

func TestRoundtrip_IntervalAndList(t *testing.T) {
	original := &ast.Library{
		Statements: []*ast.ExpressionDef{
			{
				Name: "TestInterval",
				Expression: &ast.IntervalExpression{
					LowClosed:  true,
					HighClosed: false,
					Low:        &ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
					High:       &ast.Literal{ValueType: ast.LiteralInteger, Value: "10"},
				},
			},
			{
				Name: "TestList",
				Expression: &ast.ListExpression{
					Elements: []ast.Expression{
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "1"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "2"},
						&ast.Literal{ValueType: ast.LiteralInteger, Value: "3"},
					},
				},
			},
		},
	}

	elmLib := Translate(original)
	jsonData, err := MarshalLibrary(elmLib)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	elmLib2, err := UnmarshalLibrary(jsonData)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	roundtripped, err := Import(elmLib2)
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	// Check interval
	ie, ok := roundtripped.Statements[0].Expression.(*ast.IntervalExpression)
	if !ok {
		t.Fatalf("expected IntervalExpression, got %T", roundtripped.Statements[0].Expression)
	}
	if !ie.LowClosed || ie.HighClosed {
		t.Error("interval boundary mismatch")
	}

	// Check list
	le, ok := roundtripped.Statements[1].Expression.(*ast.ListExpression)
	if !ok {
		t.Fatalf("expected ListExpression, got %T", roundtripped.Statements[1].Expression)
	}
	if len(le.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(le.Elements))
	}
}
