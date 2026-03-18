// Code generated from cql.g4 by ANTLR 4.13.2. DO NOT EDIT.

package grammar // cql
import "github.com/antlr4-go/antlr/v4"

type BasecqlVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BasecqlVisitor) VisitDefinition(ctx *DefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitLibrary(ctx *LibraryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitLibraryDefinition(ctx *LibraryDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitUsingDefinition(ctx *UsingDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitIncludeDefinition(ctx *IncludeDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitLocalIdentifier(ctx *LocalIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitAccessModifier(ctx *AccessModifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitParameterDefinition(ctx *ParameterDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitCodesystemDefinition(ctx *CodesystemDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitValuesetDefinition(ctx *ValuesetDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitCodesystems(ctx *CodesystemsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitCodesystemIdentifier(ctx *CodesystemIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitLibraryIdentifier(ctx *LibraryIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitCodeDefinition(ctx *CodeDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitConceptDefinition(ctx *ConceptDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitCodeIdentifier(ctx *CodeIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitCodesystemId(ctx *CodesystemIdContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitValuesetId(ctx *ValuesetIdContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitVersionSpecifier(ctx *VersionSpecifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitCodeId(ctx *CodeIdContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTypeSpecifier(ctx *TypeSpecifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitNamedTypeSpecifier(ctx *NamedTypeSpecifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitModelIdentifier(ctx *ModelIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitListTypeSpecifier(ctx *ListTypeSpecifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitIntervalTypeSpecifier(ctx *IntervalTypeSpecifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTupleTypeSpecifier(ctx *TupleTypeSpecifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTupleElementDefinition(ctx *TupleElementDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitChoiceTypeSpecifier(ctx *ChoiceTypeSpecifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitStatement(ctx *StatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitExpressionDefinition(ctx *ExpressionDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitContextDefinition(ctx *ContextDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitFluentModifier(ctx *FluentModifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitFunctionDefinition(ctx *FunctionDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitOperandDefinition(ctx *OperandDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitFunctionBody(ctx *FunctionBodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitQuerySource(ctx *QuerySourceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitAliasedQuerySource(ctx *AliasedQuerySourceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitAlias(ctx *AliasContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitQueryInclusionClause(ctx *QueryInclusionClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitWithClause(ctx *WithClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitWithoutClause(ctx *WithoutClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitRetrieve(ctx *RetrieveContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitContextIdentifier(ctx *ContextIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitCodePath(ctx *CodePathContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitCodeComparator(ctx *CodeComparatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTerminology(ctx *TerminologyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitQualifier(ctx *QualifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitQuery(ctx *QueryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitSourceClause(ctx *SourceClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitLetClause(ctx *LetClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitLetClauseItem(ctx *LetClauseItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitWhereClause(ctx *WhereClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitReturnClause(ctx *ReturnClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitAggregateClause(ctx *AggregateClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitStartingClause(ctx *StartingClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitSortClause(ctx *SortClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitSortDirection(ctx *SortDirectionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitSortByItem(ctx *SortByItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitQualifiedIdentifier(ctx *QualifiedIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitQualifiedIdentifierExpression(ctx *QualifiedIdentifierExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitQualifierExpression(ctx *QualifierExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitSimplePathIndexer(ctx *SimplePathIndexerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitSimplePathQualifiedIdentifier(ctx *SimplePathQualifiedIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitSimplePathReferentialIdentifier(ctx *SimplePathReferentialIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitSimpleStringLiteral(ctx *SimpleStringLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitSimpleNumberLiteral(ctx *SimpleNumberLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitDurationBetweenExpression(ctx *DurationBetweenExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitInFixSetExpression(ctx *InFixSetExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitRetrieveExpression(ctx *RetrieveExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTimingExpression(ctx *TimingExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitQueryExpression(ctx *QueryExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitNotExpression(ctx *NotExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitBooleanExpression(ctx *BooleanExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitOrExpression(ctx *OrExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitCastExpression(ctx *CastExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitAndExpression(ctx *AndExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitBetweenExpression(ctx *BetweenExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitMembershipExpression(ctx *MembershipExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitDifferenceBetweenExpression(ctx *DifferenceBetweenExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitInequalityExpression(ctx *InequalityExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitEqualityExpression(ctx *EqualityExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitExistenceExpression(ctx *ExistenceExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitImpliesExpression(ctx *ImpliesExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTermExpression(ctx *TermExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTypeExpression(ctx *TypeExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitDateTimePrecision(ctx *DateTimePrecisionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitDateTimeComponent(ctx *DateTimeComponentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitPluralDateTimePrecision(ctx *PluralDateTimePrecisionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitAdditionExpressionTerm(ctx *AdditionExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitIndexedExpressionTerm(ctx *IndexedExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitWidthExpressionTerm(ctx *WidthExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitSetAggregateExpressionTerm(ctx *SetAggregateExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTimeUnitExpressionTerm(ctx *TimeUnitExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitIfThenElseExpressionTerm(ctx *IfThenElseExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTimeBoundaryExpressionTerm(ctx *TimeBoundaryExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitElementExtractorExpressionTerm(ctx *ElementExtractorExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitConversionExpressionTerm(ctx *ConversionExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTypeExtentExpressionTerm(ctx *TypeExtentExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitPredecessorExpressionTerm(ctx *PredecessorExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitPointExtractorExpressionTerm(ctx *PointExtractorExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitMultiplicationExpressionTerm(ctx *MultiplicationExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitAggregateExpressionTerm(ctx *AggregateExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitDurationExpressionTerm(ctx *DurationExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitDifferenceExpressionTerm(ctx *DifferenceExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitCaseExpressionTerm(ctx *CaseExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitPowerExpressionTerm(ctx *PowerExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitSuccessorExpressionTerm(ctx *SuccessorExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitPolarityExpressionTerm(ctx *PolarityExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTermExpressionTerm(ctx *TermExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitInvocationExpressionTerm(ctx *InvocationExpressionTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitCaseExpressionItem(ctx *CaseExpressionItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitDateTimePrecisionSpecifier(ctx *DateTimePrecisionSpecifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitRelativeQualifier(ctx *RelativeQualifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitOffsetRelativeQualifier(ctx *OffsetRelativeQualifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitExclusiveRelativeQualifier(ctx *ExclusiveRelativeQualifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitQuantityOffset(ctx *QuantityOffsetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTemporalRelationship(ctx *TemporalRelationshipContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitConcurrentWithIntervalOperatorPhrase(ctx *ConcurrentWithIntervalOperatorPhraseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitIncludesIntervalOperatorPhrase(ctx *IncludesIntervalOperatorPhraseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitIncludedInIntervalOperatorPhrase(ctx *IncludedInIntervalOperatorPhraseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitBeforeOrAfterIntervalOperatorPhrase(ctx *BeforeOrAfterIntervalOperatorPhraseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitWithinIntervalOperatorPhrase(ctx *WithinIntervalOperatorPhraseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitMeetsIntervalOperatorPhrase(ctx *MeetsIntervalOperatorPhraseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitOverlapsIntervalOperatorPhrase(ctx *OverlapsIntervalOperatorPhraseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitStartsIntervalOperatorPhrase(ctx *StartsIntervalOperatorPhraseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitEndsIntervalOperatorPhrase(ctx *EndsIntervalOperatorPhraseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitInvocationTerm(ctx *InvocationTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitLiteralTerm(ctx *LiteralTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitExternalConstantTerm(ctx *ExternalConstantTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitIntervalSelectorTerm(ctx *IntervalSelectorTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTupleSelectorTerm(ctx *TupleSelectorTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitInstanceSelectorTerm(ctx *InstanceSelectorTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitListSelectorTerm(ctx *ListSelectorTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitCodeSelectorTerm(ctx *CodeSelectorTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitConceptSelectorTerm(ctx *ConceptSelectorTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitParenthesizedTerm(ctx *ParenthesizedTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitQualifiedMemberInvocation(ctx *QualifiedMemberInvocationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitQualifiedFunctionInvocation(ctx *QualifiedFunctionInvocationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitQualifiedFunction(ctx *QualifiedFunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitMemberInvocation(ctx *MemberInvocationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitFunctionInvocation(ctx *FunctionInvocationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitThisInvocation(ctx *ThisInvocationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitIndexInvocation(ctx *IndexInvocationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTotalInvocation(ctx *TotalInvocationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitFunction(ctx *FunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitRatio(ctx *RatioContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitBooleanLiteral(ctx *BooleanLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitNullLiteral(ctx *NullLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitStringLiteral(ctx *StringLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitNumberLiteral(ctx *NumberLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitLongNumberLiteral(ctx *LongNumberLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitDateTimeLiteral(ctx *DateTimeLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitDateLiteral(ctx *DateLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTimeLiteral(ctx *TimeLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitQuantityLiteral(ctx *QuantityLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitRatioLiteral(ctx *RatioLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitExternalConstant(ctx *ExternalConstantContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitIntervalSelector(ctx *IntervalSelectorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTupleSelector(ctx *TupleSelectorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTupleElementSelector(ctx *TupleElementSelectorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitInstanceSelector(ctx *InstanceSelectorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitInstanceElementSelector(ctx *InstanceElementSelectorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitListSelector(ctx *ListSelectorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitDisplayClause(ctx *DisplayClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitCodeSelector(ctx *CodeSelectorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitConceptSelector(ctx *ConceptSelectorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitKeyword(ctx *KeywordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitReservedWord(ctx *ReservedWordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitKeywordIdentifier(ctx *KeywordIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitObsoleteIdentifier(ctx *ObsoleteIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitFunctionIdentifier(ctx *FunctionIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitTypeNameIdentifier(ctx *TypeNameIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitReferentialIdentifier(ctx *ReferentialIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitReferentialOrTypeNameIdentifier(ctx *ReferentialOrTypeNameIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitIdentifierOrFunctionIdentifier(ctx *IdentifierOrFunctionIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitIdentifier(ctx *IdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitParamList(ctx *ParamListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitQuantity(ctx *QuantityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BasecqlVisitor) VisitUnit(ctx *UnitContext) interface{} {
	return v.VisitChildren(ctx)
}
