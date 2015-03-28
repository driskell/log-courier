/*
 * Copyright 2014 Jason Woods.
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
  "github.com/driskell/log-courier/src/lc-lib/addresspool"
)

// Remote is a structure used by transports to communicate back with
// the Publisher. It also contains the associated address information.
type Remote struct {
  sink     *Sink
  endpoint *Endpoint

  // The associated address pool
  AddressPool *addresspool.Pool
}

// Ready is called by a transport to signal it is ready for events.
// This should be triggered once connection is successful and the transport is
// ready to send data. It should NOT be called again until the transport
// receives data, otherwise the call may block.
func (e *Remote) Ready() {
	e.sink.readyChan <- e.endpoint
}

// ResponseChan returns the channel that responses should be sent on
func (e *Remote) ResponseChan() chan<- *Response {
	return e.sink.responseChan
}

// NewResponse creates a response wrapper linked to the endpoint that can be
// sent to the Publisher over the response channel
func (e *Remote) NewResponse(response interface{}) *Response {
  return &Response{e.endpoint, response}
}

// Fail is called by a transport to signal an error has occurred, and that all
// pending payloads should be returned to the publisher for retransmission
// elsewhere.
func (e *Remote) Fail(err error) {
	e.sink.failChan <- &Failure{e.endpoint, err}
}
