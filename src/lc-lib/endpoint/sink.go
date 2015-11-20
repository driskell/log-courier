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
	"github.com/driskell/log-courier/src/lc-lib/addresspool"
	"github.com/driskell/log-courier/src/lc-lib/config"
	"github.com/driskell/log-courier/src/lc-lib/internallist"
	"github.com/driskell/log-courier/src/lc-lib/transports"
	"time"
)

// Sink structure contains the control channels that each endpoint
// will utilise. The newEndpoint method attaches new endpoints to this
type Sink struct {
	endpoints    map[string]*Endpoint
	config       *config.Network

	statusChan   chan *Status
	responseChan chan transports.Response

	timeoutTimer *time.Timer

	timeoutList  internallist.List
	readyList    internallist.List
	fullList     internallist.List
	failedList   internallist.List
	priorityList internallist.List

	priorityEndpoint *Endpoint
}

// NewSink initialises a new message sink for endpoints
func NewSink(config *config.Network) *Sink {
	// TODO: Make channel sizes configurable?
	ret := &Sink{
		endpoints:    make(map[string]*Endpoint),
		config:       config,

		statusChan:   make(chan *Status, 10),
		responseChan: make(chan transports.Response, 10),

		timeoutTimer: time.NewTimer(1 * time.Second),
	}

	ret.timeoutTimer.Stop()

	for _, server := range config.Servers {
		ret.AddEndpoint(server, addresspool.NewPool(server))
	}

	return ret
}

// AddEndpoint initialises a new endpoint for a given server entry
func (f *Sink) AddEndpoint(server string, addressPool *addresspool.Pool) *Endpoint {
	endpoint := &Endpoint{
		sink:        f,
		server:      server,
		addressPool: addressPool,
	}

	endpoint.transport = transports.NewTransport(f.config.Factory, endpoint)
	endpoint.Init()

	f.endpoints[server] = endpoint

	if f.priorityList.Len() == 0 {
		f.priorityEndpoint = endpoint
	}

	f.priorityList.PushBack(&endpoint.priorityElement)

	return endpoint
}

// findEndpoint returns the endpoint associated with the given server entry, or
// nil if no endpoint is associated
func (f *Sink) findEndpoint(server string) *Endpoint {
	endpoint, ok := f.endpoints[server]
	if !ok {
		return nil
	}
	return endpoint
}

// MoveEndpointAfter ensures the endpoint specified appears directly after the
// requested endpoint, or at the beginning if nil
func (f *Sink) MoveEndpointAfter(endpoint *Endpoint, after *Endpoint) {
	f.priorityList.MoveAfter(&endpoint.priorityElement, &after.priorityElement)
}

// RemoveEndpoint requests th endpoint associated with the given server to be
// removed from the sink
func (f *Sink) RemoveEndpoint(server string) {
	endpoint, ok := f.endpoints[server]
	if !ok {
		return
	}

	// Ensure shutdown was called at the minimum (probably by our own Shutdown)
	if !endpoint.isShuttingDown() {
		return
	}

	f.priorityList.Remove(&endpoint.priorityElement)

	delete(f.endpoints, server)
}

// ShutdownEndpoint requests the endpoint associated with the given server
// entry to shutdown
func (f *Sink) ShutdownEndpoint(server string) {
	if f.shutdownEndpoint(server) {
		// Update priority endpoint if we succeeded
		f.updatePriorityEndpoint()
	}
}

// shutdownEndpoint requests the endpoint associated with the given server
// entry to shutdown, returning false if the endpoint could not be shutdown
func (f *Sink) shutdownEndpoint(server string) bool {
	endpoint := f.findEndpoint(server)
	if endpoint == nil || endpoint.IsClosing() {
		return false
	}

	if endpoint.status == endpointStatusReady {
		f.readyList.Remove(&endpoint.readyElement)
	} else if endpoint.status == endpointStatusFailed {
		f.failedList.Remove(&endpoint.failedElement)
	}

	endpoint.status = endpointStatusClosing

	// If we still have pending payloads wait for them to finish
	if endpoint.NumPending() != 0 {
		return true
	}

	if endpoint.timeoutFunc != nil {
		f.timeoutList.Remove(&endpoint.timeoutElement)
	}

	endpoint.shutdown()

	return true
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
			last = f.AddEndpoint(server, addresspool.NewPool(server))
		} else {
			// Ensure ordering
			f.MoveEndpointAfter(endpoint, last)
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

// ResponseChan returns the response channel
// All responses received from endpoints are sent through here
func (f *Sink) ResponseChan() <-chan transports.Response {
	return f.responseChan
}

// StatusChan returns the status channel
// Failed endpoints will send themselves through this channel along with the
// reason for failure
// Recovered endpoints will send themselves with a nil failure reason
// TODO: Document handling of this
func (f *Sink) StatusChan() <-chan *Status {
	return f.statusChan
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

// RegisterFull marks an endpoint as full
func (f *Sink) RegisterFull(endpoint *Endpoint) {
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

// RegisterReady marks an endpoint as ready to receive events
func (f *Sink) RegisterReady(endpoint *Endpoint) {
	// Ignore if already ready or if we were marked as failed/closing
	if endpoint.status == endpointStatusReady || endpoint.status >= endpointStatusFailed {
		return
	}

	if endpoint.status == endpointStatusFull {
		f.fullList.Remove(&endpoint.fullElement)
	}

	endpoint.status = endpointStatusReady

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

// RegisterFailed stores the endpoint on the failed list, removing it from the
// ready or full lists so no more events are sent to it
func (f *Sink) RegisterFailed(endpoint *Endpoint) {
	// Should never get here if we're closing, caller should check IsClosing()
	if endpoint.status == endpointStatusClosing {
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

// RecoverFailed removes an endpoint from the failed list and returns it to the
// idle status, as soon as the next Ready signal is received, events will flow
// again
func (f *Sink) RecoverFailed(endpoint *Endpoint) {
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
}

// updatePriorityEndpoint finds the first non-failed endpoint and marks at as
// the priority endpoint. This endpoint is used when the network method is
// "failover"
func (f *Sink) updatePriorityEndpoint() {
	var element *internallist.Element
	for element = f.priorityList.Front(); element != nil; element = element.Next() {
		endpoint := element.Value.(*Endpoint)
		if !endpoint.IsFailed() && !endpoint.IsClosing() {
			f.priorityEndpoint = endpoint
			return
		}
	}

	f.priorityEndpoint = nil
}

// IsPriorityEndpoint returns true if the given endpoint is the current priority
// endpoint
func (f *Sink) IsPriorityEndpoint(endpoint *Endpoint) bool {
	return endpoint == f.priorityEndpoint
}
