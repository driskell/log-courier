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

package transports

import (
	"context"
	"errors"

	"github.com/driskell/log-courier/lc-lib/config"
)

var (
	// ErrCongestion represents temporary congestion, rather than failure
	ErrCongestion error = errors.New("Congestion")

	// ErrInvalidState occurs when a send cannot happen because the connection has closed
	ErrInvalidState = errors.New("invalid connection state")
)

// TransportContext
type TransportContext string

const (
	// ContextConnection provides a value representing an individual connection
	// The returned interface should be treated opaque outside of the relevant transport package
	ContextConnection TransportContext = "connection"

	// ContextReceiver provides the Receiver that a connection relates to
	ContextReceiver TransportContext = "receiver"
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
	err          error
}

// NewStatusEvent generates a new StatusEvent for the given context
func NewStatusEvent(context context.Context, statusChange StatusChange, err error) *StatusEvent {
	return &StatusEvent{
		context:      context,
		statusChange: statusChange,
		err:          err,
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

// StatusChange returns the error associated with a Failed status
func (e *StatusEvent) Err() error {
	return e.err
}

// AckEvent contains information on which events have been acknowledged
type AckEvent struct {
	context  context.Context
	nonce    *string
	sequence uint32
}

// NewAckEvent generates a new AckEvent for the given Endpoint
func NewAckEvent(context context.Context, nonce *string, sequence uint32) *AckEvent {
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
func (e *AckEvent) Nonce() *string {
	return e.nonce
}

// Sequence returns the sequence value
func (e *AckEvent) Sequence() uint32 {
	return e.sequence
}

// ConnectEvent marks the start of a new connection on a reciver
type ConnectEvent struct {
	context context.Context
}

// NewConnectEvent generates a new ConnectEvent for the given Endpoint
func NewConnectEvent(context context.Context) *ConnectEvent {
	return &ConnectEvent{
		context: context,
	}
}

// Context returns the endpoint associated with this event
func (e *ConnectEvent) Context() context.Context {
	return e.context
}

// EventsEvent contains events received from a transport
type EventsEvent struct {
	context context.Context
	nonce   *string
	events  [][]byte
}

// NewEventsEvent generates a new EventsEvent for the given Endpoint
func NewEventsEvent(context context.Context, nonce *string, events [][]byte) *EventsEvent {
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
func (e *EventsEvent) Nonce() *string {
	return e.nonce
}

// Events returns the events
func (e *EventsEvent) Events() [][]byte {
	return e.events
}

// Count returns the number of events in the payload
func (e *EventsEvent) Count() uint32 {
	return uint32(len(e.events))
}

// EndEvent marks the end of a stream of events from an endpoint
type EndEvent struct {
	context context.Context
}

// NewEndEvent generates a new EndEvent for the given Endpoint
func NewEndEvent(context context.Context) *EndEvent {
	return &EndEvent{
		context: context,
	}
}

// Context returns the endpoint associated with this event
func (e *EndEvent) Context() context.Context {
	return e.context
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

// init registers this module provider
func init() {
	config.RegisterAvailable("transports", AvailableTransports)
	config.RegisterAvailable("receivers", AvailableReceivers)
}
