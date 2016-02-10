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
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/internallist"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// Sink structure contains the control channels that each endpoint
// will utilise. The newEndpoint method attaches new endpoints to this
type Sink struct {
	endpoints        map[string]*Endpoint
	config           *config.Network
	eventChan        chan transports.Event
	timeoutTimer     *time.Timer
	timeoutList      internallist.List
	readyList        internallist.List
	fullList         internallist.List
	failedList       internallist.List
	priorityList     internallist.List
	priorityEndpoint *Endpoint
}

// NewSink initialises a new message sink for endpoints
func NewSink(config *config.Network) *Sink {
	// TODO: Make channel sizes configurable?
	ret := &Sink{
		endpoints:    make(map[string]*Endpoint),
		config:       config,
		eventChan:    make(chan transports.Event, 10),
		timeoutTimer: time.NewTimer(1 * time.Second),
	}

	ret.timeoutTimer.Stop()

	for _, server := range config.Servers {
		ret.addEndpoint(server, addresspool.NewPool(server))
	}

	return ret
}

// ShutdownEndpoint requests the endpoint associated with the given server
// entry to shutdown
func (f *Sink) ShutdownEndpoint(server string) {
	if f.shutdownEndpoint(server) {
		// Update priority endpoint if we succeeded
		f.updatePriorityEndpoint()
	}
}

// ReloadConfig updates the configuration held by the sink for new endpoint
// creation. It also starts shutting down any endpoints that no longer exist in
// the server list
func (f *Sink) ReloadConfig(config *config.Network) {
	f.config = config

	// Verify the same servers are present
	var last *Endpoint
	for _, server := range config.Servers {
		if endpoint := f.findEndpoint(server); endpoint == nil {
			// Add a new endpoint
			last = f.addEndpoint(server, addresspool.NewPool(server))
		} else {
			// Ensure ordering
			f.moveEndpointAfter(endpoint, last)
			endpoint.ReloadConfig(config)
			last = endpoint
		}
	}

EndpointLoop:
	for server := range f.endpoints {
		for _, item := range config.Servers {
			if item == server {
				continue EndpointLoop
			}
		}

		// Not present in server list, shut down
		f.shutdownEndpoint(server)
	}

	// Update priority endpoint
	f.updatePriorityEndpoint()
}

// Shutdown signals all associated endpoints to begin shutting down
func (f *Sink) Shutdown() {
	for server := range f.endpoints {
		f.ShutdownEndpoint(server)
	}
}

// Count returns the number of associated endpoints present
func (f *Sink) Count() int {
	return len(f.endpoints)
}

// EventChan returns the event channel
// Status events and messages from endpoints pass through here for processing
func (f *Sink) EventChan() <-chan transports.Event {
	return f.eventChan
}

// TimeoutChan returns a channel which will receive the current time when
// the next endpoint hits its registered timeout
func (f *Sink) TimeoutChan() <-chan time.Time {
	return f.timeoutTimer.C
}

// ResetTimeoutTimer resets the TimeoutTimer() channel for the next timeout
func (f *Sink) ResetTimeoutTimer() {
	if f.timeoutList.Len() == 0 {
		f.timeoutTimer.Stop()
		return
	}

	endpoint := f.timeoutList.Front().Value.(*Endpoint)
	log.Debug("Timeout timer reset - due at %v for [%s]", endpoint.timeoutDue, endpoint.Server())
	f.timeoutTimer.Reset(endpoint.timeoutDue.Sub(time.Now()))
}

// RegisterTimeout registers an endpoint with a timeout and timeout callback
func (f *Sink) RegisterTimeout(endpoint *Endpoint, timeoutDue time.Time, timeoutFunc interface{}) {
	if endpoint.timeoutFunc != nil {
		// Remove existing entry
		f.timeoutList.Remove(&endpoint.timeoutElement)
	}

	endpoint.timeoutFunc = timeoutFunc
	endpoint.timeoutDue = timeoutDue

	// Add to the list in time order
	var existing *internallist.Element
	for existing = f.timeoutList.Front(); existing != nil; existing = existing.Next() {
		if existing.Value.(*Endpoint).timeoutDue.After(timeoutDue) {
			break
		}
	}

	f.ResetTimeoutTimer()
}

// NextTimeout returns the next endpoint pending a timeout
// It will also return true if there are more timeouts due
func (f *Sink) NextTimeout() (*Endpoint, interface{}, bool) {
	if f.timeoutList.Len() == 0 {
		return nil, nil, false
	}

	endpoint := f.timeoutList.Remove(f.timeoutList.Front()).(*Endpoint)
	callback := endpoint.timeoutFunc
	endpoint.timeoutFunc = nil

	next := f.timeoutList.Front()
	if next != nil && next.Value.(*Endpoint).timeoutDue.After(time.Now()) {
		// No more due after this
		return endpoint, callback, false
	}

	return endpoint, callback, true
}

// HasReady returns true if there is at least one endpoint ready to receive
// events
func (f *Sink) HasReady() bool {
	if f.config.Method == "failover" {
		if f.priorityEndpoint != nil && f.priorityEndpoint.IsReady() {
			return true
		}
		return false
	}

	return f.readyList.Len() != 0
}

// NextReady returns the next ready endpoint, in order of least pending payloads
// If in failover mode, it will only ever return the priority endpoint, unless
// it has failed in which case the next endpoint becomes priority
func (f *Sink) NextReady() *Endpoint {
	if f.config.Method == "failover" {
		if f.priorityEndpoint != nil && f.priorityEndpoint.IsReady() {
			return f.priorityEndpoint
		}
		return nil
	}

	if f.readyList.Len() == 0 {
		return nil
	}

	endpoint := f.readyList.Remove(f.readyList.Front()).(*Endpoint)
	endpoint.status = endpointStatusIdle
	return endpoint
}

// moveFull marks an endpoint as full
func (f *Sink) moveFull(endpoint *Endpoint) {
	// Ignore if we are already marked as full or were marked as failed/closing
	if endpoint.status >= endpointStatusFull {
		return
	}

	if endpoint.status == endpointStatusReady {
		f.readyList.Remove(&endpoint.readyElement)
	}

	endpoint.status = endpointStatusFull

	f.fullList.PushFront(&endpoint.fullElement)
}

// markReady marks an endpoint as ready to receive events
// but it does not move it to the ready list
func (f *Sink) markReady(endpoint *Endpoint) {
	// Ignore if already ready or if we were marked as failed/closing
	if endpoint.status == endpointStatusReady || endpoint.status >= endpointStatusFailed {
		return
	}

	if endpoint.status == endpointStatusFull {
		f.fullList.Remove(&endpoint.fullElement)
	}

	endpoint.status = endpointStatusReady
}

// moveReady moves a ready endpoint to the ready list
func (f *Sink) moveReady(endpoint *Endpoint) {
	if endpoint.status != endpointStatusReady {
		panic("Attempt to call moveReady on endpoint that is not ready")
	}

	// Least pending payloads takes preference
	var existing *internallist.Element
	for existing = f.readyList.Front(); existing != nil; existing = existing.Next() {
		if existing.Value.(*Endpoint).NumPending() > endpoint.NumPending() {
			break
		}
	}

	if existing == nil {
		f.readyList.PushBack(&endpoint.readyElement)
	} else {
		f.readyList.InsertBefore(&endpoint.readyElement, existing)
	}
}

// moveFailed stores the endpoint on the failed list, removing it from the
// ready or full lists so no more events are sent to it
func (f *Sink) moveFailed(endpoint *Endpoint) {
	// Should never get here if we're closing, caller should check IsClosing()
	if endpoint.status == endpointStatusClosing || endpoint.status == endpointStatusFailed {
		return
	}

	if endpoint.status == endpointStatusReady {
		f.readyList.Remove(&endpoint.readyElement)
	}

	if endpoint.status == endpointStatusFull {
		f.fullList.Remove(&endpoint.readyElement)
	}

	if f.priorityEndpoint == endpoint {
		// The priority endpoint has failed, update it
		f.updatePriorityEndpoint()
	}

	endpoint.status = endpointStatusFailed

	f.failedList.PushFront(&endpoint.failedElement)
}

// ForceFailure forces an endpoint to fail
func (f *Sink) ForceFailure(endpoint *Endpoint) {
	f.moveFailed(endpoint)
	endpoint.forceFailure()
}

// recoverFailed removes an endpoint from the failed list and marks it ready
func (f *Sink) recoverFailed(endpoint *Endpoint) {
	// Should never get here if we're closing, caller should check IsClosing()
	if endpoint.status == endpointStatusClosing {
		return
	}

	// Ignore if we haven't failed
	if endpoint.status != endpointStatusFailed {
		return
	}

	// Update the priority endpoint in case this recovered one is higher priority
	f.updatePriorityEndpoint()

	endpoint.status = endpointStatusIdle

	f.failedList.Remove(&endpoint.failedElement)

	f.markReady(endpoint)
}

// IsPriorityEndpoint returns true if the given endpoint is the current priority
// endpoint
func (f *Sink) IsPriorityEndpoint(endpoint *Endpoint) bool {
	return endpoint == f.priorityEndpoint
}
