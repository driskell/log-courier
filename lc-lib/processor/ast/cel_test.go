package ast

import (
	"context"
	"reflect"
	"testing"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/google/cel-go/common/types"
)

func TestCelNode(t *testing.T) {
	node, err := newCelProgramNode("event.value == \"test\"")
	if err != nil {
		t.Fatalf("program creation failed: %s", err)
	}
	subject := event.NewEvent(context.Background(), nil, map[string]interface{}{"value": "test"})
	result := node.Value(subject)
	if result != types.True {
		t.Fatalf("expected true result")
	}
	subject = event.NewEvent(context.Background(), nil, map[string]interface{}{"value": "nottest"})
	result = node.Value(subject)
	if result != types.False {
		t.Fatalf("expected false result")
	}
}

func TestCelNodeOptimizeToLiteral(t *testing.T) {
	node, err := newCelProgramNode("\"test\"")
	if err != nil {
		t.Fatalf("program creation failed: %s", err)
	}
	result, ok := node.(*literalNode)
	if !ok {
		t.Fatal("program expected to be literal result")
	}
	subject := event.NewEvent(context.Background(), nil, map[string]interface{}{"value": "nottest"})
	resultValue, err := result.Value(subject).ConvertToNative(reflect.TypeOf(""))
	if err != nil {
		t.Fatalf("result conversion failed: %s", err)
	}
	resultString := resultValue.(string)
	if resultString != "test" {
		t.Fatalf("program result expected to be literal \"test\" but received %s", resultString)
	}
}

func TestCelNodeFailure(t *testing.T) {
	_, err := newCelProgramNode(";fail")
	if err == nil {
		t.Fatalf("program creation did not fail")
	}
}
