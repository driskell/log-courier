package ast

import (
	"github.com/driskell/log-courier/lc-lib/event"
)

// unsetNode processes an unset statement
type unsetNode struct {
	identifier string
}

var _ ProcessNode = &unsetNode{}

// Process handles logic for the event
func (n *unsetNode) Process(subject *event.Event) *event.Event {
	if _, err := subject.Resolve(n.identifier, event.ResolveParamUnset); err != nil {
		log.Warningf("Failed to evaluate unset target: [%s] -> %s", n.identifier, err)
	}
	return subject
}
