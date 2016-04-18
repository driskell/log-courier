/*
 * Copyright 2014-2015 Jason Woods.
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

package admin

import (
	"encoding/json"
	"strconv"
)

type apiNull struct{}

// MarshalJSON returns the apiNull in JSON form
func (n apiNull) MarshalJSON() ([]byte, error) {
	return json.Marshal(nil)
}

// HumanReadable returns the apiNull as a string
func (n apiNull) HumanReadable(string) ([]byte, error) {
	return []byte("n/a"), nil
}

// APINull represents a null value
var APINull = apiNull{}

// APINumber represents an integer number in the API
type APINumber int64

// HumanReadable returns the APINumber as a string
func (n APINumber) HumanReadable(string) ([]byte, error) {
	return []byte(strconv.FormatInt(int64(n), 10)), nil
}

// APIFloat represents a floating point number in the API
type APIFloat float64

// HumanReadable returns the APIFloat as a string
func (f APIFloat) HumanReadable(string) ([]byte, error) {
	return []byte(strconv.FormatFloat(float64(f), 'g', -1, 64)), nil
}

// APIString represents a string in the API
type APIString string

// HumanReadable returns the APIString as a string
func (s APIString) HumanReadable(string) ([]byte, error) {
	return []byte(s), nil
}
