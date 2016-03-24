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
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/admin"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/internallist"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// Sink structure contains the control channels that each endpoint
// will utilise. The newEndpoint method attaches new endpoints to this
type Sink struct {
	mutex sync.RWMutex

	endpoints    map[string]*Endpoint
	config       *config.Network
	eventChan    chan transports.Event
	timeoutTimer *time.Timer

	api *admin.APIArray

	timeoutList internallist.List
	readyList   internallist.List
	fullList    internallist.List
	failedList  internallist.List
	orderedList internallist.List
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

	return ret
}

// ReloadConfig loads in a new configuration, endpoints will be shutdown if they
// are no longer in the configuration
func (f *Sink) ReloadConfig(config *config.Network) {
	// TODO: If MaxPendingPayloads is changed, update which endpoints should
	//       be marked as full
EndpointLoop:
	for endpoint := f.Front(); endpoint != nil; endpoint = endpoint.Next() {
		var server string
		for _, server = range config.Servers {
			if server == endpoint.Server() {
				continue EndpointLoop
			}
		}

		// Not present in server list anymore, shut down
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

// Front returns the first endpoint currently active
func (f *Sink) Front() *Endpoint {
	if f.orderedList.Front() == nil {
		return nil
	}
	return f.orderedList.Front().Value.(*Endpoint)
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

// resetTimeoutTimer resets the TimeoutTimer() channel for the next timeout
func (f *Sink) resetTimeoutTimer() {
	if f.timeoutList.Len() == 0 {
		f.timeoutTimer.Stop()
		return
	}

	timeout := f.timeoutList.Front().Value.(*Timeout)
	log.Debug("Timeout timer reset - due at %v", timeout.timeoutDue)
	f.timeoutTimer.Reset(timeout.timeoutDue.Sub(time.Now()))
}

// HasReady returns true if there is at least one endpoint ready to receive
// events
func (f *Sink) HasReady() bool {
	return f.readyList.Len() != 0
}

// NextReady returns the next ready endpoint, in order of least pending payloads
func (f *Sink) NextReady() *Endpoint {
	if f.readyList.Len() == 0 {
		return nil
	}

	endpoint := f.readyList.Remove(f.readyList.Front()).(*Endpoint)
	return endpoint
}

// moveFull marks an endpoint as full
func (f *Sink) moveFull(endpoint *Endpoint) {
	// Ignore if we are already marked as full or were marked as failed/closing
	if !endpoint.IsNotFull() {
		return
	}

	if endpoint.isReady {
		f.readyList.Remove(&endpoint.readyElement)
		endpoint.isReady = false
	}

	endpoint.mutex.Lock()
	endpoint.status = endpointStatusFull
	endpoint.mutex.Unlock()

	f.fullList.PushFront(&endpoint.fullElement)
}

// markActiveAndReady marks an idle endpoint as active and automatically moves
// it to the ready state (but not onto the ready list)
func (f *Sink) markActiveAndReady(endpoint *Endpoint) {
	// Ignore if not idle
	if !endpoint.IsIdle() {
		return
	}

	endpoint.mutex.Lock()
	endpoint.status = endpointStatusActive
	endpoint.mutex.Unlock()

	f.markReady(endpoint)
}

// markReady marks an endpoint as ready to receive event but it does not move it
// to the ready list
func (f *Sink) markReady(endpoint *Endpoint) {
	// Ignore if already ready or if we were marked as failed/closing
	if endpoint.isReady || !endpoint.IsAlive() {
		return
	}

	if endpoint.IsFull() {
		f.fullList.Remove(&endpoint.fullElement)
	}

	endpoint.isReady = true
}

// moveReady moves a ready endpoint to the ready list
func (f *Sink) moveReady(endpoint *Endpoint) {
	if !endpoint.isReady {
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
	if !endpoint.IsAlive() {
		return
	}

	if endpoint.isReady {
		f.readyList.Remove(&endpoint.readyElement)
		endpoint.isReady = false
	}

	if endpoint.IsFull() {
		f.fullList.Remove(&endpoint.readyElement)
	}

	endpoint.mutex.Lock()
	endpoint.status = endpointStatusFailed
	endpoint.averageLatency = 0
	endpoint.mutex.Unlock()

	f.failedList.PushFront(&endpoint.failedElement)
}

// ForceFailure forces an endpoint to fail
func (f *Sink) ForceFailure(endpoint *Endpoint) {
	f.moveFailed(endpoint)
	endpoint.forceFailure()
}

// recoverFailed removes an endpoint from the failed list and marks it ready
func (f *Sink) recoverFailed(endpoint *Endpoint) {
	// Ignore if we haven't failed
	if !endpoint.IsFailed() {
		return
	}

	endpoint.mutex.Lock()
	endpoint.status = endpointStatusIdle
	endpoint.mutex.Unlock()

	f.failedList.Remove(&endpoint.failedElement)
	f.markReady(endpoint)
}

// APIEntry returns an APIEntry that exposes status information for this sink
// It should be called BEFORE adding any endpoints as existing endpoints will
// not automatically become monitored
func (f *Sink) APIEntry() admin.APIEntry {
	if f.api == nil {
		f.api = &admin.APIArray{}
	}

	return f.api
}
