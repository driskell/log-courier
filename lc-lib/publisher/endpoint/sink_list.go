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

import "github.com/driskell/log-courier/lc-lib/addresspool"

// Count returns the number of associated endpoints present
func (s *Sink) Count() int {
	return len(s.endpoints)
}

// Front returns the first endpoint currently active
func (s *Sink) Front() *Endpoint {
	if s.orderedList.Front() == nil {
		return nil
	}
	return s.orderedList.Front().Value.(*Endpoint)
}

// addEndpoint initialises a new endpoint
func (s *Sink) addEndpoint(server string, addressPool *addresspool.Pool, finishOnFail bool) *Endpoint {
	var initialLatency float64

	if s.readyList.Len() == 0 {
		// No endpoints ready currently, use initial 0
		initialLatency = 0
	} else {
		// Use slightly over average so we don't slow down the fastest
		for entry := s.readyList.Front(); entry != nil; entry = entry.Next() {
			initialLatency = initialLatency + float64(entry.Value.(*Endpoint).AverageLatency())
		}
		initialLatency = initialLatency / float64(s.readyList.Len()) * 1.01
	}

	endpoint := &Endpoint{
		sink:           s,
		server:         server,
		addressPool:    addressPool,
		finishOnFail:   finishOnFail,
		averageLatency: initialLatency,
	}

	endpoint.Init()

	s.endpoints[server] = endpoint
	return endpoint
}

// AddEndpoint initialises a new endpoint for a given server entry and adds it
// to the back of the list of endpoints
func (s *Sink) AddEndpoint(server string, addressPool *addresspool.Pool, finishOnFail bool) *Endpoint {
	endpoint := s.addEndpoint(server, addressPool, finishOnFail)

	s.mutex.Lock()
	s.orderedList.PushBack(&endpoint.orderedElement)
	if s.api != nil {
		s.api.AddEntry(server, endpoint.apiEntry())
	}
	s.mutex.Unlock()
	return endpoint
}

// AddEndpointAfter initialises a new endpoint for a given server entry and adds
// it in the list to the position after the given endpoint. If the given
// endpoint is nil it is added at the front
func (s *Sink) AddEndpointAfter(server string, addressPool *addresspool.Pool, finishOnFail bool, after *Endpoint) *Endpoint {
	endpoint := s.addEndpoint(server, addressPool, finishOnFail)

	s.mutex.Lock()
	if after == nil {
		s.orderedList.PushFront(&endpoint.orderedElement)
	} else {
		s.orderedList.MoveAfter(&endpoint.orderedElement, &after.orderedElement)
	}
	if s.api != nil {
		s.api.AddEntry(server, endpoint.apiEntry())
	}
	s.mutex.Unlock()
	return endpoint
}

// FindEndpoint returns the endpoint associated with the given server entry, or
// nil if no endpoint is associated
func (s *Sink) FindEndpoint(server string) *Endpoint {
	endpoint, ok := s.endpoints[server]
	if !ok {
		return nil
	}
	return endpoint
}

// MoveEndpointAfter ensures the endpoint specified appears directly after the
// requested endpoint, or at the beginning if nil
func (s *Sink) MoveEndpointAfter(endpoint *Endpoint, after *Endpoint) {
	if after == nil {
		s.mutex.Lock()
		s.orderedList.PushFront(&endpoint.orderedElement)
		s.mutex.Unlock()
		return
	}

	s.mutex.Lock()
	s.orderedList.MoveAfter(&endpoint.orderedElement, &after.orderedElement)
	s.mutex.Unlock()
}

// RemoveEndpoint requests the endpoint associated with the given server to be
// removed from the sink
func (s *Sink) removeEndpoint(server string) {
	endpoint, ok := s.endpoints[server]
	if !ok {
		return
	}

	log.Debug("[%s] Endpoint has finished", server)

	// Ensure we are correctly removed from all lists
	if endpoint.IsActive() {
		s.readyList.Remove(&endpoint.readyElement)
	} else if endpoint.IsFailed() {
		s.failedList.Remove(&endpoint.failedElement)
	}

	s.Scheduler.Remove(endpoint)

	s.mutex.Lock()
	s.orderedList.Remove(&endpoint.orderedElement)
	if s.api != nil {
		s.api.RemoveEntry(server)
	}
	s.mutex.Unlock()

	delete(s.endpoints, server)
}

// ShutdownEndpoint requests the endpoint associated with the given server
// entry to shutdown, returning false if the endpoint could not be shutdown
func (s *Sink) ShutdownEndpoint(server string) bool {
	endpoint := s.FindEndpoint(server)
	if endpoint == nil || endpoint.IsClosing() {
		return false
	}

	if endpoint.IsActive() {
		s.readyList.Remove(&endpoint.readyElement)
	} else if endpoint.IsFailed() {
		s.failedList.Remove(&endpoint.failedElement)
	}

	endpoint.mutex.Lock()
	endpoint.status = endpointStatusClosing
	endpoint.mutex.Unlock()

	// If we still have pending payloads wait for them to finish
	if endpoint.NumPending() != 0 {
		return true
	}

	s.Scheduler.Remove(endpoint)
	endpoint.shutdownTransport()

	return true
}
