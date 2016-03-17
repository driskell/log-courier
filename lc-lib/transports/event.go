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

import "encoding/binary"

// Event is the interface implemented by all event structures
type Event interface {
	Observer() Observer
}

// StatusChange holds a value that represents a change in transport status
type StatusChange int

// Transport status change signals
const (
	Started StatusChange = iota
	Ready
	Failed
	Finished
)

// StatusEvent contains information about a status change for a transport
type StatusEvent struct {
	observer     Observer
	statusChange StatusChange
}

// NewStatusEvent generates a new StatusEvent for the given Observer/Endpoint
func NewStatusEvent(observer Observer, statusChange StatusChange) *StatusEvent {
	return &StatusEvent{
		observer:     observer,
		statusChange: statusChange,
	}
}

// Observer returns the endpoint associated with this event
func (e *StatusEvent) Observer() Observer {
	return e.observer
}

// StatusChange returns the status change value
func (e *StatusEvent) StatusChange() StatusChange {
	return e.statusChange
}

// AckEvent contains information on which events have been acknowledged
type AckEvent struct {
	observer Observer
	nonce    string
	sequence uint32
}

// NewAckEvent generates a new AckEvent for the given Endpoint
func NewAckEvent(observer Observer, nonce string, sequence uint32) *AckEvent {
	return &AckEvent{
		observer: observer,
		nonce:    nonce,
		sequence: sequence,
	}
}

// NewAckEventWithBytes generates a new AckEvent using bytes, conveniently
// converting them to string and uint32
func NewAckEventWithBytes(observer Observer, nonce []byte, sequence []byte) *AckEvent {
	stringNonce := string(nonce)
	integerSequence := binary.BigEndian.Uint32(sequence)
	return NewAckEvent(observer, stringNonce, integerSequence)
}

// Observer returns the endpoint associated with this event
func (e *AckEvent) Observer() Observer {
	return e.observer
}

// Nonce returns the nonce value
func (e *AckEvent) Nonce() string {
	return e.nonce
}

// Sequence returns the sequence value
func (e *AckEvent) Sequence() uint32 {
	return e.sequence
}

// PongEvent is received when a transport has responded to a Ping() request
type PongEvent struct {
	observer Observer
}

// NewPongEvent generates a new PongEvent for the given Endpoint
func NewPongEvent(observer Observer) *PongEvent {
	return &PongEvent{
		observer: observer,
	}
}

// Observer returns the endpoint associated with this event
func (e *PongEvent) Observer() Observer {
	return e.observer
}
