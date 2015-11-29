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

package transports

import "encoding/binary"

// AckResponse contains information on which events have been acknowledged and
// implements the Response interface
type AckResponse struct {
	endpoint Endpoint
	nonce    string
	sequence uint32
}

// Endpoint returns the associated endpoint
func (r *AckResponse) Endpoint() Endpoint {
	return r.endpoint
}

// Nonce returns the identifier of the payload this Ack is acknowledging
func (r *AckResponse) Nonce() string {
	return r.nonce
}

// Sequence returns the payload position this Ack is acknowledging
func (r *AckResponse) Sequence() uint32 {
	return r.sequence
}

// NewAckResponse generates a new AckResponse for the given Endpoint
func NewAckResponse(endpoint Endpoint, nonce string, sequence uint32) Response {
	return &AckResponse{
		endpoint: endpoint,
		nonce:    nonce,
		sequence: sequence,
	}
}

// NewAckResponseFromBytes convers the given bytes into a nonce and integer
// sequence and passes them to NewAckResponse
// This is a convenience function
func NewAckResponseFromBytes(endpoint Endpoint, nonce []byte, sequence []byte) Response {
	stringNonce := string(nonce)
	integerSequence := binary.BigEndian.Uint32(sequence)
	return NewAckResponse(endpoint, stringNonce, integerSequence)
}

// PongResponse is received when a transport has responded to a Ping() request
// and implements the Response interface
type PongResponse struct {
	endpoint Endpoint
}

// Endpoint returns the associated endpoint
func (r *PongResponse) Endpoint() Endpoint {
	return r.endpoint
}

// NewPongResponse generates a new PongResponse for the given Endpoint
func NewPongResponse(endpoint Endpoint) Response {
	return &PongResponse{
		endpoint: endpoint,
	}
}
