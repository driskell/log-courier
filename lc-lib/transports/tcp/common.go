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
	"crypto/tls"
	"errors"
	"fmt"
	"net"
)

const (
	// This is how often we should check for disconnect/shutdown during socket reads
	socketIntervalSeconds = 1

	// Default to TLS 1.2 minimum, supported since Go 1.2
	defaultMinTLSVersion = tls.VersionTLS12
	defaultMaxTLSVersion = 0

	// TransportTCPTCP is the transport name for plain TCP
	TransportTCPTCP = "tcp"
	// TransportTCPTLS is the transport name for encrypted TLS
	TransportTCPTLS = "tls"
)

var (
	// ErrEventTooLarge occurs when a message breaches the size limit configured
	ErrEventTooLarge = errors.New("JDAT compressed entry message too large to decode")

	// ErrUnexpectedEnd occurs when a message ends unexpectedly
	ErrUnexpectedEnd = errors.New("unexpected end of JDAT compressed entry")

	// clientName holds the client identifier to send in VERS and HELO
	clientName string = "\x00\x00\x00\x00"

	// clientNameMapping holds mapping from short name to full name for HELO and VERS
	clientNameMapping map[string]string = map[string]string{
		"LCOR": "Log Courier",
		"LCVR": "Log Carver",
		"RYLC": "Ruby Log Courier",
	}
)

type connectionSocket interface {
	net.Conn
	Setup(context.Context) error
	Desc() string
	CloseWrite() error
}

type protocolMessage interface {
	Type() string
	Write(*connection) error
}

type eventsMessage interface {
	protocolMessage
	Nonce() *string
	Events() [][]byte
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
	return fallback, fmt.Errorf("invalid or unknown TLS version: '%s'", version)
}

// getTlsVersionAsString returns a string representation of the TLS version
func getTlsVersionAsString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLSv1"
	case tls.VersionTLS11:
		return "TLSv1.1"
	case tls.VersionTLS12:
		return "TLSv1.2"
	case tls.VersionTLS13:
		return "TLSv1.3"
	}
	return fmt.Sprintf("Unknown (%d)", version)
}

func SetClientName(client string) {
	clientName = client
}
