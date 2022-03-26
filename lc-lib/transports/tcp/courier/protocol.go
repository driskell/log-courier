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

package courier

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/transports"
	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

type protocol struct {
	conn     tcp.Connection
	isClient bool

	supportsEvnt bool
}

// Negotiation starts the connection
func (p *protocol) Negotiation() (transports.Event, error) {
	if p.isClient {
		return nil, p.clientNegotiation()
	}
	return p.serverNegotiation()
}

// serverNegotiation works out the protocol version supported by the remote
func (p *protocol) serverNegotiation() (transports.Event, error) {
	message, err := p.readMsg()
	if err != nil {
		if err == tcp.ErrHardCloseRequested {
			return nil, err
		}
		return nil, fmt.Errorf("unexpected end of negotiation: %s", err)
	}

	heloMessage, ok := message.(*protocolHELO)
	if !ok {
		if messageImpl, ok := message.(transports.EventsEvent); ok {
			// Backwards compatible path with older log-courier which do not perform a negotiation
			log.Infof("[R %s < %s] Remote does not support protocol handshake", p.conn.LocalAddr().String(), p.conn.RemoteAddr().String())
			return messageImpl, nil
		}
		return nil, fmt.Errorf("unexpected %T during negotiation, expected protocolHELO", message)
	}

	log.Infof("[R %s < %s] Remote identified as %s", p.conn.LocalAddr().String(), p.conn.RemoteAddr().String(), heloMessage.Client())

	log.Debugf("[R %s > %s] Sending protocol version", p.conn.LocalAddr().String(), p.conn.RemoteAddr().String())
	if err := createProtocolVERS().Write(p.conn); err != nil {
		return nil, err
	}

	return nil, nil
}

// clientNegotiation works out the protocol version supported by the remote
func (p *protocol) clientNegotiation() error {
	log.Debugf("[T %s > %s] Sending hello", p.conn.LocalAddr().String(), p.conn.RemoteAddr().String())
	if err := createProtocolHELO().Write(p.conn); err != nil {
		return err
	}

	message, err := p.readMsg()
	if err != nil {
		if err == tcp.ErrHardCloseRequested {
			return err
		}
		return fmt.Errorf("unexpected end of negotiation: %s", err)
	}

	versMessage, ok := message.(*protocolVERS)
	if !ok {
		if _, isUnkn := message.(*protocolUNKN); isUnkn {
			p.supportsEvnt = false
			log.Infof("[R %s < %s] Remote does not support protocol handshake", p.conn.LocalAddr().String(), p.conn.RemoteAddr().String())
			return nil
		}
		return fmt.Errorf("unexpected %T reply to negotiation, expected protocolVERS", message)
	}

	log.Infof("[T %s < %s] Remote identified as %s", p.conn.LocalAddr().String(), p.conn.RemoteAddr().String(), versMessage.Client())

	p.supportsEvnt = versMessage.SupportsEVNT()
	if p.supportsEvnt {
		log.Debugf("[T %s < %s] Remote supports enhanced EVNT messages", p.conn.LocalAddr().String(), p.conn.RemoteAddr().String())
	}

	return nil
}

// SendEvents sends an event message with given nonce to the transport - only valid after Started transport event received
func (p *protocol) SendEvents(nonce string, events []*event.Event) error {
	var eventsAsBytes = make([][]byte, len(events))
	for idx, item := range events {
		eventsAsBytes[idx] = item.Bytes()
	}

	if p.supportsEvnt {
		msg := &protocolEVNT{nonce: &nonce, events: eventsAsBytes}
		log.Debugf("[T %s > %s] Sending %s payload with nonce %x and %d events", p.conn.LocalAddr().String(), p.conn.RemoteAddr().String(), msg.Type(), *msg.Nonce(), len(msg.Events()))
		return p.conn.SendMessage(msg)
	}

	msg := &protocolJDAT{nonce: &nonce, events: eventsAsBytes}
	log.Debugf("[T %s > %s] Sending %s payload with nonce %x and %d events", p.conn.LocalAddr().String(), p.conn.RemoteAddr().String(), msg.Type(), *msg.Nonce(), len(msg.Events()))
	return p.conn.SendMessage(msg)
}

// Acknowledge sends the correct connection an acknowledgement
func (p *protocol) Acknowledge(nonce *string, sequence uint32) error {
	log.Debugf("[R %s > %s] Sending acknowledgement for nonce %x with sequence %d", p.conn.LocalAddr().String(), p.conn.RemoteAddr().String(), *nonce, sequence)
	return p.conn.SendMessage(&protocolACKN{nonce: nonce, sequence: sequence})
}

// Ping sends a ping message
func (p *protocol) Ping() error {
	log.Debugf("[T %s > %s] Sending ping", p.conn.LocalAddr().String(), p.conn.RemoteAddr().String())
	return p.conn.SendMessage(&protocolPING{})
}

// Pong sends the correct connection a pong response
func (p *protocol) Pong() error {
	log.Debugf("[R %s > %s] Sending pong", p.conn.LocalAddr().String(), p.conn.RemoteAddr().String())
	return p.conn.SendMessage(&protocolPONG{})
}

// readMsg reads a single message from the connection
func (p *protocol) readMsg() (tcp.ProtocolMessage, error) {
	var header [8]byte

	if _, err := p.conn.Read(header[:]); err != nil {
		return nil, err
	}

	// Grab length of message
	bodyLength := binary.BigEndian.Uint32(header[4:8])

	var newFunc func(tcp.Connection, uint32) (tcp.ProtocolMessage, error)
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
		if p.isClient {
			return nil, errors.New("protocol error: Unexpected JDAT message received on client connection")
		}
		newFunc = newProtocolJDAT
	case bytes.Equal(header[0:4], []byte("EVNT")):
		if p.isClient {
			return nil, errors.New("protocol error: Unexpected JDAT message received on client connection")
		}
		newFunc = newProtocolEVNT
	default:
		return nil, fmt.Errorf("unexpected message code: %s", header[0:4])
	}

	return newFunc(p.conn, bodyLength)
}

// Read reads a message from the connection and calculates an event
// Returns nil event if shutdown, with an optional error
func (p *protocol) Read() (transports.Event, error) {
	message, err := p.readMsg()
	if err != nil {
		return nil, err
	}

	if p.isClient {
		switch transportEvent := message.(type) {
		case transports.AckEvent:
			log.Debugf("[T %s < %s] Received acknowledgement for nonce %x with sequence %d", p.conn.LocalAddr().String(), p.conn.RemoteAddr().String(), transportEvent.Nonce(), transportEvent.Sequence())
			return transportEvent, nil
		case *protocolPONG:
			log.Debugf("[T %s < %s] Received pong", p.conn.LocalAddr().String(), p.conn.RemoteAddr().String())
			return transportEvent, nil
		}
	} else {
		switch transportEvent := message.(type) {
		case *protocolPING:
			log.Debugf("[R %s < %s] Received ping", p.conn.LocalAddr().String(), p.conn.RemoteAddr().String())
			return transportEvent, nil
		case transports.EventsEvent:
			log.Debugf("[R %s < %s] Received payload with nonce %x and %d events", p.conn.LocalAddr().String(), p.conn.RemoteAddr().String(), transportEvent.Nonce(), transportEvent.Count())
			return transportEvent, nil
		}
	}

	return nil, fmt.Errorf("unknown protocol message %T", message)
}

// NonBlocking returns false because courier protocol blocks until full messages are received
func (p *protocol) NonBlocking() bool {
	return false
}
