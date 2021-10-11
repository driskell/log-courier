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
	"encoding/binary"
	"fmt"

	"github.com/driskell/log-courier/lc-lib/core"
)

type protocolHELO struct {
	protocolFlags []byte
	majorVersion  uint32
	minorVersion  uint32
	patchVersion  uint32
	client        string
	reserved      []byte
}

// createProtocolHELO makes a new sendable version
func createProtocolHELO() protocolMessage {
	protocolFlags := make([]byte, 4)
	return &protocolHELO{
		protocolFlags: protocolFlags,
		majorVersion:  core.LogCourierMajorVersion,
		minorVersion:  core.LogCourierMinorVersion,
		patchVersion:  core.LogCourierPatchVersion,
		client:        clientName,
		reserved:      make([]byte, 12),
	}
}

// newProtocolHELO reads a new protocolHELO
func newProtocolHELO(t *connection, bodyLength uint32) (protocolMessage, error) {
	if bodyLength > 32 {
		return nil, fmt.Errorf("protocol error: Corrupt message (HELO size %d > 32)", bodyLength)
	}

	data := make([]byte, 32)
	if bodyLength != 0 {
		if _, err := t.Read(data[:bodyLength]); err != nil {
			return nil, err
		}
	}

	return &protocolHELO{
		protocolFlags: data[:4],
		majorVersion:  binary.BigEndian.Uint32(data[4:8]),
		minorVersion:  binary.BigEndian.Uint32(data[8:12]),
		patchVersion:  binary.BigEndian.Uint32(data[12:16]),
		client:        string(data[16:20]),
		reserved:      data[20:],
	}, nil
}

// Type returns a human-readable name for the message type
func (p *protocolHELO) Type() string {
	return "HELO"
}

// Write writes a payload to the socket
func (p *protocolHELO) Write(conn *connection) error {
	// Encapsulate the message
	// 4-byte message header (HELO)
	// 4-byte uint32 data length
	if _, err := conn.Write([]byte{'H', 'E', 'L', 'O'}); err != nil {
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

	_, err := conn.Write(data)
	return err
}

func (p *protocolHELO) Client() string {
	if p.client[0] == '\x00' {
		if p.majorVersion == 0 && p.minorVersion == 0 && p.patchVersion == 0 {
			return "Unknown"
		}
		return fmt.Sprintf("Unknown %d.%d.%d", p.majorVersion, p.minorVersion, p.patchVersion)
	}
	client := p.client
	if name, has := clientNameMapping[client]; has {
		client = name
	}
	return fmt.Sprintf("%s %d.%d.%d", client, p.majorVersion, p.minorVersion, p.patchVersion)
}
