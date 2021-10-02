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

import (
	"time"

	"github.com/driskell/log-courier/lc-lib/publisher/payload"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// CanQueue returns true if there are active endpoints ready to receive events
func (s *Sink) CanQueue() bool {
	return s.readyList.Len() > 0
}

// QueuePayload locates the best endpoint to send the events to, and attempts to
// queue the events on that endpoint.
// Returns the best endpoint and any error that occurred sending the events.
func (s *Sink) QueuePayload(payload *payload.Payload) (*Endpoint, error) {
	// Single endpoint?
	if s.readyList.Len() == 1 {
		endpoint := s.readyList.Front().Value.(*Endpoint)
		return endpoint, endpoint.queuePayload(payload)
	}

	// Locate best
	entry := s.readyList.Front()
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

	err := bestEndpoint.queuePayload(payload)
	if err == transports.ErrCongestion {
		// The best endpoint is congested - so try to find ANY other non-congested endpoint
		// If any fails - just return so we can force its failure and try again
		// This is a last resort as generally the best endpoint should be quick and not congested
		// If the best endpoint is congested then potentially all are congested or all competing so fire at will
		for entry := s.readyList.Back(); entry != nil; entry = entry.Prev() {
			endpoint := entry.Value.(*Endpoint)

			// Skip the best endpoint that is congested
			if endpoint == bestEndpoint {
				continue;
			}

			// Skip warming endpoints
			if endpoint.IsWarming() {
				continue
			}

			err := endpoint.queuePayload(payload)
			if err != transports.ErrCongestion {
				return endpoint, err
			}
		}
	}

	return bestEndpoint, err
}

// ForceFailure forces the endpoint referenced by the context to fail
func (s *Sink) ForceFailure(endpoint *Endpoint) {
	if endpoint.IsFailed() {
		return
	}

	s.moveFailed(endpoint)
	endpoint.forceFailure()
}
