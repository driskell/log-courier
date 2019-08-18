package tcp

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/transports"
)

type listenerTCP struct {
	context        interface{}
	eventChan      chan<- transports.Event
	pool           *addresspool.Pool
	config         *ReceiverTCPFactory
	netConfig      *transports.ReceiverConfig
	connectionChan chan<- *socketMessage

	listenControl chan struct{}
	listenWait    sync.WaitGroup
}

// start configures and begins the accept loop routine
func (t *listenerTCP) Start(desc string, addr *net.TCPAddr) (bool, error) {
	tcplistener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return false, fmt.Errorf("Failed to listen on %s: %s", desc, err)
	}

	t.listenControl = make(chan struct{})
	t.listenWait.Add(1)

	go t.acceptLoop(desc, tcplistener)
	return false, nil
}

// acceptLoop creates new connections and pushes them to the controller to
// register and start
func (t *listenerTCP) acceptLoop(desc string, tcplistener *net.TCPListener) {
	defer func() {
		t.listenWait.Done()
	}()

	for {
		tcplistener.SetDeadline(time.Now().Add(time.Second))
		socket, err := tcplistener.Accept()
		if err == nil {
			t.newConnection(socket)
			continue
		}

		if neterr := err.(net.Error); !neterr.Timeout() {
			t.connectionChan <- &socketMessage{err: fmt.Errorf("Failed to accept on %s: %s", desc, err)}
			return
		}

		// Check for shutdown request
		select {
		case <-t.listenControl:
			return
		default:
		}
	}
}

// stop ends the accept loop
func (t *listenerTCP) Stop() {
	close(t.listenControl)
	t.listenControl = nil
	t.listenWait.Wait()
}

// newConnection sets up a new connection
func (t *listenerTCP) newConnection(socket net.Conn) {
	log.Notice("[%s] New connection from %s", t.pool.Server(), socket.RemoteAddr().String())

	conn := &connectionTCP{
		socket:         socket,
		context:        t.context,
		eventChan:      t.eventChan,
		poolServer:     t.pool.Server(),
		connectionChan: t.connectionChan,
		server:         true,
		// TODO: Additional small routine to collapse same-nonce acks into a 10-size channel matching receive limit?
		sendChan: make(chan protocolMessage, 10),
	}

	t.connectionChan <- &socketMessage{conn: conn}
}
