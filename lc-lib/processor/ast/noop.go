package ast

import "github.com/driskell/log-courier/lc-lib/event"

type noopNode struct {
}

var _ ProcessNode = &noopNode{}

func (n *noopNode) Process(subject *event.Event) *event.Event {
	return subject
}
