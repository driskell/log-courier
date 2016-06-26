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

package api

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

// Null represents a null value
var Null = apiNull{}

// Number represents an integer number in the API
type Number int64

// HumanReadable returns the Number as a string
func (n Number) HumanReadable(string) ([]byte, error) {
	return []byte(strconv.FormatInt(int64(n), 10)), nil
}

// Float represents a floating point number in the API
type Float float64

// HumanReadable returns the Float as a string
func (f Float) HumanReadable(string) ([]byte, error) {
	return []byte(strconv.FormatFloat(float64(f), 'g', -1, 64)), nil
}

// String represents a string in the API
type String string

// HumanReadable returns the String as a string
func (s String) HumanReadable(string) ([]byte, error) {
	return []byte(s), nil
}
