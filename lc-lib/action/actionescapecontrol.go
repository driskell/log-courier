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

package action

import (
	"fmt"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
)

type escapeControlNode struct {
}

var _ ast.ProcessArgumentsNode = &escapeControlNode{}

func newEscapeControlNode(*config.Config) (ast.ProcessArgumentsNode, error) {
	return &escapeControlNode{}, nil
}

func (n *escapeControlNode) Arguments() []ast.Argument {
	return []ast.Argument{
		ast.NewArgumentListString("fields", ast.ArgumentOptional),
	}
}

func (n *escapeControlNode) Init([]any) error {
	return nil
}

func (n *escapeControlNode) ProcessWithArguments(subject *event.Event, arguments []any) *event.Event {
	fields, _ := arguments[0].([]string)
	if len(fields) == 0 {
		subject.EscapeControlChars()
		return subject
	}

	for _, field := range fields {
		value, err := subject.Resolve(field, nil)
		if err != nil {
			subject.AddError("escape_control", fmt.Sprintf("Field '%s' could not be resolved: %s", field, err))
			continue
		}
		escaped, ok := event.EscapeControlValue(value)
		if !ok {
			continue
		}
		if _, isString := value.(string); isString {
			subject.MustResolve(field, escaped)
		}
		subject.TagControlChars(field)
	}
	return subject
}

// init will register the action
func init() {
	ast.RegisterAction("escape_control", newEscapeControlNode)
}
