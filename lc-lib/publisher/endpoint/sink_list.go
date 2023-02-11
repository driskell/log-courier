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
func (s *Sink) addEndpoint(poolEntry *addresspool.PoolEntry) *Endpoint {
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
		poolEntry:      poolEntry,
		averageLatency: initialLatency,
	}

	endpoint.Init()

	s.endpoints[poolEntry.Desc] = endpoint
	return endpoint
}

// AddEndpoint initialises a new endpoint for a given server entry and adds it
// to the back of the list of endpoints
func (s *Sink) AddEndpoint(server *addresspool.PoolEntry) *Endpoint {
	endpoint := s.addEndpoint(server)

	s.mutex.Lock()
	s.orderedList.PushBack(&endpoint.orderedElement)
	if s.api != nil {
		s.api.AddEntry(server.Desc, endpoint.apiEntry())
	}
	s.mutex.Unlock()
	return endpoint
}

// AddEndpointAfter initialises a new endpoint for a given server entry and adds
// it in the list to the position after the given endpoint. If the given
// endpoint is nil it is added at the front
func (s *Sink) AddEndpointAfter(server *addresspool.PoolEntry, after *Endpoint) *Endpoint {
	endpoint := s.addEndpoint(server)

	s.mutex.Lock()
	if after == nil {
		s.orderedList.PushFront(&endpoint.orderedElement)
	} else {
		s.orderedList.InsertAfter(&endpoint.orderedElement, &after.orderedElement)
	}
	if s.api != nil {
		s.api.AddEntry(server.Desc, endpoint.apiEntry())
	}
	s.mutex.Unlock()
	return endpoint
}

// FindEndpoint returns the endpoint associated with the given server entry, or
// nil if no endpoint is associated
func (s *Sink) FindEndpoint(server *addresspool.PoolEntry) *Endpoint {
	endpoint, ok := s.endpoints[server.Desc]
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
		s.orderedList.MoveToFront(&endpoint.orderedElement)
		s.mutex.Unlock()
		return
	}

	s.mutex.Lock()
	s.orderedList.MoveAfter(&endpoint.orderedElement, &after.orderedElement)
	s.mutex.Unlock()
}

// RemoveEndpoint requests the endpoint be removed from the sink
func (s *Sink) removeEndpoint(endpoint *Endpoint) {
	log.Debugf("[E %s] Endpoint has finished", endpoint.poolEntry.Desc)

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
		s.api.RemoveEntry(endpoint.poolEntry.Desc)
	}
	s.mutex.Unlock()

	delete(s.endpoints, endpoint.poolEntry.Desc)
}

// ShutdownEndpoint requests the endpoint associated with the given server
// entry to shutdown, returning false if the endpoint could not be shutdown
func (s *Sink) ShutdownEndpoint(endpoint *Endpoint) bool {
	if endpoint == nil || endpoint.IsClosing() {
		return false
	}

	if endpoint.IsActive() {
		s.readyList.Remove(&endpoint.readyElement)
	} else if endpoint.IsFailed() {
		s.failedList.Remove(&endpoint.failedElement)
	}

	// If we still have pending payloads wait for them to finish
	if endpoint.NumPending() != 0 {
		endpoint.mutex.Lock()
		endpoint.status = endpointStatusClosing
		endpoint.mutex.Unlock()
		return true
	}

	endpoint.shutdownTransport()

	endpoint.mutex.Lock()
	endpoint.status = endpointStatusClosed
	endpoint.mutex.Unlock()

	return true
}
