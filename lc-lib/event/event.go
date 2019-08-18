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

// Acknowledger is something that can be called with events once they ar acknowledged
type Acknowledger interface {
	Acknowledge([]*Event)
}

// Event describes an event
// TODO: Have Event interface and split into two structures, mapEvent and encodedEvent
//       Will mean read/write of JDAT without decoding is possible
type Event struct {
	data    map[string]interface{}
	encoded []byte
	context interface{}
	acker   Acknowledger
}

// NewEvent creates a new event structure from the given data
func NewEvent(acker Acknowledger, data map[string]interface{}, context interface{}) *Event {
	return &Event{
		acker:   acker,
		data:    data,
		context: context,
	}
}

// NewEventFromBytes creates a new event structure from the given data
func NewEventFromBytes(acker Acknowledger, data []byte, context interface{}) *Event {
	return &Event{
		acker:   acker,
		encoded: data,
		context: context,
	}
}

// Data returns the event data
func (e *Event) Data() map[string]interface{} {
	if e.data == nil {
		err := json.Unmarshal(e.encoded, &e.data)
		if err != nil {
			e.data = make(map[string]interface{})
			e.data["message"] = err.Error()
		} else {
			e.encoded = nil
		}
	}
	return e.data
}

// Bytes returns the encoded event bytes
func (e *Event) Bytes() []byte {
	if e.encoded == nil {
		var err error
		e.encoded, err = json.Marshal(e.data)
		if err != nil {
			e.encoded = make([]byte, 0)
		} else {
			e.data = nil
		}
	}
	return e.encoded
}

// Context returns the stream context for this Event - and can be used to
// distinguish events from different sources
func (e *Event) Context() interface{} {
	return e.context
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
