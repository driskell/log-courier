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

package endpoint

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/admin/api"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/internallist"
	"github.com/driskell/log-courier/lc-lib/publisher/payload"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// Context is a context key for endpoints
type Context string

const (
	// ContextSelf returns the endpoint structure from a context
	ContextSelf Context = "endpoint"
)

// Endpoint structure represents a single remote endpoint
type Endpoint struct {
	ctx context.Context

	mutex sync.RWMutex

	// The endpoint status
	status status

	api *apiEndpoint

	// Element structures for internal use by InternalList via EndpointSink
	// MUST have Value member initialised
	readyElement   internallist.Element
	failedElement  internallist.Element
	orderedElement internallist.Element

	sink            *Sink
	poolEntry       *addresspool.PoolEntry
	transport       transports.Transport
	pendingPayloads map[string]*payload.Payload
	numPayloads     int
	pongPending     bool

	lastErr           error
	lastErrTime       time.Time
	lineCount         int64
	averageLatency    float64
	transmissionStart time.Time
	estDelTime        time.Time
	warming           bool
	backoff           *core.ExpBackoff
}

// Init prepares the internal Element structures for InternalList and prepares
// the pending payload structures
func (e *Endpoint) Init() {
	e.ctx = context.WithValue(context.Background(), ContextSelf, e)

	e.warming = true
	backoffName := fmt.Sprintf("[E %s] Recovery", e.poolEntry.Desc)
	e.backoff = core.NewExpBackoff(backoffName, e.sink.config.Backoff, e.sink.config.BackoffMax)

	e.readyElement.Value = e
	e.failedElement.Value = e
	e.orderedElement.Value = e

	e.resetPayloads()

	e.transport = e.sink.config.Factory.NewTransport(e.ctx, e.PoolEntry(), e.EventChan())
}

// Context returns the endpoint's context
func (e *Endpoint) Context() context.Context {
	return e.ctx
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
	log.Debugf("[E %s] Endpoint is now shutting down", e.Server())
	e.transport.Shutdown()

	e.sink.Scheduler.SetCallback(e, e.sink.config.Timeout, func() {
		log.Warningf("[E %s] Endpoint failed to shutdown", e.Server())
		e.transport.Fail()
	})

	// Set status to closed, so we know shutdown has now been triggered
	e.mutex.Lock()
	e.status = endpointStatusClosed
	e.mutex.Unlock()
}

// Server returns the server entry this endpoint belongs to
func (e *Endpoint) Server() string {
	return e.poolEntry.Server
}

// queuePayload registers a payload with the endpoint and sends it to the
// transport
func (e *Endpoint) queuePayload(payload *payload.Payload) error {
	// Must be in a ready state
	if e.status != endpointStatusActive {
		panic(fmt.Sprintf("Endpoint is not ready (%d)", e.status))
	}

	if e.pongPending {
		e.pongPending = false
	}

	// Calculate a nonce if we don't already have one
	if payload.Nonce == "" {
		nonce := e.generateNonce()
		for {
			if _, found := e.pendingPayloads[nonce]; !found {
				break
			}
			// Collision - generate again - should be extremely rare
			nonce = e.generateNonce()
		}

		payload.Nonce = nonce
	}

	e.pendingPayloads[payload.Nonce] = payload

	e.mutex.Lock()
	e.numPayloads++
	e.updateEstDelTime()
	e.mutex.Unlock()

	if e.numPayloads == 1 {
		e.transmissionStart = time.Now()
	}

	if payload.Resending {
		log.Debugf("[E %s] Resending payload %x with %d events", e.Server(), payload.Nonce, payload.Len())
	} else {
		log.Debugf("[E %s] Sending payload %x with %d events", e.Server(), payload.Nonce, payload.Len())
	}

	if err := e.transport.SendEvents(payload.Nonce, payload.Events()); err != nil {
		return err
	}

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

// SendPing sends a ping message to the transport that it sends to the remote
// endpoint to ensure its connection is still available. Some transports may
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

// EstDelTime returns the expected time this endpoint will have delivered all of
// its events
func (e *Endpoint) EstDelTime() time.Time {
	return e.estDelTime
}

// AverageLatency returns the endpoint's average latency
func (e *Endpoint) AverageLatency() time.Duration {
	return time.Duration(e.averageLatency)
}

// ReduceLatency artificially reduces the recorded latency of the endpoint. It
// is used to ensure that really bad endpoints do not get ignored forever, as
// if events are never sent to it, the latency is never recalculated
func (e *Endpoint) ReduceLatency() {
	e.mutex.Lock()
	e.averageLatency = e.averageLatency * 0.99
	e.mutex.Unlock()
}

// updateEstDelTime updates the total expected delivery time based on the number
// of outstanding events, should be called with the mutex Lock
func (e *Endpoint) updateEstDelTime() {
	e.estDelTime = time.Now()
	for _, payload := range e.pendingPayloads {
		e.estDelTime = e.estDelTime.Add(time.Duration(e.averageLatency) * time.Duration(payload.Len()))
	}
}

// LineCount returns the endpoint's published line count
func (e *Endpoint) LineCount() int64 {
	return e.lineCount
}

// processAck processes a received acknowledgement message.
// This will pass the payload that was acked, and whether this is the first
// acknoweldgement or a later one, to the OnAck handler
// It should return whether or not the payload was completed so full status
// can be updated
func (e *Endpoint) processAck(ack transports.AckEvent, onAck func(*Endpoint, *payload.Payload, bool, int)) bool {
	// Grab the payload the ACK corresponds to by using nonce
	payload, found := e.pendingPayloads[*ack.Nonce()]
	if !found {
		// Don't fail here in case we had temporary issues and resend a payload, only for us to receive duplicate ACKN
		log.Debugf("[E %s] Duplicate/corrupt ACK received for payload %x", e.Server(), *ack.Nonce())
		return false
	}

	firstAck := !payload.HasAck()

	// Process ACK
	lineCount, complete := payload.Ack(int(ack.Sequence()))

	if complete {
		// No more events left for this payload, remove from pending list
		delete(e.pendingPayloads, *ack.Nonce())

		e.mutex.Lock()
		e.lineCount += int64(lineCount)
		e.numPayloads--

		// Mark the running average latency of this endpoint per-event over the last
		// 5 payloads
		e.averageLatency = core.CalculateRunningAverage(
			1,
			5,
			e.averageLatency,
			float64(time.Since(e.transmissionStart))/float64(payload.Len()),
		)

		e.updateEstDelTime()

		e.mutex.Unlock()

		log.Debugf("[E %s] Average latency per event: %.2f ms", e.Server(), e.averageLatency/float64(time.Millisecond))

		if e.numPayloads > 0 {
			e.transmissionStart = time.Now()
		}

		// Reset warming flag
		e.warming = false
	} else {
		e.mutex.Lock()
		e.lineCount += int64(lineCount)
		e.mutex.Unlock()
	}

	onAck(e, payload, firstAck, lineCount)

	return complete
}

// ProcessPong processes a received PONG message
func (e *Endpoint) processPong(onPong func(*Endpoint)) {
	if !e.pongPending {
		// We can ignore - we sometimes start sending again and ignore the fact we sent a PING
		return
	}

	e.pongPending = false

	onPong(e)
}

// IsWarming returns whether the endpoint is warming up or not (slow-start)
func (e *Endpoint) IsWarming() bool {
	return e.warming && e.numPayloads != 0
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

// ReloadConfig submits a new configuration to the transport, and fails
// it if it requests to be restarted, so that the new configuration can
// take effect
func (e *Endpoint) ReloadConfig(netConfig *transports.Config) {
	if e.transport.Factory().ShouldRestart(netConfig.Factory) {
		e.shutdownTransport()
	}
}

// resetPayloads resets the internal state for pending payloads
func (e *Endpoint) resetPayloads() {
	e.pendingPayloads = make(map[string]*payload.Payload)
	e.numPayloads = 0
	e.estDelTime = time.Now()
}

// Pool returns the associated address pool
// This implements part of the transports.Proxy interface for callbacks
func (e *Endpoint) PoolEntry() *addresspool.PoolEntry {
	return e.poolEntry
}

// EventChan returns the event channel transports should send events through
func (e *Endpoint) EventChan() chan<- transports.Event {
	return e.sink.eventChan
}

// LastErr returns the time the last error occurred and the error itself
// Both returned values are nil if no error is recorded
func (e *Endpoint) LastErr() (time.Time, error) {
	return e.lastErrTime, e.lastErr
}

// ForceFailure requests that the transport force itself to fail and reset
// This is normally called as a response to a timeout or other bad behaviour
// that the Transport is likely unaware of
func (e *Endpoint) forceFailure() {
	e.transport.Fail()
}

// apiEntry returns an APINavigatable that can be used to monitor this endpoint
func (e *Endpoint) apiEntry() api.Navigatable {
	if e.api == nil {
		e.api = &apiEndpoint{
			e: e,
		}
	}

	return e.api
}
