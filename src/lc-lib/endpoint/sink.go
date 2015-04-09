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

	timeoutList internallist.List
	readyList   internallist.List
	fullList    internallist.List
	failedList  internallist.List
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

	return endpoint
}

// Endpoint returns the endpoint associated with the given server entry, or nil
// if no endpoint is associated
func (f *Sink) Endpoint(server string) *Endpoint {
	endpoint, ok := f.endpoints[server]
	if !ok {
		return nil
	}
	return endpoint
}

// RemoveEndpoint requests th endpoint associated with the given server to be
// removed from the sink
func (f *Sink) RemoveEndpoint(server string) {
	endpoint, ok := f.endpoints[server]
	if !ok {
		return
	}

	// Ensure shutdown was called
	if !endpoint.isShuttingDown() {
		return
	}

	delete(f.endpoints, server)
}

// ShutdownEndpoint requests the endpoint associated with the given server
// entry to shut down
func (f *Sink) ShutdownEndpoint(server string) {
	endpoint := f.Endpoint(server)
	if endpoint == nil {
		return
	}

	// Ensure we're in a valid state - that is, no pending payloads
	if endpoint.NumPending() != 0 {
		return
	}

	if endpoint.status == endpointStatusReady {
		f.readyList.Remove(&endpoint.readyElement)
	} else if endpoint.status == endpointStatusFailed {
		f.failedList.Remove(&endpoint.failedElement)
	}

	if endpoint.timeoutFunc != nil {
		f.timeoutList.Remove(&endpoint.timeoutElement)
	}

	endpoint.shutdown()
}

// ShutdownIfMissing shutsdown associated endpoints if their server is not
// present in the given array
func (f *Sink) ShutdownIfMissing(list []string) {
EndpointLoop:
	for server := range f.endpoints {
		for _, item := range list {
			if item == server {
				continue EndpointLoop
			}
		}

		// Not present in server list, shut down
		f.ShutdownEndpoint(server)
	}
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
	return f.readyList.Len() != 0
}

// NextReady returns the next ready endpoint, in order of least pending payloads
func (f *Sink) NextReady() *Endpoint {
	endpoint := f.readyList.Remove(f.readyList.Front()).(*Endpoint)
	endpoint.status = endpointStatusIdle
	return endpoint
}

// RegisterFull marks an endpoint as full
func (f *Sink) RegisterFull(endpoint *Endpoint) {
	// Ignore if we are already marked as full or were marked as failed
	if endpoint.status == endpointStatusFull || endpoint.status == endpointStatusFailed {
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
	// Ignore if already ready or if we were marked as failed
	if endpoint.status == endpointStatusReady || endpoint.status == endpointStatusFailed {
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
	if endpoint.status == endpointStatusReady {
		f.readyList.Remove(&endpoint.readyElement)
	}

	if endpoint.status == endpointStatusFull {
		f.fullList.Remove(&endpoint.readyElement)
	}

	endpoint.status = endpointStatusFailed

	f.failedList.PushFront(&endpoint.failedElement)
}

// RecoverFailed removes an endpoint from the failed list and returns it to the
// idle status, as soon as the next Ready signal is received, events will flow
// again
func (f *Sink) RecoverFailed(endpoint *Endpoint) {
	// Ignore if we haven't failed
	if endpoint.status != endpointStatusFailed {
		return
	}

	endpoint.status = endpointStatusIdle

	f.failedList.Remove(&endpoint.failedElement)
}
