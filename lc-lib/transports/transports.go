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

import (
	"errors"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/core"
)

// ErrForcedFailure is an error a Transport can use to represent a forced
// failure by the publisher
var ErrForcedFailure = errors.New("Failed by endpoint manager")

// Observer is the interface implemented by the consumer of a transport, to
// allow the transport to communicate back
// To all intents and purposes this is the Endpoint
type Observer interface {
	Pool() *addresspool.Pool
	EventChan() chan<- Event
}

// Transport is the generic interface that all transports implement
type Transport interface {
	Fail()
	Ping() error
	ReloadConfig(interface{}, bool) bool
	Shutdown()
	Write(string, []*core.EventDescriptor) error
}

// transportFactory is the interface that all transport factories implement. The
// transport factory should store the transport's configuration and, when
// NewTransport is called, return an instance of the transport that obeys that
// configuration
type transportFactory interface {
	NewTransport(Observer, bool) Transport
}

// NewTransport returns a Transport interface initialised from the given Factory
func NewTransport(factory interface{}, observer Observer, finishOnFail bool) Transport {
	return factory.(transportFactory).NewTransport(observer, finishOnFail)
}
