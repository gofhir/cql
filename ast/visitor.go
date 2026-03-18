package ast

// Visitor defines the interface for traversing CQL AST nodes.
// Implementations return a value and an error to support evaluation.
type Visitor interface {
	// Top-level
	VisitLibrary(node *Library) (interface{}, error)

	// Definitions
	VisitExpressionDef(node *ExpressionDef) (interface{}, error)
	VisitFunctionDef(node *FunctionDef) (interface{}, error)

	// Expressions
	VisitLiteral(node *Literal) (interface{}, error)
	VisitIdentifierRef(node *IdentifierRef) (interface{}, error)
	VisitRetrieve(node *Retrieve) (interface{}, error)
	VisitQuery(node *Query) (interface{}, error)
	VisitBinaryExpression(node *BinaryExpression) (interface{}, error)
	VisitUnaryExpression(node *UnaryExpression) (interface{}, error)
	VisitIsExpression(node *IsExpression) (interface{}, error)
	VisitAsExpression(node *AsExpression) (interface{}, error)
	VisitCastExpression(node *CastExpression) (interface{}, error)
	VisitConvertExpression(node *ConvertExpression) (interface{}, error)
	VisitBooleanTestExpression(node *BooleanTestExpression) (interface{}, error)
	VisitIfThenElse(node *IfThenElse) (interface{}, error)
	VisitCaseExpression(node *CaseExpression) (interface{}, error)
	VisitBetweenExpression(node *BetweenExpression) (interface{}, error)
	VisitDurationBetween(node *DurationBetween) (interface{}, error)
	VisitDifferenceBetween(node *DifferenceBetween) (interface{}, error)
	VisitDateTimeComponentFrom(node *DateTimeComponentFrom) (interface{}, error)
	VisitDurationOf(node *DurationOf) (interface{}, error)
	VisitDifferenceOf(node *DifferenceOf) (interface{}, error)
	VisitTypeExtent(node *TypeExtent) (interface{}, error)
	VisitTimingExpression(node *TimingExpression) (interface{}, error)
	VisitMembershipExpression(node *MembershipExpression) (interface{}, error)
	VisitMemberAccess(node *MemberAccess) (interface{}, error)
	VisitFunctionCall(node *FunctionCall) (interface{}, error)
	VisitIndexAccess(node *IndexAccess) (interface{}, error)
	VisitIntervalExpression(node *IntervalExpression) (interface{}, error)
	VisitTupleExpression(node *TupleExpression) (interface{}, error)
	VisitInstanceExpression(node *InstanceExpression) (interface{}, error)
	VisitListExpression(node *ListExpression) (interface{}, error)
	VisitCodeExpression(node *CodeExpression) (interface{}, error)
	VisitConceptExpression(node *ConceptExpression) (interface{}, error)
	VisitExternalConstant(node *ExternalConstant) (interface{}, error)
	VisitThisExpression(node *ThisExpression) (interface{}, error)
	VisitIndexExpression(node *IndexExpression) (interface{}, error)
	VisitTotalExpression(node *TotalExpression) (interface{}, error)
	VisitSetAggregateExpression(node *SetAggregateExpression) (interface{}, error)
}
