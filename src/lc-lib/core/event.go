/*
* Copyright 2014 Jason Woods.
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

package core

type Event map[string]interface{}

type EventDescriptor struct {
	Stream interface{}
	Offset int64
	Event  Event
}

func NewEvent(fields map[string]interface{}, file string, offset int64, line uint64, message string) Event {
	event := Event{
		"file":    file,
		"offset":  offset,
		"message": message,
	}
	for k := range fields {
		event[k] = fields[k]
	}
	return event
}
