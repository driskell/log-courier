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

type protocolACKN struct {
	nonce    string
	sequence uint32
}

// newProtocolACKN reads a new protocolACKN
func newProtocolACKN(t connection, bodyLength uint32) (*protocolACKN, error) {
	if bodyLength != 20 {
		return nil, fmt.Errorf("Protocol error: Corrupt message (ACKN size %d != 20)", bodyLength)
	}

	message, err := t.Read(20)
	if message == nil {
		return nil, err
	}

	nonce := string(message[:16])
	sequence := binary.BigEndian.Uint32(message[16:])
	return &protocolACKN{nonce: nonce, sequence: sequence}, nil
}

// Write writes a payload to the socket
func (p *protocolACKN) Write(t connection) error {
	// Encapsulate the ack into a message
	// 4-byte message header (ACKN)
	// 16-byte nonce
	// 4-byte uint32 sequence
	if _, err := t.Write([]byte{'A', 'C', 'K', 'N'}); err != nil {
		return err
	}

	if _, err := t.Write([]byte(p.nonce)); err != nil {
		return err
	}

	var sequence [4]byte
	binary.BigEndian.PutUint32(sequence[:], uint32(p.sequence))
	_, err := t.Write(sequence[:])
	return err
}
