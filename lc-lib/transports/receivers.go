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

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
)

// Receiver is the generic interface that all receivers implement
type Receiver interface {
	ReloadConfig(*config.Config, ReceiverFactory) bool
	Acknowledge(context.Context, *string, uint32) error
	Pong(context.Context) error
	FailConnection(context.Context, error)
	Shutdown()
}

// ReceiverFactory is the interface that all receiver factories implement. The
// receiver factory should store the receiver's configuration and, when
// NewReceiver is called, return an instance of the receiver that obeys that
// configuration
type ReceiverFactory interface {
	NewReceiver(context.Context, *addresspool.Pool, chan<- Event) Receiver
}

// ReceiverRegistrarFunc is a callback that validates the configuration for
// a transport that was registered via Register
type ReceiverRegistrarFunc func(*config.Parser, string, map[string]interface{}, string) (ReceiverFactory, error)

var registeredReceivers = make(map[string]ReceiverRegistrarFunc)

// RegisterReceiver registered a transport with the configuration module by providing a
// callback that can be used to validate the configuration
func RegisterReceiver(transport string, registrarFunc ReceiverRegistrarFunc) {
	registeredReceivers[transport] = registrarFunc
}

// AvailableReceivers returns the list of registered transports available for use
func AvailableReceivers() (ret []string) {
	ret = make([]string, 0, len(registeredReceivers))
	for k := range registeredReceivers {
		ret = append(ret, k)
	}
	return
}
