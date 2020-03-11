/*
 * Copyright 2014-2016 Jason Woods.
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
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
)

// Action takes an event and returns an event, having potentially mutated or replaced it
type Action interface {
	Process(event *event.Event) *event.Event
}

// ActionRegistrarFunc is a callback that validates the configuration for
// an action that was registered via RegisterAction
type ActionRegistrarFunc func(*config.Parser, string, map[string]interface{}, string) (Action, error)

var registeredActions = make(map[string]ActionRegistrarFunc)

// RegisterAction registers an action with the configuration module by providing a
// callback that can be used to validate the configuration
func RegisterAction(name string, registrarFunc ActionRegistrarFunc) {
	registeredActions[name] = registrarFunc
}

// AvailableActions returns the list of registered actions available for use
func AvailableActions() (ret []string) {
	ret = make([]string, 0, len(registeredActions))
	for k := range registeredActions {
		ret = append(ret, k)
	}
	return
}
