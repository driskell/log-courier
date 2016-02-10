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
	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/internallist"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// AddEndpoint initialises a new endpoint for a given server entry
func (f *Sink) addEndpoint(server string, addressPool *addresspool.Pool) *Endpoint {
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

// moveEndpointAfter ensures the endpoint specified appears directly after the
// requested endpoint, or at the beginning if nil
func (f *Sink) moveEndpointAfter(endpoint *Endpoint, after *Endpoint) {
	f.priorityList.MoveAfter(&endpoint.priorityElement, &after.priorityElement)
}

// RemoveEndpoint requests the endpoint associated with the given server to be
// removed from the sink
func (f *Sink) removeEndpoint(server string) {
	endpoint, ok := f.endpoints[server]
	if !ok {
		return
	}

	// Ensure shutdown was called at the minimum (probably by our own Shutdown)
	if endpoint.status != endpointStatusClosing {
		return
	}

	f.priorityList.Remove(&endpoint.priorityElement)

	delete(f.endpoints, server)
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

// updatePriorityEndpoint finds the first non-failed endpoint and marks at as
// the priority endpoint. This endpoint is used when the network method is
// "failover"
func (f *Sink) updatePriorityEndpoint() {
	var element *internallist.Element
	for element = f.priorityList.Front(); element != nil; element = element.Next() {
		endpoint := element.Value.(*Endpoint)
		if endpoint.status <= endpointStatusFull {
			f.priorityEndpoint = endpoint
			return
		}
	}

	f.priorityEndpoint = nil
}
