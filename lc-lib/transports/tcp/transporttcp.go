/*
 * Copyright 2012-2020 Jason Woods and contributors
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
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/payload"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// transportTCP implements a transport that sends over TCP
// It also can optionally introduce a TLS layer for security
type transportTCP struct {
	// Constructor
	ctx            context.Context
	config         *TransportTCPFactory
	netConfig      *transports.Config
	finishOnFail   bool
	pool           *addresspool.Pool
	eventChan      chan<- transports.Event
	controllerChan chan error
	connectionChan chan *socketMessage
	backoff        *core.ExpBackoff

	// Internal
	connMutex    sync.RWMutex
	conn         *connection
	supportsEVNT bool
	shutdown     bool
}

// ReloadConfig returns true if the transport needs to be restarted in order
// for the new configuration to apply
func (t *transportTCP) ReloadConfig(netConfig *transports.Config, finishOnFail bool) bool {
	newConfig := netConfig.Factory.(*TransportTCPFactory)

	// Check if automatic reconnection should be enabled or not
	if t.finishOnFail != finishOnFail {
		return true
	}

	// TODO: Check timestamps of underlying certificate files to detect changes
	if newConfig.SSLCertificate != t.config.SSLCertificate || newConfig.SSLKey != t.config.SSLKey || newConfig.SSLCA != t.config.SSLCA {
		return true
	}

	// Check if timeout or max pending payloads changed
	if netConfig.Timeout != t.netConfig.Timeout || netConfig.MaxPendingPayloads != t.netConfig.MaxPendingPayloads {
		return true
	}

	return false
}

// startController starts the controller
func (t *transportTCP) startController() {
	go t.controllerRoutine()
}

// controllerRoutine is the master routine which handles connection and reconnection
// When reconnecting, the socket and all routines are torn down and restarted
func (t *transportTCP) controllerRoutine() {
	defer func() {
		t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Finished)
	}()

	// Main connect loop
	for {
		conn, err := t.connect()
		if err == nil {
			err = conn.Run(func() {
				// Send a started signal to say we're ready
				t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Started)

				t.connMutex.Lock()
				t.conn = conn
				t.supportsEVNT = t.conn.SupportsEVNT()
				shutdown := t.shutdown
				t.connMutex.Unlock()

				// Request immediate shutdown if we just noticed it - shutdown
				// will not have registered this if t.conn was not previously set as needed
				if shutdown {
					t.conn.SendMessage(nil)
				}
			})
		}

		t.connMutex.Lock()
		shutdown := t.shutdown
		t.conn = nil
		t.connMutex.Unlock()

		if err != nil {
			if err == errHardCloseRequested {
				log.Info("[%s] Transport shutdown requested", t.pool.Server())
			} else {
				log.Errorf("[%s] Transport error, disconnected: %s", t.pool.Server(), err)
			}
			if t.finishOnFail || shutdown {
				return
			}
		} else {
			log.Info("[%s] Transport disconnected gracefully", t.pool.Server())
			return
		}

		select {
		case <-t.controllerChan:
		case t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Failed):
		}

		if t.reconnectWait() {
			break
		}
	}
}

// reconnectWait waits the backoff timeout before attempting to reconnect
// It also monitors for shutdown whilst waiting.
func (t *transportTCP) reconnectWait() bool {
	now := time.Now()
	reconnectDue := now.Add(t.backoff.Trigger())

	select {
	case <-t.controllerChan:
		// Shutdown request
		return true
	case <-time.After(reconnectDue.Sub(now)):
	}

	return false
}

// getTLSConfig returns the TLS configuration for the connection
func (t *transportTCP) getTLSConfig() (tlsConfig *tls.Config) {
	tlsConfig = new(tls.Config)

	tlsConfig.MinVersion = t.config.minTLSVersion
	tlsConfig.MaxVersion = t.config.maxTLSVersion

	// Set the certificate if we set one
	if t.config.certificate != nil {
		tlsConfig.Certificates = []tls.Certificate{*t.config.certificate}
	}

	// Set CA for server verification
	tlsConfig.RootCAs = x509.NewCertPool()
	for _, cert := range t.config.caList {
		tlsConfig.RootCAs.AddCert(cert)
	}

	// Set the tlsConfig server name for server validation (required since Go 1.3)
	tlsConfig.ServerName = t.pool.Host()

	return
}

// connect selects the next address to use and triggers the connect
func (t *transportTCP) connect() (*connection, error) {
	t.checkClientCertificates()

	addr, err := t.pool.Next()
	if err != nil {
		return nil, fmt.Errorf("Failed to select next address: %s", err)
	}

	desc := t.pool.Desc()

	log.Info("[%s] Attempting to connect to %s", t.pool.Server(), desc)

	socket, err := net.DialTimeout("tcp", addr.String(), t.netConfig.Timeout)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to %s: %s", desc, err)
	}

	// Maxmium number of payloads we will queue - must be correct as otherwise
	// the endpoint implementation will block unexpectedly and be unable to
	// distribute events across multiple transports
	// Pings don't count towards limit, we only ping if this queue is empty
	sendChan := make(chan protocolMessage, t.netConfig.MaxPendingPayloads)

	// Now wrap in TLS if this is the TLS transport
	var connectionSocket connectionSocket
	if t.config.transport == TransportTCPTLS {
		connectionSocket = newConnectionSocketTLS(socket.(*net.TCPConn), t.getTLSConfig(), false, t.pool.Server())
	} else {
		connectionSocket = newConnectionSocketTCP(socket.(*net.TCPConn))
	}

	connContext := context.WithValue(t.ctx, contextIsClient, true)

	conn := newConnection(connContext, connectionSocket, t.pool.Server(), t.eventChan, sendChan)

	log.Notice("[%s] Connected to %s", t.pool.Server(), desc)
	return conn, nil
}

// Write a message to the transport - only valid after Started transport event received
func (t *transportTCP) Write(payload *payload.Payload) error {
	t.connMutex.Lock()
	defer t.connMutex.Unlock()
	if t.conn == nil {
		return errors.New("Invalid connection state")
	}
	var msg protocolMessage
	if t.supportsEVNT {
		msg = &protocolEVNT{nonce: payload.Nonce, events: payload.Events()}
	} else {
		msg = &protocolJDAT{nonce: payload.Nonce, events: payload.Events()}
	}
	return t.conn.SendMessage(msg)
}

// Ping the remote server - only valid after Started transport event received
func (t *transportTCP) Ping() error {
	t.connMutex.Lock()
	defer t.connMutex.Unlock()
	if t.conn == nil {
		return errors.New("Invalid connection state")
	}
	return t.conn.SendMessage(&protocolPING{})
}

// Fail the transport / Shutdown hard
func (t *transportTCP) Fail() {
	t.connMutex.Lock()
	defer t.connMutex.Unlock()
	if !t.shutdown {
		if t.conn != nil {
			t.conn.Teardown()
		}
		t.shutdown = true
		close(t.controllerChan)
	}
}

// Shutdown the transport gracefully - only valid after Started transport event received
func (t *transportTCP) Shutdown() {
	// Sending nil triggers graceful shutdown
	t.connMutex.Lock()
	defer t.connMutex.Unlock()
	if !t.shutdown {
		if t.conn != nil {
			t.conn.SendMessage(nil)
		}
		t.shutdown = true
		close(t.controllerChan)
	}
}

// checkClientCertificates logs a warning if it finds any certificates that are
// not currently valid
func (t *transportTCP) checkClientCertificates() {
	if t.config.certificateList == nil {
		// No certificates were specified, don't do anything
		return
	}

	now := time.Now()
	certIssues := false

	for _, cert := range t.config.certificateList {
		if cert.NotBefore.After(now) {
			log.Warning("The client certificate with common name '%s' is not valid until %s.", cert.Subject.CommonName, cert.NotBefore.Format("Jan 02 2006"))
			certIssues = true
		}

		if cert.NotAfter.Before(now) {
			log.Warning("The client certificate with common name '%s' expired on %s.", cert.Subject.CommonName, cert.NotAfter.Format("Jan 02 2006"))
			certIssues = true
		}
	}

	if certIssues {
		log.Warning("Client certificate issues may prevent successful connection.")
	}
}
