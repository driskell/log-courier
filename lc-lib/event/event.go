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
	"time"
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
//
// If it is mutated, ClearCache must be called to ensure any cached representations are
// regenerated when returning the event to the network or otherwise
// Mutations must not modify the type of the guaranteed fields listed below, or the
// behaviour is undefined and likely to result in panic
//
// The following "keys" are guaranteed to exist and be of a specific type:
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

// AddTag adds a tag to the event
func (e *Event) AddTag(tag string) {
	e.data["tags"].(Tags).Add(tag)
	e.ClearCache()
}

// RemoveTag adds a tag to the event
func (e *Event) RemoveTag(tag string) {
	e.data["tags"].(Tags).Remove(tag)
	e.ClearCache()
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
