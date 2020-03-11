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

// markActive marks an idle endpoint as active and puts it on the ready list
func (s *Sink) markActive(endpoint *Endpoint, observer Observer) {
	// Ignore if not idle
	if !endpoint.IsIdle() {
		return
	}

	log.Debug("[%s] Endpoint is ready", endpoint.Server())

	endpoint.mutex.Lock()
	endpoint.status = endpointStatusActive
	endpoint.mutex.Unlock()

	s.readyList.PushBack(&endpoint.readyElement)

	observer.OnStarted(endpoint)
}

// moveFailed stores the endpoint on the failed list, removing it from the
// ready list so no more events are sent to it
func (s *Sink) moveFailed(endpoint *Endpoint, observer Observer) {
	// Should never get here if we're closed, caller should check
	if !endpoint.IsAlive() && !endpoint.IsClosing() {
		return
	}

	log.Info("[%s] Marking endpoint as failed", endpoint.Server())

	s.ClearTimeout(&endpoint.Timeout)

	if endpoint.IsActive() {
		s.readyList.Remove(&endpoint.readyElement)
	}

	shutdown := endpoint.IsClosing()

	endpoint.mutex.Lock()
	endpoint.status = endpointStatusFailed
	endpoint.averageLatency = 0
	endpoint.mutex.Unlock()

	s.failedList.PushFront(&endpoint.failedElement)

	// endpoint.ForceFailure has no observer and calls with nil
	if observer != nil {
		observer.OnFail(endpoint)
	}

	// If we're shutting down, give up and complete transport shutdown
	if shutdown {
		endpoint.shutdownTransport()
	}
}

// recoverFailed removes an endpoint from the failed list and marks it active
func (s *Sink) recoverFailed(endpoint *Endpoint, observer Observer) {
	// Ignore if we haven't failed
	if !endpoint.IsFailed() {
		return
	}

	endpoint.mutex.Lock()
	endpoint.status = endpointStatusIdle
	endpoint.mutex.Unlock()

	s.failedList.Remove(&endpoint.failedElement)

	backoff := endpoint.backoff.Trigger()
	log.Info("[%s] Endpoint has recovered - will resume in %v", endpoint.Server(), backoff)

	// Backoff before allowing recovery
	s.RegisterTimeout(
		&endpoint.Timeout,
		backoff,
		func() {
			s.markActive(endpoint, observer)
		},
	)
}
