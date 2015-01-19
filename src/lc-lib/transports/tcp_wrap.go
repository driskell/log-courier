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
	"net"
	"time"
)

// If tls.Conn.Write ever times out it will permanently break, so we cannot use SetWriteDeadline with it directly
// So we wrap the given tcpsocket and handle the SetWriteDeadline there and check shutdown signal and loop
// Inside tls.Conn the Write blocks until it finishes and everyone is happy
type transportTcpWrap struct {
	transport *TransportTcp
	tcpsocket net.Conn

	net.Conn
}

func (w *transportTcpWrap) Read(b []byte) (int, error) {
	return w.tcpsocket.Read(b)
}

func (w *transportTcpWrap) Write(b []byte) (n int, err error) {
	length := 0

RetrySend:
	for {
		// Timeout after socket_interval_seconds, check for shutdown, and try again
		w.tcpsocket.SetWriteDeadline(time.Now().Add(socket_interval_seconds * time.Second))

		n, err = w.tcpsocket.Write(b[length:])
		length += n
		if err == nil {
			return length, err
		} else if net_err, ok := err.(net.Error); ok && net_err.Timeout() {
			// Check for shutdown, then try again
			select {
			case <-w.transport.shutdown:
				// Shutdown
				return length, err
			default:
				goto RetrySend
			}
		} else {
			return length, err
		}
	} /* loop forever */
}

func (w *transportTcpWrap) Close() error {
	return w.tcpsocket.Close()
}

func (w *transportTcpWrap) LocalAddr() net.Addr {
	return w.tcpsocket.LocalAddr()
}

func (w *transportTcpWrap) RemoteAddr() net.Addr {
	return w.tcpsocket.RemoteAddr()
}

func (w *transportTcpWrap) SetDeadline(t time.Time) error {
	return w.tcpsocket.SetDeadline(t)
}

func (w *transportTcpWrap) SetReadDeadline(t time.Time) error {
	return w.tcpsocket.SetReadDeadline(t)
}

func (w *transportTcpWrap) SetWriteDeadline(t time.Time) error {
	return w.tcpsocket.SetWriteDeadline(t)
}
