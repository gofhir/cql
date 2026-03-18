// Code generated from cql.g4 by ANTLR 4.13.2. DO NOT EDIT.

package grammar // cql
import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by cqlParser.
type cqlVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by cqlParser#definition.
	VisitDefinition(ctx *DefinitionContext) interface{}

	// Visit a parse tree produced by cqlParser#library.
	VisitLibrary(ctx *LibraryContext) interface{}

	// Visit a parse tree produced by cqlParser#libraryDefinition.
	VisitLibraryDefinition(ctx *LibraryDefinitionContext) interface{}

	// Visit a parse tree produced by cqlParser#usingDefinition.
	VisitUsingDefinition(ctx *UsingDefinitionContext) interface{}

	// Visit a parse tree produced by cqlParser#includeDefinition.
	VisitIncludeDefinition(ctx *IncludeDefinitionContext) interface{}

	// Visit a parse tree produced by cqlParser#localIdentifier.
	VisitLocalIdentifier(ctx *LocalIdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#accessModifier.
	VisitAccessModifier(ctx *AccessModifierContext) interface{}

	// Visit a parse tree produced by cqlParser#parameterDefinition.
	VisitParameterDefinition(ctx *ParameterDefinitionContext) interface{}

	// Visit a parse tree produced by cqlParser#codesystemDefinition.
	VisitCodesystemDefinition(ctx *CodesystemDefinitionContext) interface{}

	// Visit a parse tree produced by cqlParser#valuesetDefinition.
	VisitValuesetDefinition(ctx *ValuesetDefinitionContext) interface{}

	// Visit a parse tree produced by cqlParser#codesystems.
	VisitCodesystems(ctx *CodesystemsContext) interface{}

	// Visit a parse tree produced by cqlParser#codesystemIdentifier.
	VisitCodesystemIdentifier(ctx *CodesystemIdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#libraryIdentifier.
	VisitLibraryIdentifier(ctx *LibraryIdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#codeDefinition.
	VisitCodeDefinition(ctx *CodeDefinitionContext) interface{}

	// Visit a parse tree produced by cqlParser#conceptDefinition.
	VisitConceptDefinition(ctx *ConceptDefinitionContext) interface{}

	// Visit a parse tree produced by cqlParser#codeIdentifier.
	VisitCodeIdentifier(ctx *CodeIdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#codesystemId.
	VisitCodesystemId(ctx *CodesystemIdContext) interface{}

	// Visit a parse tree produced by cqlParser#valuesetId.
	VisitValuesetId(ctx *ValuesetIdContext) interface{}

	// Visit a parse tree produced by cqlParser#versionSpecifier.
	VisitVersionSpecifier(ctx *VersionSpecifierContext) interface{}

	// Visit a parse tree produced by cqlParser#codeId.
	VisitCodeId(ctx *CodeIdContext) interface{}

	// Visit a parse tree produced by cqlParser#typeSpecifier.
	VisitTypeSpecifier(ctx *TypeSpecifierContext) interface{}

	// Visit a parse tree produced by cqlParser#namedTypeSpecifier.
	VisitNamedTypeSpecifier(ctx *NamedTypeSpecifierContext) interface{}

	// Visit a parse tree produced by cqlParser#modelIdentifier.
	VisitModelIdentifier(ctx *ModelIdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#listTypeSpecifier.
	VisitListTypeSpecifier(ctx *ListTypeSpecifierContext) interface{}

	// Visit a parse tree produced by cqlParser#intervalTypeSpecifier.
	VisitIntervalTypeSpecifier(ctx *IntervalTypeSpecifierContext) interface{}

	// Visit a parse tree produced by cqlParser#tupleTypeSpecifier.
	VisitTupleTypeSpecifier(ctx *TupleTypeSpecifierContext) interface{}

	// Visit a parse tree produced by cqlParser#tupleElementDefinition.
	VisitTupleElementDefinition(ctx *TupleElementDefinitionContext) interface{}

	// Visit a parse tree produced by cqlParser#choiceTypeSpecifier.
	VisitChoiceTypeSpecifier(ctx *ChoiceTypeSpecifierContext) interface{}

	// Visit a parse tree produced by cqlParser#statement.
	VisitStatement(ctx *StatementContext) interface{}

	// Visit a parse tree produced by cqlParser#expressionDefinition.
	VisitExpressionDefinition(ctx *ExpressionDefinitionContext) interface{}

	// Visit a parse tree produced by cqlParser#contextDefinition.
	VisitContextDefinition(ctx *ContextDefinitionContext) interface{}

	// Visit a parse tree produced by cqlParser#fluentModifier.
	VisitFluentModifier(ctx *FluentModifierContext) interface{}

	// Visit a parse tree produced by cqlParser#functionDefinition.
	VisitFunctionDefinition(ctx *FunctionDefinitionContext) interface{}

	// Visit a parse tree produced by cqlParser#operandDefinition.
	VisitOperandDefinition(ctx *OperandDefinitionContext) interface{}

	// Visit a parse tree produced by cqlParser#functionBody.
	VisitFunctionBody(ctx *FunctionBodyContext) interface{}

	// Visit a parse tree produced by cqlParser#querySource.
	VisitQuerySource(ctx *QuerySourceContext) interface{}

	// Visit a parse tree produced by cqlParser#aliasedQuerySource.
	VisitAliasedQuerySource(ctx *AliasedQuerySourceContext) interface{}

	// Visit a parse tree produced by cqlParser#alias.
	VisitAlias(ctx *AliasContext) interface{}

	// Visit a parse tree produced by cqlParser#queryInclusionClause.
	VisitQueryInclusionClause(ctx *QueryInclusionClauseContext) interface{}

	// Visit a parse tree produced by cqlParser#withClause.
	VisitWithClause(ctx *WithClauseContext) interface{}

	// Visit a parse tree produced by cqlParser#withoutClause.
	VisitWithoutClause(ctx *WithoutClauseContext) interface{}

	// Visit a parse tree produced by cqlParser#retrieve.
	VisitRetrieve(ctx *RetrieveContext) interface{}

	// Visit a parse tree produced by cqlParser#contextIdentifier.
	VisitContextIdentifier(ctx *ContextIdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#codePath.
	VisitCodePath(ctx *CodePathContext) interface{}

	// Visit a parse tree produced by cqlParser#codeComparator.
	VisitCodeComparator(ctx *CodeComparatorContext) interface{}

	// Visit a parse tree produced by cqlParser#terminology.
	VisitTerminology(ctx *TerminologyContext) interface{}

	// Visit a parse tree produced by cqlParser#qualifier.
	VisitQualifier(ctx *QualifierContext) interface{}

	// Visit a parse tree produced by cqlParser#query.
	VisitQuery(ctx *QueryContext) interface{}

	// Visit a parse tree produced by cqlParser#sourceClause.
	VisitSourceClause(ctx *SourceClauseContext) interface{}

	// Visit a parse tree produced by cqlParser#letClause.
	VisitLetClause(ctx *LetClauseContext) interface{}

	// Visit a parse tree produced by cqlParser#letClauseItem.
	VisitLetClauseItem(ctx *LetClauseItemContext) interface{}

	// Visit a parse tree produced by cqlParser#whereClause.
	VisitWhereClause(ctx *WhereClauseContext) interface{}

	// Visit a parse tree produced by cqlParser#returnClause.
	VisitReturnClause(ctx *ReturnClauseContext) interface{}

	// Visit a parse tree produced by cqlParser#aggregateClause.
	VisitAggregateClause(ctx *AggregateClauseContext) interface{}

	// Visit a parse tree produced by cqlParser#startingClause.
	VisitStartingClause(ctx *StartingClauseContext) interface{}

	// Visit a parse tree produced by cqlParser#sortClause.
	VisitSortClause(ctx *SortClauseContext) interface{}

	// Visit a parse tree produced by cqlParser#sortDirection.
	VisitSortDirection(ctx *SortDirectionContext) interface{}

	// Visit a parse tree produced by cqlParser#sortByItem.
	VisitSortByItem(ctx *SortByItemContext) interface{}

	// Visit a parse tree produced by cqlParser#qualifiedIdentifier.
	VisitQualifiedIdentifier(ctx *QualifiedIdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#qualifiedIdentifierExpression.
	VisitQualifiedIdentifierExpression(ctx *QualifiedIdentifierExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#qualifierExpression.
	VisitQualifierExpression(ctx *QualifierExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#simplePathIndexer.
	VisitSimplePathIndexer(ctx *SimplePathIndexerContext) interface{}

	// Visit a parse tree produced by cqlParser#simplePathQualifiedIdentifier.
	VisitSimplePathQualifiedIdentifier(ctx *SimplePathQualifiedIdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#simplePathReferentialIdentifier.
	VisitSimplePathReferentialIdentifier(ctx *SimplePathReferentialIdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#simpleStringLiteral.
	VisitSimpleStringLiteral(ctx *SimpleStringLiteralContext) interface{}

	// Visit a parse tree produced by cqlParser#simpleNumberLiteral.
	VisitSimpleNumberLiteral(ctx *SimpleNumberLiteralContext) interface{}

	// Visit a parse tree produced by cqlParser#durationBetweenExpression.
	VisitDurationBetweenExpression(ctx *DurationBetweenExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#inFixSetExpression.
	VisitInFixSetExpression(ctx *InFixSetExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#retrieveExpression.
	VisitRetrieveExpression(ctx *RetrieveExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#timingExpression.
	VisitTimingExpression(ctx *TimingExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#queryExpression.
	VisitQueryExpression(ctx *QueryExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#notExpression.
	VisitNotExpression(ctx *NotExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#booleanExpression.
	VisitBooleanExpression(ctx *BooleanExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#orExpression.
	VisitOrExpression(ctx *OrExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#castExpression.
	VisitCastExpression(ctx *CastExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#andExpression.
	VisitAndExpression(ctx *AndExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#betweenExpression.
	VisitBetweenExpression(ctx *BetweenExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#membershipExpression.
	VisitMembershipExpression(ctx *MembershipExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#differenceBetweenExpression.
	VisitDifferenceBetweenExpression(ctx *DifferenceBetweenExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#inequalityExpression.
	VisitInequalityExpression(ctx *InequalityExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#equalityExpression.
	VisitEqualityExpression(ctx *EqualityExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#existenceExpression.
	VisitExistenceExpression(ctx *ExistenceExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#impliesExpression.
	VisitImpliesExpression(ctx *ImpliesExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#termExpression.
	VisitTermExpression(ctx *TermExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#typeExpression.
	VisitTypeExpression(ctx *TypeExpressionContext) interface{}

	// Visit a parse tree produced by cqlParser#dateTimePrecision.
	VisitDateTimePrecision(ctx *DateTimePrecisionContext) interface{}

	// Visit a parse tree produced by cqlParser#dateTimeComponent.
	VisitDateTimeComponent(ctx *DateTimeComponentContext) interface{}

	// Visit a parse tree produced by cqlParser#pluralDateTimePrecision.
	VisitPluralDateTimePrecision(ctx *PluralDateTimePrecisionContext) interface{}

	// Visit a parse tree produced by cqlParser#additionExpressionTerm.
	VisitAdditionExpressionTerm(ctx *AdditionExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#indexedExpressionTerm.
	VisitIndexedExpressionTerm(ctx *IndexedExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#widthExpressionTerm.
	VisitWidthExpressionTerm(ctx *WidthExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#setAggregateExpressionTerm.
	VisitSetAggregateExpressionTerm(ctx *SetAggregateExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#timeUnitExpressionTerm.
	VisitTimeUnitExpressionTerm(ctx *TimeUnitExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#ifThenElseExpressionTerm.
	VisitIfThenElseExpressionTerm(ctx *IfThenElseExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#timeBoundaryExpressionTerm.
	VisitTimeBoundaryExpressionTerm(ctx *TimeBoundaryExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#elementExtractorExpressionTerm.
	VisitElementExtractorExpressionTerm(ctx *ElementExtractorExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#conversionExpressionTerm.
	VisitConversionExpressionTerm(ctx *ConversionExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#typeExtentExpressionTerm.
	VisitTypeExtentExpressionTerm(ctx *TypeExtentExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#predecessorExpressionTerm.
	VisitPredecessorExpressionTerm(ctx *PredecessorExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#pointExtractorExpressionTerm.
	VisitPointExtractorExpressionTerm(ctx *PointExtractorExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#multiplicationExpressionTerm.
	VisitMultiplicationExpressionTerm(ctx *MultiplicationExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#aggregateExpressionTerm.
	VisitAggregateExpressionTerm(ctx *AggregateExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#durationExpressionTerm.
	VisitDurationExpressionTerm(ctx *DurationExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#differenceExpressionTerm.
	VisitDifferenceExpressionTerm(ctx *DifferenceExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#caseExpressionTerm.
	VisitCaseExpressionTerm(ctx *CaseExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#powerExpressionTerm.
	VisitPowerExpressionTerm(ctx *PowerExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#successorExpressionTerm.
	VisitSuccessorExpressionTerm(ctx *SuccessorExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#polarityExpressionTerm.
	VisitPolarityExpressionTerm(ctx *PolarityExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#termExpressionTerm.
	VisitTermExpressionTerm(ctx *TermExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#invocationExpressionTerm.
	VisitInvocationExpressionTerm(ctx *InvocationExpressionTermContext) interface{}

	// Visit a parse tree produced by cqlParser#caseExpressionItem.
	VisitCaseExpressionItem(ctx *CaseExpressionItemContext) interface{}

	// Visit a parse tree produced by cqlParser#dateTimePrecisionSpecifier.
	VisitDateTimePrecisionSpecifier(ctx *DateTimePrecisionSpecifierContext) interface{}

	// Visit a parse tree produced by cqlParser#relativeQualifier.
	VisitRelativeQualifier(ctx *RelativeQualifierContext) interface{}

	// Visit a parse tree produced by cqlParser#offsetRelativeQualifier.
	VisitOffsetRelativeQualifier(ctx *OffsetRelativeQualifierContext) interface{}

	// Visit a parse tree produced by cqlParser#exclusiveRelativeQualifier.
	VisitExclusiveRelativeQualifier(ctx *ExclusiveRelativeQualifierContext) interface{}

	// Visit a parse tree produced by cqlParser#quantityOffset.
	VisitQuantityOffset(ctx *QuantityOffsetContext) interface{}

	// Visit a parse tree produced by cqlParser#temporalRelationship.
	VisitTemporalRelationship(ctx *TemporalRelationshipContext) interface{}

	// Visit a parse tree produced by cqlParser#concurrentWithIntervalOperatorPhrase.
	VisitConcurrentWithIntervalOperatorPhrase(ctx *ConcurrentWithIntervalOperatorPhraseContext) interface{}

	// Visit a parse tree produced by cqlParser#includesIntervalOperatorPhrase.
	VisitIncludesIntervalOperatorPhrase(ctx *IncludesIntervalOperatorPhraseContext) interface{}

	// Visit a parse tree produced by cqlParser#includedInIntervalOperatorPhrase.
	VisitIncludedInIntervalOperatorPhrase(ctx *IncludedInIntervalOperatorPhraseContext) interface{}

	// Visit a parse tree produced by cqlParser#beforeOrAfterIntervalOperatorPhrase.
	VisitBeforeOrAfterIntervalOperatorPhrase(ctx *BeforeOrAfterIntervalOperatorPhraseContext) interface{}

	// Visit a parse tree produced by cqlParser#withinIntervalOperatorPhrase.
	VisitWithinIntervalOperatorPhrase(ctx *WithinIntervalOperatorPhraseContext) interface{}

	// Visit a parse tree produced by cqlParser#meetsIntervalOperatorPhrase.
	VisitMeetsIntervalOperatorPhrase(ctx *MeetsIntervalOperatorPhraseContext) interface{}

	// Visit a parse tree produced by cqlParser#overlapsIntervalOperatorPhrase.
	VisitOverlapsIntervalOperatorPhrase(ctx *OverlapsIntervalOperatorPhraseContext) interface{}

	// Visit a parse tree produced by cqlParser#startsIntervalOperatorPhrase.
	VisitStartsIntervalOperatorPhrase(ctx *StartsIntervalOperatorPhraseContext) interface{}

	// Visit a parse tree produced by cqlParser#endsIntervalOperatorPhrase.
	VisitEndsIntervalOperatorPhrase(ctx *EndsIntervalOperatorPhraseContext) interface{}

	// Visit a parse tree produced by cqlParser#invocationTerm.
	VisitInvocationTerm(ctx *InvocationTermContext) interface{}

	// Visit a parse tree produced by cqlParser#literalTerm.
	VisitLiteralTerm(ctx *LiteralTermContext) interface{}

	// Visit a parse tree produced by cqlParser#externalConstantTerm.
	VisitExternalConstantTerm(ctx *ExternalConstantTermContext) interface{}

	// Visit a parse tree produced by cqlParser#intervalSelectorTerm.
	VisitIntervalSelectorTerm(ctx *IntervalSelectorTermContext) interface{}

	// Visit a parse tree produced by cqlParser#tupleSelectorTerm.
	VisitTupleSelectorTerm(ctx *TupleSelectorTermContext) interface{}

	// Visit a parse tree produced by cqlParser#instanceSelectorTerm.
	VisitInstanceSelectorTerm(ctx *InstanceSelectorTermContext) interface{}

	// Visit a parse tree produced by cqlParser#listSelectorTerm.
	VisitListSelectorTerm(ctx *ListSelectorTermContext) interface{}

	// Visit a parse tree produced by cqlParser#codeSelectorTerm.
	VisitCodeSelectorTerm(ctx *CodeSelectorTermContext) interface{}

	// Visit a parse tree produced by cqlParser#conceptSelectorTerm.
	VisitConceptSelectorTerm(ctx *ConceptSelectorTermContext) interface{}

	// Visit a parse tree produced by cqlParser#parenthesizedTerm.
	VisitParenthesizedTerm(ctx *ParenthesizedTermContext) interface{}

	// Visit a parse tree produced by cqlParser#qualifiedMemberInvocation.
	VisitQualifiedMemberInvocation(ctx *QualifiedMemberInvocationContext) interface{}

	// Visit a parse tree produced by cqlParser#qualifiedFunctionInvocation.
	VisitQualifiedFunctionInvocation(ctx *QualifiedFunctionInvocationContext) interface{}

	// Visit a parse tree produced by cqlParser#qualifiedFunction.
	VisitQualifiedFunction(ctx *QualifiedFunctionContext) interface{}

	// Visit a parse tree produced by cqlParser#memberInvocation.
	VisitMemberInvocation(ctx *MemberInvocationContext) interface{}

	// Visit a parse tree produced by cqlParser#functionInvocation.
	VisitFunctionInvocation(ctx *FunctionInvocationContext) interface{}

	// Visit a parse tree produced by cqlParser#thisInvocation.
	VisitThisInvocation(ctx *ThisInvocationContext) interface{}

	// Visit a parse tree produced by cqlParser#indexInvocation.
	VisitIndexInvocation(ctx *IndexInvocationContext) interface{}

	// Visit a parse tree produced by cqlParser#totalInvocation.
	VisitTotalInvocation(ctx *TotalInvocationContext) interface{}

	// Visit a parse tree produced by cqlParser#function.
	VisitFunction(ctx *FunctionContext) interface{}

	// Visit a parse tree produced by cqlParser#ratio.
	VisitRatio(ctx *RatioContext) interface{}

	// Visit a parse tree produced by cqlParser#booleanLiteral.
	VisitBooleanLiteral(ctx *BooleanLiteralContext) interface{}

	// Visit a parse tree produced by cqlParser#nullLiteral.
	VisitNullLiteral(ctx *NullLiteralContext) interface{}

	// Visit a parse tree produced by cqlParser#stringLiteral.
	VisitStringLiteral(ctx *StringLiteralContext) interface{}

	// Visit a parse tree produced by cqlParser#numberLiteral.
	VisitNumberLiteral(ctx *NumberLiteralContext) interface{}

	// Visit a parse tree produced by cqlParser#longNumberLiteral.
	VisitLongNumberLiteral(ctx *LongNumberLiteralContext) interface{}

	// Visit a parse tree produced by cqlParser#dateTimeLiteral.
	VisitDateTimeLiteral(ctx *DateTimeLiteralContext) interface{}

	// Visit a parse tree produced by cqlParser#dateLiteral.
	VisitDateLiteral(ctx *DateLiteralContext) interface{}

	// Visit a parse tree produced by cqlParser#timeLiteral.
	VisitTimeLiteral(ctx *TimeLiteralContext) interface{}

	// Visit a parse tree produced by cqlParser#quantityLiteral.
	VisitQuantityLiteral(ctx *QuantityLiteralContext) interface{}

	// Visit a parse tree produced by cqlParser#ratioLiteral.
	VisitRatioLiteral(ctx *RatioLiteralContext) interface{}

	// Visit a parse tree produced by cqlParser#externalConstant.
	VisitExternalConstant(ctx *ExternalConstantContext) interface{}

	// Visit a parse tree produced by cqlParser#intervalSelector.
	VisitIntervalSelector(ctx *IntervalSelectorContext) interface{}

	// Visit a parse tree produced by cqlParser#tupleSelector.
	VisitTupleSelector(ctx *TupleSelectorContext) interface{}

	// Visit a parse tree produced by cqlParser#tupleElementSelector.
	VisitTupleElementSelector(ctx *TupleElementSelectorContext) interface{}

	// Visit a parse tree produced by cqlParser#instanceSelector.
	VisitInstanceSelector(ctx *InstanceSelectorContext) interface{}

	// Visit a parse tree produced by cqlParser#instanceElementSelector.
	VisitInstanceElementSelector(ctx *InstanceElementSelectorContext) interface{}

	// Visit a parse tree produced by cqlParser#listSelector.
	VisitListSelector(ctx *ListSelectorContext) interface{}

	// Visit a parse tree produced by cqlParser#displayClause.
	VisitDisplayClause(ctx *DisplayClauseContext) interface{}

	// Visit a parse tree produced by cqlParser#codeSelector.
	VisitCodeSelector(ctx *CodeSelectorContext) interface{}

	// Visit a parse tree produced by cqlParser#conceptSelector.
	VisitConceptSelector(ctx *ConceptSelectorContext) interface{}

	// Visit a parse tree produced by cqlParser#keyword.
	VisitKeyword(ctx *KeywordContext) interface{}

	// Visit a parse tree produced by cqlParser#reservedWord.
	VisitReservedWord(ctx *ReservedWordContext) interface{}

	// Visit a parse tree produced by cqlParser#keywordIdentifier.
	VisitKeywordIdentifier(ctx *KeywordIdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#obsoleteIdentifier.
	VisitObsoleteIdentifier(ctx *ObsoleteIdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#functionIdentifier.
	VisitFunctionIdentifier(ctx *FunctionIdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#typeNameIdentifier.
	VisitTypeNameIdentifier(ctx *TypeNameIdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#referentialIdentifier.
	VisitReferentialIdentifier(ctx *ReferentialIdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#referentialOrTypeNameIdentifier.
	VisitReferentialOrTypeNameIdentifier(ctx *ReferentialOrTypeNameIdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#identifierOrFunctionIdentifier.
	VisitIdentifierOrFunctionIdentifier(ctx *IdentifierOrFunctionIdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#identifier.
	VisitIdentifier(ctx *IdentifierContext) interface{}

	// Visit a parse tree produced by cqlParser#paramList.
	VisitParamList(ctx *ParamListContext) interface{}

	// Visit a parse tree produced by cqlParser#quantity.
	VisitQuantity(ctx *QuantityContext) interface{}

	// Visit a parse tree produced by cqlParser#unit.
	VisitUnit(ctx *UnitContext) interface{}
}
