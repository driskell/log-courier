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
	"github.com/driskell/log-courier/lc-lib/payload"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// Observer is the interface implemented by the observer of the sink and will
// receive callbacks on status changes it needs to action
type Observer interface {
	// OnAck is called when an acknowledgement response is received
	// The payload is given and the second argument is true if this ack is the
	// first ack for this payload
	OnAck(*Endpoint, *payload.Payload, bool, int)
	// OnFail is called when the endpoint fails
	OnFail(*Endpoint)
	// OnFinished is called when an endpoint finishes and is removed
	// Returning false prevents the endpoint from being recreated, which it will
	// be if it still exists in the configuration
	OnFinish(*Endpoint) bool
	// OnPong is called when a pong response is received from the endpoint
	OnPong(*Endpoint)
	// OnStarted is called when an endpoint starts up and is ready
	OnStarted(*Endpoint)
}

// EventChan returns the event channel
// Status events and messages from endpoints pass through here for processing
func (s *Sink) EventChan() <-chan transports.Event {
	return s.eventChan
}

// ProcessEvent performs the necessary processing of events
func (s *Sink) ProcessEvent(event transports.Event, observer Observer) {
	endpoint := event.Observer().(*Endpoint)

	switch msg := event.(type) {
	case *transports.StatusEvent:
		s.processStatusChange(msg, endpoint, observer)
	case *transports.AckEvent:
		s.processAck(msg, endpoint, observer)
	case *transports.PongEvent:
		endpoint.processPong(observer)
	default:
		panic("Invalid transport event received")
	}
}

// processStatusChange handles status change events
func (s *Sink) processStatusChange(status *transports.StatusEvent, endpoint *Endpoint, observer Observer) {
	switch status.StatusChange() {
	case transports.Failed:
		s.moveFailed(endpoint, observer)
	case transports.Started:
		if endpoint.IsFailed() {
			s.recoverFailed(endpoint, observer)
			break
		}

		// Mark as active
		s.markActive(endpoint, observer)
	case transports.Finished:
		server := endpoint.Server()
		s.removeEndpoint(server)

		// Is it still in the config?
		for _, item := range s.config.Servers {
			if item != server {
				continue
			}

			// Still in the config, ask the observer if we should re-add it
			if observer.OnFinish(endpoint) {
				s.AddEndpoint(server, addresspool.NewPool(server), endpoint.finishOnFail)
			}
			break
		}
	default:
		panic("Invalid transport status code received")
	}
}

func (s *Sink) processAck(ack *transports.AckEvent, endpoint *Endpoint, observer Observer) {
	complete := endpoint.processAck(ack, observer)

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
