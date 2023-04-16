package ast

import (
	"fmt"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/driskell/log-courier/lc-lib/processor/gen"
	"github.com/google/cel-go/common/types"
)

type Visitor struct {
	gen.BaseLogCarverVisitor

	errorListener antlr.ErrorListener
}

func NewVisitor(errorListener antlr.ErrorListener) *Visitor {
	return &Visitor{
		errorListener: errorListener,
	}
}

func (v *Visitor) Visit(tree antlr.ParseTree) interface{} {
	switch val := tree.(type) {
	case *gen.ProgramContext:
		return val.Accept(v).([]ProcessNode)
	default:
		panic("UNKNOWN")
	}
}

func (v *Visitor) VisitProgram(ctx *gen.ProgramContext) interface{} {
	return ctx.Lines().Accept(v).([]ProcessNode)
}

func (v *Visitor) VisitLines(ctx *gen.LinesContext) interface{} {
	lines := ctx.AllLine()
	nodes := make([]ProcessNode, 0, len(lines))
	for _, line := range lines {
		nodes = append(nodes, line.Accept(v).(ProcessNode))
	}
	return nodes
}

func (v *Visitor) VisitLine(ctx *gen.LineContext) interface{} {
	ifexpr := ctx.Ifexpr()
	if ifexpr != nil {
		return ifexpr.Accept(v).(ProcessNode)
	}
	return ctx.Statement().Accept(v).(ProcessNode)
}

func (v *Visitor) VisitStatement(ctx *gen.StatementContext) interface{} {
	if set := ctx.Set(); set != nil {
		return set.Accept(v).(ProcessNode)
	}
	if unset := ctx.Unset(); unset != nil {
		return unset.Accept(v).(ProcessNode)
	}
	action := ctx.Action_()
	return action.Accept(v).(ProcessNode)
}

func (v *Visitor) VisitIfexpr(ctx *gen.IfexprContext) interface{} {
	allelseifexpr := ctx.AllElseifexpr()
	elseexpr := ctx.Elseexpr()
	block := ctx.Block()
	var err error
	node := &ifNode{}
	node.condition, err = newCelProgramNode(ctx.Condition().Accept(v).(string))
	if err != nil {
		panic(err)
	}
	node.elseIfExpr = make([]*elseIfExpr, 0, len(allelseifexpr))
	for _, elseifexpr := range allelseifexpr {
		node.elseIfExpr = append(node.elseIfExpr, elseifexpr.Accept(v).(*elseIfExpr))
	}
	if elseexpr != nil {
		node.elseExpr = elseexpr.Accept(v).(*elseExpr)
	}
	node.block = block.Accept(v).([]ProcessNode)
	return node
}

func (v *Visitor) VisitElseifexpr(ctx *gen.ElseifexprContext) interface{} {
	block := ctx.Block()
	var err error
	node := &elseIfExpr{}
	node.condition, err = newCelProgramNode(ctx.Condition().Accept(v).(string))
	if err != nil {
		panic(err)
	}
	node.block = block.Accept(v).([]ProcessNode)
	return node
}

func (v *Visitor) VisitElseexpr(ctx *gen.ElseexprContext) interface{} {
	block := ctx.Block()
	node := &elseExpr{}
	node.block = block.Accept(v).([]ProcessNode)
	return node
}

func (v *Visitor) VisitCondition(ctx *gen.ConditionContext) interface{} {
	return ctx.Expr().Accept(v).(string)
}

func (v *Visitor) VisitExpr(ctx *gen.ExprContext) interface{} {
	return ctx.GetText()
}

func (v *Visitor) VisitLiteral(ctx *gen.ExprContext) interface{} {
	return ctx.GetText()
}

func (v *Visitor) VisitBlock(ctx *gen.BlockContext) interface{} {
	return ctx.Lines().Accept(v).([]ProcessNode)
}

func (v *Visitor) VisitSet(ctx *gen.SetContext) interface{} {
	resolver := ctx.Resolver().Accept(v).(string)
	var err error
	node := &setNode{}
	node.identifier = resolver
	node.value, err = newCelProgramNode(ctx.Expr().Accept(v).(string))
	if err != nil {
		panic(err)
	}
	return node
}

func (v *Visitor) VisitUnset(ctx *gen.UnsetContext) interface{} {
	resolver := ctx.Resolver().Accept(v).(string)
	node := &unsetNode{}
	node.identifier = resolver
	return node
}

func (v *Visitor) VisitResolver(ctx *gen.ResolverContext) interface{} {
	return ctx.GetText()
}

func (v *Visitor) VisitAction(ctx *gen.ActionContext) interface{} {
	identifier := ctx.IDENTIFIER().GetText()
	allargument := ctx.AllArgument()
	values := make(map[string]ValueNode)
	for _, argument := range allargument {
		argumentData := argument.Accept(v).(struct {
			name string
			node ValueNode
		})
		values[argumentData.name] = argumentData.node
	}
	node, err := newActionNode(identifier)
	if err != nil {
		v.errorListener.SyntaxError(ctx.GetParser(), nil, ctx.GetStart().GetLine(), ctx.GetStart().GetColumn(), err.Error(), nil)
		return &noopNode{}
	}
	resolverNode, err := newArgumentResolverNode(node, values)
	if err != nil {
		v.errorListener.SyntaxError(ctx.GetParser(), nil, ctx.GetStart().GetLine(), ctx.GetStart().GetColumn(), fmt.Sprintf("invalid arguments to %s: %s", identifier, err.Error()), nil)
		return &noopNode{}
	}
	return resolverNode
}

func (v *Visitor) VisitArgument(ctx *gen.ArgumentContext) interface{} {
	name := ctx.IDENTIFIER().GetText()
	node, err := newCelProgramNode(ctx.Expr().Accept(v).(string))
	if err != nil {
		v.errorListener.SyntaxError(ctx.GetParser(), nil, ctx.GetStart().GetLine(), ctx.GetStart().GetColumn(), err.Error(), nil)
		node = &literalNode{value: types.Unknown{}}
	}
	return struct {
		name string
		node ValueNode
	}{name, node}
}
