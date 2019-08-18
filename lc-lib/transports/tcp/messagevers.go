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

import "fmt"

type protocolVERS struct {
	protocolFlags []byte
}

// newProtocolVERS reads a new protocolVERS
func newProtocolVERS(t connection, bodyLength uint32) (*protocolVERS, error) {
	if bodyLength > 32 {
		return nil, fmt.Errorf("Protocol error: Corrupt message (VERS size %d > 32)", bodyLength)
	}

	protocolFlags, err := t.Read(bodyLength)
	if protocolFlags == nil {
		return nil, err
	}

	return &protocolVERS{protocolFlags: protocolFlags}, nil
}

// Write writes a payload to the socket
func (p *protocolVERS) Write(t connection) error {
	// Encapsulate the message
	// 4-byte message header (UNKN)
	// 4-byte uint32 data length (0 length for UNKN)
	_, err := t.Write([]byte{'?', '?', '?', '?', 0, 0, 0, 0})
	return err
}

// SupportsEVNT returns true if the remote side supports the enhanced message
func (p *protocolVERS) SupportsEVNT() bool {
	return len(p.protocolFlags) > 0 && p.protocolFlags[0]&0x01 == 0x01
}
