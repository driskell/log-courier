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
	tlsConfig  *tls.Config
	connCount  int
	connMutex  sync.Mutex
	connWait   sync.WaitGroup
	listenWait sync.WaitGroup
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
	t.listenWait.Add(1)
	go t.controllerRoutine()
}

// controllerRoutine managed restarting listening as things fail
func (t *receiverTCP) controllerRoutine() {
	defer func() {
		t.listenWait.Done()
	}()

	for {
		err := t.listen()
		if err == nil {
			// Shutdown
			break
		}

		log.Error("Receiver error, resetting: %s", err)

		if t.retryWait() {
			break
		}
	}

	t.closeConnections()

	log.Info("Receiver shutdown")
}

// retryWait waits the backoff timeout before attempting to listen again
// It also monitors for shutdown whilst waiting
func (t *receiverTCP) retryWait() bool {
	now := time.Now()
	setupDue := now.Add(t.backoff.Trigger())

	select {
	case <-t.ctx.Done():
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
		return fmt.Errorf("Failed to select next address: %s", err)
	}

	desc := t.pool.Desc()

	log.Info("[%s] Attempting to listen on %s", t.pool.Server(), desc)

	tcplistener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return fmt.Errorf("Failed to listen on %s: %s", desc, err)
	}

	log.Notice("[%s] Listening on %s", t.pool.Server(), desc)

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
			return fmt.Errorf("Failed to accept on %s: %s", desc, err)
		}

		// Check for shutdown request
		select {
		case <-t.ctx.Done():
			return nil
		default:
		}
	}
}

// stop ends the accept loop
func (t *receiverTCP) Shutdown() {
	t.shutdownFunc()
	t.listenWait.Wait()
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
	log.Notice("[%s < %s] New connection", t.pool.Server(), socket.RemoteAddr().String())

	// Only acknowledgements will be sent
	// TODO: Consider actual needed value, as we shouldn't expect ACK to block as they are tiny sends
	sendChan := make(chan protocolMessage, 10)

	var connectionSocket connectionSocket
	if t.config.transport == TransportTCPTLS {
		connectionSocket = newConnectionSocketTLS(socket, t.getTLSConfig(), true, t.pool.Server())
	} else {
		connectionSocket = newConnectionSocketTCP(socket)
	}

	connContext := context.WithValue(t.ctx, contextIsClient, false)
	conn := newConnection(connContext, connectionSocket, t.pool.Server(), t.eventChan, sendChan)

	t.connMutex.Lock()
	t.connCount++
	t.connections[conn] = conn
	t.connMutex.Unlock()

	t.connWait.Add(1)
	go t.connectionRoutine(socket, conn)
}

func (t *receiverTCP) closeConnections() {
	t.connMutex.Lock()
	for conn := range t.connections {
		conn.Teardown()
	}
	t.connMutex.Unlock()
	t.connWait.Wait()
}

// connectionRoutine is a routine for an individual connection that runs it and captures shutdown
func (t *receiverTCP) connectionRoutine(socket net.Conn, conn *connection) {
	defer t.connWait.Done()

	if err := conn.Run(nil); err != nil {
		if err == errHardCloseRequested {
			log.Info("[%s < %s] Connection shutdown requested", t.pool.Server(), socket.RemoteAddr().String())
		} else {
			log.Error("[%s < %s] Connection failed: %s", t.pool.Server(), socket.RemoteAddr().String(), err)
		}
	} else {
		log.Info("[%s < %s] Connection closed gracefully", t.pool.Server(), socket.RemoteAddr().String())
	}

	t.connMutex.Lock()
	t.connCount--
	delete(t.connections, conn)
	t.connMutex.Unlock()
}
