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
	data := event.Data()
	if entry, ok := data[d.Field]; ok {
		if value, ok := entry.(string); ok {
			for _, layout := range d.Formats {
				result, err := time.Parse(layout, value)
				if err != nil {
					continue
				}
				data["@timestamp"] = result
				return event
			}
		} else {
			data["_date_error"] = fmt.Sprintf("Field '%s' for date parse is not a string", d.Field)
		}
	} else {
		data["_date_error"] = fmt.Sprintf("Field '%s' is not present", d.Field)
	}
	event.AddTag("_date_failure")
	return event
}
