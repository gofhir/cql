package elm

import (
	"encoding/json"
	"fmt"
)

// MarshalLibrary serializes an ELM Library to JSON.
func MarshalLibrary(lib *Library) ([]byte, error) {
	return json.MarshalIndent(lib, "", "  ")
}

// UnmarshalLibrary deserializes an ELM Library from JSON.
func UnmarshalLibrary(data []byte) (*Library, error) {
	var lib Library
	if err := json.Unmarshal(data, &lib); err != nil {
		return nil, fmt.Errorf("elm: unmarshal library: %w", err)
	}
	return &lib, nil
}

// expressionNodeJSON is the raw JSON representation used for marshaling/unmarshaling.
// It mirrors ExpressionNode but uses json.RawMessage for the polymorphic Operand field.
type expressionNodeJSON struct {
	Type              string             `json:"type"`
	ResultTypeName    string             `json:"resultTypeName,omitempty"`
	ResultTypeSpecifier *TypeSpecifier   `json:"resultTypeSpecifier,omitempty"`
	ValueType         string             `json:"valueType,omitempty"`
	Value             string             `json:"value,omitempty"`
	Name              string             `json:"name,omitempty"`
	LibraryName       string             `json:"libraryName,omitempty"`
	Path              string             `json:"path,omitempty"`
	Source            *ExpressionNode    `json:"source,omitempty"`
	Scope             string             `json:"scope,omitempty"`
	DataType          string             `json:"dataType,omitempty"`
	TemplateID        string             `json:"templateId,omitempty"`
	CodeProperty      string             `json:"codeProperty,omitempty"`
	CodeComparator    string             `json:"codeComparator,omitempty"`
	Codes             *ExpressionNode    `json:"codes,omitempty"`
	DateProperty      string             `json:"dateProperty,omitempty"`
	DateRange         *ExpressionNode    `json:"dateRange,omitempty"`
	Operand           json.RawMessage    `json:"operand,omitempty"`
	Condition         *ExpressionNode    `json:"condition,omitempty"`
	Then              *ExpressionNode    `json:"then,omitempty"`
	Else              *ExpressionNode    `json:"else,omitempty"`
	Comparand         *ExpressionNode    `json:"comparand,omitempty"`
	CaseItem          []*CaseItem        `json:"caseItem,omitempty"`
	SourceClause      []*AliasedQuerySource `json:"sourceClause,omitempty"`
	Let               []*LetClause       `json:"let,omitempty"`
	Relationship      []*RelationshipClause `json:"relationship,omitempty"`
	Where             *ExpressionNode    `json:"where,omitempty"`
	Return            *ReturnClause      `json:"return,omitempty"`
	Sort              *SortClause        `json:"sort,omitempty"`
	Aggregate         *AggregateClause   `json:"aggregate,omitempty"`
	Low               *ExpressionNode    `json:"low,omitempty"`
	High              *ExpressionNode    `json:"high,omitempty"`
	LowClosed         *bool              `json:"lowClosed,omitempty"`
	HighClosed        *bool              `json:"highClosed,omitempty"`
	Element           []*ExpressionNode  `json:"element,omitempty"`
	ClassType         string             `json:"classType,omitempty"`
	FunctionName      string             `json:"functionName,omitempty"`
	CodeValue         string             `json:"code,omitempty"`
	System            *ExpressionNode    `json:"system,omitempty"`
	Display           string             `json:"display,omitempty"`
	IsTypeSpecifier   *TypeSpecifier     `json:"isTypeSpecifier,omitempty"`
	AsTypeSpecifier   *TypeSpecifier     `json:"asTypeSpecifier,omitempty"`
	Strict            bool               `json:"strict,omitempty"`
	ToType            *TypeSpecifier     `json:"toTypeSpecifier,omitempty"`
	Precision         string             `json:"precision,omitempty"`
	Per               *ExpressionNode    `json:"per,omitempty"`
	TestValue         string             `json:"testValue,omitempty"`
	IsNot             bool               `json:"isNot,omitempty"`
	Extent            string             `json:"extent,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for ExpressionNode.
// The Operand field is polymorphic: it can be a single *ExpressionNode
// (for unary ops) or []*ExpressionNode (for binary/N-ary ops).
func (e *ExpressionNode) MarshalJSON() ([]byte, error) {
	raw := expressionNodeJSON{
		Type:              e.Type,
		ResultTypeName:    e.ResultTypeName,
		ResultTypeSpecifier: e.ResultTypeSpecifier,
		ValueType:         e.ValueType,
		Value:             e.Value,
		Name:              e.Name,
		LibraryName:       e.LibraryName,
		Path:              e.Path,
		Source:            e.Source,
		Scope:             e.Scope,
		DataType:          e.DataType,
		TemplateID:        e.TemplateID,
		CodeProperty:      e.CodeProperty,
		CodeComparator:    e.CodeComparator,
		Codes:             e.Codes,
		DateProperty:      e.DateProperty,
		DateRange:         e.DateRange,
		Condition:         e.Condition,
		Then:              e.Then,
		Else:              e.Else,
		Comparand:         e.Comparand,
		CaseItem:          e.CaseItem,
		SourceClause:      e.SourceClause,
		Let:               e.Let,
		Relationship:      e.Relationship,
		Where:             e.Where,
		Return:            e.Return,
		Sort:              e.Sort,
		Aggregate:         e.Aggregate,
		Low:               e.Low,
		High:              e.High,
		LowClosed:         e.LowClosed,
		HighClosed:        e.HighClosed,
		Element:           e.Element,
		ClassType:         e.ClassType,
		FunctionName:      e.FunctionName,
		CodeValue:         e.CodeValue,
		System:            e.System,
		Display:           e.Display,
		IsTypeSpecifier:   e.IsTypeSpecifier,
		AsTypeSpecifier:   e.AsTypeSpecifier,
		Strict:            e.Strict,
		ToType:            e.ToType,
		Precision:         e.Precision,
		Per:               e.Per,
		TestValue:         e.TestValue,
		IsNot:             e.IsNot,
		Extent:            e.Extent,
	}

	// Marshal the polymorphic Operand field
	if e.Operand != nil {
		switch v := e.Operand.(type) {
		case *ExpressionNode:
			data, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("elm: marshal single operand: %w", err)
			}
			raw.Operand = data
		case []*ExpressionNode:
			data, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("elm: marshal operand list: %w", err)
			}
			raw.Operand = data
		}
	}

	return json.Marshal(raw)
}

// UnmarshalJSON implements custom JSON unmarshaling for ExpressionNode.
// It detects whether the "operand" field is an object (single) or array (multi).
func (e *ExpressionNode) UnmarshalJSON(data []byte) error {
	var raw expressionNodeJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("elm: unmarshal expression: %w", err)
	}

	e.Type = raw.Type
	e.ResultTypeName = raw.ResultTypeName
	e.ResultTypeSpecifier = raw.ResultTypeSpecifier
	e.ValueType = raw.ValueType
	e.Value = raw.Value
	e.Name = raw.Name
	e.LibraryName = raw.LibraryName
	e.Path = raw.Path
	e.Source = raw.Source
	e.Scope = raw.Scope
	e.DataType = raw.DataType
	e.TemplateID = raw.TemplateID
	e.CodeProperty = raw.CodeProperty
	e.CodeComparator = raw.CodeComparator
	e.Codes = raw.Codes
	e.DateProperty = raw.DateProperty
	e.DateRange = raw.DateRange
	e.Condition = raw.Condition
	e.Then = raw.Then
	e.Else = raw.Else
	e.Comparand = raw.Comparand
	e.CaseItem = raw.CaseItem
	e.SourceClause = raw.SourceClause
	e.Let = raw.Let
	e.Relationship = raw.Relationship
	e.Where = raw.Where
	e.Return = raw.Return
	e.Sort = raw.Sort
	e.Aggregate = raw.Aggregate
	e.Low = raw.Low
	e.High = raw.High
	e.LowClosed = raw.LowClosed
	e.HighClosed = raw.HighClosed
	e.Element = raw.Element
	e.ClassType = raw.ClassType
	e.FunctionName = raw.FunctionName
	e.CodeValue = raw.CodeValue
	e.System = raw.System
	e.Display = raw.Display
	e.IsTypeSpecifier = raw.IsTypeSpecifier
	e.AsTypeSpecifier = raw.AsTypeSpecifier
	e.Strict = raw.Strict
	e.ToType = raw.ToType
	e.Precision = raw.Precision
	e.Per = raw.Per
	e.TestValue = raw.TestValue
	e.IsNot = raw.IsNot
	e.Extent = raw.Extent

	// Unmarshal polymorphic operand
	if len(raw.Operand) > 0 {
		// Detect if it's an array or object by the first byte
		trimmed := trimLeadingWhitespace(raw.Operand)
		if len(trimmed) > 0 && trimmed[0] == '[' {
			var ops []*ExpressionNode
			if err := json.Unmarshal(raw.Operand, &ops); err != nil {
				return fmt.Errorf("elm: unmarshal operand array: %w", err)
			}
			e.Operand = ops
		} else if len(trimmed) > 0 && trimmed[0] == '{' {
			var op ExpressionNode
			if err := json.Unmarshal(raw.Operand, &op); err != nil {
				return fmt.Errorf("elm: unmarshal single operand: %w", err)
			}
			e.Operand = &op
		}
	}

	return nil
}

// trimLeadingWhitespace returns a slice without leading JSON whitespace.
func trimLeadingWhitespace(data []byte) []byte {
	for i, b := range data {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			return data[i:]
		}
	}
	return nil
}
