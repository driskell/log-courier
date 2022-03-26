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
	"fmt"
	"io"
	"math"

	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

type protocolEVNT struct {
	nonce  *string
	events [][]byte
}

// Reads the events from existing data
func newProtocolEVNT(conn tcp.Connection, bodyLength uint32) (tcp.ProtocolMessage, error) {
	if bodyLength != math.MaxUint32 {
		return nil, fmt.Errorf("protocol error: Corrupt message (EVNT size %d != %d)", bodyLength, uint32(math.MaxUint32))
	}

	data := make([]byte, 16)
	if _, err := conn.Read(data); err != nil {
		return nil, err
	}

	nonce := string(data)

	decompressor, err := zlib.NewReader(conn)
	if err != nil {
		return nil, err
	}

	events := make([][]byte, 0, 100)

	for {
		var size uint32
		if err := binary.Read(decompressor, binary.BigEndian, &size); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if size > 10485760 {
			return nil, tcp.ErrEventTooLarge
		}

		data := make([]byte, size)
		read := 0
		for {
			n, err := decompressor.Read(data[read:])
			read += n
			if read >= int(size) {
				break
			}
			if err != nil {
				if err == io.EOF {
					return nil, tcp.ErrUnexpectedEnd
				}
				return nil, err
			}
		}

		events = append(events, data)
	}

	return &protocolEVNT{nonce: &nonce, events: events}, nil
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

// Nonce returns the nonce - this implements eventsMessage
func (p *protocolEVNT) Nonce() *string {
	return p.nonce
}

// Events returns the events - this implements eventsMessage
func (p *protocolEVNT) Events() [][]byte {
	return p.events
}
