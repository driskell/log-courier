/*
 * Copyright 2012-2020 Jason Woods and contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ast

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/processor/expr"
)

type celProgramNode struct {
	source  string
	program cel.Program
}

var _ ValueNode = &celProgramNode{}

// newCelProgramNode parses an expression using cel-go and returns a node
func newCelProgramNode(expression string) (ValueNode, error) {
	program, err := expr.ParseExpression(expression)
	if err != nil {
		return nil, err
	}

	// Optimise code by mapping to a literal value if this expression does not access any input
	result, _, err := program.Eval(map[string]interface{}{})
	if err == nil {
		return newLiteralNode(result), nil
	}

	return &celProgramNode{
		source:  expression,
		program: program,
	}, nil
}

func (n *celProgramNode) Value(subject *event.Event) ref.Val {
	val, _, err := n.program.Eval(map[string]interface{}{"event": subject.Data()})
	if err != nil {
		log.Warningf("Failed to evaluate expression: [%s] -> %s", n.source, err)
		return nil
	}
	if types.IsUnknown(val) {
		log.Warningf("Expression evaluated to unknown value, treating as null: [%s]", n.source)
		return nil
	}
	return val
}
