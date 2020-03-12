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

package event

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// ResolveParam is a special set value you can pass to event.Resolve
type ResolveParam int

const (
	// ResolveParamUnset will unset the value at a path given to event.Resolve
	ResolveParamUnset ResolveParam = iota
)

var (
	keyMatcher = regexp.MustCompile(`^([^\[\]]+)|\[([^\[\]]+)\]`)
)

// Acknowledger is something that can be called with events once they ar acknowledged
type Acknowledger interface {
	Acknowledge([]*Event)
}

// Event describes an event
type Event struct {
	ctx   context.Context
	acker Acknowledger
	data  map[string]interface{}

	encoded []byte
}

// NewEvent creates a new event structure from the given data
func NewEvent(ctx context.Context, acker Acknowledger, data map[string]interface{}) *Event {
	ret := &Event{
		ctx:   ctx,
		acker: acker,
		data:  data,
	}
	ret.convertData()
	return ret
}

// NewEventFromBytes creates a new event structure from the given data
func NewEventFromBytes(ctx context.Context, acker Acknowledger, data []byte) *Event {
	return &Event{
		ctx:     ctx,
		acker:   acker,
		encoded: data,
	}
}

// Data returns the internal event data for reading or mutation
// The return data must NOT be mutated
func (e *Event) Data() map[string]interface{} {
	if e.data == nil {
		err := json.Unmarshal(e.encoded, &e.data)
		if err != nil {
			e.data = make(map[string]interface{})
			e.data["message"] = err.Error()
			e.data["tags"] = &Tags{"_unmarshal_failure": struct{}{}}
			e.data["@timestamp"] = time.Now()
		} else {
			e.convertData()
		}
	}
	return e.data
}

// convertData is the internal function that enforces guaranteed types
func (e *Event) convertData() {
	// Resolve tags
	if entry, ok := e.data["tags"]; ok {
		switch value := entry.(type) {
		case Tags:
		case string:
			e.data["tags"] = Tags{value: struct{}{}}
		case []string:
			tags := Tags{}
			for _, tag := range value {
				tags.Add(tag)
			}
			e.data["tags"] = tags
		default:
			e.data["tags"] = Tags{"_tags_parse_failure": struct{}{}}
			e.data["tags_parse_error"] = fmt.Sprintf("tags was not a string or string list, was %T", value)
		}
	} else {
		e.data["tags"] = Tags{}
	}
	// Resolve "@timestamp" to a time.Time
	if entry, ok := e.data["@timestamp"]; ok {
		switch value := entry.(type) {
		case time.Time:
		case string:
			parsed, err := time.Parse(time.RFC3339, value)
			if err != nil {
				e.data["@timestamp"] = time.Now()
				e.data["timestamp_parse_error"] = err
				e.AddTag("_timestamp_parse_failure")
			} else {
				e.data["@timestamp"] = parsed
			}
		default:
			e.data["@timestamp"] = time.Now()
			e.data["timestamp_parse_error"] = fmt.Sprintf("@timestamp was not a string, was %T", value)
			e.AddTag("_timestamp_parse_failure")
		}
	} else {
		e.data["@timestamp"] = time.Now()
	}
}

// MustResolve is the same as Resolve but will panic if an error occurs
//
// This should never be used with user input, and only used with hard-coded field names that are guaranteed to be correct syntax
func (e *Event) MustResolve(path string, set interface{}) interface{} {
	output, err := e.Resolve(path, set)
	if err != nil {
		panic(err)
	}
	return output
}

// Resolve will return (and optionally set) the value at the given path (field), using a[b][c] syntax
//
// It will return an error if the path is invalid, and return nil if the path does not exist
//
// If you specify non-nil in the set parameter - the path will be set to that value and
// the old value will be returned - and any non-map parts of the path will be converted to empty maps
//
// Remember ClearCache is required to flush any cached representations if you set or unset a value
//
// The following paths are guaranteed to exist and be of a specific type:
//     @timestamp: time.Time
//         If no value currently exists, it is created with the current time.
//         If the existing value is invalid, the _timestamp_parse_failure tag is added
//         and the error is stored inside the timestamp_parse_error field
//     tags: Tags (Empty Tags instance)
//         If no value currently exists, it is created empty with no tags.
//         If the existing value is invalid, the _tags_parse_failure tag is added
//         and the error is stored inside the tags_parse_error field
//
// If any error occured during unmarshaling of an event received over the network
// then a new event with only a message key (and the above defaults) will be created
// where the message is the error message, and the "_unmarshal_failure" tag is added
func (e *Event) Resolve(path string, set interface{}) (output interface{}, err error) {
	e.validateMutation(path, set)
	currentMap := e.data
	lastIndex := 0
	results := keyMatcher.FindAllStringSubmatchIndex(path, -1)
	for j := 0; j < len(results); j++ {
		// Must always join together
		if results[j][0] != lastIndex {
			return nil, fmt.Errorf("Invalid field: %s", path)
		}
		lastIndex = results[j][1]
		nameStart, nameEnd := results[j][2], results[j][3]
		if nameStart < 0 {
			nameStart, nameEnd = results[j][4], results[j][5]
		}
		name := path[nameStart:nameEnd]
		if j == len(results)-1 {
			// Last item, so will always be a value
			if value, ok := currentMap[name]; ok {
				output = value
			}
			if set != nil {
				if set == ResolveParamUnset {
					delete(currentMap, name)
				} else {
					currentMap[name] = set
				}
			}
		} else {
			// Calculate next inner
			// Can't use non-map, ignore as if it didn't exist, and keep validating
			// (However, if we are setting, overwrite it with empty map)
			if value, ok := currentMap[name].(map[string]interface{}); ok {
				currentMap = value
			} else if set != nil {
				// Convert path to empty map
				newMap := map[string]interface{}{}
				currentMap[name] = newMap
				currentMap = newMap
			} else {
				// Doesn't exist so there's no value but keep validating
				currentMap = nil
			}
		}
	}
	if lastIndex != len(path) {
		return nil, fmt.Errorf("Invalid field: %s", path)
	}
	return
}

// validateMutation ensures we do not incorrectly modify or change builtin keys
func (e *Event) validateMutation(path string, set interface{}) error {
	switch path {
	case "@timestamp":
		switch value := set.(type) {
		case time.Time:
		case ResolveParam:
			if value == ResolveParamUnset {
				return fmt.Errorf("Removal of @timestamp key is not allowed as it is builtin")
			}
		default:
			return fmt.Errorf("Cannot set builtin @timestamp key to non time value")
		}
	case "tags":
		switch value := set.(type) {
		case []string:
		case ResolveParam:
			if value == ResolveParamUnset {
				return fmt.Errorf("Removal of tags key is not allowed as it is builtin")
			}
		default:
			return fmt.Errorf("Cannot set builtin tags key to non string list value")
		}
	}
	return nil
}

// AddError adds an error tag and an error message field for a specific action that has failed
func (e *Event) AddError(action string, message string) {
	e.data[fmt.Sprintf("_%s_error", action)] = message
	e.AddTag(fmt.Sprintf("_%s_failure", action))
}

// AddTag adds a tag to the event
// Remember ClearCache is required to flush any cached representations
func (e *Event) AddTag(tag string) {
	e.data["tags"].(Tags).Add(tag)
}

// RemoveTag adds a tag to the event
// Remember ClearCache is required to flush any cached representations
func (e *Event) RemoveTag(tag string) {
	e.data["tags"].(Tags).Remove(tag)
}

// ClearCache clears any cached representations, always call it if the event is changed
func (e *Event) ClearCache() {
	e.encoded = nil
}

// Bytes returns the encoded event bytes
// The returned slice should not be modified and be treated immutable
// There is currently no way to modify it. To change the event, use Data(),
// and then use ClearCache to clear the Bytes() cache so it regenerates
func (e *Event) Bytes() []byte {
	if e.encoded == nil {
		var err error
		e.encoded, err = json.Marshal(e.data)
		if err != nil {
			e.encoded = make([]byte, 0)
		}
	}
	return e.encoded
}

// Context returns the stream context for this Event - and can be used to
// distinguish events from different sources
func (e *Event) Context() context.Context {
	return e.ctx
}

// DispatchAck processes a bulk of events and calls the required acknowledgement
// callbacks
func DispatchAck(events []*Event) {
	if len(events) == 0 {
		return
	}

	// Grab first ackFunc
	e := 0
	acker := events[0].acker
	for _, event := range events[1:] {
		e++

		// Different acker?
		if event.acker != acker {
			// Multiple event acks in the works, split up the bulk and pass to the
			// relevant callbacks
			dispatchAckForMultipleFuncs(acker, append([]*Event(nil), events[0:e]...), events[e:])
			return
		}
	}

	if acker == nil {
		// Skip where no acker
		return
	}

	// Single event type - call it's acknowledger
	acker.Acknowledge(events)
}

// dispatchAckForMultipleFuncs handles the case where a bulk of events to be
// acknowledged contains events for different processors by splitting them apart
// and calling each processor
func dispatchAckForMultipleFuncs(firstAcker Acknowledger, firstAckerEvents []*Event, events []*Event) {
	ackMap := map[Acknowledger][]*Event{}
	if firstAcker != nil {
		ackMap[firstAcker] = firstAckerEvents
	}

	for _, event := range events[1:] {
		if event.acker != nil {
			ackMap[event.acker] = append(ackMap[event.acker], event)
		}
	}

	for acker, ackEvents := range ackMap {
		acker.Acknowledge(ackEvents)
	}
}
