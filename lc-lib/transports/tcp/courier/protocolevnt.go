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
	"compress/zlib"
	"encoding/binary"

	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

// protocolEVNT is an EVNT message to send
type protocolEVNT struct {
	nonce  *string
	events [][]byte
}

// Type returns a human-readable name for the message type
func (p *protocolEVNT) Type() string {
	return "EVNT"
}

// Write writes a payload to the socket
func (p *protocolEVNT) Write(conn tcp.Connection) error {
	// Encapsulate the data into the message
	// 4-byte message header (EVNT = EVNT Data, Compressed, Enhanced over JDAT in that it streams and has no size prefix)
	// 4-byte uint32 data length of 0xFFFF (stream)
	// 16-byte nonce
	// compressed stream
	if _, err := conn.Write([]byte{'E', 'V', 'N', 'T', 255, 255, 255, 255}); err != nil {
		return err
	}

	if _, err := conn.Write([]byte(*p.nonce)); err != nil {
		return err
	}

	compressor, err := zlib.NewWriterLevel(conn, 3)
	if err != nil {
		return err
	}

	for _, singleEvent := range p.events {
		if err := binary.Write(compressor, binary.BigEndian, uint32(len(singleEvent))); err != nil {
			return err
		}

		if _, err := compressor.Write(singleEvent); err != nil {
			return err
		}
	}

	if err := compressor.Close(); err != nil {
		return err
	}
	return conn.Flush()
}

// Nonce returns the nonce
func (p *protocolEVNT) Nonce() *string {
	return p.nonce
}
