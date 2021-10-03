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
)

type protocolACKN struct {
	nonce    *string
	sequence uint32
}

// newProtocolACKN reads a new protocolACKN
func newProtocolACKN(conn *connection, bodyLength uint32) (protocolMessage, error) {
	if bodyLength != 20 {
		return nil, fmt.Errorf("protocol error: Corrupt message (ACKN size %d != 20)", bodyLength)
	}

	message := make([]byte, 20)
	if _, err := conn.Read(message); err != nil {
		return nil, err
	}

	nonce := string(message[:16])
	sequence := binary.BigEndian.Uint32(message[16:])
	return &protocolACKN{nonce: &nonce, sequence: sequence}, nil
}

// Type returns a human-readable name for the message type
func (p *protocolACKN) Type() string {
	return "ACKN"
}

// Write writes a payload to the connection
func (p *protocolACKN) Write(conn *connection) error {
	// Encapsulate the ack into a message
	// 4-byte message header (ACKN)
	// 4-byte message length
	// 16-byte nonce
	// 4-byte uint32 sequence
	if _, err := conn.Write([]byte{'A', 'C', 'K', 'N', 0, 0, 0, 20}); err != nil {
		return err
	}

	if _, err := conn.Write([]byte(*p.nonce)); err != nil {
		return err
	}

	var sequence [4]byte
	binary.BigEndian.PutUint32(sequence[:], uint32(p.sequence))
	_, err := conn.Write(sequence[:])
	return err
}
