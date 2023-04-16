// Code generated from LogCarver.c4 by ANTLR 4.12.0. DO NOT EDIT.

package gen // LogCarver
import "github.com/antlr/antlr4/runtime/Go/antlr/v4"

type BaseLogCarverVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseLogCarverVisitor) VisitProgram(ctx *ProgramContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitLines(ctx *LinesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitLine(ctx *LineContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitStatement(ctx *StatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitIfexpr(ctx *IfexprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitElseifexpr(ctx *ElseifexprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitElseexpr(ctx *ElseexprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitCondition(ctx *ConditionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitBlock(ctx *BlockContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitSet(ctx *SetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitUnset(ctx *UnsetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitResolver(ctx *ResolverContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitAction(ctx *ActionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitArgument(ctx *ArgumentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitExpr(ctx *ExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitConditionalOr(ctx *ConditionalOrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitConditionalAnd(ctx *ConditionalAndContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitRelation(ctx *RelationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitCalc(ctx *CalcContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitMemberExpr(ctx *MemberExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitLogicalNot(ctx *LogicalNotContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitNegate(ctx *NegateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitMemberCall(ctx *MemberCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitSelect(ctx *SelectContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitPrimaryExpr(ctx *PrimaryExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitIndex(ctx *IndexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitIdentOrGlobalCall(ctx *IdentOrGlobalCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitNested(ctx *NestedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitCreateList(ctx *CreateListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitCreateStruct(ctx *CreateStructContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitCreateMessage(ctx *CreateMessageContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitConstantLiteral(ctx *ConstantLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitExprList(ctx *ExprListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitListInit(ctx *ListInitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitFieldInitializerList(ctx *FieldInitializerListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitOptField(ctx *OptFieldContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitMapInitializerList(ctx *MapInitializerListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitOptExpr(ctx *OptExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitInt(ctx *IntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitUint(ctx *UintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitDouble(ctx *DoubleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitString(ctx *StringContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitBytes(ctx *BytesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitBoolTrue(ctx *BoolTrueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitBoolFalse(ctx *BoolFalseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseLogCarverVisitor) VisitNull(ctx *NullContext) interface{} {
	return v.VisitChildren(ctx)
}
