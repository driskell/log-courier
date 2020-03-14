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

package processor

import (
	"fmt"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
)

type setFieldAction struct {
	Field     string `config:"field"`
	ValueExpr string `config:"value_expr"`

	valueProgram cel.Program
}

func newSetFieldAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (ASTEntry, error) {
	var err error
	action := &setFieldAction{}
	if err = p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	action.valueProgram, err = ParseExpression(action.ValueExpr)
	if err != nil {
		return nil, err
	}
	return action, nil
}

func (f *setFieldAction) Process(event *event.Event) *event.Event {
	val, _, err := f.valueProgram.Eval(map[string]interface{}{"event": event})
	if err != nil {
		event.AddError("set_field", fmt.Sprintf("Failed to evaluate: [%s] -> %s", f.ValueExpr, err))
		return event
	}
	if types.IsUnknown(val) {
		event.AddError("set_field", fmt.Sprintf("Evaluation returned unknown: [%s]", f.ValueExpr))
		return event
	}
	_, err = event.Resolve(f.Field, val.Value())
	if err != nil {
		event.AddError("set_field", fmt.Sprintf("Failed to set field: %s", err))
	}
	return event
}

type unsetFieldAction struct {
	Field string `config:"field"`
}

func newUnsetFieldAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (ASTEntry, error) {
	action := &unsetFieldAction{}
	if err := p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	return action, nil
}

func (f *unsetFieldAction) Process(evnt *event.Event) *event.Event {
	if _, err := evnt.Resolve(f.Field, event.ResolveParamUnset); err != nil {
		evnt.AddError("unset_field", fmt.Sprintf("Failed to unset field: %s", err))
	}
	return evnt
}

// init will register the action
func init() {
	RegisterAction("set_field", newSetFieldAction)
	RegisterAction("unset_field", newUnsetFieldAction)
}
