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
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
)

type dateAction struct {
	Field   string   `config:"field"`
	Formats []string `config:"formats"`
}

func newDateAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (Action, error) {
	var err error
	action := &dateAction{}
	if err = p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	return action, nil
}

func (d *dateAction) Process(event *event.Event) *event.Event {
	entry, err := event.Resolve(d.Field, nil)
	if err != nil {
		event.AddError("date", fmt.Sprintf("Field '%s' could not be resolved: %s", d.Field, err))
		return event
	}

	var (
		value string
		ok    bool
	)
	value, ok = entry.(string)
	if !ok {
		event.AddError("date", fmt.Sprintf("Field '%s' is not present or not a string", d.Field))
		return event
	}

	for _, layout := range d.Formats {
		result, err := time.Parse(layout, value)
		if err != nil {
			continue
		}
		event.MustResolve("@timestamp", result)
		return event
	}

	event.AddError("date", fmt.Sprintf("Field '%s' could not be parsed with any of the given formats", d.Field))
	return event
}

// init will register the action
func init() {
	RegisterAction("date", newDateAction)
}
