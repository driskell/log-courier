/*
 * Copyright 2014-2015 Jason Woods.
 *
 * This file is a modification of code from Logstash Forwarder.
 * Copyright 2012-2013 Jordan Sissel and contributors.
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
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

type connectionTLS struct {
	connectionTCP

	server    bool
	tlsConfig *tls.Config
	tlsSocket *tls.Conn
	tcpSocket net.Conn
	subject   string
}

// setup wraps the socket and begins the handshake
func (t *connectionTLS) Setup() {
	t.wait.Add(1)

	if t.server {
		t.tlsSocket = tls.Server(&transportTCPWrap{controllerChan: t.controllerChan, tcpSocket: t.socket}, t.tlsConfig)
	} else {
		t.tlsSocket = tls.Client(&transportTCPWrap{controllerChan: t.controllerChan, tcpSocket: t.socket}, t.tlsConfig)
	}

	t.tcpSocket = t.socket
	t.socket = t.tlsSocket
	// TODO: Use netConfig.Timeout
	t.socket.SetDeadline(time.Now().Add(10 * time.Second))

	go t.handshake()
}

// handshake processes the initial handshake before starting the sender and receiver routines
func (t *connectionTLS) handshake() {
	defer func() {
		t.wait.Done()
	}()
	err := t.tlsSocket.Handshake()
	if err != nil {
		t.connectionChan <- &socketMessage{conn: t, err: fmt.Errorf("TLS Handshake failure with %s: %s", t.socket.RemoteAddr().String(), err)}
		return
	}

	if len(t.tlsSocket.ConnectionState().VerifiedChains) > 0 {
		t.subject = fmt.Sprintf(" [%s]", t.tlsSocket.ConnectionState().VerifiedChains[0][0].Subject)
	}

	log.Notice("[%s] Handshake completed with %s%s", t.poolServer, t.socket.RemoteAddr().String(), t.subject)

	t.connectionTCP.Setup()
}

// teardown ends the connection
func (t *connectionTLS) Teardown() {
	close(t.controllerChan)
	t.wait.Wait()
	t.socket.Close()
	t.tcpSocket.Close()
	log.Notice("[%s] Disconnected from %s%s", t.poolServer, t.socket.RemoteAddr().String(), t.subject)
}
