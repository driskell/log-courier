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

package receiver

import "errors"

// MetadataReceiver is used for the "@metadata[receiver]" entry of events
type MetadataReceiver map[string]interface{}

// VerifySetEnter checks if we can set the given key (if we're a map for example)
func (e MetadataReceiver) VerifySetEnter(string) (map[string]interface{}, error) {
	return nil, errors.New("metadata at @metadata[receiver] is read-only")
}

// VerifySet checks if we can be set to the given value
func (e MetadataReceiver) VerifySet(interface{}) (interface{}, error) {
	return nil, errors.New("metadata at @metadata[receiver] is read-only")
}
