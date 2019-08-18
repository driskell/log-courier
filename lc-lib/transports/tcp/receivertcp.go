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
	"fmt"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// ReceiverTCP implements a transport that receives over TCP
// It also can optionally introduce a TLS layer for security
type ReceiverTCP struct {
	config    *ReceiverTCPFactory
	netConfig *transports.ReceiverConfig
	listener  listener
	backoff   *core.ExpBackoff

	controllerChan chan error
	context        interface{}
	eventChan      chan<- transports.Event
	pool           *addresspool.Pool
	connectionChan chan *socketMessage
	sendChans      map[connection]chan protocolMessage
	connCount      int
}

// ReloadConfig returns true if the transport needs to be restarted in order
// for the new configuration to apply
// TODO: Update where config comes from
func (t *ReceiverTCP) ReloadConfig(cfg *config.Config) bool {
	newNetConfig := transports.FetchReceiverConfig(cfg)
	newConfig := newNetConfig.Factory.(*ReceiverTCPFactory)

	// TODO: Check timestamps of underlying certificate files to detect changes
	if newConfig.SSLCertificate != t.config.SSLCertificate || newConfig.SSLKey != t.config.SSLKey {
		return true
	}
	if len(newConfig.SSLClientCA) != len(t.config.SSLClientCA) {
		return true
	}
	for i, clientCA := range newConfig.SSLClientCA {
		if clientCA != t.config.SSLClientCA[i] {
			return true
		}
	}

	// Only copy net config just in case something in the factory did change that
	// we didn't account for which does require a restart
	t.netConfig = newNetConfig

	return false
}

// controller is the master routine which handles listening and resetting listen on network changes
func (t *ReceiverTCP) controller() {
	// Main listen loop
	for {
		var (
			err      error
			shutdown bool
		)

		shutdown, err = t.setup()
		if shutdown {
			t.teardown()
			t.closeConnections()
			return
		}
		if err == nil {
			// Connected - sit and wait for shutdown or socket message
			t.backoff.Reset()
			err = t.monitor()
		}

		if err != nil {
			log.Errorf("[%s] Receiver error, resetting: %s", t.pool.Server(), err)
		} else {
			log.Info("[%s] Receiver resetting", t.pool.Server())
		}

		t.teardown()

		// If this returns false, we are shutting down
		if !t.setupWait() {
			break
		}
	}

	t.closeConnections()
}

// monitor processes new connections and waits for shutdown or an error
func (t *ReceiverTCP) monitor() (err error) {
	for {
		select {
		// TODO: Handle configuration reload
		case err = <-t.controllerChan:
			if err == nil {
				// Shutdown request
				t.teardown()
				t.closeConnections()
				return
			}

			// Transport error - reset
			break
		case message := <-t.connectionChan:
			// Handle new connection and broken connections
			if message.err != nil {
				t.connCount--
				delete(t.sendChans, message.conn)
				message.conn.Teardown()
			} else {
				t.connCount++
				t.sendChans[message.conn] = message.conn.SendChan()
				message.conn.Setup()
			}
		}
	}
}

// setupWait waits the backoff timeout before attempting to listen again
// It also monitors for shutdown and configuration reload events while waiting
func (t *ReceiverTCP) setupWait() bool {
	now := time.Now()
	setupDue := now.Add(t.backoff.Trigger())

	select {
	// TODO: Handle configuration reload
	case <-t.controllerChan:
		// Shutdown request
		return false
	case <-time.After(setupDue.Sub(now)):
	}

	return true
}

// setup selects the next address to use and starts the listener
// Returns an error and also true if shutdown was detected
func (t *ReceiverTCP) setup() (bool, error) {
	addr, err := t.pool.Next()
	if err != nil {
		return false, fmt.Errorf("Failed to select next address: %s", err)
	}

	desc := t.pool.Desc()

	log.Info("[%s] Attempting to setup %s", t.pool.Server(), desc)

	if t.config.transport == TransportTCPTLS {
		t.listener = &listenerTLS{
			listenerTCP: listenerTCP{
				context:        t.context,
				pool:           t.pool,
				eventChan:      t.eventChan,
				config:         t.config,
				netConfig:      t.netConfig,
				connectionChan: t.connectionChan,
			},
		}
	} else {
		t.listener = &listenerTCP{
			context:        t.context,
			pool:           t.pool,
			eventChan:      t.eventChan,
			config:         t.config,
			netConfig:      t.netConfig,
			connectionChan: t.connectionChan,
		}
	}

	return t.listener.Start(desc, addr)
}

// teardown shuts down the listener
func (t *ReceiverTCP) teardown() {
	t.listener.Stop()
	t.listener = nil
}

// closeConnections closes all connections
func (t *ReceiverTCP) closeConnections() {
	for conn := range t.sendChans {
		conn.Teardown()
	}

	t.connCount = 0
}

// Shutdown the transport
func (t *ReceiverTCP) Shutdown() {
	close(t.controllerChan)
}
