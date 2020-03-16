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

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// connContext is a context key for connections
type connContext string

const (
	contextIsClient connContext = "client"
	contextEventPos connContext = "pos"
)

// errHardCloseRequested is used to signal hard close requested
var errHardCloseRequested = errors.New("Connection shutdown was requested")

type connection struct {
	// Constructor
	ctx          context.Context
	socket       connectionSocket
	poolServer   string
	eventChan    chan<- transports.Event
	sendChan     chan protocolMessage
	shutdownChan chan struct{}

	// TODO: Merge with context - become clientContext
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

func newConnection(ctx context.Context, socket connectionSocket, poolServer string, eventChan chan<- transports.Event, sendChan chan protocolMessage) *connection {
	return &connection{
		ctx:          ctx,
		socket:       socket,
		poolServer:   poolServer,
		eventChan:    eventChan,
		sendChan:     sendChan,
		shutdownChan: make(chan struct{}),
	}
}

// Run starts the connection and all its routines
func (t *connection) Run(startedCallback func()) error {
	// Only setup these channels if allowing data, without them, we never allow JDAT
	if !t.isClient() {
		// TODO: Make configurable, max we receive into memory unacknowledged before stop receiving
		t.partialAcks = make([]eventsMessage, 0, 10)
		// We allow this to block and coordinate accordingly, however we need to cache 1 for
		// older Log Courier clients in case they drop us an event payload during negotiation
		t.partialAckChan = make(chan protocolMessage, 1)
	}

	t.rwBuffer.Reader = bufio.NewReader(t.socket)
	t.rwBuffer.Writer = bufio.NewWriter(t.socket)

	t.receiverShutdownChan = make(chan struct{})
	t.senderShutdownChan = make(chan struct{})

	if err := t.socket.Setup(); err != nil {
		return err
	}

	if t.isClient() {
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
	if err != nil {
		// Request sender shutdown due to receiver error
		close(t.senderShutdownChan)
	}
	// Wait for sender to complete
	t.wait.Wait()

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
		return fmt.Errorf("Unexpected end of negotiation: %s", err)
	}

	_, ok := message.(*protocolHELO)
	if !ok {
		if messageImpl, ok := message.(eventsMessage); ok {
			// Backwards compatible path with older log-courier which do not perform a negotiation
			t.partialAckChan <- message
			event := transports.NewEventsEvent(t.ctx, messageImpl.Nonce(), messageImpl.Events())
			if t.sendEvent(event) {
				return errHardCloseRequested
			}
			// Now go async
			return nil
		}
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
		} else if err == errHardCloseRequested {
			return err
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
	// Or it exist due to a problem
	// Therefore, always force shutdown here of receiver if it wasn't already
	// This will unblock everything, including attempts to send data that won't now complete
	// as they all monitor the main shutdown channel
	t.Teardown()
}

// sender handles socket writes
func (t *connection) sender() error {
	var timeout *time.Timer
	var timeoutChan <-chan time.Time

	if !t.isClient() {
		// TODO: Configurable? It's very low impact on anything though... NB: Repeated below
		timeout = time.NewTimer(5 * time.Second)
		timeout.Stop()
	}

	ackChan := t.partialAckChan
	senderShutdownChan := t.senderShutdownChan

	for {
		var msg protocolMessage

		select {
		case <-senderShutdownChan:
			if len(t.partialAcks) == 0 {
				// No outstanding acknowledgements, so let's shutdown gracefully
				return t.socket.Close()
			}
			// Flag as closing, we will end as soon as last acknowledge sent
			log.Debugf("[%s < %s] Shutdown is waiting for acknowledgement for %d payloads", t.poolServer, t.socket.RemoteAddr().String(), len(t.partialAcks))
			senderShutdownChan = nil
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
						"Protocol violation - cannot send PING whilst in-flight events (count: %d, current nonce: %s) in progress",
						len(t.partialAcks),
						t.partialAcks[0].Nonce(),
					)
				}

				msg = &protocolPONG{}
				break
			}

			panic(fmt.Sprintf(
				"Invalid sender partialAckChan request; expected eventsMessage or *protocolPING and received %T",
				message,
			))
		case <-timeoutChan:
			// Partial ack
			log.Debugf("[%s < %s] Sending partial acknowledgement for payload %x sequence %d", t.poolServer, t.socket.RemoteAddr().String(), t.partialAcks[0].Nonce(), t.lastSequence)
			msg = &protocolACKN{nonce: t.partialAcks[0].Nonce(), sequence: t.lastSequence}
			timeout.Reset(5 * time.Second)
		case msg = <-t.sendChan:
			// Is this the end message? nil? No more to send?
			if msg == nil {
				// Close send side to generate EOF on remote
				return t.socket.CloseWrite()
			}
			if !t.isClient() {
				// ACKN Should ALWAYS be in order
				if ack, ok := msg.(*protocolACKN); ok {
					if len(t.partialAcks) == 0 {
						panic(fmt.Sprintf(
							"Out-of-order ACK received; there are no outstanding payloads yet received ACK for nonce %x (sequence %d)",
							ack.nonce,
							ack.sequence,
						))
					}
					if ack.nonce != t.partialAcks[0].Nonce() || ack.sequence <= t.lastSequence {
						panic(fmt.Sprintf(
							"Out-of-order ACK received; expected nonce %x (sequence > %d) and received ACK for nonce %x (sequence %d)",
							t.partialAcks[0].Nonce(),
							t.lastSequence,
							ack.nonce,
							ack.sequence,
						))
					}

					if ack.sequence >= uint32(len(t.partialAcks[0].Events())) {
						// Full ack, shift list
						for i := 1; i < len(t.partialAcks); i++ {
							t.partialAcks[i-1] = t.partialAcks[i]
						}
						t.partialAcks = t.partialAcks[:len(t.partialAcks)-1]

						// Reset last sequence
						t.lastSequence = 0

						// Still need timer?
						if len(t.partialAcks) == 0 {
							if senderShutdownChan == nil {
								// Finished! Call Close() to trigger final flush then exit
								// Safe as receiver is no longer running
								// This can timeout too so forward back any error...
								log.Debugf("[%s < %s] All payloads acknowledged, shutting down", t.poolServer, t.socket.RemoteAddr().String())
								return t.socket.Close()
							}
							if !timeout.Stop() {
								<-timeout.C
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

					// Reset timeout
					if timeoutChan != nil {
						if !timeout.Stop() {
							<-timeout.C
						}
						timeout.Reset(5 * time.Second)
					}
					log.Debugf("[%s < %s] Sending acknowledgement for payload %x sequence %d", t.poolServer, t.socket.RemoteAddr().String(), ack.nonce, ack.sequence)
				}
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
		if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
			// Fail the transport
			return err
		}
	}
	return nil
}

// sendEvent ships an event structure whilst also monitoring for
// any shutdown signal. Returns true if shutdown was signalled
// Does not monitor send thread errors here since we are blocking internally
// with a fully received message so let's get that handled first before
// triggering any teardown due to sender error
func (t *connection) sendEvent(transportEvent transports.Event) bool {
	select {
	case <-t.shutdownChan:
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

		var event transports.Event = nil
		if t.isClient() {
			switch messageImpl := message.(type) {
			case *protocolACKN:
				event = transports.NewAckEvent(t.ctx, messageImpl.nonce, messageImpl.sequence)
			case *protocolPONG:
				event = transports.NewPongEvent(t.ctx)
			}
		} else {
			switch messageImpl := message.(type) {
			case *protocolPING:
				// Request receiver to handle a ping response and don't deliver as handled internally in connection
				t.partialAckChan <- message
				continue
			case eventsMessage:
				// Start the sender partial acks - this blocks if too many outstanding which stops us receiving more
				t.partialAckChan <- message
				log.Debugf("[%s < %s] Received payload %x with %d events", t.poolServer, t.socket.RemoteAddr().String(), messageImpl.Nonce(), len(messageImpl.Events()))
				event = transports.NewEventsEvent(t.ctx, messageImpl.Nonce(), messageImpl.Events())
			}
		}

		if event == nil {
			return fmt.Errorf("Unknown protocol message %T", message)
		}

		if shutdown := t.sendEvent(event); shutdown {
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

// isClient returns true if this is a client side connection
func (t *connection) isClient() bool {
	contextValue := t.ctx.Value(contextIsClient)
	if contextValue == nil {
		return true
	}

	return contextValue.(bool)
}

// Acknowledge handles acknowledgement transmission once an event is complete
// It implements Acknowledger interface on the connection so any events we receive
// we can process acknowledge for
func (t *connection) Acknowledge(events []*event.Event) {
	position := events[0].Context().Value(contextEventPos).(*eventPosition)
	for _, event := range events[1:] {
		nextPosition := event.Context().Value(contextEventPos).(*eventPosition)
		if nextPosition.nonce != position.nonce {
			err := t.SendMessage(&protocolACKN{nonce: position.nonce, sequence: position.sequence})
			if err != nil {
				return
			}
		}
		position = nextPosition
	}
	t.SendMessage(&protocolACKN{nonce: position.nonce, sequence: position.sequence})
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

// SendChan returns the sendChan
func (t *connection) SendMessage(message protocolMessage) error {
	select {
	case <-t.shutdownChan:
		return errors.New("Invalid connection state")
	case t.sendChan <- message:
	}
	return nil
}

// SupportsEVNT returns true if this connection supports EVNT messages
func (t *connection) SupportsEVNT() bool {
	return t.supportsEvnt
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
