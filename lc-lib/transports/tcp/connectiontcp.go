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
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/transports"
)

type connectionTCP struct {
	socket   net.Conn
	server   bool // TODO: Merge with context - become clientContext
	rwBuffer bufio.ReadWriter

	context        interface{}
	poolServer     string
	eventChan      chan<- transports.Event
	connectionChan chan<- *socketMessage
	sendChan       chan protocolMessage

	controllerChan chan struct{}
	errChan        chan error
	wait           sync.WaitGroup
	err            error

	supportsEvnt   bool
	partialAcks    []eventsMessage
	partialAckChan chan protocolMessage
	lastSequence   uint32
}

// Run starts the connection
// Negiation for client connections runs synchronously
// Then receiver runs sychronously with async sender
// Sender is stopped via controllerChan by sending it an error on a single error buffered channel
// Sender will store an error and exit - this means if both receiver and sender fail, sender error wins as it will ignore the channel
func (t *connectionTCP) Run() error {
	t.controllerChan = make(chan struct{})
	t.errChan = make(chan error)

	// Only setup these channels if allowing data, without them, we never allow JDAT
	if t.server {
		// TODO: Make configurable, max we receive into memory unacknowledged before stop receiving
		t.partialAcks = make([]eventsMessage, 10)
		t.partialAckChan = make(chan protocolMessage)
	}

	t.rwBuffer.Reader = bufio.NewReader(t.socket)
	t.rwBuffer.Writer = bufio.NewWriter(t.socket)

	if !t.server {
		// Client mode
		t.wait.Add(1)
		t.negotiate()

		// Send a started signal to say we're ready
		if t.sendEvent(transports.NewStatusEvent(t.context, transports.Started)) {
			return nil
		}
	}

	go t.sender()
	t.receiver()

	// Exiting - wait for sender exit
	return <-t.errChan
}

// negotiate works out the protocol version supported by the remote
// negotiate is synchronous on its own
func (t *connectionTCP) negotiate() {
	if !t.writeMsg(&protocolHELO{}) {
		return
	}

	message := t.receiveMsg()
	if message == nil {
		return
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
		t.fail(fmt.Errorf("Unexpected %T reply to negotiation, expected protocolVERS", message))
		return
	}

	t.supportsEvnt = versMessage.SupportsEVNT()
	if t.supportsEvnt {
		log.Debug("[%s] Remote %s supports enhanced EVNT messages", t.poolServer, t.socket.RemoteAddr().String())
	}
}

// sender handles socket writes
// sender is async whilst receiver runs synchronously, and shuts down by receiving err and storing it
func (t *connectionTCP) sender() {
	var err error
	defer func() {
		t.errChan <- err
	}()

	var timeout *time.Timer
	var timeoutChan <-chan time.Time
	var ackChan <-chan protocolMessage

	if t.server {
		// TODO: Configurable? It's very low impact on anything though... NB: Repeated below
		timeout = time.NewTimer(5 * time.Second)
		timeout.Stop()
	}

	ackChan = t.partialAckChan

SenderLoop:
	for {
		var msg protocolMessage

		select {
		case <-t.controllerChan:
			// Shutdown
			break SenderLoop
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
					t.fail(fmt.Errorf(
						"Protocol violation - cannot send PING whilst in-flight events (count: %d, current nonce: %s) in progress (JDAT/EVNT/...)",
						len(t.partialAcks),
						t.partialAcks[0].Nonce(),
					))
					break SenderLoop
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
			// Should ALWAYS be in order
			if t.server {
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

		if !t.writeMsg(msg) {
			break
		}
	}
}

// writeMsg sends a message
func (t *connectionTCP) writeMsg(msg protocolMessage) bool {
	// Write deadline is managed by our net.Conn wrapper that TLS will call
	// into and keeps retrying writes until timeout or error
	err := msg.Write(t)
	if err == nil {
		err = t.rwBuffer.Flush()
	}
	if err != nil {
		if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
			// Fail the transport
			t.err = err
			return false
		}
	}
	return true
}

// receiver handles socket reads
// it runs synchronously
func (t *connectionTCP) receiver() {
	for {
		message := t.receiveMsg()
		if message == nil {
			break
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

			t.fail(fmt.Errorf("Unknown protocol message %T", message))
			break
		}

		var event transports.Event = nil
		switch messageImpl := message.(type) {
		case *protocolACKN:
			event = transports.NewAckEvent(t.context, messageImpl.nonce, messageImpl.sequence)
		case *protocolPONG:
			event = transports.NewPongEvent(t.context)
		}

		if event == nil {
			t.fail(fmt.Errorf("Unknown protocol message %T", message))
			break
		}

		if t.sendEvent(event) {
			break
		}
	}
}

// recieveMsg reads a single message
// Returns nil if shutdown needed
func (t *connectionTCP) receiveMsg() protocolMessage {
	message, err := t.readMsg()
	if err == nil {
		return message
	}

	// Pass the error back and abort
	t.fail(err)
	return nil
}

// readMsg reads a message from the connection
// Returns both nil message and nil error on shutdown
func (t *connectionTCP) readMsg() (protocolMessage, error) {
	var header [8]byte

	if shutdown, err := t.receiverRead(header[:]); shutdown || err != nil {
		return nil, err
	}

	// Grab length of message
	bodyLength := binary.BigEndian.Uint32(header[4:8])

	// Sanity
	if bodyLength > 10485760 {
		return nil, fmt.Errorf("Message body too large (%d > 10485760)", bodyLength)
	}

	switch {
	case bytes.Compare(header[0:4], []byte("????")) == 0: // UNKN
		return newProtocolUNKN(t, bodyLength)
	case bytes.Compare(header[0:4], []byte("HELO")) == 0:
		return newProtocolHELO(t, bodyLength)
	case bytes.Compare(header[0:4], []byte("VERS")) == 0:
		return newProtocolVERS(t, bodyLength)
	case bytes.Compare(header[0:4], []byte("PING")) == 0:
		return newProtocolPING(t, bodyLength)
	case bytes.Compare(header[0:4], []byte("PONG")) == 0:
		return newProtocolPONG(t, bodyLength)
	case bytes.Compare(header[0:4], []byte("ACKN")) == 0:
		return newProtocolACKN(t, bodyLength)
	case bytes.Compare(header[0:4], []byte("JDAT")) == 0:
		return newProtocolJDAT(t, bodyLength)
	case bytes.Compare(header[0:4], []byte("EVNT")) == 0:
		return newProtocolEVNT(t, bodyLength)
	}

	return nil, fmt.Errorf("Unexpected message code: %s", header[0:4])
}

// Read will receive the body of a message
// Returns both nil message and nil error on shutdown signal
func (t *connectionTCP) Read(length uint32) ([]byte, error) {
	var message []byte
	if length > 0 {
		// Allocate for full message
		message = make([]byte, length)
		if shutdown, err := t.receiverRead(message); shutdown || err != nil {
			return nil, err
		}
	} else {
		message = []byte("")
	}

	return message, nil
}

// receiverRead will repeatedly read from the socket until the given byte array
// is filled.
func (t *connectionTCP) receiverRead(data []byte) (bool, error) {
	var err error
	received := 0

ReceiverReadLoop:
	for {
		select {
		case err = <-t.errChan:
			// Shutdown
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

// fail sends a failure signal
func (t *connectionTCP) fail(err error) {
	select {
	case <-t.controllerChan:
	case t.connectionChan <- &socketMessage{conn: t, err: err}:
	}
}

// sendEvent ships an event structure whilst also monitoring for
// any shutdown signal. Returns true if shutdown was signalled
func (t *connectionTCP) sendEvent(transportEvent transports.Event) bool {
	select {
	case <-t.controllerChan:
		return true
	case t.eventChan <- transportEvent:
	}
	return false
}

// write to the socket
func (t *connectionTCP) Write(data []byte) (int, error) {
	return t.rwBuffer.Write(data)
}

// Acknowledge handles acknowledgement transmission once an event is complete
func (t *connectionTCP) Acknowledge(events []*event.Event) {
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

// teardown ends the connection
func (t *connectionTCP) Teardown() {
	close(t.controllerChan)
	t.wait.Wait()
	t.socket.Close()
	log.Notice("[%s] Disconnected from %s", t.poolServer, t.socket.RemoteAddr().String())
}

// SendChan returns the sendChan
func (t *connectionTCP) SendChan() chan protocolMessage {
	return t.sendChan
}

// Server returns true if this is a server connection that receives data
func (t *connectionTCP) Server() bool {
	return t.server
}

// SupportsEVNT returns true if this connection supports EVNT messages
func (t *connectionTCP) SupportsEVNT() bool {
	return t.supportsEvnt
}
