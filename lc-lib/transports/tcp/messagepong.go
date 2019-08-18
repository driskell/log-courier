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

type protocolPONG struct {
}

// newProtocolPONG reads a new protocolPONG
func newProtocolPONG(t connection, bodyLength uint32) (*protocolPONG, error) {
	if bodyLength != 0 {
		return nil, fmt.Errorf("Protocol error: Corrupt message PONG size %d != 0", bodyLength)
	}

	return &protocolPONG{}, nil
}

// Write writes a payload to the socket
func (p *protocolPONG) Write(t connection) error {
	// Encapsulate the ping into a message
	// 4-byte message header (PONG)
	// 4-byte uint32 data length (0 length for PONG)
	_, err := t.Write([]byte{'P', 'O', 'N', 'G', 0, 0, 0, 0})
	return err
}
