package ast

import (
	"github.com/driskell/log-courier/lc-lib/event"
)

// setNode processes a set statement
type setNode struct {
	identifier string
	value      ValueNode
}

var _ ProcessNode = &setNode{}

// Process handles logic for the event
func (n *setNode) Process(subject *event.Event) *event.Event {
	val := n.value.Value(subject)
	if val == nil {
		return subject
	}
	if _, err := subject.Resolve(n.identifier, val.Value()); err != nil {
		log.Warningf("Failed to evaluate set target: [%s] -> %s", n.identifier, err)
	}
	return subject
}
