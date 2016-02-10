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
	// OnAck is called when an acknowledgement response is received
	// The payload is given and the second argument is true if this ack is the
	// first ack for this payload
	OnAck(*Endpoint, *payload.Payload, bool)
	// OnPong is called when a pong response is received from the endpoint
	OnPong(*Endpoint)
}

// ProcessEvent performs the necessary processing of events
func (s *Sink) ProcessEvent(event transports.Event, observer Observer) {
	switch msg := event.(type) {
	case *transports.StatusEvent:
		s.processStatusChange(msg, observer)
	case *transports.AckEvent:
		s.processAck(msg, observer)
	case *transports.PongEvent:
		// TODO: Is there a better way to encapsulate this?
		event.Observer().(*Endpoint).processPong(observer)
	default:
		observer.OnFail(event.Observer().(*Endpoint))
	}
}

// processStatusChange handles status change events
func (s *Sink) processStatusChange(status *transports.StatusEvent, observer Observer) {
	endpoint := status.Observer().(*Endpoint)

	switch status.StatusChange() {
	case transports.Ready:
		// Ignore late messages from a closing endpoint
		if endpoint.IsClosing() {
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
		if endpoint.IsClosing() || endpoint.IsFailed() {
			break
		}

		log.Info("[%s] Marking endpoint as failed", endpoint.Server())
		s.moveFailed(endpoint)
		observer.OnFail(endpoint)
	case transports.Recovered:
		// Only mark recovered and signal ready if we were originally failed
		// This simplifies logic in transports - they can simply say recovered
		// upon initialisation instead of havin to check if it was recovering
		if !endpoint.IsFailed() {
			break
		}

		// Mark as ready
		log.Info("[%s] Endpoint recovered", endpoint.Server())
		s.recoverFailed(endpoint)
		observer.OnReady(endpoint)

		// If the endpoint is still ready, nothing was sent, add to ready list
		if endpoint.status == endpointStatusReady {
			s.moveReady(endpoint)
		}
	case transports.Finished:
		server := endpoint.Server()
		log.Debug("[%s] Endpoint has finished", server)
		s.removeEndpoint(server)

		// If finish hook returns true, allow the endpoint to be recreated
		// When the caller is shutting down it can return false to finish it
		if observer.OnFinish(endpoint) {
			// Recreate the endpoint if it's still in the config
			for _, item := range s.config.Servers {
				if item == server {
					s.addEndpoint(server, addresspool.NewPool(server))
					break
				}
			}
		}
	default:
		observer.OnFail(endpoint)
	}
}

func (s *Sink) processAck(ack *transports.AckEvent, observer Observer) {
	endpoint := ack.Observer().(*Endpoint)
	complete := endpoint.processAck(ack, observer)

	if complete && endpoint.IsFull() && endpoint.NumPending() < int(s.config.MaxPendingPayloads) {
		s.markReady(endpoint)
		observer.OnReady(endpoint)

		// If the endpoint is still ready, nothing was sent, add to ready list
		if endpoint.status == endpointStatusReady {
			s.moveReady(endpoint)
		}
	}
}
