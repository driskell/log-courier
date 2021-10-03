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
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/transports"
)

// errHardCloseRequested is used to signal hard close requested
var (
	// ErrInvalidState occurs when a send cannot happen because the connection has closed
	ErrInvalidState = errors.New("invalid connection state")

	// errHardCloseRequested occurs when the connection was closed forcefully and a hard close is requested
	errHardCloseRequested = errors.New("connection shutdown was requested")
)

type connection struct {
	// ctx uniquely identifies the connection - a new one is allocated by newConnection
	ctx          context.Context
	socket       connectionSocket
	poolServer   string
	isClient     bool
	eventChan    chan<- transports.Event
	sendChan     chan protocolMessage
	shutdownChan chan struct{}

	supportsEvnt bool
	rwBuffer     bufio.ReadWriter

	// receiverShutdownMutex ensures receiver shutdown happens once as it can be triggered externally and by sender
	receiverShutdownMutex sync.RWMutex
	// shutdown is flagged when receiver is shutting down
	receiverShutdown bool
	// receiverShutdownChan requests receiver to stop receiving data
	receiverShutdownChan chan struct{}
	// senderErr stores exit error from sender
	senderErr error
	// senderShutdownChan requests sender to begin to gracefully shutdown - called from receiver when EOF
	senderShutdownChan chan struct{}
	// wait can be used to wait for all routines to complete
	wait sync.WaitGroup
}

func newConnection(ctx context.Context, socket connectionSocket, poolServer string, isClient bool, eventChan chan<- transports.Event) *connection {
	ret := &connection{
		socket:               socket,
		poolServer:           poolServer,
		isClient:             isClient,
		eventChan:            eventChan,
		sendChan:             make(chan protocolMessage, 1),
		shutdownChan:         make(chan struct{}),
		receiverShutdownChan: make(chan struct{}),
		senderShutdownChan:   make(chan struct{}),
	}

	ret.ctx = context.WithValue(ctx, transports.ContextConnection, ret)

	return ret
}

// Run starts the connection and all its routines
func (t *connection) Run(startedCallback func()) error {
	t.rwBuffer.Reader = bufio.NewReader(t.socket)
	t.rwBuffer.Writer = bufio.NewWriter(t.socket)

	if err := t.socket.Setup(); err != nil {
		return err
	}

	if t.isClient {
		if err := t.clientNegotiation(); err != nil {
			return err
		}
	} else {
		if err := t.serverNegotiation(); err != nil {
			return err
		}
	}

	if startedCallback != nil {
		startedCallback()
	}

	t.wait.Add(1)
	go t.senderRoutine()
	err := t.receiver()
	// Request sender shutdown
	close(t.senderShutdownChan)
	// Wait for sender to complete
	t.wait.Wait()

	// Cleanup
	t.socket.Close()

	// If we had no error, did sender save one when it shutdown?
	// This is already the receiver error if receiver collected it and shutdown
	// If receiver EOF though and then we had a sender issue, take that error
	if err == nil {
		err = t.senderErr
	}

	// Unblock any external routines communicating via SendMessage
	close(t.shutdownChan)

	return err
}

// serverNegotiation works out the protocol version supported by the remote
func (t *connection) serverNegotiation() error {
	message, err := t.readMsg()
	if message == nil {
		if err == nil {
			err = io.EOF
		} else if err == errHardCloseRequested {
			return err
		}
		return fmt.Errorf("unexpected end of negotiation: %s", err)
	}

	_, ok := message.(*protocolHELO)
	if !ok {
		if messageImpl, ok := message.(eventsMessage); ok {
			// Backwards compatible path with older log-courier which do not perform a negotiation
			event := transports.NewEventsEvent(t.ctx, messageImpl.Nonce(), messageImpl.Events())
			if t.sendEvent(event) {
				return errHardCloseRequested
			}
			// Now go async
			return nil
		}
		return fmt.Errorf("unexpected %T during negotiation, expected protocolHELO", message)
	}

	if err := t.writeMsg(createProtocolVERS()); err != nil {
		return err
	}

	return nil
}

// clientNegotiation works out the protocol version supported by the remote
func (t *connection) clientNegotiation() error {
	if err := t.writeMsg(&protocolHELO{}); err != nil {
		return err
	}

	message, err := t.readMsg()
	if message == nil {
		if err == nil {
			err = io.EOF
		} else if err == errHardCloseRequested {
			return err
		}
		return fmt.Errorf("unexpected end of negotiation: %s", err)
	}

	versMessage, ok := message.(*protocolVERS)
	if !ok {
		if _, isUnkn := message.(*protocolUNKN); isUnkn {
			versMessage = &protocolVERS{protocolFlags: []byte{}}
		} else {
			versMessage = nil
		}
	}

	if versMessage == nil {
		return fmt.Errorf("unexpected %T reply to negotiation, expected protocolVERS", message)
	}

	t.supportsEvnt = versMessage.SupportsEVNT()
	if t.supportsEvnt {
		log.Debug("[%s < %s] Remote supports enhanced EVNT messages", t.poolServer, t.socket.RemoteAddr().String())
	}

	return nil
}

// senderRoutine wraps the sender into a routine and handles error communication
func (t *connection) senderRoutine() {
	defer func() {
		t.wait.Done()
	}()

	t.senderErr = t.sender()

	// Sender exits after graceful shutdown from receiver EOF
	// Or it exits due to a problem
	// Therefore, always force shutdown here of receiver if it wasn't already
	// This will unblock everything, including attempts to send data that won't now complete
	// as they all monitor the main shutdown channel
	t.Teardown()
}

// sender handles socket writes
func (t *connection) sender() error {
	for {
		var msg protocolMessage

		select {
		case <-t.senderShutdownChan:
			return nil
		case msg = <-t.sendChan:
			// Is this the end message? nil? No more to send?
			if msg == nil {
				// Close send side to generate EOF on remote
				return t.socket.CloseWrite()
			}
		}

		if err := t.writeMsg(msg); err != nil {
			return err
		}
	}
}

// writeMsg sends a message
func (t *connection) writeMsg(msg protocolMessage) error {
	// Write deadline is managed by our net.Conn wrapper that TLS will call
	// into and keeps retrying writes until timeout or error
	err := msg.Write(t)
	if err == nil {
		err = t.rwBuffer.Flush()
	}
	if err != nil {
		// Fail the transport
		return err
	}
	return nil
}

// sendEvent ships an event structure whilst also monitoring for
// any shutdown signal. Returns true if shutdown was signalled
func (t *connection) sendEvent(transportEvent transports.Event) bool {
	select {
	case <-t.receiverShutdownChan:
		return true
	case t.eventChan <- transportEvent:
	}
	return false
}

// receiver handles socket reads
// Returns nil error on shutdown, or an actual error
func (t *connection) receiver() error {
	for {
		message, err := t.readMsg()
		if message == nil {
			// Message is nil and err nil on EOF, return err or nil back
			return err
		}

		var transportEvent transports.Event = nil
		// TODO: Can events be interfaces so we can just pass on the message itself if it implements the interface?
		if t.isClient {
			switch messageImpl := message.(type) {
			case *protocolACKN:
				transportEvent = transports.NewAckEvent(t.ctx, messageImpl.nonce, messageImpl.sequence)
			case *protocolPONG:
				transportEvent = transports.NewPongEvent(t.ctx)
			}
		} else {
			switch messageImpl := message.(type) {
			case *protocolPING:
				transportEvent = transports.NewPingEvent(t.ctx)
			case eventsMessage:
				transportEvent = transports.NewEventsEvent(t.ctx, messageImpl.Nonce(), messageImpl.Events())
			}
		}

		if transportEvent == nil {
			return fmt.Errorf("unknown protocol message %T", message)
		}

		if shutdown := t.sendEvent(transportEvent); shutdown {
			return nil
		}
	}
}

// readMsg reads a message from the connection
// Returns nil message if shutdown, with an optional error
func (t *connection) readMsg() (protocolMessage, error) {
	var header [8]byte

	if shutdown, err := t.receiverRead(header[:]); shutdown || err != nil {
		if err != nil {
			if err == io.EOF {
				return nil, nil
			}
		}
		return nil, err
	}

	// Grab length of message
	bodyLength := binary.BigEndian.Uint32(header[4:8])

	var newFunc func(*connection, uint32) (protocolMessage, error)
	switch {
	case bytes.Equal(header[0:4], []byte("????")): // UNKN
		newFunc = newProtocolUNKN
	case bytes.Equal(header[0:4], []byte("HELO")):
		newFunc = newProtocolHELO
	case bytes.Equal(header[0:4], []byte("VERS")):
		newFunc = newProtocolVERS
	case bytes.Equal(header[0:4], []byte("PING")):
		newFunc = newProtocolPING
	case bytes.Equal(header[0:4], []byte("PONG")):
		newFunc = newProtocolPONG
	case bytes.Equal(header[0:4], []byte("ACKN")):
		newFunc = newProtocolACKN
	case bytes.Equal(header[0:4], []byte("JDAT")):
		newFunc = newProtocolJDAT
	case bytes.Equal(header[0:4], []byte("EVNT")):
		newFunc = newProtocolEVNT
	default:
		return nil, fmt.Errorf("unexpected message code: %s", header[0:4])
	}

	return newFunc(t, bodyLength)
}

// receiverRead will repeatedly read from the socket until the given byte array
// is filled, or sender signals an error
func (t *connection) receiverRead(data []byte) (bool, error) {
	var err error
	received := 0

ReceiverReadLoop:
	for {
		select {
		case <-t.receiverShutdownChan:
			err = errHardCloseRequested
			break ReceiverReadLoop
		default:
			// Timeout after socketIntervalSeconds, check for shutdown, and try again
			t.socket.SetReadDeadline(time.Now().Add(socketIntervalSeconds * time.Second))

			length, err := t.rwBuffer.Read(data[received:])
			received += length

			if received >= len(data) {
				// Success
				return false, nil
			}

			if err == nil {
				// Keep trying
				continue
			}

			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Keep trying
				continue
			}

			// Pass an error back
			return false, err
		}
	}

	return true, err
}

// Teardown ends the connection
// It will end the receiver to stop receiving data
// Receivers do not gracefully shutdown from our side, only remote side, so this is called to stop receiving more data
// For Transports this is used to teardown due to a problem
func (t *connection) Teardown() {
	t.receiverShutdownMutex.Lock()
	defer t.receiverShutdownMutex.Unlock()
	if t.receiverShutdown {
		return
	}
	t.receiverShutdown = true
	close(t.receiverShutdownChan)
}

// SendMessage queues a message to be sent on the connection
// Send a nil to close the connection gracefully
// If nil is sent twice - there will be a panic - so only request shutdown ONCE
// If the connection is running too slow and the queue is full then ErrSendQueueFull is returned
// If the connection has closed it will return ErrInvalidState
func (t *connection) SendMessage(message protocolMessage) error {
	if message == nil {
		close(t.sendChan)
		return nil
	}

	select {
	case <-t.shutdownChan:
		return ErrInvalidState
	case t.sendChan <- message:
	default:
		return transports.ErrCongestion
	}
	return nil
}

// SupportsEVNT returns true if this connection supports EVNT messages
func (t *connection) SupportsEVNT() bool {
	return t.supportsEvnt
}

// Read will receive data from the connection and implements io.Reader
// Used by protocol structures to fetch message data
// It should be noted that Read here will never return an incomplete read, and as such
// the int return will always be the length of the message, unless an error occurs, in
// which case it will be 0.
func (t *connection) Read(message []byte) (int, error) {
	if shutdown, err := t.receiverRead(message); shutdown || err != nil {
		return 0, err
	}
	return len(message), nil
}

// ReadByte will receive one byte from the connection and implement io.ByteReader
// Used by protocol structures to fetch message data
// io.ByteReader is necessary for compression readers to not read too much
func (t *connection) ReadByte() (byte, error) {
	var message [1]byte
	if _, err := t.Read(message[:]); err != nil {
		return 0, err
	}
	return message[0], nil
}

// Write data to the connection and implements io.Reader
// Used by protocol structures to send a message
func (t *connection) Write(data []byte) (int, error) {
	return t.rwBuffer.Write(data)
}
