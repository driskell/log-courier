package ast

import (
	"context"
	"fmt"
	"testing"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/google/cel-go/common/types"
)

type testProcessNode struct {
	shouldInitErr error

	strInit         string
	strRequiredInit string
	strNoexprInit   string

	str         string
	strRequired string
	strNoexpr   string
}

func (n *testProcessNode) ProcessWithArguments(subject *event.Event, arguments []any) *event.Event {
	if arguments[0] != nil {
		n.str = arguments[0].(string)
	}
	if arguments[1] != nil {
		n.strRequired = arguments[1].(string)
	}
	if arguments[2] != nil {
		n.strNoexpr = arguments[2].(string)
	}
	return subject
}

func (n *testProcessNode) Init(arguments []any) error {
	if arguments[0] != nil {
		n.strInit = arguments[0].(string)
	}
	if arguments[1] != nil {
		n.strRequiredInit = arguments[1].(string)
	}
	if arguments[2] != nil {
		n.strNoexprInit = arguments[2].(string)
	}
	if n.shouldInitErr != nil {
		return n.shouldInitErr
	}
	return nil
}

func (n *testProcessNode) Arguments() []Argument {
	return []Argument{
		NewArgumentString("str", ArgumentOptional),
		NewArgumentString("required", ArgumentRequired),
		NewArgumentString("noexpr", ArgumentExprDisallowed),
	}
}

func TestArgumentResolverOptionalRequired(t *testing.T) {
	node := &testProcessNode{}
	values := map[string]ValueNode{}
	_, err := newArgumentResolverNode(node, values)
	if err == nil || err.Error() != "missing required arguments: required" {
		t.Fatalf("expected missing required arguments error, received %s", err)
	}
	values["required"] = &literalNode{value: types.String("test")}
	_, err = newArgumentResolverNode(node, values)
	if err != nil {
		t.Fatalf("expected success, received %s", err)
	}
	values["str"] = &literalNode{value: types.String("test2")}
	resolver, err := newArgumentResolverNode(node, values)
	if err != nil {
		t.Fatalf("expected success, received %s", err)
	}
	if node.strInit != "test2" {
		t.Fatal("expected non-required argument to be set during Init")
	}
	if node.strNoexprInit != "" {
		t.Fatal("expected non-required argument to be zero during Init")
	}
	if node.strRequiredInit != "test" {
		t.Fatal("expected required argument to be set during Init")
	}
	subject := event.NewEvent(context.Background(), nil, map[string]interface{}{"program": "result"})
	resolver.Process(subject)
	if node.str != "test2" {
		t.Fatal("expected non-required argument to be set during process")
	}
	if node.strNoexpr != "" {
		t.Fatal("expected non-required argument to be zero during process")
	}
	if node.strRequired != "test" {
		t.Fatal("expected required argument to be set during process")
	}
}

func TestArgumentResolverLiteralExpression(t *testing.T) {
	node := &testProcessNode{}
	values := map[string]ValueNode{}
	_, err := newArgumentResolverNode(node, values)
	if err == nil || err.Error() != "missing required arguments: required" {
		t.Fatalf("expected missing required arguments error, received %s", err)
	}
	values["required"], err = newCelProgramNode("\"program\"")
	if err != nil {
		t.Fatalf("expected celNode program success, received %s", err)
	}
	resolver, err := newArgumentResolverNode(node, values)
	if err != nil {
		t.Fatalf("expected success, received %s", err)
	}
	if _, ok := resolver.(*staticArgumentResolverNode); !ok {
		t.Fatal("expected static argument resolver for all literal arguments")
	}
	if node.strInit != "" || node.strNoexprInit != "" {
		t.Fatal("expected non-required arguments to be zero during Init")
	}
	if node.strRequiredInit != "program" {
		t.Fatal("expected required argument to be set")
	}
	subject := event.NewEvent(context.Background(), nil, map[string]interface{}{"program": "result"})
	resolver.Process(subject)
	if node.str != "" || node.strNoexpr != "" {
		t.Fatal("expected non-required argument to be zero during process")
	}
	if node.strRequired != "program" {
		t.Fatal("expected required argument to be set during process")
	}
}

func TestArgumentResolverDynamicExpression(t *testing.T) {
	node := &testProcessNode{}
	values := map[string]ValueNode{}
	var err error
	values["required"], err = newCelProgramNode("event.program")
	if err != nil {
		t.Fatalf("expected celNode program success, received %s", err)
	}
	resolver, err := newArgumentResolverNode(node, values)
	if err != nil {
		t.Fatalf("expected success, received %s", err)
	}
	if len(resolver.(*argumentResolverNode).resolvers) != 1 {
		t.Fatal("expected single resolver for dynamic expression argument")
	}
	if node.strInit != "" || node.strNoexprInit != "" || node.strRequiredInit != "" {
		t.Fatal("expected all arguments to be zero during Init")
	}
	subject := event.NewEvent(context.Background(), nil, map[string]interface{}{"program": "result"})
	resolver.Process(subject)
	if node.str != "" || node.strNoexpr != "" {
		t.Fatal("expected non-required argument to be zero during process")
	}
	if node.strRequired != "result" {
		t.Fatal("expected required argument to be set during process")
	}
}

func TestArgumentResolverDisallowedExpression(t *testing.T) {
	node := &testProcessNode{}
	values := map[string]ValueNode{"required": &literalNode{value: types.String("value")}}
	var err error
	values["noexpr"], err = newCelProgramNode("event.program")
	if err != nil {
		t.Fatalf("expected celNode program success, received %s", err)
	}
	_, err = newArgumentResolverNode(node, values)
	if err == nil || err.Error() != "argument noexpr must be a literal and cannot be dynamic" {
		t.Fatalf("expected expression not allowed as argument error, received %s", err)
	}
}

func TestArgumentResolverUnknown(t *testing.T) {
	node := &testProcessNode{}
	values := map[string]ValueNode{
		"required": &literalNode{value: types.String("value1")},
		"random":   &literalNode{value: types.String("value2")},
	}
	_, err := newArgumentResolverNode(node, values)
	if err == nil || err.Error() != "unknown arguments: random" {
		t.Fatalf("expected expression not allowed as argument error, received %s", err)
	}
}

func TestArgumentResolverInitError(t *testing.T) {
	failure := fmt.Errorf("failure")
	node := &testProcessNode{}
	node.shouldInitErr = failure
	values := map[string]ValueNode{"required": &literalNode{value: types.String("value")}}
	_, err := newArgumentResolverNode(node, values)
	if err != failure {
		t.Fatalf("expected Init error to be returned, received %s", err)
	}
}
