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
	"encoding/json"
	"fmt"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
)

type jsonAction struct {
	Field  string `config:"field"`
	Remove bool   `config:"remove"`
}

func newJsonAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (ASTEntry, error) {
	var err error
	action := &jsonAction{}
	if err = p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	return action, nil
}

func (g *jsonAction) Validate(p *config.Parser, configPath string) error {
	return nil
}

func (g *jsonAction) Process(evnt *event.Event) *event.Event {
	entry, err := evnt.Resolve(g.Field, nil)
	if err != nil {
		evnt.AddError("json", fmt.Sprintf("Field '%s' failed to resolve: %s", g.Field, err))
		return evnt
	}

	var (
		value string
		ok    bool
	)
	if value, ok = entry.(string); !ok {
		evnt.AddError("json", fmt.Sprintf("Field '%s' is not present or not a string", g.Field))
		return evnt
	}

	var v map[string]interface{}
	err = json.Unmarshal([]byte(value), &v)
	if err != nil {
		evnt.AddError("json", fmt.Sprintf("Decode of field '%s' failed: %s", g.Field, err.Error()))
		return evnt
	}
	for name, value := range v {
		_, err = evnt.Resolve(name, value)
		if err != nil {
			evnt.AddError("json", fmt.Sprintf("Decode of field '%s' failed: %s", g.Field, err.Error()))
			return evnt
		}
	}
	if g.Remove {
		_, err := evnt.Resolve(g.Field, event.ResolveParamUnset)
		if err != nil {
			evnt.AddError("json", fmt.Sprintf("Failed to remove field '%s': %s", g.Field, err))
		}
	}
	return evnt
}

// init will register the action
func init() {
	RegisterAction("json", newJsonAction)
}
