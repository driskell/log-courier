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
	"bytes"
	"encoding/json"
	"runtime/pprof"
)

// apiDebug is an entry that provides debugging information
type apiDebug struct{}

// MarshalJSON returns the APICallbackEntry in JSON form
func (c *apiDebug) MarshalJSON() ([]byte, error) {
	stack, err := c.GoRoutineBytes()
	if err != nil {
		stack = []byte(err.Error())
	}

	return json.Marshal(struct {
		Stack string
	}{
		Stack: string(stack),
	})
}

// HumanReadable returns the APICallbackEntry as a string
func (c *apiDebug) HumanReadable(indent string) ([]byte, error) {
	return c.GoRoutineBytes()
}

// GoRoutineBytes returns a byte slice containing the goroutine profile, or an
// error message if an error occurred
func (c *apiDebug) GoRoutineBytes() ([]byte, error) {
	log.Warning("Generating pprof goroutine profile for debug API call")

	goroutine := pprof.Lookup("goroutine")
	buffer := new(bytes.Buffer)

	// debug=2 so we print same trace we get when we panic
	if err := goroutine.WriteTo(buffer, 2); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
