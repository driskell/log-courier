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

	"github.com/driskell/log-courier/lc-lib/payload"
)

// CanQueue returns true if there are active endpoints ready to receive events
func (s *Sink) CanQueue() bool {
	return f.readyList.Len() > 0
}

// QueuePayload locates the best endpoint to send the events to, and attempts to
// queue the events on that endpoint.
// Returns the best endpoint and any error that occurred sending the events.
func (s *Sink) QueuePayload(payload *payload.Payload) (*Endpoint, error) {
	// Single endpoint?
	if f.readyList.Len() == 1 {
		endpoint := f.readyList.Front().Value.(*Endpoint)
		return endpoint, endpoint.queuePayload(payload)
	}

	// Locate best
	entry := f.readyList.Front()
	if entry == nil {
		return nil, nil
	}

	events := time.Duration(payload.Size())
	bestEndpoint := entry.Value.(*Endpoint)
	bestEDT := bestEndpoint.EstDelTime().Add(bestEndpoint.AverageLatency() * events)

	for ; entry != nil; entry = entry.Next() {
		endpoint := entry.Value.(*Endpoint)

		// Warming endpoints have received their first payload and should not receive
		// any more
		if endpoint.IsWarming() {
			continue
		}

		endpointEDT := endpoint.EstDelTime().Add(endpoint.AverageLatency() * events)

		if endpointEDT.Before(bestEDT) {
			// If we continuously skip endpoints on every queue, they will never
			// recalculate their latency - so artificially drop it each time we skip
			// it so it eventually gets to send something and recalculate its latency
			bestEndpoint.ReduceLatency()

			bestEndpoint = endpoint
			bestEDT = endpointEDT
		} else {
			endpoint.ReduceLatency()
		}
	}

	return bestEndpoint, bestEndpoint.queuePayload(payload)
}

// ForceFailure forces an endpoint to fail
func (s *Sink) ForceFailure(endpoint *Endpoint) {
	if endpoint.IsFailed() {
		return
	}

	f.moveFailed(endpoint, nil)
	endpoint.forceFailure()
}
