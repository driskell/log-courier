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

package event

import (
	"encoding/json"
	"sort"
)

// Tags is used for the "tags" key of all events
// It aids with the addition and removal of tags during processing
type Tags map[string]struct{}

// Add a tag
func (e Tags) Add(tag string) {
	e[tag] = struct{}{}
}

// Remove a tag
func (e Tags) Remove(tag string) {
	if _, ok := e[tag]; ok {
		delete(e, tag)
	}
}

// MarshalJSON encodes the event tags as a string array
func (e Tags) MarshalJSON() ([]byte, error) {
	keys := make([]string, 0, len(e))
	for tag := range e {
		keys = append(keys, tag)
	}
	sort.Strings(keys)
	return json.Marshal(keys)
}
