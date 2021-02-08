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

package admin

import (
	"net"
	"strings"
	"time"
)

type netDialer interface {
	Dial(string, string) (net.Conn, error)
	Host() string
}

type netListener interface {
	net.Listener
	SetDeadline(time.Time) error
}

type dialerFunc func(string, string) (netDialer, error)
type listenerFunc func(string, string) (netListener, error)

var (
	registeredDialers   = make(map[string]dialerFunc)
	registeredListeners = make(map[string]listenerFunc)
)

func registerTransport(name string, dialer dialerFunc, listener listenerFunc) {
	registeredDialers[name] = dialer
	registeredListeners[name] = listener
}

func splitAdminConnectString(adminConnect string) []string {
	connect := strings.SplitN(adminConnect, ":", 2)
	if len(connect) == 1 {
		connect = append(connect, connect[0])
		connect[0] = "tcp"
	}

	return connect
}
