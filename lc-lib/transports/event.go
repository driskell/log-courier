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

package transports

import (
	"context"

	"github.com/driskell/log-courier/lc-lib/event"
)

// Event is the interface implemented by all event structures
type Event interface {
	Context() context.Context
}

// StatusChange holds a value that represents a change in transport status
type StatusChange int

// Transport status change signals
const (
	Started StatusChange = iota
	Failed
	Finished
)

// StatusEvent contains information about a status change for a transport
type StatusEvent struct {
	context      context.Context
	statusChange StatusChange
}

// NewStatusEvent generates a new StatusEvent for the given context
func NewStatusEvent(context context.Context, statusChange StatusChange) *StatusEvent {
	return &StatusEvent{
		context:      context,
		statusChange: statusChange,
	}
}

// Context returns the endpoint associated with this event
func (e *StatusEvent) Context() context.Context {
	return e.context
}

// StatusChange returns the status change value
func (e *StatusEvent) StatusChange() StatusChange {
	return e.statusChange
}

// AckEvent contains information on which events have been acknowledged
type AckEvent struct {
	context  context.Context
	nonce    string
	sequence uint32
}

// NewAckEvent generates a new AckEvent for the given Endpoint
func NewAckEvent(context context.Context, nonce string, sequence uint32) *AckEvent {
	return &AckEvent{
		context:  context,
		nonce:    nonce,
		sequence: sequence,
	}
}

// Context returns the endpoint associated with this event
func (e *AckEvent) Context() context.Context {
	return e.context
}

// Nonce returns the nonce value
func (e *AckEvent) Nonce() string {
	return e.nonce
}

// Sequence returns the sequence value
func (e *AckEvent) Sequence() uint32 {
	return e.sequence
}

// EventsEvent contains events that need to be ingested
type EventsEvent struct {
	context context.Context
	nonce   string
	events  []*event.Event
}

// NewEventsEvent generates a new EventsEvent for the given Endpoint
func NewEventsEvent(context context.Context, nonce string, events []*event.Event) *EventsEvent {
	return &EventsEvent{
		context: context,
		nonce:   nonce,
		events:  events,
	}
}

// Context returns the endpoint associated with this event
func (e *EventsEvent) Context() context.Context {
	return e.context
}

// Nonce returns the nonce
func (e *EventsEvent) Nonce() string {
	return e.nonce
}

// Events returns the events
func (e *EventsEvent) Events() []*event.Event {
	return e.events
}

// Count returns the number of events in the payload
func (e *EventsEvent) Count() uint32 {
	return uint32(len(e.events))
}

// PongEvent is received when a transport has responded to a Ping() request
type PongEvent struct {
	context context.Context
}

// NewPongEvent generates a new PongEvent for the given Endpoint
func NewPongEvent(context context.Context) *PongEvent {
	return &PongEvent{
		context: context,
	}
}

// Context returns the endpoint associated with this event
func (e *PongEvent) Context() context.Context {
	return e.context
}

// PingEvent is received when a transport has responded to a Ping() request
type PingEvent struct {
	context context.Context
}

// NewPingEvent generates a new PingEvent for the given Endpoint
func NewPingEvent(context context.Context) *PingEvent {
	return &PingEvent{
		context: context,
	}
}

// Context returns the endpoint associated with this event
func (e *PingEvent) Context() context.Context {
	return e.context
}
