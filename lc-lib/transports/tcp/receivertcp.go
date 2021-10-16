package tcp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/transports"
)

type receiverTCP struct {
	// Constructor
	ctx          context.Context
	shutdownFunc context.CancelFunc
	config       *ReceiverTCPFactory
	pool         *addresspool.Pool
	eventChan    chan<- transports.Event
	connections  map[*connection]*connection
	backoff      *core.ExpBackoff

	// Internal
	connMutex    sync.Mutex
	connWait     sync.WaitGroup
	shutdownChan chan struct{}
	shutdownOnce sync.Once
}

// ReloadConfig returns true if the transport needs to be restarted in order
// for the new configuration to apply
func (t *receiverTCP) ReloadConfig(cfg *config.Config, factory transports.ReceiverFactory) bool {
	newConfig := factory.(*ReceiverTCPFactory)

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

	return false
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

		log.Errorf("[R %s] Receiver error, resetting: %s", t.pool.Server(), err)

		if t.retryWait() {
			break
		}
	}

	// Request all connections to stop receiving and wait for them to finally close once the final ack is sent and nil message sent
	log.Infof("[R %s] Receiver shutting down and waiting for final acknowledgements to be sent", t.pool.Server())
	t.connMutex.Lock()
	for _, connection := range t.connections {
		connection.CloseRead()
	}
	t.connMutex.Unlock()
	t.connWait.Wait()

	// Ensure resources are cleaned up for the context
	t.shutdownFunc()

	log.Infof("[R %s] Receiver exiting", t.pool.Server())
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
	addr, err := t.pool.Next()
	if err != nil {
		return fmt.Errorf("failed to select next address: %s", err)
	}

	desc := t.pool.Desc()

	log.Infof("[R %s] Attempting to listen on %s", t.pool.Server(), desc)

	tcplistener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %s", desc, err)
	}

	log.Noticef("[R %s] Listening on %s", tcplistener.Addr().String(), desc)

	return t.acceptLoop(desc, tcplistener)
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
	log.Debugf("[R %s > %s] Sending acknowledgement for nonce %x with sequence %d", connection.socket.LocalAddr().String(), connection.socket.RemoteAddr().String(), *nonce, sequence)
	return connection.SendMessage(&protocolACKN{nonce: nonce, sequence: sequence})
}

// Pong sends the correct connection a pong response
func (t *receiverTCP) Pong(ctx context.Context) error {
	connection := ctx.Value(transports.ContextConnection).(*connection)
	log.Debugf("[R %s > %s] Sending pong", connection.socket.LocalAddr().String(), connection.socket.RemoteAddr().String())
	return connection.SendMessage(&protocolPONG{})
}

// FailConnection shuts down a connection that has failed
func (t *receiverTCP) FailConnection(ctx context.Context, err error) {
	connection := ctx.Value(transports.ContextConnection).(*connection)
	log.Errorf("[R %s - %s] Client failed: %s", connection.socket.LocalAddr().String(), connection.socket.RemoteAddr().String(), err)
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

// Shutdown shuts down the listener and all connections gracefully
func (t *receiverTCP) Shutdown() {
	t.shutdownOnce.Do(func() {
		close(t.shutdownChan)
	})
}

// getTLSConfig returns TLS configuration for the connection
func (t *receiverTCP) getTLSConfig() (tlsConfig *tls.Config) {
	tlsConfig = new(tls.Config)

	tlsConfig.MinVersion = t.config.minTLSVersion
	tlsConfig.MaxVersion = t.config.maxTLSVersion

	// Set the certificate if we set one
	if t.config.certificate != nil {
		tlsConfig.Certificates = []tls.Certificate{*t.config.certificate}
	}

	// Set CA for client verification
	tlsConfig.ClientCAs = x509.NewCertPool()
	for _, cert := range t.config.caList {
		tlsConfig.ClientCAs.AddCert(cert)
	}

	if len(t.config.caList) != 0 && t.config.SSLVerifyPeers {
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return
}

// startConnection sets up a new connection
func (t *receiverTCP) startConnection(socket *net.TCPConn) {
	log.Noticef("[R %s - %s] New connection", socket.LocalAddr().String(), socket.RemoteAddr().String())

	var connectionSocket connectionSocket
	if t.config.transport == TransportTCPTLS {
		connectionSocket = newConnectionSocketTLS(socket, t.getTLSConfig(), true, t.pool.Server())
	} else {
		connectionSocket = newConnectionSocketTCP(socket)
	}

	conn := newConnection(t.ctx, connectionSocket, t.pool.Server(), false, t.eventChan)

	t.connMutex.Lock()
	t.connections[conn] = conn
	t.connMutex.Unlock()

	t.connWait.Add(1)
	go t.connectionRoutine(socket, conn)
}

// connectionRoutine is a routine for an individual connection that runs it and captures shutdown
func (t *receiverTCP) connectionRoutine(socket net.Conn, conn *connection) {
	defer t.connWait.Done()

	if err := conn.Run(func() {
		conn.sendEvent(transports.NewConnectEvent(conn.ctx))
	}); err != nil {
		if err == errHardCloseRequested {
			log.Noticef("[R %s - %s] Client forcefully disconnected", socket.LocalAddr().String(), socket.RemoteAddr().String())
		} else {
			log.Errorf("[R %s - %s] Client failed: %s", socket.LocalAddr().String(), socket.RemoteAddr().String(), err)
		}
	} else {
		log.Noticef("[R %s - %s] Client closed", socket.LocalAddr().String(), socket.RemoteAddr().String())
	}

	t.connMutex.Lock()
	delete(t.connections, conn)
	t.connMutex.Unlock()
}
