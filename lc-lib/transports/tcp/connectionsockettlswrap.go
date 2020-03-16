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
	"net"
	"time"
)

// connectionSocketTLSWrap wraps a TCP socket with timeout functionality for use in TLS
// If tls.Conn.Write ever times out it will permanently break, so we cannot use
// SetWriteDeadline with it directly. So we wrap the given tcpsocket and handle
// the SetWriteDeadline there and check shutdown signal and loop. Inside
// tls.Conn the Write blocks until it finishes and everyone is happy
type connectionSocketTLSWrap struct {
	shutdownChan <-chan error
	tcpSocket    net.Conn

	net.Conn
}

func (w *connectionSocketTLSWrap) Read(b []byte) (int, error) {
	return w.tcpSocket.Read(b)
}

func (w *connectionSocketTLSWrap) Write(b []byte) (n int, err error) {
	length := 0

RetrySend:
	for {
		// Timeout after socket_interval_seconds, check for shutdown, and try again
		w.tcpSocket.SetWriteDeadline(time.Now().Add(socketIntervalSeconds * time.Second))

		n, err = w.tcpSocket.Write(b[length:])
		length += n

		if length >= len(b) {
			return length, err
		}

		if err == nil {
			// Keep trying
			continue
		}

		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// Check for shutdown, then try again
			select {
			case <-w.shutdownChan:
				// Shutdown
				return length, errHardCloseRequested
			default:
				goto RetrySend
			}
		}

		return length, err
	} /* loop forever */
}

func (w *connectionSocketTLSWrap) Close() error {
	return w.tcpSocket.Close()
}

func (w *connectionSocketTLSWrap) LocalAddr() net.Addr {
	return w.tcpSocket.LocalAddr()
}

func (w *connectionSocketTLSWrap) RemoteAddr() net.Addr {
	return w.tcpSocket.RemoteAddr()
}

func (w *connectionSocketTLSWrap) SetDeadline(t time.Time) error {
	return w.tcpSocket.SetDeadline(t)
}

func (w *connectionSocketTLSWrap) SetReadDeadline(t time.Time) error {
	return w.tcpSocket.SetReadDeadline(t)
}

func (w *connectionSocketTLSWrap) SetWriteDeadline(t time.Time) error {
	return w.tcpSocket.SetWriteDeadline(t)
}
