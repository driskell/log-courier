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
	"encoding/json"
	"fmt"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
)

type jsonNode struct {
}

var _ ast.ProcessArgumentsNode = &jsonNode{}

func newJsonNode(*config.Config) (ast.ProcessArgumentsNode, error) {
	return &jsonNode{}, nil
}

func (n *jsonNode) Arguments() []ast.Argument {
	return []ast.Argument{
		ast.NewArgumentString("field", ast.ArgumentRequired),
		ast.NewArgumentBool("remove", ast.ArgumentOptional),
	}
}

func (n *jsonNode) Init([]any) error {
	return nil
}

func (n *jsonNode) ProcessWithArguments(subject *event.Event, arguments []any) *event.Event {
	field := arguments[0].(string)
	remove := false
	if arguments[1] != nil {
		remove = arguments[1].(bool)
	}

	entry, err := subject.Resolve(field, nil)
	if err != nil {
		subject.AddError("json", fmt.Sprintf("Field '%s' failed to resolve: %s", field, err))
		return subject
	}

	var (
		value string
		ok    bool
	)
	if value, ok = entry.(string); !ok {
		subject.AddError("json", fmt.Sprintf("Field '%s' is not present or not a string", field))
		return subject
	}

	var v map[string]interface{}
	err = json.Unmarshal([]byte(value), &v)
	if err != nil {
		subject.AddError("json", fmt.Sprintf("Decode of field '%s' failed: %s", field, err.Error()))
		return subject
	}
	for name, value := range v {
		_, err = subject.Resolve(name, value)
		if err != nil {
			subject.AddError("json", fmt.Sprintf("Decode of field '%s' failed: %s", field, err.Error()))
			return subject
		}
	}
	if remove {
		_, err := subject.Resolve(field, event.ResolveParamUnset)
		if err != nil {
			subject.AddError("json", fmt.Sprintf("Failed to remove field '%s': %s", field, err))
		}
	}
	return subject
}

// init will register the action
func init() {
	ast.RegisterAction("json", newJsonNode)
}
