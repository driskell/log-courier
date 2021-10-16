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

package transports

import (
	"context"
	"errors"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
)

// ErrForcedFailure is an error a Transport can use to represent a forced
// failure by the publisher
var ErrForcedFailure = errors.New("failed by endpoint manager")

// Transport is the generic interface that all transports implement
type Transport interface {
	Fail()
	Ping() error
	ReloadConfig(*Config) bool
	Shutdown()
	SendEvents(string, []*event.Event) error
}

// TransportFactory is the interface that all transport factories implement. The
// transport factory should store the transport's configuration and, when
// NewTransport is called, return an instance of the transport that obeys that
// configuration
type TransportFactory interface {
	NewTransport(context.Context, *addresspool.Pool, chan<- Event) Transport
}

// TransportRegistrarFunc is a callback that validates the configuration for
// a transport that was registered via RegisterTransport
type TransportRegistrarFunc func(*config.Parser, string, map[string]interface{}, string) (TransportFactory, error)

var registeredTransports = make(map[string]TransportRegistrarFunc)

// RegisterTransport registers a transport with the configuration module by providing a
// callback that can be used to validate the configuration
func RegisterTransport(transport string, registrarFunc TransportRegistrarFunc) {
	registeredTransports[transport] = registrarFunc
}

// AvailableTransports returns the list of registered transports available for use
func AvailableTransports() (ret []string) {
	ret = make([]string, 0, len(registeredTransports))
	for k := range registeredTransports {
		ret = append(ret, k)
	}
	return
}
