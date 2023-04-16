package ast

import (
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/google/cel-go/common/types/ref"
)

// ProcessNode is a node in the syntax tree that processes events
type ProcessNode interface {
	Process(*event.Event) *event.Event
}

// ProcessArgumentsNode is a node in the syntax stree that processed arguments for a ProcessNode
type ProcessArgumentsNode interface {
	// ProcessWithArguments is called to process the event, and the second argument
	// contains the full argument list as defined by Arguments()
	ProcessWithArguments(*event.Event, []any) *event.Event
	// Arguments returns the set of arguments the node accepts, and if they
	// accept expressions or not
	Arguments() []Argument
	// Init is called after all non-expression arguments are set, and can be
	// used to initialise using any argument that does not allow expressions
	// All arguments required for initialisation should be set to not allow
	// expressions, and it will be within the arguments slice given. Any
	// expressions will be in the slice as nil values
	Init([]any) error
}

// ValueNode is a node in the syntax tree that returns a value
type ValueNode interface {
	Value(*event.Event) ref.Val
}
