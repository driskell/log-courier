/*
 * Copyright 2012-2020 Jason Woods and contributors
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

package event

import (
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
)

const (
	defaultStreamAddHostField     bool = true
	defaultStreamAddTimezoneField bool = false
	defaultEnableECS              bool = false
)

// StreamConfig holds the configuration for a log stream
type StreamConfig struct {
	AddHostField     bool                   `config:"add host field"`
	AddTimezoneField bool                   `config:"add timezone field"`
	EnableECS        bool                   `config:"enable ecs"`
	Fields           map[string]interface{} `config:"fields"`

	genConfig *config.General
	timezone  string
}

// Defaults initialises the default configuration for a log stream
func (sc *StreamConfig) Defaults() {
	sc.AddHostField = defaultStreamAddHostField
	sc.AddTimezoneField = defaultStreamAddTimezoneField
	sc.EnableECS = defaultEnableECS

	sc.timezone = time.Now().Format("-0700 MST")
}

// Validate validates the stream configuration and also stores a copy of the
// root configuration so we can access global fields etc
func (sc *StreamConfig) Validate(p *config.Parser, path string) (err error) {
	sc.genConfig = p.Config().General()

	// Ensure all Fields are map[string]interface{}
	if err = p.FixMapKeys(path+"/fields", sc.Fields); err != nil {
		return
	}

	return nil
}

// Decorate applies all transformations necessary from the stream configuration
// to the data that will eventually become an event
func (sc *StreamConfig) Decorate(data map[string]interface{}) map[string]interface{} {
	data["@timestamp"] = time.Now()
	if sc.AddHostField {
		if sc.EnableECS {
			data["host"] = map[string]interface{}{
				"name":     sc.genConfig.Host,
				"hostname": sc.genConfig.Host,
			}
		} else {
			data["host"] = sc.genConfig.Host
		}
	}

	if sc.AddTimezoneField {
		if sc.EnableECS {
			var (
				eventEntry map[string]interface{}
				ok         bool
			)
			if eventEntry, ok = data["event"].(map[string]interface{}); !ok {
				eventEntry = map[string]interface{}{}
				data["event"] = eventEntry
			}
			eventEntry["timezone"] = sc.timezone
		} else {
			data["timezone"] = sc.timezone
		}
	}

	for k := range sc.genConfig.GlobalFields {
		data[k] = sc.genConfig.GlobalFields[k]
	}

	for k := range sc.Fields {
		data[k] = sc.Fields[k]
	}

	return data
}
