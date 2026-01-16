package ast

import (
	"fmt"
	"reflect"
	"strings"
)

// UpgradeScript traverses the AST from the root node and returns a string script
// following the syntax expected in testing/log-carver-script.yaml.
func UpgradeScript(root interface{}) string {
	return renderNode(root, 0)
}

func renderNode(node interface{}, indent int) string {
	if node == nil {
		return ""
	}
	ind := strings.Repeat("  ", indent)
	switch n := node.(type) {
	case *ifNode:
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%sif (%s) {\n", ind, renderValueNode(n.condition)))
		for _, entry := range n.block {
			sb.WriteString(renderNode(entry, indent+1))
		}
		for _, elseif := range n.elseIfExpr {
			sb.WriteString(fmt.Sprintf("%s} else if (%s) {\n", ind, renderValueNode(elseif.condition)))
			for _, entry := range elseif.block {
				sb.WriteString(renderNode(entry, indent+1))
			}
		}
		if n.elseExpr != nil {
			sb.WriteString(fmt.Sprintf("%s} else {\n", ind))
			for _, entry := range n.elseExpr.block {
				sb.WriteString(renderNode(entry, indent+1))
			}
		}
		sb.WriteString(fmt.Sprintf("%s}\n", ind))
		return sb.String()
	case *setNode:
		return fmt.Sprintf("%sset %s = %s;\n", ind, n.identifier, renderValueNode(n.value))
	case *unsetNode:
		return fmt.Sprintf("%sunset %s;\n", ind, n.identifier)
	case *noopNode:
		return ""
	case *staticArgumentResolverNode:
		return renderActionNode(n.processNode, n.values, ind)
	case *argumentResolverNode:
		return renderActionNode(n.processNode, n.values, ind)
	case *literalNode:
		return fmt.Sprintf("%v", n.value)
	case *celProgramNode:
		return n.source
	case []ProcessNode:
		var sb strings.Builder
		for _, entry := range n {
			sb.WriteString(renderNode(entry, indent))
		}
		return sb.String()
	default:
		return fmt.Sprintf("%s# unknown node type: %T\n", ind, node)
	}
}

func renderValueNode(node interface{}) string {
	switch n := node.(type) {
	case *literalNode:
		return fmt.Sprintf("%v", n.value)
	case *celProgramNode:
		return n.source
	default:
		return fmt.Sprintf("%v", n)
	}
}

func renderActionNode(processNode ProcessArgumentsNode, values []any, ind string) string {
	name := actionNodeName(processNode)
	args := make([]string, 0, len(values))
	for i, arg := range processNode.Arguments() {
		if values[i] != nil {
			args = append(args, fmt.Sprintf("%s=%v", arg.Name(), values[i]))
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf("%s%s %s;\n", ind, name, strings.Join(args, ", "))
	}
	return fmt.Sprintf("%s%s;\n", ind, name)
}

func actionNodeName(processNode ProcessArgumentsNode) string {
	t := reflect.TypeOf(processNode)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return strings.TrimSuffix(t.Name(), "Node")
}
