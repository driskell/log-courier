package ast

import (
	"reflect"
	"testing"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/google/cel-go/common/types"
)

// CustomActionNode for testing rendering
// Mimics the structure of ActionNode from actions package
// but is defined here for isolated testing

type CustomActionNode struct {
	Name string
}

// Implement ScriptNode for CustomActionNode
func (c *CustomActionNode) RenderScript() string {
	return "action: " + c.Name
}

// Dummy implementations for test
type testActionNode struct{}

func (t *testActionNode) ProcessWithArguments(_ *event.Event, _ []any) *event.Event { return nil }
func (t *testActionNode) Arguments() []Argument {
	return []Argument{
		&argumentString{argument: argument{"field", ArgumentRequired}},
		&argumentString{argument: argument{"value", ArgumentOptional}},
	}
}
func (t *testActionNode) Init(_ []any) error { return nil }

func TestUpgradeScript_ASTPipeline(t *testing.T) {
	// Build a representative AST
	root := []ProcessNode{
		&setNode{identifier: "type", value: &literalNode{value: types.String("json")}},
		&unsetNode{identifier: "old_field"},
		&staticArgumentResolverNode{
			processNode: &testActionNode{},
			values:      []any{"message", "hello"},
		},
		&ifNode{
			condition: &celProgramNode{source: "event.message == 'test'"},
			block: []ProcessNode{
				&setNode{identifier: "flag", value: &literalNode{value: types.Bool(true)}},
			},
			elseIfExpr: []*elseIfExpr{
				{
					condition: &celProgramNode{source: "event.message == 'other'"},
					block: []ProcessNode{
						&setNode{identifier: "flag", value: &literalNode{value: types.Bool(false)}},
					},
				},
			},
			elseExpr: &elseExpr{
				block: []ProcessNode{
					&noopNode{},
				},
			},
		},
	}

	script := UpgradeScript(root)

	expected := `set type = json;
unset old_field;
testAction field=message, value=hello;
if (event.message == 'test') {
  set flag = true;
} else if (event.message == 'other') {
  set flag = false;
} else {
}
`

	if !reflect.DeepEqual(script, expected) {
		t.Errorf("Script output mismatch.\nExpected:\n%s\nGot:\n%s", expected, script)
	}
}
