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
	"github.com/driskell/log-courier/src/lc-lib/core"
	"github.com/driskell/log-courier/src/lc-lib/internallist"
	"time"
)

// Failure structure contains the reason for failure
type Failure struct {
	Endpoint *Endpoint
	Error    error
}

// Sink structure contains the control channels that each endpoint
// will utilise. The newEndpoint method attaches new endpoints to this
type Sink struct {
	endpoints    map[string]*Endpoint
	config       *core.NetworkConfig

	readyChan    chan *Endpoint
	responseChan chan *Response
	failChan     chan *Failure

	timeoutTimer *time.Timer

	timeoutList internallist.List
	readyList   internallist.List
	fullList    internallist.List
}

// NewSink initialises a new message sink for endpoints
func NewSink(config *core.NetworkConfig) *Sink {
	// TODO: Make channel sizes configurable?
	ret := &Sink{
		endpoints:    make(map[string]*Endpoint),
		config:       config,

		readyChan:    make(chan *Endpoint, 10),
		responseChan: make(chan *Response, 10),
		failChan:     make(chan *Failure, 10),

		timeoutTimer: time.NewTimer(1 * time.Second),
	}

	ret.timeoutTimer.Stop()

	return ret
}

// AddEndpoint initialises a new endpoint for a given server entry
func (f *Sink) AddEndpoint(server string, addressPool *addresspool.Pool) *Endpoint {
	endpoint := &Endpoint{
		server:      server,
		addressPool: addressPool,
	}

	remote := &Remote{
		sink:        f,
		endpoint:    endpoint,
		AddressPool: addressPool,
	}

	endpoint.remote = remote
	endpoint.transport = f.config.TransportFactory.NewTransport(remote)
	endpoint.init()

	f.endpoints[server] = endpoint

	return endpoint
}

// Shutdown signals all associated endpoints to begin shutting down
func (f *Sink) Shutdown() {
	for _, endpoint := range f.endpoints {
		endpoint.shutdown()
	}
}

// Wait for associated endpoints to complete shutting down
func (f *Sink) Wait() {
	for _, endpoint := range f.endpoints {
		endpoint.wait()
	}
}

// ReadyChan returns the readiness channel
// Endpoints that appear from this channel can be saved to a pending ready list
// using RegisterReady, or to a full list using RegisterFull
func (f *Sink) ReadyChan() <-chan *Endpoint {
	return f.readyChan
}

// ResponseChan returns the response channel
// All responses received from endpoints are sent through here
func (f *Sink) ResponseChan() <-chan *Response {
	return f.responseChan
}

// FailChan returns the failure channel
// Failed endpoints will send themselves through this channel along with the
// reason for failure
// TODO: Document handling of this
func (f *Sink) FailChan() <-chan *Failure {
	return f.failChan
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
	endpoint.ready = false
	return endpoint
}

// RegisterFull marks an endpoint as full
func (f *Sink) RegisterFull(endpoint *Endpoint) {
	if endpoint.full {
		return
	}

	if endpoint.ready {
		endpoint.ready = false
		f.readyList.Remove(&endpoint.readyElement)
	}

	endpoint.full = true

	f.fullList.PushFront(&endpoint.fullElement)
}

// RegisterReady marks an endpoint as ready to receive events
func (f *Sink) RegisterReady(endpoint *Endpoint) {
	if endpoint.ready {
		return
	}

	if endpoint.full {
		endpoint.full = false
		f.fullList.Remove(&endpoint.fullElement)
	}

	endpoint.ready = true

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
