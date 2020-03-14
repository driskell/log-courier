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
	"strconv"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
)

type dateAction struct {
	Field   string   `config:"field"`
	Remove  bool     `config:"remove"`
	Formats []string `config:"formats"`
}

func newDateAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (ASTEntry, error) {
	var err error
	action := &dateAction{}
	if err = p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	return action, nil
}

func (d *dateAction) Process(evnt *event.Event) *event.Event {
	entry, err := evnt.Resolve(d.Field, nil)
	if err != nil {
		evnt.AddError("date", fmt.Sprintf("Field '%s' could not be resolved: %s", d.Field, err))
		return evnt
	}

	var (
		value string
		ok    bool
	)
	value, ok = entry.(string)
	if !ok {
		evnt.AddError("date", fmt.Sprintf("Field '%s' is not present or not a string", d.Field))
		return evnt
	}

	for _, layout := range d.Formats {
		var (
			result time.Time
			err    error
		)

		switch layout {
		case "UNIX":
			unix, err := strconv.Atoi(layout)
			if err != nil {
				continue
			}
			result = time.Unix(int64(unix), 0)
		default:
			result, err = time.Parse(layout, value)
			if err != nil {
				continue
			}
		}

		// If year 0, we only parsed month/day etc.
		// We do not support parsing of dates without the current date
		// For that, we would likely have a flag to say only time parsed, so we can explicitly set the date
		if result.Year() == 0 {
			result = time.Date(time.Now().Year(), result.Month(), result.Day(), result.Hour(), result.Minute(), result.Second(), result.Nanosecond(), result.Location())
		}

		evnt.MustResolve("@timestamp", result)
		if d.Remove {
			_, err := evnt.Resolve(d.Field, event.ResolveParamUnset)
			if err != nil {
				evnt.AddError("date", fmt.Sprintf("Failed to remove field '%s': %s", d.Field, err))
			}
		}
		return evnt
	}

	evnt.AddError("date", fmt.Sprintf("Field '%s' could not be parsed with any of the given formats", d.Field))
	return evnt
}

// init will register the action
func init() {
	RegisterAction("date", newDateAction)
}
