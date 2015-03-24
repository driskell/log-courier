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

package publisher

import (
	"errors"
	"github.com/driskell/log-courier/src/lc-lib/core"
	"math/rand"
	"sync"
	"time"
)

// Endpoint structure represents a single remote endpoint
type Endpoint struct {
	sync.Mutex

	// The associated remote
	Remote *EndpointRemote

	// Whether this endpoint is ready for events or not
	Ready bool

	// Whether this endpoint is ready for events, but is not allowed any more due
	// to peer send queue limits
	Full bool

	// Linked lists for internal use by Publisher
	// TODO: Should be use by EndpointSink only to allow separate endpoint package
	NextTimeout *Endpoint
	PrevTimeout *Endpoint
	NextFull *Endpoint
	PrevFull *Endpoint
	NextReady *Endpoint

	// Timeout callback and when it should trigger
	TimeoutFunc TimeoutFunc
	TimeoutDue  time.Time

	server          string
	addressPool     *AddressPool
	transport       core.Transport
	pendingPayloads map[string]*PendingPayload
	numPayloads     int
	pongPending     bool

	lineCount int64
}

// Shutdown signals the transport to start shutting down
func (e *Endpoint) shutdown() {
	e.transport.Shutdown()
}

// Wait for the transport to finish
// Disconnect() should be called first on all endpoints to start them
// disconnecting, and then Wait called to allow them to finish.
func (e *Endpoint) wait() {
	e.transport.Wait()
}

// Server returns the server string from the configuration file that this
// endpoint is associated with
func (e *Endpoint) Server() string {
	return e.server
}

// SendPayload adds an event spool to the queue, sending it to the transport.
// Always called from Publisher routine to ensure concurrent pending payload
// access.
//
// Should return the payload so the publisher can track it accordingly.
func (e *Endpoint) SendPayload(payload *PendingPayload) error {
	// Calculate a nonce
	nonce := e.generateNonce()
	for {
		if _, found := e.pendingPayloads[nonce]; !found {
			break
		}
		// Collision - generate again - should be extremely rare
		nonce = e.generateNonce()
	}

	payload.Nonce = nonce
	e.pendingPayloads[nonce] = payload
	e.numPayloads++

	return e.transport.Write(payload)
}

// generateNonce creates a random string for payload identification
func (e *Endpoint) generateNonce() string {
	// This could maybe be made a bit more efficient
	nonce := make([]byte, 16)
	for i := 0; i < 16; i++ {
		nonce[i] = byte(rand.Intn(255))
	}
	return string(nonce)
}

// SendPing requests that the transport ensure its connection is still available
// by sending data across it and calling back with Pong(). Some transports may
// simply do nothing and Pong() back immediately if they are managed as such.
func (e *Endpoint) SendPing() error {
	e.pongPending = true
	return e.transport.Ping()
}

// IsPinging returns true if the endpoint is still awaiting for a PONG response
// to a previous Ping request
func (e *Endpoint) IsPinging() bool {
	return e.pongPending
}

// ProcessAck processes a received acknowledgement message.
//
// When Ack() is called by the transport we feed a channel through to the
// publisher, which calls the message processor. This message processer passes
// control here to allow us to clear pending payloads.
// The reason for this is to ensure all payload allocation and manipulation
// happens on the publisher main routine.
//
// We should return the payload that was acked, and whether this is the first
// acknoweldgement or a later one, so the publisher may track out of sync
// payload processing accordingly.
func (e *Endpoint) ProcessAck(a *AckResponse) (*PendingPayload, bool) {
	log.Debug("Acknowledgement received for payload %x sequence %d", a.Nonce, a.Sequence)

	// Grab the payload the ACK corresponds to by using nonce
	payload, found := e.pendingPayloads[a.Nonce]
	if !found {
		// Don't fail here in case we had temporary issues and resend a payload, only for us to receive duplicate ACKN
		return nil, false
	}

	ackEvents := payload.ackEvents

	// Process ACK
	lines, complete := payload.Ack(int(a.Sequence))
	e.lineCount += int64(lines)

	if complete {
		// No more events left for this payload, remove from pending list
		delete(e.pendingPayloads, a.Nonce)
		e.numPayloads--
	}

	return payload, ackEvents == 0
}

// ProcessPong processes a received PONG message
func (e *Endpoint) ProcessPong() error {
	if !e.pongPending {
		return errors.New("Unexpected PONG received")
	}

	log.Debug("[%s] Received PONG message", e.Server())
	e.pongPending = false

	return nil
}

// NumPending returns the number of pending payloads on this endpoint
func (e *Endpoint) NumPending() int {
	return e.numPayloads
}

// Recover all queued payloads and return them back to the publisher
// Called when a failure happens
func (e *Endpoint) Recover() []*PendingPayload {
	pending := make([]*PendingPayload, len(e.pendingPayloads))
	for _, payload := range e.pendingPayloads {
		pending = append(pending, payload)
	}
	e.resetPayloads()
	return pending
}

// resetPayloads resets the internal state for pending payloads
func (e *Endpoint) resetPayloads() {
	e.pendingPayloads = make(map[string]*PendingPayload)
	e.numPayloads = 0
}
