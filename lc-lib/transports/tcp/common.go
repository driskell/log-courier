/*
 * Copyright 2012-2020 Jason Woods and contributors
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
	"context"
	_ "crypto/sha256" // Support for newer SSL signature algorithms
	_ "crypto/sha512" // Support for newer SSL signature algorithms
	"errors"
	"net"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/transports"
)

const (
	// This is how often we should check for disconnect/shutdown during socket reads
	socketIntervalSeconds = 1
)

var (
	// ErrEventTooLarge occurs when a message breaches the size limit configured
	ErrEventTooLarge = errors.New("JDAT compressed entry message too large to decode")

	// ErrUnexpectedEnd occurs when a message ends unexpectedly
	ErrUnexpectedEnd = errors.New("unexpected end of JDAT compressed entry")

	// ErrIOWouldBlock is returned when a protocol is non-blocking and a read would block longer than the socket interval
	ErrIOWouldBlock = errors.New("IO would block")
)

type connectionSocket interface {
	net.Conn
	Setup(context.Context) error
	Desc() string
	CloseWrite() error
}

type Protocol interface {
	Negotiation() (transports.Event, error)
	SendEvents(string, []*event.Event) error
	Ping() error
	Pong() error
	Acknowledge(*string, uint32) error
	Read() (transports.Event, error)
	NonBlocking() bool
}

type ProtocolFactory interface {
	NewProtocol(Connection) Protocol
	SupportsAck() bool
}

type ProtocolMessage interface {
	Write(Connection) error
}

type Connection interface {
	Context() context.Context
	Write(data []byte) (int, error)
	Flush() error
	Read(data []byte) (int, error)
	SendMessage(message ProtocolMessage) error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}
