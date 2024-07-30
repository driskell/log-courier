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
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/transports"
)

// errHardCloseRequested is used to signal hard close requested
var (
	// errHardCloseRequested occurs when the connection was closed forcefully and a hard close is requested
	ErrHardCloseRequested = errors.New("connection shutdown was requested")
)

type connection struct {
	// ctx uniquely identifies the connection - a new one is allocated by newConnection
	ctx          context.Context
	shutdownFunc context.CancelFunc
	socket       connectionSocket
	protocol     Protocol
	isClient     bool
	eventChan    chan<- transports.Event
	sendChan     chan ProtocolMessage
	rwBuffer     bufio.ReadWriter

	// senderErr stores exit error from sender so if receiver succeeds but sender not we can return it
	senderErr error
	// senderShutdown stores whether sender is active or not, so SendMessage can return ErrInvalidState
	senderShutdown bool
	// receiverShutdownChan is closed by CloseRead to request receiver to stop receiving and exit gracefully
	receiverShutdownChan chan struct{}
	// wait can be used to wait for all routines to complete
	wait sync.WaitGroup
	// receiverShutdownOnce ensures CloseRead's receiverShutdownChan channel close happens only once
	receiverShutdownOnce sync.Once
	// sendShutdownLock provides access to sendShutdown and ensures only a single close of sendChan
	sendShutdownLock sync.RWMutex
}

func newConnection(ctx context.Context, socket connectionSocket, protocolFactory ProtocolFactory, isClient bool, eventChan chan<- transports.Event) *connection {
	ret := &connection{
		socket:    socket,
		isClient:  isClient,
		eventChan: eventChan,
		// TODO: Make configurable. Allow up to 100 pending messages.
		// This will cope with a max pending payload size of 100 for each connection by allowing 100 acks to be queued
		sendChan:             make(chan ProtocolMessage, 100),
		receiverShutdownChan: make(chan struct{}),
	}

	ret.protocol = protocolFactory.NewProtocol(ret)

	ret.ctx, ret.shutdownFunc = context.WithCancel(context.WithValue(ctx, transports.ContextConnection, ret))

	return ret
}

// Run starts the connection and all its routines
func (t *connection) run(startedCallback func()) error {
	defer func() {
		// Cleanup
		t.socket.Close()

		// Ensure context resources are cleaned up
		t.shutdownFunc()
	}()

	t.rwBuffer.Reader = bufio.NewReader(t.socket)
	t.rwBuffer.Writer = bufio.NewWriter(t.socket)

	if err := t.socket.Setup(t.ctx); err != nil {
		return err
	}

	event, err := t.protocol.Negotiation()
	if err != nil {
		return err
	}

	if startedCallback != nil {
		startedCallback()
	}

	// If handshake isn't implemented by a client in server mode, we may have an events message already
	// Ensure it is sent AFTER the startedCallback and not before, which might mean something isn't ready
	if event != nil {
		if err := t.sendEvent(event); err != nil {
			return err
		}
	}

	t.wait.Add(1)
	go t.senderRoutine()
	err = t.receiver()
	if err != nil && err != io.EOF {
		// Teardown if error occurs and it wasn't a graceful EOF (nil err means CloseRead called so also graceful)
		t.shutdownFunc()
	}

	// Wait for sender to complete due to Teardown or nil sent
	t.wait.Wait()

	// If we had no receiver error, did sender save one when it shutdown?
	if err == nil {
		err = t.senderErr
	}

	// EOF is not an error to be returned
	// We check it here, after checking sender error, because with TLS an EOF can come from a send too
	if err == io.EOF {
		err = nil
	}

	return err
}

// senderRoutine wraps the sender into a routine and handles error communication
func (t *connection) senderRoutine() {
	defer func() {
		t.wait.Done()
	}()

	t.senderErr = t.sender()
	if t.senderErr != nil {
		// Sender issue, close connections
		t.shutdownFunc()
	}
}

// sender handles socket writes
func (t *connection) sender() error {
	for {
		var msg ProtocolMessage

		select {
		case <-t.ctx.Done():
			return ErrHardCloseRequested
		case msg = <-t.sendChan:
			// Is this the end message? nil? No more to send?
			if msg == nil {
				// Close send side to generate EOF on remote
				return t.socket.CloseWrite()
			}
		}

		if err := msg.Write(t); err != nil {
			return err
		}
	}
}

// sendEvent ships an event structure whilst also monitoring for
// any shutdown signal. Returns true if shutdown was signalled
func (t *connection) sendEvent(transportEvent transports.Event) error {
	select {
	case <-t.ctx.Done():
		// Teardown - end now
		return ErrHardCloseRequested
	case t.eventChan <- transportEvent:
	}
	return nil
}

// receiver handles socket reads
// Returns nil error on shutdown, or an actual error
func (t *connection) receiver() (err error) {
	for {
		var transportEvent transports.Event
		transportEvent, err = t.protocol.Read()
		if err != nil {
			break
		}

		if err := t.sendEvent(transportEvent); err != nil {
			break
		}
	}

	// Some of the protocol readers might return unexpected - treat it always as EOF
	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}

	// Send EOF signal so receiver or transport can handle it accordingly, but keep existing error
	t.sendEvent(transports.NewEndEvent(t.ctx))
	return
}

// Read will receive data from the connection and implements io.Reader
// Used by protocol structures to fetch message data
// It should be noted that Read here will never return an incomplete read, and as such
// the int return will always be the length of the message, unless an error occurs, in
// which case it will be 0.
func (t *connection) Read(data []byte) (int, error) {
	var err error
	received := 0

ReceiverReadLoop:
	for {
		select {
		case <-t.ctx.Done():
			err = ErrHardCloseRequested
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
				return received, nil
			}

			if err == nil {
				if t.protocol.NonBlocking() {
					return received, ErrIOWouldBlock
				}
				// Keep trying
				continue
			}

			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				if t.protocol.NonBlocking() {
					return received, ErrIOWouldBlock
				}
				// Keep trying
				continue
			}

			// Pass an error back
			return received, err
		}
	}

	return received, err
}

// Teardown ends the connection forcefully, usually due to a problem
// No waiting happens and connections are not gracefully closed
func (t *connection) Teardown() {
	t.shutdownFunc()
}

// CloseRead stops the receiving of data from this connection
// It is used by Receiver to request we stop receiving data, whilst allowing acks to be sent for what we did receive
// Once a nil is sent via SendMessage CloseWrite then happens and we then shutdown fully
func (t *connection) CloseRead() {
	t.receiverShutdownOnce.Do(func() {
		close(t.receiverShutdownChan)
	})
}

// SendMessage queues a message to be sent on the connection
// Send a nil to begin closing the connection gracefully by performing a CloseWrite
// Once an EOF occurs on the read side (or a CloseRead occurs) shutdown will be complete
// If the connection is running too slow and the queue is full then ErrCongestion is returned
// If the connection closed or CloseWrite has already happened it will return ErrInvalidState
func (t *connection) SendMessage(message ProtocolMessage) error {
	if message == nil {
		t.sendShutdownLock.Lock()
		defer t.sendShutdownLock.Unlock()
		if t.senderShutdown {
			return nil
		}
		close(t.sendChan)
		t.senderShutdown = true
		return nil
	}

	t.sendShutdownLock.RLock()
	defer t.sendShutdownLock.RUnlock()
	if t.senderShutdown {
		return transports.ErrInvalidState
	}

	select {
	case <-t.ctx.Done():
		return transports.ErrInvalidState
	case t.sendChan <- message:
	default:
		return transports.ErrCongestion
	}
	return nil
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

// Flush any remaining data in the rwBuffer to the connection
func (t *connection) Flush() error {
	return t.rwBuffer.Flush()
}

// LocalAddr returns the local endpoint
func (t *connection) LocalAddr() net.Addr {
	return t.socket.LocalAddr()
}

// RemoteAddr returns the remote endpoint
func (t *connection) RemoteAddr() net.Addr {
	return t.socket.RemoteAddr()
}

// Context returns the connection context
func (t *connection) Context() context.Context {
	return t.ctx
}
