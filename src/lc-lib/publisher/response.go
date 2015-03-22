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

package publisher

// EndpointResponse is a wrapper that joins a response with an endpoint
// identifier which can then be sent to the Publisher
type EndpointResponse struct {
  endpoint *Endpoint
  Response interface{}
}

// Endpoint returns the associated endpoint
func (r *EndpointResponse) Endpoint() *Endpoint {
	return r.endpoint
}

// AckResponse contains information on which events have been acknowledged and
// implements the Response interface
type AckResponse struct {
	Nonce    string
	Sequence uint32
}

// PongResponse is received when a transport has responded to a Ping() request
// and implements the Response interface
type PongResponse struct {
}
