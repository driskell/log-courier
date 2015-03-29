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

package endpoint

import (
	"errors"
	"github.com/driskell/log-courier/src/lc-lib/addresspool"
	"github.com/driskell/log-courier/src/lc-lib/internallist"
	"github.com/driskell/log-courier/src/lc-lib/payload"
	"github.com/driskell/log-courier/src/lc-lib/transports"
	"math/rand"
	"sync"
	"time"
)

// Endpoint structure represents a single remote endpoint
type Endpoint struct {
	sync.Mutex

	// Whether this endpoint is ready for events or not
	ready bool

	// Whether this endpoint is ready for events, but is not allowed any more due
	// to peer send queue limits
	full bool

	// Element structures for internal use by InternalList via EndpointSink
	// MUST have Value member initialised
	timeoutElement internallist.Element
	readyElement   internallist.Element
	fullElement    internallist.Element

	// Timeout callback and when it should trigger
	timeoutFunc interface{}
	timeoutDue  time.Time

	sink            *Sink
	server          string
	addressPool     *addresspool.Pool
	transport       transports.Transport
	pendingPayloads map[string]*payload.Pending
	numPayloads     int
	pongPending     bool

	lineCount int64
}

// init prepares the internal Element structures for InternalList and sets the
// overall initial state
func (e *Endpoint) init() {
	e.timeoutElement.Value = e
	e.readyElement.Value = e
	e.fullElement.Value = e

	e.resetPayloads()
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
func (e *Endpoint) SendPayload(payload *payload.Pending) error {
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

	return e.transport.Write(payload.Nonce, payload.Events())
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
func (e *Endpoint) ProcessAck(a *transports.AckResponse) (*payload.Pending, bool) {
	log.Debug("Acknowledgement received for payload %x sequence %d", a.Nonce, a.Sequence)

	// Grab the payload the ACK corresponds to by using nonce
	payload, found := e.pendingPayloads[a.Nonce]
	if !found {
		// Don't fail here in case we had temporary issues and resend a payload, only for us to receive duplicate ACKN
		return nil, false
	}

	firstAck := !payload.HasAck()

	// Process ACK
	lines, complete := payload.Ack(int(a.Sequence))
	e.lineCount += int64(lines)

	if complete {
		// No more events left for this payload, remove from pending list
		delete(e.pendingPayloads, a.Nonce)
		e.numPayloads--
	}

	return payload, firstAck
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
func (e *Endpoint) Recover() []*payload.Pending {
	pending := make([]*payload.Pending, len(e.pendingPayloads))
	for _, payload := range e.pendingPayloads {
		pending = append(pending, payload)
	}
	e.resetPayloads()
	return pending
}

// HasTimeout returns true if this endpoint already has an associated timeout
func (e *Endpoint) HasTimeout() bool {
	return e.timeoutFunc != nil
}

// IsFull returns true if this endpoint has been marked as full
func (e *Endpoint) IsFull() bool {
	return e.full
}

// resetPayloads resets the internal state for pending payloads
func (e *Endpoint) resetPayloads() {
	e.pendingPayloads = make(map[string]*payload.Pending)
	e.numPayloads = 0
}

// Pool returns the associated address pool
// This implements part of the transports.Endpoint interface for callbacks
func (e *Endpoint) Pool() *addresspool.Pool {
  return e.addressPool
}

// Ready is called by a transport to signal it is ready for events.
// This should be triggered once connection is successful and the transport is
// ready to send data. It should NOT be called again until the transport
// receives data, otherwise the call may block.
// This implements part of the transports.Endpoint interface for callbacks
func (e *Endpoint) Ready() {
	e.sink.readyChan <- e
}

// ResponseChan returns the channel that responses should be sent on
// This implements part of the transports.Endpoint interface for callbacks
func (e *Endpoint) ResponseChan() chan<- transports.Response {
	return e.sink.responseChan
}

// Fail is called by a transport to signal an error has occurred, and that all
// pending payloads should be returned to the publisher for retransmission
// elsewhere.
// This implements part of the transports.Endpoint interface for callbacks
func (e *Endpoint) Fail(err error) {
	e.sink.failChan <- &Failure{e, err}
}
