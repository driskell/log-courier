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

import "fmt"

type protocolUNKN struct {
}

// newProtocolUNKN reads a new protocolUNKN
func newProtocolUNKN(conn *connection, bodyLength uint32) (protocolMessage, error) {
	if bodyLength != 0 {
		return nil, fmt.Errorf("Protocol error: Corrupt message UNKN size %d != 0", bodyLength)
	}

	return &protocolUNKN{}, nil
}

// Type returns a human-readable name for the message type
func (p *protocolUNKN) Type() string {
	return "UNKN"
}

// Write writes a payload to the socket
func (p *protocolUNKN) Write(conn *connection) error {
	// Encapsulate the message
	// 4-byte message header (UNKN)
	// 4-byte uint32 data length (0 length for UNKN)
	_, err := conn.Write([]byte{'?', '?', '?', '?', 0, 0, 0, 0})
	return err
}
