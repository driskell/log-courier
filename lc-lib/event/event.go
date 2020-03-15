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
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
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

// Builtin is used around builtin keys to allow damage prevention
type Builtin interface {
	// VerifySetEnter checks if we can set the given key (if we're a map for example)
	VerifySetEnter(string) (map[string]interface{}, error)
	// VerifySet checks if we can be set to the given value
	VerifySet(interface{}) (interface{}, error)
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
			e.data["@timestamp"] = Timestamp(time.Now())
			e.data["@metadata"] = map[string]interface{}{}
			e.data["tags"] = Tags{"_unmarshal_failure"}
		} else {
			e.convertData()
		}
	}
	return e.data
}

// convertData is the internal function that enforces guaranteed types
func (e *Event) convertData() {
	// Normalize "tags" first (other resolutions ignore it)
	if entry, ok := e.data["tags"]; ok {
		switch value := entry.(type) {
		case Tags:
		case string:
			e.data["tags"] = Tags{value}
		case []interface{}:
			// From unmarshaled over the wire we get []interface{}
			tags := Tags{}
			for _, tag := range value {
				tagString, ok := tag.(string)
				if !ok {
					tags = Tags{"_tags_parse_failure"}
					e.data["tags_parse_error"] = fmt.Sprintf("tags list must contain only strings, found a %T", tag)
					break
				}
				tags = append(tags, tagString)
			}
			e.data["tags"] = tags
		default:
			e.data["tags"] = Tags{"_tags_parse_failure"}
			e.data["tags_parse_error"] = fmt.Sprintf("tags was not a string or string list, was %T", value)
		}
	} else {
		e.data["tags"] = Tags{}
	}
	// Normalize "@timestamp" to a time.Time
	if entry, ok := e.data["@timestamp"]; ok {
		switch value := entry.(type) {
		case Timestamp:
		case time.Time:
			e.data["@timestamp"] = Timestamp(value)
		case string:
			parsed, err := time.Parse(time.RFC3339, value)
			if err != nil {
				e.data["@timestamp"] = Timestamp(time.Now())
				e.data["timestamp_parse_error"] = err
				e.AddTag("_timestamp_parse_failure")
			} else {
				e.data["@timestamp"] = Timestamp(parsed)
			}
		default:
			e.data["@timestamp"] = Timestamp(time.Now())
			e.data["timestamp_parse_error"] = fmt.Sprintf("@timestamp was not a string, was %T", value)
			e.AddTag("_timestamp_parse_failure")
		}
	} else {
		e.data["@timestamp"] = Timestamp(time.Now())
	}
	// Normalize "@metadata"
	e.data["@metadata"] = Metadata{}
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
//     @metadata: map[string]interface{}
//         Always empty, and overrides any value that came in over the wire.
//         It is purely for processing metadata and is removed prior to any sending
//     tags: Tags (Empty Tags instance)
//         If no value currently exists, it is created empty with no tags.
//         If the existing value is invalid, the _tags_parse_failure tag is added
//         and the error is stored inside the tags_parse_error field
//
// If any error occured during unmarshaling of an event received over the network
// then a new event with only a message key (and the above defaults) will be created
// where the message is the error message, and the "_unmarshal_failure" tag is added
func (e *Event) Resolve(path string, set interface{}) (output interface{}, err error) {
	currentMap := e.Data()
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
				if builtin, ok := currentMap[name].(Builtin); ok {
					if set == ResolveParamUnset {
						return nil, fmt.Errorf("Builtin entry '%s' cannot be unset", path)
					}
					result, err := builtin.VerifySet(set)
					if err != nil {
						return nil, err
					}
					currentMap[name] = result
				} else {
					if set == ResolveParamUnset {
						delete(currentMap, name)
					} else {
						currentMap[name] = set
					}
				}
			}
		} else {
			// Calculate next inner
			// Can't use non-map, ignore as if it didn't exist, and keep validating
			// (However, if we are setting, overwrite it with empty map)
			if value, ok := currentMap[name].(map[string]interface{}); ok {
				currentMap = value
			} else if builtin, ok := currentMap[name].(Builtin); ok {
				// Block entry to set if this is a builtin that does not want it
				enter, err := builtin.VerifySetEnter(name)
				if err != nil {
					return nil, err
				}
				currentMap = enter
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

// AddError adds an error tag and an error message field for a specific action that has failed
func (e *Event) AddError(action string, message string) {
	e.Data()[fmt.Sprintf("_%s_error", action)] = message
	e.AddTag(fmt.Sprintf("_%s_failure", action))
}

// AddTag adds a tag to the event
// Remember ClearCache is required to flush any cached representations
func (e *Event) AddTag(tag string) {
	data := e.Data()
	tags := data["tags"].(Tags)
	length := len(tags)
	idx := sort.SearchStrings(tags, tag)
	if idx >= length {
		data["tags"] = append(tags, tag)
	} else if tags[idx] != tag {
		if length+1 > cap(tags) {
			oldTags := tags
			tags = make(Tags, length+1)
			copy(tags, oldTags[:idx])
			copy(tags[idx+1:], oldTags[idx:])
		} else {
			tags = tags[:length+1]
			copy(tags[idx+1:], tags[idx:])
		}
		tags[idx] = tag
		data["tags"] = tags
	}
}

// RemoveTag adds a tag to the event
// Remember ClearCache is required to flush any cached representations
func (e *Event) RemoveTag(tag string) {
	data := e.Data()
	tags := data["tags"].(Tags)
	length := len(tags)
	idx := sort.SearchStrings(tags, tag)
	if idx < length && tags[idx] == tag {
		copy(tags[idx:], tags[idx+1:])
		data["tags"] = tags[:length-1]
	}
}

// ClearCache clears any cached representations, always call it if the event is changed
func (e *Event) ClearCache() {
	e.encoded = nil
}

// Bytes returns the encoded event bytes
// The @metadata key is completely ignored
// The returned slice should not be modified and be treated immutable
// There is currently no way to modify it. To change the event, use Data(),
// and then use ClearCache to clear the Bytes() cache so it regenerates
func (e *Event) Bytes() []byte {
	if e.encoded == nil {
		var err error
		metadata := e.data["@metadata"]
		delete(e.data, "@metadata")
		e.encoded, err = json.Marshal(e.data)
		e.data["@metadata"] = metadata
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

	for _, event := range events {
		if event.acker != nil {
			ackMap[event.acker] = append(ackMap[event.acker], event)
		}
	}

	for acker, ackEvents := range ackMap {
		acker.Acknowledge(ackEvents)
	}
}
