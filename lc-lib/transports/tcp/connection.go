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
	// window holds received events until the receiver takes them, and is nil for a client connection
	window     *transports.ReceiveWindow
	windowSize int

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

func newConnection(ctx context.Context, socket connectionSocket, protocolFactory ProtocolFactory, isClient bool, eventChan chan<- transports.Event, windowSize int, windowFill float64) *connection {
	ret := &connection{
		socket:     socket,
		isClient:   isClient,
		eventChan:  eventChan,
		windowSize: windowSize,
		// TODO: Make configurable. Allow up to 100 pending messages.
		// This will cope with a max pending payload size of 100 for each connection by allowing 100 acks to be queued
		sendChan:             make(chan ProtocolMessage, 100),
		receiverShutdownChan: make(chan struct{}),
	}

	ret.protocol = protocolFactory.NewProtocol(ret)

	ret.ctx, ret.shutdownFunc = context.WithCancel(context.WithValue(ctx, transports.ContextConnection, ret))
	if !isClient {
		ret.window = transports.NewReceiveWindow(ret.ctx, windowSize, windowFill)
	}

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

	t.wait.Add(1)
	go t.senderRoutine()
	// If handshake isn't implemented by a client in server mode, we may have an events message already, which is
	// handled after the startedCallback and once the sender runs to send its acknowledgements
	err = t.receiver(event)
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

// handleReceived passes a received event to the receiver, reading an events message into the receive window
func (t *connection) handleReceived(transportEvent transports.Event) error {
	if message, ok := transportEvent.(StreamMessage); ok {
		return t.receiveEvents(message)
	}
	return t.sendEvent(transportEvent)
}

// receiveEvents reads each event of a message into the receive window, asking the receiver to take them once they
// fill enough of the window or the message ends, and waiting only while the window is too full to hold the next
func (t *connection) receiveEvents(message StreamMessage) error {
	nonce := message.Nonce()
	for {
		raw, decoded, size, skipped, err := message.Next(t.reserveWindow)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if skipped {
			t.window.Skip(nonce)
			continue
		}
		// An event whose size was not known in advance is reserved once read
		if err := t.reserveWindow(size); err != nil {
			return err
		}
		var request bool
		if raw != nil {
			request = t.window.AddRaw(nonce, raw)
		} else {
			request = t.window.Add(nonce, decoded, size)
		}
		if request {
			if err := t.requestTake(); err != nil {
				return err
			}
		}
	}
	if t.window.End(nonce) {
		return t.requestTake()
	}
	return nil
}

// reserveWindow waits until the receive window can hold an event of the given size, first asking the receiver to take
// what it holds if it cannot
func (t *connection) reserveWindow(size int) error {
	if t.window.Fits(size) {
		return nil
	}
	if t.window.Request() {
		if err := t.requestTake(); err != nil {
			return err
		}
	}
	return t.awaitWindow(func() bool { return t.window.Fits(size) })
}

// requestTake asks the receiver to take the events held in the receive window
func (t *connection) requestTake() error {
	return t.sendEvent(transports.NewAdmissionEvent(t.ctx, t.window))
}

// awaitWindow waits until the receiver has taken enough from the receive window for ready to return true
func (t *connection) awaitWindow(ready func() bool) error {
	for !ready() {
		select {
		case <-t.ctx.Done():
			return ErrHardCloseRequested
		case <-t.window.Taken():
		}
	}
	return nil
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

// receiver handles socket reads, starting with any event already read during negotiation
// Returns nil error on shutdown, or an actual error
func (t *connection) receiver(transportEvent transports.Event) (err error) {
	for {
		if transportEvent == nil {
			transportEvent, err = t.protocol.Read()
			if err != nil {
				break
			}
		}

		if err = t.handleReceived(transportEvent); err != nil {
			break
		}
		transportEvent = nil
	}

	// Some of the protocol readers might return unexpected - treat it always as EOF
	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}

	// The receiver must take the events already read before it learns the connection ended
	if err == io.EOF && t.window != nil {
		if windowErr := t.awaitWindow(t.window.Empty); windowErr != nil {
			err = windowErr
		}
	}

	// Send EOF signal so receiver or transport can handle it accordingly, but keep existing error
	t.sendEvent(transports.NewEndEvent(t.ctx))
	return
}

// Read will receive data from the connection and implements io.Reader
// Used by protocol structures to fetch message data
// It should be noted that when the protocol is blocking, Read here will never return an
// incomplete read, and will either return a full read or an error. As such, it could run
// forever until the connection is aborted or an error occurs.
// For non-blocking protocols (those where there is no well-defined boundary of a message),
// then ErrIOWouldBlock is returned after the socketIntervalSeconds timeout even if the
// data is incomplete, and the int return value will be the number of bytes read so far.
func (t *connection) Read(data []byte) (int, error) {
	// A read the buffer can satisfy in full needs no deadline or shutdown checks, which run whenever it must be refilled
	if t.rwBuffer.Reader.Buffered() >= len(data) {
		return io.ReadFull(t.rwBuffer.Reader, data)
	}

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
	// Buffered data needs no deadline or shutdown checks, which run whenever the buffer must be refilled
	if t.rwBuffer.Reader.Buffered() > 0 {
		return t.rwBuffer.Reader.ReadByte()
	}
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

// ReceiveWindowSize returns the most event data held for the receiver at once
func (t *connection) ReceiveWindowSize() int {
	return t.windowSize
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
