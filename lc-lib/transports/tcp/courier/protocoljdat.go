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
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/driskell/log-courier/lc-lib/transports"
	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

type protocolJDAT struct {
	ctx    context.Context
	nonce  *string
	events [][]byte
}

var _ transports.EventsEvent = (*protocolJDAT)(nil)

// newProtocolJDAT creates a new structure from wire-bytes
func newProtocolJDAT(conn tcp.Connection, bodyLength uint32) (tcp.ProtocolMessage, error) {
	if bodyLength < 17 {
		return nil, fmt.Errorf("protocol error: Corrupt message (JDAT size %d < 17)", bodyLength)
	}

	if bodyLength > 10485760 {
		return nil, fmt.Errorf("protocol error: Message body too large (%d > 10485760)", bodyLength)
	}

	data := make([]byte, bodyLength)
	if _, err := conn.Read(data); err != nil {
		return nil, err
	}

	nonce := string(data[:16])

	decompressor, err := zlib.NewReader(bytes.NewReader(data[16:]))
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

	return &protocolJDAT{ctx: conn.Context(), nonce: &nonce, events: events}, nil
}

// Type returns a human-readable name for the message type
func (p *protocolJDAT) Type() string {
	return "JDAT"
}

// Write writes a payload to the socket
func (p *protocolJDAT) Write(conn tcp.Connection) error {
	var eventBuffer bytes.Buffer

	// Create the compressed data payload
	// The event data is each event, prefixed with a 4-byte uint32 length, one
	// after the other
	compressor, err := zlib.NewWriterLevel(&eventBuffer, 3)
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

	// Encapsulate the data into the message
	// 4-byte message header (JDAT = JSON Data, Compressed)
	// 4-byte uint32 data length
	// 16-byte nonce
	// Compressed data
	if _, err := conn.Write([]byte{'J', 'D', 'A', 'T'}); err != nil {
		return err
	}

	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(*p.nonce)+eventBuffer.Len()))
	if _, err := conn.Write(length[:]); err != nil {
		return err
	}

	if _, err := conn.Write([]byte(*p.nonce)); err != nil {
		return err
	}

	if _, err = conn.Write(eventBuffer.Bytes()); err != nil {
		return err
	}
	return conn.Flush()
}

// Nonce returns the nonce - this implements eventsMessage
func (p *protocolJDAT) Nonce() *string {
	return p.nonce
}

// Context returns the connection context
func (p *protocolJDAT) Context() context.Context {
	return p.ctx
}

// Events returns the events - this implements eventsMessage
func (p *protocolJDAT) Events() []map[string]interface{} {
	events := make([]map[string]interface{}, 0, len(p.events))
	for _, data := range p.events {
		var decoded map[string]interface{}
		err := json.Unmarshal(data, &decoded)
		if err != nil {
			decoded = make(map[string]interface{})
			decoded["message"] = err.Error()
			decoded["tags"] = "_unmarshal_failure"
		}
		events = append(events, decoded)
	}

	return events
}

// Count returns the number of events
func (p *protocolJDAT) Count() uint32 {
	return uint32(len(p.events))
}
