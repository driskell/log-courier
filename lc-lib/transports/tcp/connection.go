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
	shutdownFunc context.CancelFunc
	socket       connectionSocket
	poolServer   string
	isClient     bool
	eventChan    chan<- transports.Event
	sendChan     chan protocolMessage

	supportsEvnt bool
	rwBuffer     bufio.ReadWriter

	// senderErr stores exit error from sender
	senderErr error
	// receiverShutdownChan is closed by CloseRead to request receiver to stop receiving and exit gracefully
	receiverShutdownChan chan struct{}
	// wait can be used to wait for all routines to complete
	wait sync.WaitGroup
}

func newConnection(ctx context.Context, socket connectionSocket, poolServer string, isClient bool, eventChan chan<- transports.Event) *connection {
	ret := &connection{
		socket:     socket,
		poolServer: poolServer,
		isClient:   isClient,
		eventChan:  eventChan,
		// TODO: Make configurable. Allow up to 100 pending messages.
		// This will cope with a max pending payload size of 100 for each connection by allowing 100 acks to be queued
		sendChan:             make(chan protocolMessage, 100),
		receiverShutdownChan: make(chan struct{}),
	}

	ret.ctx, ret.shutdownFunc = context.WithCancel(context.WithValue(ctx, transports.ContextConnection, ret))

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
	if err != nil && err != io.EOF {
		// Teardown if error occurs and it wasn't a graceful EOF (nil err means CloseRead called so also graceful)
		t.shutdownFunc()
	}

	// EOF is not an error to be returned
	if err == io.EOF {
		err = nil
	}

	// Wait for sender to complete due to Teardown or nil sent
	t.wait.Wait()

	// Cleanup
	t.socket.Close()

	// Ensure context resources are cleaned up
	t.shutdownFunc()

	// If we had no receiver error, did sender save one when it shutdown?
	if err == nil {
		return t.senderErr
	}
	return err
}

// serverNegotiation works out the protocol version supported by the remote
func (t *connection) serverNegotiation() error {
	message, err := t.readMsg()
	if err != nil {
		if err == errHardCloseRequested {
			return err
		}
		return fmt.Errorf("unexpected end of negotiation: %s", err)
	}

	_, ok := message.(*protocolHELO)
	if !ok {
		if messageImpl, ok := message.(eventsMessage); ok {
			// Backwards compatible path with older log-courier which do not perform a negotiation
			event := transports.NewEventsEvent(t.ctx, messageImpl.Nonce(), messageImpl.Events())
			if err := t.sendEvent(event); err != nil {
				return err
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
	if err != nil {
		if err == errHardCloseRequested {
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
		log.Debugf("[C %s < %s] Remote supports enhanced EVNT messages", t.poolServer, t.socket.RemoteAddr().String())
	}

	return nil
}

// senderRoutine wraps the sender into a routine and handles error communication
func (t *connection) senderRoutine() {
	defer func() {
		t.wait.Done()
	}()

	t.senderErr = t.sender()
}

// sender handles socket writes
func (t *connection) sender() error {
	for {
		var msg protocolMessage

		select {
		case <-t.ctx.Done():
			return errHardCloseRequested
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
func (t *connection) sendEvent(transportEvent transports.Event) error {
	select {
	case <-t.receiverShutdownChan:
		// Gracefully stop receiving data
		return io.EOF
	case <-t.ctx.Done():
		// Teardown - end now
		return errHardCloseRequested
	case t.eventChan <- transportEvent:
	}
	return nil
}

// receiver handles socket reads
// Returns nil error on shutdown, or an actual error
func (t *connection) receiver() (err error) {
	for {
		var message protocolMessage
		message, err = t.readMsg()
		if err != nil {
			break
		}

		var transportEvent transports.Event = nil
		// TODO: Can events be interfaces so we can just pass on the message itself if it implements the interface?
		if t.isClient {
			switch messageImpl := message.(type) {
			case *protocolACKN:
				transportEvent = transports.NewAckEvent(t.ctx, messageImpl.nonce, messageImpl.sequence)
				log.Debugf("[T %s < %s] Received acknowledgement for nonce %x with sequence %d", t.socket.LocalAddr().String(), t.socket.RemoteAddr().String(), *messageImpl.nonce, messageImpl.sequence)
			case *protocolPONG:
				transportEvent = transports.NewPongEvent(t.ctx)
				log.Debugf("[T %s < %s] Received pong", t.socket.LocalAddr().String(), t.socket.RemoteAddr().String())
			}
		} else {
			switch messageImpl := message.(type) {
			case *protocolPING:
				transportEvent = transports.NewPingEvent(t.ctx)
				log.Debugf("[R %s < %s] Received ping", t.socket.LocalAddr().String(), t.socket.RemoteAddr().String())
			case eventsMessage:
				transportEvent = transports.NewEventsEvent(t.ctx, messageImpl.Nonce(), messageImpl.Events())
				log.Debugf("[R %s < %s] Received payload with nonce %x and %d events", t.socket.LocalAddr().String(), t.socket.RemoteAddr().String(), *messageImpl.Nonce(), len(messageImpl.Events()))
			}
		}

		if transportEvent == nil {
			return fmt.Errorf("unknown protocol message %T", message)
		}

		if err := t.sendEvent(transportEvent); err != nil {
			break
		}
	}

	if err == io.EOF {
		// Send EOF signal so receiver or transport can handle it accordingly
		err = t.sendEvent(transports.NewEndEvent(t.ctx))
	}
	return
}

// readMsg reads a message from the connection
// Returns nil message if shutdown, with an optional error
func (t *connection) readMsg() (protocolMessage, error) {
	var header [8]byte

	if err := t.receiverRead(header[:]); err != nil {
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
func (t *connection) receiverRead(data []byte) error {
	var err error
	received := 0

ReceiverReadLoop:
	for {
		select {
		case <-t.ctx.Done():
			err = errHardCloseRequested
			break ReceiverReadLoop
		case <-t.receiverShutdownChan:
			// Stop receiving any more data so we can gracefully close
			err = io.EOF
			break ReceiverReadLoop
		default:
			// Timeout after socketIntervalSeconds, check for shutdown, and try again
			t.socket.SetReadDeadline(time.Now().Add(socketIntervalSeconds * time.Second))

			length, err := t.rwBuffer.Read(data[received:])
			received += length

			if received >= len(data) {
				// Success
				return nil
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
			return err
		}
	}

	return err
}

// Teardown ends the connection forcefully, usually due to a problem
// No waiting happens and connections are not gracefully closed
func (t *connection) Teardown() {
	t.shutdownFunc()
}

// CloseRead stops the receiving of data from this connection
// It is used by Receiver to request we stop receiving data, whilst allowing acks to be sent for what we did receive
// Once a nil is sent via SendMessage CloseWrite then happens and we then shutdown fully
// Calling this function twice will cause a panic due to invalid state
func (t *connection) CloseRead() {
	close(t.receiverShutdownChan)
}

// SendMessage queues a message to be sent on the connection
// Send a nil to begin closing the connection gracefully by performing a CloseWrite
// Once an EOF occurs on the read side (or a CloseRead occurs) shutdown will be complete
// If the connection is running too slow and the queue is full then ErrCongestion is returned
// If the connection has closed it will return ErrInvalidState
// If nil is sent twice - there will be a panic - so only request shutdown ONCE
func (t *connection) SendMessage(message protocolMessage) error {
	if message == nil {
		close(t.sendChan)
		return nil
	}

	select {
	case <-t.ctx.Done():
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
	if err := t.receiverRead(message); err != nil {
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
