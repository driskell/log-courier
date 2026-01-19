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
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/transports"
)

type receiverTCP struct {
	// Constructor
	ctx          context.Context
	shutdownFunc context.CancelFunc
	config       *ReceiverFactory
	factory      transports.ReceiverFactory
	bind         string
	eventChan    chan<- transports.Event
	connections  map[*connection]*connection
	backoff      *core.ExpBackoff

	// Internal
	connMutex       sync.Mutex
	connWait        sync.WaitGroup
	shutdownChan    chan struct{}
	shutdownOnce    sync.Once
	protocolFactory ProtocolFactory
}

// Factory returns the associated factory
func (t *receiverTCP) Factory() transports.ReceiverFactory {
	return t.factory
}

func (t *receiverTCP) SupportsAck() bool {
	return t.protocolFactory.SupportsAck()
}

// startController starts the controller
func (t *receiverTCP) startController() {
	go t.controllerRoutine()
}

// controllerRoutine managed restarting listening as things fail
func (t *receiverTCP) controllerRoutine() {
	defer func() {
		t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Finished, nil)
	}()

	for {
		err := t.listen()
		if err == nil {
			// Shutdown
			break
		}

		log.Errorf("[R %s] Receiver error, resetting: %s", t.bind, err)

		if t.retryWait() {
			break
		}
	}

	// Request all connections to stop receiving and wait for them to finally close once the final ack is sent and nil message sent
	log.Infof("[R %s] Receiver shutting down and waiting for final acknowledgements to be sent", t.bind)
	t.connMutex.Lock()
	for _, conn := range t.connections {
		t.ShutdownConnectionRead(conn.ctx, fmt.Errorf("exiting"))
	}
	t.connMutex.Unlock()
	t.connWait.Wait()

	// Ensure resources are cleaned up for the context
	t.shutdownFunc()

	log.Infof("[R %s] Receiver exiting", t.bind)
}

// retryWait waits the backoff timeout before attempting to listen again
// It also monitors for shutdown whilst waiting
func (t *receiverTCP) retryWait() bool {
	now := time.Now()
	setupDue := now.Add(t.backoff.Trigger())

	select {
	case <-t.shutdownChan:
		// Shutdown request
		return true
	case <-time.After(setupDue.Sub(now)):
	}

	return false
}

// listen configures and begins the accept loop routine
func (t *receiverTCP) listen() error {
	addr, err := net.ResolveTCPAddr("tcp", t.bind)
	if err != nil {
		return fmt.Errorf("failed to select next address: %s", err)
	}

	log.Infof("[R %s] Attempting to listen", t.bind)

	tcplistener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %s", t.bind, err)
	}

	log.Noticef("[R %s] Listening", t.bind)

	return t.acceptLoop(t.bind, tcplistener)
}

// acceptLoop creates new connections and pushes them to the controller to
// register and start
func (t *receiverTCP) acceptLoop(desc string, tcplistener *net.TCPListener) error {
	for {
		tcplistener.SetDeadline(time.Now().Add(time.Second))
		socket, err := tcplistener.Accept()
		if err == nil {
			t.startConnection(socket.(*net.TCPConn))
		} else if neterr := err.(net.Error); !neterr.Timeout() {
			return fmt.Errorf("failed to accept on %s: %s", desc, err)
		}

		// Check for shutdown request
		select {
		case <-t.shutdownChan:
			return nil
		default:
		}
	}
}

// Acknowledge sends the correct connection an acknowledgement
func (t *receiverTCP) Acknowledge(ctx context.Context, nonce *string, sequence uint32) error {
	connection := ctx.Value(transports.ContextConnection).(*connection)
	return connection.protocol.Acknowledge(nonce, sequence)
}

// Pong sends the correct connection a pong response
func (t *receiverTCP) Pong(ctx context.Context) error {
	connection := ctx.Value(transports.ContextConnection).(*connection)
	return connection.protocol.Pong()
}

// FailConnection shuts down a connection that has failed
func (t *receiverTCP) FailConnection(ctx context.Context, err error) {
	connection := ctx.Value(transports.ContextConnection).(*connection)
	log.Errorf("[R %s - %s] Failing connection: %s", connection.socket.LocalAddr().String(), connection.socket.RemoteAddr().String(), err)
	connection.Teardown()
}

// ShutdownConnection shuts down a connection gracefully by closing the send side
func (t *receiverTCP) ShutdownConnection(ctx context.Context) {
	connection := ctx.Value(transports.ContextConnection).(*connection)
	log.Debugf("[R %s - %s] Closing connection", connection.socket.LocalAddr().String(), connection.socket.RemoteAddr().String())
	if err := connection.SendMessage(nil); err != nil {
		t.FailConnection(ctx, err)
	}
}

// ShutdownConnectionRead shuts down a connection gracefully by closing the read side
func (t *receiverTCP) ShutdownConnectionRead(ctx context.Context, err error) {
	connection := ctx.Value(transports.ContextConnection).(*connection)
	log.Debugf("[R %s - %s] Aborting connection: %s", connection.socket.LocalAddr().String(), connection.socket.RemoteAddr().String(), err)
	connection.CloseRead()
}

// Shutdown shuts down the listener and all connections gracefully
func (t *receiverTCP) Shutdown() {
	t.shutdownOnce.Do(func() {
		close(t.shutdownChan)
	})
}

// getTLSConfig returns TLS configuration for the connection
func (t *receiverTCP) getTLSConfig() (tlsConfig *tls.Config) {
	tlsConfig = new(tls.Config)

	tlsConfig.MinVersion = t.config.MinTLSVersion
	tlsConfig.MaxVersion = t.config.MaxTLSVersion

	// Set the certificate if we set one
	if t.config.Certificate != nil {
		tlsConfig.Certificates = []tls.Certificate{*t.config.Certificate}
	}

	// Set CA for client verification
	tlsConfig.ClientCAs = x509.NewCertPool()
	for _, cert := range t.config.CaList {
		tlsConfig.ClientCAs.AddCert(cert)
	}

	if len(t.config.CaList) != 0 && t.config.SSLVerifyPeers {
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return
}

// startConnection sets up a new connection
func (t *receiverTCP) startConnection(socket *net.TCPConn) {
	log.Debugf("[R %s - %s] New connection", socket.LocalAddr().String(), socket.RemoteAddr().String())

	var connectionSocket connectionSocket
	if t.config.EnableTls {
		connectionSocket = newConnectionSocketTLS(socket, t.getTLSConfig(), true, t.bind)
	} else {
		connectionSocket = newConnectionSocketTCP(socket)
	}

	conn := newConnection(t.ctx, connectionSocket, t.protocolFactory, false, t.eventChan)

	t.connMutex.Lock()
	t.connections[conn] = conn
	t.connMutex.Unlock()

	t.connWait.Add(1)
	go t.connectionRoutine(socket, conn)
}

// connectionRoutine is a routine for an individual connection that runs it and captures shutdown
func (t *receiverTCP) connectionRoutine(socket net.Conn, conn *connection) {
	defer t.connWait.Done()

	didStart := false
	if err := conn.run(func() {
		t.eventChan <- transports.NewConnectEvent(conn.ctx, socket.RemoteAddr().String(), conn.socket.Desc())
		didStart = true
	}); err != nil {
		if err == ErrHardCloseRequested {
			log.Noticef("[R %s - %s] Client forcefully disconnected", socket.LocalAddr().String(), socket.RemoteAddr().String())
		} else if err != io.EOF { // Ignore io.EOF as it usually means a graceful close without starting up, such as a status check on a TLS port
			log.Errorf("[R %s - %s] Client failed: %s", socket.LocalAddr().String(), socket.RemoteAddr().String(), err)
		}
	} else {
		log.Noticef("[R %s - %s] Client closed", socket.LocalAddr().String(), socket.RemoteAddr().String())
	}

	if didStart {
		t.eventChan <- transports.NewDisconnectEvent(conn.ctx, socket.RemoteAddr().String(), conn.socket.Desc())
	}

	t.connMutex.Lock()
	delete(t.connections, conn)
	t.connMutex.Unlock()
}
