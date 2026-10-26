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
)

const controlCharsTag = "_control_chars"

type escapeControlAction struct {
	Fields []string `config:"fields"`
}

func newEscapeControlAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (ASTEntry, error) {
	action := &escapeControlAction{}
	if err := p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	return action, nil
}

func (e *escapeControlAction) Process(evnt *event.Event) *event.Event {
	if len(e.Fields) == 0 {
		return escapeControlEvent(evnt)
	}

	for _, field := range e.Fields {
		value, err := evnt.Resolve(field, nil)
		if err != nil {
			evnt.AddError("escape_control", fmt.Sprintf("Field '%s' could not be resolved: %s", field, err))
			continue
		}
		escaped, ok := event.EscapeControlValue(value)
		if !ok {
			continue
		}
		if _, isString := value.(string); isString {
			evnt.MustResolve(field, escaped)
		}
		tagControlChars(evnt, field)
	}
	return evnt
}

// escapeControlEvent escapes control characters throughout the event, tagging each top-level field that had any
func escapeControlEvent(evnt *event.Event) *event.Event {
	data := evnt.Data()
	for key, value := range data {
		escapedKey, keyFound := event.EscapeControl(key)
		escaped, valueFound := event.EscapeControlValue(value)
		if !keyFound && !valueFound {
			continue
		}
		if keyFound {
			delete(data, key)
		}
		data[escapedKey] = escaped
		tagControlChars(evnt, escapedKey)
	}
	return evnt
}

func tagControlChars(evnt *event.Event, field string) {
	fieldTag, _ := event.EscapeControl(field)
	evnt.AddTag(controlCharsTag)
	evnt.AddTag(controlCharsTag + "_" + fieldTag)
}

// init will register the action
func init() {
	RegisterAction("escape_control", newEscapeControlAction)
}
