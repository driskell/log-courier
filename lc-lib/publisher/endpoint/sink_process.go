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
	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// EventChan returns the event channel
// Status events and messages from endpoints pass through here for processing
func (s *Sink) EventChan() <-chan transports.Event {
	return s.eventChan
}

// ProcessEvent performs the necessary processing of events
func (s *Sink) ProcessEvent(event transports.Event) bool {
	endpoint := event.Context().Value(ContextSelf).(*Endpoint)

	switch msg := event.(type) {
	case *transports.StatusEvent:
		s.processStatusChange(msg, endpoint)
	case *transports.AckEvent:
		s.processAck(msg, endpoint)
	case *transports.PongEvent:
		endpoint.processPong(s.OnPong)
	default:
		return false
	}

	return true
}

// processStatusChange handles status change events
func (s *Sink) processStatusChange(status *transports.StatusEvent, endpoint *Endpoint) {
	switch status.StatusChange() {
	case transports.Failed:
		s.moveFailed(endpoint)
	case transports.Started:
		if endpoint.IsFailed() {
			s.recoverFailed(endpoint)
			break
		}

		// Mark as active
		s.markActive(endpoint)
	case transports.Finished:
		server := endpoint.Server()
		s.removeEndpoint(server)

		// Is it still in the config?
		for _, item := range s.config.Servers {
			if item != server {
				continue
			}

			// Still in the config, ask the OnFinish handler if we should re-add it
			if s.OnFinish(endpoint) {
				s.AddEndpoint(server, addresspool.NewPool(server), endpoint.finishOnFail)
			}
			break
		}
	default:
		panic("Invalid transport status code received")
	}
}

func (s *Sink) processAck(ack *transports.AckEvent, endpoint *Endpoint) {
	complete := endpoint.processAck(ack, s.OnAck)

	// Everything after here runs when a payload is fully completed
	if !complete {
		return
	}

	// Do we need to finish shutting down?
	if !endpoint.IsClosing() || endpoint.NumPending() > 0 {
		return
	}

	endpoint.shutdownTransport()
}
