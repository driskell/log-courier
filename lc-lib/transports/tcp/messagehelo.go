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

type protocolHELO struct {
}

// newProtocolHELO reads a new protocolHELO
func newProtocolHELO(conn *connection, bodyLength uint32) (protocolMessage, error) {
	if bodyLength != 0 {
		return nil, fmt.Errorf("Protocol error: Corrupt message HELO size %d != 0", bodyLength)
	}

	return &protocolHELO{}, nil
}

// Write writes a payload to the socket
func (p *protocolHELO) Write(conn *connection) error {
	// Encapsulate the HELO into a message
	// 4-byte message header (HELO)
	// 4-byte uint32 data length (0 length for HELO)
	_, err := conn.Write([]byte{'H', 'E', 'L', 'O', 0, 0, 0, 0})
	return err
}
