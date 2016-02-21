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
	// OnReady is called when an endpoint enters a ready state
	OnReady(*Endpoint)
	// OnFinished is called when an endpoint finishes and is removed
	// Returning false prevents the endpoint from being recreated, which it will
	// be if it still exists in the configuration
	OnFinish(*Endpoint) bool
	// OnFail is called when the endpoint fails
	OnFail(*Endpoint)
	// OnRecovered is called when an endpoint recovers from failure
	OnRecovered(*Endpoint)
	// OnAck is called when an acknowledgement response is received
	// The payload is given and the second argument is true if this ack is the
	// first ack for this payload
	OnAck(*Endpoint, *payload.Payload, bool)
	// OnPong is called when a pong response is received from the endpoint
	OnPong(*Endpoint)
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
	case transports.Ready:
		// Ignore late messages from a closing or failing transport
		// A transport may have multiple routines that are still shutting down after
		// a failure and one of those may still be sending ready events
		if endpoint.IsClosing() || endpoint.IsFailed() {
			break
		}

		// Full?
		if endpoint.NumPending() >= int(s.config.MaxPendingPayloads) {
			log.Debug("[%s] Endpoint is full (%d pending payloads)", endpoint.Server(), endpoint.NumPending())
			s.moveFull(endpoint)
			break
		}

		// Mark endpoint as ready and call the observer
		s.markReady(endpoint)
		observer.OnReady(endpoint)

		// If the endpoint is still ready, nothing was sent, add to ready list
		if endpoint.status == endpointStatusReady {
			s.moveReady(endpoint)
		}
	case transports.Failed:
		if endpoint.IsFailed() {
			break
		}

		shutdown := endpoint.IsClosing()

		log.Info("[%s] Marking endpoint as failed", endpoint.Server())
		s.moveFailed(endpoint)
		observer.OnFail(endpoint)

		// If we're shutting down, give up and complete transport shutdown
		if shutdown {
			endpoint.shutdownTransport()
		}
	case transports.Recovered:
		// Allow idle state to also use Recovered, as idle transports always have
		// zero pending payloads as they've only just been created
		// This simplifies the transport logic a little
		if endpoint.IsFailed() {
			log.Info("[%s] Endpoint recovered", endpoint.Server())
			s.recoverFailed(endpoint)
			observer.OnRecovered(endpoint)
		} else if endpoint.IsIdle() {
			// Mark as ready
			s.markReady(endpoint)
		} else {
			break
		}

		observer.OnReady(endpoint)

		// If the endpoint is still ready, nothing was sent, add to ready list
		if endpoint.status == endpointStatusReady {
			s.moveReady(endpoint)
		}
	case transports.Finished:
		server := endpoint.Server()
		log.Debug("[%s] Endpoint has finished", server)
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
	if endpoint.IsClosing() && endpoint.NumPending() == 0 {
		endpoint.shutdownTransport()
		return
	}

	// Were we full? Are we ready again?
	if endpoint.IsFull() && endpoint.NumPending() < int(s.config.MaxPendingPayloads) {
		s.markReady(endpoint)
		observer.OnReady(endpoint)

		// If the endpoint is still ready, nothing was sent, add to ready list
		if endpoint.status == endpointStatusReady {
			s.moveReady(endpoint)
		}
	}
}
