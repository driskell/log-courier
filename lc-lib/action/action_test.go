package action

import (
	"context"
	"testing"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
)

func newTestNode(name string, arguments map[string]interface{}) (ast.ProcessNode, error) {
	values := make(map[string]ast.ValueNode, len(arguments))
	for key, value := range arguments {
		values[key] = ast.LegacyLiteral(value)
	}
	return ast.LegacyFetchAction(nil, name, values)
}

func mustNewTestNode(t *testing.T, name string, arguments map[string]interface{}) ast.ProcessNode {
	t.Helper()
	node, err := newTestNode(name, arguments)
	if err != nil {
		t.Fatalf("Creating %s action failed unexpectedly: %s", name, err)
	}
	return node
}

func processTestEvent(t *testing.T, name string, arguments map[string]interface{}, data map[string]interface{}) *event.Event {
	t.Helper()
	return mustNewTestNode(t, name, arguments).Process(event.NewEvent(context.Background(), nil, data))
}
