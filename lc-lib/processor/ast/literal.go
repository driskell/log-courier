package ast

import (
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/google/cel-go/common/types/ref"
)

type literalNode struct {
	value ref.Val
}

var _ ValueNode = &literalNode{}

func newLiteralNode(value ref.Val) *literalNode {
	return &literalNode{value}
}

func (n *literalNode) Value(subject *event.Event) ref.Val {
	return n.value
}
