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

package tcp

import (
	"context"
	"net"
)

// connectionSocketTCP holds a TCP socket
type connectionSocketTCP struct {
	*net.TCPConn
}

// newConnectionSocketTLS return a TCP socket supporting our connection interface
func newConnectionSocketTCP(tcpSocket *net.TCPConn) *connectionSocketTCP {
	return &connectionSocketTCP{TCPConn: tcpSocket}
}

// Setup is not required for a TCP connection so is nil effect
func (w *connectionSocketTCP) Setup(ctx context.Context) error {
	return nil
}

// Desc returns dash - TCP connections have no description
func (w *connectionSocketTCP) Desc() string {
	return "-"
}
