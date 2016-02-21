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

package endpoint

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/internallist"
	"github.com/driskell/log-courier/lc-lib/payload"
	"github.com/driskell/log-courier/lc-lib/transports"
)

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
	orderedElement internallist.Element

	// Timeout callback and when it should trigger
	// The Sink manages these
	timeoutFunc interface{}
	timeoutDue  time.Time

	sink            *Sink
	server          string
	addressPool     *addresspool.Pool
	finishOnFail    bool
	transport       transports.Transport
	pendingPayloads map[string]*payload.Payload
	numPayloads     int
	pongPending     bool

	lineCount      int64
	averageLatency float64
}

// Init prepares the internal Element structures for InternalList and prepares
// the pending payload structures
func (e *Endpoint) Init() {
	e.timeoutElement.Value = e
	e.readyElement.Value = e
	e.fullElement.Value = e
	e.failedElement.Value = e
	e.orderedElement.Value = e

	e.resetPayloads()

	e.transport = transports.NewTransport(e.sink.config.Factory, e, e.finishOnFail)
}

// Prev returns the previous endpoint in the ordered list
func (e *Endpoint) Prev() *Endpoint {
	if e.orderedElement.Prev() == nil {
		return nil
	}
	return e.orderedElement.Prev().Value.(*Endpoint)
}

// Next returns the next endpoint in the ordered list
func (e *Endpoint) Next() *Endpoint {
	if e.orderedElement.Next() == nil {
		return nil
	}
	return e.orderedElement.Next().Value.(*Endpoint)
}

// shutdownTransport signals the transport to start shutting down
func (e *Endpoint) shutdownTransport() {
	if e.status != endpointStatusClosing {
		return
	}

	log.Debug("[%s] Endpoint is now shutting down", e.Server())
	e.transport.Shutdown()
}

// Server returns the server string from the configuration file that this
// endpoint is associated with
func (e *Endpoint) Server() string {
	return e.server
}

// SendPayload registers a payload with the endpoint and sends it to the
// transport
func (e *Endpoint) SendPayload(payload *payload.Payload) error {
	// Must be in a ready state
	if e.status != endpointStatusReady {
		panic(fmt.Sprintf("Endpoint is not ready (%d)", e.status))
	}

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
	payload.TransmitTime = time.Now()
	e.pendingPayloads[nonce] = payload
	e.numPayloads++

	log.Debug("[%s] Sending payload %x", e.Server(), nonce)

	if err := e.transport.Write(payload.Nonce, payload.Events()); err != nil {
		return err
	}

	e.status = endpointStatusBusy
	return nil
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

// processAck processes a received acknowledgement message.
// This will pass the payload that was acked, and whether this is the first
// acknoweldgement or a later one, to the observer
// It should return whether or not the payload was completed so full status
// can be updated
func (e *Endpoint) processAck(ack *transports.AckEvent, observer Observer) bool {
	log.Debug("[%s] Acknowledgement received for payload %x sequence %d", e.Server(), ack.Nonce(), ack.Sequence())

	// Grab the payload the ACK corresponds to by using nonce
	payload, found := e.pendingPayloads[ack.Nonce()]
	if !found {
		// Don't fail here in case we had temporary issues and resend a payload, only for us to receive duplicate ACKN
		log.Debug("[%s] Duplicate/corrupt ACK received for message %x", e.Server(), ack.Nonce())
		return false
	}

	firstAck := !payload.HasAck()

	// Process ACK
	lines, complete := payload.Ack(int(ack.Sequence()))
	e.lineCount += int64(lines)

	if complete {
		// No more events left for this payload, remove from pending list
		delete(e.pendingPayloads, ack.Nonce())
		e.numPayloads--

		// Mark the running average latency of this endpoint per-event over the last
		// 5 payloads
		e.averageLatency = core.CalculateRunningAverage(
			1,
			5,
			e.averageLatency,
			float64(time.Since(payload.TransmitTime))/float64(payload.Size()),
		)

		log.Debug("[%s] Average latency per event: %f", e.Server(), e.averageLatency)
	}

	observer.OnAck(e, payload, firstAck)

	return complete
}

// ProcessPong processes a received PONG message
func (e *Endpoint) processPong(observer Observer) {
	if !e.pongPending {
		return
	}

	log.Debug("[%s] Received PONG message", e.Server())
	e.pongPending = false

	observer.OnPong(e)
}

// NumPending returns the number of pending payloads on this endpoint
func (e *Endpoint) NumPending() int {
	return e.numPayloads
}

// IsIdle returns true if this Endpoint is idle (newly created and unused)
func (e *Endpoint) IsIdle() bool {
	return e.status == endpointStatusIdle
}

// IsClosing returns true if this Endpoint is closing down
func (e *Endpoint) IsClosing() bool {
	return e.status == endpointStatusClosing
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

// ReloadConfig submits a new configuration to the transport, and returns true
// if the transports requested that it be restarted in order for the
// configuration to take effect
func (e *Endpoint) ReloadConfig(config *config.Network, finishOnFail bool) bool {
	return e.transport.ReloadConfig(config.Factory, finishOnFail)
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

// IsReady returns true if this endpoint has been marked as ready
func (e *Endpoint) IsReady() bool {
	return e.status == endpointStatusReady
}

// resetPayloads resets the internal state for pending payloads
func (e *Endpoint) resetPayloads() {
	e.pendingPayloads = make(map[string]*payload.Payload)
	e.numPayloads = 0
}

// Pool returns the associated address pool
// This implements part of the transports.Proxy interface for callbacks
func (e *Endpoint) Pool() *addresspool.Pool {
	return e.addressPool
}

// EventChan returns the event channel transports should send events through
func (e *Endpoint) EventChan() chan<- transports.Event {
	return e.sink.eventChan
}

// ForceFailure requests that the transport force itself to fail and reset
// This is normally called as a response to a timeout or other bad behaviour
// that the Transport is likely unaware of
func (e *Endpoint) forceFailure() {
	e.transport.Fail()
}
