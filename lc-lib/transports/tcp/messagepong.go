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

type protocolPONG struct {
}

// newProtocolPONG reads a new protocolPONG
func newProtocolPONG(conn *connection, bodyLength uint32) (protocolMessage, error) {
	if bodyLength != 0 {
		return nil, fmt.Errorf("Protocol error: Corrupt message PONG size %d != 0", bodyLength)
	}

	return &protocolPONG{}, nil
}

// Type returns a human-readable name for the message type
func (p *protocolPONG) Type() string {
	return "PONG"
}

// Write writes a payload to the socket
func (p *protocolPONG) Write(conn *connection) error {
	// Encapsulate the ping into a message
	// 4-byte message header (PONG)
	// 4-byte uint32 data length (0 length for PONG)
	_, err := conn.Write([]byte{'P', 'O', 'N', 'G', 0, 0, 0, 0})
	return err
}
