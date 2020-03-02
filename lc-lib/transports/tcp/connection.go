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
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// errHardCloseRequested is used to signal hard close requested
var errHardCloseRequested = errors.New("Hard close requested")

type connection struct {
	socket connectionSocket

	context    interface{}
	poolServer string
	eventChan  chan<- transports.Event
	sendChan   chan protocolMessage

	// TODO: Merge with context - become clientContext
	server       bool
	supportsEvnt bool
	rwBuffer     bufio.ReadWriter

	// shutdownOrResetChan requests sender to hard close, optionally due to an error
	shutdownOrResetChan chan error
	// senderShutdownChan requests sender to begin to gracefully shutdown
	senderShutdownChan chan struct{}
	// senderFailedChan receives error from sender so receiver can hard close
	senderFailedChan chan error
	// wait can be used to wait for all routines to complete
	wait sync.WaitGroup
	// finalErr contains the final error for this connection after TearDown completed
	finalErr error

	// partialAcks is a list of events messages we are waiting for acknowledgement inside Log Courier for
	// It allows us to keep alive with partial acknowledgements
	partialAcks []eventsMessage
	// partialAckChan blocks receiver as it tells sender about a new events message it needs to keep alive
	// Must block, because if receiver ends it sends to sender to close after all acks sent, so ordering is key
	partialAckChan chan protocolMessage
	// lastSequence is the current keep alive partial ack for the first item in partialAcks that we use to keep alive
	lastSequence uint32
}

func newConnection(socket connectionSocket, context interface{}, poolServer string, eventChan chan<- transports.Event, sendChan chan protocolMessage) *connection {
	return &connection{
		socket:              socket,
		context:             context,
		poolServer:          poolServer,
		eventChan:           eventChan,
		sendChan:            sendChan,
		shutdownOrResetChan: make(chan error),
	}
}

// setServer enabled server mode for this connection, allowing it to receive events
func (t *connection) setServer(value bool) {
	t.server = value
}

// setShutdownOrResetChan sets a channel to use for reset/shutdown (send error to reset, close to shutdown)
func (t *connection) setShutdownOrResetChan(shutdownOrResetChan chan error) {
	t.shutdownOrResetChan = shutdownOrResetChan
}

// Run starts the connection and all its routines
func (t *connection) Run() error {
	// Only setup these channels if allowing data, without them, we never allow JDAT
	if t.server {
		// TODO: Make configurable, max we receive into memory unacknowledged before stop receiving
		t.partialAcks = make([]eventsMessage, 0, 10)
		t.partialAckChan = make(chan protocolMessage)
	}

	t.rwBuffer.Reader = bufio.NewReader(t.socket)
	t.rwBuffer.Writer = bufio.NewWriter(t.socket)

	// Shutdown chan can block - it is only ever closed
	t.senderShutdownChan = make(chan struct{})
	// Failure chan must be able to store the sender's single closing error, for receiver to (optionally) collect later
	t.senderFailedChan = make(chan error, 1)

	if err := t.socket.Setup(); err != nil {
		return err
	}

	if t.server {
		if err := t.serverNegotiation(); err != nil {
			return err
		}
	} else {
		if err := t.clientNegotiation(); err != nil {
			return err
		}
	}

	// Send a started signal to say we're ready
	// TODO: Move into transporttcp ?
	if shutdown, err := t.sendEvent(transports.NewStatusEvent(t.context, transports.Started)); shutdown || err != nil {
		return err
	}

	t.wait.Add(1)
	go t.senderRoutine()
	err := t.receiver()
	if err != nil {
		// Pass to sender to trigger hard close due to failure
		close(t.senderShutdownChan)
	}
	if t.server {
		// Close partialAckChan to allow sender to shutdown if it is monitoring it and waiting for receiver
		close(t.partialAckChan)
	}
	t.wait.Wait()

	if err == nil {
		select {
		case err = <-t.senderFailedChan:
		default:
		}
	}

	return err
}

// senderRoutine wraps the sender into a routine and handles error communication
func (t *connection) senderRoutine() {
	defer func() {
		t.wait.Done()
	}()

	var err error
	if err = t.sender(); err != nil {
		// Forward error to receiver if the sender failed so it can hard close
		t.senderFailedChan <- err
	}

	if t.server {
		// If sender was requested to shutdown by receiver, then stop now
		// Otherwise, wait in case receiver tries to request further sending through partial Ack, and if it does, force it to close, but only if we did not already
		select {
		case <-t.senderShutdownChan:
		case <-t.partialAckChan:
			if err == nil {
				err = errHardCloseRequested
				t.senderFailedChan <- errHardCloseRequested
			}
		}
	}
}

// serverNegotiation works out the protocol version supported by the remote
func (t *connection) serverNegotiation() error {
	message, err := t.readMsg()
	if message == nil {
		if err == nil {
			err = io.EOF
		}
		return fmt.Errorf("Unexpected end of negotiation: %s", err)
	}

	_, ok := message.(*protocolHELO)
	if !ok {
		return fmt.Errorf("Unexpected %T during negotiation, expected protocolHELO", message)
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
		}
		return fmt.Errorf("Unexpected end of negotiation: %s", err)
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
		return fmt.Errorf("Unexpected %T reply to negotiation, expected protocolVERS", message)
	}

	t.supportsEvnt = versMessage.SupportsEVNT()
	if t.supportsEvnt {
		log.Debug("[%s] Remote %s supports enhanced EVNT messages", t.poolServer, t.socket.RemoteAddr().String())
	}

	return nil
}

// sender handles socket writes
func (t *connection) sender() error {
	var timeout *time.Timer
	var timeoutChan <-chan time.Time
	var ackChan <-chan protocolMessage
	var shutdownChan <-chan struct{}

	if t.server {
		// TODO: Configurable? It's very low impact on anything though... NB: Repeated below
		timeout = time.NewTimer(5 * time.Second)
		timeout.Stop()
	}

	ackChan = t.partialAckChan
	shutdownChan = t.senderShutdownChan

	for {
		var msg protocolMessage

		select {
		case <-shutdownChan:
			if len(t.partialAcks) == 0 {
				// No outstanding acknowldgements, so let's shutdown gracefully
				return t.socket.Close()
			}
			// Flag as closing, we will end as soon as last acknowledge sent
			shutdownChan = nil
			continue
		case message := <-ackChan:
			if partialAck, ok := message.(eventsMessage); ok {
				// Stop receiving if we now have 10
				t.partialAcks = append(t.partialAcks, partialAck)
				if len(t.partialAcks) >= 10 {
					ackChan = nil
				}

				// If timer not started, start it
				if timeoutChan == nil {
					timeoutChan = timeout.C
					timeout.Reset(5 * time.Second)
				}
				continue
			}

			if _, ok := message.(*protocolPING); ok {
				if len(t.partialAcks) > 0 {
					// Invalid - protocol violation - cannot send PING whilst Events in progress
					return fmt.Errorf(
						"Protocol violation - cannot send PING whilst in-flight events (count: %d, current nonce: %s) in progress (JDAT/EVNT/...)",
						len(t.partialAcks),
						t.partialAcks[0].Nonce(),
					)
				}

				msg = &protocolPONG{}
				break
			}

			panic(fmt.Sprintf(
				"Invalid sender ackChan request; expected eventsMessage or *protocolPING and received %T",
				message,
			))
		case <-timeoutChan:
			// Partial ack
			msg = &protocolACKN{nonce: t.partialAcks[0].Nonce(), sequence: t.lastSequence}
		case msg = <-t.sendChan:
			// Is this the end message? nil? No more to send?
			if msg == nil {
				// Close send side to generate EOF on remote
				return t.socket.CloseWrite()
			}
			if t.server {
				// ACKN Should ALWAYS be in order
				if ack, ok := msg.(*protocolACKN); ok {
					if ack.nonce != t.partialAcks[0].Nonce() || ack.sequence <= t.lastSequence {
						panic(fmt.Sprintf(
							"Out-of-order ACK received; expected nonce %s (sequence > %d) and received ACK for nonce %s (sequence %d)",
							t.partialAcks[0].Nonce(),
							t.lastSequence,
							ack.nonce,
							ack.sequence,
						))
					}

					if ack.sequence >= uint32(len(t.partialAcks[0].Events())) {
						// Full ack, shift list
						for i := 1; i < len(t.partialAcks)-1; i++ {
							t.partialAcks[i-1] = t.partialAcks[i]
						}
						t.partialAcks[len(t.partialAcks)-1] = nil

						// Reset last sequence
						t.lastSequence = 0

						// Still need timer?
						if len(t.partialAcks) == 0 {
							if shutdownChan == nil {
								// Finished! Call Close() to trigger final flush then exit
								// Safe as receiver is no longer running
								// This can timeout too so forward back any error...
								return t.socket.Close()
							}
							timeoutChan = nil
						}

						// Restore ackChan
						if ackChan == nil {
							ackChan = t.partialAckChan
						}
					} else {
						// Update last sequence
						t.lastSequence = ack.sequence
					}
				}
			}
		}

		// Reset timeout
		if timeoutChan != nil {
			timeout.Reset(10 * time.Second)
		}

		if err := t.writeMsg(msg); err != nil {
			return err
		}
	}
}

// receiver handles socket reads
// Returns nil error on shutdown, or an actual error
func (t *connection) receiver() error {
	for {
		message, err := t.readMsg()
		if message == nil {
			return err
		}

		if t.server {
			switch message.(type) {
			case *protocolPING:
				// Request receiver to handle a ping response and don't deliver as handled internally in connection
				t.partialAckChan <- message
				continue
			case eventsMessage:
				// Start the sender partial acks - this blocks if too many outstanding which stops us receiving more
				t.partialAckChan <- message
				// TODO - direct events to spooler
				continue
			}

			return fmt.Errorf("Unknown protocol message %T", message)
		}

		var event transports.Event = nil
		switch messageImpl := message.(type) {
		case *protocolACKN:
			event = transports.NewAckEvent(t.context, messageImpl.nonce, messageImpl.sequence)
		case *protocolPONG:
			event = transports.NewPongEvent(t.context)
		}

		if event == nil {
			return fmt.Errorf("Unknown protocol message %T", message)
		}

		if shutdown, err := t.sendEvent(event); shutdown || err != nil {
			return err
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
				// Flag sender that we have no more data being received and it should begin to shutdown
				close(t.senderShutdownChan)
				return nil, nil
			}
		}
		return nil, err
	}

	// Grab length of message
	bodyLength := binary.BigEndian.Uint32(header[4:8])

	// Sanity
	if bodyLength > 10485760 {
		return nil, fmt.Errorf("Message body too large (%s: %d > 10485760)", header[0:4], bodyLength)
	}

	var newFunc func(*connection, uint32) (protocolMessage, error)
	switch {
	case bytes.Compare(header[0:4], []byte("????")) == 0: // UNKN
		newFunc = newProtocolUNKN
	case bytes.Compare(header[0:4], []byte("HELO")) == 0:
		newFunc = newProtocolHELO
	case bytes.Compare(header[0:4], []byte("VERS")) == 0:
		newFunc = newProtocolVERS
	case bytes.Compare(header[0:4], []byte("PING")) == 0:
		newFunc = newProtocolPING
	case bytes.Compare(header[0:4], []byte("PONG")) == 0:
		newFunc = newProtocolPONG
	case bytes.Compare(header[0:4], []byte("ACKN")) == 0:
		newFunc = newProtocolACKN
	case bytes.Compare(header[0:4], []byte("JDAT")) == 0:
		newFunc = newProtocolJDAT
	case bytes.Compare(header[0:4], []byte("EVNT")) == 0:
		newFunc = newProtocolEVNT
	default:
		return nil, fmt.Errorf("Unexpected message code: %s", header[0:4])
	}

	return newFunc(t, bodyLength)
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
		if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
			// Fail the transport
			return err
		}
	}
	return nil
}

// receiverRead will repeatedly read from the socket until the given byte array
// is filled, or sender signals an error
func (t *connection) receiverRead(data []byte) (bool, error) {
	var err error
	received := 0

ReceiverReadLoop:
	for {
		select {
		case err = <-t.senderFailedChan:
			// Shutdown via sender due to error
			break ReceiverReadLoop
		case <-t.shutdownOrResetChan:
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

// sendEvent ships an event structure whilst also monitoring for
// any shutdown signal. Returns true if shutdown was signalled
// Does not monitor send thread errors here since we are blocking internally
// with a fully received message so let's get that handled first before
// triggering any teardown due to sender error
func (t *connection) sendEvent(transportEvent transports.Event) (bool, error) {
	select {
	case err := <-t.shutdownOrResetChan:
		return true, err
	case t.eventChan <- transportEvent:
	}
	return false, nil
}

// Read will receive the body of a message
// Returns both nil message and nil error on shutdown signal
// Used by protocol structures to fetch extra message data
// Always returns full requested message size, never partial
func (t *connection) Read(message []byte) (int, error) {
	if shutdown, err := t.receiverRead(message); shutdown || err != nil {
		return 0, err
	}

	return len(message), nil
}

// write data to the outgoing buffer
// Used by protocol structures to send a message
func (t *connection) Write(data []byte) (int, error) {
	return t.rwBuffer.Write(data)
}

// Acknowledge handles acknowledgement transmission once an event is complete
func (t *connection) Acknowledge(events []*event.Event) {
	position := events[0].Context().(*evntPosition)
	for _, event := range events[1:] {
		nextPosition := event.Context().(*evntPosition)
		if nextPosition.nonce != position.nonce {
			t.sendChan <- &protocolACKN{nonce: position.nonce, sequence: position.sequence}
			position = nextPosition
		}
	}
	t.sendChan <- &protocolACKN{nonce: position.nonce, sequence: position.sequence}
}

// Teardown ends the connection using the reset/shutdown channel, if you called setShutdownOrResetChan you can do this yourself
func (t *connection) Teardown() {
	close(t.shutdownOrResetChan)
}

// SendChan returns the sendChan
func (t *connection) SendChan() chan protocolMessage {
	return t.sendChan
}

// Server returns true if this is a server connection that receives data
func (t *connection) Server() bool {
	return t.server
}

// SupportsEVNT returns true if this connection supports EVNT messages
func (t *connection) SupportsEVNT() bool {
	return t.supportsEvnt
}
