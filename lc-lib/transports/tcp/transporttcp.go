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
	"fmt"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// transportTCP implements a transport that sends over TCP
// It also can optionally introduce a TLS layer for security
type transportTCP struct {
	// Constructor
	ctx          context.Context
	shutdownFunc context.CancelFunc
	config       *TransportTCPFactory
	netConfig    *transports.Config
	pool         *addresspool.Pool
	eventChan    chan<- transports.Event
	backoff      *core.ExpBackoff

	// Internal
	connMutex    sync.RWMutex
	conn         *connection
	supportsEVNT bool
	shutdown     bool
}

// ReloadConfig returns true if the transport needs to be restarted in order
// for the new configuration to apply
func (t *transportTCP) ReloadConfig(netConfig *transports.Config) bool {
	newConfig := netConfig.Factory.(*TransportTCPFactory)
	if newConfig.MinTLSVersion != t.config.MinTLSVersion {
		return true
	}
	if newConfig.MaxTLSVersion != t.config.MaxTLSVersion {
		return true
	}
	if newConfig.Reconnect != t.config.Reconnect {
		return true
	}
	if newConfig.ReconnectMax != t.config.ReconnectMax {
		return true
	}
	if newConfig.SSLCertificate != t.config.SSLCertificate {
		return true
	}
	if newConfig.SSLKey != t.config.SSLKey {
		return true
	}
	if newConfig.SSLCA != t.config.SSLCA {
		return true
	}
	if netConfig.MaxPendingPayloads != t.netConfig.MaxPendingPayloads {
		return true
	}
	if netConfig.Timeout != t.netConfig.Timeout {
		return true
	}

	if !reflect.DeepEqual(newConfig.certificate.Certificate, t.config.certificate.Certificate) {
		return true
	}
	if len(newConfig.caList) != len(t.config.caList) {
		return true
	}
	for index := range newConfig.caList {
		if !reflect.DeepEqual(newConfig.caList[index].Raw, t.config.caList[index].Raw) {
			return true
		}
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
		t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Finished, nil)
	}()

	// Main connect loop
MainLoop:
	for {
		conn, err := t.connect()
		if err == nil {
			err = conn.Run(func() {
				// Send a started signal to say we're ready
				t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Started, nil)

				t.connMutex.Lock()
				t.conn = conn
				t.supportsEVNT = t.conn.SupportsEVNT()
				t.connMutex.Unlock()
			})
		}

		t.connMutex.Lock()
		t.conn = nil
		t.connMutex.Unlock()

		if err != nil {
			if err == errHardCloseRequested {
				log.Noticef("[T %s] Transport forcefully disconnected", t.pool.Server())
			} else {
				log.Errorf("[T %s] Transport error, disconnected: %s", t.pool.Server(), err)
			}
		} else {
			log.Noticef("[T %s] Transport disconnected gracefully", t.pool.Server())
			break MainLoop
		}

		select {
		case <-t.ctx.Done():
			break MainLoop
		case t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Failed, err):
		}

		if t.reconnectWait() {
			break
		}
	}

	// Ensure all resources cleared up for the context
	t.shutdownFunc()
}

// reconnectWait waits the backoff timeout before attempting to reconnect
// It also monitors for shutdown whilst waiting.
func (t *transportTCP) reconnectWait() bool {
	now := time.Now()
	reconnectDue := now.Add(t.backoff.Trigger())

	select {
	case <-t.ctx.Done():
		// Failed transport
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
		return nil, fmt.Errorf("failed to select next address: %s", err)
	}

	desc := t.pool.Desc()

	log.Infof("[T %s] Attempting to connect to %s", t.pool.Server(), desc)

	socket, err := net.DialTimeout("tcp", addr.String(), t.netConfig.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %s", desc, err)
	}

	// Now wrap in TLS if this is the TLS transport
	var connectionSocket connectionSocket
	if t.config.transport == TransportTCPTLS {
		connectionSocket = newConnectionSocketTLS(socket.(*net.TCPConn), t.getTLSConfig(), false, t.pool.Server())
	} else {
		connectionSocket = newConnectionSocketTCP(socket.(*net.TCPConn))
	}

	conn := newConnection(t.ctx, connectionSocket, t.pool.Server(), true, t.eventChan)

	log.Noticef("[T %s - %s] Connected to %s", socket.LocalAddr().String(), socket.RemoteAddr().String(), desc)
	return conn, nil
}

// SendEvents sends an event message with given nonce to the transport - only valid after Started transport event received
func (t *transportTCP) SendEvents(nonce string, events []*event.Event) error {
	t.connMutex.Lock()
	defer t.connMutex.Unlock()
	if t.conn == nil {
		return transports.ErrInvalidState
	}
	var eventsAsBytes = make([][]byte, len(events))
	for idx, item := range events {
		eventsAsBytes[idx] = item.Bytes()
	}
	var msg eventsMessage
	if t.supportsEVNT {
		msg = &protocolEVNT{nonce: &nonce, events: eventsAsBytes}
	} else {
		msg = &protocolJDAT{nonce: &nonce, events: eventsAsBytes}
	}
	log.Debugf("[T %s > %s] Sending %s payload with nonce %x and %d events", t.conn.socket.LocalAddr().String(), t.conn.socket.RemoteAddr().String(), msg.Type(), *msg.Nonce(), len(msg.Events()))
	return t.conn.SendMessage(msg)
}

// Ping the remote server - only valid after Started transport event received
func (t *transportTCP) Ping() error {
	t.connMutex.Lock()
	defer t.connMutex.Unlock()
	if t.conn == nil {
		return transports.ErrInvalidState
	}
	log.Debugf("[T %s > %s] Sending ping", t.conn.socket.LocalAddr().String(), t.conn.socket.RemoteAddr().String())
	return t.conn.SendMessage(&protocolPING{})
}

// Fail the transport / Shutdown hard
func (t *transportTCP) Fail() {
	t.shutdownFunc()
}

// Shutdown the transport gracefully
func (t *transportTCP) Shutdown() {
	t.connMutex.Lock()
	defer t.connMutex.Unlock()
	if !t.shutdown {
		if t.conn == nil {
			// No connection to gracefully shutdown, use context to fail and cancel any retries
			t.shutdownFunc()
		} else {
			// Sending nil triggers graceful shutdown
			if err := t.conn.SendMessage(nil); err != nil {
				t.shutdownFunc()
			}
		}
		t.shutdown = true
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
			log.Warningf("The client certificate with common name '%s' is not valid until %s.", cert.Subject.CommonName, cert.NotBefore.Format("Jan 02 2006"))
			certIssues = true
		}

		if cert.NotAfter.Before(now) {
			log.Warningf("The client certificate with common name '%s' expired on %s.", cert.Subject.CommonName, cert.NotAfter.Format("Jan 02 2006"))
			certIssues = true
		}
	}

	if certIssues {
		log.Warningf("Client certificate issues may prevent successful connection.")
	}
}
