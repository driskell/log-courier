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
	"github.com/driskell/log-courier/lc-lib/admin/api"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/internallist"
	"github.com/driskell/log-courier/lc-lib/payload"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// Endpoint structure represents a single remote endpoint
type Endpoint struct {
	mutex sync.RWMutex

	// The endpoint status
	status status

	api *apiEndpoint

	// Element structures for internal use by InternalList via EndpointSink
	// MUST have Value member initialised
	readyElement   internallist.Element
	failedElement  internallist.Element
	orderedElement internallist.Element

	// Support scheduled task for this endpoint
	Timeout

	sink            *Sink
	server          string
	addressPool     *addresspool.Pool
	finishOnFail    bool
	transport       transports.Transport
	pendingPayloads map[string]*payload.Payload
	numPayloads     int
	pongPending     bool

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
	e.warming = true
	e.backoff = core.NewExpBackoff(e.server+" Failure", e.sink.config.Backoff, e.sink.config.BackoffMax)

	e.readyElement.Value = e
	e.failedElement.Value = e
	e.orderedElement.Value = e

	e.InitTimeout()

	e.resetPayloads()

	e.transport = e.sink.config.Factory.NewTransport(e, e.Pool(), e.EventChan(), e.finishOnFail)
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

	// Set status to closed, so we know shutdown has now been triggered
	e.status = endpointStatusClosed
}

// Server returns the server string from the configuration file that this
// endpoint is associated with
func (e *Endpoint) Server() string {
	return e.server
}

// queuePayload registers a payload with the endpoint and sends it to the
// transport
func (e *Endpoint) queuePayload(payload *payload.Payload) error {
	// Must be in a ready state
	if e.status != endpointStatusActive {
		panic(fmt.Sprintf("Endpoint is not ready (%d)", e.status))
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
	e.mutex.Unlock()

	e.updateEstDelTime()

	if e.numPayloads == 1 {
		e.transmissionStart = time.Now()
	}

	if payload.Resending {
		log.Debug("[%s] Resending payload %x (%d events)", e.Server(), payload.Nonce, payload.Size())
	} else {
		log.Debug("[%s] Sending payload %x (%d events)", e.Server(), payload.Nonce, payload.Size())
	}

	if err := e.transport.Write(payload); err != nil {
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
		e.estDelTime.Add(time.Duration(e.averageLatency) * time.Duration(payload.Size()))
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
func (e *Endpoint) processAck(ack *transports.AckEvent, onAck func(*Endpoint, *payload.Payload, bool, int)) bool {
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
	lineCount, complete := payload.Ack(int(ack.Sequence()))

	if complete {
		// No more events left for this payload, remove from pending list
		delete(e.pendingPayloads, ack.Nonce())

		e.mutex.Lock()
		e.lineCount += int64(lineCount)
		e.numPayloads--

		// Mark the running average latency of this endpoint per-event over the last
		// 5 payloads
		e.averageLatency = core.CalculateRunningAverage(
			1,
			5,
			e.averageLatency,
			float64(time.Since(e.transmissionStart))/float64(payload.Size()),
		)

		e.updateEstDelTime()

		e.mutex.Unlock()

		log.Debug("[%s] Average latency per event: %.2f ms", e.Server(), e.averageLatency/float64(time.Millisecond))

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

	log.Debug("[%s] Received PONG message", e.Server())
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

// ReloadConfig submits a new configuration to the transport, and returns true
// if the transports requested that it be restarted in order for the
// configuration to take effect
func (e *Endpoint) ReloadConfig(netConfig *transports.Config, finishOnFail bool) bool {
	return e.transport.ReloadConfig(netConfig, finishOnFail)
}

// resetPayloads resets the internal state for pending payloads
func (e *Endpoint) resetPayloads() {
	e.pendingPayloads = make(map[string]*payload.Payload)
	e.numPayloads = 0
	e.estDelTime = time.Now()
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

// apiEntry returns an APINavigatable that can be used to monitor this endpoint
func (e *Endpoint) apiEntry() api.Navigatable {
	if e.api == nil {
		e.api = &apiEndpoint{
			e: e,
		}
	}

	return e.api
}
