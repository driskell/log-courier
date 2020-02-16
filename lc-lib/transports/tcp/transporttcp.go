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
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/payload"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// TransportTCP implements a transport that sends over TCP
// It also can optionally introduce a TLS layer for security
type TransportTCP struct {
	config       *TransportTCPFactory
	netConfig    *transports.Config
	finishOnFail bool
	backoff      *core.ExpBackoff

	controllerChan chan error
	context        interface{}
	pool           *addresspool.Pool
	eventChan      chan<- transports.Event
	connectionChan chan *socketMessage
	conn           connection
}

// ReloadConfig returns true if the transport needs to be restarted in order
// for the new configuration to apply
func (t *TransportTCP) ReloadConfig(netConfig *transports.Config, finishOnFail bool) bool {
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

// controller is the master routine which handles connection and reconnection
// When reconnecting, the socket and all routines are torn down and restarted
func (t *TransportTCP) controller() {
	defer func() {
		t.eventChan <- transports.NewStatusEvent(t.context, transports.Finished)
	}()

	// Main connect loop
	for {
		var err error
		var shutdown bool

		shutdown, err = t.connect()
		if shutdown {
			t.disconnect()
			return
		}
		if err == nil {
			// Connected - sit and wait for shutdown or socket message
			t.backoff.Reset()

			err = t.conn.Run()
		}

		if err != nil {
			if t.finishOnFail {
				log.Errorf("[%s] Transport error: %s", t.pool.Server(), err)
				t.disconnect()
				return
			}

			log.Errorf("[%s] Transport error, disconnecting: %s", t.pool.Server(), err)
		} else {
			log.Info("[%s] Transport disconnecting", t.pool.Server())
		}

		t.disconnect()

		select {
		case <-t.controllerChan:
			return
		case t.eventChan <- transports.NewStatusEvent(t.context, transports.Failed):
		}

		// If this returns false, we are shutting down
		if !t.reconnectWait() {
			break
		}
	}
}

// reconnectWait waits the backoff timeout before attempting to reconnect
// It also monitors for shutdown and configuration reload events while waiting.
func (t *TransportTCP) reconnectWait() bool {
	now := time.Now()
	reconnectDue := now.Add(t.backoff.Trigger())

	select {
	// TODO: Handle configuration reload
	case <-t.controllerChan:
		// Shutdown request
		return false
	case <-time.After(reconnectDue.Sub(now)):
	}

	return true
}

// connect selects the next address to use and triggers the connect
// Returns an error and also true if shutdown was detected
func (t *TransportTCP) connect() (bool, error) {
	t.checkClientCertificates()

	addr, err := t.pool.Next()
	if err != nil {
		return false, fmt.Errorf("Failed to select next address: %s", err)
	}

	desc := t.pool.Desc()

	log.Info("[%s] Attempting to connect to %s", t.pool.Server(), desc)

	socket, err := net.DialTimeout("tcp", addr.String(), t.netConfig.Timeout)
	if err != nil {
		return false, fmt.Errorf("Failed to connect to %s: %s", desc, err)
	}

	// Maxmium number of payloads we will queue - must be correct as otherwise
	// the endpoint implementation will block unexpectedly and be unable to
	// distribute events across multiple transports
	// Pings don't count towards limit, we only ping if this queue is empty
	sendChan := make(chan protocolMessage, t.netConfig.MaxPendingPayloads)

	// Now wrap in TLS if this is the TLS transport
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

		t.conn = &connectionTLS{
			connectionTCP: connectionTCP{
				socket:         socket,
				context:        t.context,
				poolServer:     t.pool.Server(),
				eventChan:      t.eventChan,
				connectionChan: t.connectionChan,
				sendChan:       sendChan,
			},
			tlsConfig: tlsConfig,
		}
	} else {
		t.conn = &connectionTCP{
			socket:         socket,
			context:        t.context,
			poolServer:     t.pool.Server(),
			connectionChan: t.connectionChan,
			sendChan:       sendChan,
		}
	}

	log.Notice("[%s] Connected to %s", t.pool.Server(), desc)

	return false, nil
}

// disconnect shuts down the connection
func (t *TransportTCP) disconnect() {
	if t.conn != nil {
		t.conn.Teardown()
	}

	t.conn = nil
}

// Write a message to the transport
func (t *TransportTCP) Write(payload *payload.Payload) error {
	if t.conn.SupportsEVNT() {
		t.conn.SendChan() <- &protocolEVNT{nonce: payload.Nonce, events: payload.Events()}
	} else {
		t.conn.SendChan() <- &protocolJDAT{nonce: payload.Nonce, events: payload.Events()}
	}
	return nil
}

// Ping the remote server
func (t *TransportTCP) Ping() error {
	t.conn.SendChan() <- &protocolPING{}
	return nil
}

// Fail the transport
func (t *TransportTCP) Fail() {
	select {
	case t.controllerChan <- transports.ErrForcedFailure:
	default:
	}
}

// Shutdown the transport
func (t *TransportTCP) Shutdown() {
	close(t.controllerChan)
}

// checkClientCertificates logs a warning if it finds any certificates that are
// not currently valid
func (t *TransportTCP) checkClientCertificates() {
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
