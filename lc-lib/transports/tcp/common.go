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
	_ "crypto/sha256" // Support for newer SSL signature algorithms
	_ "crypto/sha512" // Support for newer SSL signature algorithms
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	"github.com/driskell/log-courier/lc-lib/event"
)

const (
	// This is how often we should check for disconnect/shutdown during socket reads
	socketIntervalSeconds = 1

	// Default to TLS 1.2 minimum, supported since Go 1.2
	defaultMinTLSVersion = tls.VersionTLS12
	defaultMaxTLSVersion = 0
)

var (
	// ErrEventTooLarge occurs when a message breaches the size limit configured
	ErrEventTooLarge = errors.New("JDAT compressed entry message too large to decode")

	// ErrUnexpectedEnd occurs when a message ends unexpectedly
	ErrUnexpectedEnd = errors.New("Unexpected end of JDAT compressed entry")

	// TransportTCPTCP is the transport name for plain TCP
	TransportTCPTCP = "tcp"
	// TransportTCPTLS is the transport name for encrypted TLS
	TransportTCPTLS = "tls"
)

type connectionSocket interface {
	net.Conn
	Setup() error
	CloseWrite() error
}

type listener interface {
	Start(string, *net.TCPAddr) (bool, error)
	Stop()
}

type protocolMessage interface {
	Type() string
	Write(*connection) error
}

type eventsMessage interface {
	protocolMessage
	Nonce() string
	Events() []*event.Event
}

type eventPosition struct {
	nonce    string
	sequence uint32
}

type socketMessage struct {
	conn *connection
	err  error
}

// parseTLSVersion parses a TLS version string into the tls library value for min/max config
// We explicitly refuse SSLv3 to mitigate POODLE vulnerability
func parseTLSVersion(version string, fallback uint16) (uint16, error) {
	switch version {
	case "":
		return fallback, nil
	case "1.0":
		return tls.VersionTLS10, nil
	case "1.1":
		return tls.VersionTLS11, nil
	case "1.2":
		return tls.VersionTLS12, nil
	case "1.3":
		return tls.VersionTLS13, nil
	}
	return tls.VersionTLS10, fmt.Errorf("Invalid or unknown TLS version: '%s'", version)
}
