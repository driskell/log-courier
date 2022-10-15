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

package courier

import (
	"encoding/binary"
	"fmt"

	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

type protocolVERS struct {
	protocolFlags []byte
	majorVersion  uint32
	minorVersion  uint32
	patchVersion  uint32
	client        string
	reserved      []byte
}

// createProtocolVERS makes a new sendable version
func createProtocolVERS() tcp.ProtocolMessage {
	protocolFlags := make([]byte, 4)
	// SupportsEVNT flag
	protocolFlags[0] = protocolFlags[0] | 0x01
	return &protocolVERS{
		protocolFlags: protocolFlags,
		majorVersion:  core.LogCourierMajorVersion,
		minorVersion:  core.LogCourierMinorVersion,
		patchVersion:  core.LogCourierPatchVersion,
		client:        clientName,
		reserved:      make([]byte, 12),
	}
}

// newProtocolVERS reads a new protocolVERS
func newProtocolVERS(t tcp.Connection, bodyLength uint32) (tcp.ProtocolMessage, error) {
	if bodyLength > 32 {
		return nil, fmt.Errorf("protocol error: Corrupt message (VERS size %d > 32)", bodyLength)
	}

	data := make([]byte, 32)
	if bodyLength != 0 {
		if _, err := t.Read(data[:bodyLength]); err != nil {
			return nil, err
		}
	}

	return &protocolVERS{
		protocolFlags: data[:4],
		majorVersion:  binary.BigEndian.Uint32(data[4:8]),
		minorVersion:  binary.BigEndian.Uint32(data[8:12]),
		patchVersion:  binary.BigEndian.Uint32(data[12:16]),
		client:        string(data[16:20]),
		reserved:      data[20:],
	}, nil
}

// Type returns a human-readable name for the message type
func (p *protocolVERS) Type() string {
	return "VERS"
}

// Write writes a payload to the socket
func (p *protocolVERS) Write(conn tcp.Connection) error {
	// Encapsulate the message
	// 4-byte message header (VERS)
	// 4-byte uint32 data length (1 length for VERS)
	if _, err := conn.Write([]byte{'V', 'E', 'R', 'S'}); err != nil {
		return err
	}

	var length [4]byte
	binary.BigEndian.PutUint32(length[:], 32)
	if _, err := conn.Write(length[:]); err != nil {
		return err
	}

	data := make([]byte, 32)
	copy(data, p.protocolFlags[:4])
	binary.BigEndian.PutUint32(data[4:8], p.majorVersion)
	binary.BigEndian.PutUint32(data[8:12], p.minorVersion)
	binary.BigEndian.PutUint32(data[12:16], p.patchVersion)
	copy(data[16:20], []byte(p.client[:4]))
	copy(data[20:], p.reserved)

	if _, err := conn.Write(data); err != nil {
		return err
	}
	return conn.Flush()
}

// SupportsEVNT returns true if the remote side supports the enhanced message
func (p *protocolVERS) SupportsEVNT() bool {
	return len(p.protocolFlags) > 0 && p.protocolFlags[0]&0x01 == 0x01
}

func (p *protocolVERS) Client() string {
	if p.client[0] == '\x00' {
		if p.majorVersion == 0 && p.minorVersion == 0 && p.patchVersion == 0 {
			return "Unknown"
		}
		return fmt.Sprintf("Unknown v%d.%d.%d", p.majorVersion, p.minorVersion, p.patchVersion)
	}
	client := p.client
	if name, has := clientNameMapping[client]; has {
		client = name
	}
	return fmt.Sprintf("%s v%d.%d.%d", client, p.majorVersion, p.minorVersion, p.patchVersion)
}
