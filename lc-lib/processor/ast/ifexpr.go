package ast

import (
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/google/cel-go/common/types"
)

// ifNode processes an event through a conditional branch
type ifNode struct {
	condition  ValueNode
	block      []ProcessNode
	elseIfExpr []*elseIfExpr
	elseExpr   *elseExpr
}

var _ ProcessNode = &ifNode{}

// Process handles logic for the event
func (l *ifNode) Process(subject *event.Event) *event.Event {
	var next []ProcessNode
	if l.condition.Value(subject).ConvertToType(types.BoolType) == types.True {
		next = l.block
	} else {
		if len(l.elseIfExpr) != 0 {
			for _, elseIfBranch := range l.elseIfExpr {
				if elseIfBranch.condition.Value(subject).ConvertToType(types.BoolType) == types.True {
					next = elseIfBranch.block
					break
				}
			}
		}
		if next == nil {
			if l.elseExpr != nil {
				next = l.elseExpr.block
			} else {
				return subject
			}
		}
	}
	for _, entry := range next {
		subject = entry.Process(subject)
	}
	return subject
}

// elseIfExpr branch
type elseIfExpr struct {
	condition ValueNode
	block     []ProcessNode
}

// elseExpr branch
type elseExpr struct {
	block []ProcessNode
}
