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

import "errors"

// Metadata is used for the "@metadata" entry of all events
type Metadata map[string]interface{}

// VerifySetEnter checks if we can set the given key (if we're a map for example)
func (e Metadata) VerifySetEnter(string) error {
	return nil
}

// VerifySet checks if we can be set to the given value
func (e Metadata) VerifySet(interface{}) (interface{}, error) {
	return nil, errors.New("Cannot set @metadata directly")
}
