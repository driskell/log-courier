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

import (
	"github.com/driskell/log-courier/src/lc-lib/core"
)

// EndpointFailure structure contains the reason for failure
type EndpointFailure struct {
	Endpoint *Endpoint
	Error    error
}

// EndpointSink structure contains the control channels that each endpoint
// will utilise. The newEndpoint method attaches new endpoints to this
type EndpointSink struct {
	endpoints    map[string]*Endpoint
	config       *core.NetworkConfig

	ReadyChan    chan *Endpoint
	ResponseChan chan *EndpointResponse
	FailChan     chan *EndpointFailure
}

// NewEndpointSink initialises a new message sink for endpoints
func NewEndpointSink(config *core.NetworkConfig) *EndpointSink {
	// TODO: Make channel sizes configurable?
	return &EndpointSink{
		endpoints:    make(map[string]*Endpoint),
		config:       config,

		ReadyChan:    make(chan *Endpoint, 10),
		ResponseChan: make(chan *EndpointResponse, 10),
		FailChan:     make(chan *EndpointFailure, 10),
	}
}

// AddEndpoint initialises a new endpoint for a given server entry
func (f *EndpointSink) AddEndpoint(server string, addressPool *AddressPool) *Endpoint {
	endpoint := &Endpoint{
		addressPool: addressPool,
	}

	remote := &EndpointRemote{
		sink:        f,
		endpoint:    endpoint,
		AddressPool: addressPool,
	}

	endpoint.Remote = remote
	endpoint.transport = f.config.TransportFactory.NewTransport(remote)
	endpoint.resetPayloads()

	f.endpoints[server] = endpoint

	return endpoint
}

// Shutdown signals all associated endpoints to begin shutting down
func (f *EndpointSink) Shutdown() {
	for _, endpoint := range f.endpoints {
		endpoint.shutdown()
	}
}

// Wait for associated endpoints to complete shutting down
func (f *EndpointSink) Wait() {
	for _, endpoint := range f.endpoints {
		endpoint.wait()
	}
}
