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

// markActive marks an idle endpoint as active and puts it on the ready list
func (f *Sink) markActive(endpoint *Endpoint, observer Observer) {
	// Ignore if not idle
	if !endpoint.IsIdle() {
		return
	}

	log.Debug("[%s] Endpoint is ready", endpoint.Server())

	endpoint.mutex.Lock()
	endpoint.status = endpointStatusActive
	endpoint.mutex.Unlock()

	f.readyList.PushBack(&endpoint.readyElement)

	observer.OnStarted(endpoint)
}

// moveFailed stores the endpoint on the failed list, removing it from the
// ready list so no more events are sent to it
func (f *Sink) moveFailed(endpoint *Endpoint, observer Observer) {
	// Should never get here if we're closing, caller should check IsClosing()
	if !endpoint.IsAlive() {
		return
	}

	log.Info("[%s] Marking endpoint as failed", endpoint.Server())

	if endpoint.IsActive() {
		f.readyList.Remove(&endpoint.readyElement)
	}

	shutdown := endpoint.IsClosing()

	endpoint.mutex.Lock()
	endpoint.status = endpointStatusFailed
	endpoint.averageLatency = 0
	endpoint.mutex.Unlock()

	f.failedList.PushFront(&endpoint.failedElement)

	// endpoint.ForceFailure has no observer and calls with nil
	if observer != nil {
		observer.OnFail(endpoint)
	}

	// If we're shutting down, give up and complete transport shutdown
	if shutdown {
		endpoint.shutdownTransport()
	}
}

// recoverFailed removes an endpoint from the failed list and marks it active
func (f *Sink) recoverFailed(endpoint *Endpoint, observer Observer) {
	// Ignore if we haven't failed
	if !endpoint.IsFailed() {
		return
	}

	endpoint.mutex.Lock()
	endpoint.status = endpointStatusIdle
	endpoint.mutex.Unlock()

	f.failedList.Remove(&endpoint.failedElement)

	backoff := endpoint.backoff.Trigger()
	log.Info("[%s] Endpoint has recovered - will resume in %v", endpoint.Server(), backoff)

	// Backoff before allowing recovery
	f.RegisterTimeout(
		&endpoint.Timeout,
		backoff,
		func() {
			f.markActive(endpoint, observer)
		},
	)
}

// APINavigatable returns an APINavigatable that exposes status information for this sink
// It should be called BEFORE adding any endpoints as existing endpoints will
// not automatically become monitored
func (f *Sink) APINavigatable() admin.APINavigatable {
	if f.api == nil {
		f.api = &admin.APIArray{}
	}

	return f.api
}
