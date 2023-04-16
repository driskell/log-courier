// Code generated from LogCarver.c4 by ANTLR 4.12.0. DO NOT EDIT.

package gen // LogCarver
import "github.com/antlr/antlr4/runtime/Go/antlr/v4"

// A complete Visitor for a parse tree produced by LogCarverParser.
type LogCarverVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by LogCarverParser#program.
	VisitProgram(ctx *ProgramContext) interface{}

	// Visit a parse tree produced by LogCarverParser#lines.
	VisitLines(ctx *LinesContext) interface{}

	// Visit a parse tree produced by LogCarverParser#line.
	VisitLine(ctx *LineContext) interface{}

	// Visit a parse tree produced by LogCarverParser#statement.
	VisitStatement(ctx *StatementContext) interface{}

	// Visit a parse tree produced by LogCarverParser#ifexpr.
	VisitIfexpr(ctx *IfexprContext) interface{}

	// Visit a parse tree produced by LogCarverParser#elseifexpr.
	VisitElseifexpr(ctx *ElseifexprContext) interface{}

	// Visit a parse tree produced by LogCarverParser#elseexpr.
	VisitElseexpr(ctx *ElseexprContext) interface{}

	// Visit a parse tree produced by LogCarverParser#condition.
	VisitCondition(ctx *ConditionContext) interface{}

	// Visit a parse tree produced by LogCarverParser#block.
	VisitBlock(ctx *BlockContext) interface{}

	// Visit a parse tree produced by LogCarverParser#set.
	VisitSet(ctx *SetContext) interface{}

	// Visit a parse tree produced by LogCarverParser#unset.
	VisitUnset(ctx *UnsetContext) interface{}

	// Visit a parse tree produced by LogCarverParser#resolver.
	VisitResolver(ctx *ResolverContext) interface{}

	// Visit a parse tree produced by LogCarverParser#action.
	VisitAction(ctx *ActionContext) interface{}

	// Visit a parse tree produced by LogCarverParser#argument.
	VisitArgument(ctx *ArgumentContext) interface{}

	// Visit a parse tree produced by LogCarverParser#expr.
	VisitExpr(ctx *ExprContext) interface{}

	// Visit a parse tree produced by LogCarverParser#conditionalOr.
	VisitConditionalOr(ctx *ConditionalOrContext) interface{}

	// Visit a parse tree produced by LogCarverParser#conditionalAnd.
	VisitConditionalAnd(ctx *ConditionalAndContext) interface{}

	// Visit a parse tree produced by LogCarverParser#relation.
	VisitRelation(ctx *RelationContext) interface{}

	// Visit a parse tree produced by LogCarverParser#calc.
	VisitCalc(ctx *CalcContext) interface{}

	// Visit a parse tree produced by LogCarverParser#MemberExpr.
	VisitMemberExpr(ctx *MemberExprContext) interface{}

	// Visit a parse tree produced by LogCarverParser#LogicalNot.
	VisitLogicalNot(ctx *LogicalNotContext) interface{}

	// Visit a parse tree produced by LogCarverParser#Negate.
	VisitNegate(ctx *NegateContext) interface{}

	// Visit a parse tree produced by LogCarverParser#MemberCall.
	VisitMemberCall(ctx *MemberCallContext) interface{}

	// Visit a parse tree produced by LogCarverParser#Select.
	VisitSelect(ctx *SelectContext) interface{}

	// Visit a parse tree produced by LogCarverParser#PrimaryExpr.
	VisitPrimaryExpr(ctx *PrimaryExprContext) interface{}

	// Visit a parse tree produced by LogCarverParser#Index.
	VisitIndex(ctx *IndexContext) interface{}

	// Visit a parse tree produced by LogCarverParser#IdentOrGlobalCall.
	VisitIdentOrGlobalCall(ctx *IdentOrGlobalCallContext) interface{}

	// Visit a parse tree produced by LogCarverParser#Nested.
	VisitNested(ctx *NestedContext) interface{}

	// Visit a parse tree produced by LogCarverParser#CreateList.
	VisitCreateList(ctx *CreateListContext) interface{}

	// Visit a parse tree produced by LogCarverParser#CreateStruct.
	VisitCreateStruct(ctx *CreateStructContext) interface{}

	// Visit a parse tree produced by LogCarverParser#CreateMessage.
	VisitCreateMessage(ctx *CreateMessageContext) interface{}

	// Visit a parse tree produced by LogCarverParser#ConstantLiteral.
	VisitConstantLiteral(ctx *ConstantLiteralContext) interface{}

	// Visit a parse tree produced by LogCarverParser#exprList.
	VisitExprList(ctx *ExprListContext) interface{}

	// Visit a parse tree produced by LogCarverParser#listInit.
	VisitListInit(ctx *ListInitContext) interface{}

	// Visit a parse tree produced by LogCarverParser#fieldInitializerList.
	VisitFieldInitializerList(ctx *FieldInitializerListContext) interface{}

	// Visit a parse tree produced by LogCarverParser#optField.
	VisitOptField(ctx *OptFieldContext) interface{}

	// Visit a parse tree produced by LogCarverParser#mapInitializerList.
	VisitMapInitializerList(ctx *MapInitializerListContext) interface{}

	// Visit a parse tree produced by LogCarverParser#optExpr.
	VisitOptExpr(ctx *OptExprContext) interface{}

	// Visit a parse tree produced by LogCarverParser#Int.
	VisitInt(ctx *IntContext) interface{}

	// Visit a parse tree produced by LogCarverParser#Uint.
	VisitUint(ctx *UintContext) interface{}

	// Visit a parse tree produced by LogCarverParser#Double.
	VisitDouble(ctx *DoubleContext) interface{}

	// Visit a parse tree produced by LogCarverParser#String.
	VisitString(ctx *StringContext) interface{}

	// Visit a parse tree produced by LogCarverParser#Bytes.
	VisitBytes(ctx *BytesContext) interface{}

	// Visit a parse tree produced by LogCarverParser#BoolTrue.
	VisitBoolTrue(ctx *BoolTrueContext) interface{}

	// Visit a parse tree produced by LogCarverParser#BoolFalse.
	VisitBoolFalse(ctx *BoolFalseContext) interface{}

	// Visit a parse tree produced by LogCarverParser#Null.
	VisitNull(ctx *NullContext) interface{}
}
