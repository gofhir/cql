package cql

import (
	"context"
	"testing"
)

// TestAFHIRNumberReachesTheListsCommonType covers a model conversion followed
// by a numeric promotion. FHIR.integer reaches a Decimal through
// FHIRHelpers.ToInteger and then ToDecimal, as the reference translator writes
// it; applying only the function left the element an Integer.
func TestAFHIRNumberReachesTheListsCommonType(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1' called FH\n" +
		"define R: First({FHIR.integer { value: 1 }, 2.5}) is Decimal\n"
	v, err := NewEngine().EvaluateExpression(context.Background(), src, "R", nil, nil)
	if err != nil || valueString(v) != "true" {
		t.Errorf("First({FHIR.integer { value: 1 }, 2.5}) is Decimal = %v, %v; want true", v, err)
	}
}

// TestPromotingLeavesValuesGoCannotCompareAlone covers the values a promotion
// walks past. Whether a list changed was asked by comparing each element with
// what came back, which panics on a value Go cannot compare: a list, or a
// Concept, which holds one. A List<FHIR.CodeableConcept> converted to
// List<Concept> crashed the evaluation, and so did a promotion three lists
// deep.
func TestPromotingLeavesValuesGoCannotCompareAlone(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ninclude FHIRHelpers version '4.0.1' called FH\n" +
		"define function F(x List<Concept>): Count(x)\n" +
		"define Concepts: F({FHIR.CodeableConcept{text: FHIR.string{value: 'x'}}})\n" +
		"define Deep: First(First(First({{{1}}, {{2.5}}}))) is Decimal\n" +
		"define DeepIf: First(First(First(if true then {{{1}}} else {{{2.5}}}))) is Decimal\n" +
		"define DeepCoalesce: First(First(Coalesce({{1}}, {{2.5}}))) is Decimal\n"
	for def, want := range map[string]string{"Concepts": "1", "Deep": "true", "DeepIf": "true", "DeepCoalesce": "true"} {
		v, err := NewEngine().EvaluateExpression(context.Background(), src, def, nil, nil)
		if err != nil || valueString(v) != want {
			t.Errorf("%s = %v, %v; want %s", def, v, err, want)
		}
	}
}
