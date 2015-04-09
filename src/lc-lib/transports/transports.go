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

package transports

import (
	"github.com/driskell/log-courier/src/lc-lib/addresspool"
	"github.com/driskell/log-courier/src/lc-lib/core"
)

// Flags to state the action required to fulfil configuration reload
const (
	reloadNone = iota
	reloadServers
	reloadTransport
)

// Transport is the generic interface that all transports implement
type Transport interface {
	// TODO: Implement this in new transport systems
	//ReloadConfig(*NetworkConfig) int
	Write(string, []*core.EventDescriptor) error
	Ping() error
	Fail()
	Shutdown()
}

// EndpointCallback is the interface implemented by the consumer of a transport,
// to allow the transport to communicate back
type EndpointCallback interface {
	Pool() *addresspool.Pool
	Ready()
	ResponseChan() chan<- Response
	Fail()
	Recover()
	Finished()
}

// transportFactory is the interface that all transport factories implement. The
// transport factory should store the transport's configuration and, when
// NewTransport is called, return an instance of the transport that obeys that
// configuration
type transportFactory interface {
	NewTransport(EndpointCallback) Transport
}

// NewTransport returns a Transport interface initialised from the given Factory
func NewTransport(factory interface{}, endpoint EndpointCallback) Transport {
	return factory.(transportFactory).NewTransport(endpoint)
}
