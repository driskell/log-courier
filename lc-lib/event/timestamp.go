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
	"errors"
	"fmt"
	"time"
)

// Timestamp is used for the "@timestamp" entry of all events
type Timestamp time.Time

// VerifySetEnter checks if we can set the given key (if we're a map for example)
func (e Timestamp) VerifySetEnter(string) error {
	return errors.New("Builtin @timestamp is not a map")
}

// VerifySet checks if we can be set to the given value
func (e Timestamp) VerifySet(set interface{}) (interface{}, error) {
	switch value := set.(type) {
	case time.Time:
		return Timestamp(value), nil
	}
	return nil, fmt.Errorf("Cannot set builtin @timestamp key to non time value")
}

// Format the timestamp
func (e Timestamp) Format(layout string) string {
	return time.Time(e).Format(layout)
}

// MarshalJSON encodes the timestamp
func (e Timestamp) MarshalJSON() ([]byte, error) {
	return time.Time(e).MarshalJSON()
}
