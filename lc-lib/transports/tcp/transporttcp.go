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
	"crypto/x509"
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
	config       *TransportTCPFactory
	netConfig    *transports.Config
	finishOnFail bool
	backoff      *core.ExpBackoff

	controllerChan chan error
	context        interface{}
	pool           *addresspool.Pool
	eventChan      chan<- transports.Event
	connectionChan chan *socketMessage
	conn           *connection

	sendMutex    sync.RWMutex
	sendChan     chan protocolMessage
	supportsEVNT bool
	shutdown     bool
}

// ReloadConfig returns true if the transport needs to be restarted in order
// for the new configuration to apply
// TODO: Modifying netConfig out of routines using it?
func (t *transportTCP) ReloadConfig(netConfig *transports.Config, finishOnFail bool) bool {
	newConfig := netConfig.Factory.(*TransportTCPFactory)
	t.finishOnFail = finishOnFail

	// TODO: Check timestamps of underlying certificate files to detect changes
	if newConfig.SSLCertificate != t.config.SSLCertificate || newConfig.SSLKey != t.config.SSLKey || newConfig.SSLCA != t.config.SSLCA {
		return true
	}

	// Only copy net config just in case something in the factory did change that
	// we didn't account for which does require a restart
	t.netConfig = netConfig

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
		t.eventChan <- transports.NewStatusEvent(t.context, transports.Finished)
	}()

	// Main connect loop
	for {
		var err error
		var shutdown bool

		err = t.connect()
		if err == nil {
			t.backoff.Reset()

			t.sendMutex.Lock()
			t.supportsEVNT = t.conn.SupportsEVNT()
			t.sendChan = t.conn.SendChan()
			t.sendMutex.Unlock()

			err = t.conn.Run()
		}

		t.sendMutex.Lock()
		shutdown = t.shutdown
		t.sendChan = nil
		t.sendMutex.Unlock()

		if err != nil {
			log.Errorf("[%s] Transport error, disconnected: %s", t.pool.Server(), err)
			if t.finishOnFail || shutdown {
				return
			}
		} else {
			log.Info("[%s] Transport disconnected gracefully", t.pool.Server())
			return
		}

		select {
		case <-t.controllerChan:
			// Ignore any error in controller chan - we're already restarting
		case t.eventChan <- transports.NewStatusEvent(t.context, transports.Failed):
			// If we had an error in controller chan, no need to flag in event chan we did error
			// So this eventChan send should skip if we find one in controllerChan
		}

		// If this returns false, we are shutting down
		if t.reconnectWait() {
			break
		}
	}
}

// reconnectWait waits the backoff timeout before attempting to reconnect
// It also monitors for shutdown and configuration reload events while waiting.
func (t *transportTCP) reconnectWait() bool {
	now := time.Now()
	reconnectDue := now.Add(t.backoff.Trigger())

	select {
	// TODO: Handle configuration reload
	case <-t.controllerChan:
		// Shutdown request
		return true
	case <-time.After(reconnectDue.Sub(now)):
	}

	return false
}

// connect selects the next address to use and triggers the connect
func (t *transportTCP) connect() error {
	t.checkClientCertificates()

	addr, err := t.pool.Next()
	if err != nil {
		return fmt.Errorf("Failed to select next address: %s", err)
	}

	desc := t.pool.Desc()

	log.Info("[%s] Attempting to connect to %s", t.pool.Server(), desc)

	socket, err := net.DialTimeout("tcp", addr.String(), t.netConfig.Timeout)
	if err != nil {
		return fmt.Errorf("Failed to connect to %s: %s", desc, err)
	}

	// Maxmium number of payloads we will queue - must be correct as otherwise
	// the endpoint implementation will block unexpectedly and be unable to
	// distribute events across multiple transports
	// Pings don't count towards limit, we only ping if this queue is empty
	sendChan := make(chan protocolMessage, t.netConfig.MaxPendingPayloads)

	// Now wrap in TLS if this is the TLS transport
	var connectionSocket connectionSocket
	if t.config.transport == TransportTCPTLS {
		tlsConfig := new(tls.Config)

		// Disable SSLv3 (mitigate POODLE vulnerability)
		tlsConfig.MinVersion = tls.VersionTLS10

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

		connectionSocket = newConnectionSocketTLS(socket.(*net.TCPConn), tlsConfig, false, t.pool.Server())
	} else {
		connectionSocket = newConnectionSocketTCP(socket.(*net.TCPConn))
	}

	t.conn = newConnection(connectionSocket, t.context, t.pool.Server(), t.eventChan, sendChan)

	log.Notice("[%s] Connected to %s", t.pool.Server(), desc)

	return nil
}

// Write a message to the transport - only valid after Started transport event received
func (t *transportTCP) Write(payload *payload.Payload) error {
	t.sendMutex.Lock()
	defer t.sendMutex.Unlock()
	if t.sendChan == nil {
		return fmt.Errorf("Invalid connection state")
	}
	var msg protocolMessage
	if t.supportsEVNT {
		msg = &protocolEVNT{nonce: payload.Nonce, events: payload.Events()}
	} else {
		msg = &protocolJDAT{nonce: payload.Nonce, events: payload.Events()}
	}
	t.sendChan <- msg
	return nil
}

// Ping the remote server - only valid after Started transport event received
func (t *transportTCP) Ping() error {
	t.sendMutex.Lock()
	defer t.sendMutex.Unlock()
	if t.sendChan == nil {
		return fmt.Errorf("Invalid connection state")
	}
	t.sendChan <- &protocolPING{}
	return nil
}

// Fail the transport
func (t *transportTCP) Fail() {
	select {
	case t.controllerChan <- transports.ErrForcedFailure:
	default:
	}
}

// Shutdown the transport - only valid after Started transport event received
func (t *transportTCP) Shutdown() {
	// Sending nil triggers graceful shutdown
	t.sendMutex.Lock()
	defer t.sendMutex.Unlock()
	if t.sendChan != nil {
		t.sendChan <- nil
	}
	t.shutdown = true
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
