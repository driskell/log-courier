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

// status holds an Endpoint status
type status int

// Endpoint statuses
const (
	// Not yet ready
	endpointStatusIdle status = iota

	// Ready to receive events
	endpointStatusReady

	// Could receive events but too many are oustanding
	endpointStatusFull

	// Do not use this endpoint, it has failed
	endpointStatusFailed
)

// StatusChange holds a value that represents a change in endpoint status that
// is sent over the status channel of the Sink
type StatusChange int

// Endpoint status signals
const (
	Ready     = iota
	Recovered
	Failed
	Finished
)

// Status structure contains the reason for failure, or nil if recovered
type Status struct {
	Endpoint *Endpoint
	Status   StatusChange
}

// Endpoint structure represents a single remote endpoint
type Endpoint struct {
	sync.Mutex

	// The endpoint status
	status status

	// Element structures for internal use by InternalList via EndpointSink
	// MUST have Value member initialised
	timeoutElement internallist.Element
	readyElement   internallist.Element
	fullElement    internallist.Element
	failedElement  internallist.Element

	// Timeout callback and when it should trigger
	timeoutFunc interface{}
	timeoutDue  time.Time

	sink            *Sink
	server          string
	addressPool     *addresspool.Pool
	transport       transports.Transport
	pendingPayloads map[string]*payload.Payload
	numPayloads     int
	pongPending     bool
	shuttingDown    bool

	lineCount int64
}

// Init prepares the internal Element structures for InternalList and prepares
// the pending payload structures
func (e *Endpoint) Init() {
	e.timeoutElement.Value = e
	e.readyElement.Value = e
	e.fullElement.Value = e

	e.resetPayloads()
}

// shutdown signals the transport to start shutting down
func (e *Endpoint) shutdown() {
	if e.shuttingDown {
		return
	}

	e.transport.Shutdown()
	e.shuttingDown = true
}

// isShuttingDown returns true if shutdown has been called
func (e *Endpoint) isShuttingDown() bool {
	return e.shuttingDown
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
func (e *Endpoint) SendPayload(payload *payload.Payload) error {
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

	log.Debug("[%s] Sending payload %x", e.Server(), nonce)

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
func (e *Endpoint) ProcessAck(a *transports.AckResponse) (*payload.Payload, bool) {
	log.Debug("[%s] Acknowledgement received for payload %x sequence %d", e.Server(), a.Nonce, a.Sequence)

	// Grab the payload the ACK corresponds to by using nonce
	payload, found := e.pendingPayloads[a.Nonce]
	if !found {
		// Don't fail here in case we had temporary issues and resend a payload, only for us to receive duplicate ACKN
		log.Debug("[%s] Duplicate/corrupt ACK received for message %x", e.Server(), a.Nonce)
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

// PullBackPending returns all queued payloads back to the publisher
// Called when a failure happens
func (e *Endpoint) PullBackPending() []*payload.Payload {
	pending := make([]*payload.Payload, 0, len(e.pendingPayloads))
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

// IsFailed returns true if this endpoint has been marked as failed
func (e *Endpoint) IsFailed() bool {
	return e.status == endpointStatusFailed
}

// IsFull returns true if this endpoint has been marked as full
func (e *Endpoint) IsFull() bool {
	return e.status == endpointStatusFull
}

// resetPayloads resets the internal state for pending payloads
func (e *Endpoint) resetPayloads() {
	e.pendingPayloads = make(map[string]*payload.Payload)
	e.numPayloads = 0
}

// Pool returns the associated address pool
// This implements part of the transports.Endpoint interface for callbacks
func (e *Endpoint) Pool() *addresspool.Pool {
  return e.addressPool
}

// Ready is called by a transport to signal it is ready for events.
// This should be triggered once connection is successful and the transport is
// ready to send data.
// This implements part of the transports.Endpoint interface for callbacks
func (e *Endpoint) Ready() {
	e.sink.statusChan <- &Status{e, Ready}
}

// Finished is called by a transport to signal it has shutdown.
// This should be triggered only after Shutdown has been called and the
// transport has disconnected and completely cleaned up, as during shutdown the
// application will exit imminently
func (e *Endpoint) Finished() {
	e.sink.statusChan <- &Status{e, Finished}
}

// ResponseChan returns the channel that responses should be sent on
// This implements part of the transports.Endpoint interface for callbacks
func (e *Endpoint) ResponseChan() chan<- transports.Response {
	return e.sink.responseChan
}

// Fail is called by a transport to signal an error has occurred, and that all
// pending payloads should be returned to the publisher for retransmission
// elsewhere. All subsequent Ready signals will also be ignored until Recover()
// is called
// This implements part of the transports.Endpoint interface for callbacks
func (e *Endpoint) Fail() {
	e.sink.statusChan <- &Status{e, Failed}
}

// Recover is called by a transport to signal a failure has recovered
// Sending of payloads will begin immediately as if a ready signal was sent to
// an idle endpoint
// This implements part of the transports.Endpoint interface for callbacks
func (e *Endpoint) Recover() {
	e.sink.statusChan <- &Status{e, Recovered}
}
