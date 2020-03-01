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
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/driskell/log-courier/lc-lib/event"
)

type protocolJDAT struct {
	nonce  string
	events []*event.Event
}

// newProtocolJDAT creates a new structure from wire-bytes
func newProtocolJDAT(conn *connection, bodyLength uint32) (protocolMessage, error) {
	if !conn.Server() {
		return nil, errors.New("Protocol error: Unexpected JDAT message received on non-server connection")
	}

	if bodyLength < 17 {
		return nil, fmt.Errorf("Protocol error: Corrupt message (JDAT size %d < 17)", bodyLength)
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

	events := make([]*event.Event, 0, 100)

	sequence := uint32(0)
	for {
		sequence++

		var size uint32
		if err := binary.Read(decompressor, binary.BigEndian, &size); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if size > 10485760 {
			return nil, ErrEventTooLarge
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
					return nil, ErrUnexpectedEnd
				}
				return nil, err
			}
		}

		events = append(events, event.NewEventFromBytes(conn, data, &evntPosition{nonce: nonce, sequence: sequence}))
	}

	return &protocolJDAT{nonce: nonce, events: events}, nil
}

// Write writes a payload to the socket
func (p *protocolJDAT) Write(conn *connection) error {
	var eventBuffer bytes.Buffer

	// Create the compressed data payload
	// The event data is each event, prefixed with a 4-byte uint32 length, one
	// after the other
	compressor, err := zlib.NewWriterLevel(&eventBuffer, 3)
	if err != nil {
		return err
	}

	for _, singleEvent := range p.events {
		if err := binary.Write(compressor, binary.BigEndian, uint32(len(singleEvent.Bytes()))); err != nil {
			return err
		}

		if _, err := compressor.Write(singleEvent.Bytes()); err != nil {
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
	binary.BigEndian.PutUint32(length[:], uint32(len(p.nonce)+eventBuffer.Len()))
	if _, err := conn.Write(length[:]); err != nil {
		return err
	}

	if _, err := conn.Write([]byte(p.nonce)); err != nil {
		return err
	}

	_, err = conn.Write(eventBuffer.Bytes())
	return err
}

// Nonce returns the nonce - this implements eventsMessage
func (p *protocolJDAT) Nonce() string {
	return p.nonce
}

// Events returns the events - this implements eventsMessage
func (p *protocolJDAT) Events() []*event.Event {
	return p.events
}
