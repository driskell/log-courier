package ast

import (
	"fmt"
	"strings"

	"github.com/driskell/log-courier/lc-lib/event"
	"golang.org/x/exp/slices"
)

type argumentResolver struct {
	argument Argument
	value    ValueNode
	index    int
}

func newArgumentResolverNode(processNode ProcessArgumentsNode, valueNodes map[string]ValueNode) (ProcessNode, error) {
	var missing []string
	var unknown []string
	var found []string
	var resolvers []*argumentResolver
	arguments := processNode.Arguments()
	index := 0
	values := make([]any, len(arguments))
	for _, argument := range arguments {
		valueNode, ok := valueNodes[argument.Name()]
		if !ok {
			if argument.IsRequired() {
				missing = append(missing, argument.Name())
			}
			index++
			continue
		}
		switch typedValue := valueNode.(type) {
		case *literalNode:
			result, err := argument.Resolve(typedValue.value)
			if err != nil {
				return nil, fmt.Errorf("argument %s is invalid: %w", argument.Name(), err)
			}
			values[index] = result
		default:
			if argument.IsExprDisallowed() {
				return nil, fmt.Errorf("argument %s must be a literal and cannot be dynamic", argument.Name())
			}
			resolvers = append(resolvers, &argumentResolver{
				argument,
				valueNode,
				index,
			})
		}
		found = append(found, argument.Name())
		index++
	}
	if len(missing) != 0 {
		return nil, fmt.Errorf("missing required arguments: %s", strings.Join(missing, ", "))
	}
	for name := range valueNodes {
		if !slices.Contains(found, name) {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) != 0 {
		return nil, fmt.Errorf("unknown arguments: %s", strings.Join(unknown, ", "))
	}
	// Call Init to allow initialisation
	if err := processNode.Init(values); err != nil {
		return nil, err
	}
	// If there are no expression arguments, return a static node that fast calls with fixed arguments
	if len(resolvers) == 0 {
		return &staticArgumentResolverNode{processNode, values}, nil
	}
	// Return the resolver node, which will resolve arguments on each call
	return &argumentResolverNode{processNode, values, resolvers}, nil
}

type staticArgumentResolverNode struct {
	processNode ProcessArgumentsNode
	values      []any
}

func (n *staticArgumentResolverNode) Process(subject *event.Event) *event.Event {
	return n.processNode.ProcessWithArguments(subject, n.values)
}

type argumentResolverNode struct {
	processNode ProcessArgumentsNode
	values      []any
	resolvers   []*argumentResolver
}

func (n *argumentResolverNode) Process(subject *event.Event) *event.Event {
	values := make([]any, len(n.values))
	copy(values, n.values)
	for _, resolver := range n.resolvers {
		result, err := resolver.argument.Resolve(resolver.value.Value(subject))
		if err != nil {
			// TODO: Error handling?
			return subject
		}
		values[resolver.index] = result
	}
	return n.processNode.ProcessWithArguments(subject, values)
}
