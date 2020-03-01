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
	"encoding/binary"
	"fmt"
)

type protocolVERS struct {
	protocolFlags []byte
}

// createProtocolVERS makes a new sendable version
func createProtocolVERS() protocolMessage {
	protocolFlags := make([]byte, 1)
	// SupportsEVNT flag
	protocolFlags[0] = protocolFlags[0] & 0x01
	return &protocolVERS{
		protocolFlags: protocolFlags,
	}
}

// newProtocolVERS reads a new protocolVERS
func newProtocolVERS(t *connection, bodyLength uint32) (protocolMessage, error) {
	if bodyLength > 32 {
		return nil, fmt.Errorf("Protocol error: Corrupt message (VERS size %d > 32)", bodyLength)
	}

	protocolFlags := make([]byte, bodyLength)
	if _, err := t.Read(protocolFlags); err != nil {
		return nil, err
	}

	return &protocolVERS{protocolFlags: protocolFlags}, nil
}

// Write writes a payload to the socket
func (p *protocolVERS) Write(conn *connection) error {
	// Encapsulate the message
	// 4-byte message header (VERS)
	// 4-byte uint32 data length (1 length for VERS)
	if _, err := conn.Write([]byte{'V', 'E', 'R', 'S'}); err != nil {
		return err
	}

	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(p.protocolFlags)))
	if _, err := conn.Write(length[:]); err != nil {
		return err
	}

	_, err := conn.Write(p.protocolFlags)
	return err
}

// SupportsEVNT returns true if the remote side supports the enhanced message
func (p *protocolVERS) SupportsEVNT() bool {
	return len(p.protocolFlags) > 0 && p.protocolFlags[0]&0x01 == 0x01
}
