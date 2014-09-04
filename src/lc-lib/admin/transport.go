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

package admin

import (
	"net"
	"time"
)

type NetListener interface {
	Accept() (net.Conn, error)
	Close() error
	Addr() net.Addr
	SetDeadline(time.Time) error
}

type connectorFunc func(string, string) (net.Conn, error)
type listenerFunc func(string, string) (NetListener, error)

var (
	registeredConnectors map[string]connectorFunc = make(map[string]connectorFunc)
	registeredListeners map[string]listenerFunc = make(map[string]listenerFunc)
)

func registerTransport(name string, connector connectorFunc, listener listenerFunc) {
	registeredConnectors[name] = connector
	registeredListeners[name] = listener
}
