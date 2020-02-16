/*
 * Copyright 2014-2015 Jason Woods.
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
	_ "crypto/sha256" // Support for newer SSL signature algorithms
	_ "crypto/sha512" // Support for newer SSL signature algorithms
	"errors"
	"net"

	"github.com/driskell/log-courier/lc-lib/event"
)

const (
	// Essentially, this is how often we should check for disconnect/shutdown during socket reads
	socketIntervalSeconds = 1
)

var (
	// ErrEventTooLarge occurs when a message breaches the size limit configured
	ErrEventTooLarge = errors.New("JDAT compressed entry message too large to decode")

	// ErrUnexpectedEnd occurs when a message ends unexpectedly
	ErrUnexpectedEnd = errors.New("Unexpected end of JDAT compressed entry")

	// ErrUnexpectedBytes occurs when a message has extraneous bytes beyond its compression stream
	ErrUnexpectedBytes = errors.New("Unexpected bytes after JDAT compressed entry end")

	// TransportTCPTCP is the transport name for plain TCP
	TransportTCPTCP = "tcp"
	// TransportTCPTLS is the transport name for encrypted TLS
	TransportTCPTLS = "tls"
)

type connection interface {
	Run() error
	Teardown()
	Server() bool
	Write([]byte) (int, error)
	Read(uint32) ([]byte, error)
	Acknowledge(events []*event.Event)
	SendChan() chan protocolMessage
	SupportsEVNT() bool
}

type listener interface {
	Start(string, *net.TCPAddr) (bool, error)
	Stop()
}

type protocolMessage interface {
	Write(connection) error
}

type eventsMessage interface {
	Nonce() string
	Events() []*event.Event
}

type socketMessage struct {
	conn connection
	err  error
}
