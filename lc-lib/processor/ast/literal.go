package ast

import (
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/google/cel-go/common/types"
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

type unknownNode struct {
	value      ref.Val
	actualType string
}

var _ ValueNode = &unknownNode{}

func newUnknownNode(reason string) *unknownNode {
	return &unknownNode{types.UnknownType, reason}
}

func (n *unknownNode) Value(subject *event.Event) ref.Val {
	return n.value
}
