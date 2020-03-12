/*
 * Copyright 2014-2015 Jason Woods.
 *
 * This file is a modification of code from Logstash Forwarder.
 * Copyright 2012-2013 Jordan Sissel and contributors.
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

func newSetFieldAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (Action, error) {
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
		event.Resolve("_set_field_error", fmt.Sprintf("Failed to evaluate set_field value_expr: [%s] -> %s", f.ValueExpr, err))
		event.AddTag("_set_field_failure")
		return event
	}
	if types.IsUnknown(val) {
		event.Resolve("_set_field_error", fmt.Sprintf("Evaluation of set_field returned unknown: [%s]", f.ValueExpr))
		event.AddTag("_set_field_failure")
		return event
	}
	event.Resolve(f.Field, val.Value())
	return event
}

type unsetFieldAction struct {
	Field string `config:"field"`
}

func newUnsetFieldAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (Action, error) {
	action := &unsetFieldAction{}
	if err := p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	return action, nil
}

func (f *unsetFieldAction) Process(evnt *event.Event) *event.Event {
	if _, err := evnt.Resolve(f.Field, event.ResolveParamUnset); err != nil {
		evnt.AddTag("_unset_field_failure")
		evnt.Resolve("_unset_field_error", fmt.Sprintf("Failed to unset field: %s", err))
	}
	return evnt
}
