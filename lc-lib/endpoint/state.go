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

import "github.com/driskell/log-courier/lc-lib/addresspool"

// addEndpoint initialises a new endpoint
func (f *Sink) addEndpoint(server string, addressPool *addresspool.Pool, finishOnFail bool) *Endpoint {
	endpoint := &Endpoint{
		sink:         f,
		server:       server,
		addressPool:  addressPool,
		finishOnFail: finishOnFail,
	}

	endpoint.Init()

	f.endpoints[server] = endpoint
	return endpoint
}

// AddEndpoint initialises a new endpoint for a given server entry and adds it
// to the back of the list of endpoints
func (f *Sink) AddEndpoint(server string, addressPool *addresspool.Pool, finishOnFail bool) *Endpoint {
	endpoint := f.addEndpoint(server, addressPool, finishOnFail)
	f.orderedList.PushBack(&endpoint.orderedElement)
	return endpoint
}

// AddEndpointAfter initialises a new endpoint for a given server entry and adds
// it in the list to the position after the given endpoint. If the given
// endpoint is nil it is added at the front
func (f *Sink) AddEndpointAfter(server string, addressPool *addresspool.Pool, finishOnFail bool, after *Endpoint) *Endpoint {
	endpoint := f.addEndpoint(server, addressPool, finishOnFail)
	if after == nil {
		f.orderedList.PushFront(&endpoint.orderedElement)
	} else {
		f.orderedList.MoveAfter(&endpoint.orderedElement, &after.orderedElement)
	}
	return endpoint
}

// FindEndpoint returns the endpoint associated with the given server entry, or
// nil if no endpoint is associated
func (f *Sink) FindEndpoint(server string) *Endpoint {
	endpoint, ok := f.endpoints[server]
	if !ok {
		return nil
	}
	return endpoint
}

// MoveEndpointAfter ensures the endpoint specified appears directly after the
// requested endpoint, or at the beginning if nil
func (f *Sink) MoveEndpointAfter(endpoint *Endpoint, after *Endpoint) {
	if after == nil {
		f.orderedList.PushFront(&endpoint.orderedElement)
		return
	}

	f.orderedList.MoveAfter(&endpoint.orderedElement, &after.orderedElement)
}

// RemoveEndpoint requests the endpoint associated with the given server to be
// removed from the sink
func (f *Sink) removeEndpoint(server string) {
	endpoint, ok := f.endpoints[server]
	if !ok {
		return
	}

	f.orderedList.Remove(&endpoint.orderedElement)

	delete(f.endpoints, server)
}

// ShutdownEndpoint requests the endpoint associated with the given server
// entry to shutdown, returning false if the endpoint could not be shutdown
func (f *Sink) ShutdownEndpoint(server string) bool {
	endpoint := f.FindEndpoint(server)
	if endpoint == nil || endpoint.IsClosing() {
		return false
	}

	if endpoint.status == endpointStatusReady {
		f.readyList.Remove(&endpoint.readyElement)
	} else if endpoint.status == endpointStatusFull {
		f.fullList.Remove(&endpoint.fullElement)
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

	endpoint.shutdownTransport()

	return true
}
