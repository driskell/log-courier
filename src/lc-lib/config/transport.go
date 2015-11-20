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

package config

// TransportRegistrarFunc is a callback that validates the configuration for
// a transport that was registered vua RegisterTransport
type TransportRegistrarFunc func(*Config, string, map[string]interface{}, string) (interface{}, error)

var registeredTransports = make(map[string]TransportRegistrarFunc)

// RegisterTransport registered a transport with the configuration module by
// providing a callback that can be used to validate the configuration
func RegisterTransport(transport string, registrarFunc TransportRegistrarFunc) {
	registeredTransports[transport] = registrarFunc
}

// AvailableTransports returns the list of registered transports available for
// use
func AvailableTransports() (ret []string) {
	ret = make([]string, 0, len(registeredTransports))
	for k := range registeredTransports {
		ret = append(ret, k)
	}
	return
}
