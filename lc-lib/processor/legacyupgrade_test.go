package processor

import (
	"testing"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
)

// testActionNoArgs is a test action with no arguments
type testActionNoArgs struct {
}

var _ ast.ProcessArgumentsNode = &testActionNoArgs{}

func newTestActionNoArgs(*config.Config) (ast.ProcessArgumentsNode, error) {
	return &testActionNoArgs{}, nil
}

func (n *testActionNoArgs) Arguments() []ast.Argument {
	return []ast.Argument{}
}

func (n *testActionNoArgs) Init([]any) error {
	return nil
}

func (n *testActionNoArgs) ProcessWithArguments(subject *event.Event, arguments []any) *event.Event {
	return subject
}

// testActionTwoArgs is a test action with two arguments
type testActionTwoArgs struct {
}

var _ ast.ProcessArgumentsNode = &testActionTwoArgs{}

func newTestActionTwoArgs(*config.Config) (ast.ProcessArgumentsNode, error) {
	return &testActionTwoArgs{}, nil
}

func (n *testActionTwoArgs) Arguments() []ast.Argument {
	return []ast.Argument{
		ast.NewArgumentString("field", ast.ArgumentRequired),
		ast.NewArgumentString("value", ast.ArgumentRequired),
	}
}

func (n *testActionTwoArgs) Init([]any) error {
	return nil
}

func (n *testActionTwoArgs) ProcessWithArguments(subject *event.Event, arguments []any) *event.Event {
	return subject
}

// testActionStringList is a test action with an optional list of strings argument
type testActionStringList struct {
}

var _ ast.ProcessArgumentsNode = &testActionStringList{}

func newTestActionStringList(*config.Config) (ast.ProcessArgumentsNode, error) {
	return &testActionStringList{}, nil
}

func (n *testActionStringList) Arguments() []ast.Argument {
	return []ast.Argument{
		ast.NewArgumentListString("tags", ast.ArgumentOptional),
	}
}

func (n *testActionStringList) Init([]any) error {
	return nil
}

func (n *testActionStringList) ProcessWithArguments(subject *event.Event, arguments []any) *event.Event {
	return subject
}

func init() {
	// Register test actions for testing
	ast.RegisterAction("test_no_args", newTestActionNoArgs)
	ast.RegisterAction("test_two_args", newTestActionTwoArgs)
	ast.RegisterAction("test_string_list", newTestActionStringList)
}

func TestLegacyUpgradeScriptSimpleAction(t *testing.T) {
	legacyCfg := &LegacyConfig{
		Pipeline: []*LegacyConfigASTEntry{
			{
				Logic: map[string]interface{}{
					"name": "test_no_args",
				},
			},
		},
	}

	script, err := LegacyUpgradeScript(nil, legacyCfg)
	if err != nil {
		t.Fatalf("LegacyUpgradeScript failed: %v", err)
	}

	expected := "test_no_args;\n"
	if script != expected {
		t.Errorf("Script mismatch.\nExpected:\n%s\nGot:\n%s", expected, script)
	}
}

func TestLegacyUpgradeScriptMultipleActions(t *testing.T) {
	legacyCfg := &LegacyConfig{
		Pipeline: []*LegacyConfigASTEntry{
			{
				Logic: map[string]interface{}{
					"name": "test_no_args",
				},
			},
			{
				Logic: map[string]interface{}{
					"name":  "test_two_args",
					"field": "message",
					"value": "test",
				},
			},
		},
	}

	script, err := LegacyUpgradeScript(nil, legacyCfg)
	if err != nil {
		t.Fatalf("LegacyUpgradeScript failed: %v", err)
	}

	expected := `test_no_args;
test_two_args field="message",
    value="test";
`
	if script != expected {
		t.Errorf("Script mismatch.\nExpected:\n%s\nGot:\n%s", expected, script)
	}
}

func TestLegacyUpgradeScriptWithArrayValues(t *testing.T) {
	legacyCfg := &LegacyConfig{
		Pipeline: []*LegacyConfigASTEntry{
			{
				Logic: map[string]interface{}{
					"name":  "test_two_args",
					"field": "message",
					"value": "processed",
				},
			},
		},
	}

	script, err := LegacyUpgradeScript(nil, legacyCfg)
	if err != nil {
		t.Fatalf("LegacyUpgradeScript failed: %v", err)
	}

	expected := `test_two_args field="message",
    value="processed";
`
	if script != expected {
		t.Errorf("Script mismatch.\nExpected:\n%s\nGot:\n%s", expected, script)
	}
}

func TestLegacyUpgradeScriptCompilationError(t *testing.T) {
	legacyCfg := &LegacyConfig{
		Pipeline: []*LegacyConfigASTEntry{
			{
				Logic: map[string]interface{}{
					"name": "unknown_action",
				},
			},
		},
	}

	// First verify compilation fails
	_, err := LegacyUpgradeScript(nil, legacyCfg)
	if err == nil {
		t.Fatalf("Expected LegacyUpgradeScript error for unknown action")
	}
}

func TestLegacyUpgradeScriptIfElseIfElse(t *testing.T) {
	legacyCfg := &LegacyConfig{
		Pipeline: []*LegacyConfigASTEntry{
			{
				Logic: map[string]interface{}{
					string(astTokenIf): "event.message.startsWith(\"{\")",
					"then": []interface{}{
						map[string]interface{}{
							"name":  "test_two_args",
							"field": "type",
							"value": "json",
						},
					},
				},
			},
			{
				Logic: map[string]interface{}{
					string(astTokenElseIf): "event.message.startsWith(\"[\")",
					"then": []interface{}{
						map[string]interface{}{
							"name":  "test_two_args",
							"field": "type",
							"value": "grok",
						},
					},
				},
			},
			{
				Logic: map[string]interface{}{
					string(astTokenElse): []interface{}{
						map[string]interface{}{
							"name": "test_no_args",
						},
					},
				},
			},
			{
				Logic: map[string]interface{}{
					string(astTokenIf): "event.type == \"error\"",
					"then": []interface{}{
						map[string]interface{}{
							"name":  "test_two_args",
							"field": "severity",
							"value": "high",
						},
					},
				},
			},
			{
				Logic: map[string]interface{}{
					string(astTokenElse): []interface{}{
						map[string]interface{}{
							"name":  "test_two_args",
							"field": "severity",
							"value": "low",
						},
					},
				},
			},
		},
	}

	// Test the script generation
	script, err := LegacyUpgradeScript(nil, legacyCfg)
	if err != nil {
		t.Fatalf("LegacyUpgradeScript failed: %v", err)
	}

	expected := `if (event.message.startsWith("{")) {
    test_two_args field="type",
        value="json";
}
else if (event.message.startsWith("[")) {
    test_two_args field="type",
        value="grok";
}
else {
    test_no_args;
}

if (event.type == "error") {
    test_two_args field="severity",
        value="high";
}
else {
    test_two_args field="severity",
        value="low";
}
`
	if script != expected {
		t.Errorf("Script mismatch.\nExpected:\n%s\nGot:\n%s", expected, script)
	}
}

func TestLegacyUpgradeScriptNestedIf(t *testing.T) {
	legacyCfg := &LegacyConfig{
		Pipeline: []*LegacyConfigASTEntry{
			{
				Logic: map[string]interface{}{
					string(astTokenIf): "event.level == \"debug\"",
					"then": []interface{}{
						map[string]interface{}{
							string(astTokenIf): "event.message.contains(\"test\")",
							"then": []interface{}{
								map[string]interface{}{
									"name":  "test_two_args",
									"field": "tag",
									"value": "debug_test",
								},
							},
						},
						map[string]interface{}{
							string(astTokenElse): []interface{}{
								map[string]interface{}{
									"name":  "test_two_args",
									"field": "tag",
									"value": "debug_other",
								},
							},
						},
					},
				},
			},
		},
	}

	// Test the script generation
	script, err := LegacyUpgradeScript(nil, legacyCfg)
	if err != nil {
		t.Fatalf("LegacyUpgradeScript failed: %v", err)
	}

	expected := `if (event.level == "debug") {
    if (event.message.contains("test")) {
        test_two_args field="tag",
            value="debug_test";
    }
    else {
        test_two_args field="tag",
            value="debug_other";
    }
}
`
	if script != expected {
		t.Errorf("Script mismatch.\nExpected:\n%s\nGot:\n%s", expected, script)
	}
}

func TestLegacyUpgradeScriptStringListOptionalArgument(t *testing.T) {
	legacyCfg := &LegacyConfig{
		Pipeline: []*LegacyConfigASTEntry{
			{
				Logic: map[string]interface{}{
					"name": "test_string_list",
					"tags": []interface{}{"error", "critical", "warning"},
				},
			},
			{
				Logic: map[string]interface{}{
					"name": "test_string_list",
				},
			},
		},
	}

	script, err := LegacyUpgradeScript(nil, legacyCfg)
	if err != nil {
		t.Fatalf("LegacyUpgradeScript failed: %v", err)
	}

	expected := `test_string_list tags=[
    "error",
    "critical",
    "warning"
];
test_string_list;
`
	if script != expected {
		t.Errorf("Script mismatch.\nExpected:\n%s\nGot:\n%s", expected, script)
	}
}
