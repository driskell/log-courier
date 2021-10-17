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
	"crypto/tls"
	"net"
	"time"
)

// connectionSocketTLS wraps a TCP socket with TLS
type connectionSocketTLS struct {
	tcpSocket *net.TCPConn
	tlsConfig *tls.Config
	server    bool
	poolDesc  string
	subject   string
	*tls.Conn
}

// newConnectionSocketTLS return a new TLS enabled socket
func newConnectionSocketTLS(tcpSocket *net.TCPConn, tlsConfig *tls.Config, server bool, poolDesc string) *connectionSocketTLS {
	return &connectionSocketTLS{
		tcpSocket: tcpSocket,
		tlsConfig: tlsConfig,
		server:    server,
		poolDesc:  poolDesc,
	}
}

// Setup wraps the socket and resolves the handshake
func (t *connectionSocketTLS) Setup(ctx context.Context) error {
	var side string
	if t.server {
		side = "R"
		t.Conn = tls.Server(&connectionSocketTLSWrap{receiverShutdownChan: ctx.Done(), tcpSocket: t.tcpSocket}, t.tlsConfig)
	} else {
		side = "T"
		t.Conn = tls.Client(&connectionSocketTLSWrap{receiverShutdownChan: ctx.Done(), tcpSocket: t.tcpSocket}, t.tlsConfig)
	}

	// TODO: Use netConfig.Timeout
	t.SetDeadline(time.Now().Add(10 * time.Second))

	err := t.Handshake()
	if err != nil {
		return err
	}

	t.subject = ""
	if len(t.ConnectionState().VerifiedChains) > 0 {
		t.subject = t.ConnectionState().VerifiedChains[0][0].Subject.String()
	} else {
		t.subject = "No client certificate"
	}

	log.Notice("[%s %s - %s] TLS handshake completed [%s]", side, t.LocalAddr().String(), t.RemoteAddr().String(), t.subject)
	return nil
}

// CloseWrite triggers EOF on remote side read by shutting down send side of TCP connection
func (t *connectionSocketTLS) CloseWrite() error {
	if err := t.Conn.CloseWrite(); err != nil {
		return err
	}
	return t.tcpSocket.CloseWrite()
}

// Desc returns the client certificate Subject
func (t *connectionSocketTLS) Desc() string {
	return t.subject
}
