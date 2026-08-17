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
	"fmt"
	"io"
	"net"
	"time"

	proxyproto "github.com/pires/go-proxyproto"
)

// proxyHeaderTimeout bounds how long a connection may take to send its PROXY header,
// matching the handshake deadline connectionSocketTLS applies for TLS
const proxyHeaderTimeout = 10 * time.Second

// proxyConnCloseWriter adds CloseWrite to a proxyproto.Conn so it satisfies tcpSocket
type proxyConnCloseWriter struct {
	*proxyproto.Conn
}

// CloseWrite half-closes the send side of the underlying TCP connection
func (w *proxyConnCloseWriter) CloseWrite() error {
	if tcpConn, ok := w.TCPConn(); ok {
		return tcpConn.CloseWrite()
	}
	return w.Close()
}

// LocalAddr returns the address of this receiver rather than the PROXY header's
// destination address, which is the address the client connected to on the proxy
func (w *proxyConnCloseWriter) LocalAddr() net.Addr {
	return w.Raw().LocalAddr()
}

// connectionSocketProxy wraps a socket with PROXY protocol support, exposing the
// original client address once the header has been read
type connectionSocketProxy struct {
	connectionSocket
	proxyConn *proxyproto.Conn
	desc      string
}

// newConnectionSocketProxy returns a new PROXY protocol enabled socket wrapping innerSocket
func newConnectionSocketProxy(proxyConn *proxyproto.Conn, innerSocket connectionSocket) *connectionSocketProxy {
	return &connectionSocketProxy{connectionSocket: innerSocket, proxyConn: proxyConn}
}

// Setup reads the PROXY header before performing any inner handshake (such as TLS), as
// the header is always the first bytes on the wire ahead of any other protocol
func (p *connectionSocketProxy) Setup(ctx context.Context) error {
	// A zero-length read triggers the header parse without consuming any of the data
	// that follows it, surfacing the parse/policy error via the normal Read path
	if _, err := p.proxyConn.Read(nil); err != nil {
		if err != io.EOF {
			log.Warningf("[R %s] PROXY protocol header from %s failed: %s", p.proxyConn.Raw().LocalAddr().String(), p.proxyConn.Raw().RemoteAddr().String(), err)
		}
		return err
	}

	if header := p.proxyConn.ProxyHeader(); header != nil {
		p.desc = fmt.Sprintf("via PROXY from %s", p.proxyConn.Raw().RemoteAddr().String())
		log.Debugf("[R %s - %s] PROXY protocol header received from %s", p.proxyConn.Raw().LocalAddr().String(), p.proxyConn.RemoteAddr().String(), p.proxyConn.Raw().RemoteAddr().String())
	}

	return p.connectionSocket.Setup(ctx)
}

// Desc returns the inner description, plus the proxy that relayed the connection
func (p *connectionSocketProxy) Desc() string {
	innerDesc := p.connectionSocket.Desc()
	if p.desc == "" {
		return innerDesc
	}
	if innerDesc == "" || innerDesc == "-" {
		return p.desc
	}
	return innerDesc + " " + p.desc
}
