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

import "encoding/json"

// AckCallback is a function to a call when events have been acknowledged
type AckCallback func([]*Event)

// ackCallbacks contains the registered callbacks for different event types
var ackCallbacks = make(map[string]AckCallback)

// Event describes an event
type Event struct {
	data      map[string]interface{}
	encoded   []byte
	context   interface{}
	eventType string
}

// NewEvent creates a new event structure from the given data
func NewEvent(eventType string, data map[string]interface{}, context interface{}) *Event {
	return &Event{
		eventType: eventType,
		data:      data,
		context:   context,
	}
}

// Data returns the event data, or nil if it was freed
func (e *Event) Data() map[string]interface{} {
	return e.data
}

// Encode returns the event data in JSON format
func (e *Event) Encode() (err error) {
	if e.encoded != nil {
		panic("event is already encoded")
	}

	e.encoded, err = json.Marshal(e.data)
	if err != nil {
		e.data = nil
	}
	return
}

// Bytes returns the encoded event bytes
func (e *Event) Bytes() []byte {
	return e.encoded
}

// Context returns the stream context for this Event - and can be used to
// distinguish events from different sources
func (e *Event) Context() interface{} {
	return e.context
}

// RegisterForAck registers a function to be called when events with the given
// type name are acknowledged
func RegisterForAck(eventType string, ackCallback AckCallback) {
	ackCallbacks[eventType] = ackCallback
}

// DispatchAck processes a bulk of events and calls the required acknowledgement
// callbacks
func DispatchAck(events []*Event) {
	if len(events) == 0 {
		return
	}

	// Grab first ackFunc
	e := 0
	eventType := events[0].eventType
	for _, event := range events[1:] {
		e++

		// Different ackFunc?
		if event.eventType != eventType {
			// Multiple event types in the works, split up the bulk and pass to the
			// relevant callbacks
			dispatchAckForMultipleFuncs(eventType, append([]*Event(nil), events[0:e]...), events[e:])
			return
		}
	}

	if eventType == "" {
		// Skip where no event type
		return
	}

	// Single event type - call it's ackFunc
	if ackCallback, ok := ackCallbacks[eventType]; ok {
		ackCallback(events)
	}
}

// dispatchAckForMultipleFuncs handles the case where a bulk of events to be
// acknowledged contains events for different processors by splitting them apart
// and calling each processor
func dispatchAckForMultipleFuncs(firstEventType string, firstAckEvents []*Event, events []*Event) {
	ackMap := map[string][]*Event{}
	if firstEventType != "" {
		ackMap[firstEventType] = firstAckEvents
	}

	for _, event := range events[1:] {
		if event.eventType != "" {
			ackMap[event.eventType] = append(ackMap[event.eventType], event)
		}
	}

	for eventType, ackEvents := range ackMap {
		if ackCallback, ok := ackCallbacks[eventType]; ok {
			ackCallback(ackEvents)
		}
	}
}
